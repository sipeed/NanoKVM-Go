#include <catch2/catch_test_macros.hpp>

#include <cstddef>

#include <lvgl.h>

#include "loading_arc.hpp"
#include "lvgl_test_context.hpp"

TEST_CASE("LoadingArc is created hidden with the shared spinner geometry",
	  "[ui][loading_arc]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t baseline = lvgl.timer_count();

	ui::LoadingArc spinner(lvgl.screen());
	spinner.create();

	REQUIRE(spinner.obj() != nullptr);
	lv_obj_update_layout(lvgl.screen());
	CHECK_FALSE(spinner.is_running());
	CHECK(lv_obj_has_flag(spinner.obj(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(spinner.obj(), LV_OBJ_FLAG_CLICKABLE));
	CHECK_FALSE(lv_obj_has_flag(spinner.obj(), LV_OBJ_FLAG_SCROLLABLE));

	CHECK(lv_obj_get_width(spinner.obj()) == ui::LoadingArc::SIZE);
	CHECK(lv_obj_get_height(spinner.obj()) == ui::LoadingArc::SIZE);
	CHECK(lv_obj_get_style_arc_width(spinner.obj(), LV_PART_INDICATOR) ==
	      ui::LoadingArc::ARC_WIDTH);
	CHECK(lv_obj_get_style_arc_rounded(spinner.obj(), LV_PART_INDICATOR));
	CHECK(lv_obj_get_style_arc_opa(spinner.obj(), LV_PART_MAIN) ==
	      LV_OPA_TRANSP);
	CHECK(lv_color_eq(
		lv_obj_get_style_arc_color(spinner.obj(), LV_PART_INDICATOR),
		lv_color_hex(ui::LoadingArc::COLOR)));
	CHECK(lv_arc_get_angle_start(spinner.obj()) == 0);
	CHECK(lv_arc_get_angle_end(spinner.obj()) == ui::LoadingArc::SWEEP_DEG);
	CHECK(lv_arc_get_bg_angle_start(spinner.obj()) == 0);
	CHECK(lv_arc_get_bg_angle_end(spinner.obj()) == 360);
	CHECK(lv_arc_get_rotation(spinner.obj()) == 0);

	CHECK(lvgl.timer_count() == baseline);

	spinner.create();
	CHECK(lv_obj_get_child_count(lvgl.screen()) == 1);

	spinner.destroy();
	CHECK(spinner.obj() == nullptr);
	CHECK(lvgl.timer_count() == baseline);
}

TEST_CASE("LoadingArc run and stop drive visibility and the spin timer",
	  "[ui][loading_arc]")
{
	test_support::LvglTestContext lvgl;
	ui::LoadingArc spinner(lvgl.screen());
	spinner.create();
	const std::size_t baseline = lvgl.timer_count();

	spinner.set_running(true);
	REQUIRE(spinner.is_running());
	CHECK_FALSE(lv_obj_has_flag(spinner.obj(), LV_OBJ_FLAG_HIDDEN));
	CHECK(lvgl.timer_count() == baseline + 1U);
	CHECK(lv_arc_get_rotation(spinner.obj()) == 0);

	lvgl.advance(ui::LoadingArc::PERIOD_MS);
	CHECK(lv_arc_get_rotation(spinner.obj()) ==
	      ui::LoadingArc::ROTATION_STEP_DEG);
	lvgl.advance(ui::LoadingArc::PERIOD_MS);
	CHECK(lv_arc_get_rotation(spinner.obj()) ==
	      2 * ui::LoadingArc::ROTATION_STEP_DEG);

	spinner.set_running(true);
	CHECK(lvgl.timer_count() == baseline + 1U);

	spinner.set_running(false);
	CHECK_FALSE(spinner.is_running());
	CHECK(lv_obj_has_flag(spinner.obj(), LV_OBJ_FLAG_HIDDEN));
	CHECK(lvgl.timer_count() == baseline);

	spinner.set_running(false);
	CHECK(lvgl.timer_count() == baseline);

	lvgl.advance(3 * ui::LoadingArc::PERIOD_MS);
	CHECK(lv_arc_get_rotation(spinner.obj()) ==
	      2 * ui::LoadingArc::ROTATION_STEP_DEG);

	spinner.set_running(true);
	CHECK(lv_arc_get_rotation(spinner.obj()) == 0);

	spinner.destroy();
	CHECK(lvgl.timer_count() == baseline);
}

TEST_CASE("LoadingArc rotation wraps around after a full revolution",
	  "[ui][loading_arc]")
{
	test_support::LvglTestContext lvgl;
	ui::LoadingArc spinner(lvgl.screen());
	spinner.create();

	spinner.set_running(true);
	constexpr uint32_t ticks_per_revolution =
		360 / ui::LoadingArc::ROTATION_STEP_DEG;
	// lv_timer_handler runs a periodic timer at most once per call, so step
	// one period at a time.
	for (uint32_t i = 0; i < ticks_per_revolution; ++i) {
		lvgl.advance(ui::LoadingArc::PERIOD_MS);
	}
	CHECK(lv_arc_get_rotation(spinner.obj()) == 0);
	CHECK(spinner.is_running());

	spinner.destroy();
}

TEST_CASE("LoadingArc destroy stops the timer and stays null-safe",
	  "[ui][loading_arc]")
{
	test_support::LvglTestContext lvgl;
	ui::LoadingArc spinner(lvgl.screen());
	spinner.create();
	const std::size_t baseline = lvgl.timer_count();

	spinner.set_running(true);
	REQUIRE(spinner.is_running());
	CHECK(lvgl.timer_count() == baseline + 1U);

	spinner.destroy();
	CHECK_FALSE(spinner.is_running());
	CHECK(spinner.obj() == nullptr);
	CHECK(lvgl.timer_count() == baseline);

	spinner.destroy();
	spinner.set_running(true);
	CHECK_FALSE(spinner.is_running());
	CHECK(lvgl.timer_count() == baseline);
}
