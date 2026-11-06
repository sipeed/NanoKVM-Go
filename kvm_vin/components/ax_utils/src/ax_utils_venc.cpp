#include <algorithm>
#include <cstdint>
#include <unistd.h>
#include <cstring>
#include <limits>

#include <ax_ivps_api.h>
#include <ax_sys_api.h>
#include <ax_venc_api.h>
#include <ax_venc_comm.h>

#include "ax_utils.hpp"

namespace {

constexpr int TIMELAPSE_ENCODE_OK = 0;
constexpr int TIMELAPSE_ENCODE_ERR_INPUT = -30;
constexpr int TIMELAPSE_ENCODE_ERR_RESOLUTION = -31;
constexpr int TIMELAPSE_ENCODE_ERR_ALLOC = -32;
constexpr int TIMELAPSE_ENCODE_ERR_SEND = -33;
constexpr int TIMELAPSE_ENCODE_ERR_STREAM = -34;
constexpr int TIMELAPSE_ENCODE_ERR_RELEASE = -35;
constexpr uint32_t H264_DEFAULT_BITRATE_KBPS = 1000;
constexpr uint32_t TIMELAPSE_H264_QP_MIN = 36;
constexpr uint32_t TIMELAPSE_H264_QP_MAX = 51;

size_t nv12_frame_size(int y_stride, int uv_stride, int height)
{
    return static_cast<size_t>(y_stride) * static_cast<size_t>(height) +
           static_cast<size_t>(uv_stride) * static_cast<size_t>(height / 2);
}

}  // namespace

static void init_h264_rc_defaults(AxeraUtils::H264& h264, uint32_t bitrate_kbps)
{
    h264.cbr.u32Gop = 1;
    h264.cbr.u32BitRate = bitrate_kbps;
    h264.cbr.u32MinQp = 10;
    h264.cbr.u32MaxQp = 51;
    h264.cbr.u32MinIQp = 10;
    h264.cbr.u32MaxIQp = 51;
    h264.cbr.u32MinIprop = 10;
    h264.cbr.u32MaxIprop = 40;

    h264.vbr.u32Gop = 1;
    h264.vbr.u32MaxBitRate = bitrate_kbps;
    h264.vbr.u32MinQp = 31;
    h264.vbr.u32MaxQp = 46;
    h264.vbr.u32MinIQp = 31;
    h264.vbr.u32MaxIQp = 46;
    h264.vbr.enVQ = AX_VENC_VBR_QUALITY_LEVEL_INV;
}

static void init_h265_rc_defaults(AxeraUtils::H265& h265)
{
    h265.cbr.u32Gop = 1;
    h265.cbr.u32BitRate = 1000;
    h265.cbr.u32MinQp = 10;
    h265.cbr.u32MaxQp = 51;
    h265.cbr.u32MinIQp = 10;
    h265.cbr.u32MaxIQp = 51;
    h265.cbr.u32MinIprop = 30;
    h265.cbr.u32MaxIprop = 40;

    h265.vbr.u32Gop = 1;
    h265.vbr.u32MaxBitRate = 1000;
    h265.vbr.u32MinQp = 31;
    h265.vbr.u32MaxQp = 46;
    h265.vbr.u32MinIQp = 31;
    h265.vbr.u32MaxIQp = 46;
    h265.vbr.enVQ = AX_VENC_VBR_QUALITY_LEVEL_INV;
}

static void init_h264_chn_attr(AxeraUtils::H264& h264,
                               AX_VENC_CHN_ATTR_T& attr,
                               uint32_t bitrate_kbps = H264_DEFAULT_BITRATE_KBPS)
{
    init_h264_rc_defaults(h264, bitrate_kbps);

    attr.stVencAttr.enType = PT_H264;
    attr.stVencAttr.u32MaxPicWidth = 3840;
    attr.stVencAttr.u32MaxPicHeight = 2160;
    attr.stVencAttr.enMemSource = AX_MEMORY_SOURCE_CMM;
    attr.stVencAttr.u32BufSize = attr.stVencAttr.u32MaxPicWidth * attr.stVencAttr.u32MaxPicHeight * 0.5;
    attr.stVencAttr.enProfile = AX_VENC_H264_BASE_PROFILE;
    if (h264.width * h264.height <= 1920 * 1080 && h264.fps <= 60) {
        attr.stVencAttr.enLevel = AX_VENC_H264_LEVEL_4_2;
    } else {
        attr.stVencAttr.enLevel = AX_VENC_H264_LEVEL_5_1;
    }
    attr.stVencAttr.u32PicWidthSrc = h264.width;
    attr.stVencAttr.u32PicHeightSrc = h264.height;
    attr.stVencAttr.stCropCfg.bEnable = AX_FALSE;
    attr.stVencAttr.enRotation = AX_ROTATION_0;
    attr.stVencAttr.enLinkMode = AX_LINK_MODE;
    attr.stVencAttr.bDeBreathEffect = AX_FALSE;
    attr.stVencAttr.bRefRingbuf = AX_TRUE;
    attr.stVencAttr.s32StopWaitTime = -1;
    attr.stVencAttr.u8InFifoDepth = 1;
    attr.stVencAttr.u8OutFifoDepth = 2;

    attr.stRcAttr.enRcMode = AX_VENC_RC_MODE_H264CBR;
    attr.stRcAttr.s32FirstFrameStartQp = -1;
    attr.stRcAttr.stFrameRate.fSrcFrameRate = h264.fps;
    attr.stRcAttr.stFrameRate.fDstFrameRate = h264.fps;
    memcpy(&attr.stRcAttr.stH264Cbr, &h264.cbr, sizeof(AX_VENC_H264_CBR_T));

    attr.stGopAttr.enGopMode = AX_VENC_GOPMODE_NORMALP;
}

