#ifndef _GNU_SOURCE
#    define _GNU_SOURCE
#endif

#include <algorithm>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <cstring>
#include <filesystem>
#include <limits>
#include <string_view>
#include <utility>

#include "snapshot_socket_server.hpp"

#include <fcntl.h>
#include <nlohmann/json.hpp>
#include <poll.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <unistd.h>

namespace {

using json = nlohmann::json;

enum class SnapshotSocketCommand {
    CaptureFresh,
    GetLatest,
};

struct ParsedSnapshotRequest {
    bool ok {false};
    int ret_code {-4};
    std::string message {"invalid request"};
    SnapshotSocketCommand command {SnapshotSocketCommand::CaptureFresh};
    CaptureRequest capture;
    std::string previous_session;
    std::uint64_t after_sequence {0};
    int wait_ms {SnapshotSocketServer::DEFAULT_LATEST_WAIT_MS};
};

struct ReadRequestResult {
    bool ok {false};
    int ret_code {-4};
    std::string message {"invalid request"};
    std::string line;
};

const char* capture_error_message(int ret_code)
{
    switch (ret_code) {
    case -5:
        return "snapshot disabled";
    case -10:
        return "no desktop";
    case -11:
        return "invalid params";
    case -12:
        return "ROI out of bounds";
    case -13:
        return "snapshot timeout";
    case -14:
        return "JPEG encode failed";
    case -15:
        return "IVPS config failed";
    case -16:
        return "snapshot busy";
    case -17:
        return "snapshot unavailable";
    case -18:
        return "snapshot cancelled";
    case -19:
        return "snapshot backend unhealthy";
    default:
        return "snapshot failed";
    }
}

json make_response_header(bool ok,
                          int ret_code,
                          std::string_view message,
                          int width = 0,
                          int height = 0,
                          std::size_t size = 0)
{
    return json {
        {"version",  SNAPSHOT_SOCKET_PROTOCOL_VERSION},
        {"ok",       ok                              },
        {"ret_code", ret_code                        },
        {"message",  message                         },
        {"width",    width                           },
        {"height",   height                          },
        {"size",     static_cast<std::uint64_t>(size)},
    };
}

bool send_all(int fd, const void* data, std::size_t size)
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
        return false;
    }
    return true;
}

bool send_response(int fd,
                   const json& header,
                   const std::uint8_t* jpeg = nullptr,
                   std::size_t jpeg_size = 0)
{
    const std::string line = header.dump() + "\n";
    if (!send_all(fd, line.data(), line.size())) {
        return false;
    }
    return jpeg_size == 0 || (jpeg != nullptr && send_all(fd, jpeg, jpeg_size));
}

void send_error_response(int fd, int ret_code, std::string_view message)
{ (void)send_response(fd, make_response_header(false, ret_code, message)); }

bool configure_client_socket(int fd)
{
    const timeval timeout {
        .tv_sec = SnapshotSocketServer::SOCKET_IO_TIMEOUT_MS / 1000,
        .tv_usec = (SnapshotSocketServer::SOCKET_IO_TIMEOUT_MS % 1000) * 1000,
    };
    return ::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout))
        == 0
        && ::setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout))
        == 0;
}

bool only_ascii_whitespace(std::string_view value)
{
    for (const char ch : value) {
        if (ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n') {
            return false;
        }
    }
    return true;
}

