#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/simulator_runtime.hpp"

using namespace lvgl_main;

TEST_CASE("simulator runtime publishes its deterministic initial snapshot",
	  "[simulator][runtime]")
{
	SimulatorRuntimeService runtime;
	const auto status = runtime.current_status();

	CHECK(status.wifi_connected);
	CHECK(status.wifi_ssid == "NanoKVM-Simulator");
	CHECK(status.active_ip == "192.0.2.10");
	CHECK(status.hdmi_input_supported);
	CHECK(status.hid_enabled);
	CHECK(status.backlight_percent == 70);
	CHECK(status.hw_version == "x86_64 Simulator");
	CHECK_FALSE(status.server_available);
}

TEST_CASE("simulator runtime accepts constructor-only initial state injection",
	  "[simulator][runtime]")
{
	RuntimeStatus injected;
	injected.hid_enabled = false;
	injected.hw_version = "test simulator";
	SimulatorRuntimeService runtime({}, {
						    .initial_status = injected,
					    });

	CHECK_FALSE(runtime.current_status().hid_enabled);
	CHECK(runtime.current_status().hw_version == "test simulator");
}

TEST_CASE("simulator runtime starts once and invokes callbacks after unlock",
	  "[simulator][runtime]")
{
	SimulatorRuntimeService *runtime_ptr = nullptr;
	int callback_count = 0;
	RuntimeCallbacks callbacks;
	callbacks.on_status = [&](const RuntimeStatusPublication &) {
		++callback_count;
		REQUIRE(runtime_ptr != nullptr);
		CHECK(runtime_ptr->current_status().wifi_supported);
	};
	SimulatorRuntimeService runtime(std::move(callbacks));
	runtime_ptr = &runtime;

	REQUIRE(runtime.start());
	CHECK(runtime.is_running());
	CHECK(callback_count == 1);
	REQUIRE(runtime.start());
	CHECK(callback_count == 1);
	runtime.stop();
	CHECK_FALSE(runtime.is_running());
}

TEST_CASE("simulator runtime applies runtime controls entirely in memory",
	  "[simulator][runtime]")
{
	SimulatorRuntimeService runtime;
	REQUIRE(runtime.start());
	runtime.set_ssh_enabled(true);
	runtime.set_ncm_enabled(true);
	runtime.set_hdmi_input_enabled(false);
	runtime.set_hid_enabled(false);
	runtime.apply_backlight_percent_now(1000);
	runtime.set_auto_sleep_enabled(true);
	runtime.set_screen_sleep(true);

	const auto status = runtime.current_status();
	CHECK(status.ssh_enabled);
	CHECK(status.ncm_enabled);
	CHECK_FALSE(status.hdmi_input_enabled);
	CHECK_FALSE(status.hid_enabled);
	CHECK(status.backlight_percent == 100);
	CHECK(status.auto_sleep_enabled);
	CHECK_FALSE(status.screen_on);
}

TEST_CASE("simulator runtime preserves service completion and WiFi flows",
	  "[simulator][runtime]")
{
	std::vector<RuntimeStatus> updates;
	RuntimeCallbacks callbacks;
	callbacks.on_status = [&](const RuntimeStatusPublication &publication) {
		updates.push_back(publication.status);
	};
	SimulatorRuntimeService runtime(std::move(callbacks));
	REQUIRE(runtime.start());
	const uint64_t event_seq_before_mcp =
		runtime.current_status().local_services_event_seq;
	runtime.set_mcp_bundle_enabled(true);
	const auto after_mcp = runtime.current_status();
	CHECK(after_mcp.mcp_service_active);
	CHECK(after_mcp.local_services_event_seq == event_seq_before_mcp + 1U);

	runtime.set_screen_timelapse_enabled(true);
	const auto after_replay = runtime.current_status();
	CHECK(after_replay.mcp_service_active);
	CHECK(after_replay.replay_service_active);
	CHECK(after_replay.local_services_event_seq ==
	      after_mcp.local_services_event_seq + 1U);

	const std::size_t updates_before_scan = updates.size();
	runtime.scan_wifi_networks();
	CHECK(updates.size() == updates_before_scan + 2U);
	CHECK_FALSE(runtime.current_status().wifi_scan_running);
	CHECK(runtime.current_status().wifi_scan_result.ok);
	CHECK_FALSE(runtime.current_status().wifi_scan_result.networks.empty());

	runtime.connect_wifi_network("Lab-Guest", "not-retained");
	const auto connected = runtime.current_status();
	CHECK(connected.wifi_connect_result.status ==
	      WifiConnectStatus::Success);
	CHECK(connected.wifi_ssid == "Lab-Guest");
	CHECK(connected.wifi_connect_message.find("not-retained") ==
	      std::string::npos);

	runtime.start_wifi_qr_ap_config();
	CHECK(runtime.current_status().wifi_ap_mode);
	const uint64_t stop_generation = runtime.stop_wifi_qr_ap_config();
	CHECK(stop_generation > 0);
	CHECK(runtime.current_status().wifi_ap_stop_completed_generation ==
	      stop_generation);
	CHECK(runtime.current_status().wifi_connected);
}

TEST_CASE(
	"simulator runtime makes factory reset non-destructive and supports injected failures",
	"[simulator][runtime]")
{
	SimulatorRuntimeService runtime(
		{},
		{
			.failing_actions = { SimulatorRuntimeAction::SetHid },
		});
	REQUIRE(runtime.start());
	const auto before_hid = runtime.current_status();
	runtime.set_hid_enabled(false);
	CHECK(runtime.current_status().hid_enabled == before_hid.hid_enabled);

	const uint64_t before_reset =
		runtime.current_status().factory_reset_failure_generation;
	REQUIRE(runtime.request_factory_reset());
	CHECK(runtime.current_status().factory_reset_failure_generation ==
	      before_reset + 1U);
}

TEST_CASE("simulator runtime switches the region in memory",
	  "[simulator][runtime]")
{
	SimulatorRuntimeService runtime;
	REQUIRE(runtime.start());

	REQUIRE(runtime.request_region_switch(DeviceRegion::Cn));
	CHECK(runtime.current_status().region == DeviceRegion::Cn);
	CHECK(runtime.current_status().region_switch_failure_generation == 0);

	REQUIRE(runtime.request_region_switch(DeviceRegion::Global));
	CHECK(runtime.current_status().region == DeviceRegion::Global);
}

TEST_CASE("simulator runtime supports an injected region switch failure",
	  "[simulator][runtime]")
{
	SimulatorRuntimeService runtime(
		{},
		{
			.failing_actions = { SimulatorRuntimeAction::RegionSwitch },
		});
	REQUIRE(runtime.start());

	CHECK_FALSE(runtime.request_region_switch(DeviceRegion::Cn));
	const RuntimeStatus status = runtime.current_status();
	CHECK(status.region == DeviceRegion::Global);
	CHECK(status.region_switch_failure_generation == 1);
}
