#pragma once

#include <cstdint>
#include <functional>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include "ui/bcona.hpp"
#include "ui/bconb.hpp"
#include "ui/scona.hpp"
#include "ui/ssid_info_row.hpp"
#include "ui/ssid_prepared_raster.hpp"
#include "ui/subpage_base.hpp"
#include "ui/wifi_page_action_policy.hpp"
#include "ui/wifi_ssid_font.hpp"

namespace ui
{

class WifiPage : public SubPageBase {
    public:
	explicit WifiPage(const lv_font_t *ssid_font = nullptr,
				  bool enable_ssid_scroll = false,
				  uint32_t ssid_image_cache_kb = 0,
				  SsidFontRenderMode ssid_font_render_mode =
					  SsidFontRenderMode::CompiledFallback);
	~WifiPage() override;

	void destroy() override;
	void on_enter() override;
	void on_pause() override;
	void on_resume() override;
	void on_exit() override;
	void set_qr_payload(const std::string &payload);
	const std::string &qr_payload() const
	{
		return _qr_payload;
	}
	void set_runtime_status(bool wifi_enabled, bool wifi_ap_mode,
				bool wifi_ap_client_connected,
				const std::string &wifi_qr_payload,
				const std::string &wifi_web_url);
	void set_runtime_wifi_state(WifiPageVisualState state,
					    std::string_view status_text, bool busy,
					    std::string_view wifi_ssid,
					    std::string_view wifi_ip,
					    bool wifi_saved_network_config,
					    bool show_qr,
					    const std::string &wifi_qr_payload,
				    const std::string &wifi_web_url);
	void set_wifi_enabled(bool enabled);
	void set_busy(bool busy);
	void set_keyboard_config_mode(bool enabled)
	{
		_keyboard_config_mode = enabled;
	}
	bool is_wifi_enabled() const
	{
		return _wifi_enabled;
	}
	void accept_prepared_ssid_raster(const SsidPreparedRasterResult &result);

	std::function<void(bool)> on_wifi_enabled_change;
	std::function<void()> on_wifi_config_click;
	std::function<void()> on_wifi_qr_dismissed;
	std::function<void()> on_wifi_page_exit;
	std::function<bool(SsidRasterRequest)> on_ssid_raster_prepare;
	std::function<void(uint64_t)> on_ssid_raster_invalidate;

    private:
	static constexpr uint32_t ACTIVE_COLOR = 0x2B7FFF;
	static constexpr const char *DEFAULT_QR_TITLE = "Connect to AP";
	static constexpr const char *WEB_QR_TITLE = "Configure WiFi";
	static constexpr const char *EMPTY_QR_PLACEHOLDER =
		"Wi-Fi credentials are not configured";
	static constexpr uint16_t SUBPAGE_TOP_PADDING = 12;
	static constexpr uint16_t INFORMATION_GROUP_BOTTOM_MARGIN = 4;
	// Keep the Wi-Fi actions in the same y=164 slot as About's Reset control.
	static constexpr uint16_t ACTION_ROW_RESET_SLOT_OFFSET =
		HEIGHT - SUBPAGE_TOP_PADDING - BConA::HEIGHT - SUBPAGE_TOP_PADDING -
		BConA::HEIGHT - 2 * SsidInfoRow::HEIGHT -
		INFORMATION_GROUP_BOTTOM_MARGIN;
	// SubPageBase starts children at LEFT_MARGIN. This additional margin
	// centers the 230px action row within the 270px logical canvas.
	static constexpr uint16_t ACTION_ROW_CENTERING_OFFSET =
		(WIDTH - SConA::WIDTH) / 2 - LEFT_MARGIN;
	static constexpr uint32_t SSID_INTERACTION_IDLE_MS = 500;

	bool _wifi_enabled = false;
	bool _wifi_ap_mode = false;
	bool _wifi_ap_client_connected = false;
	bool _wifi_station_connected = false;
	bool _busy = false;
	bool _show_qr_when_ap_mode = false;
	bool _runtime_show_qr_active = false;
	bool _wifi_toggle_pending = false;
	bool _wifi_toggle_pending_target = false;
	bool _wifi_toggle_waiting_for_off_stable = false;
	bool _wifi_off_stable_tracking = false;
	uint32_t _wifi_off_stable_since = 0;
	bool _keyboard_config_mode = false;
	bool _wifi_saved_network_config = false;
	WifiPageVisualState _visual_state = WifiPageVisualState::Off;
	std::string _wifi_ssid;
	std::string _wifi_ip;
	std::string _ssid_label_text { "--" };
	uint32_t _ssid_image_cache_kb { 0 };
	SsidFontRenderMode _ssid_font_render_mode {
		SsidFontRenderMode::CompiledFallback
	};
	bool _enable_ssid_scroll { false };
	bool _ssid_image_cache_failed { false };
	bool _ssid_presentation_applied { false };
	bool _ssid_page_active { false };
	bool _ssid_interaction_held { false };
	bool _ssid_raster_request_pending { false };
	uint64_t _ssid_page_token { 0 };
	uint64_t _ssid_generation { 0 };
	std::optional<SsidRasterIdentity> _ssid_desired_identity;
	std::optional<SsidRasterIdentity> _ssid_ready_identity;
	std::optional<SsidRasterIdentity> _ssid_failed_identity;
	std::optional<SsidPreparedRasterResult> _deferred_prepared_ssid_raster;
	lv_timer_t *_ssid_interaction_idle_timer = nullptr;
	lv_obj_t *_information_group = nullptr;
	SsidInfoRow *_ssid_info_row = nullptr;
	SConA *_ip_label = nullptr;
	BConB *_cached_network_btn = nullptr;
	BConB *_new_connection_btn = nullptr;
	BConA *_open_setup_btn = nullptr;
	lv_obj_t *_action_row = nullptr;

	lv_obj_t *_qr_popup = nullptr;
	lv_obj_t *_qr_title_label = nullptr;
	lv_obj_t *_qr_code = nullptr;
	std::string _qr_payload;
	std::string _qr_title = DEFAULT_QR_TITLE;

	void build_content() override;
	void update_ssid_label();
	void request_prepared_ssid_raster();
	void apply_runtime_ssid_fallback();
	void invalidate_prepared_ssid_raster();
	void install_prepared_ssid_raster_now(
		const SsidPreparedRasterResult &result);
	void install_deferred_prepared_ssid_raster();
	void hold_ssid_for_interaction() noexcept;
	void resume_ssid_after_idle() noexcept;
	void clear_ssid_interaction_idle_timer() noexcept;
	void add_ssid_interaction_listener(lv_obj_t *object);
	void update_ip_label();
	bool uses_runtime_tiny_ttf() const noexcept;
	void update_controls();
	void update_interaction_state();
	bool is_setup_interaction_allowed() const;
	bool wifi_paired_actions_visible() const;
	void open_wifi_config();
	void start_toggle_pending(bool requested_enabled);
	void clear_toggle_pending();
	void set_toggle_pending_indicator(bool pending);
	void clear_off_stable_confirmation();
	bool update_off_stable_confirmation(bool runtime_busy);
	bool has_pending_operation() const;
	void finish_toggle_pending_if_target(bool runtime_busy,
						     bool off_stable_confirmed);
	void show_qr_popup();
	void hide_qr_popup(bool notify = false);
	void refresh_qr_code();
	void on_subpage_destroy() override;
	static void ssid_interaction_event_cb(lv_event_t *event);
	static void ssid_interaction_idle_timer_cb(lv_timer_t *timer);

	friend class WifiPageTestAccess;
};

} // namespace ui
