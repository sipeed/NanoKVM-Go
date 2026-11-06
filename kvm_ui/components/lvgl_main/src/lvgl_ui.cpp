#ifndef NANOKVM_GO_SIMULATOR_BUILD
#define NANOKVM_GO_SIMULATOR_BUILD 0
#endif

#include <atomic>
#include <cerrno>
#include <chrono>
#include <csignal>
#include <cstring>
#include <exception>
#include <filesystem>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include <fcntl.h>
#include <lvgl.h>
#include <spdlog/spdlog.h>
#if NANOKVM_GO_SIMULATOR_BUILD
#include <src/drivers/sdl/lv_sdl_mouse.h>
#include <src/drivers/sdl/lv_sdl_window.h>
#else
#include <src/drivers/display/fb/lv_linux_fbdev.h>
#endif
#include <src/indev/lv_indev.h>
#include <src/themes/default/lv_theme_default.h>
#include <unistd.h>

#include "lvgl_main/app_catalog.hpp"
#include "lvgl_main/auto_rotation_runtime.hpp"
#include "lvgl_main/lvgl_main.hpp"
#include "lvgl_ui_internal.hpp"
#include "lvgl_main/runtime_font_catalog.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "lvgl_main/simulator_policy.hpp"
#include "lvgl_touch_diagnostics.hpp"
#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_runtime.hpp"
#else
#include "lvgl_main/http_api_server.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#endif
#include "ui/background_assets.hpp"
#include "ui/page_manager.hpp"

volatile std::sig_atomic_t g_running = 1;
#if !NANOKVM_GO_SIMULATOR_BUILD
volatile std::sig_atomic_t g_snapshot_requested = 0;
#endif
namespace
{

ui::PageManager::PageTransitionMode
to_ui_page_transition_mode(lvgl_main::UiPageTransitionMode mode)
{
	switch (mode) {
	case lvgl_main::UiPageTransitionMode::Slide:
		return ui::PageManager::PageTransitionMode::Slide;
	case lvgl_main::UiPageTransitionMode::Flip:
		return ui::PageManager::PageTransitionMode::Flip;
	}
	return ui::PageManager::PageTransitionMode::Slide;
}

ui::WifiLoadingMode
to_ui_wifi_loading_mode(bool progress)
{
	return progress ? ui::WifiLoadingMode::Progress
			: ui::WifiLoadingMode::None;
}

}

static void signal_handler(int sig)
{
	(void)sig;
	g_running = 0;
}

#if !NANOKVM_GO_SIMULATOR_BUILD
static void snapshot_signal_handler(int sig)
{
	(void)sig;
	g_snapshot_requested = 1;
}

bool consume_snapshot_request()
{
	if (g_snapshot_requested == 0) {
		return false;
	}
	g_snapshot_requested = 0;
	return true;
}
#endif

#if !NANOKVM_GO_SIMULATOR_BUILD
bool UILVGLApp::configure_screenshot_control()
{
	g_snapshot_requested = 0;
	bool enabled = _config.screenshot.enabled;
	const int selected_signal = enabled ?
		lvgl_main::screenshot_control::signal_number(_config.screenshot.signal) :
		-1;
	if (enabled && selected_signal < 0) {
		spdlog::warn("Signal screenshots disabled because the configured signal is unsupported");
		enabled = false;
	}

	auto install = [](int signal, void (*handler)(int),
			  ScreenshotSignalRegistration &registration) {
		struct sigaction action {};
		action.sa_handler = handler;
		if (::sigemptyset(&action.sa_mask) != 0) {
			spdlog::warn("Failed to initialize screenshot signal mask for {}: {}",
				     signal, std::strerror(errno));
			return false;
		}
		action.sa_flags = SA_RESTART;
		if (::sigaction(signal, &action, nullptr) != 0) {
			spdlog::warn("Failed to install screenshot signal handler for {}: {}",
				     signal, std::strerror(errno));
			return false;
		}
		registration.signal = signal;
		registration.installed = true;
		return true;
	};

	const auto handler_for = [enabled, selected_signal](int signal) {
		return enabled && signal == selected_signal ? snapshot_signal_handler :
							       SIG_IGN;
	};
	if (!install(SIGUSR1, handler_for(SIGUSR1), _screenshot_user1_signal) ||
	    !install(SIGUSR2, handler_for(SIGUSR2), _screenshot_user2_signal)) {
		release_screenshot_control();
		return false;
	}

	if (!enabled) {
		spdlog::info("[lvgl] Signal screenshots disabled by config");
		return true;
	}

	auto lease_result = lvgl_main::screenshot_control::acquire_instance_lease(
		lvgl_main::screenshot_control::kPidfilePath, ::getpid(),
		_config.screenshot.signal, lvgl_main::screenshot_control::system_ops());
	if (!lease_result.lease.has_value()) {
		spdlog::warn("Signal screenshots enabled but pidfile publication failed: {}",
			     lease_result.result.message);
		return true;
	}
	_screenshot_instance_lease.emplace(std::move(*lease_result.lease));
	spdlog::info("[lvgl] Signal screenshots enabled: signal={}, output={}",
		     lvgl_main::screenshot_control::signal_name(
			     _config.screenshot.signal),
		     _config.screenshot.output_directory);
	return true;
}

