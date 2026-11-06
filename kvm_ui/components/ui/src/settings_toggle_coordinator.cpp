#include "ui/settings_toggle_coordinator.hpp"

#include <utility>

#include <spdlog/spdlog.h>

#include "ui/settings_state_action_policy.hpp"

namespace ui
{

namespace
{

std::string_view settings_toggle_log_name(SettingsToggleId id)
{
	switch (id) {
	case SettingsToggleId::MCP:
		return "MCP service";
	case SettingsToggleId::REPLAY:
		return "Replay service";
	case SettingsToggleId::NCM:
		return "NCM";
	case SettingsToggleId::SSH:
		return "SSH";
	case SettingsToggleId::AUTO_ROTATION:
		return "Auto rotation";
	}

	return "Settings state";
}

} // namespace

SettingsToggleCoordinator::SettingsToggleCoordinator(
	uint32_t boot_pending_timeout_ms)
	: _boot_pending_timeout_ms(boot_pending_timeout_ms)
{
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		if (ToggleState *state = state_for(id)) {
			state->available = default_available(id);
		}
	}
}

SettingsToggleCoordinator::~SettingsToggleCoordinator()
{
	shutdown();
}

void SettingsToggleCoordinator::set_view_model_change_callback(
	ViewModelChangeCallback callback)
{
	_on_view_model_change = std::move(callback);
}

void SettingsToggleCoordinator::start_boot_pending()
{
	if (_is_shutting_down) {
		return;
	}

	begin_notification_batch();
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		ToggleState *state = state_for(id);
		if (state == nullptr) {
			continue;
		}
		state->boot_pending = true;
		notify(id);
	}
	end_notification_batch();

	cancel_boot_pending_timer();
	_boot_pending_timer = lv_timer_create(boot_pending_timer_cb,
					      _boot_pending_timeout_ms, this);
	if (_boot_pending_timer != nullptr) {
		lv_timer_set_repeat_count(_boot_pending_timer, 1);
	}
}

void SettingsToggleCoordinator::set_confirmed(SettingsToggleId id,
					      bool confirmed)
{
	ToggleState *state = state_for(id);
	if (state == nullptr || _is_shutting_down) {
		return;
	}

	state->confirmed = confirmed;
	state->boot_pending = false;
	if (!any_boot_pending()) {
		cancel_boot_pending_timer();
	}
	std::optional<DispatchRequest> follow_up =
		finish_action_pending_if_target(id);
	const std::weak_ptr<void> callback_lifetime = _callback_lifetime;
	notify(id);
	if (callback_lifetime.expired() || !follow_up) {
		return;
	}
	(void)dispatch_action(std::move(*follow_up));
}

void SettingsToggleCoordinator::set_available(SettingsToggleId id,
					      bool available)
{
	ToggleState *state = state_for(id);
	if (state == nullptr || _is_shutting_down) {
		return;
	}

	state->available = available;
	if (!available && !state->boot_pending) {
		(void)finish_action_pending(id);
	}
	notify(id);
}

void SettingsToggleCoordinator::set_auto_rotation_manual_mode(bool manual_mode)
{
	_auto_rotation_manual_mode = manual_mode;
}

bool SettingsToggleCoordinator::request_change(SettingsToggleId id,
					       bool target_active,
					       ActionCallback callback)
{
	ActionDispatcher dispatcher;
	if (callback) {
		dispatcher = [callback = std::move(callback)](SettingsToggleId,
							     bool target) {
			callback(target);
			return true;
		};
	}

	return request_change(
		id, target_active,
		settings_state_action_policy::PendingRequestPolicy::
			RejectWhilePending,
		std::move(dispatcher));
}

