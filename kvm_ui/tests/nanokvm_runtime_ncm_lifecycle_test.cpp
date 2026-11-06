#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "lvgl_main/settings_cli.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

class TempDir final
{
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
			("nanokvm_go_ncm_lifecycle_" +
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

	const std::filesystem::path &path() const { return _path; }

    private:
	std::filesystem::path _path;
};

class NcmFixture final
{
    public:
	explicit NcmFixture(bool script_succeeds = true)
		: _script_succeeds(script_succeeds)
		, _script(_temp.path() / "usbdev.sh")
		, _command_log(_temp.path() / "commands.log")
		, _ncm_marker(_temp.path() / "usb.ncm")
		, _rndis_marker(_temp.path() / "usb.rndis")
		, _config_path(_temp.path() / "kvm_ui.toml")
	{
		write_script();
	}

	AppConfig config() const
	{
		AppConfig config = default_app_config();
		config.nanokvm.usb_gadget_restart_script = _script.string();
		config.nanokvm.enable_high_risk_actions = true;
		return config;
	}

	std::filesystem::path config_path() const { return _config_path; }

	NcmBootMarkerPaths marker_paths() const
	{
		return NcmBootMarkerPaths { _ncm_marker.string(),
					    _rndis_marker.string() };
	}

	void write_ncm_marker()
	{
		std::ofstream f(_ncm_marker, std::ios::trunc);
		REQUIRE(f.good());
	}

	void write_rndis_marker()
	{
		std::ofstream f(_rndis_marker, std::ios::trunc);
		REQUIRE(f.good());
	}

	void remove_ncm_marker()
	{
		std::error_code ec;
		std::filesystem::remove(_ncm_marker, ec);
	}

	void remove_rndis_marker()
	{
		std::error_code ec;
		std::filesystem::remove(_rndis_marker, ec);
	}

	bool ncm_marker_exists() const
	{
		std::error_code ec;
		return std::filesystem::exists(_ncm_marker, ec);
	}

	bool rndis_marker_exists() const
	{
		std::error_code ec;
		return std::filesystem::exists(_rndis_marker, ec);
	}

	std::size_t restart_count() const
	{
		std::ifstream input(_command_log);
		std::vector<std::string> commands;
		std::string line;
		while (std::getline(input, line)) {
			if (!line.empty()) {
				commands.push_back(line);
			}
		}
		return static_cast<std::size_t>(
			std::count(commands.begin(), commands.end(),
				   "restart"));
	}

	bool wait_for_restart_count(std::size_t expected,
				   std::chrono::milliseconds timeout) const
	{
		const auto deadline =
			std::chrono::steady_clock::now() + timeout;
		while (std::chrono::steady_clock::now() < deadline) {
			if (restart_count() >= expected) {
				return true;
			}
			std::this_thread::sleep_for(10ms);
		}
		return restart_count() >= expected;
	}

    private:
	void write_script()
	{
		std::ofstream script(_script, std::ios::trunc);
		REQUIRE(script.good());
		script << "#!/bin/sh\n"
			  "echo \"$1\" >> \""
		       << _command_log.string() << "\"\n";
		if (_script_succeeds) {
			script << "exit 0\n";
		} else {
			script << "exit 1\n";
		}
		REQUIRE(script.good());
		std::error_code ec;
		std::filesystem::permissions(
			_script, std::filesystem::perms::owner_exec,
			std::filesystem::perm_options::add, ec);
		REQUIRE_FALSE(ec);
	}

	TempDir _temp;
	bool _script_succeeds;
	std::filesystem::path _script;
	std::filesystem::path _command_log;
	std::filesystem::path _ncm_marker;
	std::filesystem::path _rndis_marker;
	std::filesystem::path _config_path;
};

class StatusRecorder final
{
    public:
	RuntimeCallbacks callbacks()
	{
		RuntimeCallbacks callbacks;
		callbacks.on_status =
			[this](const RuntimeStatusPublication &publication) {
				const RuntimeStatus &status = publication.status;
				std::lock_guard<std::mutex> lock(_mutex);
				_statuses.push_back(status);
				_condition.notify_all();
			};
		return callbacks;
	}

	bool wait_for_ncm(bool enabled, bool supported,
			 std::chrono::milliseconds timeout)
	{
		return wait_for(
			[enabled, supported](const RuntimeStatus &status) {
				return status.ncm_enabled == enabled &&
				       status.ncm_supported == supported;
			},
			timeout);
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

NanoKvmRuntimeService make_runtime(NcmFixture &fixture,
				   StatusRecorder &recorder)
{
	return NanoKvmRuntimeService(fixture.config(), recorder.callbacks(),
				    fixture.config_path(),
				    []() {
					    return std::optional<std::string> {
						    "test"
					    };
				    },
				    {}, fixture.marker_paths());
}

} // namespace

TEST_CASE("NCM startup reports disabled and does not mutate USB when no marker exists")
{
	NcmFixture fixture;
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());