void UILVGLApp::release_screenshot_control()
{
	auto ignore = [](ScreenshotSignalRegistration &registration) {
		if (!registration.installed) {
			return;
		}
		struct sigaction action {};
		action.sa_handler = SIG_IGN;
		(void)::sigemptyset(&action.sa_mask);
		action.sa_flags = SA_RESTART;
		if (::sigaction(registration.signal, &action, nullptr) != 0) {
			spdlog::warn("Failed to ignore screenshot signal {} during cleanup: {}",
				     registration.signal, std::strerror(errno));
		}
		registration.installed = false;
		registration.signal = 0;
	};

	ignore(_screenshot_user2_signal);
	ignore(_screenshot_user1_signal);
	g_snapshot_requested = 0;
	_screenshot_instance_lease.reset();
}
#endif

UILVGLApp::UILVGLApp(const lvgl_main::AppConfig &config)
	: UILVGLApp(config,
		    std::filesystem::path(
			    lvgl_main::app_config_defaults::kConfigPath.data()))
{
}

UILVGLApp::UILVGLApp(const lvgl_main::AppConfig &config,
		     std::filesystem::path config_path,
		     bool touch_calibration_mode,
		     bool touch_click_marker_enabled,
		     bool wifi_keyboard_test_mode)
	: _config(config)
	, _config_path(std::move(config_path))
	, _touch_calibration_mode(touch_calibration_mode ||
				  config.runtime.touch_calibrate ||
				  config.runtime.touch_paint_test)
	, _touch_diagnostic_mode(config.runtime.touch_calibrate ?
					 TouchDiagnosticMode::Calibration :
				 config.runtime.touch_paint_test ?
					 TouchDiagnosticMode::PaintTest :
					 TouchDiagnosticMode::None)
	, _touch_click_marker_enabled(touch_click_marker_enabled)
	, _wifi_keyboard_test_mode(wifi_keyboard_test_mode)
{
}

UILVGLApp::TouchDiagnosticMode UILVGLApp::touch_diagnostic_mode() const
{
	return _touch_diagnostic_mode;
}

UILVGLApp::~UILVGLApp()
{
	g_running = 0;
	cleanup();
}

lvgl_main::RuntimeCallbacks UILVGLApp::make_runtime_callbacks()
{
	lvgl_main::RuntimeCallbacks callbacks;
	callbacks.on_status =
		[this](const lvgl_main::RuntimeStatusPublication &publication) {
			schedule_runtime_status_apply(publication);
		};
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	callbacks.on_background_assets = [this](ui::BackgroundAssets assets) {
		schedule_background_assets_apply(std::move(assets));
	};
#endif
	return callbacks;
}

