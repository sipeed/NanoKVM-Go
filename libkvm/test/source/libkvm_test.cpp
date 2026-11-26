#include <catch2/catch_test_macros.hpp>

#include "libkvm/kvm_vision.h"

TEST_CASE("HDMI control gates image reads", "[library]")
{
    struct HdmiGateReset {
        ~HdmiGateReset() { kvmv_hdmi_control(1); }
    } reset;

    uint8_t sentinel = 0;
    uint8_t* data = &sentinel;
    uint32_t size = 123;

    REQUIRE(kvmv_hdmi_control(0) == 0);
    REQUIRE(kvmv_read_img(0, 0, IMG_H264_TYPE_SPS, 0, &data, &size)
            == IMG_NOT_EXIST);
    REQUIRE(data == nullptr);
    REQUIRE(size == 0);
    REQUIRE(kvmv_hdmi_control(1) == 0);
}

TEST_CASE("Audio reads reject null output parameters", "[library][audio]")
{
    uint32_t size = 123;
    REQUIRE(kvmv_read_audio(nullptr, &size) == -1);
    REQUIRE(size == 0);

    uint8_t sentinel = 0;
    uint8_t* data = &sentinel;
    REQUIRE(kvmv_read_audio(&data, nullptr) == -1);
    REQUIRE(data == nullptr);
}

TEST_CASE("Opus playback ABI validates ownership without hardware",
          "[library][audio][playback]")
{
    REQUIRE(KVMV_AUDIO_FLAG_NONE == 0);
    REQUIRE(KVMV_AUDIO_FLAG_DISCONTINUITY == 1);
    REQUIRE(KVMV_AUDIO_MAX_OPUS_PACKET_BYTES == 1500);
    REQUIRE(KVMV_AUDIO_MAX_TIMEOUT_MS == 5000);

    REQUIRE(kvmv_audio_playback_open(nullptr)
            == KVMV_AUDIO_ERR_INVALID_ARGUMENT);

    auto* sentinel = reinterpret_cast<kvmv_audio_playback_t*>(1);
    REQUIRE(kvmv_audio_playback_open(&sentinel)
            == KVMV_AUDIO_ERR_INVALID_STATE);
    REQUIRE(sentinel == reinterpret_cast<kvmv_audio_playback_t*>(1));

    const uint8_t packet = 0;
    REQUIRE(kvmv_audio_playback_write_opus(nullptr,
                                           &packet,
                                           1,
                                           20,
                                           KVMV_AUDIO_FLAG_NONE)
            == KVMV_AUDIO_ERR_INVALID_STATE);

    kvmv_audio_playback_t* playback = nullptr;
    kvmv_audio_playback_close(nullptr);
    kvmv_audio_playback_close(&playback);
    REQUIRE(playback == nullptr);
}

TEST_CASE("pcm capture produces Opus packets", "[.hardware][audio]")
{
    struct KvmLifecycle {
        KvmLifecycle() { kvmv_init(1); }
        ~KvmLifecycle() { kvmv_deinit(); }
    } lifecycle;

    const int packets_per_second = 1000 / 20;
    const int total_packets = 5 * packets_per_second;

    for (int i = 0; i < total_packets; ++i) {
        uint8_t* data = nullptr;
        uint32_t size = 0;
        REQUIRE(kvmv_read_audio(&data, &size) == 0);
        REQUIRE(data != nullptr);
        REQUIRE(size > 0);
        REQUIRE(size <= 1500);
    }
}
