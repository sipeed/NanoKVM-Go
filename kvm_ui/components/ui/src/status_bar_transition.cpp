#include "ui/status_bar_transition.hpp"

#include <algorithm>

namespace ui::status_bar_transition
{

namespace
{

constexpr float FADE_START = 0.12f;
constexpr float FADE_IN_END = 0.38f;
constexpr float CONTENT_SWITCH = 0.50f;
constexpr float FADE_OUT_START = 0.62f;
constexpr float FADE_END = 0.88f;
constexpr uint8_t OPA_TRANSP = 0;
constexpr uint8_t OPA_COVER = 255;

uint8_t map_overlay_opa(float progress)
{
	if (progress <= FADE_START) {
		return OPA_TRANSP;
	}
	if (progress <= FADE_IN_END) {
		const float t =
			(progress - FADE_START) / (FADE_IN_END - FADE_START);
		return static_cast<uint8_t>(t * OPA_COVER);
	}
	if (progress <= FADE_OUT_START) {
		return OPA_COVER;
	}
	if (progress <= FADE_END) {
		const float t = (progress - FADE_OUT_START) /
				(FADE_END - FADE_OUT_START);
		return static_cast<uint8_t>((1.0f - t) * OPA_COVER);
	}
	return OPA_TRANSP;
}

} // namespace

LeftTransitionState build_left_transition_state(float settings_visible_ratio)
{
	const float clamped = std::clamp(settings_visible_ratio, 0.0f, 1.0f);
	const bool settings_dominant = clamped >= CONTENT_SWITCH;

	return LeftTransitionState {
		.content = settings_dominant ? LeftContent::SettingsTitle :
					       LeftContent::Icons,
		.show_icons = !settings_dominant,
		.show_title = settings_dominant,
		.overlay_opa = map_overlay_opa(clamped),
	};
}

} // namespace ui::status_bar_transition
