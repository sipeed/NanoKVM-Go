#include <array>
#include <string_view>
#include <utility>

#include <lvgl.h>

#include "ui/wifi_config_page.hpp"

namespace ui
{

namespace
{

constexpr std::size_t KEY_LAYOUT_COUNT = 15;
constexpr uint16_t TITLE_INPUT_X = 52;
constexpr uint16_t TITLE_INPUT_WIDTH = BConA::WIDTH - TITLE_INPUT_X;
constexpr uint16_t RETURN_PARENT_TOP_DISTANCE = 12;
constexpr uint16_t KEYBOARD_X = 1;
constexpr uint16_t KEYBOARD_WIDTH = 268;
constexpr uint16_t KEYBOARD_HEIGHT = 132;
constexpr uint16_t KEY_WIDTH = 48;
constexpr uint16_t KEY_HEIGHT = 40;
constexpr uint16_t KEY_GAP_X = 7;
constexpr uint16_t KEY_GAP_Y = 6;
constexpr uint32_t INPUT_BG_COLOR = 0x1C1917;
constexpr uint32_t KEY_BG_COLOR = 0x44403B;
constexpr uint32_t KEY_PRESSED_COLOR = 0x36332F;
constexpr uint32_t KEY_ACTIVE_COLOR = 0x2B7FFF;
constexpr uint32_t KEY_SECONDARY_SYMBOL_COLOR = 0xA16207;
constexpr uint32_t KEY_SECONDARY_SYMBOL_PRESSED_COLOR = 0x854D0E;
constexpr uint32_t KEY_OK_BG_COLOR = 0x00C950;
constexpr uint32_t KEY_OK_PRESSED_COLOR = 0x00A140;
constexpr uint32_t KEY_DEL_BG_COLOR = 0xFB2C36;
constexpr uint32_t KEY_DEL_PRESSED_COLOR = 0xC9232B;
constexpr uint8_t KEY_DISABLED_OPA = 96;
constexpr uint32_t INPUT_PLACEHOLDER_COLOR = 0xA8A29E;
constexpr uint32_t CANDIDATE_COMMIT_TIMEOUT_MS = 900;

struct KeyLayoutEntry {
	std::string_view label_text;
	std::string_view candidates;
	bool case_sensitive = false;
};

using KeyLayout = std::array<KeyLayoutEntry, KEY_LAYOUT_COUNT>;

constexpr KeyLayout kLetterLayout { {
	{ ".@_", ".@_", false },
	{ "abc", "abc", true },
	{ "def", "def", true },
	{ "Del", "", false },
	{ "OK", "", false },
	{ "ghi", "ghi", true },
	{ "jkl", "jkl", true },
	{ "mno", "mno", true },
	{ "Aa", "", false },
	{ "Sym", "", false },
	{ "pqrs", "pqrs", true },
	{ "tuv", "tuv", true },
	{ "wxyz", "wxyz", true },
	{ "Num", "", false },
	{ LV_SYMBOL_EYE_OPEN, "", false },
} };

constexpr KeyLayout kNumberLayout { {
	{ "1", "1", false },
	{ "2", "2", false },
	{ "3", "3", false },
	{ "Del", "", false },
	{ "OK", "", false },
	{ "4", "4", false },
	{ "5", "5", false },
	{ "6", "6", false },
	{ "0", "0", false },
	{ "Sym", "", false },
	{ "7", "7", false },
	{ "8", "8", false },
	{ "9", "9", false },
	{ "Num", "", false },
	{ LV_SYMBOL_EYE_OPEN, "", false },
} };

constexpr KeyLayout kSymbolLayout { {
	{ "! \"", "!\"", false },
	{ "# $", "#$", false },
	{ "% &", "%&", false },
	{ "Del", "", false },
	{ "OK", "", false },
	{ "' (", "'(", false },
	{ ") *", ")*", false },
	{ "+ ,", "+,", false },
	{ "Aa", "", false },
	{ "Sym", "", false },
	{ "- .", "-.", false },
	{ "/ :", "/:", false },
	{ "; <", ";<", false },
	{ "= >", "=>", false },
	{ LV_SYMBOL_EYE_OPEN, "", false },
} };

constexpr KeyLayout kSymbolSecondaryLayout { {
	{ "[", "[", false },
	{ "\\", "\\", false },
	{ "]", "]", false },
	{ "Del", "", false },
	{ "OK", "", false },
	{ "^", "^", false },
	{ "_ `", "_`", false },
	{ "{ |", "{|", false },
	{ "Aa", "", false },
	{ "ABC", "", false },
	{ "} ~", "}~", false },
	{ "Spec", "?@", false },
	{ "Spc", "", false },
	{ "Num", "", false },
	{ LV_SYMBOL_EYE_OPEN, "", false },
} };

constexpr bool is_action_label(std::string_view label)
{
	return label == "Del" || label == "OK" || label == "Aa" ||
	       label == "Sym" || label == "ABC" || label == "Num" ||
	       label == "Spc" || label == LV_SYMBOL_EYE_OPEN ||
	       label == LV_SYMBOL_EYE_CLOSE;
}

constexpr bool label_matches_candidates(std::string_view label,
					std::string_view candidates)
{
	if (is_action_label(label)) {
		return candidates.empty();
	}
	if (label == "Spec") {
		return candidates == "?@";
	}

	std::size_t candidate_index = 0;
	for (char ch : label) {
		if (ch == ' ') {
			continue;
		}
		if (candidate_index >= candidates.size() ||
		    candidates[candidate_index] != ch) {
			return false;
		}
		++candidate_index;
	}

	return candidate_index == candidates.size();
}

constexpr bool layout_labels_match_candidates(const KeyLayout &layout)
{
	for (const auto &entry : layout) {
		if (!label_matches_candidates(entry.label_text,
					      entry.candidates)) {
			return false;
		}
	}
	return true;
}

static_assert(layout_labels_match_candidates(kLetterLayout));
static_assert(layout_labels_match_candidates(kNumberLayout));
static_assert(layout_labels_match_candidates(kSymbolLayout));
static_assert(layout_labels_match_candidates(kSymbolSecondaryLayout));

char uppercase_ascii(char ch)
{
	if (ch >= 'a' && ch <= 'z') {
		return static_cast<char>(ch - ('a' - 'A'));
	}
	return ch;
}

std::string apply_case(std::string_view text, bool uppercase,
		       bool case_sensitive)
{
	std::string result(text);
	if (uppercase && case_sensitive) {
		for (auto &ch : result) {
			ch = uppercase_ascii(ch);
		}
	}
	return result;
}

uint32_t key_bg_color(bool active, bool ok, bool del)
{
	if (active) {
		return KEY_ACTIVE_COLOR;
	}
	if (ok) {
		return KEY_OK_BG_COLOR;
	}
	if (del) {
		return KEY_DEL_BG_COLOR;
	}
	return KEY_BG_COLOR;
}

uint32_t key_pressed_color(bool ok, bool del)
{
	if (ok) {
		return KEY_OK_PRESSED_COLOR;
	}
	if (del) {
		return KEY_DEL_PRESSED_COLOR;
	}
	return KEY_PRESSED_COLOR;
}

} // namespace

WifiConfigPage::WifiConfigPage()
	: SubPageBase(PageType::SUBPAGE_WIFI_CONFIG, "WiFi")
{
}

void WifiConfigPage::destroy()
{
	on_ok_click = nullptr;
	on_back_click = nullptr;
	stop_candidate_commit_timer();
	_components.clear();
	_scroll_content = nullptr;
	_back_btn = nullptr;
	Page::destroy();
	reset_lvgl_pointers();
}

void WifiConfigPage::configure(InputMode mode, std::string initial_text)
{
	stop_candidate_commit_timer();
	_input_mode = mode;
	_placeholder = mode == InputMode::Ssid ? "Enter SSID" :
						 "Enter password";
	_keyboard_page = KeyboardPage::Letters;
	_uppercase_lock = false;
	_password_visible = false;
	set_input_text(std::move(initial_text));
	update_keyboard();
}

void WifiConfigPage::set_input_text(std::string text)
{
	_input_text = std::move(text);
	if (_input_text.size() > max_input_length()) {
		_input_text.resize(max_input_length());
	}
	stop_candidate_commit_timer();
	reset_pending_candidate();
	update_input_label();
}

void WifiConfigPage::build_content()
{
	create_input_label();
	create_keyboard();
	update_keyboard();
	update_input_label();
}

void WifiConfigPage::on_subpage_destroy()
{
	stop_candidate_commit_timer();
	reset_lvgl_pointers();
}

void WifiConfigPage::reset_lvgl_pointers()
{
	stop_candidate_commit_timer();
	_input_label = nullptr;
	_keyboard = nullptr;
	_keys.clear();
	_keyboard_page = KeyboardPage::Letters;
	_uppercase_lock = false;
	_password_visible = false;
	reset_pending_candidate();
}

void WifiConfigPage::create_input_label()
{
	if (_back_btn == nullptr || _input_label != nullptr) {
		return;
	}

	_back_btn->set_text("");
	_input_label = lv_label_create(_back_btn->get_obj());
	lv_obj_add_flag(_input_label, LV_OBJ_FLAG_IGNORE_LAYOUT);
	lv_obj_set_width(_input_label, TITLE_INPUT_WIDTH);
	lv_obj_align(_input_label, LV_ALIGN_LEFT_MID, TITLE_INPUT_X, 0);
	lv_label_set_long_mode(_input_label, LV_LABEL_LONG_CLIP);
	lv_obj_set_style_text_color(_input_label, lv_color_white(), 0);
	lv_obj_set_style_text_font(_input_label, &lv_font_montserrat_16, 0);
	lv_obj_set_style_text_align(_input_label, LV_TEXT_ALIGN_LEFT, 0);
	lv_obj_clear_flag(_input_label, LV_OBJ_FLAG_CLICKABLE);
	lv_label_set_text(_input_label, _placeholder.c_str());
}

void WifiConfigPage::create_keyboard()
{
	if (_container == nullptr || _keyboard != nullptr) {
		return;
	}

	constexpr uint16_t KEYBOARD_Y =
		HEIGHT - KEYBOARD_HEIGHT - RETURN_PARENT_TOP_DISTANCE;

	_keyboard = lv_obj_create(_container);
	lv_obj_set_size(_keyboard, KEYBOARD_WIDTH, KEYBOARD_HEIGHT);
	lv_obj_set_pos(_keyboard, KEYBOARD_X, KEYBOARD_Y);
	lv_obj_set_style_bg_opa(_keyboard, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_keyboard, 0, 0);
	lv_obj_set_style_pad_all(_keyboard, 0, 0);
	lv_obj_clear_flag(_keyboard, LV_OBJ_FLAG_SCROLLABLE);

	_keys.reserve(KEY_COUNT);
	for (std::size_t row = 0; row < KEY_ROWS; ++row) {
		for (std::size_t col = 0; col < KEY_COLUMNS; ++col) {
			auto *button = lv_button_create(_keyboard);
			lv_obj_set_size(button, KEY_WIDTH, KEY_HEIGHT);
			lv_obj_set_pos(button,
				       static_cast<int32_t>(
					       col * (KEY_WIDTH + KEY_GAP_X)),
				       static_cast<int32_t>(
					       row * (KEY_HEIGHT + KEY_GAP_Y)));
			lv_obj_set_style_bg_color(
				button, lv_color_hex(KEY_BG_COLOR), 0);
			lv_obj_set_style_bg_color(
				button, lv_color_hex(KEY_PRESSED_COLOR),
				LV_STATE_PRESSED);
			lv_obj_set_style_bg_opa(button, LV_OPA_COVER, 0);
			lv_obj_set_style_radius(button, 10, 0);
			lv_obj_set_style_border_width(button, 0, 0);
			lv_obj_set_style_shadow_width(button, 0, 0);
			lv_obj_set_style_outline_width(button, 0, 0);
			lv_obj_set_style_pad_all(button, 0, 0);
			lv_obj_clear_flag(button, LV_OBJ_FLAG_SCROLLABLE);

			auto *label = lv_label_create(button);
			lv_obj_set_width(label, KEY_WIDTH - 4);
			lv_label_set_long_mode(label, LV_LABEL_LONG_CLIP);
			lv_obj_set_style_text_color(label, lv_color_white(), 0);
			lv_obj_set_style_text_font(label,
						   &lv_font_montserrat_16, 0);
			lv_obj_set_style_text_align(label, LV_TEXT_ALIGN_CENTER,
						    0);
			lv_obj_clear_flag(label, LV_OBJ_FLAG_CLICKABLE);
			lv_obj_center(label);

			_keys.push_back(KeyButton {
				.page = this,
				.obj = button,
				.label = label,
				.index = row * KEY_COLUMNS + col,
			});
			lv_obj_add_event_cb(button, key_click_event_cb,
					    LV_EVENT_CLICKED, &_keys.back());
		}
	}
}

void WifiConfigPage::update_keyboard()
{
	const KeyLayout *layout = &kLetterLayout;
	switch (_keyboard_page) {
	case KeyboardPage::Letters:
		layout = &kLetterLayout;
		break;
	case KeyboardPage::SymbolsPrimary:
		layout = &kSymbolLayout;
		break;
	case KeyboardPage::SymbolsSecondary:
		layout = &kSymbolSecondaryLayout;
		break;
	case KeyboardPage::Numbers:
		layout = &kNumberLayout;
		break;
	}

	for (std::size_t i = 0; i < _keys.size() && i < layout->size(); ++i) {
		auto &key = _keys[i];
		const auto &entry = (*layout)[i];
		key.label_text = apply_case(entry.label_text, _uppercase_lock,
					    entry.case_sensitive);
		key.candidates = apply_case(entry.candidates, _uppercase_lock,
					    entry.case_sensitive);
		key.action = action_for_label(entry.label_text);
		if (key.action == KeyAction::TogglePasswordVisibility) {
			key.label_text = _input_mode == InputMode::Password &&
							 _password_visible ?
						 LV_SYMBOL_EYE_CLOSE :
						 LV_SYMBOL_EYE_OPEN;
		}

		const bool secondary_symbol_active =
			key.action == KeyAction::SwitchSymbolsSecondary &&
			_keyboard_page == KeyboardPage::SymbolsPrimary;
		const bool active =
			(key.action == KeyAction::ToggleCase &&
			 _uppercase_lock) ||
			(key.action == KeyAction::SwitchLetters &&
			 _keyboard_page == KeyboardPage::SymbolsSecondary) ||
			(key.action == KeyAction::SwitchNumbers &&
			 _keyboard_page == KeyboardPage::Numbers) ||
			(key.action == KeyAction::TogglePasswordVisibility &&
			 _input_mode == InputMode::Password &&
			 _password_visible);
		const bool disabled =
			key.action == KeyAction::TogglePasswordVisibility &&
			_input_mode != InputMode::Password;

		if (key.label != nullptr) {
			lv_label_set_text(key.label, key.label_text.c_str());
			lv_obj_set_style_text_font(key.label,
						   &lv_font_montserrat_16, 0);
			lv_obj_set_style_text_opa(
				key.label,
				disabled ? KEY_DISABLED_OPA : LV_OPA_COVER, 0);
			lv_obj_center(key.label);
		}

		if (key.obj != nullptr) {
			const bool ok = key.action == KeyAction::Ok;
			const bool del = key.action == KeyAction::Backspace;
			const uint32_t bg_color =
				secondary_symbol_active ?
					KEY_SECONDARY_SYMBOL_COLOR :
					key_bg_color(active, ok, del);
			const uint32_t pressed_color =
				secondary_symbol_active ?
					KEY_SECONDARY_SYMBOL_PRESSED_COLOR :
					key_pressed_color(ok, del);
			lv_obj_set_style_bg_color(key.obj,
						  lv_color_hex(bg_color), 0);
			lv_obj_set_style_bg_color(key.obj,
						  lv_color_hex(pressed_color),
						  LV_STATE_PRESSED);
			lv_obj_set_style_opa(
				key.obj,
				disabled ? KEY_DISABLED_OPA : LV_OPA_COVER, 0);
			if (disabled) {
				lv_obj_clear_flag(key.obj,
						  LV_OBJ_FLAG_CLICKABLE);
				lv_obj_add_state(key.obj, LV_STATE_DISABLED);
			} else {
				lv_obj_add_flag(key.obj, LV_OBJ_FLAG_CLICKABLE);
				lv_obj_remove_state(key.obj, LV_STATE_DISABLED);
			}
			lv_obj_set_style_border_width(
				key.obj,
				active || secondary_symbol_active ? 2 : 0, 0);
			lv_obj_set_style_border_color(key.obj, lv_color_white(),
						      0);
		}
	}
}

void WifiConfigPage::update_input_label()
{
	if (_input_label != nullptr) {
		const bool is_placeholder = _input_text.empty();
		std::string password_display;
		const char *display = _placeholder.c_str();
		if (!is_placeholder) {
			if (_input_mode == InputMode::Password &&
			    !_password_visible) {
				password_display.assign(_input_text.size(),
							'*');
				if (_has_pending_candidate &&
				    !_input_text.empty()) {
					password_display.back() =
						_input_text.back();
				}
				display = password_display.c_str();
			} else {
				display = _input_text.c_str();
			}
		}
		lv_label_set_text(_input_label, display);
		lv_obj_set_style_text_color(
			_input_label,
			is_placeholder ? lv_color_hex(INPUT_PLACEHOLDER_COLOR) :
					 lv_color_white(),
			0);
		lv_obj_align(_input_label, LV_ALIGN_LEFT_MID, TITLE_INPUT_X, 0);
	}
}

std::size_t WifiConfigPage::max_input_length() const
{
	return _input_mode == InputMode::Ssid ? MAX_SSID_INPUT_LENGTH :
						MAX_PASSWORD_INPUT_LENGTH;
}

void WifiConfigPage::handle_key(KeyButton &key)
{
	switch (key.action) {
	case KeyAction::CycleInsert:
		cycle_insert(key, key.index);
		break;
	case KeyAction::ToggleCase:
		commit_pending_candidate();
		_uppercase_lock = !_uppercase_lock;
		update_keyboard();
		break;
	case KeyAction::SwitchSymbols:
		commit_pending_candidate();
		_keyboard_page = KeyboardPage::SymbolsPrimary;
		update_keyboard();
		break;
	case KeyAction::SwitchSymbolsSecondary:
		commit_pending_candidate();
		_keyboard_page = KeyboardPage::SymbolsSecondary;
		update_keyboard();
		break;
	case KeyAction::SwitchLetters:
		commit_pending_candidate();
		_keyboard_page = KeyboardPage::Letters;
		update_keyboard();
		break;
	case KeyAction::SwitchNumbers:
		commit_pending_candidate();
		_keyboard_page = _keyboard_page == KeyboardPage::Numbers ?
					 KeyboardPage::Letters :
					 KeyboardPage::Numbers;
		update_keyboard();
		break;
	case KeyAction::Space:
		commit_pending_candidate();
		if (_input_text.size() < max_input_length()) {
			_input_text += ' ';
			update_input_label();
		}
		break;
	case KeyAction::TogglePasswordVisibility:
		if (_input_mode != InputMode::Password) {
			break;
		}
		commit_pending_candidate();
		_password_visible = !_password_visible;
		update_keyboard();
		update_input_label();
		break;
	case KeyAction::Backspace:
		stop_candidate_commit_timer();
		if (!_input_text.empty()) {
			_input_text.pop_back();
		}
		reset_pending_candidate();
		update_input_label();
		break;
	case KeyAction::Ok:
		commit_pending_candidate();
		if (on_ok_click) {
			on_ok_click(_input_text);
		}
		break;
	}
}

WifiConfigPage::KeyAction
WifiConfigPage::action_for_label(std::string_view label) const
{
	if (label == "Del") {
		return KeyAction::Backspace;
	}
	if (label == "OK") {
		return KeyAction::Ok;
	}
	if (label == "Aa") {
		return KeyAction::ToggleCase;
	}
	if (label == "Sym") {
		return _keyboard_page == KeyboardPage::SymbolsPrimary ?
			       KeyAction::SwitchSymbolsSecondary :
			       KeyAction::SwitchSymbols;
	}
	if (label == LV_SYMBOL_EYE_OPEN || label == LV_SYMBOL_EYE_CLOSE) {
		return KeyAction::TogglePasswordVisibility;
	}
	if (label == "ABC") {
		return KeyAction::SwitchLetters;
	}
	if (label == "Num") {
		return KeyAction::SwitchNumbers;
	}
	if (label == "Spc") {
		return KeyAction::Space;
	}
	return KeyAction::CycleInsert;
}

void WifiConfigPage::commit_pending_candidate()
{
	const bool had_pending_candidate = _has_pending_candidate;
	stop_candidate_commit_timer();
	reset_pending_candidate();
	if (had_pending_candidate) {
		update_input_label();
	}
}

void WifiConfigPage::reset_pending_candidate()
{
	_has_pending_candidate = false;
	_pending_key_index = KEY_COUNT;
	_pending_candidate_index = 0;
}

void WifiConfigPage::start_candidate_commit_timer()
{
	stop_candidate_commit_timer();
	_candidate_commit_timer = lv_timer_create(
		candidate_commit_timer_cb, CANDIDATE_COMMIT_TIMEOUT_MS, this);
	lv_timer_set_repeat_count(_candidate_commit_timer, 1);
}

void WifiConfigPage::stop_candidate_commit_timer()
{
	if (_candidate_commit_timer != nullptr) {
		lv_timer_delete(_candidate_commit_timer);
		_candidate_commit_timer = nullptr;
	}
}

void WifiConfigPage::cycle_insert(KeyButton &key, std::size_t key_index)
{
	if (key.candidates.empty()) {
		commit_pending_candidate();
		return;
	}

	if (key.candidates.size() == 1U) {
		commit_pending_candidate();
		if (_input_text.size() < max_input_length()) {
			_input_text += key.candidates.front();
			update_input_label();
		}
		return;
	}

	if (_has_pending_candidate && _pending_key_index == key_index &&
	    !_input_text.empty()) {
		_pending_candidate_index =
			(_pending_candidate_index + 1U) % key.candidates.size();
		_input_text.back() = key.candidates[_pending_candidate_index];
		update_input_label();
		start_candidate_commit_timer();
		return;
	}

	commit_pending_candidate();
	if (_input_text.size() >= max_input_length()) {
		return;
	}

	_input_text += key.candidates.front();
	_has_pending_candidate = true;
	_pending_key_index = key_index;
	_pending_candidate_index = 0;
	update_input_label();
	start_candidate_commit_timer();
}

void WifiConfigPage::key_click_event_cb(lv_event_t *event)
{
	auto *key = static_cast<KeyButton *>(lv_event_get_user_data(event));
	if (key == nullptr || key->page == nullptr) {
		return;
	}
	key->page->handle_key(*key);
}

void WifiConfigPage::candidate_commit_timer_cb(lv_timer_t *timer)
{
	auto *page =
		static_cast<WifiConfigPage *>(lv_timer_get_user_data(timer));
	if (page == nullptr) {
		return;
	}

	page->_candidate_commit_timer = nullptr;
	page->reset_pending_candidate();
	page->update_input_label();
}

} // namespace ui
