#include <cstddef>
#include <string>
#include <string_view>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include <lvgl.h>

#include "ui/scona.hpp"
#include "ui/ssid_info_row.hpp"
#include "ui/ssid_text_image_cache.hpp"
#include "ui/wifi_latency_probe.hpp"
#include "ui/wifi_ssid_font.hpp"

namespace ui
{

class SsidTextImageCacheTestAccess final {
    public:
	static std::size_t entry_count(const SsidTextImageCache &cache)
	{
		return cache._entries.size();
	}

	static bool contains(const SsidTextImageCache &cache,
			     const std::string &label)
	{
		return cache._entries.contains(label);
	}

	static std::size_t foreground_pending_count(const SsidTextImageCache &cache)
	{
		return cache._foreground_pending_labels.size();
	}

	static std::size_t speculative_pending_count(const SsidTextImageCache &cache)
	{
		return cache._speculative_pending_labels.size();
	}

	static bool is_pinned(const SsidTextImageCache &cache,
				      const std::string &label)
	{
		return cache._pinned_labels.contains(label);
	}

	static std::size_t used_bytes(const SsidTextImageCache &cache)
	{
		return cache._used_bytes;
	}

	static void set_max_bytes(SsidTextImageCache &cache,
				  std::size_t max_bytes)
	{
		cache._max_bytes = max_bytes;
	}

	static lv_obj_t *raster_host(const SsidTextImageCache &cache)
	{
		return cache._raster_host;
	}

	static lv_obj_t *raster_label(const SsidTextImageCache &cache)
	{
		return cache._raster_label;
	}

	static const lv_draw_buf_t *scratch_snapshot(
		const SsidTextImageCache &cache)
	{
		return cache._scratch_snapshot.get();
	}
};

class SsidInfoRowTestAccess final {
    public:
	static lv_obj_t *key_label(const SsidInfoRow &row)
	{
		return row._key_label;
	}

	static lv_obj_t *value_canvas(const SsidInfoRow &row)
	{
		return row._value_canvas;
	}

	static lv_obj_t *fallback_label(const SsidInfoRow &row)
	{
		return row._fallback_label;
	}

	static const std::vector<uint8_t> &canvas_storage(const SsidInfoRow &row)
	{
		return row._canvas_storage;
	}
};

} // namespace ui

namespace
{

class LvglContext final {
    public:
	LvglContext()
	{
		lv_init();
		_display = lv_display_create(270, 240);
	}

	~LvglContext()
	{
		if (_display != nullptr) {
			lv_display_delete(_display);
		}
		lv_deinit();
	}

	LvglContext(const LvglContext &) = delete;
	LvglContext &operator=(const LvglContext &) = delete;

    private:
	lv_display_t *_display { nullptr };
};

constexpr uint16_t RASTER_SIZE = 32;
constexpr std::size_t CACHE_TEST_BUDGET = 64U * 1024U;

} // namespace

TEST_CASE("SSID image arena fixes slot geometry for its full lifetime")
{
	const std::size_t slot_bytes = static_cast<std::size_t>(
		LV_DRAW_BUF_STRIDE(ui::SConA::WIDTH, LV_COLOR_FORMAT_ARGB8888)) *
		RASTER_SIZE;
	ui::SsidTextImageArena arena(slot_bytes * 2U);

	REQUIRE(arena.configure(slot_bytes));
	CHECK(arena.slot_bytes() == slot_bytes);
	CHECK(arena.capacity() == 2U);
	CHECK_FALSE(arena.configure(slot_bytes + 4U));

	const auto first = arena.acquire();
	const auto second = arena.acquire();
	REQUIRE(first.has_value());
	REQUIRE(second.has_value());
	CHECK_FALSE(arena.acquire().has_value());
	CHECK(first->data != second->data);

	arena.release(first->index);
	arena.release(second->index);
	CHECK(arena.acquire().has_value());
}

