#include <chrono>
#include <ctime>
#include <iomanip>
#include <sstream>

#include "result_writer.h"

// 转义 JSON 字符串中的特殊字符
static std::string escapeJsonString(const std::string& input)
{
    std::ostringstream oss;
    for (char c : input) {
        switch (c) {
        case '"':
            oss << "\\\"";
            break;
        case '\\':
            oss << "\\\\";
            break;
        case '\b':
            oss << "\\b";
            break;
        case '\f':
            oss << "\\f";
            break;
        case '\n':
            oss << "\\n";
            break;
        case '\r':
            oss << "\\r";
            break;
        case '\t':
            oss << "\\t";
            break;
        default:
            if ('\x00' <= c && c <= '\x1f') {
                oss << "\\u" << std::hex << std::setw(4) << std::setfill('0')
                    << (int)c;
            } else {
                oss << c;
            }
        }
    }
    return oss.str();
}

// 获取当前时间的 RFC3339 格式（自动获取系统时区）
static std::string getCurrentTimeRFC3339()
{
    auto now = std::chrono::system_clock::now();
    auto time_t_now = std::chrono::system_clock::to_time_t(now);
    std::tm tm_now;

    // 获取本地时间
    localtime_r(&time_t_now, &tm_now);

    // 获取时区偏移量（秒）
    // tm_gmtoff 是 Linux 特有的扩展，表示本地时间与 UTC 的偏移（秒）
    long tz_offset = tm_now.tm_gmtoff;

    // 计算时区的小时和分钟
    int tz_hours = tz_offset / 3600;
    int tz_minutes = abs(tz_offset % 3600) / 60;

    // 确定时区符号
    char tz_sign = (tz_offset >= 0) ? '+' : '-';

    std::ostringstream oss;
    oss << std::put_time(&tm_now, "%Y-%m-%dT%H:%M:%S");
    oss << tz_sign << std::setw(2) << std::setfill('0') << abs(tz_hours) << ":"
        << std::setw(2) << std::setfill('0') << tz_minutes;

    return oss.str();
}

std::string formatResultsToJson(
    const OCRFrameMetadata& metadata,
    int imageWidth,
    int imageHeight,
    const std::vector<RecognitionResultWithIndex>& results,
    const std::vector<BoundingBox>& boxes)
{
    std::ostringstream jsonStream;

    // 计算整体置信度
    float totalConfidence = 0.0f;
    int validCount = 0;
    for (const auto& result : results) {
        if (result.result.success && !result.result.text.empty()) {
            totalConfidence += result.result.confidence;
            validCount++;
        }
    }
    float avgConfidence =
        (validCount > 0) ? (totalConfidence / validCount) : 0.0f;

    const std::string capturedAt = !metadata.capturedAt.empty()
        ? metadata.capturedAt
        : getCurrentTimeRFC3339();

    // 写入 JSON 内容
    jsonStream << "{\n";
    jsonStream << "  \"schema_version\": 1,\n";
    jsonStream << "  \"type\": \"ocr_observation\",\n";
    jsonStream << "  \"captured_at\": \"" << escapeJsonString(capturedAt)
               << "\",\n";
    jsonStream << "  \"capture_id\": \""
               << escapeJsonString(metadata.captureId) << "\",\n";
    jsonStream << "  \"source\": {\n";
    jsonStream << "    \"kind\": \"screen_ocr\",\n";
    jsonStream << "    \"device\": \"nanokvm-go\",\n";
    jsonStream << "    \"engine\": \"nano_ocr_npu\"\n";
    jsonStream << "  },\n";
    jsonStream << "  \"screen\": {\n";
    jsonStream << "    \"width\": " << imageWidth << ",\n";
    jsonStream << "    \"height\": " << imageHeight;
    if (!metadata.imageSha256.empty()) {
        jsonStream << ",\n    \"image_sha256\": \""
                   << escapeJsonString(metadata.imageSha256) << "\"\n";
    } else {
        jsonStream << "\n";
    }
    jsonStream << "  },\n";
    jsonStream << "  \"context\": {\n";
    jsonStream << "    \"app_name\": \"ScreenOCR\",\n";
    jsonStream << "    \"title\": \"Nano OCR Result\"\n";
    jsonStream << "  },\n";
    jsonStream << "  \"event\": {\n";
    jsonStream << "    \"type\": \"ocr_capture\",\n";
    jsonStream << "    \"reason\": \"manual\"\n";
    jsonStream << "  },\n";
    jsonStream << "  \"ocr\": {\n";
    jsonStream << "    \"language\": \"zh\",\n";

    // 拼接所有文本
    std::string fullText;
    for (size_t i = 0; i < results.size(); i++) {
        if (i > 0)
            fullText += " ";
        fullText += results[i].result.text;
    }
    jsonStream << "    \"text\": \"" << escapeJsonString(fullText) << "\",\n";
    jsonStream << "    \"confidence\": " << std::fixed << std::setprecision(4)
               << avgConfidence << ",\n";
    jsonStream << "    \"block_count\": " << results.size() << ",\n";
    jsonStream << "    \"blocks\": [\n";

    // 写入每个 block
    for (size_t i = 0; i < results.size(); i++) {
        const auto& result = results[i];
        jsonStream << "      {\n";
        jsonStream << "        \"id\": " << (i + 1) << ",\n";
        jsonStream << "        \"text\": \""
                   << escapeJsonString(result.result.text) << "\",\n";
        jsonStream << "        \"bbox\": [" << result.box.x1 << ", "
                   << result.box.y1 << ", " << result.box.x2 << ", "
                   << result.box.y2 << "],\n";
        jsonStream << "        \"confidence\": " << std::fixed
                   << std::setprecision(4) << result.result.confidence << "\n";
        jsonStream << "      }";
        if (i < results.size() - 1) {
            jsonStream << ",";
        }
        jsonStream << "\n";
    }

    jsonStream << "    ]\n";
    jsonStream << "  },\n";
    jsonStream << "  \"tags\": [\"external\", \"ocr\"]\n";
    jsonStream << "}\n";

    return jsonStream.str();
}
