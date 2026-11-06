#include "ui/settings_toggle_mediator.hpp"

#include "ui/page_registry.hpp"
#include "ui/settings_dashboard_view.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/settings_route.hpp"
#include "ui/settings_state_action_policy.hpp"
#include "ui/settings_state_route_policy.hpp"
#include "ui/settings_state_toggle_page.hpp"

namespace ui
{

SettingsToggleMediator::SettingsToggleMediator(
	SettingsDashboardView &dashboard, PageRegistry &page_registry,
	SettingsNavigator &settings_navigator,
	const PageManagerActions::Settings &actions,
	uint32_t boot_pending_timeout_ms, bool experimental_settings_subpages,
	bool enable_optimistic_actions)
	: _dashboard(dashboard)
	, _page_registry(page_registry)
	, _settings_navigator(settings_navigator)
	, _actions(actions)
	, _experimental_settings_subpages(experimental_settings_subpages)
	, _enable_optimistic_actions(enable_optimistic_actions)
	, _coordinator(boot_pending_timeout_ms)
{
	_coordinator.set_view_model_change_callback([this](SettingsToggleId) {
		if (_is_shutting_down) {
			return;
		}
		refresh_presentation();
	});
}

SettingsToggleMediator::~SettingsToggleMediator()
{
	shutdown();
}

void SettingsToggleMediator::bind_pages()
{
	if (_is_shutting_down) {
		return;
	}

	_dashboard.on_mcp_click = [this]() {
		route_home_click(SettingsToggleId::MCP);
	};
	_dashboard.on_replay_click = [this]() {
		route_home_click(SettingsToggleId::REPLAY);
	};
	_dashboard.on_ncm_click = [this]() {
		route_home_click(SettingsToggleId::NCM);
	};
	_dashboard.on_ssh_click = [this]() {
		route_home_click(SettingsToggleId::SSH);
	};
	_dashboard.on_auto_rotation_click = [this]() {
		route_home_click(SettingsToggleId::AUTO_ROTATION);
	};

	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		bind_toggle_page(id);
	}
}

void SettingsToggleMediator::start()
{
	if (_is_shutting_down || _started) {
		return;
	}

	_started = true;
	_coordinator.start_boot_pending();
	(void)request_service_refresh();
	render_all();
	start_service_refresh_timer();
}

bool SettingsToggleMediator::request_service_refresh()
{
	if (_is_shutting_down) {
		return false;
	}

	const auto callback = _actions.refresh_service_state;
	if (!callback) {
		return false;
	}
	callback();
	return true;
}

void SettingsToggleMediator::set_confirmed(SettingsToggleId id, bool active)
{
	if (!_is_shutting_down) {
		_coordinator.set_confirmed(id, active);
	}
}

void SettingsToggleMediator::set_auto_rotation_manual_mode(bool manual_mode)
{
	if (!_is_shutting_down) {
		_coordinator.set_auto_rotation_manual_mode(manual_mode);
	}
}

void SettingsToggleMediator::apply_service_runtime_state(
	const SettingsServiceRuntimeState &state)
{
	if (!_is_shutting_down) {
		(void)_coordinator.apply_service_runtime_state(state);
	}
}

void SettingsToggleMediator::refresh_presentation()
{
	if (_is_shutting_down) {
		return;
	}

	sync_availability();
	render_all();
}

void SettingsToggleMediator::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	stop_service_refresh_timer();
	_coordinator.set_view_model_change_callback(nullptr);
	_coordinator.shutdown();
	clear_page_callbacks();
}

void SettingsToggleMediator::shutdown()
{
	begin_shutdown();
}

void SettingsToggleMediator::bind_toggle_page(SettingsToggleId id)
{
	SettingsStateTogglePage *const page =
		_page_registry.settings_toggle_page(id);
	if (page == nullptr) {
		return;
	}

	page->on_back_click = [this]() {
		(void)_settings_navigator.return_to_anchor();
	};
	page->on_enabled_change = [this, id](bool active) {
		(void)request_toggle(id, active);
	};
}

void SettingsToggleMediator::route_home_click(SettingsToggleId id)
{
	using settings_state_route_policy::SettingsStateHomeAction;

	const auto action = settings_state_route_policy::home_action(
		id, _experimental_settings_subpages);
	switch (action) {
	case SettingsStateHomeAction::DirectToggle: {
		if ((is_settings_service_toggle(id) &&
		     !_coordinator.is_available(id))) {
			return;
		}
		const SettingsToggleViewModel model = _coordinator.view_model(id);
		const bool coalescing =
			request_policy(id) ==
			settings_state_action_policy::PendingRequestPolicy::CoalesceLatest;
		bool current_active = model.confirmed;
		if (model.action_pending && coalescing) {
			current_active = model.pending_target;
		}
		(void)request_toggle(id, !current_active);
		return;
	}
	case SettingsStateHomeAction::OpenSubpage:
		show_toggle_subpage(id);
		return;
	case SettingsStateHomeAction::None:
		return;
	}
}

void SettingsToggleMediator::show_toggle_subpage(SettingsToggleId id)
{
	const bool mcp_boot_pending =
		_coordinator.is_boot_pending(SettingsToggleId::MCP);
	const bool replay_boot_pending =
		_coordinator.is_boot_pending(SettingsToggleId::REPLAY);
	if ((id == SettingsToggleId::MCP && !_coordinator.is_available(id) &&
	     !mcp_boot_pending) ||
	    (id == SettingsToggleId::REPLAY && !_coordinator.is_available(id) &&
	     !replay_boot_pending)) {
		return;
	}

	if (_page_registry.settings_toggle_page(id) == nullptr) {
		return;
	}
	render_toggle(id);
	(void)_settings_navigator.show(SettingsRoute::settings_toggle(id));
}

