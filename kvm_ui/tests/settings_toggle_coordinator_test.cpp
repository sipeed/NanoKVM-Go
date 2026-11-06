#include <memory>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/settings_state_action_policy.hpp"
#include "ui/settings_toggle_coordinator.hpp"

namespace
{

using ui::SettingsToggleCoordinator;
using ui::SettingsToggleId;
using PendingRequestPolicy =
	ui::settings_state_action_policy::PendingRequestPolicy;

void advance_to_minimum_loading(test_support::LvglTestContext &lvgl)
{
	lvgl.advance(
		ui::settings_state_action_policy::kSettingsStateMinLoadingMs);
}

} // namespace

TEST_CASE(
	"SettingsToggleCoordinator keeps confirmed runtime state pending until minimum hold",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	int callback_count = 0;
	bool callback_target = false;
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   [&](bool target) {
						   ++callback_count;
						   callback_target = target;
					   }));

	CHECK(callback_count == 1);
	CHECK(callback_target);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK_FALSE(coordinator.request_change(SettingsToggleId::NCM, false,
							 [&](bool) { ++callback_count; }));
	CHECK(callback_count == 1);

	coordinator.set_confirmed(SettingsToggleId::NCM, true);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).action_pending);

	advance_to_minimum_loading(lvgl);
	const auto model = coordinator.view_model(SettingsToggleId::NCM);
	CHECK(model.confirmed);
	CHECK_FALSE(model.action_pending);
	CHECK_FALSE(model.is_pending());
}

TEST_CASE(
	"SettingsToggleCoordinator invokes delayed auto rotation action before completing pending",
	"[ui][settings][toggle-coordinator][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_auto_rotation_manual_mode(true);
	coordinator.set_confirmed(SettingsToggleId::AUTO_ROTATION, false);

	int callback_count = 0;
	bool callback_saw_pending = false;
	REQUIRE(coordinator.request_change(
		SettingsToggleId::AUTO_ROTATION, true, [&](bool target) {
			++callback_count;
			CHECK(target);
			callback_saw_pending =
				coordinator
					.view_model(
						SettingsToggleId::AUTO_ROTATION)
					.action_pending;
		}));

	CHECK(callback_count == 0);
	coordinator.set_confirmed(SettingsToggleId::AUTO_ROTATION, true);
	advance_to_minimum_loading(lvgl);

	CHECK(callback_count == 1);
	CHECK(callback_saw_pending);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::AUTO_ROTATION)
				    .action_pending);
}

TEST_CASE(
	"SettingsToggleCoordinator keeps auto rotation on the delayed reject policy",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_auto_rotation_manual_mode(true);
	coordinator.set_confirmed(SettingsToggleId::AUTO_ROTATION, false);

	int callback_count = 0;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool target) {
			CHECK(target);
			++callback_count;
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::AUTO_ROTATION, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	CHECK_FALSE(coordinator.request_change(SettingsToggleId::AUTO_ROTATION,
							 false,
							 PendingRequestPolicy::CoalesceLatest,
							 dispatcher));
	CHECK(callback_count == 0);

	coordinator.set_confirmed(SettingsToggleId::AUTO_ROTATION, true);
	advance_to_minimum_loading(lvgl);
	CHECK(callback_count == 1);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::AUTO_ROTATION)
				    .action_pending);
}

TEST_CASE(
	"SettingsToggleCoordinator clears boot and action timers during teardown",
	"[ui][settings][toggle-coordinator][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	int callback_count = 0;

	{
		auto coordinator =
			std::make_unique<SettingsToggleCoordinator>(8000);
		coordinator->start_boot_pending();
		coordinator->set_auto_rotation_manual_mode(true);
		REQUIRE(coordinator->view_model(SettingsToggleId::MCP)
				.boot_pending);
		REQUIRE(coordinator->request_change(
			SettingsToggleId::AUTO_ROTATION, true, [&](bool) {
				++callback_count;
			}));
		CHECK(lvgl.timer_count() == timer_count_before + 3U);
		coordinator.reset();
	}

	CHECK(lvgl.timer_count() == timer_count_before);
	lvgl.advance(ui::settings_state_action_policy::
			     kSettingsStateSlowPendingTimeoutMs);
	CHECK(callback_count == 0);
}

TEST_CASE(
	"SettingsToggleCoordinator tolerates destruction from a delayed callback",
	"[ui][settings][toggle-coordinator][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	auto coordinator = std::make_unique<SettingsToggleCoordinator>(8000);
	coordinator->set_auto_rotation_manual_mode(true);
	coordinator->set_confirmed(SettingsToggleId::AUTO_ROTATION, false);

	REQUIRE(coordinator->request_change(SettingsToggleId::AUTO_ROTATION,
					    true, [&](bool) {
						    coordinator.reset();
					    }));
	REQUIRE(lvgl.timer_count() == timer_count_before + 2U);

	advance_to_minimum_loading(lvgl);
	CHECK_FALSE(coordinator);
	CHECK(lvgl.timer_count() == timer_count_before);
}

