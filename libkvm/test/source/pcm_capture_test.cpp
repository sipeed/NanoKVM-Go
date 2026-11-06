#include <algorithm>
#include <array>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <future>
#include <memory>
#include <mutex>
#include <stdexcept>
#include <string_view>
#include <vector>

#include "pcm/pcm_capture.hpp"

#include <catch2/catch_test_macros.hpp>

#include "pcm/pcm_device.hpp"

namespace {

using kvm::audio::AudioCapturer;
using kvm::audio::AudioCaptureReadStatus;
using kvm::audio::AudioCaptureSource;
using kvm::audio::AudioCaptureState;
using kvm::audio::OpusEncoder;
using kvm::audio::PcmError;
using kvm::audio::PcmReadBackend;
using kvm::audio::PcmReadTimeout;
using kvm::audio::PcmRecoveryAction;
using kvm::audio::readInterleavedFrames;

struct CaptureCounters {
    std::atomic<int> created {0};
    std::atomic<int> destroyed {0};
    std::atomic<int> stop_requested {0};
};

class FakeCaptureSource final : public AudioCaptureSource {
  public:
    FakeCaptureSource(std::shared_ptr<CaptureCounters> counters,
                      std::vector<std::uint8_t> frame)
        : counters_(std::move(counters))
        , frame_(std::move(frame))
    { ++counters_->created; }

    ~FakeCaptureSource() override { ++counters_->destroyed; }

    std::uint32_t read(std::uint8_t* output, std::size_t capacity) override
    {
        if (capacity < frame_.size()) {
            throw std::runtime_error("fake output buffer is too small");
        }
        std::copy(frame_.begin(), frame_.end(), output);
        return static_cast<std::uint32_t>(frame_.size());
    }

    void requestStop() noexcept override { ++counters_->stop_requested; }

  private:
    std::shared_ptr<CaptureCounters> counters_;
    std::vector<std::uint8_t> frame_;
};

class FailingCaptureSource final : public AudioCaptureSource {
  public:
    explicit FailingCaptureSource(std::shared_ptr<CaptureCounters> counters)
        : counters_(std::move(counters))
    { ++counters_->created; }

    ~FailingCaptureSource() override { ++counters_->destroyed; }

    std::uint32_t read(std::uint8_t*, std::size_t) override
    { throw std::runtime_error("capture disconnected"); }

    void requestStop() noexcept override { ++counters_->stop_requested; }

  private:
    std::shared_ptr<CaptureCounters> counters_;
};

class TimeoutThenSuccessSource final : public AudioCaptureSource {
  public:
    explicit TimeoutThenSuccessSource(std::shared_ptr<CaptureCounters> counters)
        : counters_(std::move(counters))
    { ++counters_->created; }

    ~TimeoutThenSuccessSource() override { ++counters_->destroyed; }

    std::uint32_t read(std::uint8_t* output, std::size_t capacity) override
    {
        if (calls_++ == 0) {
            throw PcmReadTimeout();
        }
        if (capacity == 0) {
            throw std::runtime_error("fake output buffer is too small");
        }
        output[0] = 9;
        return 1;
    }

    void requestStop() noexcept override { ++counters_->stop_requested; }

  private:
    std::shared_ptr<CaptureCounters> counters_;
    int calls_ {0};
};

class OversizedCaptureSource final : public AudioCaptureSource {
  public:
    explicit OversizedCaptureSource(std::shared_ptr<CaptureCounters> counters)
        : counters_(std::move(counters))
    { ++counters_->created; }

    ~OversizedCaptureSource() override { ++counters_->destroyed; }

    std::uint32_t read(std::uint8_t*, std::size_t capacity) override
    { return static_cast<std::uint32_t>(capacity + 1); }

    void requestStop() noexcept override { ++counters_->stop_requested; }

  private:
    std::shared_ptr<CaptureCounters> counters_;
};

struct BlockingCaptureControl {
    std::mutex mutex;
    std::condition_variable condition;
    bool entered {false};
    bool stopped {false};
};

class BlockingCaptureSource final : public AudioCaptureSource {
  public:
    BlockingCaptureSource(std::shared_ptr<CaptureCounters> counters,
                          std::shared_ptr<BlockingCaptureControl> control)
        : counters_(std::move(counters))
        , control_(std::move(control))
    { ++counters_->created; }

    ~BlockingCaptureSource() override { ++counters_->destroyed; }

