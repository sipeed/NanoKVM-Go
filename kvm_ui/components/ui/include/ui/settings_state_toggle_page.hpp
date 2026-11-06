#pragma once

#include <cstdint>
#include <functional>

#include "ui/bcona.hpp"
#include "ui/control_state.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class SettingsStateTogglePage : public SubPageBase {
    public:
	SettingsStateTogglePage(const char *title, uint32_t active_color,
				const lv_image_dsc_t *icon);
	~SettingsStateTogglePage() override = default;

	void destroy() override;
	void set_enabled(bool enabled);
	void set_pending_state(bool pending, bool pending_target,
			       uint32_t pending_timeout_ms);
	ControlState *toggle_control() const;

	std::function<void(bool)> on_enabled_change;

    private:
	bool _enabled = false;
	bool _pending = false;
	bool _pending_target = false;
	uint32_t _active_color = BConA::COLOR_DEFAULT;
	const lv_image_dsc_t *_icon = nullptr;
	BConA *_toggle_btn = nullptr;

	void build_content() override;
	void update_toggle();
	void on_subpage_destroy() override;
};

} // namespace ui
