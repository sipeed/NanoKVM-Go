#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <stdexcept>

namespace kvm::audio {

enum class OpusDecodeErrorStage {
    InvalidInput,
    Initialization,
    Decode,
    Reset,
};

class OpusDecodeError final : public std::runtime_error {
  public:
    OpusDecodeError(OpusDecodeErrorStage stage,
                    int opus_code,
                    const char* context);

    OpusDecodeErrorStage stage() const noexcept { return stage_; }
    int opusCode() const noexcept { return opus_code_; }

  private:
    OpusDecodeErrorStage stage_;
    int opus_code_;
};

struct DecodedPcm {
    // Borrowed interleaved stereo S16_LE samples. The data remains valid until
    // the same OpusDecoder is decoded or reset again, or is destroyed.
    const std::int16_t* data {nullptr};

    // Number of PCM frames (samples per channel), not the total sample count.
    std::uint32_t frames {0};
};

class OpusDecoder final {
  public:
    static constexpr int SAMPLE_RATE = 48000;
    static constexpr int CHANNELS = 2;
    static constexpr int MAX_FRAMES_PER_CHANNEL = 5760;

    OpusDecoder();
    ~OpusDecoder();

    OpusDecoder(const OpusDecoder&) = delete;
    OpusDecoder& operator=(const OpusDecoder&) = delete;
    OpusDecoder(OpusDecoder&&) = delete;
    OpusDecoder& operator=(OpusDecoder&&) = delete;

    // Decode one complete Opus packet. This method does not perform packet-loss
    // concealment and does not accept an empty packet.
    DecodedPcm decode(const std::uint8_t* encoded, std::size_t size);

    // Reset the decoder prediction state. Any previously returned DecodedPcm
    // view must be treated as invalid after this call.
    void reset();

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

}  // namespace kvm::audio
