#include <memory>
#include <string>
#include <string_view>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "tailscale_projection.hpp"
#include "ui/apps_page.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/page_manager.hpp"
#include "ui/page_manager_actions.hpp"
#include "ui/settings_state_action_policy.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_main_mouse_active);
LV_IMAGE_DECLARE(img_main_mouse_default);
LV_IMAGE_DECLARE(settings_eth);
LV_IMAGE_DECLARE(network_terms_check);
}

namespace
{

constexpr uint32_t ROOT_TRANSITION_SETTLE_MS = 400;

void settle_root_transition(test_support::LvglTestContext &lvgl)
{
	lvgl.advance(ROOT_TRANSITION_SETTLE_MS);
}

bool is_effectively_visible(lv_obj_t *object)
{
	for (lv_obj_t *current = object; current != nullptr;
	     current = lv_obj_get_parent(current)) {
		if (lv_obj_has_flag(current, LV_OBJ_FLAG_HIDDEN)) {
			return false;
		}
	}
	return true;
}

lv_obj_t *find_tile_with_icon(lv_obj_t *parent, const lv_image_dsc_t *icon)
{
	if (parent == nullptr || icon == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_image_class) &&
		    lv_image_get_src(child) == icon) {
			return lv_obj_get_parent(child);
		}
		if (lv_obj_t *const match = find_tile_with_icon(child, icon);
		    match != nullptr) {
			return match;
		}
	}
	return nullptr;
}

lv_obj_t *find_image_with_source(lv_obj_t *parent,
				 const lv_image_dsc_t *image_source)
{
	if (parent == nullptr || image_source == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_image_class) &&
		    lv_image_get_src(child) == image_source) {
			return child;
		}
		if (lv_obj_t *const match =
			    find_image_with_source(child, image_source);
		    match != nullptr) {
			return match;
		}
	}
	return nullptr;
}

lv_obj_t *find_visible_label_parent(lv_obj_t *parent, std::string_view text)
{
	if (parent == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    is_effectively_visible(child) &&
		    lv_label_get_text(child) == text) {
			return lv_obj_get_parent(child);
		}
		if (lv_obj_t *const match =
			    find_visible_label_parent(child, text);
		    match != nullptr) {
			return match;
		}
	}
	return nullptr;
}

bool has_direct_label(lv_obj_t *parent, std::string_view text)
{
	if (parent == nullptr) {
		return false;
	}
	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    lv_label_get_text(child) == text) {
			return true;
		}
	}
	return false;
}

lv_obj_t *find_visible_ssid_fallback_label(lv_obj_t *parent,
						  std::string_view text)
{
	if (parent == nullptr) {
		return nullptr;
	}
	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    is_effectively_visible(child) && lv_label_get_text(child) == text &&
		    lv_obj_get_width(parent) == 230 && lv_obj_get_height(parent) == 32 &&
		    has_direct_label(parent, "SSID:")) {
			return child;
		}
		if (lv_obj_t *const match =
			    find_visible_ssid_fallback_label(child, text);
		    match != nullptr) {
			return match;
		}
	}
	return nullptr;
}

}

TEST_CASE("PageManager keeps root navigation and Wi-Fi visibility stable",
	  "[ui][page-manager][characterization]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	std::vector<bool> wifi_visibility;
	int wifi_scan_requests = 0;

	ui::PageManagerActions actions;
	actions.wifi.page_visibility_changed =
		[&wifi_visibility](bool visible) {
			wifi_visibility.push_back(visible);
		};
	actions.wifi.scan = [&wifi_scan_requests]() {
		++wifi_scan_requests;
	};
	manager.bind_actions(actions);
	manager.init(lvgl.screen());

	REQUIRE(manager.is_initialized());
	CHECK(manager.is_main_visible());
	CHECK_FALSE(manager.is_settings_visible());

	manager.show_settings();
	settle_root_transition(lvgl);
	CHECK_FALSE(manager.is_main_visible());
	CHECK(manager.is_settings_visible());

	manager.show_wifi_config();
	CHECK(manager.is_settings_visible());
	CHECK(wifi_scan_requests == 1);
	REQUIRE(wifi_visibility.size() == 1);
	CHECK(wifi_visibility.front());

	manager.show_main();
	settle_root_transition(lvgl);
	CHECK(manager.is_main_visible());
	CHECK_FALSE(manager.is_settings_visible());
	REQUIRE(wifi_visibility.size() == 2);
	CHECK_FALSE(wifi_visibility.back());
}

