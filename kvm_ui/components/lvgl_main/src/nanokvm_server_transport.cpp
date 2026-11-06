#include "nanokvm_server_transport.hpp"

#include <algorithm>
#include <array>
#include <mutex>
#include <string>
#include <utility>

#include <sys/stat.h>
#include <sys/un.h>

#include <spdlog/spdlog.h>

#include "lvgl_main/app_config_defaults.hpp"
#include "uds_http_client.hpp"

namespace lvgl_main
{
namespace
{

using Json = nlohmann::json;
using Clock = std::chrono::steady_clock;

constexpr std::size_t kMaxResponseBytes = 64U * 1024U;
constexpr auto kRequestTimeout = std::chrono::seconds(1);
constexpr auto kMinimumProbeInterval = std::chrono::milliseconds(1);
constexpr auto kMaximumProbeInterval = std::chrono::seconds(60);

enum class HttpMethod : uint8_t {
	Get,
	Post,
};

struct SupportedRoute final {
	HttpMethod method;
	std::string_view path;
};

constexpr std::array<SupportedRoute, 6> kSupportedRoutes { {
	{ HttpMethod::Get, "/api/vm/hdmi/capture" },
	{ HttpMethod::Get, "/api/network/wifi" },
	{ HttpMethod::Get, "/api/hid/mode" },
	{ HttpMethod::Get, "/api/streamer/local" },
	{ HttpMethod::Get, "/api/vm/hardware" },
	{ HttpMethod::Post, "/api/application/region" },
} };

enum class AdoptionState : uint8_t {
	LegacyOnly,
	LegacyAllowed,
	UdsAdopted,
	Stopped,
};

enum class UdsRequestMode : uint8_t {
	Legacy,
	Probe,
	Adopted,
	Stopped,
};

struct LaneState final {
	std::unique_ptr<detail::UdsHttpClient> client;
	std::mutex request_mutex;
};

bool is_default_local_url(std::string_view value)
{
	return value == app_config_defaults::kNanokvmServerBaseUrl;
}

bool is_supported_route(HttpMethod method, std::string_view path)
{
	return std::find_if(kSupportedRoutes.begin(), kSupportedRoutes.end(),
			    [method, path](const SupportedRoute &route) {
				    return route.method == method &&
					   route.path == path;
			    }) != kSupportedRoutes.end();
}

std::string_view method_name(HttpMethod method)
{
	return method == HttpMethod::Post ? "POST" : "GET";
}

bool valid_socket_path(const std::string &path)
{
	if (path.empty() || path.find('\0') != std::string::npos) {
		return false;
	}
	return path.size() < sizeof(sockaddr_un::sun_path);
}

bool socket_path_exists(const std::string &path)
{
	struct stat info {};
	if (::lstat(path.c_str(), &info) != 0) {
		return false;
	}
	return S_ISSOCK(info.st_mode);
}

std::chrono::milliseconds normalize_probe_interval(
	std::chrono::milliseconds interval)
{
	return std::clamp(interval, kMinimumProbeInterval,
			  std::chrono::duration_cast<std::chrono::milliseconds>(
				  kMaximumProbeInterval));
}

}

struct NanoKvmServerTransport::Impl final {
	using Lane = NanoKvmServerTransportLane;

	explicit Impl(const NanokvmRuntimeConfig &config, LegacyGet legacy,
		     LegacyPost legacy_post, std::filesystem::path socket,
		     std::chrono::milliseconds interval)
		: eligible(is_default_local_url(config.server_base_url))
		, legacy_get(std::move(legacy))
		, legacy_post(std::move(legacy_post))
		, socket_path(std::move(socket))
		, probe_interval(normalize_probe_interval(interval))
		, next_probe_at(Clock::time_point::min())
	{
		if (!eligible) {
			adoption = AdoptionState::LegacyOnly;
			return;
		}

		const std::string path = socket_path.string();
		if (!valid_socket_path(path)) {
			spdlog::warn(
				"NanoKVM local transport socket path is invalid; using legacy transport");
			adoption = AdoptionState::LegacyAllowed;
			return;
		}

		try {
			status.client = std::make_unique<detail::UdsHttpClient>(
				path, kRequestTimeout, kRequestTimeout, kMaxResponseBytes);
			metrics.client = std::make_unique<detail::UdsHttpClient>(
				path, kRequestTimeout, kRequestTimeout, kMaxResponseBytes);
		} catch (const std::exception &error) {
			status.client.reset();
			metrics.client.reset();
			spdlog::warn(
				"NanoKVM local transport client initialization failed: {}",
				error.what());
		}
	}

