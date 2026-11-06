#pragma once

#include <chrono>
#include <cstdint>
#include <filesystem>
#include <functional>
#include <memory>
#include <optional>
#include <string_view>

#include <nlohmann/json.hpp>

#include "lvgl_main/app_config.hpp"

namespace lvgl_main
{

inline constexpr std::string_view kNanoKvmUiSocketPath =
	"/run/kvm/nanokvm-ui.sock";

inline constexpr std::chrono::milliseconds kNanoKvmUiProbeInterval { 1000 };

enum class NanoKvmServerTransportLane : uint8_t {
	Status,
	Metrics,
};

class NanoKvmServerTransport final
{
    public:
	using Lane = NanoKvmServerTransportLane;
	using LegacyGet =
		std::function<std::optional<nlohmann::json>(std::string_view path)>;
	using LegacyPost = std::function<std::optional<nlohmann::json>(
		std::string_view path, const nlohmann::json &body)>;

	explicit NanoKvmServerTransport(
		const NanokvmRuntimeConfig &config, LegacyGet legacy_get,
		std::filesystem::path socket_path =
			std::filesystem::path(kNanoKvmUiSocketPath),
		std::chrono::milliseconds probe_interval =
			kNanoKvmUiProbeInterval,
		LegacyPost legacy_post = {});
	~NanoKvmServerTransport();

	NanoKvmServerTransport(const NanoKvmServerTransport &) = delete;
	NanoKvmServerTransport &operator=(const NanoKvmServerTransport &) = delete;

	std::optional<nlohmann::json> get_json(
		NanoKvmServerTransportLane lane, std::string_view path);
	std::optional<nlohmann::json> post_json(
		NanoKvmServerTransportLane lane, std::string_view path,
		const nlohmann::json &body);

	void start();
	void stop();

	bool is_uds_adopted() const;
	bool is_eligible() const;

    private:
	struct Impl;
	std::unique_ptr<Impl> _impl;
};

}
