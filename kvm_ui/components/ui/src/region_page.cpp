#include "ui/region_page.hpp"

#include <string>

#include "ui/hold_confirm_behavior.hpp"

extern "C" {
LV_IMAGE_DECLARE(settings_version_switch);
}

namespace ui
{

RegionPage::RegionPage()
	: SubPageBase(PageType::SUBPAGE_REGION, "Region", 0)
{
}

RegionPage::~RegionPage() = default;

void RegionPage::set_region(RegionOption region)
{
	_projected_region = region;
	if (_switch_pending) {
		return;
	}
	apply_region_value(region);
}

void RegionPage::set_switch_pending(bool pending)
{
	if (_switch_pending == pending) {
		return;
	}
	_switch_pending = pending;

	if (_switch_hold_behavior != nullptr) {
		_switch_hold_behavior->cancel();
	}

	if (_switch_btn != nullptr) {
		lv_obj_t *switch_obj = _switch_btn->get_obj();
		if (pending) {
			_switch_btn->set_pending_timeout_enabled(false);
			_switch_btn->set_pending(true);
			if (switch_obj != nullptr &&
			    lv_obj_is_valid(switch_obj)) {
				lv_obj_clear_flag(switch_obj,
						  LV_OBJ_FLAG_CLICKABLE);
			}
		} else {
			_switch_btn->set_pending(false);
			_switch_btn->set_pending_timeout_enabled(true);
			if (switch_obj != nullptr &&
			    lv_obj_is_valid(switch_obj) &&
			    _switch_hold_behavior != nullptr &&
			    _switch_hold_behavior->is_ready()) {
				lv_obj_add_flag(switch_obj,
						LV_OBJ_FLAG_CLICKABLE);
			}
		}
	}

	set_back_navigation_available(!pending);

	if (pending) {
		apply_region_value(_pending_target);
		return;
	}
	apply_region_value(_projected_region);
}

void RegionPage::build_content()
{
	constexpr int32_t title_top_margin = 12;
	constexpr int32_t title_bottom =
		title_top_margin + static_cast<int32_t>(BConA::HEIGHT);
	constexpr int32_t switch_top =
		HEIGHT - title_top_margin - BConA::HEIGHT;
	constexpr int32_t region_top =
		title_bottom +
		(switch_top - title_bottom -
		 static_cast<int32_t>(SConA::HEIGHT)) /
			2;
	constexpr int32_t region_top_margin = region_top - title_bottom;
	constexpr int32_t switch_top_margin =
		switch_top - region_top - SConA::HEIGHT;

	const std::string label(region_option_label(_projected_region));
	auto region_value = std::make_unique<SConA>(
		_scroll_content, label.c_str(), SConA::TextAlign::CENTER);
	_region_value = region_value.get();
	region_value->create();
	lv_obj_set_style_margin_top(region_value->get_obj(),
				    region_top_margin, 0);
	_components.push_back(std::move(region_value));

	auto switch_btn = std::make_unique<BConA>(_scroll_content, "Switch",
						  BConA::COLOR_DEFAULT);
	_switch_btn = switch_btn.get();
	switch_btn->create();
	lv_obj_set_style_margin_top(switch_btn->get_obj(), switch_top_margin, 0);
	switch_btn->set_left_image(&settings_version_switch);
	_components.push_back(std::move(switch_btn));

	_switch_hold_behavior = std::make_unique<HoldConfirmBehavior>(
		*_switch_btn, HoldConfirmConfig {
				      .duration_ms = SWITCH_HOLD_DURATION_MS,
				      .fill_color = SWITCH_FILL_COLOR,
				      .idle_text = "Switch",
				      .holding_text = "Keep",
			      });
	_switch_hold_behavior->on_confirm([this]() {
		if (_switch_pending || !on_switch_request) {
			return;
		}
		const RegionOption target = region_other(_projected_region);
		if (!on_switch_request(target)) {
			return;
		}
		_pending_target = target;
		set_switch_pending(true);
	});
}

void RegionPage::destroy()
{
	set_switch_pending(false);
	_switch_hold_behavior.reset();
	on_switch_request = nullptr;
	SubPageBase::destroy();
}

void RegionPage::on_pause()
{
	if (_switch_hold_behavior != nullptr) {
		_switch_hold_behavior->cancel();
	}
}

void RegionPage::on_exit()
{
	if (_switch_hold_behavior != nullptr) {
		_switch_hold_behavior->cancel();
	}
}

void RegionPage::apply_region_value(RegionOption region)
{
	if (_region_value == nullptr) {
		return;
	}
	const std::string label(region_option_label(region));
	_region_value->set_text(label.c_str());
}

void RegionPage::on_subpage_destroy()
{
	_region_value = nullptr;
	_switch_btn = nullptr;
	_switch_pending = false;
	on_switch_request = nullptr;
}

} // namespace ui
