#pragma once

#include <cstdint>

#include <lvgl.h>

namespace ui
{

enum class PageType {
	MAIN,
	SETTINGS,
	SUBPAGE_WIFI,
	SUBPAGE_PANEL,
	SUBPAGE_ABOUT,
	SUBPAGE_MORE,
	SUBPAGE_SERVO,
	SUBPAGE_APPS,
	SUBPAGE_TAILSCALE,
	SUBPAGE_REGION,
	SUBPAGE_NETWORK,
	SUBPAGE_SETTINGS_STATE_TOGGLE,
	SUBPAGE_WIFI_CONFIG,
	SUBPAGE_WIFI_SSID,
	SUBPAGE_WIFI_CONNECT_METHOD,
	SUBPAGE_WIFI_QR,
	SUBPAGE_WIFI_CONNECT_RESULT
};

class Page {
    public:
	explicit Page(PageType type);
	virtual ~Page() = default;

	Page(const Page &) = delete;
	Page &operator=(const Page &) = delete;

	virtual void create(lv_obj_t *parent) = 0;
	virtual void destroy();

	virtual void on_enter()
	{
	}
	virtual void on_exit()
	{
	}
	virtual void on_resume()
	{
	}
	virtual void on_pause()
	{
	}

	bool is_created() const
	{
		return _container != nullptr;
	}
	PageType get_type() const
	{
		return _type;
	}
	lv_obj_t *get_container() const
	{
		return _container;
	}

    protected:
	PageType _type;
	lv_obj_t *_container = nullptr;
};

} // namespace ui
