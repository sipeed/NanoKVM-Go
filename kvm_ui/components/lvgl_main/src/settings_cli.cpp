#include "lvgl_main/settings_cli.hpp"

#include <sstream>
#include <exception>
#include <string_view>
#include <utility>

#include <nlohmann/json.hpp>

#include "lvgl_main/backlight_control.hpp"

namespace lvgl_main
{

namespace
{

using json = nlohmann::json;

bool is_bool_value(std::string_view value, bool &parsed)
{
	if (value == "on") {
		parsed = true;
		return true;
	}
	if (value == "off") {
		parsed = false;
		return true;
	}
	return false;
}

bool parse_int(std::string_view value, int &parsed)
{
	try {
		std::size_t consumed = 0;
		const int number = std::stoi(std::string(value), &consumed);
		if (consumed != value.size()) {
			return false;
		}
		parsed = number;
		return true;
	} catch (const std::exception &) {
		return false;
	}
}

void set_parse_error(SettingsCliParseResult &result, std::string error)
{
	result.ok = false;
	result.exit_code = 2;
	result.error = std::move(error);
}

bool set_action(SettingsCliParseResult &result, SettingsCliAction action)
{
	result.command.cli_requested = true;
	if (result.command.has_action) {
		set_parse_error(
			result,
			"Only one --settings-* action is allowed per invocation");
		return false;
	}
	result.command.has_action = true;
	result.command.action = action;
	return true;
}

bool require_value(int argc, char *argv[], int &index, std::string_view option,
		   std::string_view &value, SettingsCliParseResult &result)
{
	if (index + 1 >= argc || argv[index + 1] == nullptr) {
		set_parse_error(result,
				std::string(option) + " requires a value");
		return false;
	}
	++index;
	value = argv[index];
	return true;
}

bool parse_bool_action(int argc, char *argv[], int &index,
		       std::string_view option, SettingsCliAction action,
		       SettingsCliParseResult &result)
{
	if (!set_action(result, action)) {
		return false;
	}
	std::string_view value;
	if (!require_value(argc, argv, index, option, value, result)) {
		return false;
	}
	bool parsed = false;
	if (!is_bool_value(value, parsed)) {
		set_parse_error(result,
				std::string(option) + " expects on|off");
		return false;
	}
	result.command.bool_value = parsed;
	return true;
}

json fields_to_json(const SettingsCliExecutionResult &result)
{
	json data = json::object();
	for (const auto &[key, value] : result.fields) {
		data[key] = value;
	}
	if (!result.items.empty()) {
		data["items"] = result.items;
	}
	return data;
}

void fail_runtime_action(SettingsCliExecutionResult &result,
			 std::string message)
{
	result.ok = false;
	result.exit_code = 4;
	result.status = "action-failed";
	result.message = std::move(message);
}

std::string bool_target_text(bool enabled)
{
	return enabled ? "on" : "off";
}

SettingsCliExecutionResult dry_run_result(SettingsCliAction action,
					  std::string message)
{
	SettingsCliExecutionResult result;
	result.ok = true;
	result.exit_code = 0;
	result.action = action;
	result.status = "dry-run";
	result.message = std::move(message);
	return result;
}

} // namespace

SettingsCliParseResult parse_settings_cli_args(int argc, char *argv[])
{
	SettingsCliParseResult result;
	result.ok = true;
	result.exit_code = 0;

	for (int i = 1; i < argc; ++i) {
		if (argv[i] == nullptr) {
			continue;
		}
		const std::string_view arg(argv[i]);

		if (arg == "-d") {
			result.command.debug_logging = true;
			result.command.debug_logging_set = true;
		} else if (arg == "--config") {
			std::string_view value;
			if (!require_value(argc, argv, i, arg, value, result)) {
				return result;
			}
			result.command.config_path =
				std::filesystem::path(std::string(value));
			result.command.config_path_set = true;
		} else if (arg == "--touch-calibrate") {
			result.command.touch_calibration = true;
			result.command.touch_calibration_set = true;
		} else if (arg == "--touch-paint-test") {
			result.command.touch_paint_test = true;
			result.command.touch_paint_test_set = true;
		} else if (arg == "--touch-click-marker") {
			result.command.touch_click_marker = true;
			result.command.touch_click_marker_set = true;
		} else if (arg == "--wifi-keyboard") {
			result.command.wifi_keyboard = true;
		} else if (arg == "--settings-dry-run") {
			result.command.cli_requested = true;
			result.command.dry_run = true;
		} else if (arg == "--json") {
			result.command.cli_requested = true;
			result.command.json = true;
		} else if (arg == "--settings-list-actions") {
			if (!set_action(result,
					SettingsCliAction::ListActions)) {
				return result;
			}
		} else if (arg == "--settings-status") {
			if (!set_action(result, SettingsCliAction::Status)) {
				return result;
			}
		} else if (arg == "--settings-video") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::Video,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-hid") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::Hid,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-ncm") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::Ncm,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-ssh") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::Ssh,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-wifi-ap") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::WifiAp,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-wifi-reset") {
			if (!set_action(result, SettingsCliAction::WifiReset)) {
				return result;
			}
		} else if (arg == "--settings-brightness") {
			if (!set_action(result,
					SettingsCliAction::Brightness)) {
				return result;
			}
			std::string_view value;
			if (!require_value(argc, argv, i, arg, value, result)) {
				return result;
			}
			int parsed = 0;
			if (!parse_int(value, parsed) ||
			    parsed < backlight::kBacklightUiMinPercent ||
			    parsed > backlight::kBacklightUiMaxPercent) {
				set_parse_error(
					result,
					"--settings-brightness expects 20..100");
				return result;
			}
			result.command.int_value = parsed;
		} else if (arg == "--settings-sleep") {
			if (!parse_bool_action(argc, argv, i, arg,
					       SettingsCliAction::Sleep,
					       result)) {
				return result;
			}
		} else if (arg == "--settings-about-reset") {
			if (!set_action(result,
					SettingsCliAction::AboutReset)) {
				return result;
			}
		} else if (arg == "--help" || arg == "-h") {
			continue;
		} else {
			set_parse_error(result, "Unknown argument: " +
							std::string(arg));
			return result;
		}
	}

	if (result.command.touch_calibration &&
	    result.command.touch_paint_test) {
		set_parse_error(
			result,
			"--touch-calibrate cannot be combined with --touch-paint-test");
	} else if (result.command.touch_click_marker &&
		   (result.command.touch_calibration ||
		    result.command.touch_paint_test)) {
		set_parse_error(
			result,
			"--touch-click-marker only supports normal UI mode");
	} else if (result.command.wifi_keyboard &&
		   (result.command.touch_calibration ||
		    result.command.touch_paint_test)) {
		set_parse_error(
			result,
			"--wifi-keyboard cannot be combined with touch test modes");
	} else if ((result.command.touch_calibration ||
		    result.command.touch_paint_test) &&
		   (result.command.cli_requested ||
		    result.command.has_action)) {
		set_parse_error(
			result,
			"Touch test modes cannot be combined with --settings-* options");
	} else if (result.command.wifi_keyboard &&
		   (result.command.cli_requested ||
		    result.command.has_action)) {
		set_parse_error(
			result,
			"--wifi-keyboard cannot be combined with --settings-* options");
	} else if (result.command.touch_click_marker &&
		   (result.command.cli_requested ||
		    result.command.has_action)) {
		set_parse_error(
			result,
			"--touch-click-marker cannot be combined with --settings-* options");
	} else if (result.command.cli_requested && !result.command.has_action) {
		set_parse_error(
			result,
			"Settings CLI options require one --settings-* action");
	}
	return result;
}

