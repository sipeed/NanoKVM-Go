#include "wifi_runtime_controller_detail.hpp"

#include <algorithm>
#include <chrono>
#include <cctype>
#include <condition_variable>
#include <exception>
#include <future>
#include <functional>
#include <mutex>
#include <thread>
#include <utility>

#include <spdlog/spdlog.h>

#include "lvgl_main/wifi_runtime_helpers.hpp"

namespace lvgl_main
{

namespace
{

constexpr std::chrono::milliseconds kApClientPollInterval { 100 };

std::string trim(std::string value)
{
	const auto first = std::find_if_not(
		value.begin(), value.end(), [](unsigned char ch) {
			return std::isspace(ch) != 0;
		});
	const auto last = std::find_if_not(value.rbegin(), value.rend(),
					   [](unsigned char ch) {
						   return std::isspace(ch) != 0;
					   })
				  .base();
	if (first >= last) {
		return {};
	}
	return std::string(first, last);
}

bool contains_text(std::string_view text, std::string_view needle)
{
	return text.find(needle) != std::string_view::npos;
}

bool script_result_ok(const WifiScriptExecutionResult &result)
{
	return result.exit_code == 0 && !result.timed_out &&
	       !contains_text(result.output, "ERROR:");
}

std::string wifi_runtime_state_name(WifiRuntimeState state)
{
	switch (state) {
	case WifiRuntimeState::Unknown:
		return "unknown";
	case WifiRuntimeState::Disconnected:
		return "disconnected";
	case WifiRuntimeState::WaitingExternalConnection:
		return "waiting-external-connection";
	case WifiRuntimeState::Connected:
		return "connected";
	case WifiRuntimeState::CacheConnecting:
		return "cache-connecting";
	case WifiRuntimeState::ApAwaitClient:
		return "ap-await-client";
	case WifiRuntimeState::ApAwaitWeb:
		return "ap-await-web";
	case WifiRuntimeState::Error:
		return "error";
	}
	return "unknown";
}


} // namespace

namespace detail
{


ApClientScanner::ApClientScanner(WifiRuntimeController::ApClientProbe probe)
	: _probe(std::move(probe))
{
}

ApClientScanner::~ApClientScanner()
{
	shutdown();
}

void ApClientScanner::start()
{
	std::lock_guard<std::mutex> lock(_mutex);
	if (_closed || _thread.joinable()) {
		return;
	}
	_stop_requested = false;
	_has_client = false;
	_thread = std::thread([this]() {
		run();
	});
}

void ApClientScanner::stop()
{
	std::thread thread;
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (!_thread.joinable()) {
			return;
		}
		_stop_requested = true;
		_condition.notify_all();
		thread = std::move(_thread);
	}
	thread.join();
}

void ApClientScanner::shutdown()
{
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_closed = true;
	}
	stop();
}

bool ApClientScanner::has_client() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _has_client;
}

bool ApClientScanner::running() const
{
	std::lock_guard<std::mutex> lock(_mutex);
	return _thread.joinable();
}

void ApClientScanner::run()
{
	std::unique_lock<std::mutex> lock(_mutex);
	while (!_stop_requested) {
		lock.unlock();
		bool has_client = false;
		try {
			has_client = _probe && _probe();
		} catch (const std::exception &e) {
			spdlog::warn("WiFi AP client probe failed: {}", e.what());
		} catch (...) {
			spdlog::warn(
				"WiFi AP client probe failed with unknown exception");
		}
		lock.lock();
		if (_stop_requested) {
			break;
		}
		_has_client = has_client;
		_condition.wait_for(lock, kApClientPollInterval,
				    [this]() {
					    return _stop_requested;
					});
	}
}

WifiRuntimeControllerState::WifiRuntimeControllerState(
	WifiRuntimeController::Dependencies dependencies,
	bool boot_reconnect_pending,
	std::chrono::milliseconds saved_network_timeout)
	: dependencies(std::move(dependencies))
	, saved_network_timeout(saved_network_timeout)
	, scanner(this->dependencies.probe_ap_client)
{
	if (boot_reconnect_pending) {
		wifi_busy = true;
		wifi_boot_reconnect_running = true;
		wifi_runtime_state = WifiRuntimeState::CacheConnecting;
		wifi_status_text = "connecting...";
	}
}

using State = WifiRuntimeControllerState;

bool is_open_locked(const State &state)
{
	return !state.closed;
}

bool is_open(const std::shared_ptr<State> &state)
{
	std::lock_guard<std::mutex> lock(state->mutex);
	return is_open_locked(*state);
}

