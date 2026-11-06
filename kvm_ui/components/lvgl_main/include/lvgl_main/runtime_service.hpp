#pragma once

#include <chrono>
#include <cstdint>
#include <functional>
#include <optional>
#include <string>
#include <vector>

#include "lvgl_main/device_region.hpp"
#include "lvgl_main/wifi_runtime_state.hpp"
#include "lvgl_main/wifi_script_parser.hpp"
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include "ui/background_assets.hpp"
#endif

namespace lvgl_main
{

enum class TailscaleActionKind : uint8_t {
	None,
	Start,
	Exit,
};

struct RuntimeStatus {
	std::string eth_ip;
	std::string wifi_ip;
	std::string ncm_ip;
	std::string rndis_ip;
	std::string active_ip;
	std::vector<std::string> main_ip_candidates;
	bool wifi_supported { false };
	bool wifi_connected { false };
	bool wifi_ap_mode { false };
	bool wifi_ap_client_connected { false };
	std::string wifi_qr_payload;
		std::string wifi_web_url;
		WifiRuntimeState wifi_runtime_state { WifiRuntimeState::Unknown };
		std::string wifi_ssid;
		bool wifi_saved_network_config { false };
		std::string wifi_status_text;
	std::string wifi_error;
	bool wifi_busy { false };
	bool wifi_boot_reconnect_running { false };
	bool wifi_show_qr { false };
	uint64_t wifi_ap_stop_completed_generation { 0 };
	bool wifi_scan_running { false };
	WifiScanResult wifi_scan_result;
	bool wifi_connect_running { false };
	WifiConnectResult wifi_connect_result;
	std::string wifi_connect_message;

	bool ssh_supported { false };
	bool ssh_enabled { false };
	bool ncm_supported { false };
	bool ncm_enabled { false };

	std::string resolution;
	std::optional<double> fps;
	std::optional<int> h264_bitrate_kbps;
	std::string display_resolution_label;
	std::string display_fps_label;
	std::string display_fps_unit_label;
	uint64_t display_label_generation { 0 };
	uint64_t display_resolution_label_generation { 0 };
	uint64_t display_fps_label_generation { 0 };
	uint64_t display_fps_unit_label_generation { 0 };
	bool hdmi_input_supported { false };
	bool hdmi_input_enabled { false };
	bool video_enabled { false };
	bool hid_supported { false };
	bool hid_enabled { false };
	std::string hid_mode;
	std::string hid_keyboard_special_label;
	std::string hid_keyboard_normal_label;
	uint64_t hid_keyboard_label_generation { 0 };
	uint64_t hid_keyboard_special_label_generation { 0 };
	uint64_t hid_keyboard_normal_label_generation { 0 };
	bool hid_mouse_active { false };
	uint64_t hid_mouse_state_generation { 0 };
	uint64_t hid_mouse_active_generation { 0 };
	uint64_t hid_mouse_clear_generation { 0 };

	bool local_services_known { false };
	bool local_services_refresh_ok { false };
	uint64_t local_services_event_seq { 0 };
	std::string local_services_error;
	bool mcp_service_supported { false };
	bool mcp_service_active { false };
	bool replay_service_supported { false };
	bool replay_service_active { false };

	std::optional<int> backlight_percent;
	bool auto_sleep_enabled { false };
	bool screen_on { true };

	std::string app_version;
	std::string hw_version;

	bool tailscale_online { false };
	std::string tailscale_code;
	std::string tailscale_peer_name;
	std::string tailscale_peer_ip;
	uint64_t tailscale_seq { 0 };
	bool tailscale_started { false };
	bool tailscale_matched { false };
	uint64_t tailscale_gate_seq { 0 };
	uint64_t tailscale_action_seq { 0 };
	TailscaleActionKind tailscale_action_kind { TailscaleActionKind::None };
	bool tailscale_action_ok { false };
	std::string tailscale_action_message;
	uint64_t tailscale_action_generation { 0 };

	DeviceRegion region { DeviceRegion::Global };

	std::string device_key;
	std::string hostname;
	std::string device_number;
	std::string image_version;
	std::string edid_name;
	std::string mac_address;
	std::string mdns_name;

	std::string wifi_signal;
	std::string uptime_text;
	std::string local_time_text;
	std::string memory_text;
	std::string storage_text;
	std::string temperature_text;
	uint64_t factory_reset_failure_generation { 0 };
	uint64_t region_switch_failure_generation { 0 };
	bool server_available { false };
	std::chrono::steady_clock::time_point updated_at {
		std::chrono::steady_clock::now()
	};
};

struct RuntimeStatusPublication {
	RuntimeStatus status;
	uint64_t revision { 0 };
};

struct RuntimeCallbacks {
	std::function<void(const RuntimeStatusPublication &)> on_status;
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	std::function<void(ui::BackgroundAssets)> on_background_assets;
#endif
};

class RuntimeService {
    public:
	virtual ~RuntimeService() = default;

	RuntimeService(const RuntimeService &) = delete;
	RuntimeService &operator=(const RuntimeService &) = delete;

	virtual bool start() = 0;
	virtual void stop() = 0;
	virtual bool is_running() const = 0;
	virtual RuntimeStatus current_status() const = 0;

	virtual void set_ssh_enabled(bool enabled) = 0;
	virtual void set_ncm_enabled(bool enabled) = 0;
	virtual void set_wifi_enabled(bool enabled) = 0;
	virtual void set_hdmi_input_enabled(bool enabled) = 0;
	virtual void set_hid_enabled(bool enabled) = 0;
	virtual void refresh_local_service_state() = 0;
	virtual void set_mcp_bundle_enabled(bool enabled) = 0;
	virtual void set_screen_timelapse_enabled(bool enabled) = 0;
	virtual void apply_backlight_percent_now(int percent) = 0;
	virtual void set_auto_sleep_enabled(bool enabled) = 0;
	virtual void set_ui_auto_rotation_enabled(bool enabled) = 0;
	virtual void set_ui_manual_display_rotation(bool enabled) = 0;
	virtual void set_screen_sleep(bool sleep) = 0;
	virtual void reset_wifi() = 0;
	virtual void scan_wifi_networks() = 0;
	virtual void connect_wifi_network(std::string ssid,
					  std::string password = {}) = 0;
	virtual void start_wifi_qr_ap_config() = 0;
	virtual uint64_t stop_wifi_qr_ap_config() = 0;
	virtual void stop_wifi_qr_ap_config_after_success() = 0;
	virtual void notify_wifi_page_visible(bool visible) = 0;
	virtual void notify_more_page_visible(bool visible) = 0;
	virtual void notify_tailscale_page_visible(bool visible) = 0;
	virtual void notify_wifi_qr_dismissed() = 0;
	virtual std::string wifi_ap_qr_payload() const = 0;
	virtual bool request_factory_reset() = 0;
	virtual bool request_region_switch(DeviceRegion target) = 0;
	virtual bool request_tailscale_start() = 0;
	virtual bool request_tailscale_exit() = 0;

    protected:
	RuntimeService() = default;
};

}
