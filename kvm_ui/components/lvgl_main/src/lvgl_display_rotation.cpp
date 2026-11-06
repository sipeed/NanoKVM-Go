/**
 * @file lvgl_display_rotation.cpp
 * LVGL display buffers, geometry, rotation, and simulator callbacks.
 */

#include <algorithm>
#include <atomic>
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <memory>
#include <string>

#include <lvgl.h>
#include <spdlog/spdlog.h>
#include <src/core/lv_obj.h>
#include <src/core/lv_obj_pos.h>
#include <src/core/lv_obj_style.h>
#include <src/core/lv_obj_style_gen.h>
#include <src/core/lv_obj_tree.h>
#include <src/display/lv_display.h>
#include <src/display/lv_display_private.h>
#include <src/misc/lv_anim.h>
#include <src/misc/lv_color.h>

#include "lvgl_main/auto_rotation_runtime.hpp"
#include "lvgl_ui_internal.hpp"
#include "ui/page_manager.hpp"

namespace
{

constexpr uint32_t kDisplayRotationFadeDurationMs = 300;

} // namespace

void UILVGLApp::release_draw_buffers()
{
	if (_draw_buf_1 != nullptr) {
		std::free(_draw_buf_1);
		_draw_buf_1 = nullptr;
	}
	if (_draw_buf_2 != nullptr) {
		std::free(_draw_buf_2);
		_draw_buf_2 = nullptr;
	}
	_draw_buf_size = 0;
}

void UILVGLApp::handle_touch_click_marker()
{
	if (!_touch_click_marker_enabled &&
	    !lvgl_main::wifi_latency_probe::visible_marker_enabled()) {
		return;
	}

	const bool visible = touch_click_marker_state.visible.load(
		std::memory_order_acquire);
	const int32_t x =
		touch_click_marker_state.x.load(std::memory_order_acquire);
	const int32_t y =
		touch_click_marker_state.y.load(std::memory_order_acquire);
	const uint64_t revision = touch_click_marker_state.revision.load(
		std::memory_order_acquire);
	if (visible == _touch_click_marker_was_visible &&
	    (!visible ||
	     (x == _touch_click_marker_x && y == _touch_click_marker_y &&
	      revision == _touch_click_marker_revision))) {
		return;
	}

	LvLockGuard lock;
	lv_obj_t *screen = lv_screen_active();
	if (screen == nullptr) {
		return;
	}

	if (_touch_click_marker_was_visible) {
		lv_area_t old_area = touch_click_marker_area(
			_touch_click_marker_x, _touch_click_marker_y);
		lv_obj_invalidate_area(screen, &old_area);
	}
	if (visible) {
		lv_area_t new_area = touch_click_marker_area(x, y);
		lv_obj_invalidate_area(screen, &new_area);
	}

	_touch_click_marker_was_visible = visible;
	_touch_click_marker_x = x;
	_touch_click_marker_y = y;
	_touch_click_marker_revision = revision;
}

bool UILVGLApp::setup_fbdev_fixed_buffers()
{
	const uint32_t hor_res = lv_display_get_horizontal_resolution(_disp);
	const lv_color_format_t cf = lv_display_get_color_format(_disp);
	const uint32_t px_size = lv_color_format_get_size(cf);
	if (hor_res == 0 || px_size == 0) {
		spdlog::error(
			"Invalid display format/resolution: hor_res={}, px_size={}",
			hor_res, px_size);
		return false;
	}

	release_draw_buffers();
	_draw_buf_size =
		static_cast<size_t>(hor_res) * px_size * kFbdevBufferLines;
	_draw_buf_1 = static_cast<uint8_t *>(std::malloc(_draw_buf_size));
	if (_draw_buf_1 == nullptr) {
		spdlog::error("Failed to allocate draw buffer 1, size={}",
			      _draw_buf_size);
		return false;
	}

	if (kFbdevBufferCount == 2U) {
		_draw_buf_2 =
			static_cast<uint8_t *>(std::malloc(_draw_buf_size));
		if (_draw_buf_2 == nullptr) {
			spdlog::error(
				"Failed to allocate draw buffer 2, size={}",
				_draw_buf_size);
			release_draw_buffers();
			return false;
		}
	}

	lv_display_set_buffers(_disp, _draw_buf_1,
			       kFbdevBufferCount == 2U ? _draw_buf_2 : nullptr,
			       static_cast<uint32_t>(_draw_buf_size),
			       LV_DISPLAY_RENDER_MODE_PARTIAL);

	spdlog::info("[lvgl] Applied fixed fbdev refresh config: "
		     "render_mode=PARTIAL, buffer_count={}, buffer_lines={}, "
		     "buffer_bytes={})",
		     kFbdevBufferCount, kFbdevBufferLines, _draw_buf_size);
	return true;
}

