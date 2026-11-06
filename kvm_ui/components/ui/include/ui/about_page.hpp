#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <string>

#include "ui/bcona.hpp"
#include "ui/scona.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class HoldConfirmBehavior;

class AboutPage : public SubPageBase {
    public:
	using ResetCallback = std::function<bool()>;

	AboutPage();
	~AboutPage() override;

	void destroy() override;
	void on_pause() override;
	void on_exit() override;
	void set_versions(const std::string &app, const std::string &fw,
			  const std::string &hw);
	void set_factory_reset_pending(bool pending);
	bool is_factory_reset_pending() const
	{
		return _factory_reset_pending;
	}

	ResetCallback on_reset_click;

	std::function<void()> on_app_ver_secret_click;

	static constexpr uint32_t SECRET_CLICK_WINDOW_MS = 2000;
	static constexpr uint8_t SECRET_CLICK_TARGET = 5;

    private:
	lv_obj_t *_version_group = nullptr;
	SConA *_hw_ver_label = nullptr;
	SConA *_app_ver_label = nullptr;
	BConA *_reset_btn = nullptr;
	std::unique_ptr<HoldConfirmBehavior> _reset_hold_behavior;
	bool _factory_reset_pending = false;
	std::string _hw_version = "--";
	std::string _app_version = "1.0.1";

	uint8_t _app_ver_click_count = 0;
	uint32_t _last_app_ver_click_tick = 0;

	void reset_app_ver_click_state();
	void handle_app_ver_click();
	static void app_ver_click_event_cb(lv_event_t *e);

	void build_content() override;
	void update_versions();
	void on_subpage_destroy() override;
};

} // namespace ui
