#include <cstdint>
#include <limits>

#include "snapshot_client.h"

#include <catch2/catch_test_macros.hpp>
#include <nlohmann/json.hpp>

TEST_CASE("get_latest request carries the OCR cursor", "[ocr][socket]")
{
    const std::string request = buildGetLatestRequest(
        {.sessionId = "session-1", .sequence = 42}, 15000);
    const auto parsed = nlohmann::json::parse(request);
    CHECK(parsed.at("version") == 1);
    CHECK(parsed.at("cmd") == "get_latest");
    CHECK(parsed.at("session_id") == "session-1");
    CHECK(parsed.at("after_sequence") == 42);
    CHECK(parsed.at("wait_ms") == 15000);
}

TEST_CASE("get_latest frame header is parsed strictly", "[ocr][socket]")
{
    const auto parsed = parseGetLatestHeader(
        R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":43,"capture_id":"session-1:99","captured_at":"2026-07-14T12:00:00.123+08:00","width":1920,"height":1080,"source":"canonical","size":1234})");
    REQUIRE(parsed.status == LatestFrameStatus::Frame);
    CHECK(parsed.sessionId == "session-1");
    CHECK(parsed.sequence == 43);
    CHECK(parsed.captureId == "session-1:99");
    CHECK(parsed.width == 1920);
    CHECK(parsed.height == 1080);
    CHECK(parsed.size == 1234);
}

TEST_CASE("get_latest accepts the full unsigned OCR sequence range",
          "[ocr][socket]")
{
    const auto parsed = parseGetLatestHeader(
        R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":18446744073709551615,"capture_id":"session-1:99","captured_at":"2026-07-14T12:00:00.123+08:00","width":1920,"height":1080,"source":"canonical","size":1234})");
    REQUIRE(parsed.status == LatestFrameStatus::Frame);
    CHECK(parsed.sequence == std::numeric_limits<std::uint64_t>::max());
}

TEST_CASE("get_latest not_modified contains no JPEG", "[ocr][socket]")
{
    const auto parsed = parseGetLatestHeader(
        R"({"version":1,"ok":true,"ret_code":0,"message":"not_modified","not_modified":true,"session_id":"session-1","sequence":43,"width":0,"height":0,"size":0})");
    REQUIRE(parsed.status == LatestFrameStatus::NotModified);
    CHECK(parsed.sessionId == "session-1");
    CHECK(parsed.sequence == 43);
    CHECK(parsed.size == 0);
}

TEST_CASE("get_latest rejects malformed canonical metadata", "[ocr][socket]")
{
    SECTION("missing capture id")
    {
        const auto parsed = parseGetLatestHeader(
            R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":1,"captured_at":"2026-07-14T12:00:00+08:00","width":1920,"height":1080,"source":"canonical","size":1234})");
        CHECK(parsed.status == LatestFrameStatus::ProtocolError);
    }

    SECTION("wrong source")
    {
        const auto parsed = parseGetLatestHeader(
            R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":1,"capture_id":"session-1:1","captured_at":"2026-07-14T12:00:00+08:00","width":1920,"height":1080,"source":"mcp","size":1234})");
        CHECK(parsed.status == LatestFrameStatus::ProtocolError);
    }

    SECTION("oversized JPEG")
    {
        const auto parsed = parseGetLatestHeader(
            R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":1,"capture_id":"session-1:1","captured_at":"2026-07-14T12:00:00+08:00","width":1920,"height":1080,"source":"canonical","size":33554433})");
        CHECK(parsed.status == LatestFrameStatus::ProtocolError);
    }
}

TEST_CASE("get_latest accepts the maximum declared JPEG size", "[ocr][socket]")
{
    const std::string header =
        R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":1,"capture_id":"session-1:1","captured_at":"2026-07-14T12:00:00+08:00","width":1920,"height":1080,"source":"canonical","size":)"
        + std::to_string(SNAPSHOT_MAX_JPEG_BYTES) + "}";

    const auto parsed = parseGetLatestHeader(header);

    REQUIRE(parsed.status == LatestFrameStatus::Frame);
    CHECK(parsed.size == SNAPSHOT_MAX_JPEG_BYTES);
}
