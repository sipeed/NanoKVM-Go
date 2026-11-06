#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/page_registry.hpp"
#include "ui/ssid_info_row.hpp"
#include "ui/ssid_raster_marquee.hpp"
#include "ui/ssid_text_image_cache.hpp"
#include "ui/wifi_page.hpp"
#include "ui/wifi_ssid_font.hpp"
#include "ui/wifi_ssid_page.hpp"

extern "C" {
LV_FONT_DECLARE(wifi_zh_font);
}

namespace ui
{

class WifiSsidPageTestAccess final {
    public:
	static std::size_t option_count(const WifiSsidPage &page)
	{
		return page._options.size();
	}

	static const std::string &label(const WifiSsidPage &page, std::size_t index)
	{
		return page._options.at(index).label;
	}

	static const std::string &ssid(const WifiSsidPage &page, std::size_t index)
	{
		return page._options.at(index).ssid;
	}

	static std::vector<std::string> foreground_text_labels(
		const WifiSsidPage &page)
	{
		return page.foreground_text_labels();
	}

	static std::vector<std::string> speculative_text_labels(
		const WifiSsidPage &page)
	{
		return page.speculative_text_labels();
	}

	static void select(WifiSsidPage &page, uint32_t index)
	{
		page._selected_index = index;
	}

	static void activate(WifiSsidPage &page)
	{
		page.activate_selected_option();
	}

	static lv_obj_t *roller(const WifiSsidPage &page)
	{
		return page._roller;
	}

	static const lv_font_t *ssid_font(const WifiSsidPage &page)
	{
		return page._ssid_font;
	}

	static const lv_font_t *visible_ssid_font(const WifiSsidPage &page)
	{
		return page._visible_ssid_font;
	}

	static SsidFontRenderMode ssid_font_render_mode(
		const WifiSsidPage &page)
	{
		return page._ssid_font_render_mode;
	}

	static const lv_font_t *selected_marquee_font(const WifiSsidPage &page)
	{
		return page._selected_marquee_label == nullptr ? nullptr :
			lv_obj_get_style_text_font(page._selected_marquee_label,
							  LV_PART_MAIN);
	}

	static const lv_font_t *cache_font(const WifiSsidPage &page)
	{
		return page._text_image_cache == nullptr ? nullptr :
			page._text_image_cache->_font;
	}

	static lv_obj_t *cache_raster_host(const WifiSsidPage &page)
	{
		return page._text_image_cache == nullptr ? nullptr :
			page._text_image_cache->_raster_host;
	}

	static lv_obj_t *cache_raster_label(const WifiSsidPage &page)
	{
		return page._text_image_cache == nullptr ? nullptr :
			page._text_image_cache->_raster_label;
	}

	static bool text_image_cache_enabled(const WifiSsidPage &page)
	{
		return page.text_image_cache_enabled();
	}

	static const SsidTextImageArena *text_image_arena(
		const WifiSsidPage &page)
	{
		return page._text_image_arena.get();
	}

	static bool cache_activation_failed(const WifiSsidPage &page)
	{
		return page._cache_activation_failed;
	}

	static void render_cache_tick(WifiSsidPage &page)
	{
		page.render_next_text_image();
	}

	static void service_cache_timer_tick(WifiSsidPage &page)
	{
		if (page.service_one_selected_raster_work()) {
			return;
		}
		page.render_next_text_image();
	}

	static const lv_draw_buf_t *cached_text_source(WifiSsidPage &page,
							 uint32_t option_index)
	{
		if (page._text_image_cache == nullptr ||
		    option_index >= page._options.size()) {
			return nullptr;
		}
		return page._text_image_cache->source_for(
			page._options[option_index].label);
	}

	static bool cache_foreground_pending(const WifiSsidPage &page)
	{
		return page._text_image_cache != nullptr &&
		       page._text_image_cache->has_foreground_pending();
	}

	static bool cached_text_row_area(const WifiSsidPage &page,
						 uint32_t option_index, lv_area_t *area)
	{
		return page.cached_text_row_area(option_index, area);
	}

	static lv_obj_t *roller_label(const WifiSsidPage &page)
	{
		return page.roller_label();
	}

	static lv_obj_t *selected_marquee_overlay(const WifiSsidPage &page)
	{
		return page._selected_marquee_overlay;
	}

	static lv_obj_t *selected_marquee_label(const WifiSsidPage &page)
	{
		return page._selected_marquee_label;
	}

	static bool selected_marquee_visible(const WifiSsidPage &page)
	{
		return page.selected_marquee_visible();
	}

	static const SsidRasterMarquee *selected_raster_marquee(
		const WifiSsidPage &page)
	{
		return page._selected_raster_marquee.get();
	}

	static bool selected_raster_work_pending(const WifiSsidPage &page)
	{
		return page.selected_raster_work_pending();
	}

	static std::string_view selected_marquee_pending_text(
		const WifiSsidPage &page)
	{
		return page._selected_marquee_pending_text;
	}

	static bool cache_timer_running(const WifiSsidPage &page)
	{
		return page._cache_timer != nullptr;
	}

	static bool roller_visible(const WifiSsidPage &page)
	{
		return page._roller != nullptr &&
		       !lv_obj_has_flag(page._roller, LV_OBJ_FLAG_HIDDEN);
	}

	static bool loading_visible(const WifiSsidPage &page)
	{
		return page._loading_box != nullptr &&
		       !lv_obj_has_flag(page._loading_box, LV_OBJ_FLAG_HIDDEN);
	}

	static uint16_t loading_rotation(const WifiSsidPage &page)
	{
		return page._loading_rotation;
	}

	static std::string_view loading_phase_name(const WifiSsidPage &page)
	{
		switch (page._loading_phase) {
		case WifiSsidPage::LoadingPhase::Idle:
			return "Idle";
		case WifiSsidPage::LoadingPhase::Climbing:
			return "Climbing";
		case WifiSsidPage::LoadingPhase::Waiting:
			return "Waiting";
		case WifiSsidPage::LoadingPhase::Rendering:
			return "Rendering";
		case WifiSsidPage::LoadingPhase::Finishing:
			return "Finishing";
		case WifiSsidPage::LoadingPhase::Done:
			return "Done";
		}
		return {};
	}

	static uint8_t loading_progress(const WifiSsidPage &page)
	{
		return page._loading_progress;
	}

	static uint32_t loading_start_tick(const WifiSsidPage &page)
	{
		return page._loading_start_tick;
	}

	static uint32_t loading_ready_tick(const WifiSsidPage &page)
	{
		return page._loading_ready_tick;
	}

	static int32_t loading_arc_end_angle(const WifiSsidPage &page)
	{
		return page._loading_arc == nullptr ? -1 :
			static_cast<int32_t>(lv_arc_get_angle_end(page._loading_arc));
	}

	static void refresh_view(WifiSsidPage &page)
	{
		page.update_view();
	}

	static void finish_cache_content(WifiSsidPage &page)
	{
		page.stop_cache_timer();
		if (page._text_image_cache != nullptr) {
			page._text_image_cache->cancel_pending();
		}
		page.mark_cache_content_ready();
	}

	static bool cache_destroyed(const WifiSsidPage &page)
	{
		return page._text_image_cache == nullptr;
	}

	static uint8_t cache_preload_phase(const WifiSsidPage &page)
	{
		return static_cast<uint8_t>(page._cache_preload_phase);
	}

	static bool initial_cache_preload_active(const WifiSsidPage &page)
	{
		return page.initial_cache_preload_active();
	}

	static bool loading_timer_running(const WifiSsidPage &page)
	{
		return page._loading_timer != nullptr;
	}

