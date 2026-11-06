/**
 * @file lvgl_input_reader.cpp
 * LVGL touch input reader, wake filtering, and touch diagnostics marker.
 */

#include <cerrno>
#include <cstdint>
#include <ctime>

#include <linux/input.h>
#include <lvgl.h>
#include <spdlog/spdlog.h>
#include <src/display/lv_display_private.h>
#include <sys/ioctl.h>
#include <unistd.h>

#include "lvgl_ui_internal.hpp"
#include "lvgl_main/touch_wake_filter.hpp"

namespace
{

static int indev_fd = -1;
static int indev_last_x = 0;
static int indev_last_y = 0;
static int indev_is_pressed = 0;
static uint64_t indev_activity_seq = 0;
static std::atomic<bool> indev_screen_on { true };
static lvgl_main::TouchWakeFilter indev_touch_wake_filter;

int64_t event_timestamp_us(const input_event &ev)
{
	return static_cast<int64_t>(ev.time.tv_sec) * 1000000LL +
	       static_cast<int64_t>(ev.time.tv_usec);
}

bool update_touch_input_state(const input_event &ev)
{
	if (ev.type == EV_ABS) {
		if (ev.code == ABS_MT_POSITION_X || ev.code == ABS_X) {
			indev_last_x = ev.value;
		} else if (ev.code == ABS_MT_POSITION_Y ||
			   ev.code == ABS_Y) {
			indev_last_y = ev.value;
		}
		return true;
	}
	if (ev.type == EV_KEY && ev.code == BTN_TOUCH) {
		indev_is_pressed = ev.value;
		return true;
	}
	return false;
}

} // namespace

TouchClickMarkerState touch_click_marker_state;

void reset_touch_wake_filter()
{
	indev_screen_on.store(true, std::memory_order_release);
	indev_touch_wake_filter.reset();
}

void reset_touch_input_state()
{
	indev_fd = -1;
	indev_last_x = 0;
	indev_last_y = 0;
	indev_is_pressed = 0;
	lvgl_main::wifi_latency_probe::set_event_clock_monotonic(false);
	reset_touch_wake_filter();
}

void set_touch_input_fd(int fd)
{
	indev_fd = fd;
	if (!lvgl_main::wifi_latency_probe::enabled()) {
		return;
	}

#if defined(EVIOCSCLOCKID)
	clockid_t clock_id = CLOCK_MONOTONIC;
	if (::ioctl(fd, EVIOCSCLOCKID, &clock_id) == 0) {
		lvgl_main::wifi_latency_probe::set_event_clock_monotonic(true);
		spdlog::warn("[wifi-latency] evdev_clock=monotonic");
		return;
	}
	spdlog::warn(
		"[wifi-latency] evdev_clock=monotonic_failed errno={} fallback=realtime",
		errno);
#else
	spdlog::warn(
		"[wifi-latency] evdev_clock=monotonic_unsupported fallback=realtime");
#endif
	lvgl_main::wifi_latency_probe::set_event_clock_monotonic(false);
}

void drain_touch_input_events(int fd)
{
	struct input_event ev;
	while (::read(fd, &ev, sizeof(ev)) ==
	       static_cast<ssize_t>(sizeof(ev))) {
		(void)update_touch_input_state(ev);
	}
}

int touch_input_last_x()
{
	return indev_last_x;
}

int touch_input_last_y()
{
	return indev_last_y;
}

bool touch_input_is_pressed()
{
	return indev_is_pressed != 0;
}

uint64_t touch_input_activity_sequence()
{
	return indev_activity_seq;
}

void record_touch_input_activity()
{
	++indev_activity_seq;
}

void set_touch_input_screen_on(bool screen_on)
{
	indev_screen_on.store(screen_on, std::memory_order_release);
}