std::string settings_cli_action_name(SettingsCliAction action)
{
	switch (action) {
	case SettingsCliAction::None:
		return "none";
	case SettingsCliAction::ListActions:
		return "list-actions";
	case SettingsCliAction::Status:
		return "status";
	case SettingsCliAction::Video:
		return "video";
	case SettingsCliAction::Hid:
		return "hid";
	case SettingsCliAction::Ncm:
		return "ncm";
	case SettingsCliAction::Ssh:
		return "ssh";
	case SettingsCliAction::WifiAp:
		return "wifi-ap";
	case SettingsCliAction::WifiReset:
		return "wifi-reset";
	case SettingsCliAction::Brightness:
		return "brightness";
	case SettingsCliAction::Sleep:
		return "sleep";
	case SettingsCliAction::AboutReset:
		return "about-reset";
	}
	return "unknown";
}

std::string settings_cli_help_text()
{
	return R"help(Settings CLI mode, one action per invocation:
  --config <path>                         Use an alternate kvm_ui.toml path.
  --touch-click-marker                    In normal UI mode, mark the last touch release with an inverted dot.
  --wifi-keyboard                         Start on WiFi SSID/password keyboard pages.
  --settings-list-actions                 List exported Settings actions.
  --settings-status                       Print current persisted/runtime-readable state.
  --settings-video on|off                 Same function as Settings Video.
  --settings-hid on|off                   Same function as Settings Control.
  --settings-ncm on|off                   Same function as Settings NCM.
  --settings-ssh on|off                   Same function as Settings SSH.
  --settings-wifi-ap on|off               Same function as WiFi subpage ON/OFF.
  --settings-wifi-reset                   Same function as WiFi Reset.
  --settings-brightness <20..100>         Same function as Panel brightness slider.
  --settings-sleep on|off                 Same function as Panel Sleep policy.
  --settings-about-reset                  Export current hidden About Reset no-op action.
  --settings-dry-run                      Validate action without side effects.
  --json                                  Print JSON result.
  -d                                      Print debug and info logs.
	)help";
}