TEST_CASE("PageManager supports reentrant Wi-Fi visibility callbacks",
	  "[ui][page-manager][characterization][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	std::vector<bool> wifi_visibility;

	ui::PageManagerActions actions;
	actions.wifi.page_visibility_changed =
		[&manager, &wifi_visibility](bool visible) {
			wifi_visibility.push_back(visible);
			if (visible) {
				manager.show_main();
			}
		};
	manager.bind_actions(actions);
	manager.init(lvgl.screen());

	manager.show_wifi();
	settle_root_transition(lvgl);

	CHECK(manager.is_main_visible());
	CHECK_FALSE(manager.is_settings_visible());
	REQUIRE(wifi_visibility.size() == 2);
	CHECK(wifi_visibility[0]);
	CHECK_FALSE(wifi_visibility[1]);
}

TEST_CASE("PageManager resolves Settings subpage actions at event time",
	  "[ui][page-manager][settings]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	std::vector<bool> before_init_targets;
	ui::PageManagerActions before_init_actions;
	before_init_actions.settings.set_ncm_enabled =
		[&before_init_targets](bool enabled) {
			before_init_targets.push_back(enabled);
		};
	manager.bind_actions(before_init_actions);
	manager.init(lvgl.screen());

	manager.set_runtime_ncm_active(false);
	manager.show_settings();
	settle_root_transition(lvgl);

	lv_obj_t *const ncm_tile =
		find_tile_with_icon(lvgl.screen(), &settings_eth);
	REQUIRE(ncm_tile != nullptr);
	lv_obj_send_event(ncm_tile, LV_EVENT_CLICKED, nullptr);

	lv_obj_t *const toggle =
		find_visible_label_parent(lvgl.screen(), "OFF");
	REQUIRE(toggle != nullptr);
	lv_obj_send_event(toggle, LV_EVENT_CLICKED, nullptr);

	CHECK(before_init_targets == std::vector<bool> { true });

	manager.set_runtime_ncm_active(true);
	lvgl.advance(
		ui::settings_state_action_policy::kSettingsStateMinLoadingMs +
		1U);

	std::vector<bool> after_init_targets;
	ui::PageManagerActions after_init_actions;
	after_init_actions.settings.set_ncm_enabled =
		[&after_init_targets](bool enabled) {
			after_init_targets.push_back(enabled);
		};
	manager.bind_actions(after_init_actions);
	lv_obj_t *const enabled_toggle =
		find_visible_label_parent(lvgl.screen(), "ON");
	REQUIRE(enabled_toggle != nullptr);
	lv_obj_send_event(enabled_toggle, LV_EVENT_CLICKED, nullptr);

	CHECK(after_init_targets == std::vector<bool> { false });
	manager.set_runtime_ncm_active(false);
	lvgl.advance(
		ui::settings_state_action_policy::kSettingsStateMinLoadingMs +
		1U);

	std::vector<bool> rebound_targets;
	ui::PageManagerActions rebound_actions;
	rebound_actions.settings.set_ncm_enabled =
		[&rebound_targets](bool enabled) {
			rebound_targets.push_back(enabled);
		};
	manager.bind_actions(rebound_actions);
	lv_obj_t *const rebound_toggle =
		find_visible_label_parent(lvgl.screen(), "OFF");
	REQUIRE(rebound_toggle != nullptr);
	lv_obj_send_event(rebound_toggle, LV_EVENT_CLICKED, nullptr);

	CHECK(rebound_targets == std::vector<bool> { true });
	CHECK(before_init_targets == std::vector<bool> { true });
	CHECK(after_init_targets == std::vector<bool> { false });
}

