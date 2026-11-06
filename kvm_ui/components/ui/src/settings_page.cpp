#include "ui/settings_page.hpp"

#include <array>
#include <utility>
#include <vector>

#include "lv_safe.hpp"
#include "ui/background_assets.hpp"
#include "ui/settings_tile_layout.hpp"

#ifndef NANOKVM_GO_RELEASE_BUILD
#define NANOKVM_GO_RELEASE_BUILD 0
#endif

extern "C" {
LV_IMAGE_DECLARE(img_settings_main_wifi);
LV_IMAGE_DECLARE(settings_agent_btn);
LV_IMAGE_DECLARE(settings_replay);
LV_IMAGE_DECLARE(settings_eth);
LV_IMAGE_DECLARE(settings_ssh);
LV_IMAGE_DECLARE(settings_auto_180);
LV_IMAGE_DECLARE(settings_screen);
LV_IMAGE_DECLARE(settings_about);
LV_IMAGE_DECLARE(settings_apps);
LV_IMAGE_DECLARE(settings_tailscale);
LV_IMAGE_DECLARE(settings_version_switch);
}

namespace ui
{

namespace
{

struct TileSlot {
	int16_t x;
	int16_t y;
};

constexpr std::array<TileSlot, 6> TILE_SLOTS = { { { 16, 16 },
						   { 102, 16 },
						   { 188, 16 },
						   { 16, 91 },
						   { 102, 91 },
						   { 188, 91 } } };

static_assert(static_cast<int>(SettingsPage::SettingsTile::WIFI) ==
		      static_cast<int>(settings_tile_layout::SettingsTile::WIFI));
static_assert(static_cast<int>(SettingsPage::SettingsTile::REGION) ==
		      static_cast<int>(
			      settings_tile_layout::SettingsTile::REGION));
static_assert(static_cast<int>(SettingsPage::SettingsTile::NETWORK) ==
		      static_cast<int>(
			      settings_tile_layout::SettingsTile::NETWORK));

}

class SettingsPage::TileButton : public ControlState {
    public:
	using ClickCallback = std::function<void()>;

	TileButton(lv_obj_t *parent, const lv_image_dsc_t *icon,
		   uint32_t active_color)
		: _parent(parent)
		, _icon(icon)
		, _active_color(active_color)
	{
	}

	~TileButton() override
	{
		destroy();
	}

	void create()
	{
		if (_obj != nullptr) {
			return;
		}

		_obj = lv_obj_create(_parent);
		lv_obj_set_size(_obj, WIDTH, HEIGHT);
		lv_obj_set_style_bg_color(_obj, lv_color_hex(BG_COLOR), 0);
		lv_obj_set_style_bg_opa(_obj, LV_OPA_COVER, 0);
		lv_obj_set_style_radius(_obj, RADIUS, 0);
		lv_obj_set_style_border_width(_obj, BORDER_WIDTH, 0);
		lv_obj_set_style_pad_all(_obj, 0, 0);
		lv_obj_set_style_clip_corner(_obj, true, 0);
		lv_obj_remove_flag(_obj, LV_OBJ_FLAG_SCROLLABLE);
		lv_obj_clear_flag(_obj, LV_OBJ_FLAG_CLICKABLE);

		_image_obj = lv_image_create(_obj);
		lv_image_set_src(_image_obj, _icon);
		lv_obj_center(_image_obj);
		lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_CLICKABLE);

		_disabled_overlay_obj = lv_obj_create(_obj);
		lv_obj_set_size(_disabled_overlay_obj, WIDTH, HEIGHT);
		lv_obj_set_pos(_disabled_overlay_obj, 0, 0);
		lv_obj_set_style_bg_color(_disabled_overlay_obj,
					  lv_color_hex(DISABLED_OVERLAY_COLOR),
					  0);
		lv_obj_set_style_bg_opa(_disabled_overlay_obj,
					DISABLED_OVERLAY_OPA, 0);
		lv_obj_set_style_radius(_disabled_overlay_obj, RADIUS, 0);
		lv_obj_set_style_border_width(_disabled_overlay_obj, 0, 0);
		lv_obj_set_style_pad_all(_disabled_overlay_obj, 0, 0);
		lv_obj_clear_flag(_disabled_overlay_obj,
				  LV_OBJ_FLAG_SCROLLABLE);
		lv_obj_add_flag(_disabled_overlay_obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_add_flag(_disabled_overlay_obj, LV_OBJ_FLAG_HIDDEN);

		lv_obj_add_event_cb(_obj, press_event_cb, LV_EVENT_PRESSED,
				    this);
		lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_RELEASED,
				    this);
		lv_obj_add_event_cb(_obj, release_event_cb, LV_EVENT_PRESS_LOST,
				    this);
		lv_obj_add_event_cb(_obj, click_event_cb, LV_EVENT_CLICKED,
				    this);

