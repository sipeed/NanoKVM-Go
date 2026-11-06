#include <condition_variable>
#include <cstddef>
#include <mutex>
#include <thread>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/runtime_state_store.hpp"
#include "lvgl_main/runtime_read_observation.hpp"

using namespace lvgl_main;

namespace
{

class TestGate final {
    public:
	void release()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_released = true;
		_condition.notify_all();
	}

	void wait()
	{
		std::unique_lock<std::mutex> lock(_mutex);
		_condition.wait(lock, [this] {
			return _released;
		});
	}

    private:
	std::mutex _mutex;
	std::condition_variable _condition;
	bool _released { false };
};

class TestRendezvous final {
    public:
	explicit TestRendezvous(std::size_t participants)
		: _participants(participants)
	{
	}

	void arrive_and_wait()
	{
		std::unique_lock<std::mutex> lock(_mutex);
		++_arrived;
		if (_arrived == _participants) {
			_released = true;
			_condition.notify_all();
			return;
		}
		_condition.wait(lock, [this] {
			return _released;
		});
	}

    private:
	std::mutex _mutex;
	std::condition_variable _condition;
	std::size_t _participants;
	std::size_t _arrived { 0 };
	bool _released { false };
};

}

TEST_CASE("systemd command observation distinguishes explicit negatives from failures")
{
	const auto active = observe_systemd_active_result(0, false);
	REQUIRE(active.is_observed());
	CHECK(active.value());

	const auto inactive = observe_systemd_active_result(3, false);
	REQUIRE(inactive.is_observed());
	CHECK_FALSE(inactive.value());

	const auto missing = observe_systemd_unit_exists_result(4, false);
	REQUIRE(missing.is_observed());
	CHECK_FALSE(missing.value());

	CHECK_FALSE(observe_systemd_active_result(0, true).is_observed());
	CHECK_FALSE(observe_systemd_active_result(-1, false).is_observed());
	CHECK_FALSE(observe_systemd_unit_exists_result(1, false).is_observed());
}

TEST_CASE("HTTP required-field observation retains explicit false and rejects absent data")
{
	const auto explicit_disabled =
		observe_required_http_field(std::optional<bool> { false });
	REQUIRE(explicit_disabled.is_observed());
	CHECK_FALSE(explicit_disabled.value());

	const auto explicit_mode = observe_required_http_field(
		std::optional<std::string> { "relative" });
	REQUIRE(explicit_mode.is_observed());
	CHECK(explicit_mode.value() == "relative");

	const auto explicit_empty =
		observe_required_http_field(std::optional<std::string> { "" });
	REQUIRE(explicit_empty.is_observed());
	CHECK(explicit_empty.value().empty());

	const auto explicit_zero =
		observe_required_http_field(std::optional<int> { 0 });
	REQUIRE(explicit_zero.is_observed());
	CHECK(explicit_zero.value() == 0);

	CHECK_FALSE(
		observe_required_http_field(std::optional<bool> {}).is_observed());
	CHECK_FALSE(observe_required_http_field(
			std::optional<std::string> {})
				.is_observed());
}

TEST_CASE("refresh observation preserves HTTP ingress committed while it ran")
{
	RuntimeStateStore store;
	RuntimeStatus observed;
	observed.eth_ip = "192.168.1.2";
	observed.main_ip_candidates = { observed.eth_ip };
	observed.app_version = "test";

	TestGate observation_ready;
	TestGate ingress_committed;
	std::thread refresh([&] {
		observation_ready.release();
		ingress_committed.wait();
		(void)store.commit(
			RefreshObservationPatch { .status = observed });
	});

	observation_ready.wait();
	REQUIRE(store.commit(DisplayIngressPatch {
		.resolution = "1920x1080",
		.fps = "60",
	}));
	ingress_committed.release();
	refresh.join();

	const RuntimeStatus current = store.current_status();
	CHECK(current.eth_ip == "192.168.1.2");
	CHECK(current.main_ip_candidates ==
	      std::vector<std::string> { "192.168.1.2" });
	CHECK(current.display_resolution_label == "1920x1080");
	CHECK(current.display_fps_label == "60");
	CHECK(current.display_resolution_label_generation == 1);
	CHECK(current.display_fps_label_generation == 1);
}

