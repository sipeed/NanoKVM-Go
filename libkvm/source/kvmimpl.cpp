#include <array>
#include <cerrno>
#include <filesystem>
#include <fstream>
#include <memory>
#include <mutex>
#include <new>
#include <unordered_set>

#include <linux/videodev2.h>
#include <opus/opus_defines.h>
#include <spdlog/spdlog.h>
#include <toml++/toml.h>
#include <unistd.h>

#include "global_para/toml_default_para.hpp"
#include "kvm_context.hpp"
#include "libkvm/kvm_vision.h"
#include "logger_default.hpp"
#include "pcm/audio_player.hpp"
#include "pcm/pcm_capture.hpp"
#include "pcm/pcm_device.hpp"
#include "version.h"

struct kvmv_audio_playback {
    std::mutex mutex;
    std::unique_ptr<kvm::audio::AudioPlayer> player;

    kvmv_audio_playback()
        : player(std::make_unique<kvm::audio::AudioPlayer>())
    {}
};

namespace {

static_assert(kvm::audio::AudioPlayer::MAX_OPUS_PACKET_BYTES
              == KVMV_AUDIO_MAX_OPUS_PACKET_BYTES);
static_assert(kvm::audio::AudioPlayer::MAX_TIMEOUT_MS
              == KVMV_AUDIO_MAX_TIMEOUT_MS);

kvm::audio::AudioCaptureState& audioCaptureState()
{
    static kvm::audio::AudioCaptureState state;
    return state;
}

void clearAudioOutput(uint8_t** data, uint32_t* size)
{
    if (data != nullptr) {
        *data = nullptr;
    }
    if (size != nullptr) {
        *size = 0;
    }
}

std::mutex& audioPlaybackRegistryMutex()
{
    static std::mutex mutex;
    return mutex;
}

std::unordered_set<kvmv_audio_playback_t*>& audioPlaybackRegistry()
{
    static std::unordered_set<kvmv_audio_playback_t*> handles;
    return handles;
}

int mapPcmPlaybackError(const kvm::audio::PcmError& error) noexcept
{
    const int code =
        error.errorCode() < 0 ? -error.errorCode() : error.errorCode();
    switch (code) {
    case EBADF:
    case EBADFD:
        return KVMV_AUDIO_ERR_INVALID_STATE;
    case ETIMEDOUT:
        return KVMV_AUDIO_ERR_TIMEOUT;
    case ENOENT:
    case ENODEV:
    case ENXIO:
        return KVMV_AUDIO_ERR_UNAVAILABLE;
    case EBUSY:
    case EAGAIN:
        return KVMV_AUDIO_ERR_BUSY;
    case EACCES:
    case EPERM:
        return KVMV_AUDIO_ERR_PERMISSION;
    case ENOMEM:
        return KVMV_AUDIO_ERR_NO_MEMORY;
    default:
        return KVMV_AUDIO_ERR_DEVICE_IO;
    }
}

template<typename Function>
int invokeAudioPlaybackApi(Function&& function) noexcept
{
    try {
        function();
        return KVMV_AUDIO_OK;
    } catch (const kvm::audio::OpusDecodeError& error) {
        if (error.opusCode() == OPUS_ALLOC_FAIL) {
            return KVMV_AUDIO_ERR_NO_MEMORY;
        }
        return KVMV_AUDIO_ERR_CODEC;
    } catch (const kvm::audio::PcmWriteTimeout&) {
        return KVMV_AUDIO_ERR_TIMEOUT;
    } catch (const kvm::audio::PcmError& error) {
        return mapPcmPlaybackError(error);
    } catch (const kvm::audio::AudioPlayerStateError&) {
        return KVMV_AUDIO_ERR_INVALID_STATE;
    } catch (const std::invalid_argument&) {
        return KVMV_AUDIO_ERR_INVALID_ARGUMENT;
    } catch (const std::bad_alloc&) {
        return KVMV_AUDIO_ERR_NO_MEMORY;
    } catch (...) {
        return KVMV_AUDIO_ERR_INTERNAL;
    }
}

void closeAllAudioPlaybacks() noexcept
{
    try {
        std::lock_guard<std::mutex> registry_lock(audioPlaybackRegistryMutex());
        for (auto* handle : audioPlaybackRegistry()) {
            if (handle == nullptr) {
                continue;
            }
            std::lock_guard<std::mutex> handle_lock(handle->mutex);
            if (handle->player != nullptr) {
                handle->player->close();
                handle->player.reset();
            }
        }
    } catch (...) {
        // kvmv_deinit() is a C ABI shutdown boundary and must never throw.
    }
}

}  // namespace

