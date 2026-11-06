#include <algorithm>
#include <array>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>
#include <string_view>
#include <tuple>

#include <sys/stat.h>

#include <catch2/catch_test_macros.hpp>

#include <toml++/toml.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/app_config_defaults.hpp"

using namespace lvgl_main;

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
			("nanokvm_go_app_config_test_" + std::to_string(stamp));
		std::filesystem::create_directories(_path);
		std::filesystem::permissions(_path,
					     std::filesystem::perms::owner_all);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	std::filesystem::path config_path() const
	{
		return _path / "kvm_ui.toml";
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
	std::ostringstream out;
	out << in.rdbuf();
	return out.str();
}

mode_t file_mode(const std::filesystem::path &path)
{
	struct stat st {};
	REQUIRE(::stat(path.c_str(), &st) == 0);
	return st.st_mode & 0777;
}

}

TEST_CASE("app config creates a default TOML file when missing")
{
	STATIC_REQUIRE(app_config_defaults::kNanokvmBacklightPercent == 70);

	TempDir temp;
	const auto path = temp.config_path();

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.created);
	CHECK_FALSE(result.repaired);
	CHECK(std::filesystem::exists(path));
	CHECK(file_mode(path) == 0600);
	CHECK(result.config.display.framebuffer_device ==
	      app_config_defaults::kFramebufferDevice);
	CHECK(result.config.touch.device == app_config_defaults::kTouchDevice);
	CHECK(result.config.http_api.enabled ==
	      app_config_defaults::kHttpApiEnabled);
	CHECK(app_config_defaults::kHttpApiPort == 65501);
	CHECK(result.config.http_api.port == app_config_defaults::kHttpApiPort);
	CHECK(result.config.runtime.debug_logging ==
	      app_config_defaults::kRuntimeDebugLogging);
	CHECK(result.config.runtime.touch_click_marker ==
	      app_config_defaults::kRuntimeTouchClickMarker);
	CHECK(result.config.runtime.touch_calibrate ==
	      app_config_defaults::kRuntimeTouchCalibrate);
	CHECK(result.config.runtime.touch_paint_test ==
	      app_config_defaults::kRuntimeTouchPaintTest);
	CHECK(result.config.screenshot.enabled ==
	      app_config_defaults::kScreenshotEnabled);
	CHECK(result.config.screenshot.signal == ScreenshotSignal::User1);
	CHECK(result.config.screenshot.output_directory ==
	      app_config_defaults::kScreenshotOutputDirectory);
	CHECK(result.effective_runtime.debug_logging ==
	      app_config_defaults::kRuntimeDebugLogging);
	CHECK(result.config.ui.page_transition_mode ==
	      UiPageTransitionMode::Slide);
	CHECK(result.config.ui.wifi_boot_reconnect_timeout_ms ==
	      app_config_defaults::kWifiBootReconnectTimeoutMs);
	CHECK(app_config_defaults::kWifiBootReconnectTimeoutMs == 15000);
	CHECK(result.config.ui.settings_state_boot_timeout_ms ==
	      app_config_defaults::kSettingsStateBootTimeoutMs);
	CHECK(app_config_defaults::kSettingsStateBootTimeoutMs == 8000);
	CHECK(result.config.fonts.cjk_region == CjkRegion::SimplifiedChinese);
	CHECK(app_config_defaults::kFontsCjkRegion == "sc");
	CHECK(result.config.fonts.ssid_image_cache_kb ==
	      app_config_defaults::kFontsSsidImageCacheKb);
	CHECK(app_config_defaults::kFontsSsidImageCacheKb == 4096);
	CHECK(result.config.ui.enable_auto_rotation ==
	      app_config_defaults::kEnableAutoRotation);
	CHECK(result.config.ui.manual_display_rotation ==
	      app_config_defaults::kManualDisplayRotation);
	CHECK(result.config.ui.experimental_settings_subpages ==
	      app_config_defaults::kExperimentalSettingsSubpages);
	CHECK(app_config_defaults::kExperimentalSettingsSubpages);
	CHECK(result.config.experimental.ssid_scroll == SsidScrollMode::Auto);
	CHECK(app_config_defaults::kExperimentalSsidScroll == "auto");
	CHECK(result.config.ui.enable_settings_optimistic_actions ==
	      app_config_defaults::kEnableSettingsOptimisticActions);
	CHECK(app_config_defaults::kEnableSettingsOptimisticActions);
	CHECK(result.config.ui.wifi_loading_progress);
	CHECK(app_config_defaults::kWifiLoadingProgress);
	CHECK(result.config.nanokvm.video_enabled ==
	      app_config_defaults::kNanokvmVideoEnabled);
	CHECK(result.config.nanokvm.hid_enabled ==
	      app_config_defaults::kNanokvmHidEnabled);
	CHECK(result.config.nanokvm.backlight_percent ==
	      app_config_defaults::kNanokvmBacklightPercent);
	CHECK(result.config.nanokvm.backlight_invert ==
	      app_config_defaults::kNanokvmBacklightInvert);
	CHECK(result.config.nanokvm.auto_sleep_enabled ==
	      app_config_defaults::kNanokvmAutoSleepEnabled);
	CHECK(result.config.nanokvm.preserve_last_known_good_state ==
	      app_config_defaults::kNanokvmPreserveLastKnownGoodState);
	CHECK(app_config_defaults::kNanokvmPreserveLastKnownGoodState);
	CHECK(result.config.nanokvm.usb_gadget_restart_script ==
	      app_config_defaults::kNanokvmUsbGadgetRestartScript);

	const auto text = read_text(path);
	CHECK(text.find("[display]") != std::string::npos);
	CHECK(text.find("gamma_green_correction") == std::string::npos);
	CHECK(text.find("gamma_green_knee_start") == std::string::npos);
	CHECK(text.find("gamma_green_knee_slope") == std::string::npos);
	CHECK(text.find("gamma_rgb_global") == std::string::npos);
	CHECK(text.find("[touch]") != std::string::npos);
	CHECK(text.find("[http_api]") != std::string::npos);
	CHECK(text.find("[runtime]") != std::string::npos);
	CHECK(text.find("debug_logging = false") != std::string::npos);
	CHECK(text.find("touch_click_marker = false") != std::string::npos);
	CHECK(text.find("touch_calibrate = false") != std::string::npos);
	CHECK(text.find("touch_paint_test = false") != std::string::npos);
	CHECK(text.find("[screenshot]") != std::string::npos);
	CHECK(text.find("signal = 'SIGUSR1'") != std::string::npos);
	CHECK(text.find("output_directory = '/root/snapshot'") !=
	      std::string::npos);
	CHECK(text.find("[ui]") != std::string::npos);
	CHECK(text.find("enable_auto_rotation = false") != std::string::npos);
	CHECK(text.find("manual_display_rotation = false") !=
	      std::string::npos);
	CHECK(text.find("experimental_settings_subpages = true") !=
	      std::string::npos);
	CHECK(text.find("enable_180_rotation") == std::string::npos);
	CHECK(text.find("enable_developer_mode") == std::string::npos);
	CHECK(text.find("enable_global_popup_demo") == std::string::npos);
	CHECK(text.find("page_transition_mode") != std::string::npos);
	CHECK(text.find("wifi_config_mode") == std::string::npos);
	CHECK(text.find("wifi_boot_reconnect_timeout_ms = 15000") !=
	      std::string::npos);
	CHECK(text.find("settings_state_boot_timeout_ms = 8000") !=
	      std::string::npos);
	CHECK(text.find("[fonts]") != std::string::npos);
	const bool default_cjk_region_rendered =
		text.find("cjk_region = \"sc\"") != std::string::npos ||
		text.find("cjk_region = 'sc'") != std::string::npos;
	CHECK(default_cjk_region_rendered);
	CHECK(text.find("ssid_image_cache_kb = 4096") != std::string::npos);
	CHECK(text.find("[experimental]") != std::string::npos);
	CHECK(text.find("reserved = true") != std::string::npos);
	CHECK(text.find("ssid_scroll = 'auto'") != std::string::npos);
	CHECK(text.find("network_terms_qr_card") == std::string::npos);
	CHECK(text.find("enable_ssid_scroll") == std::string::npos);
	CHECK(text.find("enable_settings_optimistic_actions = true") !=
	      std::string::npos);
	CHECK(text.find("wifi_loading_progress = true") != std::string::npos);
	CHECK(text.find("wifi_loading_mode") == std::string::npos);
	CHECK(text.find("ttf") == std::string::npos);
	const bool default_transition_mode_rendered =
		text.find("\"slide\"") != std::string::npos ||
		text.find("'slide'") != std::string::npos;
	CHECK(default_transition_mode_rendered);
	CHECK(text.find("usb_gadget_restart_script") != std::string::npos);
	CHECK(text.find("/opt/scripts/usbdev.sh") != std::string::npos);
	CHECK(text.find("video_enabled = true") != std::string::npos);
	CHECK(text.find("hid_enabled = true") != std::string::npos);
	CHECK(text.find("ncm_enabled") == std::string::npos);
	CHECK(text.find("backlight_percent = 70") != std::string::npos);
	CHECK(text.find("backlight_invert = false") != std::string::npos);
	CHECK(text.find("auto_sleep_enabled = false") != std::string::npos);
	CHECK(text.find("preserve_last_known_good_state = true") !=
	      std::string::npos);

	CHECK(result.config.launcher.enabled ==
	      app_config_defaults::kLauncherEnabled);
	CHECK(result.config.launcher.exit_gesture_hold_ms ==
	      app_config_defaults::kLauncherExitHoldMs);
	CHECK(result.config.launcher.exit_gesture_zone_px ==
	      app_config_defaults::kLauncherExitZonePx);
	CHECK(text.find("[launcher]") != std::string::npos);
	CHECK(text.find("apps_dir") == std::string::npos);
	CHECK(text.find("demo_script") == std::string::npos);
	CHECK(text.find("exit_gesture_hold_ms = 1200") != std::string::npos);
	CHECK(text.find("exit_gesture_zone_px = 40") != std::string::npos);
}

