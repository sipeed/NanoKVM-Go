#include "ui/wifi_flow_coordinator.hpp"

#include <utility>

#include <spdlog/spdlog.h>

#include "ui/page_registry.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/wifi_config_page.hpp"
#include "ui/wifi_connect_method_page.hpp"
#include "ui/wifi_connect_result_page.hpp"
#include "ui/wifi_page.hpp"
#include "ui/wifi_qr_page.hpp"
#include "ui/wifi_ssid_page.hpp"

namespace ui
{

namespace
{

const PageManagerActions::SsidRaster EMPTY_SSID_RASTER_ACTIONS {};

} // namespace

WifiFlowCoordinator::WifiFlowCoordinator(
	PageRegistry &page_registry, SettingsNavigator &settings_navigator,
	const PageManagerActions::Wifi &actions, bool wifi_config_mode_keyboard,
	uint32_t boot_reconnect_timeout_ms, Callbacks callbacks)
	: WifiFlowCoordinator(page_registry, settings_navigator, actions,
			      EMPTY_SSID_RASTER_ACTIONS, wifi_config_mode_keyboard,
			      boot_reconnect_timeout_ms, std::move(callbacks))
{
}

WifiFlowCoordinator::WifiFlowCoordinator(
	PageRegistry &page_registry, SettingsNavigator &settings_navigator,
	const PageManagerActions::Wifi &actions,
	const PageManagerActions::SsidRaster &ssid_raster_actions,
	bool wifi_config_mode_keyboard,
	uint32_t boot_reconnect_timeout_ms, Callbacks callbacks)
	: _page_registry(page_registry)
	, _settings_navigator(settings_navigator)
	, _actions(actions)
	, _ssid_raster_actions(ssid_raster_actions)
	, _wifi_config_mode_keyboard(wifi_config_mode_keyboard)
	, _boot_reconnect_timeout_ms(boot_reconnect_timeout_ms == 0 ?
					     kDefaultBootReconnectTimeoutMs :
					     boot_reconnect_timeout_ms)
	, _callbacks(std::move(callbacks))
{
	_settings_navigator.set_route_change_callback(
		[this](std::optional<SettingsRoute> route) {
			handle_route_change(route);
		});
}

WifiFlowCoordinator::~WifiFlowCoordinator()
{
	shutdown();
}

void WifiFlowCoordinator::bind_pages()
{
	if (_is_shutting_down) {
		return;
	}

	if (WifiPage *wifi_page = _page_registry.wifi_page();
	    wifi_page != nullptr) {
		wifi_page->set_keyboard_config_mode(_wifi_config_mode_keyboard);
		wifi_page->on_back_click = [this]() {
			notify_wifi_page_visibility(false);
			(void)_settings_navigator.return_to_anchor();
		};
		wifi_page->on_wifi_config_click = [this]() {
			(void)_settings_navigator.push(
				SettingsRoute::wifi_connect_method());
		};
		wifi_page->on_wifi_enabled_change = [this](bool enabled) {
			const auto callback = _actions.set_enabled;
			if (callback) {
				callback(enabled);
			}
		};
		wifi_page->on_wifi_qr_dismissed = [this]() {
			const auto callback = _actions.qr_dismissed;
			if (callback) {
				callback();
			}
		};
		wifi_page->on_wifi_page_exit = [this]() {
			notify_wifi_page_visibility(false);
		};
		wifi_page->on_ssid_raster_prepare =
			[this](SsidRasterRequest request) -> bool {
				const auto callback = _ssid_raster_actions.prepare;
				if (callback) {
					return callback(std::move(request));
				}
				return false;
			};
		wifi_page->on_ssid_raster_invalidate =
			[this](uint64_t page_token) {
				const auto callback = _ssid_raster_actions.invalidate_page;
				if (callback) {
					callback(page_token);
				}
			};
	}

	if (WifiConnectMethodPage *connect_method_page =
		    _page_registry.wifi_connect_method_page();
	    connect_method_page != nullptr) {
		connect_method_page->on_back_click = [this]() {
			(void)_settings_navigator.pop();
		};
		connect_method_page->on_qr_click = [this]() {
			show_wifi_qr_page();
		};
		connect_method_page->on_key_click = [this]() {
			show_wifi_ssid_page();
		};
	}

	if (WifiQrPage *qr_page = _page_registry.wifi_qr_page();
	    qr_page != nullptr) {
		qr_page->on_close_ready = [this]() {
			if (_settings_navigator.top_route() !=
			    SettingsRoute::wifi_qr()) {
				return;
			}
			if (_settings_navigator.pop() &&
			    _settings_navigator.top_route() ==
				    SettingsRoute::wifi_connect_method()) {
				_suppress_qr_cancel_reconnect_redirect = true;
			}
		};
		qr_page->on_qr_start_request = [this]() {
			const auto callback = _actions.start_qr_ap;
			if (callback) {
				callback();
			}
		};
		qr_page->on_qr_stop_request = [this]() -> uint64_t {
			const auto callback = _actions.stop_qr_ap;
			return callback ? callback() : 0;
		};
	}

	if (WifiSsidPage *ssid_page = _page_registry.wifi_ssid_page();
	    ssid_page != nullptr) {
		ssid_page->on_back_click = [this]() {
			(void)_settings_navigator.pop();
		};
		ssid_page->on_ssid_selected = [this](std::string_view ssid) {
			show_wifi_password_input(std::string(ssid));
		};
		ssid_page->on_manual_ssid_click = [this]() {
			show_wifi_manual_ssid_input();
		};
	}

	if (WifiConfigPage *manual_ssid_page =
		    _page_registry.wifi_manual_ssid_page();
	    manual_ssid_page != nullptr) {
		manual_ssid_page->on_back_click = [this]() {
			(void)_settings_navigator.pop();
		};
		manual_ssid_page->on_ok_click =
			[this](const std::string &ssid) {
				if (!ssid.empty()) {
					show_wifi_password_input(ssid);
				}
			};
	}

	if (WifiConfigPage *password_page = _page_registry.wifi_password_page();
	    password_page != nullptr) {
		password_page->on_back_click = [this]() {
			(void)_settings_navigator.pop();
		};
		password_page->on_ok_click =
			[this](const std::string &password) {
				begin_wifi_connect(password);
			};
	}

	if (WifiConnectResultPage *connect_result_page =
		    _page_registry.wifi_connect_result_page();
	    connect_result_page != nullptr) {
		connect_result_page->on_back_click = [this]() {
			(void)_settings_navigator.pop();
		};
		connect_result_page->on_success_return_request = [this]() {
			return_to_wifi_main_from_config_flow();
		};
	}

	render_connectivity();
}

void WifiFlowCoordinator::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	_suppress_qr_cancel_reconnect_redirect = false;
	_settings_navigator.set_route_change_callback(nullptr);
	cancel_boot_reconnect_timer();
	clear_page_callbacks();
	_callbacks = {};
}

