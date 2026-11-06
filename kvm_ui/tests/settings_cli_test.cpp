#include <algorithm>
#include <initializer_list>
#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/settings_cli.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

using namespace lvgl_main;

namespace
{

class RecordingSettingsRuntime final : public SettingsCliRuntimeActions {
    public:
	bool next_ok { true };
	std::string next_error;
	std::vector<std::string> calls;

	bool set_video_enabled(bool enabled, std::string &error) override
	{
		return record_bool("video", enabled, error);
	}

	bool set_hid_enabled(bool enabled, std::string &error) override
	{
		return record_bool("hid", enabled, error);
	}

	bool set_ncm_enabled(bool enabled, std::string &error) override
	{
		return record_bool("ncm", enabled, error);
	}

	bool set_ssh_enabled(bool enabled, std::string &error) override
	{
		return record_bool("ssh", enabled, error);
	}

	bool set_wifi_ap_enabled(bool enabled, std::string &error) override
	{
		return record_bool("wifi-ap", enabled, error);
	}

	bool reset_wifi(std::string &error) override
	{
		calls.emplace_back("wifi-reset");
		return finish(error);
	}

	bool set_backlight_percent(
		int percent,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) override
	{
		calls.emplace_back("brightness:" + std::to_string(percent));
		fields.emplace_back("backlight_percent",
				    std::to_string(percent));
		fields.emplace_back("backlight_raw", "123");
		return finish(error);
	}

	bool set_auto_sleep_enabled(
		bool enabled,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error) override
	{
		calls.emplace_back(std::string("sleep:") +
				   (enabled ? "on" : "off"));
		if (!enabled) {
			fields.emplace_back("backlight_raw", "200");
		}
		return finish(error);
	}

	bool about_reset_no_op(std::string &error) override
	{
		calls.emplace_back("about-reset");
		return finish(error);
	}

    private:
	bool record_bool(std::string name, bool enabled, std::string &error)
	{
		calls.emplace_back(std::move(name) + ":" +
				   (enabled ? "on" : "off"));
		return finish(error);
	}

	bool finish(std::string &error) const
	{
		if (next_ok) {
			return true;
		}
		error = next_error.empty() ? "runtime action failed" :
					     next_error;
		return false;
	}
};

SettingsCliParseResult parse(std::initializer_list<const char *> args)
{
	std::vector<char *> argv;
	argv.reserve(args.size());
	for (const char *arg : args) {
		argv.push_back(const_cast<char *>(arg));
	}
	return parse_settings_cli_args(static_cast<int>(argv.size()),
				       argv.data());
}

SettingsCliCommand bool_command(SettingsCliAction action, bool enabled)
{
	SettingsCliCommand command;
	command.action = action;
	command.has_action = true;
	command.bool_value = enabled;
	return command;
}

} // namespace

TEST_CASE("settings CLI parses brightness with alternate config")
{
	const auto result = parse({ "kvm_ui", "--config", "/tmp/a.toml",
				    "--settings-brightness", "50" });

	REQUIRE(result.ok);
	CHECK(result.command.has_action);
	CHECK(result.command.config_path == "/tmp/a.toml");
	CHECK(result.command.config_path_set);
	CHECK(result.command.action == SettingsCliAction::Brightness);
	CHECK(result.command.int_value == 50);
}

TEST_CASE("settings CLI parses sleep JSON mode")
{
	const auto result =
		parse({ "kvm_ui", "--settings-sleep", "on", "--json" });

	REQUIRE(result.ok);
	CHECK(result.command.has_action);
	CHECK(result.command.action == SettingsCliAction::Sleep);
	CHECK(result.command.bool_value);
	CHECK(result.command.json);
}

TEST_CASE("settings NCM flag targets enable only NCM")
{
	const SettingsNcmUsbFlagTargets enabled =
		settings_ncm_usb_flag_targets(true);
	CHECK(enabled.ncm_enabled);
	CHECK_FALSE(enabled.rndis_enabled);

	const SettingsNcmUsbFlagTargets disabled =
		settings_ncm_usb_flag_targets(false);
	CHECK_FALSE(disabled.ncm_enabled);
	CHECK_FALSE(disabled.rndis_enabled);
}