TEST_CASE("refresh observation updates its device fields only")
{
	RuntimeStatus initial;
	initial.ssh_supported = true;
	initial.ssh_enabled = true;
	initial.resolution = "1280x720";
	initial.fps = 60.0;
	initial.h264_bitrate_kbps = 4096;
	initial.display_resolution_label = "HTTP label";
	initial.display_resolution_label_generation = 4;
	initial.local_services_known = true;
	initial.local_services_event_seq = 7;
	initial.mcp_service_active = true;
	initial.factory_reset_failure_generation = 3;
	RuntimeStateStore store(initial);

	RuntimeStatus observed;
	observed.eth_ip = "192.168.1.2";
	observed.wifi_ip = "192.168.1.3";
	observed.ncm_ip = "192.168.42.2";
	observed.rndis_ip = "192.168.7.2";
	observed.active_ip = observed.eth_ip;
	observed.main_ip_candidates = { observed.eth_ip, observed.wifi_ip };
	observed.app_version = "app";
	observed.hw_version = "hw";
	observed.ncm_supported = true;
	observed.ncm_enabled = true;
	observed.hdmi_input_supported = true;
	observed.hdmi_input_enabled = true;
	observed.video_enabled = true;
	observed.hid_supported = true;
	observed.hid_enabled = true;
	observed.hid_mode = "normal";
	observed.backlight_percent = 75;
	observed.auto_sleep_enabled = true;
	observed.screen_on = false;
	observed.server_available = true;

	REQUIRE(store.commit(RefreshObservationPatch { .status = observed }));

	const RuntimeStatus current = store.current_status();
	CHECK(current.eth_ip == "192.168.1.2");
	CHECK(current.wifi_ip == "192.168.1.3");
	CHECK(current.ncm_ip == "192.168.42.2");
	CHECK(current.rndis_ip == "192.168.7.2");
	CHECK(current.main_ip_candidates ==
	      std::vector<std::string> { "192.168.1.2", "192.168.1.3" });
	CHECK(current.app_version == "app");
	CHECK(current.hw_version == "hw");
	CHECK(current.ncm_supported);
	CHECK(current.ncm_enabled);
	CHECK(current.hdmi_input_supported);
	CHECK(current.hdmi_input_enabled);
	CHECK(current.video_enabled);
	CHECK(current.hid_supported);
	CHECK(current.hid_enabled);
	CHECK(current.hid_mode == "normal");
	CHECK(current.backlight_percent == 75);
	CHECK(current.auto_sleep_enabled);
	CHECK_FALSE(current.screen_on);
	CHECK(current.server_available);

	CHECK(current.ssh_supported);
	CHECK(current.ssh_enabled);
	CHECK(current.resolution == "1280x720");
	CHECK(current.fps == 60.0);
	CHECK(current.h264_bitrate_kbps == 4096);
	CHECK(current.display_resolution_label == "HTTP label");
	CHECK(current.display_resolution_label_generation == 4);
	CHECK(current.local_services_known);
	CHECK(current.local_services_event_seq == 7);
	CHECK(current.mcp_service_active);
	CHECK(current.factory_reset_failure_generation == 3);
}

TEST_CASE("startup device region survives every refresh observation domain")
{
	RuntimeStatus initial;
	initial.region = DeviceRegion::Cn;
	RuntimeStateStore store(initial);

	RuntimeStatus observed;
	observed.app_version = "app";
	observed.region = DeviceRegion::Global;

	REQUIRE(store.commit(RefreshObservationPatch { .status = observed }));
	CHECK(store.current_status().region == DeviceRegion::Cn);

	RuntimeFlagsPatch flags {};
	REQUIRE(store.commit(flags));
	CHECK(store.current_status().region == DeviceRegion::Cn);
}