static void init_timelapse_h264_chn_attr(AxeraUtils::H264& h264,
                                         AX_VENC_CHN_ATTR_T& attr,
                                         uint32_t qp)
{
    init_h264_chn_attr(h264, attr);
    attr.stVencAttr.enLinkMode = AX_UNLINK_MODE;
    attr.stVencAttr.bRefRingbuf = AX_TRUE;
    attr.stVencAttr.u8InFifoDepth = 1;
    attr.stRcAttr.enRcMode = AX_VENC_RC_MODE_H264FIXQP;
    attr.stRcAttr.stH264FixQp.u32Gop = 1;
    attr.stRcAttr.stH264FixQp.u32IQp = qp;
    attr.stRcAttr.stH264FixQp.u32PQp = qp;
    attr.stRcAttr.stH264FixQp.u32BQp = qp;
}

static void init_h265_chn_attr(AxeraUtils::H265& h265, AX_VENC_CHN_ATTR_T& attr)
{
    init_h265_rc_defaults(h265);

    attr.stVencAttr.enType = PT_H265;
    attr.stVencAttr.u32MaxPicWidth = 3840;
    attr.stVencAttr.u32MaxPicHeight = 2160;
    attr.stVencAttr.enMemSource = AX_MEMORY_SOURCE_CMM;
    attr.stVencAttr.u32BufSize = attr.stVencAttr.u32MaxPicWidth * attr.stVencAttr.u32MaxPicHeight * 0.5;
    attr.stVencAttr.enProfile = AX_VENC_HEVC_MAIN_PROFILE;
    if (h265.width * h265.height <= 1920 * 1080 && h265.fps <= 60) {
        attr.stVencAttr.enLevel = AX_VENC_HEVC_LEVEL_4_1;
    } else {
        attr.stVencAttr.enLevel = AX_VENC_HEVC_LEVEL_5;
    }
    attr.stVencAttr.enTier = AX_VENC_HEVC_MAIN_TIER;
    attr.stVencAttr.u32PicWidthSrc = h265.width;
    attr.stVencAttr.u32PicHeightSrc = h265.height;
    attr.stVencAttr.stCropCfg.bEnable = AX_FALSE;
    attr.stVencAttr.enRotation = AX_ROTATION_0;
    attr.stVencAttr.enLinkMode = AX_LINK_MODE;
    attr.stVencAttr.bDeBreathEffect = AX_FALSE;
    attr.stVencAttr.bRefRingbuf = AX_TRUE;
    attr.stVencAttr.s32StopWaitTime = -1;
    attr.stVencAttr.u8InFifoDepth = 1;
    attr.stVencAttr.u8OutFifoDepth = 2;

    attr.stRcAttr.enRcMode = AX_VENC_RC_MODE_H265CBR;
    attr.stRcAttr.s32FirstFrameStartQp = -1;
    attr.stRcAttr.stFrameRate.fSrcFrameRate = h265.fps;
    attr.stRcAttr.stFrameRate.fDstFrameRate = h265.fps;
    memcpy(&attr.stRcAttr.stH265Cbr, &h265.cbr, sizeof(AX_VENC_H265_CBR_T));

    attr.stGopAttr.enGopMode = AX_VENC_GOPMODE_NORMALP;
}

void AxeraUtils::set_encode_mode(EncodeMode mode)
{
    _encode_mode = mode;
    _active_venc = (mode == EncodeMode::H265) ? static_cast<Venc*>(&_h265)
                                              : static_cast<Venc*>(&_h264);
}

bool AxeraUtils::venc_init(void)
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

    if (_encode_mode == EncodeMode::H265) {
        AX_VENC_CHN_ATTR_T venc_h265_chn_attr{};
        init_h265_chn_attr(this->_h265, venc_h265_chn_attr);

        ret = AX_VENC_CreateChnEx(&this->_h265.chn, &venc_h265_chn_attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("AX_VENC_CreateChnEx (H265) failed, ret=0x{:X}", ret);
            return false;
        }

        this->_h265.fd = AX_VENC_GetFd(this->_h265.chn);
        if (this->_h265.fd < 0) {
            (*this->_logger)->error("Failed to get H265 channel fd");
            return false;
        }
    } else {
        AX_VENC_CHN_ATTR_T venc_h264_chn_attr{};
        init_h264_chn_attr(this->_h264, venc_h264_chn_attr);

        ret = AX_VENC_CreateChnEx(&this->_h264.chn, &venc_h264_chn_attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("AX_VENC_CreateChnEx (H264) failed, ret=0x{:X}", ret);
            return false;
        }

        this->_h264.fd = AX_VENC_GetFd(this->_h264.chn);
        if (this->_h264.fd < 0) {
            (*this->_logger)->error("Failed to get H264 channel fd");
            return false;
        }
    }

    return true;
}

