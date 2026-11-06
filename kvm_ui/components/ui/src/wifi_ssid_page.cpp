#include "ui/wifi_ssid_page.hpp"

#include <algorithm>
#include <memory>
#include <new>
#include <string>
#include <utility>

#include <spdlog/spdlog.h>

#include "ui/bconb.hpp"
#include "ui/ssid_raster_marquee.hpp"
#include "ui/ssid_text_image_cache.hpp"
#include "ui/wifi_latency_probe.hpp"
#include "ui/wifi_ssid_display_policy.hpp"
#include "ui/wifi_ssid_font.hpp"

extern "C" {
	LV_IMAGE_DECLARE(img_settings_main_return);
}

namespace ui
{

namespace
{

constexpr const char *MANUAL_SSID_LABEL = "Manual SSID";
constexpr uint16_t CONTENT_WIDTH = 230;
constexpr uint16_t NAV_HEIGHT = 64;
constexpr uint16_t LOADING_BOX_HEIGHT = 112;
constexpr uint16_t LOADING_ARC_SIZE = 72;
constexpr uint16_t LOADING_ARC_WIDTH = 8;
constexpr uint32_t LOADING_PERIOD_MS = 40;
constexpr uint32_t LOADING_TRACK_COLOR = 0x44403B;
// Scan half grows 0 -> 50% at 75% of the historical 70%/4000 ms climb rate.
// rate = (3/4) * 70 / 4000 = 210 / 16000 = 21 / 1600 (% per ms);
// 50% is reached at ~3810 ms (80000 / 21).
constexpr uint32_t LOADING_PRIOR_CLIMB_TARGET = 70;
constexpr uint32_t LOADING_PRIOR_CLIMB_DURATION_MS = 4000;
constexpr uint32_t LOADING_SCAN_RATE_FACTOR_NUM = 3;
constexpr uint32_t LOADING_SCAN_RATE_FACTOR_DEN = 4;
constexpr uint32_t LOADING_SCAN_TARGET_PERCENT = 50;
constexpr uint32_t LOADING_SCAN_RATE_NUM =
	LOADING_PRIOR_CLIMB_TARGET * LOADING_SCAN_RATE_FACTOR_NUM;
constexpr uint32_t LOADING_SCAN_RATE_DEN =
	LOADING_PRIOR_CLIMB_DURATION_MS * LOADING_SCAN_RATE_FACTOR_DEN;
constexpr uint32_t LOADING_FAST_DURATION_MS = 250;
constexpr uint16_t ROLLER_HEIGHT = 112;
constexpr int16_t NAV_ICON_OFFSET = 18;
constexpr uint8_t ROLLER_CENTER_MARGIN_LEFT = 7;
constexpr std::size_t BYTES_PER_KILOBYTE = 1024;
constexpr int MINIMUM_WIFI_SIGNAL_DBM = -75;

} // namespace

WifiSsidPage::WifiSsidPage(const lv_font_t *ssid_font,
					   uint32_t ssid_image_cache_kb,
					   bool enable_ssid_scroll,
					   SsidFontRenderMode ssid_font_render_mode,
					   WifiLoadingMode wifi_loading_mode)
	: SubPageBase(PageType::SUBPAGE_WIFI_SSID, "SSID")
	, _ssid_font(resolve_wifi_ssid_font(ssid_font))
	, _visible_ssid_font(ssid_font_render_mode ==
				     SsidFontRenderMode::RuntimeTinyTtf ?
				     compiled_wifi_ssid_font() : _ssid_font)
	, _ssid_image_cache_kb(ssid_image_cache_kb)
	, _ssid_font_render_mode(ssid_font_render_mode)
	, _enable_ssid_scroll(enable_ssid_scroll)
	, _wifi_loading_mode(wifi_loading_mode)
{
}

WifiSsidPage::~WifiSsidPage() = default;

void WifiSsidPage::destroy()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	release_selected_raster_marquee();
	stop_loading_animation();
	release_text_image_cache();
	_cache_page_active = false;
	_cache_content_ready = false;
	_cache_activation_failed = false;
	reset_loading_session();
	on_ssid_selected = nullptr;
	on_manual_ssid_click = nullptr;
	on_back_click = nullptr;
	_controls.clear();
	_options.clear();
	_roller_options_text.clear();
	SubPageBase::destroy();
}

void WifiSsidPage::on_enter()
{
	++_selected_marquee_generation;
	_cache_page_active = true;
	_cache_activation_failed = false;
	update_view();
}

void WifiSsidPage::on_exit()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	release_selected_raster_marquee();
	stop_loading_animation();
	_cache_page_active = false;
	release_text_image_cache();
	_cache_activation_failed = false;
	reset_loading_session();
}

void WifiSsidPage::on_pause()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	stop_loading_animation();
	_cache_page_active = false;
	stop_cache_timer();
	stop_cache_scroll_hold();
	reset_loading_session();
	if (_text_image_cache != nullptr) {
		_text_image_cache->cancel_pending();
		_cache_content_ready = false;
	}
}

void WifiSsidPage::on_resume()
{
	_cache_page_active = true;
	update_view();
}

void WifiSsidPage::set_loading(std::string_view message)
{
	(void)message;
	_state = ViewState::Loading;
	_loading_rotation = 0;
	reset_loading_session();
	rebuild_options();
	update_view();
}

void WifiSsidPage::set_scan_results(const std::vector<WifiScanResult> &results)
{
	_results = results;
	_state = _results.empty() ? ViewState::Empty : ViewState::Results;
	rebuild_options();
	update_view();
}

void WifiSsidPage::set_empty(std::string_view message)
{
	(void)message;
	_state = ViewState::Empty;
	_results.clear();
	rebuild_options();
	update_view();
}

void WifiSsidPage::set_error(std::string_view message)
{
	(void)message;
	_state = ViewState::Error;
	_results.clear();
	rebuild_options();
	update_view();
}

void WifiSsidPage::set_show_eap_unsupported(bool show)
{
	if (_show_eap_unsupported == show) {
		return;
	}
	_show_eap_unsupported = show;
	rebuild_options();
	update_view();
}

void WifiSsidPage::build_content()
{
	WifiLatencyProbe::initialize();
	if (_back_btn != nullptr && _back_btn->get_obj() != nullptr) {
		lv_obj_add_flag(_back_btn->get_obj(), LV_OBJ_FLAG_HIDDEN);
		lv_obj_add_flag(_back_btn->get_obj(),
				LV_OBJ_FLAG_IGNORE_LAYOUT);
	}
	create_navigation();
	create_loading_indicator();
	create_roller();
	rebuild_options();
	update_view();
}

void WifiSsidPage::on_subpage_destroy()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	release_selected_raster_marquee();
	stop_loading_animation();
	release_text_image_cache();
	_cache_page_active = false;
	_cache_content_ready = false;
	_cache_activation_failed = false;
	reset_loading_session();
	_controls.clear();
	reset_lvgl_pointers();
}

