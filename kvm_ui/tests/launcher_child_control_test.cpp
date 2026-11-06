#include <cerrno>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <vector>

#include <signal.h>
#include <sys/wait.h>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/launcher_child_control.hpp"

using namespace lvgl_main::launcher_child_control;

namespace
{

struct WaitStep {
	pid_t result = 0;
	int status = 0;
	int error = 0;
};

class FakeProcessOps final : public ProcessOps {
    public:
	int send_signal(pid_t child, int signal) noexcept override
	{
		signals.push_back(signal);
		children.push_back(child);
		if (signal_errors.empty()) {
			return 0;
		}
		const int error = signal_errors.front();
		signal_errors.pop_front();
		return error;
	}

	pid_t wait_for_child(pid_t child, int *status, int options,
			     int &error) noexcept override
	{
		waited_children.push_back(child);
		wait_options.push_back(options);
		if (wait_steps.empty()) {
			error = 0;
			return 0;
		}
		const WaitStep step = wait_steps.front();
		wait_steps.pop_front();
		*status = step.status;
		error = step.error;
		return step.result;
	}

	std::chrono::steady_clock::time_point now() noexcept override
	{
		return std::chrono::steady_clock::time_point {
			std::chrono::milliseconds { elapsed_ms }
		};
	}

	void sleep_for(std::chrono::milliseconds duration) noexcept override
	{
		elapsed_ms += duration.count();
		++sleep_calls;
	}

	std::deque<int> signal_errors;
	std::deque<WaitStep> wait_steps;
	std::vector<int> signals;
	std::vector<pid_t> children;
	std::vector<pid_t> waited_children;
	std::vector<int> wait_options;
	int64_t elapsed_ms = 0;
	std::size_t sleep_calls = 0;
};

int stopped_status(int signal)
{
	return (signal << 8) | 0x7f;
}

} // namespace

TEST_CASE("Launcher child pause waits for a confirmed stopped state")
{
	FakeProcessOps ops;
	ops.wait_steps.push_back({ .result = 0 });
	ops.wait_steps.push_back(
		{ .result = 42, .status = stopped_status(SIGSTOP) });

	const StopResult result =
		stop_and_wait(ops, 42, std::chrono::milliseconds { 200 });

	CHECK(result.status == StopStatus::Stopped);
	CHECK(result.stop_signal_sent);
	REQUIRE(ops.signals.size() == 1);
	CHECK(ops.signals.front() == SIGSTOP);
	CHECK(ops.sleep_calls == 1);
	REQUIRE(ops.wait_options.size() == 2);
	CHECK((ops.wait_options.front() & WUNTRACED) != 0);
}

TEST_CASE("Launcher child resume guard retries after a failed SIGCONT")
{
	FakeProcessOps ops;
	ops.signal_errors.push_back(EPERM);
	ops.signal_errors.push_back(0);

	{
		ResumeGuard guard(ops, 42);
		CHECK(guard.resume() == EPERM);
		CHECK(guard.armed());
	}

	REQUIRE(ops.signals.size() == 2);
	CHECK(ops.signals[0] == SIGCONT);
	CHECK(ops.signals[1] == SIGCONT);
}
