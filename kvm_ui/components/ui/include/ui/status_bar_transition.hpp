#pragma once

#include <cstdint>

namespace ui::status_bar_transition
{

enum class LeftContent : uint8_t {
	Icons,
	SettingsTitle,
};

struct LeftTransitionState {
	LeftContent content { LeftContent::Icons };
	bool show_icons { true };
	bool show_title { false };
	uint8_t overlay_opa { 0 };
};

LeftTransitionState build_left_transition_state(float settings_visible_ratio);

} // namespace ui::status_bar_transition
