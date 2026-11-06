#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/wifi_runtime_controller.hpp"

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

struct LifecyclePublication {
	WifiRuntimeProjection projection;
	WifiLifecyclePatch::Phase phase {
		WifiLifecyclePatch::Phase::Observation
	};
	uint64_t operation_generation { 0 };
	bool preserve_ap_stop_completion_when_stale { false };
};

class PublicationRecorder final {
    public:
	void publish(const WifiRuntimeProjection &projection,
		     WifiLifecyclePatch::Phase phase,
		     uint64_t operation_generation,
		     bool preserve_ap_stop_completion_when_stale)
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_publications.push_back(LifecyclePublication {
			.projection = projection,
			.phase = phase,
			.operation_generation = operation_generation,
			.preserve_ap_stop_completion_when_stale =
				preserve_ap_stop_completion_when_stale,
		});
		_condition.notify_all();
	}

	bool wait_for_terminal(uint64_t generation,
			       std::chrono::milliseconds timeout)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		return _condition.wait_for(lock, timeout, [this, generation]() {
			return std::any_of(
				_publications.begin(), _publications.end(),
				[generation](const auto &publication) {
					return publication.phase ==
						       WifiLifecyclePatch::
							       Phase::Terminal &&
					       publication.operation_generation ==
						       generation;
				});
		});
	}

	bool wait_for_pending(std::chrono::milliseconds timeout)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		return _condition.wait_for(lock, timeout, [this]() {
			return std::any_of(
				_publications.begin(), _publications.end(),
				[](const auto &publication) {
					return publication.phase ==
					       WifiLifecyclePatch::Phase::Pending;
				});
		});
	}

	bool has_terminal(uint64_t generation) const
	{
		std::lock_guard<std::mutex> lock(_mutex);
		return std::any_of(
			_publications.begin(), _publications.end(),
			[generation](const auto &publication) {
				return publication.phase ==
					       WifiLifecyclePatch::Phase::Terminal &&
				       publication.operation_generation == generation;
			});
	}

	std::vector<LifecyclePublication> publications() const
	{
		std::lock_guard<std::mutex> lock(_mutex);
		return _publications;
	}

    private:
	mutable std::mutex _mutex;
	std::condition_variable _condition;
	std::vector<LifecyclePublication> _publications;
};

class FakeWifiScript final {
    public:
	WifiScriptExecutionResult run(const std::vector<std::string> &args)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		_calls.push_back(args);
		if (!args.empty() && args.front() == "ap_off" && _hold_ap_stop) {
			_ap_stop_started = true;
			_condition.notify_all();
			_condition.wait(lock, [this]() {
				return !_hold_ap_stop;
			});
		}
		if (!args.empty() && args.front() == "on" && _hold_enable) {
			_enable_started = true;
			_condition.notify_all();
			_condition.wait(lock, [this]() {
				return !_hold_enable;
			});
		}
		if (!args.empty() && args.front() == "scan") {
			return { .exit_code = 0, .output = "[]" };
		}
		if (!args.empty() && args.front() == "on") {
			return _enable_result;
		}
		return { .exit_code = 0 };
	}

	void hold_enable()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_hold_enable = true;
	}

	void hold_ap_stop()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_hold_ap_stop = true;
	}

	bool wait_for_enable_start(std::chrono::milliseconds timeout)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		return _condition.wait_for(lock, timeout, [this]() {
			return _enable_started;
		});
	}

	bool wait_for_ap_stop_start(std::chrono::milliseconds timeout)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		return _condition.wait_for(lock, timeout, [this]() {
			return _ap_stop_started;
		});
	}

	void release_enable()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_hold_enable = false;
		_condition.notify_all();
	}

	void release_ap_stop()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_hold_ap_stop = false;
		_condition.notify_all();
	}

	void set_enable_result(WifiScriptExecutionResult result)
	{
		std::lock_guard<std::mutex> lock(_mutex);
		_enable_result = std::move(result);
	}

	std::vector<std::vector<std::string>> calls() const
	{
		std::lock_guard<std::mutex> lock(_mutex);
		return _calls;
	}

    private:
	mutable std::mutex _mutex;
	std::condition_variable _condition;
	std::vector<std::vector<std::string>> _calls;
	bool _hold_ap_stop { false };
	bool _ap_stop_started { false };
	bool _hold_enable { false };
	bool _enable_started { false };
	WifiScriptExecutionResult _enable_result { .exit_code = 0 };
};

