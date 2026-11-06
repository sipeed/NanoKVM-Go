#include <algorithm>
#include <array>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <filesystem>
#include <future>
#include <map>
#include <mutex>
#include <optional>
#include <stdexcept>
#include <string>
#include <string_view>
#include <system_error>
#include <thread>
#include <utility>
#include <vector>

#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config.hpp"
#include "nanokvm_server_transport.hpp"

using namespace std::chrono_literals;
using namespace lvgl_main;

namespace
{

constexpr std::string_view kCapturePath = "/api/vm/hdmi/capture";
constexpr std::string_view kWifiPath = "/api/network/wifi";
constexpr std::string_view kHidPath = "/api/hid/mode";
constexpr std::string_view kHardwarePath = "/api/vm/hardware";
constexpr std::string_view kStreamerPath = "/api/streamer/local";
constexpr std::string_view kRegionPath = "/api/application/region";

class TempDirectory final
{
    public:
	TempDirectory()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		const uint64_t id = next_id.fetch_add(1, std::memory_order_relaxed);
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_uds_transport_" + std::to_string(stamp) + "_" +
			 std::to_string(id));
		std::filesystem::create_directories(_path);
	}

	~TempDirectory()
	{
		std::error_code error;
		std::filesystem::remove_all(_path, error);
	}

	TempDirectory(const TempDirectory &) = delete;
	TempDirectory &operator=(const TempDirectory &) = delete;

	const std::filesystem::path &path() const { return _path; }

    private:
	std::filesystem::path _path;
};

class FakeUdsHttpServer final
{
    public:
	struct Reply final {
		int status { 200 };
		std::string body { R"({})" };
	};

	explicit FakeUdsHttpServer(std::filesystem::path socket_path)
		: _socket_path(std::move(socket_path))
	{
	}

	~FakeUdsHttpServer()
	{
		stop();
	}

	FakeUdsHttpServer(const FakeUdsHttpServer &) = delete;
	FakeUdsHttpServer &operator=(const FakeUdsHttpServer &) = delete;

	void set_reply(std::string path, Reply reply)
	{
		std::lock_guard lock(_mutex);
		_replies[std::move(path)] = std::move(reply);
	}

	void block_route(std::string path)
	{
		std::lock_guard lock(_mutex);
		_blocked_route = std::move(path);
		_blocked_request_seen = false;
		_block_responses = true;
	}

	void unblock_responses()
	{
		{
			std::lock_guard lock(_mutex);
			_block_responses = false;
		}
		_events.notify_all();
	}

	bool wait_for_blocked_request(std::chrono::milliseconds timeout)
	{
		std::unique_lock lock(_mutex);
		return _events.wait_for(lock, timeout,
					[_this = this] { return _this->_blocked_request_seen; });
	}

	bool wait_for_requests(std::string_view path, std::size_t expected,
				       std::chrono::milliseconds timeout)
	{
		std::unique_lock lock(_mutex);
		return _events.wait_for(lock, timeout, [this, path, expected] {
			const auto found = _request_counts.find(std::string(path));
			return found != _request_counts.end() && found->second >= expected;
		});
	}

	void start()
	{
		const std::string path = _socket_path.string();
		if (path.empty() || path.size() >= sizeof(sockaddr_un::sun_path)) {
			throw std::runtime_error("test UDS path is invalid");
		}

		std::error_code error;
		std::filesystem::remove(_socket_path, error);

		const int listener = ::socket(AF_UNIX, SOCK_STREAM, 0);
		if (listener < 0) {
			throw std::system_error(errno, std::generic_category(),
						"create test UDS listener");
		}

		sockaddr_un address {};
		address.sun_family = AF_UNIX;
		std::memcpy(address.sun_path, path.data(), path.size());
		address.sun_path[path.size()] = '\0';
		if (::bind(listener, reinterpret_cast<const sockaddr *>(&address),
			   sizeof(address)) != 0) {
			const int error_number = errno;
			::close(listener);
			throw std::system_error(error_number, std::generic_category(),
						"bind test UDS listener");
		}
		if (::listen(listener, 8) != 0) {
			const int error_number = errno;
			::close(listener);
			throw std::system_error(error_number, std::generic_category(),
						"listen on test UDS listener");
		}

		_listener_fd = listener;
		_running.store(true, std::memory_order_release);
		_accept_thread = std::thread(
			[this, listener] { accept_connections(listener); });
	}

