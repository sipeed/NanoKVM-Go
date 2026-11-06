#include "systemd_dbus_client.hpp"

#include <chrono>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <mutex>
#include <string>
#include <thread>

#include <systemd/sd-bus.h>

namespace lvgl_main::detail
{
namespace
{

class Client final {
    public:
	Client() = default;

	SystemdDbusResult query(std::string_view unit,
	                        SystemdDbusUnitState &state, std::string &error)
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (!ensure_bus(error)) {
			return SystemdDbusResult::Failed;
		}

		const auto deadline = Clock::now() + kStatusTimeout;
		if (query_unit_state_until(unit, deadline, state, error)) {
			return SystemdDbusResult::Success;
		}

		reset_bus();
		return SystemdDbusResult::Failed;
	}

	SystemdDbusResult set_enabled_now(std::string_view unit, bool enabled,
	                                  std::string &error)
	{
		std::lock_guard<std::mutex> lock(_mutex);
		if (!ensure_bus(error)) {
			return SystemdDbusResult::Failed;
		}

		const auto deadline = Clock::now() + kMutationTimeout;
		if (!set_call_timeout_until(deadline, error) ||
		    !change_unit_file_state(unit, enabled, error) ||
		    !set_call_timeout_until(deadline, error) ||
		    !reload_manager(error) ||
		    !set_call_timeout_until(deadline, error) ||
		    !start_or_stop_unit(unit, enabled, error) ||
		    !wait_for_runtime_state(unit, enabled, deadline, error)) {
			reset_bus();
			return SystemdDbusResult::Failed;
		}

		return SystemdDbusResult::Success;
	}

	~Client()
	{
		std::lock_guard<std::mutex> lock(_mutex);
		reset_bus();
	}

	Client(const Client &) = delete;
	Client &operator=(const Client &) = delete;

    private:
	using Clock = std::chrono::steady_clock;
	static constexpr auto kStatusTimeout = std::chrono::milliseconds {800};
	static constexpr auto kMutationTimeout = std::chrono::seconds {5};
	static constexpr auto kStatePollInterval =
	        std::chrono::milliseconds {20};
	static constexpr const char *kSystemdDestination =
	        "org.freedesktop.systemd1";
	static constexpr const char *kManagerPath = "/org/freedesktop/systemd1";
	static constexpr const char *kManagerInterface =
	        "org.freedesktop.systemd1.Manager";

	static std::string error_from_code(int code)
	{
		if (code >= 0) {
			return "unknown error";
		}
		const char *message = std::strerror(-code);
		return message == nullptr ? "unknown error" : message;
	}

	static std::string bus_error_message(const sd_bus_error &bus_error,
	                                     int code)
	{
		return bus_error.message == nullptr ? error_from_code(code)
		                                    : bus_error.message;
	}

	bool ensure_bus(std::string &error)
	{
		if (_bus != nullptr) {
			return true;
		}

		const int rc = sd_bus_open_system(&_bus);
		if (rc < 0 || _bus == nullptr) {
			error = "sd_bus_open_system failed: " +
			        error_from_code(rc);
			_bus = nullptr;
			return false;
		}
		return true;
	}

	bool set_call_timeout_until(Clock::time_point deadline,
	                            std::string &error)
	{
		const auto now = Clock::now();
		if (now >= deadline) {
			error = "systemd D-Bus operation timed out";
			return false;
		}
		auto remaining =
		        std::chrono::duration_cast<std::chrono::microseconds>(
		                deadline - now);
		if (remaining.count() <= 0) {
			remaining = std::chrono::microseconds {1};
		}
		const int rc = sd_bus_set_method_call_timeout(
		        _bus, static_cast<uint64_t>(remaining.count()));
		if (rc < 0) {
			error = "sd_bus_set_method_call_timeout failed: " +
			        error_from_code(rc);
			return false;
		}
		return true;
	}

	bool query_property(std::string_view unit, std::string_view property,
	                    std::string &value, std::string &error)
	{
		char *raw_value = nullptr;
		sd_bus_error bus_error = SD_BUS_ERROR_NULL;
		const std::string object_path = systemd_unit_object_path(unit);
		const int rc = sd_bus_get_property_string(
		        _bus, kSystemdDestination, object_path.c_str(),
		        "org.freedesktop.systemd1.Unit",
		        std::string(property).c_str(), &bus_error, &raw_value);
		if (rc < 0 || raw_value == nullptr) {
			error = std::string(property) + " query failed: " +
			        bus_error_message(bus_error, rc);
			if (raw_value != nullptr) {
				std::free(raw_value);
			}
			sd_bus_error_free(&bus_error);
			return false;
		}

		value.assign(raw_value);
		std::free(raw_value);
		sd_bus_error_free(&bus_error);
		return true;
	}

	bool query_unit_state_until(std::string_view unit,
	                            Clock::time_point deadline,
	                            SystemdDbusUnitState &state,
	                            std::string &error)
	{
		return set_call_timeout_until(deadline, error) &&
		       query_property(unit, "ActiveState", state.active_state,
		                      error) &&
		       set_call_timeout_until(deadline, error) &&
		       query_property(unit, "LoadState", state.load_state,
		                      error);
	}

