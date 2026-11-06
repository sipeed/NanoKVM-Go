#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string>
#include <utility>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/about_page.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/tailscale_page.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_36);
}

namespace
{

constexpr uint32_t HOLD_DURATION_MS = 2000;

ui::TailscaleSnapshotView ready_snapshot(bool online, std::string ip)
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = true;
	snapshot.online = online;
	snapshot.ip = std::move(ip);
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

ui::TailscaleSnapshotView client_init_snapshot()
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = false;
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

lv_obj_t *back_button(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 3) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 0);
}

lv_obj_t *stage(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 3) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 1);
}

lv_obj_t *action_button(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 3) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 2);
}

lv_obj_t *code_label(const ui::Page &page)
{
	lv_obj_t *stage_obj = stage(page);
	if (stage_obj == nullptr || lv_obj_get_child_count(stage_obj) != 2) {
		return nullptr;
	}
	return lv_obj_get_child(stage_obj, 0);
}

lv_obj_t *ip_row(const ui::Page &page)
{
	lv_obj_t *stage_obj = stage(page);
	if (stage_obj == nullptr || lv_obj_get_child_count(stage_obj) != 2) {
		return nullptr;
	}
	return lv_obj_get_child(stage_obj, 1);
}

lv_obj_t *find_descendant_of_type(lv_obj_t *parent, const lv_obj_class_t *type)
{
	if (parent == nullptr || type == nullptr) {
		return nullptr;
	}
	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (child == nullptr) {
			continue;
		}
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

lv_obj_t *row_value_label(lv_obj_t *row)
{
	if (row == nullptr) {
		return nullptr;
	}
	bool past_first = false;
	for (uint32_t i = 0; i < lv_obj_get_child_count(row); ++i) {
		lv_obj_t *child = lv_obj_get_child(row, i);
		if (child == nullptr ||
		    !lv_obj_check_type(child, &lv_label_class)) {
			continue;
		}
		if (past_first) {
			return child;
		}
		past_first = true;
	}
	return nullptr;
}

std::string row_value_text(lv_obj_t *row)
{
	lv_obj_t *label = row_value_label(row);
	if (label == nullptr) {
		return {};
	}
	const char *text = lv_label_get_text(label);
	return text == nullptr ? std::string() : std::string(text);
}

bool hidden(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_HIDDEN);
}

bool clickable(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_CLICKABLE);
}

bool has_pending_arc(lv_obj_t *obj)
{
	return test_support::find_visible_descendant_of_type(
		       obj, &lv_arc_class) != nullptr;
}

lv_obj_t *hold_fill_surface(lv_obj_t *button)
{
	if (button == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(button);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(button, index);
		if (child == nullptr) {
			continue;
		}
		if (lv_obj_has_flag(child, LV_OBJ_FLAG_IGNORE_LAYOUT) &&
		    lv_obj_get_style_height(child, LV_PART_MAIN) ==
			    ui::BConA::HEIGHT &&
		    lv_obj_get_style_bg_opa(child, LV_PART_MAIN) ==
			    LV_OPA_TRANSP &&
		    lv_obj_get_child_count(child) > 0) {
			return lv_obj_get_child(child, 0);
		}
	}

	return nullptr;
}

std::string button_text(lv_obj_t *obj)
{
	return std::string(test_support::visible_direct_text(obj));
}

lv_obj_t *back_image(lv_obj_t *back)
{
	return find_descendant_of_type(back, &lv_image_class);
}

std::string code_text(const ui::Page &page)
{
	lv_obj_t *label = code_label(page);
	return label == nullptr ? std::string() :
				  std::string(lv_label_get_text(label));
}

struct TailscalePageFixture {
	TailscalePageFixture()
	{
		page.on_start_request = [this]() -> bool {
			++start_count;
			return start_accepted;
		};
		page.on_exit_request = [this]() -> bool {
			++exit_count;
			return exit_accepted;
		};
		page.on_session_visibility = [this](bool visible) {
			if (visible) {
				++visibility_true_count;
			} else {
				++visibility_false_count;
			}
		};
		page.create(lvgl.screen());
		lv_obj_update_layout(lvgl.screen());
	}

