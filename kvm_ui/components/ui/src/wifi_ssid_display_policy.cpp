#include "ui/wifi_ssid_display_policy.hpp"

#include <cstddef>
#include <cstdint>
#include <optional>
#include <vector>

namespace ui
{

namespace
{

constexpr char kHexDigits[] = "0123456789ABCDEF";
constexpr std::string_view kEmojiPlaceholder = "\xE2\x96\xA1";

// Script and Emoji ranges are frozen from Unicode 15.1 data. Update the
// ranges and their regression cases together after a Unicode-data review.

struct Utf8Codepoint {
	uint32_t value { 0 };
	std::size_t byte_offset { 0 };
	std::size_t byte_count { 0 };
};

std::optional<std::vector<Utf8Codepoint>> decode_utf8(std::string_view text)
{
	std::vector<Utf8Codepoint> codepoints;
	codepoints.reserve(text.size());
	for (std::size_t offset = 0; offset < text.size();) {
		const auto first = static_cast<unsigned char>(text[offset]);
		if (first <= 0x7FU) {
			codepoints.push_back({ first, offset, 1U });
			++offset;
			continue;
		}

		std::size_t continuation_count = 0;
		uint32_t value = 0;
		uint32_t minimum = 0;
		if (first >= 0xC2U && first <= 0xDFU) {
			continuation_count = 1U;
			value = first & 0x1FU;
			minimum = 0x80U;
		} else if (first >= 0xE0U && first <= 0xEFU) {
			continuation_count = 2U;
			value = first & 0x0FU;
			minimum = 0x800U;
		} else if (first >= 0xF0U && first <= 0xF4U) {
			continuation_count = 3U;
			value = first & 0x07U;
			minimum = 0x10000U;
		} else {
			return std::nullopt;
		}

		if (offset + continuation_count >= text.size()) {
			return std::nullopt;
		}
		for (std::size_t index = 1U; index <= continuation_count; ++index) {
			const auto continuation =
				static_cast<unsigned char>(text[offset + index]);
			if (continuation < 0x80U || continuation > 0xBFU) {
				return std::nullopt;
			}
			value = (value << 6U) | (continuation & 0x3FU);
		}
		if (value < minimum || value > 0x10FFFFU ||
		    (value >= 0xD800U && value <= 0xDFFFU)) {
			return std::nullopt;
		}
		codepoints.push_back(
			{ value, offset, continuation_count + 1U });
		offset += continuation_count + 1U;
	}
	return codepoints;
}

void append_byte_escapes(std::string &output, std::string_view source,
			 std::size_t offset, std::size_t count)
{
	for (std::size_t index = 0; index < count; ++index) {
		const auto byte = static_cast<unsigned char>(source[offset + index]);
		output += "\\x";
		output.push_back(kHexDigits[(byte >> 4U) & 0x0FU]);
		output.push_back(kHexDigits[byte & 0x0FU]);
	}
}

std::string escaped_bytes(std::string_view source)
{
	std::string output;
	output.reserve(source.size() * 4U);
	append_byte_escapes(output, source, 0, source.size());
	return output;
}

bool is_bidi_control(uint32_t value)
{
	return value == 0x061CU || value == 0x200EU || value == 0x200FU ||
	       (value >= 0x202AU && value <= 0x202EU) ||
	       (value >= 0x2066U && value <= 0x2069U);
}

bool is_invisible_format(uint32_t value)
{
	return value == 0x00ADU || value == 0x034FU ||
	       (value >= 0x180BU && value <= 0x180FU) ||
	       (value >= 0x200BU && value <= 0x200DU) ||
	       value == 0x2028U || value == 0x2029U ||
	       (value >= 0x2060U && value <= 0x2064U) ||
	       (value >= 0x206AU && value <= 0x206FU) ||
	       (value >= 0xFE00U && value <= 0xFE0FU) || value == 0xFEFFU ||
	       (value >= 0xFFF9U && value <= 0xFFFB) ||
	       (value >= 0x1D173U && value <= 0x1D17AU) ||
	       (value >= 0xE0000U && value <= 0xE0FFFU);
}

bool is_inherited_mark(uint32_t value)
{
	return (value >= 0x0300U && value <= 0x036FU) ||
	       (value >= 0x1AB0U && value <= 0x1AFFU) ||
	       (value >= 0x1DC0U && value <= 0x1DFFU) ||
	       (value >= 0x20D0U && value <= 0x20FFU) ||
	       (value >= 0xFE20U && value <= 0xFE2FU);
}

bool is_latin(uint32_t value)
{
	return (value >= 0x0041U && value <= 0x005AU) ||
	       (value >= 0x0061U && value <= 0x007AU) ||
	       (value >= 0x00C0U && value <= 0x02AFU) ||
	       (value >= 0x02B0U && value <= 0x02B8U) ||
	       (value >= 0x02E0U && value <= 0x02E4U) ||
	       (value >= 0x1D00U && value <= 0x1DBFU) ||
	       (value >= 0x1E00U && value <= 0x1EFFU) ||
	       (value >= 0x2071U && value <= 0x207FU) ||
	       (value >= 0x2090U && value <= 0x209CU) ||
	       (value >= 0x2C60U && value <= 0x2C7FU) ||
	       (value >= 0xA720U && value <= 0xA7FFU) ||
	       (value >= 0xAB30U && value <= 0xAB6FU) ||
	       (value >= 0xFB00U && value <= 0xFB06U) ||
	       (value >= 0x10780U && value <= 0x107BFU) ||
	       (value >= 0x1DF00U && value <= 0x1DFFFU) || value == 0x212AU ||
	       value == 0x212BU || value == 0x2132U || value == 0x214EU;
}

bool is_greek(uint32_t value)
{
	return (value >= 0x0370U && value <= 0x0373U) ||
	       (value >= 0x0375U && value <= 0x0377U) ||
	       (value >= 0x037AU && value <= 0x037DU) || value == 0x037FU ||
	       (value >= 0x0384U && value <= 0x03E1U) ||
	       (value >= 0x03F0U && value <= 0x03FFU) ||
	       (value >= 0x1F00U && value <= 0x1FFFU);
}

bool is_cyrillic(uint32_t value)
{
	return (value >= 0x0400U && value <= 0x052FU) ||
	       (value >= 0x1C80U && value <= 0x1C8AU) ||
	       (value >= 0x2DE0U && value <= 0x2DFFU) ||
	       (value >= 0xA640U && value <= 0xA69FU) ||
	       (value >= 0x1E030U && value <= 0x1E08FU);
}

bool is_han(uint32_t value)
{
	return (value >= 0x2E80U && value <= 0x2FD5U) ||
	       (value >= 0x31C0U && value <= 0x31EFU) ||
	       (value >= 0x3400U && value <= 0x4DBFU) ||
	       (value >= 0x4E00U && value <= 0x9FFFU) ||
	       (value >= 0xF900U && value <= 0xFAFFU) ||
	       (value >= 0x20000U && value <= 0x2EBEFU) ||
	       (value >= 0x2EBF0U && value <= 0x2EE5DU) ||
	       (value >= 0x30000U && value <= 0x323AFU) ||
	       (value >= 0x2F800U && value <= 0x2FA1FU);
}

bool is_hiragana(uint32_t value)
{
	return (value >= 0x3040U && value <= 0x309FU) ||
	       (value >= 0x1B001U && value <= 0x1B11FU) || value == 0x1B132U ||
	       (value >= 0x1B150U && value <= 0x1B152U);
}

bool is_katakana(uint32_t value)
{
	return (value >= 0x30A0U && value <= 0x30FFU) ||
	       (value >= 0x31F0U && value <= 0x31FFU) ||
	       (value >= 0xFF66U && value <= 0xFF9FU) ||
	       (value >= 0x1AFF0U && value <= 0x1AFF3U) ||
	       (value >= 0x1AFF5U && value <= 0x1AFFBU) ||
	       (value >= 0x1AFFDU && value <= 0x1AFFEU) || value == 0x1B000U ||
	       (value >= 0x1B120U && value <= 0x1B122U) || value == 0x1B155U ||
	       (value >= 0x1B164U && value <= 0x1B167U);
}

bool is_hangul(uint32_t value)
{
	return (value >= 0x1100U && value <= 0x11FFU) ||
	       (value >= 0x3130U && value <= 0x318FU) ||
	       (value >= 0xA960U && value <= 0xA97FU) ||
	       (value >= 0xAC00U && value <= 0xD7FFU) ||
	       (value >= 0xFFA0U && value <= 0xFFDCU);
}

bool is_common(uint32_t value)
{
	return (value >= 0x0020U && value <= 0x007EU) ||
	       (value >= 0x00A0U && value <= 0x00BFU) ||
	       (value >= 0x2000U && value <= 0x206FU) ||
	       (value >= 0x20A0U && value <= 0x20CFU) ||
	       (value >= 0x2100U && value <= 0x214FU) ||
	       (value >= 0x2150U && value <= 0x218FU) ||
	       (value >= 0x2190U && value <= 0x2BFFU) ||
	       (value >= 0x2E00U && value <= 0x2E7FU) ||
	       (value >= 0x3000U && value <= 0x303FU) ||
	       (value >= 0x3190U && value <= 0x319FU) ||
	       (value >= 0x3200U && value <= 0x33FFU) ||
	       (value >= 0xFE10U && value <= 0xFE1FU) ||
	       (value >= 0xFE30U && value <= 0xFE6FU) ||
	       (value >= 0xFF01U && value <= 0xFF65U) ||
	       (value >= 0xFFE0U && value <= 0xFFEEU) ||
	       (value >= 0xFFF0U && value <= 0xFFFDU);
}

bool is_allowed_direct(uint32_t value)
{
	return is_latin(value) || is_greek(value) || is_cyrillic(value) ||
	       is_han(value) || is_hiragana(value) || is_katakana(value) ||
	       is_hangul(value) || is_inherited_mark(value) || is_common(value);
}

bool is_emoji_variation_selector(uint32_t value)
{
	return value == 0xFE0EU || value == 0xFE0FU;
}

bool is_skin_tone(uint32_t value)
{
	return value >= 0x1F3FBU && value <= 0x1F3FFU;
}

bool is_regional_indicator(uint32_t value)
{
	return value >= 0x1F1E6U && value <= 0x1F1FFU;
}

bool is_emoji_base(uint32_t value)
{
	return value == 0x00A9U || value == 0x00AEU || value == 0x203CU ||
	       value == 0x2049U || value == 0x2122U || value == 0x2139U ||
	       (value >= 0x231AU && value <= 0x231BU) ||
	       (value >= 0x23E9U && value <= 0x23F3U) ||
	       (value >= 0x23F8U && value <= 0x23FAU) ||
	       (value >= 0x25AAU && value <= 0x25ABU) || value == 0x25B6U ||
	       value == 0x25C0U || (value >= 0x25FBU && value <= 0x25FEU) ||
	       (value >= 0x2600U && value <= 0x27BFU) ||
	       (value >= 0x2934U && value <= 0x2935U) ||
	       (value >= 0x2B05U && value <= 0x2B07U) ||
	       (value >= 0x2B1BU && value <= 0x2B1CU) || value == 0x2B50U ||
	       value == 0x2B55U || value == 0x3030U || value == 0x303DU ||
	       value == 0x3297U || value == 0x3299U ||
	       (value >= 0x1F000U && value <= 0x1FAFFU);
}

bool is_keycap_base(uint32_t value)
{
	return value == '#' || value == '*' || (value >= '0' && value <= '9');
}

std::size_t consume_emoji_modifiers(const std::vector<Utf8Codepoint> &points,
				    std::size_t index)
{
	while (index < points.size()) {
		const uint32_t value = points[index].value;
		if (!is_emoji_variation_selector(value) && !is_skin_tone(value) &&
		    value != 0x20E3U && !(value >= 0xE0020U && value <= 0xE007FU)) {
			break;
		}
		++index;
	}
	return index;
}

std::size_t consume_emoji_sequence(const std::vector<Utf8Codepoint> &points,
				  std::size_t index)
{
	const bool regional_indicator = is_regional_indicator(points[index].value);
	index = consume_emoji_modifiers(points, index + 1U);
	if (regional_indicator && index < points.size() &&
	    is_regional_indicator(points[index].value)) {
		index = consume_emoji_modifiers(points, index + 1U);
	}

	while (index + 1U < points.size() && points[index].value == 0x200DU &&
	       (is_emoji_base(points[index + 1U].value) ||
		is_skin_tone(points[index + 1U].value))) {
		index = consume_emoji_modifiers(points, index + 2U);
	}
	return index;
}

std::optional<std::size_t>
keycap_sequence_end(const std::vector<Utf8Codepoint> &points,
			    std::size_t index)
{
	if (!is_keycap_base(points[index].value)) {
		return std::nullopt;
	}
	std::size_t next = index + 1U;
	if (next < points.size() && is_emoji_variation_selector(points[next].value)) {
		++next;
	}
	if (next < points.size() && points[next].value == 0x20E3U) {
		return next + 1U;
	}
	return std::nullopt;
}

} // namespace

WifiSsidDisplay make_wifi_ssid_display(std::string_view raw_ssid)
{
	if (raw_ssid.empty() || raw_ssid.find('\0') != std::string_view::npos) {
		return {};
	}

	const auto decoded = decode_utf8(raw_ssid);
	if (!decoded.has_value()) {
		return { .visible = true, .text = escaped_bytes(raw_ssid) };
	}

	WifiSsidDisplay display { .visible = true };
	display.text.reserve(raw_ssid.size());
	for (std::size_t index = 0; index < decoded->size();) {
		const Utf8Codepoint &point = (*decoded)[index];
		if (const auto keycap_end = keycap_sequence_end(*decoded, index)) {
			display.text.append(kEmojiPlaceholder.data(),
					    kEmojiPlaceholder.size());
			index = *keycap_end;
			continue;
		}
		if (is_emoji_base(point.value) || is_skin_tone(point.value)) {
			display.text.append(kEmojiPlaceholder.data(),
					    kEmojiPlaceholder.size());
			index = consume_emoji_sequence(*decoded, index);
			continue;
		}
		if (is_bidi_control(point.value)) {
			return {};
		}
		if (point.value <= 0x1FU ||
		    (point.value >= 0x7FU && point.value <= 0x9FU) ||
		    is_invisible_format(point.value)) {
			append_byte_escapes(display.text, raw_ssid, point.byte_offset,
					    point.byte_count);
			++index;
			continue;
		}
		if (!is_allowed_direct(point.value)) {
			return {};
		}
		display.text.append(raw_ssid.data() + point.byte_offset,
				    point.byte_count);
		++index;
	}
	return display;
}

} // namespace ui
