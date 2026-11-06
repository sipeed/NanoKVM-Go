#include <algorithm>
#include <cerrno>
#include <cstring>
#include <filesystem>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/screenshot_cli.hpp"

namespace
{

class FakeScreenshotControlOps final : public lvgl_main::screenshot_control::ScreenshotControlOps {
    public:
	pid_t current_process_id() noexcept override
	{
		++current_process_id_calls;
		return self_pid;
	}

	bool ensure_parent_directory(const std::filesystem::path &, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	int open_file(const std::filesystem::path &, int, mode_t, int &error) noexcept override
	{
		++open_calls;
		error = 0;
		return 7;
	}

	bool close_file(int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool is_regular_file(int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool lock_exclusive_nonblocking(int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool path_matches_fd(const std::filesystem::path &, int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool truncate_file(int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	ssize_t read_file(int, char *buffer, std::size_t length,
			  std::size_t offset, int &error) noexcept override
	{
		if (offset >= record.size()) {
			error = 0;
			return 0;
		}
		const std::size_t count =
			std::min(length, record.size() - offset);
		std::memcpy(buffer, record.data() + offset, count);
		error = 0;
		return static_cast<ssize_t>(count);
	}

	ssize_t write_file(int, const char *, std::size_t length,
			   int &error) noexcept override
	{
		error = 0;
		return static_cast<ssize_t>(length);
	}

	bool sync_file(int, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool unlink_file(const std::filesystem::path &, int &error) noexcept override
	{
		error = 0;
		return true;
	}

	bool signal_process(pid_t pid, int signal, int &error) noexcept override
	{
		signals.emplace_back(pid, signal);
		error = 0;
		return true;
	}

	std::optional<std::string> executable_path(pid_t pid, int &error) override
	{
		++executable_path_calls;
		if (pid == self_pid || pid == target_pid) {
			error = 0;
			return executable;
		}
		error = ESRCH;
		return std::nullopt;
	}

	pid_t self_pid { 100 };
	pid_t target_pid { 200 };
	std::string executable { "/usr/bin/kvm_ui" };
	std::string record;
	int current_process_id_calls { 0 };
	int executable_path_calls { 0 };
	int open_calls { 0 };
	std::vector<std::pair<pid_t, int>> signals;
};

constexpr const char *kPidfilePath = "/tmp/kvm_ui_screenshot.pid";

} // namespace

TEST_CASE("screenshot CLI leaves non-screenshot arguments for the normal parser")
{
	FakeScreenshotControlOps ops;
	char program[] = "kvm_ui";
	char version[] = "--version";
	char *argv[] = { program, version };

	const auto result = lvgl_main::screenshot_cli::try_handle_screenshot_cli(
		2, argv, true, kPidfilePath, ops);

	CHECK_FALSE(result.handled);
	CHECK(ops.current_process_id_calls == 0);
	CHECK(ops.open_calls == 0);
}

TEST_CASE("screenshot CLI requires --screenshot to be mutually exclusive")
{
	FakeScreenshotControlOps ops;
	char program[] = "kvm_ui";
	char screenshot[] = "--screenshot";
	char version[] = "--version";
	char *argv[] = { program, screenshot, version };

	const auto result = lvgl_main::screenshot_cli::try_handle_screenshot_cli(
		3, argv, true, kPidfilePath, ops);

	REQUIRE(result.handled);
	CHECK(result.exit_code == 2);
	CHECK(result.message == "--screenshot must be used by itself.");
	CHECK(ops.current_process_id_calls == 0);
	CHECK(ops.open_calls == 0);
}

TEST_CASE("screenshot CLI is unavailable in simulator mode without process access")
{
	FakeScreenshotControlOps ops;
	char program[] = "kvm_ui";
	char screenshot[] = "--screenshot";
	char *argv[] = { program, screenshot };

	const auto result = lvgl_main::screenshot_cli::try_handle_screenshot_cli(
		2, argv, false, kPidfilePath, ops);

	REQUIRE(result.handled);
	CHECK(result.exit_code == 2);
	CHECK(result.message ==
	      "Screenshot requests are unavailable in simulator mode.");
	CHECK(ops.current_process_id_calls == 0);
	CHECK(ops.open_calls == 0);
}

TEST_CASE("screenshot CLI signals only the validated pidfile target")
{
	FakeScreenshotControlOps ops;
	ops.record = lvgl_main::screenshot_control::format_instance_record(
		ops.target_pid, lvgl_main::ScreenshotSignal::User1);
	char program[] = "kvm_ui";
	char screenshot[] = "--screenshot";
	char *argv[] = { program, screenshot };

	const auto result = lvgl_main::screenshot_cli::try_handle_screenshot_cli(
		2, argv, true, kPidfilePath, ops);

	REQUIRE(result.handled);
	CHECK(result.exit_code == 0);
	CHECK(result.message == "Screenshot request sent.");
	REQUIRE(ops.signals.size() == 2U);
	CHECK(ops.signals[0] == std::pair { ops.target_pid, 0 });
	CHECK(ops.signals[1] == std::pair {
		ops.target_pid,
		lvgl_main::screenshot_control::signal_number(
			lvgl_main::ScreenshotSignal::User1),
	});
}
