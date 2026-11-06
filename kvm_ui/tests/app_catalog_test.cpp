#include <chrono>
#include <filesystem>
#include <fstream>
#include <string>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_catalog.hpp"
#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_build_config.hpp"
#endif

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_app_catalog_test_" +
			 std::to_string(stamp));
		std::filesystem::create_directories(_path);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const
	{
		return _path;
	}

    private:
	std::filesystem::path _path;
};

void write_text(const std::filesystem::path &path, const std::string &text)
{
	std::ofstream output(path, std::ios::trunc);
	REQUIRE(output.good());
	output << text;
	REQUIRE(output.good());
}

} // namespace

TEST_CASE("app catalog preserves the Apps discovery contract", "[app-catalog]")
{
	TempDir temp;
	for (const char *name : { "zeta", "alpha", "_private", "missing_info",
				  "missing_main", "invalid_info" }) {
		std::filesystem::create_directory(temp.path() / name);
	}
	write_text(temp.path() / "zeta" / "main.py", "# app code\n");
	write_text(temp.path() / "zeta" / "app.json",
		   R"({"app_id":"com.sipeed.zeta","name":"Zeta"})");
	write_text(temp.path() / "alpha" / "main.py", "# app code\n");
	write_text(
		temp.path() / "alpha" / "app.json",
		R"({"app_id":"com.sipeed.alpha","name":"Alpha","version":"1.0.0"})");
	write_text(temp.path() / "_private" / "main.py", "# app code\n");
	write_text(temp.path() / "_private" / "app.json",
		   R"({"app_id":"com.sipeed._private","name":"Private"})");
	write_text(temp.path() / "missing_info" / "main.py", "# app code\n");
	write_text(
		temp.path() / "missing_main" / "app.json",
		R"({"app_id":"com.sipeed.missing_main","name":"Missing Main"})");
	write_text(temp.path() / "invalid_info" / "main.py", "# app code\n");
	write_text(temp.path() / "invalid_info" / "app.json", "not json\n");
	write_text(temp.path() / "appbase.py", "# shared SDK\n");

	const auto apps = lvgl_main::scan_apps(temp.path());
	REQUIRE(apps.size() == 2U);
	CHECK(apps[0].name == "Alpha");
	CHECK(apps[0].path == (temp.path() / "alpha" / "main.py").string());
	CHECK(apps[1].name == "Zeta");
	CHECK(apps[1].path == (temp.path() / "zeta" / "main.py").string());
}

TEST_CASE("app info parsing is strict and non-throwing", "[app-catalog]")
{
	TempDir temp;
	const auto valid_dir = temp.path() / "valid-app";
	std::filesystem::create_directory(valid_dir);
	const auto valid = valid_dir / "app.json";
	const auto missing_name = temp.path() / "missing_name.json";
	const auto wrong_name_type = temp.path() / "wrong_name_type.json";
	const auto malformed = temp.path() / "malformed.json";
	write_text(
		valid,
		R"({"app_id":"com.sipeed.valid_app","name":"Visible Name","resources":["icon.png"]})");
	write_text(missing_name, R"({"version":"1.0.0"})");
	write_text(wrong_name_type, R"({"name":42})");
	write_text(malformed, "{");

	CHECK(lvgl_main::parse_app_manifest(valid) == "Visible Name");
	CHECK(lvgl_main::parse_app_manifest(missing_name).empty());
	CHECK(lvgl_main::parse_app_manifest(wrong_name_type).empty());
	CHECK(lvgl_main::parse_app_manifest(malformed).empty());
	CHECK(lvgl_main::parse_app_manifest(temp.path() / "missing.json")
		      .empty());
	CHECK(lvgl_main::scan_apps(temp.path() / "missing").empty());
}

TEST_CASE("App environment is loaded from app.json", "[app-catalog]")
{
	TempDir temp;
	const auto manifest = temp.path() / "app.json";
	write_text(
		manifest,
		R"({"app_id":"com.sipeed.valid_app","name":"App","creator":"Team","create_time":"2026-07-30","version":"1","env":{"TOKEN":{"required":true,"secret":true,"value":"configured"},"HOST":{"default":"example.test"}}})");
	std::map<std::string, std::string> environment;
	std::string error;
	REQUIRE(lvgl_main::parse_app_environment(manifest, environment, error));
	CHECK(error.empty());
	CHECK(environment.at("TOKEN") == "configured");
	CHECK(environment.at("HOST") == "example.test");

	write_text(manifest,
		   R"({"env":{"TOKEN":{"required":true,"secret":true}}})");
	CHECK_FALSE(
		lvgl_main::parse_app_environment(manifest, environment, error));
	CHECK(error.find("TOKEN") != std::string::npos);
}

#if NANOKVM_GO_SIMULATOR_BUILD
TEST_CASE("simulator Apps catalog reads only the generated safe path",
	  "[simulator][app-catalog]")
{
	const std::filesystem::path apps_path { std::string(
		lvgl_main::simulator_build_config::kAppsPath) };
	const auto apps = lvgl_main::scan_apps(apps_path);

	REQUIRE_FALSE(apps.empty());
	for (const auto &app : apps) {
		const std::filesystem::path main_path(app.path);
		CHECK(main_path.filename() == "main.py");
		CHECK(main_path.parent_path().parent_path() == apps_path);
		CHECK(std::filesystem::is_regular_file(main_path.parent_path() /
						       "app.json"));
	}
}
#endif
