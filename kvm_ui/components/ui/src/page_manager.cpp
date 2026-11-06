#include <string>
#include <string_view>
#include <utility>

#include "ui/page_manager.hpp"

#include "ui/about_page.hpp"
#include "ui/more_page.hpp"
#include "ui/background_assets.hpp"
#include "ui/main_dashboard_view.hpp"
#include "ui/network_page.hpp"
#include "ui/page_registry.hpp"
#include "ui/panel_page.hpp"
#include "ui/region_page.hpp"
#include "ui/root_pager.hpp"
#include "ui/settings_dashboard_view.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/settings_page.hpp"
#include "ui/settings_route.hpp"
#include "ui/settings_toggle_mediator.hpp"
#include "ui/tailscale_page.hpp"
#include "ui/wifi_flow_coordinator.hpp"
#if !NANOKVM_GO_RELEASE_BUILD
#include "ui/servo_page.hpp"
#endif

namespace ui
{

namespace
{

constexpr uint32_t DEFAULT_SETTINGS_STATE_BOOT_TIMEOUT_MS = 8000;

}

PageManager::PageManager(BackgroundAssets background_assets,
			 PageTransitionMode page_transition_mode,
			 bool wifi_config_mode_keyboard,
			 uint32_t wifi_boot_reconnect_timeout_ms,
			 uint32_t settings_state_boot_timeout_ms,
			 bool experimental_settings_subpages,
			 bool enable_settings_optimistic_actions,
			 const lv_font_t *wifi_ssid_font,
			 uint32_t wifi_ssid_image_cache_kb,
			 bool enable_ssid_scroll,
			 SsidFontRenderMode ssid_font_render_mode,
			 WifiLoadingMode wifi_loading_mode)
	: _background_assets(std::move(background_assets))
	, _root_pager(std::make_unique<RootPager>(
		  root_transition_mode(page_transition_mode)))
	, _main_dashboard_view(std::make_unique<MainDashboardView>())
	, _page_registry(std::make_unique<PageRegistry>(
		  wifi_ssid_font, wifi_ssid_image_cache_kb, enable_ssid_scroll,
		  ssid_font_render_mode, wifi_loading_mode))
	, _settings_navigator(std::make_unique<SettingsNavigator>(
		  *_root_pager, *_page_registry))
	, _settings_state_boot_timeout_ms(
		  settings_state_boot_timeout_ms == 0 ?
			  DEFAULT_SETTINGS_STATE_BOOT_TIMEOUT_MS :
			  settings_state_boot_timeout_ms)
	, _settings_dashboard_view(std::make_unique<SettingsDashboardView>(
		  experimental_settings_subpages,
		  _settings_state_boot_timeout_ms))
	, _wifi_flow_coordinator(std::make_unique<WifiFlowCoordinator>(
		  *_page_registry, *_settings_navigator, _actions.wifi,
		  _actions.ssid_raster,
		  wifi_config_mode_keyboard, wifi_boot_reconnect_timeout_ms,
		  WifiFlowCoordinator::Callbacks {
			  .render_connectivity =
				  [this](WifiConnectivityViewModel connectivity) {
					  if (_settings_dashboard_view) {
						  _settings_dashboard_view
							  ->set_wifi_tile_pending(
								  connectivity
									  .boot_reconnect_pending);
						  _settings_dashboard_view
							  ->set_wifi_tile_active(
								  connectivity
									  .active);
					  }
					  set_main_status_icon_active(
						  MainStatusIcon::WIFI,
						  connectivity.active);
				  },
			  .on_boot_reconnect_timeout =
				  [this]() {
					  set_main_ip_empty_placeholder("--");
				  },
		  }))
	, _settings_toggle_mediator(std::make_unique<SettingsToggleMediator>(
		  *_settings_dashboard_view, *_page_registry,
		  *_settings_navigator, _actions.settings,
		  _settings_state_boot_timeout_ms, experimental_settings_subpages,
		  enable_settings_optimistic_actions))
{
	_settings_navigator->set_return_navigation_guard([this]() {
		if (_is_tearing_down || _settings_navigator == nullptr) {
			return false;
		}
		const auto route = _settings_navigator->top_route();
		if (!route.has_value()) {
			return true;
		}
		if (route->kind == SettingsRouteKind::About) {
			const AboutPage *about_page =
				_page_registry ? _page_registry->about_page() :
						nullptr;
			return about_page == nullptr ||
			       !about_page->is_factory_reset_pending();
		}
		if (route->kind == SettingsRouteKind::Region) {
			const RegionPage *region_page =
				_page_registry ? _page_registry->region_page() :
						 nullptr;
			return region_page == nullptr ||
			       !region_page->is_switch_pending();
		}
		return true;
	});
}

PageManager::~PageManager()
{
	_is_tearing_down = true;
	_actions = {};
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->begin_shutdown();
	}
	if (_root_pager) {
		_root_pager->begin_shutdown();
	}
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->begin_shutdown();
	}
	if (_main_dashboard_view) {
		_main_dashboard_view->begin_shutdown();
	}
	if (_settings_dashboard_view) {
		_settings_dashboard_view->begin_shutdown();
	}
	if (_page_registry) {
		_page_registry->begin_shutdown();
	}
	if (_settings_navigator) {
		_settings_navigator->shutdown();
	}
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->shutdown();
		_wifi_flow_coordinator.reset();
	}
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->shutdown();
		_settings_toggle_mediator.reset();
	}
	if (_settings_navigator) {
		_settings_navigator.reset();
	}
	if (_page_registry) {
		_page_registry->shutdown();
		_page_registry.reset();
	}

	if (_settings_dashboard_view) {
		_settings_dashboard_view->shutdown();
		_settings_dashboard_view.reset();
	}
	if (_main_dashboard_view) {
		_main_dashboard_view->shutdown();
		_main_dashboard_view.reset();
	}

	if (_root_pager) {
		_root_pager->shutdown();
		_root_pager.reset();
	}
	_initialized = false;
}

