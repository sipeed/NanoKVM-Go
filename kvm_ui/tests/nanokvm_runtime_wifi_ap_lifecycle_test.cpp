#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <future>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		const uint64_t id =
			next_id.fetch_add(1, std::memory_order_relaxed);
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_wifi_ap_lifecycle_" +
			 std::to_string(stamp) + "_" + std::to_string(id));
		std::filesystem::create_directories(_path);
		std::filesystem::permissions(_path,
					     std::filesystem::perms::owner_all);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const
	{
		return _path;
	}

    private:
	std::filesystem::path _path;
};

enum class StopOutcome : uint8_t {
	Success,
	Error,
	Timeout,
};

enum class OnOutcome : uint8_t {
	NoSavedNetwork,
	Error,
};

enum class ApStartOutcome : uint8_t {
	Success,
	Error,
};

enum class BootEnabledOutcome : uint8_t {
	Enabled,
	Disabled,
	Malformed,
	Error,
	Timeout,
};

class WifiScriptFixture final {
    public:
	explicit WifiScriptFixture(
		StopOutcome stop_outcome,
		OnOutcome on_outcome = OnOutcome::NoSavedNetwork,
		ApStartOutcome ap_start_outcome = ApStartOutcome::Success,
		BootEnabledOutcome boot_enabled_outcome =
			BootEnabledOutcome::Disabled)
		: _stop_outcome(stop_outcome)
		, _on_outcome(on_outcome)
		, _ap_start_outcome(ap_start_outcome)
		, _boot_enabled_outcome(boot_enabled_outcome)
		, _script(_temp.path() / "wifi.sh")
		, _command_log(_temp.path() / "commands.log")
		, _first_ap_off_gate(_temp.path() / "hold_first_ap_off")
		, _scan_gate(_temp.path() / "hold_scan")
	{
		write_script();
	}

	AppConfig config() const
	{
		AppConfig config = default_app_config();
		config.nanokvm.wifi_script = _script.string();
		return config;
	}

	std::filesystem::path config_path() const
	{
		return _temp.path() / "kvm_ui.toml";
	}

	void hold_first_ap_off()
	{
		std::ofstream gate(_first_ap_off_gate, std::ios::trunc);
		REQUIRE(gate.good());
	}

	void release_first_ap_off()
	{
		std::error_code ec;
		std::filesystem::remove(_first_ap_off_gate, ec);
		REQUIRE_FALSE(ec);
	}

	void hold_scan()
	{
		std::ofstream gate(_scan_gate, std::ios::trunc);
		REQUIRE(gate.good());
	}

	void release_scan()
	{
		std::error_code ec;
		std::filesystem::remove(_scan_gate, ec);
		REQUIRE_FALSE(ec);
	}

	bool wait_for_ap_off_count(std::size_t expected,
				   std::chrono::milliseconds timeout) const
	{
		const auto deadline =
			std::chrono::steady_clock::now() + timeout;
		while (std::chrono::steady_clock::now() < deadline) {
			if (ap_off_count() >= expected) {
				return true;
			}
			std::this_thread::sleep_for(10ms);
		}
		return ap_off_count() >= expected;
	}

	bool wait_for_scan_count(std::size_t expected,
				 std::chrono::milliseconds timeout) const
	{
		const auto deadline =
			std::chrono::steady_clock::now() + timeout;
		while (std::chrono::steady_clock::now() < deadline) {
			if (scan_count() >= expected) {
				return true;
			}
			std::this_thread::sleep_for(10ms);
		}
		return scan_count() >= expected;
	}

	std::vector<std::string> commands() const
	{
		auto commands = raw_commands();
		commands.erase(std::remove(commands.begin(), commands.end(),
					   "boot_enabled"),
			       commands.end());
		return commands;
	}

	std::size_t boot_enabled_count() const
	{
		const auto commands_snapshot = raw_commands();
		return static_cast<std::size_t>(
			std::count(commands_snapshot.begin(), commands_snapshot.end(),
				   "boot_enabled"));
	}