TEST_CASE("region switch failure generation advances monotonically")
{
	RuntimeStatus initial;
	initial.region = DeviceRegion::Cn;
	initial.region_switch_failure_generation = 2;
	RuntimeStateStore store(initial);
	const uint64_t initial_revision =
		store.current_publication().revision;

	CHECK_FALSE(
		store.commit(RegionSwitchPatch { .failure_generation = 2 }));
	CHECK_FALSE(
		store.commit(RegionSwitchPatch { .failure_generation = 1 }));
	CHECK(store.current_status().region_switch_failure_generation == 2);
	CHECK(store.current_publication().revision == initial_revision);

	REQUIRE(store.commit(RegionSwitchPatch { .failure_generation = 3 }));
	CHECK(store.current_status().region_switch_failure_generation == 3);
	CHECK(store.current_publication().revision == initial_revision + 1);

	REQUIRE(store.commit(RegionSwitchPatch { .failure_generation = 4 }));
	CHECK(store.current_status().region_switch_failure_generation == 4);
	CHECK(store.current_publication().revision == initial_revision + 2);
}

TEST_CASE("other patches preserve the region switch state")
{
	RuntimeStatus initial;
	initial.region = DeviceRegion::Cn;
	initial.region_switch_failure_generation = 5;
	RuntimeStateStore store(initial);

	RuntimeStatus observed;
	observed.region = DeviceRegion::Global;
	observed.region_switch_failure_generation = 99;
	observed.app_version = "app";
	REQUIRE(store.commit(RefreshObservationPatch { .status = observed }));

	RuntimeFlagsPatch flags {};
	flags.status.region = DeviceRegion::Global;
	flags.status.region_switch_failure_generation = 99;
	flags.status.auto_sleep_enabled = true;
	REQUIRE(store.commit(flags));

	DisplayMetricsPatch metrics {};
	metrics.status.region = DeviceRegion::Global;
	metrics.status.region_switch_failure_generation = 99;
	metrics.status.resolution = "1280x720";
	REQUIRE(store.commit(metrics));

	const RuntimeStatus current = store.current_status();
	CHECK(current.region == DeviceRegion::Cn);
	CHECK(current.region_switch_failure_generation == 5);
	CHECK(current.app_version == "app");
	CHECK(current.auto_sleep_enabled);
	CHECK(current.resolution == "1280x720");
}

TEST_CASE("unobserved refresh domains retain last good HTTP state")
{
	RuntimeStateStore store;
	RuntimeStatus observed;
	observed.hid_supported = true;
	observed.hid_enabled = true;
	observed.hid_mode = "normal";
	observed.server_available = true;
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = observed,
		.observed_domains = RefreshObservationDomain::Hid |
			RefreshObservationDomain::ServerAvailability,
	}));

	REQUIRE(store.commit(RefreshObservationPatch {
		.status = RuntimeStatus {},
		.observed_domains = RefreshObservationDomain::None,
	}));
	RuntimeStatus current = store.current_status();
	CHECK(current.hid_supported);
	CHECK(current.hid_enabled);
	CHECK(current.hid_mode == "normal");
	CHECK(current.server_available);

	RuntimeStatus explicit_negative;
	explicit_negative.hid_supported = true;
	explicit_negative.hid_enabled = false;
	explicit_negative.hid_mode = "relative";
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = explicit_negative,
		.observed_domains = RefreshObservationDomain::Hid,
	}));
	current = store.current_status();
	CHECK(current.hid_supported);
	CHECK_FALSE(current.hid_enabled);
	CHECK(current.hid_mode == "relative");
}

