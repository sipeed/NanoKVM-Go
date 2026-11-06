#include "lvgl_main/app_config.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <optional>
#include <sstream>
#include <string_view>
#include <system_error>

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#include <toml++/toml.hpp>

#include "lvgl_main/http_api_contract.hpp"

namespace lvgl_main
{

namespace
{

constexpr std::string_view kDisplay = "display";
constexpr std::string_view kTouch = "touch";
constexpr std::string_view kHttpApi = "http_api";
constexpr std::string_view kRuntime = "runtime";
constexpr std::string_view kScreenshot = "screenshot";
constexpr std::string_view kUi = "ui";
constexpr std::string_view kFonts = "fonts";
constexpr std::string_view kExperimental = "experimental";
constexpr std::string_view kNanokvm = "nanokvm";
constexpr std::string_view kLauncher = "launcher";

constexpr std::string_view kFramebufferDevice = "framebuffer_device";
constexpr std::array<std::string_view, 4> kObsoleteDisplayGammaKeys {
	"gamma_green_correction",
	"gamma_green_knee_start",
	"gamma_green_knee_slope",
	"gamma_rgb_global",
};
constexpr std::array<std::string_view, 3> kObsoleteUiKeys {
	"enable_global_popup_demo",
	"enable_developer_mode",
	"wifi_config_mode",
};
constexpr std::array<std::string_view, 2> kObsoleteLauncherKeys {
	"apps_dir",
	"demo_script",
};
constexpr std::array<std::string_view, 5> kObsoleteExperimentalKeys {
	"ttf",
	"enable_settings_optimistic_actions",
	"network_settings_subpage",
	"network_terms_qr_card_size_px",
	"network_terms_qr_card_radius_px",
};
constexpr std::array<std::string_view, 1> kObsoleteNanokvmKeys {
	"ncm_enabled",
};
constexpr std::string_view kDevice = "device";
constexpr std::string_view kDisabled = "disabled";
constexpr std::string_view kEnabled = "enabled";
constexpr std::string_view kBindHost = "bind_host";
constexpr std::string_view kPort = "port";
constexpr std::string_view kToken = "token";
constexpr std::string_view kRuntimeDebugLogging = "debug_logging";
constexpr std::string_view kRuntimeTouchClickMarker = "touch_click_marker";
constexpr std::string_view kRuntimeTouchCalibrate = "touch_calibrate";
constexpr std::string_view kRuntimeTouchPaintTest = "touch_paint_test";
constexpr std::string_view kScreenshotSignal = "signal";
constexpr std::string_view kScreenshotOutputDirectory = "output_directory";
constexpr std::string_view kEnableAutoRotation = "enable_auto_rotation";
constexpr std::string_view kManualDisplayRotation = "manual_display_rotation";
constexpr std::string_view kExperimentalSettingsSubpages =
	"experimental_settings_subpages";
constexpr std::string_view kLegacyEnable180Rotation = "enable_180_rotation";
constexpr std::string_view kPageTransitionMode = "page_transition_mode";
constexpr std::string_view kWifiBootReconnectTimeoutMs =
	"wifi_boot_reconnect_timeout_ms";
constexpr std::string_view kSettingsStateBootTimeoutMs =
	"settings_state_boot_timeout_ms";
constexpr std::string_view kWifiLoadingProgress =
	"wifi_loading_progress";
constexpr std::string_view kCjkRegion = "cjk_region";
constexpr std::string_view kSsidImageCacheKb = "ssid_image_cache_kb";
constexpr std::string_view kExperimentalReserved = "reserved";
constexpr std::string_view kExperimentalSsidScroll = "ssid_scroll";
constexpr std::string_view kLegacyExperimentalEnableSsidScroll =
	"enable_ssid_scroll";
constexpr std::string_view kEnableSettingsOptimisticActions =
	"enable_settings_optimistic_actions";
constexpr std::string_view kExperimentalWifiLoadingMode =
	"wifi_loading_mode";
constexpr std::string_view kEnableRuntimeBackend = "enable_runtime_backend";
constexpr std::string_view kPreserveLastKnownGoodState =
	"preserve_last_known_good_state";
constexpr std::string_view kServerBaseUrl = "server_base_url";
constexpr std::string_view kServerVerifyTls = "server_verify_tls";
constexpr std::string_view kWifiScript = "wifi_script";
constexpr std::string_view kUsbGadgetRestartScript =
	"usb_gadget_restart_script";
constexpr std::string_view kLegacyUsbScript = "usb_script";
constexpr std::string_view kBacklightRoot = "backlight_root";
constexpr std::string_view kEnableHighRiskActions = "enable_high_risk_actions";
constexpr std::string_view kVideoEnabled = "video_enabled";
constexpr std::string_view kHidEnabled = "hid_enabled";
constexpr std::string_view kBacklightPercent = "backlight_percent";
constexpr std::string_view kBacklightInvert = "backlight_invert";
constexpr std::string_view kAutoSleepEnabled = "auto_sleep_enabled";
constexpr std::string_view kLauncherPythonBin = "python_bin";
constexpr std::string_view kLauncherExitZonePx = "exit_gesture_zone_px";
constexpr std::string_view kLauncherExitHoldMs = "exit_gesture_hold_ms";
constexpr std::string_view kLauncherChildTermTimeoutMs =
	"child_term_timeout_ms";
constexpr std::uintmax_t kMaxConfigFileBytes = 64U * 1024U;
constexpr mode_t kPrivateFileMode = S_IRUSR | S_IWUSR;

std::string path_for(std::string_view section, std::string_view key)
{
	std::string result(section);
	result += ".";
	result += key;
	return result;
}

void mark_repaired(AppConfigLoadResult &result, std::string message)
{
	result.repaired = true;
	result.messages.push_back(std::move(message));
}

void mark_warning(AppConfigLoadResult &result, std::string message)
{
	result.warnings.push_back(std::move(message));
}

struct RuntimeTouchOption {
	std::string_view key;
	bool value { false };
	std::optional<toml::source_position> source;
};

bool is_valid_fbdev_path(std::string_view path)
{
	constexpr std::string_view prefix = "/dev/fb";
	if (path.size() <= prefix.size() ||
	    path.substr(0, prefix.size()) != prefix) {
		return false;
	}

	return std::all_of(path.begin() +
				   static_cast<std::ptrdiff_t>(prefix.size()),
			   path.end(), [](char ch) {
				   return ch >= '0' && ch <= '9';
			   });
}

bool is_valid_input_event_path(std::string_view path)
{
	constexpr std::string_view prefix = "/dev/input/event";
	if (path.size() <= prefix.size() ||
	    path.substr(0, prefix.size()) != prefix) {
		return false;
	}

	return std::all_of(path.begin() +
				   static_cast<std::ptrdiff_t>(prefix.size()),
			   path.end(), [](char ch) {
				   return ch >= '0' && ch <= '9';
			   });
}

bool is_non_empty_string(std::string_view text)
{
	return !text.empty();
}

bool is_valid_ui_page_transition_mode(std::string_view value)
{
	return value == "slide" || value == "flip";
}

bool is_valid_cjk_region(std::string_view value)
{
	return parse_cjk_region(value).has_value();
}

UiPageTransitionMode parse_ui_page_transition_mode(std::string_view value)
{
	return value == "flip" ? UiPageTransitionMode::Flip :
				 UiPageTransitionMode::Slide;
}

std::string_view ui_page_transition_mode_name(UiPageTransitionMode mode)
{
	switch (mode) {
	case UiPageTransitionMode::Slide:
		return "slide";
	case UiPageTransitionMode::Flip:
		return "flip";
	}
	return app_config_defaults::kUiPageTransitionMode;
}

bool is_valid_server_base_url(std::string_view url)
{
	constexpr std::string_view http_prefix = "http://";
	constexpr std::string_view https_prefix = "https://";
	return (url.size() > http_prefix.size() &&
		url.substr(0, http_prefix.size()) == http_prefix) ||
	       (url.size() > https_prefix.size() &&
		url.substr(0, https_prefix.size()) == https_prefix);
}

bool is_valid_absolute_path(std::string_view path)
{
	return path.size() > 1U && path.front() == '/';
}

bool is_valid_screenshot_signal(std::string_view value)
{
	return screenshot_control::parse_signal(value).has_value();
}

bool is_valid_screenshot_output_directory(std::string_view path)
{
	if (path.size() <= 1U || path.front() != '/' ||
	    path.find('\0') != std::string_view::npos) {
		return false;
	}

	std::size_t component_start = 1U;
	while (component_start < path.size()) {
		const std::size_t separator = path.find('/', component_start);
		const std::size_t component_end = separator == std::string_view::npos ?
							      path.size() :
							      separator;
		const std::string_view component = path.substr(
			component_start, component_end - component_start);
		if (component.empty() || component == "." || component == "..") {
			return false;
		}
		for (const unsigned char ch : component) {
			if (ch < 0x20U || ch == 0x7FU) {
				return false;
			}
		}
		if (separator == std::string_view::npos) {
			break;
		}
		component_start = separator + 1U;
	}
	return component_start <= path.size();
}

class ScopedFd final {
    public:
	explicit ScopedFd(int fd)
		: _fd(fd)
	{
	}

