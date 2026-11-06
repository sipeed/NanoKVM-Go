#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "ui/subpage_base.hpp"
#include "ui/wifi_loading_mode.hpp"
#include "ui/wifi_ssid_font.hpp"

namespace ui
{

class SsidTextImageCache;
class SsidTextImageArena;
class SsidRasterMarquee;

struct WifiScanResult {
	std::string ssid;
	bool eap = false;
	int signal { 0 };
};

class WifiSsidPage : public SubPageBase {
    public:
	enum class ViewState : uint8_t {
		Loading,
		Results,
		Empty,
		Error,
	};

	explicit WifiSsidPage(const lv_font_t *ssid_font = nullptr,
				      uint32_t ssid_image_cache_kb = 0,
				      bool enable_ssid_scroll = false,
				      SsidFontRenderMode ssid_font_render_mode =
					      SsidFontRenderMode::CompiledFallback,
				      WifiLoadingMode wifi_loading_mode =
					      WifiLoadingMode::Progress);
	~WifiSsidPage() override;

	void destroy() override;
	void on_enter() override;
	void on_exit() override;
	void on_pause() override;
	void on_resume() override;
	void set_loading(std::string_view message = "Scanning...");
	void set_scan_results(const std::vector<WifiScanResult> &results);
	void set_empty(std::string_view message = "No WiFi found");
	void set_error(std::string_view message);
	void set_show_eap_unsupported(bool show);

	std::function<void(std::string_view)> on_ssid_selected;
	std::function<void()> on_manual_ssid_click;

    private:
	static constexpr uint32_t ACTIVE_COLOR = 0x2B7FFF;
	static constexpr uint32_t ROLLER_BG_COLOR = 0x000000;
	static constexpr uint32_t ROLLER_SELECTED_COLOR = 0x44403B;
	static constexpr uint16_t TEXT_IMAGE_ROW_HEIGHT = 32;
	static constexpr uint16_t TEXT_IMAGE_ROW_COUNT = 3;
	static constexpr uint32_t CACHE_RENDER_PERIOD_MS = 10;

	enum class CachePreloadPhase : uint8_t {
		Inactive,
		Present,
		Rasterize,
	};

	enum class OptionType : uint8_t {
		Manual,
		Network,
		Unsupported,
	};

	struct RollerOption {
		std::string label;
		std::string ssid;
		OptionType type = OptionType::Manual;
	};

	ViewState _state = ViewState::Loading;
	std::vector<WifiScanResult> _results;
	std::vector<RollerOption> _options;
	std::vector<std::unique_ptr<Component>> _controls;
	std::string _roller_options_text;
	bool _show_eap_unsupported = true;
	uint32_t _selected_index = 0;
	lv_obj_t *_navigation_box = nullptr;
	lv_obj_t *_loading_box = nullptr;
	lv_obj_t *_loading_arc = nullptr;
	lv_obj_t *_roller = nullptr;
	lv_obj_t *_selected_marquee_overlay = nullptr;
	lv_obj_t *_selected_marquee_label = nullptr;
	lv_timer_t *_loading_timer = nullptr;
	lv_timer_t *_cache_timer = nullptr;
	lv_timer_t *_cache_scroll_hold_timer = nullptr;
	uint16_t _loading_rotation = 0;
	enum class LoadingPhase : uint8_t {
		Idle,
		Climbing,
		Waiting,
		Rendering,
		Finishing,
		Done,
	};
	LoadingPhase _loading_phase { LoadingPhase::Idle };
	uint32_t _loading_start_tick { 0 };
	uint32_t _loading_ready_tick { 0 };
	uint8_t _loading_progress { 0 };
	uint32_t _cache_render_total { 0 };
	uint8_t _loading_finish_base { 0 };
	const lv_font_t *_ssid_font { nullptr };
	const lv_font_t *_visible_ssid_font { nullptr };
	uint32_t _ssid_image_cache_kb { 0 };
	SsidFontRenderMode _ssid_font_render_mode {
		SsidFontRenderMode::CompiledFallback
	};
	bool _enable_ssid_scroll { false };
	WifiLoadingMode _wifi_loading_mode { WifiLoadingMode::None };
	bool _cache_page_active { false };
	bool _cache_content_ready { false };
	bool _cache_scroll_hold { false };
	bool _roller_dragged { false };
	bool _selected_marquee_hides_native_text { false };
	bool _cache_activation_failed { false };
	uint32_t _selected_marquee_generation { 0 };
	uint32_t _selected_marquee_pending_generation { 0 };
	uint32_t _selected_marquee_pending_index { 0 };
	std::string _selected_marquee_pending_text;
	CachePreloadPhase _cache_preload_phase { CachePreloadPhase::Inactive };
	bool _cache_timing_active { false };
	uint32_t _cache_result_started_tick { 0 };
	uint32_t _last_loading_presentation_tick { 0 };
	uint32_t _max_loading_presentation_gap_ms { 0 };
	bool _loading_presentation_clock_started { false };
	std::unique_ptr<SsidTextImageArena> _text_image_arena;
	std::unique_ptr<SsidTextImageCache> _text_image_cache;
	std::unique_ptr<SsidRasterMarquee> _selected_raster_marquee;

