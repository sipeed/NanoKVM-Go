#include <algorithm>
#include <cerrno>
#include <cstring>
#include <limits>
#include <sstream>
#include <utility>

#include "snapshot_client.h"

#include <nlohmann/json.hpp>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/un.h>
#include <unistd.h>

namespace {

using json = nlohmann::json;

constexpr int SOCKET_SEND_TIMEOUT_MS = 5000;
constexpr int SOCKET_RECEIVE_GRACE_MS = 5000;

class FileDescriptor {
  public:
    explicit FileDescriptor(int fd)
        : fd_(fd)
    {}

    ~FileDescriptor()
    {
        if (fd_ >= 0) {
            ::close(fd_);
        }
    }

    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;

    int get() const { return fd_; }

  private:
    int fd_ {-1};
};

bool readJsonInteger(const json& value, std::int64_t& output)
{
    if (value.is_number_unsigned()) {
        const auto raw = value.get<std::uint64_t>();
        if (raw > static_cast<std::uint64_t>(
                std::numeric_limits<std::int64_t>::max())) {
            return false;
        }
        output = static_cast<std::int64_t>(raw);
        return true;
    }
    if (value.is_number_integer()) {
        output = value.get<std::int64_t>();
        return true;
    }
    return false;
}

bool readRequiredInteger(const json& object,
                         const char* key,
                         std::int64_t& output)
{
    const auto it = object.find(key);
    return it != object.end() && readJsonInteger(*it, output);
}

bool readOptionalUnsigned(const json& object,
                          const char* key,
                          std::uint64_t& output)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return true;
    }
    if (it->is_number_unsigned()) {
        output = it->get<std::uint64_t>();
        return true;
    }
    if (it->is_number_integer()) {
        const auto value = it->get<std::int64_t>();
        if (value >= 0) {
            output = static_cast<std::uint64_t>(value);
            return true;
        }
    }
    return false;
}

bool readRequiredString(const json& object,
                        const char* key,
                        std::string& output)
{
    const auto it = object.find(key);
    if (it == object.end() || !it->is_string()) {
        return false;
    }
    output = it->get<std::string>();
    return !output.empty();
}

bool readOptionalString(const json& object,
                        const char* key,
                        std::string& output)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return true;
    }
    if (!it->is_string()) {
        return false;
    }
    output = it->get<std::string>();
    return true;
}

bool configureTimeout(int fd, int option, int timeoutMs)
{
    const timeval timeout {
        .tv_sec = timeoutMs / 1000,
        .tv_usec = (timeoutMs % 1000) * 1000,
    };
    return ::setsockopt(fd, SOL_SOCKET, option, &timeout, sizeof(timeout)) == 0;
}

bool sendAll(int fd, const void* data, std::size_t size, std::string& error)
{
    const auto* bytes = static_cast<const std::uint8_t*>(data);
    std::size_t sent = 0;
    while (sent < size) {
        const ssize_t count =
            ::send(fd, bytes + sent, size - sent, MSG_NOSIGNAL);
        if (count > 0) {
            sent += static_cast<std::size_t>(count);
            continue;
        }
        if (count < 0 && errno == EINTR) {
            continue;
        }
        error = std::string("send get_latest request: ") + std::strerror(errno);
        return false;
    }
    return true;
}

struct HeaderReadResult {
    bool ok {false};
    std::string error;
    std::string line;
    std::vector<std::uint8_t> payloadPrefix;
};

HeaderReadResult readHeaderLine(int fd)
{
    std::vector<std::uint8_t> buffer;
    buffer.reserve(4096);
    std::uint8_t chunk[8192];

    while (true) {
        const ssize_t count = ::recv(fd, chunk, sizeof(chunk), 0);
        if (count > 0) {
            buffer.insert(buffer.end(), chunk, chunk + count);
            const auto newline = std::find(buffer.begin(), buffer.end(), '\n');
            if (newline != buffer.end()) {
                const std::size_t headerSize =
                    static_cast<std::size_t>(newline - buffer.begin());
                if (headerSize > SNAPSHOT_MAX_HEADER_BYTES) {
                    return {.error = "snapshot response header too large"};
                }

                std::string line(buffer.begin(), newline);
                if (!line.empty() && line.back() == '\r') {
                    line.pop_back();
                }
                return {
                    .ok = true,
                    .line = std::move(line),
                    .payloadPrefix =
                        std::vector<std::uint8_t>(newline + 1, buffer.end()),
                };
            }
            if (buffer.size() > SNAPSHOT_MAX_HEADER_BYTES) {
                return {.error = "snapshot response header too large"};
            }
            continue;
        }
        if (count < 0 && errno == EINTR) {
            continue;
        }
        if (count == 0) {
            return {.error = "snapshot response ended before header newline"};
        }
        return {.error = std::string("read snapshot response header: ")
                    + std::strerror(errno)};
    }
}

