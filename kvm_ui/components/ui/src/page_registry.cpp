#include "ui/page_registry.hpp"

#include <utility>

#include "ui/about_page.hpp"
#include "ui/more_page.hpp"
#include "ui/apps_page.hpp"
#include "ui/network_page.hpp"
#include "ui/page.hpp"
#include "ui/panel_page.hpp"
#include "ui/region_page.hpp"
#include "ui/settings_state_toggle_page.hpp"
#include "ui/tailscale_page.hpp"
#include "ui/wifi_connect_result_page.hpp"
#include "ui/wifi_connect_method_page.hpp"
#include "ui/wifi_config_page.hpp"
#include "ui/wifi_page.hpp"
#include "ui/wifi_qr_page.hpp"
#include "ui/wifi_ssid_page.hpp"
#if !NANOKVM_GO_RELEASE_BUILD
#include "ui/servo_page.hpp"
#endif

extern "C" {
LV_IMAGE_DECLARE(settings_agent_btn);
LV_IMAGE_DECLARE(settings_replay);
LV_IMAGE_DECLARE(settings_eth);
LV_IMAGE_DECLARE(settings_ssh);
LV_IMAGE_DECLARE(settings_auto_180);
}

namespace ui
{

namespace
{

void create_hidden_page(Page &page, lv_obj_t *settings_host)
{
	page.create(settings_host);
	if (lv_obj_t *container = page.get_container(); container != nullptr) {
		lv_obj_set_pos(container, 0, 0);
		lv_obj_add_flag(container, LV_OBJ_FLAG_HIDDEN);
	}
}

template <typename PageT> void destroy_page(std::unique_ptr<PageT> &page)
{
	if (page != nullptr) {
		page->destroy();
		page.reset();
	}
}

}

PageRegistry::PageRegistry(const lv_font_t *wifi_ssid_font,
				   uint32_t wifi_ssid_image_cache_kb,
				   bool enable_ssid_scroll,
				   SsidFontRenderMode ssid_font_render_mode,
				   WifiLoadingMode wifi_loading_mode)
	: _wifi_ssid_font(wifi_ssid_font)
	, _wifi_ssid_image_cache_kb(wifi_ssid_image_cache_kb)
	, _enable_ssid_scroll(enable_ssid_scroll)
	, _ssid_font_render_mode(ssid_font_render_mode)
	, _wifi_loading_mode(wifi_loading_mode)
{
}

PageRegistry::~PageRegistry()
{
	shutdown();
}

void PageRegistry::create(lv_obj_t *settings_host)
{
	if (_is_shutting_down || settings_host == nullptr ||
	    _wifi_page != nullptr) {
		return;
	}

	_wifi_page = std::make_unique<WifiPage>(_wifi_ssid_font,
						   _enable_ssid_scroll,
						   _wifi_ssid_image_cache_kb,
						   _ssid_font_render_mode);
	create_hidden_page(*_wifi_page, settings_host);

	_wifi_connect_method_page = std::make_unique<WifiConnectMethodPage>();
	create_hidden_page(*_wifi_connect_method_page, settings_host);

	_wifi_qr_page = std::make_unique<WifiQrPage>();
	create_hidden_page(*_wifi_qr_page, settings_host);

	_wifi_ssid_page = std::make_unique<WifiSsidPage>(
		_wifi_ssid_font, _wifi_ssid_image_cache_kb, _enable_ssid_scroll,
		_ssid_font_render_mode, _wifi_loading_mode);
	create_hidden_page(*_wifi_ssid_page, settings_host);

	_wifi_manual_ssid_page = std::make_unique<WifiConfigPage>();
	create_hidden_page(*_wifi_manual_ssid_page, settings_host);

	_wifi_password_page = std::make_unique<WifiConfigPage>();
	create_hidden_page(*_wifi_password_page, settings_host);

	_wifi_connect_result_page = std::make_unique<WifiConnectResultPage>();
	create_hidden_page(*_wifi_connect_result_page, settings_host);

	_panel_page = std::make_unique<PanelPage>();
	create_hidden_page(*_panel_page, settings_host);

#if !NANOKVM_GO_RELEASE_BUILD
	_servo_page = std::make_unique<ServoPage>();
	create_hidden_page(*_servo_page, settings_host);
#endif

	_about_page = std::make_unique<AboutPage>();
	create_hidden_page(*_about_page, settings_host);

	_more_page = std::make_unique<MorePage>();
	create_hidden_page(*_more_page, settings_host);

	_settings_mcp_page = std::make_unique<SettingsStateTogglePage>(
		"MCP", 0xFB2C36, &settings_agent_btn);
	create_hidden_page(*_settings_mcp_page, settings_host);

	_settings_replay_page = std::make_unique<SettingsStateTogglePage>(
		"Replay", 0x9810FA, &settings_replay);
	create_hidden_page(*_settings_replay_page, settings_host);

	_settings_ncm_page = std::make_unique<SettingsStateTogglePage>(
		"NCM", 0x00C950, &settings_eth);
	create_hidden_page(*_settings_ncm_page, settings_host);

	_settings_ssh_page = std::make_unique<SettingsStateTogglePage>(
		"SSH", 0x14B8A6, &settings_ssh);
	create_hidden_page(*_settings_ssh_page, settings_host);

	_settings_auto_rotation_page =
		std::make_unique<SettingsStateTogglePage>("Rotation", 0xFF6900,
							  &settings_auto_180);
	create_hidden_page(*_settings_auto_rotation_page, settings_host);

	_apps_page = std::make_unique<AppsPage>();
	_apps_page->set_entries(_apps);
	create_hidden_page(*_apps_page, settings_host);

	_tailscale_page = std::make_unique<TailscalePage>();
	create_hidden_page(*_tailscale_page, settings_host);

	_region_page = std::make_unique<RegionPage>();
	create_hidden_page(*_region_page, settings_host);

	_network_page = std::make_unique<NetworkPage>();
	create_hidden_page(*_network_page, settings_host);
}

void PageRegistry::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	clear_callbacks();
}

