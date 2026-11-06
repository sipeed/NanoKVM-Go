#pragma once

/*
 * Keep the product LVGL configuration as the source of shared widget and
 * rendering settings.  The simulator replaces only the platform drivers.
 */
#include "lv_conf.h"

#undef LV_USE_DRAW_SDL
#define LV_USE_DRAW_SDL 0

#undef LV_USE_SDL
#define LV_USE_SDL	       1
#define LV_SDL_INCLUDE_PATH    <SDL2/SDL.h>
#define LV_SDL_RENDER_MODE     LV_DISPLAY_RENDER_MODE_PARTIAL
#define LV_SDL_BUF_COUNT       1
#define LV_SDL_ACCELERATED     0
#define LV_SDL_FULLSCREEN      0
#define LV_SDL_DIRECT_EXIT     0
#define LV_SDL_MOUSEWHEEL_MODE LV_SDL_MOUSEWHEEL_MODE_ENCODER

#undef LV_USE_LINUX_FBDEV
#define LV_USE_LINUX_FBDEV 0

#undef LV_USE_EVDEV
#define LV_USE_EVDEV 0
