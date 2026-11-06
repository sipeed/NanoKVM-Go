#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <deque>
#include <exception>
#include <fstream>
#include <functional>
#include <iterator>
#include <memory>
#include <optional>
#include <stdexcept>
#include <string>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include "snapshot_http_server.hpp"

#include <httplib.h>
#include <nlohmann/json.hpp>

namespace {

using json = nlohmann::json;

constexpr std::size_t MAX_TOKEN_FILE_BYTES = 4096;
constexpr std::string_view STREAM_ROUTE_PATTERN = R"(^/stream[.]mjpg$)";

std::string string_from(std::string_view value) { return std::string(value); }

void set_common_response_headers(httplib::Response& response)
{
    response.set_header(string_from(snapshot_http::CACHE_CONTROL_HEADER),
                        string_from(snapshot_http::CACHE_CONTROL_VALUE));
    response.set_header(string_from(snapshot_http::CONTENT_TYPE_OPTIONS_HEADER),
                        string_from(snapshot_http::CONTENT_TYPE_OPTIONS_VALUE));
}

void set_json_error(httplib::Response& response,
                    int status,
                    std::string_view message,
                    std::optional<int> ret_code = std::nullopt)
{
    json body {
        {"ok",      false  },
        {"message", message},
    };
    if (ret_code.has_value()) {
        body["ret_code"] = *ret_code;
    }

    response.status = status;
    set_common_response_headers(response);
    if (status == snapshot_http::HTTP_SERVICE_UNAVAILABLE) {
        response.set_header("Retry-After", "1");
    }
    response.set_content(body.dump(),
                         string_from(snapshot_http::JSON_CONTENT_TYPE));
}

bool load_bearer_token(const std::string& path,
                       std::string& token,
                       std::string& error)
{
    token.clear();
    if (path.empty()) {
        return true;
    }

    std::ifstream file(path, std::ios::binary);
    if (!file.is_open()) {
        error = "failed to open bearer token file";
        return false;
    }

    file.seekg(0, std::ios::end);
    const std::streamoff length = file.tellg();
    if (length <= 0
        || length > static_cast<std::streamoff>(MAX_TOKEN_FILE_BYTES)) {
        error = "bearer token file must contain 1..4096 bytes";
        return false;
    }

    token.resize(static_cast<std::size_t>(length));
    file.seekg(0, std::ios::beg);
    file.read(token.data(), static_cast<std::streamsize>(token.size()));
    if (!file || file.gcount() != static_cast<std::streamsize>(token.size())) {
        token.clear();
        error = "failed to read bearer token file";
        return false;
    }

    while (!token.empty() && (token.back() == '\n' || token.back() == '\r')) {
        token.pop_back();
    }
    if (token.empty()) {
        error = "bearer token file is empty";
        return false;
    }

    const bool contains_invalid_byte =
        std::any_of(token.begin(), token.end(), [](char value) {
            const auto byte = static_cast<unsigned char>(value);
            return byte <= 0x20U || byte == 0x7FU;
        });
    if (contains_invalid_byte) {
        token.clear();
        error = "bearer token must not contain whitespace or control bytes";
        return false;
    }
    return true;
}

class ManagedThreadPool final : public httplib::TaskQueue {
  public:
    ManagedThreadPool(std::size_t workers, std::function<void()> on_abandon)
        : _capacity(std::max<std::size_t>(1, workers))
        , _on_abandon(std::move(on_abandon))
    {
        try {
            _workers.reserve(_capacity);
            for (std::size_t index = 0; index < _capacity; ++index) {
                _workers.emplace_back(&ManagedThreadPool::worker_loop, this);
            }
        } catch (...) {
            {
                std::lock_guard lock(_mutex);
                _stopping = true;
                _shutdown_started = true;
            }
            _cv.notify_all();
            join_workers();
            throw;
        }
    }

    ~ManagedThreadPool() override
    {
        bool needs_shutdown = false;
        {
            std::lock_guard lock(_mutex);
            needs_shutdown = !_shutdown_started;
        }
        if (needs_shutdown) {
            abandon();
            shutdown();
        }
    }

    ManagedThreadPool(const ManagedThreadPool&) = delete;
    ManagedThreadPool& operator=(const ManagedThreadPool&) = delete;