	static bool cache_content_ready(const WifiSsidPage &page)
	{
		return page._cache_content_ready;
	}

	static bool cache_has_pending(const WifiSsidPage &page)
	{
		return page._text_image_cache != nullptr &&
		       page._text_image_cache->has_pending();
	}

	static void queue_fixed_raster(WifiSsidPage &page, std::string label)
	{
		if (page._text_image_cache == nullptr) {
			return;
		}
		page._text_image_cache->request_labels({ std::move(label) }, {});
		page.start_cache_timer();
	}

	static uint32_t max_loading_presentation_gap_ms(const WifiSsidPage &page)
	{
		return page._max_loading_presentation_gap_ms;
	}
};

} // namespace ui

namespace
{

lv_obj_t *ssid_value_canvas(lv_obj_t *container)
{
	if (container == nullptr) {
		return nullptr;
	}
	for (uint32_t index = 0; index < lv_obj_get_child_count(container); ++index) {
		lv_obj_t *child = lv_obj_get_child(container, index);
		if (lv_obj_check_type(child, &lv_canvas_class) &&
		    lv_obj_get_width(child) == ui::SsidInfoRow::VALUE_WIDTH &&
		    lv_obj_get_height(child) == ui::SsidInfoRow::HEIGHT) {
			return child;
		}
		if (lv_obj_t *const match = ssid_value_canvas(child); match != nullptr) {
			return match;
		}
	}
	return nullptr;
}

bool areas_intersect(const lv_area_t &left, const lv_area_t &right)
{
	return left.x1 <= right.x2 && right.x1 <= left.x2 &&
	       left.y1 <= right.y2 && right.y1 <= left.y2;
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

void finish_cache_loading(test_support::LvglTestContext &lvgl,
				  ui::WifiSsidPage &page)
{
	for (int index = 0;
	     index < 64 && !ui::WifiSsidPageTestAccess::cache_content_ready(page);
	     ++index) {
		ui::WifiSsidPageTestAccess::render_cache_tick(page);
	}
	REQUIRE(ui::WifiSsidPageTestAccess::cache_content_ready(page));
	lvgl.advance(250);
}

void check_selected_marquee_geometry(ui::WifiSsidPage &page,
					     uint32_t selected_index)
{
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	lv_obj_t *overlay =
		ui::WifiSsidPageTestAccess::selected_marquee_overlay(page);
	REQUIRE(roller != nullptr);
	REQUIRE(overlay != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));

	lv_obj_update_layout(roller);
	lv_area_t overlay_area {};
	lv_area_t selected_row_area {};
	lv_obj_get_coords(overlay, &overlay_area);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
		page, selected_index, &selected_row_area));
	CHECK(overlay_area.x1 == selected_row_area.x1);
	CHECK(overlay_area.y1 == selected_row_area.y1);
	CHECK(overlay_area.x2 == selected_row_area.x2);
	CHECK(overlay_area.y2 == selected_row_area.y2);

	if (selected_index > 0U) {
		lv_area_t previous_row_area {};
		REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
			page, selected_index - 1U, &previous_row_area));
		CHECK_FALSE(areas_intersect(overlay_area, previous_row_area));
	}
	if (selected_index + 1U <
	    ui::WifiSsidPageTestAccess::option_count(page)) {
		lv_area_t next_row_area {};
		REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
			page, selected_index + 1U, &next_row_area));
		CHECK_FALSE(areas_intersect(overlay_area, next_row_area));
	}
}

} // namespace

TEST_CASE("Wi-Fi SSID page hides unsupported labels and selects raw SSIDs")
{
	ui::WifiSsidPage page;
	const std::string emoji_ssid = "  Café 😀  ";
	const std::string hidden_ssid = "شبكة";
	std::string invalid_ssid;
	invalid_ssid.push_back(static_cast<char>(0xFF));
	invalid_ssid += "A";
	page.set_scan_results({
		{ .ssid = emoji_ssid },
		{ .ssid = hidden_ssid },
		{ .ssid = invalid_ssid },
	});

	REQUIRE(ui::WifiSsidPageTestAccess::option_count(page) == 3U);
	CHECK(ui::WifiSsidPageTestAccess::label(page, 0) == "Manual SSID");
	CHECK(ui::WifiSsidPageTestAccess::label(page, 1) == "  Café □  ");
	CHECK(ui::WifiSsidPageTestAccess::ssid(page, 1) == emoji_ssid);
	CHECK(ui::WifiSsidPageTestAccess::label(page, 2) == "\\xFF\\x41");
	CHECK(ui::WifiSsidPageTestAccess::ssid(page, 2) == invalid_ssid);

	std::string selected_ssid;
	page.on_ssid_selected = [&selected_ssid](std::string_view ssid) {
		selected_ssid.assign(ssid);
	};
	ui::WifiSsidPageTestAccess::select(page, 1U);
	ui::WifiSsidPageTestAccess::activate(page);
	CHECK(selected_ssid == emoji_ssid);

	ui::WifiSsidPageTestAccess::select(page, 2U);
	ui::WifiSsidPageTestAccess::activate(page);
	CHECK(selected_ssid == invalid_ssid);
}

TEST_CASE("Wi-Fi SSID page applies its injected runtime font to the roller")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20);
	page.create(lv_screen_active());

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_MAIN) ==
	      &lv_font_montserrat_20);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_SELECTED) ==
	      &lv_font_montserrat_20);

	page.destroy();
}

TEST_CASE("Wi-Fi SSID native, marquee, and cache paths share the resolved font")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;

	ui::WifiSsidPage native_page(&runtime_font, 0, true);
	native_page.create(lv_screen_active());
	lv_obj_t *native_roller = ui::WifiSsidPageTestAccess::roller(native_page);
	REQUIRE(native_roller != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font(native_page) == &runtime_font);
	CHECK(lv_obj_get_style_text_font(native_roller, LV_PART_MAIN) ==
	      &runtime_font);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_font(native_page) ==
	      &runtime_font);
	native_page.destroy();

	ui::WifiSsidPage cache_page(&runtime_font, 128, true);
	cache_page.create(lv_screen_active());
	cache_page.set_scan_results({ { .ssid = "A long network name" } });
	cache_page.on_enter();
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(cache_page));
	CHECK(ui::WifiSsidPageTestAccess::cache_font(cache_page) == &runtime_font);
	cache_page.destroy();
}