	void stop()
	{
		if (!_running.exchange(false, std::memory_order_acq_rel)) {
			return;
		}

		unblock_responses();
		if (_listener_fd >= 0) {
			::shutdown(_listener_fd, SHUT_RDWR);
			::close(_listener_fd);
			_listener_fd = -1;
		}
		if (_accept_thread.joinable()) {
			_accept_thread.join();
		}

		std::vector<std::thread> clients;
		{
			std::lock_guard lock(_clients_mutex);
			for (const int fd : _client_fds) {
				::shutdown(fd, SHUT_RDWR);
			}
			clients = std::move(_client_threads);
		}
		for (std::thread &client : clients) {
			if (client.joinable()) {
				client.join();
			}
		}

		std::error_code error;
		std::filesystem::remove(_socket_path, error);
	}

	std::size_t connection_count() const
	{
		return _connection_count.load(std::memory_order_relaxed);
	}

	std::size_t request_count(std::string_view path) const
	{
		std::lock_guard lock(_mutex);
		const auto found = _request_counts.find(std::string(path));
		return found == _request_counts.end() ? 0 : found->second;
	}

	std::size_t get_request_count(std::string_view path) const
	{
		std::lock_guard lock(_mutex);
		const auto found = _get_request_counts.find(std::string(path));
		return found == _get_request_counts.end() ? 0 : found->second;
	}

	std::size_t post_request_count(std::string_view path) const
	{
		std::lock_guard lock(_mutex);
		const auto found = _post_request_counts.find(std::string(path));
		return found == _post_request_counts.end() ? 0 : found->second;
	}

	std::string last_method() const
	{
		std::lock_guard lock(_mutex);
		return _last_method;
	}

	std::string last_content_type() const
	{
		std::lock_guard lock(_mutex);
		return _last_content_type;
	}

	std::string last_body() const
	{
		std::lock_guard lock(_mutex);
		return _last_body;
	}

    private:
	static bool send_all(int fd, std::string_view response)
	{
		std::size_t sent = 0;
		while (sent < response.size()) {
			const ssize_t result = ::send(fd, response.data() + sent,
						      response.size() - sent, MSG_NOSIGNAL);
			if (result <= 0) {
				return false;
			}
			sent += static_cast<std::size_t>(result);
		}
		return true;
	}

	struct Request final {
		std::string method;
		std::string path;
		std::string content_type;
		std::string body;
	};

	static std::optional<std::string> header_value(std::string_view head,
						       std::string_view name)
	{
		std::string needle(name);
		needle.append(": ");
		const auto found = head.find(needle);
		if (found == std::string_view::npos) {
			return std::nullopt;
		}
		const auto value_start = found + needle.size();
		const auto value_end = head.find("\r\n", value_start);
		if (value_end == std::string_view::npos) {
			return std::string(head.substr(value_start));
		}
		return std::string(
			head.substr(value_start, value_end - value_start));
	}

	static std::size_t parse_content_length(std::string_view text)
	{
		std::size_t value = 0;
		for (const char digit : text) {
			if (digit < '0' || digit > '9') {
				break;
			}
			value = value * 10 + static_cast<std::size_t>(digit - '0');
		}
		return value;
	}