TEST_CASE("settings CLI rejects multiple actions")
{
	const auto result = parse(
		{ "kvm_ui", "--settings-status", "--settings-sleep", "off" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects out of range brightness")
{
	const auto below_range =
		parse({ "kvm_ui", "--settings-brightness", "19" });
	const auto above_range =
		parse({ "kvm_ui", "--settings-brightness", "101" });

	CHECK_FALSE(below_range.ok);
	CHECK(below_range.exit_code == 2);
	CHECK_FALSE(above_range.ok);
	CHECK(above_range.exit_code == 2);
}

TEST_CASE("settings CLI rejects and does not list removed mDNS action")
{
	const auto parsed = parse({ "kvm_ui", "--settings-mdns", "on" });
	CHECK_FALSE(parsed.ok);
	CHECK(parsed.exit_code == 2);
	CHECK(parsed.error.find("--settings-mdns") != std::string::npos);

	const auto help = settings_cli_help_text();
	CHECK(help.find("--settings-mdns") == std::string::npos);

#if !NANOKVM_GO_RELEASE_BUILD
	SettingsCliCommand command;
	command.action = SettingsCliAction::ListActions;
	command.has_action = true;
	AppConfig config;
	const auto actions = execute_settings_cli_command(
		config, "/tmp/kvm_ui.toml", command);
	CHECK(std::find(actions.items.begin(), actions.items.end(),
			"mdns:on|off") == actions.items.end());
#endif
}

TEST_CASE("settings CLI list actions is a standalone action")
{
	const auto result = parse({ "kvm_ui", "--settings-list-actions" });

	REQUIRE(result.ok);
	CHECK(result.command.has_action);
	CHECK(result.command.action == SettingsCliAction::ListActions);
}

TEST_CASE("settings CLI accepts config-only UI startup override")
{
	const auto result = parse({ "kvm_ui", "--config", "/tmp/ui.toml" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.config_path == "/tmp/ui.toml");
	CHECK(result.command.config_path_set);
}

TEST_CASE("settings CLI distinguishes an inherited default config path")
{
	const auto result = parse({ "kvm_ui", "--wifi-keyboard" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.config_path_set);
}

TEST_CASE("settings CLI accepts debug flag for UI startup")
{
	const auto result = parse({ "kvm_ui", "-d" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.debug_logging);
	CHECK(result.command.debug_logging_set);
}

TEST_CASE("settings CLI reserves the version alias for top-level handling")
{
	const auto result = parse({ "kvm_ui", "-v" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
	CHECK(result.error.find("-v") != std::string::npos);
	CHECK_FALSE(result.command.has_action);
	CHECK_FALSE(result.command.debug_logging);
	CHECK_FALSE(result.command.debug_logging_set);

	const auto help = settings_cli_help_text();
	CHECK(help.find("-v, -d") == std::string::npos);
	CHECK(help.find("  -d ") != std::string::npos);
}

TEST_CASE("settings CLI accepts touch calibration startup mode")
{
	const auto result = parse(
		{ "kvm_ui", "--config", "/tmp/ui.toml", "--touch-calibrate" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.touch_calibration);
	CHECK(result.command.touch_calibration_set);
	CHECK_FALSE(result.command.touch_paint_test_set);
	CHECK_FALSE(result.command.touch_click_marker_set);
	CHECK(result.command.config_path == "/tmp/ui.toml");
}

TEST_CASE("settings CLI accepts touch paint test startup mode")
{
	const auto result = parse(
		{ "kvm_ui", "--config", "/tmp/ui.toml", "--touch-paint-test" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.touch_paint_test);
	CHECK(result.command.touch_paint_test_set);
	CHECK_FALSE(result.command.touch_calibration_set);
	CHECK_FALSE(result.command.touch_click_marker_set);
	CHECK(result.command.config_path == "/tmp/ui.toml");
}

TEST_CASE("settings CLI accepts touch click marker for normal UI")
{
	const auto result = parse({ "kvm_ui", "--config", "/tmp/ui.toml",
				    "--touch-click-marker" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.touch_click_marker);
	CHECK(result.command.touch_click_marker_set);
	CHECK_FALSE(result.command.touch_calibration_set);
	CHECK_FALSE(result.command.touch_paint_test_set);
	CHECK(result.command.config_path == "/tmp/ui.toml");
}

TEST_CASE("settings CLI accepts WiFi keyboard startup mode")
{
	const auto result = parse(
		{ "kvm_ui", "--config", "/tmp/ui.toml", "--wifi-keyboard" });

	REQUIRE(result.ok);
	CHECK_FALSE(result.command.has_action);
	CHECK(result.command.wifi_keyboard);
	CHECK(result.command.config_path == "/tmp/ui.toml");
}

TEST_CASE("settings CLI describes WiFi keyboard as runtime-backed startup mode")
{
	const auto help = settings_cli_help_text();

	CHECK(help.find("--wifi-keyboard") != std::string::npos);
	CHECK(help.find("Start on WiFi SSID/password keyboard pages") !=
	      std::string::npos);
	CHECK(help.find("without runtime hardware actions") ==
	      std::string::npos);
}

TEST_CASE("settings CLI rejects touch calibration with settings action")
{
	const auto result =
		parse({ "kvm_ui", "--touch-calibrate", "--settings-status" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects touch click marker with settings action")
{
	const auto result = parse(
		{ "kvm_ui", "--touch-click-marker", "--settings-status" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects WiFi keyboard with settings action")
{
	const auto result =
		parse({ "kvm_ui", "--wifi-keyboard", "--settings-status" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects WiFi keyboard with touch test mode")
{
	const auto result =
		parse({ "kvm_ui", "--wifi-keyboard", "--touch-paint-test" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects touch click marker with touch test mode")
{
	const auto result = parse(
		{ "kvm_ui", "--touch-click-marker", "--touch-calibrate" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI rejects multiple touch test modes")
{
	const auto result =
		parse({ "kvm_ui", "--touch-calibrate", "--touch-paint-test" });

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 2);
}

TEST_CASE("settings CLI dispatches switch actions through runtime client")
{
	struct Case {
		SettingsCliAction action;
		const char *expected_call;
		const char *expected_message;
	};

	const Case cases[] = {
		{ SettingsCliAction::Video, "video:on",
		  "Video switch updated" },
		{ SettingsCliAction::Hid, "hid:on", "HID switch updated" },
		{ SettingsCliAction::Ncm, "ncm:on", "NCM switch updated" },
		{ SettingsCliAction::Ssh, "ssh:on",
		  "SSH systemd units updated" },
		{ SettingsCliAction::WifiAp, "wifi-ap:on",
		  "WiFi AP script executed" },
	};

	for (const auto &entry : cases) {
		RecordingSettingsRuntime runtime;
		const auto result = execute_settings_cli_runtime_action(
			runtime, bool_command(entry.action, true));

		REQUIRE(result.ok);
		CHECK(result.status == "ok");
		CHECK(result.message == entry.expected_message);
		CHECK(runtime.calls ==
		      std::vector<std::string> { entry.expected_call });
	}
}

TEST_CASE("settings CLI dispatches WiFi reset through runtime client")
{
	RecordingSettingsRuntime runtime;
	SettingsCliCommand command;
	command.action = SettingsCliAction::WifiReset;
	command.has_action = true;

	const auto result =
		execute_settings_cli_runtime_action(runtime, command);

	REQUIRE(result.ok);
	CHECK(result.message == "WiFi reset script executed");
	CHECK(runtime.calls == std::vector<std::string> { "wifi-reset" });
}

TEST_CASE("settings CLI dispatches AboutReset through runtime client")
{
	RecordingSettingsRuntime runtime;
	SettingsCliCommand command;
	command.action = SettingsCliAction::AboutReset;
	command.has_action = true;

	const auto result =
		execute_settings_cli_runtime_action(runtime, command);

	REQUIRE(result.ok);
	CHECK(result.status == "no-op");
	CHECK(result.message ==
	      "About Reset currently has no hardware or config action");
	CHECK(runtime.calls == std::vector<std::string> { "about-reset" });
}

TEST_CASE("settings CLI AboutReset dry-run does not call runtime")
{
	RecordingSettingsRuntime runtime;
	SettingsCliCommand command;
	command.action = SettingsCliAction::AboutReset;
	command.has_action = true;
	command.dry_run = true;

	const auto result =
		execute_settings_cli_runtime_action(runtime, command);

	REQUIRE(result.ok);
	CHECK(result.status == "dry-run");
	CHECK(runtime.calls.empty());
}

TEST_CASE("settings CLI dispatches brightness through runtime client")
{
	RecordingSettingsRuntime runtime;
	SettingsCliCommand command;
	command.action = SettingsCliAction::Brightness;
	command.has_action = true;
	command.int_value = 55;

	const auto result =
		execute_settings_cli_runtime_action(runtime, command);

	REQUIRE(result.ok);
	CHECK(result.message == "Backlight brightness updated");
	CHECK(runtime.calls == std::vector<std::string> { "brightness:55" });
	CHECK(result.fields ==
	      std::vector<std::pair<std::string, std::string>> {
		      { "backlight_percent", "55" },
		      { "backlight_raw", "123" } });
}

TEST_CASE("settings CLI dispatches sleep through runtime client")
{
	RecordingSettingsRuntime runtime;

	const auto result = execute_settings_cli_runtime_action(
		runtime, bool_command(SettingsCliAction::Sleep, false));

	REQUIRE(result.ok);
	CHECK(result.message == "Auto sleep disabled and backlight restored");
	CHECK(runtime.calls == std::vector<std::string> { "sleep:off" });
	CHECK(result.fields ==
	      std::vector<std::pair<std::string, std::string>> {
		      { "backlight_raw", "200" } });
}

TEST_CASE("settings CLI runtime action failure is surfaced")
{
	RecordingSettingsRuntime runtime;
	runtime.next_ok = false;
	runtime.next_error = "permission denied";

	const auto result = execute_settings_cli_runtime_action(
		runtime, bool_command(SettingsCliAction::Video, true));

	CHECK_FALSE(result.ok);
	CHECK(result.exit_code == 4);
	CHECK(result.status == "action-failed");
	CHECK(result.message == "permission denied");
	CHECK(runtime.calls == std::vector<std::string> { "video:on" });
}

TEST_CASE("settings CLI dry run avoids runtime client")
{
	RecordingSettingsRuntime runtime;
	auto command = bool_command(SettingsCliAction::Video, true);
	command.dry_run = true;

	const auto result =
		execute_settings_cli_runtime_action(runtime, command);

	REQUIRE(result.ok);
	CHECK(result.status == "dry-run");
	CHECK(result.message == "would set video on");
	CHECK(runtime.calls.empty());
}