bool AxeraUtils::venc_deinit(void)
{
    AX_U32 ret;
    int retry = 5;
    const int active_chn = _active_venc ? _active_venc->chn : _h264.chn;
    const char* chn_name = (_encode_mode == EncodeMode::H265) ? "H265" : "H264";

    do {
        ret = AX_VENC_DestroyChn(active_chn);
        if (ret == AX_ERR_VENC_BUSY) {
            (*this->_logger)->info("chn-%d: AX_VENC_DestroyChn (%s) return AX_ERR_VENC_BUSY,retry...", active_chn, chn_name);
            -- retry;
            usleep(100 * 1000);
        } else {
            break;
        }
    } while (retry >= 0);

    ret = AX_VENC_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_Deinit failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::timelapse_venc_init(unsigned int qp)
{
    if (qp < TIMELAPSE_H264_QP_MIN || qp > TIMELAPSE_H264_QP_MAX) {
        (*this->_logger)->error("[timelapse] H264 FIXQP must be from 36 to 51: {}", qp);
        return false;
    }

    _timelapse_h264.width = _width;
    _timelapse_h264.height = _height;
    _timelapse_h264.fps = _fps;

    AX_VENC_MOD_ATTR_T venc_mod_attr{};
    venc_mod_attr.enVencType = AX_VENC_MULTI_ENCODER;
    venc_mod_attr.stModThdAttr.u32TotalThreadNum = 1;
    venc_mod_attr.stModThdAttr.bExplicitSched = AX_FALSE;

    AX_U32 ret = AX_VENC_Init(&venc_mod_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_Init failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VENC_CHN_ATTR_T timelapse_h264_chn_attr{};
    init_timelapse_h264_chn_attr(
        this->_timelapse_h264, timelapse_h264_chn_attr, qp);

    ret = AX_VENC_CreateChnEx(&this->_timelapse_h264.chn, &timelapse_h264_chn_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_CreateChnEx (H264) failed, ret=0x{:X}", ret);
        (void)AX_VENC_Deinit();
        return false;
    }

    this->_timelapse_h264.fd = AX_VENC_GetFd(this->_timelapse_h264.chn);
    if (this->_timelapse_h264.fd < 0) {
        (*this->_logger)->error("[timelapse] Failed to get H264 channel fd");
        (void)AX_VENC_DestroyChn(this->_timelapse_h264.chn);
        (void)AX_VENC_Deinit();
        return false;
    }

    if (!this->_timelapse_h264.start(this->_timelapse_h264.chn)) {
        (*this->_logger)->error("[timelapse] AX_VENC_StartRecvFrame (H264) failed");
        (void)AX_VENC_DestroyChn(this->_timelapse_h264.chn);
        (void)AX_VENC_Deinit();
        return false;
    }

    (*this->_logger)->info("[timelapse] H264 FIXQP encoder initialized: chn={} {}x{}@{} qp={}",
                           this->_timelapse_h264.chn,
                           this->_timelapse_h264.width,
                           this->_timelapse_h264.height,
                           this->_timelapse_h264.fps,
                           qp);
    return true;
}

bool AxeraUtils::timelapse_venc_deinit(void)
{
    AX_U32 ret;
    int retry = 5;

    if (!this->_timelapse_h264.stop(this->_timelapse_h264.chn)) {
        (*this->_logger)->error("[timelapse] AX_VENC_StopRecvFrame (H264) failed");
    }

    do {
        ret = AX_VENC_DestroyChn(this->_timelapse_h264.chn);
        if (ret == AX_ERR_VENC_BUSY) {
            (*this->_logger)->info("chn-%d: AX_VENC_DestroyChn (timelapse H264) return AX_ERR_VENC_BUSY,retry...",
                                   this->_timelapse_h264.chn);
            -- retry;
            usleep(100 * 1000);
        } else {
            break;
        }
    } while (retry >= 0);

    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_DestroyChn (H264) failed, ret=0x{:X}", ret);
        return false;
    }

    this->_timelapse_h264.fd = -1;

    ret = AX_VENC_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_Deinit failed, ret=0x{:X}", ret);
        return false;
    }

    (*this->_logger)->info("[timelapse] H264 encoder deinitialized");
    return true;
}

