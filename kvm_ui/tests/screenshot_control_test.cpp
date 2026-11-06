#include <array>
#include <cerrno>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <map>
#include <string>
#include <utility>

#include <signal.h>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/screenshot_control.hpp"

using namespace lvgl_main;
using namespace lvgl_main::screenshot_control;

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_screenshot_control_test_" +
			 std::to_string(stamp));
		std::filesystem::create_directories(_path);
	}

	~TempDir()
	{
		std::error_code error;
		std::filesystem::remove_all(_path, error);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const
	{
		return _path;
	}

    private:
	std::filesystem::path _path;
};

void write_text(const std::filesystem::path &path, const std::string &text)
{
	std::ofstream out(path, std::ios::trunc);
	REQUIRE(out.good());
	out << text;
	REQUIRE(out.good());
}

std::string read_text(const std::filesystem::path &path)
{
	std::ifstream in(path);
	REQUIRE(in.good());
	std::string text((std::istreambuf_iterator<char>(in)),
			 std::istreambuf_iterator<char>());
	return text;
}

class FakeOps final : public ScreenshotControlOps {
    public:
	struct SignalCall {
		pid_t pid { -1 };
		int signal { 0 };
	};

	pid_t self_pid { 7001 };
	std::map<pid_t, std::string> executables;
	std::array<SignalCall, 4> signal_calls {};
	std::size_t signal_call_count { 0 };
	int liveness_error { 0 };
	int delivery_error { 0 };

	pid_t current_process_id() noexcept override
	{
		return self_pid;
	}

	bool ensure_parent_directory(const std::filesystem::path &path,
					     int &error) noexcept override
	{
		return _system.ensure_parent_directory(path, error);
	}

	int open_file(const std::filesystem::path &path, int flags, mode_t mode,
		      int &error) noexcept override
	{
		return _system.open_file(path, flags, mode, error);
	}

	bool close_file(int fd, int &error) noexcept override
	{
		return _system.close_file(fd, error);
	}

	bool is_regular_file(int fd, int &error) noexcept override
	{
		return _system.is_regular_file(fd, error);
	}

	bool lock_exclusive_nonblocking(int fd, int &error) noexcept override
	{
		return _system.lock_exclusive_nonblocking(fd, error);
	}

	bool path_matches_fd(const std::filesystem::path &path, int fd,
			     int &error) noexcept override
	{
		return _system.path_matches_fd(path, fd, error);
	}

	bool truncate_file(int fd, int &error) noexcept override
	{
		return _system.truncate_file(fd, error);
	}

	ssize_t read_file(int fd, char *buffer, std::size_t length,
			  std::size_t offset, int &error) noexcept override
	{
		return _system.read_file(fd, buffer, length, offset, error);
	}

	ssize_t write_file(int fd, const char *buffer, std::size_t length,
			   int &error) noexcept override
	{
		return _system.write_file(fd, buffer, length, error);
	}

	bool sync_file(int fd, int &error) noexcept override
	{
		return _system.sync_file(fd, error);
	}

	bool unlink_file(const std::filesystem::path &path, int &error) noexcept override
	{
		return _system.unlink_file(path, error);
	}

	bool signal_process(pid_t pid, int signal, int &error) noexcept override
	{
		if (signal_call_count < signal_calls.size()) {
			signal_calls[signal_call_count++] = { pid, signal };
		}
		if (signal == 0 && liveness_error != 0) {
			error = liveness_error;
			return false;
		}
		if (signal != 0 && delivery_error != 0) {
			error = delivery_error;
			return false;
		}
		error = 0;
		return true;
	}

	std::optional<std::string> executable_path(pid_t pid, int &error) override
	{
		const auto it = executables.find(pid);
		if (it == executables.end()) {
			error = ESRCH;
			return std::nullopt;
		}
		error = 0;
		return it->second;
	}

    private:
	SystemScreenshotControlOps _system;
};

} // namespace

TEST_CASE("screenshot signal policy accepts only canonical user signals")
{
	CHECK(signal_name(ScreenshotSignal::User1) == "SIGUSR1");
	CHECK(signal_name(ScreenshotSignal::User2) == "SIGUSR2");
	CHECK(signal_number(ScreenshotSignal::User1) == SIGUSR1);
	CHECK(signal_number(ScreenshotSignal::User2) == SIGUSR2);

	const auto first = parse_signal("SIGUSR1");
	const auto second = parse_signal("SIGUSR2");
	REQUIRE(first.has_value());
	REQUIRE(second.has_value());
	CHECK(*first == ScreenshotSignal::User1);
	CHECK(*second == ScreenshotSignal::User2);
	CHECK_FALSE(parse_signal("10").has_value());
	CHECK_FALSE(parse_signal("SIGKILL").has_value());
	CHECK_FALSE(parse_signal("SIGTERM").has_value());
}

TEST_CASE("screenshot instance records are canonical and bounded")
{
	CHECK(format_instance_record(42, ScreenshotSignal::User2) ==
	      "42 SIGUSR2\n");

	const auto parsed = parse_instance_record("42 SIGUSR1\n");
	REQUIRE(parsed.has_value());
	CHECK(parsed->pid == 42);
	CHECK(parsed->signal == ScreenshotSignal::User1);

	CHECK_FALSE(parse_instance_record("0 SIGUSR1\n").has_value());
	CHECK_FALSE(parse_instance_record("042 SIGUSR1\n").has_value());
	CHECK_FALSE(parse_instance_record("42 SIGKILL\n").has_value());
	CHECK_FALSE(parse_instance_record("42 SIGUSR1").has_value());
	CHECK_FALSE(parse_instance_record("42 SIGUSR1\nextra").has_value());
	CHECK_FALSE(parse_instance_record(std::string(kMaxPidfileRecordBytes + 1U,
							 'x'))
			.has_value());
}

