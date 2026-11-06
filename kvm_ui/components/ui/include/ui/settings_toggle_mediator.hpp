#pragma once

#include <cstdint>

#include <lvgl.h>

#include "ui/page_manager_actions.hpp"
#include "ui/settings_toggle_coordinator.hpp"

namespace ui
{

class PageRegistry;
class SettingsDashboardView;
class SettingsNavigator;

class SettingsToggleMediator final {
    public:
	SettingsToggleMediator(SettingsDashboardView &dashboard,
			       PageRegistry &page_registry,
			       SettingsNavigator &settings_navigator,
			       const PageManagerActions::Settings &actions,
			       uint32_t boot_pending_timeout_ms,
			       bool experimental_settings_subpages,
			       bool enable_optimistic_actions = false);
	~SettingsToggleMediator();

	SettingsToggleMediator(const SettingsToggleMediator &) = delete;
	SettingsToggleMediator &
	operator=(const SettingsToggleMediator &) = delete;

	void bind_pages();
	void start();
	bool request_service_refresh();
	void set_confirmed(SettingsToggleId id, bool active);
	void set_auto_rotation_manual_mode(bool manual_mode);
	void apply_service_runtime_state(
		const SettingsServiceRuntimeState &state);
	void begin_shutdown();
	void shutdown();

    private:
	static constexpr uint32_t kServiceRefreshIntervalMs = 3000;

	SettingsDashboardView &_dashboard;
	PageRegistry &_page_registry;
	SettingsNavigator &_settings_navigator;
	const PageManagerActions::Settings &_actions;
	const bool _experimental_settings_subpages;
	const bool _enable_optimistic_actions;
	SettingsToggleCoordinator _coordinator;
	bool _is_shutting_down = false;
	bool _started = false;
	lv_timer_t *_service_refresh_timer = nullptr;

	void bind_toggle_page(SettingsToggleId id);
	void route_home_click(SettingsToggleId id);
	void show_toggle_subpage(SettingsToggleId id);
	void refresh_presentation();
	void render_all();
	void render_toggle(SettingsToggleId id);
	void sync_availability();
	void return_from_unavailable_subpage();
	SettingsToggleCoordinator::ActionCallback
	settings_action(SettingsToggleId id) const;
	SettingsToggleCoordinator::ActionDispatcher
	settings_action_dispatcher() const;
	settings_state_action_policy::PendingRequestPolicy
	request_policy(SettingsToggleId id) const noexcept;
	bool request_toggle(SettingsToggleId id, bool target_active);
	void start_service_refresh_timer();
	void stop_service_refresh_timer();
	void clear_page_callbacks();

	static void service_refresh_timer_cb(lv_timer_t *timer);
};

} // namespace ui