TEST_CASE("app config loads valid TOML values")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[display]
framebuffer_device = "/dev/fb1"

[touch]
device = "/dev/input/event3"
disabled = true

[http_api]
enabled = true
bind_host = "0.0.0.0"
port = 9000
token = "0123456789abcdefg"

[runtime]
debug_logging = true
touch_click_marker = false
touch_calibrate = true
touch_paint_test = false

[screenshot]
enabled = true
signal = "SIGUSR2"
output_directory = "/var/lib/kvm/screenshots"

[ui]
enable_auto_rotation = true
manual_display_rotation = true
experimental_settings_subpages = false
page_transition_mode = "flip"
wifi_boot_reconnect_timeout_ms = 23000
settings_state_boot_timeout_ms = 9000
wifi_loading_progress = false
enable_settings_optimistic_actions = true

[fonts]
cjk_region = "hk"
ssid_image_cache_kb = 2048

[experimental]
reserved = true
ssid_scroll = "on"

[nanokvm]
enable_runtime_backend = true
preserve_last_known_good_state = true
server_base_url = "https://127.0.0.1"
server_verify_tls = false
wifi_script = "/kvmcomm/scripts/wifi.sh"
usb_gadget_restart_script = "/opt/scripts/usbdev.sh"
backlight_root = "/sys/class/backlight/backlight"
enable_high_risk_actions = true
video_enabled = true
hid_enabled = true
backlight_percent = 42
backlight_invert = true
auto_sleep_enabled = true

[launcher]
enabled = false
python_bin = "python3.13"
exit_gesture_zone_px = 60
exit_gesture_hold_ms = 2000
child_term_timeout_ms = 2500
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(file_mode(path) == 0600);
	CHECK_FALSE(result.created);
	CHECK_FALSE(result.repaired);
	CHECK(result.config.display.framebuffer_device == "/dev/fb1");
	CHECK(result.config.touch.device == "/dev/input/event3");
	CHECK(result.config.touch.disabled);
	CHECK(result.config.http_api.enabled);
	CHECK(result.config.http_api.bind_host == "0.0.0.0");
	CHECK(result.config.http_api.port == 9000);
	CHECK(result.config.http_api.token == "0123456789abcdefg");
	CHECK(result.config.runtime.debug_logging);
	CHECK_FALSE(result.config.runtime.touch_click_marker);
	CHECK(result.config.runtime.touch_calibrate);
	CHECK_FALSE(result.config.runtime.touch_paint_test);
	CHECK(result.config.screenshot.enabled);
	CHECK(result.config.screenshot.signal == ScreenshotSignal::User2);
	CHECK(result.config.screenshot.output_directory ==
	      "/var/lib/kvm/screenshots");
	CHECK(result.effective_runtime.debug_logging);
	CHECK_FALSE(result.effective_runtime.touch_click_marker);
	CHECK(result.effective_runtime.touch_calibrate);
	CHECK_FALSE(result.effective_runtime.touch_paint_test);
	CHECK(result.config.ui.enable_auto_rotation);
	CHECK(result.config.ui.manual_display_rotation);
	CHECK_FALSE(result.config.ui.experimental_settings_subpages);
	CHECK(result.config.ui.page_transition_mode ==
	      UiPageTransitionMode::Flip);
	CHECK(result.config.ui.wifi_boot_reconnect_timeout_ms == 23000);
	CHECK(result.config.ui.settings_state_boot_timeout_ms == 9000);
	CHECK(result.config.fonts.cjk_region ==
	      CjkRegion::TraditionalChineseHongKong);
	CHECK(result.config.fonts.ssid_image_cache_kb == 2048);
	CHECK(result.config.experimental.ssid_scroll == SsidScrollMode::On);
	CHECK(result.config.ui.enable_settings_optimistic_actions);
	CHECK_FALSE(result.config.ui.wifi_loading_progress);
	CHECK(result.config.nanokvm.usb_gadget_restart_script ==
	      "/opt/scripts/usbdev.sh");
	CHECK(result.config.nanokvm.video_enabled);
	CHECK(result.config.nanokvm.hid_enabled);
	CHECK(result.config.nanokvm.backlight_percent == 42);
	CHECK(result.config.nanokvm.backlight_invert);
	CHECK(result.config.nanokvm.auto_sleep_enabled);
	CHECK(result.config.nanokvm.preserve_last_known_good_state);
	CHECK_FALSE(result.config.launcher.enabled);
	CHECK(result.config.launcher.python_bin == "python3.13");
	CHECK(result.config.launcher.exit_gesture_zone_px == 60);
	CHECK(result.config.launcher.exit_gesture_hold_ms == 2000);
	CHECK(result.config.launcher.child_term_timeout_ms == 2500);
}

TEST_CASE("app config persists last-known-good refresh policy and repairs bad values")
{
	TempDir temp;
	const auto path = temp.config_path();

	write_text(path,
		   "[nanokvm]\npreserve_last_known_good_state = false\n");
	const auto disabled = load_or_repair_app_config(path);
	REQUIRE(disabled.ok);
	CHECK_FALSE(disabled.config.nanokvm.preserve_last_known_good_state);
	CHECK(read_text(path).find("preserve_last_known_good_state = false") !=
	      std::string::npos);

	write_text(path,
		   "[nanokvm]\npreserve_last_known_good_state = \"false\"\n");
	const auto repaired = load_or_repair_app_config(path);
	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(repaired.config.nanokvm.preserve_last_known_good_state ==
	      app_config_defaults::kNanokvmPreserveLastKnownGoodState);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Invalid or missing 'nanokvm.preserve_last_known_good_state', restored default") !=
		      repaired.messages.end());
	CHECK(read_text(path).find("preserve_last_known_good_state = true") !=
	      std::string::npos);
}