TEST_CASE("PageManager wires optimistic Settings presentation at construction",
	  "[ui][page-manager][settings]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager(
		ui::BackgroundAssets {}, ui::PageManager::PageTransitionMode::Slide,
		false, 15000, 8000, true, true);
	std::vector<bool> targets;
	ui::PageManagerActions actions;
	actions.settings.set_ncm_enabled = [&targets](bool enabled) {
		targets.push_back(enabled);
	};
	manager.bind_actions(actions);
	manager.init(lvgl.screen());
	manager.set_runtime_ncm_active(false);
	manager.show_settings();
	settle_root_transition(lvgl);

	lv_obj_t *const ncm_tile =
		find_tile_with_icon(lvgl.screen(), &settings_eth);
	REQUIRE(ncm_tile != nullptr);
	lv_obj_send_event(ncm_tile, LV_EVENT_CLICKED, nullptr);

	lv_obj_t *const disabled_toggle =
		find_visible_label_parent(lvgl.screen(), "OFF");
	REQUIRE(disabled_toggle != nullptr);
	lv_obj_send_event(disabled_toggle, LV_EVENT_CLICKED, nullptr);
	CHECK(targets == std::vector<bool> { true });

	lv_obj_t *const optimistic_toggle =
		find_visible_label_parent(lvgl.screen(), "ON");
	REQUIRE(optimistic_toggle != nullptr);
	CHECK(test_support::find_visible_descendant_of_type(
		      optimistic_toggle, &lv_arc_class) == nullptr);

	lv_obj_send_event(optimistic_toggle, LV_EVENT_CLICKED, nullptr);
	CHECK(targets == std::vector<bool> { true });
	lv_obj_t *const final_intent_toggle =
		find_visible_label_parent(lvgl.screen(), "OFF");
	REQUIRE(final_intent_toggle != nullptr);
	CHECK(test_support::find_visible_descendant_of_type(
		      final_intent_toggle, &lv_arc_class) == nullptr);

	manager.set_runtime_ncm_active(false);
	lvgl.advance(ui::settings_state_action_policy::max_pending_wait_ms(
		ui::SettingsToggleId::NCM));
	CHECK(targets == std::vector<bool> { true });
	lv_obj_t *const confirmed_toggle =
		find_visible_label_parent(lvgl.screen(), "OFF");
	REQUIRE(confirmed_toggle != nullptr);
	CHECK(test_support::find_visible_descendant_of_type(
		      confirmed_toggle, &lv_arc_class) == nullptr);
}

