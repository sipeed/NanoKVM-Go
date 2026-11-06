#include <atomic>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <optional>
#include <string>
#include <unistd.h>

#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/ncm_runtime_state.hpp"

using namespace lvgl_main;

namespace
{

class TempMarkerDir final
{
    public:
	TempMarkerDir()
	{
		static std::atomic<uint64_t> next_id { 0 };
		const auto stamp = std::chrono::steady_clock::now()
					   .time_since_epoch()
					   .count();
		const uint64_t id =
			next_id.fetch_add(1, std::memory_order_relaxed);
		_path = std::filesystem::temp_directory_path() /
			("nanokvm_go_ncm_state_" + std::to_string(stamp) +
			 "_" + std::to_string(id));
		std::filesystem::create_directories(_path);
		std::filesystem::permissions(
			_path, std::filesystem::perms::owner_all);
	}

	~TempMarkerDir()
	{
		std::error_code ec;
		std::filesystem::remove_all(_path, ec);
	}

	TempMarkerDir(const TempMarkerDir &) = delete;
	TempMarkerDir &operator=(const TempMarkerDir &) = delete;

	const std::filesystem::path &path() const { return _path; }

	NcmBootMarkerPaths markers() const
	{
		return NcmBootMarkerPaths {
			(_path / "usb.ncm").string(),
			(_path / "usb.rndis").string(),
		};
	}

	void write_ncm()
	{
		std::ofstream f(_path / "usb.ncm", std::ios::trunc);
		REQUIRE(f.good());
	}

	void write_rndis()
	{
		std::ofstream f(_path / "usb.rndis", std::ios::trunc);
		REQUIRE(f.good());
	}

    private:
	std::filesystem::path _path;
};

bool running_as_root()
{
	return ::geteuid() == 0;
}

} // namespace

TEST_CASE("production NCM boot marker paths are the device boot contract")
{
	const auto paths = production_ncm_boot_marker_paths();
	CHECK(paths.ncm_flag == kNcmBootFlagPath);
	CHECK(paths.rndis_flag == kRndisBootFlagPath);
	CHECK(paths.ncm_flag == "/boot/usb.ncm");
	CHECK(paths.rndis_flag == "/boot/usb.rndis");
}

TEST_CASE("NCM observation reports disabled when no marker exists")
{
	TempMarkerDir temp;
	const auto observation = observe_ncm_boot_markers(temp.markers());
	CHECK(observation == NcmObservation::Disabled);
	CHECK(ncm_observation_enabled(observation) ==
	      std::optional<bool> { false });
}

TEST_CASE("NCM observation reports enabled when NCM marker exists")
{
	TempMarkerDir temp;
	temp.write_ncm();
	const auto observation = observe_ncm_boot_markers(temp.markers());
	CHECK(observation == NcmObservation::Enabled);
	CHECK(ncm_observation_enabled(observation) ==
	      std::optional<bool> { true });
}

TEST_CASE("NCM observation reports enabled when legacy RNDIS marker exists")
{
	TempMarkerDir temp;
	temp.write_rndis();
	const auto observation = observe_ncm_boot_markers(temp.markers());
	CHECK(observation == NcmObservation::Enabled);
	CHECK(ncm_observation_enabled(observation) ==
	      std::optional<bool> { true });
}

TEST_CASE("NCM observation reports enabled when both markers exist")
{
	TempMarkerDir temp;
	temp.write_ncm();
	temp.write_rndis();
	const auto observation = observe_ncm_boot_markers(temp.markers());
	CHECK(observation == NcmObservation::Enabled);
	CHECK(ncm_observation_enabled(observation) ==
	      std::optional<bool> { true });
}

TEST_CASE("NCM observation is inconclusive when a marker is non-regular")
{
	TempMarkerDir temp;
	std::error_code ec;
	std::filesystem::create_directories(temp.path() / "usb.ncm", ec);
	REQUIRE_FALSE(ec);
	const auto observation = observe_ncm_boot_markers(temp.markers());
	CHECK(observation == NcmObservation::Inconclusive);
	CHECK_FALSE(ncm_observation_enabled(observation).has_value());
}

TEST_CASE("NCM observation is inconclusive when a marker is unreadable")
{
	if (running_as_root()) {
		return;
	}
	TempMarkerDir temp;
	temp.write_ncm();
	const auto denied = temp.path() / "denied";
	std::filesystem::create_directories(denied);
	std::filesystem::permissions(denied, std::filesystem::perms::none);
	const NcmBootMarkerPaths paths {
		(denied / "usb.rndis").string(),
		(temp.path() / "usb.ncm").string(),
	};
	const auto observation = observe_ncm_boot_markers(paths);
	CHECK(observation == NcmObservation::Inconclusive);
	CHECK_FALSE(ncm_observation_enabled(observation).has_value());
	std::filesystem::permissions(denied,
				     std::filesystem::perms::owner_all);
}