void PageManager::init(lv_obj_t *screen)
{
	if (_initialized || _is_tearing_down || _root_pager == nullptr ||
	    screen == nullptr) {
		return;
	}

	_root_pager->create(screen);
	if (!_root_pager->is_created()) {
		return;
	}
	create_main_page();
	create_settings_pages();
	_root_pager->set_content_page_provider(
		[this](RootPager::ContentPage page) {
			return root_content_page(page);
		});
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->start();
	}

	_root_pager->transition_to(RootPager::ContentPage::Main);
	_root_pager->refresh_navigation_chrome();

	_initialized = true;
}

void PageManager::bind_actions(PageManagerActions actions)
{
	if (_is_tearing_down) {
		return;
	}

	_actions = std::move(actions);
	apply_direct_page_actions();
}

void PageManager::apply_direct_page_actions()
{
	if (_is_tearing_down || _page_registry == nullptr) {
		return;
	}

	if (PanelPage *panel_page = _page_registry->panel_page();
	    panel_page != nullptr) {
		panel_page->on_brightness_change =
			_actions.panel.set_brightness_percent;
		panel_page->on_sleep_enabled_change =
			_actions.panel.set_sleep_enabled;
	}

	if (AboutPage *about_page = _page_registry->about_page();
	    about_page != nullptr) {
		about_page->on_reset_click =
			_actions.about.request_factory_reset;
	}
}

SettingsWifiTilePresentation
PageManager::settings_wifi_tile_presentation() const noexcept
{
	return _settings_dashboard_view ?
		       _settings_dashboard_view->wifi_tile_presentation() :
		       SettingsWifiTilePresentation {};
}

void PageManager::set_panel_brightness_percent(int percent)
{
	if (PanelPage *panel_page =
		    _page_registry ? _page_registry->panel_page() : nullptr;
	    panel_page != nullptr &&
	    panel_page->can_apply_runtime_brightness_percent(percent)) {
		panel_page->set_brightness_percent(percent);
	}
}