void set_wifi_runtime_state(State &state, WifiRuntimeState next_state,
			    std::string status_text,
			    std::string error)
{
	if (state.wifi_runtime_state != next_state) {
		spdlog::info(
			"[runtime] NanoKVM runtime WiFi state transition {} -> {} text='{}' error='{}'",
			wifi_runtime_state_name(state.wifi_runtime_state),
			wifi_runtime_state_name(next_state), status_text,
			error);
		state.wifi_state_entered_at = std::chrono::steady_clock::now();
	}
	state.wifi_runtime_state = next_state;
	state.wifi_status_text = std::move(status_text);
	state.wifi_error = std::move(error);
}

WifiRuntimeApCredentials
current_ap_credentials(const std::shared_ptr<State> &state)
{
	if (!state->dependencies.current_ap_credentials) {
		return {};
	}
	try {
		return state->dependencies.current_ap_credentials();
	} catch (const std::exception &e) {
		spdlog::warn("WiFi AP credential lookup failed: {}", e.what());
	} catch (...) {
		spdlog::warn(
			"WiFi AP credential lookup failed with unknown exception");
	}
	return {};
}

WifiRuntimeApCredentials
prepare_ap_credentials(const std::shared_ptr<State> &state)
{
	if (!state->dependencies.prepare_ap_credentials) {
		return current_ap_credentials(state);
	}
	try {
		return state->dependencies.prepare_ap_credentials();
	} catch (const std::exception &e) {
		spdlog::warn("WiFi AP credential preparation failed: {}",
			     e.what());
	} catch (...) {
		spdlog::warn(
			"WiFi AP credential preparation failed with unknown exception");
	}
	return {};
}

WifiSavedNetworkAvailability
has_saved_network_config(const std::shared_ptr<State> &state)
{
	if (!state->dependencies.has_saved_network_config) {
		return WifiSavedNetworkAvailability::Absent;
	}
	try {
		return state->dependencies.has_saved_network_config();
	} catch (const std::exception &e) {
		spdlog::warn("WiFi saved-network check failed: {}", e.what());
	} catch (...) {
		spdlog::warn(
			"WiFi saved-network check failed with unknown exception");
	}
	return WifiSavedNetworkAvailability::Unknown;
}

WifiRuntimeProjection
build_projection_locked(const State &state,
			const WifiRuntimeApCredentials &credentials,
			ProjectionKind kind)
{
	WifiRuntimeProjection projection;
	projection.wifi_ip = state.last_observation.wifi_ip;
	projection.wifi_supported = state.last_observation.wifi_supported;
	projection.wifi_connected = state.last_observation.wifi_connected;
	projection.wifi_ap_mode = state.last_observation.wifi_ap_mode;
	projection.wifi_ap_client_connected =
		state.last_observation.wifi_ap_client_connected;
	projection.wifi_qr_payload = credentials.qr_payload;
	projection.wifi_web_url = credentials.web_url;
	projection.wifi_runtime_state = state.wifi_runtime_state;
	projection.wifi_ssid = state.last_observation.wifi_ssid;
	projection.wifi_saved_network_config = state.wifi_saved_network_config;
	projection.wifi_status_text = state.wifi_status_text;
	projection.wifi_error = state.wifi_error;
	projection.wifi_busy = state.wifi_busy;
	projection.wifi_boot_reconnect_running =
		state.wifi_boot_reconnect_running;
	projection.wifi_show_qr =
		state.wifi_page_visible &&
		(state.wifi_runtime_state == WifiRuntimeState::ApAwaitClient ||
		 state.wifi_runtime_state == WifiRuntimeState::ApAwaitWeb);
	projection.wifi_ap_stop_completed_generation =
		state.wifi_ap_stop_completed_generation;
	projection.wifi_scan_running = state.wifi_scan_running;
	projection.wifi_scan_result = state.wifi_scan_result;
	projection.wifi_connect_running = state.wifi_connect_running;
	projection.wifi_connect_result = state.wifi_connect_result;
	projection.wifi_connect_message = state.wifi_connect_message;

	if (kind == ProjectionKind::Lifecycle) {
		projection.wifi_connected =
			state.wifi_runtime_state ==
				WifiRuntimeState::Connected &&
			state.last_observation.station_ip;
		projection.wifi_ap_mode =
			state.wifi_runtime_state ==
				WifiRuntimeState::ApAwaitClient ||
			state.wifi_runtime_state ==
				WifiRuntimeState::ApAwaitWeb;
		projection.wifi_ap_client_connected =
			state.wifi_runtime_state ==
			WifiRuntimeState::ApAwaitWeb;
	} else {
		if (state.wifi_runtime_state ==
			    WifiRuntimeState::ApAwaitClient ||
		    state.wifi_runtime_state == WifiRuntimeState::ApAwaitWeb) {
			projection.wifi_ap_mode = true;
		}
		if (state.wifi_runtime_state == WifiRuntimeState::Connected) {
			projection.wifi_connected =
				state.last_observation.station_ip;
			projection.wifi_ap_mode = false;
			projection.wifi_show_qr = false;
		}
		if (state.wifi_runtime_state ==
			    WifiRuntimeState::Disconnected ||
		    state.wifi_runtime_state == WifiRuntimeState::Error) {
			projection.wifi_connected = false;
			projection.wifi_ap_mode = false;
			projection.wifi_show_qr = false;
		}
	}

	if (kind == ProjectionKind::Lifecycle && !projection.wifi_connected) {
		projection.wifi_ssid.clear();
		projection.wifi_ip.clear();
	}
	return projection;
}

