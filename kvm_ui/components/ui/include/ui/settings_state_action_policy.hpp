#pragma once

#include <cstdint>

#include "ui/settings_toggle_id.hpp"

namespace ui::settings_state_action_policy
{

inline constexpr uint32_t kSettingsStateMinLoadingMs = 2000;
inline constexpr uint32_t kSettingsServicePendingTimeoutMs = 12000;
inline constexpr uint32_t kSettingsStateDefaultPendingTimeoutMs = 10000;
inline constexpr uint32_t kSettingsStateSlowPendingTimeoutMs = 20000;
inline constexpr uint32_t kAutoRotationPendingTimeoutMs = 500;

enum class PendingDecision : uint8_t {
	KeepWaiting,
	Finish,
};

enum class PendingRequestPolicy : uint8_t {
	RejectWhilePending,
	CoalesceLatest,
};

enum class MissingCallbackAction : uint8_t {
	Ignore,
};

constexpr bool waits_for_service_state(SettingsToggleId button)
{
	return button == SettingsToggleId::MCP ||
	       button == SettingsToggleId::REPLAY;
}

constexpr bool allows_coalesced_request(SettingsToggleId button) noexcept
{
	return button == SettingsToggleId::MCP ||
	       button == SettingsToggleId::REPLAY ||
	       button == SettingsToggleId::NCM || button == SettingsToggleId::SSH;
}

constexpr bool uses_optimistic_action_presentation(
	SettingsToggleId button, bool option_enabled) noexcept
{
	return option_enabled && allows_coalesced_request(button);
}

constexpr bool submits_action_after_min_loading(SettingsToggleId button)
{
	return button == SettingsToggleId::AUTO_ROTATION;
}

constexpr bool requires_min_loading(PendingRequestPolicy policy)
{
	return policy == PendingRequestPolicy::RejectWhilePending;
}

constexpr uint32_t min_loading_ms(SettingsToggleId button)
{
	if (button == SettingsToggleId::AUTO_ROTATION) {
		return kAutoRotationPendingTimeoutMs;
	}
	return kSettingsStateMinLoadingMs;
}

constexpr uint32_t max_pending_wait_ms(SettingsToggleId button)
{
	switch (button) {
	case SettingsToggleId::MCP:
	case SettingsToggleId::REPLAY:
		return kSettingsServicePendingTimeoutMs;
	case SettingsToggleId::SSH:
		return kSettingsStateSlowPendingTimeoutMs;
	case SettingsToggleId::AUTO_ROTATION:
		return kAutoRotationPendingTimeoutMs;
	case SettingsToggleId::NCM:
		return kSettingsStateDefaultPendingTimeoutMs;
	}

	return kSettingsStateDefaultPendingTimeoutMs;
}

constexpr bool preview_active(bool current_active, bool target_active,
			      bool pending)
{
	return pending ? target_active : current_active;
}

constexpr PendingDecision pending_decision(SettingsToggleId button,
						   bool target_active,
						   bool runtime_active,
						   bool min_loading_elapsed,
						   bool max_wait_elapsed,
						   PendingRequestPolicy policy =
							   PendingRequestPolicy::RejectWhilePending)
{
	(void)button;
	if (max_wait_elapsed) {
		return PendingDecision::Finish;
	}
	if (runtime_active == target_active &&
	    (!requires_min_loading(policy) || min_loading_elapsed)) {
		return PendingDecision::Finish;
	}
	return PendingDecision::KeepWaiting;
}

constexpr MissingCallbackAction missing_callback_action(SettingsToggleId button)
{
	switch (button) {
	case SettingsToggleId::MCP:
	case SettingsToggleId::REPLAY:
	case SettingsToggleId::NCM:
	case SettingsToggleId::SSH:
	case SettingsToggleId::AUTO_ROTATION:
		return MissingCallbackAction::Ignore;
	}

	return MissingCallbackAction::Ignore;
}

constexpr bool allows_missing_callback_local_toggle(SettingsToggleId button)
{
	return missing_callback_action(button) != MissingCallbackAction::Ignore;
}

} // namespace ui::settings_state_action_policy
