#include <algorithm>
#include <array>
#include <cstdint>
#include <memory>
#include <stdexcept>
#include <type_traits>
#include <vector>

#include "pcm/opus_decoder.hpp"

#include <catch2/catch_test_macros.hpp>
#include <opus/opus.h>

namespace {

using kvm::audio::DecodedPcm;
using kvm::audio::OpusDecodeError;
using kvm::audio::OpusDecodeErrorStage;
using KvmOpusDecoder = kvm::audio::OpusDecoder;

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

        std::array<std::uint8_t, 1500> encoded {};
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

}  // namespace

TEST_CASE("Opus decoder has the fixed NanoKVM PCM format",
          "[audio][opus-decoder]")
{
    STATIC_REQUIRE(KvmOpusDecoder::SAMPLE_RATE == 48000);
    STATIC_REQUIRE(KvmOpusDecoder::CHANNELS == 2);
    STATIC_REQUIRE(KvmOpusDecoder::MAX_FRAMES_PER_CHANNEL == 5760);
    STATIC_REQUIRE_FALSE(std::is_copy_constructible_v<KvmOpusDecoder>);
    STATIC_REQUIRE_FALSE(std::is_copy_assignable_v<KvmOpusDecoder>);
}

TEST_CASE("Opus decoder decodes one complete 20 ms stereo packet",
          "[audio][opus-decoder]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto packet = encoder.encode(makePcm(frames, 1), frames);

    KvmOpusDecoder decoder;
    const DecodedPcm decoded = decoder.decode(packet.data(), packet.size());

    REQUIRE(decoded.data != nullptr);
    REQUIRE(decoded.frames == frames);
    REQUIRE(copyPcm(decoded).size()
            == static_cast<std::size_t>(frames * KvmOpusDecoder::CHANNELS));
}

TEST_CASE("Opus decoder accepts the maximum 120 ms packet",
          "[audio][opus-decoder]")
{
    constexpr int frames = KvmOpusDecoder::MAX_FRAMES_PER_CHANNEL;
    TestEncoder encoder;
    const auto packet = encoder.encode(makePcm(frames, 2), frames);

    KvmOpusDecoder decoder;
    const DecodedPcm decoded = decoder.decode(packet.data(), packet.size());

    REQUIRE(decoded.data != nullptr);
    REQUIRE(decoded.frames == frames);
}

TEST_CASE("Opus decoder instances own independent state and PCM buffers",
          "[audio][opus-decoder]")
{
    constexpr int frames = 960;
    TestEncoder encoder;
    const auto first_packet = encoder.encode(makePcm(frames, 3), frames);
    const auto second_packet = encoder.encode(makePcm(frames, 4), frames);

    KvmOpusDecoder reset_decoder;
    KvmOpusDecoder continuing_decoder;
    KvmOpusDecoder control_decoder;

    const DecodedPcm reset_first =
        reset_decoder.decode(first_packet.data(), first_packet.size());
    const DecodedPcm continuing_first =
        continuing_decoder.decode(first_packet.data(), first_packet.size());
    control_decoder.decode(first_packet.data(), first_packet.size());

    REQUIRE(reset_first.data != continuing_first.data);

    reset_decoder.reset();
    const auto continuing_second = copyPcm(
        continuing_decoder.decode(second_packet.data(), second_packet.size()));
    const auto control_second = copyPcm(
        control_decoder.decode(second_packet.data(), second_packet.size()));

    REQUIRE(continuing_second == control_second);

    KvmOpusDecoder fresh_decoder;
    const auto reset_second = copyPcm(
        reset_decoder.decode(second_packet.data(), second_packet.size()));
    const auto fresh_second = copyPcm(
        fresh_decoder.decode(second_packet.data(), second_packet.size()));

    REQUIRE(reset_second == fresh_second);
}

TEST_CASE("Opus decoder classifies invalid input and corrupt packets",
          "[audio][opus-decoder]")
{
    KvmOpusDecoder decoder;

    try {
        decoder.decode(nullptr, 0);
        FAIL("Expected empty Opus input to fail");
    } catch (const OpusDecodeError& error) {
        REQUIRE(error.stage() == OpusDecodeErrorStage::InvalidInput);
        REQUIRE(error.opusCode() == OPUS_BAD_ARG);
    }

    const std::array<std::uint8_t, 1> invalid_packet {0xff};
    try {
        decoder.decode(invalid_packet.data(), invalid_packet.size());
        FAIL("Expected corrupt Opus packet to fail");
    } catch (const OpusDecodeError& error) {
        REQUIRE(error.stage() == OpusDecodeErrorStage::Decode);
        REQUIRE(error.opusCode() == OPUS_INVALID_PACKET);
    }
}
