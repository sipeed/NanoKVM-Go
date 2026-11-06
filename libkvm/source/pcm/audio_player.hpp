#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <stdexcept>

#include "opus_decoder.hpp"
#include "pcm_playback.hpp"

namespace kvm::audio {

class AudioPlayerStateError final : public std::runtime_error {
  public:
    explicit AudioPlayerStateError(const char* message);
};

class AudioPlayer final {
  public:
    static constexpr std::size_t MAX_OPUS_PACKET_BYTES = 1500;
    static constexpr std::uint32_t MAX_TIMEOUT_MS = 5000;

    AudioPlayer();
    explicit AudioPlayer(std::unique_ptr<PcmPlaybackSink> playback);
    ~AudioPlayer();

    AudioPlayer(const AudioPlayer&) = delete;
    AudioPlayer& operator=(const AudioPlayer&) = delete;
    AudioPlayer(AudioPlayer&&) = delete;
    AudioPlayer& operator=(AudioPlayer&&) = delete;

    void writeOpus(const std::uint8_t* opus,
                   std::size_t size,
                   std::uint32_t timeout_ms,
                   bool discontinuity);
    void close() noexcept;
    bool isClosed() const noexcept;

  private:
    OpusDecoder decoder_;
    std::unique_ptr<PcmPlaybackSink> playback_;
    bool closed_ {false};
};

}  // namespace kvm::audio
