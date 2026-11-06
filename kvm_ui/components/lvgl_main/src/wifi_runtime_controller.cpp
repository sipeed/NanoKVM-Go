#include "lvgl_main/wifi_runtime_controller.hpp"

#include <chrono>
#include <future>
#include <memory>
#include <mutex>
#include <utility>
#include <vector>

#include "wifi_runtime_controller_detail.hpp"

namespace lvgl_main
{

namespace
{

using detail::ProjectionKind;
using detail::build_projection_locked;
using detail::current_ap_credentials;
using detail::has_saved_network_config;
using detail::is_open;
using detail::is_open_locked;
using detail::make_projection;
using detail::operation_is_current_locked;
using detail::tick_from_ap_mode_locked;
using detail::tick_from_cache_connecting_locked;
using detail::tick_from_connected_locked;
using detail::tick_from_disconnected_locked;
using detail::tick_from_error_locked;
using detail::publish_lifecycle;
using detail::request_ap_stop;
using detail::run_ap_start_action;
using detail::run_connect_task;
using detail::run_disable_task;
using detail::run_enable_task;
using detail::run_reset_task;
using detail::run_scan_task;
using detail::run_script;
using detail::set_wifi_runtime_state;
using detail::submit_async_task;

} // namespace

void WifiRuntimeProjection::apply_to(RuntimeStatus &status) const
{
	status.wifi_ip = wifi_ip;
	status.wifi_supported = wifi_supported;
	status.wifi_connected = wifi_connected;
	status.wifi_ap_mode = wifi_ap_mode;
	status.wifi_ap_client_connected = wifi_ap_client_connected;
	status.wifi_qr_payload = wifi_qr_payload;
	status.wifi_web_url = wifi_web_url;
	status.wifi_runtime_state = wifi_runtime_state;
	status.wifi_ssid = wifi_ssid;
	status.wifi_saved_network_config = wifi_saved_network_config;
	status.wifi_status_text = wifi_status_text;
	status.wifi_error = wifi_error;
	status.wifi_busy = wifi_busy;
	status.wifi_boot_reconnect_running = wifi_boot_reconnect_running;
	status.wifi_show_qr = wifi_show_qr;
	status.wifi_ap_stop_completed_generation =
		wifi_ap_stop_completed_generation;
	status.wifi_scan_running = wifi_scan_running;
	status.wifi_scan_result = wifi_scan_result;
	status.wifi_connect_running = wifi_connect_running;
	status.wifi_connect_result = wifi_connect_result;
	status.wifi_connect_message = wifi_connect_message;
}

WifiRuntimeController::WifiRuntimeController(
	Dependencies dependencies, bool boot_reconnect_pending,
	std::chrono::milliseconds saved_network_timeout)
	: _state(std::make_shared<detail::WifiRuntimeControllerState>(
		  std::move(dependencies), boot_reconnect_pending,
		  saved_network_timeout))
{
}

WifiRuntimeController::~WifiRuntimeController()
{
	close();
}

void WifiRuntimeController::close()
{
	if (!_state) {
		return;
	}
	std::vector<std::future<void>> tasks;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		// New tasks observe this request gate at entry. Existing tasks retain
		// their generation so close() can join them after their terminal
		// projection has reached the façade-owned state store.
		_state->closed = true;
		std::lock_guard<std::mutex> task_lock(_state->task_mutex);
		tasks.swap(_state->async_tasks);
	}
	_state->scanner.shutdown();
	for (auto &task : tasks) {
		if (task.valid()) {
			task.wait();
		}
	}
}

bool WifiRuntimeController::is_closed() const
{
	return !_state || !is_open(_state);
}

uint64_t WifiRuntimeController::operation_generation() const
{
	if (!_state) {
		return 0;
	}
	std::lock_guard<std::mutex> lock(_state->mutex);
	return _state->wifi_operation_generation;
}

