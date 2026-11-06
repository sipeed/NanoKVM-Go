#include <future>
#include <string>
#include <utility>

#include "lvgl_main/http_api_server.hpp"

#include <httplib.h>
#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>

#include "lvgl_main/nanokvm_runtime.hpp"

using json = nlohmann::json;

namespace
{

void set_json_response(httplib::Response &res, int status,
		       std::string_view error, std::string_view detail = {})
{
	res.status = status;
	json body { { "error", error } };
	if (!detail.empty()) {
		body["detail"] = detail;
	}
	res.set_content(body.dump(), "application/json");
}

void set_ok_response(httplib::Response &res)
{
	res.set_content(json { { "status", "ok" } }.dump(), "application/json");
}
} // namespace

HttpApiServer::HttpApiServer(lvgl_main::NanoKvmRuntimeService *runtime)
	: _runtime(runtime)
	, _svr(std::make_unique<httplib::Server>())
{
}

HttpApiServer::~HttpApiServer()
{
	stop();
}

bool HttpApiServer::start(int port)
{
	return start(lvgl_main::http_api::ServerConfig {
		.bind_host = "127.0.0.1",
		.port = port,
		.token = {},
	});
}

bool HttpApiServer::start(const lvgl_main::http_api::ServerConfig &config)
{
	if (_running.load(std::memory_order_acquire)) {
		return false;
	}

	const auto validation =
		lvgl_main::http_api::validate_server_config(config);
	if (!validation.allowed) {
		spdlog::error("HTTP API server refused configuration: {}",
			      validation.detail);
		return false;
	}

	const std::string auth_token = config.token;

	_svr->set_payload_max_length(
		lvgl_main::http_api::kMaxDisplayLabelsBodyBytes);

	_svr->Post("/api/v1/display/labels", [this, auth_token](
						     const httplib::Request &req,
						     httplib::Response &res) {
		try {
			if (!lvgl_main::http_api::is_authorized_bearer(
				    req.get_header_value("Authorization"),
				    auth_token)) {
				set_json_response(res, 401, "Unauthorized");
				return;
			}

			if (_runtime == nullptr) {
				set_json_response(res, 503,
						  "Service Unavailable");
				return;
			}

			const auto parsed =
				lvgl_main::http_api::parse_display_labels_payload(
					req.body);
			if (!parsed.ok()) {
				const char *error =
					parsed.status_code == 413 ?
						"Payload Too Large" :
						"Bad Request";
				set_json_response(res, parsed.status_code,
						  error, parsed.detail);
				return;
			}

			_runtime->update_display_labels(parsed.update);

			set_ok_response(res);
		} catch (const std::exception &e) {
			spdlog::error("HTTP API request handler failed: {}",
				      e.what());
			set_json_response(res, 500, "Internal Server Error");
		} catch (...) {
			spdlog::error(
				"HTTP API request handler failed with unknown error");
			set_json_response(res, 500, "Internal Server Error");
		}
	});

	_svr->Post("/api/v1/hid/state", [this, auth_token](
						const httplib::Request &req,
						httplib::Response &res) {
		try {
			if (!lvgl_main::http_api::is_authorized_bearer(
				    req.get_header_value("Authorization"),
				    auth_token)) {
				set_json_response(res, 401, "Unauthorized");
				return;
			}

			if (_runtime == nullptr) {
				set_json_response(res, 503,
						  "Service Unavailable");
				return;
			}

			if (!lvgl_main::http_api::is_json_content_type(
				    req.get_header_value("Content-Type"))) {
				set_json_response(
					res, 415, "Unsupported Media Type",
					"Content-Type must be application/json");
				return;
			}

			const auto parsed =
				lvgl_main::http_api::parse_hid_state_payload(
					req.body);
			if (!parsed.ok()) {
				const char *error =
					parsed.status_code == 413 ?
						"Payload Too Large" :
						"Bad Request";
				set_json_response(res, parsed.status_code,
						  error, parsed.detail);
				return;
			}

			_runtime->update_hid_state(parsed.update);

			set_ok_response(res);
		} catch (const std::exception &e) {
			spdlog::error("HTTP API request handler failed: {}",
				      e.what());
			set_json_response(res, 500, "Internal Server Error");
		} catch (...) {
			spdlog::error(
				"HTTP API request handler failed with unknown error");
			set_json_response(res, 500, "Internal Server Error");
		}
	});

#if NANOKVM_GO_ENABLE_BACKGROUND_IMAGE
	_svr->Post("/api/v1/background", [this, auth_token](
						 const httplib::Request &req,
						 httplib::Response &res) {
		try {
			if (!lvgl_main::http_api::is_authorized_bearer(
				    req.get_header_value("Authorization"),
				    auth_token)) {
				set_json_response(res, 401, "Unauthorized");
				return;
			}

			if (_runtime == nullptr) {
				set_json_response(res, 503,
						  "Service Unavailable");
				return;
			}

			if (!lvgl_main::http_api::is_json_content_type(
				    req.get_header_value("Content-Type"))) {
				set_json_response(
					res, 415, "Unsupported Media Type",
					"Content-Type must be application/json");
				return;
			}

			const auto parsed =
				lvgl_main::http_api::parse_background_payload(
					req.body);
			if (!parsed.ok()) {
				const char *error =
					parsed.status_code == 413 ?
						"Payload Too Large" :
						"Bad Request";
				set_json_response(res, parsed.status_code,
						  error, parsed.detail);
				return;
			}

			const auto result =
				_runtime->apply_background_selection(
					parsed.update.image_name);
			if (!result.ok) {
				set_json_response(
					res, result.http_status,
					result.http_status == 400 ?
						"Bad Request" :
						"Internal Server Error",
					result.detail);
				return;
			}

			res.set_content(json { { "status", "ok" },
					       { "image_name",
						 parsed.update.image_name } }
						.dump(),
					"application/json");
		} catch (const std::exception &e) {
			spdlog::error("HTTP API request handler failed: {}",
				      e.what());
			set_json_response(res, 500, "Internal Server Error");
		} catch (...) {
			spdlog::error(
				"HTTP API request handler failed with unknown error");
			set_json_response(res, 500, "Internal Server Error");
		}
	});
#endif

	_svr->set_error_handler(
		[](const httplib::Request &, httplib::Response &res) {
			if (res.status == 404) {
				set_json_response(res, 404, "Not Found");
				return;
			}
		});

	std::promise<bool> startup_result;
	auto startup_status = startup_result.get_future();

	_running.store(true, std::memory_order_release);
	_thread = std::thread(&HttpApiServer::run_loop, this, config.bind_host,
			      config.port, std::move(startup_result));

	bool started = false;
	try {
		started = startup_status.get();
	} catch (const std::future_error &e) {
		spdlog::error("HTTP API server startup status failed: {}",
			      e.what());
	}

	if (!started) {
		if (_thread.joinable()) {
			_thread.join();
		}
		if (_svr) {
			_svr->stop();
		}
		_running.store(false, std::memory_order_release);
		return false;
	}

	_svr->wait_until_ready();
	if (!_svr->is_running()) {
		if (_thread.joinable()) {
			_thread.join();
		}
		if (_svr) {
			_svr->stop();
		}
		_running.store(false, std::memory_order_release);
		return false;
	}

	return true;
}

