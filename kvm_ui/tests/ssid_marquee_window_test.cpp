#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <limits>
#include <vector>

#include "ui/ssid_marquee_window.hpp"

#include <catch2/catch_test_macros.hpp>

namespace
{

void set_pixel(std::vector<uint8_t> &storage, uint32_t stride, uint32_t x, uint32_t y,
	       uint32_t value)
{
	std::memcpy(storage.data() + static_cast<std::size_t>(y) * stride +
			    static_cast<std::size_t>(x) * sizeof(value),
		    &value, sizeof(value));
}

uint32_t pixel_at(const std::vector<uint8_t> &storage, uint32_t stride, uint32_t x, uint32_t y)
{
	uint32_t value = 0;
	std::memcpy(&value,
		    storage.data() + static_cast<std::size_t>(y) * stride +
			    static_cast<std::size_t>(x) * sizeof(value),
		    sizeof(value));
	return value;
}

ui::SsidArgb8888View source_view(const std::vector<uint8_t> &storage, uint32_t width,
				 uint32_t height, uint32_t stride)
{
	return {
		.data = storage.data(),
		.data_size = storage.size(),
		.width = width,
		.height = height,
		.stride_bytes = stride,
	};
}

ui::SsidMutableArgb8888View destination_view(std::vector<uint8_t> &storage, uint32_t width,
					     uint32_t height, uint32_t stride)
{
	return {
		.data = storage.data(),
		.data_size = storage.size(),
		.width = width,
		.height = height,
		.stride_bytes = stride,
	};
}

} // namespace

TEST_CASE("SSID marquee window aligns a fitting source and preserves row padding",
	  "[wifi][ssid][marquee-window]")
{
	constexpr uint32_t source_width = 3;
	constexpr uint32_t destination_width = 5;
	constexpr uint32_t height = 2;
	constexpr uint32_t source_stride = 16;
	constexpr uint32_t destination_stride = 24;
	std::vector<uint8_t> source(source_stride * height, 0xa5U);
	std::vector<uint8_t> destination(destination_stride * height, 0xcdU);
	for (uint32_t y = 0; y < height; ++y) {
		for (uint32_t x = 0; x < source_width; ++x) {
			set_pixel(source, source_stride, x, y, 1U + x + y * 10U);
		}
	}

	REQUIRE(ui::compose_ssid_marquee_window(
		source_view(source, source_width, height, source_stride),
		destination_view(destination, destination_width, height, destination_stride), 99, 7,
		ui::SsidMarqueeWindowAlignment::Left));
	for (uint32_t y = 0; y < height; ++y) {
		CHECK(pixel_at(destination, destination_stride, 0, y) == 1U + y * 10U);
		CHECK(pixel_at(destination, destination_stride, 1, y) == 2U + y * 10U);
		CHECK(pixel_at(destination, destination_stride, 2, y) == 3U + y * 10U);
		CHECK(pixel_at(destination, destination_stride, 3, y) == 0U);
		CHECK(pixel_at(destination, destination_stride, 4, y) == 0U);
		CHECK(destination[static_cast<std::size_t>(y) * destination_stride + 20U] == 0xcdU);
	}

	REQUIRE(ui::compose_ssid_marquee_window(
		source_view(source, source_width, height, source_stride),
		destination_view(destination, destination_width, height, destination_stride), 1234,
		9, ui::SsidMarqueeWindowAlignment::Right));
	for (uint32_t y = 0; y < height; ++y) {
		CHECK(pixel_at(destination, destination_stride, 0, y) == 0U);
		CHECK(pixel_at(destination, destination_stride, 1, y) == 0U);
		CHECK(pixel_at(destination, destination_stride, 2, y) == 1U + y * 10U);
		CHECK(pixel_at(destination, destination_stride, 3, y) == 2U + y * 10U);
		CHECK(pixel_at(destination, destination_stride, 4, y) == 3U + y * 10U);
		CHECK(destination[static_cast<std::size_t>(y) * destination_stride + 20U] == 0xcdU);
	}
}

