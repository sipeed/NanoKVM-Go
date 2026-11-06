#include "lvgl_main/wifi_script_parser.hpp"

#include <algorithm>
#include <cctype>
#include <optional>
#include <string>
#include <vector>

#include <nlohmann/json.hpp>

namespace lvgl_main
{

namespace
{

using json = nlohmann::json;

std::string trim_ascii(std::string_view text)
{
	auto begin = text.begin();
	auto end = text.end();
	while (begin != end &&
	       std::isspace(static_cast<unsigned char>(*begin)) != 0) {
		++begin;
	}
	while (begin != end &&
	       std::isspace(static_cast<unsigned char>(*(end - 1))) != 0) {
		--end;
	}
	return std::string(begin, end);
}

std::string first_line(std::string_view text)
{
	const std::size_t line_end = text.find('\n');
	return trim_ascii(text.substr(0, line_end));
}

bool starts_with(std::string_view text, std::string_view prefix)
{
	return text.size() >= prefix.size() &&
	       std::equal(prefix.begin(), prefix.end(), text.begin());
}

std::string output_summary(std::string_view output)
{
	std::string summary = trim_ascii(output);
	if (summary.empty()) {
		return "<empty>";
	}
	std::replace(summary.begin(), summary.end(), '\n', '|');
	constexpr std::size_t kMaxSummaryLength = 160;
	if (summary.size() > kMaxSummaryLength) {
		summary.resize(kMaxSummaryLength);
		summary += "...";
	}
	return summary;
}

bool equals_ascii_case_insensitive(std::string_view lhs, std::string_view rhs)
{
	return lhs.size() == rhs.size() &&
	       std::equal(
		       lhs.begin(), lhs.end(), rhs.begin(), [](char a, char b) {
			       return std::toupper(
					      static_cast<unsigned char>(a)) ==
				      std::toupper(
					      static_cast<unsigned char>(b));
		       });
}

int hex_digit_value(char ch)
{
	if (ch >= '0' && ch <= '9') {
		return ch - '0';
	}
	if (ch >= 'a' && ch <= 'f') {
		return ch - 'a' + 10;
	}
	if (ch >= 'A' && ch <= 'F') {
		return ch - 'A' + 10;
	}
	return -1;
}

char upper_hex_digit(unsigned value)
{
	return static_cast<char>(value < 10U ? '0' + value :
					       'A' + (value - 10U));
}

std::optional<unsigned char> parse_x_escape_byte(std::string_view text,
						 std::size_t offset)
{
	if (offset + 3U >= text.size() || text[offset] != '\\' ||
	    text[offset + 1U] != 'x') {
		return std::nullopt;
	}

	const int high = hex_digit_value(text[offset + 2U]);
	const int low = hex_digit_value(text[offset + 3U]);
	if (high < 0 || low < 0) {
		return std::nullopt;
	}
	return static_cast<unsigned char>((high << 4) | low);
}

void append_json_string_byte(std::string &output, unsigned char byte)
{
	switch (byte) {
	case '"':
		output += "\\\"";
		return;
	case '\\':
		output += "\\\\";
		return;
	case '\b':
		output += "\\b";
		return;
	case '\f':
		output += "\\f";
		return;
	case '\n':
		output += "\\n";
		return;
	case '\r':
		output += "\\r";
		return;
	case '\t':
		output += "\\t";
		return;
	default:
		break;
	}

	if (byte < 0x20U) {
		output += "\\u00";
		output.push_back(upper_hex_digit(byte >> 4U));
		output.push_back(upper_hex_digit(byte & 0x0FU));
		return;
	}

	output.push_back(static_cast<char>(byte));
}

void append_literal_x_escape(std::string &output, unsigned char byte)
{
	output += "\\\\x";
	output.push_back(upper_hex_digit(byte >> 4U));
	output.push_back(upper_hex_digit(byte & 0x0FU));
}

bool is_valid_utf8(const std::vector<unsigned char> &bytes)
{
	std::size_t offset = 0;
	while (offset < bytes.size()) {
		const unsigned char byte = bytes[offset];
		if (byte <= 0x7FU) {
			++offset;
			continue;
		}

		std::size_t continuation_count = 0;
		unsigned char min_second = 0x80U;
		unsigned char max_second = 0xBFU;
		if (byte >= 0xC2U && byte <= 0xDFU) {
			continuation_count = 1U;
		} else if (byte == 0xE0U) {
			continuation_count = 2U;
			min_second = 0xA0U;
		} else if (byte >= 0xE1U && byte <= 0xECU) {
			continuation_count = 2U;
		} else if (byte == 0xEDU) {
			continuation_count = 2U;
			max_second = 0x9FU;
		} else if (byte >= 0xEEU && byte <= 0xEFU) {
			continuation_count = 2U;
		} else if (byte == 0xF0U) {
			continuation_count = 3U;
			min_second = 0x90U;
		} else if (byte >= 0xF1U && byte <= 0xF3U) {
			continuation_count = 3U;
		} else if (byte == 0xF4U) {
			continuation_count = 3U;
			max_second = 0x8FU;
		} else {
			return false;
		}

		if (offset + continuation_count >= bytes.size()) {
			return false;
		}

		for (std::size_t index = 1; index <= continuation_count;
		     ++index) {
			const unsigned char continuation =
				bytes[offset + index];
			const bool first_continuation = index == 1U;
			if (continuation < 0x80U || continuation > 0xBFU ||
			    (first_continuation && continuation < min_second) ||
			    (first_continuation && continuation > max_second)) {
				return false;
			}
		}

		offset += continuation_count + 1U;
	}
	return true;
}

void append_x_escape_byte_run(std::string &output,
			      const std::vector<unsigned char> &bytes)
{
	if (is_valid_utf8(bytes)) {
		for (const unsigned char byte : bytes) {
			append_json_string_byte(output, byte);
		}
		return;
	}

	for (const unsigned char byte : bytes) {
		append_literal_x_escape(output, byte);
	}
}

std::string decode_non_ascii_x_escape_runs(std::string_view input)
{
	std::string decoded;
	decoded.reserve(input.size());
	for (std::size_t offset = 0; offset < input.size();) {
		std::size_t escape_offset = offset;
		std::vector<unsigned char> bytes;
		bool has_non_ascii = false;
		while (const auto byte =
			       parse_x_escape_byte(input, escape_offset)) {
			bytes.push_back(*byte);
			has_non_ascii = has_non_ascii || *byte >= 0x80U;
			escape_offset += 4U;
		}

		if (!bytes.empty()) {
			if (has_non_ascii && is_valid_utf8(bytes)) {
				for (const unsigned char byte : bytes) {
					decoded.push_back(
						static_cast<char>(byte));
				}
			} else {
				decoded.append(input.substr(
					offset, escape_offset - offset));
			}
			offset = escape_offset;
			continue;
		}

		decoded.push_back(input[offset]);
		++offset;
	}
	return decoded;
}

std::optional<std::string>
normalize_json_string_x_hex_escapes(std::string_view input)
{
	std::string normalized;
	normalized.reserve(input.size());
	bool in_string = false;
	bool changed = false;

	for (std::size_t offset = 0; offset < input.size();) {
		const char ch = input[offset];
		if (!in_string) {
			normalized.push_back(ch);
			if (ch == '"') {
				in_string = true;
			}
			++offset;
			continue;
		}

		if (ch == '\\') {
			std::size_t escape_offset = offset;
			std::vector<unsigned char> bytes;
			while (const auto byte = parse_x_escape_byte(
				       input, escape_offset)) {
				bytes.push_back(*byte);
				escape_offset += 4U;
			}

			if (!bytes.empty()) {
				append_x_escape_byte_run(normalized, bytes);
				changed = true;
				offset = escape_offset;
				continue;
			}

			if (offset + 1U < input.size() &&
			    input[offset + 1U] == 'x') {
				normalized += "\\\\x";
				changed = true;
				offset += 2U;
				continue;
			}

			normalized.push_back(ch);
			++offset;
			if (offset < input.size()) {
				normalized.push_back(input[offset]);
				++offset;
			}
			continue;
		}

		normalized.push_back(ch);
		if (ch == '"') {
			in_string = false;
		}
		++offset;
	}

	if (!changed) {
		return std::nullopt;
	}
	return normalized;
}

std::optional<std::string> json_string(const json &value, std::string_view key)
{
	if (!value.is_object()) {
		return std::nullopt;
	}
	const auto it = value.find(std::string(key));
	if (it == value.end() || !it->is_string()) {
		return std::nullopt;
	}
	return it->get<std::string>();
}

std::optional<std::string> trimmed_json_string(const json &value,
							 std::string_view key)
{
	if (const auto parsed = json_string(value, key)) {
		return trim_ascii(*parsed);
	}
	return std::nullopt;
}

int json_int_or_zero(const json &value, std::string_view key)
{
	if (!value.is_object()) {
		return 0;
	}
	const auto it = value.find(std::string(key));
	if (it == value.end()) {
		return 0;
	}
	if (it->is_number_integer() || it->is_number_unsigned()) {
		return it->get<int>();
	}
	if (it->is_string()) {
		try {
			return std::stoi(it->get<std::string>());
		} catch (const std::exception &) {
			return 0;
		}
	}
	return 0;
}

std::optional<std::string> find_status_line(std::string_view output,
					    std::string_view prefix)
{
	std::size_t offset = 0;
	while (offset <= output.size()) {
		const std::size_t line_end = output.find('\n', offset);
		const std::string line =
			trim_ascii(output.substr(offset, line_end - offset));
		if (starts_with(line, prefix)) {
			return line;
		}
		if (line_end == std::string_view::npos) {
			break;
		}
		offset = line_end + 1U;
	}
	return std::nullopt;
}

std::string first_token(std::string_view text)
{
	const std::string trimmed = trim_ascii(text);
	const std::size_t token_end = trimmed.find_first_of(" \t\r\n");
	return trimmed.substr(0, token_end);
}

void add_or_replace_strongest(std::vector<WifiScanNetwork> &networks,
			      WifiScanNetwork network)
{
	const auto existing =
		std::find_if(networks.begin(), networks.end(),
			     [&network](const WifiScanNetwork &item) {
				     return item.ssid == network.ssid;
			     });
	if (existing == networks.end()) {
		networks.push_back(std::move(network));
		return;
	}
	if (network.signal > existing->signal) {
		*existing = std::move(network);
	}
}

} // namespace

WifiScriptStatus parse_wifi_status_output(std::string_view output)
{
	const std::string line = first_line(output);
	if (starts_with(line, "CONNECTED ")) {
		return { .kind = WifiScriptStatusKind::Connected,
			 .ssid = decode_non_ascii_x_escape_runs(trim_ascii(
				 std::string_view(line).substr(10))) };
	}
	if (starts_with(line, "AP ")) {
		return { .kind = WifiScriptStatusKind::Ap,
			 .ssid = decode_non_ascii_x_escape_runs(trim_ascii(
				 std::string_view(line).substr(3))) };
	}
	if (line == "DISCONNECTED") {
		return { .kind = WifiScriptStatusKind::Disconnected };
	}
	if (line == "ERROR: NO_INTERFACE") {
		return { .kind = WifiScriptStatusKind::NoInterface };
	}
	return {};
}

std::optional<bool> parse_wifi_boot_enabled_output(std::string_view output)
{
	const std::string value = trim_ascii(output);
	if (value == "true") {
		return true;
	}
	if (value == "false") {
		return false;
	}
	return std::nullopt;
}

std::optional<int> parse_wifi_device_count_output(std::string_view output)
{
	const std::string line = first_line(output);
	if (line.empty()) {
		return std::nullopt;
	}

	int count = 0;
	for (const char ch : line) {
		if (ch < '0' || ch > '9') {
			return std::nullopt;
		}
		if (count > 10000) {
			return std::nullopt;
		}
		count = count * 10 + (ch - '0');
	}
	return count;
}

WifiScanResult parse_wifi_scan_output(std::string_view output)
{
	WifiScanResult result;
	if (trim_ascii(output).empty()) {
		result.ok = false;
		result.error = "EMPTY_OUTPUT";
		return result;
	}

	json root = json::parse(output, nullptr, false);
	if (root.is_discarded()) {
		if (const auto normalized =
			    normalize_json_string_x_hex_escapes(output)) {
			root = json::parse(*normalized, nullptr, false);
		}
	}
	if (root.is_discarded() || !root.is_array()) {
		result.ok = false;
		result.error = "INVALID_JSON";
		return result;
	}

	for (const auto &item : root) {
		auto ssid = json_string(item, "ssid");
		if (!ssid || ssid->empty()) {
			continue;
		}
		auto security = trimmed_json_string(item, "security").value_or("");
		add_or_replace_strongest(
			result.networks,
			WifiScanNetwork {
				.ssid = std::move(*ssid),
					.bssid = trimmed_json_string(item, "bssid").value_or(""),
				.signal = json_int_or_zero(item, "signal"),
				.frequency =
					json_int_or_zero(item, "frequency"),
				.security = security,
				.supported = !equals_ascii_case_insensitive(
					security, "EAP") });
	}
	return result;
}

WifiConnectResult parse_wifi_connect_output(std::string_view output,
					    int exit_code)
{
	WifiConnectResult result { .raw_summary = output_summary(output) };

	if (const auto line = find_status_line(output, "OK: Connected to ");
	    line && exit_code == 0) {
		result.status = WifiConnectStatus::Success;
		result.ssid = decode_non_ascii_x_escape_runs(
			trim_ascii(std::string_view(*line).substr(17)));
		return result;
	}

	if (const auto line = find_status_line(output, "ERROR:"); line) {
		result.status = WifiConnectStatus::Failure;
		result.error_code =
			first_token(std::string_view(*line).substr(6));
		return result;
	}

	if (exit_code != 0) {
		result.status = WifiConnectStatus::Failure;
	}
	return result;
}

WifiSavedNetworkAvailability
parse_wifi_saved_networks_output(std::string_view output)
{
	const auto trimmed = trim_ascii(output);
	if (trimmed.empty()) {
		return WifiSavedNetworkAvailability::Unknown;
	}

	const auto classify_saved_json =
		[](const json &parsed) -> WifiSavedNetworkAvailability {
		if (!parsed.is_array()) {
			return WifiSavedNetworkAvailability::Unknown;
		}

		for (const auto &entry : parsed) {
			if (!entry.is_object()) {
				continue;
			}
			const auto ssid_it = entry.find("ssid");
			if (ssid_it == entry.end() || !ssid_it->is_string()) {
				continue;
			}
			const auto &ssid = ssid_it->get_ref<const std::string &>();
			if (!ssid.empty()) {
				return WifiSavedNetworkAvailability::Present;
			}
		}
		return WifiSavedNetworkAvailability::Absent;
	};

	try {
		return classify_saved_json(json::parse(trimmed));
	} catch (const json::exception &) {
	}

	// The deployed wifi.sh saved command may emit literal \xNN byte escapes
	// for non-ASCII SSIDs, which is not standard JSON. Retry through the
	// same target-style normalizer used by the scan parser so a valid
	// saved array is not misreported as Unknown.
	if (const auto normalized =
		    normalize_json_string_x_hex_escapes(trimmed)) {
		try {
			return classify_saved_json(json::parse(*normalized));
		} catch (const json::exception &) {
		}
	}

	return WifiSavedNetworkAvailability::Unknown;
}

} // namespace lvgl_main
