#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/lvgl_main.hpp"

using namespace lvgl_main;

TEST_CASE("WiFi keyboard startup mode keeps runtime-backed services enabled")
{
	AppConfig config;
	config.nanokvm.enable_runtime_backend = true;
	config.http_api.enabled = true;
	config.http_api.port = 65501;
	config.ui.enable_auto_rotation = true;

	const auto effective = lvgl_ui_effective_config(config, true);

	CHECK(effective.nanokvm.enable_runtime_backend);
	CHECK(effective.http_api.enabled);
	CHECK(effective.http_api.port == 65501);
	CHECK(effective.ui.enable_auto_rotation);
}

TEST_CASE("normal UI startup keeps the provided config unchanged")
{
	AppConfig config;
	config.nanokvm.enable_runtime_backend = false;
	config.http_api.enabled = false;
	config.http_api.port = 0;
	config.ui.enable_auto_rotation = false;

	const auto effective = lvgl_ui_effective_config(config, false);

	CHECK_FALSE(effective.nanokvm.enable_runtime_backend);
	CHECK_FALSE(effective.http_api.enabled);
	CHECK(effective.http_api.port == 0);
	CHECK_FALSE(effective.ui.enable_auto_rotation);
}

TEST_CASE(
	"touch calibration startup enables touch without disabling runtime services")
{
	AppConfig config;
	config.touch.disabled = true;
	config.runtime.touch_calibrate = true;
	config.nanokvm.enable_runtime_backend = true;
	config.http_api.enabled = true;
	config.http_api.port = 65501;

	const auto effective = lvgl_ui_effective_config(config, false);

	CHECK_FALSE(effective.touch.disabled);
	CHECK(effective.runtime.touch_calibrate);
	CHECK_FALSE(effective.runtime.touch_paint_test);
	CHECK(effective.nanokvm.enable_runtime_backend);
	CHECK(effective.http_api.enabled);
	CHECK(effective.http_api.port == 65501);
}

TEST_CASE("touch paint startup enables touch without disabling runtime services")
{
	AppConfig config;
	config.touch.disabled = true;
	config.runtime.touch_paint_test = true;
	config.nanokvm.enable_runtime_backend = true;
	config.http_api.enabled = true;
	config.http_api.port = 65501;

	const auto effective = lvgl_ui_effective_config(config, false);

	CHECK_FALSE(effective.touch.disabled);
	CHECK_FALSE(effective.runtime.touch_calibrate);
	CHECK(effective.runtime.touch_paint_test);
	CHECK(effective.nanokvm.enable_runtime_backend);
	CHECK(effective.http_api.enabled);
	CHECK(effective.http_api.port == 65501);
}