void WifiFlowCoordinator::shutdown()
{
	begin_shutdown();
}

void WifiFlowCoordinator::show_wifi()
{
	if (_is_shutting_down || _connectivity.boot_reconnect_pending) {
		return;
	}

	(void)_settings_navigator.show(SettingsRoute::wifi());
}

void WifiFlowCoordinator::show_wifi_config()
{
	if (_is_shutting_down) {
		return;
	}

	if (_settings_navigator.empty()) {
		(void)_settings_navigator.show(SettingsRoute::wifi());
	}
	const auto top_route = _settings_navigator.top_route();
	if (!top_route.has_value() || *top_route != SettingsRoute::wifi()) {
		(void)_settings_navigator.push(SettingsRoute::wifi());
	}
	(void)_settings_navigator.push(SettingsRoute::wifi_connect_method());
	show_wifi_ssid_page();
}

void WifiFlowCoordinator::set_scan_loading(std::string_view message)
{
	if (WifiSsidPage *page = _page_registry.wifi_ssid_page();
	    page != nullptr) {
		page->set_loading(message);
	}
}

void WifiFlowCoordinator::set_scan_results(
	const std::vector<WifiScanResult> &results)
{
	if (WifiSsidPage *page = _page_registry.wifi_ssid_page();
	    page != nullptr) {
		page->set_scan_results(results);
	}
}

