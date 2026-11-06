#pragma once

#include <cstdint>
#include <memory>
#include <vector>

#include <lvgl.h>

#include "ui/apps_page.hpp"
#include "ui/settings_route.hpp"
#include "ui/wifi_loading_mode.hpp"
#include "ui/wifi_ssid_font.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

namespace ui
{

class AboutPage;
class MorePage;
class AppsPage;
class Page;
class PanelPage;
class SettingsStateTogglePage;
class TailscalePage;
class NetworkPage;
class RegionPage;
class WifiConfigPage;
class WifiConnectMethodPage;
class WifiConnectResultPage;
class WifiPage;
class WifiQrPage;
class WifiSsidPage;
#if !NANOKVM_GO_RELEASE_BUILD
class ServoPage;
#endif

class PageRegistry final {
    public:
	explicit PageRegistry(const lv_font_t *wifi_ssid_font = nullptr,
				      uint32_t wifi_ssid_image_cache_kb = 0,
				      bool enable_ssid_scroll = false,
				      SsidFontRenderMode ssid_font_render_mode =
					      SsidFontRenderMode::CompiledFallback,
				      WifiLoadingMode wifi_loading_mode =
					      WifiLoadingMode::Progress);
	~PageRegistry();

	PageRegistry(const PageRegistry &) = delete;
	PageRegistry &operator=(const PageRegistry &) = delete;

	void create(lv_obj_t *settings_host);
	void begin_shutdown();
	void shutdown();

	bool is_created() const noexcept;
	void set_apps(std::vector<AppEntry> apps);

	Page *page_for(SettingsRoute route) const noexcept;
	WifiPage *wifi_page() const noexcept;
	WifiConnectMethodPage *wifi_connect_method_page() const noexcept;
	WifiQrPage *wifi_qr_page() const noexcept;
	WifiSsidPage *wifi_ssid_page() const noexcept;
	WifiConfigPage *wifi_manual_ssid_page() const noexcept;
	WifiConfigPage *wifi_password_page() const noexcept;
	WifiConnectResultPage *wifi_connect_result_page() const noexcept;
	PanelPage *panel_page() const noexcept;
	AboutPage *about_page() const noexcept;
	MorePage *more_page() const noexcept;
	AppsPage *apps_page() const noexcept;
	TailscalePage *tailscale_page() const noexcept;
	NetworkPage *network_page() const noexcept;
	RegionPage *region_page() const noexcept;
	SettingsStateTogglePage *
	settings_toggle_page(SettingsToggleId id) const noexcept;
#if !NANOKVM_GO_RELEASE_BUILD
	ServoPage *servo_page() const noexcept;
#endif

    private:
	bool _is_shutting_down = false;
	const lv_font_t *_wifi_ssid_font { nullptr };
	uint32_t _wifi_ssid_image_cache_kb { 0 };
	bool _enable_ssid_scroll { false };
	SsidFontRenderMode _ssid_font_render_mode {
		SsidFontRenderMode::CompiledFallback
	};
	WifiLoadingMode _wifi_loading_mode { WifiLoadingMode::None };
	std::vector<AppEntry> _apps;
	std::unique_ptr<WifiPage> _wifi_page;
	std::unique_ptr<WifiConnectMethodPage> _wifi_connect_method_page;
	std::unique_ptr<WifiQrPage> _wifi_qr_page;
	std::unique_ptr<WifiSsidPage> _wifi_ssid_page;
	std::unique_ptr<WifiConfigPage> _wifi_manual_ssid_page;
	std::unique_ptr<WifiConfigPage> _wifi_password_page;
	std::unique_ptr<WifiConnectResultPage> _wifi_connect_result_page;
	std::unique_ptr<PanelPage> _panel_page;
	std::unique_ptr<AboutPage> _about_page;
	std::unique_ptr<MorePage> _more_page;
	std::unique_ptr<SettingsStateTogglePage> _settings_mcp_page;
	std::unique_ptr<SettingsStateTogglePage> _settings_replay_page;
	std::unique_ptr<SettingsStateTogglePage> _settings_ncm_page;
	std::unique_ptr<SettingsStateTogglePage> _settings_ssh_page;
	std::unique_ptr<SettingsStateTogglePage> _settings_auto_rotation_page;
#if !NANOKVM_GO_RELEASE_BUILD
	std::unique_ptr<ServoPage> _servo_page;
#endif
	std::unique_ptr<AppsPage> _apps_page;
	std::unique_ptr<TailscalePage> _tailscale_page;
	std::unique_ptr<NetworkPage> _network_page;
	std::unique_ptr<RegionPage> _region_page;

	void clear_callbacks();
};

} // namespace ui
