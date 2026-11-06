#pragma once

#include <functional>

#include <lvgl.h>

#include "ui/bcona.hpp"
#include "ui/bconc.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class PanelPage : public SubPageBase {
    public:
	PanelPage();
	~PanelPage() override = default;

	void destroy() override;
	void set_brightness_percent(int percent);
	bool can_apply_runtime_brightness_percent(int percent) const;
	void set_sleep_enabled(bool enabled);
	void set_sleep_control_available(bool available);

	std::function<void(int)> on_brightness_change;
	std::function<void(bool)> on_sleep_enabled_change;

    private:
	static constexpr uint32_t SLEEP_COLOR = 0x615FFF;
	static constexpr uint32_t SLIDER_COLOR = 0xFFB900;
	static constexpr int BRIGHTNESS_MIN_PERCENT = 20;
	static constexpr int BRIGHTNESS_MAX_PERCENT = 100;

	int _brightness_percent = BRIGHTNESS_MAX_PERCENT;
	bool _sleep_enabled = false;
	bool _suppress_brightness_callback = false;
	bool _sleep_pending = false;
	bool _sleep_pending_target = false;
	bool _sleep_control_available = true;

	BConC *_interactive_slider = nullptr;
	BConA *_sleep_btn = nullptr;

	void build_content() override;
	void update_brightness_state();
	void update_sleep_state();
	void finish_sleep_pending_if_target(bool enabled);
	void on_subpage_destroy() override;
};

} // namespace ui