bool SettingsToggleCoordinator::request_change(
	SettingsToggleId id, bool target_active,
	settings_state_action_policy::PendingRequestPolicy policy,
	ActionDispatcher dispatcher)
{
	if (policy == settings_state_action_policy::PendingRequestPolicy::
			      CoalesceLatest &&
	    !settings_state_action_policy::allows_coalesced_request(id)) {
		policy = settings_state_action_policy::PendingRequestPolicy::
			RejectWhilePending;
	}

	ToggleState *state = state_for(id);
	if (state == nullptr || _is_shutting_down ||
	    (is_settings_service_toggle(id) && !state->available)) {
		return false;
	}
	if (state->action_pending) {
		if (policy != settings_state_action_policy::PendingRequestPolicy::
				      CoalesceLatest ||
		    state->request_policy !=
				    settings_state_action_policy::PendingRequestPolicy::
					    CoalesceLatest ||
		    !dispatcher) {
			return false;
		}

		if (state->pending_target == target_active) {
			state->dispatcher = std::move(dispatcher);
			return true;
		}

		state->pending_target = target_active;
		++state->intent_generation;
		state->dispatcher = std::move(dispatcher);
		notify(id);
		return true;
	}

	state->action_pending = true;
	state->in_flight_target = target_active;
	state->pending_target = target_active;
	state->request_policy = policy;
	++state->intent_generation;
	state->dispatched_generation = state->intent_generation;
	state->min_loading_elapsed = false;
	state->max_wait_elapsed = false;
	state->dispatcher = std::move(dispatcher);
	state->delayed_dispatcher = nullptr;
	start_action_timers(id, policy);

	if (policy == settings_state_action_policy::PendingRequestPolicy::
			      RejectWhilePending &&
	    _auto_rotation_manual_mode &&
	    settings_state_action_policy::submits_action_after_min_loading(id) &&
	    state->dispatcher) {
		state->delayed_dispatcher = state->dispatcher;
		notify(id);
		return true;
	}

	std::optional<DispatchRequest> dispatch = make_dispatch_request(id, *state);
	const std::weak_ptr<void> callback_lifetime = _callback_lifetime;
	notify(id);
	if (callback_lifetime.expired()) {
		return true;
	}
	if (!dispatch) {
		return false;
	}

	return dispatch_action(std::move(*dispatch)) != DispatchResult::Missing;
}

bool SettingsToggleCoordinator::apply_service_runtime_state(
	const SettingsServiceRuntimeState &state)
{
	if (_is_shutting_down ||
	    (state.event_seq != 0 && state.event_seq <= _service_event_seq)) {
		return false;
	}
	if (state.event_seq != 0) {
		_service_event_seq = state.event_seq;
	}

	begin_notification_batch();
	if (state.known && !state.refresh_failed) {
		set_available(SettingsToggleId::MCP, state.mcp_available);
		set_confirmed(SettingsToggleId::MCP, state.mcp_active);
		set_available(SettingsToggleId::REPLAY, state.replay_available);
		set_confirmed(SettingsToggleId::REPLAY, state.replay_active);
	} else if (!state.known) {
		set_available(SettingsToggleId::MCP, false);
		set_available(SettingsToggleId::REPLAY, false);
	}
	end_notification_batch();
	return true;
}

SettingsToggleViewModel
SettingsToggleCoordinator::view_model(SettingsToggleId id) const
{
	const ToggleState *state = state_for(id);
	if (state == nullptr) {
		return {};
	}

	return {
		.confirmed = state->confirmed,
		.available = state->available,
		.boot_pending = state->boot_pending,
		.action_pending = state->action_pending,
		.pending_target = state->pending_target,
	};
}

bool SettingsToggleCoordinator::is_available(SettingsToggleId id) const
{
	const ToggleState *state = state_for(id);
	return state != nullptr && state->available;
}

bool SettingsToggleCoordinator::is_boot_pending(SettingsToggleId id) const
{
	const ToggleState *state = state_for(id);
	return state != nullptr && state->boot_pending;
}

uint32_t
SettingsToggleCoordinator::pending_timeout_ms(SettingsToggleId id) const
{
	const ToggleState *state = state_for(id);
	if (state != nullptr && state->boot_pending && !state->action_pending) {
		return _boot_pending_timeout_ms;
	}
	return settings_state_action_policy::max_pending_wait_ms(id);
}

void SettingsToggleCoordinator::shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	_callback_lifetime.reset();
	cancel_boot_pending_timer();
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		(void)finish_action_pending(id);
	}
	_on_view_model_change = nullptr;
	_pending_notifications.fill(false);
	_notification_batch_depth = 0;
}

bool SettingsToggleCoordinator::default_available(SettingsToggleId id) noexcept
{
	return !is_settings_service_toggle(id);
}

void SettingsToggleCoordinator::boot_pending_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}

	auto *self = static_cast<SettingsToggleCoordinator *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down ||
	    self->_boot_pending_timer != timer) {
		return;
	}

	self->_boot_pending_timer = nullptr;
	self->finish_all_boot_pending();
}

void SettingsToggleCoordinator::pending_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}

	auto *self = static_cast<SettingsToggleCoordinator *>(
		lv_timer_get_user_data(timer));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}
	self->handle_pending_timer(timer);
}

SettingsToggleCoordinator::ToggleState *
SettingsToggleCoordinator::state_for(SettingsToggleId id) noexcept
{
	const std::size_t index = to_settings_toggle_index(id);
	return index < _states.size() ? &_states[index] : nullptr;
}

const SettingsToggleCoordinator::ToggleState *
SettingsToggleCoordinator::state_for(SettingsToggleId id) const noexcept
{
	const std::size_t index = to_settings_toggle_index(id);
	return index < _states.size() ? &_states[index] : nullptr;
}

bool SettingsToggleCoordinator::any_boot_pending() const noexcept
{
	for (const ToggleState &state : _states) {
		if (state.boot_pending) {
			return true;
		}
	}
	return false;
}

