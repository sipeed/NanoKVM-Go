#include <catch2/catch_test_macros.hpp>

#include <algorithm>
#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <string_view>
#include <type_traits>
#include <utility>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"

namespace ui
{

struct HoldConfirmBehaviorTestAccess {
	static void fail_closed(HoldConfirmBehavior &behavior)
	{
		behavior.fail_closed();
	}
};

} // namespace ui

namespace
{

constexpr uint32_t FILL_COLOR = 0xFB2C36;
constexpr uint32_t TEST_DURATION_MS = 1000;
constexpr int32_t SCROLL_X = 5;
constexpr int32_t SCROLL_Y = 7;
constexpr int32_t BUTTON_X = 17;
constexpr int32_t BUTTON_Y = 23;

static_assert(!std::is_copy_constructible_v<ui::HoldConfirmBehavior>);
static_assert(!std::is_copy_assignable_v<ui::HoldConfirmBehavior>);
static_assert(!std::is_move_constructible_v<ui::HoldConfirmBehavior>);
static_assert(!std::is_move_assignable_v<ui::HoldConfirmBehavior>);

uint32_t normalized_duration(uint32_t duration_ms)
{
	if (duration_ms == 0) {
		return ui::HoldConfirmBehavior::DEFAULT_DURATION_MS;
	}
	return std::max(duration_ms, ui::HoldConfirmBehavior::MIN_DURATION_MS);
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

class HoldConfirmFixture {
    public:
	explicit HoldConfirmFixture(
		uint32_t duration_ms = TEST_DURATION_MS,
		bool install_confirmation_callback = true,
		ui::BConA::ClickCallback pre_attach_callback = {})
		: _duration_ms(duration_ms)
	{
		_scroll = lv_obj_create(_lvgl.screen());
		if (_scroll == nullptr) {
			return;
		}
		lv_obj_set_pos(_scroll, SCROLL_X, SCROLL_Y);
		lv_obj_set_size(_scroll, 260, 220);
		lv_obj_set_style_border_width(_scroll, 0, LV_PART_MAIN);
		lv_obj_set_style_pad_all(_scroll, 0, LV_PART_MAIN);
		lv_obj_add_flag(_scroll, LV_OBJ_FLAG_SCROLLABLE);

		_button = std::make_unique<ui::BConA>(_scroll, "Before",
						      0xFFFFFF);
		_button->on_click(std::move(pre_attach_callback));
		_button->create();
		_button->set_position(BUTTON_X, BUTTON_Y);

		_behavior = std::make_unique<ui::HoldConfirmBehavior>(
			*_button,
			ui::HoldConfirmConfig { duration_ms, FILL_COLOR,
						"Reset", "Keep" });
		if (install_confirmation_callback) {
			_behavior->on_confirm([this]() {
				++_confirmation_count;
			});
		}
		lv_obj_update_layout(_lvgl.screen());
	}

	~HoldConfirmFixture()
	{
		_behavior.reset();
		if (_button != nullptr) {
			_button->destroy();
			_button.reset();
		}
		if (_scroll != nullptr && lv_obj_is_valid(_scroll)) {
			lv_obj_delete(_scroll);
		}
		_scroll = nullptr;
	}

	HoldConfirmFixture(const HoldConfirmFixture &) = delete;
	HoldConfirmFixture &operator=(const HoldConfirmFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	lv_obj_t *scroll() const
	{
		return _scroll;
	}

	ui::BConA *button() const
	{
		return _button.get();
	}

	lv_obj_t *button_obj() const
	{
		return _button == nullptr ? nullptr : _button->get_obj();
	}

	ui::HoldConfirmBehavior *behavior() const
	{
		return _behavior.get();
	}

	std::unique_ptr<ui::HoldConfirmBehavior> &behavior_owner()
	{
		return _behavior;
	}

	lv_obj_t *clip() const
	{
		lv_obj_t *object = button_obj();
		if (object == nullptr || lv_obj_get_child_count(object) == 0) {
			return nullptr;
		}
		return lv_obj_get_child(object, 0);
	}

	lv_obj_t *surface() const
	{
		lv_obj_t *clip_object = clip();
		if (clip_object == nullptr ||
		    lv_obj_get_child_count(clip_object) == 0) {
			return nullptr;
		}
		return lv_obj_get_child(clip_object, 0);
	}

	uint32_t total_duration_ms() const
	{
		return normalized_duration(_duration_ms);
	}

	uint32_t fill_duration_ms() const
	{
		return total_duration_ms() -
		       ui::HoldConfirmBehavior::FULL_DWELL_MS;
	}

	void press()
	{
		_lvgl.press(button_obj());
	}

	void advance_to_full()
	{
		press();
		_lvgl.advance(fill_duration_ms());
	}

	int confirmation_count() const
	{
		return _confirmation_count;
	}

	int &confirmation_count_ref()
	{
		return _confirmation_count;
	}

    private:
	test_support::LvglTestContext _lvgl;
	lv_obj_t *_scroll = nullptr;
	std::unique_ptr<ui::BConA> _button;
	std::unique_ptr<ui::HoldConfirmBehavior> _behavior;
	uint32_t _duration_ms;
	int _confirmation_count = 0;
};

void check_idle(const HoldConfirmFixture &fixture)
{
	REQUIRE(fixture.behavior() != nullptr);
	CHECK_FALSE(fixture.behavior()->is_holding());
	CHECK(fixture.behavior()->progress_width() == 0);
	CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
	      "Reset");
}

void exercise_duration(uint32_t input_duration_ms, uint32_t expected_total_ms)
{
	HoldConfirmFixture fixture(input_duration_ms);
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());
	REQUIRE(expected_total_ms > ui::HoldConfirmBehavior::FULL_DWELL_MS);

	const uint32_t fill_duration =
		expected_total_ms - ui::HoldConfirmBehavior::FULL_DWELL_MS;
	fixture.press();
	fixture.lvgl().advance(fill_duration - 1);
	CHECK(fixture.confirmation_count() == 0);
	CHECK(fixture.behavior()->is_holding());

	fixture.lvgl().advance(1);
	CHECK(fixture.behavior()->progress_width() == ui::BConA::WIDTH);
	CHECK(fixture.confirmation_count() == 0);

	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS - 1);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().advance(1);
	CHECK(fixture.confirmation_count() == 1);
	check_idle(fixture);
	fixture.lvgl().release();
}

} // namespace

