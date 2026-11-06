#include <catch2/catch_test_macros.hpp>

#include <algorithm>
#include <cstddef>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/bconb.hpp"
#include "ui/scona.hpp"
#include "ui/ssid_info_row.hpp"
#include "ui/ssid_raster_marquee.hpp"
#include "ui/wifi_page.hpp"
#include "ui/wifi_ssid_display_policy.hpp"
#include "ui/wifi_ssid_font.hpp"

extern "C" {
LV_IMAGE_DECLARE(ci_add_plus);
LV_IMAGE_DECLARE(wifi_new);
LV_FONT_DECLARE(MontserratBold_20);
}

namespace ui
{

class WifiPageTestAccess final {
    public:
	static const std::string &raw_ssid(const WifiPage &page)
	{
		return page._wifi_ssid;
	}

	static const std::string &wifi_ip(const WifiPage &page)
	{
		return page._wifi_ip;
	}

	static const std::string &display_ssid(const WifiPage &page)
	{
		return page._ssid_label_text;
	}

	static lv_obj_t *information_group(const WifiPage &page)
	{
		return page._information_group;
	}

	static SsidInfoRow *ssid_info_row(const WifiPage &page)
	{
		return page._ssid_info_row;
	}

	static SConA *ip_label(const WifiPage &page)
	{
		return page._ip_label;
	}

	static lv_obj_t *scroll_content(const WifiPage &page)
	{
		return page._scroll_content;
	}

	static lv_obj_t *action_row(const WifiPage &page)
	{
		return page._action_row;
	}

	static lv_obj_t *qr_popup(const WifiPage &page)
	{
		return page._qr_popup;
	}

	static bool ssid_presentation_applied(const WifiPage &page)
	{
		return page._ssid_presentation_applied;
	}

	static bool image_cache_failed(const WifiPage &page)
	{
		return page._ssid_image_cache_failed;
	}

	static const SsidRasterMarquee *ssid_marquee(const WifiPage &page)
	{
		return page._ssid_info_row == nullptr ? nullptr :
		       page._ssid_info_row->_raster_marquee.get();
	}

	static bool raster_request_pending(const WifiPage &page)
	{
		return page._ssid_raster_request_pending;
	}

	static bool raster_ready(const WifiPage &page)
	{
		return page._ssid_ready_identity.has_value();
	}

	static bool raster_failed(const WifiPage &page)
	{
		return page._ssid_failed_identity.has_value();
	}

	static bool ssid_interaction_held(const WifiPage &page)
	{
		return page._ssid_interaction_held;
	}

	static bool has_ssid_interaction_idle_timer(const WifiPage &page)
	{
		return page._ssid_interaction_idle_timer != nullptr;
	}

	static bool has_deferred_prepared_ssid_raster(const WifiPage &page)
	{
		return page._deferred_prepared_ssid_raster.has_value();
	}
};

} // namespace ui

namespace
{

class WifiPageNavigationFixture {
    public:
	explicit WifiPageNavigationFixture(bool enable_ssid_scroll = false)
		: _page(nullptr, enable_ssid_scroll)
	{
		_page.on_wifi_enabled_change = [this](bool enabled) {
			if (enabled) {
				++_enable_requests;
			} else {
				++_disable_requests;
			}
		};
		_page.on_wifi_config_click = [this]() {
			++_setup_navigation_requests;
		};
		_page.create(_lvgl.screen());
		refresh_controls();
	}

	~WifiPageNavigationFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	WifiPageNavigationFixture(const WifiPageNavigationFixture &) = delete;
	WifiPageNavigationFixture &
	operator=(const WifiPageNavigationFixture &) = delete;

	void apply(ui::WifiPageVisualState state, std::string_view ssid,
		   bool saved_network_config, bool busy = false, bool show_qr = false,
		   std::string_view ip = {})
	{
		_page.set_runtime_wifi_state(state, {}, busy, ssid, ip,
					     saved_network_config, show_qr, {}, {});
		refresh_controls();
	}

	void set_busy(bool busy)
	{
		_page.set_busy(busy);
		refresh_controls();
	}

	void click(lv_obj_t *button)
	{
		_lvgl.press(button);
		_lvgl.release();
	}

	void advance(uint32_t ms)
	{
		_lvgl.advance(ms);
	}

	lv_obj_t *information_group() const
	{
		return ui::WifiPageTestAccess::information_group(_page);
	}

	ui::SsidInfoRow *ssid_info_row() const
	{
		return ui::WifiPageTestAccess::ssid_info_row(_page);
	}

	ui::SConA *ip_label() const
	{
		return ui::WifiPageTestAccess::ip_label(_page);
	}

	ui::WifiPage &page()
	{
		return _page;
	}

	const ui::WifiPage &page() const
	{
		return _page;
	}

	lv_obj_t *cached_network_button() const
	{
		return _cached_network_button;
	}

	lv_obj_t *new_connection_button() const
	{
		return _new_connection_button;
	}

	lv_obj_t *open_setup_button() const
	{
		return _open_setup_button;
	}

	lv_obj_t *qr_popup() const
	{
		return ui::WifiPageTestAccess::qr_popup(_page);
	}

	std::string_view ssid_text() const
	{
		return ui::WifiPageTestAccess::display_ssid(_page);
	}

	int enable_requests() const
	{
		return _enable_requests;
	}

	int disable_requests() const
	{
		return _disable_requests;
	}

	int setup_navigation_requests() const
	{
		return _setup_navigation_requests;
	}

    private:
	void refresh_controls()
	{
		lv_obj_update_layout(_lvgl.screen());
		lv_obj_t *scroll_content =
			ui::WifiPageTestAccess::scroll_content(_page);
		if (scroll_content == nullptr ||
		    lv_obj_get_child_count(scroll_content) != 3) {
			return;
		}

		lv_obj_t *action_row = ui::WifiPageTestAccess::action_row(_page);
		if (action_row != nullptr &&
		    lv_obj_get_child_count(action_row) == 3) {
			_cached_network_button = lv_obj_get_child(action_row, 0);
			_new_connection_button = lv_obj_get_child(action_row, 1);
			_open_setup_button = lv_obj_get_child(action_row, 2);
		}
	}