void PageManager::set_panel_sleep_enabled(bool enabled)
{
	if (PanelPage *panel_page =
		    _page_registry ? _page_registry->panel_page() : nullptr;
	    panel_page != nullptr) {
		panel_page->set_sleep_enabled(enabled);
	}
}

void PageManager::clear_about_factory_reset_pending()
{
	if (AboutPage *about_page =
		    _page_registry ? _page_registry->about_page() : nullptr;
	    about_page != nullptr) {
		about_page->set_factory_reset_pending(false);
	}
}

void PageManager::set_region_option(RegionOption region)
{
	if (RegionPage *region_page =
		    _page_registry ? _page_registry->region_page() : nullptr;
	    region_page != nullptr) {
		region_page->set_region(region);
	}
}

void PageManager::clear_region_switch_pending()
{
	if (RegionPage *region_page =
		    _page_registry ? _page_registry->region_page() : nullptr;
	    region_page != nullptr) {
		region_page->clear_switch_pending();
	}
}

void PageManager::set_about_versions(const std::string &app,
				     const std::string &fw,
				     const std::string &hw,
				     const std::string &ui)
{
	if (AboutPage *about_page =
		    _page_registry ? _page_registry->about_page() : nullptr;
	    about_page != nullptr) {
		about_page->set_versions(app, fw, hw);
	}
	if (MorePage *more_page =
		    _page_registry ? _page_registry->more_page() : nullptr;
	    more_page != nullptr) {
		more_page->set_versions(app, fw, hw);
		more_page->set_ui_version(ui);
	}
}

void PageManager::set_more_sysinfo(const MoreSysinfoSnapshot &snapshot)
{
	if (MorePage *more_page =
		    _page_registry ? _page_registry->more_page() : nullptr;
	    more_page != nullptr) {
		more_page->set_device_key(snapshot.device_key);
		more_page->set_hostname(snapshot.hostname);
		more_page->set_device_number(snapshot.device_number);
		more_page->set_image_version(snapshot.image_version);
		more_page->set_edid_name(snapshot.edid_name);
		more_page->set_mac_address(snapshot.mac_address);
		more_page->set_mdns_name(snapshot.mdns_name);
		more_page->set_wifi_signal(snapshot.wifi_signal);
		more_page->set_uptime(snapshot.uptime_text);
		more_page->set_local_time(snapshot.local_time_text);
		more_page->set_memory(snapshot.memory_text);
		more_page->set_storage(snapshot.storage_text);
		more_page->set_temperature(snapshot.temperature_text);
	}
}

RootPager::TransitionMode
PageManager::root_transition_mode(PageTransitionMode mode) noexcept
{
	switch (mode) {
	case PageTransitionMode::Slide:
		return RootPager::TransitionMode::Slide;
	case PageTransitionMode::Flip:
		return RootPager::TransitionMode::Flip;
	}
	return RootPager::TransitionMode::Slide;
}

RootPager::StatusIcon
PageManager::root_status_icon(MainStatusIcon icon) noexcept
{
	switch (icon) {
	case MainStatusIcon::WIFI:
		return RootPager::StatusIcon::Wifi;
	case MainStatusIcon::VIDEO:
		return RootPager::StatusIcon::Video;
	case MainStatusIcon::KEYBOARD_MOUSE:
		return RootPager::StatusIcon::KeyboardMouse;
	}
	return RootPager::StatusIcon::Wifi;
}