WifiRuntimeProjection make_projection(const std::shared_ptr<State> &state,
				      ProjectionKind kind)
{
	const WifiRuntimeApCredentials credentials =
		current_ap_credentials(state);
	std::lock_guard<std::mutex> lock(state->mutex);
	return build_projection_locked(*state, credentials, kind);
}

void publish_lifecycle(const std::shared_ptr<State> &state,
		       WifiRuntimeProjection projection,
		       WifiLifecyclePatch::Phase phase,
		       uint64_t operation_generation,
		       bool preserve_ap_stop_completion_when_stale)
{
	const auto &publisher = state->dependencies.publish_lifecycle;
	if (publisher) {
		publisher(projection, phase, operation_generation,
			  preserve_ap_stop_completion_when_stale);
	}
}

WifiScriptExecutionResult run_script(const std::shared_ptr<State> &state,
				     const std::vector<std::string> &args)
{
	if (!state->dependencies.run_script) {
		return {};
	}
	try {
		return state->dependencies.run_script(args);
	} catch (const std::exception &e) {
		spdlog::warn("WiFi script action failed: {}", e.what());
	} catch (...) {
		spdlog::warn(
			"WiFi script action failed with unknown exception");
	}
	return {};
}

WifiScriptExecutionResult stop_ap_config(const std::shared_ptr<State> &state)
{
	state->scanner.stop();
	return run_script(state, { "ap_off" });
}

bool operation_is_current_locked(const State &state,
				 uint64_t operation_generation)
{
	return is_open_locked(state) &&
	       operation_generation == state.wifi_operation_generation;
}

bool operation_can_complete_locked(const State &state,
				   uint64_t operation_generation)
{
	return operation_generation == state.wifi_operation_generation;
}

void discard_completed_async_tasks_locked(State &state)
{
	const auto first_completed = std::remove_if(
		state.async_tasks.begin(), state.async_tasks.end(),
		[](std::future<void> &task) {
			return !task.valid() ||
			       task.wait_for(std::chrono::milliseconds { 0 }) ==
				       std::future_status::ready;
		});
	state.async_tasks.erase(first_completed, state.async_tasks.end());
}

void submit_async_task(const std::shared_ptr<State> &state,
		       std::function<void()> task)
{
	std::lock_guard<std::mutex> lock(state->mutex);
	if (!is_open_locked(*state)) {
		return;
	}
	auto future = std::async(std::launch::async, [state,
						      task = std::move(
							      task)]() mutable {
		try {
			task();
		} catch (const std::exception &e) {
			spdlog::warn("WiFi async action failed: {}", e.what());
		} catch (...) {
			spdlog::warn(
				"WiFi async action failed with unknown exception");
		}
	});
	std::lock_guard<std::mutex> task_lock(state->task_mutex);
	discard_completed_async_tasks_locked(*state);
	state->async_tasks.push_back(std::move(future));
}