TEST_CASE("PageManager preserves missing action fallbacks without stale work",
	  "[ui][page-manager][characterization]")
{
	SECTION("Settings toggle does not retain pending state")
	{
		test_support::LvglTestContext lvgl;
		ui::PageManager manager;
		manager.init(lvgl.screen());
		manager.set_runtime_ncm_active(false);
		manager.show_settings();
		settle_root_transition(lvgl);

		lv_obj_t *const ncm_tile =
			find_tile_with_icon(lvgl.screen(), &settings_eth);
		REQUIRE(ncm_tile != nullptr);
		lv_obj_send_event(ncm_tile, LV_EVENT_CLICKED, nullptr);

		lv_obj_t *const toggle =
			find_visible_label_parent(lvgl.screen(), "OFF");
		REQUIRE(toggle != nullptr);
		const std::size_t timer_count_before = lvgl.timer_count();
		lv_obj_send_event(toggle, LV_EVENT_CLICKED, nullptr);

		CHECK(lvgl.timer_count() == timer_count_before);
		CHECK(test_support::find_visible_descendant_of_type(
			      toggle, &lv_arc_class) == nullptr);
	}

	SECTION("Panel Sleep ignores a missing callback before pending")
	{
		test_support::LvglTestContext lvgl;
		ui::PageManager manager;
		int stale_sleep_requests = 0;
		ui::PageManagerActions actions;
		actions.panel.set_sleep_enabled =
			[&stale_sleep_requests](bool) {
				++stale_sleep_requests;
			};
		manager.bind_actions(actions);
		manager.init(lvgl.screen());
		manager.show_panel();
		lv_obj_update_layout(lvgl.screen());
		manager.bind_actions({});

		lv_obj_t *const sleep_button =
			find_visible_label_parent(lvgl.screen(), "Sleep");
		REQUIRE(sleep_button != nullptr);
		const std::size_t timer_count_before = lvgl.timer_count();
		lv_obj_send_event(sleep_button, LV_EVENT_CLICKED, nullptr);

		CHECK(lvgl.timer_count() == timer_count_before);
		CHECK(stale_sleep_requests == 0);
		CHECK(test_support::find_visible_descendant_of_type(
			      sleep_button, &lv_arc_class) == nullptr);
	}

	SECTION("About Reset remains unlocked when no request is accepted")
	{
		test_support::LvglTestContext lvgl;
		ui::PageManager manager;
		manager.init(lvgl.screen());
		manager.show_about();
		lv_obj_update_layout(lvgl.screen());

		lv_obj_t *const reset_button =
			find_visible_label_parent(lvgl.screen(), "Reset");
		REQUIRE(reset_button != nullptr);
		constexpr uint32_t kResetHoldDurationMs = 2000;
		constexpr uint32_t kResetFillDurationMs =
			kResetHoldDurationMs -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		lvgl.press(reset_button);
		lvgl.advance(kResetFillDurationMs);
		lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		lvgl.release();

		CHECK(lv_obj_has_flag(reset_button, LV_OBJ_FLAG_CLICKABLE));
		CHECK(test_support::find_visible_descendant_of_type(
			      reset_button, &lv_arc_class) == nullptr);
		manager.show_settings_home();
		settle_root_transition(lvgl);
		CHECK(manager.is_settings_visible());
	}

	SECTION("Apps launch reads the current callback instead of a stale one")
	{
		test_support::LvglTestContext lvgl;
		ui::PageManager manager;
		manager.init(lvgl.screen());
		manager.set_apps({ { "No Launch", "/apps/no-launch" } });
		int stale_launches = 0;
		ui::PageManagerActions actions;
		actions.apps.launch = [&stale_launches](const std::string &) {
			++stale_launches;
		};
		manager.bind_actions(actions);
		manager.show_apps();
		lv_obj_update_layout(lvgl.screen());

		lv_obj_t *const app_button =
			find_visible_label_parent(lvgl.screen(), "No Launch");
		REQUIRE(app_button != nullptr);
		manager.bind_actions({});
		lv_obj_send_event(app_button, LV_EVENT_CLICKED, nullptr);

		CHECK(stale_launches == 0);
	}
}

TEST_CASE(
	"PageManager does not refresh settings state for rejected root navigation",
	"[ui][page-manager][root-navigation-regression]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager(ui::BackgroundAssets {},
				ui::PageManager::PageTransitionMode::Flip);
	int refresh_requests = 0;
	ui::PageManagerActions actions;
	actions.settings.refresh_service_state = [&refresh_requests]() {
		++refresh_requests;
	};
	manager.bind_actions(actions);
	manager.show_settings_home();
	REQUIRE(refresh_requests == 0);

	manager.init(lvgl.screen());
	REQUIRE(refresh_requests == 1);

	manager.show_settings();
	REQUIRE(refresh_requests == 2);
	CHECK(manager.is_main_visible());

	manager.show_main();
	manager.show_settings();
	manager.show_settings_home();

	CHECK(refresh_requests == 2);
	CHECK(manager.is_main_visible());

	settle_root_transition(lvgl);
	CHECK(manager.is_settings_visible());
	manager.show_main();
	CHECK(refresh_requests == 3);
}

TEST_CASE(
	"PageManager applies Wi-Fi active received during boot pending before returning",
	"[ui][page-manager][wifi-pending-regression]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager(ui::BackgroundAssets {},
				ui::PageManager::PageTransitionMode::Slide,
				true, 100);
	manager.init(lvgl.screen());

	REQUIRE(manager.settings_wifi_tile_presentation().created);

	manager.set_runtime_wifi_active(false);
	REQUIRE_FALSE(manager.settings_wifi_tile_presentation().active);
	manager.set_runtime_wifi_boot_reconnect_pending(true);
	REQUIRE(manager.settings_wifi_tile_presentation().pending);

	manager.set_runtime_wifi_active(true);

	CHECK_FALSE(manager.settings_wifi_tile_presentation().pending);
	CHECK(manager.settings_wifi_tile_presentation().active);
	lvgl.advance(101);
	CHECK(manager.settings_wifi_tile_presentation().active);
}

