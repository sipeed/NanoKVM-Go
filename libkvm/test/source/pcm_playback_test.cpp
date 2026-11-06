#include <array>
#include <cerrno>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <utility>
#include <vector>

#include "pcm/pcm_playback.hpp"

#include <catch2/catch_test_macros.hpp>

#include "pcm/pcm_device.hpp"

namespace {

using kvm::audio::PcmError;
using kvm::audio::PcmPlayback;
using kvm::audio::PcmRecoveryAction;
using kvm::audio::PcmWriteBackend;
using kvm::audio::PcmWriteTimeout;
using kvm::audio::writeInterleavedFrames;

class ScriptedPcmWriteBackend final : public PcmWriteBackend {
  public:
    explicit ScriptedPcmWriteBackend(std::vector<snd_pcm_sframes_t> results)
        : results_(std::move(results))
    {}

    snd_pcm_sframes_t write(const void* buffer,
                            snd_pcm_uframes_t frames) override
    {
        buffers.push_back(static_cast<const std::uint8_t*>(buffer));
        requested_frames.push_back(frames);
        if (next_result_ >= results_.size()) {
            return -EAGAIN;
        }
        return results_[next_result_++];
    }

    int wait(int timeout_ms) override
    {
        wait_timeouts.push_back(timeout_ms);
        current_time_ += std::chrono::milliseconds(timeout_ms);
        if (next_wait_result_ >= wait_results.size()) {
            return 0;
        }
        return wait_results[next_wait_result_++];
    }

    PcmRecoveryAction recover(int error) override
    {
        recovered_errors.push_back(error);
        if (error == -EINTR) {
            return PcmRecoveryAction::Retry;
        }
        if (error == -EPIPE) {
            return PcmRecoveryAction::Restarted;
        }
        if (error == -ESTRPIPE) {
            return suspend_recovery;
        }
        throw PcmError("scripted PCM playback failure", error);
    }

    std::chrono::steady_clock::time_point now() const override
    { return current_time_; }

    std::vector<const std::uint8_t*> buffers;
    std::vector<snd_pcm_uframes_t> requested_frames;
    std::vector<int> wait_timeouts;
    std::vector<int> wait_results;
    std::vector<int> recovered_errors;
    PcmRecoveryAction suspend_recovery {PcmRecoveryAction::Retry};

  private:
    std::vector<snd_pcm_sframes_t> results_;
    std::size_t next_result_ {0};
    std::size_t next_wait_result_ {0};
    std::chrono::steady_clock::time_point current_time_ {};
};

std::vector<std::ptrdiff_t> bufferOffsets(
    const std::uint8_t* base, const std::vector<const std::uint8_t*>& buffers)
{
    std::vector<std::ptrdiff_t> offsets;
    offsets.reserve(buffers.size());
    for (const auto* buffer : buffers) {
        offsets.push_back(buffer - base);
    }
    return offsets;
}

}  // namespace

TEST_CASE("PCM playback constants describe four 20 ms stereo periods",
          "[audio][playback]")
{
    STATIC_REQUIRE(PcmPlayback::RATE == 48000);
    STATIC_REQUIRE(PcmPlayback::CHANNELS == 2);
    STATIC_REQUIRE(PcmPlayback::PERIOD_MS == 20);
    STATIC_REQUIRE(PcmPlayback::PERIOD_FRAMES == 960);
    STATIC_REQUIRE(PcmPlayback::BUFFER_PERIODS == 4);
    STATIC_REQUIRE(PcmPlayback::BUFFER_FRAMES == 3840);
    STATIC_REQUIRE(PcmPlayback::BYTES_PER_SAMPLE == 2);
    STATIC_REQUIRE(PcmPlayback::FRAME_BYTES == 4);
}

TEST_CASE("PCM playback advances through partial writes", "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({1, 2, 1});

    writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);

    REQUIRE(backend.requested_frames
            == std::vector<snd_pcm_uframes_t> {4, 3, 1});
    REQUIRE(bufferOffsets(input.data(), backend.buffers)
            == std::vector<std::ptrdiff_t> {0, 4, 12});
    REQUIRE(backend.wait_timeouts.empty());
    REQUIRE(backend.recovered_errors.empty());
}