	std::size_t ap_off_count() const
	{
		const auto commands_snapshot = commands();
		return static_cast<std::size_t>(
			std::count(commands_snapshot.begin(),
				   commands_snapshot.end(), "ap_off"));
	}

	std::size_t scan_count() const
	{
		const auto commands_snapshot = commands();
		return static_cast<std::size_t>(
			std::count(commands_snapshot.begin(),
				   commands_snapshot.end(), "scan"));
	}

    private:
	std::string stop_behavior() const
	{
		switch (_stop_outcome) {
		case StopOutcome::Success:
			return "\t\t:\n";
		case StopOutcome::Error:
			return "\t\tprintf '%s\\n' 'ERROR: AP_OFF_FAILED'\n"
			       "\t\texit 23\n";
		case StopOutcome::Timeout:
			return "\t\tsleep 6\n";
		}
		return {};
	}

	std::string on_behavior() const
	{
		switch (_on_outcome) {
		case OnOutcome::NoSavedNetwork:
			return "\tprintf '%s\\n' 'ERROR: NO_SAVED_NETWORK'\n"
			       "\texit 0\n";
		case OnOutcome::Error:
			return "\tprintf '%s\\n' 'ERROR: ON_FAILED'\n"
			       "\texit 24\n";
		}
		return {};
	}

	std::string ap_start_behavior() const
	{
		switch (_ap_start_outcome) {
		case ApStartOutcome::Success:
			return "\t:\n";
		case ApStartOutcome::Error:
			return "\tprintf '%s\\n' 'ERROR: AP_START_FAILED'\n"
			       "\texit 25\n";
		}
		return {};
	}

	std::string boot_enabled_behavior() const
	{
		switch (_boot_enabled_outcome) {
		case BootEnabledOutcome::Enabled:
			return "\tprintf '%s\\n' 'true'\n\texit 0\n";
		case BootEnabledOutcome::Disabled:
			return "\tprintf '%s\\n' 'false'\n\texit 0\n";
		case BootEnabledOutcome::Malformed:
			return "\tprintf '%s\\n' 'not-a-boolean'\n\texit 0\n";
		case BootEnabledOutcome::Error:
			return "\tprintf '%s\\n' 'false'\n\texit 31\n";
		case BootEnabledOutcome::Timeout:
			return "\tsleep 6\n";
		}
		return {};
	}

	void write_script()
	{
		std::ofstream script(_script, std::ios::trunc);
		REQUIRE(script.good());
		script << "#!/bin/sh\n"
		       << "dir=$(dirname \"$0\")\n"
		       << "printf '%s\\n' \"$1\" >> \"$dir/commands.log\"\n"
		       << "if [ \"$1\" = \"boot_enabled\" ]; then\n"
		       << boot_enabled_behavior()
		       << "fi\n"
		       << "if [ \"$1\" = \"on\" ]; then\n"
		       << on_behavior() << "fi\n"
		       << "if [ \"$1\" = \"ap_off\" ]; then\n"
		       << "\tcount=0\n"
		       << "\tif [ -f \"$dir/ap_off_count\" ]; then\n"
		       << "\t\tcount=$(cat \"$dir/ap_off_count\")\n"
		       << "\tfi\n"
		       << "\tcount=$((count + 1))\n"
		       << "\tprintf '%s\\n' \"$count\" > \"$dir/ap_off_count\"\n"
		       << "\tif [ \"$count\" -eq 1 ] && "
			  "[ -f \"$dir/hold_first_ap_off\" ]; then\n"
		       << "\t\twhile [ -f \"$dir/hold_first_ap_off\" ]; do\n"
		       << "\t\t\tsleep 0.01\n"
		       << "\t\tdone\n"
		       << "\tfi\n"
		       << "\tif [ \"$count\" -gt 1 ]; then\n"
		       << stop_behavior() << "\tfi\n"
		       << "fi\n"
		       << "if [ \"$1\" = \"ap\" ]; then\n"
		       << ap_start_behavior() << "fi\n"
		       << "if [ \"$1\" = \"scan\" ]; then\n"
		       << "\twhile [ -f \"$dir/hold_scan\" ]; do\n"
		       << "\t\tsleep 0.01\n"
		       << "\tdone\n"
		       << "fi\n"
		       << "exit 0\n";
		REQUIRE(script.good());

		std::error_code ec;
		std::filesystem::permissions(_script,
					     std::filesystem::perms::owner_exec,
					     std::filesystem::perm_options::add,
					     ec);
		REQUIRE_FALSE(ec);
	}

