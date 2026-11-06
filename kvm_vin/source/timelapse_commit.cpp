#include <cerrno>
#include <cstring>
#include <ctime>
#include <system_error>
#include <utility>

#include "timelapse_commit.hpp"

#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

TimelapseCommitLock::TimelapseCommitLock(std::filesystem::path path)
    : _path(std::move(path))
{}

TimelapseCommitLock::~TimelapseCommitLock() { unlock(); }

TimelapseCommitLockResult TimelapseCommitLock::try_lock(std::string& error)
{
    error.clear();
    if (owns_lock()) {
        return TimelapseCommitLockResult::Acquired;
    }

    const auto parent = _path.parent_path();
    if (!parent.empty()) {
        std::error_code ec;
        (void)std::filesystem::create_directories(parent, ec);
        if (ec) {
            error = "create lock directory " + parent.string() + ": "
                + ec.message();
            return TimelapseCommitLockResult::Error;
        }
    }

    const int fd =
        ::open(_path.c_str(), O_CREAT | O_RDWR | O_CLOEXEC, S_IRUSR | S_IWUSR);
    if (fd < 0) {
        error = "open lock " + _path.string() + ": " + std::strerror(errno);
        return TimelapseCommitLockResult::Error;
    }

    if (::flock(fd, LOCK_EX | LOCK_NB) != 0) {
        const int saved_errno = errno;
        (void)::close(fd);
        if (saved_errno == EWOULDBLOCK || saved_errno == EAGAIN) {
            error = "timelapse commit lock is busy";
            return TimelapseCommitLockResult::Busy;
        }
        error = "lock " + _path.string() + ": " + std::strerror(saved_errno);
        return TimelapseCommitLockResult::Error;
    }

    _fd = fd;
    return TimelapseCommitLockResult::Acquired;
}

bool TimelapseCommitLock::owns_lock() const noexcept { return _fd >= 0; }

void TimelapseCommitLock::unlock() noexcept
{
    if (_fd < 0) {
        return;
    }

    (void)::flock(_fd, LOCK_UN);
    (void)::close(_fd);
    _fd = -1;
}

std::string timelapse_current_local_date()
{
    const std::time_t now = std::time(nullptr);
    if (now == static_cast<std::time_t>(-1)) {
        return {};
    }

    std::tm local {};
    if (::localtime_r(&now, &local) == nullptr) {
        return {};
    }

    char date[16] {};
    if (std::strftime(date, sizeof(date), "%Y-%m-%d", &local) == 0) {
        return {};
    }
    return date;
}

bool timelapse_can_commit_date(std::string_view target_date,
                               std::string_view current_date)
{ return !target_date.empty() && target_date == current_date; }
