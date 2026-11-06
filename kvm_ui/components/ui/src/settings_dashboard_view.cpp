#include "ui/settings_dashboard_view.hpp"

#include "lv_safe.hpp"
#include "ui/background_assets.hpp"
#include "ui/control_state.hpp"
#include "ui/settings_page.hpp"
#include "ui/settings_toggle_coordinator.hpp"

namespace ui
{

SettingsDashboardView::SettingsDashboardView(
	bool experimental_settings_subpages,
	uint32_t settings_state_boot_timeout_ms)
	: _experimental_settings_subpages(experimental_settings_subpages)
	, _settings_state_boot_timeout_ms(settings_state_boot_timeout_ms)
{
}

SettingsDashboardView::~SettingsDashboardView()
{
	shutdown();
}

void SettingsDashboardView::create(lv_obj_t *scroll_parent,
				   const BackgroundImage *background,
				   uint32_t wifi_pending_timeout_ms)
{
	if (_is_shutting_down || scroll_parent == nullptr ||
	    _settings_page != nullptr) {
		return;
	}

	_settings_page = std::make_unique<SettingsPage>();
	_settings_page->set_background_image(background);
	_settings_page->create(scroll_parent);
	if (lv_obj_t *settings_page_obj = _settings_page->get_container();
	    settings_page_obj != nullptr) {
		lv_obj_set_pos(settings_page_obj,
			       root_layout::content_page_x(
				       root_layout::ContentPage::SettingsHome),
			       0);
	}
	install_tile_callbacks();

	_settings_second_page_obj = lv_obj_create(scroll_parent);
	lv_obj_set_pos(_settings_second_page_obj,
		       root_layout::content_page_x(
			       root_layout::ContentPage::SettingsSecond),
		       0);
	lv_obj_set_size(_settings_second_page_obj,
			root_layout::kContentCardWidth,
			root_layout::kContentCardHeight);
	lv_obj_set_style_radius(_settings_second_page_obj,
				root_layout::kContentCardRadius, 0);
	lv_obj_set_style_bg_color(_settings_second_page_obj,
				  lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_settings_second_page_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_settings_second_page_obj, 0, 0);
	lv_obj_set_style_pad_all(_settings_second_page_obj, 0, 0);
	lv_obj_set_style_clip_corner(_settings_second_page_obj, true, 0);
	lv_obj_clear_flag(_settings_second_page_obj, LV_OBJ_FLAG_SCROLLABLE);
	_settings_second_background_obj = create_background_image_obj(
		_settings_second_page_obj, background);
	_settings_page->create_second_page_tiles(_settings_second_page_obj);
	_settings_page->set_tile_pending_timeout(
		SettingsPage::SettingsTile::WIFI, wifi_pending_timeout_ms);
	apply_wifi_tile_state();
	apply_screen_tile_state();
	apply_tailscale_tile_state();
}

void SettingsDashboardView::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	clear_tile_callbacks();
	on_wifi_click = nullptr;
	on_mcp_click = nullptr;
	on_replay_click = nullptr;
	on_ncm_click = nullptr;
	on_ssh_click = nullptr;
	on_auto_rotation_click = nullptr;
	on_screen_click = nullptr;
	on_servo_click = nullptr;
	on_second_about_click = nullptr;
	on_apps_click = nullptr;
	on_tailscale_click = nullptr;
	on_network_click = nullptr;
	on_region_click = nullptr;
}

void SettingsDashboardView::shutdown()
{
	begin_shutdown();
	if (_settings_page) {
		_settings_page->destroy();
		_settings_page.reset();
	}
	lv_delete_if_valid(_settings_second_page_obj);
	_settings_second_page_obj = nullptr;
	_settings_second_background_obj = nullptr;
}

SettingsPage *SettingsDashboardView::settings_page() const noexcept
{
	return _settings_page.get();
}

lv_obj_t *SettingsDashboardView::second_page_object() const noexcept
{
	return _settings_second_page_obj;
}