	static std::optional<Request> read_request(int fd, std::string &buffer)
	{
		constexpr std::string_view kHeadersEnd = "\r\n\r\n";
		while (true) {
			const std::size_t header_end = buffer.find(kHeadersEnd);
			if (header_end != std::string::npos) {
				const std::string head =
					buffer.substr(0, header_end);
				buffer.erase(0, header_end + kHeadersEnd.size());

				Request request;
				const std::size_t line_end = head.find("\r\n");
				const std::string_view line =
					std::string_view(head).substr(
						0, line_end == std::string::npos ?
							   head.size() :
							   line_end);
				std::size_t path_start = 0;
				if (line.starts_with("GET ")) {
					request.method = "GET";
					path_start = 4;
				} else if (line.starts_with("POST ")) {
					request.method = "POST";
					path_start = 5;
				} else {
					return std::nullopt;
				}
				const std::size_t path_end =
					line.find(' ', path_start);
				if (path_end == std::string_view::npos) {
					return std::nullopt;
				}
				request.path = std::string(line.substr(
					path_start, path_end - path_start));
				if (const auto content_type =
					    header_value(head, "Content-Type")) {
					request.content_type = *content_type;
				}
				std::size_t body_length = 0;
				if (const auto content_length = header_value(
					    head, "Content-Length")) {
					body_length =
						parse_content_length(*content_length);
				}
				while (buffer.size() < body_length) {
					char chunk[1024];
					const ssize_t received =
						::recv(fd, chunk, sizeof(chunk), 0);
					if (received <= 0) {
						return std::nullopt;
					}
					buffer.append(
						chunk,
						static_cast<std::size_t>(received));
				}
				request.body = buffer.substr(0, body_length);
				buffer.erase(0, body_length);
				return request;
			}

			char chunk[1024];
			const ssize_t received = ::recv(fd, chunk, sizeof(chunk), 0);
			if (received <= 0) {
				return std::nullopt;
			}
			buffer.append(chunk, static_cast<std::size_t>(received));
		}
	}

	void accept_connections(int listener)
	{
		while (_running.load(std::memory_order_acquire)) {
			const int client = ::accept(listener, nullptr, nullptr);
			if (client < 0) {
				if (errno == EINTR) {
					continue;
				}
				if (!_running.load(std::memory_order_acquire)) {
					break;
				}
				continue;
			}

			_connection_count.fetch_add(1, std::memory_order_relaxed);
			std::lock_guard lock(_clients_mutex);
			_client_fds.push_back(client);
			_client_threads.emplace_back(
				[this, client] { serve_client(client); });
		}
	}

	void serve_client(int client)
	{
		std::string request_buffer;
		while (_running.load(std::memory_order_acquire)) {
			const auto request = read_request(client, request_buffer);
			if (!request.has_value()) {
				break;
			}

			Reply reply;
			{
				std::unique_lock lock(_mutex);
				++_request_counts[request->path];
				if (request->method == "GET") {
					++_get_request_counts[request->path];
				} else if (request->method == "POST") {
					++_post_request_counts[request->path];
				}
				_last_method = request->method;
				_last_content_type = request->content_type;
				_last_body = request->body;
				_events.notify_all();
				if (request->path == _blocked_route &&
				    _block_responses) {
					_blocked_request_seen = true;
					_events.notify_all();
					_events.wait(lock, [this] {
						return !_block_responses ||
						       !_running.load(
							       std::memory_order_acquire);
					});
				}
				const auto found = _replies.find(request->path);
				if (found != _replies.end()) {
					reply = found->second;
				} else {
					reply.status = 404;
					reply.body = R"({"error":"not found"})";
				}
			}

			const std::string response =
				"HTTP/1.1 " + std::to_string(reply.status) +
				(reply.status >= 200 && reply.status < 300 ? " OK\r\n" : " Error\r\n") +
				"Content-Type: application/json\r\n"
				"Content-Length: " + std::to_string(reply.body.size()) + "\r\n"
				"Connection: keep-alive\r\n"
				"Keep-Alive: timeout=5, max=100\r\n\r\n" + reply.body;
			if (!send_all(client, response)) {
				break;
			}
		}

		{
			std::lock_guard lock(_clients_mutex);
			const auto found = std::find(_client_fds.begin(), _client_fds.end(), client);
			if (found != _client_fds.end()) {
				_client_fds.erase(found);
			}
		}
		::close(client);
	}

