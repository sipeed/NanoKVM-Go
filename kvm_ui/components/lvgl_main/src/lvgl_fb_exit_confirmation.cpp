/**
 * @file lvgl_fb_exit_confirmation.cpp
 * Direct framebuffer clearing and exit confirmation rendering for app sessions.
 */

#include <algorithm>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <string>
#include <thread>

#include <fcntl.h>
#include <linux/fb.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <unistd.h>

#include "lvgl_ui_internal.hpp"

#if !NANOKVM_GO_SIMULATOR_BUILD

namespace
{

static constexpr uint16_t kDialog = 0x2104;
static constexpr uint16_t kBorder = 0x7BEF;
static constexpr uint16_t kNoButton = 0x4208;
static constexpr uint16_t kYesButton = 0xC800;
static constexpr uint16_t kRed = 0xF800;
static constexpr uint16_t kWhite = 0xFFFF;

} // namespace

// Zero the entire framebuffer. A sub-app writes raw to the whole device,
// including the ~14px "dead area" column that LVGL never repaints (it draws
// only the offset visible region). Without wiping the full buffer, that column
// keeps the app's pixels after exit. Called before forcing an LVGL repaint.
void clear_framebuffer(const std::string &dev)
{
	const int fd = ::open(dev.c_str(), O_RDWR);
	if (fd < 0) {
		return;
	}
	struct fb_fix_screeninfo finfo;
	if (::ioctl(fd, FBIOGET_FSCREENINFO, &finfo) == 0 &&
	    finfo.smem_len > 0) {
		void *p = ::mmap(nullptr, finfo.smem_len, PROT_WRITE,
				 MAP_SHARED, fd, 0);
		if (p != MAP_FAILED) {
			::memset(p, 0, finfo.smem_len);
			::munmap(p, finfo.smem_len);
		}
	}
	::close(fd);
}

// Draws a centered exit confirmation directly onto the framebuffer while the
// sub-app is frozen (SIGSTOP). The original dialog area is backed up so choosing
// No can restore the app byte-for-byte without repainting the full screen.
bool ExitConfirmationOverlay::open(const std::string &dev, int rot)
{
	const int fd = ::open(dev.c_str(), O_RDWR);
	if (fd < 0) {
		return false;
	}
	struct fb_fix_screeninfo finfo;
	struct fb_var_screeninfo vinfo;
	if (::ioctl(fd, FBIOGET_FSCREENINFO, &finfo) != 0 ||
	    ::ioctl(fd, FBIOGET_VSCREENINFO, &vinfo) != 0) {
		::close(fd);
		return false;
	}
	stride = static_cast<int>(finfo.line_length);
	xres = static_cast<int>(vinfo.xres);
	yres = static_cast<int>(vinfo.yres);
	len = finfo.smem_len;
	void *mapped = ::mmap(nullptr, len, PROT_READ | PROT_WRITE,
			      MAP_SHARED, fd, 0);
	::close(fd);
	if (mapped == MAP_FAILED) {
		return false;
	}
	base = static_cast<uint8_t *>(mapped);
	rotate = rot;
	if (rotate == 90 || rotate == 270) {
		vw = yres;
		vh = xres;
	} else {
		vw = xres;
		vh = yres;
	}
	layout = lvgl_main::launcher_exit_confirmation_layout(vw, vh);
	if (!layout.valid) {
		close();
		return false;
	}
	backup.reserve(len);
	return true;
}

void ExitConfirmationOverlay::close()
{
	if (base != nullptr) {
		::munmap(base, len);
		base = nullptr;
	}
}

void ExitConfirmationOverlay::to_phys(int logical_x, int logical_y, int &physical_x,
	     int &physical_y) const
{
	if (rotate == 90) {
		physical_x = xres - 1 - logical_y;
		physical_y = logical_x;
	} else if (rotate == 270) {
		physical_x = logical_y;
		physical_y = yres - 1 - logical_x;
	} else {
		physical_x = logical_x;
		physical_y = logical_y;
	}
}