		update_appearance();
	}

	void destroy()
	{
		set_pending(false);
		cancel_deferred_apply();
		_click_callback = nullptr;
		_image_obj = nullptr;
		_disabled_overlay_obj = nullptr;
		_pending_arc_obj = nullptr;
		lv_delete_if_valid(_obj);
	}

	void set_position(int16_t x, int16_t y)
	{
		if (_obj != nullptr) {
			lv_obj_set_pos(_obj, x, y);
		}
	}

	void set_parent(lv_obj_t *parent)
	{
		if (parent == nullptr || parent == _parent) {
			return;
		}

		_parent = parent;
		if (_obj != nullptr) {
			lv_obj_set_parent(_obj, parent);
		}
	}

	void set_visible(bool visible)
	{
		if (_obj == nullptr) {
			return;
		}

		if (visible) {
			lv_obj_remove_flag(_obj, LV_OBJ_FLAG_HIDDEN);
			return;
		}

		set_pending(false);
		lv_obj_add_flag(_obj, LV_OBJ_FLAG_HIDDEN);
	}

	bool is_visible() const
	{
		return _obj != nullptr &&
		       !lv_obj_has_flag(_obj, LV_OBJ_FLAG_HIDDEN);
	}

	void on_click(ClickCallback callback)
	{
		_click_callback = std::move(callback);
		if (_obj == nullptr) {
			return;
		}
		update_clickable_state();
	}

	void set_available(bool available)
	{
		_available = available;
		if (_obj == nullptr) {
			return;
		}

		if (!_available) {
			set_pending(false);
			_is_pressed = false;
		}
		update_clickable_state();
		update_disabled_overlay();
		update_appearance();
	}

	void set_pending_timeout(uint32_t timeout_ms) override
	{
		const uint32_t next_timeout =
			timeout_ms == 0 ? DEFAULT_PENDING_TIMEOUT_MS :
					  timeout_ms;
		if (_pending_timeout_ms == next_timeout) {
			return;
		}
		_pending_timeout_ms = next_timeout;
		if (_pending) {
			start_pending_timers();
		}
	}

	void
	set_visual_state(VisualState state,
			 ApplyPolicy policy = ApplyPolicy::DEFERRED) override
	{
		_target_state = state;
		if (policy == ApplyPolicy::IMMEDIATE || _obj == nullptr) {
			_state = _target_state;
			_state_dirty = false;
			update_appearance();
			return;
		}

		_state_dirty = true;
		schedule_deferred_apply();
	}

	VisualState get_visual_state() const override
	{
		return _state_dirty ? _target_state : _state;
	}

	void set_pending(bool pending) override
	{
		if (!_available && pending) {
			pending = false;
		}
		if (_obj == nullptr) {
			_pending = false;
			stop_pending_timers();
			return;
		}
		if (pending == _pending) {
			return;
		}

		if (pending) {
			ensure_pending_arc();
			_pending = true;
			_is_pressed = false;
			_pending_rotation = 0;
			_pending_image_was_visible =
				_image_obj != nullptr &&
				!lv_obj_has_flag(_image_obj,
						 LV_OBJ_FLAG_HIDDEN);
			if (_image_obj != nullptr) {
				lv_obj_add_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
			}
			if (_pending_arc_obj != nullptr) {
				lv_arc_set_rotation(_pending_arc_obj,
						    _pending_rotation);
				lv_obj_clear_flag(_pending_arc_obj,
						  LV_OBJ_FLAG_HIDDEN);
			}
			update_clickable_state();
			update_appearance();
			start_pending_timers();
			return;
		}

		stop_pending_timers();
		_pending = false;
		if (_pending_arc_obj != nullptr) {
			lv_obj_add_flag(_pending_arc_obj, LV_OBJ_FLAG_HIDDEN);
		}
		if (_image_obj != nullptr && _pending_image_was_visible) {
			lv_obj_clear_flag(_image_obj, LV_OBJ_FLAG_HIDDEN);
		}
		_pending_image_was_visible = false;
		update_clickable_state();
		update_appearance();
	}

