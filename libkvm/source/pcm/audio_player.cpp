#include <utility>

#include "audio_player.hpp"

#include <opus/opus_defines.h>

namespace kvm::audio {

AudioPlayerStateError::AudioPlayerStateError(const char* message)
    : std::runtime_error(message)
{}

AudioPlayer::AudioPlayer()
    : AudioPlayer(std::make_unique<PcmPlayback>())
{}

AudioPlayer::AudioPlayer(std::unique_ptr<PcmPlaybackSink> playback)
    : playback_(std::move(playback))
{
    if (playback_ == nullptr) {
        throw std::invalid_argument("Audio playback sink must not be null");
    }
}

AudioPlayer::~AudioPlayer() { close(); }

void AudioPlayer::writeOpus(const std::uint8_t* opus,
                            std::size_t size,
                            std::uint32_t timeout_ms,
                            bool discontinuity)
{
    if (closed_) {
        throw AudioPlayerStateError("Audio playback session is closed");
    }
    if (opus == nullptr || size == 0 || size > MAX_OPUS_PACKET_BYTES) {
        throw std::invalid_argument("Opus packet must be complete and bounded");
    }
    if (timeout_ms == 0 || timeout_ms > MAX_TIMEOUT_MS) {
        throw std::invalid_argument("Audio playback timeout is out of range");
    }

    if (discontinuity) {
        // Reset both predictive codec state and any already queued PCM before
        // decoding the first packet after a Server-side queue drop.
        decoder_.reset();
        playback_->reset();
    }

    const DecodedPcm decoded = decoder_.decode(opus, size);
    if (decoded.data == nullptr || decoded.frames == 0) {
        throw OpusDecodeError(OpusDecodeErrorStage::Decode,
                              OPUS_INTERNAL_ERROR,
                              "Opus decoder returned an empty PCM frame");
    }
    playback_->write(decoded.data, decoded.frames, timeout_ms);
}

void AudioPlayer::close() noexcept
{
    if (closed_) {
        return;
    }
    closed_ = true;
    if (playback_ != nullptr) {
        playback_->close();
    }
}

bool AudioPlayer::isClosed() const noexcept { return closed_; }

}  // namespace kvm::audio
