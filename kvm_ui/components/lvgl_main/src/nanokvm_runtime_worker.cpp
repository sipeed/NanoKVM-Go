#include "lvgl_main/nanokvm_runtime.hpp"
#include "nanokvm_runtime_internal.hpp"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <future>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <utility>

#include <spdlog/spdlog.h>

#include "lvgl_main/nanokvm_ip_selection.hpp"
#include "lvgl_main/wifi_runtime_controller.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"
#include "nanokvm_server_transport.hpp"
#include "nanokvm_tailscale_client.hpp"
#include "ui/local_service_state_client.hpp"

namespace lvgl_main
{

namespace
{

constexpr std::chrono::seconds kPollInterval { 1 };
constexpr std::chrono::milliseconds kDisplayMetricsInterval { 333 };
constexpr std::chrono::seconds kDisplayMetricsBackoffMin { 1 };
constexpr std::chrono::seconds kDisplayMetricsBackoffMax { 3 };
constexpr std::chrono::seconds kSystemdRefreshInterval { 5 };
constexpr std::chrono::seconds kHardwareVersionRefreshInterval { 60 };
constexpr std::chrono::seconds kStorageRefreshInterval { 5 };
constexpr std::chrono::seconds kWifiSignalRefreshInterval { 5 };
constexpr std::chrono::milliseconds kActionRefreshDebounce { 150 };

struct RuntimeTaskMetadata {
	std::string_view task_class;
	std::string_view key;
	std::string_view state_owner;
};

constexpr RuntimeTaskMetadata kUnknownActionMetadata { "unclassified-action",
						       "anonymous", "unknown" };
constexpr RuntimeTaskMetadata kFullRefreshMetadata { "slow-read-collect",
						     "full_refresh",
						     "RuntimeStatus" };
constexpr RuntimeTaskMetadata kDisplayMetricsMetadata {
	"high-frequency-collect", "display_metrics",
	"RuntimeStatus::display_metrics"
};
constexpr RuntimeTaskMetadata kConfigPersistMetadata { "config-persist-action",
						       "app_config",
						       "AppConfig" };

RuntimeTaskMetadata classify_runtime_action(std::string_view name)
{

	if (name == "set_ssh_enabled") {
		return { "exclusive-write-action", "ssh",
			 "RuntimeStatus::ssh_enabled" };
	}
	if (name == "factory_reset") {
		return { "exclusive-write-action", "factory_reset", "device" };
	}
	if (name == "region_switch") {
		return { "exclusive-write-action", "region_switch", "device" };
	}
	if (name == "set_ncm_enabled") {
		return { "coalescible-write-action", "ncm",
			 "RuntimeStatus::ncm_enabled" };
	}
	if (name == "set_hdmi_input_enabled") {
		return { "coalescible-write-action", "video",
			 "RuntimeStatus::video_enabled" };
	}
	if (name == "set_hid_enabled") {
		return { "coalescible-write-action", "hid",
			 "RuntimeStatus::hid_enabled" };
	}
	if (name == "refresh_local_service_state") {
		return { "pool-backed-read", "local_services",
			 "RuntimeStatus::local_services" };
	}
	if (name == "set_mcp_bundle_enabled") {
		return { "pool-backed-write", "mcp_bundle",
			 "RuntimeStatus::local_services" };
	}
	if (name == "set_screen_timelapse_enabled") {
		return { "pool-backed-write", "screen_timelapse",
			 "RuntimeStatus::local_services" };
	}
	if (name == "set_screen_sleep") {
		return { "coalescible-write-action", "screen_sleep",
			 "RuntimeStatus::screen_on" };
	}
	if (name == "set_backlight_percent") {
		return { "coalescible-write-action", "backlight",
			 "RuntimeStatus::backlight_percent" };
	}
	if (name == "set_auto_sleep_enabled") {
		return { "coalescible-write-action", "auto_sleep",
			 "RuntimeStatus::auto_sleep_enabled" };
	}
	if (name == "submit_config_persist_snapshot") {
		return kConfigPersistMetadata;
	}
	if (name == "merge_systemd_status") {
		return { "pool-result-merge", "systemd",
			 "RuntimeStatus::systemd" };
	}
	if (name == "merge_display_metrics") {
		return { "pool-result-merge", "display_metrics",
			 "RuntimeStatus::display_metrics" };
	}
	if (name == "merge_local_service_result") {
		return { "pool-result-merge", "local_services",
			 "RuntimeStatus::local_services" };
	}
	return kUnknownActionMetadata;
}

int64_t elapsed_ms(std::chrono::steady_clock::time_point start,
		   std::chrono::steady_clock::time_point end)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(end -
								     start)
		.count();
}

std::chrono::milliseconds
display_metrics_interval_for_failures(uint8_t failure_count) noexcept
{
	if (failure_count == 0) {
		return kDisplayMetricsInterval;
	}

	auto backoff = kDisplayMetricsBackoffMin;
	for (uint8_t index = 1;
	     index < failure_count && backoff < kDisplayMetricsBackoffMax;
	     ++index) {
		backoff = std::min(kDisplayMetricsBackoffMax, backoff * 2);
	}
	return std::chrono::duration_cast<std::chrono::milliseconds>(backoff);
}

struct LocalServiceResultFields {
	bool ok { false };
	int status_code { 0 };
	bool mcp_supported { false };
	bool mcp_active { false };
	bool replay_supported { false };
	bool replay_active { false };
	std::string error;
};

LocalServiceResultFields to_local_service_result_fields(
	const ui::local_service_state::ClientResult &result)
{
	LocalServiceResultFields fields;
	fields.ok = result.ok;
	fields.status_code = result.status_code;
	fields.error = result.error;
	if (result) {
		fields.mcp_supported = result.services.mcp_supported;
		fields.mcp_active = ui::local_service_state::mcp_bundle_active(
			result.services);
		fields.replay_supported =
			result.services.screen_timelapse_supported;
		fields.replay_active =
			ui::local_service_state::screen_timelapse_active(
				result.services);
	}
	return fields;
}

}

