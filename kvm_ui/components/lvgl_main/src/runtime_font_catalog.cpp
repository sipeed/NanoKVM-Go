#include "lvgl_main/runtime_font_catalog.hpp"

#include <utility>

#include <spdlog/spdlog.h>

#include <src/libs/tiny_ttf/lv_tiny_ttf.h>

namespace lvgl_main
{

RuntimeFontCatalog::RuntimeFontCatalog(CjkRegion cjk_region)
	: RuntimeFontCatalog(cjk_region, default_ops())
{
}

RuntimeFontCatalog::RuntimeFontCatalog(CjkRegion cjk_region, RuntimeFontOps ops)
	: _cjk_region(cjk_region)
	, _ops(std::move(ops))
{
}

RuntimeFontCatalog::~RuntimeFontCatalog()
{
	reset();
}

bool RuntimeFontCatalog::load()
{
	if (_loaded) {
		return true;
	}

	reset();
	if (!_ops.create_file || !_ops.destroy) {
		spdlog::warn("Runtime SSID font catalog has incomplete LVGL font operations");
		return false;
	}

	const auto base_path = noto_sans_path();
	const auto regional_path = cjk_font_path(_cjk_region);
	_noto_sans = _ops.create_file(base_path.data(), kSsidFontSize);
	if (_noto_sans == nullptr) {
		spdlog::warn("Failed to load runtime SSID font '{}'", base_path);
		return false;
	}

	_cjk_font = _ops.create_file(regional_path.data(), kSsidFontSize);
	if (_cjk_font == nullptr) {
		spdlog::warn("Failed to load runtime CJK SSID font '{}'", regional_path);
		reset();
		return false;
	}

	_noto_sans->fallback = _cjk_font;
	_loaded = true;
	spdlog::info("[lvgl] Loaded runtime SSID fonts: base='{}', cjk='{}'",
		     base_path, regional_path);
	return true;
}

void RuntimeFontCatalog::reset()
{
	_loaded = false;
	if (_noto_sans != nullptr) {
		_noto_sans->fallback = nullptr;
	}
	if (_cjk_font != nullptr && _ops.destroy) {
		_ops.destroy(_cjk_font);
	}
	_cjk_font = nullptr;
	if (_noto_sans != nullptr && _ops.destroy) {
		_ops.destroy(_noto_sans);
	}
	_noto_sans = nullptr;
}

const lv_font_t *RuntimeFontCatalog::ssid_font() const noexcept
{
	return _loaded ? _noto_sans : nullptr;
}

ui::SsidFontRenderMode RuntimeFontCatalog::ssid_render_mode() const noexcept
{
	return _loaded && _noto_sans != nullptr && _cjk_font != nullptr ?
		ui::SsidFontRenderMode::RuntimeTinyTtf :
		ui::SsidFontRenderMode::CompiledFallback;
}

std::string_view RuntimeFontCatalog::noto_sans_path() noexcept
{
	return "A:/usr/share/fonts/NotoSans-Regular.ttf";
}

std::string_view RuntimeFontCatalog::cjk_font_path(CjkRegion region) noexcept
{
	switch (region) {
	case CjkRegion::Japan:
		return "A:/usr/share/fonts/SourceHanSansJP-Regular.otf";
	case CjkRegion::Korea:
		return "A:/usr/share/fonts/SourceHanSansKR-Regular.otf";
	case CjkRegion::SimplifiedChinese:
		return "A:/usr/share/fonts/SourceHanSansSC-Regular.otf";
	case CjkRegion::TraditionalChineseTaiwan:
		return "A:/usr/share/fonts/SourceHanSansTC-Regular.otf";
	case CjkRegion::TraditionalChineseHongKong:
		return "A:/usr/share/fonts/SourceHanSansHK-Regular.otf";
	}
	return "A:/usr/share/fonts/SourceHanSansSC-Regular.otf";
}

RuntimeFontOps RuntimeFontCatalog::default_ops()
{
	return RuntimeFontOps {
		.create_file = [](const char *path, int32_t font_size) {
			return lv_tiny_ttf_create_file(path, font_size);
		},
		.destroy = [](lv_font_t *font) {
			lv_tiny_ttf_destroy(font);
		},
	};
}

} // namespace lvgl_main
