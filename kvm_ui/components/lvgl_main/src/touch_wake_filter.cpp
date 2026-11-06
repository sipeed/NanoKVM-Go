#include "lvgl_main/touch_wake_filter.hpp"

namespace lvgl_main
{

void TouchWakeFilter::reset()
{
	_last_screen_on = true;
	_screen_off_touch_consumed = false;
	_suppress_until_release = false;
}

bool TouchWakeFilter::should_suppress(bool screen_on, bool input_activity_seen,
				      bool raw_pressed)
{
	if (screen_on || (_last_screen_on && !screen_on)) {
		_screen_off_touch_consumed = false;
	}
	_last_screen_on = screen_on;

	if (!screen_on && input_activity_seen && !_screen_off_touch_consumed) {
		_screen_off_touch_consumed = true;
		_suppress_until_release = raw_pressed;
	}

	if (!raw_pressed) {
		_suppress_until_release = false;
		return false;
	}

	return _suppress_until_release;
}

} // namespace lvgl_main
