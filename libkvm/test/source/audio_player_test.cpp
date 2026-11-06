#include <array>
#include <cstdint>
#include <memory>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "pcm/audio_player.hpp"

#include <catch2/catch_test_macros.hpp>
#include <opus/opus.h>

#include "pcm/opus_decoder.hpp"
#include "pcm/pcm_device.hpp"
#include "pcm/pcm_playback.hpp"

namespace {

using kvm::audio::AudioPlayer;
using kvm::audio::AudioPlayerStateError;
using kvm::audio::DecodedPcm;
using kvm::audio::OpusDecodeError;
using kvm::audio::OpusDecodeErrorStage;
using KvmOpusDecoder = kvm::audio::OpusDecoder;
using kvm::audio::PcmPlaybackSink;
using kvm::audio::PcmWriteTimeout;

struct EncoderDeleter {
    void operator()(::OpusEncoder* encoder) const noexcept
    {
        if (encoder != nullptr) {
            opus_encoder_destroy(encoder);
        }
    }
};

class TestEncoder {
  public:
    TestEncoder()
    {
        int opus_code = OPUS_OK;
        encoder_.reset(opus_encoder_create(KvmOpusDecoder::SAMPLE_RATE,
                                           KvmOpusDecoder::CHANNELS,
                                           OPUS_APPLICATION_AUDIO,
                                           &opus_code));
        if (encoder_ == nullptr || opus_code != OPUS_OK) {
            throw std::runtime_error("Failed to create test Opus encoder");
        }
        if (opus_encoder_ctl(encoder_.get(), OPUS_SET_BITRATE(64000))
            != OPUS_OK) {
            throw std::runtime_error("Failed to configure test Opus encoder");
        }
    }

    std::vector<std::uint8_t> encode(const std::vector<std::int16_t>& pcm,
                                     int frames)
    {
        if (pcm.size()
            != static_cast<std::size_t>(frames * KvmOpusDecoder::CHANNELS)) {
            throw std::invalid_argument("Unexpected test PCM size");
        }

        std::array<std::uint8_t, AudioPlayer::MAX_OPUS_PACKET_BYTES> encoded {};
        const int size = opus_encode(
            encoder_.get(), pcm.data(), frames, encoded.data(), encoded.size());
        if (size < 0) {
            throw std::runtime_error(opus_strerror(size));
        }
        return std::vector<std::uint8_t>(encoded.begin(),
                                         encoded.begin() + size);
    }

  private:
    std::unique_ptr<::OpusEncoder, EncoderDeleter> encoder_;
};

std::vector<std::int16_t> makePcm(int frames, int seed)
{
    std::vector<std::int16_t> pcm(
        static_cast<std::size_t>(frames * KvmOpusDecoder::CHANNELS));
    for (int frame = 0; frame < frames; ++frame) {
        const int value = ((frame * 97 + seed * 53) % 24000) - 12000;
        pcm[static_cast<std::size_t>(frame * 2)] =
            static_cast<std::int16_t>(value);
        pcm[static_cast<std::size_t>(frame * 2 + 1)] =
            static_cast<std::int16_t>(-value);
    }
    return pcm;
}

std::vector<std::int16_t> copyPcm(const DecodedPcm& pcm)
{
    if (pcm.data == nullptr) {
        return {};
    }
    const std::size_t samples =
        static_cast<std::size_t>(pcm.frames) * KvmOpusDecoder::CHANNELS;
    return std::vector<std::int16_t>(pcm.data, pcm.data + samples);
}

struct PlaybackState {
    std::vector<std::string> events;
    std::vector<std::int16_t> pcm;
    std::uint32_t frames {0};
    std::uint32_t timeout_ms {0};
    int writes {0};
    int resets {0};
    int closes {0};
    bool throw_timeout {false};
};

class FakePlaybackSink final : public PcmPlaybackSink {
  public:
    explicit FakePlaybackSink(std::shared_ptr<PlaybackState> state)
        : state_(std::move(state))
    {}

    void write(const std::int16_t* pcm,
               std::uint32_t frames,
               std::uint32_t timeout_ms) override
    {
        state_->events.emplace_back("write");
        ++state_->writes;
        state_->frames = frames;
        state_->timeout_ms = timeout_ms;
        if (state_->throw_timeout) {
            throw PcmWriteTimeout();
        }
        const std::size_t samples =
            static_cast<std::size_t>(frames) * KvmOpusDecoder::CHANNELS;
        state_->pcm.assign(pcm, pcm + samples);
    }

    void reset() override
    {
        state_->events.emplace_back("reset");
        ++state_->resets;
    }

    void close() noexcept override
    {
        state_->events.emplace_back("close");
        ++state_->closes;
    }

  private:
    std::shared_ptr<PlaybackState> state_;
};

}  // namespace