void WifiSsidPage::create_navigation()
{
	if (_scroll_content == nullptr || _navigation_box != nullptr) {
		return;
	}

	_navigation_box = lv_obj_create(_scroll_content);
	lv_obj_set_size(_navigation_box, CONTENT_WIDTH, NAV_HEIGHT);
	lv_obj_set_style_bg_opa(_navigation_box, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_navigation_box, 0, 0);
	lv_obj_set_style_pad_all(_navigation_box, 0, 0);
	lv_obj_set_flex_flow(_navigation_box, LV_FLEX_FLOW_ROW);
	lv_obj_set_flex_align(_navigation_box, LV_FLEX_ALIGN_SPACE_BETWEEN,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
	lv_obj_clear_flag(_navigation_box, LV_OBJ_FLAG_SCROLLABLE);

	auto back = std::make_unique<BConB>(_navigation_box, ACTIVE_COLOR);
	back->create();
	back->set_visual_state(ControlState::VisualState::ACTIVE,
			       ControlState::ApplyPolicy::IMMEDIATE);
	if (auto *obj = back->get_obj(); obj != nullptr) {
		lv_obj_set_style_bg_opa(obj, LV_OPA_TRANSP, 0);
	}
	back->set_image(&img_settings_main_return);
	back->set_icon_offset(-NAV_ICON_OFFSET, 0);
	back->on_click([this](BConB *) {
		if (on_back_click) {
			on_back_click();
		}
	});
	_controls.push_back(std::move(back));

	auto next = std::make_unique<BConB>(_navigation_box, ACTIVE_COLOR);
	next->create();
	if (auto *obj = next->get_obj(); obj != nullptr) {
		lv_obj_set_style_bg_opa(obj, LV_OPA_TRANSP, 0);
	}
	next->set_image(&img_settings_main_return);
	next->set_image_rotation(1800);
	next->set_icon_offset(NAV_ICON_OFFSET, 0);
	next->on_click([this](BConB *) {
		activate_selected_option();
	});
	_controls.push_back(std::move(next));
}

void WifiSsidPage::create_loading_indicator()
{
	if (_scroll_content == nullptr || _loading_box != nullptr) {
		return;
	}

	_loading_box = lv_obj_create(_scroll_content);
	lv_obj_set_size(_loading_box, CONTENT_WIDTH, LOADING_BOX_HEIGHT);
	lv_obj_set_style_margin_left(_loading_box, ROLLER_CENTER_MARGIN_LEFT,
				     0);
	lv_obj_set_style_margin_top(_loading_box, 12, 0);
	lv_obj_set_style_bg_opa(_loading_box, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_loading_box, 0, 0);
	lv_obj_set_style_pad_all(_loading_box, 0, 0);
	lv_obj_clear_flag(_loading_box, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_loading_box, LV_OBJ_FLAG_CLICKABLE);

	_loading_arc = lv_arc_create(_loading_box);
	lv_obj_set_size(_loading_arc, LOADING_ARC_SIZE, LOADING_ARC_SIZE);
	lv_obj_center(_loading_arc);
	lv_obj_clear_flag(_loading_arc, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_loading_arc, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_remove_style(_loading_arc, nullptr, LV_PART_KNOB);
	lv_obj_set_style_arc_opa(_loading_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_loading_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_loading_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_loading_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_arc_color(_loading_arc, lv_color_hex(ACTIVE_COLOR),
				   LV_PART_INDICATOR);
	lv_obj_set_style_arc_width(_loading_arc, LOADING_ARC_WIDTH,
				   LV_PART_INDICATOR);
	lv_obj_set_style_arc_rounded(_loading_arc, true, LV_PART_INDICATOR);
	lv_arc_set_bg_angles(_loading_arc, 0, 360);
	lv_arc_set_angles(_loading_arc, 0, 270);
	lv_arc_set_rotation(_loading_arc, 0);
	lv_obj_add_flag(_loading_box, LV_OBJ_FLAG_HIDDEN);
	lv_obj_add_flag(_loading_box, LV_OBJ_FLAG_IGNORE_LAYOUT);
}

void WifiSsidPage::create_roller()
{
	if (_scroll_content == nullptr || _roller != nullptr) {
		return;
	}

	_roller = lv_roller_create(_scroll_content);
	lv_obj_set_size(_roller, CONTENT_WIDTH, ROLLER_HEIGHT);
	lv_obj_set_style_margin_left(_roller, ROLLER_CENTER_MARGIN_LEFT, 0);
	lv_obj_set_style_margin_top(_roller, 12, 0);
	lv_obj_set_style_bg_color(_roller, lv_color_hex(ROLLER_BG_COLOR),
				  LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_roller, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_radius(_roller, 32, LV_PART_MAIN);
	lv_obj_set_style_border_width(_roller, 0, LV_PART_MAIN);
	lv_obj_set_style_clip_corner(_roller, true, LV_PART_MAIN);
	lv_obj_set_style_text_color(_roller, lv_color_white(), LV_PART_MAIN);
	lv_obj_set_style_text_align(_roller, LV_TEXT_ALIGN_CENTER,
				    LV_PART_MAIN);
	lv_obj_set_style_bg_color(_roller, lv_color_hex(ROLLER_SELECTED_COLOR),
				  LV_PART_SELECTED);
	lv_obj_set_style_bg_opa(_roller, LV_OPA_COVER, LV_PART_SELECTED);
	lv_obj_set_style_radius(_roller, LV_RADIUS_CIRCLE, LV_PART_SELECTED);
	lv_obj_set_style_text_color(_roller, lv_color_white(),
				    LV_PART_SELECTED);
	apply_roller_text_style();
	lv_obj_add_event_cb(_roller, roller_event_cb, LV_EVENT_VALUE_CHANGED,
					this);
	lv_obj_add_event_cb(_roller, roller_pressed_cb, LV_EVENT_PRESSED, this);
	lv_obj_add_event_cb(_roller, roller_pressing_cb, LV_EVENT_PRESSING, this);
	lv_obj_add_event_cb(_roller, roller_released_cb, LV_EVENT_RELEASED, this);
	lv_obj_add_event_cb(_roller, roller_released_cb, LV_EVENT_PRESS_LOST,
						this);
	lv_obj_add_event_cb(_roller, roller_draw_post_cb, LV_EVENT_DRAW_POST, this);
	if (_enable_ssid_scroll) {
		create_selected_marquee();
	}
}

void WifiSsidPage::create_selected_marquee()
{
	if (!_enable_ssid_scroll || _roller == nullptr ||
	    _selected_marquee_overlay != nullptr) {
		return;
	}

	_selected_marquee_overlay = lv_obj_create(_roller);
	if (_selected_marquee_overlay == nullptr) {
		spdlog::warn("SSID selected marquee overlay creation failed");
		return;
	}

	lv_obj_set_size(_selected_marquee_overlay, CONTENT_WIDTH,
				TEXT_IMAGE_ROW_HEIGHT);
	lv_obj_set_style_bg_color(_selected_marquee_overlay,
				  lv_color_hex(ROLLER_SELECTED_COLOR), LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_selected_marquee_overlay, LV_OPA_COVER,
				LV_PART_MAIN);
	lv_obj_set_style_radius(_selected_marquee_overlay, LV_RADIUS_CIRCLE,
				LV_PART_MAIN);
	lv_obj_set_style_border_width(_selected_marquee_overlay, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_selected_marquee_overlay, 0, LV_PART_MAIN);
	lv_obj_set_style_clip_corner(_selected_marquee_overlay, true, LV_PART_MAIN);
	lv_obj_clear_flag(_selected_marquee_overlay, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_selected_marquee_overlay, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_selected_marquee_overlay, LV_OBJ_FLAG_IGNORE_LAYOUT);
	lv_obj_add_flag(_selected_marquee_overlay, LV_OBJ_FLAG_FLOATING);
	lv_obj_add_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN);

	_selected_marquee_label = lv_label_create(_selected_marquee_overlay);
	if (_selected_marquee_label == nullptr) {
		spdlog::warn("SSID selected marquee label creation failed");
		lv_obj_delete(_selected_marquee_overlay);
		_selected_marquee_overlay = nullptr;
		return;
	}

	lv_obj_set_style_text_color(_selected_marquee_label, lv_color_white(),
					    LV_PART_MAIN);
	lv_obj_set_style_text_font(_selected_marquee_label,
				   _visible_ssid_font, LV_PART_MAIN);
	lv_obj_set_style_text_align(_selected_marquee_label, LV_TEXT_ALIGN_CENTER,
				    LV_PART_MAIN);
	lv_obj_set_style_pad_all(_selected_marquee_label, 0, LV_PART_MAIN);
	lv_obj_clear_flag(_selected_marquee_label, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_selected_marquee_label, LV_OBJ_FLAG_SCROLLABLE);
	lv_label_set_long_mode(_selected_marquee_label,
			       LV_LABEL_LONG_SCROLL_CIRCULAR);
	const int32_t font_height = lv_font_get_line_height(_visible_ssid_font);
	lv_obj_set_size(_selected_marquee_label, CONTENT_WIDTH,
				font_height);
	lv_obj_set_pos(_selected_marquee_label, 0,
			       (static_cast<int32_t>(TEXT_IMAGE_ROW_HEIGHT) -
				font_height) / 2);
}

bool WifiSsidPage::ensure_selected_raster_marquee()
{
	if (!text_image_cache_enabled() || _cache_activation_failed ||
	    !_cache_content_ready || !runtime_tiny_ttf_mode()) {
		return false;
	}
	if (_selected_raster_marquee != nullptr) {
		return _selected_raster_marquee->get_obj() != nullptr;
	}
	if (_selected_marquee_overlay == nullptr || _ssid_font == nullptr ||
	    _ssid_image_cache_kb == 0) {
		return false;
	}
	try {
		_selected_raster_marquee = std::make_unique<SsidRasterMarquee>(
			_selected_marquee_overlay, CONTENT_WIDTH,
			TEXT_IMAGE_ROW_HEIGHT, _ssid_font,
			static_cast<std::size_t>(_ssid_image_cache_kb) *
				BYTES_PER_KILOBYTE);
	} catch (const std::bad_alloc &) {
		return false;
	}
	lv_obj_t *viewport = _selected_raster_marquee->get_obj();
	if (viewport == nullptr) {
		release_selected_raster_marquee();
		return false;
	}
	lv_obj_set_pos(viewport, 0, 0);
	return true;
}

void WifiSsidPage::release_selected_raster_marquee()
{
	_selected_marquee_pending_generation = 0;
	_selected_marquee_pending_index = 0;
	_selected_marquee_pending_text.clear();
	_selected_raster_marquee.reset();
}

bool WifiSsidPage::ensure_text_image_cache()
{
	if (text_image_cache_enabled()) {
		return true;
	}
	if (_container == nullptr || _roller == nullptr ||
	    _ssid_image_cache_kb == 0 || !_cache_page_active ||
	    _state == ViewState::Loading || _cache_activation_failed) {
		return false;
	}
	if (lv_font_get_line_height(&lv_font_montserrat_20) >
		    TEXT_IMAGE_ROW_HEIGHT ||
	    roller_label() == nullptr) {
		spdlog::warn("SSID image cache requires its internal roller label");
		_cache_activation_failed = true;
		return false;
	}

	const std::size_t max_bytes =
		static_cast<std::size_t>(_ssid_image_cache_kb) * BYTES_PER_KILOBYTE;
	const uint32_t allocation_started [[maybe_unused]] = lv_tick_get();
	try {
		_text_image_arena = std::make_unique<SsidTextImageArena>(max_bytes);
		_text_image_cache = std::make_unique<SsidTextImageCache>(
			_container, _ssid_font, CONTENT_WIDTH,
			TEXT_IMAGE_ROW_HEIGHT, max_bytes, _text_image_arena.get());
	} catch (const std::bad_alloc &) {
		spdlog::warn("SSID image cache arena allocation failed");
		release_text_image_cache();
		_cache_activation_failed = true;
		return false;
	}
	if (!_text_image_cache->is_usable()) {
		spdlog::warn("SSID image cache arena is unavailable");
		release_text_image_cache();
		_cache_activation_failed = true;
		return false;
	}
	SPDLOG_DEBUG("[wifi] SSID image cache activation: budget_bytes={}, "
		      "elapsed_ms={}, capacity={}",
		      max_bytes, lv_tick_elaps(allocation_started),
		      _text_image_arena == nullptr ? 0U :
						      _text_image_arena->capacity());
	_cache_content_ready = false;
	apply_roller_text_style();
	return true;
}

void WifiSsidPage::release_text_image_cache()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	stop_cache_timer();
	stop_cache_scroll_hold();
	reset_loading_session();
	if (_text_image_cache != nullptr) {
		_text_image_cache->cancel_pending();
	}
	_text_image_cache.reset();
	_text_image_arena.reset();
	reset_roller_label_width();
	apply_roller_text_style();
}

void WifiSsidPage::rebuild_options()
{
	cancel_selected_marquee_request();
	hide_selected_marquee();
	stop_cache_timer();
	stop_cache_scroll_hold();
	reset_cache_scheduler();
	if (_text_image_cache != nullptr) {
		_text_image_cache->cancel_pending();
	}
	if (text_image_cache_enabled()) {
		_cache_content_ready = false;
	}
	_options.clear();
	_options.push_back(RollerOption { .label = MANUAL_SSID_LABEL,
					  .ssid = {},
					  .type = OptionType::Manual });

	switch (_state) {
	case ViewState::Loading:
		break;
		case ViewState::Results:
				for (const auto &result : _results) {
					if (result.ssid.empty()) {
						continue;
					}
					if (result.signal <= MINIMUM_WIFI_SIGNAL_DBM) {
						continue;
					}
					if (result.eap && !_show_eap_unsupported) {
					continue;
				}
				const auto display = make_wifi_ssid_display(result.ssid);
				if (!display.visible) {
					continue;
				}

				auto label = display.text;
			if (result.eap) {
				label += " (EAP)";
			}
			_options.push_back(RollerOption {
				.label = std::move(label),
				.ssid = result.ssid,
				.type = result.eap ? OptionType::Unsupported :
						     OptionType::Network });
		}
		break;
	case ViewState::Empty:
	case ViewState::Error:
		break;
	}

	if (_selected_index >= _options.size()) {
		_selected_index = 0;
	}
	update_roller_options();
}

void WifiSsidPage::update_view()
{
	const bool scanning = _state == ViewState::Loading;
	if (!scanning && _cache_page_active && !_cache_activation_failed &&
	    !text_image_cache_enabled()) {
		ensure_text_image_cache();
	}
	if (!scanning && text_image_cache_enabled() && _cache_page_active &&
	    !_cache_content_ready) {
		queue_text_image_window();
	}

	const bool cache_waiting = !scanning && _cache_page_active &&
		text_image_cache_enabled() && !_cache_content_ready;
	const bool rendering_loading = _wifi_loading_mode ==
		WifiLoadingMode::Progress &&
		_loading_phase == LoadingPhase::Rendering;
	const bool finishing_loading = _wifi_loading_mode ==
		WifiLoadingMode::Progress &&
		_loading_phase == LoadingPhase::Finishing;
	const bool show_loading_box = scanning || cache_waiting ||
		rendering_loading || finishing_loading;
	const bool show_roller = !scanning && !cache_waiting &&
		!rendering_loading && !finishing_loading;
	if (_roller != nullptr) {
		if (show_roller) {
			lv_obj_clear_flag(_roller, LV_OBJ_FLAG_HIDDEN);
			lv_obj_clear_flag(_roller, LV_OBJ_FLAG_IGNORE_LAYOUT);
		} else {
			lv_obj_add_flag(_roller, LV_OBJ_FLAG_HIDDEN);
			lv_obj_add_flag(_roller, LV_OBJ_FLAG_IGNORE_LAYOUT);
		}
	}
	if (!show_roller) {
		hide_selected_marquee();
	}

	if (_loading_box != nullptr) {
		if (show_loading_box) {
			lv_obj_clear_flag(_loading_box, LV_OBJ_FLAG_HIDDEN);
			lv_obj_clear_flag(_loading_box,
					  LV_OBJ_FLAG_IGNORE_LAYOUT);
		} else {
			lv_obj_add_flag(_loading_box, LV_OBJ_FLAG_HIDDEN);
			lv_obj_add_flag(_loading_box,
					LV_OBJ_FLAG_IGNORE_LAYOUT);
		}
	}

	if (loading_presentation_active()) {
		start_loading_animation();
		if (scanning) {
			stop_cache_timer();
			if (_text_image_cache != nullptr) {
				_text_image_cache->cancel_pending();
			}
		}
		return;
	}

	stop_loading_animation();
	if (_cache_page_active) {
		update_roller_options();
	}
	update_selected_marquee();
}

bool WifiSsidPage::loading_presentation_active() const noexcept
{
	return _cache_page_active &&
	       (_state == ViewState::Loading ||
		(text_image_cache_enabled() && !_cache_content_ready) ||
		(_wifi_loading_mode == WifiLoadingMode::Progress &&
		 (_loading_phase == LoadingPhase::Rendering ||
		  _loading_phase == LoadingPhase::Finishing)));
}

bool WifiSsidPage::initial_cache_preload_active() const noexcept
{
	return _cache_page_active && _state != ViewState::Loading &&
	       text_image_cache_enabled() && !_cache_content_ready &&
	       _cache_preload_phase != CachePreloadPhase::Inactive;
}

void WifiSsidPage::update_roller_options()
{
	if (_roller == nullptr || _options.empty()) {
		return;
	}

	apply_roller_text_style();
	const bool cache_enabled = text_image_cache_enabled();
	if (!cache_enabled) {
		reset_roller_label_width();
	}
	_roller_options_text.clear();
	for (std::size_t i = 0; i < _options.size(); ++i) {
		if (i > 0U) {
			_roller_options_text += '\n';
		}
		if (cache_enabled) {
			_roller_options_text += ' ';
		} else {
			_roller_options_text += _options[i].label;
		}
	}
	lv_roller_set_options(_roller, _roller_options_text.c_str(),
				      LV_ROLLER_MODE_NORMAL);
	if (cache_enabled) {
		set_cache_roller_label_width();
	}
	lv_roller_set_selected(_roller, _selected_index, LV_ANIM_OFF);
}

void WifiSsidPage::activate_selected_option()
{
	if (_state == ViewState::Loading) {
		return;
	}

	if (_roller != nullptr) {
		_selected_index = lv_roller_get_selected(_roller);
	}

	const RollerOption *option = current_option();
	if (option == nullptr) {
		return;
	}

	if (option->type == OptionType::Manual) {
		if (on_manual_ssid_click) {
			on_manual_ssid_click();
		}
		return;
	}

	if (option->type == OptionType::Unsupported) {
		return;
	}

	if (option->ssid.empty()) {
		return;
	}
	if (on_ssid_selected) {
		const std::string ssid = option->ssid;
		on_ssid_selected(ssid);
	}
}

void WifiSsidPage::start_loading_animation()
{
	if (_loading_arc == nullptr) {
		return;
	}

	lv_obj_clear_flag(_loading_arc, LV_OBJ_FLAG_HIDDEN);
	lv_arc_set_rotation(_loading_arc, _loading_rotation);
	apply_loading_geometry_for_mode();
	if (initial_cache_preload_active()) {
		// The preload timer alternates a presentation step and one raster step.
		// Keeping a second independent loading timer would let both callbacks
		// run in one lv_timer_handler pass and defeat the frame boundary.
		if (_loading_timer != nullptr) {
			lv_timer_delete(_loading_timer);
			_loading_timer = nullptr;
		}
		return;
	}
	if (!_loading_presentation_clock_started) {
		_loading_presentation_clock_started = true;
		_last_loading_presentation_tick = lv_tick_get();
	}
	if (_loading_timer == nullptr) {
		_loading_timer = lv_timer_create(loading_timer_cb,
						 LOADING_PERIOD_MS, this);
	}
}

void WifiSsidPage::stop_loading_animation()
{
	if (_loading_timer != nullptr) {
		lv_timer_delete(_loading_timer);
		_loading_timer = nullptr;
	}
}

void WifiSsidPage::advance_loading_presentation()
{
	if (_loading_arc == nullptr || !loading_presentation_active()) {
		return;
	}

	uint32_t gap = LOADING_PERIOD_MS;
	if (_loading_presentation_clock_started) {
		gap = lv_tick_elaps(_last_loading_presentation_tick);
		_max_loading_presentation_gap_ms =
			std::max(_max_loading_presentation_gap_ms, gap);
	} else {
		_loading_presentation_clock_started = true;
	}
	const uint32_t now = lv_tick_get();
	_last_loading_presentation_tick = now;
	const uint32_t rotation_delta = std::max(
		1U, (gap * 18U + LOADING_PERIOD_MS - 1U) / LOADING_PERIOD_MS);
	_loading_rotation =
		static_cast<uint16_t>((_loading_rotation + rotation_delta) % 360);
	lv_arc_set_rotation(_loading_arc, _loading_rotation);
	lv_obj_invalidate(_loading_arc);
}

void WifiSsidPage::advance_loading_for_mode()
{
	switch (_wifi_loading_mode) {
	case WifiLoadingMode::None:
		advance_loading_presentation();
		return;
	case WifiLoadingMode::Progress:
		advance_loading_progress();
		return;
	}
}

void WifiSsidPage::advance_loading_progress()
{
	if (_loading_arc == nullptr || !loading_presentation_active()) {
		return;
	}
	switch (_loading_phase) {
	case LoadingPhase::Climbing: {
		// Scan half: 0 -> 50% at 75% of the historical 70%/4000 ms rate
		// (21/1600 % per ms). Floor (not ceiling) so the 50% boundary
		// lands at exactly ~3810 ms instead of ~3734 ms.
		const uint32_t elapsed = lv_tick_elaps(_loading_start_tick);
		_loading_progress = static_cast<uint8_t>(
			(elapsed * LOADING_SCAN_RATE_NUM) / LOADING_SCAN_RATE_DEN);
		if (_loading_progress >= LOADING_SCAN_TARGET_PERCENT) {
			_loading_progress = LOADING_SCAN_TARGET_PERCENT;
			_loading_phase = LoadingPhase::Waiting;
		}
		break;
	}
	case LoadingPhase::Waiting:
		_loading_progress = LOADING_SCAN_TARGET_PERCENT;
		break;
	case LoadingPhase::Rendering:
		// Cache-render half: progress tracks drawn/total via the cache.
		recompute_rendering_progress();
		return;
	case LoadingPhase::Finishing: {
		// Warm cache close: linear from the finishing base to 100 over 250 ms.
		const uint32_t elapsed = lv_tick_elaps(_loading_ready_tick);
		const uint8_t span =
			static_cast<uint8_t>(100U - _loading_finish_base);
		_loading_progress = static_cast<uint8_t>(
			_loading_finish_base +
			(elapsed * span + LOADING_FAST_DURATION_MS - 1U) /
			LOADING_FAST_DURATION_MS);
		if (_loading_progress >= 100U) {
			_loading_progress = 100U;
			_loading_phase = LoadingPhase::Done;
			lv_arc_set_angles(_loading_arc, 0, 360);
			lv_obj_invalidate(_loading_arc);
			update_view();
			return;
		}
		break;
	}
	case LoadingPhase::Idle:
		// An active timer should never observe an uninitialized session. Keep
		// this branch defensive for direct/test callbacks; the next presentation
		// entry initializes the session through start_loading_animation().
		return;
	case LoadingPhase::Done:
		return;
	}
	lv_arc_set_angles(_loading_arc, 0,
		static_cast<int16_t>(_loading_progress * 360U / 100U));
	lv_obj_invalidate(_loading_arc);
}

void WifiSsidPage::recompute_rendering_progress() noexcept
{
	if (_loading_phase != LoadingPhase::Rendering ||
	    _cache_render_total == 0 || _text_image_cache == nullptr) {
		return;
	}
	const std::size_t total = _cache_render_total;
	const std::size_t pending = _text_image_cache->pending_count();
	const std::size_t drawn = pending <= total ? total - pending : 0;
	const uint8_t scaled =
		static_cast<uint8_t>((50U * drawn) / total);
	_loading_progress =
		static_cast<uint8_t>(LOADING_SCAN_TARGET_PERCENT + scaled);
	if (_loading_arc != nullptr) {
		lv_arc_set_angles(_loading_arc, 0,
			static_cast<int16_t>(_loading_progress * 360U / 100U));
		lv_obj_invalidate(_loading_arc);
	}
}

void WifiSsidPage::initialize_loading_progress()
{
	if (_loading_phase != LoadingPhase::Idle) {
		return;
	}
	_loading_progress = 0;
	_loading_phase = LoadingPhase::Climbing;
	_loading_start_tick = lv_tick_get();
}

void WifiSsidPage::apply_loading_geometry_for_mode()
{
	if (_loading_arc == nullptr) {
		return;
	}
	switch (_wifi_loading_mode) {
	case WifiLoadingMode::None:
		return;
	case WifiLoadingMode::Progress:
		initialize_loading_progress();
		lv_obj_set_style_arc_color(_loading_arc,
			lv_color_hex(LOADING_TRACK_COLOR), LV_PART_MAIN);
		lv_obj_set_style_arc_opa(_loading_arc, LV_OPA_COVER,
			LV_PART_MAIN);
		lv_obj_set_style_arc_width(_loading_arc, LOADING_ARC_WIDTH,
			LV_PART_MAIN);
		lv_arc_set_angles(_loading_arc, 0,
			static_cast<int16_t>(_loading_progress * 360U / 100U));
		lv_arc_set_rotation(_loading_arc, 0);
		return;
	}
}

void WifiSsidPage::reset_cache_scheduler() noexcept
{
	_cache_preload_phase = CachePreloadPhase::Inactive;
	_cache_timing_active = false;
	_cache_result_started_tick = 0;
	_last_loading_presentation_tick = 0;
	_max_loading_presentation_gap_ms = 0;
	_loading_presentation_clock_started = false;
}

void WifiSsidPage::reset_loading_progress() noexcept
{
	_loading_phase = LoadingPhase::Idle;
	_loading_progress = 0;
	_loading_start_tick = 0;
	_loading_ready_tick = 0;
	_cache_render_total = 0;
	_loading_finish_base = 0;
}

void WifiSsidPage::reset_loading_session() noexcept
{
	reset_cache_scheduler();
	reset_loading_progress();
}

void WifiSsidPage::queue_text_image_window()
{
	if (!text_image_cache_enabled() || !_cache_page_active ||
	    _state == ViewState::Loading) {
		return;
	}

	if (!_cache_content_ready && !_cache_timing_active) {
		_cache_timing_active = true;
		_cache_result_started_tick = lv_tick_get();
		_cache_preload_phase = CachePreloadPhase::Present;
		_loading_presentation_clock_started = true;
		_last_loading_presentation_tick = _cache_result_started_tick;
	}
	_text_image_cache->request_labels(foreground_text_labels(),
						 speculative_text_labels());
	if (!_text_image_cache->has_pending()) {
		mark_cache_content_ready();
	}
	if (_wifi_loading_mode == WifiLoadingMode::Progress &&
	    !_cache_content_ready &&
	    (_loading_phase == LoadingPhase::Idle ||
	     _loading_phase == LoadingPhase::Climbing ||
	     _loading_phase == LoadingPhase::Waiting) &&
	    _text_image_cache->has_pending()) {
		// Results arrived: enter the cache-render half and snapshot the total
		// once. Idle covers a page opened straight into cached results with no
		// preceding scan; repeated queue calls must not overwrite the snapshot.
		_cache_render_total =
			static_cast<uint32_t>(_text_image_cache->pending_count());
		_loading_progress = std::max(_loading_progress,
			static_cast<uint8_t>(LOADING_SCAN_TARGET_PERCENT));
		_loading_phase = LoadingPhase::Rendering;
	}
	if (cache_scroll_hold_active()) {
		stop_cache_timer();
	} else if (cache_timer_work_pending()) {
		start_cache_timer();
	} else {
		stop_cache_timer();
	}
	lv_obj_invalidate(_roller);
}

void WifiSsidPage::render_next_text_image()
{
	if (!text_image_cache_enabled() || cache_scroll_hold_active() ||
	    _text_image_cache == nullptr) {
		return;
	}

	if (!_cache_content_ready &&
	    _cache_preload_phase == CachePreloadPhase::Present) {
		advance_loading_for_mode();
		_cache_preload_phase = CachePreloadPhase::Rasterize;
		return;
	}

	const auto result = _text_image_cache->render_next();
	if (result == SsidTextImageCache::RenderResult::Failed) {
		disable_text_image_cache();
		return;
	}

	lv_obj_invalidate(_roller);
	if (_wifi_loading_mode == WifiLoadingMode::Progress &&
	    _loading_phase == LoadingPhase::Rendering) {
		// drawn increased: refresh 50 + 50*drawn/total immediately.
		recompute_rendering_progress();
	}
	if (!_text_image_cache->has_pending()) {
		stop_cache_timer();
		mark_cache_content_ready();
	} else if (!_cache_content_ready) {
		_cache_preload_phase = CachePreloadPhase::Present;
	}
}

void WifiSsidPage::draw_text_image_rows(lv_event_t *event)
{
	if (!text_image_cache_enabled() || event == nullptr) {
		return;
	}

	lv_layer_t *layer = lv_event_get_layer(event);
	if (layer == nullptr) {
		return;
	}

	lv_area_t roller_area {};
	lv_obj_get_coords(_roller, &roller_area);
	const lv_area_t original_clip_area = layer->_clip_area;
	const lv_area_t roller_clip_area {
		.x1 = std::max(roller_area.x1, original_clip_area.x1),
		.y1 = std::max(roller_area.y1, original_clip_area.y1),
		.x2 = std::min(roller_area.x2, original_clip_area.x2),
		.y2 = std::min(roller_area.y2, original_clip_area.y2),
	};
	if (roller_clip_area.x1 > roller_clip_area.x2 ||
	    roller_clip_area.y1 > roller_clip_area.y2) {
		return;
	}
	layer->_clip_area = roller_clip_area;
	for (uint32_t option_index = 0; option_index < _options.size();
	     ++option_index) {
		if (selected_marquee_visible() && option_index == _selected_index) {
			continue;
		}

		lv_area_t destination {};
		if (!cached_text_row_area(option_index, &destination) ||
		    destination.y2 < roller_area.y1 ||
		    destination.y1 > roller_area.y2) {
			continue;
		}
		const lv_draw_buf_t *source =
			_text_image_cache->source_for(_options[option_index].label);
		if (source == nullptr) {
			continue;
		}

		lv_draw_image_dsc_t draw_dsc;
		lv_draw_image_dsc_init(&draw_dsc);
		draw_dsc.src = source;
		lv_draw_image(layer, &draw_dsc, &destination);
	}
	layer->_clip_area = original_clip_area;
}

bool WifiSsidPage::cached_text_row_area(uint32_t option_index,
						 lv_area_t *area) const
{
	if (area == nullptr || option_index >= _options.size() ||
	    _roller == nullptr || !lv_obj_is_valid(_roller)) {
		return false;
	}

	lv_obj_t *label = roller_label();
	if (label == nullptr) {
		return false;
	}

	lv_area_t roller_area {};
	lv_area_t label_area {};
	lv_obj_get_coords(_roller, &roller_area);
	lv_obj_get_coords(label, &label_area);
	const int32_t layout_font_height =
		lv_font_get_line_height(&lv_font_montserrat_20);
	if (layout_font_height > TEXT_IMAGE_ROW_HEIGHT) {
		return false;
	}
	const int32_t row_top_offset =
		(static_cast<int32_t>(TEXT_IMAGE_ROW_HEIGHT) - layout_font_height) / 2;
	area->x1 = roller_area.x1;
	area->y1 = label_area.y1 +
		   static_cast<int32_t>(option_index) * TEXT_IMAGE_ROW_HEIGHT -
		   row_top_offset;
	area->x2 = area->x1 + CONTENT_WIDTH - 1;
	area->y2 = area->y1 + TEXT_IMAGE_ROW_HEIGHT - 1;
	return true;
}

void WifiSsidPage::update_selected_marquee()
{
	const bool compiled_overlay_ready = _selected_marquee_overlay != nullptr &&
		_selected_marquee_label != nullptr;
	const auto show_compiled_fallback = [this](const std::string &label) {
		if (_selected_raster_marquee != nullptr) {
			_selected_raster_marquee->hide();
		}
		if (_selected_marquee_label == nullptr ||
		    !lv_obj_is_valid(_selected_marquee_label) ||
		    _selected_marquee_overlay == nullptr ||
		    !lv_obj_is_valid(_selected_marquee_overlay)) {
			return;
		}
		lv_obj_remove_flag(_selected_marquee_label, LV_OBJ_FLAG_HIDDEN);
		const char *current = lv_label_get_text(_selected_marquee_label);
		if (current == nullptr || label != current) {
			lv_label_set_text(_selected_marquee_label, label.c_str());
		}
		lv_obj_clear_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN);
		start_selected_label_scroll();
		set_native_selected_text_visible(false);
		lv_obj_invalidate(_roller);
	};
	if (!_enable_ssid_scroll || !compiled_overlay_ready || _roller == nullptr ||
	    !_cache_page_active || _state == ViewState::Loading ||
	    cache_scroll_hold_active() || _selected_index >= _options.size() ||
	    !lv_obj_is_valid(_roller) ||
	    lv_obj_has_flag(_roller, LV_OBJ_FLAG_HIDDEN) ||
	    (text_image_cache_enabled() && !_cache_content_ready)) {
		hide_selected_marquee();
		return;
	}
	if (!lv_obj_is_valid(_selected_marquee_overlay) ||
	    !lv_obj_is_valid(_selected_marquee_label)) {
		hide_selected_marquee();
		return;
	}

	const lv_font_t *const font = _visible_ssid_font;
	if (font == nullptr ||
	    lv_font_get_line_height(font) > TEXT_IMAGE_ROW_HEIGHT) {
		hide_selected_marquee();
		return;
	}

	lv_obj_update_layout(_roller);
	lv_area_t row_area {};
	lv_area_t roller_area {};
	lv_area_t content_area {};
	if (!cached_text_row_area(_selected_index, &row_area)) {
		hide_selected_marquee();
		return;
	}
	lv_obj_get_coords(_roller, &roller_area);
	if (row_area.x1 < roller_area.x1 || row_area.x2 > roller_area.x2 ||
	    row_area.y1 < roller_area.y1 || row_area.y2 > roller_area.y2) {
		hide_selected_marquee();
		return;
	}

	const std::string &label = _options[_selected_index].label;
	const bool runtime_raster_needed = runtime_tiny_ttf_mode() &&
		SsidRasterMarquee::measure_text(label, _ssid_font) > CONTENT_WIDTH;
	if (runtime_tiny_ttf_mode() && !runtime_raster_needed) {
		cancel_selected_marquee_request();
		hide_selected_marquee();
		return;
	}
	lv_obj_get_content_coords(_roller, &content_area);
	lv_obj_set_pos(_selected_marquee_overlay,
		row_area.x1 - content_area.x1,
		row_area.y1 - content_area.y1);
	lv_obj_move_to_index(_selected_marquee_overlay, -1);

	if (runtime_raster_needed) {
		const bool raster_cache_healthy = text_image_cache_enabled() &&
			!_cache_activation_failed && _cache_content_ready;
		if (!raster_cache_healthy || !ensure_selected_raster_marquee()) {
			cancel_selected_marquee_request();
			show_compiled_fallback(label);
			return;
		}

		const auto request = _selected_raster_marquee->request_text(label);
		if (request == SsidRasterMarquee::RequestState::Ready) {
			_selected_marquee_pending_text.clear();
			_selected_marquee_pending_generation = 0;
			_selected_marquee_pending_index = 0;
			lv_obj_t *const viewport = _selected_raster_marquee->get_obj();
			if (viewport != nullptr && lv_obj_is_valid(viewport)) {
				lv_obj_set_pos(viewport, 0, 0);
				lv_obj_move_to_index(viewport, -1);
			}
			if (!_selected_raster_marquee->show()) {
				release_selected_raster_marquee();
				show_compiled_fallback(label);
				return;
			}
			stop_selected_label_scroll();
			lv_obj_add_flag(_selected_marquee_label, LV_OBJ_FLAG_HIDDEN);
			lv_obj_clear_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN);
			set_native_selected_text_visible(false);
			lv_obj_invalidate(_roller);
			return;
		}
		if (request == SsidRasterMarquee::RequestState::Pending) {
			_selected_marquee_pending_generation = _selected_marquee_generation;
			_selected_marquee_pending_index = _selected_index;
			_selected_marquee_pending_text = label;
			_selected_raster_marquee->hide();
			stop_selected_label_scroll();
			lv_obj_add_flag(_selected_marquee_label, LV_OBJ_FLAG_HIDDEN);
			lv_obj_add_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN);
			const bool native_source_ready = _text_image_cache != nullptr &&
				_text_image_cache->source_for(label) != nullptr;
			if (native_source_ready) {
				set_native_selected_text_visible(true);
			} else {
				// A row that has not reached the fixed cache yet would otherwise be
				// blank while the natural-width snapshot is pending.  Keep the
				// compiled overlay visible as a safe, responsive fallback.
				show_compiled_fallback(label);
			}
			start_cache_timer();
			lv_obj_invalidate(_roller);
			return;
		}

		cancel_selected_marquee_request();
		show_compiled_fallback(label);
		return;
	}

	if (_selected_raster_marquee != nullptr) {
		_selected_raster_marquee->hide();
	}
	show_compiled_fallback(label);
}

