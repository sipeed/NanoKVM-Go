#include "nanokvm_runtime_internal.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <ctime>
#include <filesystem>
#include <fstream>
#include <iomanip>
#include <initializer_list>
#include <optional>
#include <sstream>
#include <string_view>
#include <utility>
#include <vector>

#include <arpa/inet.h>
#include <ifaddrs.h>
#include <net/if.h>
#include <netinet/in.h>
#include <sys/stat.h>
#include <unistd.h>

#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>

#include "lvgl_main/backlight_control.hpp"
#include "lvgl_main/device_number_parser.hpp"
#include "lvgl_main/hardware_version.hpp"
#include "lvgl_main/nanokvm_ip_selection.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"
#include "lvgl_main/package_version.hpp"
#include "lvgl_main/runtime_read_observation.hpp"
#include "lvgl_main/wifi_runtime_helpers.hpp"
#include "lvgl_main/wifi_script_parser.hpp"
#include "nanokvm_server_transport.hpp"
#include "systemd_dbus_client.hpp"

namespace lvgl_main
{
namespace detail
{

using json = nlohmann::json;

namespace
{

constexpr std::chrono::milliseconds kCommandStatusTimeout { 800 };
constexpr std::chrono::seconds kCommandActionTimeout { 5 };
constexpr std::string_view kWifiApGateway = kWifiApGatewayIp;
constexpr std::string_view kChipTypePath = "/proc/ax_proc/chip_type";
constexpr std::string_view kDeviceKeyPath = "/device_key";
constexpr std::string_view kHostnamePath = "/etc/hostname";
constexpr std::string_view kImageVersionPath = "/boot/ver";
constexpr std::string_view kEdidCurrentPath =
	"/etc/kvm/edid/current.json";
constexpr std::string_view kUptimePath = "/proc/uptime";
constexpr std::string_view kMeminfoPath = "/proc/meminfo";
constexpr std::string_view kThermalZonePath =
	"/sys/class/thermal/thermal_zone0/temp";

int64_t elapsed_ms(std::chrono::steady_clock::time_point start,
		   std::chrono::steady_clock::time_point end)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(end -
								     start)
		.count();
}

class ScopedIfaddrs final {
    public:
	explicit ScopedIfaddrs(ifaddrs *ptr)
		: _ptr(ptr)
	{
	}

	~ScopedIfaddrs()
	{
		if (_ptr != nullptr) {
			::freeifaddrs(_ptr);
		}
	}

	ScopedIfaddrs(const ScopedIfaddrs &) = delete;
	ScopedIfaddrs &operator=(const ScopedIfaddrs &) = delete;

	ifaddrs *get() const
	{
		return _ptr;
	}

    private:
	ifaddrs *_ptr { nullptr };
};
std::optional<WifiScriptStatus>
query_wifi_script_status(const NanokvmRuntimeConfig &config,
                         bool allow_nonzero_exit)
{
	if (!detail::wifi_script_available_quiet(config.wifi_script)) {
		return std::nullopt;
	}
	const auto started_at = std::chrono::steady_clock::now();
	const auto result = detail::run_command(
		{ config.wifi_script, "status" }, kCommandStatusTimeout,
		"wifi status");
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime WiFi script args='status' exit={} duration_ms={} timeout_ms={} timed_out={} output={}",
		result.exit_code, elapsed_ms(started_at, ended_at),
		kCommandStatusTimeout.count(), result.timed_out,
		detail::command_output_summary_for_log(result.output));
	if (result.timed_out || result.output.empty() ||
	    (!allow_nonzero_exit && result.exit_code != 0)) {
		return std::nullopt;
	}
	return parse_wifi_status_output(result.output);
}
std::optional<json>
http_request_json_timed(NanoKvmServerTransport &transport,
				std::string_view method, std::string_view path)
{
	const auto started_at = std::chrono::steady_clock::now();
	std::optional<json> body;
	if (method == "GET") {
		body = transport.get_json(NanoKvmServerTransportLane::Status, path);
	}
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime HTTP {} {} ok={} duration_ms={}",
		method, path, body.has_value(),
		elapsed_ms(started_at, ended_at));
	return body;
}
std::optional<std::string>
read_sysinfo_file_line(std::string_view path)
{
	std::ifstream input { std::string(path) };
	if (!input) {
		return std::nullopt;
	}
	std::string line;
	std::getline(input, line);
	return detail::trim(std::move(line));
}

} // namespace

