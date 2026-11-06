#include <algorithm>
#include <atomic>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <iomanip>
#include <memory>
#include <mutex>
#include <optional>
#include <sstream>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include <lvgl.h>
#include <spdlog/spdlog.h>
#include <src/misc/lv_timer.h>

#include "lvgl_main/app_catalog.hpp"
#include "lvgl_main/build_info.hpp"
#include "lvgl_ui_internal.hpp"
#include "tailscale_projection.hpp"
#include "lvgl_main/simulator_policy.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"
#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_runtime.hpp"
#else
#include "lvgl_main/nanokvm_runtime.hpp"
#endif
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include "ui/background_assets.hpp"
#endif
#include "ui/page_manager.hpp"
#include "ui/wifi_page.hpp"

namespace
{

constexpr std::chrono::milliseconds kAutoSleepIdleTimeout { 30000 };

ui::WifiPageVisualState
to_wifi_page_visual_state(lvgl_main::WifiRuntimeState state)
{
	switch (state) {
	case lvgl_main::WifiRuntimeState::Connected:
		return ui::WifiPageVisualState::Connected;
	case lvgl_main::WifiRuntimeState::CacheConnecting:
	case lvgl_main::WifiRuntimeState::WaitingExternalConnection:
		return ui::WifiPageVisualState::Connecting;
	case lvgl_main::WifiRuntimeState::ApAwaitClient:
		return ui::WifiPageVisualState::ApConnect;
	case lvgl_main::WifiRuntimeState::ApAwaitWeb:
		return ui::WifiPageVisualState::WebConfigure;
	case lvgl_main::WifiRuntimeState::Error:
		return ui::WifiPageVisualState::Error;
	case lvgl_main::WifiRuntimeState::Unknown:
	case lvgl_main::WifiRuntimeState::Disconnected:
		return ui::WifiPageVisualState::Off;
	}
	return ui::WifiPageVisualState::Off;
}

const char *wifi_runtime_state_name(lvgl_main::WifiRuntimeState state)
{
	switch (state) {
	case lvgl_main::WifiRuntimeState::Unknown:
		return "Unknown";
	case lvgl_main::WifiRuntimeState::Connected:
		return "Connected";
	case lvgl_main::WifiRuntimeState::Disconnected:
		return "Disconnected";
	case lvgl_main::WifiRuntimeState::CacheConnecting:
		return "CacheConnecting";
	case lvgl_main::WifiRuntimeState::ApAwaitClient:
		return "ApAwaitClient";
	case lvgl_main::WifiRuntimeState::ApAwaitWeb:
		return "ApAwaitWeb";
	case lvgl_main::WifiRuntimeState::WaitingExternalConnection:
		return "WaitingExternalConnection";
	case lvgl_main::WifiRuntimeState::Error:
		return "Error";
	}
	return "Unknown";
}

const char *wifi_visual_state_name(ui::WifiPageVisualState state)
{
	switch (state) {
	case ui::WifiPageVisualState::Off:
		return "Off";
	case ui::WifiPageVisualState::Connected:
		return "Connected";
	case ui::WifiPageVisualState::Connecting:
		return "Connecting";
	case ui::WifiPageVisualState::ApConnect:
		return "ApConnect";
	case ui::WifiPageVisualState::WebConfigure:
		return "WebConfigure";
	case ui::WifiPageVisualState::Error:
		return "Error";
	}
	return "Unknown";
}

std::vector<ui::WifiScanResult>
to_ui_wifi_scan_results(const lvgl_main::WifiScanResult &result)
{
	std::vector<ui::WifiScanResult> ui_results;
	ui_results.reserve(result.networks.size());
	for (const auto &network : result.networks) {
		ui_results.push_back(ui::WifiScanResult { .ssid = network.ssid,
							  .eap = !network.supported,
							  .signal = network.signal });
	}
	return ui_results;
}

std::string wifi_scan_error_message(const lvgl_main::WifiScanResult &result)
{
	return result.error.empty() ? "Scan failed" : result.error;
}

std::string wifi_connect_status_message(const lvgl_main::RuntimeStatus &status)
{
	if (!status.wifi_connect_message.empty()) {
		return status.wifi_connect_message;
	}
	const auto &result = status.wifi_connect_result;
	if (result.status == lvgl_main::WifiConnectStatus::Success) {
		return result.ssid.empty() ? "Connected" :
					     "Connected to " + result.ssid;
	}
	if (!result.error_code.empty()) {
		return result.error_code;
	}
	return result.raw_summary.empty() ? "Connect failed" :
					    result.raw_summary;
}

std::string wifi_qr_payload_for_status(const lvgl_main::RuntimeStatus &status)
{
	if (status.wifi_runtime_state ==
		    lvgl_main::WifiRuntimeState::ApAwaitWeb &&
	    !status.wifi_web_url.empty()) {
		return status.wifi_web_url;
	}
	return status.wifi_qr_payload;
}

bool has_valid_fps(double fps)
{
	return std::isfinite(fps) && fps > 0.0;
}

}

