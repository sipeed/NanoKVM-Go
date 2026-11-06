#include <catch2/catch_test_macros.hpp>

#include <cstdint>
#include <string>
#include <string_view>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/bconb.hpp"
#include "ui/hold_confirm_behavior.hpp"
#include "ui/network_page.hpp"
#include "ui/scona.hpp"

extern "C" {
LV_IMAGE_DECLARE(network_terms_check);
LV_IMAGE_DECLARE(network_terms_cross);
}

namespace
{

constexpr uint32_t ON_HOLD_DURATION_MS = 2000;
constexpr uint32_t AGREE_COLOR = 0x016630;
constexpr uint32_t DECLINE_COLOR = 0xCA3500;
constexpr uint32_t TERMS_TEXT_COLOR = 0x8B949E;
constexpr char QR_URL[] = "https://wiki.sipeed.com/nkvm-legal.html";
// 39 bytes: version-3-M byte capacity starts at 27 bytes and ends at 42, so
// the fixed 125px canvas renders 29 modules at integer scale 4 (116px).
static_assert(sizeof(QR_URL) - 1 == 39);
constexpr int32_t TERMS_ROW_TOP = 12;
constexpr int32_t TERMS_BODY_TOP = 88;
constexpr int32_t TERMS_BODY_HEIGHT = 152;
constexpr int32_t QR_CARD_LEFT = 0;
// Fixed on-device measured values: the card centers the 125px QR canvas at
// x = 8 (LVGL pw/2 - w/2), and the canvas' internal margin (4 left / 5
// right) leaves exactly 12px of white per side around the dark modules
// (3.0 modules at scale 4). The card spans canvas rows 88..227 with a 12px
// clearance to the canvas bottom (rows 228..239).
constexpr int32_t QR_CARD_SIZE = 140;
constexpr int32_t QR_CARD_RADIUS = 10;
constexpr int32_t QR_CODE_SIZE = 125;
// LVGL centers children with pw / 2 - w / 2 integer math (lv_obj_refr_pos):
// 140 / 2 - 125 / 2 = 70 - 62 = 8.
constexpr int32_t QR_CODE_OFFSET = QR_CARD_SIZE / 2 - QR_CODE_SIZE / 2;
constexpr int32_t TEXT_BLOCK_LEFT = 160;
constexpr int32_t TEXT_BLOCK_WIDTH = 107;
constexpr int32_t TEXT_LINE_SPACE = 4;

// Geometric radius-clamp invariants, self-contained: the radius must stay
// within size/2 and within floor(margin * (2 + sqrt(2))) with
// margin = (QR_CARD_SIZE - QR_CODE_SIZE) / 2, so the QR's dark corner
// modules stay inside the rounded white corner.
static_assert(QR_CARD_RADIUS <= QR_CARD_SIZE / 2);
static_assert(QR_CARD_RADIUS <=
	      static_cast<int32_t>((QR_CARD_SIZE - QR_CODE_SIZE) / 2.0 *
				   (2.0 + 1.4142135623730951)));

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *scroll_child(const ui::Page &page, uint32_t index)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, index);
}

lv_obj_t *back_button(const ui::Page &page)
{
	return scroll_child(page, 0);
}

lv_obj_t *preinit_group(const ui::Page &page)
{
	return scroll_child(page, 1);
}

lv_obj_t *on_button(const ui::Page &page)
{
	return scroll_child(page, 2);
}

lv_obj_t *terms_row(const ui::Page &page)
{
	return scroll_child(page, 3);
}

lv_obj_t *terms_body(const ui::Page &page)
{
	return scroll_child(page, 4);
}

lv_obj_t *code_stage(const ui::Page &page)
{
	return scroll_child(page, 5);
}

lv_obj_t *row_button(const ui::Page &page, uint32_t index)
{
	lv_obj_t *row = terms_row(page);
	if (row == nullptr || lv_obj_get_child_count(row) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(row, index);
}

lv_obj_t *agree_button(const ui::Page &page)
{
	return row_button(page, 0);
}

lv_obj_t *decline_button(const ui::Page &page)
{
	return row_button(page, 1);
}

lv_obj_t *body_child(const ui::Page &page, uint32_t index)
{
	lv_obj_t *body = terms_body(page);
	if (body == nullptr || lv_obj_get_child_count(body) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(body, index);
}

lv_obj_t *qr_card(const ui::Page &page)
{
	return body_child(page, 0);
}

lv_obj_t *privacy_block(const ui::Page &page)
{
	return body_child(page, 1);
}

lv_obj_t *privacy_label(const ui::Page &page, uint32_t index)
{
	lv_obj_t *block = privacy_block(page);
	if (block == nullptr || lv_obj_get_child_count(block) <= index) {
		return nullptr;
	}
	return lv_obj_get_child(block, index);
}

lv_obj_t *find_descendant_of_type(lv_obj_t *parent,
				  const lv_obj_class_t *type)
{
	if (parent == nullptr || type == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, type)) {
			return child;
		}
		if (lv_obj_t *match = find_descendant_of_type(child, type);
		    match != nullptr) {
			return match;
		}
	}

	return nullptr;
}

