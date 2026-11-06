#include <catch2/catch_test_macros.hpp>

#include <string_view>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/subpage_base.hpp"

namespace
{

class BackNavigationTestPage final : public ui::SubPageBase {
    public:
	BackNavigationTestPage()
		: SubPageBase(ui::PageType::SUBPAGE_WIFI, "Back")
	{
	}

	void set_back_available(bool available)
	{
		set_back_navigation_available(available);
	}

	ui::BConA *back_button() const
	{
		return title_button();
	}

    private:
	void build_content() override
	{
	}
};

lv_obj_t *find_descendant_of_type(lv_obj_t *parent, const lv_obj_class_t *type)
{
	if (parent == nullptr || type == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, type)) {
			return child;
		}
		if (lv_obj_t *match = find_descendant_of_type(child, type);
		    match != nullptr) {
			return match;
		}
	}

	return nullptr;
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

} // namespace

TEST_CASE("SubPageBase Back availability preserves the title row layout",
	  "[ui][subpage][back-navigation]")
{
	test_support::LvglTestContext lvgl;
	BackNavigationTestPage page;
	int back_clicks = 0;
	page.on_back_click = [&back_clicks]() {
		++back_clicks;
	};
	page.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());

	ui::BConA *back_component = page.back_button();
	REQUIRE(back_component != nullptr);
	lv_obj_t *back = back_component->get_obj();
	REQUIRE(back != nullptr);
	lv_obj_t *image = find_descendant_of_type(back, &lv_image_class);
	REQUIRE(image != nullptr);

	const lv_area_t original_coordinates = object_coords(back);
	const int32_t original_width = lv_obj_get_width(back);
	const int32_t original_height = lv_obj_get_height(back);
	CHECK(test_support::visible_direct_text(back) == "Back");
	CHECK(back_component->is_left_image_visible());
	CHECK_FALSE(lv_obj_has_flag(image, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(back, LV_OBJ_FLAG_CLICKABLE));

	page.set_back_available(false);
	page.set_back_available(false);
	lv_obj_update_layout(lvgl.screen());

	CHECK_FALSE(back_component->is_left_image_visible());
	CHECK(lv_obj_has_flag(image, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(back, LV_OBJ_FLAG_CLICKABLE));
	CHECK(lv_obj_get_width(back) == original_width);
	CHECK(lv_obj_get_height(back) == original_height);
	CHECK(object_coords(back).x1 == original_coordinates.x1);
	CHECK(object_coords(back).y1 == original_coordinates.y1);
	CHECK(test_support::visible_direct_text(back) == "Back");

	lvgl.press(back);
	lvgl.release();
	CHECK(back_clicks == 0);

	back_component->set_pending(true);
	back_component->set_pending(false);
	CHECK(lv_obj_has_flag(image, LV_OBJ_FLAG_HIDDEN));

	page.set_back_available(true);
	page.set_back_available(true);
	lv_obj_update_layout(lvgl.screen());

	CHECK(back_component->is_left_image_visible());
	CHECK_FALSE(lv_obj_has_flag(image, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(back, LV_OBJ_FLAG_CLICKABLE));
	CHECK(lv_obj_get_width(back) == original_width);
	CHECK(lv_obj_get_height(back) == original_height);
	CHECK(object_coords(back).x1 == original_coordinates.x1);
	CHECK(object_coords(back).y1 == original_coordinates.y1);

	lvgl.press(back);
	lvgl.release();
	CHECK(back_clicks == 1);

	page.set_back_available(false);
	page.destroy();
	CHECK_FALSE(page.is_created());
}
