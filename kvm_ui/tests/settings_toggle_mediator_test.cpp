#include <array>
#include <cstddef>
#include <string_view>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/bcona.hpp"
#include "ui/control_state.hpp"
#include "ui/page_manager_actions.hpp"
#include "ui/page_registry.hpp"
#include "ui/root_pager.hpp"
#include "ui/settings_dashboard_view.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/settings_page.hpp"
#include "ui/settings_route.hpp"
#include "ui/settings_state_action_policy.hpp"
#include "ui/settings_state_toggle_page.hpp"
#include "ui/settings_toggle_mediator.hpp"

namespace
{

constexpr uint32_t kBootPendingTimeoutMs = 8000;
constexpr uint32_t kWifiPendingTimeoutMs = 15000;
constexpr uint32_t kServiceRefreshIntervalMs = 3000;

class SettingsToggleMediatorHarness final {
    public:
	explicit SettingsToggleMediatorHarness(
		test_support::LvglTestContext &lvgl,
		bool experimental_settings_subpages,
		bool enable_optimistic_actions = false)
		: _pager(ui::RootPager::TransitionMode::Slide)
		, _dashboard(experimental_settings_subpages,
			     kBootPendingTimeoutMs)
		, _navigator(_pager, _registry)
		, _mediator(_dashboard, _registry, _navigator,
			    _actions.settings, kBootPendingTimeoutMs,
			    experimental_settings_subpages, enable_optimistic_actions)
	{
		_pager.create(lvgl.screen());
		_dashboard.create(_pager.content_parent(), nullptr,
				  kWifiPendingTimeoutMs);
		_registry.create(_pager.settings_host());
		_mediator.bind_pages();
	}

	void start()
	{
		_mediator.start();
	}

	void clear_boot_pending()
	{
		for (const ui::SettingsToggleId id :
		     ui::kAllSettingsToggleIds) {
			_mediator.set_confirmed(id, false);
		}
	}

	ui::SettingsPage *settings_page() const noexcept
	{
		return _dashboard.settings_page();
	}

	ui::SettingsStateTogglePage *
	toggle_page(ui::SettingsToggleId id) const noexcept
	{
		return _registry.settings_toggle_page(id);
	}

	ui::PageManagerActions _actions;
	ui::RootPager _pager;
	ui::SettingsDashboardView _dashboard;
	ui::PageRegistry _registry;
	ui::SettingsNavigator _navigator;
	ui::SettingsToggleMediator _mediator;
};

ui::SettingsServiceRuntimeState
available_service_state(uint64_t event_seq, bool mcp_active, bool replay_active)
{
	return {
		.known = true,
		.refresh_failed = false,
		.event_seq = event_seq,
		.mcp_active = mcp_active,
		.mcp_available = true,
		.replay_active = replay_active,
		.replay_available = true,
	};
}

ui::ControlState *dashboard_control(ui::SettingsPage *page,
					    ui::SettingsToggleId id)
{
	if (page == nullptr) {
		return nullptr;
	}

	switch (id) {
	case ui::SettingsToggleId::MCP:
		return page->get_mcp_btn();
	case ui::SettingsToggleId::REPLAY:
		return page->get_replay_btn();
	case ui::SettingsToggleId::NCM:
		return page->get_ncm_btn();
	case ui::SettingsToggleId::SSH:
		return page->get_ssh_btn();
	case ui::SettingsToggleId::AUTO_ROTATION:
		return page->get_auto_rotation_btn();
	}

	return nullptr;
}

std::string_view button_text(ui::ControlState *control)
{
	auto *button = dynamic_cast<ui::BConA *>(control);
	return button == nullptr ? std::string_view {} :
					  test_support::visible_direct_text(button->get_obj());
}

} // namespace

