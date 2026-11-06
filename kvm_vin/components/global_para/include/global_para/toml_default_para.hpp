#ifndef __KVM_UI_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__
#define __KVM_UI_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__

#include <string_view>

namespace global_para::toml_default {

constexpr std::string_view CONFIG_FILE_PATH = "/etc/kvm/kvm_vin.toml";

/* Global logger */
constexpr std::string_view GLOBAL_LOG_LEVEL_PATH = "log.level";
constexpr std::string_view GLOBAL_LOG_LEVEL_DEF = "error";

constexpr std::string_view GLOBAL_LOG_TYPE_PATH = "log.type";
constexpr std::string_view GLOBAL_LOG_TYPE_DEF = "file";

constexpr std::string_view GLOBAL_LOG_FILE_PATH = "log.file";
constexpr std::string_view GLOBAL_LOG_FILE_DEF = "/var/log/kvmcomm/vin.log";

/* Global VIN */
constexpr std::string_view GLOBAL_UI_LOG_LEVEL_PATH = "vin.level";
constexpr std::string_view GLOBAL_UI_LOG_LEVEL_DEF = "error";

constexpr std::string_view H264_IDLE_TIMEOUT_MS_PATH = "device.idle_timeout_ms";
constexpr int              H264_IDLE_TIMEOUT_MS_DEF  = 3000;

constexpr std::string_view H264_IDLE_POLL_MS_PATH = "device.idle_poll_ms";
constexpr int              H264_IDLE_POLL_MS_DEF  = 500;

constexpr std::string_view H264_IDLE_POWER_OFF_LT7911_PATH = "device.idle_power_off_lt7911";
constexpr bool             H264_IDLE_POWER_OFF_LT7911_DEF  = false;

constexpr std::string_view DEVICE_ENABLE_PATH = "device.enable";
constexpr bool             DEVICE_ENABLE_DEF  = true;

constexpr std::string_view DEVICE_MODE_PATH = "device.mode";
constexpr std::string_view DEVICE_MODE_DEF  = "h264";

constexpr std::string_view ENCODER_RATE_CONTROL_PATH = "encoder.rate_control";
constexpr std::string_view ENCODER_RATE_CONTROL_DEF  = "vbr";

constexpr std::string_view ENCODER_BITRATE_PATH = "encoder.bitrate";
constexpr int              ENCODER_BITRATE_DEF  = 8000;
constexpr int              ENCODER_BITRATE_MIN  = 1000;
constexpr int              ENCODER_BITRATE_MAX  = 20000;

constexpr std::string_view ENCODER_GOP_PATH = "encoder.gop";
constexpr int              ENCODER_GOP_DEF  = 50;
constexpr int              ENCODER_GOP_MIN  = 1;
constexpr int              ENCODER_GOP_MAX  = 300;

constexpr std::string_view ENCODER_FPS_PATH = "encoder.fps";
constexpr int              ENCODER_FPS_DEF  = 0;
constexpr int              ENCODER_FPS_MIN  = 0;
constexpr int              ENCODER_FPS_MAX  = 120;

constexpr std::string_view SNAPSHOT_ENABLE_PATH = "snapshot.enable";
constexpr bool             SNAPSHOT_ENABLE_DEF  = false;

constexpr std::string_view SNAPSHOT_PERIODIC_ENABLE_PATH = "snapshot.periodic_enable";
constexpr bool             SNAPSHOT_PERIODIC_ENABLE_DEF  = false;

constexpr std::string_view TIMELAPSE_ENABLE_PATH = "timelapse.enable";
constexpr bool             TIMELAPSE_ENABLE_DEF  = false;

constexpr std::string_view TIMELAPSE_QP_PATH = "timelapse.qp";
constexpr int              TIMELAPSE_QP_DEF  = 42;
constexpr int              TIMELAPSE_QP_MIN  = 36;
constexpr int              TIMELAPSE_QP_MAX  = 51;
}  // namespace global_para::toml_default

#endif  // __KVM_VIN_GLOBAL_PARAMETERS_TOML_DEFAULT_HPP__
