#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>
#include <string>

#include <lvgl.h>

#include "loading_arc.hpp"
#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_36);
}

namespace
{

constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
constexpr uint32_t EXIT_HOLD_DURATION_MS = 2000;
constexpr uint32_t START_PENDING_TIMEOUT_MS = 15000;
constexpr int32_t CODE_STAGE_LEFT = 13;
constexpr int32_t CODE_STAGE_TOP = 44;
constexpr int32_t CODE_STAGE_WIDTH = 230;
constexpr int32_t CODE_STAGE_HEIGHT = 152;
constexpr int32_t EXIT_LEFT = 13;
constexpr int32_t EXIT_TOP = 164;
// The open band between the title row (bottom edge) and the Exit row (top
// edge); the code-stage content is locked to its pixel midpoint.
constexpr int32_t TITLE_BOTTOM_EDGE = 12 + ui::BConA::HEIGHT;
constexpr int32_t CONTENT_BAND_CENTER_Y =
	(TITLE_BOTTOM_EDGE + (EXIT_TOP - 1)) / 2;
constexpr int32_t CODE_LABEL_CENTER_Y = CONTENT_BAND_CENTER_Y;

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

std::string ip_row_value_text(const ui::Page &page)
{
	lv_obj_t *row = ip_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) < 2) {
		return {};
	}
	const char *text = lv_label_get_text(lv_obj_get_child(row, 1));
	return text == nullptr ? std::string() : std::string(text);
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

lv_obj_t *decline_button(const ui::Page &page)
{
	return row_button(page, 1);
}

bool hidden(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_HIDDEN);
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