TEST_CASE(
	"SettingsToggleMediator maps legacy dashboard toggles to all settings actions",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, false);
	std::array<int, ui::kSettingsToggleCount> action_calls {};
	std::array<bool, ui::kSettingsToggleCount> action_targets {};

	harness._actions.settings.set_mcp_enabled = [&](bool active) {
		++action_calls[ui::to_settings_toggle_index(
			ui::SettingsToggleId::MCP)];
		action_targets[ui::to_settings_toggle_index(
			ui::SettingsToggleId::MCP)] = active;
	};
	harness._actions.settings.set_replay_enabled = [&](bool active) {
		++action_calls[ui::to_settings_toggle_index(
			ui::SettingsToggleId::REPLAY)];
		action_targets[ui::to_settings_toggle_index(
			ui::SettingsToggleId::REPLAY)] = active;
	};
	harness._actions.settings.set_ncm_enabled = [&](bool active) {
		++action_calls[ui::to_settings_toggle_index(
			ui::SettingsToggleId::NCM)];
		action_targets[ui::to_settings_toggle_index(
			ui::SettingsToggleId::NCM)] = active;
	};
	harness._actions.settings.set_ssh_enabled = [&](bool active) {
		++action_calls[ui::to_settings_toggle_index(
			ui::SettingsToggleId::SSH)];
		action_targets[ui::to_settings_toggle_index(
			ui::SettingsToggleId::SSH)] = active;
	};
	harness._actions.settings.set_auto_rotation_enabled = [&](bool active) {
		++action_calls[ui::to_settings_toggle_index(
			ui::SettingsToggleId::AUTO_ROTATION)];
		action_targets[ui::to_settings_toggle_index(
			ui::SettingsToggleId::AUTO_ROTATION)] = active;
	};

	harness.start();
	harness.clear_boot_pending();
	harness._mediator.apply_service_runtime_state(
		available_service_state(1, false, false));

	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(settings_page != nullptr);
	REQUIRE(settings_page->on_mcp_click);
	REQUIRE(settings_page->on_replay_click);
	REQUIRE(settings_page->on_ncm_click);
	REQUIRE(settings_page->on_ssh_click);
	REQUIRE(settings_page->on_auto_rotation_click);
	settings_page->on_mcp_click();
	settings_page->on_replay_click();
	settings_page->on_ncm_click();
	settings_page->on_ssh_click();
	settings_page->on_auto_rotation_click();

	for (const ui::SettingsToggleId id : ui::kAllSettingsToggleIds) {
		const std::size_t index = ui::to_settings_toggle_index(id);
		CHECK(action_calls[index] == 1);
		CHECK(action_targets[index]);
	}
}

TEST_CASE(
	"SettingsToggleMediator resolves toggle page actions from the current bundle",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true);
	int first_action_calls = 0;
	int rebound_action_calls = 0;

	harness._actions.settings.set_ncm_enabled = [&](bool active) {
		CHECK(active);
		++first_action_calls;
	};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const ncm_page =
		harness.toggle_page(ui::SettingsToggleId::NCM);
	REQUIRE(ncm_page != nullptr);
	REQUIRE(ncm_page->on_enabled_change);
	ncm_page->on_enabled_change(true);
	CHECK(first_action_calls == 1);

	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, true);
	lvgl.advance(
		ui::settings_state_action_policy::kSettingsStateMinLoadingMs);

	ui::PageManagerActions rebound_actions;
	rebound_actions.settings.set_ncm_enabled = [&](bool active) {
		CHECK_FALSE(active);
		++rebound_action_calls;
	};
	harness._actions = std::move(rebound_actions);
	ncm_page->on_enabled_change(false);

	CHECK(first_action_calls == 1);
	CHECK(rebound_action_calls == 1);
}

