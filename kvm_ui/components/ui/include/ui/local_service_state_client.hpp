#pragma once

#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

namespace ui::local_service_state
{

inline constexpr std::string_view kDefaultSocketPath =
	"/run/kvm/nanokvm-state.sock";
inline constexpr std::string_view kMCPServiceID = "mcp";
inline constexpr std::string_view kMemoryFabricServiceID = "memoryFabric";
inline constexpr std::string_view kScreenTimelapseServiceID = "screenTimelapse";

struct ServicesSnapshot {
	int64_t revision = 0;
	bool mcp_enabled = false;
	bool mcp_supported = false;
	bool memory_fabric_enabled = false;
	bool memory_fabric_supported = false;
	bool memory_fabric_running = false;
	bool memory_fabric_running_known = false;
	bool screen_timelapse_enabled = false;
	bool screen_timelapse_supported = false;
};

struct ClientResult {
	bool ok = false;
	int status_code = 0;
	ServicesSnapshot services {};
	std::string error;

	explicit operator bool() const
	{
		return ok;
	}
};

ClientResult get_services(std::string_view socket_path = kDefaultSocketPath);
ClientResult set_service(std::string_view service_id, bool enabled,
			 std::string_view socket_path = kDefaultSocketPath);
ClientResult set_mcp_bundle(bool enabled, bool include_memory_fabric,
			    std::string_view socket_path = kDefaultSocketPath);

ClientResult parse_services_response(std::string_view body);
std::string build_set_service_request_body(bool enabled);
bool mcp_bundle_active(const ServicesSnapshot &services);
bool screen_timelapse_active(const ServicesSnapshot &services);
std::vector<std::string_view>
mcp_bundle_service_order(bool include_memory_fabric);

namespace detail
{

struct HttpResponse {
	bool ok = false;
	int status_code = 0;
	std::string body;
	std::string error;
};

HttpResponse parse_http_response(std::string_view response);

} // namespace detail

} // namespace ui::local_service_state
