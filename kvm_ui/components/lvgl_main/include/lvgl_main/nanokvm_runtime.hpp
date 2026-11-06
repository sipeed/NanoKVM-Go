#pragma once

#include <atomic>
#include <array>
#include <chrono>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <queue>
#include <string>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/device_region.hpp"
#include "lvgl_main/http_api_contract.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"
#include "lvgl_main/package_version.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "lvgl_main/runtime_state_store.hpp"
#include "lvgl_main/runtime_thread_pool.hpp"
#include "lvgl_main/wifi_script_parser.hpp"
#include "lvgl_main/wifi_runtime_state.hpp"
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include "ui/background_assets.hpp"
#endif

namespace lvgl_main
{

class WifiRuntimeController;
class NanoKvmServerTransport;
class TailscaleClient;
struct TailscaleSession;
struct TailscaleSnapshot;
enum class TailscaleAction : uint8_t;

std::string default_tailscale_socket_path();

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
struct BackgroundActionResult {
	bool ok { false };
	int http_status { 500 };
	std::string detail;
};
#endif

class NanoKvmRuntimeService final : public RuntimeService {
    public:
	using PackageVersionLoader = StartupPackageVersionCache::Loader;
	using FactoryResetRunner =
		std::function<bool(std::string_view script_path)>;
	using RegionSwitchRunner = std::function<bool(DeviceRegion)>;

	NanoKvmRuntimeService(
		AppConfig config, RuntimeCallbacks callbacks = {},
		std::filesystem::path config_path = std::filesystem::path(
			app_config_defaults::kConfigPath.data()),
		PackageVersionLoader package_version_loader = {},
		FactoryResetRunner factory_reset_runner = {},
		NcmBootMarkerPaths ncm_marker_paths =
			production_ncm_boot_marker_paths(),
		std::filesystem::path region_path =
			std::filesystem::path(kDeviceRegionPath),
		RegionSwitchRunner region_switch_runner = {},
		std::string tailscale_socket_path =
			default_tailscale_socket_path());
	~NanoKvmRuntimeService() override;

	NanoKvmRuntimeService(const NanoKvmRuntimeService &) = delete;
	NanoKvmRuntimeService &
	operator=(const NanoKvmRuntimeService &) = delete;

	bool start() override;
	bool start_settings_cli_actions();
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
	void set_video_enabled(bool enabled);
	void apply_backlight_percent_now(int percent) override;
	void set_backlight_percent(int percent);
	void set_auto_sleep_enabled(bool enabled) override;
	void set_ui_auto_rotation_enabled(bool enabled) override;
	void set_ui_manual_display_rotation(bool enabled) override;
	void set_screen_sleep(bool sleep) override;
	void set_wifi_ap_enabled(bool enabled);
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
	void update_display_labels(http_api::DisplayLabelsUpdate update);
	void update_hid_state(http_api::HidStateUpdate update);
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	BackgroundActionResult
	apply_background_selection(std::string image_name);
#endif
	bool request_factory_reset() override;
	bool request_region_switch(DeviceRegion target) override;
	bool request_tailscale_start() override;
	bool request_tailscale_exit() override;
	void about_reset_no_op() const;

	bool settings_cli_set_video_enabled(bool enabled, std::string &error);
	bool settings_cli_set_hid_enabled(bool enabled, std::string &error);
	bool settings_cli_set_ncm_enabled(bool enabled, std::string &error);
	bool settings_cli_set_ssh_enabled(bool enabled, std::string &error);
	bool settings_cli_set_wifi_ap_enabled(bool enabled, std::string &error);
	bool settings_cli_reset_wifi(std::string &error);
	bool settings_cli_set_backlight_percent(
		int percent,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error);
	bool settings_cli_set_auto_sleep_enabled(
		bool enabled,
		std::vector<std::pair<std::string, std::string>> &fields,
		std::string &error);