TEST_CASE(
	"SettingsToggleCoordinator applies service availability with event sequencing",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	std::vector<SettingsToggleId> notifications;
	coordinator.set_view_model_change_callback([&](SettingsToggleId id) {
		notifications.push_back(id);
	});

	const ui::SettingsServiceRuntimeState runtime_state {
		.known = true,
		.refresh_failed = false,
		.event_seq = 7,
		.mcp_active = true,
		.mcp_available = true,
		.replay_active = false,
		.replay_available = false,
	};
	REQUIRE(coordinator.apply_service_runtime_state(runtime_state));
	CHECK(coordinator.view_model(SettingsToggleId::MCP).confirmed);
	CHECK(coordinator.view_model(SettingsToggleId::MCP).available);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::REPLAY).available);

	const ui::SettingsServiceRuntimeState failed_refresh {
		.known = true,
		.refresh_failed = true,
		.event_seq = 8,
		.mcp_active = false,
		.mcp_available = false,
		.replay_active = true,
		.replay_available = true,
	};
	REQUIRE(coordinator.apply_service_runtime_state(failed_refresh));
	CHECK(coordinator.view_model(SettingsToggleId::MCP).confirmed);
	CHECK(coordinator.view_model(SettingsToggleId::MCP).available);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::REPLAY).available);

	CHECK_FALSE(coordinator.apply_service_runtime_state(runtime_state));
	CHECK_FALSE(notifications.empty());

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator coalesces latest intent behind an immutable first target",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	std::vector<bool> callback_targets;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId id, bool target) {
			CHECK(id == SettingsToggleId::NCM);
			callback_targets.push_back(target);
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));

	const std::vector<bool> first_action { true };
	CHECK(callback_targets == first_action);
	const auto before_confirmation =
		coordinator.view_model(SettingsToggleId::NCM);
	CHECK(before_confirmation.action_pending);
	CHECK_FALSE(before_confirmation.pending_target);

	coordinator.set_confirmed(SettingsToggleId::NCM, true);
	const std::vector<bool> expected_actions { true, false };
	CHECK(callback_targets == expected_actions);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).action_pending);

	coordinator.set_confirmed(SettingsToggleId::NCM, false);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).confirmed);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator skips a coalesced action already satisfied by runtime truth",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	std::vector<bool> callback_targets;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool target) {
			callback_targets.push_back(target);
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	coordinator.set_confirmed(SettingsToggleId::NCM, false);
	const auto before_timeout = coordinator.view_model(SettingsToggleId::NCM);
	CHECK(before_timeout.action_pending);
	CHECK_FALSE(before_timeout.pending_target);
	CHECK_FALSE(before_timeout.confirmed);

	lvgl.advance(coordinator.pending_timeout_ms(SettingsToggleId::NCM));
	const std::vector<bool> expected_actions { true };
	CHECK(callback_targets == expected_actions);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).confirmed);
}

TEST_CASE(
	"SettingsToggleCoordinator does not replay a returned coalesced target",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	std::vector<bool> callback_targets;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool target) {
			callback_targets.push_back(target);
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	coordinator.set_confirmed(SettingsToggleId::NCM, true);

	const std::vector<bool> expected_actions { true };
	CHECK(callback_targets == expected_actions);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).confirmed);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator does not follow an unavailable service confirmation",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	const ui::SettingsServiceRuntimeState available_state {
		.known = true,
		.refresh_failed = false,
		.event_seq = 1,
		.mcp_active = false,
		.mcp_available = true,
		.replay_active = false,
		.replay_available = true,
	};
	REQUIRE(coordinator.apply_service_runtime_state(available_state));

	std::vector<bool> callback_targets;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool target) {
			callback_targets.push_back(target);
			return true;
		};
	REQUIRE(coordinator.request_change(SettingsToggleId::MCP, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::MCP, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));

	const ui::SettingsServiceRuntimeState unavailable_confirmation {
		.known = true,
		.refresh_failed = false,
		.event_seq = 2,
		.mcp_active = true,
		.mcp_available = false,
		.replay_active = false,
		.replay_available = true,
	};
	REQUIRE(coordinator.apply_service_runtime_state(unavailable_confirmation));

	const std::vector<bool> expected_actions { true };
	CHECK(callback_targets == expected_actions);
	const auto model = coordinator.view_model(SettingsToggleId::MCP);
	CHECK_FALSE(model.action_pending);
	CHECK(model.confirmed);
	CHECK_FALSE(model.available);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator consumes a timed-out coalesced generation once",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	std::vector<bool> callback_targets;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool target) {
			callback_targets.push_back(target);
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));

	lvgl.advance(coordinator.pending_timeout_ms(SettingsToggleId::NCM));
	const std::vector<bool> after_first_timeout { true, true };
	CHECK(callback_targets == after_first_timeout);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).action_pending);

	lvgl.advance(coordinator.pending_timeout_ms(SettingsToggleId::NCM));
	CHECK(callback_targets == after_first_timeout);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).confirmed);
}