void NanoKvmRuntimeService::worker_loop()
{
	using Clock = std::chrono::steady_clock;

	const auto no_scheduled_time = Clock::time_point::max();
	auto next_poll_refresh = Clock::time_point::min();
	std::optional<Clock::time_point> next_action_refresh;
	auto next_display_metric_refresh = Clock::time_point::min();
	auto next_tailscale_refresh = Clock::time_point::min();
	auto next_tailscale_code_refresh = Clock::time_point::min();
	auto last_tailscale_stage = TailscaleStage::Unknown;
	bool tailscale_lane_active = false;
	bool tailscale_code_lane_active = false;
	uint8_t tailscale_failure_count { 0 };
	uint8_t tailscale_code_failure_count { 0 };
	while (_running.load(std::memory_order_acquire)) {

		const RuntimeStatus tailscale_status = current_status();
		const TailscaleStage current_stage = tailscale_stage(
			tailscale_status.tailscale_gate_seq > 0,
			tailscale_status.tailscale_started,
			tailscale_status.tailscale_matched);
		const TailscaleTickPlan tailscale_plan = tailscale_tick_plan(
			_periodic_refresh_enabled,
			_tailscale_page_visible.load(std::memory_order_acquire),
			current_stage);
		const bool tailscale_refresh_now =
			_tailscale_refresh_now.exchange(false,
							std::memory_order_acq_rel);
		const auto tailscale_tick_now = Clock::now();
		if (tailscale_plan.heartbeat != tailscale_lane_active ||
		    current_stage != last_tailscale_stage ||
		    tailscale_refresh_now) {
			next_tailscale_refresh = tailscale_tick_now;
		}
		const TailscaleCodeLaneStep tailscale_code_lane =
			tailscale_code_lane_step(
				tailscale_code_lane_active, tailscale_plan.code,
				tailscale_tick_now >=
					next_tailscale_code_refresh);
		if (tailscale_code_lane.reopened) {
			next_tailscale_code_refresh = tailscale_tick_now;
			tailscale_code_failure_count = 0;
		}
		tailscale_lane_active = tailscale_plan.heartbeat;
		last_tailscale_stage = current_stage;
		tailscale_code_lane_active = tailscale_plan.code;
		QueuedAction action;
		{
			std::unique_lock<std::mutex> lock(_queue_mutex);
			if (!_periodic_refresh_enabled) {
				while (_actions.empty() &&
				       _running.load(
					       std::memory_order_acquire)) {
					const auto now = Clock::now();
					if (_config_persist_scheduled &&
					    now >= _config_persist_deadline) {
						break;
					}
					if (_config_persist_scheduled) {
						_queue_cv.wait_until(
							lock,
							_config_persist_deadline);
					} else {
						_queue_cv.wait(lock);
					}
				}
			} else {
				const auto now = Clock::now();
				const auto next_poll_wakeup =
					next_action_refresh.value_or(
						next_poll_refresh);
				const auto next_display_metric_wakeup =
					next_action_refresh.has_value() ?
						no_scheduled_time :
						next_display_metric_refresh;
				const auto next_config_persist =
					_config_persist_scheduled ?
						_config_persist_deadline :
						no_scheduled_time;
				const auto next_tailscale_wakeup =
					tailscale_plan.heartbeat ?
						next_tailscale_refresh :
						no_scheduled_time;
				const auto next_wakeup = std::min({
					next_poll_wakeup,
					next_display_metric_wakeup,
					next_config_persist,
					next_tailscale_wakeup,
				});
				if (_actions.empty() && now < next_wakeup) {
					_queue_cv.wait_until(lock, next_wakeup);
				}
			}
			if (!_running.load(std::memory_order_acquire)) {
				break;
			}
			if (!_actions.empty()) {
				action = std::move(_actions.front());
				_actions.pop();
			}
		}

		if (action.run) {
			try {
				action.run();
			} catch (const std::exception &e) {
				spdlog::warn(
					"NanoKVM runtime action failed: {}",
					e.what());
			} catch (...) {
				spdlog::warn(
					"NanoKVM runtime action failed with unknown exception");
			}
			if (action.request_refresh) {
				next_action_refresh =
					Clock::now() + kActionRefreshDebounce;
			} else if (action.reschedule_display_metrics) {
				next_display_metric_refresh =
					Clock::now() +
					display_metrics_interval_for_failures(
						_display_metrics_failure_count);
			}
			continue;
		}

		if (_periodic_refresh_enabled) {
			const auto now = Clock::now();
			const bool action_refresh_due =
				next_action_refresh.has_value() &&
				now >= *next_action_refresh;
			const bool poll_refresh_due =
				!next_action_refresh.has_value() &&
				now >= next_poll_refresh;
			if (action_refresh_due || poll_refresh_due) {
				const char *refresh_reason =
					action_refresh_due ? "action_debounce" :
							     "poll";
				const auto refresh_started_at [[maybe_unused]] = Clock::now();
				spdlog::info(
					"[runtime] NanoKVM runtime refresh start reason={} class={} key={} state_owner={}",
					refresh_reason,
					kFullRefreshMetadata.task_class,
					kFullRefreshMetadata.key,
					kFullRefreshMetadata.state_owner);
				refresh_status();
				const auto refresh_ended_at = Clock::now();
				spdlog::info(
					"[runtime] NanoKVM runtime refresh end reason={} duration_ms={} class={} key={} state_owner={}",
					refresh_reason,
					elapsed_ms(refresh_started_at,
						   refresh_ended_at),
					kFullRefreshMetadata.task_class,
					kFullRefreshMetadata.key,
					kFullRefreshMetadata.state_owner);
				next_poll_refresh =
					refresh_ended_at + kPollInterval;
				next_action_refresh.reset();
				next_display_metric_refresh =
					refresh_ended_at +
					display_metrics_interval_for_failures(
						_display_metrics_failure_count);
				continue;
			}

			if (!next_action_refresh.has_value() &&
			    Clock::now() >= next_display_metric_refresh) {
				const auto refresh_started_at [[maybe_unused]] = Clock::now();
				SPDLOG_DEBUG(
					"[runtime] NanoKVM runtime display metrics refresh start reason=display_metrics class={} key={} state_owner={}",
					kDisplayMetricsMetadata.task_class,
					kDisplayMetricsMetadata.key,
					kDisplayMetricsMetadata.state_owner);
				const auto next_interval =
					refresh_display_metrics();
				const auto refresh_ended_at = Clock::now();
				SPDLOG_DEBUG(
					"[runtime] NanoKVM runtime display metrics refresh end reason=display_metrics duration_ms={} class={} key={} state_owner={}",
					elapsed_ms(refresh_started_at,
						   refresh_ended_at),
					kDisplayMetricsMetadata.task_class,
					kDisplayMetricsMetadata.key,
					kDisplayMetricsMetadata.state_owner);
				next_display_metric_refresh =
					refresh_ended_at + next_interval;
				continue;
			}

			if (tailscale_plan.heartbeat &&
			    Clock::now() >= next_tailscale_refresh) {
				const bool try_code = tailscale_code_lane.attempt;
				next_tailscale_refresh =
					refresh_tailscale_tick(
						try_code,
						tailscale_failure_count,
						tailscale_code_failure_count,
						next_tailscale_code_refresh);
				continue;
			}
		}

		bool persist_config = false;
		uint64_t config_generation = 0;
		std::string config_reason;
		{
			std::lock_guard<std::mutex> lock(_queue_mutex);
			persist_config = _config_persist_scheduled &&
					 Clock::now() >=
						 _config_persist_deadline;
			if (persist_config) {
				config_generation =
					_config_persist_deadline_generation;
				config_reason = _config_persist_deadline_reason;
			}
		}
		if (persist_config) {
			submit_config_persist_snapshot(
				config_generation, std::move(config_reason));
		}
	}
}

bool NanoKvmRuntimeService::enqueue_action(Action action)
{
	return enqueue_action("anonymous", std::move(action));
}

