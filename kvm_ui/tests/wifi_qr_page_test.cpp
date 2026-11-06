#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/wifi_qr_page.hpp"

namespace
{

constexpr int32_t QR_BACKGROUND_WIDTH = 237;
constexpr int32_t QR_BACKGROUND_HEIGHT = 178;
constexpr int32_t QR_CODE_SIZE = 148;
constexpr int32_t LOADING_ARC_SIZE = 72;
constexpr int32_t LOADING_ARC_WIDTH = 8;
constexpr uint32_t LOADING_ARC_COLOR = 0x2B7FFF;

lv_obj_t *find_direct_child_by_size(lv_obj_t *parent, int32_t width,
				    int32_t height)
{
	if (parent == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_get_width(child) == width &&
		    lv_obj_get_height(child) == height) {
			return child;
		}
	}

	return nullptr;
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

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

class WifiQrPageFixture {
    public:
	WifiQrPageFixture()
	{
		_page.on_qr_start_request = [this]() {
			++_start_requests;
		};
		_page.on_qr_stop_request = [this]() {
			++_stop_requests;
			return _stop_result_generation;
		};
		_page.on_close_ready = [this]() {
			++_close_ready_requests;
		};
		_page.create(_lvgl.screen());
		_page.on_enter();
		refresh_objects();
	}

	~WifiQrPageFixture()
	{
		if (!_destroyed) {
			_page.destroy();
		}
	}

