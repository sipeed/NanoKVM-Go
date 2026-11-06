#pragma once

#include <filesystem>
#include <string>
#include <utility>
#include <vector>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"

namespace lvgl_main
{

enum class SettingsCliAction {
	None,
	ListActions,
	Status,
	Video,
	Hid,
	Ncm,
	Ssh,
	WifiAp,
	WifiReset,
	Brightness,
	Sleep,
	AboutReset,
};

struct SettingsCliCommand {
	std::filesystem::path config_path { std::filesystem::path(
		app_config_defaults::kConfigPath.data()) };
	bool config_path_set { false };
	SettingsCliAction action { SettingsCliAction::None };
	bool cli_requested { false };
	bool has_action { false };
	bool dry_run { false };
	bool json { false };
	bool debug_logging { false };
	bool debug_logging_set { false };
	bool touch_calibration { false };
	bool touch_calibration_set { false };
	bool touch_paint_test { false };
	bool touch_paint_test_set { false };
	bool touch_click_marker { false };
	bool touch_click_marker_set { false };
	bool wifi_keyboard { false };
	bool bool_value { false };
	int int_value { 0 };
};

struct SettingsCliParseResult {
	bool ok { false };
	int exit_code { 2 };
	std::string error;
	SettingsCliCommand command;
};

struct SettingsCliExecutionResult {
	bool ok { false };
	int exit_code { 0 };
	SettingsCliAction action { SettingsCliAction::None };
	std::string status;
	std::string message;
	std::vector<std::pair<std::string, std::string>> fields;
	std::vector<std::string> items;
};

struct SettingsNcmUsbFlagTargets {
	bool ncm_enabled { false };
	bool rndis_enabled { false };
};

class SettingsCliRuntimeActions {
    public:
	virtual ~SettingsCliRuntimeActions() = default;

	virtual bool set_video_enabled(bool enabled, std::string &error) = 0;
	virtual bool set_hid_enabled(bool enabled, std::string &error) = 0;
	virtual bool set_ncm_enabled(bool enabled, std::string &error) = 0;
	virtual bool set_ssh_enabled(bool enabled, std::string &error) = 0;
	virtual bool set_wifi_ap_enabled(bool enabled, std::string &error) = 0;
	virtual bool reset_wifi(std::string &error) = 0;
	virtual bool set_backlight_percent(
		int percent,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) = 0;
	virtual bool set_auto_sleep_enabled(
		bool enabled,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) = 0;
	virtual bool about_reset_no_op(std::string &error) = 0;
};

constexpr SettingsNcmUsbFlagTargets settings_ncm_usb_flag_targets(bool enabled)
{
	return { .ncm_enabled = enabled, .rndis_enabled = false };
}

SettingsCliParseResult parse_settings_cli_args(int argc, char *argv[]);
std::string settings_cli_action_name(SettingsCliAction action);
std::string settings_cli_help_text();
std::string format_settings_cli_result(const SettingsCliExecutionResult &result,
				       bool json);
SettingsCliExecutionResult
execute_settings_cli_runtime_action(SettingsCliRuntimeActions &runtime,
				    const SettingsCliCommand &command);
SettingsCliExecutionResult
execute_settings_cli_command(AppConfig &config,
			     const std::filesystem::path &config_path,
			     const SettingsCliCommand &command,
			     NcmBootMarkerPaths ncm_marker_paths =
				     production_ncm_boot_marker_paths());

} // namespace lvgl_main