void UILVGLApp::cleanup()
{
	stop_auto_rotation();

#if !NANOKVM_GO_SIMULATOR_BUILD
	release_screenshot_control();
#endif

	{
		std::lock_guard<std::mutex> lock(_runtime_status_mutex);
		_runtime_status_mailbox.close();
		_runtime_status_apply_pending = false;
	}
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	{
		std::lock_guard<std::mutex> lock(_background_assets_mutex);
		_background_assets_mailbox_closed = true;
		_background_assets_apply_pending = false;
	}
#endif
	_ssid_raster_result_mailbox.close();
	if (_lvgl_initialized) {
		if (_apps_refresh_timer != nullptr) {
			lv_timer_delete(_apps_refresh_timer);
			_apps_refresh_timer = nullptr;
		}
		lv_async_call_cancel(runtime_status_apply_async_cb, this);
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
		lv_async_call_cancel(background_assets_apply_async_cb, this);
#endif
		cleanup_display_rotation_fade();
	}

	if (_ssid_raster_worker) {
		_ssid_raster_worker->close();
		_ssid_raster_worker.reset();
	}

#if !NANOKVM_GO_SIMULATOR_BUILD
	if (_http_server) {
		_http_server.reset();
	}
	_target_runtime_service = nullptr;
#endif

	if (_runtime_service) {
		_runtime_service.reset();
	}

	_page_manager.reset();
	_runtime_font_catalog.reset();

	if (_fd_tp >= 0) {
		::close(_fd_tp);
		_fd_tp = -1;
	}

	reset_touch_input_state();
	touch_click_marker_state.enabled.store(false,
					       std::memory_order_release);
	touch_click_marker_state.visible.store(false,
					       std::memory_order_release);
	touch_click_marker_state.revision.store(0U, std::memory_order_release);
	lvgl_main::wifi_latency_probe::reset();
	reset_touch_wake_filter();

	if (_indev != nullptr) {
		lv_indev_delete(_indev);
		_indev = nullptr;
	}

	if (_disp != nullptr) {
		lvgl_main::restore_color_lut_flush(_disp);
		lv_display_delete(_disp);
		_disp = nullptr;
	}
	lvgl_main::reset_color_lut_flush();

#if NANOKVM_GO_SIMULATOR_BUILD
	if (_sdl_quit_pending) {
		lv_sdl_quit();
		_sdl_quit_pending = false;
	}
#endif

	release_draw_buffers();

	if (_lvgl_initialized) {
		lv_deinit();
		_lvgl_initialized = false;
	}
	_init = false;
}

