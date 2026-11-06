#pragma once

#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <future>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

#include "lvgl_main/wifi_runtime_controller.hpp"

namespace lvgl_main
{

namespace detail
{

enum class ProjectionKind : uint8_t {
	Lifecycle,
	Observation,
};

class ApClientScanner final {
    public:
	explicit ApClientScanner(WifiRuntimeController::ApClientProbe probe);
	~ApClientScanner();

	ApClientScanner(const ApClientScanner &) = delete;
	ApClientScanner &operator=(const ApClientScanner &) = delete;

	void start();
	void stop();
	void shutdown();
	bool has_client() const;
	bool running() const;

    private:
	void run();

	WifiRuntimeController::ApClientProbe _probe;
	mutable std::mutex _mutex;
	std::condition_variable _condition;
	std::thread _thread;
	bool _stop_requested { false };
	bool _closed { false };
	bool _has_client { false };
};

struct WifiRuntimeControllerState {
	explicit WifiRuntimeControllerState(
		WifiRuntimeController::Dependencies dependencies,
		bool boot_reconnect_pending,
		std::chrono::milliseconds saved_network_timeout);

	mutable std::mutex mutex;
	std::mutex task_mutex;
	WifiRuntimeController::Dependencies dependencies;
	std::chrono::milliseconds saved_network_timeout;
	ApClientScanner scanner;
	bool closed { false };
	WifiRuntimeObservation last_observation;
	bool wifi_saved_network_config { false };
	WifiRuntimeState wifi_runtime_state { WifiRuntimeState::Unknown };
	std::chrono::steady_clock::time_point wifi_state_entered_at {
		std::chrono::steady_clock::now()
	};
	std::string wifi_status_text;
	std::string wifi_error;
	bool wifi_busy { false };
	bool wifi_boot_reconnect_running { false };
	bool wifi_ap_opened_by_ui { false };
	bool wifi_ap_config_running { false };
	bool wifi_reconnect_after_qr_exit { false };
	uint64_t wifi_operation_generation { 0 };
	uint64_t wifi_ap_config_generation { 0 };
	uint64_t wifi_ap_stop_request_generation { 0 };
	uint64_t wifi_ap_stop_completed_generation { 0 };
	uint64_t wifi_ap_stop_pending_generation { 0 };
	bool wifi_page_visible { false };
	bool wifi_scan_running { false };
	WifiScanResult wifi_scan_result;
	bool wifi_connect_running { false };
	WifiConnectResult wifi_connect_result;
	std::string wifi_connect_message;
	std::vector<std::future<void>> async_tasks;
};

bool is_open_locked(const WifiRuntimeControllerState &state);
bool is_open(const std::shared_ptr<WifiRuntimeControllerState> &state);

void set_wifi_runtime_state(WifiRuntimeControllerState &state,
			    WifiRuntimeState next_state,
			    std::string status_text = {},
			    std::string error = {});

bool operation_is_current_locked(const WifiRuntimeControllerState &state,
				 uint64_t operation_generation);

void tick_from_connected_locked(
	WifiRuntimeControllerState &state,
	const WifiRuntimeObservation &observation);
void tick_from_disconnected_locked(
	WifiRuntimeControllerState &state,
	const WifiRuntimeObservation &observation,
	bool has_saved_network_config);
void tick_from_cache_connecting_locked(
	WifiRuntimeControllerState &state,
	const WifiRuntimeObservation &observation);
bool tick_from_ap_mode_locked(
	WifiRuntimeControllerState &state,
	const WifiRuntimeObservation &observation);
void tick_from_error_locked(
	WifiRuntimeControllerState &state,
	const WifiRuntimeObservation &observation);

WifiRuntimeApCredentials current_ap_credentials(
	const std::shared_ptr<WifiRuntimeControllerState> &state);
WifiSavedNetworkAvailability has_saved_network_config(
	const std::shared_ptr<WifiRuntimeControllerState> &state);

WifiRuntimeProjection build_projection_locked(
	const WifiRuntimeControllerState &state,
	const WifiRuntimeApCredentials &credentials, ProjectionKind kind);
WifiRuntimeProjection make_projection(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	ProjectionKind kind);

void publish_lifecycle(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	WifiRuntimeProjection projection, WifiLifecyclePatch::Phase phase,
	uint64_t operation_generation,
	bool preserve_ap_stop_completion_when_stale = false);

WifiScriptExecutionResult run_script(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	const std::vector<std::string> &args);

void submit_async_task(const std::shared_ptr<WifiRuntimeControllerState> &state,
			      std::function<void()> task);
void run_enable_task(const std::shared_ptr<WifiRuntimeControllerState> &state,
			     uint64_t operation_generation);
void run_disable_task(const std::shared_ptr<WifiRuntimeControllerState> &state,
			      uint64_t operation_generation);
void run_reset_task(const std::shared_ptr<WifiRuntimeControllerState> &state,
			    uint64_t operation_generation);
void run_scan_task(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	bool should_stop_ap_config, uint64_t operation_generation,
	WifiRuntimeState previous_state, std::string previous_status_text,
	std::string previous_error);
void run_connect_task(
	const std::shared_ptr<WifiRuntimeControllerState> &state, std::string ssid,
	std::string password, uint64_t operation_generation);
void run_ap_start_action(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	uint64_t generation, uint64_t operation_generation);

uint64_t request_ap_stop(
	const std::shared_ptr<WifiRuntimeControllerState> &state,
	std::string origin);

} // namespace detail

} // namespace lvgl_main