bool NanoKvmRuntimeService::enqueue_action(const char *name, Action action,
					   bool request_refresh,
					   bool reschedule_display_metrics)
{
	if (!action) {
		return false;
	}
	if (!_running.load(std::memory_order_acquire)) {
		spdlog::warn(
			"NanoKVM runtime action ignored because service is not running");
		return false;
	}
	static std::atomic<uint64_t> next_action_id { 0 };
	const uint64_t action_id =
		next_action_id.fetch_add(1, std::memory_order_relaxed) + 1U;
	const std::string action_name = name == nullptr ? "anonymous" : name;
	const RuntimeTaskMetadata metadata =
		classify_runtime_action(action_name);
	const auto enqueued_at = std::chrono::steady_clock::now();
	Action timed_action = [action_id, action_name, metadata, enqueued_at,
			       action = std::move(action)]() mutable {
		const auto started_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime] NanoKVM runtime action start id={} name={} wait_ms={} class={} key={} state_owner={}",
			action_id, action_name,
			elapsed_ms(enqueued_at, started_at),
			metadata.task_class, metadata.key,
			metadata.state_owner);
		try {
			action();
		} catch (...) {
			const auto failed_at = std::chrono::steady_clock::now();
			spdlog::warn(
				"NanoKVM runtime action end id={} name={} ok=false duration_ms={} class={} key={} state_owner={}",
				action_id, action_name,
				elapsed_ms(started_at, failed_at),
				metadata.task_class, metadata.key,
				metadata.state_owner);
			throw;
		}
		const auto ended_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime] NanoKVM runtime action end id={} name={} ok=true duration_ms={} class={} key={} state_owner={}",
			action_id, action_name,
			elapsed_ms(started_at, ended_at), metadata.task_class,
			metadata.key, metadata.state_owner);
	};
	std::size_t queue_depth = 0;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		_actions.push(
			QueuedAction { .run = std::move(timed_action),
				       .request_refresh = request_refresh,
				       .reschedule_display_metrics =
					       reschedule_display_metrics });
		queue_depth = _actions.size();
	}
	spdlog::info(
		"[runtime] NanoKVM runtime action enqueue id={} name={} queue_depth={} class={} key={} state_owner={}",
		action_id, action_name, queue_depth, metadata.task_class,
		metadata.key, metadata.state_owner);
	_queue_cv.notify_all();
	return true;
}

void NanoKvmRuntimeService::enqueue_bool_action(BoolActionKey key,
						const char *name, bool target,
						BoolActionRead current,
						BoolActionApply apply,
						bool request_refresh)
{
	if (!current || !apply) {
		return;
	}
	if (!_running.load(std::memory_order_acquire)) {
		spdlog::warn(
			"NanoKVM runtime action ignored because service is not running");
		return;
	}

	const std::string action_name = name == nullptr ? "anonymous" : name;
	const RuntimeTaskMetadata metadata =
		classify_runtime_action(action_name);
	const auto index = static_cast<std::size_t>(key);
	if (index >= _bool_action_states.size()) {
		spdlog::warn(
			"NanoKVM runtime bool action ignored with invalid key name={}",
			action_name);
		return;
	}

	uint64_t generation = 0;
	bool publish_current = false;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		auto &state = _bool_action_states[index];
		if (state.queued) {
			if (state.target == target) {
				spdlog::info(
					"[runtime] NanoKVM runtime bool action skip name={} target={} reason=pending_target class={} key={} state_owner={}",
					action_name, target,
					metadata.task_class, metadata.key,
					metadata.state_owner);
				return;
			}
			const bool previous_target = state.target;
			state.target = target;
			generation = ++state.generation;
			spdlog::info(
				"[runtime] NanoKVM runtime bool action coalesce name={} previous_target={} target={} generation={} class={} key={} state_owner={}",
				action_name, previous_target, target,
				generation, metadata.task_class, metadata.key,
				metadata.state_owner);
			return;
		}
		if (state.running && state.target == target) {
			spdlog::info(
				"[runtime] NanoKVM runtime bool action skip name={} target={} reason=running_target class={} key={} state_owner={}",
				action_name, target, metadata.task_class,
				metadata.key, metadata.state_owner);
			return;
		}
		if (!state.running && current() == target) {
			publish_current = true;
			spdlog::info(
				"[runtime] NanoKVM runtime bool action skip name={} target={} reason=current_state class={} key={} state_owner={}",
				action_name, target, metadata.task_class,
				metadata.key, metadata.state_owner);
		} else {
			state.queued = true;
			state.target = target;
			generation = ++state.generation;
		}
	}
	if (publish_current) {
		publish_local_runtime_status_snapshot();
		return;
	}

	const bool queued = enqueue_action(
		action_name.c_str(),
		[this, key, action_name, current = std::move(current),
		 apply = std::move(apply)]() mutable {
			run_bool_action(key, action_name.c_str(),
					std::move(current), std::move(apply));
		},
		request_refresh);
	if (!queued) {
		std::lock_guard<std::mutex> lock(_queue_mutex);
		auto &state = _bool_action_states[index];
		if (state.generation == generation && state.queued &&
		    !state.running) {
			state = {};
		}
	}
}

void NanoKvmRuntimeService::run_bool_action(BoolActionKey key, const char *name,
					    BoolActionRead current,
					    BoolActionApply apply)
{
	bool target = false;
	uint64_t generation = 0;
	if (!take_bool_action_target(key, target, generation)) {
		return;
	}

	const std::string action_name = name == nullptr ? "anonymous" : name;
	const RuntimeTaskMetadata metadata =
		classify_runtime_action(action_name);
	if (current && current() == target) {
		finish_bool_action(key, generation);
		spdlog::info(
			"[runtime] NanoKVM runtime bool action skip name={} target={} generation={} reason=current_state class={} key={} state_owner={}",
			action_name, target, generation, metadata.task_class,
			metadata.key, metadata.state_owner);
		publish_local_runtime_status_snapshot();
		return;
	}

	bool ok = false;
	try {
		ok = apply && apply(target);
	} catch (...) {
		finish_bool_action(key, generation);
		publish_local_runtime_status_snapshot();
		throw;
	}
	finish_bool_action(key, generation);
	if (!ok) {
		spdlog::warn(
			"NanoKVM runtime bool action failed name={} target={} generation={} class={} key={} state_owner={}",
			action_name, target, generation, metadata.task_class,
			metadata.key, metadata.state_owner);
	}
	publish_local_runtime_status_snapshot();
}

bool NanoKvmRuntimeService::take_bool_action_target(BoolActionKey key,
						    bool &target,
						    uint64_t &generation)
{
	const auto index = static_cast<std::size_t>(key);
	if (index >= _bool_action_states.size()) {
		return false;
	}
	std::lock_guard<std::mutex> lock(_queue_mutex);
	auto &state = _bool_action_states[index];
	if (!state.queued) {
		return false;
	}
	target = state.target;
	generation = state.generation;
	state.queued = false;
	state.running = true;
	return true;
}

void NanoKvmRuntimeService::finish_bool_action(BoolActionKey key,
					       uint64_t generation)
{
	const auto index = static_cast<std::size_t>(key);
	if (index >= _bool_action_states.size()) {
		return;
	}
	std::lock_guard<std::mutex> lock(_queue_mutex);
	auto &state = _bool_action_states[index];
	if (state.generation >= generation) {
		state.running = false;
	}
}