void WifiFlowCoordinator::set_scan_empty(std::string_view message)
{
	if (WifiSsidPage *page = _page_registry.wifi_ssid_page();
	    page != nullptr) {
		page->set_empty(message);
	}
}

void WifiFlowCoordinator::set_scan_error(std::string_view message)
{
	if (WifiSsidPage *page = _page_registry.wifi_ssid_page();
	    page != nullptr) {
		page->set_error(message);
	}
}

void WifiFlowCoordinator::set_connect_loading(std::string_view message)
{
	if (WifiConnectResultPage *page =
		    _page_registry.wifi_connect_result_page();
	    page != nullptr) {
		page->set_loading(message);
	}
}

void WifiFlowCoordinator::set_connect_success(std::string_view message)
{
	if (WifiConnectResultPage *page =
		    _page_registry.wifi_connect_result_page();
	    page != nullptr) {
		page->set_success(message);
	}
}

void WifiFlowCoordinator::set_connect_failure(std::string_view message)
{
	if (WifiConnectResultPage *page =
		    _page_registry.wifi_connect_result_page();
	    page != nullptr) {
		page->set_failure(message);
	}
}

void WifiFlowCoordinator::set_ap_qr_payload(const std::string &payload)
{
	if (WifiPage *page = _page_registry.wifi_page(); page != nullptr) {
		page->set_qr_payload(payload);
	}
}

void WifiFlowCoordinator::set_qr_title(std::string_view title)
{
	if (WifiQrPage *page = _page_registry.wifi_qr_page(); page != nullptr) {
		page->set_title(title);
	}
}

void WifiFlowCoordinator::set_qr_payload(const std::string &payload)
{
	if (WifiQrPage *page = _page_registry.wifi_qr_page(); page != nullptr) {
		page->set_qr_payload(payload);
	}
}

void WifiFlowCoordinator::set_ap_stop_completed_generation(uint64_t generation)
{
	if (WifiQrPage *page = _page_registry.wifi_qr_page(); page != nullptr) {
		page->set_ap_stop_completed_generation(generation);
	}
}

void WifiFlowCoordinator::set_runtime_wifi_state(
	WifiPageVisualState state, std::string_view status_text, bool busy,
	std::string_view wifi_ssid, std::string_view wifi_ip,
	bool wifi_saved_network_config,
	bool show_qr, const std::string &wifi_qr_payload,
	const std::string &wifi_web_url)
{
	if (WifiPage *page = _page_registry.wifi_page(); page != nullptr) {
		page->set_runtime_wifi_state(state, status_text, busy, wifi_ssid, wifi_ip,
					     wifi_saved_network_config, show_qr,
					     wifi_qr_payload, wifi_web_url);
	}
}

void WifiFlowCoordinator::accept_prepared_ssid_raster(
	const SsidPreparedRasterResult &result)
{
	if (WifiPage *page = _page_registry.wifi_page(); page != nullptr) {
		page->accept_prepared_ssid_raster(result);
	}
}

void WifiFlowCoordinator::set_runtime_wifi_active(bool active)
{
	if (_is_shutting_down) {
		return;
	}

	const bool previous_active = _connectivity.active;
	_connectivity.active = active;
	if (active) {
		(void)clear_boot_reconnect_pending();
	}
	render_connectivity();
	if (active && !previous_active) {
		const auto route = _settings_navigator.top_route();
		const bool suppress_redirect =
			_suppress_qr_cancel_reconnect_redirect &&
			route == SettingsRoute::wifi_connect_method();
		if (!suppress_redirect) {
			_suppress_qr_cancel_reconnect_redirect = false;
			return_to_wifi_main_from_config_flow();
		}
	}
	if (previous_active != active) {
		spdlog::info(
			"[wifi] WIFI UI active applied: settings_tile/home_icon {}->{}",
			previous_active, active);
	}
}

