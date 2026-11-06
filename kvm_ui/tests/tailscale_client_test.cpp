#include <chrono>
#include <filesystem>
#include <optional>
#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>

#include "fixtures/fake_tailscale_server.hpp"
#include "lvgl_main/runtime_state_store.hpp"
#include "nanokvm_tailscale_client.hpp"
#include "ui/local_service_state_client.hpp"

using namespace lvgl_main;

TEST_CASE("tailscale socket path matches the local service socket")
{
	CHECK(kTailscaleSocketPath ==
	      ui::local_service_state::kDefaultSocketPath);
}

TEST_CASE("tailscale start carries the consent flag and other routes do not")
{
	CHECK(kTailscaleLoginCodeStartPath ==
	      "/api/tailscale/login/code?start=true&consent=true");
	CHECK(kTailscaleLoginCodeStartPath.find("consent") !=
	      std::string_view::npos);

	CHECK(kTailscaleLoginCodePath.find("consent") ==
	      std::string_view::npos);
	CHECK(kTailscaleLoginCodeExitPath.find("consent") ==
	      std::string_view::npos);
	CHECK(kTailscaleLoginCodeHeartbeatPath.find("consent") ==
	      std::string_view::npos);
	CHECK(kTailscaleLoginCodeDisablePath.find("consent") ==
	      std::string_view::npos);
}

TEST_CASE("tailscale code response parses and sanitizes the code")
{
	CHECK(parse_tailscale_code_response(R"({"code":"ABC123"})") ==
	      std::optional<std::string> { "ABC123" });
	CHECK(parse_tailscale_code_response(R"({"code":""})") ==
	      std::optional<std::string> { "" });

	CHECK(parse_tailscale_code_response(
		      R"({"code":"a\u0001b\tc\n"})") ==
	      std::optional<std::string> { "abc" });
	CHECK(parse_tailscale_code_response(R"({"code":"caf\u00e9"})") ==
	      std::optional<std::string> { "caf" });

	CHECK_FALSE(parse_tailscale_code_response(R"({"code":123})"));
	CHECK_FALSE(parse_tailscale_code_response(R"({"other":"x"})"));
	CHECK_FALSE(parse_tailscale_code_response("not json"));
	CHECK_FALSE(parse_tailscale_code_response(""));
}

TEST_CASE("tailscale snapshot parses every gate field with defaults")
{
	const auto full = parse_tailscale_snapshot(
		R"({"status":"online","start":true,"match":true,)"
		R"("enabled":true,"name":"MockKVM","ip":"100.64.0.1"})");
	REQUIRE(full.has_value());
	CHECK(full->online);
	CHECK(full->started);
	CHECK(full->matched);
	CHECK(full->name == "MockKVM");
	CHECK(full->ip == "100.64.0.1");

	const auto disabled = parse_tailscale_snapshot(
		R"({"status":"online","start":true,"match":true,)"
		R"("enabled":false,"name":"MockKVM","ip":"100.64.0.1"})");
	REQUIRE(disabled.has_value());
	CHECK(disabled->online);
	CHECK(disabled->started);
	CHECK(disabled->matched);
	CHECK(disabled->name == "MockKVM");
	CHECK(disabled->ip == "100.64.0.1");

	const auto missing = parse_tailscale_snapshot(R"({})");
	REQUIRE(missing.has_value());
	CHECK_FALSE(missing->online);
	CHECK_FALSE(missing->started);
	CHECK_FALSE(missing->matched);
	CHECK(missing->name.empty());
	CHECK(missing->ip.empty());

	const auto wrong_types = parse_tailscale_snapshot(
		R"({"status":42,"start":"yes","match":1,)"
		R"("enabled":null,"name":7,"ip":false})");
	REQUIRE(wrong_types.has_value());
	CHECK_FALSE(wrong_types->online);
	CHECK_FALSE(wrong_types->started);
	CHECK_FALSE(wrong_types->matched);
	CHECK(wrong_types->name.empty());
	CHECK(wrong_types->ip.empty());

	const auto offline =
		parse_tailscale_snapshot(R"({"status":"offline","start":true})");
	REQUIRE(offline.has_value());
	CHECK_FALSE(offline->online);
	CHECK(offline->started);

	CHECK_FALSE(parse_tailscale_snapshot("not json"));
	CHECK_FALSE(parse_tailscale_snapshot(""));
	CHECK_FALSE(parse_tailscale_snapshot("[1,2,3]"));
	CHECK_FALSE(parse_tailscale_snapshot("null"));
}

