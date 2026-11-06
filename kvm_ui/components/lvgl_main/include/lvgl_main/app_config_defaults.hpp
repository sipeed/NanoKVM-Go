#pragma once

#include <string_view>

namespace lvgl_main::app_config_defaults
{

inline constexpr std::string_view kConfigPath = "/etc/kvm/kvm_ui.toml";

inline constexpr std::string_view kFramebufferDevice = "/dev/fb0";

inline constexpr std::string_view kTouchDevice = "/dev/input/event0";
inline constexpr bool kTouchDisabled = false;

inline constexpr bool kHttpApiEnabled = true;
inline constexpr std::string_view kHttpApiBindHost = "127.0.0.1";
inline constexpr int kHttpApiPort = 65501;
inline constexpr std::string_view kHttpApiToken = "";
inline constexpr int kHttpApiPortMin = 0;
inline constexpr int kHttpApiPortMax = 65535;

inline constexpr bool kRuntimeDebugLogging = false;
inline constexpr bool kRuntimeTouchClickMarker = false;
inline constexpr bool kRuntimeTouchCalibrate = false;
inline constexpr bool kRuntimeTouchPaintTest = false;

inline constexpr bool kScreenshotEnabled = false;
inline constexpr std::string_view kScreenshotSignal = "SIGUSR1";
inline constexpr std::string_view kScreenshotOutputDirectory =
	"/root/snapshot";

inline constexpr bool kEnableAutoRotation = false;
inline constexpr bool kManualDisplayRotation = false;
inline constexpr bool kExperimentalSettingsSubpages = true;
inline constexpr bool kEnableSettingsOptimisticActions = true;
inline constexpr std::string_view kUiPageTransitionMode = "slide";
inline constexpr int kWifiBootReconnectTimeoutMs = 15000;
inline constexpr int kWifiBootReconnectTimeoutMsMin = 1000;
inline constexpr int kWifiBootReconnectTimeoutMsMax = 60000;
inline constexpr int kSettingsStateBootTimeoutMs = 8000;
inline constexpr int kSettingsStateBootTimeoutMsMin = 1000;
inline constexpr int kSettingsStateBootTimeoutMsMax = 60000;

inline constexpr bool kWifiLoadingProgress = true;

inline constexpr std::string_view kFontsCjkRegion = "sc";
inline constexpr int kFontsSsidImageCacheKb = 4096;
inline constexpr int kFontsSsidImageCacheKbMin = 128;
inline constexpr int kFontsSsidImageCacheKbMax = 4096;

inline constexpr std::string_view kExperimentalSsidScroll = "auto";

inline constexpr bool kNanokvmRuntimeBackendEnabled = true;
inline constexpr bool kNanokvmPreserveLastKnownGoodState = true;
inline constexpr std::string_view kNanokvmServerBaseUrl = "https://127.0.0.1";
inline constexpr bool kNanokvmServerVerifyTls = false;
inline constexpr std::string_view kNanokvmWifiScript = "/opt/scripts/wifi.sh";
inline constexpr std::string_view kNanokvmUsbGadgetRestartScript =
	"/opt/scripts/usbdev.sh";
inline constexpr std::string_view kNanokvmLegacyUsbGadgetRestartScript =
	"/kvmapp/scripts/usbdev.sh";
inline constexpr std::string_view kNanokvmBacklightRoot =
	"/sys/class/backlight/backlight";
inline constexpr bool kNanokvmHighRiskActionsEnabled = true;
inline constexpr bool kNanokvmVideoEnabled = true;
inline constexpr bool kNanokvmHidEnabled = true;
inline constexpr int kNanokvmBacklightPercent = 70;
inline constexpr int kNanokvmBacklightPercentMin = 20;
inline constexpr int kNanokvmBacklightPercentMax = 100;
inline constexpr bool kNanokvmBacklightInvert = false;
inline constexpr bool kNanokvmAutoSleepEnabled = false;

inline constexpr bool kLauncherEnabled = true;
inline constexpr std::string_view kLauncherPythonBin = "python3";
inline constexpr int kLauncherExitZonePx = 40;
inline constexpr int kLauncherExitZonePxMin = 0;
inline constexpr int kLauncherExitZonePxMax = 284;
inline constexpr int kLauncherExitHoldMs = 1200;
inline constexpr int kLauncherExitHoldMsMin = 200;
inline constexpr int kLauncherExitHoldMsMax = 5000;
inline constexpr int kLauncherChildTermTimeoutMs = 1500;
inline constexpr int kLauncherChildTermTimeoutMsMin = 100;
inline constexpr int kLauncherChildTermTimeoutMsMax = 10000;

}
