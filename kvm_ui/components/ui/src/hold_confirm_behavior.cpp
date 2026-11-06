#include "ui/hold_confirm_behavior.hpp"

#include <algorithm>
#include <utility>

namespace ui
{

HoldConfirmBehavior::HoldConfirmBehavior(BConA &button,
					 HoldConfirmConfig config)
	: _button(button)
	, _config(std::move(config))
{
	if (_config.duration_ms == 0) {
		_config.duration_ms = DEFAULT_DURATION_MS;
	} else {
		_config.duration_ms =
			std::max(_config.duration_ms, MIN_DURATION_MS);
	}

	_button.on_click({});
	_button.set_text(_config.idle_text.c_str());
	if (!attach()) {
		fail_closed();
	}
}

HoldConfirmBehavior::~HoldConfirmBehavior()
{
	detach();
}

void HoldConfirmBehavior::on_confirm(ConfirmCallback callback)
{
	_confirm_callback = std::move(callback);
}

void HoldConfirmBehavior::cancel()
{
	clear_indev_tracking();
	if (_clip_obj != nullptr) {
		lv_anim_delete(_clip_obj, progress_exec_cb);
	}
	delete_confirmation_timer();
	_state = State::IDLE;
	restore_idle_visuals();
}

bool HoldConfirmBehavior::is_ready() const
{
	return _ready;
}

bool HoldConfirmBehavior::is_holding() const
{
	return _state == State::HOLDING || _state == State::FULL;
}

int32_t HoldConfirmBehavior::progress_width() const
{
	if (_clip_obj == nullptr || !lv_obj_is_valid(_clip_obj)) {
		return 0;
	}
	return std::clamp(lv_obj_get_style_width(_clip_obj, LV_PART_MAIN), 0,
			  static_cast<int32_t>(BConA::WIDTH));
}

bool HoldConfirmBehavior::attach()
{
	_button_obj = _button.get_obj();
	if (_button_obj == nullptr || !lv_obj_is_valid(_button_obj)) {
		return false;
	}

	if (lv_obj_add_event_cb(
		    _button_obj, click_gate_cb,
		    static_cast<lv_event_code_t>(LV_EVENT_SHORT_CLICKED |
						 LV_EVENT_PREPROCESS),
		    this) == nullptr) {
		return false;
	}
	if (lv_obj_add_event_cb(_button_obj, click_gate_cb,
				static_cast<lv_event_code_t>(
					LV_EVENT_CLICKED | LV_EVENT_PREPROCESS),
				this) == nullptr) {
		return false;
	}

	_scroll_obj = find_scrollable_ancestor(_button_obj);

	_clip_obj = lv_obj_create(_button_obj);
	if (_clip_obj == nullptr) {
		return false;
	}
	lv_obj_set_size(_clip_obj, 0, BConA::HEIGHT);
	lv_obj_set_pos(_clip_obj, -static_cast<int32_t>(BConA::PADDING_LEFT),
		       -lv_obj_get_style_pad_top(_button_obj, LV_PART_MAIN));
	lv_obj_set_style_bg_opa(_clip_obj, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_clip_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_clip_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_radius(_clip_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_clip_corner(_clip_obj, false, LV_PART_MAIN);
	lv_obj_clear_flag(_clip_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_clip_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_clip_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	lv_obj_add_flag(_clip_obj, LV_OBJ_FLAG_IGNORE_LAYOUT);
	lv_obj_add_flag(_clip_obj, LV_OBJ_FLAG_HIDDEN);
	if (lv_obj_add_event_cb(_clip_obj, overlay_delete_event_cb,
				LV_EVENT_DELETE, this) == nullptr) {
		return false;
	}

	_fill_obj = lv_obj_create(_clip_obj);
	if (_fill_obj == nullptr) {
		return false;
	}
	lv_obj_set_size(_fill_obj, BConA::WIDTH, BConA::HEIGHT);
	lv_obj_set_pos(_fill_obj, 0, 0);
	lv_obj_set_style_bg_color(_fill_obj, lv_color_hex(_config.fill_color),
				  LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_fill_obj, LV_OPA_COVER, LV_PART_MAIN);
	lv_obj_set_style_border_width(_fill_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_fill_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_radius(_fill_obj, BConA::RADIUS, LV_PART_MAIN);
	lv_obj_clear_flag(_fill_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_fill_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_fill_obj, LV_OBJ_FLAG_IGNORE_LAYOUT);
	if (lv_obj_add_event_cb(_fill_obj, overlay_delete_event_cb,
				LV_EVENT_DELETE, this) == nullptr) {
		return false;
	}

	lv_obj_move_to_index(_clip_obj, 0);

	constexpr lv_event_code_t button_events[] = {
		LV_EVENT_PRESSED,    LV_EVENT_PRESSING, LV_EVENT_RELEASED,
		LV_EVENT_PRESS_LOST, LV_EVENT_CANCEL,	LV_EVENT_INDEV_RESET,
		LV_EVENT_DELETE,
	};
	for (const lv_event_code_t event_code : button_events) {
		if (lv_obj_add_event_cb(_button_obj, button_event_cb,
					event_code, this) == nullptr) {
			return false;
		}
	}

	if (_scroll_obj != nullptr) {
		if (lv_obj_add_event_cb(_scroll_obj, scroll_event_cb,
					LV_EVENT_SCROLL_BEGIN,
					this) == nullptr) {
			return false;
		}
		if (lv_obj_add_event_cb(_scroll_obj, scroll_event_cb,
					LV_EVENT_DELETE, this) == nullptr) {
			return false;
		}
	}

	_ready = true;
	return true;
}

void HoldConfirmBehavior::fail_closed()
{
	_ready = false;
	stop_timing();
	clear_indev_tracking();
	_state = State::IDLE;
	restore_idle_visuals();
	if (_button_obj != nullptr && lv_obj_is_valid(_button_obj)) {
		lv_obj_clear_flag(_button_obj, LV_OBJ_FLAG_CLICKABLE);
	}
	delete_overlay();
}

void HoldConfirmBehavior::detach()
{
	_ready = false;
	stop_timing();
	clear_indev_tracking();
	_state = State::IDLE;
	remove_callbacks();
	restore_idle_visuals();
	delete_overlay();
	_button_obj = nullptr;
	_scroll_obj = nullptr;
}

void HoldConfirmBehavior::remove_callbacks()
{
	if (_button_obj != nullptr && lv_obj_is_valid(_button_obj)) {
		lv_obj_remove_event_cb_with_user_data(_button_obj,
						      click_gate_cb, this);
		lv_obj_remove_event_cb_with_user_data(_button_obj,
						      button_event_cb, this);
	}
	if (_clip_obj != nullptr && lv_obj_is_valid(_clip_obj)) {
		lv_obj_remove_event_cb_with_user_data(
			_clip_obj, overlay_delete_event_cb, this);
	}
	if (_fill_obj != nullptr && lv_obj_is_valid(_fill_obj)) {
		lv_obj_remove_event_cb_with_user_data(
			_fill_obj, overlay_delete_event_cb, this);
	}
	if (_scroll_obj != nullptr && lv_obj_is_valid(_scroll_obj)) {
		lv_obj_remove_event_cb_with_user_data(_scroll_obj,
						      scroll_event_cb, this);
	}
}

void HoldConfirmBehavior::stop_timing()
{
	if (_clip_obj != nullptr) {
		lv_anim_delete(_clip_obj, progress_exec_cb);
	}
	delete_confirmation_timer();
}

void HoldConfirmBehavior::delete_confirmation_timer()
{
	if (_confirm_timer == nullptr) {
		return;
	}
	lv_timer_delete(_confirm_timer);
	_confirm_timer = nullptr;
}

void HoldConfirmBehavior::delete_overlay()
{
	if (_clip_obj != nullptr && lv_obj_is_valid(_clip_obj)) {
		lv_obj_delete(_clip_obj);
	} else if (_fill_obj != nullptr && lv_obj_is_valid(_fill_obj)) {
		lv_obj_delete(_fill_obj);
	}
	_clip_obj = nullptr;
	_fill_obj = nullptr;
}

void HoldConfirmBehavior::restore_idle_visuals()
{
	if (_clip_obj != nullptr && lv_obj_is_valid(_clip_obj)) {
		lv_obj_set_width(_clip_obj, 0);
		lv_obj_add_flag(_clip_obj, LV_OBJ_FLAG_HIDDEN);
	}
	restore_idle_text();
}

void HoldConfirmBehavior::restore_idle_text()
{
	if (_button_obj == nullptr || !lv_obj_is_valid(_button_obj) ||
	    _button.get_obj() != _button_obj) {
		return;
	}
	_button.set_text(_config.idle_text.c_str());
}

bool HoldConfirmBehavior::track_indev(lv_indev_t *indev)
{
	if (indev == nullptr) {
		return false;
	}

	clear_indev_tracking();
	const uint32_t event_count = lv_indev_get_event_count(indev);
	// Run before ordinary DELETE/ALL handlers which may destroy this owner.
	// Earlier third-party preprocess handlers must not destroy it first because
	// LVGL cannot safely remove a later descriptor during the same dispatch.
	constexpr lv_event_code_t delete_filter = static_cast<lv_event_code_t>(
		LV_EVENT_DELETE | LV_EVENT_PREPROCESS);
	lv_indev_add_event_cb(indev, indev_delete_event_cb, delete_filter,
			      this);
	if (lv_indev_get_event_count(indev) != event_count + 1U) {
		lv_indev_remove_event_cb_with_user_data(
			indev, indev_delete_event_cb, this);
		return false;
	}

	lv_event_dsc_t *event = lv_indev_get_event_dsc(indev, event_count);
	if (event == nullptr ||
	    lv_event_dsc_get_cb(event) != indev_delete_event_cb ||
	    lv_event_dsc_get_user_data(event) != this) {
		lv_indev_remove_event_cb_with_user_data(
			indev, indev_delete_event_cb, this);
		return false;
	}

	_indev = indev;
	_indev_delete_event = event;
	return true;
}

void HoldConfirmBehavior::clear_indev_tracking()
{
	lv_indev_t *indev = _indev;
	_indev = nullptr;
	_indev_delete_event = nullptr;
	if (indev != nullptr) {
		lv_indev_remove_event_cb_with_user_data(
			indev, indev_delete_event_cb, this);
	}
}

bool HoldConfirmBehavior::objects_valid() const
{
	return _button_obj != nullptr && _clip_obj != nullptr &&
	       _fill_obj != nullptr && lv_obj_is_valid(_button_obj) &&
	       lv_obj_is_valid(_clip_obj) && lv_obj_is_valid(_fill_obj) &&
	       _button.get_obj() == _button_obj &&
	       lv_obj_get_parent(_clip_obj) == _button_obj &&
	       lv_obj_get_parent(_fill_obj) == _clip_obj;
}

bool HoldConfirmBehavior::button_available() const
{
	return _button_obj != nullptr && lv_obj_is_valid(_button_obj) &&
	       lv_obj_has_flag(_button_obj, LV_OBJ_FLAG_CLICKABLE) &&
	       !lv_obj_has_state(_button_obj, LV_STATE_DISABLED) &&
	       !_button.is_pending();
}

bool HoldConfirmBehavior::point_inside_button(lv_indev_t *indev) const
{
	if (indev == nullptr || _button_obj == nullptr ||
	    !lv_obj_is_valid(_button_obj)) {
		return false;
	}

	lv_point_t point {};
	lv_area_t coordinates {};
	lv_indev_get_point(indev, &point);
	lv_obj_get_coords(_button_obj, &coordinates);
	return point.x >= coordinates.x1 && point.x <= coordinates.x2 &&
	       point.y >= coordinates.y1 && point.y <= coordinates.y2;
}

bool HoldConfirmBehavior::can_begin_hold(lv_event_t *event,
					 lv_indev_t *indev) const
{
	return event != nullptr && _ready && _state == State::IDLE &&
	       objects_valid() &&
	       lv_event_get_current_target_obj(event) == _button_obj &&
	       lv_event_get_target_obj(event) == _button_obj &&
	       button_available() && indev != nullptr &&
	       lv_indev_get_state(indev) == LV_INDEV_STATE_PRESSED &&
	       lv_indev_get_scroll_obj(indev) == nullptr &&
	       point_inside_button(indev);
}

bool HoldConfirmBehavior::can_confirm() const
{
	return _ready && _state == State::FULL &&
	       static_cast<bool>(_confirm_callback) && objects_valid() &&
	       progress_width() == BConA::WIDTH && _indev != nullptr &&
	       lv_indev_get_state(_indev) == LV_INDEV_STATE_PRESSED &&
	       lv_indev_get_scroll_obj(_indev) == nullptr &&
	       point_inside_button(_indev) && button_available();
}

void HoldConfirmBehavior::begin_hold(lv_indev_t *indev)
{
	if (!track_indev(indev)) {
		cancel();
		return;
	}
	_state = State::HOLDING;
	_button.set_text(_config.holding_text.c_str());
	lv_obj_set_width(_clip_obj, DISCOVERY_ANCHOR_WIDTH);
	lv_obj_clear_flag(_clip_obj, LV_OBJ_FLAG_HIDDEN);

	lv_anim_t animation {};
	lv_anim_init(&animation);
	lv_anim_set_var(&animation, _clip_obj);
	lv_anim_set_exec_cb(&animation, progress_exec_cb);
	lv_anim_set_values(&animation, DISCOVERY_ANCHOR_WIDTH, BConA::WIDTH);
	lv_anim_set_duration(&animation, _config.duration_ms - FULL_DWELL_MS);
	lv_anim_set_path_cb(&animation, lv_anim_path_ease_out);
	lv_anim_set_completed_cb(&animation, progress_completed_cb);
	lv_anim_set_user_data(&animation, this);
	if (lv_anim_start(&animation) == nullptr) {
		cancel();
	}
}

void HoldConfirmBehavior::handle_pressed(lv_event_t *event)
{
	lv_indev_t *indev = lv_event_get_indev(event);
	if (!can_begin_hold(event, indev)) {
		return;
	}
	begin_hold(indev);
}

void HoldConfirmBehavior::handle_pressing(lv_event_t *event)
{
	if (!is_holding()) {
		return;
	}
	lv_indev_t *indev = lv_event_get_indev(event);
	if (indev != _indev || !point_inside_button(indev)) {
		cancel();
	}
}

void HoldConfirmBehavior::handle_delete(lv_obj_t *target)
{
	if (target == nullptr) {
		return;
	}

	_ready = false;
	delete_confirmation_timer();
	clear_indev_tracking();
	_state = State::IDLE;

	if (target == _button_obj) {
		_button_obj = nullptr;
		return;
	}
	if (target == _clip_obj) {
		_clip_obj = nullptr;
		restore_idle_text();
		return;
	}
	if (target == _fill_obj) {
		_fill_obj = nullptr;
		if (_clip_obj != nullptr && lv_obj_is_valid(_clip_obj)) {
			lv_anim_delete(_clip_obj, progress_exec_cb);
			lv_obj_set_width(_clip_obj, 0);
			lv_obj_add_flag(_clip_obj, LV_OBJ_FLAG_HIDDEN);
		}
		restore_idle_text();
	}
}

void HoldConfirmBehavior::handle_scroll_delete(lv_obj_t *target)
{
	if (target != _scroll_obj) {
		return;
	}
	_ready = false;
	delete_confirmation_timer();
	clear_indev_tracking();
	_state = State::IDLE;
	_scroll_obj = nullptr;
}

void HoldConfirmBehavior::handle_fill_complete()
{
	if (!_ready || _state != State::HOLDING || !objects_valid()) {
		cancel();
		return;
	}

	lv_obj_set_width(_clip_obj, BConA::WIDTH);
	_state = State::FULL;
	lv_timer_t *timer =
		lv_timer_create(confirmation_timer_cb, FULL_DWELL_MS, this);
	if (timer == nullptr) {
		cancel();
		return;
	}
	_confirm_timer = timer;
	lv_timer_set_repeat_count(timer, 1);
	lv_timer_set_auto_delete(timer, true);
}

void HoldConfirmBehavior::trigger_confirmation()
{
	if (!can_confirm()) {
		cancel();
		return;
	}

	_state = State::TRIGGERING;
	clear_indev_tracking();
	restore_idle_visuals();
	ConfirmCallback callback = _confirm_callback;
	_state = State::IDLE;
	callback();
}

lv_obj_t *HoldConfirmBehavior::find_scrollable_ancestor(lv_obj_t *object)
{
	for (lv_obj_t *ancestor = lv_obj_get_parent(object);
	     ancestor != nullptr; ancestor = lv_obj_get_parent(ancestor)) {
		if (lv_obj_has_flag(ancestor, LV_OBJ_FLAG_SCROLLABLE)) {
			return ancestor;
		}
	}
	return nullptr;
}

void HoldConfirmBehavior::click_gate_cb(lv_event_t *event)
{
	if (event != nullptr) {
		lv_event_stop_processing(event);
	}
}

void HoldConfirmBehavior::button_event_cb(lv_event_t *event)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	switch (lv_event_get_code(event)) {
	case LV_EVENT_PRESSED:
		self->handle_pressed(event);
		break;
	case LV_EVENT_PRESSING:
		self->handle_pressing(event);
		break;
	case LV_EVENT_RELEASED:
	case LV_EVENT_PRESS_LOST:
	case LV_EVENT_CANCEL:
	case LV_EVENT_INDEV_RESET:
		if (self->is_holding()) {
			self->cancel();
		}
		break;
	case LV_EVENT_DELETE:
		self->handle_delete(lv_event_get_current_target_obj(event));
		break;
	default:
		break;
	}
}

void HoldConfirmBehavior::scroll_event_cb(lv_event_t *event)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	if (lv_event_get_code(event) == LV_EVENT_SCROLL_BEGIN) {
		if (self->is_holding()) {
			self->cancel();
		}
	} else if (lv_event_get_code(event) == LV_EVENT_DELETE) {
		self->handle_scroll_delete(
			lv_event_get_current_target_obj(event));
	}
}

void HoldConfirmBehavior::overlay_delete_event_cb(lv_event_t *event)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}
	self->handle_delete(lv_event_get_current_target_obj(event));
}

void HoldConfirmBehavior::indev_delete_event_cb(lv_event_t *event)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_event_get_user_data(event));
	if (self == nullptr || lv_event_get_code(event) != LV_EVENT_DELETE) {
		return;
	}

	auto *indev =
		static_cast<lv_indev_t *>(lv_event_get_current_target(event));
	if (indev != self->_indev || self->_indev_delete_event == nullptr) {
		return;
	}

	self->_indev = nullptr;
	self->_indev_delete_event = nullptr;
	self->cancel();
}

void HoldConfirmBehavior::progress_exec_cb(void *object, int32_t width)
{
	auto *clip = static_cast<lv_obj_t *>(object);
	if (clip == nullptr || !lv_obj_is_valid(clip)) {
		return;
	}
	lv_obj_set_width(clip, std::clamp(width, 0,
					  static_cast<int32_t>(BConA::WIDTH)));
}

void HoldConfirmBehavior::progress_completed_cb(lv_anim_t *animation)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_anim_get_user_data(animation));
	if (self != nullptr) {
		self->handle_fill_complete();
	}
}

void HoldConfirmBehavior::confirmation_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<HoldConfirmBehavior *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}
	self->_confirm_timer = nullptr;
	self->trigger_confirmation();
}

} // namespace ui
