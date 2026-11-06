#include "lvgl_main/ssid_raster_worker.hpp"

#include <algorithm>
#include <cmath>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <limits>
#include <mutex>
#include <optional>
#include <string_view>
#include <thread>
#include <unordered_set>
#include <utility>
#include <vector>

#include <fcntl.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <unistd.h>

#if defined(__linux__)
#include <sys/resource.h>
#include <sys/syscall.h>
#endif

/*
 * This is intentionally the sole private stb_truetype implementation unit for
 * the SSID worker.  It does not include lvgl.h or lv_tiny_ttf.h: the worker
 * owns read-only font mappings and produces only value-only ARGB pixels.
 */
#define STBTT_STATIC
#define STB_TRUETYPE_IMPLEMENTATION
#include <src/libs/tiny_ttf/stb_truetype_htcw.h>

namespace lvgl_main
{
namespace
{

constexpr std::size_t kHardMaxRasterBytes = 4U * 1024U * 1024U;
constexpr std::size_t kHardMaxGlyphBytes = 1024U * 1024U;
constexpr std::uint16_t kHardMaxDimension = 4096;
constexpr std::size_t kHardMaxCodepoints = 128;

[[nodiscard]] bool checked_multiply(std::size_t lhs, std::size_t rhs, std::size_t &out) noexcept
{
	if (lhs != 0 && rhs > std::numeric_limits<std::size_t>::max() / lhs) {
		return false;
	}
	out = lhs * rhs;
	return true;
}

[[nodiscard]] bool checked_add(std::int32_t lhs, std::int32_t rhs, std::int32_t &out) noexcept
{
	if ((rhs > 0 && lhs > std::numeric_limits<std::int32_t>::max() - rhs) ||
	    (rhs < 0 && lhs < std::numeric_limits<std::int32_t>::min() - rhs)) {
		return false;
	}
	out = lhs + rhs;
	return true;
}

[[nodiscard]] std::string native_font_path(std::string path)
{
	if (path.rfind("A:", 0) == 0) {
		path.erase(0, 2);
	}
	if (path.empty() || path.front() != '/') {
		return {};
	}
	return path;
}

class MappedFontFile final {
    public:
	MappedFontFile() = default;
	~MappedFontFile() { reset(); }

	MappedFontFile(const MappedFontFile &) = delete;
	MappedFontFile &operator=(const MappedFontFile &) = delete;

	void reset() noexcept
	{
		if (_address != MAP_FAILED) {
			::munmap(_address, _size);
			_address = MAP_FAILED;
			_size = 0;
		}
	}

	[[nodiscard]] bool load(const std::string &configured_path, std::uint32_t max_font_bytes)
	{
		reset();
		const std::string path = native_font_path(configured_path);
		if (path.empty() || max_font_bytes == 0 || max_font_bytes > kHardMaxRasterBytes * 16U) {
			return false;
		}

		const int fd = ::open(path.c_str(), O_RDONLY | O_CLOEXEC);
		if (fd < 0) {
			return false;
		}
		struct stat status {};
		if (::fstat(fd, &status) != 0 || !S_ISREG(status.st_mode) || status.st_size <= 0 ||
		    static_cast<std::uintmax_t>(status.st_size) > max_font_bytes ||
		    static_cast<std::uintmax_t>(status.st_size) > std::numeric_limits<std::size_t>::max()) {
			::close(fd);
			return false;
		}

		const auto size = static_cast<std::size_t>(status.st_size);
		// Fault in only the pages needed by the renderer. Clean file pages can
		// be reclaimed without retaining an anonymous copy in RAM or zram.
		void *address = ::mmap(nullptr, size, PROT_READ, MAP_PRIVATE, fd, 0);
		::close(fd);
		if (address == MAP_FAILED) {
			return false;
		}
		_address = address;
		_size = size;
		return true;
	}

	[[nodiscard]] const unsigned char *data() const noexcept
	{
		return static_cast<const unsigned char *>(_address);
	}