void HttpApiServer::stop()
{
	if (_svr) {
		_svr->stop();
	}
	if (_thread.joinable()) {
		_thread.join();
	}
	_running.store(false, std::memory_order_release);
}

bool HttpApiServer::is_running() const
{
	return _running.load(std::memory_order_acquire);
}

void HttpApiServer::run_loop(std::string bind_host, int port,
			     std::promise<bool> startup_result)
{
	bool startup_reported = false;
	auto report_startup = [&startup_result, &startup_reported](bool ok) {
		if (!startup_reported) {
			startup_result.set_value(ok);
			startup_reported = true;
		}
	};

	try {
		spdlog::info("[http-api] HTTP API server starting on {}:{}",
			     bind_host, port);
		if (!_svr->bind_to_port(bind_host, port)) {
			spdlog::error(
				"HTTP API server failed to bind/listen on {}:{}",
				bind_host, port);
			report_startup(false);
		} else {
			report_startup(true);
			const bool ok = _svr->listen_after_bind();
			if (!ok) {
				spdlog::error(
					"HTTP API server listen loop failed on {}:{}",
					bind_host, port);
			}
		}
	} catch (const std::exception &e) {
		spdlog::error("HTTP API server failed on {}:{}: {}", bind_host,
			      port, e.what());
		report_startup(false);
	} catch (...) {
		spdlog::error(
			"HTTP API server failed on {}:{} with unknown error",
			bind_host, port);
		report_startup(false);
	}
	_running.store(false, std::memory_order_release);
	spdlog::info("[http-api] HTTP API server stopped");
}