int AxeraUtils::timelapse_encode_nv12(const uint8_t* nv12,
                                      size_t size,
                                      int width,
                                      int height,
                                      int y_stride,
                                      int uv_stride,
                                      uint64_t pts,
                                      int timeout_ms,
                                      std::vector<uint8_t>& h264)
{
    h264.clear();

    if (nv12 == nullptr || width <= 0 || height <= 0 ||
        y_stride < width || uv_stride < width ||
        (width % 2) != 0 || (height % 2) != 0) {
        (*this->_logger)->error("[timelapse] invalid NV12 input: {}x{} stride={} / {}",
                                width, height, y_stride, uv_stride);
        return TIMELAPSE_ENCODE_ERR_INPUT;
    }

    const size_t required_size = nv12_frame_size(y_stride, uv_stride, height);
    if (required_size == 0 || size < required_size ||
        required_size > std::numeric_limits<AX_U32>::max()) {
        (*this->_logger)->error("[timelapse] invalid NV12 buffer size: have={} need={}",
                                size, required_size);
        return TIMELAPSE_ENCODE_ERR_INPUT;
    }

    if (width > 3840 || height > 3840) {
        (*this->_logger)->error("[timelapse] unsupported encoder resolution: {}x{}",
                                width, height);
        return TIMELAPSE_ENCODE_ERR_RESOLUTION;
    }

    if (_timelapse_h264.width != width || _timelapse_h264.height != height) {
        if (!this->_timelapse_h264.stop(this->_timelapse_h264.chn)) {
            (*this->_logger)->error("[timelapse] failed to stop H264 channel before resolution change");
            return TIMELAPSE_ENCODE_ERR_RESOLUTION;
        }

        AX_U32 ret = AX_VENC_ResetChn(this->_timelapse_h264.chn);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[timelapse] AX_VENC_ResetChn failed, ret=0x{:X}", ret);
            return TIMELAPSE_ENCODE_ERR_RESOLUTION;
        }

        AX_VENC_CHN_ATTR_T attr {};
        ret = AX_VENC_GetChnAttr(this->_timelapse_h264.chn, &attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[timelapse] AX_VENC_GetChnAttr failed, ret=0x{:X}", ret);
            return TIMELAPSE_ENCODE_ERR_RESOLUTION;
        }

        attr.stVencAttr.u32PicWidthSrc = static_cast<AX_U32>(width);
        attr.stVencAttr.u32PicHeightSrc = static_cast<AX_U32>(height);
        attr.stVencAttr.u32MaxPicWidth =
            std::max(attr.stVencAttr.u32MaxPicWidth, static_cast<AX_U32>(width));
        attr.stVencAttr.u32MaxPicHeight =
            std::max(attr.stVencAttr.u32MaxPicHeight, static_cast<AX_U32>(height));
        attr.stVencAttr.u32BufSize =
            attr.stVencAttr.u32MaxPicWidth * attr.stVencAttr.u32MaxPicHeight / 2;

        ret = AX_VENC_SetChnAttr(this->_timelapse_h264.chn, &attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[timelapse] AX_VENC_SetChnAttr failed, ret=0x{:X}", ret);
            return TIMELAPSE_ENCODE_ERR_RESOLUTION;
        }

        this->_timelapse_h264.width = width;
        this->_timelapse_h264.height = height;

        if (!this->_timelapse_h264.start(this->_timelapse_h264.chn)) {
            (*this->_logger)->error("[timelapse] failed to restart H264 channel after resolution change");
            return TIMELAPSE_ENCODE_ERR_RESOLUTION;
        }

        (*this->_logger)->info("[timelapse] H264 encoder resolution set to {}x{}",
                               width, height);
    }

    AX_BLK frame_blk = AX_POOL_GetBlock(AX_INVALID_POOLID,
                                        static_cast<AX_U32>(required_size),
                                        nullptr);
    if (frame_blk == AX_INVALID_BLOCKID) {
        (*this->_logger)->error("[timelapse] AX_POOL_GetBlock(NV12) failed, size={}",
                                required_size);
        return TIMELAPSE_ENCODE_ERR_ALLOC;
    }

    const AX_U64 frame_phy = AX_POOL_Handle2PhysAddr(frame_blk);
    AX_VOID* frame_vir = AX_POOL_GetBlockVirAddr(frame_blk);
    if (frame_phy == 0 || frame_vir == nullptr) {
        (*this->_logger)->error("[timelapse] failed to get AX pool block address, blk={}",
                                frame_blk);
        AX_POOL_ReleaseBlock(frame_blk);
        return TIMELAPSE_ENCODE_ERR_ALLOC;
    }

    auto* frame_vir_u8 = static_cast<uint8_t*>(frame_vir);
    std::memset(frame_vir_u8, 0, required_size);
    std::memcpy(frame_vir_u8, nv12, required_size);
    AX_S32 ret = AX_SYS_MflushCache(frame_phy, frame_vir, static_cast<AX_U32>(required_size));
    if (ret != AX_SUCCESS) {
        (*this->_logger)->warn("[timelapse] AX_SYS_MflushCache(NV12) failed, ret=0x{:X}",
                               static_cast<AX_U32>(ret));
    }

    AX_VIDEO_FRAME_INFO_T frame_info {};
    frame_info.enModId = AX_ID_USER;
    frame_info.bEndOfStream = AX_FALSE;
    frame_info.stVFrame.u32Width = static_cast<AX_U32>(width);
    frame_info.stVFrame.u32Height = static_cast<AX_U32>(height);
    frame_info.stVFrame.enImgFormat = AX_FORMAT_YUV420_SEMIPLANAR;
    frame_info.stVFrame.enVscanFormat = AX_VSCAN_FORMAT_RASTER;
    frame_info.stVFrame.stCompressInfo.enCompressMode = AX_COMPRESS_MODE_NONE;
    frame_info.stVFrame.stCompressInfo.u32CompressLevel = 0;
    frame_info.stVFrame.stDynamicRange = AX_DYNAMIC_RANGE_SDR8;
    frame_info.stVFrame.stColorGamut = AX_COLOR_GAMUT_BT709;
    frame_info.stVFrame.u32PicStride[0] = static_cast<AX_U32>(y_stride);
    frame_info.stVFrame.u32PicStride[1] = static_cast<AX_U32>(uv_stride);
    frame_info.stVFrame.u64PhyAddr[0] = frame_phy;
    frame_info.stVFrame.u64VirAddr[0] = reinterpret_cast<AX_U64>(frame_vir_u8);
    frame_info.stVFrame.u64PhyAddr[1] =
        frame_phy + static_cast<AX_U64>(y_stride) * static_cast<AX_U64>(height);
    frame_info.stVFrame.u64VirAddr[1] =
        reinterpret_cast<AX_U64>(frame_vir_u8) +
        static_cast<AX_U64>(y_stride) * static_cast<AX_U64>(height);
    frame_info.stVFrame.u32BlkId[0] = frame_blk;
    frame_info.stVFrame.u32BlkId[1] = 0;
    frame_info.stVFrame.u32BlkId[2] = 0;
    frame_info.stVFrame.u64PTS = pts;
    frame_info.stVFrame.u64SeqNum = pts;
    frame_info.stVFrame.u32FrameSize = static_cast<AX_U32>(required_size);

    ret = AX_VENC_SendFrame(this->_timelapse_h264.chn, &frame_info, timeout_ms);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_SendFrame failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        AX_POOL_ReleaseBlock(frame_blk);
        return TIMELAPSE_ENCODE_ERR_SEND;
    }

    AX_VENC_STREAM_T stream {};
    ret = AX_VENC_GetStream(this->_timelapse_h264.chn, &stream, timeout_ms);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_GetStream failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        AX_POOL_ReleaseBlock(frame_blk);
        return TIMELAPSE_ENCODE_ERR_STREAM;
    }

    if (stream.stPack.pu8Addr != nullptr && stream.stPack.u32Len > 0) {
        h264.assign(stream.stPack.pu8Addr,
                    stream.stPack.pu8Addr + stream.stPack.u32Len);
    }

    ret = AX_VENC_ReleaseStream(this->_timelapse_h264.chn, &stream);
    AX_POOL_ReleaseBlock(frame_blk);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[timelapse] AX_VENC_ReleaseStream failed, ret=0x{:X}",
                                static_cast<AX_U32>(ret));
        h264.clear();
        return TIMELAPSE_ENCODE_ERR_RELEASE;
    }

    if (h264.empty()) {
        (*this->_logger)->error("[timelapse] H264 encoder returned empty stream");
        return TIMELAPSE_ENCODE_ERR_STREAM;
    }

    (*this->_logger)->debug("[timelapse] encoded one H264 frame: {} bytes", h264.size());
    return TIMELAPSE_ENCODE_OK;
}

