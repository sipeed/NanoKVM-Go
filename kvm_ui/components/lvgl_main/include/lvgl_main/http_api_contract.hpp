#pragma once

#include <cstddef>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main::http_api
{

constexpr std::size_t kMaxDisplayLabelsBodyBytes = 1024;
constexpr std::size_t kMaxDisplayLabelFieldBytes = 64;
constexpr std::size_t kMaxHidStateBodyBytes = kMaxDisplayLabelsBodyBytes;
constexpr std::size_t kMaxHidStateStringBytes = kMaxDisplayLabelFieldBytes;
#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
constexpr std::size_t kMaxBackgroundBodyBytes = 256;
constexpr std::size_t kMaxBackgroundImageNameBytes = 128;
#endif

struct DisplayLabelsUpdate {
	std::optional<std::string> resolution;
	std::optional<std::string> fps;
	std::optional<std::string> fps_unit;
	std::optional<std::string> keyboard_special;
	std::optional<std::string> keyboard_normal;
};

enum class DisplayLabelPayloadError {
	None,
	BodyTooLarge,
	MalformedJson,
	InvalidRoot,
	UnknownField,
	InvalidFieldType,
	FieldTooLong,
};

struct DisplayLabelPayloadResult {
	DisplayLabelsUpdate update;
	DisplayLabelPayloadError error { DisplayLabelPayloadError::None };
	int status_code { 200 };
	std::string detail;

	bool ok() const
	{
		return error == DisplayLabelPayloadError::None;
	}
};

DisplayLabelPayloadResult parse_display_labels_payload(std::string_view body);

struct HidStateUpdate {
	std::optional<bool> mouse_active;
	std::optional<std::string> keyboard_special;
	std::optional<std::string> keyboard_normal;
};

enum class HidStatePayloadError {
	None,
	BodyTooLarge,
	MalformedJson,
	InvalidRoot,
	UnknownField,
	InvalidFieldType,
	FieldTooLong,
};

struct HidStatePayloadResult {
	HidStateUpdate update;
	HidStatePayloadError error { HidStatePayloadError::None };
	int status_code { 200 };
	std::string detail;

	bool ok() const
	{
		return error == HidStatePayloadError::None;
	}
};

HidStatePayloadResult parse_hid_state_payload(std::string_view body);

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
struct BackgroundUpdate {
	std::string image_name;
};

enum class BackgroundPayloadError {
	None,
	BodyTooLarge,
	MalformedJson,
	InvalidRoot,
	UnknownField,
	MissingField,
	InvalidFieldType,
	InvalidImageName,
};

struct BackgroundPayloadResult {
	BackgroundUpdate update;
	BackgroundPayloadError error { BackgroundPayloadError::None };
	int status_code { 200 };
	std::string detail;

	bool ok() const
	{
		return error == BackgroundPayloadError::None;
	}
};

BackgroundPayloadResult parse_background_payload(std::string_view body);
#endif

struct ServerConfig {
	std::string bind_host;
	int port { 0 };
	std::string token;
};

enum class ServerConfigError {
	None,
	InvalidPort,
	EmptyBindHost,
	TokenRequired,
	InvalidToken,
};

struct ServerConfigValidation {
	bool allowed { true };
	ServerConfigError error { ServerConfigError::None };
	std::string detail;
};

bool is_loopback_bind(std::string_view bind_host);
bool is_json_content_type(std::string_view content_type);
ServerConfigValidation validate_server_config(const ServerConfig &config);
bool is_authorized_bearer(std::string_view authorization_header,
			  std::string_view token);

} // namespace lvgl_main::http_api
