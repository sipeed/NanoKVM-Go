#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

namespace ui
{

enum class SettingsToggleId : uint8_t {
	MCP = 0,
	REPLAY = 1,
	NCM = 2,
	SSH = 3,
	AUTO_ROTATION = 4,
};

inline constexpr std::size_t kSettingsToggleCount = 5;

inline constexpr std::array<SettingsToggleId, kSettingsToggleCount>
	kAllSettingsToggleIds {
		SettingsToggleId::MCP,		 SettingsToggleId::REPLAY,
		SettingsToggleId::NCM,		 SettingsToggleId::SSH,
		SettingsToggleId::AUTO_ROTATION,
	};

constexpr std::size_t to_settings_toggle_index(SettingsToggleId id) noexcept
{
	return static_cast<std::size_t>(id);
}

constexpr bool is_settings_service_toggle(SettingsToggleId id) noexcept
{
	return id == SettingsToggleId::MCP || id == SettingsToggleId::REPLAY;
}

} // namespace ui