lv_obj_t *PageManager::root_content_page(RootPager::ContentPage page) const
{
	switch (page) {
	case RootPager::ContentPage::Main:
		return _main_dashboard_view == nullptr ?
			       nullptr :
			       _main_dashboard_view->content_object();
	case RootPager::ContentPage::SettingsHome:
		if (_settings_dashboard_view == nullptr) {
			return nullptr;
		}
		if (SettingsPage *settings_page =
			    _settings_dashboard_view->settings_page();
		    settings_page != nullptr) {
			return settings_page->get_container();
		}
		return nullptr;
	case RootPager::ContentPage::SettingsSecond:
		return _settings_dashboard_view == nullptr ?
			       nullptr :
			       _settings_dashboard_view->second_page_object();
	case RootPager::ContentPage::Count:
		return nullptr;
	}
	return nullptr;
}

void PageManager::create_main_page()
{
	if (_main_dashboard_view == nullptr || _root_pager == nullptr) {
		return;
	}

	_main_dashboard_view->create(_root_pager->content_parent(),
				     _background_assets.main.get());
}

void PageManager::create_settings_pages()
{
	if (_root_pager == nullptr) {
		return;
	}

	lv_obj_t *const scroll_container = _root_pager->content_parent();
	lv_obj_t *const settings_host = _root_pager->settings_host();
	if (scroll_container == nullptr || settings_host == nullptr) {
		return;
	}

	if (_settings_dashboard_view) {
		_settings_dashboard_view->create(
			scroll_container, _background_assets.settings.get(),
			_wifi_flow_coordinator ?
				_wifi_flow_coordinator
					->boot_reconnect_timeout_ms() :
				15000);
		_settings_dashboard_view->on_wifi_click = [this]() {
			show_wifi();
		};
		_settings_dashboard_view->on_screen_click = [this]() {
			show_panel();
		};
#if !NANOKVM_GO_RELEASE_BUILD
		_settings_dashboard_view->on_servo_click = [this]() {
			show_servo();
		};
#endif
		_settings_dashboard_view->on_second_about_click = [this]() {
			show_about();
		};
		_settings_dashboard_view->on_apps_click = [this]() {
			show_apps();
		};
		_settings_dashboard_view->on_tailscale_click = [this]() {
			show_tailscale();
		};
		_settings_dashboard_view->on_network_click = [this]() {
			show_network();
		};
		_settings_dashboard_view->on_region_click = [this]() {
			show_region();
		};
	}

	if (_page_registry == nullptr) {
		return;
	}
	_page_registry->create(settings_host);
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->bind_pages();
	}

	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->bind_pages();
	}

	if (PanelPage *panel_page = _page_registry->panel_page();
	    panel_page != nullptr) {
		panel_page->on_back_click = [this]() {
			show_settings_return_page();
		};
	}

#if !NANOKVM_GO_RELEASE_BUILD
	if (ServoPage *servo_page = _page_registry->servo_page();
	    servo_page != nullptr) {
		servo_page->on_back_click = [this]() {
			show_settings_return_page();
		};
	}
