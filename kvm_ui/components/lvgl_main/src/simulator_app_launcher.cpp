#include "lvgl_main/simulator_app_launcher.hpp"

#include <spdlog/spdlog.h>

namespace lvgl_main
{

SimulatorAppLaunchResult SimulatorAppLauncher::launch(std::string_view app_path)
{
	if (app_path.empty()) {
		spdlog::warn(
			"Simulator Apps mock launch rejected an empty path");
		return {
			.status = SimulatorAppLaunchStatus::InvalidRequest,
			.detail = "app path is empty",
		};
	}

	_last_launched_path = std::string(app_path);
	spdlog::info("[simulator] Simulator Apps mock launch completed: {}",
		     app_path);
	return {
		.status = SimulatorAppLaunchStatus::Completed,
		.detail = "launch simulated; no child process started",
	};
}

std::optional<std::string> SimulatorAppLauncher::last_launched_path() const
{
	return _last_launched_path;
}

} // namespace lvgl_main
