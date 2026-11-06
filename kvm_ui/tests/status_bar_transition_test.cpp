#include <catch2/catch_test_macros.hpp>

#include "ui/status_bar_transition.hpp"

namespace
{

using Content = ui::status_bar_transition::LeftContent;
using ui::status_bar_transition::build_left_transition_state;

void check_exclusive_content(float settings_visible_ratio)
{
	const auto state = build_left_transition_state(settings_visible_ratio);
	CHECK(state.show_icons != state.show_title);
	CHECK((state.content == Content::Icons) == state.show_icons);
	CHECK((state.content == Content::SettingsTitle) == state.show_title);
}

} // namespace

TEST_CASE("status bar left content follows the largest visible page area")
{
	CHECK(build_left_transition_state(0.49f).content == Content::Icons);
	CHECK(build_left_transition_state(0.50f).content ==
	      Content::SettingsTitle);
	CHECK(build_left_transition_state(0.51f).content ==
	      Content::SettingsTitle);
}

TEST_CASE("status bar transition is reversible from only the current ratio")
{
	CHECK(build_left_transition_state(0.80f).content ==
	      Content::SettingsTitle);
	CHECK(build_left_transition_state(0.60f).content ==
	      Content::SettingsTitle);
	CHECK(build_left_transition_state(0.40f).content == Content::Icons);
	CHECK(build_left_transition_state(0.20f).content == Content::Icons);
}

TEST_CASE("status bar left content never requests icons and title together")
{
	for (float progress = -0.25f; progress <= 1.25f; progress += 0.05f) {
		check_exclusive_content(progress);
	}
}

TEST_CASE("status bar overlay covers only the ambiguous transition band")
{
	const auto main_side = build_left_transition_state(0.10f);
	const auto early_enter = build_left_transition_state(0.20f);
	const auto entering_cover = build_left_transition_state(0.38f);
	const auto leaving_cover = build_left_transition_state(0.62f);
	const auto late_exit = build_left_transition_state(0.80f);
	const auto settings_side = build_left_transition_state(0.90f);

	CHECK(main_side.overlay_opa == 0);
	CHECK(early_enter.overlay_opa > main_side.overlay_opa);
	CHECK(entering_cover.overlay_opa > main_side.overlay_opa);
	CHECK(entering_cover.overlay_opa == leaving_cover.overlay_opa);
	CHECK(late_exit.overlay_opa > settings_side.overlay_opa);
	CHECK(settings_side.overlay_opa == 0);
}
