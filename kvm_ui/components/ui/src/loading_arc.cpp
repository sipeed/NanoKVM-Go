#include "loading_arc.hpp"

namespace ui
{

LoadingArc::LoadingArc(lv_obj_t *parent)
	: _parent(parent)
{
}

LoadingArc::~LoadingArc()
{
	destroy();
}

void LoadingArc::create()
{
	if (_parent == nullptr || _arc != nullptr) {
		return;
	}

	_arc = lv_arc_create(_parent);
	lv_obj_set_size(_arc, SIZE, SIZE);
	lv_obj_center(_arc);
	lv_obj_clear_flag(_arc, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(_arc, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_remove_style(_arc, nullptr, LV_PART_KNOB);
	lv_obj_set_style_arc_opa(_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_bg_opa(_arc, LV_OPA_TRANSP, LV_PART_MAIN);
	lv_obj_set_style_border_width(_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_pad_all(_arc, 0, LV_PART_MAIN);
	lv_obj_set_style_arc_width(_arc, ARC_WIDTH, LV_PART_INDICATOR);
	lv_obj_set_style_arc_rounded(_arc, true, LV_PART_INDICATOR);
	lv_obj_set_style_arc_color(_arc, lv_color_hex(COLOR),
				   LV_PART_INDICATOR);
	lv_arc_set_bg_angles(_arc, 0, 360);
	lv_arc_set_angles(_arc, 0, SWEEP_DEG);
	lv_arc_set_rotation(_arc, 0);
	lv_obj_add_flag(_arc, LV_OBJ_FLAG_HIDDEN);
}

void LoadingArc::set_running(bool running)
{
	if (running == _running) {
		return;
	}
	if (running && (_arc == nullptr || !lv_obj_is_valid(_arc))) {
		return;
	}

	_running = running;
	if (_running) {
		_rotation = 0;
		lv_arc_set_rotation(_arc, _rotation);
		lv_obj_clear_flag(_arc, LV_OBJ_FLAG_HIDDEN);
		start_timer();
		return;
	}

	stop_timer();
	lv_obj_add_flag(_arc, LV_OBJ_FLAG_HIDDEN);
}

void LoadingArc::destroy()
{
	stop_timer();
	_running = false;
	// The arc object is owned by the parent LVGL tree; only drop the
	// non-owning handles so a late timer or caller cannot touch it.
	_arc = nullptr;
	_parent = nullptr;
}

void LoadingArc::start_timer()
{
	stop_timer();
	_spin_timer = lv_timer_create(spin_timer_cb, PERIOD_MS, this);
}

void LoadingArc::stop_timer()
{
	if (_spin_timer != nullptr) {
		lv_timer_delete(_spin_timer);
		_spin_timer = nullptr;
	}
}

void LoadingArc::spin_timer_cb(lv_timer_t *timer)
{
	auto *self = static_cast<LoadingArc *>(lv_timer_get_user_data(timer));
	if (self == nullptr || !self->_running || self->_arc == nullptr ||
	    !lv_obj_is_valid(self->_arc)) {
		return;
	}

	self->_rotation = static_cast<uint16_t>(
		(self->_rotation + ROTATION_STEP_DEG) % 360);
	lv_arc_set_rotation(self->_arc, self->_rotation);
}

} // namespace ui
