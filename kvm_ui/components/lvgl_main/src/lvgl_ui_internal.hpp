#pragma once

#include <atomic>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <csignal>
#include <filesystem>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <vector>
#include <sys/types.h>

#include <lvgl.h>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/launcher_exit_confirmation.hpp"
#include "lvgl_main/runtime_service.hpp"
#include "lvgl_main/runtime_status_mailbox.hpp"
#include "lvgl_main/ssid_raster_install_gate.hpp"
#include "lvgl_main/ssid_raster_result_mailbox.hpp"
#include "lvgl_main/ssid_raster_worker.hpp"
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
#include "ui/background_assets.hpp"
#endif

#if !NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/framebuffer_snapshot.hpp"
#include "lvgl_main/screenshot_control.hpp"
#endif
#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_app_launcher.hpp"
#include "lvgl_main/simulator_build_config.hpp"
#endif

namespace lvgl_main
{

void install_color_lut_flush(lv_display_t *disp);
void restore_color_lut_flush(lv_display_t *disp);
void reset_color_lut_flush();

class AutoRotationRuntime;
class NanoKvmRuntimeService;
class RuntimeFontCatalog;
class RuntimeService;

} // namespace lvgl_main

class HttpApiServer;

namespace ui
{

class PageManager;
struct PageManagerActions;

} // namespace ui

extern volatile std::sig_atomic_t g_running;
#if !NANOKVM_GO_SIMULATOR_BUILD
extern volatile std::sig_atomic_t g_snapshot_requested;
bool consume_snapshot_request();
#endif

class LvLockGuard final {
    public:
	LvLockGuard()
	{
		lv_lock();
	}
	~LvLockGuard()
	{
		lv_unlock();
	}

	LvLockGuard(const LvLockGuard &) = delete;
	LvLockGuard &operator=(const LvLockGuard &) = delete;
};

inline constexpr int32_t kPanelPhysicalWidth = 284;
inline constexpr int32_t kPanelPhysicalHeight = 240;
inline constexpr int32_t kPanelVisibleWidth = 270;
inline constexpr int32_t kPanelVisibleHeight = 240;
inline constexpr int32_t kPanelLeftDeadAreaWidth =
	kPanelPhysicalWidth - kPanelVisibleWidth;
#if NANOKVM_GO_SIMULATOR_BUILD
inline constexpr int32_t kSimulatorLogicalWidth =
	lvgl_main::simulator_build_config::kLogicalWidth;
inline constexpr int32_t kSimulatorLogicalHeight =
	lvgl_main::simulator_build_config::kLogicalHeight;
inline constexpr uint8_t kSimulatorWindowZoom =
	static_cast<uint8_t>(lvgl_main::simulator_build_config::kWindowZoom);
static_assert(kSimulatorLogicalWidth == kPanelVisibleWidth);
static_assert(kSimulatorLogicalHeight == kPanelVisibleHeight);
static_assert(kSimulatorWindowZoom == 2U);
#endif

inline constexpr int32_t kTouchClickMarkerRadius = 2;
inline constexpr uint32_t kFbdevBufferCount = 2U;
inline constexpr uint32_t kFbdevBufferLines = 256U;

struct TouchClickMarkerState {
	std::atomic<bool> enabled { false };
	std::atomic<bool> visible { false };
	std::atomic<int32_t> x { 0 };
	std::atomic<int32_t> y { 0 };
	std::atomic<uint64_t> revision { 0 };
};

extern TouchClickMarkerState touch_click_marker_state;

