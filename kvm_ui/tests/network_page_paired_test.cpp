#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"
#include "ui/scona.hpp"

namespace
{

constexpr uint32_t EXIT_HOLD_DURATION_MS = 2000;
constexpr uint32_t PENDING_TIMEOUT_MS = 15000;
constexpr int32_t STAGE_LEFT = 13;
constexpr int32_t STAGE_TOP = 44;
constexpr int32_t STAGE_WIDTH = 230;
constexpr int32_t STAGE_HEIGHT = 152;
constexpr int32_t IP_ROW_TOP = 104;
constexpr int32_t EXIT_LEFT = 13;
constexpr int32_t EXIT_TOP = 164;

ui::TailscaleSnapshotView ready_snapshot(const char *ip = "100.64.0.1")
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = true;
	snapshot.ip = ip;
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

ui::TailscaleSnapshotView not_initialized_snapshot()
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = false;
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

lv_obj_t *code_stage(const ui::Page &page)
{
	return scroll_child(page, 5);
}

lv_obj_t *exit_button(const ui::Page &page)
{
	return scroll_child(page, 6);
}

lv_obj_t *code_label(const ui::Page &page)
{
	lv_obj_t *stage = code_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 1) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 0);
}

lv_obj_t *code_spinner(const ui::Page &page)
{
	lv_obj_t *stage = code_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 3) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 2);
}

bool spinner_visible(const ui::Page &page)
{
	lv_obj_t *spinner = code_spinner(page);
	return spinner != nullptr &&
	       !lv_obj_has_flag(spinner, LV_OBJ_FLAG_HIDDEN);
}

lv_obj_t *ip_row(const ui::Page &page)
{
	lv_obj_t *stage = code_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 1);
}

std::string ip_row_key_text(const ui::Page &page)
{
	lv_obj_t *row = ip_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) < 1) {
		return {};
	}
	const char *text = lv_label_get_text(lv_obj_get_child(row, 0));
	return text == nullptr ? std::string() : std::string(text);
}

std::string ip_row_value_text(const ui::Page &page)
{
	lv_obj_t *row = ip_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) < 2) {
		return {};
	}
	const char *text = lv_label_get_text(lv_obj_get_child(row, 1));
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

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

std::string button_text(lv_obj_t *obj)
{
	return std::string(test_support::visible_direct_text(obj));
}

bool has_pending_arc(lv_obj_t *obj)
{
	return test_support::find_visible_descendant_of_type(
		       obj, &lv_arc_class) != nullptr;
}

class NetworkPairedFixture {
    public:
	NetworkPairedFixture()
	{
		wire_exit_lane();
		_page.create(_lvgl.screen());
		refresh_layout();
	}

	~NetworkPairedFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkPairedFixture(const NetworkPairedFixture &) = delete;
	NetworkPairedFixture &operator=(const NetworkPairedFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::NetworkPage &page()
	{
		return _page;
	}

	int exit_count = 0;
	bool exit_accepted = true;

	void refresh_layout()
	{
		lv_obj_update_layout(_lvgl.screen());
	}

	void enter_paired()
	{
		_page.apply_snapshot(ready_snapshot(), 1);
		refresh_layout();
	}

	void complete_exit_hold()
	{
		constexpr uint32_t fill_ms =
			EXIT_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(exit_button(_page));
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		_lvgl.release();
	}

	void destroy_page()
	{
		_page.destroy();
	}

	void recreate_page()
	{
		wire_exit_lane();
		_page.create(_lvgl.screen());
		refresh_layout();
	}

    private:
	void wire_exit_lane()
	{
		_page.on_exit_request = [this]() -> bool {
			++exit_count;
			return exit_accepted;
		};
	}

	test_support::LvglTestContext _lvgl;
	ui::NetworkPage _page;
};

}

