#include <algorithm>
#include <atomic>
#include <chrono>
#include <ctime>
#include <exception>
#include <iomanip>
#include <limits>
#include <sstream>
#include <utility>
#include <vector>

#include "capture_coordinator.hpp"

#include <unistd.h>

struct CaptureWaiter {
    std::mutex mutex;
    std::condition_variable cv;
    CaptureRequest::Clock::time_point deadline;
    CaptureSource source {CaptureSource::Mcp};
    CaptureResponse response;
    bool completed {false};
    bool started {false};
    bool persistent {false};
};

struct CaptureJob {
    CaptureRequest request;
    std::vector<std::shared_ptr<CaptureWaiter>> waiters;
    bool high_priority {false};
    bool has_periodic_waiter {false};
};

namespace {

using Clock = CaptureRequest::Clock;

std::string make_session_id()
{
    static std::atomic<std::uint64_t> instance_sequence {0};
    const auto wall_count =
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::system_clock::now().time_since_epoch())
            .count();
    const auto steady_count =
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            Clock::now().time_since_epoch())
            .count();

    std::ostringstream stream;
    stream << std::hex << wall_count << '-' << steady_count << '-'
           << static_cast<unsigned long>(::getpid()) << '-'
           << instance_sequence.fetch_add(1, std::memory_order_relaxed);
    return stream.str();
}

std::string make_captured_at()
{
    const auto now = std::chrono::system_clock::now();
    const auto seconds =
        std::chrono::time_point_cast<std::chrono::seconds>(now);
    const auto nanoseconds =
        std::chrono::duration_cast<std::chrono::nanoseconds>(now - seconds)
            .count();
    const std::time_t time = std::chrono::system_clock::to_time_t(now);
    std::tm utc {};
    if (gmtime_r(&time, &utc) == nullptr) {
        return "1970-01-01T00:00:00.000000000Z";
    }

    char date[32] {};
    if (std::strftime(date, sizeof(date), "%Y-%m-%dT%H:%M:%S", &utc) == 0) {
        return "1970-01-01T00:00:00.000000000Z";
    }

    std::ostringstream stream;
    stream << date << '.' << std::setw(9) << std::setfill('0') << nanoseconds
           << 'Z';
    return stream.str();
}

int normalized_timeout_ms(int timeout_ms)
{
    const int normalized = timeout_ms > 0 ? timeout_ms : 500;
    return std::min(normalized, CaptureCoordinator::MAX_TIMEOUT_MS);
}

bool same_capture_profile(const CaptureRequest& lhs, const CaptureRequest& rhs)
{
    return lhs.x == rhs.x && lhs.y == rhs.y && lhs.w == rhs.w && lhs.h == rhs.h
        && lhs.quality == rhs.quality;
}

bool is_high_priority(CaptureSource source)
{ return source != CaptureSource::Periodic; }

bool complete_waiter(const std::shared_ptr<CaptureWaiter>& waiter,
                     CaptureResponse response)
{
    {
        std::lock_guard lock(waiter->mutex);
        if (waiter->completed) {
            return false;
        }
        waiter->response = std::move(response);
        waiter->completed = true;
    }
    waiter->cv.notify_all();
    return true;
}

bool waiter_completed(const std::shared_ptr<CaptureWaiter>& waiter)
{
    std::lock_guard lock(waiter->mutex);
    return waiter->completed;
}

bool has_active_high_priority_waiter(
    const std::vector<std::shared_ptr<CaptureWaiter>>& waiters)
{
    return std::any_of(waiters.begin(), waiters.end(), [](const auto& waiter) {
        std::lock_guard lock(waiter->mutex);
        return !waiter->completed && waiter->source != CaptureSource::Periodic;
    });
}

int remaining_timeout_ms(Clock::time_point deadline, Clock::time_point now)
{
    if (deadline <= now) {
        return 0;
    }

    const auto remaining = deadline - now;
    auto milliseconds =
        std::chrono::duration_cast<std::chrono::milliseconds>(remaining);
    if (milliseconds < remaining) {
        ++milliseconds;
    }

    const auto count = milliseconds.count();
    if (count > std::numeric_limits<int>::max()) {
        return std::numeric_limits<int>::max();
    }
    return std::max(1, static_cast<int>(count));
}

