#pragma once

#include <filesystem>
#include <map>
#include <string>
#include <vector>

#include "ui/apps_page.hpp"

namespace lvgl_main
{

std::string parse_app_manifest(const std::filesystem::path &file);
bool parse_app_environment(const std::filesystem::path &file,
			   std::map<std::string, std::string> &environment,
			   std::string &error);
std::vector<ui::AppEntry> scan_apps(const std::filesystem::path &directory);

} // namespace lvgl_main
