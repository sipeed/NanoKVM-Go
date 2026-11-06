#pragma once

#include <cstdint>

namespace lvgl_main
{

enum class WifiRuntimeState : uint8_t {
	Unknown,
	Connected,
	Disconnected,
	CacheConnecting,
	ApAwaitClient,
	ApAwaitWeb,
	WaitingExternalConnection,
	Error,
};

} // namespace lvgl_main
