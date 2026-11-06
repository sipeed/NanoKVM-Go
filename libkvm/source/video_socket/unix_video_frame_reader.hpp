#pragma once

#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <mutex>
#include <thread>
#include <vector>

struct FrameRef {
    const uint8_t* data {nullptr};
    uint32_t size {0};
    uint32_t width {0};
    uint32_t height {0};
    uint32_t format {0};
    bool key {false};
    uint64_t pts_us {0};
};

class UnixVideoFrameReader {
  public:
    explicit UnixVideoFrameReader(const char* socket_path);
    ~UnixVideoFrameReader();

    UnixVideoFrameReader(const UnixVideoFrameReader&) = delete;
    UnixVideoFrameReader& operator=(const UnixVideoFrameReader&) = delete;

    int read_frame(FrameRef& out, bool force_key = false, int timeout_ms = 500);
    void close_stream();

    void set_idr_request_callback(
        std::function<void()> callback,
        std::chrono::milliseconds min_interval = std::chrono::milliseconds(500));
    void request_idr() { trigger_idr(); }

  private:
    bool ensure_connected();
    void close_fd();
    bool wait_readable(const std::chrono::steady_clock::time_point& deadline);
    bool read_exact(void* data, size_t size, const std::chrono::steady_clock::time_point& deadline);

    void trigger_idr();
    void start_idr_worker();
    void stop_idr_worker();
    void idr_worker_loop();

    const char* socket_path_;
    int fd_ {-1};
    bool waiting_keyframe_ {true};
    bool idr_requested_for_connection_ {false};
    std::vector<uint8_t> frame_buffer_;
    std::mutex mutex_;

    std::function<void()> idr_request_callback_;
    std::chrono::milliseconds idr_min_interval_ {500};
    std::thread idr_worker_;
    std::mutex idr_mutex_;
    std::condition_variable idr_cv_;
    bool idr_pending_ {false};
    bool idr_worker_stop_ {false};
};