bool UILVGLApp::init()
{
	if (_init) {
		return true;
	}

	lvgl_main::wifi_latency_probe::initialize();
	const bool wifi_latency_probe_enabled =
		lvgl_main::wifi_latency_probe::enabled();
	const bool wifi_latency_probe_visible_marker_enabled =
		lvgl_main::wifi_latency_probe::visible_marker_enabled();
	touch_click_marker_state.enabled.store(_touch_click_marker_enabled ||
					       wifi_latency_probe_visible_marker_enabled,
					       std::memory_order_release);
	touch_click_marker_state.visible.store(false,
					       std::memory_order_release);
	touch_click_marker_state.x.store(0, std::memory_order_release);
	touch_click_marker_state.y.store(0, std::memory_order_release);
	touch_click_marker_state.revision.store(0U, std::memory_order_release);
	_touch_click_marker_was_visible = false;
	_touch_click_marker_revision = 0U;

	g_running = 1;
	std::signal(SIGINT, signal_handler);
	std::signal(SIGTERM, signal_handler);

#if !NANOKVM_GO_SIMULATOR_BUILD
	(void)configure_screenshot_control();
#endif

	lv_init();
	_lvgl_initialized = true;

#if NANOKVM_GO_SIMULATOR_BUILD
	_sdl_quit_pending = true;
	_disp = lv_sdl_window_create(kSimulatorLogicalWidth,
				     kSimulatorLogicalHeight);
	if (_disp == nullptr) {
		spdlog::error("Failed to create SDL display");
		cleanup();
		return false;
	}

	lv_sdl_window_set_title(
		_disp, lvgl_main::simulator_build_config::kWindowTitle.data());
	lv_sdl_window_set_resizeable(_disp, false);
	lv_sdl_window_set_zoom(_disp, kSimulatorWindowZoom);
	configure_display_geometry();
	lv_display_add_event_cb(_disp, simulator_display_delete_event_cb,
				LV_EVENT_DELETE, this);
	spdlog::info("[lvgl] Using SDL display: logical={}x{}, window={}x{}",
		     kSimulatorLogicalWidth, kSimulatorLogicalHeight,
		     kSimulatorLogicalWidth * kSimulatorWindowZoom,
		     kSimulatorLogicalHeight * kSimulatorWindowZoom);
#else
	_disp = lv_linux_fbdev_create();
	if (_disp == nullptr) {
		cleanup();
		return false;
	}

	lv_linux_fbdev_set_file(_disp,
				_config.display.framebuffer_device.c_str());
	spdlog::info("[lvgl] Using framebuffer device: {}",
		     _config.display.framebuffer_device);
	configure_display_geometry();
	if (!setup_fbdev_fixed_buffers()) {
		cleanup();
		return false;
	}

	{
		lvgl_main::install_color_lut_flush(_disp);
	}
#endif

	lv_theme_default_init(_disp, lv_color_white(), lv_color_white(), false,
			      LV_FONT_DEFAULT);

#if NANOKVM_GO_SIMULATOR_BUILD
	_indev = lv_sdl_mouse_create();
	if (_indev == nullptr) {
		spdlog::error("Failed to create SDL mouse input device");
		cleanup();
		return false;
	}
	lv_indev_set_display(_indev, _disp);
	lv_indev_add_event_cb(_indev, simulator_input_activity_event_cb,
			      LV_EVENT_ALL, this);
#else
	_indev = lv_indev_create();
	if (_indev == nullptr) {
		spdlog::error("Failed to create LVGL input device");
		cleanup();
		return false;
	}
	lv_indev_set_type(_indev, LV_INDEV_TYPE_POINTER);
	lv_indev_set_display(_indev, _disp);

	if (_config.touch.disabled) {
		reset_touch_input_state();
		lv_indev_set_read_cb(_indev, [](lv_indev_t *indev,
						lv_indev_data_t *data) {
			(void)indev;
			return read_tp_disabled(data);
		});
		spdlog::info("[lvgl] Touch input disabled by config");
	} else {
		_fd_tp = ::open(_config.touch.device.c_str(),
				O_RDONLY | O_NONBLOCK | O_CLOEXEC);
		if (_fd_tp < 0) {
			spdlog::error("open tpdev<{}> failed: {}",
				      _config.touch.device,
				      std::strerror(errno));
			cleanup();
			return false;
		}
		set_touch_input_fd(_fd_tp);
		reset_touch_wake_filter();
		if (_touch_calibration_mode) {
			lv_indev_set_read_cb(_indev, [](lv_indev_t *indev,
							lv_indev_data_t *data) {
				(void)indev;
				return read_tp(data, false);
			});
		} else {
			lv_indev_set_read_cb(_indev, [](lv_indev_t *indev,
							lv_indev_data_t *data) {
				(void)indev;
				return read_tp(data);
			});
		}
	}
#endif
	lv_indev_set_scroll_limit(_indev, kTouchScrollLimitPx);
	lv_indev_set_long_press_time(_indev, 800);
	if (_touch_click_marker_enabled) {
		lv_indev_add_event_cb(_indev, touch_click_marker_event_cb,
				      LV_EVENT_RELEASED, nullptr);
		spdlog::info("[lvgl] Touch click marker enabled");
	}
	if (wifi_latency_probe_enabled) {
		lv_indev_add_event_cb(_indev, touch_latency_probe_event_cb,
				      LV_EVENT_PRESSED, nullptr);
	}

	_init = true;
	return true;
}

