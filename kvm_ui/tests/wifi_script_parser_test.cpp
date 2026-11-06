#include <chrono>
#include <optional>
#include <random>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"
#include "lvgl_main/wifi_script_parser.hpp"

using namespace lvgl_main;

TEST_CASE("WiFi script status output parser recognizes primary states")
{
	const auto connected = parse_wifi_status_output("CONNECTED My WiFi\n");
	CHECK(connected.kind == WifiScriptStatusKind::Connected);
	CHECK(connected.ssid == "My WiFi");

	const auto ap = parse_wifi_status_output("AP NanoKVM-KVM\n");
	CHECK(ap.kind == WifiScriptStatusKind::Ap);
	CHECK(ap.ssid == "NanoKVM-KVM");

	CHECK(parse_wifi_status_output("DISCONNECTED\n").kind ==
	      WifiScriptStatusKind::Disconnected);
	CHECK(parse_wifi_status_output("ERROR: NO_INTERFACE\n").kind ==
	      WifiScriptStatusKind::NoInterface);
	CHECK(parse_wifi_status_output("OK: Connected to Lab\n").kind ==
	      WifiScriptStatusKind::Unknown);
}

TEST_CASE("WiFi script status parser decodes x-escaped UTF-8 SSIDs")
{
	const auto connected = parse_wifi_status_output(
		R"ssid(CONNECTED \xe4\xb8\xad\xe6\x96\x87\x20\x57\x69\x46\x69)ssid");
	CHECK(connected.kind == WifiScriptStatusKind::Connected);
	CHECK(connected.ssid == "中文 WiFi");

	const auto ap = parse_wifi_status_output(
		R"ssid(AP \xe4\xb8\xad\xe6\x96\x87\x20\x41\x50)ssid");
	CHECK(ap.kind == WifiScriptStatusKind::Ap);
	CHECK(ap.ssid == "中文 AP");
}

TEST_CASE("WiFi device_count parser uses only the first integer line")
{
	CHECK(parse_wifi_device_count_output("0\n") ==
	      std::optional<int> { 0 });
	CHECK(parse_wifi_device_count_output("1\n") ==
	      std::optional<int> { 1 });
	CHECK(parse_wifi_device_count_output("0\nERROR: NO_INTERFACE\n") ==
	      std::optional<int> { 0 });
	CHECK(parse_wifi_device_count_output(" 12 \nignored") ==
	      std::optional<int> { 12 });
	CHECK_FALSE(parse_wifi_device_count_output("ERROR: NO_INTERFACE\n")
			    .has_value());
	CHECK_FALSE(parse_wifi_device_count_output("\n").has_value());
}

TEST_CASE("WiFi boot-enabled parser accepts only the exact state token")
{
	CHECK(parse_wifi_boot_enabled_output("true\n") ==
	      std::optional<bool> { true });
	CHECK(parse_wifi_boot_enabled_output(" false \n") ==
	      std::optional<bool> { false });
	CHECK_FALSE(parse_wifi_boot_enabled_output("").has_value());
	CHECK_FALSE(parse_wifi_boot_enabled_output("true\nfalse\n").has_value());
	CHECK_FALSE(parse_wifi_boot_enabled_output("TRUE\n").has_value());
	CHECK_FALSE(parse_wifi_boot_enabled_output("ERROR: STATE_INVALID\n")
				    .has_value());
}

TEST_CASE("WiFi scan parser accepts an empty array")
{
	const auto result = parse_wifi_scan_output("[]\n");

	CHECK(result.ok);
	CHECK(result.error.empty());
	CHECK(result.networks.empty());
}

TEST_CASE("WiFi scan parser reads normal entries and keeps strongest duplicate")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"Lab","bssid":"00:11:22:33:44:55","signal":-70,"frequency":2412,"security":"WPA2"},
        {"ssid":"Open","bssid":"66:77:88:99:aa:bb","signal":-40,"frequency":5180,"security":"OPEN"},
        {"ssid":"Lab","bssid":"00:11:22:33:44:66","signal":-50,"frequency":2437,"security":"WPA2"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 2U);
	CHECK(result.networks[0].ssid == "Lab");
	CHECK(result.networks[0].bssid == "00:11:22:33:44:66");
	CHECK(result.networks[0].signal == -50);
	CHECK(result.networks[0].frequency == 2437);
	CHECK(result.networks[0].security == "WPA2");
	CHECK(result.networks[0].supported);
	CHECK(result.networks[1].ssid == "Open");
	CHECK(result.networks[1].supported);
}

