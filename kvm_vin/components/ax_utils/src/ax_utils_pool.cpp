#include <fstream>
#include <unistd.h>
#include <cstdlib>
#include <cstring>

#include <ax_sys_api.h>
#include <ax_vin_api.h>
#include <ax_buffer_tool.h>

#include "ax_utils.hpp"

bool AxeraUtils::pool_init(int width, int height)
{
    const std::string reboot_flag = "/var/lib/kvm/have_reboot";

    AX_U32 ret = AX_POOL_Exit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_POOL_Exit failed, ret=0x{:X}", ret);

        std::ifstream flag_file(reboot_flag);
        if (!flag_file.good()) {
            system("mkdir -p /var/lib/kvm");

            std::ofstream create_flag(reboot_flag);
            create_flag.close();

            (*this->_logger)->warn("Rebooting system due to AX_POOL_Exit failure...");
            sync();
            sleep(1);
            system("reboot");
            sleep(5);
        } else {
            (*this->_logger)->error("Already rebooted once, not rebooting again");
            flag_file.close();
        }

        return false;
    } else {
        std::remove(reboot_flag.c_str());
    }

    AX_POOL_FLOORPLAN_T pool_floor_plan = {};
    AX_FRAME_COMPRESS_INFO_T compress_info {
        .enCompressMode = AX_COMPRESS_MODE_NONE,
        .u32CompressLevel = 0
    };

    pool_floor_plan.CommPool[0].MetaSize = 0x1000;
    pool_floor_plan.CommPool[0].BlkSize = AX_VIN_GetImgBufferSize(
        height, width, AX_FORMAT_BAYER_RAW_16BPP, &compress_info, 0);
    pool_floor_plan.CommPool[0].BlkCnt = 5;
    pool_floor_plan.CommPool[0].CacheMode = AX_POOL_CACHE_MODE_NONCACHE;
    std::memset(pool_floor_plan.CommPool[0].PartitionName, 0, sizeof(pool_floor_plan.CommPool[0].PartitionName));
    std::strcpy(reinterpret_cast<char*>(pool_floor_plan.CommPool[0].PartitionName), "anonymous");

    ret = AX_POOL_SetConfig(&pool_floor_plan);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->info("Create common pool err! Pool[0] BlkSize:{}, BlkCnt:{}",
                       pool_floor_plan.CommPool[0].BlkSize,
                       pool_floor_plan.CommPool[0].BlkCnt);
        return false;
    }

    ret = AX_POOL_Init();
    if(ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_POOL_Init fail! ret=0x{:X}", ret);
        return false;
    }

    (*this->_logger)->debug("Create common pool success! Pool[0] BlkSize:{}, BlkCnt:{}",

                       pool_floor_plan.CommPool[0].BlkSize,
                       pool_floor_plan.CommPool[0].BlkCnt);

    ret = AX_VIN_Init();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_Init failed, ret=0x{:X}", ret);
        return false;
    }

    return true;
}

bool AxeraUtils::_pool_deinit(void)
{
    const std::string reboot_flag = "/var/lib/kvm/have_reboot";

    AX_U32 ret = AX_VIN_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_VIN_DeInit failed, ret=0x{:X}", ret);
        return false;
    }

    ret = AX_POOL_Exit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_POOL_Exit failed, ret=0x{:X}", ret);
        return false;
    }

    // Success - remove reboot flag file if exists
    std::remove(reboot_flag.c_str());

    return true;
}
