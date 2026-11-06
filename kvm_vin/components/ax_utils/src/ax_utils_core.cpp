#include <ax_sys_api.h>

// logger
#include <logger/default2.hpp>
#include <logger/file_sink.hpp>
#include <logger/tools.hpp>

#include "ax_utils.hpp"

AxeraUtils::~AxeraUtils(void)
{
    if (!this->_init) {
        return;
    }

    if (this->_pool_deinit() != true) {
        (*this->_logger)->error("Pool deinit failed");
    }

    AX_U32 ret = AX_SYS_Deinit();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_SYS_Deinit failed, ret=0x{:X}", ret);
        return;
    }

    this->_init = false;
}

bool AxeraUtils::init(void)
{
    if (this->_init) {
        return true;
    }

    AX_U32 ret = AX_SYS_Init();
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_SYS_Init failed, ret=0x{:X}", ret);
        this->_init = false;
        return false;
    }

    ret = AX_SYS_InitPTSBase(0);
    if (ret != AX_SUCCESS) {
        (*this->_logger)->error("AX_SYS_InitPTSBase failed, ret=0x{:X}", ret);
        AX_SYS_Deinit();
        this->_init = false;
        return false;
    }

    this->_init = true;
    return true;
}

/* Logger */
void AxeraUtils::logger_init(const std::string& level,
                                   const std::string& type,
                                   const std::string& ofilepath)
{
    /* Use Default Logger */
    this->_logger = std::make_shared<DefaultLogger2>("AxeraUtils");
    this->_logger->logger_init(level, type, ofilepath);
}

std::shared_ptr<spdlog::logger> AxeraUtils::operator->()
{
    return this->_logger->operator->();
}
