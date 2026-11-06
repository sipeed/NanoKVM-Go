#pragma once

#include <chrono>
#include <cstdint>

#include <sys/types.h>

namespace lvgl_main::launcher_child_control
{

enum class StopStatus : uint8_t {
	Stopped,
	Exited,
	Failed,
	TimedOut,
};

struct StopResult {
	StopStatus status = StopStatus::Failed;
	int error = 0;
	bool stop_signal_sent = false;
};

class ProcessOps {
    public:
	virtual ~ProcessOps() = default;

	virtual int send_signal(pid_t child, int signal) noexcept = 0;
	virtual pid_t wait_for_child(pid_t child, int *status, int options,
				     int &error) noexcept = 0;
	virtual std::chrono::steady_clock::time_point now() noexcept = 0;
	virtual void sleep_for(std::chrono::milliseconds duration) noexcept = 0;
};

class SystemProcessOps final : public ProcessOps {
    public:
	int send_signal(pid_t child, int signal) noexcept override;
	pid_t wait_for_child(pid_t child, int *status, int options,
			     int &error) noexcept override;
	std::chrono::steady_clock::time_point now() noexcept override;
	void sleep_for(std::chrono::milliseconds duration) noexcept override;
};

StopResult stop_and_wait(ProcessOps &ops, pid_t child,
			 std::chrono::milliseconds timeout) noexcept;

class ResumeGuard final {
    public:
	ResumeGuard(ProcessOps &ops, pid_t child) noexcept;
	~ResumeGuard();

	ResumeGuard(const ResumeGuard &) = delete;
	ResumeGuard &operator=(const ResumeGuard &) = delete;

	int resume() noexcept;
	void dismiss() noexcept;
	bool armed() const noexcept;

    private:
	ProcessOps &_ops;
	pid_t _child;
};

} // namespace lvgl_main::launcher_child_control