bool readJpeg(int fd,
              std::vector<std::uint8_t> prefix,
              std::size_t expectedSize,
              std::vector<std::uint8_t>& jpeg,
              std::string& error)
{
    if (prefix.size() > expectedSize) {
        error = "snapshot response contains bytes beyond declared JPEG size";
        return false;
    }

    jpeg = std::move(prefix);
    jpeg.reserve(expectedSize);
    std::uint8_t chunk[64 * 1024];
    while (jpeg.size() < expectedSize) {
        const std::size_t remaining = expectedSize - jpeg.size();
        const ssize_t count =
            ::recv(fd, chunk, std::min(remaining, sizeof(chunk)), 0);
        if (count > 0) {
            jpeg.insert(jpeg.end(), chunk, chunk + count);
            continue;
        }
        if (count < 0 && errno == EINTR) {
            continue;
        }
        if (count == 0) {
            error = "snapshot response ended before complete JPEG payload";
            return false;
        }
        error =
            std::string("read snapshot JPEG payload: ") + std::strerror(errno);
        return false;
    }

    if (jpeg.size() < 4 || jpeg[0] != 0xff || jpeg[1] != 0xd8
        || jpeg[jpeg.size() - 2] != 0xff || jpeg.back() != 0xd9) {
        error = "snapshot response payload is not a complete JPEG";
        return false;
    }
    return true;
}

LatestFrameResult transportError(const std::string& error)
{
    return {
        .status = LatestFrameStatus::TransportError,
        .error = error,
    };
}

}  // namespace

std::string buildGetLatestRequest(const LatestFrameCursor& cursor, int waitMs)
{
    const int boundedWaitMs = std::clamp(waitMs, 0, SNAPSHOT_MAX_WAIT_MS);
    const json request {
        {"version",        SNAPSHOT_PROTOCOL_VERSION},
        {"cmd",            "get_latest"             },
        {"session_id",     cursor.sessionId         },
        {"after_sequence", cursor.sequence          },
        {"wait_ms",        boundedWaitMs            },
    };
    return request.dump() + "\n";
}

ParsedLatestHeader parseGetLatestHeader(const std::string& headerLine)
{
    const json header = json::parse(headerLine, nullptr, false);
    if (header.is_discarded() || !header.is_object()) {
        return {.error = "invalid snapshot response JSON"};
    }

    std::int64_t version = 0;
    if (!readRequiredInteger(header, "version", version)
        || version != SNAPSHOT_PROTOCOL_VERSION) {
        return {.error = "unsupported or missing snapshot protocol version"};
    }

    const auto okIt = header.find("ok");
    if (okIt == header.end() || !okIt->is_boolean()) {
        return {.error = "snapshot response missing boolean ok"};
    }

    std::int64_t retCode = 0;
    if (!readRequiredInteger(header, "ret_code", retCode)) {
        return {.error = "snapshot response missing integer ret_code"};
    }

    std::string message;
    if (!readRequiredString(header, "message", message)) {
        return {.error = "snapshot response missing message"};
    }

    std::int64_t rawSize = 0;
    if (!readRequiredInteger(header, "size", rawSize) || rawSize < 0
        || static_cast<std::uint64_t>(rawSize) > SNAPSHOT_MAX_JPEG_BYTES) {
        return {.error = "snapshot response has invalid JPEG size"};
    }

    if (!okIt->get<bool>()) {
        if (rawSize != 0) {
            return {.error = "failed snapshot response contains JPEG payload"};
        }
        std::ostringstream error;
        error << "snapshot server error " << retCode << ": " << message;
        return {
            .status = LatestFrameStatus::ServerError,
            .error = error.str(),
        };
    }

    if (retCode != 0) {
        return {.error = "successful snapshot response has non-zero ret_code"};
    }

    bool notModified = false;
    const auto notModifiedIt = header.find("not_modified");
    if (notModifiedIt != header.end()) {
        if (!notModifiedIt->is_boolean()) {
            return {.error = "snapshot response has invalid not_modified"};
        }
        notModified = notModifiedIt->get<bool>();
    }

    if (notModified || message == "not_modified") {
        if (!notModified || message != "not_modified" || rawSize != 0) {
            return {.error = "inconsistent not_modified snapshot response"};
        }
        ParsedLatestHeader parsed {
            .status = LatestFrameStatus::NotModified,
            .size = 0,
        };
        if (!readOptionalString(header, "session_id", parsed.sessionId)
            || !readOptionalUnsigned(header, "sequence", parsed.sequence)) {
            return {.error = "not_modified response has invalid cursor"};
        }
        return parsed;
    }

    if (message != "ok") {
        return {.error = "successful snapshot response has invalid message"};
    }

    std::string source;
    if (!readRequiredString(header, "source", source)
        || source != "canonical") {
        return {.error = "snapshot response source is not canonical"};
    }

    ParsedLatestHeader parsed {
        .status = LatestFrameStatus::Frame,
        .size = static_cast<std::size_t>(rawSize),
    };
    if (!readRequiredString(header, "session_id", parsed.sessionId)
        || !readOptionalUnsigned(header, "sequence", parsed.sequence)
        || parsed.sequence == 0
        || !readRequiredString(header, "capture_id", parsed.captureId)
        || !readRequiredString(header, "captured_at", parsed.capturedAt)) {
        return {.error = "snapshot frame response has invalid metadata"};
    }

    std::int64_t width = 0;
    std::int64_t height = 0;
    if (!readRequiredInteger(header, "width", width) || width <= 0
        || width > std::numeric_limits<int>::max()
        || !readRequiredInteger(header, "height", height) || height <= 0
        || height > std::numeric_limits<int>::max() || parsed.size == 0) {
        return {.error =
                    "snapshot frame response has invalid dimensions or size"};
    }
    parsed.width = static_cast<int>(width);
    parsed.height = static_cast<int>(height);
    return parsed;
}