bool is_wifi_interface_name(std::string_view ifname)
{
	return detail::starts_with(ifname, "wlan") ||
	       detail::starts_with(ifname, "wlx") ||
	       detail::starts_with(ifname, "wl");
}
std::optional<bool> observe_file_exists(std::string_view path)
{
	std::error_code ec;
	const bool exists =
		std::filesystem::exists(std::filesystem::path(std::string(path)), ec);
	if (ec) {
		SPDLOG_DEBUG("[runtime] NanoKVM runtime path observation failed path={} error={}",
			      path, ec.message());
		return std::nullopt;
	}
	return exists;
}

std::optional<std::string> read_kvmcomm_package_version()
{
	const auto result = detail::run_command(
		{ "dpkg-query", "-W", "-f=${Version}", "kvmcomm" },
		kCommandStatusTimeout, "kvmcomm package version");
	if (result.timed_out || result.exit_code != 0) {
		spdlog::warn(
			"NanoKVM runtime kvmcomm package version query failed exit={} timed_out={} output={}",
			result.exit_code, result.timed_out,
			detail::command_output_summary_for_log(result.output));
		return std::nullopt;
	}

	auto version = parse_deb_package_version(result.output);
	if (!version) {
		spdlog::warn(
			"NanoKVM runtime kvmcomm package version query returned empty output");
		return std::nullopt;
	}

	spdlog::info("[runtime] NanoKVM runtime kvmcomm package version: {}",
		     *version);
	return version;
}

std::optional<std::string> read_local_hardware_version()
{
	std::ifstream input { std::string(kChipTypePath) };
	if (!input) {
		return std::nullopt;
	}

	std::string chip_type;
	std::getline(input, chip_type, '\0');
	return hardware_version_from_chip_type(chip_type);
}

StateReadObservation<bool> systemd_is_active(std::string_view service)
{
	SystemdDbusUnitState dbus_state;
	std::string dbus_error;
	const auto dbus_result = query_systemd_unit_state(
		service, dbus_state, dbus_error);
	if (dbus_result != SystemdDbusResult::Success) {
		SPDLOG_DEBUG(
			"[runtime] systemd D-Bus active-state query inconclusive service={} error={}",
			service, dbus_error);
		return StateReadObservation<bool>::inconclusive();
	}
	return StateReadObservation<bool>::observed(
		systemd_dbus_unit_is_active(dbus_state));
}

bool systemd_enable(std::string_view service)
{
	std::string dbus_error;
	const auto dbus_result = set_systemd_unit_enabled_now(
		service, true, dbus_error);
	if (dbus_result != SystemdDbusResult::Success) {
		spdlog::warn(
			"NanoKVM runtime failed to enable systemd unit over D-Bus unit={} error={}",
			service, dbus_error);
		return false;
	}
	return true;
}

bool systemd_disable(std::string_view service)
{
	std::string dbus_error;
	const auto dbus_result = set_systemd_unit_enabled_now(
		service, false, dbus_error);
	if (dbus_result != SystemdDbusResult::Success) {
		spdlog::warn(
			"NanoKVM runtime failed to disable systemd unit over D-Bus unit={} error={}",
			service, dbus_error);
		return false;
	}
	return true;
}

