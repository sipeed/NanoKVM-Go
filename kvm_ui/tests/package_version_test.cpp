#include <filesystem>
#include <memory>
#include <optional>
#include <stdexcept>
#include <string>
#include <type_traits>

#include <catch2/catch_test_macros.hpp>
#include <catch2/matchers/catch_matchers.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#include "lvgl_main/package_version.hpp"

using namespace lvgl_main;

TEST_CASE("deb package version parser trims dpkg-query output",
	  "[package-version]")
{
	const auto version =
		parse_deb_package_version(" \t1:0.0.20-2+device1\r\n");

	REQUIRE(version.has_value());
	CHECK(*version == "1:0.0.20-2+device1");
}

TEST_CASE("deb package version parser rejects blank output",
	  "[package-version]")
{
	CHECK_FALSE(parse_deb_package_version("").has_value());
	CHECK_FALSE(parse_deb_package_version(" \t\r\n").has_value());
}

TEST_CASE("startup package version caches a successful load",
	  "[package-version]")
{
	int load_count = 0;
	StartupPackageVersionCache cache([&load_count]() {
		++load_count;
		return std::optional<std::string> { "0.0.20" };
	});

	CHECK(cache.load_once() == "0.0.20");
	CHECK(cache.load_once() == "0.0.20");
	CHECK(cache.value() == "0.0.20");
	CHECK(load_count == 1);
}

TEST_CASE("startup package version caches a failed load", "[package-version]")
{
	int load_count = 0;
	StartupPackageVersionCache cache([&load_count]() {
		++load_count;
		return std::optional<std::string> {};
	});

	CHECK(cache.load_once().empty());
	CHECK(cache.load_once().empty());
	CHECK(cache.value().empty());
	CHECK(load_count == 1);
}

TEST_CASE("startup package version cache has strict single ownership",
	  "[package-version]")
{
	STATIC_REQUIRE_FALSE(
		std::is_copy_constructible_v<StartupPackageVersionCache>);
	STATIC_REQUIRE_FALSE(
		std::is_copy_assignable_v<StartupPackageVersionCache>);
	STATIC_REQUIRE_FALSE(
		std::is_move_constructible_v<StartupPackageVersionCache>);
	STATIC_REQUIRE_FALSE(
		std::is_move_assignable_v<StartupPackageVersionCache>);
}

TEST_CASE("startup package version rethrows a cached loader exception",
	  "[package-version]")
{
	int load_count = 0;
	StartupPackageVersionCache cache(
		[&load_count]() -> std::optional<std::string> {
			++load_count;
			throw std::runtime_error("package query failed");
		});

	CHECK_THROWS_WITH(cache.load_once(), "package query failed");
	CHECK_THROWS_WITH(cache.load_once(), "package query failed");
	CHECK(load_count == 1);
}

TEST_CASE("startup package version releases loader captures after load",
	  "[package-version]")
{
	auto owner = std::make_shared<int>(42);
	std::weak_ptr<int> weak_owner = owner;
	StartupPackageVersionCache cache([owner]() {
		return std::optional<std::string> { "0.0.20" };
	});

	owner.reset();
	REQUIRE_FALSE(weak_owner.expired());
	CHECK(cache.load_once() == "0.0.20");
	CHECK(weak_owner.expired());
}

TEST_CASE("runtime startup reads the package version only once",
	  "[package-version]")
{
	int load_count = 0;
	NanoKvmRuntimeService runtime(
		default_app_config(), {},
		std::filesystem::path {
			"/tmp/kvm-ui-package-version-test.toml" },
		[&load_count]() {
			++load_count;
			return std::optional<std::string> { "0.0.20" };
		});

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.current_status().app_version == "0.0.20");
	CHECK_FALSE(runtime.start_settings_cli_actions());
	runtime.stop();

	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.current_status().app_version == "0.0.20");
	runtime.stop();
	CHECK(load_count == 1);
}
