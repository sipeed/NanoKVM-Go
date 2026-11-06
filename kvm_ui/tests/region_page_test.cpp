#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/region_page.hpp"
#include "ui/scona.hpp"

namespace
{

constexpr uint32_t SWITCH_HOLD_DURATION_MS = 2000;
constexpr int32_t REGION_VALUE_TOP = 104;
constexpr int32_t SWITCH_TOP = 164;

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *child_at(const ui::Page &page, uint32_t index)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, index);
}

lv_obj_t *back_button(const ui::Page &page)
{
	return child_at(page, 0);
}

lv_obj_t *region_value_row(const ui::Page &page)
{
	return child_at(page, 1);
}

lv_obj_t *switch_button(const ui::Page &page)
{
	return child_at(page, 2);
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

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

lv_obj_t *button_image(lv_obj_t *button)
{
	return find_descendant_of_type(button, &lv_image_class);
}

lv_obj_t *progress_clip(lv_obj_t *button)
{
	if (button == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(button);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(button, index);
		if (lv_obj_has_flag(child, LV_OBJ_FLAG_IGNORE_LAYOUT) &&
		    lv_obj_get_style_height(child, LV_PART_MAIN) ==
			    ui::BConA::HEIGHT &&
		    lv_obj_get_style_bg_opa(child, LV_PART_MAIN) ==
			    LV_OPA_TRANSP) {
			return child;
		}
	}

	return nullptr;
}

int32_t progress_width(lv_obj_t *button)
{
	lv_obj_t *clip = progress_clip(button);
	return clip == nullptr ? 0 : lv_obj_get_style_width(clip, LV_PART_MAIN);
}

lv_obj_t *progress_surface(lv_obj_t *button)
{
	lv_obj_t *clip = progress_clip(button);
	if (clip == nullptr || lv_obj_get_child_count(clip) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(clip, 0);
}

bool has_pending_arc(lv_obj_t *button)
{
	return test_support::find_visible_descendant_of_type(
		       button, &lv_arc_class) != nullptr;
}

bool is_clickable(lv_obj_t *button)
{
	return button != nullptr &&
	       lv_obj_has_flag(button, LV_OBJ_FLAG_CLICKABLE);
}

class RegionPageFixture {
    public:
	RegionPageFixture()
	{
		bind_switch_callback();
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	~RegionPageFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	RegionPageFixture(const RegionPageFixture &) = delete;
	RegionPageFixture &operator=(const RegionPageFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::RegionPage &page()
	{
		return _page;
	}

	lv_obj_t *back() const
	{
		return _back;
	}

	lv_obj_t *value_row() const
	{
		return _value_row;
	}

	lv_obj_t *switch_button() const
	{
		return _switch;
	}

	std::string_view value_text() const
	{
		return test_support::visible_direct_text(_value_row);
	}

	std::string_view switch_text() const
	{
		return test_support::visible_direct_text(_switch);
	}

	void destroy_page()
	{
		_page.destroy();
		_back = nullptr;
		_value_row = nullptr;
		_switch = nullptr;
	}

	void recreate_page()
	{
		bind_switch_callback();
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	void set_switch_request_accepted(bool accepted)
	{
		_switch_request_accepted = accepted;
	}

	int switch_callback_count() const
	{
		return _switch_callback_count;
	}

	std::vector<ui::RegionOption> requested_targets() const
	{
		return _requested_targets;
	}

	void complete_hold()
	{
		constexpr uint32_t fill_ms =
			SWITCH_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(_switch);
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	}

    private:
	void bind_switch_callback()
	{
		_page.on_switch_request = [this](ui::RegionOption target) -> bool {
			++_switch_callback_count;
			_requested_targets.push_back(target);
			return _switch_request_accepted;
		};
	}

	void refresh_objects()
	{
		lv_obj_update_layout(_lvgl.screen());
		_back = back_button(_page);
		_value_row = region_value_row(_page);
		_switch = ::switch_button(_page);
	}

	test_support::LvglTestContext _lvgl;
	ui::RegionPage _page;
	lv_obj_t *_back = nullptr;
	lv_obj_t *_value_row = nullptr;
	lv_obj_t *_switch = nullptr;
	bool _switch_request_accepted = true;
	int _switch_callback_count = 0;
	std::vector<ui::RegionOption> _requested_targets;
};

void require_region_page(RegionPageFixture &fixture)
{
	REQUIRE(fixture.back() != nullptr);
	REQUIRE(fixture.value_row() != nullptr);
	REQUIRE(fixture.switch_button() != nullptr);
	REQUIRE(scroll_content(fixture.page()) != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll_content(fixture.page())) == 3);
}

} // namespace

TEST_CASE("Region page shows the projected region with the About-style layout",
	  "[ui][region][hold-confirm]")
{
	test_support::LvglTestContext lvgl;
	ui::RegionPage page;
	page.create(lvgl.screen());
	page.set_region(ui::RegionOption::ChinaRegion);
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *value_row = region_value_row(page);
	lv_obj_t *switch_btn = switch_button(page);
	REQUIRE(value_row != nullptr);
	REQUIRE(switch_btn != nullptr);

	CHECK(test_support::visible_direct_text(value_row) == "CN");
	CHECK(lv_obj_get_width(value_row) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(value_row) == ui::SConA::HEIGHT);
	const lv_area_t value_coordinates = object_coords(value_row);
	CHECK(value_coordinates.x1 == 13);
	CHECK(value_coordinates.y1 == REGION_VALUE_TOP);

	CHECK(lv_obj_get_width(switch_btn) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(switch_btn) == ui::BConA::HEIGHT);
	const lv_area_t switch_coordinates = object_coords(switch_btn);
	CHECK(switch_coordinates.x1 == 13);
	CHECK(switch_coordinates.y1 == SWITCH_TOP);
	CHECK(test_support::visible_direct_text(switch_btn) == "Switch");
	CHECK(button_image(switch_btn) != nullptr);
	CHECK(progress_width(switch_btn) == 0);
	CHECK_FALSE(has_pending_arc(switch_btn));
	CHECK(is_clickable(switch_btn));
	CHECK(is_clickable(back_button(page)));

	page.set_region(ui::RegionOption::Global);
	CHECK(test_support::visible_direct_text(value_row) == "Global");
	page.set_region(ui::RegionOption::Global);
	CHECK(test_support::visible_direct_text(value_row) == "Global");

	page.destroy();
}

TEST_CASE("Region Switch ignores clicks and requires the full 2000ms hold",
	  "[ui][region][hold-confirm]")
{
	RegionPageFixture fixture;
	require_region_page(fixture);

	CHECK(fixture.value_text() == "Global");

	lv_obj_send_event(fixture.switch_button(), LV_EVENT_SHORT_CLICKED,
			  nullptr);
	lv_obj_send_event(fixture.switch_button(), LV_EVENT_CLICKED, nullptr);
	CHECK(fixture.switch_callback_count() == 0);
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.switch_button()));

	fixture.lvgl().press(fixture.switch_button());
	CHECK(progress_width(fixture.switch_button()) >=
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(1000);
	CHECK(fixture.switch_text() == "Keep");
	lv_obj_t *surface = progress_surface(fixture.switch_button());
	REQUIRE(surface != nullptr);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(ui::RegionPage::SWITCH_FILL_COLOR)));
	CHECK(fixture.switch_callback_count() == 0);
	fixture.lvgl().release();
	CHECK(fixture.switch_callback_count() == 0);
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);

	fixture.complete_hold();
	CHECK(fixture.switch_callback_count() == 1);
	CHECK(fixture.requested_targets() ==
	      std::vector<ui::RegionOption> { ui::RegionOption::ChinaRegion });
	CHECK(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "CN");
	fixture.lvgl().release();
}

