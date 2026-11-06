#include "ui/servo_page.hpp"

#include <algorithm>
#include <cmath>
#include <utility>

#include <spdlog/spdlog.h>

namespace ui
{

ServoPage::ServoPage()
	: ServoPage(servo_pwm::make_default_backend())
{
}

ServoPage::ServoPage(std::unique_ptr<servo_pwm::Backend> pwm_backend)
	: SubPageBase(PageType::SUBPAGE_SERVO, "Servo")
	, _pwm_backend(std::move(pwm_backend))
{
	if (_pwm_backend == nullptr) {
		_pwm_backend = servo_pwm::make_default_backend();
	}
}

void ServoPage::build_content()
{
	_slider_row = lv_obj_create(_scroll_content);
	lv_obj_set_size(_slider_row, BConA::WIDTH, BConA::HEIGHT);
	lv_obj_set_style_bg_color(_slider_row,
				  lv_color_hex(BConA::COLOR_DEFAULT), 0);
	lv_obj_set_style_bg_opa(_slider_row, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_slider_row, BConA::RADIUS, 0);
	lv_obj_set_style_border_width(_slider_row, 0, 0);
	lv_obj_set_style_pad_all(_slider_row, 0, 0);
	lv_obj_set_style_clip_corner(_slider_row, true, 0);
	lv_obj_clear_flag(_slider_row, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_slider_row, LV_OBJ_FLAG_CLICKABLE);

	_slider_track = lv_obj_create(_slider_row);
	lv_obj_set_size(_slider_track, SLIDER_WIDTH, SLIDER_HEIGHT);
	lv_obj_set_pos(_slider_track, 0, 0);
	lv_obj_set_style_bg_color(_slider_track, lv_color_hex(0x1C1917), 0);
	lv_obj_set_style_bg_opa(_slider_track, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_slider_track, SLIDER_RADIUS, 0);
	lv_obj_set_style_border_width(_slider_track, 0, 0);
	lv_obj_set_style_pad_all(_slider_track, 0, 0);
	lv_obj_set_style_clip_corner(_slider_track, true, 0);
	lv_obj_clear_flag(_slider_track, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_slider_track, LV_OBJ_FLAG_CLICKABLE);

	_slider_fill = lv_obj_create(_slider_track);
	lv_obj_set_size(_slider_fill, 0, SLIDER_HEIGHT);
	lv_obj_set_pos(_slider_fill, 0, 0);
	lv_obj_set_style_bg_color(_slider_fill, lv_color_hex(SLIDER_COLOR), 0);
	lv_obj_set_style_bg_opa(_slider_fill, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_slider_fill, SLIDER_RADIUS, 0);
	lv_obj_set_style_border_width(_slider_fill, 0, 0);
	lv_obj_set_style_pad_all(_slider_fill, 0, 0);
	lv_obj_clear_flag(_slider_fill, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_slider_fill, LV_OBJ_FLAG_CLICKABLE);

	_slider_knob = lv_obj_create(_slider_row);
	lv_obj_set_size(_slider_knob, KNOB_SIZE, KNOB_SIZE);
	lv_obj_set_style_bg_color(_slider_knob, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(_slider_knob, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_slider_knob, LV_RADIUS_CIRCLE, 0);
	lv_obj_set_style_border_width(_slider_knob, 0, 0);
	lv_obj_set_style_pad_all(_slider_knob, 0, 0);
	lv_obj_clear_flag(_slider_knob, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_slider_knob, LV_OBJ_FLAG_CLICKABLE);

	lv_obj_add_event_cb(_slider_row, slider_event_cb, LV_EVENT_PRESSED,
			    this);
	lv_obj_add_event_cb(_slider_row, slider_event_cb, LV_EVENT_PRESSING,
			    this);

	_button_row = lv_obj_create(_scroll_content);
	lv_obj_set_size(_button_row, BConA::WIDTH, BConB::HEIGHT);
	lv_obj_set_style_bg_opa(_button_row, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_button_row, 0, 0);
	lv_obj_set_style_pad_all(_button_row, 0, 0);
	lv_obj_clear_flag(_button_row, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_button_row, LV_OBJ_FLAG_CLICKABLE);

	auto free = std::make_unique<BConB>(_button_row, FREE_COLOR);
	_free_btn = free.get();
	free->create();
	free->set_position(0, 0);
	free->set_symbol("Free");
	free->on_click([this](BConB *) {
		_free_mode = !_free_mode;
		if (_free_mode) {
			stop_click_sequence();
			apply_free_idle();
		}
		update_button_state();
	});
	_components.push_back(std::move(free));

	auto click = std::make_unique<BConB>(_button_row, CLICK_COLOR);
	_click_btn = click.get();
	click->create();
	click->set_position(BConA::WIDTH - BConB::WIDTH, 0);
	click->set_symbol(servo_pwm::format_idle_compact_ms(_zero_ns).c_str());
	click->set_symbol_font(&lv_font_montserrat_20);
	if (lv_obj_t *obj = click->get_obj()) {
		lv_obj_add_event_cb(obj, click_long_press_event_cb,
				    LV_EVENT_LONG_PRESSED, this);
	}
	click->on_click([this](BConB *) {
		if (_suppress_next_click) {
			_suppress_next_click = false;
			return;
		}
		start_click_sequence();
	});
	_components.push_back(std::move(click));

	update_slider_visual();
	update_title();
	update_button_state();
}

void ServoPage::destroy()
{
	interrupt_click_sequence();
	SubPageBase::destroy();
}

void ServoPage::on_pause()
{
	interrupt_click_sequence();
}

void ServoPage::on_exit()
{
	interrupt_click_sequence();
}

void ServoPage::on_subpage_destroy()
{
	_slider_row = nullptr;
	_slider_track = nullptr;
	_slider_fill = nullptr;
	_slider_knob = nullptr;
	_button_row = nullptr;
	_free_btn = nullptr;
	_click_btn = nullptr;
}

void ServoPage::update_slider_visual()
{
	if (_slider_fill == nullptr || _slider_knob == nullptr) {
		return;
	}

	const int range = servo_pwm::kIdleMaxNs - servo_pwm::kIdleMinNs;
	const int value = _idle_ns - servo_pwm::kIdleMinNs;
	const int32_t fill_width = std::clamp<int32_t>(
		static_cast<int32_t>(
			std::lround(static_cast<double>(value) * SLIDER_WIDTH /
				    static_cast<double>(range))),
		0, SLIDER_WIDTH);
	lv_obj_set_size(_slider_fill, fill_width, SLIDER_HEIGHT);

	const int32_t knob_x = std::clamp<int32_t>(fill_width - KNOB_SIZE / 2,
						   0, SLIDER_WIDTH - KNOB_SIZE);
	lv_obj_set_pos(_slider_knob, knob_x, (SLIDER_HEIGHT - KNOB_SIZE) / 2);
}

void ServoPage::update_title()
{
	set_title(servo_pwm::format_idle_ms(_idle_ns).c_str());
}

void ServoPage::update_zero_label()
{
	if (_click_btn != nullptr) {
		_click_btn->set_symbol(
			servo_pwm::format_idle_compact_ms(_zero_ns).c_str());
		_click_btn->set_symbol_font(&lv_font_montserrat_20);
	}
}

void ServoPage::update_button_state()
{
	if (_free_btn != nullptr) {
		_free_btn->set_active(_free_mode,
				      ControlState::ApplyPolicy::IMMEDIATE);
	}
	if (_click_btn != nullptr) {
		_click_btn->set_active(_click_timer != nullptr,
				       ControlState::ApplyPolicy::IMMEDIATE);
	}
}

int ServoPage::idle_from_local_x(int32_t local_x) const
{
	const int32_t clamped =
		std::clamp<int32_t>(local_x, 0, SLIDER_WIDTH - 1);
	const int32_t range = servo_pwm::kIdleMaxNs - servo_pwm::kIdleMinNs;
	const int32_t span = SLIDER_WIDTH - 1;
	return servo_pwm::clamp_idle_ns(
		servo_pwm::kIdleMinNs +
		static_cast<int>((clamped * range + span / 2) / span));
}

void ServoPage::set_idle_ns(int idle_ns, bool apply_free_mode)
{
	_idle_ns = servo_pwm::clamp_idle_ns(idle_ns);
	update_slider_visual();
	update_title();
	if (apply_free_mode && _free_mode) {
		apply_free_idle();
	}
}

void ServoPage::set_zero_from_current_idle()
{
	_zero_ns = _idle_ns;
	update_zero_label();
	spdlog::info("[servo] servo zero PWM_IDLE_NS={}", _zero_ns);
}

void ServoPage::apply_free_idle()
{
	servo_pwm::ApplyResult result;
	if (!_initialized_pwm) {
		result = _pwm_backend->initialize(_idle_ns);
		_initialized_pwm = result.ok;
	} else {
		result = _pwm_backend->write_idle(_idle_ns);
	}
	log_pwm_result("servo free", result);
}

void ServoPage::start_click_sequence()
{
	stop_click_sequence();
	_free_mode = false;
	_click_plan = servo_pwm::build_click_mode_plan(_idle_ns, _zero_ns);
	_click_step_index = 0;
	if (!_initialized_pwm) {
		const auto result =
			_pwm_backend->initialize(_click_plan.front().duty_ns);
		_initialized_pwm = result.ok;
		log_pwm_result("servo click init", result);
		if (!result.ok) {
			_click_plan.clear();
			update_button_state();
			return;
		}
	}
	run_click_step();
	update_button_state();
}

void ServoPage::stop_click_sequence()
{
	if (_click_timer != nullptr) {
		lv_timer_delete(_click_timer);
		_click_timer = nullptr;
	}
	_click_plan.clear();
	_click_step_index = 0;
	update_button_state();
}

bool ServoPage::has_active_click_sequence() const
{
	return _click_timer != nullptr || !_click_plan.empty();
}

void ServoPage::interrupt_click_sequence()
{
	const bool was_active = has_active_click_sequence();
	stop_click_sequence();
	if (!was_active || !_initialized_pwm) {
		return;
	}

	const auto result = _pwm_backend->write_idle(_zero_ns);
	if (result.ok) {
		spdlog::info("[servo] servo click interrupted PWM_IDLE_NS={}",
			     _zero_ns);
		return;
	}
	spdlog::warn("servo click interruption zero failed PWM_IDLE_NS={}: {}",
		     _zero_ns, result.error);
}

void ServoPage::run_click_step()
{
	if (_click_step_index >= _click_plan.size()) {
		stop_click_sequence();
		return;
	}

	const servo_pwm::Step step = _click_plan[_click_step_index++];
	const auto result = _pwm_backend->write_idle(step.duty_ns);
	log_pwm_result("servo click", result);
	if (!result.ok || _click_step_index >= _click_plan.size()) {
		stop_click_sequence();
		return;
	}

	_click_timer =
		lv_timer_create(click_timer_cb, step.delay_after_ms, this);
	if (_click_timer != nullptr) {
		lv_timer_set_repeat_count(_click_timer, 1);
	}
	update_button_state();
}

void ServoPage::log_pwm_result(const char *action,
			       const servo_pwm::ApplyResult &result)
{
	if (result.ok) {
		spdlog::info("[servo] {} PWM_IDLE_NS={}", action, _idle_ns);
		return;
	}
	spdlog::warn("{} failed: {}", action, result.error);
}

void ServoPage::slider_event_cb(lv_event_t *event)
{
	auto *self = static_cast<ServoPage *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_slider_row == nullptr) {
		return;
	}

	lv_indev_t *indev = lv_indev_get_act();
	if (indev == nullptr) {
		return;
	}

	lv_point_t point;
	lv_indev_get_point(indev, &point);
	lv_area_t coords;
	lv_obj_get_coords(self->_slider_row, &coords);
	self->set_idle_ns(self->idle_from_local_x(point.x - coords.x1), true);
}

void ServoPage::click_long_press_event_cb(lv_event_t *event)
{
	auto *self = static_cast<ServoPage *>(lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}
	self->_suppress_next_click = true;
	self->set_zero_from_current_idle();
}

void ServoPage::click_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<ServoPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}
	self->_click_timer = nullptr;
	self->run_click_step();
}

} // namespace ui