TEST_CASE("SSID image cache reuses an already rasterized display label")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
					      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);
	cache.request_labels({ "same" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	const lv_draw_buf_t *first_source = cache.source_for("same");
	REQUIRE(first_source != nullptr);
	CHECK(first_source->header.w == RASTER_SIZE);
	CHECK(first_source->header.h == RASTER_SIZE);

	cache.request_labels({ "same" }, {});
	CHECK_FALSE(cache.has_pending());
	CHECK(ui::SsidTextImageCacheTestAccess::entry_count(cache) == 1U);
	CHECK(cache.source_for("same") == first_source);
}

TEST_CASE("SSID image cache retains one hidden raster context and scratch snapshot")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
				      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);

	lv_obj_t *const raster_host =
		ui::SsidTextImageCacheTestAccess::raster_host(cache);
	lv_obj_t *const raster_label =
		ui::SsidTextImageCacheTestAccess::raster_label(cache);
	REQUIRE(raster_host != nullptr);
	REQUIRE(raster_label != nullptr);
	CHECK(lv_obj_has_flag(raster_host, LV_OBJ_FLAG_HIDDEN));

	cache.request_labels({ "first" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	const lv_draw_buf_t *const first_scratch =
		ui::SsidTextImageCacheTestAccess::scratch_snapshot(cache);
	REQUIRE(first_scratch != nullptr);

	cache.request_labels({ "second" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	CHECK(ui::SsidTextImageCacheTestAccess::raster_host(cache) == raster_host);
	CHECK(ui::SsidTextImageCacheTestAccess::raster_label(cache) == raster_label);
	CHECK(ui::SsidTextImageCacheTestAccess::scratch_snapshot(cache) ==
	      first_scratch);
	CHECK(lv_obj_has_flag(raster_host, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_style_text_align(raster_label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
}

TEST_CASE("SSID image cache supports right-aligned hidden raster text")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
				      ui::SsidInfoRow::VALUE_WIDTH,
				      ui::SsidInfoRow::HEIGHT, CACHE_TEST_BUDGET, &arena,
				      LV_TEXT_ALIGN_RIGHT);

	lv_obj_t *const raster_label =
		ui::SsidTextImageCacheTestAccess::raster_label(cache);
	REQUIRE(raster_label != nullptr);
	CHECK(lv_obj_get_style_text_align(raster_label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_RIGHT);
}

TEST_CASE("SSID information row uses a compiled fallback and copied cache image")
{
	LvglContext lvgl;
	ui::SsidInfoRow row(lv_screen_active());
	row.create();
	lv_obj_update_layout(lv_screen_active());

	lv_obj_t *const root = row.get_obj();
	lv_obj_t *const key_label = ui::SsidInfoRowTestAccess::key_label(row);
	lv_obj_t *const value_canvas =
		ui::SsidInfoRowTestAccess::value_canvas(row);
	lv_obj_t *const fallback_label =
		ui::SsidInfoRowTestAccess::fallback_label(row);
	REQUIRE(root != nullptr);
	REQUIRE(key_label != nullptr);
	REQUIRE(value_canvas != nullptr);
	REQUIRE(fallback_label != nullptr);
	CHECK(lv_obj_get_width(root) == ui::SsidInfoRow::WIDTH);
	CHECK(lv_obj_get_height(root) == ui::SsidInfoRow::HEIGHT);
	CHECK_FALSE(lv_obj_has_flag(root, LV_OBJ_FLAG_CLICKABLE));
	CHECK_FALSE(lv_obj_has_flag(root, LV_OBJ_FLAG_SCROLLABLE));
	CHECK(std::string_view(lv_label_get_text(key_label)) == "SSID:");
	CHECK(lv_obj_get_x(key_label) == ui::SsidInfoRow::KEY_HORIZONTAL_INSET);
	CHECK(lv_obj_get_x(value_canvas) == ui::SsidInfoRow::VALUE_X);
	CHECK(lv_obj_get_width(value_canvas) == ui::SsidInfoRow::VALUE_WIDTH);
	CHECK(lv_obj_get_height(value_canvas) == ui::SsidInfoRow::HEIGHT);
	CHECK(lv_obj_has_flag(value_canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_style_text_font(fallback_label, LV_PART_MAIN) ==
	      ui::compiled_wifi_ssid_font());
	CHECK(lv_obj_get_style_text_align(fallback_label, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_RIGHT);
	CHECK(lv_label_get_long_mode(fallback_label) == LV_LABEL_LONG_CLIP);
	CHECK_FALSE(lv_obj_has_flag(fallback_label, LV_OBJ_FLAG_HIDDEN));

	REQUIRE(row.set_fallback_text("\xE4\xB8\xAD\xE6\x96\x87 SSID"));
	CHECK(std::string_view(lv_label_get_text(fallback_label)) ==
	      "\xE4\xB8\xAD\xE6\x96\x87 SSID");

	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
				      ui::SsidInfoRow::VALUE_WIDTH,
				      ui::SsidInfoRow::HEIGHT, CACHE_TEST_BUDGET, &arena,
				      LV_TEXT_ALIGN_RIGHT);
	cache.request_labels({ "cached SSID" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	const lv_draw_buf_t *const source = cache.source_for("cached SSID");
	REQUIRE(source != nullptr);
	REQUIRE(row.set_image_source(source));
	CHECK(row.has_image_source());
	CHECK_FALSE(lv_obj_has_flag(value_canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(fallback_label, LV_OBJ_FLAG_HIDDEN));
	const lv_draw_buf_t *const canvas_buffer = lv_canvas_get_draw_buf(value_canvas);
	REQUIRE(canvas_buffer != nullptr);
	CHECK(canvas_buffer->data != source->data);
	CHECK(ui::SsidInfoRowTestAccess::canvas_storage(row).size() >=
	      static_cast<std::size_t>(canvas_buffer->header.stride) *
		      canvas_buffer->header.h);

	lv_draw_buf_t invalid_source {};
	CHECK_FALSE(row.set_image_source(&invalid_source));
	REQUIRE(row.set_fallback_text("--"));
	CHECK_FALSE(row.has_image_source());
	CHECK(lv_obj_has_flag(value_canvas, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(fallback_label, LV_OBJ_FLAG_HIDDEN));

	row.destroy();
	CHECK_FALSE(lv_obj_is_valid(root));
	CHECK(ui::SsidInfoRowTestAccess::canvas_storage(row).empty());
}

TEST_CASE("SSID image cache never exposes an arena slot before a complete row commits")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
				      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);

	cache.request_labels({ "pending" }, {});
	CHECK(cache.source_for("pending") == nullptr);
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	const lv_draw_buf_t *const source = cache.source_for("pending");
	REQUIRE(source != nullptr);
	CHECK(source->data != nullptr);
	CHECK(source->data_size >=
	      static_cast<std::size_t>(source->header.stride) * source->header.h);
}

TEST_CASE("SSID image cache returns persistent arena slots when entries clear")
{
	LvglContext lvgl;
	constexpr std::size_t row_bytes =
		static_cast<std::size_t>(RASTER_SIZE) * RASTER_SIZE * 4U;
	ui::SsidTextImageArena arena(row_bytes);

	{
		ui::SsidTextImageCache cache(lv_screen_active(),
						     &lv_font_montserrat_20, RASTER_SIZE,
						     RASTER_SIZE, row_bytes, &arena);
		cache.request_labels({ "first" }, {});
		REQUIRE(cache.render_next() ==
			ui::SsidTextImageCache::RenderResult::Rendered);
	}

	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
					      RASTER_SIZE, RASTER_SIZE, row_bytes, &arena);
	cache.request_labels({ "second" }, {});
	CHECK(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
}

TEST_CASE("SSID image cache evicts least-recent non-visible labels")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
					      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);

	cache.request_labels({ "first" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	cache.request_labels({ "second" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	REQUIRE(ui::SsidTextImageCacheTestAccess::entry_count(cache) == 2U);
	ui::SsidTextImageCacheTestAccess::set_max_bytes(
		cache, ui::SsidTextImageCacheTestAccess::used_bytes(cache));
	cache.request_labels({ "third" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);

	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "first"));
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "second"));
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "third"));
	CHECK(ui::SsidTextImageCacheTestAccess::used_bytes(cache) <=
	      CACHE_TEST_BUDGET);
}

TEST_CASE("SSID image cache renders foreground labels before full-list background work")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
						      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);

	cache.request_labels({ "selected", "previous", "next" },
			     { "before-2", "before-3", "before-4", "before-5",
			       "before-6", "after-2", "after-3", "after-4",
			       "after-5", "after-6", "outside-before", "outside-after" });
	REQUIRE(ui::SsidTextImageCacheTestAccess::foreground_pending_count(cache) ==
		3U);
	REQUIRE(ui::SsidTextImageCacheTestAccess::speculative_pending_count(cache) ==
			12U);
	CHECK(ui::SsidTextImageCacheTestAccess::is_pinned(cache, "selected"));
	CHECK(ui::SsidTextImageCacheTestAccess::is_pinned(cache, "previous"));
	CHECK(ui::SsidTextImageCacheTestAccess::is_pinned(cache, "next"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::is_pinned(cache, "before-2"));

	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "selected"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "previous"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "before-2"));
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "previous"));
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "next"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "before-2"));
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "before-2"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache,
							  "outside-before"));
}

