#pragma once

#include <string>
#include <string_view>

namespace ui
{

struct WifiSsidDisplay {
	bool visible { false };
	std::string text;
};

// Produces the safe LTR display form for a scanned SSID. The caller retains the
// raw SSID for selection and connection; this value is display-only.
WifiSsidDisplay make_wifi_ssid_display(std::string_view raw_ssid);

} // namespace ui
