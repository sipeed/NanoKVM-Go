#pragma once

#include <optional>
#include <string>
#include <string_view>

namespace snapshot_http {

inline constexpr std::string_view PATH = "/stream.mjpg";
inline constexpr std::string_view METHOD = "GET";

inline constexpr std::string_view DEFAULT_BIND_HOST = "127.0.0.1";
inline constexpr int DEFAULT_PORT = 18080;

inline constexpr int DEFAULT_QUALITY = 85;
inline constexpr int MIN_QUALITY = 1;
inline constexpr int MAX_QUALITY = 99;

inline constexpr int DEFAULT_TIMEOUT_MS = 1000;
inline constexpr int MIN_TIMEOUT_MS = 1;
inline constexpr int MAX_TIMEOUT_MS = 30000;

inline constexpr std::string_view CONTENT_TYPE_HEADER = "Content-Type";
inline constexpr std::string_view CONTENT_LENGTH_HEADER = "Content-Length";
inline constexpr std::string_view CACHE_CONTROL_HEADER = "Cache-Control";
inline constexpr std::string_view CONTENT_TYPE_OPTIONS_HEADER =
    "X-Content-Type-Options";
inline constexpr std::string_view AUTHORIZATION_HEADER = "Authorization";

inline constexpr std::string_view JPEG_CONTENT_TYPE = "image/jpeg";
inline constexpr std::string_view MJPEG_BOUNDARY = "frame";
inline constexpr std::string_view MJPEG_CONTENT_TYPE =
    "multipart/x-mixed-replace; boundary=frame";
inline constexpr std::string_view JSON_CONTENT_TYPE = "application/json";
inline constexpr std::string_view CACHE_CONTROL_VALUE = "no-store";
inline constexpr std::string_view CONTENT_TYPE_OPTIONS_VALUE = "nosniff";
inline constexpr std::string_view BEARER_PREFIX = "Bearer ";

inline constexpr int HTTP_OK = 200;
inline constexpr int HTTP_BAD_REQUEST = 400;
inline constexpr int HTTP_UNAUTHORIZED = 401;
inline constexpr int HTTP_NOT_FOUND = 404;
inline constexpr int HTTP_METHOD_NOT_ALLOWED = 405;
inline constexpr int HTTP_INTERNAL_SERVER_ERROR = 500;
inline constexpr int HTTP_SERVICE_UNAVAILABLE = 503;
inline constexpr int HTTP_GATEWAY_TIMEOUT = 504;

struct Config {
    bool enabled {false};
    std::string bind_host {DEFAULT_BIND_HOST};
    int port {DEFAULT_PORT};
    std::string token_file;
};

enum class ConfigError {
    None,
    ConflictsWithDumpRaw,
    EmptyBindHost,
    InvalidPort,
};

struct ConfigValidation {
    ConfigError error {ConfigError::None};
    std::string_view message;

    [[nodiscard]] bool ok() const { return error == ConfigError::None; }
};

[[nodiscard]] ConfigValidation validate_config(const Config& config,
                                               bool dump_raw_mode);
[[nodiscard]] bool is_authorized_bearer(std::string_view authorization_header,
                                        std::string_view token);

struct RequestParams {
    int quality {DEFAULT_QUALITY};
    int timeout_ms {DEFAULT_TIMEOUT_MS};
};

enum class QueryError {
    None,
    InvalidQuality,
    QualityOutOfRange,
    InvalidTimeout,
    TimeoutOutOfRange,
};

struct QueryParseResult {
    RequestParams params;
    QueryError error {QueryError::None};
    int ret_code {0};
    std::string_view message {"ok"};

    [[nodiscard]] bool ok() const { return error == QueryError::None; }
};

[[nodiscard]] QueryParseResult parse_query(
    std::optional<std::string_view> quality,
    std::optional<std::string_view> timeout_ms);

struct CaptureResultMapping {
    int http_status {HTTP_INTERNAL_SERVER_ERROR};
    int ret_code {0};
    std::string_view message {"snapshot failed"};
};

[[nodiscard]] CaptureResultMapping map_capture_result(int ret_code);

}  // namespace snapshot_http