TEST_CASE("tailscale snapshot sanitizes name and ip to printable ascii")
{
	const auto snapshot = parse_tailscale_snapshot(
		R"({"name":"caf\u00e9\u0001host","ip":"1.2.3.4\n\t"})");
	REQUIRE(snapshot.has_value());
	CHECK(snapshot->name == "cafhost");
	CHECK(snapshot->ip == "1.2.3.4");

	const auto long_name = parse_tailscale_snapshot(
		std::string("{\"name\":\"") + std::string(4096, 'x') +
		"\"}");
	REQUIRE(long_name.has_value());
	CHECK(long_name->name.size() == 4096);
}

TEST_CASE("tailscale stage derives the gate from the snapshot")
{
	CHECK(tailscale_stage(false, false, false) ==
	      TailscaleStage::Unknown);
	CHECK(tailscale_stage(false, true, true) ==
	      TailscaleStage::Unknown);
	CHECK(tailscale_stage(true, false, false) ==
	      TailscaleStage::NotInitialized);
	CHECK(tailscale_stage(true, false, true) ==
	      TailscaleStage::NotInitialized);
	CHECK(tailscale_stage(true, true, false) ==
	      TailscaleStage::ClientInit);
	CHECK(tailscale_stage(true, true, true) == TailscaleStage::Ready);
}

TEST_CASE("tailscale relinquish on leave deinitializes only during client init")
{
	CHECK_FALSE(tailscale_relinquish_on_leave(TailscaleStage::Unknown));
	CHECK_FALSE(
		tailscale_relinquish_on_leave(TailscaleStage::NotInitialized));
	CHECK(tailscale_relinquish_on_leave(TailscaleStage::ClientInit));
	CHECK_FALSE(tailscale_relinquish_on_leave(TailscaleStage::Ready));
}

TEST_CASE("tailscale tick plan keeps the heartbeat always on")
{
	const auto periodic_off = tailscale_tick_plan(false, true,
						      TailscaleStage::Ready);
	CHECK_FALSE(periodic_off.heartbeat);
	CHECK_FALSE(periodic_off.code);

	const auto hidden = tailscale_tick_plan(true, false,
						TailscaleStage::ClientInit);
	CHECK(hidden.heartbeat);
	CHECK_FALSE(hidden.code);

	const auto visible_unknown =
		tailscale_tick_plan(true, true, TailscaleStage::Unknown);
	CHECK(visible_unknown.heartbeat);
	CHECK_FALSE(visible_unknown.code);

	const auto visible_not_initialized = tailscale_tick_plan(
		true, true, TailscaleStage::NotInitialized);
	CHECK(visible_not_initialized.heartbeat);
	CHECK_FALSE(visible_not_initialized.code);

	const auto visible_client_init = tailscale_tick_plan(
		true, true, TailscaleStage::ClientInit);
	CHECK(visible_client_init.heartbeat);
	CHECK(visible_client_init.code);

	const auto visible_ready =
		tailscale_tick_plan(true, true, TailscaleStage::Ready);
	CHECK(visible_ready.heartbeat);
	CHECK_FALSE(visible_ready.code);
}

TEST_CASE("tailscale code lane restarts when the pairing stage reopens")
{
	const auto closed = tailscale_code_lane_step(true, false, false);
	CHECK_FALSE(closed.reopened);
	CHECK_FALSE(closed.attempt);

	const auto reopened = tailscale_code_lane_step(false, true, false);
	CHECK(reopened.reopened);
	CHECK(reopened.attempt);

	const auto still_closed = tailscale_code_lane_step(false, false, true);
	CHECK_FALSE(still_closed.reopened);
	CHECK_FALSE(still_closed.attempt);

	const auto open_due = tailscale_code_lane_step(true, true, true);
	CHECK_FALSE(open_due.reopened);
	CHECK(open_due.attempt);

	const auto open_waiting = tailscale_code_lane_step(true, true, false);
	CHECK_FALSE(open_waiting.reopened);
	CHECK_FALSE(open_waiting.attempt);
}