TEST_CASE("Accepted Region switch stays pending and replays the projection",
	  "[ui][region][hold-confirm]")
{
	RegionPageFixture fixture;
	require_region_page(fixture);
	lv_obj_t *back_image = button_image(fixture.back());
	REQUIRE(back_image != nullptr);
	constexpr uint32_t fill_ms =
		SWITCH_HOLD_DURATION_MS - ui::HoldConfirmBehavior::FULL_DWELL_MS;

	CHECK(fixture.value_text() == "Global");
	fixture.lvgl().press(fixture.switch_button());
	fixture.lvgl().advance(fill_ms - 1);
	CHECK(fixture.switch_text() == "Keep");
	CHECK(progress_width(fixture.switch_button()) < ui::BConA::WIDTH);
	fixture.lvgl().advance(1);
	CHECK(progress_width(fixture.switch_button()) == ui::BConA::WIDTH);
	CHECK(fixture.switch_callback_count() == 0);
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS - 1);
	CHECK(fixture.switch_callback_count() == 0);
	fixture.lvgl().advance(1);

	CHECK(fixture.switch_callback_count() == 1);
	CHECK(fixture.requested_targets() ==
	      std::vector<ui::RegionOption> { ui::RegionOption::ChinaRegion });
	CHECK(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "CN");
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK_FALSE(is_clickable(fixture.switch_button()));
	CHECK_FALSE(is_clickable(fixture.back()));
	CHECK(has_pending_arc(fixture.switch_button()));
	CHECK(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));

	fixture.page().set_region(ui::RegionOption::Global);
	CHECK(fixture.value_text() == "CN");

	fixture.lvgl().release();
	fixture.lvgl().advance(ui::BConA::PENDING_TIMEOUT_MS + 100);
	CHECK(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "CN");
	CHECK(has_pending_arc(fixture.switch_button()));
	CHECK_FALSE(is_clickable(fixture.switch_button()));
	CHECK_FALSE(is_clickable(fixture.back()));

	fixture.page().clear_switch_pending();
	CHECK_FALSE(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "Global");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.switch_button()));
	CHECK(is_clickable(fixture.switch_button()));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
}