TEST_CASE("Audio player decodes and writes one complete Opus packet",
          "[audio][player]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto packet = encoder.encode(makePcm(frames, 1), frames);
    KvmOpusDecoder expected_decoder;
    const auto expected =
        copyPcm(expected_decoder.decode(packet.data(), packet.size()));
    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));

    player.writeOpus(packet.data(), packet.size(), 173, false);

    REQUIRE(state->events == std::vector<std::string> {"write"});
    REQUIRE(state->writes == 1);
    REQUIRE(state->frames == frames);
    REQUIRE(state->timeout_ms == 173);
    REQUIRE(state->pcm == expected);
}

TEST_CASE("Audio player passes the bounded write timeout to playback",
          "[audio][player]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto packet = encoder.encode(makePcm(frames, 2), frames);
    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));

    player.writeOpus(packet.data(), packet.size(), 1, false);
    REQUIRE(state->timeout_ms == 1);

    state->throw_timeout = true;
    REQUIRE_THROWS_AS(
        player.writeOpus(packet.data(), packet.size(), 237, false),
        PcmWriteTimeout);
    REQUIRE(state->timeout_ms == 237);
}

TEST_CASE("Audio player resets decoder and playback before a discontinuity",
          "[audio][player]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto first_packet = encoder.encode(makePcm(frames, 3), frames);
    const auto second_packet = encoder.encode(makePcm(frames, 4), frames);

    KvmOpusDecoder fresh_decoder;
    const auto expected_after_reset = copyPcm(
        fresh_decoder.decode(second_packet.data(), second_packet.size()));

    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));
    player.writeOpus(first_packet.data(), first_packet.size(), 200, false);

    state->events.clear();
    state->pcm.clear();
    player.writeOpus(second_packet.data(), second_packet.size(), 200, true);

    REQUIRE(state->events == std::vector<std::string> {"reset", "write"});
    REQUIRE(state->resets == 1);
    REQUIRE(state->pcm == expected_after_reset);
}

TEST_CASE("Audio player does not write PCM for a corrupt Opus packet",
          "[audio][player]")
{
    const std::array<std::uint8_t, 1> invalid_packet {0xff};
    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));

    try {
        player.writeOpus(
            invalid_packet.data(), invalid_packet.size(), 200, false);
        FAIL("Expected a corrupt Opus packet to fail");
    } catch (const OpusDecodeError& error) {
        REQUIRE(error.stage() == OpusDecodeErrorStage::Decode);
        REQUIRE(error.opusCode() == OPUS_INVALID_PACKET);
    }
    REQUIRE(state->events.empty());
    REQUIRE(state->writes == 0);
    REQUIRE(state->pcm.empty());
}

TEST_CASE("Audio player close is idempotent and rejects later writes",
          "[audio][player]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto packet = encoder.encode(makePcm(frames, 5), frames);
    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));

    REQUIRE_FALSE(player.isClosed());
    player.close();
    REQUIRE(player.isClosed());
    player.close();

    REQUIRE(state->closes == 1);
    REQUIRE(state->events == std::vector<std::string> {"close"});
    REQUIRE_THROWS_AS(
        player.writeOpus(packet.data(), packet.size(), 200, false),
        AudioPlayerStateError);
    REQUIRE(state->writes == 0);
}

TEST_CASE("Audio player validates packet and timeout boundaries",
          "[audio][player]")
{
    STATIC_REQUIRE(AudioPlayer::MAX_OPUS_PACKET_BYTES == 1500);
    STATIC_REQUIRE(AudioPlayer::MAX_TIMEOUT_MS == 5000);
    REQUIRE_THROWS_AS(AudioPlayer(nullptr), std::invalid_argument);

    constexpr int frames = 960;
    TestEncoder encoder;
    auto packet = encoder.encode(makePcm(frames, 6), frames);
    auto state = std::make_shared<PlaybackState>();
    AudioPlayer player(std::make_unique<FakePlaybackSink>(state));

    REQUIRE_THROWS_AS(player.writeOpus(nullptr, packet.size(), 200, false),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(player.writeOpus(packet.data(), 0, 200, false),
                      std::invalid_argument);

    std::vector<std::uint8_t> oversized(AudioPlayer::MAX_OPUS_PACKET_BYTES + 1,
                                        0);
    REQUIRE_THROWS_AS(
        player.writeOpus(oversized.data(), oversized.size(), 200, false),
        std::invalid_argument);
    REQUIRE_THROWS_AS(player.writeOpus(packet.data(), packet.size(), 0, false),
                      std::invalid_argument);
    REQUIRE_THROWS_AS(player.writeOpus(packet.data(),
                                       packet.size(),
                                       AudioPlayer::MAX_TIMEOUT_MS + 1,
                                       false),
                      std::invalid_argument);
    REQUIRE(state->writes == 0);

    const std::size_t original_size = packet.size();
    packet.resize(AudioPlayer::MAX_OPUS_PACKET_BYTES);
    REQUIRE(opus_packet_pad(packet.data(),
                            static_cast<opus_int32>(original_size),
                            static_cast<opus_int32>(packet.size()))
            == OPUS_OK);
    player.writeOpus(
        packet.data(), packet.size(), AudioPlayer::MAX_TIMEOUT_MS, false);
    REQUIRE(state->writes == 1);
    REQUIRE(state->timeout_ms == AudioPlayer::MAX_TIMEOUT_MS);
}
