#include <atomic>
#include <chrono>
#include <filesystem>
#include <future>
#include <optional>
#include <stdexcept>
#include <string>
#include <string_view>
#include <utility>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/http_api_contract.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#include "nanokvm_runtime_internal.hpp"

using namespace lvgl_main;

namespace
{

struct LastStatusRecorder {
	std::optional<RuntimeStatus> status;
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	bool background_callback_called { false };
#endif

	RuntimeCallbacks callbacks()
	{
		RuntimeCallbacks callbacks;
		callbacks.on_status =
			[this](const RuntimeStatusPublication &publication) {
				status = publication.status;
			};
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
		callbacks.on_background_assets = [this](ui::BackgroundAssets) {
			background_callback_called = true;
		};
#endif
		return callbacks;
	}
};

NanoKvmRuntimeService make_region_switch_runtime(
	AppConfig config, RuntimeCallbacks callbacks,
	NanoKvmRuntimeService::RegionSwitchRunner region_switch_runner,
	std::filesystem::path config_path)
{
	return NanoKvmRuntimeService(
		std::move(config), std::move(callbacks),
		std::move(config_path),
		[]() {
			return std::optional<std::string> { "test" };
		},
		[](std::string_view) {
			return true;
		},
		production_ncm_boot_marker_paths(),
		std::filesystem::path(kDeviceRegionPath),
		std::move(region_switch_runner));
}

}

TEST_CASE("runtime publishes HTTP display label overrides")
{
	LastStatusRecorder recorder;
	NanoKvmRuntimeService runtime(default_app_config(),
				      recorder.callbacks());

	http_api::DisplayLabelsUpdate update;
	update.resolution = "1920x1080";
	update.fps = "59.9";
	update.fps_unit = "Hz";
	update.keyboard_special = "ctrl";
	update.keyboard_normal = "a";

	runtime.update_display_labels(std::move(update));

	REQUIRE(recorder.status.has_value());
	const RuntimeStatus current = runtime.current_status();
	CHECK(recorder.status->display_resolution_label == "1920x1080");
	CHECK(recorder.status->display_fps_label == "59.9");
	CHECK(recorder.status->display_fps_unit_label == "Hz");
	CHECK(recorder.status->hid_keyboard_special_label == "ctrl");
	CHECK(recorder.status->hid_keyboard_normal_label == "a");
	CHECK(current.display_resolution_label == "1920x1080");
	CHECK(current.display_fps_label == "59.9");
	CHECK(current.display_fps_unit_label == "Hz");
	CHECK(current.hid_keyboard_special_label == "ctrl");
	CHECK(current.hid_keyboard_normal_label == "a");
}

TEST_CASE("runtime publishes HTTP HID mouse generations and labels")
{
	LastStatusRecorder recorder;
	NanoKvmRuntimeService runtime(default_app_config(),
				      recorder.callbacks());

	http_api::HidStateUpdate active;
	active.mouse_active = true;
	active.keyboard_special = "shift";
	active.keyboard_normal = "B";
	runtime.update_hid_state(std::move(active));

	REQUIRE(recorder.status.has_value());
	CHECK(recorder.status->hid_mouse_active_generation == 1);
	CHECK(recorder.status->hid_mouse_clear_generation == 0);
	CHECK(recorder.status->hid_keyboard_special_label == "shift");
	CHECK(recorder.status->hid_keyboard_normal_label == "B");

	http_api::HidStateUpdate clear;
	clear.mouse_active = false;
	runtime.update_hid_state(std::move(clear));

	REQUIRE(recorder.status.has_value());
	const RuntimeStatus current = runtime.current_status();
	CHECK(recorder.status->hid_mouse_active_generation == 1);
	CHECK(recorder.status->hid_mouse_clear_generation == 1);
	CHECK(current.hid_mouse_active_generation == 1);
	CHECK(current.hid_mouse_clear_generation == 1);
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
TEST_CASE("runtime background action rejects invalid image names")
{
	LastStatusRecorder recorder;
	NanoKvmRuntimeService runtime(default_app_config(),
				      recorder.callbacks());

	const BackgroundActionResult result =
		runtime.apply_background_selection("../dial.bmp");

	CHECK_FALSE(result.ok);
	CHECK(result.http_status == 400);
	CHECK_FALSE(result.detail.empty());
	CHECK_FALSE(recorder.background_callback_called);
}
#endif

TEST_CASE("runtime queues the fixed factory reset script once")
{
	std::promise<std::string> requested_path_promise;
	auto requested_path = requested_path_promise.get_future();
	std::atomic<int> request_count { 0 };

	NanoKvmRuntimeService runtime(
		default_app_config(), {},
		std::filesystem::path { "/tmp/kvm-ui-factory-reset-test.toml" },
		[]() {
			return std::optional<std::string> { "test" };
		},
		[&](std::string_view script_path) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			requested_path_promise.set_value(
				std::string(script_path));
			return true;
		});

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.request_factory_reset());
	CHECK_FALSE(runtime.request_factory_reset());
	REQUIRE(requested_path.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(requested_path.get() == "/kvmcomm/scripts/reset_to_default.sh");
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 1);
}

