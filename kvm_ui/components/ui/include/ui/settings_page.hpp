#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>

#include "ui/control_state.hpp"
#include "ui/page.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

namespace ui
{

class BackgroundImage;

class SettingsPage : public Page {
    public:
	SettingsPage();
	~SettingsPage() override;

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

	void create(lv_obj_t *parent) override;
	void destroy() override;
	void create_second_page_tiles(lv_obj_t *parent);
	void set_background_image(const BackgroundImage *image);
	void set_tile_visible(SettingsTile tile, bool visible);
	void set_tile_available(SettingsTile tile, bool available);
	void set_tile_pending_timeout(SettingsTile tile, uint32_t timeout_ms);
	bool tile_is_visible(SettingsTile tile) const;

	ControlState *get_wifi_btn() const;
	ControlState *get_mcp_btn() const;
	ControlState *get_replay_btn() const;
	ControlState *get_ncm_btn() const;
	ControlState *get_ssh_btn() const;
	ControlState *get_auto_rotation_btn() const;
	ControlState *get_screen_btn() const;
	ControlState *get_second_about_btn() const;
	ControlState *get_tailscale_btn() const;
	ControlState *get_network_btn() const;
	ControlState *get_region_btn() const;

	std::function<void()> on_wifi_click;
	std::function<void()> on_mcp_click;
	std::function<void()> on_replay_click;
	std::function<void()> on_ncm_click;
	std::function<void()> on_ssh_click;
	std::function<void()> on_auto_rotation_click;
	std::function<void()> on_screen_click;
	std::function<void()> on_servo_click;
	std::function<void()> on_second_about_click;
	std::function<void()> on_apps_click;
	std::function<void()> on_tailscale_click;
	std::function<void()> on_network_click;
	std::function<void()> on_region_click;

    private:
	static constexpr uint16_t WIDTH = 270;
	static constexpr uint16_t HEIGHT = 170;
	static constexpr uint16_t CONTENT_X = 0;
	static constexpr uint16_t CONTENT_Y = 0;
	static constexpr uint16_t CONTENT_WIDTH = 270;
	static constexpr uint16_t CONTENT_HEIGHT = 170;
	static constexpr uint16_t CONTENT_RADIUS = 40;

	class TileButton;

	std::unique_ptr<TileButton> _wifi_btn;
	std::unique_ptr<TileButton> _mcp_btn;
	std::unique_ptr<TileButton> _replay_btn;
	std::unique_ptr<TileButton> _ncm_btn;
	std::unique_ptr<TileButton> _agent_btn;
	std::unique_ptr<TileButton> _ssh_btn;
	std::unique_ptr<TileButton> _auto_rotation_btn;
	std::unique_ptr<TileButton> _screen_btn;
	std::unique_ptr<TileButton> _servo_btn;
	std::unique_ptr<TileButton> _second_about_btn;
	std::unique_ptr<TileButton> _apps_btn;
	std::unique_ptr<TileButton> _tailscale_btn;
	std::unique_ptr<TileButton> _network_btn;
	std::unique_ptr<TileButton> _region_btn;
	const BackgroundImage *_background_image = nullptr;
	lv_obj_t *_background_obj = nullptr;
	lv_obj_t *_main_tile_parent = nullptr;
	lv_obj_t *_second_tile_parent = nullptr;
	bool _screen_tile_on_main = false;

	void create_tile_grid(lv_obj_t *content);
	TileButton *get_tile(SettingsTile tile) const;
	void update_screen_tile_placement();
	void relayout_all_tiles();
	void relayout_main_tiles();
	void relayout_second_page_tiles();
	void relayout_tiles(TileButton *const *tiles, std::size_t count);
};

} // namespace ui
