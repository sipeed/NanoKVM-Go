#include <optional>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/hardware_version.hpp"

using namespace lvgl_main;

TEST_CASE("chip type text maps to NanoKVM hardware version names")
{
	CHECK(hardware_version_from_chip_type("AX620QF_CHIP") ==
	      std::optional<std::string> { "Go" });
	CHECK(hardware_version_from_chip_type("AX620QE_CHIP") ==
	      std::optional<std::string> { "Go+" });
}

TEST_CASE("chip type mapping tolerates proc whitespace and rejects unknown text")
{
	CHECK(hardware_version_from_chip_type("\nAX620QF_CHIP\n") ==
	      std::optional<std::string> { "Go" });
	CHECK_FALSE(hardware_version_from_chip_type("AX620QF_CHIP 256M")
			    .has_value());
	CHECK_FALSE(hardware_version_from_chip_type("").has_value());
}
