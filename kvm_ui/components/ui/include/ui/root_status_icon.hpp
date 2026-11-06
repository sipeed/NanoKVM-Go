#pragma once

#include <cstddef>
#include <cstdint>

namespace ui
{

enum class RootStatusIcon : uint8_t {
	Wifi,
	Video,
	KeyboardMouse,
	Count,
};

inline constexpr std::size_t kRootStatusIconCount =
	static_cast<std::size_t>(RootStatusIcon::Count);

constexpr bool is_valid_root_status_icon(RootStatusIcon icon) noexcept
{
	return static_cast<std::size_t>(icon) < kRootStatusIconCount;
}

constexpr std::size_t root_status_icon_index(RootStatusIcon icon) noexcept
{
	return static_cast<std::size_t>(icon);
}

} // namespace ui