void SettingsToggleMediator::render_all()
{
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		render_toggle(id);
	}
}

void SettingsToggleMediator::render_toggle(SettingsToggleId id)
{
	SettingsToggleViewModel model = _coordinator.view_model(id);
	if (model.action_pending &&
	    settings_state_action_policy::uses_optimistic_action_presentation(
		    id, _enable_optimistic_actions)) {
		model.confirmed = model.pending_target;
		model.action_pending = false;
	}
	if (_experimental_settings_subpages) {
		if (SettingsStateTogglePage *page =
			    _page_registry.settings_toggle_page(id)) {
			page->set_enabled(model.confirmed);
			page->set_pending_state(
				model.is_pending(),
				model.action_pending ? model.pending_target :
						       model.confirmed,
				_coordinator.pending_timeout_ms(id));
		}
	}

	_dashboard.render_toggle(id, model,
				 _coordinator.pending_timeout_ms(id));
}

void SettingsToggleMediator::sync_availability()
{
	const auto mcp = _coordinator.view_model(SettingsToggleId::MCP);
	const auto replay = _coordinator.view_model(SettingsToggleId::REPLAY);

	_dashboard.set_toggle_availability(SettingsToggleId::MCP, mcp);
	_dashboard.set_toggle_availability(SettingsToggleId::REPLAY, replay);
	return_from_unavailable_subpage();
}

void SettingsToggleMediator::return_from_unavailable_subpage()
{
	if (!_experimental_settings_subpages || _settings_navigator.empty()) {
		return;
	}

	const bool mcp_boot_pending =
		_coordinator.is_boot_pending(SettingsToggleId::MCP);
	const bool replay_boot_pending =
		_coordinator.is_boot_pending(SettingsToggleId::REPLAY);
	const auto top_route = _settings_navigator.top_route();
	if (!top_route.has_value()) {
		return;
	}
	const bool mcp_unavailable =
		*top_route ==
			SettingsRoute::settings_toggle(SettingsToggleId::MCP) &&
		!_coordinator.is_available(SettingsToggleId::MCP) &&
		!mcp_boot_pending;
	const bool replay_unavailable =
		*top_route == SettingsRoute::settings_toggle(
				      SettingsToggleId::REPLAY) &&
		!_coordinator.is_available(SettingsToggleId::REPLAY) &&
		!replay_boot_pending;
	if (mcp_unavailable || replay_unavailable) {
		(void)_settings_navigator.return_to_anchor();
	}
}

SettingsToggleCoordinator::ActionCallback
SettingsToggleMediator::settings_action(SettingsToggleId id) const
{
	switch (id) {
	case SettingsToggleId::MCP:
		return _actions.set_mcp_enabled;
	case SettingsToggleId::REPLAY:
		return _actions.set_replay_enabled;
	case SettingsToggleId::NCM:
		return _actions.set_ncm_enabled;
	case SettingsToggleId::SSH:
		return _actions.set_ssh_enabled;
	case SettingsToggleId::AUTO_ROTATION:
		return _actions.set_auto_rotation_enabled;
	}

	return {};
}

SettingsToggleCoordinator::ActionDispatcher
SettingsToggleMediator::settings_action_dispatcher() const
{
	return [this](SettingsToggleId id, bool target_active) {
		const auto callback = settings_action(id);
		if (!callback) {
			return false;
		}
		callback(target_active);
		return true;
	};
}

settings_state_action_policy::PendingRequestPolicy
SettingsToggleMediator::request_policy(SettingsToggleId id) const noexcept
{
	if (_enable_optimistic_actions &&
	    settings_state_action_policy::allows_coalesced_request(id)) {
		return settings_state_action_policy::PendingRequestPolicy::CoalesceLatest;
	}
	return settings_state_action_policy::PendingRequestPolicy::RejectWhilePending;
}

bool SettingsToggleMediator::request_toggle(
	SettingsToggleId id, bool target_active)
{
	const auto policy = request_policy(id);
	if (policy ==
	    settings_state_action_policy::PendingRequestPolicy::CoalesceLatest) {
		if (!settings_action(id)) {
			return false;
		}

		return _coordinator.request_change(
			id, target_active, policy, settings_action_dispatcher());
	}

	return _coordinator.request_change(id, target_active, settings_action(id));
}

void SettingsToggleMediator::start_service_refresh_timer()
{
	if (_service_refresh_timer == nullptr) {
		_service_refresh_timer =
			lv_timer_create(service_refresh_timer_cb,
					kServiceRefreshIntervalMs, this);
	}
}

void SettingsToggleMediator::stop_service_refresh_timer()
{
	if (_service_refresh_timer != nullptr) {
		lv_timer_delete(_service_refresh_timer);
		_service_refresh_timer = nullptr;
	}
}

void SettingsToggleMediator::clear_page_callbacks()
{
	_dashboard.on_mcp_click = nullptr;
	_dashboard.on_replay_click = nullptr;
	_dashboard.on_ncm_click = nullptr;
	_dashboard.on_ssh_click = nullptr;
	_dashboard.on_auto_rotation_click = nullptr;

	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		if (SettingsStateTogglePage *page =
			    _page_registry.settings_toggle_page(id)) {
			page->on_back_click = nullptr;
			page->on_enabled_change = nullptr;
		}
	}
}

void SettingsToggleMediator::service_refresh_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}

	auto *self = static_cast<SettingsToggleMediator *>(
		lv_timer_get_user_data(timer));
	if (self != nullptr && !self->_is_shutting_down &&
	    self->_service_refresh_timer == timer) {
		(void)self->request_service_refresh();
	}
}

} // namespace ui
