#include <charconv>
#include <cstddef>
#include <system_error>

#include "snapshot_http_contract.hpp"

namespace snapshot_http {

namespace {

constexpr int RET_INVALID_REQUEST = -4;
constexpr int RET_INVALID_PARAMS = -11;

bool parse_integer(std::string_view text, int& value)
{
    if (text.empty()) {
        return false;
    }

    const char* begin = text.data();
    const char* end = begin + text.size();
    const auto [next, error] = std::from_chars(begin, end, value);
    return error == std::errc {} && next == end;
}

QueryParseResult query_error(QueryError error,
                             int ret_code,
                             std::string_view message)
{
    QueryParseResult result;
    result.error = error;
    result.ret_code = ret_code;
    result.message = message;
    return result;
}

}  // namespace

ConfigValidation validate_config(const Config& config, bool dump_raw_mode)
{
    if (!config.enabled) {
        return {};
    }
    if (dump_raw_mode) {
        return {
            ConfigError::ConflictsWithDumpRaw,
            "--jpeg-http and --dump-raw cannot be used together",
        };
    }
    if (config.bind_host.empty()) {
        return {
            ConfigError::EmptyBindHost,
            "--jpeg-http-bind must not be empty",
        };
    }
    if (config.port <= 0 || config.port > 65535) {
        return {
            ConfigError::InvalidPort,
            "--jpeg-http-port must be in range 1..65535",
        };
    }
    return {};
}

bool is_authorized_bearer(std::string_view authorization_header,
                          std::string_view token)
{
    if (token.empty()) {
        return true;
    }
    if (authorization_header.size() != BEARER_PREFIX.size() + token.size()
        || authorization_header.substr(0, BEARER_PREFIX.size())
            != BEARER_PREFIX) {
        return false;
    }

    const std::string_view provided =
        authorization_header.substr(BEARER_PREFIX.size());
    unsigned char difference = 0;
    for (std::size_t i = 0; i < token.size(); ++i) {
        difference |= static_cast<unsigned char>(provided[i])
            ^ static_cast<unsigned char>(token[i]);
    }
    return difference == 0;
}

QueryParseResult parse_query(std::optional<std::string_view> quality,
                             std::optional<std::string_view> timeout_ms)
{
    QueryParseResult result;

    if (quality.has_value()) {
        if (!parse_integer(*quality, result.params.quality)) {
            return query_error(QueryError::InvalidQuality,
                               RET_INVALID_REQUEST,
                               "quality must be an integer");
        }
        if (result.params.quality < MIN_QUALITY
            || result.params.quality > MAX_QUALITY) {
            return query_error(QueryError::QualityOutOfRange,
                               RET_INVALID_PARAMS,
                               "quality must be in range 1..99");
        }
    }

    if (timeout_ms.has_value()) {
        if (!parse_integer(*timeout_ms, result.params.timeout_ms)) {
            return query_error(QueryError::InvalidTimeout,
                               RET_INVALID_REQUEST,
                               "timeout_ms must be an integer");
        }
        if (result.params.timeout_ms < MIN_TIMEOUT_MS
            || result.params.timeout_ms > MAX_TIMEOUT_MS) {
            return query_error(QueryError::TimeoutOutOfRange,
                               RET_INVALID_PARAMS,
                               "timeout_ms must be in range 1..30000");
        }
    }

    return result;
}

CaptureResultMapping map_capture_result(int ret_code)
{
    switch (ret_code) {
    case 0:
        return {HTTP_OK, ret_code, "ok"};
    case -11:
        return {HTTP_BAD_REQUEST, ret_code, "invalid params"};
    case -12:
        return {HTTP_BAD_REQUEST, ret_code, "ROI out of bounds"};
    case -13:
        return {HTTP_GATEWAY_TIMEOUT, ret_code, "snapshot timeout"};
    case -14:
        return {HTTP_INTERNAL_SERVER_ERROR, ret_code, "JPEG encode failed"};
    case -15:
        return {
            HTTP_INTERNAL_SERVER_ERROR,
            ret_code,
            "JPEG configuration failed",
        };
    case -5:
        return {HTTP_SERVICE_UNAVAILABLE, ret_code, "snapshot disabled"};
    case -10:
        return {HTTP_SERVICE_UNAVAILABLE, ret_code, "no desktop"};
    case -16:
        return {HTTP_SERVICE_UNAVAILABLE, ret_code, "snapshot busy"};
    case -17:
        return {HTTP_SERVICE_UNAVAILABLE, ret_code, "snapshot unavailable"};
    case -18:
        return {HTTP_SERVICE_UNAVAILABLE, ret_code, "snapshot cancelled"};
    case -19:
        return {
            HTTP_SERVICE_UNAVAILABLE,
            ret_code,
            "snapshot backend unhealthy",
        };
    default:
        return {HTTP_INTERNAL_SERVER_ERROR, ret_code, "snapshot failed"};
    }
}

}  // namespace snapshot_http
