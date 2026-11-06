#include "lvgl_main/nanokvm_runtime.hpp"

#include <filesystem>
#include <string>
#include <utility>
#include <vector>

#include <spdlog/spdlog.h>

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

#include "lvgl_main/backlight_control.hpp"
#include "lvgl_main/settings_cli.hpp"

namespace lvgl_main
{

#if !NANOKVM_GO_RELEASE_BUILD
class NoopSettingsCliActions final : public SettingsCliRuntimeActions {
    public:
	bool set_video_enabled(bool, std::string &error) override
	{
		return fail(error);
	}

	bool set_hid_enabled(bool, std::string &error) override
	{
		return fail(error);
	}

	bool set_ncm_enabled(bool, std::string &error) override
	{
		return fail(error);
	}

	bool set_ssh_enabled(bool, std::string &error) override
	{
		return fail(error);
	}

	bool set_wifi_ap_enabled(bool, std::string &error) override
	{
		return fail(error);
	}

	bool reset_wifi(std::string &error) override
	{
		return fail(error);
	}

	bool set_backlight_percent(
		int, std::vector<std::pair<std::string, std::string>> &,
		std::string &error) override
	{
		return fail(error);
	}

	bool set_auto_sleep_enabled(
		bool, std::vector<std::pair<std::string, std::string>> &,
		std::string &error) override
	{
		return fail(error);
	}

	bool about_reset_no_op(std::string &error) override
	{
		(void)error;
		spdlog::info(
			"[runtime] About Reset requested via Settings CLI (runtime unavailable, no-op)");
		return true;
	}

    private:
	static bool fail(std::string &error)
	{
		error = "runtime action unavailable";
		return false;
	}
};

class RuntimeSettingsCliActions final : public SettingsCliRuntimeActions {
    public:
	explicit RuntimeSettingsCliActions(NanoKvmRuntimeService &runtime)
		: _runtime(runtime)
	{
	}

	bool set_video_enabled(bool enabled, std::string &error) override
	{
		return _runtime.settings_cli_set_video_enabled(enabled, error);
	}

	bool set_hid_enabled(bool enabled, std::string &error) override
	{
		return _runtime.settings_cli_set_hid_enabled(enabled, error);
	}

	bool set_ncm_enabled(bool enabled, std::string &error) override
	{
		return _runtime.settings_cli_set_ncm_enabled(enabled, error);
	}

	bool set_ssh_enabled(bool enabled, std::string &error) override
	{
		return _runtime.settings_cli_set_ssh_enabled(enabled, error);
	}

	bool set_wifi_ap_enabled(bool enabled, std::string &error) override
	{
		return _runtime.settings_cli_set_wifi_ap_enabled(enabled,
								 error);
	}

	bool reset_wifi(std::string &error) override
	{
		return _runtime.settings_cli_reset_wifi(error);
	}

	bool set_backlight_percent(
		int percent,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) override
	{
		return _runtime.settings_cli_set_backlight_percent(
			percent, fields, error);
	}

	bool set_auto_sleep_enabled(
		bool enabled,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) override
	{
		return _runtime.settings_cli_set_auto_sleep_enabled(
			enabled, fields, error);
	}

	bool about_reset_no_op(std::string &error) override
	{
		(void)error;
		_runtime.about_reset_no_op();
		return true;
	}