void UILVGLApp::apply_display_180_rotation(bool enabled)
{
	if (_disp == nullptr) {
		return;
	}

#if NANOKVM_GO_SIMULATOR_BUILD
	const lv_display_rotation_t display_rotation =
		enabled ? LV_DISPLAY_ROTATION_180 : LV_DISPLAY_ROTATION_0;
	constexpr int32_t display_offset_x = 0;
#else
	const lv_display_rotation_t display_rotation =
		enabled ? LV_DISPLAY_ROTATION_90 : LV_DISPLAY_ROTATION_270;
	const int32_t display_offset_x = enabled ? -kPanelLeftDeadAreaWidth :
						   kPanelLeftDeadAreaWidth;
#endif
	lv_display_set_offset(_disp, display_offset_x, 0);
	lv_display_set_rotation(_disp, display_rotation);
	_display_180_rotation_enabled.store(enabled, std::memory_order_release);
	if (lv_screen_active() != nullptr) {
		lv_obj_invalidate(lv_screen_active());
	}
	spdlog::info(
		"[lvgl] Display orientation applied: offset_x={}, rotation={}, reverse={}",
		display_offset_x,
#if NANOKVM_GO_SIMULATOR_BUILD
		enabled ? 180 : 0,
#else
		enabled ? 90 : 270,
#endif
		enabled);
}

void UILVGLApp::request_display_180_rotation(bool enabled)
{
	if (_display_rotation_fade_active) {
		_display_rotation_fade_queued_reverse = enabled ? 1 : 0;
		return;
	}

	if (_display_180_rotation_enabled.load(std::memory_order_acquire) ==
	    enabled) {
		return;
	}

	if (!start_display_rotation_fade(enabled)) {
		apply_display_180_rotation(enabled);
	}
}