TEST_CASE("Wi-Fi SSID TinyTTF mode confines the runtime font to its hidden cache")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	const lv_font_t *const fallback = ui::compiled_wifi_ssid_font();
	REQUIRE(fallback != nullptr);

	ui::WifiSsidPage native_page(
		&runtime_font, 0, true, ui::SsidFontRenderMode::RuntimeTinyTtf);
	native_page.create(lv_screen_active());
	lv_obj_t *native_roller = ui::WifiSsidPageTestAccess::roller(native_page);
	REQUIRE(native_roller != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font(native_page) == &runtime_font);
	CHECK(ui::WifiSsidPageTestAccess::visible_ssid_font(native_page) ==
	      fallback);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font_render_mode(native_page) ==
	      ui::SsidFontRenderMode::RuntimeTinyTtf);
	CHECK(lv_obj_get_style_text_font(native_roller, LV_PART_MAIN) == fallback);
	CHECK(lv_obj_get_style_text_font(native_roller, LV_PART_SELECTED) ==
	      fallback);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_overlay(native_page) !=
		nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_font(native_page) ==
	      fallback);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(native_page));
	native_page.destroy();

	ui::WifiSsidPage cache_page(
		&runtime_font, 128, true, ui::SsidFontRenderMode::RuntimeTinyTtf);
	cache_page.create(lv_screen_active());
	cache_page.set_scan_results({ { .ssid = "A long network name" } });
	cache_page.on_enter();
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(cache_page));
	CHECK(ui::WifiSsidPageTestAccess::cache_font(cache_page) == &runtime_font);
	lv_obj_t *cache_roller = ui::WifiSsidPageTestAccess::roller(cache_page);
	REQUIRE(cache_roller != nullptr);
	CHECK(lv_obj_get_style_text_font(cache_roller, LV_PART_MAIN) == fallback);
	CHECK(lv_obj_get_style_text_font(cache_roller, LV_PART_SELECTED) ==
	      fallback);
	lv_obj_t *raster_host =
		ui::WifiSsidPageTestAccess::cache_raster_host(cache_page);
	lv_obj_t *raster_label =
		ui::WifiSsidPageTestAccess::cache_raster_label(cache_page);
	REQUIRE(raster_host != nullptr);
	REQUIRE(raster_label != nullptr);
	CHECK(lv_obj_has_flag(raster_host, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_style_text_font(raster_label, LV_PART_MAIN) ==
	      &runtime_font);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_overlay(cache_page) !=
		nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_font(cache_page) ==
	      fallback);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(cache_page));
	cache_page.destroy();
}

TEST_CASE("Wi-Fi SSID TinyTTF cache failure retains a compiled scroll fallback")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	const lv_font_t *const fallback = ui::compiled_wifi_ssid_font();
	ui::WifiSsidPage page(
		&runtime_font, 1U, true, ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lv_screen_active());
	const std::string long_ssid =
		"A failed TinyTTF raster still uses the compiled scrolling fallback";
	page.set_scan_results({ { .ssid = long_ssid } });
	page.on_enter();

	REQUIRE(ui::WifiSsidPageTestAccess::cache_activation_failed(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_MAIN) == fallback);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_SELECTED) == fallback);
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_overlay(page) != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	lv_obj_t *fallback_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(fallback_label != nullptr);
	CHECK(std::string_view(lv_label_get_text(fallback_label)) == long_ssid);
	CHECK(lv_label_get_long_mode(fallback_label) ==
	      LV_LABEL_LONG_SCROLL_CIRCULAR);
	lv_anim_t *fallback_animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(fallback_animation != nullptr);
	const int32_t fallback_offset = fallback_animation->current_value;
	lvgl.advance(500);
	fallback_animation = lv_anim_get(fallback_label, nullptr);
	REQUIRE(fallback_animation != nullptr);
	CHECK(fallback_animation->current_value != fallback_offset);
	lvgl.press(roller);
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	lvgl.release();
	CHECK(lv_anim_get(fallback_label, nullptr) != nullptr);
	page.on_pause();
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID pages share one compiled fallback font")
{
	test_support::LvglTestContext lvgl;
	const lv_font_t *const fallback = ui::resolve_wifi_ssid_font(nullptr);
	REQUIRE(fallback != nullptr);
	CHECK(fallback == ui::resolve_wifi_ssid_font(nullptr));
	CHECK(fallback->fallback == &wifi_zh_font);

	ui::PageRegistry registry(nullptr);
	registry.create(lv_screen_active());
	ui::WifiPage *wifi_page = registry.wifi_page();
	ui::WifiSsidPage *ssid_page = registry.wifi_ssid_page();
	REQUIRE(wifi_page != nullptr);
	REQUIRE(ssid_page != nullptr);

	lv_obj_t *wifi_container = wifi_page->get_container();
	REQUIRE(wifi_container != nullptr);
	lv_obj_t *wifi_ssid_canvas = ssid_value_canvas(wifi_container);
	REQUIRE(wifi_ssid_canvas != nullptr);
	REQUIRE(lv_canvas_get_draw_buf(wifi_ssid_canvas) != nullptr);
	lv_obj_t *ssid_roller = ui::WifiSsidPageTestAccess::roller(*ssid_page);
	REQUIRE(ssid_roller != nullptr);
	CHECK(lv_obj_get_width(wifi_ssid_canvas) == ui::SsidInfoRow::VALUE_WIDTH);
	CHECK(lv_obj_get_height(wifi_ssid_canvas) == ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_get_style_text_font(ssid_roller, LV_PART_MAIN) == fallback);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font(*ssid_page) == fallback);

	registry.shutdown();
}

TEST_CASE("Wi-Fi registry propagates an injected SSID font to both pages")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::PageRegistry registry(&runtime_font);
	registry.create(lv_screen_active());

	ui::WifiPage *wifi_page = registry.wifi_page();
	ui::WifiSsidPage *ssid_page = registry.wifi_ssid_page();
	REQUIRE(wifi_page != nullptr);
	REQUIRE(ssid_page != nullptr);

	lv_obj_t *wifi_container = wifi_page->get_container();
	REQUIRE(wifi_container != nullptr);
	lv_obj_t *wifi_ssid_canvas = ssid_value_canvas(wifi_container);
	REQUIRE(wifi_ssid_canvas != nullptr);
	lv_obj_t *ssid_roller = ui::WifiSsidPageTestAccess::roller(*ssid_page);
	REQUIRE(ssid_roller != nullptr);

	CHECK(lv_canvas_get_draw_buf(wifi_ssid_canvas) != nullptr);
	CHECK(lv_obj_get_style_text_font(ssid_roller, LV_PART_MAIN) ==
	      &runtime_font);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font(*ssid_page) == &runtime_font);

	registry.shutdown();
}

TEST_CASE("Wi-Fi SSID page queues every option for cache background work")
{
	ui::WifiSsidPage page;
	std::vector<ui::WifiScanResult> results;
	for (int index = 1; index <= 14; ++index) {
		results.push_back(
			ui::WifiScanResult { .ssid = "Network " + std::to_string(index) });
	}
	page.set_scan_results(results);
	ui::WifiSsidPageTestAccess::select(page, 7U);

	const std::vector<std::string> foreground =
		ui::WifiSsidPageTestAccess::foreground_text_labels(page);
	const std::vector<std::string> speculative =
		ui::WifiSsidPageTestAccess::speculative_text_labels(page);
	REQUIRE(foreground.size() == 3U);
	CHECK(foreground[0] == "Network 7");
	CHECK(foreground[1] == "Network 6");
	CHECK(foreground[2] == "Network 8");
	REQUIRE(speculative.size() == 15U);
	CHECK(speculative.front() == "Manual SSID");
	CHECK(speculative[1] == "Network 1");
	CHECK(speculative[7] == "Network 7");
	CHECK(speculative.back() == "Network 14");
}

TEST_CASE("Wi-Fi SSID page hides signals at or below minus 75 dBm")
{
	ui::WifiSsidPage page;
	page.set_scan_results({
		{ .ssid = "Visible", .signal = -74 },
		{ .ssid = "Threshold", .signal = -75 },
		{ .ssid = "Weak", .signal = -76 },
	});

	REQUIRE(ui::WifiSsidPageTestAccess::option_count(page) == 2U);
	CHECK(ui::WifiSsidPageTestAccess::label(page, 1U) == "Visible");
}

