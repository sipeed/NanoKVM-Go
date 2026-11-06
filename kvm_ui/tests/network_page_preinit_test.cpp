#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string_view>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"
#include "ui/scona.hpp"

extern "C" {
LV_IMAGE_DECLARE(settings_tailscale);
}

namespace
{

constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
constexpr uint32_t ON_FILL_COLOR = 0x00838F;

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

lv_obj_t *decline_button(const ui::Page &page)
{
	lv_obj_t *row = terms_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) <= 1) {
		return nullptr;
	}
	return lv_obj_get_child(row, 1);
}

lv_obj_t *group_row(lv_obj_t *group, uint32_t index)
{
	if (group == nullptr || lv_obj_get_child_count(group) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(group, index);
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

lv_obj_t *find_descendant_of_type(lv_obj_t *parent,
				  const lv_obj_class_t *type)
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

std::string_view object_text(lv_obj_t *object)
{
	return test_support::visible_direct_text(object);
}

class NetworkPageFixture {
    public:
	NetworkPageFixture()
	{
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	~NetworkPageFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkPageFixture(const NetworkPageFixture &) = delete;
	NetworkPageFixture &operator=(const NetworkPageFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::NetworkPage &page()
	{
		return _page;
	}

	lv_obj_t *group() const
	{
		return _group;
	}

	lv_obj_t *on() const
	{
		return _on;
	}

	std::string_view on_text() const
	{
		return object_text(_on);
	}

	void destroy_page()
	{
		_page.destroy();
		_group = nullptr;
		_on = nullptr;
	}

	void recreate_page()
	{
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	void begin_hold()
	{
		_lvgl.press(_on);
	}

	void complete_hold()
	{
		constexpr uint32_t fill_ms =
			ON_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(_on);
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	}

    private:
	void refresh_objects()
	{
		lv_obj_update_layout(_lvgl.screen());
		_group = preinit_group(_page);
		_on = on_button(_page);
	}

	test_support::LvglTestContext _lvgl;
	ui::NetworkPage _page;
	lv_obj_t *_group = nullptr;
	lv_obj_t *_on = nullptr;
};

void require_preinit_stage(NetworkPageFixture &fixture)
{
	REQUIRE(back_button(fixture.page()) != nullptr);
	REQUIRE(fixture.group() != nullptr);
	REQUIRE(fixture.on() != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll_content(fixture.page())) == 8);
	REQUIRE(lv_obj_get_child_count(fixture.group()) == 2);
	REQUIRE(terms_row(fixture.page()) != nullptr);
	REQUIRE(terms_body(fixture.page()) != nullptr);
	CHECK(lv_obj_has_flag(terms_row(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_body(fixture.page()), LV_OBJ_FLAG_HIDDEN));
}

}

TEST_CASE("Network pre-init stage keeps its fixed layout and idle visuals",
	  "[ui][network][preinit]")
{
	NetworkPageFixture fixture;
	require_preinit_stage(fixture);

	lv_obj_t *sipeed = group_row(fixture.group(), 0);
	lv_obj_t *remote = group_row(fixture.group(), 1);
	REQUIRE(sipeed != nullptr);
	REQUIRE(remote != nullptr);

	CHECK(lv_obj_get_width(sipeed) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(sipeed) == ui::SConA::HEIGHT);
	CHECK(lv_obj_get_width(remote) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(remote) == ui::SConA::HEIGHT);

	const lv_area_t sipeed_coordinates = object_coords(sipeed);
	CHECK(sipeed_coordinates.x1 == 13);
	CHECK(sipeed_coordinates.y1 == 76);
	const lv_area_t remote_coordinates = object_coords(remote);
	CHECK(remote_coordinates.x1 == 13);
	CHECK(remote_coordinates.y1 == 108);
	CHECK(lv_obj_get_style_margin_bottom(remote, LV_PART_MAIN) == 4);

	CHECK(object_text(sipeed) == "Sipeed");
	CHECK(object_text(remote) == "Remote Network");

	lv_obj_t *sipeed_label =
		test_support::find_visible_direct_label(sipeed);
	lv_obj_t *remote_label =
		test_support::find_visible_direct_label(remote);
	REQUIRE(sipeed_label != nullptr);
	REQUIRE(remote_label != nullptr);
	const lv_area_t sipeed_label_coordinates = object_coords(sipeed_label);
	const lv_area_t remote_label_coordinates = object_coords(remote_label);
	const int32_t row_center = 13 + ui::SConA::WIDTH / 2;
	const int32_t sipeed_label_center =
		(sipeed_label_coordinates.x1 + sipeed_label_coordinates.x2) / 2;
	const int32_t remote_label_center =
		(remote_label_coordinates.x1 + remote_label_coordinates.x2) / 2;
	CHECK(sipeed_label_center >= row_center - 1);
	CHECK(sipeed_label_center <= row_center + 1);
	CHECK(remote_label_center >= row_center - 1);
	CHECK(remote_label_center <= row_center + 1);

	lv_obj_t *on = fixture.on();
	CHECK_FALSE(lv_obj_has_flag(on, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_width(on) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(on) == ui::BConA::HEIGHT);
	CHECK(lv_obj_get_style_radius(on, LV_PART_MAIN) == ui::BConA::RADIUS);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(on, LV_PART_MAIN),
			  lv_color_hex(ui::BConA::COLOR_DEFAULT)));
	CHECK(lv_obj_get_style_pad_left(on, LV_PART_MAIN) ==
	      ui::BConA::PADDING_LEFT);
	CHECK(lv_obj_get_style_pad_right(on, LV_PART_MAIN) == 17);

	const lv_area_t on_coordinates = object_coords(on);
	CHECK(on_coordinates.x1 == 13);
	CHECK(on_coordinates.y1 == 164);
	CHECK(fixture.on_text() == "ON");

	lv_obj_t *on_image = button_image(on);
	REQUIRE(on_image != nullptr);
	CHECK(lv_image_get_src(on_image) == &settings_tailscale);
}

TEST_CASE("Network ON gates clicks and shows the accent fill while holding",
	  "[ui][network][preinit][hold-confirm]")
{
	NetworkPageFixture fixture;
	require_preinit_stage(fixture);

	lv_obj_send_event(fixture.on(), LV_EVENT_SHORT_CLICKED, nullptr);
	lv_obj_send_event(fixture.on(), LV_EVENT_CLICKED, nullptr);
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);

	fixture.begin_hold();
	CHECK(progress_width(fixture.on()) >=
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(1000);
	CHECK(fixture.on_text() == "Keep");
	CHECK(progress_width(fixture.on()) > 0);
	lv_obj_t *surface = progress_surface(fixture.on());
	REQUIRE(surface != nullptr);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(ON_FILL_COLOR)));

	fixture.lvgl().release();
	fixture.lvgl().advance(100);
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);
}

