#include "lvgl_touch_diagnostics.hpp"

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <string>

#include <lvgl.h>

namespace
{

constexpr int32_t kTouchCalibrationDotSize = 5;
constexpr std::size_t kTouchCalibrationMaxDots = 240;
constexpr uint32_t kTouchCalibrationBackgroundColor = 0xFFFFFF;
constexpr uint32_t kTouchCalibrationGridColor = 0xE0E0E0;
constexpr uint32_t kTouchCalibrationTargetColor = 0x5F6368;
constexpr uint32_t kTouchCalibrationTargetCenterColor = 0x202124;
constexpr uint32_t kTouchCalibrationCrosshairColor = 0x0057D9;
constexpr uint32_t kTouchCalibrationButtonFillColor = 0xF1F3F4;
constexpr int32_t kTouchCalibrationCoordinateButtonWidth = 112;
constexpr int32_t kTouchCalibrationCoordinateButtonHeight = 32;
constexpr int32_t kTouchPaintCellSize = 8;
constexpr uint32_t kTouchPaintColor = 0x111111;
constexpr uint32_t kTouchPaintLabelColor = 0x5F6368;

int32_t touch_paint_columns(int32_t width)
{
	return (width + kTouchPaintCellSize - 1) / kTouchPaintCellSize;
}

int32_t touch_paint_rows(int32_t height)
{
	return (height + kTouchPaintCellSize - 1) / kTouchPaintCellSize;
}

} // namespace

TouchCalibrationBoard::TouchCalibrationBoard(lv_obj_t *screen,
					      TouchDiagnosticsGeometry geometry)
	: _screen(screen)
	, _geometry(geometry)
{
}

