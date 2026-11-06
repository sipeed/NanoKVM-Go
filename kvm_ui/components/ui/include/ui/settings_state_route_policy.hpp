#pragma once

#include <cstdint>

#include "ui/settings_toggle_id.hpp"

namespace ui::settings_state_route_policy
{

enum class SettingsStateHomeAction : uint8_t {
	DirectToggle,
	OpenSubpage,
	None,
};

enum class SettingsStateControlSurface : uint8_t {
	HomeTile,
	ToggleSubpage,
	None,
};

constexpr bool has_experimental_toggle_subpage(SettingsToggleId button)
{
	switch (button) {
	case SettingsToggleId::MCP:
	case SettingsToggleId::REPLAY:
	case SettingsToggleId::NCM:
	case SettingsToggleId::SSH:
	case SettingsToggleId::AUTO_ROTATION:
		return true;
	}
	return false;
}

constexpr SettingsStateHomeAction home_action(SettingsToggleId button,
					      bool experimental_subpages)
{
	if (!has_experimental_toggle_subpage(button)) {
		return SettingsStateHomeAction::None;
	}
	return experimental_subpages ? SettingsStateHomeAction::OpenSubpage :
				       SettingsStateHomeAction::DirectToggle;
}

constexpr SettingsStateControlSurface
control_surface(SettingsToggleId button, bool experimental_subpages)
{
	if (!has_experimental_toggle_subpage(button)) {
		return SettingsStateControlSurface::None;
	}
	return experimental_subpages ?
		       SettingsStateControlSurface::ToggleSubpage :
		       SettingsStateControlSurface::HomeTile;
}

} // namespace ui::settings_state_route_policy