    std::uint32_t read(std::uint8_t*, std::size_t) override
    {
        std::unique_lock<std::mutex> lock(control_->mutex);
        control_->entered = true;
        control_->condition.notify_all();
        control_->condition.wait(lock, [this]() { return control_->stopped; });
        return 0;
    }

    void requestStop() noexcept override
    {
        ++counters_->stop_requested;
        std::lock_guard<std::mutex> lock(control_->mutex);
        control_->stopped = true;
        control_->condition.notify_all();
    }

  private:
    std::shared_ptr<CaptureCounters> counters_;
    std::shared_ptr<BlockingCaptureControl> control_;
};

class ScriptedPcmBackend final : public PcmReadBackend {
  public:
    explicit ScriptedPcmBackend(std::vector<snd_pcm_sframes_t> results)
        : results_(std::move(results))
    {}

    snd_pcm_sframes_t read(void*, snd_pcm_uframes_t frames) override
    {
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
        return wait_result;
    }

    PcmRecoveryAction recover(int error) override
    {
        recovered_errors.push_back(error);
        if (error == -EINTR) {
            return PcmRecoveryAction::Retry;
        }
        if (error == -EPIPE || error == -ESTRPIPE) {
            return PcmRecoveryAction::Restarted;
        }
        throw PcmError("scripted PCM failure", error);
    }

    std::chrono::steady_clock::time_point now() const override
    { return current_time_; }

    std::vector<snd_pcm_uframes_t> requested_frames;
    std::vector<int> wait_timeouts;
    std::vector<int> recovered_errors;
    int wait_result {0};

  private:
    std::vector<snd_pcm_sframes_t> results_;
    std::size_t next_result_ {0};
    std::chrono::steady_clock::time_point current_time_ {};
};

}  // namespace

TEST_CASE("Audio capture constants describe one 20 ms stereo frame",
          "[audio][capture]")
{
    STATIC_REQUIRE(std::string_view(AudioCapturer::DEFAULT_DEVICE)
                   == "hw:Lt7911EXC,0");
    STATIC_REQUIRE(std::string_view(AudioCapturer::DEVICE_ENV)
                   == "LIBKVM_AUDIO_CAPTURE_DEVICE");
    STATIC_REQUIRE(std::string_view(AudioCapturer::VIRTUAL_SPEAKER_FLAG)
                   == "/boot/usb.uac2_speaker");
    STATIC_REQUIRE(std::string_view(AudioCapturer::VIRTUAL_SPEAKER_DEVICE)
                   == "hw:UAC2Gadget,0");
    STATIC_REQUIRE(AudioCapturer::RATE == OpusEncoder::SAMPLE_RATE);
    STATIC_REQUIRE(AudioCapturer::CHANNELS == OpusEncoder::CHANNELS);
    STATIC_REQUIRE(AudioCapturer::PERIOD_MS == OpusEncoder::FRAME_MS);
    STATIC_REQUIRE(AudioCapturer::PERIOD_FRAMES == OpusEncoder::FRAME_FRAMES);
    STATIC_REQUIRE(AudioCapturer::MAX_PACKET_BYTES
                   == OpusEncoder::MAX_PACKET_BYTES);
    REQUIRE(AudioCapturer::RATE == 48000);
    REQUIRE(AudioCapturer::CHANNELS == 2);
    REQUIRE(AudioCapturer::PERIOD_MS == 20);
    REQUIRE(AudioCapturer::PERIOD_FRAMES == 960);
    REQUIRE(AudioCapturer::FRAME_BYTES == 4);
    REQUIRE(AudioCapturer::PERIOD_BYTES == 3840);
    REQUIRE(1000 / AudioCapturer::PERIOD_MS == 50);
}

TEST_CASE("PCM reads discard partial frames after an XRUN", "[audio][pcm]")
{
    ScriptedPcmBackend backend({240, -EPIPE, 960});
    std::array<std::uint8_t, AudioCapturer::PERIOD_BYTES> output {};
    std::atomic<bool> cancelled {false};

    readInterleavedFrames(output.data(),
                          AudioCapturer::PERIOD_FRAMES,
                          AudioCapturer::FRAME_BYTES,
                          cancelled,
                          200,
                          backend);

    REQUIRE(backend.requested_frames.size() == 3);
    REQUIRE(backend.requested_frames[0] == 960);
    REQUIRE(backend.requested_frames[1] == 720);
    REQUIRE(backend.requested_frames[2] == 960);
    REQUIRE(backend.recovered_errors == std::vector<int> {-EPIPE});
}

