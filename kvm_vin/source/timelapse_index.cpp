#include <algorithm>
#include <cctype>
#include <ctime>
#include <limits>
#include <sstream>

#include "timelapse_index.hpp"

#include <nlohmann/json.hpp>

namespace {

std::string json_escape(const std::string& value)
{
    std::ostringstream escaped;
    for (const unsigned char ch : value) {
        switch (ch) {
        case '\\':
            escaped << "\\\\";
            break;
        case '"':
            escaped << "\\\"";
            break;
        case '\b':
            escaped << "\\b";
            break;
        case '\f':
            escaped << "\\f";
            break;
        case '\n':
            escaped << "\\n";
            break;
        case '\r':
            escaped << "\\r";
            break;
        case '\t':
            escaped << "\\t";
            break;
        default:
            if (ch < 0x20) {
                escaped << "\\u";
                constexpr char hex[] = "0123456789abcdef";
                escaped << '0' << '0' << hex[(ch >> 4) & 0x0F]
                        << hex[ch & 0x0F];
            } else {
                escaped << static_cast<char>(ch);
            }
            break;
        }
    }
    return escaped.str();
}

bool parse_decimal(std::string_view value,
                   std::size_t offset,
                   std::size_t length,
                   int& parsed)
{
    if (offset > value.size() || length > value.size() - offset) {
        return false;
    }

    parsed = 0;
    for (std::size_t i = 0; i < length; ++i) {
        const unsigned char ch = static_cast<unsigned char>(value[offset + i]);
        if (!std::isdigit(ch)) {
            return false;
        }
        parsed = parsed * 10 + static_cast<int>(ch - '0');
    }
    return true;
}

bool parse_captured_at_epoch(std::string_view value, std::time_t& epoch)
{
    if (value.size() < 20 || value[4] != '-' || value[7] != '-'
        || value[10] != 'T' || value[13] != ':' || value[16] != ':') {
        return false;
    }

    int year = 0;
    int month = 0;
    int day = 0;
    int hour = 0;
    int minute = 0;
    int second = 0;
    if (!parse_decimal(value, 0, 4, year) || !parse_decimal(value, 5, 2, month)
        || !parse_decimal(value, 8, 2, day)
        || !parse_decimal(value, 11, 2, hour)
        || !parse_decimal(value, 14, 2, minute)
        || !parse_decimal(value, 17, 2, second) || month < 1 || month > 12
        || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 60) {
        return false;
    }

    std::size_t timezone_pos = 19;
    if (value[timezone_pos] == '.') {
        ++timezone_pos;
        const std::size_t fraction_start = timezone_pos;
        while (
            timezone_pos < value.size()
            && std::isdigit(static_cast<unsigned char>(value[timezone_pos]))) {
            ++timezone_pos;
        }
        if (timezone_pos == fraction_start) {
            return false;
        }
    }

    int offset_seconds = 0;
    if (timezone_pos < value.size() && value[timezone_pos] == 'Z') {
        if (timezone_pos + 1 != value.size()) {
            return false;
        }
    } else if (timezone_pos < value.size()
               && (value[timezone_pos] == '+' || value[timezone_pos] == '-')) {
        const int sign = value[timezone_pos] == '+' ? 1 : -1;
        ++timezone_pos;
        int offset_hour = 0;
        int offset_minute = 0;
        if (!parse_decimal(value, timezone_pos, 2, offset_hour)) {
            return false;
        }
        timezone_pos += 2;
        if (timezone_pos < value.size() && value[timezone_pos] == ':') {
            ++timezone_pos;
        }
        if (!parse_decimal(value, timezone_pos, 2, offset_minute)) {
            return false;
        }
        timezone_pos += 2;
        if (timezone_pos != value.size() || offset_hour > 23
            || offset_minute > 59) {
            return false;
        }
        offset_seconds = sign * (offset_hour * 3600 + offset_minute * 60);
    } else {
        return false;
    }

    std::tm utc {};
    utc.tm_year = year - 1900;
    utc.tm_mon = month - 1;
    utc.tm_mday = day;
    utc.tm_hour = hour;
    utc.tm_min = minute;
    utc.tm_sec = second;
    utc.tm_isdst = 0;
    epoch = ::timegm(&utc);
    if (epoch == static_cast<std::time_t>(-1)) {
        return false;
    }
    epoch -= offset_seconds;
    return true;
}

}  // namespace

bool read_timelapse_next_position(std::istream& input,
                                  TimelapseIndexPosition& position)
{
    position = {};
    std::string line;
    while (std::getline(input, line)) {
        const auto entry = nlohmann::json::parse(line, nullptr, false);
        if (!entry.is_object() || !entry.contains("frame")
            || !entry.contains("pts") || !entry.contains("duration")
            || !entry["frame"].is_number_unsigned()
            || !entry["pts"].is_number_unsigned()
            || !entry["duration"].is_number_unsigned()) {
            continue;
        }

        const auto frame = entry["frame"].get<uint64_t>();
        const auto pts = entry["pts"].get<uint64_t>();
        const auto duration = entry["duration"].get<uint64_t>();
        constexpr auto max_pts =
            static_cast<uint64_t>(std::numeric_limits<int64_t>::max());
        if (frame == std::numeric_limits<uint64_t>::max() || duration == 0
            || duration > max_pts || pts > max_pts - duration) {
            return false;
        }

        position.frame = std::max(position.frame, frame + 1);
        // Resume from persisted timing, including recordings made at 30 fps.
        position.pts =
            std::max(position.pts, static_cast<int64_t>(pts + duration));
    }
    return !input.bad();
}

std::string serialize_timelapse_index_entry(const TimelapseIndexEntry& entry)
{
    std::ostringstream line;
    line << "{\"frame\":" << entry.frame << ",\"pts\":" << entry.pts
         << ",\"duration\":" << entry.duration << ",\"captured_at\":\""
         << json_escape(entry.captured_at) << "\",\"capture_id\":\""
         << json_escape(entry.capture_id) << "\",\"width\":" << entry.width
         << ",\"height\":" << entry.height
         << ",\"h264_bytes\":" << entry.h264_bytes
         << ",\"ts_offset\":" << entry.ts_offset
         << ",\"segment_id\":" << entry.segment_id << "}\n";
    return line.str();
}

std::string timelapse_local_date_from_captured_at(std::string_view captured_at)
{
    std::time_t epoch = 0;
    if (!parse_captured_at_epoch(captured_at, epoch)) {
        return {};
    }

    std::tm local {};
    if (::localtime_r(&epoch, &local) == nullptr) {
        return {};
    }

    char date[16] {};
    if (std::strftime(date, sizeof(date), "%Y-%m-%d", &local) == 0) {
        return {};
    }
    return date;
}