namespace lvgl_main::wifi_latency_probe
{

inline constexpr const char *kSentinelPath = "/tmp/kvm_ui_wifi_latency_probe";
inline constexpr const char *kMarkerSentinelPath =
	"/tmp/kvm_ui_wifi_latency_probe_marker";

void initialize();
void reset();
[[nodiscard]] bool enabled();
[[nodiscard]] bool visible_marker_enabled();
void set_event_clock_monotonic(bool monotonic);
[[nodiscard]] int64_t timestamp_now_us();
void record_touch_press_event(int64_t event_timestamp_us,
			      int64_t arrival_timestamp_us);
void record_input_read(int64_t read_timestamp_us);
void record_lvgl_pressed(int64_t handler_timestamp_us);
void record_visible_flush(const lv_display_t *disp, const lv_area_t *area);
void record_flush(uint64_t wrapper_duration_us,
			  uint64_t original_duration_us, uint64_t pixels);
void record_timer_handler(uint64_t duration_us);
void maybe_log();

} // namespace lvgl_main::wifi_latency_probe

inline constexpr uint8_t kTouchScrollLimitPx = 4;

void reset_touch_wake_filter();
void reset_touch_input_state();
void set_touch_input_fd(int fd);
void drain_touch_input_events(int fd);
[[nodiscard]] int touch_input_last_x();
[[nodiscard]] int touch_input_last_y();
[[nodiscard]] bool touch_input_is_pressed();
[[nodiscard]] uint64_t touch_input_activity_sequence();
void record_touch_input_activity();
void set_touch_input_screen_on(bool screen_on);
void read_tp(lv_indev_data_t *data, bool suppress_wake_touch = true);
void read_tp_disabled(lv_indev_data_t *data);
void touch_click_marker_event_cb(lv_event_t *event);
void touch_latency_probe_event_cb(lv_event_t *event);
lv_area_t touch_click_marker_area(int32_t x, int32_t y);
void map_raw_to_logical(const lv_display_t *disp, int raw_x, int raw_y,
			int &out_x, int &out_y);

#if !NANOKVM_GO_SIMULATOR_BUILD
void clear_framebuffer(const std::string &dev);

struct ExitConfirmationOverlay final {
	bool open(const std::string &dev, int rot);
	void close();
	void snapshot();
	void animate_in();
	void restore();
	[[nodiscard]] lvgl_main::LauncherExitConfirmationChoice
	choice_at(int logical_x, int logical_y) const;

	void to_phys(int logical_x, int logical_y, int &physical_x,
		     int &physical_y) const;
	bool offset_of(int logical_x, int logical_y, size_t &offset) const;
	bool frame_index(int logical_x, int logical_y, size_t &index) const;
	void begin_frame();
	void commit_frame();
	void set_pixel(int logical_x, int logical_y, uint16_t color);
	void blend_pixel(int logical_x, int logical_y, uint16_t color,
			 uint8_t opacity);
	void fill_rounded_rect(const lvgl_main::LauncherExitRect &rect,
			       int radius, uint16_t color,
			       uint8_t opacity = 255);
	void draw_text_centered(const char *text, int center_x, int top,
				int scale, uint16_t color,
				uint8_t opacity = 255);
	void draw_frame(
		const lvgl_main::LauncherExitConfirmationAnimationFrame &animation);
	void draw();

	static uint16_t blend_rgb565(uint16_t background, uint16_t foreground,
				     uint8_t opacity);
	static int scale_position(int value, int center, uint8_t scale_percent);
	static lvgl_main::LauncherExitRect scale_rect(
		const lvgl_main::LauncherExitRect &rect,
		const lvgl_main::LauncherExitRect &dialog, uint8_t scale_percent);
	static const uint8_t *glyph_rows(char character);

	uint8_t *base { nullptr };
	size_t len { 0 };
	int stride { 0 };
	int xres { 0 };
	int yres { 0 };
	int rotate { 90 };
	int vw { 0 };
	int vh { 0 };
	std::vector<uint8_t> backup;
	std::vector<uint16_t> frame;
	lvgl_main::LauncherExitConfirmationLayout layout;
	bool snapped { false };
};
#endif

class UILVGLApp final {
    public:
	explicit UILVGLApp(const lvgl_main::AppConfig &config);
	UILVGLApp(const lvgl_main::AppConfig &config,
		  std::filesystem::path config_path,
		  bool touch_calibration_mode = false,
		  bool touch_click_marker_enabled = false,
		  bool wifi_keyboard_test_mode = false);
	~UILVGLApp();

