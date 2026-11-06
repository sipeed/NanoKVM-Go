#include "lvgl_main/auto_rotation_runtime.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstdlib>
#include <cstring>
#include <exception>
#include <filesystem>
#include <string_view>
#include <utility>

#include <fcntl.h>
#include <spdlog/spdlog.h>
#include <unistd.h>

namespace lvgl_main
{

namespace
{

constexpr std::string_view kSamplingFrequencyFile = "sampling_frequency";
constexpr std::string_view kAccelXRawFile = "in_accel_x_raw";
constexpr std::string_view kAccelYRawFile = "in_accel_y_raw";
constexpr std::string_view kAccelZRawFile = "in_accel_z_raw";
constexpr int kXAxisThreshold = 500;
constexpr int kXAxisDominanceRatioNumerator = 3;
constexpr int kXAxisDominanceRatioDenominator = 2;

class UniqueFd final {
    public:
	explicit UniqueFd(int fd)
		: _fd(fd)
	{
	}

	~UniqueFd()
	{
		if (_fd >= 0) {
			::close(_fd);
		}
	}

	UniqueFd(const UniqueFd &) = delete;
	UniqueFd &operator=(const UniqueFd &) = delete;

	int get() const
	{
		return _fd;
	}

    private:
	int _fd { -1 };
};

std::string sysfs_path(const std::string &root, std::string_view filename)
{
	std::string path(root);
	if (!path.empty() && path.back() != '/') {
		path += '/';
	}
	path += filename;
	return path;
}

bool write_sysfs_text(const std::string &path, std::string_view value,
		      std::string &error)
{
	const UniqueFd fd(::open(path.c_str(), O_WRONLY | O_CLOEXEC));
	if (fd.get() < 0) {
		error = std::strerror(errno);
		return false;
	}

	const char *ptr = value.data();
	std::size_t remaining = value.size();
	while (remaining > 0U) {
		const ssize_t written = ::write(fd.get(), ptr, remaining);
		if (written < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (written == 0) {
			error = "short write";
			return false;
		}
		ptr += written;
		remaining -= static_cast<std::size_t>(written);
	}
	return true;
}

bool read_sysfs_int(const std::string &path, int &value, std::string &error)
{
	const UniqueFd fd(::open(path.c_str(), O_RDONLY | O_CLOEXEC));
	if (fd.get() < 0) {
		error = std::strerror(errno);
		return false;
	}

	std::array<char, 64> buffer {};
	ssize_t bytes_read = 0;
	do {
		bytes_read =
			::read(fd.get(), buffer.data(), buffer.size() - 1U);
	} while (bytes_read < 0 && errno == EINTR);
	if (bytes_read < 0) {
		error = std::strerror(errno);
		return false;
	}
	if (bytes_read == 0) {
		error = "empty read";
		return false;
	}

	buffer[static_cast<std::size_t>(bytes_read)] = '\0';
	char *end = nullptr;
	errno = 0;
	const long parsed = std::strtol(buffer.data(), &end, 10);
	if (end == buffer.data() || errno != 0) {
		error = "invalid integer";
		return false;
	}
	value = static_cast<int>(parsed);
	return true;
}

bool read_sysfs_text(const std::string &path, std::string &value,
		     std::string &error)
{
	const UniqueFd fd(::open(path.c_str(), O_RDONLY | O_CLOEXEC));
	if (fd.get() < 0) {
		error = std::strerror(errno);
		return false;
	}

	std::array<char, 128> buffer {};
	ssize_t bytes_read = 0;
	do {
		bytes_read =
			::read(fd.get(), buffer.data(), buffer.size() - 1U);
	} while (bytes_read < 0 && errno == EINTR);
	if (bytes_read < 0) {
		error = std::strerror(errno);
		return false;
	}
	if (bytes_read == 0) {
		error = "empty read";
		return false;
	}

	value.assign(buffer.data(), static_cast<std::size_t>(bytes_read));
	while (!value.empty() &&
	       (value.back() == '\n' || value.back() == '\0')) {
		value.pop_back();
	}
	return true;
}

class SysfsValueRestore final {
    public:
	explicit SysfsValueRestore(std::string path)
		: _path(std::move(path))
	{
	}

	~SysfsValueRestore()
	{
		restore();
	}

	SysfsValueRestore(const SysfsValueRestore &) = delete;
	SysfsValueRestore &operator=(const SysfsValueRestore &) = delete;

	bool capture(std::string &error)
	{
		_valid = read_sysfs_text(_path, _value, error);
		return _valid;
	}

	const std::string &value() const
	{
		return _value;
	}

	void restore()
	{
		if (!_valid) {
			return;
		}
		std::string error;
		if (!write_sysfs_text(_path, _value, error)) {
			spdlog::warn(
				"IMU auto rotation failed to restore {}: {}",
				_path, error);
		}
		_valid = false;
	}

    private:
	std::string _path;
	std::string _value;
	bool _valid { false };
};

bool path_exists(const std::filesystem::path &path, std::string &error)
{
	std::error_code ec;
	const bool exists = std::filesystem::exists(path, ec);
	if (ec) {
		error = path.string() + ": " + ec.message();
		return false;
	}
	if (!exists) {
		error = path.string() + ": missing";
		return false;
	}
	return true;
}

bool has_iio_trigger()
{
	const auto has_directory_entry = [](const std::filesystem::path &path) {
		std::error_code ec;
		std::filesystem::directory_iterator it(path, ec);
		if (ec) {
			return false;
		}
		const std::filesystem::directory_iterator end;
		return it != end;
	};

	std::error_code ec;
	const std::filesystem::path devices_path { "/sys/bus/iio/devices" };
	for (const auto &entry :
	     std::filesystem::directory_iterator(devices_path, ec)) {
		if (entry.path().filename().string().rfind("trigger", 0) == 0) {
			return true;
		}
	}

	return has_directory_entry("/sys/kernel/config/iio/triggers");
}

AutoRotationRuntimeConfig
normalize_auto_rotation_config(AutoRotationRuntimeConfig config)
{
	if (config.iio_device_path.empty()) {
		config.iio_device_path =
			AutoRotationRuntimeConfig {}.iio_device_path;
	}
	config.sampling_frequency_hz =
		std::max(1, config.sampling_frequency_hz);
	config.poll_interval =
		std::max(config.poll_interval, std::chrono::milliseconds { 1 });
	config.confirm_samples = std::max(1, config.confirm_samples);
	return config;
}

bool read_imu_sample_from_sysfs(const std::string &iio_device_path,
				ImuSample &sample, std::string &error)
{
	const auto read_axis = [&iio_device_path,
				&error](std::string_view filename, int &value) {
		const std::string path = sysfs_path(iio_device_path, filename);
		std::string read_error;
		if (!read_sysfs_int(path, value, read_error)) {
			error = path + ": " + read_error;
			return false;
		}
		return true;
	};

	return read_axis(kAccelXRawFile, sample.x_raw) &&
	       read_axis(kAccelYRawFile, sample.y_raw) &&
	       read_axis(kAccelZRawFile, sample.z_raw);
}

} // namespace

ImuOrientation classify_imu_orientation(const ImuSample &sample)
{
	const int abs_x = std::abs(sample.x_raw);
	const int max_other =
		std::max(std::abs(sample.y_raw), std::abs(sample.z_raw));
	const bool x_axis_dominant =
		static_cast<int64_t>(abs_x) * kXAxisDominanceRatioDenominator >
		static_cast<int64_t>(max_other) * kXAxisDominanceRatioNumerator;
	if (sample.x_raw > kXAxisThreshold && x_axis_dominant) {
		return ImuOrientation::Forward;
	}
	if (sample.x_raw < -kXAxisThreshold && x_axis_dominant) {
		return ImuOrientation::Reverse;
	}
	return ImuOrientation::Unknown;
}

bool rotation_for_imu_orientation(ImuOrientation orientation)
{
	return orientation == ImuOrientation::Reverse;
}

std::chrono::milliseconds
remaining_auto_rotation_sleep(std::chrono::milliseconds poll_interval,
			      std::chrono::steady_clock::duration elapsed)
{
	if (poll_interval <= std::chrono::milliseconds { 0 }) {
		return std::chrono::milliseconds { 0 };
	}
	if (elapsed >= poll_interval) {
		return std::chrono::milliseconds { 0 };
	}
	return std::chrono::duration_cast<std::chrono::milliseconds>(
		poll_interval - elapsed);
}

bool probe_auto_rotation_imu(const AutoRotationRuntimeConfig &config,
			     std::string &reason)
{
	reason.clear();
	const AutoRotationRuntimeConfig normalized_config =
		normalize_auto_rotation_config(config);
	const std::filesystem::path root { normalized_config.iio_device_path };
	if (!path_exists(root, reason)) {
		return false;
	}

	const std::string frequency_path = sysfs_path(
		normalized_config.iio_device_path, kSamplingFrequencyFile);
	std::string error;
	SysfsValueRestore sampling_frequency_restore(frequency_path);
	if (!sampling_frequency_restore.capture(error)) {
		reason = "failed to read " + frequency_path + ": " + error;
		return false;
	}

	const std::string frequency_value =
		std::to_string(normalized_config.sampling_frequency_hz);
	if (!write_sysfs_text(frequency_path, frequency_value, error)) {
		reason = "failed to write " + frequency_path + ": " + error;
		return false;
	}

	ImuSample sample;
	if (!read_imu_sample_from_sysfs(normalized_config.iio_device_path,
					sample, error)) {
		reason = error;
		return false;
	}

	return true;
}

AutoRotationConfirmation::AutoRotationConfirmation(int confirm_samples,
						   bool initial_reverse)
	: _confirm_samples(std::max(1, confirm_samples))
	, _last_published_reverse(initial_reverse)
{
}

std::optional<bool> AutoRotationConfirmation::update(ImuOrientation candidate)
{
	if (candidate == ImuOrientation::Unknown) {
		reset();
		return std::nullopt;
	}

	const bool candidate_reverse = rotation_for_imu_orientation(candidate);
	if (candidate_reverse == _last_published_reverse) {
		reset();
		return std::nullopt;
	}

	if (candidate == _pending_orientation) {
		++_consecutive_samples;
	} else {
		_pending_orientation = candidate;
		_consecutive_samples = 1;
	}

	if (_consecutive_samples < _confirm_samples) {
		return std::nullopt;
	}

	_last_published_reverse = candidate_reverse;
	reset();
	return candidate_reverse;
}

void AutoRotationConfirmation::reset()
{
	_pending_orientation = ImuOrientation::Unknown;
	_consecutive_samples = 0;
}

AutoRotationRuntime::AutoRotationRuntime(AutoRotationRuntimeConfig config,
					 OrientationCallback callback)
	: _config(normalize_auto_rotation_config(std::move(config)))
	, _callback(std::move(callback))
{
}

AutoRotationRuntime::~AutoRotationRuntime()
{
	stop();
}

bool AutoRotationRuntime::start()
{
	if (_running.load(std::memory_order_acquire)) {
		return true;
	}
	if (_worker.joinable()) {
		_worker.join();
	}

	_stop_requested.store(false, std::memory_order_release);
	_running.store(true, std::memory_order_release);
	try {
		_worker = std::thread(&AutoRotationRuntime::worker_loop, this);
	} catch (const std::exception &e) {
		_running.store(false, std::memory_order_release);
		spdlog::warn("IMU auto rotation runtime start failed: {}",
			     e.what());
		return false;
	}
	return true;
}

void AutoRotationRuntime::stop()
{
	_stop_requested.store(true, std::memory_order_release);
	_cv.notify_all();
	if (_worker.joinable()) {
		_worker.join();
	}
	_running.store(false, std::memory_order_release);
}

bool AutoRotationRuntime::is_running() const
{
	return _running.load(std::memory_order_acquire);
}

void AutoRotationRuntime::worker_loop()
{
	const auto finish = [this]() {
		_running.store(false, std::memory_order_release);
	};

	if (!_callback) {
		spdlog::warn(
			"IMU auto rotation runtime disabled: missing callback");
		finish();
		return;
	}

	const std::string frequency_path =
		sysfs_path(_config.iio_device_path, kSamplingFrequencyFile);
	std::string error;
	SysfsValueRestore sampling_frequency_restore(frequency_path);
	if (!sampling_frequency_restore.capture(error)) {
		spdlog::warn(
			"IMU auto rotation failed to read {} before changing it: {}",
			frequency_path, error);
		error.clear();
	}
	const std::string frequency_value =
		std::to_string(_config.sampling_frequency_hz);
	if (!write_sysfs_text(frequency_path, frequency_value, error)) {
		spdlog::warn(
			"IMU auto rotation disabled: failed to write {}: {}",
			frequency_path, error);
		finish();
		return;
	}
	spdlog::info(
		"[auto-rotation] IMU auto rotation sampling_frequency set to {}",
		_config.sampling_frequency_hz);

	std::string buffer_probe_reason;
	if (!probe_iio_buffer_capability(buffer_probe_reason)) {
		spdlog::warn(
			"IMU auto rotation IIO buffer mode unavailable: {}; using raw sysfs fallback",
			buffer_probe_reason);
	} else {
		spdlog::info(
			"[auto-rotation] IMU auto rotation IIO buffer probe succeeded; using raw sysfs fallback primary path");
	}

	AutoRotationConfirmation confirmation(_config.confirm_samples, false);
	bool read_warning_logged = false;

	while (!_stop_requested.load(std::memory_order_acquire)) {
		const auto loop_start = std::chrono::steady_clock::now();

		ImuSample sample;
		std::string read_error;
		if (!read_imu_sample(sample, read_error)) {
			if (!read_warning_logged) {
				spdlog::warn(
					"IMU auto rotation sample unavailable: {}",
					read_error);
				read_warning_logged = true;
			}
			confirmation.reset();
		} else {
			read_warning_logged = false;
			const ImuOrientation candidate =
				classify_imu_orientation(sample);
			const std::optional<bool> reverse =
				confirmation.update(candidate);
			if (reverse.has_value()) {
				_callback(*reverse);
			}
		}

		const auto elapsed =
			std::chrono::steady_clock::now() - loop_start;
		sleep_interruptibly(remaining_auto_rotation_sleep(
			_config.poll_interval, elapsed));
	}

	sampling_frequency_restore.restore();
	finish();
}

void AutoRotationRuntime::sleep_interruptibly(std::chrono::milliseconds duration)
{
	if (duration <= std::chrono::milliseconds { 0 }) {
		return;
	}

	std::unique_lock<std::mutex> lock(_mutex);
	_cv.wait_for(lock, duration, [this]() {
		return _stop_requested.load(std::memory_order_acquire);
	});
}

bool AutoRotationRuntime::read_imu_sample(ImuSample &sample,
					  std::string &error) const
{
	return read_imu_sample_from_sysfs(_config.iio_device_path, sample,
					  error);
}

bool AutoRotationRuntime::probe_iio_buffer_capability(std::string &reason) const
{
	const std::filesystem::path root { _config.iio_device_path };
	const std::filesystem::path scan_elements = root / "scan_elements";
	const std::filesystem::path buffer_enable = root / "buffer" / "enable";
	const std::filesystem::path buffer_length = root / "buffer" / "length";

	if (!path_exists(scan_elements, reason)) {
		return false;
	}
	if (!path_exists(buffer_enable, reason)) {
		return false;
	}
	if (!path_exists(buffer_length, reason)) {
		return false;
	}

	for (std::string_view channel :
	     { "in_accel_x", "in_accel_y", "in_accel_z" }) {
		if (!path_exists(scan_elements / (std::string(channel) + "_en"),
				 reason)) {
			return false;
		}
		if (!path_exists(scan_elements /
					 (std::string(channel) + "_index"),
				 reason)) {
			return false;
		}
		if (!path_exists(scan_elements /
					 (std::string(channel) + "_type"),
				 reason)) {
			return false;
		}
	}

	if (!has_iio_trigger()) {
		reason = "no IIO trigger devices found";
		return false;
	}

	std::string error;
	SysfsValueRestore buffer_enable_restore(buffer_enable.string());
	if (!buffer_enable_restore.capture(error)) {
		reason = "failed to read " + buffer_enable.string() + ": " +
			 error;
		return false;
	}
	if (buffer_enable_restore.value() != "0") {
		reason = buffer_enable.string() + " already enabled";
		return false;
	}

	SysfsValueRestore buffer_length_restore(buffer_length.string());
	if (!buffer_length_restore.capture(error)) {
		reason = "failed to read " + buffer_length.string() + ": " +
			 error;
		return false;
	}

	std::array<SysfsValueRestore, 3> channel_restores {
		SysfsValueRestore((scan_elements / "in_accel_x_en").string()),
		SysfsValueRestore((scan_elements / "in_accel_y_en").string()),
		SysfsValueRestore((scan_elements / "in_accel_z_en").string()),
	};
	for (auto &restore : channel_restores) {
		if (!restore.capture(error)) {
			reason = "failed to read scan element enable state: " +
				 error;
			return false;
		}
	}

	for (std::string_view channel :
	     { "in_accel_x", "in_accel_y", "in_accel_z" }) {
		const std::string en_path =
			(scan_elements / (std::string(channel) + "_en"))
				.string();
		if (!write_sysfs_text(en_path, "1", error)) {
			reason = "failed to enable " + en_path + ": " + error;
			return false;
		}
	}

	if (!write_sysfs_text(buffer_length.string(), "2", error)) {
		reason = "failed to set " + buffer_length.string() + ": " +
			 error;
		return false;
	}
	if (!write_sysfs_text(buffer_enable.string(), "1", error)) {
		reason = "failed to enable " + buffer_enable.string() + ": " +
			 error;
		return false;
	}

	buffer_enable_restore.restore();
	return true;
}

} // namespace lvgl_main
