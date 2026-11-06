#include "nanokvm_runtime_internal.hpp"

#include <algorithm>
#include <cctype>
#include <cmath>
#include <limits>
#include <memory>
#include <string>
#include <string_view>
#include <utility>

#include <curl/curl.h>
#include <spdlog/spdlog.h>

#include "lvgl_main/nanokvm_ip_selection.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"

namespace lvgl_main
{
namespace detail
{

namespace
{

using json = nlohmann::json;

constexpr std::chrono::seconds kHttpTimeout { 1 };

struct CurlEasyDeleter {
	void operator()(CURL *handle) const noexcept
	{
		if (handle != nullptr) {
			curl_easy_cleanup(handle);
		}
	}
};

struct CurlSlistDeleter {
	void operator()(curl_slist *headers) const noexcept
	{
		if (headers != nullptr) {
			curl_slist_free_all(headers);
		}
	}
};

using CurlEasyHandle = std::unique_ptr<CURL, CurlEasyDeleter>;
using CurlSlist = std::unique_ptr<curl_slist, CurlSlistDeleter>;

CURLcode libcurl_global_init_result()
{
	static const CURLcode result = curl_global_init(CURL_GLOBAL_DEFAULT);
	return result;
}

std::size_t append_response(char *contents, std::size_t size,
				std::size_t count, void *userdata) noexcept
{
	if (contents == nullptr || userdata == nullptr ||
	    (size != 0 && count > std::numeric_limits<std::size_t>::max() / size)) {
		return 0;
	}
	const auto bytes = size * count;
	if (bytes == 0) {
		return 0;
	}
	try {
		static_cast<std::string *>(userdata)->append(contents, bytes);
	} catch (...) {
		return 0;
	}
	return bytes;
}

}

std::string trim(std::string value)
{
	auto is_space = [](unsigned char ch) {
		return std::isspace(ch) != 0;
	};
	value.erase(value.begin(),
		    std::find_if(value.begin(), value.end(), [&](char ch) {
			    return !is_space(static_cast<unsigned char>(ch));
		    }));
	value.erase(
		std::find_if(value.rbegin(), value.rend(),
			     [&](char ch) {
				     return !is_space(
					     static_cast<unsigned char>(ch));
			     })
			.base(),
		value.end());
	return value;
}

bool starts_with(std::string_view text, std::string_view prefix)
{
	return starts_with_text(text, prefix);
}

bool contains_text(std::string_view text, std::string_view needle)
{
	return text.find(needle) != std::string_view::npos;
}

void redact_text(std::string &text, std::string_view secret)
{
	constexpr std::string_view kRedacted = "<redacted>";
	if (secret.empty()) {
		return;
	}

	std::size_t pos = 0;
	while ((pos = text.find(secret, pos)) != std::string::npos) {
		text.replace(pos, secret.size(), kRedacted.data(),
			     kRedacted.size());
		pos += kRedacted.size();
	}
}

std::string command_output_summary_for_log(std::string output,
					   std::string_view secret)
{
	output = trim(std::move(output));
	redact_text(output, secret);
	if (output.empty()) {
		return "<empty>";
	}
	std::replace(output.begin(), output.end(), '\n', '|');
	constexpr std::size_t kMaxSummaryLength = 160;
	if (output.size() > kMaxSummaryLength) {
		output.resize(kMaxSummaryLength);
		output += "...";
	}
	return output;
}

std::string percent_encode_form_value(std::string_view value)
{
	return percent_encode_wifi_form_value(value);
}

UrlParts parse_base_url(std::string_view url)
{
	UrlParts parts;
	std::string_view rest;
	if (starts_with(url, "https://")) {
		parts.scheme = "https";
		parts.port = 443;
		rest = url.substr(8);
	} else if (starts_with(url, "http://")) {
		parts.scheme = "http";
		parts.port = 80;
		rest = url.substr(7);
	} else {
		return parts;
	}

	const auto slash = rest.find('/');
	const auto host_port =
		slash == std::string_view::npos ? rest : rest.substr(0, slash);
	if (host_port.empty()) {
		return parts;
	}

	if (host_port.front() == '[') {
		const auto close = host_port.find(']');
		if (close == std::string_view::npos) {
			return parts;
		}
		parts.host = std::string(host_port.substr(1, close - 1));
		if (close + 1U < host_port.size()) {
			if (host_port[close + 1U] != ':') {
				return parts;
			}
			const auto port_text = host_port.substr(close + 2U);
			int port = 0;
			try {
				port = std::stoi(std::string(port_text));
			} catch (const std::exception &) {
				return parts;
			}
			if (port <= 0 || port > 65535) {
				return parts;
			}
			parts.port = port;
		}
	} else {
		const auto colon = host_port.rfind(':');
		if (colon == std::string_view::npos) {
			parts.host = std::string(host_port);
		} else {
			parts.host = std::string(host_port.substr(0, colon));
			int port = 0;
			try {
				port = std::stoi(std::string(
					host_port.substr(colon + 1U)));
			} catch (const std::exception &) {
				return parts;
			}
			if (port <= 0 || port > 65535) {
				return parts;
			}
			parts.port = port;
		}
	}

	parts.valid = !parts.host.empty();
	return parts;
}

std::optional<json> response_data(const json &root)
{
	if (root.is_object() && root.contains("code") &&
	    root["code"].is_number_integer() && root["code"].get<int>() != 0) {
		return std::nullopt;
	}

	if (root.is_object() && root.contains("data")) {
		return root["data"];
	}
	if (root.is_object() && root.contains("result")) {
		return root["result"];
	}
	return root;
}

std::optional<std::string> json_string(const json &value, std::string_view key)
{
	if (!value.is_object()) {
		return std::nullopt;
	}
	const auto it = value.find(std::string(key));
	if (it == value.end() || !it->is_string()) {
		return std::nullopt;
	}
	return it->get<std::string>();
}

std::optional<bool> json_bool(const json &value, std::string_view key)
{
	if (!value.is_object()) {
		return std::nullopt;
	}
	const auto it = value.find(std::string(key));
	if (it == value.end() || !it->is_boolean()) {
		return std::nullopt;
	}
	return it->get<bool>();
}

std::optional<std::string>
json_first_string(const json &value,
		       std::initializer_list<std::string_view> keys)
{
	for (std::string_view key : keys) {
		if (auto found = json_string(value, key);
		    found && !found->empty()) {
			return found;
		}
	}
	return std::nullopt;
}

std::optional<double> json_number(const json &value, std::string_view key)
{
	if (!value.is_object()) {
		return std::nullopt;
	}
	const auto it = value.find(std::string(key));
	if (it == value.end() || !it->is_number()) {
		return std::nullopt;
	}
	return it->get<double>();
}

bool optional_double_equal(std::optional<double> lhs,
			   std::optional<double> rhs) noexcept
{
	constexpr double kEpsilon = 0.0001;
	if (lhs.has_value() != rhs.has_value()) {
		return false;
	}
	if (!lhs.has_value()) {
		return true;
	}
	return std::fabs(*lhs - *rhs) <= kEpsilon;
}

bool display_metrics_changed(const RuntimeStatus &lhs,
			     const RuntimeStatus &rhs) noexcept
{
	return lhs.resolution != rhs.resolution ||
	       !optional_double_equal(lhs.fps, rhs.fps) ||
	       lhs.h264_bitrate_kbps != rhs.h264_bitrate_kbps;
}

bool systemd_status_changed(const RuntimeStatus &lhs,
			    const RuntimeStatus &rhs) noexcept
{
	return lhs.ssh_supported != rhs.ssh_supported ||
	       lhs.ssh_enabled != rhs.ssh_enabled;
}

bool domain_refresh_due(std::chrono::steady_clock::time_point last_refresh,
			std::chrono::steady_clock::time_point now,
			std::chrono::steady_clock::duration interval) noexcept
{
	return last_refresh == std::chrono::steady_clock::time_point {} ||
	       now - last_refresh >= interval;
}

void copy_systemd_status_fields(const RuntimeStatus &from, RuntimeStatus &to)
{
	to.ssh_supported = from.ssh_supported;
	to.ssh_enabled = from.ssh_enabled;
}

void copy_display_metrics_fields(const RuntimeStatus &from, RuntimeStatus &to)
{
	to.resolution = from.resolution;
	to.fps = from.fps;
	to.h264_bitrate_kbps = from.h264_bitrate_kbps;
}

bool local_service_status_changed(const RuntimeStatus &lhs,
				  const RuntimeStatus &rhs)
{
	return lhs.local_services_known != rhs.local_services_known ||
	       lhs.local_services_refresh_ok != rhs.local_services_refresh_ok ||
	       lhs.local_services_event_seq != rhs.local_services_event_seq ||
	       lhs.local_services_error != rhs.local_services_error ||
	       lhs.mcp_service_supported != rhs.mcp_service_supported ||
	       lhs.mcp_service_active != rhs.mcp_service_active ||
	       lhs.replay_service_supported != rhs.replay_service_supported ||
	       lhs.replay_service_active != rhs.replay_service_active;
}

void sync_runtime_flags(RuntimeStatus &status, bool video_enabled,
			bool hid_enabled, bool ncm_enabled, bool ncm_supported,
			bool auto_sleep_enabled, int backlight_percent,
			bool screen_sleeping)
{
	status.auto_sleep_enabled = auto_sleep_enabled;
	status.ncm_supported = ncm_supported;
	status.ncm_enabled = ncm_enabled;
	status.hdmi_input_supported = true;
	status.hdmi_input_enabled = video_enabled;
	status.video_enabled = video_enabled;
	status.hid_supported = true;
	status.hid_enabled = hid_enabled;
	if (!hid_enabled) {
		status.hid_mode.clear();
	}
	status.backlight_percent = backlight_percent;
	status.screen_on = !screen_sleeping;
}

void clear_display_metrics(RuntimeStatus &status)
{
	status.resolution.clear();
	status.fps.reset();
	status.h264_bitrate_kbps.reset();
}

std::optional<json> request_json_via_libcurl(
	const NanokvmRuntimeConfig &config, std::string_view base_url,
	std::string_view method, std::string_view path, std::string_view body,
	std::string_view content_type, bool disable_proxy)
{
	if (method != "GET" && method != "POST") {
		return std::nullopt;
	}
	const auto global_init_result = libcurl_global_init_result();
	if (global_init_result != CURLE_OK) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime libcurl initialization failed: {}",
			curl_easy_strerror(global_init_result));
		return std::nullopt;
	}

