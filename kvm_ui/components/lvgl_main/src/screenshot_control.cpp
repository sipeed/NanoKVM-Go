#include "lvgl_main/screenshot_control.hpp"

#include <array>
#include <charconv>
#include <cerrno>
#include <cstring>
#include <limits>
#include <utility>

#include <fcntl.h>
#include <signal.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

namespace lvgl_main::screenshot_control
{

namespace
{

constexpr mode_t kPidfileMode = S_IRUSR | S_IWUSR;
constexpr std::size_t kExecutablePathBufferBytes = 4096U;

ScreenshotControlResult success_result()
{
	return { .ok = true };
}

ScreenshotControlResult failure_result(ScreenshotControlError error,
					int system_error, std::string message)
{
	return {
		.ok = false,
		.error = error,
		.system_error = system_error,
		.message = std::move(message),
	};
}

std::string system_error_message(std::string_view action, int error)
{
	std::string message(action);
	if (error != 0) {
		message += ": ";
		message += std::strerror(error);
	}
	return message;
}

bool is_valid_pid(pid_t pid) noexcept
{
	return pid > 0;
}

bool write_all(ScreenshotControlOps &ops, int fd, std::string_view data,
		       int &error)
{
	std::size_t written = 0;
	while (written < data.size()) {
		const ssize_t result = ops.write_file(
			fd, data.data() + written, data.size() - written, error);
		if (result < 0) {
			return false;
		}
		if (result == 0) {
			error = EIO;
			return false;
		}
		written += static_cast<std::size_t>(result);
	}
	return true;
}

bool read_bounded_record(ScreenshotControlOps &ops, int fd, std::string &record,
			 int &error)
{
	std::array<char, kMaxPidfileRecordBytes + 1U> buffer {};
	std::size_t read_total = 0;
	while (read_total < buffer.size()) {
		const ssize_t result = ops.read_file(
			fd, buffer.data() + read_total, buffer.size() - read_total,
			read_total, error);
		if (result < 0) {
			return false;
		}
		if (result == 0) {
			break;
		}
		read_total += static_cast<std::size_t>(result);
	}
	if (read_total > kMaxPidfileRecordBytes) {
		error = EOVERFLOW;
		return false;
	}
	record.assign(buffer.data(), read_total);
	return true;
}

bool close_checked(ScreenshotControlOps &ops, int fd, int &error)
{
	return ops.close_file(fd, error);
}

std::optional<InstanceRecord>
read_instance_record(const std::filesystem::path &path,
			     ScreenshotControlOps &ops,
			     ScreenshotControlResult &result)
{
	int error = 0;
	const int fd = ops.open_file(path,
				     O_RDONLY | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK, 0,
				     error);
	if (fd < 0) {
		result = failure_result(
			ScreenshotControlError::PidfileOpenFailed, error,
			system_error_message("failed to open screenshot pidfile", error));
		return std::nullopt;
	}

	auto close_with_failure = [&](ScreenshotControlError failure,
				      int failure_error,
				      std::string message) {
		int close_error = 0;
		(void)close_checked(ops, fd, close_error);
		result = failure_result(failure, failure_error, std::move(message));
	};

	if (!ops.is_regular_file(fd, error)) {
		close_with_failure(
			ScreenshotControlError::PidfileReadFailed, error,
			system_error_message("screenshot pidfile is not a regular file", error));
		return std::nullopt;
	}
	if (!ops.path_matches_fd(path, fd, error)) {
		close_with_failure(
			ScreenshotControlError::PidfileReadFailed, error,
			system_error_message("screenshot pidfile changed while reading", error));
		return std::nullopt;
	}

	std::string serialized_record;
	if (!read_bounded_record(ops, fd, serialized_record, error)) {
		const ScreenshotControlError failure = error == EOVERFLOW ?
			ScreenshotControlError::MalformedPidfile :
			ScreenshotControlError::PidfileReadFailed;
		close_with_failure(
			failure, error,
			system_error_message("failed to read screenshot pidfile", error));
		return std::nullopt;
	}

	int close_error = 0;
	if (!close_checked(ops, fd, close_error)) {
		result = failure_result(
			ScreenshotControlError::PidfileReadFailed, close_error,
			system_error_message("failed to close screenshot pidfile", close_error));
		return std::nullopt;
	}

	const auto record = parse_instance_record(serialized_record);
	if (!record.has_value()) {
		result = failure_result(ScreenshotControlError::MalformedPidfile, 0,
					"screenshot pidfile record is malformed");
	}
	return record;
}

} // namespace

std::string_view signal_name(ScreenshotSignal signal) noexcept
{
	switch (signal) {
	case ScreenshotSignal::User1:
		return "SIGUSR1";
	case ScreenshotSignal::User2:
		return "SIGUSR2";
	}
	return {};
}

std::optional<ScreenshotSignal> parse_signal(std::string_view name) noexcept
{
	if (name == "SIGUSR1") {
		return ScreenshotSignal::User1;
	}
	if (name == "SIGUSR2") {
		return ScreenshotSignal::User2;
	}
	return std::nullopt;
}

int signal_number(ScreenshotSignal signal) noexcept
{
	switch (signal) {
	case ScreenshotSignal::User1:
		return SIGUSR1;
	case ScreenshotSignal::User2:
		return SIGUSR2;
	}
	return -1;
}

std::string format_instance_record(pid_t pid, ScreenshotSignal signal)
{
	if (!is_valid_pid(pid) || signal_name(signal).empty()) {
		return {};
	}
	return std::to_string(pid) + " " + std::string(signal_name(signal)) +
	       "\n";
}

std::optional<InstanceRecord>
parse_instance_record(std::string_view record) noexcept
{
	if (record.empty() || record.size() > kMaxPidfileRecordBytes ||
	    record.back() != '\n') {
		return std::nullopt;
	}

	const std::string_view body = record.substr(0, record.size() - 1U);
	const std::size_t separator = body.find(' ');
	if (separator == std::string_view::npos || separator == 0U ||
	    separator + 1U >= body.size() ||
	    body.find(' ', separator + 1U) != std::string_view::npos) {
		return std::nullopt;
	}

	const std::string_view pid_text = body.substr(0, separator);
	if (pid_text.front() == '0') {
		return std::nullopt;
	}
	for (const char ch : pid_text) {
		if (ch < '0' || ch > '9') {
			return std::nullopt;
		}
	}

	long long parsed_pid = 0;
	const auto [parse_end, parse_error] = std::from_chars(
		pid_text.data(), pid_text.data() + pid_text.size(), parsed_pid);
	if (parse_error != std::errc {} ||
	    parse_end != pid_text.data() + pid_text.size() || parsed_pid <= 0 ||
	    parsed_pid > static_cast<long long>(std::numeric_limits<pid_t>::max())) {
		return std::nullopt;
	}

	const auto signal = parse_signal(body.substr(separator + 1U));
	if (!signal.has_value()) {
		return std::nullopt;
	}
	return InstanceRecord {
		.pid = static_cast<pid_t>(parsed_pid),
		.signal = *signal,
	};
}

pid_t SystemScreenshotControlOps::current_process_id() noexcept
{
	return ::getpid();
}

bool SystemScreenshotControlOps::ensure_parent_directory(
	const std::filesystem::path &path, int &error) noexcept
{
	const auto parent = path.parent_path();
	if (parent.empty()) {
		error = 0;
		return true;
	}

	std::error_code filesystem_error;
	std::filesystem::create_directories(parent, filesystem_error);
	if (filesystem_error) {
		error = filesystem_error.value();
		return false;
	}
	if (!std::filesystem::is_directory(parent, filesystem_error) ||
	    filesystem_error) {
		error = filesystem_error ? filesystem_error.value() : ENOTDIR;
		return false;
	}

	error = 0;
	return true;
}

int SystemScreenshotControlOps::open_file(const std::filesystem::path &path,
					  int flags, mode_t mode, int &error) noexcept
{
	const int fd = ::open(path.c_str(), flags, mode);
	error = fd < 0 ? errno : 0;
	return fd;
}

bool SystemScreenshotControlOps::close_file(int fd, int &error) noexcept
{
	if (::close(fd) == 0) {
		error = 0;
		return true;
	}
	error = errno;
	return false;
}

bool SystemScreenshotControlOps::is_regular_file(int fd, int &error) noexcept
{
	struct stat status {};
	if (::fstat(fd, &status) != 0) {
		error = errno;
		return false;
	}
	if (!S_ISREG(status.st_mode)) {
		error = EINVAL;
		return false;
	}
	error = 0;
	return true;
}

bool SystemScreenshotControlOps::lock_exclusive_nonblocking(int fd,
						     int &error) noexcept
{
	for (;;) {
		if (::flock(fd, LOCK_EX | LOCK_NB) == 0) {
			error = 0;
			return true;
		}
		if (errno != EINTR) {
			error = errno;
			return false;
		}
	}
}

bool SystemScreenshotControlOps::path_matches_fd(const std::filesystem::path &path,
						  int fd, int &error) noexcept
{
	struct stat fd_status {};
	if (::fstat(fd, &fd_status) != 0) {
		error = errno;
		return false;
	}
	struct stat path_status {};
	if (::lstat(path.c_str(), &path_status) != 0) {
		error = errno;
		return false;
	}
	if (fd_status.st_dev != path_status.st_dev ||
	    fd_status.st_ino != path_status.st_ino) {
		error = ESTALE;
		return false;
	}
	error = 0;
	return true;
}

bool SystemScreenshotControlOps::truncate_file(int fd, int &error) noexcept
{
	if (::ftruncate(fd, 0) == 0) {
		error = 0;
		return true;
	}
	error = errno;
	return false;
}

ssize_t SystemScreenshotControlOps::read_file(int fd, char *buffer,
					       std::size_t length,
					       std::size_t offset, int &error) noexcept
{
	for (;;) {
		const ssize_t result = ::pread(fd, buffer, length,
					      static_cast<off_t>(offset));
		if (result >= 0 || errno != EINTR) {
			error = result < 0 ? errno : 0;
			return result;
		}
	}
}

ssize_t SystemScreenshotControlOps::write_file(int fd, const char *buffer,
						std::size_t length,
						int &error) noexcept
{
	for (;;) {
		const ssize_t result = ::write(fd, buffer, length);
		if (result >= 0 || errno != EINTR) {
			error = result < 0 ? errno : 0;
			return result;
		}
	}
}

bool SystemScreenshotControlOps::sync_file(int fd, int &error) noexcept
{
	if (::fsync(fd) == 0) {
		error = 0;
		return true;
	}
	error = errno;
	return false;
}

bool SystemScreenshotControlOps::unlink_file(const std::filesystem::path &path,
						      int &error) noexcept
{
	if (::unlink(path.c_str()) == 0) {
		error = 0;
		return true;
	}
	error = errno;
	return false;
}

bool SystemScreenshotControlOps::signal_process(pid_t pid, int signal,
						 int &error) noexcept
{
	if (::kill(pid, signal) == 0) {
		error = 0;
		return true;
	}
	error = errno;
	return false;
}

std::optional<std::string>
SystemScreenshotControlOps::executable_path(pid_t pid, int &error)
{
	if (!is_valid_pid(pid)) {
		error = EINVAL;
		return std::nullopt;
	}

	std::array<char, kExecutablePathBufferBytes> buffer {};
	const std::string proc_path = "/proc/" + std::to_string(pid) + "/exe";
	const ssize_t length = ::readlink(proc_path.c_str(), buffer.data(),
					     buffer.size());
	if (length < 0) {
		error = errno;
		return std::nullopt;
	}
	if (static_cast<std::size_t>(length) == buffer.size()) {
		error = ENAMETOOLONG;
		return std::nullopt;
	}
	error = 0;
	return std::string(buffer.data(), static_cast<std::size_t>(length));
}

SystemScreenshotControlOps &system_ops() noexcept
{
	static SystemScreenshotControlOps ops;
	return ops;
}

ScreenshotInstanceLease::ScreenshotInstanceLease(ScreenshotControlOps &ops,
						 std::filesystem::path path, int fd,
						 InstanceRecord record,
						 std::string serialized_record)
	: _ops(&ops)
	, _path(std::move(path))
	, _fd(fd)
	, _record(record)
	, _serialized_record(std::move(serialized_record))
{
}

ScreenshotInstanceLease::~ScreenshotInstanceLease()
{
	release();
}

ScreenshotInstanceLease::ScreenshotInstanceLease(ScreenshotInstanceLease &&other)
	: _ops(other._ops)
	, _path(std::move(other._path))
	, _fd(other._fd)
	, _record(other._record)
	, _serialized_record(std::move(other._serialized_record))
{
	other._ops = nullptr;
	other._fd = -1;
}

ScreenshotInstanceLease &ScreenshotInstanceLease::operator=(
	ScreenshotInstanceLease &&other)
{
	if (this == &other) {
		return *this;
	}
	release();
	_ops = other._ops;
	_path = std::move(other._path);
	_fd = other._fd;
	_record = other._record;
	_serialized_record = std::move(other._serialized_record);
	other._ops = nullptr;
	other._fd = -1;
	return *this;
}

void ScreenshotInstanceLease::release() noexcept
{
	if (_fd < 0 || _ops == nullptr) {
		_fd = -1;
		_ops = nullptr;
		return;
	}

	int error = 0;
	if (_ops->path_matches_fd(_path, _fd, error)) {
		std::array<char, kMaxPidfileRecordBytes + 1U> buffer {};
		std::size_t read_total = 0;
		bool read_ok = true;
		while (read_total < buffer.size()) {
			const ssize_t result = _ops->read_file(
				_fd, buffer.data() + read_total,
				buffer.size() - read_total, read_total, error);
			if (result < 0) {
				read_ok = false;
				break;
			}
			if (result == 0) {
				break;
			}
			read_total += static_cast<std::size_t>(result);
		}
		if (read_ok && read_total == _serialized_record.size() &&
		    std::memcmp(buffer.data(), _serialized_record.data(), read_total) ==
			    0) {
			(void)_ops->unlink_file(_path, error);
		}
	}
	(void)_ops->close_file(_fd, error);
	_fd = -1;
	_ops = nullptr;
	_path.clear();
	_serialized_record.clear();
}

bool ScreenshotInstanceLease::active() const noexcept
{
	return _fd >= 0;
}

InstanceLeaseResult acquire_instance_lease(const std::filesystem::path &path,
					   pid_t pid, ScreenshotSignal signal,
					   ScreenshotControlOps &ops)
{
	if (path.empty() || !is_valid_pid(pid) || signal_number(signal) < 0) {
		return {
			.result = failure_result(
				ScreenshotControlError::InvalidArgument, EINVAL,
				"invalid screenshot instance lease arguments"),
		};
	}

	int error = 0;
	if (!ops.ensure_parent_directory(path, error)) {
		return {
			.result = failure_result(
				ScreenshotControlError::PidfileParentCreateFailed, error,
				system_error_message(
					"failed to create screenshot pidfile directory", error)),
		};
	}
	const int fd = ops.open_file(path,
				     O_RDWR | O_CREAT | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK,
				     kPidfileMode, error);
	if (fd < 0) {
		return {
			.result = failure_result(
				ScreenshotControlError::PidfileOpenFailed, error,
				system_error_message("failed to open screenshot pidfile", error)),
		};
	}

	auto close_and_return = [&](ScreenshotControlError failure,
				    int failure_error, std::string message) {
		int close_error = 0;
		(void)close_checked(ops, fd, close_error);
		return InstanceLeaseResult {
			.result = failure_result(failure, failure_error, std::move(message)),
		};
	};

	if (!ops.is_regular_file(fd, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileOpenFailed, error,
			system_error_message("screenshot pidfile is not a regular file", error));
	}
	if (!ops.lock_exclusive_nonblocking(fd, error)) {
		const ScreenshotControlError failure =
			(error == EWOULDBLOCK || error == EAGAIN) ?
				ScreenshotControlError::PidfileBusy :
				ScreenshotControlError::PidfileOpenFailed;
		return close_and_return(
			failure, error,
			system_error_message("failed to lock screenshot pidfile", error));
	}
	if (!ops.path_matches_fd(path, fd, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileRace, error,
			system_error_message("screenshot pidfile changed while acquiring lease",
					     error));
	}
	if (!ops.truncate_file(fd, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileWriteFailed, error,
			system_error_message("failed to prepare screenshot pidfile", error));
	}

	const InstanceRecord record { .pid = pid, .signal = signal };
	const std::string serialized_record = format_instance_record(pid, signal);
	if (serialized_record.empty()) {
		return close_and_return(
			ScreenshotControlError::InvalidArgument, EINVAL,
			"failed to format screenshot pidfile record");
	}
	if (!write_all(ops, fd, serialized_record, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileWriteFailed, error,
			system_error_message("failed to write screenshot pidfile", error));
	}
	if (!ops.sync_file(fd, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileSyncFailed, error,
			system_error_message("failed to sync screenshot pidfile", error));
	}
	if (!ops.path_matches_fd(path, fd, error)) {
		return close_and_return(
			ScreenshotControlError::PidfileRace, error,
			system_error_message("screenshot pidfile changed while publishing lease",
					     error));
	}

	return {
		.lease = ScreenshotInstanceLease(ops, path, fd, record,
						 serialized_record),
		.result = success_result(),
	};
}

std::optional<std::string>
current_executable_path(ScreenshotControlOps &ops,
				ScreenshotControlResult &result)
{
	const pid_t pid = ops.current_process_id();
	if (!is_valid_pid(pid)) {
		result = failure_result(ScreenshotControlError::InvalidArgument, EINVAL,
					"current process ID is invalid");
		return std::nullopt;
	}

	int error = 0;
	auto path = ops.executable_path(pid, error);
	if (!path.has_value() || path->empty()) {
		result = failure_result(
			ScreenshotControlError::ExecutableLookupFailed, error,
			system_error_message("failed to resolve current executable", error));
		return std::nullopt;
	}
	result = success_result();
	return path;
}

ScreenshotControlResult send_screenshot_request(
	const std::filesystem::path &pidfile_path, std::string_view self_executable,
	ScreenshotControlOps &ops)
{
	if (pidfile_path.empty() || self_executable.empty()) {
		return failure_result(ScreenshotControlError::InvalidArgument, EINVAL,
				      "invalid screenshot request arguments");
	}

	ScreenshotControlResult read_result;
	const auto record = read_instance_record(pidfile_path, ops, read_result);
	if (!record.has_value()) {
		return read_result;
	}
	int error = 0;
	if (!ops.signal_process(record->pid, 0, error)) {
		return failure_result(
			ScreenshotControlError::ProcessUnavailable, error,
			system_error_message("screenshot target process is unavailable", error));
	}

	auto target_executable = ops.executable_path(record->pid, error);
	if (!target_executable.has_value() || target_executable->empty()) {
		return failure_result(
			ScreenshotControlError::ExecutableLookupFailed, error,
			system_error_message("failed to resolve screenshot target executable",
					     error));
	}
	if (*target_executable != self_executable) {
		return failure_result(
			ScreenshotControlError::ExecutableMismatch, 0,
			"screenshot pidfile executable does not match this binary");
	}

	const int signal = signal_number(record->signal);
	if (signal < 0) {
		return failure_result(ScreenshotControlError::MalformedPidfile, EINVAL,
				      "screenshot pidfile signal is unsupported");
	}
	if (!ops.signal_process(record->pid, signal, error)) {
		return failure_result(
			ScreenshotControlError::SignalFailed, error,
			system_error_message("failed to send screenshot signal", error));
	}
	return success_result();
}

} // namespace lvgl_main::screenshot_control
