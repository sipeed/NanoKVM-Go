#include "lvgl_main/nanokvm_runtime.hpp"
#include "nanokvm_runtime_internal.hpp"

#include <atomic>
#include <chrono>
#include <exception>
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
#include <spdlog/spdlog.h>

#include "lvgl_main/backlight_control.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"
#include "lvgl_main/wifi_runtime_controller.hpp"
#include "lvgl_main/wifi_script_parser.hpp"
#include "nanokvm_server_transport.hpp"
#include "nanokvm_tailscale_client.hpp"
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include "ui/background_assets.hpp"
#endif
#include "ui/local_service_state_client.hpp"

namespace lvgl_main
{

std::string default_tailscale_socket_path()
{
	return std::string(kTailscaleSocketPath);
}

namespace
{

constexpr std::chrono::seconds kFactoryResetCommandTimeout { 30 };
constexpr std::string_view kResetToDefaultScriptPath =
	"/kvmcomm/scripts/reset_to_default.sh";
StartupPackageVersionCache::Loader
make_package_version_loader(StartupPackageVersionCache::Loader loader)
{
	if (loader) {
		return loader;
	}

	return []() -> std::optional<std::string> {
		try {
			return detail::read_kvmcomm_package_version();
		} catch (const std::exception &e) {
			spdlog::warn(
				"NanoKVM runtime kvmcomm package version lookup failed: {}",
				e.what());
		} catch (...) {
			spdlog::warn(
				"NanoKVM runtime kvmcomm package version lookup failed with unknown exception");
		}
		return std::nullopt;
	};
}

bool set_backlight_percent_impl(const NanokvmRuntimeConfig &config, int percent)
{
	const auto result = backlight::write_backlight_percent(
		config.backlight_root, percent, config.backlight_invert);
	if (!result.ok) {
		spdlog::warn(
			"NanoKVM runtime failed to write backlight brightness: {}",
			result.error);
	}
	return result.ok;
}

bool set_screen_sleep_impl(const NanokvmRuntimeConfig &config, bool sleep,
			   int percent)
{
	const auto result = sleep ? backlight::write_backlight_off(
					    config.backlight_root,
					    config.backlight_invert) :
				    backlight::write_backlight_percent(
					    config.backlight_root, percent,
					    config.backlight_invert);
	if (!result.ok) {
		spdlog::warn(
			"NanoKVM runtime failed to write screen sleep={}: {}",
			sleep, result.error);
	}
	return result.ok;
}

}

NanoKvmRuntimeService::NanoKvmRuntimeService(
	AppConfig config, RuntimeCallbacks callbacks,
	std::filesystem::path config_path,
	PackageVersionLoader package_version_loader,
	FactoryResetRunner factory_reset_runner,
	NcmBootMarkerPaths ncm_marker_paths,
	std::filesystem::path region_path,
	RegionSwitchRunner region_switch_runner,
	std::string tailscale_socket_path)
	: _config(std::move(config))
	, _config_path(std::move(config_path))
	, _region_path(std::move(region_path))
	, _callbacks(std::move(callbacks))
	, _tailscale_socket_path(std::move(tailscale_socket_path))
	, _backlight_root(_config.nanokvm.backlight_root)
	, _app_version_cache(make_package_version_loader(
		  std::move(package_version_loader)))
	, _factory_reset_runner(std::move(factory_reset_runner))
	, _region_switch_runner(std::move(region_switch_runner))
	, _auto_sleep_enabled(_config.nanokvm.auto_sleep_enabled)
	, _video_enabled(_config.nanokvm.video_enabled)
	, _hid_enabled(_config.nanokvm.hid_enabled)
	, _ncm_enabled { false }
	, _ncm_supported { false }
	, _ncm_marker_paths(std::move(ncm_marker_paths))
	, _backlight_percent(backlight::clamp_backlight_percent(
		  _config.nanokvm.backlight_percent))
{
	observe_initial_ncm_state();
	_server_transport = std::make_unique<NanoKvmServerTransport>(
		_config.nanokvm,
		[config = _config.nanokvm](std::string_view path) {
			return detail::http_request_json(config, "GET", path);
		},
		std::filesystem::path(kNanoKvmUiSocketPath),
		kNanoKvmUiProbeInterval,
		[config = _config.nanokvm](std::string_view path,
					   const nlohmann::json &body) {
			return detail::http_post_json(config, path, body);
		});
	_tailscale_client =
		std::make_unique<TailscaleClient>(_tailscale_socket_path);

	if (!_factory_reset_runner) {
		_factory_reset_runner = [](std::string_view script_path) {
			if (!detail::validate_executable_script(
				    script_path, "factory reset script")) {
				return false;
			}
			return detail::run_success({ std::string(script_path) },
						    kFactoryResetCommandTimeout,
						    "restore factory defaults");
		};
	}

	if (!_region_switch_runner) {
		_region_switch_runner = [this](DeviceRegion target) {
			if (!_server_transport) {
				return false;
			}
			const auto response = _server_transport->post_json(
				NanoKvmServerTransportLane::Status,
				"/api/application/region",
				nlohmann::json {
					{ "region",
					  std::string(device_region_name(target)) },
				});
			if (!response.has_value()) {
				return false;
			}
			return detail::region_switch_response_ok(*response);
		};
	}

	const bool boot_reconnect_pending =
		detail::query_wifi_boot_enabled(_config.nanokvm);
	_wifi_runtime_controller = std::make_unique<WifiRuntimeController>(
		WifiRuntimeController::Dependencies {
			.run_script =
				[this](const std::vector<std::string> &args) {
					const auto result = detail::run_wifi_script(
						_config.nanokvm, args);
					return WifiScriptExecutionResult {
						.exit_code = result.exit_code,
						.output = result.output,
						.timed_out = result.timed_out,
					};
				},
			.current_ap_credentials =
				[]() {
					const auto &credentials =
						detail::cached_wifi_ap_credentials();
					return WifiRuntimeApCredentials {
						.ssid = credentials.ssid,
						.password =
							credentials.password,
						.qr_payload =
							detail::make_wifi_ap_qr_payload(
								credentials),
						.web_url = detail::make_wifi_web_url(
							credentials),
					};
				},
			.prepare_ap_credentials =
				[]() {
					auto &credentials =
						detail::cached_wifi_ap_credentials();
					detail::ensure_wifi_ap_credentials_complete(
						credentials);
					return WifiRuntimeApCredentials {
						.ssid = credentials.ssid,
						.password =
							credentials.password,
						.qr_payload =
							detail::make_wifi_ap_qr_payload(
								credentials),
						.web_url = detail::make_wifi_web_url(
							credentials),
					};
				},
			.write_ap_runtime_files =
				[](std::string_view password) {
					detail::write_wifi_ap_runtime_files(password);
				},
			.probe_ap_client =
				[]() {
					return detail::check_wifi_ap_client();
				},
			.has_saved_network_config =
				[this]() {
					const auto result = detail::run_wifi_script(
						_config.nanokvm, { "saved" });
					if (result.timed_out || result.exit_code != 0) {
						spdlog::warn(
							"[runtime] WiFi saved-network probe failed: exit={} timed_out={}",
							result.exit_code,
							result.timed_out);
						return WifiSavedNetworkAvailability::
							Unknown;
					}
					return parse_wifi_saved_networks_output(
						result.output);
				},
			.enqueue_serial_work =
				[this](std::string_view name,
				       std::function<void()> work) {
					const std::string action_name(name);
					return enqueue_action(
						action_name.c_str(),
						std::move(work), false);
				},
			.request_refresh =
				[this]() {
					if (!_running.load(
						    std::memory_order_acquire)) {
						return;
					}
					(void)enqueue_action(
						"wifi_connect_status_refresh",
						[this]() {
							refresh_status();
						},
						false);
				},
			.publish_lifecycle =
				[this](const WifiRuntimeProjection &projection,
				       WifiLifecyclePatch::Phase phase,
				       uint64_t operation_generation,
				       bool preserve_ap_stop_completion_when_stale) {
					RuntimeStatus status;
					projection.apply_to(status);
					publish_wifi_status(
						status, phase,
						operation_generation,
						preserve_ap_stop_completion_when_stale,
						_running.load(
							std::memory_order_acquire));
				},
		},
		boot_reconnect_pending,
		wifi_saved_network_connect_timeout(_config));

	RuntimeStatus startup;
	const bool video_enabled =
		_video_enabled.load(std::memory_order_acquire);
	const bool hid_enabled = _hid_enabled.load(std::memory_order_acquire);
	const bool ncm_enabled = _ncm_enabled.load(std::memory_order_acquire);
	const bool ncm_supported =
		_ncm_supported.load(std::memory_order_acquire);

	startup.auto_sleep_enabled =
		_auto_sleep_enabled.load(std::memory_order_acquire);
	startup.hdmi_input_supported = true;
	startup.hdmi_input_enabled = video_enabled;
	startup.video_enabled = video_enabled;
	startup.hid_supported = true;
	startup.hid_enabled = hid_enabled;
	startup.ncm_supported = ncm_supported;
	startup.ncm_enabled = ncm_enabled;
	startup.backlight_percent =
		_backlight_percent.load(std::memory_order_acquire);
	startup.screen_on = !_screen_sleeping.load(std::memory_order_acquire);
	if (boot_reconnect_pending) {
		_wifi_runtime_controller->current_projection().apply_to(
			startup);
	}
	startup.updated_at = std::chrono::steady_clock::now();
	_state_store.replace_initial_status(std::move(startup));
}

NanoKvmRuntimeService::~NanoKvmRuntimeService()
{
	stop();
}

bool NanoKvmRuntimeService::start()
{
	if (!_config.nanokvm.enable_runtime_backend) {
		spdlog::info(
			"[runtime] NanoKVM runtime backend disabled by config");
		return false;
	}

	bool restored_config_changed = false;
	if (!detail::set_hdmi_input_enabled_impl(
		    _video_enabled.load(std::memory_order_acquire))) {
		spdlog::warn(
			"NanoKVM runtime failed to restore HDMI input switch file");
		_video_enabled.store(false, std::memory_order_release);
		_config.nanokvm.video_enabled = false;
		restored_config_changed = true;
	}
	if (!detail::set_hid_enabled_impl(
		    _hid_enabled.load(std::memory_order_acquire))) {
		spdlog::warn(
			"NanoKVM runtime failed to restore HID switch file");
		_hid_enabled.store(false, std::memory_order_release);
		_config.nanokvm.hid_enabled = false;
		restored_config_changed = true;
	}
	if (!set_backlight_percent_impl(
		    _config.nanokvm,
		    _backlight_percent.load(std::memory_order_acquire))) {
		spdlog::warn(
			"NanoKVM runtime failed to restore backlight brightness");
		RuntimeStatus next = current_status();
		detail::sync_runtime_flags(
			next, _video_enabled.load(std::memory_order_acquire),
			_hid_enabled.load(std::memory_order_acquire),
			_ncm_enabled.load(std::memory_order_acquire),
			_ncm_supported.load(std::memory_order_acquire),
			_auto_sleep_enabled.load(std::memory_order_acquire),
			_backlight_percent.load(std::memory_order_acquire),
			_screen_sleeping.load(std::memory_order_acquire));
		publish_status(RuntimeFlagsPatch { .status = std::move(next) });
	}
	if (restored_config_changed) {
		schedule_config_persist("restore_switch_state");
	}

	return start_workers(true, true);
}

bool NanoKvmRuntimeService::start_settings_cli_actions()
{
	return start_workers(false, false);
}

bool NanoKvmRuntimeService::start_workers(bool enable_periodic_refresh,
					  bool enable_background_workers)
{
	bool expected = false;
	if (!_running.compare_exchange_strong(expected, true,
					      std::memory_order_acq_rel)) {
		return false;
	}
	_server_transport->start();
	try {
		const auto &app_version = _app_version_cache.load_once();
		RuntimeStatus startup = current_status();
		startup.app_version = app_version;
		if (!_device_region) {
			_device_region = read_device_region(_region_path);
		}
		startup.region = *_device_region;
		_state_store.replace_initial_status(std::move(startup));
	} catch (const std::exception &e) {
		spdlog::error(
			"NanoKVM runtime package version loader failed during startup: {}",
			e.what());
		_running.store(false, std::memory_order_release);
		return false;
	} catch (...) {
		spdlog::error(
			"NanoKVM runtime package version loader failed during startup with unknown exception");
		_running.store(false, std::memory_order_release);
		return false;
	}
	_periodic_refresh_enabled = enable_periodic_refresh;
	_tailscale_page_visible.store(false, std::memory_order_release);
	_tailscale_relinquish_pending.store(false, std::memory_order_release);
	_tailscale_refresh_now.store(false, std::memory_order_release);
	_tailscale_action_generation.store(0, std::memory_order_release);
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		_tailscale_action_pending = false;
	}
	_last_systemd_refresh = {};
	_last_hardware_version_refresh = {};
	_last_wifi_refresh = {};
	_last_storage_refresh = {};
	_last_wifi_signal_refresh = {};
	_more_page_visible.store(false, std::memory_order_release);
	_sysinfo_static_loaded = false;
	_sysinfo_static_cache = SysinfoStaticCache {};
	_systemd_refresh_generation = 0;
	_systemd_refresh_pending = false;
	_display_metrics_refresh_generation = 0;
	_display_metrics_refresh_pending = false;
	_display_metrics_backoff_until = {};
	_display_metrics_failure_count = 0;

