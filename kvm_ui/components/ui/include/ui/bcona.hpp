#pragma once

#include <functional>
#include <memory>
#include <string>

#include "ui/component.hpp"
#include "ui/control_state.hpp"

namespace ui
{

class PendingIndicator;

/**
 * @brief BConA - 大选择体组件 (Big Control A)
 *
 * 大小: 230x64, 圆角: 32
 * 通常包含左侧图标和右侧文字
 * 可作为独立按钮或子页面入口按钮
 */
class BConA
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
	using ClickCallback = std::function<void(BConA *)>;

	/**
     * @brief 构造函数
     * @param parent 父对象
     * @param text 显示文字
     * @param active_color 激活状态颜色 (HEX)
     * @param has_subpage 是否有子页面
     */
	BConA(lv_obj_t *parent, const char *text, uint32_t active_color,
	      bool has_subpage = false);

	~BConA() override;

	/**
     * @brief 创建组件
     */
	void create() override;
	void destroy() override;

	/**
     * @brief 设置文字
     * @param text 新的文字内容
	 */
	void set_text(const char *text);
	void set_text_font(const lv_font_t *font);
	void set_text_align(lv_text_align_t align);

	/**
	 * @brief 限制文字视口，并让超出视口的文字左右往返滚动
	 * @param width 文字视口宽度
	 */
	void set_text_scrolling(uint16_t width);

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

	/**
     * @brief 设置激活颜色
     * @param color 新的激活颜色 (HEX)
     */
	void set_color(uint32_t color);
	void set_default_color(uint32_t color);

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
     * @brief 设置左侧图标符号（会替换默认白色方块）
     * @param symbol LVGL 符号字符串，如 LV_SYMBOL_WIFI
     */
	void set_left_symbol(const char *symbol);

	/**
     * @brief 设置左侧图标容器的背景样式
     * @param color 背景颜色 (HEX)
     * @param radius 圆角半径
     */
	void set_left_icon_bg(uint32_t color, uint8_t radius);

	/**
     * @brief 设置右侧图标符号
     * @param symbol LVGL 符号字符串，如 LV_SYMBOL_REFRESH
     */
	void set_right_symbol(const char *symbol);

	/**
	 * @brief 设置左侧图标图片（会替换默认白色方块和符号）
	 * @param src 指向 lv_image_dsc_t 的指针
	 */
	void set_left_image(const lv_image_dsc_t *src);

	/**
	 * @brief 设置左侧图片的期望可见性，不改变图标容器的布局
	 *
	 * pending 状态会临时隐藏图片；结束 pending 后恢复本接口记录的
	 * 期望可见性，而不是恢复进入 pending 前的过期快照。
	 */
	void set_left_image_visible(bool visible);
	bool is_left_image_visible() const;

	// 常量定义
	static constexpr uint16_t WIDTH = 230;
	static constexpr uint16_t HEIGHT = 64;
	static constexpr uint16_t RADIUS = 32;
	static constexpr uint8_t FONT_SIZE = 30;
	static constexpr uint32_t COLOR_DEFAULT = 0x44403B;
	static constexpr uint8_t ICON_SIZE = 30;
	static constexpr uint8_t ICON_SIZE_MAX = 60;
	static constexpr uint8_t ICON_RADIUS = 4;
	static constexpr uint8_t PADDING_LEFT = 20;
	static constexpr uint8_t PADDING_COLUMN = 15;
	static constexpr uint32_t PENDING_TIMEOUT_MS = 10000;

    private:
	std::string _text;
	uint32_t _active_color;
	uint32_t _default_color;
	uint32_t _current_color;
	State _state;
	State _target_state;
	bool _state_dirty;
	bool _apply_scheduled;
	bool _has_subpage;
	bool _is_pressed;
	bool _left_image_has_source = false;
	bool _left_image_visible = false;
	bool _pending_icon_label_was_visible = false;
	uint32_t _pending_timeout_ms = PENDING_TIMEOUT_MS;
	bool _pending_timeout_enabled = true;
	ClickCallback _click_callback;

	lv_obj_t *_icon_obj;
	lv_obj_t *_icon_inner_label = nullptr;
	lv_obj_t *_image_obj = nullptr;
	lv_obj_t *_label_obj;
	lv_obj_t *_left_icon_label = nullptr;
	lv_obj_t *_right_icon_label = nullptr;
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
	void apply_left_image_visibility();
	void apply_pending_content_visibility(bool visible);

	// LVGL 事件回调
	static void press_event_cb(lv_event_t *e);
	static void release_event_cb(lv_event_t *e);
	static void click_event_cb(lv_event_t *e);
	static void async_apply_cb(void *user_data);
};

} // namespace ui
