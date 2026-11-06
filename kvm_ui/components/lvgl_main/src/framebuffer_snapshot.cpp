#include "lvgl_main/framebuffer_snapshot.hpp"

#include <cerrno>
#include <chrono>
#include <cstring>
#include <ctime>
#include <iomanip>
#include <limits>
#include <sstream>
#include <string>
#include <system_error>
#include <utility>

#include <fcntl.h>
#include <linux/fb.h>
#include <linux/fs.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <sys/syscall.h>
#include <unistd.h>

namespace lvgl_main::framebuffer_snapshot
{
namespace
{

constexpr ChannelLayout kRgb565Red { 11, 5, 0 };
constexpr ChannelLayout kRgb565Green { 5, 6, 0 };
constexpr ChannelLayout kRgb565Blue { 0, 5, 0 };
constexpr ChannelLayout kNoTransparency { 0, 0, 0 };
constexpr uint32_t kExpectedFramebufferWidth = 240;
constexpr uint32_t kExpectedFramebufferHeight = 284;

class ScopedFd final {
    public:
	explicit ScopedFd(int fd)
		: _fd(fd)
	{
	}

	~ScopedFd()
	{
		if (_fd >= 0) {
			::close(_fd);
		}
	}

	ScopedFd(const ScopedFd &) = delete;
	ScopedFd &operator=(const ScopedFd &) = delete;

	int get() const
	{
		return _fd;
	}

    private:
	int _fd;
};

class ScopedMapping final {
    public:
	ScopedMapping(void *address, std::size_t size)
		: _address(address)
		, _size(size)
	{
	}

	~ScopedMapping()
	{
		if (_address != nullptr && _address != MAP_FAILED) {
			::munmap(_address, _size);
		}
	}

	ScopedMapping(const ScopedMapping &) = delete;
	ScopedMapping &operator=(const ScopedMapping &) = delete;

	const uint8_t *data() const
	{
		return static_cast<const uint8_t *>(_address);
	}