	std::vector<std::string> raw_commands() const
	{
		std::ifstream input(_command_log);
		std::vector<std::string> commands;
		std::string line;
		while (std::getline(input, line)) {
			if (!line.empty()) {
				commands.push_back(line);
			}
		}
		return commands;
	}

	TempDir _temp;
	StopOutcome _stop_outcome;
	OnOutcome _on_outcome;
	ApStartOutcome _ap_start_outcome;
	BootEnabledOutcome _boot_enabled_outcome;
	std::filesystem::path _script;
	std::filesystem::path _command_log;
	std::filesystem::path _first_ap_off_gate;
	std::filesystem::path _scan_gate;
};

class StatusRecorder final {
    public:
	RuntimeCallbacks callbacks()
	{
		RuntimeCallbacks callbacks;
		callbacks.on_status =
			[this](const RuntimeStatusPublication &publication) {
				const RuntimeStatus &status =
					publication.status;
				std::lock_guard<std::mutex> lock(_mutex);
				_statuses.push_back(status);
				_condition.notify_all();
			};
		return callbacks;
	}

	bool wait_for_wifi_state(WifiRuntimeState state,
				 std::chrono::milliseconds timeout)
	{
		return wait_for(
			[state](const RuntimeStatus &status) {
				return status.wifi_runtime_state == state;
			},
			timeout);
	}

	bool wait_for_completed_generation(uint64_t generation,
					   std::chrono::milliseconds timeout)
	{
		return wait_for(
			[generation](const RuntimeStatus &status) {
				return status.wifi_ap_stop_completed_generation >=
				       generation;
			},
			timeout);
	}

	std::size_t callback_count() const
	{
		std::lock_guard<std::mutex> lock(_mutex);
		return _statuses.size();
	}

	std::vector<RuntimeStatus> statuses() const
	{
		std::lock_guard<std::mutex> lock(_mutex);
		return _statuses;
	}

    private:
	template <typename Predicate>
	bool wait_for(Predicate predicate, std::chrono::milliseconds timeout)
	{
		std::unique_lock<std::mutex> lock(_mutex);
		return _condition.wait_for(lock, timeout, [this, &predicate] {
			return std::any_of(_statuses.begin(), _statuses.end(),
					   predicate);
		});
	}

	mutable std::mutex _mutex;
	std::condition_variable _condition;
	std::vector<RuntimeStatus> _statuses;
};

NanoKvmRuntimeService make_runtime(WifiScriptFixture &script,
				   StatusRecorder &recorder)
{
	return NanoKvmRuntimeService(
		script.config(), recorder.callbacks(), script.config_path(),
		[]() {
			return std::optional<std::string> { "test" };
		});
}

void check_completed_generations_are_monotonic(const StatusRecorder &recorder)
{
	uint64_t previous = 0;
	for (const RuntimeStatus &status : recorder.statuses()) {
		CHECK(status.wifi_ap_stop_completed_generation >= previous);
		previous = status.wifi_ap_stop_completed_generation;
	}
}