ReadRequestResult read_request_line(int fd)
{
    std::string buffer;
    buffer.reserve(1024);
    char chunk[4096];

    while (buffer.size() <= SnapshotSocketServer::MAX_REQUEST_HEADER_BYTES) {
        const ssize_t count = ::recv(fd, chunk, sizeof(chunk), 0);
        if (count > 0) {
            buffer.append(chunk, static_cast<std::size_t>(count));
            const auto newline = buffer.find('\n');
            if (newline != std::string::npos) {
                if (newline > SnapshotSocketServer::MAX_REQUEST_HEADER_BYTES) {
                    return {.ret_code = -4,
                            .message = "request header too large"};
                }
                if (!only_ascii_whitespace(
                        std::string_view(buffer).substr(newline + 1))) {
                    return {.ret_code = -4,
                            .message = "multiple requests are not supported"};
                }
                std::string line = buffer.substr(0, newline);
                if (!line.empty() && line.back() == '\r') {
                    line.pop_back();
                }
                return {.ok = true,
                        .ret_code = 0,
                        .message = "ok",
                        .line = std::move(line)};
            }
            if (buffer.size()
                > SnapshotSocketServer::MAX_REQUEST_HEADER_BYTES) {
                return {.ret_code = -4, .message = "request header too large"};
            }
            continue;
        }

        if (count < 0 && errno == EINTR) {
            continue;
        }
        if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            return {.ret_code = -13, .message = "request timeout"};
        }
        return {.ret_code = -4, .message = "request must end with a newline"};
    }

    return {.ret_code = -4, .message = "request header too large"};
}

bool read_json_int(const json& value, int& output)
{
    if (value.is_number_unsigned()) {
        const auto raw = value.get<std::uint64_t>();
        if (raw > static_cast<std::uint64_t>(std::numeric_limits<int>::max())) {
            return false;
        }
        output = static_cast<int>(raw);
        return true;
    }
    if (value.is_number_integer()) {
        const auto raw = value.get<std::int64_t>();
        if (raw < static_cast<std::int64_t>(std::numeric_limits<int>::min())
            || raw
                > static_cast<std::int64_t>(std::numeric_limits<int>::max())) {
            return false;
        }
        output = static_cast<int>(raw);
        return true;
    }
    return false;
}

bool read_required_int(const json& object, const char* key, int& value)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return false;
    }
    try {
        return read_json_int(*it, value);
    } catch (...) {
        return false;
    }
}

bool read_optional_int(const json& object, const char* key, int& value)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return true;
    }
    try {
        return read_json_int(*it, value);
    } catch (...) {
        return false;
    }
}

bool read_json_uint64(const json& value, std::uint64_t& output)
{
    if (value.is_number_unsigned()) {
        output = value.get<std::uint64_t>();
        return true;
    }
    if (value.is_number_integer()) {
        const auto raw = value.get<std::int64_t>();
        if (raw < 0) {
            return false;
        }
        output = static_cast<std::uint64_t>(raw);
        return true;
    }
    return false;
}

bool read_optional_uint64(const json& object,
                          const char* key,
                          std::uint64_t& value)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return true;
    }
    try {
        return read_json_uint64(*it, value);
    } catch (...) {
        return false;
    }
}

bool read_optional_string(const json& object,
                          const char* key,
                          std::string& value)
{
    const auto it = object.find(key);
    if (it == object.end()) {
        return true;
    }
    if (!it->is_string()) {
        return false;
    }
    value = it->get<std::string>();
    return true;
}

