#pragma once

#include <cstddef>
#include <cstdint>
#include <string_view>

#include <lvgl.h>

namespace test_support
{

class LvglTestContext {
    public:
	LvglTestContext()
	{
		lv_init();

		_display = lv_display_create(270, 240);
		if (_display != nullptr) {
			if (lv_timer_t *refresh_timer =
				    lv_display_get_refr_timer(_display);
			    refresh_timer != nullptr) {
				lv_timer_pause(refresh_timer);
			}
		}

		_pointer = lv_indev_create();
		if (_pointer != nullptr) {
			lv_indev_set_type(_pointer, LV_INDEV_TYPE_POINTER);
			lv_indev_set_display(_pointer, _display);
			lv_indev_set_user_data(_pointer, this);
			lv_indev_set_read_cb(_pointer, read_cb);
			lv_indev_set_mode(_pointer, LV_INDEV_MODE_EVENT);
			pause_pointer_timer();
		}
	}

	~LvglTestContext()
	{
		if (_pointer != nullptr) {
			lv_indev_delete(_pointer);
			_pointer = nullptr;
		}
		if (_display != nullptr) {
			lv_display_delete(_display);
			_display = nullptr;
		}
		lv_deinit();
	}

	LvglTestContext(const LvglTestContext &) = delete;
	LvglTestContext &operator=(const LvglTestContext &) = delete;

	lv_display_t *display() const
	{
		return _display;
	}

	lv_indev_t *pointer() const
	{
		return _pointer;
	}

	void delete_pointer()
	{
		if (_pointer == nullptr) {
			return;
		}

		lv_indev_t *pointer = _pointer;
		_pointer = nullptr;
		_state = LV_INDEV_STATE_RELEASED;
		lv_indev_delete(pointer);
	}

	lv_obj_t *screen() const
	{
		return _display == nullptr ?
			       nullptr :
			       lv_display_get_screen_active(_display);
	}

	void press(lv_obj_t *obj)
	{
		if (obj == nullptr) {
			return;
		}

		if (lv_obj_t *active_screen = screen();
		    active_screen != nullptr) {
			lv_obj_update_layout(active_screen);
		}
		lv_area_t coordinates {};
		lv_obj_get_coords(obj, &coordinates);
		press_at((coordinates.x1 + coordinates.x2) / 2,
			 (coordinates.y1 + coordinates.y2) / 2);
	}

	void press_at(int32_t x, int32_t y)
	{
		if (_pointer == nullptr) {
			return;
		}

		if (lv_obj_t *active_screen = screen();
		    active_screen != nullptr) {
			lv_obj_update_layout(active_screen);
		}
		_point.x = x;
		_point.y = y;
		_state = LV_INDEV_STATE_RELEASED;
		read_pointer();
		_state = LV_INDEV_STATE_PRESSED;
		read_pointer();
	}

	void move_while_pressed(int32_t x, int32_t y)
	{
		_point.x = x;
		_point.y = y;
		_state = LV_INDEV_STATE_PRESSED;
		read_pointer();
	}

	void release()
	{
		_state = LV_INDEV_STATE_RELEASED;
		read_pointer();
	}

	void advance(uint32_t ms)
	{
		lv_tick_inc(ms);
		if (_state == LV_INDEV_STATE_PRESSED) {
			read_pointer();
		}
		lv_anim_refr_now();
		lv_timer_handler();
		if (lv_obj_t *active_screen = screen();
		    active_screen != nullptr) {
			lv_obj_update_layout(active_screen);
		}
	}

	std::size_t timer_count() const
	{
		std::size_t count = 0;
		for (lv_timer_t *timer = lv_timer_get_next(nullptr);
		     timer != nullptr; timer = lv_timer_get_next(timer)) {
			++count;
		}
		return count;
	}

    private:
	static void read_cb(lv_indev_t *indev, lv_indev_data_t *data)
	{
		auto *context = static_cast<LvglTestContext *>(
			lv_indev_get_user_data(indev));
		if (context == nullptr) {
			return;
		}

		data->point = context->_point;
		data->state = context->_state;
		data->continue_reading = false;
	}

	void read_pointer()
	{
		if (_pointer == nullptr) {
			return;
		}
		lv_indev_read(_pointer);
		pause_pointer_timer();
	}

	void pause_pointer_timer()
	{
		if (_pointer == nullptr) {
			return;
		}
		if (lv_timer_t *read_timer = lv_indev_get_read_timer(_pointer);
		    read_timer != nullptr) {
			lv_timer_pause(read_timer);
		}
	}

	lv_display_t *_display = nullptr;
	lv_indev_t *_pointer = nullptr;
	lv_point_t _point {};
	lv_indev_state_t _state = LV_INDEV_STATE_RELEASED;
};

inline lv_obj_t *find_visible_direct_label(lv_obj_t *parent)
{
	if (parent == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    !lv_obj_has_flag(child, LV_OBJ_FLAG_HIDDEN)) {
			return child;
		}
	}

	return nullptr;
}

inline std::string_view visible_direct_text(lv_obj_t *parent)
{
	lv_obj_t *label = find_visible_direct_label(parent);
	return label == nullptr ? std::string_view {} :
				  lv_label_get_text(label);
}

inline lv_obj_t *find_visible_descendant_of_type(lv_obj_t *parent,
						 const lv_obj_class_t *type)
{
	if (parent == nullptr || type == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_has_flag(child, LV_OBJ_FLAG_HIDDEN)) {
			continue;
		}
		if (lv_obj_check_type(child, type)) {
			return child;
		}
		if (lv_obj_t *match =
			    find_visible_descendant_of_type(child, type);
		    match != nullptr) {
			return match;
		}
	}

	return nullptr;
}

} // namespace test_support