	test_support::LvglTestContext _lvgl;
	ui::WifiPage _page;
	lv_obj_t *_cached_network_button = nullptr;
	lv_obj_t *_new_connection_button = nullptr;
	lv_obj_t *_open_setup_button = nullptr;
	int _enable_requests = 0;
	int _disable_requests = 0;
	int _setup_navigation_requests = 0;
};

void require_controls(const WifiPageNavigationFixture &fixture)
{
	REQUIRE(fixture.information_group() != nullptr);
	REQUIRE(fixture.ssid_info_row() != nullptr);
	REQUIRE(fixture.ip_label() != nullptr);
	REQUIRE(fixture.cached_network_button() != nullptr);
	REQUIRE(fixture.new_connection_button() != nullptr);
	REQUIRE(fixture.open_setup_button() != nullptr);
	CHECK(lv_obj_get_width(fixture.information_group()) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(fixture.ssid_info_row()->get_obj()) ==
	      ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_get_height(fixture.ip_label()->get_obj()) == ui::SConA::HEIGHT);
	CHECK(lv_obj_get_width(fixture.cached_network_button()) == ui::BConB::WIDTH);
	CHECK(lv_obj_get_height(fixture.cached_network_button()) == ui::BConB::HEIGHT);
	CHECK(lv_obj_get_width(fixture.new_connection_button()) == ui::BConB::WIDTH);
	CHECK(lv_obj_get_height(fixture.new_connection_button()) == ui::BConB::HEIGHT);
	CHECK(lv_obj_get_width(fixture.open_setup_button()) == ui::BConA::WIDTH);
	CHECK(lv_obj_get_height(fixture.open_setup_button()) == ui::BConA::HEIGHT);
}

bool is_clickable(lv_obj_t *object)
{
	return object != nullptr && lv_obj_has_flag(object, LV_OBJ_FLAG_CLICKABLE);
}

bool is_hidden(lv_obj_t *object)
{
	return object != nullptr && lv_obj_has_flag(object, LV_OBJ_FLAG_HIDDEN);
}

bool has_background_color(lv_obj_t *object, uint32_t color)
{
	return object != nullptr &&
	       lv_color_eq(lv_obj_get_style_bg_color(object, LV_PART_MAIN),
			   lv_color_hex(color));
}

bool contains_image_source(lv_obj_t *object, const void *source)
{
	if (object == nullptr) {
		return false;
	}
	if (lv_obj_check_type(object, &lv_image_class) &&
	    lv_image_get_src(object) == source) {
		return true;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(object); ++index) {
		if (contains_image_source(lv_obj_get_child(object, index), source)) {
			return true;
		}
	}
	return false;
}

bool visible_tree_uses_font(lv_obj_t *object, const lv_font_t *font,
			    bool ancestor_hidden = false)
{
	if (object == nullptr || font == nullptr) {
		return false;
	}
	const bool hidden = ancestor_hidden ||
			    lv_obj_has_flag(object, LV_OBJ_FLAG_HIDDEN);
	if (!hidden && lv_obj_check_type(object, &lv_label_class) &&
	    lv_obj_get_style_text_font(object, LV_PART_MAIN) == font) {
		return true;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(object); ++index) {
		if (visible_tree_uses_font(lv_obj_get_child(object, index), font,
					   hidden)) {
			return true;
		}
	}
	return false;
}

lv_obj_t *direct_child_of_type(lv_obj_t *parent, const lv_obj_class_t *type)
{
	if (parent == nullptr || type == nullptr) {
		return nullptr;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(parent); ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, type)) {
			return child;
		}
	}
	return nullptr;
}

lv_obj_t *ssid_value_canvas(const ui::WifiPage &page)
{
	ui::SsidInfoRow *const row = ui::WifiPageTestAccess::ssid_info_row(page);
	return row == nullptr ? nullptr :
		direct_child_of_type(row->get_obj(), &lv_canvas_class);
}

lv_obj_t *ssid_fallback_label(const ui::WifiPage &page)
{
	ui::SsidInfoRow *const row = ui::WifiPageTestAccess::ssid_info_row(page);
	if (row == nullptr) {
		return nullptr;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(row->get_obj());
	     ++index) {
		lv_obj_t *const child = lv_obj_get_child(row->get_obj(), index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    lv_obj_get_x(child) == ui::SsidInfoRow::VALUE_X) {
			return child;
		}
	}
	return nullptr;
}

std::vector<uint8_t> canvas_pixels(lv_obj_t *canvas)
{
	if (canvas == nullptr || !lv_obj_check_type(canvas, &lv_canvas_class)) {
		return {};
	}
	const lv_draw_buf_t *const draw_buf = lv_canvas_get_draw_buf(canvas);
	if (draw_buf == nullptr || draw_buf->data == nullptr) {
		return {};
	}
	const auto *const bytes = static_cast<const uint8_t *>(draw_buf->data);
	return { bytes, bytes + draw_buf->data_size };
}

struct MarqueeMotionEvents {
	uint32_t style_changed = 0;
	uint32_t layout_changed = 0;
};

void marquee_motion_event_cb(lv_event_t *event)
{
	auto *const events = static_cast<MarqueeMotionEvents *>(
		lv_event_get_user_data(event));
	if (events == nullptr) {
		return;
	}

	switch (lv_event_get_code(event)) {
	case LV_EVENT_STYLE_CHANGED:
		++events->style_changed;
		break;
	case LV_EVENT_LAYOUT_CHANGED:
		++events->layout_changed;
		break;
	default:
		break;
	}
}

void observe_marquee_motion_events(lv_obj_t *object,
				   MarqueeMotionEvents *events)
{
	if (object == nullptr || events == nullptr) {
		return;
	}
	lv_obj_add_event_cb(object, marquee_motion_event_cb,
			    LV_EVENT_STYLE_CHANGED, events);
	lv_obj_add_event_cb(object, marquee_motion_event_cb,
			    LV_EVENT_LAYOUT_CHANGED, events);
}

int32_t object_coordinate_x(lv_obj_t *object)
{
	lv_area_t area {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &area);
	}
	return area.x1;
}

ui::SsidPreparedRasterResult prepared_ssid_result(
	const ui::SsidRasterRequest &request, bool overflows_viewport,
	uint32_t pixel_marker)
{
	const uint16_t source_width = overflows_viewport ?
		static_cast<uint16_t>(request.identity.viewport_width + 64U) :
		request.identity.viewport_width;
	auto pixels = std::make_shared<std::vector<uint32_t>>(
		static_cast<std::size_t>(source_width) * request.identity.height,
		0xff000000U | (pixel_marker & 0x00ffffffU));
	auto raster = std::make_shared<ui::SsidPreparedRaster>();
	raster->identity = request.identity;
	raster->source_width = source_width;
	raster->source_height = request.identity.height;
	raster->stride_bytes = static_cast<uint32_t>(source_width) * sizeof(uint32_t);
	raster->text_width = overflows_viewport ? source_width :
		static_cast<int32_t>(request.identity.viewport_width / 2U);
	raster->space_advance = 5;
	raster->overflows_viewport = overflows_viewport;
	raster->pixels = std::move(pixels);
	REQUIRE(raster->has_valid_storage());
	return {.identity = request.identity, .raster = std::move(raster)};
}