void read_tp(lv_indev_data_t *data, bool suppress_wake_touch)
{
	if (indev_fd < 0) {
		data->point.x = 0;
		data->point.y = 0;
		data->state = LV_INDEV_STATE_RELEASED;
		return;
	}

	struct input_event ev;
	bool input_activity_seen = false;
	bool touch_press_seen = false;

	while (true) {
		ssize_t n = read(indev_fd, &ev, sizeof(ev));
		if (n < 0) {
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				break;
			}
			break;
		} else if (n == 0) {
			break;
		}
		if (lvgl_main::wifi_latency_probe::enabled() &&
		    ev.type == EV_KEY && ev.code == BTN_TOUCH && ev.value != 0) {
			const int64_t arrival_timestamp_us =
				lvgl_main::wifi_latency_probe::timestamp_now_us();
			int64_t kernel_timestamp_us = event_timestamp_us(ev);
			if (kernel_timestamp_us <= 0) {
				kernel_timestamp_us = arrival_timestamp_us;
			}
			lvgl_main::wifi_latency_probe::record_touch_press_event(
				kernel_timestamp_us, arrival_timestamp_us);
			touch_press_seen = true;
		}

		if (update_touch_input_state(ev)) {
			input_activity_seen = true;
			record_touch_input_activity();
		}
	}
	if (touch_press_seen) {
		lvgl_main::wifi_latency_probe::record_input_read(
			lvgl_main::wifi_latency_probe::timestamp_now_us());
	}

	data->point.x = indev_last_x;
	data->point.y = indev_last_y;
	const bool raw_pressed = indev_is_pressed != 0;
	const bool suppress_touch =
		suppress_wake_touch &&
		indev_touch_wake_filter.should_suppress(
			indev_screen_on.load(std::memory_order_acquire),
			input_activity_seen, raw_pressed);
	if (raw_pressed && !suppress_touch) {
		data->state = LV_INDEV_STATE_PRESSED;
	} else {
		data->state = LV_INDEV_STATE_RELEASED;
	}
}

void read_tp_disabled(lv_indev_data_t *data)
{
	data->point.x = 0;
	data->point.y = 0;
	data->state = LV_INDEV_STATE_RELEASED;
}

void touch_click_marker_event_cb(lv_event_t *event)
{
	if (!touch_click_marker_state.enabled.load(std::memory_order_acquire)) {
		return;
	}

	auto *indev =
		static_cast<lv_indev_t *>(lv_event_get_current_target(event));
	if (indev == nullptr) {
		return;
	}

	lv_point_t point {};
	lv_indev_get_point(indev, &point);
	touch_click_marker_state.x.store(point.x, std::memory_order_release);
	touch_click_marker_state.y.store(point.y, std::memory_order_release);
	touch_click_marker_state.visible.store(true, std::memory_order_release);
	touch_click_marker_state.revision.fetch_add(1U, std::memory_order_release);
}

void touch_latency_probe_event_cb(lv_event_t *event)
{
	if (!lvgl_main::wifi_latency_probe::enabled()) {
		return;
	}

	auto *indev =
		static_cast<lv_indev_t *>(lv_event_get_current_target(event));
	if (indev == nullptr) {
		return;
	}

	if (lvgl_main::wifi_latency_probe::visible_marker_enabled()) {
		lv_point_t point {};
		lv_indev_get_point(indev, &point);
		touch_click_marker_state.x.store(point.x, std::memory_order_release);
		touch_click_marker_state.y.store(point.y, std::memory_order_release);
		touch_click_marker_state.visible.store(true, std::memory_order_release);
		touch_click_marker_state.revision.fetch_add(1U,
						      std::memory_order_release);
	}
	lvgl_main::wifi_latency_probe::record_lvgl_pressed(
		lvgl_main::wifi_latency_probe::timestamp_now_us());
}

lv_area_t touch_click_marker_area(int32_t x, int32_t y)
{
	lv_area_t area;
	lv_area_set(&area, x - kTouchClickMarkerRadius,
		    y - kTouchClickMarkerRadius, x + kTouchClickMarkerRadius,
		    y + kTouchClickMarkerRadius);
	return area;
}

// Map raw evdev coordinates to LVGL logical coordinates using the same
// transform LVGL applies in indev_pointer_proc. The supervisor reads raw
// touch values directly (LVGL is paused), so without this the exit zone would
// sit on a rotated edge and move when 180 rotation is toggled.
void map_raw_to_logical(const lv_display_t *disp, int raw_x, int raw_y,
			       int &out_x, int &out_y)
{
	int32_t x = raw_x;
	int32_t y = raw_y;
	const int32_t hor = disp->hor_res;
	const int32_t ver = disp->ver_res;
	const uint32_t rot = disp->rotation;
	if (rot == LV_DISPLAY_ROTATION_180 || rot == LV_DISPLAY_ROTATION_270) {
		x = hor - x - 1;
		y = ver - y - 1;
	}
	if (rot == LV_DISPLAY_ROTATION_90 || rot == LV_DISPLAY_ROTATION_270) {
		const int32_t tmp = y;
		y = x;
		x = ver - tmp - 1;
	}
	out_x = static_cast<int>(x);
	out_y = static_cast<int>(y);
}