bool clickable(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_has_flag(obj, LV_OBJ_FLAG_CLICKABLE);
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

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

class NetworkCodeFixture {
    public:
	NetworkCodeFixture()
	{
		wire_exit_lane();
		_page.create(_lvgl.screen());
		// Deliberately opt the gate in so this suite keeps covering the
		// enabled exit machinery; the shipped default (hidden) is owned
		// by the dedicated raw-page guard test below.
		_page.set_code_stage_exit_enabled(true);
		refresh_layout();
	}

	~NetworkCodeFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkCodeFixture(const NetworkCodeFixture &) = delete;
	NetworkCodeFixture &operator=(const NetworkCodeFixture &) = delete;

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

	void complete_hold()
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
			EXIT_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(exit_button(_page));
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		_lvgl.release();
	}

	void enter_terms()
	{
		complete_hold();
		refresh_layout();
	}

	void destroy_page()
	{
		_page.destroy();
	}

	void recreate_page()
	{
		wire_exit_lane();
		_page.create(_lvgl.screen());
		_page.set_code_stage_exit_enabled(true);
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

void require_stage_widgets(const ui::Page &page)
{
	REQUIRE(back_button(page) != nullptr);
	REQUIRE(preinit_group(page) != nullptr);
	REQUIRE(on_button(page) != nullptr);
	REQUIRE(terms_row(page) != nullptr);
	REQUIRE(terms_body(page) != nullptr);
	REQUIRE(code_stage(page) != nullptr);
	REQUIRE(exit_button(page) != nullptr);
	REQUIRE(ip_row(page) != nullptr);
}

void require_preinit_stage(const ui::Page &page)
{
	require_stage_widgets(page);
	CHECK_FALSE(hidden(back_button(page)));
	CHECK_FALSE(hidden(preinit_group(page)));
	CHECK_FALSE(hidden(on_button(page)));
	CHECK(hidden(terms_row(page)));
	CHECK(hidden(terms_body(page)));
	CHECK(hidden(code_stage(page)));
	CHECK(hidden(exit_button(page)));
}

void require_terms_stage(const ui::Page &page)
{
	require_stage_widgets(page);
	CHECK(hidden(back_button(page)));
	CHECK(hidden(preinit_group(page)));
	CHECK(hidden(on_button(page)));
	CHECK_FALSE(hidden(terms_row(page)));
	CHECK_FALSE(hidden(terms_body(page)));
	CHECK(hidden(code_stage(page)));
	CHECK(hidden(exit_button(page)));
}

void require_code_holder_visible(const ui::Page &page)
{
	require_stage_widgets(page);
	CHECK_FALSE(hidden(back_button(page)));
	CHECK(hidden(preinit_group(page)));
	CHECK(hidden(on_button(page)));
	CHECK(hidden(terms_row(page)));
	CHECK(hidden(terms_body(page)));
	CHECK_FALSE(hidden(code_stage(page)));
	CHECK(hidden(exit_button(page)));
	CHECK(hidden(code_label(page)));
	CHECK(spinner_visible(page));
	CHECK(hidden(ip_row(page)));
}

void require_paired_stage(const ui::Page &page)
{
	require_stage_widgets(page);
	CHECK_FALSE(hidden(back_button(page)));
	CHECK(hidden(preinit_group(page)));
	CHECK(hidden(on_button(page)));
	CHECK(hidden(terms_row(page)));
	CHECK(hidden(terms_body(page)));
	CHECK_FALSE(hidden(code_stage(page)));
	CHECK(hidden(code_label(page)));
	CHECK_FALSE(hidden(ip_row(page)));
	CHECK_FALSE(hidden(exit_button(page)));
}

}

TEST_CASE("Network code stage replicates the locked client-init geometry",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	REQUIRE(code_stage(fixture.page()) != nullptr);
	CHECK(hidden(code_stage(fixture.page())));
	require_preinit_stage(fixture.page());

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	require_code_holder_visible(fixture.page());

	lv_obj_t *stage = code_stage(fixture.page());
	const lv_area_t stage_coordinates = object_coords(stage);
	CHECK(stage_coordinates.x1 == CODE_STAGE_LEFT);
	CHECK(stage_coordinates.y1 == CODE_STAGE_TOP);
	CHECK(lv_obj_get_width(stage) == CODE_STAGE_WIDTH);
	CHECK(lv_obj_get_height(stage) == CODE_STAGE_HEIGHT);
	CHECK(stage_coordinates.y2 == CODE_STAGE_TOP + CODE_STAGE_HEIGHT - 1);
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
	CHECK(lv_obj_get_child_count(stage) == 3);

	lv_obj_t *label = code_label(fixture.page());
	REQUIRE(label != nullptr);
	CHECK(hidden(label));
	CHECK(lv_obj_get_style_text_font(label, LV_PART_MAIN) ==
	      &MontserratBold_36);
	CHECK(lv_obj_get_style_text_align(label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
	CHECK(lv_color_eq(lv_obj_get_style_text_color(label, LV_PART_MAIN),
			  lv_color_white()));
	CHECK(lv_label_get_long_mode(label) == LV_LABEL_LONG_WRAP);
	CHECK(code_text(fixture.page()).empty());
	lv_obj_t *spinner = code_spinner(fixture.page());
	REQUIRE(spinner != nullptr);
	CHECK(lv_obj_get_width(spinner) == ui::LoadingArc::SIZE);
	CHECK(lv_obj_get_height(spinner) == ui::LoadingArc::SIZE);
	const lv_area_t spinner_coordinates = object_coords(spinner);
	CHECK(spinner_coordinates.x1 ==
	      CODE_STAGE_LEFT +
		      (CODE_STAGE_WIDTH -
		       static_cast<int32_t>(ui::LoadingArc::SIZE)) /
			      2);
	CHECK(spinner_coordinates.y1 ==
	      CODE_STAGE_TOP +
		      (CODE_STAGE_HEIGHT -
		       static_cast<int32_t>(ui::LoadingArc::SIZE)) /
			      2);
	CHECK((spinner_coordinates.y1 + spinner_coordinates.y2) / 2 ==
	      CODE_LABEL_CENTER_Y);

	// A resolved code swaps the arc for the label in the same slot.
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK_FALSE(hidden(label));
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK(lv_obj_get_width(label) == CODE_STAGE_WIDTH);
	const lv_area_t label_coordinates = object_coords(label);
	CHECK((label_coordinates.y1 + label_coordinates.y2) / 2 ==
	      CODE_LABEL_CENTER_Y);

	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network code stage routes on entry per API stage",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	ui::TailscaleSnapshotView unknown;
	fixture.page().apply_snapshot(unknown, 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	require_preinit_stage(fixture.page());

	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	require_preinit_stage(fixture.page());

	fixture.page().apply_snapshot(client_init_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	require_code_holder_visible(fixture.page());

	fixture.page().apply_snapshot(ready_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	require_paired_stage(fixture.page());
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");

	fixture.page().apply_snapshot(client_init_snapshot(), 5);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	require_code_holder_visible(fixture.page());

	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	require_preinit_stage(fixture.page());
	CHECK(fixture.page().rendered_seq() == 0);
	fixture.page().apply_snapshot(client_init_snapshot(), 5);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
}

TEST_CASE("Network code label groups the pairing code",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();

	fixture.page().apply_code("123456", 1);
	CHECK(code_text(fixture.page()) == "123 456");

	fixture.page().apply_code("12", 2);
	CHECK(code_text(fixture.page()) == "12");

	fixture.page().apply_code("", 3);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));

	fixture.page().apply_code("999999", 3);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));
	fixture.page().apply_code("999999", 2);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));
	fixture.page().apply_code("999999", 4);
	CHECK(code_text(fixture.page()) == "999 999");
	CHECK_FALSE(spinner_visible(fixture.page()));
}

