#include <array>
#include <string>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/runtime_font_catalog.hpp"

using namespace lvgl_main;

TEST_CASE("runtime font catalog maps every configured CJK region")
{
	const std::array expected_paths {
		std::pair { CjkRegion::Japan,
				    "A:/usr/share/fonts/SourceHanSansJP-Regular.otf" },
		std::pair { CjkRegion::Korea,
				    "A:/usr/share/fonts/SourceHanSansKR-Regular.otf" },
		std::pair { CjkRegion::SimplifiedChinese,
				    "A:/usr/share/fonts/SourceHanSansSC-Regular.otf" },
		std::pair { CjkRegion::TraditionalChineseTaiwan,
				    "A:/usr/share/fonts/SourceHanSansTC-Regular.otf" },
		std::pair { CjkRegion::TraditionalChineseHongKong,
				    "A:/usr/share/fonts/SourceHanSansHK-Regular.otf" },
	};

	CHECK(RuntimeFontCatalog::noto_sans_path() ==
	      "A:/usr/share/fonts/NotoSans-Regular.ttf");
	for (const auto &[region, path] : expected_paths) {
		CHECK(RuntimeFontCatalog::cjk_font_path(region) == path);
	}
}

TEST_CASE("runtime font catalog retains the base and CJK fallback together")
{
	lv_font_t noto {};
	lv_font_t cjk {};
	std::vector<std::string> created_paths;
	std::vector<int32_t> created_sizes;
	std::vector<lv_font_t *> destroyed_fonts;
	RuntimeFontCatalog catalog(
		CjkRegion::Korea,
		RuntimeFontOps {
			.create_file = [&created_paths, &created_sizes, &noto,
						&cjk](const char *path, int32_t size) {
				created_paths.emplace_back(path);
				created_sizes.push_back(size);
				return created_paths.size() == 1U ? &noto : &cjk;
			},
			.destroy = [&destroyed_fonts](lv_font_t *font) {
				destroyed_fonts.push_back(font);
			},
		});

	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::CompiledFallback);
	REQUIRE(catalog.load());
	REQUIRE(created_paths.size() == 2U);
	CHECK(created_paths[0] == "A:/usr/share/fonts/NotoSans-Regular.ttf");
	CHECK(created_paths[1] ==
	      "A:/usr/share/fonts/SourceHanSansKR-Regular.otf");
	CHECK(created_sizes[0] == RuntimeFontCatalog::kSsidFontSize);
	CHECK(created_sizes[1] == RuntimeFontCatalog::kSsidFontSize);
	CHECK(catalog.ssid_font() == &noto);
	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::RuntimeTinyTtf);
	CHECK(noto.fallback == &cjk);

	catalog.reset();
	CHECK(catalog.ssid_font() == nullptr);
	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::CompiledFallback);
	CHECK(noto.fallback == nullptr);
	REQUIRE(destroyed_fonts.size() == 2U);
	CHECK(destroyed_fonts[0] == &cjk);
	CHECK(destroyed_fonts[1] == &noto);
}

TEST_CASE("runtime font catalog falls back cleanly when the CJK face fails")
{
	lv_font_t noto {};
	std::vector<lv_font_t *> destroyed_fonts;
	uint32_t create_calls = 0;
	RuntimeFontCatalog catalog(
		CjkRegion::SimplifiedChinese,
		RuntimeFontOps {
			.create_file = [&create_calls, &noto](const char *, int32_t) {
				++create_calls;
				return create_calls == 1U ? &noto : nullptr;
			},
			.destroy = [&destroyed_fonts](lv_font_t *font) {
				destroyed_fonts.push_back(font);
			},
		});

	CHECK_FALSE(catalog.load());
	CHECK(catalog.ssid_font() == nullptr);
	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::CompiledFallback);
	CHECK(noto.fallback == nullptr);
	REQUIRE(destroyed_fonts.size() == 1U);
	CHECK(destroyed_fonts.front() == &noto);
}

TEST_CASE("runtime font catalog keeps compiled fallback mode when the base face fails")
{
	RuntimeFontCatalog catalog(
		CjkRegion::SimplifiedChinese,
		RuntimeFontOps {
			.create_file = [](const char *, int32_t) {
				return static_cast<lv_font_t *>(nullptr);
			},
			.destroy = [](lv_font_t *) {},
		});

	CHECK_FALSE(catalog.load());
	CHECK(catalog.ssid_font() == nullptr);
	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::CompiledFallback);
}

TEST_CASE("runtime font catalog uses compiled fallback mode with incomplete operations")
{
	RuntimeFontCatalog catalog(CjkRegion::SimplifiedChinese, RuntimeFontOps {});

	CHECK_FALSE(catalog.load());
	CHECK(catalog.ssid_font() == nullptr);
	CHECK(catalog.ssid_render_mode() ==
	      ui::SsidFontRenderMode::CompiledFallback);
}