void run_enable_task(const std::shared_ptr<State> &state,
		     uint64_t operation_generation)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
	}
	const auto result = run_script(state, { "on" });
	const bool script_failed = !script_result_ok(result);

	WifiRuntimeProjection projection;
	bool should_publish = false;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		if (state->wifi_runtime_state ==
		    WifiRuntimeState::CacheConnecting) {
			if (contains_text(result.output,
					  "ERROR: NO_INTERFACE")) {
				state->wifi_busy = false;
				state->wifi_boot_reconnect_running = false;
				set_wifi_runtime_state(*state,
						       WifiRuntimeState::Error,
						       "No WiFi", "No WiFi");
			} else if (contains_text(result.output,
						 "ERROR: NO_SAVED_NETWORK")) {
				state->wifi_busy = false;
				state->wifi_boot_reconnect_running = false;
				set_wifi_runtime_state(
					*state, WifiRuntimeState::ApAwaitClient,
					"Connect to AP");
			} else if (script_failed) {
				state->wifi_busy = false;
				state->wifi_boot_reconnect_running = false;
				state->wifi_ap_opened_by_ui = false;
				set_wifi_runtime_state(
					*state, WifiRuntimeState::Error,
					"Enable failed",
					result.timed_out ? "Enable timed out" :
							   "Enable failed");
			} else {
				state->wifi_busy = true;
				state->wifi_state_entered_at =
					std::chrono::steady_clock::now();
				set_wifi_runtime_state(
					*state,
					WifiRuntimeState::CacheConnecting,
					"connecting...");
			}
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
		should_publish = true;
	}
	if (should_publish) {
		publish_lifecycle(state, std::move(projection),
				  WifiLifecyclePatch::Phase::Terminal,
				  operation_generation);
	}
}

void run_disable_task(const std::shared_ptr<State> &state,
		      uint64_t operation_generation)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
		++state->wifi_ap_config_generation;
		state->wifi_ap_config_running = false;
		state->wifi_ap_opened_by_ui = false;
	}
	const bool ap_stop_ok = script_result_ok(stop_ap_config(state));
	const auto result = run_script(state, { "off" });
	const bool disable_ok = script_result_ok(result);
	const bool ok = ap_stop_ok && disable_ok;
	if (!ap_stop_ok) {
		spdlog::warn(
			"NanoKVM runtime failed to stop WiFi AP before disabling WiFi");
	}
	if (!disable_ok) {
		spdlog::warn(
			"NanoKVM runtime failed to set WiFi enabled=false: {}",
			trim(result.output));
	}

	WifiRuntimeProjection projection;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		state->wifi_busy = false;
		state->wifi_boot_reconnect_running = false;
		state->wifi_ap_opened_by_ui = false;
		if (ok) {
			set_wifi_runtime_state(*state,
					       WifiRuntimeState::Disconnected,
					       "Disconnected");
		} else if (!ap_stop_ok) {
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "AP stop failed",
					       "AP stop failed");
		} else {
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "Disconnect failed",
					       result.timed_out ?
						       "Disconnect timed out" :
						       "Disconnect failed");
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Terminal,
			  operation_generation);
}

void run_reset_task(const std::shared_ptr<State> &state,
		    uint64_t operation_generation)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
		++state->wifi_ap_config_generation;
		state->wifi_ap_config_running = false;
		state->wifi_ap_opened_by_ui = false;
	}
	const bool ap_stop_ok = script_result_ok(stop_ap_config(state));
	const bool clear_ok = script_result_ok(run_script(state, { "clear" }));
	const bool off_ok = script_result_ok(run_script(state, { "off" }));
	const bool reset_ok = clear_ok && off_ok;
	const bool ok = ap_stop_ok && reset_ok;
	if (!ap_stop_ok) {
		spdlog::warn(
			"NanoKVM runtime failed to stop WiFi AP before resetting WiFi");
	}
	if (!reset_ok) {
		spdlog::warn("NanoKVM runtime failed to reset WiFi");
	}

	WifiRuntimeProjection projection;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		state->wifi_busy = false;
		state->wifi_boot_reconnect_running = false;
		state->wifi_ap_opened_by_ui = false;
		if (ok) {
			set_wifi_runtime_state(*state,
					       WifiRuntimeState::Disconnected,
					       "Disconnected");
		} else if (!ap_stop_ok) {
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "AP stop failed",
					       "AP stop failed");
		} else {
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "Reset failed", "Reset failed");
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Terminal,
			  operation_generation);
}