void start_ui_owned_ap(NanoKvmRuntimeService &runtime,
		       const WifiScriptFixture &script,
		       StatusRecorder &recorder)
{
	runtime.notify_wifi_page_visible(true);
	runtime.start_wifi_qr_ap_config();
	REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::ApAwaitClient,
					     2s));
	CHECK_FALSE(runtime.current_status().wifi_busy);
	CHECK(script.commands() == std::vector<std::string> { "ap_off", "ap" });
}

TEST_CASE("runtime obtains boot reconnect projection from wifi.sh",
	  "[runtime][wifi]")
{
	WifiScriptFixture script(StopOutcome::Success, OnOutcome::NoSavedNetwork,
					 ApStartOutcome::Success, BootEnabledOutcome::Enabled);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);

	CHECK(script.boot_enabled_count() == 1U);
	const RuntimeStatus status = runtime.current_status();
	CHECK(status.wifi_boot_reconnect_running);
	CHECK(status.wifi_busy);
	CHECK(status.wifi_runtime_state == WifiRuntimeState::CacheConnecting);
	runtime.stop();
}

TEST_CASE("runtime fails closed when boot-enabled query is unusable",
	  "[runtime][wifi]")
{
	for (const auto outcome : { BootEnabledOutcome::Malformed,
					   BootEnabledOutcome::Error,
					   BootEnabledOutcome::Timeout }) {
		WifiScriptFixture script(StopOutcome::Success,
						 OnOutcome::NoSavedNetwork,
						 ApStartOutcome::Success, outcome);
		StatusRecorder recorder;
		auto runtime = make_runtime(script, recorder);

		CHECK(script.boot_enabled_count() == 1U);
		const RuntimeStatus status = runtime.current_status();
		CHECK_FALSE(status.wifi_boot_reconnect_running);
		CHECK_FALSE(status.wifi_busy);
		runtime.stop();
	}
}

void run_stop_completion_case(StopOutcome outcome)
{
	WifiScriptFixture script(outcome);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);
	REQUIRE(runtime.start_settings_cli_actions());

	runtime.notify_wifi_page_visible(true);
	script.hold_first_ap_off();
	runtime.start_wifi_qr_ap_config();
	REQUIRE(script.wait_for_ap_off_count(1, 1s));

	const auto request_started_at = std::chrono::steady_clock::now();
	const uint64_t generation = runtime.stop_wifi_qr_ap_config();
	const auto request_duration =
		std::chrono::steady_clock::now() - request_started_at;
	REQUIRE(generation != 0);
	CHECK(request_duration < 1s);
	CHECK(runtime.stop_wifi_qr_ap_config() == generation);
	runtime.notify_wifi_page_visible(false);

	script.release_first_ap_off();
	const auto completion_timeout = outcome == StopOutcome::Timeout ? 8s :
									  2s;
	REQUIRE(recorder.wait_for_completed_generation(generation,
						       completion_timeout));
	CHECK(runtime.current_status().wifi_ap_stop_completed_generation ==
	      generation);
	CHECK(script.commands() ==
	      std::vector<std::string> { "ap_off", "ap", "ap_off" });
	CHECK(script.ap_off_count() == 2);

	runtime.notify_wifi_page_visible(true);
	runtime.notify_wifi_page_visible(false);
	std::this_thread::sleep_for(100ms);
	CHECK(script.ap_off_count() == 2);
	check_completed_generations_are_monotonic(recorder);
	runtime.stop();
}

} // namespace

TEST_CASE("runtime does not stop an AP when setup is only ready",
	  "[runtime][wifi][ap]")
{
	WifiScriptFixture script(StopOutcome::Success);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);
	REQUIRE(runtime.start_settings_cli_actions());

	runtime.notify_wifi_page_visible(true);
	runtime.set_wifi_enabled(true);
	REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::ApAwaitClient,
					     2s));
	runtime.notify_wifi_qr_dismissed();
	CHECK(runtime.current_status().wifi_runtime_state ==
	      WifiRuntimeState::ApAwaitClient);
	CHECK_FALSE(runtime.current_status().wifi_busy);
	runtime.notify_wifi_page_visible(false);
	std::this_thread::sleep_for(100ms);

	CHECK(script.commands() == std::vector<std::string> { "on" });
	CHECK(runtime.current_status().wifi_ap_stop_completed_generation == 1);
	runtime.stop();
}

