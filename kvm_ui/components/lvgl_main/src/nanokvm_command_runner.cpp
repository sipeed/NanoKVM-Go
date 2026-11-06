#include "nanokvm_runtime_internal.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstring>
#include <filesystem>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include <fcntl.h>
#include <poll.h>
#include <signal.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>

#include <spdlog/spdlog.h>

namespace lvgl_main
{
namespace detail
{

namespace
{

constexpr std::chrono::milliseconds kCommandPollInterval { 50 };
constexpr std::chrono::milliseconds kCommandTerminateGrace { 200 };
constexpr std::chrono::milliseconds kCommandKillGrace { 800 };

int64_t elapsed_ms(std::chrono::steady_clock::time_point start,
		   std::chrono::steady_clock::time_point end)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(end -
							     start)
		.count();
}

std::optional<CommandProcess>
spawn_command(const std::vector<std::string> &args)
{
	if (args.empty()) {
		return std::nullopt;
	}

	std::vector<std::string> argv_storage = args;
	std::vector<char *> argv;
	argv.reserve(argv_storage.size() + 1U);
	for (auto &arg : argv_storage) {
		argv.push_back(arg.data());
	}
	argv.push_back(nullptr);

	int pipe_fds[2] { -1, -1 };
	if (::pipe(pipe_fds) != 0) {
		return std::nullopt;
	}
	ScopedFd read_fd(pipe_fds[0]);
	ScopedFd write_fd(pipe_fds[1]);
	if (::fcntl(read_fd.get(), F_SETFD, FD_CLOEXEC) != 0 ||
	    ::fcntl(write_fd.get(), F_SETFD, FD_CLOEXEC) != 0) {
		return std::nullopt;
	}
	const int read_flags = ::fcntl(read_fd.get(), F_GETFL, 0);
	if (read_flags < 0 ||
	    ::fcntl(read_fd.get(), F_SETFL, read_flags | O_NONBLOCK) != 0) {
		return std::nullopt;
	}

	const pid_t pid = ::fork();
	if (pid < 0) {
		return std::nullopt;
	}
	if (pid == 0) {
		// Keep timeout termination scoped to this command tree.
		if (::setpgid(0, 0) != 0) {
			::_exit(127);
		}
		read_fd.reset();
		if (::dup2(write_fd.get(), STDOUT_FILENO) < 0 ||
		    ::dup2(write_fd.get(), STDERR_FILENO) < 0) {
			::_exit(127);
		}
		write_fd.reset();

		::execvp(argv.front(), argv.data());
		::_exit(127);
	}
	(void)::setpgid(pid, pid);

	write_fd.reset();
	return CommandProcess { .pid = pid,
				.read_fd = ScopedFd(read_fd.release()) };
}

void drain_command_output(CommandProcess &process, CommandResult &result,
			  bool &pipe_open)
{
	std::array<char, 256> buffer {};
	while (pipe_open) {
		const ssize_t bytes_read = ::read(process.read_fd.get(),
						  buffer.data(), buffer.size());
		if (bytes_read > 0) {
			result.output.append(
				buffer.data(),
				static_cast<std::size_t>(bytes_read));
			continue;
		}
		if (bytes_read == 0) {
			pipe_open = false;
			process.read_fd.reset();
			return;
		}
		if (errno == EINTR) {
			continue;
		}
		if (errno == EAGAIN || errno == EWOULDBLOCK) {
			return;
		}
		pipe_open = false;
		process.read_fd.reset();
		return;
	}
}

void signal_command_process(pid_t pid, int signal_number)
{
	if (pid <= 0) {
		return;
	}
	(void)::kill(-pid, signal_number);
	(void)::kill(pid, signal_number);
}

void wait_for_child_detached(pid_t pid)
{
	if (pid <= 0) {
		return;
	}
	std::thread([pid] {
		int status = 0;
		while (::waitpid(pid, &status, 0) < 0 && errno == EINTR) {
		}
	}).detach();
}

CommandResult collect_command_result(CommandProcess process,
				     std::chrono::milliseconds timeout)
{
	CommandResult result;
	const auto started_at = std::chrono::steady_clock::now();
	const auto deadline = started_at + timeout;
	auto terminate_deadline = deadline + kCommandTerminateGrace;
	auto abandon_deadline = deadline + kCommandKillGrace;
	bool pipe_open = process.read_fd.get() >= 0;
	bool child_exited = false;
	bool terminate_sent = false;
	bool kill_sent = false;
	bool reaper_started = false;

	while (pipe_open || !child_exited) {
		if (pipe_open) {
			drain_command_output(process, result, pipe_open);
		}

		const auto now = std::chrono::steady_clock::now();
		if (!result.timed_out && (pipe_open || !child_exited) &&
		    now >= deadline) {
			result.timed_out = true;
			result.exit_code = -1;
			if (result.output.empty()) {
				result.output = "ERROR: COMMAND_TIMEOUT";
			} else if (!detail::contains_text(
					 result.output, "ERROR: COMMAND_TIMEOUT")) {
				result.output += "\nERROR: COMMAND_TIMEOUT";
			}
			signal_command_process(process.pid, SIGTERM);
			terminate_sent = true;
			terminate_deadline = now + kCommandTerminateGrace;
		}
		if (result.timed_out && terminate_sent && !kill_sent &&
		    (pipe_open || !child_exited) && now >= terminate_deadline) {
			signal_command_process(process.pid, SIGKILL);
			kill_sent = true;
			abandon_deadline = now + kCommandKillGrace;
		}

		const bool can_reap_child = !pipe_open ||
					    (result.timed_out && kill_sent);
		if (!child_exited && can_reap_child) {
			int status = 0;
			const pid_t wait_result =
				::waitpid(process.pid, &status, WNOHANG);
			if (wait_result == process.pid) {
				child_exited = true;
				if (!result.timed_out) {
					result.exit_code =
						WIFEXITED(status) ?
							WEXITSTATUS(status) :
							-1;
				}
			} else if (wait_result < 0 && errno != EINTR) {
				child_exited = true;
				result.exit_code = -1;
			}
		}

		if (result.timed_out && kill_sent && now >= abandon_deadline) {
			if (pipe_open) {
				pipe_open = false;
				process.read_fd.reset();
			}
			if (!child_exited) {
				wait_for_child_detached(process.pid);
				child_exited = true;
				reaper_started = true;
			}
		}

		if (!pipe_open && child_exited) {
			break;
		}

		auto wait_duration = kCommandPollInterval;
		const auto next_deadline =
			!result.timed_out ? deadline :
					    (!kill_sent ? terminate_deadline :
								  abandon_deadline);
		if (next_deadline <= now) {
			wait_duration = std::chrono::milliseconds { 0 };
		} else {
			wait_duration =
				std::min(wait_duration,
					 std::chrono::duration_cast<
						 std::chrono::milliseconds>(
						 next_deadline - now));
		}

		if (pipe_open) {
			pollfd poll_fd { .fd = process.read_fd.get(),
					 .events = POLLIN | POLLHUP | POLLERR,
					 .revents = 0 };
			(void)::poll(&poll_fd, 1,
				     static_cast<int>(wait_duration.count()));
		} else if (!child_exited) {
			std::this_thread::sleep_for(wait_duration);
		}
	}

	if (reaper_started) {
		result.exit_code = -1;
	}
	return result;
}

} // namespace

ScopedFd::ScopedFd(int fd)
	: _fd(fd)
{
}

ScopedFd::~ScopedFd()
{
	reset();
}

ScopedFd::ScopedFd(ScopedFd &&other) noexcept
	: _fd(other.release())
{
}

ScopedFd &ScopedFd::operator=(ScopedFd &&other) noexcept
{
	if (this != &other) {
		reset(other.release());
	}
	return *this;
}

int ScopedFd::get() const
{
	return _fd;
}

int ScopedFd::release()
{
	const int fd = _fd;
	_fd = -1;
	return fd;
}

void ScopedFd::reset(int fd)
{
	if (_fd >= 0) {
		::close(_fd);
	}
	_fd = fd;
}

std::string command_name_for_log(const std::vector<std::string> &args)
{
	if (args.empty()) {
		return "<empty>";
	}
	const std::filesystem::path path(args.front());
	const std::string filename = path.filename().string();
	return filename.empty() ? args.front() : filename;
}

std::string command_args_summary_for_log(const std::vector<std::string> &args)
{
	if (args.empty()) {
		return "<empty>";
	}
	std::string summary;
	for (const auto &arg : args) {
		if (!summary.empty()) {
			summary += ' ';
		}
		summary += arg;
		constexpr std::size_t kMaxSummaryLength = 160;
		if (summary.size() > kMaxSummaryLength) {
			summary.resize(kMaxSummaryLength);
			summary += "...";
			break;
		}
	}
	return summary;
}

CommandResult run_command(const std::vector<std::string> &args,
			  std::chrono::milliseconds timeout,
			  std::string_view summary)
{
	const auto started_at = std::chrono::steady_clock::now();
	const std::string command_name = command_name_for_log(args);
	const std::string command_summary =
		summary.empty() ? command_args_summary_for_log(args) :
				  std::string(summary);
	auto process = spawn_command(args);
	if (!process) {
		const auto ended_at = std::chrono::steady_clock::now();
		spdlog::warn(
			"NanoKVM runtime command spawn failed command='{}' summary='{}' duration_ms={} timeout_ms={} timed_out=false",
			command_name, command_summary,
			elapsed_ms(started_at, ended_at), timeout.count());
		return {};
	}
	auto result = collect_command_result(std::move(*process), timeout);
	const auto ended_at = std::chrono::steady_clock::now();
	if (result.timed_out) {
		spdlog::warn(
			"NanoKVM runtime command timed out command='{}' summary='{}' exit={} duration_ms={} timeout_ms={} timed_out={}",
			command_name, command_summary, result.exit_code,
			elapsed_ms(started_at, ended_at), timeout.count(),
			result.timed_out);
	} else {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime command end command='{}' summary='{}' exit={} duration_ms={} timeout_ms={} timed_out={}",
			command_name, command_summary, result.exit_code,
			elapsed_ms(started_at, ended_at), timeout.count(),
			result.timed_out);
	}
	return result;
}