TEST_CASE("SSID marquee window composes source gap and wrap spans", "[wifi][ssid][marquee-window]")
{
	constexpr uint32_t source_width = 6;
	constexpr uint32_t destination_width = 4;
	constexpr uint32_t height = 1;
	constexpr uint32_t source_stride = source_width * sizeof(uint32_t);
	constexpr uint32_t destination_stride = destination_width * sizeof(uint32_t);
	std::vector<uint8_t> source(source_stride, 0U);
	std::vector<uint8_t> destination(destination_stride, 0U);
	for (uint32_t x = 0; x < source_width; ++x) {
		set_pixel(source, source_stride, x, 0, x + 1U);
	}
	const auto source_pixels = source_view(source, source_width, height, source_stride);
	const auto destination_pixels =
		destination_view(destination, destination_width, height, destination_stride);
	const auto render = [&](uint64_t phase) {
		REQUIRE(ui::compose_ssid_marquee_window(source_pixels, destination_pixels, phase, 2,
							ui::SsidMarqueeWindowAlignment::Left));
		return std::array<uint32_t, 4>{
			pixel_at(destination, destination_stride, 0, 0),
			pixel_at(destination, destination_stride, 1, 0),
			pixel_at(destination, destination_stride, 2, 0),
			pixel_at(destination, destination_stride, 3, 0),
		};
	};

	CHECK(render(0) == std::array<uint32_t, 4>{1, 2, 3, 4});
	CHECK(render(3) == std::array<uint32_t, 4>{4, 5, 6, 0});
	CHECK(render(5) == std::array<uint32_t, 4>{6, 0, 0, 1});
	CHECK(render(6) == std::array<uint32_t, 4>{0, 0, 1, 2});
	CHECK(render(7) == std::array<uint32_t, 4>{0, 1, 2, 3});
	CHECK(render(std::numeric_limits<uint64_t>::max()) == std::array<uint32_t, 4>{0, 1, 2, 3});
}

TEST_CASE("SSID marquee window handles maximum source width and wide gap phase",
	  "[wifi][ssid][marquee-window]")
{
	constexpr uint32_t source_width = std::numeric_limits<uint16_t>::max();
	constexpr uint32_t destination_width = 230;
	constexpr uint32_t stride = source_width * sizeof(uint32_t);
	std::vector<uint8_t> source(stride, 0U);
	std::vector<uint8_t> destination(destination_width * sizeof(uint32_t), 0xffU);
	set_pixel(source, stride, source_width - 1U, 0, 0x11223344U);

	REQUIRE(ui::compose_ssid_marquee_window(
		source_view(source, source_width, 1, stride),
		destination_view(destination, destination_width, 1,
				 destination_width * sizeof(uint32_t)),
		static_cast<uint64_t>(source_width) - 1U, std::numeric_limits<uint32_t>::max(),
		ui::SsidMarqueeWindowAlignment::Left));
	CHECK(pixel_at(destination, destination_width * sizeof(uint32_t), 0, 0) == 0x11223344U);
	for (uint32_t x = 1; x < destination_width; ++x) {
		CHECK(pixel_at(destination, destination_width * sizeof(uint32_t), x, 0) == 0U);
	}
}

TEST_CASE("SSID marquee window rejects invalid ARGB8888 geometry", "[wifi][ssid][marquee-window]")
{
	std::vector<uint8_t> source(16, 0U);
	std::vector<uint8_t> destination(16, 0U);
	auto valid_source = source_view(source, 4, 1, 16);
	auto valid_destination = destination_view(destination, 4, 1, 16);

	auto invalid_source = valid_source;
	invalid_source.stride_bytes = 15;
	CHECK_FALSE(ui::compose_ssid_marquee_window(invalid_source, valid_destination, 0, 1,
						    ui::SsidMarqueeWindowAlignment::Left));

	auto invalid_destination = valid_destination;
	invalid_destination.data_size = 15;
	CHECK_FALSE(ui::compose_ssid_marquee_window(valid_source, invalid_destination, 0, 1,
						    ui::SsidMarqueeWindowAlignment::Left));

	invalid_destination = valid_destination;
	invalid_destination.height = 2;
	CHECK_FALSE(ui::compose_ssid_marquee_window(valid_source, invalid_destination, 0, 1,
						    ui::SsidMarqueeWindowAlignment::Left));
}