lv_obj_t *direct_label_with_text(lv_obj_t *parent, std::string_view text)
{
	if (parent == nullptr) {
		return nullptr;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(parent); ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    std::string_view(lv_label_get_text(child)) == text) {
			return child;
		}
	}
	return nullptr;
}

} // namespace

TEST_CASE("Wi-Fi main uses About-style SSID and IP information rows",
	  "[ui][wifi][navigation][layout]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	lv_obj_t *const information_group = fixture.information_group();
	lv_obj_t *const ssid_row = fixture.ssid_info_row()->get_obj();
	lv_obj_t *const ip_row = fixture.ip_label()->get_obj();
	lv_obj_t *const fallback_label = ssid_fallback_label(fixture.page());
	REQUIRE(information_group != nullptr);
	REQUIRE(ssid_row != nullptr);
	REQUIRE(ip_row != nullptr);
	REQUIRE(fallback_label != nullptr);
	lv_obj_t *const logical_canvas = lv_obj_get_parent(information_group);
	REQUIRE(logical_canvas != nullptr);
	lv_area_t information_group_coords {};
	lv_area_t logical_canvas_coords {};
	lv_obj_get_coords(information_group, &information_group_coords);
	lv_obj_get_coords(logical_canvas, &logical_canvas_coords);
	CHECK(information_group_coords.y1 - logical_canvas_coords.y1 == 76);
	CHECK(lv_obj_get_y(ssid_row) == 0);
	CHECK(lv_obj_get_y(ip_row) == ui::SsidInfoRow::HEIGHT);
	lv_obj_t *const ssid_key = direct_label_with_text(ssid_row, "SSID:");
	lv_obj_t *const ip_key = direct_label_with_text(ip_row, "IP:");
	REQUIRE(ssid_key != nullptr);
	REQUIRE(ip_key != nullptr);
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == "--");
	CHECK(lv_obj_get_style_text_align(fallback_label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_RIGHT);
	CHECK(lv_obj_get_style_pad_left(ip_row, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_right(ip_row, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_top(ip_row, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_style_pad_bottom(ip_row, LV_PART_MAIN) == 0);
	CHECK(lv_obj_get_x(ssid_key) == ui::SsidInfoRow::KEY_HORIZONTAL_INSET);
	CHECK(lv_obj_get_x(ip_key) == ui::SsidInfoRow::KEY_HORIZONTAL_INSET);
	CHECK(lv_obj_get_x(fallback_label) + lv_obj_get_width(fallback_label) ==
	      ui::SsidInfoRow::WIDTH - ui::SsidInfoRow::VALUE_RIGHT_INSET);

	auto check_ip_value_alignment = [&](std::string_view value) {
		lv_obj_t *const ip_value = direct_label_with_text(ip_row, value);
		REQUIRE(ip_value != nullptr);
		lv_area_t ip_key_coords {};
		lv_area_t ip_value_coords {};
		lv_obj_get_coords(ip_key, &ip_key_coords);
		lv_obj_get_coords(ip_value, &ip_value_coords);
		CHECK(lv_obj_get_x(ip_key) == lv_obj_get_x(ssid_key));
		CHECK(ip_value_coords.x1 > ip_key_coords.x2);
		CHECK(lv_obj_get_x(ip_value) + lv_obj_get_width(ip_value) ==
		      lv_obj_get_x(fallback_label) + lv_obj_get_width(fallback_label));
	};

	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true,
		      false, false, "192.0.2.8");
	CHECK(fixture.ssid_text() == "Office WiFi");
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == "Office WiFi");
	check_ip_value_alignment("192.0.2.8");
	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true,
		      false, false, "198.51.100.255");
	check_ip_value_alignment("198.51.100.255");

	struct IpStateCase {
		ui::WifiPageVisualState state;
		std::string_view ip;
		std::string_view expected;
	};
	const std::vector<IpStateCase> cases {
		{ ui::WifiPageVisualState::Connected, "", "--" },
		{ ui::WifiPageVisualState::Connecting, "192.0.2.9", "--" },
		{ ui::WifiPageVisualState::ApConnect, "192.0.2.9", "--" },
		{ ui::WifiPageVisualState::WebConfigure, "192.0.2.9", "--" },
		{ ui::WifiPageVisualState::Off, "192.0.2.9", "--" },
		{ ui::WifiPageVisualState::Error, "192.0.2.9", "--" },
	};
	for (const IpStateCase &test_case : cases) {
		fixture.apply(test_case.state, "Office WiFi", true, false, false,
			      test_case.ip);
		REQUIRE(direct_label_with_text(ip_row, test_case.expected) != nullptr);
		check_ip_value_alignment(test_case.expected);
	}
}

TEST_CASE("Wi-Fi main scroll option moves long compiled SSIDs",
	  "[ui][wifi][navigation][ssid][scroll]")
{
	WifiPageNavigationFixture fixture(true);
	require_controls(fixture);
	fixture.page().on_enter();
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(fixture.page()));
	CHECK(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(fixture.page()));

	fixture.apply(ui::WifiPageVisualState::Connected, "short", true);
	lv_obj_t *const fallback_label = ssid_fallback_label(fixture.page());
	REQUIRE(fallback_label != nullptr);
	CHECK(lv_label_get_long_mode(fallback_label) ==
	      LV_LABEL_LONG_CLIP);
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);

	const std::string_view long_ssid =
		"A connected SSID must move through the fixed value viewport";
	fixture.apply(ui::WifiPageVisualState::Connected, long_ssid, true);
	CHECK(fixture.ssid_text() == long_ssid);
	CHECK(lv_label_get_long_mode(fallback_label) ==
	      LV_LABEL_LONG_CLIP);
	fixture.advance(499);
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	fixture.advance(1);
	CHECK_FALSE(ui::WifiPageTestAccess::ssid_interaction_held(fixture.page()));
	lv_anim_t *animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(animation != nullptr);
	const int32_t initial_offset = animation->current_value;
	fixture.advance(500);
	animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(animation != nullptr);
	CHECK(animation->current_value != initial_offset);
	fixture.page().on_pause();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	CHECK_FALSE(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(
		fixture.page()));
	fixture.page().on_resume();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	fixture.advance(499);
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	fixture.advance(1);
	CHECK(lv_anim_get(fallback_label, nullptr) != nullptr);
	fixture.page().on_exit();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	fixture.page().on_enter();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	fixture.advance(500);
	CHECK(lv_anim_get(fallback_label, nullptr) != nullptr);
	fixture.apply(ui::WifiPageVisualState::Connected, long_ssid, true);
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == long_ssid);
}