    bool enqueue(std::function<void()> task) override
    {
        {
            std::lock_guard lock(_mutex);
            if (_stopping || _in_flight >= _capacity) {
                return false;
            }
            _jobs.push_back(std::move(task));
            ++_in_flight;
        }
        _cv.notify_one();
        return true;
    }

    void shutdown() override
    {
        {
            std::lock_guard lock(_mutex);
            if (_shutdown_started) {
                return;
            }
            _stopping = true;
            _shutdown_started = true;
        }
        _cv.notify_all();
        join_workers();
    }

  private:
    void worker_loop()
    {
        for (;;) {
            std::function<void()> task;
            {
                std::unique_lock lock(_mutex);
                _cv.wait(lock,
                         [this]() { return _stopping || !_jobs.empty(); });
                if (_stopping && _jobs.empty()) {
                    return;
                }
                task = std::move(_jobs.front());
                _jobs.pop_front();
            }

            try {
                task();
            } catch (...) {
                abandon();
            }

            std::lock_guard lock(_mutex);
            --_in_flight;
        }
    }

    void join_workers()
    {
        for (auto& worker : _workers) {
            if (worker.joinable()) {
                worker.join();
            }
        }
    }

    void abandon() noexcept
    {
        try {
            if (_on_abandon) {
                _on_abandon();
            }
        } catch (...) {}
    }

    const std::size_t _capacity;
    std::function<void()> _on_abandon;
    std::mutex _mutex;
    std::condition_variable _cv;
    std::deque<std::function<void()>> _jobs;
    std::vector<std::thread> _workers;
    std::size_t _in_flight {0};
    bool _stopping {false};
    bool _shutdown_started {false};
};

}  // namespace

struct SnapshotHttpServer::StartupState {
    std::mutex mutex;
    std::condition_variable cv;
    bool reported {false};
    bool ready {false};

    void report(bool value)
    {
        {
            std::lock_guard lock(mutex);
            if (reported) {
                return;
            }
            ready = value;
            reported = true;
        }
        cv.notify_all();
    }

    void wait(bool& value)
    {
        std::unique_lock lock(mutex);
        cv.wait(lock, [this]() { return reported; });
        value = ready;
    }
};

SnapshotHttpServer::SnapshotHttpServer(CaptureCoordinator& capture_coordinator,
                                       ILogger& logger,
                                       std::size_t max_clients)
    : _capture_coordinator(capture_coordinator)
    , _logger(logger)
    , _max_clients(std::max<std::size_t>(1, max_clients))
{}

SnapshotHttpServer::~SnapshotHttpServer() { stop(); }

bool SnapshotHttpServer::start(const snapshot_http::Config& config)
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    if (_running.load(std::memory_order_acquire) || _thread.joinable()) {
        _logger->error("[snapshot_http] server is already running");
        return false;
    }
    if (!config.enabled) {
        _logger->error("[snapshot_http] refused disabled configuration");
        return false;
    }

    const auto validation = snapshot_http::validate_config(config, false);
    if (!validation.ok()) {
        _logger->error("[snapshot_http] refused configuration: {}",
                       validation.message);
        return false;
    }

    std::string bearer_token;
    std::string token_error;
    if (!load_bearer_token(config.token_file, bearer_token, token_error)) {
        _logger->error("[snapshot_http] {}", token_error);
        return false;
    }
    auto startup_state = std::make_shared<StartupState>();
    {
        std::lock_guard admission_lock(_admission_mutex);
        _stop_requested.store(false, std::memory_order_release);
    }

    try {
        _server = std::make_unique<httplib::Server>();
        configure_server(std::move(bearer_token), startup_state);
        _thread = std::thread(&SnapshotHttpServer::run_loop,
                              this,
                              config.bind_host,
                              config.port,
                              startup_state);
    } catch (const std::exception& error) {
        _server.reset();
        _logger->error("[snapshot_http] failed to start: {}", error.what());
        return false;
    } catch (...) {
        _server.reset();
        _logger->error("[snapshot_http] failed to start with unknown error");
        return false;
    }

    bool ready = false;
    startup_state->wait(ready);
    if (!ready) {
        if (claim_stop_request() && _server) {
            _server->stop();
        }
        if (_thread.joinable()) {
            _thread.join();
        }
        _running.store(false, std::memory_order_release);
        _server.reset();
        return false;
    }

    if (!_server->is_running()) {
        if (_thread.joinable()) {
            _thread.join();
        }
        _running.store(false, std::memory_order_release);
        _server.reset();
        return false;
    }
    _running.store(true, std::memory_order_release);
    if (!_server->is_running()) {
        _running.store(false, std::memory_order_release);
        if (_thread.joinable()) {
            _thread.join();
        }
        _server.reset();
        return false;
    }

    _logger->info("[snapshot_http] listening on {}:{} with max {} clients",
                  config.bind_host,
                  config.port,
                  _max_clients);
    return true;
}