TEST_CASE("runtime rejects factory reset when high-risk actions are disabled")
{
	AppConfig config = default_app_config();
	config.nanokvm.enable_high_risk_actions = false;
	std::atomic<int> request_count { 0 };

	NanoKvmRuntimeService runtime(
		std::move(config), {},
		std::filesystem::path {
			"/tmp/kvm-ui-factory-reset-disabled-test.toml" },
		[]() {
			return std::optional<std::string> { "test" };
		},
		[&](std::string_view) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			return true;
		});

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK_FALSE(runtime.request_factory_reset());
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 0);
}

TEST_CASE("runtime publishes factory reset failure and allows retry")
{
	std::promise<uint64_t> failure_generation_promise;
	auto failure_generation = failure_generation_promise.get_future();
	std::atomic<bool> failure_reported { false };

	RuntimeCallbacks callbacks;
	callbacks.on_status = [&](const RuntimeStatusPublication &publication) {
		const RuntimeStatus &status = publication.status;
		if (status.factory_reset_failure_generation == 0 ||
		    failure_reported.exchange(true,
					      std::memory_order_acq_rel)) {
			return;
		}
		failure_generation_promise.set_value(
			status.factory_reset_failure_generation);
	};

	NanoKvmRuntimeService runtime(
		default_app_config(), std::move(callbacks),
		std::filesystem::path {
			"/tmp/kvm-ui-factory-reset-failure-test.toml" },
		[]() {
			return std::optional<std::string> { "test" };
		},
		[](std::string_view) {
			return false;
		});

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.request_factory_reset());
	REQUIRE(failure_generation.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(failure_generation.get() == 1);
	CHECK(runtime.current_status().factory_reset_failure_generation == 1);
	CHECK(runtime.request_factory_reset());
	runtime.stop();
}

TEST_CASE("runtime queues the region switch request once")
{
	std::promise<DeviceRegion> requested_target_promise;
	auto requested_target = requested_target_promise.get_future();
	std::atomic<int> request_count { 0 };

	auto runtime = make_region_switch_runtime(
		default_app_config(), {},
		[&](DeviceRegion target) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			requested_target_promise.set_value(target);
			return true;
		},
		"/tmp/kvm-ui-region-switch-test.toml");

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.request_region_switch(DeviceRegion::Global));
	CHECK_FALSE(runtime.request_region_switch(DeviceRegion::Cn));
	REQUIRE(requested_target.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(requested_target.get() == DeviceRegion::Global);
	CHECK_FALSE(runtime.request_region_switch(DeviceRegion::Cn));
	CHECK(runtime.current_status().region_switch_failure_generation == 0);
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 1);
}

TEST_CASE("runtime publishes each region switch failure and accepts a retry")
{
	std::promise<void> first_failure_promise;
	auto first_failure = first_failure_promise.get_future();
	std::promise<void> second_failure_promise;
	auto second_failure = second_failure_promise.get_future();
	std::atomic<uint64_t> observed_generation { 0 };
	std::atomic<int> request_count { 0 };

	RuntimeCallbacks callbacks;
	callbacks.on_status = [&](const RuntimeStatusPublication &publication) {
		const uint64_t generation =
			publication.status.region_switch_failure_generation;
		if (generation <=
		    observed_generation.load(std::memory_order_acquire)) {
			return;
		}
		observed_generation.store(generation, std::memory_order_release);
		if (generation == 1) {
			first_failure_promise.set_value();
		} else if (generation == 2) {
			second_failure_promise.set_value();
		}
	};

	auto runtime = make_region_switch_runtime(
		default_app_config(), std::move(callbacks),
		[&](DeviceRegion) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			return false;
		},
		"/tmp/kvm-ui-region-switch-failure-test.toml");

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.request_region_switch(DeviceRegion::Global));
	REQUIRE(first_failure.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(runtime.current_status().region_switch_failure_generation == 1);
	CHECK(runtime.request_region_switch(DeviceRegion::Cn));
	REQUIRE(second_failure.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(runtime.current_status().region_switch_failure_generation == 2);
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 2);
}