	~TailscalePageFixture()
	{
		if (page.is_created()) {
			page.destroy();
		}
	}

	test_support::LvglTestContext lvgl;
	ui::TailscalePage page;
	int start_count = 0;
	int exit_count = 0;
	int visibility_true_count = 0;
	int visibility_false_count = 0;
	bool start_accepted = true;
	bool exit_accepted = true;

	void complete_hold()
	{
		constexpr uint32_t fill_ms =
			HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		lvgl.press(action_button(page));
		lvgl.advance(fill_ms);
		lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	}
};

}

TEST_CASE("Tailscale page anchors the gate button to the About Reset slot",
	  "[ui][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::AboutPage about_page;
	about_page.create(lvgl.screen());
	ui::TailscalePage tailscale_page;
	tailscale_page.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *about_scroll = lv_obj_get_child(about_page.get_container(), 0);
	REQUIRE(about_scroll != nullptr);
	REQUIRE(lv_obj_get_child_count(about_scroll) == 3);
	lv_obj_t *about_reset = lv_obj_get_child(about_scroll, 2);
	REQUIRE(about_reset != nullptr);

	REQUIRE(scroll_content(tailscale_page) != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll_content(tailscale_page)) == 3);
	REQUIRE(action_button(tailscale_page) != nullptr);
	REQUIRE(stage(tailscale_page) != nullptr);

	lv_area_t reset_area {};
	lv_area_t action_area {};
	lv_area_t stage_area {};
	lv_obj_get_coords(about_reset, &reset_area);
	lv_obj_get_coords(action_button(tailscale_page), &action_area);
	lv_obj_get_coords(stage(tailscale_page), &stage_area);

	CHECK(action_area.x1 == reset_area.x1);
	CHECK(action_area.y1 == reset_area.y1);
	CHECK(action_area.y2 == reset_area.y2);
	CHECK(action_area.y1 == 164);
	CHECK(lv_obj_get_width(action_button(tailscale_page)) ==
	      ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(action_button(tailscale_page)) ==
	      ui::BConA::HEIGHT);
	CHECK(stage_area.y1 == 44);
	CHECK(stage_area.y2 == 195);

	lv_obj_t *scroll = scroll_content(tailscale_page);
	CHECK(lv_obj_get_scroll_top(scroll) + lv_obj_get_scroll_bottom(scroll) <=
	      0);

	CHECK(button_text(action_button(tailscale_page)) == "Start");
	CHECK_FALSE(hidden(action_button(tailscale_page)));
	CHECK_FALSE(hidden(code_label(tailscale_page)));
	CHECK(std::string(lv_label_get_text(code_label(tailscale_page))) ==
	      "--");
	CHECK(hidden(ip_row(tailscale_page)));
	CHECK(lv_obj_get_style_text_font(code_label(tailscale_page),
					 LV_PART_MAIN) ==
	      &MontserratBold_36);

	tailscale_page.destroy();
	about_page.destroy();
}

TEST_CASE("Tailscale page centers the stage content on the full screen",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	lv_obj_t *scroll = scroll_content(fixture.page);
	REQUIRE(scroll != nullptr);

	fixture.page.apply_snapshot(client_init_snapshot(), 1);
	lv_obj_update_layout(fixture.lvgl.screen());
	REQUIRE_FALSE(hidden(code_label(fixture.page)));

	lv_area_t code_area {};
	lv_obj_get_coords(code_label(fixture.page), &code_area);
	CHECK((code_area.y1 + code_area.y2) / 2 == 119);
	CHECK(lv_obj_get_scroll_top(scroll) + lv_obj_get_scroll_bottom(scroll) <=
	      0);

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 2);
	lv_obj_update_layout(fixture.lvgl.screen());
	REQUIRE_FALSE(hidden(ip_row(fixture.page)));

	lv_area_t ip_area {};
	lv_obj_get_coords(ip_row(fixture.page), &ip_area);
	CHECK(ip_area.y1 == 104);
	CHECK(ip_area.y2 == 135);
	CHECK(lv_obj_get_scroll_top(scroll) + lv_obj_get_scroll_bottom(scroll) <=
	      0);
}