bool AxeraUtils::H264::start(int chn)
{
    AX_VENC_RECV_PIC_PARAM_T recv_param{};
    recv_param.s32RecvPicNum = -1;
    AX_U32 ret = AX_VENC_StartRecvFrame(chn, &recv_param);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::H264::stop(int chn)
{
    AX_U32 ret = AX_VENC_StopRecvFrame(chn);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::H265::start(int chn)
{
    AX_VENC_RECV_PIC_PARAM_T recv_param{};
    recv_param.s32RecvPicNum = -1;
    AX_U32 ret = AX_VENC_StartRecvFrame(chn, &recv_param);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::H265::stop(int chn)
{
    AX_U32 ret = AX_VENC_StopRecvFrame(chn);
    return (ret != AX_SUCCESS) ? false : true;
}

bool AxeraUtils::venc_start_recv()
{
    std::lock_guard<std::mutex> lock(_main_venc_mutex);
    if (_main_venc_reconfiguring) {
        return false;
    }
    if (_main_venc_recv_started) {
        return true;
    }
    if (!_active_venc || !_active_venc->start(_active_venc->chn)) {
        return false;
    }

    _main_venc_recv_started = true;
    return true;
}

bool AxeraUtils::venc_stop_recv()
{
    std::lock_guard<std::mutex> lock(_main_venc_mutex);
    if (_main_venc_reconfiguring) {
        return false;
    }
    if (!_main_venc_recv_started) {
        return true;
    }
    if (!_active_venc || !_active_venc->stop(_active_venc->chn)) {
        return false;
    }

    _main_venc_recv_started = false;
    return true;
}

bool AxeraUtils::main_venc_reconfiguring() const
{
    std::lock_guard<std::mutex> lock(_main_venc_mutex);
    return _main_venc_reconfiguring;
}

bool AxeraUtils::venc_set_resolution(int chn, int width, int height)
{
    auto set_chn_resolution = [this](int chn, int w, int h, const char* name) -> bool {
        AX_U32 ret = AX_VENC_ResetChn(chn);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[{}] AX_VENC_ResetChn ({}) failed, ret=0x{:X}", chn, name, ret);
            return false;
        }

        AX_VENC_CHN_ATTR_T attr{};
        ret = AX_VENC_GetChnAttr(chn, &attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[{}] AX_VENC_GetChnAttr ({}) failed, ret=0x{:X}", chn, name, ret);
            return false;
        }

        attr.stVencAttr.u32PicWidthSrc = w;
        attr.stVencAttr.u32PicHeightSrc = h;

        ret = AX_VENC_SetChnAttr(chn, &attr);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("[{}] AX_VENC_SetChnAttr ({}) failed, ret=0x{:X}", chn, name, ret);
            return false;
        }

        (*this->_logger)->debug("[{}] {} resolution set to {}x{}", chn, name, w, h);
        return true;
    };

    const char* venc_name = (_encode_mode == EncodeMode::H265) ? "H265" : "H264";
    bool ok = set_chn_resolution(chn, width, height, venc_name);
    if (ok) {
        _active_venc->width = width;
        _active_venc->height = height;
    }
    return ok;
}

bool AxeraUtils::venc_set_fps(int chn, float src_fps, float dst_fps)
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

    if (_active_venc) {
        _active_venc->fps = static_cast<int>(param.stFrameRate.fDstFrameRate);
    }

    (*this->_logger)->debug("[{}] fps set: src={} dst={}", chn,
                           param.stFrameRate.fSrcFrameRate,
                           param.stFrameRate.fDstFrameRate);
    return true;
}

