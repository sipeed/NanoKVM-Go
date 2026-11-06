#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <fstream>
#include <future>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include "snapshot_http_contract.hpp"
#include "snapshot_http_server.hpp"

#include <arpa/inet.h>
#include <catch2/catch_test_macros.hpp>
#include <httplib.h>
#include <nlohmann/json.hpp>
#include <spdlog/logger.h>
#include <spdlog/sinks/null_sink.h>
#include <sys/socket.h>
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
            "snapshot-http-test-" + std::to_string(sequence.fetch_add(1)),
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

int reserve_loopback_port()
{
    UniqueFd socket(::socket(AF_INET, SOCK_STREAM | SOCK_CLOEXEC, 0));
    if (socket.get() < 0) {
        return -1;
    }

    sockaddr_in address {};
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    address.sin_port = htons(0);
    if (::bind(socket.get(),
               reinterpret_cast<const sockaddr*>(&address),
               sizeof(address))
        != 0) {
        return -1;
    }

    socklen_t address_size = sizeof(address);
    if (::getsockname(socket.get(),
                      reinterpret_cast<sockaddr*>(&address),
                      &address_size)
        != 0) {
        return -1;
    }
    return static_cast<int>(ntohs(address.sin_port));
}

class TempTokenFile {
  public:
    explicit TempTokenFile(std::string_view contents)
    {
        static std::atomic<unsigned long> sequence {0};
        _path = "/tmp/kvm-vin-http-token-" + std::to_string(::getpid()) + '-'
            + std::to_string(sequence.fetch_add(1));

        std::ofstream file(_path, std::ios::binary | std::ios::trunc);
        if (!file.is_open()) {
            return;
        }
        file.write(contents.data(), static_cast<std::streamsize>(contents.size()));
        file.close();
        _valid = !file.fail();
    }

    ~TempTokenFile() { (void)::unlink(_path.c_str()); }

    TempTokenFile(const TempTokenFile&) = delete;
    TempTokenFile& operator=(const TempTokenFile&) = delete;

    [[nodiscard]] bool valid() const { return _valid; }
    [[nodiscard]] const std::string& path() const { return _path; }

  private:
    std::string _path;
    bool _valid {false};
};

struct CaptureFakeConfig {
    int ret_code {CaptureCoordinator::RESULT_OK};
    int width {1280};
    int height {720};
    std::vector<std::uint8_t> jpeg {0xff, 0xd8, 0xff, 0xd9};
    std::vector<std::vector<std::uint8_t>> jpeg_frames;
    bool block {false};
};

class RecordingCaptureFake {
  public:
    explicit RecordingCaptureFake(CaptureFakeConfig config = {})
        : _config(std::move(config))
    {}

    int capture(const CaptureRequest& request,
                AxeraUtils::SnapshotResult& result)
    {
        std::unique_lock lock(_mutex);
        _requests.push_back(request);
        ++_call_count;
        ++_active_count;
        _max_active = std::max(_max_active, _active_count);
        _cv.notify_all();

        if (_config.block) {
            _cv.wait(lock, [this]() { return _released; });
        }

        const int ret_code = _config.ret_code;
        const int width = _config.width;
        const int height = _config.height;
        std::vector<std::uint8_t> jpeg = _config.jpeg;
        if (!_config.jpeg_frames.empty()) {
            const auto frame_index = (_call_count - 1) % _config.jpeg_frames.size();
            jpeg = _config.jpeg_frames.at(frame_index);
        }
        --_active_count;
        _cv.notify_all();
        lock.unlock();

        if (ret_code == CaptureCoordinator::RESULT_OK) {
            result.width = width;
            result.height = height;
            result.data = jpeg;
        }
        return ret_code;
    }

    bool wait_for_calls(std::size_t count,
                        std::chrono::milliseconds timeout = 2s)
    {
        std::unique_lock lock(_mutex);
        return _cv.wait_for(
            lock, timeout, [this, count]() { return _call_count >= count; });
    }

    void release()
    {
        {
            std::lock_guard lock(_mutex);
            _released = true;
        }
        _cv.notify_all();
    }

    [[nodiscard]] std::size_t call_count() const
    {
        std::lock_guard lock(_mutex);
        return _call_count;
    }

    [[nodiscard]] std::size_t max_active() const
    {
        std::lock_guard lock(_mutex);
        return _max_active;
    }

