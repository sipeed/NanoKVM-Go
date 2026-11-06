#include <catch2/catch_test_macros.hpp>

#include <cstddef>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/about_page.hpp"
#include "ui/page.hpp"

namespace
{

constexpr uint32_t CLICK_GAP_MS = 100;
constexpr uint32_t SECRET_WINDOW_MS = ui::AboutPage::SECRET_CLICK_WINDOW_MS;

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *version_group(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 1);
}

lv_obj_t *app_ver_object(const ui::Page &page)
{
	lv_obj_t *group = version_group(page);
	if (group == nullptr || lv_obj_get_child_count(group) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(group, 1);
}

class SecretClickFixture {
    public:
	SecretClickFixture()
	{
		_page.on_app_ver_secret_click = [this]() {
			++_secret_count;
		};
		_page.create(_lvgl.screen());
	}

	~SecretClickFixture()
	{
		if (_page.is_created()) {
			_page.destroy();
		}
	}

	SecretClickFixture(const SecretClickFixture &) = delete;
	SecretClickFixture &
	operator=(const SecretClickFixture &) = delete;

	test_support::LvglTestContext &lvgl()
	{
		return _lvgl;
	}

	ui::AboutPage &page()
	{
		return _page;
	}

	int secret_count() const
	{
		return _secret_count;
	}

	void click()
	{
		lv_obj_t *obj = app_ver_object(_page);
		REQUIRE(obj != nullptr);
		lv_obj_send_event(obj, LV_EVENT_CLICKED, nullptr);
	}

    private:
	test_support::LvglTestContext _lvgl;
	ui::AboutPage _page;
	int _secret_count = 0;
};

TEST_CASE("APP Ver SConA is clickable for secret entry",
	  "[about][secret]")
{
	test_support::LvglTestContext lvgl;
	ui::AboutPage page;
	page.create(lvgl.screen());

	lv_obj_t *obj = app_ver_object(page);
	REQUIRE(obj != nullptr);
	REQUIRE(lv_obj_has_flag(obj, LV_OBJ_FLAG_CLICKABLE));

	page.destroy();
}

TEST_CASE("Five consecutive APP Ver clicks enter More", "[about][secret]")
{
	SecretClickFixture f;

	for (int i = 0; i < 4; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
		REQUIRE(f.secret_count() == 0);
	}
	f.click();
	REQUIRE(f.secret_count() == 1);

	// Counter resets after firing; another five-trigger sequence fires again.
	for (int i = 0; i < 4; ++i) {
		f.lvgl().advance(CLICK_GAP_MS);
		f.click();
		REQUIRE(f.secret_count() == 1);
	}
	f.lvgl().advance(CLICK_GAP_MS);
	f.click();
	REQUIRE(f.secret_count() == 2);
}

TEST_CASE("Click gap beyond the window resets the counter",
	  "[about][secret]")
{
	SecretClickFixture f;

	for (int i = 0; i < 3; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
	}
	REQUIRE(f.secret_count() == 0);

	// A gap larger than the window must reset the running count.
	f.lvgl().advance(SECRET_WINDOW_MS + 1000);
	for (int i = 0; i < 4; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
	}
	REQUIRE(f.secret_count() == 0);

	// The fifth click of the new sequence fires.
	f.click();
	REQUIRE(f.secret_count() == 1);
}

TEST_CASE("on_pause clears the click counter", "[about][secret]")
{
	SecretClickFixture f;

	for (int i = 0; i < 4; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
	}
	REQUIRE(f.secret_count() == 0);

	f.page().on_pause();

	// Four clicks after pause must not fire (counter was cleared to 0).
	for (int i = 0; i < 4; ++i) {
		f.lvgl().advance(CLICK_GAP_MS);
		f.click();
	}
	REQUIRE(f.secret_count() == 0);

	// Fifth click of the post-pause sequence fires.
	f.lvgl().advance(CLICK_GAP_MS);
	f.click();
	REQUIRE(f.secret_count() == 1);
}

TEST_CASE("Factory reset pending blocks secret entry", "[about][secret]")
{
	SecretClickFixture f;

	f.page().set_factory_reset_pending(true);
	for (int i = 0; i < 6; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
	}
	REQUIRE(f.secret_count() == 0);

	f.page().set_factory_reset_pending(false);
	for (int i = 0; i < 4; ++i) {
		f.click();
		f.lvgl().advance(CLICK_GAP_MS);
	}
	REQUIRE(f.secret_count() == 0);

	f.click();
	REQUIRE(f.secret_count() == 1);
}

} // namespace