TEST_CASE("Network code updates are gated to the client-init stage",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(not_initialized_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(spinner_visible(fixture.page()));

	fixture.page().apply_snapshot(ready_snapshot(), 2);
	fixture.page().apply_code("123456", 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK(hidden(code_label(fixture.page())));
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");
}

TEST_CASE("Network snapshot sequence is monotonic and hidden pages ignore "
	  "updates",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 5);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 5);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	fixture.page().apply_snapshot(not_initialized_snapshot(), 5);
	fixture.page().apply_snapshot(not_initialized_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 5);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	fixture.page().apply_snapshot(not_initialized_snapshot(), 6);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 6);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);

	lv_obj_add_flag(fixture.page().get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.page().apply_snapshot(client_init_snapshot(), 7);
	fixture.page().apply_code("123456", 1);
	lv_obj_clear_flag(fixture.page().get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 6);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);

	fixture.page().apply_snapshot(client_init_snapshot(), 7);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network terms caches snapshots and codes without re-routing",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Terms);
	require_terms_stage(fixture.page());

	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(spinner_visible(fixture.page()));
	require_terms_stage(fixture.page());

	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	lv_obj_send_event(decline, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	require_preinit_stage(fixture.page());

	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network terms agree opens the code stage with locked back "
	  "navigation",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	uint32_t start_calls = 0;
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();

	CHECK(start_calls == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	require_code_holder_visible(fixture.page());
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));

	fixture.page().apply_code("654321", 1);
	CHECK(code_text(fixture.page()) == "654 321");
}

TEST_CASE("Network start watchdog unlocks and falls back when the start "
	  "never confirms",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().on_start_request = []() -> bool {
		return true;
	};
	fixture.enter_terms();

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS - 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(1);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	require_preinit_stage(fixture.page());

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS - 1);
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.lvgl().advance(1);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
}

TEST_CASE("Network page notifies session visibility on enter and exit",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	uint32_t true_calls = 0;
	uint32_t false_calls = 0;
	fixture.page().on_session_visibility =
		[&true_calls, &false_calls](bool visible) {
			if (visible) {
				++true_calls;
			} else {
				++false_calls;
			}
		};

	fixture.page().on_enter();
	CHECK(true_calls == 1);
	CHECK(false_calls == 0);

	fixture.page().on_exit();
	CHECK(true_calls == 1);
	CHECK(false_calls == 1);

	fixture.page().on_enter();
	CHECK(true_calls == 2);
	CHECK(false_calls == 1);

	fixture.destroy_page();
	fixture.page().on_exit();
	CHECK(true_calls == 2);
	CHECK(false_calls == 1);
}

TEST_CASE("Network code stage resets on exit, enter, destroy, and recreate",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");

	fixture.page().on_pause();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	require_preinit_stage(fixture.page());
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()).empty());
	CHECK(spinner_visible(fixture.page()));

	fixture.destroy_page();
	fixture.recreate_page();
	require_preinit_stage(fixture.page());
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(fixture.page().rendered_seq() == 0);
	CHECK(code_text(fixture.page()).empty());
	CHECK_FALSE(spinner_visible(fixture.page()));
}

