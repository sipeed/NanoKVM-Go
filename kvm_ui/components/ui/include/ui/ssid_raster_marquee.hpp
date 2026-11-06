#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "ui/ssid_prepared_raster.hpp"

namespace ui
{

class SsidTextImageArena;
class SsidTextImageCache;

// A fixed-window, image-backed marquee for SSID text rendered with a runtime
// font. The natural-width raster is produced once per effective text; timer
// ticks copy only the visible viewport and never move LVGL objects.
class SsidRasterMarquee final {
    public:
	// A request is deliberately split from its rasterization so callers can
	// enqueue a long SSID without blocking the LVGL input callback.
	enum class RequestState : uint8_t {
		Rejected,
		Pending,
		Ready,
		Failed,
		Idle,
	};

	SsidRasterMarquee(lv_obj_t *parent, uint16_t viewport_width,
			 uint16_t height, const lv_font_t *font,
			 std::size_t max_bytes,
			 lv_text_align_t text_align = LV_TEXT_ALIGN_LEFT);
	~SsidRasterMarquee();

	SsidRasterMarquee(const SsidRasterMarquee &) = delete;
	SsidRasterMarquee &operator=(const SsidRasterMarquee &) = delete;

	// Installs pixels that have already been rasterized by the non-LVGL worker.
	// No font lookup, text measurement, or snapshotting happens on this path.
	bool install_prepared_raster(const SsidPreparedRaster &raster,
				     bool start_visible) noexcept;

	// Queue one natural-width snapshot without executing the expensive LVGL
	// snapshot operation. `process_pending` performs at most one raster step.
	RequestState request_text(std::string_view text);
	RequestState process_pending();
	void cancel_pending() noexcept;
	bool is_pending() const noexcept;
	bool is_ready() const noexcept
	{
		return is_active();
	}

	void hide() noexcept;
	bool show() noexcept;
	void pause() noexcept;
	void resume() noexcept;
	void destroy() noexcept;

	bool is_active() const noexcept;
	bool is_visible() const noexcept;
	bool animation_running() const noexcept;
	int32_t text_width() const noexcept
	{
		return _text_width;
	}

	lv_obj_t *get_obj() const noexcept
	{
		return _viewport;
	}
	lv_obj_t *window_image() const noexcept
	{
		return _window_image;
	}
	uint32_t rendered_phase() const noexcept
	{
		return _rendered_phase;
	}
	uint64_t frame_revision() const noexcept
	{
		return _frame_revision;
	}

	static int32_t measure_text(std::string_view text,
				    const lv_font_t *font) noexcept;

    private:
	lv_obj_t *_parent { nullptr };
	lv_obj_t *_viewport { nullptr };
	lv_obj_t *_window_image { nullptr };
	lv_timer_t *_timer { nullptr };
	const lv_font_t *_font { nullptr };
	uint16_t _viewport_width { 0 };
	uint16_t _height { 0 };
	std::size_t _max_bytes { 0 };
	lv_text_align_t _text_align { LV_TEXT_ALIGN_LEFT };
	int32_t _text_width { 0 };
	uint32_t _source_width { 0 };
	uint32_t _gap_width { 0 };
	uint32_t _track_width { 0 };
	uint32_t _phase_anchor_tick { 0 };
	uint32_t _phase_anchor { 0 };
	uint32_t _rendered_phase { 0 };
	uint64_t _frame_revision { 0 };
	bool _animation_requested { false };
	std::string _text;
	std::string _failed_text;
	std::string _pending_text;
	int32_t _pending_width { 0 };
	std::unique_ptr<SsidTextImageArena> _arena;
	std::unique_ptr<SsidTextImageCache> _cache;
	std::unique_ptr<lv_draw_buf_t> _prepared_draw_buf;
	std::vector<uint8_t> _prepared_storage;
	const lv_draw_buf_t *_source_draw_buf { nullptr };
	std::unique_ptr<lv_draw_buf_t> _window_draw_buf;
	std::vector<uint8_t> _window_storage;

	bool prepare_source(std::string_view text, int32_t width);
	bool publish_source(std::string_view text, int32_t width);
	bool stage_window(const lv_draw_buf_t *source, std::size_t byte_budget,
			  std::vector<uint8_t> &storage,
			  std::unique_ptr<lv_draw_buf_t> &draw_buf) const noexcept;
	bool attach_window(const lv_draw_buf_t *source,
			   std::vector<uint8_t> storage,
			   std::unique_ptr<lv_draw_buf_t> draw_buf,
			   int32_t text_width, uint32_t gap_width) noexcept;
	bool compose_phase(uint32_t phase, bool invalidate) noexcept;
	void clear_source() noexcept;
	bool start_animation() noexcept;
	void stop_animation() noexcept;
	static void timer_cb(lv_timer_t *timer);
};

} // namespace ui