void SnapshotHttpServer::stop()
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    if (claim_stop_request() && _server) {
        _server->stop();
    }
    if (_thread.joinable()) {
        _thread.join();
    }
    _running.store(false, std::memory_order_release);
    _server.reset();
}

bool SnapshotHttpServer::is_running() const
{ return _running.load(std::memory_order_acquire); }

void SnapshotHttpServer::configure_server(
    std::string bearer_token,
    const std::shared_ptr<StartupState>& startup_state)
{
    const std::size_t max_clients = _max_clients;
    auto task_queue = std::make_shared<std::unique_ptr<httplib::TaskQueue>>(
        std::make_unique<ManagedThreadPool>(max_clients, [this]() {
            if (claim_stop_request() && _server) {
                _server->stop();
                _server->decommission();
            }
        }));
    _server->new_task_queue = [task_queue, startup_state]() {
        if (!*task_queue) {
            startup_state->report(false);
            throw std::runtime_error("HTTP task queue was already consumed");
        }
        auto* queue = task_queue->release();
        startup_state->report(true);
        return queue;
    };
    _server->set_read_timeout(std::chrono::milliseconds(SOCKET_IO_TIMEOUT_MS));
    _server->set_write_timeout(std::chrono::milliseconds(SOCKET_IO_TIMEOUT_MS));
    _server->set_idle_interval(std::chrono::milliseconds(IDLE_INTERVAL_MS));
    _server->set_keep_alive_timeout(KEEP_ALIVE_TIMEOUT_SECONDS);
    _server->set_keep_alive_max_count(1);
    _server->set_payload_max_length(MAX_REQUEST_BODY_BYTES);

    _server->set_pre_routing_handler([](const httplib::Request& request,
                                        httplib::Response& response) {
        if (request.path == snapshot_http::PATH
            && request.method != snapshot_http::METHOD) {
            set_json_error(response,
                           snapshot_http::HTTP_METHOD_NOT_ALLOWED,
                           "method not allowed");
            response.set_header("Allow", string_from(snapshot_http::METHOD));
            return httplib::Server::HandlerResponse::Handled;
        }
        return httplib::Server::HandlerResponse::Unhandled;
    });

    _server->Get(
        string_from(STREAM_ROUTE_PATTERN),
        [this, bearer_token = std::move(bearer_token)](
            const httplib::Request& request, httplib::Response& response) {
            handle_stream(request, response, bearer_token);
        });

    _server->set_error_handler(
        [](const httplib::Request&, httplib::Response& response) {
            if (response.status == snapshot_http::HTTP_NOT_FOUND
                && response.body.empty()) {
                set_json_error(
                    response, snapshot_http::HTTP_NOT_FOUND, "not found");
                return httplib::Server::HandlerResponse::Handled;
            }
            return httplib::Server::HandlerResponse::Unhandled;
        });

    _server->set_exception_handler([this](const httplib::Request&,
                                          httplib::Response& response,
                                          std::exception_ptr error) {
        try {
            if (error) {
                std::rethrow_exception(error);
            }
        } catch (const std::exception& exception) {
            _logger->error("[snapshot_http] request handler failed: {}",
                           exception.what());
        } catch (...) {
            _logger->error(
                "[snapshot_http] request handler failed with unknown error");
        }
        set_json_error(response,
                       snapshot_http::HTTP_INTERNAL_SERVER_ERROR,
                       "internal server error");
    });
}

