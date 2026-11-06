#pragma once

#include <cstddef>
#include <cstdint>

namespace ui::root_layout
{

enum class ContentPage : uint8_t {
	Main,
	SettingsHome,
	SettingsSecond,
	Count,
};

inline constexpr uint16_t kScreenWidth = 270;
inline constexpr uint16_t kScreenHeight = 240;
inline constexpr uint16_t kPageGap = 12;
inline constexpr uint16_t kContentAreaY = 58;
inline constexpr uint16_t kContentCardWidth = kScreenWidth;
inline constexpr uint16_t kContentCardHeight = 170;
inline constexpr uint16_t kContentCardRadius = 40;
inline constexpr std::size_t kContentPageCount =
	static_cast<std::size_t>(ContentPage::Count);
inline constexpr int32_t kContentPageStep =
	static_cast<int32_t>(kContentCardWidth) + kPageGap;
inline constexpr int32_t kPageIndicatorY =
	static_cast<int32_t>(kContentAreaY) + kContentCardHeight + 4;

constexpr int32_t content_page_index(ContentPage page) noexcept
{
	const auto index = static_cast<std::size_t>(page);
	return index < kContentPageCount ? static_cast<int32_t>(index) : 0;
}

constexpr ContentPage content_page_from_index(int32_t index) noexcept
{
	if (index <= 0) {
		return ContentPage::Main;
	}
	const int32_t last_index = static_cast<int32_t>(kContentPageCount - 1U);
	if (index >= last_index) {
		return static_cast<ContentPage>(last_index);
	}
	return static_cast<ContentPage>(index);
}

constexpr int32_t content_page_x(ContentPage page) noexcept
{
	return content_page_index(page) * kContentPageStep;
}

} // namespace ui::root_layout