TEST_CASE("Wi-Fi SSID page draws cached text with the roller and stops work when paused")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, true);
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = "First network" },
		{ .ssid = "Second network" },
		{ .ssid = "Third network" },
		{ .ssid = "Fourth network" },
		{ .ssid = "Fifth network" },
		{ .ssid = "Sixth network" },
		{ .ssid = "Seventh network" },
	});
	page.on_enter();

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_MAIN) ==
	      &lv_font_montserrat_20);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_SELECTED) ==
	      &lv_font_montserrat_20);
	// The roller keeps its single internal label; the only additional child is
	// the non-interactive selected-row marquee overlay.
	CHECK(lv_obj_get_child_count(roller) == 2U);
	lv_obj_t *roller_label = ui::WifiSsidPageTestAccess::roller_label(page);
	REQUIRE(roller_label != nullptr);
	CHECK(lv_obj_has_class(roller_label, &lv_label_class));
	CHECK(lv_obj_get_width(roller_label) == lv_obj_get_width(roller));
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_overlay(page) !=
	      nullptr);
	CHECK(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));

	// The first preload tick presents and returns, so LVGL has an opportunity
	// to refresh the arc before a row snapshot runs.
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	const lv_draw_buf_t *manual_source =
		ui::WifiSsidPageTestAccess::cached_text_source(page, 0U);
	CHECK(manual_source == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 2U);

	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	manual_source = ui::WifiSsidPageTestAccess::cached_text_source(page, 0U);
	REQUIRE(manual_source != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::cache_foreground_pending(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));

	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 1U) == nullptr);
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	const lv_draw_buf_t *first_network_source =
		ui::WifiSsidPageTestAccess::cached_text_source(page, 1U);
	REQUIRE(first_network_source != nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_foreground_pending(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	finish_cache_loading(lvgl, page);
	CHECK(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	lv_obj_set_style_anim_duration(roller, 200, LV_PART_MAIN);
	lv_area_t label_before {};
	lv_area_t row_before {};
	lv_obj_get_coords(roller_label, &label_before);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(page, 1U,
									   &row_before));
	lv_roller_set_selected(roller, 2U, LV_ANIM_ON);
	lvgl.advance(40);

	lv_area_t label_during {};
	lv_area_t row_during {};
	lv_obj_get_coords(roller_label, &label_during);
	REQUIRE(label_during.y1 != label_before.y1);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(page, 1U,
									   &row_during));
	CHECK(row_during.y1 - row_before.y1 ==
	      label_during.y1 - label_before.y1);

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	page.destroy();
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));
}

TEST_CASE("Wi-Fi SSID page keeps legacy rotation active in None mode")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::None);
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = "First network" },
		{ .ssid = "Second network" },
		{ .ssid = "Third network" },
		{ .ssid = "Fourth network" },
		{ .ssid = "Fifth network" },
		{ .ssid = "Sixth network" },
		{ .ssid = "Seventh network" },
	});
	page.on_enter();

	REQUIRE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_rotation(page) == 0U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 270);

	lvgl.advance(40);

	CHECK(ui::WifiSsidPageTestAccess::loading_rotation(page) != 0U);
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID Progress cancels an in-flight finish on scan and exit")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	page.create(lv_screen_active());
	page.set_loading();
	page.on_enter();
	lvgl.advance(3810);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Waiting");

	// A warm cache close (nothing left to raster) enters Finishing; a new scan
	// is a session boundary that cancels the in-flight finish and starts a
	// fresh climb without leaving the old result roller visible.
	ui::WifiSsidPageTestAccess::finish_cache_content(page);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Finishing");
	REQUIRE(ui::WifiSsidPageTestAccess::loading_timer_running(page));

	page.set_loading();
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Climbing");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 0U);
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_timer_running(page));

	page.on_exit();
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Idle");
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));
	lvgl.advance(300);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Idle");
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID Progress preserves a scan session through cached results")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	page.create(lv_screen_active());
	page.set_loading();
	page.on_enter();

	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Climbing");
	REQUIRE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	const uint32_t scan_start_tick =
		ui::WifiSsidPageTestAccess::loading_start_tick(page);

	// Scan half: 0 -> 50% at 75% of the former 70%/4000 ms rate. At the former
	// 3000 ms boundary it is still climbing; it reaches 50% near 3810 ms.
	lvgl.advance(3000);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Climbing");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) < 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) < 180);
	lvgl.advance(810);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Waiting");
	REQUIRE(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
		scan_start_tick);

	const std::vector<ui::WifiScanResult> results {
		{ .ssid = "Ready network" },
		{ .ssid = "Background network" },
	};
	page.set_scan_results(results);
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	// Result delivery enters the cache-render half; the session origin tick
	// and the 50% boundary are preserved, not restarted at zero.
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	REQUIRE(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	// A Present-phase tick and a view refresh stay within the same Rendering
	// session; drawn is still zero so progress holds at the 50% boundary.
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	ui::WifiSsidPageTestAccess::refresh_view(page);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);

	uint8_t previous_progress = 0U;
	for (int index = 0;
	     index < 64 && !ui::WifiSsidPageTestAccess::cache_content_ready(page);
	     ++index) {
		ui::WifiSsidPageTestAccess::render_cache_tick(page);
		// Cache-render progress tracks drawn/total and is monotonic.
		const uint8_t progress =
			ui::WifiSsidPageTestAccess::loading_progress(page);
		CHECK(progress >= previous_progress);
		previous_progress = progress;
	}

	REQUIRE(ui::WifiSsidPageTestAccess::cache_content_ready(page));
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Done");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 100U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 360);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	REQUIRE(ui::WifiSsidPageTestAccess::option_count(page) == 3U);

	// A duplicate result publication after Done must not restart the session;
	// mark_cache_content_ready is idempotent and the origin tick is preserved.
	page.set_scan_results(results);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Done");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 100U);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	ui::WifiSsidPageTestAccess::finish_cache_content(page);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Done");
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);

	std::string selected_ssid;
	page.on_ssid_selected = [&selected_ssid](std::string_view ssid) {
		selected_ssid.assign(ssid);
	};
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	ui::WifiSsidPageTestAccess::select(page, 1U);
	ui::WifiSsidPageTestAccess::activate(page);
	CHECK(selected_ssid == "Ready network");

	// An explicit new scan remains the ordinary session boundary.
	page.set_loading();
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Climbing");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 0U);
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.on_pause();
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Idle");
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID Progress snaps to 50% when results arrive mid-scan")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	page.create(lv_screen_active());
	page.set_loading();
	page.on_enter();
	const uint32_t scan_start_tick =
		ui::WifiSsidPageTestAccess::loading_start_tick(page);

	// Deliver results well before the scan half reaches 50%.
	lvgl.advance(1500);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Climbing");
	REQUIRE(ui::WifiSsidPageTestAccess::loading_progress(page) < 50U);

	page.set_scan_results({ { .ssid = "Early network" } });
	// Result delivery snaps the bar forward to the 50% cache-render boundary;
	// it never moves backward and the scan origin tick is preserved.
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page reveals results when image caching is disabled")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 0, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	page.create(lv_screen_active());
	page.set_scan_results({ { .ssid = "Native fallback network" } });
	page.on_enter();

	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID Progress handles an immediately ready warm cache")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	const std::vector<ui::WifiScanResult> results {
		{ .ssid = "Warm cache network" },
	};
	page.create(lv_screen_active());
	page.set_scan_results(results);
	page.on_enter();
	finish_cache_loading(lvgl, page);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Done");

	// A deliberate new scan resets the session. The warmed cache then reports
	// ready during result delivery, so the first finishing frame must already
	// be at the 50% boundary instead of briefly clearing to zero.
	page.set_loading();
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Climbing");
	const uint32_t scan_start_tick =
		ui::WifiSsidPageTestAccess::loading_start_tick(page);
	page.set_scan_results(results);
	REQUIRE(ui::WifiSsidPageTestAccess::cache_content_ready(page));
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Finishing");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	CHECK(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	const uint32_t finish_ready_tick =
		ui::WifiSsidPageTestAccess::loading_ready_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);

	// Refreshing and duplicate ready notification preserve this immediate finish.
	ui::WifiSsidPageTestAccess::refresh_view(page);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Finishing");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	CHECK(ui::WifiSsidPageTestAccess::loading_ready_tick(page) ==
	      finish_ready_tick);
	ui::WifiSsidPageTestAccess::finish_cache_content(page);

	lvgl.advance(250);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) == "Done");
	CHECK(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID Progress preserves its session for empty and error results")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, false,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::Progress);
	page.create(lv_screen_active());
	page.set_loading();
	page.on_enter();
	lvgl.advance(4000);
	REQUIRE(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
		"Waiting");
	REQUIRE(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	const uint32_t scan_start_tick =
		ui::WifiSsidPageTestAccess::loading_start_tick(page);

	page.set_empty();
	// Empty/error results only requeue cache work (Manual SSID); they must not
	// reset the Progress session. Delivery enters the cache-render half.
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	page.set_error("scan failed");
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_progress(page) == 50U);
	CHECK(ui::WifiSsidPageTestAccess::loading_start_tick(page) ==
	      scan_start_tick);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page alternates preload presentation and raster work")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128);
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = "First network" },
		{ .ssid = "Second network" },
	});
	page.on_enter();

	REQUIRE(ui::WifiSsidPageTestAccess::initial_cache_preload_active(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 1U);
	CHECK(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_has_pending(page));

	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 2U);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 0U) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::loading_phase_name(page) ==
	      "Rendering");
	CHECK(ui::WifiSsidPageTestAccess::loading_arc_end_angle(page) == 180);

	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 1U);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 0U) != nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_content_ready(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::roller_visible(page));

	// A new scanning state cancels either phase before it can publish stale
	// work, then hands presentation back to the regular loading timer.
	page.set_loading();
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 0U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_has_pending(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK(ui::WifiSsidPageTestAccess::loading_timer_running(page));

	page.on_pause();
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 0U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page records the final raster-to-ready presentation gap")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128);
	page.create(lv_screen_active());
	page.set_scan_results({ { .ssid = "Only network" } });
	page.on_enter();

	// Manual SSID and the network each use Present -> Rasterize. Delay only the
	// final raster step to prove its gap is included when the roller is revealed.
	lvgl.advance(10);
	lvgl.advance(10);
	lvgl.advance(10);
	lvgl.advance(200);

	CHECK(ui::WifiSsidPageTestAccess::cache_content_ready(page));
	CHECK(ui::WifiSsidPageTestAccess::max_loading_presentation_gap_ms(page) >=
	      200U);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page cancels every initial preload phase before teardown")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128);
	page.create(lv_screen_active());

	page.set_scan_results({ { .ssid = "Old network" } });
	page.on_enter();
	REQUIRE(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	// Replace results while the next cache tick is a Rasterize phase. The old
	// queue must be cancelled before the replacement gets its own Present phase.
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	REQUIRE(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 2U);
	page.set_scan_results({ { .ssid = "Replacement network" } });
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 1U);
	CHECK(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	page.on_pause();
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 0U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	lvgl.advance(100);
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 0U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	page.on_resume();
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	REQUIRE(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 2U);
	page.on_exit();
	CHECK(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 0U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));
	lvgl.advance(100);
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));

	page.on_enter();
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	REQUIRE(ui::WifiSsidPageTestAccess::cache_preload_phase(page) == 2U);
	page.destroy();
	lvgl.advance(100);
}

