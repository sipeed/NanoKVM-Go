#pragma once

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main
{
namespace detail
{

class UdsHttpClient final
{
public:
	struct Response final {
		int status { 0 };
		std::string body;
	};

	UdsHttpClient(std::string socket_path,
		      std::chrono::milliseconds connect_timeout,
		      std::chrono::milliseconds io_timeout,
		      std::size_t max_response_bytes);
	~UdsHttpClient();

	UdsHttpClient(const UdsHttpClient &) = delete;
	UdsHttpClient &operator=(const UdsHttpClient &) = delete;

	std::optional<Response> get(std::string_view path);
	std::optional<Response> post(std::string_view path,
				     std::string_view json_body);

	void stop();

	std::string_view last_error() const { return _last_error; }

private:
	class Reader;

	enum class ReadResult : uint8_t {
		Success,
		SuccessServerClose,
		Failure,
	};

	bool ensure_connected();
	bool send_all(std::string_view data);
	ReadResult read_response(Response &out, Reader &reader);
	bool wait_ready(int fd, short events);
	static int timeout_ms(std::chrono::milliseconds value);
	bool interrupted() const;
	void close_connection();
	void clear_stale_interrupt();
	std::optional<Response> request_once(std::string_view request,
					     bool &pre_response_failure);
	std::optional<Response> finish_request(std::optional<Response> result);

	std::string _socket_path;
	std::chrono::milliseconds _connect_timeout;
	std::chrono::milliseconds _io_timeout;
	std::size_t _max_response_bytes;
	int _fd { -1 };
	int _wake_fd { -1 };
	std::string _last_error;
};

}
}