TEST_CASE("Rejected Region switch request remains idle and unlocked",
	  "[ui][region][hold-confirm]")
{
	RegionPageFixture fixture;
	fixture.set_switch_request_accepted(false);
	require_region_page(fixture);
	lv_obj_t *back_image = button_image(fixture.back());
	REQUIRE(back_image != nullptr);

	fixture.complete_hold();
	CHECK(fixture.switch_callback_count() == 1);
	CHECK(fixture.requested_targets() ==
	      std::vector<ui::RegionOption> { ui::RegionOption::ChinaRegion });
	CHECK_FALSE(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "Global");
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.switch_button()));
	CHECK(is_clickable(fixture.switch_button()));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
	fixture.lvgl().release();
}

TEST_CASE("Region page lifecycle cancels an unfinished switch hold",
	  "[ui][region][hold-confirm]")
{
	RegionPageFixture fixture;
	require_region_page(fixture);

	fixture.lvgl().press(fixture.switch_button());
	fixture.lvgl().advance(1000);
	REQUIRE(fixture.switch_text() == "Keep");

	SECTION("pause")
	{
		fixture.page().on_pause();
	}

	SECTION("exit")
	{
		fixture.page().on_exit();
	}

	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK(is_clickable(fixture.switch_button()));
	CHECK_FALSE(has_pending_arc(fixture.switch_button()));

	fixture.lvgl().release();
	fixture.lvgl().advance(SWITCH_HOLD_DURATION_MS);
	CHECK(fixture.switch_callback_count() == 0);
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK(fixture.value_text() == "Global");
}

TEST_CASE("Destroying and recreating Region clears switch loading",
	  "[ui][region][hold-confirm]")
{
	RegionPageFixture fixture;
	require_region_page(fixture);
	const std::size_t initial_timer_count = fixture.lvgl().timer_count();

	fixture.complete_hold();
	REQUIRE(fixture.page().is_switch_pending());
	REQUIRE(has_pending_arc(fixture.switch_button()));
	fixture.lvgl().release();
	fixture.destroy_page();
	fixture.lvgl().advance(SWITCH_HOLD_DURATION_MS);

	CHECK(fixture.switch_callback_count() == 1);
	CHECK_FALSE(fixture.page().is_created());
	CHECK(fixture.lvgl().timer_count() <= initial_timer_count);

	fixture.recreate_page();
	require_region_page(fixture);
	CHECK_FALSE(fixture.page().is_switch_pending());
	CHECK(fixture.value_text() == "Global");
	CHECK(fixture.switch_text() == "Switch");
	CHECK(progress_width(fixture.switch_button()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.switch_button()));
	CHECK(is_clickable(fixture.switch_button()));
	CHECK(is_clickable(fixture.back()));
	lv_obj_t *back_image = button_image(fixture.back());
	REQUIRE(back_image != nullptr);
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));

	fixture.set_switch_request_accepted(false);
	fixture.complete_hold();
	CHECK(fixture.switch_callback_count() == 2);
	CHECK(fixture.value_text() == "Global");
	CHECK_FALSE(fixture.page().is_switch_pending());
	fixture.lvgl().release();
}

TEST_CASE("Region page texts fit their containers", "[ui][region][layout]")
{
	test_support::LvglTestContext lvgl;
	ui::RegionPage page;
	page.create(lvgl.screen());
	page.set_region(ui::RegionOption::ChinaRegion);
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *title_label =
		test_support::find_visible_direct_label(back_button(page));
	REQUIRE(title_label != nullptr);
	CHECK(lv_text_get_width(
		      "Region", 6,
		      lv_obj_get_style_text_font(title_label, LV_PART_MAIN), 0) <=
	      lv_obj_get_width(title_label));

	lv_obj_t *switch_label =
		test_support::find_visible_direct_label(switch_button(page));
	REQUIRE(switch_label != nullptr);
	CHECK(lv_text_get_width(
		      "Switch", 6,
		      lv_obj_get_style_text_font(switch_label, LV_PART_MAIN),
		      0) <= lv_obj_get_width(switch_label));

	lv_obj_t *value_label =
		test_support::find_visible_direct_label(region_value_row(page));
	REQUIRE(value_label != nullptr);
	for (const ui::RegionOption region : { ui::RegionOption::ChinaRegion,
					       ui::RegionOption::Global }) {
		const std::string_view label = ui::region_option_label(region);
		CHECK(lv_text_get_width(
			      label.data(), label.size(),
			      lv_obj_get_style_text_font(value_label,
							 LV_PART_MAIN),
			      0) <= ui::SConA::WIDTH - 24);
	}

	page.destroy();
}
