#pragma once

#include <functional>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

#include "lvgl_main/runtime_service.hpp"

namespace lvgl_main
{

enum class SimulatorRuntimeAction {
	SetSsh,
	SetNcm,
	SetWifi,
	SetHdmiInput,
	SetHid,
	RefreshLocalServices,
	SetMcpBundle,
	SetScreenTimelapse,
	SetBacklight,
	SetAutoSleep,
	SetAutoRotation,
	SetManualRotation,
	SetScreenSleep,
	ResetWifi,
	ScanWifi,
	ConnectWifi,
	StartWifiQr,
	StopWifiQr,
	FactoryReset,
	RegionSwitch,
};

struct SimulatorRuntimeTestConfig {
	std::optional<RuntimeStatus> initial_status;
	std::vector<SimulatorRuntimeAction> failing_actions;
};

class SimulatorRuntimeService final : public RuntimeService {
    public:
	explicit SimulatorRuntimeService(
		RuntimeCallbacks callbacks = {},
		SimulatorRuntimeTestConfig test_config = {});
	~SimulatorRuntimeService() override = default;

	SimulatorRuntimeService(const SimulatorRuntimeService &) = delete;
	SimulatorRuntimeService &
	operator=(const SimulatorRuntimeService &) = delete;

	bool start() override;
	void stop() override;
	bool is_running() const override;
	RuntimeStatus current_status() const override;

	void set_ssh_enabled(bool enabled) override;
	void set_ncm_enabled(bool enabled) override;
	void set_wifi_enabled(bool enabled) override;
	void set_hdmi_input_enabled(bool enabled) override;
	void set_hid_enabled(bool enabled) override;
	void refresh_local_service_state() override;
	void set_mcp_bundle_enabled(bool enabled) override;
	void set_screen_timelapse_enabled(bool enabled) override;
	void apply_backlight_percent_now(int percent) override;
	void set_auto_sleep_enabled(bool enabled) override;
	void set_ui_auto_rotation_enabled(bool enabled) override;
	void set_ui_manual_display_rotation(bool enabled) override;
	void set_screen_sleep(bool sleep) override;
	void reset_wifi() override;
	void scan_wifi_networks() override;
	void connect_wifi_network(std::string ssid,
				  std::string password = {}) override;
	void start_wifi_qr_ap_config() override;
	uint64_t stop_wifi_qr_ap_config() override;
	void stop_wifi_qr_ap_config_after_success() override;
	void notify_wifi_page_visible(bool visible) override;
	void notify_more_page_visible(bool visible) override;
	void notify_tailscale_page_visible(bool visible) override;
	void notify_wifi_qr_dismissed() override;
	std::string wifi_ap_qr_payload() const override;
	bool request_factory_reset() override;
	bool request_region_switch(DeviceRegion target) override;
	bool request_tailscale_start() override;
	bool request_tailscale_exit() override;

    private:
	using StatusMutation = std::function<void(RuntimeStatus &)>;

	bool action_fails(SimulatorRuntimeAction action) const;
	void mutate(SimulatorRuntimeAction action, StatusMutation mutation,
		    StatusMutation failure_mutation = {});
	void publish(const RuntimeStatusPublication &publication,
		     const RuntimeCallbacks &callbacks) const;
	void restore_connected_wifi(RuntimeStatus &status) const;

	mutable std::mutex _mutex;
	RuntimeStatus _status;
	uint64_t _status_revision { 0 };
	RuntimeCallbacks _callbacks;
	SimulatorRuntimeTestConfig _test_config;
	bool _ui_auto_rotation_enabled { false };
	bool _ui_manual_display_rotation { false };
	bool _running { false };
};

RuntimeStatus default_simulator_runtime_status();

}
