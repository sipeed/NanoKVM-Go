#include <chrono>
#include <ctime>
#include <fstream>
#include <string>
#include <sys/stat.h>

#include <fmt/format.h>

#include "ax_runtime.hpp"
#include "lt7911_monitor.hpp"
#include "lt7911_utils.hpp"
#include "ocr_latest_frame_store.hpp"
#include "snapshot_http_server.hpp"
#include "snapshot_service.hpp"
#include "snapshot_socket_server.hpp"
#include "timelapse_service.hpp"

namespace {

bool is_ax_module_loaded()
{
    struct stat st;
    return stat("/sys/module/ax_audio", &st) == 0;
}

}  // namespace

bool wait_for_ax_module(int timeout_seconds)
{
    const int interval_ms = 500;
    int waited_ms = 0;

    while (waited_ms < timeout_seconds * 1000) {
        if (is_ax_module_loaded()) {
            return true;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(interval_ms));
        waited_ms += interval_ms;
    }

    return false;
}

bool detect_initial_resolution(AxVinData& ax_vin, ILogger& logger, int retries, int interval_ms)
{
    for (int i = 0; i < retries; ++i) {
        std::ifstream width_file("/proc/lt7911_info/width");
        std::ifstream height_file("/proc/lt7911_info/height");
        std::ifstream fps_file("/proc/lt7911_info/fps");

        int w = 0, h = 0, f = 0;

        if (width_file >> w && height_file >> h && fps_file >> f) {
            if (w > 0 && h > 0 && f > 0) {
                ax_vin.width = w;
                ax_vin.height = h;
                ax_vin.fps = f;
                logger->info("LT7911 detected: {}x{}@{}", w, h, f);
                return true;
            }
        }

        if (i + 1 < retries) {
            std::this_thread::sleep_for(std::chrono::milliseconds(interval_ms));
        }
    }

    logger->warn("LT7911 signal not ready or invalid after timeout. Using default 1080p60.");
    ax_vin.width = 1920;
    ax_vin.height = 1080;
    ax_vin.fps = 60;
    return false;
}

bool run_dump_raw_mode(AxeraUtils& axera_utils, const AxVinData& ax_vin, ILogger& logger)
{
    if (!wait_for_ax_module(10)) {
        logger->error("AX module not loaded");
        return false;
    }

    if (!axera_utils.init()) {
        logger->error("Axera util init failed");
        return false;
    }

    if (!axera_utils.pool_init(3840, 2160)) {
        logger->error("Axera common pool init failed");
        return false;
    }

    if (!axera_utils.vin_init()) {
        logger->error("Axera vin init failed");
        return false;
    }

    auto now = std::chrono::system_clock::now();
    auto time_t = std::chrono::system_clock::to_time_t(now);
    std::tm tm = *std::localtime(&time_t);
    char time_str[64];
    std::strftime(time_str, sizeof(time_str), "%Y%m%d_%H%M%S", &tm);

    std::string filename = fmt::format("/root/vin_{}_{:d}x{:d}.yuyv422",
                                       time_str, ax_vin.width, ax_vin.height);

    logger->info("Dumping YUYV422 frame to {}...", filename);
    if (axera_utils.dump_yuyv_to_file(filename, 1000)) {
        logger->info("YUYV422 frame dumped successfully");
    } else {
        logger->error("Failed to dump YUYV422 frame");
    }

    axera_utils.vin_deinit();
    return true;
}

