#include "lvgl_main/backlight_control.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <memory>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main::backlight
{

namespace
{

// Android BrightnessUtils-style gamma/exponential curve constants.
constexpr double kGammaR = 0.5;
constexpr double kGammaA = 0.17883277;
constexpr double kGammaB = 0.28466892;
constexpr double kGammaC = 0.55991073;
constexpr double kGammaCurveMax = 12.0;
constexpr double kMinLinearBrightness = 0.20;

std::optional<int> read_int_file(const std::filesystem::path &path,
				 std::string &error)
{
	std::FILE *file = std::fopen(path.c_str(), "re");
	if (file == nullptr) {
		error = path.string() + ": " + std::strerror(errno);
		return std::nullopt;
	}
	std::unique_ptr<std::FILE, decltype(&std::fclose)> close_file(
		file, &std::fclose);

	std::array<char, 64> buffer {};
	if (std::fgets(buffer.data(), static_cast<int>(buffer.size()), file) ==
	    nullptr) {
		error = path.string() + ": empty or unreadable";
		return std::nullopt;
	}

	try {
		std::size_t consumed = 0;
		const int value = std::stoi(buffer.data(), &consumed);
		(void)consumed;
		return value;
	} catch (const std::exception &) {
		error = path.string() + ": invalid integer";
		return std::nullopt;
	}
}

int read_raw_max(const std::filesystem::path &root)
{
	std::string ignored_error;
	const auto value =
		read_int_file(root / "max_brightness", ignored_error);
	return normalize_raw_max(value.value_or(kBacklightRawMaxDefault));
}

bool write_text_file(const std::filesystem::path &path, std::string_view text,
		     std::string &error)
{
	std::FILE *file = std::fopen(path.c_str(), "we");
	if (file == nullptr) {
		error = path.string() + ": " + std::strerror(errno);
		return false;
	}
	std::unique_ptr<std::FILE, decltype(&std::fclose)> close_file(
		file, &std::fclose);

	const auto written = std::fwrite(text.data(), 1U, text.size(), file);
	if (written != text.size() || std::fflush(file) != 0) {
		error = path.string() + ": write failed";
		return false;
	}
	return true;
}

BacklightWriteResult write_raw(const std::filesystem::path &root, int raw)
{
	BacklightWriteResult result;
	result.raw_max = read_raw_max(root);
	result.raw = std::clamp(raw, 0, result.raw_max);
	result.ok = write_text_file(root / "brightness",
				    std::to_string(result.raw), result.error);
	return result;
}

double ui_percent_to_gamma(int percent)
{
	const int clamped = clamp_backlight_percent(percent);
	const int span = kBacklightUiMaxPercent - kBacklightUiMinPercent;
	return static_cast<double>(clamped - kBacklightUiMinPercent) /
	       static_cast<double>(span);
}

double gamma_to_android_curve(double gamma)
{
	const double clamped_gamma = std::clamp(gamma, 0.0, 1.0);
	const double curve =
		clamped_gamma <= kGammaR ?
			std::pow(clamped_gamma / kGammaR, 2.0) :
			std::exp((clamped_gamma - kGammaC) / kGammaA) + kGammaB;
	return std::clamp(curve / kGammaCurveMax, 0.0, 1.0);
}

double android_curve_to_gamma(double curve)
{
	const double scaled_curve =
		std::clamp(curve, 0.0, 1.0) * kGammaCurveMax;
	if (scaled_curve <= 1.0) {
		return std::clamp(kGammaR * std::sqrt(scaled_curve), 0.0, 1.0);
	}

	const double gamma =
		kGammaA * std::log(scaled_curve - kGammaB) + kGammaC;
	return std::clamp(gamma, 0.0, 1.0);
}

} // namespace

int clamp_backlight_percent(int percent)
{
	return std::clamp(percent, kBacklightUiMinPercent,
			  kBacklightUiMaxPercent);
}

int normalize_raw_max(int raw_max)
{
	return raw_max > 0 ? raw_max : kBacklightRawMaxDefault;
}

int backlight_percent_to_raw(int percent, int raw_max, bool invert)
{
	const int max = normalize_raw_max(raw_max);
	const double curved =
		gamma_to_android_curve(ui_percent_to_gamma(percent));
	const double normalized =
		kMinLinearBrightness + (1.0 - kMinLinearBrightness) * curved;
	int raw = static_cast<int>(
		std::lround((invert ? 1.0 - normalized : normalized) * max));
	if (invert && raw == max && max > 0) {
		raw = max - 1;
	} else if (!invert && raw == 0 && max > 0) {
		raw = 1;
	}
	return std::clamp(raw, 0, max);
}

int backlight_raw_to_percent(int raw, int raw_max, bool invert)
{
	const int max = normalize_raw_max(raw_max);
	const int clamped_raw = std::clamp(raw, 0, max);
	const double raw_ratio =
		static_cast<double>(clamped_raw) / static_cast<double>(max);
	const double normalized = invert ? 1.0 - raw_ratio : raw_ratio;
	const double curved = (normalized - kMinLinearBrightness) /
			      (1.0 - kMinLinearBrightness);
	const double gamma = android_curve_to_gamma(curved);
	const int percent = static_cast<int>(std::lround(
		kBacklightUiMinPercent +
		gamma * (kBacklightUiMaxPercent - kBacklightUiMinPercent)));
	return clamp_backlight_percent(percent);
}

BacklightState read_backlight_state(const std::filesystem::path &root,
				    int fallback_percent, bool invert)
{
	BacklightState state;
	state.raw_max = read_raw_max(root);
	state.percent = clamp_backlight_percent(fallback_percent);

	std::string error;
	const auto raw = read_int_file(root / "brightness", error);
	if (!raw) {
		state.error = error;
		return state;
	}

	state.available = true;
	state.raw = std::clamp(*raw, 0, state.raw_max);
	state.screen_on = invert ? state.raw < state.raw_max : state.raw > 0;
	return state;
}

BacklightWriteResult write_backlight_percent(const std::filesystem::path &root,
					     int percent, bool invert)
{
	const int raw_max = read_raw_max(root);
	return write_raw(root,
			 backlight_percent_to_raw(percent, raw_max, invert));
}

BacklightWriteResult write_backlight_off(const std::filesystem::path &root,
					 bool invert)
{
	const int raw_max = read_raw_max(root);
	return write_raw(root, invert ? raw_max : 0);
}

} // namespace lvgl_main::backlight