	std::filesystem::path _socket_path;
	std::atomic<bool> _running { false };
	std::atomic<std::size_t> _connection_count { 0 };
	int _listener_fd { -1 };
	std::thread _accept_thread;
	mutable std::mutex _mutex;
	std::condition_variable _events;
	std::map<std::string, Reply, std::less<>> _replies;
	std::map<std::string, std::size_t, std::less<>> _request_counts;
	std::map<std::string, std::size_t, std::less<>> _get_request_counts;
	std::map<std::string, std::size_t, std::less<>> _post_request_counts;
	std::string _last_method;
	std::string _last_content_type;
	std::string _last_body;
	std::string _blocked_route;
	bool _blocked_request_seen { false };
	bool _block_responses { false };
	std::mutex _clients_mutex;
	std::vector<int> _client_fds;
	std::vector<std::thread> _client_threads;
};

NanokvmRuntimeConfig local_config()
{
	return NanokvmRuntimeConfig {};
}

}

TEST_CASE("NanoKVM UDS transport adopts valid JSON without interpreting its domain")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
		}, socket_path, 1ms);

	const auto first = transport.get_json(NanoKvmServerTransportLane::Status,
						      kCapturePath);
	REQUIRE(first);
	CHECK(first->at("source") == "uds");
	CHECK(transport.is_uds_adopted());

	const auto second = transport.get_json(NanoKvmServerTransportLane::Status,
						       kCapturePath);
	REQUIRE(second);
	CHECK(second->at("source") == "uds");
	CHECK(server.request_count(kCapturePath) == 2);
	CHECK(server.connection_count() == 1);
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport has independent status and metrics lanes")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"status"})" });
	server.set_reply(std::string(kStreamerPath),
			 { 200, R"({"source":"metrics"})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
		}, socket_path, 1ms);

	const auto status = transport.get_json(NanoKvmServerTransportLane::Status,
						       kCapturePath);
	REQUIRE(status);
	const auto metrics = transport.get_json(NanoKvmServerTransportLane::Metrics,
								kStreamerPath);
	REQUIRE(metrics);
	CHECK(status->at("source") == "status");
	CHECK(metrics->at("source") == "metrics");
	CHECK(transport.is_uds_adopted());
	CHECK(server.connection_count() == 2);
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport metrics lane progresses while status is blocked")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"status"})" });
	server.set_reply(std::string(kStreamerPath),
			 { 200, R"({"source":"metrics"})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		}, socket_path, 1ms);

	REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status,
					   kCapturePath));
	REQUIRE(transport.is_uds_adopted());
	server.block_route(std::string(kCapturePath));

	auto blocked_status = std::async(std::launch::async, [&transport] {
		return transport.get_json(NanoKvmServerTransportLane::Status,
					  kCapturePath);
	});
	REQUIRE(server.wait_for_blocked_request(500ms));

	auto metrics = std::async(std::launch::async, [&transport] {
		return transport.get_json(NanoKvmServerTransportLane::Metrics,
						  kStreamerPath);
	});
	REQUIRE(metrics.wait_for(500ms) == std::future_status::ready);
	const auto metrics_response = metrics.get();
	REQUIRE(metrics_response);
	CHECK(metrics_response->at("source") == "metrics");

	server.unblock_responses();
	REQUIRE(blocked_status.wait_for(500ms) == std::future_status::ready);
	REQUIRE(blocked_status.get());
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport status lane progresses while metrics is blocked")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"status"})" });
	server.set_reply(std::string(kStreamerPath),
			 { 200, R"({"source":"metrics"})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		}, socket_path, 1ms);

	REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status,
				   kCapturePath));
	REQUIRE(transport.is_uds_adopted());
	server.block_route(std::string(kStreamerPath));

	auto blocked_metrics = std::async(std::launch::async, [&transport] {
		return transport.get_json(NanoKvmServerTransportLane::Metrics,
					  kStreamerPath);
	});
	REQUIRE(server.wait_for_blocked_request(500ms));

	auto status = std::async(std::launch::async, [&transport] {
		return transport.get_json(NanoKvmServerTransportLane::Status,
					  kCapturePath);
	});
	REQUIRE(status.wait_for(500ms) == std::future_status::ready);
	const auto status_response = status.get();
	REQUIRE(status_response);
	CHECK(status_response->at("source") == "status");

	server.unblock_responses();
	REQUIRE(blocked_metrics.wait_for(500ms) == std::future_status::ready);
	REQUIRE(blocked_metrics.get());
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE(
	"NanoKVM UDS transport falls back before adoption and remains sticky after it")
{
	SECTION("a failed local response uses the legacy adapter")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath), { 503, R"({"error":"busy"})" });
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
			}, socket_path, 1ms);

		const auto response = transport.get_json(
			NanoKvmServerTransportLane::Status, kCapturePath);
		REQUIRE(response);
		CHECK(response->at("source") == "legacy");
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(legacy_calls.load() == 1);
	}

	SECTION("a later local failure does not return to legacy after adoption")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
			}, socket_path, 1ms);

		REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status,
						   kCapturePath));
		REQUIRE(transport.is_uds_adopted());
		server.set_reply(std::string(kCapturePath), { 503, R"({"error":"down"})" });

		CHECK_FALSE(transport.get_json(NanoKvmServerTransportLane::Status,
						       kCapturePath));
		CHECK(legacy_calls.load() == 0);
	}

	SECTION("a later malformed response does not return to legacy after adoption")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json { { "source", "legacy" } }
				};
			}, socket_path, 1ms);

		REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status,
						   kCapturePath));
		REQUIRE(transport.is_uds_adopted());
		server.set_reply(std::string(kCapturePath), { 200, "{not-json" });

		CHECK_FALSE(transport.get_json(NanoKvmServerTransportLane::Status,
						       kCapturePath));
		CHECK(legacy_calls.load() == 0);
	}
}

