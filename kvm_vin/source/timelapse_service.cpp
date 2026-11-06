#include <cerrno>
#include <chrono>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <limits>
#include <system_error>
#include <utility>
#include <vector>

#include "timelapse_service.hpp"

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#include "timelapse_commit.hpp"
#include "timelapse_index.hpp"
#include "timelapse_ts_muxer.hpp"

namespace {

constexpr auto kSkippedTimelapseFrameLogInterval = std::chrono::minutes(1);

bool should_log_skipped_timelapse_frame(
    std::chrono::steady_clock::time_point& last_logged)
{
    const auto now = std::chrono::steady_clock::now();
    if (last_logged != std::chrono::steady_clock::time_point {}
        && now - last_logged < kSkippedTimelapseFrameLogInterval) {
        return false;
    }
    last_logged = now;
    return true;
}

bool ensure_dir(const std::filesystem::path& dir, ILogger& logger)
{
    if (dir.empty()) {
        logger->error("[timelapse] output path is empty");
        return false;
    }

    std::error_code ec;
    if (std::filesystem::exists(dir, ec)) {
        if (ec) {
            logger->error("[timelapse] failed to check {}: {}",
                          dir.string(),
                          ec.message());
            return false;
        }
        if (!std::filesystem::is_directory(dir, ec)) {
            logger->error("[timelapse] output path is not a directory: {}",
                          dir.string());
            return false;
        }
        return true;
    }

    (void)std::filesystem::create_directories(dir, ec);
    if (ec) {
        logger->error(
            "[timelapse] failed to create {}: {}", dir.string(), ec.message());
        return false;
    }

    return true;
}

bool write_all_fd(int fd, const char* data, size_t size, std::string& error)
{
    size_t written = 0;
    while (written < size) {
        const ssize_t ret = ::write(fd, data + written, size - written);
        if (ret < 0) {
            if (errno == EINTR || errno == EAGAIN || errno == EWOULDBLOCK) {
                continue;
            }
            error = std::strerror(errno);
            return false;
        }
        if (ret == 0) {
            error = "write returned zero";
            return false;
        }
        written += static_cast<size_t>(ret);
    }
    return true;
}

bool truncate_file_to(const std::filesystem::path& path,
                      uint64_t size,
                      ILogger& logger)
{
    const int fd = ::open(path.c_str(), O_WRONLY | O_CLOEXEC);
    if (fd < 0) {
        if (errno == ENOENT && size == 0) {
            return true;
        }
        logger->error("[timelapse] failed to open {} for rollback: {}",
                      path.string(),
                      std::strerror(errno));
        return false;
    }

    bool ok = true;
    if (::ftruncate(fd, static_cast<off_t>(size)) != 0) {
        logger->error("[timelapse] failed to truncate {} to {}: {}",
                      path.string(),
                      size,
                      std::strerror(errno));
        ok = false;
    } else if (::fdatasync(fd) != 0) {
        logger->error("[timelapse] failed to sync truncated {}: {}",
                      path.string(),
                      std::strerror(errno));
        ok = false;
    }

    (void)::close(fd);
    return ok;
}

void fsync_parent_directory(const std::filesystem::path& path, ILogger& logger)
{
    const auto parent = path.parent_path();
    if (parent.empty()) {
        return;
    }

    const int dir_fd =
        ::open(parent.c_str(), O_RDONLY | O_DIRECTORY | O_CLOEXEC);
    if (dir_fd < 0) {
        logger->warn(
            "[timelapse] failed to open stable parent dir for fsync: {} "
            "error={}",
            parent.string(),
            std::strerror(errno));
        return;
    }
    if (::fsync(dir_fd) != 0) {
        logger->warn(
            "[timelapse] failed to fsync stable parent dir: {} error={}",
            parent.string(),
            std::strerror(errno));
    }
    (void)::close(dir_fd);
}

bool update_stable_offset(const std::filesystem::path& stable_path,
                          uint64_t stable_offset,
                          ILogger& logger)
{
    std::filesystem::path tmp_path = stable_path;
    tmp_path += ".tmp";

    const int fd = ::open(tmp_path.c_str(),
                          O_CREAT | O_TRUNC | O_WRONLY | O_CLOEXEC,
                          S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH);
    if (fd < 0) {
        logger->error(
            "[timelapse] failed to open stable temp file: {} error={}",
            tmp_path.string(),
            std::strerror(errno));
        return false;
    }

    const std::string content = std::to_string(stable_offset) + "\n";
    std::string write_error;
    bool ok = write_all_fd(fd, content.data(), content.size(), write_error);
    if (!ok) {
        logger->error(
            "[timelapse] failed to write stable temp file: {} error={}",
            tmp_path.string(),
            write_error);
    } else if (::fsync(fd) != 0) {
        logger->error(
            "[timelapse] failed to sync stable temp file: {} error={}",
            tmp_path.string(),
            std::strerror(errno));
        ok = false;
    }

    if (::close(fd) != 0) {
        logger->error(
            "[timelapse] failed to close stable temp file: {} error={}",
            tmp_path.string(),
            std::strerror(errno));
        ok = false;
    }

    if (!ok) {
        (void)::unlink(tmp_path.c_str());
        return false;
    }

    if (::rename(tmp_path.c_str(), stable_path.c_str()) != 0) {
        logger->error(
            "[timelapse] failed to replace stable file: {} -> {} error={}",
            tmp_path.string(),
            stable_path.string(),
            std::strerror(errno));
        (void)::unlink(tmp_path.c_str());
        return false;
    }

    fsync_parent_directory(stable_path, logger);
    return true;
}

bool append_index_entry(const std::filesystem::path& index_path,
                        uint64_t frame_index,
                        int64_t pts,
                        const std::string& captured_at,
                        const std::string& capture_id,
                        int width,
                        int height,
                        size_t h264_size,
                        uint64_t ts_offset,
                        uint32_t segment_id,
                        uint64_t& previous_index_size,
                        ILogger& logger)
{
    previous_index_size = 0;

    const int fd = ::open(index_path.c_str(),
                          O_CREAT | O_RDWR | O_CLOEXEC,
                          S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH);
    if (fd < 0) {
        logger->error("[timelapse] failed to open index: {} error={}",
                      index_path.string(),
                      std::strerror(errno));
        return false;
    }

    struct stat st {};
    if (::fstat(fd, &st) != 0 || st.st_size < 0) {
        logger->error("[timelapse] failed to stat index: {} error={}",
                      index_path.string(),
                      std::strerror(errno));
        (void)::close(fd);
        return false;
    }
    previous_index_size = static_cast<uint64_t>(st.st_size);

    if (::lseek(fd, 0, SEEK_END) < 0) {
        logger->error("[timelapse] failed to seek index: {} error={}",
                      index_path.string(),
                      std::strerror(errno));
        (void)::close(fd);
        return false;
    }

    const std::string payload =
        serialize_timelapse_index_entry(TimelapseIndexEntry {
            .frame = frame_index,
            .pts = pts,
            .duration = TimelapseTsMuxer::FRAME_DURATION_90K,
            .captured_at = captured_at,
            .capture_id = capture_id,
            .width = width,
            .height = height,
            .h264_bytes = h264_size,
            .ts_offset = ts_offset,
            .segment_id = segment_id,
        });
    std::string write_error;
    bool ok = write_all_fd(fd, payload.data(), payload.size(), write_error);
    if (!ok) {
        logger->error("[timelapse] failed to write index: {} error={}",
                      index_path.string(),
                      write_error);
    } else if (::fsync(fd) != 0) {
        logger->error("[timelapse] failed to sync index: {} error={}",
                      index_path.string(),
                      std::strerror(errno));
        ok = false;
    }

    if (!ok) {
        (void)::ftruncate(fd, static_cast<off_t>(previous_index_size));
        (void)::lseek(fd, static_cast<off_t>(previous_index_size), SEEK_SET);
        (void)::close(fd);
        return false;
    }

    if (::close(fd) != 0) {
        logger->error("[timelapse] failed to close index: {} error={}",
                      index_path.string(),
                      std::strerror(errno));
        (void)truncate_file_to(index_path, previous_index_size, logger);
        return false;
    }

    return true;
}

}  // namespace

