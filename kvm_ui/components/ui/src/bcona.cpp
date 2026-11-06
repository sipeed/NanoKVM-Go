#include "ui/bcona.hpp"

#include "lvgl.h"

#include "pending_indicator.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_30);
}

namespace ui
{

BConA::BConA(lv_obj_t *parent, const char *text, uint32_t active_color,
	     bool has_subpage)
	: Component(parent)
	, _text(text == nullptr ? "" : text)
	, _active_color(active_color)
	, _default_color(COLOR_DEFAULT)
	, _current_color(COLOR_DEFAULT)
	, _state(State::DEFAULT)
	, _target_state(State::DEFAULT)
	, _state_dirty(false)
	, _apply_scheduled(false)
	, _has_subpage(has_subpage)
	, _is_pressed(false)
	, _click_callback(nullptr)
	, _icon_obj(nullptr)
	, _label_obj(nullptr)
{
}

BConA::~BConA()
{
	destroy();
}

void BConA::create()
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
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);

	// 设置 Flex 布局 (图标左、文字右)
	lv_obj_set_flex_flow(_obj, LV_FLEX_FLOW_ROW);
	lv_obj_set_flex_align(_obj, LV_FLEX_ALIGN_SPACE_BETWEEN,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
	lv_obj_set_style_pad_left(_obj, PADDING_LEFT, 0);
	lv_obj_set_style_pad_column(_obj, PADDING_COLUMN, 0);

	// 创建图标占位符 (白色矩形)
	_icon_obj = lv_obj_create(_obj);
	lv_obj_set_size(_icon_obj, ICON_SIZE, ICON_SIZE);
	lv_obj_set_style_bg_color(_icon_obj, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_icon_obj, ICON_RADIUS, 0);
	lv_obj_set_style_border_width(_icon_obj, 0, 0);
	lv_obj_set_style_pad_all(_icon_obj, 0, 0);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_CLICKABLE);

	// 创建图标内部符号标签（居中于占位符内）
	_icon_inner_label = lv_label_create(_icon_obj);
	lv_label_set_text(_icon_inner_label, "");
	lv_obj_set_style_text_color(_icon_inner_label, lv_color_white(), 0);
	lv_obj_center(_icon_inner_label);

	// 创建文字标签
	_label_obj = lv_label_create(_obj);
	lv_label_set_text(_label_obj, _text.c_str());
	lv_obj_set_style_text_color(_label_obj, lv_color_white(), 0);
	lv_obj_set_style_text_font(_label_obj, &MontserratBold_30, 0);

	// 创建左侧符号标签（默认隐藏）
	_left_icon_label = lv_label_create(_obj);
	lv_label_set_text(_left_icon_label, "");
	lv_obj_set_style_text_color(_left_icon_label, lv_color_white(), 0);
	lv_obj_add_flag(_left_icon_label, LV_OBJ_FLAG_HIDDEN);

	// 创建右侧符号标签（默认隐藏）
	_right_icon_label = lv_label_create(_obj);
	lv_label_set_text(_right_icon_label, "");
	lv_obj_set_style_text_color(_right_icon_label, lv_color_white(), 0);
	lv_obj_add_flag(_right_icon_label, LV_OBJ_FLAG_HIDDEN);

	_pending_indicator = std::make_unique<PendingIndicator>(
		_icon_obj, [this](bool visible) {
			apply_pending_content_visibility(visible);
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

void BConA::destroy()
{
	set_pending(false);
	_pending_indicator.reset();
	cancel_deferred_apply();
	_click_callback = nullptr;
	_icon_obj = nullptr;
	_icon_inner_label = nullptr;
	_image_obj = nullptr;
	_label_obj = nullptr;
	_left_icon_label = nullptr;
	_right_icon_label = nullptr;
	Component::destroy();
}

void BConA::set_left_symbol(const char *symbol)
{
	if (_icon_inner_label == nullptr) {
		return;
	}
	if (symbol != nullptr && symbol[0] != '\0') {
		lv_label_set_text(_icon_inner_label, symbol);
		lv_obj_clear_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
	} else {
		lv_obj_add_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
	}
}

void BConA::set_left_icon_bg(uint32_t color, uint8_t radius)
{
	if (_icon_obj == nullptr) {
		return;
	}
	lv_obj_set_style_bg_color(_icon_obj, lv_color_hex(color), 0);
	lv_obj_set_style_radius(_icon_obj, radius, 0);
}

void BConA::set_right_symbol(const char *symbol)
{
	if (_right_icon_label == nullptr) {
		return;
	}
	if (symbol != nullptr && symbol[0] != '\0') {
		lv_label_set_text(_right_icon_label, symbol);
		lv_obj_clear_flag(_right_icon_label, LV_OBJ_FLAG_HIDDEN);
	} else {
		lv_obj_add_flag(_right_icon_label, LV_OBJ_FLAG_HIDDEN);
	}
}

void BConA::set_left_image(const lv_image_dsc_t *src)
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
		_left_image_has_source = true;
		_left_image_visible = true;
		apply_left_image_visibility();
		if (_icon_inner_label != nullptr) {
			lv_obj_add_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
		}
		lv_obj_set_size(_icon_obj, ICON_SIZE_MAX, ICON_SIZE_MAX);
		// 让 60x60 区域中心与左侧圆弧圆心 (RADIUS, HEIGHT/2) 对齐
		lv_obj_set_style_margin_left(
			_icon_obj,
			-(PADDING_LEFT - (RADIUS - ICON_SIZE_MAX / 2)), 0);
		// 隐藏占位符白色背景，让图标直接显示在按钮背景上
		lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_TRANSP, 0);
		lv_obj_add_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	} else {
		_left_image_has_source = false;
		_left_image_visible = false;
		lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		if (_icon_inner_label != nullptr) {
			if (!is_pending()) {
				lv_obj_clear_flag(_icon_inner_label,
							  LV_OBJ_FLAG_HIDDEN);
			}
		}
		lv_obj_set_size(_icon_obj, ICON_SIZE, ICON_SIZE);
		lv_obj_set_style_margin_left(_icon_obj, 0, 0);
		// 恢复白色占位符背景
		lv_obj_set_style_bg_color(_icon_obj, lv_color_white(), 0);
		lv_obj_set_style_bg_opa(_icon_obj, LV_OPA_COVER, 0);
		lv_obj_clear_flag(_icon_obj, LV_OBJ_FLAG_OVERFLOW_VISIBLE);
	}
}