TEST_CASE("NanoKVM UDS transport permits only the fixed method and path pairs")
{
	struct Route final {
		std::string_view path;
		NanoKvmServerTransportLane lane;
	};
	constexpr std::array<Route, 5> kRoutes {
		Route { kCapturePath, NanoKvmServerTransportLane::Status },
		Route { kWifiPath, NanoKvmServerTransportLane::Status },
		Route { kHidPath, NanoKvmServerTransportLane::Status },
		Route { kHardwarePath, NanoKvmServerTransportLane::Status },
		Route { kStreamerPath, NanoKvmServerTransportLane::Metrics },
	};

	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	for (const Route &route : kRoutes) {
		server.set_reply(
			std::string(route.path),
			{ 200, "{\"path\":\"" + std::string(route.path) + "\"}" });
	}
	server.start();

	std::atomic<int> legacy_calls { 0 };
	std::atomic<int> legacy_post_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		}, socket_path, 1ms,
		[&legacy_post_calls](std::string_view,
				     const nlohmann::json &) {
			++legacy_post_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		});

	for (const Route &route : kRoutes) {
		const auto response = transport.get_json(route.lane, route.path);
		REQUIRE(response);
		CHECK(response->at("path") == route.path);
		CHECK(server.request_count(route.path) == 1);
		CHECK(server.get_request_count(route.path) == 1);
		CHECK(server.post_request_count(route.path) == 0);
	}
	CHECK(transport.is_uds_adopted());
	CHECK(server.connection_count() == 2);
	CHECK(legacy_calls.load() == 0);

	CHECK_FALSE(transport.get_json(NanoKvmServerTransportLane::Status,
				       kRegionPath));
	for (const Route &route : kRoutes) {
		CHECK_FALSE(transport.post_json(
			NanoKvmServerTransportLane::Status, route.path,
			nlohmann::json { { "region", "cn" } }));
		CHECK(server.request_count(route.path) == 1);
		CHECK(server.post_request_count(route.path) == 0);
	}
	CHECK_FALSE(transport.get_json(NanoKvmServerTransportLane::Status,
				       "/api/unknown"));
	CHECK_FALSE(transport.post_json(
		NanoKvmServerTransportLane::Status, "/api/unknown",
		nlohmann::json { { "region", "cn" } }));
	CHECK(server.request_count(kRegionPath) == 0);
	CHECK(server.post_request_count(kRegionPath) == 0);
	CHECK(server.request_count("/api/unknown") == 0);
	CHECK(legacy_calls.load() == 0);
	CHECK(legacy_post_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport rejects invalid local payloads before adoption")
{
	SECTION("malformed JSON falls back to legacy")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath), { 200, "{not-json" });
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json { { "source", "legacy" } }
				};
			}, socket_path, 1ms);

		const auto response = transport.get_json(
			NanoKvmServerTransportLane::Status, kCapturePath);
		REQUIRE(response);
		CHECK(response->at("source") == "legacy");
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(legacy_calls.load() == 1);
	}

	SECTION("oversized payload falls back to legacy")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath),
				 { 200, std::string(70U * 1024U, 'x') });
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json { { "source", "legacy" } }
				};
			}, socket_path, 1ms);

		const auto response = transport.get_json(
			NanoKvmServerTransportLane::Status, kCapturePath);
		REQUIRE(response);
		CHECK(response->at("source") == "legacy");
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(legacy_calls.load() == 1);
	}

	SECTION("read timeout falls back to legacy")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path = temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
		server.block_route(std::string(kCapturePath));
		server.start();

		std::atomic<int> legacy_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(), [&legacy_calls](std::string_view) {
				++legacy_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json { { "source", "legacy" } }
				};
			}, socket_path, 1ms);

		auto response = std::async(std::launch::async, [&transport] {
			return transport.get_json(NanoKvmServerTransportLane::Status,
						  kCapturePath);
		});
		REQUIRE(server.wait_for_blocked_request(500ms));
		REQUIRE(response.wait_for(3s) == std::future_status::ready);
		server.unblock_responses();
		const auto result = response.get();
		REQUIRE(result);
		CHECK(result->at("source") == "legacy");
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(legacy_calls.load() == 1);
	}
}