WifiRuntimeController::Dependencies make_dependencies(
	FakeWifiScript &script, PublicationRecorder &recorder,
	WifiRuntimeController::SerialWorkEnqueue enqueue_serial_work = {},
	std::function<bool()> probe_ap_client = {},
	std::function<WifiSavedNetworkAvailability()> has_saved_network_config = {})
{
	if (!enqueue_serial_work) {
		enqueue_serial_work = [](std::string_view,
					 std::function<void()> work) {
			work();
			return true;
		};
	}
	if (!probe_ap_client) {
		probe_ap_client = []() {
			return false;
		};
	}
	if (!has_saved_network_config) {
		has_saved_network_config = []() {
			return WifiSavedNetworkAvailability::Absent;
		};
	}
	return WifiRuntimeController::Dependencies {
		.run_script =
			[&script](const std::vector<std::string> &args) {
				return script.run(args);
			},
		.current_ap_credentials =
			[]() {
				return WifiRuntimeApCredentials {
					.ssid = "NanoKVM-Test",
					.password = "test-password",
					.qr_payload =
						"WIFI:T:WPA;S:NanoKVM-Test;P:test-password;;",
					.web_url = "http://172.24.1.1/",
				};
			},
		.prepare_ap_credentials =
			[]() {
				return WifiRuntimeApCredentials {
					.ssid = "NanoKVM-Test",
					.password = "test-password",
					.qr_payload =
						"WIFI:T:WPA;S:NanoKVM-Test;P:test-password;;",
					.web_url = "http://172.24.1.1/",
				};
			},
		.write_ap_runtime_files =
			[](std::string_view) {
			},
		.probe_ap_client = std::move(probe_ap_client),
		.has_saved_network_config = std::move(has_saved_network_config),
		.enqueue_serial_work = std::move(enqueue_serial_work),
		.request_refresh =
			[]() {
			},
		.publish_lifecycle =
			[&recorder](
				const WifiRuntimeProjection &projection,
				WifiLifecyclePatch::Phase phase,
				uint64_t operation_generation,
				bool preserve_ap_stop_completion_when_stale) {
				recorder.publish(
					projection, phase, operation_generation,
					preserve_ap_stop_completion_when_stale);
			},
	};
}

} // namespace

TEST_CASE("WiFi runtime controller publishes a typed scan lifecycle")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder),
					 false, 15s);

	controller.scan_networks();
	REQUIRE(recorder.wait_for_pending(1s));
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	const auto publications = recorder.publications();
	REQUIRE(publications.size() >= 2U);
	CHECK(publications.front().phase == WifiLifecyclePatch::Phase::Pending);
	CHECK(publications.front().operation_generation == 1);
	CHECK_FALSE(publications.back().projection.wifi_scan_running);
	CHECK(publications.back().projection.wifi_scan_result.ok);
	controller.close();
}

TEST_CASE("WiFi runtime controller checks saved config on first observation")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder, {},
							   {},
							   []() {
								   return WifiSavedNetworkAvailability::Present;
							   }),
					 false, 15s);

	const WifiRuntimeProjection projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });

	CHECK(projection.wifi_runtime_state ==
	      WifiRuntimeState::CacheConnecting);
	CHECK(projection.wifi_saved_network_config);
	CHECK(projection.wifi_busy);
	CHECK(projection.wifi_boot_reconnect_running);
	controller.close();
}

TEST_CASE("WiFi runtime controller projects unavailable saved config")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder),
						 false, 15s);

	const WifiRuntimeProjection projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });

	CHECK_FALSE(projection.wifi_saved_network_config);
	controller.close();
}

TEST_CASE("WiFi runtime controller keeps saved config visible while WiFi is off")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder, {},
								   {},
								   []() {
									   return WifiSavedNetworkAvailability::Present;
								   }),
						 false, 15s);

	const WifiRuntimeProjection projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = false });

	CHECK(projection.wifi_saved_network_config);
	CHECK(projection.wifi_runtime_state == WifiRuntimeState::Disconnected);
	CHECK_FALSE(projection.wifi_busy);
	CHECK_FALSE(projection.wifi_boot_reconnect_running);
	controller.close();
}

TEST_CASE("WiFi runtime controller ignores a superseded async completion")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder),
					 false, 15s);

	script.hold_enable();
	controller.set_enabled(true);
	REQUIRE(script.wait_for_enable_start(1s));
	controller.set_enabled(false);
	REQUIRE(recorder.wait_for_terminal(2, 1s));
	script.release_enable();
	std::this_thread::sleep_for(100ms);

	const auto publications = recorder.publications();
	std::size_t old_terminal_count = 0;
	for (const auto &publication : publications) {
		if (publication.phase == WifiLifecyclePatch::Phase::Terminal &&
		    publication.operation_generation == 1) {
			++old_terminal_count;
		}
	}
	CHECK(old_terminal_count == 0U);
	controller.close();
}