	bool change_unit_file_state(std::string_view unit, bool enabled,
	                            std::string &error)
	{
		sd_bus_error bus_error = SD_BUS_ERROR_NULL;
		const std::string unit_name(unit);
		int rc = 0;
		if (enabled) {
			rc = sd_bus_call_method(_bus, kSystemdDestination,
			                      kManagerPath, kManagerInterface,
			                      "EnableUnitFiles", &bus_error,
			                      nullptr, "asbb", 1U,
			                      unit_name.c_str(), 0, 0);
		} else {
			rc = sd_bus_call_method(_bus, kSystemdDestination,
			                      kManagerPath, kManagerInterface,
			                      "DisableUnitFiles", &bus_error,
			                      nullptr, "asb", 1U,
			                      unit_name.c_str(), 0);
		}
		if (rc < 0) {
			error = std::string(enabled ? "EnableUnitFiles"
			                            : "DisableUnitFiles") +
			        " failed: " + bus_error_message(bus_error, rc);
			sd_bus_error_free(&bus_error);
			return false;
		}
		sd_bus_error_free(&bus_error);
		return true;
	}

	bool reload_manager(std::string &error)
	{
		sd_bus_error bus_error = SD_BUS_ERROR_NULL;
		const int rc = sd_bus_call_method(_bus, kSystemdDestination,
		                                kManagerPath, kManagerInterface,
		                                "Reload", &bus_error, nullptr,
		                                nullptr);
		if (rc < 0) {
			error = "Reload failed: " +
			        bus_error_message(bus_error, rc);
			sd_bus_error_free(&bus_error);
			return false;
		}
		sd_bus_error_free(&bus_error);
		return true;
	}

	bool start_or_stop_unit(std::string_view unit, bool enabled,
	                        std::string &error)
	{
		sd_bus_error bus_error = SD_BUS_ERROR_NULL;
		const std::string unit_name(unit);
		const int rc = sd_bus_call_method(
		        _bus, kSystemdDestination, kManagerPath,
		        kManagerInterface, enabled ? "StartUnit" : "StopUnit",
		        &bus_error, nullptr, "ss", unit_name.c_str(),
		        "replace");
		if (rc < 0) {
			error = std::string(enabled ? "StartUnit"
			                            : "StopUnit") +
			        " failed: " + bus_error_message(bus_error, rc);
			sd_bus_error_free(&bus_error);
			return false;
		}
		sd_bus_error_free(&bus_error);
		return true;
	}

	bool wait_for_runtime_state(std::string_view unit, bool enabled,
	                            Clock::time_point deadline,
	                            std::string &error)
	{
		while (Clock::now() < deadline) {
			SystemdDbusUnitState state;
			if (!query_unit_state_until(unit, deadline, state,
			                            error)) {
				return false;
			}
			if (!systemd_dbus_unit_exists(state)) {
				error = "systemd unit is not loaded";
				return false;
			}
			if (enabled && systemd_dbus_unit_is_active(state)) {
				return true;
			}
			if (!enabled && systemd_dbus_unit_is_stopped(state)) {
				return true;
			}
			if (enabled && state.active_state == "failed") {
				error = "systemd unit entered failed state";
				return false;
			}
			std::this_thread::sleep_for(kStatePollInterval);
		}
		error = "systemd unit state transition timed out";
		return false;
	}

	void reset_bus()
	{
		if (_bus != nullptr) {
			_bus = sd_bus_unref(_bus);
		}
	}

	std::mutex _mutex;
	sd_bus *_bus {nullptr};
};

Client &client()
{
	static Client instance;
	return instance;
}

bool is_object_path_safe_unit_char(unsigned char value)
{
	return (value >= 'a' && value <= 'z') ||
	       (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9');
}

} // namespace

std::string systemd_unit_object_path(std::string_view unit)
{
	static constexpr char hex[] = "0123456789abcdef";
	std::string path = "/org/freedesktop/systemd1/unit/";
	path.reserve(path.size() + unit.size());
	for (const unsigned char value : unit) {
		if (is_object_path_safe_unit_char(value)) {
			path.push_back(static_cast<char>(value));
			continue;
		}
		path.push_back('_');
		path.push_back(hex[(value >> 4U) & 0x0fU]);
		path.push_back(hex[value & 0x0fU]);
	}
	return path;
}

bool systemd_dbus_unit_is_active(const SystemdDbusUnitState &state) noexcept
{
	return state.load_state != "not-found" &&
	       (state.active_state == "active" ||
	        state.active_state == "reloading");
}

bool systemd_dbus_unit_exists(const SystemdDbusUnitState &state) noexcept
{
	return state.load_state != "not-found";
}

bool systemd_dbus_unit_is_stopped(const SystemdDbusUnitState &state) noexcept
{
	return systemd_dbus_unit_exists(state) &&
	       (state.active_state == "inactive" ||
	        state.active_state == "failed");
}

SystemdDbusResult query_systemd_unit_state(std::string_view unit,
                                           SystemdDbusUnitState &state,
                                           std::string &error)
{
	return client().query(unit, state, error);
}

SystemdDbusResult set_systemd_unit_enabled_now(std::string_view unit,
                                               bool enabled, std::string &error)
{
	return client().set_enabled_now(unit, enabled, error);
}

} // namespace lvgl_main::detail
