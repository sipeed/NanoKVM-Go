#pragma once

#include <atomic>
#include <future>
#include <memory>
#include <string>
#include <thread>

#include "lvgl_main/http_api_contract.hpp"

namespace httplib
{
class Server;
}

namespace lvgl_main
{
class NanoKvmRuntimeService;
}

class HttpApiServer {
    public:
	explicit HttpApiServer(lvgl_main::NanoKvmRuntimeService *runtime);
	~HttpApiServer();

	HttpApiServer(const HttpApiServer &) = delete;
	HttpApiServer &operator=(const HttpApiServer &) = delete;

	bool start(int port);
	bool start(const lvgl_main::http_api::ServerConfig &config);
	void stop();
	bool is_running() const;

    private:
	lvgl_main::NanoKvmRuntimeService *_runtime;
	std::unique_ptr<httplib::Server> _svr;
	std::thread _thread;
	std::atomic<bool> _running { false };

	void run_loop(std::string bind_host, int port,
		      std::promise<bool> startup_result);
};
