#include <deque>
#include <memory>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "ax_utils.hpp"

#include <ax_sys_api.h>
#include <ax_venc_api.h>
#include <catch2/catch_test_macros.hpp>
#include <spdlog/logger.h>
#include <spdlog/sinks/null_sink.h>

namespace {

class NullLogger final : public ILogger {
  public:
    NullLogger()
    {
        auto sink = std::make_shared<spdlog::sinks::null_sink_mt>();
        _logger = std::make_shared<spdlog::logger>(
            "vin-pipeline-reconfigure-test", std::move(sink));
    }

    void logger_init(const std::string&,
                     const std::string&,
                     const std::string&) override
    {}

    std::shared_ptr<spdlog::logger> operator->() override { return _logger; }

  private:
    std::shared_ptr<spdlog::logger> _logger;
};

struct FakeBackend {
    std::vector<std::string> events;
    std::deque<AX_S32> link_results;
    std::deque<AX_S32> unlink_results;
    std::deque<AX_S32> start_results;
    std::deque<AX_S32> stop_results;
    std::deque<bool> vin_init_results;
    std::deque<bool> vin_deinit_results;
    AX_VENC_CHN_ATTR_T channel_attr {};
    AX_VENC_CHN_ATTR_T created_channel_attr {};
    AX_VENC_RC_PARAM_T rc_param {};

    void reset_encoder(int width, int height, int fps)
    {
        channel_attr = {};
        channel_attr.stVencAttr.u32PicWidthSrc =
            static_cast<AX_U32>(width);
        channel_attr.stVencAttr.u32PicHeightSrc =
            static_cast<AX_U32>(height);
        rc_param = {};
        rc_param.stFrameRate.fSrcFrameRate = static_cast<AX_F32>(fps);
        rc_param.stFrameRate.fDstFrameRate = static_cast<AX_F32>(fps);
    }
};

FakeBackend* active_backend = nullptr;

class BackendScope {
  public:
    explicit BackendScope(FakeBackend& backend)
    {
        REQUIRE(active_backend == nullptr);
        active_backend = &backend;
    }

    ~BackendScope() { active_backend = nullptr; }

    BackendScope(const BackendScope&) = delete;
    BackendScope& operator=(const BackendScope&) = delete;
};

FakeBackend& backend()
{
    if (active_backend == nullptr) {
        throw std::logic_error("reconfigure test backend is not active");
    }
    return *active_backend;
}

template <typename Value>
Value next_result(std::deque<Value>& results, Value fallback)
{
    if (results.empty()) {
        return fallback;
    }
    const Value result = results.front();
    results.pop_front();
    return result;
}

std::string dimensions(int width, int height, int fps)
{
    return std::to_string(width) + 'x' + std::to_string(height) + '@'
        + std::to_string(fps);
}

void require_dimensions(AxeraUtils& utils,
                        int expected_width,
                        int expected_height,
                        int expected_fps)
{
    int width = 0;
    int height = 0;
    int fps = 0;
    REQUIRE(utils.get_resolution(width, height));
    REQUIRE(utils.get_fps(fps));
    REQUIRE(width == expected_width);
    REQUIRE(height == expected_height);
    REQUIRE(fps == expected_fps);
}

void require_encoder_state(const FakeBackend& fake,
                           int expected_width,
                           int expected_height,
                           int expected_fps)
{
    REQUIRE(fake.channel_attr.stVencAttr.u32PicWidthSrc
            == static_cast<AX_U32>(expected_width));
    REQUIRE(fake.channel_attr.stVencAttr.u32PicHeightSrc
            == static_cast<AX_U32>(expected_height));
    REQUIRE(fake.rc_param.stFrameRate.fSrcFrameRate
            == static_cast<AX_F32>(expected_fps));
    REQUIRE(fake.rc_param.stFrameRate.fDstFrameRate
            == static_cast<AX_F32>(expected_fps));
}

}  // namespace

AxeraUtils::~AxeraUtils() = default;

void AxeraUtils::logger_init(const std::string&,
                             const std::string&,
                             const std::string&)
{
    _logger = std::make_shared<NullLogger>();
}

std::shared_ptr<spdlog::logger> AxeraUtils::operator->()
{
    return _logger->operator->();
}

bool AxeraUtils::vin_init()
{
    int width = 0;
    int height = 0;
    int fps = 0;
    (void)get_resolution(width, height);
    (void)get_fps(fps);
    auto& fake = backend();
    fake.events.push_back("vin_init:" + dimensions(width, height, fps));
    return next_result(fake.vin_init_results, true);
}

bool AxeraUtils::vin_deinit()
{
    auto& fake = backend();
    fake.events.emplace_back("vin_deinit");
    return next_result(fake.vin_deinit_results, true);
}