TEST_CASE("PCM playback waits for EAGAIN without losing its write position",
          "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({1, -EAGAIN, 3});
    backend.wait_results = {1};

    writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);

    REQUIRE(backend.requested_frames
            == std::vector<snd_pcm_uframes_t> {4, 3, 3});
    REQUIRE(bufferOffsets(input.data(), backend.buffers)
            == std::vector<std::ptrdiff_t> {0, 4, 4});
    REQUIRE(backend.wait_timeouts == std::vector<int> {50});
    REQUIRE(backend.recovered_errors.empty());
}

TEST_CASE("PCM playback retries EINTR at the current frame",
          "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({1, -EINTR, 3});

    writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);

    REQUIRE(backend.requested_frames
            == std::vector<snd_pcm_uframes_t> {4, 3, 3});
    REQUIRE(bufferOffsets(input.data(), backend.buffers)
            == std::vector<std::ptrdiff_t> {0, 4, 4});
    REQUIRE(backend.recovered_errors == std::vector<int> {-EINTR});
}

TEST_CASE("PCM playback continues at its current offset after resume",
          "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({1, -ESTRPIPE, 3});

    writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);

    REQUIRE(backend.requested_frames
            == std::vector<snd_pcm_uframes_t> {4, 3, 3});
    REQUIRE(bufferOffsets(input.data(), backend.buffers)
            == std::vector<std::ptrdiff_t> {0, 4, 4});
    REQUIRE(backend.recovered_errors == std::vector<int> {-ESTRPIPE});
}

TEST_CASE("PCM playback replays the complete block after stream restart",
          "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;

    for (const int restart_error : {-EPIPE, -ESTRPIPE}) {
        CAPTURE(restart_error);
        std::array<std::uint8_t, frames * frame_bytes> input {};
        ScriptedPcmWriteBackend backend({1, restart_error, 4});
        backend.suspend_recovery = PcmRecoveryAction::Restarted;

        writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);

        REQUIRE(backend.requested_frames
                == std::vector<snd_pcm_uframes_t> {4, 3, 4});
        REQUIRE(bufferOffsets(input.data(), backend.buffers)
                == std::vector<std::ptrdiff_t> {0, 4, 0});
        REQUIRE(backend.recovered_errors == std::vector<int> {restart_error});
    }
}

TEST_CASE("PCM playback enforces one total write deadline", "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({});

    REQUIRE_THROWS_AS(
        writeInterleavedFrames(input.data(), frames, frame_bytes, 120, backend),
        PcmWriteTimeout);
    REQUIRE(backend.wait_timeouts == std::vector<int> {50, 50, 20});
    REQUIRE(backend.recovered_errors.empty());
}

TEST_CASE("PCM playback surfaces a fatal device error", "[audio][playback]")
{
    constexpr snd_pcm_uframes_t frames = 4;
    constexpr std::size_t frame_bytes = 4;
    std::array<std::uint8_t, frames * frame_bytes> input {};
    ScriptedPcmWriteBackend backend({-ENODEV});

    try {
        writeInterleavedFrames(input.data(), frames, frame_bytes, 200, backend);
        FAIL("Expected a disconnected playback device to fail");
    } catch (const PcmError& error) {
        REQUIRE(error.errorCode() == -ENODEV);
    }
    REQUIRE(backend.recovered_errors == std::vector<int> {-ENODEV});
}

TEST_CASE("PCM playback rejects invalid low-level write parameters",
          "[audio][playback]")
{
    std::array<std::uint8_t, 16> input {};
    ScriptedPcmWriteBackend backend({4});

    REQUIRE_THROWS_AS(writeInterleavedFrames(nullptr, 4, 4, 200, backend),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(writeInterleavedFrames(input.data(), 0, 4, 200, backend),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(writeInterleavedFrames(input.data(), 4, 0, 200, backend),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(writeInterleavedFrames(input.data(), 4, 4, 0, backend),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(writeInterleavedFrames(input.data(), 4, 4, -1, backend),
                      std::invalid_argument);
    REQUIRE(backend.requested_frames.empty());
}