void WifiFlowCoordinator::set_boot_reconnect_pending(bool pending)
{
	if (_is_shutting_down) {
		return;
	}
	if (!pending) {
		if (clear_boot_reconnect_pending()) {
			render_connectivity();
		}
		return;
	}
	if (_connectivity.boot_reconnect_pending) {
		return;
	}

	_connectivity.boot_reconnect_pending = true;
	render_connectivity();
	start_boot_reconnect_timer();
}

uint32_t WifiFlowCoordinator::boot_reconnect_timeout_ms() const noexcept
{
	return _boot_reconnect_timeout_ms;
}

void WifiFlowCoordinator::handle_route_change(std::optional<SettingsRoute> route)
{
	if (_is_shutting_down) {
		return;
	}

	if (!route.has_value() ||
	    *route != SettingsRoute::wifi_connect_method()) {
		_suppress_qr_cancel_reconnect_redirect = false;
	}
	notify_wifi_page_visibility(route.has_value() &&
				    is_wifi_settings_route(*route));
}

void WifiFlowCoordinator::notify_wifi_page_visibility(bool visible)
{
	if (_is_shutting_down || _wifi_page_visible == visible) {
		return;
	}

	_wifi_page_visible = visible;
	const auto callback = _actions.page_visibility_changed;
	if (callback) {
		callback(visible);
	}
}

void WifiFlowCoordinator::show_wifi_ssid_page()
{
	WifiSsidPage *ssid_page = _page_registry.wifi_ssid_page();
	if (ssid_page == nullptr) {
		return;
	}
	ssid_page->set_loading("Scanning...");
	(void)_settings_navigator.push(SettingsRoute::wifi_ssid());
	const auto callback = _actions.scan;
	if (callback) {
		callback();
	}
}

void WifiFlowCoordinator::show_wifi_manual_ssid_input()
{
	WifiConfigPage *manual_ssid_page =
		_page_registry.wifi_manual_ssid_page();
	if (manual_ssid_page == nullptr) {
		return;
	}
	manual_ssid_page->configure(WifiConfigPage::InputMode::Ssid);
	(void)_settings_navigator.push(SettingsRoute::wifi_manual_ssid());
}

void WifiFlowCoordinator::show_wifi_password_input(std::string ssid)
{
	WifiConfigPage *password_page = _page_registry.wifi_password_page();
	if (password_page == nullptr || ssid.empty()) {
		return;
	}
	_pending_ssid = std::move(ssid);
	password_page->configure(WifiConfigPage::InputMode::Password);
	(void)_settings_navigator.push(SettingsRoute::wifi_password());
}

void WifiFlowCoordinator::begin_wifi_connect(std::string password)
{
	WifiConnectResultPage *connect_result_page =
		_page_registry.wifi_connect_result_page();
	if (connect_result_page == nullptr || _pending_ssid.empty()) {
		return;
	}
	_pending_password = std::move(password);
	connect_result_page->set_loading();
	(void)_settings_navigator.push(SettingsRoute::wifi_connect_result());
	request_wifi_connect();
}

void WifiFlowCoordinator::request_wifi_connect()
{
	if (_pending_ssid.empty()) {
		return;
	}
	const auto callback = _actions.connect;
	if (callback) {
		callback(_pending_ssid, _pending_password);
		return;
	}
	if (WifiConnectResultPage *page =
		    _page_registry.wifi_connect_result_page();
	    page != nullptr) {
		page->set_failure("Connect handler missing");
	}
}

void WifiFlowCoordinator::show_wifi_qr_page()
{
	WifiQrPage *qr_page = _page_registry.wifi_qr_page();
	WifiPage *wifi_page = _page_registry.wifi_page();
	if (qr_page == nullptr || wifi_page == nullptr) {
		return;
	}
	qr_page->set_title("Connect to AP");
	qr_page->set_qr_payload(wifi_page->qr_payload());
	(void)_settings_navigator.push(SettingsRoute::wifi_qr());
}

