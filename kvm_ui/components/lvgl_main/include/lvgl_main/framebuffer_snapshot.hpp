#pragma once

#include <array>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <span>
#include <string>
#include <system_error>
#include <vector>

#include <linux/fb.h>

namespace lvgl_main::framebuffer_snapshot
{

enum class ViewRotation : uint8_t {
	None,
	Clockwise90,
	Clockwise270,
};

struct ChannelLayout {
	uint32_t offset = 0;
	uint32_t length = 0;
	uint32_t msb_right = 0;

	bool operator==(const ChannelLayout &) const = default;
};

struct FramebufferGeometry {
	uint32_t width = 0;
	uint32_t height = 0;
	uint32_t virtual_width = 0;
	uint32_t virtual_height = 0;
	uint32_t bits_per_pixel = 0;
	uint32_t line_length = 0;
	uint32_t xoffset = 0;
	uint32_t yoffset = 0;
	ChannelLayout red;
	ChannelLayout green;
	ChannelLayout blue;
	ChannelLayout transparency;
	uint32_t nonstandard = 0;
	uint32_t grayscale = 0;
	uint32_t framebuffer_type = FB_TYPE_PACKED_PIXELS;
	uint32_t visual = FB_VISUAL_TRUECOLOR;
};

struct CapturedFrame {
	uint32_t width = 0;
	uint32_t height = 0;
	std::vector<uint16_t> pixels;
};

struct CaptureResult {
	bool ok = false;
	CapturedFrame frame;
	std::string error;
};

struct EncodeResult {
	bool ok = false;
	uint32_t width = 0;
	uint32_t height = 0;
	std::vector<uint8_t> bytes;
	std::string error;
};

struct SaveResult {
	bool ok = false;
	std::filesystem::path path;
	std::string error;
};

namespace detail
{

std::error_code rename_no_replace(const std::filesystem::path &source,
				  const std::filesystem::path &destination);

} // namespace detail

std::array<uint8_t, 3> rgb565_to_rgb888(uint16_t pixel);
bool has_expected_panel_geometry(const FramebufferGeometry &geometry);
CaptureResult copy_visible_rgb565(std::span<const uint8_t> mapping,
				  const FramebufferGeometry &geometry);
CaptureResult capture_framebuffer(const std::filesystem::path &device);
EncodeResult encode_bmp(const CapturedFrame &frame, ViewRotation rotation);
std::string
format_snapshot_filename(std::chrono::system_clock::time_point captured_at,
			 uint32_t collision_index = 0);
SaveResult save_bmp_snapshot(const CapturedFrame &frame, ViewRotation rotation,
			     const std::filesystem::path &output_directory,
			     std::chrono::system_clock::time_point captured_at);

} // namespace lvgl_main::framebuffer_snapshot
