/**
 * @file lvgl_app_session.cpp
 * LVGL framebuffer snapshots and external app session lifecycle.
 */

#include <cerrno>
#include <chrono>
#include <cstring>
#include <exception>
#include <filesystem>
#include <map>
#include <memory>
#include <span>
#include <string>
#include <thread>
#include <utility>

#include <fcntl.h>
#include <linux/input.h>
#include <lvgl.h>
#include <src/core/lv_obj.h>
#include <src/display/lv_display_private.h>
#include <src/draw/lv_draw_buf.h>
#include <src/others/snapshot/lv_snapshot.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#include <spdlog/spdlog.h>

#include "lvgl_main/app_catalog.hpp"
#include "lvgl_main/framebuffer_snapshot.hpp"
#include "lvgl_main/launcher_child_control.hpp"
#include "lvgl_main/launcher_exit_gesture.hpp"
#include "lvgl_ui_internal.hpp"
#include "lvgl_main/simulator_policy.hpp"

#if !NANOKVM_GO_SIMULATOR_BUILD
lvgl_main::framebuffer_snapshot::ViewRotation
UILVGLApp::snapshot_view_rotation() const
{
	if (_disp != nullptr && _disp->rotation == LV_DISPLAY_ROTATION_90) {
		return lvgl_main::framebuffer_snapshot::ViewRotation::Clockwise270;
	}
	return lvgl_main::framebuffer_snapshot::ViewRotation::Clockwise90;
}

void UILVGLApp::save_snapshot_capture(
	lvgl_main::framebuffer_snapshot::CaptureResult capture,
	std::chrono::system_clock::time_point captured_at)
{
	if (!capture.ok) {
		spdlog::error("Framebuffer snapshot failed: {}", capture.error);
		return;
	}

	const auto saved = lvgl_main::framebuffer_snapshot::save_bmp_snapshot(
		capture.frame, snapshot_view_rotation(),
		_config.screenshot.output_directory, captured_at);
	if (!saved.ok) {
		spdlog::error("Framebuffer snapshot failed: {}", saved.error);
		return;
	}
	spdlog::info("[lvgl] Framebuffer snapshot saved: {}",
		     saved.path.string());
}

void UILVGLApp::handle_snapshot_request()
{
	if (!consume_snapshot_request()) {
		return;
	}

	const auto captured_at = std::chrono::system_clock::now();
	try {
		lv_draw_buf_t *snapshot_raw = nullptr;
		{
			LvLockGuard lock;
			lv_obj_t *screen = lv_screen_active();
			if (screen != nullptr) {
				snapshot_raw = lv_snapshot_take(
					screen, LV_COLOR_FORMAT_RGB565);
			}
		}
		std::unique_ptr<lv_draw_buf_t, decltype(&lv_draw_buf_destroy)>
			snapshot(snapshot_raw, &lv_draw_buf_destroy);
		if (!snapshot) {
			spdlog::error("LVGL snapshot failed: no draw buffer");
			return;
		}

		const lvgl_main::framebuffer_snapshot::FramebufferGeometry
			geometry {
				.width = snapshot->header.w,
				.height = snapshot->header.h,
				.virtual_width = snapshot->header.w,
				.virtual_height = snapshot->header.h,
				.bits_per_pixel = 16,
				.line_length = snapshot->header.stride,
				.red = { 11, 5, 0 },
				.green = { 5, 6, 0 },
				.blue = { 0, 5, 0 },
			};
		auto capture =
			lvgl_main::framebuffer_snapshot::copy_visible_rgb565(
				std::span<const uint8_t>(snapshot->data,
							 snapshot->data_size),
				geometry);
		if (!capture.ok) {
			spdlog::error("LVGL snapshot failed: {}",
				      capture.error);
			return;
		}

		const auto saved =
			lvgl_main::framebuffer_snapshot::save_bmp_snapshot(
				capture.frame,
				lvgl_main::framebuffer_snapshot::ViewRotation::
					None,
				_config.screenshot.output_directory, captured_at);
		if (!saved.ok) {
			spdlog::error("LVGL snapshot failed: {}", saved.error);
			return;
		}
		spdlog::info("[lvgl] LVGL snapshot saved: {}",
			     saved.path.string());
	} catch (const std::exception &error) {
		spdlog::error("LVGL snapshot failed: {}", error.what());
	}
}