void UILVGLApp::schedule_runtime_status_apply(
	const lvgl_main::RuntimeStatusPublication &publication)
{
	std::lock_guard<std::mutex> lock(_runtime_status_mutex);
	if (!_runtime_status_mailbox.offer(publication)) {
		return;
	}
	if (_runtime_status_apply_pending) {
		return;
	}

	_runtime_status_apply_pending = true;
	if (lv_async_call(runtime_status_apply_async_cb, this) !=
	    LV_RESULT_OK) {
		_runtime_status_apply_pending = false;
		spdlog::warn("Failed to schedule runtime status UI apply");
	}
}

void UILVGLApp::apply_pending_runtime_status()
{
	std::optional<lvgl_main::RuntimeStatusPublication> publication;
	{
		std::lock_guard<std::mutex> lock(_runtime_status_mutex);
		_runtime_status_apply_pending = false;
		publication = _runtime_status_mailbox.take_next();
		if (!publication.has_value()) {
			return;
		}
	}

	LvLockGuard lock;
	apply_runtime_status(publication->status);
}

void UILVGLApp::runtime_status_apply_async_cb(void *user_data)
{
	auto *self = static_cast<UILVGLApp *>(user_data);
	if (self == nullptr) {
		return;
	}

	self->apply_pending_runtime_status();
}

void UILVGLApp::schedule_ssid_raster_result(
	ui::SsidPreparedRasterResult result)
{
	(void)_ssid_raster_result_mailbox.offer(std::move(result));
}

void UILVGLApp::apply_pending_ssid_raster_result()
{
	if (should_defer_ssid_raster_install()) {
		return;
	}
	std::optional<ui::SsidPreparedRasterResult> result =
		_ssid_raster_result_mailbox.take();
	if (!result.has_value()) {
		return;
	}

	LvLockGuard lock;
	if (_page_manager && _page_manager->is_initialized()) {
		_page_manager->accept_prepared_ssid_raster(*result);
	}
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
void UILVGLApp::schedule_background_assets_apply(ui::BackgroundAssets assets)
{
	std::lock_guard<std::mutex> lock(_background_assets_mutex);
	if (_background_assets_mailbox_closed) {
		return;
	}

	_pending_background_assets = std::move(assets);
	if (_background_assets_apply_pending) {
		return;
	}

	_background_assets_apply_pending = true;
	if (lv_async_call(background_assets_apply_async_cb, this) !=
	    LV_RESULT_OK) {
		_background_assets_apply_pending = false;
		spdlog::warn("Failed to schedule background asset UI apply");
	}
}

void UILVGLApp::apply_pending_background_assets()
{
	ui::BackgroundAssets assets;
	{
		std::lock_guard<std::mutex> lock(_background_assets_mutex);
		_background_assets_apply_pending = false;
		if (_background_assets_mailbox_closed) {
			return;
		}
		assets = std::move(_pending_background_assets);
	}

	LvLockGuard lock;
	if (_page_manager && _page_manager->is_initialized()) {
		_page_manager->replace_background_assets(std::move(assets));
	}
}

void UILVGLApp::background_assets_apply_async_cb(void *user_data)
{
	auto *self = static_cast<UILVGLApp *>(user_data);
	if (self == nullptr) {
		return;
	}

	self->apply_pending_background_assets();
}
#endif

void UILVGLApp::apps_refresh_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<UILVGLApp *>(lv_timer_get_user_data(timer));
	if (self == nullptr || !self->_page_manager ||
	    !self->_page_manager->is_initialized()) {
		return;
	}
	self->_page_manager->set_apps(lvgl_main::scan_apps(
		lvgl_main::simulator_policy::launcher_apps_path()));
}

