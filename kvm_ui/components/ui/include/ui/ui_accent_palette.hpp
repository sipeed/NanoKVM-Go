#pragma once

#include <array>
#include <cstdint>

namespace ui::accent
{

inline constexpr uint32_t ACCENT_PURPLE_950 = 0x3C0366;
inline constexpr uint32_t ACCENT_BLUE_950 = 0x162456;
inline constexpr uint32_t ACCENT_LIME_950 = 0x192E03;
inline constexpr uint32_t ACCENT_CYAN_950 = 0x053345;
inline constexpr uint32_t ACCENT_FUCHSIA_800 = 0x8A0194;
inline constexpr uint32_t ACCENT_INDIGO_700 = 0x432DD7;
inline constexpr uint32_t ACCENT_PINK_800 = 0xA3004C;
inline constexpr uint32_t ACCENT_TEAL_800 = 0x005F5A;
inline constexpr uint32_t ACCENT_GREEN_800 = 0x016630;
inline constexpr uint32_t ACCENT_SKY_700 = 0x0069A8;
inline constexpr uint32_t ACCENT_ORANGE_700 = 0xCA3500;
inline constexpr uint32_t ACCENT_LIME_700 = 0x497D00;
inline constexpr uint32_t ACCENT_YELLOW_700 = 0xA65F00;

inline constexpr std::array<uint32_t, 13> PALETTE{
	ACCENT_PURPLE_950,
	ACCENT_BLUE_950,
	ACCENT_LIME_950,
	ACCENT_CYAN_950,
	ACCENT_FUCHSIA_800,
	ACCENT_INDIGO_700,
	ACCENT_PINK_800,
	ACCENT_TEAL_800,
	ACCENT_GREEN_800,
	ACCENT_SKY_700,
	ACCENT_ORANGE_700,
	ACCENT_LIME_700,
	ACCENT_YELLOW_700,
};

} // namespace ui::accent
