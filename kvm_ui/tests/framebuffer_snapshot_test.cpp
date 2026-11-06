#include <array>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <limits>
#include <span>
#include <string>
#include <system_error>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include <linux/fb.h>

#include "lvgl_main/framebuffer_snapshot.hpp"

using namespace lvgl_main::framebuffer_snapshot;

namespace
{

constexpr ChannelLayout kRed { 11, 5 };
constexpr ChannelLayout kGreen { 5, 6 };
constexpr ChannelLayout kBlue { 0, 5 };

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_snapshot_test_" + std::to_string(stamp));
		std::filesystem::create_directories(_path);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const
	{
		return _path;
	}

    private:
	std::filesystem::path _path;
};

void write_le16(std::vector<uint8_t> &bytes, std::size_t offset, uint16_t value)
{
	REQUIRE(offset + 1U < bytes.size());
	bytes[offset] = static_cast<uint8_t>(value & 0xFFU);
	bytes[offset + 1U] = static_cast<uint8_t>(value >> 8U);
}

uint16_t read_le16(const std::vector<uint8_t> &bytes, std::size_t offset)
{
	REQUIRE(offset + 1U < bytes.size());
	return static_cast<uint16_t>(bytes[offset]) |
	       static_cast<uint16_t>(bytes[offset + 1U] << 8U);
}

uint32_t read_le32(const std::vector<uint8_t> &bytes, std::size_t offset)
{
	REQUIRE(offset + 3U < bytes.size());
	return static_cast<uint32_t>(bytes[offset]) |
	       (static_cast<uint32_t>(bytes[offset + 1U]) << 8U) |
	       (static_cast<uint32_t>(bytes[offset + 2U]) << 16U) |
	       (static_cast<uint32_t>(bytes[offset + 3U]) << 24U);
}

std::vector<uint8_t> bmp_pixels_top_down(const EncodeResult &result)
{
	constexpr std::size_t kBmpHeaderSize = 54;
	REQUIRE(result.bytes.size() >= kBmpHeaderSize);
	CHECK(result.bytes[0] == 'B');
	CHECK(result.bytes[1] == 'M');
	CHECK(read_le32(result.bytes, 2) == result.bytes.size());
	CHECK(read_le16(result.bytes, 6) == 0);
	CHECK(read_le16(result.bytes, 8) == 0);
	CHECK(read_le32(result.bytes, 10) == kBmpHeaderSize);
	CHECK(read_le32(result.bytes, 14) == 40);
	CHECK(read_le32(result.bytes, 18) == result.width);
	CHECK(read_le32(result.bytes, 22) == result.height);
	CHECK(read_le16(result.bytes, 26) == 1);
	CHECK(read_le16(result.bytes, 28) == 24);
	CHECK(read_le32(result.bytes, 30) == 0);
	CHECK(read_le32(result.bytes, 38) == 0);
	CHECK(read_le32(result.bytes, 42) == 0);
	CHECK(read_le32(result.bytes, 46) == 0);
	CHECK(read_le32(result.bytes, 50) == 0);

	const std::size_t row_bytes = static_cast<std::size_t>(result.width) * 3U;
	const std::size_t padded_row_bytes = (row_bytes + 3U) & ~std::size_t { 3U };
	REQUIRE(read_le32(result.bytes, 34) == padded_row_bytes * result.height);
	REQUIRE(result.bytes.size() == kBmpHeaderSize +
					 padded_row_bytes * result.height);

	std::vector<uint8_t> pixels;
	pixels.reserve(row_bytes * result.height);
	for (uint32_t output_y = 0; output_y < result.height; ++output_y) {
		const uint32_t file_y = result.height - 1U - output_y;
		const std::size_t row_offset =
			kBmpHeaderSize + static_cast<std::size_t>(file_y) * padded_row_bytes;
		for (uint32_t output_x = 0; output_x < result.width; ++output_x) {
			const std::size_t pixel_offset = row_offset + output_x * 3U;
			pixels.push_back(result.bytes[pixel_offset + 2U]);
			pixels.push_back(result.bytes[pixel_offset + 1U]);
			pixels.push_back(result.bytes[pixel_offset]);
		}
		for (std::size_t padding = row_bytes; padding < padded_row_bytes;
		     ++padding) {
			CHECK(result.bytes[row_offset + padding] == 0);
		}
	}
	return pixels;
}

void append_rgb(std::vector<uint8_t> &out, uint16_t pixel)
{
	const auto rgb = rgb565_to_rgb888(pixel);
	out.insert(out.end(), rgb.begin(), rgb.end());
}

std::vector<uint8_t> read_binary(const std::filesystem::path &path)
{
	std::ifstream in(path, std::ios::binary);
	REQUIRE(in.good());
	return { std::istreambuf_iterator<char>(in),
		 std::istreambuf_iterator<char>() };
}

