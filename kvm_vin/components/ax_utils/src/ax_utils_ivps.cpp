#include <ax_ivps_api.h>
#include <ax_sys_api.h>
#include <ax_buffer_tool.h>

#include <memory>
#include <poll.h>

#include "ax_utils.hpp"

template<std::size_t Align>
constexpr std::size_t size_align(std::size_t x) noexcept {
    static_assert(Align > 0);
    return (x + Align - 1) / Align * Align;
}

bool AxeraUtils::ivps_init(void)
{
    AX_U32 ret = AX_IVPS_Init();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_Init failed, ret=0x{:X}", ret);
        return false;
    };

    AX_IVPS_GRP_ATTR_T ivps_grp_attr = {
        .nInFifoDepth = 1,
        .ePipeline = AX_IVPS_PIPELINE_DEFAULT
    };

    this->_ivps_grp = 0;
    ret = AX_IVPS_CreateGrp(this->_ivps_grp, &ivps_grp_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_CreateGrp failed, ret=0x{:X}", ret);
        return false;
    }

    AX_IVPS_PIPELINE_ATTR_T pipeline_attr = {};
    pipeline_attr.nOutChnNum = 1;
    pipeline_attr.nInDebugFifoDepth = 0;
    pipeline_attr.nOutFifoDepth[0] = 1;
    pipeline_attr.eFRCMode = AX_IVPS_FRC_RATIO;
    pipeline_attr.tFilter[0][0].bEngage = AX_TRUE;
    pipeline_attr.tFilter[0][0].nDstPicWidth = this->_width;
    pipeline_attr.tFilter[0][0].nDstPicHeight = this->_height;
    pipeline_attr.tFilter[0][0].nDstPicStride = size_align<128>(pipeline_attr.tFilter[0][0].nDstPicWidth);
    pipeline_attr.tFilter[0][0].eDstPicFormat = AX_FORMAT_YUV420_SEMIPLANAR;
    pipeline_attr.tFilter[0][0].eEngine = AX_IVPS_ENGINE_TDP;
    pipeline_attr.tFilter[0][0].tFRC.fSrcFrameRate = _fps;
    pipeline_attr.tFilter[0][0].tFRC.fDstFrameRate = _fps;
    pipeline_attr.tFilter[0][0].tCompressInfo.enCompressMode = AX_COMPRESS_MODE_LOSSY;
    pipeline_attr.tFilter[0][0].tCompressInfo.u32CompressLevel = 4;

    ret = AX_IVPS_SetPipelineAttr(this->_ivps_grp, &pipeline_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_SetPipelineAttr failed, ret=0x{:X}", ret);
        return false;
    }

    AX_IVPS_POOL_ATTR_T ivps_pool_attr = {
        .ePoolSrc = AX_POOL_SOURCE_USER,
        .PoolId = static_cast<AX_POOL>(this->_ivps_pool_id),
    };

    ret = AX_IVPS_SetGrpPoolAttr(this->_ivps_grp, &ivps_pool_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_SetGrpPoolAttr failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_SetChnPoolAttr(this->_ivps_grp, 0, &ivps_pool_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_SetChnPoolAttr failed for channel 0, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_EnableChn(this->_ivps_grp, 0);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_EnableChn failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_StartGrp(this->_ivps_grp);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_StartGrp failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::ivps_deinit(void)
{
    AX_U32 ret = AX_IVPS_StopGrp(this->_ivps_grp);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_StopGrp failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_DisableChn(this->_ivps_grp, 0);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_DisableChn failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_DestoryGrp(this->_ivps_grp);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_DestoryGrp failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_Deinit failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::ivps_set_crop(bool enable, int x, int y, int w, int h)
{
    AX_IVPS_PIPELINE_ATTR_T pipeline_attr = {};

    AX_U32 ret = AX_IVPS_GetPipelineAttr(this->_ivps_grp, &pipeline_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_IVPS_GetPipelineAttr failed, ret=0x{:X}", this->_ivps_grp, ret);
        return false;
    }

    if (enable) {
        pipeline_attr.tFilter[0][0].bCrop = AX_TRUE;
        pipeline_attr.tFilter[0][0].nDstPicWidth = w;
        pipeline_attr.tFilter[0][0].nDstPicHeight = h;
        pipeline_attr.tFilter[0][0].nDstPicStride = size_align<128>(pipeline_attr.tFilter[0][0].nDstPicWidth);
        pipeline_attr.tFilter[0][0].tCropRect.nX = x;
        pipeline_attr.tFilter[0][0].tCropRect.nY = y;
        pipeline_attr.tFilter[0][0].tCropRect.nW = w;
        pipeline_attr.tFilter[0][0].tCropRect.nH = h;
    } else {
        pipeline_attr.tFilter[0][0].bCrop = AX_FALSE;
        pipeline_attr.tFilter[0][0].nDstPicWidth = this->_width;
        pipeline_attr.tFilter[0][0].nDstPicHeight = this->_height;
        pipeline_attr.tFilter[0][0].nDstPicStride = size_align<128>(pipeline_attr.tFilter[0][0].nDstPicWidth);
        pipeline_attr.tFilter[0][0].tCropRect.nX = 0;
        pipeline_attr.tFilter[0][0].tCropRect.nY = 0;
        pipeline_attr.tFilter[0][0].tCropRect.nW = 0;
        pipeline_attr.tFilter[0][0].tCropRect.nH = 0;
    }

    ret = AX_IVPS_DisableChn(this->_ivps_grp, 0);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_DisableChn failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_IVPS_SetPipelineAttr(this->_ivps_grp, &pipeline_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("[{}] AX_IVPS_SetPipelineAttr failed, ret=0x{:X}", this->_ivps_grp, ret);
        return false;
    }

    ret = AX_IVPS_EnableChn(this->_ivps_grp, 0);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_IVPS_EnableChn failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}
