#include "ui/about_page.hpp"

#include "ui/hold_confirm_behavior.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_settings_main_subwifi_reset);
}

namespace ui
{

AboutPage::AboutPage()
	: SubPageBase(PageType::SUBPAGE_ABOUT, "About", 0)
{
}

AboutPage::~AboutPage() = default;

void AboutPage::set_versions(const std::string &app, const std::string &fw,
			     const std::string &hw)
{
	(void)fw;
	_app_version = app;
	_hw_version = hw;
	update_versions();
}

void AboutPage::set_factory_reset_pending(bool pending)
{
	if (_factory_reset_pending == pending) {
		return;
	}
	_factory_reset_pending = pending;

	if (_reset_hold_behavior != nullptr) {
		_reset_hold_behavior->cancel();
	}

	if (_reset_btn != nullptr) {
		lv_obj_t *reset_obj = _reset_btn->get_obj();
		if (pending) {
			_reset_btn->set_pending_timeout_enabled(false);
			_reset_btn->set_pending(true);
			if (reset_obj != nullptr &&
			    lv_obj_is_valid(reset_obj)) {
				lv_obj_clear_flag(reset_obj,
						  LV_OBJ_FLAG_CLICKABLE);
			}
		} else {
			_reset_btn->set_pending(false);
			_reset_btn->set_pending_timeout_enabled(true);
			if (reset_obj != nullptr &&
			    lv_obj_is_valid(reset_obj) &&
			    _reset_hold_behavior != nullptr &&
			    _reset_hold_behavior->is_ready()) {
				lv_obj_add_flag(reset_obj,
						LV_OBJ_FLAG_CLICKABLE);
			}
		}
	}

	set_back_navigation_available(!pending);
}

void AboutPage::build_content()
{
	constexpr int32_t title_top_margin = 12;
	constexpr int32_t version_bottom_margin = 4;
	constexpr int32_t reset_top = HEIGHT - title_top_margin - BConA::HEIGHT;
	constexpr int32_t reset_top_margin = reset_top - title_top_margin -
					     BConA::HEIGHT - 2 * SConA::HEIGHT -
					     version_bottom_margin;

	_version_group = lv_obj_create(_scroll_content);
	lv_obj_set_width(_version_group, SConA::WIDTH);
	lv_obj_set_height(_version_group, LV_SIZE_CONTENT);
	lv_obj_set_style_bg_opa(_version_group, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_version_group, 0, 0);
	lv_obj_set_style_pad_all(_version_group, 0, 0);
	lv_obj_set_flex_flow(_version_group, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_style_pad_row(_version_group, 0, 0);
	lv_obj_clear_flag(_version_group, LV_OBJ_FLAG_SCROLLABLE);

	auto hw_ver = std::make_unique<SConA>(_version_group, "",
					      SConA::TextAlign::LEFT);
	_hw_ver_label = hw_ver.get();
	hw_ver->create();
	_components.push_back(std::move(hw_ver));

	auto app_ver = std::make_unique<SConA>(_version_group, "",
					       SConA::TextAlign::LEFT);
	_app_ver_label = app_ver.get();
	app_ver->create();
	lv_obj_set_style_margin_bottom(app_ver->get_obj(),
				       version_bottom_margin, 0);
	lv_obj_t *app_ver_obj = app_ver->get_obj();
	if (app_ver_obj != nullptr) {
		lv_obj_add_flag(app_ver_obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_set_style_bg_opa(app_ver_obj, LV_OPA_TRANSP,
					LV_STATE_PRESSED);
		lv_obj_add_event_cb(app_ver_obj,
				    &AboutPage::app_ver_click_event_cb,
				    LV_EVENT_CLICKED, this);
	}
	_components.push_back(std::move(app_ver));

	auto reset = std::make_unique<BConA>(_scroll_content, "Reset",
					     BConA::COLOR_DEFAULT);
	_reset_btn = reset.get();
	reset->create();
	lv_obj_set_style_margin_top(reset->get_obj(), reset_top_margin, 0);
	reset->set_left_image(&img_settings_main_subwifi_reset);
	_components.push_back(std::move(reset));

	_reset_hold_behavior = std::make_unique<HoldConfirmBehavior>(
		*_reset_btn, HoldConfirmConfig {
				     .duration_ms = 2000,
				     .fill_color = 0xFB2C36,
				     .idle_text = "Reset",
				     .holding_text = "Keep",
			     });
	_reset_hold_behavior->on_confirm([this]() {
		if (_factory_reset_pending || !on_reset_click) {
			return;
		}
		if (on_reset_click()) {
			set_factory_reset_pending(true);
		}
	});

	update_versions();
}

void AboutPage::destroy()
{
	set_factory_reset_pending(false);
	reset_app_ver_click_state();
	_reset_hold_behavior.reset();
	on_reset_click = nullptr;
	on_app_ver_secret_click = nullptr;
	SubPageBase::destroy();
}

void AboutPage::on_pause()
{
	reset_app_ver_click_state();
	if (_reset_hold_behavior != nullptr) {
		_reset_hold_behavior->cancel();
	}
}

void AboutPage::on_exit()
{
	reset_app_ver_click_state();
	if (_reset_hold_behavior != nullptr) {
		_reset_hold_behavior->cancel();
	}
}

void AboutPage::update_versions()
{
	if (_hw_ver_label != nullptr) {
		_hw_ver_label->set_dual_text("HW Ver:", _hw_version.c_str());
	}
	if (_app_ver_label != nullptr) {
		_app_ver_label->set_dual_text("APP Ver:", _app_version.c_str());
	}
}

void AboutPage::app_ver_click_event_cb(lv_event_t *e)
{
	auto *self = static_cast<AboutPage *>(lv_event_get_user_data(e));
	if (self != nullptr) {
		self->handle_app_ver_click();
	}
}

void AboutPage::handle_app_ver_click()
{
	if (_factory_reset_pending) {
		return;
	}
	const uint32_t elapsed = lv_tick_elaps(_last_app_ver_click_tick);
	_last_app_ver_click_tick = lv_tick_get();
	if (elapsed > SECRET_CLICK_WINDOW_MS) {
		_app_ver_click_count = 1;
		return;
	}
	if (++_app_ver_click_count < SECRET_CLICK_TARGET) {
		return;
	}
	_app_ver_click_count = 0;
	_last_app_ver_click_tick = 0;
	if (on_app_ver_secret_click) {
		on_app_ver_secret_click();
	}
}

void AboutPage::reset_app_ver_click_state()
{
	_app_ver_click_count = 0;
	_last_app_ver_click_tick = 0;
}

void AboutPage::on_subpage_destroy()
{
	_version_group = nullptr;
	_hw_ver_label = nullptr;
	_app_ver_label = nullptr;
	_reset_btn = nullptr;
	_factory_reset_pending = false;
	_app_ver_click_count = 0;
	_last_app_ver_click_tick = 0;
	on_reset_click = nullptr;
	on_app_ver_secret_click = nullptr;
}

} // namespace ui