#endif

	if (AboutPage *about_page = _page_registry->about_page();
	    about_page != nullptr) {
		about_page->on_back_click = [this]() {
			show_settings_return_page();
		};
		about_page->on_app_ver_secret_click = [this]() {
			show_more();
		};
	}

	if (MorePage *more_page = _page_registry->more_page();
	    more_page != nullptr) {
		more_page->on_back_click = [this]() {
			const auto visibility =
				_actions.more.page_visibility_changed;
			if (visibility) {
				visibility(false);
			}
			if (_settings_navigator != nullptr) {
				(void)_settings_navigator->pop();
			}
		};
	}
	apply_direct_page_actions();

	if (AppsPage *apps_page = _page_registry->apps_page();
	    apps_page != nullptr) {
		apps_page->on_back_click = [this]() {
			show_settings_return_page();
		};
		apps_page->on_app_launch = [this](const std::string &path) {
			const auto callback = _actions.apps.launch;
			if (callback) {
				callback(path);
			}
		};
	}

	if (TailscalePage *tailscale_page = _page_registry->tailscale_page();
	    tailscale_page != nullptr) {
		tailscale_page->on_back_click = [this]() {
			show_settings_return_page();
		};
		tailscale_page->on_session_visibility = [this](bool visible) {
			const auto visibility =
				_actions.tailscale.page_visibility_changed;
			if (visibility) {
				visibility(visible);
			}
		};
		tailscale_page->on_start_request = [this]() -> bool {
			const auto callback = _actions.tailscale.start;
			return callback && callback();
		};
		tailscale_page->on_exit_request = [this]() -> bool {
			const auto callback = _actions.tailscale.exit;
			return callback && callback();
		};
	}

	if (NetworkPage *network_page = _page_registry->network_page();
	    network_page != nullptr) {
		network_page->on_back_click = [this]() {
			show_settings_return_page();
		};
		network_page->on_session_visibility = [this](bool visible) {
			const auto visibility =
				_actions.tailscale.page_visibility_changed;
			if (visibility) {
				visibility(visible);
			}
		};
		network_page->on_start_request = [this]() -> bool {
			const auto callback = _actions.tailscale.start;
			return callback && callback();
		};
		network_page->on_exit_request = [this]() -> bool {
			const auto callback = _actions.tailscale.exit;
			return callback && callback();
		};
	}

	if (RegionPage *region_page = _page_registry->region_page();
	    region_page != nullptr) {
		region_page->on_back_click = [this]() {
			show_settings_return_page();
		};
		region_page->on_switch_request = [this](RegionOption target) {
			const auto callback = _actions.region.request_switch;
			return callback && callback(target);
		};
	}

	hide_all_settings_pages();
}

void PageManager::set_apps(std::vector<ui::AppEntry> apps)
{
	if (_page_registry) {
		_page_registry->set_apps(std::move(apps));
	}
}

void PageManager::show_apps()
{
	show_settings_subpage(SettingsRoute::apps());
}

void PageManager::set_apps_tile_visible(bool visible)
{
	if (_settings_dashboard_view) {
		_settings_dashboard_view->set_apps_tile_visible(visible);
	}
}

void PageManager::show_tailscale()
{
	show_settings_subpage(SettingsRoute::tailscale());
	if (_page_registry == nullptr) {
		return;
	}
	TailscalePage *tailscale_page = _page_registry->tailscale_page();
	if (tailscale_page == nullptr) {
		return;
	}
	if (_tailscale_snapshot_valid) {
		tailscale_page->apply_snapshot(_tailscale_snapshot,
					       _tailscale_snapshot_seq);
	}
	tailscale_page->apply_code(_tailscale_code, _tailscale_code_seq);
}

void PageManager::show_network()
{
	show_settings_subpage(SettingsRoute::network());
	if (_page_registry == nullptr) {
		return;
	}
	NetworkPage *network_page = _page_registry->network_page();
	if (network_page == nullptr) {
		return;
	}
	if (_network_snapshot_valid) {
		network_page->apply_snapshot(_network_snapshot,
					     _network_snapshot_seq);
	}
	network_page->apply_code(_network_code, _network_code_seq);
}

void PageManager::set_tailscale_tile_active(bool active)
{
	if (_settings_dashboard_view) {
		_settings_dashboard_view->set_tailscale_tile_active(active);
	}
}

void PageManager::set_tailscale_snapshot(
	const TailscaleSnapshotView &snapshot)
{
	const bool changed =
		_tailscale_snapshot_dirty || !_tailscale_snapshot_valid ||
		_tailscale_snapshot.has_snapshot != snapshot.has_snapshot ||
		_tailscale_snapshot.online != snapshot.online ||
		_tailscale_snapshot.started != snapshot.started ||
		_tailscale_snapshot.matched != snapshot.matched ||
		_tailscale_snapshot.name != snapshot.name ||
		_tailscale_snapshot.ip != snapshot.ip;

	if (changed) {
		_tailscale_snapshot = snapshot;
		_tailscale_snapshot_valid = true;
		_tailscale_snapshot_dirty = false;
		++_tailscale_snapshot_seq;
	}

	if (_page_registry == nullptr) {
		return;
	}
	if (TailscalePage *tailscale_page = _page_registry->tailscale_page();
	    tailscale_page != nullptr) {
		tailscale_page->apply_snapshot(_tailscale_snapshot,
					       _tailscale_snapshot_seq);
	}
}