bool ExitConfirmationOverlay::offset_of(int logical_x, int logical_y, size_t &offset) const
{
	int physical_x = 0;
	int physical_y = 0;
	to_phys(logical_x, logical_y, physical_x, physical_y);
	if (physical_x < 0 || physical_y < 0 || physical_x >= xres ||
	    physical_y >= yres) {
		return false;
	}
	offset = static_cast<size_t>(physical_y) * stride +
		 static_cast<size_t>(physical_x) * 2;
	return offset + 1 < len;
}

void ExitConfirmationOverlay::snapshot()
{
	if (base == nullptr || snapped) {
		return;
	}
	backup.assign(base, base + len);
	snapped = true;
}

bool ExitConfirmationOverlay::frame_index(int logical_x, int logical_y, size_t &index) const
{
	if (!layout.dialog.contains(logical_x, logical_y)) {
		return false;
	}
	index = static_cast<size_t>(logical_y - layout.dialog.y) *
			static_cast<size_t>(layout.dialog.width) +
		static_cast<size_t>(logical_x - layout.dialog.x);
	return index < frame.size();
}

void ExitConfirmationOverlay::begin_frame()
{
	if (!snapped || backup.size() != len) {
		return;
	}
	frame.assign(static_cast<size_t>(layout.dialog.width) *
			     static_cast<size_t>(layout.dialog.height),
		     0);
	for (int logical_y = layout.dialog.y;
	     logical_y < layout.dialog.y + layout.dialog.height;
	     ++logical_y) {
		for (int logical_x = layout.dialog.x;
		     logical_x < layout.dialog.x + layout.dialog.width;
		     ++logical_x) {
			size_t offset = 0;
			size_t index = 0;
			if (offset_of(logical_x, logical_y, offset) &&
			    frame_index(logical_x, logical_y, index)) {
				frame[index] = static_cast<uint16_t>(
					backup[offset] |
					(static_cast<uint16_t>(backup[offset + 1])
					 << 8));
			}
		}
	}
}

void ExitConfirmationOverlay::commit_frame()
{
	if (base == nullptr || frame.empty()) {
		return;
	}
	for (int logical_y = layout.dialog.y;
	     logical_y < layout.dialog.y + layout.dialog.height;
	     ++logical_y) {
		for (int logical_x = layout.dialog.x;
		     logical_x < layout.dialog.x + layout.dialog.width;
		     ++logical_x) {
			size_t offset = 0;
			size_t index = 0;
			if (offset_of(logical_x, logical_y, offset) &&
			    frame_index(logical_x, logical_y, index)) {
				base[offset] = frame[index] & 0xFF;
				base[offset + 1] =
					(frame[index] >> 8) & 0xFF;
			}
		}
	}
	frame.clear();
}

void ExitConfirmationOverlay::restore()
{
	if (base == nullptr || !snapped || backup.size() != len) {
		return;
	}
	for (int logical_y = layout.dialog.y;
	     logical_y < layout.dialog.y + layout.dialog.height;
	     ++logical_y) {
		for (int logical_x = layout.dialog.x;
		     logical_x < layout.dialog.x + layout.dialog.width;
		     ++logical_x) {
			size_t offset = 0;
			if (offset_of(logical_x, logical_y, offset)) {
				base[offset] = backup[offset];
				base[offset + 1] = backup[offset + 1];
			}
		}
	}
	snapped = false;
	frame.clear();
}

void ExitConfirmationOverlay::set_pixel(int logical_x, int logical_y, uint16_t color)
{
	size_t index = 0;
	if (frame_index(logical_x, logical_y, index)) {
		frame[index] = color;
		return;
	}
	size_t offset = 0;
	if (!offset_of(logical_x, logical_y, offset)) {
		return;
	}
	base[offset] = color & 0xFF;
	base[offset + 1] = (color >> 8) & 0xFF;
}