TEST_CASE("runtime publishes terminal Wi-Fi command failures",
	  "[runtime][wifi]")
{
	SECTION("enable failure clears the pending state")
	{
		WifiScriptFixture script(StopOutcome::Success,
					 OnOutcome::Error);
		StatusRecorder recorder;
		auto runtime = make_runtime(script, recorder);
		REQUIRE(runtime.start_settings_cli_actions());

		runtime.set_wifi_enabled(true);
		REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::Error,
						     2s));

		const RuntimeStatus status = runtime.current_status();
		CHECK_FALSE(status.wifi_busy);
		CHECK(status.wifi_status_text == "Enable failed");
		CHECK(status.wifi_error == "Enable failed");
		CHECK(script.commands() == std::vector<std::string> { "on" });
		runtime.stop();
	}

	SECTION("AP start failure clears the pending state")
	{
		WifiScriptFixture script(StopOutcome::Success,
					 OnOutcome::NoSavedNetwork,
					 ApStartOutcome::Error);
		StatusRecorder recorder;
		auto runtime = make_runtime(script, recorder);
		REQUIRE(runtime.start_settings_cli_actions());

		runtime.start_wifi_qr_ap_config();
		REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::Error,
						     2s));

		const RuntimeStatus status = runtime.current_status();
		CHECK_FALSE(status.wifi_busy);
		CHECK(status.wifi_status_text == "AP failed");
		CHECK(status.wifi_error == "AP failed");
		CHECK(script.commands() ==
		      std::vector<std::string> { "ap_off", "ap", "ap_off" });
		runtime.stop();
	}
}

TEST_CASE("runtime never reports success after UI-owned AP cleanup fails",
	  "[runtime][wifi][ap]")
{
	SECTION("disable")
	{
		WifiScriptFixture script(StopOutcome::Error);
		StatusRecorder recorder;
		auto runtime = make_runtime(script, recorder);
		REQUIRE(runtime.start_settings_cli_actions());
		start_ui_owned_ap(runtime, script, recorder);

		runtime.set_wifi_enabled(false);
		REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::Error,
						     2s));

		const RuntimeStatus status = runtime.current_status();
		CHECK_FALSE(status.wifi_busy);
		CHECK(status.wifi_status_text == "AP stop failed");
		CHECK(status.wifi_error == "AP stop failed");
		CHECK(script.commands() ==
		      std::vector<std::string> { "ap_off", "ap", "ap_off",
						 "off" });
		runtime.stop();
	}

	SECTION("reset")
	{
		WifiScriptFixture script(StopOutcome::Error);
		StatusRecorder recorder;
		auto runtime = make_runtime(script, recorder);
		REQUIRE(runtime.start_settings_cli_actions());
		start_ui_owned_ap(runtime, script, recorder);

		runtime.reset_wifi();
		REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::Error,
						     2s));

		const RuntimeStatus status = runtime.current_status();
		CHECK_FALSE(status.wifi_busy);
		CHECK(status.wifi_status_text == "AP stop failed");
		CHECK(status.wifi_error == "AP stop failed");
		CHECK(script.commands() ==
		      std::vector<std::string> { "ap_off", "ap", "ap_off",
						 "clear", "off" });
		runtime.stop();
	}
}

