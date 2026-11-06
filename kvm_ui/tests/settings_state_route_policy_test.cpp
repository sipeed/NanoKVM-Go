#include <array>

#include <catch2/catch_test_macros.hpp>

#include "ui/settings_state_route_policy.hpp"

namespace
{

using Button = ui::SettingsToggleId;
using HomeAction = ui::settings_state_route_policy::SettingsStateHomeAction;
using ControlSurface =
	ui::settings_state_route_policy::SettingsStateControlSurface;

constexpr std::array kExperimentalButtons {
	Button::MCP, Button::REPLAY,	    Button::NCM,
	Button::SSH, Button::AUTO_ROTATION,
};

} // namespace

TEST_CASE("settings state route policy opens experimental subpages")
{
	for (const auto button : kExperimentalButtons) {
		CHECK(ui::settings_state_route_policy::
			      has_experimental_toggle_subpage(button));
		CHECK(ui::settings_state_route_policy::home_action(
			      button, true) == HomeAction::OpenSubpage);
		CHECK(ui::settings_state_route_policy::control_surface(
			      button, true) == ControlSurface::ToggleSubpage);
	}
}

TEST_CASE("settings state route policy keeps legacy direct toggle mode")
{
	for (const auto button : kExperimentalButtons) {
		CHECK(ui::settings_state_route_policy::home_action(
			      button, false) == HomeAction::DirectToggle);
		CHECK(ui::settings_state_route_policy::control_surface(
			      button, false) == ControlSurface::HomeTile);
	}
}
