#pragma once

#include <functional>
#include <string_view>

#include <ax_venc_rc.h>
#include <global_para/toml_default_para.hpp>

enum class EncoderCodec {
    H264,
    H265,
};

enum class EncoderRateControl {
    CBR,
    VBR,
};

struct EncoderSettings {
    EncoderRateControl rate_control {EncoderRateControl::VBR};
    int bitrate {global_para::toml_default::ENCODER_BITRATE_DEF};
    int gop {global_para::toml_default::ENCODER_GOP_DEF};
    int fps {global_para::toml_default::ENCODER_FPS_DEF};

    bool operator==(const EncoderSettings&) const = default;
};

struct EncoderSettingsOperations {
    std::function<bool(EncoderRateControl)> set_rate_control;
    std::function<bool(unsigned int)> set_bitrate;
    std::function<bool(unsigned int)> set_gop;
    std::function<bool(int)> set_fps;
};

using EncoderSettingsWarning = std::function<void(std::string_view)>;

EncoderSettings load_encoder_settings(
    const EncoderSettingsWarning& warning = EncoderSettingsWarning {});
std::string_view encoder_rate_control_name(EncoderRateControl rate_control);
AX_VENC_RC_MODE_E encoder_rate_control_mode(EncoderCodec codec,
                                            EncoderRateControl rate_control);
bool apply_encoder_settings(const EncoderSettings& settings,
                            const EncoderSettingsOperations& operations);
