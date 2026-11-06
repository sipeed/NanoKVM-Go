#include <catch2/catch_test_macros.hpp>

#include <string>
#include <utility>

#include "tailscale_projection.hpp"
#include "ui/tailscale_gate_view.hpp"

namespace
{

ui::TailscaleSnapshotView ready_snapshot(bool online, std::string ip)
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = true;
	snapshot.online = online;
	snapshot.ip = std::move(ip);
	return snapshot;
}

}

TEST_CASE("Tailscale gate view renders the stand-in before the first snapshot",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);

	REQUIRE(view.button_visible);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Start);
	REQUIRE(view.button_interactive);
	REQUIRE(view.code_visible);
	REQUIRE(view.code_source == ui::TailscaleCodeSource::Placeholder);
	REQUIRE_FALSE(view.ip_visible);
	REQUIRE(std::string(ui::tailscale_gate_button_text(view)) == "Start");
}

TEST_CASE("Tailscale gate view locks the start button while init is pending",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = false;
	input.snapshot = snapshot;
	input.init_pending = true;

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.button_visible);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Start);
	REQUIRE_FALSE(view.button_interactive);
	REQUIRE(view.code_source == ui::TailscaleCodeSource::Placeholder);
}

TEST_CASE("Tailscale gate view prefers the local failure message",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = false;
	input.snapshot = snapshot;
	input.has_message = true;

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.code_visible);
	REQUIRE(view.code_source == ui::TailscaleCodeSource::Message);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Start);
}

TEST_CASE("Tailscale gate view hides the button and IP row during client init",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = true;
	snapshot.started = true;
	snapshot.matched = false;
	snapshot.online = true;
	snapshot.ip = "100.64.0.1";
	input.snapshot = snapshot;
	input.has_message = true;

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE_FALSE(view.button_visible);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::None);
	REQUIRE_FALSE(view.button_interactive);
	REQUIRE(view.code_visible);
	REQUIRE(view.code_source == ui::TailscaleCodeSource::SessionCode);
	REQUIRE_FALSE(view.ip_visible);
	REQUIRE(std::string(ui::tailscale_gate_button_text(view)) == "--");

	snapshot.online = false;
	input.snapshot = snapshot;
	const ui::TailscaleGateView offline_view = ui::tailscale_gate_view(input);
	REQUIRE_FALSE(offline_view.button_visible);
	REQUIRE_FALSE(offline_view.ip_visible);
}

TEST_CASE("Tailscale gate view shows Exit and the IP row once the stage is ready",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "100.64.0.1");

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.button_visible);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Exit);
	REQUIRE(view.button_interactive);
	REQUIRE_FALSE(view.code_visible);
	REQUIRE(view.ip_visible);
	REQUIRE(std::string(ui::tailscale_gate_button_text(view)) == "Exit");
}

TEST_CASE("Tailscale gate view ignores the tunnel online flag for the ready IP row",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "100.64.0.1");

	ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.ip_visible);

	input.snapshot = ready_snapshot(true, "100.64.0.1");
	view = ui::tailscale_gate_view(input);
	REQUIRE(view.ip_visible);
}

TEST_CASE("Tailscale gate view hides the ready IP row while a failure message shows",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "100.64.0.1");
	input.has_message = true;

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Exit);
	REQUIRE(view.code_visible);
	REQUIRE(view.code_source == ui::TailscaleCodeSource::Message);
	REQUIRE_FALSE(view.ip_visible);
}

TEST_CASE("Tailscale gate view hides the ready IP row when the cached IP is empty",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "");

	ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE_FALSE(view.ip_visible);

	input.snapshot = ready_snapshot(true, "");
	view = ui::tailscale_gate_view(input);
	REQUIRE_FALSE(view.ip_visible);
}

TEST_CASE("Tailscale gate view locks the Exit button while it is pending",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "100.64.0.1");
	input.exit_pending = true;

	const ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.button_role == ui::TailscaleButtonRole::Exit);
	REQUIRE_FALSE(view.button_interactive);
	REQUIRE(view.ip_visible);
}

TEST_CASE("Tailscale gate view never reads the IP row from the pending flags",
	  "[ui][tailscale]")
{
	ui::TailscaleGateInput input;
	input.snapshot = ready_snapshot(false, "100.64.0.1");

	ui::TailscaleGateView view = ui::tailscale_gate_view(input);
	REQUIRE(view.ip_visible);

	input.init_pending = true;
	view = ui::tailscale_gate_view(input);
	REQUIRE(view.ip_visible);

	input.init_pending = false;
	input.exit_pending = true;
	view = ui::tailscale_gate_view(input);
	REQUIRE(view.ip_visible);
}

TEST_CASE("Tailscale runtime action kinds project onto the UI kinds",
	  "[ui][tailscale]")
{
	REQUIRE(lvgl_main::to_ui_tailscale_action_kind(
			lvgl_main::TailscaleActionKind::None) ==
		ui::TailscaleActionKind::None);
	REQUIRE(lvgl_main::to_ui_tailscale_action_kind(
			lvgl_main::TailscaleActionKind::Start) ==
		ui::TailscaleActionKind::Start);
	REQUIRE(lvgl_main::to_ui_tailscale_action_kind(
			lvgl_main::TailscaleActionKind::Exit) ==
		ui::TailscaleActionKind::Exit);
}

TEST_CASE("Tailscale tile is active only once initialized and matched",
	  "[ui][tailscale]")
{
	const auto active_for = [](bool started, bool matched) {
		lvgl_main::RuntimeStatus status;
		status.tailscale_started = started;
		status.tailscale_matched = matched;
		return lvgl_main::tailscale_tile_active(status);
	};

	REQUIRE_FALSE(active_for(false, false));
	REQUIRE_FALSE(active_for(false, true));
	REQUIRE_FALSE(active_for(true, false));
	REQUIRE(active_for(true, true));
}

TEST_CASE("Tailscale gate stage partitions the snapshot fields",
	  "[ui][tailscale]")
{
	ui::TailscaleSnapshotView snapshot;
	REQUIRE(ui::tailscale_gate_stage(snapshot) ==
		ui::TailscaleGateStage::Unknown);

	snapshot.has_snapshot = true;
	snapshot.started = false;
	snapshot.matched = false;
	REQUIRE(ui::tailscale_gate_stage(snapshot) ==
		ui::TailscaleGateStage::NotInitialized);

	snapshot.started = true;
	REQUIRE(ui::tailscale_gate_stage(snapshot) ==
		ui::TailscaleGateStage::ClientInit);

	snapshot.matched = true;
	REQUIRE(ui::tailscale_gate_stage(snapshot) ==
		ui::TailscaleGateStage::Ready);
}
