#include "ui/tailscale_page.hpp"

#include <memory>
#include <string>

#include "ui/hold_confirm_behavior.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_36);
LV_IMAGE_DECLARE(settings_tailscale);
}

namespace ui
{
namespace
{

constexpr char kCodePlaceholder[] = "--";
constexpr char kStartText[] = "Start";
constexpr char kExitText[] = "Exit";
constexpr char kKeepText[] = "Keep";
constexpr char kIpKey[] = "IP:";

std::string format_pairing_code(const std::string &code)
{
	if (code.size() != 6) {
		return code;
	}
	return code.substr(0, 3) + " " + code.substr(3);
}

bool valid_obj(lv_obj_t *obj)
{
	return obj != nullptr && lv_obj_is_valid(obj);
}

}

TailscalePage::TailscalePage()
	: SubPageBase(PageType::SUBPAGE_TAILSCALE, "Tailscale")
{
}

TailscalePage::~TailscalePage() = default;

void TailscalePage::build_content()
{
	_stage = lv_obj_create(_scroll_content);
	lv_obj_set_size(_stage, STAGE_WIDTH, STAGE_HEIGHT);
	lv_obj_set_style_margin_top(_stage, STAGE_MARGIN_TOP, 0);
	lv_obj_set_style_bg_opa(_stage, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_stage, 0, 0);
	lv_obj_set_style_pad_all(_stage, 0, 0);
	lv_obj_set_flex_flow(_stage, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_flex_align(_stage, LV_FLEX_ALIGN_CENTER,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
	lv_obj_set_style_pad_row(_stage, 12, 0);
	lv_obj_clear_flag(_stage, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_stage, LV_OBJ_FLAG_CLICKABLE);

	_code_label = lv_label_create(_stage);
	lv_obj_set_width(_code_label, STAGE_WIDTH);
	lv_label_set_long_mode(_code_label, LV_LABEL_LONG_WRAP);
	lv_obj_set_style_text_align(_code_label, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_set_style_text_color(_code_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_code_label, &MontserratBold_36, 0);
	lv_label_set_text(_code_label, kCodePlaceholder);

	auto ip_row = std::make_unique<SConA>(_stage, "",
					      SConA::TextAlign::LEFT);
	_ip_row = ip_row.get();
	ip_row->create();
	_components.push_back(std::move(ip_row));

	lv_obj_add_flag(_ip_row->get_obj(), LV_OBJ_FLAG_HIDDEN);

	auto action = std::make_unique<BConA>(_scroll_content, kStartText,
					      ACCENT_COLOR);
	_action_btn = action.get();
	action->create();
	lv_obj_set_style_margin_top(action->get_obj(), ACTION_MARGIN_TOP, 0);
	action->set_left_image(&settings_tailscale);
	_components.push_back(std::move(action));

	render();
}

void TailscalePage::on_enter()
{
	reset_transient_state();
	if (on_session_visibility) {
		on_session_visibility(true);
	}
	render();
}

void TailscalePage::on_pause()
{
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
	}
	set_back_navigation_available(true);
}

void TailscalePage::on_exit()
{
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
	}
	set_back_navigation_available(true);
	if (on_session_visibility) {
		on_session_visibility(false);
	}
}

void TailscalePage::destroy()
{
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
	}
	_hold_behavior.reset();
	on_session_visibility = nullptr;
	on_start_request = nullptr;
	on_exit_request = nullptr;
	SubPageBase::destroy();
}

void TailscalePage::apply_snapshot(const TailscaleSnapshotView &snapshot,
				   uint64_t gate_seq)
{
	if (!is_created() ||
	    lv_obj_has_flag(get_container(), LV_OBJ_FLAG_HIDDEN)) {
		return;
	}
	if (gate_seq <= _gate_seq) {
		return;
	}
	_gate_seq = gate_seq;
	_snapshot = snapshot;

	if (_init_pending && snapshot.has_snapshot && snapshot.started) {
		_init_pending = false;
	}
	if (_exit_pending && !tailscale_gate_ready(snapshot)) {
		_exit_pending = false;
	}
	refresh_message_lifetime();

	render();
}

void TailscalePage::apply_code(const std::string &code, uint64_t seq)
{
	if (!is_created() ||
	    lv_obj_has_flag(get_container(), LV_OBJ_FLAG_HIDDEN)) {
		return;
	}
	if (!tailscale_gate_client_init(_snapshot)) {
		return;
	}
	if (seq <= _code_seq) {
		return;
	}
	_code_seq = seq;
	_code = code;
	render();
}

void TailscalePage::apply_action_result(TailscaleActionKind kind, bool ok,
					const std::string &message,
					uint64_t seq)
{
	if (!is_created() ||
	    lv_obj_has_flag(get_container(), LV_OBJ_FLAG_HIDDEN)) {
		return;
	}
	if (seq <= _action_seq) {
		return;
	}
	_action_seq = seq;

	if (kind == TailscaleActionKind::Start) {
		_init_pending = false;
		if (ok) {
			clear_message();
		} else {
			_message = message;
			_has_message = !_message.empty();
			_message_from_exit = false;
		}
	} else if (kind == TailscaleActionKind::Exit) {
		_exit_pending = false;
		if (ok) {
			clear_message();
		} else {
			_message = message;
			_has_message = !_message.empty();
			_message_from_exit = true;
		}
	}

	render();
}

void TailscalePage::render()
{
	if (!is_created()) {
		return;
	}

	TailscaleGateInput input;
	input.snapshot = _snapshot;
	input.init_pending = _init_pending;
	input.exit_pending = _exit_pending;
	input.has_message = _has_message;
	const TailscaleGateView view = tailscale_gate_view(input);

	apply_button_role(view.button_role);

	if (_action_btn != nullptr) {
		_action_btn->set_text(tailscale_gate_button_text(view));
		lv_obj_t *button_obj = _action_btn->get_obj();
		if (valid_obj(button_obj)) {
			if (view.button_visible) {
				lv_obj_clear_flag(button_obj,
						  LV_OBJ_FLAG_HIDDEN);
			} else {
				lv_obj_add_flag(button_obj, LV_OBJ_FLAG_HIDDEN);
			}
			if (view.button_interactive) {
				lv_obj_add_flag(button_obj, LV_OBJ_FLAG_CLICKABLE);
			} else {
				lv_obj_clear_flag(button_obj,
						  LV_OBJ_FLAG_CLICKABLE);
			}
		}
		const bool pending = _init_pending || _exit_pending;
		if (pending && _hold_behavior != nullptr) {
			_hold_behavior->cancel();
		}
		if (pending && view.button_visible) {
			_action_btn->set_pending_timeout_enabled(false);
			_action_btn->set_pending(true);
		} else {
			_action_btn->set_pending(false);
			_action_btn->set_pending_timeout_enabled(true);
		}
	}

	if (valid_obj(_code_label)) {
		std::string text;
		if (view.code_source == TailscaleCodeSource::Message) {
			text = _message;
		} else if (view.code_source ==
			   TailscaleCodeSource::SessionCode) {
			text = format_pairing_code(_code);
		}
		if (text.empty()) {
			text = kCodePlaceholder;
		}
		lv_label_set_text(_code_label, text.c_str());
		if (view.code_visible) {
			lv_obj_clear_flag(_code_label, LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_code_label, LV_OBJ_FLAG_HIDDEN);
		}
	}

	if (_ip_row != nullptr && valid_obj(_ip_row->get_obj())) {
		if (view.ip_visible) {
			_ip_row->set_dual_text(kIpKey, _snapshot.ip.c_str());
			lv_obj_clear_flag(_ip_row->get_obj(),
					  LV_OBJ_FLAG_HIDDEN);
		} else {
			lv_obj_add_flag(_ip_row->get_obj(), LV_OBJ_FLAG_HIDDEN);
		}
	}

	set_back_navigation_available(!(_init_pending || _exit_pending));
}

void TailscalePage::apply_button_role(TailscaleButtonRole role)
{
	if (role == _button_role) {
		return;
	}

	// Tear down the outgoing gesture path before installing the new one:
	// a gesture already in progress must never confirm the new role.
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
		_hold_behavior.reset();
	}
	_button_role = role;