TEST_CASE("app config validates and repairs CJK font regions")
{
	const auto jp = parse_cjk_region("jp");
	const auto kr = parse_cjk_region("kr");
	const auto sc = parse_cjk_region("sc");
	const auto tc = parse_cjk_region("tc");
	const auto hk = parse_cjk_region("hk");
	REQUIRE(jp.has_value());
	REQUIRE(kr.has_value());
	REQUIRE(sc.has_value());
	REQUIRE(tc.has_value());
	REQUIRE(hk.has_value());
	CHECK(*jp == CjkRegion::Japan);
	CHECK(*kr == CjkRegion::Korea);
	CHECK(*sc == CjkRegion::SimplifiedChinese);
	CHECK(*tc == CjkRegion::TraditionalChineseTaiwan);
	CHECK(*hk == CjkRegion::TraditionalChineseHongKong);
	CHECK_FALSE(parse_cjk_region("CN").has_value());

	TempDir temp;
	const auto path = temp.config_path();
	write_text(path, "[fonts]\ncjk_region = \"invalid\"\n");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.fonts.cjk_region == CjkRegion::SimplifiedChinese);
	CHECK(std::find(result.messages.begin(), result.messages.end(),
					"Invalid or missing 'fonts.cjk_region', restored default") !=
		      result.messages.end());
	const auto repaired_text = read_text(path);
	const bool repaired_cjk_region_rendered =
		repaired_text.find("cjk_region = \"sc\"") != std::string::npos ||
		repaired_text.find("cjk_region = 'sc'") != std::string::npos;
	CHECK(repaired_cjk_region_rendered);
}

TEST_CASE("app config validates screenshot capture configuration")
{
	TempDir temp;
	const auto path = temp.config_path();

	write_text(path,
		   R"toml(
[screenshot]
enabled = "yes"
signal = "SIGKILL"
output_directory = "/root/../unsafe"
future_screenshot_key = "keep"
)toml");

	const auto repaired = load_or_repair_app_config(path);

	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK_FALSE(repaired.config.screenshot.enabled);
	CHECK(repaired.config.screenshot.signal == ScreenshotSignal::User1);
	CHECK(repaired.config.screenshot.output_directory ==
	      app_config_defaults::kScreenshotOutputDirectory);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Invalid or missing 'screenshot.enabled', restored default") !=
	      repaired.messages.end());
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Invalid or missing 'screenshot.signal', restored default") !=
	      repaired.messages.end());
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Invalid or missing 'screenshot.output_directory', restored default") !=
	      repaired.messages.end());

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("signal = 'SIGUSR1'") != std::string::npos);
	CHECK(repaired_text.find("output_directory = '/root/snapshot'") !=
	      std::string::npos);
	const bool future_key_preserved =
		repaired_text.find("future_screenshot_key = \"keep\"") !=
				std::string::npos ||
		repaired_text.find("future_screenshot_key = 'keep'") !=
				std::string::npos;
	CHECK(future_key_preserved);
}

TEST_CASE("app config validates SSID image cache budget")
{
	TempDir temp;
	const auto path = temp.config_path();
	const auto expect_default = [&path](const std::string &value) {
		write_text(path, "[fonts]\nssid_image_cache_kb = " + value + "\n");
		const auto result = load_or_repair_app_config(path);
		REQUIRE(result.ok);
		CHECK(result.config.fonts.ssid_image_cache_kb ==
		      app_config_defaults::kFontsSsidImageCacheKb);
		CHECK(std::find(
			result.messages.begin(), result.messages.end(),
			"Invalid or missing 'fonts.ssid_image_cache_kb', restored default") !=
		      result.messages.end());
		CHECK(read_text(path).find("ssid_image_cache_kb = 4096") !=
		      std::string::npos);
	};

	write_text(path, "[fonts]\n");
	const auto missing = load_or_repair_app_config(path);
	REQUIRE(missing.ok);
	CHECK(missing.config.fonts.ssid_image_cache_kb ==
	      app_config_defaults::kFontsSsidImageCacheKb);
	CHECK(std::find(
		missing.messages.begin(), missing.messages.end(),
		"Invalid or missing 'fonts.ssid_image_cache_kb', restored default") !=
	      missing.messages.end());

	expect_default("\"1024\"");
	expect_default("127");
	expect_default("4097");

	for (const int budget : { app_config_defaults::kFontsSsidImageCacheKbMin,
				  app_config_defaults::kFontsSsidImageCacheKbMax }) {
		write_text(path, "[fonts]\nssid_image_cache_kb = " +
				     std::to_string(budget) + "\n");
		const auto result = load_or_repair_app_config(path);
		REQUIRE(result.ok);
		CHECK(result.config.fonts.ssid_image_cache_kb == budget);
	}
}

TEST_CASE("app config removes obsolete experimental ttf key")
{
	TempDir temp;
	const auto path = temp.config_path();

	write_text(path,
		   "[experimental]\nttf = true\nfuture_option = \"keep\"\n");

	const auto repaired = load_or_repair_app_config(path);

	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(repaired.config.experimental.ssid_scroll == SsidScrollMode::Auto);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Removed obsolete 'experimental.ttf'") !=
	      repaired.messages.end());
	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("[experimental]") != std::string::npos);
	CHECK(repaired_text.find("reserved = true") != std::string::npos);
	CHECK(repaired_text.find("ssid_scroll = 'auto'") !=
	      std::string::npos);
	CHECK(repaired_text.find("ttf") == std::string::npos);
	const bool future_option_preserved =
		repaired_text.find("future_option = \"keep\"") !=
				std::string::npos ||
		repaired_text.find("future_option = 'keep'") != std::string::npos;
	CHECK(future_option_preserved);
}

TEST_CASE("app config restores missing experimental placeholder")
{
	TempDir temp;
	const auto path = temp.config_path();
	const std::string placeholder = "reserved = true";
	auto config_text = render_app_config_toml(default_app_config());
	const auto placeholder_position = config_text.find(placeholder);
	REQUIRE(placeholder_position != std::string::npos);
	config_text.replace(placeholder_position, placeholder.size(),
			    "future_option = \"keep\"");
	write_text(path, config_text);

	const auto repaired = load_or_repair_app_config(path);

	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
			"Invalid or missing 'experimental.reserved', restored default") !=
	      repaired.messages.end());
	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("reserved = true") != std::string::npos);
	const bool future_option_preserved =
		repaired_text.find("future_option = \"keep\"") !=
				std::string::npos ||
		repaired_text.find("future_option = 'keep'") != std::string::npos;
	CHECK(future_option_preserved);
}

