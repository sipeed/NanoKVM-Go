#include "lvgl_main/device_region.hpp"

#include <exception>
#include <fstream>
#include <iterator>
#include <optional>
#include <string>

#include <spdlog/spdlog.h>

namespace lvgl_main
{

namespace
{

constexpr std::string_view kRegionWhitespace = " \t\n\r\f\v";

std::optional<DeviceRegion>
parse_device_region_token(std::string_view contents) noexcept
{
	const auto first = contents.find_first_not_of(kRegionWhitespace);
	if (first == std::string_view::npos) {
		return std::nullopt;
	}
	const auto last = contents.find_last_not_of(kRegionWhitespace);
	const auto token = contents.substr(first, last - first + 1U);
	if (token == "cn") {
		return DeviceRegion::Cn;
	}
	if (token == "global") {
		return DeviceRegion::Global;
	}
	return std::nullopt;
}

} // namespace

DeviceRegion parse_device_region(std::string_view contents) noexcept
{
	return parse_device_region_token(contents)
		.value_or(DeviceRegion::Global);
}

std::string_view device_region_name(DeviceRegion region) noexcept
{
	switch (region) {
	case DeviceRegion::Cn:
		return "cn";
	case DeviceRegion::Global:
		break;
	}
	return "global";
}

DeviceRegion read_device_region(const std::filesystem::path &path)
{
	DeviceRegion region = DeviceRegion::Global;
	try {
		std::ifstream input(path, std::ios::binary);
		if (!input) {
			spdlog::warn(
				"NanoKVM runtime device region marker unreadable path={}; using global",
				path.string());
		} else {
			const std::string contents(
				(std::istreambuf_iterator<char>(input)),
				std::istreambuf_iterator<char>());
			if (input.bad()) {
				spdlog::warn(
					"NanoKVM runtime device region marker read failed path={}; using global",
					path.string());
			} else {
				const auto parsed =
					parse_device_region_token(contents);
				if (parsed) {
					region = *parsed;
				} else {
					spdlog::warn(
						"NanoKVM runtime device region marker has unrecognized content path={}; using global",
						path.string());
				}
			}
		}
	} catch (const std::exception &e) {
		spdlog::warn(
			"NanoKVM runtime device region marker read failed path={} error={}; using global",
			path.string(), e.what());
		region = DeviceRegion::Global;
	} catch (...) {
		spdlog::warn(
			"NanoKVM runtime device region marker read failed path={}; using global",
			path.string());
		region = DeviceRegion::Global;
	}
	spdlog::info("[runtime] device region: {}", device_region_name(region));
	return region;
}

} // namespace lvgl_main
