#include <filesystem>
#include <fstream>
#include <iterator>
#include <string>
#include <string_view>
#include <vector>

#include "encoder_settings.hpp"

#include <catch2/catch_test_macros.hpp>
#include <tools/toml_file_parser.hpp>
#include <unistd.h>

namespace {

namespace GT = global_para::toml_default;

struct EncoderSettingsFixture {
    EncoderSettingsFixture()
        : path(std::filesystem::temp_directory_path()
               / ("kvm_vin_encoder_settings_" + std::to_string(::getpid())
                  + ".toml"))
    { write_config(""); }

    ~EncoderSettingsFixture()
    {
        std::error_code error;
        (void)std::filesystem::remove(path, error);
    }

    void write_config(std::string_view config) const
    {
        std::ofstream output(path, std::ios::trunc);
        REQUIRE(output);
        output << config;
        REQUIRE(output);
        output.close();
        tools::TomlFileParser::set_path(path.string());
    }

    std::string read_config() const
    {
        std::ifstream input(path, std::ios::binary);
        REQUIRE(input);
        return std::string(std::istreambuf_iterator<char>(input),
                           std::istreambuf_iterator<char>());
    }

    std::filesystem::path path;
};

struct RecordingEncoderOperations {
    std::vector<std::string> calls;
    std::string fail_at;

    EncoderSettingsOperations operations()
    {
        return EncoderSettingsOperations {
            .set_rate_control =
                [this](EncoderRateControl rate_control) {
                    calls.emplace_back(
                        "rate_control:"
                        + std::string(encoder_rate_control_name(rate_control)));
                    return fail_at != "rate_control";
                },
            .set_bitrate =
                [this](unsigned int bitrate) {
                    calls.emplace_back("bitrate:" + std::to_string(bitrate));
                    return fail_at != "bitrate";
                },
            .set_gop =
                [this](unsigned int gop) {
                    calls.emplace_back("gop:" + std::to_string(gop));
                    return fail_at != "gop";
                },
            .set_fps =
                [this](int fps) {
                    calls.emplace_back("fps:" + std::to_string(fps));
                    return fail_at != "fps";
                },
        };
    }
};

EncoderSettings load_with_warnings(std::vector<std::string>& warnings)
{
    return load_encoder_settings([&warnings](std::string_view warning) {
        warnings.emplace_back(warning);
    });
}

}  // namespace

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings use defaults when section is missing",
                 "[encoder][config]")
{
    write_config("[device]\nmode = \"h264\"\n");

    std::vector<std::string> warnings;
    const EncoderSettings settings = load_with_warnings(warnings);
    CHECK(settings == EncoderSettings {});
    CHECK(settings.rate_control == EncoderRateControl::VBR);
    CHECK(settings.bitrate == GT::ENCODER_BITRATE_DEF);
    CHECK(settings.gop == GT::ENCODER_GOP_DEF);
    CHECK(settings.fps == GT::ENCODER_FPS_DEF);
    REQUIRE(warnings.size() == 4);
    for (const auto& warning : warnings) {
        CHECK(warning.find("missing or has the wrong type")
              != std::string::npos);
    }
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings load valid values",
                 "[encoder][config]")
{
    write_config(R"(
[encoder]
rate_control = "cbr"
bitrate = 12345
gop = 180
fps = 60
)");

    const EncoderSettings settings = load_encoder_settings();
    CHECK(settings.rate_control == EncoderRateControl::CBR);
    CHECK(settings.bitrate == 12345);
    CHECK(settings.gop == 180);
    CHECK(settings.fps == 60);
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Loading encoder settings does not modify the config file",
                 "[encoder][config]")
{
    write_config(R"([device]
mode = "h264"

[encoder]
rate_control = "vbr"
bitrate = 9000
gop = 75
fps = 30

[custom]
value = "preserve exact bytes"
)");
    const std::string before = read_config();

    const EncoderSettings settings = load_encoder_settings();

    CHECK(settings.bitrate == 9000);
    CHECK(read_config() == before);
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings replace invalid values with defaults",
                 "[encoder][config]")
{
    write_config(R"(
[encoder]
rate_control = "abr"
bitrate = 999
gop = 301
fps = -1
)");

    std::vector<std::string> warnings;
    CHECK(load_with_warnings(warnings) == EncoderSettings {});
    REQUIRE(warnings.size() == 4);
    CHECK(warnings[0].find("has invalid value") != std::string::npos);
    CHECK(warnings[1].find("is outside") != std::string::npos);
    CHECK(warnings[2].find("is outside") != std::string::npos);
    CHECK(warnings[3].find("is outside") != std::string::npos);
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings replace wrong TOML types with defaults",
                 "[encoder][config]")
{
    write_config(R"(
[encoder]
rate_control = 1
bitrate = "8000"
gop = false
fps = "60"
)");

    std::vector<std::string> warnings;
    CHECK(load_with_warnings(warnings) == EncoderSettings {});
    REQUIRE(warnings.size() == 4);
    for (const auto& warning : warnings) {
        CHECK(warning.find("missing or has the wrong type")
              != std::string::npos);
    }
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings only replace invalid fields",
                 "[encoder][config]")
{
    write_config(R"(
[encoder]
rate_control = "cbr"
bitrate = 12000
gop = 0
fps = 30
)");

    const EncoderSettings settings = load_encoder_settings();
    CHECK(settings.rate_control == EncoderRateControl::CBR);
    CHECK(settings.bitrate == 12000);
    CHECK(settings.gop == GT::ENCODER_GOP_DEF);
    CHECK(settings.fps == 30);
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings accept boundary values",
                 "[encoder][config]")
{
    SECTION("minimum values")
    {
        write_config(R"(
[encoder]
rate_control = "vbr"
bitrate = 1000
gop = 1
fps = 0
)");

        const EncoderSettings settings = load_encoder_settings();
        CHECK(settings.bitrate == GT::ENCODER_BITRATE_MIN);
        CHECK(settings.gop == GT::ENCODER_GOP_MIN);
        CHECK(settings.fps == GT::ENCODER_FPS_MIN);
    }

    SECTION("maximum values")
    {
        write_config(R"(
[encoder]
rate_control = "vbr"
bitrate = 20000
gop = 300
fps = 120
)");

        const EncoderSettings settings = load_encoder_settings();
        CHECK(settings.bitrate == GT::ENCODER_BITRATE_MAX);
        CHECK(settings.gop == GT::ENCODER_GOP_MAX);
        CHECK(settings.fps == GT::ENCODER_FPS_MAX);
    }
}

