#include <array>
#include <cstdint>
#include <limits>
#include <type_traits>
#include <vector>

#include "pcm/opus_encoder.hpp"

#include <catch2/catch_test_macros.hpp>
#include <opus/opus.h>

#include "pcm/opus_decoder.hpp"

namespace {

using kvm::audio::OpusEncodeError;
using kvm::audio::OpusEncodeErrorStage;
using KvmOpusDecoder = kvm::audio::OpusDecoder;
using KvmOpusEncoder = kvm::audio::OpusEncoder;

std::vector<std::int16_t> makePcm(int seed)
{
    std::vector<std::int16_t> pcm(static_cast<std::size_t>(
        KvmOpusEncoder::FRAME_FRAMES * KvmOpusEncoder::CHANNELS));
    for (int frame = 0; frame < KvmOpusEncoder::FRAME_FRAMES; ++frame) {
        const int value = ((frame * 97 + seed * 53) % 24000) - 12000;
        pcm[static_cast<std::size_t>(frame * 2)] =
            static_cast<std::int16_t>(value);
        pcm[static_cast<std::size_t>(frame * 2 + 1)] =
            static_cast<std::int16_t>(-value);
    }
    return pcm;
}

}  // namespace

TEST_CASE("Opus encoder has the fixed NanoKVM capture format",
          "[audio][opus-encoder]")
{
    STATIC_REQUIRE(KvmOpusEncoder::SAMPLE_RATE == 48000);
    STATIC_REQUIRE(KvmOpusEncoder::CHANNELS == 2);
    STATIC_REQUIRE(KvmOpusEncoder::FRAME_MS == 20);
    STATIC_REQUIRE(KvmOpusEncoder::FRAME_FRAMES == 960);
    STATIC_REQUIRE(KvmOpusEncoder::BITRATE == 128000);
    STATIC_REQUIRE(KvmOpusEncoder::COMPLEXITY == 4);
    STATIC_REQUIRE(KvmOpusEncoder::MAX_PACKET_BYTES == 1500);
    STATIC_REQUIRE_FALSE(std::is_copy_constructible_v<KvmOpusEncoder>);
    STATIC_REQUIRE_FALSE(std::is_copy_assignable_v<KvmOpusEncoder>);
}

TEST_CASE("Opus encoder produces complete 20 ms packets",
          "[audio][opus-encoder]")
{
    KvmOpusEncoder encoder;
    KvmOpusDecoder decoder;
    std::array<std::uint8_t, KvmOpusEncoder::MAX_PACKET_BYTES> output {};

    for (int seed : {1, 2}) {
        const auto pcm = makePcm(seed);
        const std::uint32_t size = encoder.encode(pcm.data(),
                                                  KvmOpusEncoder::FRAME_FRAMES,
                                                  output.data(),
                                                  output.size());

        REQUIRE(size > 0);
        REQUIRE(size <= output.size());
        const auto decoded = decoder.decode(output.data(), size);
        REQUIRE(decoded.data != nullptr);
        REQUIRE(decoded.frames == KvmOpusEncoder::FRAME_FRAMES);
    }
}

TEST_CASE("Opus encoder classifies invalid buffers", "[audio][opus-encoder]")
{
    KvmOpusEncoder encoder;
    const auto pcm = makePcm(3);
    std::array<std::uint8_t, KvmOpusEncoder::MAX_PACKET_BYTES> output {};

    const auto require_invalid_input = [&](const std::int16_t* input,
                                           std::uint32_t frames,
                                           std::uint8_t* encoded,
                                           std::size_t capacity) {
        try {
            encoder.encode(input, frames, encoded, capacity);
            FAIL("Expected invalid Opus encoder input to fail");
        } catch (const OpusEncodeError& error) {
            REQUIRE(error.stage() == OpusEncodeErrorStage::InvalidInput);
            REQUIRE(error.opusCode() == OPUS_BAD_ARG);
        }
    };

    require_invalid_input(
        nullptr, KvmOpusEncoder::FRAME_FRAMES, output.data(), output.size());
    require_invalid_input(pcm.data(), 0, output.data(), output.size());
    require_invalid_input(
        pcm.data(), KvmOpusEncoder::FRAME_FRAMES, nullptr, output.size());
    require_invalid_input(
        pcm.data(), KvmOpusEncoder::FRAME_FRAMES, output.data(), 0);
    require_invalid_input(pcm.data(),
                          std::numeric_limits<std::uint32_t>::max(),
                          output.data(),
                          output.size());
    require_invalid_input(
        pcm.data(),
        KvmOpusEncoder::FRAME_FRAMES,
        output.data(),
        static_cast<std::size_t>(std::numeric_limits<std::int32_t>::max()) + 1);
}

TEST_CASE("Opus encoder reports unsupported frame sizes as codec failures",
          "[audio][opus-encoder]")
{
    KvmOpusEncoder encoder;
    const auto pcm = makePcm(4);
    std::array<std::uint8_t, KvmOpusEncoder::MAX_PACKET_BYTES> output {};

    try {
        encoder.encode(pcm.data(), 1, output.data(), output.size());
        FAIL("Expected an unsupported Opus frame size to fail");
    } catch (const OpusEncodeError& error) {
        REQUIRE(error.stage() == OpusEncodeErrorStage::Encode);
        REQUIRE(error.opusCode() == OPUS_BAD_ARG);
    }
}