    private:
	using Action = std::function<void()>;
	using SyncAction = std::function<bool(std::string &)>;
	struct QueuedAction {
		Action run;
		bool request_refresh { true };
		bool reschedule_display_metrics { false };
	};
	enum class BoolActionKey : uint8_t {
		Video,
		Hid,
		Ncm,
		AutoSleep,
		ScreenSleep,
		Count,
	};
	struct BoolActionState {
		bool queued { false };
		bool running { false };
		bool target { false };
		uint64_t generation { 0 };
	};
	struct ConfigPersistWork {
		AppConfig config;
		uint64_t generation { 0 };
		std::string reason;
	};
	enum class LocalServiceActionKind : uint8_t {
		Refresh,
		SetMcpBundle,
		SetScreenTimelapse,
	};
	struct QueuedLocalServiceAction {
		LocalServiceActionKind kind { LocalServiceActionKind::Refresh };
		bool target_enabled { false };
	};
	using BoolActionRead = std::function<bool()>;
	using BoolActionApply = std::function<bool(bool)>;
	static constexpr std::size_t kBoolActionKeyCount =
		static_cast<std::size_t>(BoolActionKey::Count);

	void worker_loop();
	bool start_workers(bool enable_periodic_refresh,
			   bool enable_background_workers);
	bool enqueue_action(Action action);
	bool enqueue_action(const char *name, Action action,
			    bool request_refresh = true,
			    bool reschedule_display_metrics = false);
	void enqueue_bool_action(BoolActionKey key, const char *name,
				 bool target, BoolActionRead current,
				 BoolActionApply apply,
				 bool request_refresh = true);
	void run_bool_action(BoolActionKey key, const char *name,
			     BoolActionRead current, BoolActionApply apply);
	bool take_bool_action_target(BoolActionKey key, bool &target,
				     uint64_t &generation);
	void finish_bool_action(BoolActionKey key, uint64_t generation);
	void schedule_config_persist(std::string reason);
	bool persist_config_now(std::string_view reason, std::string &error);
	bool run_settings_cli_sync_action(const char *name, SyncAction action,
					  std::string &error);
	void submit_config_persist_snapshot(uint64_t generation,
					    std::string reason);
	void flush_config_persist();
	bool start_config_persist_worker();
	void stop_config_persist_worker();
	void config_persist_loop();
	void refresh_status();

	void collect_sysinfo_static();
	void launch_local_service_action(LocalServiceActionKind kind,
					 bool target_enabled = false);
	void merge_local_service_result(uint64_t generation,
					LocalServiceActionKind kind,
					bool target_enabled, bool ok,
					int status_code, bool mcp_supported,
					bool mcp_active, bool replay_supported,
					bool replay_active, std::string error,
					int64_t duration_ms);
	std::chrono::milliseconds refresh_display_metrics();

	std::chrono::steady_clock::time_point
	refresh_tailscale_tick(bool try_code, uint8_t &heartbeat_failure_count,
			       uint8_t &code_failure_count,
			       std::chrono::steady_clock::time_point
				       &next_code_refresh);

	void publish_tailscale_snapshot(const TailscaleSnapshot &snapshot,
					uint64_t action_generation);
	void publish_tailscale_snapshot_failure(uint64_t action_generation);
	bool launch_tailscale_action(TailscaleAction action);
	void merge_tailscale_action_result(uint64_t generation,
					   TailscaleAction action, bool ok,
					   std::string message);

	void publish_tailscale_session(const TailscaleSession &session,
				       bool force);
	void launch_systemd_status_refresh();
	void merge_systemd_status(uint64_t generation, RuntimeStatus status,
				  int64_t duration_ms, bool ok,
				  std::string error);
	void merge_display_metrics(uint64_t generation, RuntimeStatus status,
				   int64_t duration_ms, bool ok,
				   std::string error);
	void publish_local_runtime_status_snapshot();
	void observe_initial_ncm_state();
	void publish_factory_reset_failure();
	void publish_region_switch_failure();
	void publish_status(RuntimeStatusPatch patch,
			    bool notify_callback = true);
	void
	publish_wifi_status(const RuntimeStatus &status,
			    WifiLifecyclePatch::Phase phase =
				    WifiLifecyclePatch::Phase::Observation,
			    uint64_t operation_generation = 0,
			    bool preserve_ap_stop_completion_when_stale = false,
			    bool notify_callback = true);

