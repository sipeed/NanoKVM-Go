#pragma once

#include "ui/region_page_policy.hpp"
#include "ui/ssid_prepared_raster.hpp"

#include <cstdint>
#include <functional>
#include <string>

namespace ui
{

struct PageManagerActions {
	struct Settings {
		std::function<void(bool)> set_mcp_enabled;
		std::function<void(bool)> set_replay_enabled;
		std::function<void(bool)> set_ncm_enabled;
		std::function<void(bool)> set_ssh_enabled;
		std::function<void(bool)> set_auto_rotation_enabled;
		std::function<void()> refresh_service_state;
	} settings;

	struct Wifi {
		std::function<void(bool)> page_visibility_changed;
		std::function<void()> scan;
		std::function<void(const std::string &, const std::string &)>
			connect;
		std::function<void()> start_qr_ap;
		std::function<uint64_t()> stop_qr_ap;
		std::function<void()> stop_qr_ap_after_success;
		std::function<void(bool)> set_enabled;
		std::function<void()> reset;
		std::function<void()> qr_dismissed;
	} wifi;

	struct SsidRaster {
		std::function<bool(SsidRasterRequest)> prepare;
		std::function<void(uint64_t)> invalidate_page;
	} ssid_raster;

	struct Panel {
		std::function<void(int)> set_brightness_percent;
		std::function<void(bool)> set_sleep_enabled;
	} panel;

	struct About {
		std::function<bool()> request_factory_reset;
	} about;

	struct Region {
		std::function<bool(RegionOption)> request_switch;
	} region;

	struct Apps {
		std::function<void(const std::string &)> launch;
	} apps;

	struct More {
		std::function<void(bool)> page_visibility_changed;
	} more;

	struct Tailscale {
		std::function<void(bool)> page_visibility_changed;
		std::function<bool()> start;
		std::function<bool()> exit;
	} tailscale;
};

}