TEST_CASE(
	"PageManager applies the last Wi-Fi active state when boot pending times out",
	"[ui][page-manager][wifi-pending-regression]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager(ui::BackgroundAssets {},
				ui::PageManager::PageTransitionMode::Slide,
				true, 100);
	manager.init(lvgl.screen());

	REQUIRE(manager.settings_wifi_tile_presentation().created);
	REQUIRE(manager.settings_wifi_tile_presentation().active);

	manager.set_runtime_wifi_boot_reconnect_pending(true);
	REQUIRE(manager.settings_wifi_tile_presentation().pending);
	manager.set_runtime_wifi_active(false);
	REQUIRE(manager.settings_wifi_tile_presentation().pending);
	CHECK(manager.settings_wifi_tile_presentation().active);

	lvgl.advance(100);

	CHECK_FALSE(manager.settings_wifi_tile_presentation().pending);
	CHECK_FALSE(manager.settings_wifi_tile_presentation().active);
}

TEST_CASE(
	"PageManager keeps Wi-Fi runtime facts independent and applies subpage state synchronously",
	"[ui][page-manager][wifi-state-facts]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	REQUIRE(manager.settings_wifi_tile_presentation().created);

	manager.set_main_ip_candidates({ "192.0.2.10" });
	manager.set_runtime_wifi_active(true);
	manager.set_wifi_runtime_state(ui::WifiPageVisualState::Off,
					       "Disconnected", false, "", "", false, false,
					       std::string {}, std::string {});

	CHECK(find_visible_label_parent(lvgl.screen(), "192.0.2.10") !=
	      nullptr);
	CHECK(manager.settings_wifi_tile_presentation().active);

	manager.show_wifi();
	CHECK(find_visible_ssid_fallback_label(lvgl.screen(), "--") != nullptr);
	CHECK(manager.settings_wifi_tile_presentation().active);

	manager.set_wifi_runtime_state(ui::WifiPageVisualState::Connected,
					       "Connected", false, "Office WiFi", "192.0.2.8", true,
					       false,
					       std::string {}, std::string {});

	CHECK(find_visible_ssid_fallback_label(lvgl.screen(), "Office WiFi") !=
	      nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "192.0.2.8") != nullptr);
	CHECK(manager.settings_wifi_tile_presentation().active);

	manager.set_wifi_runtime_state(ui::WifiPageVisualState::Connected,
					       "Connected", false, "Office WiFi", "192.0.2.8", true,
					       false,
					       std::string {}, std::string {});
	CHECK(find_visible_ssid_fallback_label(lvgl.screen(), "Office WiFi") !=
	      nullptr);
}

TEST_CASE("PageManager teardown returns LVGL timer ownership to the context",
	  "[ui][page-manager][characterization][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	int refresh_requests = 0;

	for (int iteration = 0; iteration < 2; ++iteration) {
		auto manager = std::make_unique<ui::PageManager>(
			ui::BackgroundAssets {},
			ui::PageManager::PageTransitionMode::Flip);
		ui::PageManagerActions actions;
		actions.settings.refresh_service_state = [&refresh_requests]() {
			++refresh_requests;
		};
		manager->bind_actions(actions);
		manager->init(lvgl.screen());
		REQUIRE(refresh_requests > 0);
		manager->set_main_mouse_active(true);
		manager->set_main_ip_empty_placeholder("Reconnecting");
		lv_obj_t *const active_mouse = find_image_with_source(
			lvgl.screen(), &img_main_mouse_active);
		REQUIRE(active_mouse != nullptr);
		CHECK(is_effectively_visible(active_mouse));
		lvgl.advance(801);
		lv_obj_t *const default_mouse = find_image_with_source(
			lvgl.screen(), &img_main_mouse_default);
		REQUIRE(default_mouse != nullptr);
		CHECK(is_effectively_visible(default_mouse));
		manager->set_runtime_wifi_boot_reconnect_pending(true);
		const auto wifi_tile =
			manager->settings_wifi_tile_presentation();
		REQUIRE(wifi_tile.created);
		REQUIRE(wifi_tile.pending);
		REQUIRE(lvgl.timer_count() > timer_count_before);

		const int refresh_requests_before_flip = refresh_requests;
		manager->show_settings();
		REQUIRE(refresh_requests == refresh_requests_before_flip + 1);
		manager->show_main();
		CHECK(refresh_requests == refresh_requests_before_flip + 1);

		const int refresh_requests_at_shutdown = refresh_requests;
		manager.reset();

		CHECK(lvgl.timer_count() == timer_count_before);
		lvgl.advance(16000);
		CHECK(lvgl.timer_count() == timer_count_before);
		CHECK(refresh_requests == refresh_requests_at_shutdown);
	}
}