TEST_CASE("Wi-Fi SSID page retains cached rows for the active roller scroll")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128);
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = "AAAAA" },
		{ .ssid = "BBBBB" },
		{ .ssid = "CCCCC" },
		{ .ssid = "DDDDD" },
		{ .ssid = "EEEEE" },
	});
	page.on_enter();

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	finish_cache_loading(lvgl, page);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_source(page, 0U) !=
		nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));

	lv_obj_set_style_anim_duration(roller, 200, LV_PART_MAIN);
	lv_roller_set_selected(roller, 2U, LV_ANIM_ON);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 2U) == nullptr);

	lvgl.press(roller);
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 2U) == nullptr);
	lvgl.advance(40);
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 2U) == nullptr);

	lv_area_t roller_area {};
	lv_area_t manual_area {};
	lv_obj_get_coords(roller, &roller_area);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(page, 0U,
										   &manual_area));
	REQUIRE(manual_area.y2 >= roller_area.y1);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 0U) !=
	      nullptr);

	page.destroy();
}

TEST_CASE("Wi-Fi SSID cache arena is lazy, retained on pause, and released on exit")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128);
	page.create(lv_screen_active());
	page.set_scan_results({ { .ssid = "Network" } });

	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));

	page.on_enter();
	const ui::SsidTextImageArena *const arena =
		ui::WifiSsidPageTestAccess::text_image_arena(page);
	REQUIRE(arena != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == arena);
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));

	page.on_resume();
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == arena);
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));

	page.on_exit();
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));

	page.on_enter();
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page restores the native roller width after cache fallback")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 1U, true);
	const std::string fallback_ssid =
		"This long SSID must keep its selected marquee after cache fallback";
	page.create(lv_screen_active());
	page.set_scan_results({ { .ssid = "AAAAA" } });
	page.on_enter();

	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	ui::WifiSsidPageTestAccess::render_cache_tick(page);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_activation_failed(page));
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::roller_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_visible(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::loading_timer_running(page));
	page.set_scan_results({ { .ssid = fallback_ssid } });
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(page) == nullptr);
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	lv_obj_t *roller_label = ui::WifiSsidPageTestAccess::roller_label(page);
	lv_obj_t *overlay_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(roller != nullptr);
	REQUIRE(roller_label != nullptr);
	REQUIRE(overlay_label != nullptr);
	CHECK(lv_obj_get_style_width(roller_label, LV_PART_MAIN) ==
	      LV_SIZE_CONTENT);

	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(std::string_view(lv_label_get_text(overlay_label)) == fallback_ssid);
	CHECK(lv_label_get_long_mode(overlay_label) ==
	      LV_LABEL_LONG_SCROLL_CIRCULAR);
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) == LV_OPA_TRANSP);

	page.destroy();
}

TEST_CASE("Wi-Fi SSID image cache budget reaches the page registry scanner")
{
	test_support::LvglTestContext lvgl;
	ui::PageRegistry registry(nullptr, 128);
	registry.create(lv_screen_active());

	ui::WifiSsidPage *page = registry.wifi_ssid_page();
	REQUIRE(page != nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(*page));
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(*page) == nullptr);

	page->set_scan_results({ { .ssid = "Network" } });
	page->on_enter();
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(*page));
	CHECK(ui::WifiSsidPageTestAccess::text_image_arena(*page) != nullptr);

	registry.shutdown();
}

TEST_CASE("Wi-Fi page registry enables scrolling on both SSID surfaces")
{
	test_support::LvglTestContext lvgl;
	ui::PageRegistry registry(nullptr, 0, true);
	registry.create(lv_screen_active());

	ui::WifiPage *main_page = registry.wifi_page();
	ui::WifiSsidPage *scanner_page = registry.wifi_ssid_page();
	REQUIRE(main_page != nullptr);
	REQUIRE(scanner_page != nullptr);
	lv_obj_t *main_container = main_page->get_container();
	REQUIRE(main_container != nullptr);
	lv_obj_t *ssid_canvas = ssid_value_canvas(main_container);
	REQUIRE(ssid_canvas != nullptr);
	CHECK(lv_obj_get_width(ssid_canvas) == ui::SsidInfoRow::VALUE_WIDTH);
	CHECK(lv_obj_get_height(ssid_canvas) == ui::SsidInfoRow::HEIGHT);
	CHECK_FALSE(lv_obj_has_flag(ssid_canvas, LV_OBJ_FLAG_SCROLLABLE));
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_overlay(*scanner_page) !=
	      nullptr);

	registry.shutdown();
}

