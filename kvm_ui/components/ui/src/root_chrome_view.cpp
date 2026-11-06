#include "ui/root_chrome_view.hpp"

#include <algorithm>
#include <array>
#include <cstdio>
#include <ctime>

#include "lv_safe.hpp"
#include "ui/status_bar_transition.hpp"

extern "C" {
LV_IMAGE_DECLARE(img_main_state_wifi);
LV_IMAGE_DECLARE(img_main_state_video);
LV_IMAGE_DECLARE(img_main_state_keyboard_and_mouse);
LV_FONT_DECLARE(MontserratBold_26);
LV_FONT_DECLARE(MontserratBold_28);
}

namespace ui
{

RootChromeView::RootChromeView(bool scroll_fade_enabled)
	: _scroll_fade_enabled(scroll_fade_enabled)
{
}

RootChromeView::~RootChromeView()
{
	shutdown();
}

void RootChromeView::create(lv_obj_t *root_parent)
{
	if (_is_shutting_down || _status_bar_obj != nullptr ||
	    root_parent == nullptr) {
		return;
	}

	create_status_bar(root_parent);
	create_page_indicator(root_parent);
	render_navigation({});
}

void RootChromeView::render_navigation(RootChromeNavigationModel model)
{
	if (_is_shutting_down) {
		return;
	}

	update_page_indicator(model);
	if (_status_bar_fade_active) {
		return;
	}
	render_fixed_status_bar(model);
}

void RootChromeView::begin_scroll_fade(float progress)
{
	if (_is_shutting_down || !_scroll_fade_enabled) {
		return;
	}

	_status_bar_fade_active = true;
	update_scroll_fade(progress);
}

void RootChromeView::update_scroll_fade(float progress)
{
	if (_is_shutting_down || !_status_bar_fade_active ||
	    !_scroll_fade_enabled) {
		return;
	}

	apply_status_bar_fade(progress);
}

void RootChromeView::end_scroll_fade(RootChromeNavigationModel final_model)
{
	if (_is_shutting_down) {
		return;
	}

	_status_bar_fade_active = false;
	render_navigation(final_model);
}

void RootChromeView::set_status_icon_active(RootStatusIcon icon, bool active)
{
	if (!is_valid_root_status_icon(icon)) {
		return;
	}

	const std::size_t index = root_status_icon_index(icon);
	_status_icon_active[index] = active;
}

void RootChromeView::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	if (_clock_timer != nullptr) {
		lv_timer_delete(_clock_timer);
		_clock_timer = nullptr;
	}
}

void RootChromeView::shutdown()
{
	begin_shutdown();
	if (_status_bar_fade_overlay != nullptr) {
		lv_delete_if_valid(_status_bar_fade_overlay);
		_status_bar_fade_overlay = nullptr;
	}

	lv_delete_if_valid(_status_bar_obj);
	lv_delete_if_valid(_page_indicator_obj);

	_status_bar_obj = nullptr;
	_status_icon_objs.fill(nullptr);
	_status_title_label = nullptr;
	_time_label = nullptr;
	_page_indicator_obj = nullptr;
	_page_indicator_dot_objs.fill(nullptr);
	_status_bar_fade_active = false;
}

void RootChromeView::create_status_bar(lv_obj_t *root_parent)
{
	_status_bar_obj = lv_obj_create(root_parent);
	lv_obj_set_pos(_status_bar_obj, 15, 13);
	lv_obj_set_size(_status_bar_obj, 244, 40);
	lv_obj_set_style_bg_opa(_status_bar_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_status_bar_obj, 0, 0);
	lv_obj_set_style_pad_all(_status_bar_obj, 0, 0);
	lv_obj_clear_flag(_status_bar_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_status_bar_obj, LV_OBJ_FLAG_CLICKABLE);

	_status_title_label = lv_label_create(_status_bar_obj);
	lv_obj_set_style_text_color(_status_title_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_status_title_label, &MontserratBold_28, 0);
	lv_label_set_text(_status_title_label, "Settings");
	lv_obj_align(_status_title_label, LV_ALIGN_LEFT_MID, 0, 0);
	lv_obj_add_flag(_status_title_label, LV_OBJ_FLAG_HIDDEN);
	lv_obj_clear_flag(_status_title_label, LV_OBJ_FLAG_CLICKABLE);

	const std::array<const lv_image_dsc_t *, kRootStatusIconCount> icons = {
		&img_main_state_wifi,
		&img_main_state_video,
		&img_main_state_keyboard_and_mouse,
	};
	for (std::size_t index = 0; index < icons.size(); ++index) {
		lv_obj_t *icon_obj = lv_image_create(_status_bar_obj);
		lv_image_set_src(icon_obj, icons[index]);
		lv_obj_clear_flag(icon_obj, LV_OBJ_FLAG_CLICKABLE);
		_status_icon_objs[index] = icon_obj;
		apply_status_icon_visual(index);
	}
	reflow_status_icons();

	_time_label = lv_label_create(_status_bar_obj);
	lv_obj_set_style_text_color(_time_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_time_label, &MontserratBold_26, 0);
	lv_label_set_text(_time_label, "--:--");
	lv_obj_align(_time_label, LV_ALIGN_RIGHT_MID, 0, 0);
	lv_obj_clear_flag(_time_label, LV_OBJ_FLAG_CLICKABLE);
	update_main_time();
	_clock_timer = lv_timer_create(clock_timer_cb, 1000, this);

	create_status_bar_fade_overlay();
}

