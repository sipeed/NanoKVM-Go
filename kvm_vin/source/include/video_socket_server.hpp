#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <string>

#include <spdlog/logger.h>

#include <logger/ilogger.hpp>

class VideoSocketServer {
  public:
    bool start(const char* path, int epoll_fd, ILogger& logger);
    void stop();

    bool handle_event(int fd, uint32_t events);
    bool has_client() const { return client_fd_ >= 0; }
    int listen_fd() const { return listen_fd_; }
    int client_fd() const { return client_fd_; }

    bool send_frame(const uint8_t* data,
                    uint32_t size,
                    uint32_t width,
                    uint32_t height,
                    uint32_t format,
                    bool key_frame,
                    uint64_t pts_us,
                    uint32_t fps);

  private:
    bool add_fd_to_epoll(int fd, uint32_t events);
    void remove_fd_from_epoll(int fd);
    void accept_pending_clients();
    void replace_client(int fd);
    void close_client();
    bool send_all(const uint8_t* data, size_t size);

    std::shared_ptr<spdlog::logger> logger_;
    std::string path_;
    int epoll_fd_ {-1};
    int listen_fd_ {-1};
    int client_fd_ {-1};
    uint64_t seq_ {0};
};
