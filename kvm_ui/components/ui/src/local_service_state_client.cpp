#include "ui/local_service_state_client.hpp"

#include <cerrno>
#include <cstdlib>
#include <cstring>
#include <sstream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include <sys/socket.h>
#include <sys/time.h>
#include <sys/un.h>
#include <unistd.h>

#include <nlohmann/json.hpp>

namespace ui::local_service_state
{

namespace
{

using json = nlohmann::json;

constexpr std::size_t kMaxHTTPResponseBytes = 64U * 1024U;
constexpr int kSocketTimeoutSeconds = 1;

class UniqueFD {
    public:
	explicit UniqueFD(int fd)
		: _fd(fd)
	{
	}

	~UniqueFD()
	{
		if (_fd >= 0) {
			(void)::close(_fd);
		}
	}

	UniqueFD(const UniqueFD &) = delete;
	UniqueFD &operator=(const UniqueFD &) = delete;

	int get() const
	{
		return _fd;
	}

    private:
	int _fd = -1;
};

ClientResult make_error(std::string error, int status_code = 0)
{
	ClientResult result;
	result.status_code = status_code;
	result.error = std::move(error);
	return result;
}

bool is_supported_service_id(std::string_view service_id)
{
	return service_id == kMCPServiceID ||
	       service_id == kMemoryFabricServiceID ||
	       service_id == kScreenTimelapseServiceID;
}

bool send_all(int fd, std::string_view data, std::string &error)
{
	const char *cursor = data.data();
	std::size_t remaining = data.size();
	while (remaining > 0U) {
#ifdef MSG_NOSIGNAL
		const ssize_t sent =
			::send(fd, cursor, remaining, MSG_NOSIGNAL);
#else
		const ssize_t sent = ::send(fd, cursor, remaining, 0);
#endif
		if (sent < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (sent == 0) {
			error = "socket closed while sending";
			return false;
		}
		cursor += sent;
		remaining -= static_cast<std::size_t>(sent);
	}
	return true;
}

bool recv_all(int fd, std::string &response, std::string &error)
{
	char buffer[4096];
	for (;;) {
		const ssize_t received = ::recv(fd, buffer, sizeof(buffer), 0);
		if (received < 0) {
			if (errno == EINTR) {
				continue;
			}
			error = std::strerror(errno);
			return false;
		}
		if (received == 0) {
			return true;
		}
		if (response.size() + static_cast<std::size_t>(received) >
		    kMaxHTTPResponseBytes) {
			error = "response too large";
			return false;
		}
		response.append(buffer, static_cast<std::size_t>(received));
	}
}

std::string build_http_request(std::string_view method, std::string_view path,
			       std::string_view body)
{
	std::ostringstream out;
	out << method << ' ' << path << " HTTP/1.1\r\n";
	out << "Host: localhost\r\n";
	out << "Connection: close\r\n";
	if (!body.empty()) {
		out << "Content-Type: application/json\r\n";
		out << "Content-Length: " << body.size() << "\r\n";
	}
	out << "\r\n";
	out << body;
	return out.str();
}

detail::HttpResponse request_unix_http(std::string_view socket_path,
				       std::string_view method,
				       std::string_view path,
				       std::string_view body)
{
#ifdef SOCK_CLOEXEC
	UniqueFD fd(::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0));
#else
	UniqueFD fd(::socket(AF_UNIX, SOCK_STREAM, 0));
#endif
	if (fd.get() < 0) {
		detail::HttpResponse response;
		response.error = std::strerror(errno);
		return response;
	}

	timeval timeout {};
	timeout.tv_sec = kSocketTimeoutSeconds;
	(void)::setsockopt(fd.get(), SOL_SOCKET, SO_RCVTIMEO, &timeout,
			   sizeof(timeout));
	(void)::setsockopt(fd.get(), SOL_SOCKET, SO_SNDTIMEO, &timeout,
			   sizeof(timeout));

	sockaddr_un addr {};
	addr.sun_family = AF_UNIX;
	if (socket_path.size() >= sizeof(addr.sun_path)) {
		detail::HttpResponse response;
		response.error = "socket path is too long";
		return response;
	}
	std::memcpy(addr.sun_path, socket_path.data(), socket_path.size());
	addr.sun_path[socket_path.size()] = '\0';

	if (::connect(fd.get(), reinterpret_cast<sockaddr *>(&addr),
		      sizeof(addr)) != 0) {
		detail::HttpResponse response;
		response.error = std::strerror(errno);
		return response;
	}

	const std::string request = build_http_request(method, path, body);
	std::string io_error;
	if (!send_all(fd.get(), request, io_error)) {
		detail::HttpResponse response;
		response.error = io_error;
		return response;
	}

	std::string response;
	if (!recv_all(fd.get(), response, io_error)) {
		detail::HttpResponse http_response;
		http_response.error = io_error;
		return http_response;
	}

	return detail::parse_http_response(response);
}

std::string service_path(std::string_view service_id)
{
	std::string path = "/v1/services/";
	path.append(service_id);
	return path;
}

ClientResult result_from_http_response(const detail::HttpResponse &response)
{
	if (!response.ok) {
		return make_error(response.error, response.status_code);
	}

	ClientResult result = parse_services_response(response.body);
	result.status_code = response.status_code;
	return result;
}

} // namespace

ClientResult get_services(std::string_view socket_path)
{
	const detail::HttpResponse response =
		request_unix_http(socket_path, "GET", "/v1/services", {});
	return result_from_http_response(response);
}

ClientResult set_service(std::string_view service_id, bool enabled,
			 std::string_view socket_path)
{
	if (!is_supported_service_id(service_id)) {
		return make_error("unsupported service id");
	}

	const std::string body = build_set_service_request_body(enabled);
	const detail::HttpResponse response = request_unix_http(
		socket_path, "PUT", service_path(service_id), body);
	return result_from_http_response(response);
}

ClientResult set_mcp_bundle(bool enabled, bool include_memory_fabric,
			    std::string_view socket_path)
{
	ClientResult last_result;
	for (const std::string_view service_id :
	     mcp_bundle_service_order(include_memory_fabric)) {
		last_result = set_service(service_id, enabled, socket_path);
		if (!last_result) {
			return last_result;
		}
	}
	return last_result;
}

ClientResult parse_services_response(std::string_view body)
{
	try {
		const json root = json::parse(body);
		const json &services = root.at("services");
		const json &mcp = services.at(std::string(kMCPServiceID));
		const json &memory_fabric =
			services.at(std::string(kMemoryFabricServiceID));
		const json &screen_timelapse =
			services.at(std::string(kScreenTimelapseServiceID));

		ClientResult result;
		result.ok = true;
		result.services.revision = root.value("revision", 0);
		result.services.mcp_enabled = mcp.at("enabled").get<bool>();
		result.services.mcp_supported = mcp.value("supported", false);
		result.services.memory_fabric_enabled =
			memory_fabric.at("enabled").get<bool>();
		result.services.memory_fabric_supported =
			memory_fabric.value("supported", false);
		if (const auto running = memory_fabric.find("running");
		    running != memory_fabric.end()) {
			result.services.memory_fabric_running =
				running->get<bool>();
			result.services.memory_fabric_running_known = true;
		}
		result.services.screen_timelapse_enabled =
			screen_timelapse.at("enabled").get<bool>();
		result.services.screen_timelapse_supported =
			screen_timelapse.value("supported", false);
		return result;
	} catch (const std::exception &error) {
		return make_error(error.what());
	}
}

std::string build_set_service_request_body(bool enabled)
{
	json body;
	body["enabled"] = enabled;
	body["source"] = "screen";
	return body.dump();
}

bool mcp_bundle_active(const ServicesSnapshot &services)
{
	return services.mcp_enabled || services.memory_fabric_enabled;
}

bool screen_timelapse_active(const ServicesSnapshot &services)
{
	return services.screen_timelapse_enabled;
}

std::vector<std::string_view>
mcp_bundle_service_order(bool include_memory_fabric)
{
	std::vector<std::string_view> order { kMCPServiceID };
	if (include_memory_fabric) {
		order.emplace_back(kMemoryFabricServiceID);
	}
	return order;
}

namespace detail
{

HttpResponse parse_http_response(std::string_view response)
{
	const std::string_view separator = "\r\n\r\n";
	const std::size_t header_end = response.find(separator);
	if (header_end == std::string_view::npos) {
		HttpResponse parsed;
		parsed.error = "invalid HTTP response";
		return parsed;
	}

	const std::string_view headers = response.substr(0, header_end);
	const std::size_t line_end = headers.find("\r\n");
	const std::string_view status_line =
		line_end == std::string_view::npos ?
			headers :
			headers.substr(0, line_end);

	const std::size_t first_space = status_line.find(' ');
	if (first_space == std::string_view::npos ||
	    first_space + 4U > status_line.size()) {
		HttpResponse parsed;
		parsed.error = "invalid HTTP status line";
		return parsed;
	}

	const std::string status_text(status_line.substr(first_space + 1U, 3U));
	char *end = nullptr;
	const long status_code = std::strtol(status_text.c_str(), &end, 10);
	if (end == status_text.c_str() || *end != '\0' || status_code <= 0 ||
	    status_code > 999) {
		HttpResponse parsed;
		parsed.error = "invalid HTTP status code";
		return parsed;
	}

	HttpResponse parsed;
	parsed.status_code = static_cast<int>(status_code);
	parsed.body =
		std::string(response.substr(header_end + separator.size()));
	if (parsed.status_code < 200 || parsed.status_code >= 300) {
		parsed.error =
			"HTTP status " + std::to_string(parsed.status_code);
		return parsed;
	}

	parsed.ok = true;
	return parsed;
}

} // namespace detail

} // namespace ui::local_service_state
