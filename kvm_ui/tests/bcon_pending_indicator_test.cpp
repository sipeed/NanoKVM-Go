#include <catch2/catch_test_macros.hpp>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/bconb.hpp"

namespace
{

bool has_pending_arc(lv_obj_t *button)
{
	return test_support::find_visible_descendant_of_type(button, &lv_arc_class) !=
	       nullptr;
}

} // namespace

TEST_CASE("BConA and BConB share the pending indicator lifecycle",
	  "[ui][controls][pending]")
{
	test_support::LvglTestContext lvgl;
	ui::BConA large(lvgl.screen(), "Wi-Fi", 0x2B7FFF);
	ui::BConB compact(lvgl.screen(), 0x2B7FFF);
	large.create();
	compact.create();
	large.set_left_symbol(LV_SYMBOL_WIFI);
	compact.set_symbol(LV_SYMBOL_WIFI);

	const std::size_t timer_count_before = lvgl.timer_count();
	large.set_pending_timeout(80);
	compact.set_pending_timeout(80);
	large.set_pending(true);
	compact.set_pending(true);

	REQUIRE(large.is_pending());
	REQUIRE(compact.is_pending());
	REQUIRE(has_pending_arc(large.get_obj()));
	REQUIRE(has_pending_arc(compact.get_obj()));
	CHECK(lvgl.timer_count() == timer_count_before + 4U);

	lvgl.advance(40);
	CHECK(large.is_pending());
	CHECK(compact.is_pending());

	lvgl.advance(40);
	CHECK_FALSE(large.is_pending());
	CHECK_FALSE(compact.is_pending());
	CHECK_FALSE(has_pending_arc(large.get_obj()));
	CHECK_FALSE(has_pending_arc(compact.get_obj()));
	CHECK(lvgl.timer_count() == timer_count_before);

	large.destroy();
	compact.destroy();
}

TEST_CASE("BConB pending teardown cancels its spinner timer",
	  "[ui][controls][pending]")
{
	test_support::LvglTestContext lvgl;
	ui::BConB compact(lvgl.screen(), 0x2B7FFF);
	compact.create();
	compact.set_symbol(LV_SYMBOL_WIFI);
	compact.set_pending_timeout_enabled(false);

	const std::size_t timer_count_before = lvgl.timer_count();
	compact.set_pending(true);
	REQUIRE(compact.is_pending());
	REQUIRE(has_pending_arc(compact.get_obj()));
	REQUIRE(lvgl.timer_count() == timer_count_before + 1U);

	compact.destroy();
	CHECK(lvgl.timer_count() == timer_count_before);
}
