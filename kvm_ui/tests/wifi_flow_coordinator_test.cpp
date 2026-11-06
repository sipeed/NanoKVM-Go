#include <array>
#include <cstddef>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/page_registry.hpp"
#include "ui/root_layout.hpp"
#include "ui/root_pager.hpp"
#include "ui/settings_navigator.hpp"
#include "ui/wifi_config_page.hpp"
#include "ui/wifi_connect_method_page.hpp"
#include "ui/wifi_flow_coordinator.hpp"
#include "ui/page_manager_actions.hpp"
#include "ui/wifi_page.hpp"
#include "ui/wifi_qr_page.hpp"
#include "ui/wifi_ssid_page.hpp"

namespace
{

constexpr uint32_t kTransitionSettleMs = 400;

using ContentPages = std::array<lv_obj_t *, ui::root_layout::kContentPageCount>;

ContentPages create_content_pages(ui::RootPager &pager)
{
	ContentPages pages {};
	for (std::size_t index = 0; index < pages.size(); ++index) {
		pages[index] = lv_obj_create(pager.content_parent());
		lv_obj_set_size(pages[index],
				ui::root_layout::kContentCardWidth,
				ui::root_layout::kContentCardHeight);
		lv_obj_set_pos(pages[index],
			       static_cast<int32_t>(index) *
				       ui::root_layout::kContentPageStep,
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
			case ui::RootPager::ContentPage::Count:
				return nullptr;
			}
			return nullptr;
		});
}

void settle_transition(test_support::LvglTestContext &lvgl)
{
	lvgl.advance(kTransitionSettleMs);
}

} // namespace

TEST_CASE("WifiFlowCoordinator owns nested routes and Wi-Fi callback wiring",
	  "[ui][wifi-flow]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	std::vector<bool> visibility;
	std::vector<std::pair<bool, bool>> connectivity;
	std::vector<std::optional<ui::SettingsRoute>> render_routes;
	std::vector<bool> enabled_requests;
	int scan_requests = 0;
	int qr_dismissed_requests = 0;
	std::vector<uint64_t> ssid_raster_prepares;
	std::vector<uint64_t> ssid_raster_invalidations;
	std::vector<std::pair<std::string, std::string>> connect_requests;

	ui::PageManagerActions actions;
	ui::WifiFlowCoordinator::Callbacks callbacks;
	callbacks.render_connectivity =
		[&connectivity, &navigator,
		 &render_routes](ui::WifiConnectivityViewModel model) {
			connectivity.emplace_back(model.active,
						  model.boot_reconnect_pending);
			render_routes.push_back(navigator.top_route());
		};
	actions.wifi.page_visibility_changed = [&visibility](bool visible) {
		visibility.push_back(visible);
	};
	actions.wifi.scan = [&scan_requests]() {
		++scan_requests;
	};
	actions.wifi.connect =
		[&connect_requests](const std::string &ssid,
				    const std::string &password) {
			connect_requests.emplace_back(ssid, password);
		};
	actions.wifi.set_enabled = [&enabled_requests](bool enabled) {
		enabled_requests.push_back(enabled);
	};
	actions.wifi.qr_dismissed = [&qr_dismissed_requests]() {
		++qr_dismissed_requests;
	};
	actions.ssid_raster.prepare =
		[&ssid_raster_prepares](ui::SsidRasterRequest request) {
			ssid_raster_prepares.push_back(request.identity.generation);
			return true;
		};
	actions.ssid_raster.invalidate_page =
		[&ssid_raster_invalidations](uint64_t page_token) {
			ssid_raster_invalidations.push_back(page_token);
		};

	ui::WifiFlowCoordinator flow(registry, navigator, actions.wifi,
				     actions.ssid_raster, true, 100, callbacks);
	flow.bind_pages();
	connectivity.clear();
	render_routes.clear();
	flow.set_runtime_wifi_active(false);
	CHECK(connectivity ==
	      std::vector<std::pair<bool, bool>> { { false, false } });

	pager.transition_to(ui::RootPager::ContentPage::SettingsHome);
	settle_transition(lvgl);
	flow.show_wifi();
	REQUIRE(navigator.top_route() == ui::SettingsRoute::wifi());
	REQUIRE(visibility == std::vector<bool> { true });

	ui::WifiPage *wifi_page = registry.wifi_page();
	REQUIRE(wifi_page != nullptr);
	ui::SsidRasterRequest raster_request;
	raster_request.identity.page_token = 23;
	raster_request.identity.generation = 41;
	REQUIRE(wifi_page->on_ssid_raster_prepare);
	CHECK(wifi_page->on_ssid_raster_prepare(raster_request));
	wifi_page->on_ssid_raster_invalidate(23);
	CHECK(ssid_raster_prepares == std::vector<uint64_t> { 41 });
	CHECK(ssid_raster_invalidations == std::vector<uint64_t> { 23 });
	flow.set_ap_qr_payload("WIFI:T:WPA;S:Office;;");
	CHECK(wifi_page->qr_payload() == "WIFI:T:WPA;S:Office;;");
	wifi_page->on_wifi_enabled_change(true);
	wifi_page->on_wifi_qr_dismissed();
	CHECK(enabled_requests == std::vector<bool> { true });
	CHECK(qr_dismissed_requests == 1);

	flow.show_wifi_config();
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi_ssid());
	CHECK(scan_requests == 1);
	CHECK(visibility == std::vector<bool> { true });

	ui::WifiSsidPage *ssid_page = registry.wifi_ssid_page();
	ui::WifiConfigPage *password_page = registry.wifi_password_page();
	REQUIRE(ssid_page != nullptr);
	REQUIRE(password_page != nullptr);
	ssid_page->on_ssid_selected("Office");
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi_password());
	password_page->on_ok_click("secret");
	CHECK(navigator.top_route() ==
	      ui::SettingsRoute::wifi_connect_result());
	REQUIRE(connect_requests.size() == 1U);
	CHECK(connect_requests.front().first == "Office");
	CHECK(connect_requests.front().second == "secret");

	const std::size_t route_count_before_success = navigator.size();
	flow.set_runtime_wifi_active(true);
	CHECK(connectivity == std::vector<std::pair<bool, bool>> {
				      { false, false }, { true, false } });
	CHECK(render_routes.back() == ui::SettingsRoute::wifi_connect_result());
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());
	CHECK(navigator.size() == route_count_before_success + 1U);

	const std::size_t route_count_after_success = navigator.size();
	flow.set_runtime_wifi_active(true);
	CHECK(connectivity ==
	      std::vector<std::pair<bool, bool>> {
		      { false, false }, { true, false }, { true, false } });
	CHECK(navigator.size() == route_count_after_success);

	REQUIRE(navigator.show_root_page(ui::RootPager::ContentPage::Main));
	settle_transition(lvgl);
	CHECK(visibility == std::vector<bool> { true, false });
	actions = {};
	wifi_page->on_wifi_enabled_change(false);
	CHECK(enabled_requests == std::vector<bool> { true });
	CHECK_FALSE(wifi_page->on_ssid_raster_prepare(raster_request));
	wifi_page->on_ssid_raster_invalidate(29);
	CHECK(ssid_raster_invalidations == std::vector<uint64_t> { 23 });
}

