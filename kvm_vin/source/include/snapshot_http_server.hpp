#pragma once

#include <atomic>
#include <cstddef>
#include <memory>
#include <mutex>
#include <string>
#include <thread>

#include <logger/ilogger.hpp>

#include "capture_coordinator.hpp"
#include "snapshot_http_contract.hpp"

namespace httplib {
class DataSink;
class Request;
class Response;
class Server;
}  // namespace httplib

class SnapshotHttpServer {
  public:
    static constexpr std::size_t DEFAULT_MAX_CLIENTS = 2;
    static constexpr std::size_t MAX_JPEG_BYTES = 32 * 1024 * 1024;
    static constexpr std::size_t MAX_REQUEST_BODY_BYTES = 1024;
    static constexpr int SOCKET_IO_TIMEOUT_MS = 5000;
    static constexpr int KEEP_ALIVE_TIMEOUT_SECONDS = 2;
    static constexpr int IDLE_INTERVAL_MS = 100;
    static constexpr int DISCONNECT_POLL_MS = 20;
    static constexpr int STREAM_FRAME_INTERVAL_MS = 0;

    SnapshotHttpServer(CaptureCoordinator& capture_coordinator,
                       ILogger& logger,
                       std::size_t max_clients = DEFAULT_MAX_CLIENTS);
    ~SnapshotHttpServer();

    SnapshotHttpServer(const SnapshotHttpServer&) = delete;
    SnapshotHttpServer& operator=(const SnapshotHttpServer&) = delete;

    bool start(const snapshot_http::Config& config);
    void stop();

    [[nodiscard]] bool is_running() const;

  private:
    struct StartupState;

    void configure_server(std::string bearer_token,
                          const std::shared_ptr<StartupState>& startup_state);
    void handle_stream(const httplib::Request& request,
                       httplib::Response& response,
                       const std::string& bearer_token);
    [[nodiscard]] CaptureResponse capture_frame(
        const httplib::Request& request,
        const snapshot_http::RequestParams& params);
    [[nodiscard]] bool write_frame(
        const std::shared_ptr<const AxeraUtils::SnapshotResult>& frame,
        httplib::DataSink& sink) const;
    void run_loop(std::string bind_host,
                  int port,
                  std::shared_ptr<StartupState> startup_state);
    [[nodiscard]] bool claim_stop_request();

    CaptureCoordinator& _capture_coordinator;
    ILogger& _logger;
    const std::size_t _max_clients;

    mutable std::mutex _lifecycle_mutex;
    std::mutex _admission_mutex;
    std::unique_ptr<httplib::Server> _server;
    std::thread _thread;
    std::atomic<bool> _running {false};
    std::atomic<bool> _stop_requested {false};
};
