#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <stdexcept>

namespace kvm::audio {

enum class OpusEncodeErrorStage {
    InvalidInput,
    Initialization,
    Configuration,
    Encode,
};

class OpusEncodeError final : public std::runtime_error {
  public:
    OpusEncodeError(OpusEncodeErrorStage stage,
                    int opus_code,
                    const char* context);

    OpusEncodeErrorStage stage() const noexcept { return stage_; }
    int opusCode() const noexcept { return opus_code_; }

  private:
    OpusEncodeErrorStage stage_;
    int opus_code_;
};

class OpusEncoder final {
  public:
    static constexpr int SAMPLE_RATE = 48000;
    static constexpr int CHANNELS = 2;
    static constexpr int FRAME_MS = 20;
    static constexpr int FRAME_FRAMES = (SAMPLE_RATE * FRAME_MS) / 1000;
    static constexpr int BITRATE = 128000;
    static constexpr int COMPLEXITY = 4;
    static constexpr std::size_t MAX_PACKET_BYTES = 1500;

    OpusEncoder();
    ~OpusEncoder();

    OpusEncoder(const OpusEncoder&) = delete;
    OpusEncoder& operator=(const OpusEncoder&) = delete;
    OpusEncoder(OpusEncoder&&) = delete;
    OpusEncoder& operator=(OpusEncoder&&) = delete;

    std::uint32_t encode(const std::int16_t* pcm,
                         std::uint32_t frames,
                         std::uint8_t* output,
                         std::size_t capacity);

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

}  // namespace kvm::audio