void complete_ap_stop_action(const std::shared_ptr<State> &state,
			     uint64_t generation, bool ok, bool timed_out,
			     bool ap_stop_attempted, bool reconnect_attempted,
			     bool reconnect_ok, bool reconnect_timed_out,
			     bool apply_terminal_state,
			     uint64_t operation_generation,
			     std::string_view origin)
{
	WifiRuntimeProjection projection;
	uint64_t completed_generation = 0;
	const bool reconnect_failed = reconnect_attempted && !reconnect_ok;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		state->wifi_ap_stop_completed_generation = std::max(
			state->wifi_ap_stop_completed_generation, generation);
		const bool pending_request =
			state->wifi_ap_stop_pending_generation == generation;
		const bool current_request =
			operation_generation ==
				state->wifi_operation_generation &&
			pending_request;
		if (pending_request) {
			state->wifi_ap_stop_pending_generation = 0;
		}
		if (current_request) {
			state->wifi_ap_opened_by_ui = false;
			if (apply_terminal_state) {
				state->wifi_boot_reconnect_running = false;
				if (ap_stop_attempted && !ok) {
					state->wifi_busy = false;
					set_wifi_runtime_state(
						*state, WifiRuntimeState::Error,
						"AP stop failed",
						timed_out ? "AP stop timed out" :
							    "AP stop failed");
				} else if (reconnect_failed) {
					state->wifi_busy = false;
					set_wifi_runtime_state(
						*state, WifiRuntimeState::Error,
						"Reconnect failed",
						reconnect_timed_out ?
							"Reconnect timed out" :
							"Reconnect failed");
				} else if (reconnect_attempted) {
					state->wifi_busy = true;
					set_wifi_runtime_state(
						*state,
						WifiRuntimeState::CacheConnecting,
						"connecting...");
				} else if (ap_stop_attempted) {
					state->wifi_busy = false;
					set_wifi_runtime_state(
						*state, WifiRuntimeState::Disconnected,
						"Disconnected");
				} else if (state->wifi_runtime_state !=
					   WifiRuntimeState::Error) {
					state->wifi_busy = false;
					state->wifi_status_text =
						"AP stop not required";
					state->wifi_error.clear();
				}
			}
		}
		completed_generation = state->wifi_ap_stop_completed_generation;
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	if (ok && !reconnect_failed) {
		spdlog::info(
			"[runtime] NanoKVM runtime WiFi AP stop complete origin={} request_generation={} completed_generation={} timed_out={} reconnect_attempted={}",
			origin, generation, completed_generation, timed_out,
			reconnect_attempted);
	} else if (reconnect_failed) {
		spdlog::warn(
			"NanoKVM runtime WiFi reconnect after AP stop failed origin={} request_generation={} completed_generation={} timed_out={}",
			origin, generation, completed_generation,
			reconnect_timed_out);
	} else {
		spdlog::warn(
			"NanoKVM runtime WiFi AP stop failed origin={} request_generation={} completed_generation={} timed_out={}",
			origin, generation, completed_generation, timed_out);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Terminal,
			  operation_generation, true);
}

void run_ap_stop_action(const std::shared_ptr<State> &state,
			uint64_t generation, uint64_t operation_generation,
			bool reconnect_after_stop,
			uint64_t reconnect_ap_config_generation,
			std::string origin)
{
	bool stale = false;
	bool current_session = false;
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!is_open_locked(*state)) {
			return;
		}
		stale = operation_generation !=
			state->wifi_operation_generation;
		current_session = reconnect_ap_config_generation ==
				  state->wifi_ap_config_generation;
	}
	if (stale || !current_session) {
		complete_ap_stop_action(state, generation, true, false, false,
					false, true, false, false,
					operation_generation, origin);
		return;
	}
	const auto ap_stop_result = stop_ap_config(state);
	const bool ap_stop_ok = script_result_ok(ap_stop_result);
	bool reconnect_attempted = false;
	bool reconnect_ok = true;
	bool reconnect_timed_out = false;
	bool reconnect_allowed = false;
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		current_session = operation_generation ==
					  state->wifi_operation_generation &&
				  reconnect_ap_config_generation ==
					  state->wifi_ap_config_generation;
		reconnect_allowed = current_session && is_open_locked(*state);
	}
	if (ap_stop_ok && reconnect_after_stop && reconnect_allowed) {
		const auto reconnect_result = run_script(state, { "on" });
		reconnect_attempted = true;
		reconnect_ok = script_result_ok(reconnect_result);
		reconnect_timed_out = reconnect_result.timed_out;
	}
	complete_ap_stop_action(state, generation, ap_stop_ok,
				ap_stop_result.timed_out, true,
				reconnect_attempted, reconnect_ok,
				reconnect_timed_out, current_session,
				operation_generation, origin);
}

