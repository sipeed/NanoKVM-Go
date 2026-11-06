#include <algorithm>
#include <chrono>
#include <condition_variable>
#include <filesystem>
#include <fstream>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "fixtures/fake_tailscale_server.hpp"
#include "lvgl_main/app_config.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "nanokvm_tailscale_client.hpp"

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

constexpr std::chrono::milliseconds kWaitBudget { 5000 };

class StatusRecorder final {
    public:
	RuntimeCallbacks callbacks()
	{
		RuntimeCallbacks callbacks;
		callbacks.on_status = [this](
					      const RuntimeStatusPublication
						      &publication) {
			std::lock_guard lock(_mutex);
			_statuses.push_back(publication.status);
			_condition.notify_all();
		};
		return callbacks;
	}

	bool wait_for(std::function<bool(const RuntimeStatus &)> predicate,
		      std::chrono::milliseconds timeout = kWaitBudget)
	{
		std::unique_lock lock(_mutex);
		return _condition.wait_for(lock, timeout, [this, &predicate] {
			return std::any_of(_statuses.begin(), _statuses.end(),
					   predicate);
		});
	}

	std::vector<RuntimeStatus> statuses() const
	{
		std::lock_guard lock(_mutex);
		return _statuses;
	}

    private:
	mutable std::mutex _mutex;
	std::condition_variable _condition;
	std::vector<RuntimeStatus> _statuses;
};

class TailscaleFixture final {
    private:
	tests::TempDirectory _temp;
	std::filesystem::path _region_path { _temp.path() / "region" };
	std::filesystem::path _config_path { _temp.path() / "kvm_ui.toml" };
	std::filesystem::path _socket_path { _temp.path() / "state.sock" };
	NcmBootMarkerPaths _markers { (_temp.path() / "usb.ncm").string(),
				      (_temp.path() / "usb.rndis").string() };
	bool _region_ok { false };

    public:
	explicit TailscaleFixture(std::string_view region_name = "cn")
	{
		std::ofstream region(_region_path, std::ios::trunc);
		region << region_name;
		_region_ok = region.good();
	}

	bool region_ok() const { return _region_ok; }

	void reply(std::string path, std::string body)
	{
		server.set_reply(std::move(path),
				  tests::FakeTailscaleServer::Reply {
					  .status = 200,
					  .body = std::move(body),
					  .delay = 0ms });
	}

	void reply_to_everything(std::string heartbeat_body)
	{
		reply(std::string(kTailscaleLoginCodePath), R"({"code":"123456"})");
		reply(std::string(kTailscaleLoginCodeHeartbeatPath),
		      std::move(heartbeat_body));
		reply(std::string(kTailscaleLoginCodeStartPath),
		      R"({"status":true,"message":""})");
		reply(std::string(kTailscaleLoginCodeExitPath),
		      R"({"status":true,"message":""})");
		reply(std::string(kTailscaleLoginCodeDisablePath),
		      R"({"code":"%off%"})");
	}

	std::unique_ptr<NanoKvmRuntimeService> make_runtime()
	{
		AppConfig config = default_app_config();
		config.nanokvm.enable_high_risk_actions = true;
		return std::make_unique<NanoKvmRuntimeService>(
			config, recorder.callbacks(), _config_path,
			[]() {
				return std::optional<std::string> { "test" };
			},
			NanoKvmRuntimeService::FactoryResetRunner {}, _markers,
			_region_path,
			NanoKvmRuntimeService::RegionSwitchRunner {},
			_socket_path.string());
	}

	bool wait_for_requests(
		std::function<bool()> predicate,
		std::chrono::milliseconds timeout = kWaitBudget)
	{
		const auto deadline = std::chrono::steady_clock::now() + timeout;
		while (std::chrono::steady_clock::now() < deadline) {
			if (predicate()) {
				return true;
			}
			std::this_thread::sleep_for(20ms);
		}
		return predicate();
	}

	tests::FakeTailscaleServer server { _socket_path };
	StatusRecorder recorder;
};

}

TEST_CASE("tailscale exit action runs over the state socket in global region")
{
	TailscaleFixture fixture("global");
	REQUIRE(fixture.region_ok());
	fixture.reply_to_everything(
		R"({"status":"online","start":true,"match":true,)"
		R"("enabled":true,"name":"MockKVM","ip":"100.64.0.1"})");
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_started && status.tailscale_matched;
	}));

	const auto start_requests =
		fixture.server.request_count(kTailscaleLoginCodeStartPath);
	REQUIRE(runtime->request_tailscale_exit());
	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_action_kind ==
			       TailscaleActionKind::Exit &&
		       status.tailscale_action_ok;
	}));

	CHECK(fixture.server.request_count(kTailscaleLoginCodeExitPath) == 1);
	CHECK(fixture.server.request_count(kTailscaleLoginCodeStartPath) ==
	      start_requests);
	const auto log = fixture.server.request_log();
	CHECK(std::find(log.begin(), log.end(),
			std::string(kTailscaleLoginCodeExitPath)) != log.end());

	runtime->stop();
	fixture.server.stop();
}

