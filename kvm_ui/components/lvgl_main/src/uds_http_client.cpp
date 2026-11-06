#include "uds_http_client.hpp"

#include <algorithm>
#include <cerrno>
#include <cstdint>
#include <cstring>
#include <limits>

#include <fcntl.h>
#include <poll.h>
#include <sys/eventfd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

namespace lvgl_main
{
namespace detail
{
namespace
{

constexpr std::size_t kMaxHeaderBytes = 16U * 1024U;
constexpr std::size_t kMaxChunkSize = 16U * 1024U * 1024U;
constexpr std::string_view kHeaderEnd = "\r\n\r\n";

bool set_blocking(int fd, bool blocking)
{
	const int flags = fcntl(fd, F_GETFL, 0);
	if (flags < 0) {
		return false;
	}
	const int next =
		blocking ? (flags & ~O_NONBLOCK) : (flags | O_NONBLOCK);
	return fcntl(fd, F_SETFL, next) == 0;
}

bool equals_ignore_ascii_case(std::string_view a, std::string_view b)
{
	if (a.size() != b.size()) {
		return false;
	}
	for (std::size_t i = 0; i < a.size(); ++i) {
		char ca = a[i];
		char cb = b[i];
		if (ca >= 'A' && ca <= 'Z') {
			ca += 32;
		}
		if (cb >= 'A' && cb <= 'Z') {
			cb += 32;
		}
		if (ca != cb) {
			return false;
		}
	}
	return true;
}

std::optional<std::string_view> header_value(std::string_view headers,
					     std::string_view name)
{
	std::size_t pos = 0;
	while (pos < headers.size()) {
		const auto line_end = headers.find("\r\n", pos);
		const auto line = headers.substr(
			pos, line_end == std::string_view::npos ?
				     std::string_view::npos :
				     line_end - pos);
		const auto colon = line.find(':');
		if (colon != std::string_view::npos &&
		    equals_ignore_ascii_case(line.substr(0, colon), name)) {
			auto value = line.substr(colon + 1);
			while (!value.empty() &&
			       (value.front() == ' ' || value.front() == '\t')) {
				value.remove_prefix(1);
			}
			while (!value.empty() &&
			       (value.back() == ' ' || value.back() == '\t')) {
				value.remove_suffix(1);
			}
			return value;
		}
		if (line_end == std::string_view::npos) {
			break;
		}
		pos = line_end + 2;
	}
	return std::nullopt;
}

} // namespace

UdsHttpClient::UdsHttpClient(std::string socket_path,
			     std::chrono::milliseconds connect_timeout,
			     std::chrono::milliseconds io_timeout,
			     std::size_t max_response_bytes)
	: _socket_path(std::move(socket_path))
	, _connect_timeout(connect_timeout)
	, _io_timeout(io_timeout)
	, _max_response_bytes(max_response_bytes)
{
	_wake_fd = eventfd(0, EFD_CLOEXEC | EFD_NONBLOCK);
}

UdsHttpClient::~UdsHttpClient()
{
	close_connection();
	if (_wake_fd >= 0) {
		::close(_wake_fd);
	}
}

void UdsHttpClient::stop()
{
	if (_wake_fd < 0) {
		return;
	}
	const std::uint64_t one = 1;
	static_cast<void>(
		::write(_wake_fd, &one, static_cast<ssize_t>(sizeof(one))));
}

bool UdsHttpClient::interrupted() const
{
	if (_wake_fd < 0) {
		return false;
	}
	pollfd probe { _wake_fd, POLLIN, 0 };
	return ::poll(&probe, 1, 0) > 0 && (probe.revents & POLLIN) != 0;
}

void UdsHttpClient::close_connection()
{
	if (_fd >= 0) {
		::close(_fd);
		_fd = -1;
	}
}

int UdsHttpClient::timeout_ms(std::chrono::milliseconds value)
{
	return static_cast<int>(std::min<std::chrono::milliseconds::rep>(
		value.count(),
		static_cast<std::chrono::milliseconds::rep>(
			std::numeric_limits<int>::max())));
}

bool UdsHttpClient::wait_ready(int fd, short events)
{
	pollfd fds[2] { { fd, events, 0 }, { _wake_fd, POLLIN, 0 } };
	const int nfds = _wake_fd >= 0 ? 2 : 1;
	for (;;) {
		const int rc = ::poll(fds, nfds, timeout_ms(_io_timeout));
		if (rc < 0) {
			if (errno == EINTR) {
				if (interrupted()) {
					return false;
				}
				continue;
			}
			return false;
		}
		if (rc == 0) {
			return false;
		}
		if (nfds == 2 && (fds[1].revents & POLLIN) != 0) {
			return false;
		}
		return (fds[0].revents & (events | POLLERR | POLLHUP)) != 0;
	}
}

class UdsHttpClient::Reader final
{
public:
	Reader(int fd, UdsHttpClient &client) : _fd(fd), _client(client) {}