void WifiSsidPage::hide_selected_marquee()
{
	set_native_selected_text_visible(true);
	if (_selected_raster_marquee != nullptr) {
		_selected_raster_marquee->hide();
	}
	stop_selected_label_scroll();
	if (_selected_marquee_overlay != nullptr &&
	    lv_obj_is_valid(_selected_marquee_overlay)) {
		lv_obj_add_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN);
	}
	if (_roller != nullptr && lv_obj_is_valid(_roller)) {
		lv_obj_invalidate(_roller);
	}
}

void WifiSsidPage::cancel_selected_marquee_request() noexcept
{
	++_selected_marquee_generation;
	_selected_marquee_pending_generation = 0;
	_selected_marquee_pending_index = 0;
	_selected_marquee_pending_text.clear();
	if (_selected_raster_marquee != nullptr) {
		_selected_raster_marquee->cancel_pending();
	}
}

bool WifiSsidPage::selected_raster_work_pending() const noexcept
{
	return _selected_raster_marquee != nullptr &&
	       _selected_raster_marquee->is_pending() &&
	       !_selected_marquee_pending_text.empty();
}

bool WifiSsidPage::cache_timer_work_pending() const noexcept
{
	if (!_cache_page_active || _state == ViewState::Loading ||
	    cache_scroll_hold_active()) {
		return false;
	}
	const bool fixed_cache_pending = text_image_cache_enabled() &&
		_text_image_cache->has_pending();
	const bool selected_cache_pending = selected_raster_work_pending() &&
		runtime_tiny_ttf_mode() && !_cache_activation_failed &&
		_cache_content_ready && text_image_cache_enabled();
	return fixed_cache_pending || selected_cache_pending;
}