void PageManager::set_tailscale_code(const std::string &code, uint64_t seq)
{
	_tailscale_code = code;
	_tailscale_code_seq = seq;
	if (_page_registry == nullptr) {
		return;
	}
	if (TailscalePage *tailscale_page = _page_registry->tailscale_page();
	    tailscale_page != nullptr) {
		tailscale_page->apply_code(code, seq);
	}
}

void PageManager::set_tailscale_action_result(TailscaleActionKind kind, bool ok,
					      const std::string &message,
					      uint64_t seq)
{
	if (ok && seq != _tailscale_dirty_action_seq) {
		_tailscale_dirty_action_seq = seq;
		_tailscale_snapshot_dirty = true;
	}
	if (_page_registry == nullptr) {
		return;
	}
	if (TailscalePage *tailscale_page = _page_registry->tailscale_page();
	    tailscale_page != nullptr) {
		tailscale_page->apply_action_result(kind, ok, message, seq);
	}
	if (NetworkPage *network_page = _page_registry->network_page();
	    network_page != nullptr) {
		network_page->apply_action_result(kind, ok, message, seq);
	}
}

uint64_t PageManager::tailscale_snapshot_seq() const noexcept
{
	return _tailscale_snapshot_seq;
}

void PageManager::set_network_snapshot(
	const TailscaleSnapshotView &snapshot)
{
	const bool changed =
		!_network_snapshot_valid ||
		_network_snapshot.has_snapshot != snapshot.has_snapshot ||
		_network_snapshot.online != snapshot.online ||
		_network_snapshot.started != snapshot.started ||
		_network_snapshot.matched != snapshot.matched ||
		_network_snapshot.name != snapshot.name ||
		_network_snapshot.ip != snapshot.ip;

	if (changed) {
		_network_snapshot = snapshot;
		_network_snapshot_valid = true;
		++_network_snapshot_seq;
	}

	if (_page_registry == nullptr) {
		return;
	}
	if (NetworkPage *network_page = _page_registry->network_page();
	    network_page != nullptr) {
		network_page->apply_snapshot(_network_snapshot,
					     _network_snapshot_seq);
	}
}

void PageManager::set_network_code(const std::string &code, uint64_t seq)
{
	_network_code = code;
	_network_code_seq = seq;
	if (_page_registry == nullptr) {
		return;
	}
	if (NetworkPage *network_page = _page_registry->network_page();
	    network_page != nullptr) {
		network_page->apply_code(code, seq);
	}
}

void PageManager::show_main()
{
	if (!_initialized || _is_tearing_down ||
	    _settings_navigator == nullptr || _root_pager == nullptr ||
	    _root_pager->is_flip_animating()) {
		return;
	}
	if (_settings_toggle_mediator) {
		(void)_settings_toggle_mediator->request_service_refresh();
	}
	(void)_settings_navigator->show_root_page(RootPager::ContentPage::Main);
}

void PageManager::show_settings()
{
	if (!_initialized || _is_tearing_down ||
	    _settings_navigator == nullptr || _root_pager == nullptr ||
	    _root_pager->is_flip_animating()) {
		return;
	}
	if (_settings_toggle_mediator) {
		(void)_settings_toggle_mediator->request_service_refresh();
	}
	(void)_settings_navigator->show_root_page(
		RootPager::ContentPage::SettingsHome);
}

bool PageManager::is_main_visible() const
{
	return _initialized && _root_pager != nullptr &&
	       _root_pager->is_main_visible();
}

bool PageManager::is_settings_visible() const
{
	return _initialized && _root_pager != nullptr &&
	       _root_pager->is_settings_visible();
}

