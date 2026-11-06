#pragma once

#include <cstdint>
#include <functional>
#include <memory>

#include <lvgl.h>

#include "ui/root_layout.hpp"
#include "ui/settings_toggle_id.hpp"
#include "ui/settings_wifi_tile_presentation.hpp"

namespace ui
{

class BackgroundImage;
class ControlState;
class SettingsPage;
struct SettingsToggleViewModel;

class SettingsDashboardView final {
    public:
	explicit SettingsDashboardView(bool experimental_settings_subpages,
				       uint32_t settings_state_boot_timeout_ms);
	~SettingsDashboardView();

	SettingsDashboardView(const SettingsDashboardView &) = delete;
	SettingsDashboardView &
	operator=(const SettingsDashboardView &) = delete;

	void create(lv_obj_t *scroll_parent, const BackgroundImage *background,
		    uint32_t wifi_pending_timeout_ms);
	void begin_shutdown();
	void shutdown();

	SettingsPage *settings_page() const noexcept;
	lv_obj_t *second_page_object() const noexcept;

	void replace_background(const BackgroundImage *background);
	void set_apps_tile_visible(bool visible);
	void set_tailscale_tile_active(bool active);
	void set_wifi_tile_active(bool active);
	void set_wifi_tile_pending(bool pending);
	SettingsWifiTilePresentation wifi_tile_presentation() const noexcept;
	void set_screen_tile_active(bool active);
	void set_toggle_availability(SettingsToggleId id,
				     const SettingsToggleViewModel &model);
	void render_toggle(SettingsToggleId id,
			   const SettingsToggleViewModel &model,
			   uint32_t pending_timeout_ms);

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
	const bool _experimental_settings_subpages;
	const uint32_t _settings_state_boot_timeout_ms;
	bool _is_shutting_down = false;
	bool _wifi_tile_active = true;
	bool _screen_tile_active = false;
	bool _tailscale_tile_active = false;
	std::unique_ptr<SettingsPage> _settings_page;
	lv_obj_t *_settings_second_page_obj = nullptr;
	lv_obj_t *_settings_second_background_obj = nullptr;

	void install_tile_callbacks();
	void clear_tile_callbacks();
	void apply_wifi_tile_state();
	void apply_screen_tile_state();
	void apply_tailscale_tile_state();
	ControlState *toggle_tile_control(SettingsToggleId id) const noexcept;
	static void apply_tile_state(ControlState *control, bool active);
};

}