bool NanoKvmRuntimeService::run_settings_cli_sync_action(const char *name,
							 SyncAction action,
							 std::string &error)
{
	if (!action) {
		error = "missing runtime action";
		return false;
	}
	if (!_running.load(std::memory_order_acquire)) {
		error = "runtime service is not running";
		return false;
	}

	auto promise =
		std::make_shared<std::promise<std::pair<bool, std::string>>>();
	auto future = promise->get_future();
	const bool queued = enqueue_action(
		name,
		[action = std::move(action), promise]() mutable {
			std::string action_error;
			bool ok = false;
			try {
				ok = action(action_error);
			} catch (const std::exception &e) {
				action_error = e.what();
			} catch (...) {
				action_error = "unknown runtime action failure";
			}
			if (!ok && action_error.empty()) {
				action_error = "runtime action failed";
			}
			promise->set_value(
				{ ok, ok ? std::string {} : action_error });
			if (!ok) {
				spdlog::warn(
					"NanoKVM runtime Settings CLI action failed: {}",
					action_error);
			}
		},
		false);
	if (!queued) {
		error = "failed to enqueue runtime action";
		return false;
	}

	auto [ok, action_error] = future.get();
	if (!ok) {
		error = std::move(action_error);
		if (error.empty()) {
			error = "runtime action failed";
		}
	}
	return ok;
}

void NanoKvmRuntimeService::collect_sysinfo_static()
{
	if (_sysinfo_static_loaded) {
		return;
	}
	if (auto value = detail::read_device_key()) {
		_sysinfo_static_cache.device_key = *value;
	}
	if (auto value = detail::read_hostname()) {
		_sysinfo_static_cache.hostname = *value;
		_sysinfo_static_cache.mdns_name = *value + ".local";
	}
	if (auto value = detail::read_device_number()) {
		_sysinfo_static_cache.device_number = *value;
	}
	if (auto value = detail::read_image_version()) {
		_sysinfo_static_cache.image_version = *value;
	}
	if (auto value = detail::read_edid_name()) {
		_sysinfo_static_cache.edid_name = *value;
	}
	if (auto value = detail::read_mac_address()) {
		_sysinfo_static_cache.mac_address = *value;
	}
	_sysinfo_static_loaded = true;
}

void NanoKvmRuntimeService::refresh_status()
{
	const auto now = std::chrono::steady_clock::now();
	RuntimeStatus current = current_status();
	RuntimeStatus next;
	auto &server_transport = *_server_transport;
	RefreshObservationDomain observed_domains =
		RefreshObservationDomain::None;
	const bool preserve_last_known_good =
		_config.nanokvm.preserve_last_known_good_state;
	const uint64_t wifi_operation_generation =
		_wifi_runtime_controller ?
			_wifi_runtime_controller->operation_generation() :
			0;
	next.app_version = _app_version_cache.value();
	observed_domains |= RefreshObservationDomain::AppVersion;
	const bool video_enabled =
		_video_enabled.load(std::memory_order_acquire);
	const bool hid_enabled = _hid_enabled.load(std::memory_order_acquire);
	const bool auto_sleep_enabled =
		_auto_sleep_enabled.load(std::memory_order_acquire);
	const int backlight_percent =
		_backlight_percent.load(std::memory_order_acquire);
	const bool screen_sleeping =
		_screen_sleeping.load(std::memory_order_acquire);
	const bool hardware_version_due =
		detail::domain_refresh_due(_last_hardware_version_refresh, now,
				   kHardwareVersionRefreshInterval);
	const bool systemd_due = detail::domain_refresh_due(
		_last_systemd_refresh, now,
						    kSystemdRefreshInterval);
	const bool streamer_due = video_enabled &&
				  _display_metrics_backoff_until <= now &&
				  !_display_metrics_refresh_pending;
	int64_t hardware_version_duration_ms = 0;
	try {
		const bool network_observed = detail::collect_ip_addresses(next);
		if (network_observed) {
			observed_domains |= RefreshObservationDomain::NetworkAddresses;
		}
		if (video_enabled) {
			if (detail::collect_video_status(server_transport, next)) {
				observed_domains |= RefreshObservationDomain::Video;
			}
		}
		if (detail::collect_hid_status(server_transport, next)) {
			observed_domains |= RefreshObservationDomain::Hid;
		}
		const bool wifi_observed =
			detail::collect_wifi_status(_config.nanokvm, server_transport, next,
					    network_observed,
					    !preserve_last_known_good);
		if (wifi_observed) {
			observed_domains |= RefreshObservationDomain::Wifi;
		}
		if (_wifi_runtime_controller &&
		    (wifi_observed || !preserve_last_known_good)) {
			_wifi_runtime_controller
				->project_observation(WifiRuntimeObservation {
					.wifi_supported = next.wifi_supported,
					.wifi_connected = next.wifi_connected,
					.wifi_ap_mode = next.wifi_ap_mode,
					.wifi_ap_client_connected =
						next.wifi_ap_client_connected,
					.station_ip = is_station_wifi_ip(
						next.wifi_ip),
					.wifi_ip = next.wifi_ip,
					.wifi_ssid = next.wifi_ssid,
					.wifi_error = next.wifi_error,
				})
				.apply_to(next);
		}
		_last_wifi_refresh = std::chrono::steady_clock::now();
		if (hardware_version_due) {
			const auto started_at =
				std::chrono::steady_clock::now();
			if (detail::collect_hardware_version_status(server_transport, next)) {
				observed_domains |= RefreshObservationDomain::HardwareVersion;
			}
			const auto ended_at = std::chrono::steady_clock::now();
			hardware_version_duration_ms =
				elapsed_ms(started_at, ended_at);
			_last_hardware_version_refresh = ended_at;
		} else {
			next.hw_version = current.hw_version;
		}
		if (systemd_due && !_systemd_refresh_pending) {
			launch_systemd_status_refresh();
		}
		if (detail::collect_ncm_status(_ncm_marker_paths, next)) {
			observed_domains |= RefreshObservationDomain::Ncm;
			_ncm_enabled.store(next.ncm_enabled,
					    std::memory_order_release);
			_ncm_supported.store(true, std::memory_order_release);
		}
		if (detail::collect_backlight_status(_config.nanokvm, backlight_percent,
						   screen_sleeping, next)) {
			observed_domains |= RefreshObservationDomain::Backlight;
		}

		if (_more_page_visible.load(std::memory_order_acquire)) {
			collect_sysinfo_static();
			next.device_key = _sysinfo_static_cache.device_key;
			next.hostname = _sysinfo_static_cache.hostname;
			next.device_number =
				_sysinfo_static_cache.device_number;
			next.image_version =
				_sysinfo_static_cache.image_version;
			next.edid_name = _sysinfo_static_cache.edid_name;
			next.mac_address = _sysinfo_static_cache.mac_address;
			next.mdns_name = _sysinfo_static_cache.mdns_name;
			observed_domains |=
				RefreshObservationDomain::SystemInfoStatic;

			RuntimeStatus active_tmp;
			const bool active_ok = detail::collect_temperature(active_tmp) &&
					       detail::collect_memory(active_tmp) &&
					       detail::collect_uptime(active_tmp) &&
					       detail::collect_local_time(active_tmp);
			if (active_ok) {
				next.temperature_text =
					active_tmp.temperature_text;
				next.memory_text = active_tmp.memory_text;
				next.uptime_text = active_tmp.uptime_text;
				next.local_time_text =
					active_tmp.local_time_text;
				observed_domains |=
					RefreshObservationDomain::SystemInfoActive;
			}
			if (detail::domain_refresh_due(_last_storage_refresh, now,
					       kStorageRefreshInterval)) {
				if (detail::collect_storage(next)) {
					observed_domains |=
						RefreshObservationDomain::
							SystemInfoStorage;
				}
				_last_storage_refresh =
					std::chrono::steady_clock::now();
			}
			if (detail::domain_refresh_due(_last_wifi_signal_refresh, now,
					       kWifiSignalRefreshInterval)) {
				if (detail::collect_wifi_signal(next,
							next.wifi_connected)) {
					observed_domains |=
						RefreshObservationDomain::
							SystemInfoSignal;
				}
				_last_wifi_signal_refresh =
					std::chrono::steady_clock::now();
			}
		}
		if (streamer_due) {
			(void)refresh_display_metrics();
		}
	} catch (const std::exception &e) {
		spdlog::warn("NanoKVM runtime status refresh failed: {}",
			     e.what());
	} catch (...) {
		spdlog::warn(
			"NanoKVM runtime status refresh failed with unknown exception");
	}

	next.auto_sleep_enabled = auto_sleep_enabled;
	next.hdmi_input_supported = true;
	next.hdmi_input_enabled = video_enabled;
	next.video_enabled = video_enabled;
	next.hid_supported = true;
	next.hid_enabled = hid_enabled;
	if (!hid_enabled) {
		next.hid_mode.clear();
	}
	next.updated_at = std::chrono::steady_clock::now();
	observed_domains |= RefreshObservationDomain::RuntimeFlags;
	if (next.server_available) {
		observed_domains |= RefreshObservationDomain::ServerAvailability;
	}
	const RefreshObservationDomain merge_domains =
		preserve_last_known_good ? observed_domains :
						   RefreshObservationDomain::All;
	if (preserve_last_known_good &&
	    merge_domains != RefreshObservationDomain::All) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime refresh retained unobserved domains mask={} observed_mask={}",
			static_cast<uint16_t>(RefreshObservationDomain::All) &
				~static_cast<uint16_t>(merge_domains),
			static_cast<uint16_t>(merge_domains));
	}
	spdlog::info(
		"[runtime] NanoKVM runtime refresh domains systemd_due={} systemd_pending={} hardware_version_due={} hardware_version_duration_ms={} streamer_due={} display_failures={} wifi_age_ms={} class={} key={} state_owner={}",
		systemd_due, _systemd_refresh_pending, hardware_version_due,
		hardware_version_duration_ms, streamer_due,
		_display_metrics_failure_count,
		_last_wifi_refresh == std::chrono::steady_clock::time_point {} ?
			0 :
			elapsed_ms(_last_wifi_refresh, next.updated_at),
		kFullRefreshMetadata.task_class, kFullRefreshMetadata.key,
		kFullRefreshMetadata.state_owner);
	publish_status(RefreshObservationPatch {
		.status = std::move(next),
		.wifi_operation_generation = wifi_operation_generation,
		.observed_domains = merge_domains,
	});
}