WifiRuntimeProjection WifiRuntimeController::current_projection() const
{
	return _state ? make_projection(_state, ProjectionKind::Lifecycle) :
			WifiRuntimeProjection {};
}

WifiRuntimeProjection
WifiRuntimeController::project_observation(WifiRuntimeObservation observation)
{
	if (!_state) {
		return {};
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	const WifiSavedNetworkAvailability saved_network_avail =
		has_saved_network_config(_state);
	const bool reconnect_saved_network_config =
		observation.wifi_supported &&
		(saved_network_avail == WifiSavedNetworkAvailability::Present);
	WifiRuntimeProjection projection;
	bool stop_scanner = false;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		_state->last_observation = std::move(observation);
		switch (saved_network_avail) {
		case WifiSavedNetworkAvailability::Present:
			_state->wifi_saved_network_config = true;
			break;
		case WifiSavedNetworkAvailability::Absent:
			_state->wifi_saved_network_config = false;
			break;
		case WifiSavedNetworkAvailability::Unknown:
			break;
		}
		if (is_open_locked(*_state)) {
			if (_state->wifi_scan_running) {
				_state->wifi_busy = true;
				_state->wifi_boot_reconnect_running = false;
				_state->wifi_status_text = "Scanning...";
				_state->wifi_error.clear();
			} else if (_state->wifi_connect_running) {
				_state->wifi_busy = true;
				_state->wifi_boot_reconnect_running = false;
				set_wifi_runtime_state(
					*_state,
					WifiRuntimeState::CacheConnecting,
					"connecting...");
			} else if (!_state->last_observation.wifi_error.empty()) {
				_state->wifi_busy = false;
				_state->wifi_boot_reconnect_running = false;
				_state->wifi_ap_opened_by_ui = false;
				set_wifi_runtime_state(
					*_state, WifiRuntimeState::Error,
					"No WiFi",
					_state->last_observation.wifi_error);
			} else {
				switch (_state->wifi_runtime_state) {
				case WifiRuntimeState::Unknown:
				case WifiRuntimeState::Disconnected:
				case WifiRuntimeState::WaitingExternalConnection:
					tick_from_disconnected_locked(
						*_state,
						_state->last_observation,
						reconnect_saved_network_config);
					break;
				case WifiRuntimeState::Connected:
					tick_from_connected_locked(
						*_state,
						_state->last_observation);
					break;
				case WifiRuntimeState::CacheConnecting:
					tick_from_cache_connecting_locked(
						*_state,
						_state->last_observation);
					break;
				case WifiRuntimeState::ApAwaitClient:
				case WifiRuntimeState::ApAwaitWeb:
					stop_scanner = tick_from_ap_mode_locked(
						*_state,
						_state->last_observation);
					break;
				case WifiRuntimeState::Error:
					tick_from_error_locked(
						*_state,
						_state->last_observation);
					break;
				}
			}
		}
		projection = build_projection_locked(
			*_state, credentials, ProjectionKind::Observation);
	}
	if (stop_scanner) {
		_state->scanner.stop();
	}
	return projection;
}

void WifiRuntimeController::set_enabled(bool enabled)
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	uint64_t operation_generation = 0;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			return;
		}
		operation_generation = ++_state->wifi_operation_generation;
		_state->wifi_reconnect_after_qr_exit = false;
		if (enabled) {
			_state->wifi_busy = true;
			_state->wifi_boot_reconnect_running = false;
			++_state->wifi_ap_config_generation;
			_state->wifi_ap_config_running = false;
			_state->wifi_ap_opened_by_ui = false;
			set_wifi_runtime_state(
				*_state, WifiRuntimeState::CacheConnecting,
				"connecting...");
		} else {
			_state->wifi_busy = true;
			_state->wifi_boot_reconnect_running = false;
			++_state->wifi_ap_config_generation;
			_state->wifi_ap_config_running = false;
			_state->wifi_ap_opened_by_ui = false;
			_state->wifi_status_text = "Disconnecting...";
			_state->wifi_error.clear();
		}
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Pending,
			  operation_generation);
	if (enabled) {
		submit_async_task(_state, [_state = _state,
					   operation_generation]() {
			run_enable_task(_state, operation_generation);
		});
	} else {
		submit_async_task(_state, [_state = _state,
					   operation_generation]() {
			run_disable_task(_state, operation_generation);
		});
	}
}