bool AxeraUtils::Jpeg::start(int chn)
{
    AX_VENC_RECV_PIC_PARAM_T receive {};
    receive.s32RecvPicNum = -1;
    return AX_VENC_StartRecvFrame(chn, &receive) == AX_SUCCESS;
}

bool AxeraUtils::Jpeg::stop(int chn)
{
    return AX_VENC_StopRecvFrame(chn) == AX_SUCCESS;
}

extern "C" AX_S32 AX_SYS_Link(const AX_MOD_INFO_T*, const AX_MOD_INFO_T*)
{
    auto& fake = backend();
    fake.events.emplace_back("link");
    return next_result(fake.link_results, static_cast<AX_S32>(AX_SUCCESS));
}

extern "C" AX_S32 AX_SYS_UnLink(const AX_MOD_INFO_T*, const AX_MOD_INFO_T*)
{
    auto& fake = backend();
    fake.events.emplace_back("unlink");
    return next_result(fake.unlink_results, static_cast<AX_S32>(AX_SUCCESS));
}

extern "C" AX_S32 AX_VENC_Init(const AX_VENC_MOD_ATTR_T*)
{
    backend().events.emplace_back("venc_init");
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_Deinit()
{
    backend().events.emplace_back("venc_deinit");
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_CreateChnEx(
    VENC_CHN* chn,
    const AX_VENC_CHN_ATTR_T* attr)
{
    auto& fake = backend();
    fake.events.emplace_back("venc_create");
    fake.created_channel_attr = *attr;
    *chn = 2;
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_DestroyChn(VENC_CHN)
{
    backend().events.emplace_back("venc_destroy");
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_GetFd(VENC_CHN)
{
    backend().events.emplace_back("venc_get_fd");
    return 7;
}

extern "C" AX_S32 AX_VENC_StartRecvFrame(
    VENC_CHN,
    const AX_VENC_RECV_PIC_PARAM_T*)
{
    auto& fake = backend();
    fake.events.emplace_back("venc_start");
    return next_result(fake.start_results, static_cast<AX_S32>(AX_SUCCESS));
}

extern "C" AX_S32 AX_VENC_StopRecvFrame(VENC_CHN)
{
    auto& fake = backend();
    fake.events.emplace_back("venc_stop");
    return next_result(fake.stop_results, static_cast<AX_S32>(AX_SUCCESS));
}

extern "C" AX_S32 AX_VENC_ResetChn(VENC_CHN)
{
    backend().events.emplace_back("venc_reset");
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_GetChnAttr(VENC_CHN,
                                      AX_VENC_CHN_ATTR_T* attr)
{
    auto& fake = backend();
    fake.events.emplace_back("venc_get_attr");
    *attr = fake.channel_attr;
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_SetChnAttr(
    VENC_CHN,
    const AX_VENC_CHN_ATTR_T* attr)
{
    auto& fake = backend();
    fake.channel_attr = *attr;
    fake.events.push_back(
        "venc_set_attr:"
        + std::to_string(attr->stVencAttr.u32PicWidthSrc) + 'x'
        + std::to_string(attr->stVencAttr.u32PicHeightSrc));
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_GetRcParam(VENC_CHN,
                                      AX_VENC_RC_PARAM_T* param)
{
    auto& fake = backend();
    fake.events.emplace_back("venc_get_rc");
    *param = fake.rc_param;
    return AX_SUCCESS;
}

extern "C" AX_S32 AX_VENC_SetRcParam(
    VENC_CHN,
    const AX_VENC_RC_PARAM_T* param)
{
    auto& fake = backend();
    fake.rc_param = *param;
    fake.events.push_back(
        "venc_set_rc:"
        + std::to_string(
            static_cast<int>(param->stFrameRate.fDstFrameRate)));
    return AX_SUCCESS;
}

TEST_CASE("Timelapse encoder uses its configured fixed QP",
          "[ax-utils][timelapse]")
{
    FakeBackend fake;
    BackendScope backend_scope(fake);

    AxeraUtils utils(3840, 2160, 30);
    utils.logger_init("off", "console");

    REQUIRE(utils.timelapse_venc_init(42));
    REQUIRE(fake.created_channel_attr.stRcAttr.enRcMode
            == AX_VENC_RC_MODE_H264FIXQP);
    REQUIRE(fake.created_channel_attr.stRcAttr.stH264FixQp.u32IQp == 42);
    REQUIRE(fake.created_channel_attr.stRcAttr.stH264FixQp.u32PQp == 42);
    REQUIRE(fake.created_channel_attr.stRcAttr.stH264FixQp.u32BQp == 42);
    REQUIRE(fake.created_channel_attr.stRcAttr.stH264FixQp.u32Gop == 1);
    REQUIRE(fake.created_channel_attr.stVencAttr.enLinkMode == AX_UNLINK_MODE);
    REQUIRE(utils.timelapse_venc_deinit());
}

TEST_CASE("Timelapse encoder rejects invalid fixed QP",
          "[ax-utils][timelapse]")
{
    FakeBackend fake;
    BackendScope backend_scope(fake);

    AxeraUtils utils(1920, 1080, 30);
    utils.logger_init("off", "console");

    REQUIRE_FALSE(utils.timelapse_venc_init(35));
    REQUIRE_FALSE(utils.timelapse_venc_init(52));
    REQUIRE(fake.events.empty());
}

TEST_CASE("VIN pipeline reconfigure preserves the active main VENC flow",
          "[ax-utils][reconfigure]")
{
    FakeBackend fake;
    fake.reset_encoder(1920, 1080, 30);
    BackendScope backend_scope(fake);

    AxeraUtils utils(1920, 1080, 30);
    utils.logger_init("off", "console");
    utils.set_venc_enabled(true);
    utils.set_jenc_enabled(false);

    REQUIRE(utils.venc_start_recv());
    fake.events.clear();

    REQUIRE(utils.reconfigure_vin_pipeline(1280, 720, 60));

    const std::vector<std::string> expected {
        "venc_stop",
        "unlink",
        "vin_deinit",
        "venc_reset",
        "venc_get_attr",
        "venc_set_attr:1280x720",
        "venc_get_rc",
        "venc_set_rc:60",
        "vin_init:1280x720@60",
        "link",
        "venc_start",
    };
    REQUIRE(fake.events == expected);
    require_dimensions(utils, 1280, 720, 60);
    require_encoder_state(fake, 1280, 720, 60);
    REQUIRE_FALSE(utils.main_venc_reconfiguring());

    fake.events.clear();
    REQUIRE(utils.venc_stop_recv());
    REQUIRE(fake.events == std::vector<std::string> {"venc_stop"});
}

TEST_CASE("VIN pipeline reconfigure skips main VENC when it is disabled",
          "[ax-utils][reconfigure]")
{
    FakeBackend fake;
    fake.reset_encoder(1920, 1080, 30);
    BackendScope backend_scope(fake);

    AxeraUtils utils(1920, 1080, 30);
    utils.logger_init("off", "console");
    utils.set_venc_enabled(false);
    utils.set_jenc_enabled(true);

    REQUIRE(utils.reconfigure_vin_pipeline(2560, 1440, 50));

    const std::vector<std::string> expected {
        "unlink",
        "vin_deinit",
        "vin_init:2560x1440@50",
        "link",
    };
    REQUIRE(fake.events == expected);
    require_dimensions(utils, 2560, 1440, 50);
    require_encoder_state(fake, 1920, 1080, 30);
    REQUIRE_FALSE(utils.main_venc_reconfiguring());
}

TEST_CASE("VIN pipeline reconfigure restores state after a late link failure",
          "[ax-utils][reconfigure]")
{
    FakeBackend fake;
    fake.reset_encoder(1920, 1080, 30);
    fake.link_results = {
        static_cast<AX_S32>(AX_ERR_LINK_UNKNOWN),
        static_cast<AX_S32>(AX_SUCCESS),
    };
    BackendScope backend_scope(fake);

    AxeraUtils utils(1920, 1080, 30);
    utils.logger_init("off", "console");
    utils.set_venc_enabled(true);
    utils.set_jenc_enabled(false);

    REQUIRE(utils.venc_start_recv());
    fake.events.clear();

    REQUIRE_FALSE(utils.reconfigure_vin_pipeline(1280, 720, 60));

    const std::vector<std::string> expected {
        "venc_stop",
        "unlink",
        "vin_deinit",
        "venc_reset",
        "venc_get_attr",
        "venc_set_attr:1280x720",
        "venc_get_rc",
        "venc_set_rc:60",
        "vin_init:1280x720@60",
        "link",
        "vin_deinit",
        "venc_get_rc",
        "venc_set_rc:30",
        "venc_reset",
        "venc_get_attr",
        "venc_set_attr:1920x1080",
        "vin_init:1920x1080@30",
        "link",
        "venc_start",
    };
    REQUIRE(fake.events == expected);
    require_dimensions(utils, 1920, 1080, 30);
    require_encoder_state(fake, 1920, 1080, 30);
    REQUIRE_FALSE(utils.main_venc_reconfiguring());

    fake.events.clear();
    REQUIRE(utils.venc_stop_recv());
    REQUIRE(fake.events == std::vector<std::string> {"venc_stop"});
}