TEST_CASE("first unobserved refresh stays at the startup default")
{
	RuntimeStateStore store;
	RuntimeStatus failed;
	failed.hid_enabled = true;
	failed.server_available = true;
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = failed,
		.observed_domains = RefreshObservationDomain::None,
	}));

	const RuntimeStatus current = store.current_status();
	CHECK_FALSE(current.hid_enabled);
	CHECK_FALSE(current.server_available);
}

TEST_CASE("legacy refresh projection remains selectable through all domains")
{
	RuntimeStatus initial;
	initial.hid_enabled = true;
	initial.hid_mode = "normal";
	initial.server_available = true;
	RuntimeStateStore store(initial);

	REQUIRE(store.commit(RefreshObservationPatch {
		.status = RuntimeStatus {},
		.observed_domains = RefreshObservationDomain::All,
	}));
	const RuntimeStatus current = store.current_status();
	CHECK_FALSE(current.hid_enabled);
	CHECK(current.hid_mode.empty());
	CHECK_FALSE(current.server_available);
}

TEST_CASE("observed server unavailability replaces the prior available state")
{
	RuntimeStatus initial;
	initial.server_available = true;
	RuntimeStateStore store(initial);

	RuntimeStatus unavailable;
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = unavailable,
		.observed_domains = RefreshObservationDomain::ServerAvailability,
	}));

	CHECK_FALSE(store.current_status().server_available);
}

TEST_CASE("More page sysinfo domains merge independently and retain unobserved values")
{
	RuntimeStatus initial;
	initial.storage_text = "1.0/2.0 GB";
	initial.wifi_signal = "-99 dBm";
	RuntimeStateStore store(initial);

	RuntimeStatus observed;
	observed.device_key = "new-key";
	observed.hostname = "new-host";
	observed.device_number = "AX620QE_LP4_NANOAGENT_512M";
	observed.image_version = "0.1.6";
	observed.edid_name = "nanokvm-go-3840x2160-30";
	observed.mac_address = "aa:bb:cc:dd:ee:ff";
	observed.mdns_name = "new-host.local";
	observed.temperature_text = "49.4\u00B0C";
	observed.uptime_text = "16:03:25";
	observed.local_time_text = "2026-08-27 10:28:45";
	observed.memory_text = "66/246 MiB";
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = observed,
		.observed_domains = RefreshObservationDomain::SystemInfoStatic |
					   RefreshObservationDomain::SystemInfoActive,
	}));
	const RuntimeStatus after_static_active = store.current_status();
	CHECK(after_static_active.device_key == "new-key");
	CHECK(after_static_active.hostname == "new-host");
	CHECK(after_static_active.device_number ==
	      "AX620QE_LP4_NANOAGENT_512M");
	CHECK(after_static_active.image_version == "0.1.6");
	CHECK(after_static_active.edid_name == "nanokvm-go-3840x2160-30");
	CHECK(after_static_active.mac_address == "aa:bb:cc:dd:ee:ff");
	CHECK(after_static_active.mdns_name == "new-host.local");
	CHECK(after_static_active.temperature_text == "49.4\u00B0C");
	CHECK(after_static_active.uptime_text == "16:03:25");
	CHECK(after_static_active.local_time_text ==
	      "2026-08-27 10:28:45");
	CHECK(after_static_active.memory_text == "66/246 MiB");
	CHECK(after_static_active.storage_text == "1.0/2.0 GB");
	CHECK(after_static_active.wifi_signal == "-99 dBm");

	RuntimeStatus storage_observed;
	storage_observed.storage_text = "24.5/29.0 GB";
	storage_observed.device_key = "must-not-overwrite";
	storage_observed.temperature_text = "must-not-overwrite";
	storage_observed.wifi_signal = "must-not-overwrite";
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = storage_observed,
		.observed_domains =
			RefreshObservationDomain::SystemInfoStorage,
	}));
	const RuntimeStatus after_storage = store.current_status();
	CHECK(after_storage.storage_text == "24.5/29.0 GB");
	CHECK(after_storage.device_key == "new-key");
	CHECK(after_storage.temperature_text == "49.4\u00B0C");
	CHECK(after_storage.wifi_signal == "-99 dBm");

	RuntimeStatus signal_observed;
	signal_observed.wifi_signal.clear();
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = signal_observed,
		.observed_domains =
			RefreshObservationDomain::SystemInfoSignal,
	}));
	const RuntimeStatus after_signal = store.current_status();
	CHECK(after_signal.wifi_signal.empty());
	CHECK(after_signal.storage_text == "24.5/29.0 GB");
	CHECK(after_signal.device_key == "new-key");

	RuntimeStatus empty_observed;
	empty_observed.device_key = "erased";
	empty_observed.storage_text = "erased";
	REQUIRE(store.commit(RefreshObservationPatch {
		.status = empty_observed,
		.observed_domains = RefreshObservationDomain::None,
	}));
	const RuntimeStatus after_none = store.current_status();
	CHECK(after_none.device_key == "new-key");
	CHECK(after_none.storage_text == "24.5/29.0 GB");
	CHECK(after_none.wifi_signal.empty());
}

