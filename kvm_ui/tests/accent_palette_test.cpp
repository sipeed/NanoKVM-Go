#include <catch2/catch_test_macros.hpp>

#include <cmath>
#include <set>

#include "ui/ui_accent_palette.hpp"

namespace
{

double linearize_channel(double channel)
{
	if (channel <= 0.04045)
		return channel / 12.92;
	return std::pow((channel + 0.055) / 1.055, 2.4);
}

double white_contrast(uint32_t color)
{
	const double red = linearize_channel(((color >> 16) & 0xFF) / 255.0);
	const double green = linearize_channel(((color >> 8) & 0xFF) / 255.0);
	const double blue = linearize_channel((color & 0xFF) / 255.0);
	const double luminance =
		0.2126 * red + 0.7152 * green + 0.0722 * blue;
	return 1.05 / (luminance + 0.05);
}

} // namespace

TEST_CASE("reserved accent palette holds the white contrast gate",
	  "[ui][accent][palette]")
{
	for (const uint32_t color : ui::accent::PALETTE) {
		INFO("accent color " << color);
		CHECK(white_contrast(color) >= 4.5);
	}
}

TEST_CASE("reserved accent palette values are unique",
	  "[ui][accent][palette]")
{
	const std::set<uint32_t> unique(ui::accent::PALETTE.begin(),
					ui::accent::PALETTE.end());
	CHECK(unique.size() == ui::accent::PALETTE.size());
}

TEST_CASE("reserved accent palette size matches the approved count",
	  "[ui][accent][palette]")
{
	CHECK(ui::accent::PALETTE.size() == 13U);
}