void BConA::set_left_image_visible(bool visible)
{
	_left_image_visible = _left_image_has_source && visible;
	apply_left_image_visibility();
}

bool BConA::is_left_image_visible() const
{
	return _left_image_has_source && _left_image_visible;
}

void BConA::set_text(const char *text)
{
	_text = std::string(text == nullptr ? "" : text);
	if (_label_obj != nullptr) {
		lv_label_set_text(_label_obj, _text.c_str());
	}
}

void BConA::set_text_font(const lv_font_t *font)
{
	if (_label_obj != nullptr && font != nullptr) {
		lv_obj_set_style_text_font(_label_obj, font, 0);
	}
}

void BConA::set_text_align(lv_text_align_t align)
{
	if (_label_obj != nullptr) {
		lv_obj_set_style_text_align(_label_obj, align, 0);
	}
}

void BConA::set_text_scrolling(uint16_t width)
{
	if (_obj == nullptr || _label_obj == nullptr || width == 0) {
		return;
	}

	lv_obj_set_style_pad_right(_obj, PADDING_LEFT, 0);
	lv_label_set_long_mode(_label_obj, LV_LABEL_LONG_SCROLL);
	lv_obj_set_width(_label_obj, width);
}

void BConA::set_state(State state)
{
	_state = state;
	_target_state = state;
	_state_dirty = false;
	update_appearance();
}

void BConA::set_color(uint32_t color)
{
	_active_color = color;
	update_appearance();
}

void BConA::set_default_color(uint32_t color)
{
	_default_color = color;
	update_appearance();
}

void BConA::set_visual_state(VisualState state, ApplyPolicy policy)
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

ControlState::VisualState BConA::get_visual_state() const
{
	const State state = _state_dirty ? _target_state : _state;
	return (state == State::ACTIVE) ? VisualState::ACTIVE :
					  VisualState::DEFAULT;
}

void BConA::set_pending(bool pending)
{
	if (_pending_indicator == nullptr) {
		return;
	}

	update_pending_arc_color();
	_pending_indicator->set_pending(pending);
}

