#include <thread>
#include <cerrno>
#include <unistd.h>
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <linux/videodev2.h>

#include <global_para/toml_default_para.hpp>
#include <tools/toml_file_parser.hpp>

#include "frame_capture.hpp"
#include "lt7911_utils.hpp"
#include "video_frame_protocol.h"

FrameCapture::FrameCapture(AxeraUtils& axera, std::atomic<bool>& running)
    : _axera(axera), _running(running)
{
}

FrameCapture::~FrameCapture()
{
    stop();
}

bool FrameCapture::start(int width, int height, int fps)
{
    {
        namespace GT = global_para::toml_default;
        using TP = tools::TomlFileParser;
        const int idle_ms = TP::get<int>(std::string(GT::H264_IDLE_TIMEOUT_MS_PATH))
                                .value_or(GT::H264_IDLE_TIMEOUT_MS_DEF);
        const int poll_ms = TP::get<int>(std::string(GT::H264_IDLE_POLL_MS_PATH))
                                .value_or(GT::H264_IDLE_POLL_MS_DEF);
        _idle_timeout_us = (idle_ms > 0)
            ? static_cast<uint64_t>(idle_ms) * 1000ULL
            : 0ULL;
        _idle_poll_ms = (poll_ms > 0) ? poll_ms : GT::H264_IDLE_POLL_MS_DEF;
        _idle_power_off_lt7911 =
            TP::get<bool>(std::string(GT::H264_IDLE_POWER_OFF_LT7911_PATH))
                .value_or(GT::H264_IDLE_POWER_OFF_LT7911_DEF);
    }

    _venc_fd = _axera.get_venc_fd();
    if (_venc_fd < 0) {
        _axera->error("Failed to get VENC fd");
        return false;
    }

    if (!_input_stable.load()
        && (!_axera.venc_start_recv() || !_axera.venc_stop_recv())) {
        _axera->error(
            "Failed to prepare VENC receive state before LT7911 is stable");
        _venc_fd = -1;
        return false;
    }

    _venc_wakeup_fd = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
    if (_venc_wakeup_fd < 0) {
        _venc_fd = -1;
        return false;
    }

    (void)width;
    (void)height;
    (void)fps;

    _venc_thread = std::thread(&FrameCapture::_venc_loop, this);

    return true;
}

void FrameCapture::stop()
{
    _running = false;

    if (_venc_wakeup_fd >= 0) {
        uint64_t one = 1;
        (void)write(_venc_wakeup_fd, &one, sizeof(one));
    }

    if (_venc_thread.joinable())  _venc_thread.join();

    if (_venc_wakeup_fd >= 0) {
        close(_venc_wakeup_fd);
        _venc_wakeup_fd = -1;
    }

    _venc_fd = -1;
}

void FrameCapture::update_resolution(int width, int height, int fps)
{
    (void)width;
    (void)height;
    (void)fps;
}

void FrameCapture::set_input_stable(bool stable)
{
    const bool was_stable = _input_stable.exchange(stable);
    if (was_stable == stable) {
        return;
    }

    _input_state_version.fetch_add(1);
    _axera->debug("VENC input gate {}",
                  stable ? "opened (LT7911 stable)"
                         : "closed (LT7911 not stable)");

    if (_venc_wakeup_fd >= 0) {
        uint64_t one = 1;
        (void)write(_venc_wakeup_fd, &one, sizeof(one));
    }
}

