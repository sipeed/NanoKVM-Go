#pragma once

#include <chrono>
#include <cstdint>
#include <mutex>
#include <string>

#include <global_para/toml_default_para.hpp>
#include <logger/ilogger.hpp>

#include "ax_utils.hpp"

struct TimelapseConfig {
    static constexpr const char* DEFAULT_OUTPUT_PATH =
        "/var/lib/memfab/timelapse";

    bool enable {false};
    int qp {global_para::toml_default::TIMELAPSE_QP_DEF};
    std::string output_path {DEFAULT_OUTPUT_PATH};
};

class TimelapseService {
  public:
    TimelapseService(AxeraUtils& axera_utils,
                     TimelapseConfig config,
                     ILogger& logger);
    ~TimelapseService();

    TimelapseService(const TimelapseService&) = delete;
    TimelapseService& operator=(const TimelapseService&) = delete;

    bool start();
    void stop();

    bool append_snapshot_frame(const AxeraUtils::SnapshotResult& result,
                               const std::string& captured_at,
                               const std::string& capture_id);

  private:
    AxeraUtils& _axera_utils;
    TimelapseConfig _config;
    ILogger& _logger;
    std::mutex _mutex;
    bool _started {false};
    uint64_t _encode_sequence {0};
    std::chrono::steady_clock::time_point _last_lock_contention_log;
    std::chrono::steady_clock::time_point _last_stale_day_log;
};