TEST_CASE("NanoKVM UDS transport keeps a bounded probe cadence before socket adoption")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
		}, socket_path, 50ms);

	REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status, kCapturePath));
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
	server.start();

	const auto before_probe = transport.get_json(
		NanoKvmServerTransportLane::Status, kCapturePath);
	REQUIRE(before_probe);
	CHECK(before_probe->at("source") == "legacy");
	CHECK(server.request_count(kCapturePath) == 0);

	std::this_thread::sleep_for(75ms);
	const auto after_probe = transport.get_json(
		NanoKvmServerTransportLane::Status, kCapturePath);
	REQUIRE(after_probe);
	CHECK(after_probe->at("source") == "uds");
	CHECK(transport.is_uds_adopted());
	CHECK(legacy_calls.load() == 2);
}

TEST_CASE("NanoKVM UDS transport can start again after a completed stop")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		}, socket_path, 1ms);

	REQUIRE(transport.get_json(NanoKvmServerTransportLane::Status,
				   kCapturePath));
	REQUIRE(transport.is_uds_adopted());
	transport.stop();
	CHECK_FALSE(transport.get_json(NanoKvmServerTransportLane::Status,
				      kCapturePath));

	transport.start();
	const auto restarted = transport.get_json(
		NanoKvmServerTransportLane::Status, kCapturePath);
	REQUIRE(restarted);
	CHECK(restarted->at("source") == "uds");
	CHECK(transport.is_uds_adopted());
	CHECK(server.request_count(kCapturePath) == 2);
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport only uses local transport for the default server URL")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
	server.start();

	NanokvmRuntimeConfig config = local_config();
	config.server_base_url = "https://remote.example";
	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		config, [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
		}, socket_path, 1ms);

	const auto response = transport.get_json(NanoKvmServerTransportLane::Status,
						 kCapturePath);
	REQUIRE(response);
	CHECK(response->at("source") == "legacy");
	CHECK_FALSE(transport.is_eligible());
	CHECK_FALSE(transport.is_uds_adopted());
	CHECK(server.request_count(kCapturePath) == 0);
	CHECK(legacy_calls.load() == 1);
}

