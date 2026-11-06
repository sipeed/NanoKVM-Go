#include <cstdio>
#include <stdexcept>
#include <string>

#include "log.h"

#include <catch2/catch_test_macros.hpp>
#include <unistd.h>

namespace {

class StreamCapture {
  public:
    explicit StreamCapture(FILE* stream)
        : stream_(stream)
        , savedFd_(dup(fileno(stream)))
        , capture_(tmpfile())
    {
        if (savedFd_ < 0 || capture_ == nullptr
            || dup2(fileno(capture_), fileno(stream_)) < 0) {
            throw std::runtime_error("failed to capture stdio stream");
        }
    }

    ~StreamCapture()
    {
        fflush(stream_);
        if (savedFd_ >= 0) {
            (void)dup2(savedFd_, fileno(stream_));
            close(savedFd_);
        }
        if (capture_ != nullptr) {
            fclose(capture_);
        }
    }

    StreamCapture(const StreamCapture&) = delete;
    StreamCapture& operator=(const StreamCapture&) = delete;

    std::string read()
    {
        fflush(stream_);
        rewind(capture_);
        std::string output;
        char buffer[256];
        std::size_t count = 0;
        while ((count = fread(buffer, 1, sizeof(buffer), capture_)) > 0) {
            output.append(buffer, count);
        }
        return output;
    }

  private:
    FILE* stream_ {nullptr};
    int savedFd_ {-1};
    FILE* capture_ {nullptr};
};

}  // namespace

TEST_CASE("OCR log levels route operational events to stderr", "[ocr][log]")
{
    StreamCapture stdoutCapture(stdout);
    StreamCapture stderrCapture(stderr);
    log_set_level(LogLevel::INFO);

    log_info("test", "ordinary OCR detail");
    log_warn("test", "sequence gap %d", 3);
    log_error("test", "socket unavailable");

    const std::string standardOutput = stdoutCapture.read();
    const std::string errorOutput = stderrCapture.read();
    CHECK(standardOutput.find("ordinary OCR detail") != std::string::npos);
    CHECK(standardOutput.find("sequence gap") == std::string::npos);
    CHECK(standardOutput.find("socket unavailable") == std::string::npos);
    CHECK(errorOutput.find("warning: sequence gap 3") != std::string::npos);
    CHECK(errorOutput.find("socket unavailable") != std::string::npos);

    log_set_level(LogLevel::INFO);
}

TEST_CASE("OCR log formatting preserves prefixes and width-correct values", "[ocr][log]")
{
    StreamCapture stdoutCapture(stdout);
    StreamCapture stderrCapture(stderr);
    log_set_level(LogLevel::INFO);

    const std::size_t frameCount = 42;
    const long long durationMs = 9876543210LL;
    log(LogLevel::INFO,
        "generic",
        "frames=%zu duration=%lld label=%s",
        frameCount,
        durationMs,
        "ready");
    log_info("info", "frame=%u size=%zu", 7U, frameCount);
    log_warn("warn", "retry=%d reason=%s", 3, "gap");
    log_error("error", "code=%d offset=%lld", -2, durationMs);

    CHECK(stdoutCapture.read()
          == "[generic] frames=42 duration=9876543210 label=ready\n"
             "[info] frame=7 size=42\n");
    CHECK(stderrCapture.read()
          == "[warn] warning: retry=3 reason=gap\n"
             "[error] code=-2 offset=9876543210\n");

    log_set_level(LogLevel::INFO);
}

TEST_CASE("error log level suppresses info while retaining warnings and errors",
          "[ocr][log]")
{
    StreamCapture stdoutCapture(stdout);
    StreamCapture stderrCapture(stderr);
    log_set_level(LogLevel::ERROR);

    log(LogLevel::INFO, "generic", "hidden %d", 1);
    log_info("info", "hidden %s", "detail");
    log_warn("warn", "visible %d", 2);
    log_error("error", "visible %d", 3);

    CHECK(stdoutCapture.read().empty());
    CHECK(stderrCapture.read()
          == "[warn] warning: visible 2\n[error] visible 3\n");

    log_set_level(LogLevel::INFO);
}

TEST_CASE("none log level suppresses stdout and stderr", "[ocr][log]")
{
    StreamCapture stdoutCapture(stdout);
    StreamCapture stderrCapture(stderr);
    log_set_level(LogLevel::NONE);

    log_info("test", "hidden info");
    log_warn("test", "hidden warning");
    log_error("test", "hidden error");

    CHECK(stdoutCapture.read().empty());
    CHECK(stderrCapture.read().empty());

    log_set_level(LogLevel::INFO);
}