TEST_CASE("WiFi scan parser preserves SSID boundary whitespace")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"  Lab WiFi \t","bssid":" 00:11:22:33:44:55 ","signal":-70,"frequency":2412,"security":" WPA2 "}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 1U);
	CHECK(result.networks[0].ssid == "  Lab WiFi \t");
	CHECK(result.networks[0].bssid == "00:11:22:33:44:55");
	CHECK(result.networks[0].security == "WPA2");
}

TEST_CASE("WiFi scan parser marks EAP networks unsupported")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"Corp","bssid":"00:11:22:33:44:55","signal":-45,"frequency":2412,"security":"EAP"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 1U);
	CHECK(result.networks[0].ssid == "Corp");
	CHECK(result.networks[0].security == "EAP");
	CHECK_FALSE(result.networks[0].supported);
}

TEST_CASE("WiFi scan parser accepts device x-escaped UTF-8 SSIDs")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"\xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95","bssid":"12:34:56:78:9a:bc","signal":-38,"frequency":2462,"security":"WPA2"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 1U);
	CHECK(result.networks[0].ssid == "中文测试");
	CHECK(result.networks[0].bssid == "12:34:56:78:9a:bc");
	CHECK(result.networks[0].signal == -38);
	CHECK(result.networks[0].frequency == 2462);
	CHECK(result.networks[0].security == "WPA2");
	CHECK(result.networks[0].supported);
}

TEST_CASE("WiFi scan parser preserves invalid escaped SSID bytes for the script")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"\xff\x80","bssid":"12:34:56:78:9a:bc","signal":-38,"frequency":2462,"security":"WPA2"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 1U);
	CHECK(result.networks[0].ssid == R"ssid(\xFF\x80)ssid");
	CHECK(wifi_script_ssid_arg(result.networks[0].ssid) ==
	      R"ssid(\xFF\x80)ssid");
}

TEST_CASE("WiFi scan parser accepts mixed standard and x-escaped entries")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"Lab","bssid":"00:11:22:33:44:55","signal":-70,"frequency":2412,"security":"WPA2"},
        {"ssid":"\xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95","bssid":"66:77:88:99:aa:bb","signal":-40,"frequency":5180,"security":"OPEN"},
        {"ssid":"Corp","bssid":"cc:dd:ee:ff:00:11","signal":-55,"frequency":5200,"security":"EAP"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 3U);
	CHECK(result.networks[0].ssid == "Lab");
	CHECK(result.networks[0].security == "WPA2");
	CHECK(result.networks[0].supported);
	CHECK(result.networks[1].ssid == "中文测试");
	CHECK(result.networks[1].security == "OPEN");
	CHECK(result.networks[1].supported);
	CHECK(result.networks[2].ssid == "Corp");
	CHECK(result.networks[2].security == "EAP");
	CHECK_FALSE(result.networks[2].supported);
}

TEST_CASE("WiFi scan parser keeps valid JSON backslash x text unchanged")
{
	const auto result = parse_wifi_scan_output(R"json([
        {"ssid":"literal \\x41","bssid":"00:11:22:33:44:55","signal":-70,"frequency":2412,"security":"OPEN"}
    ])json");

	REQUIRE(result.ok);
	REQUIRE(result.networks.size() == 1U);
	CHECK(result.networks[0].ssid == R"ssid(literal \x41)ssid");
}

TEST_CASE("WiFi scan parser rejects non JSON prefixes and suffixes")
{
	const auto prefix = parse_wifi_scan_output(R"json(scan log
        [{"ssid":"\xe4\xb8\xad\xe6\x96\x87","security":"OPEN"}]
    )json");
	CHECK_FALSE(prefix.ok);
	CHECK(prefix.error == "INVALID_JSON");

	const auto suffix = parse_wifi_scan_output(R"json([
        {"ssid":"\xe4\xb8\xad\xe6\x96\x87","security":"OPEN"}
    ]
    trailing log)json");
	CHECK_FALSE(suffix.ok);
	CHECK(suffix.error == "INVALID_JSON");
}

