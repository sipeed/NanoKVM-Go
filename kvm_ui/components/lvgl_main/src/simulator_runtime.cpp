#include "lvgl_main/simulator_runtime.hpp"

#include <algorithm>
#include <utility>

#include "lvgl_main/build_info.hpp"

namespace lvgl_main
{

namespace
{

constexpr int kBacklightMinPercent = 20;
constexpr int kBacklightMaxPercent = 100;
constexpr std::string_view kSimulatorSsid = "NanoKVM-Simulator";
constexpr std::string_view kSimulatorIp = "192.0.2.10";
constexpr std::string_view kSimulatorQrPayload =
	"https://simulator.invalid/wifi";

void stamp(RuntimeStatus &status)
{
	status.updated_at = std::chrono::steady_clock::now();
}

}

RuntimeStatus default_simulator_runtime_status()
{
	RuntimeStatus status;
	status.wifi_supported = true;
	status.wifi_connected = true;
	status.wifi_runtime_state = WifiRuntimeState::Connected;
	status.wifi_ssid = std::string(kSimulatorSsid);
	status.wifi_saved_network_config = true;
	status.wifi_status_text = "Connected (simulated)";
	status.wifi_ip = std::string(kSimulatorIp);
	status.active_ip = std::string(kSimulatorIp);
	status.main_ip_candidates = { status.wifi_ip };
	status.ssh_supported = true;
	status.ncm_supported = true;
	status.hdmi_input_supported = true;
	status.hdmi_input_enabled = true;
	status.video_enabled = true;
	status.hid_supported = true;
	status.hid_enabled = true;
	status.hid_mode = "simulated";
	status.local_services_known = true;
	status.local_services_refresh_ok = true;
	status.mcp_service_supported = true;
	status.replay_service_supported = true;
	status.backlight_percent = 70;
	status.auto_sleep_enabled = false;
	status.screen_on = true;
	status.resolution = "270x240";
	status.display_resolution_label = "270x240";
	status.display_fps_label = "Simulator";
	status.display_fps_unit_label = "FPS";
	status.app_version = build_info::kAppVersion;
	status.hw_version = "x86_64 Simulator";
	status.server_available = false;
	stamp(status);
	return status;
}

SimulatorRuntimeService::SimulatorRuntimeService(
	RuntimeCallbacks callbacks, SimulatorRuntimeTestConfig test_config)
	: _status(test_config.initial_status.value_or(
		  default_simulator_runtime_status()))
	, _callbacks(std::move(callbacks))
	, _test_config(std::move(test_config))
{
}

bool SimulatorRuntimeService::start()
{
	RuntimeStatusPublication publication;
	RuntimeCallbacks callbacks;
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (_running) {
			return true;
		}
		_running = true;
		stamp(_status);
		publication = { .status = _status,
				.revision = ++_status_revision };
		callbacks = _callbacks;
	}
	publish(publication, callbacks);
	return true;
}

void SimulatorRuntimeService::stop()
{
	std::lock_guard<std::mutex> lock(_mutex);
	_running = false;
	_callbacks = {};
}

bool SimulatorRuntimeService::is_running() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _running;
}

RuntimeStatus SimulatorRuntimeService::current_status() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _status;
}

void SimulatorRuntimeService::set_ssh_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetSsh,
	       [enabled](RuntimeStatus &status) {
		       status.ssh_enabled = enabled;
	       });
}

void SimulatorRuntimeService::set_ncm_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetNcm,
	       [enabled](RuntimeStatus &status) {
		       status.ncm_enabled = enabled;
	       });
}

void SimulatorRuntimeService::set_wifi_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetWifi, [this, enabled](
							RuntimeStatus &status) {
		if (enabled) {
			restore_connected_wifi(status);
			return;
		}
		status.wifi_connected = false;
		status.wifi_ap_mode = false;
		status.wifi_ap_client_connected = false;
		status.wifi_show_qr = false;
		status.wifi_qr_payload.clear();
		status.wifi_connect_running = false;
		status.wifi_connect_result = {};
		status.wifi_connect_message.clear();
		status.wifi_runtime_state = WifiRuntimeState::Disconnected;
		status.wifi_ssid.clear();
		status.wifi_ip.clear();
		status.wifi_status_text = "Disabled (simulated)";
	});
}

void SimulatorRuntimeService::set_hdmi_input_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetHdmiInput,
	       [enabled](RuntimeStatus &status) {
		       status.hdmi_input_enabled = enabled;
	       });
}

void SimulatorRuntimeService::set_hid_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetHid,
	       [enabled](RuntimeStatus &status) {
		       status.hid_enabled = enabled;
	       });
}