	WifiQrPageFixture(const WifiQrPageFixture &) = delete;
	WifiQrPageFixture &operator=(const WifiQrPageFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::WifiQrPage &page()
	{
		return _page;
	}

	lv_obj_t *container() const
	{
		return _container;
	}

	lv_obj_t *card() const
	{
		return _card;
	}

	lv_obj_t *qr_background() const
	{
		return _qr_background;
	}

	lv_obj_t *qr_code() const
	{
		return _qr_code;
	}

	lv_obj_t *loading_arc() const
	{
		return _loading_arc;
	}

	void set_stop_result_generation(uint64_t generation)
	{
		_stop_result_generation = generation;
	}

	void disable_stop_callback()
	{
		_page.on_qr_stop_request = nullptr;
	}

	void click(lv_obj_t *target)
	{
		lv_obj_send_event(target, LV_EVENT_CLICKED, nullptr);
	}

	void reenter()
	{
		_page.on_exit();
		_page.on_enter();
		refresh_objects();
	}

	void destroy_page()
	{
		_page.destroy();
		_destroyed = true;
		_container = nullptr;
		_card = nullptr;
		_qr_background = nullptr;
		_qr_code = nullptr;
		_loading_arc = nullptr;
	}

	int start_requests() const
	{
		return _start_requests;
	}

	int stop_requests() const
	{
		return _stop_requests;
	}

	int close_ready_requests() const
	{
		return _close_ready_requests;
	}

    private:
	void refresh_objects()
	{
		lv_obj_update_layout(_lvgl.screen());
		_container = _page.get_container();
		if (_container == nullptr) {
			return;
		}
		_card = find_direct_child_by_size(_container, 270, 240);
		_qr_background = find_direct_child_by_size(
			_card, QR_BACKGROUND_WIDTH, QR_BACKGROUND_HEIGHT);
		if (_qr_background == nullptr) {
			return;
		}
		_loading_arc =
			find_descendant_of_type(_qr_background, &lv_arc_class);
		const uint32_t child_count =
			lv_obj_get_child_count(_qr_background);
		for (uint32_t index = 0; index < child_count; ++index) {
			lv_obj_t *child =
				lv_obj_get_child(_qr_background, index);
			if (!lv_obj_check_type(child, &lv_arc_class) &&
			    lv_obj_get_width(child) == QR_CODE_SIZE &&
			    lv_obj_get_height(child) == QR_CODE_SIZE) {
				_qr_code = child;
				break;
			}
		}
	}

	test_support::LvglTestContext _lvgl;
	ui::WifiQrPage _page;
	lv_obj_t *_container = nullptr;
	lv_obj_t *_card = nullptr;
	lv_obj_t *_qr_background = nullptr;
	lv_obj_t *_qr_code = nullptr;
	lv_obj_t *_loading_arc = nullptr;
	uint64_t _stop_result_generation = 41;
	int _start_requests = 0;
	int _stop_requests = 0;
	int _close_ready_requests = 0;
	bool _destroyed = false;
};

void require_qr_layout(const WifiQrPageFixture &fixture)
{
	REQUIRE(fixture.container() != nullptr);
	REQUIRE(fixture.card() != nullptr);
	REQUIRE(fixture.qr_background() != nullptr);
	REQUIRE(fixture.qr_code() != nullptr);
	REQUIRE(fixture.loading_arc() != nullptr);
	CHECK(lv_obj_get_width(fixture.container()) == 270);
	CHECK(lv_obj_get_height(fixture.container()) == 240);
	CHECK(lv_obj_get_width(fixture.card()) == 270);
	CHECK(lv_obj_get_height(fixture.card()) == 240);
	CHECK(lv_obj_get_width(fixture.qr_background()) == QR_BACKGROUND_WIDTH);
	CHECK(lv_obj_get_height(fixture.qr_background()) ==
	      QR_BACKGROUND_HEIGHT);
	CHECK(lv_obj_get_width(fixture.qr_code()) == QR_CODE_SIZE);
	CHECK(lv_obj_get_height(fixture.qr_code()) == QR_CODE_SIZE);
	CHECK(lv_obj_has_flag(fixture.loading_arc(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(fixture.qr_code(), LV_OBJ_FLAG_HIDDEN));
}

void check_closing_visuals(const WifiQrPageFixture &fixture,
			   const lv_area_t &container_before,
			   const lv_area_t &card_before,
			   const lv_area_t &background_before)
{
	CHECK(lv_obj_has_flag(fixture.qr_code(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(fixture.loading_arc(), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_width(fixture.loading_arc()) == LOADING_ARC_SIZE);
	CHECK(lv_obj_get_height(fixture.loading_arc()) == LOADING_ARC_SIZE);
	CHECK(lv_obj_get_style_arc_width(fixture.loading_arc(),
					 LV_PART_INDICATOR) ==
	      LOADING_ARC_WIDTH);
	CHECK(lv_color_eq(lv_obj_get_style_arc_color(fixture.loading_arc(),
						     LV_PART_INDICATOR),
			  lv_color_hex(LOADING_ARC_COLOR)));

	const lv_area_t arc_coordinates = object_coords(fixture.loading_arc());
	const lv_area_t background_coordinates =
		object_coords(fixture.qr_background());
	const int32_t arc_center_x =
		(arc_coordinates.x1 + arc_coordinates.x2) / 2;
	const int32_t arc_center_y =
		(arc_coordinates.y1 + arc_coordinates.y2) / 2;
	const int32_t background_center_x =
		(background_coordinates.x1 + background_coordinates.x2) / 2;
	const int32_t background_center_y =
		(background_coordinates.y1 + background_coordinates.y2) / 2;
	CHECK(arc_center_x >= background_center_x - 1);
	CHECK(arc_center_x <= background_center_x + 1);
	CHECK(arc_center_y >= background_center_y - 1);
	CHECK(arc_center_y <= background_center_y + 1);

	const lv_area_t container_after = object_coords(fixture.container());
	const lv_area_t card_after = object_coords(fixture.card());
	const lv_area_t background_after =
		object_coords(fixture.qr_background());
	CHECK(container_after.x1 == container_before.x1);
	CHECK(container_after.y1 == container_before.y1);
	CHECK(card_after.x1 == card_before.x1);
	CHECK(card_after.y1 == card_before.y1);
	CHECK(background_after.x1 == background_before.x1);
	CHECK(background_after.y1 == background_before.y1);
}

} // namespace

TEST_CASE("QR dismissal gates every existing click surface while closing",
	  "[ui][wifi][qr]")
{
	SECTION("container")
	{
		WifiQrPageFixture fixture;
		require_qr_layout(fixture);
		const lv_area_t container_before =
			object_coords(fixture.container());
		const lv_area_t card_before = object_coords(fixture.card());
		const lv_area_t background_before =
			object_coords(fixture.qr_background());

		fixture.click(fixture.container());
		CHECK(fixture.stop_requests() == 1);
		check_closing_visuals(fixture, container_before, card_before,
				      background_before);
		fixture.click(fixture.card());
		fixture.click(fixture.qr_background());
		fixture.click(fixture.qr_code());
		CHECK(fixture.stop_requests() == 1);
	}

	SECTION("card")
	{
		WifiQrPageFixture fixture;
		require_qr_layout(fixture);
		fixture.click(fixture.card());
		CHECK(fixture.stop_requests() == 1);
	}

	SECTION("white QR background")
	{
		WifiQrPageFixture fixture;
		require_qr_layout(fixture);
		fixture.click(fixture.qr_background());
		CHECK(fixture.stop_requests() == 1);
	}

	SECTION("QR code")
	{
		WifiQrPageFixture fixture;
		require_qr_layout(fixture);
		fixture.click(fixture.qr_code());
		CHECK(fixture.stop_requests() == 1);
	}
}

TEST_CASE(
	"QR page accepts only the matching stop completion and resets sessions",
	"[ui][wifi][qr]")
{
	WifiQrPageFixture fixture;
	require_qr_layout(fixture);
	CHECK(fixture.start_requests() == 1);

	fixture.set_stop_result_generation(41);
	fixture.click(fixture.qr_code());
	fixture.page().set_ap_stop_completed_generation(40);
	CHECK(fixture.close_ready_requests() == 0);
	fixture.page().set_ap_stop_completed_generation(41);
	CHECK(fixture.close_ready_requests() == 1);

	fixture.reenter();
	require_qr_layout(fixture);
	CHECK(fixture.start_requests() == 2);
	fixture.page().set_ap_stop_completed_generation(41);
	CHECK(fixture.close_ready_requests() == 1);

	fixture.set_stop_result_generation(42);
	fixture.click(fixture.qr_code());
	fixture.page().set_ap_stop_completed_generation(41);
	CHECK(fixture.close_ready_requests() == 1);
	fixture.page().set_ap_stop_completed_generation(42);
	CHECK(fixture.close_ready_requests() == 2);
}

TEST_CASE("QR page closes through the LVGL queue when stop cannot be requested",
	  "[ui][wifi][qr]")
{
	WifiQrPageFixture fixture;
	require_qr_layout(fixture);
	fixture.disable_stop_callback();
	fixture.click(fixture.qr_code());
	CHECK(fixture.stop_requests() == 0);
	CHECK(fixture.close_ready_requests() == 0);
	fixture.lvgl().advance(0);
	CHECK(fixture.close_ready_requests() == 1);
}

TEST_CASE("Destroying a closing QR page cancels its timer and callback",
	  "[ui][wifi][qr]")
{
	WifiQrPageFixture fixture;
	require_qr_layout(fixture);
	const std::size_t initial_timer_count = fixture.lvgl().timer_count();
	fixture.click(fixture.qr_code());
	REQUIRE(fixture.stop_requests() == 1);
	REQUIRE(fixture.lvgl().timer_count() > initial_timer_count);

	fixture.destroy_page();
	fixture.lvgl().advance(2000);
	CHECK(fixture.close_ready_requests() == 0);
	CHECK(fixture.lvgl().timer_count() <= initial_timer_count);
}
