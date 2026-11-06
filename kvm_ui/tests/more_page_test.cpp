#include <catch2/catch_test_macros.hpp>

#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

#include <lvgl.h>

#include "lvgl_test_context.hpp"
#include "ui/more_page.hpp"
#include "ui/page.hpp"
#include "ui/scona_wide.hpp"

namespace
{

constexpr std::size_t kItemCount = 18;

// Order must match MorePage::InfoItem (see more_page.cpp kInfoLabels).
constexpr std::string_view kExpectedLabels[] = {
	"Hardware Version:", "Device Key:",   "Hostname:", "Device Number:",
	"IP:",                "mDNS:",         "MAC:",      "WiFi Signal:",
	"Image Version:",     "APP Version:",  "EDID:",
	"Resolution:",        "Uptime:",       "Time:",     "Memory:",
	"Storage:",           "Temperature:",  "UI Version:",
};

lv_obj_t *scroll_content(const ui::Page &page)
{
	lv_obj_t *container = page.get_container();
	if (container == nullptr || lv_obj_get_child_count(container) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(container, 0);
}

lv_obj_t *back_button(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) == 0) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 0);
}

lv_obj_t *content_group(const ui::Page &page)
{
	lv_obj_t *scroll = scroll_content(page);
	if (scroll == nullptr || lv_obj_get_child_count(scroll) < 2) {
		return nullptr;
	}
	return lv_obj_get_child(scroll, 1);
}

std::string_view back_title(const ui::Page &page)
{
	return test_support::visible_direct_text(back_button(page));
}

lv_obj_t *visible_label_at(lv_obj_t *parent, std::size_t index)
{
	if (parent == nullptr) {
		return nullptr;
	}
	std::size_t seen = 0;
	const uint32_t count = lv_obj_get_child_count(parent);
	for (uint32_t i = 0; i < count; ++i) {
		lv_obj_t *child = lv_obj_get_child(parent, i);
		if (lv_obj_check_type(child, &lv_label_class) &&
		    !lv_obj_has_flag(child, LV_OBJ_FLAG_HIDDEN)) {
			if (seen == index) {
				return child;
			}
			++seen;
		}
	}
	return nullptr;
}

std::string_view label_text(lv_obj_t *parent, std::size_t index)
{
	lv_obj_t *label = visible_label_at(parent, index);
	return label == nullptr ? std::string_view {} :
				 lv_label_get_text(label);
}

lv_obj_t *item_desc_obj(lv_obj_t *group, std::size_t i)
{
	if (group == nullptr) {
		return nullptr;
	}
	return lv_obj_get_child(group, static_cast<int32_t>(2 * i));
}

lv_obj_t *item_value_obj(lv_obj_t *group, std::size_t i)
{
	if (group == nullptr) {
		return nullptr;
	}
	return lv_obj_get_child(group, static_cast<int32_t>(2 * i + 1));
}

std::string_view item_value_text(lv_obj_t *group, std::size_t i)
{
	return label_text(item_value_obj(group, i), 0);
}

