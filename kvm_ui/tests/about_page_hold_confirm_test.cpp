#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>
#include <string_view>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/about_page.hpp"
#include "ui/bcona.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/wifi_page.hpp"

namespace
{

constexpr uint32_t RESET_HOLD_DURATION_MS = 2000;
constexpr uint32_t RESET_FILL_COLOR = 0xFB2C36;

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *reset_button(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 3) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 2);
}

lv_obj_t *back_button(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 0);
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

class AboutPageFixture {
    public:
	AboutPageFixture()
	{
		bind_reset_callback();
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	~AboutPageFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	AboutPageFixture(const AboutPageFixture &) = delete;
	AboutPageFixture &operator=(const AboutPageFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::AboutPage &page()
	{
		return _page;
	}

	lv_obj_t *reset() const
	{
		return _reset;
	}

	lv_obj_t *back() const
	{
		return _back;
	}

	lv_obj_t *back_image() const
	{
		return button_image(_back);
	}

	std::string_view reset_text() const
	{
		return test_support::visible_direct_text(_reset);
	}

	void destroy_page()
	{
		_page.destroy();
		_back = nullptr;
		_reset = nullptr;
	}

	void recreate_page()
	{
		bind_reset_callback();
		_page.create(_lvgl.screen());
		refresh_objects();
	}

	int reset_callback_count() const
	{
		return _reset_callback_count;
	}

	int back_callback_count() const
	{
		return _back_callback_count;
	}

	void set_reset_request_accepted(bool accepted)
	{
		_reset_request_accepted = accepted;
	}

	void complete_hold()
	{
		constexpr uint32_t fill_ms =
			RESET_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(_reset);
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	}

    private:
	void bind_reset_callback()
	{
		_page.on_reset_click = [this]() -> bool {
			++_reset_callback_count;
			return _reset_request_accepted;
		};
		_page.on_back_click = [this]() {
			++_back_callback_count;
			_page.on_exit();
		};
	}

	void refresh_objects()
	{
		lv_obj_update_layout(_lvgl.screen());
		_back = back_button(_page);
		_reset = reset_button(_page);
	}

	test_support::LvglTestContext _lvgl;
	ui::AboutPage _page;
	lv_obj_t *_back = nullptr;
	lv_obj_t *_reset = nullptr;
	bool _reset_request_accepted = true;
	int _reset_callback_count = 0;
	int _back_callback_count = 0;
};

void require_about_reset(AboutPageFixture &fixture)
{
	REQUIRE(fixture.back() != nullptr);
	REQUIRE(fixture.reset() != nullptr);
	REQUIRE(scroll_content(fixture.page()) != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll_content(fixture.page())) == 3);
}

} // namespace

TEST_CASE("About Reset keeps its fixed layout and idle visuals",
	  "[ui][about][hold-confirm]")
{
	test_support::LvglTestContext lvgl;
	ui::AboutPage about_page;
	about_page.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *about_reset = reset_button(about_page);
	REQUIRE(about_reset != nullptr);

	CHECK_FALSE(lv_obj_has_flag(about_reset, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_width(about_reset) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(about_reset) == ui::BConA::HEIGHT);

	const lv_area_t about_coordinates = object_coords(about_reset);
	CHECK(about_coordinates.x1 == 13);
	CHECK(about_coordinates.y1 == 164);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(about_reset, LV_PART_MAIN),
				  lv_color_hex(ui::BConA::COLOR_DEFAULT)));
	CHECK(lv_obj_get_style_radius(about_reset, LV_PART_MAIN) ==
	      ui::BConA::RADIUS);
	CHECK(test_support::visible_direct_text(about_reset) == "Reset");

	lv_obj_t *about_label =
		test_support::find_visible_direct_label(about_reset);
	REQUIRE(about_label != nullptr);
	CHECK(lv_obj_get_style_text_font(about_label, LV_PART_MAIN) != nullptr);

	lv_obj_t *about_image = button_image(about_reset);
	REQUIRE(about_image != nullptr);

	about_page.destroy();
}