std::optional<SettingsToggleCoordinator::DispatchRequest>
SettingsToggleCoordinator::finish_action_pending_if_target(SettingsToggleId id)
{
	ToggleState *state = state_for(id);
	if (state == nullptr || !state->action_pending) {
		return std::nullopt;
	}

	const auto decision = settings_state_action_policy::pending_decision(
		id, state->in_flight_target, state->confirmed,
		state->min_loading_elapsed, state->max_wait_elapsed,
		state->request_policy);
	if (decision != settings_state_action_policy::PendingDecision::Finish) {
		return std::nullopt;
	}

	const bool newer_intent =
		state->intent_generation > state->dispatched_generation;
	const bool needs_action = state->pending_target != state->confirmed;
	if (state->request_policy ==
		    settings_state_action_policy::PendingRequestPolicy::CoalesceLatest &&
	    newer_intent && needs_action && state->available && state->dispatcher) {
		state->in_flight_target = state->pending_target;
		state->dispatched_generation = state->intent_generation;
		start_action_timers(id, state->request_policy);
		return make_dispatch_request(id, *state);
	}

	(void)finish_action_pending(id);
	return std::nullopt;
}

bool SettingsToggleCoordinator::finish_action_pending(SettingsToggleId id)
{
	ToggleState *state = state_for(id);
	if (state == nullptr) {
		return false;
	}

	const bool changed = state->action_pending ||
			     state->min_loading_elapsed ||
			     state->max_wait_elapsed ||
			     static_cast<bool>(state->dispatcher) ||
			     static_cast<bool>(state->delayed_dispatcher);
	state->action_pending = false;
	state->in_flight_target = state->confirmed;
	state->pending_target = state->confirmed;
	state->min_loading_elapsed = false;
	state->max_wait_elapsed = false;
	state->request_policy = settings_state_action_policy::PendingRequestPolicy::
		RejectWhilePending;
	state->dispatched_generation = state->intent_generation;
	state->dispatcher = nullptr;
	state->delayed_dispatcher = nullptr;
	cancel_action_timers(*state);
	return changed;
}

void SettingsToggleCoordinator::start_action_timers(
	SettingsToggleId id,
	settings_state_action_policy::PendingRequestPolicy policy)
{
	ToggleState *state = state_for(id);
	if (state == nullptr) {
		return;
	}

	cancel_action_timers(*state);
	state->min_loading_elapsed = false;
	state->max_wait_elapsed = false;
	if (settings_state_action_policy::requires_min_loading(policy)) {
		state->min_loading_timer = lv_timer_create(
			pending_timer_cb,
			settings_state_action_policy::min_loading_ms(id), this);
		if (state->min_loading_timer != nullptr) {
			lv_timer_set_repeat_count(state->min_loading_timer, 1);
		}
	}
	state->max_wait_timer = lv_timer_create(
		pending_timer_cb,
		settings_state_action_policy::max_pending_wait_ms(id), this);
	if (state->max_wait_timer != nullptr) {
		lv_timer_set_repeat_count(state->max_wait_timer, 1);
	}
}

void SettingsToggleCoordinator::cancel_action_timers(ToggleState &state)
{
	if (state.min_loading_timer != nullptr) {
		lv_timer_delete(state.min_loading_timer);
		state.min_loading_timer = nullptr;
	}
	if (state.max_wait_timer != nullptr) {
		lv_timer_delete(state.max_wait_timer);
		state.max_wait_timer = nullptr;
	}
}

void SettingsToggleCoordinator::cancel_boot_pending_timer()
{
	if (_boot_pending_timer != nullptr) {
		lv_timer_delete(_boot_pending_timer);
		_boot_pending_timer = nullptr;
	}
}

void SettingsToggleCoordinator::finish_all_boot_pending()
{
	begin_notification_batch();
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		ToggleState *state = state_for(id);
		if (state == nullptr || !state->boot_pending) {
			continue;
		}
		state->boot_pending = false;
		if (is_settings_service_toggle(id) && !state->available) {
			(void)finish_action_pending(id);
		}
		notify(id);
	}
	end_notification_batch();
}