TEST_CASE("BConA owns constructor and updated text", "[ui][hold-confirm]")
{
	test_support::LvglTestContext lvgl;
	REQUIRE(lvgl.display() != nullptr);
	REQUIRE(lvgl.screen() != nullptr);

	std::string initial = "Reset";
	ui::BConA button(lvgl.screen(), initial.c_str(), 0xFFFFFF);
	initial[0] = 'X';
	button.create();
	CHECK(test_support::visible_direct_text(button.get_obj()) == "Reset");

	std::string updated = "Keep";
	button.set_text(updated.c_str());
	updated[0] = 'X';
	button.destroy();
	button.create();
	CHECK(test_support::visible_direct_text(button.get_obj()) == "Keep");

	button.destroy();
}

TEST_CASE("LVGL test context advances a held pointer sample",
	  "[ui][hold-confirm]")
{
	test_support::LvglTestContext lvgl;
	REQUIRE(lvgl.display() != nullptr);
	REQUIRE(lvgl.screen() != nullptr);

	lv_obj_t *object = lv_obj_create(lvgl.screen());
	REQUIRE(object != nullptr);
	lv_obj_set_pos(object, 20, 20);
	lv_obj_set_size(object, 100, 64);
	lv_obj_add_flag(object, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(object, LV_OBJ_FLAG_SCROLLABLE);

	int pressing_count = 0;
	lv_obj_add_event_cb(
		object,
		[](lv_event_t *event) {
			auto *count = static_cast<int *>(
				lv_event_get_user_data(event));
			++(*count);
		},
		LV_EVENT_PRESSING, &pressing_count);

	lvgl.press(object);
	const int count_after_press = pressing_count;
	lvgl.advance(40);
	CHECK(pressing_count > count_after_press);

	lvgl.release();
	lv_obj_delete(object);
}

TEST_CASE("Hold confirmation blocks all click paths after attachment",
	  "[ui][hold-confirm]")
{
	int pre_attach_clicks = 0;
	HoldConfirmFixture fixture(TEST_DURATION_MS, true,
				   [&pre_attach_clicks](ui::BConA *) {
					   ++pre_attach_clicks;
				   });
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());

	SECTION("the pre-attach callback stays cleared after detach")
	{
		fixture.behavior_owner().reset();
		lv_obj_send_event(fixture.button_obj(), LV_EVENT_CLICKED,
				  nullptr);
		CHECK(pre_attach_clicks == 0);
		CHECK(fixture.confirmation_count() == 0);
	}

	SECTION("post-attach component and raw callbacks stay gated")
	{
		int post_attach_clicks = 0;
		fixture.button()->on_click([&post_attach_clicks](ui::BConA *) {
			++post_attach_clicks;
		});
		int raw_short_clicks = 0;
		REQUIRE(lv_obj_add_event_cb(
				fixture.button_obj(),
				[](lv_event_t *event) {
					auto *count = static_cast<int *>(
						lv_event_get_user_data(event));
					++(*count);
				},
				LV_EVENT_SHORT_CLICKED,
				&raw_short_clicks) != nullptr);

		lv_obj_send_event(fixture.button_obj(), LV_EVENT_CLICKED,
				  nullptr);
		lv_obj_send_event(fixture.button_obj(), LV_EVENT_SHORT_CLICKED,
				  nullptr);

		CHECK(pre_attach_clicks == 0);
		CHECK(post_attach_clicks == 0);
		CHECK(raw_short_clicks == 0);
		CHECK(fixture.confirmation_count() == 0);
	}
}

