#include "lvgl_main/launcher_child_control.hpp"

#include <cerrno>
#include <chrono>
#include <thread>

#include <signal.h>
#include <sys/wait.h>
#include <unistd.h>

namespace lvgl_main::launcher_child_control
{

int SystemProcessOps::send_signal(pid_t child, int signal) noexcept
{
	if (::kill(child, signal) == 0) {
		return 0;
	}
	return errno;
}

pid_t SystemProcessOps::wait_for_child(pid_t child, int *status, int options,
				       int &error) noexcept
{
	const pid_t result = ::waitpid(child, status, options);
	error = result < 0 ? errno : 0;
	return result;
}

std::chrono::steady_clock::time_point SystemProcessOps::now() noexcept
{
	return std::chrono::steady_clock::now();
}

void SystemProcessOps::sleep_for(std::chrono::milliseconds duration) noexcept
{
	std::this_thread::sleep_for(duration);
}

StopResult stop_and_wait(ProcessOps &ops, pid_t child,
			 std::chrono::milliseconds timeout) noexcept
{
	if (child <= 0) {
		return { .status = StopStatus::Failed, .error = ECHILD };
	}

	const int stop_error = ops.send_signal(child, SIGSTOP);
	if (stop_error != 0) {
		return {
			.status = stop_error == ESRCH ? StopStatus::Exited :
							StopStatus::Failed,
			.error = stop_error,
		};
	}

	const auto deadline = ops.now() + timeout;
	for (;;) {
		int status = 0;
		int wait_error = 0;
		const pid_t wait_result = ops.wait_for_child(
			child, &status, WNOHANG | WUNTRACED, wait_error);
		if (wait_result == child) {
			if (WIFSTOPPED(status)) {
				return { .status = StopStatus::Stopped,
					 .stop_signal_sent = true };
			}
			if (WIFEXITED(status) || WIFSIGNALED(status)) {
				return { .status = StopStatus::Exited,
					 .stop_signal_sent = true };
			}
		} else if (wait_result < 0) {
			if (wait_error == EINTR) {
				continue;
			}
			return {
				.status = wait_error == ECHILD ?
						  StopStatus::Exited :
						  StopStatus::Failed,
				.error = wait_error,
				.stop_signal_sent = true,
			};
		}

		if (ops.now() >= deadline) {
			return { .status = StopStatus::TimedOut,
				 .error = ETIMEDOUT,
				 .stop_signal_sent = true };
		}
		ops.sleep_for(std::chrono::milliseconds { 5 });
	}
}

ResumeGuard::ResumeGuard(ProcessOps &ops, pid_t child) noexcept
	: _ops(ops)
	, _child(child)
{
}

ResumeGuard::~ResumeGuard()
{
	(void)resume();
}

int ResumeGuard::resume() noexcept
{
	if (_child <= 0) {
		return 0;
	}
	const int error = _ops.send_signal(_child, SIGCONT);
	if (error == 0 || error == ESRCH) {
		_child = -1;
	}
	return error;
}

void ResumeGuard::dismiss() noexcept
{
	_child = -1;
}

bool ResumeGuard::armed() const noexcept
{
	return _child > 0;
}

} // namespace lvgl_main::launcher_child_control