TEST_CASE("runtime scan preserves the confirmed AP state while AP stop runs",
	  "[runtime][wifi][scan]")
{
	WifiScriptFixture script(StopOutcome::Success);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);
	REQUIRE(runtime.start_settings_cli_actions());

	runtime.notify_wifi_page_visible(true);
	runtime.set_wifi_enabled(true);
	REQUIRE(recorder.wait_for_wifi_state(WifiRuntimeState::ApAwaitClient,
					     2s));

	script.hold_scan();
	runtime.scan_wifi_networks();
	REQUIRE(script.wait_for_scan_count(1, 1s));

	const RuntimeStatus pending = runtime.current_status();
	CHECK(pending.wifi_runtime_state == WifiRuntimeState::ApAwaitClient);
	CHECK(pending.wifi_busy);
	CHECK(pending.wifi_scan_running);

	script.release_scan();
	runtime.stop();

	const RuntimeStatus completed = runtime.current_status();
	CHECK(completed.wifi_runtime_state == WifiRuntimeState::Disconnected);
	CHECK_FALSE(completed.wifi_busy);
	CHECK_FALSE(completed.wifi_scan_running);
	CHECK(script.commands() ==
	      std::vector<std::string> { "on", "ap_off", "scan" });
}

TEST_CASE("runtime serializes Wi-Fi AP stop completion and ownership cleanup",
	  "[runtime][wifi][ap]")
{
	SECTION("successful stop")
	{
		run_stop_completion_case(StopOutcome::Success);
	}

	SECTION("script error still completes the request")
	{
		run_stop_completion_case(StopOutcome::Error);
	}

	SECTION("timed out script still completes the request")
	{
		run_stop_completion_case(StopOutcome::Timeout);
	}
}

TEST_CASE("runtime cleans up an in-progress UI AP when the Wi-Fi flow exits",
	  "[runtime][wifi][ap]")
{
	WifiScriptFixture script(StopOutcome::Success);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);
	REQUIRE(runtime.start_settings_cli_actions());

	runtime.notify_wifi_page_visible(true);
	script.hold_first_ap_off();
	runtime.start_wifi_qr_ap_config();
	REQUIRE(script.wait_for_ap_off_count(1, 1s));

	const auto exit_started_at = std::chrono::steady_clock::now();
	runtime.notify_wifi_page_visible(false);
	const auto exit_duration =
		std::chrono::steady_clock::now() - exit_started_at;
	CHECK(exit_duration < 1s);
	runtime.notify_wifi_page_visible(false);

	script.release_first_ap_off();
	REQUIRE(recorder.wait_for_completed_generation(1, 2s));
	CHECK(script.commands() ==
	      std::vector<std::string> { "ap_off", "ap", "ap_off" });
	CHECK(script.ap_off_count() == 2);

	runtime.notify_wifi_page_visible(false);
	std::this_thread::sleep_for(100ms);
	CHECK(script.ap_off_count() == 2);
	runtime.stop();
}

TEST_CASE("runtime shutdown joins a blocked QR AP action",
	  "[runtime][wifi][ap]")
{
	WifiScriptFixture script(StopOutcome::Success);
	StatusRecorder recorder;
	auto runtime = make_runtime(script, recorder);
	REQUIRE(runtime.start_settings_cli_actions());

	script.hold_first_ap_off();
	runtime.start_wifi_qr_ap_config();
	REQUIRE(script.wait_for_ap_off_count(1, 1s));
	REQUIRE(runtime.stop_wifi_qr_ap_config() != 0);
	const std::size_t callbacks_before_shutdown = recorder.callback_count();

	auto shutdown = std::async(std::launch::async, [&runtime] {
		runtime.stop();
	});
	CHECK(shutdown.wait_for(100ms) == std::future_status::timeout);

	script.release_first_ap_off();
	REQUIRE(shutdown.wait_for(2s) == std::future_status::ready);
	shutdown.get();
	const std::size_t callbacks_after_shutdown = recorder.callback_count();
	std::this_thread::sleep_for(100ms);
	CHECK(callbacks_after_shutdown == callbacks_before_shutdown);
	CHECK(recorder.callback_count() == callbacks_after_shutdown);
}
