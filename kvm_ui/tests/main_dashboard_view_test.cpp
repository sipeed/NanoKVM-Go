#include <string_view>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/main_dashboard_view.hpp"
#include "ui/root_layout.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_main_mouse_active);
LV_IMAGE_DECLARE(img_main_mouse_default);
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

	const uint32_t child_count = lv_obj_get_child_count(object);
	for (uint32_t index = 0; index < child_count; ++index) {
		if (contains_label(lv_obj_get_child(object, index), expected)) {
			return true;
		}
	}
	return false;
}

bool contains_image_source(lv_obj_t *object, const lv_image_dsc_t *source)
{
	if (object == nullptr || source == nullptr) {
		return false;
	}
	if (lv_obj_check_type(object, &lv_image_class) &&
	    lv_image_get_src(object) == source) {
		return true;
	}

	const uint32_t child_count = lv_obj_get_child_count(object);
	for (uint32_t index = 0; index < child_count; ++index) {
		if (contains_image_source(lv_obj_get_child(object, index),
					  source)) {
			return true;
		}
	}
	return false;
}

} // namespace

TEST_CASE("Main dashboard keeps presentation state and timer teardown local",
	  "[ui][main-dashboard][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();

	ui::MainDashboardView view;
	view.create(lvgl.screen(), nullptr);
	lv_obj_update_layout(lvgl.screen());

	REQUIRE(view.is_created());
	REQUIRE(view.content_object() != nullptr);
	CHECK(lv_obj_get_width(view.content_object()) ==
	      ui::root_layout::kContentCardWidth);
	CHECK(lv_obj_get_height(view.content_object()) ==
	      ui::root_layout::kContentCardHeight);
	CHECK(lvgl.timer_count() == timer_count_before + 1U);

	view.set_resolution_text("1920x1080");
	view.set_fps_text("60");
	view.set_keyboard_special_text("Ctrl");
	lvgl.advance(333);
	CHECK(contains_label(view.content_object(), "1920x1080"));
	CHECK(contains_label(view.content_object(), "60"));
	CHECK(contains_label(view.content_object(), "Ctrl"));

	view.set_mouse_active(true);
	CHECK(contains_image_source(view.content_object(),
				    &img_main_mouse_active));
	lvgl.advance(800);
	CHECK(contains_image_source(view.content_object(),
				    &img_main_mouse_default));

	view.set_ip_empty_placeholder("Reconnecting");
	CHECK(lvgl.timer_count() == timer_count_before + 2U);
	view.begin_shutdown();
	CHECK(view.is_created());
	CHECK(lvgl.timer_count() == timer_count_before);
	view.shutdown();
	CHECK_FALSE(view.is_created());
	CHECK(lvgl.timer_count() == timer_count_before);

	lvgl.advance(1000);
	CHECK(lvgl.timer_count() == timer_count_before);
}