TEST_CASE("About Reset gates clicks and cancels a partial hold",
	  "[ui][about][hold-confirm]")
{
	AboutPageFixture fixture;
	require_about_reset(fixture);
	lv_obj_t *back_image = fixture.back_image();
	REQUIRE(back_image != nullptr);

	lv_obj_send_event(fixture.reset(), LV_EVENT_SHORT_CLICKED, nullptr);
	lv_obj_send_event(fixture.reset(), LV_EVENT_CLICKED, nullptr);
	CHECK(fixture.reset_callback_count() == 0);
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.reset()));

	fixture.lvgl().press(fixture.reset());
	CHECK(progress_width(fixture.reset()) >=
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(1000);
	CHECK(fixture.reset_text() == "Keep");
	CHECK(progress_width(fixture.reset()) > 0);
	lv_obj_t *surface = progress_surface(fixture.reset());
	REQUIRE(surface != nullptr);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(surface, LV_PART_MAIN),
			  lv_color_hex(RESET_FILL_COLOR)));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));

	fixture.lvgl().press(fixture.back());
	fixture.lvgl().release();
	CHECK(fixture.back_callback_count() == 1);
	CHECK(fixture.reset_callback_count() == 0);
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK(is_clickable(fixture.reset()));
	CHECK_FALSE(has_pending_arc(fixture.reset()));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
}

TEST_CASE("Accepted About Reset stays loading and locks navigation",
	  "[ui][about][hold-confirm]")
{
	AboutPageFixture fixture;
	require_about_reset(fixture);
	lv_obj_t *back_image = fixture.back_image();
	REQUIRE(back_image != nullptr);
	const lv_area_t original_back_coordinates =
		object_coords(fixture.back());
	const int32_t original_back_width = lv_obj_get_width(fixture.back());
	const int32_t original_back_height = lv_obj_get_height(fixture.back());
	const std::string_view original_back_title =
		test_support::visible_direct_text(fixture.back());
	constexpr uint32_t fill_ms =
		RESET_HOLD_DURATION_MS - ui::HoldConfirmBehavior::FULL_DWELL_MS;

	fixture.lvgl().press(fixture.reset());
	CHECK(progress_width(fixture.reset()) ==
	      ui::HoldConfirmBehavior::DISCOVERY_ANCHOR_WIDTH);
	fixture.lvgl().advance(fill_ms - 1);
	CHECK(fixture.reset_text() == "Keep");
	CHECK(progress_width(fixture.reset()) < ui::BConA::WIDTH);
	fixture.lvgl().advance(1);
	CHECK(progress_width(fixture.reset()) == ui::BConA::WIDTH);
	CHECK(fixture.reset_text() == "Keep");
	fixture.lvgl().advance(ui::HoldConfirmBehavior::FULL_DWELL_MS - 1);
	CHECK(fixture.reset_text() == "Keep");
	CHECK(fixture.reset_callback_count() == 0);
	fixture.lvgl().advance(1);

	CHECK(fixture.reset_callback_count() == 1);
	CHECK(fixture.page().is_created());
	CHECK(fixture.page().is_factory_reset_pending());
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(is_clickable(fixture.reset()));
	CHECK_FALSE(is_clickable(fixture.back()));
	CHECK(has_pending_arc(fixture.reset()));
	CHECK(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_width(fixture.back()) == original_back_width);
	CHECK(lv_obj_get_height(fixture.back()) == original_back_height);
	CHECK(object_coords(fixture.back()).x1 == original_back_coordinates.x1);
	CHECK(object_coords(fixture.back()).y1 == original_back_coordinates.y1);
	CHECK(test_support::visible_direct_text(fixture.back()) ==
	      original_back_title);

	fixture.lvgl().release();
	fixture.lvgl().advance(ui::BConA::PENDING_TIMEOUT_MS + 100);
	CHECK(fixture.page().is_factory_reset_pending());
	CHECK(has_pending_arc(fixture.reset()));
	CHECK_FALSE(is_clickable(fixture.reset()));
	CHECK_FALSE(is_clickable(fixture.back()));

	fixture.complete_hold();
	CHECK(fixture.reset_callback_count() == 1);
	fixture.lvgl().release();
	fixture.lvgl().press(fixture.back());
	fixture.lvgl().release();
	CHECK(fixture.back_callback_count() == 0);

	fixture.page().set_factory_reset_pending(false);
	CHECK_FALSE(fixture.page().is_factory_reset_pending());
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.reset()));
	CHECK(is_clickable(fixture.reset()));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
	fixture.lvgl().press(fixture.back());
	fixture.lvgl().release();
	CHECK(fixture.back_callback_count() == 1);
}