TEST_CASE(
	"SettingsToggleMediator preserves coalesced intent across missing and rebound actions",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	std::vector<bool> first_targets;
	std::vector<bool> rebound_targets;
	harness._actions.settings.set_ncm_enabled =
		[&first_targets](bool active) {
			first_targets.push_back(active);
		};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const ncm_page =
		harness.toggle_page(ui::SettingsToggleId::NCM);
	REQUIRE(ncm_page != nullptr);
	REQUIRE(ncm_page->on_enabled_change);
	ui::ControlState *const ncm_control = ncm_page->toggle_control();
	REQUIRE(ncm_control != nullptr);
	ncm_page->on_enabled_change(true);
	CHECK(first_targets == std::vector<bool> { true });

	// The old callback exists while false is accepted. It is only a queued
	// intent; no second action is dispatched yet.
	ncm_page->on_enabled_change(false);
	CHECK(first_targets == std::vector<bool> { true });
	CHECK_FALSE(ncm_control->is_active());

	// A missing callback rejects a later press without disturbing the immutable
	// action or the latest accepted false intent.
	harness._actions.settings.set_ncm_enabled = {};
	ncm_page->on_enabled_change(true);
	CHECK(first_targets == std::vector<bool> { true });
	CHECK_FALSE(ncm_control->is_active());

	ui::PageManagerActions rebound_actions;
	rebound_actions.settings.set_ncm_enabled =
		[&rebound_targets](bool active) {
			rebound_targets.push_back(active);
		};
	harness._actions = std::move(rebound_actions);
	CHECK(first_targets == std::vector<bool> { true });
	CHECK(rebound_targets.empty());

	// The accepted false intent resolves its callback only when it dispatches,
	// so this follow-up reaches the rebound action rather than the old closure.
	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, true);
	CHECK(rebound_targets == std::vector<bool> { false });
	CHECK(first_targets == std::vector<bool> { true });
	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, false);
	lvgl.advance(ui::settings_state_action_policy::max_pending_wait_ms(
		ui::SettingsToggleId::NCM));
}

TEST_CASE(
	"SettingsToggleMediator rejects an optimistic request with no current action",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const ncm_page =
		harness.toggle_page(ui::SettingsToggleId::NCM);
	REQUIRE(ncm_page != nullptr);
	REQUIRE(ncm_page->on_enabled_change);
	ui::ControlState *const ncm_control = ncm_page->toggle_control();
	REQUIRE(ncm_control != nullptr);
	const std::size_t timer_count_before = lvgl.timer_count();

	ncm_page->on_enabled_change(true);

	CHECK(lvgl.timer_count() == timer_count_before);
	CHECK_FALSE(ncm_control->is_active());
	CHECK_FALSE(ncm_control->is_pending());
	CHECK(button_text(ncm_control) == "OFF");
}

TEST_CASE(
	"SettingsToggleMediator keeps legacy action pending presentation disabled by default",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true);
	int action_calls = 0;
	harness._actions.settings.set_ncm_enabled = [&](bool active) {
		CHECK(active);
		++action_calls;
	};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const ncm_page =
		harness.toggle_page(ui::SettingsToggleId::NCM);
	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(ncm_page != nullptr);
	REQUIRE(settings_page != nullptr);
	ui::ControlState *const ncm_page_control = ncm_page->toggle_control();
	ui::ControlState *const ncm_dashboard_control = dashboard_control(
		settings_page, ui::SettingsToggleId::NCM);
	REQUIRE(ncm_page_control != nullptr);
	REQUIRE(ncm_dashboard_control != nullptr);

	ncm_page->on_enabled_change(true);

	CHECK(action_calls == 1);
	CHECK(ncm_page_control->is_pending());
	CHECK(button_text(ncm_page_control) == "...");
	CHECK_FALSE(ncm_dashboard_control->is_active());
	CHECK_FALSE(ncm_dashboard_control->is_pending());

	// With the experiment disabled, a second target remains rejected while the
	// original legacy request is pending.
	ncm_page->on_enabled_change(false);
	CHECK(action_calls == 1);
	CHECK(ncm_page_control->is_pending());
	CHECK(button_text(ncm_page_control) == "...");
}

