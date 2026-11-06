#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "ui/background_assets.hpp"
#include "ui/page_manager_actions.hpp"
#include "ui/region_page_policy.hpp"
#include "ui/root_pager.hpp"
#include "ui/settings_service_runtime_state.hpp"
#include "ui/settings_wifi_tile_presentation.hpp"
#include "ui/tailscale_gate_view.hpp"
#include "ui/wifi_loading_mode.hpp"
#include "ui/wifi_page_action_policy.hpp"
#include "ui/wifi_ssid_font.hpp"
#include "ui/wifi_ssid_page.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

namespace ui
{

struct AppEntry;
struct SettingsRoute;
class MainDashboardView;
class PageRegistry;
class SettingsDashboardView;
class SettingsNavigator;
class SettingsToggleMediator;
class WifiFlowCoordinator;

struct MoreSysinfoSnapshot {
	std::string device_key;
	std::string hostname;
	std::string device_number;
	std::string image_version;
	std::string edid_name;
	std::string mac_address;
	std::string mdns_name;
	std::string wifi_signal;
	std::string uptime_text;
	std::string local_time_text;
	std::string memory_text;
	std::string storage_text;
	std::string temperature_text;
};

class PageManager {
    public:
	enum class MainStatusIcon : uint8_t {
		WIFI = 0,
		VIDEO = 1,
		KEYBOARD_MOUSE = 2
	};

	enum class PageTransitionMode : uint8_t {
		Slide,
		Flip
	};

	using SettingsServiceRuntimeState = ui::SettingsServiceRuntimeState;

	explicit PageManager(BackgroundAssets background_assets = {},
			     PageTransitionMode page_transition_mode =
				     PageTransitionMode::Slide,
			     bool wifi_config_mode_keyboard = true,
			     uint32_t wifi_boot_reconnect_timeout_ms = 15000,
			     uint32_t settings_state_boot_timeout_ms = 8000,
			     bool experimental_settings_subpages = true,
			     bool enable_settings_optimistic_actions = true,
			     const lv_font_t *wifi_ssid_font = nullptr,
			     uint32_t wifi_ssid_image_cache_kb = 0,
			     bool enable_ssid_scroll = false,
			     SsidFontRenderMode ssid_font_render_mode =
				     SsidFontRenderMode::CompiledFallback,
			     WifiLoadingMode wifi_loading_mode =
				     WifiLoadingMode::Progress);
	~PageManager();

	PageManager(const PageManager &) = delete;
	PageManager &operator=(const PageManager &) = delete;

	void init(lv_obj_t *screen);
	void bind_actions(PageManagerActions actions);

	void show_main();
	void show_settings();

	bool is_main_visible() const;
	bool is_settings_visible() const;
	bool is_initialized() const
	{
		return _initialized;
	}

	void show_settings_home();
	void show_wifi();
	void show_panel();
	void show_servo();
	void show_about();
	void show_more();
	void show_apps();
	void show_tailscale();
	void show_network();
	void show_region();

	void set_apps(std::vector<ui::AppEntry> apps);
	void set_apps_tile_visible(bool visible);
	void set_tailscale_tile_active(bool active);
	void set_tailscale_snapshot(const TailscaleSnapshotView &snapshot);
	void set_tailscale_code(const std::string &code, uint64_t seq);
	void set_tailscale_action_result(TailscaleActionKind kind, bool ok,
					 const std::string &message, uint64_t seq);
	uint64_t tailscale_snapshot_seq() const noexcept;
	void set_network_snapshot(const TailscaleSnapshotView &snapshot);
	void set_network_code(const std::string &code, uint64_t seq);
	void show_wifi_config();
	void set_wifi_scan_loading(std::string_view message = "Scanning...");
	void set_wifi_scan_results(const std::vector<WifiScanResult> &results);
	void set_wifi_scan_empty(std::string_view message = "No WiFi found");
	void set_wifi_scan_error(std::string_view message);
	void set_wifi_connect_loading(std::string_view message = "");
	void set_wifi_connect_success(std::string_view message = "Connected");
	void
	set_wifi_connect_failure(std::string_view message = "Connect failed");
	void set_wifi_ap_qr_payload(const std::string &payload);
	void set_wifi_qr_title(std::string_view title);
	void set_wifi_qr_payload(const std::string &payload);
	void set_wifi_ap_stop_completed_generation(uint64_t generation);
	void set_wifi_runtime_state(WifiPageVisualState state,
					    std::string_view status_text, bool busy,
					    std::string_view wifi_ssid,
					    std::string_view wifi_ip,
					    bool wifi_saved_network_config,
					    bool show_qr,
				    const std::string &wifi_qr_payload,
				    const std::string &wifi_web_url);
	void accept_prepared_ssid_raster(
		const SsidPreparedRasterResult &result);
	SettingsWifiTilePresentation
	settings_wifi_tile_presentation() const noexcept;
	void set_panel_brightness_percent(int percent);
	void set_panel_sleep_enabled(bool enabled);
	void clear_about_factory_reset_pending();
	void set_region_option(RegionOption region);
	void clear_region_switch_pending();
	void set_about_versions(const std::string &app, const std::string &fw,
				const std::string &hw, const std::string &ui);

