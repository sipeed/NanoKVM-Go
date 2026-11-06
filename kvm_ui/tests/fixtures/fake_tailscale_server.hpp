#pragma once

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <filesystem>
#include <map>
#include <mutex>
#include <optional>
#include <stdexcept>
#include <string>
#include <string_view>
#include <system_error>
#include <thread>
#include <vector>

#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

namespace tests
{

class TempDirectory final {
    public:
	TempDirectory()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		const uint64_t id =
			next_id.fetch_add(1, std::memory_order_relaxed);
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_tailscale_" + std::to_string(stamp) + "_" +
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

class FakeTailscaleServer final {
    public:
	struct Reply final {
		int status { 200 };
		std::string body { R"({})" };
		std::chrono::milliseconds delay { 0 };
	};

	explicit FakeTailscaleServer(std::filesystem::path socket_path)
		: _socket_path(std::move(socket_path))
	{
	}

	~FakeTailscaleServer() { stop(); }

	FakeTailscaleServer(const FakeTailscaleServer &) = delete;
	FakeTailscaleServer &operator=(const FakeTailscaleServer &) = delete;

	void set_reply(std::string path, Reply reply)
	{
		std::lock_guard lock(_mutex);
		_replies[std::move(path)] = std::move(reply);
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
			throw std::system_error(error_number,
						std::generic_category(),
						"bind test UDS listener");
		}
		if (::listen(listener, 8) != 0) {
			const int error_number = errno;
			::close(listener);
			throw std::system_error(error_number,
						std::generic_category(),
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

	std::vector<std::string> request_log() const
	{
		std::lock_guard lock(_mutex);
		return _request_log;
	}

    private:
	static bool send_all(int fd, std::string_view data)
	{
		std::size_t sent = 0;
		while (sent < data.size()) {
			const ssize_t result =
				::send(fd, data.data() + sent,
				       data.size() - sent, MSG_NOSIGNAL);
			if (result <= 0) {
				return false;
			}
			sent += static_cast<std::size_t>(result);
		}
		return true;
	}

	static std::optional<std::string> read_request_path(int fd,
							    std::string &buffer)
	{
		constexpr std::string_view kHeadersEnd = "\r\n\r\n";
		for (;;) {
			const std::size_t header_end = buffer.find(kHeadersEnd);
			if (header_end != std::string::npos) {
				const std::string request =
					buffer.substr(0, header_end);
				buffer.erase(0,
					     header_end + kHeadersEnd.size());
				const std::size_t line_end =
					request.find("\r\n");
				const std::string_view line =
					std::string_view(request).substr(
						0, line_end == std::string::npos ?
							   request.size() :
							   line_end);
				constexpr std::string_view kGetPrefix = "GET ";
				if (!line.starts_with(kGetPrefix)) {
					return std::nullopt;
				}
				const std::size_t path_end =
					line.find(' ', kGetPrefix.size());
				if (path_end == std::string_view::npos) {
					return std::nullopt;
				}
				return std::string(line.substr(
					kGetPrefix.size(),
					path_end - kGetPrefix.size()));
			}

			char chunk[1024];
			const ssize_t received =
				::recv(fd, chunk, sizeof(chunk), 0);
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
			const auto path =
				read_request_path(client, request_buffer);
			if (!path.has_value()) {
				break;
			}

			Reply reply;
			{
				std::lock_guard lock(_mutex);
				++_request_counts[*path];
				_request_log.push_back(*path);
				const auto found = _replies.find(*path);
				if (found != _replies.end()) {
					reply = found->second;
				} else {
					reply.status = 404;
					reply.body = R"({"error":"not found"})";
				}
			}

			const std::string response =
				"HTTP/1.1 " + std::to_string(reply.status) +
				(reply.status >= 200 && reply.status < 300 ?
					 " OK\r\n" :
					 " Error\r\n") +
				"Content-Type: application/json\r\n"
				"Content-Length: " +
				std::to_string(reply.body.size()) +
				"\r\n"
				"Connection: keep-alive\r\n\r\n" +
				reply.body;
			if (reply.delay.count() > 0) {
				std::this_thread::sleep_for(reply.delay);
			}
			if (!send_all(client, response)) {
				break;
			}
		}

		{
			std::lock_guard lock(_clients_mutex);
			const auto found = std::find(_client_fds.begin(),
						     _client_fds.end(), client);
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
	std::map<std::string, Reply, std::less<>> _replies;
	std::map<std::string, std::size_t, std::less<>> _request_counts;
	std::vector<std::string> _request_log;
	std::mutex _clients_mutex;
	std::vector<int> _client_fds;
	std::vector<std::thread> _client_threads;
};

} // namespace tests
