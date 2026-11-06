#include <dlfcn.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <fstream>
#include <unistd.h>

#include <ax_sys_api.h>
#include <ax_mipi_rx_api.h>
#include <ax_vin_api.h>
#include <ax_isp_api.h>

#include "ax_utils.hpp"

std::string_view ISP_LIB_PATH = "/opt/lib/libsns_dummy.so";

struct RegVal {
    uint32_t addr;
    uint32_t value;
};

const RegVal pad_settings[] = {
    {0x0230000C, 0x00040003},
    {0x02300018, 0x00040003},
    {0x02300024, 0x00060003},
    {0x02300030, 0x00040003},
    {0x0230003C, 0x00060003},
    {0x02300084, 0x00040003},
};

bool AxeraUtils::vin_init(void)
{
    AX_U32 ret = this->_mipi_init();
    if (ret != true) {
        (*this->_logger)->error("axera_mipi_init failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VIN_DEV_ATTR_T dev_attr{};
    dev_attr.eDevMode = AX_VIN_DEV_ONLINE;
    dev_attr.bImgDataEnable = AX_TRUE;
    dev_attr.bNonImgDataEnable = AX_FALSE;
    dev_attr.eSnsIntfType = AX_SNS_INTF_TYPE_MIPI_RAW;
    dev_attr.eSnsMode = AX_SNS_LINEAR_ONLY_MODE;
    dev_attr.eBayerPattern = AX_BP_BGGR;
    dev_attr.ePixelFmt = AX_FORMAT_BAYER_RAW_16BPP;
    dev_attr.eSnsOutputMode = AX_SNS_NORMAL;
    dev_attr.tFrameRateCtrl.fSrcFrameRate = 1.0f;
    dev_attr.tFrameRateCtrl.fDstFrameRate = 1.0f;
    dev_attr.tCompressInfo.enCompressMode = AX_COMPRESS_MODE_NONE;
    dev_attr.tCompressInfo.u32CompressLevel = 0;
    dev_attr.tMipiIntfAttr.szImgVc[0] = 0;
    dev_attr.tMipiIntfAttr.szImgDt[0] = AX_MIPI_CSI_DT_YUV422_8BIT;

    for (int i = 0; i < 4; ++i) {
        dev_attr.tDevImgRgn[i].nStartX = 0;
        dev_attr.tDevImgRgn[i].nStartY = 0;
        dev_attr.tDevImgRgn[i].nWidth = _width;
        dev_attr.tDevImgRgn[i].nHeight = _height;
    }

    (*this->_logger)->debug(
        "eDevMode: {}, bImgDataEnable: {}, bNonImgDataEnable: {}, eSnsIntfType: {}, eSnsMode: {}, eBayerPattern: {}, ePixelFmt: {}",
        static_cast<int>(dev_attr.eDevMode), static_cast<int>(dev_attr.bImgDataEnable), static_cast<int>(dev_attr.bNonImgDataEnable),
        static_cast<int>(dev_attr.eSnsIntfType), static_cast<int>(dev_attr.eSnsMode), static_cast<int>(dev_attr.eBayerPattern), static_cast<int>(dev_attr.ePixelFmt)
    );

    for (int i = 0; i < 4; ++i) {
        (*this->_logger)->debug(
            "tDevImgRgn[{}]: ({}, {}, {}, {})", i,
            static_cast<int>(dev_attr.tDevImgRgn[i].nStartX), static_cast<int>(dev_attr.tDevImgRgn[i].nStartY),
            static_cast<int>(dev_attr.tDevImgRgn[i].nWidth), static_cast<int>(dev_attr.tDevImgRgn[i].nHeight)
        );
    }

    (*this->_logger)->debug(
        "tFrameRateCtrl.fSrcFrameRate: {:.2f}, tFrameRateCtrl.fDstFrameRate: {:.2f}",
        dev_attr.tFrameRateCtrl.fSrcFrameRate, dev_attr.tFrameRateCtrl.fDstFrameRate
    );

    (*this->_logger)->debug(
        "tCompressInfo.enCompressMode: {}, tCompressInfo.u32CompressLevel: {}",
        static_cast<int>(dev_attr.tCompressInfo.enCompressMode), static_cast<int>(dev_attr.tCompressInfo.u32CompressLevel)
    );

    (*this->_logger)->debug(
        "szImgVc[0]: {}, szImgDt[0]: 0x{:02x}",
        dev_attr.tMipiIntfAttr.szImgVc[0], dev_attr.tMipiIntfAttr.szImgDt[0]
    );

    ret = AX_VIN_CreateDev(this->_mipi_dev, &dev_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_CreateDev failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_SetDevAttr(this->_mipi_dev, &dev_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_CreateDev failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VIN_DEV_BIND_PIPE_T dev_bind_pipe = {
        1,
        { static_cast<AX_U32>(this->_mipi_dev) },
        { 0x1 },
    };

    ret = AX_VIN_SetDevBindPipe(this->_mipi_dev, &dev_bind_pipe);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_SetDevBindPipe failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_SetDevBindMipi(this->_mipi_dev, this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_SetDevBindMipi failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VIN_PIPE_ATTR_T pipe_attr = {
        .ePipeWorkMode = AX_VIN_PIPE_ISP_BYPASS_MODE,
        .tPipeImgRgn = {0, 0, static_cast<AX_U32>(_width), static_cast<AX_U32>(_height)},
        .nWidthStride = static_cast<AX_U32>(_width),
        .eBayerPattern = AX_BP_BGGR,
        .ePixelFmt = AX_FORMAT_BAYER_RAW_16BPP,
        .eSnsMode = AX_SNS_LINEAR_ONLY_MODE,
        .tCompressInfo = {AX_COMPRESS_MODE_NONE, 0},
        .tNrAttr = {{AX_FALSE, {AX_COMPRESS_MODE_NONE, 0}}, {AX_FALSE, {AX_COMPRESS_MODE_NONE, 0}}},
        .tFrameRateCtrl = {AX_INVALID_FRMRATE, AX_INVALID_FRMRATE},
    };

    ret = AX_VIN_CreatePipe(this->_mipi_dev, &pipe_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_CreatePipe failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_SetPipeAttr(this->_mipi_dev, &pipe_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VI_SetPipeAttr failed, ret=0x{:X}", ret);
        return false;
    }

    this->_isp_lib = dlopen(ISP_LIB_PATH.data(), RTLD_LOCAL | RTLD_LAZY);
    if (!this->_isp_lib) {
        (*this->_logger)->error("ISP dlopen failed: {}", ISP_LIB_PATH);
        return false;
    } else {
        (*this->_logger)->debug("dlopen {} success", ISP_LIB_PATH);
    }

    AX_SENSOR_REGISTER_FUNC_T *sns_dummy_obj = (AX_SENSOR_REGISTER_FUNC_T *)dlsym(this->_isp_lib, "gSnsdummyObj");

    ret = AX_ISP_RegisterSensor(this->_mipi_dev, sns_dummy_obj);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP Register Sensor Failed, ret=0x{:X}", ret);
        return false;
    }

    AX_SNS_ATTR_T sns_attr = {
        .nWidth = static_cast<AX_U32>(_width),
        .nHeight = static_cast<AX_U32>(_height),
        .fFrameRate = 60,
        .eSnsMode = AX_SNS_LINEAR_ONLY_MODE,
        .eRawType = AX_RT_RAW16,
        .eBayerPattern = AX_BP_BGGR,
        .bTestPatternEnable = AX_FALSE,
    };

    ret = AX_ISP_SetSnsAttr(this->_mipi_dev, &sns_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_SetSnsAttr failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Create(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Create failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Open(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Open failed, ret=0x{:X}", ret);
        return false;
    }

    AX_VIN_CHN_ATTR_T chn_attr = {
        .nWidth = static_cast<AX_U32>(_width),
        .nHeight = static_cast<AX_U32>(_height),
        .nWidthStride = static_cast<AX_U32>(_width),
        .eImgFormat = AX_FORMAT_YUV420_SEMIPLANAR,
        .nDepth = 1,
        .tCompressInfo = {AX_COMPRESS_MODE_NONE, 0},
        .tFrameRateCtrl = {AX_INVALID_FRMRATE, AX_INVALID_FRMRATE},
    };

    ret = AX_VIN_SetChnAttr(this->_mipi_dev, static_cast<AX_VIN_CHN_ID_E>(this->_mipi_dev), &chn_attr);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_SetChnAttr failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_SetChnFrameMode(this->_mipi_dev, static_cast<AX_VIN_CHN_ID_E>(this->_mipi_dev), AX_VIN_FRAME_MODE_OFF);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_SetChnFrameMode failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_EnableChn(this->_mipi_dev, static_cast<AX_VIN_CHN_ID_E>(this->_mipi_dev));
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_EnableChn failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_StartPipe(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_StartPipe failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Start(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Start failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_EnableDev(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_EnableDev failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_StreamOn(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_StreamOn failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::vin_deinit(void)
{
    AX_U32 ret = AX_ISP_StreamOff(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_StreamOff failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_DisableDev(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_DisableDev failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Stop(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Stop failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_StopPipe(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_StopPipe failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_DisableChn(this->_mipi_dev, static_cast<AX_VIN_CHN_ID_E>(this->_mipi_dev));
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_DisableChn failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Close(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Close failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_Destroy(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_Destroy failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_ISP_UnRegisterSensor(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_ISP_UnRegisterSensor failed, ret=0x{:X}", ret);
        return false;
    }

    if (this->_isp_lib) {
        dlclose(this->_isp_lib);
        (*this->_logger)->debug("dlclose {}", ISP_LIB_PATH);
        this->_isp_lib = nullptr;
    }

    ret = AX_VIN_DestroyPipe(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_DestroyPipe failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_VIN_DestroyDev(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_DestroyDev failed, ret=0x{:X}", ret);
        return false;
    }

    if (this->_mipi_dev >= 0) {
        if (!this->_mipi_deinit()) {
            (*this->_logger)->error("MIPI deinit failed");
        }
    }

    return true;
}

bool AxeraUtils::dump_yuyv_to_file(const std::string& filepath, int timeout_ms)
{
    AX_IMG_INFO_T img_info{};
    AX_U32 ret = AX_VIN_GetYuvFrame(this->_mipi_dev, AX_VIN_CHN_ID_MAIN, &img_info, timeout_ms);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_GetYuvFrame failed, ret=0x{:X}", ret);
        return false;
    }

    const auto& frame = img_info.tFrameInfo.stVFrame;
    (*this->_logger)->debug(
        "Got YUYV422 frame: SeqNum={}, FrameSize={}, Width={}, Height={}, Stride={}, Format=0x{:02X}",
        frame.u64SeqNum,
        frame.u32FrameSize,
        frame.u32Width,
        frame.u32Height,
        frame.u32PicStride[0],
        static_cast<uint32_t>(frame.enImgFormat)
    );

    std::ofstream outfile(filepath, std::ios::binary);
    if (!outfile.is_open()) {
        (*this->_logger)->error("Failed to open file for writing: {}", filepath);
        AX_VIN_ReleaseYuvFrame(this->_mipi_dev, AX_VIN_CHN_ID_MAIN, &img_info);
        return false;
    }

    AX_U32 frame_bytes = frame.u32PicStride[0] * frame.u32Height * 2;

    AX_VOID *vir_addr = AX_SYS_Mmap(frame.u64PhyAddr[0], frame_bytes);
    if (!vir_addr) {
        (*this->_logger)->error(
            "AX_SYS_Mmap failed for YUYV422 frame, addr=0x{:016X}, size={}",
            static_cast<unsigned long long>(frame.u64PhyAddr[0]),
            frame_bytes
        );
        outfile.close();
        AX_VIN_ReleaseYuvFrame(this->_mipi_dev, AX_VIN_CHN_ID_MAIN, &img_info);
        return false;
    }

    (*this->_logger)->debug(
        "AX_SYS_Mmap success: PhyAddr=0x{:016X}, VirtAddr={}, size={}",
        static_cast<unsigned long long>(frame.u64PhyAddr[0]),
        vir_addr,
        frame_bytes
    );

    outfile.write(reinterpret_cast<char*>(vir_addr), frame_bytes);
    if (!outfile.good()) {
        (*this->_logger)->error("Failed to write full YUYV422 frame to file");
    } else {
        (*this->_logger)->debug("YUYV422 frame written successfully to {}", filepath);
    }

    outfile.close();

    ret = AX_SYS_Munmap(vir_addr, frame_bytes);
    if (ret) {
        (*this->_logger)->error("AX_SYS_Munmap failed for addr=0x{:016X}, ret=0x{:X}",
                                static_cast<unsigned long long>(frame.u64PhyAddr[0]), ret);
    }

    AX_VIN_ReleaseYuvFrame(this->_mipi_dev, AX_VIN_CHN_ID_MAIN, &img_info);
    return true;
}

bool AxeraUtils::_mipi_init(void)
{
    AX_U32 ret = AX_MIPI_RX_Init();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_Init failed, ret=0x{:X}", ret);
        return false;
    }

    AX_MIPI_RX_SetLaneCombo(AX_LANE_COMBO_MODE_0);

    AX_MIPI_RX_DEV_T mipi_dev = {
        .eInputMode = AX_INPUT_MODE_MIPI,
        .tMipiAttr = {
            .ePhyMode = AX_MIPI_PHY_TYPE_DPHY,
            .eLaneNum = AX_MIPI_DATA_LANE_4,
            .nDataRate = 600,
            .nDataLaneMap = {0, 1, 3, 4},
            .nClkLane = {2, 5}
        }
    };

    this->_mipi_dev = 0;
    ret = AX_MIPI_RX_SetAttr(this->_mipi_dev, &mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_SetAttr failed, ret=0x{:X}", ret);
        this->_mipi_dev = -1;
        return false;
    }

    ret = AX_MIPI_RX_Reset(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_Reset, ret=0x{:X}", ret);
        this->_mipi_dev = -1;
        return false;
    }

    ret = AX_MIPI_RX_Start(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_Start failed, ret=0x{:X}", ret);
        this->_mipi_dev = -1;
        return false;
    }

    ret = _set_pinmux();
    if (ret != true) {
        (*this->_logger)->error("Set pinmux failed");
        this->_mipi_dev = -1;
        return false;
    }

    return true;
}

bool AxeraUtils::_mipi_deinit(void)
{
    AX_U32 ret = AX_MIPI_RX_Stop(this->_mipi_dev);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_Stop failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_MIPI_RX_DeInit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_MIPI_RX_DeInit failed, ret=0x{:X}", ret);
        return false;
    }

    this->_mipi_dev = -1;
    return true;
}

bool AxeraUtils::_set_pinmux(void)
{
    int mem_fd = open("/dev/mem", O_RDWR | O_SYNC);
    if (mem_fd < 0) {
        (*this->_logger)->error("Failed to open /dev/mem, error: {}", strerror(errno));
        return false;
    }

    for (const auto& rv : pad_settings) {
        off_t page_base = rv.addr & ~(4096 - 1);
        off_t page_offset = rv.addr - page_base;

        void* map_base = mmap(nullptr, 4096, PROT_READ | PROT_WRITE, MAP_SHARED, mem_fd, page_base);
        if (map_base == MAP_FAILED) {
            (*this->_logger)->error("mmap failed for address 0x{:X}, error: {}", rv.addr, strerror(errno));
            continue;
        }

        volatile uint32_t* reg = reinterpret_cast<volatile uint32_t*>(
            reinterpret_cast<uint8_t*>(map_base) + page_offset);
        *reg = rv.value;

        munmap(map_base, 4096);
    }

    close(mem_fd);
    return true;
}
