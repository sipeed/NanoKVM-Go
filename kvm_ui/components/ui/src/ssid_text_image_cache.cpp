#include "ui/ssid_text_image_cache.hpp"
#include "ui/wifi_latency_probe.hpp"

#include <chrono>
#include <new>
#include <utility>

#include <spdlog/spdlog.h>

namespace ui
{

SsidTextImageArena::SsidTextImageArena(std::size_t max_bytes)
	: _storage(max_bytes == 0
				   ? nullptr
				   : std::make_unique<uint8_t[]>(max_bytes))
	, _storage_bytes(max_bytes)
{
}

bool SsidTextImageArena::configure(std::size_t slot_bytes) noexcept
{
	if (slot_bytes == 0) {
		return false;
	}
	if (_slot_bytes != 0) {
		return _slot_bytes == slot_bytes;
	}

	const std::size_t slot_count = _storage_bytes / slot_bytes;
	if (slot_count == 0) {
		return false;
	}

	try {
		_available_slots.reserve(slot_count);
		for (std::size_t index = 0; index < slot_count; ++index) {
			_available_slots.push_back(slot_count - index - 1U);
		}
	} catch (const std::bad_alloc &) {
		_available_slots.clear();
		return false;
	}

	_slot_bytes = slot_bytes;
	_capacity = slot_count;
	return true;
}

std::optional<SsidTextImageArena::Slot> SsidTextImageArena::acquire() noexcept
{
	if (_available_slots.empty() || _slot_bytes == 0) {
		return std::nullopt;
	}

	const std::size_t index = _available_slots.back();
	_available_slots.pop_back();
	return Slot { .data = _storage.get() + index * _slot_bytes,
		      .index = index };
}

void SsidTextImageArena::release(std::size_t index) noexcept
{
	if (index >= _capacity || _available_slots.size() >= _capacity) {
		return;
	}
	_available_slots.push_back(index);
}

std::size_t SsidTextImageArena::slot_bytes() const noexcept
{
	return _slot_bytes;
}

std::size_t SsidTextImageArena::capacity() const noexcept
{
	return _capacity;
}

SsidTextImageCache::SsidTextImageCache(lv_obj_t *raster_parent,
					 const lv_font_t *font, uint16_t width,
					 uint16_t height, std::size_t max_bytes,
					 SsidTextImageArena *arena,
					 lv_text_align_t text_align)
	: _raster_parent(raster_parent)
	, _font(font)
	, _width(width)
	, _height(height)
	, _text_align(text_align)
	, _max_bytes(max_bytes)
	, _arena(arena)
{
	const auto configure_started = std::chrono::steady_clock::now();
	const std::size_t row_bytes =
		static_cast<std::size_t>(LV_DRAW_BUF_STRIDE(
			_width, LV_COLOR_FORMAT_ARGB8888)) * _height;
	if (_arena == nullptr || !_arena->configure(row_bytes) ||
	    !create_raster_context()) {
		_arena = nullptr;
	}
	const auto configure_duration [[maybe_unused]] = std::chrono::duration_cast<
		std::chrono::microseconds>(std::chrono::steady_clock::now() -
						      configure_started);
	SPDLOG_DEBUG("[wifi] SSID image cache configured: arena_bytes={}, "
		      "slot_bytes={}, capacity={}, setup_us={}",
		      max_bytes, row_bytes, _arena == nullptr ? 0U : _arena->capacity(),
		      configure_duration.count());
}

SsidTextImageCache::~SsidTextImageCache()
{
	// Cached draw buffers return their arena slots before the scratch snapshot
	// and reusable LVGL objects are released. The page owns the parent until
	// this destructor has completed.
	cancel_pending();
	clear();
	_scratch_snapshot.reset();
	destroy_raster_context();
}

void SsidTextImageCache::request_labels(
	const std::vector<std::string> &foreground_labels,
	const std::vector<std::string> &speculative_labels)
{
	cancel_pending();
	_pinned_labels.clear();
	std::unordered_set<std::string> requested_labels;

	for (const std::string &label : foreground_labels) {
		if (label.empty()) {
			continue;
		}

		const auto [_, inserted] = requested_labels.insert(label);
		if (!inserted) {
			continue;
		}
		_pinned_labels.insert(label);

		const auto entry = _entries.find(label);
		if (entry != _entries.end()) {
			touch(entry->second);
			SPDLOG_DEBUG("[wifi] SSID image cache hit");
			continue;
		}

		_foreground_pending_labels.push_back(label);
	}

	for (const std::string &label : speculative_labels) {
		if (label.empty()) {
			continue;
		}

		const auto [_, inserted] = requested_labels.insert(label);
		if (!inserted) {
			continue;
		}
		const auto entry = _entries.find(label);
		if (entry != _entries.end()) {
			// Full-list background requests must not make every old entry recent;
			// only foreground visibility updates LRU recency.
			continue;
		}

		_speculative_pending_labels.push_back(label);
	}
}

void SsidTextImageCache::cancel_pending()
{
	_foreground_pending_labels.clear();
	_speculative_pending_labels.clear();
}

SsidTextImageCache::RenderResult SsidTextImageCache::render_next()
{
	std::vector<std::string> *pending_labels = &_foreground_pending_labels;
	if (pending_labels->empty()) {
		pending_labels = &_speculative_pending_labels;
	}
	if (pending_labels->empty()) {
		return RenderResult::Idle;
	}

	std::string label = std::move(pending_labels->front());
	pending_labels->erase(pending_labels->begin());
	if (const auto existing = _entries.find(label);
	    existing != _entries.end()) {
		touch(existing->second);
		return RenderResult::Idle;
	}

	const auto render_started = std::chrono::steady_clock::now();
	lv_draw_buf_t *snapshot = take_snapshot(label);
	const auto render_duration = std::chrono::duration_cast<
			std::chrono::microseconds>(std::chrono::steady_clock::now() -
							      render_started);
	if (snapshot == nullptr || snapshot->data_size == 0 || _arena == nullptr) {
		spdlog::warn("SSID image cache snapshot failed after {} us",
				     render_duration.count());
		return RenderResult::Failed;
	}

	const std::size_t bytes = _arena->slot_bytes();
	if (!reserve_space(bytes)) {
		spdlog::warn("SSID image cache could not reserve {} bytes", bytes);
		return RenderResult::Failed;
	}
	CachedBuffer draw_buf = copy_to_arena(*snapshot);
	if (draw_buf.draw_buf == nullptr) {
		spdlog::warn("SSID image cache could not acquire a persistent row slot");
		return RenderResult::Failed;
	}

	Entry entry {
		.draw_buf = std::move(draw_buf),
		.bytes = bytes,
	};
	touch(entry);
	_used_bytes += bytes;
	_entries.emplace(std::move(label), std::move(entry));
	SPDLOG_DEBUG("[wifi] SSID image cache rendered: bytes={}, used={}, "
			      "budget={}, duration_us={}, pending={}",
			      bytes, _used_bytes, _max_bytes, render_duration.count(),
			      pending_count());
	return RenderResult::Rendered;
}

const lv_draw_buf_t *SsidTextImageCache::source_for(const std::string &label)
{
	const auto entry = _entries.find(label);
	if (entry == _entries.end()) {
		return nullptr;
	}

	touch(entry->second);
	return entry->second.draw_buf.draw_buf.get();
}

bool SsidTextImageCache::is_usable() const noexcept
{
	return _arena != nullptr && _arena->capacity() > 0 &&
	       _raster_host != nullptr && _raster_label != nullptr &&
	       lv_obj_is_valid(_raster_host) && lv_obj_is_valid(_raster_label);
}

bool SsidTextImageCache::has_foreground_pending() const noexcept
{
	return !_foreground_pending_labels.empty();
}

bool SsidTextImageCache::has_pending() const noexcept
{
	return has_foreground_pending() || !_speculative_pending_labels.empty();
}

std::size_t SsidTextImageCache::pending_count() const noexcept
{
	return _foreground_pending_labels.size() +
	       _speculative_pending_labels.size();
}

void SsidTextImageCache::clear()
{
	cancel_pending();
	_pinned_labels.clear();
	_entries.clear();
	_used_bytes = 0;
}

void SsidTextImageCache::SnapshotBufferDeleter::operator()(
		lv_draw_buf_t *draw_buf) const noexcept
{
	if (draw_buf != nullptr) {
		lv_draw_buf_destroy(draw_buf);
	}
}

SsidTextImageCache::CachedBuffer::~CachedBuffer()
{
	if (draw_buf != nullptr && _arena != nullptr) {
		_arena->release(_slot_index);
	}
}

bool SsidTextImageCache::create_raster_context() noexcept
{
	if (_raster_parent == nullptr || _font == nullptr || _width == 0 ||
	    _height == 0 || !lv_obj_is_valid(_raster_parent)) {
		return false;
	}

	const int32_t font_height = lv_font_get_line_height(_font);
	if (font_height <= 0 || font_height > _height) {
		return false;
	}

	_raster_host = lv_obj_create(_raster_parent);
	if (_raster_host == nullptr) {
		return false;
	}

	lv_obj_add_flag(_raster_host, LV_OBJ_FLAG_IGNORE_LAYOUT);
	// Keep the reusable tree out of normal page rendering. It is briefly made
	// drawable only while LVGL snapshots it on this same thread.
	lv_obj_add_flag(_raster_host, LV_OBJ_FLAG_HIDDEN);
	lv_obj_set_pos(_raster_host, 0, -static_cast<int32_t>(_height));
	lv_obj_set_size(_raster_host, _width, _height);
	lv_obj_set_style_bg_opa(_raster_host, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_raster_host, 0, 0);
	lv_obj_set_style_shadow_width(_raster_host, 0, 0);
	lv_obj_set_style_outline_width(_raster_host, 0, 0);
	lv_obj_set_style_transform_width(_raster_host, 0, 0);
	lv_obj_set_style_transform_height(_raster_host, 0, 0);
	lv_obj_set_style_pad_all(_raster_host, 0, 0);
	lv_obj_clear_flag(_raster_host, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_raster_host, LV_OBJ_FLAG_SCROLLABLE);

	_raster_label = lv_label_create(_raster_host);
	if (_raster_label == nullptr) {
		destroy_raster_context();
		return false;
	}

	lv_obj_set_pos(_raster_label, 0,
		       (static_cast<int32_t>(_height) - font_height) / 2);
	lv_obj_set_size(_raster_label, _width, font_height);
	lv_obj_set_style_bg_opa(_raster_label, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_raster_label, 0, 0);
	// Avoid theme-driven expansion. The label itself still reserves a small
	// glyph overdraw area, which is cropped below after its snapshot is taken.
	lv_obj_set_style_shadow_width(_raster_label, 0, 0);
	lv_obj_set_style_outline_width(_raster_label, 0, 0);
	lv_obj_set_style_transform_width(_raster_label, 0, 0);
	lv_obj_set_style_transform_height(_raster_label, 0, 0);
	lv_obj_set_style_pad_all(_raster_label, 0, 0);
	lv_obj_set_style_text_color(_raster_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_raster_label, _font, 0);
	lv_obj_set_style_text_align(_raster_label, _text_align, 0);
	lv_obj_clear_flag(_raster_label, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_raster_label, LV_OBJ_FLAG_SCROLLABLE);
	lv_label_set_long_mode(_raster_label, LV_LABEL_LONG_CLIP);
	return true;
}

void SsidTextImageCache::destroy_raster_context() noexcept
{
	_raster_label = nullptr;
	if (_raster_host != nullptr && lv_obj_is_valid(_raster_host)) {
		lv_obj_delete(_raster_host);
	}
	_raster_host = nullptr;
}

lv_draw_buf_t *SsidTextImageCache::take_snapshot(std::string_view label)
{
	const ScopedWifiLatencyProbe latency_probe(WifiLatencyProbeStage::Snapshot);
	if (_raster_host == nullptr || _raster_label == nullptr ||
	    !lv_obj_is_valid(_raster_host) || !lv_obj_is_valid(_raster_label)) {
		return nullptr;
	}

	const std::string owned_label(label);
	lv_label_set_text(_raster_label, owned_label.c_str());
	lv_obj_update_layout(_raster_host);

	if (_scratch_snapshot == nullptr) {
		_scratch_snapshot.reset(lv_snapshot_create_draw_buf(
			_raster_host, LV_COLOR_FORMAT_ARGB8888));
	}
	if (_scratch_snapshot == nullptr) {
		return nullptr;
	}

	lv_obj_clear_flag(_raster_host, LV_OBJ_FLAG_HIDDEN);
	if (lv_snapshot_take_to_draw_buf(_raster_host,
					 LV_COLOR_FORMAT_ARGB8888,
					 _scratch_snapshot.get()) == LV_RESULT_OK) {
		lv_obj_add_flag(_raster_host, LV_OBJ_FLAG_HIDDEN);
		return _scratch_snapshot.get();
	}

	// Text/font changes can increase LVGL's extended draw size. Replace the
	// scratch buffer once with a correctly shaped buffer and retry; the old
	// buffer is released only after the replacement exists.
	SnapshotBufferPtr replacement(lv_snapshot_create_draw_buf(
		_raster_host, LV_COLOR_FORMAT_ARGB8888));
	if (replacement == nullptr ||
	    lv_snapshot_take_to_draw_buf(_raster_host,
					 LV_COLOR_FORMAT_ARGB8888,
					 replacement.get()) != LV_RESULT_OK) {
		lv_obj_add_flag(_raster_host, LV_OBJ_FLAG_HIDDEN);
		return nullptr;
	}
	lv_obj_add_flag(_raster_host, LV_OBJ_FLAG_HIDDEN);
	_scratch_snapshot = std::move(replacement);
	return _scratch_snapshot.get();
}

SsidTextImageCache::CachedBuffer
SsidTextImageCache::copy_to_arena(const lv_draw_buf_t &snapshot)
{
	CachedBuffer result;
	const std::size_t source_stride = snapshot.header.stride;
	const std::size_t source_height = snapshot.header.h;
	if (_arena == nullptr || snapshot.header.w < _width ||
	    snapshot.header.h < _height || snapshot.data == nullptr ||
	    snapshot.header.cf != LV_COLOR_FORMAT_ARGB8888 || source_stride == 0 ||
	    source_height == 0 ||
	    source_stride > snapshot.data_size / source_height) {
		return result;
	}

	const int32_t width_extension =
		static_cast<int32_t>(snapshot.header.w) - static_cast<int32_t>(_width);
	const int32_t height_extension =
		static_cast<int32_t>(snapshot.header.h) - static_cast<int32_t>(_height);
	if (width_extension < 0 || height_extension < 0 ||
	    width_extension % 2 != 0 || height_extension % 2 != 0 ||
	    width_extension != height_extension ||
	    source_stride / 4U < snapshot.header.w) {
		return result;
	}

	const auto slot = _arena->acquire();
	if (!slot) {
		return result;
	}

	try {
		result.draw_buf = std::make_unique<lv_draw_buf_t>();
	} catch (const std::bad_alloc &) {
		_arena->release(slot->index);
		return {};
	}
	if (lv_draw_buf_init(result.draw_buf.get(), _width, _height,
			     LV_COLOR_FORMAT_ARGB8888, LV_STRIDE_AUTO, slot->data,
			     static_cast<uint32_t>(_arena->slot_bytes())) != LV_RESULT_OK) {
		_arena->release(slot->index);
		result.draw_buf.reset();
		return {};
	}
	// `lv_draw_buf_copy` covers image pixels, not stride padding. Clear this
	// single acquired row before copying so source_for() never publishes any
	// uninitialized arena byte.
	lv_draw_buf_clear(result.draw_buf.get(), nullptr);

	const int32_t ext_draw_size = width_extension / 2;
	lv_area_t source_area {};
	source_area.x1 = ext_draw_size;
	source_area.y1 = ext_draw_size;
	source_area.x2 = ext_draw_size + static_cast<int32_t>(_width) - 1;
	source_area.y2 = ext_draw_size + static_cast<int32_t>(_height) - 1;
	lv_draw_buf_copy(result.draw_buf.get(), nullptr, &snapshot, &source_area);
	result._arena = _arena;
	result._slot_index = slot->index;
	return result;
}

bool SsidTextImageCache::reserve_space(std::size_t bytes)
{
	if (bytes > _max_bytes) {
		return false;
	}

	while (_used_bytes > _max_bytes - bytes) {
		auto candidate = _entries.end();
		for (auto entry = _entries.begin(); entry != _entries.end(); ++entry) {
			if (_pinned_labels.contains(entry->first)) {
				continue;
			}
			if (candidate == _entries.end() ||
			    entry->second.last_used < candidate->second.last_used) {
				candidate = entry;
			}
		}

		if (candidate == _entries.end()) {
			return false;
		}

		_used_bytes -= candidate->second.bytes;
		_entries.erase(candidate);
		SPDLOG_DEBUG("[wifi] SSID image cache evicted one entry");
	}

	return true;
}

void SsidTextImageCache::touch(Entry &entry) noexcept
{
	++_use_sequence;
	entry.last_used = _use_sequence;
}

} // namespace ui