	~Impl() = default;

	LaneState &lane_state(Lane lane)
	{
		return lane == Lane::Status ? status : metrics;
	}

	bool stopped() const
	{
		std::lock_guard lock(state_mutex);
		return adoption == AdoptionState::Stopped;
	}

	bool adopted() const
	{
		std::lock_guard lock(state_mutex);
		return adoption == AdoptionState::UdsAdopted;
	}

	UdsRequestMode select_uds_request()
	{
		std::lock_guard lock(state_mutex);
		if (adoption == AdoptionState::Stopped) {
			return UdsRequestMode::Stopped;
		}
		if (adoption == AdoptionState::UdsAdopted) {
			return UdsRequestMode::Adopted;
		}
		if (!eligible || !status.client || !metrics.client) {
			return UdsRequestMode::Legacy;
		}

		const auto now = Clock::now();
		if (now < next_probe_at) {
			return UdsRequestMode::Legacy;
		}
		next_probe_at = now + probe_interval;
		return UdsRequestMode::Probe;
	}

	void adopt_uds()
	{
		std::lock_guard lock(state_mutex);
		if (adoption != AdoptionState::Stopped) {
			adoption = AdoptionState::UdsAdopted;
		}
	}

	bool can_call_legacy() const
	{
		std::lock_guard lock(state_mutex);
		return adoption != AdoptionState::Stopped &&
		       adoption != AdoptionState::UdsAdopted;
	}

	std::optional<Json> request_uds(LaneState &lane, HttpMethod method,
					std::string_view path,
					const Json *body)
	{
		if (!lane.client || path.empty() || path.front() != '/' ||
		    (method == HttpMethod::Post && body == nullptr)) {
			return std::nullopt;
		}

		const std::string_view name = method_name(method);
		try {
			const auto result =
				method == HttpMethod::Post ?
					lane.client->post(path, body->dump()) :
					lane.client->get(path);
			if (!result) {
				SPDLOG_DEBUG(
					"[runtime] NanoKVM local UDS {} {} failed: {}",
					name, path, lane.client->last_error());
				return std::nullopt;
			}
			if (result->status < 200 || result->status >= 300) {
				SPDLOG_DEBUG(
					"[runtime] NanoKVM local UDS {} {} returned status {}",
					name, path, result->status);
				return std::nullopt;
			}
			if (result->body.size() > kMaxResponseBytes) {
				SPDLOG_DEBUG(
					"[runtime] NanoKVM local UDS {} {} response exceeds limit",
					name, path);
				return std::nullopt;
			}
			return Json::parse(result->body);
		} catch (const std::exception &error) {
			SPDLOG_DEBUG(
				"[runtime] NanoKVM local UDS {} {} failed: {}", name,
				path, error.what());
		} catch (...) {
			SPDLOG_DEBUG(
				"[runtime] NanoKVM local UDS {} {} failed with unknown error",
				name, path);
		}
		return std::nullopt;
	}

	std::optional<Json> request_legacy(HttpMethod method, std::string_view path,
					   const Json *body)
	{
		if (!can_call_legacy()) {
			return std::nullopt;
		}

		const std::string_view name = method_name(method);
		try {
			if (method == HttpMethod::Post) {
				return legacy_post ? legacy_post(path, *body) :
						     std::nullopt;
			}
			return legacy_get ? legacy_get(path) : std::nullopt;
		} catch (const std::exception &error) {
			SPDLOG_DEBUG("[runtime] NanoKVM legacy {} {} failed: {}",
				     name, path, error.what());
		} catch (...) {
			SPDLOG_DEBUG(
				"[runtime] NanoKVM legacy {} {} failed with unknown error",
				name, path);
		}
		return std::nullopt;
	}

