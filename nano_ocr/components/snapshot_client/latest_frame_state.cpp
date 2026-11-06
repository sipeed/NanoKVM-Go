#include "latest_frame_state.h"

const LatestFrameCursor& LatestFrameState::cursor() const { return cursor_; }

LatestFrameEvaluation LatestFrameState::evaluate(
    const LatestFrameResult& result) const
{
    if (result.status == LatestFrameStatus::NotModified) {
        return {
            .disposition = LatestFrameDisposition::NotModified,
            .sessionChanged = !cursor_.sessionId.empty()
                && !result.currentSessionId.empty()
                && result.currentSessionId != cursor_.sessionId,
        };
    }
    if (result.status != LatestFrameStatus::Frame) {
        return {.disposition = LatestFrameDisposition::Unavailable};
    }
    return evaluateFrame(result.frame);
}

bool LatestFrameState::complete(const LatestFrame& frame, bool ocrSucceeded)
{
    if (!ocrSucceeded
        || evaluateFrame(frame).disposition
            != LatestFrameDisposition::Process) {
        return false;
    }

    cursor_.sessionId = frame.sessionId;
    cursor_.sequence = frame.sequence;
    return true;
}

LatestFrameEvaluation LatestFrameState::evaluateFrame(
    const LatestFrame& frame) const
{
    if (frame.sessionId.empty() || frame.sequence == 0) {
        return {.disposition = LatestFrameDisposition::Stale};
    }

    if (cursor_.sessionId.empty()) {
        return {.disposition = LatestFrameDisposition::Process};
    }
    if (frame.sessionId != cursor_.sessionId) {
        return {
            .disposition = LatestFrameDisposition::Process,
            .sessionChanged = true,
        };
    }
    if (frame.sequence <= cursor_.sequence) {
        return {.disposition = LatestFrameDisposition::Stale};
    }

    return {
        .disposition = LatestFrameDisposition::Process,
        .gap = frame.sequence - cursor_.sequence > 1
            ? frame.sequence - cursor_.sequence - 1
            : 0,
    };
}