    private:
	void *_address {MAP_FAILED};
	std::size_t _size {0};
};

[[nodiscard]] bool decode_utf8(std::string_view value, std::vector<std::uint32_t> &codepoints)
{
	if (value.empty() || value.size() > std::numeric_limits<std::uint32_t>::max()) {
		return false;
	}

	codepoints.clear();
	codepoints.reserve(std::min(value.size(), kHardMaxCodepoints));
	for (std::size_t index = 0; index < value.size();) {
		const auto first = static_cast<std::uint8_t>(value[index]);
		std::uint32_t codepoint = 0;
		std::size_t continuation_count = 0;
		if (first <= 0x7fU) {
			codepoint = first;
		} else if (first >= 0xc2U && first <= 0xdfU) {
			codepoint = first & 0x1fU;
			continuation_count = 1;
		} else if (first >= 0xe0U && first <= 0xefU) {
			codepoint = first & 0x0fU;
			continuation_count = 2;
		} else if (first >= 0xf0U && first <= 0xf4U) {
			codepoint = first & 0x07U;
			continuation_count = 3;
		} else {
			return false;
		}

		if (index + continuation_count >= value.size()) {
			return false;
		}
		for (std::size_t offset = 1; offset <= continuation_count; ++offset) {
			const auto next = static_cast<std::uint8_t>(value[index + offset]);
			if ((next & 0xc0U) != 0x80U) {
				return false;
			}
			codepoint = (codepoint << 6U) | (next & 0x3fU);
		}

		const bool overlong = (continuation_count == 1 && codepoint < 0x80U) ||
				      (continuation_count == 2 && codepoint < 0x800U) ||
				      (continuation_count == 3 && codepoint < 0x10000U);
		if (overlong || codepoint > 0x10ffffU ||
		    (codepoint >= 0xd800U && codepoint <= 0xdfffU) || codepoint < 0x20U ||
		    codepoint == 0x7fU) {
			return false;
		}

		codepoints.push_back(codepoint);
		if (codepoints.size() > kHardMaxCodepoints) {
			return false;
		}
		index += continuation_count + 1;
	}
	return !codepoints.empty();
}

[[nodiscard]] bool is_valid_request(const ui::SsidRasterRequest &request,
				    const SsidRasterWorkerConfig &config)
{
	const ui::SsidRasterIdentity &identity = request.identity;
	if (identity.page_token == 0 || identity.generation == 0 || identity.font_generation == 0 ||
	    identity.viewport_width == 0 || identity.height == 0 ||
	    identity.viewport_width > kHardMaxDimension || identity.height > kHardMaxDimension ||
	    identity.font_size_px != config.font_size_px || identity.font_size_px == 0 ||
	    identity.byte_budget == 0 || identity.byte_budget > kHardMaxRasterBytes ||
	    identity.display_text.empty() || identity.display_text.size() > config.max_text_bytes ||
	    identity.alignment != ui::SsidRasterAlignment::Right) {
		return false;
	}

	std::size_t static_bytes = 0;
	return checked_multiply(identity.viewport_width, identity.height, static_bytes) &&
	       checked_multiply(static_bytes, sizeof(std::uint32_t), static_bytes) &&
	       static_bytes <= identity.byte_budget;
}

struct FontFace {
	MappedFontFile bytes;
	stbtt_fontinfo info {};
	float scale {0.0F};
	int ascent {0};
	int descent {0};
	int line_gap {0};

	void reset() noexcept
	{
		info = {};
		bytes.reset();
		scale = 0.0F;
		ascent = 0;
		descent = 0;
		line_gap = 0;
	}