TEST_CASE("NanoKVM UDS transport stop wakes an in-flight lane without legacy fallback")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kCapturePath), { 200, R"({"source":"uds"})" });
	server.block_route(std::string(kCapturePath));
	server.start();

	std::atomic<int> legacy_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(), [&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> { nlohmann::json { { "source", "legacy" } } };
		}, socket_path, 1ms);

	auto request = std::async(std::launch::async, [&transport] {
		return transport.get_json(NanoKvmServerTransportLane::Status,
					  kCapturePath);
	});
	REQUIRE(server.wait_for_blocked_request(500ms));

	auto first_stop = std::async(std::launch::async, [&transport] {
		transport.stop();
	});
	auto second_stop = std::async(std::launch::async, [&transport] {
		transport.stop();
	});
	REQUIRE(first_stop.wait_for(500ms) == std::future_status::ready);
	REQUIRE(second_stop.wait_for(500ms) == std::future_status::ready);
	first_stop.get();
	second_stop.get();
	server.unblock_responses();
	REQUIRE(request.wait_for(500ms) == std::future_status::ready);
	CHECK_FALSE(request.get());
	CHECK(legacy_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport posts the region switch as JSON")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kRegionPath), { 200, R"({"code":0})" });
	server.start();

	std::atomic<int> legacy_calls { 0 };
	std::atomic<int> legacy_post_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(),
		[&legacy_calls](std::string_view) {
			++legacy_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		},
		socket_path, 1ms,
		[&legacy_post_calls](std::string_view,
				     const nlohmann::json &) {
			++legacy_post_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		});

	const nlohmann::json body { { "region", "global" } };
	const auto response = transport.post_json(
		NanoKvmServerTransportLane::Status, kRegionPath, body);
	REQUIRE(response);
	CHECK(response->at("code") == 0);
	CHECK(transport.is_uds_adopted());
	CHECK(server.request_count(kRegionPath) == 1);
	CHECK(server.get_request_count(kRegionPath) == 0);
	CHECK(server.post_request_count(kRegionPath) == 1);
	CHECK(server.last_method() == "POST");
	CHECK(server.last_content_type() == "application/json");
	CHECK(server.last_body() == body.dump());
	CHECK(legacy_calls.load() == 0);
	CHECK(legacy_post_calls.load() == 0);
}

TEST_CASE("NanoKVM UDS transport never falls back to legacy for a failed POST")
{
	SECTION("a probing POST with a non-2xx response is rejected")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kRegionPath),
				 { 503, R"({"error":"busy"})" });
		server.start();

		std::atomic<int> legacy_post_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(),
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms,
			[&legacy_post_calls](std::string_view,
					     const nlohmann::json &) {
				++legacy_post_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json {
						{ "source", "legacy" }
					}
				};
			});

		CHECK_FALSE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath,
			nlohmann::json { { "region", "cn" } }));
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(server.post_request_count(kRegionPath) == 1);
		CHECK(legacy_post_calls.load() == 0);
	}

	SECTION("a non-2xx POST after adoption is rejected")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kRegionPath),
				 { 200, R"({"code":0})" });
		server.start();

		std::atomic<int> legacy_post_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(),
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms,
			[&legacy_post_calls](std::string_view,
					     const nlohmann::json &) {
				++legacy_post_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json {
						{ "source", "legacy" }
					}
				};
			});

		const nlohmann::json body { { "region", "cn" } };
		REQUIRE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath, body));
		REQUIRE(transport.is_uds_adopted());
		server.set_reply(std::string(kRegionPath),
				 { 503, R"({"error":"down"})" });

		CHECK_FALSE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath, body));
		CHECK(legacy_post_calls.load() == 0);
	}

	SECTION("a malformed POST response after adoption is rejected")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kRegionPath),
				 { 200, R"({"code":0})" });
		server.start();

		std::atomic<int> legacy_post_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(),
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms,
			[&legacy_post_calls](std::string_view,
					     const nlohmann::json &) {
				++legacy_post_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json {
						{ "source", "legacy" }
					}
				};
			});

		const nlohmann::json body { { "region", "cn" } };
		REQUIRE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath, body));
		REQUIRE(transport.is_uds_adopted());
		server.set_reply(std::string(kRegionPath), { 200, "{not-json" });

		CHECK_FALSE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath, body));
		CHECK(legacy_post_calls.load() == 0);
	}

	SECTION("a POST read timeout after adoption is rejected")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kRegionPath),
				 { 200, R"({"code":0})" });
		server.start();

		std::atomic<int> legacy_post_calls { 0 };
		NanoKvmServerTransport transport(
			local_config(),
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms,
			[&legacy_post_calls](std::string_view,
					     const nlohmann::json &) {
				++legacy_post_calls;
				return std::optional<nlohmann::json> {
					nlohmann::json {
						{ "source", "legacy" }
					}
				};
			});

		const nlohmann::json body { { "region", "cn" } };
		REQUIRE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath, body));
		REQUIRE(transport.is_uds_adopted());
		server.block_route(std::string(kRegionPath));

		auto response = std::async(std::launch::async,
					   [&transport, &body] {
						   return transport.post_json(
							   NanoKvmServerTransportLane::Status,
							   kRegionPath, body);
					   });
		REQUIRE(server.wait_for_blocked_request(500ms));
		REQUIRE(response.wait_for(3s) == std::future_status::ready);
		server.unblock_responses();
		CHECK_FALSE(response.get());
		CHECK(legacy_post_calls.load() == 0);
	}
}

