#include "ui/settings_tile_layout.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

namespace ui::settings_tile_layout
{

SettingsTileLayout build_settings_tile_layout(bool replay_visible)
{
	SettingsTileLayout layout;
	layout.main = { SettingsTile::WIFI,
			SettingsTile::MCP,
			replay_visible ? SettingsTile::REPLAY :
					 SettingsTile::SCREEN,
			SettingsTile::NCM,
			SettingsTile::SSH,
			SettingsTile::AUTO_ROTATION };

	if (replay_visible) {
		layout.second = {
			SettingsTile::SCREEN,
#if !NANOKVM_GO_RELEASE_BUILD
			SettingsTile::SERVO,
#endif
			SettingsTile::APPS,
			SettingsTile::TAILSCALE,
			SettingsTile::REGION,
			SettingsTile::NETWORK,
			SettingsTile::SECOND_ABOUT
		};
	} else {
		layout.second = {
#if !NANOKVM_GO_RELEASE_BUILD
			SettingsTile::SERVO,
#endif
			SettingsTile::APPS,
			SettingsTile::TAILSCALE,
			SettingsTile::REGION,
			SettingsTile::NETWORK,
			SettingsTile::SECOND_ABOUT
		};
	}

	return layout;
}

} // namespace ui::settings_tile_layout
