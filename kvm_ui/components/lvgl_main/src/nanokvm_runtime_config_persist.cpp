#include "lvgl_main/nanokvm_runtime.hpp"

#include <chrono>
#include <exception>
#include <mutex>
#include <string_view>
#include <thread>
#include <utility>

#include <spdlog/spdlog.h>

namespace lvgl_main
{

namespace
{

constexpr std::chrono::milliseconds kConfigPersistDebounce { 200 };

struct RuntimeTaskMetadata {
	std::string_view task_class;
	std::string_view key;
	std::string_view state_owner;
};

constexpr RuntimeTaskMetadata kConfigPersistMetadata { "config-persist-action",
						       "app_config",
						       "AppConfig" };

int64_t elapsed_ms(std::chrono::steady_clock::time_point start,
			   std::chrono::steady_clock::time_point end)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(end -
								     start)
		.count();
}

} // namespace

void NanoKvmRuntimeService::schedule_config_persist(std::string reason)
{
	uint64_t generation = 0;
	std::string scheduled_reason;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		generation = ++_config_persist_generation;
		_config_persist_scheduled = true;
		_config_persist_deadline = std::chrono::steady_clock::now() +
					   kConfigPersistDebounce;
		_config_persist_deadline_generation = generation;
		_config_persist_deadline_reason = std::move(reason);
		scheduled_reason = _config_persist_deadline_reason;
	}
	SPDLOG_DEBUG(
		"[runtime] NanoKVM runtime config persist scheduled generation={} delay_ms={} reason={} class={} key={} state_owner={}",
		generation, kConfigPersistDebounce.count(), scheduled_reason,
		kConfigPersistMetadata.task_class, kConfigPersistMetadata.key,
		kConfigPersistMetadata.state_owner);
	_queue_cv.notify_all();
}

bool NanoKvmRuntimeService::persist_config_now(std::string_view reason,
					       std::string &error)
{
	AppConfig snapshot;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		_config.nanokvm.video_enabled =
			_video_enabled.load(std::memory_order_acquire);
		_config.nanokvm.hid_enabled =
			_hid_enabled.load(std::memory_order_acquire);
		_config.nanokvm.backlight_percent =
			_backlight_percent.load(std::memory_order_acquire);
		_config.nanokvm.auto_sleep_enabled =
			_auto_sleep_enabled.load(std::memory_order_acquire);
		snapshot = _config;
		_config_persist_scheduled = false;
	}

	const auto started_at = std::chrono::steady_clock::now();
	const bool saved = save_app_config(snapshot, error, _config_path);
	const auto ended_at = std::chrono::steady_clock::now();
	if (!saved) {
		spdlog::warn(
			"NanoKVM runtime failed to persist config immediately reason={}: {}",
			reason, error);
	}
	spdlog::info(
		"[runtime] NanoKVM runtime config persist immediate reason={} ok={} duration_ms={} class={} key={} state_owner={}",
		reason, saved, elapsed_ms(started_at, ended_at),
		kConfigPersistMetadata.task_class, kConfigPersistMetadata.key,
		kConfigPersistMetadata.state_owner);
	return saved;
}

void NanoKvmRuntimeService::submit_config_persist_snapshot(uint64_t generation,
							   std::string reason)
{
	ConfigPersistWork work;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (!_config_persist_scheduled ||
		    _config_persist_deadline_generation != generation ||
		    std::chrono::steady_clock::now() <
			    _config_persist_deadline) {
			return;
		}

		_config.nanokvm.video_enabled =
			_video_enabled.load(std::memory_order_acquire);
		_config.nanokvm.hid_enabled =
			_hid_enabled.load(std::memory_order_acquire);
		_config.nanokvm.backlight_percent =
			_backlight_percent.load(std::memory_order_acquire);
		_config.nanokvm.auto_sleep_enabled =
			_auto_sleep_enabled.load(std::memory_order_acquire);
		work.config = _config;
		work.generation = generation;
		work.reason = std::move(reason);
		_config_persist_scheduled = false;
	}

	bool accepted = false;
	{
		std::lock_guard<std::mutex> lock(_config_persist_mutex);
		if (_config_persist_accepting &&
		    !_config_persist_stop_requested) {
			_config_persist_work = std::move(work);
			accepted = true;
		}
	}
	if (accepted) {
		_config_persist_cv.notify_all();
		return;
	}

	const auto started_at = std::chrono::steady_clock::now();
	std::string error;
	const bool saved = save_app_config(work.config, error, _config_path);
	if (!saved) {
		spdlog::warn(
			"NanoKVM runtime failed to persist config generation={} reason={}: {}",
			work.generation, work.reason, error);
	}
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime config persist fallback end generation={} reason={} ok={} duration_ms={} class={} key={} state_owner={}",
		work.generation, work.reason, saved,
		elapsed_ms(started_at, ended_at),
		kConfigPersistMetadata.task_class, kConfigPersistMetadata.key,
		kConfigPersistMetadata.state_owner);
}