    [[nodiscard]] std::vector<CaptureRequest> requests() const
    {
        std::lock_guard lock(_mutex);
        return _requests;
    }

  private:
    const CaptureFakeConfig _config;
    mutable std::mutex _mutex;
    std::condition_variable _cv;
    std::vector<CaptureRequest> _requests;
    std::size_t _call_count {0};
    std::size_t _active_count {0};
    std::size_t _max_active {0};
    bool _released {false};
};

class HttpHarness {
  public:
    explicit HttpHarness(CaptureFakeConfig config = {},
                         std::size_t max_clients =
                             SnapshotHttpServer::DEFAULT_MAX_CLIENTS)
        : fake(std::move(config))
        , coordinator([this](const CaptureRequest& request,
                             AxeraUtils::SnapshotResult& result) {
            return fake.capture(request, result);
        })
        , server(coordinator, logger, max_clients)
    {
        coordinator.start();
    }

    ~HttpHarness()
    {
        server.stop();
        fake.release();
        coordinator.stop();
    }

    HttpHarness(const HttpHarness&) = delete;
    HttpHarness& operator=(const HttpHarness&) = delete;

    bool start(const std::string& token_file = {},
               const std::string& bind_host = "127.0.0.1")
    {
        for (int attempt = 0; attempt < 8; ++attempt) {
            const int candidate = reserve_loopback_port();
            if (candidate <= 0) {
                continue;
            }

            snapshot_http::Config config;
            config.enabled = true;
            config.bind_host = bind_host;
            config.port = candidate;
            config.token_file = token_file;
            if (server.start(config)) {
                _port = candidate;
                return true;
            }
        }
        return false;
    }

    [[nodiscard]] int port() const { return _port; }

    RecordingCaptureFake fake;
    CaptureCoordinator coordinator;
    NullLogger logger;
    SnapshotHttpServer server;

  private:
    int _port {-1};
};

httplib::Result perform_request(int port,
                                std::string method,
                                std::string target,
                                httplib::Headers headers = {})
{
    httplib::Client client("127.0.0.1", port);
    client.set_connection_timeout(1s);
    client.set_read_timeout(2s);
    client.set_write_timeout(2s);

    if (method == "GET") {
        return client.Get(target, headers);
    }

    httplib::Request request;
    request.method = std::move(method);
    request.path = std::move(target);
    request.headers = std::move(headers);
    return client.send(request);
}

struct StreamResult {
    bool response_received {false};
    int status {0};
    httplib::Headers headers;
    std::string body;
    std::size_t frames {0};
};

StreamResult perform_stream_request(int port,
                                    std::string target,
                                    httplib::Headers headers = {},
                                    std::size_t frames = 2);

class AsyncHttpCalls {
  public:
    explicit AsyncHttpCalls(RecordingCaptureFake& fake)
        : _fake(fake)
    {}

    ~AsyncHttpCalls()
    {
        _fake.release();
        for (auto& future : _futures) {
            if (future.valid()) {
                future.wait();
            }
        }
    }

    AsyncHttpCalls(const AsyncHttpCalls&) = delete;
    AsyncHttpCalls& operator=(const AsyncHttpCalls&) = delete;

    void launch_get(int port, std::string target)
    {
        _futures.push_back(
            std::async(std::launch::async,
                       [port, target = std::move(target)]() mutable {
                           return perform_stream_request(port,
                                                        std::move(target));
                       }));
    }

    StreamResult get(std::size_t index) { return _futures.at(index).get(); }

  private:
    RecordingCaptureFake& _fake;
    std::vector<std::future<StreamResult>> _futures;
};

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

json parse_body(const httplib::Result& result)
{ return json::parse(result->body); }

std::string response_header(const StreamResult& result, std::string_view name)
{
    const auto it = result.headers.find(std::string(name));
    return it == result.headers.end() ? std::string() : it->second;
}

