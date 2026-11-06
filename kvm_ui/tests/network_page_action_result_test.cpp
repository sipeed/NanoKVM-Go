#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"

namespace
{

constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
constexpr uint32_t START_PENDING_TIMEOUT_MS = 15000;

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

lv_obj_t *on_button(const ui::Page &page)
{
	return scroll_child(page, 2);
}

lv_obj_t *terms_row(const ui::Page &page)
{
	return scroll_child(page, 3);
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

lv_obj_t *ip_row(const ui::Page &page)
{
	lv_obj_t *stage = code_stage(page);
	if (stage == nullptr || lv_obj_get_child_count(stage) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(stage, 1);
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

std::string code_text(const ui::Page &page)
{
	lv_obj_t *label = code_label(page);
	if (label == nullptr) {
		return {};
	}
	const char *text = lv_label_get_text(label);
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

bool has_pending_arc(lv_obj_t *obj)
{
	return test_support::find_visible_descendant_of_type(
		       obj, &lv_arc_class) != nullptr;
}

class NetworkActionResultFixture {
    public:
	NetworkActionResultFixture()
	{
		_page.create(_lvgl.screen());
		refresh_layout();
	}

	~NetworkActionResultFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkActionResultFixture(const NetworkActionResultFixture &) = delete;
	NetworkActionResultFixture &
	operator=(const NetworkActionResultFixture &) = delete;

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

	void complete_exit_hold()
	{
		constexpr uint32_t fill_ms =
			ON_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(exit_button(_page));
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		_lvgl.release();
	}

	void enter_terms()
	{
		complete_on_hold();
		refresh_layout();
	}

	void enter_paired()
	{
		_page.apply_snapshot(ready_snapshot(), 1);
		refresh_layout();
	}

    private:
	test_support::LvglTestContext _lvgl;
	ui::NetworkPage _page;
};

void arm_start_lane(NetworkActionResultFixture &fixture, uint32_t &start_calls)
{
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	fixture.enter_terms();
	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
}

}

TEST_CASE("Network start failure renders FAIL and holds it across "
	  "heartbeats",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	CHECK(start_calls == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "FAIL", 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "FAIL");
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "FAIL");

	fixture.page().apply_snapshot(client_init_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));

	fixture.page().apply_code("123456", 1);
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network start timeout renders TIMEOUT and holds it across "
	  "heartbeats",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "TIMEOUT", 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "TIMEOUT");
	CHECK(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "TIMEOUT");
}

TEST_CASE("Network result arriving after the watchdog is ignored",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "FAIL", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(spinner_visible(fixture.page()));
}

TEST_CASE("Network repeated and stale action results are dropped",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "FAIL", 1);
	fixture.refresh_layout();
	REQUIRE(code_text(fixture.page()) == "FAIL");

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "OTHER", 1);
	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "OTHER", 0);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "FAIL");
}

TEST_CASE("Network successful start keeps the pending until the snapshot "
	  "confirms",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	REQUIRE_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, true,
					   "", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network exit failure renders inside the paired stage and holds",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t exit_calls = 0;
	fixture.page().on_exit_request = [&exit_calls]() -> bool {
		++exit_calls;
		return true;
	};
	fixture.enter_paired();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(exit_calls == 1);
	REQUIRE_FALSE(clickable(back_button(fixture.page())));
	REQUIRE(has_pending_arc(exit_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit, false,
					   "NO_DAEMON", 2);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "NO_DAEMON");
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.page().apply_snapshot(ready_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(code_text(fixture.page()) == "NO_DAEMON");
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK(hidden(ip_row(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network a new exit attempt clears the held message",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t exit_calls = 0;
	fixture.page().on_exit_request = [&exit_calls]() -> bool {
		++exit_calls;
		return true;
	};
	fixture.enter_paired();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit, false,
					   "NO_DAEMON", 2);
	fixture.refresh_layout();
	REQUIRE(code_text(fixture.page()) == "NO_DAEMON");
	REQUIRE_FALSE(hidden(code_label(fixture.page())));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(exit_calls == 2);
	CHECK(hidden(code_label(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network a new start attempt clears the held message",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	REQUIRE(start_calls == 1);

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "FAIL", 1);
	fixture.refresh_layout();
	REQUIRE(code_text(fixture.page()) == "FAIL");

	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);

	fixture.enter_terms();
	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();

	CHECK(start_calls == 2);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));
}

TEST_CASE("Network consent failure renders CONSENT",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t start_calls = 0;
	arm_start_lane(fixture, start_calls);
	REQUIRE_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start, false,
					   "CONSENT", 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "CONSENT");
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network successful exit keeps the pending until the snapshot "
	  "confirms",
	  "[ui][network][action-result]")
{
	NetworkActionResultFixture fixture;
	uint32_t exit_calls = 0;
	fixture.page().on_exit_request = [&exit_calls]() -> bool {
		++exit_calls;
		return true;
	};
	fixture.enter_paired();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Paired);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(exit_calls == 1);
	REQUIRE_FALSE(clickable(back_button(fixture.page())));
	REQUIRE(has_pending_arc(exit_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit, true,
					   "", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK_FALSE(clickable(back_button(fixture.page())));
	CHECK(has_pending_arc(exit_button(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
}
