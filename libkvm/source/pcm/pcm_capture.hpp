#pragma once

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <string>

#include "opus_encoder.hpp"

namespace kvm::audio {

class AudioCaptureSource {
  public:
    virtual ~AudioCaptureSource() = default;
    virtual std::uint32_t read(std::uint8_t* output, std::size_t capacity) = 0;
    virtual void requestStop() noexcept = 0;
};

class AudioCapturer final : public AudioCaptureSource {
  public:
    static constexpr const char* DEFAULT_DEVICE = "hw:Lt7911EXC,0";
    static constexpr const char* DEVICE_ENV = "LIBKVM_AUDIO_CAPTURE_DEVICE";
    static constexpr const char* VIRTUAL_SPEAKER_FLAG = "/boot/usb.uac2_speaker";
    static constexpr const char* VIRTUAL_SPEAKER_DEVICE = "hw:UAC2Gadget,0";
    static constexpr unsigned int RATE = OpusEncoder::SAMPLE_RATE;
    static constexpr unsigned int CHANNELS = OpusEncoder::CHANNELS;
    static constexpr unsigned int PERIOD_MS = OpusEncoder::FRAME_MS;
    static constexpr unsigned int PERIOD_FRAMES = OpusEncoder::FRAME_FRAMES;
    static constexpr unsigned int BYTES_PER_SAMPLE = 2;
    static constexpr unsigned int FRAME_BYTES = CHANNELS * BYTES_PER_SAMPLE;
    static constexpr unsigned int PERIOD_BYTES = PERIOD_FRAMES * FRAME_BYTES;
    static constexpr std::size_t MAX_PACKET_BYTES =
        OpusEncoder::MAX_PACKET_BYTES;

    AudioCapturer();
    explicit AudioCapturer(std::string device_name);
    ~AudioCapturer() override;

    AudioCapturer(const AudioCapturer&) = delete;
    AudioCapturer& operator=(const AudioCapturer&) = delete;
    AudioCapturer(AudioCapturer&&) = delete;
    AudioCapturer& operator=(AudioCapturer&&) = delete;

    std::uint32_t read(std::uint8_t* output, std::size_t capacity) override;
    void requestStop() noexcept override;

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

enum class AudioCaptureReadStatus {
    Success,
    NoData,
    RetryLater,
    Failed,
    Stopped,
};

struct AudioCaptureReadResult {
    AudioCaptureReadStatus status {AudioCaptureReadStatus::Failed};
    std::uint32_t size {0};
    bool device_initialized {false};
    std::string error;
};

class AudioCaptureState {
  public:
    using Factory = std::function<std::unique_ptr<AudioCaptureSource>()>;

    explicit AudioCaptureState(
        Factory factory = {},
        std::chrono::milliseconds retry_delay = std::chrono::seconds(1));
    ~AudioCaptureState();

    AudioCaptureState(const AudioCaptureState&) = delete;
    AudioCaptureState& operator=(const AudioCaptureState&) = delete;

    void start();
    void stop() noexcept;
    AudioCaptureReadResult read(std::uint8_t* output, std::size_t capacity);

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

}  // namespace kvm::audio
