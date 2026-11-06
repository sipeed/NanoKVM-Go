#pragma once

#include <chrono>
#include <filesystem>
#include <initializer_list>
#include <optional>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include <sys/stat.h>
#include <sys/types.h>

#include <httplib.h>
#include <nlohmann/json.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"
#include "lvgl_main/runtime_read_observation.hpp"
#include "lvgl_main/runtime_service.hpp"

namespace lvgl_main
{

class NanoKvmServerTransport;

namespace detail
{

struct WifiApCredentials {
	std::string ssid;
	std::string password;
	std::string ap_ip;
	bool confirmed { false };
};

struct NcmUsbFlagTargets {
	bool ncm_enabled { false };
	bool rndis_enabled { false };
};

constexpr NcmUsbFlagTargets ncm_usb_flag_targets(bool enabled)
{
	return { .ncm_enabled = enabled, .rndis_enabled = false };
}

struct WifiInterfaceInfo {
	std::vector<std::string> names;
	bool has_ap_gateway { false };
	bool observed { false };
};

struct UrlParts {
	std::string scheme;
	std::string host;
	int port { 80 };
	bool valid { false };
};

struct CommandResult {
	int exit_code { -1 };
	std::string output;
	bool timed_out { false };
};

class ScopedFd final {
    public:
	explicit ScopedFd(int fd = -1);

	~ScopedFd();

	ScopedFd(const ScopedFd &) = delete;
	ScopedFd &operator=(const ScopedFd &) = delete;
	ScopedFd(ScopedFd &&other) noexcept;
	ScopedFd &operator=(ScopedFd &&other) noexcept;

	int get() const;
	int release();
	void reset(int fd = -1);

