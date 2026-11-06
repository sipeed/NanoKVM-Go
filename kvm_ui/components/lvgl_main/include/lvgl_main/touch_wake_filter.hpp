#pragma once

namespace lvgl_main
{

class TouchWakeFilter final {
    public:
	void reset();
	bool should_suppress(bool screen_on, bool input_activity_seen,
			     bool raw_pressed);

    private:
	bool _last_screen_on { true };
	bool _screen_off_touch_consumed { false };
	bool _suppress_until_release { false };
};

} // namespace lvgl_main