	if (enable_background_workers && !_runtime_thread_pool.start()) {
		_running.store(false, std::memory_order_release);
		flush_config_persist();
		return false;
	}
	if (enable_background_workers && !start_config_persist_worker()) {
		_running.store(false, std::memory_order_release);
		_queue_cv.notify_all();
		flush_config_persist();
		_runtime_thread_pool.stop();
		return false;
	}

	try {
		_worker =
			std::thread(&NanoKvmRuntimeService::worker_loop, this);
	} catch (const std::exception &e) {
		spdlog::error("NanoKVM runtime worker start failed: {}",
			      e.what());
		_running.store(false, std::memory_order_release);
		_queue_cv.notify_all();
		stop_config_persist_worker();
		_runtime_thread_pool.stop();
		return false;
	} catch (...) {
		spdlog::error(
			"NanoKVM runtime worker start failed with unknown exception");
		_running.store(false, std::memory_order_release);
		_queue_cv.notify_all();
		stop_config_persist_worker();
		_runtime_thread_pool.stop();
		return false;
	}
	return true;
}

void NanoKvmRuntimeService::stop()
{
	bool was_running = false;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		was_running = _running.exchange(false,
						std::memory_order_acq_rel);
	}
	if (was_running) {
		_queue_cv.notify_all();
	}
	if (_server_transport) {
		_server_transport->stop();
	}
	if (_tailscale_client) {
		_tailscale_client->stop();
	}
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->close();
	}
	if (_worker.joinable()) {
		_worker.join();
	}
	stop_config_persist_worker();
	flush_config_persist();
	_runtime_thread_pool.stop();
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		std::queue<QueuedAction> empty;
		_actions.swap(empty);
		_bool_action_states = {};
	}
	_factory_reset_requested.store(false, std::memory_order_release);
	_region_switch_requested.store(false, std::memory_order_release);
}

