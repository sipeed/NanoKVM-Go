#include <string_view>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/apps_page.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_18);
}

namespace
{

bool contains_label(lv_obj_t *object, std::string_view expected)
{
	if (object == nullptr) {
		return false;
	}
	if (lv_obj_check_type(object, &lv_label_class) &&
	    std::string_view(lv_label_get_text(object)) == expected) {
		return true;
	}
	const uint32_t count = lv_obj_get_child_count(object);
	for (uint32_t index = 0; index < count; ++index) {
		if (contains_label(lv_obj_get_child(object, index), expected)) {
			return true;
		}
	}
	return false;
}

lv_obj_t *find_label(lv_obj_t *object, std::string_view expected)
{
	if (object == nullptr) {
		return nullptr;
	}
	if (lv_obj_check_type(object, &lv_label_class) &&
	    std::string_view(lv_label_get_text(object)) == expected) {
		return object;
	}
	const uint32_t count = lv_obj_get_child_count(object);
	for (uint32_t index = 0; index < count; ++index) {
		if (lv_obj_t *label =
			    find_label(lv_obj_get_child(object, index), expected);
		    label != nullptr) {
			return label;
		}
	}
	return nullptr;
}

} // namespace

TEST_CASE("Apps page refreshes entries after creation", "[apps-page]")
{
	test_support::LvglTestContext context;
	ui::AppsPage page;
	page.set_entries({ { "Old App", "/apps/old/main.py" } });
	page.create(context.screen());
	CHECK(contains_label(page.get_container(), "Old App"));

	page.set_entries({ { "First App", "/apps/first/main.py" },
			   { "Second App", "/apps/second/main.py" } });
	CHECK_FALSE(contains_label(page.get_container(), "Old App"));
	CHECK(contains_label(page.get_container(), "First App"));
	CHECK(contains_label(page.get_container(), "Second App"));

	page.destroy();
}

TEST_CASE("Apps page shows a persistent scrollbar for overflowing entries",
	  "[apps-page]")
{
	test_support::LvglTestContext context;
	ui::AppsPage page;
	page.set_entries({ { "First App", "/apps/first/main.py" },
			   { "Second App", "/apps/second/main.py" },
			   { "Third App", "/apps/third/main.py" },
			   { "Fourth App", "/apps/fourth/main.py" } });
	page.create(context.screen());
	lv_obj_update_layout(context.screen());

	lv_obj_t *scroll_content = lv_obj_get_child(page.get_container(), 0);
	REQUIRE(scroll_content != nullptr);
	CHECK(lv_obj_get_scrollbar_mode(scroll_content) ==
	      LV_SCROLLBAR_MODE_AUTO);
	CHECK(lv_obj_get_style_width(scroll_content, LV_PART_SCROLLBAR) == 5);
	CHECK(lv_color_eq(
		lv_obj_get_style_bg_color(scroll_content, LV_PART_SCROLLBAR),
		lv_color_hex(0xA8A29E)));
	CHECK(lv_obj_get_style_bg_opa(scroll_content, LV_PART_SCROLLBAR) ==
	      LV_OPA_COVER);
	CHECK(lv_obj_get_style_radius(scroll_content, LV_PART_SCROLLBAR) ==
	      LV_RADIUS_CIRCLE);
	CHECK(lv_obj_get_style_pad_right(scroll_content, LV_PART_SCROLLBAR) == 8);
	CHECK(lv_obj_get_style_pad_top(scroll_content, LV_PART_SCROLLBAR) == 16);
	CHECK(lv_obj_get_style_pad_bottom(scroll_content, LV_PART_SCROLLBAR) ==
	      16);

	lv_area_t horizontal_area;
	lv_area_t vertical_area;
	lv_obj_get_scrollbar_area(scroll_content, &horizontal_area,
				  &vertical_area);
	CHECK(vertical_area.x2 >= vertical_area.x1);
	CHECK(vertical_area.y2 >= vertical_area.y1);
	CHECK(horizontal_area.x2 < horizontal_area.x1);
	const int32_t initial_scrollbar_y = vertical_area.y1;

	lv_obj_scroll_to_y(scroll_content,
			   lv_obj_get_scroll_bottom(scroll_content), LV_ANIM_OFF);
	lv_obj_update_layout(context.screen());
	lv_obj_get_scrollbar_area(scroll_content, &horizontal_area,
				  &vertical_area);
	CHECK(lv_obj_get_scroll_bottom(scroll_content) == 0);
	CHECK(vertical_area.y1 > initial_scrollbar_y);
	lv_area_t scroll_content_area;
	lv_obj_get_coords(scroll_content, &scroll_content_area);
	CHECK(vertical_area.y2 == scroll_content_area.y2 - 16);

	page.destroy();
}

TEST_CASE("Apps page keeps long names clear of the left icon and scrolls them",
	  "[apps-page]")
{
	test_support::LvglTestContext context;
	ui::AppsPage page;
	constexpr std::string_view long_name =
		"NanoKVM Application With A Very Long Display Name";
	page.set_entries({ { std::string(long_name), "/apps/long/main.py" } });
	page.create(context.screen());
	lv_obj_update_layout(context.screen());

	lv_obj_t *name_label = find_label(page.get_container(), long_name);
	REQUIRE(name_label != nullptr);
	CHECK(lv_obj_get_style_text_font(name_label, LV_PART_MAIN) ==
	      &MontserratBold_18);
	CHECK(lv_obj_get_style_text_align(name_label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_RIGHT);
	CHECK(lv_label_get_long_mode(name_label) == LV_LABEL_LONG_SCROLL);
	CHECK(lv_anim_count_running() > 0);
	CHECK(lv_obj_get_width(name_label) ==
	      ui::BConA::WIDTH - 2 * ui::BConA::PADDING_LEFT -
		      ui::BConA::ICON_SIZE - ui::BConA::PADDING_COLUMN);

	lv_obj_t *button = lv_obj_get_parent(name_label);
	REQUIRE(button != nullptr);
	lv_obj_t *icon = lv_obj_get_child(button, 0);
	REQUIRE(icon != nullptr);
	CHECK(lv_obj_get_x(name_label) >=
	      lv_obj_get_x(icon) + lv_obj_get_width(icon) +
		      ui::BConA::PADDING_COLUMN);

	page.destroy();
}

TEST_CASE("Apps page shows Voice Bridge without scrolling", "[apps-page]")
{
	test_support::LvglTestContext context;
	ui::AppsPage page;
	constexpr std::string_view app_name = "Voice Bridge";
	page.set_entries(
		{ { std::string(app_name), "/apps/voice-bridge/main.py" } });
	page.create(context.screen());
	lv_obj_update_layout(context.screen());

	lv_obj_t *name_label = find_label(page.get_container(), app_name);
	REQUIRE(name_label != nullptr);
	CHECK(lv_obj_get_style_text_font(name_label, LV_PART_MAIN) ==
	      &MontserratBold_18);
	CHECK(lv_text_get_width(app_name.data(), app_name.size(),
				&MontserratBold_18, 0) <= lv_obj_get_width(name_label));
	CHECK(lv_anim_count_running() == 0);

	page.destroy();
}
