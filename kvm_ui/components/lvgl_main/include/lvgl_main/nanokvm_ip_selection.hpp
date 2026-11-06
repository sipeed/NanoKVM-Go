#pragma once

#include <algorithm>
#include <string>
#include <string_view>
#include <vector>

namespace lvgl_main
{

enum class UsbNetworkProtocol {
	NONE,
	NCM,
	RNDIS,
};

struct MainIpSelectionInputs {
	bool ncm_enabled { false };
	bool rndis_enabled { false };
	std::string ncm_ip;
	std::string rndis_ip;
	std::string wlan0_ip;
};

inline bool starts_with_text(std::string_view text, std::string_view prefix)
{
	return text.size() >= prefix.size() &&
	       text.substr(0, prefix.size()) == prefix;
}

inline UsbNetworkProtocol
classify_usb_network_interface(std::string_view ifname, bool ncm_enabled,
			       bool rndis_enabled)
{
	if (starts_with_text(ifname, "ncm")) {
		return ncm_enabled ? UsbNetworkProtocol::NCM :
				     UsbNetworkProtocol::NONE;
	}
	if (starts_with_text(ifname, "rndis")) {
		return rndis_enabled ? UsbNetworkProtocol::RNDIS :
				       UsbNetworkProtocol::NONE;
	}
	if (ifname == "usb1") {
		return ncm_enabled && rndis_enabled ? UsbNetworkProtocol::NCM :
						      UsbNetworkProtocol::NONE;
	}
	if (ifname != "usb0") {
		return UsbNetworkProtocol::NONE;
	}
	if (ncm_enabled && rndis_enabled) {
		return UsbNetworkProtocol::RNDIS;
	}
	if (ncm_enabled) {
		return UsbNetworkProtocol::NCM;
	}
	return rndis_enabled ? UsbNetworkProtocol::RNDIS :
			       UsbNetworkProtocol::NONE;
}

inline void add_unique_ip(std::vector<std::string> &ips, const std::string &ip)
{
	if (ip.empty()) {
		return;
	}
	if (std::find(ips.begin(), ips.end(), ip) == ips.end()) {
		ips.push_back(ip);
	}
}

inline std::vector<std::string>
select_main_ip_candidates(const MainIpSelectionInputs &input)
{
	std::vector<std::string> ips;
	add_unique_ip(ips, input.wlan0_ip);
	return ips;
}

} // namespace lvgl_main
