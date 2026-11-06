#include <chrono>
#include <cstring>
#include <cerrno>
#include <fcntl.h>
#include <sys/epoll.h>
#include <unistd.h>

#include "lt7911_monitor.hpp"

enum class Lt7911State {
    WaitEvent,
    Stable,
    Disappeared,
    ResError,
    NewRes,
    Handshake,
    Reconfigure,
};

static Lt7911State parse_device_status(const std::string& status)
{
    if (status == "stable")                               return Lt7911State::Stable;
    if (status == "disappear")                            return Lt7911State::Disappeared;
    if (status == "unknown res" || status == "error res") return Lt7911State::ResError;
    if (status == "new res")                              return Lt7911State::NewRes;
    return Lt7911State::WaitEvent;
}

std::thread start_lt7911_monitor_thread(std::shared_ptr<AxVinData> ax_vin,
                                        AxeraUtils& axera_utils,
                                        Lt7911Utils& lt7911_utils,
                                        std::unique_ptr<FrameCapture>& frame_capture,
                                        std::atomic<bool>& running,
                                        ILogger& logger)
{
    return std::thread([ax_vin, &axera_utils, &lt7911_utils, &frame_capture, &running, &logger]() {
        const char* status_path = "/proc/lt7911_info/status";
        const auto set_input_stable = [&frame_capture](bool stable) {
            if (frame_capture) {
                frame_capture->set_input_stable(stable);
            }
        };

        set_input_stable(false);

        int status_fd = open(status_path, O_RDONLY);
        if (status_fd < 0) {
            logger->error("Failed to open {} for epoll: {}", status_path, strerror(errno));
            return;
        }

        int epfd = epoll_create1(0);
        if (epfd < 0) {
            logger->error("Failed to create epoll instance: {}", strerror(errno));
            close(status_fd);
            return;
        }

        struct epoll_event ev;
        ev.events = EPOLLPRI | EPOLLERR;
        ev.data.fd = status_fd;
        if (epoll_ctl(epfd, EPOLL_CTL_ADD, status_fd, &ev) < 0) {
            logger->error("Failed to add fd to epoll: {}", strerror(errno));
            close(status_fd);
            close(epfd);
            return;
        }

        char dummy_buf[128];
        read(status_fd, dummy_buf, sizeof(dummy_buf));

        std::string last_status;
        std::string status;
        if (lt7911_utils.read_status(status)) {
            last_status = status;
        }

        Lt7911State state = Lt7911State::WaitEvent;

        if (!last_status.empty()) {
            state = parse_device_status(last_status);
            logger->debug("LT7911 initial status: {}", last_status);
        }

        int new_width = 0, new_height = 0, new_fps = 0;
        struct epoll_event events[1];

        while (running) {
            switch (state) {

            case Lt7911State::WaitEvent: {
                int nfds = epoll_wait(epfd, events, 1, 2000);

                if (nfds < 0) {
                    if (errno == EINTR) break;
                    logger->error("epoll_wait error: {}", strerror(errno));
                    std::this_thread::sleep_for(std::chrono::milliseconds(1000));
                    break;
                }
                if (nfds == 0) break;

                if (!(events[0].events & EPOLLPRI)) break;

                lseek(status_fd, 0, SEEK_SET);
                int len = read(status_fd, dummy_buf, sizeof(dummy_buf));
                (void)len;

                if (!lt7911_utils.read_status(status)) {
                    logger->error("Failed to read LT7911 status");
                    set_input_stable(false);
                    break;
                }

                if (status != last_status) {
                    logger->debug("LT7911 status changed: {} -> {}", last_status, status);
                    last_status = status;
                }

                state = parse_device_status(status);
                if (state != Lt7911State::Stable) {
                    set_input_stable(false);
                }
                break;
            }

            case Lt7911State::Stable: {
                logger->debug("LT7911 status is stable");
                set_input_stable(true);
                state = Lt7911State::WaitEvent;
                break;
            }

            case Lt7911State::Disappeared: {
                set_input_stable(false);
                logger->debug("LT7911 status is disappear, waiting...");
                state = Lt7911State::WaitEvent;
                break;
            }

            case Lt7911State::ResError: {
                set_input_stable(false);
                logger->error("Failed to read LT7911 status: {}, resetting LT7911", status);
                lt7911_utils.reset_lt7911(false);
                std::this_thread::sleep_for(std::chrono::milliseconds(500));
                lt7911_utils.reset_lt7911(true);
                state = Lt7911State::WaitEvent;
                break;
            }

            case Lt7911State::NewRes: {
                set_input_stable(false);
                logger->info("LT7911 detected new resolution, checking...");

                new_width = 0;
                new_height = 0;
                new_fps = 0;
                lt7911_utils.read_res(new_width, new_height, new_fps);

                if (new_width == 0 || new_height == 0 || new_fps == 0) {
                    logger->error("Failed to read LT7911 resolution: {}x{}@{}", new_width, new_height, new_fps);
                    state = Lt7911State::WaitEvent;
                    break;
                }

                logger->info("LT7911 resolution changed to {}x{}, fps:{}", new_width, new_height, new_fps);

                if (!lt7911_utils.write_status("ok")) {
                    logger->error("Failed to write LT7911 status");
                    state = Lt7911State::WaitEvent;
                    break;
                }

                state = Lt7911State::Handshake;
                break;
            }

            case Lt7911State::Handshake: {
                set_input_stable(false);
                std::this_thread::sleep_for(std::chrono::milliseconds(200));

                std::string check_status;
                if (!lt7911_utils.read_status(check_status)) {
                    logger->error("Failed to verify status after handshake");
                    state = Lt7911State::WaitEvent;
                    break;
                }
                last_status = check_status;

                if (check_status != "stable") {
                    logger->error("LT7911 status is not stable after resolution change: {}", check_status);
                    state = Lt7911State::WaitEvent;
                    break;
                }

                state = Lt7911State::Reconfigure;
                break;
            }

            case Lt7911State::Reconfigure: {
                int current_width = 0;
                int current_height = 0;
                int current_fps = 0;
                axera_utils.get_resolution(current_width, current_height);
                axera_utils.get_fps(current_fps);
                if (current_width == new_width && current_height == new_height && current_fps == new_fps) {
                    logger->info("Resolution already set to {}x{}@{}, skipping", new_width, new_height, new_fps);
                    state = Lt7911State::Stable;
                    break;
                }

                if (!axera_utils.reconfigure_vin_pipeline(new_width, new_height, new_fps)) {
                    logger->error("VIN pipeline reconfigure failed for {}x{}@{}",
                                  new_width, new_height, new_fps);
                    state = Lt7911State::WaitEvent;
                    break;
                }

                ax_vin->width = new_width;
                ax_vin->height = new_height;
                ax_vin->fps = new_fps;
                if (frame_capture) {
                    frame_capture->update_resolution(new_width, new_height, new_fps);
                }

                state = Lt7911State::Stable;
                break;
            }

            } // switch
        } // while

        close(epfd);
        close(status_fd);
        logger->info("lt7911 thread exiting...");
    });
}