void NanoKvmRuntimeService::flush_config_persist()
{
	ConfigPersistWork work;
	{
		std::lock_guard<std::mutex> lock(_queue_mutex);
		if (!_config_persist_scheduled) {
			return;
		}
		_config.nanokvm.video_enabled =
			_video_enabled.load(std::memory_order_acquire);
		_config.nanokvm.hid_enabled =
			_hid_enabled.load(std::memory_order_acquire);
		_config.nanokvm.backlight_percent =
			_backlight_percent.load(std::memory_order_acquire);
		_config.nanokvm.auto_sleep_enabled =
			_auto_sleep_enabled.load(std::memory_order_acquire);
		work.config = _config;
		work.generation = _config_persist_deadline_generation;
		work.reason = _config_persist_deadline_reason;
		_config_persist_scheduled = false;
	}

	const auto started_at = std::chrono::steady_clock::now();
	std::string error;
	const bool saved = save_app_config(work.config, error, _config_path);
	if (!saved) {
		spdlog::warn(
			"NanoKVM runtime failed to flush config generation={} reason={}: {}",
			work.generation, work.reason, error);
	}
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime config flush end generation={} reason={} ok={} duration_ms={} class={} key={} state_owner={}",
		work.generation, work.reason, saved,
		elapsed_ms(started_at, ended_at),
		kConfigPersistMetadata.task_class, kConfigPersistMetadata.key,
		kConfigPersistMetadata.state_owner);
}

bool NanoKvmRuntimeService::start_config_persist_worker()
{
	{
		std::lock_guard<std::mutex> lock(_config_persist_mutex);
		if (_config_persist_accepting ||
		    _config_persist_worker.joinable()) {
			return false;
		}
		_config_persist_work.reset();
		_config_persist_accepting = true;
		_config_persist_stop_requested = false;
	}

	try {
		_config_persist_worker = std::thread(
			&NanoKvmRuntimeService::config_persist_loop, this);
	} catch (const std::exception &e) {
		spdlog::error(
			"NanoKVM runtime config persist worker start failed: {}",
			e.what());
		std::lock_guard<std::mutex> lock(_config_persist_mutex);
		_config_persist_accepting = false;
		_config_persist_stop_requested = true;
		return false;
	} catch (...) {
		spdlog::error(
			"NanoKVM runtime config persist worker start failed with unknown exception");
		std::lock_guard<std::mutex> lock(_config_persist_mutex);
		_config_persist_accepting = false;
		_config_persist_stop_requested = true;
		return false;
	}

	return true;
}

void NanoKvmRuntimeService::stop_config_persist_worker()
{
	{
		std::lock_guard<std::mutex> lock(_config_persist_mutex);
		_config_persist_accepting = false;
		_config_persist_stop_requested = true;
	}
	_config_persist_cv.notify_all();
	if (_config_persist_worker.joinable()) {
		_config_persist_worker.join();
	}
}

void NanoKvmRuntimeService::config_persist_loop()
{
	while (true) {
		ConfigPersistWork work;
		{
			std::unique_lock<std::mutex> lock(
				_config_persist_mutex);
			_config_persist_cv.wait(lock, [this] {
				return _config_persist_stop_requested ||
				       _config_persist_work.has_value();
			});
			if (!_config_persist_work.has_value() &&
			    _config_persist_stop_requested) {
				break;
			}
			work = std::move(*_config_persist_work);
			_config_persist_work.reset();
		}

		const auto started_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime] NanoKVM runtime config persist start generation={} reason={} class={} key={} state_owner={}",
			work.generation, work.reason,
			kConfigPersistMetadata.task_class,
			kConfigPersistMetadata.key,
			kConfigPersistMetadata.state_owner);
		std::string error;
		{
			std::lock_guard<std::mutex> lock(_queue_mutex);
			if (work.generation != _config_persist_generation) {
				SPDLOG_DEBUG(
					"[runtime] NanoKVM runtime config persist stale generation={} current={} reason={} class={} key={} state_owner={}",
					work.generation,
					_config_persist_generation, work.reason,
					kConfigPersistMetadata.task_class,
					kConfigPersistMetadata.key,
					kConfigPersistMetadata.state_owner);
				continue;
			}
		}
		const bool saved =
			save_app_config(work.config, error, _config_path);
		if (!saved) {
			spdlog::warn(
				"NanoKVM runtime failed to persist config generation={} reason={}: {}",
				work.generation, work.reason, error);
		}
		const auto ended_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime] NanoKVM runtime config persist end generation={} reason={} ok={} duration_ms={} class={} key={} state_owner={}",
			work.generation, work.reason, saved,
			elapsed_ms(started_at, ended_at),
			kConfigPersistMetadata.task_class,
			kConfigPersistMetadata.key,
			kConfigPersistMetadata.state_owner);
	}
}

} // namespace lvgl_main