void SimulatorRuntimeService::refresh_local_service_state()
{
	mutate(SimulatorRuntimeAction::RefreshLocalServices,
	       [](RuntimeStatus &status) {
		       status.local_services_known = true;
		       status.local_services_refresh_ok = true;
		       status.local_services_error.clear();
		       ++status.local_services_event_seq;
	       });
}

void SimulatorRuntimeService::set_mcp_bundle_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetMcpBundle,
	       [enabled](RuntimeStatus &status) {
		       status.mcp_service_active = enabled;
		       ++status.local_services_event_seq;
	       });
}

void SimulatorRuntimeService::set_screen_timelapse_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetScreenTimelapse,
	       [enabled](RuntimeStatus &status) {
		       status.replay_service_active = enabled;
		       ++status.local_services_event_seq;
	       });
}

void SimulatorRuntimeService::apply_backlight_percent_now(int percent)
{
	mutate(SimulatorRuntimeAction::SetBacklight,
	       [percent](RuntimeStatus &status) {
		       status.backlight_percent =
			       std::clamp(percent, kBacklightMinPercent,
					  kBacklightMaxPercent);
	       });
}

void SimulatorRuntimeService::set_auto_sleep_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetAutoSleep,
	       [enabled](RuntimeStatus &status) {
		       status.auto_sleep_enabled = enabled;
	       });
}

void SimulatorRuntimeService::set_ui_auto_rotation_enabled(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetAutoRotation,
	       [this, enabled](RuntimeStatus &) {
		       _ui_auto_rotation_enabled = enabled;
	       });
}

void SimulatorRuntimeService::set_ui_manual_display_rotation(bool enabled)
{
	mutate(SimulatorRuntimeAction::SetManualRotation,
	       [this, enabled](RuntimeStatus &) {
		       _ui_manual_display_rotation = enabled;
	       });
}

void SimulatorRuntimeService::set_screen_sleep(bool sleep)
{
	mutate(SimulatorRuntimeAction::SetScreenSleep,
	       [sleep](RuntimeStatus &status) {
		       status.screen_on = !sleep;
	       });
}

void SimulatorRuntimeService::reset_wifi()
{
	mutate(SimulatorRuntimeAction::ResetWifi, [](RuntimeStatus &status) {
		status.wifi_connected = false;
		status.wifi_ap_mode = false;
		status.wifi_ap_client_connected = false;
		status.wifi_show_qr = false;
		status.wifi_qr_payload.clear();
		status.wifi_connect_running = false;
		status.wifi_connect_result = {};
		status.wifi_connect_message.clear();
		status.wifi_runtime_state = WifiRuntimeState::Disconnected;
		status.wifi_ssid.clear();
		status.wifi_saved_network_config = false;
		status.wifi_ip.clear();
		status.wifi_status_text = "Reset complete (simulated)";
	});
}

void SimulatorRuntimeService::scan_wifi_networks()
{
	mutate(SimulatorRuntimeAction::ScanWifi, [](RuntimeStatus &status) {
		status.wifi_scan_running = true;
	});
	mutate(SimulatorRuntimeAction::ScanWifi, [](RuntimeStatus &status) {
		status.wifi_scan_running = false;
		status.wifi_scan_result.ok = true;
		status.wifi_scan_result.error.clear();
		status.wifi_scan_result.networks = {
			{ .ssid = "NanoKVM-Simulator",
			  .signal = -30,
			  .frequency = 2412,
			  .security = "WPA2" },
			{ .ssid = "Lab-Guest",
			  .signal = -55,
			  .frequency = 5180,
			  .security = "Open" },
		};
	});
}

void SimulatorRuntimeService::connect_wifi_network(std::string ssid,
						   std::string password)
{
	(void)password;
	const std::string requested_ssid = std::move(ssid);
	mutate(SimulatorRuntimeAction::ConnectWifi,
	       [requested_ssid](RuntimeStatus &status) {
		       status.wifi_connect_running = true;
		       status.wifi_connect_message.clear();
		       status.wifi_connect_result = {};
		       status.wifi_connect_result.ssid = requested_ssid;
	       });
	mutate(SimulatorRuntimeAction::ConnectWifi,
	       [this, requested_ssid](RuntimeStatus &status) {
		       status.wifi_connect_running = false;
		       status.wifi_connect_result.ssid = requested_ssid;
		       if (requested_ssid.empty()) {
			       status.wifi_connect_result.status =
				       WifiConnectStatus::Failure;
			       status.wifi_connect_result.error_code =
				       "missing-ssid";
			       status.wifi_connect_message = "SSID is required";
			       return;
		       }
		       restore_connected_wifi(status);
		       status.wifi_ssid = requested_ssid;
		       status.wifi_saved_network_config = true;
		       status.wifi_connect_result.status =
			       WifiConnectStatus::Success;
		       status.wifi_connect_message = "Connected (simulated)";
	       });
}

