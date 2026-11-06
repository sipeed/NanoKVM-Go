#pragma once

#include <cstddef>
#include <cstdint>
#include <iosfwd>
#include <string>
#include <string_view>

struct TimelapseIndexEntry {
    uint64_t frame {0};
    int64_t pts {0};
    int64_t duration {0};
    std::string captured_at;
    std::string capture_id;
    int width {0};
    int height {0};
    size_t h264_bytes {0};
    uint64_t ts_offset {0};
    uint32_t segment_id {0};
};

struct TimelapseIndexPosition {
    uint64_t frame {0};
    int64_t pts {0};
};

bool read_timelapse_next_position(std::istream& input,
                                  TimelapseIndexPosition& position);
std::string serialize_timelapse_index_entry(const TimelapseIndexEntry& entry);
std::string timelapse_local_date_from_captured_at(std::string_view captured_at);
