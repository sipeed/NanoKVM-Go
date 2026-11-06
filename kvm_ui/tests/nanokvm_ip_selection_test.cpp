#include <vector>
#include <string>
#include <utility>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/nanokvm_ip_selection.hpp"

using namespace lvgl_main;

namespace
{

std::vector<std::string> select_ips(bool ncm_enabled, bool rndis_enabled,
				    std::string ncm_ip, std::string rndis_ip,
				    std::string wlan0_ip)
{
	return select_main_ip_candidates({ .ncm_enabled = ncm_enabled,
					   .rndis_enabled = rndis_enabled,
					   .ncm_ip = std::move(ncm_ip),
					   .rndis_ip = std::move(rndis_ip),
					   .wlan0_ip = std::move(wlan0_ip) });
}

} // namespace

TEST_CASE("NanoKVM main IP candidates only use wlan0")
{
	CHECK(select_ips(false, false, "", "", "192.168.1.50") ==
	      std::vector<std::string> { "192.168.1.50" });
	CHECK(select_ips(false, false, "", "", "") ==
	      std::vector<std::string> {});
	CHECK(select_ips(true, false, "10.0.0.2", "", "") ==
	      std::vector<std::string> {});
	CHECK(select_ips(false, true, "", "172.16.0.2", "") ==
	      std::vector<std::string> {});
	CHECK(select_ips(true, true, "10.0.0.2", "172.16.0.2", "") ==
	      std::vector<std::string> {});
	CHECK(select_ips(true, true, "10.0.0.2", "172.16.0.2",
			 "192.168.1.50") ==
	      std::vector<std::string> { "192.168.1.50" });
	CHECK(select_ips(false, true, "", "172.16.0.2", "192.168.1.50") ==
	      std::vector<std::string> { "192.168.1.50" });
	CHECK(select_ips(true, true, "", "172.16.0.2", "192.168.1.50") ==
	      std::vector<std::string> { "192.168.1.50" });
}

TEST_CASE("NanoKVM USB interface names follow protocol flag layout")
{
	CHECK(classify_usb_network_interface("usb0", true, true) ==
	      UsbNetworkProtocol::RNDIS);
	CHECK(classify_usb_network_interface("usb1", true, true) ==
	      UsbNetworkProtocol::NCM);
	CHECK(classify_usb_network_interface("usb0", true, false) ==
	      UsbNetworkProtocol::NCM);
	CHECK(classify_usb_network_interface("usb0", false, true) ==
	      UsbNetworkProtocol::RNDIS);
	CHECK(classify_usb_network_interface("usb1", true, false) ==
	      UsbNetworkProtocol::NONE);
	CHECK(classify_usb_network_interface("wlan0", true, true) ==
	      UsbNetworkProtocol::NONE);
}
