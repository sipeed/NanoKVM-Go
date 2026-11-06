#include <string>

#include <catch2/catch_test_macros.hpp>

#include "ui/wifi_ssid_display_policy.hpp"

TEST_CASE("Wi-Fi SSID display policy preserves approved LTR scripts")
{
	const std::string ssid = "Café Ωμέγα Привет 中文かなカナ한글";

	const auto display = ui::make_wifi_ssid_display(ssid);

	REQUIRE(display.visible);
	CHECK(display.text == ssid);

	const auto combining = ui::make_wifi_ssid_display("Cafe\xCC\x81");
	REQUIRE(combining.visible);
	CHECK(combining.text == "Cafe\xCC\x81");

	const auto han_extension_i =
		ui::make_wifi_ssid_display("\xF0\xAE\xAF\xB0");
	REQUIRE(han_extension_i.visible);
	CHECK(han_extension_i.text == "\xF0\xAE\xAF\xB0");

	const auto katakana_extended_b =
		ui::make_wifi_ssid_display("\xF0\x9A\xBF\xB0");
	REQUIRE(katakana_extended_b.visible);
	CHECK(katakana_extended_b.text == "\xF0\x9A\xBF\xB0");
}

TEST_CASE("Wi-Fi SSID display policy replaces complete emoji sequences")
{
	const auto zwj_sequence = ui::make_wifi_ssid_display("Home 👩🏽‍💻");
	REQUIRE(zwj_sequence.visible);
	CHECK(zwj_sequence.text == "Home □");

	const auto keycap_sequence = ui::make_wifi_ssid_display("Wi-Fi 1️⃣");
	REQUIRE(keycap_sequence.visible);
	CHECK(keycap_sequence.text == "Wi-Fi □");
}

TEST_CASE("Wi-Fi SSID display policy hides unsupported scripts and BiDi text")
{
	CHECK_FALSE(ui::make_wifi_ssid_display("شبكة").visible);
	CHECK_FALSE(ui::make_wifi_ssid_display("Lab شبكة").visible);
	CHECK_FALSE(ui::make_wifi_ssid_display("हिन्दी").visible);
	CHECK_FALSE(ui::make_wifi_ssid_display("Հայերեն").visible);
	CHECK_FALSE(ui::make_wifi_ssid_display("\xCF\xA2").visible); // Coptic U+03E2
	CHECK_FALSE(ui::make_wifi_ssid_display("\xF0\x9B\x85\xB0").visible); // Nushu U+1B170
	CHECK_FALSE(ui::make_wifi_ssid_display("Lab\xE2\x80\x8EWiFi").visible);
}

TEST_CASE("Wi-Fi SSID display policy escapes controls and non-UTF-8 bytes")
{
	const auto control = ui::make_wifi_ssid_display("Lab\nWiFi\x7F");
	REQUIRE(control.visible);
	CHECK(control.text == "Lab\\x0AWiFi\\x7F");

	const auto line_separator =
		ui::make_wifi_ssid_display("Lab\xE2\x80\xA8WiFi");
	REQUIRE(line_separator.visible);
	CHECK(line_separator.text == "Lab\\xE2\\x80\\xA8WiFi");

	std::string invalid;
	invalid.push_back(static_cast<char>(0xFF));
	invalid += "A";
	const auto escaped = ui::make_wifi_ssid_display(invalid);
	REQUIRE(escaped.visible);
	CHECK(escaped.text == "\\xFF\\x41");

	const std::string nul_ssid("Lab\0WiFi", 8U);
	CHECK_FALSE(ui::make_wifi_ssid_display(nul_ssid).visible);
}

TEST_CASE("Wi-Fi SSID display policy retains Latin extensions")
{
	const std::string ssid = "\xCA\xB0\xE2\xB1\xA0\xEF\xAC\x80";

	const auto display = ui::make_wifi_ssid_display(ssid);

	REQUIRE(display.visible);
	CHECK(display.text == ssid);
}