TEST_CASE("PCM reads retain partial frames after EINTR", "[audio][pcm]")
{
    ScriptedPcmBackend backend({240, -EINTR, 720});
    std::array<std::uint8_t, AudioCapturer::PERIOD_BYTES> output {};
    std::atomic<bool> cancelled {false};

    readInterleavedFrames(output.data(),
                          AudioCapturer::PERIOD_FRAMES,
                          AudioCapturer::FRAME_BYTES,
                          cancelled,
                          200,
                          backend);

    REQUIRE(backend.requested_frames.size() == 3);
    REQUIRE(backend.requested_frames[0] == 960);
    REQUIRE(backend.requested_frames[1] == 720);
    REQUIRE(backend.requested_frames[2] == 720);
    REQUIRE(backend.recovered_errors == std::vector<int> {-EINTR});
}

TEST_CASE("PCM reads have a total deadline", "[audio][pcm]")
{
    ScriptedPcmBackend backend({});
    std::array<std::uint8_t, AudioCapturer::PERIOD_BYTES> output {};
    std::atomic<bool> cancelled {false};

    REQUIRE_THROWS_AS(readInterleavedFrames(output.data(),
                                            AudioCapturer::PERIOD_FRAMES,
                                            AudioCapturer::FRAME_BYTES,
                                            cancelled,
                                            120,
                                            backend),
                      PcmReadTimeout);
    REQUIRE(backend.wait_timeouts == std::vector<int> {50, 50, 20});
    REQUIRE(backend.recovered_errors.empty());
}

TEST_CASE("PCM device errors are not treated as recoverable", "[audio][pcm]")
{
    ScriptedPcmBackend backend({-ENODEV});
    std::array<std::uint8_t, AudioCapturer::PERIOD_BYTES> output {};
    std::atomic<bool> cancelled {false};

    REQUIRE_THROWS_AS(readInterleavedFrames(output.data(),
                                            AudioCapturer::PERIOD_FRAMES,
                                            AudioCapturer::FRAME_BYTES,
                                            cancelled,
                                            200,
                                            backend),
                      PcmError);
    REQUIRE(backend.recovered_errors == std::vector<int> {-ENODEV});
}

TEST_CASE("Audio capture retries after an initialization failure",
          "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    int attempts = 0;
    AudioCaptureState state(
        [&]() -> std::unique_ptr<AudioCaptureSource> {
            ++attempts;
            if (attempts == 1) {
                throw std::runtime_error("device unavailable");
            }
            return std::make_unique<FakeCaptureSource>(
                counters, std::vector<std::uint8_t> {1, 2, 3});
        },
        std::chrono::milliseconds::zero());
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    const auto first = state.read(output.data(), output.size());
    REQUIRE(first.status == AudioCaptureReadStatus::Failed);
    REQUIRE(first.error == "device unavailable");

    const auto second = state.read(output.data(), output.size());
    REQUIRE(second.status == AudioCaptureReadStatus::Success);
    REQUIRE(second.device_initialized);
    REQUIRE(second.size == 3);
    REQUIRE(output[0] == 1);
    REQUIRE(output[1] == 2);
    REQUIRE(output[2] == 3);

    const auto third = state.read(output.data(), output.size());
    REQUIRE(third.status == AudioCaptureReadStatus::Success);
    REQUIRE_FALSE(third.device_initialized);
    REQUIRE(attempts == 2);
}

TEST_CASE("Audio capture initialization retries are rate limited",
          "[audio][capture]")
{
    int attempts = 0;
    AudioCaptureState state(
        [&]() -> std::unique_ptr<AudioCaptureSource> {
            ++attempts;
            throw std::runtime_error("device unavailable");
        },
        std::chrono::hours(1));
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Failed);
    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::RetryLater);
    REQUIRE(attempts == 1);
}