uint16_t ExitConfirmationOverlay::blend_rgb565(uint16_t background, uint16_t foreground,
			     uint8_t opacity)
{
	if (opacity == 255) {
		return foreground;
	}
	const uint32_t inverse = 255U - opacity;
	const uint32_t red =
		((((background >> 11) & 0x1FU) * inverse) +
		 (((foreground >> 11) & 0x1FU) * opacity) + 127U) /
		255U;
	const uint32_t green =
		((((background >> 5) & 0x3FU) * inverse) +
		 (((foreground >> 5) & 0x3FU) * opacity) + 127U) /
		255U;
	const uint32_t blue =
		(((background & 0x1FU) * inverse) +
		 ((foreground & 0x1FU) * opacity) + 127U) /
		255U;
	return static_cast<uint16_t>((red << 11) | (green << 5) | blue);
}

void ExitConfirmationOverlay::blend_pixel(int logical_x, int logical_y, uint16_t color,
		 uint8_t opacity)
{
	size_t index = 0;
	if (frame_index(logical_x, logical_y, index)) {
		frame[index] = blend_rgb565(frame[index], color, opacity);
		return;
	}
	set_pixel(logical_x, logical_y, color);
}

int ExitConfirmationOverlay::scale_position(int value, int center, uint8_t scale_percent)
{
	return center + (value - center) * scale_percent / 100;
}

lvgl_main::LauncherExitRect ExitConfirmationOverlay::scale_rect(
	const lvgl_main::LauncherExitRect &rect,
	const lvgl_main::LauncherExitRect &dialog, uint8_t scale_percent)
{
	const int center_x = dialog.x + dialog.width / 2;
	const int center_y = dialog.y + dialog.height / 2;
	return {
		scale_position(rect.x, center_x, scale_percent),
		scale_position(rect.y, center_y, scale_percent),
		std::max(1, rect.width * scale_percent / 100),
		std::max(1, rect.height * scale_percent / 100),
	};
}

void ExitConfirmationOverlay::fill_rounded_rect(const lvgl_main::LauncherExitRect &rect,
		       int radius, uint16_t color, uint8_t opacity)
{
	const double effective_radius = std::min(
		static_cast<double>(radius),
		std::min(rect.width, rect.height) / 2.0);
	for (int logical_y = rect.y; logical_y < rect.y + rect.height;
	     ++logical_y) {
		for (int logical_x = rect.x;
		     logical_x < rect.x + rect.width; ++logical_x) {
			const double pixel_x = logical_x + 0.5;
			const double pixel_y = logical_y + 0.5;
			const double nearest_x = std::clamp(
				pixel_x, rect.x + effective_radius,
				rect.x + rect.width - effective_radius);
			const double nearest_y = std::clamp(
				pixel_y, rect.y + effective_radius,
				rect.y + rect.height - effective_radius);
			const double delta_x = pixel_x - nearest_x;
			const double delta_y = pixel_y - nearest_y;
			if (delta_x * delta_x + delta_y * delta_y <=
			    effective_radius * effective_radius) {
				blend_pixel(logical_x, logical_y, color,
					    opacity);
			}
		}
	}
}

const uint8_t *ExitConfirmationOverlay::glyph_rows(char character)
{
	static constexpr uint8_t e[7] = {
		0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F
	};
	static constexpr uint8_t x[7] = {
		0x11, 0x0A, 0x04, 0x04, 0x04, 0x0A, 0x11
	};
	static constexpr uint8_t i[7] = {
		0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x1F
	};
	static constexpr uint8_t t[7] = {
		0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04
	};
	static constexpr uint8_t question[7] = {
		0x0E, 0x11, 0x01, 0x02, 0x04, 0x00, 0x04
	};
	static constexpr uint8_t n[7] = {
		0x11, 0x19, 0x19, 0x15, 0x13, 0x13, 0x11
	};
	static constexpr uint8_t o[7] = {
		0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E
	};
	static constexpr uint8_t y[7] = {
		0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04
	};
	static constexpr uint8_t s[7] = {
		0x0F, 0x10, 0x10, 0x0E, 0x01, 0x01, 0x1E
	};
	switch (character) {
	case 'E':
		return e;
	case 'X':
		return x;
	case 'I':
		return i;
	case 'T':
		return t;
	case '?':
		return question;
	case 'N':
		return n;
	case 'O':
		return o;
	case 'Y':
		return y;
	case 'S':
		return s;
	default:
		return nullptr;
	}
}