bool NanoKvmRuntimeService::is_running() const
{
	return _running.load(std::memory_order_acquire);
}

RuntimeStatus NanoKvmRuntimeService::current_status() const
{
	return _state_store.current_status();
}

void NanoKvmRuntimeService::set_ssh_enabled(bool enabled)
{
	enqueue_action("set_ssh_enabled", [this, enabled] {
		bool ok = true;
		if (enabled) {
			const bool socket_ok = detail::systemd_enable("ssh.socket");
			const bool service_ok = detail::systemd_enable("ssh.service");
			ok = socket_ok && service_ok;
		} else {
			const bool socket_ok = detail::systemd_disable("ssh.socket");
			const bool service_ok = detail::systemd_disable("ssh.service");
			ok = socket_ok && service_ok;
			detail::spawn_command_detached({ "pkill", "sshd" });
		}
		if (!ok) {
			spdlog::warn(
				"NanoKVM runtime failed to set SSH enabled={}",
				enabled);
		}
		++_systemd_refresh_generation;
		_systemd_refresh_pending = false;
		_last_systemd_refresh = {};
	});
}

void NanoKvmRuntimeService::set_ncm_enabled(bool enabled)
{
	enqueue_bool_action(
		BoolActionKey::Ncm, "set_ncm_enabled", enabled,
		[this] {
			if (const auto value = ncm_observation_enabled(
				    observe_ncm_boot_markers(_ncm_marker_paths))) {
				_ncm_enabled.store(*value,
					    std::memory_order_release);
				_ncm_supported.store(true,
					    std::memory_order_release);
			}
			return _ncm_enabled.load(std::memory_order_acquire);
		},
		[this](bool target) {
			if (!detail::set_ncm_enabled_impl(_ncm_marker_paths,
					     _config.nanokvm
						     .usb_gadget_restart_script,
					     target)) {
				spdlog::warn(
					"NanoKVM runtime failed to set NCM enabled={}",
					target);
				return false;
			}
			_ncm_enabled.store(target, std::memory_order_release);
			_ncm_supported.store(true, std::memory_order_release);
			return true;
		});
}

