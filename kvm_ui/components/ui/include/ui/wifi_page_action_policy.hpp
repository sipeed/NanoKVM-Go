#pragma once

#include <cstdint>

namespace ui
{

enum class WifiPageVisualState : uint8_t {
	Off,
	Connected,
	Connecting,
	ApConnect,
	WebConfigure,
	Error,
};

enum class WifiPrimaryAction : uint8_t {
	RequestEnable,
	RequestDisable,
	OpenSetup,
	Blocked,
};

struct WifiPrimaryActionPolicy {
	WifiPrimaryAction action;
	const char *text;
	bool active;
};

WifiPrimaryActionPolicy resolve_wifi_primary_action_policy(
	WifiPageVisualState state, bool keyboard_config_mode,
	bool interaction_blocked);

} // namespace ui