void ExitConfirmationOverlay::draw_text_centered(const char *text, int center_x, int top,
			int scale, uint16_t color, uint8_t opacity)
{
	const int length = static_cast<int>(std::strlen(text));
	const int text_width = length * 5 * scale +
			       std::max(0, length - 1) * scale;
	int cursor_x = center_x - text_width / 2;
	for (int character = 0; character < length; ++character) {
		const uint8_t *rows = glyph_rows(text[character]);
		if (rows != nullptr) {
			for (int row = 0; row < 7; ++row) {
				for (int column = 0; column < 5; ++column) {
					if ((rows[row] &
					     (1U << (4 - column))) == 0) {
						continue;
					}
					for (int pixel_y = 0; pixel_y < scale;
					     ++pixel_y) {
						for (int pixel_x = 0;
						     pixel_x < scale; ++pixel_x) {
							blend_pixel(
								cursor_x + column * scale +
									pixel_x,
								top + row * scale +
									pixel_y,
								color, opacity);
						}
					}
				}
			}
		}
		cursor_x += 6 * scale;
	}
}

void ExitConfirmationOverlay::draw_frame(
	const lvgl_main::LauncherExitConfirmationAnimationFrame &animation)
{
	if (base == nullptr || !snapped) {
		return;
	}
	const auto dialog =
		scale_rect(layout.dialog, layout.dialog,
			   animation.scale_percent);
	const lvgl_main::LauncherExitRect dialog_inner {
		layout.dialog.x + 2,
		layout.dialog.y + 2,
		layout.dialog.width - 4,
		layout.dialog.height - 4,
	};
	const auto inner = scale_rect(dialog_inner, layout.dialog,
				      animation.scale_percent);
	const auto no_button = scale_rect(
		layout.no_button, layout.dialog, animation.scale_percent);
	const auto yes_button = scale_rect(
		layout.yes_button, layout.dialog, animation.scale_percent);
	fill_rounded_rect(dialog,
			  std::max(1, 18 * animation.scale_percent / 100),
			  kBorder, animation.opacity);
	fill_rounded_rect(inner,
			  std::max(1, 16 * animation.scale_percent / 100),
			  kDialog, animation.opacity);
	fill_rounded_rect(no_button,
			  std::max(1, 12 * animation.scale_percent / 100),
			  kNoButton, animation.opacity);
	fill_rounded_rect(yes_button,
			  std::max(1, 12 * animation.scale_percent / 100),
			  kYesButton, animation.opacity);
	draw_text_centered("EXIT?", dialog.x + dialog.width / 2,
			   dialog.y + 16 * animation.scale_percent / 100,
			   4, kRed, animation.opacity);
	draw_text_centered(
		"NO", no_button.x + no_button.width / 2,
		no_button.y + (no_button.height - 21) / 2, 3, kWhite,
		animation.opacity);
	draw_text_centered(
		"YES", yes_button.x + yes_button.width / 2,
		yes_button.y + (yes_button.height - 21) / 2, 3, kWhite,
		animation.opacity);
}

void ExitConfirmationOverlay::draw()
{
	begin_frame();
	draw_frame(lvgl_main::launcher_exit_confirmation_animation_frame(
		lvgl_main::LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES - 1));
	commit_frame();
}

void ExitConfirmationOverlay::animate_in()
{
	for (std::size_t index = 0;
	     index < lvgl_main::LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES;
	     ++index) {
		begin_frame();
		draw_frame(
			lvgl_main::launcher_exit_confirmation_animation_frame(
				index));
		commit_frame();
		if (index + 1 <
		    lvgl_main::LAUNCHER_EXIT_CONFIRMATION_ANIMATION_FRAMES) {
			std::this_thread::sleep_for(
				std::chrono::milliseconds(24));
		}
	}
}

lvgl_main::LauncherExitConfirmationChoice ExitConfirmationOverlay::choice_at(int logical_x,
					       int logical_y) const
{
	return lvgl_main::launcher_exit_confirmation_choice(
		layout, logical_x, logical_y);
}

#endif