TEST_CASE("Network code loading arc rotates until the code arrives",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();

	lv_obj_t *spinner = code_spinner(fixture.page());
	REQUIRE(spinner != nullptr);
	REQUIRE(spinner_visible(fixture.page()));
	CHECK(hidden(code_label(fixture.page())));
	CHECK(lv_arc_get_rotation(spinner) == 0);

	fixture.lvgl().advance(ui::LoadingArc::PERIOD_MS);
	CHECK(lv_arc_get_rotation(spinner) == ui::LoadingArc::ROTATION_STEP_DEG);
	fixture.lvgl().advance(ui::LoadingArc::PERIOD_MS);
	CHECK(lv_arc_get_rotation(spinner) ==
	      2 * ui::LoadingArc::ROTATION_STEP_DEG);

	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network code loading arc pauses and resumes with the page",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	const std::size_t running = fixture.lvgl().timer_count();

	fixture.page().on_pause();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == running - 1U);

	fixture.page().on_resume();
	CHECK(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == running);

	// A resolved code keeps the arc stopped across pause and resume.
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	REQUIRE(code_text(fixture.page()) == "123 456");
	const std::size_t resolved = fixture.lvgl().timer_count();
	fixture.page().on_pause();
	fixture.page().on_resume();
	CHECK(fixture.lvgl().timer_count() == resolved);
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network code loading arc stops on every stage exit path",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	// Terms never starts the arc; measure the baseline after the hold
	// gesture settled.
	fixture.enter_terms();
	require_terms_stage(fixture.page());
	CHECK_FALSE(spinner_visible(fixture.page()));
	const std::size_t baseline = fixture.lvgl().timer_count();

	// Terms is snapshot-immune; decline back to PreInit before routing.
	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	lv_obj_send_event(decline, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	require_preinit_stage(fixture.page());

	// The Code stage runs exactly one spin timer.
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == baseline + 1U);

	// Routing back to PreInit stops it.
	fixture.page().apply_snapshot(not_initialized_snapshot(), 2);
	fixture.refresh_layout();
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == baseline);

	// Routing to Paired stops it too.
	fixture.page().apply_snapshot(client_init_snapshot(), 3);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	fixture.page().apply_snapshot(ready_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == baseline);

	// Teardown while running restores the baseline and a late on_exit
	// stays safe.
	fixture.page().apply_snapshot(client_init_snapshot(), 5);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	fixture.destroy_page();
	CHECK(fixture.lvgl().timer_count() == baseline);
	fixture.page().on_exit();
	CHECK(fixture.lvgl().timer_count() == baseline);
}

TEST_CASE("Network start failure message suppresses the loading arc",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	uint32_t start_calls = 0;
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	fixture.enter_terms();

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	REQUIRE(start_calls == 1);
	REQUIRE(spinner_visible(fixture.page()));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start,
					   false, "FAIL", 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "FAIL");
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK_FALSE(spinner_visible(fixture.page()));

	// The next client-init heartbeat clears the hold and resumes loading.
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()).empty());
	CHECK(hidden(code_label(fixture.page())));
	CHECK(spinner_visible(fixture.page()));
}

TEST_CASE("Network disabled stage never runs the loading arc",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().on_start_request = []() -> bool {
		return true;
	};
	fixture.enter_terms();

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	const std::size_t spinning = fixture.lvgl().timer_count();

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start,
					   false, "DISABLED", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK_FALSE(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == spinning - 1U);
}

TEST_CASE("Network start watchdog re-routes a cached client-init snapshot "
	  "back to the loading arc",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().on_start_request = []() -> bool {
		return true;
	};
	fixture.enter_terms();
	// Cache a client-init snapshot while TERMS suppresses routing.
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	require_terms_stage(fixture.page());

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	const std::size_t spinning = fixture.lvgl().timer_count();

	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS);
	fixture.refresh_layout();
	// The pending window is released (back navigation restored) but the
	// cached client-init snapshot keeps the arc spinning in the Code stage.
	CHECK(clickable(back_button(fixture.page())));
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(code_label(fixture.page())));
	CHECK(spinner_visible(fixture.page()));
	CHECK(fixture.lvgl().timer_count() == spinning);
}

TEST_CASE("Network code delivered while hidden renders text on entry",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	lv_obj_add_flag(fixture.page().get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().rendered_seq() == 0);
	CHECK_FALSE(spinner_visible(fixture.page()));

	lv_obj_clear_flag(fixture.page().get_container(), LV_OBJ_FLAG_HIDDEN);
	fixture.page().on_enter();
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(spinner_visible(fixture.page()));

	fixture.page().apply_code("123456", 1);
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK_FALSE(spinner_visible(fixture.page()));
}

