#pragma once

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <mutex>
#include <thread>
#include <vector>

#include <logger/ilogger.hpp>

#include "capture_coordinator.hpp"

class TimelapseService;
class OCRLatestFrameStore;

struct SnapshotConfig {
    bool enable {true};
    bool periodic_enable {false};
};

class SnapshotService {
  public:
    SnapshotService(CaptureCoordinator& capture_coordinator,
                    SnapshotConfig config,
                    std::atomic<bool>& running,
                    ILogger& logger,
                    TimelapseService* timelapse_service = nullptr,
                    OCRLatestFrameStore* ocr_latest_frame_store = nullptr);
    ~SnapshotService();

    SnapshotService(const SnapshotService&) = delete;
    SnapshotService& operator=(const SnapshotService&) = delete;

    void start();
    void stop();

  private:
    void run();
    bool capture_and_publish_if_changed();
    bool publish_if_changed(const CaptureResponse& response);

    CaptureCoordinator& _capture_coordinator;
    SnapshotConfig _config;
    std::atomic<bool>& _running;
    ILogger& _logger;
    TimelapseService* _timelapse_service {nullptr};
    OCRLatestFrameStore* _ocr_latest_frame_store {nullptr};
    std::thread _worker;
    std::mutex _mutex;
    std::condition_variable _cv;
    bool _stop_requested {false};
    CaptureTicket* _active_ticket {nullptr};
    CaptureResponse _pending_response;
    bool _has_pending_response {false};
    bool _pending_published {false};
    bool _pending_timelapse_appended {false};
    std::vector<uint8_t> _last_ocr_jpeg;
    std::vector<uint8_t> _last_timelapse_jpeg;
};