TEST_CASE("WiFi connect parser recognizes success and failure")
{
	const auto success =
		parse_wifi_connect_output("OK: Connected to Lab WiFi\n", 0);
	CHECK(success.status == WifiConnectStatus::Success);
	CHECK(success.ssid == "Lab WiFi");
	CHECK(success.error_code.empty());
	CHECK(success.raw_summary == "OK: Connected to Lab WiFi");

	const auto failure =
		parse_wifi_connect_output("ERROR: CONNECT_FAILED\n", 1);
	CHECK(failure.status == WifiConnectStatus::Failure);
	CHECK(failure.ssid.empty());
	CHECK(failure.error_code == "CONNECT_FAILED");
	CHECK(failure.raw_summary == "ERROR: CONNECT_FAILED");
}

TEST_CASE("WiFi connect parser decodes x-escaped UTF-8 SSID")
{
	const auto success = parse_wifi_connect_output(
		R"ssid(OK: Connected to \xe4\xb8\xad\xe6\x96\x87\x20\x57\x69\x46\x69)ssid",
		0);

	CHECK(success.status == WifiConnectStatus::Success);
	CHECK(success.ssid == "中文 WiFi");
}

TEST_CASE("WiFi script SSID arguments escape non ASCII bytes")
{
	CHECK(wifi_script_ssid_arg("Lab WiFi") == "Lab WiFi");
	CHECK(wifi_script_ssid_arg("中文 WiFi") ==
	      R"ssid(\xe4\xb8\xad\xe6\x96\x87\x20\x57\x69\x46\x69)ssid");
}

TEST_CASE("WiFi script connect password is redacted for logs")
{
	CHECK(wifi_script_args_for_log(
		      { "connect", "Lab WiFi", "secret pass" }) ==
	      "connect Lab WiFi <redacted>");
	CHECK(wifi_script_args_for_log({ "connect", "Open WiFi" }) ==
	      "connect Open WiFi");
	CHECK(wifi_script_args_for_log({ "connect",
					 wifi_script_ssid_arg("中文测试"),
					 "secret pass" }) ==
	      R"ssid(connect \xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95 <redacted>)ssid");
	CHECK(wifi_script_args_for_log({ "ap", "NanoKVM", "ap pass" }) ==
	      "ap NanoKVM <redacted>");
}

TEST_CASE("WiFi reset action clears saved networks")
{
	NanokvmRuntimeConfig config;
	config.wifi_script = "/opt/scripts/wifi.sh";

	const auto clear_argv = make_wifi_reset_clear_argv(config);
	REQUIRE(clear_argv.size() == 2U);
	CHECK(clear_argv[0] == "/opt/scripts/wifi.sh");
	CHECK(clear_argv[1] == "clear");

	const auto off_argv = make_wifi_reset_off_argv(config);
	REQUIRE(off_argv.size() == 2U);
	CHECK(off_argv[0] == "/opt/scripts/wifi.sh");
	CHECK(off_argv[1] == "off");
}

TEST_CASE("WiFi QR dismiss timeout distinguishes direct exit and web flow")
{
	CHECK(wifi_qr_dismiss_wait_timeout(false) ==
	      kWifiQrDismissDirectTimeout);
	CHECK(wifi_qr_dismiss_wait_timeout(true) ==
	      kWifiQrDismissAfterClientTimeout);
}

TEST_CASE("WiFi saved-network reconnect timeout uses app config")
{
	AppConfig config;
	CHECK(wifi_saved_network_connect_timeout(config) ==
	      std::chrono::milliseconds { 15000 });

	config.ui.wifi_boot_reconnect_timeout_ms = 23000;
	CHECK(wifi_saved_network_connect_timeout(config) ==
	      std::chrono::milliseconds { 23000 });
}

TEST_CASE("WiFi startup reconnect UI state blocks settings entry")
{
	const auto connecting = make_wifi_runtime_ui_state(
		{ .boot_reconnect_running = true,
		  .wifi_runtime_state = WifiRuntimeState::CacheConnecting,
		  .has_main_ip = false });

	CHECK(connecting.main_ip_placeholder == "Reconnecting...");
	CHECK(connecting.settings_wifi_pending);
	CHECK_FALSE(connecting.settings_wifi_click_enabled);

	const auto connected = make_wifi_runtime_ui_state(
		{ .boot_reconnect_running = false,
		  .wifi_runtime_state = WifiRuntimeState::Connected,
		  .has_main_ip = true });

	CHECK(connected.main_ip_placeholder.empty());
	CHECK_FALSE(connected.settings_wifi_pending);
	CHECK(connected.settings_wifi_click_enabled);
}

