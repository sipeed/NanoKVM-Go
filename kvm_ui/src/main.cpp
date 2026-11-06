#include <cstring>
#include <filesystem>
#include <iostream>
#include <memory>
#include <utility>

#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

#include <lvgl_main/app_config_defaults.hpp>
#include <lvgl_main/build_info.hpp>
#include <lvgl_main/lvgl_main.hpp>
#include <lvgl_main/screenshot_cli.hpp>
#if NANOKVM_GO_SIMULATOR_BUILD
#include <lvgl_main/simulator_build_config.hpp>
#include <lvgl_main/simulator_policy.hpp>
#endif
#if !NANOKVM_GO_RELEASE_BUILD
#include <lvgl_main/settings_cli.hpp>
#endif

namespace
{

struct EffectiveRuntimeOptions {
	bool debug_logging { false };
#if !NANOKVM_GO_RELEASE_BUILD
	bool touch_click_marker { false };
	bool touch_calibration { false };
	bool touch_paint_test { false };
	bool wifi_keyboard { false };
#endif
};

#if !NANOKVM_GO_RELEASE_BUILD
EffectiveRuntimeOptions make_effective_runtime_options(
	const lvgl_main::RuntimeOptionsConfig &config_runtime,
	const lvgl_main::SettingsCliCommand &command)
{
	EffectiveRuntimeOptions options {
		.debug_logging = config_runtime.debug_logging,
		.touch_click_marker = config_runtime.touch_click_marker,
		.touch_calibration = config_runtime.touch_calibrate,
		.touch_paint_test = config_runtime.touch_paint_test,
		.wifi_keyboard = command.wifi_keyboard,
	};

	if (command.debug_logging_set) {
		options.debug_logging = command.debug_logging;
	}
	if (command.touch_click_marker_set || command.touch_calibration_set ||
	    command.touch_paint_test_set) {
		options.touch_click_marker = command.touch_click_marker;
		options.touch_calibration = command.touch_calibration;
		options.touch_paint_test = command.touch_paint_test;
	}
	if (command.wifi_keyboard) {
		options.touch_calibration = false;
		options.touch_paint_test = false;
	}
	return options;
}
#endif

void print_help(const char *program)
{
	const char *app = (program != nullptr && program[0] != '\0') ?
				  program :
				  lvgl_main::build_info::kAppName;
	std::cout
		<< "Usage: " << app
#if NANOKVM_GO_RELEASE_BUILD
		<< " [--help|-h] [-v|--version] [--screenshot]\n\n"
#else
		<< " [--help|-h] [-v|--version] [--screenshot] [-d] [--touch-click-marker] [--touch-calibrate] [--touch-paint-test] [--wifi-keyboard]\n\n"
#endif
		<< "Informational options:\n"
		<< "  -v, --version                        Print app version and git commit.\n\n"
		<< "Screenshot request:\n"
		<< "  --screenshot                         Send the configured screenshot signal to an already-running instance.\n"
		<< "                                       Must be used by itself; does not start the UI.\n\n"
#if !NANOKVM_GO_RELEASE_BUILD
		<< "Runtime options:\n"
		<< "  -d                                   Print debug and info logs.\n\n"
		<< "  --touch-click-marker                 In normal UI mode, mark the last touch release with an inverted dot.\n\n"
		<< "  --touch-calibrate                    Start the LVGL touch calibration drawing board.\n\n"
		<< "  --touch-paint-test                   Start the white-background touch paint coverage test.\n\n"
		<< "  --wifi-keyboard                      Start on WiFi SSID/password keyboard pages.\n\n"
		<< "    Debug and touch runtime options can also be set in [runtime]. Command line options override config values.\n\n"
#endif
		<< "Configuration file:\n"
#if NANOKVM_GO_SIMULATOR_BUILD
		<< "  " << lvgl_main::simulator_build_config::kDefaultConfigPath
		<< "\n"
		<< "    Simulator default; --config accepts only safe host paths.\n\n";
#else
		<< "  " << lvgl_main::app_config_defaults::kConfigPath << "\n"
		<< "    Created with defaults on first start when missing.\n"
		<< "    Invalid fields are repaired to defaults when possible.\n"
		<< "    Invalid TOML syntax is backed up and replaced with defaults.\n\n";
#endif
#if !NANOKVM_GO_RELEASE_BUILD
	std::cout << lvgl_main::settings_cli_help_text();
#endif
}

void print_version()
{
	std::cout << lvgl_main::build_info::kAppName << " "
		  << lvgl_main::build_info::kAppVersion << " (git "
		  << lvgl_main::build_info::kGitCommit << ")\n";
}

bool is_version_argument(const char *argument)
{
	return argument != nullptr && (std::strcmp(argument, "-v") == 0 ||
				       std::strcmp(argument, "--version") == 0);
}

bool is_help_argument(const char *argument)
{
	return argument != nullptr && (std::strcmp(argument, "--help") == 0 ||
				       std::strcmp(argument, "-h") == 0);
}

#if NANOKVM_GO_RELEASE_BUILD
bool parse_release_args(int argc, char *argv[], bool &version_requested,
			bool &help_requested)
{
	version_requested = false;
	help_requested = false;
	for (int i = 1; i < argc; ++i) {
		const char *arg = argv[i];
		if (arg == nullptr) {
			continue;
		}
		if (is_version_argument(arg)) {
			version_requested = true;
		} else if (is_help_argument(arg)) {
			help_requested = true;
		} else {
			return false;
		}
	}
	return true;
}
#endif

void configure_logging(const char *level)
{
	auto console_sink =
		std::make_shared<spdlog::sinks::stdout_color_sink_mt>();
	console_sink->set_color(spdlog::level::trace, console_sink->cyan);
	console_sink->set_color(spdlog::level::debug, console_sink->blue);
	console_sink->set_color(spdlog::level::info, console_sink->green);
	console_sink->set_color(spdlog::level::warn, console_sink->yellow);
	console_sink->set_color(spdlog::level::err, console_sink->red);
	console_sink->set_color(spdlog::level::critical, console_sink->red);

	auto logger = std::make_shared<spdlog::logger>("kvm_ui", console_sink);
	logger->set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%n] [%^%l%$] %v");
	spdlog::set_default_logger(std::move(logger));
	spdlog::set_level(spdlog::level::from_str(level));
}

} // namespace

