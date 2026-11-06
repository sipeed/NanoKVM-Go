#include "timelapse_ts_muxer.hpp"

#include <cerrno>
#include <climits>
#include <cstring>
#include <limits>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include <dlfcn.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

extern "C" {
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libavutil/error.h>
#include <libavutil/mem.h>
#include <libavutil/pixfmt.h>
}

namespace {

constexpr AVRational kTsTimeBase {1, 90000};
constexpr int kAvioBufferSize = 32768;
constexpr size_t kTsPacketSize = 188;
constexpr uint32_t kSingleSegmentId = 0;

class FfmpegApi {
  public:
    FfmpegApi()
        : library_(::dlopen("libavformat.so." AV_STRINGIFY(LIBAVFORMAT_VERSION_MAJOR),
                            RTLD_NOW | RTLD_LOCAL),
                   &::dlclose)
    {
        if (!library_) {
            const char* error = ::dlerror();
            throw std::runtime_error(std::string("load system FFmpeg failed: ") +
                                     (error != nullptr ? error : "unknown loader error"));
        }

        // dlsym also searches libavformat's libavcodec and libavutil dependencies.
#define LOAD_FFMPEG(name) name = load<decltype(name)>(#name)
        LOAD_FFMPEG(av_strerror);
        LOAD_FFMPEG(avformat_alloc_output_context2);
        LOAD_FFMPEG(avformat_free_context);
        LOAD_FFMPEG(avformat_new_stream);
        LOAD_FFMPEG(avformat_write_header);
        LOAD_FFMPEG(av_interleaved_write_frame);
        LOAD_FFMPEG(av_write_trailer);
        LOAD_FFMPEG(av_malloc);
        LOAD_FFMPEG(av_freep);
        LOAD_FFMPEG(avio_alloc_context);
        LOAD_FFMPEG(avio_context_free);
        LOAD_FFMPEG(avio_flush);
        LOAD_FFMPEG(av_packet_alloc);
        LOAD_FFMPEG(av_packet_free);
        LOAD_FFMPEG(av_new_packet);
#undef LOAD_FFMPEG
    }

    std::string error_string(int ret) const
    {
        char buffer[AV_ERROR_MAX_STRING_SIZE] {};
        av_strerror(ret, buffer, sizeof(buffer));
        return buffer;
    }

    decltype(&::av_strerror) av_strerror {};
    decltype(&::avformat_alloc_output_context2) avformat_alloc_output_context2 {};
    decltype(&::avformat_free_context) avformat_free_context {};
    decltype(&::avformat_new_stream) avformat_new_stream {};
    decltype(&::avformat_write_header) avformat_write_header {};
    decltype(&::av_interleaved_write_frame) av_interleaved_write_frame {};
    decltype(&::av_write_trailer) av_write_trailer {};
    decltype(&::av_malloc) av_malloc {};
    decltype(&::av_freep) av_freep {};
    decltype(&::avio_alloc_context) avio_alloc_context {};
    decltype(&::avio_context_free) avio_context_free {};
    decltype(&::avio_flush) avio_flush {};
    decltype(&::av_packet_alloc) av_packet_alloc {};
    decltype(&::av_packet_free) av_packet_free {};
    decltype(&::av_new_packet) av_new_packet {};

  private:
    template <typename Function>
    Function load(const char* name)
    {
        ::dlerror();
        void* symbol = ::dlsym(library_.get(), name);
        const char* error = ::dlerror();
        if (error != nullptr || symbol == nullptr) {
            throw std::runtime_error(std::string("resolve FFmpeg ") + name +
                                     " failed: " +
                                     (error != nullptr ? error : "symbol is null"));
        }
        return reinterpret_cast<Function>(symbol);
    }