TEST_CASE("Network code stage bundles the exit button with the code label",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	// While the spinner runs, the exit button hides with the code label.
	CHECK(spinner_visible(fixture.page()));
	CHECK(hidden(code_label(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK(code_text(fixture.page()) == "123 456");
	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	CHECK_FALSE(hidden(exit));
	const lv_area_t exit_coordinates = object_coords(exit);
	CHECK(exit_coordinates.x1 == EXIT_LEFT);
	CHECK(exit_coordinates.y1 == EXIT_TOP);
	CHECK(lv_obj_get_width(exit) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(exit) == ui::BConA::HEIGHT);
	CHECK(button_text(exit) == "Exit");
	CHECK(clickable(back_button(fixture.page())));

	// A late empty code withdraws the label and the bundled button.
	fixture.page().apply_code("", 2);
	fixture.refresh_layout();
	CHECK(spinner_visible(fixture.page()));
	CHECK(hidden(code_label(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	fixture.page().apply_code("654321", 3);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "654 321");
	CHECK_FALSE(hidden(exit_button(fixture.page())));
}

TEST_CASE("Network code exit hold pends with the code label still shown",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	REQUIRE_FALSE(hidden(code_label(fixture.page())));

	// A short press must not arm the exit request.
	fixture.lvgl().press(exit);
	fixture.lvgl().advance(200);
	fixture.lvgl().release();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 0);
	CHECK_FALSE(has_pending_arc(exit));
	CHECK(clickable(back_button(fixture.page())));

	fixture.complete_exit_hold();
	fixture.refresh_layout();

	// The stage-aware dispatch re-renders the code stage: the code label
	// stays visible (render_code), the pending arc shows, and back
	// navigation locks.
	CHECK(fixture.exit_count == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK(button_text(exit) == "Exit");
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network code heartbeat re-render keeps an in-progress exit hold "
	  "alive",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);

	// A client-init heartbeat mid-hold re-enters show_code_stage() through
	// route_from_snapshot(); the hold must survive the re-render.
	fixture.lvgl().press(exit);
	fixture.lvgl().advance(1000);
	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();

	constexpr uint32_t fill_ms =
		EXIT_HOLD_DURATION_MS - ui::HoldConfirmBehavior::FULL_DWELL_MS;
	fixture.lvgl().advance(fill_ms - 1000);
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	fixture.lvgl().release();
	fixture.refresh_layout();

	CHECK(fixture.exit_count == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
}

TEST_CASE("Network code exit button stays hidden in the terms and disabled "
	  "stages",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;
	fixture.page().on_start_request = []() -> bool {
		return true;
	};

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	CHECK(hidden(exit_button(fixture.page())));

	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	lv_obj_send_event(decline, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	require_preinit_stage(fixture.page());
	CHECK(hidden(exit_button(fixture.page())));

	fixture.enter_terms();
	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(hidden(exit_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Start,
					   false, "DISABLED", 1);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Disabled);
	CHECK(hidden(exit_button(fixture.page())));
}

TEST_CASE("Network code exit pends until the session leaves client-init",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);

	// A short press never issues the request.
	fixture.lvgl().press(exit);
	fixture.lvgl().advance(200);
	fixture.lvgl().release();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 0);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	// A second hold while pending is swallowed by the pending guard.
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);

	// Client-init heartbeats keep the pending: the session has not left
	// the stage the exit was armed in.
	fixture.page().apply_snapshot(client_init_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	// Leaving client-init confirms the exit and routes to pre-init.
	fixture.page().apply_snapshot(not_initialized_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	require_preinit_stage(fixture.page());
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));
}

TEST_CASE("Network code exit pending clears into paired on a ready "
	  "snapshot",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 1);
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	// The session coming up anyway releases the pending into paired.
	fixture.page().apply_snapshot(ready_snapshot(), 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Paired);
	require_paired_stage(fixture.page());
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit_button(fixture.page())));
	CHECK(ip_row_value_text(fixture.page()) == "100.64.0.1");
}

