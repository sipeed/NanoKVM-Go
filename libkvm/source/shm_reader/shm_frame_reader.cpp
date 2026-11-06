#include "shm_frame_reader.hpp"

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstdio>
#include <ctime>
#include <stddef.h>
#include <thread>

#ifndef SHM_READER_TEST_LOG
#define SHM_READER_TEST_LOG 0
#endif

#if SHM_READER_TEST_LOG
#define SR_LOG(fmt, ...) std::fprintf(stderr, "[ShmFrameReader] " fmt "\n", ##__VA_ARGS__)
#else
#define SR_LOG(fmt, ...)
#endif

static inline uint64_t monotonic_us()
{
    struct timespec ts{};
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return static_cast<uint64_t>(ts.tv_sec) * 1000000ULL +
           static_cast<uint64_t>(ts.tv_nsec) / 1000ULL;
}

void ShmFrameReader::close_mapping()
{
    shm_frame_close(handle_);

    handle_ = nullptr;
    ring_ = nullptr;
    map_size_ = 0;
    last_frame_id_ = 0;
    map_ino_ = 0;
}

void ShmFrameReader::start_idr_worker()
{
    {
        std::lock_guard<std::mutex> lk(idr_mutex_);
        idr_worker_stop_ = false;
        idr_pending_ = false;
    }
    idr_worker_ = std::thread(&ShmFrameReader::idr_worker_loop, this);
}

void ShmFrameReader::stop_idr_worker()
{
    if (!idr_worker_.joinable()) return;
    {
        std::lock_guard<std::mutex> lk(idr_mutex_);
        idr_worker_stop_ = true;
    }
    idr_cv_.notify_one();
    idr_worker_.join();
}

void ShmFrameReader::idr_worker_loop()
{
    bool have_fired = false;
    std::chrono::steady_clock::time_point last_fired{};

    for (;;) {
        std::unique_lock<std::mutex> lk(idr_mutex_);
        idr_cv_.wait(lk, [this] {
            return idr_worker_stop_ || idr_pending_;
        });
        if (idr_worker_stop_) return;

        /* consume the pending flag; bursts collapse into one request */
        idr_pending_ = false;
        lk.unlock();

        /* rate-limit: skip if we fired too recently.
           Use a flag for the first request to avoid subtracting a sentinel
           time_point (overflows and would suppress everything). */
        const auto now = std::chrono::steady_clock::now();
        if (have_fired && (now - last_fired) < idr_min_interval_) {
            SR_LOG("idr_worker: trigger suppressed by rate-limit");
            continue;
        }
        have_fired = true;
        last_fired = now;

        /* blocking socket round-trip happens here, off the read path */
        if (idr_request_callback_) {
            SR_LOG("idr_worker: sending IDR request");
            idr_request_callback_();
            SR_LOG("idr_worker: IDR request returned");
        } else {
            SR_LOG("idr_worker: no callback set, skip");
        }
    }
}

static bool shm_inode_changed(const char* name, ino_t cached_ino, bool& open_failed)
{
    open_failed = false;
    int fd = shm_open(name, O_RDONLY, 0);
    if (fd < 0) {
        open_failed = true;
        return false;
    }
    struct stat st{};
    int rc = fstat(fd, &st);
    close(fd);
    if (rc != 0) {
        open_failed = true;
        return false;
    }
    return st.st_ino != cached_ino;
}

bool ShmFrameReader::ensure_open()
{
    if (ring_) return true;

    shm_frame_handle_t* handle = shm_frame_open(shm_name_);
    if (!handle) {
        SR_LOG("ensure_open: shm_open failed, name=%s", shm_name_);
        return false;
    }

    handle_ = handle;
    ring_ = handle_->ring;
    map_size_ = handle_->size;
    map_ino_ = handle_->ino;
    SR_LOG("ensure_open: success, name=%s max_frame=%u map_size=%zu",
           shm_name_, ring_->max_frame_size, map_size_);
    return true;
}