	const auto request_url = std::string(base_url) + std::string(path);

	CurlEasyHandle handle(curl_easy_init());
	if (!handle) {
		SPDLOG_DEBUG("[runtime] NanoKVM runtime HTTP {} {} handle setup failed",
			      method, path);
		return std::nullopt;
	}

	std::string response;
	response.reserve(256);
	CurlSlist headers;
	if (method == "POST") {
		const std::string header =
			"Content-Type: " + std::string(content_type);
		headers.reset(curl_slist_append(nullptr, header.c_str()));
		if (!headers) {
			SPDLOG_DEBUG(
				"[runtime] NanoKVM runtime HTTP {} {} header setup failed",
				method, path);
			return std::nullopt;
		}
	}

	const auto timeout_ms = static_cast<long>(
		std::chrono::duration_cast<std::chrono::milliseconds>(kHttpTimeout)
			.count());
	CURLcode option_result = CURLE_OK;
	auto set_option = [&](CURLoption option, auto value) {
		option_result = curl_easy_setopt(handle.get(), option, value);
		return option_result == CURLE_OK;
	};
	if (!set_option(CURLOPT_URL, request_url.c_str()) ||
	    !set_option(CURLOPT_WRITEFUNCTION, append_response) ||
	    !set_option(CURLOPT_WRITEDATA, &response) ||
	    !set_option(CURLOPT_CONNECTTIMEOUT_MS, timeout_ms) ||
	    !set_option(CURLOPT_TIMEOUT_MS, timeout_ms) ||
	    !set_option(CURLOPT_NOSIGNAL, 1L) ||
	    !set_option(CURLOPT_FOLLOWLOCATION, 0L) ||
	    !set_option(CURLOPT_FAILONERROR, 0L)) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} option setup failed: {}",
			method, path, curl_easy_strerror(option_result));
		return std::nullopt;
	}
	if (disable_proxy && !set_option(CURLOPT_NOPROXY, "*")) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} proxy setup failed: {}",
			method, path, curl_easy_strerror(option_result));
		return std::nullopt;
	}
	if (starts_with(base_url, "https://") &&
	    (!set_option(CURLOPT_SSL_VERIFYPEER,
			 config.server_verify_tls ? 1L : 0L) ||
	     !set_option(CURLOPT_SSL_VERIFYHOST,
			 config.server_verify_tls ? 2L : 0L))) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTPS {} {} TLS setup failed: {}",
			method, path, curl_easy_strerror(option_result));
		return std::nullopt;
	}
	if (method == "GET") {
		if (!set_option(CURLOPT_HTTPGET, 1L)) {
			SPDLOG_DEBUG(
				"[runtime] NanoKVM runtime HTTP {} {} method setup failed: {}",
				method, path, curl_easy_strerror(option_result));
			return std::nullopt;
		}
	} else if (!set_option(CURLOPT_POST, 1L) ||
		   !set_option(CURLOPT_POSTFIELDS, body.data()) ||
		   !set_option(CURLOPT_POSTFIELDSIZE_LARGE,
			       static_cast<curl_off_t>(body.size())) ||
		   !set_option(CURLOPT_HTTPHEADER, headers.get())) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} method setup failed: {}",
			method, path, curl_easy_strerror(option_result));
		return std::nullopt;
	}

	const auto perform_result = curl_easy_perform(handle.get());
	if (perform_result != CURLE_OK) {
		SPDLOG_DEBUG("[runtime] NanoKVM runtime HTTP {} {} failed: {}", method,
			      path, curl_easy_strerror(perform_result));
		return std::nullopt;
	}

	long status = 0;
	if (curl_easy_getinfo(handle.get(), CURLINFO_RESPONSE_CODE, &status) !=
	    CURLE_OK) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} status lookup failed", method,
			path);
		return std::nullopt;
	}
	if (status < 200 || status >= 300) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} returned status {}", method,
			path, status);
		return std::nullopt;
	}
	try {
		return json::parse(response);
	} catch (const std::exception &e) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime HTTP {} {} JSON parse failed: {}",
			method, path, e.what());
		return std::nullopt;
	}
}

