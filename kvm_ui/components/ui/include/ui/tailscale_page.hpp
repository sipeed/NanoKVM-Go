#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <string>

#include "ui/bcona.hpp"
#include "ui/scona.hpp"
#include "ui/subpage_base.hpp"
#include "ui/tailscale_gate_view.hpp"

namespace ui
{

class HoldConfirmBehavior;

class TailscalePage : public SubPageBase {
    public:
	TailscalePage();
	~TailscalePage() override;

	void on_enter() override;
	void on_pause() override;
	void on_exit() override;
	void destroy() override;

	std::function<void(bool)> on_session_visibility;
	std::function<bool()> on_start_request;
	std::function<bool()> on_exit_request;

	void apply_snapshot(const TailscaleSnapshotView &snapshot,
			    uint64_t gate_seq);
	void apply_code(const std::string &code, uint64_t seq);
	void apply_action_result(TailscaleActionKind kind, bool ok,
				 const std::string &message, uint64_t seq);

	uint64_t rendered_seq() const
	{
		return _gate_seq;
	}

	static constexpr uint32_t START_HOLD_DURATION_MS = 2000;
	static constexpr uint32_t ACCENT_COLOR = 0x00838F;

    protected:
	void build_content() override;

    private:
	static constexpr uint16_t STAGE_WIDTH = 230;
	static constexpr uint16_t STAGE_HEIGHT = 152;
	static constexpr int16_t STAGE_MARGIN_TOP = -44;
	static constexpr int32_t TITLE_TOP_MARGIN = 12;
	static constexpr int32_t LAYOUT_ROW_GAP = 12;
	static constexpr int32_t ACTION_TOP =
		HEIGHT - TITLE_TOP_MARGIN - BConA::HEIGHT;
	static constexpr int32_t ACTION_MARGIN_TOP =
		ACTION_TOP - (TITLE_TOP_MARGIN + BConA::HEIGHT + LAYOUT_ROW_GAP +
			      STAGE_MARGIN_TOP + STAGE_HEIGHT + LAYOUT_ROW_GAP);

	void render();
	void apply_button_role(TailscaleButtonRole role);
	void reset_transient_state();
	void clear_message();
	void refresh_message_lifetime();
	void on_subpage_destroy() override;

	lv_obj_t *_stage = nullptr;
	lv_obj_t *_code_label = nullptr;
	SConA *_ip_row = nullptr;
	BConA *_action_btn = nullptr;
	std::unique_ptr<HoldConfirmBehavior> _hold_behavior;
	TailscaleButtonRole _button_role = TailscaleButtonRole::None;

	TailscaleSnapshotView _snapshot;
	uint64_t _gate_seq = 0;
	uint64_t _code_seq = 0;
	uint64_t _action_seq = 0;
	std::string _code;
	std::string _message;
	bool _has_message = false;
	bool _message_from_exit = false;
	bool _init_pending = false;
	bool _exit_pending = false;
};

}
