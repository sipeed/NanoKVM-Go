#include "ui/network_page.hpp"

#include "loading_arc.hpp"
#include "ui/bconb.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/ui_accent_palette.hpp"

extern "C" {
LV_FONT_DECLARE(MontserratBold_28);
LV_FONT_DECLARE(MontserratBold_36);
LV_IMAGE_DECLARE(settings_tailscale);
LV_IMAGE_DECLARE(network_terms_check);
LV_IMAGE_DECLARE(network_terms_cross);
}

namespace ui
{

namespace
{

constexpr char kExitText[] = "Exit";
constexpr char kKeepText[] = "Keep";
constexpr char kIpKey[] = "IP:";
// Fixed legal landing page for the local privacy-terms acknowledgement.
// Kept in sync with the published policy page; 39 ASCII bytes, which the
// fixed 125px canvas renders as QR version 3 (29 modules x integer scale 4).
constexpr char kTermsQrUrl[] = "https://wiki.sipeed.com/nkvm-legal.html";
constexpr char kPrivacyLeftText[] = "PRIVACY POLICY";
constexpr char kPrivacyRightText[] = "TERMS OF SERVICE";
constexpr char kDisabledHeadlineText[] = "Not available";
constexpr char kDisabledSublineText[] =
	"Remote Network is not enabled on this device.";
constexpr char kDisabledActionMessage[] = "DISABLED";
constexpr uint32_t kMutedTextColor = 0x8B949E;
constexpr int32_t TERMS_BODY_MARGIN_TOP = 12;
constexpr int32_t TERMS_BODY_HEIGHT = 152;
constexpr int32_t QR_CARD_LEFT = 0;
// On-device measured card geometry: the 140px side centers the fixed 125px
// QR canvas at x = 8 (LVGL pw/2 - w/2), and the canvas' internal margin
// (4px left / 5px right) leaves exactly 12px of white per side around the
// dark modules = 3.0 modules at the current scale 4, below the 4-module
// quiet-zone ideal. A 12px clearance separates the card bottom (canvas
// y=227) from the canvas bottom; radius 10 stays well inside the geometric
// corner-module limit floor(7.5*(2+sqrt(2))) = 25.
constexpr int32_t QR_CARD_SIZE = 140;
constexpr int32_t QR_CARD_RADIUS = 10;
// Fixed lv_qrcode canvas. The module count and integer scale are chosen by
// lv_qrcode_update from the payload length: qrcodegen picks the smallest
// version that fits (ECC MEDIUM) and then grows it while `canvas / modules`
// stays constant. With the fixed legal URL that is version 3 (29 modules) at
// scale 4 = 116 rendered px with a 4px internal margin (4 left / 5 right);
// a shorter payload would still land on version 2 (25 modules x scale 5 =
// 125 px, no margin).
constexpr int32_t QR_CODE_SIZE = 125;
constexpr int32_t TEXT_BLOCK_LEFT = 160;
constexpr int32_t TEXT_BLOCK_WIDTH = 107;
constexpr int32_t TEXT_BLOCK_PAD_ROW = 12;
constexpr int32_t TEXT_LINE_SPACE = 4;

void set_obj_visible(lv_obj_t *obj, bool visible)
{
	if (obj == nullptr || !lv_obj_is_valid(obj)) {
		return;
	}
	if (visible) {
		lv_obj_clear_flag(obj, LV_OBJ_FLAG_HIDDEN);
		return;
	}
	lv_obj_add_flag(obj, LV_OBJ_FLAG_HIDDEN);
}

lv_obj_t *build_privacy_label(lv_obj_t *parent, const char *text)
{
	lv_obj_t *label = lv_label_create(parent);
	lv_obj_set_width(label, TEXT_BLOCK_WIDTH);
	lv_label_set_long_mode(label, LV_LABEL_LONG_WRAP);
	lv_label_set_text(label, text);
	lv_obj_set_style_text_align(label, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_set_style_text_font(label, &lv_font_montserrat_16, 0);
	lv_obj_set_style_text_color(label, lv_color_hex(kMutedTextColor), 0);
	lv_obj_set_style_text_letter_space(label, 1, 0);
	lv_obj_set_style_text_line_space(label, TEXT_LINE_SPACE, 0);
	return label;
}

}

NetworkPage::NetworkPage()
	: SubPageBase(PageType::SUBPAGE_NETWORK, "Network", 0)
{
}

NetworkPage::~NetworkPage() = default;

void NetworkPage::build_content()
{
	constexpr int32_t title_top_margin = 12;
	constexpr int32_t brand_bottom_margin = 4;
	constexpr int32_t on_top = HEIGHT - title_top_margin - BConA::HEIGHT;
	constexpr int32_t on_top_margin = on_top - title_top_margin -
					  BConA::HEIGHT - 2 * SConA::HEIGHT -
					  brand_bottom_margin;
	constexpr int32_t on_padding_right = 17;

	_preinit_group = lv_obj_create(_scroll_content);
	lv_obj_set_width(_preinit_group, SConA::WIDTH);
	lv_obj_set_height(_preinit_group, LV_SIZE_CONTENT);
	lv_obj_set_style_bg_opa(_preinit_group, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_preinit_group, 0, 0);
	lv_obj_set_style_pad_all(_preinit_group, 0, 0);
	lv_obj_set_flex_flow(_preinit_group, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_style_pad_row(_preinit_group, 0, 0);
	lv_obj_clear_flag(_preinit_group, LV_OBJ_FLAG_SCROLLABLE);

	auto sipeed = std::make_unique<SConA>(_preinit_group, "Sipeed",
					      SConA::TextAlign::CENTER);
	_sipeed_row = sipeed.get();
	sipeed->create();
	_components.push_back(std::move(sipeed));

	auto remote = std::make_unique<SConA>(_preinit_group, "Remote Network",
					      SConA::TextAlign::CENTER);
	_remote_row = remote.get();
	remote->create();
	lv_obj_set_style_margin_bottom(remote->get_obj(), brand_bottom_margin,
				       0);
	_components.push_back(std::move(remote));

	auto on = std::make_unique<BConA>(_scroll_content, "ON",
					  BConA::COLOR_DEFAULT);
	_on_btn = on.get();
	on->create();
	lv_obj_set_style_margin_top(on->get_obj(), on_top_margin, 0);
	lv_obj_set_style_pad_right(on->get_obj(), on_padding_right, 0);
	on->set_left_image(&settings_tailscale);
	_components.push_back(std::move(on));

	_on_hold_behavior = std::make_unique<HoldConfirmBehavior>(
		*_on_btn, HoldConfirmConfig {
				  .duration_ms = ON_HOLD_DURATION_MS,
				  .fill_color = ACCENT_COLOR,
				  .idle_text = "ON",
				  .holding_text = "Keep",
			  });
	_on_hold_behavior->on_confirm([this]() { show_terms_stage(); });

	constexpr int32_t terms_row_offset =
		(WIDTH - SConA::WIDTH) / 2 - LEFT_MARGIN;

	_terms_row = lv_obj_create(_scroll_content);
	lv_obj_set_size(_terms_row, SConA::WIDTH, BConB::HEIGHT);
	lv_obj_set_style_bg_opa(_terms_row, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_terms_row, 0, 0);
	lv_obj_set_style_pad_all(_terms_row, 0, 0);
	lv_obj_set_style_margin_left(_terms_row, terms_row_offset, 0);
	lv_obj_set_scrollbar_mode(_terms_row, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_terms_row, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_terms_row, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_set_flex_flow(_terms_row, LV_FLEX_FLOW_ROW);
	lv_obj_set_flex_align(_terms_row, LV_FLEX_ALIGN_SPACE_BETWEEN,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

	auto agree = std::make_unique<BConB>(_terms_row,
					     accent::ACCENT_GREEN_800);
	_agree_btn = agree.get();
	agree->create();
	agree->set_active(true, ControlState::ApplyPolicy::IMMEDIATE);
	agree->set_image(&network_terms_check);
	agree->on_click([this](BConB *) {
		if (_init_pending || on_start_request == nullptr ||
		    !on_start_request()) {
			return;
		}
		arm_start_pending();
		show_code_stage();
	});
	_components.push_back(std::move(agree));

	auto decline = std::make_unique<BConB>(_terms_row,
					       accent::ACCENT_ORANGE_700);
	_decline_btn = decline.get();
	decline->create();
	decline->set_active(true, ControlState::ApplyPolicy::IMMEDIATE);
	decline->set_image(&network_terms_cross);
	decline->on_click([this](BConB *) { show_preinit_stage(); });
	_components.push_back(std::move(decline));

	_terms_body = lv_obj_create(_scroll_content);
	lv_obj_set_size(_terms_body, WIDTH, TERMS_BODY_HEIGHT);
	lv_obj_set_style_bg_opa(_terms_body, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_terms_body, 0, 0);
	lv_obj_set_style_pad_all(_terms_body, 0, 0);
	lv_obj_set_style_margin_left(_terms_body, -LEFT_MARGIN, 0);
	lv_obj_set_style_margin_top(_terms_body, TERMS_BODY_MARGIN_TOP, 0);
	lv_obj_set_scrollbar_mode(_terms_body, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_terms_body, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_terms_body, LV_OBJ_FLAG_CLICKABLE);

	_qr_card = lv_obj_create(_terms_body);
	lv_obj_set_size(_qr_card, QR_CARD_SIZE, QR_CARD_SIZE);
	lv_obj_set_pos(_qr_card, QR_CARD_LEFT, 0);
	lv_obj_set_style_bg_color(_qr_card, lv_color_white(), 0);
	lv_obj_set_style_bg_opa(_qr_card, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_qr_card, QR_CARD_RADIUS, 0);
	lv_obj_set_style_border_width(_qr_card, 0, 0);
	lv_obj_set_style_pad_all(_qr_card, 0, 0);
	lv_obj_set_style_margin_all(_qr_card, 0, 0);
	lv_obj_set_style_shadow_width(_qr_card, 0, 0);
	lv_obj_set_style_outline_width(_qr_card, 0, 0);
	lv_obj_set_scrollbar_mode(_qr_card, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_qr_card, LV_OBJ_FLAG_SCROLLABLE);

	_qr_code = lv_qrcode_create(_qr_card);
	lv_qrcode_set_size(_qr_code, QR_CODE_SIZE);
	lv_qrcode_set_dark_color(_qr_code, lv_color_black());
	lv_qrcode_set_light_color(_qr_code, lv_color_white());
	lv_obj_center(_qr_code);

	_privacy_block = lv_obj_create(_terms_body);
	lv_obj_set_size(_privacy_block, TEXT_BLOCK_WIDTH, TERMS_BODY_HEIGHT);
	lv_obj_set_pos(_privacy_block, TEXT_BLOCK_LEFT, 0);
	lv_obj_set_style_bg_opa(_privacy_block, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_privacy_block, 0, 0);
	lv_obj_set_style_pad_all(_privacy_block, 0, 0);
	lv_obj_set_style_pad_row(_privacy_block, TEXT_BLOCK_PAD_ROW, 0);
	lv_obj_set_scrollbar_mode(_privacy_block, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_privacy_block, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_privacy_block, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_set_flex_flow(_privacy_block, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_flex_align(_privacy_block, LV_FLEX_ALIGN_CENTER,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

	_privacy_left = build_privacy_label(_privacy_block, kPrivacyLeftText);
	_privacy_right = build_privacy_label(_privacy_block, kPrivacyRightText);

	_code_stage = lv_obj_create(_scroll_content);
	lv_obj_set_size(_code_stage, CODE_STAGE_WIDTH, CODE_STAGE_HEIGHT);
	lv_obj_set_style_margin_top(_code_stage, CODE_STAGE_MARGIN_TOP, 0);
	lv_obj_set_style_bg_opa(_code_stage, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_code_stage, 0, 0);
	lv_obj_set_style_pad_all(_code_stage, 0, 0);
	lv_obj_set_flex_flow(_code_stage, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_flex_align(_code_stage, LV_FLEX_ALIGN_CENTER,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
	lv_obj_set_style_pad_row(_code_stage, 12, 0);
	lv_obj_clear_flag(_code_stage, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_code_stage, LV_OBJ_FLAG_CLICKABLE);

	_code_label = lv_label_create(_code_stage);
	lv_obj_set_width(_code_label, CODE_STAGE_WIDTH);
	lv_label_set_long_mode(_code_label, LV_LABEL_LONG_WRAP);
	lv_obj_set_style_text_align(_code_label, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_set_style_text_color(_code_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_code_label, &MontserratBold_36, 0);
	// LVGL seeds a fresh label with "Text"; start empty so the loading arc
	// is the only placeholder presentation.
	lv_label_set_text(_code_label, "");

	auto ip_row = std::make_unique<SConA>(_code_stage, "",
					      SConA::TextAlign::LEFT);
	_ip_row = ip_row.get();
	ip_row->create();
	lv_obj_add_flag(_ip_row->get_obj(), LV_OBJ_FLAG_HIDDEN);
	_components.push_back(std::move(ip_row));

	// Appended last so the code label and IP row keep their flex child
	// indices; flex skips the hidden arc until render_code() starts it.
	_loading_arc = std::make_unique<LoadingArc>(_code_stage);
	_loading_arc->create();

	auto exit = std::make_unique<BConA>(_scroll_content, kExitText,
					    ACCENT_COLOR);
	_exit_btn = exit.get();
	exit->create();
	lv_obj_set_style_margin_top(exit->get_obj(), EXIT_MARGIN_TOP, 0);
	exit->set_left_image(&settings_tailscale);
	lv_obj_add_flag(exit->get_obj(), LV_OBJ_FLAG_HIDDEN);
	_components.push_back(std::move(exit));

	_exit_hold_behavior = std::make_unique<HoldConfirmBehavior>(
		*_exit_btn, HoldConfirmConfig {
				    .duration_ms = ON_HOLD_DURATION_MS,
				    .fill_color = ACCENT_COLOR,
				    .idle_text = kExitText,
				    .holding_text = kKeepText,
			});
	_exit_hold_behavior->on_confirm([this]() {
		if (_exit_pending || on_exit_request == nullptr ||
		    !on_exit_request()) {
			return;
		}
		arm_exit_pending();
		if (_stage == Stage::Code) {
			render_code();
			return;
		}
		render_paired();
	});

	_disabled_stage = lv_obj_create(_scroll_content);
	lv_obj_set_size(_disabled_stage, CODE_STAGE_WIDTH, CODE_STAGE_HEIGHT);
	lv_obj_set_style_margin_top(_disabled_stage, CODE_STAGE_MARGIN_TOP, 0);
	lv_obj_set_style_bg_opa(_disabled_stage, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_disabled_stage, 0, 0);
	lv_obj_set_style_pad_all(_disabled_stage, 0, 0);
	lv_obj_set_flex_flow(_disabled_stage, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_flex_align(_disabled_stage, LV_FLEX_ALIGN_CENTER,
			      LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
	lv_obj_set_style_pad_row(_disabled_stage, 12, 0);
	lv_obj_clear_flag(_disabled_stage, LV_OBJ_FLAG_SCROLLABLE);
	lv_obj_clear_flag(_disabled_stage, LV_OBJ_FLAG_CLICKABLE);

	_disabled_headline = lv_label_create(_disabled_stage);
	lv_obj_set_width(_disabled_headline, CODE_STAGE_WIDTH);
	lv_label_set_long_mode(_disabled_headline, LV_LABEL_LONG_WRAP);
	lv_obj_set_style_text_align(_disabled_headline, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_set_style_text_color(_disabled_headline, lv_color_white(), 0);
	lv_obj_set_style_text_font(_disabled_headline, &MontserratBold_28, 0);
	lv_label_set_text(_disabled_headline, kDisabledHeadlineText);

	_disabled_subline = lv_label_create(_disabled_stage);
	lv_obj_set_width(_disabled_subline, CODE_STAGE_WIDTH);
	lv_label_set_long_mode(_disabled_subline, LV_LABEL_LONG_WRAP);
	lv_obj_set_style_text_align(_disabled_subline, LV_TEXT_ALIGN_CENTER, 0);
	lv_obj_set_style_text_color(_disabled_subline,
				    lv_color_hex(kMutedTextColor), 0);
	lv_obj_set_style_text_font(_disabled_subline, &lv_font_montserrat_16, 0);
	lv_obj_set_style_text_line_space(_disabled_subline, TEXT_LINE_SPACE, 0);
	lv_label_set_text(_disabled_subline, kDisabledSublineText);

	set_terms_stage_visible(false);
	set_code_stage_visible(false);
	set_disabled_stage_visible(false);
	_qr_payload.clear();
	refresh_terms_qr();
}

void NetworkPage::on_enter()
{
	cancel_hold_behaviors();
	clear_pendings();
	_gate_seq = 0;
	_code_seq = 0;
	_code.clear();
	_snapshot = TailscaleSnapshotView {};
	show_preinit_stage();
	if (on_session_visibility) {
		on_session_visibility(true);
	}
}

void NetworkPage::destroy()
{
	cancel_hold_behaviors();
	_on_hold_behavior.reset();
	_exit_hold_behavior.reset();
	delete_pending_timer();
	_loading_arc.reset();
	on_start_request = nullptr;
	on_exit_request = nullptr;
	on_session_visibility = nullptr;
	SubPageBase::destroy();
}

void NetworkPage::on_pause()
{
	cancel_hold_behaviors();
	clear_pendings();
	if (_loading_arc != nullptr) {
		_loading_arc->set_running(false);
	}
}

void NetworkPage::on_resume()
{
	if (_stage == Stage::Code) {
		render_code();
		return;
	}
	if (_loading_arc != nullptr) {
		_loading_arc->set_running(false);
	}
}

void NetworkPage::on_exit()
{
	cancel_hold_behaviors();
	clear_pendings();
	if (_loading_arc != nullptr) {
		_loading_arc->set_running(false);
	}
	if (on_session_visibility) {
		on_session_visibility(false);
	}
}

void NetworkPage::apply_snapshot(const TailscaleSnapshotView &snapshot,
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
		clear_pendings();
	}
	if (_exit_pending) {
		// The exit pending is bounded by the stage it was armed in: the
		// paired stage waits for the session to leave Ready, the code
		// stage waits for it to leave ClientInit.
		const bool still_on_exit_stage =
			(_stage == Stage::Code) ?
				tailscale_gate_client_init(snapshot) :
				tailscale_gate_ready(snapshot);
		if (!still_on_exit_stage) {
			clear_pendings();
		}
	}
	if (_stage == Stage::Terms) {
		return;
	}
	if (_stage == Stage::Disabled) {
		return;
	}
	if (_message_hold) {
		if (_stage == Stage::Code) {
			const TailscaleGateStage stage =
				tailscale_gate_stage(_snapshot);
			if (stage != TailscaleGateStage::ClientInit &&
			    stage != TailscaleGateStage::Ready) {
				return;
			}
			clear_message_hold();
		} else if (_stage == Stage::Paired) {
			if (tailscale_gate_ready(_snapshot)) {
				render_paired();
				return;
			}
			clear_message_hold();
		} else {
			clear_message_hold();
		}
	}
	route_from_snapshot();
}

void NetworkPage::apply_code(const std::string &code, uint64_t seq)
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
	if (_stage != Stage::Code) {
		return;
	}
	render_code();
}

void NetworkPage::apply_action_result(TailscaleActionKind kind, bool ok,
				      const std::string &message, uint64_t seq)
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
		if (!_init_pending || ok) {
			return;
		}
		clear_pendings();
		if (message == kDisabledActionMessage) {
			show_disabled_stage();
			return;
		}
		set_message_hold(message);
		return;
	}
	if (kind == TailscaleActionKind::Exit) {
		if (!_exit_pending || ok) {
			return;
		}
		clear_pendings();
		set_message_hold(message);
	}
}

void NetworkPage::show_preinit_stage()
{
	if (!is_created()) {
		return;
	}
	cancel_hold_behaviors();
	set_terms_stage_visible(false);
	set_code_stage_visible(false);
	render_exit(false);
	set_disabled_stage_visible(false);
	set_title_visible(true);
	set_preinit_stage_visible(true);
	_stage = Stage::PreInit;
}

void NetworkPage::show_terms_stage()
{
	if (!is_created()) {
		return;
	}
	cancel_hold_behaviors();
	set_preinit_stage_visible(false);
	set_code_stage_visible(false);
	render_exit(false);
	set_disabled_stage_visible(false);
	set_title_visible(false);
	set_terms_stage_visible(true);
	_stage = Stage::Terms;
	refresh_terms_qr();
}

void NetworkPage::show_code_stage()
{
	if (!is_created()) {
		return;
	}
	// Mirror show_paired_stage(): client-init heartbeats re-enter this
	// stage through route_from_snapshot(), so an in-progress Exit hold
	// must survive; only the pre-init ON hold is cancelled.
	if (_on_hold_behavior != nullptr) {
		_on_hold_behavior->cancel();
	}
	set_preinit_stage_visible(false);
	set_terms_stage_visible(false);
	set_disabled_stage_visible(false);
	set_title_visible(true);
	set_code_stage_visible(true);
	set_obj_visible(_ip_row != nullptr ? _ip_row->get_obj() : nullptr,
			false);
	_stage = Stage::Code;
	render_code();
}

void NetworkPage::show_paired_stage()
{
	if (!is_created()) {
		return;
	}
	if (_on_hold_behavior != nullptr) {
		_on_hold_behavior->cancel();
	}
	set_preinit_stage_visible(false);
	set_terms_stage_visible(false);
	set_disabled_stage_visible(false);
	set_title_visible(true);
	set_code_stage_visible(true);
	render_exit(true);
	_stage = Stage::Paired;
	render_paired();
}

void NetworkPage::show_disabled_stage()
{
	if (!is_created()) {
		return;
	}
	cancel_hold_behaviors();
	set_preinit_stage_visible(false);
	set_terms_stage_visible(false);
	set_code_stage_visible(false);
	render_exit(false);
	set_title_visible(true);
	set_disabled_stage_visible(true);
	_stage = Stage::Disabled;
}

void NetworkPage::route_from_snapshot()
{
	if (_init_pending) {
		show_code_stage();
		return;
	}
	switch (tailscale_gate_stage(_snapshot)) {
	case TailscaleGateStage::ClientInit:
		show_code_stage();
		break;
	case TailscaleGateStage::Ready:
		show_paired_stage();
		break;
	case TailscaleGateStage::Unknown:
	case TailscaleGateStage::NotInitialized:
		show_preinit_stage();
		break;
	}
}

void NetworkPage::render_code()
{
	if (_code_label == nullptr || !lv_obj_is_valid(_code_label)) {
		return;
	}
	const bool show_message = _message_hold && !_action_message.empty();
	const bool show_code =
		!show_message && _stage == Stage::Code && !_code.empty();
	const bool show_spinner =
		!show_message && !show_code && _stage == Stage::Code;
	const bool label_visible = show_message || show_code;

	if (_loading_arc != nullptr) {
		_loading_arc->set_running(show_spinner);
	}
	if (label_visible) {
		const std::string text = show_message ?
						 _action_message :
						 format_pairing_code(_code);
		lv_label_set_text(_code_label, text.c_str());
		set_obj_visible(_code_label, true);
	} else {
		lv_label_set_text(_code_label, "");
		set_obj_visible(_code_label, false);
	}
	// The Exit button is bundled with the code label: it hides with the
	// label while the spinner runs. A pending exit stays visible so the
	// bundling rule can never swallow its pending arc. The enable gate is
	// temporary until the exit lane supports client-init teardown.
	render_exit((_code_stage_exit_enabled && label_visible) || _exit_pending);
}

void NetworkPage::render_paired()
{
	if (_loading_arc != nullptr) {
		_loading_arc->set_running(false);
	}
	const bool show_message = _message_hold && !_action_message.empty();
	if (show_message && _code_label != nullptr &&
	    lv_obj_is_valid(_code_label)) {
		lv_label_set_text(_code_label, _action_message.c_str());
		set_obj_visible(_code_label, true);
	} else {
		set_obj_visible(_code_label, false);
	}
	if (_ip_row != nullptr && lv_obj_is_valid(_ip_row->get_obj())) {
		if (show_message || _snapshot.ip.empty()) {
			lv_obj_add_flag(_ip_row->get_obj(), LV_OBJ_FLAG_HIDDEN);
		} else {
			_ip_row->set_dual_text(kIpKey, _snapshot.ip.c_str());
			lv_obj_clear_flag(_ip_row->get_obj(), LV_OBJ_FLAG_HIDDEN);
		}
	}
	render_exit(true);
}

void NetworkPage::set_message_hold(const std::string &message)
{
	_message_hold = !message.empty();
	_action_message = message;
	if (!_message_hold) {
		return;
	}
	if (_stage == Stage::Paired) {
		render_paired();
		return;
	}
	show_code_stage();
}

void NetworkPage::clear_message_hold()
{
	_message_hold = false;
	_action_message.clear();
}

void NetworkPage::arm_pending_timer()
{
	if (_pending_timer == nullptr) {
		_pending_timer =
			lv_timer_create(pending_timeout_cb, PENDING_TIMEOUT_MS, this);
		return;
	}
	lv_timer_reset(_pending_timer);
	lv_timer_resume(_pending_timer);
}

void NetworkPage::arm_start_pending()
{
	clear_message_hold();
	_init_pending = true;
	set_back_navigation_available(false);
	arm_pending_timer();
}

void NetworkPage::arm_exit_pending()
{
	clear_message_hold();
	_exit_pending = true;
	set_back_navigation_available(false);
	arm_pending_timer();
}

void NetworkPage::clear_pendings()
{
	if (_pending_timer != nullptr) {
		lv_timer_pause(_pending_timer);
	}
	_init_pending = false;
	_exit_pending = false;
	if (_exit_btn != nullptr && lv_obj_is_valid(_exit_btn->get_obj())) {
		_exit_btn->set_pending(false);
	}
	clear_message_hold();
	set_back_navigation_available(true);
}

void NetworkPage::delete_pending_timer()
{
	if (_pending_timer == nullptr) {
		return;
	}
	lv_timer_delete(_pending_timer);
	_pending_timer = nullptr;
}

void NetworkPage::on_pending_timeout()
{
	if (!_init_pending && !_exit_pending) {
		return;
	}
	_init_pending = false;
	_exit_pending = false;
	set_back_navigation_available(true);
	clear_message_hold();
	route_from_snapshot();
}

void NetworkPage::pending_timeout_cb(lv_timer_t *timer)
{
	lv_timer_pause(timer);
	auto *self = static_cast<NetworkPage *>(lv_timer_get_user_data(timer));
	if (self == nullptr) {
		return;
	}
	self->on_pending_timeout();
}

void NetworkPage::cancel_hold_behaviors()
{
	if (_on_hold_behavior != nullptr) {
		_on_hold_behavior->cancel();
	}
	if (_exit_hold_behavior != nullptr) {
		_exit_hold_behavior->cancel();
	}
}

void NetworkPage::set_title_visible(bool visible)
{
	set_obj_visible(title_button() != nullptr ? title_button()->get_obj() :
						    nullptr,
			visible);
}

void NetworkPage::set_preinit_stage_visible(bool visible)
{
	set_obj_visible(_preinit_group, visible);
	set_obj_visible(_on_btn != nullptr ? _on_btn->get_obj() : nullptr,
			visible);
}

void NetworkPage::set_terms_stage_visible(bool visible)
{
	set_obj_visible(_terms_row, visible);
	set_obj_visible(_terms_body, visible);
}

void NetworkPage::set_code_stage_visible(bool visible)
{
	set_obj_visible(_code_stage, visible);
	if (!visible) {
		set_obj_visible(_ip_row != nullptr ? _ip_row->get_obj() : nullptr,
				false);
		if (_loading_arc != nullptr) {
			_loading_arc->set_running(false);
		}
	}
}

void NetworkPage::render_exit(bool visible)
{
	if (_exit_btn == nullptr || !lv_obj_is_valid(_exit_btn->get_obj())) {
		return;
	}
	if (_exit_pending) {
		// The BConA-internal indicator timeout must never start; the
		// page watchdog owns the pending budget.
		_exit_btn->set_pending_timeout_enabled(false);
		_exit_btn->set_pending(true);
	} else {
		_exit_btn->set_pending(false);
		_exit_btn->set_pending_timeout_enabled(true);
	}
	set_obj_visible(_exit_btn->get_obj(), visible);
}

void NetworkPage::set_disabled_stage_visible(bool visible)
{
	set_obj_visible(_disabled_stage, visible);
}

std::string NetworkPage::format_pairing_code(const std::string &code)
{
	if (code.size() != 6) {
		return code;
	}
	return code.substr(0, 3) + " " + code.substr(3);
}

void NetworkPage::refresh_terms_qr()
{
	if (_qr_code == nullptr || !lv_obj_is_valid(_qr_code)) {
		return;
	}
	// The payload is a build-time constant; `_qr_payload` records only what the
	// current widget already encodes, so it is set after a successful encode.
	// The create-time invalidation therefore forces exactly one encode into a
	// fresh canvas, and a failed encode stays retryable instead of saturating
	// the marker with an unencoded widget.
	if (_qr_payload == kTermsQrUrl) {
		return;
	}
	if (lv_qrcode_update(_qr_code, kTermsQrUrl,
			     static_cast<uint32_t>(sizeof(kTermsQrUrl) - 1)) ==
	    LV_RESULT_OK) {
		_qr_payload = kTermsQrUrl;
	}
}

void NetworkPage::on_subpage_destroy()
{
	cancel_hold_behaviors();
	_on_hold_behavior.reset();
	_exit_hold_behavior.reset();
	delete_pending_timer();
	_loading_arc.reset();
	on_start_request = nullptr;
	on_exit_request = nullptr;
	on_session_visibility = nullptr;
	_code_stage = nullptr;
	_code_label = nullptr;
	_ip_row = nullptr;
	_exit_btn = nullptr;
	_snapshot = TailscaleSnapshotView {};
	_gate_seq = 0;
	_code_seq = 0;
	_action_seq = 0;
	_code.clear();
	clear_message_hold();
	_code_stage_exit_enabled = false;
	_init_pending = false;
	_exit_pending = false;
	_stage = Stage::PreInit;
	_preinit_group = nullptr;
	_sipeed_row = nullptr;
	_remote_row = nullptr;
	_on_btn = nullptr;
	_terms_row = nullptr;
	_agree_btn = nullptr;
	_decline_btn = nullptr;
	_terms_body = nullptr;
	_qr_card = nullptr;
	_qr_code = nullptr;
	_privacy_block = nullptr;
	_privacy_left = nullptr;
	_privacy_right = nullptr;
	_disabled_stage = nullptr;
	_disabled_headline = nullptr;
	_disabled_subline = nullptr;
}

}
