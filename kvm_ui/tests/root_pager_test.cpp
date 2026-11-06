#include <array>
#include <cstddef>
#include <string_view>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_test_context.hpp"
#include "ui/root_layout.hpp"
#include "ui/root_pager.hpp"

namespace
{

constexpr uint32_t kTransitionSettleMs = 400;

using ContentPages = std::array<lv_obj_t *, ui::root_layout::kContentPageCount>;

ContentPages create_content_pages(ui::RootPager &pager)
{
	ContentPages pages {};
	for (std::size_t index = 0; index < pages.size(); ++index) {
		pages[index] = lv_obj_create(pager.content_parent());
		lv_obj_set_size(pages[index],
				ui::root_layout::kContentCardWidth,
				ui::root_layout::kContentCardHeight);
		lv_obj_set_pos(pages[index],
			       static_cast<int32_t>(index) *
				       ui::root_layout::kContentPageStep,
			       0);
		lv_obj_clear_flag(pages[index], LV_OBJ_FLAG_SCROLLABLE);
	}
	return pages;
}

void install_page_provider(ui::RootPager &pager, const ContentPages &pages)
{
	pager.set_content_page_provider(
		[&pages](ui::RootPager::ContentPage page) -> lv_obj_t * {
			switch (page) {
			case ui::RootPager::ContentPage::Main:
				return pages[0];
			case ui::RootPager::ContentPage::SettingsHome:
				return pages[1];
			case ui::RootPager::ContentPage::SettingsSecond:
				return pages[2];
			case ui::RootPager::ContentPage::Count:
				return nullptr;
			}
			return nullptr;
		});
}

lv_obj_t *direct_child_with_size(lv_obj_t *parent, int32_t width,
				 int32_t height)
{
	if (parent == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(parent);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(parent, index);
		if (lv_obj_get_width(child) == width &&
		    lv_obj_get_height(child) == height) {
			return child;
		}
	}

	return nullptr;
}

lv_obj_t *settings_title_label(lv_obj_t *status_bar)
{
	if (status_bar == nullptr) {
		return nullptr;
	}

	const uint32_t child_count = lv_obj_get_child_count(status_bar);
	for (uint32_t index = 0; index < child_count; ++index) {
		lv_obj_t *const child = lv_obj_get_child(status_bar, index);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    std::string_view(lv_label_get_text(child)) == "Settings") {
			return child;
		}
	}

	return nullptr;
}

std::array<lv_obj_t *, ui::kRootStatusIconCount>
status_icon_objects(lv_obj_t *status_bar)
{
	std::array<lv_obj_t *, ui::kRootStatusIconCount> icons {};
	if (status_bar == nullptr) {
		return icons;
	}

	std::size_t icon_index = 0;
	const uint32_t child_count = lv_obj_get_child_count(status_bar);
	for (uint32_t index = 0;
	     index < child_count && icon_index < icons.size(); ++index) {
		lv_obj_t *const child = lv_obj_get_child(status_bar, index);
		if (lv_obj_check_type(child, &lv_image_class)) {
			icons[icon_index++] = child;
		}
	}

	return icons;
}

std::size_t active_indicator_count(lv_obj_t *indicator)
{
	if (indicator == nullptr) {
		return 0;
	}

	std::size_t count = 0;
	const uint32_t child_count = lv_obj_get_child_count(indicator);
	for (uint32_t index = 0; index < child_count; ++index) {
		if (lv_obj_get_width(lv_obj_get_child(indicator, index)) == 8) {
			++count;
		}
	}
	return count;
}

std::size_t active_indicator_index(lv_obj_t *indicator)
{
	if (indicator == nullptr) {
		return ui::root_layout::kContentPageCount;
	}

	const uint32_t child_count = lv_obj_get_child_count(indicator);
	for (uint32_t index = 0; index < child_count; ++index) {
		if (lv_obj_get_width(lv_obj_get_child(indicator, index)) == 8) {
			return index;
		}
	}
	return ui::root_layout::kContentPageCount;
}

} // namespace