TEST_CASE("tailscale action response parses status and message")
{
	const auto ok = parse_tailscale_action_response(
		R"({"status":true,"message":"ignored"})");
	REQUIRE(ok.has_value());
	CHECK(ok->ok);
	CHECK(ok->message.empty());

	const auto failed = parse_tailscale_action_response(
		R"({"status":false,"message":"NO_DAEMON"})");
	REQUIRE(failed.has_value());
	CHECK_FALSE(failed->ok);
	CHECK(failed->message == "NO_DAEMON");

	const auto no_message =
		parse_tailscale_action_response(R"({"status":false})");
	REQUIRE(no_message.has_value());
	CHECK_FALSE(no_message->ok);
	CHECK(no_message->message.empty());

	const auto long_message = parse_tailscale_action_response(
		R"({"status":false,"message":"ABCDEFGHIJKLMN"})");
	REQUIRE(long_message.has_value());
	CHECK(long_message->message == "ABCDEFGHIJ");

	const auto sanitized = parse_tailscale_action_response(
		R"({"status":false,"message":"a\u0001b\tcdefghijkl"})");
	REQUIRE(sanitized.has_value());
	CHECK(sanitized->message == "abcdefghij");

	const auto non_ascii = parse_tailscale_action_response(
		R"({"status":false,"message":"caf\u00e9"})");
	REQUIRE(non_ascii.has_value());
	CHECK(non_ascii->message == "caf");

	CHECK_FALSE(parse_tailscale_action_response(R"({"status":"true"})"));
	CHECK_FALSE(parse_tailscale_action_response(R"({"message":"x"})"));
	CHECK_FALSE(parse_tailscale_action_response("[1]"));
	CHECK_FALSE(parse_tailscale_action_response("nope"));
	CHECK_FALSE(parse_tailscale_action_response(""));
}

TEST_CASE("tailscale online directive requires an exact shape")
{
	const auto peer =
		parse_tailscale_online_code("%online%myhost%100.64.0.1%");
	REQUIRE(peer.has_value());
	CHECK(peer->name == "myhost");
	CHECK(peer->ip == "100.64.0.1");

	CHECK_FALSE(parse_tailscale_online_code("%online%%100.64.0.1%"));
	CHECK_FALSE(parse_tailscale_online_code("%online%myhost%%"));
	CHECK_FALSE(parse_tailscale_online_code("%online%myhost%100.64.0.1"));
	CHECK_FALSE(parse_tailscale_online_code("%online%a%b%c%"));
	CHECK_FALSE(parse_tailscale_online_code("%online%h\u00e9llo%1.2.3.4%"));
	CHECK_FALSE(parse_tailscale_online_code("%online%"));
	CHECK_FALSE(parse_tailscale_online_code("plain-code"));
}

TEST_CASE("tailscale off directive and sanitize boundaries")
{
	CHECK(is_tailscale_off_code("%off%"));
	CHECK_FALSE(is_tailscale_off_code("%off%x"));
	CHECK_FALSE(is_tailscale_off_code("off"));

	CHECK(sanitize_tailscale_display_code("a\r\nb\tc") == "abc");
	CHECK(sanitize_tailscale_display_code("keep 0x20..0x7E!") ==
	      "keep 0x20..0x7E!");
	CHECK(sanitize_tailscale_display_code("\x7f") == "");
	CHECK(sanitize_tailscale_display_code("caf\xc3\xa9") == "caf");
}

TEST_CASE("tailscale sas detection accepts only three ascii digits")
{
	CHECK(is_tailscale_sas_code("123"));
	CHECK(is_tailscale_sas_code("000"));
	CHECK(is_tailscale_sas_code("987"));

	CHECK_FALSE(is_tailscale_sas_code("1234"));
	CHECK_FALSE(is_tailscale_sas_code("12"));
	CHECK_FALSE(is_tailscale_sas_code(""));
	CHECK_FALSE(is_tailscale_sas_code("12a"));
	CHECK_FALSE(is_tailscale_sas_code("1 2"));
	CHECK_FALSE(is_tailscale_sas_code("\xEF\xBC\x91\xEF\xBC\x92\xEF\xBC\x93"));
	CHECK_FALSE(is_tailscale_sas_code("12\xEF\xBC\x93"));
	CHECK_FALSE(is_tailscale_sas_code("%off%"));
}

