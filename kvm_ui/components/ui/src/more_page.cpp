#include "ui/more_page.hpp"

#include <cstddef>
#include <iterator>
#include <memory>
#include <string>
#include <vector>

#include "lvgl.h"

namespace ui
{

MorePage::MorePage()
	: SubPageBase(PageType::SUBPAGE_MORE, "More", 0)
{
}

void MorePage::set_versions(const std::string &app, const std::string &fw,
			    const std::string &hw)
{
	(void)fw;
	set_value(InfoItem::HwVer, hw);
	set_value(InfoItem::AppVer, app);
}

void MorePage::set_ip_candidates(const std::vector<std::string> &candidates)
{
	std::string chosen;
	for (const auto &candidate : candidates) {
		if (!candidate.empty()) {
			chosen = candidate;
			break;
		}
	}
	set_value(InfoItem::Ip, chosen);
}

void MorePage::set_resolution(const std::string &label)
{
	set_value(InfoItem::Resolution, label);
}

void MorePage::set_ui_version(const std::string &version)
{
	set_value(InfoItem::UiVer, version);
}

void MorePage::set_device_key(const std::string &value)
{
	set_value(InfoItem::DeviceUid, value);
}

void MorePage::set_hostname(const std::string &value)
{
	set_value(InfoItem::Hostname, value);
}

void MorePage::set_device_number(const std::string &value)
{
	set_value(InfoItem::DeviceNo, value);
}

void MorePage::set_image_version(const std::string &value)
{
	set_value(InfoItem::ImageVer, value);
}

void MorePage::set_edid_name(const std::string &value)
{
	set_value(InfoItem::Edid, value);
}

void MorePage::set_mac_address(const std::string &value)
{
	set_value(InfoItem::Mac, value);
}

void MorePage::set_mdns_name(const std::string &value)
{
	set_value(InfoItem::Mdns, value);
}

void MorePage::set_wifi_signal(const std::string &value)
{
	set_value(InfoItem::Signal, value);
}

void MorePage::set_uptime(const std::string &value)
{
	set_value(InfoItem::Uptime, value);
}

void MorePage::set_local_time(const std::string &value)
{
	set_value(InfoItem::Time, value);
}

void MorePage::set_memory(const std::string &value)
{
	set_value(InfoItem::Memory, value);
}

void MorePage::set_storage(const std::string &value)
{
	set_value(InfoItem::Storage, value);
}

void MorePage::set_temperature(const std::string &value)
{
	set_value(InfoItem::Temp, value);
}

void MorePage::build_content()
{
	constexpr int32_t item_gap = 4;

	// 顺序须与 MorePage::InfoItem 一致。
	constexpr const char *kInfoLabels[] = {
		"Hardware Version:", "Device Key:",   "Hostname:", "Device Number:",
		"IP:",                "mDNS:",         "MAC:",      "WiFi Signal:",
		"Image Version:",     "APP Version:",  "EDID:",
		"Resolution:",        "Uptime:",       "Time:",     "Memory:",
		"Storage:",           "Temperature:", "UI Version:",
	};
	static_assert(std::size(kInfoLabels) == kItemCount,
		      "label count must match InfoItem count");

	_content_group = lv_obj_create(_scroll_content);
	lv_obj_set_width(_content_group, SConAWide::WIDTH);
	lv_obj_set_height(_content_group, LV_SIZE_CONTENT);
	lv_obj_set_style_bg_opa(_content_group, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(_content_group, 0, 0);
	lv_obj_set_style_pad_all(_content_group, 0, 0);
	lv_obj_set_flex_flow(_content_group, LV_FLEX_FLOW_COLUMN);
	lv_obj_set_style_pad_row(_content_group, 0, 0);
	lv_obj_clear_flag(_content_group, LV_OBJ_FLAG_SCROLLABLE);
	// 抵消子页面左侧 pad_left=13，使内容组从 x=0 满宽铺满 270 视觉画布。
	lv_obj_set_style_margin_left(_content_group,
				     -static_cast<int32_t>(LEFT_MARGIN), 0);

	for (std::size_t i = 0; i < kItemCount; ++i) {
		auto desc = std::make_unique<SConAWide>(
			_content_group, kInfoLabels[i],
			SConA::TextAlign::LEFT);
		desc->create();
		_items[i].desc = desc.get();
		_components.push_back(std::move(desc));

		auto value = std::make_unique<SConAWide>(
			_content_group, "--", SConA::TextAlign::RIGHT);
		value->create();
		value->set_single_text_scroll_circular(true);
		_items[i].value = value.get();
		if (i + 1 < kItemCount) {
			lv_obj_set_style_margin_bottom(value->get_obj(),
						       item_gap, 0);
		}
		_components.push_back(std::move(value));
	}
}

void MorePage::set_value(InfoItem item, const std::string &value)
{
	const std::string normalized =
		value.empty() ? std::string("--") : value;
	const auto index = static_cast<std::size_t>(item);
	if (_items[index].value != nullptr) {
		_items[index].value->set_text(normalized.c_str());
	}
}

void MorePage::on_subpage_destroy()
{
	_content_group = nullptr;
	for (auto &row : _items) {
		row.desc = nullptr;
		row.value = nullptr;
	}
}

} // namespace ui