std::thread start_ax_runtime_thread(AxeraUtils& axera_utils,
                                    CaptureCoordinator& capture_coordinator,
                                    Lt7911Utils* lt7911_utils,
                                    std::unique_ptr<FrameCapture>& frame_capture,
                                    std::shared_ptr<AxVinData> ax_vin,
                                    EncoderConfig encoder_config,
                                    std::atomic<bool>& running,
                                    ILogger& logger,
                                    std::optional<snapshot_http::Config>
                                        snapshot_http_config)
{
    const bool snapshot_http_mode =
        snapshot_http_config.has_value() && snapshot_http_config->enabled;
    if (snapshot_http_mode) {
        encoder_config.venc_enable = false;
        encoder_config.snapshot.enable = true;
        encoder_config.snapshot.periodic_enable = false;
        encoder_config.timelapse.enable = false;
    }
    axera_utils.set_venc_enabled(encoder_config.venc_enable);
    axera_utils.set_jenc_enabled(encoder_config.snapshot.enable);

    return std::thread([&axera_utils, &capture_coordinator, lt7911_utils,
                        &frame_capture, ax_vin, encoder_config, &running, &logger,
                        snapshot_http_mode,
                        snapshot_http_config = std::move(snapshot_http_config)]() {
        logger->info("Axera vin init...");
        const bool enable_venc_stream = encoder_config.venc_enable;
        const SnapshotConfig snapshot_config = encoder_config.snapshot;
        const TimelapseConfig timelapse_config = encoder_config.timelapse;
        const bool enable_jenc = snapshot_config.enable;
        const bool enable_periodic_snapshot =
            snapshot_config.enable && snapshot_config.periodic_enable;
        if (snapshot_config.periodic_enable && !snapshot_config.enable) {
            logger->warn("Periodic snapshot requested but snapshot.enable is false; "
                         "periodic snapshot worker will not start");
        }

        bool vin_initialized = false;
        bool venc_initialized = false;
        bool jenc_initialized = false;
        bool timelapse_vdec_initialized = false;
        bool timelapse_venc_initialized = false;
        bool linked = false;
        std::thread lt7911_thread;
        OCRLatestFrameStore ocr_latest_frame_store;
        TimelapseService timelapse_service(axera_utils, timelapse_config, logger);
        SnapshotService snapshot_service(capture_coordinator,
                                         snapshot_config,
                                         running,
                                         logger,
                                         &timelapse_service,
                                         &ocr_latest_frame_store);
        SnapshotSocketServer snapshot_socket_server(
            capture_coordinator,
            ocr_latest_frame_store,
            [enable_jenc]() { return enable_jenc; },
            logger);
        std::unique_ptr<SnapshotHttpServer> snapshot_http_server;
        if (snapshot_http_mode) {
            snapshot_http_server = std::make_unique<SnapshotHttpServer>(
                capture_coordinator, logger);
        }

        auto cleanup = [&]() {
            if (snapshot_http_server) {
                snapshot_http_server->stop();
            }
            if (snapshot_http_mode) {
                capture_coordinator.stop();
            } else {
                snapshot_service.stop();
                ocr_latest_frame_store.stop();
                snapshot_socket_server.stop();
                capture_coordinator.stop();
            }
            if (lt7911_thread.joinable()) {
                lt7911_thread.join();
            }
            if (!snapshot_http_mode) {
                timelapse_service.stop();
                if (frame_capture) {
                    frame_capture->stop();
                    frame_capture.reset();
                    logger->info("Frame capture threads stopped");
                }
            }
            if (linked && !axera_utils.link_deinit()) {
                logger->error("Axera Link deinit failed");
            }
            if (timelapse_venc_initialized && !axera_utils.timelapse_venc_deinit()) {
                logger->error("Timelapse H264 encoder deinit failed");
            }
            if (timelapse_vdec_initialized && !axera_utils.vdec_deinit()) {
                logger->error("Timelapse VDEC deinit failed");
            }
            if (jenc_initialized && !axera_utils.jenc_deinit()) {
                logger->error("JENC deinit failed");
            }
            if (venc_initialized && !axera_utils.venc_deinit()) {
                logger->error("VENC deinit failed");
            }
            if (vin_initialized && !axera_utils.vin_deinit()) {
                logger->error("VIN deinit failed");
            }
        };

        if (!axera_utils.init()) {
            logger->error("Axera util init failed");
            running = false;
            return;
        }

        if (!axera_utils.pool_init(3840, 2160)) {
            logger->error("Axera common pool init failed");
            running = false;
            return;
        }

        if (!axera_utils.vin_init()) {
            logger->error("Axera vin init failed");
            running = false;
            return;
        }
        vin_initialized = true;

        if (enable_venc_stream && !axera_utils.venc_init()) {
            logger->error("Axera VENC init failed");
            running = false;
            cleanup();
            return;
        }
        venc_initialized = enable_venc_stream;

        if (enable_venc_stream) {
            const int venc_chn = axera_utils.get_venc_chn();
            const EncoderCodec codec =
                axera_utils.encode_mode() == AxeraUtils::EncodeMode::H265
                    ? EncoderCodec::H265
                    : EncoderCodec::H264;
            const EncoderSettingsOperations operations {
                .set_rate_control =
                    [&axera_utils, venc_chn, codec](EncoderRateControl rate_control) {
                        return axera_utils.venc_set_rc_mode(
                            venc_chn,
                            encoder_rate_control_mode(codec, rate_control));
                    },
                .set_bitrate =
                    [&axera_utils, venc_chn](unsigned int bitrate) {
                        return axera_utils.venc_set_bitrate(venc_chn, bitrate);
                    },
                .set_gop =
                    [&axera_utils, venc_chn](unsigned int gop) {
                        return axera_utils.venc_set_gop(venc_chn, gop);
                    },
                .set_fps =
                    [&axera_utils, venc_chn](int fps) {
                        return axera_utils.venc_set_fps(
                            venc_chn, 0.0f, static_cast<float>(fps));
                    },
            };
            if (!apply_encoder_settings(encoder_config.settings, operations)) {
                logger->error("Failed to apply encoder settings");
                running = false;
                cleanup();
                return;
            }
            logger->info("Encoder settings applied before VIN->VENC link");
        }

        if (enable_jenc && !axera_utils.jenc_init()) {
            logger->error("Axera JENC init failed");
            running = false;
            cleanup();
            return;
        }
        jenc_initialized = enable_jenc;

        if (timelapse_config.enable && !axera_utils.vdec_init()) {
            logger->error("Timelapse VDEC init failed");
            running = false;
            cleanup();
            return;
        }
        timelapse_vdec_initialized = timelapse_config.enable;

        if (timelapse_config.enable &&
            !axera_utils.timelapse_venc_init(
                static_cast<unsigned int>(timelapse_config.qp))) {
            logger->error("Timelapse H264 encoder init failed");
            running = false;
            cleanup();
            return;
        }
        timelapse_venc_initialized = timelapse_config.enable;

        if (!axera_utils.link_init()) {
            logger->error("Axera Link init failed");
            running = false;
            cleanup();
            return;
        }
        linked = true;

        if (enable_jenc) {
            capture_coordinator.start();
            logger->info("Capture coordinator started after VIN->JENC link");
        }
        if (snapshot_http_mode && !capture_coordinator.is_running()) {
            logger->error(
                "Capture coordinator is not ready for MJPEG HTTP mode");
            running = false;
            cleanup();
            return;
        }
        if (snapshot_http_mode) {
            if (!snapshot_http_server->start(*snapshot_http_config)) {
                logger->error("Failed to start MJPEG HTTP server");
                running = false;
                cleanup();
                return;
            }
            logger->info(
                "MJPEG HTTP listener started after VIN->JENC link and "
                "capture coordinator");
        } else if (!snapshot_socket_server.start()) {
            logger->error("Failed to start snapshot socket server");
            running = false;
            cleanup();
            return;
        }

        if (enable_venc_stream) {
            frame_capture = std::make_unique<FrameCapture>(axera_utils, running);
            frame_capture->set_lt7911(lt7911_utils);
            if (lt7911_utils) {
                frame_capture->set_input_stable(false);
            }
            if (!frame_capture->start(ax_vin->width, ax_vin->height, ax_vin->fps)) {
                logger->error("Failed to start frame capture");
                running = false;
                cleanup();
                return;
            }
        } else {
            logger->info("Skip H264/H265 frame capture (VENC disabled)");
        }

        if (!snapshot_http_mode && !timelapse_service.start()) {
            logger->error("Timelapse service start failed");
            running = false;
            cleanup();
            return;
        }

        if (!snapshot_http_mode && enable_periodic_snapshot) {
            snapshot_service.start();
        }

        if (lt7911_utils) {
            lt7911_thread = start_lt7911_monitor_thread(
                ax_vin, axera_utils, *lt7911_utils, frame_capture, running, logger);
        } else {
            logger->warn("LT7911 monitor is unavailable");
        }

        while (running) {
            if (snapshot_http_mode && snapshot_http_server
                && !snapshot_http_server->is_running()) {
                logger->error(
                    "MJPEG HTTP listener stopped unexpectedly; stopping AX "
                    "runtime");
                running = false;
                break;
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(200));
        }

        cleanup();

        logger->info("Axera vin thread exiting...");
    });
}