void run_ap_start_action(const std::shared_ptr<State> &state,
			 uint64_t generation, uint64_t operation_generation)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
	}
	const WifiRuntimeApCredentials credentials =
		prepare_ap_credentials(state);
	(void)run_script(state, { "ap_off" });
	const auto result = run_script(state, { "ap", credentials.ssid,
						credentials.password });

	WifiRuntimeProjection projection;
	bool should_publish = false;
	bool should_stop_stale_ap = false;
	bool write_runtime_files = false;
	bool start_scanner = false;
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		const bool operation_current = operation_can_complete_locked(
			*state, operation_generation);
		const bool in_ap_state =
			state->wifi_runtime_state ==
				WifiRuntimeState::ApAwaitClient ||
			state->wifi_runtime_state ==
				WifiRuntimeState::ApAwaitWeb;
		const bool current_start =
			operation_current &&
			generation == state->wifi_ap_config_generation &&
			state->wifi_ap_config_running;
		if (!current_start) {
			if (operation_current &&
			    generation == state->wifi_ap_config_generation) {
				state->wifi_ap_config_running = false;
			}
			should_stop_stale_ap =
				!state->wifi_ap_config_running &&
				!in_ap_state &&
				state->wifi_runtime_state !=
					WifiRuntimeState::Connected &&
				state->wifi_ap_stop_pending_generation == 0;
		} else if (!script_result_ok(result)) {
			state->wifi_ap_config_running = false;
			state->wifi_ap_opened_by_ui = false;
			state->wifi_busy = false;
			state->wifi_boot_reconnect_running = false;
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "AP failed", "AP failed");
			should_stop_stale_ap =
				state->wifi_ap_stop_pending_generation == 0;
			should_publish = true;
		} else {
			state->wifi_ap_config_running = false;
			state->wifi_ap_opened_by_ui = true;
			state->wifi_busy = false;
			state->wifi_boot_reconnect_running = false;
			set_wifi_runtime_state(*state,
					       WifiRuntimeState::ApAwaitClient,
					       "Connect to AP");
			write_runtime_files = true;
			start_scanner = true;
			should_publish = true;
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	if (should_stop_stale_ap && is_open(state)) {
		(void)run_script(state, { "ap_off" });
	}
	if (write_runtime_files && is_open(state) &&
	    state->dependencies.write_ap_runtime_files) {
		state->dependencies.write_ap_runtime_files(
			credentials.password);
	}
	if (start_scanner && is_open(state)) {
		state->scanner.start();
	}
	if (should_publish) {
		publish_lifecycle(state, std::move(projection),
				  WifiLifecyclePatch::Phase::Terminal,
				  operation_generation);
	}
}

void run_scan_task(const std::shared_ptr<State> &state,
		   bool should_stop_ap_config, uint64_t operation_generation,
		   WifiRuntimeState previous_state,
		   std::string previous_status_text, std::string previous_error)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
	}
	const bool ap_stop_ok = !should_stop_ap_config ||
				script_result_ok(stop_ap_config(state));
	WifiScanResult scan_result;
	if (!ap_stop_ok) {
		spdlog::warn(
			"NanoKVM runtime failed to stop WiFi AP before scan");
		scan_result.ok = false;
		scan_result.error = "AP_STOP_FAILED";
	} else {
		const auto result = run_script(state, { "scan" });
		scan_result = parse_wifi_scan_output(result.output);
		if (!script_result_ok(result)) {
			scan_result.ok = false;
			scan_result.error = trim(result.output);
			if (scan_result.error.empty()) {
				scan_result.error = "SCAN_FAILED";
			}
		}
	}

	WifiRuntimeProjection projection;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		state->wifi_scan_running = false;
		state->wifi_scan_result = std::move(scan_result);
		if (!state->wifi_connect_running) {
			state->wifi_busy = false;
		}
		if (should_stop_ap_config) {
			if (ap_stop_ok) {
				set_wifi_runtime_state(
					*state, WifiRuntimeState::Disconnected,
					"Disconnected");
			} else {
				set_wifi_runtime_state(*state,
						       WifiRuntimeState::Error,
						       "AP stop failed",
						       "AP stop failed");
			}
		} else {
			set_wifi_runtime_state(*state, previous_state,
					       std::move(previous_status_text),
					       std::move(previous_error));
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Terminal,
			  operation_generation);
}