    std::unique_ptr<void, int (*)(void*)> library_;
};

struct AppendIoContext {
    int fd {-1};
    uint64_t committed_offset {0};
    std::vector<uint8_t> pending;
    std::string error;
};

bool set_error(std::string& error, const std::string& message)
{
    error = message;
    return false;
}

bool set_context_error(AppendIoContext& ctx, const std::string& message)
{
    if (ctx.error.empty()) {
        ctx.error = message;
    }
    return false;
}

bool write_all_packets(AppendIoContext& ctx, const uint8_t* data, size_t size)
{
    if (size == 0) {
        return true;
    }
    if (ctx.fd < 0 || data == nullptr || (size % kTsPacketSize) != 0) {
        return set_context_error(ctx, "invalid TS packet batch");
    }
    for (size_t offset = 0; offset < size; offset += kTsPacketSize) {
        if (data[offset] != 0x47) {
            return set_context_error(ctx, "muxer output has an invalid TS sync byte");
        }
    }

    const uint64_t batch_start = ctx.committed_offset;
    size_t written = 0;
    while (written < size) {
        const ssize_t ret = ::write(ctx.fd, data + written, size - written);
        if (ret < 0) {
            if (errno == EINTR || errno == EAGAIN || errno == EWOULDBLOCK) {
                continue;
            }

            const uint64_t rollback_offset =
                batch_start + (static_cast<uint64_t>(written) / kTsPacketSize) *
                                  static_cast<uint64_t>(kTsPacketSize);
            (void)::ftruncate(ctx.fd, static_cast<off_t>(rollback_offset));
            (void)::lseek(ctx.fd, static_cast<off_t>(rollback_offset), SEEK_SET);
            ctx.committed_offset = rollback_offset;
            return set_context_error(ctx,
                                     "write TS packet batch failed: " +
                                         std::string(std::strerror(errno)));
        }
        if (ret == 0) {
            const uint64_t rollback_offset =
                batch_start + (static_cast<uint64_t>(written) / kTsPacketSize) *
                                  static_cast<uint64_t>(kTsPacketSize);
            (void)::ftruncate(ctx.fd, static_cast<off_t>(rollback_offset));
            (void)::lseek(ctx.fd, static_cast<off_t>(rollback_offset), SEEK_SET);
            ctx.committed_offset = rollback_offset;
            return set_context_error(ctx, "write TS packet batch returned zero");
        }

        written += static_cast<size_t>(ret);
    }

    ctx.committed_offset = batch_start + static_cast<uint64_t>(size);
    return true;
}

bool accept_muxer_bytes(AppendIoContext& ctx, const uint8_t* buf, size_t size)
{
    if (size == 0) {
        return true;
    }
    if (buf == nullptr) {
        return set_context_error(ctx, "invalid muxer output buffer");
    }

    if (ctx.pending.empty()) {
        const size_t complete_size = size - (size % kTsPacketSize);
        if (complete_size > 0 && !write_all_packets(ctx, buf, complete_size)) {
            return false;
        }
        const size_t tail_size = size - complete_size;
        if (tail_size > 0) {
            ctx.pending.insert(ctx.pending.end(), buf + complete_size, buf + size);
        }
        return true;
    }

    ctx.pending.insert(ctx.pending.end(), buf, buf + size);
    const size_t complete_size = ctx.pending.size() - (ctx.pending.size() % kTsPacketSize);
    if (complete_size == 0) {
        return true;
    }

    if (!write_all_packets(ctx, ctx.pending.data(), complete_size)) {
        return false;
    }

    ctx.pending.erase(ctx.pending.begin(),
                      ctx.pending.begin() + static_cast<std::ptrdiff_t>(complete_size));
    return true;
}

int write_packet(void* opaque, const uint8_t* buf, int buf_size)
{
    auto* ctx = static_cast<AppendIoContext*>(opaque);
    if (ctx == nullptr || buf_size < 0) {
        return AVERROR(EIO);
    }
    if (!accept_muxer_bytes(*ctx, buf, static_cast<size_t>(buf_size))) {
        return AVERROR(EIO);
    }

    return buf_size;
}

bool open_aligned_output(const std::string& output_path,
                         AppendIoContext& io_ctx,
                         uint64_t& start_offset,
                         std::string& error)
{
    const int fd = ::open(output_path.c_str(),
                          O_CREAT | O_WRONLY | O_CLOEXEC,
                          S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH);
    if (fd < 0) {
        return set_error(error,
                         "failed to open output TS for append: " +
                             std::string(std::strerror(errno)));
    }

    struct stat st {};
    if (::fstat(fd, &st) != 0) {
        const int saved_errno = errno;
        (void)::close(fd);
        return set_error(error,
                         "failed to stat output TS: " +
                             std::string(std::strerror(saved_errno)));
    }
    if (st.st_size < 0) {
        (void)::close(fd);
        return set_error(error, "invalid output TS size");
    }

    const auto current_size = static_cast<uint64_t>(st.st_size);
    const uint64_t aligned_size =
        current_size - (current_size % static_cast<uint64_t>(kTsPacketSize));
    if (aligned_size != current_size &&
        ::ftruncate(fd, static_cast<off_t>(aligned_size)) != 0) {
        const int saved_errno = errno;
        (void)::close(fd);
        return set_error(error,
                         "failed to truncate output TS to packet boundary: " +
                             std::string(std::strerror(saved_errno)));
    }

    if (::lseek(fd, static_cast<off_t>(aligned_size), SEEK_SET) < 0) {
        const int saved_errno = errno;
        (void)::close(fd);
        return set_error(error,
                         "failed to seek output TS: " +
                             std::string(std::strerror(saved_errno)));
    }

    io_ctx.fd = fd;
    io_ctx.committed_offset = aligned_size;
    start_offset = aligned_size;
    return true;
}

void rollback_output(AppendIoContext& io_ctx, uint64_t offset)
{
    if (io_ctx.fd < 0) {
        return;
    }
    (void)::ftruncate(io_ctx.fd, static_cast<off_t>(offset));
    (void)::lseek(io_ctx.fd, static_cast<off_t>(offset), SEEK_SET);
    io_ctx.committed_offset = offset;
    io_ctx.pending.clear();
}

bool flush_output(AppendIoContext& io_ctx, uint64_t start_offset, std::string& error)
{
    if (!io_ctx.error.empty()) {
        rollback_output(io_ctx, start_offset);
        return set_error(error, io_ctx.error);
    }
    if (!io_ctx.pending.empty()) {
        rollback_output(io_ctx, start_offset);
        return set_error(error, "muxer output ended with a partial TS packet");
    }
    if ((io_ctx.committed_offset % static_cast<uint64_t>(kTsPacketSize)) != 0) {
        rollback_output(io_ctx, start_offset);
        return set_error(error, "output TS is not packet aligned after append");
    }
    if (::fdatasync(io_ctx.fd) != 0) {
        const int saved_errno = errno;
        rollback_output(io_ctx, start_offset);
        return set_error(error,
                         "failed to sync output TS: " +
                             std::string(std::strerror(saved_errno)));
    }
    return true;
}

}  // namespace

