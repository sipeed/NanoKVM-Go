#include "lvgl_main/lvgl_main.hpp"

lvgl_main::AppConfig
lvgl_ui_effective_config(const lvgl_main::AppConfig &config,
			 bool wifi_keyboard_test_mode)
{
	(void)wifi_keyboard_test_mode;
	auto ui_config = config;
	if (ui_config.runtime.touch_calibrate ||
	    ui_config.runtime.touch_paint_test) {
		ui_config.touch.disabled = false;
	}
	return ui_config;
}
