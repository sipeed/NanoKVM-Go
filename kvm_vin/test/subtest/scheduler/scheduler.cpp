#include <chrono>

#include <catch2/catch_all.hpp>
#include <catch2/catch_test_macros.hpp>
#include <tools/scheduler.hpp>
#include <tools/state_manager.hpp>
#include <tools/system_uptime.hpp>

using namespace tools;
using namespace std::chrono_literals;

// TEST_CASE("Scheduler executes a single job at the given interval",
//           "[scheduler]")
// {
//     IntervalScheduler scheduler;
//     std::atomic<int> count {0};
//     scheduler.submit([&] { ++count; }, 10ms);
//
//     std::thread runner([&] { scheduler.run(); });
//     std::this_thread::sleep_for(35ms);
//     auto _ = StateManager::get_instance().set_state("need_exit", "true");
//     runner.join();
//
//     REQUIRE(count >= 3);
//     REQUIRE(count <= 4);
// }

// TEST_CASE("Scheduler handles multiple jobs with different intervals",
//           "[scheduler]")
// {
//     IntervalScheduler scheduler;
//     std::atomic<int> fast {0}, slow {0};
//     scheduler.submit([&] { ++fast; }, 5ms);
//     scheduler.submit([&] { ++slow; }, 20ms);
//
//     std::thread runner([&] { scheduler.run(); });
//     std::this_thread::sleep_for(50ms);
//     auto _ = StateManager::get_instance().set_state("need_exit", "true");
//     runner.join();
//
//     REQUIRE(fast >= 9);
//     REQUIRE(slow >= 2);
// }

TEST_CASE("Scheduler handles multiple jobs with different intervals",
          "[scheduler]")
{
    using TimeType = std::chrono::milliseconds;
    IntervalScheduler scheduler;
    const auto target1 = 50ms;
    auto prev1 = SystemUptime::now2<TimeType>();
    scheduler.submit(
        [&prev1, target1]() {
            auto used = SystemUptime::elapsed(prev1);

            REQUIRE(used >= target1 * 0.9);
            REQUIRE(used <= target1 * 1.1);

            prev1 = SystemUptime::now2<TimeType>();
        },
        target1);

    const auto target2 = 120ms;
    auto prev2 = SystemUptime::now2<TimeType>();
    scheduler.submit(
        [&prev2, target2]() {
            auto used = SystemUptime::elapsed(prev2);

            REQUIRE(used >= target2 * 0.9);
            REQUIRE(used <= target2 * 1.1);

            prev2 = SystemUptime::now2<TimeType>();
        },
        target2);

    std::thread runner([&] { scheduler.run(); });
    std::this_thread::sleep_for(2s);
    auto _ = StateManager::get_instance().set_state("need_exit", "true");
    runner.join();
    auto __ = StateManager::get_instance().set_state("need_exit", "false");
}
