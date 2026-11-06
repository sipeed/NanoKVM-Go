#pragma once

#include <cstddef>
#include <cstdint>
#include <condition_variable>
#include <mutex>
#include <string>
#include <vector>

#include <ax_venc_comm.h>
#include <ax_venc_rc.h>
#include <logger/ilogger.hpp>
#include <nlohmann/json.hpp>
#include <sys/socket.h>
#include <sys/un.h>

class AxeraUtils final : public ILogger {
  public:
    enum class EncodeMode {
        H264,
        H265,
    };

    struct VencStream {
        AX_VENC_STREAM_T raw {};
    };

    enum class VencFrameResult {
        Ok,
        QueueEmpty,
        Reconfiguring,
        Error,
    };

    class Venc {
      public:
        virtual ~Venc() = default;
        virtual bool start(int chn) = 0;
        virtual bool stop(int chn) = 0;
        int chn {0};
        int width {0};
        int height {0};
        int fps {0};
        int fd {-1};
    };

    class Jpeg : public Venc {
      public:
        ~Jpeg() override = default;
        int quality {1};
        bool start(int chn) override;
        bool stop(int chn) override;
    };

    class H264 : public Venc {
      public:
        ~H264() override = default;
        AX_VENC_H264_CBR_T cbr {};
        AX_VENC_H264_VBR_T vbr {};
        bool start(int chn) override;
        bool stop(int chn) override;
    };

    class H265 : public Venc {
      public:
        ~H265() override = default;
        AX_VENC_H265_CBR_T cbr {};
        AX_VENC_H265_VBR_T vbr {};
        bool start(int chn) override;
        bool stop(int chn) override;
    };

    AxeraUtils(int width, int height, int fps = 30)
        : _width(width)
        , _height(height)
        , _fps(fps)
    {
        _jpeg.width = width;
        _jpeg.height = height;
        _jpeg.fps = fps;
        _h264.width = width;
        _h264.height = height;
        _h264.fps = fps;
        _h265.width = width;
        _h265.height = height;
        _h265.fps = fps;
        _timelapse_h264.width = width;
        _timelapse_h264.height = height;
        _timelapse_h264.fps = fps;
    }
    AxeraUtils() = default;
    ~AxeraUtils();

    /* Logger */
    void logger_init(const std::string& level,
                     const std::string& type,
                     const std::string& ofilepath = "") override;
    std::shared_ptr<spdlog::logger> operator->() override;

    bool init(void);

    bool get_resolution(int& width, int& height)
    {
        width = _width;
        height = _height;
        return true;
    }

    bool set_resolution(int width, int height)
    {
        bool ok = true;
        if (_venc_enabled) {
            ok = venc_set_resolution(get_venc_chn(), width, height) && ok;
        }
        if (ok) {
            _width = width;
            _height = height;
        }
        return ok;
    }

    bool get_fps(int& fps)
    {
        fps = _fps;
        return true;
    }

    bool set_fps(int fps)
    {
        bool ok = true;
        if (_venc_enabled) {
            ok = venc_set_fps(get_venc_chn(), fps, 0) && ok;
        }
        if (_jenc_enabled) {
            ok = jenc_set_fps(get_jpeg_chn(), fps, 0) && ok;
        }
        if (ok) {
            _fps = fps;
        }
        return ok;
    }

    int get_h264_chn() const { return _h264.chn; }
    int get_h265_chn() const { return _h265.chn; }
    int get_jpeg_chn() const { return _jpeg.chn; }
    int get_venc_chn() const { return _active_venc ? _active_venc->chn : _h264.chn; }

    int get_h264_fd() const { return _h264.fd; }
    int get_h265_fd() const { return _h265.fd; }
    int get_jpeg_fd() const { return _jpeg.fd; }
    int get_venc_fd()  const { return _active_venc ? _active_venc->fd  : _h264.fd; }

    bool h264_start_recv() { return _h264.start(_h264.chn); }
    bool h264_stop_recv()  { return _h264.stop(_h264.chn); }
    bool h265_start_recv() { return _h265.start(_h265.chn); }
    bool h265_stop_recv()  { return _h265.stop(_h265.chn); }
    bool venc_start_recv();
    bool venc_stop_recv();
    bool main_venc_reconfiguring() const;