	bool is_pending() const override
	{
		return _pending;
	}

    private:
	static constexpr int32_t WIDTH = 64;
	static constexpr int32_t HEIGHT = 64;
	static constexpr int32_t RADIUS = 24;
	static constexpr int32_t BORDER_WIDTH = 4;
	static constexpr uint32_t BG_COLOR = 0x1C1917;
	static constexpr uint32_t DISABLED_OVERLAY_COLOR = 0x737373;
	static constexpr lv_opa_t DISABLED_OVERLAY_OPA = LV_OPA_50;
	static constexpr uint32_t DEFAULT_PENDING_TIMEOUT_MS = 10000;

	lv_obj_t *_parent = nullptr;
	lv_obj_t *_obj = nullptr;
	lv_obj_t *_image_obj = nullptr;
	lv_obj_t *_disabled_overlay_obj = nullptr;
	const lv_image_dsc_t *_icon = nullptr;
	uint32_t _active_color = BG_COLOR;
	VisualState _state = VisualState::DEFAULT;
	VisualState _target_state = VisualState::DEFAULT;
	bool _state_dirty = false;
	bool _apply_scheduled = false;
	bool _is_pressed = false;
	bool _available = true;
	bool _pending = false;
	bool _pending_image_was_visible = false;
	uint16_t _pending_rotation = 0;
	uint32_t _pending_timeout_ms = DEFAULT_PENDING_TIMEOUT_MS;
	ClickCallback _click_callback;

	lv_obj_t *_pending_arc_obj = nullptr;
	lv_timer_t *_pending_spin_timer = nullptr;
	lv_timer_t *_pending_timeout_timer = nullptr;

	static uint32_t apply_press_overlay(uint32_t base_color)
	{
		uint8_t r = (base_color >> 16) & 0xFF;
		uint8_t g = (base_color >> 8) & 0xFF;
		uint8_t b = base_color & 0xFF;

		r = static_cast<uint8_t>(r * 0.8f);
		g = static_cast<uint8_t>(g * 0.8f);
		b = static_cast<uint8_t>(b * 0.8f);

		return (r << 16) | (g << 8) | b;
	}

	uint32_t current_border_color() const
	{
		return _state == VisualState::ACTIVE ? _active_color : BG_COLOR;
	}

	void update_appearance()
	{
		if (_obj == nullptr) {
			return;
		}

		uint32_t bg_color = BG_COLOR;
		uint32_t border_color = current_border_color();
		if (_is_pressed) {
			bg_color = apply_press_overlay(BG_COLOR);
			border_color = apply_press_overlay(border_color);
		}

		lv_obj_set_style_bg_color(_obj, lv_color_hex(bg_color), 0);
		lv_obj_set_style_border_color(_obj, lv_color_hex(border_color),
					      0);
	}

	void update_clickable_state()
	{
		if (_obj == nullptr) {
			return;
		}
		if (_click_callback && _available && !_pending) {
			lv_obj_add_flag(_obj, LV_OBJ_FLAG_CLICKABLE);
			return;
		}
		lv_obj_clear_flag(_obj, LV_OBJ_FLAG_CLICKABLE);
	}

	void update_disabled_overlay()
	{
		if (_disabled_overlay_obj == nullptr) {
			return;
		}
		if (_available) {
			lv_obj_add_flag(_disabled_overlay_obj,
					LV_OBJ_FLAG_HIDDEN);
			return;
		}
		lv_obj_clear_flag(_disabled_overlay_obj, LV_OBJ_FLAG_HIDDEN);
	}

	void apply_pending_state()
	{
		if (!_state_dirty) {
			return;
		}

		_state = _target_state;
		_state_dirty = false;
		update_appearance();
	}