namespace
{

constexpr uint32_t kTailscaleHoldMs = 2000;

bool has_pending_arc(lv_obj_t *object)
{
	return test_support::find_visible_descendant_of_type(
		       object, &lv_arc_class) != nullptr;
}

}

TEST_CASE("PageManager projects and replays the tailscale snapshot",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = false;
	manager.set_tailscale_snapshot(snapshot);

	manager.show_tailscale();
	settle_root_transition(lvgl);
	CHECK(manager.is_settings_visible());
	CHECK(find_visible_label_parent(lvgl.screen(), "Start") != nullptr);

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	ready.online = true;
	ready.ip = "100.64.0.1";
	manager.set_tailscale_snapshot(ready);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);

	manager.set_tailscale_snapshot(ready);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, true, "",
					    1);
	manager.show_main();
	settle_root_transition(lvgl);
	manager.show_tailscale();
	settle_root_transition(lvgl);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);
}

TEST_CASE("PageManager resolves tailscale actions at event time",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	ready.online = true;
	ready.ip = "100.64.0.1";
	manager.set_tailscale_snapshot(ready);
	manager.show_tailscale();
	settle_root_transition(lvgl);

	int start_requests = 0;
	int exit_requests = 0;
	ui::PageManagerActions actions;
	actions.tailscale.start = [&start_requests]() -> bool {
		++start_requests;
		return true;
	};
	actions.tailscale.exit = [&exit_requests]() -> bool {
		++exit_requests;
		return true;
	};
	manager.bind_actions(actions);

	lv_obj_t *const button =
		find_visible_label_parent(lvgl.screen(), "Exit");
	REQUIRE(button != nullptr);
	lvgl.press(button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();

	CHECK(exit_requests == 1);
	CHECK(start_requests == 0);
}

TEST_CASE("PageManager keeps the tailscale exit failure message across heartbeats",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	ready.online = true;
	ready.ip = "100.64.0.1";
	manager.set_tailscale_snapshot(ready);
	manager.show_tailscale();
	settle_root_transition(lvgl);

	ui::PageManagerActions actions;
	actions.tailscale.exit = []() -> bool { return true; };
	manager.bind_actions(actions);

	lv_obj_t *const button =
		find_visible_label_parent(lvgl.screen(), "Exit");
	REQUIRE(button != nullptr);
	lvgl.press(button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, false,
					    "FAIL", 1);
	CHECK(find_visible_label_parent(lvgl.screen(), "FAIL") != nullptr);

	manager.set_tailscale_snapshot(ready);
	manager.set_tailscale_snapshot(ready);
	CHECK(find_visible_label_parent(lvgl.screen(), "FAIL") != nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);
}

TEST_CASE("PageManager arms the tailscale override once per action seq",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	manager.set_tailscale_snapshot(ready);
	const uint64_t initial_seq = manager.tailscale_snapshot_seq();
	REQUIRE(initial_seq == 1);

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, true,
					    "", 1);
	manager.set_tailscale_snapshot(ready);
	CHECK(manager.tailscale_snapshot_seq() == initial_seq + 1);

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, true,
					    "", 1);
	manager.set_tailscale_snapshot(ready);
	CHECK(manager.tailscale_snapshot_seq() == initial_seq + 1);
	manager.set_tailscale_snapshot(ready);
	CHECK(manager.tailscale_snapshot_seq() == initial_seq + 1);

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, true,
					    "", 2);
	manager.set_tailscale_snapshot(ready);
	CHECK(manager.tailscale_snapshot_seq() == initial_seq + 2);
}

