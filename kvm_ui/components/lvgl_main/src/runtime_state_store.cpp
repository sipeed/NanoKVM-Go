#include "lvgl_main/runtime_state_store.hpp"

#include <algorithm>
#include <chrono>
#include <utility>

namespace lvgl_main
{

RuntimeStateStore::RuntimeStateStore(RuntimeStatus initial_status)
	: _status(std::move(initial_status))
{
}

RuntimeStatus RuntimeStateStore::current_status() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _status;
}

RuntimeStatusPublication RuntimeStateStore::current_publication() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return { .status = _status, .revision = _revision };
}

void RuntimeStateStore::replace_initial_status(RuntimeStatus initial_status)
{
	std::lock_guard<std::mutex> lock(_mutex);
	_status = std::move(initial_status);
}

std::optional<RuntimeStatusPublication>
RuntimeStateStore::commit(const RuntimeStatusPatch &patch)
{
	std::lock_guard<std::mutex> lock(_mutex);
	const bool applied = std::visit(
		[this](const auto &typed_patch) {
			return apply(typed_patch);
		},
		patch);
	if (!applied) {
		return std::nullopt;
	}

	_status.updated_at = std::chrono::steady_clock::now();
	++_revision;
	return RuntimeStatusPublication { .status = _status,
					  .revision = _revision };
}

bool RuntimeStateStore::apply(const RefreshObservationPatch &patch)
{
	const RuntimeStatus &from = patch.status;
	const bool legacy_projection =
		patch.observed_domains == RefreshObservationDomain::All;
	const auto observed = [&patch](RefreshObservationDomain domain) {
		return has_refresh_observation(patch.observed_domains, domain);
	};
	if (observed(RefreshObservationDomain::NetworkAddresses)) {
		_status.eth_ip = from.eth_ip;
		_status.wifi_ip = from.wifi_ip;
		_status.ncm_ip = from.ncm_ip;
		_status.rndis_ip = from.rndis_ip;
		_status.active_ip = from.active_ip;
		_status.main_ip_candidates = from.main_ip_candidates;
	}
	if (observed(RefreshObservationDomain::AppVersion)) {
		_status.app_version = from.app_version;
	}
	if (observed(RefreshObservationDomain::HardwareVersion)) {
		_status.hw_version = from.hw_version;
	}
	if (observed(RefreshObservationDomain::Ncm)) {
		_status.ncm_supported = from.ncm_supported;
		_status.ncm_enabled = from.ncm_enabled;
	}
	if (observed(RefreshObservationDomain::RuntimeFlags)) {
		_status.auto_sleep_enabled = from.auto_sleep_enabled;
		if (legacy_projection) {
			_status.hdmi_input_supported = from.hdmi_input_supported;
			_status.hdmi_input_enabled = from.hdmi_input_enabled;
			_status.video_enabled = from.video_enabled;
			_status.hid_supported = from.hid_supported;
			_status.hid_enabled = from.hid_enabled;
			_status.hid_mode = from.hid_mode;
		}
	}
	if (observed(RefreshObservationDomain::Video)) {
		_status.hdmi_input_supported = from.hdmi_input_supported;
		_status.hdmi_input_enabled = from.hdmi_input_enabled;
		_status.video_enabled = from.video_enabled;
	}
	if (observed(RefreshObservationDomain::Hid)) {
		_status.hid_supported = from.hid_supported;
		_status.hid_enabled = from.hid_enabled;
		_status.hid_mode = from.hid_mode;
	}
	if (observed(RefreshObservationDomain::Backlight)) {
		_status.backlight_percent = from.backlight_percent;
		_status.screen_on = from.screen_on;
	}
	if (observed(RefreshObservationDomain::ServerAvailability)) {
		_status.server_available = from.server_available;
	}
	if (observed(RefreshObservationDomain::SystemInfoStatic)) {
		_status.device_key = from.device_key;
		_status.hostname = from.hostname;
		_status.device_number = from.device_number;
		_status.image_version = from.image_version;
		_status.edid_name = from.edid_name;
		_status.mac_address = from.mac_address;
		_status.mdns_name = from.mdns_name;
	}
	if (observed(RefreshObservationDomain::SystemInfoActive)) {
		_status.uptime_text = from.uptime_text;
		_status.local_time_text = from.local_time_text;
		_status.memory_text = from.memory_text;
		_status.temperature_text = from.temperature_text;
	}
	if (observed(RefreshObservationDomain::SystemInfoStorage)) {
		_status.storage_text = from.storage_text;
	}

	if (observed(RefreshObservationDomain::SystemInfoSignal)) {
		_status.wifi_signal = from.wifi_signal;
	}

	if (_pending_wifi_operation_generation == 0 &&
	    patch.wifi_operation_generation ==
		    _latest_wifi_operation_generation) {
		if (observed(RefreshObservationDomain::Wifi)) {

			copy_wifi_status_fields(from, false);
		}
	}

	return true;
}

