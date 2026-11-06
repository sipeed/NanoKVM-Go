#include <array>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <fstream>
#include <iostream>
#include <iterator>
#include <numbers>
#include <stdexcept>
#include <string>
#include <thread>

#include <catch2/catch_test_macros.hpp>
#include <unistd.h>

#include "libkvm/kvm_vision.h"
#include "pcm/opus_encoder.hpp"

namespace {

using KvmOpusEncoder = kvm::audio::OpusEncoder;

constexpr double TONE_HZ = 1000.0;
constexpr double TONE_AMPLITUDE = 12000.0;
constexpr std::uint32_t PROBE_PACKETS = 250;
constexpr std::uint32_t WRITE_TIMEOUT_MS = 500;
constexpr const char* PCM_STATUS_PATH =
    "/proc/asound/UAC2Gadget/pcm0p/sub0/status";

class PlaybackHandle {
  public:
    ~PlaybackHandle() { kvmv_audio_playback_close(&handle_); }

    PlaybackHandle(const PlaybackHandle&) = delete;
    PlaybackHandle& operator=(const PlaybackHandle&) = delete;

    PlaybackHandle() = default;

    kvmv_audio_playback_t** out() noexcept { return &handle_; }
    kvmv_audio_playback_t* get() const noexcept { return handle_; }
    void close() noexcept { kvmv_audio_playback_close(&handle_); }

  private:
    kvmv_audio_playback_t* handle_ {nullptr};
};

std::string readTextFile(const char* path)
{
    std::ifstream input(path);
    if (!input.is_open()) {
        throw std::runtime_error(std::string("Failed to open ") + path);
    }
    return std::string(std::istreambuf_iterator<char>(input),
                       std::istreambuf_iterator<char>());
}

void printPcmStatus(const char* label)
{
    std::cout << "--- PCM status: " << label << " ---\n"
              << readTextFile(PCM_STATUS_PATH) << std::flush;
}

bool printLoadedLibkvmMapping()
{
    std::ifstream maps("/proc/self/maps");
    if (!maps.is_open()) {
        throw std::runtime_error("Failed to open /proc/self/maps");
    }

    bool found = false;
    std::string line;
    while (std::getline(maps, line)) {
        if (line.find("libkvm.so.0") == std::string::npos) {
            continue;
        }
        found = true;
        std::cout << "Loaded libkvm mapping: " << line << std::endl;
    }
    return found;
}

const char* audioErrorName(int result)
{
    switch (result) {
    case KVMV_AUDIO_OK:
        return "ok";
    case KVMV_AUDIO_ERR_INVALID_ARGUMENT:
        return "invalid argument";
    case KVMV_AUDIO_ERR_INVALID_STATE:
        return "invalid state";
    case KVMV_AUDIO_ERR_CODEC:
        return "codec";
    case KVMV_AUDIO_ERR_TIMEOUT:
        return "timeout (host microphone endpoint may be closed)";
    case KVMV_AUDIO_ERR_UNAVAILABLE:
        return "device unavailable";
    case KVMV_AUDIO_ERR_BUSY:
        return "device busy";
    case KVMV_AUDIO_ERR_PERMISSION:
        return "permission";
    case KVMV_AUDIO_ERR_DEVICE_IO:
        return "device I/O (use strace ioctl to distinguish ENOTTY/EIO)";
    case KVMV_AUDIO_ERR_NO_MEMORY:
        return "out of memory";
    case KVMV_AUDIO_ERR_INTERNAL:
        return "internal";
    default:
        return "unknown";
    }
}

void fillSineFrame(
    std::array<std::int16_t,
               KvmOpusEncoder::FRAME_FRAMES * KvmOpusEncoder::CHANNELS>& pcm,
    std::uint64_t frame_offset)
{
    for (std::uint32_t frame = 0; frame < KvmOpusEncoder::FRAME_FRAMES;
         ++frame) {
        const double sample_index = static_cast<double>(frame_offset + frame);
        const double phase = 2.0 * std::numbers::pi * TONE_HZ * sample_index
            / KvmOpusEncoder::SAMPLE_RATE;
        const auto sample = static_cast<std::int16_t>(
            std::lround(TONE_AMPLITUDE * std::sin(phase)));
        const std::size_t base =
            static_cast<std::size_t>(frame) * KvmOpusEncoder::CHANNELS;
        pcm[base] = sample;
        pcm[base + 1] = sample;
    }
}

}  // namespace

TEST_CASE("Static libalsa playback emits a 1 kHz UAC2 probe",
          "[.hardware][audio][playback-probe]")
{
    PlaybackHandle playback;
    const int open_result = kvmv_audio_playback_open(playback.out());
    INFO("playback open result=" << open_result << " ("
                                 << audioErrorName(open_result) << ")");
    REQUIRE(open_result == KVMV_AUDIO_OK);
    REQUIRE(playback.get() != nullptr);
    REQUIRE(printLoadedLibkvmMapping());
    printPcmStatus("opened");

    std::cout << "NanoKVM UAC2 probe opened; pid=" << getpid()
              << ", tone=1000Hz, duration="
              << (PROBE_PACKETS * KvmOpusEncoder::FRAME_MS) / 1000.0 << "s"
              << std::endl;
    std::this_thread::sleep_for(std::chrono::seconds(2));

    KvmOpusEncoder encoder;
    std::array<std::int16_t,
               KvmOpusEncoder::FRAME_FRAMES * KvmOpusEncoder::CHANNELS>
        pcm {};
    std::array<std::uint8_t, KvmOpusEncoder::MAX_PACKET_BYTES> opus {};

    const auto playback_started = std::chrono::steady_clock::now();
    for (std::uint32_t packet = 0; packet < PROBE_PACKETS; ++packet) {
        fillSineFrame(
            pcm,
            static_cast<std::uint64_t>(packet) * KvmOpusEncoder::FRAME_FRAMES);
        const std::uint32_t size = encoder.encode(
            pcm.data(), KvmOpusEncoder::FRAME_FRAMES, opus.data(), opus.size());
        const int write_result =
            kvmv_audio_playback_write_opus(playback.get(),
                                           opus.data(),
                                           size,
                                           WRITE_TIMEOUT_MS,
                                           KVMV_AUDIO_FLAG_NONE);
        INFO("packet=" << packet << ", encoded=" << size
                       << ", write result=" << write_result << " ("
                       << audioErrorName(write_result) << ")");
        REQUIRE(write_result == KVMV_AUDIO_OK);
        if (packet == 0) {
            std::cout << "First Opus/PCM write succeeded" << std::endl;
        }
        if (packet == 50 || packet == 150) {
            printPcmStatus(packet == 50 ? "running sample 1"
                                        : "running sample 2");
        }
    }

    const double elapsed_seconds =
        std::chrono::duration<double>(std::chrono::steady_clock::now()
                                      - playback_started)
            .count();
    std::cout << "UAC2 probe completed: " << PROBE_PACKETS
              << " Opus packets written in " << elapsed_seconds << "s"
              << std::endl;

    playback.close();
    REQUIRE(playback.get() == nullptr);
    printPcmStatus("closed");
}