TEST_CASE("Wi-Fi registry confines scan TinyTTF to raster presentation")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	const lv_font_t *const fallback = ui::compiled_wifi_ssid_font();
	ui::PageRegistry registry(&runtime_font, 0, true,
		ui::SsidFontRenderMode::RuntimeTinyTtf);
	registry.create(lv_screen_active());

	ui::WifiSsidPage *page = registry.wifi_ssid_page();
	REQUIRE(page != nullptr);
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(*page);
	REQUIRE(roller != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::ssid_font(*page) == &runtime_font);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_MAIN) == fallback);
	CHECK(lv_obj_get_style_text_font(roller, LV_PART_SELECTED) == fallback);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_overlay(*page) !=
		nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_font(*page) == fallback);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(*page));

	registry.shutdown();
}

TEST_CASE("Wi-Fi SSID page uses its compiled binary font without a runtime font")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page;
	page.create(lv_screen_active());

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	const lv_font_t *main_font =
		lv_obj_get_style_text_font(roller, LV_PART_MAIN);
	const lv_font_t *selected_font =
		lv_obj_get_style_text_font(roller, LV_PART_SELECTED);
	REQUIRE(main_font != nullptr);
	CHECK(main_font == selected_font);
	CHECK(main_font->fallback == &wifi_zh_font);

	page.destroy();
}

TEST_CASE("Wi-Fi SSID page keeps selected labels static when scrolling is disabled")
{
	test_support::LvglTestContext lvgl;
	const std::string long_ssid =
		"This long SSID must remain a native selected label without marquee";

	ui::WifiSsidPage native_page(&lv_font_montserrat_20);
	native_page.create(lv_screen_active());
	native_page.set_scan_results({ { .ssid = long_ssid } });
	native_page.on_enter();
	lv_obj_t *native_roller =
		ui::WifiSsidPageTestAccess::roller(native_page);
	REQUIRE(native_roller != nullptr);
	lv_roller_set_selected(native_roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(native_roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_overlay(native_page) ==
	      nullptr);
	CHECK_FALSE(
		ui::WifiSsidPageTestAccess::selected_marquee_visible(native_page));
	CHECK(lv_obj_get_child_count(native_roller) == 1U);
	CHECK(lv_obj_get_style_text_opa(native_roller, LV_PART_SELECTED) ==
	      LV_OPA_COVER);
	native_page.destroy();

	ui::WifiSsidPage cache_page(&lv_font_montserrat_20, 128);
	cache_page.create(lv_screen_active());
	cache_page.set_scan_results({ { .ssid = long_ssid } });
	cache_page.on_enter();
	finish_cache_loading(lvgl, cache_page);
	REQUIRE(ui::WifiSsidPageTestAccess::roller_visible(cache_page));
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(cache_page));
	lv_obj_t *cache_roller = ui::WifiSsidPageTestAccess::roller(cache_page);
	REQUIRE(cache_roller != nullptr);
	lv_roller_set_selected(cache_roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(cache_roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_overlay(cache_page) ==
	      nullptr);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(cache_page));
	CHECK(lv_obj_get_child_count(cache_roller) == 1U);
	CHECK(lv_obj_get_style_text_opa(cache_roller, LV_PART_SELECTED) ==
	      LV_OPA_COVER);
	cache_page.destroy();
}

TEST_CASE("Wi-Fi SSID page waits for a same-selection roller rebound before restoring its marquee")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(nullptr, 0, true,
		ui::SsidFontRenderMode::CompiledFallback);
	const std::string long_ssid =
		"A long compiled fallback SSID must stay hidden through a same-row rebound";
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = long_ssid },
		{ .ssid = "Adjacent static network" },
	});
	page.on_enter();

	lv_obj_t *const roller = ui::WifiSsidPageTestAccess::roller(page);
	lv_obj_t *const overlay =
		ui::WifiSsidPageTestAccess::selected_marquee_overlay(page);
	lv_obj_t *const overlay_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(roller != nullptr);
	REQUIRE(overlay != nullptr);
	REQUIRE(overlay_label != nullptr);

	// Establish the long selection without a setup animation, then exercise the
	// normal release alignment with a short drag that resolves to the same row.
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(std::string_view(lv_label_get_text(overlay_label)) == long_ssid);

	constexpr uint32_t settle_duration_ms = 200U;
	lv_obj_set_style_anim_duration(roller, settle_duration_ms, LV_PART_MAIN);
	lv_obj_update_layout(lvgl.screen());
	lv_area_t roller_area {};
	lv_obj_get_coords(roller, &roller_area);
	lvgl.press(roller);
	lvgl.move_while_pressed((roller_area.x1 + roller_area.x2) / 2,
				(roller_area.y1 + roller_area.y2) / 2 - 8);
	lvgl.release();

	CHECK(lv_roller_get_selected(roller) == 1U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_obj_has_flag(overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) == LV_OPA_COVER);
	CHECK(lv_anim_get(ui::WifiSsidPageTestAccess::roller_label(page), nullptr) !=
	      nullptr);
	lvgl.advance(settle_duration_ms - 1U);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));

	lvgl.advance(1U);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK_FALSE(lv_obj_has_flag(overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_label_get_long_mode(overlay_label) ==
	      LV_LABEL_LONG_SCROLL_CIRCULAR);
	CHECK(lv_anim_get(overlay_label, nullptr) != nullptr);
	check_selected_marquee_geometry(page, 1U);

	lv_area_t selected_row_area {};
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
		page, 1U, &selected_row_area));
	lv_obj_get_coords(roller, &roller_area);
	CHECK(selected_row_area.y1 == roller_area.y1 +
	      (lv_obj_get_height(roller) - 32) / 2);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page scrolls only the selected native roller label")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 0, true);
	page.create(lv_screen_active());
	const std::string long_ssid =
		"This is a deliberately long SSID that needs circular marquee display";
	page.set_scan_results({
		{ .ssid = long_ssid },
		{ .ssid = "Other network remains static" },
	});
	page.on_enter();

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	lv_obj_t *overlay =
		ui::WifiSsidPageTestAccess::selected_marquee_overlay(page);
	lv_obj_t *overlay_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(roller != nullptr);
	REQUIRE(overlay != nullptr);
	REQUIRE(overlay_label != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_obj_get_child_count(roller) == 2U);
	CHECK(lv_label_get_long_mode(overlay_label) ==
	      LV_LABEL_LONG_SCROLL_CIRCULAR);
	CHECK(lv_obj_get_width(overlay_label) == lv_obj_get_width(roller));
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_TRANSP);

	lvgl.press(roller);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) == nullptr);
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_COVER);
	lvgl.release();
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_TRANSP);

	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(std::string_view(lv_label_get_text(overlay_label)) == long_ssid);
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_TRANSP);
	lv_anim_t *native_animation = lv_anim_get(overlay_label, nullptr);
	REQUIRE(native_animation != nullptr);
	const int32_t native_offset = native_animation->current_value;
	lvgl.advance(500);
	native_animation = lv_anim_get(overlay_label, nullptr);
	REQUIRE(native_animation != nullptr);
	CHECK(native_animation->current_value != native_offset);

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) == nullptr);
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) == LV_OPA_COVER);
	page.on_resume();
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) != nullptr);
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page keeps its selected marquee on the image-cache path")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(&lv_font_montserrat_20, 128, true);
	page.create(lv_screen_active());
	const std::string long_ssid =
		"This is a deliberately long cached SSID that needs circular marquee display";
	page.set_scan_results({
		{ .ssid = long_ssid },
		{ .ssid = "Other cached network remains static" },
	});
	page.on_enter();

	finish_cache_loading(lvgl, page);
	REQUIRE(ui::WifiSsidPageTestAccess::roller_visible(page));
	REQUIRE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	lv_obj_t *overlay_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(roller != nullptr);
	REQUIRE(overlay_label != nullptr);
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_COVER);

	lvgl.press(roller);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) == nullptr);
	lvgl.release();
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_obj_get_style_text_opa(roller, LV_PART_SELECTED) ==
	      LV_OPA_COVER);

	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(std::string_view(lv_label_get_text(overlay_label)) == long_ssid);
	CHECK(ui::WifiSsidPageTestAccess::cached_text_source(page, 1U) != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_source(page, 0U) != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_source(page, 2U) != nullptr);
	check_selected_marquee_geometry(page, 1U);
	lv_anim_t *cache_animation = lv_anim_get(overlay_label, nullptr);
	REQUIRE(cache_animation != nullptr);
	const int32_t cache_offset = cache_animation->current_value;
	lvgl.advance(500);
	cache_animation = lv_anim_get(overlay_label, nullptr);
	REQUIRE(cache_animation != nullptr);
	CHECK(cache_animation->current_value != cache_offset);

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	page.on_exit();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(lv_anim_get(overlay_label, nullptr) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));
	page.destroy();
}

