#include "nanokvm_tailscale_client.hpp"

#include <algorithm>
#include <utility>

#include <nlohmann/json.hpp>

namespace lvgl_main
{

namespace
{

bool is_printable_ascii_field(std::string_view value)
{
	if (value.empty()) {
		return false;
	}
	for (const char c : value) {
		const auto byte = static_cast<unsigned char>(c);
		if (byte < 0x20U || byte > 0x7EU || c == '%') {
			return false;
		}
	}
	return true;
}

bool http_success(const detail::UdsHttpClient::Response &response)
{
	return response.status >= 200 && response.status < 300;
}

constexpr std::size_t kTailscaleActionMessageMax = 10;

std::string tailscale_action_message(std::string_view raw)
{
	std::string sanitized = sanitize_tailscale_display_code(raw);
	if (sanitized.size() > kTailscaleActionMessageMax) {
		sanitized.resize(kTailscaleActionMessageMax);
	}
	return sanitized;
}

}

std::chrono::milliseconds
tailscale_interval_for_failures(uint8_t failure_count) noexcept
{
	if (failure_count == 0) {
		return std::chrono::duration_cast<std::chrono::milliseconds>(
			kTailscaleInterval);
	}

	auto backoff = kTailscaleBackoffMin;
	for (uint8_t index = 1;
	     index < failure_count && backoff < kTailscaleBackoffMax; ++index) {
		backoff = std::min(kTailscaleBackoffMax, backoff * 2);
	}
	return std::chrono::duration_cast<std::chrono::milliseconds>(backoff);
}

std::chrono::steady_clock::time_point tailscale_next_refresh_at(
	std::chrono::steady_clock::time_point now, bool ok,
	uint8_t &failure_count) noexcept
{
	if (ok) {
		failure_count = 0;
		return now + kTailscaleInterval;
	}
	++failure_count;
	return now + tailscale_interval_for_failures(failure_count);
}

std::optional<std::string>
parse_tailscale_code_response(std::string_view body)
{
	try {
		const auto root = nlohmann::json::parse(body);
		const auto &code = root.at("code");
		if (!code.is_string()) {
			return std::nullopt;
		}
		return sanitize_tailscale_display_code(
			code.get_ref<const std::string &>());
	} catch (const std::exception &) {
		return std::nullopt;
	}
}

std::optional<TailscaleSnapshot>
parse_tailscale_snapshot(std::string_view body)
{
	nlohmann::json root;
	try {
		root = nlohmann::json::parse(body);
	} catch (const std::exception &) {
		return std::nullopt;
	}
	if (!root.is_object()) {
		return std::nullopt;
	}

	TailscaleSnapshot snapshot;
	const auto online = root.find("status");
	if (online != root.end() && online->is_string() &&
	    online->get_ref<const std::string &>() == "online") {
		snapshot.online = true;
	}
	const auto bool_field = [&root](const char *key) {
		const auto found = root.find(key);
		return found != root.end() && found->is_boolean() &&
		       found->get<bool>();
	};
	const auto string_field = [&root](const char *key) {
		const auto found = root.find(key);
		if (found == root.end() || !found->is_string()) {
			return std::string {};
		}
		return sanitize_tailscale_display_code(
			found->get_ref<const std::string &>());
	};
	snapshot.started = bool_field("start");
	snapshot.matched = bool_field("match");
	snapshot.name = string_field("name");
	snapshot.ip = string_field("ip");
	return snapshot;
}

std::optional<TailscaleActionResult>
parse_tailscale_action_response(std::string_view body)
{
	nlohmann::json root;
	try {
		root = nlohmann::json::parse(body);
	} catch (const std::exception &) {
		return std::nullopt;
	}
	if (!root.is_object()) {
		return std::nullopt;
	}
	const auto status = root.find("status");
	if (status == root.end() || !status->is_boolean()) {
		return std::nullopt;
	}

	TailscaleActionResult result;
	result.ok = status->get<bool>();
	if (result.ok) {
		return result;
	}
	const auto message = root.find("message");
	if (message != root.end() && message->is_string()) {
		result.message = tailscale_action_message(
			message->get_ref<const std::string &>());
	}
	return result;
}

std::optional<TailscalePeerInfo>
parse_tailscale_online_code(std::string_view code)
{
	if (!code.starts_with(kTailscaleOnlineCodePrefix)) {
		return std::nullopt;
	}
	std::string_view rest = code.substr(kTailscaleOnlineCodePrefix.size());
	if (rest.empty() || rest.back() != '%') {
		return std::nullopt;
	}
	rest.remove_suffix(1);

	const auto separator = rest.find('%');
	if (separator == std::string_view::npos) {
		return std::nullopt;
	}
	const std::string_view name = rest.substr(0, separator);
	const std::string_view ip = rest.substr(separator + 1);
	if (!is_printable_ascii_field(name) || !is_printable_ascii_field(ip)) {
		return std::nullopt;
	}
	return TailscalePeerInfo { .name = std::string(name),
				   .ip = std::string(ip) };
}

bool is_tailscale_off_code(std::string_view code) noexcept
{
	return code == kTailscaleOffCode;
}

bool is_tailscale_sas_code(std::string_view code) noexcept
{
	if (code.size() != 3) {
		return false;
	}
	for (const char c : code) {
		if (c < '0' || c > '9') {
			return false;
		}
	}
	return true;
}

std::string sanitize_tailscale_display_code(std::string_view code)
{
	std::string sanitized;
	sanitized.reserve(code.size());
	for (const char c : code) {
		const auto byte = static_cast<unsigned char>(c);
		if (byte >= 0x20U && byte <= 0x7EU) {
			sanitized.push_back(c);
		}
	}
	return sanitized;
}

std::optional<TailscaleSession>
tailscale_session_from_code(std::string_view code)
{
	if (code.starts_with(kTailscaleOnlineCodePrefix)) {
		const auto peer = parse_tailscale_online_code(code);
		if (!peer.has_value()) {
			return std::nullopt;
		}
		TailscaleSession session;
		session.online = true;
		session.peer_name = peer->name;
		session.peer_ip = peer->ip;
		return session;
	}

	TailscaleSession session;
	if (is_tailscale_off_code(code)) {
		return session;
	}
	session.code = std::string(code);
	return session;
}

TailscaleClient::TailscaleClient(std::string socket_path,
				 std::chrono::milliseconds action_timeout)
	: _action_timeout(action_timeout)
	, _client(std::make_unique<detail::UdsHttpClient>(
		  socket_path, kRequestTimeout, kRequestTimeout,
		  kMaxResponseBytes))
	, _action_client(std::make_unique<detail::UdsHttpClient>(
		  std::move(socket_path), action_timeout, action_timeout,
		  kMaxResponseBytes))
{
}

TailscaleClient::~TailscaleClient() = default;

std::optional<detail::UdsHttpClient::Response>
TailscaleClient::request(detail::UdsHttpClient &client, std::string_view path)
{
	return client.get(path);
}

std::optional<std::string> TailscaleClient::fetch_code()
{
	const auto response = request(*_client, kTailscaleLoginCodePath);
	if (!response.has_value() || !http_success(*response)) {
		return std::nullopt;
	}
	return parse_tailscale_code_response(response->body);
}

bool TailscaleClient::request_disable()
{

	const auto response =
		request(*_client, kTailscaleLoginCodeDisablePath);
	return response.has_value() && http_success(*response);
}

std::optional<TailscaleSnapshot> TailscaleClient::fetch_snapshot()
{
	const auto response =
		request(*_client, kTailscaleLoginCodeHeartbeatPath);
	if (!response.has_value() || !http_success(*response)) {
		return std::nullopt;
	}
	return parse_tailscale_snapshot(response->body);
}

TailscaleActionResult TailscaleClient::request_action(TailscaleAction action)
{
	if (!_action_client) {
		return { false, std::string(kTailscaleActionFailMessage) };
	}
	const std::string_view path =
		action == TailscaleAction::Start ? kTailscaleLoginCodeStartPath :
						       kTailscaleLoginCodeExitPath;
	const auto started_at = std::chrono::steady_clock::now();
	const auto response = _action_client->get(path);
	const auto elapsed = std::chrono::steady_clock::now() - started_at;
	if (!response.has_value()) {
		const bool timed_out =
			elapsed + kTimeoutEpsilon >= _action_timeout;
		return { false,
			 std::string(timed_out ? kTailscaleActionTimeoutMessage :
						  kTailscaleActionFailMessage) };
	}
	if (!http_success(*response)) {
		return { false, std::string(kTailscaleActionFailMessage) };
	}
	const auto parsed = parse_tailscale_action_response(response->body);
	if (!parsed.has_value()) {
		return { false, std::string(kTailscaleActionFailMessage) };
	}
	return *parsed;
}

void TailscaleClient::stop()
{
	if (_client) {
		_client->stop();
	}
	if (_action_client) {
		_action_client->stop();
	}
}

std::string_view TailscaleClient::last_error() const
{
	return _client ? _client->last_error() : std::string_view {};
}

}
