#include "lvgl_main/app_catalog.hpp"

#include <algorithm>
#include <fstream>
#include <regex>
#include <utility>

#include <nlohmann/json.hpp>

namespace lvgl_main
{

std::string parse_app_manifest(const std::filesystem::path &file)
{
	std::ifstream in(file);
	if (!in) {
		return {};
	}

	try {
		const nlohmann::json info = nlohmann::json::parse(in);
		if (!info.is_object()) {
			return {};
		}
		const auto app_id = info.find("app_id");
		static const std::regex app_id_pattern(
			"^[a-z][a-z0-9]*(\\.[a-z][a-z0-9_]*)+$");
		if (app_id == info.end() || !app_id->is_string() ||
		    !std::regex_match(app_id->get<std::string>(),
				      app_id_pattern) ||
		    std::count(app_id->get_ref<const std::string &>().begin(),
			       app_id->get_ref<const std::string &>().end(),
			       '.') < 2) {
			return {};
		}
		std::string directory = file.parent_path().filename().string();
		std::replace(directory.begin(), directory.end(), '-', '_');
		const std::string package = app_id->get<std::string>();
		const auto separator = package.rfind('.');
		if (separator == std::string::npos ||
		    package.substr(separator + 1) != directory) {
			return {};
		}
		const auto name = info.find("name");
		if (name == info.end() || !name->is_string()) {
			return {};
		}
		std::string value = name->get<std::string>();
		const auto first = value.find_first_not_of(" \t\r\n");
		if (first == std::string::npos) {
			return {};
		}
		const auto last = value.find_last_not_of(" \t\r\n");
		return value.substr(first, last - first + 1);
	} catch (const nlohmann::json::exception &) {
		return {};
	}
}

bool parse_app_environment(const std::filesystem::path &file,
			   std::map<std::string, std::string> &environment,
			   std::string &error)
{
	environment.clear();
	error.clear();
	static const std::regex name_pattern("^[A-Za-z_][A-Za-z0-9_]*$");
	try {
		std::ifstream stream(file);
		if (!stream) {
			error = "cannot open app.json";
			return false;
		}
		nlohmann::json document;
		stream >> document;
		if (!document.is_object()) {
			error = "app.json must contain an object";
			return false;
		}
		const auto env_it = document.find("env");
		if (env_it == document.end()) {
			return true;
		}
		if (!env_it->is_object()) {
			error = "the env field in app.json must contain an object";
			return false;
		}
		for (auto it = env_it->begin(); it != env_it->end(); ++it) {
			if (!std::regex_match(it.key(), name_pattern) ||
			    !it.value().is_object()) {
				error = "invalid env entry in app.json";
				return false;
			}
			const auto &spec = it.value();
			const bool required = spec.value("required", false);
			std::string value = spec.value("value", std::string {});
			if (value.empty()) {
				value = spec.value("default", std::string {});
			}
			if (required && value.empty()) {
				error = "required App environment is not configured: " +
					it.key();
				return false;
			}
			if (!value.empty()) {
				environment.emplace(it.key(), std::move(value));
			}
		}
		return true;
	} catch (const std::exception &exception) {
		error = exception.what();
		return false;
	}
}

std::vector<ui::AppEntry> scan_apps(const std::filesystem::path &directory)
{
	std::vector<ui::AppEntry> apps;
	std::error_code ec;
	std::filesystem::directory_iterator it(directory, ec);
	if (ec) {
		return apps;
	}

	const std::filesystem::directory_iterator end;
	for (; it != end; it.increment(ec)) {
		if (ec) {
			break;
		}

		const auto &entry = *it;
		if (!entry.is_directory(ec) || ec) {
			continue;
		}
		const auto &app_dir = entry.path();
		const std::string directory_name = app_dir.filename().string();
		if (directory_name.empty() || directory_name.front() == '_') {
			continue;
		}

		const auto main_path = app_dir / "main.py";
		const auto info_path = app_dir / "app.json";
		if (!std::filesystem::is_regular_file(main_path, ec) || ec) {
			continue;
		}
		if (!std::filesystem::is_regular_file(info_path, ec) || ec) {
			continue;
		}

		std::string name = parse_app_manifest(info_path);
		if (name.empty()) {
			continue;
		}
		apps.push_back({ std::move(name), main_path.string() });
	}

	std::sort(apps.begin(), apps.end(),
		  [](const ui::AppEntry &lhs, const ui::AppEntry &rhs) {
			  return lhs.name < rhs.name;
		  });
	return apps;
}

} // namespace lvgl_main