TEST_CASE("SSID raster marquee request pipeline publishes one fixed window")
{
	test_support::LvglTestContext lvgl;
	lv_obj_t *parent = lv_obj_create(lv_screen_active());
	REQUIRE(parent != nullptr);
	lv_obj_set_size(parent, 230, 32);
	const std::string long_ssid =
		"A deferred SSID raster marquee publishes one fixed visible window";
	{
		ui::SsidRasterMarquee marquee(parent, 230, 32,
			&lv_font_montserrat_20, 256U * 1024U);
		REQUIRE(marquee.request_text(long_ssid) ==
			ui::SsidRasterMarquee::RequestState::Pending);
		REQUIRE(marquee.process_pending() ==
			ui::SsidRasterMarquee::RequestState::Ready);
		CHECK(marquee.is_ready());
		CHECK_FALSE(marquee.is_visible());
		REQUIRE(marquee.show());
		CHECK(marquee.animation_running());
		REQUIRE(marquee.window_image() != nullptr);
		CHECK(lv_obj_get_child_count(marquee.get_obj()) == 1U);
		CHECK(lv_obj_get_style_width(marquee.window_image(), LV_PART_MAIN) == 230);
		CHECK(lv_obj_get_style_height(marquee.window_image(), LV_PART_MAIN) == 32);
		const void *const source = lv_image_get_src(marquee.window_image());
		REQUIRE(source != nullptr);
	}
	lv_obj_delete(parent);
}

TEST_CASE("SSID raster marquee keeps a ready deferred source hidden until published")
{
	test_support::LvglTestContext lvgl;
	lv_obj_t *parent = lv_obj_create(lv_screen_active());
	REQUIRE(parent != nullptr);
	lv_obj_set_size(parent, 230, 32);
	const std::string long_ssid =
		"A deferred SSID raster source must not become visible before its owner"
		" validates and publishes it";
	{
		ui::SsidRasterMarquee marquee(parent, 230, 32,
			&lv_font_montserrat_20, 256U * 1024U);
		REQUIRE(marquee.request_text(long_ssid) ==
			ui::SsidRasterMarquee::RequestState::Pending);
		CHECK_FALSE(marquee.is_visible());
		REQUIRE(marquee.process_pending() ==
			ui::SsidRasterMarquee::RequestState::Ready);
		CHECK(marquee.is_ready());
		CHECK_FALSE(marquee.is_visible());
		REQUIRE(marquee.window_image() != nullptr);
		CHECK(lv_obj_get_child_count(marquee.get_obj()) == 1U);
		CHECK(lv_obj_has_flag(marquee.window_image(), LV_OBJ_FLAG_HIDDEN));
		const uint64_t hidden_revision = marquee.frame_revision();
		lvgl.advance(500);
		CHECK(marquee.frame_revision() == hidden_revision);
		REQUIRE(marquee.show());
		CHECK(marquee.is_visible());
		CHECK(marquee.animation_running());
	}
	lv_obj_delete(parent);
}

