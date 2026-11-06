#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <mutex>
#include <optional>
#include <string>
#include <thread>

namespace lvgl_main
{

struct ImuSample {
	int x_raw { 0 };
	int y_raw { 0 };
	int z_raw { 0 };
};

enum class ImuOrientation : uint8_t {
	Unknown,
	Forward,
	Reverse,
};

struct AutoRotationRuntimeConfig {
	std::string iio_device_path { "/sys/bus/iio/devices/iio:device1" };
	int sampling_frequency_hz { 100 };
	std::chrono::milliseconds poll_interval { 125 };
	int confirm_samples { 3 };
};

ImuOrientation classify_imu_orientation(const ImuSample &sample);
bool rotation_for_imu_orientation(ImuOrientation orientation);
std::chrono::milliseconds
remaining_auto_rotation_sleep(std::chrono::milliseconds poll_interval,
			      std::chrono::steady_clock::duration elapsed);
bool probe_auto_rotation_imu(const AutoRotationRuntimeConfig &config,
			     std::string &reason);

class AutoRotationConfirmation final {
    public:
	explicit AutoRotationConfirmation(int confirm_samples,
					  bool initial_reverse = false);

	std::optional<bool> update(ImuOrientation candidate);
	void reset();

    private:
	ImuOrientation _pending_orientation { ImuOrientation::Unknown };
	int _consecutive_samples { 0 };
	int _confirm_samples { 1 };
	bool _last_published_reverse { false };
};

class AutoRotationRuntime final {
    public:
	using OrientationCallback = std::function<void(bool reverse)>;

	AutoRotationRuntime(AutoRotationRuntimeConfig config,
			    OrientationCallback callback);
	~AutoRotationRuntime();

	AutoRotationRuntime(const AutoRotationRuntime &) = delete;
	AutoRotationRuntime &operator=(const AutoRotationRuntime &) = delete;

	bool start();
	void stop();
	bool is_running() const;

    private:
	void worker_loop();
	void sleep_interruptibly(std::chrono::milliseconds duration);
	bool read_imu_sample(ImuSample &sample, std::string &error) const;
	bool probe_iio_buffer_capability(std::string &reason) const;

	AutoRotationRuntimeConfig _config;
	OrientationCallback _callback;
	std::thread _worker;
	std::atomic<bool> _running { false };
	std::atomic<bool> _stop_requested { false };
	mutable std::mutex _mutex;
	std::condition_variable _cv;
};

} // namespace lvgl_main
