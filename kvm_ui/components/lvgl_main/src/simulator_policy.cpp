#include "lvgl_main/simulator_policy.hpp"

#include <array>
#include <system_error>

#if NANOKVM_GO_SIMULATOR_BUILD
#include "lvgl_main/simulator_build_config.hpp"
#endif

namespace lvgl_main::simulator_policy
{

namespace
{

constexpr std::array<std::string_view, 8> kProtectedRoots {
	"/boot",    "/dev", "/sys",	"/run/kvm",
	"/etc/kvm", "/opt", "/kvmcomm", "/tmp/kvm",
};
#if !NANOKVM_GO_SIMULATOR_BUILD
constexpr std::string_view kProductionLauncherAppsPath = "/kvmcomm/apps";
#endif

bool is_same_or_descendant(const std::filesystem::path &path,
			   const std::filesystem::path &root)
{
	auto path_it = path.begin();
	auto root_it = root.begin();
	for (; root_it != root.end(); ++root_it, ++path_it) {
		if (path_it == path.end() || *path_it != *root_it) {
			return false;
		}
	}
	return true;
}

std::filesystem::path
make_absolute_normalized(const std::filesystem::path &path, std::error_code &ec)
{
	std::filesystem::path absolute_path =
		std::filesystem::absolute(path, ec);
	if (ec) {
		return {};
	}
	return absolute_path.lexically_normal();
}

} // namespace

bool is_protected_target_path(const std::filesystem::path &path)
{
	if (path.empty()) {
		return false;
	}

	std::error_code ec;
	const auto normalized = make_absolute_normalized(path, ec);
	if (ec || normalized == normalized.root_path()) {
		return true;
	}

	for (const std::string_view root_text : kProtectedRoots) {
		const std::filesystem::path root(root_text);
		if (is_same_or_descendant(normalized, root)) {
			return true;
		}
	}
	return false;
}

bool resolve_config_path(const std::filesystem::path &requested_path,
			 bool explicitly_requested,
			 std::filesystem::path &resolved_path,
			 std::string &error)
{
	error.clear();
#if NANOKVM_GO_SIMULATOR_BUILD
	if (!explicitly_requested) {
		resolved_path = std::filesystem::path(
			simulator_build_config::kDefaultConfigPath);
		return true;
	}
#endif

	if (requested_path.empty()) {
		error = "simulator config path must not be empty";
		return false;
	}

	std::error_code ec;
	resolved_path = make_absolute_normalized(requested_path, ec);
	if (ec) {
		error = "failed to resolve simulator config path: " +
			ec.message();
		return false;
	}
	if (is_protected_target_path(resolved_path)) {
		error = "simulator config path is protected: " +
			resolved_path.string();
		return false;
	}
	return true;
}

std::filesystem::path launcher_apps_path()
{
#if NANOKVM_GO_SIMULATOR_BUILD
	return std::filesystem::path(simulator_build_config::kAppsPath);
#else
	return std::filesystem::path(kProductionLauncherAppsPath);
#endif
}

AppConfig make_effective_config(const AppConfig &config)
{
	auto effective = config;
#if NANOKVM_GO_SIMULATOR_BUILD
	effective.display.framebuffer_device = "simulator-display";
	effective.touch.device = "simulator-pointer";
	effective.http_api.enabled = false;
	effective.http_api.bind_host.clear();
	effective.http_api.port = 0;
	effective.http_api.token.clear();
	effective.screenshot.enabled = false;
	effective.runtime.touch_calibrate = false;
	effective.runtime.touch_paint_test = false;
	effective.ui.enable_auto_rotation = false;
	effective.nanokvm.enable_runtime_backend = true;
	effective.nanokvm.server_base_url.clear();
	effective.nanokvm.server_verify_tls = false;
	effective.nanokvm.wifi_script.clear();
	effective.nanokvm.usb_gadget_restart_script.clear();
	effective.nanokvm.backlight_root.clear();
	effective.nanokvm.enable_high_risk_actions = false;
	effective.launcher.enabled = true;
	effective.launcher.python_bin.clear();
#endif
	return effective;
}

} // namespace lvgl_main::simulator_policy
