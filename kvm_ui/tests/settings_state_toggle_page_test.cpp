#include <catch2/catch_test_macros.hpp>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/settings_state_toggle_page.hpp"

TEST_CASE("Settings state toggle page renders pending through BConA",
	  "[ui][settings][pending]")
{
	test_support::LvglTestContext lvgl;
	ui::SettingsStateTogglePage page("MCP", 0x22C55E, nullptr);
	page.create(lvgl.screen());

	ui::ControlState *const control = page.toggle_control();
	REQUIRE(control != nullptr);
	CHECK_FALSE(control->is_pending());

	page.set_enabled(false);
	page.set_pending_state(true, true, 60);
	CHECK(control->is_pending());
	CHECK(control->is_active());
	CHECK(test_support::find_visible_descendant_of_type(
		      page.get_container(), &lv_arc_class) != nullptr);

	page.set_pending_state(false, false, 60);
	CHECK_FALSE(control->is_pending());
	CHECK_FALSE(control->is_active());
	CHECK(test_support::find_visible_descendant_of_type(
		      page.get_container(), &lv_arc_class) == nullptr);

	page.destroy();
}
