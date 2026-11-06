/**
 * @file wifi_latency_probe.cpp
 * Bounded target-only timing aggregates for the Wi-Fi TinyTTF latency trace.
 */

#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <ctime>
#include <filesystem>
#include <mutex>

#include <spdlog/spdlog.h>

#include "lvgl_main/wifi_latency_probe_activation.hpp"
#include "lvgl_main/wifi_latency_probe_stats.hpp"
#include "lvgl_ui_internal.hpp"
#include "ui/wifi_latency_probe.hpp"

namespace lvgl_main::wifi_latency_probe
{

namespace
{

using SteadyClock = std::chrono::steady_clock;

constexpr auto kLogInterval = std::chrono::seconds(5);
std::atomic<bool> g_enabled {false};
std::atomic<bool> g_visible_marker_enabled {false};
std::atomic<bool> g_event_clock_monotonic {false};

struct PendingPress {
	bool active {false};
	bool read_recorded {false};
	bool handler_recorded {false};
	int64_t event_timestamp_us {0};
	int64_t arrival_timestamp_us {0};
	int64_t read_timestamp_us {0};
	int64_t handler_timestamp_us {0};
};

struct ProbeState {
	std::mutex mutex;
	bool enabled {false};
	PendingPress pending;
	SampleSet event_to_arrival_us;
	SampleSet event_to_read_us;
	SampleSet read_to_handler_us;
	SampleSet event_to_handler_us;
	SampleSet handler_to_flush_us;
	SampleSet event_to_flush_us;
	SampleSet timer_handler_us;
	SampleSet flush_wrapper_us;
	SampleSet flush_original_us;
	SampleSet flush_pixels;
	uint64_t flush_count {0U};
	uint64_t flush_pixel_total {0U};
	uint64_t marker_flush_count {0U};
	SteadyClock::time_point last_log {};
};

ProbeState &state()
{
	static ProbeState value;
	return value;
}

void reset_locked(ProbeState &probe)
{
	probe.pending = {};
	probe.event_to_arrival_us = {};
	probe.event_to_read_us = {};
	probe.read_to_handler_us = {};
	probe.event_to_handler_us = {};
	probe.handler_to_flush_us = {};
	probe.event_to_flush_us = {};
	probe.timer_handler_us = {};
	probe.flush_wrapper_us = {};
	probe.flush_original_us = {};
	probe.flush_pixels = {};
	probe.flush_count = 0U;
	probe.flush_pixel_total = 0U;
	probe.marker_flush_count = 0U;
	probe.last_log = SteadyClock::now();
}

void clear_interval_locked(ProbeState &probe)
{
	probe.event_to_arrival_us = {};
	probe.event_to_read_us = {};
	probe.read_to_handler_us = {};
	probe.event_to_handler_us = {};
	probe.handler_to_flush_us = {};
	probe.event_to_flush_us = {};
	probe.timer_handler_us = {};
	probe.flush_wrapper_us = {};
	probe.flush_original_us = {};
	probe.flush_pixels = {};
	probe.flush_count = 0U;
	probe.flush_pixel_total = 0U;
	probe.marker_flush_count = 0U;
}

uint64_t non_negative_delta_us(int64_t end_us, int64_t begin_us)
{
	if (end_us <= begin_us) {
		return 0U;
	}
	const uint64_t delta = static_cast<uint64_t>(end_us - begin_us);
	return std::min(delta, static_cast<uint64_t>(60000000U));
}

int64_t selected_clock_now_us()
{
	timespec now {};
	const clockid_t clock_id = g_event_clock_monotonic.load(std::memory_order_acquire)
				       ? CLOCK_MONOTONIC
				       : CLOCK_REALTIME;
	if (::clock_gettime(clock_id, &now) != 0) {
		return 0;
	}
	return static_cast<int64_t>(now.tv_sec) * 1000000LL +
	       static_cast<int64_t>(now.tv_nsec / 1000L);
}

struct SampleSummary {
	uint64_t count {0U};
	uint64_t p50 {0U};
	uint64_t p95 {0U};
	uint64_t max {0U};
};

SampleSummary summarize(const SampleSet &samples)
{
	return {
	    .count = samples.count(),
	    .p50 = samples.percentile(50U),
	    .p95 = samples.percentile(95U),
	    .max = samples.max(),
	};
}

} // namespace

void initialize()
{
	std::error_code error;
	const bool aggregate_sentinel_present =
		std::filesystem::exists(kSentinelPath, error) && !error;
	error.clear();
	const bool marker_sentinel_present =
		std::filesystem::exists(kMarkerSentinelPath, error) && !error;
	const auto activation =
		activation_from_sentinels(aggregate_sentinel_present, marker_sentinel_present);
	g_enabled.store(activation.aggregate_enabled, std::memory_order_release);
	g_visible_marker_enabled.store(activation.visible_marker_enabled,
				       std::memory_order_release);
	g_event_clock_monotonic.store(false, std::memory_order_release);
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	reset_locked(probe);
	probe.enabled = activation.aggregate_enabled;
	if (probe.enabled) {
		spdlog::warn("[wifi-latency] enabled sentinel={} visible_marker={} "
				     "aggregate_interval_s=5 samples=64 event_clock=pending",
				     kSentinelPath, activation.visible_marker_enabled);
	}
}

void reset()
{
	g_enabled.store(false, std::memory_order_release);
	g_visible_marker_enabled.store(false, std::memory_order_release);
	g_event_clock_monotonic.store(false, std::memory_order_release);
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	reset_locked(probe);
	probe.enabled = false;
}

bool enabled()
{
	return g_enabled.load(std::memory_order_acquire);
}

bool visible_marker_enabled()
{
	return g_visible_marker_enabled.load(std::memory_order_acquire);
}

void set_event_clock_monotonic(bool monotonic)
{
	g_event_clock_monotonic.store(monotonic, std::memory_order_release);
}

int64_t timestamp_now_us()
{
	return selected_clock_now_us();
}

void record_touch_press_event(int64_t event_timestamp_us, int64_t arrival_timestamp_us)
{
	if (!enabled()) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (!probe.enabled) {
		return;
	}
	probe.pending = PendingPress {
	    .active = true,
	    .event_timestamp_us = event_timestamp_us,
	    .arrival_timestamp_us = arrival_timestamp_us,
	};
	probe.event_to_arrival_us.add(
	    non_negative_delta_us(arrival_timestamp_us, event_timestamp_us));
}

void record_input_read(int64_t read_timestamp_us)
{
	if (!enabled()) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (!probe.enabled || !probe.pending.active || probe.pending.read_recorded) {
		return;
	}
	probe.pending.read_recorded = true;
	probe.pending.read_timestamp_us = read_timestamp_us;
	probe.event_to_read_us.add(
	    non_negative_delta_us(read_timestamp_us, probe.pending.event_timestamp_us));
}

void record_lvgl_pressed(int64_t handler_timestamp_us)
{
	if (!enabled()) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (!probe.enabled || !probe.pending.active || probe.pending.handler_recorded) {
		return;
	}
	probe.pending.handler_recorded = true;
	probe.pending.handler_timestamp_us = handler_timestamp_us;
	probe.event_to_handler_us.add(
	    non_negative_delta_us(handler_timestamp_us, probe.pending.event_timestamp_us));
	if (probe.pending.read_recorded) {
		probe.read_to_handler_us.add(
		    non_negative_delta_us(handler_timestamp_us, probe.pending.read_timestamp_us));
	}
}

void record_visible_flush(const lv_display_t *disp, const lv_area_t *area)
{
	if (!enabled() || !visible_marker_enabled() || area == nullptr) {
		return;
	}
	if (!touch_click_marker_state.visible.load(std::memory_order_acquire)) {
		return;
	}
	const int32_t marker_x = touch_click_marker_state.x.load(std::memory_order_acquire) +
					 (disp == nullptr ? 0 : lv_display_get_offset_x(disp));
	const int32_t marker_y = touch_click_marker_state.y.load(std::memory_order_acquire) +
					 (disp == nullptr ? 0 : lv_display_get_offset_y(disp));
	if (area->x2 < marker_x - kTouchClickMarkerRadius ||
	    area->x1 > marker_x + kTouchClickMarkerRadius ||
	    area->y2 < marker_y - kTouchClickMarkerRadius ||
	    area->y1 > marker_y + kTouchClickMarkerRadius) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (!probe.enabled || !probe.pending.active || !probe.pending.handler_recorded) {
		return;
	}
	const int64_t flush_timestamp_us = timestamp_now_us();
	probe.handler_to_flush_us.add(
	    non_negative_delta_us(flush_timestamp_us, probe.pending.handler_timestamp_us));
	probe.event_to_flush_us.add(
	    non_negative_delta_us(flush_timestamp_us, probe.pending.event_timestamp_us));
	++probe.marker_flush_count;
	probe.pending.active = false;
}

void record_flush(uint64_t wrapper_duration_us, uint64_t original_duration_us, uint64_t pixels)
{
	if (!enabled()) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (!probe.enabled) {
		return;
	}
	probe.flush_wrapper_us.add(wrapper_duration_us);
	probe.flush_original_us.add(original_duration_us);
	probe.flush_pixels.add(pixels);
	++probe.flush_count;
	probe.flush_pixel_total += pixels;
}

void record_timer_handler(uint64_t duration_us)
{
	if (!enabled()) {
		return;
	}
	auto &probe = state();
	std::lock_guard<std::mutex> lock(probe.mutex);
	if (probe.enabled) {
		probe.timer_handler_us.add(duration_us);
	}
}

void maybe_log()
{
	if (!enabled()) {
		return;
	}

	struct LogSnapshot {
		uint64_t flush_count {0U};
		uint64_t flush_pixel_total {0U};
		uint64_t marker_flush_count {0U};
		SampleSummary event_to_arrival;
		SampleSummary event_to_read;
		SampleSummary read_to_handler;
		SampleSummary event_to_handler;
		SampleSummary handler_to_flush;
		SampleSummary event_to_flush;
		SampleSummary timer_handler;
		SampleSummary flush_wrapper;
		SampleSummary flush_original;
		SampleSummary flush_pixels;
	};

	LogSnapshot snapshot;
	auto &probe = state();
	{
		std::lock_guard<std::mutex> lock(probe.mutex);
		if (!probe.enabled || SteadyClock::now() - probe.last_log < kLogInterval) {
			return;
		}
		probe.last_log = SteadyClock::now();
		snapshot.flush_count = probe.flush_count;
		snapshot.flush_pixel_total = probe.flush_pixel_total;
		snapshot.marker_flush_count = probe.marker_flush_count;
		snapshot.event_to_arrival = summarize(probe.event_to_arrival_us);
		snapshot.event_to_read = summarize(probe.event_to_read_us);
		snapshot.read_to_handler = summarize(probe.read_to_handler_us);
		snapshot.event_to_handler = summarize(probe.event_to_handler_us);
		snapshot.handler_to_flush = summarize(probe.handler_to_flush_us);
		snapshot.event_to_flush = summarize(probe.event_to_flush_us);
		snapshot.timer_handler = summarize(probe.timer_handler_us);
		snapshot.flush_wrapper = summarize(probe.flush_wrapper_us);
		snapshot.flush_original = summarize(probe.flush_original_us);
		snapshot.flush_pixels = summarize(probe.flush_pixels);
		clear_interval_locked(probe);
	}
	const ui::WifiLatencyProbeReport ui_report = ui::WifiLatencyProbe::take_report();
	spdlog::warn("[wifi-latency] input n(arrival/read/read_to_handler/event_to_handler/"
		     "handler_to_flush/event_to_flush)={}/{}/{}/{}/{}/{} "
		     "p50_us={}/{}/{}/{}/{}/{} p95_us={}/{}/{}/{}/{}/{} "
		     "max_us={}/{}/{}/{}/{}/{}",
		     snapshot.event_to_arrival.count, snapshot.event_to_read.count,
		     snapshot.read_to_handler.count, snapshot.event_to_handler.count,
		     snapshot.handler_to_flush.count, snapshot.event_to_flush.count,
		     snapshot.event_to_arrival.p50, snapshot.event_to_read.p50,
		     snapshot.read_to_handler.p50, snapshot.event_to_handler.p50,
		     snapshot.handler_to_flush.p50, snapshot.event_to_flush.p50,
		     snapshot.event_to_arrival.p95, snapshot.event_to_read.p95,
		     snapshot.read_to_handler.p95, snapshot.event_to_handler.p95,
		     snapshot.handler_to_flush.p95, snapshot.event_to_flush.p95,
		     snapshot.event_to_arrival.max, snapshot.event_to_read.max,
		     snapshot.read_to_handler.max, snapshot.event_to_handler.max,
		     snapshot.handler_to_flush.max, snapshot.event_to_flush.max);
	spdlog::warn(
	    "[wifi-latency] work "
	    "n(timer/wrapper/original/pixels/snapshot/request/process/publish/tick)="
	    "{}/{}/{}/{}/{}/{}/{}/{}/{} "
	    "p50_us_or_px={}/{}/{}/{}/{}/{}/{}/{}/{} "
	    "p95_us_or_px={}/{}/{}/{}/{}/{}/{}/{}/{} "
	    "max_us_or_px={}/{}/{}/{}/{}/{}/{}/{}/{} "
	    "flush_pixel_total={} marker_flushes={}",
	    snapshot.timer_handler.count, snapshot.flush_wrapper.count,
	    snapshot.flush_original.count, snapshot.flush_pixels.count, ui_report.snapshot.count,
	    ui_report.marquee_request.count, ui_report.marquee_process.count,
	    ui_report.marquee_publish.count, ui_report.marquee_tick.count,
	    snapshot.timer_handler.p50, snapshot.flush_wrapper.p50, snapshot.flush_original.p50,
	    snapshot.flush_pixels.p50, ui_report.snapshot.p50_us, ui_report.marquee_request.p50_us,
	    ui_report.marquee_process.p50_us, ui_report.marquee_publish.p50_us,
	    ui_report.marquee_tick.p50_us, snapshot.timer_handler.p95, snapshot.flush_wrapper.p95,
	    snapshot.flush_original.p95, snapshot.flush_pixels.p95, ui_report.snapshot.p95_us,
	    ui_report.marquee_request.p95_us, ui_report.marquee_process.p95_us,
	    ui_report.marquee_publish.p95_us, ui_report.marquee_tick.p95_us,
	    snapshot.timer_handler.max, snapshot.flush_wrapper.max, snapshot.flush_original.max,
	    snapshot.flush_pixels.max, ui_report.snapshot.max_us, ui_report.marquee_request.max_us,
	    ui_report.marquee_process.max_us, ui_report.marquee_publish.max_us,
	    ui_report.marquee_tick.max_us, snapshot.flush_pixel_total, snapshot.marker_flush_count);
}

} // namespace lvgl_main::wifi_latency_probe
