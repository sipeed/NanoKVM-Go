#pragma once

#include <filesystem>
#include <string>

#include "lvgl_main/screenshot_control.hpp"

namespace lvgl_main::screenshot_cli
{

struct ScreenshotCliResult {
	bool handled { false };
	int exit_code { 0 };
	std::string message;
};

// Handles only the mutually exclusive --screenshot command. A non-screenshot
// invocation is left untouched for the normal application CLI parser.
ScreenshotCliResult try_handle_screenshot_cli(
	int argc, char *const argv[], bool target_available,
	const std::filesystem::path &pidfile_path,
	screenshot_control::ScreenshotControlOps &ops);

} // namespace lvgl_main::screenshot_cli