ui::PageManagerActions UILVGLApp::build_page_manager_actions()
{
	ui::PageManagerActions actions;

	actions.ssid_raster.prepare = [this](ui::SsidRasterRequest request) {
		if (_ssid_raster_worker) {
			return _ssid_raster_worker->submit(std::move(request));
		}
		return false;
	};
	actions.ssid_raster.invalidate_page = [this](uint64_t page_token) {
		if (_ssid_raster_worker) {
			_ssid_raster_worker->invalidate_page(page_token);
		}
	};

#if NANOKVM_GO_SIMULATOR_BUILD
	actions.apps.launch = [this](const std::string &path) {
		const auto result = _simulator_app_launcher.launch(path);
		if (!result.ok()) {
			spdlog::warn("Simulator Apps mock launch failed: {}",
				     result.detail);
		}
	};
#else
	actions.apps.launch = [this](const std::string &path) {
		{
			std::lock_guard<std::mutex> lock(_launch_mtx);
			_launch_path = path;
		}
		_launch_request.store(true, std::memory_order_release);
	};
#endif

	actions.settings.set_auto_rotation_enabled = [this](bool enabled) {
		if (_auto_rotation_imu_available) {
			set_auto_rotation_enabled(enabled);
			return;
		}
		set_manual_display_rotation_enabled(enabled);
	};

	if (!_runtime_service) {
		return actions;
	}

	actions.settings.set_ncm_enabled = [this](bool enabled) {
		if (_runtime_service) {
			_runtime_service->set_ncm_enabled(enabled);
		}
	};
	actions.settings.refresh_service_state = [this]() {
		if (_runtime_service) {
			_runtime_service->refresh_local_service_state();
		}
	};
	actions.settings.set_mcp_enabled = [this](bool enabled) {
		if (_runtime_service) {
			_runtime_service->set_mcp_bundle_enabled(enabled);
		}
	};
	actions.settings.set_replay_enabled = [this](bool enabled) {
		if (_runtime_service) {
			_runtime_service->set_screen_timelapse_enabled(enabled);
		}
	};
	actions.settings.set_ssh_enabled = [this](bool enabled) {
		if (_runtime_service) {
			_runtime_service->set_ssh_enabled(enabled);
		}
	};

	actions.wifi.page_visibility_changed = [this](bool visible) {
		if (_runtime_service) {
			_runtime_service->notify_wifi_page_visible(visible);
		}
	};
	actions.more.page_visibility_changed = [this](bool visible) {
		if (_runtime_service) {
			_runtime_service->notify_more_page_visible(visible);
		}
	};
	actions.tailscale.page_visibility_changed = [this](bool visible) {
		if (_runtime_service) {
			_runtime_service->notify_tailscale_page_visible(visible);
		}
	};
	actions.tailscale.start = [this]() -> bool {
		return _runtime_service &&
		       _runtime_service->request_tailscale_start();
	};
	actions.tailscale.exit = [this]() -> bool {
		return _runtime_service &&
		       _runtime_service->request_tailscale_exit();
	};
	actions.wifi.scan = [this]() {
		if (_runtime_service) {
			_runtime_service->scan_wifi_networks();
		}
	};
	actions.wifi.connect = [this](const std::string &ssid,
				      const std::string &password) {
		if (_runtime_service) {
			_runtime_service->connect_wifi_network(ssid, password);
		}
	};
	actions.wifi.start_qr_ap = [this]() {
		if (_runtime_service) {
			_runtime_service->start_wifi_qr_ap_config();
		}
	};
	actions.wifi.stop_qr_ap = [this]() -> uint64_t {
		if (_runtime_service) {
			return _runtime_service->stop_wifi_qr_ap_config();
		}
		return 0;
	};
	actions.wifi.stop_qr_ap_after_success = [this]() {
		if (_runtime_service) {
			_runtime_service->stop_wifi_qr_ap_config_after_success();
		}
	};
	actions.wifi.set_enabled = [this](bool enabled) {
		if (_runtime_service) {
			_runtime_service->set_wifi_enabled(enabled);
		}
	};
	actions.wifi.reset = [this]() {
		if (_runtime_service) {
			_runtime_service->reset_wifi();
		}
	};
	actions.wifi.qr_dismissed = [this]() {
		if (_runtime_service) {
			_runtime_service->notify_wifi_qr_dismissed();
		}
	};

	actions.panel.set_brightness_percent = [this](int percent) {
		if (_runtime_service) {
			_runtime_service->apply_backlight_percent_now(percent);
		}
	};
	actions.panel.set_sleep_enabled = [this](bool enabled) {
		_auto_sleep_policy_enabled.store(enabled,
						 std::memory_order_release);
		if (!enabled) {
			_auto_sleep_sleep_request_active = false;
			_auto_sleep_screen_off_observed = false;
			_auto_sleep_wake_request_active = false;
			_last_input_activity = std::chrono::steady_clock::now();
		}
		if (_runtime_service) {
			_runtime_service->set_auto_sleep_enabled(enabled);
		}
	};

	actions.about.request_factory_reset = [this]() -> bool {
		return _runtime_service &&
		       _runtime_service->request_factory_reset();
	};
	actions.region.request_switch = [this](ui::RegionOption target) -> bool {
		return _runtime_service &&
		       _runtime_service->request_region_switch(
			       target == ui::RegionOption::ChinaRegion ?
				       lvgl_main::DeviceRegion::Cn :
				       lvgl_main::DeviceRegion::Global);
	};
	return actions;
}