TEST_CASE("WiFi runtime controller restores a confirmed WiFi connection after QR exit")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
		.wifi_ssid = "Saved WiFi",
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	CHECK(controller.stop_qr_ap_config() == stop_generation);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	queued_work.back()();
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto calls = script.calls();
	REQUIRE(calls.size() == 4U);
	CHECK(calls[0] == std::vector<std::string> { "ap_off" });
	CHECK(calls[1][0] == "ap");
	CHECK(calls[2] == std::vector<std::string> { "ap_off" });
	CHECK(calls[3] == std::vector<std::string> { "on" });

	const auto publications = recorder.publications();
	const auto terminal = std::find_if(
		publications.begin(), publications.end(),
		[stop_operation_generation](const auto &publication) {
			return publication.phase == WifiLifecyclePatch::Phase::Terminal &&
			       publication.operation_generation == stop_operation_generation;
		});
	REQUIRE(terminal != publications.end());
	CHECK(terminal->projection.wifi_ap_stop_completed_generation ==
	      stop_generation);
	CHECK(terminal->projection.wifi_runtime_state ==
	      WifiRuntimeState::CacheConnecting);
	CHECK(terminal->projection.wifi_busy);
	CHECK(terminal->projection.wifi_status_text == "connecting...");
	controller.close();
}

TEST_CASE("WiFi runtime controller does not reconnect after QR exit without a confirmed connection")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	queued_work.back()();
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto calls = script.calls();
	REQUIRE(calls.size() == 3U);
	CHECK(calls.back() == std::vector<std::string> { "ap_off" });
	CHECK(std::none_of(calls.begin(), calls.end(), [](const auto &args) {
		return !args.empty() && args.front() == "on";
	}));
	controller.close();
}

TEST_CASE("WiFi runtime controller withholds QR completion until reconnect finishes")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.hold_enable();
	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	std::thread stop_worker(std::move(queued_work.back()));
	const bool enable_started = script.wait_for_enable_start(1s);
	if (enable_started) {
		CHECK_FALSE(recorder.has_terminal(stop_operation_generation));
	}

	script.release_enable();
	stop_worker.join();
	REQUIRE(enable_started);
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));
	const auto publications = recorder.publications();
	const auto terminal = std::find_if(
		publications.begin(), publications.end(),
		[stop_operation_generation](const auto &publication) {
			return publication.phase == WifiLifecyclePatch::Phase::Terminal &&
			       publication.operation_generation == stop_operation_generation;
		});
	REQUIRE(terminal != publications.end());
	CHECK(terminal->projection.wifi_ap_stop_completed_generation ==
	      stop_generation);
	controller.close();
}

TEST_CASE("WiFi runtime controller does not reconnect after close begins during QR AP stop")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
							 std::function<void()> work) {
						  queued_work.push_back(std::move(work));
						  return true;
					  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.hold_ap_stop();
	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	std::thread stop_worker(std::move(queued_work.back()));
	const bool ap_stop_started = script.wait_for_ap_stop_start(1s);
	if (ap_stop_started) {
		controller.close();
	}
	script.release_ap_stop();
	stop_worker.join();
	REQUIRE(ap_stop_started);
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto calls = script.calls();
	CHECK(calls.size() == 3U);
	CHECK(std::none_of(calls.begin(), calls.end(), [](const auto &args) {
		return !args.empty() && args.front() == "on";
	}));
}