TEST_CASE("independent HTTP ingress patches preserve each other")
{
	RuntimeStateStore store;
	TestRendezvous start { 3 };

	std::thread display([&] {
		start.arrive_and_wait();
		(void)store.commit(DisplayIngressPatch {
			.resolution = "1280x720",
			.fps_unit = "Hz",
		});
	});
	std::thread hid([&] {
		start.arrive_and_wait();
		(void)store.commit(HidIngressPatch {
			.mouse_active = true,
			.keyboard_special = "ctrl",
		});
	});

	start.arrive_and_wait();
	display.join();
	hid.join();

	const RuntimeStatus current = store.current_status();
	CHECK(current.display_resolution_label == "1280x720");
	CHECK(current.display_fps_unit_label == "Hz");
	CHECK(current.hid_mouse_active);
	CHECK(current.hid_keyboard_special_label == "ctrl");
	CHECK(store.current_publication().revision == 2);
}

TEST_CASE("startup snapshot replacement retains publication ordering")
{
	RuntimeStateStore store;
	REQUIRE(store.commit(DisplayIngressPatch {
		.resolution = "1280x720",
	}));
	const uint64_t published_revision =
		store.current_publication().revision;

	RuntimeStatus startup = store.current_status();
	startup.app_version = "test";
	store.replace_initial_status(std::move(startup));

	CHECK(store.current_publication().revision == published_revision);
	REQUIRE(store.commit(HidIngressPatch {
		.mouse_active = true,
	}));
	CHECK(store.current_publication().revision == published_revision + 1);
}

TEST_CASE("superseded Wi-Fi terminal patch cannot complete newer request")
{
	RuntimeStateStore store;
	RuntimeStatus confirmed;
	confirmed.wifi_runtime_state = WifiRuntimeState::Connected;
	confirmed.wifi_connected = true;
	confirmed.wifi_saved_network_config = true;
	confirmed.wifi_status_text = "Connected";
	REQUIRE(store.commit(WifiLifecyclePatch { .status = confirmed }));

	RuntimeStatus enabling = confirmed;
	enabling.wifi_busy = true;
	enabling.wifi_status_text = "connecting...";

	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = enabling,
		.operation_generation = 1,
		.phase = WifiLifecyclePatch::Phase::Pending,
	}));

	RuntimeStatus disabling = confirmed;
	disabling.wifi_busy = true;
	disabling.wifi_status_text = "Disconnecting...";
	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = disabling,
		.operation_generation = 2,
		.phase = WifiLifecyclePatch::Phase::Pending,
	}));

	RuntimeStatus stale_terminal;
	stale_terminal.wifi_runtime_state = WifiRuntimeState::Connected;
	stale_terminal.wifi_connected = true;
	stale_terminal.wifi_busy = false;
	CHECK_FALSE(store.commit(WifiLifecyclePatch {
		.status = stale_terminal,
		.operation_generation = 1,
		.phase = WifiLifecyclePatch::Phase::Terminal,
	}));

	const RuntimeStatus after_stale = store.current_status();
	CHECK(after_stale.wifi_runtime_state == WifiRuntimeState::Connected);
	CHECK(after_stale.wifi_connected);
	CHECK(after_stale.wifi_saved_network_config);
	CHECK(after_stale.wifi_busy);

	RuntimeStatus terminal;
	terminal.wifi_runtime_state = WifiRuntimeState::Disconnected;
	terminal.wifi_busy = false;
	terminal.wifi_status_text = "Disconnected";
	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = terminal,
		.operation_generation = 2,
		.phase = WifiLifecyclePatch::Phase::Terminal,
	}));
	CHECK_FALSE(store.current_status().wifi_busy);
	CHECK_FALSE(store.current_status().wifi_saved_network_config);
	CHECK(store.current_publication().revision == 4);
}