void FrameCapture::_venc_loop()
{
    AxeraUtils::VencStream stream{};
    if (_venc_fd < 0 || _venc_wakeup_fd < 0) {
        _axera->error("VENC/event fd unavailable, stop venc loop");
        return;
    }

    int epfd = epoll_create1(EPOLL_CLOEXEC);
    if (epfd < 0) {
        _axera->error("epoll_create1 failed, errno={}", errno);
        return;
    }

    epoll_event ev{};
    ev.events = EPOLLIN;
    ev.data.fd = _venc_fd;
    if (epoll_ctl(epfd, EPOLL_CTL_ADD, _venc_fd, &ev) < 0) {
        _axera->error("epoll_ctl add venc fd failed, errno={}", errno);
        close(epfd);
        return;
    }

    ev.events = EPOLLIN;
    ev.data.fd = _venc_wakeup_fd;
    if (epoll_ctl(epfd, EPOLL_CTL_ADD, _venc_wakeup_fd, &ev) < 0) {
        _axera->error("epoll_ctl add wakeup fd failed, errno={}", errno);
        close(epfd);
        return;
    }

    if (!_video_socket.start(KVM_VIN_VIDEO_SOCK_PATH, epfd, _axera)) {
        close(epfd);
        return;
    }

    epoll_event events[4]{};
    _venc_recv_on = false;
    _lt7911_power_on = true;
    bool wait_for_keyframe = true;
    uint64_t input_state_version = _input_state_version.load();

    if (_input_stable.load()) {
        if (_axera.venc_start_recv()) {
            _venc_recv_on = true;
        } else {
            _axera->error("Failed to start VENC RecvFrame at startup");
        }
    } else {
        _axera->debug("VENC encoder waiting for stable LT7911 input at startup");
    }

    if (_idle_timeout_us > 0) {
        if (_axera.venc_stop_recv()) {
            _venc_recv_on = false;
            _axera->debug("VENC encoder paused at startup (idle-pause enabled)");
        } else {
            _axera->error("Failed to stop VENC RecvFrame at startup");
        }

        if (_idle_power_off_lt7911) {
            _toggle_lt7911_power(false);
        }
    }

    if (_venc_recv_on) {
        (void)_axera.venc_request_idr(_axera.get_venc_chn(), true);
    }

    std::chrono::steady_clock::time_point last_reader_active{};

    while (_running) {
        int n = epoll_wait(epfd, events, 4, _idle_poll_ms);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            _axera->error("epoll_wait failed, errno={}", errno);
            continue;
        }

        bool should_exit = false;
        bool reader_connected = false;
        bool input_state_changed = false;

        auto refresh_input_state = [&]() {
            const uint64_t current_version = _input_state_version.load();
            if (current_version == input_state_version) {
                return false;
            }
            input_state_version = current_version;
            wait_for_keyframe = true;
            return true;
        };

        input_state_changed = refresh_input_state();

        for (int i = 0; i < n; ++i) {
            const int fd = events[i].data.fd;
            const uint32_t revents = events[i].events;

            if ((revents & (EPOLLERR | EPOLLHUP)) != 0) {
                _axera->error("epoll fd error/hup, fd={}, events=0x{:X}", fd, revents);
            }

            if (fd == _venc_wakeup_fd && (revents & EPOLLIN) != 0) {
                uint64_t wake_val = 0;
                while (read(_venc_wakeup_fd, &wake_val, sizeof(wake_val)) == sizeof(wake_val)) {
                }
                if (!_running) {
                    should_exit = true;
                    break;
                }
            }

            if (fd == _video_socket.listen_fd() || fd == _video_socket.client_fd()) {
                if (_video_socket.handle_event(fd, revents)) {
                    reader_connected = true;
                    wait_for_keyframe = true;
                }
                continue;
            }

            if (fd != _venc_fd || (revents & EPOLLIN) == 0) {
                continue;
            }

            input_state_changed = refresh_input_state() || input_state_changed;
            if (!_venc_recv_on || !_input_stable.load()) {
                std::this_thread::sleep_for(std::chrono::milliseconds(10));
                continue;
            }

            const auto frame_result = _axera.venc_get_frame(stream, 0);
            if (frame_result != AxeraUtils::VencFrameResult::Ok) {
                if (frame_result == AxeraUtils::VencFrameResult::Error) {
                    _axera->error("Failed to get venc frame");
                }
                continue;
            }

            input_state_changed = refresh_input_state() || input_state_changed;
            const bool input_stable = _input_stable.load();
            const bool key_frame = stream.raw.stPack.enCodingType == 0;
            const bool frame_ready =
                input_stable && (!wait_for_keyframe || key_frame);

            if (input_stable) {
                _axera->debug("Got venc frame: size={} pts={}",
                              stream.raw.stPack.u32Len,
                              stream.raw.stPack.u64PTS);

                if (key_frame) {
                    wait_for_keyframe = false;
                }

                if (frame_ready && _video_socket.has_client()) {
                    int w = 0;
                    int h = 0;
                    _axera.get_resolution(w, h);

                    const uint32_t vfmt =
                        (_axera.encode_mode() == AxeraUtils::EncodeMode::H265)
                            ? V4L2_PIX_FMT_HEVC
                            : V4L2_PIX_FMT_H264;
                    int fps = 0;
                    (void)_axera.get_fps(fps);

                    (void)_video_socket.send_frame(
                        stream.raw.stPack.pu8Addr,
                        stream.raw.stPack.u32Len,
                        static_cast<uint32_t>(w),
                        static_cast<uint32_t>(h),
                        vfmt,
                        key_frame,
                        static_cast<uint64_t>(stream.raw.stPack.u64PTS),
                        static_cast<uint32_t>(fps));
                }
            }

            if (!_axera.venc_release_frame(stream)) {
                _axera->error("Failed to release venc frame");
            }
        }

        if (should_exit) {
            break;
        }

        const auto now = std::chrono::steady_clock::now();
        bool reader_active = true;
        if (_idle_timeout_us > 0) {
            const bool reader_connected_now = _video_socket.has_client();
            if (reader_connected_now) {
                last_reader_active = now;
            }
            const bool reader_recently_active =
                last_reader_active != std::chrono::steady_clock::time_point{}
                && (now - last_reader_active
                    < std::chrono::microseconds(
                        static_cast<int64_t>(_idle_timeout_us)));
            reader_active = reader_connected_now || reader_recently_active;
        }

        if (reader_active && !_lt7911_power_on) {
            _toggle_lt7911_power(true);
        }

        input_state_changed = refresh_input_state() || input_state_changed;
        const bool input_stable = _input_stable.load();
        const bool should_receive = reader_active && input_stable;
        bool started_now = false;

        if (_venc_recv_on && !should_receive) {
            if (_axera.venc_stop_recv()) {
                _venc_recv_on = false;
                wait_for_keyframe = true;
                if (!input_stable) {
                    _axera->debug(
                        "VENC encoder paused (LT7911 input is not stable)");
                } else {
                    _axera->debug("VENC encoder paused (no reader for {} ms)",
                                  _idle_timeout_us / 1000);
                }
            } else if (_axera.main_venc_reconfiguring()) {
                _axera->debug("Skip VENC pause while main VENC is reconfiguring");
            } else {
                _axera->error("Failed to stop VENC RecvFrame");
            }
        } else if (!_venc_recv_on && should_receive) {
            if (_axera.venc_start_recv()) {
                _venc_recv_on = true;
                started_now = true;
                wait_for_keyframe = true;
                _axera->debug("VENC encoder resumed (LT7911 input is stable)");
                (void)_axera.venc_request_idr(_axera.get_venc_chn(), true);
            } else if (_axera.main_venc_reconfiguring()) {
                _axera->debug("Skip VENC resume while main VENC is reconfiguring");
            } else {
                _axera->error("Failed to start VENC RecvFrame");
            }
        }

        if (!started_now && _venc_recv_on && input_stable &&
            (reader_connected || input_state_changed)) {
            wait_for_keyframe = true;
            (void)_axera.venc_request_idr(_axera.get_venc_chn(), true);
        }

        if (_idle_timeout_us > 0 && !reader_active &&
            _idle_power_off_lt7911 && _lt7911_power_on) {
            _toggle_lt7911_power(false);
        }
    }

    if (!_lt7911_power_on) {
        _toggle_lt7911_power(true);
    }

    if (!_venc_recv_on) {
        _axera.venc_start_recv();
        _venc_recv_on = true;
    }

    _video_socket.stop();
    close(epfd);
}

void FrameCapture::_toggle_lt7911_power(bool on)
{
    if (!_lt7911) {
        _lt7911_power_on = on;
        return;
    }
    if (_lt7911->reset_lt7911(on)) {
        _lt7911_power_on = on;
        if (!on) {
            set_input_stable(false);
        }
        _axera->debug("LT7911 power {}", on ? "on" : "off");
    } else {
        _axera->error("Failed to {} LT7911", on ? "power on" : "power off");
    }
}
