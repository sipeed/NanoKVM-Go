#pragma once

#include <cstdint>
#include <functional>
#include <string>

#include <lvgl.h>

#include "ui/bcona.hpp"

namespace ui
{

struct HoldConfirmConfig {
	uint32_t duration_ms = 2000;
	uint32_t fill_color = 0xFB2C36;
	std::string idle_text;
	std::string holding_text;
};

class HoldConfirmBehavior final {
    public:
	using ConfirmCallback = std::function<void()>;
	static constexpr uint32_t DEFAULT_DURATION_MS = 2000;
	static constexpr uint32_t MIN_DURATION_MS = 500;
	static constexpr uint32_t FULL_DWELL_MS =
		static_cast<uint32_t>(LV_DEF_REFR_PERIOD) * 2U;
	static constexpr int32_t DISCOVERY_ANCHOR_WIDTH =
		(static_cast<int32_t>(BConA::WIDTH) + 2) / 3;

	static_assert(MIN_DURATION_MS > FULL_DWELL_MS);
	static_assert(DISCOVERY_ANCHOR_WIDTH == 77);

	explicit HoldConfirmBehavior(BConA &button, HoldConfirmConfig config);
	~HoldConfirmBehavior();

	HoldConfirmBehavior(const HoldConfirmBehavior &) = delete;
	HoldConfirmBehavior &operator=(const HoldConfirmBehavior &) = delete;
	HoldConfirmBehavior(HoldConfirmBehavior &&) = delete;
	HoldConfirmBehavior &operator=(HoldConfirmBehavior &&) = delete;

	void on_confirm(ConfirmCallback callback);
	void cancel();
	bool is_ready() const;
	bool is_holding() const;
	int32_t progress_width() const;

    private:
	enum class State : uint8_t {
		IDLE,
		HOLDING,
		FULL,
		TRIGGERING,
	};

	BConA &_button;
	HoldConfirmConfig _config;
	ConfirmCallback _confirm_callback;
	State _state = State::IDLE;
	bool _ready = false;
	lv_obj_t *_button_obj = nullptr;
	lv_obj_t *_clip_obj = nullptr;
	lv_obj_t *_fill_obj = nullptr;
	lv_obj_t *_scroll_obj = nullptr;
	lv_indev_t *_indev = nullptr;
	lv_event_dsc_t *_indev_delete_event = nullptr;
	lv_timer_t *_confirm_timer = nullptr;

	bool attach();
	void fail_closed();
	void detach();
	void remove_callbacks();
	void stop_timing();
	void delete_confirmation_timer();
	void delete_overlay();
	void restore_idle_visuals();
	void restore_idle_text();
	bool track_indev(lv_indev_t *indev);
	void clear_indev_tracking();

	bool objects_valid() const;
	bool button_available() const;
	bool point_inside_button(lv_indev_t *indev) const;
	bool can_begin_hold(lv_event_t *event, lv_indev_t *indev) const;
	bool can_confirm() const;
	void begin_hold(lv_indev_t *indev);
	void handle_pressed(lv_event_t *event);
	void handle_pressing(lv_event_t *event);
	void handle_delete(lv_obj_t *target);
	void handle_scroll_delete(lv_obj_t *target);
	void handle_fill_complete();
	void trigger_confirmation();

	static lv_obj_t *find_scrollable_ancestor(lv_obj_t *object);
	static void click_gate_cb(lv_event_t *event);
	static void button_event_cb(lv_event_t *event);
	static void scroll_event_cb(lv_event_t *event);
	static void overlay_delete_event_cb(lv_event_t *event);
	static void indev_delete_event_cb(lv_event_t *event);
	static void progress_exec_cb(void *object, int32_t width);
	static void progress_completed_cb(lv_anim_t *animation);
	static void confirmation_timer_cb(lv_timer_t *timer);

	friend struct HoldConfirmBehaviorTestAccess;
};

} // namespace ui
