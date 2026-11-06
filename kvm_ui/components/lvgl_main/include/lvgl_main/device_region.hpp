#pragma once

#include <cstdint>
#include <filesystem>
#include <string_view>

namespace lvgl_main
{

inline constexpr std::string_view kDeviceRegionPath = "/etc/kvm/region";

enum class DeviceRegion : uint8_t {
	Global,
	Cn,
};

DeviceRegion parse_device_region(std::string_view contents) noexcept;
std::string_view device_region_name(DeviceRegion region) noexcept;
DeviceRegion read_device_region(const std::filesystem::path &path);

} // namespace lvgl_main
