#include "tailscale_projection.hpp"

#include "ui/page_manager.hpp"

namespace lvgl_main
{

ui::TailscaleActionKind
to_ui_tailscale_action_kind(TailscaleActionKind kind)
{
	switch (kind) {
	case TailscaleActionKind::Start:
		return ui::TailscaleActionKind::Start;
	case TailscaleActionKind::Exit:
		return ui::TailscaleActionKind::Exit;
	case TailscaleActionKind::None:
		return ui::TailscaleActionKind::None;
	}
	return ui::TailscaleActionKind::None;
}

bool tailscale_tile_active(const RuntimeStatus &status) noexcept
{
	return status.tailscale_started && status.tailscale_matched;
}

void project_tailscale_status(ui::PageManager &page_manager,
			      const RuntimeStatus &status)
{
	ui::TailscaleSnapshotView snapshot;
	snapshot.has_snapshot = status.tailscale_gate_seq > 0;
	snapshot.online = status.tailscale_online;
	snapshot.started = status.tailscale_started;
	snapshot.matched = status.tailscale_matched;
	snapshot.name = status.tailscale_peer_name;
	snapshot.ip = status.tailscale_peer_ip;
	page_manager.set_tailscale_action_result(
		to_ui_tailscale_action_kind(status.tailscale_action_kind),
		status.tailscale_action_ok, status.tailscale_action_message,
		status.tailscale_action_seq);
	page_manager.set_tailscale_snapshot(snapshot);
	page_manager.set_tailscale_code(status.tailscale_code,
					status.tailscale_seq);
	page_manager.set_network_snapshot(snapshot);
	page_manager.set_network_code(status.tailscale_code,
				      status.tailscale_seq);
}

}
