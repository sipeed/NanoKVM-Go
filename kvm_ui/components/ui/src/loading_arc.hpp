#pragma once

#include <cstdint>

#include <lvgl.h>

namespace ui
{

/**
 * @brief Page-level indeterminate loading spinner.
 *
 * Mirrors the WifiConnectResultPage spinner visual (60x60 arc, 6px rounded
 * indicator, #2B7FFF, 270-degree sweep, +18 degrees per 40ms tick) as a
 * reusable component. Purely presentational: no timeout and no content
 * ownership - the host page alone decides when the spinner runs and what
 * happens to sibling content.
 */
class LoadingArc {
    public:
	static constexpr uint16_t SIZE = 60;
	static constexpr uint16_t ARC_WIDTH = 6;
	static constexpr uint32_t COLOR = 0x2B7FFF;
	static constexpr uint32_t PERIOD_MS = 40;
	static constexpr uint16_t ROTATION_STEP_DEG = 18;
	static constexpr uint16_t SWEEP_DEG = 270;

	explicit LoadingArc(lv_obj_t *parent);
	~LoadingArc();

	LoadingArc(const LoadingArc &) = delete;
	LoadingArc &operator=(const LoadingArc &) = delete;

	void create();
	void set_running(bool running);
	bool is_running() const
	{
		return _running;
	}

	lv_obj_t *obj() const
	{
		return _arc;
	}

	void destroy();

    private:
	lv_obj_t *_parent = nullptr;
	lv_obj_t *_arc = nullptr;
	lv_timer_t *_spin_timer = nullptr;
	uint16_t _rotation = 0;
	bool _running = false;

	void start_timer();
	void stop_timer();

	static void spin_timer_cb(lv_timer_t *timer);
};

} // namespace ui