bool WifiSsidPage::service_one_selected_raster_work()
{
	if (!selected_raster_work_pending()) {
		return false;
	}
	const bool request_is_current =
		_selected_marquee_pending_generation == _selected_marquee_generation &&
		_selected_marquee_pending_index == _selected_index &&
		_selected_index < _options.size() &&
		_selected_marquee_pending_text == _options[_selected_index].label;
	const bool request_is_allowed = _cache_page_active &&
		_state != ViewState::Loading && !cache_scroll_hold_active() &&
		_cache_content_ready && text_image_cache_enabled() &&
		!_cache_activation_failed && runtime_tiny_ttf_mode();
	if (!request_is_current || !request_is_allowed) {
		cancel_selected_marquee_request();
		return false;
	}

	const auto result = _selected_raster_marquee->process_pending();
	if (result == SsidRasterMarquee::RequestState::Pending) {
		return true;
	}
	if (result == SsidRasterMarquee::RequestState::Ready) {
		_selected_marquee_pending_generation = 0;
		_selected_marquee_pending_index = 0;
		_selected_marquee_pending_text.clear();
		update_selected_marquee();
		return true;
	}
	if (result == SsidRasterMarquee::RequestState::Failed) {
		_selected_marquee_pending_generation = 0;
		_selected_marquee_pending_index = 0;
		_selected_marquee_pending_text.clear();
		update_selected_marquee();
		return true;
	}
	return false;
}

