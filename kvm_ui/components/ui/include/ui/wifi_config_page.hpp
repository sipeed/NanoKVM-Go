#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <string>
#include <string_view>
#include <vector>

#include "ui/subpage_base.hpp"

namespace ui
{

class WifiConfigPage : public SubPageBase {
    public:
	enum class InputMode : uint8_t {
		Ssid,
		Password,
	};

	WifiConfigPage();
	~WifiConfigPage() override = default;

	void destroy() override;
	void configure(InputMode mode, std::string initial_text = {});
	void set_input_text(std::string text);
	const std::string &input_text() const
	{
		return _input_text;
	}

	std::function<void(const std::string &)> on_ok_click;

    private:
	enum class KeyAction : uint8_t {
		CycleInsert,
		ToggleCase,
		SwitchSymbols,
		SwitchSymbolsSecondary,
		SwitchLetters,
		SwitchNumbers,
		TogglePasswordVisibility,
		Space,
		Backspace,
		Ok,
	};

	enum class KeyboardPage : uint8_t {
		Letters,
		SymbolsPrimary,
		SymbolsSecondary,
		Numbers,
	};

	struct KeyButton {
		WifiConfigPage *page = nullptr;
		lv_obj_t *obj = nullptr;
		lv_obj_t *label = nullptr;
		KeyAction action = KeyAction::CycleInsert;
		std::size_t index = 0;
		std::string label_text;
		std::string candidates;
	};

	static constexpr std::size_t KEY_ROWS = 3;
	static constexpr std::size_t KEY_COLUMNS = 5;
	static constexpr std::size_t KEY_COUNT = KEY_ROWS * KEY_COLUMNS;
	static constexpr std::size_t MAX_SSID_INPUT_LENGTH = 32;
	static constexpr std::size_t MAX_PASSWORD_INPUT_LENGTH = 63;

	lv_obj_t *_input_label = nullptr;
	lv_obj_t *_keyboard = nullptr;
	InputMode _input_mode = InputMode::Password;
	std::string _placeholder = "Enter password";
	std::string _input_text;
	KeyboardPage _keyboard_page = KeyboardPage::Letters;
	bool _uppercase_lock = false;
	bool _password_visible = false;
	bool _has_pending_candidate = false;
	std::size_t _pending_key_index = KEY_COUNT;
	std::size_t _pending_candidate_index = 0;
	lv_timer_t *_candidate_commit_timer = nullptr;
	std::vector<KeyButton> _keys;

	void build_content() override;
	void on_subpage_destroy() override;
	void reset_lvgl_pointers();
	void create_input_label();
	void create_keyboard();
	void update_keyboard();
	void update_input_label();
	std::size_t max_input_length() const;
	void handle_key(KeyButton &key);
	KeyAction action_for_label(std::string_view label) const;
	void commit_pending_candidate();
	void reset_pending_candidate();
	void start_candidate_commit_timer();
	void stop_candidate_commit_timer();
	void cycle_insert(KeyButton &key, std::size_t key_index);

	static void key_click_event_cb(lv_event_t *event);
	static void candidate_commit_timer_cb(lv_timer_t *timer);
};

} // namespace ui