TEST_CASE("app config validates experimental SSID scrolling")
{
	TempDir temp;
	const auto path = temp.config_path();
	const std::string default_field = "ssid_scroll = 'auto'";
	const std::string disabled_field = "ssid_scroll = 'off'";
	const std::string enabled_field = "ssid_scroll = 'on'";
	auto config_text = render_app_config_toml(default_app_config());
	const std::string reserved_field = "reserved = true";
	const auto reserved_position = config_text.find(reserved_field);
	REQUIRE(reserved_position != std::string::npos);
	config_text.insert(reserved_position + reserved_field.size(),
				   "\nfuture_option = \"keep\"");
	write_text(path, config_text);

	const auto automatic = load_or_repair_app_config(path);

	REQUIRE(automatic.ok);
	CHECK_FALSE(automatic.repaired);
	CHECK(automatic.config.experimental.ssid_scroll == SsidScrollMode::Auto);

	const auto default_position_after_load = config_text.find(default_field);
	REQUIRE(default_position_after_load != std::string::npos);
	config_text.replace(default_position_after_load, default_field.size(),
				    disabled_field);
	write_text(path, config_text);

	const auto disabled = load_or_repair_app_config(path);

	REQUIRE(disabled.ok);
	CHECK_FALSE(disabled.repaired);
	CHECK(disabled.config.experimental.ssid_scroll == SsidScrollMode::Off);

	const auto disabled_position = config_text.find(disabled_field);
	REQUIRE(disabled_position != std::string::npos);
	config_text.replace(disabled_position, disabled_field.size(), enabled_field);
	write_text(path, config_text);

	const auto enabled = load_or_repair_app_config(path);

	REQUIRE(enabled.ok);
	CHECK_FALSE(enabled.repaired);
	CHECK(enabled.config.experimental.ssid_scroll == SsidScrollMode::On);

	const auto enabled_position = config_text.find(enabled_field);
	REQUIRE(enabled_position != std::string::npos);
	config_text.replace(enabled_position, enabled_field.size(),
				    "ssid_scroll = 'yes'");
	config_text.insert(config_text.find("future_option"),
				   "enable_ssid_scroll = true\n");
	write_text(path, config_text);

	const auto repaired = load_or_repair_app_config(path);

	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(repaired.config.experimental.ssid_scroll == SsidScrollMode::Auto);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
				"Invalid 'experimental.ssid_scroll', restored default") !=
	      repaired.messages.end());
	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find(default_field) != std::string::npos);
	CHECK(repaired_text.find("enable_ssid_scroll") == std::string::npos);
	const bool future_option_preserved =
		repaired_text.find("future_option = \"keep\"") !=
					std::string::npos ||
		repaired_text.find("future_option = 'keep'") != std::string::npos;
	CHECK(future_option_preserved);

	auto wrong_type_text = repaired_text;
	const auto default_position_after_repair =
		wrong_type_text.find(default_field);
	REQUIRE(default_position_after_repair != std::string::npos);
	wrong_type_text.replace(default_position_after_repair, default_field.size(),
					"ssid_scroll = true");
	wrong_type_text.insert(wrong_type_text.find("future_option"),
				       "enable_ssid_scroll = false\n");
	write_text(path, wrong_type_text);

	const auto wrong_type = load_or_repair_app_config(path);

	REQUIRE(wrong_type.ok);
	CHECK(wrong_type.repaired);
	CHECK(wrong_type.config.experimental.ssid_scroll == SsidScrollMode::Auto);
	CHECK(std::find(wrong_type.messages.begin(), wrong_type.messages.end(),
				"Invalid 'experimental.ssid_scroll', restored default") !=
	      wrong_type.messages.end());
	const auto wrong_type_repaired_text = read_text(path);
	CHECK(wrong_type_repaired_text.find(default_field) != std::string::npos);
	CHECK(wrong_type_repaired_text.find("enable_ssid_scroll") ==
	      std::string::npos);

	auto missing_text = wrong_type_repaired_text;
	const auto default_position_before_missing = missing_text.find(default_field);
	REQUIRE(default_position_before_missing != std::string::npos);
	missing_text.erase(default_position_before_missing, default_field.size());
	if (default_position_before_missing < missing_text.size() &&
	    missing_text[default_position_before_missing] == '\n') {
		missing_text.erase(default_position_before_missing, 1);
	}
	write_text(path, missing_text);

	const auto missing = load_or_repair_app_config(path);

	REQUIRE(missing.ok);
	CHECK(missing.repaired);
	CHECK(missing.config.experimental.ssid_scroll == SsidScrollMode::Auto);
	CHECK(std::find(missing.messages.begin(), missing.messages.end(),
				"Invalid or missing 'experimental.ssid_scroll', restored default") !=
	      missing.messages.end());
	CHECK(read_text(path).find(default_field) != std::string::npos);
}

TEST_CASE("app config migrates legacy experimental SSID scrolling")
{
	TempDir temp;
	const auto path = temp.config_path();

	for (const auto &[legacy_value, expected_mode, expected_text] :
	     std::array {
		     std::tuple { true, SsidScrollMode::On,
				  std::string_view { "ssid_scroll = 'on'" } },
		     std::tuple { false, SsidScrollMode::Off,
				  std::string_view { "ssid_scroll = 'off'" } },
	     }) {
		write_text(path, std::string("[experimental]\nreserved = true\n") +
				 "enable_ssid_scroll = " +
				 (legacy_value ? "true\n" : "false\n") +
				 "future_option = \"keep\"\n");

		const auto result = load_or_repair_app_config(path);
		REQUIRE(result.ok);
		CHECK(result.repaired);
		CHECK(result.config.experimental.ssid_scroll == expected_mode);
		CHECK(std::find(result.messages.begin(), result.messages.end(),
				"Missing 'experimental.ssid_scroll', migrated legacy "
				"'experimental.enable_ssid_scroll'") !=
		      result.messages.end());
		const auto text = read_text(path);
		CHECK(text.find(expected_text) != std::string::npos);
		CHECK(text.find("enable_ssid_scroll") == std::string::npos);
		CHECK(text.find("future_option = 'keep'") != std::string::npos);
	}
}

TEST_CASE("app config removes legacy SSID scrolling key when current mode wins")
{
	TempDir temp;
	const auto path = temp.config_path();

	write_text(path,
		   "[experimental]\nreserved = true\nssid_scroll = \"on\"\n"
		   "enable_ssid_scroll = false\nfuture_option = \"keep\"\n");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.experimental.ssid_scroll == SsidScrollMode::On);
	CHECK(std::find(result.messages.begin(), result.messages.end(),
				"Removed legacy 'experimental.enable_ssid_scroll'") !=
	      result.messages.end());
	const auto text = read_text(path);
	CHECK(text.find("ssid_scroll = 'on'") != std::string::npos);
	CHECK(text.find("enable_ssid_scroll") == std::string::npos);
	CHECK(text.find("future_option = 'keep'") != std::string::npos);
}

TEST_CASE("app config retires the experimental network settings subpage key")
{
	TempDir temp;
	const auto path = temp.config_path();

	const auto build_config = [&path](std::string_view value) {
		auto config_text = render_app_config_toml(default_app_config());
		const auto reserved_position =
			config_text.find("reserved = true");
		REQUIRE(reserved_position != std::string::npos);
		config_text.insert(
			reserved_position + std::string("reserved = true").size(),
			"\nnetwork_settings_subpage = " + std::string(value) +
				"\nfuture_network_key = \"keep\"");
		write_text(path, config_text);
	};

	build_config("true");
	const auto enabled = load_or_repair_app_config(path);
	REQUIRE(enabled.ok);
	CHECK(enabled.repaired);
	CHECK(std::find(enabled.messages.begin(), enabled.messages.end(),
			"Removed obsolete "
			"'experimental.network_settings_subpage'") !=
	      enabled.messages.end());
	auto text = read_text(path);
	CHECK(text.find("network_settings_subpage") == std::string::npos);
	CHECK(text.find("reserved = true") != std::string::npos);
	CHECK(text.find("future_network_key =") != std::string::npos);
	CHECK(text.find("'keep'") != std::string::npos);

	const auto clean = load_or_repair_app_config(path);
	REQUIRE(clean.ok);
	CHECK_FALSE(clean.repaired);
	CHECK(read_text(path).find("network_settings_subpage") ==
	      std::string::npos);

	build_config("false");
	const auto disabled = load_or_repair_app_config(path);
	REQUIRE(disabled.ok);
	CHECK(disabled.repaired);
	CHECK(std::find(disabled.messages.begin(), disabled.messages.end(),
			"Removed obsolete "
			"'experimental.network_settings_subpage'") !=
	      disabled.messages.end());
	CHECK(read_text(path).find("network_settings_subpage") ==
	      std::string::npos);

	build_config("\"yes\"");
	const auto wrong_type = load_or_repair_app_config(path);
	REQUIRE(wrong_type.ok);
	CHECK(wrong_type.repaired);
	CHECK(std::find(wrong_type.messages.begin(), wrong_type.messages.end(),
			"Removed obsolete "
			"'experimental.network_settings_subpage'") !=
	      wrong_type.messages.end());
	CHECK(read_text(path).find("network_settings_subpage") ==
	      std::string::npos);
}