	AppConfig _config;
	std::filesystem::path _config_path;
	std::filesystem::path _region_path;
	std::optional<DeviceRegion> _device_region;
	RuntimeCallbacks _callbacks;

	std::unique_ptr<NanoKvmServerTransport> _server_transport;

	std::string _tailscale_socket_path;
	std::unique_ptr<TailscaleClient> _tailscale_client;
	RuntimeThreadPool _runtime_thread_pool;
	const std::string _backlight_root;
	StartupPackageVersionCache _app_version_cache;
	FactoryResetRunner _factory_reset_runner;
	RegionSwitchRunner _region_switch_runner;

	RuntimeStateStore _state_store;

	std::mutex _queue_mutex;
	std::condition_variable _queue_cv;
	std::queue<QueuedAction> _actions;
	std::array<BoolActionState, kBoolActionKeyCount> _bool_action_states {};
	std::thread _worker;
	std::atomic<bool> _running { false };
	std::atomic<bool> _factory_reset_requested { false };
	std::atomic<uint64_t> _factory_reset_failure_generation { 0 };
	std::atomic<bool> _region_switch_requested { false };
	std::atomic<uint64_t> _region_switch_failure_generation { 0 };
	bool _periodic_refresh_enabled { true };
	std::atomic<bool> _auto_sleep_enabled { false };
	std::atomic<bool> _video_enabled { false };
	std::atomic<bool> _hid_enabled { false };
	std::atomic<bool> _ncm_enabled { false };
	std::atomic<bool> _ncm_supported { false };
	NcmBootMarkerPaths _ncm_marker_paths;
	std::atomic<int> _backlight_percent {
		app_config_defaults::kNanokvmBacklightPercent
	};
	std::atomic<bool> _screen_sleeping { false };
	bool _config_persist_scheduled { false };
	uint64_t _config_persist_generation { 0 };
	std::chrono::steady_clock::time_point _config_persist_deadline {};
	uint64_t _config_persist_deadline_generation { 0 };
	std::string _config_persist_deadline_reason;

	std::mutex _config_persist_mutex;
	std::condition_variable _config_persist_cv;
	std::optional<ConfigPersistWork> _config_persist_work;
	std::thread _config_persist_worker;
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	std::mutex _background_action_mutex;
#endif
	bool _config_persist_accepting { false };
	bool _config_persist_stop_requested { false };
	std::chrono::steady_clock::time_point _last_systemd_refresh {};
	std::chrono::steady_clock::time_point _last_hardware_version_refresh {};
	std::chrono::steady_clock::time_point _last_wifi_refresh {};
	std::chrono::steady_clock::time_point _last_storage_refresh {};
	std::chrono::steady_clock::time_point _last_wifi_signal_refresh {};
	std::atomic<bool> _more_page_visible { false };

	std::atomic<bool> _tailscale_page_visible { false };

	std::atomic<bool> _tailscale_relinquish_pending { false };
	std::atomic<bool> _tailscale_refresh_now { false };
	std::atomic<uint64_t> _tailscale_action_generation { 0 };
	bool _tailscale_action_pending { false };

	struct SysinfoStaticCache {
		std::string device_key;
		std::string hostname;
		std::string device_number;
		std::string image_version;
		std::string edid_name;
		std::string mac_address;
		std::string mdns_name;
	};
	SysinfoStaticCache _sysinfo_static_cache {};
	bool _sysinfo_static_loaded { false };
	uint64_t _systemd_refresh_generation { 0 };
	bool _systemd_refresh_pending { false };
	uint64_t _display_metrics_refresh_generation { 0 };
	bool _display_metrics_refresh_pending { false };
	std::chrono::steady_clock::time_point _display_metrics_backoff_until {};
	uint8_t _display_metrics_failure_count { 0 };
	uint64_t _local_services_generation { 0 };
	bool _local_services_pending { false };
	bool _local_services_refresh_after_pending { false };
	std::queue<QueuedLocalServiceAction> _local_services_queued_actions;

	std::unique_ptr<WifiRuntimeController> _wifi_runtime_controller;
};

}