StreamResult perform_stream_request(int port,
                                    std::string target,
                                    httplib::Headers headers,
                                    std::size_t frames)
{
    StreamResult stream_result;
    httplib::Client client("127.0.0.1", port);
    client.set_connection_timeout(1s);
    client.set_read_timeout(2s);
    client.set_write_timeout(2s);

    auto body = std::make_shared<std::string>();
    auto frame_count = std::make_shared<std::size_t>(0);
    auto scanned = std::make_shared<std::size_t>(0);
    auto response_handler = [&stream_result](const httplib::Response& response) {
        stream_result.response_received = true;
        stream_result.status = response.status;
        stream_result.headers = response.headers;
        return true;
    };
    auto result = client.Get(
        target,
        headers,
        response_handler,
        [body, frame_count, scanned, frames](const char* data,
                                             std::size_t length) {
            body->append(data, length);
            for (;;) {
                const auto marker = body->find("\xff\xd9", *scanned);
                if (marker == std::string::npos) {
                    break;
                }
                ++*frame_count;
                *scanned = marker + 2;
            }
            return *frame_count < frames;
        });
    (void)result;
    stream_result.body = std::move(*body);
    stream_result.frames = *frame_count;
    return stream_result;
}

}  // namespace

TEST_CASE("Snapshot HTTP returns a standard multipart MJPEG stream",
          "[capture][snapshot-http]")
{
    const auto defaults =
        snapshot_http::parse_query(std::nullopt, std::nullopt);
    REQUIRE(defaults.ok());
    REQUIRE(defaults.params.quality == snapshot_http::DEFAULT_QUALITY);
    REQUIRE(defaults.params.timeout_ms == snapshot_http::DEFAULT_TIMEOUT_MS);

    const std::vector<std::uint8_t> expected_jpeg {
        0xff, 0xd8, 0x00, 0x7f, 0x80, 0xff, 0xd9,
    };
    const std::vector<std::uint8_t> second_jpeg {
        0xff, 0xd8, 0x11, 0x22, 0x33, 0xff, 0xd9,
    };
    CaptureFakeConfig fake_config;
    fake_config.width = 1920;
    fake_config.height = 1080;
    fake_config.jpeg = expected_jpeg;
    fake_config.jpeg_frames = {expected_jpeg, second_jpeg};
    HttpHarness harness(std::move(fake_config));
    REQUIRE(harness.start());

    const auto result =
        perform_stream_request(harness.port(), "/stream.mjpg", {}, 2);
    REQUIRE(result.response_received);
    REQUIRE(result.status == snapshot_http::HTTP_OK);

    const std::string expected_body(
        reinterpret_cast<const char*>(expected_jpeg.data()),
        expected_jpeg.size());
    REQUIRE(response_header(result, "Content-Type")
            == snapshot_http::MJPEG_CONTENT_TYPE);
    REQUIRE(response_header(result, "Transfer-Encoding") == "chunked");
    REQUIRE(response_header(result, "Content-Length").empty());
    REQUIRE(result.body.find(expected_body) != std::string::npos);
    const std::string expected_second_body(
        reinterpret_cast<const char*>(second_jpeg.data()), second_jpeg.size());
    REQUIRE(result.body.find(expected_second_body) != std::string::npos);
    REQUIRE(result.body.find("--frame\r\nContent-Type: image/jpeg\r\n")
            != std::string::npos);
    REQUIRE(result.body.find("Content-Length: 7\r\n\r\n")
            != std::string::npos);
    REQUIRE(result.frames >= 2);
    REQUIRE(response_header(result, "Cache-Control")
            == snapshot_http::CACHE_CONTROL_VALUE);
    REQUIRE(response_header(result, "X-Content-Type-Options")
            == snapshot_http::CONTENT_TYPE_OPTIONS_VALUE);

    const auto requests = harness.fake.requests();
    REQUIRE(requests.size() >= 2);
    const auto& request = requests.front();
    REQUIRE(request.x == 0);
    REQUIRE(request.y == 0);
    REQUIRE(request.w == 0);
    REQUIRE(request.h == 0);
    REQUIRE(request.quality == snapshot_http::DEFAULT_QUALITY);
    REQUIRE(request.timeout_ms > 0);
    REQUIRE(request.timeout_ms <= snapshot_http::DEFAULT_TIMEOUT_MS);
    REQUIRE(request.source == CaptureSource::SnapshotHttp);
}