TEST_CASE("app config retires the experimental network terms QR card tuning keys")
{
	TempDir temp;
	const auto path = temp.config_path();

	const auto build_config = [&path](std::string_view size_value,
					  std::string_view radius_value) {
		auto config_text = render_app_config_toml(default_app_config());
		const auto reserved_position =
			config_text.find("reserved = true");
		REQUIRE(reserved_position != std::string::npos);
		config_text.insert(
			reserved_position + std::string("reserved = true").size(),
			"\nnetwork_terms_qr_card_size_px = " +
				std::string(size_value) +
				"\nnetwork_terms_qr_card_radius_px = " +
				std::string(radius_value) +
				"\nfuture_network_key = \"keep\"");
		write_text(path, config_text);
	};

	build_config("140", "10");
	const auto measured = load_or_repair_app_config(path);
	REQUIRE(measured.ok);
	CHECK(measured.repaired);
	CHECK(std::find(measured.messages.begin(), measured.messages.end(),
			"Removed obsolete "
			"'experimental.network_terms_qr_card_size_px'") !=
	      measured.messages.end());
	CHECK(std::find(measured.messages.begin(), measured.messages.end(),
			"Removed obsolete "
			"'experimental.network_terms_qr_card_radius_px'") !=
	      measured.messages.end());
	auto text = read_text(path);
	CHECK(text.find("network_terms_qr_card") == std::string::npos);
	CHECK(text.find("reserved = true") != std::string::npos);
	CHECK(text.find("future_network_key =") != std::string::npos);
	CHECK(text.find("'keep'") != std::string::npos);

	const auto clean = load_or_repair_app_config(path);
	REQUIRE(clean.ok);
	CHECK_FALSE(clean.repaired);
	CHECK(read_text(path).find("network_terms_qr_card") ==
	      std::string::npos);

	build_config("\"big\"", "\"round\"");
	const auto wrong_type = load_or_repair_app_config(path);
	REQUIRE(wrong_type.ok);
	CHECK(wrong_type.repaired);
	CHECK(std::find(wrong_type.messages.begin(), wrong_type.messages.end(),
			"Removed obsolete "
			"'experimental.network_terms_qr_card_size_px'") !=
	      wrong_type.messages.end());
	CHECK(std::find(wrong_type.messages.begin(), wrong_type.messages.end(),
			"Removed obsolete "
			"'experimental.network_terms_qr_card_radius_px'") !=
	      wrong_type.messages.end());
	CHECK(read_text(path).find("network_terms_qr_card") ==
	      std::string::npos);
}

TEST_CASE("SSID scroll mode resolves auto from the effective font mode")
{
	CHECK(resolve_ssid_scroll_enabled(SsidScrollMode::On, false));
	CHECK(resolve_ssid_scroll_enabled(SsidScrollMode::On, true));
	CHECK_FALSE(resolve_ssid_scroll_enabled(SsidScrollMode::Off, false));
	CHECK_FALSE(resolve_ssid_scroll_enabled(SsidScrollMode::Off, true));
	CHECK_FALSE(resolve_ssid_scroll_enabled(SsidScrollMode::Auto, false));
	CHECK(resolve_ssid_scroll_enabled(SsidScrollMode::Auto, true));
}

TEST_CASE("app config validates ui settings optimistic actions")
{
	TempDir temp;
	const auto path = temp.config_path();
	const std::string field_true =
		"enable_settings_optimistic_actions = true";
	const std::string field_false =
		"enable_settings_optimistic_actions = false";
	auto config_text = render_app_config_toml(default_app_config());
	write_text(path, config_text);
	const auto def = load_or_repair_app_config(path);
	REQUIRE(def.ok);
	CHECK_FALSE(def.repaired);
	CHECK(def.config.ui.enable_settings_optimistic_actions);

	config_text = read_text(path);
	const auto field_position = config_text.find(field_true);
	REQUIRE(field_position != std::string::npos);
	config_text.erase(field_position, field_true.size() + 1);
	const auto reserved_position = config_text.find("reserved = true");
	REQUIRE(reserved_position != std::string::npos);
	config_text.insert(reserved_position + std::string("reserved = true").size(),
			   "\nfuture_optimistic_key = \"keep\"");
	write_text(path, config_text);

	const auto missing = load_or_repair_app_config(path);
	REQUIRE(missing.ok);
	CHECK(missing.repaired);
	CHECK(missing.config.ui.enable_settings_optimistic_actions);
	CHECK(std::find(missing.messages.begin(), missing.messages.end(),
			"Invalid or missing "
			"'ui.enable_settings_optimistic_actions', restored default") !=
	      missing.messages.end());
	CHECK(read_text(path).find(field_true) != std::string::npos);
	CHECK(read_text(path).find("future_optimistic_key") != std::string::npos);

	config_text = read_text(path);
	const auto true_position = config_text.find(field_true);
	REQUIRE(true_position != std::string::npos);
	config_text.replace(true_position, field_true.size(), field_false);
	write_text(path, config_text);
	const auto disabled = load_or_repair_app_config(path);
	REQUIRE(disabled.ok);
	CHECK_FALSE(disabled.repaired);
	CHECK_FALSE(disabled.config.ui.enable_settings_optimistic_actions);

	config_text = read_text(path);
	const auto false_position = config_text.find(field_false);
	REQUIRE(false_position != std::string::npos);
	config_text.replace(false_position, field_false.size(),
			    "enable_settings_optimistic_actions = \"yes\"");
	write_text(path, config_text);
	const auto repaired = load_or_repair_app_config(path);
	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(repaired.config.ui.enable_settings_optimistic_actions);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
			"Invalid or missing "
			"'ui.enable_settings_optimistic_actions', restored default") !=
	      repaired.messages.end());
	CHECK(read_text(path).find(field_true) != std::string::npos);
	CHECK(read_text(path).find("future_optimistic_key") != std::string::npos);
}

TEST_CASE("app config retires experimental settings optimistic actions key")
{
	TempDir temp;
	const auto path = temp.config_path();

	auto config_text = render_app_config_toml(default_app_config());
	const auto field_position =
		config_text.find("enable_settings_optimistic_actions = true");
	REQUIRE(field_position != std::string::npos);
	config_text.erase(
		field_position,
		std::string("enable_settings_optimistic_actions = true").size() + 1);
	const auto reserved_position = config_text.find("reserved = true");
	REQUIRE(reserved_position != std::string::npos);
	config_text.insert(
		reserved_position + std::string("reserved = true").size(),
		"\nenable_settings_optimistic_actions = false\nfuture_optimistic_key = \"keep\"");
	write_text(path, config_text);

	const auto result = load_or_repair_app_config(path);
	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.ui.enable_settings_optimistic_actions);
	CHECK(std::find(result.messages.begin(), result.messages.end(),
			"Removed obsolete 'experimental.enable_settings_optimistic_actions'") !=
	      result.messages.end());
	const auto text = read_text(path);
	CHECK(text.find("enable_settings_optimistic_actions = true") !=
	      std::string::npos);
	CHECK(text.find("enable_settings_optimistic_actions = false") ==
	      std::string::npos);
	CHECK(text.find("future_optimistic_key") != std::string::npos);
}

