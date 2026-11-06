#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <cstring>
#include <future>
#include <memory>
#include <mutex>
#include <regex>
#include <string>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include "snapshot_socket_server.hpp"

#include <catch2/catch_test_macros.hpp>
#include <fcntl.h>
#include <nlohmann/json.hpp>
#include <spdlog/logger.h>
#include <spdlog/sinks/null_sink.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

namespace {

using json = nlohmann::json;
using namespace std::chrono_literals;

class NullLogger final : public ILogger {
  public:
    NullLogger()
    {
        static std::atomic<unsigned long> sequence {0};
        auto sink = std::make_shared<spdlog::sinks::null_sink_mt>();
        _logger = std::make_shared<spdlog::logger>(
            "snapshot-socket-test-" + std::to_string(sequence.fetch_add(1)),
            std::move(sink));
    }

    void logger_init(const std::string&,
                     const std::string&,
                     const std::string&) override
    {}

    std::shared_ptr<spdlog::logger> operator->() override { return _logger; }

  private:
    std::shared_ptr<spdlog::logger> _logger;
};

class UniqueFd {
  public:
    explicit UniqueFd(int fd = -1)
        : _fd(fd)
    {}

    ~UniqueFd()
    {
        if (_fd >= 0) {
            ::close(_fd);
        }
    }

    UniqueFd(const UniqueFd&) = delete;
    UniqueFd& operator=(const UniqueFd&) = delete;

    UniqueFd(UniqueFd&& other) noexcept
        : _fd(std::exchange(other._fd, -1))
    {}

    UniqueFd& operator=(UniqueFd&& other) noexcept
    {
        if (this != &other) {
            if (_fd >= 0) {
                ::close(_fd);
            }
            _fd = std::exchange(other._fd, -1);
        }
        return *this;
    }

    [[nodiscard]] int get() const { return _fd; }

  private:
    int _fd;
};

class SocketPath {
  public:
    SocketPath()
    {
        static std::atomic<unsigned long> sequence {0};
        _path = "/tmp/kvm-vin-snapshot-test-" + std::to_string(::getpid()) + '-'
            + std::to_string(sequence.fetch_add(1)) + ".sock";
        (void)::unlink(_path.c_str());
    }

    ~SocketPath() { (void)::unlink(_path.c_str()); }

    [[nodiscard]] const std::string& value() const { return _path; }

  private:
    std::string _path;
};

class ReleaseFlagGuard {
  public:
    ReleaseFlagGuard(std::mutex& mutex, std::condition_variable& cv, bool& flag)
        : _mutex(mutex)
        , _cv(cv)
        , _flag(flag)
    {}

    ~ReleaseFlagGuard() { release(); }

    void release()
    {
        {
            std::lock_guard lock(_mutex);
            _flag = true;
        }
        _cv.notify_all();
    }

  private:
    std::mutex& _mutex;
    std::condition_variable& _cv;
    bool& _flag;
};

struct SocketResponse {
    json header;
    std::vector<std::uint8_t> jpeg;
};

void set_socket_timeout(int fd)
{
    const timeval timeout {.tv_sec = 2, .tv_usec = 0};
    REQUIRE(::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout))
            == 0);
    REQUIRE(::setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout))
            == 0);
}

UniqueFd connect_client(const std::string& path)
{
    UniqueFd client(::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0));
    REQUIRE(client.get() >= 0);
    set_socket_timeout(client.get());

    sockaddr_un address {};
    address.sun_family = AF_UNIX;
    REQUIRE(path.size() < sizeof(address.sun_path));
    std::memcpy(address.sun_path, path.c_str(), path.size() + 1);
    const socklen_t address_size = static_cast<socklen_t>(
        offsetof(sockaddr_un, sun_path) + path.size() + 1);
    REQUIRE(::connect(client.get(),
                      reinterpret_cast<const sockaddr*>(&address),
                      address_size)
            == 0);
    return client;
}

void send_all(int fd, std::string_view data)
{
    std::size_t sent = 0;
    while (sent < data.size()) {
        const ssize_t count =
            ::send(fd, data.data() + sent, data.size() - sent, MSG_NOSIGNAL);
        REQUIRE(count > 0);
        sent += static_cast<std::size_t>(count);
    }
}