bool WifiRuntimeController::set_enabled_sync(bool enabled, std::string &error)
{
	if (!_state || !is_open(_state)) {
		error = "WiFi runtime is unavailable";
		return false;
	}
	const auto result = run_script(_state, { enabled ? "on" : "off" });
	if (result.exit_code != 0) {
		error = "failed to run WiFi script";
		return false;
	}

	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			error = "WiFi runtime is unavailable";
			return false;
		}
		_state->wifi_reconnect_after_qr_exit = false;
		_state->wifi_busy = false;
		_state->wifi_boot_reconnect_running = false;
		_state->wifi_ap_opened_by_ui = enabled;
		_state->wifi_ap_config_running = false;
		set_wifi_runtime_state(
			*_state,
			enabled ? WifiRuntimeState::ApAwaitClient :
				  WifiRuntimeState::Disconnected,
			enabled ? "Connect to AP" : "Disconnected");
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Observation, 0);
	return true;
}

bool WifiRuntimeController::reset_sync(std::string &error)
{
	if (!_state || !is_open(_state)) {
		error = "WiFi runtime is unavailable";
		return false;
	}
	const auto clear_result = run_script(_state, { "clear" });
	const auto off_result = run_script(_state, { "off" });
	if (clear_result.exit_code != 0 || off_result.exit_code != 0) {
		error = "failed to run WiFi reset script";
		return false;
	}

	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			error = "WiFi runtime is unavailable";
			return false;
		}
		_state->wifi_reconnect_after_qr_exit = false;
		_state->wifi_busy = false;
		_state->wifi_boot_reconnect_running = false;
		_state->wifi_ap_opened_by_ui = false;
		_state->wifi_ap_config_running = false;
		set_wifi_runtime_state(*_state, WifiRuntimeState::Disconnected,
				       "Disconnected");
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Observation, 0);
	return true;
}

void WifiRuntimeController::reset()
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	uint64_t operation_generation = 0;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			return;
		}
		operation_generation = ++_state->wifi_operation_generation;
		_state->wifi_reconnect_after_qr_exit = false;
		_state->wifi_busy = true;
		_state->wifi_boot_reconnect_running = false;
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Pending,
			  operation_generation);
	submit_async_task(_state, [_state = _state, operation_generation]() {
		run_reset_task(_state, operation_generation);
	});
}

void WifiRuntimeController::scan_networks()
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	bool stop_ap_config = false;
	uint64_t operation_generation = 0;
	WifiRuntimeState previous_state = WifiRuntimeState::Unknown;
	std::string previous_status_text;
	std::string previous_error;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state) || _state->wifi_scan_running ||
		    _state->wifi_connect_running) {
			return;
		}
		operation_generation = ++_state->wifi_operation_generation;
		_state->wifi_reconnect_after_qr_exit = false;
		previous_state = _state->wifi_runtime_state;
		previous_status_text = _state->wifi_status_text;
		previous_error = _state->wifi_error;
		stop_ap_config = _state->wifi_runtime_state ==
					 WifiRuntimeState::ApAwaitClient ||
				 _state->wifi_runtime_state ==
					 WifiRuntimeState::ApAwaitWeb ||
				 _state->wifi_ap_opened_by_ui;
		++_state->wifi_ap_config_generation;
		_state->wifi_ap_config_running = false;
		_state->wifi_scan_running = true;
		_state->wifi_scan_result = WifiScanResult {};
		_state->wifi_busy = true;
		_state->wifi_boot_reconnect_running = false;
		if (stop_ap_config) {
			_state->wifi_ap_opened_by_ui = false;
		}
		_state->wifi_status_text = "Scanning...";
		_state->wifi_error.clear();
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Pending,
			  operation_generation);
	submit_async_task(
		_state, [_state = _state, stop_ap_config, operation_generation,
			 previous_state,
			 previous_status_text = std::move(previous_status_text),
			 previous_error = std::move(previous_error)]() mutable {
			run_scan_task(_state, stop_ap_config,
				      operation_generation, previous_state,
				      std::move(previous_status_text),
				      std::move(previous_error));
		});
}

