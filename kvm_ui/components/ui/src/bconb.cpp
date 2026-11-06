#include "ui/bconb.hpp"

#include <src/core/lv_obj.h>

#include "lvgl.h"

#include "pending_indicator.hpp"

namespace ui
{

BConB::BConB(lv_obj_t *parent, uint32_t active_color)
	: Component(parent)
	, _active_color(active_color)
	, _current_color(COLOR_DEFAULT)
	, _state(State::DEFAULT)
	, _target_state(State::DEFAULT)
	, _state_dirty(false)
	, _apply_scheduled(false)
	, _is_pressed(false)
	, _click_callback(nullptr)
	, _icon_obj(nullptr)
{
}

BConB::~BConB()
{
	destroy();
}

void BConB::create()
{
	if (_obj != nullptr) {
		return;
	}

	// 创建主容器
	_obj = lv_obj_create(_parent);
	lv_obj_set_size(_obj, WIDTH, HEIGHT);
	lv_obj_set_style_bg_color(_obj, lv_color_hex(COLOR_DEFAULT), 0);
	lv_obj_set_style_bg_opa(_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_obj, RADIUS, 0);
	lv_obj_set_style_border_width(_obj, 0, 0);
	lv_obj_set_style_clip_corner(_obj, true, 0);
	lv_obj_remove_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);

	// 创建图标占位符 (白色矩形)，居中
	_icon_obj = lv_obj_create(_obj);
	lv_obj_set_size(_icon_obj, ICON_SIZE, ICON_SIZE);
	lv_obj_set_style_bg_color(_icon_obj, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_icon_obj, 0, 0);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_center(_icon_obj);

	_pending_indicator = std::make_unique<PendingIndicator>(
		_icon_obj, [this](bool) {
			apply_icon_content_visibility();
		});
	_pending_indicator->set_timeout(_pending_timeout_ms);
	_pending_indicator->set_timeout_enabled(_pending_timeout_enabled);

	// 添加事件监听
	lv_obj_add_flag(_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(_obj, press_event_cb, LV_EVENT_PRESSED, this);
	lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_RELEASED, this);
	lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_PRESS_LOST, this);
	lv_obj_add_event_cb(_obj, click_event_cb, LV_EVENT_CLICKED, this);

	update_appearance();
}

void BConB::destroy()
{
	set_pending(false);
	_pending_indicator.reset();
	cancel_deferred_apply();
	_click_callback = nullptr;
	_icon_obj = nullptr;
	_image_obj = nullptr;
	_symbol_label = nullptr;
	_has_image = false;
	_has_symbol = false;
	Component::destroy();
}

void BConB::set_state(State state)
{
	_state = state;
	_target_state = state;
	_state_dirty = false;
	update_appearance();
}

void BConB::set_visual_state(VisualState state, ApplyPolicy policy)
{
	_target_state = (state == VisualState::ACTIVE) ? State::ACTIVE :
							 State::DEFAULT;

	if (policy == ApplyPolicy::IMMEDIATE || _obj == nullptr) {
		_state = _target_state;
		_state_dirty = false;
		update_appearance();
		return;
	}

	_state_dirty = true;
	schedule_deferred_apply();
}

ControlState::VisualState BConB::get_visual_state() const
{
	const State state = _state_dirty ? _target_state : _state;
	return (state == State::ACTIVE) ? VisualState::ACTIVE :
					  VisualState::DEFAULT;
}

void BConB::set_pending(bool pending)
{
	if (_pending_indicator == nullptr) {
		return;
	}

	update_pending_arc_color();
	_pending_indicator->set_pending(pending);
}

bool BConB::is_pending() const
{
	return _pending_indicator != nullptr && _pending_indicator->is_pending();
}

void BConB::set_pending_timeout(uint32_t timeout_ms)
{
	const uint32_t next_timeout = timeout_ms == 0 ? PENDING_TIMEOUT_MS :
							timeout_ms;
	if (_pending_timeout_ms == next_timeout) {
		return;
	}

	_pending_timeout_ms = next_timeout;
	if (_pending_indicator != nullptr) {
		_pending_indicator->set_timeout(_pending_timeout_ms);
	}
}

void BConB::set_pending_timeout_enabled(bool enabled)
{
	if (_pending_timeout_enabled == enabled) {
		return;
	}

	_pending_timeout_enabled = enabled;
	if (_pending_indicator != nullptr) {
		_pending_indicator->set_timeout_enabled(_pending_timeout_enabled);
	}
}

void BConB::on_click(ClickCallback callback)
{
	_click_callback = std::move(callback);
}

void BConB::set_image(const lv_image_dsc_t *src)
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
		_has_image = true;
		_has_symbol = false;
		if (_symbol_label != nullptr) {
			lv_obj_add_flag(_symbol_label, LV_OBJ_FLAG_HIDDEN);
		}
		lv_image_set_src(_image_obj, src);
		lv_image_set_pivot(_image_obj, src->header.w / 2,
					   src->header.h / 2);
		lv_obj_set_size(_icon_obj, ICON_SIZE_MAX, ICON_SIZE_MAX);
		lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_TRANSP, 0);
		lv_obj_add_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	} else {
		_has_image = false;
		lv_obj_set_size(_icon_obj, ICON_SIZE, ICON_SIZE);
		lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		if (!_has_symbol) {
			lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_COVER, 0);
			lv_obj_clear_flag(_icon_obj,
						  LV_OBJ_FLAG_OVERFLOW_VISIBLE);
		}
	}
	apply_icon_content_visibility();
}