void WifiFlowCoordinator::return_to_wifi_main_from_config_flow()
{
	const auto route = _settings_navigator.top_route();
	if (!route.has_value() || !is_wifi_config_flow_route(*route)) {
		return;
	}
	if (route->kind == SettingsRouteKind::WifiQr) {
		const auto callback = _actions.stop_qr_ap_after_success;
		if (callback) {
			callback();
		}
	}
	(void)_settings_navigator.push(SettingsRoute::wifi());
}

bool WifiFlowCoordinator::clear_boot_reconnect_pending()
{
	if (!_connectivity.boot_reconnect_pending) {
		cancel_boot_reconnect_timer();
		return false;
	}

	_connectivity.boot_reconnect_pending = false;
	cancel_boot_reconnect_timer();
	return true;
}

void WifiFlowCoordinator::render_connectivity()
{
	const auto callback = _callbacks.render_connectivity;
	if (callback) {
		callback(_connectivity);
	}
}

void WifiFlowCoordinator::start_boot_reconnect_timer()
{
	cancel_boot_reconnect_timer();
	_boot_reconnect_timer = lv_timer_create(
		boot_reconnect_timer_cb, _boot_reconnect_timeout_ms, this);
	if (_boot_reconnect_timer != nullptr) {
		lv_timer_set_repeat_count(_boot_reconnect_timer, 1);
	}
}

void WifiFlowCoordinator::cancel_boot_reconnect_timer()
{
	if (_boot_reconnect_timer != nullptr) {
		lv_timer_delete(_boot_reconnect_timer);
		_boot_reconnect_timer = nullptr;
	}
}

void WifiFlowCoordinator::clear_page_callbacks()
{
	if (WifiPage *page = _page_registry.wifi_page(); page != nullptr) {
		page->on_back_click = nullptr;
		page->on_wifi_enabled_change = nullptr;
		page->on_wifi_config_click = nullptr;
		page->on_wifi_qr_dismissed = nullptr;
		page->on_wifi_page_exit = nullptr;
		page->on_ssid_raster_prepare = nullptr;
		page->on_ssid_raster_invalidate = nullptr;
	}
	if (WifiConnectMethodPage *page =
		    _page_registry.wifi_connect_method_page();
	    page != nullptr) {
		page->on_back_click = nullptr;
		page->on_qr_click = nullptr;
		page->on_key_click = nullptr;
	}
	if (WifiQrPage *page = _page_registry.wifi_qr_page(); page != nullptr) {
		page->on_close_ready = nullptr;
		page->on_qr_start_request = nullptr;
		page->on_qr_stop_request = nullptr;
	}
	if (WifiSsidPage *page = _page_registry.wifi_ssid_page();
	    page != nullptr) {
		page->on_back_click = nullptr;
		page->on_ssid_selected = nullptr;
		page->on_manual_ssid_click = nullptr;
	}
	if (WifiConfigPage *page = _page_registry.wifi_manual_ssid_page();
	    page != nullptr) {
		page->on_back_click = nullptr;
		page->on_ok_click = nullptr;
	}
	if (WifiConfigPage *page = _page_registry.wifi_password_page();
	    page != nullptr) {
		page->on_back_click = nullptr;
		page->on_ok_click = nullptr;
	}
	if (WifiConnectResultPage *page =
		    _page_registry.wifi_connect_result_page();
	    page != nullptr) {
		page->on_back_click = nullptr;
		page->on_success_return_request = nullptr;
	}
}

void WifiFlowCoordinator::boot_reconnect_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}

	auto *self = static_cast<WifiFlowCoordinator *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down ||
	    self->_boot_reconnect_timer != timer) {
		return;
	}

	self->_boot_reconnect_timer = nullptr;
	if (!self->clear_boot_reconnect_pending()) {
		return;
	}
	self->render_connectivity();
	const auto callback = self->_callbacks.on_boot_reconnect_timeout;
	if (callback) {
		callback();
	}
}

} // namespace ui
