#pragma once

#include <string>
#include <string_view>

namespace lvgl_main::detail
{

enum class SystemdDbusResult {
	Success,
	Failed,
};

struct SystemdDbusUnitState {
	std::string active_state;
	std::string load_state;
};

bool systemd_dbus_unit_is_active(const SystemdDbusUnitState &state) noexcept;
bool systemd_dbus_unit_exists(const SystemdDbusUnitState &state) noexcept;
bool systemd_dbus_unit_is_stopped(const SystemdDbusUnitState &state) noexcept;

std::string systemd_unit_object_path(std::string_view unit);

SystemdDbusResult query_systemd_unit_state(std::string_view unit,
                                           SystemdDbusUnitState &state,
                                           std::string &error);

SystemdDbusResult set_systemd_unit_enabled_now(std::string_view unit,
                                               bool enabled,
                                               std::string &error);

} // namespace lvgl_main::detail