TEST_CASE("Wi-Fi main keeps SSIDs static when scrolling is disabled",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture(false);
	fixture.apply(ui::WifiPageVisualState::Connected,
		"A long connected SSID stays clipped when the option is disabled", true);
	lv_obj_t *const fallback_label = ssid_fallback_label(fixture.page());
	REQUIRE(fallback_label != nullptr);
	CHECK(lv_label_get_long_mode(fallback_label) == LV_LABEL_LONG_CLIP);
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
}

TEST_CASE("Wi-Fi main keeps an injected TinyTTF font off the visible object tree",
	  "[ui][wifi][ssid]")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiPage page(&runtime_font, false, 128,
			  ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lvgl.screen());

	REQUIRE(page.get_container() != nullptr);
	REQUIRE(ssid_value_canvas(page) != nullptr);
	CHECK_FALSE(visible_tree_uses_font(page.get_container(), &runtime_font));
	page.destroy();
}

TEST_CASE("Wi-Fi main installs only the latest prepared TinyTTF SSID",
	  "[ui][wifi][ssid]")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiPage page(&runtime_font, false, 128,
			  ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lvgl.screen());
	page.on_enter();
	lvgl.advance(500);
	lv_obj_update_layout(lvgl.screen());
	std::vector<ui::SsidRasterRequest> requests;
	std::size_t invalidations = 0;
	page.on_ssid_raster_prepare = [&requests](ui::SsidRasterRequest request) {
		requests.push_back(std::move(request));
		return true;
	};
	page.on_ssid_raster_invalidate = [&invalidations](uint64_t) {
		++invalidations;
	};

	lv_obj_t *const container = page.get_container();
	lv_obj_t *const canvas = ssid_value_canvas(page);
	lv_obj_t *const fallback_label = ssid_fallback_label(page);
	REQUIRE(container != nullptr);
	REQUIRE(canvas != nullptr);
	REQUIRE(fallback_label != nullptr);
	REQUIRE(lv_canvas_get_draw_buf(canvas) != nullptr);
	CHECK(lv_obj_get_width(canvas) == ui::SsidInfoRow::VALUE_WIDTH);
	CHECK(lv_obj_get_height(canvas) == ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_get_x(canvas) == ui::SsidInfoRow::VALUE_X);
	CHECK_FALSE(visible_tree_uses_font(container, &runtime_font));

	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
					    "short", "192.0.2.8", true, false, {}, {});
	REQUIRE(requests.size() == 1U);
	CHECK(ui::WifiPageTestAccess::raster_request_pending(page));
	CHECK_FALSE(ui::WifiPageTestAccess::raster_ready(page));
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == "short");
	CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));

	page.accept_prepared_ssid_raster(
		prepared_ssid_result(requests.front(), false, 0x112233U));
	CHECK_FALSE(ui::WifiPageTestAccess::raster_request_pending(page));
	CHECK(ui::WifiPageTestAccess::raster_ready(page));
	CHECK_FALSE(ui::WifiPageTestAccess::raster_failed(page));
	CHECK_FALSE(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(fallback_label, LV_OBJ_FLAG_HIDDEN));
	const std::vector<uint8_t> short_pixels = canvas_pixels(canvas);

	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
					    "short", "192.0.2.8", true, false, {}, {});
	CHECK(requests.size() == 1U);
	CHECK(canvas_pixels(canvas) == short_pixels);

	page.set_runtime_wifi_state(
		ui::WifiPageVisualState::Connected, {}, false,
		"A replacement SSID must reject the stale worker result",
		"192.0.2.8", true, false, {}, {});
	REQUIRE(requests.size() == 2U);
	CHECK(ui::WifiPageTestAccess::raster_request_pending(page));
	CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	page.accept_prepared_ssid_raster(
		prepared_ssid_result(requests.front(), false, 0x445566U));
	CHECK(ui::WifiPageTestAccess::raster_request_pending(page));
	CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	page.accept_prepared_ssid_raster(
		prepared_ssid_result(requests.back(), false, 0x778899U));
	CHECK(ui::WifiPageTestAccess::raster_ready(page));
	CHECK_FALSE(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK(canvas_pixels(canvas) != short_pixels);

	page.set_runtime_wifi_state(ui::WifiPageVisualState::Off, {}, false, {}, {},
					    true, false, {}, {});
	CHECK(invalidations == 1U);
	CHECK_FALSE(ui::WifiPageTestAccess::raster_ready(page));
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == "--");
	CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	const std::vector<uint8_t> placeholder_pixels = canvas_pixels(canvas);
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Off, {}, false, {}, {},
					    true, false, {}, {});
	CHECK(invalidations == 1U);
	CHECK(canvas_pixels(canvas) == placeholder_pixels);

	page.destroy();
	CHECK(ui::WifiPageTestAccess::ssid_info_row(page) == nullptr);
}

