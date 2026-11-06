#pragma once

#include <array>
#include <chrono>
#include <cstddef>
#include <cstdint>

namespace ui
{

// Target-only diagnostic seam. It samples
// /tmp/kvm_ui_wifi_latency_probe once at first use, and its aggregate logs
// intentionally contain durations and counters only.
enum class WifiLatencyProbeStage : uint8_t {
	Snapshot,
	MarqueeRequest,
	MarqueeProcess,
	MarqueePublish,
	MarqueeTick,
	Count,
};

struct WifiLatencyProbeSummary {
	uint64_t count {0};
	uint64_t sampled_count {0};
	uint64_t p50_us {0};
	uint64_t p95_us {0};
	uint64_t max_us {0};
};

struct WifiLatencyProbeReport {
	WifiLatencyProbeSummary snapshot;
	WifiLatencyProbeSummary marquee_request;
	WifiLatencyProbeSummary marquee_process;
	WifiLatencyProbeSummary marquee_publish;
	WifiLatencyProbeSummary marquee_tick;
};

// Keeps the current report interval bounded without allocation. Quantiles use
// the most recent samples; count and max cover the full interval.
class WifiLatencyProbeStats final {
    public:
	static constexpr std::size_t kSampleCapacity = 256U;

	void add(uint64_t duration_us) noexcept;
	WifiLatencyProbeSummary summary() const noexcept;
	void clear() noexcept;

    private:
	std::array<uint64_t, kSampleCapacity> _samples {};
	std::size_t _next_sample {0};
	std::size_t _sampled_count {0};
	uint64_t _count {0};
	uint64_t _max_us {0};
};

class WifiLatencyProbe final {
    public:
	// Call during UI/page construction so latency-sensitive callbacks only
	// perform a cached branch.
	static void initialize() noexcept;
	static bool is_enabled() noexcept;
	static void record(WifiLatencyProbeStage stage, uint64_t duration_us) noexcept;
	static WifiLatencyProbeReport take_report() noexcept;
};

class ScopedWifiLatencyProbe final {
    public:
	explicit ScopedWifiLatencyProbe(WifiLatencyProbeStage stage) noexcept;
	~ScopedWifiLatencyProbe();

	ScopedWifiLatencyProbe(const ScopedWifiLatencyProbe &) = delete;
	ScopedWifiLatencyProbe &operator=(const ScopedWifiLatencyProbe &) = delete;

    private:
	WifiLatencyProbeStage _stage;
	std::chrono::steady_clock::time_point _started {};
	bool _enabled {false};
};

} // namespace ui