void NanoKvmRuntimeService::launch_local_service_action(
	LocalServiceActionKind kind, bool target_enabled)
{
	if (!_running.load(std::memory_order_acquire)) {
		return;
	}

	uint64_t generation = 0;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (_local_services_pending) {
			if (kind == LocalServiceActionKind::Refresh) {
				_local_services_refresh_after_pending = true;
			} else {
				_local_services_queued_actions.push(
					QueuedLocalServiceAction {
						.kind = kind,
						.target_enabled =
							target_enabled,
					});
			}
			return;
		}
		_local_services_pending = true;
		generation = ++_local_services_generation;
	}

	const char *task_name = "local_services_refresh";
	if (kind == LocalServiceActionKind::SetMcpBundle) {
		task_name = "local_services_set_mcp_bundle";
	} else if (kind == LocalServiceActionKind::SetScreenTimelapse) {
		task_name = "local_services_set_screen_timelapse";
	}

	const bool submitted = _runtime_thread_pool.submit(task_name, [this,
								       generation,
								       kind,
								       target_enabled] {
		const auto started_at = std::chrono::steady_clock::now();
		ui::local_service_state::ClientResult result;
		switch (kind) {
		case LocalServiceActionKind::Refresh:
			result = ui::local_service_state::get_services();
			break;
		case LocalServiceActionKind::SetMcpBundle: {
			const auto current =
				ui::local_service_state::get_services();
			if (!current) {
				result = current;
				break;
			}
			if (!current.services.mcp_supported) {
				result = current;
				break;
			}
			result = ui::local_service_state::set_mcp_bundle(
				target_enabled,
				current.services.memory_fabric_supported);
			break;
		}
		case LocalServiceActionKind::SetScreenTimelapse: {
			const auto current =
				ui::local_service_state::get_services();
			if (!current) {
				result = current;
				break;
			}
			if (!current.services.screen_timelapse_supported) {
				result = current;
				break;
			}
			result = ui::local_service_state::set_service(
				ui::local_service_state::kScreenTimelapseServiceID,
				target_enabled);
			break;
		}
		}
		const auto fields = to_local_service_result_fields(result);
		const auto ended_at = std::chrono::steady_clock::now();
		const int64_t duration_ms = elapsed_ms(started_at, ended_at);
		const bool queued = enqueue_action(
			"merge_local_service_result",
			[this, generation, kind, target_enabled, fields,
			 duration_ms]() mutable {
				merge_local_service_result(
					generation, kind, target_enabled,
					fields.ok, fields.status_code,
					fields.mcp_supported, fields.mcp_active,
					fields.replay_supported,
					fields.replay_active,
					std::move(fields.error), duration_ms);
			},
			false);
		if (!queued) {
			spdlog::warn(
				"NanoKVM runtime failed to queue local service result generation={}",
				generation);
		}
	});
	if (submitted) {
		return;
	}

	merge_local_service_result(
		generation, kind, target_enabled, false, 0, false, false, false,
		false, "runtime thread pool rejected local service task", 0);
}

void NanoKvmRuntimeService::merge_local_service_result(
	uint64_t generation, LocalServiceActionKind kind, bool target_enabled,
	bool ok, int status_code, bool mcp_supported, bool mcp_active,
	bool replay_supported, bool replay_active, std::string error,
	int64_t duration_ms)
{
	bool start_refresh = false;
	std::optional<QueuedLocalServiceAction> next_action;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (generation != _local_services_generation) {
			return;
		}
		_local_services_pending = false;
		if (!_local_services_queued_actions.empty()) {
			next_action = _local_services_queued_actions.front();
			_local_services_queued_actions.pop();
		} else if (_local_services_refresh_after_pending) {
			_local_services_refresh_after_pending = false;
			start_refresh = true;
		}
	}

	RuntimeStatus current = current_status();
	RuntimeStatus next = current;
	next.local_services_event_seq = generation;
	next.local_services_refresh_ok = ok;
	next.local_services_error = std::move(error);
	if (ok) {
		next.local_services_known = true;
		next.mcp_service_supported = mcp_supported;
		next.mcp_service_active = mcp_active;
		next.replay_service_supported = replay_supported;
		next.replay_service_active = replay_active;
	} else if (!next.local_services_known) {
		next.mcp_service_supported = false;
		next.mcp_service_active = false;
		next.replay_service_supported = false;
		next.replay_service_active = false;
	}
	next.updated_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime local services result generation={} kind={} target={} ok={} status={} duration_ms={} error={}",
		generation,
		kind == LocalServiceActionKind::Refresh	     ? "refresh" :
		kind == LocalServiceActionKind::SetMcpBundle ? "set_mcp" :
							       "set_replay",
		target_enabled, ok, status_code, duration_ms,
		next.local_services_error);
	if (detail::local_service_status_changed(current, next)) {
		publish_status(
			LocalServicesPatch { .status = std::move(next) });
	}

	if (next_action.has_value()) {
		launch_local_service_action(next_action->kind,
					    next_action->target_enabled);
	} else if (start_refresh) {
		launch_local_service_action(LocalServiceActionKind::Refresh);
	}
}