void WifiSsidPage::start_selected_label_scroll()
{
	if (_selected_marquee_overlay == nullptr || _selected_marquee_label == nullptr ||
	    !lv_obj_is_valid(_selected_marquee_overlay) ||
	    !lv_obj_is_valid(_selected_marquee_label) ||
	    lv_obj_has_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN) ||
	    lv_obj_has_flag(_selected_marquee_label, LV_OBJ_FLAG_HIDDEN)) {
		return;
	}
	if (lv_label_get_long_mode(_selected_marquee_label) !=
	    LV_LABEL_LONG_SCROLL_CIRCULAR) {
		lv_obj_update_layout(_selected_marquee_overlay);
		lv_label_set_long_mode(_selected_marquee_label,
			LV_LABEL_LONG_SCROLL_CIRCULAR);
	}
}

void WifiSsidPage::stop_selected_label_scroll()
{
	if (_selected_marquee_label != nullptr &&
	    lv_obj_is_valid(_selected_marquee_label)) {
		lv_label_set_long_mode(_selected_marquee_label, LV_LABEL_LONG_CLIP);
	}
}

bool WifiSsidPage::selected_marquee_visible() const noexcept
{
	const bool raster_visible = _selected_raster_marquee != nullptr &&
		_selected_raster_marquee->is_visible();
	const bool label_visible = _selected_marquee_overlay != nullptr &&
	       _selected_marquee_label != nullptr &&
	       lv_obj_is_valid(_selected_marquee_overlay) &&
	       lv_obj_is_valid(_selected_marquee_label) &&
	       !lv_obj_has_flag(_selected_marquee_overlay, LV_OBJ_FLAG_HIDDEN) &&
	       !lv_obj_has_flag(_selected_marquee_label, LV_OBJ_FLAG_HIDDEN);
	return raster_visible || label_visible;
}

