#pragma once

#include <cstdint>

#include <lvgl.h>

namespace ui
{

/**
 * @brief 组件基类，所有 UI 组件的抽象基类
 *
 * 提供基本的 LVGL 对象管理和位置设置功能
 */
class Component {
    public:
	explicit Component(lv_obj_t *parent);
	virtual ~Component();

	// 禁止拷贝和赋值
	Component(const Component &) = delete;
	Component &operator=(const Component &) = delete;

	// 允许移动
	Component(Component &&other) noexcept;
	Component &operator=(Component &&other) noexcept;

	/**
     * @brief 创建组件，纯虚函数由子类实现
     */
	virtual void create() = 0;

	/**
     * @brief 销毁组件
     */
	virtual void destroy();

	/**
     * @brief 设置组件位置
     * @param x X 坐标
     * @param y Y 坐标
     */
	virtual void set_position(int16_t x, int16_t y);

	/**
     * @brief 获取内部 LVGL 对象
     * @return lv_obj_t* 指向 LVGL 对象的指针
     */
	lv_obj_t *get_obj() const
	{
		return _obj;
	}

	/**
     * @brief 检查组件是否已创建
     * @return true 已创建
     * @return false 未创建
     */
	bool is_created() const
	{
		return _obj != nullptr;
	}

    protected:
	lv_obj_t *_parent; ///< 父对象
	lv_obj_t *_obj; ///< 本组件的 LVGL 对象
};

} // namespace ui