TEST_CASE("PageManager consumes the tailscale override on the next snapshot",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	ready.online = true;
	ready.ip = "100.64.0.1";
	manager.set_tailscale_snapshot(ready);

	ui::PageManagerActions actions;
	actions.tailscale.exit = []() -> bool { return true; };
	manager.bind_actions(actions);

	manager.show_tailscale();
	settle_root_transition(lvgl);

	lv_obj_t *button = find_visible_label_parent(lvgl.screen(), "Exit");
	REQUIRE(button != nullptr);
	lvgl.press(button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();
	REQUIRE(has_pending_arc(button));

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Exit, true,
					    "", 1);
	CHECK_FALSE(has_pending_arc(button));

	manager.set_tailscale_snapshot(ready);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);
}

TEST_CASE("the tailscale projection applies the action result before the snapshot",
	  "[ui][page-manager][characterization][tailscale]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	ui::TailscaleSnapshotView ready;
	ready.has_snapshot = true;
	ready.started = true;
	ready.matched = true;
	ready.online = true;
	ready.ip = "100.64.0.1";
	manager.set_tailscale_snapshot(ready);

	ui::PageManagerActions actions;
	actions.tailscale.exit = []() -> bool { return true; };
	manager.bind_actions(actions);

	manager.show_tailscale();
	settle_root_transition(lvgl);

	lv_obj_t *button = find_visible_label_parent(lvgl.screen(), "Exit");
	REQUIRE(button != nullptr);
	lvgl.press(button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();
	REQUIRE(has_pending_arc(button));

	lvgl_main::RuntimeStatus status;
	status.tailscale_gate_seq = 2;
	status.tailscale_online = true;
	status.tailscale_started = false;
	status.tailscale_matched = false;
	status.tailscale_action_seq = 1;
	status.tailscale_action_kind = lvgl_main::TailscaleActionKind::Exit;
	status.tailscale_action_ok = true;
	status.tailscale_seq = 1;

	lvgl_main::project_tailscale_status(manager, status);

	CHECK(find_visible_label_parent(lvgl.screen(), "Start") != nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") == nullptr);
}

TEST_CASE("the tailscale projection routes the client-init lanes into the "
	  "Network page",
	  "[ui][page-manager][characterization][network]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	int visibility_events = 0;
	bool last_visible = false;
	ui::PageManagerActions actions;
	actions.tailscale.page_visibility_changed =
		[&visibility_events, &last_visible](bool visible) {
			++visibility_events;
			last_visible = visible;
		};
	manager.bind_actions(actions);

	manager.show_network();
	settle_root_transition(lvgl);
	CHECK(manager.is_settings_visible());
	CHECK(find_visible_label_parent(lvgl.screen(), "ON") != nullptr);
	CHECK(visibility_events == 1);
	CHECK(last_visible);

	manager.show_main();
	settle_root_transition(lvgl);
	CHECK(visibility_events == 2);
	CHECK_FALSE(last_visible);

	lvgl_main::RuntimeStatus status;
	status.tailscale_gate_seq = 1;
	status.tailscale_started = true;
	status.tailscale_matched = false;
	status.tailscale_code = "123456";
	status.tailscale_seq = 7;
	lvgl_main::project_tailscale_status(manager, status);

	manager.show_network();
	settle_root_transition(lvgl);
	CHECK(manager.is_settings_visible());
	CHECK(find_visible_label_parent(lvgl.screen(), "123 456") != nullptr);
	CHECK(visibility_events == 3);
	CHECK(last_visible);

	status.tailscale_code = "654321";
	status.tailscale_seq = 8;
	lvgl_main::project_tailscale_status(manager, status);
	CHECK(find_visible_label_parent(lvgl.screen(), "654 321") != nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "123 456") == nullptr);

	manager.show_main();
	settle_root_transition(lvgl);
	CHECK(visibility_events == 4);
	CHECK_FALSE(last_visible);

	manager.show_network();
	settle_root_transition(lvgl);
	CHECK(find_visible_label_parent(lvgl.screen(), "654 321") != nullptr);
	CHECK(visibility_events == 5);
	CHECK(last_visible);
}

TEST_CASE("the tailscale projection routes the ready lane and exit request "
	  "through the Network page",
	  "[ui][page-manager][characterization][network]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	int exit_calls = 0;
	ui::PageManagerActions actions;
	actions.tailscale.exit = [&exit_calls]() -> bool {
		++exit_calls;
		return true;
	};
	manager.bind_actions(actions);

	lvgl_main::RuntimeStatus status;
	status.tailscale_gate_seq = 1;
	status.tailscale_started = true;
	status.tailscale_matched = true;
	status.tailscale_online = true;
	status.tailscale_peer_ip = "100.64.0.1";

	manager.show_network();
	settle_root_transition(lvgl);
	lvgl_main::project_tailscale_status(manager, status);
	lv_obj_update_layout(lvgl.screen());

	CHECK(manager.is_settings_visible());
	CHECK(find_visible_label_parent(lvgl.screen(), "100.64.0.1") != nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") != nullptr);

	lv_obj_t *button = find_visible_label_parent(lvgl.screen(), "Exit");
	lvgl.press(button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();
	CHECK(exit_calls == 1);
	REQUIRE(has_pending_arc(button));

	status.tailscale_gate_seq = 2;
	status.tailscale_started = false;
	status.tailscale_matched = false;
	status.tailscale_online = false;
	lvgl_main::project_tailscale_status(manager, status);
	lv_obj_update_layout(lvgl.screen());

	CHECK(find_visible_label_parent(lvgl.screen(), "Exit") == nullptr);
	CHECK(find_visible_label_parent(lvgl.screen(), "ON") != nullptr);
}

TEST_CASE("PageManager forwards a tailscale start failure to the Network "
	  "page message hold",
	  "[ui][page-manager][characterization][network]")
{
	test_support::LvglTestContext lvgl;
	ui::PageManager manager;
	manager.init(lvgl.screen());

	int start_calls = 0;
	ui::PageManagerActions actions;
	actions.tailscale.start = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	manager.bind_actions(actions);

	ui::TailscaleSnapshotView not_initialized;
	not_initialized.has_snapshot = true;
	not_initialized.started = false;
	not_initialized.matched = false;
	manager.set_network_snapshot(not_initialized);
	manager.show_network();
	settle_root_transition(lvgl);
	CHECK(find_visible_label_parent(lvgl.screen(), "ON") != nullptr);

	lv_obj_t *const on_button =
		find_visible_label_parent(lvgl.screen(), "ON");
	REQUIRE(on_button != nullptr);
	lvgl.press(on_button);
	lvgl.advance(kTailscaleHoldMs -
		     ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
	lvgl.release();
	lv_obj_update_layout(lvgl.screen());

	lv_obj_t *const agree_image =
		find_image_with_source(lvgl.screen(), &network_terms_check);
	REQUIRE(agree_image != nullptr);
	lv_obj_t *const agree =
		lv_obj_get_parent(lv_obj_get_parent(agree_image));
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	lv_obj_update_layout(lvgl.screen());
	CHECK(start_calls == 1);

	manager.set_tailscale_action_result(ui::TailscaleActionKind::Start,
					    false, "FAIL", 1);
	lv_obj_update_layout(lvgl.screen());
	CHECK(find_visible_label_parent(lvgl.screen(), "FAIL") != nullptr);

	manager.set_network_snapshot(not_initialized);
	not_initialized.online = true;
	manager.set_network_snapshot(not_initialized);
	not_initialized.online = false;
	manager.set_network_snapshot(not_initialized);
	lv_obj_update_layout(lvgl.screen());
	CHECK(find_visible_label_parent(lvgl.screen(), "FAIL") != nullptr);
}