TEST_CASE("Wi-Fi main defers the latest TinyTTF raster until input is quiet",
	  "[ui][wifi][ssid][scroll]")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiPage page(&runtime_font, true, 128,
			  ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lvgl.screen());
	std::vector<ui::SsidRasterRequest> requests;
	int setup_clicks = 0;
	int toggle_clicks = 0;
	int back_clicks = 0;
	page.on_ssid_raster_prepare = [&requests](ui::SsidRasterRequest request) {
		requests.push_back(std::move(request));
		return true;
	};
	page.on_wifi_config_click = [&setup_clicks]() {
		++setup_clicks;
	};
	page.on_wifi_enabled_change = [&toggle_clicks](bool) {
		++toggle_clicks;
	};
	page.on_back_click = [&back_clicks]() {
		++back_clicks;
	};
	page.on_enter();

	page.set_runtime_wifi_state(
		ui::WifiPageVisualState::Connected, {}, false,
		"A first long runtime SSID that must never win after replacement",
		"192.0.2.8", true, false, {}, {});
	REQUIRE(requests.size() == 1U);
	const ui::SsidPreparedRasterResult first =
		prepared_ssid_result(requests.front(), true, 0x112233U);
	page.accept_prepared_ssid_raster(first);
	CHECK(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	CHECK_FALSE(ui::WifiPageTestAccess::raster_ready(page));

	page.set_runtime_wifi_state(
		ui::WifiPageVisualState::Connected, {}, false,
		"A replacement long runtime SSID retains only its latest generation",
		"192.0.2.8", true, false, {}, {});
	REQUIRE(requests.size() == 2U);
	CHECK_FALSE(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	page.accept_prepared_ssid_raster(first);
	CHECK_FALSE(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	page.accept_prepared_ssid_raster(
		prepared_ssid_result(requests.back(), true, 0x445566U));
	CHECK(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));

	lv_obj_t *const scroll_content =
		ui::WifiPageTestAccess::scroll_content(page);
	REQUIRE(scroll_content != nullptr);
	lv_obj_send_event(scroll_content, LV_EVENT_SCROLL_BEGIN, nullptr);
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lvgl.advance(250);
	lv_obj_send_event(scroll_content, LV_EVENT_SCROLL, nullptr);
	lvgl.advance(250);
	lv_obj_send_event(scroll_content, LV_EVENT_SCROLL_END, nullptr);
	lvgl.advance(499);
	CHECK_FALSE(ui::WifiPageTestAccess::raster_ready(page));
	CHECK(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	lvgl.advance(1);
	CHECK(ui::WifiPageTestAccess::raster_ready(page));
	CHECK_FALSE(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	const ui::SsidRasterMarquee *const marquee =
		ui::WifiPageTestAccess::ssid_marquee(page);
	REQUIRE(marquee != nullptr);
	CHECK(marquee->animation_running());

	lv_obj_t *const action_row = ui::WifiPageTestAccess::action_row(page);
	REQUIRE(action_row != nullptr);
	REQUIRE(lv_obj_get_child_count(action_row) == 3U);
	lv_obj_t *const cached_network = lv_obj_get_child(action_row, 0);
	lv_obj_t *const new_connection = lv_obj_get_child(action_row, 1);
	lv_obj_t *const open_setup = lv_obj_get_child(action_row, 2);
	REQUIRE(cached_network != nullptr);
	REQUIRE(new_connection != nullptr);
	REQUIRE(open_setup != nullptr);

	lvgl.press(new_connection);
	CHECK_FALSE(marquee->animation_running());
	lvgl.release();
	CHECK(setup_clicks == 1);
	lvgl.advance(499);
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(1);
	CHECK(marquee->animation_running());

	page.set_runtime_wifi_state(ui::WifiPageVisualState::Off, {}, false, {}, {},
				    false, false, {}, {});
	lvgl.press(open_setup);
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lvgl.release();
	CHECK(setup_clicks == 2);
	lv_obj_send_event(open_setup, LV_EVENT_PRESSED, nullptr);
	lv_obj_send_event(open_setup, LV_EVENT_PRESSING, nullptr);
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lv_obj_send_event(open_setup, LV_EVENT_PRESS_LOST, nullptr);
	CHECK(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(page));

	lv_obj_t *const back_button = lv_obj_get_child(scroll_content, 0);
	REQUIRE(back_button != nullptr);
	lvgl.press(back_button);
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lvgl.release();
	CHECK(back_clicks == 1);

	page.set_runtime_wifi_state(
		ui::WifiPageVisualState::Connected, {}, false,
		"A latest runtime SSID restores the paired action", "192.0.2.8", true,
		false, {}, {});
	lvgl.press(cached_network);
	CHECK(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lvgl.release();
	CHECK(toggle_clicks == 1);

	page.on_pause();
	CHECK_FALSE(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(page));
	CHECK_FALSE(ui::WifiPageTestAccess::ssid_interaction_held(page));
	lvgl.advance(600);
	page.on_resume();
	CHECK(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(page));
	page.on_exit();
	CHECK_FALSE(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(page));
	page.destroy();
	CHECK_FALSE(ui::WifiPageTestAccess::has_ssid_interaction_idle_timer(page));
}

TEST_CASE("Wi-Fi main reuses a hidden-page prepared TinyTTF marquee",
	  "[ui][wifi][ssid][scroll]")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiPage page(&runtime_font, true, 128,
			  ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lvgl.screen());
	REQUIRE(page.get_container() != nullptr);
	lv_obj_add_flag(page.get_container(), LV_OBJ_FLAG_HIDDEN);
	std::vector<ui::SsidRasterRequest> requests;
	std::size_t invalidations = 0;
	page.on_ssid_raster_prepare = [&requests](ui::SsidRasterRequest request) {
		requests.push_back(std::move(request));
		return true;
	};
	page.on_ssid_raster_invalidate = [&invalidations](uint64_t) {
		++invalidations;
	};

	const std::string long_ssid =
		"A Runtime TinyTTF SSID is rasterized once into a fixed window";
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
		long_ssid, "192.0.2.8", true, false, {}, {});
	REQUIRE(requests.size() == 1U);
	page.accept_prepared_ssid_raster(
		prepared_ssid_result(requests.front(), true, 0xabcdefU));
	const ui::SsidRasterMarquee *const marquee =
		ui::WifiPageTestAccess::ssid_marquee(page);
	REQUIRE(marquee != nullptr);
	CHECK_FALSE(marquee->is_visible());
	CHECK_FALSE(marquee->animation_running());
	REQUIRE(marquee->window_image() != nullptr);
	CHECK(lv_obj_get_child_count(marquee->get_obj()) == 1U);
	CHECK(lv_obj_get_style_width(marquee->window_image(), LV_PART_MAIN) ==
	      ui::SsidInfoRow::VALUE_WIDTH);
	CHECK(lv_obj_get_style_height(marquee->window_image(), LV_PART_MAIN) ==
	      ui::SsidInfoRow::HEIGHT);
	const void *const source = lv_image_get_src(marquee->window_image());
	REQUIRE(source != nullptr);
	CHECK(contains_image_source(page.get_container(), source));
	CHECK_FALSE(visible_tree_uses_font(page.get_container(), &runtime_font));
	const uint64_t hidden_revision = marquee->frame_revision();
	lvgl.advance(500);
	CHECK(marquee->frame_revision() == hidden_revision);

	lv_obj_clear_flag(page.get_container(), LV_OBJ_FLAG_HIDDEN);
	page.on_enter();
	CHECK(requests.size() == 1U);
	CHECK_FALSE(marquee->is_visible());
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(499);
	CHECK_FALSE(marquee->is_visible());
	lvgl.advance(1);
	REQUIRE(marquee->is_visible());
	REQUIRE(marquee->animation_running());
	lv_obj_update_layout(lvgl.screen());
	const int32_t image_x = object_coordinate_x(marquee->window_image());
	const int32_t viewport_x = object_coordinate_x(marquee->get_obj());
	const uint64_t initial_revision = marquee->frame_revision();
	MarqueeMotionEvents motion_events {};
	observe_marquee_motion_events(marquee->window_image(), &motion_events);
	observe_marquee_motion_events(marquee->get_obj(), &motion_events);
	const uint32_t track_width = requests.front().identity.viewport_width + 64U +
		5U * LV_LABEL_WAIT_CHAR_COUNT;
	lvgl.advance(track_width * 25U);
	CHECK(marquee->rendered_phase() == 0U);
	CHECK(marquee->frame_revision() == initial_revision);
	// One late callback jumps to the absolute 40 px/s phase instead of replaying
	// five overdue frames.
	lvgl.advance(550);
	CHECK(marquee->rendered_phase() == 22U);
	CHECK(marquee->frame_revision() == initial_revision + 1U);
	CHECK(object_coordinate_x(marquee->window_image()) == image_x);
	CHECK(object_coordinate_x(marquee->get_obj()) == viewport_x);
	CHECK(lv_image_get_src(marquee->window_image()) == source);
	lvgl.advance(100);
	CHECK(marquee->rendered_phase() == 26U);
	CHECK(marquee->frame_revision() == initial_revision + 2U);
	CHECK(motion_events.style_changed == 0U);
	CHECK(motion_events.layout_changed == 0U);

	int setup_clicks = 0;
	int back_clicks = 0;
	page.on_wifi_config_click = [&setup_clicks]() {
		++setup_clicks;
	};
	page.on_back_click = [&back_clicks]() {
		++back_clicks;
	};
	lv_obj_t *const action_row = ui::WifiPageTestAccess::action_row(page);
	lv_obj_t *const scroll_content =
		ui::WifiPageTestAccess::scroll_content(page);
	REQUIRE(action_row != nullptr);
	REQUIRE(scroll_content != nullptr);
	REQUIRE(lv_obj_get_child_count(action_row) == 3U);
	REQUIRE(lv_obj_get_child_count(scroll_content) == 3U);
	lv_obj_t *const new_connection = lv_obj_get_child(action_row, 1);
	lv_obj_t *const back_button = lv_obj_get_child(scroll_content, 0);
	REQUIRE(new_connection != nullptr);
	REQUIRE(back_button != nullptr);
	CHECK(marquee->animation_running());
	lvgl.press(new_connection);
	CHECK_FALSE(marquee->animation_running());
	lvgl.release();
	CHECK(setup_clicks == 1);
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(500);
	CHECK(marquee->animation_running());
	lvgl.press(back_button);
	CHECK_FALSE(marquee->animation_running());
	lvgl.release();
	CHECK(back_clicks == 1);
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(500);
	CHECK(marquee->animation_running());

	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
		long_ssid, "198.51.100.8", true, false, {}, {});
	CHECK(requests.size() == 1U);
	CHECK(ui::WifiPageTestAccess::ssid_marquee(page) == marquee);
	CHECK(lv_image_get_src(marquee->window_image()) == source);

	const uint32_t paused_phase = marquee->rendered_phase();
	const uint64_t paused_revision = marquee->frame_revision();
	page.on_pause();
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(500);
	CHECK(marquee->rendered_phase() == paused_phase);
	CHECK(marquee->frame_revision() == paused_revision);
	page.on_resume();
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(500);
	CHECK(marquee->animation_running());
	page.on_exit();
	CHECK_FALSE(marquee->animation_running());
	CHECK(ui::WifiPageTestAccess::ssid_marquee(page) == marquee);
	CHECK(lv_image_get_src(marquee->window_image()) == source);
	CHECK(invalidations == 0U);
	page.on_enter();
	CHECK(requests.size() == 1U);
	CHECK(ui::WifiPageTestAccess::ssid_marquee(page) == marquee);
	CHECK_FALSE(marquee->animation_running());
	lvgl.advance(500);
	CHECK(marquee->animation_running());
	page.destroy();
	CHECK(invalidations == 1U);
}

TEST_CASE("Wi-Fi main remembers a TinyTTF worker failure until SSID changes",
	  "[ui][wifi][ssid]")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiPage page(&runtime_font, true, 128,
			  ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lvgl.screen());
	page.on_enter();
	std::vector<ui::SsidRasterRequest> requests;
	std::size_t invalidations = 0;
	page.on_ssid_raster_prepare = [&requests](ui::SsidRasterRequest request) {
		requests.push_back(std::move(request));
		return true;
	};
	page.on_ssid_raster_invalidate = [&invalidations](uint64_t) {
		++invalidations;
	};
	const std::string long_ssid =
		"A TinyTTF raster failure must still scroll through compiled fallback";
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
					    long_ssid, "192.0.2.8",
					    true, false, {}, {});
	REQUIRE(requests.size() == 1U);
	page.accept_prepared_ssid_raster(
		{.identity = requests.front().identity, .raster = nullptr});
	CHECK(ui::WifiPageTestAccess::has_deferred_prepared_ssid_raster(page));
	lvgl.advance(500);

	lv_obj_t *const fallback_label = ssid_fallback_label(page);
	lv_obj_t *const canvas = ssid_value_canvas(page);
	REQUIRE(fallback_label != nullptr);
	REQUIRE(canvas != nullptr);
	CHECK(ui::WifiPageTestAccess::image_cache_failed(page));
	CHECK(ui::WifiPageTestAccess::raster_failed(page));
	CHECK_FALSE(ui::WifiPageTestAccess::raster_request_pending(page));
	CHECK(std::string_view(lv_label_get_text(fallback_label)) ==
	      long_ssid);
	CHECK(lv_obj_get_style_text_font(fallback_label, LV_PART_MAIN) ==
	      ui::compiled_wifi_ssid_font());
	CHECK(lv_label_get_long_mode(fallback_label) ==
	      LV_LABEL_LONG_SCROLL_CIRCULAR);
	lv_anim_t *animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(animation != nullptr);
	const int32_t initial_offset = animation->current_value;
	lvgl.advance(500);
	animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(animation != nullptr);
	CHECK(animation->current_value != initial_offset);
	CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(visible_tree_uses_font(page.get_container(), &runtime_font));
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
		long_ssid, "192.0.2.9", true, false, {}, {});
	CHECK(requests.size() == 1U);
	page.on_pause();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	page.on_resume();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	lvgl.advance(500);
	CHECK(lv_anim_get(fallback_label, nullptr) != nullptr);
	page.on_exit();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	CHECK(invalidations == 0U);
	page.on_enter();
	CHECK(requests.size() == 1U);
	lvgl.advance(500);
	page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {}, false,
		"A different SSID may request a fresh worker raster", "192.0.2.9",
		true, false, {}, {});
	CHECK(requests.size() == 2U);
	page.destroy();
}

