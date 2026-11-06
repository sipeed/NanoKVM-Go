#include "lvgl_main/launcher_exit_gesture.hpp"

#include <algorithm>

namespace lvgl_main
{

LauncherExitGestureSide launcher_exit_gesture_start_side(
	int logical_x, int horizontal_resolution, int edge_zone)
{
	if (horizontal_resolution <= 0 || edge_zone <= 0 || logical_x < 0 ||
	    logical_x >= horizontal_resolution) {
		return LauncherExitGestureSide::None;
	}

	const int effective_zone =
		std::min(edge_zone, horizontal_resolution / 2);
	if (logical_x < effective_zone) {
		return LauncherExitGestureSide::Left;
	}
	if (logical_x >= horizontal_resolution - effective_zone) {
		return LauncherExitGestureSide::Right;
	}
	return LauncherExitGestureSide::None;
}

bool launcher_exit_gesture_reached(LauncherExitGestureSide side,
				   int start_x, int current_x,
				   int horizontal_resolution)
{
	if (horizontal_resolution <= 0 || start_x < 0 || current_x < 0 ||
	    start_x >= horizontal_resolution ||
	    current_x >= horizontal_resolution) {
		return false;
	}

	const double inward_threshold = horizontal_resolution / 8.0;
	if (side == LauncherExitGestureSide::Left) {
		return current_x - start_x >= inward_threshold;
	}
	if (side == LauncherExitGestureSide::Right) {
		return start_x - current_x >= inward_threshold;
	}
	return false;
}

} // namespace lvgl_main
