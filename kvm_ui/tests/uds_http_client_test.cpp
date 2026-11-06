#include <atomic>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <filesystem>
#include <functional>
#include <string>
#include <system_error>
#include <thread>
#include <utility>
#include <vector>

#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <catch2/catch_test_macros.hpp>

#include "uds_http_client.hpp"

using namespace std::chrono_literals;
using namespace lvgl_main::detail;

namespace
{

constexpr std::chrono::milliseconds kIoTimeout { 1000 };

class TempDirectory final
{
    public:
	TempDirectory()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("uds_http_client_test_" + std::to_string(stamp) + "_" +
			 std::to_string(next_id.fetch_add(1)));
		std::filesystem::create_directories(_path);
	}

	~TempDirectory()
	{
		std::error_code error;
		std::filesystem::remove_all(_path, error);
	}

	std::filesystem::path socket_path(std::string_view name) const
	{
		return _path / std::string(name);
	}

    private:
	std::filesystem::path _path;
};

std::string read_request_headers(int fd)
{
	std::string data;
	char chunk[512];
	while (data.find("\r\n\r\n") == std::string::npos) {
		const auto rc = ::recv(fd, chunk, sizeof(chunk), 0);
		if (rc <= 0) {
			break;
		}
		data.append(chunk, static_cast<std::size_t>(rc));
	}
	return data;
}

std::size_t request_content_length(const std::string &request)
{
	constexpr std::string_view kHeader = "Content-Length: ";
	const auto found = request.find(kHeader);
	if (found == std::string::npos) {
		return 0;
	}
	std::size_t value = 0;
	for (std::size_t index = found + kHeader.size();
	     index < request.size(); ++index) {
		const char digit = request[index];
		if (digit < '0' || digit > '9') {
			break;
		}
		value = value * 10 + static_cast<std::size_t>(digit - '0');
	}
	return value;
}

std::string read_request(int fd)
{
	std::string data = read_request_headers(fd);
	const auto header_end = data.find("\r\n\r\n");
	if (header_end == std::string::npos) {
		return data;
	}
	const std::size_t body_start = header_end + 4;
	const std::size_t body_length = request_content_length(data);
	char chunk[512];
	while (data.size() - body_start < body_length) {
		const auto rc = ::recv(fd, chunk, sizeof(chunk), 0);
		if (rc <= 0) {
			break;
		}
		data.append(chunk, static_cast<std::size_t>(rc));
	}
	return data;
}

void send_all(int fd, std::string_view data)
{
	std::size_t sent = 0;
	while (sent < data.size()) {
		const auto rc = ::send(fd, data.data() + sent, data.size() - sent,
				       MSG_NOSIGNAL);
		if (rc <= 0) {
			return;
		}
		sent += static_cast<std::size_t>(rc);
	}
}

class ScriptedUdsServer final
{
    public:
	using Handler = std::function<void(int, int)>;

	ScriptedUdsServer(std::filesystem::path socket_path,
			  int connection_count, Handler handler)
		: _socket_path(std::move(socket_path))
		, _connection_count(connection_count)
		, _handler(std::move(handler))
	{
	}

	~ScriptedUdsServer() { stop(); }

	bool start()
	{
		std::error_code error;
		std::filesystem::remove(_socket_path, error);

		_listener_fd = ::socket(AF_UNIX, SOCK_STREAM, 0);
		if (_listener_fd < 0) {
			return false;
		}
		sockaddr_un address {};
		address.sun_family = AF_UNIX;
		std::strncpy(address.sun_path, _socket_path.c_str(),
			     sizeof(address.sun_path) - 1);
		if (::bind(_listener_fd, reinterpret_cast<const sockaddr *>(&address),
			   sizeof(address)) != 0 ||
		    ::listen(_listener_fd, 4) != 0) {
			::close(_listener_fd);
			_listener_fd = -1;
			return false;
		}
		_thread = std::thread([this] { accept_loop(); });
		return true;
	}

	void stop()
	{
		if (_listener_fd >= 0) {
			::shutdown(_listener_fd, SHUT_RDWR);
			::close(_listener_fd);
			_listener_fd = -1;
		}
		if (_thread.joinable()) {
			_thread.join();
		}
	}

	int accepted_count() const { return _accepted.load(); }

    private:
	void accept_loop()
	{
		for (int i = 0; i < _connection_count; ++i) {
			const int fd = ::accept(_listener_fd, nullptr, nullptr);
			if (fd < 0) {
				return;
			}
			_accepted.fetch_add(1);
			_handler(i, fd);
			::close(fd);
		}
	}

