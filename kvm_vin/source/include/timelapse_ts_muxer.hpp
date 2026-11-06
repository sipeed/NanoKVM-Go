#pragma once

#include <cstddef>
#include <cstdint>
#include <string>

class TimelapseTsMuxer {
  public:
    static constexpr unsigned int FRAME_RATE = 5;
    static constexpr int64_t FRAME_DURATION_90K = 90000 / FRAME_RATE;

    struct AppendResult {
        uint64_t ts_offset {0};
        uint64_t stable_offset {0};
        uint32_t segment_id {0};
    };

    static bool append_h264_frame(const std::string& output_path,
                                  const uint8_t* h264,
                                  size_t h264_size,
                                  int width,
                                  int height,
                                  int64_t pts,
                                  AppendResult& result,
                                  std::string& error);
};
