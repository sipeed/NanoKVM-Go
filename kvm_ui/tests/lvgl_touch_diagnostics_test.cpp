#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "lvgl_touch_diagnostics.hpp"

namespace
{

// Physical panel geometry passed from the single source of truth (lvgl_ui.cpp
// owns the real kPanelPhysicalWidth/Height). The diagnostic module must not
// reproduce these literals.
constexpr int32_t kPhysicalWidth = 284;
constexpr int32_t kPhysicalHeight = 240;
constexpr TouchDiagnosticsGeometry kPhysicalGeometry { kPhysicalWidth,
							kPhysicalHeight };

bool is_label(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_check_type(obj, &lv_label_class);
}

// Find a label anywhere under root whose text contains substr.
lv_obj_t *find_label_containing(lv_obj_t *root, std::string_view substr)
{
	if (root == nullptr) {
		return nullptr;
	}
	if (is_label(root)) {
		std::string_view text(lv_label_get_text(root));
		if (text.find(substr) != std::string_view::npos) {
			return root;
		}
	}
	const uint32_t count = lv_obj_get_child_count(root);
	for (uint32_t index = 0; index < count; ++index) {
		if (lv_obj_t *found = find_label_containing(
			    lv_obj_get_child(root, index), substr);
		    found != nullptr) {
			return found;
		}
	}
	return nullptr;
}

lv_obj_t *first_canvas(lv_obj_t *screen)
{
	if (screen == nullptr) {
		return nullptr;
	}
	const uint32_t count = lv_obj_get_child_count(screen);
	for (uint32_t index = 0; index < count; ++index) {
		lv_obj_t *child = lv_obj_get_child(screen, index);
		if (child != nullptr && !is_label(child)) {
			return child;
		}
	}
	return nullptr;
}

lv_obj_t *parent_of_label_with(lv_obj_t *root, std::string_view substr)
{
	lv_obj_t *label = find_label_containing(root, substr);
	return label == nullptr ? nullptr : lv_obj_get_parent(label);
}

} // namespace

TEST_CASE("Touch calibration board geometry and null-safe create",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;

	SECTION("create with a null screen is a no-op")
	{
		TouchCalibrationBoard board(nullptr, kPhysicalGeometry);
		board.create();
		// No crash; the active screen is untouched.
		CHECK(ctx.screen() != nullptr);
		CHECK(lv_obj_get_child_count(ctx.screen()) == 0);
	}

	SECTION("create builds a full physical-area canvas")
	{
		TouchCalibrationBoard board(ctx.screen(), kPhysicalGeometry);
		board.create();
		lv_obj_update_layout(ctx.screen());

		lv_obj_t *canvas = first_canvas(ctx.screen());
		REQUIRE(canvas != nullptr);
		CHECK(lv_obj_get_width(canvas) == kPhysicalWidth);
		CHECK(lv_obj_get_height(canvas) == kPhysicalHeight);
		CHECK(lv_obj_get_style_bg_opa(canvas, 0) == LV_OPA_COVER);
		CHECK(!lv_obj_has_flag(canvas, LV_OBJ_FLAG_SCROLLABLE));
		CHECK(lv_obj_has_flag(canvas, LV_OBJ_FLAG_CLICKABLE));
	}
}

TEST_CASE("Calibration reacts to pointer with a dot and coordinate label",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	TouchCalibrationBoard board(ctx.screen(), kPhysicalGeometry);
	board.create();
	lv_obj_update_layout(ctx.screen());

	lv_obj_t *canvas = first_canvas(ctx.screen());
	REQUIRE(canvas != nullptr);
	REQUIRE(find_label_containing(ctx.screen(), "(---,---)") != nullptr);

	const uint32_t canvas_children_before = lv_obj_get_child_count(canvas);

	ctx.press(canvas);
	ctx.advance(40);

	// A dot was added to the canvas.
	CHECK(lv_obj_get_child_count(canvas) == canvas_children_before + 1);
	// The coordinate label no longer shows the placeholder.
	CHECK(find_label_containing(ctx.screen(), "(---,---)") == nullptr);

	ctx.release();
	ctx.advance(10);
}

