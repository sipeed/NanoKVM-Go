#include "ui/panel_page.hpp"

#include <algorithm>

#include <spdlog/spdlog.h>

extern "C" {
LV_IMAGE_DECLARE(img_settings_main_subpanel_sleep);
LV_IMAGE_DECLARE(img_settings_main_subpanel_bn);
}

namespace ui
{

PanelPage::PanelPage()
	: SubPageBase(PageType::SUBPAGE_PANEL, "Panel")
{
}

void PanelPage::set_brightness_percent(int percent)
{
	_brightness_percent = std::clamp(percent, BRIGHTNESS_MIN_PERCENT,
					 BRIGHTNESS_MAX_PERCENT);
	update_brightness_state();
	if (_interactive_slider != nullptr &&
	    !_interactive_slider->is_interacting() &&
	    _interactive_slider->get_value() == _brightness_percent) {
		_interactive_slider->clear_interaction_guard();
	}
}

bool PanelPage::can_apply_runtime_brightness_percent(int percent) const
{
	const int clamped = std::clamp(percent, BRIGHTNESS_MIN_PERCENT,
				       BRIGHTNESS_MAX_PERCENT);
	if (_interactive_slider == nullptr || clamped == _brightness_percent) {
		return true;
	}
	return !_interactive_slider->is_interaction_guard_active();
}

void PanelPage::set_sleep_enabled(bool enabled)
{
	_sleep_enabled = enabled;
	finish_sleep_pending_if_target(enabled);
	update_sleep_state();
}

void PanelPage::build_content()
{
	auto interactive_slider =
		std::make_unique<BConC>(_scroll_content, SLIDER_COLOR);
	_interactive_slider = interactive_slider.get();
	interactive_slider->create();
	interactive_slider->set_left_image(&img_settings_main_subpanel_bn);
	interactive_slider->set_range(BRIGHTNESS_MIN_PERCENT,
				      BRIGHTNESS_MAX_PERCENT);
	interactive_slider->set_value(_brightness_percent);
	interactive_slider->on_value_changed([this](int value) {
		_brightness_percent = value;
		if (!_suppress_brightness_callback && on_brightness_change) {
			on_brightness_change(_brightness_percent);
		}
	});
	_components.push_back(std::move(interactive_slider));

	auto sleep =
		std::make_unique<BConA>(_scroll_content, "Sleep", SLEEP_COLOR);
	_sleep_btn = sleep.get();
	sleep->create();
	sleep->set_left_image(&img_settings_main_subpanel_sleep);
	sleep->on_click([this](BConA *btn) {
		if (_sleep_pending && btn->is_pending()) {
			return;
		}
		if (!on_sleep_enabled_change) {
			spdlog::warn(
				"Panel Sleep clicked without runtime callback; ignoring");
			return;
		}
		const bool requested_enabled = !_sleep_enabled;
		_sleep_pending = true;
		_sleep_pending_target = requested_enabled;
		btn->set_pending(true);
		spdlog::info("[panel] Panel Sleep clicked");
		on_sleep_enabled_change(requested_enabled);
	});
	_components.push_back(std::move(sleep));

	update_brightness_state();
	update_sleep_state();
}

void PanelPage::set_sleep_control_available(bool available)
{
	_sleep_control_available = available;
	if (_sleep_btn == nullptr) {
		return;
	}

	lv_obj_t *obj = _sleep_btn->get_obj();
	if (obj == nullptr) {
		return;
	}

	if (available) {
		lv_obj_clear_flag(obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_add_flag(obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_set_style_opa(obj, LV_OPA_100, LV_PART_MAIN);
	} else {
		lv_obj_clear_flag(obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_set_style_opa(obj, LV_OPA_50, LV_PART_MAIN);
		_sleep_btn->set_pending(false);
	}
}

void PanelPage::destroy()
{
	if (_interactive_slider != nullptr) {
		_interactive_slider->on_value_changed(nullptr);
	}
	on_brightness_change = nullptr;
	on_sleep_enabled_change = nullptr;
	SubPageBase::destroy();
}

void PanelPage::update_brightness_state()
{
	if (_interactive_slider == nullptr) {
		return;
	}
	_suppress_brightness_callback = true;
	_interactive_slider->set_value(_brightness_percent);
	_suppress_brightness_callback = false;
}

void PanelPage::update_sleep_state()
{
	if (_sleep_btn != nullptr) {
		_sleep_btn->set_active(_sleep_enabled,
				       BConA::ApplyPolicy::IMMEDIATE);
	}
}

void PanelPage::finish_sleep_pending_if_target(bool enabled)
{
	if (!_sleep_pending || _sleep_pending_target == enabled) {
		_sleep_pending = false;
		if (_sleep_btn != nullptr) {
			_sleep_btn->set_pending(false);
		}
		return;
	}

	if (_sleep_btn != nullptr && !_sleep_btn->is_pending()) {
		_sleep_pending = false;
	}
}

void PanelPage::on_subpage_destroy()
{
	on_brightness_change = nullptr;
	on_sleep_enabled_change = nullptr;
	_interactive_slider = nullptr;
	_sleep_btn = nullptr;
}

} // namespace ui