ParsedSnapshotRequest parse_snapshot_request(std::string_view line)
{
    const json request = json::parse(line, nullptr, false);
    if (request.is_discarded() || !request.is_object()) {
        return {.ret_code = -3, .message = "invalid JSON"};
    }

    int version = 0;
    if (!read_required_int(request, "version", version)) {
        return {.ret_code = -4, .message = "missing or invalid version"};
    }
    if (version != SNAPSHOT_SOCKET_PROTOCOL_VERSION) {
        return {.ret_code = -4, .message = "unsupported version"};
    }

    const auto command = request.find("cmd");
    if (command == request.end() || !command->is_string()) {
        return {.ret_code = -4, .message = "missing or invalid cmd"};
    }
    const auto& command_name = command->get_ref<const std::string&>();

    if (command_name == "get_latest") {
        ParsedSnapshotRequest parsed;
        parsed.command = SnapshotSocketCommand::GetLatest;
        if (!read_optional_string(
                request, "session_id", parsed.previous_session)) {
            return {.ret_code = -4, .message = "invalid session_id"};
        }
        if (parsed.previous_session.size() > 256) {
            return {.ret_code = -11, .message = "session_id too long"};
        }
        if (!read_optional_uint64(
                request, "after_sequence", parsed.after_sequence)) {
            return {.ret_code = -4, .message = "invalid after_sequence"};
        }
        if (!read_optional_int(request, "wait_ms", parsed.wait_ms)) {
            return {.ret_code = -4, .message = "invalid wait_ms"};
        }
        if (parsed.wait_ms < 0
            || parsed.wait_ms > SnapshotSocketServer::MAX_LATEST_WAIT_MS) {
            return {.ret_code = -11, .message = "wait_ms out of range"};
        }
        parsed.ok = true;
        parsed.ret_code = 0;
        parsed.message = "ok";
        return parsed;
    }

    if (command_name != "capture_fresh") {
        return {.ret_code = -2, .message = "unsupported cmd"};
    }

    CaptureRequest capture;
    capture.source = CaptureSource::Mcp;
    capture.timeout_ms = 1000;
    if (!read_optional_int(request, "quality", capture.quality)
        || !read_optional_int(request, "x", capture.x)
        || !read_optional_int(request, "y", capture.y)
        || !read_optional_int(request, "w", capture.w)
        || !read_optional_int(request, "h", capture.h)) {
        return {.ret_code = -4, .message = "invalid capture parameters"};
    }
    if (capture.quality < 1 || capture.quality > 100) {
        return {.ret_code = -11, .message = "invalid quality"};
    }

    if (!read_optional_int(request, "timeout_ms", capture.timeout_ms)) {
        return {.ret_code = -4, .message = "invalid timeout_ms"};
    }

    return {
        .ok = true,
        .ret_code = 0,
        .message = "ok",
        .command = SnapshotSocketCommand::CaptureFresh,
        .capture = capture,
    };
}

bool client_disconnected(int fd)
{
    pollfd descriptor {
        .fd = fd,
        .events = POLLIN,
        .revents = 0,
    };
    const int ready = ::poll(&descriptor, 1, 0);
    if (ready < 0) {
        return errno != EINTR;
    }
    return ready > 0
        && (descriptor.revents & (POLLIN | POLLERR | POLLHUP | POLLNVAL)) != 0;
}

class LatestWaiterGuard {
  public:
    explicit LatestWaiterGuard(std::atomic<std::size_t>& counter)
        : _counter(counter)
    {}

    ~LatestWaiterGuard()
    { _counter.fetch_sub(1, std::memory_order_acq_rel); }

    LatestWaiterGuard(const LatestWaiterGuard&) = delete;
    LatestWaiterGuard& operator=(const LatestWaiterGuard&) = delete;

  private:
    std::atomic<std::size_t>& _counter;
};

bool safe_remove_socket(const std::string& path, ILogger& logger)
{
    struct stat state {};
    if (::lstat(path.c_str(), &state) != 0) {
        if (errno == ENOENT) {
            return true;
        }
        logger->error("[snapshot_socket] lstat {} failed: {}",
                      path,
                      std::strerror(errno));
        return false;
    }
    if (!S_ISSOCK(state.st_mode)) {
        logger->error(
            "[snapshot_socket] refusing to replace non-socket path: {}", path);
        return false;
    }
    if (::unlink(path.c_str()) != 0) {
        logger->error("[snapshot_socket] unlink {} failed: {}",
                      path,
                      std::strerror(errno));
        return false;
    }
    return true;
}

void remove_socket_if_present(const std::string& path)
{
    struct stat state {};
    if (::lstat(path.c_str(), &state) == 0 && S_ISSOCK(state.st_mode)) {
        (void)::unlink(path.c_str());
    }
}

}  // namespace