void run_connect_task(const std::shared_ptr<State> &state, std::string ssid,
		      std::string password, uint64_t operation_generation)
{
	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_is_current_locked(*state,
						 operation_generation)) {
			return;
		}
	}
	state->scanner.stop();

	std::vector<std::string> args { "connect", wifi_script_ssid_arg(ssid) };
	if (!password.empty()) {
		args.push_back(std::move(password));
	}
	const auto result = run_script(state, args);
	auto connect_result =
		parse_wifi_connect_output(result.output, result.exit_code);
	if (connect_result.status == WifiConnectStatus::Unknown) {
		connect_result.status = WifiConnectStatus::Failure;
	}

	const bool connected = connect_result.status ==
			       WifiConnectStatus::Success;
	std::string message;
	if (connected) {
		message = connect_result.ssid.empty() ?
				  "Connected" :
				  "Connected to " + connect_result.ssid;
	} else if (!connect_result.error_code.empty()) {
		message = connect_result.error_code;
	} else {
		message = connect_result.raw_summary.empty() ?
				  "Connect failed" :
				  connect_result.raw_summary;
	}

	WifiRuntimeProjection projection;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!operation_can_complete_locked(*state,
						   operation_generation)) {
			return;
		}
		state->wifi_connect_running = false;
		state->wifi_connect_result = std::move(connect_result);
		state->wifi_connect_message = message;
		state->wifi_busy = false;
		state->wifi_boot_reconnect_running = false;
		state->wifi_ap_opened_by_ui = false;
		if (connected) {
			set_wifi_runtime_state(
				*state, WifiRuntimeState::Connected, message);
		} else {
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "Connect failed", message);
		}
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Terminal,
			  operation_generation);
	if (connected && is_open(state) &&
	    state->dependencies.request_refresh) {
		state->dependencies.request_refresh();
	}
}

void tick_from_connected_locked(State &state,
				const WifiRuntimeObservation &observation)
{
	if (observation.wifi_connected) {
		state.wifi_boot_reconnect_running = false;
		set_wifi_runtime_state(state, WifiRuntimeState::Connected,
				       observation.wifi_ssid.empty() ?
					       "Connected" :
					       observation.wifi_ssid);
		state.wifi_busy = false;
		return;
	}
	state.wifi_busy = false;
	state.wifi_boot_reconnect_running = false;
	state.wifi_ap_opened_by_ui = false;
	set_wifi_runtime_state(state, WifiRuntimeState::Disconnected,
			       "Disconnected");
}

void tick_from_disconnected_locked(State &state,
				   const WifiRuntimeObservation &observation,
				   bool has_saved_network_config)
{
	if (observation.wifi_connected) {
		state.wifi_busy = false;
		state.wifi_boot_reconnect_running = false;
		state.wifi_ap_opened_by_ui = false;
		set_wifi_runtime_state(state, WifiRuntimeState::Connected,
				       observation.wifi_ssid.empty() ?
					       "Connected" :
					       observation.wifi_ssid);
		return;
	}
	if (observation.wifi_ap_mode) {
		state.wifi_boot_reconnect_running = false;
		if (observation.wifi_ap_client_connected ||
		    state.wifi_runtime_state == WifiRuntimeState::ApAwaitWeb) {
			set_wifi_runtime_state(state,
					       WifiRuntimeState::ApAwaitWeb,
					       "Configure WiFi");
		} else {
			set_wifi_runtime_state(state,
					       WifiRuntimeState::ApAwaitClient,
					       "Connect to AP");
		}
		state.wifi_busy = false;
		return;
	}
	if (state.wifi_runtime_state == WifiRuntimeState::Unknown &&
	    observation.wifi_supported && has_saved_network_config) {
		state.wifi_busy = true;
		state.wifi_boot_reconnect_running = true;
		set_wifi_runtime_state(state, WifiRuntimeState::CacheConnecting,
				       "connecting...");
		return;
	}
	state.wifi_busy = false;
	state.wifi_boot_reconnect_running = false;
	set_wifi_runtime_state(state, WifiRuntimeState::Disconnected,
			       "Disconnected");
}

void tick_from_cache_connecting_locked(
	State &state, const WifiRuntimeObservation &observation)
{
	if (observation.wifi_connected) {
		state.wifi_busy = false;
		state.wifi_boot_reconnect_running = false;
		state.wifi_ap_opened_by_ui = false;
		set_wifi_runtime_state(state, WifiRuntimeState::Connected,
				       observation.wifi_ssid.empty() ?
					       "Connected" :
					       observation.wifi_ssid);
		return;
	}
	const auto elapsed =
		std::chrono::steady_clock::now() - state.wifi_state_entered_at;
	if (elapsed >= state.saved_network_timeout) {
		state.wifi_busy = false;
		state.wifi_boot_reconnect_running = false;
		set_wifi_runtime_state(state, WifiRuntimeState::ApAwaitClient,
				       "Connect to AP");
	} else {
		state.wifi_busy = true;
		set_wifi_runtime_state(state, WifiRuntimeState::CacheConnecting,
				       "connecting...");
	}
}