TEST_CASE("Tailscale page walks the init, pairing, and ready stages",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	REQUIRE(button_text(action_button(fixture.page)) == "Start");

	fixture.page.apply_snapshot(not_initialized_snapshot(), 1);
	CHECK_FALSE(hidden(action_button(fixture.page)));
	CHECK(button_text(action_button(fixture.page)) == "Start");
	CHECK(clickable(action_button(fixture.page)));
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(client_init_snapshot(), 2);
	CHECK(hidden(action_button(fixture.page)));
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_code("123456", 1);
	CHECK(code_text(fixture.page) == "123 456");

	fixture.page.apply_snapshot(ready_snapshot(false, ""), 3);
	CHECK_FALSE(hidden(action_button(fixture.page)));
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK(clickable(action_button(fixture.page)));
	CHECK(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 4);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK_FALSE(hidden(ip_row(fixture.page)));
	CHECK(row_value_text(ip_row(fixture.page)) == "100.64.0.1");
	CHECK(hidden(code_label(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(false, "100.64.0.1"), 5);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK_FALSE(hidden(ip_row(fixture.page)));
	CHECK(row_value_text(ip_row(fixture.page)) == "100.64.0.1");
}

TEST_CASE("Tailscale page shows the ready IP row without the online flag",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(ready_snapshot(false, "100.64.0.1"), 1);
	lv_obj_update_layout(fixture.lvgl.screen());
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	REQUIRE_FALSE(hidden(ip_row(fixture.page)));
	CHECK(row_value_text(ip_row(fixture.page)) == "100.64.0.1");

	lv_area_t ip_area {};
	lv_obj_get_coords(ip_row(fixture.page), &ip_area);
	CHECK(ip_area.y1 == 104);
	CHECK(ip_area.y2 == 135);
}

TEST_CASE("Tailscale page hides the ready IP row while the failure message shows",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(false, "100.64.0.1"), 1);
	fixture.page.apply_action_result(ui::TailscaleActionKind::Exit, false,
					 "FAIL", 1);
	REQUIRE(code_text(fixture.page) == "FAIL");
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));
	CHECK(button_text(action_button(fixture.page)) == "Exit");

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page ignores stale gate and code observations",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 5);
	REQUIRE(fixture.page.rendered_seq() == 5);
	REQUIRE(button_text(action_button(fixture.page)) == "Exit");

	fixture.page.apply_snapshot(ready_snapshot(false, ""), 5);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK_FALSE(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(false, ""), 4);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK_FALSE(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(true, "10.0.0.1"), 6);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK(row_value_text(ip_row(fixture.page)) == "10.0.0.1");
}

TEST_CASE("Tailscale page writes codes only during client init",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(not_initialized_snapshot(), 1);
	fixture.page.apply_code("999999", 1);
	CHECK(code_text(fixture.page) == "--");

	fixture.page.apply_snapshot(client_init_snapshot(), 2);
	fixture.page.apply_code("111111", 2);
	CHECK(code_text(fixture.page) == "111 111");

	fixture.page.apply_code("222222", 2);
	CHECK(code_text(fixture.page) == "111 111");

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 3);
	CHECK(hidden(code_label(fixture.page)));
}

TEST_CASE("Tailscale page requires a long press to start initialization",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	lv_obj_t *action = action_button(fixture.page);

	lv_obj_send_event(action, LV_EVENT_SHORT_CLICKED, nullptr);
	lv_obj_send_event(action, LV_EVENT_CLICKED, nullptr);
	CHECK(fixture.start_count == 0);

	fixture.lvgl.press(action);
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	CHECK(button_text(action) == "Keep");
	CHECK(fixture.start_count == 0);
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	fixture.lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);

	CHECK(fixture.start_count == 1);
	CHECK(has_pending_arc(action));
	CHECK_FALSE(clickable(action));
	CHECK(button_text(action) == "Start");

	fixture.lvgl.release();
	fixture.page.apply_action_result(ui::TailscaleActionKind::Start, true, "",
					 1);
	CHECK_FALSE(has_pending_arc(action));
	CHECK(clickable(action));
}

