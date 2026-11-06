
#include "ax_utils.hpp"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <cstring>
#include <exception>
#include <limits>
#include <unistd.h>

#include <ax_ivps_api.h>
#include <ax_venc_api.h>
#include <ax_venc_comm.h>

#include "ax_utils.hpp"

static void init_jpeg_param(const AxeraUtils::Jpeg& jpeg, AX_VENC_JPEG_PARAM_T& param)
{
    static const AX_U8 kYQt[64] = {
        16,11,10,16,24,40,51,61,12,12,14,19,26,58,60,55,
        14,13,16,24,40,57,69,56,14,17,22,29,51,87,80,62,
        18,22,37,56,68,109,103,77,24,35,55,64,81,104,113,92,
        49,64,78,87,103,121,120,101,72,92,95,98,112,100,103,99,
    };
    std::memcpy(param.u8YQt, kYQt, sizeof(kYQt));

    static const AX_U8 kCbCrQt[64] = {
        17,18,24,47,99,99,99,99,18,21,26,66,99,99,99,99,
        24,26,56,99,99,99,99,99,47,66,99,99,99,99,99,99,
        99,99,99,99,99,99,99,99,99,99,99,99,99,99,99,99,
        99,99,99,99,99,99,99,99,99,99,99,99,99,99,99,99,
    };
    std::memcpy(param.u8CbCrQt, kCbCrQt, sizeof(kCbCrQt));

    param.u32Qfactor = jpeg.quality;
    param.bEnableRoi = AX_FALSE;
    param.bSaveNonRoiQt = AX_FALSE;
    param.u32RoiQfactor = 0;
    param.u32MCUPerECS = 0;
    param.bDblkEnable = AX_FALSE;
}

static void init_jpeg_chn_attr(const AxeraUtils::Jpeg& jpeg, AX_VENC_CHN_ATTR_T& attr)
{
    attr.stVencAttr.enType = PT_JPEG;
    attr.stVencAttr.u32MaxPicWidth = 3840;
    attr.stVencAttr.u32MaxPicHeight = 3840;
    attr.stVencAttr.enMemSource = AX_MEMORY_SOURCE_CMM;
    attr.stVencAttr.u32BufSize = attr.stVencAttr.u32MaxPicWidth * attr.stVencAttr.u32MaxPicHeight * 0.5;
    attr.stVencAttr.u32PicWidthSrc = jpeg.width;
    attr.stVencAttr.u32PicHeightSrc = jpeg.height;
    attr.stVencAttr.stCropCfg.bEnable = AX_FALSE;
    attr.stVencAttr.enRotation = AX_ROTATION_0;
    attr.stVencAttr.enLinkMode = AX_LINK_MODE;
    attr.stVencAttr.bDeBreathEffect = AX_FALSE;
    attr.stVencAttr.bRefRingbuf = AX_FALSE;
    attr.stVencAttr.s32StopWaitTime = -1;
    attr.stVencAttr.u8InFifoDepth = 1;
    attr.stVencAttr.u8OutFifoDepth = 1;

    attr.stRcAttr.s32FirstFrameStartQp = -1;
    attr.stRcAttr.stFrameRate.fSrcFrameRate = jpeg.fps;
    attr.stRcAttr.stFrameRate.fDstFrameRate = jpeg.fps;

    attr.stGopAttr.enGopMode = AX_VENC_GOPMODE_NORMALP;
}