void UILVGLApp::bind_runtime_actions()
{
	if (!_page_manager || !_runtime_service) {
		return;
	}

	_page_manager->bind_actions(build_page_manager_actions());
	_page_manager->set_wifi_ap_qr_payload(
		_runtime_service->wifi_ap_qr_payload());
}

void UILVGLApp::start_runtime_backend()
{
	if (_runtime_service) {
		return;
	}

#if NANOKVM_GO_SIMULATOR_BUILD
	_runtime_service = std::make_unique<lvgl_main::SimulatorRuntimeService>(
		make_runtime_callbacks());
	bind_runtime_actions();
	{
		LvLockGuard lock;
		apply_runtime_status(_runtime_service->current_status());
	}
	if (!_runtime_service->start()) {
		spdlog::error("Failed to start simulator runtime backend");
		_runtime_service.reset();
	}
#else
	if (_config.nanokvm.enable_runtime_backend) {
		auto target_runtime =
			std::make_unique<lvgl_main::NanoKvmRuntimeService>(
				_config, make_runtime_callbacks(),
				_config_path);
		_target_runtime_service = target_runtime.get();
		_runtime_service = std::move(target_runtime);
		bind_runtime_actions();
		{
			LvLockGuard lock;
			apply_runtime_status(
				_runtime_service->current_status());
		}
		if (!_runtime_service->start()) {
			spdlog::error(
				"Failed to start NanoKVM runtime backend");
			_target_runtime_service = nullptr;
			_runtime_service.reset();
		} else {
			_runtime_service->refresh_local_service_state();
		}
	} else {
		spdlog::info(
			"[lvgl] NanoKVM runtime backend disabled by config");
	}
#endif
}