void NanoKvmRuntimeService::refresh_local_service_state()
{
	launch_local_service_action(LocalServiceActionKind::Refresh);
}

void NanoKvmRuntimeService::set_mcp_bundle_enabled(bool enabled)
{
	launch_local_service_action(LocalServiceActionKind::SetMcpBundle,
				    enabled);
}

void NanoKvmRuntimeService::set_screen_timelapse_enabled(bool enabled)
{
	launch_local_service_action(LocalServiceActionKind::SetScreenTimelapse,
				    enabled);
}

void NanoKvmRuntimeService::set_wifi_enabled(bool enabled)
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->set_enabled(enabled);
	}
}

void NanoKvmRuntimeService::set_hdmi_input_enabled(bool enabled)
{
	enqueue_bool_action(
		BoolActionKey::Video, "set_hdmi_input_enabled", enabled,
		[this] {
			return _video_enabled.load(std::memory_order_acquire);
		},
		[this](bool target) {
			if (!detail::set_hdmi_input_enabled_impl(target)) {
				spdlog::warn(
					"NanoKVM runtime failed to set HDMI input enabled={}",
					target);
				return false;
			}
			_video_enabled.store(target, std::memory_order_release);
			_config.nanokvm.video_enabled = target;
			schedule_config_persist("video_enabled");
			return true;
		});
}

