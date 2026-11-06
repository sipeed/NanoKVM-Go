#pragma once

#include <string>

#include "ui/component.hpp"

namespace ui
{

/**
 * @brief SConA - 小选择体组件 (Small Control A)
 *
 * 大小: 230x32, 圆角: 12
 * 静态显示组件，不可点击
 * 支持单行文字（左/中/右对齐）和双列文字（左列+右列）
 */
class SConA : public Component {
    public:
	/**
     * @brief 文字对齐方式
     */
	enum class TextAlign {
		LEFT, ///<< 左对齐
		CENTER, ///<< 居中对齐
		RIGHT ///<< 右对齐
	};

	/**
     * @brief 构造函数
     * @param parent 父对象
     * @param text 显示文字
     * @param align 文字对齐方式 (默认居中)
     * @param text_font 可选的非拥有字体；nullptr 使用 Montserrat-20，提供时
     *                  必须在组件销毁前保持有效
     */
	SConA(lv_obj_t *parent, const char *text,
	      TextAlign align = TextAlign::CENTER,
	      const lv_font_t *text_font = nullptr);

	~SConA() override;

	/**
     * @brief 创建组件
     */
	void create() override;

	/**
	 * @brief 设置文字
	 * @param text 新的文字内容
	 */
	void set_text(const char *text);

	/**
	 * @brief 用借用的图片源替换单文本展示
	 *
	 * 图片源不由 SConA 持有；调用方必须在清除图片源后才释放它。成功后
	 * 单文本标签会隐藏，双列模式不支持该展示方式。
	 */
	bool set_image_source(const void *source);

	/**
	 * @brief 清除借用图片源并恢复文本展示
	 */
	void clear_image_source();

	bool has_image_source() const
	{
		return _image_source != nullptr;
	}

	/**
     * @brief 设置对齐方式
     * @param align 对齐方式
     */
	void set_align(TextAlign align);

	/**
     * @brief 设置双列文字（左文本 + 右文本）
     * @param left_text 左侧文本
     * @param right_text 右侧文本
     */
	void set_dual_text(const char *left_text, const char *right_text);

	/**
	 * @brief 清除双列模式并恢复单文本模式
	 */
	void clear_dual_text();

	/**
	 * @brief 启用或关闭单文本超长内容的循环滚动
	 *
	 * 默认关闭。双列模式始终保持原有静态布局。
	 */
	void set_single_text_scroll_circular(bool enabled);

	void destroy() override;

	/**
     * @brief 获取当前对齐方式
     * @return TextAlign 当前对齐方式
     */
	TextAlign get_align() const
	{
		return _align;
	}

	// 常量定义
	static constexpr uint16_t WIDTH = 230;
	static constexpr uint16_t HEIGHT = 32;
	static constexpr uint16_t RADIUS = 12;
	static constexpr uint8_t FONT_SIZE = 20;
	static constexpr uint32_t COLOR_DEFAULT = 0x44403B;

    protected:
	/// 宽度变体构造：复用 SConA 全部行为，仅尺寸宽度不同。
	SConA(lv_obj_t *parent, const char *text, TextAlign align,
	      const lv_font_t *text_font, uint16_t width);

    private:
	static constexpr uint16_t SINGLE_TEXT_HORIZONTAL_INSET = 12;

	std::string _text;
	std::string _dual_left_text;
	std::string _dual_right_text;
	const lv_font_t *_text_font;
	TextAlign _align;
	uint16_t _width;
	bool _dual_mode;
	bool _single_text_scroll_circular = false;

	lv_obj_t *_left_label_obj;
	lv_obj_t *_right_label_obj;
	lv_obj_t *_image_obj = nullptr;
	const void *_image_source = nullptr;

	/**
	 * @brief 更新标签对齐方式
	 */
	void update_alignment();
	void set_label_text_if_changed(lv_obj_t *label, const char *text);
	void apply_image_visibility();
};

} // namespace ui