SnapshotSocketServer::SnapshotSocketServer(
    CaptureCoordinator& capture_coordinator,
    SnapshotEnabled snapshot_enabled,
    ILogger& logger,
    std::string socket_path,
    std::size_t max_connections)
    : _capture_coordinator(capture_coordinator)
    , _snapshot_enabled(std::move(snapshot_enabled))
    , _logger(logger)
    , _socket_path(std::move(socket_path))
    , _max_connections(std::max<std::size_t>(1, max_connections))
    , _max_latest_waiters(
          _max_connections > 1 ? _max_connections - 1 : 0)
{}

SnapshotSocketServer::SnapshotSocketServer(
    CaptureCoordinator& capture_coordinator,
    OCRLatestFrameStore& ocr_latest_frame_store,
    SnapshotEnabled snapshot_enabled,
    ILogger& logger,
    std::string socket_path,
    std::size_t max_connections)
    : SnapshotSocketServer(capture_coordinator,
                           std::move(snapshot_enabled),
                           logger,
                           std::move(socket_path),
                           max_connections)
{
    _ocr_latest_frame_store = &ocr_latest_frame_store;
}

SnapshotSocketServer::~SnapshotSocketServer() { stop(); }

bool SnapshotSocketServer::start()
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    {
        std::lock_guard lock(_mutex);
        if (_running) {
            return true;
        }
    }

    const std::filesystem::path socket_path(_socket_path);
    if (socket_path.empty() || socket_path.filename().empty()) {
        _logger->error("[snapshot_socket] invalid socket path");
        return false;
    }
    if (_socket_path.size() >= sizeof(sockaddr_un::sun_path)) {
        _logger->error("[snapshot_socket] socket path is too long: {}",
                       _socket_path);
        return false;
    }

    std::error_code error;
    const auto parent = socket_path.parent_path();
    if (!parent.empty()) {
        std::filesystem::create_directories(parent, error);
        if (error) {
            _logger->error("[snapshot_socket] create {} failed: {}",
                           parent.string(),
                           error.message());
            return false;
        }
    }
    if (!safe_remove_socket(_socket_path, _logger)) {
        return false;
    }

    const int listen_fd = ::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
    if (listen_fd < 0) {
        _logger->error("[snapshot_socket] socket failed: {}",
                       std::strerror(errno));
        return false;
    }

    sockaddr_un address {};
    address.sun_family = AF_UNIX;
    std::memcpy(
        address.sun_path, _socket_path.c_str(), _socket_path.size() + 1);
    const socklen_t address_size = static_cast<socklen_t>(
        offsetof(sockaddr_un, sun_path) + _socket_path.size() + 1);
    if (::bind(listen_fd,
               reinterpret_cast<const sockaddr*>(&address),
               address_size)
        != 0) {
        _logger->error("[snapshot_socket] bind {} failed: {}",
                       _socket_path,
                       std::strerror(errno));
        ::close(listen_fd);
        return false;
    }
    if (::chmod(_socket_path.c_str(), 0600) != 0) {
        _logger->error("[snapshot_socket] chmod {} failed: {}",
                       _socket_path,
                       std::strerror(errno));
        ::close(listen_fd);
        remove_socket_if_present(_socket_path);
        return false;
    }
    if (::listen(listen_fd, static_cast<int>(_max_connections)) != 0) {
        _logger->error("[snapshot_socket] listen failed: {}",
                       std::strerror(errno));
        ::close(listen_fd);
        remove_socket_if_present(_socket_path);
        return false;
    }

    const int flags = ::fcntl(listen_fd, F_GETFL, 0);
    if (flags < 0 || ::fcntl(listen_fd, F_SETFL, flags | O_NONBLOCK) != 0) {
        _logger->error(
            "[snapshot_socket] failed to make listener nonblocking: {}",
            std::strerror(errno));
        ::close(listen_fd);
        remove_socket_if_present(_socket_path);
        return false;
    }

    {
        std::lock_guard lock(_mutex);
        _listen_fd = listen_fd;
        _stop_requested = false;
        _running = true;
    }

    try {
        _workers.reserve(_max_connections);
        for (std::size_t index = 0; index < _max_connections; ++index) {
            _workers.emplace_back(&SnapshotSocketServer::worker_loop, this);
        }
        _accept_thread =
            std::thread(&SnapshotSocketServer::accept_loop, this, listen_fd);
    } catch (const std::exception& exception) {
        _logger->error("[snapshot_socket] failed to start threads: {}",
                       exception.what());
        {
            std::lock_guard lock(_mutex);
            _stop_requested = true;
            _running = false;
            _listen_fd = -1;
        }
        ::shutdown(listen_fd, SHUT_RDWR);
        ::close(listen_fd);
        _client_cv.notify_all();
        for (auto& worker : _workers) {
            if (worker.joinable()) {
                worker.join();
            }
        }
        _workers.clear();
        remove_socket_if_present(_socket_path);
        return false;
    }

    _logger->info("[snapshot_socket] listening on {} with max {} clients",
                  _socket_path,
                  _max_connections);
    return true;
}

