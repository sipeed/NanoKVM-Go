#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <limits>
#include <mutex>
#include <regex>
#include <thread>
#include <vector>

#include "capture_coordinator.hpp"

#include <catch2/catch_test_macros.hpp>

namespace {

using namespace std::chrono_literals;

CaptureRequest make_request(int id,
                            CaptureSource source = CaptureSource::Mcp,
                            int timeout_ms = 2000)
{
    CaptureRequest request;
    request.x = id;
    request.y = 0;
    request.w = 16;
    request.h = 16;
    request.quality = 85;
    request.timeout_ms = timeout_ms;
    request.source = source;
    return request;
}

class BlockingCaptureFake {
  public:
    explicit BlockingCaptureFake(bool block_first = true)
        : _block_first(block_first)
    {}

    int capture(const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result)
    {
        std::unique_lock lock(_mutex);
        const std::size_t call_number = ++_call_count;
        ++_active;
        _max_active = std::max(_max_active, _active);
        _order.push_back(request.x);
        _timeouts.push_back(request.timeout_ms);
        _cv.notify_all();

        if (_block_first && call_number == 1) {
            _cv.wait(lock, [this]() { return _first_released; });
        }

        --_active;
        _cv.notify_all();
        lock.unlock();

        result.width = 16;
        result.height = 16;
        result.data = {static_cast<uint8_t>(request.x & 0xff)};
        return 0;
    }

    bool wait_for_calls(std::size_t count)
    {
        std::unique_lock lock(_mutex);
        return _cv.wait_for(
            lock, 2s, [this, count]() { return _call_count >= count; });
    }

    void release_first()
    {
        {
            std::lock_guard lock(_mutex);
            _first_released = true;
        }
        _cv.notify_all();
    }

    std::size_t call_count() const
    {
        std::lock_guard lock(_mutex);
        return _call_count;
    }

    std::size_t max_active() const
    {
        std::lock_guard lock(_mutex);
        return _max_active;
    }

    std::vector<int> order() const
    {
        std::lock_guard lock(_mutex);
        return _order;
    }

    std::vector<int> timeouts() const
    {
        std::lock_guard lock(_mutex);
        return _timeouts;
    }

  private:
    const bool _block_first;
    mutable std::mutex _mutex;
    std::condition_variable _cv;
    std::size_t _call_count {0};
    std::size_t _active {0};
    std::size_t _max_active {0};
    std::vector<int> _order;
    std::vector<int> _timeouts;
    bool _first_released {false};
};

class FirstCaptureReleaseGuard {
  public:
    explicit FirstCaptureReleaseGuard(BlockingCaptureFake& fake)
        : _fake(fake)
    {}

    ~FirstCaptureReleaseGuard() { _fake.release_first(); }

  private:
    BlockingCaptureFake& _fake;
};

}  // namespace

TEST_CASE("CaptureRequest defaults to MCP source", "[capture][coordinator]")
{
    const CaptureRequest request;
    REQUIRE(request.source == CaptureSource::Mcp);
}

TEST_CASE("CaptureCoordinator serializes capture operations",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto first = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto second = coordinator.submit(make_request(2));
    auto third = coordinator.submit(make_request(3));

    REQUIRE(coordinator.queued_requests() == 2);
    fake.release_first();

    REQUIRE(first.wait().ok());
    REQUIRE(second.wait().ok());
    REQUIRE(third.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 3);
    REQUIRE(fake.max_active() == 1);
}

TEST_CASE("CaptureCoordinator bounds the waiting queue",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator(
        [&fake](const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result) {
            return fake.capture(request, result);
        },
        2);
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto queued_one = coordinator.submit(make_request(2));
    auto queued_two = coordinator.submit(make_request(3));
    auto rejected = coordinator.submit(make_request(4));

    REQUIRE(coordinator.queued_requests() == 2);
    REQUIRE(rejected.wait().ret_code == CaptureCoordinator::RESULT_BUSY);

    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(queued_one.wait().ok());
    REQUIRE(queued_two.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 3);
}