TEST_CASE("Failed hold attachment keeps click gates until behavior destruction",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());
	lv_obj_t *overlay = fixture.clip();
	REQUIRE(overlay != nullptr);

	ui::HoldConfirmBehaviorTestAccess::fail_closed(*fixture.behavior());
	CHECK_FALSE(fixture.behavior()->is_ready());
	check_idle(fixture);
	CHECK_FALSE(
		lv_obj_has_flag(fixture.button_obj(), LV_OBJ_FLAG_CLICKABLE));
	CHECK_FALSE(lv_obj_is_valid(overlay));
	CHECK(fixture.confirmation_count() == 0);

	lv_obj_add_flag(fixture.button_obj(), LV_OBJ_FLAG_CLICKABLE);
	int component_clicks = 0;
	fixture.button()->on_click([&component_clicks](ui::BConA *) {
		++component_clicks;
	});
	int raw_short_clicks = 0;
	int raw_clicks = 0;
	REQUIRE(lv_obj_add_event_cb(
			fixture.button_obj(),
			[](lv_event_t *event) {
				auto *count = static_cast<int *>(
					lv_event_get_user_data(event));
				++(*count);
			},
			LV_EVENT_SHORT_CLICKED, &raw_short_clicks) != nullptr);
	REQUIRE(lv_obj_add_event_cb(
			fixture.button_obj(),
			[](lv_event_t *event) {
				auto *count = static_cast<int *>(
					lv_event_get_user_data(event));
				++(*count);
			},
			LV_EVENT_CLICKED, &raw_clicks) != nullptr);

	lv_obj_send_event(fixture.button_obj(), LV_EVENT_SHORT_CLICKED,
			  nullptr);
	lv_obj_send_event(fixture.button_obj(), LV_EVENT_CLICKED, nullptr);
	CHECK(component_clicks == 0);
	CHECK(raw_short_clicks == 0);
	CHECK(raw_clicks == 0);
	CHECK(fixture.confirmation_count() == 0);

	fixture.behavior_owner().reset();
	lv_obj_send_event(fixture.button_obj(), LV_EVENT_CLICKED, nullptr);
	CHECK(component_clicks == 1);
	CHECK(raw_short_clicks == 0);
	CHECK(raw_clicks == 1);
}

