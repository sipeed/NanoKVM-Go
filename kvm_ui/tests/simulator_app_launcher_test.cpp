#include <string>
#include <string_view>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/simulator_app_launcher.hpp"

TEST_CASE("simulator Apps launcher records a safe mock launch",
	  "[simulator][apps]")
{
	lvgl_main::SimulatorAppLauncher launcher;
	const std::string app_path = "/safe/project/apps/example/main.py";

	const auto result = launcher.launch(app_path);
	CHECK(result.status == lvgl_main::SimulatorAppLaunchStatus::Completed);
	CHECK(result.ok());
	CHECK(result.detail == "launch simulated; no child process started");
	CHECK(launcher.last_launched_path() == app_path);
}

TEST_CASE("simulator Apps launcher rejects an empty request without mutation",
	  "[simulator][apps]")
{
	lvgl_main::SimulatorAppLauncher launcher;
	REQUIRE(launcher.launch("/safe/project/apps/example/main.py").ok());

	const auto result = launcher.launch(std::string_view {});
	CHECK(result.status ==
	      lvgl_main::SimulatorAppLaunchStatus::InvalidRequest);
	CHECK_FALSE(result.ok());
	CHECK(result.detail == "app path is empty");
	CHECK(launcher.last_launched_path() ==
	      "/safe/project/apps/example/main.py");
}