	void schedule_deferred_apply()
	{
		if (_obj == nullptr || _apply_scheduled) {
			return;
		}

		_apply_scheduled = true;
		lv_async_call(async_apply_cb, this);
	}

	void cancel_deferred_apply()
	{
		if (!_apply_scheduled) {
			return;
		}

		lv_async_call_cancel(async_apply_cb, this);
		_apply_scheduled = false;
	}

	void ensure_pending_arc()
	{
		if (_obj == nullptr || _pending_arc_obj != nullptr) {
			return;
		}

		_pending_arc_obj = lv_arc_create(_obj);
		lv_obj_set_size(_pending_arc_obj, 36, 36);
		lv_obj_center(_pending_arc_obj);
		lv_obj_clear_flag(_pending_arc_obj, LV_OBJ_FLAG_CLICKABLE);
		lv_obj_clear_flag(_pending_arc_obj, LV_OBJ_FLAG_SCROLLABLE);
		lv_obj_remove_style(_pending_arc_obj, nullptr, LV_PART_KNOB);
		lv_obj_set_style_arc_opa(_pending_arc_obj, LV_OPA_TRANSP,
					 LV_PART_MAIN);
		lv_obj_set_style_bg_opa(_pending_arc_obj, LV_OPA_TRANSP,
					LV_PART_MAIN);
		lv_obj_set_style_border_width(_pending_arc_obj, 0,
					      LV_PART_MAIN);
		lv_obj_set_style_pad_all(_pending_arc_obj, 0, LV_PART_MAIN);
		lv_obj_set_style_arc_width(_pending_arc_obj, 4,
					   LV_PART_INDICATOR);
		lv_obj_set_style_arc_rounded(_pending_arc_obj, true,
					     LV_PART_INDICATOR);
		lv_obj_set_style_arc_color(_pending_arc_obj,
					   lv_color_hex(_active_color),
					   LV_PART_INDICATOR);
		lv_arc_set_bg_angles(_pending_arc_obj, 0, 360);
		lv_arc_set_angles(_pending_arc_obj, 0, 270);
		lv_arc_set_rotation(_pending_arc_obj, 0);
		lv_obj_add_flag(_pending_arc_obj, LV_OBJ_FLAG_HIDDEN);
	}

	void start_pending_timers()
	{
		stop_pending_timers();
		_pending_spin_timer =
			lv_timer_create(pending_spin_timer_cb, 40, this);
		_pending_timeout_timer = lv_timer_create(
			pending_timeout_timer_cb, _pending_timeout_ms, this);
		lv_timer_set_repeat_count(_pending_timeout_timer, 1);
	}

	void stop_pending_timers()
	{
		if (_pending_spin_timer != nullptr) {
			lv_timer_delete(_pending_spin_timer);
			_pending_spin_timer = nullptr;
		}
		if (_pending_timeout_timer != nullptr) {
			lv_timer_delete(_pending_timeout_timer);
			_pending_timeout_timer = nullptr;
		}
	}

	static void press_event_cb(lv_event_t *e)
	{
		auto *self =
			static_cast<TileButton *>(lv_event_get_user_data(e));
		if (self == nullptr || self->_obj == nullptr ||
		    !self->_available || self->_pending) {
			return;
		}

		self->_is_pressed = true;
		self->update_appearance();
	}

	static void release_event_cb(lv_event_t *e)
	{
		auto *self =
			static_cast<TileButton *>(lv_event_get_user_data(e));
		if (self == nullptr || self->_obj == nullptr) {
			return;
		}

		self->_is_pressed = false;
		self->update_appearance();
	}

	static void click_event_cb(lv_event_t *e)
	{
		auto *self =
			static_cast<TileButton *>(lv_event_get_user_data(e));
		if (self == nullptr || !self->_available || self->_pending ||
		    !self->_click_callback) {
			return;
		}

		self->_click_callback();
	}

	static void async_apply_cb(void *user_data)
	{
		auto *self = static_cast<TileButton *>(user_data);
		if (self == nullptr) {
			return;
		}

		self->_apply_scheduled = false;
		self->apply_pending_state();
	}

