#pragma once

#include <cstddef>
#include <cstdint>
#include <deque>
#include <vector>

#include <lvgl.h>

// Private touch-diagnostics boards extracted from lvgl_ui.cpp.
//
// This header is internal to components/lvgl_main/src/ and is never installed
// or exposed through lvgl_main/include. UILVGLApp stack-allocates these
// boards inside run_touch_calibration()/run_touch_paint_test() and passes the
// physical panel geometry from its single source of truth; the diagnostic
// implementation never reproduces the 284x240 panel literals.

struct TouchDiagnosticsGeometry {
	int32_t width { 0 };
	int32_t height { 0 };
};

class TouchCalibrationBoard final {
    public:
	explicit TouchCalibrationBoard(lv_obj_t *screen,
				       TouchDiagnosticsGeometry geometry);

	void create();

    private:
	static void draw_event_cb(lv_event_t *event);
	static void clear_event_cb(lv_event_t *event);
	static void style_plain_rect(lv_obj_t *obj, uint32_t color,
				     lv_opa_t opa = LV_OPA_COVER);

	lv_obj_t *create_line(lv_obj_t *parent, int32_t x, int32_t y,
			      int32_t width, int32_t height, uint32_t color,
			      lv_opa_t opa = LV_OPA_COVER);
	void create_grid();
	void create_target(int32_t x, int32_t y);
	void create_crosshair();
	void enable_overlay_drawing(lv_obj_t *obj);
	void create_coordinate_button();
	void create_labels();
	void draw_at(const lv_point_t &raw_point);
	void move_crosshair(const lv_point_t &point);
	void add_dot(const lv_point_t &point);
	void update_coordinate_label(const lv_point_t &point);
	void clear_trace();

	lv_obj_t *_screen { nullptr };
	lv_obj_t *_canvas { nullptr };
	lv_obj_t *_title_label { nullptr };
	lv_obj_t *_coordinate_label { nullptr };
	lv_obj_t *_hint_label { nullptr };
	lv_obj_t *_cross_x { nullptr };
	lv_obj_t *_cross_y { nullptr };
	std::deque<lv_obj_t *> _dots;
	lv_point_t _last_draw_point {};
	bool _has_last_draw_point { false };
	std::size_t _point_count { 0 };
	TouchDiagnosticsGeometry _geometry {};
};

class TouchPaintCoverageBoard final {
    public:
	explicit TouchPaintCoverageBoard(lv_obj_t *screen,
					 TouchDiagnosticsGeometry geometry);

	void create();

    private:
	static void draw_event_cb(lv_event_t *event);
	static void style_plain_rect(lv_obj_t *obj, uint32_t color,
				     lv_opa_t opa = LV_OPA_COVER);

	void create_labels();
	void enable_overlay_drawing(lv_obj_t *obj);
	void begin_stroke();
	void paint_at(const lv_point_t &raw_point);
	void paint_segment(const lv_point_t &from, const lv_point_t &to);
	void paint_cell(const lv_point_t &point);
	void update_bounds(const lv_point_t &point);
	void update_bounds_label();

	lv_obj_t *_screen { nullptr };
	lv_obj_t *_canvas { nullptr };
	lv_obj_t *_title_label { nullptr };
	lv_obj_t *_bounds_label { nullptr };
	std::vector<lv_obj_t *> _paint_cells;
	lv_point_t _last_point {};
	bool _has_last_point { false };
	bool _has_bounds { false };
	int32_t _min_x { 0 };
	int32_t _max_x { 0 };
	int32_t _min_y { 0 };
	int32_t _max_y { 0 };
	std::size_t _painted_cell_count { 0 };
	TouchDiagnosticsGeometry _geometry {};
};
