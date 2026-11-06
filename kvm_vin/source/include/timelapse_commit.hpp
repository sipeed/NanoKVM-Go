#pragma once

#include <filesystem>
#include <string>
#include <string_view>

enum class TimelapseCommitLockResult {
    Acquired,
    Busy,
    Error,
};

// Coordinates the short on-disk Timelapse commit with the server-side cleanup
// path. JPEG decoding and H.264 encoding intentionally happen before this
// lock is acquired.
class TimelapseCommitLock {
  public:
    static constexpr const char* DEFAULT_PATH = "/run/kvm/timelapse.lock";

    explicit TimelapseCommitLock(
        std::filesystem::path path = std::filesystem::path(DEFAULT_PATH));
    ~TimelapseCommitLock();

    TimelapseCommitLock(const TimelapseCommitLock&) = delete;
    TimelapseCommitLock& operator=(const TimelapseCommitLock&) = delete;

    TimelapseCommitLockResult try_lock(std::string& error);
    bool owns_lock() const noexcept;
    void unlock() noexcept;

  private:
    std::filesystem::path _path;
    int _fd {-1};
};

std::string timelapse_current_local_date();
bool timelapse_can_commit_date(std::string_view target_date,
                               std::string_view current_date);
