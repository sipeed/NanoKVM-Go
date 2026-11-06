#include "control_client.hpp"

#include <cerrno>
#include <cstring>
#include <cstdlib>
#include <cstdio>

#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <spdlog/spdlog.h>

int KvmVinControlClient::request(const std::string& payload)
{
    int fd = ::socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) {
        spdlog::error("[control_client] socket failed: {}", std::strerror(errno));
        return -1;
    }

    sockaddr_un addr{};
    addr.sun_family = AF_UNIX;
    std::snprintf(addr.sun_path, sizeof(addr.sun_path), "%s", sock_path_.c_str());

    if (::connect(fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) != 0) {
        spdlog::error("[control_client] connect {} failed: {}", sock_path_, std::strerror(errno));
        ::close(fd);
        return -1;
    }

    if (::send(fd, payload.c_str(), payload.size(), 0) < 0) {
        spdlog::error("[control_client] send failed: {}", std::strerror(errno));
        ::close(fd);
        return -1;
    }

    char resp_buf[512] = {0};
    ssize_t n = ::recv(fd, resp_buf, sizeof(resp_buf) - 1, 0);
    ::close(fd);
    if (n <= 0) {
        spdlog::error("[control_client] recv failed");
        return -1;
    }

    std::string resp(resp_buf, static_cast<size_t>(n));
    bool ok = (resp.find("\"ok\":true") != std::string::npos);

    int ret_code = -1;
    auto pos = resp.find("\"ret_code\":");
    if (pos != std::string::npos) {
        pos += std::strlen("\"ret_code\":");
        ret_code = std::atoi(resp.c_str() + pos);
    }

    if (!ok) {
        spdlog::warn("[control_client] request failed: {}", resp);
        return ret_code == 0 ? -1 : ret_code;
    }
    return 0;
}

int KvmVinControlClient::set_fps(float src_fps, float dst_fps)
{
    char req[256] = {0};
    std::snprintf(req, sizeof(req),
                  "{\"cmd\":\"set_fps\",\"src_fps\":%.2f,\"dst_fps\":%.2f}",
                  src_fps, dst_fps);
    return request(req);
}

int KvmVinControlClient::set_gop(unsigned int gop)
{
    char req[256] = {0};
    std::snprintf(req, sizeof(req),
                  "{\"cmd\":\"set_gop\",\"gop\":%u}",
                  gop);
    return request(req);
}

int KvmVinControlClient::set_bitrate(unsigned int bitrate)
{
    char req[256] = {0};
    std::snprintf(req, sizeof(req),
                  "{\"cmd\":\"set_bitrate\",\"bitrate\":%u}",
                  bitrate);
    return request(req);
}

int KvmVinControlClient::set_rc_mode(int mode)
{
    char req[256] = {0};
    std::snprintf(req, sizeof(req),
                  "{\"cmd\":\"set_rc_mode\",\"mode\":%d}",
                  mode);
    return request(req);
}

int KvmVinControlClient::request_idr(bool instant)
{
    char req[256] = {0};
    std::snprintf(req, sizeof(req),
                  "{\"cmd\":\"request_idr\",\"instant\":%s}",
                  instant ? "true" : "false");
    return request(req);
}
