#include <atomic>
#include <filesystem>
#include <stdexcept>
#include <string>

#include "timelapse_commit.hpp"

#include <catch2/catch_test_macros.hpp>
#include <sys/wait.h>
#include <unistd.h>

namespace {

class TemporaryDirectory {
  public:
    TemporaryDirectory()
    {
        static std::atomic<unsigned long> sequence {0};
        const auto suffix = std::to_string(::getpid()) + "-"
            + std::to_string(sequence.fetch_add(1));
        _path = std::filesystem::temp_directory_path()
            / ("kvm-vin-timelapse-commit-" + suffix);
        std::error_code ec;
        if (!std::filesystem::create_directories(_path, ec) || ec) {
            throw std::runtime_error(
                "failed to create temporary test directory");
        }
    }

    ~TemporaryDirectory()
    {
        std::error_code ec;
        (void)std::filesystem::remove_all(_path, ec);
    }

    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;

    const std::filesystem::path& path() const { return _path; }

  private:
    std::filesystem::path _path;
};

}  // namespace

TEST_CASE("Timelapse commit date guard rejects stale and future frames",
          "[timelapse][commit]")
{
    CHECK(timelapse_can_commit_date("2026-07-20", "2026-07-20"));
    CHECK_FALSE(timelapse_can_commit_date("2026-07-19", "2026-07-20"));
    CHECK_FALSE(timelapse_can_commit_date("2026-07-21", "2026-07-20"));
    CHECK_FALSE(timelapse_can_commit_date("", "2026-07-20"));
    CHECK_FALSE(timelapse_can_commit_date("2026-07-20", ""));
}

TEST_CASE("Timelapse commit lock is exclusive and releases with RAII",
          "[timelapse][commit]")
{
    TemporaryDirectory directory;
    const auto lock_path = directory.path() / "run" / "timelapse.lock";
    std::string error;

    TimelapseCommitLock second(lock_path);
    {
        TimelapseCommitLock first(lock_path);
        REQUIRE(first.try_lock(error) == TimelapseCommitLockResult::Acquired);
        REQUIRE(first.owns_lock());

        CHECK(second.try_lock(error) == TimelapseCommitLockResult::Busy);
        CHECK_FALSE(second.owns_lock());
    }

    REQUIRE(second.try_lock(error) == TimelapseCommitLockResult::Acquired);
    CHECK(second.owns_lock());
    second.unlock();
    CHECK_FALSE(second.owns_lock());
}

TEST_CASE("Timelapse commit lock excludes another process",
          "[timelapse][commit]")
{
    TemporaryDirectory directory;
    const auto lock_path = directory.path() / "run" / "timelapse.lock";
    std::string error;
    TimelapseCommitLock first(lock_path);
    REQUIRE(first.try_lock(error) == TimelapseCommitLockResult::Acquired);

    int result_pipe[2] {};
    REQUIRE(::pipe(result_pipe) == 0);
    const pid_t child = ::fork();
    REQUIRE(child >= 0);
    if (child == 0) {
        (void)::close(result_pipe[0]);
        TimelapseCommitLock second(lock_path);
        const auto result = second.try_lock(error);
        const unsigned char value =
            result == TimelapseCommitLockResult::Busy ? 1 : 0;
        (void)::write(result_pipe[1], &value, sizeof(value));
        (void)::close(result_pipe[1]);
        _exit(value == 1 ? 0 : 1);
    }

    (void)::close(result_pipe[1]);
    unsigned char value = 0;
    REQUIRE(::read(result_pipe[0], &value, sizeof(value))
            == static_cast<ssize_t>(sizeof(value)));
    (void)::close(result_pipe[0]);

    int status = 0;
    REQUIRE(::waitpid(child, &status, 0) == child);
    REQUIRE(WIFEXITED(status));
    CHECK(WEXITSTATUS(status) == 0);
    CHECK(value == 1);
}
