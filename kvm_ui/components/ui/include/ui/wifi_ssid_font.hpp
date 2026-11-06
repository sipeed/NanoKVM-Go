#pragma once

#include <cstdint>

#include <lvgl.h>

namespace ui
{

inline constexpr uint16_t kWifiSsidFontSizePx = 20;

// Distinguishes a file-backed TinyTTF font from the compiled fallback chain.
// A non-null lv_font_t pointer alone is not enough to choose an SSID rendering
// path because tests and future compiled fonts may also supply one.
enum class SsidFontRenderMode : uint8_t {
	CompiledFallback,
	RuntimeTinyTtf,
};

// Returns the one process-lifetime compiled fallback chain
// (Montserrat-20 -> wifi_zh_font). Callers never own or destroy the returned
// font.
const lv_font_t *compiled_wifi_ssid_font() noexcept;

// Returns the borrowed runtime SSID font when available. Otherwise returns the
// one process-lifetime compiled fallback chain (Montserrat-20 -> wifi_zh_font).
// Callers never own or destroy the returned font.
const lv_font_t *resolve_wifi_ssid_font(const lv_font_t *runtime_font) noexcept;

} // namespace ui