	static void pending_spin_timer_cb(lv_timer_t *timer)
	{
		auto *self = static_cast<TileButton *>(
			lv_timer_get_user_data(timer));
		if (self == nullptr || !self->_pending ||
		    self->_pending_arc_obj == nullptr) {
			return;
		}

		self->_pending_rotation = static_cast<uint16_t>(
			(self->_pending_rotation + 18) % 360);
		lv_arc_set_rotation(self->_pending_arc_obj,
				    self->_pending_rotation);
	}

	static void pending_timeout_timer_cb(lv_timer_t *timer)
	{
		auto *self = static_cast<TileButton *>(
			lv_timer_get_user_data(timer));
		if (self == nullptr) {
			return;
		}

		self->_pending_timeout_timer = nullptr;
		self->set_pending(false);
	}
};

SettingsPage::SettingsPage()
	: Page(PageType::SETTINGS)
{
}

SettingsPage::~SettingsPage()
{
	destroy();
}

void SettingsPage::create(lv_obj_t *parent)
{
	if (_container != nullptr) {
		return;
	}

	_container = lv_obj_create(parent);
	lv_obj_set_size(_container, WIDTH, HEIGHT);
	lv_obj_set_style_bg_color(_container, lv_color_hex(0x44403B), 0);
	lv_obj_set_style_bg_opa(_container, LV_OPA_COVER, 0);
	lv_obj_set_style_radius(_container, CONTENT_RADIUS, 0);
	lv_obj_set_style_border_width(_container, 0, 0);
	lv_obj_set_style_pad_all(_container, 0, 0);
	lv_obj_set_style_clip_corner(_container, true, 0);
	lv_obj_set_scrollbar_mode(_container, LV_SCROLLBAR_MODE_OFF);
	lv_obj_clear_flag(_container, LV_OBJ_FLAG_SCROLLABLE);
	_background_obj =
		create_background_image_obj(_container, _background_image);

	lv_obj_t *content = lv_obj_create(_container);
	_main_tile_parent = content;
	lv_obj_set_pos(content, CONTENT_X, CONTENT_Y);
	lv_obj_set_size(content, CONTENT_WIDTH, CONTENT_HEIGHT);
	lv_obj_set_style_radius(content, 0, 0);
	lv_obj_set_style_bg_opa(content, LV_OPA_TRANSP, 0);
	lv_obj_set_style_border_width(content, 0, 0);
	lv_obj_set_style_pad_all(content, 0, 0);
	lv_obj_clear_flag(content, LV_OBJ_FLAG_CLICKABLE);
	lv_obj_clear_flag(content, LV_OBJ_FLAG_SCROLLABLE);

	create_tile_grid(content);
}

void SettingsPage::destroy()
{
	_screen_tile_on_main = false;
	_main_tile_parent = nullptr;
	_second_tile_parent = nullptr;
	_wifi_btn.reset();
	_mcp_btn.reset();
	_replay_btn.reset();
	_ncm_btn.reset();
	_agent_btn.reset();
	_ssh_btn.reset();
	_auto_rotation_btn.reset();
	_screen_btn.reset();
	_servo_btn.reset();
	_second_about_btn.reset();
	_apps_btn.reset();
	_tailscale_btn.reset();
	_network_btn.reset();
	_region_btn.reset();
	_background_obj = nullptr;

	Page::destroy();
}

void SettingsPage::set_background_image(const BackgroundImage *image)
{
	if (_container == nullptr) {
		_background_image = image;
		return;
	}
	lv_delete_if_valid(_background_obj);
	_background_image = image;
	_background_obj =
		create_background_image_obj(_container, _background_image);
}

void SettingsPage::set_tile_visible(SettingsTile tile, bool visible)
{
	TileButton *tile_button = get_tile(tile);
	if (tile_button == nullptr) {
		return;
	}

	tile_button->set_visible(visible);
	relayout_all_tiles();
}

void SettingsPage::set_tile_available(SettingsTile tile, bool available)
{
	TileButton *tile_button = get_tile(tile);
	if (tile_button == nullptr) {
		return;
	}

	tile_button->set_available(available);
}

void SettingsPage::set_tile_pending_timeout(SettingsTile tile,
					    uint32_t timeout_ms)
{
	TileButton *tile_button = get_tile(tile);
	if (tile_button == nullptr) {
		return;
	}

	tile_button->set_pending_timeout(timeout_ms);
}