    private:
	void *_address;
	std::size_t _size;
};

bool checked_add(std::size_t left, std::size_t right, std::size_t &result)
{
	if (right > std::numeric_limits<std::size_t>::max() - left) {
		return false;
	}
	result = left + right;
	return true;
}

bool checked_multiply(std::size_t left, std::size_t right, std::size_t &result)
{
	if (left != 0 &&
	    right > std::numeric_limits<std::size_t>::max() / left) {
		return false;
	}
	result = left * right;
	return true;
}

CaptureResult capture_error(std::string message)
{
	return CaptureResult { .error = std::move(message) };
}

EncodeResult encode_error(std::string message)
{
	return EncodeResult { .error = std::move(message) };
}

SaveResult save_error(std::string message)
{
	return SaveResult { .error = std::move(message) };
}

bool write_all(int fd, std::span<const uint8_t> bytes, std::string &error)
{
	std::size_t written = 0;
	while (written < bytes.size()) {
		const ssize_t count = ::write(fd, bytes.data() + written,
					      bytes.size() - written);
		if (count < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (count == 0) {
			error = "short write";
			return false;
		}
		written += static_cast<std::size_t>(count);
	}
	return true;
}

std::filesystem::path
make_temporary_path(const std::filesystem::path &directory,
		    std::string_view filename, uint32_t attempt)
{
	return directory / ("." + std::string(filename) + ".tmp." +
			    std::to_string(static_cast<long long>(::getpid())) +
			    "." + std::to_string(attempt));
}

} // namespace

namespace detail
{

std::error_code rename_no_replace(const std::filesystem::path &source,
				  const std::filesystem::path &destination)
{
#if defined(SYS_renameat2) && defined(RENAME_NOREPLACE)
	if (::syscall(SYS_renameat2, AT_FDCWD, source.c_str(), AT_FDCWD,
		      destination.c_str(), RENAME_NOREPLACE) == 0) {
		return {};
	}
	const int rename_error = errno;
	if (rename_error != ENOSYS && rename_error != EINVAL &&
	    rename_error != EOPNOTSUPP) {
		return { rename_error, std::generic_category() };
	}
#endif

	if (::link(source.c_str(), destination.c_str()) != 0) {
		return { errno, std::generic_category() };
	}
	if (::unlink(source.c_str()) == 0) {
		return {};
	}

	const int unlink_error = errno;
	(void)::unlink(destination.c_str());
	return { unlink_error, std::generic_category() };
}

} // namespace detail

std::array<uint8_t, 3> rgb565_to_rgb888(uint16_t pixel)
{
	const uint8_t red = static_cast<uint8_t>((pixel >> 11U) & 0x1FU);
	const uint8_t green = static_cast<uint8_t>((pixel >> 5U) & 0x3FU);
	const uint8_t blue = static_cast<uint8_t>(pixel & 0x1FU);
	return {
		static_cast<uint8_t>((red << 3U) | (red >> 2U)),
		static_cast<uint8_t>((green << 2U) | (green >> 4U)),
		static_cast<uint8_t>((blue << 3U) | (blue >> 2U)),
	};
}

bool has_expected_panel_geometry(const FramebufferGeometry &geometry)
{
	return geometry.width == kExpectedFramebufferWidth &&
	       geometry.height == kExpectedFramebufferHeight;
}

CaptureResult copy_visible_rgb565(std::span<const uint8_t> mapping,
				  const FramebufferGeometry &geometry)
{
	if (geometry.bits_per_pixel != 16 || geometry.red != kRgb565Red ||
	    geometry.green != kRgb565Green || geometry.blue != kRgb565Blue ||
	    geometry.transparency != kNoTransparency) {
		return capture_error("framebuffer is not canonical RGB565");
	}
	if (geometry.nonstandard != 0 || geometry.grayscale != 0 ||
	    geometry.framebuffer_type != FB_TYPE_PACKED_PIXELS ||
	    geometry.visual != FB_VISUAL_TRUECOLOR) {
		return capture_error(
			"framebuffer is not packed true-color RGB565");
	}
	if (geometry.width == 0 || geometry.height == 0 ||
	    geometry.line_length == 0) {
		return capture_error("framebuffer geometry is empty");
	}

	std::size_t row_pixel_end = 0;
	if (!checked_add(geometry.xoffset, geometry.width, row_pixel_end)) {
		return capture_error("framebuffer row width overflows");
	}
	const std::size_t virtual_width = geometry.virtual_width == 0 ?
						  geometry.width :
						  geometry.virtual_width;
	if (row_pixel_end > virtual_width) {
		return capture_error(
			"framebuffer visible row exceeds virtual width");
	}
	std::size_t row_byte_end = 0;
	if (!checked_multiply(row_pixel_end, sizeof(uint16_t), row_byte_end) ||
	    row_byte_end > geometry.line_length) {
		return capture_error("framebuffer row exceeds line length");
	}

	std::size_t last_row = 0;
	if (!checked_add(geometry.yoffset, geometry.height - 1U, last_row)) {
		return capture_error("framebuffer row offset overflows");
	}
	std::size_t visible_row_end = 0;
	const std::size_t virtual_height = geometry.virtual_height == 0 ?
						   geometry.height :
						   geometry.virtual_height;
	if (!checked_add(geometry.yoffset, geometry.height, visible_row_end) ||
	    visible_row_end > virtual_height) {
		return capture_error(
			"framebuffer visible page exceeds virtual height");
	}
	std::size_t last_row_offset = 0;
	if (!checked_multiply(last_row, geometry.line_length,
			      last_row_offset)) {
		return capture_error("framebuffer mapping offset overflows");
	}
	std::size_t required_size = 0;
	if (!checked_add(last_row_offset, row_byte_end, required_size) ||
	    required_size > mapping.size()) {
		return capture_error("framebuffer mapping is truncated");
	}

	std::size_t pixel_count = 0;
	if (!checked_multiply(geometry.width, geometry.height, pixel_count)) {
		return capture_error("framebuffer pixel count overflows");
	}

	CapturedFrame frame {
		.width = geometry.width,
		.height = geometry.height,
		.pixels = std::vector<uint16_t>(pixel_count),
	};
	for (std::size_t y = 0; y < geometry.height; ++y) {
		const std::size_t row_offset =
			(static_cast<std::size_t>(geometry.yoffset) + y) *
			geometry.line_length;
		const std::size_t visible_offset =
			row_offset +
			static_cast<std::size_t>(geometry.xoffset) * 2U;
		for (std::size_t x = 0; x < geometry.width; ++x) {
			const std::size_t byte_offset = visible_offset + x * 2U;
			frame.pixels[y * geometry.width + x] =
				static_cast<uint16_t>(mapping[byte_offset]) |
				static_cast<uint16_t>(mapping[byte_offset + 1U]
						      << 8U);
		}
	}

	return CaptureResult { .ok = true, .frame = std::move(frame) };
}

CaptureResult capture_framebuffer(const std::filesystem::path &device)
{
	const int fd = ::open(device.c_str(), O_RDONLY | O_CLOEXEC);
	if (fd < 0) {
		return capture_error("failed to open framebuffer '" +
				     device.string() +
				     "': " + std::strerror(errno));
	}
	ScopedFd scoped_fd(fd);

	struct fb_fix_screeninfo fixed_info {};
	if (::ioctl(scoped_fd.get(), FBIOGET_FSCREENINFO, &fixed_info) != 0) {
		return capture_error(
			"failed to query fixed framebuffer info for '" +
			device.string() + "': " + std::strerror(errno));
	}
	struct fb_var_screeninfo variable_info {};
	if (::ioctl(scoped_fd.get(), FBIOGET_VSCREENINFO, &variable_info) !=
	    0) {
		return capture_error(
			"failed to query variable framebuffer info for '" +
			device.string() + "': " + std::strerror(errno));
	}
	if (fixed_info.smem_len == 0) {
		return capture_error("framebuffer mapping is empty for '" +
				     device.string() + "'");
	}

	const FramebufferGeometry geometry {
		.width = variable_info.xres,
		.height = variable_info.yres,
		.virtual_width = variable_info.xres_virtual,
		.virtual_height = variable_info.yres_virtual,
		.bits_per_pixel = variable_info.bits_per_pixel,
		.line_length = fixed_info.line_length,
		.xoffset = variable_info.xoffset,
		.yoffset = variable_info.yoffset,
		.red = { variable_info.red.offset, variable_info.red.length,
			 variable_info.red.msb_right },
		.green = { variable_info.green.offset,
			   variable_info.green.length,
			   variable_info.green.msb_right },
		.blue = { variable_info.blue.offset, variable_info.blue.length,
			  variable_info.blue.msb_right },
		.transparency = { variable_info.transp.offset,
				  variable_info.transp.length,
				  variable_info.transp.msb_right },
		.nonstandard = variable_info.nonstd,
		.grayscale = variable_info.grayscale,
		.framebuffer_type = fixed_info.type,
		.visual = fixed_info.visual,
	};
	if (!has_expected_panel_geometry(geometry)) {
		return capture_error("unexpected framebuffer geometry for '" +
				     device.string() +
				     "': expected 240x284, got " +
				     std::to_string(geometry.width) + "x" +
				     std::to_string(geometry.height));
	}

	void *address = ::mmap(nullptr, fixed_info.smem_len, PROT_READ,
			       MAP_SHARED, scoped_fd.get(), 0);
	if (address == MAP_FAILED) {
		return capture_error("failed to map framebuffer '" +
				     device.string() +
				     "': " + std::strerror(errno));
	}
	ScopedMapping mapping(address, fixed_info.smem_len);

	return copy_visible_rgb565(
		std::span<const uint8_t>(mapping.data(), fixed_info.smem_len),
		geometry);
}

namespace
{

void write_le16(std::vector<uint8_t> &bytes, std::size_t offset,
			uint16_t value)
{
	bytes[offset] = static_cast<uint8_t>(value & 0xFFU);
	bytes[offset + 1U] = static_cast<uint8_t>(value >> 8U);
}

void write_le32(std::vector<uint8_t> &bytes, std::size_t offset,
			uint32_t value)
{
	bytes[offset] = static_cast<uint8_t>(value & 0xFFU);
	bytes[offset + 1U] = static_cast<uint8_t>((value >> 8U) & 0xFFU);
	bytes[offset + 2U] = static_cast<uint8_t>((value >> 16U) & 0xFFU);
	bytes[offset + 3U] = static_cast<uint8_t>((value >> 24U) & 0xFFU);
}

} // namespace

EncodeResult encode_bmp(const CapturedFrame &frame, ViewRotation rotation)
{
	if (frame.width == 0 || frame.height == 0) {
		return encode_error("captured frame is empty");
	}
	const bool rotated = rotation != ViewRotation::None;
	const uint32_t output_width = rotated ? frame.height : frame.width;
	const uint32_t output_height = rotated ? frame.width : frame.height;
	if (output_width > static_cast<uint32_t>(std::numeric_limits<int32_t>::max()) ||
	    output_height >
		static_cast<uint32_t>(std::numeric_limits<int32_t>::max())) {
		return encode_error("BMP dimensions exceed signed 32-bit range");
	}

	std::size_t pixel_count = 0;
	if (!checked_multiply(frame.width, frame.height, pixel_count) ||
	    pixel_count != frame.pixels.size()) {
		return encode_error("captured frame pixel count is invalid");
	}

	std::size_t row_bytes = 0;
	if (!checked_multiply(static_cast<std::size_t>(output_width), 3U,
				      row_bytes)) {
		return encode_error("BMP row size overflows");
	}
	std::size_t padded_row_bytes = 0;
	std::size_t row_with_padding = 0;
	if (!checked_add(row_bytes, 3U, row_with_padding)) {
		return encode_error("BMP row padding overflows");
	}
	padded_row_bytes = row_with_padding & ~static_cast<std::size_t>(3U);

	std::size_t pixel_bytes = 0;
	std::size_t output_size = 0;
	if (!checked_multiply(padded_row_bytes,
				      static_cast<std::size_t>(output_height),
				      pixel_bytes) ||
	    !checked_add(54U, pixel_bytes, output_size) ||
	    output_size > std::numeric_limits<uint32_t>::max() ||
	    pixel_bytes > std::numeric_limits<uint32_t>::max()) {
		return encode_error("BMP output size overflows");
	}

	EncodeResult result {
		.ok = true,
		.width = output_width,
		.height = output_height,
	};
	result.bytes.resize(output_size, 0);
	result.bytes[0] = 'B';
	result.bytes[1] = 'M';
	write_le32(result.bytes, 2, static_cast<uint32_t>(output_size));
	write_le32(result.bytes, 10, 54U);
	write_le32(result.bytes, 14, 40U);
	write_le32(result.bytes, 18, output_width);
	write_le32(result.bytes, 22, output_height);
	write_le16(result.bytes, 26, 1U);
	write_le16(result.bytes, 28, 24U);
	write_le32(result.bytes, 30, 0U);
	write_le32(result.bytes, 34, static_cast<uint32_t>(pixel_bytes));

	std::size_t write_offset = 54U;
	for (uint32_t file_y = 0; file_y < output_height; ++file_y) {
		const uint32_t output_y = output_height - 1U - file_y;
		for (uint32_t output_x = 0; output_x < output_width;
		     ++output_x) {
			uint32_t physical_x = 0;
			uint32_t physical_y = 0;
			if (rotation == ViewRotation::None) {
				physical_x = output_x;
				physical_y = output_y;
			} else if (rotation == ViewRotation::Clockwise90) {
				physical_x = frame.width - 1U - output_y;
				physical_y = output_x;
			} else {
				physical_x = output_y;
				physical_y = frame.height - 1U - output_x;
			}
			const auto rgb = rgb565_to_rgb888(
				frame.pixels[physical_y * frame.width +
						     physical_x]);
			result.bytes[write_offset++] = rgb[2];
			result.bytes[write_offset++] = rgb[1];
			result.bytes[write_offset++] = rgb[0];
		}
		write_offset += padded_row_bytes - row_bytes;
	}
	return result;
}

std::string
format_snapshot_filename(std::chrono::system_clock::time_point captured_at,
			 uint32_t collision_index)
{
	const auto whole_seconds =
		std::chrono::floor<std::chrono::seconds>(captured_at);
	const auto micros =
		std::chrono::duration_cast<std::chrono::microseconds>(
			captured_at - whole_seconds)
			.count();
	const std::time_t wall_time =
		std::chrono::system_clock::to_time_t(whole_seconds);
	std::tm local_time {};
	if (::localtime_r(&wall_time, &local_time) == nullptr) {
		return {};
	}

	std::ostringstream name;
	name << "snapshot_" << std::put_time(&local_time, "%Y%m%d_%H%M%S")
	     << '_' << std::setfill('0') << std::setw(6) << micros;
	if (collision_index > 0) {
		name << '_' << std::setw(2) << collision_index;
	}
	name << ".bmp";
	return name.str();
}

SaveResult save_bmp_snapshot(const CapturedFrame &frame, ViewRotation rotation,
				     const std::filesystem::path &output_directory,
				     std::chrono::system_clock::time_point captured_at)
{
	const auto encoded = encode_bmp(frame, rotation);
	if (!encoded.ok) {
		return save_error("failed to encode BMP: " + encoded.error);
	}

	std::error_code ec;
	std::filesystem::create_directories(output_directory, ec);
	if (ec) {
		return save_error("failed to create snapshot directory '" +
				  output_directory.string() +
				  "': " + ec.message());
	}
	if (!std::filesystem::is_directory(output_directory, ec) || ec) {
		return save_error("snapshot output path is not a directory: " +
				  output_directory.string());
	}

	for (uint32_t collision = 0; collision <= 99; ++collision) {
		const std::string filename =
			format_snapshot_filename(captured_at, collision);
		if (filename.empty()) {
			return save_error(
				"failed to format snapshot timestamp");
		}
		const auto final_path = output_directory / filename;

		std::filesystem::path temporary_path;
		int fd = -1;
		for (uint32_t attempt = 0; attempt < 100; ++attempt) {
			temporary_path = make_temporary_path(output_directory,
							     filename, attempt);
			fd = ::open(temporary_path.c_str(),
				    O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC |
					    O_NOFOLLOW,
				    S_IRUSR | S_IWUSR);
			if (fd >= 0) {
				break;
			}
			if (errno != EEXIST) {
				return save_error(
					"failed to create temporary snapshot '" +
					temporary_path.string() +
					"': " + std::strerror(errno));
			}
		}
		if (fd < 0) {
			return save_error(
				"failed to choose a temporary snapshot path");
		}

		std::string write_error;
		const bool wrote = write_all(fd, encoded.bytes, write_error);
		const int close_result = ::close(fd);
		if (!wrote || close_result != 0) {
			const std::string message =
				!wrote ? write_error : std::strerror(errno);
			std::filesystem::remove(temporary_path, ec);
			return save_error(
				"failed to write temporary snapshot '" +
				temporary_path.string() + "': " + message);
		}

		const std::error_code publish_error =
			detail::rename_no_replace(temporary_path, final_path);
		if (publish_error == std::errc::file_exists) {
			std::error_code remove_ec;
			std::filesystem::remove(temporary_path, remove_ec);
			continue;
		}
		if (publish_error) {
			std::error_code remove_ec;
			std::filesystem::remove(temporary_path, remove_ec);
			return save_error("failed to publish snapshot '" +
					  final_path.string() +
					  "': " + publish_error.message());
		}
		return SaveResult { .ok = true, .path = final_path };
	}

	return save_error("snapshot filename collision limit exhausted");
}

} // namespace lvgl_main::framebuffer_snapshot