std::optional<json> http_request_json(const NanokvmRuntimeConfig &config,
				      std::string_view method,
				      std::string_view path,
				      const httplib::Params &params)
{
	const auto url = parse_base_url(config.server_base_url);
	if (!url.valid) {
		spdlog::warn("NanoKVM runtime invalid server_base_url: {}",
			     config.server_base_url);
		return std::nullopt;
	}

	std::string body;
	std::string_view content_type;
	if (method == "POST") {
		for (const auto &[key, value] : params) {
			if (!body.empty()) {
				body += '&';
			}
			body += percent_encode_form_value(key);
			body += '=';
			body += percent_encode_form_value(value);
		}
		content_type = "application/x-www-form-urlencoded";
	}

	if (url.scheme == "https") {
		return request_json_via_libcurl(config, config.server_base_url, method,
						path, body, content_type, false);
	}
	const auto scheme_prefix_len = url.scheme.size() + 3;
	const auto rest =
		std::string_view(config.server_base_url).substr(scheme_prefix_len);
	const auto slash = rest.find('/');
	const auto origin = std::string(config.server_base_url).substr(
		0, scheme_prefix_len +
			   (slash == std::string_view::npos ? rest.size() : slash));
	return request_json_via_libcurl(config, origin, method, path, body,
					content_type, true);
}

std::optional<json> http_post_json(const NanokvmRuntimeConfig &config,
				   std::string_view path,
				   const nlohmann::json &body)
{
	const auto url = parse_base_url(config.server_base_url);
	if (!url.valid) {
		spdlog::warn("NanoKVM runtime invalid server_base_url: {}",
			     config.server_base_url);
		return std::nullopt;
	}

	const std::string payload = body.dump();
	if (url.scheme == "https") {
		return request_json_via_libcurl(config, config.server_base_url, "POST",
						path, payload, "application/json",
						false);
	}
	const auto scheme_prefix_len = url.scheme.size() + 3;
	const auto rest =
		std::string_view(config.server_base_url).substr(scheme_prefix_len);
	const auto slash = rest.find('/');
	const auto origin = std::string(config.server_base_url).substr(
		0, scheme_prefix_len +
			   (slash == std::string_view::npos ? rest.size() : slash));
	return request_json_via_libcurl(config, origin, "POST", path, payload,
					"application/json", true);
}

bool region_switch_response_ok(const nlohmann::json &response)
{
	if (!response.is_object()) {
		return false;
	}
	const auto code = response.find("code");
	return code != response.end() && code->is_number_integer() &&
	       code->get<int>() == 0;
}

}
}
