#pragma once

#include <cstdint>
#include <functional>
#include <string>
#include <string_view>

#include "ui/page.hpp"

namespace ui
{

class WifiQrPage : public Page {
    public:
	WifiQrPage();
	~WifiQrPage() override = default;

	void create(lv_obj_t *parent) override;
	void destroy() override;
	void on_enter() override;
	void on_exit() override;
	void set_title(std::string_view title);
	void set_qr_payload(const std::string &payload);
	void set_ap_stop_completed_generation(uint64_t generation);

	std::function<void()> on_close_ready;
	std::function<void()> on_qr_start_request;
	std::function<uint64_t()> on_qr_stop_request;

    private:
	static constexpr const char *DEFAULT_TITLE = "Configure WiFi";
	static constexpr const char *EMPTY_QR_PLACEHOLDER =
		"Wi-Fi credentials are not configured";

	lv_obj_t *_qr_title_label = nullptr;
	lv_obj_t *_qr_background = nullptr;
	lv_obj_t *_qr_code = nullptr;
	lv_obj_t *_loading_arc = nullptr;
	lv_timer_t *_loading_timer = nullptr;
	std::string _qr_title = DEFAULT_TITLE;
	std::string _qr_payload;
	bool _closing = false;
	bool _close_ready_scheduled = false;
	uint64_t _stop_request_generation = 0;
	uint16_t _loading_rotation = 0;

	void refresh_qr_code();
	void reset_session();
	void begin_closing();
	void start_loading_animation();
	void stop_loading_animation();
	void schedule_close_ready();
	void notify_close_ready();
	static void click_event_cb(lv_event_t *event);
	static void loading_timer_cb(lv_timer_t *timer);
	static void close_ready_async_cb(void *user_data);
};

} // namespace ui