void NanoKvmRuntimeService::set_hid_enabled(bool enabled)
{
	enqueue_bool_action(
		BoolActionKey::Hid, "set_hid_enabled", enabled,
		[this] {
			return _hid_enabled.load(std::memory_order_acquire);
		},
		[this](bool target) {
			if (!detail::set_hid_enabled_impl(target)) {
				spdlog::warn(
					"NanoKVM runtime failed to set HID enabled={}",
					target);
				return false;
			}
			_hid_enabled.store(target, std::memory_order_release);
			_config.nanokvm.hid_enabled = target;
			schedule_config_persist("hid_enabled");
			return true;
		});
}

void NanoKvmRuntimeService::set_video_enabled(bool enabled)
{
	set_hdmi_input_enabled(enabled);
}

bool NanoKvmRuntimeService::request_factory_reset()
{
	if (!_config.nanokvm.enable_high_risk_actions) {
		spdlog::warn(
			"NanoKVM runtime factory reset rejected because high-risk actions are disabled");
		return false;
	}

	bool expected = false;
	if (!_factory_reset_requested.compare_exchange_strong(
		    expected, true, std::memory_order_acq_rel)) {
		spdlog::warn(
			"NanoKVM runtime factory reset ignored because a request is already pending");
		return false;
	}

	const bool queued = enqueue_action(
		"factory_reset",
		[this] {
			bool ok = false;
			try {
				ok = _factory_reset_runner(
					kResetToDefaultScriptPath);
			} catch (...) {
				_factory_reset_requested.store(
					false, std::memory_order_release);
				publish_factory_reset_failure();
				throw;
			}

			if (!ok) {
				_factory_reset_requested.store(
					false, std::memory_order_release);
				publish_factory_reset_failure();
				spdlog::warn(
					"NanoKVM runtime factory reset failed: path={}",
					kResetToDefaultScriptPath);
				return;
			}

			spdlog::info(
				"[runtime] NanoKVM runtime factory reset script completed: path={}",
				kResetToDefaultScriptPath);
		},
		false);
	if (!queued) {
		_factory_reset_requested.store(false,
					       std::memory_order_release);
	}
	return queued;
}

bool NanoKvmRuntimeService::request_region_switch(DeviceRegion target)
{
	if (!_running.load(std::memory_order_acquire)) {
		spdlog::warn(
			"NanoKVM runtime region switch rejected because service is not running");
		return false;
	}

	if (!_config.nanokvm.enable_high_risk_actions) {
		spdlog::warn(
			"NanoKVM runtime region switch rejected because high-risk actions are disabled");
		return false;
	}

	bool expected = false;
	if (!_region_switch_requested.compare_exchange_strong(
		    expected, true, std::memory_order_acq_rel)) {
		spdlog::warn(
			"NanoKVM runtime region switch ignored because a request is already pending");
		return false;
	}

	const bool queued = enqueue_action(
		"region_switch",
		[this, target] {
			bool ok = false;
			try {
				ok = _region_switch_runner(target);
			} catch (...) {
				_region_switch_requested.store(
					false, std::memory_order_release);
				publish_region_switch_failure();
				throw;
			}

			if (!ok) {
				_region_switch_requested.store(
					false, std::memory_order_release);
				publish_region_switch_failure();
				spdlog::warn(
					"NanoKVM runtime region switch failed: target={}",
					device_region_name(target));
				return;
			}

			spdlog::info(
				"[runtime] NanoKVM runtime region switch request succeeded: target={}",
				device_region_name(target));
		},
		false);
	if (!queued) {
		_region_switch_requested.store(false,
					       std::memory_order_release);
	}
	return queued;
}

void NanoKvmRuntimeService::about_reset_no_op() const
{
	spdlog::info(
		"[runtime] About Reset requested via Settings CLI (no-op)");
}