TEST_CASE("WiFi connected state requires a station IP")
{
	CHECK_FALSE(is_station_wifi_ip(""));
	CHECK_FALSE(is_station_wifi_ip("192.168.12.1"));
	CHECK(is_station_wifi_ip("192.168.1.50"));
}

TEST_CASE("WiFi AP web URL encodes the AP password")
{
	const std::string url =
		make_wifi_ap_web_url("192.168.12.1", "current ap pass");

	CHECK(url == "http://192.168.12.1/#/wifi?p=current+ap+pass");
	CHECK(url.find("NanoKVM-KVM") == std::string::npos);
	CHECK(make_wifi_ap_web_url("", "current ap pass").empty());
	CHECK(make_wifi_ap_web_url("192.168.12.1", "").empty());
}

TEST_CASE("WiFi AP password policy is fixed length alphanumeric")
{
	CHECK(kWifiApPasswordLength == 8U);
	REQUIRE(kWifiApPasswordCharset.size() == 62U);
	for (unsigned char ch : kWifiApPasswordCharset) {
		CHECK(((ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') ||
		       (ch >= 'a' && ch <= 'z')));
	}

	CHECK(is_valid_wifi_ap_password("aB3dE4f5"));
	CHECK_FALSE(is_valid_wifi_ap_password("short7"));
	CHECK_FALSE(is_valid_wifi_ap_password("aB3dE4f!"));

	std::mt19937 generator(42U);
	CHECK(is_valid_wifi_ap_password(make_wifi_ap_password(generator)));
}

TEST_CASE("WiFi saved networks parser returns Present for non-empty array")
{
	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":"MyWiFi"}])json") ==
	      WifiSavedNetworkAvailability::Present);

	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":"Net1"},{"ssid":"Net2"}])json") ==
	      WifiSavedNetworkAvailability::Present);
}

TEST_CASE("WiFi saved networks parser returns Absent for empty array")
{
	CHECK(parse_wifi_saved_networks_output("[]\n") ==
	      WifiSavedNetworkAvailability::Absent);

	CHECK(parse_wifi_saved_networks_output("[]") ==
	      WifiSavedNetworkAvailability::Absent);
}

TEST_CASE(
	"WiFi saved networks parser returns Unknown for blank, empty, or malformed output")
{
	CHECK(parse_wifi_saved_networks_output("") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("   \n") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("not json") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("{}") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("\"string\"") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("42") ==
	      WifiSavedNetworkAvailability::Unknown);
	CHECK(parse_wifi_saved_networks_output("null") ==
	      WifiSavedNetworkAvailability::Unknown);
}

TEST_CASE(
	"WiFi saved networks parser returns Absent for array with no valid SSID entries")
{
	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"bssid":"00:11:22:33:44:55"}])json") ==
	      WifiSavedNetworkAvailability::Absent);

	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":""}])json") ==
	      WifiSavedNetworkAvailability::Absent);

	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":""},{"ssid":""}])json") ==
	      WifiSavedNetworkAvailability::Absent);
}

TEST_CASE(
	"WiFi saved networks parser returns Present when at least one entry has a non-empty SSID")
{
	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":""},{"ssid":"RealNet"}])json") ==
	      WifiSavedNetworkAvailability::Present);

	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":"RealNet"},{"ssid":""}])json") ==
	      WifiSavedNetworkAvailability::Present);
}

TEST_CASE(
	"WiFi saved networks parser accepts device x-escaped UTF-8 SSIDs")
{
	CHECK(parse_wifi_saved_networks_output(
		      R"json([{"ssid":"\xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95","security":"WPA2"}])json") ==
	      WifiSavedNetworkAvailability::Present);
}

TEST_CASE(
	"WiFi saved networks parser keeps mixed standard and x-escaped entries Present")
{
	CHECK(parse_wifi_saved_networks_output(R"json([
        {"ssid":"Lab","security":"WPA2"},
        {"ssid":"\xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95","security":"OPEN"}
    ])json") ==
	      WifiSavedNetworkAvailability::Present);

	// A non-ASCII escaped entry must not pollute a later standard entry
	// into Unknown: the array is still a valid saved list.
	CHECK(parse_wifi_saved_networks_output(R"json([
        {"ssid":"\xe4\xb8\xad\xe6\x96\x87","security":"OPEN"},
        {"ssid":"Guest","security":"WPA2"}
    ])json") ==
	      WifiSavedNetworkAvailability::Present);
}