#ifdef __cplusplus
extern "C" {
#endif

static inline void ensure_logger_config_file()
{
    namespace fs = std::filesystem;
    namespace GT = global_para::toml_default;
    fs::path toml_path(std::string(GT::CONFIG_FILE_PATH));

    if (fs::exists(toml_path)) {
        return;
    }

    fs::create_directories(toml_path.parent_path());
    std::ofstream ofs(toml_path);
    if (ofs.is_open()) {
        ofs << "[log]\n";
        ofs << "level = \"" << GT::GLOBAL_LOG_LEVEL_DEF << "\"\n";
    }
}

static inline int fetch_frame(uint8_t type, uint8_t** pp_data, uint32_t* p_size) {
    auto& ctx = KVMContext::getInstance();
    static FrameRef frame;

    int ret = -1;
    switch (type) {
        case IMG_MJPEG_TYPE:
            return IMG_NOT_EXIST;
        case IMG_H264_TYPE_SPS:
        case IMG_H265_TYPE_SPS:
            ret = ctx.getVideoReader().read_frame(frame);
            break;
        default:
            return IMG_NOT_EXIST;
    }

    if (ret != 0 || frame.data == nullptr || frame.size == 0)
        return IMG_NOT_EXIST;

    *pp_data = const_cast<uint8_t*>(frame.data);
    *p_size = frame.size;

    if (frame.format == V4L2_PIX_FMT_HEVC) {
        return frame.key ? IMG_H265_TYPE_IF : IMG_H265_TYPE_PF;
    } else {
        return frame.key ? IMG_H264_TYPE_IF : IMG_H264_TYPE_PF;
    }
}

static inline int return_empty_image(uint8_t** pp_data, uint32_t* p_size)
{
    if (pp_data != nullptr) {
        *pp_data = nullptr;
    }
    if (p_size != nullptr) {
        *p_size = 0;
    }
    return IMG_NOT_EXIST;
}

__attribute__((visibility("default"))) void kvmv_init(uint8_t _debug_info_en)
{
    auto& ctx = KVMContext::getInstance();
    auto& encoder_state = ctx.getEncoderState();

    try {
        ensure_logger_config_file();
    } catch (const std::exception& err) {
        spdlog::warn("[kvmv_init] Failed to create config file<{}>: {}",
                     global_para::toml_default::CONFIG_FILE_PATH, err.what());
    }

    toml::table logger_config;
    try {
        logger_config = toml::parse_file(std::string(global_para::toml_default::CONFIG_FILE_PATH));
    } catch (const toml::parse_error& err) {
        spdlog::warn("[kvmv_init] Failed to parse config file<{}>: {}, skip",
                     global_para::toml_default::CONFIG_FILE_PATH, err.what());
    }

    std::string logger_level = logger_config["log"]["level"]
                                   .value_or(std::string(global_para::toml_default::GLOBAL_LOG_LEVEL_DEF));
    ctx.setLoggerLevel(logger_level);

    if (logger_level == "default") {
        if (_debug_info_en) {
            spdlog::set_level(spdlog::level::debug);
            ctx.setLoggerLevel("debug");
        } else {
            spdlog::set_level(spdlog::level::warn);
            ctx.setLoggerLevel("warn");
        }
    } else {
        spdlog::set_level(spdlog::level::from_str(logger_level));
    }

    auto logger = std::make_shared<DefaultLogger>("main");
    logger->logger_init(ctx.getLoggerLevel(), "console");
    ctx.setLogger(logger);

    encoder_state.gop = 0;
    encoder_state.fps = 0;
    encoder_state.rate_control_mode = VENC_VBR;
    ctx.setHdmiEnabled(true);
    audioCaptureState().start();
    ctx.setDeinitialized(false);
}

__attribute__((visibility("default"))) int kvmv_read_img(
    uint16_t _width,
    uint16_t _height,
    uint8_t _type,
    uint16_t _qlty,
    uint8_t** _pp_kvm_data,
    uint32_t* _p_kvmv_data_size)
{
    (void)_width;
    (void)_height;
    (void)_qlty;

    auto& ctx = KVMContext::getInstance();

    if (ctx.isDeinitialized())
        return IMG_VENC_ERROR;

    if (!ctx.isHdmiEnabled())
        return return_empty_image(_pp_kvm_data, _p_kvmv_data_size);

    return fetch_frame(_type, _pp_kvm_data, _p_kvmv_data_size);
}

__attribute__((visibility("default"))) int kvmv_get_sps_frame(
    uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size)
{
    (void)_pp_kvm_data; (void)_p_kvmv_data_size;
    return 0;
}

__attribute__((visibility("default"))) int kvmv_get_pps_frame(
    uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size)
{
    (void)_pp_kvm_data; (void)_p_kvmv_data_size;
    return 0;
}

__attribute__((visibility("default"))) int kvmv_free_data(uint8_t** _pp_kvm_data)
{
    (void)_pp_kvm_data;
    return 0;
}

__attribute__((visibility("default"))) void kvmv_free_all_data() {}

__attribute__((visibility("default"))) int kvmv_set_gop(uint16_t _gop)
{
    auto& ctx = KVMContext::getInstance();
    int ret = ctx.getControlClient().set_gop(static_cast<unsigned int>(_gop));
    if (ret != 0) {
        return -1;
    }
    ctx.getEncoderState().gop = _gop;
    return 0;
}

__attribute__((visibility("default"))) int kvmv_set_bitrate(uint16_t _bitrate)
{
    auto& ctx = KVMContext::getInstance();
    int ret = ctx.getControlClient().set_bitrate(static_cast<unsigned int>(_bitrate));
    if (ret != 0) {
        return -1;
    }
    return 0;
}

__attribute__((visibility("default"))) int kvmv_set_fps(uint8_t _fps)
{
    auto& ctx = KVMContext::getInstance();
    int ret = ctx.getControlClient().set_fps(0, static_cast<float>(_fps));
    if (ret != 0) {
        return -1;
    }
    ctx.getEncoderState().fps = _fps;
    return 0;
}

__attribute__((visibility("default"))) int kvmv_get_fps(void)
{
    auto& ctx = KVMContext::getInstance();
    return ctx.getEncoderState().fps;
}

__attribute__((visibility("default"))) void kvmv_deinit()
{
    auto& ctx = KVMContext::getInstance();
    ctx.setDeinitialized(true);
    audioCaptureState().stop();
    closeAllAudioPlaybacks();
    ctx.getVideoReader().close_stream();
    spdlog::drop_all();
}

__attribute__((visibility("default"))) int kvmv_hdmi_control(uint8_t _en)
{
    auto& ctx = KVMContext::getInstance();
    ctx.setHdmiEnabled(_en != 0);
    return 0;
}

__attribute__((visibility("default"))) int kvmv_set_rate_control(uint8_t mode)
{
    auto& ctx = KVMContext::getInstance();
    int ret = ctx.getControlClient().set_rc_mode(static_cast<int>(mode));
    if (ret != 0) {
        return -1;
    }
    ctx.getEncoderState().rate_control_mode = mode;
    return 0;
}

__attribute__((visibility("default"))) int kvmv_force_idr(void)
{
    auto& ctx = KVMContext::getInstance();

    if (ctx.isDeinitialized())
        return -1;

    // Non-blocking: only wakes the reader's IDR worker thread, which performs
    // the actual control-socket round-trip off the caller's thread. Bursts are
    // coalesced and rate-limited inside UnixVideoFrameReader.
    ctx.getVideoReader().request_idr();
    return 0;
}

__attribute__((visibility("default"))) void kvmv_close_video_stream(void)
{
    auto& ctx = KVMContext::getInstance();
    ctx.getVideoReader().close_stream();
}

__attribute__((visibility("default"))) int kvmv_read_audio(
    uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size)
{
    clearAudioOutput(_pp_kvm_data, _p_kvmv_data_size);
    if (_pp_kvm_data == nullptr || _p_kvmv_data_size == nullptr) {
        return -1;
    }

    auto& ctx = KVMContext::getInstance();
    auto logger = ctx.getLogger();
    if (ctx.isDeinitialized()) {
        return -1;
    }

    thread_local std::array<uint8_t,
                            kvm::audio::AudioCapturer::MAX_PACKET_BYTES>
        audio_output {};
    kvm::audio::AudioCaptureReadResult result;
    try {
        result =
            audioCaptureState().read(audio_output.data(), audio_output.size());
    } catch (const std::exception& error) {
        if (logger != nullptr) {
            (*logger)->error("[kvmv_read_audio] Unexpected audio error: {}",
                             error.what());
        }
        return -1;
    } catch (...) {
        if (logger != nullptr) {
            (*logger)->error(
                "[kvmv_read_audio] Unexpected unknown audio error");
        }
        return -1;
    }
    if (result.device_initialized && logger != nullptr) {
        (*logger)->info(
            "[kvmv_read_audio] Audio device initialized successfully");
    }
    switch (result.status) {
    case kvm::audio::AudioCaptureReadStatus::Success:
        *_pp_kvm_data = result.size == 0 ? nullptr : audio_output.data();
        *_p_kvmv_data_size = result.size;
        return 0;
    case kvm::audio::AudioCaptureReadStatus::Failed:
        if (logger != nullptr) {
            (*logger)->error("[kvmv_read_audio] Audio capture failed: {}",
                             result.error);
        }
        return -1;
    case kvm::audio::AudioCaptureReadStatus::NoData:
    case kvm::audio::AudioCaptureReadStatus::RetryLater:
    case kvm::audio::AudioCaptureReadStatus::Stopped:
        return -1;
    }

    return -1;
}

__attribute__((visibility("default"))) void kvmv_reset_audio_capture(void)
{
    auto& state = audioCaptureState();
    state.stop();
    if (!KVMContext::getInstance().isDeinitialized()) {
        state.start();
    }
}

__attribute__((visibility("default"))) int kvmv_audio_playback_open(
    kvmv_audio_playback_t** out)
{
    if (out == nullptr) {
        return KVMV_AUDIO_ERR_INVALID_ARGUMENT;
    }
    if (*out != nullptr) {
        return KVMV_AUDIO_ERR_INVALID_STATE;
    }
    return invokeAudioPlaybackApi([out]() {
        if (KVMContext::getInstance().isDeinitialized()) {
            throw kvm::audio::AudioPlayerStateError("libkvm is deinitialized");
        }
        auto handle = std::make_unique<kvmv_audio_playback_t>();
        {
            std::lock_guard<std::mutex> lock(audioPlaybackRegistryMutex());
            if (KVMContext::getInstance().isDeinitialized()) {
                throw kvm::audio::AudioPlayerStateError(
                    "libkvm was deinitialized while opening audio playback");
            }
            audioPlaybackRegistry().insert(handle.get());
        }
        *out = handle.release();
    });
}

__attribute__((visibility("default"))) int kvmv_audio_playback_write_opus(
    kvmv_audio_playback_t* playback,
    const uint8_t* opus,
    uint32_t size,
    uint32_t timeout_ms,
    uint32_t flags)
{
    if (playback == nullptr) {
        return KVMV_AUDIO_ERR_INVALID_STATE;
    }
    if (opus == nullptr || size == 0 || size > KVMV_AUDIO_MAX_OPUS_PACKET_BYTES
        || timeout_ms == 0 || timeout_ms > KVMV_AUDIO_MAX_TIMEOUT_MS
        || (flags & ~KVMV_AUDIO_FLAG_VALID_MASK) != 0) {
        return KVMV_AUDIO_ERR_INVALID_ARGUMENT;
    }
    return invokeAudioPlaybackApi([=]() {
        if (KVMContext::getInstance().isDeinitialized()) {
            throw kvm::audio::AudioPlayerStateError("libkvm is deinitialized");
        }
        std::lock_guard<std::mutex> lock(playback->mutex);
        if (playback->player == nullptr) {
            throw kvm::audio::AudioPlayerStateError(
                "Audio playback handle is closed");
        }
        playback->player->writeOpus(
            opus,
            size,
            timeout_ms,
            (flags & KVMV_AUDIO_FLAG_DISCONTINUITY) != 0);
    });
}

__attribute__((visibility("default"))) void kvmv_audio_playback_close(
    kvmv_audio_playback_t** playback)
{
    if (playback == nullptr || *playback == nullptr) {
        return;
    }

    std::unique_ptr<kvmv_audio_playback_t> handle(*playback);
    *playback = nullptr;

    try {
        std::lock_guard<std::mutex> registry_lock(audioPlaybackRegistryMutex());
        audioPlaybackRegistry().erase(handle.get());
    } catch (...) {
        // Continue closing the caller-owned handle even during shutdown.
    }

    try {
        std::lock_guard<std::mutex> handle_lock(handle->mutex);
        if (handle->player != nullptr) {
            handle->player->close();
            handle->player.reset();
        }
    } catch (...) {
        // The unique_ptr still destroys all successfully-created resources.
    }
}

#ifdef __cplusplus
}
#endif