StateReadObservation<bool> systemd_unit_exists(std::string_view unit)
{
	SystemdDbusUnitState dbus_state;
	std::string dbus_error;
	const auto dbus_result = query_systemd_unit_state(
		unit, dbus_state, dbus_error);
	if (dbus_result != SystemdDbusResult::Success) {
		SPDLOG_DEBUG(
			"[runtime] systemd D-Bus load-state query inconclusive unit={} error={}",
			unit, dbus_error);
		return StateReadObservation<bool>::inconclusive();
	}
	return StateReadObservation<bool>::observed(
		systemd_dbus_unit_exists(dbus_state));
}

bool collect_ip_addresses(RuntimeStatus &status)
{
	const auto ncm_enabled = observe_file_exists(kNcmBootFlagPath);
	const auto rndis_enabled = observe_file_exists(kRndisBootFlagPath);
	if (!ncm_enabled || !rndis_enabled) {
		return false;
	}
	std::string wlan0_ip;

	ifaddrs *raw = nullptr;
	if (::getifaddrs(&raw) != 0) {
		spdlog::warn("NanoKVM runtime getifaddrs failed: {}",
			     std::strerror(errno));
		return false;
	}
	ScopedIfaddrs addrs(raw);

	for (const auto *ifa = addrs.get(); ifa != nullptr;
	     ifa = ifa->ifa_next) {
		if (ifa->ifa_addr == nullptr ||
		    ifa->ifa_addr->sa_family != AF_INET) {
			continue;
		}
		if ((ifa->ifa_flags & IFF_UP) == 0U ||
		    (ifa->ifa_flags & IFF_LOOPBACK) != 0U) {
			continue;
		}

		std::array<char, INET_ADDRSTRLEN> address {};
		const auto *addr =
			reinterpret_cast<const sockaddr_in *>(ifa->ifa_addr);
		if (::inet_ntop(AF_INET, &addr->sin_addr, address.data(),
				static_cast<socklen_t>(address.size())) ==
		    nullptr) {
			continue;
		}

		const std::string ifname =
			ifa->ifa_name != nullptr ? ifa->ifa_name : "";
		const std::string ip = address.data();
		if (ifname == "wlan0" && wlan0_ip.empty()) {
			wlan0_ip = ip;
		}
		if (!is_station_wifi_ip(ip)) {
			continue;
		}

		if ((detail::starts_with(ifname, "eth") ||
		     detail::starts_with(ifname, "en")) &&
		    status.eth_ip.empty()) {
			status.eth_ip = ip;
		}
		if (is_wifi_interface_name(ifname) && status.wifi_ip.empty()) {
			status.wifi_ip = ip;
		}
		const UsbNetworkProtocol usb_protocol =
			classify_usb_network_interface(ifname, *ncm_enabled,
							       *rndis_enabled);
		if (usb_protocol == UsbNetworkProtocol::NCM &&
		    status.ncm_ip.empty()) {
			status.ncm_ip = ip;
		} else if (usb_protocol == UsbNetworkProtocol::RNDIS &&
			   status.rndis_ip.empty()) {
			status.rndis_ip = ip;
		}
	}

	status.main_ip_candidates =
		select_main_ip_candidates({ .ncm_enabled = *ncm_enabled,
						    .rndis_enabled = *rndis_enabled,
					    .ncm_ip = status.ncm_ip,
					    .rndis_ip = status.rndis_ip,
					    .wlan0_ip = wlan0_ip });
	status.active_ip = !status.main_ip_candidates.empty() ?
					   status.main_ip_candidates.front() :
					   std::string {};
	return true;
}