lv_obj_t *button_image(lv_obj_t *button)
{
	return find_descendant_of_type(button, &lv_image_class);
}

lv_obj_t *qr_code(const ui::Page &page)
{
	return find_descendant_of_type(qr_card(page), &lv_qrcode_class);
}

// lv_qrcode keeps the two-entry I1 palette in the first 8 bytes of its draw
// buffer (lv_qrcode_update writes pixels at data + 8) and stores 1 = dark
// module. A canvas that was never encoded — or whose encode failed — is
// cleared to zero, so counting non-zero payload bytes proves that the widget
// really holds a rendered code rather than an empty white square.
uint32_t qr_dark_byte_count(lv_obj_t *code)
{
	lv_draw_buf_t *const draw_buf = lv_canvas_get_draw_buf(code);
	if (draw_buf == nullptr || draw_buf->data == nullptr ||
	    draw_buf->data_size <= 8) {
		return 0;
	}
	const auto *const bytes =
		static_cast<const uint8_t *>(draw_buf->data) + 8;
	// Guard the assumed I1 layout (palette entries at the buffer head, pixel
	// rows after them) so a future LVGL layout change fails the assertion
	// instead of silently counting palette bytes as modules.
	if (draw_buf->data_size != draw_buf->header.stride * draw_buf->header.h +
					  8) {
		return 0;
	}
	uint32_t dark = 0;
	for (uint32_t index = 0; index + 8 < draw_buf->data_size; ++index) {
		if (bytes[index] != 0) {
			++dark;
		}
	}
	return dark;
}

lv_area_t object_coords(lv_obj_t *object)
{
	lv_area_t coordinates {};
	if (object != nullptr) {
		lv_obj_get_coords(object, &coordinates);
	}
	return coordinates;
}

class NetworkTermsFixture {
    public:
	NetworkTermsFixture()
	{
		_page.create(_lvgl.screen());
		refresh_layout();
	}

	~NetworkTermsFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	NetworkTermsFixture(const NetworkTermsFixture &) = delete;
	NetworkTermsFixture &operator=(const NetworkTermsFixture &) = delete;

	ui::NetworkPage &page()
	{
		return _page;
	}

	void refresh_layout()
	{
		lv_obj_update_layout(_lvgl.screen());
	}

	void complete_hold()
	{
		constexpr uint32_t fill_ms =
			ON_HOLD_DURATION_MS -
			ui::HoldConfirmBehavior::FULL_DWELL_MS;
		_lvgl.press(on_button(_page));
		_lvgl.advance(fill_ms);
		_lvgl.advance(ui::HoldConfirmBehavior::FULL_DWELL_MS);
		_lvgl.release();
	}

	void enter_terms()
	{
		complete_hold();
		refresh_layout();
	}

	void destroy_page()
	{
		_page.destroy();
	}

	void recreate_page()
	{
		_page.create(_lvgl.screen());
		refresh_layout();
	}

    private:
	test_support::LvglTestContext _lvgl;
	ui::NetworkPage _page;
};