void NanoKvmRuntimeService::apply_backlight_percent_now(int percent)
{
	if (!_running.load(std::memory_order_acquire)) {
		spdlog::warn(
			"NanoKVM runtime backlight fast path ignored because service is not running");
		return;
	}

	const int clamped = backlight::clamp_backlight_percent(percent);
	const auto result = backlight::write_backlight_percent(
		_backlight_root, clamped, _config.nanokvm.backlight_invert);
	if (!result.ok) {
		spdlog::warn(
			"NanoKVM runtime failed to write backlight brightness fast path: {}",
			result.error);
		return;
	}

	_screen_sleeping.store(false, std::memory_order_release);
	_backlight_percent.store(clamped, std::memory_order_release);
	schedule_config_persist("backlight_percent");
}

void NanoKvmRuntimeService::set_backlight_percent(int percent)
{
	enqueue_action("set_backlight_percent", [this, percent] {
		const int clamped = backlight::clamp_backlight_percent(percent);
		if (!set_backlight_percent_impl(_config.nanokvm, clamped)) {
			spdlog::warn(
				"NanoKVM runtime failed to set backlight percent={}",
				percent);
			return;
		}
		_screen_sleeping.store(false, std::memory_order_release);
		_backlight_percent.store(clamped, std::memory_order_release);
		_config.nanokvm.backlight_percent = clamped;
		schedule_config_persist("backlight_percent");
	});
}

void NanoKvmRuntimeService::set_screen_sleep(bool sleep)
{
	enqueue_bool_action(
		BoolActionKey::ScreenSleep, "set_screen_sleep", sleep,
		[this] {
			return _screen_sleeping.load(std::memory_order_acquire);
		},
		[this](bool target) {
			const int percent = _backlight_percent.load(
				std::memory_order_acquire);
			if (!set_screen_sleep_impl(_config.nanokvm, target,
						   percent)) {
				spdlog::warn(
					"NanoKVM runtime failed to set screen sleep={}",
					target);
				return false;
			}
			_screen_sleeping.store(target,
					       std::memory_order_release);
			return true;
		});
}

void NanoKvmRuntimeService::set_auto_sleep_enabled(bool enabled)
{
	enqueue_bool_action(
		BoolActionKey::AutoSleep, "set_auto_sleep_enabled", enabled,
		[this] {
			return _auto_sleep_enabled.load(
				std::memory_order_acquire);
		},
		[this](bool target) {
			_auto_sleep_enabled.store(target,
						  std::memory_order_release);
			_config.nanokvm.auto_sleep_enabled = target;
			if (!target &&
			    !set_screen_sleep_impl(
				    _config.nanokvm, false,
				    _backlight_percent.load(
					    std::memory_order_acquire))) {
				spdlog::warn(
					"NanoKVM runtime failed to wake screen after disabling auto-sleep");
			} else if (!target) {
				_screen_sleeping.store(
					false, std::memory_order_release);
			}
			schedule_config_persist("auto_sleep_enabled");
			return true;
		});
}

void NanoKvmRuntimeService::set_ui_auto_rotation_enabled(bool enabled)
{
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		_config.ui.enable_auto_rotation = enabled;
	}
	schedule_config_persist("ui_enable_auto_rotation");
}

void NanoKvmRuntimeService::set_ui_manual_display_rotation(bool enabled)
{
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		_config.ui.manual_display_rotation = enabled;
	}
	schedule_config_persist("ui_manual_display_rotation");
}

void NanoKvmRuntimeService::set_wifi_ap_enabled(bool enabled)
{
	set_wifi_enabled(enabled);
}