TEST_CASE("WifiFlowCoordinator replays connectivity stored before page binding",
	  "[ui][wifi-flow][connectivity]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	std::vector<std::pair<bool, bool>> connectivity;
	ui::PageManagerActions actions;
	ui::WifiFlowCoordinator::Callbacks callbacks;
	callbacks.render_connectivity =
		[&connectivity](ui::WifiConnectivityViewModel model) {
			connectivity.emplace_back(model.active,
						  model.boot_reconnect_pending);
		};

	ui::WifiFlowCoordinator flow(registry, navigator, actions.wifi, true,
				     100, callbacks);
	flow.set_runtime_wifi_active(false);
	flow.set_boot_reconnect_pending(true);
	connectivity.clear();

	flow.bind_pages();
	REQUIRE(connectivity ==
		std::vector<std::pair<bool, bool>> { { false, true } });

	flow.set_runtime_wifi_active(true);
	CHECK(connectivity == std::vector<std::pair<bool, bool>> {
				      { false, true }, { true, false } });
}

TEST_CASE("WifiFlowCoordinator keeps Via after QR cancellation reconnect",
	  "[ui][wifi-flow][qr]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	int qr_stop_requests = 0;
	int qr_stop_after_success_requests = 0;
	ui::PageManagerActions actions;
	actions.wifi.stop_qr_ap = [&qr_stop_requests]() {
		++qr_stop_requests;
		return uint64_t { 41 };
	};
	actions.wifi.stop_qr_ap_after_success =
		[&qr_stop_after_success_requests]() {
			++qr_stop_after_success_requests;
		};

	ui::WifiFlowCoordinator flow(registry, navigator, actions.wifi, true, 100);
	flow.bind_pages();
	flow.set_runtime_wifi_active(false);

	pager.transition_to(ui::RootPager::ContentPage::SettingsHome);
	settle_transition(lvgl);
	flow.show_wifi();
	ui::WifiPage *wifi_page = registry.wifi_page();
	ui::WifiConnectMethodPage *connect_method_page =
		registry.wifi_connect_method_page();
	ui::WifiQrPage *qr_page = registry.wifi_qr_page();
	REQUIRE(wifi_page != nullptr);
	REQUIRE(connect_method_page != nullptr);
	REQUIRE(qr_page != nullptr);

	wifi_page->on_wifi_config_click();
	REQUIRE(navigator.top_route() == ui::SettingsRoute::wifi_connect_method());
	connect_method_page->on_qr_click();
	REQUIRE(navigator.top_route() == ui::SettingsRoute::wifi_qr());

	lv_obj_send_event(qr_page->get_container(), LV_EVENT_CLICKED, nullptr);
	REQUIRE(qr_stop_requests == 1);
	flow.set_ap_stop_completed_generation(41);
	REQUIRE(navigator.top_route() == ui::SettingsRoute::wifi_connect_method());

	const std::size_t route_count_before_reconnect = navigator.size();
	flow.set_runtime_wifi_active(true);
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi_connect_method());
	CHECK(navigator.size() == route_count_before_reconnect);
	CHECK(qr_stop_after_success_requests == 0);

	flow.set_runtime_wifi_active(false);
	flow.set_runtime_wifi_active(true);
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi_connect_method());
	CHECK(navigator.size() == route_count_before_reconnect);

	connect_method_page->on_key_click();
	REQUIRE(navigator.top_route() == ui::SettingsRoute::wifi_ssid());
	flow.set_runtime_wifi_active(false);
	const std::size_t route_count_after_user_navigation = navigator.size();
	flow.set_runtime_wifi_active(true);
	CHECK(navigator.top_route() == ui::SettingsRoute::wifi());
	CHECK(navigator.size() == route_count_after_user_navigation + 1U);
}