void PageRegistry::shutdown()
{
	begin_shutdown();

	destroy_page(_region_page);
	destroy_page(_tailscale_page);
	destroy_page(_network_page);
	destroy_page(_apps_page);
	destroy_page(_settings_auto_rotation_page);
	destroy_page(_settings_ssh_page);
	destroy_page(_settings_ncm_page);
	destroy_page(_settings_replay_page);
	destroy_page(_settings_mcp_page);
	destroy_page(_about_page);
	destroy_page(_more_page);
	destroy_page(_panel_page);
#if !NANOKVM_GO_RELEASE_BUILD
	destroy_page(_servo_page);
#endif
	destroy_page(_wifi_page);
	destroy_page(_wifi_connect_method_page);
	destroy_page(_wifi_qr_page);
	destroy_page(_wifi_ssid_page);
	destroy_page(_wifi_manual_ssid_page);
	destroy_page(_wifi_password_page);
	destroy_page(_wifi_connect_result_page);
}

bool PageRegistry::is_created() const noexcept
{
	return _wifi_page != nullptr;
}

void PageRegistry::set_apps(std::vector<AppEntry> apps)
{
	if (_is_shutting_down || _apps == apps) {
		return;
	}

	_apps = std::move(apps);
	if (_apps_page != nullptr) {
		_apps_page->set_entries(_apps);
	}
}

Page *PageRegistry::page_for(SettingsRoute route) const noexcept
{
	switch (route.kind) {
	case SettingsRouteKind::Wifi:
		return wifi_page();
	case SettingsRouteKind::WifiConnectMethod:
		return wifi_connect_method_page();
	case SettingsRouteKind::WifiQr:
		return wifi_qr_page();
	case SettingsRouteKind::WifiSsid:
		return wifi_ssid_page();
	case SettingsRouteKind::WifiManualSsid:
		return wifi_manual_ssid_page();
	case SettingsRouteKind::WifiPassword:
		return wifi_password_page();
	case SettingsRouteKind::WifiConnectResult:
		return wifi_connect_result_page();
	case SettingsRouteKind::Panel:
		return panel_page();
	case SettingsRouteKind::Servo:
#if !NANOKVM_GO_RELEASE_BUILD
		return servo_page();
#else
		return nullptr;
#endif
	case SettingsRouteKind::About:
		return about_page();
	case SettingsRouteKind::More:
		return more_page();
	case SettingsRouteKind::Apps:
		return apps_page();
	case SettingsRouteKind::Tailscale:
		return tailscale_page();
	case SettingsRouteKind::Network:
		return network_page();
	case SettingsRouteKind::Region:
		return region_page();
	case SettingsRouteKind::SettingsToggle:
		return settings_toggle_page(route.toggle_id);
	}
	return nullptr;
}

WifiPage *PageRegistry::wifi_page() const noexcept
{
	return _wifi_page.get();
}

WifiConnectMethodPage *PageRegistry::wifi_connect_method_page() const noexcept
{
	return _wifi_connect_method_page.get();
}

WifiQrPage *PageRegistry::wifi_qr_page() const noexcept
{
	return _wifi_qr_page.get();
}

WifiSsidPage *PageRegistry::wifi_ssid_page() const noexcept
{
	return _wifi_ssid_page.get();
}

WifiConfigPage *PageRegistry::wifi_manual_ssid_page() const noexcept
{
	return _wifi_manual_ssid_page.get();
}

WifiConfigPage *PageRegistry::wifi_password_page() const noexcept
{
	return _wifi_password_page.get();
}

WifiConnectResultPage *PageRegistry::wifi_connect_result_page() const noexcept
{
	return _wifi_connect_result_page.get();
}

PanelPage *PageRegistry::panel_page() const noexcept
{
	return _panel_page.get();
}

AboutPage *PageRegistry::about_page() const noexcept
{
	return _about_page.get();
}

MorePage *PageRegistry::more_page() const noexcept
{
	return _more_page.get();
}

AppsPage *PageRegistry::apps_page() const noexcept
{
	return _apps_page.get();
}