TEST_CASE("SConA keeps Montserrat-20 as its default font", "[ui][components]")
{
	test_support::LvglTestContext lvgl;
	ui::SConA control(lv_screen_active(), "Default");
	control.create();

	REQUIRE(control.get_obj() != nullptr);
	lv_obj_t *label = lv_obj_get_child(control.get_obj(), 0);
	REQUIRE(label != nullptr);
	CHECK(lv_obj_get_style_text_font(label, LV_PART_MAIN) ==
	      &lv_font_montserrat_20);

	control.destroy();
}

TEST_CASE("SConA keeps a copy of single text supplied by the caller",
	  "[ui][components]")
{
	test_support::LvglTestContext lvgl;
	std::string initial_text = "Original";
	ui::SConA control(lvgl.screen(), initial_text.c_str());
	initial_text = "A replacement that invalidates the original c_str pointer";
	control.create();

	REQUIRE(control.get_obj() != nullptr);
	CHECK(test_support::visible_direct_text(control.get_obj()) == "Original");

	control.destroy();
}

TEST_CASE("Wi-Fi main SSID applies the scan display policy without changing raw state",
	  "[ui][wifi][ssid]")
{
	test_support::LvglTestContext lvgl;
	ui::WifiPage page;
	page.create(lv_screen_active());

	lv_obj_t *container = page.get_container();
	REQUIRE(container != nullptr);
	REQUIRE(ssid_fallback_label(page) != nullptr);

	const std::vector<std::string> raw_values {
		"Office WiFi",
		"Emoji 😀 network",
		"Control\nnetwork",
		"Malformed \xF0\x28\x8C\x28 network",
		"شبكة",
	};
	for (const std::string &raw : raw_values) {
		page.set_runtime_wifi_state(ui::WifiPageVisualState::Connected, {},
					   false, raw, {}, true, false, {}, {});
		const ui::WifiSsidDisplay expected = ui::make_wifi_ssid_display(raw);
		const std::string expected_text =
			expected.visible ? expected.text : "--";
		CHECK(ui::WifiPageTestAccess::display_ssid(page) == expected_text);
		CHECK(ui::WifiPageTestAccess::raw_ssid(page) == raw);
	}

	page.destroy();
}