TEST_CASE("screenshot instance lease replaces stale record and cleans its own file")
{
	TempDir temp;
	const auto pidfile = temp.path() / "kvm_ui.pid";
	write_text(pidfile, "999 SIGUSR1\n");

	FakeOps ops;
	auto acquired = acquire_instance_lease(pidfile, 4242,
					       ScreenshotSignal::User2, ops);
	REQUIRE(acquired.result.ok);
	REQUIRE(acquired.lease.has_value());
	CHECK(acquired.lease->active());
	CHECK(read_text(pidfile) == "4242 SIGUSR2\n");

	auto duplicate = acquire_instance_lease(pidfile, 4243,
					       ScreenshotSignal::User1, ops);
	CHECK_FALSE(duplicate.result.ok);
	CHECK(duplicate.result.error == ScreenshotControlError::PidfileBusy);

	acquired.lease.reset();
	CHECK_FALSE(std::filesystem::exists(pidfile));
}

TEST_CASE("screenshot instance lease creates a missing pidfile parent directory")
{
	TempDir temp;
	const auto pidfile = temp.path() / "run" / "kvm" / "kvm_ui.pid";
	FakeOps ops;

	auto acquired = acquire_instance_lease(pidfile, 4242,
					       ScreenshotSignal::User1, ops);
	REQUIRE(acquired.result.ok);
	REQUIRE(acquired.lease.has_value());
	CHECK(std::filesystem::is_directory(pidfile.parent_path()));
	CHECK(read_text(pidfile) == "4242 SIGUSR1\n");

	acquired.lease.reset();
	CHECK_FALSE(std::filesystem::exists(pidfile));
}

TEST_CASE("screenshot instance lease preserves a replacement pidfile")
{
	TempDir temp;
	const auto pidfile = temp.path() / "kvm_ui.pid";
	FakeOps ops;
	auto acquired = acquire_instance_lease(pidfile, 4242,
					       ScreenshotSignal::User1, ops);
	REQUIRE(acquired.result.ok);
	REQUIRE(acquired.lease.has_value());

	std::filesystem::remove(pidfile);
	write_text(pidfile, "999 SIGUSR2\n");
	acquired.lease.reset();

	CHECK(read_text(pidfile) == "999 SIGUSR2\n");
}

TEST_CASE("screenshot sender validates executable identity before signaling")
{
	TempDir temp;
	const auto pidfile = temp.path() / "kvm_ui.pid";
	write_text(pidfile, "4242 SIGUSR2\n");

	FakeOps ops;
	ops.executables.emplace(4242, "/usr/bin/kvm_ui");
	const auto sent = send_screenshot_request(pidfile, "/usr/bin/kvm_ui", ops);

	REQUIRE(sent.ok);
	REQUIRE(ops.signal_call_count == 2U);
	CHECK(ops.signal_calls[0].pid == 4242);
	CHECK(ops.signal_calls[0].signal == 0);
	CHECK(ops.signal_calls[1].pid == 4242);
	CHECK(ops.signal_calls[1].signal == SIGUSR2);

	FakeOps mismatch_ops;
	mismatch_ops.executables.emplace(4242, "/usr/bin/not-kvm-ui");
	const auto mismatch = send_screenshot_request(
		pidfile, "/usr/bin/kvm_ui", mismatch_ops);
	CHECK_FALSE(mismatch.ok);
	CHECK(mismatch.error == ScreenshotControlError::ExecutableMismatch);
	CHECK(mismatch_ops.signal_call_count == 1U);
}

TEST_CASE("screenshot sender reports stale and undeliverable instances")
{
	TempDir temp;
	const auto pidfile = temp.path() / "kvm_ui.pid";
	write_text(pidfile, "4242 SIGUSR1\n");

	FakeOps unavailable_ops;
	unavailable_ops.liveness_error = ESRCH;
	const auto unavailable = send_screenshot_request(
		pidfile, "/usr/bin/kvm_ui", unavailable_ops);
	CHECK_FALSE(unavailable.ok);
	CHECK(unavailable.error == ScreenshotControlError::ProcessUnavailable);

	FakeOps failed_delivery_ops;
	failed_delivery_ops.executables.emplace(4242, "/usr/bin/kvm_ui");
	failed_delivery_ops.delivery_error = EPERM;
	const auto failed_delivery = send_screenshot_request(
		pidfile, "/usr/bin/kvm_ui", failed_delivery_ops);
	CHECK_FALSE(failed_delivery.ok);
	CHECK(failed_delivery.error == ScreenshotControlError::SignalFailed);
	CHECK(failed_delivery_ops.signal_call_count == 2U);
}

TEST_CASE("screenshot sender resolves the current executable through the seam")
{
	FakeOps ops;
	ops.executables.emplace(ops.self_pid, "/usr/bin/kvm_ui");
	ScreenshotControlResult result;
	const auto executable = current_executable_path(ops, result);

	REQUIRE(executable.has_value());
	CHECK(result.ok);
	CHECK(*executable == "/usr/bin/kvm_ui");
}
