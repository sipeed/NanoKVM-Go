#pragma once

#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <memory>
#include <mutex>
#include <string>

#include "capture_coordinator.hpp"

struct OCRLatestFrame {
    std::shared_ptr<const AxeraUtils::SnapshotResult> frame;
    std::string session_id;
    std::string capture_id;
    std::string captured_at;
    std::uint64_t sequence {0};
};

enum class OCRLatestFrameStatus {
    Frame,
    NotModified,
    Stopped,
};

struct OCRLatestFrameResult {
    OCRLatestFrameStatus status {OCRLatestFrameStatus::NotModified};
    std::shared_ptr<const OCRLatestFrame> latest;
    std::string session_id;
    std::uint64_t sequence {0};
};

class OCRLatestFrameStore {
  public:
    OCRLatestFrameStore() = default;
    ~OCRLatestFrameStore();

    OCRLatestFrameStore(const OCRLatestFrameStore&) = delete;
    OCRLatestFrameStore& operator=(const OCRLatestFrameStore&) = delete;

    // Publishing is explicit so unrelated captures, such as MCP
    // capture_fresh requests, cannot advance the OCR sequence.
    [[nodiscard]] bool publish(const CaptureResponse& response);

    [[nodiscard]] OCRLatestFrameResult get_latest(
        std::string previous_session,
        std::uint64_t after_sequence,
        std::chrono::milliseconds wait_timeout);

    void stop();
    [[nodiscard]] bool is_stopped() const;

  private:
    [[nodiscard]] bool has_update_locked(const std::string& previous_session,
                                         std::uint64_t after_sequence) const;
    [[nodiscard]] OCRLatestFrameResult current_result_locked(
        OCRLatestFrameStatus status) const;

    mutable std::mutex _mutex;
    std::condition_variable _cv;
    std::shared_ptr<const OCRLatestFrame> _latest;
    std::string _session_id;
    std::uint64_t _sequence {0};
    bool _stopped {false};
};