bool UILVGLApp::handle_launcher_snapshot_request(pid_t &child,
						 bool child_stopped)
{
	if (!consume_snapshot_request()) {
		return false;
	}

	const auto captured_at = std::chrono::system_clock::now();
	try {
		if (child_stopped) {
			save_snapshot_capture(
				lvgl_main::framebuffer_snapshot::
					capture_framebuffer(
						_config.display
							.framebuffer_device),
				captured_at);
			return false;
		}

		if (child <= 0) {
			spdlog::error(
				"Framebuffer snapshot failed: Launcher child is unavailable");
			return true;
		}

		lvgl_main::launcher_child_control::SystemProcessOps process_ops;
		const auto stop_result =
			lvgl_main::launcher_child_control::stop_and_wait(
				process_ops, child,
				std::chrono::milliseconds { 200 });
		lvgl_main::launcher_child_control::ResumeGuard resume_guard(
			process_ops, stop_result.stop_signal_sent ? child : -1);
		auto resume_and_log = [&]() -> int {
			const int resume_error = resume_guard.resume();
			if (resume_error != 0 && resume_error != ESRCH) {
				spdlog::error(
					"Framebuffer snapshot failed to resume Launcher child {}: {}",
					child, std::strerror(resume_error));
			}
			return resume_error;
		};

		if (stop_result.status ==
		    lvgl_main::launcher_child_control::StopStatus::Exited) {
			resume_guard.dismiss();
			child = -1;
			return true;
		}
		if (stop_result.status ==
		    lvgl_main::launcher_child_control::StopStatus::TimedOut) {
			const int resume_error = resume_and_log();
			if (resume_error == ESRCH) {
				child = -1;
				return true;
			}
			spdlog::error(
				"Framebuffer snapshot failed: timed out stopping Launcher child {}",
				child);
			return false;
		}
		if (stop_result.status !=
		    lvgl_main::launcher_child_control::StopStatus::Stopped) {
			const int resume_error = resume_and_log();
			if (resume_error == ESRCH ||
			    stop_result.error == ESRCH ||
			    stop_result.error == ECHILD) {
				child = -1;
				return true;
			}
			spdlog::error(
				"Framebuffer snapshot failed: could not confirm Launcher child {} stopped: {}",
				child, std::strerror(stop_result.error));
			return false;
		}

		auto capture =
			lvgl_main::framebuffer_snapshot::capture_framebuffer(
				_config.display.framebuffer_device);
		const int resume_error = resume_and_log();
		if (resume_error == ESRCH) {
			child = -1;
			return true;
		}
		if (resume_error != 0) {
			return false;
		}
		save_snapshot_capture(std::move(capture), captured_at);
	} catch (const std::exception &error) {
		spdlog::error("Framebuffer snapshot failed: {}", error.what());
	}
	return false;
}
#endif
#if !NANOKVM_GO_SIMULATOR_BUILD

namespace
{

struct ExclusiveTouchGrab {
	explicit ExclusiveTouchGrab(int touch_fd)
		: fd(touch_fd)
	{
	}

	~ExclusiveTouchGrab()
	{
		release();
	}

	bool acquire()
	{
		if (active) {
			return true;
		}
		if (fd < 0 || ::ioctl(fd, EVIOCGRAB, 1) != 0) {
			return false;
		}
		active = true;
		return true;
	}

