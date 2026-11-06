#pragma once

#include <cstdint>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main
{

enum class SimulatorAppLaunchStatus : uint8_t {
	Completed,
	InvalidRequest,
};

struct SimulatorAppLaunchResult {
	SimulatorAppLaunchStatus status {
		SimulatorAppLaunchStatus::InvalidRequest
	};
	std::string detail;

	bool ok() const
	{
		return status == SimulatorAppLaunchStatus::Completed;
	}
};

class SimulatorAppLauncher final {
    public:
	SimulatorAppLaunchResult launch(std::string_view app_path);
	std::optional<std::string> last_launched_path() const;

    private:
	std::optional<std::string> _last_launched_path;
};

} // namespace lvgl_main
