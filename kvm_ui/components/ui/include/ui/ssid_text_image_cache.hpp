#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include <lvgl.h>

namespace ui
{

// LVGL-thread arena for fixed-size SSID image rows. WifiSsidPage creates it
// only for an active non-Loading cache presentation and outlives its cache
// entries until that page activation ends.
class SsidTextImageArena final {
    public:
	struct Slot {
		uint8_t *data { nullptr };
		std::size_t index { 0 };
	};

	explicit SsidTextImageArena(std::size_t max_bytes);

	SsidTextImageArena(const SsidTextImageArena &) = delete;
	SsidTextImageArena &operator=(const SsidTextImageArena &) = delete;

	bool configure(std::size_t slot_bytes) noexcept;
	std::optional<Slot> acquire() noexcept;
	void release(std::size_t index) noexcept;
	std::size_t slot_bytes() const noexcept;
	std::size_t capacity() const noexcept;

    private:
	// `_storage` owns every persistent image byte. A slot is published only
	// after its complete draw-buffer extent has been copied.
	std::unique_ptr<uint8_t[]> _storage;
	std::size_t _storage_bytes { 0 };
	std::vector<std::size_t> _available_slots;
	std::size_t _slot_bytes { 0 };
	std::size_t _capacity { 0 };
};

// LVGL-thread owner for pre-rasterized SSID display labels. It deliberately
// knows only the already-sanitized label text, fixed render geometry, and font.
class SsidTextImageCache final {
    public:
	enum class RenderResult : uint8_t {
		Idle,
		Rendered,
		Failed,
	};

	SsidTextImageCache(lv_obj_t *raster_parent, const lv_font_t *font,
			   uint16_t width, uint16_t height,
			   std::size_t max_bytes, SsidTextImageArena *arena,
			   lv_text_align_t text_align = LV_TEXT_ALIGN_CENTER);
	~SsidTextImageCache();

	SsidTextImageCache(const SsidTextImageCache &) = delete;
	SsidTextImageCache &operator=(const SsidTextImageCache &) = delete;

	// Foreground labels render before background labels and remain pinned against
	// LRU eviction until the next request. The caller may queue the whole list.
	void request_labels(const std::vector<std::string> &foreground_labels,
			    const std::vector<std::string> &speculative_labels);
	void cancel_pending();
	RenderResult render_next();
	const lv_draw_buf_t *source_for(const std::string &label);
	bool is_usable() const noexcept;
	bool has_foreground_pending() const noexcept;
	bool has_pending() const noexcept;
	std::size_t pending_count() const noexcept;
	void clear();

    private:
	struct SnapshotBufferDeleter {
		void operator()(lv_draw_buf_t *draw_buf) const noexcept;
	};

	using SnapshotBufferPtr =
		std::unique_ptr<lv_draw_buf_t, SnapshotBufferDeleter>;

	struct CachedBuffer {
		std::unique_ptr<lv_draw_buf_t> draw_buf;
		SsidTextImageArena *_arena { nullptr };
		std::size_t _slot_index { 0 };

		CachedBuffer() = default;
		CachedBuffer(CachedBuffer &&) noexcept = default;
		CachedBuffer &operator=(CachedBuffer &&) = delete;
		~CachedBuffer();

		CachedBuffer(const CachedBuffer &) = delete;
		CachedBuffer &operator=(const CachedBuffer &) = delete;
	};

	struct Entry {
		CachedBuffer draw_buf;
		std::size_t bytes { 0 };
		uint64_t last_used { 0 };
	};

	lv_obj_t *_raster_parent { nullptr };
	lv_obj_t *_raster_host { nullptr };
	lv_obj_t *_raster_label { nullptr };
	const lv_font_t *_font { nullptr };
	uint16_t _width { 0 };
	uint16_t _height { 0 };
	lv_text_align_t _text_align { LV_TEXT_ALIGN_CENTER };
	std::size_t _max_bytes { 0 };
	SsidTextImageArena *_arena { nullptr };
	std::size_t _used_bytes { 0 };
	uint64_t _use_sequence { 0 };
	std::unordered_map<std::string, Entry> _entries;
	std::unordered_set<std::string> _pinned_labels;
	std::vector<std::string> _foreground_pending_labels;
	std::vector<std::string> _speculative_pending_labels;
	SnapshotBufferPtr _scratch_snapshot;

	bool create_raster_context() noexcept;
	void destroy_raster_context() noexcept;
	lv_draw_buf_t *take_snapshot(std::string_view label);
	CachedBuffer copy_to_arena(const lv_draw_buf_t &snapshot);
	bool reserve_space(std::size_t bytes);
	void touch(Entry &entry) noexcept;

	friend class SsidTextImageCacheTestAccess;
	friend class WifiSsidPageTestAccess;
	friend class WifiPageTestAccess;
};

} // namespace ui
