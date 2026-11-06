#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/launcher_exit_confirmation.hpp"

using lvgl_main::LauncherExitConfirmationChoice;
using lvgl_main::launcher_exit_confirmation_animation_frame;
using lvgl_main::launcher_exit_confirmation_choice;
using lvgl_main::launcher_exit_confirmation_layout;

TEST_CASE("Launcher exit confirmation is centered with separate choices")
{
	const auto layout = launcher_exit_confirmation_layout(284, 240);

	REQUIRE(layout.valid);
	CHECK(layout.dialog.x == 32);
	CHECK(layout.dialog.y == 56);
	CHECK(layout.dialog.width == 220);
	CHECK(layout.dialog.height == 128);
	CHECK(layout.no_button.x == 48);
	CHECK(layout.yes_button.x == 148);
}

TEST_CASE("Launcher exit confirmation hit tests No and Yes buttons")
{
	const auto layout = launcher_exit_confirmation_layout(284, 240);

	CHECK(launcher_exit_confirmation_choice(layout, 92, 148) ==
	      LauncherExitConfirmationChoice::No);
	CHECK(launcher_exit_confirmation_choice(layout, 192, 148) ==
	      LauncherExitConfirmationChoice::Yes);
	CHECK(launcher_exit_confirmation_choice(layout, 142, 148) ==
	      LauncherExitConfirmationChoice::None);
	CHECK(launcher_exit_confirmation_choice(layout, 142, 80) ==
	      LauncherExitConfirmationChoice::None);
}

TEST_CASE("Launcher exit confirmation rejects a display that is too small")
{
	const auto layout = launcher_exit_confirmation_layout(100, 80);
	CHECK_FALSE(layout.valid);
	CHECK(launcher_exit_confirmation_choice(layout, 50, 50) ==
	      LauncherExitConfirmationChoice::None);
}

TEST_CASE("Launcher exit confirmation animation eases to full size")
{
	auto previous = launcher_exit_confirmation_animation_frame(0);
	CHECK(previous.scale_percent < 100);
	CHECK(previous.opacity < 255);

	for (std::size_t index = 1;
	     index < lvgl_main::LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES;
	     ++index) {
		const auto current =
			launcher_exit_confirmation_animation_frame(index);
		CHECK(current.scale_percent >= previous.scale_percent);
		CHECK(current.opacity > previous.opacity);
		previous = current;
	}

	CHECK(previous.scale_percent == 100);
	CHECK(previous.opacity == 255);
	CHECK(launcher_exit_confirmation_animation_frame(100).scale_percent ==
	      100);
}