TEST_CASE("Audio capture rebuilds the device after a read failure",
          "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    int attempts = 0;
    AudioCaptureState state(
        [&]() -> std::unique_ptr<AudioCaptureSource> {
            ++attempts;
            if (attempts == 1) {
                return std::make_unique<FailingCaptureSource>(counters);
            }
            return std::make_unique<FakeCaptureSource>(
                counters, std::vector<std::uint8_t> {7, 8});
        },
        std::chrono::milliseconds::zero());
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    const auto failed = state.read(output.data(), output.size());
    REQUIRE(failed.status == AudioCaptureReadStatus::Failed);
    REQUIRE(failed.error == "capture disconnected");
    REQUIRE(counters->destroyed == 1);

    const auto recovered = state.read(output.data(), output.size());
    REQUIRE(recovered.status == AudioCaptureReadStatus::Success);
    REQUIRE(recovered.device_initialized);
    REQUIRE(recovered.size == 2);
    REQUIRE(output[0] == 7);
    REQUIRE(output[1] == 8);
    REQUIRE(attempts == 2);
}

TEST_CASE("Audio capture timeout keeps the current device", "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    int attempts = 0;
    AudioCaptureState state(
        [&]() {
            ++attempts;
            return std::make_unique<TimeoutThenSuccessSource>(counters);
        },
        std::chrono::hours(1));
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    const auto no_data = state.read(output.data(), output.size());
    REQUIRE(no_data.status == AudioCaptureReadStatus::NoData);
    REQUIRE(no_data.device_initialized);
    REQUIRE(counters->destroyed == 0);

    const auto recovered = state.read(output.data(), output.size());
    REQUIRE(recovered.status == AudioCaptureReadStatus::Success);
    REQUIRE_FALSE(recovered.device_initialized);
    REQUIRE(recovered.size == 1);
    REQUIRE(output[0] == 9);
    REQUIRE(attempts == 1);
}

TEST_CASE("Audio capture rejects oversized source output", "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    AudioCaptureState state(
        [&]() { return std::make_unique<OversizedCaptureSource>(counters); },
        std::chrono::milliseconds::zero());
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    const auto result = state.read(output.data(), output.size());
    REQUIRE(result.status == AudioCaptureReadStatus::Failed);
    REQUIRE(result.error == "Audio capture returned more bytes than requested");
    REQUIRE(counters->destroyed == 1);
}

TEST_CASE("Audio capture start clears initialization retry cooldown",
          "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    int attempts = 0;
    AudioCaptureState state(
        [&]() -> std::unique_ptr<AudioCaptureSource> {
            ++attempts;
            if (attempts == 1) {
                throw std::runtime_error("device unavailable");
            }
            return std::make_unique<FakeCaptureSource>(
                counters, std::vector<std::uint8_t> {1});
        },
        std::chrono::hours(1));
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Failed);
    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::RetryLater);

    state.stop();
    state.start();
    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Success);
    REQUIRE(attempts == 2);
}

TEST_CASE("Audio capture stop releases the device and start rebuilds it",
          "[audio][capture]")
{
    auto counters = std::make_shared<CaptureCounters>();
    AudioCaptureState state(
        [&]() {
            return std::make_unique<FakeCaptureSource>(
                counters, std::vector<std::uint8_t> {4, 5});
        },
        std::chrono::milliseconds::zero());
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Success);
    state.stop();
    REQUIRE(counters->destroyed == 1);
    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Stopped);

    state.start();
    REQUIRE(state.read(output.data(), output.size()).status
            == AudioCaptureReadStatus::Success);
    REQUIRE(counters->created == 2);
    state.stop();
    REQUIRE(counters->destroyed == 2);
}

TEST_CASE("Audio capture stop cancels and joins an active read",
          "[audio][capture]")
{
    using namespace std::chrono_literals;

    auto counters = std::make_shared<CaptureCounters>();
    auto control = std::make_shared<BlockingCaptureControl>();
    AudioCaptureState state(
        [=]() {
            return std::make_unique<BlockingCaptureSource>(counters, control);
        },
        std::chrono::milliseconds::zero());
    std::array<std::uint8_t, AudioCapturer::MAX_PACKET_BYTES> output {};

    auto pending = std::async(std::launch::async, [&]() {
        return state.read(output.data(), output.size());
    });

    {
        std::unique_lock<std::mutex> lock(control->mutex);
        REQUIRE(control->condition.wait_for(
            lock, 1s, [&]() { return control->entered; }));
    }

    state.stop();
    REQUIRE(pending.wait_for(1s) == std::future_status::ready);
    REQUIRE(pending.get().status == AudioCaptureReadStatus::Stopped);
    REQUIRE(counters->stop_requested == 1);
    REQUIRE(counters->destroyed == 1);
}
