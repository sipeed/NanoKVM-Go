#pragma once

#include <cstdint>
#include <optional>
#include <string>
#include <string_view>

namespace lvgl_main
{

// Persistent boot marker paths. On the target these files under /boot are the
// sole source of NCM startup truth: usbdev.sh builds the gadget from them at
// boot, the Web API mutates them, and NanoKvmRuntimeService observes and
// applies them. kvm_ui no longer keeps a duplicate boolean in kvm_ui.toml.
struct NcmBootMarkerPaths {
	std::string ncm_flag;
	std::string rndis_flag;
};

// Production boot marker paths. Tests inject temporary paths instead.
inline constexpr std::string_view kNcmBootFlagPath = "/boot/usb.ncm";
inline constexpr std::string_view kRndisBootFlagPath = "/boot/usb.rndis";

// Build the production boot marker path pair.
NcmBootMarkerPaths production_ncm_boot_marker_paths() noexcept;

// Tri-state result of observing the NCM boot markers. A marker read failure is
// inconclusive so a caller can retain the last known state instead of
// projecting a stale or invented value.
enum class NcmObservation : uint8_t {
	Inconclusive,
	Disabled,
	Enabled,
};

// Observe the NCM and legacy RNDIS boot markers. Either marker present means
// NCM is enabled; neither present means disabled; a read failure on either
// marker is inconclusive. The observation is pure filesystem read and cannot
// reach the USB mutator, which keeps the startup no-mutation invariant
// directly testable with temporary paths.
NcmObservation
observe_ncm_boot_markers(const NcmBootMarkerPaths &paths) noexcept;

// Reduce an observation to a boolean NCM-enabled value for projection.
// Inconclusive yields std::nullopt so callers retain the last confirmed state.
std::optional<bool> ncm_observation_enabled(NcmObservation obs) noexcept;

} // namespace lvgl_main