std::chrono::milliseconds NanoKvmRuntimeService::refresh_display_metrics()
{
	const auto now = std::chrono::steady_clock::now();
	const bool video_enabled =
		_video_enabled.load(std::memory_order_acquire);
	if (!video_enabled) {
		++_display_metrics_refresh_generation;
		_display_metrics_refresh_pending = false;
		_display_metrics_failure_count = 0;
		_display_metrics_backoff_until = {};

		const bool hid_enabled =
			_hid_enabled.load(std::memory_order_acquire);
		RuntimeStatus current = current_status();
		RuntimeStatus next = current;
		detail::sync_runtime_flags(
			next, false, hid_enabled,
			_ncm_enabled.load(std::memory_order_acquire),
			_ncm_supported.load(std::memory_order_acquire),
			_auto_sleep_enabled.load(std::memory_order_acquire),
			_backlight_percent.load(std::memory_order_acquire),
			_screen_sleeping.load(std::memory_order_acquire));
		detail::clear_display_metrics(next);
		if (detail::display_metrics_changed(current, next) ||
		    current.hdmi_input_enabled != next.hdmi_input_enabled ||
		    current.video_enabled != next.video_enabled) {
			next.updated_at = std::chrono::steady_clock::now();
			publish_status(RuntimeFlagsPatch { .status = next });
			publish_status(DisplayMetricsPatch {
				.status = std::move(next) });
		}
		return kDisplayMetricsInterval;
	}

	if (_display_metrics_refresh_pending) {
		return display_metrics_interval_for_failures(
			_display_metrics_failure_count);
	}
	if (_display_metrics_backoff_until > now) {
		return std::chrono::duration_cast<std::chrono::milliseconds>(
			_display_metrics_backoff_until - now);
	}

	const uint64_t generation = ++_display_metrics_refresh_generation;
	_display_metrics_refresh_pending = true;

	auto *const server_transport = _server_transport.get();
	const bool submitted = _runtime_thread_pool.submit(
		"display_metrics", [this, generation, server_transport] {
			RuntimeStatus snapshot;
			snapshot.server_available = false;
			bool ok = true;
			std::string error;
			const auto started_at =
				std::chrono::steady_clock::now();
			try {
				detail::collect_streamer_status(*server_transport, snapshot);
				if (!snapshot.server_available) {
					ok = false;
					error = "server_unavailable";
				}
			} catch (const std::exception &e) {
				ok = false;
				error = e.what();
			} catch (...) {
				ok = false;
				error = "unknown";
			}
			const auto ended_at = std::chrono::steady_clock::now();
			enqueue_action(
				"merge_display_metrics",
				[this, generation, status = std::move(snapshot),
				 duration_ms = elapsed_ms(started_at, ended_at),
				 ok, error = std::move(error)]() mutable {
					merge_display_metrics(generation,
							      std::move(status),
							      duration_ms, ok,
							      std::move(error));
				},
				false, true);
		});
	if (!submitted) {
		_display_metrics_refresh_pending = false;
		_display_metrics_failure_count = std::min<uint8_t>(
			_display_metrics_failure_count + 1U, 8U);
		const auto interval = display_metrics_interval_for_failures(
			_display_metrics_failure_count);
		_display_metrics_backoff_until = now + interval;
		spdlog::warn(
			"NanoKVM runtime display metrics submit failed generation={} backoff_ms={} class={} key={} state_owner={}",
			generation, interval.count(),
			kDisplayMetricsMetadata.task_class,
			kDisplayMetricsMetadata.key,
			kDisplayMetricsMetadata.state_owner);
		return interval;
	}

	SPDLOG_DEBUG(
		"[runtime] NanoKVM runtime display metrics submitted generation={} class={} key={} state_owner={}",
		generation, kDisplayMetricsMetadata.task_class,
		kDisplayMetricsMetadata.key,
		kDisplayMetricsMetadata.state_owner);
	return display_metrics_interval_for_failures(
		_display_metrics_failure_count);
}

void NanoKvmRuntimeService::launch_systemd_status_refresh()
{
	if (_systemd_refresh_pending) {
		return;
	}

	const uint64_t generation = ++_systemd_refresh_generation;
	_systemd_refresh_pending = true;
	const bool preserve_last_known_good =
		_config.nanokvm.preserve_last_known_good_state;
	const bool submitted = _runtime_thread_pool.submit(
		"systemd_status", [this, generation, preserve_last_known_good] {
			RuntimeStatus snapshot;
			bool ok = true;
			std::string error;
			const auto started_at =
				std::chrono::steady_clock::now();
			try {
				const bool observed = detail::collect_systemd_status(
					snapshot, preserve_last_known_good);
				if (!observed && preserve_last_known_good) {
					ok = false;
					error = "inconclusive";
				}
			} catch (const std::exception &e) {
				ok = false;
				error = e.what();
			} catch (...) {
				ok = false;
				error = "unknown";
			}
			const auto ended_at = std::chrono::steady_clock::now();
			enqueue_action(
				"merge_systemd_status",
				[this, generation, status = std::move(snapshot),
				 duration_ms = elapsed_ms(started_at, ended_at),
				 ok, error = std::move(error)]() mutable {
					merge_systemd_status(generation,
							     std::move(status),
							     duration_ms, ok,
							     std::move(error));
				},
				false);
		});
	if (!submitted) {
		_systemd_refresh_pending = false;
		_last_systemd_refresh = std::chrono::steady_clock::now();
		spdlog::warn(
			"NanoKVM runtime systemd status submit failed generation={} class={} key=systemd state_owner=RuntimeStatus::systemd",
			generation, kFullRefreshMetadata.task_class);
		return;
	}

	SPDLOG_DEBUG(
		"[runtime] NanoKVM runtime systemd status submitted generation={} class={} key=systemd state_owner=RuntimeStatus::systemd",
		generation, kFullRefreshMetadata.task_class);
}