SnapshotClient::SnapshotClient(std::string socketPath)
    : socketPath_(std::move(socketPath))
{}

LatestFrameResult SnapshotClient::getLatest(const LatestFrameCursor& cursor,
                                            int waitMs) const
{
    if (socketPath_.empty()) {
        return transportError("snapshot socket path is empty");
    }
    if (waitMs < 0 || waitMs > SNAPSHOT_MAX_WAIT_MS) {
        return {
            .status = LatestFrameStatus::ProtocolError,
            .error = "get_latest wait_ms is outside 0..30000",
        };
    }

    sockaddr_un address {};
    address.sun_family = AF_UNIX;
    if (socketPath_.size() >= sizeof(address.sun_path)) {
        return transportError("snapshot socket path is too long");
    }
    std::memcpy(address.sun_path, socketPath_.c_str(), socketPath_.size() + 1);
    const socklen_t addressSize = static_cast<socklen_t>(
        offsetof(sockaddr_un, sun_path) + socketPath_.size() + 1);

    const int rawFd = ::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
    if (rawFd < 0) {
        return transportError(std::string("create snapshot socket: ")
                              + std::strerror(errno));
    }
    FileDescriptor fd(rawFd);

    if (!configureTimeout(fd.get(), SO_SNDTIMEO, SOCKET_SEND_TIMEOUT_MS)
        || !configureTimeout(
            fd.get(), SO_RCVTIMEO, waitMs + SOCKET_RECEIVE_GRACE_MS)) {
        return transportError(std::string("configure snapshot socket: ")
                              + std::strerror(errno));
    }

    if (::connect(
            fd.get(), reinterpret_cast<const sockaddr*>(&address), addressSize)
        != 0) {
        return transportError(std::string("connect snapshot socket: ")
                              + std::strerror(errno));
    }

    const std::string request = buildGetLatestRequest(cursor, waitMs);
    std::string ioError;
    if (!sendAll(fd.get(), request.data(), request.size(), ioError)) {
        return transportError(ioError);
    }

    HeaderReadResult input = readHeaderLine(fd.get());
    if (!input.ok) {
        return transportError(input.error);
    }

    ParsedLatestHeader header = parseGetLatestHeader(input.line);
    if (header.status == LatestFrameStatus::ProtocolError) {
        return {
            .status = LatestFrameStatus::ProtocolError,
            .error = header.error,
        };
    }
    if (header.status == LatestFrameStatus::ServerError) {
        return {
            .status = LatestFrameStatus::ServerError,
            .error = header.error,
        };
    }
    if (header.status == LatestFrameStatus::NotModified) {
        if (!input.payloadPrefix.empty()) {
            return {
                .status = LatestFrameStatus::ProtocolError,
                .error = "not_modified response contains a payload",
            };
        }
        return {
            .status = LatestFrameStatus::NotModified,
            .currentSessionId = std::move(header.sessionId),
            .currentSequence = header.sequence,
        };
    }

    std::vector<std::uint8_t> jpeg;
    if (!readJpeg(fd.get(),
                  std::move(input.payloadPrefix),
                  header.size,
                  jpeg,
                  ioError)) {
        return {
            .status = LatestFrameStatus::ProtocolError,
            .error = ioError,
        };
    }

    return {
        .status = LatestFrameStatus::Frame,
        .frame =
            {
                    .sessionId = std::move(header.sessionId),
                    .sequence = header.sequence,
                    .captureId = std::move(header.captureId),
                    .capturedAt = std::move(header.capturedAt),
                    .width = header.width,
                    .height = header.height,
                    .jpeg = std::move(jpeg),
                    },
    };
}