	std::optional<Json> get(Lane lane, std::string_view path)
	{
		if (!is_supported_route(HttpMethod::Get, path)) {
			return std::nullopt;
		}

		LaneState &state = lane_state(lane);
		std::unique_lock request_lock(state.request_mutex);
		const UdsRequestMode mode = select_uds_request();
		if (mode == UdsRequestMode::Stopped) {
			return std::nullopt;
		}

		const bool try_uds =
			mode == UdsRequestMode::Adopted ||
			(mode == UdsRequestMode::Probe &&
			 socket_path_exists(socket_path.string()));

		if (try_uds) {
			const auto local =
				request_uds(state, HttpMethod::Get, path, nullptr);
			if (local.has_value()) {
				adopt_uds();
				if (stopped()) {
					return std::nullopt;
				}
				return local;
			}
			if (adopted() || stopped()) {
				return std::nullopt;
			}
		}

		return request_legacy(HttpMethod::Get, path, nullptr);
	}

	std::optional<Json> post(Lane lane, std::string_view path,
				 const Json &body)
	{
		if (!is_supported_route(HttpMethod::Post, path)) {
			return std::nullopt;
		}

		LaneState &state = lane_state(lane);
		std::unique_lock request_lock(state.request_mutex);
		const UdsRequestMode mode = select_uds_request();
		if (mode == UdsRequestMode::Stopped) {
			return std::nullopt;
		}

		const bool try_uds =
			mode == UdsRequestMode::Adopted ||
			(mode == UdsRequestMode::Probe &&
			 socket_path_exists(socket_path.string()));

		if (try_uds) {
			const auto local =
				request_uds(state, HttpMethod::Post, path, &body);
			if (local.has_value()) {
				adopt_uds();
				if (stopped()) {
					return std::nullopt;
				}
				return local;
			}
			return std::nullopt;
		}

		return request_legacy(HttpMethod::Post, path, &body);
	}

	void start_transport()
	{
		std::lock_guard stop_lock(stop_mutex);
		std::lock_guard lock(state_mutex);
		adoption = eligible ? AdoptionState::LegacyAllowed :
				      AdoptionState::LegacyOnly;
		next_probe_at = Clock::time_point::min();
	}

	void stop_transport()
	{
		std::lock_guard stop_lock(stop_mutex);
		{
			std::lock_guard lock(state_mutex);
			if (adoption == AdoptionState::Stopped) {
				return;
			}
			adoption = AdoptionState::Stopped;
		}

		if (status.client) {
			status.client->stop();
		}
		if (metrics.client) {
			metrics.client->stop();
		}
		std::lock_guard status_lock(status.request_mutex);
		std::lock_guard metrics_lock(metrics.request_mutex);
	}

	bool eligible { false };
	LegacyGet legacy_get;
	LegacyPost legacy_post;
	std::filesystem::path socket_path;
	std::chrono::milliseconds probe_interval;
	Clock::time_point next_probe_at;
	std::mutex stop_mutex;
	mutable std::mutex state_mutex;
	AdoptionState adoption { AdoptionState::LegacyAllowed };
	LaneState status;
	LaneState metrics;
};

NanoKvmServerTransport::NanoKvmServerTransport(
	const NanokvmRuntimeConfig &config, LegacyGet legacy_get,
	std::filesystem::path socket_path, std::chrono::milliseconds probe_interval,
	LegacyPost legacy_post)
	: _impl(std::make_unique<Impl>(config, std::move(legacy_get),
				       std::move(legacy_post),
				       std::move(socket_path), probe_interval))
{
}

NanoKvmServerTransport::~NanoKvmServerTransport()
{
	stop();
}

std::optional<nlohmann::json> NanoKvmServerTransport::get_json(
	NanoKvmServerTransportLane lane, std::string_view path)
{
	if (!_impl) {
		return std::nullopt;
	}
	return _impl->get(lane, path);
}

std::optional<nlohmann::json> NanoKvmServerTransport::post_json(
	NanoKvmServerTransportLane lane, std::string_view path,
	const nlohmann::json &body)
{
	if (!_impl) {
		return std::nullopt;
	}
	return _impl->post(lane, path, body);
}

void NanoKvmServerTransport::start()
{
	if (_impl) {
		_impl->start_transport();
	}
}

void NanoKvmServerTransport::stop()
{
	if (_impl) {
		_impl->stop_transport();
	}
}

bool NanoKvmServerTransport::is_uds_adopted() const
{
	return _impl != nullptr && _impl->adopted();
}

bool NanoKvmServerTransport::is_eligible() const
{
	return _impl != nullptr && _impl->eligible;
}

}
