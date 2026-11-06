#pragma once

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <string_view>

#include "uds_http_client.hpp"

namespace lvgl_main
{

inline constexpr std::string_view kTailscaleSocketPath =
	"/run/kvm/nanokvm-state.sock";
inline constexpr std::string_view kTailscaleLoginCodePath =
	"/api/tailscale/login/code";
inline constexpr std::string_view kTailscaleLoginCodeDisablePath =
	"/api/tailscale/login/code?disable=true";
inline constexpr std::string_view kTailscaleLoginCodeStartPath =
	"/api/tailscale/login/code?start=true&consent=true";
inline constexpr std::string_view kTailscaleLoginCodeExitPath =
	"/api/tailscale/login/code?exit=true";
inline constexpr std::string_view kTailscaleLoginCodeHeartbeatPath =
	"/api/tailscale/login/code?heartbeat=true";

inline constexpr std::string_view kTailscaleOnlineCodePrefix = "%online%";
inline constexpr std::string_view kTailscaleOffCode = "%off%";

inline constexpr std::string_view kTailscaleActionFailMessage = "FAIL";
inline constexpr std::string_view kTailscaleActionTimeoutMessage =
	"TIMEOUT";

inline constexpr std::chrono::seconds kTailscaleInterval { 3 };
inline constexpr std::chrono::seconds kTailscaleBackoffMin { 1 };
inline constexpr std::chrono::seconds kTailscaleBackoffMax { 30 };

struct TailscalePeerInfo {
	std::string name;
	std::string ip;
};

struct TailscaleSession {
	bool online { false };
	std::string code;
	std::string peer_name;
	std::string peer_ip;
};

struct TailscaleSnapshot final {
	bool online { false };
	bool started { false };
	bool matched { false };
	std::string name;
	std::string ip;
};

enum class TailscaleStage : uint8_t {
	Unknown,
	NotInitialized,
	ClientInit,
	Ready,
};

enum class TailscaleAction : uint8_t {
	Start,
	Exit,
};

struct TailscaleTickPlan final {
	bool heartbeat { false };
	bool code { false };
};

struct TailscaleCodeLaneStep final {
	bool reopened { false };
	bool attempt { false };
};

struct TailscaleActionResult final {
	bool ok { false };
	std::string message;
};

constexpr TailscaleStage tailscale_stage(bool have_snapshot, bool started,
					bool matched) noexcept
{
	if (!have_snapshot) {
		return TailscaleStage::Unknown;
	}
	if (!started) {
		return TailscaleStage::NotInitialized;
	}
	return matched ? TailscaleStage::Ready : TailscaleStage::ClientInit;
}

constexpr bool tailscale_relinquish_on_leave(TailscaleStage stage) noexcept
{
	return stage == TailscaleStage::ClientInit;
}

constexpr TailscaleTickPlan tailscale_tick_plan(bool periodic_enabled,
						bool page_visible,
						TailscaleStage stage) noexcept
{
	if (!periodic_enabled) {
		return {};
	}
	return { true,
		 page_visible && stage == TailscaleStage::ClientInit };
}

constexpr TailscaleCodeLaneStep tailscale_code_lane_step(
	bool was_applicable, bool now_applicable, bool due) noexcept
{
	if (!now_applicable) {
		return {};
	}
	if (!was_applicable) {
		return { true, true };
	}
	return { false, due };
}

std::chrono::milliseconds
tailscale_interval_for_failures(uint8_t failure_count) noexcept;

std::chrono::steady_clock::time_point
tailscale_next_refresh_at(std::chrono::steady_clock::time_point now, bool ok,
			  uint8_t &failure_count) noexcept;

std::optional<std::string>
parse_tailscale_code_response(std::string_view body);

std::optional<TailscaleSnapshot>
parse_tailscale_snapshot(std::string_view body);

std::optional<TailscaleActionResult>
parse_tailscale_action_response(std::string_view body);

std::optional<TailscalePeerInfo>
parse_tailscale_online_code(std::string_view code);

bool is_tailscale_off_code(std::string_view code) noexcept;

bool is_tailscale_sas_code(std::string_view code) noexcept;

std::string sanitize_tailscale_display_code(std::string_view code);

std::optional<TailscaleSession>
tailscale_session_from_code(std::string_view code);

class TailscaleClient final {
    public:
	static constexpr std::chrono::seconds kDefaultActionTimeout { 15 };

	explicit TailscaleClient(
		std::string socket_path = std::string(kTailscaleSocketPath),
		std::chrono::milliseconds action_timeout =
			kDefaultActionTimeout);
	~TailscaleClient();

	TailscaleClient(const TailscaleClient &) = delete;
	TailscaleClient &operator=(const TailscaleClient &) = delete;

	std::optional<std::string> fetch_code();

	bool request_disable();

	std::optional<TailscaleSnapshot> fetch_snapshot();

	TailscaleActionResult request_action(TailscaleAction action);

	void stop();

	std::string_view last_error() const;

    private:
	static constexpr std::chrono::seconds kRequestTimeout { 1 };
	static constexpr std::size_t kMaxResponseBytes = 64U * 1024U;
	static constexpr std::chrono::milliseconds kTimeoutEpsilon { 10 };

	std::optional<detail::UdsHttpClient::Response>
	request(detail::UdsHttpClient &client, std::string_view path);

	std::chrono::milliseconds _action_timeout;
	std::unique_ptr<detail::UdsHttpClient> _client;
	std::unique_ptr<detail::UdsHttpClient> _action_client;
};

}