TEST_CASE("app config validates ui wifi loading progress")
{
	TempDir temp;
	const auto path = temp.config_path();

	auto config = default_app_config();
	write_text(path, render_app_config_toml(config));
	const auto def = load_or_repair_app_config(path);
	REQUIRE(def.ok);
	CHECK_FALSE(def.repaired);
	CHECK(def.config.ui.wifi_loading_progress);
	{
		const auto text = read_text(path);
		CHECK(text.find("wifi_loading_progress = true") !=
		      std::string::npos);
		CHECK(text.find("wifi_loading_mode") == std::string::npos);
	}

	config.ui.wifi_loading_progress = true;
	write_text(path, render_app_config_toml(config));
	const auto a = load_or_repair_app_config(path);
	REQUIRE(a.ok);
	CHECK_FALSE(a.repaired);
	CHECK(a.config.ui.wifi_loading_progress);

	config.ui.wifi_loading_progress = false;
	write_text(path, render_app_config_toml(config));
	const auto b = load_or_repair_app_config(path);
	REQUIRE(b.ok);
	CHECK_FALSE(b.repaired);
	CHECK_FALSE(b.config.ui.wifi_loading_progress);

	auto text = render_app_config_toml(config);
	const auto key_pos = text.find("wifi_loading_progress");
	REQUIRE(key_pos != std::string::npos);
	const auto line_start = text.rfind('\n', key_pos) + 1;
	const auto line_end = text.find('\n', key_pos);
	text.replace(line_start, line_end - line_start,
		     "wifi_loading_progress = \"yes\"");
	write_text(path, text);
	const auto repaired = load_or_repair_app_config(path);
	REQUIRE(repaired.ok);
	CHECK(repaired.repaired);
	CHECK(repaired.config.ui.wifi_loading_progress);
	CHECK(std::find(repaired.messages.begin(), repaired.messages.end(),
			"Invalid 'ui.wifi_loading_progress', "
			"restored default") != repaired.messages.end());
}

TEST_CASE("app config migrates legacy experimental wifi loading mode")
{
	TempDir temp;
	const auto path = temp.config_path();

	write_text(path,
		   R"toml(
[experimental]
reserved = true
ssid_scroll = "off"
wifi_loading_mode = "progress"
)toml");
	const auto progress_result = load_or_repair_app_config(path);
	REQUIRE(progress_result.ok);
	CHECK(progress_result.repaired);
	CHECK(progress_result.config.ui.wifi_loading_progress);
	CHECK(std::any_of(progress_result.messages.begin(),
			 progress_result.messages.end(),
			 [](const auto &message) {
				 return message ==
					"Missing 'ui.wifi_loading_progress', "
					"migrated legacy "
					"'experimental.wifi_loading_mode'";
			 }));
	{
		const auto text = read_text(path);
		CHECK(text.find("wifi_loading_progress = true") !=
		      std::string::npos);
		CHECK(text.find("wifi_loading_mode") == std::string::npos);
	}

	write_text(path,
		   R"toml(
[experimental]
reserved = true
ssid_scroll = "off"
wifi_loading_mode = "none"
)toml");
	const auto none_result = load_or_repair_app_config(path);
	REQUIRE(none_result.ok);
	CHECK(none_result.repaired);
	CHECK_FALSE(none_result.config.ui.wifi_loading_progress);
	CHECK(read_text(path).find("wifi_loading_mode") ==
	      std::string::npos);
}

TEST_CASE("app config migrates obsolete launcher keys")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[launcher]
enabled = false
python_bin = "python3.13"
apps_dir = "/data/apps"
demo_script = "/legacy/demo.py"
exit_gesture_zone_px = 60
exit_gesture_hold_ms = 2000
child_term_timeout_ms = 2500
future_launcher_key = "keep"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(file_mode(path) == 0600);
	CHECK_FALSE(result.config.launcher.enabled);
	CHECK(result.config.launcher.python_bin == "python3.13");
	CHECK(result.config.launcher.exit_gesture_zone_px == 60);
	CHECK(result.config.launcher.exit_gesture_hold_ms == 2000);
	CHECK(result.config.launcher.child_term_timeout_ms == 2500);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'launcher.apps_dir'";
			  }));
	CHECK(std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message ==
			       "Removed obsolete 'launcher.demo_script'";
		}));

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("apps_dir") == std::string::npos);
	CHECK(repaired_text.find("demo_script") == std::string::npos);
	const auto repaired_table = toml::parse(repaired_text);
	const auto *repaired_launcher_table =
		repaired_table["launcher"].as_table();
	REQUIRE(repaired_launcher_table != nullptr);
	const auto persisted_enabled =
		(*repaired_launcher_table)["enabled"].value<bool>();
	const auto persisted_python_bin =
		(*repaired_launcher_table)["python_bin"].value<std::string>();
	const auto persisted_exit_zone =
		(*repaired_launcher_table)["exit_gesture_zone_px"]
			.value<int64_t>();
	const auto persisted_exit_hold =
		(*repaired_launcher_table)["exit_gesture_hold_ms"]
			.value<int64_t>();
	const auto persisted_child_timeout =
		(*repaired_launcher_table)["child_term_timeout_ms"]
			.value<int64_t>();
	const auto preserved_future_launcher_key =
		(*repaired_launcher_table)["future_launcher_key"]
			.value<std::string>();
	REQUIRE(persisted_enabled.has_value());
	REQUIRE(persisted_python_bin.has_value());
	REQUIRE(persisted_exit_zone.has_value());
	REQUIRE(persisted_exit_hold.has_value());
	REQUIRE(persisted_child_timeout.has_value());
	REQUIRE(preserved_future_launcher_key.has_value());
	CHECK_FALSE(*persisted_enabled);
	CHECK(*persisted_python_bin == "python3.13");
	CHECK(*persisted_exit_zone == 60);
	CHECK(*persisted_exit_hold == 2000);
	CHECK(*persisted_child_timeout == 2500);
	CHECK(*preserved_future_launcher_key == "keep");
}

TEST_CASE("app config removes obsolete display gamma keys")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[display]
framebuffer_device = "/dev/fb1"
gamma_green_correction = 0.75
gamma_green_knee_start = 48
gamma_green_knee_slope = 0.5
gamma_rgb_global = 0.9
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.display.framebuffer_device == "/dev/fb1");
	const bool gamma_removal_reported = std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message.find("display.gamma_green_correction") !=
			       std::string::npos;
		});
	CHECK(gamma_removal_reported);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("framebuffer_device = '/dev/fb1'") !=
	      std::string::npos);
	CHECK(repaired_text.find("gamma_green_correction") ==
	      std::string::npos);
	CHECK(repaired_text.find("gamma_green_knee_start") ==
	      std::string::npos);
	CHECK(repaired_text.find("gamma_green_knee_slope") ==
	      std::string::npos);
	CHECK(repaired_text.find("gamma_rgb_global") == std::string::npos);
}

TEST_CASE("app config removes obsolete nanokvm ncm_enabled key")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[nanokvm]
enable_runtime_backend = true
wifi_script = "/opt/scripts/wifi.sh"
usb_gadget_restart_script = "/opt/scripts/usbdev.sh"
backlight_root = "/sys/class/backlight/backlight"
enable_high_risk_actions = true
video_enabled = true
hid_enabled = true
ncm_enabled = true
backlight_percent = 70
backlight_invert = false
auto_sleep_enabled = false
future_nanokvm_key = "keep"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(file_mode(path) == 0600);
	CHECK(result.config.nanokvm.video_enabled);
	CHECK(result.config.nanokvm.hid_enabled);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'nanokvm.ncm_enabled'";
			  }));

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("ncm_enabled") == std::string::npos);
	CHECK(repaired_text.find("video_enabled = true") !=
	      std::string::npos);
	CHECK(repaired_text.find("hid_enabled = true") !=
	      std::string::npos);
	const auto repaired_table = toml::parse(repaired_text);
	const auto *repaired_nanokvm_table =
		repaired_table["nanokvm"].as_table();
	REQUIRE(repaired_nanokvm_table != nullptr);
	const auto preserved_future_key =
		(*repaired_nanokvm_table)["future_nanokvm_key"]
			.value<std::string>();
	REQUIRE(preserved_future_key.has_value());
	CHECK(*preserved_future_key == "keep");
}

TEST_CASE("app config removes obsolete nanokvm ncm_enabled false value")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[nanokvm]
video_enabled = true
hid_enabled = true
ncm_enabled = false
backlight_percent = 70
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'nanokvm.ncm_enabled'";
			  }));

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("ncm_enabled") == std::string::npos);
}

TEST_CASE("app config removes invalid nanokvm ncm_enabled type")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[nanokvm]
video_enabled = true
hid_enabled = true
ncm_enabled = "bad"
backlight_percent = 70
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'nanokvm.ncm_enabled'";
			  }));

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("ncm_enabled") == std::string::npos);
}