void write_binary(const std::filesystem::path &path,
		  std::span<const uint8_t> bytes)
{
	std::ofstream out(path, std::ios::binary | std::ios::trunc);
	REQUIRE(out.good());
	out.write(reinterpret_cast<const char *>(bytes.data()),
		  static_cast<std::streamsize>(bytes.size()));
	REQUIRE(out.good());
}

std::size_t count_temporary_files(const std::filesystem::path &directory)
{
	std::size_t count = 0;
	for (const auto &entry :
	     std::filesystem::directory_iterator(directory)) {
		if (entry.path().filename().string().find(".tmp") !=
		    std::string::npos) {
			++count;
		}
	}
	return count;
}

} // namespace

TEST_CASE("RGB565 expands channels with bit replication")
{
	CHECK(rgb565_to_rgb888(0x0000U) == std::array<uint8_t, 3> { 0, 0, 0 });
	CHECK(rgb565_to_rgb888(0xFFFFU) ==
	      std::array<uint8_t, 3> { 255, 255, 255 });
	CHECK(rgb565_to_rgb888(0xF800U) ==
	      std::array<uint8_t, 3> { 255, 0, 0 });
	CHECK(rgb565_to_rgb888(0x07E0U) ==
	      std::array<uint8_t, 3> { 0, 255, 0 });
	CHECK(rgb565_to_rgb888(0x001FU) ==
	      std::array<uint8_t, 3> { 0, 0, 255 });
	CHECK(rgb565_to_rgb888(0x8410U) ==
	      std::array<uint8_t, 3> { 132, 130, 132 });
}

TEST_CASE("visible RGB565 copy honors stride and source offsets")
{
	FramebufferGeometry geometry {
		.width = 2,
		.height = 2,
		.virtual_width = 4,
		.virtual_height = 3,
		.bits_per_pixel = 16,
		.line_length = 8,
		.xoffset = 1,
		.yoffset = 1,
		.red = kRed,
		.green = kGreen,
		.blue = kBlue,
	};
	std::vector<uint8_t> mapping(24, 0);
	write_le16(mapping, 10, 0xF800U);
	write_le16(mapping, 12, 0x07E0U);
	write_le16(mapping, 18, 0x001FU);
	write_le16(mapping, 20, 0xFFFFU);

	const auto result = copy_visible_rgb565(mapping, geometry);

	REQUIRE(result.ok);
	CHECK(result.frame.width == 2);
	CHECK(result.frame.height == 2);
	CHECK(result.frame.pixels ==
	      std::vector<uint16_t> { 0xF800U, 0x07E0U, 0x001FU, 0xFFFFU });
}

TEST_CASE("BMP encoding maps clockwise 90 and 270 degree views")
{
	const CapturedFrame frame {
		.width = 2,
		.height = 3,
		.pixels = { 0x0000U, 0xF800U, 0x07E0U, 0x001FU, 0xFFFFU,
			    0xFFE0U },
	};

	const auto clockwise90 = encode_bmp(frame, ViewRotation::Clockwise90);
	REQUIRE(clockwise90.ok);
	CHECK(clockwise90.width == 3);
	CHECK(clockwise90.height == 2);
	std::vector<uint8_t> expected90;
	for (const uint16_t pixel :
	     { 0xF800U, 0x001FU, 0xFFE0U, 0x0000U, 0x07E0U, 0xFFFFU }) {
		append_rgb(expected90, pixel);
	}
	CHECK(bmp_pixels_top_down(clockwise90) == expected90);

	const auto clockwise270 = encode_bmp(frame, ViewRotation::Clockwise270);
	REQUIRE(clockwise270.ok);
	std::vector<uint8_t> expected270;
	for (const uint16_t pixel :
	     { 0xFFFFU, 0x07E0U, 0x0000U, 0xFFE0U, 0x001FU, 0xF800U }) {
		append_rgb(expected270, pixel);
	}
	CHECK(bmp_pixels_top_down(clockwise270) == expected270);
}

TEST_CASE("BMP encoding preserves LVGL snapshot orientation")
{
	const CapturedFrame frame {
		.width = 2,
		.height = 1,
		.pixels = { 0xF800U, 0x07E0U },
	};

	const auto encoded = encode_bmp(frame, ViewRotation::None);

	REQUIRE(encoded.ok);
	CHECK(encoded.width == 2);
	CHECK(encoded.height == 1);
	std::vector<uint8_t> expected;
	append_rgb(expected, 0xF800U);
	append_rgb(expected, 0x07E0U);
	CHECK(bmp_pixels_top_down(encoded) == expected);
}