TEST_CASE("SSID image cache keeps visible labels pinned and cancels stale work")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(CACHE_TEST_BUDGET);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
						      RASTER_SIZE, RASTER_SIZE, CACHE_TEST_BUDGET, &arena);

	cache.request_labels({ "first" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	cache.request_labels({ "second" }, {});
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	REQUIRE(ui::SsidTextImageCacheTestAccess::entry_count(cache) == 2U);
	ui::SsidTextImageCacheTestAccess::set_max_bytes(
		cache, ui::SsidTextImageCacheTestAccess::used_bytes(cache));

	cache.request_labels({ "first", "second", "third" }, {});
	REQUIRE(ui::SsidTextImageCacheTestAccess::foreground_pending_count(cache) ==
		1U);
	CHECK(cache.render_next() == ui::SsidTextImageCache::RenderResult::Failed);
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "first"));
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "second"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "third"));

	cache.request_labels({ "stale-a" }, { "stale-b" });
	REQUIRE(ui::SsidTextImageCacheTestAccess::foreground_pending_count(cache) ==
		1U);
	REQUIRE(ui::SsidTextImageCacheTestAccess::speculative_pending_count(cache) ==
		1U);
	cache.request_labels({ "fresh" }, { "fresh-preload" });
	REQUIRE(ui::SsidTextImageCacheTestAccess::foreground_pending_count(cache) ==
		1U);
	REQUIRE(ui::SsidTextImageCacheTestAccess::speculative_pending_count(cache) ==
		1U);
	REQUIRE(cache.render_next() == ui::SsidTextImageCache::RenderResult::Rendered);
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "stale-a"));
	CHECK_FALSE(ui::SsidTextImageCacheTestAccess::contains(cache, "stale-b"));
	CHECK(ui::SsidTextImageCacheTestAccess::contains(cache, "fresh"));
}