TEST_CASE("tailscale session deduction covers directives and plain codes")
{
	const auto plain = tailscale_session_from_code("654321");
	REQUIRE(plain.has_value());
	CHECK_FALSE(plain->online);
	CHECK(plain->code == "654321");
	CHECK(plain->peer_name.empty());
	CHECK(plain->peer_ip.empty());

	const auto online =
		tailscale_session_from_code("%online%host%1.2.3.4%");
	REQUIRE(online.has_value());
	CHECK(online->online);
	CHECK(online->code.empty());
	CHECK(online->peer_name == "host");
	CHECK(online->peer_ip == "1.2.3.4");

	const auto off = tailscale_session_from_code("%off%");
	REQUIRE(off.has_value());
	CHECK_FALSE(off->online);
	CHECK(off->code.empty());
	CHECK(off->peer_name.empty());

	CHECK_FALSE(tailscale_session_from_code("%online%broken"));
	CHECK_FALSE(tailscale_session_from_code("%online%%%"));
}

TEST_CASE("tailscale failure backoff doubles up to thirty seconds")
{
	using namespace std::chrono_literals;
	CHECK(tailscale_interval_for_failures(0) == 3s);
	CHECK(tailscale_interval_for_failures(1) == 1s);
	CHECK(tailscale_interval_for_failures(2) == 2s);
	CHECK(tailscale_interval_for_failures(3) == 4s);
	CHECK(tailscale_interval_for_failures(4) == 8s);
	CHECK(tailscale_interval_for_failures(5) == 16s);
	CHECK(tailscale_interval_for_failures(6) == 30s);
	CHECK(tailscale_interval_for_failures(8) == 30s);
}

TEST_CASE("tailscale code backoff never slows the heartbeat cadence")
{
	using namespace std::chrono_literals;
	using Clock = std::chrono::steady_clock;
	const auto now = Clock::time_point { 100s };
	uint8_t heartbeat_failures = 0;
	uint8_t code_failures = 0;

	CHECK(tailscale_next_refresh_at(now, true, heartbeat_failures) - now ==
	      3s);
	CHECK(heartbeat_failures == 0);

	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 1s);
	CHECK(code_failures == 1);
	CHECK(heartbeat_failures == 0);
	CHECK(tailscale_next_refresh_at(now, true, heartbeat_failures) - now ==
	      3s);

	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 2s);
	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 4s);
	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 8s);
	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 16s);
	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 30s);
	CHECK(tailscale_next_refresh_at(now, false, code_failures) - now == 30s);
	CHECK(code_failures == 7);

	CHECK(tailscale_next_refresh_at(now, true, heartbeat_failures) - now ==
	      3s);
	CHECK(heartbeat_failures == 0);
	CHECK(code_failures == 7);

	CHECK(tailscale_next_refresh_at(now, true, code_failures) - now == 3s);
	CHECK(code_failures == 0);

	CHECK(tailscale_next_refresh_at(now, false, heartbeat_failures) - now ==
	      1s);
	CHECK(tailscale_next_refresh_at(now, false, heartbeat_failures) - now ==
	      2s);
	CHECK(heartbeat_failures == 2);
	CHECK(code_failures == 0);
}

TEST_CASE("TailscalePatch applies newer sessions and rejects stale ones")
{
	RuntimeStateStore store;
	RuntimeStatus first;
	first.tailscale_online = true;
	first.tailscale_peer_name = "host";
	first.tailscale_peer_ip = "100.64.0.1";
	first.tailscale_seq = 4;
	REQUIRE(store.commit(TailscalePatch { .status = first }));
	CHECK(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_peer_name == "host");
	CHECK(store.current_status().tailscale_seq == 4);

	RuntimeStatus stale = first;
	stale.tailscale_online = false;
	CHECK_FALSE(store.commit(TailscalePatch { .status = stale }));
	CHECK(store.current_status().tailscale_online);

	RuntimeStatus next;
	next.tailscale_online = false;
	next.tailscale_code = "123456";
	next.tailscale_seq = 5;
	REQUIRE(store.commit(TailscalePatch { .status = next }));
	CHECK_FALSE(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_code == "123456");
	CHECK(store.current_status().tailscale_peer_name.empty());
	CHECK(store.current_status().tailscale_seq == 5);
}