int UILVGLApp::run()
{
	if (!_init) {
		return -1;
	}

	lv_obj_t *scr = lv_screen_active();
	lv_obj_set_style_bg_color(scr, lv_color_black(), 0);
	lv_obj_set_style_bg_opa(scr, LV_OPA_COVER, 0);

	const TouchDiagnosticMode diagnostic_mode = touch_diagnostic_mode();
	if (diagnostic_mode == TouchDiagnosticMode::Calibration) {
		return run_touch_calibration();
	}
	if (diagnostic_mode == TouchDiagnosticMode::PaintTest) {
		return run_touch_paint_test();
	}

	ui::BackgroundAssets background_assets;
	std::string auto_rotation_probe_reason;
#if NANOKVM_GO_SIMULATOR_BUILD
	_auto_rotation_imu_available = false;
	auto_rotation_probe_reason = "disabled in simulator mode";
#else
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	background_assets = ui::load_default_background_assets(
		ui::kDefaultBackgroundTargetWidth,
		ui::kDefaultBackgroundTargetHeight);
#endif
	_auto_rotation_imu_available = lvgl_main::probe_auto_rotation_imu(
		lvgl_main::AutoRotationRuntimeConfig {},
		auto_rotation_probe_reason);
#endif
	if (_auto_rotation_imu_available) {
		spdlog::info("[lvgl] IMU auto rotation available");
	} else {
		spdlog::info(
			"[lvgl] Manual display rotation fallback enabled: {}",
			auto_rotation_probe_reason.empty() ?
				"probe failed" :
				auto_rotation_probe_reason);
		if (_config.ui.manual_display_rotation) {
			apply_display_180_rotation(true);
		}
	}

	const lv_font_t *wifi_ssid_font = nullptr;
	ui::SsidFontRenderMode wifi_ssid_font_render_mode =
		ui::SsidFontRenderMode::CompiledFallback;
	_runtime_font_catalog =
		std::make_unique<lvgl_main::RuntimeFontCatalog>(
			_config.fonts.cjk_region);
	(void)_runtime_font_catalog->load();
	wifi_ssid_font = _runtime_font_catalog->ssid_font();
	wifi_ssid_font_render_mode = _runtime_font_catalog->ssid_render_mode();
	if (wifi_ssid_font_render_mode ==
	    ui::SsidFontRenderMode::RuntimeTinyTtf) {
		lvgl_main::SsidRasterWorkerConfig worker_config;
		worker_config.base_font_path =
			std::string(lvgl_main::RuntimeFontCatalog::noto_sans_path());
		worker_config.cjk_font_path = std::string(
			lvgl_main::RuntimeFontCatalog::cjk_font_path(
				_config.fonts.cjk_region));
		worker_config.font_size_px =
			lvgl_main::RuntimeFontCatalog::kSsidFontSize;
		try {
			_ssid_raster_worker =
				std::make_unique<lvgl_main::SsidRasterWorker>(
					std::move(worker_config),
					[this](ui::SsidPreparedRasterResult result) {
						schedule_ssid_raster_result(std::move(result));
					});
		} catch (const std::exception &error) {
			spdlog::warn("SSID raster worker unavailable: {}", error.what());
			_ssid_raster_worker.reset();
		}
	}

	_page_manager = std::make_unique<ui::PageManager>(
		std::move(background_assets),
		to_ui_page_transition_mode(_config.ui.page_transition_mode),
		true,
		static_cast<uint32_t>(
			_config.ui.wifi_boot_reconnect_timeout_ms),
		static_cast<uint32_t>(
			_config.ui.settings_state_boot_timeout_ms),
		_config.ui.experimental_settings_subpages,
		_config.ui.enable_settings_optimistic_actions,
		wifi_ssid_font,
		static_cast<uint32_t>(_config.fonts.ssid_image_cache_kb),
		resolve_ssid_scroll_enabled(
			_config.experimental.ssid_scroll,
			wifi_ssid_font_render_mode ==
				ui::SsidFontRenderMode::RuntimeTinyTtf),
		wifi_ssid_font_render_mode,
		to_ui_wifi_loading_mode(_config.ui.wifi_loading_progress));
	_page_manager->set_apps(lvgl_main::scan_apps(
		lvgl_main::simulator_policy::launcher_apps_path()));
	_page_manager->init(scr);
	_page_manager->bind_actions(build_page_manager_actions());
	_page_manager->set_apps_tile_visible(_config.launcher.enabled);
	_apps_refresh_timer =
		lv_timer_create(apps_refresh_timer_cb, 10'000, this);
	_page_manager->set_auto_rotation_manual_mode(
		!_auto_rotation_imu_available);
	_page_manager->set_runtime_auto_rotation_active(
		_auto_rotation_imu_available ?
			_config.ui.enable_auto_rotation :
			_display_180_rotation_enabled.load(
				std::memory_order_acquire));
	if (_auto_rotation_imu_available && _config.ui.enable_auto_rotation) {
		start_auto_rotation();
	}

	start_runtime_backend();

	if (_wifi_keyboard_test_mode) {
		_page_manager->show_wifi_config();
	}

#if !NANOKVM_GO_SIMULATOR_BUILD
	if (_config.http_api.enabled && _config.http_api.port > 0) {
		if (_target_runtime_service != nullptr) {
			lvgl_main::http_api::ServerConfig http_config {
				.bind_host = _config.http_api.bind_host,
				.port = _config.http_api.port,
				.token = _config.http_api.token,
			};
			_http_server = std::make_unique<HttpApiServer>(
				_target_runtime_service);
			if (!_http_server->start(http_config)) {
				spdlog::error(
					"Failed to start HTTP API server on {}:{}",
					http_config.bind_host,
					http_config.port);
				_http_server.reset();
			}
		} else {
			spdlog::warn(
				"HTTP API server not started because NanoKVM runtime backend is unavailable");
		}
	} else {
		spdlog::info("[lvgl] HTTP API server disabled by config");
	}
#endif

	const auto input_activity_started = std::chrono::steady_clock::now();
	_last_input_activity = input_activity_started;
	_last_input_activity_seq = touch_input_activity_sequence();
	_ssid_raster_install_gate.reset(_last_input_activity_seq,
					       input_activity_started);
	run_lvgl_event_loop(true);

	spdlog::info("[lvgl] Shutting down gracefully...");

	cleanup();

	return 0;
}

int UILVGLApp::run_touch_calibration()
{
	if (!_init) {
		return -1;
	}

	lv_obj_t *scr = lv_screen_active();
	if (scr == nullptr) {
		cleanup();
		return -1;
	}

	start_runtime_backend();

	TouchCalibrationBoard board(scr,
					  { kPanelPhysicalWidth,
					    kPanelPhysicalHeight });
	board.create();
	spdlog::info(
		"[lvgl] Touch calibration drawing board started via runtime path: device={}, framebuffer={}",
		_config.touch.device, _config.display.framebuffer_device);

	run_lvgl_event_loop(false);

	spdlog::info("[lvgl] Touch calibration drawing board exiting...");
	cleanup();
	return 0;
}

int UILVGLApp::run_touch_paint_test()
{
	if (!_init) {
		return -1;
	}

	lv_obj_t *scr = lv_screen_active();
	if (scr == nullptr) {
		cleanup();
		return -1;
	}

	start_runtime_backend();

	TouchPaintCoverageBoard board(scr,
					      { kPanelPhysicalWidth,
						kPanelPhysicalHeight });
	board.create();
	spdlog::info(
		"[lvgl] Touch paint coverage test started via runtime path: device={}, framebuffer={}",
		_config.touch.device, _config.display.framebuffer_device);

	run_lvgl_event_loop(false);

	spdlog::info("[lvgl] Touch paint coverage test exiting...");
	cleanup();
	return 0;
}

int lvgl_ui_main(const lvgl_main::AppConfig &config,
		 const std::filesystem::path &config_path,
		 bool touch_click_marker_enabled, bool wifi_keyboard_test_mode)
{
	std::filesystem::path effective_config_path = config_path;
#if NANOKVM_GO_SIMULATOR_BUILD
	const bool explicit_config_path =
		config_path !=
		std::filesystem::path(
			lvgl_main::app_config_defaults::kConfigPath.data());
	std::string simulator_path_error;
	if (!lvgl_main::simulator_policy::resolve_config_path(
		    config_path, explicit_config_path, effective_config_path,
		    simulator_path_error)) {
		spdlog::error("{}", simulator_path_error);
		return -1;
	}
#endif
	auto ui_config =
		lvgl_ui_effective_config(config, wifi_keyboard_test_mode);
	ui_config =
		lvgl_main::simulator_policy::make_effective_config(ui_config);

	UILVGLApp app(ui_config, effective_config_path, false,
		      touch_click_marker_enabled, wifi_keyboard_test_mode);

	if (!app.init()) {
		return -1;
	}

	return app.run();
}

int lvgl_touch_calibration_main(const lvgl_main::AppConfig &config,
				const std::filesystem::path &config_path)
{
#if NANOKVM_GO_SIMULATOR_BUILD
	(void)config;
	(void)config_path;
	spdlog::error("Touch calibration is unavailable in simulator mode");
	return -1;
#else
	auto calibration_config = config;
	calibration_config.runtime.touch_calibrate = true;
	calibration_config.runtime.touch_paint_test = false;
	return lvgl_ui_main(calibration_config, config_path);
#endif
}

int lvgl_touch_paint_test_main(const lvgl_main::AppConfig &config,
			       const std::filesystem::path &config_path)
{
#if NANOKVM_GO_SIMULATOR_BUILD
	(void)config;
	(void)config_path;
	spdlog::error("Touch paint test is unavailable in simulator mode");
	return -1;
#else
	auto paint_config = config;
	paint_config.runtime.touch_calibrate = false;
	paint_config.runtime.touch_paint_test = true;
	return lvgl_ui_main(paint_config, config_path);
#endif
}