TEST_CASE("app config loads explicit experimental settings subpages true")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path, render_app_config_toml(default_app_config()));

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK_FALSE(result.repaired);
	CHECK(result.config.ui.experimental_settings_subpages);
}

TEST_CASE("app config resolves runtime touch mutex by last configured option")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[runtime]
debug_logging = true
touch_calibrate = true
touch_click_marker = true
touch_paint_test = true
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.runtime.debug_logging);
	CHECK(result.config.runtime.touch_calibrate);
	CHECK(result.config.runtime.touch_click_marker);
	CHECK(result.config.runtime.touch_paint_test);
	CHECK(result.effective_runtime.debug_logging);
	CHECK_FALSE(result.effective_runtime.touch_calibrate);
	CHECK_FALSE(result.effective_runtime.touch_click_marker);
	CHECK(result.effective_runtime.touch_paint_test);
	REQUIRE(result.warnings.size() == 1U);
	CHECK(result.warnings.front().find("runtime.touch_paint_test") !=
	      std::string::npos);
	CHECK(result.warnings.front().find("runtime.touch_calibrate") !=
	      std::string::npos);
	CHECK(result.warnings.front().find("runtime.touch_click_marker") !=
	      std::string::npos);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("touch_calibrate = true") !=
	      std::string::npos);
	CHECK(repaired_text.find("touch_click_marker = true") !=
	      std::string::npos);
	CHECK(repaired_text.find("touch_paint_test = true") !=
	      std::string::npos);
}

TEST_CASE("app config migrates legacy USB script key")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[display]
framebuffer_device = "/dev/fb0"

[touch]
device = "/dev/input/event0"
disabled = false

[http_api]
enabled = false
bind_host = "127.0.0.1"
port = 0
token = ""

[nanokvm]
enable_runtime_backend = true
server_base_url = "https://127.0.0.1"
server_verify_tls = false
wifi_script = "/kvmcomm/scripts/wifi.sh"
usb_script = "/custom/usbdev.sh"
backlight_root = "/sys/class/backlight/backlight"
enable_high_risk_actions = true
video_enabled = false
hid_enabled = false
ncm_enabled = false
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.nanokvm.usb_gadget_restart_script ==
	      "/custom/usbdev.sh");

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("usb_gadget_restart_script") !=
	      std::string::npos);
	CHECK(repaired_text.find("/custom/usbdev.sh") != std::string::npos);
	CHECK(repaired_text.find("usb_script") == std::string::npos);
	CHECK(repaired_text.find("ncm_enabled") == std::string::npos);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'nanokvm.ncm_enabled'";
			  }));
}

TEST_CASE("app config migrates obsolete UI keys and legacy manual rotation")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[ui]
enable_global_popup_demo = true
enable_180_rotation = true
enable_developer_mode = false
page_transition_mode = "slide"
future_ui_key = "keep"
wifi_config_mode = "toggle"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.ui.enable_auto_rotation);
	CHECK(result.config.ui.experimental_settings_subpages ==
	      app_config_defaults::kExperimentalSettingsSubpages);
	const bool popup_removal_reported = std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message.find("ui.enable_global_popup_demo") !=
			       std::string::npos;
		});
	CHECK(popup_removal_reported);
	const bool wifi_config_mode_removal_reported = std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message ==
			       "Removed obsolete 'ui.wifi_config_mode'";
		});
	CHECK(wifi_config_mode_removal_reported);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("enable_auto_rotation = true") !=
	      std::string::npos);
	const auto repaired_table = toml::parse(repaired_text);
	const auto preserved_future_ui_key =
		repaired_table["ui"]["future_ui_key"].value<std::string>();
	REQUIRE(preserved_future_ui_key.has_value());
	CHECK(*preserved_future_ui_key == "keep");
	CHECK(repaired_text.find("experimental_settings_subpages = true") !=
	      std::string::npos);
	const bool repaired_experimental_field_message = std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message.find(
				       "ui.experimental_settings_subpages") !=
			       std::string::npos;
		});
	CHECK(repaired_experimental_field_message);
	CHECK(repaired_text.find("enable_180_rotation") == std::string::npos);
	CHECK(repaired_text.find("enable_developer_mode") == std::string::npos);
	CHECK(repaired_text.find("enable_global_popup_demo") ==
	      std::string::npos);
	CHECK(repaired_text.find("wifi_config_mode") == std::string::npos);
}

TEST_CASE("app config migrates old default USB restart script path")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[nanokvm]
usb_gadget_restart_script = "/kvmapp/scripts/usbdev.sh"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.nanokvm.usb_gadget_restart_script ==
	      app_config_defaults::kNanokvmUsbGadgetRestartScript);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("/opt/scripts/usbdev.sh") !=
	      std::string::npos);
	CHECK(repaired_text.find("/kvmapp/scripts/usbdev.sh") ==
	      std::string::npos);
}

TEST_CASE("app config migrates legacy USB script key with old default path")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[nanokvm]
usb_script = "/kvmapp/scripts/usbdev.sh"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.nanokvm.usb_gadget_restart_script ==
	      app_config_defaults::kNanokvmUsbGadgetRestartScript);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("usb_gadget_restart_script") !=
	      std::string::npos);
	CHECK(repaired_text.find("/opt/scripts/usbdev.sh") !=
	      std::string::npos);
	CHECK(repaired_text.find("/kvmapp/scripts/usbdev.sh") ==
	      std::string::npos);
	CHECK(repaired_text.find("usb_script") == std::string::npos);
}

TEST_CASE("app config repairs invalid fields and preserves valid fields")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[display]
framebuffer_device = "/tmp/fb0"

[touch]
device = "relative-event"
disabled = "yes"

[http_api]
enabled = true
bind_host = "127.0.0.1"
port = 70000
token = ""

[ui]
enable_auto_rotation = "yes"
manual_display_rotation = "yes"
experimental_settings_subpages = "maybe"
enable_developer_mode = 1
page_transition_mode = "curl"
wifi_config_mode = "portal"
wifi_boot_reconnect_timeout_ms = 0
settings_state_boot_timeout_ms = 0