TEST_CASE("Hold confirmation works without a scrollable ancestor",
	  "[ui][hold-confirm]")
{
	test_support::LvglTestContext lvgl;
	REQUIRE(lvgl.display() != nullptr);
	REQUIRE(lvgl.screen() != nullptr);

	lv_obj_t *parent = lv_obj_create(lvgl.screen());
	REQUIRE(parent != nullptr);
	lv_obj_set_pos(parent, SCROLL_X, SCROLL_Y);
	lv_obj_set_size(parent, 260, 220);
	lv_obj_set_style_border_width(parent, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(parent, 0, LV_PART_MAIN);

	ui::BConA button(parent, "Before", 0xFFFFFF);
	button.create();
	button.set_position(BUTTON_X, BUTTON_Y);
	for (lv_obj_t *ancestor = lv_obj_get_parent(button.get_obj());
	     ancestor != nullptr; ancestor = lv_obj_get_parent(ancestor)) {
		lv_obj_clear_flag(ancestor, LV_OBJ_FLAG_SCROLLABLE);
	}
	lv_obj_update_layout(lvgl.screen());

	int confirmation_count = 0;
	{
		ui::HoldConfirmBehavior behavior(
			button,
			ui::HoldConfirmConfig {
				ui::HoldConfirmBehavior::MIN_DURATION_MS,
				FILL_COLOR, "Reset", "Keep" });
		behavior.on_confirm([&confirmation_count]() {
			++confirmation_count;
		});

		REQUIRE(behavior.is_ready());
		CHECK(lv_obj_has_flag(button.get_obj(), LV_OBJ_FLAG_CLICKABLE));
		lvgl.press(button.get_obj());
		lvgl.advance(ui::HoldConfirmBehavior::MIN_DURATION_MS -
			     ui::HoldConfirmBehavior::FULL_DWELL_MS);
		CHECK(behavior.progress_width() == ui::BConA::WIDTH);
		CHECK(confirmation_count == 0);
		lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		CHECK(confirmation_count == 1);
		CHECK_FALSE(behavior.is_holding());
		CHECK(behavior.progress_width() == 0);
		CHECK(test_support::visible_direct_text(button.get_obj()) ==
		      "Reset");
		lvgl.release();
	}

	CHECK(lv_obj_has_flag(button.get_obj(), LV_OBJ_FLAG_CLICKABLE));
	button.destroy();
	lv_obj_delete(parent);
}

TEST_CASE("Hold confirmation overlay is anchored behind BConA content",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());
	lv_obj_t *button = fixture.button_obj();
	lv_obj_t *clip = fixture.clip();
	lv_obj_t *surface = fixture.surface();
	REQUIRE(button != nullptr);
	REQUIRE(clip != nullptr);
	REQUIRE(surface != nullptr);

	CHECK(lv_obj_get_parent(clip) == button);
	CHECK(lv_obj_get_index(clip) == 0);
	CHECK(lv_obj_get_parent(surface) == clip);
	CHECK(lv_obj_get_index(surface) == 0);

	CHECK_FALSE(lv_obj_has_flag(clip, LV_OBJ_FLAG_CLICKABLE));
	CHECK(lv_obj_has_flag(clip, LV_OBJ_FLAG_IGNORE_LAYOUT));
	CHECK(lv_obj_has_flag(clip, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(clip, LV_OBJ_FLAG_OVERFLOW_VISIBLE));
	CHECK(lv_obj_get_style_bg_opa(clip, LV_PART_MAIN) == LV_OPA_TRANSP);
	CHECK(lv_obj_get_style_width(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_height(clip, LV_PART_MAIN) == ui::BConA::HEIGHT);
	CHECK(lv_obj_get_style_x(clip, LV_PART_MAIN) ==
	      -static_cast<int32_t>(ui::BConA::PADDING_LEFT));
	CHECK(lv_obj_get_style_y(clip, LV_PART_MAIN) ==
	      -lv_obj_get_style_pad_top(button, LV_PART_MAIN));
	CHECK(lv_obj_get_style_border_width(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_left(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_right(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_top(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_bottom(clip, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_radius(clip, LV_PART_MAIN) == 0);
	CHECK_FALSE(lv_obj_get_style_clip_corner(clip, LV_PART_MAIN));

	CHECK_FALSE(lv_obj_has_flag(surface, LV_OBJ_FLAG_CLICKABLE));
	CHECK(lv_obj_has_flag(surface, LV_OBJ_FLAG_IGNORE_LAYOUT));
	CHECK(lv_obj_get_style_width(surface, LV_PART_MAIN) ==
	      ui::BConA::WIDTH);
	CHECK(lv_obj_get_style_height(surface, LV_PART_MAIN) ==
	      ui::BConA::HEIGHT);
	CHECK(lv_obj_get_style_x(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_y(surface, LV_PART_MAIN) == 0);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(FILL_COLOR)));
	CHECK(lv_obj_get_style_bg_opa(surface, LV_PART_MAIN) == LV_OPA_COVER);
	CHECK(lv_obj_get_style_border_width(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_left(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_right(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_top(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_bottom(surface, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_radius(surface, LV_PART_MAIN) ==
	      ui::BConA::RADIUS);

	const lv_area_t scroll_before = object_coords(fixture.scroll());
	const lv_area_t button_before = object_coords(button);
	const lv_area_t clip_before = object_coords(clip);
	const lv_area_t surface_before = object_coords(surface);
	CHECK(button_before.x1 == scroll_before.x1 + BUTTON_X);
	CHECK(button_before.y1 == scroll_before.y1 + BUTTON_Y);
	CHECK(clip_before.x1 == button_before.x1);
	CHECK(clip_before.y1 == button_before.y1);
	CHECK(clip_before.y2 == button_before.y2);
	CHECK(surface_before.x1 == button_before.x1);
	CHECK(surface_before.y1 == button_before.y1);
	CHECK(surface_before.y2 == button_before.y2);

	fixture.press();
	CHECK(test_support::visible_direct_text(button) == "Keep");
	CHECK_FALSE(lv_obj_has_flag(clip, LV_OBJ_FLAG_HIDDEN));
	CHECK(fixture.behavior()->progress_width() ==
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(button, LV_PART_MAIN),
			  lv_color_hex(0x36332F)));
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(FILL_COLOR)));

	fixture.lvgl().advance(fixture.fill_duration_ms() / 2);
	const lv_area_t clip_grown = object_coords(clip);
	CHECK(fixture.behavior()->progress_width() > 0);
	CHECK(clip_grown.x1 == clip_before.x1);
	CHECK(clip_grown.x2 > clip_before.x2);
	CHECK(clip_grown.x2 ==
	      clip_grown.x1 + fixture.behavior()->progress_width() - 1);
	fixture.lvgl().release();
}

TEST_CASE("Hold confirmation fill uses ease-out progress", "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());

	fixture.press();
	const int32_t initial_progress = fixture.behavior()->progress_width();
	CHECK(initial_progress >=
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	const uint32_t sample_interval = fixture.fill_duration_ms() / 4;
	fixture.lvgl().advance(sample_interval);
	const int32_t early_progress = fixture.behavior()->progress_width();
	fixture.lvgl().advance(sample_interval);
	const int32_t middle_progress = fixture.behavior()->progress_width();
	fixture.lvgl().advance(sample_interval);
	const int32_t late_progress = fixture.behavior()->progress_width();
	CHECK(initial_progress <= early_progress);
	CHECK(early_progress <= middle_progress);
	CHECK(middle_progress <= late_progress);
	CHECK(early_progress - initial_progress >
	      late_progress - middle_progress);
	CHECK(middle_progress > ui::BConA::WIDTH / 2);
	CHECK(late_progress < ui::BConA::WIDTH);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().release();
}

TEST_CASE("Hold confirmation requires the full fill and dwell",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());

	fixture.press();
	CHECK(fixture.behavior()->progress_width() ==
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(fixture.fill_duration_ms() - 1);
	CHECK(fixture.behavior()->progress_width() < ui::BConA::WIDTH);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().advance(1);
	CHECK(fixture.behavior()->is_holding());
	CHECK(fixture.behavior()->progress_width() == ui::BConA::WIDTH);
	CHECK(fixture.confirmation_count() == 0);

	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS - 1);
	CHECK(fixture.behavior()->progress_width() == ui::BConA::WIDTH);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().advance(1);
	CHECK(fixture.confirmation_count() == 1);
	check_idle(fixture);

	fixture.lvgl().release();
	lv_obj_send_event(fixture.button_obj(), LV_EVENT_CLICKED, nullptr);
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS * 2);
	CHECK(fixture.confirmation_count() == 1);
}

TEST_CASE("Release during the full dwell cancels confirmation",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	fixture.advance_to_full();
	REQUIRE(fixture.behavior()->progress_width() == ui::BConA::WIDTH);

	fixture.lvgl().release();
	check_idle(fixture);
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS * 2);
	CHECK(fixture.confirmation_count() == 0);
}

TEST_CASE("Hold confirmation cancels every interrupted gesture",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	fixture.press();
	fixture.lvgl().advance(200);
	REQUIRE(fixture.behavior()->is_holding());
	REQUIRE(fixture.behavior()->progress_width() > 0);

	SECTION("release")
	{
		fixture.lvgl().release();
	}

	SECTION("press lost")
	{
		lv_obj_send_event(fixture.button_obj(), LV_EVENT_PRESS_LOST,
				  nullptr);
	}

	SECTION("cancel event")
	{
		lv_obj_send_event(fixture.button_obj(), LV_EVENT_CANCEL,
				  nullptr);
	}

	SECTION("scroll begin")
	{
		lv_obj_send_event(fixture.scroll(), LV_EVENT_SCROLL_BEGIN,
				  nullptr);
	}

	SECTION("input reset")
	{
		lv_indev_reset(fixture.lvgl().pointer(), fixture.button_obj());
		fixture.lvgl().release();
	}

	SECTION("pointer exits the press-locked button")
	{
		const lv_area_t coordinates =
			object_coords(fixture.button_obj());
		fixture.lvgl().move_while_pressed(
			coordinates.x2 + 1,
			(coordinates.y1 + coordinates.y2) / 2);
	}

	SECTION("explicit cancel")
	{
		fixture.behavior()->cancel();
	}

	check_idle(fixture);
	fixture.lvgl().advance(fixture.total_duration_ms() * 2);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().release();
}

TEST_CASE("Full-state validation rejects unavailable controls",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture(ui::HoldConfirmBehavior::MIN_DURATION_MS);
	REQUIRE(fixture.behavior() != nullptr);
	fixture.advance_to_full();
	REQUIRE(fixture.behavior()->progress_width() == ui::BConA::WIDTH);

	SECTION("nonclickable")
	{
		lv_obj_clear_flag(fixture.button_obj(), LV_OBJ_FLAG_CLICKABLE);
	}

	SECTION("disabled")
	{
		lv_obj_add_state(fixture.button_obj(), LV_STATE_DISABLED);
	}

	SECTION("pending")
	{
		fixture.button()->set_pending(true);
	}

	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	CHECK(fixture.confirmation_count() == 0);
	check_idle(fixture);
	fixture.button()->set_pending(false);
	fixture.lvgl().release();
}

TEST_CASE("Hold confirmation without a callback fails closed",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture(TEST_DURATION_MS, false);
	REQUIRE(fixture.behavior() != nullptr);
	REQUIRE(fixture.behavior()->is_ready());

	fixture.advance_to_full();
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	CHECK(fixture.confirmation_count() == 0);
	check_idle(fixture);
	fixture.lvgl().release();
}

TEST_CASE("Confirmation callback may synchronously destroy the behavior",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	fixture.behavior()->on_confirm([&fixture]() {
		++fixture.confirmation_count_ref();
		fixture.behavior_owner().reset();
	});

	fixture.advance_to_full();
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	CHECK(fixture.confirmation_count() == 1);
	CHECK(fixture.behavior() == nullptr);
	CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
	      "Reset");

	fixture.lvgl().release();
	fixture.lvgl().advance(TEST_DURATION_MS * 2);
	CHECK(fixture.confirmation_count() == 1);
}

TEST_CASE("Hold confirmation normalizes and honors configured durations",
	  "[ui][hold-confirm]")
{
	SECTION("one millisecond clamps to 500")
	{
		exercise_duration(1, ui::HoldConfirmBehavior::MIN_DURATION_MS);
	}

	SECTION("explicit 500 milliseconds")
	{
		exercise_duration(ui::HoldConfirmBehavior::MIN_DURATION_MS,
				  ui::HoldConfirmBehavior::MIN_DURATION_MS);
	}

	SECTION("750 milliseconds")
	{
		exercise_duration(750, 750);
	}

	SECTION("zero uses the 2000 millisecond default")
	{
		exercise_duration(0,
				  ui::HoldConfirmBehavior::DEFAULT_DURATION_MS);
	}
}

TEST_CASE("Destroying hold behavior stops its timing resources",
	  "[ui][hold-confirm]")
{
	SECTION("during holding removes the width animation")
	{
		HoldConfirmFixture fixture;
		REQUIRE(fixture.behavior() != nullptr);
		lv_obj_t *clip = fixture.clip();
		REQUIRE(clip != nullptr);
		fixture.press();
		fixture.lvgl().advance(200);
		REQUIRE(lv_anim_get(clip, nullptr) != nullptr);

		fixture.behavior_owner().reset();
		CHECK_FALSE(lv_obj_is_valid(clip));
		CHECK(lv_anim_get(clip, nullptr) == nullptr);
		CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
		      "Reset");
		fixture.lvgl().advance(TEST_DURATION_MS * 2);
		CHECK(fixture.confirmation_count() == 0);
		fixture.lvgl().release();
	}

	SECTION("during full removes exactly the dwell timer")
	{
		HoldConfirmFixture fixture;
		REQUIRE(fixture.behavior() != nullptr);
		const std::size_t baseline_timers =
			fixture.lvgl().timer_count();
		fixture.advance_to_full();
		REQUIRE(fixture.behavior()->progress_width() ==
			ui::BConA::WIDTH);
		CHECK(fixture.lvgl().timer_count() == baseline_timers + 1);

		fixture.behavior_owner().reset();
		CHECK(fixture.lvgl().timer_count() == baseline_timers);
		CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
		      "Reset");
		fixture.lvgl().advance(TEST_DURATION_MS * 2);
		CHECK(fixture.confirmation_count() == 0);
		fixture.lvgl().release();
	}
}

TEST_CASE("Deleting hold overlay objects fails closed", "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	fixture.press();
	fixture.lvgl().advance(200);
	REQUIRE(fixture.behavior()->is_holding());

	SECTION("delete the clip")
	{
		lv_obj_t *clip = fixture.clip();
		REQUIRE(clip != nullptr);
		lv_obj_delete(clip);
	}

	SECTION("delete the fill surface")
	{
		lv_obj_t *clip = fixture.clip();
		lv_obj_t *surface = fixture.surface();
		REQUIRE(clip != nullptr);
		REQUIRE(surface != nullptr);
		lv_obj_delete(surface);
		CHECK(lv_obj_has_flag(clip, LV_OBJ_FLAG_HIDDEN));
		CHECK(lv_obj_get_style_width(clip, LV_PART_MAIN) == 0);
	}

	CHECK_FALSE(fixture.behavior()->is_ready());
	check_idle(fixture);
	fixture.lvgl().advance(TEST_DURATION_MS * 2);
	CHECK(fixture.confirmation_count() == 0);
	fixture.lvgl().release();
}

TEST_CASE("Deleting the target BConA during a hold is safe",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	fixture.press();
	fixture.lvgl().advance(200);
	REQUIRE(fixture.behavior()->is_holding());

	fixture.button()->destroy();
	CHECK(fixture.button_obj() == nullptr);
	CHECK_FALSE(fixture.behavior()->is_ready());
	CHECK_FALSE(fixture.behavior()->is_holding());
	CHECK(fixture.behavior()->progress_width() == 0);

	fixture.behavior_owner().reset();
	fixture.lvgl().advance(TEST_DURATION_MS * 2);
	CHECK(fixture.confirmation_count() == 0);
}

TEST_CASE("Deleting the active input device cancels confirmation safely",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);

	SECTION("during Holding")
	{
		fixture.press();
		fixture.lvgl().advance(200);
		REQUIRE(fixture.behavior()->is_holding());
		REQUIRE(fixture.behavior()->progress_width() > 0);
	}

	SECTION("during Full dwell")
	{
		fixture.advance_to_full();
		REQUIRE(fixture.behavior()->is_holding());
		REQUIRE(fixture.behavior()->progress_width() ==
			ui::BConA::WIDTH);
	}

	fixture.lvgl().delete_pointer();
	REQUIRE(fixture.lvgl().pointer() == nullptr);
	REQUIRE_FALSE(fixture.behavior()->is_holding());
	CHECK(fixture.behavior()->progress_width() == 0);
	CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
	      "Reset");
	fixture.lvgl().advance(TEST_DURATION_MS * 2);
	CHECK(fixture.confirmation_count() == 0);
}

TEST_CASE("Earlier indev delete callback may destroy hold behavior safely",
	  "[ui][hold-confirm]")
{
	HoldConfirmFixture fixture;
	REQUIRE(fixture.behavior() != nullptr);
	lv_indev_t *pointer = fixture.lvgl().pointer();
	REQUIRE(pointer != nullptr);
	REQUIRE(lv_indev_get_event_count(pointer) == 0);

	struct DeleteContext {
		std::unique_ptr<ui::HoldConfirmBehavior> *behavior_owner;
		int early_count = 0;
		int trailing_count = 0;
	};
	DeleteContext context { &fixture.behavior_owner() };
	lv_indev_add_event_cb(
		pointer,
		[](lv_event_t *event) {
			auto *delete_context = static_cast<DeleteContext *>(
				lv_event_get_user_data(event));
			++delete_context->early_count;
			delete_context->behavior_owner->reset();
		},
		LV_EVENT_DELETE, &context);
	REQUIRE(lv_indev_get_event_count(pointer) == 1);

	fixture.press();
	REQUIRE(fixture.behavior()->is_holding());
	REQUIRE(lv_indev_get_event_count(pointer) == 2);
	lv_indev_add_event_cb(
		pointer,
		[](lv_event_t *event) {
			auto *delete_context = static_cast<DeleteContext *>(
				lv_event_get_user_data(event));
			++delete_context->trailing_count;
		},
		LV_EVENT_DELETE, &context);
	REQUIRE(lv_indev_get_event_count(pointer) == 3);
	REQUIRE(lv_obj_get_child_count(fixture.button_obj()) == 5);

	fixture.lvgl().delete_pointer();
	CHECK(context.early_count == 1);
	CHECK(context.trailing_count == 1);
	CHECK(fixture.behavior() == nullptr);
	CHECK(test_support::visible_direct_text(fixture.button_obj()) ==
	      "Reset");
	CHECK(lv_obj_get_child_count(fixture.button_obj()) == 4);
	fixture.lvgl().advance(TEST_DURATION_MS * 2);
	CHECK(fixture.confirmation_count() == 0);
}