void WifiRuntimeController::connect_network(std::string ssid,
					    std::string password)
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	bool should_launch = false;
	uint64_t operation_generation = 0;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state) || _state->wifi_scan_running ||
		    _state->wifi_connect_running) {
			return;
		}
		_state->wifi_connect_result = WifiConnectResult {};
		_state->wifi_connect_result.ssid = ssid;
		_state->wifi_reconnect_after_qr_exit = false;
		if (ssid.empty()) {
			_state->wifi_connect_result.status =
				WifiConnectStatus::Failure;
			_state->wifi_connect_result.error_code = "INVALID_ARGS";
			_state->wifi_connect_message = "INVALID_ARGS";
			_state->wifi_busy = false;
			_state->wifi_boot_reconnect_running = false;
			set_wifi_runtime_state(*_state, WifiRuntimeState::Error,
					       "Connect failed",
					       _state->wifi_connect_message);
		} else {
			operation_generation =
				++_state->wifi_operation_generation;
			_state->wifi_connect_running = true;
			_state->wifi_connect_message = "Connecting...";
			_state->wifi_busy = true;
			_state->wifi_boot_reconnect_running = false;
			++_state->wifi_ap_config_generation;
			_state->wifi_ap_config_running = false;
			_state->wifi_ap_opened_by_ui = false;
			set_wifi_runtime_state(
				*_state, WifiRuntimeState::CacheConnecting,
				"connecting...");
			should_launch = true;
		}
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	if (should_launch) {
		publish_lifecycle(_state, std::move(projection),
				  WifiLifecyclePatch::Phase::Pending,
				  operation_generation);
		submit_async_task(_state, [_state = _state,
					   ssid = std::move(ssid),
					   password = std::move(password),
					   operation_generation]() mutable {
			run_connect_task(_state, std::move(ssid),
					 std::move(password),
					 operation_generation);
		});
		return;
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Observation, 0);
}

void WifiRuntimeController::start_qr_ap_config()
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	uint64_t ap_config_generation = 0;
	uint64_t operation_generation = 0;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state) || _state->wifi_scan_running ||
		    _state->wifi_connect_running) {
			return;
		}
		_state->wifi_busy = true;
		_state->wifi_boot_reconnect_running = false;
		_state->wifi_ap_opened_by_ui = false;
		_state->wifi_ap_config_running = true;
		_state->wifi_reconnect_after_qr_exit =
			_state->wifi_runtime_state == WifiRuntimeState::Connected &&
			_state->last_observation.wifi_connected;
		ap_config_generation = ++_state->wifi_ap_config_generation;
		operation_generation = ++_state->wifi_operation_generation;
		_state->wifi_status_text = "Starting AP...";
		_state->wifi_error.clear();
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(_state, std::move(projection),
			  WifiLifecyclePatch::Phase::Pending,
			  operation_generation);

	const auto &enqueue = _state->dependencies.enqueue_serial_work;
	const bool queued =
		enqueue &&
		enqueue("wifi_qr_ap_start",
			[state = _state, ap_config_generation,
			 operation_generation]() {
				run_ap_start_action(state, ap_config_generation,
						    operation_generation);
			});
	if (queued) {
		return;
	}

	const WifiRuntimeApCredentials failure_credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection failure_projection;
	bool should_publish = false;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (operation_is_current_locked(*_state,
						operation_generation) &&
		    ap_config_generation == _state->wifi_ap_config_generation &&
		    _state->wifi_ap_config_running) {
			_state->wifi_ap_config_running = false;
			_state->wifi_ap_opened_by_ui = false;
			_state->wifi_busy = false;
			_state->wifi_boot_reconnect_running = false;
			set_wifi_runtime_state(*_state, WifiRuntimeState::Error,
					       "AP start unavailable",
					       "AP start unavailable");
			failure_projection = build_projection_locked(
				*_state, failure_credentials,
				ProjectionKind::Lifecycle);
			should_publish = true;
		}
	}
	if (should_publish) {
		publish_lifecycle(_state, std::move(failure_projection),
				  WifiLifecyclePatch::Phase::Terminal,
				  operation_generation);
	}
}