TEST_CASE("stale refresh observation cannot overwrite a completed Wi-Fi action")
{
	RuntimeStateStore store;
	RuntimeStatus observed;
	observed.eth_ip = "192.168.1.2";
	observed.wifi_ip = "192.168.1.3";
	observed.wifi_connected = true;
	observed.wifi_runtime_state = WifiRuntimeState::Connected;
	observed.wifi_busy = true;
	observed.wifi_status_text = "connecting...";

	RuntimeStatus pending = observed;
	pending.wifi_busy = true;
	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = pending,
		.operation_generation = 1,
		.phase = WifiLifecyclePatch::Phase::Pending,
	}));

	RuntimeStatus terminal;
	terminal.wifi_runtime_state = WifiRuntimeState::Disconnected;
	terminal.wifi_busy = false;
	terminal.wifi_status_text = "Disconnected";
	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = terminal,
		.operation_generation = 1,
		.phase = WifiLifecyclePatch::Phase::Terminal,
	}));

	REQUIRE(store.commit(RefreshObservationPatch {
		.status = observed,
		.wifi_operation_generation = 0,
	}));

	const RuntimeStatus current = store.current_status();
	CHECK(current.eth_ip == "192.168.1.2");
	CHECK(current.wifi_runtime_state == WifiRuntimeState::Disconnected);
	CHECK_FALSE(current.wifi_connected);
	CHECK_FALSE(current.wifi_busy);
	CHECK(current.wifi_status_text == "Disconnected");
}

TEST_CASE("stale AP stop completion only advances its monotonic QR generation")
{
	RuntimeStateStore store;
	RuntimeStatus pending;
	pending.wifi_runtime_state = WifiRuntimeState::Connected;
	pending.wifi_connected = true;
	pending.wifi_busy = true;
	pending.wifi_status_text = "Disconnecting...";
	REQUIRE(store.commit(WifiLifecyclePatch {
		.status = pending,
		.operation_generation = 5,
		.phase = WifiLifecyclePatch::Phase::Pending,
	}));

	RuntimeStatus stale_completion;
	stale_completion.wifi_runtime_state = WifiRuntimeState::Disconnected;
	stale_completion.wifi_busy = false;
	stale_completion.wifi_ap_stop_completed_generation = 9;
	const auto publication = store.commit(WifiLifecyclePatch {
		.status = stale_completion,
		.operation_generation = 4,
		.phase = WifiLifecyclePatch::Phase::Terminal,
		.preserve_ap_stop_completion_when_stale = true,
	});
	REQUIRE(publication.has_value());

	CHECK(publication->status.wifi_runtime_state ==
	      WifiRuntimeState::Connected);
	CHECK(publication->status.wifi_connected);
	CHECK(publication->status.wifi_busy);
	CHECK(publication->status.wifi_ap_stop_completed_generation == 9);
}