TEST_CASE("tailscale leaving during client init de-initializes once")
{
	TailscaleFixture fixture;
	REQUIRE(fixture.region_ok());
	fixture.reply_to_everything(
		R"({"status":"offline","start":true,"match":false})");
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_started && !status.tailscale_matched;
	}));

	runtime->notify_tailscale_page_visible(true);
	runtime->notify_tailscale_page_visible(false);

	REQUIRE(fixture.wait_for_requests([&fixture] {
		return fixture.server.request_count(
			       kTailscaleLoginCodeDisablePath) >= 1;
	}));

	CHECK(fixture.server.request_count(kTailscaleLoginCodeDisablePath) ==
	      1);
	CHECK(fixture.server.request_count(kTailscaleLoginCodeExitPath) == 0);

	runtime->stop();
	fixture.server.stop();
}

TEST_CASE("tailscale leaving while ready sends no de-initialization")
{
	TailscaleFixture fixture;
	REQUIRE(fixture.region_ok());
	fixture.reply_to_everything(
		R"({"status":"online","start":true,"match":true,)"
		R"("enabled":true,"name":"MockKVM","ip":"100.64.0.1"})");
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_started && status.tailscale_matched;
	}));

	const auto heartbeats_before =
		fixture.server.request_count(kTailscaleLoginCodeHeartbeatPath);
	runtime->notify_tailscale_page_visible(true);
	runtime->notify_tailscale_page_visible(false);

	REQUIRE(fixture.wait_for_requests([&fixture, heartbeats_before] {
		return fixture.server.request_count(
			       kTailscaleLoginCodeHeartbeatPath) >
		       heartbeats_before;
	}));

	CHECK(fixture.server.request_count(kTailscaleLoginCodeDisablePath) ==
	      0);
	CHECK(fixture.server.request_count(kTailscaleLoginCodeExitPath) == 0);

	runtime->stop();
	fixture.server.stop();
}

TEST_CASE("tailscale heartbeat ignores a legacy enabled field")
{
	TailscaleFixture fixture;
	REQUIRE(fixture.region_ok());
	fixture.reply_to_everything(
		R"({"status":"online","start":true,"match":true,)"
		R"("enabled":true,"name":"MockKVM","ip":"100.64.0.1"})");
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_started && status.tailscale_matched &&
		       status.tailscale_peer_ip == "100.64.0.1";
	}));

	runtime->stop();
	fixture.server.stop();
}

TEST_CASE("tailscale SAS keeps the last claim code and the normal cadence")
{
	TailscaleFixture fixture;
	REQUIRE(fixture.region_ok());
	fixture.reply_to_everything(
		R"({"status":"offline","start":true,"match":false})");
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	runtime->notify_tailscale_page_visible(true);
	REQUIRE(fixture.recorder.wait_for([](const RuntimeStatus &status) {
		return status.tailscale_started && !status.tailscale_matched &&
		       status.tailscale_code == "123456";
	}));

	const auto claim_seq = runtime->current_status().tailscale_seq;
	const auto claim_requests =
		fixture.server.request_count(kTailscaleLoginCodePath);

	constexpr auto kSasWaitBudget { 10s };
	fixture.reply(std::string(kTailscaleLoginCodePath), R"({"code":"123"})");
	REQUIRE(fixture.wait_for_requests(
		[&fixture, claim_requests] {
			return fixture.server.request_count(
				       kTailscaleLoginCodePath) > claim_requests;
		},
		kSasWaitBudget));
	CHECK(runtime->current_status().tailscale_code == "123456");
	CHECK(runtime->current_status().tailscale_seq == claim_seq);

	fixture.reply(std::string(kTailscaleLoginCodePath),
		      R"({"code":"654321"})");
	REQUIRE(fixture.wait_for_requests(
		[&runtime] {
			return runtime->current_status().tailscale_code ==
			       "654321";
		},
		kSasWaitBudget));
	CHECK(runtime->current_status().tailscale_seq > claim_seq);

	runtime->stop();
	fixture.server.stop();
}

TEST_CASE("tailscale lanes run without a server feature manifest")
{
	TailscaleFixture fixture;
	REQUIRE(fixture.region_ok());
	fixture.server.start();
	auto runtime = fixture.make_runtime();
	REQUIRE(runtime->start());

	runtime->notify_tailscale_page_visible(true);
	REQUIRE(fixture.wait_for_requests([&fixture] {
		return fixture.server.request_count(
			       kTailscaleLoginCodeHeartbeatPath) > 0;
	}));

	const RuntimeStatus sticky = runtime->current_status();
	CHECK_FALSE(sticky.tailscale_started);
	CHECK_FALSE(sticky.tailscale_matched);
	CHECK_FALSE(sticky.tailscale_online);
	CHECK(sticky.tailscale_gate_seq == 0);
	CHECK(sticky.tailscale_action_kind == TailscaleActionKind::None);
	CHECK(sticky.tailscale_action_message.empty());

	fixture.reply(std::string(kTailscaleLoginCodeStartPath),
		      R"({"status":true,"message":""})");
	REQUIRE(runtime->request_tailscale_start());
	REQUIRE(fixture.wait_for_requests([&fixture] {
		return fixture.server.request_count(
			       kTailscaleLoginCodeStartPath) >= 1;
	}));

	runtime->stop();
	fixture.server.stop();
}