void SettingsDashboardView::replace_background(const BackgroundImage *background)
{
	if (_is_shutting_down) {
		return;
	}
	if (_settings_page) {
		_settings_page->set_background_image(background);
	}
	if (_settings_second_page_obj != nullptr) {
		lv_delete_if_valid(_settings_second_background_obj);
		_settings_second_background_obj = create_background_image_obj(
			_settings_second_page_obj, background);
	}
}

void SettingsDashboardView::set_apps_tile_visible(bool visible)
{
	if (_is_shutting_down || !_settings_page) {
		return;
	}
	_settings_page->set_tile_visible(SettingsPage::SettingsTile::APPS,
					 visible);
}

void SettingsDashboardView::set_tailscale_tile_active(bool active)
{
	_tailscale_tile_active = active;
	apply_tailscale_tile_state();
}

void SettingsDashboardView::set_wifi_tile_active(bool active)
{
	_wifi_tile_active = active;
	apply_wifi_tile_state();
}

void SettingsDashboardView::set_wifi_tile_pending(bool pending)
{
	if (_is_shutting_down || !_settings_page) {
		return;
	}
	if (ControlState *control = _settings_page->get_wifi_btn()) {
		control->set_pending(pending);
	}
	if (!pending) {
		apply_wifi_tile_state();
	}
}

SettingsWifiTilePresentation
SettingsDashboardView::wifi_tile_presentation() const noexcept
{
	const ControlState *const control =
		_settings_page ? _settings_page->get_wifi_btn() : nullptr;
	if (control == nullptr) {
		return {};
	}
	return {
		.created = true,
		.active = control->is_active(),
		.pending = control->is_pending(),
	};
}

void SettingsDashboardView::set_screen_tile_active(bool active)
{
	_screen_tile_active = active;
	apply_screen_tile_state();
}

void SettingsDashboardView::set_toggle_availability(
	SettingsToggleId id, const SettingsToggleViewModel &model)
{
	if (_is_shutting_down || !_settings_page || model.boot_pending) {
		return;
	}

	switch (id) {
	case SettingsToggleId::MCP:
		_settings_page->set_tile_available(
			SettingsPage::SettingsTile::MCP, model.available);
		return;
	case SettingsToggleId::REPLAY:
		_settings_page->set_tile_available(
			SettingsPage::SettingsTile::REPLAY, model.available);
		_settings_page->set_tile_visible(
			SettingsPage::SettingsTile::REPLAY, model.available);
		return;
	case SettingsToggleId::NCM:
	case SettingsToggleId::SSH:
	case SettingsToggleId::AUTO_ROTATION:
		return;
	}
}

void SettingsDashboardView::render_toggle(SettingsToggleId id,
					  const SettingsToggleViewModel &model,
					  uint32_t pending_timeout_ms)
{
	if (_is_shutting_down) {
		return;
	}

	ControlState *control = toggle_tile_control(id);
	if (control == nullptr) {
		return;
	}

	if (_experimental_settings_subpages) {
		apply_tile_state(control, model.confirmed);
		return;
	}

	const uint32_t timeout_ms = model.boot_pending &&
						    !model.action_pending ?
					    _settings_state_boot_timeout_ms :
					    pending_timeout_ms;
	control->set_pending_timeout(timeout_ms);
	control->set_active(model.action_pending ? model.pending_target :
						   model.confirmed,
			    ControlState::ApplyPolicy::IMMEDIATE);
	control->set_pending(model.is_pending());
}

ControlState *
SettingsDashboardView::toggle_tile_control(SettingsToggleId id) const noexcept
{
	if (_settings_page == nullptr) {
		return nullptr;
	}

	switch (id) {
	case SettingsToggleId::MCP:
		return _settings_page->get_mcp_btn();
	case SettingsToggleId::REPLAY:
		return _settings_page->get_replay_btn();
	case SettingsToggleId::NCM:
		return _settings_page->get_ncm_btn();
	case SettingsToggleId::SSH:
		return _settings_page->get_ssh_btn();
	case SettingsToggleId::AUTO_ROTATION:
		return _settings_page->get_auto_rotation_btn();
	}
	return nullptr;
}

