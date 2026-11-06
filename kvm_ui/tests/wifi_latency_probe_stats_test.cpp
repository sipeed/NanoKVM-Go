#include <array>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/wifi_latency_probe_activation.hpp"
#include "lvgl_main/wifi_latency_probe_stats.hpp"

using lvgl_main::wifi_latency_probe::SampleSet;

TEST_CASE("wifi latency probe uses nearest-rank percentiles")
{
	SampleSet samples;
	samples.add(10U);
	samples.add(20U);
	samples.add(30U);
	samples.add(40U);

	CHECK(samples.count() == 4U);
	CHECK(samples.percentile(50U) == 20U);
	CHECK(samples.percentile(95U) == 40U);
	CHECK(samples.max() == 40U);
}

TEST_CASE("wifi latency probe keeps a bounded recent percentile window")
{
	SampleSet samples;
	for (uint64_t value = 1U; value <= 65U; ++value) {
		samples.add(value);
	}

	CHECK(samples.count() == 65U);
	CHECK(samples.percentile(50U) == 33U);
	CHECK(samples.percentile(95U) == 62U);
	CHECK(samples.max() == 65U);
}

TEST_CASE("wifi latency probe only enables a visible marker by explicit opt-in")
{
	using lvgl_main::wifi_latency_probe::activation_from_sentinels;

	struct Case {
		bool aggregate_sentinel_present;
		bool marker_sentinel_present;
		bool aggregate_enabled;
		bool visible_marker_enabled;
	};
	constexpr std::array cases {
		Case {false, false, false, false},
		Case {true, false, true, false},
		Case {false, true, false, false},
		Case {true, true, true, true},
	};

	for (const Case test_case : cases) {
		const auto activation = activation_from_sentinels(
			test_case.aggregate_sentinel_present,
			test_case.marker_sentinel_present);
		CHECK(activation.aggregate_enabled == test_case.aggregate_enabled);
		CHECK(activation.visible_marker_enabled ==
		      test_case.visible_marker_enabled);
	}
}
