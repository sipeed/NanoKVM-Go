#include "lvgl_main/ncm_runtime_state.hpp"

#include <filesystem>
#include <system_error>

#include <spdlog/spdlog.h>

namespace lvgl_main
{

namespace {

// Read a single boot marker. Returns std::nullopt only on a real filesystem
// error or a non-regular entry (corruption); a genuinely absent marker is a
// clean false observation so a caller can distinguish disabled from unreadable.
std::optional<bool> marker_present(std::string_view path) noexcept
{
	std::error_code ec;
	const auto type =
		std::filesystem::status(std::filesystem::path(std::string(path)),
				    ec)
			.type();
	using ft = std::filesystem::file_type;
	if (type == ft::not_found) {
		return false;
	}
	if (ec) {
		return std::nullopt;
	}
	return type == ft::regular ? std::optional<bool> { true } :
				    std::nullopt;
}

} // namespace

NcmBootMarkerPaths production_ncm_boot_marker_paths() noexcept
{
	return NcmBootMarkerPaths { std::string(kNcmBootFlagPath),
				    std::string(kRndisBootFlagPath) };
}

NcmObservation
observe_ncm_boot_markers(const NcmBootMarkerPaths &paths) noexcept
{
	const auto ncm = marker_present(paths.ncm_flag);
	const auto rndis = marker_present(paths.rndis_flag);
	if (!ncm || !rndis) {
		return NcmObservation::Inconclusive;
	}
	return (*ncm || *rndis) ? NcmObservation::Enabled :
				  NcmObservation::Disabled;
}

std::optional<bool> ncm_observation_enabled(NcmObservation obs) noexcept
{
	switch (obs) {
	case NcmObservation::Enabled:
		return true;
	case NcmObservation::Disabled:
		return false;
	case NcmObservation::Inconclusive:
		break;
	}
	return std::nullopt;
}

} // namespace lvgl_main
