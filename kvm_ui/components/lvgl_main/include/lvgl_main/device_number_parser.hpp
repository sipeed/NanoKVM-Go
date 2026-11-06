#pragma once

#include <algorithm>
#include <cctype>
#include <optional>
#include <sstream>
#include <string>
#include <string_view>

namespace lvgl_main
{

// Trim leading and trailing ASCII whitespace from `value` in place.
inline std::string trim_device_number_text(std::string value)
{
	auto is_space = [](unsigned char ch) {
		return std::isspace(ch) != 0;
	};
	value.erase(value.begin(),
		   std::find_if(value.begin(), value.end(), [&](char ch) {
			   return !is_space(static_cast<unsigned char>(ch));
		   }));
	value.erase(
		std::find_if(value.rbegin(), value.rend(), [&](char ch) {
			return !is_space(static_cast<unsigned char>(ch));
		}).base(),
		value.end());
	return value;
}

// Extract the Device Number from `ax_efuse_tool --str-get` output.
//
// The first non-empty, non-`#` line carries the value as `[prefix]number`; the
// prefix may itself contain `]`, so multiple brackets can appear. Returns the
// substring after the LAST `]` of that line, trimmed. If that line has no `]`,
// falls back to the whole trimmed line (keeps the previous whole-line
// behavior). Returns std::nullopt when the output has no meaningful line.
//
// `ax_efuse_tool --str-get` is a target-device command; host tests only feed
// sample stdout to verify the parsing contract.
inline std::optional<std::string>
parse_device_number_output(std::string_view output)
{
	std::istringstream lines { std::string { output } };
	std::string line;
	while (std::getline(lines, line)) {
		std::string trimmed = trim_device_number_text(std::move(line));
		if (trimmed.empty() || trimmed[0] == '#') {
			continue;
		}
		if (const auto pos = trimmed.rfind(']');
		    pos != std::string::npos) {
			return trim_device_number_text(trimmed.substr(pos + 1));
		}
		return trimmed;
	}
	return std::nullopt;
}

} // namespace lvgl_main
