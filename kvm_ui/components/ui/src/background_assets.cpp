#include "ui/background_assets.hpp"

#include <utility>

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include <algorithm>
#include <cerrno>
#include <cctype>
#include <cmath>
#include <cstddef>
#include <cstring>
#include <exception>
#include <limits>
#include <optional>
#include <sstream>
#include <string>

#include <fcntl.h>
#include <spdlog/spdlog.h>
#include <sys/stat.h>
#include <toml++/toml.hpp>
#include <unistd.h>

#define STB_IMAGE_IMPLEMENTATION
#define STBI_ONLY_BMP
#include <stb_image.h>
#endif

namespace ui
{

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
namespace
{

constexpr const char *kImageKey = "image";
constexpr int kDesiredChannels = 4;
constexpr std::size_t kRgbaChannels = 4;
constexpr std::size_t kMaxSourcePixels = 4096U * 4096U;
constexpr mode_t kBackgroundConfigFileMode = S_IRUSR | S_IWUSR | S_IRGRP |
					     S_IROTH;

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

	int release()
	{
		const int fd = _fd;
		_fd = -1;
		return fd;
	}

    private:
	int _fd { -1 };
};

struct StbiImageDeleter {
	void operator()(stbi_uc *data) const
	{
		stbi_image_free(data);
	}
};

using StbiImagePtr = std::unique_ptr<stbi_uc, StbiImageDeleter>;

std::optional<std::filesystem::path>
read_image_path(const toml::table &config, const char *section_name,
		const std::filesystem::path &config_path)
{
	const toml::table *section = config[section_name].as_table();
	if (section == nullptr) {
		spdlog::warn("Background config missing [{}] section",
			     section_name);
		return std::nullopt;
	}

	const auto image_value = (*section)[kImageKey].value<std::string>();
	if (!image_value.has_value() || image_value->empty()) {
		spdlog::warn(
			"Background config [{}].{} must be a non-empty string",
			section_name, kImageKey);
		return std::nullopt;
	}

	std::filesystem::path image_path(*image_value);
	if (image_path.is_relative()) {
		image_path = config_path.parent_path() / image_path;
	}

	std::error_code ec;
	if (!std::filesystem::is_regular_file(image_path, ec)) {
		spdlog::warn("Background image '{}' is not a regular file",
			     image_path.string());
		return std::nullopt;
	}

	return image_path;
}

bool source_dimensions_are_safe(int width, int height)
{
	if (width <= 0 || height <= 0) {
		return false;
	}

	const auto w = static_cast<std::size_t>(width);
	const auto h = static_cast<std::size_t>(height);
	if (w > std::numeric_limits<std::size_t>::max() / h) {
		return false;
	}

	const std::size_t pixels = w * h;
	if (pixels > kMaxSourcePixels) {
		return false;
	}

	return pixels <=
	       std::numeric_limits<std::size_t>::max() / kRgbaChannels;
}

uint16_t rgb888_to_rgb565(uint8_t red, uint8_t green, uint8_t blue)
{
	return static_cast<uint16_t>(((red & 0xF8U) << 8U) |
				     ((green & 0xFCU) << 3U) |
				     ((blue & 0xF8U) >> 3U));
}

const char *stbi_error()
{
	const char *reason = stbi_failure_reason();
	return reason == nullptr ? "unknown error" : reason;
}

bool has_bmp_extension(std::string_view name)
{
	constexpr std::string_view suffix = ".bmp";
	if (name.size() <= suffix.size()) {
		return false;
	}
	const std::string_view actual =
		name.substr(name.size() - suffix.size());
	for (std::size_t i = 0; i < suffix.size(); ++i) {
		const auto ch = static_cast<unsigned char>(actual[i]);
		if (static_cast<char>(std::tolower(ch)) != suffix[i]) {
			return false;
		}
	}
	return true;
}

bool write_all(int fd, std::string_view data, std::string &error)
{
	const char *ptr = data.data();
	std::size_t remaining = data.size();
	while (remaining > 0U) {
		const ssize_t written = ::write(fd, ptr, remaining);
		if (written < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (written == 0) {
			error = "short write";
			return false;
		}
		ptr += written;
		remaining -= static_cast<std::size_t>(written);
	}
	return true;
}

std::filesystem::path tmp_path_for(const std::filesystem::path &path,
				   int attempt)
{
	auto tmp_path = path;
	tmp_path += ".tmp." + std::to_string(::getpid()) + "." +
		    std::to_string(attempt);
	return tmp_path;
}

bool sync_parent_directory(const std::filesystem::path &path,
			   std::string &error)
{
	const auto parent = path.parent_path();
	if (parent.empty()) {
		return true;
	}

	const int fd =
		::open(parent.c_str(), O_RDONLY | O_DIRECTORY | O_CLOEXEC);
	if (fd < 0) {
		error = "failed to open background config directory '" +
			parent.string() + "': " + std::strerror(errno);
		return false;
	}
	ScopedFd scoped_fd(fd);
	if (::fsync(scoped_fd.get()) != 0) {
		error = "failed to sync background config directory '" +
			parent.string() + "': " + std::strerror(errno);
		return false;
	}
	return true;
}

bool write_background_table(const std::filesystem::path &path,
			    const toml::table &table, std::string &error)
{
	std::error_code ec;
	const auto parent = path.parent_path();
	if (!parent.empty()) {
		std::filesystem::create_directories(parent, ec);
		if (ec) {
			error = "failed to create background config directory '" +
				parent.string() + "': " + ec.message();
			return false;
		}
	}

	std::ostringstream payload;
	payload << table << '\n';
	const std::string data = payload.str();

	for (int attempt = 0; attempt < 100; ++attempt) {
		const auto tmp_path = tmp_path_for(path, attempt);
		const int fd = ::open(tmp_path.c_str(),
				      O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC |
					      O_NOFOLLOW,
				      kBackgroundConfigFileMode);
		if (fd < 0) {
			if (errno == EEXIST) {
				continue;
			}
			error = "failed to create temporary background config '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			return false;
		}

		ScopedFd scoped_fd(fd);
		std::string write_error;
		if (!write_all(scoped_fd.get(), data, write_error)) {
			error = "failed to write temporary background config '" +
				tmp_path.string() + "': " + write_error;
			std::filesystem::remove(tmp_path, ec);
			return false;
		}
		if (::fsync(scoped_fd.get()) != 0) {
			error = "failed to sync temporary background config '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			std::filesystem::remove(tmp_path, ec);
			return false;
		}
		if (::close(scoped_fd.release()) != 0) {
			error = "failed to close temporary background config '" +
				tmp_path.string() +
				"': " + std::strerror(errno);
			std::filesystem::remove(tmp_path, ec);
			return false;
		}

		std::filesystem::rename(tmp_path, path, ec);
		if (ec) {
			error = "failed to replace background config '" +
				path.string() + "': " + ec.message();
			std::filesystem::remove(tmp_path, ec);
			return false;
		}
		return sync_parent_directory(path, error);
	}

	error = "failed to choose a temporary background config path for '" +
		path.string() + "'";
	return false;
}

int sample_cover_coordinate(int dst_pos, int src_size, int dst_size,
			    double scale)
{
	const double scaled_size = static_cast<double>(src_size) * scale;
	const double crop_offset =
		(scaled_size - static_cast<double>(dst_size)) / 2.0;
	const double src_pos =
		(static_cast<double>(dst_pos) + 0.5 + crop_offset) / scale;
	const int sampled = static_cast<int>(std::floor(src_pos));
	return std::clamp(sampled, 0, src_size - 1);
}

std::vector<uint16_t> scale_cover_rgba_to_rgb565(const stbi_uc *rgba,
						 int src_width, int src_height,
						 uint16_t target_width,
						 uint16_t target_height)
{
	const auto target_pixels = static_cast<std::size_t>(target_width) *
				   static_cast<std::size_t>(target_height);
	std::vector<uint16_t> pixels(target_pixels);

	const double scale_x = static_cast<double>(target_width) /
			       static_cast<double>(src_width);
	const double scale_y = static_cast<double>(target_height) /
			       static_cast<double>(src_height);
	const double scale = std::max(scale_x, scale_y);

	for (uint16_t y = 0; y < target_height; ++y) {
		const int src_y = sample_cover_coordinate(y, src_height,
							  target_height, scale);
		for (uint16_t x = 0; x < target_width; ++x) {
			const int src_x = sample_cover_coordinate(
				x, src_width, target_width, scale);
			const std::size_t src_index =
				(static_cast<std::size_t>(src_y) *
					 static_cast<std::size_t>(src_width) +
				 static_cast<std::size_t>(src_x)) *
				kRgbaChannels;
			pixels[static_cast<std::size_t>(y) * target_width + x] =
				rgb888_to_rgb565(rgba[src_index],
						 rgba[src_index + 1U],
						 rgba[src_index + 2U]);
		}
	}

	return pixels;
}

std::unique_ptr<BackgroundImage>
load_background_image(const std::filesystem::path &path, uint16_t target_width,
		      uint16_t target_height)
{
	int info_width = 0;
	int info_height = 0;
	int info_channels = 0;
	if (stbi_info(path.string().c_str(), &info_width, &info_height,
		      &info_channels) == 0) {
		spdlog::warn("Failed to inspect BMP background '{}': {}",
			     path.string(), stbi_error());
		return nullptr;
	}

	if (!source_dimensions_are_safe(info_width, info_height)) {
		spdlog::warn(
			"BMP background '{}' has unsupported dimensions {}x{}",
			path.string(), info_width, info_height);
		return nullptr;
	}

	int src_width = 0;
	int src_height = 0;
	int src_channels = 0;
	StbiImagePtr image(stbi_load(path.string().c_str(), &src_width,
				     &src_height, &src_channels,
				     kDesiredChannels));
	if (!image) {
		spdlog::warn("Failed to decode BMP background '{}': {}",
			     path.string(), stbi_error());
		return nullptr;
	}

	if (!source_dimensions_are_safe(src_width, src_height)) {
		spdlog::warn(
			"Decoded BMP background '{}' has unsupported dimensions {}x{}",
			path.string(), src_width, src_height);
		return nullptr;
	}

	auto pixels = scale_cover_rgba_to_rgb565(image.get(), src_width,
						 src_height, target_width,
						 target_height);
	return std::make_unique<BackgroundImage>(target_width, target_height,
						 std::move(pixels));
}

void load_configured_image(const toml::table &config, const char *section_name,
			   const std::filesystem::path &config_path,
			   uint16_t target_width, uint16_t target_height,
			   std::unique_ptr<BackgroundImage> &output)
{
	const auto image_path =
		read_image_path(config, section_name, config_path);
	if (!image_path.has_value()) {
		return;
	}

	try {
		output = load_background_image(*image_path, target_width,
					       target_height);
	} catch (const std::exception &error) {
		spdlog::warn("Failed to load [{}] BMP background '{}': {}",
			     section_name, image_path->string(), error.what());
	}
}

} // namespace
#endif

BackgroundImage::BackgroundImage(uint16_t width, uint16_t height,
				 std::vector<uint16_t> pixels)
	: _width(width)
	, _height(height)
	, _pixels(std::move(pixels))
{
	const auto expected_pixels = static_cast<std::size_t>(_width) *
				     static_cast<std::size_t>(_height);
	if (_width == 0 || _height == 0 || _pixels.size() != expected_pixels) {
		_width = 0;
		_height = 0;
		_pixels.clear();
		return;
	}

	_descriptor.header.magic = LV_IMAGE_HEADER_MAGIC;
	_descriptor.header.cf = LV_COLOR_FORMAT_RGB565;
	_descriptor.header.flags = 0;
	_descriptor.header.w = _width;
	_descriptor.header.h = _height;
	_descriptor.header.stride =
		static_cast<uint16_t>(_width * sizeof(uint16_t));
	_descriptor.header.reserved_2 = 0;
	_descriptor.data_size =
		static_cast<uint32_t>(_pixels.size() * sizeof(uint16_t));
	_descriptor.data = reinterpret_cast<const uint8_t *>(_pixels.data());
	_descriptor.reserved = nullptr;
}

const lv_image_dsc_t *BackgroundImage::descriptor() const
{
	return _pixels.empty() ? nullptr : &_descriptor;
}

uint16_t BackgroundImage::rgb565_pixel(uint16_t x, uint16_t y) const
{
	if (x >= _width || y >= _height || _pixels.empty()) {
		return 0;
	}
	return _pixels[static_cast<std::size_t>(y) * _width + x];
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
BackgroundAssets
load_background_assets(const std::filesystem::path &config_path,
		       uint16_t target_width, uint16_t target_height)
{
	BackgroundAssets assets;
	if (target_width == 0 || target_height == 0) {
		spdlog::warn("Background target dimensions must be non-zero");
		return assets;
	}

	std::error_code ec;
	if (!std::filesystem::is_regular_file(config_path, ec)) {
		spdlog::warn("Background config '{}' is not available",
			     config_path.string());
		return assets;
	}

	try {
		const toml::table config =
			toml::parse_file(config_path.string());
		load_configured_image(config, "main", config_path, target_width,
				      target_height, assets.main);
		load_configured_image(config, "settings", config_path,
				      target_width, target_height,
				      assets.settings);
	} catch (const toml::parse_error &error) {
		spdlog::warn("Failed to parse background config '{}': {}",
			     config_path.string(), error.description());
	} catch (const std::exception &error) {
		spdlog::warn("Failed to load background config '{}': {}",
			     config_path.string(), error.what());
	}

	return assets;
}

BackgroundAssets load_default_background_assets(uint16_t target_width,
						uint16_t target_height)
{
	return load_background_assets(kDefaultBackgroundConfigPath,
				      target_width, target_height);
}

bool validate_background_image_name(std::string_view image_name,
				    std::string &error)
{
	error.clear();
	if (image_name.empty()) {
		error = "image_name must be non-empty";
		return false;
	}
	if (image_name.size() > kMaxBackgroundImageNameBytes) {
		error = "image_name is too long";
		return false;
	}
	if (image_name == "." || image_name == "..") {
		error = "image_name must be a file name";
		return false;
	}
	if (!has_bmp_extension(image_name)) {
		error = "image_name must end with .bmp";
		return false;
	}

	for (const char ch : image_name) {
		const auto byte = static_cast<unsigned char>(ch);
		if (!(std::isalnum(byte) != 0 || ch == '_' || ch == '-' ||
		      ch == '.')) {
			error = "image_name may only contain A-Z, a-z, 0-9, '.', '_' and '-'";
			return false;
		}
	}
	return true;
}

bool load_background_assets_by_name(std::string_view image_name,
				    uint16_t target_width,
				    uint16_t target_height,
				    BackgroundAssets &assets,
				    std::string &error,
				    const std::filesystem::path &directory)
{
	error.clear();
	std::string validation_error;
	if (!validate_background_image_name(image_name, validation_error)) {
		error = validation_error;
		return false;
	}
	if (target_width == 0 || target_height == 0) {
		error = "background target dimensions must be non-zero";
		return false;
	}

	const auto image_path = directory / std::string(image_name);
	std::error_code ec;
	const auto status = std::filesystem::symlink_status(image_path, ec);
	if (ec) {
		error = "failed to inspect background image '" +
			image_path.string() + "': " + ec.message();
		return false;
	}
	if (!std::filesystem::is_regular_file(status)) {
		error = "background image is not a regular file: " +
			image_path.string();
		return false;
	}

	BackgroundAssets next_assets;
	next_assets.main =
		load_background_image(image_path, target_width, target_height);
	next_assets.settings =
		load_background_image(image_path, target_width, target_height);
	if (!next_assets.main || !next_assets.settings) {
		error = "failed to load BMP background: " + image_path.string();
		return false;
	}

	assets = std::move(next_assets);
	return true;
}

bool save_background_selection(std::string_view image_name, std::string &error,
			       const std::filesystem::path &config_path)
{
	error.clear();
	std::string validation_error;
	if (!validate_background_image_name(image_name, validation_error)) {
		error = validation_error;
		return false;
	}

	const std::string image(image_name);
	toml::table table {
		{ "main", toml::table { { kImageKey, image } } },
		{ "settings", toml::table { { kImageKey, image } } },
	};
	return write_background_table(config_path, table, error);
}
#endif

lv_obj_t *create_background_image_obj(lv_obj_t *parent,
				      const BackgroundImage *image)
{
	if (parent == nullptr || image == nullptr ||
	    image->descriptor() == nullptr) {
		return nullptr;
	}

	lv_obj_t *image_obj = lv_image_create(parent);
	lv_image_set_src(image_obj, image->descriptor());
	lv_obj_set_pos(image_obj, 0, 0);
	lv_obj_set_size(image_obj, image->width(), image->height());
	lv_obj_clear_flag(image_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(image_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_move_to_index(image_obj, 0);
	return image_obj;
}

} // namespace ui