void require_terms_stage(const ui::Page &page)
{
	REQUIRE(back_button(page) != nullptr);
	REQUIRE(preinit_group(page) != nullptr);
	REQUIRE(on_button(page) != nullptr);
	REQUIRE(terms_row(page) != nullptr);
	REQUIRE(terms_body(page) != nullptr);
	REQUIRE(code_stage(page) != nullptr);
	CHECK(lv_obj_has_flag(back_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(preinit_group(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(on_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(terms_row(page), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(terms_body(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(code_stage(page), LV_OBJ_FLAG_HIDDEN));
}

void require_preinit_stage(const ui::Page &page)
{
	REQUIRE(back_button(page) != nullptr);
	REQUIRE(preinit_group(page) != nullptr);
	REQUIRE(on_button(page) != nullptr);
	REQUIRE(terms_row(page) != nullptr);
	REQUIRE(terms_body(page) != nullptr);
	REQUIRE(code_stage(page) != nullptr);
	CHECK_FALSE(lv_obj_has_flag(back_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(preinit_group(page), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(on_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_row(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_body(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(code_stage(page), LV_OBJ_FLAG_HIDDEN));
}

void require_code_stage(const ui::Page &page)
{
	REQUIRE(back_button(page) != nullptr);
	REQUIRE(preinit_group(page) != nullptr);
	REQUIRE(on_button(page) != nullptr);
	REQUIRE(terms_row(page) != nullptr);
	REQUIRE(terms_body(page) != nullptr);
	REQUIRE(code_stage(page) != nullptr);
	CHECK_FALSE(lv_obj_has_flag(back_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(preinit_group(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(on_button(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_row(page), LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(terms_body(page), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(code_stage(page), LV_OBJ_FLAG_HIDDEN));
}

} // namespace

TEST_CASE("Network terms stage renders the locked mockup geometry",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	lv_obj_t *row = terms_row(fixture.page());
	const lv_area_t row_coordinates = object_coords(row);
	CHECK(row_coordinates.x1 == 20);
	CHECK(row_coordinates.y1 == TERMS_ROW_TOP);
	CHECK(lv_obj_get_width(row) == ui::SConA::WIDTH);
	CHECK(lv_obj_get_height(row) == ui::BConB::HEIGHT);

	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	CHECK(lv_obj_get_width(agree) == ui::BConB::WIDTH);
	CHECK(lv_obj_get_height(agree) == ui::BConB::HEIGHT);
	CHECK(lv_obj_get_style_radius(agree, LV_PART_MAIN) ==
	      ui::BConB::RADIUS);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(agree, LV_PART_MAIN),
			  lv_color_hex(AGREE_COLOR)));
	const lv_area_t agree_coordinates = object_coords(agree);
	CHECK(agree_coordinates.x1 == 20);
	CHECK(agree_coordinates.y1 == TERMS_ROW_TOP);
	lv_obj_t *agree_image = button_image(agree);
	REQUIRE(agree_image != nullptr);
	CHECK(lv_image_get_src(agree_image) == &network_terms_check);
	CHECK(network_terms_check.header.w == 36);
	CHECK(network_terms_check.header.h == 36);
	CHECK(network_terms_check.header.cf == LV_COLOR_FORMAT_ARGB8888);

	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	CHECK(lv_obj_get_width(decline) == ui::BConB::WIDTH);
	CHECK(lv_obj_get_height(decline) == ui::BConB::HEIGHT);
	CHECK(lv_obj_get_style_radius(decline, LV_PART_MAIN) ==
	      ui::BConB::RADIUS);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(decline, LV_PART_MAIN),
			  lv_color_hex(DECLINE_COLOR)));
	const lv_area_t decline_coordinates = object_coords(decline);
	CHECK(decline_coordinates.x1 == 150);
	CHECK(decline_coordinates.y1 == TERMS_ROW_TOP);
	lv_obj_t *decline_image = button_image(decline);
	REQUIRE(decline_image != nullptr);
	CHECK(lv_image_get_src(decline_image) == &network_terms_cross);
	CHECK(network_terms_cross.header.w == 36);
	CHECK(network_terms_cross.header.h == 36);
	CHECK(network_terms_cross.header.cf == LV_COLOR_FORMAT_ARGB8888);

	lv_obj_t *body = terms_body(fixture.page());
	const lv_area_t body_coordinates = object_coords(body);
	CHECK(body_coordinates.x1 == 0);
	CHECK(body_coordinates.y1 == TERMS_BODY_TOP);
	CHECK(lv_obj_get_width(body) == 270);
	CHECK(lv_obj_get_height(body) == TERMS_BODY_HEIGHT);

	lv_obj_t *card = qr_card(fixture.page());
	REQUIRE(card != nullptr);
	const lv_area_t card_coordinates = object_coords(card);
	CHECK(card_coordinates.x1 == QR_CARD_LEFT);
	CHECK(card_coordinates.y1 == TERMS_BODY_TOP);
	CHECK(lv_obj_get_width(card) == QR_CARD_SIZE);
	CHECK(lv_obj_get_height(card) == QR_CARD_SIZE);
	// The card (rows 88..227) keeps a 12px clearance to the canvas bottom
	// (rows 228..239), which keeps it clear of the physical rounded
	// screen corner.
	CHECK(card_coordinates.y2 == TERMS_BODY_TOP + QR_CARD_SIZE - 1);
	CHECK(card_coordinates.y2 == 227);
	CHECK(lv_obj_get_style_radius(card, LV_PART_MAIN) == QR_CARD_RADIUS);
	CHECK(lv_color_eq(lv_obj_get_style_bg_color(card, LV_PART_MAIN),
			  lv_color_white()));

	lv_obj_t *code = qr_code(fixture.page());
	REQUIRE(code != nullptr);
	CHECK(lv_obj_get_width(code) == QR_CODE_SIZE);
	CHECK(lv_obj_get_height(code) == QR_CODE_SIZE);
	const lv_area_t code_coordinates = object_coords(code);
	CHECK(code_coordinates.x1 == QR_CARD_LEFT + QR_CODE_OFFSET);
	CHECK(code_coordinates.x1 == 8);
	CHECK(code_coordinates.y1 == TERMS_BODY_TOP + QR_CODE_OFFSET);
	CHECK(code_coordinates.y1 == 96);

	lv_obj_t *block = privacy_block(fixture.page());
	REQUIRE(block != nullptr);
	const lv_area_t block_coordinates = object_coords(block);
	CHECK(block_coordinates.x1 == TEXT_BLOCK_LEFT);
	CHECK(block_coordinates.y1 == TERMS_BODY_TOP);
	CHECK(lv_obj_get_width(block) == TEXT_BLOCK_WIDTH);
	CHECK(lv_obj_get_height(block) == TERMS_BODY_HEIGHT);
	CHECK(lv_obj_get_child_count(block) == 2);

	lv_obj_t *left = privacy_label(fixture.page(), 0);
	REQUIRE(left != nullptr);
	CHECK(std::string_view(lv_label_get_text(left)) == "PRIVACY POLICY");
	CHECK(lv_obj_get_width(left) == TEXT_BLOCK_WIDTH);
	// Two wrapped lines at line-height 18 plus line-space 4: the phrase
	// must not collapse to one line or overflow to three.
	CHECK(lv_obj_get_height(left) == 40);
	CHECK(lv_obj_get_style_text_font(left, LV_PART_MAIN) ==
	      &lv_font_montserrat_16);
	CHECK(lv_color_eq(lv_obj_get_style_text_color(left, LV_PART_MAIN),
			  lv_color_hex(TERMS_TEXT_COLOR)));
	CHECK(lv_obj_get_style_text_letter_space(left, LV_PART_MAIN) == 1);
	CHECK(lv_obj_get_style_text_line_space(left, LV_PART_MAIN) ==
	      TEXT_LINE_SPACE);
	CHECK(lv_label_get_long_mode(left) == LV_LABEL_LONG_WRAP);
	CHECK(lv_obj_get_style_text_align(left, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
	const lv_area_t left_coordinates = object_coords(left);

	lv_obj_t *right = privacy_label(fixture.page(), 1);
	REQUIRE(right != nullptr);
	CHECK(std::string_view(lv_label_get_text(right)) ==
	      "TERMS OF SERVICE");
	CHECK(lv_obj_get_width(right) == TEXT_BLOCK_WIDTH);
	CHECK(lv_obj_get_height(right) == 40);
	CHECK(lv_obj_get_style_text_font(right, LV_PART_MAIN) ==
	      &lv_font_montserrat_16);
	CHECK(lv_color_eq(lv_obj_get_style_text_color(right, LV_PART_MAIN),
			  lv_color_hex(TERMS_TEXT_COLOR)));
	CHECK(lv_obj_get_style_text_letter_space(right, LV_PART_MAIN) == 1);
	CHECK(lv_obj_get_style_text_line_space(right, LV_PART_MAIN) ==
	      TEXT_LINE_SPACE);
	CHECK(lv_label_get_long_mode(right) == LV_LABEL_LONG_WRAP);
	CHECK(lv_obj_get_style_text_align(right, LV_PART_MAIN) ==
	      LV_TEXT_ALIGN_CENTER);
	const lv_area_t right_coordinates = object_coords(right);

	// The two-line labels form one vertically centered group next to the
	// QR card; the group midpoint keeps the strips' old y164 center.
	CHECK((left_coordinates.y1 + right_coordinates.y2) / 2 >= 163);
	CHECK((left_coordinates.y1 + right_coordinates.y2) / 2 <= 165);
}

TEST_CASE("Network terms decline returns to pre-init without an API call",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;
	uint32_t start_calls = 0;
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	lv_obj_t *decline = decline_button(fixture.page());
	REQUIRE(decline != nullptr);
	lv_obj_send_event(decline, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();

	CHECK(start_calls == 0);
	require_preinit_stage(fixture.page());
}

TEST_CASE("Network terms agree follows the start-request lane",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	// Unwired lane: stay in TERMS silently.
	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	require_terms_stage(fixture.page());

	// Rejected start: stay in TERMS with no visual change (v1).
	uint32_t start_calls = 0;
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return false;
	};
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(start_calls == 1);
	require_terms_stage(fixture.page());

	// Confirmed start: the verification-code stage opens with back
	// navigation locked until the heartbeat confirms the session started.
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(start_calls == 2);
	require_code_stage(fixture.page());
	CHECK_FALSE(
		lv_obj_has_flag(back_button(fixture.page()), LV_OBJ_FLAG_CLICKABLE));

	ui::TailscaleSnapshotView started;
	started.has_snapshot = true;
	started.started = true;
	fixture.page().apply_snapshot(started, 1);
	fixture.refresh_layout();
	require_code_stage(fixture.page());
	CHECK(lv_obj_has_flag(back_button(fixture.page()), LV_OBJ_FLAG_CLICKABLE));
}

TEST_CASE("Network terms agree ignores re-entrant clicks while pending",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;
	fixture.enter_terms();
	require_terms_stage(fixture.page());

	uint32_t start_calls = 0;
	fixture.page().on_start_request = [&start_calls]() -> bool {
		++start_calls;
		return true;
	};
	lv_obj_t *agree = agree_button(fixture.page());
	REQUIRE(agree != nullptr);
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(start_calls == 1);
	require_code_stage(fixture.page());

	// A queued CLICKED reaching the hidden agree button must not
	// double-issue the start request while the first one pends.
	lv_obj_send_event(agree, LV_EVENT_CLICKED, nullptr);
	fixture.refresh_layout();
	CHECK(start_calls == 1);
	require_code_stage(fixture.page());
	CHECK_FALSE(
		lv_obj_has_flag(back_button(fixture.page()), LV_OBJ_FLAG_CLICKABLE));
}

TEST_CASE("Network terms QR encodes the fixed legal URL",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;

	// The payload is derived from nothing at runtime: it is the constant legal
	// page, so an unknown or changing Wi-Fi IP can never alter the code.
	CHECK(fixture.page().qr_payload() == QR_URL);
	REQUIRE(qr_code(fixture.page()) != nullptr);
	CHECK(qr_dark_byte_count(qr_code(fixture.page())) > 0);

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	CHECK(fixture.page().qr_payload() == QR_URL);

	fixture.page().on_exit();
	fixture.page().on_enter();
	require_preinit_stage(fixture.page());
	CHECK(fixture.page().qr_payload() == QR_URL);

	// A recreated page re-encodes the constant into the new widget instead of
	// relying on the previous widget's canvas.
	fixture.enter_terms();
	require_terms_stage(fixture.page());
	fixture.destroy_page();
	fixture.recreate_page();
	require_preinit_stage(fixture.page());
	CHECK(fixture.page().qr_payload() == QR_URL);
	REQUIRE(qr_code(fixture.page()) != nullptr);
	CHECK(lv_obj_get_width(qr_code(fixture.page())) == QR_CODE_SIZE);
	CHECK(lv_obj_get_height(qr_code(fixture.page())) == QR_CODE_SIZE);
	// The recreated canvas was really re-encoded, not just re-marked.
	CHECK(qr_dark_byte_count(qr_code(fixture.page())) > 0);
}

TEST_CASE("Network terms resets to pre-init on enter and recreate",
	  "[ui][network][terms]")
{
	NetworkTermsFixture fixture;

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	fixture.page().on_pause();
	fixture.page().on_enter();
	require_preinit_stage(fixture.page());

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	fixture.page().on_exit();
	fixture.page().on_enter();
	require_preinit_stage(fixture.page());

	fixture.enter_terms();
	require_terms_stage(fixture.page());
	fixture.destroy_page();
	fixture.recreate_page();
	require_preinit_stage(fixture.page());
	CHECK(fixture.page().qr_payload() == QR_URL);

	// The ON hold stays usable after every reset path.
	fixture.enter_terms();
	require_terms_stage(fixture.page());
}
