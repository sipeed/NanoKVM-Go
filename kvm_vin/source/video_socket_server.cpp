#include "video_socket_server.hpp"

#include <cerrno>
#include <cstring>
#include <string>

#include <fcntl.h>
#include <sys/epoll.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <unistd.h>

#include "video_frame_protocol.h"

namespace {

bool set_nonblocking(int fd)
{
    int flags = fcntl(fd, F_GETFL, 0);
    if (flags < 0) return false;
    return fcntl(fd, F_SETFL, flags | O_NONBLOCK) == 0;
}

void ensure_parent_dir(const std::string& path)
{
    const auto pos = path.find_last_of('/');
    if (pos == std::string::npos || pos == 0) return;
    const std::string parent = path.substr(0, pos);
    (void)::mkdir(parent.c_str(), 0755);
}

} // namespace

bool VideoSocketServer::start(const char* path, int epoll_fd, ILogger& logger)
{
    logger_ = logger.operator->();
    path_ = path ? path : KVM_VIN_VIDEO_SOCK_PATH;
    epoll_fd_ = epoll_fd;

    ensure_parent_dir(path_);
    (void)::unlink(path_.c_str());

    listen_fd_ = ::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
    if (listen_fd_ < 0) {
        logger_->error("[video_socket] socket failed: {}", std::strerror(errno));
        return false;
    }

    if (!set_nonblocking(listen_fd_)) {
        logger_->error("[video_socket] set nonblocking failed: {}", std::strerror(errno));
        stop();
        return false;
    }

    sockaddr_un addr{};
    addr.sun_family = AF_UNIX;
    std::snprintf(addr.sun_path, sizeof(addr.sun_path), "%s", path_.c_str());

    if (::bind(listen_fd_, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) != 0) {
        logger_->error("[video_socket] bind {} failed: {}", path_, std::strerror(errno));
        stop();
        return false;
    }

    if (::listen(listen_fd_, 2) != 0) {
        logger_->error("[video_socket] listen failed: {}", std::strerror(errno));
        stop();
        return false;
    }

    if (!add_fd_to_epoll(listen_fd_, EPOLLIN)) {
        stop();
        return false;
    }

    return true;
}

void VideoSocketServer::stop()
{
    close_client();

    if (listen_fd_ >= 0) {
        remove_fd_from_epoll(listen_fd_);
        ::close(listen_fd_);
        listen_fd_ = -1;
    }

    if (!path_.empty()) {
        (void)::unlink(path_.c_str());
    }
}

bool VideoSocketServer::add_fd_to_epoll(int fd, uint32_t events)
{
    if (epoll_fd_ < 0 || fd < 0) return false;

    epoll_event ev{};
    ev.events = events;
    ev.data.fd = fd;
    if (epoll_ctl(epoll_fd_, EPOLL_CTL_ADD, fd, &ev) < 0) {
        if (logger_) {
            logger_->error("[video_socket] epoll add fd={} failed: {}", fd, std::strerror(errno));
        }
        return false;
    }
    return true;
}

void VideoSocketServer::remove_fd_from_epoll(int fd)
{
    if (epoll_fd_ >= 0 && fd >= 0) {
        (void)epoll_ctl(epoll_fd_, EPOLL_CTL_DEL, fd, nullptr);
    }
}

bool VideoSocketServer::handle_event(int fd, uint32_t events)
{
    bool connected = false;

    if (fd == listen_fd_) {
        accept_pending_clients();
        connected = has_client();
    } else if (fd == client_fd_) {
        if ((events & (EPOLLERR | EPOLLHUP | EPOLLRDHUP)) != 0) {
            close_client();
        } else if ((events & EPOLLIN) != 0) {
            char buf[64];
            ssize_t n = ::recv(client_fd_, buf, sizeof(buf), MSG_DONTWAIT);
            if (n <= 0) {
                close_client();
            }
        }
    }

    return connected;
}

void VideoSocketServer::accept_pending_clients()
{
    for (;;) {
        int fd = ::accept4(listen_fd_, nullptr, nullptr, SOCK_CLOEXEC);
        if (fd < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR) {
                return;
            }
            if (logger_) {
                logger_->warn("[video_socket] accept failed: {}", std::strerror(errno));
            }
            return;
        }

        replace_client(fd);
    }
}

void VideoSocketServer::replace_client(int fd)
{
    close_client();

    client_fd_ = fd;
    if (!add_fd_to_epoll(client_fd_, EPOLLIN | EPOLLRDHUP | EPOLLERR | EPOLLHUP)) {
        ::close(client_fd_);
        client_fd_ = -1;
        return;
    }

}

void VideoSocketServer::close_client()
{
    if (client_fd_ < 0) return;

    remove_fd_from_epoll(client_fd_);
    ::close(client_fd_);
    client_fd_ = -1;
}

bool VideoSocketServer::send_frame(const uint8_t* data,
                                   uint32_t size,
                                   uint32_t width,
                                   uint32_t height,
                                   uint32_t format,
                                   bool key_frame,
                                   uint64_t pts_us,
                                   uint32_t fps)
{
    if (client_fd_ < 0 || data == nullptr || size == 0) {
        return false;
    }

    kvm_video_frame_header_t header{};
    header.magic = KVM_VIDEO_FRAME_MAGIC;
    header.version = KVM_VIDEO_FRAME_VERSION;
    header.header_len = KVM_VIDEO_FRAME_HEADER_LEN;
    header.payload_len = size;
    header.width = width;
    header.height = height;
    header.format = format;
    header.flags = key_frame ? KVM_VIDEO_FRAME_FLAG_KEY : 0;
    header.pts_us = pts_us;
    header.seq = ++seq_;
    header.fps = fps;

    if (!send_all(reinterpret_cast<const uint8_t*>(&header), sizeof(header))) {
        return false;
    }
    if (!send_all(data, size)) {
        return false;
    }

    return true;
}

bool VideoSocketServer::send_all(const uint8_t* data, size_t size)
{
    size_t sent = 0;
    while (sent < size) {
        ssize_t n = ::send(client_fd_, data + sent, size - sent, MSG_NOSIGNAL);
        if (n > 0) {
            sent += static_cast<size_t>(n);
            continue;
        }
        if (n < 0 && errno == EINTR) {
            continue;
        }
        if (n < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            close_client();
            return false;
        }

        close_client();
        return false;
    }
    return true;
}
