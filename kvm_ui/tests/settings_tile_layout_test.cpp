#include <initializer_list>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "ui/settings_tile_layout.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

namespace
{

using Tile = ui::settings_tile_layout::SettingsTile;

void check_tile_order(const std::vector<Tile> &actual,
		      std::initializer_list<Tile> expected)
{
	REQUIRE(actual.size() == expected.size());
	auto expected_it = expected.begin();
	for (std::size_t index = 0; index < actual.size();
	     ++index, ++expected_it) {
		CHECK(actual[index] == *expected_it);
	}
}

} // namespace

TEST_CASE("settings tile layout keeps replay on main page when visible")
{
	const auto layout =
		ui::settings_tile_layout::build_settings_tile_layout(true);

	check_tile_order(layout.main,
			 { Tile::WIFI, Tile::MCP, Tile::REPLAY, Tile::NCM,
			   Tile::SSH, Tile::AUTO_ROTATION });

#if !NANOKVM_GO_RELEASE_BUILD
	check_tile_order(layout.second,
			 { Tile::SCREEN, Tile::SERVO, Tile::APPS,
			   Tile::TAILSCALE, Tile::REGION, Tile::NETWORK,
			   Tile::SECOND_ABOUT });
#else
	check_tile_order(layout.second,
			 { Tile::SCREEN, Tile::APPS, Tile::TAILSCALE,
			   Tile::REGION, Tile::NETWORK, Tile::SECOND_ABOUT });
#endif

	CHECK(layout.second.back() == Tile::SECOND_ABOUT);
	CHECK(layout.second[layout.second.size() - 2] == Tile::NETWORK);
	CHECK(layout.second[layout.second.size() - 3] == Tile::REGION);
	CHECK(layout.second[layout.second.size() - 4] == Tile::TAILSCALE);
}

TEST_CASE(
	"settings tile layout promotes screen to main page when replay is hidden")
{
	const auto layout =
		ui::settings_tile_layout::build_settings_tile_layout(false);

	check_tile_order(layout.main,
			 { Tile::WIFI, Tile::MCP, Tile::SCREEN, Tile::NCM,
			   Tile::SSH, Tile::AUTO_ROTATION });

#if !NANOKVM_GO_RELEASE_BUILD
	check_tile_order(layout.second,
			 { Tile::SERVO, Tile::APPS, Tile::TAILSCALE,
			   Tile::REGION, Tile::NETWORK, Tile::SECOND_ABOUT });
#else
	check_tile_order(layout.second,
			 { Tile::APPS, Tile::TAILSCALE, Tile::REGION,
			   Tile::NETWORK, Tile::SECOND_ABOUT });
#endif

	CHECK(layout.second.back() == Tile::SECOND_ABOUT);
	CHECK(layout.second[layout.second.size() - 2] == Tile::NETWORK);
	CHECK(layout.second[layout.second.size() - 3] == Tile::REGION);
	CHECK(layout.second[layout.second.size() - 4] == Tile::TAILSCALE);
}