	bool init();
	int run();
	int run_touch_calibration();
	int run_touch_paint_test();

    private:
	enum class TouchDiagnosticMode : uint8_t {
		None,
		Calibration,
		PaintTest,
	};

	bool setup_fbdev_fixed_buffers();
	TouchDiagnosticMode touch_diagnostic_mode() const;
	lvgl_main::RuntimeCallbacks make_runtime_callbacks();
	ui::PageManagerActions build_page_manager_actions();
	void bind_runtime_actions();
	void start_runtime_backend();
	void run_lvgl_event_loop(bool enable_runtime_policies);
#if !NANOKVM_GO_SIMULATOR_BUILD
	void run_app_session(const std::string &script);
#endif
	void schedule_runtime_status_apply(
		const lvgl_main::RuntimeStatusPublication &publication);
	void apply_pending_runtime_status();
	void schedule_ssid_raster_result(
		ui::SsidPreparedRasterResult result);
	void apply_pending_ssid_raster_result();
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	void schedule_background_assets_apply(ui::BackgroundAssets assets);
	void apply_pending_background_assets();
#endif
	void apply_runtime_status(const lvgl_main::RuntimeStatus &status);
	void sync_wifi_scan_status(const lvgl_main::RuntimeStatus &status);
	void sync_wifi_connect_status(const lvgl_main::RuntimeStatus &status);
	void update_input_activity();
	void update_ssid_raster_install_gate();
	[[nodiscard]] bool should_defer_ssid_raster_install() const;
	void handle_auto_sleep();
	void set_auto_rotation_enabled(bool enabled);
	void set_manual_display_rotation_enabled(bool enabled);
	void apply_display_180_rotation(bool enabled);
	void request_display_180_rotation(bool enabled);
	bool start_display_rotation_fade(bool enabled);
	bool start_display_rotation_fade_animation(int32_t from_opa,
						   int32_t to_opa,
						   bool fading_out);
	void finish_display_rotation_fade_out();
	void finish_display_rotation_fade_in();
	void cleanup_display_rotation_fade();
	void start_auto_rotation();
	void stop_auto_rotation();
	void handle_auto_rotation();
	void configure_display_geometry();
	void handle_touch_click_marker();
	void release_draw_buffers();
	void cleanup();
#if !NANOKVM_GO_SIMULATOR_BUILD
	struct ScreenshotSignalRegistration {
		int signal { 0 };
		bool installed { false };
	};

	bool configure_screenshot_control();
	void release_screenshot_control();
	lvgl_main::framebuffer_snapshot::ViewRotation
		snapshot_view_rotation() const;
	void save_snapshot_capture(
		lvgl_main::framebuffer_snapshot::CaptureResult capture,
		std::chrono::system_clock::time_point captured_at);
	void handle_snapshot_request();
	bool handle_launcher_snapshot_request(pid_t &child, bool child_stopped);
#endif
	static void runtime_status_apply_async_cb(void *user_data);
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	static void background_assets_apply_async_cb(void *user_data);
#endif
	static void apps_refresh_timer_cb(lv_timer_t *timer);
	static void display_rotation_fade_opa_exec_cb(void *var, int32_t value);
	static void display_rotation_fade_ready_cb(lv_anim_t *anim);
#if NANOKVM_GO_SIMULATOR_BUILD
	static void simulator_display_delete_event_cb(lv_event_t *event);
	static void simulator_input_activity_event_cb(lv_event_t *event);
#endif