TEST_CASE("MorePage renders 18 sysinfo item pairs", "[more_page]")
{
	test_support::LvglTestContext lvgl;
	ui::MorePage page;
	page.create(lvgl.screen());

	REQUIRE(page.is_created());

	// SubPageBase back button + content group only; no Reset button.
	lv_obj_t *scroll = scroll_content(page);
	REQUIRE(scroll != nullptr);
	REQUIRE(lv_obj_get_child_count(scroll) == 2);

	REQUIRE(back_title(page) == "More");

	lv_obj_t *group = content_group(page);
	REQUIRE(group != nullptr);
	REQUIRE(lv_obj_get_child_count(group) == kItemCount * 2);

	lv_obj_update_layout(lvgl.screen());

	// KV 容器复用 SConAWide 满宽：宽度 == 270 且铺满整屏宽度，
	// 左缘位于返回按钮左侧之外（抵消子页面 pad_left=13 的负边距）。
	REQUIRE(lv_obj_get_width(group) == 270);
	REQUIRE(lv_obj_get_width(group) ==
		lv_obj_get_width(lvgl.screen()));
	REQUIRE(lv_obj_get_x(group) < lv_obj_get_x(back_button(page)));

	for (std::size_t i = 0; i < kItemCount; ++i) {
		lv_obj_t *desc_obj = item_desc_obj(group, i);
		lv_obj_t *value_obj = item_value_obj(group, i);
		REQUIRE(desc_obj != nullptr);
		REQUIRE(value_obj != nullptr);

		INFO("item " << i << " label");
		REQUIRE(label_text(desc_obj, 0) == kExpectedLabels[i]);
		REQUIRE(item_value_text(group, i) == "--");

		// desc stays static wrap; value scrolls circularly when long.
		REQUIRE(lv_label_get_long_mode(visible_label_at(desc_obj, 0)) ==
			LV_LABEL_LONG_WRAP);
		REQUIRE(lv_label_get_long_mode(visible_label_at(value_obj, 0)) ==
			LV_LABEL_LONG_SCROLL_CIRCULAR);

		// desc is left-aligned (positive x offset from alignment),
		// value is right-aligned (negative x offset).
		REQUIRE(lv_obj_get_x_aligned(visible_label_at(desc_obj, 0)) > 0);
		REQUIRE(lv_obj_get_x_aligned(visible_label_at(value_obj, 0)) < 0);
	}

	page.destroy();
	REQUIRE_FALSE(page.is_created());
}

TEST_CASE("MorePage setters update the five collected items", "[more_page]")
{
	test_support::LvglTestContext lvgl;
	ui::MorePage page;
	page.create(lvgl.screen());

	lv_obj_t *group = content_group(page);
	REQUIRE(group != nullptr);

	// Item indices must match MorePage::InfoItem.
	constexpr std::size_t kHwVer = 0;
	constexpr std::size_t kIp = 4;
	constexpr std::size_t kAppVer = 9;
	constexpr std::size_t kResolution = 11;
	constexpr std::size_t kUiVer = 17;

	// fw is intentionally ignored, matching AboutPage::set_versions.
	page.set_versions("2.3.4", "ignored-fw", "rev-c");
	REQUIRE(item_value_text(group, kHwVer) == "rev-c");
	REQUIRE(item_value_text(group, kAppVer) == "2.3.4");

	page.set_ip_candidates({ "10.0.0.1" });
	REQUIRE(item_value_text(group, kIp) == "10.0.0.1");
	// First non-empty candidate wins.
	page.set_ip_candidates({ "", "192.168.1.2", "10.0.0.5" });
	REQUIRE(item_value_text(group, kIp) == "192.168.1.2");
	// No candidates -> "--".
	page.set_ip_candidates({});
	REQUIRE(item_value_text(group, kIp) == "--");

	page.set_resolution("3840x2160@30Hz");
	REQUIRE(item_value_text(group, kResolution) == "3840x2160@30Hz");
	// Empty resolution -> "--".
	page.set_resolution("");
	REQUIRE(item_value_text(group, kResolution) == "--");

	page.set_ui_version("f7a0ee7489a7");
	REQUIRE(item_value_text(group, kUiVer) == "f7a0ee7489a7");
	// Empty UI version -> "--".
	page.set_ui_version("");
	REQUIRE(item_value_text(group, kUiVer) == "--");

	// Uncollected items remain "--".
	constexpr std::size_t kDeviceUid = 1;
	constexpr std::size_t kTemp = 16;
	REQUIRE(item_value_text(group, kDeviceUid) == "--");
	REQUIRE(item_value_text(group, kTemp) == "--");

	page.destroy();
}