void TouchCalibrationBoard::create()
{
	if (_screen == nullptr) {
		return;
	}

	lv_obj_set_style_bg_color(
		_screen, lv_color_hex(kTouchCalibrationBackgroundColor),
		0);
	lv_obj_set_style_bg_opa(_screen, LV_OPA_COVER, 0);
	lv_obj_remove_flag(_screen, LV_OBJ_FLAG_SCROLLABLE);

	_canvas = lv_obj_create(_screen);
	lv_obj_set_pos(_canvas, 0, 0);
	lv_obj_set_size(_canvas, _geometry.width, _geometry.height);
	lv_obj_set_style_bg_color(
		_canvas, lv_color_hex(kTouchCalibrationBackgroundColor),
		0);
	lv_obj_set_style_bg_opa(_canvas, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_canvas, 0, 0);
	lv_obj_set_style_radius(_canvas, 0, 0);
	lv_obj_remove_flag(_canvas, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_canvas, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(_canvas, draw_event_cb, LV_EVENT_PRESSED,
			    this);
	lv_obj_add_event_cb(_canvas, draw_event_cb, LV_EVENT_PRESSING,
			    this);

	create_grid();
	create_crosshair();
	create_labels();
}

void TouchCalibrationBoard::draw_event_cb(lv_event_t *event)
{
	auto *self = static_cast<TouchCalibrationBoard *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	lv_indev_t *indev = lv_indev_get_act();
	if (indev == nullptr) {
		return;
	}

	lv_point_t point {};
	lv_indev_get_point(indev, &point);
	self->draw_at(point);
}

void TouchCalibrationBoard::clear_event_cb(lv_event_t *event)
{
	auto *self = static_cast<TouchCalibrationBoard *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	self->clear_trace();
}

void TouchCalibrationBoard::style_plain_rect(lv_obj_t *obj, uint32_t color,
					     lv_opa_t opa)
{
	lv_obj_set_style_bg_color(obj, lv_color_hex(color), 0);
	lv_obj_set_style_bg_opa(obj, opa, 0);
	lv_obj_set_style_border_width(obj, 0, 0);
	lv_obj_set_style_radius(obj, 0, 0);
	lv_obj_remove_flag(obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(obj, LV_OBJ_FLAG_CLICKABLE);
}

lv_obj_t *TouchCalibrationBoard::create_line(lv_obj_t *parent, int32_t x,
					      int32_t y, int32_t width,
					      int32_t height, uint32_t color,
					      lv_opa_t opa)
{
	auto *line = lv_obj_create(parent);
	lv_obj_set_pos(line, x, y);
	lv_obj_set_size(line, width, height);
	style_plain_rect(line, color, opa);
	return line;
}

void TouchCalibrationBoard::create_grid()
{
	if (_canvas == nullptr) {
		return;
	}

	for (int32_t x = 45; x < _geometry.width; x += 45) {
		create_line(_canvas, x, 0, 1, _geometry.height,
			    kTouchCalibrationGridColor, LV_OPA_COVER);
	}
	for (int32_t y = 40; y < _geometry.height; y += 40) {
		create_line(_canvas, 0, y, _geometry.width, 1,
			    kTouchCalibrationGridColor, LV_OPA_COVER);
	}

	create_target(15, 15);
	create_target(_geometry.width - 16, 15);
	create_target(_geometry.width / 2, _geometry.height / 2);
	create_target(15, _geometry.height - 16);
	create_target(_geometry.width - 16, _geometry.height - 16);
}

void TouchCalibrationBoard::create_target(int32_t x, int32_t y)
{
	constexpr int32_t arm = 9;
	create_line(_canvas, x - arm, y, arm * 2 + 1, 1,
		    kTouchCalibrationTargetColor, LV_OPA_COVER);
	create_line(_canvas, x, y - arm, 1, arm * 2 + 1,
		    kTouchCalibrationTargetColor, LV_OPA_COVER);
	create_line(_canvas, x - 2, y - 2, 5, 5,
		    kTouchCalibrationTargetCenterColor, LV_OPA_COVER);
}

void TouchCalibrationBoard::create_crosshair()
{
	_cross_x = create_line(_canvas, 0, 0, _geometry.width, 1,
			       kTouchCalibrationCrosshairColor, LV_OPA_70);
	_cross_y = create_line(_canvas, 0, 0, 1, _geometry.height,
			       kTouchCalibrationCrosshairColor, LV_OPA_70);
	lv_obj_add_flag(_cross_x, LV_OBJ_FLAG_HIDDEN);
	lv_obj_add_flag(_cross_y, LV_OBJ_FLAG_HIDDEN);
}

void TouchCalibrationBoard::enable_overlay_drawing(lv_obj_t *obj)
{
	lv_obj_add_flag(obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(obj, draw_event_cb, LV_EVENT_PRESSED, this);
	lv_obj_add_event_cb(obj, draw_event_cb, LV_EVENT_PRESSING, this);
}

void TouchCalibrationBoard::create_coordinate_button()
{
	auto *button = lv_obj_create(_screen);
	lv_obj_set_pos(button,
		       (_geometry.width -
			kTouchCalibrationCoordinateButtonWidth) /
			       2,
		       _geometry.height / 4 -
			       kTouchCalibrationCoordinateButtonHeight / 2);
	lv_obj_set_size(button, kTouchCalibrationCoordinateButtonWidth,
			kTouchCalibrationCoordinateButtonHeight);
	lv_obj_set_style_bg_color(
		button, lv_color_hex(kTouchCalibrationButtonFillColor), 0);
	lv_obj_set_style_bg_opa(button, LV_OPA_COVER, 0);
	lv_obj_set_style_border_color(button, lv_color_hex(0x0057D9), 0);
	lv_obj_set_style_border_width(button, 1, 0);
	lv_obj_set_style_radius(button, 8, 0);
	lv_obj_set_style_pad_all(button, 0, 0);
	lv_obj_remove_flag(button, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(button, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(button, clear_event_cb, LV_EVENT_CLICKED, this);

	_coordinate_label = lv_label_create(button);
	lv_label_set_text(_coordinate_label, "(---,---)");
	lv_obj_set_style_text_color(_coordinate_label,
				    lv_color_hex(0x0057D9), 0);
	lv_obj_set_style_text_font(_coordinate_label,
				   &lv_font_montserrat_20, 0);
	lv_obj_center(_coordinate_label);
}

void TouchCalibrationBoard::create_labels()
{
	_title_label = lv_label_create(_screen);
	lv_label_set_text(_title_label, "Touch Cal");
	lv_obj_set_style_text_color(_title_label, lv_color_hex(0x3C4043), 0);
	lv_obj_set_pos(_title_label, 8, _geometry.height - 29);
	enable_overlay_drawing(_title_label);

	create_coordinate_button();

	_hint_label = lv_label_create(_screen);
	lv_label_set_text(_hint_label, "Draw to verify touch mapping");
	lv_obj_set_style_text_color(_hint_label, lv_color_hex(0x5F6368), 0);
	lv_obj_set_pos(_hint_label, 8, _geometry.height - 17);
	enable_overlay_drawing(_hint_label);
}

void TouchCalibrationBoard::draw_at(const lv_point_t &raw_point)
{
	if (_canvas == nullptr) {
		return;
	}

	const lv_point_t point {
		.x = static_cast<lv_coord_t>(std::clamp<int32_t>(
			raw_point.x, 0, _geometry.width - 1)),
		.y = static_cast<lv_coord_t>(std::clamp<int32_t>(
			raw_point.y, 0, _geometry.height - 1)),
	};

	move_crosshair(point);

	const int32_t dx = _has_last_draw_point ?
				   point.x - _last_draw_point.x :
				   999;
	const int32_t dy = _has_last_draw_point ?
				   point.y - _last_draw_point.y :
				   999;
	if (!_has_last_draw_point || dx * dx + dy * dy >= 9) {
		add_dot(point);
	}

	update_coordinate_label(raw_point);
}

void TouchCalibrationBoard::move_crosshair(const lv_point_t &point)
{
	if (_cross_x == nullptr || _cross_y == nullptr) {
		return;
	}

	lv_obj_set_y(_cross_x, point.y);
	lv_obj_set_x(_cross_y, point.x);
	lv_obj_clear_flag(_cross_x, LV_OBJ_FLAG_HIDDEN);
	lv_obj_clear_flag(_cross_y, LV_OBJ_FLAG_HIDDEN);
}

void TouchCalibrationBoard::add_dot(const lv_point_t &point)
{
	const uint32_t color = (_point_count % 3U == 0U) ? 0x0057D9 :
			       (_point_count % 3U == 1U) ? 0x0B8043 :
							   0xC2185B;
	auto *dot = lv_obj_create(_canvas);
	lv_obj_set_size(dot, kTouchCalibrationDotSize,
			kTouchCalibrationDotSize);
	lv_obj_set_pos(dot, point.x - kTouchCalibrationDotSize / 2,
		       point.y - kTouchCalibrationDotSize / 2);
	style_plain_rect(dot, color, LV_OPA_COVER);
	lv_obj_set_style_radius(dot, kTouchCalibrationDotSize / 2, 0);

	_dots.push_back(dot);
	while (_dots.size() > kTouchCalibrationMaxDots) {
		lv_obj_delete(_dots.front());
		_dots.pop_front();
	}

	++_point_count;
	_last_draw_point = point;
	_has_last_draw_point = true;
}

void TouchCalibrationBoard::update_coordinate_label(const lv_point_t &point)
{
	if (_coordinate_label == nullptr) {
		return;
	}

	std::string text = "(" + std::to_string(point.x) + "," +
			   std::to_string(point.y) + ")";
	lv_label_set_text(_coordinate_label, text.c_str());
	lv_obj_center(_coordinate_label);
}

void TouchCalibrationBoard::clear_trace()
{
	for (lv_obj_t *dot : _dots) {
		lv_obj_delete(dot);
	}
	_dots.clear();
	_point_count = 0;
	_has_last_draw_point = false;
	if (_cross_x != nullptr) {
		lv_obj_add_flag(_cross_x, LV_OBJ_FLAG_HIDDEN);
	}
	if (_cross_y != nullptr) {
		lv_obj_add_flag(_cross_y, LV_OBJ_FLAG_HIDDEN);
	}
	if (_coordinate_label != nullptr) {
		lv_label_set_text(_coordinate_label, "(---,---)");
		lv_obj_center(_coordinate_label);
	}
}

TouchPaintCoverageBoard::TouchPaintCoverageBoard(lv_obj_t *screen,
						 TouchDiagnosticsGeometry geometry)
	: _screen(screen)
	, _geometry(geometry)
{
	const int32_t columns = touch_paint_columns(geometry.width);
	const int32_t rows = touch_paint_rows(geometry.height);
	_paint_cells.assign(
		static_cast<std::size_t>(columns) *
			static_cast<std::size_t>(rows),
		nullptr);
}

void TouchPaintCoverageBoard::create()
{
	if (_screen == nullptr) {
		return;
	}

	lv_obj_set_style_bg_color(
		_screen, lv_color_hex(kTouchCalibrationBackgroundColor),
		0);
	lv_obj_set_style_bg_opa(_screen, LV_OPA_COVER, 0);
	lv_obj_remove_flag(_screen, LV_OBJ_FLAG_SCROLLABLE);

	_canvas = lv_obj_create(_screen);
	lv_obj_set_pos(_canvas, 0, 0);
	lv_obj_set_size(_canvas, _geometry.width, _geometry.height);
	lv_obj_set_style_bg_color(
		_canvas, lv_color_hex(kTouchCalibrationBackgroundColor),
		0);
	lv_obj_set_style_bg_opa(_canvas, LV_OPA_COVER, 0);
	lv_obj_set_style_border_width(_canvas, 0, 0);
	lv_obj_set_style_radius(_canvas, 0, 0);
	lv_obj_remove_flag(_canvas, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_canvas, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(_canvas, draw_event_cb, LV_EVENT_PRESSED,
			    this);
	lv_obj_add_event_cb(_canvas, draw_event_cb, LV_EVENT_PRESSING,
			    this);

	create_labels();
}

void TouchPaintCoverageBoard::draw_event_cb(lv_event_t *event)
{
	auto *self = static_cast<TouchPaintCoverageBoard *>(
		lv_event_get_user_data(event));
	if (self == nullptr) {
		return;
	}

	lv_indev_t *indev = lv_indev_get_act();
	if (indev == nullptr) {
		return;
	}

	lv_point_t point {};
	lv_indev_get_point(indev, &point);
	if (lv_event_get_code(event) == LV_EVENT_PRESSED) {
		self->begin_stroke();
	}
	self->paint_at(point);
}

void TouchPaintCoverageBoard::style_plain_rect(lv_obj_t *obj, uint32_t color,
					       lv_opa_t opa)
{
	lv_obj_set_style_bg_color(obj, lv_color_hex(color), 0);
	lv_obj_set_style_bg_opa(obj, opa, 0);
	lv_obj_set_style_border_width(obj, 0, 0);
	lv_obj_set_style_radius(obj, 0, 0);
	lv_obj_remove_flag(obj, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(obj, LV_OBJ_FLAG_CLICKABLE);
}

void TouchPaintCoverageBoard::create_labels()
{
	_title_label = lv_label_create(_screen);
	lv_label_set_text(_title_label, "Touch Paint Test");
	lv_obj_set_style_text_color(
		_title_label, lv_color_hex(kTouchPaintLabelColor), 0);
	lv_obj_set_pos(_title_label, 6, 5);
	enable_overlay_drawing(_title_label);

	_bounds_label = lv_label_create(_screen);
	lv_label_set_text(_bounds_label, "Paint to reveal touch range");
	lv_obj_set_style_text_color(
		_bounds_label, lv_color_hex(kTouchPaintLabelColor), 0);
	lv_obj_set_pos(_bounds_label, 6, _geometry.height - 19);
	enable_overlay_drawing(_bounds_label);
}

void TouchPaintCoverageBoard::enable_overlay_drawing(lv_obj_t *obj)
{
	lv_obj_add_flag(obj, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_add_event_cb(obj, draw_event_cb, LV_EVENT_PRESSED, this);
	lv_obj_add_event_cb(obj, draw_event_cb, LV_EVENT_PRESSING, this);
}

void TouchPaintCoverageBoard::begin_stroke()
{
	_has_last_point = false;
}

void TouchPaintCoverageBoard::paint_at(const lv_point_t &raw_point)
{
	if (_canvas == nullptr) {
		return;
	}

	const lv_point_t point {
		.x = static_cast<lv_coord_t>(std::clamp<int32_t>(
			raw_point.x, 0, _geometry.width - 1)),
		.y = static_cast<lv_coord_t>(std::clamp<int32_t>(
			raw_point.y, 0, _geometry.height - 1)),
	};

	if (_has_last_point) {
		paint_segment(_last_point, point);
	} else {
		paint_cell(point);
	}
	_last_point = point;
	_has_last_point = true;

	update_bounds(raw_point);
	update_bounds_label();
}

void TouchPaintCoverageBoard::paint_segment(const lv_point_t &from,
					    const lv_point_t &to)
{
	const int32_t dx = to.x - from.x;
	const int32_t dy = to.y - from.y;
	const int32_t distance = std::max(std::abs(dx), std::abs(dy));
	const int32_t steps = std::max(1, distance / 2);
	for (int32_t i = 0; i <= steps; ++i) {
		const lv_point_t point {
			.x = static_cast<lv_coord_t>(from.x +
						     dx * i / steps),
			.y = static_cast<lv_coord_t>(from.y +
						     dy * i / steps),
		};
		paint_cell(point);
	}
}

void TouchPaintCoverageBoard::paint_cell(const lv_point_t &point)
{
	const int32_t columns = touch_paint_columns(_geometry.width);
	const int32_t rows = touch_paint_rows(_geometry.height);
	const int32_t cell_x =
		std::clamp<int32_t>(point.x / kTouchPaintCellSize, 0,
				    columns - 1);
	const int32_t cell_y = std::clamp<int32_t>(
		point.y / kTouchPaintCellSize, 0, rows - 1);
	const std::size_t index =
		static_cast<std::size_t>(cell_y) *
			static_cast<std::size_t>(columns) +
		static_cast<std::size_t>(cell_x);
	if (index >= _paint_cells.size() ||
	    _paint_cells[index] != nullptr) {
		return;
	}

	const int32_t x = cell_x * kTouchPaintCellSize;
	const int32_t y = cell_y * kTouchPaintCellSize;
	auto *mark = lv_obj_create(_canvas);
	lv_obj_set_pos(mark, x, y);
	lv_obj_set_size(mark,
			std::min(kTouchPaintCellSize,
				 _geometry.width - x),
			std::min(kTouchPaintCellSize,
				 _geometry.height - y));
	style_plain_rect(mark, kTouchPaintColor, LV_OPA_COVER);

	_paint_cells[index] = mark;
	++_painted_cell_count;
}

void TouchPaintCoverageBoard::update_bounds(const lv_point_t &point)
{
	if (!_has_bounds) {
		_min_x = point.x;
		_max_x = point.x;
		_min_y = point.y;
		_max_y = point.y;
		_has_bounds = true;
		return;
	}

	_min_x = std::min<int32_t>(_min_x, point.x);
	_max_x = std::max<int32_t>(_max_x, point.x);
	_min_y = std::min<int32_t>(_min_y, point.y);
	_max_y = std::max<int32_t>(_max_y, point.y);
}

void TouchPaintCoverageBoard::update_bounds_label()
{
	if (_bounds_label == nullptr || !_has_bounds) {
		return;
	}

	const std::size_t total_cells = _paint_cells.size();
	const std::size_t percent =
		total_cells == 0 ?
			0 :
			(_painted_cell_count * 100U) / total_cells;
	std::string text = "x " + std::to_string(_min_x) + ".." +
			   std::to_string(_max_x) + " y " +
			   std::to_string(_min_y) + ".." +
			   std::to_string(_max_y) + " " +
			   std::to_string(percent) + "%";
	lv_label_set_text(_bounds_label, text.c_str());
}