	void build_content() override;
	void on_subpage_destroy() override;
	void create_navigation();
	void create_loading_indicator();
	void create_roller();
	void create_selected_marquee();
	bool ensure_selected_raster_marquee();
	void release_selected_raster_marquee();
	bool ensure_text_image_cache();
	void release_text_image_cache();
	void rebuild_options();
	void update_view();
	void update_roller_options();
	bool loading_presentation_active() const noexcept;
	bool initial_cache_preload_active() const noexcept;
	void activate_selected_option();
	void start_loading_animation();
	void stop_loading_animation();
	void advance_loading_presentation();
	void advance_loading_for_mode();
	void advance_loading_progress();
	void recompute_rendering_progress() noexcept;
	void initialize_loading_progress();
	void apply_loading_geometry_for_mode();
	void reset_cache_scheduler() noexcept;
	void reset_loading_progress() noexcept;
	void reset_loading_session() noexcept;
	void queue_text_image_window();
	void render_next_text_image();
	void draw_text_image_rows(lv_event_t *event);
	bool cached_text_row_area(uint32_t option_index,
					  lv_area_t *area) const;
	void update_selected_marquee();
	void hide_selected_marquee();
	void cancel_selected_marquee_request() noexcept;
	bool selected_raster_work_pending() const noexcept;
	bool cache_timer_work_pending() const noexcept;
	bool service_one_selected_raster_work();
	void start_selected_label_scroll();
	void stop_selected_label_scroll();
	bool selected_marquee_visible() const noexcept;
	void set_native_selected_text_visible(bool visible);
	void mark_cache_content_ready();
	void start_cache_timer();
	void stop_cache_timer();
	void start_cache_drag_hold();
	void start_cache_scroll_hold();
	void finish_cache_drag_hold();
	void stop_cache_scroll_hold();
	bool cache_scroll_hold_active() const noexcept;
	void disable_text_image_cache();
	void apply_roller_text_style();
	std::vector<std::string> foreground_text_labels() const;
	std::vector<std::string> speculative_text_labels() const;
	lv_obj_t *roller_label() const noexcept;
	void set_cache_roller_label_width();
	void reset_roller_label_width();
	bool text_image_cache_enabled() const noexcept;
	bool runtime_tiny_ttf_mode() const noexcept;
	void reset_lvgl_pointers();
	const RollerOption *current_option() const;

	static void roller_event_cb(lv_event_t *event);
	static void roller_pressed_cb(lv_event_t *event);
	static void roller_pressing_cb(lv_event_t *event);
	static void roller_released_cb(lv_event_t *event);
	static void roller_draw_post_cb(lv_event_t *event);
	static void loading_timer_cb(lv_timer_t *timer);
	static void cache_timer_cb(lv_timer_t *timer);
	static void cache_scroll_hold_timer_cb(lv_timer_t *timer);

	friend class WifiSsidPageTestAccess;
};

} // namespace ui