bool AxeraUtils::venc_set_rc_mode(int chn, AX_VENC_RC_MODE_E mode)
{
    AX_VENC_RC_PARAM_T param{};
    AX_U32 ret = AX_VENC_GetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    if (param.enRcMode == mode) {
        (*this->_logger)->debug("[{}] RC mode already {}", chn, static_cast<int>(mode));
        return true;
    }

    const bool is_h265 = (_encode_mode == EncodeMode::H265);

    if (is_h265) {
        if (mode != AX_VENC_RC_MODE_H265CBR && mode != AX_VENC_RC_MODE_H265VBR) {
            (*this->_logger)->error("[{}] Unsupported RC mode for H265: {}", chn, static_cast<int>(mode));
            return false;
        }
        param.enRcMode = mode;
        if (mode == AX_VENC_RC_MODE_H265CBR) {
            this->_h265.cbr.u32Gop = param.stH265Vbr.u32Gop;
            this->_h265.cbr.u32BitRate = param.stH265Vbr.u32MaxBitRate;
            memcpy(&param.stH265Cbr, &this->_h265.cbr, sizeof(AX_VENC_H265_CBR_T));
        } else {
            this->_h265.vbr.u32Gop = param.stH265Cbr.u32Gop;
            this->_h265.vbr.u32MaxBitRate = param.stH265Cbr.u32BitRate;
            memcpy(&param.stH265Vbr, &this->_h265.vbr, sizeof(AX_VENC_H265_VBR_T));
        }
    } else {
        if (mode != AX_VENC_RC_MODE_H264CBR && mode != AX_VENC_RC_MODE_H264VBR) {
            (*this->_logger)->error("[{}] Unsupported RC mode for H264: {}", chn, static_cast<int>(mode));
            return false;
        }
        param.enRcMode = mode;
        if (mode == AX_VENC_RC_MODE_H264CBR) {
            this->_h264.cbr.u32Gop = param.stH264Vbr.u32Gop;
            this->_h264.cbr.u32BitRate = param.stH264Vbr.u32MaxBitRate;
            memcpy(&param.stH264Cbr, &this->_h264.cbr, sizeof(AX_VENC_H264_CBR_T));
        } else {
            this->_h264.vbr.u32Gop = param.stH264Cbr.u32Gop;
            this->_h264.vbr.u32MaxBitRate = param.stH264Cbr.u32BitRate;
            memcpy(&param.stH264Vbr, &this->_h264.vbr, sizeof(AX_VENC_H264_VBR_T));
        }
    }

    ret = AX_VENC_SetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    (*this->_logger)->debug("[{}] RC mode set to {}", chn, static_cast<int>(mode));
    return true;
}

bool AxeraUtils::venc_set_bitrate(int chn, unsigned int bitrate)
{
    AX_VENC_RC_PARAM_T param{};
    AX_U32 ret = AX_VENC_GetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    if (param.enRcMode == AX_VENC_RC_MODE_H264CBR) {
        param.stH264Cbr.u32BitRate = bitrate;
        this->_h264.cbr.u32BitRate = bitrate;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H264VBR) {
        param.stH264Vbr.u32MaxBitRate = bitrate;
        this->_h264.vbr.u32MaxBitRate = bitrate;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H265CBR) {
        param.stH265Cbr.u32BitRate = bitrate;
        this->_h265.cbr.u32BitRate = bitrate;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H265VBR) {
        param.stH265Vbr.u32MaxBitRate = bitrate;
        this->_h265.vbr.u32MaxBitRate = bitrate;
    } else {
        (*this->_logger)->error("[{}] Unsupported RC mode for set_bitrate: {}", chn, static_cast<int>(param.enRcMode));
        return false;
    }

    ret = AX_VENC_SetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    (*this->_logger)->debug("[{}] bitrate set to {}", chn, bitrate);
    return true;
}