static bool sleep_brief_until_deadline(const std::chrono::steady_clock::time_point& deadline)
{
    const auto now = std::chrono::steady_clock::now();
    if (now >= deadline) return false;

    const auto remain = deadline - now;
    const auto nap = (remain > std::chrono::microseconds(1000))
        ? std::chrono::microseconds(1000)
        : std::chrono::duration_cast<std::chrono::microseconds>(remain);
    std::this_thread::sleep_for(nap);
    return true;
}

static void sleep_brief_no_deadline()
{
    std::this_thread::sleep_for(std::chrono::microseconds(1000));
}

int ShmFrameReader::read_frame(FrameRef& out, bool force_key, int timeout_ms)
{
    std::lock_guard<std::mutex> lock(mutex_);

    const bool wait_forever = (timeout_ms < 0);

    if (!ensure_open()) {
        SR_LOG("read_frame: ensure_open failed, timeout_ms=%d", timeout_ms);
        std::this_thread::sleep_for(std::chrono::milliseconds(200));
        return -1;
    }

    __atomic_store_n(&ring_->reader_heartbeat_us, monotonic_us(),
                     __ATOMIC_RELEASE);

    const auto deadline = wait_forever
        ? std::chrono::steady_clock::time_point::max()
        : (std::chrono::steady_clock::now() + std::chrono::milliseconds(timeout_ms));
    bool need_key = force_key;

    constexpr auto kStaleCheckInterval = std::chrono::milliseconds(500);
    auto next_stale_check = std::chrono::steady_clock::now() + kStaleCheckInterval;
    auto next_heartbeat = std::chrono::steady_clock::now() + std::chrono::milliseconds(200);
    uint64_t stuck_frame_id = last_frame_id_;

    while (wait_forever || std::chrono::steady_clock::now() < deadline) {
        if (!ring_) {
            if (!ensure_open()) {
                if (wait_forever) {
                    std::this_thread::sleep_for(std::chrono::milliseconds(100));
                    continue;
                }
                if (!sleep_brief_until_deadline(deadline)) break;
                continue;
            }
            need_key = force_key;
            stuck_frame_id = 0;
            next_stale_check = std::chrono::steady_clock::now() + kStaleCheckInterval;
            next_heartbeat = std::chrono::steady_clock::now() + std::chrono::milliseconds(200);
            __atomic_store_n(&ring_->reader_heartbeat_us, monotonic_us(),
                             __ATOMIC_RELEASE);
        }

        const auto now_tp = std::chrono::steady_clock::now();
        if (now_tp >= next_heartbeat) {
            __atomic_store_n(&ring_->reader_heartbeat_us, monotonic_us(),
                             __ATOMIC_RELEASE);
            next_heartbeat = now_tp + std::chrono::milliseconds(200);
        }

        uint64_t frame_id = __atomic_load_n(&ring_->frame.frame_id, __ATOMIC_ACQUIRE);
        if (frame_id != 0 && frame_id < last_frame_id_) {
            SR_LOG("read_frame: writer restart detected, reset last_frame_id from %llu to 0 (frame_id=%llu)",
                   static_cast<unsigned long long>(last_frame_id_),
                   static_cast<unsigned long long>(frame_id));
            last_frame_id_ = 0;
            stuck_frame_id = 0;
            need_key = force_key;
        }

        if (frame_id == stuck_frame_id) {
            const auto now = std::chrono::steady_clock::now();
            if (now >= next_stale_check) {
                next_stale_check = now + kStaleCheckInterval;
                bool open_failed = false;
                if (shm_inode_changed(shm_name_, map_ino_, open_failed)) {
                    SR_LOG("read_frame: writer restarted (inode changed), remap name=%s", shm_name_);
                    close_mapping();
                    if (!ensure_open()) {
                        SR_LOG("read_frame: remap failed, name=%s", shm_name_);
                        if (wait_forever) {
                            std::this_thread::sleep_for(std::chrono::milliseconds(100));
                            continue;
                        }
                        return -1;
                    }
                    need_key = force_key;
                    stuck_frame_id = 0;
                    continue;
                }
                if (open_failed) {
                    SR_LOG("read_frame: writer shm gone, name=%s", shm_name_);
                }
            }
        } else {
            stuck_frame_id = frame_id;
            next_stale_check = std::chrono::steady_clock::now() + kStaleCheckInterval;
        }

        if (frame_id == 0 || frame_id == last_frame_id_) {
            if (wait_forever) {
                sleep_brief_no_deadline();
            } else if (!sleep_brief_until_deadline(deadline)) {
                break;
            }
            continue;
        }

        if (flock(handle_->fd, LOCK_SH | LOCK_NB) < 0) {
            if (errno == EWOULDBLOCK || errno == EAGAIN) {
                if (wait_forever) {
                    sleep_brief_no_deadline();
                } else if (!sleep_brief_until_deadline(deadline)) {
                    break;
                }
                continue;
            }
            SR_LOG("read_frame: shared lock failed, errno=%d", errno);
            close_mapping();
            continue;
        }

        frame_id = __atomic_load_n(&ring_->frame.frame_id, __ATOMIC_ACQUIRE);
        if (frame_id == 0 || frame_id == last_frame_id_) {
            (void)flock(handle_->fd, LOCK_UN);
            if (wait_forever) {
                sleep_brief_no_deadline();
            } else if (!sleep_brief_until_deadline(deadline)) {
                break;
            }
            continue;
        }

        const uint32_t len = ring_->frame.data_len;
        if (len == 0 || len > ring_->max_frame_size) {
            (void)flock(handle_->fd, LOCK_UN);
            if (wait_forever) {
                sleep_brief_no_deadline();
            } else if (!sleep_brief_until_deadline(deadline)) {
                break;
            }
            continue;
        }

        const uint32_t w   = ring_->frame.width;
        const uint32_t h   = ring_->frame.height;
        const uint32_t f   = ring_->frame.format;
        const uint8_t  k   = ring_->frame.key_frame;
        const uint64_t pts = ring_->frame.pts_us;

        frame_buffer_.resize(len);
        memcpy(frame_buffer_.data(), ring_->data, len);

        if (flock(handle_->fd, LOCK_UN) < 0) {
            SR_LOG("read_frame: unlock failed, errno=%d", errno);
            close_mapping();
            continue;
        }

        last_frame_id_ = frame_id;

        if (need_key && !k) {
            continue;
        }
        need_key = force_key;

        out.data   = frame_buffer_.data();
        out.size   = len;
        out.width  = w;
        out.height = h;
        out.format = f;
        out.key    = (k != 0);
        out.pts_us = pts;
        SR_LOG("read_frame: ok frame_id=%llu len=%u key=%u pts_us=%llu",
               static_cast<unsigned long long>(frame_id),
               len,
               static_cast<unsigned int>(k),
               static_cast<unsigned long long>(pts));
        return 0;
    }

    const uint64_t cur_frame_id = ring_ ? __atomic_load_n(&ring_->frame.frame_id, __ATOMIC_ACQUIRE) : 0;
    SR_LOG("read_frame: timeout timeout_ms=%d frame_id=%llu last_frame_id=%llu",
           timeout_ms,
           static_cast<unsigned long long>(cur_frame_id),
           static_cast<unsigned long long>(last_frame_id_));

    if (ring_ && cur_frame_id == last_frame_id_) {
        SR_LOG("read_frame: possible stale mapping detected, remap shm name=%s", shm_name_);
        close_mapping();
        if (ensure_open()) {
            SR_LOG("read_frame: remap success, name=%s", shm_name_);
        } else {
            SR_LOG("read_frame: remap failed, name=%s", shm_name_);
        }
    }

    return -1;
}
