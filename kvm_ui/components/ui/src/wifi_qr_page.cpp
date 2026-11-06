#include "ui/wifi_qr_page.hpp"

#include <cstring>

#include "lv_safe.hpp"

namespace ui
{

namespace
{

constexpr int32_t WIDTH = 270;
constexpr int32_t HEIGHT = 240;
constexpr int32_t CARD_W = WIDTH;
constexpr int32_t CARD_H = HEIGHT;
constexpr int32_t QR_MARGIN_BOTTOM = 15;
constexpr int32_t QR_BG_W = 237;
constexpr int32_t QR_BG_H = 178;
constexpr uint16_t LOADING_ARC_SIZE = 72;
constexpr uint16_t LOADING_ARC_WIDTH = 8;
constexpr uint32_t LOADING_PERIOD_MS = 40;
constexpr uint32_t ACTIVE_COLOR = 0x2B7FFF;

} // namespace

WifiQrPage::WifiQrPage()
	: Page(PageType::SUBPAGE_WIFI_QR)
{
}

void WifiQrPage::create(lv_obj_t *parent)
{
	if (_container != nullptr) {
		return;
	}

	_container = lv_obj_create(parent);
	lv_obj_set_size(_container, WIDTH, HEIGHT);
	lv_obj_set_style_bg_color(_container, lv_color_black(), 0);
	lv_obj_set_style_bg_opa(_container, LV_OPA_70, 0);
	lv_obj_set_style_border_width(_container, 0, 0);
	lv_obj_set_style_radius(_container, 40, 0);
	lv_obj_set_style_pad_all(_container, 0, 0);
	lv_obj_set_style_margin_all(_container, 0, 0);
	lv_obj_set_style_shadow_width(_container, 0, 0);
	lv_obj_set_style_outline_width(_container, 0, 0);
	lv_obj_set_scrollbar_mode(_container, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_container, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_container, LV_OBJ_FLAG_CLICKABLE);

	lv_obj_t *card = lv_obj_create(_container);
	lv_obj_set_size(card, CARD_W, CARD_H);
	lv_obj_set_pos(card, 0, 0);
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

	_qr_title_label = lv_label_create(card);
	lv_label_set_text(_qr_title_label, _qr_title.c_str());
	lv_obj_set_style_text_color(_qr_title_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_qr_title_label, &lv_font_montserrat_20, 0);
	lv_obj_align(_qr_title_label, LV_ALIGN_TOP_MID, 0, 8);

	_qr_background = lv_obj_create(card);
	lv_obj_set_size(_qr_background, QR_BG_W, QR_BG_H);
	lv_obj_align(_qr_background, LV_ALIGN_BOTTOM_MID, 0, -QR_MARGIN_BOTTOM);
	lv_obj_set_style_bg_color(_qr_background, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(_qr_background, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_qr_background, 30, 0);
	lv_obj_set_style_border_width(_qr_background, 0, 0);
	lv_obj_set_style_pad_all(_qr_background, 0, 0);
	lv_obj_set_style_margin_all(_qr_background, 0, 0);
	lv_obj_set_style_shadow_width(_qr_background, 0, 0);
	lv_obj_set_style_outline_width(_qr_background, 0, 0);
	lv_obj_set_scrollbar_mode(_qr_background, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_qr_background, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_qr_background, LV_OBJ_FLAG_CLICKABLE);

	_qr_code = lv_qrcode_create(_qr_background);
	lv_qrcode_set_size(_qr_code, 148);
	lv_qrcode_set_dark_color(_qr_code, lv_color_hex(0x000000));
	lv_qrcode_set_light_color(_qr_code, lv_color_hex(0xFFFFFF));
	lv_obj_center(_qr_code);
	lv_obj_add_flag(_qr_code, LV_OBJ_FLAG_CLICKABLE);

	_loading_arc = lv_arc_create(_qr_background);
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
	lv_obj_add_flag(_loading_arc, LV_OBJ_FLAG_HIDDEN);
	refresh_qr_code();

	lv_obj_add_event_cb(_container, click_event_cb, LV_EVENT_CLICKED, this);
	lv_obj_add_event_cb(card, click_event_cb, LV_EVENT_CLICKED, this);
	lv_obj_add_event_cb(_qr_background, click_event_cb, LV_EVENT_CLICKED,
			    this);
	lv_obj_add_event_cb(_qr_code, click_event_cb, LV_EVENT_CLICKED, this);
	reset_session();
}

void WifiQrPage::destroy()
{
	reset_session();
	on_close_ready = nullptr;
	on_qr_start_request = nullptr;
	on_qr_stop_request = nullptr;
	_qr_title_label = nullptr;
	_qr_background = nullptr;
	_qr_code = nullptr;
	_loading_arc = nullptr;
	lv_delete_if_valid(_container);
}

void WifiQrPage::on_enter()
{
	reset_session();
	if (on_qr_start_request) {
		on_qr_start_request();
	}
}

void WifiQrPage::on_exit()
{
	reset_session();
}

void WifiQrPage::set_qr_payload(const std::string &payload)
{
	_qr_payload = payload;
	refresh_qr_code();
}

void WifiQrPage::set_ap_stop_completed_generation(uint64_t generation)
{
	if (!_closing || _stop_request_generation == 0 ||
	    generation < _stop_request_generation) {
		return;
	}
	notify_close_ready();
}

void WifiQrPage::set_title(std::string_view title)
{
	_qr_title = title.empty() ? DEFAULT_TITLE : std::string(title);
	refresh_qr_code();
}

void WifiQrPage::refresh_qr_code()
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

void WifiQrPage::reset_session()
{
	lv_async_call_cancel(close_ready_async_cb, this);
	_close_ready_scheduled = false;
	_closing = false;
	_stop_request_generation = 0;
	stop_loading_animation();
	_loading_rotation = 0;
	if (_loading_arc != nullptr && lv_obj_is_valid(_loading_arc)) {
		lv_arc_set_rotation(_loading_arc, _loading_rotation);
		lv_obj_add_flag(_loading_arc, LV_OBJ_FLAG_HIDDEN);
	}
	if (_qr_code != nullptr && lv_obj_is_valid(_qr_code)) {
		lv_obj_clear_flag(_qr_code, LV_OBJ_FLAG_HIDDEN);
	}
	refresh_qr_code();
}

void WifiQrPage::begin_closing()
{
	if (_closing) {
		return;
	}

	_closing = true;
	if (_qr_code != nullptr && lv_obj_is_valid(_qr_code)) {
		lv_obj_add_flag(_qr_code, LV_OBJ_FLAG_HIDDEN);
	}
	start_loading_animation();
	_stop_request_generation = on_qr_stop_request ? on_qr_stop_request() :
							0;
	if (_stop_request_generation == 0) {
		schedule_close_ready();
	}
}

void WifiQrPage::start_loading_animation()
{
	if (_loading_arc == nullptr || !lv_obj_is_valid(_loading_arc)) {
		return;
	}

	lv_obj_clear_flag(_loading_arc, LV_OBJ_FLAG_HIDDEN);
	lv_arc_set_rotation(_loading_arc, _loading_rotation);
	if (_loading_timer == nullptr) {
		_loading_timer = lv_timer_create(loading_timer_cb,
						 LOADING_PERIOD_MS, this);
	}
}

void WifiQrPage::stop_loading_animation()
{
	if (_loading_timer != nullptr) {
		lv_timer_delete(_loading_timer);
		_loading_timer = nullptr;
	}
}

void WifiQrPage::schedule_close_ready()
{
	if (_close_ready_scheduled) {
		return;
	}
	_close_ready_scheduled = true;
	lv_async_call(close_ready_async_cb, this);
}

void WifiQrPage::notify_close_ready()
{
	if (!_closing || !on_close_ready) {
		return;
	}
	on_close_ready();
}

void WifiQrPage::click_event_cb(lv_event_t *event)
{
	auto *self = static_cast<WifiQrPage *>(lv_event_get_user_data(event));
	if (self != nullptr) {
		self->begin_closing();
	}
}

void WifiQrPage::loading_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<WifiQrPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr || !self->_closing ||
	    self->_loading_arc == nullptr ||
	    !lv_obj_is_valid(self->_loading_arc)) {
		return;
	}

	self->_loading_rotation =
		static_cast<uint16_t>((self->_loading_rotation + 18) % 360);
	lv_arc_set_rotation(self->_loading_arc, self->_loading_rotation);
}

void WifiQrPage::close_ready_async_cb(void *user_data)
{
	auto *self = static_cast<WifiQrPage *>(user_data);
	if (self == nullptr) {
		return;
	}
	self->_close_ready_scheduled = false;
	self->notify_close_ready();
}

} // namespace ui