	[[nodiscard]] bool load(const std::string &path, const SsidRasterWorkerConfig &config)
	{
		if (!bytes.load(path, config.max_font_bytes)) {
			return false;
		}
		const int offset = stbtt_GetFontOffsetForIndex(bytes.data(), 0);
		if (offset < 0 || stbtt_InitFont(&info, bytes.data(), offset) == 0) {
			reset();
			return false;
		}
		scale = stbtt_ScaleForMappingEmToPixels(&info, config.font_size_px);
		if (!(scale > 0.0F)) {
			reset();
			return false;
		}
		stbtt_GetFontVMetrics(&info, &ascent, &descent, &line_gap);
		return true;
	}
};

struct FontSet {
	FontFace base;
	FontFace cjk;

	void reset() noexcept
	{
		base.reset();
		cjk.reset();
	}

	[[nodiscard]] bool load(const SsidRasterWorkerConfig &config)
	{
		const bool ready = base.load(config.base_font_path, config) &&
				   cjk.load(config.cjk_font_path, config);
		if (!ready) {
			reset();
		}
		return ready;
	}
};

struct Glyph {
	const FontFace *face {nullptr};
	int glyph_index {0};
	std::int32_t advance {0};
	int x1 {0};
	int y1 {0};
	int x2 {0};
	int y2 {0};
};

[[nodiscard]] bool select_glyph(const FontSet &fonts, std::uint32_t codepoint, Glyph &glyph)
{
	const FontFace *face = &fonts.base;
	int glyph_index = stbtt_FindGlyphIndex(&face->info, static_cast<int>(codepoint));
	if (glyph_index == 0) {
		face = &fonts.cjk;
		glyph_index = stbtt_FindGlyphIndex(&face->info, static_cast<int>(codepoint));
	}
	if (glyph_index == 0) {
		face = &fonts.base;
		glyph_index = stbtt_FindGlyphIndex(&face->info, 0xfffd);
	}
	if (glyph_index == 0) {
		face = &fonts.cjk;
		glyph_index = stbtt_FindGlyphIndex(&face->info, 0xfffd);
	}
	if (glyph_index == 0) {
		return false;
	}

	int advance_width = 0;
	int left_side_bearing = 0;
	stbtt_GetGlyphHMetrics(&face->info, glyph_index, &advance_width, &left_side_bearing);
	const auto advance = static_cast<std::int32_t>(
	    std::floor(static_cast<float>(advance_width) * face->scale + 0.5F));
	if (advance < 0) {
		return false;
	}

	glyph.face = face;
	glyph.glyph_index = glyph_index;
	glyph.advance = advance;
	stbtt_GetGlyphBitmapBox(&face->info, glyph_index, face->scale, face->scale, &glyph.x1,
				&glyph.y1, &glyph.x2, &glyph.y2);
	return true;
}

[[nodiscard]] ui::SsidPreparedRasterResult make_failure(const ui::SsidRasterIdentity &identity)
{
	return {.identity = identity, .raster = nullptr};
}

[[nodiscard]] ui::SsidPreparedRasterResult
rasterize_request(const SsidRasterWorkerConfig &config, const FontSet &fonts,
		  const ui::SsidRasterRequest &request,
		  const SsidRasterWorker::CancelPredicate &is_cancelled)
{
	const ui::SsidRasterIdentity &identity = request.identity;
	if (!is_valid_request(request, config) || is_cancelled()) {
		return make_failure(identity);
	}

	std::vector<std::uint32_t> codepoints;
	if (!decode_utf8(identity.display_text, codepoints) || is_cancelled()) {
		return make_failure(identity);
	}

	std::vector<Glyph> glyphs;
	glyphs.reserve(codepoints.size());
	std::int32_t text_width = 0;
	for (std::size_t index = 0; index < codepoints.size(); ++index) {
		if (is_cancelled()) {
			return make_failure(identity);
		}

		Glyph glyph;
		if (!select_glyph(fonts, codepoints[index], glyph)) {
			return make_failure(identity);
		}
		if (index + 1 < codepoints.size()) {
			Glyph next;
			if (!select_glyph(fonts, codepoints[index + 1], next)) {
				return make_failure(identity);
			}
			if (next.face == glyph.face) {
				const int kerning = stbtt_GetGlyphKernAdvance(
				    &glyph.face->info, glyph.glyph_index, next.glyph_index);
				const auto kerning_px = static_cast<std::int32_t>(std::floor(
				    static_cast<float>(kerning) * glyph.face->scale + 0.5F));
				if (!checked_add(glyph.advance, kerning_px, glyph.advance) ||
				    glyph.advance < 0) {
					return make_failure(identity);
				}
			}
		}
		if (!checked_add(text_width, glyph.advance, text_width)) {
			return make_failure(identity);
		}
		glyphs.push_back(glyph);
	}
	if (text_width <= 0) {
		return make_failure(identity);
	}

	const bool overflows = identity.scroll_enabled &&
			       text_width > static_cast<std::int32_t>(identity.viewport_width);
	const std::size_t source_width = overflows
					     ? static_cast<std::size_t>(text_width)
					     : static_cast<std::size_t>(identity.viewport_width);
	if (source_width == 0 || source_width > std::numeric_limits<std::uint16_t>::max()) {
		return make_failure(identity);
	}
	std::size_t pixel_count = 0;
	std::size_t byte_count = 0;
	if (!checked_multiply(source_width, identity.height, pixel_count) ||
	    !checked_multiply(pixel_count, sizeof(std::uint32_t), byte_count)) {
		return make_failure(identity);
	}
	std::size_t persistent_bytes = byte_count;
	if (overflows) {
		std::size_t window_pixel_count = 0;
		std::size_t window_bytes = 0;
		if (!checked_multiply(identity.viewport_width, identity.height,
				      window_pixel_count) ||
		    !checked_multiply(window_pixel_count, sizeof(std::uint32_t),
				      window_bytes) ||
		    byte_count > std::numeric_limits<std::size_t>::max() - window_bytes) {
			return make_failure(identity);
		}
		persistent_bytes += window_bytes;
	}
	if (persistent_bytes > identity.byte_budget ||
	    persistent_bytes > kHardMaxRasterBytes) {
		return make_failure(identity);
	}

	std::vector<std::uint32_t> pixels(pixel_count, 0U);
	const int line_height = static_cast<int>(std::floor(
	    fonts.base.scale *
	    static_cast<float>(fonts.base.ascent - fonts.base.descent + fonts.base.line_gap)));
	const int base_line = static_cast<int>(std::floor(
	    fonts.base.scale * static_cast<float>(fonts.base.line_gap - fonts.base.descent)));
	const int top = (static_cast<int>(identity.height) - line_height) / 2;
	// LVGL measures base_line upward from the bottom of line_height. Its
	// draw-label path places TinyTTF's inclusive (+1) glyph box one pixel above
	// the mathematical baseline; mirror that snapshot geometry exactly.
	const int baseline = top + line_height - base_line - 1;
	std::int32_t pen_x =
	    overflows ? 0 : static_cast<std::int32_t>(identity.viewport_width) - text_width;

	for (const Glyph &glyph : glyphs) {
		if (is_cancelled()) {
			return make_failure(identity);
		}

		const int glyph_width = glyph.x2 - glyph.x1 + 1;
		const int glyph_height = glyph.y2 - glyph.y1 + 1;
		if (glyph_width > 0 && glyph_height > 0) {
			std::size_t glyph_size = 0;
			if (glyph_width > static_cast<int>(kHardMaxDimension) ||
			    glyph_height > static_cast<int>(kHardMaxDimension) ||
			    !checked_multiply(static_cast<std::size_t>(glyph_width),
					      static_cast<std::size_t>(glyph_height), glyph_size) ||
			    glyph_size > kHardMaxGlyphBytes || glyph_size > identity.byte_budget) {
				return make_failure(identity);
			}

			std::vector<std::uint8_t> bitmap(glyph_size, 0U);
			stbtt_MakeGlyphBitmap(&glyph.face->info, bitmap.data(), glyph_width,
					      glyph_height, glyph_width, glyph.face->scale,
					      glyph.face->scale, glyph.glyph_index);
			const std::int32_t origin_x = pen_x + glyph.x1;
			const int origin_y = baseline + glyph.y1;
			for (int row = 0; row < glyph_height; ++row) {
				if (is_cancelled()) {
					return make_failure(identity);
				}
				const int destination_y = origin_y + row;
				if (destination_y < 0 || destination_y >= identity.height) {
					continue;
				}
				for (int column = 0; column < glyph_width; ++column) {
					const std::int32_t destination_x = origin_x + column;
					if (destination_x < 0 ||
					    destination_x >=
						static_cast<std::int32_t>(source_width)) {
						continue;
					}
					const auto alpha =
					    bitmap[static_cast<std::size_t>(row) * glyph_width +
						   static_cast<std::size_t>(column)];
					const std::size_t pixel_index =
					    static_cast<std::size_t>(destination_y) * source_width +
					    static_cast<std::size_t>(destination_x);
					if (alpha >
					    static_cast<std::uint8_t>(pixels[pixel_index] >> 24U)) {
						pixels[pixel_index] =
						    (static_cast<std::uint32_t>(alpha) << 24U) |
						    0x00ffffffU;
					}
				}
			}
		}
		if (!checked_add(pen_x, glyph.advance, pen_x)) {
			return make_failure(identity);
		}
	}

	Glyph space;
	if (!select_glyph(fonts, 0x20U, space) || space.advance <= 0) {
		return make_failure(identity);
	}

	auto raster = std::make_shared<ui::SsidPreparedRaster>();
	raster->identity = identity;
	raster->source_width = static_cast<std::uint16_t>(source_width);
	raster->source_height = identity.height;
	raster->stride_bytes = static_cast<std::uint32_t>(source_width * sizeof(std::uint32_t));
	raster->text_width = text_width;
	raster->space_advance = space.advance;
	raster->overflows_viewport = overflows;
	raster->pixels = std::make_shared<const std::vector<std::uint32_t>>(std::move(pixels));
	if (!raster->has_valid_storage() || is_cancelled()) {
		return make_failure(identity);
	}
	return {.identity = identity, .raster = std::move(raster)};
}

} // namespace

struct SsidRasterWorker::Impl {
	struct PendingRequest {
		ui::SsidRasterRequest request;
		std::uint64_t revision {0};
	};