void RootChromeView::create_status_bar_fade_overlay()
{
	if (_status_bar_obj == nullptr || _status_bar_fade_overlay != nullptr) {
		return;
	}

	_status_bar_fade_overlay = lv_obj_create(_status_bar_obj);
	lv_obj_set_size(_status_bar_fade_overlay, kStatusBarFadeOverlayWidth,
			kStatusBarFadeOverlayHeight);
	lv_obj_set_pos(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_bg_color(_status_bar_fade_overlay, lv_color_black(),
				  0);
	lv_obj_set_style_bg_opa(_status_bar_fade_overlay, LV_OPA_TRANSP, 0);
	lv_obj_set_style_radius(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_border_width(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_pad_all(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_margin_all(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_shadow_width(_status_bar_fade_overlay, 0, 0);
	lv_obj_set_style_outline_width(_status_bar_fade_overlay, 0, 0);
	lv_obj_clear_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_HIDDEN);
}

void RootChromeView::create_page_indicator(lv_obj_t *root_parent)
{
	_page_indicator_obj = lv_obj_create(root_parent);
	lv_obj_set_size(_page_indicator_obj, kPageIndicatorWidth,
			kPageIndicatorHeight);
	lv_obj_set_pos(_page_indicator_obj,
		       (root_layout::kScreenWidth - kPageIndicatorWidth) / 2,
		       root_layout::kPageIndicatorY);
	lv_obj_set_style_bg_opa(_page_indicator_obj, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_page_indicator_obj, 0, 0);
	lv_obj_set_style_pad_all(_page_indicator_obj, 0, 0);
	lv_obj_clear_flag(_page_indicator_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_page_indicator_obj, LV_OBJ_FLAG_CLICKABLE);

	for (lv_obj_t *&dot_obj : _page_indicator_dot_objs) {
		dot_obj = lv_obj_create(_page_indicator_obj);
		lv_obj_set_style_bg_color(dot_obj, lv_color_white(), 0);
		lv_obj_set_style_border_width(dot_obj, 0, 0);
		lv_obj_set_style_pad_all(dot_obj, 0, 0);
		lv_obj_set_style_radius(dot_obj, LV_RADIUS_CIRCLE, 0);
		lv_obj_clear_flag(dot_obj, LV_OBJ_FLAG_SCROLLABLE);
		lv_obj_clear_flag(dot_obj, LV_OBJ_FLAG_CLICKABLE);
	}
}

void RootChromeView::update_page_indicator(RootChromeNavigationModel model)
{
	if (_page_indicator_obj == nullptr) {
		return;
	}

	if (model.subpage_visible) {
		lv_obj_add_flag(_page_indicator_obj, LV_OBJ_FLAG_HIDDEN);
		return;
	}

	lv_obj_clear_flag(_page_indicator_obj, LV_OBJ_FLAG_HIDDEN);
	const int32_t active_index = std::clamp(
		model.active_page_index, 0,
		static_cast<int32_t>(root_layout::kContentPageCount - 1U));
	constexpr int32_t kActiveDotWidth = 8;
	constexpr int32_t kInactiveDotWidth = 5;
	constexpr int32_t kDotGap = 10;
	int32_t total_width = 0;
	for (std::size_t index = 0; index < _page_indicator_dot_objs.size();
	     ++index) {
		const bool active = static_cast<int32_t>(index) == active_index;
		total_width += active ? kActiveDotWidth : kInactiveDotWidth;
	}
	total_width += kDotGap * static_cast<int32_t>(
					 _page_indicator_dot_objs.size() - 1U);
	int32_t dot_x = (kPageIndicatorWidth - total_width) / 2;
	for (std::size_t index = 0; index < _page_indicator_dot_objs.size();
	     ++index) {
		const bool active = static_cast<int32_t>(index) == active_index;
		const int32_t dot_size = active ? kActiveDotWidth :
						  kInactiveDotWidth;
		if (lv_obj_t *dot_obj = _page_indicator_dot_objs[index];
		    dot_obj != nullptr) {
			lv_obj_set_size(dot_obj, dot_size, dot_size);
			lv_obj_set_style_bg_opa(
				dot_obj, active ? LV_OPA_COVER : LV_OPA_50, 0);
			lv_obj_set_pos(dot_obj, dot_x,
				       (kPageIndicatorHeight - dot_size) / 2);
		}
		dot_x += dot_size + kDotGap;
	}
}

void RootChromeView::render_fixed_status_bar(RootChromeNavigationModel model)
{
	const int32_t active_index = std::clamp(
		model.active_page_index, 0,
		static_cast<int32_t>(root_layout::kContentPageCount - 1U));
	const bool settings_mode = model.subpage_visible || active_index > 0;
	apply_status_bar_left_content(settings_mode);

	if (_status_bar_fade_overlay != nullptr) {
		lv_obj_set_style_bg_opa(_status_bar_fade_overlay, LV_OPA_TRANSP,
					0);
		lv_obj_add_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_HIDDEN);
	}
}

void RootChromeView::apply_status_bar_left_content(bool settings_mode)
{
	if (settings_mode) {
		for (lv_obj_t *icon_obj : _status_icon_objs) {
			if (icon_obj != nullptr) {
				lv_obj_add_flag(icon_obj, LV_OBJ_FLAG_HIDDEN);
			}
		}
		if (_status_title_label != nullptr) {
			lv_obj_clear_flag(_status_title_label,
					  LV_OBJ_FLAG_HIDDEN);
		}
		return;
	}

	if (_status_title_label != nullptr) {
		lv_obj_add_flag(_status_title_label, LV_OBJ_FLAG_HIDDEN);
	}

	for (std::size_t index = 0; index < _status_icon_objs.size(); ++index) {
		apply_status_icon_visual(index);
	}
	reflow_status_icons();
}

void RootChromeView::update_main_time()
{
	if (_time_label == nullptr) {
		return;
	}

	std::time_t now = std::time(nullptr);
	std::tm local_tm {};
#if defined(_WIN32)
	localtime_s(&local_tm, &now);
#else
	::tzset();
	localtime_r(&now, &local_tm);
#endif

	char time_buf[6] = {};
	std::snprintf(time_buf, sizeof(time_buf), "%02d:%02d", local_tm.tm_hour,
		      local_tm.tm_min);
	lv_label_set_text(_time_label, time_buf);
}

void RootChromeView::apply_status_icon_visual(std::size_t index)
{
	if (index >= _status_icon_objs.size()) {
		return;
	}

	lv_obj_t *icon_obj = _status_icon_objs[index];
	if (icon_obj == nullptr) {
		return;
	}

	if (_status_icon_active[index]) {
		lv_obj_clear_flag(icon_obj, LV_OBJ_FLAG_HIDDEN);
	} else {
		lv_obj_add_flag(icon_obj, LV_OBJ_FLAG_HIDDEN);
	}
}

void RootChromeView::reflow_status_icons()
{
	if (_status_bar_obj == nullptr) {
		return;
	}

	static constexpr int32_t kStatusBarHeight = 40;
	static constexpr int32_t kIconGap = 10;
	int32_t icon_x = 0;
	for (std::size_t index = 0; index < _status_icon_objs.size(); ++index) {
		if (!_status_icon_active[index]) {
			continue;
		}
		lv_obj_t *icon_obj = _status_icon_objs[index];
		if (icon_obj == nullptr) {
			continue;
		}

		const auto *icon_dsc = static_cast<const lv_image_dsc_t *>(
			lv_image_get_src(icon_obj));
		if (icon_dsc == nullptr) {
			continue;
		}
		const int32_t icon_width =
			static_cast<int32_t>(icon_dsc->header.w);
		const int32_t icon_height =
			static_cast<int32_t>(icon_dsc->header.h);
		lv_obj_set_pos(icon_obj, icon_x,
			       (kStatusBarHeight - icon_height) / 2);
		icon_x += icon_width + kIconGap;
	}
}

void RootChromeView::apply_status_bar_fade(float progress)
{
	if (_status_bar_fade_overlay == nullptr) {
		return;
	}

	const float clamped_progress = std::clamp(progress, 0.0F, 1.0F);
	const auto state = status_bar_transition::build_left_transition_state(
		clamped_progress);
	const lv_opa_t opacity = static_cast<lv_opa_t>(state.overlay_opa);
	if (opacity > LV_OPA_TRANSP) {
		lv_obj_clear_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_HIDDEN);
	} else {
		lv_obj_add_flag(_status_bar_fade_overlay, LV_OBJ_FLAG_HIDDEN);
	}

	lv_obj_set_style_bg_opa(_status_bar_fade_overlay, opacity, 0);
	lv_obj_invalidate(_status_bar_fade_overlay);
	apply_status_bar_left_content(
		state.content ==
		status_bar_transition::LeftContent::SettingsTitle);
}

void RootChromeView::clock_timer_cb(lv_timer_t *timer)
{
	if (timer == nullptr) {
		return;
	}

	auto *self =
		static_cast<RootChromeView *>(lv_timer_get_user_data(timer));
	if (self != nullptr) {
		self->update_main_time();
	}
}

} // namespace ui
