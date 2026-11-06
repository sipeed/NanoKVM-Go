#include "ui/wifi_connect_method_page.hpp"

#include <memory>

extern "C" {
LV_IMAGE_DECLARE(settings_wifi_qrcode);
LV_IMAGE_DECLARE(settings_wifi_password);
}

namespace ui
{

WifiConnectMethodPage::WifiConnectMethodPage()
	: SubPageBase(PageType::SUBPAGE_WIFI_CONNECT_METHOD, "Via")
{
}

void WifiConnectMethodPage::destroy()
{
	on_qr_click = nullptr;
	on_key_click = nullptr;
	on_back_click = nullptr;
	SubPageBase::destroy();
}

void WifiConnectMethodPage::build_content()
{
	auto password = std::make_unique<BConA>(_scroll_content, "PASSWD",
						BConA::COLOR_DEFAULT, true);
	password->create();
	password->set_left_image(&settings_wifi_password);
	password->on_click([this](BConA *) {
		if (on_key_click) {
			on_key_click();
		}
	});
	_components.push_back(std::move(password));

	auto qrcode = std::make_unique<BConA>(_scroll_content, "QRCODE",
					      BConA::COLOR_DEFAULT, true);
	qrcode->create();
	qrcode->set_left_image(&settings_wifi_qrcode);
	qrcode->on_click([this](BConA *) {
		if (on_qr_click) {
			on_qr_click();
		}
	});
	_components.push_back(std::move(qrcode));
}

} // namespace ui