void UILVGLApp::run_lvgl_event_loop(bool enable_runtime_policies)
{
	while (g_running != 0) {
		uint32_t next_timer_ms = 0;
		const bool probe_enabled = lvgl_main::wifi_latency_probe::enabled();
		const auto timer_handler_started = probe_enabled ?
			std::chrono::steady_clock::now() :
			std::chrono::steady_clock::time_point {};
		{
			LvLockGuard lock;
			next_timer_ms = lv_timer_handler();
		}
		if (probe_enabled) {
			const auto timer_handler_completed =
				std::chrono::steady_clock::now();
			lvgl_main::wifi_latency_probe::record_timer_handler(
				static_cast<uint64_t>(
					std::chrono::duration_cast<std::chrono::microseconds>(
						timer_handler_completed - timer_handler_started)
						.count()));
		}

		if (enable_runtime_policies) {
			update_input_activity();
		}
		update_ssid_raster_install_gate();
		if (_ssid_raster_worker) {
			apply_pending_ssid_raster_result();
		}
#if !NANOKVM_GO_SIMULATOR_BUILD
		handle_snapshot_request();
#endif

#if !NANOKVM_GO_SIMULATOR_BUILD
		if (enable_runtime_policies &&
		    _launch_request.exchange(false)) {
			std::string path;
			{
				std::lock_guard<std::mutex> lock(_launch_mtx);
				path = _launch_path;
			}
			run_app_session(path);
			continue;
		}
#endif
		if (enable_runtime_policies) {
			handle_touch_click_marker();
			handle_auto_rotation();
			handle_auto_sleep();
		}
		if (probe_enabled) {
			lvgl_main::wifi_latency_probe::maybe_log();
		}
		const uint32_t sleep_ms = std::clamp(next_timer_ms, 1U, 25U);
		std::this_thread::sleep_for(
			std::chrono::milliseconds(sleep_ms));
	}
}

void UILVGLApp::sync_wifi_scan_status(const lvgl_main::RuntimeStatus &status)
{
	if (!_page_manager) {
		return;
	}

	if (status.wifi_scan_running) {
		if (!_last_wifi_scan_running) {
			_page_manager->set_wifi_scan_loading("Scanning...");
		}
		_wifi_scan_result_pending = true;
		_last_wifi_scan_running = true;
		return;
	}

	if (_wifi_scan_result_pending || _last_wifi_scan_running) {
		if (!status.wifi_scan_result.ok) {
			_page_manager->set_wifi_scan_error(
				wifi_scan_error_message(
					status.wifi_scan_result));
		} else {
			const auto results = to_ui_wifi_scan_results(
				status.wifi_scan_result);
			if (results.empty()) {
				_page_manager->set_wifi_scan_empty(
					"No WiFi found");
			} else {
				_page_manager->set_wifi_scan_results(results);
			}
		}
		_wifi_scan_result_pending = false;
	}

	_last_wifi_scan_running = false;
}

void UILVGLApp::sync_wifi_connect_status(const lvgl_main::RuntimeStatus &status)
{
	if (!_page_manager) {
		return;
	}

	const auto current_status = status.wifi_connect_result.status;
	const std::string current_message = wifi_connect_status_message(status);
	const bool has_result =
		current_status == lvgl_main::WifiConnectStatus::Success ||
		current_status == lvgl_main::WifiConnectStatus::Failure;
	const bool result_changed =
		current_status != _last_wifi_connect_status ||
		status.wifi_connect_result.ssid != _last_wifi_connect_ssid ||
		current_message != _last_wifi_connect_message;

	if (status.wifi_connect_running) {
		if (!_last_wifi_connect_running || result_changed) {
			_page_manager->set_wifi_connect_loading();
		}
		_wifi_connect_result_pending = true;
		_last_wifi_connect_running = true;
		_last_wifi_connect_status = current_status;
		_last_wifi_connect_ssid = status.wifi_connect_result.ssid;
		_last_wifi_connect_message = current_message;
		return;
	}

	if (has_result && (_wifi_connect_result_pending ||
			   _last_wifi_connect_running || result_changed)) {
		if (current_status == lvgl_main::WifiConnectStatus::Success) {
			_page_manager->set_wifi_connect_success(
				current_message);
		} else {
			_page_manager->set_wifi_connect_failure(
				current_message);
		}
		_wifi_connect_result_pending = false;
	}

	_last_wifi_connect_running = false;
	_last_wifi_connect_status = current_status;
	_last_wifi_connect_ssid = status.wifi_connect_result.ssid;
	_last_wifi_connect_message = current_message;
}

