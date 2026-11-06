#include "ui/wifi_connect_result_page.hpp"

namespace ui
{

namespace
{

constexpr uint16_t STATUS_WIDTH = 230;
constexpr uint16_t STATUS_HEIGHT = 32;
constexpr uint16_t FAILURE_STATUS_HEIGHT = 120;
constexpr uint16_t SPINNER_BOX_WIDTH = 230;
constexpr uint16_t SPINNER_BOX_HEIGHT = 64;
constexpr uint16_t SPINNER_SIZE = 60;
constexpr uint16_t SPINNER_ARC_WIDTH = 6;
constexpr uint32_t SPINNER_PERIOD_MS = 40;

} // namespace

WifiConnectResultPage::WifiConnectResultPage()
	: SubPageBase(PageType::SUBPAGE_WIFI_CONNECT_RESULT, "WiFi")
{
}

void WifiConnectResultPage::destroy()
{
	on_success_return_request = nullptr;
	on_back_click = nullptr;
	stop_spinner();
	cancel_success_return();
	_components.clear();
	_scroll_content = nullptr;
	_back_btn = nullptr;
	Page::destroy();
	reset_lvgl_pointers();
}

void WifiConnectResultPage::set_loading(std::string_view message)
{
	_state = State::Loading;
	_message = std::string(message);
	_spinner_rotation = 0;
	cancel_success_return();
	update_view();
}

void WifiConnectResultPage::set_success(std::string_view message)
{
	_state = State::Success;
	_message = std::string(message);
	update_view();
	schedule_success_return();
}

void WifiConnectResultPage::set_failure(std::string_view message)
{
	_state = State::Failure;
	_message = std::string(message);
	cancel_success_return();
	update_view();
}

void WifiConnectResultPage::build_content()
{
	create_status();
	create_spinner();
	update_view();
}

void WifiConnectResultPage::on_subpage_destroy()
{
	stop_spinner();
	cancel_success_return();
	reset_lvgl_pointers();
}

void WifiConnectResultPage::reset_lvgl_pointers()
{
	_status_box = nullptr;
	_status_label = nullptr;
	_spinner_box = nullptr;
	_spinner_arc = nullptr;
}

void WifiConnectResultPage::create_status()
{
	if (_scroll_content == nullptr || _status_box != nullptr) {
		return;
	}

	_status_box = lv_obj_create(_scroll_content);
	lv_obj_set_size(_status_box, STATUS_WIDTH, STATUS_HEIGHT);
	lv_obj_set_style_bg_opa(_status_box, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_status_box, 0, 0);
	lv_obj_set_style_pad_all(_status_box, 0, 0);
	lv_obj_clear_flag(_status_box, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_status_box, LV_OBJ_FLAG_CLICKABLE);

	_status_label = lv_label_create(_status_box);
	lv_obj_set_width(_status_label, STATUS_WIDTH);
	lv_label_set_long_mode(_status_label, LV_LABEL_LONG_WRAP);
	lv_obj_set_style_text_color(_status_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_status_label, &lv_font_montserrat_20, 0);
	lv_obj_set_style_text_align(_status_label, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_center(_status_label);
}

void WifiConnectResultPage::create_spinner()
{
	if (_scroll_content == nullptr || _spinner_box != nullptr) {
		return;
	}

	_spinner_box = lv_obj_create(_scroll_content);
	lv_obj_set_size(_spinner_box, SPINNER_BOX_WIDTH, SPINNER_BOX_HEIGHT);
	lv_obj_set_style_bg_opa(_spinner_box, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_spinner_box, 0, 0);
	lv_obj_set_style_pad_all(_spinner_box, 0, 0);
	lv_obj_clear_flag(_spinner_box, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_spinner_box, LV_OBJ_FLAG_CLICKABLE);

	_spinner_arc = lv_arc_create(_spinner_box);
	lv_obj_set_size(_spinner_arc, SPINNER_SIZE, SPINNER_SIZE);
	lv_obj_center(_spinner_arc);
	lv_obj_clear_flag(_spinner_arc, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_spinner_arc, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_remove_style(_spinner_arc, nullptr, LV_PART_KNOB);
	lv_obj_set_style_arc_opa(_spinner_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_spinner_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_spinner_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_spinner_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_arc_width(_spinner_arc, SPINNER_ARC_WIDTH,
				   LV_PART_INDICATOR);
	lv_obj_set_style_arc_rounded(_spinner_arc, true, LV_PART_INDICATOR);
	lv_arc_set_bg_angles(_spinner_arc, 0, 360);
	lv_arc_set_angles(_spinner_arc, 0, 270);
	lv_arc_set_rotation(_spinner_arc, 0);
}

void WifiConnectResultPage::update_view()
{
	if (_status_box != nullptr) {
		lv_obj_set_height(_status_box, _state == State::Failure ?
						       FAILURE_STATUS_HEIGHT :
						       STATUS_HEIGHT);
		if (_state == State::Loading) {
			lv_obj_add_flag(_status_box, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(_status_box, LV_OBJ_FLAG_HIDDEN);
		}
	}

	if (_status_label != nullptr) {
		lv_label_set_text(_status_label, _state == State::Loading ?
							 "" :
							 _message.c_str());
		lv_obj_set_style_text_color(
			_status_label,
			_state == State::Failure ?
				lv_color_hex(FAILURE_TEXT_COLOR) :
				lv_color_white(),
			0);
		lv_obj_center(_status_label);
	}

	if (_spinner_box != nullptr) {
		if (_state == State::Failure) {
			lv_obj_add_flag(_spinner_box, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(_spinner_box, LV_OBJ_FLAG_HIDDEN);
		}
	}

	if (_spinner_arc != nullptr) {
		if (_state == State::Failure) {
			lv_obj_add_flag(_spinner_arc, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_clear_flag(_spinner_arc, LV_OBJ_FLAG_HIDDEN);
			lv_obj_set_style_arc_color(
				_spinner_arc, lv_color_hex(spinner_color()),
				LV_PART_INDICATOR);
			lv_arc_set_rotation(_spinner_arc, _spinner_rotation);
		}
	}

	if (_state == State::Loading) {
		start_spinner();
	} else {
		stop_spinner();
	}
	update_back_enabled();
}

void WifiConnectResultPage::update_back_enabled()
{
	if (_back_btn == nullptr || _back_btn->get_obj() == nullptr) {
		return;
	}
	if (_state == State::Loading) {
		lv_obj_clear_flag(_back_btn->get_obj(), LV_OBJ_FLAG_CLICKABLE);
	} else {
		lv_obj_add_flag(_back_btn->get_obj(), LV_OBJ_FLAG_CLICKABLE);
	}
}

void WifiConnectResultPage::start_spinner()
{
	if (_spinner_arc == nullptr) {
		return;
	}
	lv_obj_clear_flag(_spinner_arc, LV_OBJ_FLAG_HIDDEN);
	if (_spinner_timer == nullptr) {
		_spinner_timer = lv_timer_create(spinner_timer_cb,
						 SPINNER_PERIOD_MS, this);
	}
}

void WifiConnectResultPage::stop_spinner()
{
	if (_spinner_timer != nullptr) {
		lv_timer_delete(_spinner_timer);
		_spinner_timer = nullptr;
	}
}

void WifiConnectResultPage::schedule_success_return()
{
	if (_success_return_scheduled || !on_success_return_request) {
		return;
	}
	_success_return_scheduled = true;
	lv_async_call(success_return_async_cb, this);
}

void WifiConnectResultPage::cancel_success_return()
{
	if (!_success_return_scheduled) {
		return;
	}
	lv_async_call_cancel(success_return_async_cb, this);
	_success_return_scheduled = false;
}

uint32_t WifiConnectResultPage::spinner_color() const
{
	switch (_state) {
	case State::Success:
		return SUCCESS_COLOR;
	case State::Failure:
		return ACTIVE_COLOR;
	case State::Loading:
	default:
		return ACTIVE_COLOR;
	}
}

void WifiConnectResultPage::spinner_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<WifiConnectResultPage *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr || self->_state != State::Loading ||
	    self->_spinner_arc == nullptr) {
		return;
	}

	self->_spinner_rotation =
		static_cast<uint16_t>((self->_spinner_rotation + 18) % 360);
	lv_arc_set_rotation(self->_spinner_arc, self->_spinner_rotation);
}

void WifiConnectResultPage::success_return_async_cb(void *user_data)
{
	auto *self = static_cast<WifiConnectResultPage *>(user_data);
	if (self == nullptr) {
		return;
	}

	self->_success_return_scheduled = false;
	if (self->on_success_return_request) {
		self->on_success_return_request();
	}
}

} // namespace ui