TEST_CASE("WiFi runtime controller clears a stale QR AP stop before the next QR exit")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
							 std::function<void()> work) {
						  queued_work.push_back(std::move(work));
						  return true;
					  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.hold_ap_stop();
	const uint64_t first_stop_generation = controller.stop_qr_ap_config();
	REQUIRE(first_stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	std::thread first_stop_worker(std::move(queued_work.back()));
	const bool ap_stop_started = script.wait_for_ap_stop_start(1s);
	if (ap_stop_started) {
		controller.start_qr_ap_config();
	}
	script.release_ap_stop();
	first_stop_worker.join();
	REQUIRE(ap_stop_started);
	REQUIRE(queued_work.size() == 3U);

	const uint64_t second_stop_generation = controller.stop_qr_ap_config();
	CHECK(second_stop_generation > first_stop_generation);
	REQUIRE(queued_work.size() == 4U);
	const uint64_t second_stop_operation_generation =
		controller.operation_generation();
	queued_work[2]();
	queued_work[3]();
	REQUIRE(recorder.wait_for_terminal(second_stop_operation_generation, 1s));

	const auto calls = script.calls();
	CHECK(std::none_of(calls.begin(), calls.end(), [](const auto &args) {
		return !args.empty() && args.front() == "on";
	}));
	controller.close();
}

TEST_CASE("WiFi runtime controller preserves QR success over a late AP stop")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
							 std::function<void()> work) {
						  queued_work.push_back(std::move(work));
						  return true;
					  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.hold_ap_stop();
	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	std::thread stop_worker(std::move(queued_work.back()));
	const bool ap_stop_started = script.wait_for_ap_stop_start(1s);
	if (ap_stop_started) {
		const auto success_projection = controller.project_observation(
			WifiRuntimeObservation {
				.wifi_supported = true,
				.wifi_connected = true,
				.station_ip = true,
				.wifi_ssid = "Configured WiFi",
			});
		CHECK(success_projection.wifi_runtime_state ==
		      WifiRuntimeState::Connected);
		controller.stop_qr_ap_config_after_success();
	}
	script.release_ap_stop();
	stop_worker.join();
	REQUIRE(ap_stop_started);
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto publications = recorder.publications();
	const auto terminal = std::find_if(
		publications.begin(), publications.end(),
		[stop_operation_generation](const auto &publication) {
			return publication.phase == WifiLifecyclePatch::Phase::Terminal &&
			       publication.operation_generation == stop_operation_generation;
		});
	REQUIRE(terminal != publications.end());
	CHECK(terminal->projection.wifi_runtime_state ==
	      WifiRuntimeState::Connected);
	CHECK(terminal->projection.wifi_ap_stop_completed_generation ==
	      stop_generation);

	const auto calls = script.calls();
	CHECK(std::none_of(calls.begin(), calls.end(), [](const auto &args) {
		return !args.empty() && args.front() == "on";
	}));
	controller.close();
}

TEST_CASE("WiFi runtime controller reports reconnect script failure after QR exit")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.set_enable_result({ .exit_code = 1, .output = "ERROR: RECONNECT" });
	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	queued_work.back()();
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto publications = recorder.publications();
	const auto terminal = std::find_if(
		publications.begin(), publications.end(),
		[stop_operation_generation](const auto &publication) {
			return publication.phase == WifiLifecyclePatch::Phase::Terminal &&
			       publication.operation_generation == stop_operation_generation;
		});
	REQUIRE(terminal != publications.end());
	CHECK(terminal->projection.wifi_ap_stop_completed_generation ==
	      stop_generation);
	CHECK(terminal->projection.wifi_runtime_state == WifiRuntimeState::Error);
	CHECK_FALSE(terminal->projection.wifi_busy);
	CHECK(terminal->projection.wifi_status_text == "Reconnect failed");
	CHECK(terminal->projection.wifi_error == "Reconnect failed");
	controller.close();
}

TEST_CASE("WiFi runtime controller reports reconnect timeout after QR exit")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.project_observation(WifiRuntimeObservation {
		.wifi_supported = true,
		.wifi_connected = true,
		.station_ip = true,
	});
	controller.start_qr_ap_config();
	REQUIRE(queued_work.size() == 1U);
	queued_work.front()();
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	script.set_enable_result({ .exit_code = 0, .timed_out = true });
	const uint64_t stop_generation = controller.stop_qr_ap_config();
	REQUIRE(stop_generation != 0);
	REQUIRE(queued_work.size() == 2U);
	const uint64_t stop_operation_generation = controller.operation_generation();
	queued_work.back()();
	REQUIRE(recorder.wait_for_terminal(stop_operation_generation, 1s));

	const auto publications = recorder.publications();
	const auto terminal = std::find_if(
		publications.begin(), publications.end(),
		[stop_operation_generation](const auto &publication) {
			return publication.phase == WifiLifecyclePatch::Phase::Terminal &&
			       publication.operation_generation == stop_operation_generation;
		});
	REQUIRE(terminal != publications.end());
	CHECK(terminal->projection.wifi_runtime_state == WifiRuntimeState::Error);
	CHECK_FALSE(terminal->projection.wifi_busy);
	CHECK(terminal->projection.wifi_status_text == "Reconnect failed");
	CHECK(terminal->projection.wifi_error == "Reconnect timed out");
	controller.close();
}

TEST_CASE("WiFi runtime controller drops queued AP work after close")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::vector<std::function<void()>> queued_work;
	WifiRuntimeController controller(
		make_dependencies(script, recorder,
				  [&queued_work](std::string_view,
						 std::function<void()> work) {
					  queued_work.push_back(
						  std::move(work));
					  return true;
				  }),
		false, 15s);

	controller.start_qr_ap_config();
	REQUIRE(recorder.wait_for_pending(1s));
	REQUIRE(queued_work.size() == 1U);
	controller.close();
	queued_work.front()();

	CHECK(script.calls().empty());
	CHECK(recorder.publications().size() == 1U);
}

