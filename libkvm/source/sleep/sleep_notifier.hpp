#pragma once

#include <atomic>
#include <condition_variable>
#include <deque>
#include <mutex>
#include <thread>
#include <cstdint>

enum class sleep_msg : uint8_t {
    idle = 0,
    active = 1,
};

class sleep_notifier {
public:
    static sleep_notifier& instance();

    void start();
    void stop();
    bool send(sleep_msg msg);

private:
    sleep_notifier();
    ~sleep_notifier();
    sleep_notifier(const sleep_notifier&) = delete;
    sleep_notifier& operator=(const sleep_notifier&) = delete;

    void thread_func();
    bool ensure_connected();
    int connect_nonblock(const char* path, int timeout_ms);
    bool send_all(const char* buf, size_t len);

private:
    std::atomic<bool> running_{false};
    std::thread worker_thread_;
    std::mutex lock_;
    std::condition_variable cv_;
    std::deque<sleep_msg> send_queue_;
    int socket_fd_{-1};

    const char* socket_path_ = "/run/kvm/kvm_sleep_sock";
    int reconnect_interval_ms_ = 1000;
    int connect_timeout_ms_ = 2000;
    size_t max_queue_size_ = 1;
};