void WifiSsidPage::set_native_selected_text_visible(bool visible)
{
	if (_roller == nullptr || !lv_obj_is_valid(_roller)) {
		_selected_marquee_hides_native_text = false;
		return;
	}

	if (visible) {
		if (_selected_marquee_hides_native_text) {
			lv_obj_set_style_text_opa(_roller, LV_OPA_COVER,
						  LV_PART_SELECTED);
			_selected_marquee_hides_native_text = false;
		}
		return;
	}

	if (!text_image_cache_enabled() && !_selected_marquee_hides_native_text) {
		lv_obj_set_style_text_opa(_roller, LV_OPA_TRANSP, LV_PART_SELECTED);
		_selected_marquee_hides_native_text = true;
	}
}

void WifiSsidPage::mark_cache_content_ready()
{
	if (_cache_content_ready) {
		return;
	}
	// The final raster step can take longer than the preceding Present step.
	// Treat the list reveal as the last loading-presentation update so the
	// timing diagnostic includes that final visible gap as well.
	if (_cache_timing_active && _loading_presentation_clock_started) {
		_max_loading_presentation_gap_ms = std::max(
			_max_loading_presentation_gap_ms,
			lv_tick_elaps(_last_loading_presentation_tick));
	}
	const uint32_t elapsed_ms [[maybe_unused]] =
		_cache_timing_active ? lv_tick_elaps(_cache_result_started_tick) : 0U;
	SPDLOG_DEBUG("[wifi] SSID image cache ready: elapsed_ms={}, "
		      "max_presentation_gap_ms={}",
		      elapsed_ms, _max_loading_presentation_gap_ms);
	_cache_content_ready = true;
	_cache_preload_phase = CachePreloadPhase::Inactive;
	_cache_timing_active = false;
	if (_wifi_loading_mode == WifiLoadingMode::Progress &&
	    _loading_phase == LoadingPhase::Idle) {
		// A cache with no pending rows can become ready before the first
		// presentation callback. Establish the session boundary once so the
		// fast finish still has a valid origin tick.
		initialize_loading_progress();
	}
	if (_wifi_loading_mode == WifiLoadingMode::Progress &&
	    _loading_phase == LoadingPhase::Rendering) {
		// drawn == total: progress already climbed to 100 via drawn/total.
		_loading_progress = 100;
		_loading_phase = LoadingPhase::Done;
		if (_loading_arc != nullptr) {
			lv_arc_set_angles(_loading_arc, 0, 360);
			lv_obj_invalidate(_loading_arc);
		}
	} else if (_wifi_loading_mode == WifiLoadingMode::Progress &&
	    (_loading_phase == LoadingPhase::Climbing ||
	     _loading_phase == LoadingPhase::Waiting)) {
		// Warm cache, nothing left to raster: close through 250 ms finish.
		_loading_progress = std::max(_loading_progress,
			static_cast<uint8_t>(LOADING_SCAN_TARGET_PERCENT));
		_loading_finish_base = _loading_progress;
		_loading_ready_tick = lv_tick_get();
		_loading_phase = LoadingPhase::Finishing;
	}
	update_view();
}

