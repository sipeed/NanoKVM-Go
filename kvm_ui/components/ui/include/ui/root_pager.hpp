#pragma once

#include <cstdint>
#include <functional>
#include <memory>

#include <lvgl.h>

#include "ui/root_layout.hpp"
#include "ui/root_status_icon.hpp"

namespace ui
{

class RootChromeView;

class RootPager final {
    public:
	enum class TransitionMode : uint8_t {
		Slide,
		Flip,
	};

	using ContentPage = root_layout::ContentPage;
	using StatusIcon = RootStatusIcon;

	using ContentPageProvider = std::function<lv_obj_t *(ContentPage)>;

	explicit RootPager(TransitionMode transition_mode);
	~RootPager();

	RootPager(const RootPager &) = delete;
	RootPager &operator=(const RootPager &) = delete;

	void create(lv_obj_t *screen);
	void begin_shutdown();
	void shutdown();

	bool is_created() const noexcept;
	bool is_flip_animating() const noexcept;
	bool is_subpage_visible() const noexcept;
	bool is_main_visible() const;
	bool is_settings_visible() const;

	lv_obj_t *content_parent() const noexcept;
	lv_obj_t *settings_host() const noexcept;
	ContentPage current_content_page() const noexcept;
	ContentPage nearest_content_page() const;

	void set_content_page_provider(ContentPageProvider provider);
	void transition_to(ContentPage page);
	void enter_settings_subpage();
	void leave_settings_subpage();
	void refresh_navigation_chrome();

	void set_status_icon_active(StatusIcon icon, bool active);

    private:
	static constexpr int32_t kPageSwipeTriggerDistance = 40;
	static constexpr uint32_t kRootSnapAnimationMs = 240;
	static constexpr uint32_t kRootFlipAnimationMs = 280;
	static constexpr int32_t kRootFlipOverlayWidth = 284;
	static constexpr int32_t kRootFlipOverlayHeight = 240;

	struct SwipeTracking {
		bool active { false };
		bool has_start_point { false };
		bool has_latest_point { false };
		bool has_accumulated_dx { false };
		lv_point_t start_point {};
		lv_point_t latest_point {};
		int32_t accumulated_dx { 0 };
		lv_dir_t gesture_dir { LV_DIR_NONE };
	};

	const TransitionMode _transition_mode;
	std::unique_ptr<RootChromeView> _chrome;
	bool _is_shutting_down = false;
	bool _subpage_visible = false;
	ContentPage _current_content_page = ContentPage::Main;
	SwipeTracking _swipe_tracking {};
	ContentPageProvider _content_page_provider;
	lv_obj_t *_main_page_obj = nullptr;
	lv_obj_t *_scroll_container = nullptr;
	lv_obj_t *_settings_host = nullptr;
	lv_obj_t *_flip_overlay = nullptr;
	lv_obj_t *_flip_image = nullptr;
	lv_draw_buf_t *_flip_from_snapshot = nullptr;
	lv_draw_buf_t *_flip_to_snapshot = nullptr;
	bool _flip_animating = false;
	bool _flip_showing_to_snapshot = false;
	ContentPage _flip_target_page = ContentPage::Main;

	void create_root_container(lv_obj_t *screen);
	void create_settings_host();
	void set_root_scroll_enabled(bool enabled);
	void set_scroll_position(ContentPage page,
				 lv_anim_enable_t anim_enable);
	void snap_to_nearest();
	void reset_swipe_tracking();
	void begin_swipe_tracking();
	void update_swipe_tracking();
	bool resolve_swipe_page_delta(int32_t &page_delta) const;
	bool start_flip_transition(ContentPage page);
	void finish_flip_transition();
	void cleanup_flip_transition();
	lv_obj_t *content_page_object(ContentPage page) const;
	void render_navigation_chrome();
	void begin_status_bar_fade_from_scroll();
	void update_status_bar_fade_from_scroll();

	static int32_t content_page_x(ContentPage page) noexcept;
	static int32_t content_page_index(ContentPage page) noexcept;
	static ContentPage content_page_from_index(int32_t index) noexcept;
	static void flip_anim_exec_cb(lv_anim_t *anim, int32_t value);
	static void flip_anim_ready_cb(lv_anim_t *anim);
	static void scroll_begin_cb(lv_event_t *event);
	static void scroll_cb(lv_event_t *event);
	static void scroll_end_cb(lv_event_t *event);
};

} // namespace ui