Clock::time_point latest_active_deadline(
    const std::vector<std::shared_ptr<CaptureWaiter>>& waiters)
{
    const auto now = Clock::now();
    Clock::time_point latest {};

    for (const auto& waiter : waiters) {
        if (waiter_completed(waiter)) {
            continue;
        }
        if (waiter->persistent) {
            latest = std::max(latest,
                              now
                                  + std::chrono::milliseconds(
                                      CaptureCoordinator::MAX_TIMEOUT_MS));
            continue;
        }
        if (waiter->deadline <= now) {
            (void)complete_waiter(
                waiter,
                CaptureResponse {.ret_code =
                                     CaptureCoordinator::RESULT_TIMEOUT});
            continue;
        }
        latest = std::max(latest, waiter->deadline);
    }

    return latest;
}

}  // namespace

CaptureTicket::CaptureTicket(std::shared_ptr<CaptureWaiter> waiter)
    : _waiter(std::move(waiter))
{}

CaptureTicket::~CaptureTicket() { cancel(); }

CaptureTicket::CaptureTicket(CaptureTicket&& other) noexcept
    : _waiter(std::move(other._waiter))
{}

CaptureTicket& CaptureTicket::operator=(CaptureTicket&& other) noexcept
{
    if (this != &other) {
        cancel();
        _waiter = std::move(other._waiter);
    }
    return *this;
}

CaptureResponse CaptureTicket::wait()
{
    if (!_waiter) {
        return CaptureResponse {
            .ret_code = CaptureCoordinator::RESULT_UNAVAILABLE,
        };
    }

    std::unique_lock lock(_waiter->mutex);
    if (!_waiter->completed && _waiter->persistent) {
        _waiter->cv.wait(lock, [this]() { return _waiter->completed; });
    } else if (!_waiter->completed
               && !_waiter->cv.wait_until(lock, _waiter->deadline, [this]() {
                      return _waiter->completed;
                  })) {
        _waiter->response = CaptureResponse {
            .ret_code = CaptureCoordinator::RESULT_TIMEOUT,
        };
        _waiter->completed = true;
    }
    return _waiter->response;
}

void CaptureTicket::cancel()
{
    if (!_waiter) {
        return;
    }
    (void)complete_waiter(
        _waiter,
        CaptureResponse {.ret_code = CaptureCoordinator::RESULT_CANCELLED});
}

bool CaptureTicket::valid() const { return _waiter != nullptr; }

CaptureCoordinator::CaptureCoordinator(CaptureOperation capture_operation,
                                       std::size_t queue_capacity,
                                       std::size_t max_high_priority_burst)
    : _capture_operation(std::move(capture_operation))
    , _queue_capacity(std::max<std::size_t>(1, queue_capacity))
    , _max_high_priority_burst(
          std::max<std::size_t>(1, max_high_priority_burst))
    , _session_id(make_session_id())
{}

CaptureCoordinator::~CaptureCoordinator() { stop(); }

void CaptureCoordinator::start()
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    std::lock_guard lock(_mutex);
    if (_accepting || _worker.joinable() || !_backend_healthy) {
        return;
    }

    _stop_requested = false;
    _accepting = true;
    _high_priority_streak = 0;
    _worker = std::thread(&CaptureCoordinator::run, this);
}

void CaptureCoordinator::stop()
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    std::vector<std::shared_ptr<CaptureJob>> cancelled_jobs;
    {
        std::lock_guard lock(_mutex);
        _accepting = false;
        _stop_requested = true;

        while (!_high_priority_jobs.empty()) {
            cancelled_jobs.push_back(std::move(_high_priority_jobs.front()));
            _high_priority_jobs.pop_front();
        }
        while (!_periodic_jobs.empty()) {
            cancelled_jobs.push_back(std::move(_periodic_jobs.front()));
            _periodic_jobs.pop_front();
        }
        _queued_requests = 0;
    }

    for (const auto& job : cancelled_jobs) {
        for (const auto& waiter : job->waiters) {
            (void)complete_waiter(
                waiter,
                CaptureResponse {
                    .ret_code = CaptureCoordinator::RESULT_UNAVAILABLE,
                });
        }
    }

    _cv.notify_all();
    if (_worker.joinable()) {
        _worker.join();
    }
}

