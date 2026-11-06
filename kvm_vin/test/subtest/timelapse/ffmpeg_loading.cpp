#include "timelapse_ts_muxer.hpp"

#include <catch2/catch_test_macros.hpp>

#include <array>
#include <atomic>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <iterator>
#include <string>
#include <thread>

#include <link.h>

namespace {

std::atomic<bool> library_unavailable {false};
std::atomic<unsigned int> load_attempts {0};

unsigned int loaded_ffmpeg_libraries()
{
    unsigned int loaded = 0;
    ::dl_iterate_phdr(
        [](dl_phdr_info* info, size_t, void* data) {
            auto& flags = *static_cast<unsigned int*>(data);
            if (std::strstr(info->dlpi_name, "libavformat.so.") != nullptr) flags |= 1;
            if (std::strstr(info->dlpi_name, "libavcodec.so.") != nullptr) flags |= 2;
            if (std::strstr(info->dlpi_name, "libavutil.so.") != nullptr) flags |= 4;
            return 0;
        },
        &loaded);
    return loaded;
}

struct TemporaryDirectory {
    std::filesystem::path path;
    ~TemporaryDirectory()
    {
        std::error_code error;
        std::filesystem::remove_all(path, error);
    }
};

}  // namespace

extern "C" void* __real_dlopen(const char*, int);

extern "C" void* __wrap_dlopen(const char* name, int flags)
{
    if (name != nullptr && std::strncmp(name, "libavformat.so.", 15) == 0) {
        ++load_attempts;
        if (library_unavailable) {
            return __real_dlopen("libnanokvm-test-unavailable-ffmpeg.so", flags);
        }
    }
    return __real_dlopen(name, flags);
}

TEST_CASE("System FFmpeg loads once on valid recording and retries after failure",
          "[timelapse][ffmpeg]")
{
    REQUIRE(loaded_ffmpeg_libraries() == 0);

    auto pattern = (std::filesystem::temp_directory_path() /
                    "kvm-ffmpeg-loading-XXXXXX").string();
    REQUIRE(::mkdtemp(pattern.data()) != nullptr);
    const TemporaryDirectory temporary {pattern};
    const auto existing_path = temporary.path / "existing.ts";
    const std::string original = "partial TS output must survive a loader failure";
    {
        std::ofstream output(existing_path, std::ios::binary);
        output << original;
        REQUIRE(output.good());
    }

    constexpr std::array<uint8_t, 36> frame {
        0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1e, 0xab, 0x40, 0x50, 0x1e,
        0xd0, 0x08, 0x9f, 0x97, 0x01, 0x6e, 0x40, 0x00, 0x00, 0x00, 0x01, 0x68,
        0xce, 0x3c, 0x80, 0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x21, 0xa0,
    };

    TimelapseTsMuxer::AppendResult result;
    std::string error;
    CHECK_FALSE(TimelapseTsMuxer::append_h264_frame(
        existing_path.string(), nullptr, 0, 1920, 1080, 0, result, error));
    CHECK(load_attempts == 0);
    CHECK(loaded_ffmpeg_libraries() == 0);

    library_unavailable = true;
    CHECK_FALSE(TimelapseTsMuxer::append_h264_frame(existing_path.string(),
        frame.data(), frame.size(), 1920, 1080, 0, result, error));
    CHECK(error.find("load system FFmpeg failed:") != std::string::npos);
    CHECK(load_attempts == 1);
    CHECK(loaded_ffmpeg_libraries() == 0);
    {
        std::ifstream input(existing_path, std::ios::binary);
        const std::string contents((std::istreambuf_iterator<char>(input)), {});
        CHECK(contents == original);
    }

    library_unavailable = false;
    struct RecordingResult {
        bool ok {false};
        std::string error;
        TimelapseTsMuxer::AppendResult offsets;
    };
    std::array<RecordingResult, 4> recordings;
    std::array<std::thread, 4> threads;
    for (size_t i = 0; i < threads.size(); ++i) {
        threads[i] = std::thread([&, i] {
            const auto path = temporary.path / (std::to_string(i) + ".ts");
            auto& recording = recordings[i];
            recording.ok = TimelapseTsMuxer::append_h264_frame(path.string(),
                frame.data(), frame.size(), 1920, 1080, 0,
                recording.offsets, recording.error);
        });
    }
    for (auto& thread : threads) thread.join();
    for (const auto& recording : recordings) {
        INFO(recording.error);
        CHECK(recording.ok);
        CHECK(recording.offsets.stable_offset > 0);
        CHECK(recording.offsets.stable_offset % 188 == 0);
    }
    CHECK(load_attempts == 2);
    CHECK(loaded_ffmpeg_libraries() == 7);
}
