#pragma once

#include <chrono>
#include <cstdint>

namespace lvgl_main
{

/**
 * UI-thread-only input quiet gate for expensive prepared-SSID installation.
 *
 * A raster worker may replace the mailbox value while this gate is closed. The
 * caller must check the gate before taking that one-slot mailbox so an input
 * event never waits behind a natural-width LVGL source copy.
 */
class SsidRasterInstallGate final {
    public:
	using Clock = std::chrono::steady_clock;
	using TimePoint = Clock::time_point;

	static constexpr auto kQuietInterval = std::chrono::milliseconds(500);

	void reset(uint64_t input_activity_sequence, TimePoint now) noexcept
	{
		_input_activity_sequence = input_activity_sequence;
		_quiet_until = now;
	}

	void observe(uint64_t input_activity_sequence, TimePoint now) noexcept
	{
		if (input_activity_sequence == _input_activity_sequence) {
			return;
		}
		_input_activity_sequence = input_activity_sequence;
		_quiet_until = now + kQuietInterval;
	}

	[[nodiscard]] bool blocks_install(bool touch_pressed,
					TimePoint now) const noexcept
	{
		return touch_pressed || now < _quiet_until;
	}

    private:
	uint64_t _input_activity_sequence {0};
	TimePoint _quiet_until {};
};

} // namespace lvgl_main