TEST_CASE("BMP encoding produces physical panel screenshot dimensions")
{
	const CapturedFrame frame {
		.width = 240,
		.height = 284,
		.pixels = std::vector<uint16_t>(240U * 284U, 0),
	};

	const auto encoded = encode_bmp(frame, ViewRotation::Clockwise90);

	REQUIRE(encoded.ok);
	CHECK(encoded.width == 284);
	CHECK(encoded.height == 240);
	REQUIRE(encoded.bytes.size() == 54U + 284U * 240U * 3U);
	CHECK(bmp_pixels_top_down(encoded).size() == 284U * 240U * 3U);
}

TEST_CASE("BMP encoding rejects invalid dimensions and pixel storage")
{
	const auto empty = encode_bmp({}, ViewRotation::None);
	CHECK_FALSE(empty.ok);
	CHECK_FALSE(empty.error.empty());

	const CapturedFrame oversized_dimensions {
		.width = std::numeric_limits<uint32_t>::max(),
		.height = 1,
	};
	const auto oversized =
		encode_bmp(oversized_dimensions, ViewRotation::None);
	CHECK_FALSE(oversized.ok);
	CHECK(oversized.error.find("signed 32-bit") != std::string::npos);

	const CapturedFrame invalid_pixels {
		.width = 2,
		.height = 2,
		.pixels = { 0x0000U, 0xFFFFU, 0xF800U },
	};
	const auto invalid = encode_bmp(invalid_pixels, ViewRotation::None);
	CHECK_FALSE(invalid.ok);
	CHECK_FALSE(invalid.error.empty());
}

TEST_CASE("framebuffer geometry rejects unsupported layouts and bounds")
{
	const std::vector<uint8_t> mapping(32, 0);
	FramebufferGeometry geometry {
		.width = 2,
		.height = 2,
		.bits_per_pixel = 32,
		.line_length = 8,
		.xoffset = 0,
		.yoffset = 0,
		.red = kRed,
		.green = kGreen,
		.blue = kBlue,
	};
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.bits_per_pixel = 16;
	geometry.green = ChannelLayout { 6, 5 };
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.green = kGreen;
	geometry.line_length = 2;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.line_length = 8;
	geometry.yoffset = 4;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
}

TEST_CASE("framebuffer metadata requires canonical packed true-color RGB565")
{
	const std::vector<uint8_t> mapping(8, 0);
	FramebufferGeometry geometry {
		.width = 2,
		.height = 2,
		.virtual_width = 2,
		.virtual_height = 2,
		.bits_per_pixel = 16,
		.line_length = 4,
		.xoffset = 0,
		.yoffset = 0,
		.red = kRed,
		.green = kGreen,
		.blue = kBlue,
		.transparency = ChannelLayout { 0, 0, 0 },
		.nonstandard = 0,
		.grayscale = 0,
		.framebuffer_type = FB_TYPE_PACKED_PIXELS,
		.visual = FB_VISUAL_TRUECOLOR,
	};

	REQUIRE(copy_visible_rgb565(mapping, geometry).ok);

	geometry.red.msb_right = 1;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.red.msb_right = 0;
	geometry.transparency.length = 1;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.transparency.length = 0;
	geometry.nonstandard = 1;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.nonstandard = 0;
	geometry.framebuffer_type = FB_TYPE_PLANES;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.framebuffer_type = FB_TYPE_PACKED_PIXELS;
	geometry.visual = FB_VISUAL_PSEUDOCOLOR;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
	geometry.visual = FB_VISUAL_TRUECOLOR;
	geometry.virtual_width = 1;
	CHECK_FALSE(copy_visible_rgb565(mapping, geometry).ok);
}

TEST_CASE("framebuffer panel geometry is exactly 240 by 284 before rotation")
{
	FramebufferGeometry geometry { .width = 240, .height = 284 };
	CHECK(has_expected_panel_geometry(geometry));
	geometry.width = 284;
	geometry.height = 240;
	CHECK_FALSE(has_expected_panel_geometry(geometry));
}

