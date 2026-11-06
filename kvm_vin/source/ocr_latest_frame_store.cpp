#include <algorithm>
#include <limits>
#include <utility>

#include "ocr_latest_frame_store.hpp"

OCRLatestFrameStore::~OCRLatestFrameStore() { stop(); }

bool OCRLatestFrameStore::publish(const CaptureResponse& response)
{
    if (!response.ok() || response.session_id.empty()
        || response.capture_id.empty() || response.captured_at.empty()
        || response.frame->width <= 0 || response.frame->height <= 0
        || response.frame->data.empty()) {
        return false;
    }

    {
        std::lock_guard lock(_mutex);
        if (_stopped) {
            return false;
        }

        const bool new_session = response.session_id != _session_id;
        if (!new_session
            && _sequence == std::numeric_limits<std::uint64_t>::max()) {
            return false;
        }

        const std::uint64_t next_sequence = new_session ? 1 : _sequence + 1;
        auto latest = std::make_shared<const OCRLatestFrame>(OCRLatestFrame {
            .frame = response.frame,
            .session_id = response.session_id,
            .capture_id = response.capture_id,
            .captured_at = response.captured_at,
            .sequence = next_sequence,
        });

        _session_id = response.session_id;
        _sequence = next_sequence;
        _latest = std::move(latest);
    }

    _cv.notify_all();
    return true;
}

OCRLatestFrameResult OCRLatestFrameStore::get_latest(
    std::string previous_session,
    std::uint64_t after_sequence,
    std::chrono::milliseconds wait_timeout)
{
    wait_timeout = std::max(wait_timeout, std::chrono::milliseconds::zero());

    std::unique_lock lock(_mutex);
    const auto ready = [this, &previous_session, after_sequence]() {
        return _stopped || has_update_locked(previous_session, after_sequence);
    };

    if (!ready() && wait_timeout > std::chrono::milliseconds::zero()) {
        (void)_cv.wait_for(lock, wait_timeout, ready);
    }

    if (_stopped) {
        return current_result_locked(OCRLatestFrameStatus::Stopped);
    }
    if (has_update_locked(previous_session, after_sequence)) {
        return current_result_locked(OCRLatestFrameStatus::Frame);
    }
    return current_result_locked(OCRLatestFrameStatus::NotModified);
}

void OCRLatestFrameStore::stop()
{
    {
        std::lock_guard lock(_mutex);
        if (_stopped) {
            return;
        }
        _stopped = true;
    }
    _cv.notify_all();
}

bool OCRLatestFrameStore::is_stopped() const
{
    std::lock_guard lock(_mutex);
    return _stopped;
}

bool OCRLatestFrameStore::has_update_locked(const std::string& previous_session,
                                            std::uint64_t after_sequence) const
{
    if (!_latest) {
        return false;
    }
    return previous_session != _session_id || _sequence > after_sequence;
}

OCRLatestFrameResult OCRLatestFrameStore::current_result_locked(
    OCRLatestFrameStatus status) const
{
    return OCRLatestFrameResult {
        .status = status,
        .latest = status == OCRLatestFrameStatus::Frame ? _latest : nullptr,
        .session_id = _session_id,
        .sequence = _sequence,
    };
}