TEST_CASE(
	"SettingsToggleMediator coalesces optimistic service targets on both surfaces",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	std::array<int, ui::kSettingsToggleCount> action_calls {};
	std::array<std::vector<bool>, ui::kSettingsToggleCount> action_targets;

	const auto bind_action = [&](ui::SettingsToggleId id) {
		const auto action = [&, id](bool active) {
			const std::size_t index = ui::to_settings_toggle_index(id);
			++action_calls[index];
			action_targets[index].push_back(active);
		};
		switch (id) {
		case ui::SettingsToggleId::MCP:
			harness._actions.settings.set_mcp_enabled = action;
			return;
		case ui::SettingsToggleId::REPLAY:
			harness._actions.settings.set_replay_enabled = action;
			return;
		case ui::SettingsToggleId::NCM:
			harness._actions.settings.set_ncm_enabled = action;
			return;
		case ui::SettingsToggleId::SSH:
			harness._actions.settings.set_ssh_enabled = action;
			return;
		case ui::SettingsToggleId::AUTO_ROTATION:
			return;
		}
	};
	for (const ui::SettingsToggleId id : { ui::SettingsToggleId::MCP,
						      ui::SettingsToggleId::REPLAY,
						      ui::SettingsToggleId::NCM,
						      ui::SettingsToggleId::SSH }) {
		bind_action(id);
	}

	harness.start();
	harness.clear_boot_pending();
	harness._mediator.apply_service_runtime_state(
		available_service_state(1, false, false));
	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(settings_page != nullptr);

	for (const ui::SettingsToggleId id : { ui::SettingsToggleId::MCP,
						      ui::SettingsToggleId::REPLAY,
						      ui::SettingsToggleId::NCM,
						      ui::SettingsToggleId::SSH }) {
		ui::SettingsStateTogglePage *const page = harness.toggle_page(id);
		ui::ControlState *const dashboard = dashboard_control(settings_page, id);
		REQUIRE(page != nullptr);
		REQUIRE(page->on_enabled_change);
		REQUIRE(page->toggle_control() != nullptr);
		REQUIRE(dashboard != nullptr);

		page->on_enabled_change(true);
		CHECK(action_calls[ui::to_settings_toggle_index(id)] == 1);
		CHECK(page->toggle_control()->is_active());
		CHECK_FALSE(page->toggle_control()->is_pending());
		CHECK(button_text(page->toggle_control()) == "ON");
		CHECK(dashboard->is_active());
		CHECK_FALSE(dashboard->is_pending());

		// Later presses are accepted by the presentation path and replace only
		// the latest desired target. The first runtime action is unchanged.
		page->on_enabled_change(false);
		CHECK(action_calls[ui::to_settings_toggle_index(id)] == 1);
		CHECK_FALSE(page->toggle_control()->is_active());
		CHECK_FALSE(page->toggle_control()->is_pending());
		CHECK_FALSE(dashboard->is_active());
		CHECK_FALSE(dashboard->is_pending());
		page->on_enabled_change(true);
		page->on_enabled_change(false);
		CHECK(action_calls[ui::to_settings_toggle_index(id)] == 1);
		CHECK_FALSE(page->toggle_control()->is_active());
		CHECK_FALSE(dashboard->is_active());

		// Confirmation of the immutable first target dispatches only the final
		// unsatisfied intent, with no optimistic spinner or minimum hold.
		harness._mediator.set_confirmed(id, true);
		CHECK_FALSE(page->toggle_control()->is_active());
		CHECK_FALSE(page->toggle_control()->is_pending());
		CHECK_FALSE(dashboard->is_active());
		CHECK_FALSE(dashboard->is_pending());
		CHECK(action_calls[ui::to_settings_toggle_index(id)] == 2);
		CHECK(action_targets[ui::to_settings_toggle_index(id)] ==
		      std::vector<bool> { true, false });
		lvgl.advance(ui::settings_state_action_policy::max_pending_wait_ms(id));
		harness._mediator.set_confirmed(id, false);
		CHECK_FALSE(page->toggle_control()->is_active());
		CHECK_FALSE(page->toggle_control()->is_pending());
		CHECK_FALSE(dashboard->is_active());
		CHECK_FALSE(dashboard->is_pending());
	}
}

TEST_CASE(
	"SettingsToggleMediator derives legacy-dashboard presses from latest intent",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, false, true);
	std::vector<bool> action_targets;
	harness._actions.settings.set_ncm_enabled = [&](bool active) {
		action_targets.push_back(active);
	};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(settings_page != nullptr);
	ui::ControlState *const ncm_dashboard = dashboard_control(
		settings_page, ui::SettingsToggleId::NCM);
	REQUIRE(ncm_dashboard != nullptr);
	REQUIRE(settings_page->on_ncm_click);

	settings_page->on_ncm_click();
	CHECK(action_targets == std::vector<bool> { true });
	CHECK(ncm_dashboard->is_active());
	CHECK_FALSE(ncm_dashboard->is_pending());

	// The normal-looking control remains interactive. Each press alternates the
	// latest desired target while the first runtime action stays in flight.
	settings_page->on_ncm_click();
	CHECK(action_targets == std::vector<bool> { true });
	CHECK_FALSE(ncm_dashboard->is_active());
	CHECK_FALSE(ncm_dashboard->is_pending());
	settings_page->on_ncm_click();
	settings_page->on_ncm_click();
	CHECK(action_targets == std::vector<bool> { true });
	CHECK_FALSE(ncm_dashboard->is_active());

	// Matching the immutable first target dispatches the final desired target.
	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, true);
	CHECK_FALSE(ncm_dashboard->is_active());
	CHECK_FALSE(ncm_dashboard->is_pending());
	CHECK(action_targets == std::vector<bool> { true, false });
	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, false);
	CHECK_FALSE(ncm_dashboard->is_active());
	CHECK_FALSE(ncm_dashboard->is_pending());
}

