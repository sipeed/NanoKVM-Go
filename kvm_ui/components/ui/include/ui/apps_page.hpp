#pragma once

#include <functional>
#include <string>
#include <vector>

#include "ui/bcona.hpp"
#include "ui/subpage_base.hpp"

namespace ui
{

struct AppEntry {
	std::string name;
	std::string path;
	bool operator==(const AppEntry &) const = default;
};

class AppsPage : public SubPageBase {
    public:
	AppsPage();
	~AppsPage() override = default;

	void destroy() override;

	// Updates the live page when already created.
	void set_entries(std::vector<AppEntry> entries);

	std::function<void(const std::string &)> on_app_launch;

    private:
	static constexpr uint32_t APP_COLOR = 0x00A86B;
	static constexpr uint32_t SCROLLBAR_COLOR = 0xA8A29E;
	static constexpr uint8_t SCROLLBAR_WIDTH = 5;
	static constexpr uint8_t SCROLLBAR_EDGE_PADDING = 8;
	static constexpr uint8_t SCROLLBAR_VERTICAL_PADDING = 16;
	static constexpr uint16_t APP_NAME_WIDTH =
		BConA::WIDTH - 2 * BConA::PADDING_LEFT - BConA::ICON_SIZE -
		BConA::PADDING_COLUMN;

	std::vector<AppEntry> _entries;

	void configure_scrollbar();
	void build_content() override;
	void on_subpage_destroy() override;
};

} // namespace ui