[nanokvm]
video_enabled = "yes"
hid_enabled = 1
ncm_enabled = "bad"
backlight_percent = 0
backlight_invert = "no"
auto_sleep_enabled = "yes"
)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(file_mode(path) == 0600);
	CHECK_FALSE(result.replaced_invalid);
	CHECK(result.config.display.framebuffer_device ==
	      app_config_defaults::kFramebufferDevice);
	CHECK(result.config.touch.device == app_config_defaults::kTouchDevice);
	CHECK(result.config.touch.disabled ==
	      app_config_defaults::kTouchDisabled);
	CHECK(result.config.http_api.port == app_config_defaults::kHttpApiPort);
	CHECK(result.config.ui.enable_auto_rotation ==
	      app_config_defaults::kEnableAutoRotation);
	CHECK(result.config.ui.manual_display_rotation ==
	      app_config_defaults::kManualDisplayRotation);
	CHECK(result.config.ui.experimental_settings_subpages ==
	      app_config_defaults::kExperimentalSettingsSubpages);
	CHECK(result.config.ui.page_transition_mode ==
	      UiPageTransitionMode::Slide);
	CHECK(result.config.ui.wifi_boot_reconnect_timeout_ms ==
	      app_config_defaults::kWifiBootReconnectTimeoutMs);
	CHECK(result.config.ui.settings_state_boot_timeout_ms ==
	      app_config_defaults::kSettingsStateBootTimeoutMs);
	CHECK(result.config.nanokvm.video_enabled ==
	      app_config_defaults::kNanokvmVideoEnabled);
	CHECK(result.config.nanokvm.hid_enabled ==
	      app_config_defaults::kNanokvmHidEnabled);
	CHECK(result.config.nanokvm.backlight_percent ==
	      app_config_defaults::kNanokvmBacklightPercent);
	CHECK(result.config.nanokvm.backlight_invert ==
	      app_config_defaults::kNanokvmBacklightInvert);
	CHECK(result.config.nanokvm.auto_sleep_enabled ==
	      app_config_defaults::kNanokvmAutoSleepEnabled);

	const auto repaired_text = read_text(path);
	CHECK(repaired_text.find("relative-event") == std::string::npos);
	CHECK(repaired_text.find("page_transition_mode") != std::string::npos);
	CHECK(repaired_text.find("wifi_config_mode") == std::string::npos);
	CHECK(repaired_text.find("wifi_boot_reconnect_timeout_ms = 15000") !=
	      std::string::npos);
	CHECK(repaired_text.find("settings_state_boot_timeout_ms = 8000") !=
	      std::string::npos);
	CHECK(repaired_text.find("enable_auto_rotation = false") !=
	      std::string::npos);
	CHECK(repaired_text.find("manual_display_rotation = false") !=
	      std::string::npos);
	CHECK(repaired_text.find("experimental_settings_subpages = true") !=
	      std::string::npos);
	const bool repaired_experimental_field_message = std::any_of(
		result.messages.begin(), result.messages.end(),
		[](const auto &message) {
			return message.find(
				       "ui.experimental_settings_subpages") !=
			       std::string::npos;
		});
	CHECK(repaired_experimental_field_message);
	CHECK(repaired_text.find("enable_180_rotation") == std::string::npos);
	CHECK(repaired_text.find("enable_developer_mode") == std::string::npos);
	const bool repaired_transition_mode_rendered =
		repaired_text.find("\"slide\"") != std::string::npos ||
		repaired_text.find("'slide'") != std::string::npos;
	CHECK(repaired_transition_mode_rendered);
	CHECK(repaired_text.find("curl") == std::string::npos);
	CHECK(repaired_text.find("portal") == std::string::npos);
	CHECK(repaired_text.find("/dev/fb0") != std::string::npos);
	CHECK(repaired_text.find("port = 65501") != std::string::npos);
	CHECK(repaired_text.find("video_enabled = true") != std::string::npos);
	CHECK(repaired_text.find("hid_enabled = true") != std::string::npos);
	CHECK(repaired_text.find("ncm_enabled") == std::string::npos);
	CHECK(std::any_of(result.messages.begin(), result.messages.end(),
			  [](const auto &message) {
				  return message ==
					 "Removed obsolete 'nanokvm.ncm_enabled'";
			  }));
	CHECK(repaired_text.find("backlight_percent = 70") !=
	      std::string::npos);
	CHECK(repaired_text.find("backlight_invert = false") !=
	      std::string::npos);
	CHECK(repaired_text.find("auto_sleep_enabled = false") !=
	      std::string::npos);
}

TEST_CASE("app config saves runtime switch and panel settings")
{
	TempDir temp;
	const auto path = temp.config_path();

	auto config = default_app_config();
	config.fonts.ssid_image_cache_kb = 2048;
	config.experimental.ssid_scroll = SsidScrollMode::On;
	config.screenshot.enabled = true;
	config.screenshot.signal = ScreenshotSignal::User2;
	config.screenshot.output_directory = "/var/lib/kvm/screenshots";
	config.ui.enable_settings_optimistic_actions = true;
	config.ui.wifi_loading_progress = true;
	config.nanokvm.video_enabled = true;
	config.nanokvm.hid_enabled = true;
	config.nanokvm.backlight_percent = 20;
	config.nanokvm.backlight_invert = true;
	config.nanokvm.auto_sleep_enabled = true;

	std::string error;
	REQUIRE(save_app_config(config, error, path));

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.config.nanokvm.video_enabled);
	CHECK(result.config.nanokvm.hid_enabled);
	CHECK(result.config.nanokvm.backlight_percent == 20);
	CHECK(result.config.nanokvm.backlight_invert);
	CHECK(result.config.nanokvm.auto_sleep_enabled);
	CHECK(result.config.fonts.ssid_image_cache_kb == 2048);
	CHECK(result.config.experimental.ssid_scroll == SsidScrollMode::On);
	CHECK(result.config.screenshot.enabled);
	CHECK(result.config.screenshot.signal == ScreenshotSignal::User2);
	CHECK(result.config.screenshot.output_directory ==
	      "/var/lib/kvm/screenshots");
	CHECK(result.config.ui.enable_settings_optimistic_actions);
	CHECK(result.config.ui.wifi_loading_progress);
	const auto text = read_text(path);
	CHECK(text.find("wifi_config_mode") == std::string::npos);
	CHECK(text.find("video_enabled = true") != std::string::npos);
	CHECK(text.find("hid_enabled = true") != std::string::npos);
	CHECK(text.find("ncm_enabled") == std::string::npos);
	CHECK(text.find("backlight_percent = 20") != std::string::npos);
	CHECK(text.find("backlight_invert = true") != std::string::npos);
	CHECK(text.find("auto_sleep_enabled = true") != std::string::npos);
	CHECK(text.find("ssid_image_cache_kb = 2048") != std::string::npos);
	CHECK(text.find("ssid_scroll = 'on'") != std::string::npos);
	CHECK(text.find("signal = 'SIGUSR2'") != std::string::npos);
	CHECK(text.find("output_directory = '/var/lib/kvm/screenshots'") !=
	      std::string::npos);
	CHECK(text.find("enable_settings_optimistic_actions = true") !=
	      std::string::npos);
	CHECK(text.find("wifi_loading_progress = true") != std::string::npos);
	CHECK(text.find("apps_dir") == std::string::npos);
	CHECK(text.find("demo_script") == std::string::npos);
}

TEST_CASE("app config repairs unsafe HTTP bind by restoring loopback defaults")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path,
		   R"toml(
[display]
framebuffer_device = "/dev/fb0"

[touch]
device = "/dev/input/event0"
disabled = false

[http_api]
enabled = true
bind_host = "0.0.0.0"
port = 65501
token = ""

)toml");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.config.http_api.bind_host ==
	      app_config_defaults::kHttpApiBindHost);
	CHECK(result.config.http_api.token ==
	      app_config_defaults::kHttpApiToken);
}

TEST_CASE("app config backs up invalid TOML and writes defaults")
{
	TempDir temp;
	const auto path = temp.config_path();
	write_text(path, "[display\nframebuffer_device = \"/dev/fb1\"\n");

	const auto result = load_or_repair_app_config(path);

	REQUIRE(result.ok);
	CHECK(result.repaired);
	CHECK(result.replaced_invalid);
	REQUIRE(result.backup_path.has_value());
	CHECK(std::filesystem::exists(*result.backup_path));
	CHECK(file_mode(*result.backup_path) == 0600);
	CHECK(std::filesystem::exists(path));
	CHECK(file_mode(path) == 0600);
	CHECK(result.config.display.framebuffer_device ==
	      app_config_defaults::kFramebufferDevice);

	const auto default_text = read_text(path);
	CHECK(default_text.find("/dev/fb0") != std::string::npos);
}

TEST_CASE("app config refuses symlink and oversized config files")
{
	TempDir temp;
	const auto path = temp.config_path();
	const auto target = path.parent_path() / "target.toml";
	write_text(target, render_app_config_toml(default_app_config()));
	std::filesystem::create_symlink(target, path);

	const auto symlink_result = load_or_repair_app_config(path);
	CHECK_FALSE(symlink_result.ok);

	std::filesystem::remove(path);
	std::filesystem::create_symlink(path.parent_path() / "missing.toml",
					path);

	const auto dangling_symlink_result = load_or_repair_app_config(path);
	CHECK_FALSE(dangling_symlink_result.ok);

	std::filesystem::remove(path);
	write_text(path, std::string(64U * 1024U + 1U, ' '));

	const auto oversized_result = load_or_repair_app_config(path);
	CHECK_FALSE(oversized_result.ok);
}