void UILVGLApp::apply_runtime_status(const lvgl_main::RuntimeStatus &status)
{
	_auto_sleep_policy_enabled.store(status.auto_sleep_enabled,
					 std::memory_order_release);
	_runtime_screen_on.store(status.screen_on, std::memory_order_release);
	set_touch_input_screen_on(status.screen_on);

	if (!_page_manager || !_page_manager->is_initialized()) {
		return;
	}

	const bool wifi_active = status.wifi_connected;
	const ui::WifiPageVisualState wifi_visual_state =
		to_wifi_page_visual_state(status.wifi_runtime_state);

	static bool previous_wifi_status_initialized = false;
	static lvgl_main::WifiRuntimeState previous_wifi_runtime_state =
		lvgl_main::WifiRuntimeState::Unknown;
	static ui::WifiPageVisualState previous_wifi_visual_state =
		ui::WifiPageVisualState::Off;
	static bool previous_wifi_busy = false;
	static bool previous_wifi_boot_reconnect_running = false;
	static bool previous_wifi_show_qr = false;
	static bool previous_wifi_connected = false;
	static bool previous_wifi_ap_mode = false;
	static bool previous_wifi_ap_client_connected = false;
	static bool previous_wifi_active = false;
	static std::size_t previous_wifi_qr_payload_len = 0;
	static std::size_t previous_wifi_web_url_len = 0;
	static std::string previous_wifi_status_text;

	if (!previous_wifi_status_initialized ||
	    previous_wifi_runtime_state != status.wifi_runtime_state ||
	    previous_wifi_visual_state != wifi_visual_state ||
	    previous_wifi_busy != status.wifi_busy ||
	    previous_wifi_boot_reconnect_running !=
		    status.wifi_boot_reconnect_running ||
	    previous_wifi_show_qr != status.wifi_show_qr ||
	    previous_wifi_connected != status.wifi_connected ||
	    previous_wifi_ap_mode != status.wifi_ap_mode ||
	    previous_wifi_ap_client_connected !=
		    status.wifi_ap_client_connected ||
	    previous_wifi_active != wifi_active ||
	    previous_wifi_qr_payload_len != status.wifi_qr_payload.size() ||
	    previous_wifi_web_url_len != status.wifi_web_url.size() ||
	    previous_wifi_status_text != status.wifi_status_text) {
		spdlog::info(
			"[wifi] WIFI UI runtime mapped: runtime={} visual={} busy={} "
			"boot_reconnect={} show_qr={} active={} connected={} "
			"ap_mode={} ap_client={} "
			"status_text=\"{}\" qr_payload_len={} web_url_len={}",
			wifi_runtime_state_name(status.wifi_runtime_state),
			wifi_visual_state_name(wifi_visual_state),
			status.wifi_busy, status.wifi_boot_reconnect_running,
			status.wifi_show_qr, wifi_active, status.wifi_connected,
			status.wifi_ap_mode, status.wifi_ap_client_connected,
			status.wifi_status_text, status.wifi_qr_payload.size(),
			status.wifi_web_url.size());
		previous_wifi_status_initialized = true;
		previous_wifi_runtime_state = status.wifi_runtime_state;
		previous_wifi_visual_state = wifi_visual_state;
		previous_wifi_busy = status.wifi_busy;
		previous_wifi_boot_reconnect_running =
			status.wifi_boot_reconnect_running;
		previous_wifi_show_qr = status.wifi_show_qr;
		previous_wifi_connected = status.wifi_connected;
		previous_wifi_ap_mode = status.wifi_ap_mode;
		previous_wifi_ap_client_connected =
			status.wifi_ap_client_connected;
		previous_wifi_active = wifi_active;
		previous_wifi_qr_payload_len = status.wifi_qr_payload.size();
		previous_wifi_web_url_len = status.wifi_web_url.size();
		previous_wifi_status_text = status.wifi_status_text;
	}

	const auto wifi_ui_state = lvgl_main::make_wifi_runtime_ui_state(
		{ .boot_reconnect_running = status.wifi_boot_reconnect_running,
		  .wifi_runtime_state = status.wifi_runtime_state,
		  .has_main_ip = !status.main_ip_candidates.empty() });
	_page_manager->set_main_ip_empty_placeholder(
		wifi_ui_state.main_ip_placeholder);
	_page_manager->set_runtime_wifi_boot_reconnect_pending(
		wifi_ui_state.settings_wifi_pending);
	_page_manager->set_main_ip_candidates(status.main_ip_candidates);
	_page_manager->set_runtime_wifi_active(wifi_active);
	_page_manager->set_runtime_hdmi_input_active(status.hdmi_input_enabled);
	_page_manager->set_runtime_hid_active(status.hid_enabled);

	if (!status.display_resolution_label.empty()) {
		_page_manager->set_main_resolution_text(
			status.display_resolution_label.c_str());
	} else if (!status.resolution.empty()) {
		_page_manager->set_main_resolution_text(
			status.resolution.c_str());
	}

	if (!status.display_fps_label.empty()) {
		_page_manager->set_main_fps_text(
			status.display_fps_label.c_str());
	} else if (status.fps.has_value()) {
		if (has_valid_fps(*status.fps)) {
			std::ostringstream fps;
			fps << std::fixed
			    << std::setprecision(*status.fps >= 100.0 ? 0 : 1)
			    << *status.fps;
			_page_manager->set_main_fps_text(fps.str().c_str());
		} else {
			_page_manager->set_main_fps_text("");
		}
	}
	if (!status.display_fps_unit_label.empty()) {
		_page_manager->set_main_fps_unit_text(
			status.display_fps_unit_label.c_str());
	} else {
		_page_manager->set_main_fps_unit_text("FPS");
	}
	if (status.hid_keyboard_special_label_generation !=
	    _last_hid_keyboard_special_label_generation) {
		_last_hid_keyboard_special_label_generation =
			status.hid_keyboard_special_label_generation;
		_page_manager->set_main_keyboard_special_text(
			status.hid_keyboard_special_label.c_str());
	}
	if (status.hid_keyboard_normal_label_generation !=
	    _last_hid_keyboard_normal_label_generation) {
		_last_hid_keyboard_normal_label_generation =
			status.hid_keyboard_normal_label_generation;
		_page_manager->set_main_keyboard_normal_text(
			status.hid_keyboard_normal_label.c_str());
	}
	if (status.hid_mouse_state_generation !=
	    _last_hid_mouse_state_generation) {
		_last_hid_mouse_state_generation =
			status.hid_mouse_state_generation;
		_page_manager->set_main_mouse_active(status.hid_mouse_active);
	}

	_page_manager->set_wifi_runtime_state(
		wifi_visual_state, status.wifi_status_text, status.wifi_busy,
		status.wifi_ssid, status.wifi_ip, status.wifi_saved_network_config,
		status.wifi_show_qr, status.wifi_qr_payload,
		status.wifi_web_url);
	if (wifi_visual_state == ui::WifiPageVisualState::ApConnect) {
		_page_manager->set_wifi_qr_title("Connect to AP");
	} else if (wifi_visual_state == ui::WifiPageVisualState::WebConfigure) {
		_page_manager->set_wifi_qr_title("Configure WiFi");
	}
	_page_manager->set_wifi_qr_payload(wifi_qr_payload_for_status(status));
	_page_manager->set_wifi_ap_stop_completed_generation(
		status.wifi_ap_stop_completed_generation);
	sync_wifi_scan_status(status);
	sync_wifi_connect_status(status);
	if (status.backlight_percent.has_value()) {
		_page_manager->set_panel_brightness_percent(
			*status.backlight_percent);
	}
	_page_manager->set_panel_sleep_enabled(status.auto_sleep_enabled);
	_page_manager->set_runtime_ncm_active(status.ncm_enabled);
	if (status.ssh_supported) {
		_page_manager->set_runtime_ssh_active(status.ssh_enabled);
	}
	_page_manager->apply_settings_service_runtime_state(
		ui::PageManager::SettingsServiceRuntimeState {
			.known = status.local_services_known,
			.refresh_failed = !status.local_services_refresh_ok,
			.event_seq = status.local_services_event_seq,
			.mcp_active = status.mcp_service_active,
			.mcp_available = status.mcp_service_supported,
			.replay_active = status.replay_service_active,
			.replay_available = status.replay_service_supported,
		});
	_page_manager->set_runtime_panel_active(
		status.backlight_percent.has_value());
	if (status.factory_reset_failure_generation !=
	    _last_factory_reset_failure_generation) {
		_last_factory_reset_failure_generation =
			status.factory_reset_failure_generation;
		_page_manager->clear_about_factory_reset_pending();
	}
	_page_manager->set_region_option(ui::region_option_from_cn(
		status.region == lvgl_main::DeviceRegion::Cn));
	if (status.region_switch_failure_generation !=
	    _last_region_switch_failure_generation) {
		_last_region_switch_failure_generation =
			status.region_switch_failure_generation;
		_page_manager->clear_region_switch_pending();
	}
	_page_manager->set_about_versions(
		status.app_version.empty() ? "--" : status.app_version, "--",
		status.hw_version.empty() ? "--" : status.hw_version,
		lvgl_main::build_info::kGitCommit);
	_page_manager->set_tailscale_tile_active(
		lvgl_main::tailscale_tile_active(status));
	lvgl_main::project_tailscale_status(*_page_manager, status);
	_page_manager->set_more_sysinfo(ui::MoreSysinfoSnapshot {
		.device_key = status.device_key,
		.hostname = status.hostname,
		.device_number = status.device_number,
		.image_version = status.image_version,
		.edid_name = status.edid_name,
		.mac_address = status.mac_address,
		.mdns_name = status.mdns_name,
		.wifi_signal = status.wifi_signal,
		.uptime_text = status.uptime_text,
		.local_time_text = status.local_time_text,
		.memory_text = status.memory_text,
		.storage_text = status.storage_text,
		.temperature_text = status.temperature_text,
	});
}