TEST_CASE("Rejected About Reset request remains idle and unlocked",
	  "[ui][about][hold-confirm]")
{
	AboutPageFixture fixture;
	fixture.set_reset_request_accepted(false);
	require_about_reset(fixture);
	lv_obj_t *back_image = fixture.back_image();
	REQUIRE(back_image != nullptr);

	fixture.complete_hold();
	CHECK(fixture.reset_callback_count() == 1);
	CHECK_FALSE(fixture.page().is_factory_reset_pending());
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.reset()));
	CHECK(is_clickable(fixture.reset()));
	CHECK(is_clickable(fixture.back()));
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));
	fixture.lvgl().release();
}

TEST_CASE("About page lifecycle cancels an unfinished Reset hold",
	  "[ui][about][hold-confirm]")
{
	AboutPageFixture fixture;
	require_about_reset(fixture);

	fixture.lvgl().press(fixture.reset());
	fixture.lvgl().advance(1000);
	REQUIRE(fixture.reset_text() == "Keep");

	SECTION("pause")
	{
		fixture.page().on_pause();
	}

	SECTION("exit")
	{
		fixture.page().on_exit();
	}

	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK(is_clickable(fixture.reset()));
	CHECK_FALSE(has_pending_arc(fixture.reset()));

	fixture.lvgl().release();
	fixture.lvgl().advance(RESET_HOLD_DURATION_MS);
	CHECK(fixture.reset_callback_count() == 0);
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
}

TEST_CASE("Destroying and recreating About clears factory reset loading",
	  "[ui][about][hold-confirm]")
{
	AboutPageFixture fixture;
	require_about_reset(fixture);
	const std::size_t initial_timer_count = fixture.lvgl().timer_count();

	fixture.complete_hold();
	REQUIRE(fixture.page().is_factory_reset_pending());
	REQUIRE(has_pending_arc(fixture.reset()));
	fixture.lvgl().release();
	fixture.destroy_page();
	fixture.lvgl().advance(RESET_HOLD_DURATION_MS);

	CHECK(fixture.reset_callback_count() == 1);
	CHECK_FALSE(fixture.page().is_created());
	CHECK(fixture.lvgl().timer_count() <= initial_timer_count);

	fixture.recreate_page();
	require_about_reset(fixture);
	CHECK_FALSE(lv_obj_has_flag(fixture.reset(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(fixture.page().is_factory_reset_pending());
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.reset()));
	CHECK(is_clickable(fixture.reset()));
	CHECK(is_clickable(fixture.back()));
	lv_obj_t *back_image = fixture.back_image();
	REQUIRE(back_image != nullptr);
	CHECK_FALSE(lv_obj_has_flag(back_image, LV_OBJ_FLAG_HIDDEN));

	fixture.set_reset_request_accepted(false);
	fixture.complete_hold();
	CHECK(fixture.reset_callback_count() == 2);
	CHECK(fixture.reset_text() == "Reset");
	CHECK(progress_width(fixture.reset()) == 0);
	CHECK_FALSE(has_pending_arc(fixture.reset()));
	fixture.lvgl().release();
}