TimelapseService::TimelapseService(AxeraUtils& axera_utils,
                                   TimelapseConfig config,
                                   ILogger& logger)
    : _axera_utils(axera_utils)
    , _config(std::move(config))
    , _logger(logger)
{}

TimelapseService::~TimelapseService() { stop(); }

bool TimelapseService::start()
{
    std::lock_guard lock(_mutex);
    if (!_config.enable || _started) {
        return true;
    }

    _started = true;
    _logger->info("[timelapse] enabled: output={}",
                  _config.output_path.empty()
                      ? TimelapseConfig::DEFAULT_OUTPUT_PATH
                      : _config.output_path);
    return true;
}

void TimelapseService::stop()
{
    std::lock_guard lock(_mutex);
    _started = false;
}

bool TimelapseService::append_snapshot_frame(
    const AxeraUtils::SnapshotResult& result,
    const std::string& captured_at,
    const std::string& capture_id)
{
    std::lock_guard lock(_mutex);
    if (!_config.enable || !_started) {
        return true;
    }

    if (capture_id.empty()) {
        _logger->error("[timelapse] skip frame without capture_id");
        return false;
    }

    if (result.data.empty()) {
        _logger->error("[timelapse] skip empty JPEG");
        return false;
    }

    std::filesystem::path output_dir = _config.output_path.empty()
        ? std::filesystem::path(TimelapseConfig::DEFAULT_OUTPUT_PATH)
        : std::filesystem::path(_config.output_path);
    const std::string output_date =
        timelapse_local_date_from_captured_at(captured_at);
    if (output_date.empty()) {
        _logger->error("[timelapse] invalid captured_at: {}", captured_at);
        return false;
    }

    if (_encode_sequence
        > static_cast<uint64_t>(std::numeric_limits<int64_t>::max()
                                / TimelapseTsMuxer::FRAME_DURATION_90K)) {
        _logger->error("[timelapse] encoder input sequence is too large: {}",
                       _encode_sequence);
        return false;
    }
    // Persisted PTS is read from the index in the locked commit below. This PTS
    // only keeps the hardware encoder input sequence monotonic and must not
    // depend on a sidecar read before locking.
    const uint64_t encode_pts = _encode_sequence
        * static_cast<uint64_t>(TimelapseTsMuxer::FRAME_DURATION_90K);
    ++_encode_sequence;
    int width = 0;
    int height = 0;
    int y_stride = 0;
    int uv_stride = 0;
    size_t required_size = 0;
    int ret = _axera_utils.vdec_get_jpeg_info(result.data.data(),
                                              result.data.size(),
                                              width,
                                              height,
                                              y_stride,
                                              uv_stride,
                                              required_size);
    if (ret != 0) {
        _logger->error(
            "[timelapse] failed to get JPEG info: capture_id={} ret={}",
            capture_id,
            ret);
        return false;
    }

    std::vector<uint8_t> nv12(required_size);
    size_t written_size = 0;
    ret = _axera_utils.vdec_decode_jpeg(result.data.data(),
                                        result.data.size(),
                                        nv12.data(),
                                        nv12.size(),
                                        width,
                                        height,
                                        y_stride,
                                        uv_stride,
                                        written_size);
    if (ret != 0) {
        _logger->error(
            "[timelapse] failed to decode JPEG: capture_id={} ret={}",
            capture_id,
            ret);
        return false;
    }

    std::vector<uint8_t> h264;
    ret = _axera_utils.timelapse_encode_nv12(nv12.data(),
                                             written_size,
                                             width,
                                             height,
                                             y_stride,
                                             uv_stride,
                                             encode_pts,
                                             2000,
                                             h264);
    if (ret != 0 || h264.empty()) {
        _logger->error(
            "[timelapse] failed to encode H264: capture_id={} ret={}",
            capture_id,
            ret);
        return false;
    }

    TimelapseCommitLock commit_lock;
    std::string lock_error;
    switch (commit_lock.try_lock(lock_error)) {
    case TimelapseCommitLockResult::Acquired:
        break;
    case TimelapseCommitLockResult::Busy:
        if (should_log_skipped_timelapse_frame(_last_lock_contention_log)) {
            _logger->warn(
                "[timelapse] skip encoded frame: commit lock busy "
                "capture_id={}",
                capture_id);
        }
        return true;
    case TimelapseCommitLockResult::Error:
        _logger->error("[timelapse] failed to acquire commit lock: {}",
                       lock_error);
        return false;
    }

    const std::string current_date = timelapse_current_local_date();
    if (!timelapse_can_commit_date(output_date, current_date)) {
        if (should_log_skipped_timelapse_frame(_last_stale_day_log)) {
            _logger->warn(
                "[timelapse] skip encoded frame for non-current date: "
                "captured_date={} current_date={} capture_id={}",
                output_date,
                current_date.empty() ? "unavailable" : current_date,
                capture_id);
        }
        return true;
    }

    const std::filesystem::path output_date_dir = output_dir / output_date;
    if (!ensure_dir(output_date_dir, _logger)) {
        return false;
    }

    const std::filesystem::path output_path =
        output_date_dir / (output_date + ".ts");
    std::filesystem::path index_path = output_path;
    index_path += ".idx";
    std::filesystem::path stable_path = output_path;
    stable_path += ".stable";

    std::ifstream index_input(index_path);
    TimelapseIndexPosition next_position;
    if (!read_timelapse_next_position(index_input, next_position)
        || next_position.pts > std::numeric_limits<int64_t>::max()
                - TimelapseTsMuxer::FRAME_DURATION_90K) {
        _logger->error("[timelapse] invalid index timing: {}",
                       index_path.string());
        return false;
    }
    const uint64_t frame_index = next_position.frame;
    const int64_t pts = next_position.pts;

    std::string mux_error;
    TimelapseTsMuxer::AppendResult mux_result;
    if (!TimelapseTsMuxer::append_h264_frame(output_path.string(),
                                             h264.data(),
                                             h264.size(),
                                             width,
                                             height,
                                             pts,
                                             mux_result,
                                             mux_error)) {
        _logger->error("[timelapse] failed to append TS: {} error={}",
                       output_path.string(),
                       mux_error);
        return false;
    }

    uint64_t previous_index_size = 0;
    if (!append_index_entry(index_path,
                            frame_index,
                            pts,
                            captured_at,
                            capture_id,
                            width,
                            height,
                            h264.size(),
                            mux_result.ts_offset,
                            mux_result.segment_id,
                            previous_index_size,
                            _logger)) {
        (void)truncate_file_to(output_path, mux_result.ts_offset, _logger);
        return false;
    }

    if (!update_stable_offset(stable_path, mux_result.stable_offset, _logger)) {
        (void)truncate_file_to(index_path, previous_index_size, _logger);
        (void)truncate_file_to(output_path, mux_result.ts_offset, _logger);
        return false;
    }

    _logger->debug(
        "[timelapse] appended frame={} captured_at={} capture_id={} -> {} "
        "({}x{}, {} bytes, ts_offset={}, stable_offset={}, segment_id={})",
        frame_index,
        captured_at,
        capture_id,
        output_path.string(),
        width,
        height,
        h264.size(),
        mux_result.ts_offset,
        mux_result.stable_offset,
        mux_result.segment_id);
    return true;
}