CaptureTicket CaptureCoordinator::submit(CaptureRequest request)
{
    const auto now = Clock::now();
    const auto maximum_deadline =
        now + std::chrono::milliseconds(MAX_TIMEOUT_MS);
    const bool persistent = request.source == CaptureSource::Periodic
        && request.deadline == Clock::time_point {};
    if (persistent) {
        request.deadline = maximum_deadline;
    } else if (request.deadline == Clock::time_point {}) {
        request.deadline = now
            + std::chrono::milliseconds(
                               normalized_timeout_ms(request.timeout_ms));
    } else if (request.source != CaptureSource::Periodic
               && request.deadline > maximum_deadline) {
        request.deadline = maximum_deadline;
    }

    auto waiter = std::make_shared<CaptureWaiter>();
    waiter->deadline = request.deadline;
    waiter->source = request.source;
    waiter->persistent = persistent;
    CaptureTicket ticket(waiter);

    if (request.deadline <= now) {
        (void)complete_waiter(
            waiter,
            CaptureResponse {.ret_code = CaptureCoordinator::RESULT_TIMEOUT});
        return ticket;
    }

    {
        std::lock_guard lock(_mutex);
        prune_completed_waiters_locked();
        if (!_accepting) {
            (void)complete_waiter(
                waiter,
                CaptureResponse {
                    .ret_code = CaptureCoordinator::RESULT_UNAVAILABLE,
                });
            return ticket;
        }

        bool periodic_already_queued = false;
        for (const auto& queued_job : _high_priority_jobs) {
            periodic_already_queued |= queued_job->has_periodic_waiter;
        }
        for (const auto& queued_job : _periodic_jobs) {
            periodic_already_queued |= queued_job->has_periodic_waiter;
        }

        bool promote_to_high_priority = false;
        auto job =
            find_coalescable_job_locked(request, promote_to_high_priority);

        if (request.source == CaptureSource::Periodic && periodic_already_queued
            && !job) {
            (void)complete_waiter(
                waiter,
                CaptureResponse {.ret_code = CaptureCoordinator::RESULT_BUSY});
            return ticket;
        }

        const bool use_periodic_reserve =
            request.source == CaptureSource::Periodic
            && !periodic_already_queued;
        if (_queued_requests >= _queue_capacity && !use_periodic_reserve) {
            (void)complete_waiter(
                waiter,
                CaptureResponse {.ret_code = CaptureCoordinator::RESULT_BUSY});
            return ticket;
        }

        if (job) {
            job->waiters.push_back(waiter);
            job->has_periodic_waiter |=
                request.source == CaptureSource::Periodic;
            job->request.timeout_ms =
                std::max(normalized_timeout_ms(job->request.timeout_ms),
                         normalized_timeout_ms(request.timeout_ms));
            ++_queued_requests;

            if (promote_to_high_priority) {
                job->high_priority = true;
                job->request.source = request.source;
                _high_priority_jobs.push_back(std::move(job));
            }
        } else {
            job = std::make_shared<CaptureJob>();
            job->request = request;
            job->waiters.push_back(waiter);
            job->high_priority = is_high_priority(request.source);
            job->has_periodic_waiter =
                request.source == CaptureSource::Periodic;
            if (job->high_priority) {
                _high_priority_jobs.push_back(job);
            } else {
                _periodic_jobs.push_back(job);
            }
            ++_queued_requests;
        }
    }

    _cv.notify_one();
    return ticket;
}

CaptureResponse CaptureCoordinator::capture(CaptureRequest request)
{
    auto ticket = submit(std::move(request));
    return ticket.wait();
}

bool CaptureCoordinator::is_running() const
{
    std::lock_guard lock(_mutex);
    return _accepting;
}

std::size_t CaptureCoordinator::queued_jobs() const
{
    std::lock_guard lock(_mutex);
    const auto live_jobs = [](const auto& queue) {
        return static_cast<std::size_t>(
            std::count_if(queue.begin(), queue.end(), [](const auto& job) {
                return std::any_of(job->waiters.begin(),
                                   job->waiters.end(),
                                   [](const auto& waiter) {
                                       return !waiter_completed(waiter);
                                   });
            }));
    };
    return live_jobs(_high_priority_jobs) + live_jobs(_periodic_jobs);
}