void WifiSsidPage::start_cache_timer()
{
	if (!_cache_page_active || !cache_timer_work_pending() ||
	    _cache_timer != nullptr || cache_scroll_hold_active()) {
		return;
	}

	_cache_timer = lv_timer_create(cache_timer_cb, CACHE_RENDER_PERIOD_MS, this);
	if (_cache_timer == nullptr) {
		spdlog::warn("SSID image cache timer creation failed");
		disable_text_image_cache();
	}
}

void WifiSsidPage::stop_cache_timer()
{
	if (_cache_timer != nullptr) {
		lv_timer_delete(_cache_timer);
		_cache_timer = nullptr;
	}
}

void WifiSsidPage::start_cache_scroll_hold()
{
	start_cache_drag_hold();
	if (!cache_scroll_hold_active()) {
		return;
	}

	if (_roller == nullptr || !lv_obj_is_valid(_roller)) {
		_cache_scroll_hold = false;
		return;
	}

	const uint32_t animation_duration =
		lv_obj_get_style_anim_duration(_roller, LV_PART_MAIN);
	if (animation_duration == 0U) {
		_cache_scroll_hold = false;
		update_selected_marquee();
		return;
	}

	_cache_scroll_hold_timer = lv_timer_create(cache_scroll_hold_timer_cb,
						      animation_duration, this);
	if (_cache_scroll_hold_timer == nullptr) {
		spdlog::warn("SSID image cache scroll hold timer creation failed");
		_cache_scroll_hold = false;
		update_selected_marquee();
		return;
	}
	lv_timer_set_repeat_count(_cache_scroll_hold_timer, 1);
}

void WifiSsidPage::start_cache_drag_hold()
{
	if (_roller == nullptr || !lv_obj_is_valid(_roller)) {
		return;
	}

	cancel_selected_marquee_request();
	// LVGL moves the private label while pressed, before it reports the new
	// selection. Keep both cache sources and the selected marquee out of that
	// moving viewport until the selected row is stable again.
	if (_cache_scroll_hold_timer != nullptr) {
		lv_timer_delete(_cache_scroll_hold_timer);
		_cache_scroll_hold_timer = nullptr;
	}
	_cache_scroll_hold = true;
	hide_selected_marquee();
	if (text_image_cache_enabled()) {
		stop_cache_timer();
	}
}

void WifiSsidPage::finish_cache_drag_hold()
{
	if (!_cache_scroll_hold || _cache_scroll_hold_timer != nullptr) {
		return;
	}

	_cache_scroll_hold = false;
	queue_text_image_window();
	update_selected_marquee();
}

