#pragma once

#include <chrono>
#include <cstddef>
#include <initializer_list>
#include <random>
#include <string>
#include <string_view>
#include <vector>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/wifi_runtime_state.hpp"

namespace lvgl_main
{

inline constexpr std::chrono::seconds kWifiNoSavedNetworkQrDelay { 1 };
inline constexpr std::chrono::seconds kWifiQrDismissDirectTimeout { 2 };
inline constexpr std::chrono::seconds kWifiQrDismissAfterClientTimeout { 8 };
inline constexpr std::string_view kWifiApGatewayIp = "192.168.12.1";
inline constexpr std::size_t kWifiApPasswordLength { 8 };
inline constexpr std::string_view kWifiApPasswordCharset =
	"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
inline constexpr std::string_view kWifiConnectingMainIpPlaceholder =
	"Reconnecting...";

inline std::chrono::milliseconds
wifi_saved_network_connect_timeout(const AppConfig &config)
{
	return std::chrono::milliseconds(
		config.ui.wifi_boot_reconnect_timeout_ms);
}

struct WifiRuntimeUiStateInput {
	bool boot_reconnect_running { false };
	WifiRuntimeState wifi_runtime_state { WifiRuntimeState::Unknown };
	bool has_main_ip { false };
};

struct WifiRuntimeUiState {
	std::string_view main_ip_placeholder;
	bool settings_wifi_pending { false };
	bool settings_wifi_click_enabled { true };
};

inline bool wifi_runtime_state_is_connecting(WifiRuntimeState state)
{
	return state == WifiRuntimeState::CacheConnecting ||
	       state == WifiRuntimeState::WaitingExternalConnection;
}

inline WifiRuntimeUiState
make_wifi_runtime_ui_state(const WifiRuntimeUiStateInput &input)
{
	WifiRuntimeUiState state;
	const bool connecting =
		wifi_runtime_state_is_connecting(input.wifi_runtime_state);
	if (connecting && !input.has_main_ip) {
		state.main_ip_placeholder = kWifiConnectingMainIpPlaceholder;
	}
	state.settings_wifi_pending = input.boot_reconnect_running;
	state.settings_wifi_click_enabled = !input.boot_reconnect_running;
	return state;
}

inline bool is_station_wifi_ip(std::string_view ip)
{
	return !ip.empty() && ip != kWifiApGatewayIp;
}

inline std::vector<std::string>
make_wifi_script_argv(const NanokvmRuntimeConfig &config,
		      const std::vector<std::string> &args)
{
	std::vector<std::string> command;
	command.reserve(args.size() + 1U);
	command.push_back(config.wifi_script);
	for (const auto &arg : args) {
		command.push_back(arg);
	}
	return command;
}

inline std::vector<std::string>
make_wifi_script_argv(const NanokvmRuntimeConfig &config,
		      std::initializer_list<std::string_view> args)
{
	std::vector<std::string> command;
	command.reserve(args.size() + 1U);
	command.push_back(config.wifi_script);
	for (std::string_view arg : args) {
		command.emplace_back(arg);
	}
	return command;
}

inline bool wifi_script_arg_is_secret(const std::vector<std::string> &args,
				      std::size_t index)
{
	if (args.empty() || index != 2U) {
		return false;
	}
	const std::string_view command = args.front();
	return command == "ap" || command == "connect" ||
	       command == "connect_start";
}

inline std::string_view
wifi_script_secret_for_log(const std::vector<std::string> &args)
{
	return args.size() > 2U && wifi_script_arg_is_secret(args, 2U) ?
		       std::string_view(args[2]) :
		       std::string_view();
}

inline std::string
wifi_script_args_for_log(const std::vector<std::string> &args)
{
	std::string text;
	for (std::size_t i = 0; i < args.size(); ++i) {
		if (!text.empty()) {
			text += ' ';
		}
		text += wifi_script_arg_is_secret(args, i) ? "<redacted>" :
							     args[i];
	}
	return text;
}

inline std::vector<std::string>
make_wifi_reset_clear_argv(const NanokvmRuntimeConfig &config)
{
	return make_wifi_script_argv(config, { "clear" });
}

inline std::vector<std::string>
make_wifi_reset_off_argv(const NanokvmRuntimeConfig &config)
{
	return make_wifi_script_argv(config, { "off" });
}

inline bool wifi_script_output_has_error(std::string_view output)
{
	return output.find("ERROR:") != std::string_view::npos;
}

inline std::string wifi_script_ssid_arg(std::string_view ssid)
{
	bool has_non_ascii = false;
	for (unsigned char ch : ssid) {
		if (ch >= 0x80U) {
			has_non_ascii = true;
			break;
		}
	}
	if (!has_non_ascii) {
		return std::string(ssid);
	}

	constexpr std::string_view hex = "0123456789abcdef";
	std::string escaped;
	escaped.reserve(ssid.size() * 4U);
	for (unsigned char ch : ssid) {
		escaped += "\\x";
		escaped += hex[(ch >> 4U) & 0x0FU];
		escaped += hex[ch & 0x0FU];
	}
	return escaped;
}

inline bool wifi_script_result_ok(int exit_code, std::string_view output)
{
	return exit_code == 0 && !wifi_script_output_has_error(output);
}

inline bool is_valid_wifi_ap_password(std::string_view password)
{
	if (password.size() != kWifiApPasswordLength) {
		return false;
	}

	for (unsigned char ch : password) {
		const bool alphanumeric = (ch >= '0' && ch <= '9') ||
					  (ch >= 'A' && ch <= 'Z') ||
					  (ch >= 'a' && ch <= 'z');
		if (!alphanumeric) {
			return false;
		}
	}
	return true;
}

template <typename UniformRandomBitGenerator>
inline std::string make_wifi_ap_password(UniformRandomBitGenerator &generator)
{
	std::uniform_int_distribution<std::size_t> distribution(
		0U, kWifiApPasswordCharset.size() - 1U);
	std::string password;
	password.reserve(kWifiApPasswordLength);
	for (std::size_t i = 0; i < kWifiApPasswordLength; ++i) {
		password += kWifiApPasswordCharset[distribution(generator)];
	}
	return password;
}

inline std::string percent_encode_wifi_form_value(std::string_view value)
{
	constexpr std::string_view hex = "0123456789ABCDEF";
	std::string encoded;
	for (unsigned char ch : value) {
		const bool unreserved = (ch >= 'A' && ch <= 'Z') ||
					(ch >= 'a' && ch <= 'z') ||
					(ch >= '0' && ch <= '9') || ch == '-' ||
					ch == '_' || ch == '.' || ch == '*';
		if (unreserved) {
			encoded += static_cast<char>(ch);
		} else if (ch == ' ') {
			encoded += '+';
		} else {
			encoded += '%';
			encoded += hex[(ch >> 4U) & 0x0FU];
			encoded += hex[ch & 0x0FU];
		}
	}
	return encoded;
}

inline std::string make_wifi_ap_web_url(std::string_view ap_ip,
					std::string_view password)
{
	if (ap_ip.empty() || password.empty()) {
		return {};
	}
	return "http://" + std::string(ap_ip) +
	       "/#/wifi?p=" + percent_encode_wifi_form_value(password);
}

inline bool wifi_script_reports_no_saved_network(std::string_view output)
{
	return output.find("ERROR: NO_SAVED_NETWORK") != std::string_view::npos;
}

inline bool wifi_script_reports_no_interface(std::string_view output)
{
	return output.find("ERROR: NO_INTERFACE") != std::string_view::npos;
}

inline std::chrono::seconds
wifi_qr_dismiss_wait_timeout(bool ap_or_web_client_seen)
{
	return ap_or_web_client_seen ? kWifiQrDismissAfterClientTimeout :
				       kWifiQrDismissDirectTimeout;
}

} // namespace lvgl_main
