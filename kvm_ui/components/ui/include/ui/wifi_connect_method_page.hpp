#pragma once

#include <functional>

#include "ui/subpage_base.hpp"

namespace ui
{

class WifiConnectMethodPage : public SubPageBase {
    public:
	WifiConnectMethodPage();
	~WifiConnectMethodPage() override = default;

	void destroy() override;

	std::function<void()> on_qr_click;
	std::function<void()> on_key_click;

    private:
	void build_content() override;
};

} // namespace ui
