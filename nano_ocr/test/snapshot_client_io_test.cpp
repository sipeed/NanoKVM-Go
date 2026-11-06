#include <atomic>
#include <cerrno>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <stdexcept>
#include <string>
#include <thread>

#include "snapshot_client.h"

#include <catch2/catch_test_macros.hpp>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

namespace {

std::string uniqueSocketPath()
{
    static std::atomic<unsigned int> counter {0};
    return "/tmp/nano-ocr-snapshot-test-" + std::to_string(getpid()) + "-"
        + std::to_string(counter.fetch_add(1)) + ".sock";
}

class FakeSnapshotServer {
  public:
    explicit FakeSnapshotServer(std::string response,
                                std::string socketPath = {})
    {
        path_ = socketPath.empty() ? uniqueSocketPath() : std::move(socketPath);
        unlink(path_.c_str());

        listenFd_ = socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
        if (listenFd_ < 0) {
            throw std::runtime_error(std::string("socket: ")
                                     + std::strerror(errno));
        }

        sockaddr_un address {};
        address.sun_family = AF_UNIX;
        std::memcpy(address.sun_path, path_.c_str(), path_.size() + 1);
        const socklen_t addressSize = static_cast<socklen_t>(
            offsetof(sockaddr_un, sun_path) + path_.size() + 1);
        if (bind(listenFd_,
                 reinterpret_cast<const sockaddr*>(&address),
                 addressSize)
                != 0
            || listen(listenFd_, 1) != 0) {
            const std::string error = std::strerror(errno);
            close(listenFd_);
            listenFd_ = -1;
            unlink(path_.c_str());
            throw std::runtime_error("bind/listen: " + error);
        }

        worker_ = std::thread([this, response = std::move(response)] {
            const int client =
                accept4(listenFd_, nullptr, nullptr, SOCK_CLOEXEC);
            if (client >= 0) {
                char request[4096];
                ssize_t received = 0;
                while ((received = recv(client, request, sizeof(request), 0))
                       > 0) {
                    if (std::memchr(
                            request, '\n', static_cast<std::size_t>(received))
                        != nullptr) {
                        break;
                    }
                }

                std::size_t sent = 0;
                while (sent < response.size()) {
                    const ssize_t count = send(client,
                                               response.data() + sent,
                                               response.size() - sent,
                                               MSG_NOSIGNAL);
                    if (count > 0) {
                        sent += static_cast<std::size_t>(count);
                    } else if (count < 0 && errno == EINTR) {
                        continue;
                    } else {
                        break;
                    }
                }
                close(client);
            }
        });
    }

    ~FakeSnapshotServer()
    {
        if (worker_.joinable()) {
            worker_.join();
        }
        if (listenFd_ >= 0) {
            close(listenFd_);
        }
        unlink(path_.c_str());
    }

    const std::string& path() const { return path_; }

  private:
    std::string path_;
    int listenFd_ {-1};
    std::thread worker_;
};

std::string frameHeader(std::size_t size)
{
    return std::string(
               R"({"version":1,"ok":true,"ret_code":0,"message":"ok","session_id":"session-1","sequence":9,"capture_id":"session-1:21","captured_at":"2026-07-14T12:00:00.123+08:00","width":2,"height":1,"source":"canonical","size":)")
        + std::to_string(size) + "}\n";
}

}  // namespace

TEST_CASE("SnapshotClient reads a JSON header followed by raw JPEG bytes",
          "[ocr][socket]")
{
    const std::string jpeg("\xff\xd8\x01\x02\xff\xd9", 6);
    FakeSnapshotServer server(frameHeader(jpeg.size()) + jpeg);

    SnapshotClient client(server.path());
    const LatestFrameResult result = client.getLatest({}, 1000);

    REQUIRE(result.status == LatestFrameStatus::Frame);
    CHECK(result.frame.sessionId == "session-1");
    CHECK(result.frame.sequence == 9);
    CHECK(result.frame.captureId == "session-1:21");
    CHECK(result.frame.jpeg
          == std::vector<std::uint8_t>(jpeg.begin(), jpeg.end()));
}