WifiInterfaceInfo collect_wifi_interfaces()
{
	WifiInterfaceInfo info;
	ifaddrs *raw = nullptr;
	if (::getifaddrs(&raw) != 0) {
		spdlog::warn("NanoKVM runtime getifaddrs failed: {}",
			     std::strerror(errno));
		return info;
	}
	info.observed = true;
	ScopedIfaddrs addrs(raw);

	for (const auto *ifa = addrs.get(); ifa != nullptr;
	     ifa = ifa->ifa_next) {
		if (ifa->ifa_name == nullptr ||
		    !is_wifi_interface_name(ifa->ifa_name)) {
			continue;
		}
		const std::string ifname = ifa->ifa_name;
		if (std::find(info.names.begin(), info.names.end(), ifname) ==
		    info.names.end()) {
			info.names.push_back(ifname);
		}
		if (ifa->ifa_addr == nullptr ||
		    ifa->ifa_addr->sa_family != AF_INET) {
			continue;
		}

		std::array<char, INET_ADDRSTRLEN> address {};
		const auto *addr =
			reinterpret_cast<const sockaddr_in *>(ifa->ifa_addr);
		if (::inet_ntop(AF_INET, &addr->sin_addr, address.data(),
				static_cast<socklen_t>(address.size())) ==
		    nullptr) {
			continue;
		}
		if (std::string_view(address.data()) == kWifiApGateway) {
			info.has_ap_gateway = true;
		}
	}

	return info;
}

void collect_streamer_status(NanoKvmServerTransport &transport,
			     RuntimeStatus &status)
{
	const auto body = transport.get_json(
		NanoKvmServerTransportLane::Metrics, "/api/streamer/local");
	if (!body) {
		return;
	}
	const auto data = detail::response_data(*body);
	if (!data || !data->is_object()) {
		return;
	}
	status.server_available = true;

	if (const auto *params = data->contains("params") ? &(*data)["params"] :
							    nullptr) {
		if (auto resolution = detail::json_string(*params, "resolution");
		    resolution && !resolution->empty()) {
			status.resolution = *resolution;
		}
	}

	if (!data->contains("streamer") || !(*data)["streamer"].is_object()) {
		return;
	}
	const auto &streamer = (*data)["streamer"];
	if (streamer.contains("source") && streamer["source"].is_object()) {
		const auto &source = streamer["source"];
		if (source.contains("resolution") &&
		    source["resolution"].is_object()) {
			const auto &res = source["resolution"];
			if (res.contains("width") && res.contains("height") &&
			    res["width"].is_number() &&
			    res["height"].is_number()) {
				const int width = res["width"].get<int>();
				const int height = res["height"].get<int>();
				if (width > 0 && height > 0) {
					status.resolution =
						std::to_string(width) + "x" +
						std::to_string(height);
				}
			}
		}
		if (auto fps = detail::json_number(source, "captured_fps");
		    fps && *fps > 0.0) {
			status.fps = *fps;
		}
	}
	if (!status.fps && streamer.contains("stream") &&
	    streamer["stream"].is_object()) {
		if (auto fps = detail::json_number(streamer["stream"], "queued_fps");
		    fps && *fps > 0.0) {
			status.fps = *fps;
		}
	}
	if (streamer.contains("h264") && streamer["h264"].is_object()) {
		if (auto bitrate =
			    detail::json_number(streamer["h264"], "real_bitrate");
		    bitrate && *bitrate >= 0.0) {
			status.h264_bitrate_kbps = static_cast<int>(*bitrate);
		}
	}
}

bool collect_hid_status(NanoKvmServerTransport &transport,
			RuntimeStatus &status)
{
	const auto body = transport.get_json(
		NanoKvmServerTransportLane::Status, "/api/hid/mode");
	if (!body) {
		return false;
	}
	const auto data = detail::response_data(*body);
	if (!data) {
		return false;
	}
	status.server_available = true;
	status.hid_supported = true;
	const auto mode = observe_required_http_field(
		detail::json_string(*data, "mode"));
	if (!mode.is_observed()) {
		return false;
	}
	status.hid_mode = mode.value();
	status.hid_enabled = mode.value() == "normal";
	return true;
}