std::string read_line(int fd)
{
    std::string line;
    while (line.size() <= SnapshotSocketServer::MAX_REQUEST_HEADER_BYTES) {
        char ch = '\0';
        const ssize_t count = ::recv(fd, &ch, 1, 0);
        REQUIRE(count == 1);
        if (ch == '\n') {
            return line;
        }
        line.push_back(ch);
    }
    FAIL("response header did not end with a newline");
    return {};
}

std::vector<std::uint8_t> read_exact(int fd, std::size_t size)
{
    std::vector<std::uint8_t> data(size);
    std::size_t received = 0;
    while (received < size) {
        const ssize_t count =
            ::recv(fd, data.data() + received, size - received, 0);
        REQUIRE(count > 0);
        received += static_cast<std::size_t>(count);
    }
    return data;
}

SocketResponse read_response(int fd)
{
    SocketResponse response;
    response.header = json::parse(read_line(fd));
    const auto size = response.header.at("size").get<std::size_t>();
    response.jpeg = read_exact(fd, size);
    return response;
}

template<typename Predicate>
bool wait_until(Predicate predicate, std::chrono::milliseconds timeout = 2s)
{
    const auto deadline = std::chrono::steady_clock::now() + timeout;
    while (std::chrono::steady_clock::now() < deadline) {
        if (predicate()) {
            return true;
        }
        std::this_thread::sleep_for(5ms);
    }
    return predicate();
}

bool is_rfc3339_nano_utc(const std::string& timestamp)
{
    static const std::regex pattern(
        R"(^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{9}Z$)");
    return std::regex_match(timestamp, pattern);
}