TEST_CASE("TailscaleSnapshotPatch applies the gate and keeps failures sticky")
{
	RuntimeStateStore store;

	RuntimeStatus first;
	first.tailscale_online = true;
	first.tailscale_started = true;
	first.tailscale_matched = true;
	first.tailscale_peer_name = "host";
	first.tailscale_peer_ip = "100.64.0.1";
	first.tailscale_gate_seq = 1;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = first, .snapshot_valid = true }));
	CHECK(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_started);
	CHECK(store.current_status().tailscale_matched);
	CHECK(store.current_status().tailscale_peer_name == "host");
	CHECK(store.current_status().tailscale_peer_ip == "100.64.0.1");
	CHECK(store.current_status().tailscale_gate_seq == 1);

	RuntimeStatus duplicate;
	duplicate.tailscale_gate_seq = 1;
	CHECK_FALSE(store.commit(TailscaleSnapshotPatch {
		.status = duplicate, .snapshot_valid = true }));

	RuntimeStatus failure;
	failure.tailscale_online = false;
	failure.tailscale_seq = 1;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = failure, .snapshot_valid = false }));
	CHECK_FALSE(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_started);
	CHECK(store.current_status().tailscale_matched);
	CHECK(store.current_status().tailscale_peer_name == "host");
	CHECK(store.current_status().tailscale_peer_ip == "100.64.0.1");
	CHECK(store.current_status().tailscale_gate_seq == 1);
	CHECK(store.current_status().tailscale_seq == 1);

	CHECK_FALSE(store.commit(TailscaleSnapshotPatch {
		.status = failure, .snapshot_valid = false }));

	RuntimeStatus recovered;
	recovered.tailscale_online = true;
	recovered.tailscale_started = true;
	recovered.tailscale_matched = false;
	recovered.tailscale_gate_seq = 2;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = recovered, .snapshot_valid = true }));
	CHECK(store.current_status().tailscale_online);
	CHECK_FALSE(store.current_status().tailscale_matched);
	CHECK(store.current_status().tailscale_gate_seq == 2);
}

TEST_CASE("TailscaleSnapshotPatch drops snapshots older than an action")
{
	RuntimeStateStore store;

	RuntimeStatus before;
	before.tailscale_started = true;
	before.tailscale_matched = true;
	before.tailscale_gate_seq = 1;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = before, .snapshot_valid = true }));

	RuntimeStatus action;
	action.tailscale_action_seq = 1;
	action.tailscale_action_kind = TailscaleActionKind::Exit;
	action.tailscale_action_ok = true;
	action.tailscale_action_generation = 1;
	REQUIRE(store.commit(
		TailscaleActionResultPatch { .status = action }));
	CHECK(store.current_status().tailscale_action_generation == 1);

	RuntimeStatus stale = before;
	stale.tailscale_gate_seq = 2;
	CHECK_FALSE(store.commit(TailscaleSnapshotPatch {
		.status = stale, .snapshot_valid = true }));
	CHECK(store.current_status().tailscale_gate_seq == 1);

	RuntimeStatus fresh = stale;
	fresh.tailscale_action_generation = 1;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = fresh, .snapshot_valid = true }));
	CHECK(store.current_status().tailscale_gate_seq == 2);
}

TEST_CASE("periodic refresh never erases the Tailscale gate snapshot")
{
	RuntimeStateStore store;
	RuntimeStatus gate;
	gate.tailscale_started = true;
	gate.tailscale_matched = true;
	gate.tailscale_peer_name = "host";
	gate.tailscale_peer_ip = "100.64.0.1";
	gate.tailscale_gate_seq = 1;
	REQUIRE(store.commit(TailscaleSnapshotPatch {
		.status = gate, .snapshot_valid = true }));

	RuntimeStatus observed;
	observed.app_version = "app";
	REQUIRE(store.commit(RefreshObservationPatch { .status = observed }));
	CHECK(store.current_status().tailscale_started);
	CHECK(store.current_status().tailscale_matched);
	CHECK(store.current_status().tailscale_peer_name == "host");
	CHECK(store.current_status().tailscale_peer_ip == "100.64.0.1");
	CHECK(store.current_status().tailscale_gate_seq == 1);
}