bool collect_video_status(NanoKvmServerTransport &transport,
				  RuntimeStatus &status)
{
	const auto body = transport.get_json(
		NanoKvmServerTransportLane::Status, "/api/vm/hdmi/capture");
	if (!body) {
		return false;
	}
	const auto data = detail::response_data(*body);
	if (!data) {
		return false;
	}
	status.server_available = true;
	status.hdmi_input_supported = true;
	const auto enabled = observe_required_http_field(
		detail::json_bool(*data, "enabled"));
	if (!enabled.is_observed()) {
		return false;
	}
	status.hdmi_input_enabled = enabled.value();
	status.video_enabled = enabled.value();
	return true;
}

bool collect_wifi_status(const NanokvmRuntimeConfig &config,
			 NanoKvmServerTransport &transport,
			 RuntimeStatus &status,
			 bool station_ip_observed,
			 bool allow_nonzero_script_output)
{
	const auto started_at = std::chrono::steady_clock::now();
	const auto interfaces = collect_wifi_interfaces();
	bool primary_observed = interfaces.observed && interfaces.names.empty();
	bool api_reported_supported = false;
	bool api_reported_ap_mode = false;
	status.wifi_supported = !interfaces.names.empty();
	if (interfaces.has_ap_gateway) {
		status.wifi_ap_mode = true;
	}

	bool api_reported_ap_client_connected = false;
	bool api_reported_connected = false;

	const auto body =
		http_request_json_timed(transport, "GET", "/api/network/wifi");
	if (body) {
		const auto data = detail::response_data(*body);
		if (data && data->is_object()) {
			status.server_available = true;
			status.wifi_supported = true;
			if (auto supported = detail::json_bool(*data, "supported")) {
				status.wifi_supported = *supported;
				api_reported_supported = true;
			}
			if (auto ap_mode = detail::json_bool(*data, "apMode")) {
				status.wifi_ap_mode = *ap_mode;
				api_reported_ap_mode = true;
			}
			if (auto connected = detail::json_bool(*data, "connected")) {
				status.wifi_connected = *connected;
				api_reported_connected = true;
			}
			if (auto client_connected =
				    detail::json_bool(*data, "apClientConnected")) {
				status.wifi_ap_client_connected =
					*client_connected;
				api_reported_ap_client_connected = true;
			}
			detail::update_wifi_ap_credentials_from_api(*data);
		}
	}

	const auto script_status = query_wifi_script_status(
		config, allow_nonzero_script_output);
	if (script_status) {
		switch (script_status->kind) {
		case WifiScriptStatusKind::Connected:
				primary_observed = true;
			status.wifi_supported = true;
			status.wifi_connected = true;
			status.wifi_ap_mode = false;
			status.wifi_ssid = script_status->ssid;
			break;
		case WifiScriptStatusKind::Ap:
				primary_observed = true;
			status.wifi_supported = true;
			status.wifi_connected = false;
			status.wifi_ap_mode = true;
			status.wifi_ssid = script_status->ssid;
			break;
		case WifiScriptStatusKind::Disconnected:
				primary_observed = true;
			status.wifi_supported = true;
			status.wifi_connected = false;
			status.wifi_ap_mode = false;
			break;
		case WifiScriptStatusKind::NoInterface:
				primary_observed = true;
			status.wifi_supported = false;
			status.wifi_connected = false;
			status.wifi_ap_mode = false;
			status.wifi_error = "No WiFi";
			break;
		case WifiScriptStatusKind::Unknown:
			break;
		}
	}
	if (api_reported_supported && api_reported_ap_mode &&
	    api_reported_connected &&
	    (!status.wifi_ap_mode || api_reported_ap_client_connected)) {
		primary_observed = true;
	}

	bool ap_client_observed = !status.wifi_ap_mode ||
					  api_reported_ap_client_connected;
	if (status.wifi_ap_mode && !api_reported_ap_client_connected) {
		if (auto count = detail::query_wifi_device_count(
			    config, allow_nonzero_script_output)) {
			status.wifi_ap_client_connected = *count > 0;
			ap_client_observed = true;
		} else if (auto client_connected =
				       detail::hostapd_ap_client_connected(
					       interfaces.names)) {
			status.wifi_ap_client_connected = *client_connected;
			ap_client_observed = true;
		}
	}
	if (status.wifi_ap_mode && !api_reported_connected) {
		status.wifi_connected = false;
	}
	const bool requires_station_ip = status.wifi_connected;
	status.wifi_connected = status.wifi_connected &&
					is_station_wifi_ip(status.wifi_ip);

	const auto &credentials = detail::cached_wifi_ap_credentials();
	const auto qr_payload = detail::make_wifi_ap_qr_payload(credentials);
	const auto web_url = detail::make_wifi_web_url(credentials);
	status.wifi_qr_payload = qr_payload;
	status.wifi_web_url = web_url;
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime WiFi collect end duration_ms={} supported={} connected={} ap_mode={} ap_client={} ssid='{}' error='{}'",
		elapsed_ms(started_at, ended_at), status.wifi_supported,
		status.wifi_connected, status.wifi_ap_mode,
		status.wifi_ap_client_connected, status.wifi_ssid,
		status.wifi_error);
	return primary_observed &&
	       (!requires_station_ip || station_ip_observed) &&
	       ap_client_observed;
}