TEST_CASE("SSID image cache reports a snapshot that cannot fit its budget")
{
	LvglContext lvgl;
	ui::SsidTextImageArena arena(1U);
	ui::SsidTextImageCache cache(lv_screen_active(), &lv_font_montserrat_20,
					      RASTER_SIZE, RASTER_SIZE, 1U, &arena);
	cache.request_labels({ "oversized" }, {});

	CHECK(cache.render_next() == ui::SsidTextImageCache::RenderResult::Failed);
	CHECK(ui::SsidTextImageCacheTestAccess::entry_count(cache) == 0U);
	CHECK(ui::SsidTextImageCacheTestAccess::used_bytes(cache) == 0U);
}

TEST_CASE("Wi-Fi latency probe keeps bounded timing summaries",
	  "[ui][wifi][ssid][latency]")
{
	ui::WifiLatencyProbeStats stats;
	stats.add(500U);
	stats.add(100U);
	stats.add(300U);
	stats.add(200U);
	stats.add(400U);

	const ui::WifiLatencyProbeSummary summary = stats.summary();
	CHECK(summary.count == 5U);
	CHECK(summary.sampled_count == 5U);
	CHECK(summary.p50_us == 300U);
	CHECK(summary.p95_us == 500U);
	CHECK(summary.max_us == 500U);

	for (std::size_t value = 0U;
	     value < ui::WifiLatencyProbeStats::kSampleCapacity + 3U; ++value) {
		stats.add(value);
	}
	const ui::WifiLatencyProbeSummary bounded = stats.summary();
	CHECK(bounded.count ==
	      5U + ui::WifiLatencyProbeStats::kSampleCapacity + 3U);
	CHECK(bounded.sampled_count == ui::WifiLatencyProbeStats::kSampleCapacity);
	CHECK(bounded.max_us == 500U);

	stats.clear();
	const ui::WifiLatencyProbeSummary cleared = stats.summary();
	CHECK(cleared.count == 0U);
	CHECK(cleared.sampled_count == 0U);
	CHECK(cleared.p50_us == 0U);
	CHECK(cleared.p95_us == 0U);
	CHECK(cleared.max_us == 0U);
}