void SnapshotSocketServer::stop()
{
    std::lock_guard lifecycle_lock(_lifecycle_mutex);
    int listen_fd = -1;
    std::deque<int> pending_clients;
    {
        std::lock_guard lock(_mutex);
        if (!_running && !_accept_thread.joinable() && _workers.empty()) {
            return;
        }
        _stop_requested = true;
        _running = false;
        listen_fd = std::exchange(_listen_fd, -1);
        pending_clients.swap(_pending_clients);
        if (listen_fd >= 0) {
            // Wake the accept loop, but keep the descriptor open until that
            // thread has exited so the number cannot be reused underneath it.
            (void)::shutdown(listen_fd, SHUT_RDWR);
        }
        // Active descriptors are closed by their worker.  Keep shutdown under
        // the same mutex used by the worker's erase/close sequence so a
        // descriptor cannot be closed, reused by another subsystem and then
        // accidentally shut down here.
        for (const int client_fd : _active_clients) {
            (void)::shutdown(client_fd, SHUT_RDWR);
        }
    }

    for (const int client_fd : pending_clients) {
        (void)::shutdown(client_fd, SHUT_RDWR);
        (void)::close(client_fd);
    }
    _client_cv.notify_all();

    if (_accept_thread.joinable()) {
        _accept_thread.join();
    }
    if (listen_fd >= 0) {
        (void)::close(listen_fd);
    }
    for (auto& worker : _workers) {
        if (worker.joinable()) {
            worker.join();
        }
    }
    _workers.clear();

    {
        std::lock_guard lock(_mutex);
        _active_clients.clear();
        _pending_clients.clear();
    }
    remove_socket_if_present(_socket_path);
}

bool SnapshotSocketServer::is_running() const
{
    std::lock_guard lock(_mutex);
    return _running;
}

bool SnapshotSocketServer::stopping() const
{
    std::lock_guard lock(_mutex);
    return _stop_requested;
}

void SnapshotSocketServer::accept_loop(int listen_fd)
{
    pollfd descriptor {
        .fd = listen_fd,
        .events = POLLIN,
        .revents = 0,
    };

    while (!stopping()) {
        descriptor.revents = 0;
        const int ready = ::poll(&descriptor, 1, 200);
        if (ready < 0) {
            if (errno == EINTR) {
                continue;
            }
            if (!stopping()) {
                _logger->error("[snapshot_socket] poll failed: {}",
                               std::strerror(errno));
            }
            break;
        }
        if (ready == 0) {
            continue;
        }
        if ((descriptor.revents & (POLLERR | POLLHUP | POLLNVAL)) != 0) {
            break;
        }
        if ((descriptor.revents & POLLIN) == 0) {
            continue;
        }

        while (!stopping()) {
            const int client_fd =
                ::accept4(listen_fd, nullptr, nullptr, SOCK_CLOEXEC);
            if (client_fd < 0) {
                if (errno == EINTR) {
                    continue;
                }
                if (errno == EAGAIN || errno == EWOULDBLOCK) {
                    break;
                }
                if (!stopping()) {
                    _logger->warn("[snapshot_socket] accept failed: {}",
                                  std::strerror(errno));
                }
                break;
            }

            if (!configure_client_socket(client_fd)) {
                _logger->warn(
                    "[snapshot_socket] failed to set client timeout: {}",
                    std::strerror(errno));
                ::close(client_fd);
                continue;
            }

            bool accepted = false;
            {
                std::lock_guard lock(_mutex);
                if (!_stop_requested
                    && _active_clients.size() + _pending_clients.size()
                        < _max_connections) {
                    _pending_clients.push_back(client_fd);
                    accepted = true;
                }
            }

            if (!accepted) {
                send_error_response(client_fd, -16, "snapshot server busy");
                (void)::shutdown(client_fd, SHUT_RDWR);
                ::close(client_fd);
                continue;
            }
            _client_cv.notify_one();
        }
    }
}