void NanoKvmRuntimeService::merge_systemd_status(uint64_t generation,
						 RuntimeStatus status,
						 int64_t duration_ms, bool ok,
						 std::string error)
{
	if (generation != _systemd_refresh_generation) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime systemd status stale generation={} current={} duration_ms={}",
			generation, _systemd_refresh_generation, duration_ms);
		return;
	}

	_systemd_refresh_pending = false;
	_last_systemd_refresh = std::chrono::steady_clock::now();
	if (!ok) {
		spdlog::warn(
			"NanoKVM runtime systemd status merge failed generation={} duration_ms={} error={} class={} key=systemd state_owner=RuntimeStatus::systemd",
			generation, duration_ms, error,
			kFullRefreshMetadata.task_class);
		return;
	}

	RuntimeStatus current = current_status();
	RuntimeStatus next = current;
	detail::copy_systemd_status_fields(status, next);
	spdlog::info(
		"[runtime] NanoKVM runtime systemd status merge generation={} duration_ms={} changed={} ssh={} ssh_supported={} class={} key=systemd state_owner=RuntimeStatus::systemd",
		generation, duration_ms,
		detail::systemd_status_changed(current, next),
		next.ssh_enabled, next.ssh_supported,
		kFullRefreshMetadata.task_class);
	if (!detail::systemd_status_changed(current, next)) {
		return;
	}
	next.updated_at = std::chrono::steady_clock::now();
	publish_status(SystemdStatusPatch { .status = std::move(next) });
}

void NanoKvmRuntimeService::merge_display_metrics(uint64_t generation,
						  RuntimeStatus status,
						  int64_t duration_ms, bool ok,
						  std::string error)
{
	if (generation != _display_metrics_refresh_generation) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime display metrics stale generation={} current={} duration_ms={}",
			generation, _display_metrics_refresh_generation,
			duration_ms);
		return;
	}

	_display_metrics_refresh_pending = false;
	const auto now = std::chrono::steady_clock::now();
	if (!ok) {
		_display_metrics_failure_count = std::min<uint8_t>(
			_display_metrics_failure_count + 1U, 8U);
		const auto interval = display_metrics_interval_for_failures(
			_display_metrics_failure_count);
		_display_metrics_backoff_until = now + interval;
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime display metrics merge failed generation={} duration_ms={} error={} failures={} backoff_ms={} class={} key={} state_owner={}",
			generation, duration_ms, error,
			_display_metrics_failure_count, interval.count(),
			kDisplayMetricsMetadata.task_class,
			kDisplayMetricsMetadata.key,
			kDisplayMetricsMetadata.state_owner);
		return;
	}

	_display_metrics_failure_count = 0;
	_display_metrics_backoff_until = {};
	const bool video_enabled =
		_video_enabled.load(std::memory_order_acquire);
	const bool hid_enabled = _hid_enabled.load(std::memory_order_acquire);
	RuntimeStatus current = current_status();
	RuntimeStatus next = current;
	if (video_enabled) {
		detail::copy_display_metrics_fields(status, next);
		next.server_available = true;
	} else {
		detail::clear_display_metrics(next);
	}
	detail::sync_runtime_flags(next, video_enabled, hid_enabled,
			   _ncm_enabled.load(std::memory_order_acquire),
			   _ncm_supported.load(std::memory_order_acquire),
			   _auto_sleep_enabled.load(std::memory_order_acquire),
			   _backlight_percent.load(std::memory_order_acquire),
			   _screen_sleeping.load(std::memory_order_acquire));
	SPDLOG_DEBUG(
		"[runtime] NanoKVM runtime display metrics merge generation={} duration_ms={} changed={} resolution='{}' fps={} bitrate={} class={} key={} state_owner={}",
		generation, duration_ms,
		detail::display_metrics_changed(current, next),
		next.resolution, next.fps.value_or(0.0),
		next.h264_bitrate_kbps.value_or(0),
		kDisplayMetricsMetadata.task_class, kDisplayMetricsMetadata.key,
		kDisplayMetricsMetadata.state_owner);
	if (!detail::display_metrics_changed(current, next) &&
	    current.video_enabled == next.video_enabled &&
	    current.hdmi_input_enabled == next.hdmi_input_enabled) {
		return;
	}
	next.updated_at = std::chrono::steady_clock::now();
	publish_status(DisplayMetricsPatch { .status = next });
	publish_status(RuntimeFlagsPatch { .status = std::move(next) });
}

void NanoKvmRuntimeService::observe_initial_ncm_state()
{
	const NcmObservation observation =
		observe_ncm_boot_markers(_ncm_marker_paths);
	const auto value = ncm_observation_enabled(observation);
	if (value) {
		_ncm_enabled.store(*value, std::memory_order_release);
		_ncm_supported.store(true, std::memory_order_release);
		return;
	}
	spdlog::warn(
		"[runtime] NanoKVM NCM boot marker observation inconclusive; NCM status reported as unsupported until next refresh");
}

void NanoKvmRuntimeService::publish_local_runtime_status_snapshot()
{
	RuntimeStatus next = current_status();
	detail::sync_runtime_flags(
		next, _video_enabled.load(std::memory_order_acquire),
			   _hid_enabled.load(std::memory_order_acquire),
			   _ncm_enabled.load(std::memory_order_acquire),
			   _ncm_supported.load(std::memory_order_acquire),
			   _auto_sleep_enabled.load(std::memory_order_acquire),
			   _backlight_percent.load(std::memory_order_acquire),
			   _screen_sleeping.load(std::memory_order_acquire));
	if (!next.video_enabled) {
		detail::clear_display_metrics(next);
		next.server_available = false;
	}
	next.updated_at = std::chrono::steady_clock::now();
	publish_status(RuntimeFlagsPatch { .status = next });
	if (!next.video_enabled) {
		publish_status(
			DisplayMetricsPatch { .status = std::move(next) });
	}
}

void NanoKvmRuntimeService::publish_factory_reset_failure()
{
	const uint64_t generation = _factory_reset_failure_generation.fetch_add(
					    1, std::memory_order_acq_rel) +
				    1U;
	publish_status(FactoryResetPatch { .failure_generation = generation });
}

void NanoKvmRuntimeService::publish_region_switch_failure()
{
	const uint64_t generation = _region_switch_failure_generation.fetch_add(
					    1, std::memory_order_acq_rel) +
				    1U;
	publish_status(RegionSwitchPatch { .failure_generation = generation });
}

void NanoKvmRuntimeService::publish_status(RuntimeStatusPatch patch,
					   bool notify_callback)
{
	const auto publication = _state_store.commit(patch);
	if (!publication.has_value() || !notify_callback) {
		return;
	}
	if (_callbacks.on_status) {
		try {
			_callbacks.on_status(*publication);
		} catch (const std::exception &e) {
			spdlog::warn(
				"NanoKVM runtime status callback failed: {}",
				e.what());
		} catch (...) {
			spdlog::warn(
				"NanoKVM runtime status callback failed with unknown exception");
		}
	}
}

void NanoKvmRuntimeService::publish_wifi_status(
	const RuntimeStatus &status, WifiLifecyclePatch::Phase phase,
	uint64_t operation_generation,
	bool preserve_ap_stop_completion_when_stale, bool notify_callback)
{
	publish_status(
		WifiLifecyclePatch {
			.status = status,
			.operation_generation = operation_generation,
			.phase = phase,
			.preserve_ap_stop_completion_when_stale =
				preserve_ap_stop_completion_when_stale,
		},
		notify_callback);
}