	if (_action_btn == nullptr) {
		return;
	}

	if (role == TailscaleButtonRole::Start) {
		_hold_behavior = std::make_unique<HoldConfirmBehavior>(
			*_action_btn, HoldConfirmConfig {
					      .duration_ms =
						      START_HOLD_DURATION_MS,
					      .fill_color = ACCENT_COLOR,
					      .idle_text = kStartText,
					      .holding_text = kKeepText,
				      });
		_hold_behavior->on_confirm([this]() {
			if (_init_pending || !on_start_request) {
				return;
			}
			if (!on_start_request()) {
				return;
			}
			clear_message();
			_init_pending = true;
			render();
		});
		return;
	}

	if (role == TailscaleButtonRole::Exit) {
		_hold_behavior = std::make_unique<HoldConfirmBehavior>(
			*_action_btn, HoldConfirmConfig {
					      .duration_ms =
						      START_HOLD_DURATION_MS,
					      .fill_color = ACCENT_COLOR,
					      .idle_text = kExitText,
					      .holding_text = kKeepText,
				      });
		_hold_behavior->on_confirm([this]() {
			if (_exit_pending || !on_exit_request) {
				return;
			}
			if (!on_exit_request()) {
				return;
			}
			clear_message();
			_exit_pending = true;
			render();
		});
		return;
	}
}

void TailscalePage::clear_message()
{
	_message.clear();
	_has_message = false;
	_message_from_exit = false;
}

void TailscalePage::refresh_message_lifetime()
{
	if (!_has_message) {
		return;
	}
	const TailscaleGateStage stage = tailscale_gate_stage(_snapshot);
	if (_message_from_exit) {
		if (stage != TailscaleGateStage::Ready) {
			clear_message();
		}
		return;
	}
	if (stage == TailscaleGateStage::ClientInit ||
	    stage == TailscaleGateStage::Ready) {
		clear_message();
	}
}

void TailscalePage::reset_transient_state()
{
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
	}
	_snapshot = TailscaleSnapshotView {};
	_gate_seq = 0;
	_code_seq = 0;
	_code.clear();
	clear_message();
	_init_pending = false;
	_exit_pending = false;
	set_back_navigation_available(true);
}

void TailscalePage::on_subpage_destroy()
{
	if (_hold_behavior != nullptr) {
		_hold_behavior->cancel();
	}
	_hold_behavior.reset();
	_stage = nullptr;
	_code_label = nullptr;
	_ip_row = nullptr;
	_action_btn = nullptr;
	_button_role = TailscaleButtonRole::None;
	_snapshot = TailscaleSnapshotView {};
	_gate_seq = 0;
	_code_seq = 0;
	_action_seq = 0;
	_code.clear();
	clear_message();
	_init_pending = false;
	_exit_pending = false;
	on_session_visibility = nullptr;
	on_start_request = nullptr;
	on_exit_request = nullptr;
}

}