void SnapshotSocketServer::worker_loop()
{
    while (true) {
        int client_fd = -1;
        {
            std::unique_lock lock(_mutex);
            _client_cv.wait(lock, [this]() {
                return _stop_requested || !_pending_clients.empty();
            });
            if (_stop_requested && _pending_clients.empty()) {
                return;
            }
            client_fd = _pending_clients.front();
            _pending_clients.pop_front();
            _active_clients.insert(client_fd);
        }

        try {
            handle_client(client_fd);
        } catch (const std::exception& exception) {
            _logger->error("[snapshot_socket] client handler failed: {}",
                           exception.what());
            send_error_response(client_fd, -17, "snapshot unavailable");
        } catch (...) {
            _logger->error("[snapshot_socket] client handler failed");
            send_error_response(client_fd, -17, "snapshot unavailable");
        }

        {
            std::lock_guard lock(_mutex);
            _active_clients.erase(client_fd);
            (void)::shutdown(client_fd, SHUT_RDWR);
            ::close(client_fd);
        }
    }
}

void SnapshotSocketServer::handle_client(int client_fd)
{
    const auto input = read_request_line(client_fd);
    if (!input.ok) {
        send_error_response(client_fd, input.ret_code, input.message);
        return;
    }

    const auto parsed = parse_snapshot_request(input.line);
    if (!parsed.ok) {
        send_error_response(client_fd, parsed.ret_code, parsed.message);
        return;
    }

    if (parsed.command == SnapshotSocketCommand::GetLatest) {
        handle_get_latest(client_fd,
                          parsed.previous_session,
                          parsed.after_sequence,
                          parsed.wait_ms);
        return;
    }

    handle_capture_fresh(client_fd, parsed.capture);
}

void SnapshotSocketServer::handle_capture_fresh(int client_fd,
                                                CaptureRequest request)
{
    bool snapshot_enabled = false;
    try {
        snapshot_enabled = _snapshot_enabled && _snapshot_enabled();
    } catch (...) {
        snapshot_enabled = false;
    }
    if (!snapshot_enabled) {
        send_error_response(client_fd, -5, "snapshot disabled");
        return;
    }

    auto ticket = _capture_coordinator.submit(std::move(request));
    std::atomic<bool> finished {false};
    std::jthread disconnect_monitor(
        [&ticket, &finished, client_fd](std::stop_token stop_token) {
            pollfd descriptor {
                .fd = client_fd,
                .events = POLLIN,
                .revents = 0,
            };
            while (!stop_token.stop_requested()
                   && !finished.load(std::memory_order_acquire)) {
                descriptor.revents = 0;
                const int ready = ::poll(&descriptor, 1, 20);
                if (ready < 0 && errno == EINTR) {
                    continue;
                }
                if (ready < 0
                    || (ready > 0
                        && (descriptor.revents
                            & (POLLIN | POLLERR | POLLHUP | POLLNVAL))
                            != 0)) {
                    ticket.cancel();
                    return;
                }
            }
        });

    const CaptureResponse response = ticket.wait();
    finished.store(true, std::memory_order_release);
    disconnect_monitor.request_stop();
    disconnect_monitor.join();

    if (!response.ok()) {
        send_error_response(client_fd,
                            response.ret_code,
                            capture_error_message(response.ret_code));
        return;
    }

    const auto& frame = *response.frame;
    if (frame.width <= 0 || frame.height <= 0 || frame.data.empty()
        || frame.data.size() > MAX_JPEG_BYTES || response.session_id.empty()
        || response.capture_id.empty() || response.captured_at.empty()) {
        send_error_response(client_fd, -14, "invalid JPEG response");
        return;
    }

    json header = make_response_header(
        true, 0, "ok", frame.width, frame.height, frame.data.size());
    header["capture_id"] = response.capture_id;
    header["captured_at"] = response.captured_at;
    header["session_id"] = response.session_id;
    header["source"] = "mcp";
    (void)send_response(
        client_fd, header, frame.data.data(), frame.data.size());
}

