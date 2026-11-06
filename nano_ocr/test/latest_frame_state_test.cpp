#include "latest_frame_state.h"

#include <catch2/catch_test_macros.hpp>

namespace {

LatestFrameResult frameResult(const char* sessionId, std::uint64_t sequence)
{
    return {
        .status = LatestFrameStatus::Frame,
        .frame =
            {
                    .sessionId = sessionId,
                    .sequence = sequence,
                    },
    };
}

void commit(LatestFrameState& state,
            const char* sessionId,
            std::uint64_t sequence)
{
    const LatestFrameResult result = frameResult(sessionId, sequence);
    REQUIRE(state.evaluate(result).disposition
            == LatestFrameDisposition::Process);
    REQUIRE(state.complete(result.frame, true));
}

}  // namespace

TEST_CASE("same-session OCR frames advance in sequence", "[ocr][socket][state]")
{
    LatestFrameState state;

    const LatestFrameResult first = frameResult("session-1", 1);
    const LatestFrameEvaluation firstEvaluation = state.evaluate(first);
    CHECK(firstEvaluation.disposition == LatestFrameDisposition::Process);
    CHECK_FALSE(firstEvaluation.sessionChanged);
    CHECK(firstEvaluation.gap == 0);
    REQUIRE(state.complete(first.frame, true));

    const LatestFrameResult second = frameResult("session-1", 2);
    const LatestFrameEvaluation secondEvaluation = state.evaluate(second);
    CHECK(secondEvaluation.disposition == LatestFrameDisposition::Process);
    CHECK_FALSE(secondEvaluation.sessionChanged);
    CHECK(secondEvaluation.gap == 0);
    REQUIRE(state.complete(second.frame, true));

    CHECK(state.cursor().sessionId == "session-1");
    CHECK(state.cursor().sequence == 2);
}

TEST_CASE("slow OCR observes a latest-wins sequence gap",
          "[ocr][socket][state]")
{
    LatestFrameState state;
    commit(state, "session-1", 2);

    const LatestFrameResult latest = frameResult("session-1", 7);
    const LatestFrameEvaluation evaluation = state.evaluate(latest);
    CHECK(evaluation.disposition == LatestFrameDisposition::Process);
    CHECK_FALSE(evaluation.sessionChanged);
    CHECK(evaluation.gap == 4);

    REQUIRE(state.complete(latest.frame, true));
    CHECK(state.cursor().sequence == 7);
}

TEST_CASE("a new kvm_vin session may restart its OCR sequence",
          "[ocr][socket][state]")
{
    LatestFrameState state;
    commit(state, "session-old", 50);

    const LatestFrameResult restarted = frameResult("session-new", 1);
    const LatestFrameEvaluation evaluation = state.evaluate(restarted);
    CHECK(evaluation.disposition == LatestFrameDisposition::Process);
    CHECK(evaluation.sessionChanged);
    CHECK(evaluation.gap == 0);

    REQUIRE(state.complete(restarted.frame, true));
    CHECK(state.cursor().sessionId == "session-new");
    CHECK(state.cursor().sequence == 1);
}

TEST_CASE("stale and non-increasing OCR frames never replace the cursor",
          "[ocr][socket][state]")
{
    LatestFrameState state;
    commit(state, "session-1", 5);

    for (const std::uint64_t sequence : {5ULL, 4ULL}) {
        const LatestFrameResult stale = frameResult("session-1", sequence);
        CHECK(state.evaluate(stale).disposition
              == LatestFrameDisposition::Stale);
        CHECK_FALSE(state.complete(stale.frame, true));
        CHECK(state.cursor().sessionId == "session-1");
        CHECK(state.cursor().sequence == 5);
    }
}

TEST_CASE("not_modified reports do not advance or reset the committed cursor",
          "[ocr][socket][state]")
{
    LatestFrameState state;
    commit(state, "session-1", 5);

    LatestFrameResult unchanged {
        .status = LatestFrameStatus::NotModified,
        .currentSessionId = "session-1",
        .currentSequence = 99,
    };
    LatestFrameEvaluation evaluation = state.evaluate(unchanged);
    CHECK(evaluation.disposition == LatestFrameDisposition::NotModified);
    CHECK_FALSE(evaluation.sessionChanged);
    CHECK(state.cursor().sessionId == "session-1");
    CHECK(state.cursor().sequence == 5);

    unchanged.currentSessionId = "session-2";
    unchanged.currentSequence = 0;
    evaluation = state.evaluate(unchanged);
    CHECK(evaluation.disposition == LatestFrameDisposition::NotModified);
    CHECK(evaluation.sessionChanged);
    CHECK(state.cursor().sessionId == "session-1");
    CHECK(state.cursor().sequence == 5);
}

TEST_CASE("only a successful OCR pipeline commits the frame cursor",
          "[ocr][socket][state]")
{
    LatestFrameState state;
    commit(state, "session-1", 1);

    const LatestFrameResult next = frameResult("session-1", 2);
    REQUIRE(state.evaluate(next).disposition
            == LatestFrameDisposition::Process);
    CHECK_FALSE(state.complete(next.frame, false));
    CHECK(state.cursor().sequence == 1);

    REQUIRE(state.complete(next.frame, true));
    CHECK(state.cursor().sequence == 2);
}
