#include "video_socket/unix_video_frame_reader.hpp"

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <thread>

#include <poll.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include "../../../kvm_vin/source/include/video_frame_protocol.h"

namespace {

constexpr uint32_t kMaxVideoFrameSize = 2U * 1024U * 1024U;

int remaining_ms(const std::chrono::steady_clock::time_point& deadline)
{
    if (deadline == std::chrono::steady_clock::time_point::max()) {
        return -1;
    }
    const auto now = std::chrono::steady_clock::now();
    if (now >= deadline) {
        return 0;
    }
    return static_cast<int>(
        std::chrono::duration_cast<std::chrono::milliseconds>(deadline - now).count());
}

void sleep_until_deadline(const std::chrono::steady_clock::time_point& deadline)
{
    if (deadline == std::chrono::steady_clock::time_point::max()) {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        return;
    }
    const auto now = std::chrono::steady_clock::now();
    if (now >= deadline) return;
    const auto remain = deadline - now;
    if (remain > std::chrono::milliseconds(20)) {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    } else {
        std::this_thread::sleep_for(remain);
    }
}

} // namespace

UnixVideoFrameReader::UnixVideoFrameReader(const char* socket_path)
    : socket_path_(socket_path)
{
}

UnixVideoFrameReader::~UnixVideoFrameReader()
{
    stop_idr_worker();
    close_fd();
}

void UnixVideoFrameReader::set_idr_request_callback(
    std::function<void()> callback,
    std::chrono::milliseconds min_interval)
{
    stop_idr_worker();
    idr_request_callback_ = std::move(callback);
    idr_min_interval_ = min_interval;
    if (idr_request_callback_) {
        start_idr_worker();
    }
}

void UnixVideoFrameReader::close_stream()
{
    std::lock_guard<std::mutex> lock(mutex_);
    close_fd();
    std::vector<uint8_t>().swap(frame_buffer_);
}

void UnixVideoFrameReader::close_fd()
{
    if (fd_ >= 0) {
        ::close(fd_);
        fd_ = -1;
    }
    waiting_keyframe_ = true;
    idr_requested_for_connection_ = false;
}

bool UnixVideoFrameReader::ensure_connected()
{
    if (fd_ >= 0) return true;

    int fd = ::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
    if (fd < 0) {
        return false;
    }

    sockaddr_un addr{};
    addr.sun_family = AF_UNIX;
    std::snprintf(addr.sun_path, sizeof(addr.sun_path), "%s", socket_path_);

    if (::connect(fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) != 0) {
        ::close(fd);
        return false;
    }

    fd_ = fd;
    waiting_keyframe_ = true;
    idr_requested_for_connection_ = false;
    return true;
}

bool UnixVideoFrameReader::wait_readable(const std::chrono::steady_clock::time_point& deadline)
{
    if (fd_ < 0) return false;

    pollfd pfd{};
    pfd.fd = fd_;
    pfd.events = POLLIN;

    for (;;) {
        const int timeout = remaining_ms(deadline);
        if (timeout == 0 && deadline != std::chrono::steady_clock::time_point::max()) {
            return false;
        }

        int rc = ::poll(&pfd, 1, timeout);
        if (rc > 0) {
            if ((pfd.revents & POLLIN) != 0) return true;
            close_fd();
            return false;
        }
        if (rc == 0) return false;
        if (errno == EINTR) continue;
        close_fd();
        return false;
    }
}

bool UnixVideoFrameReader::read_exact(void* data,
                                      size_t size,
                                      const std::chrono::steady_clock::time_point& deadline)
{
    uint8_t* out = static_cast<uint8_t*>(data);
    size_t read_size = 0;

    while (read_size < size) {
        if (!wait_readable(deadline)) {
            if (read_size > 0) {
                close_fd();
            }
            return false;
        }

        ssize_t n = ::recv(fd_, out + read_size, size - read_size, 0);
        if (n > 0) {
            read_size += static_cast<size_t>(n);
            continue;
        }
        if (n < 0 && errno == EINTR) {
            continue;
        }

        close_fd();
        return false;
    }
    return true;
}

int UnixVideoFrameReader::read_frame(FrameRef& out, bool force_key, int timeout_ms)
{
    std::lock_guard<std::mutex> lock(mutex_);
    const auto deadline = timeout_ms < 0
        ? std::chrono::steady_clock::time_point::max()
        : std::chrono::steady_clock::now() + std::chrono::milliseconds(timeout_ms);

    while (timeout_ms < 0 || std::chrono::steady_clock::now() < deadline) {
        if (!ensure_connected()) {
            sleep_until_deadline(deadline);
            continue;
        }

        if (!idr_requested_for_connection_) {
            idr_requested_for_connection_ = true;
            trigger_idr();
        }

        kvm_video_frame_header_t header{};
        if (!read_exact(&header, sizeof(header), deadline)) {
            continue;
        }

        if (header.magic != KVM_VIDEO_FRAME_MAGIC ||
            header.version != KVM_VIDEO_FRAME_VERSION ||
            header.header_len != KVM_VIDEO_FRAME_HEADER_LEN ||
            header.payload_len == 0 ||
            header.payload_len > kMaxVideoFrameSize) {
            close_fd();
            continue;
        }

        frame_buffer_.resize(header.payload_len);
        if (!read_exact(frame_buffer_.data(), frame_buffer_.size(), deadline)) {
            close_fd();
            continue;
        }

        const bool is_key = (header.flags & KVM_VIDEO_FRAME_FLAG_KEY) != 0;
        if ((waiting_keyframe_ || force_key) && !is_key) {
            trigger_idr();
            continue;
        }
        waiting_keyframe_ = false;

        out.data = frame_buffer_.data();
        out.size = header.payload_len;
        out.width = header.width;
        out.height = header.height;
        out.format = header.format;
        out.key = is_key;
        out.pts_us = header.pts_us;
        return 0;
    }

    return -1;
}

void UnixVideoFrameReader::trigger_idr()
{
    {
        std::lock_guard<std::mutex> lk(idr_mutex_);
        idr_pending_ = true;
    }
    idr_cv_.notify_one();
}

void UnixVideoFrameReader::start_idr_worker()
{
    {
        std::lock_guard<std::mutex> lk(idr_mutex_);
        idr_worker_stop_ = false;
        idr_pending_ = false;
    }
    idr_worker_ = std::thread(&UnixVideoFrameReader::idr_worker_loop, this);
}

void UnixVideoFrameReader::stop_idr_worker()
{
    if (!idr_worker_.joinable()) return;
    {
        std::lock_guard<std::mutex> lk(idr_mutex_);
        idr_worker_stop_ = true;
    }
    idr_cv_.notify_one();
    idr_worker_.join();
}

void UnixVideoFrameReader::idr_worker_loop()
{
    bool have_fired = false;
    std::chrono::steady_clock::time_point last_fired{};

    for (;;) {
        std::unique_lock<std::mutex> lk(idr_mutex_);
        idr_cv_.wait(lk, [this] {
            return idr_worker_stop_ || idr_pending_;
        });
        if (idr_worker_stop_) return;

        idr_pending_ = false;
        lk.unlock();

        const auto now = std::chrono::steady_clock::now();
        if (have_fired && (now - last_fired) < idr_min_interval_) {
            continue;
        }
        have_fired = true;
        last_fired = now;

        if (idr_request_callback_) {
            idr_request_callback_();
        }
    }
}
