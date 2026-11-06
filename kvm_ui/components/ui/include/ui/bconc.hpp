#pragma once

#include <chrono>
#include <functional>

#include "ui/component.hpp"

namespace ui
{

/**
 * @brief BConC - 大滑动条组件 (Big Control C)
 *
 * 视觉与 BConA 保持一致:
 * - 外壳尺寸: 230x64, 圆角: 32
 * - 左侧图标位置与 BConA 对齐
 * - 整体容器作为滑动交互区域
 */
class BConC : public Component {
    public:
	using ValueChangedCallback = std::function<void(int)>;

	BConC(lv_obj_t *parent, uint32_t fill_color);
	~BConC() override = default;

	void create() override;

	void set_left_image(const lv_image_dsc_t *src);

	void set_range(int min, int max);
	void set_value(int value);
	int get_value() const
	{
		return _value;
	}
	bool is_interacting() const
	{
		return _is_interacting;
	}
	bool is_interaction_guard_active() const;
	void clear_interaction_guard();

	void on_value_changed(ValueChangedCallback callback);

	static constexpr uint16_t WIDTH = 230;
	static constexpr uint16_t HEIGHT = 64;
	static constexpr uint16_t RADIUS = 32;
	static constexpr uint32_t COLOR_DEFAULT = 0x44403B;

    private:
	static constexpr uint8_t ICON_SIZE_MAX = 60;
	static constexpr int INTERACTION_GUARD_MS = 1000;

	int _min = 0;
	int _max = 100;
	int _value = 50;
	uint32_t _fill_color = 0xFFB900;
	bool _is_interacting = false;
	bool _has_interacted = false;
	std::chrono::steady_clock::time_point _last_interaction_end {};
	ValueChangedCallback _value_changed_callback = nullptr;

	lv_obj_t *_fill_obj = nullptr;
	lv_obj_t *_icon_obj = nullptr;
	lv_obj_t *_image_obj = nullptr;

	void update_visual();
	void update_from_touch(int32_t x);
	void begin_interaction();
	void end_interaction();
	static int clamp_value(int value, int min, int max);
	static void press_event_cb(lv_event_t *e);
	static void pressing_event_cb(lv_event_t *e);
	static void release_event_cb(lv_event_t *e);
};

} // namespace ui
