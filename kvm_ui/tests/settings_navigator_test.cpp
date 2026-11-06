#include <array>
#include <cstddef>
#include <optional>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/page_registry.hpp"
#include "ui/root_pager.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/wifi_ssid_page.hpp"

namespace
{

constexpr int32_t kContentCardWidth = 270;
constexpr int32_t kContentCardHeight = 170;
constexpr int32_t kContentPageStep = 282;
constexpr uint32_t kTransitionSettleMs = 400;

using ContentPages = std::array<lv_obj_t *, 3>;

ContentPages create_content_pages(ui::RootPager &pager)
{
	ContentPages pages {};
	for (std::size_t index = 0; index < pages.size(); ++index) {
		pages[index] = lv_obj_create(pager.content_parent());
		lv_obj_set_size(pages[index], kContentCardWidth,
				kContentCardHeight);
		lv_obj_set_pos(pages[index],
			       static_cast<int32_t>(index) * kContentPageStep,
			       0);
		lv_obj_clear_flag(pages[index], LV_OBJ_FLAG_SCROLLABLE);
	}
	return pages;
}

void install_page_provider(ui::RootPager &pager, const ContentPages &pages)
{
	pager.set_content_page_provider(
		[&pages](ui::RootPager::ContentPage page) -> lv_obj_t * {
			switch (page) {
			case ui::RootPager::ContentPage::Main:
				return pages[0];
			case ui::RootPager::ContentPage::SettingsHome:
				return pages[1];
			case ui::RootPager::ContentPage::SettingsSecond:
				return pages[2];
			}
			return nullptr;
		});
}

void settle_transition(test_support::LvglTestContext &lvgl)
{
	lvgl.advance(kTransitionSettleMs);
}

} // namespace

TEST_CASE("SettingsNavigator preserves return anchors and nested Wi-Fi routes",
	  "[ui][settings-navigator]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	std::vector<std::optional<ui::SettingsRoute>> route_changes;
	navigator.set_route_change_callback(
		[&route_changes](std::optional<ui::SettingsRoute> route) {
			route_changes.push_back(route);
		});

	pager.transition_to(ui::RootPager::ContentPage::SettingsSecond);
	settle_transition(lvgl);
	REQUIRE(navigator.show(ui::SettingsRoute::wifi()));
	CHECK(navigator.return_anchor() ==
	      ui::RootPager::ContentPage::SettingsSecond);
	CHECK(pager.is_subpage_visible());
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());

	REQUIRE(navigator.push(ui::SettingsRoute::wifi_connect_method()));
	REQUIRE(navigator.push(ui::SettingsRoute::wifi_ssid()));
	CHECK(navigator.size() == 3U);
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi_ssid());

	REQUIRE(navigator.pop());
	CHECK(navigator.top_route() ==
	      ui::SettingsRoute::wifi_connect_method());
	REQUIRE(navigator.pop());
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());
	REQUIRE(navigator.pop());
	settle_transition(lvgl);
	CHECK(navigator.empty());
	CHECK_FALSE(pager.is_subpage_visible());
	CHECK(pager.current_content_page() ==
	      ui::RootPager::ContentPage::SettingsSecond);
	REQUIRE_FALSE(route_changes.empty());
	CHECK_FALSE(route_changes.back().has_value());
}

TEST_CASE("SettingsNavigator clears nested page work before registry teardown",
	  "[ui][settings-navigator][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();

	{
		ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
		pager.create(lvgl.screen());
		const ContentPages content_pages = create_content_pages(pager);
		install_page_provider(pager, content_pages);

		ui::PageRegistry registry;
		registry.create(pager.settings_host());
		ui::SettingsNavigator navigator(pager, registry);

		ui::WifiSsidPage *ssid_page = registry.wifi_ssid_page();
		REQUIRE(ssid_page != nullptr);
		REQUIRE(navigator.show(ui::SettingsRoute::wifi()));
		ssid_page->set_loading("Scanning...");
		REQUIRE(navigator.push(ui::SettingsRoute::wifi_ssid()));
		const std::size_t timer_count_with_ssid_loading =
			lvgl.timer_count();

		navigator.clear();
		CHECK(navigator.empty());
		CHECK(lvgl.timer_count() + 1U == timer_count_with_ssid_loading);
	}

	CHECK(lvgl.timer_count() == timer_count_before);
}

TEST_CASE("SettingsNavigator preserves repeated routes as nested pages",
	  "[ui][settings-navigator]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);

	REQUIRE(navigator.show(ui::SettingsRoute::wifi()));
	REQUIRE(navigator.push(ui::SettingsRoute::wifi_connect_method()));
	REQUIRE(navigator.push(ui::SettingsRoute::wifi()));
	CHECK(navigator.size() == 3U);
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());

	REQUIRE(navigator.pop());
	CHECK(navigator.size() == 2U);
	CHECK(navigator.top_route() ==
	      ui::SettingsRoute::wifi_connect_method());
}

TEST_CASE("SettingsNavigator rejects navigation while a root Flip is active",
	  "[ui][settings-navigator][flip]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Flip);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);

	pager.transition_to(ui::RootPager::ContentPage::SettingsHome);
	REQUIRE(pager.is_flip_animating());
	CHECK_FALSE(navigator.show(ui::SettingsRoute::wifi()));
	CHECK(navigator.empty());

	settle_transition(lvgl);
	CHECK_FALSE(pager.is_flip_animating());
	CHECK(navigator.show(ui::SettingsRoute::wifi()));
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());
}