    private:
	NanoKvmRuntimeService &_runtime;
};

bool settings_cli_action_needs_runtime(const SettingsCliCommand &command)
{
	if (command.dry_run) {
		return false;
	}
	switch (command.action) {
	case SettingsCliAction::Video:
	case SettingsCliAction::Hid:
	case SettingsCliAction::Ncm:
	case SettingsCliAction::Ssh:
	case SettingsCliAction::WifiAp:
	case SettingsCliAction::WifiReset:
	case SettingsCliAction::Brightness:
	case SettingsCliAction::Sleep:
		return true;
	case SettingsCliAction::None:
	case SettingsCliAction::ListActions:
	case SettingsCliAction::Status:
		return false;
	case SettingsCliAction::AboutReset:
		return !command.dry_run;
	}
	return false;
}

SettingsCliExecutionResult
execute_settings_cli_command(AppConfig &config,
			     const std::filesystem::path &config_path,
			     const SettingsCliCommand &command,
			     NcmBootMarkerPaths ncm_marker_paths)
{
	SettingsCliExecutionResult result;
	result.action = command.action;
	result.ok = true;
	result.exit_code = 0;
	result.status = command.dry_run ? "dry-run" : "ok";

	auto add_bool_field = [&result](std::string key, bool value) {
		result.fields.emplace_back(std::move(key),
					   value ? "true" : "false");
	};

	switch (command.action) {
	case SettingsCliAction::ListActions:
		result.items = {
			"navigation:wifi",    "wifi-ap:on|off",
			"wifi-reset",	      "video:on|off",
			"hid:on|off",	      "ncm:on|off",
			"ssh:on|off",	      "navigation:screen-panel",
			"brightness:20..100", "sleep:on|off",
			"about-reset:no-op"
		};
		result.message =
			"Settings actions exported for non-UI validation";
		return result;

	case SettingsCliAction::Status: {
		const auto state = backlight::read_backlight_state(
			config.nanokvm.backlight_root,
			config.nanokvm.backlight_percent,
			config.nanokvm.backlight_invert);
		result.fields.emplace_back("config_path", config_path.string());
		result.fields.emplace_back(
			"backlight_percent",
			std::to_string(config.nanokvm.backlight_percent));
		add_bool_field("auto_sleep_enabled",
			       config.nanokvm.auto_sleep_enabled);
		add_bool_field("backlight_invert",
			       config.nanokvm.backlight_invert);
		add_bool_field("video_enabled", config.nanokvm.video_enabled);
		add_bool_field("hid_enabled", config.nanokvm.hid_enabled);
		const auto ncm_observation =
			observe_ncm_boot_markers(ncm_marker_paths);
		const auto ncm_value = ncm_observation_enabled(ncm_observation);
		if (ncm_value) {
			add_bool_field("ncm_enabled", *ncm_value);
		} else {
			result.fields.emplace_back("ncm_state", "unknown");
			result.fields.emplace_back(
				"ncm_error",
				"boot marker observation inconclusive");
		}
		result.fields.emplace_back("backlight_root",
					   config.nanokvm.backlight_root);
		add_bool_field("backlight_available", state.available);
		if (state.available) {
			result.fields.emplace_back("backlight_raw",
						   std::to_string(state.raw));
			result.fields.emplace_back(
				"backlight_raw_max",
				std::to_string(state.raw_max));
			add_bool_field("screen_on", state.screen_on);
		} else if (!state.error.empty()) {
			result.fields.emplace_back("backlight_error",
						   state.error);
		}
		result.message =
			"Current persisted settings and readable backlight state";
		return result;
	}

	case SettingsCliAction::Brightness:
		if (command.dry_run) {
			const int percent = backlight::clamp_backlight_percent(
				command.int_value);
			result.fields.emplace_back("backlight_percent",
						   std::to_string(percent));
			result.fields.emplace_back(
				"backlight_raw",
				std::to_string(backlight::backlight_percent_to_raw(
					percent,
					backlight::kBacklightRawMaxDefault,
					config.nanokvm.backlight_invert)));
			result.message =
				"would write backlight brightness and persist config";
			return result;
		}
		break;

	case SettingsCliAction::None:
		result.ok = false;
		result.exit_code = 2;
		result.status = "parse-error";
		result.message = "No settings action specified";
		return result;

	case SettingsCliAction::Video:
	case SettingsCliAction::Hid:
	case SettingsCliAction::Ncm:
	case SettingsCliAction::Ssh:
	case SettingsCliAction::WifiAp:
	case SettingsCliAction::WifiReset:
	case SettingsCliAction::Sleep:
	case SettingsCliAction::AboutReset:
		break;
	}

	if (!settings_cli_action_needs_runtime(command)) {
		NoopSettingsCliActions noop_actions;
		return execute_settings_cli_runtime_action(noop_actions,
							   command);
	}

	NanoKvmRuntimeService runtime(config, {}, config_path);
	RuntimeSettingsCliActions runtime_actions(runtime);
	if (!runtime.start_settings_cli_actions()) {
		result.ok = false;
		result.exit_code = 4;
		result.status = "runtime-start-failed";
		result.message =
			"Failed to start runtime Settings CLI action worker";
		return result;
	}
	result = execute_settings_cli_runtime_action(runtime_actions, command);
	runtime.stop();
	return result;

	result.ok = false;
	result.exit_code = 2;
	result.status = "parse-error";
	result.message = "Unknown settings action";
	return result;
}
#endif


} // namespace lvgl_main
