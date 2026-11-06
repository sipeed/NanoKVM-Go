#include <chrono>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "ui/background_assets.hpp"

namespace
{

struct Rgb {
	uint8_t r;
	uint8_t g;
	uint8_t b;
};

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_bg_test_" + std::to_string(stamp));
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

void write_u16(std::ofstream &out, uint16_t value)
{
	const char bytes[] = {
		static_cast<char>(value & 0xFFU),
		static_cast<char>((value >> 8U) & 0xFFU),
	};
	out.write(bytes, sizeof(bytes));
}

void write_u32(std::ofstream &out, uint32_t value)
{
	const char bytes[] = {
		static_cast<char>(value & 0xFFU),
		static_cast<char>((value >> 8U) & 0xFFU),
		static_cast<char>((value >> 16U) & 0xFFU),
		static_cast<char>((value >> 24U) & 0xFFU),
	};
	out.write(bytes, sizeof(bytes));
}

void write_i32(std::ofstream &out, int32_t value)
{
	write_u32(out, static_cast<uint32_t>(value));
}

void write_text(const std::filesystem::path &path, const std::string &text)
{
	std::ofstream out(path, std::ios::trunc);
	REQUIRE(out.good());
	out << text;
	REQUIRE(out.good());
}

void write_bmp_24(const std::filesystem::path &path, int32_t width,
		  int32_t height, const std::vector<Rgb> &pixels)
{
	REQUIRE(width > 0);
	REQUIRE(height > 0);
	REQUIRE(pixels.size() == static_cast<std::size_t>(width * height));

	constexpr uint32_t file_header_size = 14;
	constexpr uint32_t dib_header_size = 40;
	constexpr uint32_t pixel_offset = file_header_size + dib_header_size;
	const uint32_t row_stride =
		static_cast<uint32_t>(((width * 3) + 3) & ~3);
	const uint32_t image_size = row_stride * static_cast<uint32_t>(height);
	const uint32_t file_size = pixel_offset + image_size;

	std::ofstream out(path, std::ios::binary | std::ios::trunc);
	REQUIRE(out.good());

	out.put('B');
	out.put('M');
	write_u32(out, file_size);
	write_u16(out, 0);
	write_u16(out, 0);
	write_u32(out, pixel_offset);

	write_u32(out, dib_header_size);
	write_i32(out, width);
	write_i32(out, height);
	write_u16(out, 1);
	write_u16(out, 24);
	write_u32(out, 0);
	write_u32(out, image_size);
	write_i32(out, 2835);
	write_i32(out, 2835);
	write_u32(out, 0);
	write_u32(out, 0);

	const std::vector<char> padding(
		row_stride - static_cast<uint32_t>(width * 3), 0);
	for (int32_t y = height - 1; y >= 0; --y) {
		for (int32_t x = 0; x < width; ++x) {
			const Rgb pixel =
				pixels[static_cast<std::size_t>(y * width + x)];
			out.put(static_cast<char>(pixel.b));
			out.put(static_cast<char>(pixel.g));
			out.put(static_cast<char>(pixel.r));
		}
		out.write(padding.data(),
			  static_cast<std::streamsize>(padding.size()));
	}

	REQUIRE(out.good());
}

uint16_t rgb565(uint8_t red, uint8_t green, uint8_t blue)
{
	return static_cast<uint16_t>(((red & 0xF8U) << 8U) |
				     ((green & 0xFCU) << 3U) |
				     ((blue & 0xF8U) >> 3U));
}

} // namespace

TEST_CASE("background assets tolerate missing config")
{
	TempDir temp;

	const auto assets =
		ui::load_background_assets(temp.path() / "bg.toml", 4, 4);

	CHECK_FALSE(assets.main);
	CHECK_FALSE(assets.settings);
}