TEST_CASE("Wi-Fi SSID page scrolls TinyTTF selections through a fixed window")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiSsidPage page(&runtime_font, 128, true,
		ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lv_screen_active());
	const std::string long_ssid =
		"A selected Runtime TinyTTF SSID scrolls through one fixed raster window";
	page.set_scan_results({
		{ .ssid = long_ssid },
		{ .ssid = "Other cached network remains static" },
	});
	page.on_enter();
	finish_cache_loading(lvgl, page);

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);

	const ui::SsidRasterMarquee *const marquee =
		ui::WifiSsidPageTestAccess::selected_raster_marquee(page);
	REQUIRE(marquee != nullptr);
	CHECK_FALSE(marquee->is_visible());
	CHECK_FALSE(marquee->is_ready());
	CHECK(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK(lv_image_get_src(marquee->window_image()) == nullptr);

	// The selection callback only queues the natural-width raster. The shared
	// cache timer gives that request one bounded snapshot step before fixed rows.
	ui::WifiSsidPageTestAccess::service_cache_timer_tick(page);
	REQUIRE(marquee->is_visible());
	REQUIRE(marquee->animation_running());
	REQUIRE(marquee->window_image() != nullptr);
	CHECK(lv_obj_get_child_count(marquee->get_obj()) == 1U);
	CHECK(lv_obj_get_style_width(marquee->window_image(), LV_PART_MAIN) == 230);
	CHECK(lv_obj_get_style_height(marquee->window_image(), LV_PART_MAIN) == 32);
	const void *const source = lv_image_get_src(marquee->window_image());
	REQUIRE(source != nullptr);
	CHECK_FALSE(lv_obj_has_flag(
		ui::WifiSsidPageTestAccess::selected_marquee_overlay(page),
		LV_OBJ_FLAG_HIDDEN));
	lv_obj_t *const fallback_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(fallback_label != nullptr);
	CHECK(lv_obj_has_flag(fallback_label, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	CHECK_FALSE(visible_tree_uses_font(page.get_container(), &runtime_font));

	lv_area_t selected_area {};
	lv_area_t viewport_area {};
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
		page, 1U, &selected_area));
	lv_obj_get_coords(marquee->get_obj(), &viewport_area);
	CHECK(viewport_area.x1 == selected_area.x1);
	CHECK(viewport_area.y1 == selected_area.y1);
	CHECK(viewport_area.x2 == selected_area.x2);
	CHECK(viewport_area.y2 == selected_area.y2);

	lv_obj_update_layout(lvgl.screen());
	auto coordinate_x = [](lv_obj_t *object) {
		lv_area_t area {};
		lv_obj_get_coords(object, &area);
		return area.x1;
	};
	const int32_t initial_x = coordinate_x(marquee->window_image());
	const uint64_t initial_revision = marquee->frame_revision();
	lvgl.advance(550);
	CHECK(marquee->rendered_phase() == 22U);
	CHECK(marquee->frame_revision() == initial_revision + 1U);
	CHECK(coordinate_x(marquee->window_image()) == initial_x);
	CHECK(lv_image_get_src(marquee->window_image()) == source);

	const uint32_t held_phase = marquee->rendered_phase();
	lvgl.press(roller);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK_FALSE(marquee->animation_running());
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	CHECK(coordinate_x(marquee->window_image()) == initial_x);
	CHECK(marquee->rendered_phase() == held_phase);
	lvgl.release();
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(marquee->animation_running());
	CHECK(lv_image_get_src(marquee->window_image()) == source);

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK_FALSE(marquee->animation_running());
	CHECK(lv_anim_get(fallback_label, nullptr) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	page.on_resume();
	finish_cache_loading(lvgl, page);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	CHECK(marquee->animation_running());
	CHECK(lv_image_get_src(marquee->window_image()) == source);
	page.on_exit();
	CHECK(ui::WifiSsidPageTestAccess::selected_raster_marquee(page) == nullptr);
	CHECK(ui::WifiSsidPageTestAccess::cache_destroyed(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page coalesces TinyTTF marquee requests to the latest selection")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiSsidPage page(&runtime_font, 128, true,
		ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lv_screen_active());
	const std::string long_a =
		"A first long Runtime TinyTTF selection must not publish after a swipe";
	const std::string long_b =
		"A second long Runtime TinyTTF selection is superseded before raster";
	const std::string long_c =
		"A final long Runtime TinyTTF selection owns the only completed marquee";
	page.set_scan_results({
		{ .ssid = long_a },
		{ .ssid = long_b },
		{ .ssid = long_c },
	});
	page.on_enter();
	finish_cache_loading(lvgl, page);

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	auto select = [roller](uint32_t index) {
		lv_roller_set_selected(roller, index, LV_ANIM_OFF);
		lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	};

	select(1U);
	const ui::SsidRasterMarquee *const marquee =
		ui::WifiSsidPageTestAccess::selected_raster_marquee(page);
	REQUIRE(marquee != nullptr);
	CHECK(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK_FALSE(marquee->is_visible());
	CHECK(lv_image_get_src(marquee->window_image()) == nullptr);

	select(2U);
	CHECK(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_pending_text(page) ==
	      long_b);
	select(3U);
	CHECK(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_pending_text(page) ==
	      long_c);
	CHECK_FALSE(marquee->is_visible());
	CHECK(lv_image_get_src(marquee->window_image()) == nullptr);

	// A pending fixed-row miss may not share the selected handoff's timer tick.
	ui::WifiSsidPageTestAccess::queue_fixed_raster(page,
		"A separate fixed row can only rasterize after the selected marquee");
	REQUIRE(ui::WifiSsidPageTestAccess::cache_has_pending(page));
	ui::WifiSsidPageTestAccess::service_cache_timer_tick(page);
	REQUIRE(marquee->is_ready());
	REQUIRE(marquee->is_visible());
	CHECK(marquee->animation_running());
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK(ui::WifiSsidPageTestAccess::selected_marquee_pending_text(page).empty());
	CHECK(ui::WifiSsidPageTestAccess::cache_has_pending(page));

	ui::WifiSsidPageTestAccess::service_cache_timer_tick(page);
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_has_pending(page));
	page.destroy();
}

TEST_CASE("Wi-Fi SSID page cancels a pending TinyTTF marquee before teardown")
{
	test_support::LvglTestContext lvgl;
	lv_font_t runtime_font = lv_font_montserrat_20;
	ui::WifiSsidPage page(&runtime_font, 128, true,
		ui::SsidFontRenderMode::RuntimeTinyTtf);
	page.create(lv_screen_active());
	page.set_scan_results({ {
		.ssid = "A pending Runtime TinyTTF marquee must not outlive page teardown",
	} });
	page.on_enter();
	finish_cache_loading(lvgl, page);

	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);
	const ui::SsidRasterMarquee *const marquee =
		ui::WifiSsidPageTestAccess::selected_raster_marquee(page);
	REQUIRE(marquee != nullptr);
	REQUIRE(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));

	page.on_pause();
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_raster_work_pending(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::cache_timer_running(page));
	CHECK_FALSE(ui::WifiSsidPageTestAccess::selected_marquee_visible(page));
	page.destroy();
	lvgl.advance(20);
}

TEST_CASE("Wi-Fi SSID page aligns its native fallback marquee with the selected row")
{
	test_support::LvglTestContext lvgl;
	ui::WifiSsidPage page(nullptr, 0U, true,
		ui::SsidFontRenderMode::CompiledFallback,
		ui::WifiLoadingMode::None);
	page.create(lv_screen_active());
	page.set_scan_results({
		{ .ssid = "A long SSID rendered by the native fallback marquee" },
		{ .ssid = "Adjacent native fallback SSID" },
	});
	page.on_enter();
	lv_obj_t *roller = ui::WifiSsidPageTestAccess::roller(page);
	REQUIRE(roller != nullptr);
	const lv_font_t *const fallback_font =
		lv_obj_get_style_text_font(roller, LV_PART_MAIN);
	REQUIRE(fallback_font != nullptr);
	const int32_t fallback_line_height =
		lv_font_get_line_height(fallback_font);
	const int32_t expected_line_space = fallback_line_height < 32 ?
		32 - fallback_line_height : 0;
	CHECK_FALSE(ui::WifiSsidPageTestAccess::text_image_cache_enabled(page));
	CHECK(lv_obj_get_height(roller) == 112);
	CHECK(lv_obj_get_style_text_line_space(roller, LV_PART_MAIN) ==
	      expected_line_space);
	CHECK(lv_obj_get_style_text_line_space(roller, LV_PART_SELECTED) ==
	      expected_line_space);
	lv_obj_set_style_anim_duration(roller, 0, LV_PART_MAIN);
	lv_roller_set_selected(roller, 1U, LV_ANIM_OFF);
	lv_obj_send_event(roller, LV_EVENT_VALUE_CHANGED, nullptr);

	lv_obj_t *const overlay =
		ui::WifiSsidPageTestAccess::selected_marquee_overlay(page);
	lv_obj_t *const overlay_label =
		ui::WifiSsidPageTestAccess::selected_marquee_label(page);
	REQUIRE(overlay != nullptr);
	REQUIRE(overlay_label != nullptr);
	const lv_font_t *const marquee_font =
		lv_obj_get_style_text_font(overlay_label, LV_PART_MAIN);
	REQUIRE(marquee_font != nullptr);
	const int32_t marquee_font_height = lv_font_get_line_height(marquee_font);
	CHECK(lv_obj_get_height(overlay_label) == marquee_font_height);
	CHECK(lv_obj_get_y(overlay_label) ==
	      (32 - marquee_font_height) / 2);
	lv_area_t overlay_area {};
	lv_area_t overlay_label_area {};
	lv_obj_get_coords(overlay, &overlay_area);
	lv_obj_get_coords(overlay_label, &overlay_label_area);
	const int32_t top_margin = overlay_label_area.y1 - overlay_area.y1;
	const int32_t bottom_margin = overlay_area.y2 - overlay_label_area.y2;
	const int32_t margin_delta = top_margin > bottom_margin ?
		top_margin - bottom_margin : bottom_margin - top_margin;
	CHECK(margin_delta <= 1);

	lv_area_t roller_area {};
	lv_area_t selected_area {};
	lv_obj_get_coords(roller, &roller_area);
	REQUIRE(ui::WifiSsidPageTestAccess::cached_text_row_area(
		page, 1U, &selected_area));
	CHECK(selected_area.y1 == roller_area.y1 + (112 - 32) / 2);
	CHECK(selected_area.y2 == selected_area.y1 + 32 - 1);
	check_selected_marquee_geometry(page, 1U);
	page.destroy();
}