    void set_venc_enabled(bool enabled) { _venc_enabled = enabled; }
    bool venc_enabled() const { return _venc_enabled; }
    void set_jenc_enabled(bool enabled) { _jenc_enabled = enabled; }
    bool jenc_enabled() const { return _jenc_enabled; }

    void set_encode_mode(EncodeMode mode);
    EncodeMode encode_mode() const { return _encode_mode; }

    bool pool_init(int width, int height);

    bool vin_init(void);
    bool vin_deinit(void);

    bool ivps_init(void);
    bool ivps_deinit(void);
    bool ivps_set_crop(bool enable, int x, int y, int w, int h);

    bool venc_init(void);
    bool venc_deinit(void);
    bool venc_set_resolution(int chn, int width, int height);
    bool venc_set_fps(int chn, float src_fps, float dst_fps = 0);
    bool venc_set_rc_mode(int chn, AX_VENC_RC_MODE_E mode);
    bool venc_set_bitrate(int chn, unsigned int bitrate);
    bool venc_set_gop(int chn, unsigned int gop);
    VencFrameResult venc_get_frame(VencStream& stream, int timeout_ms = 500);
    bool venc_release_frame(VencStream& stream);
    bool venc_request_idr(int chn, bool instant = false);
    bool reconfigure_vin_pipeline(int width, int height, int fps);

    bool timelapse_venc_init(unsigned int qp);
    bool timelapse_venc_deinit(void);
    int timelapse_encode_nv12(const uint8_t* nv12,
                              size_t size,
                              int width,
                              int height,
                              int y_stride,
                              int uv_stride,
                              uint64_t pts,
                              int timeout_ms,
                              std::vector<uint8_t>& h264);

    bool jenc_init(void);
    bool jenc_deinit(void);
    bool jenc_set_fps(int chn, float src_fps, float dst_fps);
    bool jenc_set_quality(int chn, int quality);
    bool jenc_reset_chn(int chn);
    int jenc_get_frame(std::vector<uint8_t>& data,
                       int timeout_ms = 500,
                       int skip_frames = 0);
    bool jenc_set_crop(bool enable, int x, int y, int w, int h);

    struct SnapshotResult {
        std::vector<uint8_t> data;
        int width {0};
        int height {0};
    };

    int jenc_snapshot(int x,
                      int y,
                      int w,
                      int h,
                      int quality,
                      int timeout_ms,
                      SnapshotResult& result);

    bool vdec_init(void);
    bool vdec_deinit(void);
    int vdec_get_jpeg_info(const uint8_t* data,
                           size_t size,
                           int& width,
                           int& height,
                           int& y_stride,
                           int& uv_stride,
                           size_t& required_size);
    int vdec_decode_jpeg(const uint8_t* data,
                         size_t size,
                         uint8_t* output,
                         size_t output_size,
                         int& width,
                         int& height,
                         int& y_stride,
                         int& uv_stride,
                         size_t& written_size);
    int vdec_decode_jpeg_file(const std::string& path,
                              uint8_t* output,
                              size_t output_size,
                              int& width,
                              int& height,
                              int& y_stride,
                              int& uv_stride,
                              size_t& written_size);

    bool link_init(void);
    bool link_deinit(void);

    bool dump_yuyv_to_file(const std::string& filepath, int timeout_ms = 0);

  private:
    bool _init {false};
    std::shared_ptr<ILogger> _logger {nullptr};
    int _mipi_dev {-1};
    int _ivps_grp {-1};
    void *_isp_lib {nullptr};
    int _width {1920};
    int _height {1080};
    int _fps {30};
    int _ivps_pool_id {-1};

    bool _venc_enabled {true};
    bool _jenc_enabled {false};

    Jpeg _jpeg {};
    H264 _h264 {};
    H265 _h265 {};
    H264 _timelapse_h264 {};
    EncodeMode _encode_mode {EncodeMode::H264};
    Venc* _active_venc {&_h264};
    std::mutex _snapshot_mutex;
    std::mutex _link_mutex;
    std::unique_lock<std::mutex> _jenc_link_snapshot_guard;
    std::mutex _vin_pipeline_mutex;
    mutable std::mutex _main_venc_mutex;
    std::condition_variable _main_venc_cv;
    bool _main_venc_recv_started {false};
    bool _main_venc_reconfiguring {false};
    int _main_venc_stream_users {0};

    bool _mipi_init(void);
    bool _pool_deinit(void);
    bool _mipi_deinit(void);
    bool _set_pinmux(void);
};