	bool server_closed() const { return _server_closed; }

	std::string take_buffer() { return std::move(_buf); }

	bool fill(std::size_t target)
	{
		char chunk[4096];
		while (_buf.size() < target) {
			if (!_client.wait_ready(_fd, POLLIN)) {
				return false;
			}
			const auto rc = ::recv(_fd, chunk, sizeof(chunk), 0);
			if (rc < 0) {
				if (errno == EINTR) {
					continue;
				}
				return false;
			}
			if (rc == 0) {
				_server_closed = true;
				return false;
			}
			_buf.append(chunk, static_cast<std::size_t>(rc));
		}
		return true;
	}

	bool read_line(std::string &line, std::size_t cap)
	{
		for (;;) {
			const auto pos = _buf.find("\r\n");
			if (pos != std::string::npos) {
				if (pos > cap) {
					return false;
				}
				line.assign(_buf, 0, pos);
				_buf.erase(0, pos + 2);
				return true;
			}
			if (_buf.size() > cap) {
				return false;
			}
			if (!fill(_buf.size() + 1)) {
				return false;
			}
		}
	}

	bool read_exact(std::size_t count, std::string &out)
	{
		if (!fill(count)) {
			return false;
		}
		out.append(_buf, 0, count);
		_buf.erase(0, count);
		return true;
	}