TEST_CASE("TailscaleActionResultPatch applies once and rejects stale results")
{
	RuntimeStateStore store;

	RuntimeStatus start;
	start.tailscale_action_seq = 1;
	start.tailscale_action_kind = TailscaleActionKind::Start;
	start.tailscale_action_ok = true;
	start.tailscale_action_generation = 1;
	REQUIRE(store.commit(TailscaleActionResultPatch { .status = start }));
	CHECK(store.current_status().tailscale_action_seq == 1);
	CHECK(store.current_status().tailscale_action_kind ==
	      TailscaleActionKind::Start);
	CHECK(store.current_status().tailscale_action_ok);
	CHECK(store.current_status().tailscale_action_message.empty());

	CHECK_FALSE(
		store.commit(TailscaleActionResultPatch { .status = start }));

	RuntimeStatus exit_action;
	exit_action.tailscale_action_seq = 2;
	exit_action.tailscale_action_kind = TailscaleActionKind::Exit;
	exit_action.tailscale_action_ok = false;
	exit_action.tailscale_action_message = "NO_DAEMON";
	exit_action.tailscale_action_generation = 2;
	REQUIRE(store.commit(
		TailscaleActionResultPatch { .status = exit_action }));
	CHECK(store.current_status().tailscale_action_seq == 2);
	CHECK(store.current_status().tailscale_action_kind ==
	      TailscaleActionKind::Exit);
	CHECK_FALSE(store.current_status().tailscale_action_ok);
	CHECK(store.current_status().tailscale_action_message == "NO_DAEMON");
	CHECK(store.current_status().tailscale_action_generation == 2);
}

TEST_CASE("periodic refresh never erases the Tailscale session")
{
	RuntimeStateStore store;
	RuntimeStatus session;
	session.tailscale_online = true;
	session.tailscale_peer_name = "host";
	session.tailscale_peer_ip = "100.64.0.1";
	session.tailscale_seq = 3;
	REQUIRE(store.commit(TailscalePatch { .status = session }));

	RuntimeStatus observed;
	observed.app_version = "app";
	REQUIRE(store.commit(RefreshObservationPatch { .status = observed }));
	CHECK(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_peer_name == "host");
	CHECK(store.current_status().tailscale_seq == 3);

	RuntimeFlagsPatch flags {};
	REQUIRE(store.commit(flags));
	CHECK(store.current_status().tailscale_online);
	CHECK(store.current_status().tailscale_seq == 3);
}

TEST_CASE("tailscale client reuses one keep-alive connection")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodePath),
			 { 200, R"({"code":"654321"})" });
	server.set_reply(std::string(kTailscaleLoginCodeHeartbeatPath),
			 { 200, R"({"status":"online"})" });
	server.start();

	TailscaleClient client(socket_path.string());
	CHECK(client.fetch_code() == std::optional<std::string> { "654321" });
	CHECK(client.fetch_code() == std::optional<std::string> { "654321" });
	CHECK(client.fetch_snapshot().has_value());

	CHECK(server.request_count(kTailscaleLoginCodePath) == 2);
	CHECK(server.request_count(kTailscaleLoginCodeHeartbeatPath) == 1);
	CHECK(server.connection_count() == 1);
}

TEST_CASE("tailscale disable reads the directive response without parsing it")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeDisablePath),
			 { 200, R"({"code":"%off%"})" });
	server.set_reply(std::string(kTailscaleLoginCodeHeartbeatPath),
			 { 200, R"({"status":"online"})" });
	server.start();

	TailscaleClient client(socket_path.string());
	REQUIRE(client.request_disable());

	CHECK(client.fetch_snapshot().has_value());
	CHECK(server.request_count(kTailscaleLoginCodeDisablePath) == 1);
	CHECK(server.connection_count() == 1);
}

TEST_CASE("tailscale client reports failures for non-2xx and malformed bodies")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodePath),
			 { 500, R"({"error":"boom"})" });
	server.set_reply(std::string(kTailscaleLoginCodeHeartbeatPath),
			 { 200, "not json" });
	server.start();

	TailscaleClient client(socket_path.string());
	CHECK_FALSE(client.fetch_code());
	CHECK_FALSE(client.fetch_snapshot());
	CHECK_FALSE(client.request_disable());
}