void SnapshotHttpServer::handle_stream(const httplib::Request& request,
                                       httplib::Response& response,
                                       const std::string& bearer_token)
{
    if (_stop_requested.load(std::memory_order_acquire)) {
        set_json_error(response,
                       snapshot_http::HTTP_SERVICE_UNAVAILABLE,
                       "MJPEG server is stopping",
                       CaptureCoordinator::RESULT_UNAVAILABLE);
        return;
    }

    if (!bearer_token.empty()
        && (request.get_header_value_count(
                string_from(snapshot_http::AUTHORIZATION_HEADER))
                != 1
            || !snapshot_http::is_authorized_bearer(
                request.get_header_value(
                    string_from(snapshot_http::AUTHORIZATION_HEADER)),
                bearer_token))) {
        set_json_error(
            response, snapshot_http::HTTP_UNAUTHORIZED, "unauthorized");
        response.set_header("WWW-Authenticate", "Bearer");
        return;
    }

    if (request.has_header("Range")) {
        set_json_error(response,
                       snapshot_http::HTTP_BAD_REQUEST,
                       "range requests are not supported",
                       -4);
        return;
    }

    const bool has_unknown_query = std::any_of(
        request.params.begin(), request.params.end(), [](const auto& item) {
            return item.first != "quality" && item.first != "timeout_ms";
        });
    if (has_unknown_query) {
        set_json_error(response,
                       snapshot_http::HTTP_BAD_REQUEST,
                       "unknown query parameter",
                       -4);
        return;
    }

    if (request.get_param_value_count("quality") > 1
        || request.get_param_value_count("timeout_ms") > 1) {
        set_json_error(response,
                       snapshot_http::HTTP_BAD_REQUEST,
                       "query parameters must not be repeated",
                       -4);
        return;
    }

    std::optional<std::string> quality;
    std::optional<std::string> timeout_ms;
    if (request.has_param("quality")) {
        quality = request.get_param_value("quality");
    }
    if (request.has_param("timeout_ms")) {
        timeout_ms = request.get_param_value("timeout_ms");
    }

    const auto parsed = snapshot_http::parse_query(
        quality.has_value() ? std::optional<std::string_view>(*quality)
                            : std::nullopt,
        timeout_ms.has_value() ? std::optional<std::string_view>(*timeout_ms)
                               : std::nullopt);
    if (!parsed.ok()) {
        set_json_error(response,
                       snapshot_http::HTTP_BAD_REQUEST,
                       parsed.message,
                       parsed.ret_code);
        return;
    }

    CaptureResponse capture_response = capture_frame(request, parsed.params);
    if (!capture_response.ok()) {
        const int ret_code =
            capture_response.ret_code == 0 ? -14 : capture_response.ret_code;
        const auto mapping = snapshot_http::map_capture_result(ret_code);
        set_json_error(
            response, mapping.http_status, mapping.message, mapping.ret_code);
        return;
    }

    const auto& first_frame = *capture_response.frame;
    if (first_frame.width <= 0 || first_frame.height <= 0
        || first_frame.data.empty()
        || first_frame.data.size() > MAX_JPEG_BYTES) {
        const auto mapping = snapshot_http::map_capture_result(-14);
        set_json_error(response,
                       mapping.http_status,
                       "invalid JPEG response",
                       mapping.ret_code);
        return;
    }

    response.status = snapshot_http::HTTP_OK;
    set_common_response_headers(response);

    auto pending_frame = capture_response.frame;
    const auto params = parsed.params;
    response.set_chunked_content_provider(
        string_from(snapshot_http::MJPEG_CONTENT_TYPE),
        [this, &request, params, pending_frame = std::move(pending_frame)](
            std::size_t, httplib::DataSink& sink) mutable {
            if (_stop_requested.load(std::memory_order_acquire)
                || (sink.is_writable && !sink.is_writable())) {
                return false;
            }

            auto frame = std::move(pending_frame);
            if (!frame) {
                CaptureResponse next = capture_frame(request, params);
                if (!next.ok()) {
                    if (next.ret_code != CaptureCoordinator::RESULT_CANCELLED
                        && next.ret_code
                            != CaptureCoordinator::RESULT_UNAVAILABLE) {
                        _logger->warn(
                            "[snapshot_http] MJPEG stream capture failed, "
                            "ret={}; closing stream",
                            next.ret_code);
                    }
                    return false;
                }

                frame = std::move(next.frame);
                if (frame->width <= 0 || frame->height <= 0
                    || frame->data.empty()
                    || frame->data.size() > MAX_JPEG_BYTES) {
                    _logger->error(
                        "[snapshot_http] invalid JPEG frame in MJPEG stream");
                    return false;
                }
            }

            const bool written = write_frame(frame, sink);
            if (written && STREAM_FRAME_INTERVAL_MS > 0) {
                std::this_thread::sleep_for(
                    std::chrono::milliseconds(STREAM_FRAME_INTERVAL_MS));
            }
            return written;
        });
}

