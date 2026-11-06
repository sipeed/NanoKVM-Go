#include <catch2/catch_test_macros.hpp>

#include <string_view>

#include <lvgl.h>

#include "ui/wifi_connect_method_page.hpp"

namespace
{

class LvglContext {
    public:
	LvglContext()
	{
		lv_init();
		_display = lv_display_create(270, 240);
	}

	~LvglContext()
	{
		if (_display != nullptr) {
			lv_display_delete(_display);
		}
		lv_deinit();
	}

	LvglContext(const LvglContext &) = delete;
	LvglContext &operator=(const LvglContext &) = delete;

	lv_display_t *display() const
	{
		return _display;
	}

    private:
	lv_display_t *_display = nullptr;
};

lv_obj_t *find_visible_direct_label(lv_obj_t *parent)
{
	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *child = lv_obj_get_child(parent, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    !lv_obj_has_flag(child, LV_OBJ_FLAG_HIDDEN)) {
			return child;
		}
	}

	return nullptr;
}

} // namespace

TEST_CASE("Wi-Fi setup methods prioritize password and route callbacks")
{
	LvglContext lvgl;
	REQUIRE(lvgl.display() != nullptr);
	REQUIRE(lv_screen_active() != nullptr);

	ui::WifiConnectMethodPage page;
	page.create(lv_screen_active());

	lv_obj_t *container = page.get_container();
	REQUIRE(container != nullptr);
	lv_obj_t *scroll_content = lv_obj_get_child(container, 0);
	REQUIRE(scroll_content != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll_content) == 3);

	lv_obj_t *back_button = lv_obj_get_child(scroll_content, 0);
	lv_obj_t *button_1 = lv_obj_get_child(scroll_content, 1);
	lv_obj_t *button_2 = lv_obj_get_child(scroll_content, 2);
	REQUIRE(back_button != nullptr);
	REQUIRE(button_1 != nullptr);
	REQUIRE(button_2 != nullptr);

	lv_obj_t *button_1_label = find_visible_direct_label(button_1);
	lv_obj_t *button_2_label = find_visible_direct_label(button_2);
	REQUIRE(button_1_label != nullptr);
	REQUIRE(button_2_label != nullptr);

	const std::string_view button_1_text =
		lv_label_get_text(button_1_label);
	const std::string_view button_2_text =
		lv_label_get_text(button_2_label);
	CHECK(button_1_text == "PASSWD");
	CHECK(button_2_text == "QRCODE");
	CHECK(button_1_text.size() <= 6);
	CHECK(button_2_text.size() <= 6);

	int key_clicks = 0;
	int qr_clicks = 0;
	page.on_key_click = [&key_clicks]() {
		++key_clicks;
	};
	page.on_qr_click = [&qr_clicks]() {
		++qr_clicks;
	};

	lv_obj_send_event(button_1, LV_EVENT_CLICKED, nullptr);
	CHECK(key_clicks == 1);
	CHECK(qr_clicks == 0);

	key_clicks = 0;
	qr_clicks = 0;
	lv_obj_send_event(button_2, LV_EVENT_CLICKED, nullptr);
	CHECK(key_clicks == 0);
	CHECK(qr_clicks == 1);

	page.destroy();
	CHECK_FALSE(page.is_created());
}
