#include "ui/subpage_base.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_settings_main_return);
}

namespace ui
{

SubPageBase::SubPageBase(PageType type, const char *title, uint16_t row_gap,
			 uint16_t top_padding)
	: Page(type)
	, _title(title == nullptr ? "" : title)
	, _row_gap(row_gap)
	, _top_padding(top_padding)
{
}

void SubPageBase::create(lv_obj_t *parent)
{
	if (_container != nullptr) {
		return;
	}

	_container = lv_obj_create(parent);
	lv_obj_set_size(_container, WIDTH, HEIGHT);
	lv_obj_set_style_bg_opa(_container, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_container, 0, 0);
	lv_obj_set_style_pad_all(_container, 0, 0);
	lv_obj_set_scrollbar_mode(_container, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_container, LV_OBJ_FLAG_SCROLLABLE);

	_scroll_content = lv_obj_create(_container);
	lv_obj_set_size(_scroll_content, WIDTH, HEIGHT);
	lv_obj_set_pos(_scroll_content, 0, 0);
	lv_obj_set_style_bg_opa(_scroll_content, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_scroll_content, 0, 0);
	lv_obj_set_style_pad_all(_scroll_content, 0, 0);
	lv_obj_set_flex_flow(_scroll_content, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_style_pad_left(_scroll_content, LEFT_MARGIN, 0);
	lv_obj_set_style_pad_top(_scroll_content, _top_padding, 0);
	lv_obj_set_style_pad_row(_scroll_content, _row_gap, 0);
	lv_obj_set_scroll_dir(_scroll_content, LV_DIR_VER);
	lv_obj_set_scrollbar_mode(_scroll_content, LV_SCROLLBAR_MODE_OFF);
	lv_obj_add_flag(_scroll_content, LV_OBJ_FLAG_SCROLLABLE);

	auto back = std::make_unique<BConA>(_scroll_content, _title.c_str(),
					    PAGE_BG_COLOR);
	_back_btn = back.get();
	back->create();
	back->set_default_color(PAGE_BG_COLOR);
	back->set_active(true);
	back->set_left_image(&img_settings_main_return);
	back->set_left_image_visible(_back_navigation_available);
	back->on_click([this](BConA *) {
		if (on_back_click) {
			on_back_click();
		}
	});
	_components.push_back(std::move(back));

	build_content();
}

void SubPageBase::set_title(const char *title)
{
	_title = title == nullptr ? "" : title;
	if (_back_btn != nullptr) {
		_back_btn->set_text(_title.c_str());
	}
}

void SubPageBase::set_back_navigation_available(bool available)
{
	_back_navigation_available = available;
	if (_back_btn == nullptr) {
		return;
	}

	lv_obj_t *back_obj = _back_btn->get_obj();
	if (back_obj == nullptr || !lv_obj_is_valid(back_obj)) {
		return;
	}

	_back_btn->set_left_image_visible(available);
	if (available) {
		lv_obj_add_flag(back_obj, LV_OBJ_FLAG_CLICKABLE);
		return;
	}

	lv_obj_clear_flag(back_obj, LV_OBJ_FLAG_CLICKABLE);
}

void SubPageBase::destroy()
{
	set_back_navigation_available(true);
	_components.clear();
	_scroll_content = nullptr;
	_back_btn = nullptr;
	_back_navigation_available = true;
	on_subpage_destroy();
	Page::destroy();
}

} // namespace ui
