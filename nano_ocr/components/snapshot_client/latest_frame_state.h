#ifndef LATEST_FRAME_STATE_H
#define LATEST_FRAME_STATE_H

#include <cstdint>

#include "snapshot_client.h"

enum class LatestFrameDisposition {
    Process,
    NotModified,
    Stale,
    Unavailable,
};

struct LatestFrameEvaluation {
    LatestFrameDisposition disposition {LatestFrameDisposition::Unavailable};
    bool sessionChanged {false};
    std::uint64_t gap {0};
};

class LatestFrameState {
  public:
    const LatestFrameCursor& cursor() const;

    LatestFrameEvaluation evaluate(const LatestFrameResult& result) const;

    // A frame becomes the next request cursor only after its OCR pipeline
    // completed successfully. Failed or stale frames never advance it.
    bool complete(const LatestFrame& frame, bool ocrSucceeded);

  private:
    LatestFrameEvaluation evaluateFrame(const LatestFrame& frame) const;

    LatestFrameCursor cursor_;
};

#endif  // LATEST_FRAME_STATE_H
