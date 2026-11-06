#include "ui/ssid_marquee_window.hpp"

#include <algorithm>
#include <cstring>
#include <limits>

namespace ui
{
namespace
{

constexpr std::size_t kArgb8888BytesPerPixel = sizeof(uint32_t);

template <typename View> [[nodiscard]] bool has_valid_geometry(const View &view) noexcept
{
	if (view.data == nullptr || view.width == 0 || view.height == 0 ||
	    view.width > std::numeric_limits<std::size_t>::max() / kArgb8888BytesPerPixel) {
		return false;
	}

	const std::size_t row_bytes = static_cast<std::size_t>(view.width) * kArgb8888BytesPerPixel;
	if (view.stride_bytes < row_bytes ||
	    view.height > std::numeric_limits<std::size_t>::max() /
				  static_cast<std::size_t>(view.stride_bytes)) {
		return false;
	}
	return static_cast<std::size_t>(view.stride_bytes) * view.height <= view.data_size;
}

void copy_pixels(const SsidArgb8888View &source, const SsidMutableArgb8888View &destination,
		 uint32_t source_x, uint32_t destination_x, uint32_t count) noexcept
{
	if (count == 0) {
		return;
	}
	const std::size_t copy_bytes = static_cast<std::size_t>(count) * kArgb8888BytesPerPixel;
	for (uint32_t row = 0; row < destination.height; ++row) {
		const uint8_t *const source_row =
			source.data + static_cast<std::size_t>(row) * source.stride_bytes;
		uint8_t *const destination_row =
			destination.data + static_cast<std::size_t>(row) * destination.stride_bytes;
		std::memcpy(destination_row + static_cast<std::size_t>(destination_x) *
						      kArgb8888BytesPerPixel,
			    source_row +
				    static_cast<std::size_t>(source_x) * kArgb8888BytesPerPixel,
			    copy_bytes);
	}
}

} // namespace

bool compose_ssid_marquee_window(const SsidArgb8888View &source,
				 const SsidMutableArgb8888View &destination, uint64_t phase_px,
				 uint32_t gap_width_px,
				 SsidMarqueeWindowAlignment alignment) noexcept
{
	if (!has_valid_geometry(source) || !has_valid_geometry(destination) ||
	    source.height != destination.height) {
		return false;
	}

	const std::size_t destination_row_bytes =
		static_cast<std::size_t>(destination.width) * kArgb8888BytesPerPixel;
	for (uint32_t row = 0; row < destination.height; ++row) {
		std::memset(destination.data +
				    static_cast<std::size_t>(row) * destination.stride_bytes,
			    0, destination_row_bytes);
	}

	if (source.width <= destination.width) {
		const uint32_t destination_x = alignment == SsidMarqueeWindowAlignment::Right
						       ? destination.width - source.width
						       : 0;
		copy_pixels(source, destination, 0, destination_x, source.width);
		return true;
	}

	const uint64_t track_width = static_cast<uint64_t>(source.width) + gap_width_px;
	if (track_width == 0) {
		return false;
	}
	const uint64_t phase = phase_px % track_width;
	uint32_t destination_x = 0;
	uint32_t remaining = destination.width;

	if (phase < source.width) {
		const uint32_t first_count =
			static_cast<uint32_t>(std::min<uint64_t>(remaining, source.width - phase));
		copy_pixels(source, destination, static_cast<uint32_t>(phase), destination_x,
			    first_count);
		destination_x += first_count;
		remaining -= first_count;
	}

	if (remaining == 0) {
		return true;
	}

	const uint64_t cursor = phase + destination_x;
	if (cursor < track_width) {
		const uint32_t transparent_count =
			static_cast<uint32_t>(std::min<uint64_t>(remaining, track_width - cursor));
		destination_x += transparent_count;
		remaining -= transparent_count;
	}

	if (remaining > 0) {
		// source.width > destination.width guarantees one wrapped source span
		// finishes the row; no per-pixel modulo loop is needed.
		copy_pixels(source, destination, 0, destination_x, remaining);
	}
	return true;
}

} // namespace ui
