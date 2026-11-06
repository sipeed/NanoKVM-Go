#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <string>

#include "ui/scona.hpp"
#include "ui/subpage_base.hpp"
#include "ui/tailscale_gate_view.hpp"

namespace ui
{

class BConB;
class HoldConfirmBehavior;
class LoadingArc;

class NetworkPage : public SubPageBase {
    public:
	NetworkPage();
	~NetworkPage() override;

	void destroy() override;
	void on_enter() override;
	void on_pause() override;
	void on_resume() override;
	void on_exit() override;

	std::function<bool()> on_start_request;
	std::function<bool()> on_exit_request;
	std::function<void(bool)> on_session_visibility;

	enum class Stage : uint8_t { PreInit, Terms, Code, Paired, Disabled };

	// Temporary gate for the code-stage Exit button: the exit lane
	// (`?exit=true`) does not tear down ClientInit sessions yet, so the
	// button is hidden by default until the exit/disable lane supports
	// ClientInit. Tests enable it to keep covering the exit machinery.
	void set_code_stage_exit_enabled(bool enabled)
	{
		_code_stage_exit_enabled = enabled;
	}

	void apply_snapshot(const TailscaleSnapshotView &snapshot,
			    uint64_t gate_seq);
	void apply_code(const std::string &code, uint64_t seq);
	void apply_action_result(TailscaleActionKind kind, bool ok,
				 const std::string &message, uint64_t seq);

	Stage stage() const
	{
		return _stage;
	}

	uint64_t rendered_seq() const
	{
		return _gate_seq;
	}

	const std::string &qr_payload() const
	{
		return _qr_payload;
	}

	static constexpr uint32_t ACCENT_COLOR = 0x00838F;

    protected:
	void build_content() override;
	void on_subpage_destroy() override;

    private:
	static constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
	static constexpr uint16_t CODE_STAGE_WIDTH = 230;
	static constexpr uint16_t CODE_STAGE_HEIGHT = 152;
	// The stage straddles the flanking BConA rows symmetrically (-32 on
	// both margins), so its flex center lands exactly on the midpoint of
	// the open band between the title row (bottom edge 76) and the Exit
	// row (top edge 164): any centered child — code label, loading arc,
	// or paired IP row — lands at pixel midpoint 119.
	static constexpr int16_t CODE_STAGE_MARGIN_TOP = -32;
	static constexpr int16_t EXIT_MARGIN_TOP =
		(HEIGHT - 12 - BConA::HEIGHT) -
		(12 + BConA::HEIGHT + CODE_STAGE_MARGIN_TOP + CODE_STAGE_HEIGHT);
	static constexpr uint32_t PENDING_TIMEOUT_MS = 15000;

	void show_preinit_stage();
	void show_terms_stage();
	void show_code_stage();
	void show_paired_stage();
	void show_disabled_stage();
	void route_from_snapshot();
	void render_code();
	void render_paired();
	void set_message_hold(const std::string &message);
	void clear_message_hold();
	void arm_pending_timer();
	void arm_start_pending();
	void arm_exit_pending();
	void clear_pendings();
	void delete_pending_timer();
	void on_pending_timeout();
	static void pending_timeout_cb(lv_timer_t *timer);
	void cancel_hold_behaviors();
	void set_title_visible(bool visible);
	void set_preinit_stage_visible(bool visible);
	void set_terms_stage_visible(bool visible);
	void set_code_stage_visible(bool visible);
	void render_exit(bool visible);
	void set_disabled_stage_visible(bool visible);
	void refresh_terms_qr();
	static std::string format_pairing_code(const std::string &code);

	lv_obj_t *_preinit_group = nullptr;
	SConA *_sipeed_row = nullptr;
	SConA *_remote_row = nullptr;
	BConA *_on_btn = nullptr;
	std::unique_ptr<HoldConfirmBehavior> _on_hold_behavior;

	lv_obj_t *_terms_row = nullptr;
	BConB *_agree_btn = nullptr;
	BConB *_decline_btn = nullptr;
	lv_obj_t *_terms_body = nullptr;
	lv_obj_t *_qr_card = nullptr;
	lv_obj_t *_qr_code = nullptr;
	lv_obj_t *_privacy_block = nullptr;
	lv_obj_t *_privacy_left = nullptr;
	lv_obj_t *_privacy_right = nullptr;
	std::string _qr_payload;

	lv_obj_t *_code_stage = nullptr;
	lv_obj_t *_code_label = nullptr;
	SConA *_ip_row = nullptr;
	std::unique_ptr<LoadingArc> _loading_arc;
	BConA *_exit_btn = nullptr;
	std::unique_ptr<HoldConfirmBehavior> _exit_hold_behavior;
	TailscaleSnapshotView _snapshot;
	uint64_t _gate_seq = 0;
	uint64_t _code_seq = 0;
	uint64_t _action_seq = 0;
	std::string _code;
	bool _message_hold = false;
	std::string _action_message;
	bool _code_stage_exit_enabled = false;
	bool _init_pending = false;
	bool _exit_pending = false;
	lv_timer_t *_pending_timer = nullptr;
	lv_obj_t *_disabled_stage = nullptr;
	lv_obj_t *_disabled_headline = nullptr;
	lv_obj_t *_disabled_subline = nullptr;
	Stage _stage = Stage::PreInit;
};

}
