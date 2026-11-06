#include "lvgl_main/launcher_exit_confirmation.hpp"

#include <algorithm>
#include <array>

namespace lvgl_main
{

bool LauncherExitRect::contains(int point_x, int point_y) const
{
	return point_x >= x && point_y >= y && point_x < x + width &&
	       point_y < y + height;
}

LauncherExitConfirmationLayout
launcher_exit_confirmation_layout(int width, int height)
{
	if (width < 120 || height < 100) {
		return {};
	}

	constexpr int margin = 16;
	constexpr int button_gap = 12;
	constexpr int button_height = 40;
	LauncherExitConfirmationLayout layout;
	layout.dialog.width = std::min(220, width - 2 * margin);
	layout.dialog.height = std::min(128, height - 2 * margin);
	layout.dialog.x = (width - layout.dialog.width) / 2;
	layout.dialog.y = (height - layout.dialog.height) / 2;

	const int button_width =
		(layout.dialog.width - 2 * margin - button_gap) / 2;
	const int button_y = layout.dialog.y + layout.dialog.height - margin -
			     button_height;
	layout.no_button = {
		layout.dialog.x + margin,
		button_y,
		button_width,
		button_height,
	};
	layout.yes_button = {
		layout.no_button.x + button_width + button_gap,
		button_y,
		button_width,
		button_height,
	};
	layout.valid = button_width > 0;
	return layout;
}

LauncherExitConfirmationChoice launcher_exit_confirmation_choice(
	const LauncherExitConfirmationLayout &layout, int logical_x,
	int logical_y)
{
	if (!layout.valid) {
		return LauncherExitConfirmationChoice::None;
	}
	if (layout.no_button.contains(logical_x, logical_y)) {
		return LauncherExitConfirmationChoice::No;
	}
	if (layout.yes_button.contains(logical_x, logical_y)) {
		return LauncherExitConfirmationChoice::Yes;
	}
	return LauncherExitConfirmationChoice::None;
}

LauncherExitConfirmationAnimationFrame
launcher_exit_confirmation_animation_frame(std::size_t index)
{
	static constexpr std::array<LauncherExitConfirmationAnimationFrame,
				    LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES>
		frames { {
			{ 82, 48 },
			{ 87, 88 },
			{ 92, 136 },
			{ 96, 184 },
			{ 99, 224 },
			{ 100, 255 },
		} };
	return frames[std::min(index, frames.size() - 1)];
}

} // namespace lvgl_main
