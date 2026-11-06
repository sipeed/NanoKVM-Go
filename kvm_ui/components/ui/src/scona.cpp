#include "ui/scona.hpp"

#include <cstring>

#include "lvgl.h"

namespace ui
{

SConA::SConA(lv_obj_t *parent, const char *text, TextAlign align,
		     const lv_font_t *text_font)
	: SConA(parent, text, align, text_font, WIDTH)
{
}

SConA::SConA(lv_obj_t *parent, const char *text, TextAlign align,
	     const lv_font_t *text_font, uint16_t width)
	: Component(parent)
	, _text(text != nullptr ? text : "")
	, _text_font(text_font)
	, _align(align)
	, _width(width)
	, _dual_mode(false)
	, _left_label_obj(nullptr)
	, _right_label_obj(nullptr)
{
}

SConA::~SConA()
{
	destroy();
}

void SConA::create()
{
	if (_obj != nullptr) {
		return;
	}

	// 创建主容器
	_obj = lv_obj_create(_parent);
	lv_obj_set_size(_obj, _width, HEIGHT);
	lv_obj_set_style_bg_opa(_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_radius(_obj, RADIUS, 0);
	lv_obj_set_style_border_width(_obj, 0, 0);
	lv_obj_set_style_clip_corner(_obj, true, 0);
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);

	// 清除可点击标志
	lv_obj_clear_flag(_obj, LV_OBJ_FLAG_CLICKABLE);

	const lv_font_t *const text_font =
		_text_font != nullptr ? _text_font : &lv_font_montserrat_20;

	// 左侧文字标签（单文本模式下也复用该标签）
	_left_label_obj = lv_label_create(_obj);
	lv_label_set_text(_left_label_obj, _text.c_str());
	lv_obj_set_style_text_color(_left_label_obj, lv_color_white(), 0);
	lv_obj_set_style_text_font(_left_label_obj, text_font, 0);

	// 右侧文字标签（双文本模式使用）
	_right_label_obj = lv_label_create(_obj);
	lv_label_set_text(_right_label_obj, "");
	lv_obj_set_style_text_color(_right_label_obj, lv_color_white(), 0);
	lv_obj_set_style_text_font(_right_label_obj, text_font, 0);
	lv_obj_add_flag(_right_label_obj, LV_OBJ_FLAG_HIDDEN);

	// 设置对齐
	update_alignment();
}

void SConA::set_text(const char *text)
{
	const std::string value = text != nullptr ? text : "";
	if (_text == value) {
		return;
	}

	_text = value;
	if (_left_label_obj != nullptr && !_dual_mode) {
		set_label_text_if_changed(_left_label_obj, _text.c_str());
	}
}

bool SConA::set_image_source(const void *source)
{
	if (source == nullptr) {
		clear_image_source();
		return true;
	}
	if (_obj == nullptr || !lv_obj_is_valid(_obj) || _dual_mode) {
		return false;
	}

	if (_image_obj == nullptr || !lv_obj_is_valid(_image_obj)) {
		_image_obj = lv_image_create(_obj);
		if (_image_obj == nullptr) {
			return false;
		}
		_image_source = nullptr;
		lv_obj_set_pos(_image_obj, 0, 0);
		lv_obj_set_size(_image_obj, _width, HEIGHT);
		lv_image_set_inner_align(_image_obj, LV_IMAGE_ALIGN_TOP_LEFT);
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_SCROLLABLE);
	}

	if (_image_source != source) {
		lv_image_set_src(_image_obj, source);
		if (lv_image_get_src(_image_obj) != source) {
			return false;
		}
		_image_source = source;
	}
	apply_image_visibility();
	return true;
}

void SConA::clear_image_source()
{
	_image_source = nullptr;
	if (_image_obj != nullptr && lv_obj_is_valid(_image_obj)) {
		lv_image_set_src(_image_obj, nullptr);
	}
	apply_image_visibility();
}

void SConA::set_align(TextAlign align)
{
	if (_align == align) {
		return;
	}

	_align = align;
	update_alignment();
}

void SConA::set_dual_text(const char *left_text, const char *right_text)
{
	const std::string left = left_text != nullptr ? left_text : "";
	const std::string right = right_text != nullptr ? right_text : "";
	if (_dual_mode && _dual_left_text == left && _dual_right_text == right) {
		return;
	}

	clear_image_source();
	_dual_left_text = left;
	_dual_right_text = right;
	_dual_mode = true;
	update_alignment();
}

void SConA::clear_dual_text()
{
	if (!_dual_mode) {
		return;
	}

	_dual_mode = false;
	update_alignment();
}

void SConA::set_single_text_scroll_circular(bool enabled)
{
	if (_single_text_scroll_circular == enabled) {
		return;
	}

	_single_text_scroll_circular = enabled;
	update_alignment();
}

void SConA::destroy()
{
	if (_image_obj != nullptr && lv_obj_is_valid(_image_obj)) {
		lv_image_set_src(_image_obj, nullptr);
	}
	_image_source = nullptr;
	_left_label_obj = nullptr;
	_right_label_obj = nullptr;
	_image_obj = nullptr;
	Component::destroy();
}