TEST_CASE("Network paired stage replicates the locked ready geometry",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;

	REQUIRE(exit_button(fixture.page()) != nullptr);
	REQUIRE(ip_row(fixture.page()) != nullptr);
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));

	fixture.enter_paired();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	lv_obj_t *stage = code_stage(fixture.page());
	const lv_area_t stage_coordinates = object_coords(stage);
	CHECK(stage_coordinates.x1 == STAGE_LEFT);
	CHECK(stage_coordinates.y1 == STAGE_TOP);
	CHECK(lv_obj_get_width(stage) == STAGE_WIDTH);
	CHECK(lv_obj_get_height(stage) == STAGE_HEIGHT);

	lv_obj_t *row = ip_row(fixture.page());
	CHECK_FALSE(hidden(row));
	const lv_area_t row_coordinates = object_coords(row);
	CHECK(row_coordinates.x1 == STAGE_LEFT);
	CHECK(row_coordinates.y1 == IP_ROW_TOP);
	CHECK(lv_obj_get_width(row) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(row) == ui::SConA::HEIGHT);
	CHECK(row_coordinates.y2 == IP_ROW_TOP + ui::SConA::HEIGHT - 1);
	CHECK(ip_row_key_text(fixture.page()) == "IP:");
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");
	CHECK(hidden(code_label(fixture.page())));

	lv_obj_t *exit = exit_button(fixture.page());
	CHECK_FALSE(hidden(exit));
	const lv_area_t exit_coordinates = object_coords(exit);
	CHECK(exit_coordinates.x1 == EXIT_LEFT);
	CHECK(exit_coordinates.y1 == EXIT_TOP);
	CHECK(lv_obj_get_width(exit) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(exit) == ui::BConA::HEIGHT);
	CHECK(exit_coordinates.y2 == EXIT_TOP + ui::BConA::HEIGHT - 1);
	CHECK(lv_obj_get_style_radius(exit, LV_PART_MAIN) == ui::BConA::RADIUS);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(exit, LV_PART_MAIN),
			  lv_color_hex(ui::BConA::COLOR_DEFAULT)));
	CHECK(test_support::find_visible_descendant_of_type(
		      exit, &lv_image_class) != nullptr);
	CHECK(button_text(exit) == "Exit");
	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network paired stage hides the Exit button and IP row in every "
	  "other stage",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;

	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));
	CHECK(hidden(code_label(fixture.page())));
	CHECK(spinner_visible(fixture.page()));

	fixture.page().apply_snapshot(ready_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK_FALSE(hidden(exit_button(fixture.page())));
	CHECK_FALSE(hidden(ip_row(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));
}

TEST_CASE("Network paired IP row refreshes from snapshots and hides when "
	  "empty",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");

	fixture.page().apply_snapshot(ready_snapshot("100.64.0.2"), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.2");
	CHECK_FALSE(hidden(ip_row(fixture.page())));

	fixture.page().apply_snapshot(ready_snapshot("100.64.0.3"), 2);
	fixture.page().apply_snapshot(ready_snapshot("100.64.0.3"), 1);
	fixture.refresh_layout();
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.2");

	fixture.page().apply_snapshot(ready_snapshot(""), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(hidden(ip_row(fixture.page())));
	CHECK_FALSE(hidden(exit_button(fixture.page())));
}

TEST_CASE("Network paired exit requires a full hold and pends until the "
	  "session leaves ready",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);

	fixture.lvgl().press(exit);
	fixture.lvgl().advance(200);
	fixture.lvgl().release();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 0);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK(button_text(exit) == "Exit");

	CHECK_FALSE(clickable(back_button(fixture.page())));
	CHECK(has_pending_arc(exit));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);

	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(code_label(fixture.page())));
	CHECK(spinner_visible(fixture.page()));
}

TEST_CASE("Network paired exit stays idle on rejected or unwired requests",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();

	fixture.exit_accepted = false;
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.page().on_exit_request = nullptr;
	const int previous_count = fixture.exit_count;
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == previous_count);
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
}

TEST_CASE("Network paired exit watchdog unlocks a never-confirmed exit",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(PENDING_TIMEOUT_MS - 1);
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.lvgl().advance(1);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));

	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 2);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network paired heartbeat re-render keeps an in-progress exit "
	  "hold alive",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();

	lv_obj_t *exit = exit_button(fixture.page());
	fixture.lvgl().press(exit);
	fixture.lvgl().advance(1000);
	fixture.page().apply_snapshot(ready_snapshot("100.64.0.9"), 2);
	fixture.refresh_layout();
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.9");
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	constexpr uint32_t remaining_ms =
		EXIT_HOLD_DURATION_MS - 1000 -
		ui::HoldConfirmBehavior::FULL_DWELL_MS;
	fixture.lvgl().advance(remaining_ms);
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	fixture.lvgl().release();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network paired still-ready heartbeat keeps the exit pending "
	  "window alive",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(fixture.exit_count == 1);
	REQUIRE(has_pending_arc(exit_button(fixture.page())));
	REQUIRE_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(ready_snapshot("100.64.0.10"), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.10");
	CHECK(has_pending_arc(exit_button(fixture.page())));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);

	fixture.page().apply_snapshot(client_init_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));
}

TEST_CASE("Network paired pending clears on pause, exit, enter, and "
	  "recreate",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().on_pause();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	fixture.page().apply_snapshot(ready_snapshot(), 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.destroy_page();
	fixture.recreate_page();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));
}

TEST_CASE("Network paired exit lane survives a destroy and recreate cycle",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.destroy_page();
	fixture.recreate_page();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);

	fixture.enter_paired();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	const int exit_before = fixture.exit_count;
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == exit_before + 1);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network paired failure message keeps the loading arc stopped",
	  "[ui][network][paired]")
{
	NetworkPairedFixture fixture;
	fixture.enter_paired();
	REQUIRE_FALSE(spinner_visible(fixture.page()));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(fixture.exit_count == 1);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit, false,
					   "FAIL", 1);
	fixture.refresh_layout();
	lv_obj_t *label = code_label(fixture.page());
	REQUIRE(label != nullptr);
	CHECK_FALSE(hidden(label));
	const char *text = lv_label_get_text(label);
	CHECK((text != nullptr && std::string(text) == "FAIL"));
	CHECK_FALSE(spinner_visible(fixture.page()));
}
