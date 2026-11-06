#pragma once

#include <cstdint>
#include <functional>
#include <memory>

#include "ui/bcona.hpp"
#include "ui/region_page_policy.hpp"
#include "ui/scona.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

class HoldConfirmBehavior;

class RegionPage : public SubPageBase {
    public:
	using SwitchCallback = std::function<bool(RegionOption target)>;

	static constexpr uint32_t SWITCH_HOLD_DURATION_MS = 2000;
	static constexpr uint32_t SWITCH_FILL_COLOR = 0xFFB900;

	RegionPage();
	~RegionPage() override;

	void destroy() override;
	void on_pause() override;
	void on_exit() override;

	void set_region(RegionOption region);
	void set_switch_pending(bool pending);
	void clear_switch_pending()
	{
		set_switch_pending(false);
	}
	bool is_switch_pending() const
	{
		return _switch_pending;
	}

	SwitchCallback on_switch_request;

    protected:
	void build_content() override;

    private:
	SConA *_region_value = nullptr;
	BConA *_switch_btn = nullptr;
	std::unique_ptr<HoldConfirmBehavior> _switch_hold_behavior;
	RegionOption _projected_region = RegionOption::Global;
	RegionOption _pending_target = RegionOption::Global;
	bool _switch_pending = false;

	void apply_region_value(RegionOption region);
	void on_subpage_destroy() override;
};

} // namespace ui
