#pragma once

#include "lvgl_main/runtime_service.hpp"
#include "ui/tailscale_gate_view.hpp"

namespace ui
{
class PageManager;
}

namespace lvgl_main
{

ui::TailscaleActionKind to_ui_tailscale_action_kind(TailscaleActionKind kind);

bool tailscale_tile_active(const RuntimeStatus &status) noexcept;

void project_tailscale_status(ui::PageManager &page_manager,
			      const RuntimeStatus &status);

}
