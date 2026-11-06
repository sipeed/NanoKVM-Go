#pragma once

#include <cstdint>
#include <filesystem>
#include <optional>
#include <string>
#include <vector>

#include "lvgl_main/app_config_defaults.hpp"
#include "lvgl_main/screenshot_control.hpp"

namespace lvgl_main
{

struct DisplayConfig {
	std::string framebuffer_device {
		app_config_defaults::kFramebufferDevice.data()
	};
};

struct TouchConfig {
	std::string device { app_config_defaults::kTouchDevice.data() };
	bool disabled { app_config_defaults::kTouchDisabled };
};

struct HttpApiConfig {
	bool enabled { app_config_defaults::kHttpApiEnabled };
	std::string bind_host { app_config_defaults::kHttpApiBindHost.data() };
	int port { app_config_defaults::kHttpApiPort };
	std::string token { app_config_defaults::kHttpApiToken.data() };
};

struct RuntimeOptionsConfig {
	bool debug_logging { app_config_defaults::kRuntimeDebugLogging };
	bool touch_click_marker {
		app_config_defaults::kRuntimeTouchClickMarker
	};
	bool touch_calibrate { app_config_defaults::kRuntimeTouchCalibrate };
	bool touch_paint_test { app_config_defaults::kRuntimeTouchPaintTest };
};

struct ScreenshotConfig {
	bool enabled { app_config_defaults::kScreenshotEnabled };
	ScreenshotSignal signal { ScreenshotSignal::User1 };
	std::string output_directory {
		app_config_defaults::kScreenshotOutputDirectory.data()
	};
};

enum class UiPageTransitionMode : uint8_t {
	Slide,
	Flip
};
struct UiConfig {
	bool enable_auto_rotation { app_config_defaults::kEnableAutoRotation };
	bool manual_display_rotation {
		app_config_defaults::kManualDisplayRotation
	};
	bool experimental_settings_subpages {
		app_config_defaults::kExperimentalSettingsSubpages
	};
	bool enable_settings_optimistic_actions {
		app_config_defaults::kEnableSettingsOptimisticActions
	};
	UiPageTransitionMode page_transition_mode {
		UiPageTransitionMode::Slide
	};
	int wifi_boot_reconnect_timeout_ms {
		app_config_defaults::kWifiBootReconnectTimeoutMs
	};
	int settings_state_boot_timeout_ms {
		app_config_defaults::kSettingsStateBootTimeoutMs
	};
	bool wifi_loading_progress {
		app_config_defaults::kWifiLoadingProgress
	};
};

enum class CjkRegion : uint8_t {
	Japan,
	Korea,
	SimplifiedChinese,
	TraditionalChineseTaiwan,
	TraditionalChineseHongKong,
};

std::string_view cjk_region_name(CjkRegion region) noexcept;
std::optional<CjkRegion> parse_cjk_region(std::string_view value) noexcept;

enum class SsidScrollMode : uint8_t {
	On,
	Off,
	Auto,
};

std::string_view ssid_scroll_mode_name(SsidScrollMode mode) noexcept;
std::optional<SsidScrollMode>
parse_ssid_scroll_mode(std::string_view value) noexcept;
bool resolve_ssid_scroll_enabled(SsidScrollMode mode,
				 bool runtime_tiny_ttf) noexcept;

struct FontConfig {
	CjkRegion cjk_region { CjkRegion::SimplifiedChinese };
	int ssid_image_cache_kb { app_config_defaults::kFontsSsidImageCacheKb };
};

struct ExperimentalConfig {
	SsidScrollMode ssid_scroll { SsidScrollMode::Auto };
};

struct NanokvmRuntimeConfig {
	bool enable_runtime_backend {
		app_config_defaults::kNanokvmRuntimeBackendEnabled
	};
	bool preserve_last_known_good_state {
		app_config_defaults::kNanokvmPreserveLastKnownGoodState
	};
	std::string server_base_url {
		app_config_defaults::kNanokvmServerBaseUrl.data()
	};
	bool server_verify_tls { app_config_defaults::kNanokvmServerVerifyTls };
	std::string wifi_script {
		app_config_defaults::kNanokvmWifiScript.data()
	};
	std::string usb_gadget_restart_script {
		app_config_defaults::kNanokvmUsbGadgetRestartScript.data()
	};
	std::string backlight_root {
		app_config_defaults::kNanokvmBacklightRoot.data()
	};
	bool enable_high_risk_actions {
		app_config_defaults::kNanokvmHighRiskActionsEnabled
	};
	bool video_enabled { app_config_defaults::kNanokvmVideoEnabled };
	bool hid_enabled { app_config_defaults::kNanokvmHidEnabled };
	int backlight_percent { app_config_defaults::kNanokvmBacklightPercent };
	bool backlight_invert { app_config_defaults::kNanokvmBacklightInvert };
	bool auto_sleep_enabled {
		app_config_defaults::kNanokvmAutoSleepEnabled
	};
};

struct LauncherConfig {
	bool enabled { app_config_defaults::kLauncherEnabled };
	std::string python_bin {
		app_config_defaults::kLauncherPythonBin.data()
	};
	int exit_gesture_zone_px { app_config_defaults::kLauncherExitZonePx };
	int exit_gesture_hold_ms { app_config_defaults::kLauncherExitHoldMs };
	int child_term_timeout_ms {
		app_config_defaults::kLauncherChildTermTimeoutMs
	};
};

struct AppConfig {
	DisplayConfig display;
	TouchConfig touch;
	HttpApiConfig http_api;
	RuntimeOptionsConfig runtime;
	ScreenshotConfig screenshot;
	UiConfig ui;
	FontConfig fonts;
	ExperimentalConfig experimental;
	NanokvmRuntimeConfig nanokvm;
	LauncherConfig launcher;
};

struct AppConfigLoadResult {
	AppConfig config;
	RuntimeOptionsConfig effective_runtime;
	std::filesystem::path path;
	std::optional<std::filesystem::path> backup_path;
	std::vector<std::string> messages;
	std::vector<std::string> warnings;
	std::string error;
	bool ok { false };
	bool created { false };
	bool repaired { false };
	bool replaced_invalid { false };
};

AppConfig default_app_config();
std::string render_app_config_toml(const AppConfig &config);
bool save_app_config(const AppConfig &config, std::string &error,
		     const std::filesystem::path &path = std::filesystem::path(
			     app_config_defaults::kConfigPath.data()));

AppConfigLoadResult load_or_repair_app_config(
	const std::filesystem::path &path =
		std::filesystem::path(app_config_defaults::kConfigPath.data()));

}