TEST_CASE("SnapshotClient rejects a truncated raw JPEG payload",
          "[ocr][socket]")
{
    const std::string truncated("\xff\xd8\xff\xd9", 4);
    FakeSnapshotServer server(frameHeader(6) + truncated);

    SnapshotClient client(server.path());
    const LatestFrameResult result = client.getLatest({}, 1000);

    CHECK(result.status == LatestFrameStatus::ProtocolError);
    CHECK(result.error.find("complete JPEG") != std::string::npos);
}

TEST_CASE("SnapshotClient accepts a payload-free not_modified response",
          "[ocr][socket]")
{
    FakeSnapshotServer server(
        R"({"version":1,"ok":true,"ret_code":0,"message":"not_modified","not_modified":true,"session_id":"session-1","sequence":9,"width":0,"height":0,"size":0})"
        "\n");

    SnapshotClient client(server.path());
    const LatestFrameResult result =
        client.getLatest({.sessionId = "session-1", .sequence = 9}, 1000);

    CHECK(result.status == LatestFrameStatus::NotModified);
    CHECK(result.currentSessionId == "session-1");
    CHECK(result.currentSequence == 9);
}

TEST_CASE("SnapshotClient reconnects when the socket appears after startup",
          "[ocr][socket]")
{
    const std::string socketPath = uniqueSocketPath();
    unlink(socketPath.c_str());
    SnapshotClient client(socketPath);

    const LatestFrameResult unavailable = client.getLatest({}, 1000);
    REQUIRE(unavailable.status == LatestFrameStatus::TransportError);

    const std::string jpeg("\xff\xd8\x01\x02\xff\xd9", 6);
    FakeSnapshotServer server(frameHeader(jpeg.size()) + jpeg, socketPath);
    const LatestFrameResult connected = client.getLatest({}, 1000);

    REQUIRE(connected.status == LatestFrameStatus::Frame);
    CHECK(connected.frame.sequence == 9);
    CHECK(connected.frame.jpeg
          == std::vector<std::uint8_t>(jpeg.begin(), jpeg.end()));
}

TEST_CASE("SnapshotClient preserves snapshot server errors", "[ocr][socket]")
{
    FakeSnapshotServer server(
        R"({"version":1,"ok":false,"ret_code":-16,"message":"latest frame server busy","size":0})"
        "\n");
    SnapshotClient client(server.path());

    const LatestFrameResult result = client.getLatest({}, 1000);

    CHECK(result.status == LatestFrameStatus::ServerError);
    CHECK(result.error.find("-16") != std::string::npos);
    CHECK(result.error.find("latest frame server busy") != std::string::npos);
}

TEST_CASE("SnapshotClient enforces declared and actual JPEG size boundaries",
          "[ocr][socket]")
{
    SECTION("declared size above the protocol maximum")
    {
        FakeSnapshotServer server(frameHeader(SNAPSHOT_MAX_JPEG_BYTES + 1));
        SnapshotClient client(server.path());

        const LatestFrameResult result = client.getLatest({}, 1000);

        CHECK(result.status == LatestFrameStatus::ProtocolError);
        CHECK(result.error.find("invalid JPEG size") != std::string::npos);
    }

    SECTION("actual payload contains bytes beyond its declared size")
    {
        const std::string jpegWithTrailingBytes(
            "\xff\xd8\x01\x02\xff\xd9\xaa\xbb", 8);
        FakeSnapshotServer server(frameHeader(6) + jpegWithTrailingBytes);
        SnapshotClient client(server.path());

        const LatestFrameResult result = client.getLatest({}, 1000);

        CHECK(result.status == LatestFrameStatus::ProtocolError);
        CHECK(result.error.find("beyond declared JPEG size")
              != std::string::npos);
    }
}
