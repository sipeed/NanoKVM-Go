#pragma once

#include <array>
#include <cstddef>
#include <string>
#include <vector>

#include "ui/scona_wide.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

/**
 * @brief More 子页面
 *
 * 由 About 子页面连续点击 5 次 APP Ver 后进入的详情信息子页面。按
 * `MorePage::InfoItem` 的顺序（见 more_page.cpp 的 kInfoLabels 数组）
 * 依次展示 18 项设备系统信息；每项由两个 SConA 纵向堆叠
 * 构成：第一个左对齐显示描述，第二个右对齐显示实际值。已采集项显示真实
 * 值，未采集项显示 "--" 占位。
 */
class MorePage : public SubPageBase {
    public:
	MorePage();
	~MorePage() override = default;

	/// 更新 HW Ver 与 APP Ver（fw 参数被忽略，与 About 一致）。
	void set_versions(const std::string &app, const std::string &fw,
			  const std::string &hw);
	/// 更新 IP 项：取首个非空候选，空则回退 "--"。
	void set_ip_candidates(const std::vector<std::string> &candidates);
	/// 更新 Resolution 项。
	void set_resolution(const std::string &label);
	/// 更新 UI Version 项（构建期 git hash），空则回退 "--"。
	void set_ui_version(const std::string &version);

	// More 子页面设备系统信息 setter（值来自 RuntimeStatus 投影）。空值
	// 由 set_value 统一回退 "--"，与既有 setter 一致。
	void set_device_key(const std::string &value);
	void set_hostname(const std::string &value);
	void set_device_number(const std::string &value);
	void set_image_version(const std::string &value);
	void set_edid_name(const std::string &value);
	void set_mac_address(const std::string &value);
	void set_mdns_name(const std::string &value);
	void set_wifi_signal(const std::string &value);
	void set_uptime(const std::string &value);
	void set_local_time(const std::string &value);
	void set_memory(const std::string &value);
	void set_storage(const std::string &value);
	void set_temperature(const std::string &value);

    private:
	enum class InfoItem : std::size_t {
		HwVer = 0, DeviceUid, Hostname, DeviceNo,
		Ip, Mdns, Mac, Signal,
		ImageVer, AppVer,
		Edid, Resolution,
		Uptime, Time, Memory, Storage, Temp, UiVer,
		Count,
	};

	struct ItemRow {
		SConA *desc = nullptr;
		SConA *value = nullptr;
	};

	static constexpr std::size_t kItemCount =
		static_cast<std::size_t>(InfoItem::Count);

	lv_obj_t *_content_group = nullptr;
	std::array<ItemRow, kItemCount> _items{};

	void build_content() override;
	void set_value(InfoItem item, const std::string &value);
	void on_subpage_destroy() override;
};

} // namespace ui