TEST_CASE(
	"SettingsToggleMediator applies unavailable service state before confirmation",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	std::vector<bool> action_targets;
	harness._actions.settings.set_mcp_enabled =
		[&action_targets](bool active) {
			action_targets.push_back(active);
		};
	harness.start();
	harness.clear_boot_pending();
	harness._mediator.apply_service_runtime_state(
		available_service_state(1, false, false));

	ui::SettingsStateTogglePage *const mcp_page =
		harness.toggle_page(ui::SettingsToggleId::MCP);
	REQUIRE(mcp_page != nullptr);
	REQUIRE(mcp_page->on_enabled_change);
	mcp_page->on_enabled_change(true);
	mcp_page->on_enabled_change(false);
	CHECK(action_targets == std::vector<bool> { true });

	// The availability bit is applied first inside the runtime-state batch. The
	// same snapshot's active value therefore cannot trigger a follow-up action.
	harness._mediator.apply_service_runtime_state({
		.known = true,
		.refresh_failed = false,
		.event_seq = 2,
		.mcp_active = true,
		.mcp_available = false,
		.replay_active = false,
		.replay_available = true,
	});
	CHECK(action_targets == std::vector<bool> { true });
	lvgl.advance(ui::settings_state_action_policy::max_pending_wait_ms(
		ui::SettingsToggleId::MCP));
	CHECK(action_targets == std::vector<bool> { true });
}

TEST_CASE(
	"SettingsToggleMediator reconciles optimistic targets to confirmed state",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	int action_calls = 0;
	harness._actions.settings.set_ncm_enabled = [&](bool active) {
		CHECK(active);
		++action_calls;
	};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const ncm_page =
		harness.toggle_page(ui::SettingsToggleId::NCM);
	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(ncm_page != nullptr);
	REQUIRE(settings_page != nullptr);
	ui::ControlState *const page_control = ncm_page->toggle_control();
	ui::ControlState *const dashboard_state = dashboard_control(
		settings_page, ui::SettingsToggleId::NCM);
	REQUIRE(page_control != nullptr);
	REQUIRE(dashboard_state != nullptr);

	ncm_page->on_enabled_change(true);
	CHECK(action_calls == 1);
	CHECK(page_control->is_active());
	CHECK(dashboard_state->is_active());

	// An authoritative mismatch remains coordinator-pending until its existing
	// timeout, then replaces the optimistic target on both surfaces.
	harness._mediator.set_confirmed(ui::SettingsToggleId::NCM, false);
	lvgl.advance(ui::settings_state_action_policy::max_pending_wait_ms(
		ui::SettingsToggleId::NCM));
	CHECK_FALSE(page_control->is_active());
	CHECK_FALSE(dashboard_state->is_active());
	CHECK_FALSE(page_control->is_pending());
	CHECK_FALSE(dashboard_state->is_pending());
	CHECK(button_text(page_control) == "OFF");
}

