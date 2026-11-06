#pragma once

#include <cstddef>
#include <cstdint>
#include <limits>
#include <memory>
#include <string>
#include <vector>

namespace ui
{

enum class SsidRasterAlignment : uint8_t {
	Right,
};

/**
 * Value-only render identity shared between the UI thread and the raster worker.
 *
 * This header deliberately has no LVGL dependency.  A result remains meaningful
 * after its originating page has gone away, so consumers must validate every
 * identity field before installing it into an LVGL object.
 */
struct SsidRasterIdentity {
	uint64_t page_token {0};
	uint64_t generation {0};
	uint64_t font_generation {0};
	uint16_t viewport_width {0};
	uint16_t height {0};
	uint16_t font_size_px {0};
	SsidRasterAlignment alignment {SsidRasterAlignment::Right};
	bool scroll_enabled {false};
	uint32_t byte_budget {0};
	std::string display_text;

	bool operator==(const SsidRasterIdentity &other) const noexcept
	{
		return page_token == other.page_token && generation == other.generation &&
		       font_generation == other.font_generation &&
		       viewport_width == other.viewport_width && height == other.height &&
		       font_size_px == other.font_size_px && alignment == other.alignment &&
		       scroll_enabled == other.scroll_enabled && byte_budget == other.byte_budget &&
		       display_text == other.display_text;
	}

	bool same_render_input(const SsidRasterIdentity &other) const noexcept
	{
		return page_token == other.page_token && font_generation == other.font_generation &&
		       viewport_width == other.viewport_width && height == other.height &&
		       font_size_px == other.font_size_px && alignment == other.alignment &&
		       scroll_enabled == other.scroll_enabled && byte_budget == other.byte_budget &&
		       display_text == other.display_text;
	}
};

struct SsidRasterRequest {
	SsidRasterIdentity identity;
};

/**
 * Pixels are logical ARGB8888 values (0xAARRGGBB), not an LVGL draw buffer.
 * The LVGL-thread adapter is the only code allowed to turn this into an LVGL
 * image source.
 */
struct SsidPreparedRaster {
	SsidRasterIdentity identity;
	uint16_t source_width {0};
	uint16_t source_height {0};
	uint32_t stride_bytes {0};
	int32_t text_width {0};
	int32_t space_advance {0};
	bool overflows_viewport {false};
	std::shared_ptr<const std::vector<uint32_t>> pixels;

	bool has_valid_storage() const noexcept
	{
		if (identity.viewport_width == 0 || identity.height == 0 ||
		    identity.byte_budget == 0 || source_width == 0 ||
		    source_height != identity.height || text_width <= 0 || space_advance <= 0 ||
		    !pixels) {
			return false;
		}

		const std::size_t width = source_width;
		const std::size_t height = source_height;
		if (width > std::numeric_limits<std::size_t>::max() / sizeof(uint32_t) ||
		    stride_bytes != width * sizeof(uint32_t) ||
		    height > std::numeric_limits<std::size_t>::max() / stride_bytes ||
		    width > std::numeric_limits<std::size_t>::max() / height) {
			return false;
		}
		const std::size_t source_bytes =
			static_cast<std::size_t>(stride_bytes) * height;
		std::size_t required_bytes = source_bytes;
		if (overflows_viewport) {
			if (!identity.scroll_enabled || source_width <= identity.viewport_width ||
			    text_width <= static_cast<int32_t>(identity.viewport_width) ||
			    source_width != static_cast<uint32_t>(text_width)) {
				return false;
			}
			const std::size_t viewport_width = identity.viewport_width;
			if (viewport_width >
			    std::numeric_limits<std::size_t>::max() / sizeof(uint32_t)) {
				return false;
			}
			const std::size_t viewport_stride =
				viewport_width * sizeof(uint32_t);
			if (height > std::numeric_limits<std::size_t>::max() /
					     viewport_stride) {
				return false;
			}
			const std::size_t viewport_bytes = viewport_stride * height;
			if (source_bytes > std::numeric_limits<std::size_t>::max() -
						   viewport_bytes) {
				return false;
			}
			required_bytes += viewport_bytes;
		} else if (source_width != identity.viewport_width ||
			   (identity.scroll_enabled &&
			    text_width > static_cast<int32_t>(identity.viewport_width))) {
			return false;
		}
		return required_bytes <= identity.byte_budget &&
		       pixels->size() == width * height;
	}
};

struct SsidPreparedRasterResult {
	SsidRasterIdentity identity;
	std::shared_ptr<const SsidPreparedRaster> raster;

	bool is_success() const noexcept
	{
		return raster && raster->identity == identity && raster->has_valid_storage();
	}
};

} // namespace ui
