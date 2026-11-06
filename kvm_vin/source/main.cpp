#include <chrono>
#include <cstdio>
#include <filesystem>
#include <fstream>
#include <memory>
#include <optional>
#include <string_view>
#include <thread>
#include <csignal>

// configs
#include <global_para/toml_default_para.hpp>

// buildinfo
#include <buildinfo/version.hpp>

// tools
#include <cxxopts.hpp>
#include <tools/generator.hpp>
#include <tools/scheduler.hpp>
#include <tools/segfault_stack_tracer.h>
#include <tools/system_uptime.hpp>
#include <tools/termination_handler.hpp>
#include <tools/toml_file_parser.hpp>
#include <tools/tools.hpp>
#include <nlohmann/json.hpp>

// logger
#include <logger/default2.hpp>
#include <logger/file_sink.hpp>
#include <logger/tools.hpp>

// axera
#include "ax_utils.hpp"
#include "ax_runtime.hpp"
#include "capture_coordinator.hpp"
#include "control_server.hpp"
#include "encoder_settings.hpp"
#include "frame_capture.hpp"
#include "lt7911_utils.hpp"
#include "snapshot_http_contract.hpp"

static std::unique_ptr<ILogger> main_logger;
static std::unique_ptr<AxeraUtils> axera_utils;
static std::unique_ptr<Lt7911Utils> lt7911_utils;
static std::atomic<bool> running(true);
static std::unique_ptr<FrameCapture> frame_capture;

struct LoggerCfg {
    std::string level;
    std::string type;
    std::string file;
};

void signal_handler(int)
{
    running = false;
}

