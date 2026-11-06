#include "nanokvm_runtime_internal.hpp"

#include <array>
#include <chrono>
#include <cstdint>
#include <cerrno>
#include <cstring>
#include <filesystem>
#include <initializer_list>
#include <optional>
#include <random>
#include <string_view>
#include <utility>
#include <vector>

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>

#include "lvgl_main/wifi_runtime_helpers.hpp"
#include "lvgl_main/wifi_script_parser.hpp"

namespace lvgl_main
{
namespace detail
{

using json = nlohmann::json;

namespace
{

constexpr std::chrono::milliseconds kCommandStatusTimeout { 800 };
constexpr std::chrono::seconds kCommandActionTimeout { 5 };
constexpr std::chrono::seconds kWifiConnectCommandTimeout { 15 };
constexpr std::chrono::seconds kWifiApCommandTimeout { 10 };
constexpr std::chrono::seconds kWifiScanCommandTimeout { 10 };
constexpr std::chrono::seconds kWifiSavedCommandTimeout { 5 };
constexpr std::string_view kHdmiEnablePath = "/tmp/kvm/hdmi_enable";
constexpr std::string_view kHidEnablePath = "/tmp/kvm/hid_enable";
constexpr std::string_view kWifiApPassPath = "/tmp/ap.pass";
constexpr std::string_view kWifiConfigPath = "/tmp/wifi_config";
constexpr std::string_view kWifiApGateway = kWifiApGatewayIp;

int64_t elapsed_ms(std::chrono::steady_clock::time_point start,
			   std::chrono::steady_clock::time_point end)
{
	return std::chrono::duration_cast<std::chrono::milliseconds>(end - start)
		.count();
}

} // namespace

bool wifi_script_available_quiet(std::string_view script_path)
{
	const std::string path(script_path);
	struct stat st {};
	return ::stat(path.c_str(), &st) == 0 && S_ISREG(st.st_mode) &&
	       ::access(path.c_str(), X_OK) == 0;
}

std::optional<int> query_wifi_device_count(const NanokvmRuntimeConfig &config,
                                           bool allow_nonzero_exit)
{
	if (!wifi_script_available_quiet(config.wifi_script)) {
		return std::nullopt;
	}
	const auto started_at = std::chrono::steady_clock::now();
	const auto result = detail::run_command(
		{ config.wifi_script, "device_count" }, kCommandStatusTimeout,
		"wifi device_count");
	const auto ended_at = std::chrono::steady_clock::now();
	spdlog::info(
		"[runtime] NanoKVM runtime WiFi script args='device_count' exit={} duration_ms={} timeout_ms={} timed_out={} output={}",
		result.exit_code, elapsed_ms(started_at, ended_at),
		kCommandStatusTimeout.count(), result.timed_out,
		detail::command_output_summary_for_log(result.output));
	if (result.timed_out || result.output.empty() ||
	    (!allow_nonzero_exit && result.exit_code != 0)) {
		return std::nullopt;
	}
	return parse_wifi_device_count_output(result.output);
}

detail::CommandResult run_wifi_script(const NanokvmRuntimeConfig &config,
				      const std::vector<std::string> &args)
{
	if (!validate_wifi_script(config.wifi_script)) {
		return { .exit_code = -1,
			 .output = "ERROR: SCRIPT_UNAVAILABLE" };
	}
	const auto timeout = [&args] {
		if (args.empty()) {
			return std::chrono::milliseconds(kCommandActionTimeout);
		}
		if (args.front() == "status" || args.front() == "boot_enabled" ||
		    args.front() == "device_count" || args.front() == "ap_ip" ||
		    args.front() == "signal") {
			return kCommandStatusTimeout;
		}
		if (args.front() == "scan") {
			return std::chrono::milliseconds(
				kWifiScanCommandTimeout);
		}
		if (args.front() == "saved") {
			return std::chrono::milliseconds(
				kWifiSavedCommandTimeout);
		}
		if (args.front() == "connect" || args.front() == "enterprise" ||
		    args.front() == "on") {
			return std::chrono::milliseconds(
				kWifiConnectCommandTimeout);
		}
		if (args.front() == "ap") {
			return std::chrono::milliseconds(kWifiApCommandTimeout);
		}
		return std::chrono::milliseconds(kCommandActionTimeout);
	}();
	const std::string command_summary =
		"wifi " + wifi_script_args_for_log(args);
	const auto started_at = std::chrono::steady_clock::now();
	const auto result = detail::run_command(
		make_wifi_script_argv(config, args), timeout, command_summary);
	const auto ended_at = std::chrono::steady_clock::now();
	const std::string_view secret = wifi_script_secret_for_log(args);
	spdlog::info(
		"[runtime] NanoKVM runtime WiFi script args='{}' exit={} duration_ms={} timeout_ms={} timed_out={} output={}",
		wifi_script_args_for_log(args), result.exit_code,
		elapsed_ms(started_at, ended_at), timeout.count(),
		result.timed_out,
		detail::command_output_summary_for_log(result.output, secret));
	return result;
}

bool query_wifi_boot_enabled(const NanokvmRuntimeConfig &config)
{
	const auto result = run_wifi_script(config, { "boot_enabled" });
	if (result.timed_out || result.exit_code != 0) {
		spdlog::warn(
			"[runtime] WiFi boot-enabled query failed closed: exit={} timed_out={}",
			result.exit_code, result.timed_out);
		return false;
	}

	const auto enabled = parse_wifi_boot_enabled_output(result.output);
	if (!enabled.has_value()) {
		spdlog::warn("[runtime] WiFi boot-enabled query returned malformed output");
		return false;
	}
	return *enabled;
}

bool write_flag_file(const std::filesystem::path &path)
{
	std::error_code ec;
	const auto parent = path.parent_path();
	if (!parent.empty()) {
		std::filesystem::create_directories(parent, ec);
		if (ec) {
			spdlog::warn(
				"Failed to create flag file directory {}: {}",
				parent.string(), ec.message());
			return false;
		}
	}

	const int fd =
		::open(path.c_str(),
		       O_WRONLY | O_CREAT | O_TRUNC | O_CLOEXEC | O_NOFOLLOW,
		       S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH);
	if (fd < 0) {
		spdlog::warn("Failed to create runtime switch file {}: {}",
			     path.string(), std::strerror(errno));
		return false;
	}

	detail::ScopedFd scoped_fd(fd);
	constexpr std::string_view payload = "1\n";
	const ssize_t written =
		::write(scoped_fd.get(), payload.data(), payload.size());
	if (written != static_cast<ssize_t>(payload.size())) {
		spdlog::warn("Failed to write runtime switch file {}: {}",
			     path.string(),
			     written < 0 ? std::strerror(errno) :
					   "short write");
		return false;
	}
	return true;
}

void write_wifi_ap_runtime_files(std::string_view password)
{
	if (!detail::write_text_file_secure(
		    std::filesystem::path(std::string(kWifiApPassPath)),
		    password, S_IRUSR | S_IWUSR)) {
		return;
	}

	spdlog::info("[runtime] NanoKVM runtime wrote AP key file {}",
		     kWifiApPassPath);
	if (write_flag_file(
		    std::filesystem::path(std::string(kWifiConfigPath)))) {
		spdlog::info(
			"[runtime] NanoKVM runtime wrote WiFi config marker {}",
			kWifiConfigPath);
	}
}

bool remove_flag_file(const std::filesystem::path &path)
{
	std::error_code ec;
	const bool removed = std::filesystem::remove(path, ec);
	if (!ec) {
		(void)removed;
		return true;
	}
	if (ec == std::errc::no_such_file_or_directory) {
		return true;
	}
	spdlog::warn("Failed to remove runtime switch file {}: {}",
		     path.string(), ec.message());
	return false;
}

bool set_flag_file_enabled(std::string_view path, bool enabled)
{
	const std::filesystem::path flag_path { std::string(path) };
	return enabled ? write_flag_file(flag_path) :
			 remove_flag_file(flag_path);
}

std::pair<std::string, std::string> make_wifi_ap_credentials()
{
	std::array<char, 64> host {};
	std::string suffix = "KVM";
	if (::gethostname(host.data(), host.size()) == 0) {
		host.back() = '\0';
		std::string name = detail::trim(host.data());
		if (!name.empty()) {
			const auto dash = name.rfind('-');
			suffix = dash == std::string::npos ?
					 name :
					 name.substr(dash + 1U);
		}
	}

	std::string ssid = "NanoKVM-" + suffix;
	std::random_device random_device;
	std::array<std::mt19937::result_type, 8U> seed_data {};
	for (auto &value : seed_data) {
		value = random_device();
	}
	std::seed_seq seed(seed_data.begin(), seed_data.end());
	std::mt19937 generator(seed);
	return { ssid, make_wifi_ap_password(generator) };
}

void ensure_wifi_ap_credentials_complete(WifiApCredentials &credentials)
{
	if (!credentials.ssid.empty() && !credentials.password.empty() &&
	    !credentials.ap_ip.empty()) {
		return;
	}

	const auto generated = make_wifi_ap_credentials();
	if (credentials.ssid.empty()) {
		credentials.ssid = generated.first;
	}
	if (credentials.password.empty()) {
		credentials.password = generated.second;
	}
	if (credentials.ap_ip.empty()) {
		credentials.ap_ip = std::string(kWifiApGateway);
	}
}

WifiApCredentials &cached_wifi_ap_credentials()
{
	static WifiApCredentials credentials = [] {
		const auto [ssid, password] = make_wifi_ap_credentials();
		return WifiApCredentials { .ssid = ssid,
					   .password = password,
					   .ap_ip = std::string(kWifiApGateway),
					   .confirmed = false };
	}();
	return credentials;
}

namespace
{

std::optional<std::string>
json_nested_first_string(const json &value,
			 std::initializer_list<std::string_view> object_keys,
			 std::initializer_list<std::string_view> value_keys)
{
	if (!value.is_object()) {
		return std::nullopt;
	}
	for (std::string_view object_key : object_keys) {
		const auto it = value.find(std::string(object_key));
		if (it != value.end() && it->is_object()) {
			if (auto nested = detail::json_first_string(*it, value_keys)) {
				return nested;
			}
		}
	}
	return std::nullopt;
}

} // namespace

void update_wifi_ap_credentials_from_api(const json &data)
{
	const std::initializer_list<std::string_view> ap_objects { "ap",
								   "apConfig",
								   "apInfo" };
	auto &credentials = cached_wifi_ap_credentials();
	auto ap_ip = detail::json_first_string(
		data, { "apIp", "apIP", "ap_ip", "apGateway", "ap_gateway" });
	if (!ap_ip) {
		ap_ip = json_nested_first_string(
			data, ap_objects, { "ip", "gateway", "apIp", "ap_ip" });
	}
	if (ap_ip) {
		credentials.ap_ip = *ap_ip;
	}

	auto ssid = detail::json_first_string(
		data, { "apSsid", "apSSID", "ap_ssid" });
	if (!ssid) {
		ssid = json_nested_first_string(
			data, ap_objects, { "ssid", "apSsid", "ap_ssid" });
	}
	auto password = detail::json_first_string(data, { "apPassword", "apPasswd",
						  "ap_password", "ap_passwd" });
	if (!password) {
		password = json_nested_first_string(
			data, ap_objects,
			{ "password", "passwd", "apPassword", "ap_password" });
	}
	if (ssid) {
		credentials.ssid = *ssid;
	}
	if (password) {
		credentials.password = *password;
		credentials.confirmed = true;
	}
}

std::string escape_wifi_qr_field(std::string_view value)
{
	std::string escaped;
	escaped.reserve(value.size());
	for (char ch : value) {
		if (ch == '\\' || ch == ';' || ch == ',' || ch == ':') {
			escaped += '\\';
		}
		escaped += ch;
	}
	return escaped;
}

std::string make_wifi_ap_qr_payload(const WifiApCredentials &credentials)
{
	return "WIFI:T:WPA;S:" + escape_wifi_qr_field(credentials.ssid) +
	       ";P:" + escape_wifi_qr_field(credentials.password) + ";;";
}

std::string make_wifi_web_url(const WifiApCredentials &credentials)
{
	return make_wifi_ap_web_url(credentials.ap_ip, credentials.password);
}

bool hostapd_sta_list_has_client(std::string_view output)
{
	std::size_t start = 0;
	while (start <= output.size()) {
		const std::size_t end = output.find('\n', start);
		const std::string line = detail::trim(std::string(
			output.substr(start, end == std::string_view::npos ?
						     std::string_view::npos :
						     end - start)));
		if (!line.empty() && line != "FAIL" &&
		    line.find(':') != std::string::npos) {
			return true;
		}
		if (end == std::string_view::npos) {
			break;
		}
		start = end + 1U;
	}
	return false;
}

std::optional<bool>
hostapd_ap_client_connected(const std::vector<std::string> &interface_names)
{
	bool queried = false;
	for (const auto &ifname : interface_names) {
		if (!detail::is_wifi_interface_name(ifname)) {
			continue;
		}
		std::string output;
		if (!detail::run_output_success({ "hostapd_cli", "-i", ifname,
					  "list_sta" },
					output, kCommandStatusTimeout,
					"hostapd_cli list_sta " + ifname)) {
			continue;
		}
		queried = true;
		if (hostapd_sta_list_has_client(output)) {
			return true;
		}
	}
	if (queried) {
		return false;
	}
	return std::nullopt;
}

bool check_wifi_ap_client()
{
	const auto wifi_ifaces = detail::collect_wifi_interfaces();
	for (const auto &name : wifi_ifaces.names) {
		if (!detail::is_wifi_interface_name(name)) {
			continue;
		}
		std::string output;
		if (detail::run_output_success({ "hostapd_cli", "-i", name,
					 "list_sta" },
				       output, kCommandStatusTimeout,
				       "hostapd_cli list_sta " + name)) {
			if (hostapd_sta_list_has_client(output)) {
				return true;
			}
		}
	}
	return false;
}

bool validate_executable_script(std::string_view script_path,
				std::string_view description)
{
	const std::string path(script_path);
	struct stat st {};
	if (::stat(path.c_str(), &st) != 0) {
		const std::string reason =
			(errno == ENOENT || errno == ENOTDIR) ?
				"missing" :
				std::strerror(errno);
		spdlog::warn(
			"NanoKVM runtime {} unavailable: path={}, reason={}",
			description, path, reason);
		return false;
	}
	if (!S_ISREG(st.st_mode)) {
		spdlog::warn(
			"NanoKVM runtime {} unavailable: path={}, reason=not regular file",
			description, path);
		return false;
	}
	if (::access(path.c_str(), X_OK) != 0) {
		spdlog::warn(
			"NanoKVM runtime {} unavailable: path={}, reason=not executable: {}",
			description, path, std::strerror(errno));
		return false;
	}
	return true;
}

bool validate_wifi_script(std::string_view script_path)
{
	return validate_executable_script(script_path, "WiFi script");
}

bool set_hdmi_input_enabled_impl(bool enabled)
{
	return set_flag_file_enabled(kHdmiEnablePath, enabled);
}

bool set_hid_enabled_impl(bool enabled)
{
	return set_flag_file_enabled(kHidEnablePath, enabled);
}

bool validate_ncm_restart_script(std::string_view script_path)
{
	return validate_executable_script(script_path, "NCM restart script");
}

bool set_ncm_enabled_impl(const NcmBootMarkerPaths &paths,
				   std::string_view restart_script, bool enabled)
{
	const NcmUsbFlagTargets targets = ncm_usb_flag_targets(enabled);
	const bool ncm_ok =
		set_flag_file_enabled(paths.ncm_flag, targets.ncm_enabled);
	const bool rndis_ok = set_flag_file_enabled(paths.rndis_flag,
						    targets.rndis_enabled);
	if (!ncm_ok || !rndis_ok) {
		return false;
	}
	if (!validate_ncm_restart_script(restart_script)) {
		return false;
	}
	if (!detail::run_success({ std::string(restart_script), "restart" },
				 kCommandActionTimeout, "usb gadget restart")) {
		spdlog::warn(
			"NanoKVM runtime NCM restart script failed: path={}, reason=restart command failed",
			restart_script);
		return false;
	}
	return true;
}

} // namespace detail
} // namespace lvgl_main
