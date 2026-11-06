#include "lvgl_main/http_api_contract.hpp"

#include <array>
#include <cctype>
#include <utility>

#include <nlohmann/json.hpp>

namespace lvgl_main::http_api
{

namespace
{

using json = nlohmann::json;

constexpr std::array<std::string_view, 5> kDisplayLabelAllowedFields = {
	"resolution", "fps", "fps_unit", "keyboard_special", "keyboard_normal",
};

constexpr std::array<std::string_view, 3> kHidStateAllowedFields = {
	"mouse_active",
	"keyboard_special",
	"keyboard_normal",
};

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
constexpr std::array<std::string_view, 1> kBackgroundAllowedFields = {
	"image_name",
};
#endif

constexpr std::size_t kMinBearerTokenBytes = 16;

bool is_allowed_display_label_field(std::string_view field)
{
	for (const auto allowed : kDisplayLabelAllowedFields) {
		if (field == allowed) {
			return true;
		}
	}
	return false;
}

bool is_allowed_hid_state_field(std::string_view field)
{
	for (const auto allowed : kHidStateAllowedFields) {
		if (field == allowed) {
			return true;
		}
	}
	return false;
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
bool is_allowed_background_field(std::string_view field)
{
	for (const auto allowed : kBackgroundAllowedFields) {
		if (field == allowed) {
			return true;
		}
	}
	return false;
}

bool has_bmp_extension(std::string_view name)
{
	constexpr std::string_view suffix = ".bmp";
	if (name.size() <= suffix.size()) {
		return false;
	}
	const std::string_view actual =
		name.substr(name.size() - suffix.size());
	for (std::size_t i = 0; i < suffix.size(); ++i) {
		const auto ch = static_cast<unsigned char>(actual[i]);
		if (static_cast<char>(std::tolower(ch)) != suffix[i]) {
			return false;
		}
	}
	return true;
}

bool is_valid_background_image_name(std::string_view image_name)
{
	if (image_name.empty() ||
	    image_name.size() > kMaxBackgroundImageNameBytes) {
		return false;
	}
	if (image_name == "." || image_name == ".." ||
	    !has_bmp_extension(image_name)) {
		return false;
	}
	for (const char ch : image_name) {
		const auto byte = static_cast<unsigned char>(ch);
		if (!(std::isalnum(byte) != 0 || ch == '_' || ch == '-' ||
		      ch == '.')) {
			return false;
		}
	}
	return true;
}
#endif

bool is_valid_token(std::string_view token)
{
	if (token.size() < kMinBearerTokenBytes) {
		return false;
	}
	for (const char ch : token) {
		const auto byte = static_cast<unsigned char>(ch);
		if (byte <= 0x20U || byte == 0x7FU) {
			return false;
		}
	}
	return true;
}

void set_display_label_field(DisplayLabelsUpdate &update, std::string_view key,
			     std::string value)
{
	if (key == "resolution") {
		update.resolution = std::move(value);
	} else if (key == "fps") {
		update.fps = std::move(value);
	} else if (key == "fps_unit") {
		update.fps_unit = std::move(value);
	} else if (key == "keyboard_special") {
		update.keyboard_special = std::move(value);
	} else if (key == "keyboard_normal") {
		update.keyboard_normal = std::move(value);
	}
}

void set_hid_state_string_field(HidStateUpdate &update, std::string_view key,
				std::string value)
{
	if (key == "keyboard_special") {
		update.keyboard_special = std::move(value);
	} else if (key == "keyboard_normal") {
		update.keyboard_normal = std::move(value);
	}
}

DisplayLabelPayloadResult payload_error(DisplayLabelPayloadError error,
					int status_code, std::string detail)
{
	DisplayLabelPayloadResult result;
	result.error = error;
	result.status_code = status_code;
	result.detail = std::move(detail);
	return result;
}

HidStatePayloadResult hid_payload_error(HidStatePayloadError error,
					int status_code, std::string detail)
{
	HidStatePayloadResult result;
	result.error = error;
	result.status_code = status_code;
	result.detail = std::move(detail);
	return result;
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
BackgroundPayloadResult background_payload_error(BackgroundPayloadError error,
						 int status_code,
						 std::string detail)
{
	BackgroundPayloadResult result;
	result.error = error;
	result.status_code = status_code;
	result.detail = std::move(detail);
	return result;
}
#endif

} // namespace

DisplayLabelPayloadResult parse_display_labels_payload(std::string_view body)
{
	if (body.size() > kMaxDisplayLabelsBodyBytes) {
		return payload_error(DisplayLabelPayloadError::BodyTooLarge,
				     413, "request body too large");
	}

	json parsed;
	try {
		parsed = json::parse(body);
	} catch (const std::exception &e) {
		return payload_error(DisplayLabelPayloadError::MalformedJson,
				     400, e.what());
	}

	if (!parsed.is_object()) {
		return payload_error(DisplayLabelPayloadError::InvalidRoot, 400,
				     "request body must be a JSON object");
	}

	DisplayLabelPayloadResult result;
	for (auto it = parsed.cbegin(); it != parsed.cend(); ++it) {
		const std::string &key = it.key();
		if (!is_allowed_display_label_field(key)) {
			return payload_error(
				DisplayLabelPayloadError::UnknownField, 400,
				"unknown field: " + key);
		}
		if (!it.value().is_string()) {
			return payload_error(
				DisplayLabelPayloadError::InvalidFieldType, 400,
				"field must be a string: " + key);
		}

		std::string value = it.value().get<std::string>();
		if (value.size() > kMaxDisplayLabelFieldBytes) {
			return payload_error(
				DisplayLabelPayloadError::FieldTooLong, 400,
				"field too long: " + key);
		}
		set_display_label_field(result.update, key, std::move(value));
	}

	return result;
}

HidStatePayloadResult parse_hid_state_payload(std::string_view body)
{
	if (body.size() > kMaxHidStateBodyBytes) {
		return hid_payload_error(HidStatePayloadError::BodyTooLarge,
					 413, "request body too large");
	}

	json parsed;
	try {
		parsed = json::parse(body);
	} catch (const std::exception &e) {
		return hid_payload_error(HidStatePayloadError::MalformedJson,
					 400, e.what());
	}

	if (!parsed.is_object()) {
		return hid_payload_error(HidStatePayloadError::InvalidRoot, 400,
					 "request body must be a JSON object");
	}

	HidStatePayloadResult result;
	for (auto it = parsed.cbegin(); it != parsed.cend(); ++it) {
		const std::string &key = it.key();
		if (!is_allowed_hid_state_field(key)) {
			return hid_payload_error(
				HidStatePayloadError::UnknownField, 400,
				"unknown field: " + key);
		}

		if (key == "mouse_active") {
			if (!it.value().is_boolean()) {
				return hid_payload_error(
					HidStatePayloadError::InvalidFieldType,
					400, "field must be a bool: " + key);
			}
			result.update.mouse_active = it.value().get<bool>();
			continue;
		}

		if (!it.value().is_string()) {
			return hid_payload_error(
				HidStatePayloadError::InvalidFieldType, 400,
				"field must be a string: " + key);
		}

		std::string value = it.value().get<std::string>();
		if (value.size() > kMaxHidStateStringBytes) {
			return hid_payload_error(
				HidStatePayloadError::FieldTooLong, 400,
				"field too long: " + key);
		}
		set_hid_state_string_field(result.update, key,
					   std::move(value));
	}

	return result;
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
BackgroundPayloadResult parse_background_payload(std::string_view body)
{
	if (body.size() > kMaxBackgroundBodyBytes) {
		return background_payload_error(
			BackgroundPayloadError::BodyTooLarge, 413,
			"request body too large");
	}

	json parsed;
	try {
		parsed = json::parse(body);
	} catch (const std::exception &e) {
		return background_payload_error(
			BackgroundPayloadError::MalformedJson, 400, e.what());
	}

	if (!parsed.is_object()) {
		return background_payload_error(
			BackgroundPayloadError::InvalidRoot, 400,
			"request body must be a JSON object");
	}

	BackgroundPayloadResult result;
	bool has_image_name = false;
	for (auto it = parsed.cbegin(); it != parsed.cend(); ++it) {
		const std::string &key = it.key();
		if (!is_allowed_background_field(key)) {
			return background_payload_error(
				BackgroundPayloadError::UnknownField, 400,
				"unknown field: " + key);
		}
		if (!it.value().is_string()) {
			return background_payload_error(
				BackgroundPayloadError::InvalidFieldType, 400,
				"field must be a string: " + key);
		}

		std::string value = it.value().get<std::string>();
		if (!is_valid_background_image_name(value)) {
			return background_payload_error(
				BackgroundPayloadError::InvalidImageName, 400,
				"image_name must be a BMP file name using A-Z, a-z, 0-9, '.', '_' or '-'");
		}
		result.update.image_name = std::move(value);
		has_image_name = true;
	}

	if (!has_image_name) {
		return background_payload_error(
			BackgroundPayloadError::MissingField, 400,
			"missing required field: image_name");
	}

	return result;
}
#endif

bool is_loopback_bind(std::string_view bind_host)
{
	return bind_host == "127.0.0.1" || bind_host == "::1";
}

bool is_json_content_type(std::string_view content_type)
{
	constexpr std::string_view kJsonContentType = "application/json";
	if (content_type == kJsonContentType) {
		return true;
	}
	return content_type.size() > kJsonContentType.size() &&
	       content_type.substr(0, kJsonContentType.size()) ==
		       kJsonContentType &&
	       content_type[kJsonContentType.size()] == ';';
}

ServerConfigValidation validate_server_config(const ServerConfig &config)
{
	if (config.bind_host.empty()) {
		return { false, ServerConfigError::EmptyBindHost,
			 "HTTP API bind host must not be empty" };
	}
	if (config.port <= 0 || config.port > 65535) {
		return { false, ServerConfigError::InvalidPort,
			 "HTTP API port must be in range 1..65535" };
	}
	if (!config.token.empty() && !is_valid_token(config.token)) {
		return {
			false, ServerConfigError::InvalidToken,
			"http_api.token must be at least 16 non-whitespace bytes"
		};
	}
	if (!is_loopback_bind(config.bind_host) && config.token.empty()) {
		return {
			false, ServerConfigError::TokenRequired,
			"http_api.token is required for non-loopback http_api.bind_host"
		};
	}
	return {};
}

bool is_authorized_bearer(std::string_view authorization_header,
			  std::string_view token)
{
	if (token.empty()) {
		return true;
	}

	constexpr std::string_view kPrefix = "Bearer ";
	if (authorization_header.size() != kPrefix.size() + token.size() ||
	    authorization_header.substr(0, kPrefix.size()) != kPrefix) {
		return false;
	}

	unsigned char diff = 0;
	const auto provided = authorization_header.substr(kPrefix.size());
	for (std::size_t i = 0; i < token.size(); ++i) {
		diff |= static_cast<unsigned char>(provided[i]) ^
			static_cast<unsigned char>(token[i]);
	}
	return diff == 0;
}

} // namespace lvgl_main::http_api
