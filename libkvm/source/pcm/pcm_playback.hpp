#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <string>

namespace kvm::audio {

struct PcmPlaybackParameters {
    std::uint32_t rate {0};
    std::uint32_t channels {0};
    std::uint32_t period_frames {0};
    std::uint32_t buffer_frames {0};
    std::uint32_t frame_bytes {0};
};

class PcmPlaybackSink {
  public:
    virtual ~PcmPlaybackSink() = default;

    virtual void write(const std::int16_t* pcm,
                       std::uint32_t frames,
                       std::uint32_t timeout_ms) = 0;
    virtual void reset() = 0;
    virtual void close() noexcept = 0;
};

class PcmPlayback final : public PcmPlaybackSink {
  public:
    static constexpr const char* DEFAULT_DEVICE = "hw:UAC2Gadget,0";
    static constexpr std::uint32_t RATE = 48000;
    static constexpr std::uint32_t CHANNELS = 2;
    static constexpr std::uint32_t PERIOD_MS = 20;
    static constexpr std::uint32_t PERIOD_FRAMES = (RATE * PERIOD_MS) / 1000;
    static constexpr std::uint32_t BUFFER_PERIODS = 4;
    static constexpr std::uint32_t BUFFER_FRAMES =
        PERIOD_FRAMES * BUFFER_PERIODS;
    static constexpr std::uint32_t BYTES_PER_SAMPLE = 2;
    static constexpr std::uint32_t FRAME_BYTES = CHANNELS * BYTES_PER_SAMPLE;

    explicit PcmPlayback(std::string device_name = DEFAULT_DEVICE);
    ~PcmPlayback() override;

    PcmPlayback(const PcmPlayback&) = delete;
    PcmPlayback& operator=(const PcmPlayback&) = delete;
    PcmPlayback(PcmPlayback&&) = delete;
    PcmPlayback& operator=(PcmPlayback&&) = delete;

    void write(const std::int16_t* pcm,
               std::uint32_t frames,
               std::uint32_t timeout_ms) override;
    void reset() override;
    void close() noexcept override;

    const PcmPlaybackParameters& parameters() const noexcept;

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

}  // namespace kvm::audio