void PageManager::hide_all_settings_pages()
{
	if (_settings_navigator != nullptr) {
		_settings_navigator->clear();
	}
}

void PageManager::show_settings_home()
{
	if (_is_tearing_down || _settings_navigator == nullptr ||
	    _root_pager == nullptr || _root_pager->settings_host() == nullptr ||
	    _root_pager->is_flip_animating()) {
		return;
	}

	if (_settings_toggle_mediator) {
		(void)_settings_toggle_mediator->request_service_refresh();
	}
	(void)_settings_navigator->show_root_page(
		RootPager::ContentPage::SettingsHome);
}

void PageManager::show_settings_return_page()
{
	if (_is_tearing_down || _settings_navigator == nullptr) {
		return;
	}

	(void)_settings_navigator->return_to_anchor();
}

void PageManager::show_wifi()
{
	if (!_is_tearing_down && _wifi_flow_coordinator) {
		_wifi_flow_coordinator->show_wifi();
	}
}

void PageManager::show_wifi_config()
{
	if (!_is_tearing_down && _wifi_flow_coordinator) {
		_wifi_flow_coordinator->show_wifi_config();
	}
}

void PageManager::set_wifi_scan_loading(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_scan_loading(message);
	}
}

void PageManager::set_wifi_scan_results(
	const std::vector<WifiScanResult> &results)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_scan_results(results);
	}
}

void PageManager::set_wifi_scan_empty(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_scan_empty(message);
	}
}

void PageManager::set_wifi_scan_error(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_scan_error(message);
	}
}

void PageManager::set_wifi_connect_loading(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_connect_loading(message);
	}
}

void PageManager::set_wifi_connect_success(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_connect_success(message);
	}
}

void PageManager::set_wifi_connect_failure(std::string_view message)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_connect_failure(message);
	}
}

void PageManager::set_wifi_ap_qr_payload(const std::string &payload)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_ap_qr_payload(payload);
	}
}

void PageManager::set_wifi_qr_payload(const std::string &payload)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_qr_payload(payload);
	}
}

void PageManager::set_wifi_qr_title(std::string_view title)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_qr_title(title);
	}
}

void PageManager::set_wifi_ap_stop_completed_generation(uint64_t generation)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_ap_stop_completed_generation(
			generation);
	}
}

void PageManager::set_wifi_runtime_state(WifiPageVisualState state,
						 std::string_view status_text,
						 bool busy, std::string_view wifi_ssid,
						 std::string_view wifi_ip,
						 bool wifi_saved_network_config, bool show_qr,
						 const std::string &wifi_qr_payload,
						 const std::string &wifi_web_url)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_runtime_wifi_state(
				state, status_text, busy, wifi_ssid, wifi_ip,
				wifi_saved_network_config, show_qr, wifi_qr_payload,
				wifi_web_url);
	}
}

void PageManager::accept_prepared_ssid_raster(
	const SsidPreparedRasterResult &result)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->accept_prepared_ssid_raster(result);
	}
}

void PageManager::show_panel()
{
	show_settings_subpage(SettingsRoute::panel());
}

void PageManager::show_servo()
{
#if !NANOKVM_GO_RELEASE_BUILD
	show_settings_subpage(SettingsRoute::servo());
#endif
}

void PageManager::show_about()
{
	show_settings_subpage(SettingsRoute::about());
}

void PageManager::show_region()
{
	show_settings_subpage(SettingsRoute::region());
}

void PageManager::show_more()
{
	if (_is_tearing_down || _settings_navigator == nullptr) {
		return;
	}
	(void)_settings_navigator->push(SettingsRoute::more());
	const auto visibility = _actions.more.page_visibility_changed;
	if (visibility) {
		visibility(true);
	}
}

void PageManager::show_settings_subpage(SettingsRoute route)
{
	if (_is_tearing_down || _settings_navigator == nullptr) {
		return;
	}
	(void)_settings_navigator->show(route);
}

