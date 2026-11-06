#pragma once

#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <optional>
#include <string>
#include <string_view>

#include <sys/types.h>

namespace lvgl_main
{

enum class ScreenshotSignal : uint8_t {
	User1,
	User2,
};

namespace screenshot_control
{

inline constexpr std::string_view kPidfilePath = "/run/kvm/kvm_ui.pid";
inline constexpr std::size_t kMaxPidfileRecordBytes = 64U;

std::string_view signal_name(ScreenshotSignal signal) noexcept;
std::optional<ScreenshotSignal> parse_signal(std::string_view name) noexcept;
int signal_number(ScreenshotSignal signal) noexcept;

struct InstanceRecord {
	pid_t pid { -1 };
	ScreenshotSignal signal { ScreenshotSignal::User1 };
};

std::string format_instance_record(pid_t pid, ScreenshotSignal signal);
std::optional<InstanceRecord>
parse_instance_record(std::string_view record) noexcept;

enum class ScreenshotControlError : uint8_t {
	None,
	InvalidArgument,
	PidfileParentCreateFailed,
	PidfileOpenFailed,
	PidfileBusy,
	PidfileRace,
	PidfileWriteFailed,
	PidfileSyncFailed,
	PidfileReadFailed,
	MalformedPidfile,
	ProcessUnavailable,
	ExecutableLookupFailed,
	ExecutableMismatch,
	SignalFailed,
};

struct ScreenshotControlResult {
	bool ok { false };
	ScreenshotControlError error { ScreenshotControlError::None };
	int system_error { 0 };
	std::string message;
};

class ScreenshotControlOps {
    public:
	virtual ~ScreenshotControlOps() = default;

	virtual pid_t current_process_id() noexcept = 0;
	virtual bool ensure_parent_directory(const std::filesystem::path &path,
					     int &error) noexcept = 0;
	virtual int open_file(const std::filesystem::path &path, int flags,
			      mode_t mode, int &error) noexcept = 0;
	virtual bool close_file(int fd, int &error) noexcept = 0;
	virtual bool is_regular_file(int fd, int &error) noexcept = 0;
	virtual bool lock_exclusive_nonblocking(int fd, int &error) noexcept = 0;
	virtual bool path_matches_fd(const std::filesystem::path &path, int fd,
			     int &error) noexcept = 0;
	virtual bool truncate_file(int fd, int &error) noexcept = 0;
	virtual ssize_t read_file(int fd, char *buffer, std::size_t length,
			  std::size_t offset, int &error) noexcept = 0;
	virtual ssize_t write_file(int fd, const char *buffer,
			   std::size_t length, int &error) noexcept = 0;
	virtual bool sync_file(int fd, int &error) noexcept = 0;
	virtual bool unlink_file(const std::filesystem::path &path,
			 int &error) noexcept = 0;
	virtual bool signal_process(pid_t pid, int signal, int &error) noexcept = 0;
	virtual std::optional<std::string> executable_path(pid_t pid,
					      int &error) = 0;
};

class SystemScreenshotControlOps final : public ScreenshotControlOps {
    public:
	pid_t current_process_id() noexcept override;
	bool ensure_parent_directory(const std::filesystem::path &path,
				     int &error) noexcept override;
	int open_file(const std::filesystem::path &path, int flags, mode_t mode,
		      int &error) noexcept override;
	bool close_file(int fd, int &error) noexcept override;
	bool is_regular_file(int fd, int &error) noexcept override;
	bool lock_exclusive_nonblocking(int fd, int &error) noexcept override;
	bool path_matches_fd(const std::filesystem::path &path, int fd,
			     int &error) noexcept override;
	bool truncate_file(int fd, int &error) noexcept override;
	ssize_t read_file(int fd, char *buffer, std::size_t length,
			  std::size_t offset, int &error) noexcept override;
	ssize_t write_file(int fd, const char *buffer, std::size_t length,
			   int &error) noexcept override;
	bool sync_file(int fd, int &error) noexcept override;
	bool unlink_file(const std::filesystem::path &path,
			 int &error) noexcept override;
	bool signal_process(pid_t pid, int signal, int &error) noexcept override;
	std::optional<std::string> executable_path(pid_t pid,
					      int &error) override;
};

SystemScreenshotControlOps &system_ops() noexcept;

class ScreenshotInstanceLease;
struct InstanceLeaseResult;

InstanceLeaseResult acquire_instance_lease(const std::filesystem::path &path,
					   pid_t pid, ScreenshotSignal signal,
					   ScreenshotControlOps &ops);

class ScreenshotInstanceLease final {
    public:
	~ScreenshotInstanceLease();

	ScreenshotInstanceLease(const ScreenshotInstanceLease &) = delete;
	ScreenshotInstanceLease &operator=(const ScreenshotInstanceLease &) =
		delete;
	ScreenshotInstanceLease(ScreenshotInstanceLease &&other);
	ScreenshotInstanceLease &operator=(ScreenshotInstanceLease &&other);

	void release() noexcept;
	[[nodiscard]] bool active() const noexcept;

    private:
	ScreenshotInstanceLease(ScreenshotControlOps &ops,
				std::filesystem::path path, int fd,
				InstanceRecord record, std::string serialized_record);

	ScreenshotControlOps *_ops { nullptr };
	std::filesystem::path _path;
	int _fd { -1 };
	InstanceRecord _record;
	std::string _serialized_record;

	friend InstanceLeaseResult acquire_instance_lease(
		const std::filesystem::path &path, pid_t pid,
		ScreenshotSignal signal, ScreenshotControlOps &ops);
};

struct InstanceLeaseResult {
	std::optional<ScreenshotInstanceLease> lease;
	ScreenshotControlResult result;
};

std::optional<std::string>
current_executable_path(ScreenshotControlOps &ops,
			ScreenshotControlResult &result);

ScreenshotControlResult send_screenshot_request(
	const std::filesystem::path &pidfile_path, std::string_view self_executable,
	ScreenshotControlOps &ops);

} // namespace screenshot_control

} // namespace lvgl_main
