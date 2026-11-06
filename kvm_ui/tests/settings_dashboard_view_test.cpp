#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "tailscale_projection.hpp"
#include "ui/settings_dashboard_view.hpp"
#include "ui/settings_page.hpp"
#include "ui/settings_toggle_coordinator.hpp"

TEST_CASE("SettingsDashboardView owns Settings cards and tile presentation",
	  "[ui][settings-dashboard][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	ui::SettingsDashboardView dashboard(false, 8000);
	int wifi_clicks = 0;
	dashboard.on_wifi_click = [&wifi_clicks]() {
		++wifi_clicks;
	};

	dashboard.create(lvgl.screen(), nullptr, 15000);

	ui::SettingsPage *settings_page = dashboard.settings_page();
	REQUIRE(settings_page != nullptr);
	REQUIRE(settings_page->get_container() != nullptr);
	REQUIRE(dashboard.second_page_object() != nullptr);
	lv_obj_update_layout(lvgl.screen());
	CHECK(lv_obj_get_x(settings_page->get_container()) == 282);
	CHECK(lv_obj_get_x(dashboard.second_page_object()) == 564);

	CHECK_FALSE(settings_page->tile_is_visible(
		ui::SettingsPage::SettingsTile::REGION));
	CHECK_FALSE(settings_page->tile_is_visible(
		ui::SettingsPage::SettingsTile::TAILSCALE));

	settings_page->on_wifi_click();
	CHECK(wifi_clicks == 1);

	ui::ControlState *wifi_control = settings_page->get_wifi_btn();
	ui::ControlState *mcp_control = settings_page->get_mcp_btn();
	REQUIRE(wifi_control != nullptr);
	REQUIRE(mcp_control != nullptr);
	CHECK(wifi_control->is_active());

	dashboard.set_wifi_tile_active(false);
	CHECK_FALSE(wifi_control->is_active());
	dashboard.set_wifi_tile_pending(true);
	CHECK(wifi_control->is_pending());
	dashboard.set_wifi_tile_pending(false);
	CHECK_FALSE(wifi_control->is_pending());

	ui::SettingsToggleViewModel model {
		.confirmed = true,
	};
	dashboard.render_toggle(ui::SettingsToggleId::MCP, model, 500);
	CHECK(mcp_control->is_active());
	CHECK_FALSE(mcp_control->is_pending());

	model.action_pending = true;
	model.pending_target = false;
	dashboard.render_toggle(ui::SettingsToggleId::MCP, model, 500);
	CHECK_FALSE(mcp_control->is_active());
	CHECK(mcp_control->is_pending());

	dashboard.begin_shutdown();
	CHECK_FALSE(static_cast<bool>(settings_page->on_wifi_click));
	dashboard.shutdown();
	CHECK(dashboard.settings_page() == nullptr);
	CHECK(dashboard.second_page_object() == nullptr);
	CHECK(lvgl.timer_count() == timer_count_before);
}

TEST_CASE("SettingsDashboardView shows the Network tile by default",
	  "[ui][settings-dashboard][network-tile]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	ui::SettingsDashboardView dashboard(false, 8000);
	dashboard.create(lvgl.screen(), nullptr, 15000);

	ui::SettingsPage *settings_page = dashboard.settings_page();
	REQUIRE(settings_page != nullptr);

	CHECK(settings_page->tile_is_visible(
		ui::SettingsPage::SettingsTile::NETWORK));

	int network_clicks = 0;
	dashboard.on_network_click = [&network_clicks]() {
		++network_clicks;
	};
	REQUIRE(settings_page->on_network_click != nullptr);
	settings_page->on_network_click();
	CHECK(network_clicks == 1);

	dashboard.begin_shutdown();
	dashboard.shutdown();
	CHECK(lvgl.timer_count() == timer_count_before);
}
