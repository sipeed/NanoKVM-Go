#pragma once

#include <cctype>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main
{

inline std::string trim_hardware_version_text(std::string_view text)
{
	auto begin = text.begin();
	auto end = text.end();
	auto is_space = [](unsigned char ch) {
		return std::isspace(ch) != 0;
	};

	while (begin != end && is_space(static_cast<unsigned char>(*begin))) {
		++begin;
	}
	while (begin != end &&
	       is_space(static_cast<unsigned char>(*(end - 1)))) {
		--end;
	}
	return std::string(begin, end);
}

inline std::optional<std::string>
hardware_version_from_chip_type(std::string_view chip_type)
{
	const std::string text = trim_hardware_version_text(chip_type);
	if (text == "AX620QF_CHIP") {
		return std::string("Go");
	}
	if (text == "AX620QE_CHIP") {
		return std::string("Go+");
	}
	return std::nullopt;
}

} // namespace lvgl_main
