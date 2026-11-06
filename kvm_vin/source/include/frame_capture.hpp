#pragma once

#include <atomic>
#include <cstdint>
#include <thread>
#include "ax_utils.hpp"
#include "video_socket_server.hpp"

class Lt7911Utils;

class FrameCapture {
  public:
    FrameCapture(AxeraUtils& axera, std::atomic<bool>& running);
    ~FrameCapture();

    bool start(int width, int height, int fps);
    void stop();
    void update_resolution(int width, int height, int fps);
    void set_input_stable(bool stable);

    void set_lt7911(Lt7911Utils* lt7911) { _lt7911 = lt7911; }

  private:
    AxeraUtils& _axera;
    std::atomic<bool>& _running;
    Lt7911Utils* _lt7911 {nullptr};

    VideoSocketServer _video_socket;

    std::thread _venc_thread;
    int _venc_fd        {-1};
    int _venc_wakeup_fd {-1};
    bool _venc_recv_on  {true};
    bool _lt7911_power_on {true};
    std::atomic<bool> _input_stable {true};
    std::atomic<uint64_t> _input_state_version {0};

    uint64_t _idle_timeout_us       {0};
    int      _idle_poll_ms          {500};
    bool     _idle_power_off_lt7911 {false};

    void _venc_loop();
    void _toggle_lt7911_power(bool on);
};