ControlState *SettingsPage::get_wifi_btn() const
{
	return _wifi_btn.get();
}

ControlState *SettingsPage::get_mcp_btn() const
{
	return _mcp_btn.get();
}

ControlState *SettingsPage::get_replay_btn() const
{
	return _agent_btn.get();
}

ControlState *SettingsPage::get_ncm_btn() const
{
	return _ncm_btn.get();
}

ControlState *SettingsPage::get_ssh_btn() const
{
	return _ssh_btn.get();
}

ControlState *SettingsPage::get_auto_rotation_btn() const
{
	return _auto_rotation_btn.get();
}

ControlState *SettingsPage::get_screen_btn() const
{
	return _screen_btn.get();
}

ControlState *SettingsPage::get_second_about_btn() const
{
	return _second_about_btn.get();
}

ControlState *SettingsPage::get_tailscale_btn() const
{
	return _tailscale_btn.get();
}

ControlState *SettingsPage::get_network_btn() const
{
	return _network_btn.get();
}

ControlState *SettingsPage::get_region_btn() const
{
	return _region_btn.get();
}

void SettingsPage::create_second_page_tiles(lv_obj_t *parent)
{
	if (parent == nullptr || _second_about_btn != nullptr) {
		return;
	}
	_second_tile_parent = parent;

	auto create_tile = [](lv_obj_t *tile_parent, const lv_image_dsc_t *icon,
			      uint32_t active_color) {
		auto tile = std::make_unique<TileButton>(tile_parent, icon,
							 active_color);
		tile->create();
		return tile;
	};

	_second_about_btn = create_tile(parent, &settings_about, 0x1C1917);
	_screen_btn = create_tile(parent, &settings_screen, 0xFFB900);
#if !NANOKVM_GO_RELEASE_BUILD
	_servo_btn = create_tile(parent, &settings_screen, 0x2B7FFF);
#endif
	_screen_btn->on_click([this]() {
		if (on_screen_click) {
			on_screen_click();
		}
	});
#if !NANOKVM_GO_RELEASE_BUILD
	_servo_btn->on_click([this]() {
		if (on_servo_click) {
			on_servo_click();
		}
	});
#endif
	_second_about_btn->on_click([this]() {
		if (on_second_about_click) {
			on_second_about_click();
		}
	});
	_apps_btn = create_tile(parent, &settings_apps, 0x00A86B);
	_apps_btn->on_click([this]() {
		if (on_apps_click) {
			on_apps_click();
		}
	});
	_tailscale_btn = create_tile(parent, &settings_tailscale, 0x00838F);
	_tailscale_btn->on_click([this]() {
		if (on_tailscale_click) {
			on_tailscale_click();
		}
	});
	_tailscale_btn->set_active(false, ControlState::ApplyPolicy::IMMEDIATE);
	_tailscale_btn->set_visible(false);
	_network_btn = create_tile(parent, &settings_tailscale, 0x00838F);
	_network_btn->on_click([this]() {
		if (on_network_click) {
			on_network_click();
		}
	});
	_network_btn->set_active(false, ControlState::ApplyPolicy::IMMEDIATE);
	_region_btn = create_tile(parent, &settings_version_switch, 0x1C1917);
	_region_btn->on_click([this]() {
		if (on_region_click) {
			on_region_click();
		}
	});
	_region_btn->set_visible(false);
	relayout_all_tiles();
}

void SettingsPage::create_tile_grid(lv_obj_t *content)
{
	auto create_tile = [](lv_obj_t *parent, const lv_image_dsc_t *icon,
			      uint32_t active_color) {
		auto tile = std::make_unique<TileButton>(parent, icon,
							 active_color);
		tile->create();
		return tile;
	};

	_wifi_btn = create_tile(content, &img_settings_main_wifi, 0x2B7FFF);
	_wifi_btn->on_click([this]() {
		if (on_wifi_click) {
			on_wifi_click();
		}
	});

	_mcp_btn = create_tile(content, &settings_agent_btn, 0xFB2C36);
	_mcp_btn->on_click([this]() {
		if (on_mcp_click) {
			on_mcp_click();
		}
	});

	_agent_btn = create_tile(content, &settings_replay, 0x9810FA);
	_agent_btn->on_click([this]() {
		if (on_replay_click) {
			on_replay_click();
		}
	});

	_ncm_btn = create_tile(content, &settings_eth, 0x00C950);
	_ncm_btn->on_click([this]() {
		if (on_ncm_click) {
			on_ncm_click();
		}
	});

	_ssh_btn = create_tile(content, &settings_ssh, 0x14B8A6);
	_ssh_btn->on_click([this]() {
		if (on_ssh_click) {
			on_ssh_click();
		}
	});

	_auto_rotation_btn = create_tile(content, &settings_auto_180, 0xFF6900);
	_auto_rotation_btn->on_click([this]() {
		if (on_auto_rotation_click) {
			on_auto_rotation_click();
		}
	});
	relayout_main_tiles();
}