bool collect_hardware_version_status(NanoKvmServerTransport &transport,
					     RuntimeStatus &status)
{
	bool observed = false;
	if (auto hardware_version = read_local_hardware_version()) {
		status.hw_version = *hardware_version;
		observed = true;
	}

	const auto hardware = transport.get_json(
		NanoKvmServerTransportLane::Status, "/api/vm/hardware");
	if (hardware) {
		const auto data = detail::response_data(*hardware);
		if (data) {
			status.server_available = true;
		}
	}
	return observed;
}

std::optional<std::string> read_device_key()
{
	return read_sysinfo_file_line(kDeviceKeyPath);
}

std::optional<std::string> read_hostname()
{
	if (auto value = read_sysinfo_file_line(kHostnamePath)) {
		return value;
	}
	char buf[256] {};
	if (::gethostname(buf, sizeof(buf)) == 0) {
		auto host = detail::trim(std::string(buf));
		if (!host.empty()) {
			return host;
		}
	}
	return std::nullopt;
}

std::optional<std::string> read_image_version()
{
	auto raw = read_sysinfo_file_line(kImageVersionPath);
	if (!raw) {
		return std::nullopt;
	}
	const auto pos = raw->rfind('-');
	auto version = (pos != std::string::npos && pos + 1 < raw->size())
			       ? raw->substr(pos + 1) :
			       *raw;
	return detail::trim(std::move(version));
}

std::optional<std::string> read_edid_name()
{
	std::ifstream input { std::string(kEdidCurrentPath) };
	if (!input) {
		return std::nullopt;
	}
	std::string content((std::istreambuf_iterator<char>(input)),
			     std::istreambuf_iterator<char>());
	try {
		const auto doc = json::parse(content);
		auto name = doc.value("name", std::string {});
		return detail::trim(std::move(name));
	} catch (const std::exception &e) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM runtime EDID parse failed: {}",
			e.what());
		return std::nullopt;
	}
}

std::optional<std::string> read_mac_address()
{
	for (const char *iface : { "wlan0", "eth0" }) {
		const std::string path =
			std::string("/sys/class/net/") + iface + "/address";
		if (auto value = read_sysinfo_file_line(path)) {
			if (!value->empty()) {
				return value;
			}
		}
	}
	return std::nullopt;
}

std::optional<std::string> read_device_number()
{
	const auto result = detail::run_command(
		{ "ax_efuse_tool", "--str-get" }, kCommandStatusTimeout,
		"device number");
	if (result.timed_out || result.exit_code != 0) {
		return std::nullopt;
	}
	return parse_device_number_output(result.output);
}

std::optional<long> parse_meminfo_kib(const std::string &line)
{
	std::istringstream ls(line);
	std::string key;
	long value = 0;
	if (!(ls >> key >> value)) {
		return std::nullopt;
	}
	return value;
}