bool BConA::is_pending() const
{
	return _pending_indicator != nullptr && _pending_indicator->is_pending();
}

void BConA::set_pending_timeout(uint32_t timeout_ms)
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

void BConA::set_pending_timeout_enabled(bool enabled)
{
	if (_pending_timeout_enabled == enabled) {
		return;
	}
	_pending_timeout_enabled = enabled;
	if (_pending_indicator != nullptr) {
		_pending_indicator->set_timeout_enabled(_pending_timeout_enabled);
	}
}

void BConA::on_click(ClickCallback callback)
{
	_click_callback = std::move(callback);
}

uint32_t BConA::apply_press_overlay(uint32_t base_color)
{
	uint8_t r = (base_color >> 16) & 0xFF;
	uint8_t g = (base_color >> 8) & 0xFF;
	uint8_t b = base_color & 0xFF;

	r = static_cast<uint8_t>(r * 0.8f);
	g = static_cast<uint8_t>(g * 0.8f);
	b = static_cast<uint8_t>(b * 0.8f);

	return (r << 16) | (g << 8) | b;
}

void BConA::update_appearance()
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
		color = _default_color;
	}

	_current_color = _is_pressed ?
				 _current_color :
				 (_state == State::ACTIVE ? _active_color :
							    _default_color);
	lv_obj_set_style_bg_color(_obj, lv_color_hex(color), 0);
	update_pending_arc_color();
}

void BConA::apply_pending_state()
{
	if (!_state_dirty) {
		return;
	}

	_state = _target_state;
	_state_dirty = false;
	update_appearance();
}

void BConA::schedule_deferred_apply()
{
	if (_obj == nullptr || _apply_scheduled) {
		return;
	}

	_apply_scheduled = true;
	lv_async_call(async_apply_cb, this);
}

void BConA::cancel_deferred_apply()
{
	if (!_apply_scheduled) {
		return;
	}

	lv_async_call_cancel(async_apply_cb, this);
	_apply_scheduled = false;
}

uint32_t BConA::pending_arc_color() const
{
	return _current_color == _active_color ? 0xFFFFFF : _active_color;
}

void BConA::update_pending_arc_color()
{
	if (_pending_indicator == nullptr) {
		return;
	}

	_pending_indicator->set_arc_color(pending_arc_color());
}

void BConA::apply_left_image_visibility()
{
	if (_image_obj == nullptr) {
		return;
	}

	if (_left_image_has_source && _left_image_visible && !is_pending()) {
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		return;
	}

	lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
}

void BConA::apply_pending_content_visibility(bool visible)
{
	if (!visible) {
		_pending_icon_label_was_visible =
			_icon_inner_label != nullptr &&
			!lv_obj_has_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
		if (_image_obj != nullptr) {
			apply_left_image_visibility();
		}
		if (_icon_inner_label != nullptr) {
			lv_obj_add_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
		}
		return;
	}

	if (_image_obj != nullptr) {
		apply_left_image_visibility();
	}
	if (_icon_inner_label != nullptr && _pending_icon_label_was_visible) {
		lv_obj_clear_flag(_icon_inner_label, LV_OBJ_FLAG_HIDDEN);
	}
	_pending_icon_label_was_visible = false;
}

void BConA::press_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConA *>(lv_event_get_user_data(e));
	if (self == nullptr || self->_obj == nullptr) {
		return;
	}

	self->_is_pressed = true;
	self->update_appearance();
}

void BConA::release_event_cb(lv_event_t *e)
{
	(void)e;
	auto *self = static_cast<BConA *>(lv_event_get_user_data(e));
	if (self == nullptr || self->_obj == nullptr) {
		return;
	}

	self->_is_pressed = false;
	self->update_appearance();
}

void BConA::click_event_cb(lv_event_t *e)
{
	auto *self = static_cast<BConA *>(lv_event_get_user_data(e));
	if (self == nullptr) {
		return;
	}

	if (self->_click_callback) {
		self->_click_callback(self);
	}
}

void BConA::async_apply_cb(void *user_data)
{
	auto *self = static_cast<BConA *>(user_data);
	if (self == nullptr) {
		return;
	}

	self->_apply_scheduled = false;
	self->apply_pending_state();
}

} // namespace ui