void SnapshotSocketServer::handle_get_latest(int client_fd,
                                             std::string previous_session,
                                             std::uint64_t after_sequence,
                                             int wait_ms)
{
    if (_ocr_latest_frame_store == nullptr) {
        send_error_response(client_fd, -17, "latest frame unavailable");
        return;
    }

    const std::size_t active =
        _active_latest_waiters.fetch_add(1, std::memory_order_acq_rel);
    if (active >= _max_latest_waiters) {
        _active_latest_waiters.fetch_sub(1, std::memory_order_acq_rel);
        send_error_response(client_fd, -16, "latest frame server busy");
        return;
    }
    LatestWaiterGuard waiter_guard(_active_latest_waiters);

    const auto deadline =
        std::chrono::steady_clock::now() + std::chrono::milliseconds(wait_ms);
    OCRLatestFrameResult result;

    while (true) {
        if (stopping() || client_disconnected(client_fd)) {
            return;
        }

        const auto now = std::chrono::steady_clock::now();
        const auto remaining = deadline > now
            ? std::chrono::duration_cast<std::chrono::milliseconds>(deadline
                                                                    - now)
            : std::chrono::milliseconds::zero();
        const auto wait_slice = std::min(
            remaining,
            std::chrono::milliseconds(LATEST_WAIT_SLICE_MS));
        result = _ocr_latest_frame_store->get_latest(
            previous_session, after_sequence, wait_slice);

        if (result.status != OCRLatestFrameStatus::NotModified
            || wait_ms == 0 || std::chrono::steady_clock::now() >= deadline) {
            break;
        }
    }

    if (result.status == OCRLatestFrameStatus::Stopped) {
        send_error_response(client_fd, -17, "latest frame unavailable");
        return;
    }

    if (result.status == OCRLatestFrameStatus::NotModified) {
        json header = make_response_header(true, 0, "not_modified");
        header["not_modified"] = true;
        if (!result.session_id.empty()) {
            header["session_id"] = result.session_id;
            header["sequence"] = result.sequence;
        }
        (void)send_response(client_fd, header);
        return;
    }

    const auto& latest = result.latest;
    if (!latest || !latest->frame || latest->session_id.empty()
        || latest->capture_id.empty() || latest->captured_at.empty()) {
        send_error_response(client_fd, -17, "invalid latest frame");
        return;
    }

    const auto& frame = *latest->frame;
    if (frame.width <= 0 || frame.height <= 0 || frame.data.empty()
        || frame.data.size() > MAX_JPEG_BYTES) {
        send_error_response(client_fd, -14, "invalid JPEG response");
        return;
    }

    json header = make_response_header(
        true, 0, "ok", frame.width, frame.height, frame.data.size());
    header["capture_id"] = latest->capture_id;
    header["captured_at"] = latest->captured_at;
    header["session_id"] = latest->session_id;
    header["sequence"] = latest->sequence;
    header["source"] = "canonical";
    (void)send_response(
        client_fd, header, frame.data.data(), frame.data.size());
}
