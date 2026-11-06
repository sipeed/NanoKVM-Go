#pragma once

#include <cstdint>
#include <functional>
#include <string>
#include <string_view>

#include "ui/bcona.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class WifiConnectResultPage : public SubPageBase {
    public:
	enum class State : uint8_t {
		Loading,
		Success,
		Failure,
	};

	WifiConnectResultPage();
	~WifiConnectResultPage() override = default;

	void destroy() override;
	void set_loading(std::string_view message = "");
	void set_success(std::string_view message = "Connected");
	void set_failure(std::string_view message = "Connect failed");
	State state() const
	{
		return _state;
	}

	std::function<void()> on_success_return_request;

    private:
	static constexpr uint32_t ACTIVE_COLOR = 0x2B7FFF;
	static constexpr uint32_t FAILURE_TEXT_COLOR = 0xFFB900;
	static constexpr uint32_t SUCCESS_COLOR = 0x00C950;

	State _state = State::Loading;
	std::string _message;
	bool _success_return_scheduled = false;
	uint16_t _spinner_rotation = 0;
	lv_obj_t *_status_box = nullptr;
	lv_obj_t *_status_label = nullptr;
	lv_obj_t *_spinner_box = nullptr;
	lv_obj_t *_spinner_arc = nullptr;
	lv_timer_t *_spinner_timer = nullptr;

	void build_content() override;
	void on_subpage_destroy() override;
	void reset_lvgl_pointers();
	void create_status();
	void create_spinner();
	void update_view();
	void update_back_enabled();
	void start_spinner();
	void stop_spinner();
	void schedule_success_return();
	void cancel_success_return();
	uint32_t spinner_color() const;

	static void spinner_timer_cb(lv_timer_t *timer);
	static void success_return_async_cb(void *user_data);
};

} // namespace ui
