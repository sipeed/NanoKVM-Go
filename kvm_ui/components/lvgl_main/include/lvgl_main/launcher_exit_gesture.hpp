#pragma once

namespace lvgl_main
{

enum class LauncherExitGestureSide {
	None,
	Left,
	Right,
};

[[nodiscard]] LauncherExitGestureSide launcher_exit_gesture_start_side(
	int logical_x, int horizontal_resolution, int edge_zone);
[[nodiscard]] bool launcher_exit_gesture_reached(
	LauncherExitGestureSide side, int start_x, int current_x,
	int horizontal_resolution);

} // namespace lvgl_main