std::size_t CaptureCoordinator::queued_requests() const
{
    std::lock_guard lock(_mutex);
    const auto live_requests = [](const auto& queue) {
        std::size_t count = 0;
        for (const auto& job : queue) {
            count += static_cast<std::size_t>(std::count_if(
                job->waiters.begin(),
                job->waiters.end(),
                [](const auto& waiter) { return !waiter_completed(waiter); }));
        }
        return count;
    };
    return live_requests(_high_priority_jobs) + live_requests(_periodic_jobs);
}

void CaptureCoordinator::run()
{
    while (true) {
        std::shared_ptr<CaptureJob> job;
        {
            std::unique_lock lock(_mutex);
            _cv.wait(lock, [this]() {
                return _stop_requested || !_high_priority_jobs.empty()
                    || !_periodic_jobs.empty();
            });

            if (_stop_requested && _high_priority_jobs.empty()
                && _periodic_jobs.empty()) {
                break;
            }

            job = take_next_job_locked();
            if (job) {
                _queued_requests -=
                    std::min(_queued_requests, job->waiters.size());
            }
        }

        if (job) {
            execute_job(job);
        }
    }
}

void CaptureCoordinator::execute_job(const std::shared_ptr<CaptureJob>& job)
{
    int ret_code = RESULT_UNAVAILABLE;
    std::shared_ptr<const AxeraUtils::SnapshotResult> frame;
    std::string capture_id;
    std::string captured_at;

    while (true) {
        const auto deadline = latest_active_deadline(job->waiters);
        if (deadline == Clock::time_point {}) {
            return;
        }

        {
            std::lock_guard lock(_mutex);
            if (_stop_requested) {
                ret_code = RESULT_UNAVAILABLE;
                break;
            }
        }

        const auto now = Clock::now();
        const int timeout_ms = remaining_timeout_ms(deadline, now);
        if (timeout_ms <= 0) {
            continue;
        }

        CaptureRequest request = job->request;
        request.deadline = deadline;
        request.timeout_ms = std::min(
            normalized_timeout_ms(job->request.timeout_ms), timeout_ms);

        AxeraUtils::SnapshotResult result;
        bool has_active_waiter = false;
        for (const auto& waiter : job->waiters) {
            std::lock_guard lock(waiter->mutex);
            if (!waiter->completed) {
                waiter->started = true;
                has_active_waiter = true;
            }
        }
        if (!has_active_waiter) {
            return;
        }

        try {
            ret_code = _capture_operation(request, result);
        } catch (const std::exception&) {
            ret_code = RESULT_BACKEND_UNHEALTHY;
        } catch (...) {
            ret_code = RESULT_BACKEND_UNHEALTHY;
        }

        if (ret_code == RESULT_BUSY) {
            const auto retry_deadline = latest_active_deadline(job->waiters);
            if (retry_deadline == Clock::time_point {}) {
                return;
            }
            const int retry_remaining =
                remaining_timeout_ms(retry_deadline, Clock::now());
            if (retry_remaining <= 0) {
                continue;
            }

            if (job->has_periodic_waiter
                && !has_active_high_priority_waiter(job->waiters)) {
                std::this_thread::sleep_for(
                    std::chrono::milliseconds(std::min(10, retry_remaining)));
                if (requeue_periodic_job(job)) {
                    return;
                }
                ret_code = RESULT_UNAVAILABLE;
                break;
            }

            std::this_thread::sleep_for(
                std::chrono::milliseconds(std::min(10, retry_remaining)));
            continue;
        }

        if (ret_code == RESULT_OK) {
            frame = std::make_shared<const AxeraUtils::SnapshotResult>(
                std::move(result));
            captured_at = make_captured_at();
            capture_id =
                _session_id + ":" + std::to_string(++_capture_sequence);
        }

        if (ret_code == RESULT_BACKEND_UNHEALTHY) {
            quarantine_backend();
        }
        break;
    }

    const auto completed_at = Clock::now();
    for (const auto& waiter : job->waiters) {
        if (waiter_completed(waiter)) {
            continue;
        }
        if (!waiter->persistent && waiter->deadline <= completed_at) {
            (void)complete_waiter(waiter,
                                  CaptureResponse {.ret_code = RESULT_TIMEOUT});
            continue;
        }
        (void)complete_waiter(waiter,
                              CaptureResponse {
                                  .ret_code = ret_code,
                                  .frame = frame,
                                  .session_id = _session_id,
                                  .capture_id = capture_id,
                                  .captured_at = captured_at,
                              });
    }
}

