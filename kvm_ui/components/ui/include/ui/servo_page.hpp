#pragma once

#include <memory>
#include <vector>

#include <lvgl.h>

#include "ui/bconb.hpp"
#include "ui/servo_pwm.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class ServoPage : public SubPageBase {
    public:
	ServoPage();
	explicit ServoPage(std::unique_ptr<servo_pwm::Backend> pwm_backend);
	~ServoPage() override = default;

	void destroy() override;
	void on_pause() override;
	void on_exit() override;

    private:
	static constexpr uint32_t FREE_COLOR = 0x00C950;
	static constexpr uint32_t CLICK_COLOR = 0x2B7FFF;
	static constexpr uint32_t SLIDER_COLOR = 0xFFB900;
	static constexpr int32_t SLIDER_WIDTH = BConA::WIDTH;
	static constexpr int32_t SLIDER_HEIGHT = 64;
	static constexpr int32_t SLIDER_RADIUS = 32;
	static constexpr int32_t KNOB_SIZE = 30;

	std::unique_ptr<servo_pwm::Backend> _pwm_backend;
	int _idle_ns = servo_pwm::kDefaultIdleNs;
	int _zero_ns = servo_pwm::kDefaultIdleNs;
	bool _free_mode = false;
	bool _initialized_pwm = false;
	bool _suppress_next_click = false;
	std::vector<servo_pwm::Step> _click_plan;
	std::size_t _click_step_index = 0;
	lv_timer_t *_click_timer = nullptr;

	lv_obj_t *_slider_row = nullptr;
	lv_obj_t *_slider_track = nullptr;
	lv_obj_t *_slider_fill = nullptr;
	lv_obj_t *_slider_knob = nullptr;
	lv_obj_t *_button_row = nullptr;
	BConB *_free_btn = nullptr;
	BConB *_click_btn = nullptr;

	void build_content() override;
	void on_subpage_destroy() override;
	void update_slider_visual();
	void update_title();
	void update_zero_label();
	void update_button_state();
	int idle_from_local_x(int32_t local_x) const;
	void set_idle_ns(int idle_ns, bool apply_free_mode);
	void set_zero_from_current_idle();
	void apply_free_idle();
	void start_click_sequence();
	void stop_click_sequence();
	void interrupt_click_sequence();
	bool has_active_click_sequence() const;
	void run_click_step();
	void log_pwm_result(const char *action,
			    const servo_pwm::ApplyResult &result);

	friend class ServoPageTestAccess;

	static void slider_event_cb(lv_event_t *event);
	static void click_long_press_event_cb(lv_event_t *event);
	static void click_timer_cb(lv_timer_t *timer);
};

} // namespace ui