void SettingsDashboardView::install_tile_callbacks()
{
	if (_settings_page == nullptr) {
		return;
	}

	_settings_page->on_wifi_click = [this]() {
		if (!_is_shutting_down && on_wifi_click) {
			on_wifi_click();
		}
	};
	_settings_page->on_mcp_click = [this]() {
		if (!_is_shutting_down && on_mcp_click) {
			on_mcp_click();
		}
	};
	_settings_page->on_replay_click = [this]() {
		if (!_is_shutting_down && on_replay_click) {
			on_replay_click();
		}
	};
	_settings_page->on_ncm_click = [this]() {
		if (!_is_shutting_down && on_ncm_click) {
			on_ncm_click();
		}
	};
	_settings_page->on_ssh_click = [this]() {
		if (!_is_shutting_down && on_ssh_click) {
			on_ssh_click();
		}
	};
	_settings_page->on_auto_rotation_click = [this]() {
		if (!_is_shutting_down && on_auto_rotation_click) {
			on_auto_rotation_click();
		}
	};
	_settings_page->on_screen_click = [this]() {
		if (!_is_shutting_down && on_screen_click) {
			on_screen_click();
		}
	};
	_settings_page->on_servo_click = [this]() {
		if (!_is_shutting_down && on_servo_click) {
			on_servo_click();
		}
	};
	_settings_page->on_second_about_click = [this]() {
		if (!_is_shutting_down && on_second_about_click) {
			on_second_about_click();
		}
	};
	_settings_page->on_apps_click = [this]() {
		if (!_is_shutting_down && on_apps_click) {
			on_apps_click();
		}
	};
	_settings_page->on_tailscale_click = [this]() {
		if (!_is_shutting_down && on_tailscale_click) {
			on_tailscale_click();
		}
	};
	_settings_page->on_network_click = [this]() {
		if (!_is_shutting_down && on_network_click) {
			on_network_click();
		}
	};
	_settings_page->on_region_click = [this]() {
		if (!_is_shutting_down && on_region_click) {
			on_region_click();
		}
	};
}

void SettingsDashboardView::clear_tile_callbacks()
{
	if (_settings_page == nullptr) {
		return;
	}

	_settings_page->on_wifi_click = nullptr;
	_settings_page->on_mcp_click = nullptr;
	_settings_page->on_replay_click = nullptr;
	_settings_page->on_ncm_click = nullptr;
	_settings_page->on_ssh_click = nullptr;
	_settings_page->on_auto_rotation_click = nullptr;
	_settings_page->on_screen_click = nullptr;
	_settings_page->on_servo_click = nullptr;
	_settings_page->on_second_about_click = nullptr;
	_settings_page->on_apps_click = nullptr;
	_settings_page->on_tailscale_click = nullptr;
	_settings_page->on_network_click = nullptr;
	_settings_page->on_region_click = nullptr;
}

void SettingsDashboardView::apply_wifi_tile_state()
{
	if (_is_shutting_down || !_settings_page) {
		return;
	}
	apply_tile_state(_settings_page->get_wifi_btn(), _wifi_tile_active);
}

void SettingsDashboardView::apply_screen_tile_state()
{
	if (_is_shutting_down || !_settings_page) {
		return;
	}
	apply_tile_state(_settings_page->get_screen_btn(), _screen_tile_active);
	apply_tile_state(_settings_page->get_second_about_btn(), false);
	apply_tile_state(_settings_page->get_region_btn(), false);
}

void SettingsDashboardView::apply_tailscale_tile_state()
{
	if (_is_shutting_down || !_settings_page) {
		return;
	}
	apply_tile_state(_settings_page->get_tailscale_btn(),
			 _tailscale_tile_active);
}

void SettingsDashboardView::apply_tile_state(ControlState *control, bool active)
{
	if (control != nullptr && !control->is_pending()) {
		control->set_active(active,
				    ControlState::ApplyPolicy::IMMEDIATE);
	}
}

}