TEST_CASE("Network code exit failure shows the message with the bundled "
	  "idle exit",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(fixture.exit_count == 1);
	REQUIRE(has_pending_arc(exit));

	// An ok result does not release the pending; only the confirming
	// snapshot, a failure result, or the watchdog does.
	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit, true,
					   "", 1);
	fixture.refresh_layout();
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_action_result(ui::TailscaleActionKind::Exit,
					   false, "FAIL", 2);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit));
	// The failure message owns the label and the bundled exit stays
	// visible and idle, mirroring the paired stage's message hold.
	CHECK(code_text(fixture.page()) == "FAIL");
	CHECK_FALSE(hidden(code_label(fixture.page())));
	CHECK_FALSE(hidden(exit));
	CHECK_FALSE(spinner_visible(fixture.page()));

	// The message hold suppresses backward snapshot re-routing.
	fixture.page().apply_snapshot(not_initialized_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "FAIL");

	fixture.page().apply_snapshot(client_init_snapshot(), 4);
	fixture.refresh_layout();
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK_FALSE(hidden(exit_button(fixture.page())));
}

TEST_CASE("Network code exit watchdog unlocks a never-confirmed exit",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(fixture.exit_count == 1);
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.lvgl().advance(START_PENDING_TIMEOUT_MS - 1);
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	CHECK(has_pending_arc(exit));
	fixture.lvgl().advance(1);
	fixture.refresh_layout();
	CHECK(clickable(back_button(fixture.page())));

	// The watchdog re-routes from the cached snapshot: still client-init,
	// so the code stage comes back with the code and an idle exit.
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(code_text(fixture.page()) == "123 456");
	CHECK_FALSE(hidden(exit));
	CHECK_FALSE(has_pending_arc(exit));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == 2);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network code exit pending clears on pause, exit, enter, and "
	  "recreate",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE_FALSE(clickable(back_button(fixture.page())));

	fixture.page().on_pause();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	CHECK(clickable(back_button(fixture.page())));
	CHECK_FALSE(has_pending_arc(exit));

	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.page().on_exit();
	fixture.page().on_enter();
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK_FALSE(clickable(back_button(fixture.page())));
	fixture.destroy_page();
	fixture.recreate_page();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(clickable(back_button(fixture.page())));
	CHECK(hidden(exit_button(fixture.page())));

	// The exit lane is rewired after recreate.
	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	const int exit_before = fixture.exit_count;
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	CHECK(fixture.exit_count == exit_before + 1);
	CHECK_FALSE(clickable(back_button(fixture.page())));
}

TEST_CASE("Network code pending exit stays visible when a late empty code "
	  "arrives",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();

	lv_obj_t *exit = exit_button(fixture.page());
	REQUIRE(exit != nullptr);
	fixture.complete_exit_hold();
	fixture.refresh_layout();
	REQUIRE(fixture.exit_count == 1);
	REQUIRE(has_pending_arc(exit));

	// The bundling rule must not hide a pending exit: the pending arc is
	// the only progress signal while the label is withdrawn.
	fixture.page().apply_code("", 2);
	fixture.refresh_layout();
	CHECK(hidden(code_label(fixture.page())));
	CHECK(spinner_visible(fixture.page()));
	CHECK_FALSE(hidden(exit));
	CHECK(has_pending_arc(exit));
	CHECK_FALSE(clickable(back_button(fixture.page())));

	fixture.page().apply_snapshot(not_initialized_snapshot(), 3);
	fixture.refresh_layout();
	CHECK(fixture.page().stage() == ui::NetworkPage::Stage::PreInit);
	CHECK(hidden(exit_button(fixture.page())));
	CHECK(clickable(back_button(fixture.page())));
}

