#include "ui/wifi_ssid_font.hpp"

extern "C" {
	LV_FONT_DECLARE(wifi_zh_font);
}

namespace ui
{

const lv_font_t *compiled_wifi_ssid_font() noexcept
{
	static const lv_font_t fallback_font = [] {
		auto font = lv_font_montserrat_20;
		font.fallback = &wifi_zh_font;
		return font;
	}();
	return &fallback_font;
}

const lv_font_t *resolve_wifi_ssid_font(const lv_font_t *runtime_font) noexcept
{
	if (runtime_font != nullptr) {
		return runtime_font;
	}

	return compiled_wifi_ssid_font();
}

} // namespace ui