TEST_CASE("Wi-Fi action rows expose the correct visible setup entry",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	fixture.apply(ui::WifiPageVisualState::Off, "", false);
	CHECK(is_hidden(fixture.cached_network_button()));
	CHECK(is_hidden(fixture.new_connection_button()));
	CHECK_FALSE(is_hidden(fixture.open_setup_button()));
	CHECK(is_clickable(fixture.open_setup_button()));
	CHECK(test_support::visible_direct_text(fixture.open_setup_button()) ==
	      "Connect Wi-Fi ");
	lv_obj_t *open_setup_label =
		test_support::find_visible_direct_label(fixture.open_setup_button());
	REQUIRE(open_setup_label != nullptr);
	CHECK(lv_obj_get_style_text_font(open_setup_label, LV_PART_MAIN) ==
	      &MontserratBold_20);
	CHECK(contains_image_source(fixture.open_setup_button(), &ci_add_plus));
	CHECK_FALSE(contains_image_source(fixture.open_setup_button(), &wifi_new));
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));

	fixture.click(fixture.open_setup_button());
	CHECK(fixture.setup_navigation_requests() == 1);
	CHECK(fixture.enable_requests() == 0);
	CHECK(fixture.disable_requests() == 0);

	fixture.apply(ui::WifiPageVisualState::Off, "", true);
	CHECK_FALSE(is_hidden(fixture.cached_network_button()));
	CHECK_FALSE(is_hidden(fixture.new_connection_button()));
	CHECK(is_hidden(fixture.open_setup_button()));
	CHECK(is_clickable(fixture.cached_network_button()));
	CHECK(contains_image_source(fixture.new_connection_button(), &ci_add_plus));
	CHECK_FALSE(contains_image_source(fixture.new_connection_button(), &wifi_new));
	CHECK(has_background_color(fixture.new_connection_button(), 0x2B7FFF));
	fixture.click(fixture.cached_network_button());
	CHECK(fixture.enable_requests() == 1);
}

TEST_CASE(
	"Wi-Fi connected without a saved config keeps paired actions visible",
	"[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	// Station is connected even though the saved-network probe has not yet
	// returned true (e.g. its non-ASCII SSID defeated the JSON parser). The
	// action row must not fall back to the unconfigured Connect Wi-Fi entry.
	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", false);

	CHECK_FALSE(is_hidden(fixture.cached_network_button()));
	CHECK_FALSE(is_hidden(fixture.new_connection_button()));
	CHECK(is_hidden(fixture.open_setup_button()));
	CHECK(is_clickable(fixture.cached_network_button()));
	CHECK(is_clickable(fixture.new_connection_button()));
	CHECK_FALSE(is_clickable(fixture.open_setup_button()));
	CHECK(contains_image_source(fixture.new_connection_button(), &ci_add_plus));
	CHECK_FALSE(contains_image_source(fixture.new_connection_button(), &wifi_new));
	CHECK(has_background_color(fixture.new_connection_button(), 0x2B7FFF));
	CHECK(fixture.ssid_text() == "Office WiFi");

	// The right New button still enters the existing Via setup flow.
	fixture.click(fixture.new_connection_button());
	CHECK(fixture.setup_navigation_requests() == 1);
	CHECK(fixture.enable_requests() == 0);
	CHECK(fixture.disable_requests() == 0);

	// The left cached-network button issues the existing disable request
	// for the connected Station instead of being a dead control.
	fixture.click(fixture.cached_network_button());
	CHECK(fixture.disable_requests() == 1);
	CHECK(fixture.enable_requests() == 0);
}

TEST_CASE("Wi-Fi cached-network action keeps saved configuration after turn-off",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true);
	fixture.apply(ui::WifiPageVisualState::Off, "", true);

	REQUIRE(is_clickable(fixture.cached_network_button()));
	fixture.click(fixture.cached_network_button());
	CHECK(fixture.enable_requests() == 1);
}