	std::filesystem::path _socket_path;
	int _connection_count;
	Handler _handler;
	int _listener_fd { -1 };
	std::thread _thread;
	std::atomic<int> _accepted { 0 };
};

UdsHttpClient make_client(const std::filesystem::path &socket_path,
			  std::chrono::milliseconds io_timeout = kIoTimeout)
{
	return UdsHttpClient(socket_path.string(), 1000ms, io_timeout,
			     64U * 1024U);
}

}

TEST_CASE("uds http client parses content-length and reuses keep-alive",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(
		temp.socket_path("cl.sock"), 1, [](int, int fd) {
			read_request_headers(fd);
			send_all(fd,
				 "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n"
				 "Connection: keep-alive\r\n\r\nfirst");
			read_request_headers(fd);
			send_all(fd,
				 "HTTP/1.1 200 OK\r\nContent-Length: 6\r\n"
				 "Connection: keep-alive\r\n\r\nsecond");
		});
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("cl.sock"));
	const auto first = client.get("/one");
	REQUIRE(first.has_value());
	CHECK(first->status == 200);
	CHECK(first->body == "first");

	const auto second = client.get("/two");
	REQUIRE(second.has_value());
	CHECK(second->body == "second");
	CHECK(server.accepted_count() == 1);
}

TEST_CASE("uds http client assembles chunked bodies with trailers",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(temp.socket_path("chunked.sock"), 1,
				 [](int, int fd) {
					 read_request_headers(fd);
					 send_all(fd,
						  "HTTP/1.1 200 OK\r\n"
						  "Transfer-Encoding: chunked\r\n"
						  "Connection: keep-alive\r\n\r\n"
						  "4\r\nWiki\r\n"
						  "5\r\npedia\r\n"
						  "E\r\n in\r\n\r\nchunks.\r\n"
						  "0\r\nX-Trailer: yes\r\n\r\n");
				 });
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("chunked.sock"));
	const auto result = client.get("/chunked");
	REQUIRE(result.has_value());
	CHECK(result->status == 200);
	CHECK(result->body == "Wikipedia in\r\n\r\nchunks.");
	CHECK(server.accepted_count() == 1);
}

TEST_CASE("uds http client keeps buffered bytes in eof-delimited bodies",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(temp.socket_path("eof.sock"), 1,
				 [](int, int fd) {
					 read_request_headers(fd);
					 send_all(fd,
						  "HTTP/1.1 200 OK\r\n"
						  "Connection: close\r\n\r\n"
						  "BODY-BYTES");
				 });
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("eof.sock"));
	const auto result = client.get("/eof");
	REQUIRE(result.has_value());
	CHECK(result->status == 200);
	CHECK(result->body == "BODY-BYTES");
	CHECK(server.accepted_count() == 1);
}

TEST_CASE("uds http client retries once on a stale keep-alive connection",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(
		temp.socket_path("stale.sock"), 2, [](int index, int fd) {
			read_request_headers(fd);
			if (index == 0) {
				send_all(fd,
					 "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n"
					 "Connection: keep-alive\r\n\r\nok");
			} else {
				send_all(fd,
					 "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n"
					 "Connection: keep-alive\r\n\r\nagain");
			}
		});
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("stale.sock"));
	REQUIRE(client.get("/first").has_value());
	const auto retried = client.get("/second");
	REQUIRE(retried.has_value());
	CHECK(retried->body == "again");
	CHECK(server.accepted_count() == 2);
}

TEST_CASE("uds http client stop wakes an in-flight request and recovers",
	  "[uds_http_client]")
{
	TempDirectory temp;
	std::atomic<bool> first_request_seen { false };
	std::atomic<bool> release_first { false };
	ScriptedUdsServer server(
		temp.socket_path("stop.sock"), 2, [&](int index, int fd) {
			read_request_headers(fd);
			if (index == 0) {
				first_request_seen.store(true);
				while (!release_first.load()) {
					std::this_thread::sleep_for(5ms);
				}
				return;
			}
			send_all(fd,
				 "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n"
				 "Connection: keep-alive\r\n\r\nok");
		});
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("stop.sock"), 5000ms);
	std::optional<UdsHttpClient::Response> blocked_result;
	std::thread requester([&] { blocked_result = client.get("/blocked"); });
	for (int i = 0; i < 400 && !first_request_seen.load(); ++i) {
		std::this_thread::sleep_for(5ms);
	}
	REQUIRE(first_request_seen.load());
	const auto stop_started = std::chrono::steady_clock::now();
	client.stop();
	requester.join();
	const auto stopped_after = std::chrono::steady_clock::now() - stop_started;

	CHECK_FALSE(blocked_result.has_value());
	CHECK(stopped_after < 2000ms);
	release_first.store(true);

	const auto recovered = client.get("/again");
	REQUIRE(recovered.has_value());
	CHECK(recovered->body == "ok");
}

