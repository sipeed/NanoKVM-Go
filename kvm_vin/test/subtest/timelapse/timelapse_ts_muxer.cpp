#include "timelapse_ts_muxer.hpp"

#include <catch2/catch_test_macros.hpp>

#include <algorithm>
#include <array>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <string>

#include <unistd.h>

namespace {

constexpr size_t kTsPacketSize = 188;

std::filesystem::path temp_ts_path(const std::string& suffix)
{
    return std::filesystem::temp_directory_path() /
           ("timelapse_muxer_" + std::to_string(::getpid()) + "_" + suffix + ".ts");
}

void write_bytes(const std::filesystem::path& path, size_t size)
{
    std::ofstream output(path, std::ios::binary | std::ios::trunc);
    REQUIRE(output);
    std::array<char, kTsPacketSize> packet {};
    packet[0] = 0x47;
    while (size > 0) {
        const size_t chunk = std::min(size, packet.size());
        output.write(packet.data(), static_cast<std::streamsize>(chunk));
        size -= chunk;
    }
    REQUIRE(output);
}

const std::array<uint8_t, 36>& sample_h264()
{
    static const std::array<uint8_t, 36> data {
        0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1e, 0xab, 0x40, 0x50, 0x1e,
        0xd0, 0x08, 0x9f, 0x97, 0x01, 0x6e, 0x40, 0x00, 0x00, 0x00, 0x01, 0x68,
        0xce, 0x3c, 0x80, 0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x21, 0xa0,
    };
    return data;
}

void remove_if_exists(const std::filesystem::path& path)
{
    std::error_code ec;
    (void)std::filesystem::remove(path, ec);
}

}  // namespace

TEST_CASE("TimelapseTsMuxer truncates partial TS tail before append", "[timelapse][ts]")
{
    const auto path = temp_ts_path("partial_tail");
    remove_if_exists(path);
    write_bytes(path, 156);

    TimelapseTsMuxer::AppendResult result;
    std::string error;
    const auto& h264 = sample_h264();
    const bool ok = TimelapseTsMuxer::append_h264_frame(path.string(),
                                                        h264.data(),
                                                        h264.size(),
                                                        1920,
                                                        1080,
                                                        0,
                                                        result,
                                                        error);
    INFO(error);
    REQUIRE(ok);

    const auto size = std::filesystem::file_size(path);
    CHECK(result.ts_offset == 0);
    CHECK(result.stable_offset == size);
    CHECK(result.segment_id == 0);
    CHECK(size > 0);
    CHECK((size % kTsPacketSize) == 0);
    remove_if_exists(path);
}

TEST_CASE("TimelapseTsMuxer appends at aligned TS boundary", "[timelapse][ts]")
{
    const auto path = temp_ts_path("aligned_append");
    remove_if_exists(path);
    write_bytes(path, kTsPacketSize);

    TimelapseTsMuxer::AppendResult result;
    std::string error;
    const auto& h264 = sample_h264();
    const bool ok = TimelapseTsMuxer::append_h264_frame(
        path.string(),
        h264.data(),
        h264.size(),
        1920,
        1080,
        TimelapseTsMuxer::FRAME_DURATION_90K,
        result,
        error);
    INFO(error);
    REQUIRE(ok);

    const auto size = std::filesystem::file_size(path);
    CHECK(result.ts_offset == kTsPacketSize);
    CHECK(result.stable_offset == size);
    CHECK(result.segment_id == 0);
    CHECK(size > kTsPacketSize);
    CHECK((size % kTsPacketSize) == 0);
    remove_if_exists(path);
}
