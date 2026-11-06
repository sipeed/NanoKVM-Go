#include "ui/root_pager.hpp"

#include <algorithm>
#include <cmath>
#include <memory>
#include <utility>

#include "lv_safe.hpp"
#include "ui/root_chrome_view.hpp"

namespace ui
{

namespace
{

constexpr uint32_t kRootBackgroundColor = 0x000000;

} // namespace

RootPager::RootPager(TransitionMode transition_mode)
	: _transition_mode(transition_mode)
	, _chrome(std::make_unique<RootChromeView>(transition_mode ==
						   TransitionMode::Slide))
{
}

RootPager::~RootPager()
{
	shutdown();
}

void RootPager::create(lv_obj_t *screen)
{
	if (_is_shutting_down || _main_page_obj != nullptr ||
	    screen == nullptr) {
		return;
	}

	create_root_container(screen);
	_chrome->create(_main_page_obj);
	create_settings_host();
	set_scroll_position(ContentPage::Main, LV_ANIM_OFF);
}

void RootPager::begin_shutdown()
{
	if (_is_shutting_down) {
		return;
	}

	_is_shutting_down = true;
	_content_page_provider = nullptr;
	lv_anim_delete(this, nullptr);
	cleanup_flip_transition();
	_flip_animating = false;
	if (_chrome != nullptr) {
		_chrome->begin_shutdown();
	}
}

void RootPager::shutdown()
{
	begin_shutdown();
	if (_chrome != nullptr) {
		_chrome->shutdown();
	}

	lv_delete_if_valid(_settings_host);
	lv_delete_if_valid(_scroll_container);
	lv_delete_if_valid(_main_page_obj);

	_main_page_obj = nullptr;
	_scroll_container = nullptr;
	_settings_host = nullptr;
	_subpage_visible = false;
}

bool RootPager::is_created() const noexcept
{
	return _main_page_obj != nullptr;
}

bool RootPager::is_flip_animating() const noexcept
{
	return _flip_animating;
}

bool RootPager::is_subpage_visible() const noexcept
{
	return _subpage_visible;
}

bool RootPager::is_main_visible() const
{
	if (!is_created() || _scroll_container == nullptr) {
		return false;
	}
	return !_subpage_visible && lv_obj_get_scroll_x(_scroll_container) <
					    root_layout::kContentPageStep / 2;
}

bool RootPager::is_settings_visible() const
{
	return is_created() && (_subpage_visible || !is_main_visible());
}

lv_obj_t *RootPager::content_parent() const noexcept
{
	return _scroll_container;
}

lv_obj_t *RootPager::settings_host() const noexcept
{
	return _settings_host;
}

RootPager::ContentPage RootPager::current_content_page() const noexcept
{
	return _current_content_page;
}

RootPager::ContentPage RootPager::nearest_content_page() const
{
	if (_scroll_container == nullptr) {
		return _current_content_page;
	}

	const int32_t current_x = lv_obj_get_scroll_x(_scroll_container);
	const int32_t page_index = std::clamp(
		(current_x + root_layout::kContentPageStep / 2) /
			root_layout::kContentPageStep,
		0, static_cast<int32_t>(root_layout::kContentPageCount - 1U));
	return content_page_from_index(page_index);
}

void RootPager::set_content_page_provider(ContentPageProvider provider)
{
	if (_is_shutting_down) {
		return;
	}
	_content_page_provider = std::move(provider);
}

void RootPager::transition_to(ContentPage page)
{
	if (_scroll_container == nullptr || _is_shutting_down) {
		return;
	}

	set_root_scroll_enabled(true);
	if (_transition_mode == TransitionMode::Flip) {
		if (page == _current_content_page) {
			set_scroll_position(page, LV_ANIM_OFF);
			return;
		}
		if (start_flip_transition(page)) {
			return;
		}
		set_scroll_position(page, LV_ANIM_OFF);
		return;
	}

	set_scroll_position(page, LV_ANIM_ON);
}

void RootPager::snap_to_nearest()
{
	if (!is_created() || _scroll_container == nullptr ||
	    _is_shutting_down) {
		return;
	}
	if (_flip_animating) {
		lv_obj_scroll_to_x(_scroll_container,
				   content_page_x(_current_content_page),
				   LV_ANIM_OFF);
		return;
	}
	if (_transition_mode == TransitionMode::Flip) {
		lv_obj_scroll_to_x(_scroll_container,
				   content_page_x(_current_content_page),
				   LV_ANIM_OFF);
		return;
	}

	const int32_t current_x = lv_obj_get_scroll_x(_scroll_container);
	int32_t page_index = content_page_index(_current_content_page);
	int32_t page_delta = 0;
	if (resolve_swipe_page_delta(page_delta)) {
		page_index += page_delta;
	} else {
		const int32_t page_offset =
			current_x - content_page_x(_current_content_page);
		if (page_offset > kPageSwipeTriggerDistance) {
			++page_index;
		} else if (page_offset < -kPageSwipeTriggerDistance) {
			--page_index;
		}
	}
	page_index = std::clamp(
		page_index, 0,
		static_cast<int32_t>(root_layout::kContentPageCount - 1U));
	const ContentPage target_page = content_page_from_index(page_index);
	_current_content_page = target_page;
	render_navigation_chrome();
	if (std::abs(current_x - content_page_x(target_page)) <= 1) {
		return;
	}
	lv_obj_scroll_to_x(_scroll_container, content_page_x(target_page),
			   LV_ANIM_ON);
}

void RootPager::enter_settings_subpage()
{
	if (_is_shutting_down || _settings_host == nullptr) {
		return;
	}

	_subpage_visible = true;
	set_root_scroll_enabled(false);
	if (_scroll_container != nullptr) {
		lv_obj_add_flag(_scroll_container, LV_OBJ_FLAG_HIDDEN);
	}
	lv_obj_clear_flag(_settings_host, LV_OBJ_FLAG_HIDDEN);
}

void RootPager::leave_settings_subpage()
{
	if (_is_shutting_down || _settings_host == nullptr) {
		return;
	}

	_subpage_visible = false;
	set_root_scroll_enabled(true);
	lv_obj_add_flag(_settings_host, LV_OBJ_FLAG_HIDDEN);
	if (_scroll_container != nullptr) {
		lv_obj_clear_flag(_scroll_container, LV_OBJ_FLAG_HIDDEN);
	}
}

void RootPager::refresh_navigation_chrome()
{
	if (_is_shutting_down) {
		return;
	}
	render_navigation_chrome();
}

void RootPager::set_status_icon_active(StatusIcon icon, bool active)
{
	if (_chrome == nullptr) {
		return;
	}
	_chrome->set_status_icon_active(icon, active);
	render_navigation_chrome();
}

void RootPager::create_root_container(lv_obj_t *screen)
{
	_main_page_obj = lv_obj_create(screen);
	lv_obj_set_size(_main_page_obj, root_layout::kScreenWidth,
			root_layout::kScreenHeight);
	lv_obj_set_pos(_main_page_obj, 0, 0);
	lv_obj_set_style_bg_color(_main_page_obj,
				  lv_color_hex(kRootBackgroundColor), 0);
	lv_obj_set_style_bg_opa(_main_page_obj, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_main_page_obj, 0, 0);
	lv_obj_set_style_radius(_main_page_obj, 0, 0);
	lv_obj_set_style_pad_all(_main_page_obj, 0, 0);
	lv_obj_set_scrollbar_mode(_main_page_obj, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_main_page_obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_main_page_obj, LV_OBJ_FLAG_CLICKABLE);

	_scroll_container = lv_obj_create(_main_page_obj);
	lv_obj_set_size(_scroll_container, root_layout::kScreenWidth,
			root_layout::kContentCardHeight);
	lv_obj_set_style_bg_opa(_scroll_container, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_scroll_container, 0, 0);
	lv_obj_set_style_pad_all(_scroll_container, 0, 0);
	lv_obj_set_pos(_scroll_container, 0, root_layout::kContentAreaY);
	lv_obj_set_scroll_dir(_scroll_container,
			      _transition_mode == TransitionMode::Slide ?
				      LV_DIR_HOR :
				      LV_DIR_NONE);
	lv_obj_set_scroll_snap_x(_scroll_container, LV_SCROLL_SNAP_NONE);
	lv_obj_set_scroll_snap_y(_scroll_container, LV_SCROLL_SNAP_NONE);
	lv_obj_add_flag(_scroll_container, LV_OBJ_FLAG_SCROLL_ONE);
	lv_obj_clear_flag(_scroll_container, LV_OBJ_FLAG_SCROLL_MOMENTUM);
	lv_obj_clear_flag(_scroll_container, LV_OBJ_FLAG_SCROLL_ELASTIC);
	lv_obj_set_scrollbar_mode(_scroll_container, LV_SCROLLBAR_MODE_OFF);
	lv_obj_add_event_cb(_scroll_container, scroll_begin_cb,
			    LV_EVENT_SCROLL_BEGIN, this);
	lv_obj_add_event_cb(_scroll_container, scroll_cb, LV_EVENT_SCROLL,
			    this);
	lv_obj_add_event_cb(_scroll_container, scroll_end_cb,
			    LV_EVENT_SCROLL_END, this);
}

void RootPager::create_settings_host()
{
	_settings_host = lv_obj_create(_main_page_obj);
	lv_obj_set_size(_settings_host, root_layout::kScreenWidth,
			root_layout::kScreenHeight);
	lv_obj_set_pos(_settings_host, 0, 0);
	lv_obj_set_style_bg_color(_settings_host,
				  lv_color_hex(kRootBackgroundColor), 0);
	lv_obj_set_style_bg_opa(_settings_host, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_settings_host, 0, 0);
	lv_obj_set_style_pad_all(_settings_host, 0, 0);
	lv_obj_set_scrollbar_mode(_settings_host, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_settings_host, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_settings_host, LV_OBJ_FLAG_HIDDEN);
}

void RootPager::set_root_scroll_enabled(bool enabled)
{
	if (_scroll_container == nullptr) {
		return;
	}
	if (!enabled || _transition_mode != TransitionMode::Slide ||
	    _flip_animating) {
		reset_swipe_tracking();
	}
	lv_obj_set_scroll_dir(
		_scroll_container,
		enabled && _transition_mode == TransitionMode::Slide &&
				!_flip_animating ?
			LV_DIR_HOR :
			LV_DIR_NONE);
}

void RootPager::set_scroll_position(ContentPage page,
				    lv_anim_enable_t anim_enable)
{
	reset_swipe_tracking();
	if (_scroll_container == nullptr) {
		return;
	}

	_current_content_page = page;
	render_navigation_chrome();
	lv_obj_scroll_to_x(_scroll_container, content_page_x(page),
			   anim_enable);
}

void RootPager::reset_swipe_tracking()
{
	_swipe_tracking = {};
}

void RootPager::begin_swipe_tracking()
{
	reset_swipe_tracking();
	if (!is_created() || _scroll_container == nullptr ||
	    _is_shutting_down || _transition_mode != TransitionMode::Slide ||
	    _subpage_visible) {
		return;
	}

	_swipe_tracking.active = true;
	lv_indev_t *indev = lv_indev_active();
	if (indev == nullptr) {
		return;
	}

	lv_point_t point {};
	lv_indev_get_point(indev, &point);
	_swipe_tracking.start_point = point;
	_swipe_tracking.latest_point = point;
	_swipe_tracking.has_start_point = true;
	_swipe_tracking.has_latest_point = true;

	const lv_dir_t gesture_dir = lv_indev_get_gesture_dir(indev);
	if (gesture_dir == LV_DIR_LEFT || gesture_dir == LV_DIR_RIGHT) {
		_swipe_tracking.gesture_dir = gesture_dir;
	}

	lv_point_t vect {};
	lv_indev_get_vect(indev, &vect);
	if (vect.x != 0) {
		_swipe_tracking.accumulated_dx += static_cast<int32_t>(vect.x);
		_swipe_tracking.has_accumulated_dx = true;
	}
}

void RootPager::update_swipe_tracking()
{
	if (!_swipe_tracking.active) {
		return;
	}
	if (!is_created() || _scroll_container == nullptr ||
	    _is_shutting_down || _transition_mode != TransitionMode::Slide ||
	    _subpage_visible) {
		reset_swipe_tracking();
		return;
	}

	lv_indev_t *indev = lv_indev_active();
	if (indev == nullptr) {
		return;
	}

	lv_point_t point {};
	lv_indev_get_point(indev, &point);
	_swipe_tracking.latest_point = point;
	_swipe_tracking.has_latest_point = true;

	const lv_dir_t gesture_dir = lv_indev_get_gesture_dir(indev);
	if (gesture_dir == LV_DIR_LEFT || gesture_dir == LV_DIR_RIGHT) {
		_swipe_tracking.gesture_dir = gesture_dir;
	}

	lv_point_t vect {};
	lv_indev_get_vect(indev, &vect);
	if (vect.x != 0) {
		_swipe_tracking.accumulated_dx += static_cast<int32_t>(vect.x);
		_swipe_tracking.has_accumulated_dx = true;
	}
}

bool RootPager::resolve_swipe_page_delta(int32_t &page_delta) const
{
	page_delta = 0;
	if (!_swipe_tracking.active) {
		return false;
	}

	int32_t dx = 0;
	bool has_dx = false;
	if (_swipe_tracking.has_start_point &&
	    _swipe_tracking.has_latest_point) {
		dx = static_cast<int32_t>(_swipe_tracking.latest_point.x) -
		     static_cast<int32_t>(_swipe_tracking.start_point.x);
		has_dx = true;
	}
	if (_swipe_tracking.has_accumulated_dx &&
	    (!has_dx ||
	     std::abs(_swipe_tracking.accumulated_dx) > std::abs(dx))) {
		dx = _swipe_tracking.accumulated_dx;
		has_dx = true;
	}

	if (has_dx) {
		if (dx <= -kPageSwipeTriggerDistance) {
			page_delta = 1;
			return true;
		}
		if (dx >= kPageSwipeTriggerDistance) {
			page_delta = -1;
			return true;
		}
	}

	if (_swipe_tracking.gesture_dir == LV_DIR_LEFT) {
		page_delta = 1;
		return true;
	}
	if (_swipe_tracking.gesture_dir == LV_DIR_RIGHT) {
		page_delta = -1;
		return true;
	}

	return false;
}

bool RootPager::start_flip_transition(ContentPage page)
{
	lv_obj_t *from_page = content_page_object(_current_content_page);
	lv_obj_t *to_page = content_page_object(page);
	lv_obj_t *top_layer = lv_layer_top();
	if (from_page == nullptr || to_page == nullptr ||
	    top_layer == nullptr) {
		return false;
	}

	_flip_from_snapshot =
		lv_snapshot_take(from_page, LV_COLOR_FORMAT_RGB565);
	_flip_to_snapshot = lv_snapshot_take(to_page, LV_COLOR_FORMAT_RGB565);
	if (_flip_from_snapshot == nullptr || _flip_to_snapshot == nullptr) {
		cleanup_flip_transition();
		return false;
	}

	_flip_overlay = lv_obj_create(top_layer);
	if (_flip_overlay == nullptr) {
		cleanup_flip_transition();
		return false;
	}
	lv_obj_set_size(_flip_overlay, kRootFlipOverlayWidth,
			kRootFlipOverlayHeight);
	lv_obj_set_pos(_flip_overlay, 0, 0);
	lv_obj_set_style_bg_opa(_flip_overlay, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_flip_overlay, 0, 0);
	lv_obj_set_style_pad_all(_flip_overlay, 0, 0);
	lv_obj_set_scrollbar_mode(_flip_overlay, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_flip_overlay, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_flip_overlay, LV_OBJ_FLAG_CLICKABLE);

	_flip_image = lv_image_create(_flip_overlay);
	if (_flip_image == nullptr) {
		cleanup_flip_transition();
		return false;
	}
	lv_image_set_src(_flip_image, _flip_from_snapshot);
	lv_image_set_pivot(_flip_image, root_layout::kContentCardWidth / 2,
			   root_layout::kContentCardHeight / 2);
	lv_image_set_scale_x(_flip_image, LV_SCALE_NONE);
	lv_obj_set_pos(_flip_image, 0, root_layout::kContentAreaY);
	lv_obj_clear_flag(_flip_image, LV_OBJ_FLAG_CLICKABLE);

	_flip_target_page = page;
	_flip_showing_to_snapshot = false;
	_flip_animating = true;
	set_root_scroll_enabled(false);

	lv_anim_t anim {};
	lv_anim_init(&anim);
	lv_anim_set_var(&anim, this);
	lv_anim_set_custom_exec_cb(&anim, flip_anim_exec_cb);
	lv_anim_set_values(&anim, 0, 200);
	lv_anim_set_time(&anim, kRootFlipAnimationMs);
	lv_anim_set_path_cb(&anim, lv_anim_path_ease_in_out);
	lv_anim_set_ready_cb(&anim, flip_anim_ready_cb);
	lv_anim_set_user_data(&anim, this);
	if (lv_anim_start(&anim) == nullptr) {
		cleanup_flip_transition();
		_flip_animating = false;
		set_root_scroll_enabled(true);
		return false;
	}
	return true;
}

void RootPager::finish_flip_transition()
{
	if (!_flip_animating) {
		cleanup_flip_transition();
		return;
	}

	const ContentPage target_page = _flip_target_page;
	set_scroll_position(target_page, LV_ANIM_OFF);
	cleanup_flip_transition();
	_flip_animating = false;
	set_root_scroll_enabled(true);
}

void RootPager::cleanup_flip_transition()
{
	lv_delete_if_valid(_flip_overlay);
	_flip_overlay = nullptr;
	_flip_image = nullptr;

	if (_flip_from_snapshot != nullptr) {
		lv_draw_buf_destroy(_flip_from_snapshot);
		_flip_from_snapshot = nullptr;
	}
	if (_flip_to_snapshot != nullptr) {
		lv_draw_buf_destroy(_flip_to_snapshot);
		_flip_to_snapshot = nullptr;
	}
	_flip_showing_to_snapshot = false;
}

lv_obj_t *RootPager::content_page_object(ContentPage page) const
{
	return _content_page_provider ? _content_page_provider(page) : nullptr;
}

void RootPager::render_navigation_chrome()
{
	if (_chrome == nullptr) {
		return;
	}

	_chrome->render_navigation({ .active_page_index = content_page_index(
					     _current_content_page),
				     .subpage_visible = _subpage_visible });
}

void RootPager::begin_status_bar_fade_from_scroll()
{
	if (_chrome == nullptr || _scroll_container == nullptr ||
	    _transition_mode != TransitionMode::Slide || _subpage_visible) {
		return;
	}

	const int32_t scroll_x = lv_obj_get_scroll_x(_scroll_container);
	const int32_t clamped_x =
		std::clamp(scroll_x, 0, root_layout::kContentPageStep);
	const float progress =
		static_cast<float>(clamped_x) /
		static_cast<float>(root_layout::kContentPageStep);
	_chrome->begin_scroll_fade(progress);
}

void RootPager::update_status_bar_fade_from_scroll()
{
	if (_chrome == nullptr || _scroll_container == nullptr ||
	    _transition_mode != TransitionMode::Slide || _subpage_visible) {
		return;
	}

	const int32_t scroll_x = lv_obj_get_scroll_x(_scroll_container);
	const int32_t clamped_x =
		std::clamp(scroll_x, 0, root_layout::kContentPageStep);
	const float progress =
		static_cast<float>(clamped_x) /
		static_cast<float>(root_layout::kContentPageStep);
	_chrome->update_scroll_fade(progress);
}

int32_t RootPager::content_page_x(ContentPage page) noexcept
{
	return root_layout::content_page_x(page);
}

int32_t RootPager::content_page_index(ContentPage page) noexcept
{
	return root_layout::content_page_index(page);
}

RootPager::ContentPage
RootPager::content_page_from_index(int32_t index) noexcept
{
	return root_layout::content_page_from_index(index);
}

void RootPager::flip_anim_exec_cb(lv_anim_t *anim, int32_t value)
{
	auto *self = static_cast<RootPager *>(lv_anim_get_user_data(anim));
	if (self == nullptr || self->_flip_image == nullptr ||
	    !lv_obj_is_valid(self->_flip_image)) {
		return;
	}

	const bool showing_to_snapshot = value >= 100;
	if (showing_to_snapshot != self->_flip_showing_to_snapshot) {
		lv_image_set_src(self->_flip_image,
				 showing_to_snapshot ?
					 self->_flip_to_snapshot :
					 self->_flip_from_snapshot);
		self->_flip_showing_to_snapshot = showing_to_snapshot;
	}

	const int32_t distance_from_center = showing_to_snapshot ? value - 100 :
								   100 - value;
	const int32_t scale_x =
		std::max(1, distance_from_center *
				    static_cast<int32_t>(LV_SCALE_NONE) / 100);
	lv_image_set_scale_x(self->_flip_image, static_cast<uint32_t>(scale_x));
}

void RootPager::flip_anim_ready_cb(lv_anim_t *anim)
{
	auto *self = static_cast<RootPager *>(lv_anim_get_user_data(anim));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}
	self->finish_flip_transition();
}

void RootPager::scroll_begin_cb(lv_event_t *event)
{
	auto *self = static_cast<RootPager *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}

	self->begin_swipe_tracking();
	self->begin_status_bar_fade_from_scroll();

	auto *anim = static_cast<lv_anim_t *>(lv_event_get_param(event));
	if (anim == nullptr) {
		return;
	}
	lv_anim_set_duration(anim, kRootSnapAnimationMs);
	lv_anim_set_path_cb(anim, lv_anim_path_ease_out);
}

void RootPager::scroll_cb(lv_event_t *event)
{
	auto *self = static_cast<RootPager *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}

	self->update_swipe_tracking();
	self->update_status_bar_fade_from_scroll();
}

void RootPager::scroll_end_cb(lv_event_t *event)
{
	auto *self = static_cast<RootPager *>(lv_event_get_user_data(event));
	if (self == nullptr || self->_is_shutting_down) {
		return;
	}

	self->update_swipe_tracking();
	self->snap_to_nearest();
	self->reset_swipe_tracking();
	if (self->_chrome != nullptr) {
		self->_chrome->end_scroll_fade(
			{ .active_page_index = content_page_index(
				  self->_current_content_page),
			  .subpage_visible = self->_subpage_visible });
	}
}

} // namespace ui
