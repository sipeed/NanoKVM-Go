#include <optional>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/device_number_parser.hpp"

using namespace lvgl_main;

TEST_CASE("parse_device_number_output takes the substring after the last ']'")
{
	CHECK(parse_device_number_output("[BOARD]AX620QE_LP4_512M") ==
	      std::optional<std::string> { "AX620QE_LP4_512M" });
	// Multiple brackets: only the part after the LAST ']' is the value.
	CHECK(parse_device_number_output("[A][B]AX620QE_LP4_512M") ==
	      std::optional<std::string> { "AX620QE_LP4_512M" });
}

TEST_CASE("parse_device_number_output falls back to the whole line without ']'")
{
	CHECK(parse_device_number_output("AX620QE_LP4_NANOAGENT_512M") ==
	      std::optional<std::string> { "AX620QE_LP4_NANOAGENT_512M" });
	CHECK(parse_device_number_output("no brackets here") ==
	      std::optional<std::string> { "no brackets here" });
}

TEST_CASE("parse_device_number_output trims whitespace around the value")
{
	// The extracted value is trimmed of surrounding ASCII whitespace.
	CHECK(parse_device_number_output("[A]   AX620QE   ") ==
	      std::optional<std::string> { "AX620QE" });
	// Whitespace-only tail after ']' collapses to empty.
	CHECK(parse_device_number_output("[A]   ") == std::optional<std::string> { "" });
}

TEST_CASE("parse_device_number_output skips blank and comment lines")
{
	CHECK(parse_device_number_output("\n# ax_efuse_tool v1.2\n[BOARD]AX620QE\n") ==
	      std::optional<std::string> { "AX620QE" });
	CHECK(parse_device_number_output("# only comments\n\n# more") ==
	      std::nullopt);
	CHECK(parse_device_number_output("") == std::nullopt);
}

TEST_CASE("parse_device_number_output only inspects the first meaningful line")
{
	// A later meaningful line with a bracket must not override the first
	// meaningful line that lacks one (whole-line fallback applies).
	CHECK(parse_device_number_output("plain-first\n[BRACKET]second") ==
	      std::optional<std::string> { "plain-first" });
}