	REQUIRE(recorder.wait_for_ncm(false, true, 1s));
	CHECK(fixture.restart_count() == 0);
	CHECK_FALSE(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM startup reports enabled from NCM marker and does not mutate USB")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());

	REQUIRE(recorder.wait_for_ncm(true, true, 1s));
	CHECK(fixture.restart_count() == 0);
	CHECK(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM startup reports enabled from legacy RNDIS marker")
{
	NcmFixture fixture;
	fixture.write_rndis_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());

	REQUIRE(recorder.wait_for_ncm(true, true, 1s));
	CHECK(fixture.restart_count() == 0);
	CHECK_FALSE(fixture.ncm_marker_exists());
	CHECK(fixture.rndis_marker_exists());
}

TEST_CASE("NCM explicit enable creates the NCM marker and restarts USB")
{
	NcmFixture fixture;
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(false, true, 1s));

	runtime.set_ncm_enabled(true);
	REQUIRE(recorder.wait_for_ncm(true, true, 2s));

	CHECK(fixture.restart_count() == 1);
	CHECK(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM explicit disable removes both markers and restarts USB")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	runtime.set_ncm_enabled(false);
	REQUIRE(recorder.wait_for_ncm(false, true, 2s));

	CHECK(fixture.restart_count() == 1);
	CHECK_FALSE(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM settings CLI enable mirrors the UI action")
{
	NcmFixture fixture;
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(false, true, 1s));

	std::string error;
	REQUIRE(runtime.settings_cli_set_ncm_enabled(true, error));
	REQUIRE(recorder.wait_for_ncm(true, true, 2s));

	CHECK(error.empty());
	CHECK(fixture.restart_count() == 1);
	CHECK(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM settings CLI disable mirrors the UI action")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	std::string error;
	REQUIRE(runtime.settings_cli_set_ncm_enabled(false, error));
	REQUIRE(recorder.wait_for_ncm(false, true, 2s));

	CHECK(error.empty());
	CHECK(fixture.restart_count() == 1);
	CHECK_FALSE(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM local disable after external Web disable stays disabled without a restart")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	fixture.remove_ncm_marker();
	runtime.set_ncm_enabled(false);
	REQUIRE(recorder.wait_for_ncm(false, true, 2s));

	CHECK(fixture.restart_count() == 0);
	CHECK_FALSE(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM local enable after external Web disable re-enables and restarts USB")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	fixture.remove_ncm_marker();
	runtime.set_ncm_enabled(true);
	REQUIRE(fixture.wait_for_restart_count(1, 2s));
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	CHECK(fixture.restart_count() == 1);
	CHECK(fixture.ncm_marker_exists());
	CHECK_FALSE(fixture.rndis_marker_exists());
}

TEST_CASE("NCM periodic refresh adopts an external Web disable")
{
	NcmFixture fixture;
	fixture.write_ncm_marker();
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(true, true, 1s));

	fixture.remove_ncm_marker();
	REQUIRE(recorder.wait_for_ncm(false, true, 4s));

	CHECK(fixture.restart_count() == 0);
	CHECK_FALSE(fixture.ncm_marker_exists());
}

TEST_CASE("NCM settings CLI enable reports failure without confirming the target")
{
	NcmFixture fixture(false);
	StatusRecorder recorder;
	auto runtime = make_runtime(fixture, recorder);
	REQUIRE(runtime.start());
	REQUIRE(recorder.wait_for_ncm(false, true, 1s));

	std::string error;
	REQUIRE_FALSE(runtime.settings_cli_set_ncm_enabled(true, error));
	CHECK_FALSE(error.empty());
	REQUIRE(fixture.wait_for_restart_count(1, 2s));
	CHECK(fixture.restart_count() == 1);
}

#if !NANOKVM_GO_RELEASE_BUILD
TEST_CASE("settings CLI status reports NCM from live boot markers")
{
	NcmFixture fixture;
	SettingsCliCommand command;
	command.action = SettingsCliAction::Status;
	command.has_action = true;
	AppConfig config = fixture.config();

	const auto disabled = execute_settings_cli_command(
		config, fixture.config_path(), command, fixture.marker_paths());
	REQUIRE(disabled.ok);
	const auto disabled_field = std::find_if(
		disabled.fields.begin(), disabled.fields.end(),
		[](const auto &field) { return field.first == "ncm_enabled"; });
	REQUIRE(disabled_field != disabled.fields.end());
	CHECK(disabled_field->second == "false");
	CHECK(std::find_if(disabled.fields.begin(), disabled.fields.end(),
			  [](const auto &field) {
				  return field.first == "ncm_state";
			  }) == disabled.fields.end());

	fixture.write_ncm_marker();
	const auto enabled = execute_settings_cli_command(
		config, fixture.config_path(), command, fixture.marker_paths());
	REQUIRE(enabled.ok);
	const auto enabled_field = std::find_if(
		enabled.fields.begin(), enabled.fields.end(),
		[](const auto &field) { return field.first == "ncm_enabled"; });
	REQUIRE(enabled_field != enabled.fields.end());
	CHECK(enabled_field->second == "true");
}

TEST_CASE("settings CLI status exposes inconclusive NCM observation as unknown")
{
	NcmFixture fixture;
	std::error_code ec;
	std::filesystem::create_directories(fixture.marker_paths().ncm_flag,
					   ec);
	SettingsCliCommand command;
	command.action = SettingsCliAction::Status;
	command.has_action = true;
	AppConfig config = fixture.config();

	const auto result = execute_settings_cli_command(
		config, fixture.config_path(), command, fixture.marker_paths());
	REQUIRE(result.ok);
	CHECK(std::find_if(result.fields.begin(), result.fields.end(),
			  [](const auto &field) {
				  return field.first == "ncm_enabled";
			  }) == result.fields.end());
	const auto state_field = std::find_if(
		result.fields.begin(), result.fields.end(),
		[](const auto &field) { return field.first == "ncm_state"; });
	REQUIRE(state_field != result.fields.end());
	CHECK(state_field->second == "unknown");
	const auto error_field = std::find_if(
		result.fields.begin(), result.fields.end(),
		[](const auto &field) { return field.first == "ncm_error"; });
	REQUIRE(error_field != result.fields.end());
	CHECK_FALSE(error_field->second.empty());
}
#endif
