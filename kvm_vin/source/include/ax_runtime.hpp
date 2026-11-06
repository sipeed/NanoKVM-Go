#pragma once

#include <atomic>
#include <memory>
#include <optional>
#include <thread>

#include <logger/ilogger.hpp>

#include "ax_utils.hpp"
#include "capture_coordinator.hpp"
#include "encoder_settings.hpp"
#include "frame_capture.hpp"
#include "snapshot_http_contract.hpp"
#include "snapshot_service.hpp"
#include "timelapse_service.hpp"

class Lt7911Utils;

struct AxVinData {
    int width;
    int height;
    int fps;
};

struct EncoderConfig {
    bool venc_enable {true};
    EncoderSettings settings;
    SnapshotConfig snapshot;
    TimelapseConfig timelapse;
};

bool wait_for_ax_module(int timeout_seconds = 10);

bool detect_initial_resolution(AxVinData& ax_vin, ILogger& logger,
                               int retries = 1, int interval_ms = 100);

bool run_dump_raw_mode(AxeraUtils& axera_utils, const AxVinData& ax_vin, ILogger& logger);

std::thread start_ax_runtime_thread(AxeraUtils& axera_utils,
                                    CaptureCoordinator& capture_coordinator,
                                    Lt7911Utils* lt7911_utils,
                                    std::unique_ptr<FrameCapture>& frame_capture,
                                    std::shared_ptr<AxVinData> ax_vin,
                                    EncoderConfig encoder_config,
                                    std::atomic<bool>& running,
                                    ILogger& logger,
                                    std::optional<snapshot_http::Config>
                                        snapshot_http_config = std::nullopt);
