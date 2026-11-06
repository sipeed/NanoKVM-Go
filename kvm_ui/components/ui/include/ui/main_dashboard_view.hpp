#pragma once

#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "ui/background_assets.hpp"
#include "ui/root_layout.hpp"

namespace ui
{

class MainDashboardView final {
    public:
	MainDashboardView() = default;
	~MainDashboardView();

	MainDashboardView(const MainDashboardView &) = delete;
	MainDashboardView &operator=(const MainDashboardView &) = delete;

	void create(lv_obj_t *parent, const BackgroundImage *background);
	void begin_shutdown();
	void shutdown();

	bool is_created() const noexcept;
	lv_obj_t *content_object() const noexcept;
	void replace_background(const BackgroundImage *background);

	void set_mouse_active(bool active);

	void set_resolution_text(const char *text);
	void set_fps_text(const char *text);
	void set_fps_unit_text(const char *text);
	void set_keyboard_special_text(const char *text);
	void set_keyboard_normal_text(const char *text);
	void set_ip_empty_placeholder(std::string_view text);
	void set_ip_candidates(const std::vector<std::string> &candidates);

    private:
	static constexpr uint32_t kDisplayMetricRefreshMs = 333;
	static constexpr uint32_t kIpReconnectAnimationPeriodMs = 450;
	static constexpr uint32_t kMouseIdleTimeoutMs = 800;

	bool _is_shutting_down = false;
	lv_obj_t *_content_obj = nullptr;
	lv_obj_t *_background_obj = nullptr;
	lv_obj_t *_ip_label = nullptr;
	lv_obj_t *_resolution_label = nullptr;
	lv_obj_t *_resolution_obj = nullptr;
	lv_obj_t *_fps_value_label = nullptr;
	lv_obj_t *_fps_value_obj = nullptr;
	lv_obj_t *_fps_unit_label = nullptr;
	lv_obj_t *_fps_unit_obj = nullptr;
	lv_obj_t *_keyboard_special_label = nullptr;
	lv_obj_t *_keyboard_normal_label = nullptr;
	lv_obj_t *_video_status_obj = nullptr;
	lv_obj_t *_mouse_indicator_obj = nullptr;
	lv_obj_t *_hid_status_obj = nullptr;
	lv_timer_t *_mouse_idle_timer = nullptr;
	lv_timer_t *_display_metric_timer = nullptr;
	lv_timer_t *_ip_reconnect_animation_timer = nullptr;

	std::string _resolution_text = "--";
	std::string _pending_resolution_text = "--";
	std::vector<std::string> _ip_candidates;
	std::string _ip_empty_placeholder = "--";
	uint8_t _ip_reconnect_dot_count = 3;
	bool _ip_reconnect_dots_increasing = false;
	bool _ip_reconnect_animation_active = false;
	std::string _fps_text = "--";
	std::string _pending_fps_text = "--";
	std::string _keyboard_special_text = "-";
	std::string _keyboard_normal_text = "-";
	bool _resolution_dirty = false;
	bool _fps_dirty = false;
	bool _mouse_active = false;
	bool _hid_data_available = false;

	void flush_display_metrics();
	void render_resolution_text();
	void render_fps_text();
	void render_keyboard_texts();
	void render_ip_candidate();
	void update_ip_font(bool reconnecting);
	void start_ip_reconnect_animation();
	void stop_ip_reconnect_animation();
	void render_ip_reconnect_animation_frame();
	void apply_mouse_indicator_visual();
	void set_resolution_label_text(const char *text);
	void set_fps_label_text(const char *text);
	void set_keyboard_special_label_text(const char *text);
	void set_keyboard_normal_label_text(const char *text);

	static void mouse_idle_timer_cb(lv_timer_t *timer);
	static void display_metric_timer_cb(lv_timer_t *timer);
	static void ip_reconnect_animation_timer_cb(lv_timer_t *timer);
};

} // namespace ui
