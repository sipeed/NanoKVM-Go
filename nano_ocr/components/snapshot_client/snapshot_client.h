#ifndef SNAPSHOT_CLIENT_H
#define SNAPSHOT_CLIENT_H

#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

inline constexpr int SNAPSHOT_PROTOCOL_VERSION = 1;
inline constexpr int SNAPSHOT_MAX_WAIT_MS = 30000;
inline constexpr std::size_t SNAPSHOT_MAX_HEADER_BYTES = 64 * 1024;
inline constexpr std::size_t SNAPSHOT_MAX_JPEG_BYTES = 32 * 1024 * 1024;

struct LatestFrameCursor {
    std::string sessionId;
    std::uint64_t sequence {0};
};

struct LatestFrame {
    std::string sessionId;
    std::uint64_t sequence {0};
    std::string captureId;
    std::string capturedAt;
    int width {0};
    int height {0};
    std::vector<std::uint8_t> jpeg;
};

enum class LatestFrameStatus {
    Frame,
    NotModified,
    ServerError,
    TransportError,
    ProtocolError,
};

struct ParsedLatestHeader {
    LatestFrameStatus status {LatestFrameStatus::ProtocolError};
    std::string error;
    std::string sessionId;
    std::uint64_t sequence {0};
    std::string captureId;
    std::string capturedAt;
    int width {0};
    int height {0};
    std::size_t size {0};
};

struct LatestFrameResult {
    LatestFrameStatus status {LatestFrameStatus::TransportError};
    std::string error;
    LatestFrame frame;
    std::string currentSessionId;
    std::uint64_t currentSequence {0};
};

std::string buildGetLatestRequest(const LatestFrameCursor& cursor, int waitMs);
ParsedLatestHeader parseGetLatestHeader(const std::string& headerLine);

class SnapshotClient {
  public:
    explicit SnapshotClient(std::string socketPath);

    LatestFrameResult getLatest(const LatestFrameCursor& cursor,
                                int waitMs) const;

  private:
    std::string socketPath_;
};

#endif  // SNAPSHOT_CLIENT_H
