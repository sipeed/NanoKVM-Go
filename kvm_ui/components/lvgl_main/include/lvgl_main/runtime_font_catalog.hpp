#pragma once

#include <cstdint>
#include <functional>
#include <string_view>

#include <lvgl.h>

#include "lvgl_main/app_config.hpp"
#include "ui/wifi_ssid_font.hpp"

namespace lvgl_main
{

struct RuntimeFontOps {
	std::function<lv_font_t *(const char *path, int32_t font_size)> create_file;
	std::function<void(lv_font_t *font)> destroy;
};

// Owns the file-backed fonts used by the current Wi-Fi SSID presentation path.
// It must be created and destroyed on the LVGL thread while LVGL is initialized.
class RuntimeFontCatalog final {
    public:
	static constexpr int32_t kSsidFontSize = ui::kWifiSsidFontSizePx;

	explicit RuntimeFontCatalog(CjkRegion cjk_region);
	RuntimeFontCatalog(CjkRegion cjk_region, RuntimeFontOps ops);
	~RuntimeFontCatalog();

	RuntimeFontCatalog(const RuntimeFontCatalog &) = delete;
	RuntimeFontCatalog &operator=(const RuntimeFontCatalog &) = delete;

	bool load();
	void reset();

	const lv_font_t *ssid_font() const noexcept;
	ui::SsidFontRenderMode ssid_render_mode() const noexcept;

	static std::string_view noto_sans_path() noexcept;
	static std::string_view cjk_font_path(CjkRegion region) noexcept;

    private:
	static RuntimeFontOps default_ops();

	CjkRegion _cjk_region;
	RuntimeFontOps _ops;
	lv_font_t *_noto_sans { nullptr };
	lv_font_t *_cjk_font { nullptr };
	bool _loaded { false };
};

} // namespace lvgl_main