void spawn_command_detached(const std::vector<std::string> &args)
{
	auto process = spawn_command(args);
	if (!process) {
		return;
	}
	process->read_fd.reset();
	wait_for_child_detached(process->pid);
}

bool run_success(const std::vector<std::string> &args,
			std::chrono::milliseconds timeout,
			std::string_view summary)
{
	const auto result = run_command(args, timeout, summary);
	if (result.exit_code != 0) {
		spdlog::warn(
			"NanoKVM runtime command failed: exit={}, timed_out={}, output={}",
			result.exit_code, result.timed_out,
			detail::trim(result.output));
		return false;
	}
	return true;
}

bool run_output_success(const std::vector<std::string> &args,
			std::string &output, std::chrono::milliseconds timeout,
			std::string_view summary)
{
	const auto result = run_command(args, timeout, summary);
	output = detail::trim(result.output);
	return result.exit_code == 0;
}

bool write_text_file_secure(const std::filesystem::path &path,
			    std::string_view payload, mode_t mode)
{
	const int fd = ::open(
		path.c_str(),
		O_WRONLY | O_CREAT | O_TRUNC | O_CLOEXEC | O_NOFOLLOW, mode);
	if (fd < 0) {
		spdlog::warn("Failed to create runtime file {}: {}",
			     path.string(), std::strerror(errno));
		return false;
	}

	ScopedFd scoped_fd(fd);
	if (::fchmod(scoped_fd.get(), mode) != 0) {
		spdlog::warn("Failed to set runtime file permissions {}: {}",
			     path.string(), std::strerror(errno));
		return false;
	}
	const ssize_t written =
		::write(scoped_fd.get(), payload.data(), payload.size());
	if (written != static_cast<ssize_t>(payload.size())) {
		spdlog::warn(
			"Failed to write runtime file {}: {}", path.string(),
			written < 0 ? std::strerror(errno) : "short write");
		return false;
	}
	return true;
}

} // namespace detail
} // namespace lvgl_main
