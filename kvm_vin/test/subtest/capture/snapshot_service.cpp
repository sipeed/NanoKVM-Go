#include <atomic>
#include <chrono>
#include <cstdint>
#include <cstdlib>
#include <filesystem>
#include <memory>
#include <mutex>
#include <stdexcept>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "snapshot_service.hpp"

#include <catch2/catch_test_macros.hpp>
#include <spdlog/logger.h>
#include <spdlog/sinks/null_sink.h>
#include <sys/timex.h>

#include "ocr_latest_frame_store.hpp"
#include "timelapse_service.hpp"

namespace {

struct TimelapseAppendCall {
    std::vector<std::uint8_t> jpeg;
    std::string captured_at;
    std::string capture_id;
};

std::mutex timelapse_calls_mutex;
std::vector<TimelapseAppendCall>* timelapse_calls {nullptr};
int* timelapse_failures_remaining {nullptr};
const std::vector<std::uint8_t>* timelapse_failure_jpeg {nullptr};

class TimelapseCallRecorder {
  public:
    explicit TimelapseCallRecorder(
        std::vector<TimelapseAppendCall>& calls,
        int* failures_remaining = nullptr,
        const std::vector<std::uint8_t>* failure_jpeg = nullptr)
    {
        std::lock_guard lock(timelapse_calls_mutex);
        timelapse_calls = &calls;
        timelapse_failures_remaining = failures_remaining;
        timelapse_failure_jpeg = failure_jpeg;
    }

    ~TimelapseCallRecorder()
    {
        std::lock_guard lock(timelapse_calls_mutex);
        timelapse_calls = nullptr;
        timelapse_failures_remaining = nullptr;
        timelapse_failure_jpeg = nullptr;
    }

    TimelapseCallRecorder(const TimelapseCallRecorder&) = delete;
    TimelapseCallRecorder& operator=(const TimelapseCallRecorder&) = delete;
};

}  // namespace

// The production TimelapseService implementation is not linked into the
// focused unit-test target. Provide the minimal symbols needed by
// SnapshotService and record append metadata for consistency tests.
TimelapseService::TimelapseService(AxeraUtils& axera_utils,
                                   TimelapseConfig config,
                                   ILogger& logger)
    : _axera_utils(axera_utils)
    , _config(std::move(config))
    , _logger(logger)
{}

TimelapseService::~TimelapseService() = default;

bool TimelapseService::append_snapshot_frame(
    const AxeraUtils::SnapshotResult& result,
    const std::string& captured_at,
    const std::string& capture_id)
{
    std::lock_guard lock(timelapse_calls_mutex);
    if (timelapse_calls != nullptr) {
        timelapse_calls->push_back(TimelapseAppendCall {
            .jpeg = result.data,
            .captured_at = captured_at,
            .capture_id = capture_id,
        });
    }
    if (timelapse_failures_remaining != nullptr
        && *timelapse_failures_remaining > 0
        && (timelapse_failure_jpeg == nullptr
            || result.data == *timelapse_failure_jpeg)) {
        --*timelapse_failures_remaining;
        return false;
    }
    return true;
}

// Make the production time-synchronization gate deterministic for this test
// executable instead of depending on the build host or target device NTP
// state. glibc's dynamic symbol is safely interposed by the executable.
extern "C" int adjtimex(struct timex* value) noexcept
{
    value->status = 0;
    return TIME_OK;
}

namespace {

using namespace std::chrono_literals;

class NullLogger final : public ILogger {
  public:
    NullLogger()
    {
        auto sink = std::make_shared<spdlog::sinks::null_sink_mt>();
        _logger = std::make_shared<spdlog::logger>("snapshot-service-test",
                                                   std::move(sink));
    }

    void logger_init(const std::string&,
                     const std::string&,
                     const std::string&) override
    {}

    std::shared_ptr<spdlog::logger> operator->() override { return _logger; }

  private:
    std::shared_ptr<spdlog::logger> _logger;
};

class TemporaryDirectory {
  public:
    TemporaryDirectory()
    {
        std::string pattern = "/tmp/kvm-vin-snapshot-service-XXXXXX";
        std::vector<char> writable(pattern.begin(), pattern.end());
        writable.push_back('\0');
        const char* path = ::mkdtemp(writable.data());
        if (path == nullptr) {
            throw std::runtime_error(
                "failed to create snapshot test directory");
        }
        _path = path;
    }

    ~TemporaryDirectory()
    {
        std::error_code error;
        (void)std::filesystem::remove_all(_path, error);
    }

    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;

    [[nodiscard]] const std::string& path() const { return _path; }

  private:
    std::string _path;
};

class CurrentDirectoryGuard {
  public:
    explicit CurrentDirectoryGuard(const std::filesystem::path& path)
        : _previous(std::filesystem::current_path())
    { std::filesystem::current_path(path); }

    ~CurrentDirectoryGuard()
    {
        std::error_code error;
        std::filesystem::current_path(_previous, error);
    }

    CurrentDirectoryGuard(const CurrentDirectoryGuard&) = delete;
    CurrentDirectoryGuard& operator=(const CurrentDirectoryGuard&) = delete;