TEST_CASE("Snapshot HTTP query contract validates boundaries and types",
          "[capture][snapshot-http][contract]")
{
    const auto lower = snapshot_http::parse_query(
        std::string_view("1"), std::string_view("1"));
    REQUIRE(lower.ok());
    REQUIRE(lower.params.quality == snapshot_http::MIN_QUALITY);
    REQUIRE(lower.params.timeout_ms == snapshot_http::MIN_TIMEOUT_MS);

    const auto upper = snapshot_http::parse_query(
        std::string_view("99"), std::string_view("30000"));
    REQUIRE(upper.ok());
    REQUIRE(upper.params.quality == snapshot_http::MAX_QUALITY);
    REQUIRE(upper.params.timeout_ms == snapshot_http::MAX_TIMEOUT_MS);

    struct QueryCase {
        const char* name;
        std::optional<std::string_view> quality;
        std::optional<std::string_view> timeout_ms;
        snapshot_http::QueryError error;
        int ret_code;
    };
    const std::vector<QueryCase> cases {
        {"empty quality",
         std::string_view(""),
         std::nullopt,
         snapshot_http::QueryError::InvalidQuality,
         -4},
        {"text quality",
         std::string_view("abc"),
         std::nullopt,
         snapshot_http::QueryError::InvalidQuality,
         -4},
        {"fractional quality",
         std::string_view("1.5"),
         std::nullopt,
         snapshot_http::QueryError::InvalidQuality,
         -4},
        {"overflow quality",
         std::string_view("999999999999999999999999"),
         std::nullopt,
         snapshot_http::QueryError::InvalidQuality,
         -4},
        {"quality below minimum",
         std::string_view("0"),
         std::nullopt,
         snapshot_http::QueryError::QualityOutOfRange,
         -11},
        {"quality above maximum",
         std::string_view("100"),
         std::nullopt,
         snapshot_http::QueryError::QualityOutOfRange,
         -11},
        {"empty timeout",
         std::nullopt,
         std::string_view(""),
         snapshot_http::QueryError::InvalidTimeout,
         -4},
        {"text timeout",
         std::nullopt,
         std::string_view("abc"),
         snapshot_http::QueryError::InvalidTimeout,
         -4},
        {"timeout below minimum",
         std::nullopt,
         std::string_view("0"),
         snapshot_http::QueryError::TimeoutOutOfRange,
         -11},
        {"timeout above maximum",
         std::nullopt,
         std::string_view("30001"),
         snapshot_http::QueryError::TimeoutOutOfRange,
         -11},
    };

    for (const auto& test_case : cases) {
        CAPTURE(test_case.name);
        const auto parsed = snapshot_http::parse_query(
            test_case.quality, test_case.timeout_ms);
        CHECK_FALSE(parsed.ok());
        CHECK(parsed.error == test_case.error);
        CHECK(parsed.ret_code == test_case.ret_code);
    }
}

TEST_CASE("Snapshot HTTP rejects invalid query strings before capture",
          "[capture][snapshot-http]")
{
    HttpHarness harness;
    REQUIRE(harness.start());

    struct InvalidQueryCase {
        const char* target;
        int ret_code;
    };
    const std::vector<InvalidQueryCase> cases {
        {"/stream.mjpg?quality=abc", -4},
        {"/stream.mjpg?quality=0", -11},
        {"/stream.mjpg?quality=100", -11},
        {"/stream.mjpg?timeout_ms=abc", -4},
        {"/stream.mjpg?timeout_ms=0", -11},
        {"/stream.mjpg?timeout_ms=30001", -11},
        {"/stream.mjpg?quality=80&quality=81", -4},
        {"/stream.mjpg?unknown=1", -4},
    };

    for (const auto& test_case : cases) {
        CAPTURE(test_case.target);
        const auto result =
            perform_request(harness.port(), "GET", test_case.target);
        REQUIRE(result);
        CHECK(result->status == snapshot_http::HTTP_BAD_REQUEST);
        const auto body = parse_body(result);
        CHECK(body.at("ok") == false);
        CHECK(body.at("ret_code") == test_case.ret_code);
    }
    REQUIRE(harness.fake.call_count() == 0);

    const auto valid = perform_stream_request(
        harness.port(), "/stream.mjpg?quality=99&timeout_ms=2000", {}, 2);
    REQUIRE(valid.response_received);
    REQUIRE(valid.status == snapshot_http::HTTP_OK);
    const auto requests = harness.fake.requests();
    REQUIRE(requests.size() >= 2);
    REQUIRE(requests.front().quality == 99);
    REQUIRE(requests.front().timeout_ms > 0);
    REQUIRE(requests.front().timeout_ms <= 2000);
}

