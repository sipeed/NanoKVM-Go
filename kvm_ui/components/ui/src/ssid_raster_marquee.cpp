#include "ui/ssid_raster_marquee.hpp"

#include "ui/ssid_marquee_window.hpp"
#include "ui/wifi_latency_probe.hpp"

#include <algorithm>
#include <limits>
#include <new>
#include <utility>
#include <vector>

#include <spdlog/spdlog.h>

#include "ui/ssid_text_image_cache.hpp"

namespace ui
{

namespace
{

constexpr uint32_t kFramePeriodMs = 100;
constexpr uint32_t kSpeedPixelsPerSecond = 40;
constexpr std::size_t kArgb8888BytesPerPixel = sizeof(uint32_t);

void configure_viewport(lv_obj_t *object)
{
	if (object == nullptr) {
		return;
	}
	lv_obj_set_style_bg_opa(object, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(object, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(object, 0, LV_PART_MAIN);
	lv_obj_set_style_radius(object, 0, LV_PART_MAIN);
	lv_obj_set_style_clip_corner(object, true, LV_PART_MAIN);
	lv_obj_clear_flag(object, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(object, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(object, LV_OBJ_FLAG_IGNORE_LAYOUT);
}

void configure_image(lv_obj_t *object)
{
	if (object == nullptr) {
		return;
	}
	lv_obj_clear_flag(object, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(object, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(object, LV_OBJ_FLAG_IGNORE_LAYOUT);
	lv_obj_set_pos(object, 0, 0);
}

[[nodiscard]] bool draw_buf_bytes(const lv_draw_buf_t *draw_buf,
				  uint32_t expected_height,
				  std::size_t &bytes) noexcept
{
	if (draw_buf == nullptr || draw_buf->data == nullptr || draw_buf->header.w == 0 ||
	    draw_buf->header.h != expected_height ||
	    draw_buf->header.cf != LV_COLOR_FORMAT_ARGB8888 ||
	    draw_buf->header.stride == 0) {
		return false;
	}
	const std::size_t width = draw_buf->header.w;
	if (width > std::numeric_limits<std::size_t>::max() / kArgb8888BytesPerPixel) {
		return false;
	}
	const std::size_t minimum_stride = width * kArgb8888BytesPerPixel;
	const std::size_t stride = draw_buf->header.stride;
	if (stride < minimum_stride ||
	    expected_height > std::numeric_limits<std::size_t>::max() / stride) {
		return false;
	}
	bytes = stride * expected_height;
	return bytes <= draw_buf->data_size;
}

} // namespace

SsidRasterMarquee::SsidRasterMarquee(lv_obj_t *parent,
					     uint16_t viewport_width,
					     uint16_t height,
					     const lv_font_t *font,
					     std::size_t max_bytes,
					     lv_text_align_t text_align)
	: _parent(parent)
	, _font(font)
	, _viewport_width(viewport_width)
	, _height(height)
	, _max_bytes(max_bytes)
	, _text_align(text_align)
{
	if (_parent == nullptr || !lv_obj_is_valid(_parent) || _viewport_width == 0 ||
	    _height == 0) {
		return;
	}

	_viewport = lv_obj_create(_parent);
	if (_viewport == nullptr) {
		return;
	}
	lv_obj_set_size(_viewport, _viewport_width, _height);
	configure_viewport(_viewport);
	lv_obj_add_flag(_viewport, LV_OBJ_FLAG_HIDDEN);

	_window_image = lv_image_create(_viewport);
	if (_window_image == nullptr) {
		destroy();
		return;
	}
	configure_image(_window_image);
	lv_obj_set_size(_window_image, _viewport_width, _height);
	lv_obj_add_flag(_window_image, LV_OBJ_FLAG_HIDDEN);
}

SsidRasterMarquee::~SsidRasterMarquee()
{
	destroy();
}

int32_t SsidRasterMarquee::measure_text(std::string_view text,
						const lv_font_t *font) noexcept
{
	if (font == nullptr || text.empty() || text.size() >
		static_cast<std::size_t>(std::numeric_limits<uint32_t>::max())) {
		return 0;
	}
	return std::max<int32_t>(0, lv_text_get_width(
		text.data(), static_cast<uint32_t>(text.size()), font, 0));
}

bool SsidRasterMarquee::stage_window(
	const lv_draw_buf_t *source, std::size_t byte_budget,
	std::vector<uint8_t> &storage,
	std::unique_ptr<lv_draw_buf_t> &draw_buf) const noexcept
{
	std::size_t source_bytes = 0;
	if (_viewport_width == 0 || _height == 0 || byte_budget == 0 ||
	    !draw_buf_bytes(source, _height, source_bytes) ||
	    source->header.w <= _viewport_width) {
		return false;
	}

	const std::size_t stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(_viewport_width, LV_COLOR_FORMAT_ARGB8888));
	if (stride == 0 ||
	    _height > std::numeric_limits<std::size_t>::max() / stride) {
		return false;
	}
	const std::size_t window_bytes = stride * _height;
	if (window_bytes > std::numeric_limits<uint32_t>::max() ||
	    source_bytes > byte_budget || window_bytes > byte_budget - source_bytes) {
		return false;
	}

	try {
		storage.assign(window_bytes, 0U);
		draw_buf = std::make_unique<lv_draw_buf_t>();
	} catch (const std::bad_alloc &) {
		storage.clear();
		draw_buf.reset();
		return false;
	}
	if (lv_draw_buf_init(draw_buf.get(), _viewport_width, _height,
			     LV_COLOR_FORMAT_ARGB8888, LV_STRIDE_AUTO,
			     storage.data(), static_cast<uint32_t>(storage.size())) !=
	    LV_RESULT_OK) {
		storage.clear();
		draw_buf.reset();
		return false;
	}

	const SsidArgb8888View source_view {
		.data = source->data,
		.data_size = source->data_size,
		.width = source->header.w,
		.height = source->header.h,
		.stride_bytes = source->header.stride,
	};
	const SsidMutableArgb8888View destination_view {
		.data = storage.data(),
		.data_size = storage.size(),
		.width = draw_buf->header.w,
		.height = draw_buf->header.h,
		.stride_bytes = draw_buf->header.stride,
	};
	if (!compose_ssid_marquee_window(
		source_view, destination_view, 0, 0,
		_text_align == LV_TEXT_ALIGN_RIGHT
			? SsidMarqueeWindowAlignment::Right
			: SsidMarqueeWindowAlignment::Left)) {
		storage.clear();
		draw_buf.reset();
		return false;
	}
	return true;
}

bool SsidRasterMarquee::attach_window(
	const lv_draw_buf_t *source, std::vector<uint8_t> storage,
	std::unique_ptr<lv_draw_buf_t> draw_buf, int32_t text_width,
	uint32_t gap_width) noexcept
{
	std::size_t source_bytes = 0;
	if (_window_image == nullptr || !lv_obj_is_valid(_window_image) ||
	    !draw_buf_bytes(source, _height, source_bytes) ||
	    source->header.w <= _viewport_width || text_width <= _viewport_width ||
	    draw_buf == nullptr || storage.empty() || draw_buf->data != storage.data() ||
	    gap_width == 0) {
		return false;
	}
	const uint64_t track_width =
		static_cast<uint64_t>(source->header.w) + gap_width;
	if (track_width == 0 ||
	    track_width > std::numeric_limits<uint32_t>::max()) {
		return false;
	}

	stop_animation();
	lv_image_set_src(_window_image, nullptr);
	_window_draw_buf.reset();
	_window_storage.clear();
	_source_draw_buf = source;
	_window_storage = std::move(storage);
	_window_draw_buf = std::move(draw_buf);
	lv_image_set_src(_window_image, _window_draw_buf.get());
	lv_obj_set_size(_window_image, _viewport_width, _height);
	if (lv_image_get_src(_window_image) != _window_draw_buf.get()) {
		lv_image_set_src(_window_image, nullptr);
		_window_draw_buf.reset();
		_window_storage.clear();
		_source_draw_buf = nullptr;
		return false;
	}

	_text_width = text_width;
	_source_width = source->header.w;
	_gap_width = gap_width;
	_track_width = static_cast<uint32_t>(track_width);
	_phase_anchor_tick = lv_tick_get();
	_phase_anchor = 0;
	_rendered_phase = 0;
	_frame_revision = 0;
	lv_obj_add_flag(_window_image, LV_OBJ_FLAG_HIDDEN);
	lv_obj_add_flag(_viewport, LV_OBJ_FLAG_HIDDEN);
	return true;
}

bool SsidRasterMarquee::install_prepared_raster(
	const SsidPreparedRaster &raster, bool start_visible) noexcept
{
	if (_viewport == nullptr || _window_image == nullptr ||
	    !lv_obj_is_valid(_viewport) || !lv_obj_is_valid(_window_image) ||
	    !raster.has_valid_storage() || !raster.overflows_viewport ||
	    raster.identity.viewport_width != _viewport_width ||
	    raster.source_height != _height ||
	    raster.source_width <= _viewport_width ||
	    raster.text_width <= static_cast<int32_t>(_viewport_width) ||
	    raster.space_advance <= 0) {
		return false;
	}

	const std::size_t stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(raster.source_width, LV_COLOR_FORMAT_ARGB8888));
	if (stride == 0 ||
	    raster.source_height > std::numeric_limits<std::size_t>::max() / stride) {
		return false;
	}
	const std::size_t bytes = stride * raster.source_height;
	if (bytes > std::numeric_limits<uint32_t>::max()) {
		return false;
	}

	const int64_t gap_width = static_cast<int64_t>(raster.space_advance) *
				  LV_LABEL_WAIT_CHAR_COUNT;
	if (gap_width <= 0 ||
	    gap_width > std::numeric_limits<uint32_t>::max()) {
		return false;
	}

	std::vector<uint8_t> source_storage;
	std::unique_ptr<lv_draw_buf_t> source_draw_buf;
	std::vector<uint8_t> window_storage;
	std::unique_ptr<lv_draw_buf_t> window_draw_buf;
	try {
		source_storage.assign(bytes, 0U);
		source_draw_buf = std::make_unique<lv_draw_buf_t>();
	} catch (const std::bad_alloc &) {
		return false;
	}
	if (lv_draw_buf_init(source_draw_buf.get(), raster.source_width,
			     raster.source_height, LV_COLOR_FORMAT_ARGB8888,
			     LV_STRIDE_AUTO, source_storage.data(),
			     static_cast<uint32_t>(source_storage.size())) != LV_RESULT_OK) {
		return false;
	}
	for (uint16_t y = 0; y < raster.source_height; ++y) {
		uint8_t *const destination =
			source_storage.data() +
			static_cast<std::size_t>(y) * source_draw_buf->header.stride;
		for (uint16_t x = 0; x < raster.source_width; ++x) {
			const uint32_t pixel = (*raster.pixels)[
				static_cast<std::size_t>(y) * raster.source_width + x];
			destination[static_cast<std::size_t>(x) * 4U] =
				static_cast<uint8_t>(pixel);
			destination[static_cast<std::size_t>(x) * 4U + 1U] =
				static_cast<uint8_t>(pixel >> 8U);
			destination[static_cast<std::size_t>(x) * 4U + 2U] =
				static_cast<uint8_t>(pixel >> 16U);
			destination[static_cast<std::size_t>(x) * 4U + 3U] =
				static_cast<uint8_t>(pixel >> 24U);
		}
	}

	if (!stage_window(source_draw_buf.get(), raster.identity.byte_budget,
			  window_storage, window_draw_buf)) {
		return false;
	}

	clear_source();
	_prepared_storage = std::move(source_storage);
	_prepared_draw_buf = std::move(source_draw_buf);
	if (!attach_window(_prepared_draw_buf.get(), std::move(window_storage),
			   std::move(window_draw_buf), raster.text_width,
			   static_cast<uint32_t>(gap_width))) {
		clear_source();
		return false;
	}
	_failed_text.clear();
	if (start_visible) {
		return show();
	}
	return true;
}

SsidRasterMarquee::RequestState SsidRasterMarquee::request_text(
	std::string_view text)
{
	const ScopedWifiLatencyProbe latency_probe(
		WifiLatencyProbeStage::MarqueeRequest);
	if (_viewport == nullptr || _font == nullptr || _parent == nullptr ||
	    !lv_obj_is_valid(_viewport)) {
		return RequestState::Rejected;
	}
	const int32_t width = measure_text(text, _font);
	if (width <= static_cast<int32_t>(_viewport_width) || width <= 0 ||
	    _max_bytes == 0) {
		_failed_text.clear();
		cancel_pending();
		hide();
		return RequestState::Rejected;
	}
	if (_text == text && is_active() && !is_pending()) {
		return RequestState::Ready;
	}
	if (_pending_text == text && is_pending()) {
		return RequestState::Pending;
	}
	if (_failed_text == text && !is_active() && !is_pending()) {
		hide();
		return RequestState::Failed;
	}

	if (!prepare_source(text, width)) {
		try {
			_failed_text.assign(text.data(), text.size());
		} catch (const std::bad_alloc &) {
			_failed_text.clear();
		}
		spdlog::warn("[wifi] SSID raster marquee setup failed: "
			     "width={}, height={}, budget_bytes={}",
			     width, _height, _max_bytes);
		hide();
		return RequestState::Failed;
	}
	try {
		_pending_text.assign(text.data(), text.size());
	} catch (const std::bad_alloc &) {
		clear_source();
		return RequestState::Failed;
	}
	_pending_width = width;
	return RequestState::Pending;
}

SsidRasterMarquee::RequestState SsidRasterMarquee::process_pending()
{
	const ScopedWifiLatencyProbe latency_probe(
		WifiLatencyProbeStage::MarqueeProcess);
	if (!is_pending()) {
		return is_active() ? RequestState::Ready : RequestState::Idle;
	}
	if (_cache == nullptr || _pending_text.empty() ||
	    _pending_width <= static_cast<int32_t>(_viewport_width)) {
		cancel_pending();
		return RequestState::Failed;
	}

	const auto result = _cache->render_next();
	if (result == SsidTextImageCache::RenderResult::Failed) {
		try {
			_failed_text = _pending_text;
		} catch (const std::bad_alloc &) {
			_failed_text.clear();
		}
		clear_source();
		return RequestState::Failed;
	}
	if (result == SsidTextImageCache::RenderResult::Idle &&
	    _cache->source_for(_pending_text) == nullptr) {
		try {
			_failed_text = _pending_text;
		} catch (const std::bad_alloc &) {
			_failed_text.clear();
		}
		clear_source();
		return RequestState::Failed;
	}
	if (_cache->source_for(_pending_text) == nullptr) {
		return RequestState::Pending;
	}
	if (!publish_source(_pending_text, _pending_width)) {
		try {
			_failed_text = _pending_text;
		} catch (const std::bad_alloc &) {
			_failed_text.clear();
		}
		clear_source();
		return RequestState::Failed;
	}
	_pending_text.clear();
	_pending_width = 0;
	_failed_text.clear();
	return RequestState::Ready;
}

void SsidRasterMarquee::cancel_pending() noexcept
{
	if (_cache != nullptr) {
		_cache->cancel_pending();
	}
	_pending_text.clear();
	_pending_width = 0;
	if (_text.empty()) {
		clear_source();
	}
}

bool SsidRasterMarquee::is_pending() const noexcept
{
	return _cache != nullptr && !_pending_text.empty() &&
	       _cache->has_foreground_pending();
}

bool SsidRasterMarquee::prepare_source(std::string_view text, int32_t width)
{
	if (width <= 0 || width > std::numeric_limits<uint16_t>::max() ||
	    _height == 0) {
		return false;
	}
	const std::size_t source_stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(static_cast<uint16_t>(width),
				   LV_COLOR_FORMAT_ARGB8888));
	const std::size_t window_stride = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(_viewport_width, LV_COLOR_FORMAT_ARGB8888));
	if (source_stride == 0 || window_stride == 0 ||
	    _height > std::numeric_limits<std::size_t>::max() / source_stride ||
	    _height > std::numeric_limits<std::size_t>::max() / window_stride) {
		return false;
	}
	const std::size_t source_bytes = source_stride * _height;
	const std::size_t window_bytes = window_stride * _height;
	if (source_bytes > _max_bytes || window_bytes > _max_bytes - source_bytes) {
		return false;
	}

	// A new request cannot share the old natural-width arena: its slot size is
	// fixed for the cache lifetime. Detach visible users before replacing it.
	clear_source();
	try {
		_arena = std::make_unique<SsidTextImageArena>(source_bytes);
		_cache = std::make_unique<SsidTextImageCache>(
			_parent, _font, static_cast<uint16_t>(width), _height,
			source_bytes, _arena.get(), _text_align);
	} catch (const std::bad_alloc &) {
		clear_source();
		return false;
	}
	if (_cache == nullptr || !_cache->is_usable()) {
		clear_source();
		return false;
	}
	std::vector<std::string> foreground;
	try {
		foreground.emplace_back(text);
	} catch (const std::bad_alloc &) {
		clear_source();
		return false;
	}
	_cache->request_labels(foreground, {});
	return _cache->has_foreground_pending();
}

bool SsidRasterMarquee::publish_source(std::string_view text, int32_t width)
{
	const ScopedWifiLatencyProbe latency_probe(
		WifiLatencyProbeStage::MarqueePublish);
	if (_cache == nullptr || _window_image == nullptr ||
	    !lv_obj_is_valid(_window_image)) {
		return false;
	}
	const lv_draw_buf_t *const source = _cache->source_for(std::string(text));
	if (source == nullptr || source->header.w != width) {
		return false;
	}
	const int32_t space_width = std::max<int32_t>(1,
		lv_font_get_glyph_width(_font, ' ', ' '));
	const int64_t gap_width =
		static_cast<int64_t>(space_width) * LV_LABEL_WAIT_CHAR_COUNT;
	if (gap_width <= 0 ||
	    gap_width > std::numeric_limits<uint32_t>::max()) {
		return false;
	}
	const uint64_t track_width =
		static_cast<uint64_t>(width) + static_cast<uint64_t>(gap_width);
	if (track_width > std::numeric_limits<uint32_t>::max()) {
		return false;
	}

	std::string published_text;
	std::vector<uint8_t> window_storage;
	std::unique_ptr<lv_draw_buf_t> window_draw_buf;
	try {
		published_text.assign(text.data(), text.size());
	} catch (const std::bad_alloc &) {
		return false;
	}
	if (!stage_window(source, _max_bytes, window_storage, window_draw_buf) ||
	    !attach_window(source, std::move(window_storage),
			   std::move(window_draw_buf), width,
			   static_cast<uint32_t>(gap_width))) {
		return false;
	}
	_text = std::move(published_text);
	return true;
}

bool SsidRasterMarquee::compose_phase(uint32_t phase, bool invalidate) noexcept
{
	std::size_t source_bytes = 0;
	std::size_t window_bytes = 0;
	if (!is_active() || !draw_buf_bytes(_source_draw_buf, _height, source_bytes) ||
	    !draw_buf_bytes(_window_draw_buf.get(), _height, window_bytes) ||
	    _track_width == 0) {
		return false;
	}
	const SsidArgb8888View source {
		.data = _source_draw_buf->data,
		.data_size = _source_draw_buf->data_size,
		.width = _source_draw_buf->header.w,
		.height = _source_draw_buf->header.h,
		.stride_bytes = _source_draw_buf->header.stride,
	};
	const SsidMutableArgb8888View destination {
		.data = _window_draw_buf->data,
		.data_size = _window_draw_buf->data_size,
		.width = _window_draw_buf->header.w,
		.height = _window_draw_buf->header.h,
		.stride_bytes = _window_draw_buf->header.stride,
	};
	phase %= _track_width;
	if (!compose_ssid_marquee_window(
		    source, destination, phase, _gap_width,
		    _text_align == LV_TEXT_ALIGN_RIGHT
			    ? SsidMarqueeWindowAlignment::Right
			    : SsidMarqueeWindowAlignment::Left)) {
		return false;
	}
	_rendered_phase = phase;
	if (invalidate && is_visible()) {
		lv_obj_invalidate(_window_image);
		++_frame_revision;
	}
	return true;
}

void SsidRasterMarquee::clear_source() noexcept
{
	stop_animation();
	if (_window_image != nullptr && lv_obj_is_valid(_window_image)) {
		lv_image_set_src(_window_image, nullptr);
		lv_obj_add_flag(_window_image, LV_OBJ_FLAG_HIDDEN);
	}
	if (_viewport != nullptr && lv_obj_is_valid(_viewport)) {
		lv_obj_add_flag(_viewport, LV_OBJ_FLAG_HIDDEN);
	}
	_source_draw_buf = nullptr;
	_window_draw_buf.reset();
	_window_storage.clear();
	_cache.reset();
	_arena.reset();
	_prepared_draw_buf.reset();
	_prepared_storage.clear();
	_text.clear();
	_pending_text.clear();
	_pending_width = 0;
	_text_width = 0;
	_source_width = 0;
	_gap_width = 0;
	_track_width = 0;
	_phase_anchor_tick = 0;
	_phase_anchor = 0;
	_rendered_phase = 0;
	_frame_revision = 0;
	_animation_requested = false;
}

bool SsidRasterMarquee::start_animation() noexcept
{
	if (!is_active() || !is_visible()) {
		return false;
	}
	stop_animation();
	_animation_requested = true;
	_phase_anchor_tick = lv_tick_get();
	_phase_anchor = _rendered_phase;
	_timer = lv_timer_create(timer_cb, kFramePeriodMs, this);
	if (_timer == nullptr) {
		_animation_requested = false;
	}
	return _timer != nullptr;
}

void SsidRasterMarquee::stop_animation() noexcept
{
	_animation_requested = false;
	if (_timer == nullptr) {
		return;
	}
	lv_timer_t *const timer = _timer;
	_timer = nullptr;
	lv_timer_delete(timer);
}

void SsidRasterMarquee::timer_cb(lv_timer_t *timer)
{
	const ScopedWifiLatencyProbe latency_probe(WifiLatencyProbeStage::MarqueeTick);
	auto *const self = static_cast<SsidRasterMarquee *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr || self->_timer != timer || !self->is_visible() ||
	    !self->is_active() || self->_track_width == 0) {
		return;
	}
	const uint64_t elapsed = lv_tick_elaps(self->_phase_anchor_tick);
	const uint64_t traveled = elapsed * kSpeedPixelsPerSecond / 1000U;
	const uint32_t phase = static_cast<uint32_t>(
		(static_cast<uint64_t>(self->_phase_anchor) + traveled) %
		self->_track_width);
	if (phase == self->_rendered_phase) {
		return;
	}
	if (!self->compose_phase(phase, true)) {
		self->stop_animation();
	}
}

void SsidRasterMarquee::hide() noexcept
{
	stop_animation();
	if (_viewport != nullptr && lv_obj_is_valid(_viewport)) {
		lv_obj_add_flag(_viewport, LV_OBJ_FLAG_HIDDEN);
	}
}

bool SsidRasterMarquee::show() noexcept
{
	if (!is_active() || _viewport == nullptr || !lv_obj_is_valid(_viewport)) {
		return false;
	}
	lv_obj_clear_flag(_window_image, LV_OBJ_FLAG_HIDDEN);
	lv_obj_clear_flag(_viewport, LV_OBJ_FLAG_HIDDEN);
	if (!animation_running() && !start_animation()) {
		hide();
		return false;
	}
	return true;
}

void SsidRasterMarquee::pause() noexcept
{
	stop_animation();
}

void SsidRasterMarquee::resume() noexcept
{
	if (is_active() && !is_visible()) {
		(void)show();
	} else if (is_visible() && !_animation_requested) {
		(void)start_animation();
	}
}

void SsidRasterMarquee::destroy() noexcept
{
	if (_viewport == nullptr && _cache == nullptr && _arena == nullptr &&
	    _prepared_draw_buf == nullptr && _window_draw_buf == nullptr) {
		return;
	}
	clear_source();
	if (_viewport != nullptr && lv_obj_is_valid(_viewport)) {
		lv_obj_delete(_viewport);
	}
	_viewport = nullptr;
	_window_image = nullptr;
	_parent = nullptr;
	_failed_text.clear();
}

bool SsidRasterMarquee::is_active() const noexcept
{
	return (_cache != nullptr || _prepared_draw_buf != nullptr) &&
	       _source_draw_buf != nullptr && _window_draw_buf != nullptr &&
	       !_window_storage.empty() && _text_width > 0 && _source_width > 0 &&
	       _track_width > 0 && _window_image != nullptr &&
	       lv_obj_is_valid(_window_image) &&
	       lv_image_get_src(_window_image) == _window_draw_buf.get();
}

bool SsidRasterMarquee::is_visible() const noexcept
{
	return _viewport != nullptr && lv_obj_is_valid(_viewport) &&
	       !lv_obj_has_flag(_viewport, LV_OBJ_FLAG_HIDDEN);
}

bool SsidRasterMarquee::animation_running() const noexcept
{
	return _animation_requested;
}

} // namespace ui
