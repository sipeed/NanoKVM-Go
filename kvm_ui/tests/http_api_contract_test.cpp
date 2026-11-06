#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config_defaults.hpp"
#include "lvgl_main/http_api_contract.hpp"

using namespace lvgl_main::http_api;

namespace
{

constexpr const char *kToken = "0123456789abcdef0123456789abcdef";
constexpr int kDefaultHttpApiPort =
	lvgl_main::app_config_defaults::kHttpApiPort;

} // namespace

TEST_CASE("display label payload rejects malformed or oversized JSON")
{
	const auto malformed = parse_display_labels_payload("{");
	REQUIRE_FALSE(malformed.ok());
	CHECK(malformed.status_code == 400);
	CHECK(malformed.error == DisplayLabelPayloadError::MalformedJson);

	const std::string oversized(kMaxDisplayLabelsBodyBytes + 1U, 'x');
	const auto too_large = parse_display_labels_payload(oversized);
	REQUIRE_FALSE(too_large.ok());
	CHECK(too_large.status_code == 413);
	CHECK(too_large.error == DisplayLabelPayloadError::BodyTooLarge);
}

TEST_CASE("display label payload rejects invalid schema")
{
	const auto unknown = parse_display_labels_payload(
		R"({"resolution":"1920 * 1080","extra":"x"})");
	REQUIRE_FALSE(unknown.ok());
	CHECK(unknown.status_code == 400);
	CHECK(unknown.error == DisplayLabelPayloadError::UnknownField);

	const auto hid_field =
		parse_display_labels_payload(R"({"mouse_active":true})");
	REQUIRE_FALSE(hid_field.ok());
	CHECK(hid_field.status_code == 400);
	CHECK(hid_field.error == DisplayLabelPayloadError::UnknownField);

	const auto invalid_type = parse_display_labels_payload(R"({"fps":60})");
	REQUIRE_FALSE(invalid_type.ok());
	CHECK(invalid_type.status_code == 400);
	CHECK(invalid_type.error == DisplayLabelPayloadError::InvalidFieldType);

	const std::string long_value(kMaxDisplayLabelFieldBytes + 1U, 'a');
	const auto too_long = parse_display_labels_payload(
		std::string(R"({"keyboard_normal":")") + long_value + R"("})");
	REQUIRE_FALSE(too_long.ok());
	CHECK(too_long.status_code == 400);
	CHECK(too_long.error == DisplayLabelPayloadError::FieldTooLong);
}

TEST_CASE("display label payload accepts allowed string fields")
{
	const auto parsed = parse_display_labels_payload(
		R"({"resolution":"1920 * 1080","fps":"60.0","fps_unit":"FPS","keyboard_special":"Cmd","keyboard_normal":"c"})");

	REQUIRE(parsed.ok());
	REQUIRE(parsed.update.resolution.has_value());
	REQUIRE(parsed.update.fps.has_value());
	REQUIRE(parsed.update.fps_unit.has_value());
	REQUIRE(parsed.update.keyboard_special.has_value());
	REQUIRE(parsed.update.keyboard_normal.has_value());
	CHECK(*parsed.update.resolution == "1920 * 1080");
	CHECK(*parsed.update.fps == "60.0");
	CHECK(*parsed.update.fps_unit == "FPS");
	CHECK(*parsed.update.keyboard_special == "Cmd");
	CHECK(*parsed.update.keyboard_normal == "c");
}

TEST_CASE("HID state payload rejects malformed or oversized JSON")
{
	const auto malformed = parse_hid_state_payload("{");
	REQUIRE_FALSE(malformed.ok());
	CHECK(malformed.status_code == 400);
	CHECK(malformed.error == HidStatePayloadError::MalformedJson);

	const std::string oversized(kMaxHidStateBodyBytes + 1U, 'x');
	const auto too_large = parse_hid_state_payload(oversized);
	REQUIRE_FALSE(too_large.ok());
	CHECK(too_large.status_code == 413);
	CHECK(too_large.error == HidStatePayloadError::BodyTooLarge);
}

TEST_CASE("HID state payload rejects invalid schema")
{
	const auto invalid_root =
		parse_hid_state_payload(R"(["mouse_active"])");
	REQUIRE_FALSE(invalid_root.ok());
	CHECK(invalid_root.status_code == 400);
	CHECK(invalid_root.error == HidStatePayloadError::InvalidRoot);

	const auto unknown =
		parse_hid_state_payload(R"({"mouse_active":true,"extra":"x"})");
	REQUIRE_FALSE(unknown.ok());
	CHECK(unknown.status_code == 400);
	CHECK(unknown.error == HidStatePayloadError::UnknownField);

	const auto string_bool =
		parse_hid_state_payload(R"({"mouse_active":"true"})");
	REQUIRE_FALSE(string_bool.ok());
	CHECK(string_bool.status_code == 400);
	CHECK(string_bool.error == HidStatePayloadError::InvalidFieldType);

	const auto invalid_string =
		parse_hid_state_payload(R"({"keyboard_normal":false})");
	REQUIRE_FALSE(invalid_string.ok());
	CHECK(invalid_string.status_code == 400);
	CHECK(invalid_string.error == HidStatePayloadError::InvalidFieldType);

	const std::string long_value(kMaxHidStateStringBytes + 1U, 'a');
	const auto too_long = parse_hid_state_payload(
		std::string(R"({"keyboard_special":")") + long_value + R"("})");
	REQUIRE_FALSE(too_long.ok());
	CHECK(too_long.status_code == 400);
	CHECK(too_long.error == HidStatePayloadError::FieldTooLong);
}

TEST_CASE("HID state payload accepts optional fields")
{
	const auto empty = parse_hid_state_payload(R"({})");
	REQUIRE(empty.ok());
	CHECK_FALSE(empty.update.mouse_active.has_value());
	CHECK_FALSE(empty.update.keyboard_special.has_value());
	CHECK_FALSE(empty.update.keyboard_normal.has_value());

	const auto parsed = parse_hid_state_payload(
		R"({"mouse_active":true,"keyboard_special":"ctrl","keyboard_normal":"a"})");
	REQUIRE(parsed.ok());
	REQUIRE(parsed.update.mouse_active.has_value());
	REQUIRE(parsed.update.keyboard_special.has_value());
	REQUIRE(parsed.update.keyboard_normal.has_value());
	CHECK(*parsed.update.mouse_active);
	CHECK(*parsed.update.keyboard_special == "ctrl");
	CHECK(*parsed.update.keyboard_normal == "a");
}

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
TEST_CASE("background payload rejects invalid schema")
{
	const auto malformed = parse_background_payload("{");
	REQUIRE_FALSE(malformed.ok());
	CHECK(malformed.status_code == 400);
	CHECK(malformed.error == BackgroundPayloadError::MalformedJson);

	const std::string oversized(kMaxBackgroundBodyBytes + 1U, 'x');
	const auto too_large = parse_background_payload(oversized);
	REQUIRE_FALSE(too_large.ok());
	CHECK(too_large.status_code == 413);
	CHECK(too_large.error == BackgroundPayloadError::BodyTooLarge);

	const auto invalid_root = parse_background_payload(R"(["bg.bmp"])");
	REQUIRE_FALSE(invalid_root.ok());
	CHECK(invalid_root.error == BackgroundPayloadError::InvalidRoot);

	const auto missing = parse_background_payload(R"({})");
	REQUIRE_FALSE(missing.ok());
	CHECK(missing.error == BackgroundPayloadError::MissingField);

	const auto unknown = parse_background_payload(
		R"({"image_name":"bg.bmp","extra":"x"})");
	REQUIRE_FALSE(unknown.ok());
	CHECK(unknown.error == BackgroundPayloadError::UnknownField);

	const auto invalid_type =
		parse_background_payload(R"({"image_name":1})");
	REQUIRE_FALSE(invalid_type.ok());
	CHECK(invalid_type.error == BackgroundPayloadError::InvalidFieldType);

	const auto traversal =
		parse_background_payload(R"({"image_name":"../bg.bmp"})");
	REQUIRE_FALSE(traversal.ok());
	CHECK(traversal.error == BackgroundPayloadError::InvalidImageName);

	const auto wrong_ext =
		parse_background_payload(R"({"image_name":"bg.png"})");
	REQUIRE_FALSE(wrong_ext.ok());
	CHECK(wrong_ext.error == BackgroundPayloadError::InvalidImageName);
}

TEST_CASE("background payload accepts BMP file name")
{
	const auto parsed =
		parse_background_payload(R"({"image_name":"dial_bg-01.BMP"})");

	REQUIRE(parsed.ok());
	CHECK(parsed.update.image_name == "dial_bg-01.BMP");
}
#endif

TEST_CASE("HTTP API auth policy protects non-loopback binds")
{
	CHECK(kDefaultHttpApiPort == 65501);
	CHECK(validate_server_config({ "127.0.0.1", kDefaultHttpApiPort, "" })
		      .allowed);
	CHECK_FALSE(
		validate_server_config({ "localhost", kDefaultHttpApiPort, "" })
			.allowed);
	CHECK(validate_server_config(
		      { "localhost", kDefaultHttpApiPort, kToken })
		      .allowed);

	const auto exposed_without_token =
		validate_server_config({ "0.0.0.0", kDefaultHttpApiPort, "" });
	REQUIRE_FALSE(exposed_without_token.allowed);
	CHECK(exposed_without_token.error == ServerConfigError::TokenRequired);

	CHECK(validate_server_config({ "0.0.0.0", kDefaultHttpApiPort, kToken })
		      .allowed);
	CHECK_FALSE(validate_server_config(
			    { "0.0.0.0", kDefaultHttpApiPort, "short" })
			    .allowed);
	CHECK_FALSE(validate_server_config({ "0.0.0.0", kDefaultHttpApiPort,
					     "0123456789abcde " })
			    .allowed);
	CHECK_FALSE(validate_server_config({ "127.0.0.1", -1, "" }).allowed);
	CHECK_FALSE(validate_server_config({ "127.0.0.1", 65536, "" }).allowed);
}

TEST_CASE("HTTP API JSON content type contract accepts JSON only")
{
	CHECK(is_json_content_type("application/json"));
	CHECK(is_json_content_type("application/json; charset=utf-8"));
	CHECK_FALSE(is_json_content_type(""));
	CHECK_FALSE(is_json_content_type("text/plain"));
	CHECK_FALSE(is_json_content_type("application/x-www-form-urlencoded"));
}

TEST_CASE("bearer auth validates token only when configured")
{
	CHECK(is_authorized_bearer("", ""));
	CHECK(is_authorized_bearer("Ignored", ""));
	CHECK(is_authorized_bearer(std::string("Bearer ") + kToken, kToken));
	CHECK_FALSE(is_authorized_bearer("", kToken));
	CHECK_FALSE(
		is_authorized_bearer(std::string("Basic ") + kToken, kToken));
	CHECK_FALSE(is_authorized_bearer("Bearer wrong", kToken));
}
