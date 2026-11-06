#include "ui/settings_state_toggle_page.hpp"

#include <memory>

namespace ui
{

SettingsStateTogglePage::SettingsStateTogglePage(const char *title,
						 uint32_t active_color,
						 const lv_image_dsc_t *icon)
	: SubPageBase(PageType::SUBPAGE_SETTINGS_STATE_TOGGLE, title)
	, _active_color(active_color)
	, _icon(icon)
{
}

void SettingsStateTogglePage::destroy()
{
	on_enabled_change = nullptr;
	SubPageBase::destroy();
}

void SettingsStateTogglePage::set_enabled(bool enabled)
{
	_enabled = enabled;
	if (!_pending) {
		_pending_target = enabled;
	}
	update_toggle();
}

void SettingsStateTogglePage::set_pending_state(bool pending,
						bool pending_target,
						uint32_t pending_timeout_ms)
{
	_pending = pending;
	_pending_target = pending_target;
	update_toggle();
	if (_toggle_btn != nullptr) {
		_toggle_btn->set_pending_timeout(pending_timeout_ms);
		_toggle_btn->set_pending(pending);
	}
}

ControlState *SettingsStateTogglePage::toggle_control() const
{
	return _toggle_btn;
}

void SettingsStateTogglePage::build_content()
{
	auto toggle =
		std::make_unique<BConA>(_scroll_content, "OFF", _active_color);
	_toggle_btn = toggle.get();
	toggle->create();
	toggle->set_left_image(_icon);
	toggle->on_click([this](BConA *btn) {
		if (_pending || (btn != nullptr && btn->is_pending())) {
			return;
		}
		if (on_enabled_change) {
			on_enabled_change(!_enabled);
		}
	});
	_components.push_back(std::move(toggle));

	update_toggle();
}

void SettingsStateTogglePage::update_toggle()
{
	if (_toggle_btn == nullptr) {
		return;
	}

	if (_pending) {
		_toggle_btn->set_text("...");
		_toggle_btn->set_active(_pending_target,
					BConA::ApplyPolicy::IMMEDIATE);
		return;
	}

	_toggle_btn->set_text(_enabled ? "ON" : "OFF");
	_toggle_btn->set_active(_enabled, BConA::ApplyPolicy::IMMEDIATE);
}

void SettingsStateTogglePage::on_subpage_destroy()
{
	on_enabled_change = nullptr;
	_toggle_btn = nullptr;
}

} // namespace ui
