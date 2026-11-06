#include <catch2/catch_test_macros.hpp>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/wifi_page.hpp"

namespace
{

lv_obj_t *find_wifi_scroll_content(lv_obj_t *container)
{
	if (container == nullptr) {
		return nullptr;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(container); ++index) {
		lv_obj_t *candidate = lv_obj_get_child(container, index);
		if (candidate == nullptr || lv_obj_get_child_count(candidate) != 3) {
			continue;
		}
		lv_obj_t *action_row = lv_obj_get_child(candidate, 2);
		if (action_row != nullptr && lv_obj_get_child_count(action_row) == 3) {
			return candidate;
		}
	}
	return nullptr;
}

class WifiPagePendingFixture {
    public:
	WifiPagePendingFixture()
	{
		_page.on_wifi_enabled_change = [this](bool enabled) {
			if (enabled) {
				++_enable_requests;
			}
		};
		_page.on_wifi_config_click = [this]() {
			++_new_connection_requests;
		};
		_page.create(_lvgl.screen());
		refresh_controls();
	}

	~WifiPagePendingFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	void apply(ui::WifiPageVisualState state, bool busy,
		   bool saved_network_config)
	{
		_page.set_runtime_wifi_state(state, {}, busy, "",
					     {}, saved_network_config, false, {}, {});
	}

	void click(lv_obj_t *button)
	{
		_lvgl.press(button);
		_lvgl.release();
	}

	lv_obj_t *cached_network_button() const
	{
		return _cached_network_button;
	}

	lv_obj_t *new_connection_button() const
	{
		return _new_connection_button;
	}

	lv_obj_t *back_button() const
	{
		return _back_button;
	}

	int enable_requests() const
	{
		return _enable_requests;
	}

	int new_connection_requests() const
	{
		return _new_connection_requests;
	}

    private:
	void refresh_controls()
	{
		lv_obj_update_layout(_lvgl.screen());
		lv_obj_t *container = _page.get_container();
		if (container == nullptr ||
		    lv_obj_get_child_count(container) == 0) {
			return;
		}
		lv_obj_t *scroll_content = find_wifi_scroll_content(container);
		if (scroll_content == nullptr ||
		    lv_obj_get_child_count(scroll_content) != 3) {
			return;
		}
		_back_button = lv_obj_get_child(scroll_content, 0);
		lv_obj_t *action_row = lv_obj_get_child(scroll_content, 2);
		if (action_row != nullptr &&
		    lv_obj_get_child_count(action_row) == 3) {
			_cached_network_button = lv_obj_get_child(action_row, 0);
			_new_connection_button = lv_obj_get_child(action_row, 1);
		}
	}

	test_support::LvglTestContext _lvgl;
	ui::WifiPage _page;
	lv_obj_t *_back_button = nullptr;
	lv_obj_t *_cached_network_button = nullptr;
	lv_obj_t *_new_connection_button = nullptr;
	int _enable_requests = 0;
	int _new_connection_requests = 0;
};

bool is_clickable(lv_obj_t *object)
{
	return object != nullptr && lv_obj_has_flag(object, LV_OBJ_FLAG_CLICKABLE);
}

bool has_pending_arc(lv_obj_t *button)
{
	return test_support::find_visible_descendant_of_type(button, &lv_arc_class) !=
	       nullptr;
}

} // namespace

TEST_CASE("Wi-Fi cached-network request blocks both controls until runtime confirmation",
	  "[ui][wifi][pending]")
{
	WifiPagePendingFixture fixture;
	fixture.apply(ui::WifiPageVisualState::Off, false, true);

	REQUIRE(is_clickable(fixture.cached_network_button()));
	REQUIRE(is_clickable(fixture.new_connection_button()));
	fixture.click(fixture.cached_network_button());

	CHECK(fixture.enable_requests() == 1);
	CHECK_FALSE(is_clickable(fixture.cached_network_button()));
	CHECK_FALSE(is_clickable(fixture.new_connection_button()));
	CHECK_FALSE(is_clickable(fixture.back_button()));
	CHECK(has_pending_arc(fixture.cached_network_button()));

	fixture.click(fixture.new_connection_button());
	CHECK(fixture.new_connection_requests() == 0);

	fixture.apply(ui::WifiPageVisualState::Connecting, true, true);
	CHECK(has_pending_arc(fixture.cached_network_button()));
	fixture.apply(ui::WifiPageVisualState::Connected, false, true);
	CHECK_FALSE(has_pending_arc(fixture.cached_network_button()));
}

TEST_CASE("Wi-Fi runtime busy keeps both actions unavailable without a reset hold",
	  "[ui][wifi][pending]")
{
	WifiPagePendingFixture fixture;
	fixture.apply(ui::WifiPageVisualState::Connecting, true, true);

	CHECK_FALSE(is_clickable(fixture.cached_network_button()));
	CHECK_FALSE(is_clickable(fixture.new_connection_button()));
	CHECK_FALSE(is_clickable(fixture.back_button()));
	CHECK(has_pending_arc(fixture.cached_network_button()));
}

TEST_CASE("Wi-Fi pending spinner is projected when runtime state arrives first",
	  "[ui][wifi][pending]")
{
	test_support::LvglTestContext lvgl;
	ui::WifiPage page;
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connecting, {}, true,
					    "", {}, true, false, {}, {});
	page.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *container = page.get_container();
	REQUIRE(container != nullptr);
	lv_obj_t *scroll_content = find_wifi_scroll_content(container);
	REQUIRE(scroll_content != nullptr);
	lv_obj_t *action_row = lv_obj_get_child(scroll_content, 2);
	REQUIRE(action_row != nullptr);
	lv_obj_t *cached_network = lv_obj_get_child(action_row, 0);
	REQUIRE(cached_network != nullptr);
	CHECK(has_pending_arc(cached_network));

	page.destroy();
}