void SimulatorRuntimeService::start_wifi_qr_ap_config()
{
	mutate(SimulatorRuntimeAction::StartWifiQr, [](RuntimeStatus &status) {
		status.wifi_connected = false;
		status.wifi_ap_mode = true;
		status.wifi_show_qr = true;
		status.wifi_runtime_state = WifiRuntimeState::ApAwaitClient;
		status.wifi_qr_payload = std::string(kSimulatorQrPayload);
		status.wifi_status_text = "AP configuration (simulated)";
	});
}

uint64_t SimulatorRuntimeService::stop_wifi_qr_ap_config()
{
	uint64_t generation = 0;
	mutate(SimulatorRuntimeAction::StopWifiQr,
	       [this, &generation](RuntimeStatus &status) {
		       ++status.wifi_ap_stop_completed_generation;
		       generation = status.wifi_ap_stop_completed_generation;
		       restore_connected_wifi(status);
	       });
	return generation;
}

void SimulatorRuntimeService::stop_wifi_qr_ap_config_after_success()
{
	(void)stop_wifi_qr_ap_config();
}

void SimulatorRuntimeService::notify_wifi_page_visible(bool)
{
}

void SimulatorRuntimeService::notify_more_page_visible(bool)
{

}

void SimulatorRuntimeService::notify_tailscale_page_visible(bool)
{

}

void SimulatorRuntimeService::notify_wifi_qr_dismissed()
{
}

std::string SimulatorRuntimeService::wifi_ap_qr_payload() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _status.wifi_qr_payload;
}

bool SimulatorRuntimeService::request_factory_reset()
{
	mutate(SimulatorRuntimeAction::FactoryReset, [](RuntimeStatus &status) {
		++status.factory_reset_failure_generation;
	});
	return !action_fails(SimulatorRuntimeAction::FactoryReset);
}

bool SimulatorRuntimeService::request_region_switch(DeviceRegion target)
{
	mutate(
		SimulatorRuntimeAction::RegionSwitch,
		[target](RuntimeStatus &status) {
			status.region = target;
		},
		[](RuntimeStatus &status) {
			++status.region_switch_failure_generation;
		});
	return !action_fails(SimulatorRuntimeAction::RegionSwitch);
}

bool SimulatorRuntimeService::request_tailscale_start()
{
	return false;
}

bool SimulatorRuntimeService::request_tailscale_exit()
{
	return false;
}

bool SimulatorRuntimeService::action_fails(SimulatorRuntimeAction action) const
{
	return std::find(_test_config.failing_actions.begin(),
			 _test_config.failing_actions.end(),
			 action) != _test_config.failing_actions.end();
}

void SimulatorRuntimeService::mutate(SimulatorRuntimeAction action,
				     StatusMutation mutation,
				     StatusMutation failure_mutation)
{
	RuntimeStatusPublication publication;
	RuntimeCallbacks callbacks;
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (!_running) {
			return;
		}
		if (action_fails(action)) {
			if (!failure_mutation) {
				return;
			}
			failure_mutation(_status);
		} else {
			mutation(_status);
		}
		stamp(_status);
		publication = { .status = _status,
				.revision = ++_status_revision };
		callbacks = _callbacks;
	}
	publish(publication, callbacks);
}

void SimulatorRuntimeService::publish(
	const RuntimeStatusPublication &publication,
	const RuntimeCallbacks &callbacks) const
{
	if (!callbacks.on_status) {
		return;
	}
	try {
		callbacks.on_status(publication);
	} catch (...) {
	}
}

void SimulatorRuntimeService::restore_connected_wifi(RuntimeStatus &status) const
{
	status.wifi_supported = true;
	status.wifi_connected = true;
	status.wifi_ap_mode = false;
	status.wifi_ap_client_connected = false;
	status.wifi_show_qr = false;
	status.wifi_qr_payload.clear();
	status.wifi_runtime_state = WifiRuntimeState::Connected;
	status.wifi_ssid = std::string(kSimulatorSsid);
	status.wifi_saved_network_config = true;
	status.wifi_ip = std::string(kSimulatorIp);
	status.active_ip = status.wifi_ip;
	status.main_ip_candidates = { status.wifi_ip };
	status.wifi_status_text = "Connected (simulated)";
}

}