TEST_CASE(
	"SettingsToggleCoordinator resolves the follow-up through the newest dispatcher",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	std::vector<bool> first_dispatcher_targets;
	std::vector<bool> rebound_dispatcher_targets;
	const SettingsToggleCoordinator::ActionDispatcher first_dispatcher =
		[&](SettingsToggleId, bool target) {
			first_dispatcher_targets.push_back(target);
			return true;
		};
	const SettingsToggleCoordinator::ActionDispatcher rebound_dispatcher =
		[&](SettingsToggleId, bool target) {
			rebound_dispatcher_targets.push_back(target);
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   first_dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   rebound_dispatcher));
	coordinator.set_confirmed(SettingsToggleId::NCM, true);

	const std::vector<bool> first_expected { true };
	const std::vector<bool> rebound_expected { false };
	CHECK(first_dispatcher_targets == first_expected);
	CHECK(rebound_dispatcher_targets == rebound_expected);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).action_pending);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator clears a coalesced follow-up that loses its dispatcher",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_confirmed(SettingsToggleId::NCM, false);

	int initial_dispatch_count = 0;
	const SettingsToggleCoordinator::ActionDispatcher initial_dispatcher =
		[&](SettingsToggleId, bool) {
			++initial_dispatch_count;
			return true;
		};
	const SettingsToggleCoordinator::ActionDispatcher missing_dispatcher =
		[](SettingsToggleId, bool) { return false; };

	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, true,
					   PendingRequestPolicy::CoalesceLatest,
					   initial_dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::NCM, false,
					   PendingRequestPolicy::CoalesceLatest,
					   missing_dispatcher));
	coordinator.set_confirmed(SettingsToggleId::NCM, true);

	CHECK(initial_dispatch_count == 1);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::NCM).action_pending);
	CHECK(coordinator.view_model(SettingsToggleId::NCM).confirmed);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator clears coalesced intent when service availability ends",
	"[ui][settings][toggle-coordinator]")
{
	test_support::LvglTestContext lvgl;
	SettingsToggleCoordinator coordinator(8000);
	coordinator.set_available(SettingsToggleId::MCP, true);
	coordinator.set_confirmed(SettingsToggleId::MCP, false);

	int callback_count = 0;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool) {
			++callback_count;
			return true;
		};

	REQUIRE(coordinator.request_change(SettingsToggleId::MCP, true,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	REQUIRE(coordinator.request_change(SettingsToggleId::MCP, false,
					   PendingRequestPolicy::CoalesceLatest,
					   dispatcher));
	coordinator.set_available(SettingsToggleId::MCP, false);
	coordinator.set_confirmed(SettingsToggleId::MCP, true);

	CHECK(callback_count == 1);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::MCP).action_pending);
	CHECK_FALSE(coordinator.view_model(SettingsToggleId::MCP).available);

	(void)lvgl;
}

TEST_CASE(
	"SettingsToggleCoordinator tolerates destruction from a coalesced follow-up",
	"[ui][settings][toggle-coordinator][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	auto coordinator = std::make_unique<SettingsToggleCoordinator>(8000);
	coordinator->set_confirmed(SettingsToggleId::NCM, false);

	int callback_count = 0;
	const SettingsToggleCoordinator::ActionDispatcher dispatcher =
		[&](SettingsToggleId, bool) {
			++callback_count;
			if (callback_count == 2) {
				coordinator.reset();
			}
			return true;
		};

	REQUIRE(coordinator->request_change(SettingsToggleId::NCM, true,
					    PendingRequestPolicy::CoalesceLatest,
					    dispatcher));
	REQUIRE(coordinator->request_change(SettingsToggleId::NCM, false,
					    PendingRequestPolicy::CoalesceLatest,
					    dispatcher));
	coordinator->set_confirmed(SettingsToggleId::NCM, true);

	CHECK_FALSE(coordinator);
	CHECK(callback_count == 2);
	CHECK(lvgl.timer_count() == timer_count_before);
}