SettingsCliExecutionResult
execute_settings_cli_runtime_action(SettingsCliRuntimeActions &runtime,
				    const SettingsCliCommand &command)
{
	SettingsCliExecutionResult result;
	result.action = command.action;
	result.ok = true;
	result.exit_code = 0;
	result.status = command.dry_run ? "dry-run" : "ok";

	auto call_bool = [&](auto &&fn, std::string success_message) {
		if (command.dry_run) {
			return result;
		}
		std::string error;
		if (!fn(command.bool_value, error)) {
			fail_runtime_action(result, std::move(error));
			return result;
		}
		result.message = std::move(success_message);
		return result;
	};

	switch (command.action) {
	case SettingsCliAction::Video:
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set video ") +
					bool_target_text(command.bool_value));
		}
		return call_bool(
			[&](bool enabled, std::string &error) {
				return runtime.set_video_enabled(enabled,
								 error);
			},
			"Video switch updated");

	case SettingsCliAction::Hid:
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set HID ") +
					bool_target_text(command.bool_value));
		}
		return call_bool(
			[&](bool enabled, std::string &error) {
				return runtime.set_hid_enabled(enabled, error);
			},
			"HID switch updated");

	case SettingsCliAction::Ncm:
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set NCM ") +
					bool_target_text(command.bool_value));
		}
		return call_bool(
			[&](bool enabled, std::string &error) {
				return runtime.set_ncm_enabled(enabled, error);
			},
			"NCM switch updated");

	case SettingsCliAction::Ssh:
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set SSH ") +
					bool_target_text(command.bool_value));
		}
		return call_bool(
			[&](bool enabled, std::string &error) {
				return runtime.set_ssh_enabled(enabled, error);
			},
			"SSH systemd units updated");

	case SettingsCliAction::WifiAp:
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set WiFi AP ") +
					bool_target_text(command.bool_value));
		}
		return call_bool(
			[&](bool enabled, std::string &error) {
				return runtime.set_wifi_ap_enabled(enabled,
								   error);
			},
			"WiFi AP script executed");

	case SettingsCliAction::WifiReset: {
		if (command.dry_run) {
			return dry_run_result(command.action,
					      "would restart WiFi AP");
		}
		std::string error;
		if (!runtime.reset_wifi(error)) {
			fail_runtime_action(result, std::move(error));
			return result;
		}
		result.message = "WiFi reset script executed";
		return result;
	}

	case SettingsCliAction::Brightness: {
		const int percent =
			backlight::clamp_backlight_percent(command.int_value);
		if (command.dry_run) {
			result.fields.emplace_back("backlight_percent",
						   std::to_string(percent));
			result.fields.emplace_back(
				"backlight_raw",
				std::to_string(
					backlight::backlight_percent_to_raw(
						percent)));
			result.message =
				"would write backlight brightness and persist config";
			return result;
		}
		std::string error;
		if (!runtime.set_backlight_percent(percent, result.fields,
						   error)) {
			fail_runtime_action(result, std::move(error));
			return result;
		}
		result.message = "Backlight brightness updated";
		return result;
	}

	case SettingsCliAction::Sleep: {
		if (command.dry_run) {
			return dry_run_result(
				command.action,
				std::string("would set auto sleep ") +
					bool_target_text(command.bool_value));
		}
		std::string error;
		if (!runtime.set_auto_sleep_enabled(command.bool_value,
						    result.fields, error)) {
			fail_runtime_action(result, std::move(error));
			return result;
		}
		result.message =
			command.bool_value ?
				"Auto sleep enabled; UI runtime sleeps after 30s idle" :
				"Auto sleep disabled and backlight restored";
		return result;
	}

	case SettingsCliAction::AboutReset: {
		if (command.dry_run) {
			result.status = "dry-run";
			result.message =
				"About Reset currently has no hardware or config action";
			return result;
		}
		std::string error;
		if (!runtime.about_reset_no_op(error)) {
			fail_runtime_action(result, std::move(error));
			return result;
		}
		result.status = "no-op";
		result.message =
			"About Reset currently has no hardware or config action";
		return result;
	}

	case SettingsCliAction::None:
	case SettingsCliAction::ListActions:
	case SettingsCliAction::Status:
		result.ok = false;
		result.exit_code = 2;
		result.status = "parse-error";
		result.message = "Action is not a runtime Settings mutation";
		return result;
	}

	result.ok = false;
	result.exit_code = 2;
	result.status = "parse-error";
	result.message = "Unknown Settings CLI action";
	return result;
}

std::string format_settings_cli_result(const SettingsCliExecutionResult &result,
				       bool json_output)
{
	if (json_output) {
		json output;
		output["ok"] = result.ok;
		output["exit_code"] = result.exit_code;
		output["action"] = settings_cli_action_name(result.action);
		output["status"] = result.status;
		output["message"] = result.message;
		output["data"] = fields_to_json(result);
		return output.dump() + "\n";
	}

	std::ostringstream out;
	out << settings_cli_action_name(result.action) << ": " << result.status;
	if (!result.message.empty()) {
		out << " - " << result.message;
	}
	out << '\n';
	for (const auto &item : result.items) {
		out << item << '\n';
	}
	for (const auto &[key, value] : result.fields) {
		out << key << "=" << value << '\n';
	}
	return out.str();
}

} // namespace lvgl_main
