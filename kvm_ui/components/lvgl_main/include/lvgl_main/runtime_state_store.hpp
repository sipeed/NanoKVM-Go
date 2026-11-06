#pragma once

#include <cstdint>
#include <mutex>
#include <optional>
#include <string>
#include <variant>

#include "lvgl_main/runtime_service.hpp"

namespace lvgl_main
{

enum class RefreshObservationDomain : uint16_t {
	None = 0,
	NetworkAddresses = 1U << 0U,
	Wifi = 1U << 1U,
	HardwareVersion = 1U << 2U,
	Ncm = 1U << 3U,
	Backlight = 1U << 4U,
	ServerAvailability = 1U << 5U,
	Video = 1U << 6U,
	Hid = 1U << 7U,
	RuntimeFlags = 1U << 8U,
	AppVersion = 1U << 9U,
	SystemInfoStatic = 1U << 10U,
	SystemInfoActive = 1U << 11U,
	SystemInfoStorage = 1U << 12U,
	SystemInfoSignal = 1U << 13U,
	All = (1U << 14U) - 1U,
};

constexpr RefreshObservationDomain operator|(RefreshObservationDomain lhs,
							  RefreshObservationDomain rhs) noexcept
{
	return static_cast<RefreshObservationDomain>(
		static_cast<uint16_t>(lhs) | static_cast<uint16_t>(rhs));
}

constexpr RefreshObservationDomain &operator|=(RefreshObservationDomain &lhs,
								RefreshObservationDomain rhs) noexcept
{
	lhs = lhs | rhs;
	return lhs;
}

constexpr bool has_refresh_observation(RefreshObservationDomain domains,
						       RefreshObservationDomain domain) noexcept
{
	return (static_cast<uint16_t>(domains) & static_cast<uint16_t>(domain)) !=
	       0;
}

struct RefreshObservationPatch {
	RuntimeStatus status;
	uint64_t wifi_operation_generation { 0 };

	RefreshObservationDomain observed_domains { RefreshObservationDomain::All };
};

struct WifiLifecyclePatch {
	enum class Phase : uint8_t {
		Observation,
		Pending,
		Terminal,
	};

	RuntimeStatus status;
	uint64_t operation_generation { 0 };
	Phase phase { Phase::Observation };
	bool preserve_ap_stop_completion_when_stale { false };
};

struct DisplayIngressPatch {
	std::optional<std::string> resolution;
	std::optional<std::string> fps;
	std::optional<std::string> fps_unit;
	std::optional<std::string> keyboard_special;
	std::optional<std::string> keyboard_normal;
};

struct HidIngressPatch {
	std::optional<bool> mouse_active;
	std::optional<std::string> keyboard_special;
	std::optional<std::string> keyboard_normal;
};

struct SystemdStatusPatch {
	RuntimeStatus status;
};

struct LocalServicesPatch {
	RuntimeStatus status;
};

struct TailscalePatch {
	RuntimeStatus status;
};

struct TailscaleSnapshotPatch {
	RuntimeStatus status;
	bool snapshot_valid { false };
};

struct TailscaleActionResultPatch {
	RuntimeStatus status;
};

struct DisplayMetricsPatch {
	RuntimeStatus status;
};

struct RuntimeFlagsPatch {
	RuntimeStatus status;
};

struct FactoryResetPatch {
	uint64_t failure_generation { 0 };
};

struct RegionSwitchPatch {
	uint64_t failure_generation { 0 };
};

using RuntimeStatusPatch =
	std::variant<RefreshObservationPatch, WifiLifecyclePatch,
		     DisplayIngressPatch, HidIngressPatch, SystemdStatusPatch,
		     LocalServicesPatch, DisplayMetricsPatch, RuntimeFlagsPatch,
		     FactoryResetPatch, RegionSwitchPatch, TailscalePatch,
		     TailscaleSnapshotPatch, TailscaleActionResultPatch>;

class RuntimeStateStore {
    public:
	RuntimeStateStore() = default;
	explicit RuntimeStateStore(RuntimeStatus initial_status);

	RuntimeStatus current_status() const;
	RuntimeStatusPublication current_publication() const;

	void replace_initial_status(RuntimeStatus initial_status);
	std::optional<RuntimeStatusPublication>
	commit(const RuntimeStatusPatch &patch);

    private:
	bool apply(const RefreshObservationPatch &patch);
	bool apply(const WifiLifecyclePatch &patch);
	bool apply(const DisplayIngressPatch &patch);
	bool apply(const HidIngressPatch &patch);
	bool apply(const SystemdStatusPatch &patch);
	bool apply(const LocalServicesPatch &patch);
	bool apply(const DisplayMetricsPatch &patch);
	bool apply(const RuntimeFlagsPatch &patch);
	bool apply(const TailscalePatch &patch);
	bool apply(const TailscaleSnapshotPatch &patch);
	bool apply(const TailscaleActionResultPatch &patch);
	bool apply(const FactoryResetPatch &patch);
	bool apply(const RegionSwitchPatch &patch);

	void copy_wifi_status_fields(const RuntimeStatus &from,
						 bool include_wifi_ip = true);

	mutable std::mutex _mutex;
	RuntimeStatus _status;
	uint64_t _revision { 0 };
	uint64_t _latest_wifi_operation_generation { 0 };
	uint64_t _pending_wifi_operation_generation { 0 };
};

}
