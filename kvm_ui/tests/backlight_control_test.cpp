#include <chrono>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/backlight_control.hpp"

using namespace lvgl_main::backlight;

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_backlight_test_" + std::to_string(stamp));
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

void write_text(const std::filesystem::path &path, const std::string &text)
{
	std::ofstream out(path, std::ios::trunc);
	REQUIRE(out.good());
	out << text;
	REQUIRE(out.good());
}

std::string read_text(const std::filesystem::path &path)
{
	std::ifstream in(path);
	REQUIRE(in.good());
	std::ostringstream out;
	out << in.rdbuf();
	return out.str();
}

} // namespace

TEST_CASE("backlight percent maps to absolute normal raw values by default")
{
	CHECK(backlight_percent_to_raw(100, 255) == 255);
	CHECK(backlight_percent_to_raw(80, 255) == 105);
	CHECK(backlight_percent_to_raw(60, 255) == 68);
	CHECK(backlight_percent_to_raw(40, 255) == 55);
	CHECK(backlight_percent_to_raw(20, 255) == 51);
	CHECK(backlight_percent_to_raw(0, 255) == 51);
	CHECK(backlight_percent_to_raw(101, 255) == 255);
	CHECK(backlight_percent_to_raw(20, 255) != 0);
	CHECK(backlight_percent_to_raw(80, 255) != 204);
	CHECK(backlight_percent_to_raw(60, 255) != 128);
	CHECK(backlight_percent_to_raw(40, 255) != 64);
	CHECK(clamp_backlight_percent(0) == 20);
	CHECK(clamp_backlight_percent(101) == 100);
}

TEST_CASE("backlight percent can map to inverted raw values")
{
	CHECK(backlight_percent_to_raw(100, 255, true) == 0);
	CHECK(backlight_percent_to_raw(80, 255, true) == 150);
	CHECK(backlight_percent_to_raw(60, 255, true) == 187);
	CHECK(backlight_percent_to_raw(40, 255, true) == 200);
	CHECK(backlight_percent_to_raw(20, 255, true) == 204);
	CHECK(backlight_percent_to_raw(0, 255, true) == 204);
	CHECK(backlight_percent_to_raw(101, 255, true) == 0);
	CHECK(backlight_percent_to_raw(20, 255, true) != 255);
	CHECK(backlight_percent_to_raw(80, 255, true) != 64);
	CHECK(backlight_percent_to_raw(60, 255, true) != 128);
	CHECK(backlight_percent_to_raw(40, 255, true) != 191);
}

TEST_CASE("backlight raw never reverses as UI percent increases")
{
	int previous = backlight_percent_to_raw(20, 255);
	for (int percent = 21; percent <= 100; ++percent) {
		const int raw = backlight_percent_to_raw(percent, 255);
		CHECK(raw >= previous);
		previous = raw;
	}

	previous = backlight_percent_to_raw(20, 255, true);
	for (int percent = 21; percent <= 100; ++percent) {
		const int raw = backlight_percent_to_raw(percent, 255, true);
		CHECK(raw <= previous);
		previous = raw;
	}
}

TEST_CASE("backlight raw maps back through the nonlinear diagnostic inverse")
{
	CHECK(backlight_raw_to_percent(255, 255) == 100);
	CHECK(backlight_raw_to_percent(105, 255) == 80);
	CHECK(backlight_raw_to_percent(68, 255) == 60);
	CHECK(std::abs(backlight_raw_to_percent(55, 255) - 40) <= 1);
	CHECK(backlight_raw_to_percent(51, 255) == 20);
	CHECK(backlight_raw_to_percent(0, 255) == 20);
	for (const int percent : { 20, 35, 50, 60, 75, 100 }) {
		const int raw = backlight_percent_to_raw(percent, 255);
		const int round_trip = backlight_raw_to_percent(raw, 255);
		CHECK(std::abs(round_trip - percent) <= 1);
	}

	CHECK(backlight_raw_to_percent(0, 255, true) == 100);
	CHECK(backlight_raw_to_percent(150, 255, true) == 80);
	CHECK(backlight_raw_to_percent(187, 255, true) == 60);
	CHECK(std::abs(backlight_raw_to_percent(200, 255, true) - 40) <= 1);
	CHECK(backlight_raw_to_percent(204, 255, true) == 20);
	CHECK(backlight_raw_to_percent(255, 255, true) == 20);
	for (const int percent : { 20, 35, 50, 60, 75, 100 }) {
		const int raw = backlight_percent_to_raw(percent, 255, true);
		const int round_trip = backlight_raw_to_percent(raw, 255, true);
		CHECK(std::abs(round_trip - percent) <= 1);
	}
}

TEST_CASE("backlight state keeps fallback percent instead of raw inverse")
{
	TempDir temp;
	write_text(temp.path() / "max_brightness", "255\n");
	write_text(temp.path() / "brightness", "32\n");

	const auto state = read_backlight_state(temp.path(), 60);
	REQUIRE(state.available);
	CHECK(state.raw == 32);
	CHECK(state.raw_max == 255);
	CHECK(state.screen_on);
	CHECK(state.percent == 60);
}

TEST_CASE("backlight helper reads and writes mock sysfs")
{
	TempDir temp;
	write_text(temp.path() / "max_brightness", "255\n");
	write_text(temp.path() / "brightness", "0\n");

	const auto full = write_backlight_percent(temp.path(), 100);
	REQUIRE(full.ok);
	CHECK(full.raw == 255);
	CHECK(read_text(temp.path() / "brightness") == "255");

	const auto low = write_backlight_percent(temp.path(), 20);
	REQUIRE(low.ok);
	CHECK(low.raw == 51);
	CHECK(read_text(temp.path() / "brightness") == "51");

	auto state = read_backlight_state(temp.path(), 100);
	REQUIRE(state.available);
	CHECK(state.screen_on);
	CHECK(state.percent == 100);

	const auto off = write_backlight_off(temp.path());
	REQUIRE(off.ok);
	CHECK(off.raw == 0);
	state = read_backlight_state(temp.path(), 75);
	REQUIRE(state.available);
	CHECK_FALSE(state.screen_on);
	CHECK(state.percent == 75);
}

TEST_CASE("backlight helper reads and writes inverted mock sysfs")
{
	TempDir temp;
	write_text(temp.path() / "max_brightness", "255\n");
	write_text(temp.path() / "brightness", "255\n");

	const auto full = write_backlight_percent(temp.path(), 100, true);
	REQUIRE(full.ok);
	CHECK(full.raw == 0);
	CHECK(read_text(temp.path() / "brightness") == "0");

	const auto low = write_backlight_percent(temp.path(), 20, true);
	REQUIRE(low.ok);
	CHECK(low.raw == 204);
	CHECK(read_text(temp.path() / "brightness") == "204");

	auto state = read_backlight_state(temp.path(), 100, true);
	REQUIRE(state.available);
	CHECK(state.screen_on);
	CHECK(state.percent == 100);

	const auto off = write_backlight_off(temp.path(), true);
	REQUIRE(off.ok);
	CHECK(off.raw == 255);
	state = read_backlight_state(temp.path(), 75, true);
	REQUIRE(state.available);
	CHECK_FALSE(state.screen_on);
	CHECK(state.percent == 75);
}