TEST_CASE("Network ON completes a full hold and enters the terms stage",
	  "[ui][network][preinit][hold-confirm]")
{
	NetworkPageFixture fixture;
	require_preinit_stage(fixture);
	constexpr uint32_t fill_ms =
		ON_HOLD_DURATION_MS - ui::HoldConfirmBehavior::FULL_DWELL_MS;

	fixture.begin_hold();
	CHECK(progress_width(fixture.on()) ==
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(fill_ms - 1);
	CHECK(fixture.on_text() == "Keep");
	CHECK(progress_width(fixture.on()) < ui::BConA::WIDTH);
	fixture.lvgl().advance(1);
	CHECK(progress_width(fixture.on()) == ui::BConA::WIDTH);
	CHECK(fixture.on_text() == "Keep");
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS - 1);
	CHECK(fixture.on_text() == "Keep");
	fixture.lvgl().advance(1);

	CHECK(fixture.page().is_created());
	CHECK(lv_obj_has_flag(back_button(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(fixture.group(), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(fixture.on(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(
		lv_obj_has_flag(terms_row(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(
		lv_obj_has_flag(terms_body(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	fixture.lvgl().release();

	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	lv_obj_send_event(decline, LV_EVENT_CLICKED, nullptr);
	CHECK_FALSE(
		lv_obj_has_flag(back_button(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(fixture.group(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(fixture.on(), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_row(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_body(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);

	fixture.complete_hold();
	CHECK_FALSE(
		lv_obj_has_flag(terms_row(fixture.page()), LV_OBJ_FLAG_HIDDEN));
	fixture.lvgl().release();
}

TEST_CASE("Network pre-init cancels the hold on pause, exit, and destroy",
	  "[ui][network][preinit][hold-confirm]")
{
	NetworkPageFixture fixture;
	require_preinit_stage(fixture);

	fixture.begin_hold();
	fixture.lvgl().advance(1000);
	CHECK(fixture.on_text() == "Keep");
	fixture.page().on_pause();
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);
	fixture.lvgl().release();

	fixture.begin_hold();
	fixture.lvgl().advance(1000);
	CHECK(fixture.on_text() == "Keep");
	fixture.page().on_exit();
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);
	fixture.lvgl().release();

	fixture.begin_hold();
	fixture.lvgl().advance(1000);
	CHECK(progress_width(fixture.on()) > 0);
	fixture.destroy_page();
	CHECK_FALSE(fixture.page().is_created());

	fixture.recreate_page();
	require_preinit_stage(fixture);
	CHECK(fixture.on_text() == "ON");
	CHECK(progress_width(fixture.on()) == 0);
}
