#pragma once

#include "ui/scona.hpp"

namespace ui
{

/**
 * @brief SConAWide - SConA 的满宽变体 (270x32)
 *
 * 复用 SConA 全部行为，仅尺寸宽度从 230 扩展到 270（满 270 视觉画布）。
 * 圆角、背景、字体、对齐、单/双列文本、循环滚动与借用图片源能力均与
 * SConA 一致；用于 More 子页面等需要更长可见宽度的只读文本行。
 */
class SConAWide : public SConA {
    public:
	static constexpr uint16_t WIDTH = 270;

	/**
     * @brief 构造函数（参数语义与 SConA 一致，宽度固定为 270）
     */
	SConAWide(lv_obj_t *parent, const char *text,
		  TextAlign align = TextAlign::CENTER,
		  const lv_font_t *text_font = nullptr)
		: SConA(parent, text, align, text_font, WIDTH)
	{
	}
};

} // namespace ui