	void set_more_sysinfo(const MoreSysinfoSnapshot &snapshot);

	void set_main_mouse_active(bool active);

	void set_main_status_icon_active(MainStatusIcon icon, bool active);

	void set_main_resolution_text(const char *text);
	void set_main_fps_text(const char *text);
	void set_main_fps_unit_text(const char *text);
	void set_main_keyboard_special_text(const char *text);
	void set_main_keyboard_normal_text(const char *text);
	void set_main_ip_empty_placeholder(std::string_view text);
	void set_main_ip_candidates(const std::vector<std::string> &candidates);
	void replace_background_assets(BackgroundAssets background_assets);

	void set_runtime_wifi_active(bool active);
	void set_runtime_wifi_boot_reconnect_pending(bool pending);
	void set_runtime_hdmi_input_active(bool active);
	void set_runtime_hid_active(bool active);
	void set_runtime_ncm_active(bool active);
	void set_runtime_ssh_active(bool active);
	void set_auto_rotation_manual_mode(bool manual_mode);
	void set_runtime_auto_rotation_active(bool active);
	void set_runtime_panel_active(bool active);
	void apply_settings_service_runtime_state(
		const SettingsServiceRuntimeState &state);

    private:
	PageManagerActions _actions;
	bool _initialized = false;
	bool _is_tearing_down = false;
	BackgroundAssets _background_assets;
	std::unique_ptr<RootPager> _root_pager;
	std::unique_ptr<MainDashboardView> _main_dashboard_view;
	std::unique_ptr<PageRegistry> _page_registry;
	std::unique_ptr<SettingsNavigator> _settings_navigator;
	uint32_t _settings_state_boot_timeout_ms = 8000;
	std::unique_ptr<SettingsDashboardView> _settings_dashboard_view;
	std::unique_ptr<WifiFlowCoordinator> _wifi_flow_coordinator;
	std::unique_ptr<SettingsToggleMediator> _settings_toggle_mediator;
	bool _tailscale_snapshot_valid = false;
	bool _tailscale_snapshot_dirty = false;
	uint64_t _tailscale_dirty_action_seq = 0;
	TailscaleSnapshotView _tailscale_snapshot;
	uint64_t _tailscale_snapshot_seq = 0;
	std::string _tailscale_code;
	uint64_t _tailscale_code_seq = 0;
	bool _network_snapshot_valid = false;
	TailscaleSnapshotView _network_snapshot;
	uint64_t _network_snapshot_seq = 0;
	std::string _network_code;
	uint64_t _network_code_seq = 0;
	void create_main_page();
	void create_settings_pages();
	void apply_direct_page_actions();
	void hide_all_settings_pages();
	lv_obj_t *root_content_page(RootPager::ContentPage page) const;
	static RootPager::TransitionMode
	root_transition_mode(PageTransitionMode mode) noexcept;
	static RootPager::StatusIcon
	root_status_icon(MainStatusIcon icon) noexcept;
	void show_settings_return_page();
	void show_settings_subpage(SettingsRoute route);
	void
	refresh_background_images(const BackgroundAssets &background_assets);
};

}
