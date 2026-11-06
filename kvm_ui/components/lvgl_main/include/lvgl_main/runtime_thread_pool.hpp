#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <functional>
#include <mutex>
#include <queue>
#include <string>
#include <thread>
#include <vector>

namespace lvgl_main
{

class RuntimeThreadPool final {
    public:
	static constexpr std::size_t kDefaultWorkerCount { 2 };
	static constexpr std::size_t kMaxWorkerCount { 3 };

	explicit RuntimeThreadPool(
		std::size_t worker_count = kDefaultWorkerCount);
	~RuntimeThreadPool();

	RuntimeThreadPool(const RuntimeThreadPool &) = delete;
	RuntimeThreadPool &operator=(const RuntimeThreadPool &) = delete;

	bool start();
	void stop();
	bool submit(std::string name, std::function<void()> task);
	bool is_running() const;

    private:
	struct Task {
		uint64_t id { 0 };
		std::string name;
		std::function<void()> run;
		std::chrono::steady_clock::time_point enqueued_at;
	};

	void worker_loop(std::size_t worker_index);

	const std::size_t _worker_count;
	mutable std::mutex _mutex;
	std::condition_variable _cv;
	std::queue<Task> _tasks;
	std::vector<std::thread> _workers;
	std::atomic<bool> _running { false };
	std::atomic<uint64_t> _next_task_id { 0 };
	bool _accepting { false };
};

} // namespace lvgl_main
