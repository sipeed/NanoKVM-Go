#include <chrono>
#include <cstdint>
#include <future>
#include <memory>
#include <string>
#include <thread>
#include <utility>

#include "ocr_latest_frame_store.hpp"

#include <catch2/catch_test_macros.hpp>

namespace {

using namespace std::chrono_literals;

CaptureResponse make_response(std::string session_id,
                              std::string capture_id,
                              std::uint8_t marker)
{
    auto frame = std::make_shared<AxeraUtils::SnapshotResult>();
    frame->width = 1920;
    frame->height = 1080;
    frame->data = {0xff, 0xd8, marker, 0xff, 0xd9};

    return CaptureResponse {
        .ret_code = CaptureCoordinator::RESULT_OK,
        .frame = std::move(frame),
        .session_id = std::move(session_id),
        .capture_id = std::move(capture_id),
        .captured_at = "2026-07-14T12:00:00.000000000Z",
    };
}

}  // namespace

TEST_CASE("OCR latest frame returns a published frame immediately",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    const CaptureResponse response = make_response("session-a", "capture-1", 1);

    REQUIRE(store.publish(response));
    const auto result = store.get_latest("", 0, 1s);

    REQUIRE(result.status == OCRLatestFrameStatus::Frame);
    REQUIRE(result.latest);
    CHECK(result.latest->frame == response.frame);
    CHECK(result.latest->session_id == "session-a");
    CHECK(result.latest->capture_id == "capture-1");
    CHECK(result.latest->sequence == 1);
    CHECK(result.session_id == "session-a");
    CHECK(result.sequence == 1);
}

TEST_CASE("OCR latest frame reports not modified after its wait timeout",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));

    const auto started = std::chrono::steady_clock::now();
    const auto result = store.get_latest("session-a", 1, 40ms);
    const auto elapsed = std::chrono::steady_clock::now() - started;

    REQUIRE(result.status == OCRLatestFrameStatus::NotModified);
    CHECK_FALSE(result.latest);
    CHECK(result.session_id == "session-a");
    CHECK(result.sequence == 1);
    CHECK(elapsed >= 20ms);
}

TEST_CASE("OCR latest frame publish wakes a waiting consumer",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));

    auto waiting = std::async(std::launch::async, [&store]() {
        return store.get_latest("session-a", 1, 2s);
    });
    REQUIRE(waiting.wait_for(20ms) == std::future_status::timeout);

    REQUIRE(store.publish(make_response("session-a", "capture-2", 2)));
    REQUIRE(waiting.wait_for(500ms) == std::future_status::ready);

    const auto result = waiting.get();
    REQUIRE(result.status == OCRLatestFrameStatus::Frame);
    REQUIRE(result.latest);
    CHECK(result.latest->capture_id == "capture-2");
    CHECK(result.latest->sequence == 2);
}

TEST_CASE("OCR latest frame uses a capacity-one latest-wins store",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));
    REQUIRE(store.publish(make_response("session-a", "capture-2", 2)));
    REQUIRE(store.publish(make_response("session-a", "capture-3", 3)));

    const auto result = store.get_latest("session-a", 0, 0ms);

    REQUIRE(result.status == OCRLatestFrameStatus::Frame);
    REQUIRE(result.latest);
    CHECK(result.latest->capture_id == "capture-3");
    CHECK(result.latest->sequence == 3);
    REQUIRE(result.latest->frame);
    CHECK(result.latest->frame->data[2] == 3);
}

TEST_CASE(
    "Slow OCR consumers observe a real sequence gap and only the latest frame",
    "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));

    const auto first = store.get_latest("", 0, 0ms);
    REQUIRE(first.status == OCRLatestFrameStatus::Frame);
    REQUIRE(first.latest);
    REQUIRE(first.latest->sequence == 1);

    for (std::uint8_t marker = 2; marker <= 10; ++marker) {
        REQUIRE(store.publish(make_response(
            "session-a", "capture-" + std::to_string(marker), marker)));
    }

    const auto latest =
        store.get_latest("session-a", first.latest->sequence, 0ms);
    REQUIRE(latest.status == OCRLatestFrameStatus::Frame);
    REQUIRE(latest.latest);
    CHECK(latest.latest->sequence == 10);
    CHECK(latest.latest->sequence - first.latest->sequence == 9);
    CHECK(latest.latest->capture_id == "capture-10");
    REQUIRE(latest.latest->frame);
    CHECK(latest.latest->frame->data[2] == 10);
}

TEST_CASE("OCR sequence advances only for successful explicit publishes",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));

    CaptureResponse failed = make_response("session-a", "capture-2", 2);
    failed.ret_code = CaptureCoordinator::RESULT_BUSY;
    CHECK_FALSE(store.publish(failed));

    const auto unchanged = store.get_latest("session-a", 1, 0ms);
    CHECK(unchanged.status == OCRLatestFrameStatus::NotModified);
    CHECK(unchanged.sequence == 1);

    REQUIRE(store.publish(make_response("session-a", "capture-3", 3)));
    const auto advanced = store.get_latest("session-a", 1, 0ms);
    REQUIRE(advanced.status == OCRLatestFrameStatus::Frame);
    REQUIRE(advanced.latest);
    CHECK(advanced.latest->sequence == 2);
}

TEST_CASE("OCR latest frame session change resets its sequence",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;
    REQUIRE(store.publish(make_response("session-a", "capture-a1", 1)));
    REQUIRE(store.publish(make_response("session-a", "capture-a2", 2)));
    REQUIRE(store.publish(make_response("session-b", "capture-b1", 3)));

    const auto started = std::chrono::steady_clock::now();
    const auto result = store.get_latest("session-a", 999, 1s);
    const auto elapsed = std::chrono::steady_clock::now() - started;

    REQUIRE(result.status == OCRLatestFrameStatus::Frame);
    REQUIRE(result.latest);
    CHECK(result.latest->session_id == "session-b");
    CHECK(result.latest->capture_id == "capture-b1");
    CHECK(result.latest->sequence == 1);
    CHECK(elapsed < 100ms);
}

TEST_CASE("OCR latest frame waits when no frame has been published",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;

    auto waiting = std::async(std::launch::async, [&store]() {
        return store.get_latest("old-session", 99, 2s);
    });
    REQUIRE(waiting.wait_for(20ms) == std::future_status::timeout);

    REQUIRE(store.publish(make_response("session-a", "capture-1", 1)));
    REQUIRE(waiting.wait_for(500ms) == std::future_status::ready);

    const auto result = waiting.get();
    REQUIRE(result.status == OCRLatestFrameStatus::Frame);
    REQUIRE(result.latest);
    CHECK(result.latest->session_id == "session-a");
    CHECK(result.latest->sequence == 1);
}

TEST_CASE("OCR latest frame shutdown wakes consumers and rejects publishes",
          "[capture][ocr-latest]")
{
    OCRLatestFrameStore store;

    auto waiting = std::async(std::launch::async, [&store]() {
        return store.get_latest("session-a", 1, 5s);
    });
    REQUIRE(waiting.wait_for(20ms) == std::future_status::timeout);

    store.stop();
    REQUIRE(waiting.wait_for(500ms) == std::future_status::ready);

    const auto result = waiting.get();
    CHECK(result.status == OCRLatestFrameStatus::Stopped);
    CHECK_FALSE(result.latest);
    CHECK(store.is_stopped());
    CHECK_FALSE(store.publish(make_response("session-a", "capture-1", 1)));

    store.stop();
}
