#include <atomic>
#include <chrono>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <optional>
#include <string>
#include <string_view>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/app_config.hpp"
#include "lvgl_main/device_region.hpp"
#include "lvgl_main/nanokvm_runtime.hpp"
#include "lvgl_main/ncm_runtime_state.hpp"

using namespace lvgl_main;

namespace
{

class TempDir final {
    public:
	TempDir()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		const uint64_t id =
			next_id.fetch_add(1, std::memory_order_relaxed);
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_device_region_" + std::to_string(stamp) +
			 "_" + std::to_string(id));
		std::filesystem::create_directories(_path);
		std::filesystem::permissions(
			_path, std::filesystem::perms::owner_all);
	}

	~TempDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempDir(const TempDir &) = delete;
	TempDir &operator=(const TempDir &) = delete;

	const std::filesystem::path &path() const { return _path; }

	std::filesystem::path marker_path() const { return _path / "region"; }

	void write_marker(std::string_view contents)
	{
		std::ofstream marker(marker_path(), std::ios::trunc);
		REQUIRE(marker.good());
		marker << contents;
		REQUIRE(marker.good());
	}

    private:
	std::filesystem::path _path;
};

const std::filesystem::path kRegionTestConfigPath {
	"/tmp/kvm-ui-device-region-test.toml"
};

NanoKvmRuntimeService
make_region_runtime(const std::filesystem::path &region_path)
{
	return NanoKvmRuntimeService(
		default_app_config(), {}, kRegionTestConfigPath,
		[]() {
			return std::optional<std::string> { "test" };
		},
		{}, production_ncm_boot_marker_paths(), region_path,
		NanoKvmRuntimeService::RegionSwitchRunner {},
		default_tailscale_socket_path());
}

}

TEST_CASE("device region parser accepts only exact cn and global tokens",
	  "[device-region]")
{
	CHECK(parse_device_region("cn") == DeviceRegion::Cn);
	CHECK(parse_device_region(" cn \r\n") == DeviceRegion::Cn);
	CHECK(parse_device_region("\ncn\n") == DeviceRegion::Cn);
	CHECK(parse_device_region("global") == DeviceRegion::Global);
	CHECK(parse_device_region(" global\r\n") == DeviceRegion::Global);
	CHECK(device_region_name(DeviceRegion::Cn) == "cn");
	CHECK(device_region_name(DeviceRegion::Global) == "global");
}

TEST_CASE("device region parser falls back to global for malformed content",
	  "[device-region]")
{
	CHECK(parse_device_region("") == DeviceRegion::Global);
	CHECK(parse_device_region(" \t\r\n\f\v") == DeviceRegion::Global);
	CHECK(parse_device_region("CN") == DeviceRegion::Global);
	CHECK(parse_device_region("Cn") == DeviceRegion::Global);
	CHECK(parse_device_region("GLOBAL") == DeviceRegion::Global);
	CHECK(parse_device_region("cn\njunk") == DeviceRegion::Global);
	CHECK(parse_device_region("global extra") == DeviceRegion::Global);
	CHECK(parse_device_region("\xEF\xBB\xBF"
				  "cn") == DeviceRegion::Global);
	CHECK(parse_device_region(std::string(4096, 'x')) ==
	      DeviceRegion::Global);
}

TEST_CASE("device region reader accepts trimmed cn and global markers",
	  "[device-region]")
{
	TempDir temp;
	temp.write_marker("cn\n");
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Cn);

	temp.write_marker(" global \r\n");
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);
}

TEST_CASE("device region reader falls back to global for an empty marker",
	  "[device-region]")
{
	TempDir temp;
	temp.write_marker("");
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);
}

TEST_CASE("device region reader falls back to global for invalid markers",
	  "[device-region]")
{
	TempDir temp;
	temp.write_marker("cn\njunk");
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);

	temp.write_marker("GLOBAL");
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);
}

TEST_CASE("device region reader falls back to global for a missing path",
	  "[device-region]")
{
	TempDir temp;
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);
}

TEST_CASE("device region reader falls back to global for a directory path",
	  "[device-region]")
{
	TempDir temp;
	std::error_code ec;
	std::filesystem::create_directories(temp.marker_path(), ec);
	REQUIRE_FALSE(ec);
	CHECK(read_device_region(temp.marker_path()) == DeviceRegion::Global);
}

TEST_CASE("runtime startup reads the injected region marker",
	  "[device-region]")
{
	TempDir temp;

	temp.write_marker("cn");
	{
		auto runtime = make_region_runtime(temp.marker_path());
		REQUIRE(runtime.start_settings_cli_actions());
		CHECK(runtime.current_status().region == DeviceRegion::Cn);
		runtime.stop();
	}

	temp.write_marker("global");
	{
		auto runtime = make_region_runtime(temp.marker_path());
		REQUIRE(runtime.start_settings_cli_actions());
		CHECK(runtime.current_status().region ==
		      DeviceRegion::Global);
		runtime.stop();
	}
}

TEST_CASE("runtime startup falls back to global for missing or invalid markers",
	  "[device-region]")
{
	TempDir temp;

	{
		auto runtime = make_region_runtime(temp.marker_path());
		REQUIRE(runtime.start_settings_cli_actions());
		CHECK(runtime.current_status().region ==
		      DeviceRegion::Global);
		runtime.stop();
	}

	temp.write_marker("CN");
	{
		auto runtime = make_region_runtime(temp.marker_path());
		REQUIRE(runtime.start_settings_cli_actions());
		CHECK(runtime.current_status().region ==
		      DeviceRegion::Global);
		runtime.stop();
	}
}

TEST_CASE("runtime region marker is read once per service instance",
	  "[device-region]")
{
	TempDir temp;
	temp.write_marker("cn");

	auto runtime = make_region_runtime(temp.marker_path());
	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.current_status().region == DeviceRegion::Cn);
	runtime.stop();

	temp.write_marker("global");
	REQUIRE(runtime.start_settings_cli_actions());
	CHECK(runtime.current_status().region == DeviceRegion::Cn);
	runtime.stop();
}
