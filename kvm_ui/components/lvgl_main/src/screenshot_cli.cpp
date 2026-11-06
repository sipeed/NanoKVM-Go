#include "lvgl_main/screenshot_cli.hpp"

#include <cstring>
#include <utility>

namespace lvgl_main::screenshot_cli
{

namespace
{

bool is_screenshot_argument(const char *argument)
{
	return argument != nullptr && std::strcmp(argument, "--screenshot") == 0;
}

std::string failure_message(const screenshot_control::ScreenshotControlResult &result)
{
	return result.message.empty() ? "Screenshot request failed." :
		"Screenshot request failed: " + result.message;
}

} // namespace

ScreenshotCliResult try_handle_screenshot_cli(
	int argc, char *const argv[], bool target_available,
	const std::filesystem::path &pidfile_path,
	screenshot_control::ScreenshotControlOps &ops)
{
	bool screenshot_requested = false;
	for (int index = 1; index < argc; ++index) {
		if (is_screenshot_argument(argv[index])) {
			screenshot_requested = true;
			break;
		}
	}
	if (!screenshot_requested) {
		return {};
	}

	if (argc != 2 || !is_screenshot_argument(argv[1])) {
		return {
			.handled = true,
			.exit_code = 2,
			.message = "--screenshot must be used by itself.",
		};
	}
	if (!target_available) {
		return {
			.handled = true,
			.exit_code = 2,
			.message = "Screenshot requests are unavailable in simulator mode.",
		};
	}

	screenshot_control::ScreenshotControlResult executable_result;
	const auto executable =
		screenshot_control::current_executable_path(ops, executable_result);
	if (!executable.has_value()) {
		return {
			.handled = true,
			.exit_code = 1,
			.message = failure_message(executable_result),
		};
	}

	const auto request_result = screenshot_control::send_screenshot_request(
		pidfile_path, *executable, ops);
	if (!request_result.ok) {
		return {
			.handled = true,
			.exit_code = 1,
			.message = failure_message(request_result),
		};
	}
	return {
		.handled = true,
		.exit_code = 0,
		.message = "Screenshot request sent.",
	};
}

} // namespace lvgl_main::screenshot_cli