bool collect_temperature(RuntimeStatus &status)
{
	const auto raw = read_sysinfo_file_line(kThermalZonePath);
	if (!raw || raw->empty()) {
		return false;
	}
	long millidegrees = 0;
	try {
		millidegrees = std::stol(*raw);
	} catch (...) {
		return false;
	}
	const double celsius = static_cast<double>(millidegrees) / 1000.0;
	std::ostringstream out;
	out << std::fixed << std::setprecision(1) << celsius << "\u00B0C";
	status.temperature_text = out.str();
	return true;
}

bool collect_memory(RuntimeStatus &status)
{
	std::ifstream input { std::string(kMeminfoPath) };
	if (!input) {
		return false;
	}
	std::optional<long> mem_total_kib;
	std::optional<long> mem_avail_kib;
	std::string line;
	while (std::getline(input, line)) {
		if (mem_total_kib && mem_avail_kib) {
			break;
		}
		if (line.rfind("MemTotal:", 0) == 0) {
			mem_total_kib = parse_meminfo_kib(line);
		} else if (line.rfind("MemAvailable:", 0) == 0) {
			mem_avail_kib = parse_meminfo_kib(line);
		}
	}
	if (!mem_total_kib || !mem_avail_kib || *mem_total_kib <= 0) {
		return false;
	}
	const long used_kib =
		std::max(0L, *mem_total_kib - *mem_avail_kib);
	const long total_mib = (*mem_total_kib + 512) / 1024;
	const long used_mib = (used_kib + 512) / 1024;
	status.memory_text = std::to_string(used_mib) + "/" +
			      std::to_string(total_mib) + " MiB";
	return true;
}

bool collect_uptime(RuntimeStatus &status)
{
	std::ifstream input { std::string(kUptimePath) };
	if (!input) {
		return false;
	}
	double seconds = 0.0;
	if (!(input >> seconds) || seconds < 0.0) {
		return false;
	}
	const auto total = static_cast<long long>(seconds);
	const long long days = total / 86400;
	const long long hours = (total % 86400) / 3600;
	const long long minutes = (total % 3600) / 60;
	const long long secs = total % 60;
	std::ostringstream out;
	if (days > 0) {
		out << days << "d " << std::setw(2) << std::setfill('0')
		    << hours << ":" << std::setw(2) << minutes << ":"
		    << std::setw(2) << secs;
	} else {
		out << std::setw(2) << std::setfill('0') << hours << ":"
		    << std::setw(2) << minutes << ":" << std::setw(2)
		    << secs;
	}
	status.uptime_text = out.str();
	return true;
}

bool collect_local_time(RuntimeStatus &status)
{
	const std::time_t now = std::time(nullptr);
	if (now == static_cast<std::time_t>(-1)) {
		return false;
	}
	std::tm local {};
	if (::localtime_r(&now, &local) == nullptr) {
		return false;
	}
	char buf[32] {};
	std::strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M:%S", &local);
	status.local_time_text = buf;
	return true;
}

bool collect_storage(RuntimeStatus &status)
{
	const auto result = detail::run_command(
		{ "df", "-B1", "/" }, kCommandStatusTimeout, "storage");
	if (result.timed_out || result.exit_code != 0) {
		return false;
	}
	std::istringstream lines(result.output);
	std::string header;
	if (!std::getline(lines, header)) {
		return false;
	}
	std::string row;
	if (!std::getline(lines, row)) {
		return false;
	}
	std::istringstream data(row);
	std::string fs;
	long long total_bytes = 0;
	long long used_bytes = 0;
	long long avail_bytes = 0;
	long long use_pct = 0;
	if (!(data >> fs >> total_bytes >> used_bytes >> avail_bytes >>
	      use_pct)) {
		return false;
	}
	(void)avail_bytes;
	(void)use_pct;
	if (total_bytes <= 0 || used_bytes < 0) {
		return false;
	}
	const double gb = 1024.0 * 1024.0 * 1024.0;
	const double total_gb = static_cast<double>(total_bytes) / gb;
	const double used_gb = static_cast<double>(used_bytes) / gb;
	std::ostringstream out;
	out << std::fixed << std::setprecision(1) << used_gb << "/"
	    << std::setprecision(1) << total_gb << " GB";
	status.storage_text = out.str();
	return true;
}

