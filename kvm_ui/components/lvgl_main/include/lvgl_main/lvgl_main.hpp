#pragma once

#include "lvgl_main/app_config.hpp"

int lvgl_ui_main(
	const lvgl_main::AppConfig &config,
	const std::filesystem::path &config_path = std::filesystem::path(
		lvgl_main::app_config_defaults::kConfigPath.data()),
	bool touch_click_marker_enabled = false,
	bool wifi_keyboard_test_mode = false);

lvgl_main::AppConfig
lvgl_ui_effective_config(const lvgl_main::AppConfig &config,
			 bool wifi_keyboard_test_mode = false);

int lvgl_touch_calibration_main(
	const lvgl_main::AppConfig &config,
	const std::filesystem::path &config_path = std::filesystem::path(
		lvgl_main::app_config_defaults::kConfigPath.data()));

int lvgl_touch_paint_test_main(
	const lvgl_main::AppConfig &config,
	const std::filesystem::path &config_path = std::filesystem::path(
		lvgl_main::app_config_defaults::kConfigPath.data()));
