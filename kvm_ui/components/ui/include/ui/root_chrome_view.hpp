#pragma once

#include <array>
#include <cstddef>
#include <cstdint>

#include <lvgl.h>

#include "ui/root_layout.hpp"
#include "ui/root_status_icon.hpp"

namespace ui
{

struct RootChromeNavigationModel {
	int32_t active_page_index { 0 };
	bool subpage_visible { false };
};

class RootChromeView final {
    public:
	explicit RootChromeView(bool scroll_fade_enabled);
	~RootChromeView();

	RootChromeView(const RootChromeView &) = delete;
	RootChromeView &operator=(const RootChromeView &) = delete;

	void create(lv_obj_t *root_parent);
	void render_navigation(RootChromeNavigationModel model);
	void begin_scroll_fade(float progress);
	void update_scroll_fade(float progress);
	void end_scroll_fade(RootChromeNavigationModel final_model);

	void set_status_icon_active(RootStatusIcon icon, bool active);

	void begin_shutdown();
	void shutdown();

    private:
	static constexpr int32_t kPageIndicatorWidth = 48;
	static constexpr int32_t kPageIndicatorHeight = 8;
	static constexpr int32_t kStatusBarFadeOverlayWidth = 160;
	static constexpr int32_t kStatusBarFadeOverlayHeight = 40;

	const bool _scroll_fade_enabled;
	bool _is_shutting_down = false;
	std::array<lv_obj_t *, kRootStatusIconCount> _status_icon_objs {};
	std::array<bool, kRootStatusIconCount> _status_icon_active { true, false,
										 false };
	lv_obj_t *_status_bar_obj = nullptr;
	lv_obj_t *_status_title_label = nullptr;
	lv_obj_t *_time_label = nullptr;
	lv_obj_t *_page_indicator_obj = nullptr;
	std::array<lv_obj_t *, root_layout::kContentPageCount>
		_page_indicator_dot_objs {};
	lv_timer_t *_clock_timer = nullptr;
	lv_obj_t *_status_bar_fade_overlay = nullptr;
	bool _status_bar_fade_active = false;

	void create_status_bar(lv_obj_t *root_parent);
	void create_status_bar_fade_overlay();
	void create_page_indicator(lv_obj_t *root_parent);
	void update_page_indicator(RootChromeNavigationModel model);
	void render_fixed_status_bar(RootChromeNavigationModel model);
	void apply_status_bar_left_content(bool settings_mode);
	void update_main_time();
	void apply_status_icon_visual(std::size_t index);
	void reflow_status_icons();
	void apply_status_bar_fade(float progress);

	static void clock_timer_cb(lv_timer_t *timer);
};

} // namespace ui
