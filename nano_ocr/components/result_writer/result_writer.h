#ifndef RESULT_WRITER_H
#define RESULT_WRITER_H

#include <cstdint>
#include <string>
#include <vector>

#include "ocr_det.h"
#include "ocr_rec.h"

struct OCRFrameMetadata {
    std::string capturedAt;
    std::string captureId;
    std::string imageSha256;
    std::string sessionId;
    std::uint64_t sequence {0};
};

/**
 * @brief 使用显式采集元数据格式化 OCR 结果
 *
 * Socket 输入使用显式采集时间，observation 不依赖磁盘截图路径。
 */
std::string formatResultsToJson(
    const OCRFrameMetadata& metadata,
    int imageWidth,
    int imageHeight,
    const std::vector<RecognitionResultWithIndex>& results,
    const std::vector<BoundingBox>& boxes);

#endif  // RESULT_WRITER_H
