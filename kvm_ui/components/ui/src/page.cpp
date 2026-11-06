#include "ui/page.hpp"

#include "lv_safe.hpp"

namespace ui
{

Page::Page(PageType type)
	: _type(type)
{
}

void Page::destroy()
{
	lv_delete_if_valid(_container);
}

} // namespace ui