    private:
	int _fd { -1 };
};

struct CommandProcess {
	pid_t pid { -1 };
	ScopedFd read_fd;
};

std::string command_name_for_log(const std::vector<std::string> &args);
std::string command_args_summary_for_log(
	const std::vector<std::string> &args);
CommandResult run_command(const std::vector<std::string> &args,
			  std::chrono::milliseconds timeout,
			  std::string_view summary = {});
void spawn_command_detached(const std::vector<std::string> &args);
bool run_success(const std::vector<std::string> &args,
			std::chrono::milliseconds timeout,
			std::string_view summary = {});
bool run_output_success(const std::vector<std::string> &args,
			std::string &output, std::chrono::milliseconds timeout,
			std::string_view summary = {});
bool write_text_file_secure(const std::filesystem::path &path,
			    std::string_view payload, mode_t mode);

bool wifi_script_available_quiet(std::string_view script_path);
CommandResult run_wifi_script(const NanokvmRuntimeConfig &config,
			      const std::vector<std::string> &args);
bool query_wifi_boot_enabled(const NanokvmRuntimeConfig &config);
std::optional<int> query_wifi_device_count(
		const NanokvmRuntimeConfig &config, bool allow_nonzero_exit);
bool validate_executable_script(std::string_view script_path,
				std::string_view description);
bool validate_wifi_script(std::string_view script_path);
bool validate_ncm_restart_script(std::string_view script_path);

std::string trim(std::string value);
bool starts_with(std::string_view text, std::string_view prefix);
bool contains_text(std::string_view text, std::string_view needle);
void redact_text(std::string &text, std::string_view secret);
std::string command_output_summary_for_log(
	std::string output, std::string_view secret = {});
std::string percent_encode_form_value(std::string_view value);
UrlParts parse_base_url(std::string_view url);

std::optional<nlohmann::json>
response_data(const nlohmann::json &root);
std::optional<std::string>
json_string(const nlohmann::json &value, std::string_view key);
std::optional<bool> json_bool(const nlohmann::json &value,
			      std::string_view key);
std::optional<std::string>
json_first_string(const nlohmann::json &value,
			std::initializer_list<std::string_view> keys);
std::optional<double> json_number(const nlohmann::json &value,
				  std::string_view key);
bool optional_double_equal(std::optional<double> lhs,
				   std::optional<double> rhs) noexcept;

bool display_metrics_changed(const RuntimeStatus &lhs,
			     const RuntimeStatus &rhs) noexcept;
bool systemd_status_changed(const RuntimeStatus &lhs,
			    const RuntimeStatus &rhs) noexcept;
bool domain_refresh_due(std::chrono::steady_clock::time_point last_refresh,
			std::chrono::steady_clock::time_point now,
			std::chrono::steady_clock::duration interval) noexcept;
void copy_systemd_status_fields(const RuntimeStatus &from, RuntimeStatus &to);
void copy_display_metrics_fields(const RuntimeStatus &from, RuntimeStatus &to);
bool local_service_status_changed(const RuntimeStatus &lhs,
				  const RuntimeStatus &rhs);
void sync_runtime_flags(RuntimeStatus &status, bool video_enabled,
			bool hid_enabled, bool ncm_enabled, bool ncm_supported,
			bool auto_sleep_enabled, int backlight_percent,
			bool screen_sleeping);
void clear_display_metrics(RuntimeStatus &status);

std::optional<nlohmann::json>
http_request_json(const NanokvmRuntimeConfig &config,
			 std::string_view method, std::string_view path,
			 const httplib::Params &params = {});

std::optional<nlohmann::json>
http_post_json(const NanokvmRuntimeConfig &config, std::string_view path,
	       const nlohmann::json &body);

bool region_switch_response_ok(const nlohmann::json &response);

bool is_wifi_interface_name(std::string_view ifname);

std::optional<bool> observe_file_exists(std::string_view path);
std::optional<std::string> read_kvmcomm_package_version();
std::optional<std::string> read_local_hardware_version();

StateReadObservation<bool> systemd_is_active(std::string_view service);
bool systemd_enable(std::string_view service);
bool systemd_disable(std::string_view service);
StateReadObservation<bool> systemd_unit_exists(std::string_view unit);

bool collect_ip_addresses(RuntimeStatus &status);
WifiInterfaceInfo collect_wifi_interfaces();

bool write_flag_file(const std::filesystem::path &path);
bool remove_flag_file(const std::filesystem::path &path);
bool set_flag_file_enabled(std::string_view path, bool enabled);
void write_wifi_ap_runtime_files(std::string_view password);

std::pair<std::string, std::string> make_wifi_ap_credentials();
void ensure_wifi_ap_credentials_complete(WifiApCredentials &credentials);
WifiApCredentials &cached_wifi_ap_credentials();
void update_wifi_ap_credentials_from_api(const nlohmann::json &data);
std::string escape_wifi_qr_field(std::string_view value);
std::string make_wifi_ap_qr_payload(const WifiApCredentials &credentials);
std::string make_wifi_web_url(const WifiApCredentials &credentials);

bool hostapd_sta_list_has_client(std::string_view output);
std::optional<bool> hostapd_ap_client_connected(
		const std::vector<std::string> &interface_names);
bool check_wifi_ap_client();

bool set_hdmi_input_enabled_impl(bool enabled);
bool set_hid_enabled_impl(bool enabled);
bool set_ncm_enabled_impl(const NcmBootMarkerPaths &paths,
				  std::string_view restart_script, bool enabled);

void collect_streamer_status(NanoKvmServerTransport &transport,
			     RuntimeStatus &status);
bool collect_hid_status(NanoKvmServerTransport &transport,
			RuntimeStatus &status);
bool collect_video_status(NanoKvmServerTransport &transport,
			RuntimeStatus &status);
bool collect_wifi_status(const NanokvmRuntimeConfig &config,
			 NanoKvmServerTransport &transport,
			 RuntimeStatus &status, bool station_ip_observed,
			 bool allow_nonzero_script_output);
bool collect_hardware_version_status(
	NanoKvmServerTransport &transport, RuntimeStatus &status);

std::optional<std::string> read_device_key();
std::optional<std::string> read_hostname();
std::optional<std::string> read_image_version();
std::optional<std::string> read_edid_name();
std::optional<std::string> read_mac_address();
std::optional<std::string> read_device_number();

std::optional<long> parse_meminfo_kib(const std::string &line);
bool collect_temperature(RuntimeStatus &status);
bool collect_memory(RuntimeStatus &status);
bool collect_uptime(RuntimeStatus &status);
bool collect_local_time(RuntimeStatus &status);
bool collect_storage(RuntimeStatus &status);
bool collect_wifi_signal(RuntimeStatus &status, bool wifi_connected);
bool collect_systemd_status(RuntimeStatus &status,
			    bool preserve_last_known_good);
bool collect_ncm_status(const NcmBootMarkerPaths &paths,
			RuntimeStatus &status);
bool collect_backlight_status(const NanokvmRuntimeConfig &config,
			      int fallback_percent, bool screen_sleeping,
			      RuntimeStatus &status);

}
}
