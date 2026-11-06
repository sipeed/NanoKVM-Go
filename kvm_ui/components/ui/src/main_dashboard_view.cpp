#include "ui/main_dashboard_view.hpp"

#include <algorithm>
#include <cerrno>
#include <cmath>
#include <cstdlib>
#include <string>
#include <string_view>
#include <utility>

#include "lv_safe.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_main_mouse_default);
LV_IMAGE_DECLARE(img_main_mouse_active);
LV_FONT_DECLARE(MontserratBold_24);
LV_FONT_DECLARE(MontserratBold_26);
}

namespace ui
{

namespace
{

constexpr std::string_view kIpReconnectPlaceholderBase = "Reconnecting";

std::string trim_ascii(std::string_view text)
{
	auto begin = text.begin();
	auto end = text.end();
	while (begin != end && (*begin == ' ' || *begin == '\t')) {
		++begin;
	}
	while (begin != end && (*(end - 1) == ' ' || *(end - 1) == '\t')) {
		--end;
	}
	return std::string(begin, end);
}

bool parse_positive_int(std::string_view text)
{
	const std::string value = trim_ascii(text);
	if (value.empty()) {
		return false;
	}
	int result = 0;
	for (const char ch : value) {
		if (ch < '0' || ch > '9') {
			return false;
		}
		if (result > 100000) {
			return false;
		}
		result = result * 10 + (ch - '0');
	}
	return result > 0;
}

bool is_valid_resolution_text(std::string_view text)
{
	const std::string value = trim_ascii(text);
	if (value.empty() || value == "--") {
		return false;
	}

	const std::size_t separator = value.find_first_of("xX*");
	if (separator == std::string::npos) {
		return false;
	}
	const std::string_view view(value);
	return view.find_first_of("xX*", separator + 1U) ==
		       std::string_view::npos &&
	       parse_positive_int(view.substr(0, separator)) &&
	       parse_positive_int(view.substr(separator + 1U));
}

bool is_valid_fps_text(std::string_view text)
{
	const std::string value = trim_ascii(text);
	if (value.empty() || value == "--") {
		return false;
	}

	char *end = nullptr;
	errno = 0;
	const double parsed = std::strtod(value.c_str(), &end);
	return errno == 0 && end != value.c_str() && end != nullptr &&
	       *end == '\0' && std::isfinite(parsed) && parsed > 0.0;
}

bool has_hid_label_data(const std::string &special, const std::string &normal)
{
	const auto has_data = [](const std::string &value) {
		return !value.empty() && value != "-";
	};
	return has_data(special) || has_data(normal);
}

} // namespace

MainDashboardView::~MainDashboardView()
{
	shutdown();
}

void MainDashboardView::create(lv_obj_t *parent,
			       const BackgroundImage *background)
{
	if (parent == nullptr || _content_obj != nullptr || _is_shutting_down) {
		return;
	}

	_content_obj = lv_obj_create(parent);
	lv_obj_set_pos(_content_obj, 0, 0);
	lv_obj_set_size(_content_obj, root_layout::kContentCardWidth,
			root_layout::kContentCardHeight);
	lv_obj_set_style_radius(_content_obj, root_layout::kContentCardRadius,
				0);
	lv_obj_set_style_bg_color(_content_obj, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_content_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_content_obj, 0, 0);
	lv_obj_set_style_pad_all(_content_obj, 0, 0);
	lv_obj_set_style_clip_corner(_content_obj, true, 0);
	lv_obj_clear_flag(_content_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_content_obj, LV_OBJ_FLAG_CLICKABLE);
	_background_obj = create_background_image_obj(_content_obj, background);

	lv_obj_t *ip_card = lv_obj_create(_content_obj);
	lv_obj_set_pos(ip_card, 15, 15);
	lv_obj_set_size(ip_card, 240, 60);
	lv_obj_set_style_radius(ip_card, 25, 0);
	lv_obj_set_style_bg_color(ip_card, lv_color_hex(0x1C1917), 0);
	lv_obj_set_style_bg_opa(ip_card, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(ip_card, 0, 0);
	lv_obj_set_style_pad_all(ip_card, 0, 0);
	lv_obj_clear_flag(ip_card, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(ip_card, LV_OBJ_FLAG_CLICKABLE);

	_ip_label = lv_label_create(ip_card);
	lv_label_set_text(_ip_label, "--");
	lv_obj_set_style_text_font(_ip_label, &MontserratBold_26, 0);
	lv_obj_set_style_text_color(_ip_label, lv_color_white(), 0);
	lv_obj_center(_ip_label);
	lv_obj_clear_flag(_ip_label, LV_OBJ_FLAG_CLICKABLE);

	_resolution_obj = lv_obj_create(_content_obj);
	lv_obj_set_pos(_resolution_obj, 20, 86);
	lv_obj_set_size(_resolution_obj, 116, 40);
	lv_obj_set_style_bg_color(_resolution_obj, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_resolution_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_resolution_obj, 0, 0);
	lv_obj_set_style_radius(_resolution_obj, 0, 0);
	lv_obj_set_style_pad_all(_resolution_obj, 0, 0);
	lv_obj_clear_flag(_resolution_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_resolution_obj, LV_OBJ_FLAG_CLICKABLE);

	_resolution_label = lv_label_create(_resolution_obj);
	lv_label_set_text(_resolution_label, _resolution_text.c_str());
	lv_obj_set_style_text_font(_resolution_label, &lv_font_montserrat_20,
				   0);
	lv_obj_set_style_text_color(_resolution_label, lv_color_white(), 0);
	lv_obj_center(_resolution_label);
	lv_obj_clear_flag(_resolution_label, LV_OBJ_FLAG_CLICKABLE);

	_video_status_obj = lv_obj_create(_content_obj);
	lv_obj_set_pos(_video_status_obj, 24, 126);
	lv_obj_set_size(_video_status_obj, 23, 23);
	lv_obj_set_style_radius(_video_status_obj, LV_RADIUS_CIRCLE, 0);
	lv_obj_set_style_pad_all(_video_status_obj, 0, 0);
	lv_obj_clear_flag(_video_status_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_video_status_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_set_style_border_width(_video_status_obj, 0, 0);
	lv_obj_set_style_border_color(_video_status_obj, lv_color_hex(0xFB2C36),
				      0);
	lv_obj_set_style_bg_color(_video_status_obj, lv_color_hex(0xFB2C36), 0);
	lv_obj_set_style_bg_opa(_video_status_obj, LV_OPA_COVER, 0);

	_fps_value_obj = lv_obj_create(_content_obj);
	lv_obj_set_pos(_fps_value_obj, 54, 126);
	lv_obj_set_size(_fps_value_obj, 46, 23);
	lv_obj_set_style_bg_color(_fps_value_obj, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_fps_value_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_fps_value_obj, 0, 0);
	lv_obj_set_style_radius(_fps_value_obj, 0, 0);
	lv_obj_set_style_pad_all(_fps_value_obj, 0, 0);
	lv_obj_clear_flag(_fps_value_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_fps_value_obj, LV_OBJ_FLAG_CLICKABLE);

	_fps_value_label = lv_label_create(_fps_value_obj);
	lv_label_set_text(_fps_value_label, _fps_text.c_str());
	lv_obj_set_style_text_font(_fps_value_label, &lv_font_montserrat_20, 0);
	lv_obj_set_style_text_color(_fps_value_label, lv_color_white(), 0);
	lv_obj_center(_fps_value_label);
	lv_obj_clear_flag(_fps_value_label, LV_OBJ_FLAG_CLICKABLE);

	_fps_unit_obj = lv_obj_create(_content_obj);
	lv_obj_set_pos(_fps_unit_obj, 103, 126);
	lv_obj_set_size(_fps_unit_obj, 33, 23);
	lv_obj_set_style_bg_color(_fps_unit_obj, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_fps_unit_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_fps_unit_obj, 0, 0);
	lv_obj_set_style_radius(_fps_unit_obj, 0, 0);
	lv_obj_set_style_pad_all(_fps_unit_obj, 0, 0);
	lv_obj_clear_flag(_fps_unit_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_fps_unit_obj, LV_OBJ_FLAG_CLICKABLE);

	_fps_unit_label = lv_label_create(_fps_unit_obj);
	lv_label_set_text(_fps_unit_label, "FPS");
	lv_obj_set_style_text_font(_fps_unit_label, &lv_font_montserrat_16, 0);
	lv_obj_set_style_text_color(_fps_unit_label, lv_color_white(), 0);
	lv_obj_center(_fps_unit_label);
	lv_obj_clear_flag(_fps_unit_label, LV_OBJ_FLAG_CLICKABLE);

	lv_obj_t *divider = lv_obj_create(_content_obj);
	lv_obj_set_pos(divider, 150, 98);
	lv_obj_set_size(divider, 2, 46);
	lv_obj_set_style_bg_color(divider, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(divider, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(divider, 0, 0);
	lv_obj_set_style_radius(divider, 0, 0);
	lv_obj_set_style_pad_all(divider, 0, 0);
	lv_obj_clear_flag(divider, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(divider, LV_OBJ_FLAG_CLICKABLE);

	lv_obj_t *keyboard_special = lv_obj_create(_content_obj);
	lv_obj_set_pos(keyboard_special, 163, 96);
	lv_obj_set_size(keyboard_special, 55, 21);
	lv_obj_set_style_bg_color(keyboard_special, lv_color_hex(0x1C1917), 0);
	lv_obj_set_style_bg_opa(keyboard_special, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(keyboard_special, 0, 0);
	lv_obj_set_style_radius(keyboard_special, 8, 0);
	lv_obj_set_style_pad_all(keyboard_special, 0, 0);
	lv_obj_clear_flag(keyboard_special, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(keyboard_special, LV_OBJ_FLAG_CLICKABLE);

	_keyboard_special_label = lv_label_create(keyboard_special);
	_keyboard_special_text = "-";
	lv_label_set_text(_keyboard_special_label,
			  _keyboard_special_text.c_str());
	lv_obj_set_style_text_font(_keyboard_special_label,
				   &lv_font_montserrat_16, 0);
	lv_obj_set_style_text_color(_keyboard_special_label, lv_color_white(),
				    0);
	lv_obj_center(_keyboard_special_label);
	lv_obj_clear_flag(_keyboard_special_label, LV_OBJ_FLAG_CLICKABLE);

	_mouse_indicator_obj = lv_image_create(_content_obj);
	lv_image_set_src(_mouse_indicator_obj, &img_main_mouse_default);
	lv_obj_set_pos(_mouse_indicator_obj, 230, 98);
	lv_obj_clear_flag(_mouse_indicator_obj, LV_OBJ_FLAG_CLICKABLE);
	apply_mouse_indicator_visual();

	lv_obj_t *keyboard_normal = lv_obj_create(_content_obj);
	lv_obj_set_pos(keyboard_normal, 190, 117);
	lv_obj_set_size(keyboard_normal, 30, 40);
	lv_obj_set_style_bg_color(keyboard_normal, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(keyboard_normal, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(keyboard_normal, 0, 0);
	lv_obj_set_style_radius(keyboard_normal, 0, 0);
	lv_obj_set_style_pad_all(keyboard_normal, 0, 0);
	lv_obj_clear_flag(keyboard_normal, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(keyboard_normal, LV_OBJ_FLAG_CLICKABLE);

	_keyboard_normal_label = lv_label_create(keyboard_normal);
	_keyboard_normal_text = "-";
	lv_label_set_text(_keyboard_normal_label,
			  _keyboard_normal_text.c_str());
	lv_obj_set_style_text_font(_keyboard_normal_label,
				   &lv_font_montserrat_20, 0);
	lv_obj_set_style_text_color(_keyboard_normal_label, lv_color_white(),
				    0);
	lv_obj_center(_keyboard_normal_label);
	lv_obj_clear_flag(_keyboard_normal_label, LV_OBJ_FLAG_CLICKABLE);

	_hid_status_obj = lv_obj_create(_content_obj);
	lv_obj_set_pos(_hid_status_obj, 226, 126);
	lv_obj_set_size(_hid_status_obj, 23, 23);
	lv_obj_set_style_radius(_hid_status_obj, LV_RADIUS_CIRCLE, 0);
	lv_obj_set_style_pad_all(_hid_status_obj, 0, 0);
	lv_obj_clear_flag(_hid_status_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_hid_status_obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_set_style_border_width(_hid_status_obj, 0, 0);
	lv_obj_set_style_border_color(_hid_status_obj, lv_color_hex(0xFFB900),
				      0);
	lv_obj_set_style_bg_color(_hid_status_obj, lv_color_hex(0xFFB900), 0);
	lv_obj_set_style_bg_opa(_hid_status_obj, LV_OPA_COVER, 0);

	_display_metric_timer = lv_timer_create(display_metric_timer_cb,
						kDisplayMetricRefreshMs, this);
}

void MainDashboardView::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	if (_mouse_idle_timer != nullptr) {
		lv_timer_delete(_mouse_idle_timer);
		_mouse_idle_timer = nullptr;
	}
	if (_display_metric_timer != nullptr) {
		lv_timer_delete(_display_metric_timer);
		_display_metric_timer = nullptr;
	}
	if (_ip_reconnect_animation_timer != nullptr) {
		lv_timer_delete(_ip_reconnect_animation_timer);
		_ip_reconnect_animation_timer = nullptr;
	}
}

void MainDashboardView::shutdown()
{
	begin_shutdown();
	lv_delete_if_valid(_content_obj);
	_content_obj = nullptr;
	_background_obj = nullptr;
	_ip_label = nullptr;
	_resolution_label = nullptr;
	_resolution_obj = nullptr;
	_fps_value_label = nullptr;
	_fps_value_obj = nullptr;
	_fps_unit_label = nullptr;
	_fps_unit_obj = nullptr;
	_keyboard_special_label = nullptr;
	_keyboard_normal_label = nullptr;
	_video_status_obj = nullptr;
	_mouse_indicator_obj = nullptr;
	_hid_status_obj = nullptr;
}

bool MainDashboardView::is_created() const noexcept
{
	return _content_obj != nullptr;
}

lv_obj_t *MainDashboardView::content_object() const noexcept
{
	return _content_obj;
}

void MainDashboardView::replace_background(const BackgroundImage *background)
{
	if (_content_obj == nullptr || _is_shutting_down) {
		return;
	}
	lv_delete_if_valid(_background_obj);
	_background_obj = create_background_image_obj(_content_obj, background);
}

void MainDashboardView::set_mouse_active(bool active)
{
	_mouse_active = active;
	apply_mouse_indicator_visual();
	if (!active) {
		if (_mouse_idle_timer != nullptr) {
			lv_timer_delete(_mouse_idle_timer);
			_mouse_idle_timer = nullptr;
		}
		return;
	}

	if (_mouse_indicator_obj == nullptr || _is_shutting_down) {
		return;
	}

	if (_mouse_idle_timer == nullptr) {
		_mouse_idle_timer = lv_timer_create(mouse_idle_timer_cb,
						    kMouseIdleTimeoutMs, this);
		lv_timer_set_repeat_count(_mouse_idle_timer, 1);
	} else {
		lv_timer_reset(_mouse_idle_timer);
	}
}

void MainDashboardView::set_resolution_text(const char *text)
{
	const char *value = text != nullptr ? text : "";
	_pending_resolution_text = is_valid_resolution_text(value) ? value :
								     "--";
	_resolution_dirty = true;
}

void MainDashboardView::set_fps_text(const char *text)
{
	const char *value = text != nullptr ? text : "";
	_pending_fps_text = is_valid_fps_text(value) ? value : "--";
	_fps_dirty = true;
}

void MainDashboardView::set_fps_unit_text(const char *text)
{
	if (_fps_unit_label != nullptr) {
		lv_label_set_text(_fps_unit_label, text != nullptr ? text : "");
	}
}

void MainDashboardView::set_keyboard_special_text(const char *text)
{
	_keyboard_special_text = text != nullptr ? text : "";
	_hid_data_available = has_hid_label_data(_keyboard_special_text,
						 _keyboard_normal_text);
	render_keyboard_texts();
}

void MainDashboardView::set_keyboard_normal_text(const char *text)
{
	_keyboard_normal_text = text != nullptr ? text : "";
	_hid_data_available = has_hid_label_data(_keyboard_special_text,
						 _keyboard_normal_text);
	render_keyboard_texts();
}

void MainDashboardView::set_ip_empty_placeholder(std::string_view text)
{
	const std::string next = text.empty() ? std::string("--") :
						std::string(text);
	if (next == _ip_empty_placeholder) {
		return;
	}

	_ip_empty_placeholder = next;
	if (_ip_candidates.empty()) {
		render_ip_candidate();
	}
}

void MainDashboardView::set_ip_candidates(
	const std::vector<std::string> &candidates)
{
	std::vector<std::string> next;
	for (const auto &candidate : candidates) {
		if (!candidate.empty() && std::find(next.begin(), next.end(),
						    candidate) == next.end()) {
			next.push_back(candidate);
		}
	}

	if (next == _ip_candidates) {
		return;
	}

	_ip_candidates = std::move(next);
	render_ip_candidate();
}

void MainDashboardView::flush_display_metrics()
{
	if (_resolution_dirty) {
		_resolution_text = _pending_resolution_text;
		_resolution_dirty = false;
		render_resolution_text();
	}

	if (_fps_dirty) {
		_fps_text = _pending_fps_text;
		_fps_dirty = false;
		render_fps_text();
	}
}

void MainDashboardView::render_resolution_text()
{
	set_resolution_label_text(_resolution_text.c_str());
}

void MainDashboardView::render_fps_text()
{
	set_fps_label_text(_fps_text.c_str());
}

void MainDashboardView::render_keyboard_texts()
{
	set_keyboard_special_label_text(
		_hid_data_available ? _keyboard_special_text.c_str() : "-");
	set_keyboard_normal_label_text(
		_hid_data_available ? _keyboard_normal_text.c_str() : "-");
}

void MainDashboardView::render_ip_candidate()
{
	if (_ip_label == nullptr) {
		return;
	}
	if (_ip_candidates.empty()) {
		const std::string_view placeholder =
			_ip_empty_placeholder.empty() ?
				std::string_view("--") :
				std::string_view(_ip_empty_placeholder);
		const bool reconnecting = placeholder ==
						  kIpReconnectPlaceholderBase ||
					  placeholder == "Reconnecting...";
		update_ip_font(reconnecting);
		if (reconnecting) {
			start_ip_reconnect_animation();
		} else {
			stop_ip_reconnect_animation();
			lv_label_set_text(_ip_label, placeholder.data());
			lv_obj_center(_ip_label);
		}
		return;
	}
	stop_ip_reconnect_animation();
	update_ip_font(false);
	lv_label_set_text(_ip_label, _ip_candidates.front().c_str());
	lv_obj_center(_ip_label);
}

void MainDashboardView::update_ip_font(bool reconnecting)
{
	if (_ip_label == nullptr) {
		return;
	}
	lv_obj_set_style_text_font(
		_ip_label,
		reconnecting ? &MontserratBold_24 : &MontserratBold_26, 0);
}

void MainDashboardView::start_ip_reconnect_animation()
{
	if (_ip_label == nullptr) {
		return;
	}
	if (!_ip_reconnect_animation_active) {
		_ip_reconnect_dot_count = 3;
		_ip_reconnect_dots_increasing = false;
	}
	_ip_reconnect_animation_active = true;
	render_ip_reconnect_animation_frame();
	if (_ip_reconnect_animation_timer != nullptr) {
		lv_timer_reset(_ip_reconnect_animation_timer);
		return;
	}
	_ip_reconnect_animation_timer =
		lv_timer_create(ip_reconnect_animation_timer_cb,
				kIpReconnectAnimationPeriodMs, this);
}

void MainDashboardView::stop_ip_reconnect_animation()
{
	_ip_reconnect_animation_active = false;
	_ip_reconnect_dot_count = 3;
	_ip_reconnect_dots_increasing = false;
	if (_ip_reconnect_animation_timer != nullptr) {
		lv_timer_delete(_ip_reconnect_animation_timer);
		_ip_reconnect_animation_timer = nullptr;
	}
}

void MainDashboardView::render_ip_reconnect_animation_frame()
{
	if (_ip_label == nullptr) {
		return;
	}
	std::string text(kIpReconnectPlaceholderBase);
	text.append(_ip_reconnect_dot_count, '.');
	lv_label_set_text(_ip_label, text.c_str());
	lv_obj_center(_ip_label);
}

void MainDashboardView::apply_mouse_indicator_visual()
{
	if (_mouse_indicator_obj == nullptr) {
		return;
	}
	lv_image_set_src(_mouse_indicator_obj, _mouse_active ?
						       &img_main_mouse_active :
						       &img_main_mouse_default);
}

void MainDashboardView::set_resolution_label_text(const char *text)
{
	if (_resolution_label != nullptr) {
		lv_label_set_text(_resolution_label,
				  text != nullptr ? text : "");
		lv_obj_center(_resolution_label);
	}
}

void MainDashboardView::set_fps_label_text(const char *text)
{
	if (_fps_value_label != nullptr) {
		lv_label_set_text(_fps_value_label,
				  text != nullptr ? text : "");
		lv_obj_center(_fps_value_label);
	}
}

void MainDashboardView::set_keyboard_special_label_text(const char *text)
{
	if (_keyboard_special_label != nullptr) {
		lv_label_set_text(_keyboard_special_label,
				  text != nullptr ? text : "");
	}
}

void MainDashboardView::set_keyboard_normal_label_text(const char *text)
{
	if (_keyboard_normal_label != nullptr) {
		lv_label_set_text(_keyboard_normal_label,
				  text != nullptr ? text : "");
	}
}

void MainDashboardView::mouse_idle_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}
	auto *self =
		static_cast<MainDashboardView *>(lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}

	self->_mouse_idle_timer = nullptr;
	self->_mouse_active = false;
	self->apply_mouse_indicator_visual();
}

void MainDashboardView::display_metric_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}
	auto *self =
		static_cast<MainDashboardView *>(lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}
	self->flush_display_metrics();
}

void MainDashboardView::ip_reconnect_animation_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}
	auto *self =
		static_cast<MainDashboardView *>(lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down ||
	    self->_ip_reconnect_animation_timer != timer ||
	    !self->_ip_reconnect_animation_active) {
		return;
	}

	if (self->_ip_reconnect_dots_increasing) {
		if (self->_ip_reconnect_dot_count >= 3) {
			self->_ip_reconnect_dots_increasing = false;
			--self->_ip_reconnect_dot_count;
		} else {
			++self->_ip_reconnect_dot_count;
		}
	} else {
		if (self->_ip_reconnect_dot_count == 0) {
			self->_ip_reconnect_dots_increasing = true;
			++self->_ip_reconnect_dot_count;
		} else {
			--self->_ip_reconnect_dot_count;
		}
	}

	self->render_ip_reconnect_animation_frame();
}

} // namespace ui
