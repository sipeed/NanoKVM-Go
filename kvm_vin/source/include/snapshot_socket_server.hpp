#pragma once

#include <atomic>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <functional>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_set>
#include <vector>

#include <logger/ilogger.hpp>

#include "capture_coordinator.hpp"
#include "ocr_latest_frame_store.hpp"

inline constexpr int SNAPSHOT_SOCKET_PROTOCOL_VERSION = 1;
inline constexpr const char* KVM_VIN_SNAPSHOT_SOCK_PATH =
    "/run/kvm/vin_snapshot.sock";

class SnapshotSocketServer {
  public:
    using SnapshotEnabled = std::function<bool()>;

    static constexpr std::size_t DEFAULT_MAX_CONNECTIONS = 8;
    static constexpr std::size_t MAX_REQUEST_HEADER_BYTES = 16 * 1024;
    static constexpr std::size_t MAX_JPEG_BYTES = 32 * 1024 * 1024;
    static constexpr int SOCKET_IO_TIMEOUT_MS = 5000;
    static constexpr int DEFAULT_LATEST_WAIT_MS = 15000;
    static constexpr int MAX_LATEST_WAIT_MS = 30000;
    static constexpr int LATEST_WAIT_SLICE_MS = 200;

    SnapshotSocketServer(CaptureCoordinator& capture_coordinator,
                         SnapshotEnabled snapshot_enabled,
                         ILogger& logger,
                         std::string socket_path = KVM_VIN_SNAPSHOT_SOCK_PATH,
                         std::size_t max_connections = DEFAULT_MAX_CONNECTIONS);
    SnapshotSocketServer(CaptureCoordinator& capture_coordinator,
                         OCRLatestFrameStore& ocr_latest_frame_store,
                         SnapshotEnabled snapshot_enabled,
                         ILogger& logger,
                         std::string socket_path = KVM_VIN_SNAPSHOT_SOCK_PATH,
                         std::size_t max_connections = DEFAULT_MAX_CONNECTIONS);
    ~SnapshotSocketServer();

    SnapshotSocketServer(const SnapshotSocketServer&) = delete;
    SnapshotSocketServer& operator=(const SnapshotSocketServer&) = delete;

    bool start();
    void stop();

    [[nodiscard]] bool is_running() const;
    [[nodiscard]] const std::string& socket_path() const
    { return _socket_path; }

  private:
    void accept_loop(int listen_fd);
    void worker_loop();
    void handle_client(int client_fd);
    void handle_capture_fresh(int client_fd, CaptureRequest request);
    void handle_get_latest(int client_fd,
                           std::string previous_session,
                           std::uint64_t after_sequence,
                           int wait_ms);
    bool stopping() const;

    CaptureCoordinator& _capture_coordinator;
    OCRLatestFrameStore* _ocr_latest_frame_store {nullptr};
    SnapshotEnabled _snapshot_enabled;
    ILogger& _logger;
    std::string _socket_path;
    const std::size_t _max_connections;
    const std::size_t _max_latest_waiters;
    std::atomic<std::size_t> _active_latest_waiters {0};

    mutable std::mutex _lifecycle_mutex;
    mutable std::mutex _mutex;
    std::condition_variable _client_cv;
    std::deque<int> _pending_clients;
    std::unordered_set<int> _active_clients;
    std::vector<std::thread> _workers;
    std::thread _accept_thread;
    int _listen_fd {-1};
    bool _running {false};
    bool _stop_requested {false};
};
