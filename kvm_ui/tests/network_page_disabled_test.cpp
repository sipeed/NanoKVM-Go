#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_28);
}

namespace
{

constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
constexpr uint32_t START_PENDING_TIMEOUT_MS = 15000;
constexpr int32_t DISABLED_STAGE_LEFT = 13;
constexpr int32_t DISABLED_STAGE_TOP = 44;
constexpr int32_t DISABLED_STAGE_WIDTH = 230;
constexpr int32_t DISABLED_STAGE_HEIGHT = 152;
constexpr uint32_t DISABLED_SUBLINE_COLOR = 0x8B949E;

ui::TailscaleSnapshotView client_init_snapshot()
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = false;
	return snapshot;
}

ui::TailscaleSnapshotView not_initialized_snapshot()
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = false;
	snapshot.matched = false;
	return snapshot;
}

ui::TailscaleSnapshotView ready_snapshot()
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = true;
	snapshot.ip = "100.64.0.1";
	return snapshot;
}

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *scroll_child(const ui::Page &page, uint32_t index)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, index);
}

lv_obj_t *back_button(const ui::Page &page)
{
	return scroll_child(page, 0);
}

lv_obj_t *preinit_group(const ui::Page &page)
{
	return scroll_child(page, 1);
}

lv_obj_t *on_button(const ui::Page &page)
{
	return scroll_child(page, 2);
}

lv_obj_t *terms_row(const ui::Page &page)
{
	return scroll_child(page, 3);
}

lv_obj_t *terms_body(const ui::Page &page)
{
	return scroll_child(page, 4);
}

lv_obj_t *code_stage(const ui::Page &page)
{
	return scroll_child(page, 5);
}

lv_obj_t *exit_button(const ui::Page &page)
{
	return scroll_child(page, 6);
}

lv_obj_t *disabled_stage(const ui::Page &page)
{
	return scroll_child(page, 7);
}

lv_obj_t *disabled_headline(const ui::Page &page)
{
	lv_obj_t *stage = disabled_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 1) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 0);
}

lv_obj_t *disabled_subline(const ui::Page &page)
{
	lv_obj_t *stage = disabled_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 1);
}

lv_obj_t *row_button(const ui::Page &page, uint32_t index)
{
	lv_obj_t *row = terms_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(row, index);
}

lv_obj_t *agree_button(const ui::Page &page)
{
	return row_button(page, 0);
}

std::string label_text(lv_obj_t *label)
{
	if (label == nullptr) {
		return {};
	}
	const char *text = lv_label_get_text(label);
	return text == nullptr ? std::string() : std::string(text);
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

bool hidden(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_HIDDEN);
}

bool clickable(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_CLICKABLE);
}

class NetworkDisabledFixture {
    public:
	NetworkDisabledFixture()
	{
		_page.create(_lvgl.screen());
		refresh_layout();
	}

	~NetworkDisabledFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkDisabledFixture(const NetworkDisabledFixture &) = delete;
	NetworkDisabledFixture &
	operator=(const NetworkDisabledFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::NetworkPage &page()
	{
		return _page;
	}

	void refresh_layout()
	{
		lv_obj_update_layout(_lvgl.screen());
	}

	void complete_on_hold()
	{
		constexpr uint32_t fill_ms =
			ON_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(on_button(_page));
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		_lvgl.release();
	}

	void enter_terms()
	{
		complete_on_hold();
		refresh_layout();
	}

	void agree()
	{
		lv_obj_t *agree = agree_button(_page);
		REQUIRE(agree != nullptr);
		lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
		refresh_layout();
	}

    private:
	test_support::LvglTestContext _lvgl;
	ui::NetworkPage _page;
};

void arm_start_lane(NetworkDisabledFixture &fixture, uint32_t &start_calls)
{
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	fixture.enter_terms();
	fixture.agree();
}

}