void BConB::set_symbol(const char *symbol)
{
	if (_icon_obj == nullptr) {
		return;
	}
	if (_symbol_label == nullptr) {
		_symbol_label = lv_label_create(_icon_obj);
		lv_obj_set_style_text_color(_symbol_label, lv_color_white(), 0);
		lv_obj_set_style_text_font(_symbol_label,
					   &lv_font_montserrat_28, 0);
		lv_obj_clear_flag(_symbol_label, LV_OBJ_FLAG_CLICKABLE);
	}

	if (symbol != nullptr && symbol[0] != '\0') {
		_has_symbol = true;
		_has_image = false;
		if (_image_obj != nullptr) {
			lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		}
		lv_label_set_text(_symbol_label, symbol);
		lv_obj_set_size(_icon_obj, ICON_SIZE_MAX, ICON_SIZE_MAX);
		lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_TRANSP, 0);
		lv_obj_add_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
		lv_obj_center(_symbol_label);
		apply_icon_content_visibility();
		return;
	}

	_has_symbol = false;
	lv_obj_add_flag(_symbol_label, LV_OBJ_FLAG_HIDDEN);
	if (!_has_image) {
		lv_obj_set_size(_icon_obj, ICON_SIZE, ICON_SIZE);
		lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_COVER, 0);
		lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	}
	apply_icon_content_visibility();
}

void BConB::set_symbol_font(const lv_font_t *font)
{
	if (_symbol_label == nullptr || font == nullptr) {
		return;
	}
	lv_obj_set_style_text_font(_symbol_label, font, 0);
	lv_obj_center(_symbol_label);
}

void BConB::set_image_rotation(int32_t angle)
{
	if (_image_obj == nullptr) {
		return;
	}
	lv_image_set_rotation(_image_obj, angle);
}

void BConB::set_icon_offset(int16_t x, int16_t y)
{
	if (_icon_obj == nullptr) {
		return;
	}
	lv_obj_align(_icon_obj, LV_ALIGN_CENTER, x, y);
}

uint32_t BConB::apply_press_overlay(uint32_t base_color)
{
	uint8_t r = (base_color >> 16) & 0xFF;
	uint8_t g = (base_color >> 8) & 0xFF;
	uint8_t b = base_color & 0xFF;

	r = static_cast<uint8_t>(r * 0.8f);
	g = static_cast<uint8_t>(g * 0.8f);
	b = static_cast<uint8_t>(b * 0.8f);

	return (r << 16) | (g << 8) | b;
}

void BConB::update_appearance()
{
	if (_obj == nullptr) {
		return;
	}

	uint32_t color;
	if (_is_pressed) {
		color = apply_press_overlay(_current_color);
	} else if (_state == State::ACTIVE) {
		color = _active_color;
	} else {
		color = COLOR_DEFAULT;
	}

	_current_color = _is_pressed ?
				 _current_color :
					 (_state == State::ACTIVE ? _active_color :
								    COLOR_DEFAULT);
	lv_obj_set_style_bg_color(_obj, lv_color_hex(color), 0);
	update_pending_arc_color();
}

void BConB::apply_pending_state()
{
	if (!_state_dirty) {
		return;
	}

	_state = _target_state;
	_state_dirty = false;
	update_appearance();
}

void BConB::schedule_deferred_apply()
{
	if (_obj == nullptr || _apply_scheduled) {
		return;
	}

	_apply_scheduled = true;
	lv_async_call(async_apply_cb, this);
}

void BConB::cancel_deferred_apply()
{
	if (!_apply_scheduled) {
		return;
	}

	lv_async_call_cancel(async_apply_cb, this);
	_apply_scheduled = false;
}

uint32_t BConB::pending_arc_color() const
{
	return _current_color == _active_color ? 0xFFFFFF : _active_color;
}

void BConB::update_pending_arc_color()
{
	if (_pending_indicator == nullptr) {
		return;
	}

	_pending_indicator->set_arc_color(pending_arc_color());
}

void BConB::apply_icon_content_visibility()
{
	const bool show_content = !is_pending();
	if (_image_obj != nullptr) {
		if (_has_image && show_content) {
			lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		}
	}
	if (_symbol_label != nullptr) {
		if (_has_symbol && show_content) {
			lv_obj_clear_flag(_symbol_label, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_symbol_label, LV_OBJ_FLAG_HIDDEN);
		}
	}
}

void BConB::press_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConB *>(lv_event_get_user_data(e));
	if (self == nullptr || self->_obj == nullptr) {
		return;
	}

	self->_is_pressed = true;
	self->update_appearance();
}

void BConB::release_event_cb(lv_event_t *e)
{
	(void)e;
	auto *self = static_cast<BConB *>(lv_event_get_user_data(e));
	if (self == nullptr || self->_obj == nullptr) {
		return;
	}

	self->_is_pressed = false;
	self->update_appearance();
}

void BConB::click_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConB *>(lv_event_get_user_data(e));
	if (self == nullptr) {
		return;
	}

	if (self->_click_callback) {
		self->_click_callback(self);
	}
}

void BConB::async_apply_cb(void *user_data)
{
	auto *self = static_cast<BConB *>(user_data);
	if (self == nullptr) {
		return;
	}

	self->_apply_scheduled = false;
	self->apply_pending_state();
}

} // namespace ui
