#include "sleep_notifier.hpp"

#include <arpa/inet.h>
#include <cerrno>
#include <cstring>
#include <fcntl.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/types.h>
#include <sys/select.h>
#include <unistd.h>
#include <chrono>

sleep_notifier& sleep_notifier::instance() {
    static sleep_notifier inst;
    return inst;
}

sleep_notifier::sleep_notifier() = default;

sleep_notifier::~sleep_notifier() {
    stop();
}

void sleep_notifier::start() {
    bool expected = false;
    if (!running_.compare_exchange_strong(expected, true))
        return;

    worker_thread_ = std::thread([this]{ thread_func(); });
}

void sleep_notifier::stop() {
    bool expected = true;
    if (!running_.compare_exchange_strong(expected, false)) {
        return;
    }

    {
        std::lock_guard<std::mutex> lk(lock_);
        cv_.notify_all();
    }

    if (worker_thread_.joinable())
        worker_thread_.join();

    if (socket_fd_ != -1) {
        close(socket_fd_);
        socket_fd_ = -1;
    }
}

bool sleep_notifier::send(sleep_msg msg) {
    std::lock_guard<std::mutex> lk(lock_);
    if (send_queue_.size() >= max_queue_size_) {
        send_queue_.pop_front();
    }
    send_queue_.push_back(msg);
    cv_.notify_one();
    return true;
}

void sleep_notifier::thread_func() {
    while (running_.load()) {
        if (socket_fd_ < 0) {
            if (!ensure_connected()) {
                std::this_thread::sleep_for(std::chrono::milliseconds(reconnect_interval_ms_));
                continue;
            }
        }

        std::unique_lock<std::mutex> lk(lock_);
        cv_.wait(lk, [this]{ return !send_queue_.empty() || !running_.load(); });

        if (!running_.load()) break;

        sleep_msg msg = send_queue_.front();
        send_queue_.pop_front();
        lk.unlock();

        uint8_t buf = static_cast<uint8_t>(msg);
        if (!send_all(reinterpret_cast<const char*>(&buf), sizeof(buf))) {
            if (socket_fd_ != -1) {
                close(socket_fd_);
                socket_fd_ = -1;
            }
            std::lock_guard<std::mutex> lk2(lock_);
            if (send_queue_.size() < max_queue_size_) {
                send_queue_.push_front(msg);
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(reconnect_interval_ms_));
        }
    }

    if (socket_fd_ != -1) {
        close(socket_fd_);
        socket_fd_ = -1;
    }
}

bool sleep_notifier::ensure_connected() {
    if (socket_fd_ >= 0) return true;
    int fd = connect_nonblock(socket_path_, connect_timeout_ms_);
    if (fd < 0) return false;
    socket_fd_ = fd;
    return true;
}

int sleep_notifier::connect_nonblock(const char* path, int timeout_ms) {
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) return -1;

    int flags = fcntl(fd, F_GETFL, 0);
    if (flags == -1) flags = 0;
    fcntl(fd, F_SETFL, flags | O_NONBLOCK);

    sockaddr_un addr{};
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, path, sizeof(addr.sun_path) - 1);

    int rc = connect(fd, (sockaddr*)&addr, sizeof(addr));
    if (rc == 0) {
        fcntl(fd, F_SETFL, flags);
        return fd;
    } else if (errno != EINPROGRESS) {
        close(fd);
        return -1;
    }

    fd_set wfds;
    FD_ZERO(&wfds);
    FD_SET(fd, &wfds);
    timeval tv{};
    tv.tv_sec = timeout_ms / 1000;
    tv.tv_usec = (timeout_ms % 1000) * 1000;

    rc = select(fd + 1, nullptr, &wfds, nullptr, &tv);
    if (rc <= 0) { close(fd); return -1; }

    int so_error = 0;
    socklen_t len = sizeof(so_error);
    if (getsockopt(fd, SOL_SOCKET, SO_ERROR, &so_error, &len) < 0) {
        close(fd); return -1;
    }
    if (so_error != 0) {
        close(fd); return -1;
    }

    fcntl(fd, F_SETFL, flags);
    return fd;
}

bool sleep_notifier::send_all(const char* buf, size_t len) {
    size_t sent = 0;
    while (sent < len) {
        ssize_t n = ::send(socket_fd_, buf + sent, len - sent, MSG_NOSIGNAL);
        if (n < 0) {
            if (errno == EINTR) continue;
            return false;
        }
        sent += static_cast<size_t>(n);
    }
    return true;
}
