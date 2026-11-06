#pragma once

#include <cstddef>
#include <cstdint>

namespace lvgl_main
{

struct LauncherExitRect {
	int x { 0 };
	int y { 0 };
	int width { 0 };
	int height { 0 };

	[[nodiscard]] bool contains(int point_x, int point_y) const;
};

struct LauncherExitConfirmationLayout {
	LauncherExitRect dialog;
	LauncherExitRect no_button;
	LauncherExitRect yes_button;
	bool valid { false };
};

enum class LauncherExitConfirmationChoice {
	None,
	No,
	Yes,
};

struct LauncherExitConfirmationAnimationFrame {
	uint8_t scale_percent { 100 };
	uint8_t opacity { 255 };
};

inline constexpr std::size_t LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES = 6;

[[nodiscard]] LauncherExitConfirmationLayout
launcher_exit_confirmation_layout(int width, int height);
[[nodiscard]] LauncherExitConfirmationChoice launcher_exit_confirmation_choice(
	const LauncherExitConfirmationLayout &layout, int logical_x,
	int logical_y);
[[nodiscard]] LauncherExitConfirmationAnimationFrame
launcher_exit_confirmation_animation_frame(std::size_t index);

} // namespace lvgl_main