bool RuntimeStateStore::apply(const WifiLifecyclePatch &patch)
{
	switch (patch.phase) {
	case WifiLifecyclePatch::Phase::Observation:
		if (_pending_wifi_operation_generation != 0) {
			return false;
		}
		break;
	case WifiLifecyclePatch::Phase::Pending:
		if (patch.operation_generation == 0 ||
		    patch.operation_generation <=
			    _latest_wifi_operation_generation) {
			return false;
		}
		_latest_wifi_operation_generation = patch.operation_generation;
		_pending_wifi_operation_generation = patch.operation_generation;
		break;
	case WifiLifecyclePatch::Phase::Terminal:
		if (patch.operation_generation == 0 ||
		    patch.operation_generation !=
			    _pending_wifi_operation_generation) {
			if (patch.preserve_ap_stop_completion_when_stale &&
			    patch.status.wifi_ap_stop_completed_generation >
				    _status.wifi_ap_stop_completed_generation) {
				_status.wifi_ap_stop_completed_generation =
					patch.status
						.wifi_ap_stop_completed_generation;
				return true;
			}
			return false;
		}
		_pending_wifi_operation_generation = 0;
		break;
	}

	copy_wifi_status_fields(patch.status);
	return true;
}

bool RuntimeStateStore::apply(const DisplayIngressPatch &patch)
{
	if (!patch.resolution.has_value() && !patch.fps.has_value() &&
	    !patch.fps_unit.has_value() &&
	    !patch.keyboard_special.has_value() &&
	    !patch.keyboard_normal.has_value()) {
		return false;
	}

	if (patch.resolution.has_value()) {
		_status.display_resolution_label = *patch.resolution;
		++_status.display_resolution_label_generation;
	}
	if (patch.fps.has_value()) {
		_status.display_fps_label = *patch.fps;
		++_status.display_fps_label_generation;
	}
	if (patch.fps_unit.has_value()) {
		_status.display_fps_unit_label = *patch.fps_unit;
		++_status.display_fps_unit_label_generation;
	}
	if (patch.keyboard_special.has_value()) {
		_status.hid_keyboard_special_label = *patch.keyboard_special;
		++_status.hid_keyboard_special_label_generation;
	}
	if (patch.keyboard_normal.has_value()) {
		_status.hid_keyboard_normal_label = *patch.keyboard_normal;
		++_status.hid_keyboard_normal_label_generation;
	}

	++_status.display_label_generation;
	if (patch.keyboard_special.has_value() ||
	    patch.keyboard_normal.has_value()) {
		++_status.hid_keyboard_label_generation;
	}
	return true;
}

bool RuntimeStateStore::apply(const HidIngressPatch &patch)
{
	if (!patch.mouse_active.has_value() &&
	    !patch.keyboard_special.has_value() &&
	    !patch.keyboard_normal.has_value()) {
		return false;
	}

	if (patch.mouse_active.has_value()) {
		_status.hid_mouse_active = *patch.mouse_active;
		++_status.hid_mouse_state_generation;
		if (*patch.mouse_active) {
			++_status.hid_mouse_active_generation;
		} else {
			++_status.hid_mouse_clear_generation;
		}
	}
	if (patch.keyboard_special.has_value()) {
		_status.hid_keyboard_special_label = *patch.keyboard_special;
		++_status.hid_keyboard_special_label_generation;
	}
	if (patch.keyboard_normal.has_value()) {
		_status.hid_keyboard_normal_label = *patch.keyboard_normal;
		++_status.hid_keyboard_normal_label_generation;
	}
	if (patch.keyboard_special.has_value() ||
	    patch.keyboard_normal.has_value()) {
		++_status.hid_keyboard_label_generation;
	}
	return true;
}

bool RuntimeStateStore::apply(const SystemdStatusPatch &patch)
{
	_status.ssh_supported = patch.status.ssh_supported;
	_status.ssh_enabled = patch.status.ssh_enabled;
	return true;
}

bool RuntimeStateStore::apply(const LocalServicesPatch &patch)
{
	_status.local_services_known = patch.status.local_services_known;
	_status.local_services_refresh_ok =
		patch.status.local_services_refresh_ok;
	_status.local_services_event_seq =
		patch.status.local_services_event_seq;
	_status.local_services_error = patch.status.local_services_error;
	_status.mcp_service_supported = patch.status.mcp_service_supported;
	_status.mcp_service_active = patch.status.mcp_service_active;
	_status.replay_service_supported =
		patch.status.replay_service_supported;
	_status.replay_service_active = patch.status.replay_service_active;
	return true;
}

bool RuntimeStateStore::apply(const DisplayMetricsPatch &patch)
{
	_status.resolution = patch.status.resolution;
	_status.fps = patch.status.fps;
	_status.h264_bitrate_kbps = patch.status.h264_bitrate_kbps;
	_status.server_available = patch.status.server_available;
	return true;
}