TEST_CASE("MorePage sysinfo setters populate the mirror items", "[more_page]")
{
	test_support::LvglTestContext lvgl;
	ui::MorePage page;
	page.create(lvgl.screen());

	lv_obj_t *group = content_group(page);
	REQUIRE(group != nullptr);

	// Item indices must match MorePage::InfoItem.
	constexpr std::size_t kDeviceUid = 1;
	constexpr std::size_t kHostname = 2;
	constexpr std::size_t kDeviceNo = 3;
	constexpr std::size_t kMdns = 5;
	constexpr std::size_t kMac = 6;
	constexpr std::size_t kSignal = 7;
	constexpr std::size_t kImageVer = 8;
	constexpr std::size_t kEdid = 10;
	constexpr std::size_t kUptime = 12;
	constexpr std::size_t kTime = 13;
	constexpr std::size_t kMemory = 14;
	constexpr std::size_t kStorage = 15;
	constexpr std::size_t kTemp = 16;

	page.set_device_key("ax_uid: 0x0123456789abcdef");
	REQUIRE(item_value_text(group, kDeviceUid) == "ax_uid: 0x0123456789abcdef");
	page.set_hostname("kvm-c715");
	REQUIRE(item_value_text(group, kHostname) == "kvm-c715");
	page.set_device_number("AX620QE_LP4_NANOAGENT_512M");
	REQUIRE(item_value_text(group, kDeviceNo) ==
		"AX620QE_LP4_NANOAGENT_512M");
	page.set_mdns_name("kvm-c715.local");
	REQUIRE(item_value_text(group, kMdns) == "kvm-c715.local");
	page.set_mac_address("aa:bb:cc:dd:ee:ff");
	REQUIRE(item_value_text(group, kMac) == "aa:bb:cc:dd:ee:ff");
	page.set_wifi_signal("-45 dBm");
	REQUIRE(item_value_text(group, kSignal) == "-45 dBm");
	page.set_image_version("0.1.6");
	REQUIRE(item_value_text(group, kImageVer) == "0.1.6");
	page.set_edid_name("nanokvm-go-3840x2160-30");
	REQUIRE(item_value_text(group, kEdid) ==
		"nanokvm-go-3840x2160-30");
	page.set_uptime("16:03:25");
	REQUIRE(item_value_text(group, kUptime) == "16:03:25");
	page.set_local_time("2026-08-27 10:28:45");
	REQUIRE(item_value_text(group, kTime) == "2026-08-27 10:28:45");
	page.set_memory("66/246 MiB");
	REQUIRE(item_value_text(group, kMemory) == "66/246 MiB");
	page.set_storage("24.5/29.0 GB");
	REQUIRE(item_value_text(group, kStorage) == "24.5/29.0 GB");
	page.set_temperature("49.4\u00B0C");
	REQUIRE(item_value_text(group, kTemp) == "49.4\u00B0C");

	// Empty values normalize to "--" for every sysinfo setter.
	page.set_device_key("");
	REQUIRE(item_value_text(group, kDeviceUid) == "--");
	page.set_hostname("");
	REQUIRE(item_value_text(group, kHostname) == "--");
	page.set_device_number("");
	REQUIRE(item_value_text(group, kDeviceNo) == "--");
	page.set_mdns_name("");
	REQUIRE(item_value_text(group, kMdns) == "--");
	page.set_mac_address("");
	REQUIRE(item_value_text(group, kMac) == "--");
	page.set_wifi_signal("");
	REQUIRE(item_value_text(group, kSignal) == "--");
	page.set_image_version("");
	REQUIRE(item_value_text(group, kImageVer) == "--");
	page.set_edid_name("");
	REQUIRE(item_value_text(group, kEdid) == "--");
	page.set_uptime("");
	REQUIRE(item_value_text(group, kUptime) == "--");
	page.set_local_time("");
	REQUIRE(item_value_text(group, kTime) == "--");
	page.set_memory("");
	REQUIRE(item_value_text(group, kMemory) == "--");
	page.set_storage("");
	REQUIRE(item_value_text(group, kStorage) == "--");
	page.set_temperature("");
	REQUIRE(item_value_text(group, kTemp) == "--");

	page.destroy();
}

TEST_CASE("SConAWide reuses SConA behaviour at 270px width", "[scona_wide]")
{
	test_support::LvglTestContext lvgl;

	REQUIRE(ui::SConA::WIDTH == 230);
	REQUIRE(ui::SConAWide::WIDTH == 270);

	ui::SConAWide wide(lvgl.screen(), "label", ui::SConA::TextAlign::LEFT);
	wide.create();
	lv_obj_update_layout(lvgl.screen());
	REQUIRE(lv_obj_get_width(wide.get_obj()) == 270);
	REQUIRE(lv_obj_get_height(wide.get_obj()) == 32);
	REQUIRE_FALSE(lv_obj_has_flag(wide.get_obj(), LV_OBJ_FLAG_CLICKABLE));
	wide.destroy();
}
} // namespace