CaptureResponse SnapshotHttpServer::capture_frame(
    const httplib::Request& request, const snapshot_http::RequestParams& params)
{
    CaptureRequest capture_request {
        .x = 0,
        .y = 0,
        .w = 0,
        .h = 0,
        .quality = params.quality,
        .timeout_ms = params.timeout_ms,
        .source = CaptureSource::SnapshotHttp,
    };

    CaptureTicket ticket;
    {
        std::lock_guard admission_lock(_admission_mutex);
        if (_stop_requested.load(std::memory_order_acquire)) {
            return CaptureResponse {
                .ret_code = CaptureCoordinator::RESULT_UNAVAILABLE,
            };
        }
        ticket = _capture_coordinator.submit(std::move(capture_request));
    }
    std::atomic<bool> finished {false};
    std::jthread disconnect_monitor(
        [this, &request, &ticket, &finished](std::stop_token stop_token) {
            while (!stop_token.stop_requested()
                   && !finished.load(std::memory_order_acquire)) {
                bool disconnected = false;
                try {
                    disconnected = request.is_connection_closed();
                } catch (...) {
                    disconnected = true;
                }
                if (_stop_requested.load(std::memory_order_acquire)
                    || disconnected) {
                    ticket.cancel();
                    return;
                }
                std::this_thread::sleep_for(
                    std::chrono::milliseconds(DISCONNECT_POLL_MS));
            }
        });

    CaptureResponse capture_response = ticket.wait();
    finished.store(true, std::memory_order_release);
    disconnect_monitor.request_stop();
    disconnect_monitor.join();
    return capture_response;
}

bool SnapshotHttpServer::write_frame(
    const std::shared_ptr<const AxeraUtils::SnapshotResult>& frame,
    httplib::DataSink& sink) const
{
    if (!frame || frame->data.empty() || frame->data.size() > MAX_JPEG_BYTES) {
        return false;
    }

    const std::string part_header = "--"
        + string_from(snapshot_http::MJPEG_BOUNDARY) + "\r\n"
        + string_from(snapshot_http::CONTENT_TYPE_HEADER) + ": "
        + string_from(snapshot_http::JPEG_CONTENT_TYPE) + "\r\n"
        + string_from(snapshot_http::CONTENT_LENGTH_HEADER) + ": "
        + std::to_string(frame->data.size()) + "\r\n\r\n";
    if (!sink.write(part_header.data(), part_header.size())) {
        return false;
    }
    if (!sink.write(reinterpret_cast<const char*>(frame->data.data()),
                    frame->data.size())) {
        return false;
    }
    constexpr std::string_view PART_END = "\r\n";
    return sink.write(PART_END.data(), PART_END.size());
}

void SnapshotHttpServer::run_loop(std::string bind_host,
                                  int port,
                                  std::shared_ptr<StartupState> startup_state)
{
    try {
        if (!_server->bind_to_port(bind_host, port)) {
            _logger->error(
                "[snapshot_http] failed to bind {}:{}", bind_host, port);
            startup_state->report(false);
        } else {
            if (!_server->listen_after_bind()
                && !_stop_requested.load(std::memory_order_acquire)) {
                _logger->error("[snapshot_http] listen loop failed on {}:{}",
                               bind_host,
                               port);
            }
        }
    } catch (const std::exception& error) {
        _logger->error("[snapshot_http] server failed on {}:{}: {}",
                       bind_host,
                       port,
                       error.what());
        startup_state->report(false);
    } catch (...) {
        _logger->error(
            "[snapshot_http] server failed on {}:{} with unknown error",
            bind_host,
            port);
        startup_state->report(false);
    }

    startup_state->report(false);
    _running.store(false, std::memory_order_release);
    _logger->info("[snapshot_http] server stopped");
}

bool SnapshotHttpServer::claim_stop_request()
{
    std::lock_guard admission_lock(_admission_mutex);
    return !_stop_requested.exchange(true, std::memory_order_acq_rel);
}