TEST_CASE("RootPager preserves slide navigation chrome and subpage hosting",
	  "[ui][root-pager]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages pages = create_content_pages(pager);
	install_page_provider(pager, pages);
	lv_obj_update_layout(lvgl.screen());
	lv_obj_t *const root = lv_obj_get_parent(pager.content_parent());
	REQUIRE(root != nullptr);
	lv_obj_t *const status_bar = direct_child_with_size(root, 244, 40);
	lv_obj_t *const indicator = direct_child_with_size(root, 48, 8);
	REQUIRE(status_bar != nullptr);
	REQUIRE(indicator != nullptr);
	lv_obj_t *const settings_title = settings_title_label(status_bar);
	REQUIRE(settings_title != nullptr);
	const auto status_icons = status_icon_objects(status_bar);
	for (lv_obj_t *icon : status_icons) {
		REQUIRE(icon != nullptr);
	}

	REQUIRE(pager.is_created());
	CHECK(pager.is_main_visible());
	CHECK_FALSE(pager.is_settings_visible());
	CHECK_FALSE(lv_obj_has_flag(status_icons[0], LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(status_icons[1], LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(status_icons[2], LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));
	CHECK(active_indicator_count(indicator) == 1);
	CHECK(active_indicator_index(indicator) == 0);

	pager.set_status_icon_active(ui::RootPager::StatusIcon::Video, true);
	pager.set_status_icon_active(ui::RootPager::StatusIcon::KeyboardMouse,
				     true);
	pager.set_status_icon_active(ui::RootPager::StatusIcon::Count, true);
	lv_obj_update_layout(lvgl.screen());
	CHECK_FALSE(lv_obj_has_flag(status_icons[1], LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(status_icons[2], LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_x(status_icons[0]) == 0);
	CHECK(lv_obj_get_x(status_icons[1]) > lv_obj_get_x(status_icons[0]));
	CHECK(lv_obj_get_x(status_icons[2]) > lv_obj_get_x(status_icons[1]));
	pager.set_status_icon_active(ui::RootPager::StatusIcon::Wifi, false);
	lv_obj_update_layout(lvgl.screen());
	CHECK(lv_obj_has_flag(status_icons[0], LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_get_x(status_icons[1]) == 0);
	CHECK(lv_obj_get_x(status_icons[2]) > lv_obj_get_x(status_icons[1]));

	pager.transition_to(ui::RootPager::ContentPage::SettingsHome);
	lvgl.advance(kTransitionSettleMs);
	CHECK_FALSE(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));
	for (lv_obj_t *icon : status_icons) {
		CHECK(lv_obj_has_flag(icon, LV_OBJ_FLAG_HIDDEN));
	}
	CHECK(active_indicator_count(indicator) == 1);
	CHECK(active_indicator_index(indicator) == 1);

	pager.transition_to(ui::RootPager::ContentPage::SettingsSecond);
	lvgl.advance(kTransitionSettleMs);
	CHECK(pager.current_content_page() ==
	      ui::RootPager::ContentPage::SettingsSecond);
	CHECK_FALSE(pager.is_main_visible());
	CHECK(pager.is_settings_visible());
	CHECK(active_indicator_count(indicator) == 1);
	CHECK(active_indicator_index(indicator) == 2);

	pager.enter_settings_subpage();
	pager.refresh_navigation_chrome();
	CHECK(pager.is_subpage_visible());
	CHECK(lv_obj_has_flag(pager.settings_host(), LV_OBJ_FLAG_HIDDEN) ==
	      false);
	CHECK(lv_obj_has_flag(indicator, LV_OBJ_FLAG_HIDDEN));
	pager.leave_settings_subpage();
	pager.refresh_navigation_chrome();
	CHECK_FALSE(pager.is_subpage_visible());
	CHECK(lv_obj_has_flag(pager.settings_host(), LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(indicator, LV_OBJ_FLAG_HIDDEN));

	pager.transition_to(ui::RootPager::ContentPage::Main);
	lvgl.advance(kTransitionSettleMs);
	CHECK(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(status_icons[1], LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(status_icons[2], LV_OBJ_FLAG_HIDDEN));
}

TEST_CASE("RootPager keeps slide chrome fade and shutdown ownership stable",
	  "[ui][root-pager][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.create(lvgl.screen());
	const ContentPages pages = create_content_pages(pager);
	install_page_provider(pager, pages);
	lv_obj_update_layout(lvgl.screen());

	REQUIRE(lvgl.timer_count() == timer_count_before + 1U);
	lv_obj_t *const root = lv_obj_get_parent(pager.content_parent());
	REQUIRE(root != nullptr);
	lv_obj_t *const status_bar = direct_child_with_size(root, 244, 40);
	REQUIRE(status_bar != nullptr);
	lv_obj_t *const settings_title = settings_title_label(status_bar);
	lv_obj_t *const fade_overlay =
		direct_child_with_size(status_bar, 160, 40);
	REQUIRE(settings_title != nullptr);
	REQUIRE(fade_overlay != nullptr);
	lv_obj_t *const scroll_container = pager.content_parent();
	REQUIRE(scroll_container != nullptr);

	CHECK(lv_obj_has_flag(fade_overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));

	lv_obj_scroll_to_x(scroll_container, ui::root_layout::kContentPageStep,
			   LV_ANIM_ON);
	lvgl.advance(40);
	CHECK(lv_obj_get_scroll_x(scroll_container) > 0);
	CHECK_FALSE(lv_obj_has_flag(fade_overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));

	lvgl.advance(120);
	CHECK_FALSE(lv_obj_has_flag(fade_overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));

	lvgl.advance(kTransitionSettleMs);
	CHECK(lv_obj_has_flag(fade_overlay, LV_OBJ_FLAG_HIDDEN));
	CHECK_FALSE(lv_obj_has_flag(settings_title, LV_OBJ_FLAG_HIDDEN));
	CHECK(pager.current_content_page() ==
	      ui::RootPager::ContentPage::SettingsHome);

	pager.begin_shutdown();
	CHECK(lvgl.timer_count() == timer_count_before);
	pager.shutdown();
	pager.shutdown();
	CHECK_FALSE(pager.is_created());
	CHECK(lvgl.timer_count() == timer_count_before);
}

TEST_CASE("RootPager applies stored status icon state when chrome is created",
	  "[ui][root-pager][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	ui::RootPager pager(ui::RootPager::TransitionMode::Slide);
	pager.set_status_icon_active(ui::RootPager::StatusIcon::Video, true);

	pager.create(lvgl.screen());
	lv_obj_update_layout(lvgl.screen());
	lv_obj_t *const root = lv_obj_get_parent(pager.content_parent());
	REQUIRE(root != nullptr);
	lv_obj_t *const status_bar = direct_child_with_size(root, 244, 40);
	REQUIRE(status_bar != nullptr);
	const auto status_icons = status_icon_objects(status_bar);
	REQUIRE(status_icons[1] != nullptr);
	CHECK_FALSE(lv_obj_has_flag(status_icons[1], LV_OBJ_FLAG_HIDDEN));
	pager.set_status_icon_active(ui::RootPager::StatusIcon::Count, false);
	CHECK_FALSE(lv_obj_has_flag(status_icons[1], LV_OBJ_FLAG_HIDDEN));

	pager.begin_shutdown();
	pager.shutdown();
	pager.shutdown();
}

TEST_CASE("RootPager completes flip transitions and releases its timer",
	  "[ui][root-pager][lifecycle]")
{
	test_support::LvglTestContext lvgl;
	const std::size_t timer_count_before = lvgl.timer_count();

	{
		ui::RootPager pager(ui::RootPager::TransitionMode::Flip);
		pager.create(lvgl.screen());
		const ContentPages pages = create_content_pages(pager);
		install_page_provider(pager, pages);

		CHECK(lvgl.timer_count() == timer_count_before + 1U);
		pager.transition_to(ui::RootPager::ContentPage::SettingsHome);
		CHECK(pager.is_flip_animating());
		lvgl.advance(kTransitionSettleMs);
		CHECK_FALSE(pager.is_flip_animating());
		CHECK(pager.current_content_page() ==
		      ui::RootPager::ContentPage::SettingsHome);
	}

	CHECK(lvgl.timer_count() == timer_count_before);
}
