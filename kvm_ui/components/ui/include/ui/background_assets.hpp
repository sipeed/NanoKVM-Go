#pragma once

#include <cstdint>
#include <cstddef>
#include <memory>
#include <vector>

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include <filesystem>
#include <string>
#include <string_view>
#endif

#include <lvgl.h>

namespace ui
{

class BackgroundImage {
    public:
	BackgroundImage(uint16_t width, uint16_t height,
			std::vector<uint16_t> pixels);

	BackgroundImage(const BackgroundImage &) = delete;
	BackgroundImage &operator=(const BackgroundImage &) = delete;
	BackgroundImage(BackgroundImage &&) = delete;
	BackgroundImage &operator=(BackgroundImage &&) = delete;

	const lv_image_dsc_t *descriptor() const;
	uint16_t width() const
	{
		return _width;
	}
	uint16_t height() const
	{
		return _height;
	}
	uint16_t rgb565_pixel(uint16_t x, uint16_t y) const;

    private:
	uint16_t _width { 0 };
	uint16_t _height { 0 };
	std::vector<uint16_t> _pixels;
	lv_image_dsc_t _descriptor {};
};

struct BackgroundAssets {
	std::unique_ptr<BackgroundImage> main;
	std::unique_ptr<BackgroundImage> settings;
};

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
constexpr const char *kDefaultBackgroundConfigPath = "/kvmcomm/ui/bg/bg.toml";
constexpr const char *kDefaultBackgroundDirectory = "/kvmcomm/ui/bg";
constexpr std::size_t kMaxBackgroundImageNameBytes = 128;
constexpr uint16_t kDefaultBackgroundTargetWidth = 270;
constexpr uint16_t kDefaultBackgroundTargetHeight = 170;

BackgroundAssets
load_background_assets(const std::filesystem::path &config_path,
		       uint16_t target_width, uint16_t target_height);

BackgroundAssets load_default_background_assets(uint16_t target_width,
						uint16_t target_height);

bool validate_background_image_name(std::string_view image_name,
				    std::string &error);

bool load_background_assets_by_name(
	std::string_view image_name, uint16_t target_width,
	uint16_t target_height, BackgroundAssets &assets, std::string &error,
	const std::filesystem::path &directory =
		std::filesystem::path(kDefaultBackgroundDirectory));

bool save_background_selection(
	std::string_view image_name, std::string &error,
	const std::filesystem::path &config_path =
		std::filesystem::path(kDefaultBackgroundConfigPath));
#endif

lv_obj_t *create_background_image_obj(lv_obj_t *parent,
				      const BackgroundImage *image);

} // namespace ui
