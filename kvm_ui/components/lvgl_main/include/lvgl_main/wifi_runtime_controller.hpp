#pragma once

#include <chrono>
#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <string_view>
#include <vector>

#include "lvgl_main/runtime_state_store.hpp"
#include "lvgl_main/wifi_runtime_state.hpp"
#include "lvgl_main/wifi_script_parser.hpp"

namespace lvgl_main
{

struct WifiScriptExecutionResult {
	int exit_code { -1 };
	std::string output;
	bool timed_out { false };
};

struct WifiRuntimeApCredentials {
	std::string ssid;
	std::string password;
	std::string qr_payload;
	std::string web_url;
};

struct WifiRuntimeObservation {
	bool wifi_supported { false };
	bool wifi_connected { false };
	bool wifi_ap_mode { false };
	bool wifi_ap_client_connected { false };
	bool station_ip { false };
	std::string wifi_ip;
	std::string wifi_ssid;
	std::string wifi_error;
};

struct WifiRuntimeProjection {
	std::string wifi_ip;
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

	void apply_to(RuntimeStatus &status) const;
};

namespace detail
{
struct WifiRuntimeControllerState;
}

class WifiRuntimeController final {
    public:
	using ScriptRunner = std::function<WifiScriptExecutionResult(
		const std::vector<std::string> &args)>;
	using ApCredentialsProvider = std::function<WifiRuntimeApCredentials()>;
	using ApRuntimeFileWriter =
		std::function<void(std::string_view password)>;
	using ApClientProbe = std::function<bool()>;
	using SavedNetworkConfigProbe = std::function<WifiSavedNetworkAvailability()>;
	using SerialWorkEnqueue = std::function<bool(
		std::string_view name, std::function<void()> work)>;
	using RefreshRequest = std::function<void()>;
	using LifecyclePublisher = std::function<void(
		const WifiRuntimeProjection &projection,
		WifiLifecyclePatch::Phase phase, uint64_t operation_generation,
		bool preserve_ap_stop_completion_when_stale)>;

	struct Dependencies {
		ScriptRunner run_script;
		ApCredentialsProvider current_ap_credentials;
		ApCredentialsProvider prepare_ap_credentials;
		ApRuntimeFileWriter write_ap_runtime_files;
		ApClientProbe probe_ap_client;
		SavedNetworkConfigProbe has_saved_network_config;
		SerialWorkEnqueue enqueue_serial_work;
		RefreshRequest request_refresh;
		LifecyclePublisher publish_lifecycle;
	};

	WifiRuntimeController(Dependencies dependencies,
			      bool boot_reconnect_pending,
			      std::chrono::milliseconds saved_network_timeout);
	~WifiRuntimeController();

	WifiRuntimeController(const WifiRuntimeController &) = delete;
	WifiRuntimeController &
	operator=(const WifiRuntimeController &) = delete;

	void close();
	bool is_closed() const;
	uint64_t operation_generation() const;

	WifiRuntimeProjection current_projection() const;
	WifiRuntimeProjection
	project_observation(WifiRuntimeObservation observation);

	void set_enabled(bool enabled);
	void reset();
	void scan_networks();
	void connect_network(std::string ssid, std::string password = {});
	void start_qr_ap_config();
	uint64_t stop_qr_ap_config();
	void stop_qr_ap_config_after_success();
	void notify_page_visible(bool visible);
	void notify_qr_dismissed();
	std::string qr_payload() const;
	bool set_enabled_sync(bool enabled, std::string &error);
	bool reset_sync(std::string &error);

    private:
	std::shared_ptr<detail::WifiRuntimeControllerState> _state;
};

} // namespace lvgl_main