TEST_CASE("Tailscale page requires a long press to exit",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	lv_obj_t *action = action_button(fixture.page);
	REQUIRE(button_text(action) == "Exit");

	lv_obj_send_event(action, LV_EVENT_SHORT_CLICKED, nullptr);
	lv_obj_send_event(action, LV_EVENT_CLICKED, nullptr);
	CHECK(fixture.exit_count == 0);

	fixture.lvgl.press(action);
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	CHECK(button_text(action) == "Keep");
	CHECK(fixture.exit_count == 0);
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	fixture.lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);

	CHECK(fixture.exit_count == 1);
	CHECK(fixture.start_count == 0);
	CHECK(has_pending_arc(action));
	CHECK_FALSE(clickable(action));
	CHECK(button_text(action) == "Exit");
}

TEST_CASE("Tailscale page Start hold fill uses the page accent",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	lv_obj_t *action = action_button(fixture.page);
	REQUIRE(button_text(action) == "Start");

	fixture.lvgl.press(action);
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	CHECK(button_text(action) == "Keep");

	lv_obj_t *surface = hold_fill_surface(action);
	REQUIRE(surface != nullptr);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(ui::TailscalePage::ACCENT_COLOR)));

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page locks navigation while initialization pends",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	lv_obj_t *back = back_button(fixture.page);
	lv_obj_t *image = back_image(back);
	REQUIRE(image != nullptr);
	REQUIRE(clickable(back));
	REQUIRE_FALSE(hidden(image));

	fixture.complete_hold();
	CHECK(fixture.start_count == 1);
	CHECK_FALSE(clickable(back));
	CHECK(hidden(image));

	fixture.lvgl.release();
	fixture.page.apply_action_result(ui::TailscaleActionKind::Start, false,
					 "TIMEOUT", 1);
	CHECK(code_text(fixture.page) == "TIMEOUT");
	CHECK(clickable(back));
	CHECK_FALSE(hidden(image));
	CHECK(clickable(action_button(fixture.page)));
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK(button_text(action_button(fixture.page)) == "Start");
}

TEST_CASE("Tailscale page locks navigation while exit pends",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	lv_obj_t *back = back_button(fixture.page);
	lv_obj_t *image = back_image(back);

	fixture.complete_hold();
	CHECK(fixture.exit_count == 1);
	CHECK(has_pending_arc(action_button(fixture.page)));
	CHECK_FALSE(clickable(back));
	CHECK(hidden(image));

	fixture.lvgl.release();
	fixture.page.apply_action_result(ui::TailscaleActionKind::Exit, true, "",
					 1);
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK(clickable(back));
	CHECK_FALSE(hidden(image));

	fixture.page.apply_snapshot(not_initialized_snapshot(), 2);
	CHECK(button_text(action_button(fixture.page)) == "Start");
	CHECK(clickable(action_button(fixture.page)));
}

TEST_CASE("Tailscale page keeps the Start button idle on a rejected request",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.start_accepted = false;
	lv_obj_t *action = action_button(fixture.page);

	fixture.complete_hold();
	CHECK(fixture.start_count == 1);
	CHECK_FALSE(has_pending_arc(action));
	CHECK(clickable(action));
	CHECK(clickable(back_button(fixture.page)));
	CHECK(button_text(action) == "Start");

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page keeps the Exit button idle on a rejected request",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	fixture.exit_accepted = false;

	fixture.complete_hold();
	CHECK(fixture.exit_count == 1);
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK(clickable(action_button(fixture.page)));
	CHECK(clickable(back_button(fixture.page)));
	CHECK(button_text(action_button(fixture.page)) == "Exit");

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page restores navigation when a pending start pauses",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	lv_obj_t *back = back_button(fixture.page);

	fixture.complete_hold();
	REQUIRE(fixture.start_count == 1);
	REQUIRE_FALSE(clickable(back));

	fixture.page.on_pause();
	CHECK(clickable(back));
	CHECK_FALSE(hidden(back_image(back)));

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page clears the start pending when the stage jumps ahead",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	lv_obj_t *back = back_button(fixture.page);

	fixture.complete_hold();
	REQUIRE(fixture.start_count == 1);
	REQUIRE(has_pending_arc(action_button(fixture.page)));
	REQUIRE_FALSE(clickable(back));

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 2);
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK(clickable(back));
	CHECK_FALSE(hidden(back_image(back)));
	CHECK(button_text(action_button(fixture.page)) == "Exit");

	fixture.lvgl.release();
	CHECK(fixture.exit_count == 0);
}

