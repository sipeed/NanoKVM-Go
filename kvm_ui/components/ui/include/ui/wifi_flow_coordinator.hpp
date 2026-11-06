#pragma once

#include <cstdint>
#include <functional>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "ui/page_manager_actions.hpp"
#include "ui/settings_route.hpp"
#include "ui/wifi_page_action_policy.hpp"

namespace ui
{

class PageRegistry;
class SettingsNavigator;
struct WifiScanResult;

struct WifiConnectivityViewModel {
	bool active { true };
	bool boot_reconnect_pending { false };
};

class WifiFlowCoordinator final {
    public:
	struct Callbacks {
		std::function<void(WifiConnectivityViewModel)>
			render_connectivity;
		std::function<void()> on_boot_reconnect_timeout;
	};

	WifiFlowCoordinator(PageRegistry &page_registry,
			    SettingsNavigator &settings_navigator,
			    const PageManagerActions::Wifi &actions,
			    bool wifi_config_mode_keyboard,
			    uint32_t boot_reconnect_timeout_ms,
			    Callbacks callbacks = {});
	WifiFlowCoordinator(PageRegistry &page_registry,
			    SettingsNavigator &settings_navigator,
			    const PageManagerActions::Wifi &actions,
			    const PageManagerActions::SsidRaster &ssid_raster_actions,
			    bool wifi_config_mode_keyboard,
			    uint32_t boot_reconnect_timeout_ms,
			    Callbacks callbacks = {});
	~WifiFlowCoordinator();

	WifiFlowCoordinator(const WifiFlowCoordinator &) = delete;
	WifiFlowCoordinator &operator=(const WifiFlowCoordinator &) = delete;

	void bind_pages();
	void begin_shutdown();
	void shutdown();

	void show_wifi();
	void show_wifi_config();
	void set_scan_loading(std::string_view message = "Scanning...");
	void set_scan_results(const std::vector<WifiScanResult> &results);
	void set_scan_empty(std::string_view message = "No WiFi found");
	void set_scan_error(std::string_view message);
	void set_connect_loading(std::string_view message = "");
	void set_connect_success(std::string_view message = "Connected");
	void set_connect_failure(std::string_view message = "Connect failed");
	void set_ap_qr_payload(const std::string &payload);
	void set_qr_title(std::string_view title);
	void set_qr_payload(const std::string &payload);
	void set_ap_stop_completed_generation(uint64_t generation);
	void set_runtime_wifi_state(WifiPageVisualState state,
					    std::string_view status_text, bool busy,
					    std::string_view wifi_ssid,
					    std::string_view wifi_ip,
					    bool wifi_saved_network_config,
					    bool show_qr,
				    const std::string &wifi_qr_payload,
				    const std::string &wifi_web_url);
	void set_runtime_wifi_active(bool active);
	void accept_prepared_ssid_raster(
		const SsidPreparedRasterResult &result);
	void set_boot_reconnect_pending(bool pending);

	uint32_t boot_reconnect_timeout_ms() const noexcept;

    private:
	static constexpr uint32_t kDefaultBootReconnectTimeoutMs = 15000;

	PageRegistry &_page_registry;
	SettingsNavigator &_settings_navigator;
	const PageManagerActions::Wifi &_actions;
	const PageManagerActions::SsidRaster &_ssid_raster_actions;
	const bool _wifi_config_mode_keyboard;
	const uint32_t _boot_reconnect_timeout_ms;
	Callbacks _callbacks;
	bool _is_shutting_down = false;
	WifiConnectivityViewModel _connectivity;
	bool _wifi_page_visible = false;
	// Armed only after QR cancellation has actually resumed Via; cleared after
	// leaving Via or during shutdown.
	bool _suppress_qr_cancel_reconnect_redirect = false;
	std::string _pending_ssid;
	std::string _pending_password;
	lv_timer_t *_boot_reconnect_timer = nullptr;

	void handle_route_change(std::optional<SettingsRoute> route);
	void notify_wifi_page_visibility(bool visible);
	void show_wifi_ssid_page();
	void show_wifi_manual_ssid_input();
	void show_wifi_password_input(std::string ssid);
	void begin_wifi_connect(std::string password);
	void request_wifi_connect();
	void show_wifi_qr_page();
	void return_to_wifi_main_from_config_flow();
	bool clear_boot_reconnect_pending();
	void render_connectivity();
	void start_boot_reconnect_timer();
	void cancel_boot_reconnect_timer();
	void clear_page_callbacks();

	static void boot_reconnect_timer_cb(lv_timer_t *timer);
};

} // namespace ui
