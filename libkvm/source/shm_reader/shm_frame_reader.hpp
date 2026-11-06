#pragma once

#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <cstring>
#include <functional>
#include <mutex>
#include <thread>
#include <vector>
#include <sys/mman.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

#include "../../../kvm_vin/source/include/shm_frame.h"

struct FrameRef {
    const uint8_t* data {nullptr};
    uint32_t size {0};
    uint32_t width {0};
    uint32_t height {0};
    uint32_t format {0};
    bool key {false};
    uint64_t pts_us {0};
};

class ShmFrameReader {
public:
    explicit ShmFrameReader(const char* shm_name, bool = false)
        : shm_name_(shm_name) {}

    ~ShmFrameReader() {
        stop_idr_worker();
        close_mapping();
    }

    ShmFrameReader(const ShmFrameReader&) = delete;
    ShmFrameReader& operator=(const ShmFrameReader&) = delete;

    /**
     * Latest-frame read: copies the current shm frame into an internal buffer.
     * The returned pointer remains valid until the next read_frame() call.
     * @param out       Output frame reference (pointer into the internal buffer)
     * @param force_key If true, skip non-keyframes until a keyframe is found
     * @param timeout_ms  Timeout in milliseconds; pass -1 to wait indefinitely
     * @return 0 on success, -1 on failure/timeout
     */
    int read_frame(FrameRef& out, bool force_key = false, int timeout_ms = 300);

    /**
     * Set callback to request an IDR frame when the reader falls behind.
     * The callback runs on a dedicated worker thread (never in the read path)
     * and is rate-limited, so triggering it from the hot loop is cheap and
     * never blocks. Pass a min interval to coalesce bursts of triggers.
     * @param callback     Function performing the (possibly blocking) IDR request
     * @param min_interval Minimum spacing between two actual IDR requests
     */
    void set_idr_request_callback(
        std::function<void()> callback,
        std::chrono::milliseconds min_interval = std::chrono::milliseconds(500)) {
        stop_idr_worker();
        idr_request_callback_ = std::move(callback);
        idr_min_interval_ = min_interval;
        if (idr_request_callback_) {
            start_idr_worker();
        }
    }

    /**
     * Request an IDR frame on demand (e.g. from an external caller such as the
     * WebRTC RTCP path). Like the internal fall-behind trigger, this only sets a
     * flag and wakes the worker thread; the blocking socket round-trip happens
     * on that thread, so the caller never blocks. Rate-limited and coalesced
     * with the same min_interval as the automatic triggers. No-op until a
     * callback has been installed via set_idr_request_callback().
     */
    void request_idr() { trigger_idr(); }

private:
    bool ensure_open();
    void close_mapping();

    /* Lightweight signal from the read loop. Sets the flag under idr_mutex_
       (held only momentarily, never during the worker's socket call) to avoid
       lost wakeups, then notifies. Does not block on I/O. */
    void trigger_idr() {
        {
            std::lock_guard<std::mutex> lk(idr_mutex_);
            idr_pending_ = true;
        }
        idr_cv_.notify_one();
    }

    void start_idr_worker();
    void stop_idr_worker();
    void idr_worker_loop();

    const char* shm_name_;
    shm_frame_handle_t* handle_ {nullptr};
    shm_frame_ring_t* ring_ {nullptr};
    size_t map_size_ {0};
    uint64_t last_frame_id_ {0};
    ino_t map_ino_ {0};
    std::vector<uint8_t> frame_buffer_;
    std::mutex mutex_;

    /* IDR request offloading */
    std::function<void()> idr_request_callback_;
    std::chrono::milliseconds idr_min_interval_ {500};
    std::thread idr_worker_;
    std::mutex idr_mutex_;
    std::condition_variable idr_cv_;
    bool idr_pending_ {false};
    bool idr_worker_stop_ {false};
};