bool NanoKvmRuntimeService::settings_cli_set_video_enabled(bool enabled,
							   std::string &error)
{
	return run_settings_cli_sync_action(
		"set_hdmi_input_enabled",
		[this, enabled](std::string &action_error) {
			if (!detail::set_hdmi_input_enabled_impl(enabled)) {
				action_error =
					"failed to update HDMI enable flag";
				return false;
			}
			_video_enabled.store(enabled,
					     std::memory_order_release);
			_config.nanokvm.video_enabled = enabled;
			publish_local_runtime_status_snapshot();
			return persist_config_now("video_enabled",
						  action_error);
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_hid_enabled(bool enabled,
							 std::string &error)
{
	return run_settings_cli_sync_action(
		"set_hid_enabled",
		[this, enabled](std::string &action_error) {
			if (!detail::set_hid_enabled_impl(enabled)) {
				action_error =
					"failed to update HID enable flag";
				return false;
			}
			_hid_enabled.store(enabled, std::memory_order_release);
			_config.nanokvm.hid_enabled = enabled;
			publish_local_runtime_status_snapshot();
			return persist_config_now("hid_enabled", action_error);
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_ncm_enabled(bool enabled,
							 std::string &error)
{
	return run_settings_cli_sync_action(
		"set_ncm_enabled",
		[this, enabled](std::string &action_error) {
			if (!detail::set_ncm_enabled_impl(_ncm_marker_paths,
					     _config.nanokvm
						     .usb_gadget_restart_script,
					     enabled)) {
				action_error =
					"failed to update NCM flags or restart USB gadget";
				return false;
			}
			_ncm_enabled.store(enabled, std::memory_order_release);
			_ncm_supported.store(true, std::memory_order_release);
			publish_local_runtime_status_snapshot();
			return true;
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_ssh_enabled(bool enabled,
							 std::string &error)
{
	return run_settings_cli_sync_action(
		"set_ssh_enabled",
		[this, enabled](std::string &action_error) {
			bool ok = true;
			if (enabled) {
				ok = detail::systemd_enable("ssh.socket") &&
				     detail::systemd_enable("ssh.service");
			} else {
				ok = detail::systemd_disable("ssh.socket") &&
				     detail::systemd_disable("ssh.service");
				detail::spawn_command_detached({ "pkill", "sshd" });
			}
			if (!ok) {
				action_error =
					"failed to update SSH systemd units";
				return false;
			}
			RuntimeStatus next = current_status();
			next.ssh_supported = true;
			next.ssh_enabled = enabled;
			next.updated_at = std::chrono::steady_clock::now();
			publish_status(SystemdStatusPatch {
				.status = std::move(next) });
			return true;
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_wifi_ap_enabled(bool enabled,
							     std::string &error)
{
	return run_settings_cli_sync_action(
		"set_wifi_enabled",
		[this, enabled](std::string &action_error) {
			if (!_wifi_runtime_controller) {
				action_error = "WiFi runtime is unavailable";
				return false;
			}
			return _wifi_runtime_controller->set_enabled_sync(
				enabled, action_error);
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_reset_wifi(std::string &error)
{
	return run_settings_cli_sync_action(
		"reset_wifi",
		[this](std::string &action_error) {
			if (!_wifi_runtime_controller) {
				action_error = "WiFi runtime is unavailable";
				return false;
			}
			return _wifi_runtime_controller->reset_sync(
				action_error);
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_backlight_percent(
	int percent, std::vector<std::pair<std::string, std::string>> &fields,
	std::string &error)
{
	return run_settings_cli_sync_action(
		"set_backlight_percent",
		[this, percent, &fields](std::string &action_error) {
			const int clamped =
				backlight::clamp_backlight_percent(percent);
			const auto write_result =
				backlight::write_backlight_percent(
					_config.nanokvm.backlight_root, clamped,
					_config.nanokvm.backlight_invert);
			if (!write_result.ok) {
				action_error = write_result.error;
				return false;
			}
			_screen_sleeping.store(false,
					       std::memory_order_release);
			_backlight_percent.store(clamped,
						 std::memory_order_release);
			_config.nanokvm.backlight_percent = clamped;
			fields.emplace_back("backlight_percent",
					    std::to_string(clamped));
			fields.emplace_back("backlight_raw",
					    std::to_string(write_result.raw));
			publish_local_runtime_status_snapshot();
			return persist_config_now("backlight_percent",
						  action_error);
		},
		error);
}

bool NanoKvmRuntimeService::settings_cli_set_auto_sleep_enabled(
	bool enabled, std::vector<std::pair<std::string, std::string>> &fields,
	std::string &error)
{
	return run_settings_cli_sync_action(
		"set_auto_sleep_enabled",
		[this, enabled, &fields](std::string &action_error) {
			if (!enabled) {
				const auto write_result =
					backlight::write_backlight_percent(
						_config.nanokvm.backlight_root,
						_backlight_percent.load(
							std::memory_order_acquire),
						_config.nanokvm
							.backlight_invert);
				if (!write_result.ok) {
					action_error = write_result.error;
					return false;
				}
				_screen_sleeping.store(
					false, std::memory_order_release);
				fields.emplace_back(
					"backlight_raw",
					std::to_string(write_result.raw));
			}

			_auto_sleep_enabled.store(enabled,
						  std::memory_order_release);
			_config.nanokvm.auto_sleep_enabled = enabled;
			publish_local_runtime_status_snapshot();
			return persist_config_now("auto_sleep_enabled",
						  action_error);
		},
		error);
}

void NanoKvmRuntimeService::reset_wifi()
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->reset();
	}
}

void NanoKvmRuntimeService::scan_wifi_networks()
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->scan_networks();
	}
}

void NanoKvmRuntimeService::connect_wifi_network(std::string ssid,
						 std::string password)
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->connect_network(std::move(ssid),
							  std::move(password));
	}
}

void NanoKvmRuntimeService::start_wifi_qr_ap_config()
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->start_qr_ap_config();
	}
}

uint64_t NanoKvmRuntimeService::stop_wifi_qr_ap_config()
{
	return _wifi_runtime_controller ?
		       _wifi_runtime_controller->stop_qr_ap_config() :
		       0;
}

void NanoKvmRuntimeService::stop_wifi_qr_ap_config_after_success()
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->stop_qr_ap_config_after_success();
	}
}

void NanoKvmRuntimeService::notify_wifi_page_visible(bool visible)
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->notify_page_visible(visible);
	}
}

void NanoKvmRuntimeService::notify_more_page_visible(bool visible)
{
	const bool was_visible =
		_more_page_visible.exchange(visible, std::memory_order_acq_rel);
	if (visible && !was_visible) {
		enqueue_action("more_page_visible", [this] {
		}, true);
	}
}

void NanoKvmRuntimeService::notify_tailscale_page_visible(bool visible)
{
	const bool was_visible =
		_tailscale_page_visible.exchange(visible, std::memory_order_acq_rel);
	if (visible == was_visible) {
		return;
	}
	if (visible) {
		_tailscale_relinquish_pending.store(false,
						   std::memory_order_release);
		_tailscale_refresh_now.store(true, std::memory_order_release);
	} else {
		const RuntimeStatus status = current_status();
		const TailscaleStage stage = tailscale_stage(
			status.tailscale_gate_seq > 0, status.tailscale_started,
			status.tailscale_matched);
		if (tailscale_relinquish_on_leave(stage)) {
			_tailscale_relinquish_pending.store(
				true, std::memory_order_release);
		}
	}

	_queue_cv.notify_all();
}

void NanoKvmRuntimeService::notify_wifi_qr_dismissed()
{
	if (_wifi_runtime_controller) {
		_wifi_runtime_controller->notify_qr_dismissed();
	}
}

std::string NanoKvmRuntimeService::wifi_ap_qr_payload() const
{
	return _wifi_runtime_controller ?
		       _wifi_runtime_controller->qr_payload() :
		       std::string {};
}

void NanoKvmRuntimeService::update_display_labels(
	http_api::DisplayLabelsUpdate update)
{
	publish_status(DisplayIngressPatch {
		.resolution = std::move(update.resolution),
		.fps = std::move(update.fps),
		.fps_unit = std::move(update.fps_unit),
		.keyboard_special = std::move(update.keyboard_special),
		.keyboard_normal = std::move(update.keyboard_normal),
	});
}

void NanoKvmRuntimeService::update_hid_state(http_api::HidStateUpdate update)
{
	publish_status(HidIngressPatch {
		.mouse_active = std::move(update.mouse_active),
		.keyboard_special = std::move(update.keyboard_special),
		.keyboard_normal = std::move(update.keyboard_normal),
	});
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
BackgroundActionResult
NanoKvmRuntimeService::apply_background_selection(std::string image_name)
{
	std::lock_guard<std::mutex> lock(_background_action_mutex);
	ui::BackgroundAssets assets;
	std::string error;
	if (!ui::load_background_assets_by_name(
		    image_name, ui::kDefaultBackgroundTargetWidth,
		    ui::kDefaultBackgroundTargetHeight, assets, error)) {
		return { .ok = false, .http_status = 400, .detail = error };
	}
	if (!ui::save_background_selection(image_name, error)) {
		return { .ok = false, .http_status = 500, .detail = error };
	}

	if (_callbacks.on_background_assets) {
		try {
			_callbacks.on_background_assets(std::move(assets));
		} catch (const std::exception &e) {
			spdlog::warn(
				"NanoKVM runtime background callback failed: {}",
				e.what());
			return { .ok = false,
				 .http_status = 500,
				 .detail = "background callback failed" };
		} catch (...) {
			spdlog::warn(
				"NanoKVM runtime background callback failed with unknown exception");
			return { .ok = false,
				 .http_status = 500,
				 .detail = "background callback failed" };
		}
	}
	return { .ok = true, .http_status = 200 };
}
#endif

}