TEST_CASE("snapshot writer preserves files and adds collision suffix")
{
	TempDir temp;
	const CapturedFrame frame {
		.width = 1,
		.height = 1,
		.pixels = { 0xF800U },
	};
	const auto captured_at = std::chrono::system_clock::time_point {
		std::chrono::seconds { 1'700'000'000 } +
		std::chrono::microseconds { 123456 }
	};

	const auto first = save_bmp_snapshot(frame, ViewRotation::Clockwise90,
					     temp.path(), captured_at);
	REQUIRE(first.ok);
	std::error_code permissions_error;
	const auto permissions =
		std::filesystem::status(first.path, permissions_error).permissions();
	REQUIRE_FALSE(permissions_error);
	CHECK((permissions & std::filesystem::perms::owner_read) !=
	      std::filesystem::perms::none);
	CHECK((permissions & std::filesystem::perms::owner_write) !=
	      std::filesystem::perms::none);
	CHECK((permissions & std::filesystem::perms::group_all) ==
	      std::filesystem::perms::none);
	CHECK((permissions & std::filesystem::perms::others_all) ==
	      std::filesystem::perms::none);
	const auto first_bytes = read_binary(first.path);

	const auto second = save_bmp_snapshot(frame, ViewRotation::Clockwise90,
					      temp.path(), captured_at);
	REQUIRE(second.ok);
	CHECK(second.path != first.path);
	CHECK(second.path.stem().string() ==
	      first.path.stem().string() + "_01");
	CHECK(read_binary(first.path) == first_bytes);
	CHECK(count_temporary_files(temp.path()) == 0);
}

TEST_CASE("snapshot publication never replaces an existing destination")
{
	TempDir temp;
	const auto temporary_path = temp.path() / ".snapshot.tmp";
	const auto final_path = temp.path() / "snapshot.bmp";
	const std::array<uint8_t, 3> temporary_bytes { 1, 2, 3 };
	const std::array<uint8_t, 3> final_bytes { 4, 5, 6 };
	write_binary(temporary_path, temporary_bytes);
	write_binary(final_path, final_bytes);

	const std::error_code error =
		detail::rename_no_replace(temporary_path, final_path);

	CHECK(error == std::errc::file_exists);
	CHECK(read_binary(temporary_path) ==
	      std::vector<uint8_t>(temporary_bytes.begin(),
				   temporary_bytes.end()));
	CHECK(read_binary(final_path) ==
	      std::vector<uint8_t>(final_bytes.begin(), final_bytes.end()));
}

TEST_CASE("snapshot filenames include local time microseconds and suffixes")
{
	const auto captured_at = std::chrono::system_clock::time_point {
		std::chrono::seconds { 1'700'000'000 } +
		std::chrono::microseconds { 123456 }
	};
	const std::string first = format_snapshot_filename(captured_at);
	const std::string second = format_snapshot_filename(captured_at, 1);

	REQUIRE(first.size() == 35);
	CHECK(first.starts_with("snapshot_"));
	CHECK(first.ends_with(".bmp"));
	CHECK(first[17] == '_');
	CHECK(first[24] == '_');
	CHECK(first.substr(25, 6) == "123456");
	CHECK(second == first.substr(0, first.size() - 4U) + "_01.bmp");
	CHECK(format_snapshot_filename(
		      captured_at + std::chrono::microseconds { 1 }) != first);
}

TEST_CASE("snapshot writer reports an unusable output directory")
{
	TempDir temp;
	const auto not_a_directory = temp.path() / "blocked";
	write_binary(not_a_directory, std::array<uint8_t, 1> { 0x01U });
	const CapturedFrame frame {
		.width = 1,
		.height = 1,
		.pixels = { 0xFFFFU },
	};

	const auto result = save_bmp_snapshot(frame, ViewRotation::Clockwise90,
					      not_a_directory,
					      std::chrono::system_clock::now());

	CHECK_FALSE(result.ok);
	CHECK(count_temporary_files(temp.path()) == 0);
}

TEST_CASE("RGB565 copy rejects empty overflowed and truncated geometry")
{
	const std::vector<uint8_t> mapping(32, 0);
	FramebufferGeometry geometry {
		.width = 0,
		.height = 2,
		.bits_per_pixel = 16,
		.line_length = 8,
		.xoffset = 0,
		.yoffset = 0,
		.red = kRed,
		.green = kGreen,
		.blue = kBlue,
	};
	auto result = copy_visible_rgb565(mapping, geometry);
	CHECK_FALSE(result.ok);
	CHECK_FALSE(result.error.empty());

	geometry.width = 2;
	geometry.xoffset = std::numeric_limits<uint32_t>::max();
	result = copy_visible_rgb565(mapping, geometry);
	CHECK_FALSE(result.ok);
	CHECK_FALSE(result.error.empty());

	geometry.xoffset = 0;
	geometry.width = 3;
	geometry.height = 2;
	geometry.line_length = 6;
	result = copy_visible_rgb565(
		std::span<const uint8_t>(mapping.data(), 11), geometry);
	CHECK_FALSE(result.ok);
	CHECK_FALSE(result.error.empty());
}

TEST_CASE("framebuffer capture reports a missing device without throwing")
{
	TempDir temp;
	const auto device = temp.path() / "missing-fb";

	const auto result = capture_framebuffer(device);

	CHECK_FALSE(result.ok);
	CHECK_FALSE(result.error.empty());
	CHECK(result.error.find(device.string()) != std::string::npos);
}