void CaptureCoordinator::quarantine_backend()
{
    std::vector<std::shared_ptr<CaptureJob>> cancelled_jobs;
    {
        std::lock_guard lock(_mutex);
        _accepting = false;
        _stop_requested = true;
        _backend_healthy = false;

        while (!_high_priority_jobs.empty()) {
            cancelled_jobs.push_back(std::move(_high_priority_jobs.front()));
            _high_priority_jobs.pop_front();
        }
        while (!_periodic_jobs.empty()) {
            cancelled_jobs.push_back(std::move(_periodic_jobs.front()));
            _periodic_jobs.pop_front();
        }
        _queued_requests = 0;
    }

    for (const auto& job : cancelled_jobs) {
        for (const auto& waiter : job->waiters) {
            (void)complete_waiter(
                waiter, CaptureResponse {.ret_code = RESULT_BACKEND_UNHEALTHY});
        }
    }
    _cv.notify_all();
}

void CaptureCoordinator::prune_completed_waiters_locked()
{
    const auto prune_queue = [this](auto& queue) {
        for (auto job_it = queue.begin(); job_it != queue.end();) {
            auto& waiters = (*job_it)->waiters;
            const auto previous_size = waiters.size();
            waiters.erase(std::remove_if(
                              waiters.begin(), waiters.end(), waiter_completed),
                          waiters.end());
            (*job_it)->has_periodic_waiter = std::any_of(
                waiters.begin(), waiters.end(), [](const auto& waiter) {
                    return waiter->source == CaptureSource::Periodic;
                });
            _queued_requests -=
                std::min(_queued_requests, previous_size - waiters.size());

            if (waiters.empty()) {
                job_it = queue.erase(job_it);
            } else {
                ++job_it;
            }
        }
    };

    prune_queue(_high_priority_jobs);
    prune_queue(_periodic_jobs);
}

bool CaptureCoordinator::requeue_periodic_job(
    const std::shared_ptr<CaptureJob>& job)
{
    std::size_t active_waiters = 0;
    for (const auto& waiter : job->waiters) {
        if (!waiter_completed(waiter)) {
            ++active_waiters;
        }
    }
    if (active_waiters == 0) {
        return false;
    }

    {
        std::lock_guard lock(_mutex);
        if (_stop_requested || !_backend_healthy) {
            return false;
        }
        job->high_priority = false;
        job->request.source = CaptureSource::Periodic;
        _periodic_jobs.push_back(job);
        _queued_requests += active_waiters;
    }
    _cv.notify_one();
    return true;
}

std::shared_ptr<CaptureJob> CaptureCoordinator::take_next_job_locked()
{
    if (!_periodic_jobs.empty()
        && (_high_priority_jobs.empty()
            || _high_priority_streak >= _max_high_priority_burst)) {
        auto job = std::move(_periodic_jobs.front());
        _periodic_jobs.pop_front();
        _high_priority_streak = 0;
        return job;
    }

    if (!_high_priority_jobs.empty()) {
        auto job = std::move(_high_priority_jobs.front());
        _high_priority_jobs.pop_front();
        _high_priority_streak =
            std::min(_max_high_priority_burst, _high_priority_streak + 1);
        return job;
    }

    if (!_periodic_jobs.empty()) {
        auto job = std::move(_periodic_jobs.front());
        _periodic_jobs.pop_front();
        _high_priority_streak = 0;
        return job;
    }

    return nullptr;
}

std::shared_ptr<CaptureJob> CaptureCoordinator::find_coalescable_job_locked(
    const CaptureRequest& request, bool& promote_to_high_priority)
{
    for (const auto& job : _high_priority_jobs) {
        if (same_capture_profile(job->request, request)) {
            return job;
        }
    }

    for (auto it = _periodic_jobs.begin(); it != _periodic_jobs.end(); ++it) {
        if (!same_capture_profile((*it)->request, request)) {
            continue;
        }

        auto job = *it;
        if (is_high_priority(request.source)) {
            _periodic_jobs.erase(it);
            promote_to_high_priority = true;
        }
        return job;
    }

    return nullptr;
}
