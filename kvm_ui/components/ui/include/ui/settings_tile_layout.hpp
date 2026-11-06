#pragma once

#include <vector>

namespace ui::settings_tile_layout
{

enum class SettingsTile {
	WIFI,
	MCP,
	REPLAY,
	NCM,
	SSH,
	AUTO_ROTATION,
	SCREEN,
	SERVO,
	SECOND_ABOUT,
	APPS,
	TAILSCALE,
	REGION,
	NETWORK,
};

struct SettingsTileLayout {
	std::vector<SettingsTile> main;
	std::vector<SettingsTile> second;
};

SettingsTileLayout build_settings_tile_layout(bool replay_visible);

} // namespace ui::settings_tile_layout