bool TimelapseTsMuxer::append_h264_frame(const std::string& output_path,
                                         const uint8_t* h264,
                                         size_t h264_size,
                                         int width,
                                         int height,
                                         int64_t pts,
                                         AppendResult& result,
                                         std::string& error)
{
    error.clear();
    result = {};

    if (output_path.empty()) {
        return set_error(error, "output path is empty");
    }
    if (h264 == nullptr || h264_size == 0 || h264_size > static_cast<size_t>(INT_MAX)) {
        return set_error(error, "invalid H264 frame buffer");
    }
    if (width <= 0 || height <= 0) {
        return set_error(error, "invalid video dimensions");
    }
    if (pts < 0
        || pts > std::numeric_limits<int64_t>::max() - FRAME_DURATION_90K) {
        return set_error(error, "frame PTS is out of range");
    }

    const FfmpegApi* ffmpeg = nullptr;
    try {
        // Thread-safe first use; keep the libraries loaded across recording frames.
        // Failed initialization is retried on the next call, before touching the TS.
        static const FfmpegApi api;
        ffmpeg = &api;
    } catch (const std::exception& e) {
        return set_error(error, e.what());
    }

    AppendIoContext io_ctx;
    uint64_t start_offset = 0;
    if (!open_aligned_output(output_path, io_ctx, start_offset, error)) {
        return false;
    }

    AVFormatContext* format_ctx = nullptr;
    AVIOContext* avio_ctx = nullptr;
    AVPacket* packet = nullptr;

    auto cleanup = [&]() {
        if (packet != nullptr) {
            ffmpeg->av_packet_free(&packet);
        }
        if (avio_ctx != nullptr) {
            ffmpeg->av_freep(&avio_ctx->buffer);
            ffmpeg->avio_context_free(&avio_ctx);
        }
        if (format_ctx != nullptr) {
            format_ctx->pb = nullptr;
            ffmpeg->avformat_free_context(format_ctx);
        }
        if (io_ctx.fd >= 0) {
            (void)::close(io_ctx.fd);
            io_ctx.fd = -1;
        }
    };

    int ret = ffmpeg->avformat_alloc_output_context2(&format_ctx,
                                             nullptr,
                                             "mpegts",
                                             output_path.c_str());
    if (ret < 0 || format_ctx == nullptr) {
        cleanup();
        return set_error(error, "avformat_alloc_output_context2 failed: " +
                                ffmpeg->error_string(ret));
    }

    auto* avio_buffer = static_cast<unsigned char*>(ffmpeg->av_malloc(kAvioBufferSize));
    if (avio_buffer == nullptr) {
        cleanup();
        return set_error(error, "av_malloc AVIO buffer failed");
    }

    avio_ctx = ffmpeg->avio_alloc_context(avio_buffer,
                                  kAvioBufferSize,
                                  1,
                                  &io_ctx,
                                  nullptr,
                                  write_packet,
                                  nullptr);
    if (avio_ctx == nullptr) {
        ffmpeg->av_freep(&avio_buffer);
        cleanup();
        return set_error(error, "avio_alloc_context failed");
    }

    format_ctx->pb = avio_ctx;
    format_ctx->flags |= AVFMT_FLAG_CUSTOM_IO;

    AVStream* stream = ffmpeg->avformat_new_stream(format_ctx, nullptr);
    if (stream == nullptr) {
        cleanup();
        return set_error(error, "avformat_new_stream failed");
    }

    stream->time_base = kTsTimeBase;
    stream->avg_frame_rate = AVRational {static_cast<int>(FRAME_RATE), 1};
    stream->codecpar->codec_type = AVMEDIA_TYPE_VIDEO;
    stream->codecpar->codec_id = AV_CODEC_ID_H264;
    stream->codecpar->codec_tag = 0;
    stream->codecpar->format = AV_PIX_FMT_YUV420P;
    stream->codecpar->width = width;
    stream->codecpar->height = height;

    ret = ffmpeg->avformat_write_header(format_ctx, nullptr);
    if (ret < 0) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error, "avformat_write_header failed: " + ffmpeg->error_string(ret));
    }

    packet = ffmpeg->av_packet_alloc();
    if (packet == nullptr) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error, "av_packet_alloc failed");
    }

    ret = ffmpeg->av_new_packet(packet, static_cast<int>(h264_size));
    if (ret < 0) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error, "av_new_packet failed: " + ffmpeg->error_string(ret));
    }
    std::memcpy(packet->data, h264, h264_size);

    packet->stream_index = stream->index;
    packet->pts = pts;
    packet->dts = packet->pts;
    packet->duration = FRAME_DURATION_90K;
    packet->flags |= AV_PKT_FLAG_KEY;

    ret = ffmpeg->av_interleaved_write_frame(format_ctx, packet);
    if (ret < 0) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error,
                         "av_interleaved_write_frame failed: " +
                             (!io_ctx.error.empty() ? io_ctx.error : ffmpeg->error_string(ret)));
    }

    ret = ffmpeg->av_write_trailer(format_ctx);
    if (ret < 0) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error,
                         "av_write_trailer failed: " +
                             (!io_ctx.error.empty() ? io_ctx.error : ffmpeg->error_string(ret)));
    }

    ffmpeg->avio_flush(avio_ctx);
    if (avio_ctx->error < 0) {
        rollback_output(io_ctx, start_offset);
        cleanup();
        return set_error(error,
                         "avio_flush failed: " +
                             (!io_ctx.error.empty() ? io_ctx.error
                                                   : ffmpeg->error_string(avio_ctx->error)));
    }

    if (!flush_output(io_ctx, start_offset, error)) {
        cleanup();
        return false;
    }

    result.ts_offset = start_offset;
    result.stable_offset = io_ctx.committed_offset;
    result.segment_id = kSingleSegmentId;

    cleanup();
    return true;
}
