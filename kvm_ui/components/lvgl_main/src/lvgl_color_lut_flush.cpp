/**
 * @file lvgl_color_lut_flush.cpp
 * ST7789P3 RGB565 flush wrapper and identity color LUTs.
 */

#include <algorithm>
#include <chrono>
#include <cstdint>

#include <spdlog/spdlog.h>
#include <src/display/lv_display_private.h>

#include "lvgl_ui_internal.hpp"

namespace
{

// --- RGB565 identity LUT and framebuffer flush wrapper ---
static uint8_t g_red_lut[32];
static uint8_t g_green_lut[64];
static uint8_t g_blue_lut[32];
static lv_display_flush_cb_t s_original_flush = nullptr;

static void build_identity_color_luts()
{
	for (int i = 0; i < 32; i++) {
		g_red_lut[i] = static_cast<uint8_t>(i);
		g_blue_lut[i] = static_cast<uint8_t>(i);
	}

	for (int i = 0; i < 64; i++) {
		g_green_lut[i] = static_cast<uint8_t>(i);
	}
}

static void lut_flush_cb(lv_display_t *disp, const lv_area_t *area,
			 uint8_t *px_map)
{
	const bool probe_enabled = lvgl_main::wifi_latency_probe::enabled();
	const auto wrapper_started = probe_enabled ? std::chrono::steady_clock::now() :
						    std::chrono::steady_clock::time_point {};
	uint64_t pixels_flushed = 0U;
	if (px_map != nullptr && area != nullptr) {
		const int32_t w = lv_area_get_width(area);
		const int32_t h = lv_area_get_height(area);
		if (w > 0 && h > 0) {
			const uint32_t total = static_cast<uint32_t>(w) *
					       static_cast<uint32_t>(h);
			pixels_flushed = total;
			auto *pixels = reinterpret_cast<uint16_t *>(px_map);
			for (uint32_t i = 0; i < total; i++) {
				const uint16_t px = pixels[i];
				const uint16_t r = g_red_lut[(px >> 11) & 0x1F];
				const uint16_t g =
					g_green_lut[(px >> 5) & 0x3F];
				const uint16_t b = g_blue_lut[px & 0x1F];
				pixels[i] = static_cast<uint16_t>((r << 11) |
								  (g << 5) | b);
			}

			if (touch_click_marker_state.enabled.load(
				    std::memory_order_acquire) &&
			    touch_click_marker_state.visible.load(
				    std::memory_order_acquire)) {
				const int32_t display_offset_x =
					disp == nullptr ? 0 : disp->offset_x;
				const int32_t display_offset_y =
					disp == nullptr ? 0 : disp->offset_y;
				const int32_t marker_x =
					touch_click_marker_state.x.load(
						std::memory_order_acquire) +
					display_offset_x;
				const int32_t marker_y =
					touch_click_marker_state.y.load(
						std::memory_order_acquire) +
					display_offset_y;
				const int32_t y_min = std::max(
					area->y1,
					marker_y - kTouchClickMarkerRadius);
				const int32_t y_max = std::min(
					area->y2,
					marker_y + kTouchClickMarkerRadius);
				const int32_t x_min = std::max(
					area->x1,
					marker_x - kTouchClickMarkerRadius);
				const int32_t x_max = std::min(
					area->x2,
					marker_x + kTouchClickMarkerRadius);
				for (int32_t y = y_min; y <= y_max; ++y) {
					for (int32_t x = x_min; x <= x_max;
					     ++x) {
						const int32_t dx = x - marker_x;
						const int32_t dy = y - marker_y;
						if (dx * dx + dy * dy >
						    kTouchClickMarkerRadius *
							    kTouchClickMarkerRadius) {
							continue;
						}
						const uint32_t row =
							static_cast<uint32_t>(
								y - area->y1);
						const uint32_t col =
							static_cast<uint32_t>(
								x - area->x1);
						const uint32_t index =
							row * static_cast<
								      uint32_t>(
								      w) +
							col;
						if (index < total) {
							pixels[index] = static_cast<
								uint16_t>(
								~pixels[index]);
						}
					}
				}
			}
		}
	}
	const auto original_started = probe_enabled ?
		std::chrono::steady_clock::now() : std::chrono::steady_clock::time_point {};
	if (s_original_flush != nullptr) {
		s_original_flush(disp, area, px_map);
	}
	if (!probe_enabled) {
		return;
	}
	const auto completed = std::chrono::steady_clock::now();
	const uint64_t original_duration_us = static_cast<uint64_t>(
		std::chrono::duration_cast<std::chrono::microseconds>(
			completed - original_started)
			.count());
	const uint64_t wrapper_duration_us = static_cast<uint64_t>(
		std::chrono::duration_cast<std::chrono::microseconds>(
			completed - wrapper_started)
			.count());
	lvgl_main::wifi_latency_probe::record_flush(wrapper_duration_us,
						     original_duration_us, pixels_flushed);
	lvgl_main::wifi_latency_probe::record_visible_flush(disp, area);
}

} // namespace

namespace lvgl_main
{

void install_color_lut_flush(lv_display_t *disp)
{
	build_identity_color_luts();
	s_original_flush = disp->flush_cb;
	disp->flush_cb = lut_flush_cb;
	spdlog::info(
		"[lvgl] RGB565 identity LUT flush wrapper enabled");
}

void restore_color_lut_flush(lv_display_t *disp)
{
	if (disp != nullptr && s_original_flush != nullptr &&
	    disp->flush_cb == lut_flush_cb) {
		disp->flush_cb = s_original_flush;
	}
}

void reset_color_lut_flush()
{
	s_original_flush = nullptr;
}

} // namespace lvgl_main