TEST_CASE("NanoKVM UDS transport uses the legacy POST only when UDS "
	  "is not eligible")
{
	SECTION("a non-local server URL posts once through the legacy adapter")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		FakeUdsHttpServer server(socket_path);
		server.set_reply(std::string(kRegionPath),
				 { 200, R"({"code":0})" });
		server.start();

		NanokvmRuntimeConfig config = local_config();
		config.server_base_url = "http://example.invalid";
		std::atomic<int> legacy_post_calls { 0 };
		std::string seen_path;
		nlohmann::json seen_body;
		NanoKvmServerTransport transport(
			config,
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms,
			[&legacy_post_calls, &seen_path,
			 &seen_body](std::string_view path,
				     const nlohmann::json &body) {
				++legacy_post_calls;
				seen_path = std::string(path);
				seen_body = body;
				return std::optional<nlohmann::json> {
					nlohmann::json {
						{ "source", "legacy" }
					}
				};
			});

		const auto response = transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath,
			nlohmann::json { { "region", "cn" } });
		REQUIRE(response);
		CHECK(response->at("source") == "legacy");
		CHECK_FALSE(transport.is_eligible());
		CHECK_FALSE(transport.is_uds_adopted());
		CHECK(server.request_count(kRegionPath) == 0);
		CHECK(legacy_post_calls.load() == 1);
		CHECK(seen_path == std::string(kRegionPath));
		CHECK(seen_body == nlohmann::json { { "region", "cn" } });
	}

	SECTION("an absent legacy POST returns nothing without crashing")
	{
		TempDirectory temp;
		const std::filesystem::path socket_path =
			temp.path() / "server.sock";
		NanokvmRuntimeConfig config = local_config();
		config.server_base_url = "http://example.invalid";
		NanoKvmServerTransport transport(
			config,
			[](std::string_view) {
				return std::optional<nlohmann::json> {};
			},
			socket_path, 1ms);

		CHECK_FALSE(transport.post_json(
			NanoKvmServerTransportLane::Status, kRegionPath,
			nlohmann::json { { "region", "cn" } }));
	}
}

TEST_CASE("NanoKVM UDS transport rejects POST after stop")
{
	TempDirectory temp;
	const std::filesystem::path socket_path = temp.path() / "server.sock";
	FakeUdsHttpServer server(socket_path);
	server.set_reply(std::string(kRegionPath), { 200, R"({"code":0})" });
	server.start();

	std::atomic<int> legacy_post_calls { 0 };
	NanoKvmServerTransport transport(
		local_config(),
		[](std::string_view) {
			return std::optional<nlohmann::json> {};
		},
		socket_path, 1ms,
		[&legacy_post_calls](std::string_view,
				     const nlohmann::json &) {
			++legacy_post_calls;
			return std::optional<nlohmann::json> {
				nlohmann::json { { "source", "legacy" } }
			};
		});

	REQUIRE(transport.post_json(NanoKvmServerTransportLane::Status,
				    kRegionPath,
				    nlohmann::json { { "region", "cn" } }));
	transport.stop();
	CHECK_FALSE(transport.post_json(
		NanoKvmServerTransportLane::Status, kRegionPath,
		nlohmann::json { { "region", "cn" } }));
	CHECK(server.post_request_count(kRegionPath) == 1);
	CHECK(legacy_post_calls.load() == 0);
}
