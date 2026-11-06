#include "control_server.hpp"

#include <cerrno>
#include <cstring>
#include <string>

#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <unistd.h>
#include <fcntl.h>

#include <nlohmann/json.hpp>

namespace {

bool recv_request(int fd, std::string& req)
{
    char buf[4096] = {0};
    ssize_t n = recv(fd, buf, sizeof(buf) - 1, 0);
    if (n <= 0) return false;
    req.assign(buf, static_cast<size_t>(n));
    return true;
}

void send_response(int fd, bool ok, int ret_code, const std::string& msg)
{
    nlohmann::json resp = {
        {"ok", ok},
        {"ret_code", ret_code},
        {"message", msg}
    };
    std::string s = resp.dump();
    (void)send(fd, s.c_str(), s.size(), 0);
}

void handle_request(int fd,
                    const std::string& req_text,
                    AxeraUtils& axera,
                    ILogger& logger)
{
    try {
        auto req = nlohmann::json::parse(req_text);
        std::string cmd = req.value("cmd", "");

        if (!axera.venc_enabled() &&
            (cmd == "set_fps" || cmd == "set_gop" ||
             cmd == "set_bitrate" || cmd == "set_rc_mode" ||
             cmd == "request_idr")) {
            send_response(fd, false, -5, "venc disabled");
            return;
        }

        if (cmd == "set_fps") {
            int chn = axera.get_venc_chn();
            float src_fps = req.value("src_fps", 0.0f);
            float dst_fps = req.value("dst_fps", 0.0f);
            bool ok = axera.venc_set_fps(chn, src_fps, dst_fps);
            send_response(fd, ok, ok ? 0 : -1, ok ? "set_fps ok" : "set_fps failed");
            return;
        }

        if (cmd == "set_gop") {
            int chn = axera.get_venc_chn();
            unsigned int gop = req.value("gop", 0U);
            bool ok = axera.venc_set_gop(chn, gop);
            send_response(fd, ok, ok ? 0 : -1, ok ? "set_gop ok" : "set_gop failed");
            return;
        }

        if (cmd == "set_bitrate") {
            int chn = axera.get_venc_chn();
            unsigned int bitrate = req.value("bitrate", 0U);
            bool ok = axera.venc_set_bitrate(chn, bitrate);
            send_response(fd, ok, ok ? 0 : -1, ok ? "set_bitrate ok" : "set_bitrate failed");
            return;
        }

        if (cmd == "set_rc_mode") {
            int chn = axera.get_venc_chn();
            int mode = req.value("mode", -1);
            AX_VENC_RC_MODE_E rc_mode;
            const bool is_h265 = (axera.encode_mode() == AxeraUtils::EncodeMode::H265);
            if (mode == 0) {
                rc_mode = is_h265 ? AX_VENC_RC_MODE_H265CBR : AX_VENC_RC_MODE_H264CBR;
            } else if (mode == 1) {
                rc_mode = is_h265 ? AX_VENC_RC_MODE_H265VBR : AX_VENC_RC_MODE_H264VBR;
            } else {
                send_response(fd, false, -4, "invalid rc mode");
                return;
            }

            bool ok = axera.venc_set_rc_mode(chn, rc_mode);
            send_response(fd, ok, ok ? 0 : -1, ok ? "set_rc_mode ok" : "set_rc_mode failed");
            return;
        }

        if (cmd == "request_idr") {
            int chn = axera.get_venc_chn();
            bool instant = req.value("instant", true);
            logger->info("[control_server] request_idr received, chn={} instant={}", chn, instant);
            bool ok = axera.venc_request_idr(chn, instant);
            send_response(fd, ok, ok ? 0 : -1, ok ? "request_idr ok" : "request_idr failed");
            return;
        }

        send_response(fd, false, -2, "unknown cmd");
    } catch (const std::exception& e) {
        logger->error("[control_server] parse/handle failed: {}", e.what());
        send_response(fd, false, -3, e.what());
    }
}

} // namespace

std::thread start_control_server_thread(AxeraUtils& axera_utils,
                                        std::atomic<bool>& running,
                                        ILogger& logger)
{
    return std::thread([&axera_utils, &running, &logger]() {
        std::string sock_path = KVM_VIN_CTRL_SOCK_PATH;
        auto pos = sock_path.find_last_of('/');
        if (pos != std::string::npos && pos > 0) {
            std::string parent_dir = sock_path.substr(0, pos);
            if (::mkdir(parent_dir.c_str(), 0755) != 0 && errno != EEXIST) {
                logger->error("[control_server] mkdir {} failed: {}", parent_dir, std::strerror(errno));
                return;
            }
        }

        ::unlink(KVM_VIN_CTRL_SOCK_PATH);

        int server_fd = ::socket(AF_UNIX, SOCK_STREAM, 0);
        if (server_fd < 0) {
            logger->error("[control_server] socket failed: {}", std::strerror(errno));
            return;
        }

        sockaddr_un addr{};
        addr.sun_family = AF_UNIX;
        std::snprintf(addr.sun_path, sizeof(addr.sun_path), "%s", KVM_VIN_CTRL_SOCK_PATH);

        if (::bind(server_fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) != 0) {
            logger->error("[control_server] bind failed: {}", std::strerror(errno));
            ::close(server_fd);
            return;
        }

        if (::listen(server_fd, 8) != 0) {
            logger->error("[control_server] listen failed: {}", std::strerror(errno));
            ::close(server_fd);
            ::unlink(KVM_VIN_CTRL_SOCK_PATH);
            return;
        }

        int flags = fcntl(server_fd, F_GETFL, 0);
        if (flags >= 0) {
            (void)fcntl(server_fd, F_SETFL, flags | O_NONBLOCK);
        }

        logger->info("[control_server] listening on {}", KVM_VIN_CTRL_SOCK_PATH);

        while (running) {
            int client_fd = ::accept(server_fd, nullptr, nullptr);
            if (client_fd < 0) {
                if (errno == EINTR || errno == EAGAIN || errno == EWOULDBLOCK) {
                    usleep(20 * 1000);
                    continue;
                }
                usleep(20 * 1000);
                continue;
            }

            std::string req;
            if (recv_request(client_fd, req)) {
                handle_request(client_fd, req, axera_utils, logger);
            }
            ::close(client_fd);
        }

        ::close(server_fd);
        ::unlink(KVM_VIN_CTRL_SOCK_PATH);
    });
}
