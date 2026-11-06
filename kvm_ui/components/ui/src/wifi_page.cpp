#include <atomic>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <limits>

#include "ui/wifi_page.hpp"

#include <lvgl.h>
#include <spdlog/spdlog.h>

#include "lv_safe.hpp"
#include "ui/ssid_info_row.hpp"
#include "ui/wifi_latency_probe.hpp"
#include "ui/wifi_ssid_display_policy.hpp"

extern "C" {
	LV_IMAGE_DECLARE(img_settings_main_subwifi_wifi);
	LV_IMAGE_DECLARE(ci_add_plus);
	LV_FONT_DECLARE(MontserratBold_20);
}

namespace ui
{

namespace
{

constexpr uint32_t WIFI_OFF_STABLE_CONFIRM_MS = 3000;
constexpr std::size_t BYTES_PER_KILOBYTE = 1024U;
constexpr uint64_t SSID_RUNTIME_FONT_GENERATION = 1;
std::atomic<uint64_t> next_ssid_page_token { 1 };

uint64_t allocate_ssid_page_token() noexcept
{
	uint64_t token = next_ssid_page_token.fetch_add(1, std::memory_order_relaxed);
	if (token == 0) {
		token = next_ssid_page_token.fetch_add(1, std::memory_order_relaxed);
	}
	return token;
}

const char *visual_state_name(WifiPageVisualState state)
{
	switch (state) {
	case WifiPageVisualState::Off:
		return "Off";
	case WifiPageVisualState::Connected:
		return "Connected";
	case WifiPageVisualState::Connecting:
		return "Connecting";
	case WifiPageVisualState::ApConnect:
		return "ApConnect";
	case WifiPageVisualState::WebConfigure:
		return "WebConfigure";
	case WifiPageVisualState::Error:
		return "Error";
	}
	return "Unknown";
}

bool is_off_visual_state(WifiPageVisualState state)
{
	return state == WifiPageVisualState::Off ||
	       state == WifiPageVisualState::Error;
}

bool is_enabled_visual_state(WifiPageVisualState state)
{
	return state == WifiPageVisualState::Connected ||
	       state == WifiPageVisualState::ApConnect ||
	       state == WifiPageVisualState::WebConfigure;
}

} // namespace

WifiPage::WifiPage(const lv_font_t *ssid_font, bool enable_ssid_scroll,
			   uint32_t ssid_image_cache_kb,
			   SsidFontRenderMode ssid_font_render_mode)
	: SubPageBase(PageType::SUBPAGE_WIFI, "WI-FI", 0)
	, _ssid_image_cache_kb(ssid_image_cache_kb)
	, _ssid_font_render_mode(ssid_font_render_mode)
	, _enable_ssid_scroll(enable_ssid_scroll)
	, _ssid_page_token(allocate_ssid_page_token())
{
	(void)ssid_font;
}

WifiPage::~WifiPage() = default;

void WifiPage::set_qr_payload(const std::string &payload)
{
	_qr_payload = payload;
	refresh_qr_code();
}

void WifiPage::set_runtime_status(bool wifi_enabled, bool wifi_ap_mode,
				  bool wifi_ap_client_connected,
				  const std::string &wifi_qr_payload,
				  const std::string &wifi_web_url)
{
	WifiPageVisualState state = WifiPageVisualState::Off;
	if (wifi_enabled && wifi_ap_mode && wifi_ap_client_connected) {
		state = WifiPageVisualState::WebConfigure;
	} else if (wifi_enabled && wifi_ap_mode) {
		state = WifiPageVisualState::ApConnect;
	} else if (wifi_enabled) {
		state = WifiPageVisualState::Connected;
	}
	set_runtime_wifi_state(state, {}, false, {}, {}, false,
				       wifi_enabled && wifi_ap_mode, wifi_qr_payload,
				       wifi_web_url);
}

void WifiPage::set_runtime_wifi_state(WifiPageVisualState state,
					      std::string_view status_text, bool busy,
					      std::string_view wifi_ssid,
					      std::string_view wifi_ip,
					      bool wifi_saved_network_config,
					      bool show_qr,
				      const std::string &wifi_qr_payload,
				      const std::string &wifi_web_url)
{
	(void)status_text;
	const WifiPageVisualState previous_state = _visual_state;
	const bool previous_show_qr = _runtime_show_qr_active;
	const bool previous_busy = _busy;
	const bool previous_enabled = _wifi_enabled;
	const bool previous_ap_mode = _wifi_ap_mode;
	const bool previous_ap_client_connected = _wifi_ap_client_connected;
	const bool previous_toggle_pending = _wifi_toggle_pending;
	const std::size_t previous_qr_payload_len = _qr_payload.size();
	const bool runtime_busy = busy;

	_visual_state = state;
	_busy = runtime_busy;
	_wifi_ssid = wifi_ssid;
	_wifi_ip = wifi_ip;
	_wifi_saved_network_config = wifi_saved_network_config;
	_runtime_show_qr_active = show_qr;

	switch (_visual_state) {
	case WifiPageVisualState::Off:
		_wifi_enabled = false;
		_wifi_ap_mode = false;
		_wifi_ap_client_connected = false;
		_wifi_station_connected = false;
		_qr_title = DEFAULT_QR_TITLE;
		_qr_payload = wifi_qr_payload;
		break;
	case WifiPageVisualState::Connected:
		_wifi_enabled = true;
		_wifi_ap_mode = false;
		_wifi_ap_client_connected = false;
		_wifi_station_connected = true;
		_qr_title = DEFAULT_QR_TITLE;
		_qr_payload = wifi_qr_payload;
		break;
	case WifiPageVisualState::Connecting:
		_wifi_enabled = true;
		_wifi_ap_mode = false;
		_wifi_ap_client_connected = false;
		_wifi_station_connected = false;
		_qr_title = DEFAULT_QR_TITLE;
		_qr_payload = wifi_qr_payload;
		break;
	case WifiPageVisualState::ApConnect:
		_wifi_enabled = true;
		_wifi_ap_mode = true;
		_wifi_ap_client_connected = false;
		_wifi_station_connected = false;
		_qr_title = DEFAULT_QR_TITLE;
		_qr_payload = wifi_qr_payload;
		break;
	case WifiPageVisualState::WebConfigure:
		_wifi_enabled = true;
		_wifi_ap_mode = true;
		_wifi_ap_client_connected = true;
		_wifi_station_connected = false;
		_qr_title = WEB_QR_TITLE;
		_qr_payload = wifi_web_url.empty() ? wifi_qr_payload :
						     wifi_web_url;
		break;
	case WifiPageVisualState::Error:
		_wifi_enabled = false;
		_wifi_ap_mode = false;
		_wifi_ap_client_connected = false;
		_wifi_station_connected = false;
		_qr_title = DEFAULT_QR_TITLE;
		_qr_payload = wifi_qr_payload;
		break;
	}

	if (runtime_busy && !previous_busy && !_wifi_toggle_pending) {
		_wifi_toggle_pending = true;
		_wifi_toggle_pending_target = _wifi_enabled;
		_wifi_toggle_waiting_for_off_stable = false;
		set_toggle_pending_indicator(true);
	}

	const bool off_stable_confirmed =
		update_off_stable_confirmation(runtime_busy);
	finish_toggle_pending_if_target(runtime_busy, off_stable_confirmed);

	if (!_wifi_toggle_pending) {
		clear_off_stable_confirmation();
	}

	_busy = runtime_busy || has_pending_operation();

	const bool waiting_for_off_confirmation = _wifi_toggle_waiting_for_off_stable;
	if (_keyboard_config_mode || waiting_for_off_confirmation || !show_qr ||
	    !_wifi_ap_mode) {
		hide_qr_popup();
		_show_qr_when_ap_mode = false;
	} else if (_visual_state == WifiPageVisualState::ApConnect &&
		   (previous_state != _visual_state || !previous_show_qr ||
		    _show_qr_when_ap_mode)) {
		show_qr_popup();
		_show_qr_when_ap_mode = false;
	} else if (_qr_popup != nullptr) {
		refresh_qr_code();
	}

	update_ssid_label();
	update_ip_label();
	update_controls();
	update_interaction_state();

	if (previous_state != _visual_state || previous_busy != _busy ||
	    previous_show_qr != _runtime_show_qr_active ||
	    previous_enabled != _wifi_enabled ||
	    previous_ap_mode != _wifi_ap_mode ||
	    previous_ap_client_connected != _wifi_ap_client_connected ||
	    previous_toggle_pending != _wifi_toggle_pending ||
	    previous_qr_payload_len != _qr_payload.size()) {
		spdlog::info(
				"[wifi] WIFI UI visual applied: state {}->{} runtime_busy={} "
				"effective_busy {}->{} show_qr {}->{} enabled {}->{} "
				"ap_mode {}->{} ap_client {}->{} toggle_pending {}->{} "
				"saved_network={} ssid_len={} qr_payload_len {}->{} web_url_len={}",
			visual_state_name(previous_state),
			visual_state_name(_visual_state), runtime_busy,
			previous_busy, _busy, previous_show_qr,
			_runtime_show_qr_active, previous_enabled,
			_wifi_enabled, previous_ap_mode, _wifi_ap_mode,
				previous_ap_client_connected, _wifi_ap_client_connected,
				previous_toggle_pending, _wifi_toggle_pending,
				_wifi_saved_network_config, _wifi_ssid.size(),
				previous_qr_payload_len, wifi_qr_payload.size(),
				wifi_web_url.size());
	}
}

void WifiPage::set_wifi_enabled(bool enabled)
{
	_wifi_enabled = enabled;
	_wifi_ap_mode = enabled;
	_wifi_station_connected = false;
	_visual_state = enabled ? WifiPageVisualState::ApConnect :
					  WifiPageVisualState::Off;
	if (!enabled) {
		_wifi_ssid.clear();
	}
	_wifi_ip.clear();
	clear_off_stable_confirmation();
	finish_toggle_pending_if_target(false, true);
	if (!_wifi_enabled) {
		hide_qr_popup();
	}
	update_ssid_label();
	update_ip_label();
	update_controls();
	update_interaction_state();
}

void WifiPage::set_busy(bool busy)
{
	const bool previous_busy = _busy;
	_busy = busy;
	if (!_busy && !has_pending_operation()) {
		clear_off_stable_confirmation();
	}
	if (previous_busy != _busy) {
		spdlog::info("[wifi] WIFI UI busy applied: {}->{}",
			     previous_busy, _busy);
	}
	update_controls();
	update_interaction_state();
}

void WifiPage::build_content()
{
	WifiLatencyProbe::initialize();
	_information_group = lv_obj_create(_scroll_content);
	lv_obj_set_width(_information_group, SConA::WIDTH);
	lv_obj_set_height(_information_group, LV_SIZE_CONTENT);
	lv_obj_set_style_bg_opa(_information_group, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_information_group, 0, 0);
	lv_obj_set_style_pad_all(_information_group, 0, 0);
	lv_obj_set_flex_flow(_information_group, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_style_pad_row(_information_group, 0, 0);
	lv_obj_clear_flag(_information_group, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_information_group, LV_OBJ_FLAG_SCROLLABLE);

	auto ssid_info = std::make_unique<SsidInfoRow>(_information_group);
	_ssid_info_row = ssid_info.get();
	ssid_info->create();
	// A WifiPage is constructed hidden before it becomes the active route. Keep
	// both the compiled fallback and a later prepared raster static until enter.
	ssid_info->pause_ssid_scroll();
	_components.push_back(std::move(ssid_info));
	if (auto *back_button = title_button(); back_button != nullptr) {
		add_ssid_interaction_listener(back_button->get_obj());
	}
	add_ssid_interaction_listener(_scroll_content);

	auto ip_info = std::make_unique<SConA>(_information_group, "",
					       SConA::TextAlign::LEFT);
	_ip_label = ip_info.get();
	ip_info->create();
	lv_obj_set_style_pad_all(ip_info->get_obj(), 0, 0);
	lv_obj_set_style_margin_bottom(ip_info->get_obj(),
				       INFORMATION_GROUP_BOTTOM_MARGIN, 0);
	_components.push_back(std::move(ip_info));

	_action_row = lv_obj_create(_scroll_content);
	lv_obj_set_size(_action_row, SConA::WIDTH, BConB::HEIGHT);
	lv_obj_set_style_bg_opa(_action_row, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_action_row, 0, 0);
	lv_obj_set_style_pad_all(_action_row, 0, 0);
	lv_obj_set_style_margin_left(_action_row, ACTION_ROW_CENTERING_OFFSET, 0);
	lv_obj_set_style_margin_top(_action_row, ACTION_ROW_RESET_SLOT_OFFSET, 0);
	lv_obj_set_scrollbar_mode(_action_row, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_action_row, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_action_row, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_set_flex_flow(_action_row, LV_FLEX_FLOW_ROW);
	lv_obj_set_flex_align(_action_row, LV_FLEX_ALIGN_SPACE_BETWEEN,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

	auto cached_network = std::make_unique<BConB>(_action_row, ACTIVE_COLOR);
	_cached_network_btn = cached_network.get();
	cached_network->create();
	cached_network->set_image(&img_settings_main_subwifi_wifi);
	cached_network->on_click([this](BConB *) {
		if (_busy || has_pending_operation() ||
		    !wifi_paired_actions_visible() ||
		    _visual_state == WifiPageVisualState::Connecting) {
			return;
		}

		const bool requested_enabled = is_off_visual_state(_visual_state);
		start_toggle_pending(requested_enabled);
		if (on_wifi_enabled_change) {
			on_wifi_enabled_change(requested_enabled);
			return;
		}
		set_wifi_enabled(requested_enabled);
	});
	add_ssid_interaction_listener(cached_network->get_obj());
	_components.push_back(std::move(cached_network));

	auto new_connection = std::make_unique<BConB>(_action_row, ACTIVE_COLOR);
	_new_connection_btn = new_connection.get();
	new_connection->create();
	new_connection->set_image(&ci_add_plus);
	new_connection->set_active(true, ControlState::ApplyPolicy::IMMEDIATE);
	new_connection->on_click([this](BConB *) {
		open_wifi_config();
	});
	add_ssid_interaction_listener(new_connection->get_obj());
	_components.push_back(std::move(new_connection));

	auto open_setup =
		std::make_unique<BConA>(_action_row, "Connect Wi-Fi ", ACTIVE_COLOR);
	_open_setup_btn = open_setup.get();
	open_setup->create();
	open_setup->set_text_font(&MontserratBold_20);
	open_setup->set_left_image(&ci_add_plus);
	open_setup->set_active(true, ControlState::ApplyPolicy::IMMEDIATE);
	open_setup->on_click([this](BConA *) {
		open_wifi_config();
	});
	add_ssid_interaction_listener(open_setup->get_obj());
	_components.push_back(std::move(open_setup));

	set_toggle_pending_indicator(_wifi_toggle_pending);
	update_ssid_label();
	update_ip_label();
	update_controls();
	update_interaction_state();
}

void WifiPage::destroy()
{
	_ssid_page_active = false;
	_ssid_interaction_held = false;
	clear_ssid_interaction_idle_timer();
	invalidate_prepared_ssid_raster();
	clear_toggle_pending();
	clear_off_stable_confirmation();
	if (on_wifi_page_exit) {
		on_wifi_page_exit();
	}
	on_wifi_enabled_change = nullptr;
	on_wifi_config_click = nullptr;
	on_wifi_qr_dismissed = nullptr;
	on_wifi_page_exit = nullptr;
	on_ssid_raster_prepare = nullptr;
	on_ssid_raster_invalidate = nullptr;
	hide_qr_popup();
	SubPageBase::destroy();
	_ssid_presentation_applied = false;
	_ssid_image_cache_failed = false;
	_ssid_raster_request_pending = false;
}

void WifiPage::on_enter()
{
	_ssid_page_active = true;
	hold_ssid_for_interaction();
	update_ssid_label();
	resume_ssid_after_idle();
}

void WifiPage::on_pause()
{
	_ssid_page_active = false;
	_ssid_interaction_held = false;
	clear_ssid_interaction_idle_timer();
	if (_ssid_info_row != nullptr) {
		_ssid_info_row->pause_ssid_scroll();
	}
}

void WifiPage::on_resume()
{
	_ssid_page_active = true;
	hold_ssid_for_interaction();
	update_ssid_label();
	resume_ssid_after_idle();
}

void WifiPage::on_exit()
{
	_ssid_page_active = false;
	_ssid_interaction_held = false;
	clear_ssid_interaction_idle_timer();
	if (_ssid_info_row != nullptr) {
		_ssid_info_row->pause_ssid_scroll();
	}
	clear_toggle_pending();
	clear_off_stable_confirmation();
	_busy = false;
	update_controls();
	update_interaction_state();
}

void WifiPage::update_ssid_label()
{
	std::string display_text = "--";
	if (_wifi_station_connected && !_wifi_ssid.empty()) {
		const WifiSsidDisplay display = make_wifi_ssid_display(_wifi_ssid);
		if (display.visible) {
			display_text = display.text;
		}
	}
	const bool text_changed = display_text != _ssid_label_text;
	_ssid_label_text = std::move(display_text);

	if (!uses_runtime_tiny_ttf()) {
		if (!text_changed && _ssid_presentation_applied) {
			return;
		}
		_ssid_presentation_applied = false;
		if (_ssid_info_row == nullptr) {
			return;
		}
		_ssid_info_row->clear_raster_marquee();
		_ssid_info_row->set_fallback_scroll_enabled(_enable_ssid_scroll);
		if (_ssid_info_row->set_fallback_text(_ssid_label_text.c_str())) {
			_ssid_presentation_applied = true;
			return;
		}
		spdlog::warn("[wifi] SSID information fallback label is unavailable");
		return;
	}

	if (!_wifi_station_connected || _ssid_label_text == "--") {
		if (!text_changed && _ssid_presentation_applied &&
		    !_ssid_desired_identity) {
			return;
		}
		invalidate_prepared_ssid_raster();
		apply_runtime_ssid_fallback();
		return;
	}
	request_prepared_ssid_raster();
}

void WifiPage::apply_runtime_ssid_fallback()
{
	_ssid_presentation_applied = false;
	if (_ssid_info_row == nullptr) {
		return;
	}
	_ssid_info_row->clear_raster_marquee();
	_ssid_info_row->set_fallback_scroll_enabled(_enable_ssid_scroll);
	if (_ssid_info_row->set_fallback_text(_ssid_label_text.c_str())) {
		_ssid_presentation_applied = true;
	}
}

void WifiPage::request_prepared_ssid_raster()
{
	if (_ssid_info_row == nullptr) {
		return;
	}
	SsidRasterIdentity identity;
	identity.page_token = _ssid_page_token;
	identity.font_generation = SSID_RUNTIME_FONT_GENERATION;
	identity.viewport_width = SsidInfoRow::VALUE_WIDTH;
	identity.height = SsidInfoRow::HEIGHT;
	identity.font_size_px = kWifiSsidFontSizePx;
	identity.alignment = SsidRasterAlignment::Right;
	identity.scroll_enabled = _enable_ssid_scroll;
	const uint64_t configured_bytes =
		static_cast<uint64_t>(_ssid_image_cache_kb) * BYTES_PER_KILOBYTE;
	identity.byte_budget = configured_bytes <=
				       std::numeric_limits<uint32_t>::max() ?
			       static_cast<uint32_t>(configured_bytes) :
			       std::numeric_limits<uint32_t>::max();
	identity.display_text = _ssid_label_text;
	if (_ssid_desired_identity &&
	    _ssid_desired_identity->same_render_input(identity) &&
	    (_ssid_ready_identity.has_value() || _ssid_raster_request_pending ||
	     _ssid_failed_identity.has_value())) {
		return;
	}

	identity.generation = ++_ssid_generation;
	if (identity.generation == 0) {
		identity.generation = ++_ssid_generation;
	}
	_ssid_desired_identity = identity;
	_ssid_ready_identity.reset();
	_ssid_failed_identity.reset();
	_deferred_prepared_ssid_raster.reset();
	_ssid_raster_request_pending = false;
	_ssid_image_cache_failed = false;
	apply_runtime_ssid_fallback();
	if (on_ssid_raster_prepare) {
		_ssid_raster_request_pending =
			on_ssid_raster_prepare({.identity = std::move(identity)});
		if (!_ssid_raster_request_pending) {
			_ssid_failed_identity = *_ssid_desired_identity;
			_ssid_image_cache_failed = true;
		}
		return;
	}
	_ssid_failed_identity = *_ssid_desired_identity;
	_ssid_image_cache_failed = true;
}

void WifiPage::invalidate_prepared_ssid_raster()
{
	_ssid_desired_identity.reset();
	_ssid_ready_identity.reset();
	_ssid_failed_identity.reset();
	_deferred_prepared_ssid_raster.reset();
	_ssid_raster_request_pending = false;
	if (on_ssid_raster_invalidate && _ssid_page_token != 0) {
		on_ssid_raster_invalidate(_ssid_page_token);
	}
}

void WifiPage::accept_prepared_ssid_raster(
	const SsidPreparedRasterResult &result)
{
	if (!uses_runtime_tiny_ttf() || !_ssid_desired_identity ||
	    result.identity != *_ssid_desired_identity ||
	    result.identity.page_token != _ssid_page_token ||
	    result.identity.generation != _ssid_generation) {
		return;
	}
	if (_ssid_interaction_held) {
		// The worker result owns immutable pixels through a shared_ptr. Retaining
		// exactly one validated value is constant-time and avoids the LVGL-thread
		// source copy until the user's 500 ms quiet interval has elapsed.
		_deferred_prepared_ssid_raster = result;
		return;
	}
	install_prepared_ssid_raster_now(result);
}

void WifiPage::install_prepared_ssid_raster_now(
	const SsidPreparedRasterResult &result)
{
	if (!uses_runtime_tiny_ttf() || !_ssid_desired_identity ||
	    result.identity != *_ssid_desired_identity ||
	    result.identity.page_token != _ssid_page_token ||
	    result.identity.generation != _ssid_generation) {
		return;
	}
	if (!result.is_success() || _ssid_info_row == nullptr) {
		_ssid_raster_request_pending = false;
		_ssid_failed_identity = result.identity;
		_ssid_image_cache_failed = true;
		apply_runtime_ssid_fallback();
		return;
	}
	if (!_ssid_info_row->set_prepared_raster(*result.raster, _ssid_page_active)) {
		_ssid_raster_request_pending = false;
		_ssid_failed_identity = result.identity;
		_ssid_image_cache_failed = true;
		apply_runtime_ssid_fallback();
		return;
	}
	_ssid_ready_identity = result.identity;
	_ssid_failed_identity.reset();
	_ssid_raster_request_pending = false;
	_ssid_image_cache_failed = false;
	_ssid_presentation_applied = true;
}

void WifiPage::install_deferred_prepared_ssid_raster()
{
	if (!_deferred_prepared_ssid_raster.has_value()) {
		return;
	}

	SsidPreparedRasterResult result =
		std::move(*_deferred_prepared_ssid_raster);
	_deferred_prepared_ssid_raster.reset();
	install_prepared_ssid_raster_now(result);
}

void WifiPage::hold_ssid_for_interaction() noexcept
{
	_ssid_interaction_held = true;
	if (_ssid_interaction_idle_timer != nullptr) {
		lv_timer_pause(_ssid_interaction_idle_timer);
	}
	if (_ssid_info_row != nullptr) {
		_ssid_info_row->pause_ssid_scroll();
	}
}

void WifiPage::resume_ssid_after_idle() noexcept
{
	if (!_ssid_page_active) {
		return;
	}

	_ssid_interaction_held = true;
	if (_ssid_info_row != nullptr) {
		_ssid_info_row->pause_ssid_scroll();
	}
	if (_ssid_interaction_idle_timer != nullptr) {
		lv_timer_reset(_ssid_interaction_idle_timer);
		lv_timer_resume(_ssid_interaction_idle_timer);
		return;
	}

	_ssid_interaction_idle_timer = lv_timer_create(
		ssid_interaction_idle_timer_cb, SSID_INTERACTION_IDLE_MS, this);
	if (_ssid_interaction_idle_timer != nullptr) {
		lv_timer_set_repeat_count(_ssid_interaction_idle_timer, 1);
	}
}

void WifiPage::clear_ssid_interaction_idle_timer() noexcept
{
	if (_ssid_interaction_idle_timer != nullptr) {
		lv_timer_delete(_ssid_interaction_idle_timer);
		_ssid_interaction_idle_timer = nullptr;
	}
}

void WifiPage::add_ssid_interaction_listener(lv_obj_t *object)
{
	if (object == nullptr || !lv_obj_is_valid(object)) {
		return;
	}
	lv_obj_add_event_cb(object, ssid_interaction_event_cb, LV_EVENT_ALL, this);
}

void WifiPage::ssid_interaction_event_cb(lv_event_t *event)
{
	if (event == nullptr) {
		return;
	}
	auto *const self = static_cast<WifiPage *>(lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	switch (lv_event_get_code(event)) {
	case LV_EVENT_PRESSED:
	case LV_EVENT_PRESSING:
	case LV_EVENT_SCROLL_BEGIN:
	case LV_EVENT_SCROLL:
	case LV_EVENT_SCROLL_THROW_BEGIN:
		self->hold_ssid_for_interaction();
		break;
	case LV_EVENT_RELEASED:
	case LV_EVENT_PRESS_LOST:
	case LV_EVENT_INDEV_RESET:
	case LV_EVENT_SCROLL_END:
		self->resume_ssid_after_idle();
		break;
	default:
		break;
	}
}

void WifiPage::ssid_interaction_idle_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}
	auto *const self = static_cast<WifiPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr || self->_ssid_interaction_idle_timer != timer) {
		return;
	}

	// A one-shot LVGL timer is auto-deleted after this callback. Clear the owned
	// pointer before installing pixels or resuming so lifecycle reentry cannot
	// delete a timer that LVGL is already retiring.
	self->_ssid_interaction_idle_timer = nullptr;
	if (!self->_ssid_page_active) {
		return;
	}
	self->_ssid_interaction_held = false;
	self->install_deferred_prepared_ssid_raster();
	if (self->_ssid_info_row != nullptr) {
		self->_ssid_info_row->resume_ssid_scroll();
	}
}

void WifiPage::update_ip_label()
{
	if (_ip_label == nullptr) {
		return;
	}
	const char *const display_text =
		_wifi_station_connected && !_wifi_ip.empty() ? _wifi_ip.c_str() : "--";
	_ip_label->set_dual_text("IP:", display_text);
}

bool WifiPage::uses_runtime_tiny_ttf() const noexcept
{
	return _ssid_font_render_mode == SsidFontRenderMode::RuntimeTinyTtf;
}

void WifiPage::update_controls()
{
	if (_cached_network_btn != nullptr) {
		_cached_network_btn->set_active(
			_wifi_station_connected,
			ControlState::ApplyPolicy::IMMEDIATE);
	}
	if (_new_connection_btn != nullptr) {
		_new_connection_btn->set_active(true,
			ControlState::ApplyPolicy::IMMEDIATE);
	}
}

void WifiPage::start_toggle_pending(bool requested_enabled)
{
	_wifi_toggle_pending = true;
	_wifi_toggle_pending_target = requested_enabled;
	_wifi_toggle_waiting_for_off_stable = !requested_enabled;
	if (_wifi_toggle_waiting_for_off_stable) {
		clear_off_stable_confirmation();
	}
	set_toggle_pending_indicator(true);
	update_controls();
	update_interaction_state();
}

void WifiPage::clear_toggle_pending()
{
	set_toggle_pending_indicator(false);
	_wifi_toggle_pending = false;
	_wifi_toggle_waiting_for_off_stable = false;
}

void WifiPage::set_toggle_pending_indicator(bool pending)
{
	if (_cached_network_btn != nullptr) {
		_cached_network_btn->set_pending(pending);
	}
}

void WifiPage::clear_off_stable_confirmation()
{
	_wifi_off_stable_tracking = false;
	_wifi_off_stable_since = 0;
}

bool WifiPage::update_off_stable_confirmation(bool runtime_busy)
{
	const bool waiting_for_off = _wifi_toggle_waiting_for_off_stable;
	if (!waiting_for_off || runtime_busy ||
	    !is_off_visual_state(_visual_state)) {
		clear_off_stable_confirmation();
		return false;
	}

	if (!_wifi_off_stable_tracking) {
		_wifi_off_stable_tracking = true;
		_wifi_off_stable_since = lv_tick_get();
		return false;
	}

	return lv_tick_elaps(_wifi_off_stable_since) >=
	       WIFI_OFF_STABLE_CONFIRM_MS;
}

bool WifiPage::has_pending_operation() const
{
	return _wifi_toggle_pending;
}

void WifiPage::finish_toggle_pending_if_target(bool runtime_busy,
					       bool off_stable_confirmed)
{
	if (!_wifi_toggle_pending) {
		return;
	}

	if (_wifi_toggle_pending_target) {
		const bool reached_enabled_state =
			is_enabled_visual_state(_visual_state);
		const bool failed_enable = !runtime_busy &&
					   is_off_visual_state(_visual_state);
		if (reached_enabled_state || failed_enable) {
			clear_toggle_pending();
		}
		return;
	}

	if (!_wifi_toggle_waiting_for_off_stable) {
		if (!runtime_busy) {
			clear_toggle_pending();
		}
		return;
	}

	if (!runtime_busy && _visual_state == WifiPageVisualState::Error) {
		clear_toggle_pending();
		return;
	}

	if (!runtime_busy && off_stable_confirmed) {
		clear_toggle_pending();
	}
}

void WifiPage::update_interaction_state()
{
	const bool setup_interaction_allowed = is_setup_interaction_allowed();
	const bool controls_blocked = !setup_interaction_allowed;
	const bool paired_actions_visible = wifi_paired_actions_visible();
	const bool cached_network_available =
		!controls_blocked && paired_actions_visible;
	const bool show_open_setup = !paired_actions_visible;

	const auto set_hidden = [](lv_obj_t *object, bool hidden) {
		if (object == nullptr) {
			return;
		}
		if (hidden) {
			lv_obj_add_flag(object, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(object, LV_OBJ_FLAG_HIDDEN);
		}
	};
	const auto set_clickable = [](lv_obj_t *object, bool clickable) {
		if (object == nullptr) {
			return;
		}
		if (clickable) {
			lv_obj_add_flag(object, LV_OBJ_FLAG_CLICKABLE);
		} else {
			lv_obj_clear_flag(object, LV_OBJ_FLAG_CLICKABLE);
		}
	};

	lv_obj_t *cached_network =
		_cached_network_btn == nullptr ? nullptr : _cached_network_btn->get_obj();
	lv_obj_t *new_connection =
		_new_connection_btn == nullptr ? nullptr : _new_connection_btn->get_obj();
	lv_obj_t *open_setup =
		_open_setup_btn == nullptr ? nullptr : _open_setup_btn->get_obj();
	set_hidden(cached_network, show_open_setup);
	set_hidden(new_connection, show_open_setup);
	set_hidden(open_setup, !show_open_setup);
	set_clickable(cached_network,
		      !show_open_setup && cached_network_available);
	set_clickable(new_connection,
		      !show_open_setup && setup_interaction_allowed);
	set_clickable(open_setup, show_open_setup && setup_interaction_allowed);
	set_back_navigation_available(!controls_blocked);
}

bool WifiPage::is_setup_interaction_allowed() const
{
	return !_busy && !has_pending_operation() &&
	       _visual_state != WifiPageVisualState::Connecting;
}

bool WifiPage::wifi_paired_actions_visible() const
{
	// The action row shows the paired On/Off + New controls whenever the
	// device has a saved configuration or a confirmed Station connection.
	// Connect Wi-Fi is the fallback only when neither is true, so a connected
	// device whose saved-network probe is still Unknown is not sent back to the
	// unconfigured Connect Wi-Fi entry.
	return _wifi_saved_network_config || _wifi_station_connected;
}

void WifiPage::open_wifi_config()
{
	if (!is_setup_interaction_allowed()) {
		return;
	}
	if (on_wifi_config_click) {
		on_wifi_config_click();
	}
}

void WifiPage::show_qr_popup()
{
	static constexpr int32_t CARD_W = WIDTH;
	static constexpr int32_t CARD_H = HEIGHT;
	static constexpr int32_t CARD_X = 0;
	static constexpr int32_t CARD_Y = 0;

	static constexpr int32_t QR_MARGIN_BOTTOM = 15;
	static constexpr int32_t QR_BG_W = 237;
	static constexpr int32_t QR_BG_H = 178;

	if (_container == nullptr) {
		return;
	}

	if (_qr_popup != nullptr) {
		refresh_qr_code();
		return;
	}

	// Overlay mask
	_qr_popup = lv_obj_create(_container);
	lv_obj_set_size(_qr_popup, WIDTH, HEIGHT);
	lv_obj_set_pos(_qr_popup, 0, 0);
	lv_obj_set_style_bg_color(_qr_popup, lv_color_black(), 0);
	lv_obj_set_style_bg_opa(_qr_popup, LV_OPA_70, 0);
	lv_obj_set_style_border_width(_qr_popup, 0, 0);
	lv_obj_set_style_radius(_qr_popup, 40, 0);
	lv_obj_set_style_pad_all(_qr_popup, 0, 0);
	lv_obj_set_style_margin_all(_qr_popup, 0, 0);
	lv_obj_set_style_shadow_width(_qr_popup, 0, 0);
	lv_obj_set_style_outline_width(_qr_popup, 0, 0);
	lv_obj_set_scrollbar_mode(_qr_popup, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_qr_popup, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_qr_popup, LV_OBJ_FLAG_CLICKABLE);

	// Popup card
	lv_obj_t *card = lv_obj_create(_qr_popup);
	lv_obj_set_size(card, CARD_W, CARD_H);
	lv_obj_set_pos(card, CARD_X, CARD_Y);
	lv_obj_set_style_bg_color(card, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(card, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(card, 39, 0);
	lv_obj_set_style_border_width(card, 2, 0);
	lv_obj_set_style_border_color(card, lv_color_hex(0x1C1917), 0);
	lv_obj_set_style_pad_all(card, 0, 0);
	lv_obj_set_style_margin_all(card, 0, 0);
	lv_obj_set_style_shadow_width(card, 0, 0);
	lv_obj_set_style_outline_width(card, 0, 0);
	lv_obj_set_scrollbar_mode(card, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(card, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(card, LV_OBJ_FLAG_CLICKABLE);

	// Title
	_qr_title_label = lv_label_create(card);
	lv_label_set_text(_qr_title_label, _qr_title.c_str());
	lv_obj_set_style_text_color(_qr_title_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_qr_title_label, &lv_font_montserrat_20, 0);
	lv_obj_align(_qr_title_label, LV_ALIGN_TOP_MID, 0, 8);

	// White QR background
	lv_obj_t *qr_bg = lv_obj_create(card);
	lv_obj_set_size(qr_bg, QR_BG_W, QR_BG_H);
	lv_obj_align(qr_bg, LV_ALIGN_BOTTOM_MID, 0, -QR_MARGIN_BOTTOM);
	lv_obj_set_style_bg_color(qr_bg, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(qr_bg, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(qr_bg, 30, 0);
	lv_obj_set_style_border_width(qr_bg, 0, 0);
	lv_obj_set_style_pad_all(qr_bg, 0, 0);
	lv_obj_set_style_margin_all(qr_bg, 0, 0);
	lv_obj_set_style_shadow_width(qr_bg, 0, 0);
	lv_obj_set_style_outline_width(qr_bg, 0, 0);
	lv_obj_set_scrollbar_mode(qr_bg, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(qr_bg, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(qr_bg, LV_OBJ_FLAG_CLICKABLE);

	// QR widget area: 148x148, centered in white box
	_qr_code = lv_qrcode_create(qr_bg);
	lv_qrcode_set_size(_qr_code, 148);
	lv_qrcode_set_dark_color(_qr_code, lv_color_hex(0x000000));
	lv_qrcode_set_light_color(_qr_code, lv_color_hex(0xFFFFFF));
	lv_obj_center(_qr_code);
	lv_obj_add_flag(_qr_code, LV_OBJ_FLAG_CLICKABLE);
	refresh_qr_code();

	// Close on click anywhere in popup
	auto click_cb = [](lv_event_t *e) {
		auto *self = static_cast<WifiPage *>(lv_event_get_user_data(e));
		if (self != nullptr) {
			self->hide_qr_popup(true);
		}
	};
	lv_obj_add_event_cb(_qr_popup, click_cb, LV_EVENT_CLICKED, this);
	lv_obj_add_event_cb(card, click_cb, LV_EVENT_CLICKED, this);
	lv_obj_add_event_cb(qr_bg, click_cb, LV_EVENT_CLICKED, this);
	lv_obj_add_event_cb(_qr_code, click_cb, LV_EVENT_CLICKED, this);
}

void WifiPage::hide_qr_popup(bool notify)
{
	const bool had_popup = _qr_popup != nullptr;
	_qr_title_label = nullptr;
	_qr_code = nullptr;
	lv_delete_if_valid(_qr_popup);
	if (notify && had_popup && on_wifi_qr_dismissed) {
		on_wifi_qr_dismissed();
	}
}

void WifiPage::refresh_qr_code()
{
	if (_qr_title_label != nullptr) {
		lv_label_set_text(_qr_title_label, _qr_title.c_str());
	}

	if (_qr_code == nullptr) {
		return;
	}

	const char *data = _qr_payload.empty() ? EMPTY_QR_PLACEHOLDER :
						 _qr_payload.c_str();
	lv_qrcode_update(_qr_code, data, strlen(data));
}

void WifiPage::on_subpage_destroy()
{
	_wifi_toggle_pending = false;
	_wifi_toggle_waiting_for_off_stable = false;
	_busy = false;
	clear_off_stable_confirmation();
	on_wifi_enabled_change = nullptr;
	on_wifi_qr_dismissed = nullptr;
	on_wifi_page_exit = nullptr;
	_information_group = nullptr;
	_ssid_info_row = nullptr;
	_ip_label = nullptr;
	_cached_network_btn = nullptr;
	_new_connection_btn = nullptr;
	_open_setup_btn = nullptr;
	_action_row = nullptr;
	_qr_popup = nullptr;
	_qr_title_label = nullptr;
	_qr_code = nullptr;
}

} // namespace ui