TEST_CASE("Tailscale page clears an exit pending when the stage reopens",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	fixture.complete_hold();
	REQUIRE(fixture.exit_count == 1);
	REQUIRE(has_pending_arc(action_button(fixture.page)));

	fixture.page.apply_snapshot(client_init_snapshot(), 2);
	CHECK(hidden(action_button(fixture.page)));
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page does not fire the new role from a running hold",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(not_initialized_snapshot(), 1);
	REQUIRE(button_text(action_button(fixture.page)) == "Start");

	fixture.lvgl.press(action_button(fixture.page));
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 2);
	REQUIRE(button_text(action_button(fixture.page)) == "Exit");

	fixture.lvgl.advance(HOLD_DURATION_MS);
	fixture.lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	CHECK(fixture.start_count == 0);
	CHECK(fixture.exit_count == 0);

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page drops an exit hold when the stage falls back",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	REQUIRE(button_text(action_button(fixture.page)) == "Exit");

	fixture.lvgl.press(action_button(fixture.page));
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);

	fixture.page.apply_snapshot(client_init_snapshot(), 2);
	CHECK(hidden(action_button(fixture.page)));

	fixture.lvgl.advance(HOLD_DURATION_MS);
	fixture.lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	CHECK(fixture.exit_count == 0);
	CHECK(fixture.start_count == 0);

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page keeps the back button reachable under the stage overlap",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	int back_count = 0;
	fixture.page.on_back_click = [&back_count]() { ++back_count; };

	lv_obj_t *stage_obj = stage(fixture.page);
	lv_obj_t *back = back_button(fixture.page);
	REQUIRE(stage_obj != nullptr);
	REQUIRE(back != nullptr);
	CHECK_FALSE(clickable(stage_obj));

	lv_area_t stage_area {};
	lv_obj_get_coords(stage_obj, &stage_area);
	lv_area_t back_area {};
	lv_obj_get_coords(back, &back_area);

	const int32_t overlap_x = (stage_area.x1 + stage_area.x2) / 2;
	const int32_t overlap_y = back_area.y2 - 1;
	REQUIRE(overlap_x >= stage_area.x1);
	REQUIRE(overlap_x <= stage_area.x2);
	REQUIRE(overlap_y >= stage_area.y1);
	REQUIRE(overlap_y <= stage_area.y2);
	REQUIRE(overlap_y >= back_area.y1);
	REQUIRE(overlap_y <= back_area.y2);

	fixture.lvgl.press_at(overlap_x, overlap_y);
	fixture.lvgl.release();
	CHECK(back_count == 1);
}

TEST_CASE("Tailscale page renders pairing payloads verbatim except the six-char split",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(client_init_snapshot(), 1);

	struct CodeCase {
		const char *code;
		const char *expected;
	};
	static const CodeCase cases[] = {
		{ "", "--" },
		{ "1", "1" },
		{ "12345", "12345" },
		{ "123456", "123 456" },
		{ "1234567", "1234567" },
		{ "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" },
	};

	uint64_t seq = 0;
	for (const CodeCase &entry : cases) {
		++seq;
		fixture.page.apply_code(entry.code, seq);
		CHECK(std::string(lv_label_get_text(code_label(fixture.page))) ==
		      entry.expected);
		CHECK(lv_obj_get_style_text_font(code_label(fixture.page),
						 LV_PART_MAIN) ==
		      &MontserratBold_36);
	}
}