  private:
    std::filesystem::path _previous;
};

template<typename Predicate>
bool wait_until(Predicate predicate, std::chrono::milliseconds timeout = 3s)
{
    const auto deadline = std::chrono::steady_clock::now() + timeout;
    while (std::chrono::steady_clock::now() < deadline) {
        if (predicate()) {
            return true;
        }
        std::this_thread::sleep_for(5ms);
    }
    return predicate();
}

}  // namespace

TEST_CASE("Snapshot service publishes canonical JPEGs without filesystem writes",
          "[capture][snapshot-service][ocr-latest]")
{
    constexpr int required_captures = 4;
    const std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0x42, 0xff, 0xd9};
    std::atomic<int> captures {0};
    std::atomic<bool> all_periodic {true};

    CaptureCoordinator coordinator(
        [&captures, &all_periodic, &jpeg](const CaptureRequest& request,
                                          AxeraUtils::SnapshotResult& result) {
            if (request.source != CaptureSource::Periodic) {
                all_periodic.store(false);
            }
            ++captures;
            result.width = 1920;
            result.height = 1080;
            result.data = jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    TemporaryDirectory output;
    CurrentDirectoryGuard working_directory(output.path());
    OCRLatestFrameStore latest_store;
    std::atomic<bool> running {true};
    NullLogger logger;
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            nullptr,
                            &latest_store);
    service.start();

    REQUIRE(wait_until(
        [&captures]() { return captures.load() >= required_captures; }));
    service.stop();
    running.store(false);
    coordinator.stop();

    REQUIRE(captures.load() >= required_captures);
    CHECK(all_periodic.load());
    const auto latest = latest_store.get_latest("", 0, 0ms);
    REQUIRE(latest.status == OCRLatestFrameStatus::Frame);
    REQUIRE(latest.latest);
    CHECK(latest.latest->sequence == 1);
    CHECK(latest.latest->frame->data == jpeg);

    CHECK(std::filesystem::is_empty(output.path()));
}

TEST_CASE("Snapshot service sends the in-memory canonical frame to Timelapse",
          "[capture][snapshot-service][ocr-latest]")
{
    const std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0x66, 0xff, 0xd9};
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures, &jpeg](const CaptureRequest&,
                           AxeraUtils::SnapshotResult& result) {
            ++captures;
            result.width = 1920;
            result.height = 1080;
            result.data = jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    OCRLatestFrameStore latest_store;
    std::atomic<bool> running {true};
    NullLogger logger;
    AxeraUtils axera_utils(1920, 1080, 60);
    TimelapseService timelapse_service(
        axera_utils, TimelapseConfig {.enable = true}, logger);
    std::vector<TimelapseAppendCall> calls;
    TimelapseCallRecorder recorder(calls);
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            &timelapse_service,
                            &latest_store);
    service.start();

    REQUIRE(wait_until([&calls, &captures]() {
        std::lock_guard lock(timelapse_calls_mutex);
        return captures.load() >= 4 && !calls.empty();
    }));
    service.stop();
    running.store(false);
    coordinator.stop();

    const auto latest = latest_store.get_latest("", 0, 0ms);
    REQUIRE(latest.status == OCRLatestFrameStatus::Frame);
    REQUIRE(latest.latest);
    REQUIRE(latest.latest->sequence == 1);

    TimelapseAppendCall call;
    {
        std::lock_guard lock(timelapse_calls_mutex);
        REQUIRE(calls.size() == 1);
        call = calls.front();
    }
    CHECK(call.jpeg == jpeg);
    CHECK(call.capture_id == latest.latest->capture_id);
    CHECK(call.captured_at == latest.latest->captured_at);
    CHECK(captures.load() >= 4);
}

TEST_CASE("Snapshot service retries Timelapse without republishing OCR",
          "[capture][snapshot-service][ocr-latest]")
{
    const std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0x77, 0xff, 0xd9};
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures, &jpeg](const CaptureRequest&,
                           AxeraUtils::SnapshotResult& result) {
            ++captures;
            result.width = 1920;
            result.height = 1080;
            result.data = jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    OCRLatestFrameStore latest_store;
    std::atomic<bool> running {true};
    NullLogger logger;
    AxeraUtils axera_utils(1920, 1080, 60);
    TimelapseService timelapse_service(
        axera_utils, TimelapseConfig {.enable = true}, logger);
    std::vector<TimelapseAppendCall> calls;
    int failures_remaining = 1;
    TimelapseCallRecorder recorder(calls, &failures_remaining);
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            &timelapse_service,
                            &latest_store);
    service.start();

    REQUIRE(wait_until([&calls, &captures]() {
        std::lock_guard lock(timelapse_calls_mutex);
        return captures.load() >= 4 && calls.size() >= 2;
    }));
    service.stop();
    running.store(false);
    coordinator.stop();

    const auto latest = latest_store.get_latest("", 0, 0ms);
    REQUIRE(latest.status == OCRLatestFrameStatus::Frame);
    REQUIRE(latest.latest);
    CHECK(latest.latest->sequence == 1);
    {
        std::lock_guard lock(timelapse_calls_mutex);
        REQUIRE(calls.size() == 2);
        CHECK(calls[0].jpeg == jpeg);
        CHECK(calls[1].jpeg == jpeg);
        CHECK(calls[0].capture_id == calls[1].capture_id);
        CHECK(calls[0].captured_at == calls[1].captured_at);
    }
    CHECK(failures_remaining == 0);
}

