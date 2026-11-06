#pragma once

#include <cstdint>

namespace ui
{

/**
 * @brief 统一的控件视觉状态切换接口（default/active）
 */
class ControlState {
    public:
	enum class VisualState {
		DEFAULT,
		ACTIVE
	};

	enum class ApplyPolicy {
		DEFERRED, ///< 仅记录状态，下一轮 UI 线程异步应用
		IMMEDIATE ///< 立即应用到控件外观
	};

	virtual ~ControlState() = default;

	virtual void
	set_visual_state(VisualState state,
			 ApplyPolicy policy = ApplyPolicy::DEFERRED) = 0;
	virtual VisualState get_visual_state() const = 0;
	virtual void set_pending(bool pending)
	{
		(void)pending;
	}
	virtual bool is_pending() const
	{
		return false;
	}
	virtual void set_pending_timeout(uint32_t timeout_ms)
	{
		(void)timeout_ms;
	}

	void set_active(bool active, ApplyPolicy policy = ApplyPolicy::DEFERRED)
	{
		set_visual_state(active ? VisualState::ACTIVE :
					  VisualState::DEFAULT,
				 policy);
	}

	bool is_active() const
	{
		return get_visual_state() == VisualState::ACTIVE;
	}

	void toggle_active(ApplyPolicy policy = ApplyPolicy::DEFERRED)
	{
		set_active(!is_active(), policy);
	}
};

} // namespace ui