TEST_CASE_METHOD(EncoderSettingsFixture,
                 "Encoder settings use defaults for invalid TOML",
                 "[encoder][config]")
{
    write_config("[encoder\nbitrate = 12000\n");

    CHECK(load_encoder_settings() == EncoderSettings {});
}

TEST_CASE("Encoder settings apply in runtime order", "[encoder][startup]")
{
    const EncoderSettings settings {
        .rate_control = EncoderRateControl::VBR,
        .bitrate = 12000,
        .gop = 180,
        .fps = 30,
    };
    RecordingEncoderOperations recorder;
    const EncoderSettingsOperations operations = recorder.operations();

    REQUIRE(apply_encoder_settings(settings, operations));
    CHECK(recorder.calls
          == std::vector<std::string> {
              "rate_control:vbr",
              "bitrate:12000",
              "gop:180",
              "fps:30",
          });
}

TEST_CASE("Encoder rate control maps codec and mode to AX values",
          "[encoder][startup]")
{
    CHECK(encoder_rate_control_mode(EncoderCodec::H264, EncoderRateControl::CBR)
          == AX_VENC_RC_MODE_H264CBR);
    CHECK(encoder_rate_control_mode(EncoderCodec::H264, EncoderRateControl::VBR)
          == AX_VENC_RC_MODE_H264VBR);
    CHECK(encoder_rate_control_mode(EncoderCodec::H265, EncoderRateControl::CBR)
          == AX_VENC_RC_MODE_H265CBR);
    CHECK(encoder_rate_control_mode(EncoderCodec::H265, EncoderRateControl::VBR)
          == AX_VENC_RC_MODE_H265VBR);
}

TEST_CASE("Encoder settings skip zero FPS", "[encoder][startup]")
{
    EncoderSettings settings;
    settings.fps = 0;
    RecordingEncoderOperations recorder;
    const EncoderSettingsOperations operations = recorder.operations();

    REQUIRE(apply_encoder_settings(settings, operations));
    CHECK(recorder.calls
          == std::vector<std::string> {
              "rate_control:vbr",
              "bitrate:8000",
              "gop:50",
          });
}

TEST_CASE("Encoder settings stop on the first apply failure",
          "[encoder][startup]")
{
    const EncoderSettings settings {
        .rate_control = EncoderRateControl::CBR,
        .bitrate = 10000,
        .gop = 100,
        .fps = 25,
    };

    SECTION("rate control failure")
    {
        RecordingEncoderOperations recorder {.fail_at = "rate_control"};
        const EncoderSettingsOperations operations = recorder.operations();
        CHECK_FALSE(apply_encoder_settings(settings, operations));
        CHECK(recorder.calls == std::vector<std::string> {"rate_control:cbr"});
    }

    SECTION("bitrate failure")
    {
        RecordingEncoderOperations recorder {.fail_at = "bitrate"};
        const EncoderSettingsOperations operations = recorder.operations();
        CHECK_FALSE(apply_encoder_settings(settings, operations));
        CHECK(recorder.calls
              == std::vector<std::string> {
                  "rate_control:cbr",
                  "bitrate:10000",
              });
    }

    SECTION("GOP failure")
    {
        RecordingEncoderOperations recorder {.fail_at = "gop"};
        const EncoderSettingsOperations operations = recorder.operations();
        CHECK_FALSE(apply_encoder_settings(settings, operations));
        CHECK(recorder.calls
              == std::vector<std::string> {
                  "rate_control:cbr",
                  "bitrate:10000",
                  "gop:100",
              });
    }

    SECTION("FPS failure")
    {
        RecordingEncoderOperations recorder {.fail_at = "fps"};
        const EncoderSettingsOperations operations = recorder.operations();
        CHECK_FALSE(apply_encoder_settings(settings, operations));
        CHECK(recorder.calls
              == std::vector<std::string> {
                  "rate_control:cbr",
                  "bitrate:10000",
                  "gop:100",
                  "fps:25",
              });
    }
}