	bool release()
	{
		if (!active) {
			return true;
		}
		if (::ioctl(fd, EVIOCGRAB, 0) != 0) {
			return false;
		}
		active = false;
		return true;
	}

	int fd { -1 };
	bool active { false };
};

} // namespace

void UILVGLApp::run_app_session(const std::string &script)
{
	const auto &lc = _config.launcher;
	if (!lc.enabled) {
		spdlog::warn(
			"Launcher disabled by config; ignoring launch request");
		return;
	}
	if (_config.touch.disabled || _fd_tp < 0) {
		spdlog::error(
			"App session needs touch input for the exit gesture; "
			"aborting launch");
		return;
	}
	if (script.empty()) {
		spdlog::error("App launch request has an empty script path");
		return;
	}

	const std::string &target = script;
	std::map<std::string, std::string> app_environment;
	std::string app_environment_error;
	if (!lvgl_main::parse_app_environment(
		    std::filesystem::path(target).parent_path() / "app.json",
		    app_environment, app_environment_error)) {
		spdlog::error("Cannot launch App: {}", app_environment_error);
		return;
	}
	// Sub-apps render raw to fb0, bypassing the LVGL rotation the host applies.
	// Tell appbase how to rotate so apps appear upright: host ROTATION_270 (normal)
	// needs 90, ROTATION_90 (180-flipped) needs 270.
	const char *fb_rotate =
		(_disp->rotation == LV_DISPLAY_ROTATION_90) ? "270" : "90";
	spdlog::info(
		"[lvgl] Entering app session: {} {} (APPBASE_FB_ROTATE={})",
		lc.python_bin, target, fb_rotate);

	// Detach evdev from LVGL; this session reads _fd_tp directly.
	set_touch_input_fd(-1);

	const pid_t pid = ::fork();
	if (pid < 0) {
		spdlog::error("fork failed: {}", std::strerror(errno));
		set_touch_input_fd(_fd_tp);
		reset_touch_wake_filter();
		return;
	}
	if (pid == 0) {
		::setenv("APPBASE_FB_ROTATE", fb_rotate, 1);
		for (const auto &[name, value] : app_environment) {
			::setenv(name.c_str(), value.c_str(), 1);
		}
		const char *existing_python_path = ::getenv("PYTHONPATH");
		std::string python_path =
			lvgl_main::simulator_policy::launcher_apps_path()
				.string();
		if (existing_python_path != nullptr &&
		    existing_python_path[0] != '\0') {
			python_path += ":";
			python_path += existing_python_path;
		}
		::setenv("PYTHONPATH", python_path.c_str(), 1);
		const std::filesystem::path app_directory =
			std::filesystem::path(target).parent_path();
		if (::chdir(app_directory.c_str()) != 0) {
			::_exit(127);
		}
		::execlp(lc.python_bin.c_str(), lc.python_bin.c_str(),
			 target.c_str(), static_cast<char *>(nullptr));
		::_exit(127);
	}

	pid_t child = pid;
	bool exit_requested = false;

	// A short inward swipe from either edge pauses the App and opens a centered
	// confirmation. The original gesture release is ignored by the dialog so it
	// cannot accidentally activate either choice.
	const int edge_zone = lc.exit_gesture_zone_px;
	const int hor_res =
		static_cast<int>(lv_display_get_horizontal_resolution(_disp));

	ExitConfirmationOverlay confirmation;
	ExclusiveTouchGrab touch_grab(_fd_tp);
	const bool confirmation_ok = confirmation.open(
		_config.display.framebuffer_device,
		(_disp->rotation == LV_DISPLAY_ROTATION_90) ? 270 : 90);
	if (!confirmation_ok) {
		spdlog::error(
			"Exit confirmation overlay could not open the framebuffer");
	}

	auto gesture_side = lvgl_main::LauncherExitGestureSide::None;
	int gesture_start_x = 0;
	bool confirmation_visible = false;
	bool confirmation_accepts_input = false;
	auto pressed_choice =
		lvgl_main::LauncherExitConfirmationChoice::None;
	bool prev_pressed = false;
	bool child_stopped = false;
#if NANOKVM_GO_DEBUG_BUILD
	lvgl_main::launcher_child_control::SystemProcessOps launcher_process_ops;
#endif

	auto stop_child_for_confirmation = [&]() {
		if (child_stopped) {
			return true;
		}
#if NANOKVM_GO_DEBUG_BUILD
		const auto stop_result =
			lvgl_main::launcher_child_control::stop_and_wait(
				launcher_process_ops, child,
				std::chrono::milliseconds { 200 });
		lvgl_main::launcher_child_control::ResumeGuard resume_guard(
			launcher_process_ops,
			stop_result.stop_signal_sent ? child : -1);
		if (stop_result.status ==
		    lvgl_main::launcher_child_control::StopStatus::Exited) {
			resume_guard.dismiss();
			child = -1;
			return false;
		}
		if (stop_result.status ==
		    lvgl_main::launcher_child_control::StopStatus::Stopped) {
			resume_guard.dismiss();
			child_stopped = true;
			return true;
		}

		const int resume_error = resume_guard.resume();
		if (resume_error == ESRCH) {
			child = -1;
		} else if (resume_error != 0) {
			spdlog::error(
				"Exit gesture failed to resume Launcher child {}: {}",
				child, std::strerror(resume_error));
		}
		spdlog::error(
			"Exit gesture could not confirm Launcher child {} stopped: {}",
			child,
			stop_result.status ==
					lvgl_main::launcher_child_control::StopStatus::
						TimedOut ?
				"timed out" :
				std::strerror(stop_result.error));
		return false;
#else
		if (::kill(child, SIGSTOP) == 0) {
			child_stopped = true;
			return true;
		}
		spdlog::error("Exit gesture failed to stop Launcher child {}: {}",
			      child, std::strerror(errno));
		return false;
#endif
	};

	while (g_running != 0 && !exit_requested) {
#if !NANOKVM_GO_SIMULATOR_BUILD
		if (handle_launcher_snapshot_request(child, child_stopped)) {
			break;
		}
#endif
		drain_touch_input_events(_fd_tp);

		const bool pressed = touch_input_is_pressed();
		int logical_x = 0;
		int logical_y = 0;
		map_raw_to_logical(_disp, touch_input_last_x(),
				   touch_input_last_y(), logical_x, logical_y);

		if (confirmation_visible) {
			if (!confirmation_accepts_input) {
				if (!pressed && prev_pressed) {
					confirmation_accepts_input = true;
				}
			} else if (pressed && !prev_pressed) {
				pressed_choice =
					confirmation.choice_at(logical_x, logical_y);
			} else if (!pressed && prev_pressed) {
				const auto released_choice =
					confirmation.choice_at(logical_x, logical_y);
				if (pressed_choice !=
					    lvgl_main::LauncherExitConfirmationChoice::
						    None &&
				    pressed_choice == released_choice) {
					if (released_choice ==
					    lvgl_main::LauncherExitConfirmationChoice::Yes) {
						exit_requested = true;
						spdlog::info(
							"[lvgl] Exit confirmed, ending app session");
					} else {
						confirmation.restore();
						if (!touch_grab.release()) {
							spdlog::error(
								"Exit confirmation failed to release exclusive touch input: {}",
								std::strerror(errno));
							exit_requested = true;
						} else if (::kill(child, SIGCONT) == 0) {
							child_stopped = false;
							confirmation_visible = false;
							confirmation_accepts_input = false;
							spdlog::info(
								"[lvgl] Exit cancelled, resuming app session");
						} else {
							spdlog::error(
								"Exit confirmation failed to resume Launcher child {}: {}",
								child, std::strerror(errno));
							exit_requested = true;
						}
					}
				}
				pressed_choice =
					lvgl_main::LauncherExitConfirmationChoice::None;
			}
		} else if (pressed && !prev_pressed) {
			gesture_side = lvgl_main::launcher_exit_gesture_start_side(
				logical_x, hor_res, edge_zone);
				gesture_start_x = logical_x;
		} else if (!pressed && prev_pressed) {
			gesture_side = lvgl_main::LauncherExitGestureSide::None;
		} else if (
			pressed &&
			lvgl_main::launcher_exit_gesture_reached(
					gesture_side, gesture_start_x, logical_x, hor_res)) {
				gesture_side = lvgl_main::LauncherExitGestureSide::None;
				if (!confirmation_ok) {
					spdlog::error(
						"Exit confirmation is unavailable");
				} else if (!touch_grab.acquire()) {
					spdlog::error(
						"Exit confirmation could not acquire exclusive touch input: {}",
						std::strerror(errno));
				} else if (stop_child_for_confirmation()) {
					confirmation.snapshot();
					confirmation.animate_in();
					confirmation_visible = true;
					confirmation_accepts_input = false;
					pressed_choice = lvgl_main::
						LauncherExitConfirmationChoice::None;
					spdlog::info(
						"[lvgl] Exit confirmation opened by edge swipe");
				} else {
					if (!touch_grab.release()) {
						spdlog::error(
							"Exit gesture failed to release exclusive touch input: {}",
							std::strerror(errno));
					}
					if (child <= 0) {
						break;
					}
				}
		}
		prev_pressed = pressed;

		if (!child_stopped &&
		    ::waitpid(child, nullptr, WNOHANG) == child) {
			spdlog::info("[lvgl] App child exited on its own");
			child = -1;
			break;
		}

		std::this_thread::sleep_for(std::chrono::milliseconds(20));
	}

	// Resume the child if it was frozen, so the termination signals land.
	if (child > 0 && child_stopped) {
		::kill(child, SIGCONT);
		child_stopped = false;
	}
	confirmation.close();

	if (child > 0) {
		::kill(child, SIGTERM);
		const auto deadline =
			std::chrono::steady_clock::now() +
			std::chrono::milliseconds(lc.child_term_timeout_ms);
		bool reaped = false;
		while (std::chrono::steady_clock::now() < deadline) {
			if (::waitpid(child, nullptr, WNOHANG) == child) {
				reaped = true;
				break;
			}
			std::this_thread::sleep_for(
				std::chrono::milliseconds(20));
		}
		if (!reaped) {
			spdlog::warn(
				"App child did not exit on SIGTERM; sending SIGKILL");
			::kill(child, SIGKILL);
			while (::waitpid(child, nullptr, 0) < 0 &&
			       errno == EINTR) {
			}
		}
	}

	// Drain residual touch events so the gesture release is not seen as a tap.
	drain_touch_input_events(_fd_tp);
	if (!touch_grab.release()) {
		spdlog::error(
			"App session failed to release exclusive touch input: {}",
			std::strerror(errno));
	}
	reset_touch_input_state();
	set_touch_input_fd(_fd_tp);

	// The main loop was paused for the whole session, so the activity timer
	// is stale; refresh it so auto-sleep does not blank the screen the
	// instant the UI returns.
	_last_input_activity = std::chrono::steady_clock::now();
	_last_input_activity_seq = touch_input_activity_sequence();

	// Wipe the whole framebuffer (incl. the dead-area column the app wrote
	// but LVGL never repaints), then force an immediate full redraw of the
	// UI so nothing of the app's screen lingers.
	clear_framebuffer(_config.display.framebuffer_device);
	{
		LvLockGuard lock;
		if (lv_screen_active() != nullptr) {
			lv_obj_invalidate(lv_screen_active());
		}
		lv_refr_now(_disp);
	}
	spdlog::info("[lvgl] App session ended, UI restored");
}
#endif