TEST_CASE("background assets load absolute and relative BMP paths")
{
	TempDir temp;
	const auto main_path = temp.path() / "main.bmp";
	const auto settings_path = temp.path() / "settings.bmp";
	write_bmp_24(main_path, 1, 1, { { Rgb { 255, 0, 0 } } });
	write_bmp_24(settings_path, 1, 1, { { Rgb { 0, 255, 0 } } });
	write_text(temp.path() / "bg.toml",
		   "[main]\nimage = \"" + main_path.string() +
			   "\"\n\n[settings]\nimage = \"settings.bmp\"\n");

	const auto assets =
		ui::load_background_assets(temp.path() / "bg.toml", 4, 4);

	REQUIRE(assets.main);
	REQUIRE(assets.settings);
	REQUIRE(assets.main->descriptor() != nullptr);
	REQUIRE(assets.settings->descriptor() != nullptr);
	CHECK(assets.main->width() == 4);
	CHECK(assets.main->height() == 4);
	CHECK(assets.main->rgb565_pixel(0, 0) == rgb565(255, 0, 0));
	CHECK(assets.settings->rgb565_pixel(3, 3) == rgb565(0, 255, 0));
}

TEST_CASE("background assets ignore invalid config entries")
{
	TempDir temp;
	write_text(temp.path() / "bg.toml",
		   R"toml(
[main]
image = ""

[settings]
image = "missing.bmp"
)toml");

	const auto assets =
		ui::load_background_assets(temp.path() / "bg.toml", 4, 4);

	CHECK_FALSE(assets.main);
	CHECK_FALSE(assets.settings);
}

TEST_CASE("background assets scale to cover and center crop")
{
	TempDir temp;
	const auto image_path = temp.path() / "wide.bmp";
	write_bmp_24(image_path, 4, 2,
		     {
			     Rgb { 255, 0, 0 },
			     Rgb { 0, 255, 0 },
			     Rgb { 0, 0, 255 },
			     Rgb { 255, 255, 255 },
			     Rgb { 255, 0, 0 },
			     Rgb { 0, 255, 0 },
			     Rgb { 0, 0, 255 },
			     Rgb { 255, 255, 255 },
		     });
	write_text(
		temp.path() / "bg.toml",
		"[main]\nimage = \"wide.bmp\"\n\n[settings]\nimage = \"wide.bmp\"\n");

	const auto assets =
		ui::load_background_assets(temp.path() / "bg.toml", 2, 2);

	REQUIRE(assets.main);
	CHECK(assets.main->rgb565_pixel(0, 0) == rgb565(0, 255, 0));
	CHECK(assets.main->rgb565_pixel(1, 0) == rgb565(0, 0, 255));
	CHECK(assets.main->rgb565_pixel(0, 1) == rgb565(0, 255, 0));
	CHECK(assets.main->rgb565_pixel(1, 1) == rgb565(0, 0, 255));
}

TEST_CASE("background image names reject paths and non-BMP files")
{
	std::string error;
	CHECK(ui::validate_background_image_name("dial.bmp", error));
	CHECK(ui::validate_background_image_name("dial-01_BG.BMP", error));
	CHECK_FALSE(ui::validate_background_image_name("", error));
	CHECK_FALSE(ui::validate_background_image_name("../dial.bmp", error));
	CHECK_FALSE(ui::validate_background_image_name("sub/dial.bmp", error));
	CHECK_FALSE(ui::validate_background_image_name("dial.png", error));
	CHECK_FALSE(ui::validate_background_image_name("dial bg.bmp", error));
}

TEST_CASE("background assets load by image name and persist selection")
{
	TempDir temp;
	const auto image_path = temp.path() / "dial.bmp";
	const auto config_path = temp.path() / "bg.toml";
	write_bmp_24(image_path, 1, 1, { { Rgb { 12, 34, 56 } } });

	ui::BackgroundAssets direct_assets;
	std::string error;
	REQUIRE(ui::load_background_assets_by_name(
		"dial.bmp", 3, 3, direct_assets, error, temp.path()));
	REQUIRE(direct_assets.main);
	REQUIRE(direct_assets.settings);
	CHECK(direct_assets.main->rgb565_pixel(0, 0) == rgb565(12, 34, 56));
	CHECK(direct_assets.settings->rgb565_pixel(2, 2) == rgb565(12, 34, 56));

	REQUIRE(ui::save_background_selection("dial.bmp", error, config_path));
	const auto persisted_assets =
		ui::load_background_assets(config_path, 3, 3);
	REQUIRE(persisted_assets.main);
	REQUIRE(persisted_assets.settings);
	CHECK(persisted_assets.main->rgb565_pixel(1, 1) == rgb565(12, 34, 56));
	CHECK(persisted_assets.settings->rgb565_pixel(1, 1) ==
	      rgb565(12, 34, 56));
}