TEST_CASE("Network disabled stage renders the approved notice",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	REQUIRE(start_calls == 1);
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 1);
	fixture.refresh_layout();

	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(hidden(back_button(fixture.page())));
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(preinit_group(fixture.page())));
	CHECK(hidden(on_button(fixture.page())));
	CHECK(hidden(terms_row(fixture.page())));
	CHECK(hidden(terms_body(fixture.page())));
	CHECK(hidden(code_stage(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	lv_obj_t *stage = disabled_stage(fixture.page());
	REQUIRE(stage != nullptr);
	CHECK_FALSE(hidden(stage));

	lv_obj_t *headline = disabled_headline(fixture.page());
	REQUIRE(headline != nullptr);
	CHECK(label_text(headline) == "Not available");
	CHECK(lv_obj_get_width(headline) == DISABLED_STAGE_WIDTH);
	CHECK(lv_label_get_long_mode(headline) == LV_LABEL_LONG_WRAP);
	CHECK(lv_obj_get_style_text_align(headline, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
	CHECK(lv_obj_get_style_text_font(headline, LV_PART_MAIN) ==
	      &MontserratBold_28);
	CHECK(lv_color_eq(lv_obj_get_style_text_color(headline, LV_PART_MAIN),
			  lv_color_white()));

	lv_obj_t *subline = disabled_subline(fixture.page());
	REQUIRE(subline != nullptr);
	CHECK(label_text(subline) ==
	      "Remote Network is not enabled on this device.");
	CHECK(lv_obj_get_width(subline) == DISABLED_STAGE_WIDTH);
	CHECK(lv_label_get_long_mode(subline) == LV_LABEL_LONG_WRAP);
	CHECK(lv_obj_get_style_text_align(subline, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
	CHECK(lv_obj_get_style_text_line_space(subline, LV_PART_MAIN) == 4);
	CHECK(lv_obj_get_style_text_font(subline, LV_PART_MAIN) ==
	      &lv_font_montserrat_16);
	CHECK(lv_color_eq(lv_obj_get_style_text_color(subline, LV_PART_MAIN),
			  lv_color_hex(DISABLED_SUBLINE_COLOR)));

	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
}

TEST_CASE("Network disabled stage keeps the locked column geometry",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 1);
	fixture.refresh_layout();

	lv_obj_t *stage = disabled_stage(fixture.page());
	REQUIRE(stage != nullptr);
	const lv_area_t stage_coordinates = object_coords(stage);
	CHECK(stage_coordinates.x1 == DISABLED_STAGE_LEFT);
	CHECK(stage_coordinates.y1 == DISABLED_STAGE_TOP);
	CHECK(lv_obj_get_width(stage) == DISABLED_STAGE_WIDTH);
	CHECK(lv_obj_get_height(stage) == DISABLED_STAGE_HEIGHT);
	CHECK(stage_coordinates.y2 ==
	      DISABLED_STAGE_TOP + DISABLED_STAGE_HEIGHT - 1);
	CHECK(lv_obj_get_style_bg_opa(stage, LV_PART_MAIN) == LV_OPA_TRANSP);
	CHECK(lv_obj_get_style_border_width(stage, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_flex_flow(stage, LV_PART_MAIN) ==
	      LV_FLEX_FLOW_COLUMN);
	CHECK(lv_obj_get_style_pad_top(stage, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_bottom(stage, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_left(stage, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_right(stage, LV_PART_MAIN) == 0);
	CHECK_FALSE(lv_obj_has_flag(stage, LV_OBJ_FLAG_SCROLLABLE));
	CHECK_FALSE(lv_obj_has_flag(stage, LV_OBJ_FLAG_CLICKABLE));
	CHECK(lv_obj_get_child_count(stage) == 2);
}

TEST_CASE("Network disabled stage ignores snapshots and codes",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);

	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(hidden(disabled_stage(fixture.page())));

	fixture.page().apply_snapshot(client_init_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(hidden(disabled_stage(fixture.page())));

	fixture.page().apply_snapshot(ready_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(hidden(disabled_stage(fixture.page())));
	CHECK(hidden(code_stage(fixture.page())));
	CHECK(fixture.page().rendered_seq() == 4);

	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK(label_text(disabled_headline(fixture.page())) == "Not available");
	CHECK(label_text(disabled_subline(fixture.page())) ==
	      "Remote Network is not enabled on this device.");
}

TEST_CASE("Network result after the disabled watchdog is ignored",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	REQUIRE_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(hidden(disabled_stage(fixture.page())));
}

TEST_CASE("Network disabled stage can be re-entered and reached again",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);

	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(hidden(disabled_stage(fixture.page())));

	fixture.enter_terms();
	fixture.agree();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLED", 2);
	fixture.refresh_layout();
	CHECK(start_calls == 2);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(hidden(disabled_stage(fixture.page())));
}

TEST_CASE("Network non-DISABLED start failure still uses the message hold",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "FAIL", 1);
	fixture.refresh_layout();

	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(disabled_stage(fixture.page())));
	lv_obj_t *label = lv_obj_get_child(code_stage(fixture.page()), 0);
	REQUIRE(label != nullptr);
	CHECK(label_text(label) == "FAIL");
}

TEST_CASE("Network near-miss disabled messages fall back to the hold",
	  "[ui][network][disabled]")
{
	NetworkDisabledFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "DISABLEDX", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(disabled_stage(fixture.page())));
	lv_obj_t *label = lv_obj_get_child(code_stage(fixture.page()), 0);
	REQUIRE(label != nullptr);
	CHECK(label_text(label) == "DISABLEDX");

	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.enter_terms();
	fixture.agree();
	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "disabled", 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(disabled_stage(fixture.page())));
	CHECK(label_text(label) == "disabled");
}
