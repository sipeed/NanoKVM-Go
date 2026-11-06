#pragma once

#include <atomic>
#include <memory>
#include <thread>

#include <logger/ilogger.hpp>

#include "ax_runtime.hpp"
#include "ax_utils.hpp"
#include "frame_capture.hpp"
#include "lt7911_utils.hpp"

std::thread start_lt7911_monitor_thread(std::shared_ptr<AxVinData> ax_vin,
                                        AxeraUtils& axera_utils,
                                        Lt7911Utils& lt7911_utils,
                                        std::unique_ptr<FrameCapture>& frame_capture,
                                        std::atomic<bool>& running,
                                        ILogger& logger);
