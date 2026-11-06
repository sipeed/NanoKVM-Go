#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/launcher_exit_gesture.hpp"

using lvgl_main::LauncherExitGestureSide;
using lvgl_main::launcher_exit_gesture_reached;
using lvgl_main::launcher_exit_gesture_start_side;

TEST_CASE("Launcher exit gesture can start from either screen edge")
{
	CHECK(launcher_exit_gesture_start_side(0, 320, 20) ==
	      LauncherExitGestureSide::Left);
	CHECK(launcher_exit_gesture_start_side(19, 320, 20) ==
	      LauncherExitGestureSide::Left);
	CHECK(launcher_exit_gesture_start_side(20, 320, 20) ==
	      LauncherExitGestureSide::None);
	CHECK(launcher_exit_gesture_start_side(299, 320, 20) ==
	      LauncherExitGestureSide::None);
	CHECK(launcher_exit_gesture_start_side(300, 320, 20) ==
	      LauncherExitGestureSide::Right);
	CHECK(launcher_exit_gesture_start_side(319, 320, 20) ==
	      LauncherExitGestureSide::Right);
}

TEST_CASE("Launcher exit gesture reaches confirmation after one eighth inward")
{
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Left, 10, 49, 320));
	CHECK(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Left, 10, 50, 320));
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Right, 310, 271, 320));
	CHECK(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Right, 310, 270, 320));
}

TEST_CASE("Launcher exit gesture ignores edge taps and short swipes")
{
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Left, 39, 39, 320));
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Right, 300, 300, 320));
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Left, 5, 30, 320));
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Right, 315, 290, 320));
}

TEST_CASE("Launcher exit gesture rejects invalid input")
{
	CHECK(launcher_exit_gesture_start_side(0, 0, 20) ==
	      LauncherExitGestureSide::None);
	CHECK(launcher_exit_gesture_start_side(-1, 320, 20) ==
	      LauncherExitGestureSide::None);
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::None, 0, 160, 320));
	CHECK_FALSE(launcher_exit_gesture_reached(
		LauncherExitGestureSide::Left, 0, 100, 0));
}