TEST_CASE("CaptureCoordinator reserves one periodic queue slot",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator(
        [&fake](const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result) {
            return fake.capture(request, result);
        },
        2);
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto mcp_one = coordinator.submit(make_request(2));
    auto mcp_two = coordinator.submit(make_request(3));
    auto periodic =
        coordinator.submit(make_request(4, CaptureSource::Periodic, 20));
    auto overflow = coordinator.submit(make_request(5));

    REQUIRE(coordinator.queued_requests() == 3);
    REQUIRE(overflow.wait().ret_code == CaptureCoordinator::RESULT_BUSY);

    std::this_thread::sleep_for(50ms);
    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(mcp_one.wait().ok());
    REQUIRE(mcp_two.wait().ok());
    REQUIRE(periodic.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 4);
}

TEST_CASE("CaptureCoordinator yields after periodic backend busy",
          "[capture][coordinator]")
{
    std::mutex mutex;
    std::condition_variable cv;
    bool periodic_entered = false;
    bool release_periodic = false;
    int periodic_calls = 0;
    std::vector<int> order;

    CaptureCoordinator coordinator(
        [&](const CaptureRequest& request, AxeraUtils::SnapshotResult& result) {
            bool return_busy = false;
            {
                std::unique_lock lock(mutex);
                order.push_back(request.x);
                if (request.source == CaptureSource::Periodic
                    && ++periodic_calls == 1) {
                    periodic_entered = true;
                    cv.notify_all();
                    cv.wait(lock,
                            [&release_periodic]() { return release_periodic; });
                    return_busy = true;
                }
            }

            if (return_busy) {
                return CaptureCoordinator::RESULT_BUSY;
            }
            result.width = 16;
            result.height = 16;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    auto periodic =
        coordinator.submit(make_request(1, CaptureSource::Periodic, 20));
    bool entered_ok = false;
    {
        std::unique_lock lock(mutex);
        entered_ok = cv.wait_for(
            lock, 2s, [&periodic_entered]() { return periodic_entered; });
    }
    if (!entered_ok) {
        {
            std::lock_guard lock(mutex);
            release_periodic = true;
        }
        cv.notify_all();
        coordinator.stop();
    }
    REQUIRE(entered_ok);

    auto mcp = coordinator.submit(make_request(2));
    {
        std::lock_guard lock(mutex);
        release_periodic = true;
    }
    cv.notify_all();

    REQUIRE(mcp.wait().ok());
    REQUIRE(periodic.wait().ok());
    coordinator.stop();

    REQUIRE(order == std::vector<int> {1, 2, 1});
}

TEST_CASE("CaptureCoordinator skips expired queued requests",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto expired = coordinator.submit(make_request(2, CaptureSource::Mcp, 30));
    auto following = coordinator.submit(make_request(3));

    REQUIRE(expired.wait().ret_code == CaptureCoordinator::RESULT_TIMEOUT);
    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(following.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 2);
    REQUIRE(fake.order() == std::vector<int> {1, 3});
}

TEST_CASE("CaptureCoordinator deducts queue time from backend timeout",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto queued = coordinator.submit(make_request(2, CaptureSource::Mcp, 1000));

    std::this_thread::sleep_for(100ms);
    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(queued.wait().ok());
    coordinator.stop();

    const auto timeouts = fake.timeouts();
    REQUIRE(timeouts.size() == 2);
    REQUIRE(timeouts[1] > 0);
    REQUIRE(timeouts[1] < 1000);
}

TEST_CASE("CaptureCoordinator rejects already expired requests",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake(false);
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();

    auto request = make_request(1);
    request.deadline = CaptureRequest::Clock::now() - 1ms;
    auto ticket = coordinator.submit(request);

    REQUIRE(ticket.wait().ret_code == CaptureCoordinator::RESULT_TIMEOUT);
    coordinator.stop();
    REQUIRE(fake.call_count() == 0);
}

TEST_CASE("CaptureCoordinator cancels queued requests",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator(
        [&fake](const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result) {
            return fake.capture(request, result);
        },
        1);
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto cancelled = coordinator.submit(make_request(2));
    cancelled.cancel();
    auto following = coordinator.submit(make_request(3));

    REQUIRE(cancelled.wait().ret_code == CaptureCoordinator::RESULT_CANCELLED);
    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(following.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 2);
    REQUIRE(fake.order() == std::vector<int> {1, 3});
}

TEST_CASE("CaptureCoordinator coalesces matching queued requests",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto first_waiter = coordinator.submit(make_request(2));
    auto second_waiter = coordinator.submit(make_request(2));

    REQUIRE(coordinator.queued_jobs() == 1);
    REQUIRE(coordinator.queued_requests() == 2);

    fake.release_first();
    REQUIRE(active.wait().ok());
    const auto first_response = first_waiter.wait();
    const auto second_response = second_waiter.wait();
    coordinator.stop();

    REQUIRE(first_response.ok());
    REQUIRE(second_response.ok());
    REQUIRE(first_response.frame == second_response.frame);
    REQUIRE_FALSE(first_response.session_id.empty());
    REQUIRE(first_response.session_id == second_response.session_id);
    REQUIRE_FALSE(first_response.capture_id.empty());
    REQUIRE(first_response.capture_id == second_response.capture_id);
    REQUIRE(std::regex_match(
        first_response.captured_at,
        std::regex(
            R"(^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{9}Z$)")));
    REQUIRE(first_response.captured_at == second_response.captured_at);
    REQUIRE(fake.call_count() == 2);
}

TEST_CASE("CaptureCoordinator never coalesces with an active capture",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto fresh = coordinator.submit(make_request(1));

    REQUIRE(coordinator.queued_jobs() == 1);
    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(fresh.wait().ok());
    coordinator.stop();

    REQUIRE(fake.call_count() == 2);
}

TEST_CASE("CaptureCoordinator schedules periodic work after an MCP burst",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator(
        [&fake](const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result) {
            return fake.capture(request, result);
        },
        8,
        2);
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(100));
    REQUIRE(fake.wait_for_calls(1));
    auto periodic =
        coordinator.submit(make_request(1, CaptureSource::Periodic));
    auto mcp_one = coordinator.submit(make_request(2));
    auto mcp_two = coordinator.submit(make_request(3));
    auto mcp_three = coordinator.submit(make_request(4));

    fake.release_first();
    REQUIRE(active.wait().ok());
    REQUIRE(periodic.wait().ok());
    REQUIRE(mcp_one.wait().ok());
    REQUIRE(mcp_two.wait().ok());
    REQUIRE(mcp_three.wait().ok());
    coordinator.stop();

    REQUIRE(fake.order() == std::vector<int> {100, 2, 1, 3, 4});
}

TEST_CASE("CaptureCoordinator retries transient backend busy",
          "[capture][coordinator]")
{
    std::atomic<int> calls {0};
    CaptureCoordinator coordinator(
        [&calls](const CaptureRequest&, AxeraUtils::SnapshotResult& result) {
            const int call = ++calls;
            if (call < 3) {
                return CaptureCoordinator::RESULT_BUSY;
            }
            result.width = 16;
            result.height = 16;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    const auto response = coordinator.capture(make_request(1));
    coordinator.stop();

    REQUIRE(response.ok());
    REQUIRE(calls.load() == 3);
}

TEST_CASE("CaptureCoordinator cancellation stops backend busy retries",
          "[capture][coordinator]")
{
    std::atomic<int> calls {0};
    std::mutex mutex;
    std::condition_variable cv;
    bool entered = false;
    bool released = false;
    CaptureCoordinator coordinator(
        [&](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++calls;
            std::unique_lock lock(mutex);
            entered = true;
            cv.notify_all();
            cv.wait(lock, [&released]() { return released; });
            return CaptureCoordinator::RESULT_BUSY;
        });
    coordinator.start();

    auto ticket = coordinator.submit(make_request(1));
    bool entered_ok = false;
    {
        std::unique_lock lock(mutex);
        entered_ok = cv.wait_for(lock, 2s, [&entered]() { return entered; });
    }
    if (!entered_ok) {
        {
            std::lock_guard lock(mutex);
            released = true;
        }
        cv.notify_all();
        coordinator.stop();
    }
    REQUIRE(entered_ok);

    ticket.cancel();
    const auto response = ticket.wait();
    {
        std::lock_guard lock(mutex);
        released = true;
    }
    cv.notify_all();
    coordinator.stop();

    REQUIRE(response.ret_code == CaptureCoordinator::RESULT_CANCELLED);
    REQUIRE(calls.load() == 1);
}

TEST_CASE("CaptureCoordinator clamps excessive request timeouts",
          "[capture][coordinator]")
{
    std::atomic<int> received_timeout {0};
    CaptureCoordinator coordinator(
        [&received_timeout](const CaptureRequest& request,
                            AxeraUtils::SnapshotResult& result) {
            received_timeout = request.timeout_ms;
            result.width = 16;
            result.height = 16;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    auto request = make_request(1);
    request.timeout_ms = std::numeric_limits<int>::max();
    const auto response = coordinator.capture(request);
    coordinator.stop();

    REQUIRE(response.ok());
    REQUIRE(received_timeout.load() > 0);
    REQUIRE(received_timeout.load() <= CaptureCoordinator::MAX_TIMEOUT_MS);
}

TEST_CASE("CaptureCoordinator continues after a backend failure",
          "[capture][coordinator]")
{
    std::atomic<int> calls {0};
    CaptureCoordinator coordinator(
        [&calls](const CaptureRequest&, AxeraUtils::SnapshotResult& result) {
            if (++calls == 1) {
                return -14;
            }
            result.width = 16;
            result.height = 16;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    REQUIRE(coordinator.capture(make_request(1)).ret_code == -14);
    REQUIRE(coordinator.capture(make_request(2)).ok());
    coordinator.stop();

    REQUIRE(calls.load() == 2);
}

TEST_CASE("CaptureCoordinator quarantines an unhealthy backend",
          "[capture][coordinator]")
{
    std::atomic<int> calls {0};
    std::mutex mutex;
    std::condition_variable cv;
    bool entered = false;
    bool released = false;
    CaptureCoordinator coordinator(
        [&](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++calls;
            std::unique_lock lock(mutex);
            entered = true;
            cv.notify_all();
            cv.wait(lock, [&released]() { return released; });
            return CaptureCoordinator::RESULT_BACKEND_UNHEALTHY;
        });
    coordinator.start();

    auto first = coordinator.submit(make_request(1));
    bool entered_ok = false;
    {
        std::unique_lock lock(mutex);
        entered_ok = cv.wait_for(lock, 2s, [&entered]() { return entered; });
    }
    if (!entered_ok) {
        {
            std::lock_guard lock(mutex);
            released = true;
        }
        cv.notify_all();
        coordinator.stop();
    }
    REQUIRE(entered_ok);
    auto queued = coordinator.submit(make_request(2));
    {
        std::lock_guard lock(mutex);
        released = true;
    }
    cv.notify_all();

    REQUIRE(first.wait().ret_code
            == CaptureCoordinator::RESULT_BACKEND_UNHEALTHY);
    REQUIRE(queued.wait().ret_code
            == CaptureCoordinator::RESULT_BACKEND_UNHEALTHY);
    REQUIRE_FALSE(coordinator.is_running());
    REQUIRE(coordinator.capture(make_request(3)).ret_code
            == CaptureCoordinator::RESULT_UNAVAILABLE);
    coordinator.stop();

    REQUIRE(calls.load() == 1);
}

TEST_CASE("CaptureCoordinator quarantines backend exceptions",
          "[capture][coordinator]")
{
    CaptureCoordinator coordinator(
        [](const CaptureRequest&, AxeraUtils::SnapshotResult&) -> int {
            throw 1;
        });
    coordinator.start();

    REQUIRE(coordinator.capture(make_request(1)).ret_code
            == CaptureCoordinator::RESULT_BACKEND_UNHEALTHY);
    REQUIRE_FALSE(coordinator.is_running());
    coordinator.stop();
}

TEST_CASE("CaptureCoordinator stop cancels queued work and is idempotent",
          "[capture][coordinator]")
{
    BlockingCaptureFake fake;
    CaptureCoordinator coordinator([&fake](const CaptureRequest& request,
                                           AxeraUtils::SnapshotResult& result) {
        return fake.capture(request, result);
    });
    coordinator.start();
    FirstCaptureReleaseGuard release_guard(fake);

    auto active = coordinator.submit(make_request(1));
    REQUIRE(fake.wait_for_calls(1));
    auto queued = coordinator.submit(make_request(2));

    std::jthread stopper([&coordinator]() { coordinator.stop(); });
    const auto queued_response = queued.wait();
    fake.release_first();
    const auto active_response = active.wait();
    stopper.join();
    coordinator.stop();

    REQUIRE(queued_response.ret_code == CaptureCoordinator::RESULT_UNAVAILABLE);
    REQUIRE(active_response.ok());
    REQUIRE(fake.call_count() == 1);
    REQUIRE(coordinator.capture(make_request(3)).ret_code
            == CaptureCoordinator::RESULT_UNAVAILABLE);
}
