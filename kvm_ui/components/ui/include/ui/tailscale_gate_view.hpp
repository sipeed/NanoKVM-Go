#pragma once

#include <cstdint>
#include <string>

namespace ui
{

struct TailscaleSnapshotView {
	bool has_snapshot { false };
	bool online { false };
	bool started { false };
	bool matched { false };
	std::string name;
	std::string ip;
};

enum class TailscaleActionKind : uint8_t {
	None,
	Start,
	Exit,
};

enum class TailscaleButtonRole : uint8_t {
	None,
	Start,
	Exit,
};

enum class TailscaleCodeSource : uint8_t {
	Placeholder,
	SessionCode,
	Message,
};

enum class TailscaleGateStage : uint8_t {
	Unknown,
	NotInitialized,
	ClientInit,
	Ready,
};

struct TailscaleGateInput {
	TailscaleSnapshotView snapshot;
	bool init_pending { false };
	bool exit_pending { false };
	bool has_message { false };
};

struct TailscaleGateView {
	bool button_visible { false };
	TailscaleButtonRole button_role { TailscaleButtonRole::None };
	bool button_interactive { false };
	bool code_visible { false };
	TailscaleCodeSource code_source { TailscaleCodeSource::Placeholder };
	bool ip_visible { false };
};

constexpr TailscaleGateStage
tailscale_gate_stage(const TailscaleSnapshotView &snapshot) noexcept
{
	if (!snapshot.has_snapshot) {
		return TailscaleGateStage::Unknown;
	}
	if (!snapshot.started) {
		return TailscaleGateStage::NotInitialized;
	}
	return snapshot.matched ? TailscaleGateStage::Ready :
				  TailscaleGateStage::ClientInit;
}

constexpr bool
tailscale_gate_ready(const TailscaleSnapshotView &snapshot) noexcept
{
	return tailscale_gate_stage(snapshot) == TailscaleGateStage::Ready;
}

constexpr bool
tailscale_gate_client_init(const TailscaleSnapshotView &snapshot) noexcept
{
	return tailscale_gate_stage(snapshot) == TailscaleGateStage::ClientInit;
}

constexpr TailscaleGateView
tailscale_gate_view(const TailscaleGateInput &input) noexcept
{
	TailscaleGateView view;
	if (tailscale_gate_ready(input.snapshot)) {
		view.button_visible = true;
		view.button_role = TailscaleButtonRole::Exit;
		view.button_interactive = !input.exit_pending;
		view.code_visible = input.has_message;
		view.code_source = TailscaleCodeSource::Message;
		// The IP row follows the Exit button: whenever the stage is ready
		// the cached peer IP is shown regardless of the tunnel online flag,
		// and a failure message keeps the centered code area for itself.
		view.ip_visible = !input.has_message && !input.snapshot.ip.empty();
		return view;
	}

	if (tailscale_gate_client_init(input.snapshot)) {
		view.code_visible = true;
		view.code_source = TailscaleCodeSource::SessionCode;
		return view;
	}

	view.button_visible = true;
	view.button_role = TailscaleButtonRole::Start;
	view.button_interactive = !input.init_pending;
	view.code_visible = true;
	view.code_source = input.has_message ? TailscaleCodeSource::Message :
					       TailscaleCodeSource::Placeholder;
	return view;
}

constexpr const char *
tailscale_gate_button_text(const TailscaleGateView &view) noexcept
{
	if (view.button_role == TailscaleButtonRole::Start) {
		return "Start";
	}
	if (view.button_role == TailscaleButtonRole::Exit) {
		return "Exit";
	}
	return "--";
}

}