CaptureResponse make_canonical_response(std::string session_id,
                                        std::string capture_id,
                                        std::uint8_t marker)
{
    auto frame = std::make_shared<AxeraUtils::SnapshotResult>();
    frame->width = 1280;
    frame->height = 720;
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

TEST_CASE("Snapshot socket capture_fresh uses full-frame defaults",
          "[capture][snapshot-socket]")
{
    std::mutex request_mutex;
    CaptureRequest observed;
    bool captured = false;
    const std::vector<std::uint8_t> expected_jpeg {0xff, 0xd8, 0xff, 0xd9};

    CaptureCoordinator coordinator(
        [&](const CaptureRequest& request, AxeraUtils::SnapshotResult& result) {
            {
                std::lock_guard lock(request_mutex);
                observed = request;
                captured = true;
            }
            result.width = 640;
            result.height = 480;
            result.data = expected_jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 2);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(client.get(),
             R"({"version":1,"cmd":"capture_fresh"})"
             "\n");
    const auto response = read_response(client.get());

    REQUIRE(response.header.at("version") == SNAPSHOT_SOCKET_PROTOCOL_VERSION);
    REQUIRE(response.header.at("ok") == true);
    REQUIRE(response.header.at("ret_code") == 0);
    REQUIRE(response.header.at("source") == "mcp");
    REQUIRE(response.header.at("width") == 640);
    REQUIRE(response.header.at("height") == 480);
    REQUIRE(response.header.at("size") == expected_jpeg.size());
    REQUIRE_FALSE(response.header.at("session_id").get<std::string>().empty());
    REQUIRE_FALSE(response.header.at("capture_id").get<std::string>().empty());
    REQUIRE(is_rfc3339_nano_utc(
        response.header.at("captured_at").get<std::string>()));
    REQUIRE(response.jpeg == expected_jpeg);

    {
        std::lock_guard lock(request_mutex);
        REQUIRE(captured);
        REQUIRE(observed.source == CaptureSource::Mcp);
        REQUIRE(observed.quality == 85);
        REQUIRE(observed.x == 0);
        REQUIRE(observed.y == 0);
        REQUIRE(observed.w == 0);
        REQUIRE(observed.h == 0);
        REQUIRE(observed.timeout_ms > 0);
        REQUIRE(observed.timeout_ms <= 1000);
    }

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket stays available when capture is disabled",
          "[capture][snapshot-socket]")
{
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++captures;
            return CaptureCoordinator::RESULT_OK;
        });

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return false; }, logger, socket_path.value(), 1);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(client.get(),
             R"({"version":1,"cmd":"capture_fresh"})"
             "\n");
    const auto response = read_response(client.get());

    REQUIRE(response.header.at("ok") == false);
    REQUIRE(response.header.at("ret_code") == -5);
    REQUIRE(response.header.at("size") == 0);
    REQUIRE(captures.load() == 0);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket rejects non-integer optional fields",
          "[capture][snapshot-socket]")
{
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++captures;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 2);
    REQUIRE(server.start());

    SECTION("wrong JSON type")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(),
                 R"({"version":1,"cmd":"capture_fresh","quality":"85"})"
                 "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
        REQUIRE(response.header.at("size") == 0);
    }

    SECTION("integer overflow")
    {
        auto client = connect_client(socket_path.value());
        send_all(
            client.get(),
            R"({"version":1,"cmd":"capture_fresh","timeout_ms":2147483648})"
            "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
        REQUIRE(response.header.at("size") == 0);
    }

    SECTION("negative integer overflow")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(),
                 R"({"version":1,"cmd":"capture_fresh","x":-2147483649})"
                 "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
        REQUIRE(response.header.at("size") == 0);
    }

    REQUIRE(captures.load() == 0);
    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket validates protocol framing and command fields",
          "[capture][snapshot-socket]")
{
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++captures;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 2);
    REQUIRE(server.start());

    SECTION("unsupported version")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(),
                 R"({"version":2,"cmd":"capture_fresh"})"
                 "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
    }

    SECTION("unsupported command")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(),
                 R"({"version":1,"cmd":"unknown"})"
                 "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -2);
    }

    SECTION("newline is mandatory")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(), R"({"version":1,"cmd":"capture_fresh"})");
        REQUIRE(::shutdown(client.get(), SHUT_WR) == 0);
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
    }

    SECTION("request header is bounded")
    {
        auto client = connect_client(socket_path.value());
        send_all(client.get(),
                 std::string(SnapshotSocketServer::MAX_REQUEST_HEADER_BYTES + 1,
                             'x'));
        REQUIRE(::shutdown(client.get(), SHUT_WR) == 0);
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == -4);
    }

    REQUIRE(captures.load() == 0);
    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket coalesced clients share capture metadata",
          "[capture][snapshot-socket]")
{
    std::mutex mutex;
    std::condition_variable cv;
    bool first_entered = false;
    bool release_first = false;
    std::atomic<int> captures {0};
    const std::vector<std::uint8_t> expected_jpeg {1, 2, 3, 4, 5};

    CaptureCoordinator coordinator(
        [&](const CaptureRequest& request, AxeraUtils::SnapshotResult& result) {
            const int call = ++captures;
            if (call == 1) {
                std::unique_lock lock(mutex);
                first_entered = true;
                cv.notify_all();
                cv.wait(lock, [&release_first]() { return release_first; });
            }
            result.width = request.x == 99 ? 16 : 1920;
            result.height = request.x == 99 ? 16 : 1080;
            result.data =
                request.x == 99 ? std::vector<std::uint8_t> {9} : expected_jpeg;
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();
    ReleaseFlagGuard first_release_guard(mutex, cv, release_first);

    CaptureRequest blocker;
    blocker.x = 99;
    blocker.w = 16;
    blocker.h = 16;
    blocker.timeout_ms = 2000;
    auto blocker_ticket = coordinator.submit(blocker);
    {
        std::unique_lock lock(mutex);
        REQUIRE(cv.wait_for(
            lock, 2s, [&first_entered]() { return first_entered; }));
    }

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 2);
    REQUIRE(server.start());

    auto first_client = connect_client(socket_path.value());
    auto second_client = connect_client(socket_path.value());
    const std::string request = R"({"version":1,"cmd":"capture_fresh"})"
                                "\n";
    send_all(first_client.get(), request);
    send_all(second_client.get(), request);
    REQUIRE(wait_until(
        [&coordinator]() { return coordinator.queued_requests() == 2; }));

    first_release_guard.release();

    REQUIRE(blocker_ticket.wait().ok());
    const auto first = read_response(first_client.get());
    const auto second = read_response(second_client.get());

    REQUIRE(first.header.at("ok") == true);
    REQUIRE(second.header.at("ok") == true);
    REQUIRE(first.jpeg == expected_jpeg);
    REQUIRE(second.jpeg == expected_jpeg);
    REQUIRE(first.header.at("session_id") == second.header.at("session_id"));
    REQUIRE(first.header.at("capture_id") == second.header.at("capture_id"));
    REQUIRE(first.header.at("captured_at") == second.header.at("captured_at"));
    REQUIRE(captures.load() == 2);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket enforces its active connection limit",
          "[capture][snapshot-socket]")
{
    std::mutex mutex;
    std::condition_variable cv;
    bool entered = false;
    bool release = false;

    CaptureCoordinator coordinator(
        [&](const CaptureRequest&, AxeraUtils::SnapshotResult& result) {
            std::unique_lock lock(mutex);
            entered = true;
            cv.notify_all();
            cv.wait(lock, [&release]() { return release; });
            result.width = 1;
            result.height = 1;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();
    ReleaseFlagGuard release_guard(mutex, cv, release);

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 1);
    REQUIRE(server.start());

    auto first_client = connect_client(socket_path.value());
    send_all(first_client.get(),
             R"({"version":1,"cmd":"capture_fresh"})"
             "\n");
    {
        std::unique_lock lock(mutex);
        REQUIRE(cv.wait_for(lock, 2s, [&entered]() { return entered; }));
    }

    auto second_client = connect_client(socket_path.value());
    send_all(second_client.get(),
             R"({"version":1,"cmd":"capture_fresh"})"
             "\n");
    const auto busy = read_response(second_client.get());
    REQUIRE(busy.header.at("ok") == false);
    REQUIRE(busy.header.at("ret_code") == -16);
    REQUIRE(busy.header.at("size") == 0);

    release_guard.release();
    REQUIRE(read_response(first_client.get()).header.at("ok") == true);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket stop cancels active clients and joins promptly",
          "[capture][snapshot-socket]")
{
    std::mutex mutex;
    std::condition_variable cv;
    bool entered = false;
    bool release = false;

    CaptureCoordinator coordinator(
        [&](const CaptureRequest&, AxeraUtils::SnapshotResult& result) {
            std::unique_lock lock(mutex);
            entered = true;
            cv.notify_all();
            cv.wait(lock, [&release]() { return release; });
            result.width = 1;
            result.height = 1;
            result.data = {1};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();
    ReleaseFlagGuard release_guard(mutex, cv, release);

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator, []() { return true; }, logger, socket_path.value(), 1);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(client.get(),
             R"({"version":1,"cmd":"capture_fresh"})"
             "\n");
    {
        std::unique_lock lock(mutex);
        REQUIRE(cv.wait_for(lock, 2s, [&entered]() { return entered; }));
    }

    std::atomic<bool> stopped {false};
    std::jthread stopper([&server, &stopped]() {
        server.stop();
        stopped.store(true, std::memory_order_release);
    });
    const bool stopped_promptly = wait_until(
        [&stopped]() { return stopped.load(std::memory_order_acquire); });

    if (!stopped_promptly) {
        (void)::shutdown(client.get(), SHUT_RDWR);
        release_guard.release();
    }
    REQUIRE(stopped_promptly);
    stopper.join();

    release_guard.release();
    coordinator.stop();
}

TEST_CASE("Snapshot socket get_latest returns the canonical frame without JENC",
          "[capture][snapshot-socket][ocr-latest]")
{
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            ++captures;
            return CaptureCoordinator::RESULT_OK;
        });
    OCRLatestFrameStore latest_store;
    const auto canonical =
        make_canonical_response("session-a", "session-a:7", 7);
    REQUIRE(latest_store.publish(canonical));

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return false; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(
        client.get(),
        R"({"version":1,"cmd":"get_latest","session_id":"old-session","after_sequence":99,"wait_ms":1000})"
        "\n");
    const auto response = read_response(client.get());

    REQUIRE(response.header.at("ok") == true);
    REQUIRE(response.header.at("message") == "ok");
    REQUIRE(response.header.at("source") == "canonical");
    REQUIRE(response.header.at("session_id") == "session-a");
    REQUIRE(response.header.at("sequence") == 1);
    REQUIRE(response.header.at("capture_id") == "session-a:7");
    REQUIRE(response.header.at("captured_at")
            == "2026-07-14T12:00:00.000000000Z");
    REQUIRE(response.header.at("width") == 1280);
    REQUIRE(response.header.at("height") == 720);
    REQUIRE(response.jpeg == canonical.frame->data);
    REQUIRE(captures.load() == 0);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket get_latest returns not_modified after timeout",
          "[capture][snapshot-socket][ocr-latest]")
{
    CaptureCoordinator coordinator(
        [](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            return CaptureCoordinator::RESULT_OK;
        });
    OCRLatestFrameStore latest_store;
    REQUIRE(latest_store.publish(
        make_canonical_response("session-a", "session-a:1", 1)));

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return true; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    const auto started = std::chrono::steady_clock::now();
    send_all(
        client.get(),
        R"({"version":1,"cmd":"get_latest","session_id":"session-a","after_sequence":1,"wait_ms":40})"
        "\n");
    const auto response = read_response(client.get());
    const auto elapsed = std::chrono::steady_clock::now() - started;

    REQUIRE(response.header.at("ok") == true);
    REQUIRE(response.header.at("message") == "not_modified");
    REQUIRE(response.header.at("not_modified") == true);
    REQUIRE(response.header.at("session_id") == "session-a");
    REQUIRE(response.header.at("sequence") == 1);
    REQUIRE(response.header.at("size") == 0);
    REQUIRE(response.jpeg.empty());
    REQUIRE(elapsed >= 20ms);

    server.stop();
    coordinator.stop();
}

TEST_CASE(
    "Snapshot socket high-frequency MCP captures do not alter OCR sequence",
    "[capture][snapshot-socket][ocr-latest]")
{
    std::atomic<int> captures {0};
    CaptureCoordinator coordinator(
        [&captures](const CaptureRequest&, AxeraUtils::SnapshotResult& result) {
            ++captures;
            result.width = 320;
            result.height = 200;
            result.data = {0xff, 0xd8, 0x55, 0xff, 0xd9};
            return CaptureCoordinator::RESULT_OK;
        });
    coordinator.start();

    OCRLatestFrameStore latest_store;
    REQUIRE(latest_store.publish(
        make_canonical_response("session-a", "session-a:1", 1)));

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return true; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    auto latest_client = connect_client(socket_path.value());
    send_all(
        latest_client.get(),
        R"({"version":1,"cmd":"get_latest","session_id":"session-a","after_sequence":1,"wait_ms":5000})"
        "\n");
    std::this_thread::sleep_for(40ms);

    constexpr int mcp_capture_count = 32;
    for (int capture = 0; capture < mcp_capture_count; ++capture) {
        auto capture_client = connect_client(socket_path.value());
        send_all(capture_client.get(),
                 R"({"version":1,"cmd":"capture_fresh","timeout_ms":1000})"
                 "\n");
        const auto capture_response = read_response(capture_client.get());
        REQUIRE(capture_response.header.at("ok") == true);
        REQUIRE(capture_response.header.at("source") == "mcp");
    }
    REQUIRE(captures.load() == mcp_capture_count);

    const auto unchanged = latest_store.get_latest("session-a", 1, 0ms);
    REQUIRE(unchanged.status == OCRLatestFrameStatus::NotModified);
    REQUIRE(unchanged.sequence == 1);

    const auto second = make_canonical_response("session-a", "session-a:2", 2);
    REQUIRE(latest_store.publish(second));
    const auto latest_response = read_response(latest_client.get());
    REQUIRE(latest_response.header.at("ok") == true);
    REQUIRE(latest_response.header.at("sequence") == 2);
    REQUIRE(latest_response.header.at("capture_id") == "session-a:2");
    REQUIRE(latest_response.jpeg == second.frame->data);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket rejects canonical JPEG larger than 32 MiB",
          "[capture][snapshot-socket][ocr-latest]")
{
    CaptureCoordinator coordinator(
        [](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            return CaptureCoordinator::RESULT_OK;
        });

    auto oversized_frame = std::make_shared<AxeraUtils::SnapshotResult>();
    oversized_frame->width = 3840;
    oversized_frame->height = 2160;
    oversized_frame->data.resize(SnapshotSocketServer::MAX_JPEG_BYTES + 1,
                                 0x55);
    CaptureResponse oversized {
        .ret_code = CaptureCoordinator::RESULT_OK,
        .frame = std::move(oversized_frame),
        .session_id = "session-a",
        .capture_id = "session-a:1",
        .captured_at = "2026-07-14T12:00:00.000000000Z",
    };

    OCRLatestFrameStore latest_store;
    REQUIRE(latest_store.publish(oversized));

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return true; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(
        client.get(),
        R"({"version":1,"cmd":"get_latest","session_id":"old-session","after_sequence":0,"wait_ms":0})"
        "\n");
    const auto response = read_response(client.get());

    REQUIRE(response.header.at("ok") == false);
    REQUIRE(response.header.at("ret_code") == -14);
    REQUIRE(response.header.at("message") == "invalid JPEG response");
    REQUIRE(response.header.at("size") == 0);
    REQUIRE(response.jpeg.empty());

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket validates get_latest request fields",
          "[capture][snapshot-socket][ocr-latest]")
{
    CaptureCoordinator coordinator(
        [](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            return CaptureCoordinator::RESULT_OK;
        });
    OCRLatestFrameStore latest_store;
    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return true; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    const auto expect_invalid = [&](std::string_view request, int ret_code) {
        auto client = connect_client(socket_path.value());
        send_all(client.get(), std::string(request) + "\n");
        const auto response = read_response(client.get());
        REQUIRE(response.header.at("ok") == false);
        REQUIRE(response.header.at("ret_code") == ret_code);
        REQUIRE(response.header.at("size") == 0);
    };

    expect_invalid(R"({"version":1,"cmd":"get_latest","session_id":7})", -4);
    expect_invalid(R"({"version":1,"cmd":"get_latest","after_sequence":-1})",
                   -4);
    expect_invalid(R"({"version":1,"cmd":"get_latest","wait_ms":"100"})", -4);
    expect_invalid(R"({"version":1,"cmd":"get_latest","wait_ms":30001})", -11);

    server.stop();
    coordinator.stop();
}

TEST_CASE("Snapshot socket stop wakes a get_latest long poll promptly",
          "[capture][snapshot-socket][ocr-latest]")
{
    CaptureCoordinator coordinator(
        [](const CaptureRequest&, AxeraUtils::SnapshotResult&) {
            return CaptureCoordinator::RESULT_OK;
        });
    OCRLatestFrameStore latest_store;
    REQUIRE(latest_store.publish(
        make_canonical_response("session-a", "session-a:1", 1)));

    NullLogger logger;
    SocketPath socket_path;
    SnapshotSocketServer server(
        coordinator,
        latest_store,
        []() { return true; },
        logger,
        socket_path.value(),
        2);
    REQUIRE(server.start());

    auto client = connect_client(socket_path.value());
    send_all(
        client.get(),
        R"({"version":1,"cmd":"get_latest","session_id":"session-a","after_sequence":1,"wait_ms":30000})"
        "\n");
    std::this_thread::sleep_for(40ms);

    std::atomic<bool> stopped {false};
    std::jthread stopper([&server, &stopped]() {
        server.stop();
        stopped.store(true, std::memory_order_release);
    });
    REQUIRE(wait_until(
        [&stopped]() { return stopped.load(std::memory_order_acquire); }, 1s));
    stopper.join();

    coordinator.stop();
}
