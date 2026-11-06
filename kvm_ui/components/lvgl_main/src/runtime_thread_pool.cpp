#include "lvgl_main/runtime_thread_pool.hpp"

#include <algorithm>
#include <chrono>
#include <exception>
#include <utility>

#include <spdlog/spdlog.h>

namespace lvgl_main
{

namespace
{

constexpr std::size_t kMinWorkerCount { 1 };

int64_t elapsed_ms(std::chrono::steady_clock::time_point started_at,
		   std::chrono::steady_clock::time_point ended_at)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(ended_at -
								     started_at)
		.count();
}

} // namespace

RuntimeThreadPool::RuntimeThreadPool(std::size_t worker_count)
	: _worker_count(std::clamp(worker_count, kMinWorkerCount,
				   RuntimeThreadPool::kMaxWorkerCount))
{
	if (_worker_count != worker_count) {
		spdlog::warn(
			"NanoKVM runtime pool worker count clamped requested={} actual={} max={}",
			worker_count, _worker_count,
			RuntimeThreadPool::kMaxWorkerCount);
	}
}

RuntimeThreadPool::~RuntimeThreadPool()
{
	stop();
}

bool RuntimeThreadPool::start()
{
	bool expected = false;
	if (!_running.compare_exchange_strong(expected, true,
					      std::memory_order_acq_rel)) {
		return false;
	}

	{
		std::lock_guard<std::mutex> lock(_mutex);
		_accepting = true;
	}

	try {
		_workers.reserve(_worker_count);
		for (std::size_t index = 0; index < _worker_count; ++index) {
			_workers.emplace_back(&RuntimeThreadPool::worker_loop,
					      this, index);
		}
	} catch (const std::exception &e) {
		spdlog::error("NanoKVM runtime pool start failed: {}",
			      e.what());
		stop();
		return false;
	} catch (...) {
		spdlog::error(
			"NanoKVM runtime pool start failed with unknown exception");
		stop();
		return false;
	}

	spdlog::info("[runtime-pool] NanoKVM runtime pool started workers={}",
		     _worker_count);
	return true;
}

void RuntimeThreadPool::stop()
{
	const bool was_running =
		_running.exchange(false, std::memory_order_acq_rel);
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_accepting = false;
	}
	_cv.notify_all();

	for (std::thread &worker : _workers) {
		if (worker.joinable()) {
			worker.join();
		}
	}
	_workers.clear();

	std::size_t dropped_count = 0;
	{
		std::lock_guard<std::mutex> lock(_mutex);
		dropped_count = _tasks.size();
		std::queue<Task> empty;
		_tasks.swap(empty);
	}

	if (dropped_count > 0) {
		spdlog::warn(
			"NanoKVM runtime pool dropped queued tasks count={}",
			dropped_count);
	}
	if (was_running || dropped_count > 0) {
		spdlog::info(
			"[runtime-pool] NanoKVM runtime pool stopped workers={} dropped={}",
			_worker_count, dropped_count);
	}
}

bool RuntimeThreadPool::submit(std::string name, std::function<void()> task)
{
	if (!task) {
		spdlog::warn("NanoKVM runtime pool rejected empty task");
		return false;
	}
	if (name.empty()) {
		name = "anonymous";
	}

	const std::string task_name = name;
	uint64_t task_id = 0;
	std::size_t queue_depth = 0;
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (!_accepting || !_running.load(std::memory_order_acquire)) {
			spdlog::warn(
				"NanoKVM runtime pool task rejected name={} reason=not_running",
				name);
			return false;
		}

		task_id =
			_next_task_id.fetch_add(1, std::memory_order_relaxed) +
			1U;
		_tasks.push(Task {
			.id = task_id,
			.name = std::move(name),
			.run = std::move(task),
			.enqueued_at = std::chrono::steady_clock::now(),
		});
		queue_depth = _tasks.size();
	}

	spdlog::info(
		"[runtime-pool] NanoKVM runtime pool task submit id={} name={} queue_depth={}",
		task_id, task_name, queue_depth);
	_cv.notify_one();
	return true;
}

bool RuntimeThreadPool::is_running() const
{
	return _running.load(std::memory_order_acquire);
}

void RuntimeThreadPool::worker_loop(std::size_t worker_index)
{
	while (true) {
		Task task;
		{
			std::unique_lock<std::mutex> lock(_mutex);
			_cv.wait(lock, [this] {
				return !_running.load(
					       std::memory_order_acquire) ||
				       !_tasks.empty();
			});
			if (!_running.load(std::memory_order_acquire)) {
				break;
			}
			task = std::move(_tasks.front());
			_tasks.pop();
		}

		const auto started_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime-pool] NanoKVM runtime pool task start worker={} id={} name={} wait_ms={}",
			worker_index, task.id, task.name,
			elapsed_ms(task.enqueued_at, started_at));
		try {
			task.run();
		} catch (const std::exception &e) {
			const auto failed_at = std::chrono::steady_clock::now();
			spdlog::warn(
				"NanoKVM runtime pool task failed worker={} id={} name={} duration_ms={} error={}",
				worker_index, task.id, task.name,
				elapsed_ms(started_at, failed_at), e.what());
			continue;
		} catch (...) {
			const auto failed_at = std::chrono::steady_clock::now();
			spdlog::warn(
				"NanoKVM runtime pool task failed worker={} id={} name={} duration_ms={} error=unknown",
				worker_index, task.id, task.name,
				elapsed_ms(started_at, failed_at));
			continue;
		}

		const auto ended_at = std::chrono::steady_clock::now();
		spdlog::info(
			"[runtime-pool] NanoKVM runtime pool task end worker={} id={} name={} duration_ms={}",
			worker_index, task.id, task.name,
			elapsed_ms(started_at, ended_at));
	}
}

} // namespace lvgl_main