bool RuntimeStateStore::apply(const RuntimeFlagsPatch &patch)
{
	_status.auto_sleep_enabled = patch.status.auto_sleep_enabled;
	_status.ncm_supported = patch.status.ncm_supported;
	_status.ncm_enabled = patch.status.ncm_enabled;
	_status.hdmi_input_supported = patch.status.hdmi_input_supported;
	_status.hdmi_input_enabled = patch.status.hdmi_input_enabled;
	_status.video_enabled = patch.status.video_enabled;
	_status.hid_supported = patch.status.hid_supported;
	_status.hid_enabled = patch.status.hid_enabled;
	_status.hid_mode = patch.status.hid_mode;
	_status.backlight_percent = patch.status.backlight_percent;
	_status.screen_on = patch.status.screen_on;
	return true;
}

bool RuntimeStateStore::apply(const TailscalePatch &patch)
{
	const RuntimeStatus &from = patch.status;
	if (from.tailscale_seq <= _status.tailscale_seq) {
		return false;
	}
	_status.tailscale_online = from.tailscale_online;
	_status.tailscale_code = from.tailscale_code;
	_status.tailscale_peer_name = from.tailscale_peer_name;
	_status.tailscale_peer_ip = from.tailscale_peer_ip;
	_status.tailscale_seq = from.tailscale_seq;
	return true;
}

bool RuntimeStateStore::apply(const TailscaleSnapshotPatch &patch)
{
	const RuntimeStatus &from = patch.status;
	if (from.tailscale_action_generation <
	    _status.tailscale_action_generation) {
		return false;
	}
	if (!patch.snapshot_valid) {
		if (from.tailscale_seq <= _status.tailscale_seq) {
			return false;
		}
		_status.tailscale_online = false;
		_status.tailscale_seq = from.tailscale_seq;
		return true;
	}
	if (from.tailscale_gate_seq <= _status.tailscale_gate_seq) {
		return false;
	}
	_status.tailscale_online = from.tailscale_online;
	_status.tailscale_started = from.tailscale_started;
	_status.tailscale_matched = from.tailscale_matched;
	_status.tailscale_peer_name = from.tailscale_peer_name;
	_status.tailscale_peer_ip = from.tailscale_peer_ip;
	_status.tailscale_gate_seq = from.tailscale_gate_seq;
	_status.tailscale_action_generation = from.tailscale_action_generation;
	return true;
}

bool RuntimeStateStore::apply(const TailscaleActionResultPatch &patch)
{
	const RuntimeStatus &from = patch.status;
	if (from.tailscale_action_seq <= _status.tailscale_action_seq) {
		return false;
	}
	_status.tailscale_action_seq = from.tailscale_action_seq;
	_status.tailscale_action_kind = from.tailscale_action_kind;
	_status.tailscale_action_ok = from.tailscale_action_ok;
	_status.tailscale_action_message = from.tailscale_action_message;
	_status.tailscale_action_generation = from.tailscale_action_generation;
	return true;
}

bool RuntimeStateStore::apply(const FactoryResetPatch &patch)
{
	if (patch.failure_generation <=
	    _status.factory_reset_failure_generation) {
		return false;
	}
	_status.factory_reset_failure_generation = patch.failure_generation;
	return true;
}

bool RuntimeStateStore::apply(const RegionSwitchPatch &patch)
{
	if (patch.failure_generation <=
	    _status.region_switch_failure_generation) {
		return false;
	}
	_status.region_switch_failure_generation = patch.failure_generation;
	return true;
}

void RuntimeStateStore::copy_wifi_status_fields(const RuntimeStatus &from,
								bool include_wifi_ip)
{
	if (include_wifi_ip) {
		_status.wifi_ip = from.wifi_ip;
	}
	_status.wifi_supported = from.wifi_supported;
	_status.wifi_connected = from.wifi_connected;
	_status.wifi_ap_mode = from.wifi_ap_mode;
	_status.wifi_ap_client_connected = from.wifi_ap_client_connected;
	_status.wifi_qr_payload = from.wifi_qr_payload;
	_status.wifi_web_url = from.wifi_web_url;
	_status.wifi_runtime_state = from.wifi_runtime_state;
	_status.wifi_ssid = from.wifi_ssid;
	_status.wifi_saved_network_config = from.wifi_saved_network_config;
	_status.wifi_status_text = from.wifi_status_text;
	_status.wifi_error = from.wifi_error;
	_status.wifi_busy = from.wifi_busy;
	_status.wifi_boot_reconnect_running = from.wifi_boot_reconnect_running;
	_status.wifi_show_qr = from.wifi_show_qr;
	_status.wifi_ap_stop_completed_generation =
		std::max(_status.wifi_ap_stop_completed_generation,
			 from.wifi_ap_stop_completed_generation);
	_status.wifi_scan_running = from.wifi_scan_running;
	_status.wifi_scan_result = from.wifi_scan_result;
	_status.wifi_connect_running = from.wifi_connect_running;
	_status.wifi_connect_result = from.wifi_connect_result;
	_status.wifi_connect_message = from.wifi_connect_message;
}

}