void PageManager::set_main_mouse_active(bool active)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_mouse_active(active);
	}
}

void PageManager::set_main_status_icon_active(MainStatusIcon icon, bool active)
{
	if (_root_pager != nullptr) {
		_root_pager->set_status_icon_active(root_status_icon(icon),
						    active);
	}
}

void PageManager::set_main_resolution_text(const char *text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_resolution_text(text);
	}
	if (MorePage *more_page =
		    _page_registry ? _page_registry->more_page() : nullptr;
	    more_page != nullptr) {
		more_page->set_resolution(text != nullptr ? text : "");
	}
}

void PageManager::set_main_fps_text(const char *text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_fps_text(text);
	}
}

void PageManager::set_main_fps_unit_text(const char *text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_fps_unit_text(text);
	}
}

void PageManager::set_main_keyboard_special_text(const char *text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_keyboard_special_text(text);
	}
}

void PageManager::set_main_keyboard_normal_text(const char *text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_keyboard_normal_text(text);
	}
}

void PageManager::set_main_ip_empty_placeholder(std::string_view text)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_ip_empty_placeholder(text);
	}
	if (MorePage *more_page =
		    _page_registry ? _page_registry->more_page() : nullptr;
	    more_page != nullptr) {
		more_page->set_ip_candidates({});
	}
}

void PageManager::set_main_ip_candidates(
	const std::vector<std::string> &candidates)
{
	if (_main_dashboard_view) {
		_main_dashboard_view->set_ip_candidates(candidates);
	}
	if (MorePage *more_page =
		    _page_registry ? _page_registry->more_page() : nullptr;
	    more_page != nullptr) {
		more_page->set_ip_candidates(candidates);
	}
}

void PageManager::set_runtime_wifi_active(bool active)
{
	if (_wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_runtime_wifi_active(active);
	}
}

void PageManager::set_runtime_wifi_boot_reconnect_pending(bool pending)
{
	if (!_is_tearing_down && _wifi_flow_coordinator) {
		_wifi_flow_coordinator->set_boot_reconnect_pending(pending);
	}
}

void PageManager::replace_background_assets(BackgroundAssets background_assets)
{
	refresh_background_images(background_assets);
	_background_assets = std::move(background_assets);
}

void PageManager::set_runtime_hdmi_input_active(bool active)
{
	set_main_status_icon_active(MainStatusIcon::VIDEO, active);
}

void PageManager::set_runtime_hid_active(bool active)
{
	set_main_status_icon_active(MainStatusIcon::KEYBOARD_MOUSE, active);
}

void PageManager::set_runtime_ncm_active(bool active)
{
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->set_confirmed(SettingsToggleId::NCM,
							 active);
	}
}

void PageManager::set_runtime_ssh_active(bool active)
{
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->set_confirmed(SettingsToggleId::SSH,
							 active);
	}
}

void PageManager::set_auto_rotation_manual_mode(bool manual_mode)
{
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->set_auto_rotation_manual_mode(
			manual_mode);
	}
}

void PageManager::set_runtime_auto_rotation_active(bool active)
{
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->set_confirmed(
			SettingsToggleId::AUTO_ROTATION, active);
	}
}

void PageManager::set_runtime_panel_active(bool active)
{
	if (_settings_dashboard_view) {
		_settings_dashboard_view->set_screen_tile_active(active);
	}
}

void PageManager::apply_settings_service_runtime_state(
	const SettingsServiceRuntimeState &state)
{
	if (_settings_toggle_mediator) {
		_settings_toggle_mediator->apply_service_runtime_state(state);
	}
}

void PageManager::refresh_background_images(
	const BackgroundAssets &background_assets)
{
	if (_is_tearing_down) {
		return;
	}
	if (_main_dashboard_view) {
		_main_dashboard_view->replace_background(
			background_assets.main.get());
	}
	if (_settings_dashboard_view) {
		_settings_dashboard_view->replace_background(
			background_assets.settings.get());
	}
}
}