bool AxeraUtils::venc_set_gop(int chn, unsigned int gop)
{
    AX_VENC_RC_PARAM_T param{};
    AX_U32 ret = AX_VENC_GetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_GetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    if (param.enRcMode == AX_VENC_RC_MODE_H264CBR) {
        param.stH264Cbr.u32Gop = gop;
        this->_h264.cbr.u32Gop = gop;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H264VBR) {
        param.stH264Vbr.u32Gop = gop;
        this->_h264.vbr.u32Gop = gop;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H265CBR) {
        param.stH265Cbr.u32Gop = gop;
        this->_h265.cbr.u32Gop = gop;
    } else if (param.enRcMode == AX_VENC_RC_MODE_H265VBR) {
        param.stH265Vbr.u32Gop = gop;
        this->_h265.vbr.u32Gop = gop;
    } else {
        (*this->_logger)->error("[{}] Unsupported RC mode for set_gop: {}", chn, static_cast<int>(param.enRcMode));
        return false;
    }

    ret = AX_VENC_SetRcParam(chn, &param);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_SetRcParam failed, ret=0x{:X}", chn, ret);
        return false;
    }

    (*this->_logger)->debug("[{}] gop set to {}", chn, gop);
    return true;
}

bool AxeraUtils::reconfigure_vin_pipeline(int width, int height, int fps)
{
    if (width <= 0 || height <= 0 || fps <= 0) {
        (*this->_logger)->error("invalid VIN pipeline reconfigure target: {}x{}@{}",
                                width, height, fps);
        return false;
    }

    std::unique_lock<std::mutex> pipeline_lock(
        _vin_pipeline_mutex, std::try_to_lock);
    if (!pipeline_lock.owns_lock()) {
        (*this->_logger)->error("VIN pipeline reconfigure already in progress");
        return false;
    }

    const bool use_main_venc = _venc_enabled;
    if (use_main_venc && !_active_venc) {
        (*this->_logger)->error("main VENC reconfigure requested without active VENC");
        return false;
    }

    const int old_width = _width;
    const int old_height = _height;
    const int old_fps = _fps;
    const int chn = use_main_venc ? get_venc_chn() : -1;
    const int old_venc_width = use_main_venc ? _active_venc->width : 0;
    const int old_venc_height = use_main_venc ? _active_venc->height : 0;
    const int old_venc_fps = use_main_venc ? _active_venc->fps : 0;

    bool main_venc_window_open = false;
    bool was_started = false;
    if (use_main_venc) {
        std::unique_lock<std::mutex> main_venc_lock(_main_venc_mutex);
        if (_main_venc_reconfiguring) {
            (*this->_logger)->error(
                "main VENC reconfigure window is already active");
            return false;
        }

        _main_venc_reconfiguring = true;
        main_venc_window_open = true;
        _main_venc_cv.wait(main_venc_lock, [this]() {
            return _main_venc_stream_users == 0;
        });
        was_started = _main_venc_recv_started;
    }

    bool recv_stopped = false;
    bool link_stopped = false;
    bool vin_stopped = false;
    bool venc_resolution_updated = false;
    bool venc_fps_updated = false;
    bool internal_state_updated = false;

    auto clear_main_venc_window = [this, &main_venc_window_open]() {
        if (!main_venc_window_open) {
            return;
        }
        {
            std::lock_guard<std::mutex> guard(_main_venc_mutex);
            _main_venc_reconfiguring = false;
            main_venc_window_open = false;
        }
        _main_venc_cv.notify_all();
    };

    auto set_recv_started = [this](bool started) {
        std::lock_guard<std::mutex> guard(_main_venc_mutex);
        _main_venc_recv_started = started;
    };

    auto restore_pipeline_state = [&]() {
        if (!venc_resolution_updated && !venc_fps_updated && !internal_state_updated) {
            return;
        }

        if (!link_stopped) {
            if (link_deinit()) {
                link_stopped = true;
            } else {
                (*this->_logger)->error(
                    "failed to deinit AX link while rolling back VIN pipeline "
                    "reconfigure");
            }
        }
        if (!vin_stopped) {
            if (vin_deinit()) {
                vin_stopped = true;
            } else {
                (*this->_logger)->error(
                    "failed to deinit VIN while rolling back VIN pipeline "
                    "reconfigure");
            }
        }

        if (use_main_venc) {
            if (venc_fps_updated) {
                if (!venc_set_fps(
                        chn, static_cast<float>(old_venc_fps), 0)) {
                    (*this->_logger)->error(
                        "failed to restore main VENC fps after VIN pipeline "
                        "reconfigure failure");
                }
                venc_fps_updated = false;
            }
            if (venc_resolution_updated) {
                if (!venc_set_resolution(
                        chn, old_venc_width, old_venc_height)) {
                    (*this->_logger)->error(
                        "failed to restore main VENC resolution after VIN "
                        "pipeline reconfigure failure");
                }
                venc_resolution_updated = false;
            }

            _active_venc->width = old_venc_width;
            _active_venc->height = old_venc_height;
            _active_venc->fps = old_venc_fps;
        }

        _width = old_width;
        _height = old_height;
        _fps = old_fps;
        internal_state_updated = false;
    };

    auto restore_after_failure = [&](const char* step) {
        (*this->_logger)->error("VIN pipeline reconfigure failed at {}", step);

        restore_pipeline_state();

        if (vin_stopped) {
            if (vin_init()) {
                vin_stopped = false;
            } else {
                (*this->_logger)->error(
                    "failed to restore VIN after pipeline reconfigure failure");
            }
        }
        if (link_stopped) {
            if (link_init()) {
                link_stopped = false;
            } else {
                (*this->_logger)->error(
                    "failed to restore AX link after pipeline reconfigure "
                    "failure");
            }
        }
        if (use_main_venc && was_started && recv_stopped) {
            if (_active_venc && _active_venc->start(chn)) {
                set_recv_started(true);
                recv_stopped = false;
            } else {
                (*this->_logger)->error(
                    "failed to restore main VENC recv after pipeline "
                    "reconfigure failure");
            }
        }

        clear_main_venc_window();
        return false;
    };

    if (use_main_venc && was_started) {
        if (!_active_venc->stop(chn)) {
            clear_main_venc_window();
            (*this->_logger)->error(
                "failed to stop main VENC recv before VIN pipeline "
                "reconfigure");
            return false;
        }
        set_recv_started(false);
        recv_stopped = true;
    }

    if (!link_deinit()) {
        return restore_after_failure("link_deinit");
    }
    link_stopped = true;

    if (!vin_deinit()) {
        return restore_after_failure("vin_deinit");
    }
    vin_stopped = true;

    if (use_main_venc) {
        if (!venc_set_resolution(chn, width, height)) {
            return restore_after_failure("main_venc_set_resolution");
        }
        venc_resolution_updated = true;
        if (!venc_set_fps(chn, static_cast<float>(fps), 0)) {
            return restore_after_failure("main_venc_set_fps");
        }
        venc_fps_updated = true;
    }

    // AX SDK skips dynamic frame-rate updates for PT_JPEG. Keep the JENC
    // channel attributes unchanged; linked VIN frames provide each frame's
    // dimensions, while the pipeline metadata tracks the current source FPS.
    _width = width;
    _height = height;
    _fps = fps;
    internal_state_updated = true;

    if (!vin_init()) {
        return restore_after_failure("vin_init");
    }
    vin_stopped = false;

    if (!link_init()) {
        return restore_after_failure("link_init");
    }
    link_stopped = false;

    if (use_main_venc && was_started) {
        if (!_active_venc->start(chn)) {
            return restore_after_failure("main_venc_start_recv");
        }
        set_recv_started(true);
        recv_stopped = false;
    }

    (*this->_logger)->info(
        "VIN pipeline reconfigured to {}x{}@{} (main VENC={}, JENC={})",
        width,
        height,
        fps,
        use_main_venc ? "enabled" : "disabled",
        _jenc_enabled ? "enabled" : "disabled");
    clear_main_venc_window();
    return true;
}

