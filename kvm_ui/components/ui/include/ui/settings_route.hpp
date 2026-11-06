#pragma once

#include <cstdint>

#include "ui/settings_toggle_id.hpp"

namespace ui
{

enum class SettingsRouteKind : uint8_t {
	Wifi,
	WifiConnectMethod,
	WifiQr,
	WifiSsid,
	WifiManualSsid,
	WifiPassword,
	WifiConnectResult,
	Panel,
	Servo,
	About,
	More,
	Apps,
	Tailscale,
	Region,
	Network,
	SettingsToggle,
};

struct SettingsRoute {
	SettingsRouteKind kind { SettingsRouteKind::Wifi };
	SettingsToggleId toggle_id { SettingsToggleId::MCP };

	static constexpr SettingsRoute wifi() noexcept
	{
		return { SettingsRouteKind::Wifi };
	}
	static constexpr SettingsRoute wifi_connect_method() noexcept
	{
		return { SettingsRouteKind::WifiConnectMethod };
	}
	static constexpr SettingsRoute wifi_qr() noexcept
	{
		return { SettingsRouteKind::WifiQr };
	}
	static constexpr SettingsRoute wifi_ssid() noexcept
	{
		return { SettingsRouteKind::WifiSsid };
	}
	static constexpr SettingsRoute wifi_manual_ssid() noexcept
	{
		return { SettingsRouteKind::WifiManualSsid };
	}
	static constexpr SettingsRoute wifi_password() noexcept
	{
		return { SettingsRouteKind::WifiPassword };
	}
	static constexpr SettingsRoute wifi_connect_result() noexcept
	{
		return { SettingsRouteKind::WifiConnectResult };
	}
	static constexpr SettingsRoute panel() noexcept
	{
		return { SettingsRouteKind::Panel };
	}
	static constexpr SettingsRoute servo() noexcept
	{
		return { SettingsRouteKind::Servo };
	}
	static constexpr SettingsRoute about() noexcept
	{
		return { SettingsRouteKind::About };
	}
	static constexpr SettingsRoute more() noexcept
	{
		return { SettingsRouteKind::More };
	}
	static constexpr SettingsRoute apps() noexcept
	{
		return { SettingsRouteKind::Apps };
	}
	static constexpr SettingsRoute tailscale() noexcept
	{
		return { SettingsRouteKind::Tailscale };
	}
	static constexpr SettingsRoute network() noexcept
	{
		return { SettingsRouteKind::Network };
	}
	static constexpr SettingsRoute region() noexcept
	{
		return { SettingsRouteKind::Region };
	}
	static constexpr SettingsRoute
	settings_toggle(SettingsToggleId id) noexcept
	{
		return { SettingsRouteKind::SettingsToggle, id };
	}

	constexpr bool operator==(const SettingsRoute &) const = default;
};

constexpr bool is_wifi_settings_route(SettingsRoute route) noexcept
{
	switch (route.kind) {
	case SettingsRouteKind::Wifi:
	case SettingsRouteKind::WifiConnectMethod:
	case SettingsRouteKind::WifiQr:
	case SettingsRouteKind::WifiSsid:
	case SettingsRouteKind::WifiManualSsid:
	case SettingsRouteKind::WifiPassword:
	case SettingsRouteKind::WifiConnectResult:
		return true;
	case SettingsRouteKind::Panel:
	case SettingsRouteKind::Servo:
	case SettingsRouteKind::About:
	case SettingsRouteKind::More:
	case SettingsRouteKind::Apps:
	case SettingsRouteKind::Tailscale:
	case SettingsRouteKind::Region:
	case SettingsRouteKind::Network:
	case SettingsRouteKind::SettingsToggle:
		return false;
	}
	return false;
}

constexpr bool is_wifi_config_flow_route(SettingsRoute route) noexcept
{
	switch (route.kind) {
	case SettingsRouteKind::WifiConnectMethod:
	case SettingsRouteKind::WifiQr:
	case SettingsRouteKind::WifiSsid:
	case SettingsRouteKind::WifiManualSsid:
	case SettingsRouteKind::WifiPassword:
	case SettingsRouteKind::WifiConnectResult:
		return true;
	case SettingsRouteKind::Wifi:
	case SettingsRouteKind::Panel:
	case SettingsRouteKind::Servo:
	case SettingsRouteKind::About:
	case SettingsRouteKind::More:
	case SettingsRouteKind::Apps:
	case SettingsRouteKind::Tailscale:
	case SettingsRouteKind::Region:
	case SettingsRouteKind::Network:
	case SettingsRouteKind::SettingsToggle:
		return false;
	}
	return false;
}

} // namespace ui