TEST_CASE("Network code content centers between the title and exit buttons",
	  "[ui][network][code]")
{
	NetworkCodeFixture fixture;

	fixture.page().apply_snapshot(client_init_snapshot(), 1);
	fixture.page().apply_code("123456", 1);
	fixture.refresh_layout();
	REQUIRE(fixture.page().stage() == ui::NetworkPage::Stage::Code);

	lv_obj_t *title = back_button(fixture.page());
	lv_obj_t *exit = exit_button(fixture.page());
	lv_obj_t *stage = code_stage(fixture.page());
	lv_obj_t *label = code_label(fixture.page());
	REQUIRE(title != nullptr);
	REQUIRE(exit != nullptr);
	REQUIRE(stage != nullptr);
	REQUIRE(label != nullptr);
	REQUIRE_FALSE(hidden(exit));
	REQUIRE_FALSE(hidden(label));

	// The live band: title bottom edge to exit top edge.
	const lv_area_t title_coordinates = object_coords(title);
	const lv_area_t exit_coordinates = object_coords(exit);
	CHECK(title_coordinates.y2 + 1 == TITLE_BOTTOM_EDGE);
	CHECK(exit_coordinates.y1 == EXIT_TOP);
	const int32_t band_midpoint =
		(title_coordinates.y2 + exit_coordinates.y1) / 2;
	CHECK(band_midpoint == CONTENT_BAND_CENTER_Y);

	// The stage straddles the band symmetrically: twice its continuous
	// center equals twice the band's continuous center (240 == 240).
	const lv_area_t stage_coordinates = object_coords(stage);
	CHECK(stage_coordinates.y1 == TITLE_BOTTOM_EDGE - 32);
	CHECK(stage_coordinates.y2 + 1 == EXIT_TOP + 32);
	CHECK(stage_coordinates.y1 + stage_coordinates.y2 + 1 ==
	      TITLE_BOTTOM_EDGE + EXIT_TOP);

	// The visible content's integer pixel midpoint matches the band's.
	const lv_area_t label_coordinates = object_coords(label);
	CHECK((label_coordinates.y1 + label_coordinates.y2) / 2 ==
	      band_midpoint);

	// The loading arc owns the same midpoint while the code is loading.
	fixture.page().apply_code("", 2);
	fixture.refresh_layout();
	REQUIRE(spinner_visible(fixture.page()));
	CHECK(hidden(exit_button(fixture.page())));
	lv_obj_t *spinner = code_spinner(fixture.page());
	REQUIRE(spinner != nullptr);
	const lv_area_t spinner_coordinates = object_coords(spinner);
	CHECK((spinner_coordinates.y1 + spinner_coordinates.y2) / 2 ==
	      CONTENT_BAND_CENTER_Y);
}

TEST_CASE("Network code stage keeps the exit button hidden by default",
	  "[ui][network][code]")
{
	// Deliberately bypasses NetworkCodeFixture: the fixture opts the gate
	// in to cover the enabled path, so this guard owns a raw page and
	// locks the shipped default (hidden) while the exit lane cannot tear
	// down client-init sessions.
	test_support::LvglTestContext lvgl;
	ui::NetworkPage page;
	page.on_start_request = []() -> bool {
		return true;
	};
	page.on_exit_request = []() -> bool {
		return true;
	};
	page.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());

	// Loading: the spinner runs and the gated exit stays hidden.
	page.apply_snapshot(client_init_snapshot(), 1);
	lv_obj_update_layout(lvgl.screen());
	REQUIRE(page.stage() == ui::NetworkPage::Stage::Code);
	CHECK(spinner_visible(page));
	CHECK(hidden(exit_button(page)));

	// A delivered code shows the label but not the gated exit.
	page.apply_code("123456", 1);
	lv_obj_update_layout(lvgl.screen());
	CHECK(code_text(page) == "123 456");
	CHECK_FALSE(hidden(code_label(page)));
	CHECK(hidden(exit_button(page)));

	// A held action failure message also leaves the gated exit hidden.
	page.on_exit();
	page.on_enter();
	lv_obj_update_layout(lvgl.screen());
	lv_obj_t *on = on_button(page);
	REQUIRE(on != nullptr);
	constexpr uint32_t fill_ms =
		ON_HOLD_DURATION_MS - ui::HoldConfirmBehavior::FULL_DWELL_MS;
	lvgl.press(on);
	lvgl.advance(fill_ms);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();
	lv_obj_update_layout(lvgl.screen());
	REQUIRE(page.stage() == ui::NetworkPage::Stage::Terms);
	lv_obj_t *agree = agree_button(page);
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	lv_obj_update_layout(lvgl.screen());
	REQUIRE(page.stage() == ui::NetworkPage::Stage::Code);
	page.apply_action_result(ui::TailscaleActionKind::Start, false,
				 "FAIL", 1);
	lv_obj_update_layout(lvgl.screen());
	CHECK(code_text(page) == "FAIL");
	CHECK_FALSE(hidden(code_label(page)));
	CHECK(hidden(exit_button(page)));

	page.destroy();
}
