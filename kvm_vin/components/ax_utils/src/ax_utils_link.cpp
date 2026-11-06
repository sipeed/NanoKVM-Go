#include <ax_sys_api.h>
#include <ax_ivps_api.h>

#include "ax_utils.hpp"

bool AxeraUtils::link_init(void)
{
    std::lock_guard<std::mutex> link_lock(_link_mutex);

    if (this->jenc_enabled() && !_jenc_link_snapshot_guard.owns_lock()) {
        _jenc_link_snapshot_guard = std::unique_lock<std::mutex>(_snapshot_mutex);
    }

    auto release_jenc_snapshot_guard = [this]() {
        if (_jenc_link_snapshot_guard.owns_lock()) {
            _jenc_link_snapshot_guard.unlock();
        }
        _jenc_link_snapshot_guard = std::unique_lock<std::mutex>();
    };

    AX_MOD_INFO_T src_mod, dst_mod;

    src_mod.enModId = AX_ID_VIN;
    src_mod.s32GrpId = 0;
    src_mod.s32ChnId = this->_mipi_dev;
    dst_mod.enModId = AX_ID_VENC;
    dst_mod.s32GrpId = 0;

    auto link_channel = [&](int chn, const char* name) {
        dst_mod.s32ChnId = chn;
        AX_U32 ret = AX_SYS_Link(&src_mod, &dst_mod);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("AX_SYS_Link failed (VIN->{}), ret=0x{:X}", name, ret);
            return false;
        }
        return true;
    };

    auto unlink_channel = [&](int chn) {
        dst_mod.s32ChnId = chn;
        (void)AX_SYS_UnLink(&src_mod, &dst_mod);
    };

    bool linked_venc = false;
    if (this->venc_enabled()) {
        if (!link_channel(this->get_venc_chn(), "VENC")) {
            return false;
        }
        linked_venc = true;
    }

    if (this->jenc_enabled()) {
        if (!link_channel(this->_jpeg.chn, "JENC")) {
            if (linked_venc) {
                unlink_channel(this->get_venc_chn());
            }
            return false;
        }
    }

    release_jenc_snapshot_guard();
    return true;
}

bool AxeraUtils::link_deinit(void)
{
    std::lock_guard<std::mutex> link_lock(_link_mutex);

    bool snapshot_guard_acquired_here = false;
    if (this->jenc_enabled() && !_jenc_link_snapshot_guard.owns_lock()) {
        _jenc_link_snapshot_guard = std::unique_lock<std::mutex>(_snapshot_mutex);
        snapshot_guard_acquired_here = true;
    }

    auto release_jenc_snapshot_guard_if_acquired_here = [&]() {
        if (!snapshot_guard_acquired_here) {
            return;
        }
        if (_jenc_link_snapshot_guard.owns_lock()) {
            _jenc_link_snapshot_guard.unlock();
        }
        _jenc_link_snapshot_guard = std::unique_lock<std::mutex>();
        snapshot_guard_acquired_here = false;
    };

    AX_MOD_INFO_T src_mod, dst_mod;

    src_mod.enModId = AX_ID_VIN;
    src_mod.s32GrpId = 0;
    src_mod.s32ChnId = this->_mipi_dev;
    dst_mod.enModId = AX_ID_VENC;
    dst_mod.s32GrpId = 0;

    auto unlink_channel = [&](int chn, const char* name) {
        dst_mod.s32ChnId = chn;
        AX_U32 ret = AX_SYS_UnLink(&src_mod, &dst_mod);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error("AX_SYS_UnLink failed (VIN->{}), ret=0x{:X}", name, ret);
            return false;
        }
        return true;
    };

    auto link_channel = [&](int chn, const char* name) {
        dst_mod.s32ChnId = chn;
        AX_U32 ret = AX_SYS_Link(&src_mod, &dst_mod);
        if (ret != AX_SUCCESS) {
            (*this->_logger)->error(
                "AX_SYS_Link failed while restoring VIN->{}, ret=0x{:X}",
                name,
                ret);
            return false;
        }
        return true;
    };

    bool venc_unlinked = false;
    if (this->venc_enabled()) {
        if (!unlink_channel(this->get_venc_chn(), "VENC")) {
            release_jenc_snapshot_guard_if_acquired_here();
            return false;
        }
        venc_unlinked = true;
    }

    if (this->jenc_enabled()) {
        if (!unlink_channel(this->_jpeg.chn, "JENC")) {
            const bool restored =
                !venc_unlinked
                || link_channel(this->get_venc_chn(), "VENC");
            if (restored) {
                release_jenc_snapshot_guard_if_acquired_here();
            } else {
                (*this->_logger)->error(
                    "failed to restore VIN->VENC after JENC unlink failure; "
                    "snapshot access remains blocked");
            }
            return false;
        }
    }

    return true;
}
