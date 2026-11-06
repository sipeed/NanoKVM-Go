#include <catch2/catch_test_macros.hpp>

#include "ui/settings_state_action_policy.hpp"

namespace
{

using Button = ui::SettingsToggleId;

} // namespace

TEST_CASE(
	"settings state buttons do not locally toggle without runtime callbacks")
{
	CHECK_FALSE(ui::settings_state_action_policy::
			    allows_missing_callback_local_toggle(Button::MCP));
	CHECK_FALSE(
		ui::settings_state_action_policy::
			allows_missing_callback_local_toggle(Button::REPLAY));
	CHECK_FALSE(ui::settings_state_action_policy::
			    allows_missing_callback_local_toggle(Button::NCM));
	CHECK_FALSE(ui::settings_state_action_policy::
			    allows_missing_callback_local_toggle(Button::SSH));
	CHECK_FALSE(ui::settings_state_action_policy::
			    allows_missing_callback_local_toggle(
				    Button::AUTO_ROTATION));
}

TEST_CASE("settings state action pending waits through transient source state")
{
	using ui::settings_state_action_policy::PendingDecision;
	using ui::settings_state_action_policy::PendingRequestPolicy;
	using ui::settings_state_action_policy::pending_decision;

	CHECK(pending_decision(Button::SSH, true, false, false, false) ==
	      PendingDecision::KeepWaiting);
	CHECK(pending_decision(Button::SSH, true, true, false, false) ==
	      PendingDecision::KeepWaiting);
	CHECK(pending_decision(Button::SSH, true, true, true, false) ==
	      PendingDecision::Finish);
	CHECK(pending_decision(Button::SSH, true, false, true, false) ==
	      PendingDecision::KeepWaiting);
	CHECK(pending_decision(Button::SSH, true, false, true, true) ==
	      PendingDecision::Finish);
	CHECK(pending_decision(Button::SSH, true, true, false, false,
			       PendingRequestPolicy::CoalesceLatest) ==
	      PendingDecision::Finish);
}

TEST_CASE(
	"settings state action policy skips only the invisible hold for coalesced intent")
{
	using ui::settings_state_action_policy::PendingRequestPolicy;
	using ui::settings_state_action_policy::requires_min_loading;

	CHECK(requires_min_loading(PendingRequestPolicy::RejectWhilePending));
	CHECK_FALSE(requires_min_loading(PendingRequestPolicy::CoalesceLatest));
}

TEST_CASE(
	"settings state action policy exposes longer wait windows for slow toggles")
{
	using ui::settings_state_action_policy::max_pending_wait_ms;
	using ui::settings_state_action_policy::min_loading_ms;

	CHECK(min_loading_ms(Button::AUTO_ROTATION) == 500);
	CHECK(max_pending_wait_ms(Button::AUTO_ROTATION) == 500);
	CHECK(max_pending_wait_ms(Button::SSH) > min_loading_ms(Button::SSH));
	CHECK(max_pending_wait_ms(Button::MCP) == 12000);
}

TEST_CASE("settings state action preview uses requested target while pending")
{
	using ui::settings_state_action_policy::preview_active;

	CHECK(preview_active(false, true, true));
	CHECK_FALSE(preview_active(true, false, true));
	CHECK(preview_active(true, false, false));
}

TEST_CASE("manual rotation action submits after the loading hold")
{
	using ui::settings_state_action_policy::submits_action_after_min_loading;

	CHECK_FALSE(submits_action_after_min_loading(Button::MCP));
	CHECK_FALSE(submits_action_after_min_loading(Button::REPLAY));
	CHECK_FALSE(submits_action_after_min_loading(Button::NCM));
	CHECK_FALSE(submits_action_after_min_loading(Button::SSH));
	CHECK(submits_action_after_min_loading(Button::AUTO_ROTATION));
}

TEST_CASE("optimistic action presentation is limited to four service toggles")
{
	using ui::settings_state_action_policy::allows_coalesced_request;
	using ui::settings_state_action_policy::
		uses_optimistic_action_presentation;

	for (const Button id : { Button::MCP, Button::REPLAY, Button::NCM,
						 Button::SSH }) {
		CHECK(allows_coalesced_request(id));
		CHECK_FALSE(uses_optimistic_action_presentation(id, false));
		CHECK(uses_optimistic_action_presentation(id, true));
	}
	CHECK_FALSE(allows_coalesced_request(Button::AUTO_ROTATION));
	CHECK_FALSE(uses_optimistic_action_presentation(
		Button::AUTO_ROTATION, false));
	CHECK_FALSE(uses_optimistic_action_presentation(
		Button::AUTO_ROTATION, true));
}