void SettingsToggleCoordinator::handle_pending_timer(lv_timer_t *timer)
{
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		ToggleState *state = state_for(id);
		if (state == nullptr) {
			continue;
		}
		if (state->min_loading_timer == timer) {
			state->min_loading_timer = nullptr;
			state->min_loading_elapsed = true;
			ActionDispatcher delayed_dispatcher =
				std::move(state->delayed_dispatcher);
			state->delayed_dispatcher = nullptr;
			if (delayed_dispatcher) {
				const DispatchResult result = dispatch_action({
					.id = id,
					.target_active = state->in_flight_target,
					.generation = state->dispatched_generation,
					.dispatcher = std::move(delayed_dispatcher),
				});
				if (result != DispatchResult::Dispatched) {
					return;
				}
			}

			state = state_for(id);
			if (state == nullptr || !state->action_pending) {
				return;
			}
			std::optional<DispatchRequest> follow_up =
				finish_action_pending_if_target(id);
			if (!follow_up && state->action_pending) {
				return;
			}
			const std::weak_ptr<void> callback_lifetime =
				_callback_lifetime;
			notify(id);
			if (callback_lifetime.expired() || !follow_up) {
				return;
			}
			(void)dispatch_action(std::move(*follow_up));
			return;
		}
		if (state->max_wait_timer == timer) {
			state->max_wait_timer = nullptr;
			state->max_wait_elapsed = true;
			ActionDispatcher delayed_dispatcher =
				std::move(state->delayed_dispatcher);
			state->delayed_dispatcher = nullptr;
			if (delayed_dispatcher) {
				const DispatchResult result = dispatch_action({
					.id = id,
					.target_active = state->in_flight_target,
					.generation = state->dispatched_generation,
					.dispatcher = std::move(delayed_dispatcher),
				});
				if (result != DispatchResult::Dispatched) {
					return;
				}
			}

			state = state_for(id);
			if (state == nullptr || !state->action_pending) {
				return;
			}
			std::optional<DispatchRequest> follow_up =
				finish_action_pending_if_target(id);
			if (!follow_up && state->action_pending) {
				return;
			}
			const std::weak_ptr<void> callback_lifetime =
				_callback_lifetime;
			notify(id);
			if (callback_lifetime.expired() || !follow_up) {
				return;
			}
			(void)dispatch_action(std::move(*follow_up));
			return;
		}
	}
}

bool SettingsToggleCoordinator::dispatch_is_current(
	const DispatchRequest &request) const noexcept
{
	const ToggleState *state = state_for(request.id);
	return state != nullptr && state->action_pending &&
	       state->dispatched_generation == request.generation &&
	       state->in_flight_target == request.target_active;
}

SettingsToggleCoordinator::DispatchResult
SettingsToggleCoordinator::dispatch_action(DispatchRequest request)
{
	if (!dispatch_is_current(request)) {
		return DispatchResult::Superseded;
	}
	if (!request.dispatcher) {
		spdlog::warn("{} toggle ignored: runtime callback missing",
			     settings_toggle_log_name(request.id));
		if (dispatch_is_current(request)) {
			(void)finish_action_pending(request.id);
			notify(request.id);
		}
		return DispatchResult::Missing;
	}

	const std::weak_ptr<void> callback_lifetime = _callback_lifetime;
	const bool dispatched =
		request.dispatcher(request.id, request.target_active);
	if (callback_lifetime.expired()) {
		return DispatchResult::OwnerDestroyed;
	}
	if (dispatched) {
		return DispatchResult::Dispatched;
	}

	spdlog::warn("{} toggle ignored: runtime callback missing",
		     settings_toggle_log_name(request.id));
	if (dispatch_is_current(request)) {
		(void)finish_action_pending(request.id);
		notify(request.id);
	}
	return DispatchResult::Missing;
}

std::optional<SettingsToggleCoordinator::DispatchRequest>
SettingsToggleCoordinator::make_dispatch_request(SettingsToggleId id,
							   ToggleState &state)
{
	return DispatchRequest {
		.id = id,
		.target_active = state.in_flight_target,
		.generation = state.dispatched_generation,
		.dispatcher = state.dispatcher,
	};
}

void SettingsToggleCoordinator::begin_notification_batch() noexcept
{
	++_notification_batch_depth;
}

void SettingsToggleCoordinator::end_notification_batch()
{
	if (_notification_batch_depth == 0) {
		return;
	}
	--_notification_batch_depth;
	if (_notification_batch_depth != 0 || !_on_view_model_change ||
	    _is_shutting_down) {
		return;
	}

	SettingsToggleId changed_id = SettingsToggleId::MCP;
	bool has_change = false;
	for (const SettingsToggleId id : kAllSettingsToggleIds) {
		const std::size_t index = to_settings_toggle_index(id);
		if (index < _pending_notifications.size() &&
		    _pending_notifications[index]) {
			if (!has_change) {
				changed_id = id;
				has_change = true;
			}
			_pending_notifications[index] = false;
		}
	}
	if (has_change) {
		_on_view_model_change(changed_id);
	}
}

void SettingsToggleCoordinator::notify(SettingsToggleId id)
{
	const std::size_t index = to_settings_toggle_index(id);
	if (index >= _pending_notifications.size() || _is_shutting_down) {
		return;
	}
	if (_notification_batch_depth != 0) {
		_pending_notifications[index] = true;
		return;
	}
	if (_on_view_model_change) {
		_on_view_model_change(id);
	}
}

} // namespace ui