	bool read_to_eof(std::string &out, std::size_t cap)
	{
		for (;;) {
			if (!_client.wait_ready(_fd, POLLIN)) {
				return false;
			}
			char chunk[4096];
			const auto rc = ::recv(_fd, chunk, sizeof(chunk), 0);
			if (rc < 0) {
				if (errno == EINTR) {
					continue;
				}
				return false;
			}
			if (rc == 0) {
				_server_closed = true;
				return true;
			}
			out.append(chunk, static_cast<std::size_t>(rc));
			if (out.size() > cap) {
				return false;
			}
		}
	}

private:
	int _fd;
	UdsHttpClient &_client;
	std::string _buf;
	bool _server_closed { false };
};

bool UdsHttpClient::ensure_connected()
{
	if (_fd >= 0) {
		return true;
	}

	const int fd = ::socket(AF_UNIX, SOCK_STREAM | SOCK_CLOEXEC, 0);
	if (fd < 0) {
		_last_error = "socket";
		return false;
	}

	sockaddr_un addr {};
	addr.sun_family = AF_UNIX;
	std::strncpy(addr.sun_path, _socket_path.c_str(),
		     sizeof(addr.sun_path) - 1);

	if (!set_blocking(fd, false)) {
		_last_error = "fcntl";
		::close(fd);
		return false;
	}

	const int rc = ::connect(fd, reinterpret_cast<sockaddr *>(&addr),
				 sizeof(addr));
	if (rc < 0 && errno != EINPROGRESS) {
		_last_error = "connect";
		::close(fd);
		return false;
	}
	if (rc != 0) {
		pollfd fds[2] { { fd, POLLOUT, 0 }, { _wake_fd, POLLIN, 0 } };
		const int nfds = _wake_fd >= 0 ? 2 : 1;
		const int poll_rc = ::poll(fds, nfds, timeout_ms(_connect_timeout));
		int socket_error = 0;
		socklen_t error_len = sizeof(socket_error);
		if (poll_rc <= 0 ||
		    (nfds == 2 && (fds[1].revents & POLLIN) != 0) ||
		    getsockopt(fd, SOL_SOCKET, SO_ERROR, &socket_error,
			       &error_len) != 0 ||
		    socket_error != 0) {
			_last_error = poll_rc == 0 ? "connect timeout" : "connect";
			::close(fd);
			return false;
		}
	}

	if (!set_blocking(fd, true)) {
		_last_error = "fcntl";
		::close(fd);
		return false;
	}
	_fd = fd;
	return true;
}

bool UdsHttpClient::send_all(std::string_view data)
{
	std::size_t sent = 0;
	while (sent < data.size()) {
		if (!wait_ready(_fd, POLLOUT)) {
			return false;
		}
		const auto rc = ::send(_fd, data.data() + sent,
				       data.size() - sent, MSG_NOSIGNAL);
		if (rc < 0) {
			if (errno == EINTR) {
				continue;
			}
			return false;
		}
		if (rc == 0) {
			return false;
		}
		sent += static_cast<std::size_t>(rc);
	}
	return true;
}

UdsHttpClient::ReadResult
UdsHttpClient::read_response(Response &out, Reader &reader)
{
	std::string status_line;
	if (!reader.read_line(status_line, kMaxHeaderBytes)) {
		_last_error = "response headers";
		return ReadResult::Failure;
	}
	if (!status_line.starts_with("HTTP/")) {
		_last_error = "malformed status line";
		return ReadResult::Failure;
	}
	const auto code_pos = status_line.find(' ');
	if (code_pos == std::string_view::npos ||
	    code_pos + 4 > status_line.size()) {
		_last_error = "malformed status line";
		return ReadResult::Failure;
	}
	int status = 0;
	for (std::size_t i = code_pos + 1; i < code_pos + 4; ++i) {
		const char c = status_line[i];
		if (c < '0' || c > '9') {
			_last_error = "malformed status line";
			return ReadResult::Failure;
		}
		status = status * 10 + (c - '0');
	}
	out.status = status;

	std::string headers;
	headers.reserve(512);
	std::size_t header_bytes = 0;
	for (;;) {
		std::string line;
		if (!reader.read_line(line, kMaxHeaderBytes)) {
			_last_error = "response headers";
			return ReadResult::Failure;
		}
		if (line.empty()) {
			break;
		}
		header_bytes += line.size() + 2;
		if (header_bytes > kMaxHeaderBytes) {
			_last_error = "response headers exceed limit";
			return ReadResult::Failure;
		}
		if (!headers.empty()) {
			headers.append("\r\n");
		}
		headers.append(line);
	}

	const auto transfer_encoding = header_value(headers, "transfer-encoding");
	const auto content_length = header_value(headers, "content-length");
	const auto connection = header_value(headers, "connection");
	const bool server_will_close =
		connection.has_value() &&
		equals_ignore_ascii_case(*connection, "close");

	std::string body;

	if (transfer_encoding.has_value() &&
	    equals_ignore_ascii_case(*transfer_encoding, "chunked")) {
		for (;;) {
			std::string size_line;
			if (!reader.read_line(size_line, 64)) {
				_last_error = "chunk size";
				return ReadResult::Failure;
			}
			const auto semi = size_line.find(';');
			const auto size_token =
				std::string_view(size_line).substr(0, semi);
			if (size_token.empty()) {
				_last_error = "chunk size";
				return ReadResult::Failure;
			}
			std::size_t chunk_size = 0;
			for (const char c : size_token) {
				int digit = -1;
				if (c >= '0' && c <= '9') {
					digit = c - '0';
				} else if (c >= 'a' && c <= 'f') {
					digit = c - 'a' + 10;
				} else if (c >= 'A' && c <= 'F') {
					digit = c - 'A' + 10;
				} else {
					_last_error = "chunk size";
					return ReadResult::Failure;
				}
				chunk_size = chunk_size * 16U +
					     static_cast<std::size_t>(digit);
				if (chunk_size > kMaxChunkSize) {
					_last_error = "response exceeds limit";
					return ReadResult::Failure;
				}
			}
			if (chunk_size == 0) {
				std::size_t trailer_bytes = 0;
				for (;;) {
					std::string trailer;
					if (!reader.read_line(trailer,
							      kMaxHeaderBytes)) {
						_last_error = "chunk trailer";
						return ReadResult::Failure;
					}
					if (trailer.empty()) {
						break;
					}
					trailer_bytes += trailer.size() + 2;
					if (trailer_bytes > kMaxHeaderBytes) {
						_last_error = "chunk trailers exceed limit";
						return ReadResult::Failure;
					}
				}
				break;
			}
			if (body.size() + chunk_size > _max_response_bytes) {
				_last_error = "response exceeds limit";
				return ReadResult::Failure;
			}
			if (!reader.read_exact(chunk_size, body)) {
				_last_error = "chunk data";
				return ReadResult::Failure;
			}
			std::string terminator;
			if (!reader.read_line(terminator, 2) ||
			    !terminator.empty()) {
				_last_error = "chunk terminator";
				return ReadResult::Failure;
			}
		}
	} else if (content_length.has_value()) {
		std::size_t length = 0;
		for (const char c : *content_length) {
			if (c < '0' || c > '9') {
				_last_error = "content length";
				return ReadResult::Failure;
			}
			length = length * 10U + static_cast<std::size_t>(c - '0');
			if (length > _max_response_bytes) {
				_last_error = "response exceeds limit";
				return ReadResult::Failure;
			}
		}
		if (!reader.read_exact(length, body)) {
			_last_error = "response body";
			return ReadResult::Failure;
		}
	} else if (server_will_close) {
		body = reader.take_buffer();
		if (body.size() > _max_response_bytes) {
			_last_error = "response exceeds limit";
			return ReadResult::Failure;
		}
		if (!reader.read_to_eof(body, _max_response_bytes)) {
			_last_error = body.size() > _max_response_bytes ?
					      "response exceeds limit" :
					      "response body";
			return ReadResult::Failure;
		}
	} else {
		_last_error = "missing content length";
		return ReadResult::Failure;
	}

	out.body = std::move(body);
	return server_will_close ? ReadResult::SuccessServerClose :
				   ReadResult::Success;
}

std::optional<UdsHttpClient::Response>
UdsHttpClient::request_once(std::string_view request,
			    bool &pre_response_failure)
{
	pre_response_failure = false;
	if (!ensure_connected()) {
		pre_response_failure = true;
		return std::nullopt;
	}

	if (!send_all(request)) {
		_last_error = "write";
		pre_response_failure = true;
		close_connection();
		return std::nullopt;
	}

	Response response;
	Reader reader(_fd, *this);
	const auto result = read_response(response, reader);
	if (result == ReadResult::Failure) {
		pre_response_failure = reader.server_closed();
		if (_last_error.empty()) {
			_last_error = reader.server_closed() ? "closed" : "read";
		}
		close_connection();
		return std::nullopt;
	}
	if (result == ReadResult::SuccessServerClose) {
		close_connection();
	}
	return response;
}

void UdsHttpClient::clear_stale_interrupt()
{
	if (_wake_fd < 0) {
		return;
	}
	std::uint64_t value = 0;
	static_cast<void>(::read(_wake_fd, &value, sizeof(value)));
}

std::optional<UdsHttpClient::Response>
UdsHttpClient::finish_request(std::optional<Response> result)
{
	if (!result && interrupted()) {
		_last_error = "interrupted";
	} else if (!result && _last_error.empty()) {
		_last_error = "io";
	}
	return result;
}

std::optional<UdsHttpClient::Response>
UdsHttpClient::get(std::string_view path)
{
	_last_error.clear();
	clear_stale_interrupt();

	std::string request;
	request.reserve(path.size() + 96);
	request.append("GET ");
	request.append(path);
	request.append(" HTTP/1.1\r\nHost: localhost\r\nAccept: */*\r\n"
		       "Connection: keep-alive\r\n\r\n");

	bool pre_response_failure = false;
	auto result = request_once(request, pre_response_failure);
	if (!result && pre_response_failure && !interrupted()) {
		_last_error.clear();
		pre_response_failure = false;
		result = request_once(request, pre_response_failure);
	}
	return finish_request(std::move(result));
}

std::optional<UdsHttpClient::Response>
UdsHttpClient::post(std::string_view path, std::string_view json_body)
{
	_last_error.clear();
	clear_stale_interrupt();

	std::string request;
	request.reserve(path.size() + json_body.size() + 128);
	request.append("POST ");
	request.append(path);
	request.append(" HTTP/1.1\r\nHost: localhost\r\n"
		       "Content-Type: application/json\r\nContent-Length: ");
	request.append(std::to_string(json_body.size()));
	request.append("\r\nConnection: keep-alive\r\n\r\n");
	request.append(json_body);

	bool pre_response_failure = false;
	return finish_request(request_once(request, pre_response_failure));
}

}
}
