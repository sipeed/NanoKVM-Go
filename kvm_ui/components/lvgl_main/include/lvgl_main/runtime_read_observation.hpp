#pragma once

#include <optional>
#include <utility>

namespace lvgl_main
{

// A state read is either a validated observation (including false, empty, or
// zero) or inconclusive. The latter represents timeout, spawn, transport, or
// parse failure and must not become a state value.
template <typename T>
class StateReadObservation final {
    public:
	static StateReadObservation observed(T value)
	{
		return StateReadObservation(std::move(value));
	}

	static StateReadObservation inconclusive()
	{
		return StateReadObservation();
	}

	bool is_observed() const noexcept
	{
		return _value.has_value();
	}

	const T &value() const
	{
		return *_value;
	}

    private:
	explicit StateReadObservation(T value)
		: _value(std::move(value))
	{
	}

	StateReadObservation() = default;

	std::optional<T> _value;
};

inline StateReadObservation<bool>
observe_systemd_active_result(int exit_code, bool timed_out)
{
	if (timed_out ||
	    (exit_code != 0 && exit_code != 3 && exit_code != 4)) {
		return StateReadObservation<bool>::inconclusive();
	}
	return StateReadObservation<bool>::observed(exit_code == 0);
}

inline StateReadObservation<bool>
observe_systemd_unit_exists_result(int exit_code, bool timed_out)
{
	if (timed_out ||
	    (exit_code != 0 && exit_code != 3 && exit_code != 4)) {
		return StateReadObservation<bool>::inconclusive();
	}
	return StateReadObservation<bool>::observed(exit_code == 0 ||
								    exit_code == 3);
}

template <typename T>
StateReadObservation<T>
observe_required_http_field(std::optional<T> value)
{
	if (!value.has_value()) {
		return StateReadObservation<T>::inconclusive();
	}
	return StateReadObservation<T>::observed(std::move(*value));
}

} // namespace lvgl_main