AxeraUtils::VencFrameResult AxeraUtils::venc_get_frame(VencStream& stream, int timeout_ms)
{
    const int chn = _active_venc ? _active_venc->chn : _h264.chn;
    std::unique_lock<std::mutex> lock(_main_venc_mutex);
    if (_main_venc_reconfiguring) {
        return VencFrameResult::Reconfiguring;
    }

    AX_U32 ret = AX_VENC_GetStream(chn, &stream.raw, timeout_ms);
    if (ret == AX_ERR_VENC_QUEUE_EMPTY) {
        return VencFrameResult::QueueEmpty;
    }
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_GetStream failed, ret=0x{:X}", ret);
        return VencFrameResult::Error;
    }

    ++_main_venc_stream_users;
    return VencFrameResult::Ok;
}

bool AxeraUtils::venc_release_frame(VencStream& stream)
{
    const int chn = _active_venc ? _active_venc->chn : _h264.chn;
    AX_U32 ret = AX_VENC_ReleaseStream(chn, &stream.raw);
    {
        std::lock_guard<std::mutex> lock(_main_venc_mutex);
        if (_main_venc_stream_users > 0) {
            --_main_venc_stream_users;
        }
    }
    _main_venc_cv.notify_all();

    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VENC_ReleaseStream failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::venc_request_idr(int chn, bool instant)
{
    AX_U32 ret = AX_VENC_RequestIDR(chn, instant ? AX_TRUE : AX_FALSE);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_VENC_RequestIDR failed, ret=0x{:X}", chn, ret);
        return false;
    }

    (*this->_logger)->debug("[{}] IDR frame requested (instant={})", chn, instant);
    return true;
}