void WifiSsidPage::stop_cache_scroll_hold()
{
	if (_cache_scroll_hold_timer != nullptr) {
		lv_timer_delete(_cache_scroll_hold_timer);
		_cache_scroll_hold_timer = nullptr;
	}
	_cache_scroll_hold = false;
	_roller_dragged = false;
}

bool WifiSsidPage::cache_scroll_hold_active() const noexcept
{
	return _cache_scroll_hold;
}

void WifiSsidPage::disable_text_image_cache()
{
	_cache_activation_failed = true;
	cancel_selected_marquee_request();
	release_selected_raster_marquee();
	release_text_image_cache();
	_cache_content_ready = true;
	reset_cache_scheduler();
	update_roller_options();
	update_view();
}

void WifiSsidPage::apply_roller_text_style()
{
	if (_roller == nullptr) {
		return;
	}

	if (text_image_cache_enabled()) {
		const lv_font_t *const layout_font = runtime_tiny_ttf_mode() ?
			_visible_ssid_font : &lv_font_montserrat_20;
		const int32_t font_height =
			lv_font_get_line_height(layout_font);
		const int32_t line_space = TEXT_IMAGE_ROW_HEIGHT > font_height ?
			static_cast<int32_t>(TEXT_IMAGE_ROW_HEIGHT) - font_height :
			0;
		lv_obj_set_style_text_font(_roller, layout_font,
						   LV_PART_MAIN);
		lv_obj_set_style_text_font(_roller, layout_font,
						   LV_PART_SELECTED);
		lv_obj_set_style_text_line_space(_roller, line_space,
					     LV_PART_MAIN);
		lv_obj_set_style_text_line_space(_roller, line_space,
					     LV_PART_SELECTED);
		lv_obj_set_height(_roller, ROLLER_HEIGHT);
		return;
	}

	lv_obj_set_style_text_font(_roller, _visible_ssid_font, LV_PART_MAIN);
	lv_obj_set_style_text_font(_roller, _visible_ssid_font, LV_PART_SELECTED);
	const int32_t font_height =
		lv_font_get_line_height(_visible_ssid_font);
	const int32_t line_space = TEXT_IMAGE_ROW_HEIGHT > font_height ?
		static_cast<int32_t>(TEXT_IMAGE_ROW_HEIGHT) - font_height :
		0;
	lv_obj_set_style_text_line_space(_roller, line_space, LV_PART_MAIN);
	lv_obj_set_style_text_line_space(_roller, line_space,
					 LV_PART_SELECTED);
	// Keep the native fallback on the same fixed row contract as the image
	// cache.  lv_roller_set_visible_row_count() derives the height from the
	// font and would shrink this 230x112 roller for the 22px fallback font.
	lv_obj_set_height(_roller, ROLLER_HEIGHT);
}

std::vector<std::string> WifiSsidPage::foreground_text_labels() const
{
	std::vector<std::string> labels;
	if (_selected_index >= _options.size()) {
		return labels;
	}

	labels.push_back(_options[_selected_index].label);
	if (_selected_index > 0) {
		labels.push_back(_options[_selected_index - 1].label);
	}
	if (_selected_index + 1 < _options.size()) {
		labels.push_back(_options[_selected_index + 1].label);
	}
	return labels;
}

std::vector<std::string> WifiSsidPage::speculative_text_labels() const
{
	std::vector<std::string> labels;
	labels.reserve(_options.size());
	for (const RollerOption &option : _options) {
		labels.push_back(option.label);
	}
	return labels;
}

lv_obj_t *WifiSsidPage::roller_label() const noexcept
{
	if (_roller == nullptr || !lv_obj_is_valid(_roller)) {
		return nullptr;
	}

	lv_obj_t *label = lv_obj_get_child(_roller, 0);
	if (label == nullptr || !lv_obj_is_valid(label) ||
	    !lv_obj_has_class(label, &lv_label_class)) {
		return nullptr;
	}
	return label;
}

void WifiSsidPage::set_cache_roller_label_width()
{
	lv_obj_t *label = roller_label();
	if (label == nullptr) {
		return;
	}

	// The placeholder is only one space wide. Its motion invalidates the old and
	// new label bounds, so make those bounds span every cached image pixel.
	lv_obj_set_width(label, CONTENT_WIDTH);
}

void WifiSsidPage::reset_roller_label_width()
{
	lv_obj_t *label = roller_label();
	if (label == nullptr) {
		return;
	}

	lv_obj_remove_local_style_prop(label, LV_STYLE_WIDTH, LV_PART_MAIN);
}

bool WifiSsidPage::text_image_cache_enabled() const noexcept
{
	return _text_image_cache != nullptr && _roller != nullptr;
}

bool WifiSsidPage::runtime_tiny_ttf_mode() const noexcept
{
	return _ssid_font_render_mode == SsidFontRenderMode::RuntimeTinyTtf;
}

void WifiSsidPage::reset_lvgl_pointers()
{
	_navigation_box = nullptr;
	_loading_box = nullptr;
	_loading_arc = nullptr;
	_roller = nullptr;
	_selected_marquee_overlay = nullptr;
	_selected_marquee_label = nullptr;
	_cache_timer = nullptr;
	_cache_scroll_hold_timer = nullptr;
	_cache_scroll_hold = false;
	_roller_dragged = false;
	_selected_marquee_hides_native_text = false;
	_selected_marquee_pending_generation = 0;
	_selected_marquee_pending_index = 0;
	_selected_marquee_pending_text.clear();
}

const WifiSsidPage::RollerOption *WifiSsidPage::current_option() const
{
	if (_selected_index >= _options.size()) {
		return nullptr;
	}
	return &_options[_selected_index];
}

void WifiSsidPage::roller_event_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiSsidPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_roller == nullptr) {
		return;
	}
	const uint32_t selected_index = lv_roller_get_selected(self->_roller);
	const bool selection_changed = selected_index != self->_selected_index;
	self->_selected_index = selected_index;
	if (selection_changed) {
		self->cancel_selected_marquee_request();
		self->start_cache_scroll_hold();
	}
	self->queue_text_image_window();
}

void WifiSsidPage::roller_pressed_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiSsidPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_roller == nullptr) {
		return;
	}
	self->_roller_dragged = false;
	self->start_cache_drag_hold();
}

void WifiSsidPage::roller_pressing_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiSsidPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_roller == nullptr) {
		return;
	}

	lv_indev_t *const indev = lv_indev_active();
	if (indev == nullptr) {
		return;
	}
	lv_point_t movement {};
	lv_indev_get_vect(indev, &movement);
	if (movement.y != 0) {
		self->_roller_dragged = true;
	}
}

void WifiSsidPage::roller_released_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiSsidPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_roller == nullptr) {
		return;
	}
	const bool dragged = self->_roller_dragged;
	self->_roller_dragged = false;
	if (dragged && self->_cache_scroll_hold_timer == nullptr) {
		// LVGL aligns its private label after every vertical drag, including a
		// rebound to the current option.  Delay the floating overlay until that
		// existing animation has settled.
		self->start_cache_scroll_hold();
		return;
	}
	self->finish_cache_drag_hold();
}

void WifiSsidPage::roller_draw_post_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiSsidPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_roller == nullptr) {
		return;
	}
	self->draw_text_image_rows(event);
}

void WifiSsidPage::loading_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<WifiSsidPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr || !self->loading_presentation_active() ||
	    self->_loading_arc == nullptr) {
		return;
	}

	self->advance_loading_for_mode();
}

void WifiSsidPage::cache_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<WifiSsidPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}
	if (!self->cache_timer_work_pending()) {
		self->stop_cache_timer();
		return;
	}

	if (self->service_one_selected_raster_work()) {
		return;
	}
	self->render_next_text_image();
}

void WifiSsidPage::cache_scroll_hold_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<WifiSsidPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}

	self->_cache_scroll_hold_timer = nullptr;
	self->_cache_scroll_hold = false;
	if (self->text_image_cache_enabled() && self->_cache_page_active &&
	    self->_state != ViewState::Loading) {
		self->queue_text_image_window();
	}
	self->update_selected_marquee();
}

} // namespace ui
