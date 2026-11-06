#pragma once

#include <cstdint>

namespace ui
{

struct SettingsServiceRuntimeState {
	bool known { false };
	bool refresh_failed { false };
	uint64_t event_seq { 0 };
	bool mcp_active { false };
	bool mcp_available { false };
	bool replay_active { false };
	bool replay_available { false };
};

} // namespace ui