	bool _init { false };
	bool _lvgl_initialized { false };
	bool _sdl_quit_pending { false };
	lv_display_t *_disp { nullptr };
	int _fd_tp { -1 };
	lv_indev_t *_indev { nullptr };
	lv_timer_t *_apps_refresh_timer { nullptr };
	std::unique_ptr<lvgl_main::RuntimeFontCatalog> _runtime_font_catalog;
	std::unique_ptr<lvgl_main::SsidRasterWorker> _ssid_raster_worker;
	std::unique_ptr<ui::PageManager> _page_manager;
#if !NANOKVM_GO_SIMULATOR_BUILD
	std::unique_ptr<HttpApiServer> _http_server;
	lvgl_main::NanoKvmRuntimeService *_target_runtime_service { nullptr };
#endif
	std::unique_ptr<lvgl_main::RuntimeService> _runtime_service;
	std::unique_ptr<lvgl_main::AutoRotationRuntime> _auto_rotation_runtime;
	lvgl_main::AppConfig _config;
	std::filesystem::path _config_path;
#if !NANOKVM_GO_SIMULATOR_BUILD
	std::optional<lvgl_main::screenshot_control::ScreenshotInstanceLease>
		_screenshot_instance_lease;
	ScreenshotSignalRegistration _screenshot_user1_signal;
	ScreenshotSignalRegistration _screenshot_user2_signal;
#endif
#if NANOKVM_GO_SIMULATOR_BUILD
	lvgl_main::SimulatorAppLauncher _simulator_app_launcher;
#else
	std::atomic<bool> _launch_request { false };
	std::mutex _launch_mtx;
	std::string _launch_path;
#endif
	lv_obj_t *_display_rotation_fade_overlay { nullptr };
	bool _touch_calibration_mode { false };
	TouchDiagnosticMode _touch_diagnostic_mode { TouchDiagnosticMode::None };
	bool _touch_click_marker_enabled { false };
	bool _wifi_keyboard_test_mode { false };
	bool _auto_rotation_imu_available { false };
	bool _display_rotation_fade_active { false };
	bool _display_rotation_fade_fading_out { false };
	bool _display_rotation_fade_target_reverse { false };
	int _display_rotation_fade_queued_reverse { -1 };
	bool _touch_click_marker_was_visible { false };
	int32_t _touch_click_marker_x { 0 };
	int32_t _touch_click_marker_y { 0 };
	uint64_t _touch_click_marker_revision { 0 };
	std::mutex _runtime_status_mutex;
	lvgl_main::RuntimeStatusMailbox _runtime_status_mailbox;
	bool _runtime_status_apply_pending { false };
	lvgl_main::SsidRasterResultMailbox _ssid_raster_result_mailbox;
	lvgl_main::SsidRasterInstallGate _ssid_raster_install_gate;
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	std::mutex _background_assets_mutex;
	ui::BackgroundAssets _pending_background_assets;
	bool _background_assets_apply_pending { false };
	bool _background_assets_mailbox_closed { false };
#endif
	std::atomic<bool> _auto_sleep_policy_enabled { false };
	std::atomic<bool> _display_180_rotation_enabled { false };
	std::atomic<int> _auto_rotation_pending_reverse { -1 };
	std::atomic<bool> _runtime_screen_on { true };
	std::chrono::steady_clock::time_point _last_input_activity {
		std::chrono::steady_clock::now()
	};
	uint64_t _last_input_activity_seq { 0 };
	bool _auto_sleep_policy_was_enabled { false };
	bool _auto_sleep_sleep_request_active { false };
	bool _auto_sleep_screen_off_observed { false };
	bool _auto_sleep_wake_request_active { false };
	bool _last_wifi_scan_running { false };
	bool _wifi_scan_result_pending { false };
	bool _last_wifi_connect_running { false };
	bool _wifi_connect_result_pending { false };
	lvgl_main::WifiConnectStatus _last_wifi_connect_status {
		lvgl_main::WifiConnectStatus::Unknown
	};
	std::string _last_wifi_connect_ssid;
	std::string _last_wifi_connect_message;
	uint64_t _last_hid_keyboard_special_label_generation { 0 };
	uint64_t _last_hid_keyboard_normal_label_generation { 0 };
	uint64_t _last_hid_mouse_state_generation { 0 };
	uint64_t _last_factory_reset_failure_generation { 0 };
	uint64_t _last_region_switch_failure_generation { 0 };
	uint8_t *_draw_buf_1 { nullptr };
	uint8_t *_draw_buf_2 { nullptr };
	size_t _draw_buf_size { 0 };
};