TEST_CASE("Snapshot service deduplicates a Timelapse-only static frame",
          "[capture][snapshot-service][timelapse]")
{
    constexpr int required_captures = 4;
    const std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0x88, 0xff, 0xd9};
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures, &jpeg](const CaptureRequest&,
                           AxeraUtils::SnapshotResult& result) {
            ++captures;
            result.width = 1920;
            result.height = 1080;
            result.data = jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    std::atomic<bool> running {true};
    NullLogger logger;
    AxeraUtils axera_utils(1920, 1080, 60);
    TimelapseService timelapse_service(
        axera_utils, TimelapseConfig {.enable = true}, logger);
    std::vector<TimelapseAppendCall> calls;
    TimelapseCallRecorder recorder(calls);
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            &timelapse_service,
                            nullptr);
    service.start();

    REQUIRE(wait_until(
        [&captures]() { return captures.load() >= required_captures; }));
    service.stop();
    running.store(false);
    coordinator.stop();

    {
        std::lock_guard lock(timelapse_calls_mutex);
        REQUIRE(calls.size() == 1);
        CHECK(calls.front().jpeg == jpeg);
    }
}

TEST_CASE("Snapshot service isolates Timelapse from OCR publish failures",
          "[capture][snapshot-service][ocr-latest][timelapse]")
{
    constexpr int required_captures = 4;
    const std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0x99, 0xff, 0xd9};
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures, &jpeg](const CaptureRequest&,
                           AxeraUtils::SnapshotResult& result) {
            ++captures;
            result.width = 1920;
            result.height = 1080;
            result.data = jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    OCRLatestFrameStore stopped_store;
    stopped_store.stop();
    std::atomic<bool> running {true};
    NullLogger logger;
    AxeraUtils axera_utils(1920, 1080, 60);
    TimelapseService timelapse_service(
        axera_utils, TimelapseConfig {.enable = true}, logger);
    std::vector<TimelapseAppendCall> calls;
    TimelapseCallRecorder recorder(calls);
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            &timelapse_service,
                            &stopped_store);
    service.start();

    REQUIRE(wait_until(
        [&captures]() { return captures.load() >= required_captures; }));
    service.stop();
    running.store(false);
    coordinator.stop();

    {
        std::lock_guard lock(timelapse_calls_mutex);
        REQUIRE(calls.size() == 1);
        CHECK(calls.front().jpeg == jpeg);
    }
    CHECK(stopped_store.get_latest("", 0, 0ms).status
          == OCRLatestFrameStatus::Stopped);
}

TEST_CASE("Snapshot service republishes OCR when a failed frame rolls back",
          "[capture][snapshot-service][ocr-latest][timelapse]")
{
    const std::vector<std::uint8_t> jpeg_a {0xff, 0xd8, 0xa1, 0xff, 0xd9};
    const std::vector<std::uint8_t> jpeg_b {0xff, 0xd8, 0xb2, 0xff, 0xd9};
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures, &jpeg_a, &jpeg_b](const CaptureRequest&,
                                      AxeraUtils::SnapshotResult& result) {
            const int capture = captures.fetch_add(1);
            result.width = 1920;
            result.height = 1080;
            result.data = capture == 1 ? jpeg_b : jpeg_a;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    OCRLatestFrameStore latest_store;
    std::atomic<bool> running {true};
    NullLogger logger;
    AxeraUtils axera_utils(1920, 1080, 60);
    TimelapseService timelapse_service(
        axera_utils, TimelapseConfig {.enable = true}, logger);
    std::vector<TimelapseAppendCall> calls;
    int failures_remaining = 1;
    TimelapseCallRecorder recorder(calls, &failures_remaining, &jpeg_b);
    SnapshotService service(coordinator,
                            SnapshotConfig {
                                .enable = true,
                                .periodic_enable = true,
                            },
                            running,
                            logger,
                            &timelapse_service,
                            &latest_store);
    service.start();

    REQUIRE(wait_until([&latest_store, &captures]() {
        const auto latest = latest_store.get_latest("", 0, 0ms);
        return captures.load() >= 4 && latest.latest
            && latest.latest->sequence >= 3;
    }));
    service.stop();
    running.store(false);
    coordinator.stop();

    const auto latest = latest_store.get_latest("", 0, 0ms);
    REQUIRE(latest.status == OCRLatestFrameStatus::Frame);
    REQUIRE(latest.latest);
    CHECK(latest.latest->sequence == 3);
    CHECK(latest.latest->frame->data == jpeg_a);
    {
        std::lock_guard lock(timelapse_calls_mutex);
        REQUIRE(calls.size() == 2);
        CHECK(calls[0].jpeg == jpeg_a);
        CHECK(calls[1].jpeg == jpeg_b);
    }
    CHECK(failures_remaining == 0);
}