TEST_CASE(
	"WifiFlowCoordinator bounds boot reconnect pending and cancels its timer",
	"[ui][wifi-flow][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	std::vector<std::pair<bool, bool>> connectivity;
	int timeout_count = 0;
	ui::PageManagerActions actions;
	ui::WifiFlowCoordinator::Callbacks callbacks;
	callbacks.render_connectivity =
		[&connectivity](ui::WifiConnectivityViewModel model) {
			connectivity.emplace_back(model.active,
						  model.boot_reconnect_pending);
		};
	callbacks.on_boot_reconnect_timeout = [&timeout_count]() {
		++timeout_count;
	};

	ui::WifiFlowCoordinator flow(registry, navigator, actions.wifi, true,
				     100, callbacks);
	const std::size_t timer_count_before = lvgl.timer_count();
	flow.set_boot_reconnect_pending(true);
	CHECK(connectivity ==
	      std::vector<std::pair<bool, bool>> { { true, true } });
	CHECK(lvgl.timer_count() == timer_count_before + 1U);

	lvgl.advance(100);
	CHECK(connectivity == std::vector<std::pair<bool, bool>> {
				      { true, true }, { true, false } });
	CHECK(timeout_count == 1);

	const std::size_t timer_count_after_expiry = lvgl.timer_count();
	flow.set_boot_reconnect_pending(true);
	CHECK(lvgl.timer_count() == timer_count_after_expiry + 1U);
	flow.shutdown();
	CHECK(lvgl.timer_count() == timer_count_after_expiry);
}

TEST_CASE("WifiFlowCoordinator characterizes repeated connectivity and shutdown",
	  "[ui][wifi-flow][characterization][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages content_pages = create_content_pages(pager);
	install_page_provider(pager, content_pages);

	ui::PageRegistry registry;
	registry.create(pager.settings_host());
	ui::SettingsNavigator navigator(pager, registry);
	bool rendered_active = true;
	bool rendered_pending = false;
	int timeout_count = 0;
	std::vector<std::string> events;
	ui::PageManagerActions actions;
	ui::WifiFlowCoordinator::Callbacks callbacks;
	callbacks.render_connectivity =
		[&rendered_active, &rendered_pending,
		 &events](ui::WifiConnectivityViewModel model) {
			rendered_active = model.active;
			rendered_pending = model.boot_reconnect_pending;
			events.push_back(
				model.active ?
					(model.boot_reconnect_pending ?
						 "connectivity:on:pending" :
						 "connectivity:on") :
					(model.boot_reconnect_pending ?
						 "connectivity:off:pending" :
						 "connectivity:off"));
		};
	callbacks.on_boot_reconnect_timeout = [&events, &rendered_active,
					       &timeout_count]() {
		++timeout_count;
		events.push_back(rendered_active ? "timeout:on" :
						   "timeout:off");
	};

	ui::WifiFlowCoordinator flow(registry, navigator, actions.wifi, true,
				     100, callbacks);
	flow.bind_pages();
	events.clear();
	flow.set_runtime_wifi_active(false);
	flow.set_boot_reconnect_pending(true);
	flow.set_boot_reconnect_pending(true);
	flow.set_runtime_wifi_active(true);

	CHECK(events == std::vector<std::string> { "connectivity:off",
						   "connectivity:off:pending",
						   "connectivity:on" });
	CHECK_FALSE(rendered_pending);

	const std::size_t route_count_before_repeat = navigator.size();
	events.clear();
	flow.set_runtime_wifi_active(true);
	CHECK(events == std::vector<std::string> { "connectivity:on" });
	CHECK(navigator.size() == route_count_before_repeat);

	flow.set_runtime_wifi_active(false);
	flow.set_boot_reconnect_pending(true);
	events.clear();
	lvgl.advance(100);
	CHECK(events ==
	      std::vector<std::string> { "connectivity:off", "timeout:off" });
	CHECK(timeout_count == 1);

	flow.set_boot_reconnect_pending(true);
	events.clear();
	flow.shutdown();
	lvgl.advance(101);
	CHECK(events.empty());
}
