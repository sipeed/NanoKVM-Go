#pragma once

#include <cstdint>

namespace ui
{

// Loading-animation strategy for the Wi-Fi scan page. None keeps the legacy
// continuous rotating arc; Progress is a circular progress bar with two
// monotonic halves: the scan half climbs 0->50% on a timer (~3810 ms at 75%
// of the former 70%/4000 ms rate), then the cache-render half advances
// 50->100% from drawn/total raster progress. A warm cache with nothing to
// raster closes 50->100% over 250 ms instead.
enum class WifiLoadingMode : uint8_t {
	None,
	Progress,
};

} // namespace ui
