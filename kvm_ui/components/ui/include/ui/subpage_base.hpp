#pragma once

#include <functional>
#include <memory>
#include <string>
#include <vector>

#include "ui/bcona.hpp"
#include "ui/component.hpp"
#include "ui/page.hpp"

namespace ui
{

class SubPageBase : public Page {
    public:
	explicit SubPageBase(PageType type, const char *title,
			     uint16_t row_gap = 12, uint16_t top_padding = 12);
	~SubPageBase() override = default;

	void create(lv_obj_t *parent) final;
	void destroy() override;

	std::function<void()> on_back_click;

    protected:
	static constexpr uint16_t WIDTH = 270;
	static constexpr uint16_t HEIGHT = 240;
	static constexpr uint16_t LEFT_MARGIN = 13;
	static constexpr uint32_t PAGE_BG_COLOR = 0x000000;

	std::vector<std::unique_ptr<Component>> _components;
	lv_obj_t *_scroll_content = nullptr;
	BConA *_back_btn = nullptr;

	virtual void build_content() = 0;
	void set_title(const char *title);
	void set_back_navigation_available(bool available);
	BConA *title_button() const
	{
		return _back_btn;
	}
	virtual void on_subpage_destroy()
	{
	}

    private:
	std::string _title;
	uint16_t _row_gap = 12;
	uint16_t _top_padding = 12;
	bool _back_navigation_available = true;
};

} // namespace ui