TEST_CASE(
	"SettingsToggleMediator excludes auto rotation from optimistic presentation",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true, true);
	int action_calls = 0;
	harness._actions.settings.set_auto_rotation_enabled = [&](bool active) {
		CHECK(active);
		++action_calls;
	};
	harness.start();
	harness.clear_boot_pending();

	ui::SettingsStateTogglePage *const page =
		harness.toggle_page(ui::SettingsToggleId::AUTO_ROTATION);
	REQUIRE(page != nullptr);
	REQUIRE(page->toggle_control() != nullptr);
	page->on_enabled_change(true);

	CHECK(action_calls == 1);
	CHECK(page->toggle_control()->is_pending());
	CHECK(button_text(page->toggle_control()) == "...");
	page->on_enabled_change(false);
	CHECK(action_calls == 1);
	CHECK(page->toggle_control()->is_pending());
	CHECK(button_text(page->toggle_control()) == "...");

	// Auto Rotation retains its normal pending/minimum-loading behavior.
	harness._mediator.set_confirmed(ui::SettingsToggleId::AUTO_ROTATION, true);
	CHECK(page->toggle_control()->is_pending());
	lvgl.advance(ui::settings_state_action_policy::min_loading_ms(
		ui::SettingsToggleId::AUTO_ROTATION));
	CHECK(page->toggle_control()->is_active());
	CHECK_FALSE(page->toggle_control()->is_pending());
	CHECK(button_text(page->toggle_control()) == "ON");
}

TEST_CASE(
	"SettingsToggleMediator renders batched service state and returns unavailable subpages",
	"[ui][settings-toggle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true);
	harness.start();
	harness.clear_boot_pending();

	harness._mediator.apply_service_runtime_state(
		available_service_state(10, true, true));
	ui::SettingsStateTogglePage *const mcp_page =
		harness.toggle_page(ui::SettingsToggleId::MCP);
	ui::SettingsStateTogglePage *const replay_page =
		harness.toggle_page(ui::SettingsToggleId::REPLAY);
	REQUIRE(mcp_page != nullptr);
	REQUIRE(replay_page != nullptr);
	ui::ControlState *const mcp_control = mcp_page->toggle_control();
	ui::ControlState *const replay_control = replay_page->toggle_control();
	REQUIRE(mcp_control != nullptr);
	REQUIRE(replay_control != nullptr);
	CHECK(mcp_control->is_active());
	CHECK(replay_control->is_active());

	harness._mediator.apply_service_runtime_state(
		available_service_state(10, false, false));
	CHECK(mcp_control->is_active());
	CHECK(replay_control->is_active());

	harness._mediator.apply_service_runtime_state(
		available_service_state(11, false, false));
	CHECK_FALSE(mcp_control->is_active());
	CHECK_FALSE(replay_control->is_active());

	ui::SettingsPage *const settings_page = harness.settings_page();
	REQUIRE(settings_page != nullptr);
	settings_page->on_mcp_click();
	CHECK(harness._navigator.top_route() ==
	      ui::SettingsRoute::settings_toggle(ui::SettingsToggleId::MCP));

	harness._mediator.apply_service_runtime_state({
		.known = true,
		.refresh_failed = false,
		.event_seq = 12,
		.mcp_active = false,
		.mcp_available = false,
		.replay_active = false,
		.replay_available = true,
	});
	CHECK(harness._navigator.empty());
}

TEST_CASE(
	"SettingsToggleMediator owns its refresh timer and clears page callbacks on shutdown",
	"[ui][settings-toggle][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleMediatorHarness harness(lvgl, true);
	int refresh_requests = 0;
	harness._actions.settings.refresh_service_state = [&]() {
		++refresh_requests;
	};

	harness.start();
	harness.clear_boot_pending();
	CHECK(refresh_requests == 1);

	lvgl.advance(kServiceRefreshIntervalMs);
	CHECK(refresh_requests == 2);
	const std::size_t timer_count_before_shutdown = lvgl.timer_count();
	CHECK(timer_count_before_shutdown > 0U);

	ui::SettingsStateTogglePage *const mcp_page =
		harness.toggle_page(ui::SettingsToggleId::MCP);
	REQUIRE(mcp_page != nullptr);
	REQUIRE(mcp_page->on_enabled_change);
	harness._mediator.shutdown();
	CHECK(lvgl.timer_count() + 1U == timer_count_before_shutdown);
	CHECK_FALSE(harness._dashboard.on_mcp_click);
	CHECK_FALSE(mcp_page->on_back_click);
	CHECK_FALSE(mcp_page->on_enabled_change);
	CHECK_FALSE(harness._mediator.request_service_refresh());

	lvgl.advance(kServiceRefreshIntervalMs);
	CHECK(refresh_requests == 2);
	harness._mediator.shutdown();
	CHECK(lvgl.timer_count() + 1U == timer_count_before_shutdown);
}
