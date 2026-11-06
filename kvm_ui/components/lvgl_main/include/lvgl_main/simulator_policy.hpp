#pragma once

#include <filesystem>
#include <string>
#include <string_view>

#include "lvgl_main/app_config.hpp"

namespace lvgl_main::simulator_policy
{

bool is_protected_target_path(const std::filesystem::path &path);

bool resolve_config_path(const std::filesystem::path &requested_path,
			 bool explicitly_requested,
			 std::filesystem::path &resolved_path,
			 std::string &error);

std::filesystem::path launcher_apps_path();

AppConfig make_effective_config(const AppConfig &config);

} // namespace lvgl_main::simulator_policy
