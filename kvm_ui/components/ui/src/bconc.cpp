#include <algorithm>

#include "ui/bconc.hpp"

namespace ui
{

BConC::BConC(lv_obj_t *parent, uint32_t fill_color)
	: Component(parent)
	, _fill_color(fill_color)
{
}

void BConC::create()
{
	if (_obj != nullptr) {
		return;
	}

	_obj = lv_obj_create(_parent);
	lv_obj_set_size(_obj, WIDTH, HEIGHT);
	lv_obj_set_style_bg_color(_obj, lv_color_hex(COLOR_DEFAULT), 0);
	lv_obj_set_style_bg_opa(_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_obj, RADIUS, 0);
	lv_obj_set_style_border_width(_obj, 0, 0);
	lv_obj_set_style_pad_all(_obj, 0, 0);
	lv_obj_set_style_clip_corner(_obj, true, 0);
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_obj, LV_OBJ_FLAG_CLICKABLE);

	_fill_obj = lv_obj_create(_obj);
	lv_obj_set_size(_fill_obj, 0, HEIGHT);
	lv_obj_set_pos(_fill_obj, 0, 0);
	lv_obj_set_style_bg_color(_fill_obj, lv_color_hex(_fill_color), 0);
	lv_obj_set_style_bg_opa(_fill_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_fill_obj, RADIUS, 0);
	lv_obj_set_style_border_width(_fill_obj, 0, 0);
	lv_obj_set_style_pad_all(_fill_obj, 0, 0);
	lv_obj_clear_flag(_fill_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_fill_obj, LV_OBJ_FLAG_CLICKABLE);

	_icon_obj = lv_obj_create(_obj);
	lv_obj_set_size(_icon_obj, ICON_SIZE_MAX, ICON_SIZE_MAX);
	lv_obj_set_pos(_icon_obj, RADIUS - (ICON_SIZE_MAX / 2),
		       (HEIGHT - ICON_SIZE_MAX) / 2);
	lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_icon_obj, 0, 0);
	lv_obj_set_style_pad_all(_icon_obj, 0, 0);
	lv_obj_add_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_CLICKABLE);

	lv_obj_add_event_cb(_obj, press_event_cb, LV_EVENT_PRESSED, this);
	lv_obj_add_event_cb(_obj, pressing_event_cb, LV_EVENT_PRESSING, this);
	lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_RELEASED, this);
	lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_PRESS_LOST, this);
	lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_CANCEL, this);

	update_visual();
}

void BConC::set_left_image(const lv_image_dsc_t *src)
{
	if (_icon_obj == nullptr) {
		return;
	}

	if (_image_obj == nullptr) {
		_image_obj = lv_image_create(_icon_obj);
		lv_obj_center(_image_obj);
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_CLICKABLE);
	}

	if (src != nullptr) {
		lv_image_set_src(_image_obj, src);
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
	} else {
		lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
	}
}

void BConC::set_range(int min, int max)
{
	if (min >= max) {
		return;
	}
	_min = min;
	_max = max;
	_value = clamp_value(_value, _min, _max);
	update_visual();
}

void BConC::set_value(int value)
{
	const int clamped = clamp_value(value, _min, _max);
	if (clamped == _value) {
		return;
	}

	_value = clamped;
	update_visual();

	if (_value_changed_callback) {
		_value_changed_callback(_value);
	}
}

void BConC::on_value_changed(ValueChangedCallback callback)
{
	_value_changed_callback = std::move(callback);
}

bool BConC::is_interaction_guard_active() const
{
	if (_is_interacting) {
		return true;
	}
	if (!_has_interacted) {
		return false;
	}

	const auto elapsed =
		std::chrono::duration_cast<std::chrono::milliseconds>(
			std::chrono::steady_clock::now() -
			_last_interaction_end);
	return elapsed.count() < INTERACTION_GUARD_MS;
}

void BConC::clear_interaction_guard()
{
	_is_interacting = false;
	_has_interacted = false;
	_last_interaction_end = {};
}

void BConC::update_visual()
{
	if (_obj == nullptr || _fill_obj == nullptr) {
		return;
	}

	const int32_t range = _max - _min;
	int32_t fill_width = HEIGHT; // 最小值显示为一个圆形填充

	if (range > 0) {
		const int32_t fill_span = WIDTH - HEIGHT;
		fill_width += ((_value - _min) * fill_span) / range;
	}
	fill_width = std::clamp<int32_t>(fill_width, HEIGHT, WIDTH);

	lv_obj_set_size(_fill_obj, fill_width, HEIGHT);
}

void BConC::update_from_touch(int32_t x)
{
	const int32_t local_x = std::clamp<int32_t>(x, 0, WIDTH - 1);
	const int32_t range = _max - _min;
	static constexpr int32_t EDGE_COMPENSATION = 20;

	int value = _min;
	if (range > 0) {
		if (local_x <= EDGE_COMPENSATION) {
			set_value(_min);
			return;
		}
		if (local_x >= (WIDTH - 1 - EDGE_COMPENSATION)) {
			set_value(_max);
			return;
		}

		const int32_t usable_span =
			(WIDTH - 1) - (EDGE_COMPENSATION * 2);
		const int32_t shifted_x = local_x - EDGE_COMPENSATION;
		value = _min +
			static_cast<int>((shifted_x * range + usable_span / 2) /
					 usable_span);
	}
	set_value(value);
}

int BConC::clamp_value(int value, int min, int max)
{
	return std::clamp(value, min, max);
}

void BConC::begin_interaction()
{
	_is_interacting = true;
	_has_interacted = true;
}

void BConC::end_interaction()
{
	if (!_is_interacting) {
		return;
	}
	_is_interacting = false;
	_last_interaction_end = std::chrono::steady_clock::now();
}

void BConC::press_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConC *>(lv_event_get_user_data(e));
	if (self == nullptr || self->_obj == nullptr) {
		return;
	}

	lv_indev_t *indev = lv_indev_get_act();
	if (indev == nullptr) {
		return;
	}

	self->begin_interaction();

	lv_point_t point;
	lv_indev_get_point(indev, &point);
	lv_area_t coords;
	lv_obj_get_coords(self->_obj, &coords);
	const int32_t local_x = point.x - coords.x1;
	self->update_from_touch(local_x);
}

void BConC::pressing_event_cb(lv_event_t *e)
{
	press_event_cb(e);
}

void BConC::release_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConC *>(lv_event_get_user_data(e));
	if (self == nullptr) {
		return;
	}
	self->end_interaction();
}

} // namespace ui
