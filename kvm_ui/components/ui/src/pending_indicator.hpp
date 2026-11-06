#pragma once

#include <cstdint>
#include <functional>

#include <lvgl.h>

namespace ui
{

/**
 * @brief BCon 系列控件共用的 pending 转圈指示器。
 *
 * 控件仍负责自己的图标显隐；本类只管理与原 BConA 一致的 arc、旋转和
 * 超时生命周期。
 */
class PendingIndicator {
    public:
	using ContentVisibilityCallback = std::function<void(bool visible)>;

	static constexpr uint32_t DEFAULT_TIMEOUT_MS = 10000;

	PendingIndicator(lv_obj_t *icon_parent,
			 ContentVisibilityCallback set_content_visible);
	~PendingIndicator();

	PendingIndicator(const PendingIndicator &) = delete;
	PendingIndicator &operator=(const PendingIndicator &) = delete;

	void set_pending(bool pending);
	bool is_pending() const;
	void set_timeout(uint32_t timeout_ms);
	void set_timeout_enabled(bool enabled);
	void set_arc_color(uint32_t color);
	void destroy();

    private:
	lv_obj_t *_icon_parent = nullptr;
	lv_obj_t *_arc_obj = nullptr;
	lv_timer_t *_spin_timer = nullptr;
	lv_timer_t *_timeout_timer = nullptr;
	ContentVisibilityCallback _set_content_visible;
	bool _pending = false;
	uint16_t _rotation = 0;
	uint32_t _timeout_ms = DEFAULT_TIMEOUT_MS;
	bool _timeout_enabled = true;
	uint32_t _arc_color = 0;

	void ensure_arc();
	void update_arc_color();
	void start_timers();
	void stop_timers();

	static void spin_timer_cb(lv_timer_t *timer);
	static void timeout_timer_cb(lv_timer_t *timer);
};

} // namespace ui
