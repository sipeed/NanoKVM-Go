#pragma once

#include <cstddef>
#include <cstdint>

namespace ui
{

enum class SsidMarqueeWindowAlignment : uint8_t {
	Left,
	Right,
};

struct SsidArgb8888View {
	const uint8_t *data{nullptr};
	std::size_t data_size{0};
	uint32_t width{0};
	uint32_t height{0};
	uint32_t stride_bytes{0};
};

struct SsidMutableArgb8888View {
	uint8_t *data{nullptr};
	std::size_t data_size{0};
	uint32_t width{0};
	uint32_t height{0};
	uint32_t stride_bytes{0};
};

/**
 * Composes one fixed-size marquee window from a natural-width ARGB8888 source.
 *
 * When the source fits, it is copied once using the requested alignment. When
 * it overflows, phase_px addresses a repeating [source, transparent gap] track.
 * Destination row padding is deliberately left untouched.
 */
[[nodiscard]] bool compose_ssid_marquee_window(const SsidArgb8888View &source,
					       const SsidMutableArgb8888View &destination,
					       uint64_t phase_px, uint32_t gap_width_px,
					       SsidMarqueeWindowAlignment alignment) noexcept;

} // namespace ui