TEST_CASE("Snapshot HTTP maps capture result codes to HTTP status",
          "[capture][snapshot-http]")
{
    struct MappingCase {
        int ret_code;
        int http_status;
    };
    const std::vector<MappingCase> contract_cases {
        {0, snapshot_http::HTTP_OK},
        {-11, snapshot_http::HTTP_BAD_REQUEST},
        {-12, snapshot_http::HTTP_BAD_REQUEST},
        {-13, snapshot_http::HTTP_GATEWAY_TIMEOUT},
        {-14, snapshot_http::HTTP_INTERNAL_SERVER_ERROR},
        {-15, snapshot_http::HTTP_INTERNAL_SERVER_ERROR},
        {-5, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-10, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-16, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-17, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-18, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-19, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-999, snapshot_http::HTTP_INTERNAL_SERVER_ERROR},
    };
    for (const auto& test_case : contract_cases) {
        CAPTURE(test_case.ret_code);
        const auto mapping =
            snapshot_http::map_capture_result(test_case.ret_code);
        CHECK(mapping.http_status == test_case.http_status);
        CHECK(mapping.ret_code == test_case.ret_code);
    }

    const std::vector<MappingCase> server_cases {
        {-13, snapshot_http::HTTP_GATEWAY_TIMEOUT},
        {-17, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-19, snapshot_http::HTTP_SERVICE_UNAVAILABLE},
        {-14, snapshot_http::HTTP_INTERNAL_SERVER_ERROR},
        {-15, snapshot_http::HTTP_INTERNAL_SERVER_ERROR},
    };
    for (const auto& test_case : server_cases) {
        CAPTURE(test_case.ret_code);
        CaptureFakeConfig config;
        config.ret_code = test_case.ret_code;
        HttpHarness harness(std::move(config));
        REQUIRE(harness.start());

        const auto result =
            perform_request(harness.port(), "GET", "/stream.mjpg");
        REQUIRE(result);
        CHECK(result->status == test_case.http_status);
        const auto body = parse_body(result);
        CHECK(body.at("ok") == false);
        CHECK(body.at("ret_code") == test_case.ret_code);
        if (test_case.http_status
            == snapshot_http::HTTP_SERVICE_UNAVAILABLE) {
            CHECK(result->get_header_value("Retry-After") == "1");
        }
    }
}

TEST_CASE("Snapshot HTTP rejects non GET methods and unknown paths",
          "[capture][snapshot-http]")
{
    HttpHarness harness;
    REQUIRE(harness.start());

    const std::vector<std::string> methods {
        "POST", "PUT", "DELETE", "HEAD", "OPTIONS",
    };
    for (const auto& method : methods) {
        CAPTURE(method);
        const auto result =
            perform_request(harness.port(), method, "/stream.mjpg");
        REQUIRE(result);
        CHECK(result->status == snapshot_http::HTTP_METHOD_NOT_ALLOWED);
        CHECK(result->get_header_value("Allow") == snapshot_http::METHOD);
        if (method != "HEAD") {
            const auto body = parse_body(result);
            CHECK(body.at("ok") == false);
        }
    }

    const auto unknown =
        perform_request(harness.port(), "GET", "/not-found");
    REQUIRE(unknown);
    REQUIRE(unknown->status == snapshot_http::HTTP_NOT_FOUND);
    REQUIRE(parse_body(unknown).at("ok") == false);

    const auto removed_snapshot =
        perform_request(harness.port(), "GET", "/snapshot.jpg");
    REQUIRE(removed_snapshot);
    REQUIRE(removed_snapshot->status == snapshot_http::HTTP_NOT_FOUND);
    REQUIRE(harness.fake.call_count() == 0);
}

