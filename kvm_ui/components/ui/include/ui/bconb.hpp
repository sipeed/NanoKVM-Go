#pragma once

#include <functional>
#include <memory>

#include "ui/component.hpp"
#include "ui/control_state.hpp"

namespace ui
{

class PendingIndicator;

/**
 * @brief BConB - 大选择体组件 B (Big Control B)
 *
 * 大小: 100x64, 圆角: 32
 * 只包含居中图标，没有文字
 * 通常两个 BConB 横向组合使用
 */
class BConB
	: public Component
	, public ControlState {
    public:
	/**
     * @brief 组件状态
     */
	enum class State {
		DEFAULT, ///< 默认状态 (#44403B)
		ACTIVE, ///< 激活状态 (功能特定颜色)
		PRESSED ///< 按下状态 (当前颜色 * 0.8)
	};

	/**
     * @brief 点击回调函数类型
     */
	using ClickCallback = std::function<void(BConB *)>;

	/**
     * @brief 构造函数
     * @param parent 父对象
     * @param active_color 激活状态颜色 (HEX)
     */
	BConB(lv_obj_t *parent, uint32_t active_color);

	~BConB() override;

	/**
     * @brief 创建组件
     */
	void create() override;
	void destroy() override;

	/**
     * @brief 设置状态
     * @param state 新状态
     */
	void set_state(State state);

	/**
     * @brief 获取当前状态
     * @return State 当前状态
     */
	State get_state() const
	{
		return _state;
	}

	void
	set_visual_state(VisualState state,
			 ApplyPolicy policy = ApplyPolicy::DEFERRED) override;
	VisualState get_visual_state() const override;
	void set_pending(bool pending) override;
	void set_pending_timeout(uint32_t timeout_ms) override;
	void set_pending_timeout_enabled(bool enabled);
	bool is_pending() const override;

	/**
     * @brief 设置点击回调
     * @param callback 回调函数
     */
	void on_click(ClickCallback callback);

	/**
     * @brief 设置图标图片
     * @param src 指向 lv_image_dsc_t 的指针
	 */
	void set_image(const lv_image_dsc_t *src);
	void set_symbol(const char *symbol);
	void set_symbol_font(const lv_font_t *font);
	void set_image_rotation(int32_t angle);
	void set_icon_offset(int16_t x, int16_t y);

	// 常量定义
	static constexpr uint16_t WIDTH = 100;
	static constexpr uint16_t HEIGHT = 64;
	static constexpr uint16_t RADIUS = 32;
	static constexpr uint32_t COLOR_DEFAULT = 0x44403B;
	static constexpr uint8_t ICON_SIZE = 30;
	static constexpr uint8_t ICON_SIZE_MAX = 60;
	static constexpr uint32_t PENDING_TIMEOUT_MS = 10000;

    private:
	uint32_t _active_color;
	uint32_t _current_color;
	State _state;
	State _target_state;
	bool _state_dirty;
	bool _apply_scheduled;
	bool _is_pressed;
	uint32_t _pending_timeout_ms = PENDING_TIMEOUT_MS;
	bool _pending_timeout_enabled = true;
	ClickCallback _click_callback;

	lv_obj_t *_icon_obj;
	lv_obj_t *_image_obj = nullptr;
	lv_obj_t *_symbol_label = nullptr;
	bool _has_image = false;
	bool _has_symbol = false;
	std::unique_ptr<PendingIndicator> _pending_indicator;

	/**
     * @brief 应用按下状态的颜色叠加 (乘以 0.8)
     * @param base_color 基础颜色
     * @return uint32_t 叠加后的颜色
     */
	static uint32_t apply_press_overlay(uint32_t base_color);

	/**
     * @brief 更新外观以匹配当前状态
     */
	void update_appearance();
	void apply_pending_state();
	void schedule_deferred_apply();
	void cancel_deferred_apply();
	uint32_t pending_arc_color() const;
	void update_pending_arc_color();
	void apply_icon_content_visibility();

	// LVGL 事件回调
	static void press_event_cb(lv_event_t *e);
	static void release_event_cb(lv_event_t *e);
	static void click_event_cb(lv_event_t *e);
	static void async_apply_cb(void *user_data);
};

} // namespace ui
