#include <catch2/catch_test_macros.hpp>

#include <array>
#include <string_view>

#include "ui/wifi_page_action_policy.hpp"

namespace
{

struct PolicyCase {
	ui::WifiPageVisualState state;
	bool keyboard_config_mode;
	ui::WifiPrimaryAction action;
	std::string_view text;
	bool active;
};

} // namespace

TEST_CASE("Wi-Fi primary action policy maps stable keyboard states",
	  "[ui][wifi][policy]")
{
	constexpr std::array cases {
		PolicyCase { ui::WifiPageVisualState::Off, true,
			     ui::WifiPrimaryAction::RequestEnable, "OFF",
			     false },
		PolicyCase { ui::WifiPageVisualState::Error, true,
			     ui::WifiPrimaryAction::RequestEnable, "OFF",
			     false },
		PolicyCase { ui::WifiPageVisualState::Connecting, true,
			     ui::WifiPrimaryAction::Blocked, "...", false },
		PolicyCase { ui::WifiPageVisualState::ApConnect, true,
			     ui::WifiPrimaryAction::OpenSetup, "OFF", false },
		PolicyCase { ui::WifiPageVisualState::WebConfigure, true,
			     ui::WifiPrimaryAction::OpenSetup, "OFF", false },
		PolicyCase { ui::WifiPageVisualState::Connected, true,
			     ui::WifiPrimaryAction::RequestDisable, "ON",
			     true },
	};

	for (const PolicyCase &test_case : cases) {
		const ui::WifiPrimaryActionPolicy policy =
			ui::resolve_wifi_primary_action_policy(
				test_case.state, test_case.keyboard_config_mode,
				false);
		CHECK(policy.action == test_case.action);
		CHECK(policy.text == test_case.text);
		CHECK(policy.active == test_case.active);
	}
}

TEST_CASE("Wi-Fi primary action policy preserves non-keyboard compatibility",
	  "[ui][wifi][policy]")
{
	constexpr std::array cases {
		PolicyCase { ui::WifiPageVisualState::Off, false,
			     ui::WifiPrimaryAction::RequestEnable, "OFF",
			     false },
		PolicyCase { ui::WifiPageVisualState::Error, false,
			     ui::WifiPrimaryAction::RequestEnable, "OFF",
			     false },
		PolicyCase { ui::WifiPageVisualState::Connecting, false,
			     ui::WifiPrimaryAction::Blocked, "...", false },
		PolicyCase { ui::WifiPageVisualState::ApConnect, false,
			     ui::WifiPrimaryAction::RequestDisable, "ON",
			     false },
		PolicyCase { ui::WifiPageVisualState::WebConfigure, false,
			     ui::WifiPrimaryAction::RequestDisable, "ON",
			     false },
		PolicyCase { ui::WifiPageVisualState::Connected, false,
			     ui::WifiPrimaryAction::RequestDisable, "ON",
			     true },
	};

	for (const PolicyCase &test_case : cases) {
		const ui::WifiPrimaryActionPolicy policy =
			ui::resolve_wifi_primary_action_policy(
				test_case.state, test_case.keyboard_config_mode,
				false);
		CHECK(policy.action == test_case.action);
		CHECK(policy.text == test_case.text);
		CHECK(policy.active == test_case.active);
	}
}

TEST_CASE("Wi-Fi primary action policy blocks every state during interaction",
	  "[ui][wifi][policy]")
{
	constexpr std::array states {
		ui::WifiPageVisualState::Off,
		ui::WifiPageVisualState::Connected,
		ui::WifiPageVisualState::Connecting,
		ui::WifiPageVisualState::ApConnect,
		ui::WifiPageVisualState::WebConfigure,
		ui::WifiPageVisualState::Error,
	};

	for (const ui::WifiPageVisualState state : states) {
		for (const bool keyboard_config_mode : { false, true }) {
			const ui::WifiPrimaryActionPolicy policy =
				ui::resolve_wifi_primary_action_policy(
					state, keyboard_config_mode, true);
			CHECK(policy.action == ui::WifiPrimaryAction::Blocked);
			CHECK(policy.text == std::string_view { "..." });
			CHECK_FALSE(policy.active);
		}
	}
}
