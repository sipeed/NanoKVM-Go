#include <string>

#include "encoder_settings.hpp"

#include <tools/toml_file_parser.hpp>

namespace {

namespace GT = global_para::toml_default;
using TP = tools::TomlFileParser;

void emit_warning(const EncoderSettingsWarning& warning, std::string message)
{
    if (warning) {
        warning(message);
    }
}

EncoderRateControl default_rate_control()
{
    return GT::ENCODER_RATE_CONTROL_DEF == "cbr" ? EncoderRateControl::CBR
                                                 : EncoderRateControl::VBR;
}

EncoderRateControl load_rate_control(const EncoderSettingsWarning& warning)
{
    const auto value =
        TP::get<std::string>(std::string(GT::ENCODER_RATE_CONTROL_PATH));
    if (!value.has_value()) {
        emit_warning(warning,
                     std::string(GT::ENCODER_RATE_CONTROL_PATH)
                         + " is missing or has the wrong type; using default "
                         + std::string(GT::ENCODER_RATE_CONTROL_DEF));
        return default_rate_control();
    }
    if (*value == "cbr") {
        return EncoderRateControl::CBR;
    }
    if (*value == "vbr") {
        return EncoderRateControl::VBR;
    }
    emit_warning(warning,
                 std::string(GT::ENCODER_RATE_CONTROL_PATH)
                     + " has invalid value '" + *value + "'; using default "
                     + std::string(GT::ENCODER_RATE_CONTROL_DEF));
    return default_rate_control();
}

int load_bounded_int(std::string_view path,
                     int default_value,
                     int min_value,
                     int max_value,
                     const EncoderSettingsWarning& warning)
{
    const auto value = TP::get<int>(std::string(path));
    if (!value.has_value()) {
        emit_warning(warning,
                     std::string(path)
                         + " is missing or has the wrong type; using default "
                         + std::to_string(default_value));
        return default_value;
    }
    if (*value < min_value || *value > max_value) {
        emit_warning(warning,
                     std::string(path) + " value " + std::to_string(*value)
                         + " is outside [" + std::to_string(min_value) + ", "
                         + std::to_string(max_value) + "]; using default "
                         + std::to_string(default_value));
        return default_value;
    }
    return *value;
}

}  // namespace

EncoderSettings load_encoder_settings(const EncoderSettingsWarning& warning)
{
    return EncoderSettings {
        .rate_control = load_rate_control(warning),
        .bitrate = load_bounded_int(GT::ENCODER_BITRATE_PATH,
                                    GT::ENCODER_BITRATE_DEF,
                                    GT::ENCODER_BITRATE_MIN,
                                    GT::ENCODER_BITRATE_MAX,
                                    warning),
        .gop = load_bounded_int(GT::ENCODER_GOP_PATH,
                                GT::ENCODER_GOP_DEF,
                                GT::ENCODER_GOP_MIN,
                                GT::ENCODER_GOP_MAX,
                                warning),
        .fps = load_bounded_int(GT::ENCODER_FPS_PATH,
                                GT::ENCODER_FPS_DEF,
                                GT::ENCODER_FPS_MIN,
                                GT::ENCODER_FPS_MAX,
                                warning),
    };
}

std::string_view encoder_rate_control_name(EncoderRateControl rate_control)
{
    switch (rate_control) {
    case EncoderRateControl::CBR:
        return "cbr";
    case EncoderRateControl::VBR:
        return "vbr";
    }
    return "unknown";
}

AX_VENC_RC_MODE_E encoder_rate_control_mode(EncoderCodec codec,
                                            EncoderRateControl rate_control)
{
    if (codec == EncoderCodec::H265) {
        return rate_control == EncoderRateControl::CBR
            ? AX_VENC_RC_MODE_H265CBR
            : AX_VENC_RC_MODE_H265VBR;
    }
    return rate_control == EncoderRateControl::CBR ? AX_VENC_RC_MODE_H264CBR
                                                   : AX_VENC_RC_MODE_H264VBR;
}

bool apply_encoder_settings(const EncoderSettings& settings,
                            const EncoderSettingsOperations& operations)
{
    if (!operations.set_rate_control || !operations.set_bitrate
        || !operations.set_gop) {
        return false;
    }

    if (!operations.set_rate_control(settings.rate_control)) {
        return false;
    }
    if (!operations.set_bitrate(static_cast<unsigned int>(settings.bitrate))) {
        return false;
    }
    if (!operations.set_gop(static_cast<unsigned int>(settings.gop))) {
        return false;
    }
    if (settings.fps > 0) {
        if (!operations.set_fps || !operations.set_fps(settings.fps)) {
            return false;
        }
    }

    return true;
}