	SsidRasterWorkerConfig config;
	Completion completion;
	TestRasterizer test_rasterizer;
	std::mutex mutex;
	std::condition_variable wake;
	std::optional<PendingRequest> pending;
	std::unordered_set<std::uint64_t> invalidated_page_tokens;
	std::uint64_t revision {0};
	bool closing {false};
	std::thread thread;

	Impl(SsidRasterWorkerConfig initial_config, Completion initial_completion,
	     TestRasterizer initial_test_rasterizer)
	    : config(std::move(initial_config)), completion(std::move(initial_completion)),
	      test_rasterizer(std::move(initial_test_rasterizer))
	{
	}

	[[nodiscard]] bool cancelled(std::uint64_t request_revision, std::uint64_t page_token)
	{
		const std::lock_guard lock(mutex);
		return closing || request_revision != revision ||
		       invalidated_page_tokens.contains(page_token);
	}

	void run() noexcept
	{
#if defined(__linux__)
		const auto thread_id = static_cast<id_t>(::syscall(SYS_gettid));
		(void)::setpriority(PRIO_PROCESS, thread_id, 10);
#endif

		FontSet fonts;
		bool fonts_attempted = static_cast<bool>(test_rasterizer);
		bool fonts_ready = static_cast<bool>(test_rasterizer);
		for (;;) {
			PendingRequest current;
			{
				std::unique_lock lock(mutex);
				wake.wait(lock, [this] { return closing || pending.has_value(); });
				if (closing) {
					return;
				}
				current = std::move(*pending);
				pending.reset();
			}

			const CancelPredicate is_cancelled =
			    [this, request_revision = current.revision,
			     page_token = current.request.identity.page_token] {
				    return cancelled(request_revision, page_token);
			    };
			if (is_cancelled()) {
				continue;
			}

			ui::SsidPreparedRasterResult result =
			    make_failure(current.request.identity);
			try {
				if (!fonts_attempted) {
					fonts_attempted = true;
					fonts_ready = fonts.load(config);
				}
				if (fonts_ready && !is_cancelled()) {
					result =
					    test_rasterizer
						? test_rasterizer(current.request, is_cancelled)
						: rasterize_request(config, fonts, current.request,
								    is_cancelled);
				}
			} catch (...) {
				// Allocation, file and injected-test failures degrade
				// to the compiled UI fallback; they must never
				// terminate the process.
				fonts.reset();
				fonts_attempted = static_cast<bool>(test_rasterizer);
				fonts_ready = static_cast<bool>(test_rasterizer);
				result = make_failure(current.request.identity);
			}
			result.identity = current.request.identity;
			if (result.raster && (result.raster->identity != result.identity ||
					      !result.raster->has_valid_storage())) {
				result.raster.reset();
			}

			{
				std::lock_guard lock(mutex);
				if (closing || current.revision != revision ||
				    invalidated_page_tokens.contains(
					current.request.identity.page_token)) {
					continue;
				}
			}
			// Do not hold the worker mutex while handing off. submit() is
			// called on the latency-sensitive LVGL thread. A request that
			// overtakes this point is still rejected by the UI's complete
			// identity guard.
			try {
				if (completion) {
					completion(std::move(result));
				}
			} catch (...) {
				// Completion failures must not terminate the
				// application worker.
			}
		}
	}
};

SsidRasterWorker::SsidRasterWorker(SsidRasterWorkerConfig config, Completion completion,
				   TestRasterizer test_rasterizer)
    : _impl(std::make_unique<Impl>(std::move(config), std::move(completion),
				   std::move(test_rasterizer)))
{
	_impl->thread = std::thread([impl = _impl.get()] { impl->run(); });
}

SsidRasterWorker::~SsidRasterWorker()
{
	close();
}

bool SsidRasterWorker::submit(ui::SsidRasterRequest request)
{
	if (!_impl || !is_valid_request(request, _impl->config)) {
		return false;
	}

	{
		const std::lock_guard lock(_impl->mutex);
		if (_impl->closing) {
			return false;
		}
		_impl->invalidated_page_tokens.erase(request.identity.page_token);
		_impl->pending = Impl::PendingRequest {
		    .request = std::move(request),
		    .revision = ++_impl->revision,
		};
	}
	_impl->wake.notify_one();
	return true;
}

void SsidRasterWorker::invalidate_page(std::uint64_t page_token) noexcept
{
	if (!_impl || page_token == 0) {
		return;
	}
	{
		const std::lock_guard lock(_impl->mutex);
		try {
			_impl->invalidated_page_tokens.insert(page_token);
		} catch (...) {
			// Preserve noexcept teardown under allocation pressure. Revision
			// cancellation is conservative (it may also cancel another page's
			// work), but no stale result may escape.
			++_impl->revision;
		}
		if (_impl->pending && _impl->pending->request.identity.page_token == page_token) {
			_impl->pending.reset();
		}
	}
	_impl->wake.notify_one();
}

void SsidRasterWorker::close() noexcept
{
	if (!_impl) {
		return;
	}
	{
		const std::lock_guard lock(_impl->mutex);
		_impl->closing = true;
		_impl->pending.reset();
	}
	_impl->wake.notify_all();
	if (_impl->thread.joinable()) {
		_impl->thread.join();
	}
}

} // namespace lvgl_main
