#pragma once

#include <cstdint>
#include <string_view>

namespace ui
{

enum class RegionOption : uint8_t {
	ChinaRegion,
	Global,
};

constexpr RegionOption region_other(RegionOption region) noexcept
{
	return region == RegionOption::ChinaRegion ? RegionOption::Global :
						     RegionOption::ChinaRegion;
}

constexpr RegionOption region_option_from_cn(bool is_cn) noexcept
{
	return is_cn ? RegionOption::ChinaRegion : RegionOption::Global;
}

constexpr std::string_view region_option_label(RegionOption region) noexcept
{
	return region == RegionOption::ChinaRegion ? "CN" : "Global";
}

} // namespace ui
