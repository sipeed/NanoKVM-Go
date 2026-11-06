#pragma once

#include <filesystem>
#include <string>

namespace lvgl_main::backlight
{

inline constexpr int kBacklightRawMaxDefault = 255;
inline constexpr int kBacklightUiMinPercent = 20;
inline constexpr int kBacklightUiMaxPercent = 100;

struct BacklightState {
	bool available { false };
	int raw { 0 };
	int raw_max { kBacklightRawMaxDefault };
	// Configured or caller-provided UI percent. Sysfs raw readings do not
	// override this value because raw 0..max is only an observation channel.
	int percent { kBacklightUiMaxPercent };
	bool screen_on { true };
	std::string error;
};

struct BacklightWriteResult {
	bool ok { false };
	int raw { 0 };
	int raw_max { kBacklightRawMaxDefault };
	std::string error;
};

int clamp_backlight_percent(int percent);
int normalize_raw_max(int raw_max);
int backlight_percent_to_raw(int percent, int raw_max = kBacklightRawMaxDefault,
			     bool invert = false);
// Nonlinear inverse helper for diagnostics and tests only. Runtime UI percent
// is sourced from config or the last successful set request, not from sysfs raw.
int backlight_raw_to_percent(int raw, int raw_max = kBacklightRawMaxDefault,
			     bool invert = false);
BacklightState read_backlight_state(const std::filesystem::path &root,
				    int fallback_percent, bool invert = false);
BacklightWriteResult write_backlight_percent(const std::filesystem::path &root,
					     int percent, bool invert = false);
BacklightWriteResult write_backlight_off(const std::filesystem::path &root,
					 bool invert = false);

} // namespace lvgl_main::backlight