int main(int argc, char *argv[])
{
	constexpr bool screenshot_cli_target_available =
#if NANOKVM_GO_SIMULATOR_BUILD
		false;
#else
		true;
#endif
	const auto screenshot_cli_result =
		lvgl_main::screenshot_cli::try_handle_screenshot_cli(
			argc, argv, screenshot_cli_target_available,
			lvgl_main::screenshot_control::kPidfilePath,
			lvgl_main::screenshot_control::system_ops());
	if (screenshot_cli_result.handled) {
		auto &output = screenshot_cli_result.exit_code == 0 ? std::cout :
								      std::cerr;
		output << screenshot_cli_result.message << "\n";
		return screenshot_cli_result.exit_code;
	}

#if NANOKVM_GO_RELEASE_BUILD
	bool release_version_requested = false;
	bool release_help_requested = false;
	if (!parse_release_args(argc, argv, release_version_requested,
				release_help_requested)) {
		return 2;
	}
	if (release_version_requested) {
		print_version();
		return 0;
	}
	if (release_help_requested) {
		print_help(argv[0]);
		return 0;
	}
#else
	for (int i = 1; i < argc; ++i) {
		const char *arg = argv[i];
		if (arg == nullptr) {
			continue;
		}
		if (is_version_argument(arg)) {
			print_version();
			return 0;
		}
	}

	for (int i = 1; i < argc; ++i) {
		const char *arg = argv[i];
		if (arg == nullptr) {
			continue;
		}
		if (is_help_argument(arg)) {
			print_help(argv[0]);
			return 0;
		}
	}
#endif

#if !NANOKVM_GO_RELEASE_BUILD
	auto cli_parse = lvgl_main::parse_settings_cli_args(argc, argv);
	if (!cli_parse.ok) {
		std::cerr << cli_parse.error << "\n";
		std::cerr << "Run with --help to see available options.\n";
		return cli_parse.exit_code;
	}
#if NANOKVM_GO_SIMULATOR_BUILD
	if (cli_parse.command.cli_requested || cli_parse.command.has_action) {
		std::cerr
			<< "Settings CLI actions are unavailable in simulator mode.\n";
		return 2;
	}
	if (cli_parse.command.touch_calibration ||
	    cli_parse.command.touch_paint_test) {
		std::cerr
			<< "Touch diagnostic modes are unavailable in simulator mode.\n";
		return 2;
	}
	std::string simulator_path_error;
	std::filesystem::path simulator_config_path;
	if (!lvgl_main::simulator_policy::resolve_config_path(
		    cli_parse.command.config_path,
		    cli_parse.command.config_path_set, simulator_config_path,
		    simulator_path_error)) {
		std::cerr << simulator_path_error << "\n";
		return 2;
	}
	cli_parse.command.config_path = std::move(simulator_config_path);
#endif

	const char *settings_cli_log_level =
		cli_parse.command.debug_logging ? "debug" : "warn";
	configure_logging(settings_cli_log_level);

#if !NANOKVM_GO_SIMULATOR_BUILD
	if (cli_parse.command.has_action) {
		lvgl_main::AppConfig config = lvgl_main::default_app_config();
		if (cli_parse.command.action !=
		    lvgl_main::SettingsCliAction::ListActions) {
			const auto config_result =
				lvgl_main::load_or_repair_app_config(
					cli_parse.command.config_path);
			if (!config_result.ok) {
				std::cerr << "Failed to load config "
					  << config_result.path.string() << ": "
					  << config_result.error << "\n";
				return 2;
			}
			for (const auto &warning : config_result.warnings) {
				std::cerr << "Warning: " << warning << "\n";
			}
			config = config_result.config;
		}

		auto result = lvgl_main::execute_settings_cli_command(
			config, cli_parse.command.config_path,
			cli_parse.command);
		std::cout << lvgl_main::format_settings_cli_result(
			result, cli_parse.command.json);
		return result.exit_code;
	}
#endif
#endif

#if NANOKVM_GO_RELEASE_BUILD
	const auto config_result = lvgl_main::load_or_repair_app_config();
	EffectiveRuntimeOptions effective_runtime {
		.debug_logging = config_result.effective_runtime.debug_logging,
	};
#else
	const auto config_result = lvgl_main::load_or_repair_app_config(
		cli_parse.command.config_path);
	auto effective_runtime = make_effective_runtime_options(
		config_result.effective_runtime, cli_parse.command);
#endif
	const char *log_level = effective_runtime.debug_logging ? "debug" :
								  "warn";
	// Configure process-wide spdlog after config load so
	// [runtime].debug_logging can apply, before workers/HTTP/LVGL.
	configure_logging(log_level);

	if (!config_result.ok) {
		spdlog::error("Failed to load config {}: {}",
			      config_result.path.string(), config_result.error);
		return 2;
	}

	spdlog::info("[main] Starting UI...");
	for (const auto &message : config_result.messages) {
		if (config_result.repaired || config_result.replaced_invalid) {
			spdlog::warn("{}", message);
		} else {
			spdlog::info("[main] {}", message);
		}
	}
	for (const auto &warning : config_result.warnings) {
		spdlog::warn("{}", warning);
	}
	spdlog::info("[main] Using config: {}", config_result.path.string());
	spdlog::info("[main] Using framebuffer: {}",
		     config_result.config.display.framebuffer_device);

	auto ui_config = config_result.config;
#if NANOKVM_GO_SIMULATOR_BUILD
	ui_config =
		lvgl_main::simulator_policy::make_effective_config(ui_config);
	effective_runtime.touch_click_marker =
		ui_config.runtime.touch_click_marker;
	effective_runtime.touch_calibration = false;
	effective_runtime.touch_paint_test = false;
#endif
#if !NANOKVM_GO_RELEASE_BUILD
	ui_config.runtime.touch_click_marker =
		effective_runtime.touch_click_marker;
	ui_config.runtime.touch_calibrate = effective_runtime.touch_calibration;
	ui_config.runtime.touch_paint_test = effective_runtime.touch_paint_test;
#endif

	int ret = 0;
#if !NANOKVM_GO_RELEASE_BUILD
	if (effective_runtime.touch_calibration ||
	    effective_runtime.touch_paint_test) {
		if (ui_config.touch.disabled) {
			spdlog::warn(
				"Touch input is disabled in config; enabling it for touch test mode");
		}
		if (effective_runtime.touch_calibration) {
			spdlog::info(
				"[main] Starting touch calibration drawing board via runtime path...");
		} else {
			spdlog::info(
				"[main] Starting touch paint coverage test via runtime path...");
		}
		ret = lvgl_ui_main(ui_config, config_result.path,
				   effective_runtime.touch_click_marker,
				   effective_runtime.wifi_keyboard);
	} else {
		// Run the LVGL UI
		if (effective_runtime.wifi_keyboard) {
			spdlog::info(
				"[main] Starting WiFi keyboard test mode...");
		}
		ret = lvgl_ui_main(ui_config, config_result.path,
				   effective_runtime.touch_click_marker,
				   effective_runtime.wifi_keyboard);
	}
#else
	ret = lvgl_ui_main(ui_config, config_result.path);
#endif

	spdlog::info("[main] LVGL UI exited with code: {}", ret);
	return ret;
}