	~ScopedFd()
	{
		if (_fd >= 0) {
			::close(_fd);
		}
	}

	ScopedFd(const ScopedFd &) = delete;
	ScopedFd &operator=(const ScopedFd &) = delete;

	int get() const
	{
		return _fd;
	}

	int release()
	{
		const int fd = _fd;
		_fd = -1;
		return fd;
	}

    private:
	int _fd { -1 };
};

bool set_private_file_permissions(int fd, std::string_view path,
				  std::string &error)
{
	struct stat st {};
	if (::fstat(fd, &st) != 0) {
		error = "failed to inspect config file '" + std::string(path) +
			"': " + std::strerror(errno);
		return false;
	}
	if (!S_ISREG(st.st_mode)) {
		error = "refusing to chmod non-regular config file '" +
			std::string(path) + "'";
		return false;
	}
	if (::fchmod(fd, kPrivateFileMode) != 0) {
		error = "failed to set private permissions on '" +
			std::string(path) + "': " + std::strerror(errno);
		return false;
	}
	return true;
}

bool set_private_file_permissions(const std::filesystem::path &path,
				  std::string &error)
{
	const int fd = ::open(path.c_str(), O_RDONLY | O_CLOEXEC | O_NOFOLLOW);
	if (fd < 0) {
		error = "failed to open config file '" + path.string() +
			"' for permission repair: " + std::strerror(errno);
		return false;
	}

	ScopedFd scoped_fd(fd);
	return set_private_file_permissions(scoped_fd.get(), path.string(),
					    error);
}

bool validate_parent_directory(const std::filesystem::path &path,
			       std::string &error)
{
	const auto parent = path.parent_path();
	if (parent.empty()) {
		return true;
	}

	struct stat st {};
	if (::lstat(parent.c_str(), &st) != 0) {
		error = "failed to inspect config directory '" +
			parent.string() + "': " + std::strerror(errno);
		return false;
	}
	if (S_ISLNK(st.st_mode) || !S_ISDIR(st.st_mode)) {
		error = "refusing to use non-directory config parent '" +
			parent.string() + "'";
		return false;
	}
	if (st.st_uid != 0U && st.st_uid != ::geteuid()) {
		error = "refusing to use config directory with unexpected owner '" +
			parent.string() + "'";
		return false;
	}
	if ((st.st_mode & (S_IWGRP | S_IWOTH)) != 0U) {
		error = "refusing to use group/world-writable config directory '" +
			parent.string() + "'";
		return false;
	}
	return true;
}

bool read_all(int fd, std::size_t size, std::string &data, std::string &error)
{
	data.assign(size, '\0');
	char *ptr = data.data();
	std::size_t remaining = size;
	while (remaining > 0U) {
		const ssize_t bytes_read = ::read(fd, ptr, remaining);
		if (bytes_read < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (bytes_read == 0) {
			error = "short read";
			return false;
		}
		ptr += bytes_read;
		remaining -= static_cast<std::size_t>(bytes_read);
	}
	return true;
}

bool read_config_file(const std::filesystem::path &path, std::string &data,
		      std::string &error)
{
	const int fd = ::open(path.c_str(), O_RDONLY | O_CLOEXEC | O_NOFOLLOW);
	if (fd < 0) {
		error = "failed to open config file '" + path.string() +
			"': " + std::strerror(errno);
		return false;
	}

	ScopedFd scoped_fd(fd);
	struct stat st {};
	if (::fstat(scoped_fd.get(), &st) != 0) {
		error = "failed to inspect config file '" + path.string() +
			"': " + std::strerror(errno);
		return false;
	}
	if (!S_ISREG(st.st_mode)) {
		error = "refusing to read non-regular config file '" +
			path.string() + "'";
		return false;
	}
	if (st.st_size < 0 ||
	    static_cast<std::uintmax_t>(st.st_size) > kMaxConfigFileBytes) {
		error = "refusing to read oversized config file '" +
			path.string() + "'";
		return false;
	}
	if (!set_private_file_permissions(scoped_fd.get(), path.string(),
					  error)) {
		return false;
	}

	std::string read_error;
	if (!read_all(scoped_fd.get(), static_cast<std::size_t>(st.st_size),
		      data, read_error)) {
		error = "failed to read config file '" + path.string() +
			"': " + read_error;
		return false;
	}
	return true;
}

bool sync_parent_directory(const std::filesystem::path &path,
			   std::string &error)
{
	const auto parent = path.parent_path();
	if (parent.empty()) {
		return true;
	}

	const int fd =
		::open(parent.c_str(), O_RDONLY | O_DIRECTORY | O_CLOEXEC);
	if (fd < 0) {
		error = "failed to open config directory '" + parent.string() +
			"': " + std::strerror(errno);
		return false;
	}
	ScopedFd scoped_fd(fd);
	if (::fsync(scoped_fd.get()) != 0) {
		error = "failed to sync config directory '" + parent.string() +
			"': " + std::strerror(errno);
		return false;
	}
	return true;
}

bool write_all(int fd, std::string_view data, std::string &error)
{
	const char *ptr = data.data();
	std::size_t remaining = data.size();
	while (remaining > 0U) {
		const ssize_t written = ::write(fd, ptr, remaining);
		if (written < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (written == 0) {
			error = "short write";
			return false;
		}
		ptr += written;
		remaining -= static_cast<std::size_t>(written);
	}
	return true;
}

std::filesystem::path tmp_path_for(const std::filesystem::path &path,
				   int attempt)
{
	auto tmp_path = path;
	tmp_path += ".tmp." + std::to_string(::getpid()) + "." +
		    std::to_string(attempt);
	return tmp_path;
}

bool path_is_absent(const std::filesystem::path &path)
{
	std::error_code ec;
	const auto status = std::filesystem::symlink_status(path, ec);
	return status.type() == std::filesystem::file_type::not_found;
}

bool write_table(const std::filesystem::path &path, const toml::table &table,
		 std::string &error)
{
	std::error_code ec;
	const auto parent = path.parent_path();
	if (!parent.empty()) {
		std::filesystem::create_directories(parent, ec);
		if (ec) {
			error = "failed to create config directory '" +
				parent.string() + "': " + ec.message();
			return false;
		}
	}
	if (!validate_parent_directory(path, error)) {
		return false;
	}

	std::ostringstream payload;
	payload << table << '\n';

	for (int attempt = 0; attempt < 100; ++attempt) {
		const auto tmp_path = tmp_path_for(path, attempt);
		const int fd = ::open(tmp_path.c_str(),
				      O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC |
					      O_NOFOLLOW,
				      kPrivateFileMode);
		if (fd < 0) {
			if (errno == EEXIST) {
				continue;
			}
			error = "failed to create temporary config file '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			return false;
		}

		ScopedFd scoped_fd(fd);
		std::string write_error;
		const std::string data = payload.str();
		if (!write_all(scoped_fd.get(), data, write_error)) {
			error = "failed to write temporary config file '" +
				tmp_path.string() + "': " + write_error;
			std::error_code remove_ec;
			std::filesystem::remove(tmp_path, remove_ec);
			return false;
		}
		if (::fsync(scoped_fd.get()) != 0) {
			error = "failed to sync temporary config file '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			std::error_code remove_ec;
			std::filesystem::remove(tmp_path, remove_ec);
			return false;
		}
		if (::close(scoped_fd.release()) != 0) {
			error = "failed to close temporary config file '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			std::error_code remove_ec;
			std::filesystem::remove(tmp_path, remove_ec);
			return false;
		}

		std::filesystem::rename(tmp_path, path, ec);
		if (ec) {
			error = "failed to replace config file '" +
				path.string() + "': " + ec.message();
			std::error_code remove_ec;
			std::filesystem::remove(tmp_path, remove_ec);
			return false;
		}

		return sync_parent_directory(path, error);
	}

	error = "failed to choose a temporary config path for '" +
		path.string() + "'";
	return false;
}

toml::table to_toml_table(const AppConfig &config)
{
	return toml::table {
		{ std::string(kDisplay),
		  toml::table {
			  { std::string(kFramebufferDevice),
			    config.display.framebuffer_device },
		  } },
		{ std::string(kTouch),
		  toml::table {
			  { std::string(kDevice), config.touch.device },
			  { std::string(kDisabled), config.touch.disabled },
		  } },
		{ std::string(kHttpApi),
		  toml::table {
			  { std::string(kEnabled), config.http_api.enabled },
			  { std::string(kBindHost), config.http_api.bind_host },
			  { std::string(kPort),
			    static_cast<int64_t>(config.http_api.port) },
			  { std::string(kToken), config.http_api.token },
		  } },
		{ std::string(kRuntime),
		  toml::table {
			  { std::string(kRuntimeDebugLogging),
			    config.runtime.debug_logging },
			  { std::string(kRuntimeTouchClickMarker),
			    config.runtime.touch_click_marker },
			  { std::string(kRuntimeTouchCalibrate),
			    config.runtime.touch_calibrate },
			  { std::string(kRuntimeTouchPaintTest),
			    config.runtime.touch_paint_test },
			  } },
		{ std::string(kScreenshot),
		  toml::table {
			  { std::string(kEnabled), config.screenshot.enabled },
			  { std::string(kScreenshotSignal),
			    std::string(screenshot_control::signal_name(
				    config.screenshot.signal)) },
			  { std::string(kScreenshotOutputDirectory),
			    config.screenshot.output_directory },
		  } },
		{ std::string(kUi),
		  toml::table {
			  { std::string(kEnableAutoRotation),
			    config.ui.enable_auto_rotation },
			  { std::string(kManualDisplayRotation),
			    config.ui.manual_display_rotation },
			  { std::string(kExperimentalSettingsSubpages),
			    config.ui.experimental_settings_subpages },
			  { std::string(kEnableSettingsOptimisticActions),
			    config.ui.enable_settings_optimistic_actions },
			  { std::string(kPageTransitionMode),
			    std::string(ui_page_transition_mode_name(
				    config.ui.page_transition_mode)) },
			  { std::string(kWifiBootReconnectTimeoutMs),
			    static_cast<int64_t>(
				    config.ui.wifi_boot_reconnect_timeout_ms) },
			  { std::string(kSettingsStateBootTimeoutMs),
			    static_cast<int64_t>(
				    config.ui.settings_state_boot_timeout_ms) },
			  { std::string(kWifiLoadingProgress),
			    config.ui.wifi_loading_progress },
			  } },
		{ std::string(kFonts),
		  toml::table {
			  { std::string(kCjkRegion),
			    std::string(cjk_region_name(config.fonts.cjk_region)) },
			  { std::string(kSsidImageCacheKb),
			    static_cast<int64_t>(config.fonts.ssid_image_cache_kb) },
		  } },
		{ std::string(kExperimental),
		  toml::table {
			  { std::string(kExperimentalReserved), true },
			  { std::string(kExperimentalSsidScroll),
			    std::string(ssid_scroll_mode_name(
				    config.experimental.ssid_scroll)) },
		  } },
		{ std::string(kNanokvm),
		  toml::table {
			  { std::string(kEnableRuntimeBackend),
			    config.nanokvm.enable_runtime_backend },
			  { std::string(kPreserveLastKnownGoodState),
			    config.nanokvm.preserve_last_known_good_state },
			  { std::string(kServerBaseUrl),
			    config.nanokvm.server_base_url },
			  { std::string(kServerVerifyTls),
			    config.nanokvm.server_verify_tls },
			  { std::string(kWifiScript),
			    config.nanokvm.wifi_script },
			  { std::string(kUsbGadgetRestartScript),
			    config.nanokvm.usb_gadget_restart_script },
			  { std::string(kBacklightRoot),
			    config.nanokvm.backlight_root },
			  { std::string(kEnableHighRiskActions),
			    config.nanokvm.enable_high_risk_actions },
			  { std::string(kVideoEnabled),
			    config.nanokvm.video_enabled },
			  { std::string(kHidEnabled),
			    config.nanokvm.hid_enabled },
			  { std::string(kBacklightPercent),
			    static_cast<int64_t>(
				    config.nanokvm.backlight_percent) },
			  { std::string(kBacklightInvert),
			    config.nanokvm.backlight_invert },
			  { std::string(kAutoSleepEnabled),
			    config.nanokvm.auto_sleep_enabled },
		  } },
		{ std::string(kLauncher),
		  toml::table {
			  { std::string(kEnabled), config.launcher.enabled },
			  { std::string(kLauncherPythonBin),
			    config.launcher.python_bin },
			  { std::string(kLauncherExitZonePx),
			    static_cast<int64_t>(
				    config.launcher.exit_gesture_zone_px) },
			  { std::string(kLauncherExitHoldMs),
			    static_cast<int64_t>(
				    config.launcher.exit_gesture_hold_ms) },
			  { std::string(kLauncherChildTermTimeoutMs),
			    static_cast<int64_t>(
				    config.launcher.child_term_timeout_ms) },
		  } },
	};
}

toml::table &ensure_table(toml::table &root, std::string_view section,
			  AppConfigLoadResult &result)
{
	if (auto *existing = root[section].as_table()) {
		return *existing;
	}

	if (root.contains(section)) {
		mark_repaired(result, "Invalid table '" + std::string(section) +
					      "', restored defaults");
	} else {
		mark_repaired(result, "Missing table '" + std::string(section) +
					      "', restored defaults");
	}
	root.insert_or_assign(std::string(section), toml::table {});
	return *root[section].as_table();
}

toml::table &ensure_optional_table(toml::table &root, std::string_view section,
				   const toml::table &fallback,
				   AppConfigLoadResult &result)
{
	if (auto *existing = root[section].as_table()) {
		return *existing;
	}

	const bool had_invalid_value = root.contains(section);
	root.insert_or_assign(std::string(section), fallback);
	if (had_invalid_value) {
		mark_repaired(result, "Invalid table '" + std::string(section) +
					      "', restored defaults");
	}
	return *root[section].as_table();
}

std::string read_string(toml::table &table, std::string_view section,
			std::string_view key, std::string_view fallback,
			bool (*validator)(std::string_view),
			AppConfigLoadResult &result)
{
	const auto field_path = path_for(section, key);
	if (auto value = table[key].value<std::string>()) {
		if (validator == nullptr || validator(*value)) {
			table.insert_or_assign(std::string(key), *value);
			return *value;
		}
	}

	table.insert_or_assign(std::string(key), std::string(fallback));
	mark_repaired(result, "Invalid or missing '" + field_path +
				      "', restored default");
	return std::string(fallback);
}

std::string read_string_with_legacy_key(
	toml::table &table, std::string_view section, std::string_view key,
	std::string_view legacy_key, std::string_view fallback,
	bool (*validator)(std::string_view), AppConfigLoadResult &result)
{
	const auto field_path = path_for(section, key);
	if (auto value = table[key].value<std::string>()) {
		if (validator == nullptr || validator(*value)) {
			table.insert_or_assign(std::string(key), *value);
			return *value;
		}
		table.insert_or_assign(std::string(key), std::string(fallback));
		mark_repaired(result,
			      "Invalid '" + field_path + "', restored default");
		return std::string(fallback);
	}

	if (auto legacy_value = table[legacy_key].value<std::string>()) {
		if (validator == nullptr || validator(*legacy_value)) {
			table.insert_or_assign(std::string(key), *legacy_value);
			table.erase(std::string(legacy_key));
			mark_repaired(result,
				      "Missing '" + field_path +
					      "', migrated legacy '" +
					      path_for(section, legacy_key) +
					      "'");
			return *legacy_value;
		}
	}

	table.insert_or_assign(std::string(key), std::string(fallback));
	mark_repaired(result, "Invalid or missing '" + field_path +
				      "', restored default");
	return std::string(fallback);
}

bool read_bool(toml::table &table, std::string_view section,
	       std::string_view key, bool fallback, AppConfigLoadResult &result)
{
	if (auto value = table[key].value_exact<bool>()) {
		table.insert_or_assign(std::string(key), *value);
		return *value;
	}

	table.insert_or_assign(std::string(key), fallback);
	mark_repaired(result, "Invalid or missing '" + path_for(section, key) +
				      "', restored default");
	return fallback;
}

bool read_bool_with_legacy_key(toml::table &table, std::string_view section,
			       std::string_view key,
			       std::string_view legacy_key, bool fallback,
			       AppConfigLoadResult &result)
{
	const auto field_path = path_for(section, key);
	const auto legacy_path = path_for(section, legacy_key);
	const bool had_legacy_key = table.contains(legacy_key);

	if (auto value = table[key].value_exact<bool>()) {
		table.insert_or_assign(std::string(key), *value);
		if (had_legacy_key) {
			table.erase(std::string(legacy_key));
			mark_repaired(result,
				      "Removed legacy '" + legacy_path + "'");
		}
		return *value;
	}

	if (table.contains(key)) {
		table.insert_or_assign(std::string(key), fallback);
		if (had_legacy_key) {
			table.erase(std::string(legacy_key));
		}
		mark_repaired(result,
			      "Invalid '" + field_path + "', restored default");
		return fallback;
	}

	if (auto legacy_value = table[legacy_key].value_exact<bool>()) {
		table.insert_or_assign(std::string(key), *legacy_value);
		table.erase(std::string(legacy_key));
		mark_repaired(result, "Missing '" + field_path +
					      "', migrated legacy '" +
					      legacy_path + "'");
		return *legacy_value;
	}

	table.insert_or_assign(std::string(key), fallback);
	if (had_legacy_key) {
		table.erase(std::string(legacy_key));
	}
	mark_repaired(result, "Invalid or missing '" + field_path +
				      "', restored default");
	return fallback;
}

SsidScrollMode read_ssid_scroll_mode(toml::table &table,
				      AppConfigLoadResult &result)
{
	const auto field_path = path_for(kExperimental, kExperimentalSsidScroll);
	const auto legacy_path = path_for(
		kExperimental, kLegacyExperimentalEnableSsidScroll);
	const bool has_current_key = table.contains(kExperimentalSsidScroll);
	const bool had_legacy_key =
		table.contains(kLegacyExperimentalEnableSsidScroll);

	if (auto value = table[kExperimentalSsidScroll].value<std::string>()) {
		if (const auto mode = parse_ssid_scroll_mode(*value)) {
			table.insert_or_assign(
				std::string(kExperimentalSsidScroll),
				std::string(ssid_scroll_mode_name(*mode)));
			if (had_legacy_key) {
				table.erase(
					std::string(kLegacyExperimentalEnableSsidScroll));
				mark_repaired(result,
					      "Removed legacy '" + legacy_path + "'");
			}
			return *mode;
		}

		table.insert_or_assign(
			std::string(kExperimentalSsidScroll),
			std::string(app_config_defaults::kExperimentalSsidScroll));
		if (had_legacy_key) {
			table.erase(std::string(kLegacyExperimentalEnableSsidScroll));
		}
		mark_repaired(result,
			      "Invalid '" + field_path + "', restored default");
		return SsidScrollMode::Auto;
	}

	if (has_current_key) {
		table.insert_or_assign(
			std::string(kExperimentalSsidScroll),
			std::string(app_config_defaults::kExperimentalSsidScroll));
		if (had_legacy_key) {
			table.erase(std::string(kLegacyExperimentalEnableSsidScroll));
		}
		mark_repaired(result,
			      "Invalid '" + field_path + "', restored default");
		return SsidScrollMode::Auto;
	}

	if (auto legacy_value =
		    table[kLegacyExperimentalEnableSsidScroll].value_exact<bool>()) {
		const auto mode = *legacy_value ? SsidScrollMode::On :
						 SsidScrollMode::Off;
		table.insert_or_assign(std::string(kExperimentalSsidScroll),
				       std::string(ssid_scroll_mode_name(mode)));
		table.erase(std::string(kLegacyExperimentalEnableSsidScroll));
		mark_repaired(result, "Missing '" + field_path +
				      "', migrated legacy '" + legacy_path + "'");
		return mode;
	}

	table.insert_or_assign(
		std::string(kExperimentalSsidScroll),
		std::string(app_config_defaults::kExperimentalSsidScroll));
	if (had_legacy_key) {
		table.erase(std::string(kLegacyExperimentalEnableSsidScroll));
	}
	mark_repaired(result, "Invalid or missing '" + field_path +
			      "', restored default");
	return SsidScrollMode::Auto;
}

bool read_wifi_loading_progress(toml::table &ui_table,
			       toml::table &experimental_table,
			       AppConfigLoadResult &result)
{
	const bool fallback = app_config_defaults::kWifiLoadingProgress;
	const auto field_path = path_for(kUi, kWifiLoadingProgress);
	const auto legacy_path =
		path_for(kExperimental, kExperimentalWifiLoadingMode);
	const bool had_legacy =
		experimental_table.contains(kExperimentalWifiLoadingMode);

	if (auto value = ui_table[kWifiLoadingProgress].value_exact<bool>()) {
		ui_table.insert_or_assign(std::string(kWifiLoadingProgress),
					  *value);
		if (had_legacy) {
			experimental_table.erase(
				std::string(kExperimentalWifiLoadingMode));
			mark_repaired(result,
				      "Removed legacy '" + legacy_path + "'");
		}
		return *value;
	}

	if (ui_table.contains(kWifiLoadingProgress)) {
		ui_table.insert_or_assign(std::string(kWifiLoadingProgress),
					  fallback);
		if (had_legacy) {
			experimental_table.erase(
				std::string(kExperimentalWifiLoadingMode));
		}
		mark_repaired(result,
			      "Invalid '" + field_path + "', restored default");
		return fallback;
	}

	if (auto legacy = experimental_table[kExperimentalWifiLoadingMode]
				 .value<std::string>()) {
		const bool migrated = (*legacy == "progress");
		ui_table.insert_or_assign(std::string(kWifiLoadingProgress),
					  migrated);
		experimental_table.erase(
			std::string(kExperimentalWifiLoadingMode));
		mark_repaired(result, "Missing '" + field_path +
				      "', migrated legacy '" +
				      legacy_path + "'");
		return migrated;
	}

	ui_table.insert_or_assign(std::string(kWifiLoadingProgress), fallback);
	if (had_legacy) {
		experimental_table.erase(
			std::string(kExperimentalWifiLoadingMode));
	}
	mark_repaired(result, "Invalid or missing '" + field_path +
			      "', restored default");
	return fallback;
}

std::optional<toml::source_position> source_position_for(toml::table &table,
							 std::string_view key)
{
	auto view = table[key];
	if (auto *node = view.node()) {
		const auto position = node->source().begin;
		if (position) {
			return position;
		}
	}
	return std::nullopt;
}

RuntimeTouchOption read_runtime_touch_option(toml::table &table,
					     std::string_view key,
					     AppConfigLoadResult &result)
{
	RuntimeTouchOption option;
	option.key = key;
	option.source = source_position_for(table, key);
	option.value = read_bool(table, kRuntime, key, false, result);
	return option;
}

bool option_is_after(const RuntimeTouchOption &candidate,
		     const RuntimeTouchOption &current)
{
	if (candidate.source.has_value() && current.source.has_value()) {
		return *candidate.source > *current.source;
	}
	return candidate.source.has_value() && !current.source.has_value();
}

std::string runtime_option_path(std::string_view key)
{
	return path_for(kRuntime, key);
}

void apply_runtime_touch_mutex(RuntimeOptionsConfig &runtime,
			       const std::array<RuntimeTouchOption, 3U> &options,
			       AppConfigLoadResult &result)
{
	std::optional<RuntimeTouchOption> active;
	std::vector<std::string_view> ignored;

	for (const auto &option : options) {
		if (!option.value) {
			continue;
		}
		if (!active.has_value() || option_is_after(option, *active)) {
			if (active.has_value()) {
				ignored.push_back(active->key);
			}
			active = option;
		} else {
			ignored.push_back(option.key);
		}
	}

	runtime.touch_click_marker = false;
	runtime.touch_calibrate = false;
	runtime.touch_paint_test = false;
	if (!active.has_value()) {
		return;
	}

	if (active->key == kRuntimeTouchClickMarker) {
		runtime.touch_click_marker = true;
	} else if (active->key == kRuntimeTouchCalibrate) {
		runtime.touch_calibrate = true;
	} else if (active->key == kRuntimeTouchPaintTest) {
		runtime.touch_paint_test = true;
	}

	if (ignored.empty()) {
		return;
	}

	std::string message =
		"Multiple runtime touch options enabled in config; using '" +
		runtime_option_path(active->key) + "' and ignoring '" +
		runtime_option_path(ignored.front()) + "'";
	for (std::size_t i = 1; i < ignored.size(); ++i) {
		message += ", '";
		message += runtime_option_path(ignored[i]);
		message += "'";
	}
	mark_warning(result, std::move(message));
}

int read_int_range(toml::table &table, std::string_view section,
		   std::string_view key, int fallback, int min_value,
		   int max_value, AppConfigLoadResult &result)
{
	if (auto value = table[key].value_exact<int64_t>()) {
		if (*value >= min_value && *value <= max_value) {
			const int parsed = static_cast<int>(*value);
			table.insert_or_assign(std::string(key),
					       static_cast<int64_t>(parsed));
			return parsed;
		}
	}

	table.insert_or_assign(std::string(key),
			       static_cast<int64_t>(fallback));
	mark_repaired(result, "Invalid or missing '" + path_for(section, key) +
				      "', restored default");
	return fallback;
}

float read_float_range(toml::table &table, std::string_view section,
		       std::string_view key, float fallback, float min_value,
		       float max_value, AppConfigLoadResult &result)
{
	if (auto value = table[key].value<double>()) {
		if (*value >= min_value && *value <= max_value) {
			const float parsed = static_cast<float>(*value);
			table.insert_or_assign(std::string(key),
					       static_cast<double>(parsed));
			return parsed;
		}
	}

	table.insert_or_assign(std::string(key), static_cast<double>(fallback));
	mark_repaired(result, "Invalid or missing '" + path_for(section, key) +
				      "', restored default");
	return fallback;
}

std::filesystem::path choose_backup_path(const std::filesystem::path &path)
{
	auto candidate = path;
	candidate += ".bak";
	if (path_is_absent(candidate)) {
		return candidate;
	}

	for (int i = 1; i < 1000; ++i) {
		candidate = path;
		candidate += ".bak." + std::to_string(i);
		if (path_is_absent(candidate)) {
			return candidate;
		}
	}

	candidate = path;
	candidate += ".bak.latest";
	return candidate;
}

void repair_http_cross_field(toml::table &http_table, AppConfig &config,
			     AppConfigLoadResult &result)
{
	lvgl_main::http_api::ServerConfig server_config {
		.bind_host = config.http_api.bind_host,
		.port = config.http_api.port,
		.token = config.http_api.token,
	};
	const auto validation =
		lvgl_main::http_api::validate_server_config(server_config);
	if (config.http_api.enabled && config.http_api.port > 0 &&
	    !validation.allowed) {
		config.http_api.bind_host =
			app_config_defaults::kHttpApiBindHost;
		config.http_api.token = app_config_defaults::kHttpApiToken;
		http_table.insert_or_assign(std::string(kBindHost),
					    config.http_api.bind_host);
		http_table.insert_or_assign(std::string(kToken),
					    config.http_api.token);
		mark_repaired(
			result,
			"Unsafe or invalid HTTP API bind/token combination, "
			"restored loopback defaults");
	}
}

void remove_obsolete_display_gamma_keys(toml::table &display_table,
					AppConfigLoadResult &result)
{
	for (const auto key : kObsoleteDisplayGammaKeys) {
		if (!display_table.contains(key)) {
			continue;
		}
		display_table.erase(std::string(key));
		mark_repaired(result, "Removed obsolete '" +
					      path_for(kDisplay, key) + "'");
	}
}

void remove_obsolete_ui_keys(toml::table &ui_table, AppConfigLoadResult &result)
{
	for (const auto key : kObsoleteUiKeys) {
		if (!ui_table.contains(key)) {
			continue;
		}
		ui_table.erase(std::string(key));
		mark_repaired(result,
			      "Removed obsolete '" + path_for(kUi, key) + "'");
	}
}

void remove_obsolete_launcher_keys(toml::table &launcher_table,
				   AppConfigLoadResult &result)
{
	for (const auto key : kObsoleteLauncherKeys) {
		if (!launcher_table.contains(key)) {
			continue;
		}
		launcher_table.erase(std::string(key));
		mark_repaired(result, "Removed obsolete '" +
					      path_for(kLauncher, key) + "'");
	}
}

void remove_obsolete_experimental_keys(toml::table &experimental_table,
				       AppConfigLoadResult &result)
{
	for (const auto key : kObsoleteExperimentalKeys) {
		if (!experimental_table.contains(key)) {
			continue;
		}
		experimental_table.erase(std::string(key));
		mark_repaired(result, "Removed obsolete '" +
					      path_for(kExperimental, key) + "'");
	}
}

void remove_obsolete_nanokvm_keys(toml::table &nanokvm_table,
				AppConfigLoadResult &result)
{
	for (const auto key : kObsoleteNanokvmKeys) {
		if (!nanokvm_table.contains(key)) {
			continue;
		}
		nanokvm_table.erase(std::string(key));
		mark_repaired(result, "Removed obsolete '" +
				      path_for(kNanokvm, key) + "'");
	}
}

AppConfig read_and_repair(toml::table &root, AppConfigLoadResult &result)
{
	AppConfig config = default_app_config();

	auto &display_table = ensure_table(root, kDisplay, result);
	auto &touch_table = ensure_table(root, kTouch, result);
	auto &http_table = ensure_table(root, kHttpApi, result);
	auto &runtime_table = ensure_table(root, kRuntime, result);
	auto &screenshot_table = ensure_table(root, kScreenshot, result);
	auto &ui_table = ensure_table(root, kUi, result);
	auto &fonts_table = ensure_table(root, kFonts, result);
	auto &experimental_table = ensure_table(root, kExperimental, result);
	auto &nanokvm_table = ensure_optional_table(
		root, kNanokvm, *to_toml_table(config)[kNanokvm].as_table(),
		result);
	remove_obsolete_nanokvm_keys(nanokvm_table, result);

	config.display.framebuffer_device =
		read_string(display_table, kDisplay, kFramebufferDevice,
			    app_config_defaults::kFramebufferDevice,
			    is_valid_fbdev_path, result);
	remove_obsolete_display_gamma_keys(display_table, result);

	config.touch.device = read_string(touch_table, kTouch, kDevice,
					  app_config_defaults::kTouchDevice,
					  is_valid_input_event_path, result);
	config.touch.disabled = read_bool(touch_table, kTouch, kDisabled,
					  app_config_defaults::kTouchDisabled,
					  result);

	config.http_api.enabled =
		read_bool(http_table, kHttpApi, kEnabled,
			  app_config_defaults::kHttpApiEnabled, result);
	config.http_api.bind_host =
		read_string(http_table, kHttpApi, kBindHost,
			    app_config_defaults::kHttpApiBindHost,
			    is_non_empty_string, result);
	config.http_api.port = read_int_range(
		http_table, kHttpApi, kPort, app_config_defaults::kHttpApiPort,
		app_config_defaults::kHttpApiPortMin,
		app_config_defaults::kHttpApiPortMax, result);
	config.http_api.token = read_string(http_table, kHttpApi, kToken,
					    app_config_defaults::kHttpApiToken,
					    nullptr, result);
	repair_http_cross_field(http_table, config, result);

	config.runtime.debug_logging =
		read_bool(runtime_table, kRuntime, kRuntimeDebugLogging,
			  app_config_defaults::kRuntimeDebugLogging, result);
	const std::array runtime_touch_options {
		read_runtime_touch_option(runtime_table,
					  kRuntimeTouchClickMarker, result),
		read_runtime_touch_option(runtime_table, kRuntimeTouchCalibrate,
					  result),
		read_runtime_touch_option(runtime_table, kRuntimeTouchPaintTest,
					  result),
	};
	config.runtime.touch_click_marker = runtime_touch_options[0].value;
	config.runtime.touch_calibrate = runtime_touch_options[1].value;
	config.runtime.touch_paint_test = runtime_touch_options[2].value;
	result.effective_runtime = config.runtime;
	apply_runtime_touch_mutex(result.effective_runtime,
				  runtime_touch_options, result);

	config.screenshot.enabled =
		read_bool(screenshot_table, kScreenshot, kEnabled,
			  app_config_defaults::kScreenshotEnabled, result);
	config.screenshot.signal = screenshot_control::parse_signal(read_string(
		screenshot_table, kScreenshot, kScreenshotSignal,
		app_config_defaults::kScreenshotSignal, is_valid_screenshot_signal,
		result))
					 .value_or(ScreenshotSignal::User1);
	config.screenshot.output_directory = read_string(
		screenshot_table, kScreenshot, kScreenshotOutputDirectory,
		app_config_defaults::kScreenshotOutputDirectory,
		is_valid_screenshot_output_directory, result);

	remove_obsolete_ui_keys(ui_table, result);
	config.ui.enable_auto_rotation = read_bool_with_legacy_key(
		ui_table, kUi, kEnableAutoRotation, kLegacyEnable180Rotation,
		app_config_defaults::kEnableAutoRotation, result);
	config.ui.manual_display_rotation =
		read_bool(ui_table, kUi, kManualDisplayRotation,
			  app_config_defaults::kManualDisplayRotation, result);
	config.ui.experimental_settings_subpages = read_bool(
		ui_table, kUi, kExperimentalSettingsSubpages,
		app_config_defaults::kExperimentalSettingsSubpages, result);
	config.ui.enable_settings_optimistic_actions = read_bool(
		ui_table, kUi, kEnableSettingsOptimisticActions,
		app_config_defaults::kEnableSettingsOptimisticActions, result);
	config.ui.page_transition_mode = parse_ui_page_transition_mode(
		read_string(ui_table, kUi, kPageTransitionMode,
			    app_config_defaults::kUiPageTransitionMode,
			    is_valid_ui_page_transition_mode, result));
	config.ui.wifi_boot_reconnect_timeout_ms = read_int_range(
		ui_table, kUi, kWifiBootReconnectTimeoutMs,
		app_config_defaults::kWifiBootReconnectTimeoutMs,
		app_config_defaults::kWifiBootReconnectTimeoutMsMin,
		app_config_defaults::kWifiBootReconnectTimeoutMsMax, result);
	config.ui.settings_state_boot_timeout_ms = read_int_range(
		ui_table, kUi, kSettingsStateBootTimeoutMs,
		app_config_defaults::kSettingsStateBootTimeoutMs,
		app_config_defaults::kSettingsStateBootTimeoutMsMin,
		app_config_defaults::kSettingsStateBootTimeoutMsMax, result);
	config.ui.wifi_loading_progress = read_wifi_loading_progress(
		ui_table, experimental_table, result);
	config.fonts.cjk_region = parse_cjk_region(read_string(
		fonts_table, kFonts, kCjkRegion, app_config_defaults::kFontsCjkRegion,
		is_valid_cjk_region, result))
					 .value_or(CjkRegion::SimplifiedChinese);
	config.fonts.ssid_image_cache_kb = read_int_range(
		fonts_table, kFonts, kSsidImageCacheKb,
		app_config_defaults::kFontsSsidImageCacheKb,
		app_config_defaults::kFontsSsidImageCacheKbMin,
		app_config_defaults::kFontsSsidImageCacheKbMax, result);
	remove_obsolete_experimental_keys(experimental_table, result);
	config.experimental.ssid_scroll =
		read_ssid_scroll_mode(experimental_table, result);
	const auto experimental_reserved =
		experimental_table[kExperimentalReserved].value_exact<bool>();
	if (!experimental_reserved.has_value() || !*experimental_reserved) {
		experimental_table.insert_or_assign(
			std::string(kExperimentalReserved), true);
		mark_repaired(
			result,
			"Invalid or missing '" +
				path_for(kExperimental, kExperimentalReserved) +
				"', restored default");
	}

	config.nanokvm.enable_runtime_backend = read_bool(
		nanokvm_table, kNanokvm, kEnableRuntimeBackend,
		app_config_defaults::kNanokvmRuntimeBackendEnabled, result);
	config.nanokvm.preserve_last_known_good_state = read_bool(
		nanokvm_table, kNanokvm, kPreserveLastKnownGoodState,
		app_config_defaults::kNanokvmPreserveLastKnownGoodState, result);
	config.nanokvm.server_base_url =
		read_string(nanokvm_table, kNanokvm, kServerBaseUrl,
			    app_config_defaults::kNanokvmServerBaseUrl,
			    is_valid_server_base_url, result);
	config.nanokvm.server_verify_tls =
		read_bool(nanokvm_table, kNanokvm, kServerVerifyTls,
			  app_config_defaults::kNanokvmServerVerifyTls, result);
	config.nanokvm.wifi_script =
		read_string(nanokvm_table, kNanokvm, kWifiScript,
			    app_config_defaults::kNanokvmWifiScript,
			    is_valid_absolute_path, result);
	config.nanokvm.usb_gadget_restart_script = read_string_with_legacy_key(
		nanokvm_table, kNanokvm, kUsbGadgetRestartScript,
		kLegacyUsbScript,
		app_config_defaults::kNanokvmUsbGadgetRestartScript,
		is_valid_absolute_path, result);
	if (config.nanokvm.usb_gadget_restart_script ==
	    app_config_defaults::kNanokvmLegacyUsbGadgetRestartScript) {
		config.nanokvm.usb_gadget_restart_script =
			app_config_defaults::kNanokvmUsbGadgetRestartScript;
		nanokvm_table.insert_or_assign(
			std::string(kUsbGadgetRestartScript),
			config.nanokvm.usb_gadget_restart_script);
		mark_repaired(
			result,
			"Legacy default '" +
				path_for(kNanokvm, kUsbGadgetRestartScript) +
				"' migrated to '" +
				std::string(
					app_config_defaults::
						kNanokvmUsbGadgetRestartScript) +
				"'");
	}
	config.nanokvm.backlight_root =
		read_string(nanokvm_table, kNanokvm, kBacklightRoot,
			    app_config_defaults::kNanokvmBacklightRoot,
			    is_valid_absolute_path, result);
	config.nanokvm.enable_high_risk_actions = read_bool(
		nanokvm_table, kNanokvm, kEnableHighRiskActions,
		app_config_defaults::kNanokvmHighRiskActionsEnabled, result);
	config.nanokvm.video_enabled =
		read_bool(nanokvm_table, kNanokvm, kVideoEnabled,
			  app_config_defaults::kNanokvmVideoEnabled, result);
	config.nanokvm.hid_enabled =
		read_bool(nanokvm_table, kNanokvm, kHidEnabled,
			  app_config_defaults::kNanokvmHidEnabled, result);
	config.nanokvm.backlight_percent = read_int_range(
		nanokvm_table, kNanokvm, kBacklightPercent,
		app_config_defaults::kNanokvmBacklightPercent,
		app_config_defaults::kNanokvmBacklightPercentMin,
		app_config_defaults::kNanokvmBacklightPercentMax, result);
	config.nanokvm.backlight_invert =
		read_bool(nanokvm_table, kNanokvm, kBacklightInvert,
			  app_config_defaults::kNanokvmBacklightInvert, result);
	config.nanokvm.auto_sleep_enabled = read_bool(
		nanokvm_table, kNanokvm, kAutoSleepEnabled,
		app_config_defaults::kNanokvmAutoSleepEnabled, result);

	auto &launcher_table = ensure_optional_table(
		root, kLauncher, *to_toml_table(config)[kLauncher].as_table(),
		result);
	remove_obsolete_launcher_keys(launcher_table, result);
	config.launcher.enabled =
		read_bool(launcher_table, kLauncher, kEnabled,
			  app_config_defaults::kLauncherEnabled, result);
	config.launcher.python_bin =
		read_string(launcher_table, kLauncher, kLauncherPythonBin,
			    app_config_defaults::kLauncherPythonBin,
			    is_non_empty_string, result);
	config.launcher.exit_gesture_zone_px = read_int_range(
		launcher_table, kLauncher, kLauncherExitZonePx,
		app_config_defaults::kLauncherExitZonePx,
		app_config_defaults::kLauncherExitZonePxMin,
		app_config_defaults::kLauncherExitZonePxMax, result);
	config.launcher.exit_gesture_hold_ms = read_int_range(
		launcher_table, kLauncher, kLauncherExitHoldMs,
		app_config_defaults::kLauncherExitHoldMs,
		app_config_defaults::kLauncherExitHoldMsMin,
		app_config_defaults::kLauncherExitHoldMsMax, result);
	config.launcher.child_term_timeout_ms = read_int_range(
		launcher_table, kLauncher, kLauncherChildTermTimeoutMs,
		app_config_defaults::kLauncherChildTermTimeoutMs,
		app_config_defaults::kLauncherChildTermTimeoutMsMin,
		app_config_defaults::kLauncherChildTermTimeoutMsMax, result);

	return config;
}

}

AppConfig default_app_config()
{
	return {};
}

std::string_view cjk_region_name(CjkRegion region) noexcept
{
	switch (region) {
	case CjkRegion::Japan:
		return "jp";
	case CjkRegion::Korea:
		return "kr";
	case CjkRegion::SimplifiedChinese:
		return "sc";
	case CjkRegion::TraditionalChineseTaiwan:
		return "tc";
	case CjkRegion::TraditionalChineseHongKong:
		return "hk";
	}
	return app_config_defaults::kFontsCjkRegion;
}

std::optional<CjkRegion> parse_cjk_region(std::string_view value) noexcept
{
	if (value == "jp") {
		return CjkRegion::Japan;
	}
	if (value == "kr") {
		return CjkRegion::Korea;
	}
	if (value == "sc") {
		return CjkRegion::SimplifiedChinese;
	}
	if (value == "tc") {
		return CjkRegion::TraditionalChineseTaiwan;
	}
	if (value == "hk") {
		return CjkRegion::TraditionalChineseHongKong;
	}
	return std::nullopt;
}

std::string_view ssid_scroll_mode_name(SsidScrollMode mode) noexcept
{
	switch (mode) {
	case SsidScrollMode::On:
		return "on";
	case SsidScrollMode::Off:
		return "off";
	case SsidScrollMode::Auto:
		return app_config_defaults::kExperimentalSsidScroll;
	}
	return app_config_defaults::kExperimentalSsidScroll;
}

std::optional<SsidScrollMode>
parse_ssid_scroll_mode(std::string_view value) noexcept
{
	if (value == "on") {
		return SsidScrollMode::On;
	}
	if (value == "off") {
		return SsidScrollMode::Off;
	}
	if (value == "auto") {
		return SsidScrollMode::Auto;
	}
	return std::nullopt;
}

bool resolve_ssid_scroll_enabled(SsidScrollMode mode,
				 bool runtime_tiny_ttf) noexcept
{
	switch (mode) {
	case SsidScrollMode::On:
		return true;
	case SsidScrollMode::Off:
		return false;
	case SsidScrollMode::Auto:
		return runtime_tiny_ttf;
	}
	return false;
}

std::string render_app_config_toml(const AppConfig &config)
{
	std::ostringstream out;
	out << to_toml_table(config) << '\n';
	return out.str();
}

bool save_app_config(const AppConfig &config, std::string &error,
		     const std::filesystem::path &path)
{
	return write_table(path, to_toml_table(config), error);
}

AppConfigLoadResult load_or_repair_app_config(const std::filesystem::path &path)
{
	AppConfigLoadResult result;
	result.path = path;

	std::error_code ec;
	const auto status = std::filesystem::symlink_status(path, ec);
	if (status.type() == std::filesystem::file_type::not_found) {
		const auto table = to_toml_table(result.config);
		if (!write_table(path, table, result.error)) {
			return result;
		}
		result.ok = true;
		result.created = true;
		result.messages.push_back("Created default config file '" +
					  path.string() + "'");
		return result;
	}
	if (ec) {
		result.error = "failed to inspect config file '" +
			       path.string() + "': " + ec.message();
		return result;
	}
	if (std::filesystem::is_symlink(status) ||
	    !std::filesystem::is_regular_file(status)) {
		result.error = "refusing to read non-regular config file '" +
			       path.string() + "'";
		return result;
	}

	std::string source;
	if (!read_config_file(path, source, result.error)) {
		return result;
	}
	toml::table table;
	try {
		table = toml::parse(source);
	} catch (const toml::parse_error &e) {
		const auto backup_path = choose_backup_path(path);
		std::filesystem::rename(path, backup_path, ec);
		if (ec) {
			result.error =
				"failed to backup invalid config file '" +
				path.string() + "': " + ec.message();
			return result;
		}
		if (!set_private_file_permissions(backup_path, result.error)) {
			return result;
		}
		if (!sync_parent_directory(path, result.error)) {
			return result;
		}

		const auto default_table = to_toml_table(result.config);
		if (!write_table(path, default_table, result.error)) {
			return result;
		}
		result.ok = true;
		result.repaired = true;
		result.replaced_invalid = true;
		result.backup_path = backup_path;
		result.messages.push_back("Invalid TOML syntax in '" +
					  path.string() +
					  "': " + std::string(e.description()));
		result.messages.push_back("Backed up invalid config to '" +
					  backup_path.string() + "'");
		result.messages.push_back("Restored default config file '" +
					  path.string() + "'");
		return result;
	} catch (const std::exception &e) {
		result.error = "failed to read config file '" + path.string() +
			       "': " + e.what();
		return result;
	}

	result.config = read_and_repair(table, result);
	if (result.repaired) {
		if (!write_table(path, table, result.error)) {
			result.ok = false;
			return result;
		}
	}
	result.ok = true;
	return result;
}

}