bool AxeraUtils::jenc_init(void)
{
    AX_VENC_MOD_ATTR_T venc_mod_attr{};
    venc_mod_attr.enVencType = AX_VENC_MULTI_ENCODER;
    venc_mod_attr.stModThdAttr.u32TotalThreadNum = 1;
    venc_mod_attr.stModThdAttr.bExplicitSched = AX_FALSE;

    AX_U32 ret = AX_VENC_Init(&venc_mod_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_Init failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VENC_CHN_ATTR_T venc_jpeg_chn_attr{};
    init_jpeg_chn_attr(this->_jpeg, venc_jpeg_chn_attr);

    ret = AX_VENC_CreateChnEx(&this->_jpeg.chn, &venc_jpeg_chn_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_CreateChnEx (JPEG) failed, ret=0x{:X}", ret);
        return false;
    }

    this->_jpeg.fd = AX_VENC_GetFd(this->_jpeg.chn);
    if (this->_jpeg.fd < 0) {
        (*this->_logger)->error("Failed to get JPEG channel fd");
        return false;
    }

    AX_VENC_JPEG_PARAM_T jpeg_param{};
    init_jpeg_param(this->_jpeg, jpeg_param);
    ret = AX_VENC_SetJpegParam(this->_jpeg.chn, &jpeg_param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetJpegParam failed, ret=0x{:X}", this->_jpeg.chn, ret);
        return false;
    }

    return true;
}

bool AxeraUtils::jenc_deinit(void)
{
    AX_U32 ret;
    int retry = 5;

    do {
        ret = AX_VENC_DestroyChn(this->_jpeg.chn);
        if (ret == AX_ERR_VENC_BUSY) {
            (*this->_logger)->info("chn-%d: AX_VENC_DestroyChn (JPEG) return AX_ERR_VENC_BUSY,retry...", this->_jpeg.chn);
            -- retry;
            usleep(100 * 1000);
        } else {
            break;
        }
    } while (retry >= 0);

    retry = 5;

    ret = AX_VENC_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_Deinit failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::Jpeg::start(int chn)
{
    AX_VENC_RECV_PIC_PARAM_T recv_param{};
    recv_param.s32RecvPicNum = -1;
    AX_U32 ret = AX_VENC_StartRecvFrame(chn, &recv_param);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::Jpeg::stop(int chn)
{
    AX_U32 ret = AX_VENC_StopRecvFrame(chn);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::jenc_set_fps(int chn, float src_fps, float dst_fps)
{
    AX_VENC_RC_PARAM_T param{};
    AX_U32 ret = AX_VENC_GetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    if (src_fps > 0) {
        param.stFrameRate.fSrcFrameRate = src_fps;
    }

    if (dst_fps > 0) {
        if (dst_fps > param.stFrameRate.fSrcFrameRate) {
            dst_fps = param.stFrameRate.fSrcFrameRate;
        }
        param.stFrameRate.fDstFrameRate = dst_fps;
    } else {
        param.stFrameRate.fDstFrameRate = param.stFrameRate.fSrcFrameRate;
    }

    ret = AX_VENC_SetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    this->_jpeg.fps = static_cast<int>(param.stFrameRate.fDstFrameRate);

    (*this->_logger)->debug("[{}] fps set: src={} dst={}", chn,
                           param.stFrameRate.fSrcFrameRate,
                           param.stFrameRate.fDstFrameRate);
    return true;
}

bool AxeraUtils::jenc_set_quality(int chn, int quality)
{
    AX_VENC_JPEG_PARAM_T param{};
    AX_U32 ret = AX_VENC_GetJpegParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetJpegParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    param.u32Qfactor = quality;

    ret = AX_VENC_SetJpegParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetJpegParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    return true;
}

bool AxeraUtils::jenc_reset_chn(int chn)
{
    AX_U32 ret = AX_VENC_ResetChn(chn);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_ResetChn failed, ret=0x{:X}", chn, ret);
        return false;
    }

    if (!this->_jpeg.start(chn)) {
        (*this->_logger)->error("Failed to start JPEG channel after reset");
        return false;
    }

    if (!this->_jpeg.stop(chn)) {
        (*this->_logger)->error("Failed to stop JPEG channel after reset");
        return false;
    }

    return true;
}

int AxeraUtils::jenc_get_frame(std::vector<uint8_t>& data, int timeout_ms, int skip_frames)
{
    using Clock = std::chrono::steady_clock;

    data.clear();
    if (timeout_ms <= 0) timeout_ms = 500;
    if (skip_frames < 0) skip_frames = 0;

    const auto deadline = Clock::now() + std::chrono::milliseconds(timeout_ms);
    AX_VENC_STREAM_T stream{};
    bool started = false;
    bool stream_acquired = false;
    int result_code = 0;

    if (!this->_jpeg.start(this->_jpeg.chn)) {
        (*this->_logger)->error("[snapshot] failed to start JPEG channel");
        return -14;
    }
    started = true;

    try {
        for (int i = 0; i <= skip_frames; ++i) {
            const auto now = Clock::now();
            if (now >= deadline) {
                (*this->_logger)->error("[snapshot] JPEG frame deadline exceeded");
                result_code = -13;
                break;
            }

            const auto remaining = deadline - now;
            auto remaining_ms =
                std::chrono::duration_cast<std::chrono::milliseconds>(remaining);
            if (remaining_ms < remaining) {
                ++remaining_ms;
            }
            const auto remaining_count = std::min<int64_t>(
                remaining_ms.count(), std::numeric_limits<int>::max());

            const AX_U32 ret = AX_VENC_GetStream(
                this->_jpeg.chn, &stream,
                std::max(1, static_cast<int>(remaining_count)));
            if (ret != AX_SUCCESS) {
                if (ret == AX_ERR_VENC_TIMEOUT ||
                    ret == AX_ERR_VENC_QUEUE_EMPTY ||
                    ret == AX_ERR_VENC_BUF_EMPTY) {
                    (*this->_logger)->error(
                        "[snapshot] AX_VENC_GetStream timeout, ret=0x{:X}", ret);
                    result_code = -13;
                } else {
                    (*this->_logger)->error(
                        "[snapshot] AX_VENC_GetStream failed, ret=0x{:X}", ret);
                    result_code = -14;
                }
                break;
            }
            stream_acquired = true;

            if (i == skip_frames) {
                data.assign(stream.stPack.pu8Addr,
                            stream.stPack.pu8Addr + stream.stPack.u32Len);
            }

            const AX_U32 release_ret =
                AX_VENC_ReleaseStream(this->_jpeg.chn, &stream);
            if (release_ret != AX_SUCCESS) {
                (*this->_logger)->error(
                    "[snapshot] AX_VENC_ReleaseStream failed, ret=0x{:X}",
                    release_ret);
                result_code = -14;
                break;
            }
            stream_acquired = false;
        }
    } catch (const std::exception& e) {
        (*this->_logger)->error("[snapshot] failed to copy JPEG frame: {}", e.what());
        result_code = -14;
    } catch (...) {
        (*this->_logger)->error("[snapshot] failed to copy JPEG frame");
        result_code = -14;
    }

    if (stream_acquired) {
        const AX_U32 release_ret =
            AX_VENC_ReleaseStream(this->_jpeg.chn, &stream);
        if (release_ret != AX_SUCCESS) {
            (*this->_logger)->error(
                "[snapshot] cleanup AX_VENC_ReleaseStream failed, ret=0x{:X}",
                release_ret);
        }
    }

    if (started && !this->_jpeg.stop(this->_jpeg.chn)) {
        (*this->_logger)->error("Failed to stop JPEG channel");
        result_code = -14;
    }

    if (result_code != 0) {
        data.clear();
    }
    return result_code;
}

bool AxeraUtils::jenc_set_crop(bool enable, int x, int y, int w, int h)
{
    AX_VENC_CHN_ATTR_T attr{};
    AX_U32 ret = AX_VENC_GetChnAttr(this->_jpeg.chn, &attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetChnAttr ({}) failed, ret=0x{:X}", this->_jpeg.chn, "JPEG", ret);
        return false;
    }

    if (enable) {
        attr.stVencAttr.stCropCfg.bEnable = AX_TRUE;
        attr.stVencAttr.stCropCfg.stRect.s32X = x;
        attr.stVencAttr.stCropCfg.stRect.s32Y = y;
        attr.stVencAttr.stCropCfg.stRect.u32Width = w;
        attr.stVencAttr.stCropCfg.stRect.u32Height = h;
    } else {
        attr.stVencAttr.stCropCfg.bEnable = AX_FALSE;
        attr.stVencAttr.stCropCfg.stRect.s32X = 0;
        attr.stVencAttr.stCropCfg.stRect.s32Y = 0;
        attr.stVencAttr.stCropCfg.stRect.u32Width = 0;
        attr.stVencAttr.stCropCfg.stRect.u32Height = 0;
    }

    ret = AX_VENC_SetChnAttr(this->_jpeg.chn, &attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetChnAttr ({}) failed, ret=0x{:X}", this->_jpeg.chn, "JPEG", ret);
        return false;
    }

    return true;
}

int AxeraUtils::jenc_snapshot(int x, int y, int w, int h,
                              int quality,
                              int timeout_ms, SnapshotResult& result)
{
    using Clock = std::chrono::steady_clock;

    result.data.clear();
    result.width = 0;
    result.height = 0;

    /* -16 snapshot busy */
    std::unique_lock<std::mutex> lock(_snapshot_mutex, std::try_to_lock);
    if (!lock.owns_lock()) {
        (*this->_logger)->warn("[snapshot] busy");
        return -16;
    }

    /* -11 parameter validation */
    if (quality < AX_MIN_JENC_QFACTOR) quality = AX_MIN_JENC_QFACTOR;
    else if (quality > AX_MAX_JENC_QFACTOR) quality = AX_MAX_JENC_QFACTOR;

    if (timeout_ms <= 0) timeout_ms = 500;
    const auto deadline = Clock::now() + std::chrono::milliseconds(timeout_ms);

    /* default to full frame */
    if (x == 0 && y == 0 && w == 0 && h == 0) {
        w = _width;
        h = _height;
    }

    if (x < 0 || y < 0 || w <= 0 || h <= 0) {
        (*this->_logger)->error("[snapshot] invalid params: ({},{},{},{})", x, y, w, h);
        return -11;
    }

    /* -12 clamp to source bounds */
    if (x >= _width || y >= _height) {
        (*this->_logger)->error("[snapshot] ROI out of bounds");
        return -12;
    }

    if (x + w > _width)  w = _width - x;
    if (y + h > _height) h = _height - y;

    if (w <= 0 || h <= 0) {
        (*this->_logger)->error("[snapshot] ROI out of bounds after clamp: ({},{},{},{})", x, y, w, h);
        return -12;
    }

    int crop_w = w;
    int crop_h = h;
    int out_w = crop_w;
    int out_h = crop_h;
    out_w = (out_w + 1) & ~1;   /* align to 2 */
    out_h = (out_h + 1) & ~1;
    if (out_w < 2 || out_h < 2) {
        (*this->_logger)->error("[snapshot] output too small: {}x{}", out_w, out_h);
        return -11;
    }

    int ret_code = 0;

    const bool need_crop = !(x == 0 && y == 0 && w == _width && h == _height);
    if (!jenc_set_crop(need_crop, need_crop ? x : 0, need_crop ? y : 0, need_crop ? w : 0, need_crop ? h : 0)) {
        (*this->_logger)->error("[snapshot] jenc_set_crop failed");
        ret_code = -15;
    } else if (!jenc_set_quality(this->_jpeg.chn, quality)) {
        (*this->_logger)->error("Failed to set JPEG quality");
        ret_code = -15;
    } else {
        const auto now = Clock::now();
        if (now >= deadline) {
            ret_code = -13;
        } else {
            const auto remaining = deadline - now;
            auto remaining_ms =
                std::chrono::duration_cast<std::chrono::milliseconds>(remaining);
            if (remaining_ms < remaining) {
                ++remaining_ms;
            }
            const auto remaining_count = std::min<int64_t>(
                remaining_ms.count(), std::numeric_limits<int>::max());
            ret_code = jenc_get_frame(
                result.data,
                std::max(1, static_cast<int>(remaining_count)),
                1);
        }

        if (ret_code != 0) {
            (*this->_logger)->error(
                "[snapshot] failed to get JPEG frame, ret={}", ret_code);
        } else {
            result.width = out_w;
            result.height = out_h;
        }
    }

    if (!jenc_reset_chn(this->_jpeg.chn)) {
        (*this->_logger)->error("[snapshot] failed to reset JPEG channel");
        ret_code = -19;
    }

    if (ret_code != 0) {
        result.data.clear();
        result.width = 0;
        result.height = 0;
    }
    return ret_code;
}
