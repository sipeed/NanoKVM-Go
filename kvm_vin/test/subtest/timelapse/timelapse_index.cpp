#include <cstdlib>
#include <ctime>
#include <sstream>
#include <string>

#include "timelapse_index.hpp"

#include <catch2/catch_test_macros.hpp>
#include <nlohmann/json.hpp>

#include "timelapse_ts_muxer.hpp"

namespace {

class ScopedTimezone {
  public:
    explicit ScopedTimezone(const char* timezone)
    {
        const char* current = std::getenv("TZ");
        if (current != nullptr) {
            _had_timezone = true;
            _timezone = current;
        }
        REQUIRE(::setenv("TZ", timezone, 1) == 0);
        ::tzset();
    }

    ~ScopedTimezone()
    {
        if (_had_timezone) {
            (void)::setenv("TZ", _timezone.c_str(), 1);
        } else {
            (void)::unsetenv("TZ");
        }
        ::tzset();
    }

    ScopedTimezone(const ScopedTimezone&) = delete;
    ScopedTimezone& operator=(const ScopedTimezone&) = delete;

  private:
    bool _had_timezone {false};
    std::string _timezone;
};

}  // namespace

TEST_CASE("Timelapse index writes canonical capture metadata",
          "[timelapse][index]")
{
    const TimelapseIndexEntry entry {
        .frame = 7,
        .pts = 21000,
        .duration = 3000,
        .captured_at = "2026-07-14T05:43:21.123456789Z",
        .capture_id = "session-1:42",
        .width = 1920,
        .height = 1080,
        .h264_bytes = 4567,
        .ts_offset = 1880,
        .segment_id = 3,
    };

    const std::string line = serialize_timelapse_index_entry(entry);
    REQUIRE_FALSE(line.empty());
    CHECK(line.back() == '\n');

    const auto parsed = nlohmann::json::parse(line);
    CHECK(parsed.at("frame") == 7);
    CHECK(parsed.at("pts") == 21000);
    CHECK(parsed.at("duration") == 3000);
    CHECK(parsed.at("captured_at") == entry.captured_at);
    CHECK(parsed.at("capture_id") == entry.capture_id);
    CHECK_FALSE(parsed.contains("jpeg_time"));
    CHECK_FALSE(parsed.contains("jpeg"));
    CHECK(parsed.at("width") == 1920);
    CHECK(parsed.at("height") == 1080);
    CHECK(parsed.at("h264_bytes") == 4567);
    CHECK(parsed.at("ts_offset") == 1880);
    CHECK(parsed.at("segment_id") == 3);
}

TEST_CASE("Timelapse output date follows captured_at in the local timezone",
          "[timelapse][index]")
{
    ScopedTimezone timezone("UTC-8");

    CHECK(
        timelapse_local_date_from_captured_at("2026-07-14T23:30:00.123456789Z")
        == "2026-07-15");
    CHECK(timelapse_local_date_from_captured_at("2026-07-15T01:30:00+02:00")
          == "2026-07-15");
    CHECK(timelapse_local_date_from_captured_at("not-a-timestamp").empty());
}

TEST_CASE("Timelapse index escapes canonical capture metadata",
          "[timelapse][index]")
{
    const TimelapseIndexEntry entry {
        .captured_at = "2026-07-14T05:43:21Z\n",
        .capture_id = "session\"\\:1",
    };

    const auto parsed =
        nlohmann::json::parse(serialize_timelapse_index_entry(entry));
    CHECK(parsed.at("captured_at") == entry.captured_at);
    CHECK(parsed.at("capture_id") == entry.capture_id);
    CHECK_FALSE(parsed.contains("jpeg_time"));
    CHECK_FALSE(parsed.contains("jpeg"));
}

TEST_CASE("Timelapse starts an empty recording at zero", "[timelapse][index]")
{
    std::istringstream input;
    TimelapseIndexPosition position;
    REQUIRE(read_timelapse_next_position(input, position));
    CHECK(position.frame == 0);
    CHECK(position.pts == 0);
}

TEST_CASE("Timelapse appends 5 fps frames after a legacy 30 fps recording",
          "[timelapse][index]")
{
    std::string index = serialize_timelapse_index_entry(TimelapseIndexEntry {
                            .frame = 0,
                            .pts = 0,
                            .duration = 3000,
                        })
        + serialize_timelapse_index_entry(TimelapseIndexEntry {
            .frame = 1,
            .pts = 3000,
            .duration = 3000,
        });

    std::istringstream legacy_input(index);
    TimelapseIndexPosition position;
    REQUIRE(read_timelapse_next_position(legacy_input, position));
    CHECK(position.frame == 2);
    CHECK(position.pts == 6000);

    index += serialize_timelapse_index_entry(TimelapseIndexEntry {
        .frame = position.frame,
        .pts = position.pts,
        .duration = TimelapseTsMuxer::FRAME_DURATION_90K,
    });

    std::istringstream restarted_input(index);
    REQUIRE(read_timelapse_next_position(restarted_input, position));
    CHECK(position.frame == 3);
    CHECK(position.pts == 24000);
}

TEST_CASE("Timelapse ignores an incomplete index tail when resuming timing",
          "[timelapse][index]")
{
    std::istringstream input(
        "\n{\"frame\":4,\"pts\":72000,\"duration\":18000}\n"
        "{\"frame\":5,\"pts\":90000,\"duration\":");
    TimelapseIndexPosition position;
    REQUIRE(read_timelapse_next_position(input, position));
    CHECK(position.frame == 5);
    CHECK(position.pts == 90000);
}

TEST_CASE("Timelapse rejects overflowing index timing", "[timelapse][index]")
{
    for (const char* entry : {
             R"({"frame":18446744073709551615,"pts":0,"duration":18000})",
             R"({"frame":0,"pts":9223372036854775807,"duration":18000})",
             R"({"frame":0,"pts":0,"duration":18446744073709551615})",
             R"({"frame":0,"pts":0,"duration":0})",
         }) {
        std::istringstream input(entry);
        TimelapseIndexPosition position;
        REQUIRE_FALSE(read_timelapse_next_position(input, position));
    }
}
