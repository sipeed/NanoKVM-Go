#include "ui/wifi_latency_probe.hpp"

#include <algorithm>
#include <array>
#include <chrono>
#include <filesystem>
#include <system_error>

namespace ui
{
namespace
{

constexpr char kProbeSentinelPath[] = "/tmp/kvm_ui_wifi_latency_probe";
constexpr std::size_t kStageCount = static_cast<std::size_t>(WifiLatencyProbeStage::Count);

std::size_t stage_index(WifiLatencyProbeStage stage) noexcept
{
	return static_cast<std::size_t>(stage);
}

struct ProbeState {
	bool initialized {false};
	bool enabled {false};
	std::array<WifiLatencyProbeStats, kStageCount> stats {};
};

ProbeState &probe_state() noexcept
{
	static ProbeState state;
	return state;
}

} // namespace

void WifiLatencyProbeStats::add(uint64_t duration_us) noexcept
{
	_samples[_next_sample] = duration_us;
	_next_sample = (_next_sample + 1U) % _samples.size();
	_sampled_count = std::min(_sampled_count + 1U, _samples.size());
	++_count;
	_max_us = std::max(_max_us, duration_us);
}

WifiLatencyProbeSummary WifiLatencyProbeStats::summary() const noexcept
{
	WifiLatencyProbeSummary result {
	    .count = _count,
	    .sampled_count = _sampled_count,
	    .max_us = _max_us,
	};
	if (_sampled_count == 0U) {
		return result;
	}

	std::array<uint64_t, kSampleCapacity> ordered {};
	std::copy_n(_samples.begin(), _sampled_count, ordered.begin());
	std::sort(ordered.begin(), ordered.begin() + static_cast<std::ptrdiff_t>(_sampled_count));
	const auto quantile_index = [this](uint64_t percent) noexcept {
		return static_cast<std::size_t>(((percent * _sampled_count + 99U) / 100U) - 1U);
	};
	result.p50_us = ordered[quantile_index(50U)];
	result.p95_us = ordered[quantile_index(95U)];
	return result;
}

void WifiLatencyProbeStats::clear() noexcept
{
	_next_sample = 0U;
	_sampled_count = 0U;
	_count = 0U;
	_max_us = 0U;
}

void WifiLatencyProbe::initialize() noexcept
{
	ProbeState &state = probe_state();
	if (state.initialized) {
		return;
	}

	std::error_code error;
	state.enabled = std::filesystem::exists(kProbeSentinelPath, error) && !error;
	state.initialized = true;
}

bool WifiLatencyProbe::is_enabled() noexcept
{
	const ProbeState &state = probe_state();
	return state.initialized && state.enabled;
}

void WifiLatencyProbe::record(WifiLatencyProbeStage stage, uint64_t duration_us) noexcept
{
	ProbeState &state = probe_state();
	if (!state.enabled || stage_index(stage) >= state.stats.size()) {
		return;
	}

	state.stats[stage_index(stage)].add(duration_us);
}

WifiLatencyProbeReport WifiLatencyProbe::take_report() noexcept
{
	ProbeState &state = probe_state();
	if (!state.enabled) {
		return {};
	}

	WifiLatencyProbeReport report {
	    .snapshot = state.stats[stage_index(WifiLatencyProbeStage::Snapshot)].summary(),
	    .marquee_request =
		state.stats[stage_index(WifiLatencyProbeStage::MarqueeRequest)].summary(),
	    .marquee_process =
		state.stats[stage_index(WifiLatencyProbeStage::MarqueeProcess)].summary(),
	    .marquee_publish =
		state.stats[stage_index(WifiLatencyProbeStage::MarqueePublish)].summary(),
	    .marquee_tick = state.stats[stage_index(WifiLatencyProbeStage::MarqueeTick)].summary(),
	};
	for (WifiLatencyProbeStats &stats : state.stats) {
		stats.clear();
	}
	return report;
}

ScopedWifiLatencyProbe::ScopedWifiLatencyProbe(WifiLatencyProbeStage stage) noexcept
    : _stage(stage), _enabled(WifiLatencyProbe::is_enabled())
{
	if (_enabled) {
		_started = std::chrono::steady_clock::now();
	}
}

ScopedWifiLatencyProbe::~ScopedWifiLatencyProbe()
{
	if (!_enabled) {
		return;
	}
	const auto duration = std::chrono::duration_cast<std::chrono::microseconds>(
	    std::chrono::steady_clock::now() - _started);
	WifiLatencyProbe::record(_stage,
				 static_cast<uint64_t>(std::max<int64_t>(0, duration.count())));
}

} // namespace ui