TEST_CASE("uds http client rejects oversize content-length",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(temp.socket_path("oversize.sock"), 1,
				 [](int, int fd) {
					 read_request_headers(fd);
					 send_all(fd,
						  "HTTP/1.1 200 OK\r\n"
						  "Content-Length: 70000\r\n"
						  "Connection: keep-alive\r\n\r\n");
				 });
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("oversize.sock"));
	CHECK_FALSE(client.get("/big").has_value());
}

TEST_CASE("uds http client rejects cumulative oversized headers",
	  "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(temp.socket_path("headers.sock"), 1,
				 [](int, int fd) {
					 read_request_headers(fd);
					 std::string response =
						 "HTTP/1.1 200 OK\r\n";
					 for (int i = 0; i < 300; ++i) {
						 response +=
							 "X-Padding-Header: "
							 "0123456789012345678901234567890123456789"
							 "0123456789\r\n";
					 }
					 response += "Content-Length: 2\r\n\r\nok";
					 send_all(fd, response);
				 });
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("headers.sock"));
	CHECK_FALSE(client.get("/headers").has_value());
}

TEST_CASE("uds http client posts a json body with content length",
	  "[uds_http_client]")
{
	TempDirectory temp;
	std::string captured;
	ScriptedUdsServer server(temp.socket_path("post.sock"), 1,
				 [&captured](int, int fd) {
					 captured = read_request(fd);
					 const std::string body =
						 R"({"code":0})";
					 send_all(fd,
						  "HTTP/1.1 200 OK\r\n"
						  "Content-Length: " +
							  std::to_string(
								  body.size()) +
							  "\r\nConnection: "
							  "keep-alive\r\n\r\n" +
							  body);
				 });
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("post.sock"));
	const std::string body = R"({"region":"global"})";
	const auto response = client.post("/api/application/region", body);
	REQUIRE(response.has_value());
	CHECK(response->status == 200);
	CHECK(response->body == R"({"code":0})");
	server.stop();

	const auto header_end = captured.find("\r\n\r\n");
	REQUIRE(header_end != std::string::npos);
	const std::string headers = captured.substr(0, header_end + 4);
	CHECK(headers.starts_with(
		"POST /api/application/region HTTP/1.1\r\n"));
	CHECK(headers.find("\r\nContent-Type: application/json\r\n") !=
	      std::string::npos);
	CHECK(headers.find("\r\nContent-Length: " +
			   std::to_string(body.size()) + "\r\n") !=
	      std::string::npos);
	CHECK(captured.substr(header_end + 4) == body);
	CHECK(server.accepted_count() == 1);
}

TEST_CASE("uds http client does not retry a failed post", "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(temp.socket_path("post-no-retry.sock"), 1,
				 [](int, int fd) { read_request_headers(fd); });
	REQUIRE(server.start());

	auto client =
		make_client(temp.socket_path("post-no-retry.sock"), 5000ms);
	const auto started = std::chrono::steady_clock::now();
	const auto result =
		client.post("/api/application/region", R"({"region":"cn"})");
	const auto elapsed = std::chrono::steady_clock::now() - started;
	CHECK_FALSE(result.has_value());
	CHECK(elapsed < 2000ms);
	CHECK(server.accepted_count() == 1);
}

TEST_CASE("uds http client reconnects after a failed post", "[uds_http_client]")
{
	TempDirectory temp;
	ScriptedUdsServer server(
		temp.socket_path("post-reconnect.sock"), 2,
		[](int index, int fd) {
			if (index == 0) {
				read_request_headers(fd);
				return;
			}
			read_request(fd);
			send_all(fd,
				 "HTTP/1.1 200 OK\r\nContent-Length: 10\r\n"
				 "Connection: keep-alive\r\n\r\n{\"code\":0}");
		});
	REQUIRE(server.start());

	auto client = make_client(temp.socket_path("post-reconnect.sock"));
	CHECK_FALSE(
		client.post("/api/application/region", R"({"region":"cn"})")
			.has_value());
	const auto second =
		client.post("/api/application/region", R"({"region":"cn"})");
	REQUIRE(second.has_value());
	CHECK(second->status == 200);
	CHECK(second->body == R"({"code":0})");
	CHECK(server.accepted_count() == 2);
}
