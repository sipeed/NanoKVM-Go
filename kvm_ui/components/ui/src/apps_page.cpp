#include "ui/apps_page.hpp"

#include <utility>

#include <spdlog/spdlog.h>

extern "C" {
LV_FONT_DECLARE(MontserratBold_18);
}

namespace ui
{

AppsPage::AppsPage()
	: SubPageBase(PageType::SUBPAGE_APPS, "Apps")
{
}

void AppsPage::set_entries(std::vector<AppEntry> entries)
{
	if (_entries == entries) {
		return;
	}
	_entries = std::move(entries);
	if (_scroll_content == nullptr) {
		return;
	}
	// SubPageBase owns the back button as the first component. Rebuild only
	// the dynamic App buttons so the page can refresh without recreation.
	while (_components.size() > 1) {
		_components.pop_back();
	}
	build_content();
	lv_obj_update_layout(_scroll_content);
}

void AppsPage::configure_scrollbar()
{
	lv_obj_set_scrollbar_mode(_scroll_content, LV_SCROLLBAR_MODE_AUTO);
	lv_obj_set_style_width(_scroll_content, SCROLLBAR_WIDTH,
			       LV_PART_SCROLLBAR);
	lv_obj_set_style_bg_color(_scroll_content,
				  lv_color_hex(SCROLLBAR_COLOR),
				  LV_PART_SCROLLBAR);
	lv_obj_set_style_bg_opa(_scroll_content, LV_OPA_COVER,
				LV_PART_SCROLLBAR);
	lv_obj_set_style_radius(_scroll_content, LV_RADIUS_CIRCLE,
				LV_PART_SCROLLBAR);
	lv_obj_set_style_pad_right(_scroll_content, SCROLLBAR_EDGE_PADDING,
				   LV_PART_SCROLLBAR);
	lv_obj_set_style_pad_top(_scroll_content, SCROLLBAR_VERTICAL_PADDING,
				 LV_PART_SCROLLBAR);
	lv_obj_set_style_pad_bottom(_scroll_content, SCROLLBAR_VERTICAL_PADDING,
				    LV_PART_SCROLLBAR);
}

void AppsPage::build_content()
{
	configure_scrollbar();
	for (const auto &entry : _entries) {
		auto btn = std::make_unique<BConA>(
			_scroll_content, entry.name.c_str(), APP_COLOR);
		btn->create();
		btn->set_text_font(&MontserratBold_18);
		btn->set_text_align(LV_TEXT_ALIGN_RIGHT);
		btn->set_text_scrolling(APP_NAME_WIDTH);
		btn->set_left_symbol("APP");
		btn->set_left_icon_bg(0x1C1917, BConA::ICON_RADIUS);
		const std::string path = entry.path;
		btn->on_click([this, path](BConA *) {
			spdlog::info("[apps] Apps launch requested: {}", path);
			if (on_app_launch) {
				on_app_launch(path);
			}
		});
		_components.push_back(std::move(btn));
	}
}

void AppsPage::destroy()
{
	on_app_launch = nullptr;
	SubPageBase::destroy();
}

void AppsPage::on_subpage_destroy()
{
	on_app_launch = nullptr;
}

} // namespace ui