TEST_CASE("tailscale client dispatches the action endpoints")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeStartPath),
			 { 200, R"({"status":true})" });
	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 200, R"({"status":false,"message":"NO_DAEMON"})" });
	server.start();

	TailscaleClient client(socket_path.string());
	const auto start = client.request_action(TailscaleAction::Start);
	CHECK(start.ok);
	CHECK(start.message.empty());

	const auto exit = client.request_action(TailscaleAction::Exit);
	CHECK_FALSE(exit.ok);
	CHECK(exit.message == "NO_DAEMON");

	CHECK(server.request_count(kTailscaleLoginCodeStartPath) == 1);
	CHECK(server.request_count(kTailscaleLoginCodeExitPath) == 1);
}

TEST_CASE("tailscale exit endpoint reports success and daemon failures")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 200, R"({"status":false,"message":"NO_DAEMON"})" });
	server.start();

	CHECK(kTailscaleLoginCodeExitPath ==
	      "/api/tailscale/login/code?exit=true");

	TailscaleClient client(socket_path.string());
	const auto failed = client.request_action(TailscaleAction::Exit);
	CHECK_FALSE(failed.ok);
	CHECK(failed.message == "NO_DAEMON");

	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 200, R"({"status":true})" });
	const auto ok = client.request_action(TailscaleAction::Exit);
	CHECK(ok.ok);
	CHECK(ok.message.empty());

	CHECK(server.request_count(kTailscaleLoginCodeExitPath) == 2);
	CHECK(server.request_log() ==
	      std::vector<std::string>(2,
				       std::string(
					       kTailscaleLoginCodeExitPath)));
}

TEST_CASE("tailscale snapshot and action lanes use separate connections")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(
		std::string(kTailscaleLoginCodeHeartbeatPath),
		{ 200,
		  R"({"status":"online","start":true,"match":true,)"
		  R"("enabled":false,"name":"host","ip":"100.64.0.1"})" });
	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 200, R"({"status":true})" });
	server.start();

	TailscaleClient client(socket_path.string());
	const auto snapshot = client.fetch_snapshot();
	REQUIRE(snapshot.has_value());
	CHECK(snapshot->online);
	CHECK(snapshot->started);
	CHECK(snapshot->matched);
	CHECK(snapshot->name == "host");
	CHECK(snapshot->ip == "100.64.0.1");

	CHECK(client.request_action(TailscaleAction::Exit).ok);
	CHECK(server.request_count(kTailscaleLoginCodeHeartbeatPath) == 1);
	CHECK(server.connection_count() == 2);
}

TEST_CASE("tailscale action local failures collapse to FAIL")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeStartPath),
			 { 200, "not json" });
	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 500, R"({"error":"boom"})" });
	server.start();

	TailscaleClient client(socket_path.string());
	CHECK(client.request_action(TailscaleAction::Start).message ==
	      "FAIL");
	CHECK(client.request_action(TailscaleAction::Exit).message == "FAIL");

	tests::TempDirectory missing_temp;
	TailscaleClient missing_client(
		(missing_temp.path() / "missing.sock").string());
	CHECK(missing_client.request_action(TailscaleAction::Start).message ==
	      "FAIL");
	CHECK(missing_client.request_action(TailscaleAction::Exit).message ==
	      "FAIL");
}

TEST_CASE("tailscale action timeout reports TIMEOUT")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeStartPath),
			 { 200, R"({"status":true})",
			   std::chrono::milliseconds(600) });
	server.start();

	TailscaleClient client(socket_path.string(),
			       std::chrono::milliseconds(120));
	const auto result = client.request_action(TailscaleAction::Start);
	CHECK_FALSE(result.ok);
	CHECK(result.message == "TIMEOUT");
}

TEST_CASE("tailscale exit action timeout reports TIMEOUT")
{
	tests::TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "state.sock";
	tests::FakeTailscaleServer server(socket_path);
	server.set_reply(std::string(kTailscaleLoginCodeExitPath),
			 { 200, R"({"status":true})",
			   std::chrono::milliseconds(600) });
	server.start();

	TailscaleClient client(socket_path.string(),
			       std::chrono::milliseconds(120));
	const auto result = client.request_action(TailscaleAction::Exit);
	CHECK_FALSE(result.ok);
	CHECK(result.message == "TIMEOUT");
}

TEST_CASE("tailscale client surfaces transport errors")
{
	tests::TempDirectory temp;
	const std::filesystem::path missing = temp.path() / "missing.sock";

	TailscaleClient client(missing.string());
	CHECK_FALSE(client.fetch_code());
	CHECK_FALSE(client.last_error().empty());
}