TEST_CASE("WiFi runtime controller joins its AP client scanner on close")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	std::atomic<uint32_t> probe_calls { 0 };
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {},
				  [&probe_calls]() {
					  ++probe_calls;
					  return false;
				  }),
		false, 15s);

	controller.start_qr_ap_config();
	REQUIRE(recorder.wait_for_terminal(1, 1s));
	const auto deadline = std::chrono::steady_clock::now() + 1s;
	while (probe_calls.load() == 0 &&
	       std::chrono::steady_clock::now() < deadline) {
		std::this_thread::sleep_for(10ms);
	}
	REQUIRE(probe_calls.load() > 0);

	controller.close();
	const uint32_t calls_after_close = probe_calls.load();
	std::this_thread::sleep_for(150ms);
	CHECK(probe_calls.load() == calls_after_close);
}

TEST_CASE("WiFi runtime controller preserves SSID boundary whitespace")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(make_dependencies(script, recorder),
						 false, 15s);

	controller.connect_network("  Lab WiFi \t", "password");
	REQUIRE(recorder.wait_for_terminal(1, 1s));

	const auto calls = script.calls();
	const auto connect = std::find_if(calls.begin(), calls.end(),
					  [](const auto &args) {
						  return !args.empty() && args.front() == "connect";
					  });
	REQUIRE(connect != calls.end());
	REQUIRE(connect->size() == 3U);
	CHECK((*connect)[1] == "  Lab WiFi \t");
	CHECK((*connect)[2] == "password");
	controller.close();
}

TEST_CASE(
	"WiFi runtime controller preserves Present through Unknown probe results")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	int probe_calls = 0;
	auto probe = [&probe_calls]() {
		++probe_calls;
		if (probe_calls == 1) {
			return WifiSavedNetworkAvailability::Present;
		}
		return WifiSavedNetworkAvailability::Unknown;
	};
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {}, {}, std::move(probe)),
		false, 15s);

	auto projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });
	CHECK(projection.wifi_saved_network_config);

	projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = false });
	CHECK(projection.wifi_saved_network_config);
	controller.close();
}

TEST_CASE(
	"WiFi runtime controller keeps Absent through Unknown probe results")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	int probe_calls = 0;
	auto probe = [&probe_calls]() {
		++probe_calls;
		if (probe_calls == 1) {
			return WifiSavedNetworkAvailability::Absent;
		}
		return WifiSavedNetworkAvailability::Unknown;
	};
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {}, {}, std::move(probe)),
		false, 15s);

	auto projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });
	CHECK_FALSE(projection.wifi_saved_network_config);

	projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });
	CHECK_FALSE(projection.wifi_saved_network_config);
	controller.close();
}

TEST_CASE(
	"WiFi runtime controller starts with safe disabled and recovers with Present")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	int probe_calls = 0;
	auto probe = [&probe_calls]() {
		++probe_calls;
		if (probe_calls == 1) {
			return WifiSavedNetworkAvailability::Unknown;
		}
		return WifiSavedNetworkAvailability::Present;
	};
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {}, {}, std::move(probe)),
		false, 15s);

	auto projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });
	CHECK_FALSE(projection.wifi_saved_network_config);

	projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = true });
	CHECK(projection.wifi_saved_network_config);
	controller.close();
}

TEST_CASE(
	"WiFi runtime controller saved config Present with unsupported interface stays usable")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {}, {},
				  []() {
					  return WifiSavedNetworkAvailability::
						  Present;
				  }),
		false, 15s);

	const auto projection = controller.project_observation(
		WifiRuntimeObservation { .wifi_supported = false });

	CHECK(projection.wifi_saved_network_config);
	CHECK(projection.wifi_runtime_state == WifiRuntimeState::Disconnected);
	CHECK_FALSE(projection.wifi_boot_reconnect_running);
	controller.close();
}

TEST_CASE(
	"WiFi runtime controller boot marker does not set saved config on init")
{
	FakeWifiScript script;
	PublicationRecorder recorder;
	WifiRuntimeController controller(
		make_dependencies(script, recorder, {}, {},
				  []() {
					  return WifiSavedNetworkAvailability::
						  Absent;
				  }),
		true, 15s);

	const auto projection = controller.current_projection();

	CHECK_FALSE(projection.wifi_saved_network_config);
	CHECK(projection.wifi_boot_reconnect_running);
	controller.close();
}