bool collect_wifi_signal(RuntimeStatus &status, bool wifi_connected)
{
	if (!wifi_connected) {
		status.wifi_signal.clear();
		return true;
	}
	const auto result = detail::run_command(
		{ "iw", "dev", "wlan0", "link" }, kCommandStatusTimeout,
		"wifi signal");
	if (result.timed_out || result.exit_code != 0) {
		return false;
	}
	std::istringstream lines(result.output);
	std::string line;
	while (std::getline(lines, line)) {
		const auto pos = line.find("signal:");
		if (pos == std::string::npos) {
			continue;
		}
		std::istringstream ls(line.substr(pos + 7));
		int dbm = 0;
		std::string unit;
		if (ls >> dbm >> unit) {
			status.wifi_signal = std::to_string(dbm) + " dBm";
			return true;
		}
	}
	return false;
}

bool collect_systemd_status(RuntimeStatus &status,
			    bool preserve_last_known_good)
{
	if (!preserve_last_known_good) {
		const auto observed_true =
			[](const StateReadObservation<bool> &observation) {
				return observation.is_observed() && observation.value();
			};
		status.ssh_enabled =
			observed_true(systemd_is_active("ssh.service")) ||
			observed_true(systemd_is_active("ssh.socket"));
		status.ssh_supported =
			status.ssh_enabled ||
			observed_true(systemd_unit_exists("ssh.service")) ||
			observed_true(systemd_unit_exists("ssh.socket"));
		return true;
	}

	const auto service_active = systemd_is_active("ssh.service");
	if (!service_active.is_observed()) {
		return false;
	}
	if (service_active.value()) {
		status.ssh_enabled = true;
		status.ssh_supported = true;
		return true;
	}

	const auto socket_active = systemd_is_active("ssh.socket");
	if (!socket_active.is_observed()) {
		return false;
	}
	if (socket_active.value()) {
		status.ssh_enabled = true;
		status.ssh_supported = true;
		return true;
	}

	const auto service_exists = systemd_unit_exists("ssh.service");
	if (!service_exists.is_observed()) {
		return false;
	}
	if (service_exists.value()) {
		status.ssh_enabled = false;
		status.ssh_supported = true;
		return true;
	}

	const auto socket_exists = systemd_unit_exists("ssh.socket");
	if (!socket_exists.is_observed()) {
		return false;
	}
	status.ssh_enabled = false;
	status.ssh_supported = socket_exists.value();
	return true;
}

bool collect_ncm_status(const NcmBootMarkerPaths &paths,
			    RuntimeStatus &status)
{
	const NcmObservation observation = observe_ncm_boot_markers(paths);
	const auto value = ncm_observation_enabled(observation);
	if (!value) {
		SPDLOG_DEBUG(
			"[runtime] NanoKVM NCM boot marker observation inconclusive during refresh; retaining last known state");
		return false;
	}
	status.ncm_enabled = *value;
	status.ncm_supported = true;
	return true;
}

bool collect_backlight_status(const NanokvmRuntimeConfig &config,
			      int fallback_percent, bool screen_sleeping,
			      RuntimeStatus &status)
{
	const auto state = backlight::read_backlight_state(
		config.backlight_root, fallback_percent,
		config.backlight_invert);
	if (!state.available) {
		return false;
	}
	status.screen_on = !screen_sleeping && state.screen_on;
	status.backlight_percent = state.percent;
	return true;
}

} // namespace detail
} // namespace lvgl_main