TEST_CASE("Tailscale page keeps the start failure message until startup begins",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.complete_hold();
	fixture.lvgl.release();
	fixture.page.apply_action_result(ui::TailscaleActionKind::Start, false,
					 "FAIL", 1);
	REQUIRE(code_text(fixture.page) == "FAIL");

	fixture.page.apply_snapshot(not_initialized_snapshot(), 1);
	CHECK(code_text(fixture.page) == "FAIL");

	fixture.page.apply_snapshot(client_init_snapshot(), 2);
	CHECK(code_text(fixture.page) == "--");
}

TEST_CASE("Tailscale page keeps the ready failure message across heartbeats",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);

	fixture.complete_hold();
	fixture.lvgl.release();
	fixture.page.apply_action_result(ui::TailscaleActionKind::Exit, false,
					 "FAIL", 1);
	REQUIRE(code_text(fixture.page) == "FAIL");
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));
	CHECK_FALSE(has_pending_arc(action_button(fixture.page)));
	CHECK(clickable(action_button(fixture.page)));
	CHECK(clickable(back_button(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 2);
	CHECK(fixture.page.rendered_seq() == 2);
	CHECK(code_text(fixture.page) == "FAIL");
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 3);
	CHECK(code_text(fixture.page) == "FAIL");
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(not_initialized_snapshot(), 4);
	CHECK(code_text(fixture.page) == "--");
	CHECK_FALSE(hidden(code_label(fixture.page)));
}

TEST_CASE("Tailscale page clears a ready failure message when a new attempt starts",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;
	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	fixture.page.apply_action_result(ui::TailscaleActionKind::Exit, false,
					 "FAIL", 1);
	REQUIRE(code_text(fixture.page) == "FAIL");

	fixture.complete_hold();
	CHECK(code_text(fixture.page) == "--");
	CHECK(fixture.exit_count == 1);

	fixture.lvgl.release();
}

TEST_CASE("Tailscale page ignores observations while hidden",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	REQUIRE(fixture.page.rendered_seq() == 1);
	REQUIRE(button_text(action_button(fixture.page)) == "Exit");

	lv_obj_add_flag(fixture.page.get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.page.apply_snapshot(ready_snapshot(false, ""), 2);
	CHECK(fixture.page.rendered_seq() == 1);
	CHECK(button_text(action_button(fixture.page)) == "Exit");

	lv_obj_clear_flag(fixture.page.get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.page.apply_snapshot(not_initialized_snapshot(), 2);
	CHECK(fixture.page.rendered_seq() == 2);
	CHECK(button_text(action_button(fixture.page)) == "Start");
	CHECK(hidden(ip_row(fixture.page)));
}

TEST_CASE("Tailscale page resets to the stand-in on entry",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	REQUIRE(button_text(action_button(fixture.page)) == "Exit");
	fixture.page.on_exit();
	CHECK(fixture.visibility_false_count == 1);

	fixture.page.on_enter();
	CHECK(fixture.visibility_true_count == 1);
	CHECK(fixture.page.rendered_seq() == 0);
	CHECK(button_text(action_button(fixture.page)) == "Start");
	CHECK_FALSE(hidden(code_label(fixture.page)));
	CHECK(code_text(fixture.page) == "--");
	CHECK(hidden(ip_row(fixture.page)));

	fixture.page.apply_snapshot(ready_snapshot(true, "100.64.0.1"), 1);
	CHECK(button_text(action_button(fixture.page)) == "Exit");
	CHECK(row_value_text(ip_row(fixture.page)) == "100.64.0.1");
}

TEST_CASE("Tailscale page lifecycle cancels an unfinished start hold",
	  "[ui][tailscale]")
{
	TailscalePageFixture fixture;

	fixture.lvgl.press(action_button(fixture.page));
	fixture.lvgl.advance(HOLD_DURATION_MS / 2);
	REQUIRE(button_text(action_button(fixture.page)) == "Keep");

	fixture.page.on_pause();
	CHECK(button_text(action_button(fixture.page)) == "Start");
	CHECK(fixture.start_count == 0);

	fixture.lvgl.release();
	fixture.lvgl.advance(HOLD_DURATION_MS);
	CHECK(fixture.start_count == 0);
}