uint64_t WifiRuntimeController::stop_qr_ap_config()
{
	return _state ? request_ap_stop(_state, "qr-dismiss") : 0;
}

void WifiRuntimeController::stop_qr_ap_config_after_success()
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	bool should_publish = false;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			return;
		}
		const bool in_ap_config_state =
			_state->wifi_runtime_state ==
				WifiRuntimeState::ApAwaitClient ||
			_state->wifi_runtime_state ==
				WifiRuntimeState::ApAwaitWeb;
		const bool has_pending_ap_stop =
			_state->wifi_ap_stop_pending_generation != 0;
		if (!in_ap_config_state && !_state->wifi_ap_opened_by_ui &&
		    !_state->wifi_ap_config_running && !has_pending_ap_stop) {
			return;
		}
		_state->wifi_busy = false;
		_state->wifi_boot_reconnect_running = false;
		_state->wifi_reconnect_after_qr_exit = false;
		++_state->wifi_ap_config_generation;
		_state->wifi_ap_config_running = false;
		_state->wifi_ap_opened_by_ui = false;
		projection = build_projection_locked(*_state, credentials,
						     ProjectionKind::Lifecycle);
		should_publish = true;
	}
	_state->scanner.stop();
	if (should_publish) {
		publish_lifecycle(_state, std::move(projection),
				  WifiLifecyclePatch::Phase::Observation, 0);
	}
}

void WifiRuntimeController::notify_page_visible(bool visible)
{
	if (!_state) {
		return;
	}
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(_state);
	WifiRuntimeProjection projection;
	bool needs_cleanup = false;
	bool needs_publish = false;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			return;
		}
		const bool visibility_changed = _state->wifi_page_visible !=
						visible;
		_state->wifi_page_visible = visible;
		needs_cleanup = !visible && (_state->wifi_ap_config_running ||
					     _state->wifi_ap_opened_by_ui);
		if (visibility_changed && !needs_cleanup) {
			projection = build_projection_locked(
				*_state, credentials,
				ProjectionKind::Lifecycle);
			needs_publish = true;
		}
	}
	if (needs_cleanup) {
		(void)request_ap_stop(_state, "wifi-flow-exit");
		return;
	}
	if (needs_publish) {
		publish_lifecycle(_state, std::move(projection),
				  WifiLifecyclePatch::Phase::Observation, 0);
	}
}

void WifiRuntimeController::notify_qr_dismissed()
{
	if (!_state) {
		return;
	}
	bool in_ap_config_state = false;
	{
		std::lock_guard<std::mutex> lock(_state->mutex);
		if (!is_open_locked(*_state)) {
			return;
		}
		in_ap_config_state = _state->wifi_runtime_state ==
					     WifiRuntimeState::ApAwaitClient ||
				     _state->wifi_runtime_state ==
					     WifiRuntimeState::ApAwaitWeb;
	}
	if (in_ap_config_state) {
		(void)request_ap_stop(_state, "legacy-qr-dismiss");
	}
}

std::string WifiRuntimeController::qr_payload() const
{
	return _state ? current_ap_credentials(_state).qr_payload :
			std::string {};
}

} // namespace lvgl_main
