#pragma once

namespace lvgl_main::wifi_latency_probe
{

/**
 * Filesystem sentinel sampling policy for target latency diagnostics.
 * Aggregate timing is useful without changing pixels. The visible marker is a
 * separate opt-in because it intentionally alters the framebuffer.
 */
struct Activation {
	bool aggregate_enabled {false};
	bool visible_marker_enabled {false};
};

[[nodiscard]] constexpr Activation activation_from_sentinels(
	bool aggregate_sentinel_present, bool marker_sentinel_present) noexcept
{
	return {
		.aggregate_enabled = aggregate_sentinel_present,
		.visible_marker_enabled =
			aggregate_sentinel_present && marker_sentinel_present,
	};
}

} // namespace lvgl_main::wifi_latency_probe
