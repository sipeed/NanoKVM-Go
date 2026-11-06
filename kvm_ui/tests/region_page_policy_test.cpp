#include <catch2/catch_test_macros.hpp>

#include <cstdint>

#include "ui/region_page_policy.hpp"

namespace
{

using ui::RegionOption;

} // namespace

TEST_CASE("Region option labels map to their server tokens")
{
	STATIC_REQUIRE(ui::region_option_label(RegionOption::ChinaRegion) ==
		       "CN");
	STATIC_REQUIRE(ui::region_option_label(RegionOption::Global) ==
		       "Global");
}

TEST_CASE("Region option resolves from the CN flag")
{
	STATIC_REQUIRE(ui::region_option_from_cn(true) ==
		       RegionOption::ChinaRegion);
	STATIC_REQUIRE(ui::region_option_from_cn(false) ==
		       RegionOption::Global);
}

TEST_CASE("Region option other flips between the two regions")
{
	STATIC_REQUIRE(ui::region_other(RegionOption::ChinaRegion) ==
		       RegionOption::Global);
	STATIC_REQUIRE(ui::region_other(RegionOption::Global) ==
		       RegionOption::ChinaRegion);
	STATIC_REQUIRE(ui::region_other(ui::region_other(
			       RegionOption::ChinaRegion)) ==
		       RegionOption::ChinaRegion);
}
