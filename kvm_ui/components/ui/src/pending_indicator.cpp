#include "pending_indicator.hpp"

#include <utility>

namespace ui
{

PendingIndicator::PendingIndicator(
	lv_obj_t *icon_parent, ContentVisibilityCallback set_content_visible)
	: _icon_parent(icon_parent)
	, _set_content_visible(std::move(set_content_visible))
{
}

PendingIndicator::~PendingIndicator()
{
	destroy();
}

void PendingIndicator::set_pending(bool pending)
{
	if (_icon_parent == nullptr) {
		_pending = false;
		stop_timers();
		return;
	}
	if (pending == _pending) {
		return;
	}

	if (pending) {
		ensure_arc();
		update_arc_color();
		_pending = true;
		_rotation = 0;
		if (_set_content_visible) {
			_set_content_visible(false);
		}
		if (_arc_obj != nullptr) {
			lv_arc_set_rotation(_arc_obj, _rotation);
			lv_obj_clear_flag(_arc_obj, LV_OBJ_FLAG_HIDDEN);
		}
		start_timers();
		return;
	}

	stop_timers();
	_pending = false;
	if (_arc_obj != nullptr) {
		lv_obj_add_flag(_arc_obj, LV_OBJ_FLAG_HIDDEN);
	}
	if (_set_content_visible) {
		_set_content_visible(true);
	}
}

bool PendingIndicator::is_pending() const
{
	return _pending;
}

void PendingIndicator::set_timeout(uint32_t timeout_ms)
{
	const uint32_t next_timeout =
		timeout_ms == 0 ? DEFAULT_TIMEOUT_MS : timeout_ms;
	if (_timeout_ms == next_timeout) {
		return;
	}

	_timeout_ms = next_timeout;
	if (_pending) {
		start_timers();
	}
}

void PendingIndicator::set_timeout_enabled(bool enabled)
{
	if (_timeout_enabled == enabled) {
		return;
	}

	_timeout_enabled = enabled;
	if (_pending) {
		start_timers();
	}
}

void PendingIndicator::set_arc_color(uint32_t color)
{
	_arc_color = color;
	update_arc_color();
}

void PendingIndicator::destroy()
{
	set_pending(false);
	_icon_parent = nullptr;
	_arc_obj = nullptr;
	_set_content_visible = nullptr;
}

void PendingIndicator::ensure_arc()
{
	if (_icon_parent == nullptr || _arc_obj != nullptr) {
		return;
	}

	_arc_obj = lv_arc_create(_icon_parent);
	lv_obj_set_size(_arc_obj, 36, 36);
	lv_obj_center(_arc_obj);
	lv_obj_clear_flag(_arc_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_arc_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_remove_style(_arc_obj, nullptr, LV_PART_KNOB);
	lv_obj_set_style_arc_opa(_arc_obj, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_arc_obj, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_arc_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_arc_obj, 0, LV_PART_MAIN);
	lv_obj_set_style_arc_width(_arc_obj, 4, LV_PART_INDICATOR);
	lv_obj_set_style_arc_rounded(_arc_obj, true, LV_PART_INDICATOR);
	update_arc_color();
	lv_arc_set_bg_angles(_arc_obj, 0, 360);
	lv_arc_set_angles(_arc_obj, 0, 270);
	lv_arc_set_rotation(_arc_obj, 0);
	lv_obj_add_flag(_arc_obj, LV_OBJ_FLAG_HIDDEN);
}

void PendingIndicator::update_arc_color()
{
	if (_arc_obj == nullptr) {
		return;
	}

	lv_obj_set_style_arc_color(_arc_obj, lv_color_hex(_arc_color),
					   LV_PART_INDICATOR);
}

void PendingIndicator::start_timers()
{
	stop_timers();
	_spin_timer = lv_timer_create(spin_timer_cb, 40, this);
	if (!_timeout_enabled) {
		return;
	}

	_timeout_timer = lv_timer_create(timeout_timer_cb, _timeout_ms, this);
	lv_timer_set_repeat_count(_timeout_timer, 1);
}

void PendingIndicator::stop_timers()
{
	if (_spin_timer != nullptr) {
		lv_timer_delete(_spin_timer);
		_spin_timer = nullptr;
	}
	if (_timeout_timer != nullptr) {
		lv_timer_delete(_timeout_timer);
		_timeout_timer = nullptr;
	}
}

void PendingIndicator::spin_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<PendingIndicator *>(lv_timer_get_user_data(timer));
	if (self == nullptr || !self->_pending || self->_arc_obj == nullptr) {
		return;
	}

	self->_rotation = static_cast<uint16_t>((self->_rotation + 18) % 360);
	lv_arc_set_rotation(self->_arc_obj, self->_rotation);
}

void PendingIndicator::timeout_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<PendingIndicator *>(lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}

	self->_timeout_timer = nullptr;
	self->set_pending(false);
}

} // namespace ui
