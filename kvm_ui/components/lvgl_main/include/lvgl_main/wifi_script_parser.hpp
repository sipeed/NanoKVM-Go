#pragma once

#include <cstdint>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace lvgl_main
{

enum class WifiScriptStatusKind : uint8_t {
	Connected,
	Ap,
	Disconnected,
	NoInterface,
	Unknown,
};

struct WifiScriptStatus {
	WifiScriptStatusKind kind { WifiScriptStatusKind::Unknown };
	std::string ssid;
};

struct WifiScanNetwork {
	std::string ssid;
	std::string bssid;
	int signal { 0 };
	int frequency { 0 };
	std::string security;
	bool supported { true };
};

struct WifiScanResult {
	bool ok { true };
	std::string error;
	std::vector<WifiScanNetwork> networks;
};

enum class WifiSavedNetworkAvailability : uint8_t {
	Unknown,
	Absent,
	Present,
};

enum class WifiConnectStatus : uint8_t {
	Success,
	Failure,
	Unknown,
};

struct WifiConnectResult {
	WifiConnectStatus status { WifiConnectStatus::Unknown };
	std::string ssid;
	std::string error_code;
	std::string raw_summary;
};

WifiScriptStatus parse_wifi_status_output(std::string_view output);
std::optional<int> parse_wifi_device_count_output(std::string_view output);
std::optional<bool> parse_wifi_boot_enabled_output(std::string_view output);
WifiScanResult parse_wifi_scan_output(std::string_view output);
WifiConnectResult parse_wifi_connect_output(std::string_view output,
					    int exit_code = 0);
WifiSavedNetworkAvailability
parse_wifi_saved_networks_output(std::string_view output);

} // namespace lvgl_main
