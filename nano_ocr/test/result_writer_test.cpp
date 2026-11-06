#include "result_writer.h"

#include <catch2/catch_test_macros.hpp>
#include <nlohmann/json.hpp>

TEST_CASE("socket observation uses explicit metadata",
          "[ocr][result]")
{
    const OCRFrameMetadata metadata {
        .capturedAt = "2026-07-14T12:34:56.123+08:00",
        .captureId = "session-1:99",
        .sessionId = "session-1",
        .sequence = 7,
    };
    const std::string output =
        formatResultsToJson(metadata, 1920, 1080, {}, {});
    const auto parsed = nlohmann::json::parse(output);

    CHECK(parsed.at("captured_at") == metadata.capturedAt);
    CHECK(parsed.at("capture_id") == metadata.captureId);
    CHECK(parsed.at("screen").size() == 2);
    CHECK(parsed.at("screen").at("width") == 1920);
    CHECK(parsed.at("screen").at("height") == 1080);
}