TailscalePage *PageRegistry::tailscale_page() const noexcept
{
	return _tailscale_page.get();
}

NetworkPage *PageRegistry::network_page() const noexcept
{
	return _network_page.get();
}

RegionPage *PageRegistry::region_page() const noexcept
{
	return _region_page.get();
}

SettingsStateTogglePage *
PageRegistry::settings_toggle_page(SettingsToggleId id) const noexcept
{
	switch (id) {
	case SettingsToggleId::MCP:
		return _settings_mcp_page.get();
	case SettingsToggleId::REPLAY:
		return _settings_replay_page.get();
	case SettingsToggleId::NCM:
		return _settings_ncm_page.get();
	case SettingsToggleId::SSH:
		return _settings_ssh_page.get();
	case SettingsToggleId::AUTO_ROTATION:
		return _settings_auto_rotation_page.get();
	}
	return nullptr;
}

#if !NANOKVM_GO_RELEASE_BUILD
ServoPage *PageRegistry::servo_page() const noexcept
{
	return _servo_page.get();
}
#endif

void PageRegistry::clear_callbacks()
{
	if (_wifi_page != nullptr) {
		_wifi_page->on_back_click = nullptr;
		_wifi_page->on_wifi_enabled_change = nullptr;
		_wifi_page->on_wifi_config_click = nullptr;
		_wifi_page->on_wifi_page_exit = nullptr;
	}
	if (_wifi_connect_method_page != nullptr) {
		_wifi_connect_method_page->on_back_click = nullptr;
		_wifi_connect_method_page->on_qr_click = nullptr;
		_wifi_connect_method_page->on_key_click = nullptr;
	}
	if (_wifi_qr_page != nullptr) {
		_wifi_qr_page->on_close_ready = nullptr;
		_wifi_qr_page->on_qr_start_request = nullptr;
		_wifi_qr_page->on_qr_stop_request = nullptr;
	}
	if (_wifi_ssid_page != nullptr) {
		_wifi_ssid_page->on_back_click = nullptr;
		_wifi_ssid_page->on_ssid_selected = nullptr;
		_wifi_ssid_page->on_manual_ssid_click = nullptr;
	}
	if (_wifi_manual_ssid_page != nullptr) {
		_wifi_manual_ssid_page->on_back_click = nullptr;
		_wifi_manual_ssid_page->on_ok_click = nullptr;
	}
	if (_wifi_password_page != nullptr) {
		_wifi_password_page->on_back_click = nullptr;
		_wifi_password_page->on_ok_click = nullptr;
	}
	if (_wifi_connect_result_page != nullptr) {
		_wifi_connect_result_page->on_back_click = nullptr;
		_wifi_connect_result_page->on_success_return_request = nullptr;
	}
	if (_panel_page != nullptr) {
		_panel_page->on_back_click = nullptr;
		_panel_page->on_brightness_change = nullptr;
		_panel_page->on_sleep_enabled_change = nullptr;
	}
#if !NANOKVM_GO_RELEASE_BUILD
	if (_servo_page != nullptr) {
		_servo_page->on_back_click = nullptr;
	}
#endif
	if (_about_page != nullptr) {
		_about_page->on_back_click = nullptr;
		_about_page->on_reset_click = nullptr;
		_about_page->on_app_ver_secret_click = nullptr;
	}
	if (_more_page != nullptr) {
		_more_page->on_back_click = nullptr;
	}
	if (_settings_mcp_page != nullptr) {
		_settings_mcp_page->on_back_click = nullptr;
		_settings_mcp_page->on_enabled_change = nullptr;
	}
	if (_settings_replay_page != nullptr) {
		_settings_replay_page->on_back_click = nullptr;
		_settings_replay_page->on_enabled_change = nullptr;
	}
	if (_settings_ncm_page != nullptr) {
		_settings_ncm_page->on_back_click = nullptr;
		_settings_ncm_page->on_enabled_change = nullptr;
	}
	if (_settings_ssh_page != nullptr) {
		_settings_ssh_page->on_back_click = nullptr;
		_settings_ssh_page->on_enabled_change = nullptr;
	}
	if (_settings_auto_rotation_page != nullptr) {
		_settings_auto_rotation_page->on_back_click = nullptr;
		_settings_auto_rotation_page->on_enabled_change = nullptr;
	}
	if (_apps_page != nullptr) {
		_apps_page->on_back_click = nullptr;
		_apps_page->on_app_launch = nullptr;
	}
	if (_tailscale_page != nullptr) {
		_tailscale_page->on_back_click = nullptr;
		_tailscale_page->on_session_visibility = nullptr;
		_tailscale_page->on_start_request = nullptr;
		_tailscale_page->on_exit_request = nullptr;
	}
	if (_network_page != nullptr) {
		_network_page->on_back_click = nullptr;
		_network_page->on_session_visibility = nullptr;
		_network_page->on_start_request = nullptr;
		_network_page->on_exit_request = nullptr;
	}
	if (_region_page != nullptr) {
		_region_page->on_back_click = nullptr;
		_region_page->on_switch_request = nullptr;
	}
}

}
