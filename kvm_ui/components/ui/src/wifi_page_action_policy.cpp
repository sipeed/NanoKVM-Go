#include "ui/wifi_page_action_policy.hpp"

namespace ui
{

WifiPrimaryActionPolicy
resolve_wifi_primary_action_policy(WifiPageVisualState state,
				   bool keyboard_config_mode,
				   bool interaction_blocked)
{
	if (interaction_blocked || state == WifiPageVisualState::Connecting) {
		return { WifiPrimaryAction::Blocked, "...", false };
	}

	switch (state) {
	case WifiPageVisualState::Off:
	case WifiPageVisualState::Error:
		return { WifiPrimaryAction::RequestEnable, "OFF", false };
	case WifiPageVisualState::Connected:
		return { WifiPrimaryAction::RequestDisable, "ON", true };
	case WifiPageVisualState::ApConnect:
	case WifiPageVisualState::WebConfigure:
		if (keyboard_config_mode) {
			return { WifiPrimaryAction::OpenSetup, "OFF", false };
		}
		return { WifiPrimaryAction::RequestDisable, "ON", false };
	case WifiPageVisualState::Connecting:
		break;
	}

	return { WifiPrimaryAction::Blocked, "...", false };
}

} // namespace ui