void NanoKvmRuntimeService::publish_tailscale_session(
	const TailscaleSession &session, bool force)
{
	const RuntimeStatus current = current_status();
	const bool changed = current.tailscale_online != session.online ||
			     current.tailscale_code != session.code ||
			     current.tailscale_peer_name !=
				     session.peer_name ||
			     current.tailscale_peer_ip != session.peer_ip;
	if (!changed && !force) {

		return;
	}

	RuntimeStatus next;
	next.tailscale_online = session.online;
	next.tailscale_code = session.code;
	next.tailscale_peer_name = session.peer_name;
	next.tailscale_peer_ip = session.peer_ip;
	next.tailscale_seq = current.tailscale_seq + 1;
	publish_status(TailscalePatch { .status = std::move(next) });
}

std::chrono::steady_clock::time_point
NanoKvmRuntimeService::refresh_tailscale_tick(
	bool try_code, uint8_t &heartbeat_failure_count,
	uint8_t &code_failure_count,
	std::chrono::steady_clock::time_point &next_code_refresh)
{
	using Clock = std::chrono::steady_clock;
	const auto started_at = Clock::now();
	if (!_tailscale_client) {
		return tailscale_next_refresh_at(started_at, false,
						 heartbeat_failure_count);
	}

	if (_tailscale_relinquish_pending.exchange(false,
						  std::memory_order_acq_rel)) {
		if (!_tailscale_client->request_disable()) {
			spdlog::warn(
				"[runtime] NanoKVM tailscale disable request failed: {}",
				_tailscale_client->last_error());
		}
	}

	const uint64_t action_generation =
		_tailscale_action_generation.load(std::memory_order_acquire);

	const auto snapshot = _tailscale_client->fetch_snapshot();
	if (!snapshot.has_value()) {
		spdlog::warn("[runtime] NanoKVM tailscale heartbeat failed: {}",
			     _tailscale_client->last_error());
		if (action_generation == _tailscale_action_generation.load(
						 std::memory_order_acquire)) {
			publish_tailscale_snapshot_failure(action_generation);
		}
		return tailscale_next_refresh_at(started_at, false,
						 heartbeat_failure_count);
	}

	if (action_generation !=
	    _tailscale_action_generation.load(std::memory_order_acquire)) {
		return tailscale_next_refresh_at(started_at, true,
						 heartbeat_failure_count);
	}

	publish_tailscale_snapshot(*snapshot, action_generation);
	const auto next_heartbeat = tailscale_next_refresh_at(
		started_at, true, heartbeat_failure_count);

	if (!try_code) {
		return next_heartbeat;
	}

	const auto code = _tailscale_client->fetch_code();
	bool code_ok = false;
	if (code.has_value()) {
		if (is_tailscale_sas_code(*code)) {
			code_ok = true;
		} else {
			const auto session =
				tailscale_session_from_code(*code);
			if (session.has_value()) {
				publish_tailscale_session(*session, false);
				code_ok = true;
			}
		}
	}
	if (!code_ok) {
		const RuntimeStatus code_failure_current = current_status();
		spdlog::warn(
			"[runtime] NanoKVM tailscale login code unavailable ({}); retaining code display",
			_tailscale_client->last_error());
		publish_tailscale_session(
			TailscaleSession {
				.online = false,
				.code = code_failure_current.tailscale_code },
			true);
	}
	next_code_refresh = tailscale_next_refresh_at(
		started_at, code_ok, code_failure_count);
	return next_heartbeat;
}

namespace
{

TailscaleActionKind tailscale_action_kind_for(TailscaleAction action)
{
	switch (action) {
	case TailscaleAction::Start:
		return TailscaleActionKind::Start;
	case TailscaleAction::Exit:
		return TailscaleActionKind::Exit;
	}
	return TailscaleActionKind::None;
}

}

void NanoKvmRuntimeService::publish_tailscale_snapshot(
	const TailscaleSnapshot &snapshot, uint64_t action_generation)
{
	const RuntimeStatus gate_base = current_status();
	RuntimeStatus gate;
	gate.tailscale_online = snapshot.online;
	gate.tailscale_started = snapshot.started;
	gate.tailscale_matched = snapshot.matched;
	gate.tailscale_peer_name = snapshot.name;
	gate.tailscale_peer_ip = snapshot.ip;
	gate.tailscale_gate_seq = gate_base.tailscale_gate_seq + 1;
	gate.tailscale_action_generation = action_generation;
	publish_status(TailscaleSnapshotPatch {
		.status = std::move(gate), .snapshot_valid = true });
}

void NanoKvmRuntimeService::publish_tailscale_snapshot_failure(
	uint64_t action_generation)
{
	const RuntimeStatus current = current_status();
	RuntimeStatus failure;
	failure.tailscale_online = false;
	failure.tailscale_seq = current.tailscale_seq + 1;
	failure.tailscale_action_generation = action_generation;
	publish_status(TailscaleSnapshotPatch {
		.status = std::move(failure), .snapshot_valid = false });
}

bool NanoKvmRuntimeService::launch_tailscale_action(TailscaleAction action)
{
	if (!_running.load(std::memory_order_acquire) ||
	    !_tailscale_client) {
		return false;
	}

	uint64_t generation = 0;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (_tailscale_action_pending) {
			return false;
		}
		_tailscale_action_pending = true;
		generation = _tailscale_action_generation.fetch_add(
				     1, std::memory_order_acq_rel) +
			     1;
	}

	const bool submitted = _runtime_thread_pool.submit(
		"tailscale_action", [this, generation, action] {
			const TailscaleActionResult result =
				_tailscale_client->request_action(action);
			const std::string message =
				(!result.ok && result.message.empty()) ?
					std::string(
						kTailscaleActionFailMessage) :
					result.message;
			const bool queued = enqueue_action(
				"merge_tailscale_action_result",
				[this, generation, action, ok = result.ok,
				 message]() mutable {
					merge_tailscale_action_result(
						generation, action, ok,
						std::move(message));
				},
				false);
			if (!queued) {
				spdlog::warn(
					"NanoKVM runtime failed to queue tailscale action result generation={}",
					generation);
			}
		});
	if (submitted) {
		return true;
	}

	merge_tailscale_action_result(
		generation, action, false,
		std::string(kTailscaleActionFailMessage));
	return false;
}

void NanoKvmRuntimeService::merge_tailscale_action_result(
	uint64_t generation, TailscaleAction action, bool ok,
	std::string message)
{
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (generation != _tailscale_action_generation.load(
					  std::memory_order_acquire)) {
			return;
		}
		_tailscale_action_pending = false;
	}

	const RuntimeStatus current = current_status();
	RuntimeStatus next;
	next.tailscale_action_seq = current.tailscale_action_seq + 1;
	next.tailscale_action_kind = tailscale_action_kind_for(action);
	next.tailscale_action_ok = ok;
	next.tailscale_action_message = std::move(message);
	next.tailscale_action_generation = generation;
	publish_status(
		TailscaleActionResultPatch { .status = std::move(next) });

	_tailscale_refresh_now.store(true, std::memory_order_release);
	_queue_cv.notify_all();
}

bool NanoKvmRuntimeService::request_tailscale_start()
{
	return launch_tailscale_action(TailscaleAction::Start);
}

bool NanoKvmRuntimeService::request_tailscale_exit()
{
	return launch_tailscale_action(TailscaleAction::Exit);
}

}
