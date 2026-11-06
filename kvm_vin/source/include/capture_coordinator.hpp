#pragma once

#include <chrono>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <thread>

#include "ax_utils.hpp"

enum class CaptureSource {
    Periodic,
    Mcp,
    SnapshotHttp,
};

struct CaptureRequest {
    using Clock = std::chrono::steady_clock;

    int x {0};
    int y {0};
    int w {0};
    int h {0};
    int quality {85};
    int timeout_ms {500};
    Clock::time_point deadline {};
    CaptureSource source {CaptureSource::Mcp};
};

struct CaptureResponse {
    int ret_code {-17};
    std::shared_ptr<const AxeraUtils::SnapshotResult> frame;
    std::string session_id;
    std::string capture_id;
    std::string captured_at;

    [[nodiscard]] bool ok() const { return ret_code == 0 && frame != nullptr; }
};

struct CaptureWaiter;
struct CaptureJob;
class CaptureTicket;

class CaptureCoordinator {
  public:
    static constexpr int RESULT_OK = 0;
    static constexpr int RESULT_TIMEOUT = -13;
    static constexpr int RESULT_BUSY = -16;
    static constexpr int RESULT_UNAVAILABLE = -17;
    static constexpr int RESULT_CANCELLED = -18;
    static constexpr int RESULT_BACKEND_UNHEALTHY = -19;
    static constexpr int MAX_TIMEOUT_MS = 30000;

    using CaptureOperation =
        std::function<int(const CaptureRequest&, AxeraUtils::SnapshotResult&)>;

    explicit CaptureCoordinator(CaptureOperation capture_operation,
                                std::size_t queue_capacity = 8,
                                std::size_t max_high_priority_burst = 3);
    ~CaptureCoordinator();

    CaptureCoordinator(const CaptureCoordinator&) = delete;
    CaptureCoordinator& operator=(const CaptureCoordinator&) = delete;

    void start();
    void stop();

    [[nodiscard]] CaptureTicket submit(CaptureRequest request);
    [[nodiscard]] CaptureResponse capture(CaptureRequest request);

    [[nodiscard]] bool is_running() const;
    [[nodiscard]] std::size_t queued_jobs() const;
    [[nodiscard]] std::size_t queued_requests() const;

  private:
    void run();
    void execute_job(const std::shared_ptr<CaptureJob>& job);
    void quarantine_backend();
    void prune_completed_waiters_locked();
    bool requeue_periodic_job(const std::shared_ptr<CaptureJob>& job);
    std::shared_ptr<CaptureJob> take_next_job_locked();
    std::shared_ptr<CaptureJob> find_coalescable_job_locked(
        const CaptureRequest& request, bool& promote_to_high_priority);

    CaptureOperation _capture_operation;
    const std::size_t _queue_capacity;
    const std::size_t _max_high_priority_burst;

    mutable std::mutex _lifecycle_mutex;
    mutable std::mutex _mutex;
    std::condition_variable _cv;
    std::deque<std::shared_ptr<CaptureJob>> _high_priority_jobs;
    std::deque<std::shared_ptr<CaptureJob>> _periodic_jobs;
    std::thread _worker;
    std::size_t _queued_requests {0};
    std::size_t _high_priority_streak {0};
    bool _accepting {false};
    bool _stop_requested {false};
    bool _backend_healthy {true};
    std::string _session_id;
    std::uint64_t _capture_sequence {0};
};

class CaptureTicket {
  public:
    CaptureTicket() = default;
    ~CaptureTicket();

    CaptureTicket(const CaptureTicket&) = delete;
    CaptureTicket& operator=(const CaptureTicket&) = delete;
    CaptureTicket(CaptureTicket&& other) noexcept;
    CaptureTicket& operator=(CaptureTicket&& other) noexcept;

    [[nodiscard]] CaptureResponse wait();
    void cancel();
    [[nodiscard]] bool valid() const;

  private:
    friend class CaptureCoordinator;

    explicit CaptureTicket(std::shared_ptr<CaptureWaiter> waiter);

    std::shared_ptr<CaptureWaiter> _waiter;
};