SettingsPage::TileButton *SettingsPage::get_tile(SettingsTile tile) const
{
	switch (tile) {
	case SettingsTile::WIFI:
		return _wifi_btn.get();
	case SettingsTile::MCP:
		return _mcp_btn.get();
	case SettingsTile::REPLAY:
		return _agent_btn.get();
	case SettingsTile::NCM:
		return _ncm_btn.get();
	case SettingsTile::SSH:
		return _ssh_btn.get();
	case SettingsTile::AUTO_ROTATION:
		return _auto_rotation_btn.get();
	case SettingsTile::SCREEN:
		return _screen_btn.get();
	case SettingsTile::SERVO:
		return _servo_btn.get();
	case SettingsTile::SECOND_ABOUT:
		return _second_about_btn.get();
	case SettingsTile::APPS:
		return _apps_btn.get();
	case SettingsTile::TAILSCALE:
		return _tailscale_btn.get();
	case SettingsTile::NETWORK:
		return _network_btn.get();
	case SettingsTile::REGION:
		return _region_btn.get();
	}

	return nullptr;
}

bool SettingsPage::tile_is_visible(SettingsTile tile) const
{
	const TileButton *tile_button = get_tile(tile);
	return tile_button != nullptr && tile_button->is_visible();
}

void SettingsPage::update_screen_tile_placement()
{
	if (_screen_btn == nullptr || _main_tile_parent == nullptr ||
	    _second_tile_parent == nullptr) {
		return;
	}

	const bool screen_on_main = !tile_is_visible(SettingsTile::REPLAY);
	if (screen_on_main == _screen_tile_on_main) {
		return;
	}

	_screen_btn->set_parent(screen_on_main ? _main_tile_parent :
						 _second_tile_parent);
	_screen_tile_on_main = screen_on_main;
}

void SettingsPage::relayout_all_tiles()
{
	update_screen_tile_placement();
	relayout_main_tiles();
	relayout_second_page_tiles();
}

void SettingsPage::relayout_main_tiles()
{
	const auto layout = settings_tile_layout::build_settings_tile_layout(
		tile_is_visible(SettingsTile::REPLAY));
	std::vector<TileButton *> tiles;
	tiles.reserve(layout.main.size());
	for (const settings_tile_layout::SettingsTile tile : layout.main) {
		tiles.push_back(get_tile(static_cast<SettingsTile>(tile)));
	}
	relayout_tiles(tiles.data(), tiles.size());
}

void SettingsPage::relayout_second_page_tiles()
{
	const auto layout = settings_tile_layout::build_settings_tile_layout(
		tile_is_visible(SettingsTile::REPLAY));
	std::vector<TileButton *> tiles;
	tiles.reserve(layout.second.size());
	for (const settings_tile_layout::SettingsTile tile : layout.second) {
		tiles.push_back(get_tile(static_cast<SettingsTile>(tile)));
	}
	relayout_tiles(tiles.data(), tiles.size());
}

void SettingsPage::relayout_tiles(TileButton *const *tiles, std::size_t count)
{
	std::size_t slot_index = 0;
	for (std::size_t i = 0; i < count && slot_index < TILE_SLOTS.size();
	     ++i) {
		TileButton *tile = tiles[i];
		if (tile == nullptr || !tile->is_visible()) {
			continue;
		}

		const TileSlot slot = TILE_SLOTS[slot_index];
		tile->set_position(slot.x, slot.y);
		++slot_index;
	}
}

}