bool UILVGLApp::start_display_rotation_fade(bool enabled)
{
	lv_obj_t *top_layer = lv_layer_top();
	if (_disp == nullptr || top_layer == nullptr) {
		return false;
	}

	_display_rotation_fade_overlay = lv_obj_create(top_layer);
	if (_display_rotation_fade_overlay == nullptr) {
		return false;
	}
#if NANOKVM_GO_SIMULATOR_BUILD
	lv_obj_set_size(_display_rotation_fade_overlay, kSimulatorLogicalWidth,
			kSimulatorLogicalHeight);
#else
	lv_obj_set_size(_display_rotation_fade_overlay, kPanelPhysicalWidth,
			kPanelPhysicalHeight);
#endif
	lv_obj_set_pos(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_bg_color(_display_rotation_fade_overlay,
				  lv_color_black(), 0);
	lv_obj_set_style_bg_opa(_display_rotation_fade_overlay, LV_OPA_TRANSP,
				0);
	lv_obj_set_style_radius(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_border_width(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_pad_all(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_margin_all(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_shadow_width(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_style_outline_width(_display_rotation_fade_overlay, 0, 0);
	lv_obj_set_scrollbar_mode(_display_rotation_fade_overlay,
				  LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_display_rotation_fade_overlay,
			  LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_add_flag(_display_rotation_fade_overlay, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_invalidate(_display_rotation_fade_overlay);

	_display_rotation_fade_active = true;
	_display_rotation_fade_fading_out = true;
	_display_rotation_fade_target_reverse = enabled;
	_display_rotation_fade_queued_reverse = -1;

	if (!start_display_rotation_fade_animation(LV_OPA_TRANSP, LV_OPA_COVER,
						   true)) {
		cleanup_display_rotation_fade();
		return false;
	}
	return true;
}

bool UILVGLApp::start_display_rotation_fade_animation(int32_t from_opa,
						      int32_t to_opa,
						      bool fading_out)
{
	if (_display_rotation_fade_overlay == nullptr ||
	    !lv_obj_is_valid(_display_rotation_fade_overlay)) {
		return false;
	}

	_display_rotation_fade_fading_out = fading_out;
	lv_anim_delete(_display_rotation_fade_overlay,
		       display_rotation_fade_opa_exec_cb);

	lv_anim_t anim {};
	lv_anim_init(&anim);
	lv_anim_set_var(&anim, _display_rotation_fade_overlay);
	lv_anim_set_exec_cb(&anim, display_rotation_fade_opa_exec_cb);
	lv_anim_set_values(&anim, from_opa, to_opa);
	lv_anim_set_time(&anim, kDisplayRotationFadeDurationMs);
	lv_anim_set_path_cb(&anim, lv_anim_path_linear);
	lv_anim_set_ready_cb(&anim, display_rotation_fade_ready_cb);
	lv_anim_set_user_data(&anim, this);
	return lv_anim_start(&anim) != nullptr;
}

void UILVGLApp::finish_display_rotation_fade_out()
{
	if (!_display_rotation_fade_active) {
		return;
	}

	if (_display_rotation_fade_queued_reverse >= 0) {
		_display_rotation_fade_target_reverse =
			_display_rotation_fade_queued_reverse != 0;
		_display_rotation_fade_queued_reverse = -1;
	}

	apply_display_180_rotation(_display_rotation_fade_target_reverse);
	if (_display_rotation_fade_overlay != nullptr &&
	    lv_obj_is_valid(_display_rotation_fade_overlay)) {
		lv_obj_set_style_bg_opa(_display_rotation_fade_overlay,
					LV_OPA_COVER, 0);
		lv_obj_invalidate(_display_rotation_fade_overlay);
	}

	if (!start_display_rotation_fade_animation(LV_OPA_COVER, LV_OPA_TRANSP,
						   false)) {
		cleanup_display_rotation_fade();
	}
}

void UILVGLApp::finish_display_rotation_fade_in()
{
	const int queued_reverse = _display_rotation_fade_queued_reverse;
	_display_rotation_fade_queued_reverse = -1;
	cleanup_display_rotation_fade();

	if (queued_reverse < 0) {
		return;
	}

	const bool reverse = queued_reverse != 0;
	if (!_display_rotation_fade_active &&
	    _display_180_rotation_enabled.load(std::memory_order_acquire) ==
		    reverse) {
		return;
	}
	if (!start_display_rotation_fade(reverse)) {
		apply_display_180_rotation(reverse);
	}
}

void UILVGLApp::cleanup_display_rotation_fade()
{
	if (_display_rotation_fade_overlay != nullptr) {
		if (lv_obj_is_valid(_display_rotation_fade_overlay)) {
			lv_anim_delete(_display_rotation_fade_overlay,
				       display_rotation_fade_opa_exec_cb);
			lv_obj_delete(_display_rotation_fade_overlay);
		}
		_display_rotation_fade_overlay = nullptr;
	}

	_display_rotation_fade_active = false;
	_display_rotation_fade_fading_out = false;
	_display_rotation_fade_queued_reverse = -1;
}

void UILVGLApp::display_rotation_fade_opa_exec_cb(void *var, int32_t value)
{
	auto *overlay = static_cast<lv_obj_t *>(var);
	if (overlay == nullptr || !lv_obj_is_valid(overlay)) {
		return;
	}

	const int32_t opa =
		std::clamp(value, 0, static_cast<int32_t>(LV_OPA_COVER));
	lv_obj_set_style_bg_opa(overlay, static_cast<lv_opa_t>(opa), 0);
	lv_obj_invalidate(overlay);
}

void UILVGLApp::display_rotation_fade_ready_cb(lv_anim_t *anim)
{
	auto *self = static_cast<UILVGLApp *>(lv_anim_get_user_data(anim));
	if (self == nullptr) {
		return;
	}

	if (self->_display_rotation_fade_fading_out) {
		self->finish_display_rotation_fade_out();
	} else {
		self->finish_display_rotation_fade_in();
	}
}

void UILVGLApp::start_auto_rotation()
{
	if (_touch_calibration_mode) {
		return;
	}
	if (!_auto_rotation_imu_available) {
		return;
	}
	if (_auto_rotation_runtime && _auto_rotation_runtime->is_running()) {
		return;
	}

	_auto_rotation_pending_reverse.store(-1, std::memory_order_release);
	lvgl_main::AutoRotationRuntimeConfig runtime_config;
	_auto_rotation_runtime =
		std::make_unique<lvgl_main::AutoRotationRuntime>(
			runtime_config, [this](bool reverse) {
				_auto_rotation_pending_reverse.store(
					reverse ? 1 : 0,
					std::memory_order_release);
			});
	if (!_auto_rotation_runtime->start()) {
		spdlog::warn("IMU auto rotation runtime start failed");
		_auto_rotation_runtime.reset();
	}
}

void UILVGLApp::stop_auto_rotation()
{
	_auto_rotation_pending_reverse.store(-1, std::memory_order_release);
	if (_auto_rotation_runtime) {
		_auto_rotation_runtime->stop();
		_auto_rotation_runtime.reset();
	}
	_auto_rotation_pending_reverse.store(-1, std::memory_order_release);
}

void UILVGLApp::set_auto_rotation_enabled(bool enabled)
{
	if (!_auto_rotation_imu_available) {
		stop_auto_rotation();
		if (_page_manager) {
			_page_manager->set_runtime_auto_rotation_active(
				_display_180_rotation_enabled.load(
					std::memory_order_acquire));
		}
		return;
	}

	if (_config.ui.enable_auto_rotation != enabled) {
		_config.ui.enable_auto_rotation = enabled;
		if (enabled) {
			start_auto_rotation();
		} else {
			stop_auto_rotation();
		}

		if (_runtime_service) {
			_runtime_service->set_ui_auto_rotation_enabled(enabled);
		} else {
			std::string error;
			if (!lvgl_main::save_app_config(_config, error,
							_config_path)) {
				spdlog::warn(
					"Failed to persist auto rotation setting: {}",
					error);
			}
		}
	}

	if (_page_manager) {
		_page_manager->set_runtime_auto_rotation_active(
			_config.ui.enable_auto_rotation);
	}
}

void UILVGLApp::set_manual_display_rotation_enabled(bool enabled)
{
	if (_auto_rotation_imu_available) {
		return;
	}

	stop_auto_rotation();
	request_display_180_rotation(enabled);
	if (_config.ui.manual_display_rotation != enabled) {
		_config.ui.manual_display_rotation = enabled;
		if (_runtime_service) {
			_runtime_service->set_ui_manual_display_rotation(
				enabled);
		} else {
			std::string error;
			if (!lvgl_main::save_app_config(_config, error,
							_config_path)) {
				spdlog::warn(
					"Failed to persist manual display rotation setting: {}",
					error);
			}
		}
	}
	if (_page_manager) {
		_page_manager->set_runtime_auto_rotation_active(enabled);
	}
}

void UILVGLApp::handle_auto_rotation()
{
	const int pending = _auto_rotation_pending_reverse.exchange(
		-1, std::memory_order_acq_rel);
	if (pending < 0 || !_config.ui.enable_auto_rotation ||
	    !_auto_rotation_runtime || !_auto_rotation_runtime->is_running()) {
		return;
	}

	const bool reverse = pending != 0;
	if (_display_180_rotation_enabled.load(std::memory_order_acquire) ==
	    reverse) {
		return;
	}

	LvLockGuard lock;
	request_display_180_rotation(reverse);
}

void UILVGLApp::configure_display_geometry()
{
#if NANOKVM_GO_SIMULATOR_BUILD
	if (_disp == nullptr) {
		return;
	}

	lv_display_set_resolution(_disp, kSimulatorLogicalWidth,
				  kSimulatorLogicalHeight);
	lv_display_set_physical_resolution(_disp, kSimulatorLogicalWidth,
					   kSimulatorLogicalHeight);
	lv_display_set_offset(_disp, 0, 0);
	lv_display_set_rotation(_disp, LV_DISPLAY_ROTATION_0);
	_display_180_rotation_enabled.store(false, std::memory_order_release);
	spdlog::info(
		"[lvgl] Simulator display geometry configured: logical={}x{}, zoom={}, "
		"physical_panel_emulation=off",
		kSimulatorLogicalWidth, kSimulatorLogicalHeight,
		kSimulatorWindowZoom);
	return;
#else
	if (_touch_calibration_mode) {
		// Calibration must expose the full physical framebuffer and raw touch area.
		lv_display_set_resolution(_disp, kPanelPhysicalHeight,
					  kPanelPhysicalWidth);
		lv_display_set_physical_resolution(_disp, kPanelPhysicalHeight,
						   kPanelPhysicalWidth);
		lv_display_set_offset(_disp, 0, 0);
		lv_display_set_rotation(_disp, LV_DISPLAY_ROTATION_270);
		spdlog::info(
			"[lvgl] Touch calibration display geometry configured: "
			"physical={}x{}, visible={}x{}, offset_x=0, rotation=270",
			kPanelPhysicalWidth, kPanelPhysicalHeight,
			kPanelPhysicalWidth, kPanelPhysicalHeight);
		return;
	}

	// The fbdev driver stores geometry before the 90/270 degree LVGL rotation.
	lv_display_set_resolution(_disp, kPanelVisibleHeight,
				  kPanelVisibleWidth);
	lv_display_set_physical_resolution(_disp, kPanelPhysicalHeight,
					   kPanelPhysicalWidth);
	apply_display_180_rotation(false);
	spdlog::info(
		"[lvgl] Display geometry configured: physical={}x{}, visible={}x{}, "
		"left_dead_area={}, ui_auto_rotation={}",
		kPanelPhysicalWidth, kPanelPhysicalHeight, kPanelVisibleWidth,
		kPanelVisibleHeight, kPanelLeftDeadAreaWidth,
		_config.ui.enable_auto_rotation);
#endif
}

#if NANOKVM_GO_SIMULATOR_BUILD
void UILVGLApp::simulator_display_delete_event_cb(lv_event_t *event)
{
	if (event == nullptr || lv_event_get_code(event) != LV_EVENT_DELETE) {
		return;
	}

	auto *self = static_cast<UILVGLApp *>(lv_event_get_user_data(event));
	auto *display =
		static_cast<lv_display_t *>(lv_event_get_current_target(event));
	if (self != nullptr && display == self->_disp) {
		self->_disp = nullptr;
		g_running = 0;
	}
}

void UILVGLApp::simulator_input_activity_event_cb(lv_event_t *event)
{
	if (event == nullptr) {
		return;
	}

	switch (lv_event_get_code(event)) {
	case LV_EVENT_PRESSED:
	case LV_EVENT_PRESSING:
	case LV_EVENT_RELEASED:
		record_touch_input_activity();
		break;
	default:
		break;
	}
}
#endif
