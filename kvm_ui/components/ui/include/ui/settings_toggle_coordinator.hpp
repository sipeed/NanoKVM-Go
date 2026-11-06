#pragma once

#include <array>
#include <cstdint>
#include <functional>
#include <memory>
#include <optional>

#include <lvgl.h>

#include "ui/settings_service_runtime_state.hpp"
#include "ui/settings_state_action_policy.hpp"
#include "ui/settings_toggle_id.hpp"

namespace ui
{

struct SettingsToggleViewModel {
	bool confirmed { false };
	bool available { true };
	bool boot_pending { false };
	bool action_pending { false };
	bool pending_target { false };

	bool is_pending() const noexcept
	{
		return boot_pending || action_pending;
	}
};

class SettingsToggleCoordinator final {
    public:
	using ActionCallback = std::function<void(bool)>;
	using ActionDispatcher = std::function<bool(SettingsToggleId, bool)>;
	using ViewModelChangeCallback = std::function<void(SettingsToggleId)>;

	explicit SettingsToggleCoordinator(uint32_t boot_pending_timeout_ms);
	~SettingsToggleCoordinator();

	SettingsToggleCoordinator(const SettingsToggleCoordinator &) = delete;
	SettingsToggleCoordinator &
	operator=(const SettingsToggleCoordinator &) = delete;

	void set_view_model_change_callback(ViewModelChangeCallback callback);
	void start_boot_pending();
	void set_confirmed(SettingsToggleId id, bool confirmed);
	void set_available(SettingsToggleId id, bool available);
	void set_auto_rotation_manual_mode(bool manual_mode);
	bool request_change(SettingsToggleId id, bool target_active,
				    ActionCallback callback);
	bool request_change(
		SettingsToggleId id, bool target_active,
		settings_state_action_policy::PendingRequestPolicy policy,
		ActionDispatcher dispatcher);
	bool apply_service_runtime_state(
		const SettingsServiceRuntimeState &state);

	SettingsToggleViewModel view_model(SettingsToggleId id) const;
	bool is_available(SettingsToggleId id) const;
	bool is_boot_pending(SettingsToggleId id) const;
	uint32_t pending_timeout_ms(SettingsToggleId id) const;
	void shutdown();

    private:
	struct ToggleState {
		bool confirmed { false };
		bool available { true };
		bool boot_pending { false };
		bool action_pending { false };
		bool in_flight_target { false };
		bool pending_target { false };
		bool min_loading_elapsed { false };
		bool max_wait_elapsed { false };
		settings_state_action_policy::PendingRequestPolicy request_policy {
			settings_state_action_policy::PendingRequestPolicy::
				RejectWhilePending
		};
		uint64_t intent_generation { 0 };
		uint64_t dispatched_generation { 0 };
		ActionDispatcher dispatcher;
		ActionDispatcher delayed_dispatcher;
		lv_timer_t *min_loading_timer = nullptr;
		lv_timer_t *max_wait_timer = nullptr;
	};

	struct DispatchRequest {
		SettingsToggleId id { SettingsToggleId::MCP };
		bool target_active { false };
		uint64_t generation { 0 };
		ActionDispatcher dispatcher;
	};

	enum class DispatchResult : uint8_t {
		Dispatched,
		Missing,
		OwnerDestroyed,
		Superseded,
	};

	std::array<ToggleState, kSettingsToggleCount> _states {};
	uint32_t _boot_pending_timeout_ms = 0;
	uint64_t _service_event_seq = 0;
	bool _auto_rotation_manual_mode = false;
	bool _is_shutting_down = false;
	std::shared_ptr<void> _callback_lifetime { std::make_shared<int>(0) };
	lv_timer_t *_boot_pending_timer = nullptr;
	ViewModelChangeCallback _on_view_model_change;
	uint32_t _notification_batch_depth = 0;
	std::array<bool, kSettingsToggleCount> _pending_notifications {};

	static bool default_available(SettingsToggleId id) noexcept;
	static void boot_pending_timer_cb(lv_timer_t *timer);
	static void pending_timer_cb(lv_timer_t *timer);

	ToggleState *state_for(SettingsToggleId id) noexcept;
	const ToggleState *state_for(SettingsToggleId id) const noexcept;
	bool any_boot_pending() const noexcept;
	std::optional<DispatchRequest>
	finish_action_pending_if_target(SettingsToggleId id);
	bool finish_action_pending(SettingsToggleId id);
	void start_action_timers(SettingsToggleId id,
			 settings_state_action_policy::PendingRequestPolicy policy);
	void cancel_action_timers(ToggleState &state);
	void cancel_boot_pending_timer();
	void finish_all_boot_pending();
	void handle_pending_timer(lv_timer_t *timer);
	bool dispatch_is_current(const DispatchRequest &request) const noexcept;
	DispatchResult dispatch_action(DispatchRequest request);
	std::optional<DispatchRequest>
	make_dispatch_request(SettingsToggleId id, ToggleState &state);
	void begin_notification_batch() noexcept;
	void end_notification_batch();
	void notify(SettingsToggleId id);
};

} // namespace ui