void SConA::update_alignment()
{
	if (_left_label_obj == nullptr) {
		return;
	}

	if (_dual_mode) {
		if (lv_label_get_long_mode(_left_label_obj) != LV_LABEL_LONG_WRAP) {
			lv_label_set_long_mode(_left_label_obj, LV_LABEL_LONG_WRAP);
		}
		lv_obj_remove_local_style_prop(_left_label_obj, LV_STYLE_WIDTH,
					       LV_PART_MAIN);
		lv_obj_remove_local_style_prop(_left_label_obj, LV_STYLE_TEXT_ALIGN,
					       LV_PART_MAIN);
		set_label_text_if_changed(_left_label_obj,
						  _dual_left_text.c_str());
		lv_obj_align(_left_label_obj, LV_ALIGN_LEFT_MID, 12, 0);

		if (_right_label_obj != nullptr) {
			set_label_text_if_changed(_right_label_obj,
						  _dual_right_text.c_str());
			lv_obj_align(_right_label_obj, LV_ALIGN_RIGHT_MID, -12,
					     0);
			lv_obj_clear_flag(_right_label_obj, LV_OBJ_FLAG_HIDDEN);
		}
		apply_image_visibility();
		return;
	}

	set_label_text_if_changed(_left_label_obj, _text.c_str());
	if (_right_label_obj != nullptr) {
		lv_obj_add_flag(_right_label_obj, LV_OBJ_FLAG_HIDDEN);
	}

	if (_single_text_scroll_circular) {
		if (lv_label_get_long_mode(_left_label_obj) !=
		    LV_LABEL_LONG_SCROLL_CIRCULAR) {
			lv_label_set_long_mode(_left_label_obj,
					       LV_LABEL_LONG_SCROLL_CIRCULAR);
		}
		lv_obj_set_width(_left_label_obj,
				 SINGLE_TEXT_HORIZONTAL_INSET * 2 < _width ?
					 _width - SINGLE_TEXT_HORIZONTAL_INSET * 2 :
					 _width);
		switch (_align) {
		case TextAlign::LEFT:
			lv_obj_set_style_text_align(_left_label_obj,
						LV_TEXT_ALIGN_LEFT, LV_PART_MAIN);
			break;
		case TextAlign::RIGHT:
			lv_obj_set_style_text_align(_left_label_obj,
						LV_TEXT_ALIGN_RIGHT, LV_PART_MAIN);
			break;
		case TextAlign::CENTER:
		default:
			lv_obj_set_style_text_align(_left_label_obj,
						LV_TEXT_ALIGN_CENTER, LV_PART_MAIN);
			break;
		}
	} else {
		if (lv_label_get_long_mode(_left_label_obj) != LV_LABEL_LONG_WRAP) {
			lv_label_set_long_mode(_left_label_obj, LV_LABEL_LONG_WRAP);
		}
		lv_obj_remove_local_style_prop(_left_label_obj, LV_STYLE_WIDTH,
					       LV_PART_MAIN);
		lv_obj_remove_local_style_prop(_left_label_obj, LV_STYLE_TEXT_ALIGN,
					       LV_PART_MAIN);
	}

	switch (_align) {
	case TextAlign::LEFT:
		lv_obj_align(_left_label_obj, LV_ALIGN_LEFT_MID,
			     SINGLE_TEXT_HORIZONTAL_INSET, 0);
		break;
	case TextAlign::RIGHT:
		lv_obj_align(_left_label_obj, LV_ALIGN_RIGHT_MID,
			     -SINGLE_TEXT_HORIZONTAL_INSET, 0);
		break;
	case TextAlign::CENTER:
	default:
		lv_obj_align(_left_label_obj, LV_ALIGN_CENTER, 0, 0);
		break;
	}

	apply_image_visibility();
}

void SConA::set_label_text_if_changed(lv_obj_t *label, const char *text)
{
	if (label == nullptr || !lv_obj_is_valid(label)) {
		return;
	}

	const char *const value = text != nullptr ? text : "";
	if (std::strcmp(lv_label_get_text(label), value) != 0) {
		lv_label_set_text(label, value);
	}
}

void SConA::apply_image_visibility()
{
	const bool image_visible = _image_source != nullptr && _image_obj != nullptr &&
				   lv_obj_is_valid(_image_obj);
	if (_image_obj != nullptr && lv_obj_is_valid(_image_obj)) {
		if (image_visible) {
			lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		}
	}
	if (_left_label_obj != nullptr && lv_obj_is_valid(_left_label_obj)) {
		if (image_visible) {
			lv_obj_add_flag(_left_label_obj, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(_left_label_obj, LV_OBJ_FLAG_HIDDEN);
		}
	}
	if (_right_label_obj != nullptr && lv_obj_is_valid(_right_label_obj)) {
		if (image_visible || !_dual_mode) {
			lv_obj_add_flag(_right_label_obj, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(_right_label_obj, LV_OBJ_FLAG_HIDDEN);
		}
	}
}

} // namespace ui