bool tick_from_ap_mode_locked(State &state,
			      const WifiRuntimeObservation &observation)
{
	bool stop_scanner = false;
	if (state.scanner.running() && state.scanner.has_client()) {
		stop_scanner = true;
		set_wifi_runtime_state(state, WifiRuntimeState::ApAwaitWeb,
				       "Configure WiFi");
	} else if (observation.wifi_ap_client_connected) {
		set_wifi_runtime_state(state, WifiRuntimeState::ApAwaitWeb,
				       "Configure WiFi");
	} else if (state.wifi_runtime_state != WifiRuntimeState::ApAwaitWeb) {
		set_wifi_runtime_state(state, WifiRuntimeState::ApAwaitClient,
				       "Connect to AP");
	}
	if (observation.wifi_connected) {
		stop_scanner = true;
		state.wifi_busy = false;
		state.wifi_boot_reconnect_running = false;
		set_wifi_runtime_state(state, WifiRuntimeState::Connected,
				       observation.wifi_ssid.empty() ?
					       "Connected" :
					       observation.wifi_ssid);
		return stop_scanner;
	}
	state.wifi_busy = false;
	return stop_scanner;
}

void tick_from_error_locked(State &state,
			    const WifiRuntimeObservation &observation)
{
	if (observation.wifi_connected) {
		state.wifi_busy = false;
		state.wifi_boot_reconnect_running = false;
		state.wifi_ap_opened_by_ui = false;
		set_wifi_runtime_state(state, WifiRuntimeState::Connected,
				       observation.wifi_ssid.empty() ?
					       "Connected" :
					       observation.wifi_ssid);
		return;
	}
	state.wifi_busy = false;
	state.wifi_boot_reconnect_running = false;
}

uint64_t request_ap_stop(const std::shared_ptr<State> &state,
			 std::string origin)
{
	WifiRuntimeProjection projection;
	bool needs_ap_stop = false;
	bool reconnect_after_stop = false;
	uint64_t request_generation = 0;
	uint64_t operation_generation = 0;
	uint64_t reconnect_ap_config_generation = 0;
	{
		const WifiRuntimeApCredentials credentials =
			current_ap_credentials(state);
		std::lock_guard<std::mutex> lock(state->mutex);
		if (!is_open_locked(*state)) {
			return 0;
		}
		if (state->wifi_ap_stop_pending_generation != 0) {
			return state->wifi_ap_stop_pending_generation;
		}
		const bool ap_cleanup_owned = state->wifi_ap_opened_by_ui ||
					      state->wifi_ap_config_running;
		request_generation = ++state->wifi_ap_stop_request_generation;
		operation_generation = ++state->wifi_operation_generation;
		state->wifi_ap_stop_pending_generation = request_generation;
		state->wifi_busy = true;
		state->wifi_boot_reconnect_running = false;
		reconnect_ap_config_generation =
			++state->wifi_ap_config_generation;
		state->wifi_ap_config_running = false;
		needs_ap_stop = ap_cleanup_owned;
		reconnect_after_stop = ap_cleanup_owned &&
				       state->wifi_reconnect_after_qr_exit;
		state->wifi_reconnect_after_qr_exit = false;
		state->wifi_ap_opened_by_ui = false;
		state->wifi_status_text = "Stopping AP...";
		state->wifi_error.clear();
		projection = build_projection_locked(*state, credentials,
						     ProjectionKind::Lifecycle);
	}
	publish_lifecycle(state, std::move(projection),
			  WifiLifecyclePatch::Phase::Pending,
			  operation_generation);
	if (!needs_ap_stop) {
		complete_ap_stop_action(state, request_generation, true, false,
					false, false, true, false, true,
					operation_generation, origin + "-noop");
		return request_generation;
	}

	const auto &enqueue = state->dependencies.enqueue_serial_work;
	const bool queued =
		enqueue &&
		enqueue("wifi_qr_ap_stop", [state, request_generation,
					    operation_generation, reconnect_after_stop,
					    reconnect_ap_config_generation,
					    origin = std::move(origin)]() {
			run_ap_stop_action(state, request_generation,
					   operation_generation, reconnect_after_stop,
					   reconnect_ap_config_generation, origin);
		});
	if (queued) {
		return request_generation;
	}

	{
		std::lock_guard<std::mutex> lock(state->mutex);
		if (operation_is_current_locked(*state, operation_generation)) {
			state->wifi_ap_config_running = false;
			state->wifi_ap_opened_by_ui = false;
			state->wifi_busy = false;
			state->wifi_boot_reconnect_running = false;
			set_wifi_runtime_state(*state, WifiRuntimeState::Error,
					       "AP stop unavailable",
					       "AP stop unavailable");
		}
	}
	spdlog::warn("NanoKVM runtime QR AP stop queue rejected");
	complete_ap_stop_action(state, request_generation, false, false, false,
					false, true, false, true, operation_generation,
					"queue-rejected");
	return request_generation;
}


} // namespace detail

} // namespace lvgl_main
