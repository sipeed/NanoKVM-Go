#include "ui/component.hpp"

#include "lv_safe.hpp"

namespace ui
{

Component::Component(lv_obj_t *parent)
	: _parent(parent)
	, _obj(nullptr)
{
}

Component::~Component()
{
	destroy();
}

Component::Component(Component &&other) noexcept
	: _parent(other._parent)
	, _obj(other._obj)
{
	other._obj = nullptr;
	other._parent = nullptr;
}

Component &Component::operator=(Component &&other) noexcept
{
	if (this != &other) {
		destroy();
		_parent = other._parent;
		_obj = other._obj;
		other._obj = nullptr;
		other._parent = nullptr;
	}
	return *this;
}

void Component::destroy()
{
	lv_delete_if_valid(_obj);
}

void Component::set_position(int16_t x, int16_t y)
{
	if (_obj != nullptr) {
		lv_obj_set_pos(_obj, x, y);
	}
}

} // namespace ui