void UILVGLApp::update_input_activity()
{
	const uint64_t activity_seq = touch_input_activity_sequence();
	if (activity_seq == _last_input_activity_seq) {
		return;
	}

	_last_input_activity_seq = activity_seq;
	_last_input_activity = std::chrono::steady_clock::now();
	if ((_auto_sleep_sleep_request_active ||
	     _auto_sleep_screen_off_observed) &&
	    !_auto_sleep_wake_request_active && _runtime_service) {
		_runtime_service->set_screen_sleep(false);
		_auto_sleep_wake_request_active = true;
	}
	_auto_sleep_sleep_request_active = false;
}

void UILVGLApp::update_ssid_raster_install_gate()
{
	_ssid_raster_install_gate.observe(touch_input_activity_sequence(),
					     std::chrono::steady_clock::now());
}

bool UILVGLApp::should_defer_ssid_raster_install() const
{
	return _ssid_raster_install_gate.blocks_install(
		touch_input_is_pressed(), std::chrono::steady_clock::now());
}

void UILVGLApp::handle_auto_sleep()
{
	if (!_runtime_service) {
		return;
	}

	const auto now = std::chrono::steady_clock::now();
	const bool policy_enabled =
		_auto_sleep_policy_enabled.load(std::memory_order_acquire);
	if (!policy_enabled) {
		_auto_sleep_policy_was_enabled = false;
		_auto_sleep_sleep_request_active = false;
		_auto_sleep_screen_off_observed = false;
		_auto_sleep_wake_request_active = false;
		_last_input_activity = now;
		return;
	}

	if (!_auto_sleep_policy_was_enabled) {
		_auto_sleep_policy_was_enabled = true;
		_auto_sleep_sleep_request_active = false;
		_auto_sleep_screen_off_observed = false;
		_auto_sleep_wake_request_active = false;
		_last_input_activity = now;
		return;
	}

	const bool screen_on =
		_runtime_screen_on.load(std::memory_order_acquire);
	if (_auto_sleep_wake_request_active) {
		if (screen_on) {
			_auto_sleep_wake_request_active = false;
			_auto_sleep_screen_off_observed = false;
			_last_input_activity = now;
		}
		return;
	}

	if (!screen_on) {
		_auto_sleep_sleep_request_active = true;
		_auto_sleep_screen_off_observed = true;
		return;
	}

	if (_auto_sleep_screen_off_observed) {
		_auto_sleep_sleep_request_active = false;
		_auto_sleep_screen_off_observed = false;
		_last_input_activity = now;
		return;
	}

	if (_auto_sleep_sleep_request_active) {
		return;
	}

	if (now - _last_input_activity >= kAutoSleepIdleTimeout) {
		_runtime_service->set_screen_sleep(true);
		_auto_sleep_sleep_request_active = true;
	}
}