TEST_CASE("Calibration caps retained dots and evicts oldest",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	TouchCalibrationBoard board(ctx.screen(), kPhysicalGeometry);
	board.create();
	lv_obj_update_layout(ctx.screen());

	lv_obj_t *canvas = first_canvas(ctx.screen());
	REQUIRE(canvas != nullptr);
	// Fixed canvas children: grid lines, five targets, and the crosshair.
	const uint32_t fixed_children = lv_obj_get_child_count(canvas);

	ctx.press(canvas);
	ctx.advance(10);

	// Move to many distinct, well-separated points so each press adds a dot.
	// 300 guaranteed-jump moves far exceeds the 240-dot cap.
	for (int i = 0; i < 300; ++i) {
		const int32_t x = (i * 53) % kPhysicalWidth;
		const int32_t y = (i * 71 + 37) % kPhysicalHeight;
		ctx.move_while_pressed(x, y);
		ctx.advance(5);
	}

	const uint32_t after = lv_obj_get_child_count(canvas);
	// Dots were added (eviction path exercised) and capped at 240.
	CHECK(after > fixed_children);
	CHECK(after <= fixed_children + 240);

	ctx.release();
	ctx.advance(10);
}

TEST_CASE("Calibration clear button resets dots and the label",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	TouchCalibrationBoard board(ctx.screen(), kPhysicalGeometry);
	board.create();
	lv_obj_update_layout(ctx.screen());

	lv_obj_t *canvas = first_canvas(ctx.screen());
	REQUIRE(canvas != nullptr);
	const uint32_t fixed_children = lv_obj_get_child_count(canvas);

	ctx.press(canvas);
	ctx.advance(10);
	ctx.move_while_pressed(80, 60);
	ctx.advance(10);
	ctx.move_while_pressed(200, 180);
	ctx.advance(10);
	CHECK(lv_obj_get_child_count(canvas) > fixed_children);
	CHECK(find_label_containing(ctx.screen(), "(---,---)") == nullptr);

	// The clear button owns the coordinate label.
	lv_obj_t *clear_button = parent_of_label_with(ctx.screen(), "(");
	REQUIRE(clear_button != nullptr);

	ctx.press(clear_button);
	ctx.advance(10);
	ctx.release();
	ctx.advance(10);

	// Dots removed; placeholder restored.
	CHECK(lv_obj_get_child_count(canvas) == fixed_children);
	REQUIRE(find_label_containing(ctx.screen(), "(---,---)") != nullptr);
}

TEST_CASE("Paint coverage board paints cells and reports bounds with percent",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	TouchPaintCoverageBoard board(ctx.screen(), kPhysicalGeometry);
	board.create();
	lv_obj_update_layout(ctx.screen());

	lv_obj_t *canvas = first_canvas(ctx.screen());
	REQUIRE(canvas != nullptr);
	CHECK(lv_obj_get_width(canvas) == kPhysicalWidth);
	REQUIRE(find_label_containing(ctx.screen(), "Paint to reveal") !=
		nullptr);

	const uint32_t canvas_children_before = lv_obj_get_child_count(canvas);

	ctx.press(canvas);
	ctx.advance(10);
	ctx.move_while_pressed(40, 40);
	ctx.advance(10);
	ctx.move_while_pressed(250, 200);
	ctx.advance(10);

	// Painted cells were created on the canvas.
	CHECK(lv_obj_get_child_count(canvas) > canvas_children_before);
	// The bounds label now carries coverage percent and a coordinate range.
	CHECK(find_label_containing(ctx.screen(), "%") != nullptr);
	CHECK(find_label_containing(ctx.screen(), "x ") != nullptr);
	CHECK(find_label_containing(ctx.screen(), "Paint to reveal") == nullptr);

	ctx.release();
	ctx.advance(10);
}

TEST_CASE("Paint segment interpolation fills cells between endpoints",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	TouchPaintCoverageBoard board(ctx.screen(), kPhysicalGeometry);
	board.create();
	lv_obj_update_layout(ctx.screen());

	lv_obj_t *canvas = first_canvas(ctx.screen());
	REQUIRE(canvas != nullptr);
	const uint32_t canvas_children_before = lv_obj_get_child_count(canvas);

	ctx.press(canvas);
	ctx.advance(10);
	// A single far jump paints the interpolated segment, not just one cell.
	ctx.move_while_pressed(260, 220);
	ctx.advance(10);

	const uint32_t painted = lv_obj_get_child_count(canvas) -
				 canvas_children_before;
	CHECK(painted >= 3);

	ctx.release();
	ctx.advance(10);
}

TEST_CASE("Diagnostic board scope ends before LVGL test context teardown",
	  "[touch-diagnostics]")
{
	test_support::LvglTestContext ctx;
	{
		TouchCalibrationBoard board(ctx.screen(), kPhysicalGeometry);
		board.create();
		lv_obj_update_layout(ctx.screen());
		lv_obj_t *canvas = first_canvas(ctx.screen());
		REQUIRE(canvas != nullptr);
		ctx.press(canvas);
		ctx.advance(10);
		// Board destroyed here while LVGL is still alive. No further events
		// fire on its objects before context teardown.
	}
	CHECK(ctx.screen() != nullptr);
}