TEST_CASE("Wi-Fi action row remains centered below the two-row information group",
	  "[ui][wifi][navigation][layout]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);
	fixture.apply(ui::WifiPageVisualState::Off, "", true);

	lv_obj_t *action_row = lv_obj_get_parent(fixture.cached_network_button());
	REQUIRE(action_row != nullptr);
	lv_obj_t *logical_canvas = lv_obj_get_parent(action_row);
	REQUIRE(logical_canvas != nullptr);
	lv_obj_t *information_group = fixture.information_group();
	lv_obj_t *ssid_row = fixture.ssid_info_row()->get_obj();
	lv_obj_t *ip_row = fixture.ip_label()->get_obj();
	REQUIRE(information_group != nullptr);
	REQUIRE(ssid_row != nullptr);
	REQUIRE(ip_row != nullptr);
	REQUIRE(lv_obj_get_parent(information_group) == logical_canvas);
	REQUIRE(lv_obj_get_parent(ssid_row) == information_group);
	REQUIRE(lv_obj_get_parent(ip_row) == information_group);
	lv_area_t action_row_coords {};
	lv_area_t logical_canvas_coords {};
	lv_area_t information_group_coords {};
	lv_area_t ssid_row_coords {};
	lv_area_t ip_row_coords {};
	lv_obj_get_coords(action_row, &action_row_coords);
	lv_obj_get_coords(logical_canvas, &logical_canvas_coords);
	lv_obj_get_coords(information_group, &information_group_coords);
	lv_obj_get_coords(ssid_row, &ssid_row_coords);
	lv_obj_get_coords(ip_row, &ip_row_coords);
	CHECK(lv_obj_get_width(logical_canvas) == 270);
	CHECK(lv_obj_get_height(logical_canvas) == 240);
	CHECK(lv_obj_get_width(information_group) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(ssid_row) == ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_get_height(ip_row) == ui::SConA::HEIGHT);
	CHECK(information_group_coords.x1 - logical_canvas_coords.x1 == 13);
	CHECK(information_group_coords.y1 - logical_canvas_coords.y1 == 76);
	CHECK(ssid_row_coords.x1 == information_group_coords.x1);
	CHECK(ssid_row_coords.y1 == information_group_coords.y1);
	CHECK(ip_row_coords.x1 == information_group_coords.x1);
	CHECK(ip_row_coords.y1 - information_group_coords.y1 ==
	      ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_get_width(action_row) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(action_row) == ui::BConB::HEIGHT);
	// Flex children are positioned relative to the parent's content box, which
	// begins after SubPageBase's 13px left padding. Compare in canvas space.
	CHECK(action_row_coords.x1 - logical_canvas_coords.x1 == 20);
	CHECK(action_row_coords.x1 - logical_canvas_coords.x1 ==
	      (lv_obj_get_width(logical_canvas) - lv_obj_get_width(action_row)) /
		      2);
	CHECK(action_row_coords.y1 - logical_canvas_coords.y1 == 164);
	CHECK(lv_obj_get_x(fixture.new_connection_button()) -
	      lv_obj_get_x(fixture.cached_network_button()) ==
	      ui::BConB::WIDTH + 30);

	fixture.apply(ui::WifiPageVisualState::Off, "", false);
	lv_area_t open_setup_coords {};
	lv_obj_get_coords(action_row, &action_row_coords);
	lv_obj_get_coords(fixture.open_setup_button(), &open_setup_coords);
	CHECK_FALSE(is_hidden(fixture.open_setup_button()));
	CHECK(open_setup_coords.x1 == action_row_coords.x1);
	CHECK(open_setup_coords.y1 == action_row_coords.y1);
	CHECK(lv_obj_get_width(fixture.open_setup_button()) ==
	      lv_obj_get_width(action_row));
	CHECK(lv_obj_get_height(fixture.open_setup_button()) ==
	      lv_obj_get_height(action_row));
}

TEST_CASE("Wi-Fi cached-network action only switches confirmed connection state",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true);
	fixture.click(fixture.cached_network_button());
	CHECK(fixture.disable_requests() == 1);
	CHECK(fixture.enable_requests() == 0);
}

TEST_CASE("Wi-Fi New Connection enters setup without reset or cached enable",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true);
	fixture.click(fixture.new_connection_button());

	CHECK(fixture.setup_navigation_requests() == 1);
	CHECK(fixture.enable_requests() == 0);
	CHECK(fixture.disable_requests() == 0);
}

TEST_CASE("Wi-Fi Connect Wi-Fi setup respects busy, pending, and connecting locks",
	  "[ui][wifi][navigation]")
{
	WifiPageNavigationFixture fixture;
	require_controls(fixture);

	fixture.apply(ui::WifiPageVisualState::Off, "", false);
	fixture.set_busy(true);
	CHECK_FALSE(is_hidden(fixture.open_setup_button()));
	CHECK_FALSE(is_clickable(fixture.open_setup_button()));
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));
	fixture.click(fixture.open_setup_button());
	CHECK(fixture.setup_navigation_requests() == 0);

	fixture.set_busy(false);
	CHECK(is_clickable(fixture.open_setup_button()));
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));

	fixture.apply(ui::WifiPageVisualState::Connecting, "", false);
	CHECK_FALSE(is_hidden(fixture.open_setup_button()));
	CHECK_FALSE(is_clickable(fixture.open_setup_button()));
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));
	fixture.click(fixture.open_setup_button());
	CHECK(fixture.setup_navigation_requests() == 0);

	fixture.apply(ui::WifiPageVisualState::ApConnect, "", false, false, true);
	REQUIRE(fixture.qr_popup() != nullptr);
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));
	fixture.apply(ui::WifiPageVisualState::ApConnect, "", false);
	CHECK(fixture.qr_popup() == nullptr);
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));

	fixture.apply(ui::WifiPageVisualState::Connected, "Office WiFi", true);
	fixture.click(fixture.cached_network_button());
	CHECK(fixture.disable_requests() == 1);
	fixture.apply(ui::WifiPageVisualState::Off, "", false, true);
	fixture.set_busy(false);
	CHECK_FALSE(is_hidden(fixture.open_setup_button()));
	CHECK_FALSE(is_clickable(fixture.open_setup_button()));
	CHECK(has_background_color(fixture.open_setup_button(), 0x2B7FFF));
	fixture.click(fixture.open_setup_button());
	CHECK(fixture.setup_navigation_requests() == 0);
}