TEST_CASE("runtime publishes a region switch failure when the runner throws")
{
	std::promise<void> first_failure_promise;
	auto first_failure = first_failure_promise.get_future();
	std::promise<void> second_failure_promise;
	auto second_failure = second_failure_promise.get_future();
	std::atomic<uint64_t> observed_generation { 0 };
	std::atomic<int> request_count { 0 };

	RuntimeCallbacks callbacks;
	callbacks.on_status = [&](const RuntimeStatusPublication &publication) {
		const uint64_t generation =
			publication.status.region_switch_failure_generation;
		if (generation <=
		    observed_generation.load(std::memory_order_acquire)) {
			return;
		}
		observed_generation.store(generation, std::memory_order_release);
		if (generation == 1) {
			first_failure_promise.set_value();
		} else if (generation == 2) {
			second_failure_promise.set_value();
		}
	};

	auto runtime = make_region_switch_runtime(
		default_app_config(), std::move(callbacks),
		[&](DeviceRegion) -> bool {
			request_count.fetch_add(1, std::memory_order_relaxed);
			throw std::runtime_error(
				"injected region switch failure");
		},
		"/tmp/kvm-ui-region-switch-throw-test.toml");

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.request_region_switch(DeviceRegion::Global));
	REQUIRE(first_failure.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(runtime.current_status().region_switch_failure_generation == 1);
	CHECK(runtime.is_running());
	CHECK(runtime.request_region_switch(DeviceRegion::Cn));
	REQUIRE(second_failure.wait_for(std::chrono::seconds { 1 }) ==
		std::future_status::ready);
	CHECK(runtime.current_status().region_switch_failure_generation == 2);
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 2);
}

TEST_CASE("runtime rejects the region switch when high-risk actions are disabled")
{
	AppConfig config = default_app_config();
	config.nanokvm.enable_high_risk_actions = false;
	std::atomic<int> request_count { 0 };

	auto runtime = make_region_switch_runtime(
		std::move(config), {},
		[&](DeviceRegion) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			return true;
		},
		"/tmp/kvm-ui-region-switch-disabled-test.toml");

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK_FALSE(runtime.request_region_switch(DeviceRegion::Global));
	runtime.stop();
	CHECK(request_count.load(std::memory_order_relaxed) == 0);
}

TEST_CASE("runtime rejects the region switch before the service starts")
{
	std::atomic<int> request_count { 0 };

	auto runtime = make_region_switch_runtime(
		default_app_config(), {},
		[&](DeviceRegion) {
			request_count.fetch_add(1, std::memory_order_relaxed);
			return true;
		},
		"/tmp/kvm-ui-region-switch-not-running-test.toml");

	CHECK_FALSE(runtime.request_region_switch(DeviceRegion::Cn));
	CHECK(request_count.load(std::memory_order_relaxed) == 0);
}

TEST_CASE("region switch response accepts only a zero integer code")
{
	CHECK(detail::region_switch_response_ok(
		nlohmann::json { { "code", 0 }, { "msg", "success" } }));
	CHECK_FALSE(detail::region_switch_response_ok(nlohmann::json {
		{ "code", -1 }, { "msg", "set region failed" } }));
	CHECK_FALSE(detail::region_switch_response_ok(
		nlohmann::json { { "message", "ok" } }));
	CHECK_FALSE(
		detail::region_switch_response_ok(nlohmann::json { { "code", "0" } }));
	CHECK_FALSE(
		detail::region_switch_response_ok(nlohmann::json { { "code", 0.5 } }));
	CHECK_FALSE(detail::region_switch_response_ok(
		nlohmann::json { { "code", true } }));
	CHECK_FALSE(detail::region_switch_response_ok(nlohmann::json::array()));
	CHECK_FALSE(detail::region_switch_response_ok(nlohmann::json()));
}