TEST_CASE("Snapshot HTTP supports optional bearer token on every bind address",
          "[capture][snapshot-http][auth]")
{
    snapshot_http::Config non_loopback;
    non_loopback.enabled = true;
    non_loopback.bind_host = "0.0.0.0";
    REQUIRE(snapshot_http::validate_config(non_loopback, false).ok());

    HttpHarness unauthenticated_harness;
    REQUIRE(unauthenticated_harness.start({}, "0.0.0.0"));
    const auto unauthenticated = perform_stream_request(
        unauthenticated_harness.port(), "/stream.mjpg", {}, 2);
    REQUIRE(unauthenticated.response_received);
    REQUIRE(unauthenticated.status == snapshot_http::HTTP_OK);

    TempTokenFile token_file("test-token\r\n");
    REQUIRE(token_file.valid());
    HttpHarness harness;
    REQUIRE(harness.start(token_file.path(), "0.0.0.0"));

    const auto missing =
        perform_request(harness.port(), "GET", "/stream.mjpg");
    REQUIRE(missing);
    REQUIRE(missing->status == snapshot_http::HTTP_UNAUTHORIZED);
    REQUIRE(missing->get_header_value("WWW-Authenticate") == "Bearer");
    REQUIRE(parse_body(missing).at("ok") == false);

    httplib::Headers wrong_headers;
    wrong_headers.emplace("Authorization", "Bearer wrong-token");
    const auto wrong = perform_request(
        harness.port(), "GET", "/stream.mjpg", std::move(wrong_headers));
    REQUIRE(wrong);
    REQUIRE(wrong->status == snapshot_http::HTTP_UNAUTHORIZED);
    REQUIRE(harness.fake.call_count() == 0);

    httplib::Headers correct_headers;
    correct_headers.emplace("Authorization", "Bearer test-token");
    const auto correct = perform_stream_request(
        harness.port(), "/stream.mjpg", std::move(correct_headers), 2);
    REQUIRE(correct.response_received);
    REQUIRE(correct.status == snapshot_http::HTTP_OK);
    REQUIRE(harness.fake.call_count() >= 2);
}

TEST_CASE("Snapshot HTTP concurrent GET requests serialize capture operation",
          "[capture][snapshot-http][concurrency]")
{
    CaptureFakeConfig config;
    config.block = true;
    config.jpeg = {0xff, 0xd8, 0x42, 0xff, 0xd9};
    const std::string expected_body(
        reinterpret_cast<const char*>(config.jpeg.data()), config.jpeg.size());
    HttpHarness harness(std::move(config));
    REQUIRE(harness.start());

    AsyncHttpCalls calls(harness.fake);
    calls.launch_get(harness.port(), "/stream.mjpg?timeout_ms=2000");
    REQUIRE(harness.fake.wait_for_calls(1));

    calls.launch_get(harness.port(), "/stream.mjpg?timeout_ms=2000");
    REQUIRE(wait_until(
        [&harness]() { return harness.coordinator.queued_requests() == 1; }));

    harness.fake.release();
    const auto first = calls.get(0);
    const auto second = calls.get(1);
    REQUIRE(first.response_received);
    REQUIRE(second.response_received);
    REQUIRE(first.status == snapshot_http::HTTP_OK);
    REQUIRE(second.status == snapshot_http::HTTP_OK);
    REQUIRE(first.body.find(expected_body) != std::string::npos);
    REQUIRE(second.body.find(expected_body) != std::string::npos);
    REQUIRE(first.frames >= 2);
    REQUIRE(second.frames >= 2);
    REQUIRE(harness.fake.call_count() >= 2);
    REQUIRE(harness.fake.max_active() == 1);
}

TEST_CASE("Snapshot HTTP stop cancels an active request and can restart",
          "[capture][snapshot-http][lifecycle]")
{
    CaptureFakeConfig config;
    config.block = true;
    HttpHarness harness(std::move(config));
    REQUIRE(harness.start());

    AsyncHttpCalls calls(harness.fake);
    calls.launch_get(harness.port(), "/stream.mjpg?timeout_ms=30000");
    REQUIRE(harness.fake.wait_for_calls(1));

    const auto stop_started = std::chrono::steady_clock::now();
    auto stop_future = std::async(std::launch::async, [&harness]() {
        harness.server.stop();
    });
    std::this_thread::sleep_for(100ms);
    harness.fake.release();
    stop_future.wait();
    const auto stop_elapsed = std::chrono::steady_clock::now() - stop_started;
    REQUIRE(stop_elapsed < 2s);
    REQUIRE_FALSE(harness.server.is_running());

    const auto cancelled = calls.get(0);
    REQUIRE(cancelled.response_received);
    REQUIRE(cancelled.status == snapshot_http::HTTP_SERVICE_UNAVAILABLE);
    const auto cancelled_body = json::parse(cancelled.body);
    REQUIRE(cancelled_body.at("ok") == false);
    REQUIRE(cancelled_body.at("ret_code")
            == CaptureCoordinator::RESULT_CANCELLED);

    harness.server.stop();
    REQUIRE(harness.start());
    const auto restarted =
        perform_stream_request(harness.port(), "/stream.mjpg", {}, 2);
    REQUIRE(restarted.response_received);
    REQUIRE(restarted.status == snapshot_http::HTTP_OK);
    REQUIRE(restarted.frames >= 2);
    REQUIRE(harness.fake.call_count() >= 3);
}