static void create_toml_config()
{
    using TP = tools::TomlFileParser;
    namespace GT = global_para::toml_default;

    const auto to_string = [](std::string_view view) {
        return std::string(view);
    };

#define SET_TP_STRING(name) \
    TP::set(to_string(GT::name##_PATH), to_string(GT::name##_DEF))

#define SET_TP(name) TP::set(to_string(GT::name##_PATH), GT::name##_DEF)

    SET_TP_STRING(GLOBAL_LOG_LEVEL);
    SET_TP_STRING(GLOBAL_LOG_TYPE);
    SET_TP_STRING(GLOBAL_LOG_FILE);
    SET_TP(H264_IDLE_TIMEOUT_MS);
    SET_TP(H264_IDLE_POLL_MS);
    SET_TP(H264_IDLE_POWER_OFF_LT7911);
    SET_TP(DEVICE_ENABLE);
    SET_TP_STRING(DEVICE_MODE);
    SET_TP(SNAPSHOT_ENABLE);
    SET_TP(SNAPSHOT_PERIODIC_ENABLE);
    SET_TP(TIMELAPSE_ENABLE);
    SET_TP(TIMELAPSE_QP);
}

int main(int argc, char* argv[])
{
    tools::SegfaultStackTracker::init();
    tools::TerminatorHandler::init();

    bool debug_mode = false;
    bool dump_raw_mode = false;
    snapshot_http::Config jpeg_http_config;
    {
        cxxopts::Options cli_options(argv[0], "VIN App for NanoKVM-Go");
        cli_options.add_options()("h,help", "Print help message")(
            "v,version", "Print VERSION")("d,debug", "Enable debug mode")(
            "dump-raw", "Dump one YUYV422 frame")(
            "jpeg-http", "Enable the MJPEG HTTP stream (GET /stream.mjpg)")(
            "jpeg-http-bind",
            "MJPEG HTTP bind address",
            cxxopts::value<std::string>()->default_value(
                std::string(snapshot_http::DEFAULT_BIND_HOST)),
            "ADDRESS")("jpeg-http-port",
                       "MJPEG HTTP listen port",
                       cxxopts::value<int>()->default_value(
                           std::to_string(snapshot_http::DEFAULT_PORT)),
                       "PORT")("jpeg-http-token-file",
                               "File containing the MJPEG HTTP bearer token",
                               cxxopts::value<std::string>(),
                               "PATH");

        try {
            auto cli_result = cli_options.parse(argc, argv);
            if (cli_result.count("help")) {
                fmt::println("{}", cli_options.help());
                return 0;
            }
            if (cli_result.count("version")) {
                fmt::println(
                    "{} - {} - {}", GIT_HASH, GIT_DIRTY, BUILD_TIMESTAMP);
                return 0;
            }

            debug_mode = cli_result.count("debug") != 0;
            dump_raw_mode = cli_result.count("dump-raw") != 0;
            jpeg_http_config.enabled = cli_result.count("jpeg-http") != 0;
            jpeg_http_config.bind_host =
                cli_result["jpeg-http-bind"].as<std::string>();
            jpeg_http_config.port = cli_result["jpeg-http-port"].as<int>();
            if (cli_result.count("jpeg-http-token-file")) {
                jpeg_http_config.token_file =
                    cli_result["jpeg-http-token-file"].as<std::string>();
            }

            const bool has_http_modifier =
                cli_result.count("jpeg-http-bind") != 0
                || cli_result.count("jpeg-http-port") != 0
                || cli_result.count("jpeg-http-token-file") != 0;
            if (!jpeg_http_config.enabled && has_http_modifier) {
                fmt::println(stderr,
                             "error: --jpeg-http-bind, --jpeg-http-port and "
                             "--jpeg-http-token-file require --jpeg-http");
                return 2;
            }

            const auto validation =
                snapshot_http::validate_config(jpeg_http_config, dump_raw_mode);
            if (!validation.ok()) {
                fmt::println(stderr, "error: {}", validation.message);
                return 2;
            }

            if (dump_raw_mode || jpeg_http_config.enabled) {
                debug_mode = true;
            }
        } catch (const cxxopts::exceptions::exception& error) {
            fmt::println(stderr, "error: {}", error.what());
            fmt::println(stderr, "Run with --help to see available options.");
            return 2;
        }
    }

    {
        namespace fs = std::filesystem;
        const auto& path = global_para::toml_default::CONFIG_FILE_PATH;
        fs::path toml_path(path);
        bool toml_cfg_exists = fs::exists(path);
        if (!toml_cfg_exists) {
            fs::create_directory(toml_path.parent_path());
            std::ofstream(toml_path).close();
        }
        tools::TomlFileParser::set_path(path);
        if (!toml_cfg_exists) {
            /* write default configs */
            create_toml_config();
        }
    }

#define CFG_STRING(name) \
    tools::TomlFileParser::get<std::string>( \
        std::string(global_para::toml_default::name##_PATH)) \
        .value_or(std::string(global_para::toml_default::name##_DEF))

#define CFG_INT(name) \
    tools::TomlFileParser::get<int>( \
        std::string(global_para::toml_default::name##_PATH)) \
        .value_or(global_para::toml_default::name##_DEF)

#define CFG_BOOL(name) \
    tools::TomlFileParser::get<bool>( \
        std::string(global_para::toml_default::name##_PATH)) \
        .value_or(global_para::toml_default::name##_DEF)

    const LoggerCfg g_logger_cfg {
        .level = debug_mode ? "debug" : LoggerTools::adjust_log_level(CFG_STRING(GLOBAL_LOG_LEVEL)),
        .type = debug_mode ? "console" : CFG_STRING(GLOBAL_LOG_TYPE),
        .file = CFG_STRING(GLOBAL_LOG_FILE)};
    RotatingSink::configure(
        g_logger_cfg.file, RotatingSink::DEF_SIZE, RotatingSink::DEF_FILES);
    GlobalLogger::init(g_logger_cfg.level);

    /* init main logger */
    main_logger = std::make_unique<DefaultLogger2>("main");
    main_logger->logger_init(g_logger_cfg.level, g_logger_cfg.type);
    (*main_logger)->debug("main function start");

    SnapshotConfig snapshot_cfg {
        .enable = CFG_BOOL(SNAPSHOT_ENABLE),
        .periodic_enable = CFG_BOOL(SNAPSHOT_PERIODIC_ENABLE),
    };
    int timelapse_qp = CFG_INT(TIMELAPSE_QP);
    if (timelapse_qp < global_para::toml_default::TIMELAPSE_QP_MIN ||
        timelapse_qp > global_para::toml_default::TIMELAPSE_QP_MAX) {
        timelapse_qp = global_para::toml_default::TIMELAPSE_QP_DEF;
        (*main_logger)->warn("timelapse.qp is invalid, using default {}", timelapse_qp);
    }
    EncoderConfig encoder_cfg {
        .venc_enable = CFG_BOOL(DEVICE_ENABLE),
        .settings = load_encoder_settings([](std::string_view warning) {
            (*main_logger)->warn("{}", warning);
        }),
        .snapshot = snapshot_cfg,
        .timelapse = TimelapseConfig {
            .enable = CFG_BOOL(TIMELAPSE_ENABLE),
            .qp = timelapse_qp,
            .output_path = TimelapseConfig::DEFAULT_OUTPUT_PATH,
        },
    };
    std::optional<snapshot_http::Config> snapshot_http_config;
    if (jpeg_http_config.enabled) {
        encoder_cfg.venc_enable = false;
        encoder_cfg.snapshot.enable = true;
        encoder_cfg.snapshot.periodic_enable = false;
        encoder_cfg.timelapse.enable = false;
        snapshot_http_config = jpeg_http_config;
        (*main_logger)->info(
            "MJPEG HTTP mode: bind={} port={} auth={}; disabling main "
            "VENC, periodic snapshot and timelapse",
            jpeg_http_config.bind_host,
            jpeg_http_config.port,
            jpeg_http_config.token_file.empty() ? "disabled" : "token-file");
    }

    /* ui cfg */
    const LoggerCfg vin_logger_cfg = g_logger_cfg;

    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);

    auto ax_vin = std::make_shared<AxVinData>();
    detect_initial_resolution(*ax_vin, *main_logger);

    /* Initialize axera util */
    axera_utils = std::make_unique<AxeraUtils>(ax_vin->width, ax_vin->height, ax_vin->fps);
    axera_utils->logger_init(vin_logger_cfg.level, vin_logger_cfg.type, vin_logger_cfg.file);
    axera_utils->set_venc_enabled(encoder_cfg.venc_enable);
    axera_utils->set_jenc_enabled(encoder_cfg.snapshot.enable);
    (*main_logger)->info("Encoder enable: VENC={} JENC={} PERIODIC_SNAPSHOT={} VDEC={}",
        encoder_cfg.venc_enable ? "true" : "false",
        encoder_cfg.snapshot.enable ? "true" : "false",
        encoder_cfg.snapshot.periodic_enable ? "true" : "false",
        encoder_cfg.timelapse.enable ? "true" : "false");
    (*main_logger)->info("Encoder settings: RC={} bitrate={} GOP={} FPS={}",
        encoder_rate_control_name(encoder_cfg.settings.rate_control),
        encoder_cfg.settings.bitrate,
        encoder_cfg.settings.gop,
        encoder_cfg.settings.fps);

    {
        std::string mode_str = CFG_STRING(DEVICE_MODE);
        if (mode_str == "h265") {
            axera_utils->set_encode_mode(AxeraUtils::EncodeMode::H265);
            (*main_logger)->info("Encode mode: H265");
        } else {
            if (mode_str != "h264") {
                (*main_logger)->warn("Unknown encode mode='{}', falling back to 'h264'", mode_str);
            }
            axera_utils->set_encode_mode(AxeraUtils::EncodeMode::H264);
            (*main_logger)->info("Encode mode: H264");
        }
    }

    /* Initialize lt7911 util */
    lt7911_utils = std::make_unique<Lt7911Utils>();
    lt7911_utils->logger_init(vin_logger_cfg.level, vin_logger_cfg.type, vin_logger_cfg.file);

    if (dump_raw_mode) {
        if (!run_dump_raw_mode(*axera_utils, *ax_vin, *main_logger)) {
            return -1;
        }
        return 0;
    }

    if (!wait_for_ax_module(10)) {
        return -1;
    }

    CaptureCoordinator capture_coordinator(
        [](const CaptureRequest& request,
           AxeraUtils::SnapshotResult& result) {
            return axera_utils->jenc_snapshot(
                request.x, request.y, request.w, request.h,
                request.quality, request.timeout_ms, result);
        });

    std::thread th = start_ax_runtime_thread(
        *axera_utils, capture_coordinator, lt7911_utils.get(), frame_capture, ax_vin,
        encoder_cfg, running, *main_logger, snapshot_http_config);

    std::thread ctrl_thread;
    if (snapshot_http_config.has_value()) {
        (*main_logger)->info("Skip control server in MJPEG HTTP mode");
    } else {
        ctrl_thread = start_control_server_thread(
            *axera_utils, running, *main_logger);
    }

    if (th.joinable()) {
        th.join();
    }

    if (ctrl_thread.joinable()) {
        ctrl_thread.join();
    }

    spdlog::drop_all();
    return 0;
}
