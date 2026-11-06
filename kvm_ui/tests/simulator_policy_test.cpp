#include <filesystem>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/simulator_policy.hpp"

#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_build_config.hpp"
#endif

using namespace lvgl_main;

TEST_CASE("simulator policy rejects only protected target paths",
	  "[simulator][policy]")
{
	CHECK(simulator_policy::is_protected_target_path("/boot/config.txt"));
	CHECK(simulator_policy::is_protected_target_path("/dev/fb0"));
	CHECK(simulator_policy::is_protected_target_path("/sys/class/pwm"));
	CHECK(simulator_policy::is_protected_target_path("/run/kvm/state"));
	CHECK(simulator_policy::is_protected_target_path(
		"/etc/kvm/kvm_ui.toml"));
	CHECK(simulator_policy::is_protected_target_path(
		"/opt/scripts/wifi.sh"));
	CHECK(simulator_policy::is_protected_target_path(
		"/kvmcomm/background"));
	CHECK(simulator_policy::is_protected_target_path("/tmp/kvm/state"));
	CHECK(simulator_policy::is_protected_target_path(
		"/etc/kvm/../kvm/file"));
	CHECK(simulator_policy::is_protected_target_path("/"));

	CHECK_FALSE(simulator_policy::is_protected_target_path(
		"/etc/kvm2/ui.toml"));
	CHECK_FALSE(
		simulator_policy::is_protected_target_path("/tmp/kvm-ui.toml"));
}

TEST_CASE("simulator policy resolves a safe explicit config path")
{
	std::filesystem::path resolved;
	std::string error;
	REQUIRE(simulator_policy::resolve_config_path(
		"/tmp/kvm-ui-simulator-safe.toml", true, resolved, error));
	CHECK(resolved == "/tmp/kvm-ui-simulator-safe.toml");
	CHECK(error.empty());

	CHECK_FALSE(simulator_policy::resolve_config_path(
		"/etc/kvm/kvm_ui.toml", true, resolved, error));
	CHECK(error.find("protected") != std::string::npos);
}

TEST_CASE("simulator effective config is safe and idempotent")
{
	AppConfig config;
	config.http_api.enabled = true;
	config.http_api.port = 65501;
	config.screenshot.enabled = true;
	config.runtime.touch_calibrate = true;
	config.runtime.touch_paint_test = true;
	config.ui.enable_auto_rotation = true;
	config.ui.manual_display_rotation = true;
	config.nanokvm.enable_runtime_backend = false;
	config.nanokvm.enable_high_risk_actions = true;
	config.launcher.enabled = false;

	const auto once = simulator_policy::make_effective_config(config);
	const auto twice = simulator_policy::make_effective_config(once);

#if NANOKVM_GO_SIMULATOR_BUILD
	CHECK_FALSE(once.http_api.enabled);
	CHECK(once.http_api.port == 0);
	CHECK_FALSE(once.screenshot.enabled);
	CHECK_FALSE(once.runtime.touch_calibrate);
	CHECK_FALSE(once.runtime.touch_paint_test);
	CHECK_FALSE(once.ui.enable_auto_rotation);
	CHECK(once.ui.manual_display_rotation);
	CHECK(once.nanokvm.enable_runtime_backend);
	CHECK_FALSE(once.nanokvm.enable_high_risk_actions);
	CHECK(once.launcher.enabled);
	CHECK(twice.http_api.port == once.http_api.port);
#else
	CHECK(once.http_api.enabled == config.http_api.enabled);
	CHECK(once.http_api.port == config.http_api.port);
	CHECK(once.screenshot.enabled == config.screenshot.enabled);
	CHECK(once.runtime.touch_calibrate == config.runtime.touch_calibrate);
#endif
}

TEST_CASE("launcher Apps path follows the build mode")
{
	const auto path = simulator_policy::launcher_apps_path();
#if NANOKVM_GO_SIMULATOR_BUILD
	CHECK(path == std::filesystem::path(
			      std::string(simulator_build_config::kAppsPath)));
	CHECK(path != std::filesystem::path("/kvmcomm/apps"));
#else
	CHECK(path == std::filesystem::path("/kvmcomm/apps"));
#endif
}

TEST_CASE("simulator policy uses the generated default only in simulator builds")
{
	std::filesystem::path resolved;
	std::string error;
#if NANOKVM_GO_SIMULATOR_BUILD
	REQUIRE(simulator_policy::resolve_config_path({}, false, resolved,
						      error));
	CHECK(resolved == simulator_build_config::kDefaultConfigPath);
#else
	CHECK_FALSE(simulator_policy::resolve_config_path({}, false, resolved,
							  error));
	CHECK(error.find("empty") != std::string::npos);
#endif
}
