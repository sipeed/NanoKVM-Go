#pragma once

#include <lvgl.h>

namespace ui
{

inline void lv_delete_if_valid(lv_obj_t *&obj)
{
	if (obj == nullptr) {
		return;
	}

	if (lv_obj_is_valid(obj)) {
		lv_obj_delete(obj);
	}

	obj = nullptr;
}

} // namespace ui
