#include <chrono>
#include <utility>

#include "snapshot_service.hpp"

#include <sys/timex.h>

#include "ocr_latest_frame_store.hpp"
#include "timelapse_service.hpp"

namespace {

#if defined(KVM_VIN_TEST_SNAPSHOT_INTERVAL_MS)
constexpr auto kPeriodicSnapshotInterval =
    std::chrono::milliseconds(KVM_VIN_TEST_SNAPSHOT_INTERVAL_MS);
#else
constexpr auto kPeriodicSnapshotInterval = std::chrono::seconds(20);
#endif

bool is_time_synchronized()
{
    struct timex tx {};
    const int state = adjtimex(&tx);
    return state != TIME_ERROR && (tx.status & STA_UNSYNC) == 0;
}

}  // namespace

SnapshotService::SnapshotService(CaptureCoordinator& capture_coordinator,
                                 SnapshotConfig config,
                                 std::atomic<bool>& running,
                                 ILogger& logger,
                                 TimelapseService* timelapse_service,
                                 OCRLatestFrameStore* ocr_latest_frame_store)
    : _capture_coordinator(capture_coordinator)
    , _config(std::move(config))
    , _running(running)
    , _logger(logger)
    , _timelapse_service(timelapse_service)
    , _ocr_latest_frame_store(ocr_latest_frame_store)
{}

SnapshotService::~SnapshotService() { stop(); }

void SnapshotService::start()
{
    if (!_config.enable || _worker.joinable()) {
        return;
    }

    {
        std::lock_guard lock(_mutex);
        _stop_requested = false;
    }

    _worker = std::thread(&SnapshotService::run, this);
}

void SnapshotService::stop()
{
    {
        std::lock_guard lock(_mutex);
        _stop_requested = true;
        if (_active_ticket != nullptr) {
            _active_ticket->cancel();
        }
    }
    _cv.notify_all();

    if (_worker.joinable()) {
        _worker.join();
    }
}

void SnapshotService::run()
{
    auto next_snapshot_time = std::chrono::steady_clock::now();
    bool time_synchronized = false;

    _logger->info(
        "[snapshot] periodic canonical capture enabled: interval={}ms; "
        "JPEG disk persistence disabled",
        std::chrono::duration_cast<std::chrono::milliseconds>(
            kPeriodicSnapshotInterval)
            .count());

    while (_running) {
        {
            std::unique_lock lock(_mutex);
            if (_stop_requested) {
                break;
            }
        }

        const auto now = std::chrono::steady_clock::now();
        if (now >= next_snapshot_time) {
            if (!time_synchronized) {
                time_synchronized = is_time_synchronized();
            }

            if (time_synchronized) {
                (void)capture_and_publish_if_changed();
            } else {
                _logger->debug(
                    "[snapshot] skip periodic snapshot: time not synchronized");
            }

            next_snapshot_time =
                std::chrono::steady_clock::now() + kPeriodicSnapshotInterval;
        }

        std::unique_lock lock(_mutex);
        if (_cv.wait_for(lock, std::chrono::milliseconds(200), [this]() {
                return _stop_requested;
            })) {
            break;
        }
    }
}

bool SnapshotService::capture_and_publish_if_changed()
{
    auto ticket = _capture_coordinator.submit(CaptureRequest {
        .x = 0,
        .y = 0,
        .w = 0,
        .h = 0,
        .quality = 75,
        .timeout_ms = 1000,
        .source = CaptureSource::Periodic,
    });

    {
        std::lock_guard lock(_mutex);
        if (_stop_requested) {
            ticket.cancel();
        } else {
            _active_ticket = &ticket;
        }
    }

    const auto response = ticket.wait();
    {
        std::lock_guard lock(_mutex);
        if (_active_ticket == &ticket) {
            _active_ticket = nullptr;
        }
    }

    if (!response.ok()) {
        _logger->error("[snapshot] periodic snapshot failed, ret={}",
                       response.ret_code);
        return false;
    }

    return publish_if_changed(response);
}

bool SnapshotService::publish_if_changed(const CaptureResponse& response)
{
    const auto& result = *response.frame;
    const auto clear_pending = [this]() {
        _pending_response = {};
        _has_pending_response = false;
        _pending_published = false;
        _pending_timelapse_appended = false;
    };

    if (!_has_pending_response || !_pending_response.frame
        || _pending_response.frame->data != result.data) {
        _pending_response = response;
        _has_pending_response = true;
        _pending_published = _ocr_latest_frame_store == nullptr
            || (!_last_ocr_jpeg.empty() && result.data == _last_ocr_jpeg);
        _pending_timelapse_appended = _timelapse_service == nullptr
            || (!_last_timelapse_jpeg.empty()
                && result.data == _last_timelapse_jpeg);
    }

    if (!_pending_published) {
        if (_ocr_latest_frame_store->publish(_pending_response)) {
            _pending_published = true;
            _last_ocr_jpeg = _pending_response.frame->data;
        } else {
            _logger->warn(
                "[snapshot] failed to publish canonical OCR frame: {}",
                _pending_response.capture_id);
        }
    }

    if (!_pending_timelapse_appended) {
        const auto& pending = _pending_response;
        _pending_timelapse_appended = _timelapse_service->append_snapshot_frame(
            *pending.frame, pending.captured_at, pending.capture_id);
        if (!_pending_timelapse_appended) {
            _logger->warn(
                "[snapshot] failed to append canonical Timelapse frame: {}",
                pending.capture_id);
        } else {
            _last_timelapse_jpeg = pending.frame->data;
        }
    }

    const bool succeeded = _pending_published && _pending_timelapse_appended;
    if (succeeded) {
        _logger->debug("[snapshot] canonical JPEG distributed ({} bytes)",
                       result.data.size());
        clear_pending();
    }

    return succeeded;
}
