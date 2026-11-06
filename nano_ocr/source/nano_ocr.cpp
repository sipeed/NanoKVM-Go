#include <algorithm>
#include <array>
#include <atomic>
#include <cctype>
#include <cerrno>
#include <climits>
#include <csignal>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <fstream>
#include <string>
#include <vector>

#include <curl/curl.h>
#include <sys/stat.h>
#include <unistd.h>

#include "latest_frame_state.h"
#include "log.h"
#include "ocr_det.h"
#include "ocr_image.h"
#include "ocr_rec.h"
#include "result_writer.h"
#include "snapshot_client.h"

// 全局标志变量，控制日志输出
static bool g_enableTimeLog = false;  // 默认关闭时间日志
static bool g_enableMemLog = false;  // 默认关闭内存日志

// 全局变量用于控制监控循环
static std::atomic<bool> g_running(true);

namespace {

constexpr int DEFAULT_SOCKET_WAIT_MS = 15000;

struct CurlGlobalInitializer {
    CurlGlobalInitializer()
        : code(curl_global_init(CURL_GLOBAL_DEFAULT))
    {}

    ~CurlGlobalInitializer()
    {
        if (code == CURLE_OK) {
            curl_global_cleanup();
        }
    }

    bool ok() const { return code == CURLE_OK; }

    CURLcode code;
};

CurlGlobalInitializer& getCurlGlobalInitializer()
{
    static CurlGlobalInitializer initializer;
    return initializer;
}

size_t curlWriteCallback(char* ptr, size_t size, size_t nmemb, void* userdata)
{
    size_t totalSize = size * nmemb;
    if (ptr && userdata) {
        auto* response = static_cast<std::string*>(userdata);
        response->append(ptr, totalSize);
    }
    return totalSize;
}

bool parseNonNegativeInt(const char* value, int& out)
{
    if (!value || *value == '\0') {
        return false;
    }

    errno = 0;
    char* end = nullptr;
    long parsed = std::strtol(value, &end, 10);
    if (errno != 0 || end == value || *end != '\0' || parsed < 0
        || parsed > INT_MAX) {
        return false;
    }

    out = static_cast<int>(parsed);
    return true;
}

bool interruptibleSleepSeconds(int seconds)
{
    for (int elapsedMs = 0; elapsedMs < seconds * 1000 && g_running;
         elapsedMs += 100) {
        usleep(100000);
    }
    return g_running;
}

std::string safeOutputToken(const std::string& value)
{
    std::string output;
    output.reserve(value.size());
    for (const unsigned char ch : value) {
        if (std::isalnum(ch) || ch == '-' || ch == '_') {
            output.push_back(static_cast<char>(ch));
        } else {
            output.push_back('_');
        }
    }
    return output.empty() ? "unknown" : output;
}

}  // namespace

void signalHandler(int signum)
{
    printf("\nReceived signal %d, shutting down...\n", signum);
    g_running = false;
}

// ============================================
// 包装函数：兼容旧代码，使用 log 库的功能
// ============================================

// 包装函数：记录时间戳
inline void recordTime(const char* stepName) { log_record_time(stepName); }

// 包装函数：打印时间统计
inline void printTiming() { log_print_timing(); }

// 包装函数：打印内存使用
inline void printMemoryUsage(const std::string& step)
{
    if (g_enableMemLog) {
        log_memory(step);
    }
}

// 包装函数：打印最终内存总结
inline void printFinalMemorySummary()
{
    if (g_enableMemLog) {
        log_final_memory_summary();
    }
}

// 包装函数：获取内存信息
inline MemoryInfo getMemoryInfo() { return get_memory_info(); }

// 上传 JSON 到 MCP 服务（使用 libcurl）
bool uploadJsonToMcp(const std::string& jsonContent, const std::string& mcpUrl)
{
    printf("  Uploading JSON to MCP service: %s\n", mcpUrl.c_str());

    CurlGlobalInitializer& curlGlobal = getCurlGlobalInitializer();
    if (!curlGlobal.ok()) {
        printf("  Error: curl_global_init failed: %s\n",
               curl_easy_strerror(curlGlobal.code));
        return false;
    }

    CURL* curl = curl_easy_init();
    if (!curl) {
        printf("  Error: curl_easy_init failed\n");
        return false;
    }

    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");
    if (!headers) {
        printf("  Error: Failed to create curl headers\n");
        curl_easy_cleanup(curl);
        return false;
    }

    std::string response;
    long httpCode = 0;

    curl_easy_setopt(curl, CURLOPT_URL, mcpUrl.c_str());
    curl_easy_setopt(curl, CURLOPT_POST, 1L);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, jsonContent.data());
    curl_easy_setopt(curl,
                     CURLOPT_POSTFIELDSIZE_LARGE,
                     static_cast<curl_off_t>(jsonContent.size()));
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curlWriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, 2000L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS, 5000L);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);

    CURLcode result = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (result != CURLE_OK) {
        printf("  Warning: Upload failed: %s\n", curl_easy_strerror(result));
        printf("  HTTP status: %ld\n", httpCode);
        printf("  Response: %s\n", response.c_str());
        return false;
    }

    // 如果响应中包含 JSON 且有 "ok": true，则认为成功
    if (response.find("\"ok\":true") != std::string::npos
        || response.find("\"ok\": true") != std::string::npos) {
        printf("  Upload successful!\n");
        printf("  HTTP status: %ld\n", httpCode);
        printf("  Response: %s\n", response.c_str());
        return true;
    } else if (response.find("\"stored\"") != std::string::npos) {
        // 另一种成功标志：包含 stored 字段
        printf("  Upload successful!\n");
        printf("  HTTP status: %ld\n", httpCode);
        printf("  Response: %s\n", response.c_str());
        return true;
    } else {
        printf("  Warning: Upload failed or unexpected response!\n");
        printf("  HTTP status: %ld\n", httpCode);
        printf("  Response: %s\n", response.c_str());
        return false;
    }
}

int ocrPipelineFrame(
    std::string detModelPath,
    std::string recModelPath,
    std::string dictPath,
    const BGRImage& inputImage,
    const OCRFrameMetadata& frameMetadata,
    const std::string& inputDescription,
    std::string outputBase,
    bool saveImages,
    bool saveJson = true,
    bool uploadJson = false,
    std::string mcpUrl = "http://127.0.0.1:8743/v1/observations")
{
    // 如果启用了时间日志，初始化性能分析
    if (g_enableTimeLog) {
        log_perf_init();
        log_enable_perf_log(true);
        recordTime("Start");
    }

    // 如果启用了内存日志，也启用性能分析
    if (g_enableMemLog) {
        log_enable_perf_log(true);
    }

    if (!inputImage.isValid()) {
        log_error("ocr_image", "Cannot process an invalid in-memory image");
        return -1;
    }
    std::string outputPath = outputBase + "_output";

    log_info("ocr_pipeline", "========================================");
    log_info("ocr_pipeline", "OCR Pipeline Started");
    log_info("ocr_pipeline", "========================================");
    log_info("ocr_pipeline", "Detection Model: %s", detModelPath.c_str());
    log_info("ocr_pipeline", "Recognition Model: %s", recModelPath.c_str());
    log_info("ocr_pipeline", "Dictionary: %s", dictPath.c_str());
    log_info("ocr_pipeline", "Input: %s", inputDescription.c_str());
    log_info("ocr_pipeline", "Output: %s", outputPath.c_str());
    log_info("ocr_pipeline", "Save Images: %s", saveImages ? "yes" : "no");
    log_info("ocr_pipeline", "Save JSON: %s", saveJson ? "yes" : "no");
    if (uploadJson) {
        log_info("ocr_pipeline", "Upload JSON: yes (%s)", mcpUrl.c_str());
    } else {
        log_info("ocr_pipeline", "Upload JSON: no");
    }
    log_info("ocr_pipeline",
             "Time Log: %s",
             g_enableTimeLog ? "enabled" : "disabled");
    log_info(
        "ocr_pipeline", "Mem Log: %s", g_enableMemLog ? "enabled" : "disabled");
    log_info("ocr_pipeline", "========================================");
    log_info("ocr_pipeline", "");

    // 记录初始内存
    printMemoryUsage("Start");

    // 步骤 1: 加载 OcrDetNPU 模型
    log_info("ocr_det", "Step 1: Loading OCR detection model to NPU...");
    OcrDetNPU detector(detModelPath);

    if (!detector.isModelLoaded()) {
        log_error("ocr_det", "Failed to load model!");
        return -1;
    }
    log_info("ocr_det", "Model loaded successfully!");
    recordTime("Step 1 - Load Model");

    // 记录模型加载后的基础内存（包含 SDK 开销）
    MemoryInfo baseMemory = getMemoryInfo();
    log_info("memory", "");
    log_info("memory",
             "[Memory Analysis] NPU SDK base memory: %.2f MB",
             baseMemory.vmRSS / 1024.0);
    log_info("memory",
             "  Note: AXERA SDK pre-allocates memory pools for performance.");
    log_info("memory", "  This is normal even for small models.");
    log_info("memory", "");

    // 步骤 2: 校验输入的内存 BGR 图像
    log_info("ocr_image", "Step 2: Validating in-memory BGR image...");
    log_info("ocr_image",
             "Image is ready: %dx%d",
             inputImage.width,
             inputImage.height);
    recordTime("Step 2 - Validate Image");
    log_info("ocr_image", "");

    // 步骤 3: 获取 BGR 图像
    log_info("ocr_image", "Step 3: Getting BGR image structure...");
    BGRImage bgrImage = inputImage;
    log_info("ocr_image",
             "BGR image: %dx%d, %d channels",
             bgrImage.width,
             bgrImage.height,
             bgrImage.channels);
    recordTime("Step 3 - Get BGR Image");
    log_info("ocr_image", "");

    // 步骤 4: 计算分块裁剪参数
    log_info("ocr_crop", "Step 4: Calculating tile crop regions...");

    // 参数设置：缩小倍数 2，重叠区域 20 像素，输出尺寸 640x480
    float scaleFactor = 2.0f;
    int overlapPixels = 20;
    int targetWidth = 640;
    int targetHeight = 480;

    // 计算裁剪区域
    std::vector<CropRegion> cropRegions = calculateCropRegions(bgrImage.width,
                                                               bgrImage.height,
                                                               scaleFactor,
                                                               overlapPixels,
                                                               targetWidth,
                                                               targetHeight);

    if (cropRegions.empty()) {
        log_error("ocr_crop", "No valid crop regions generated!");
        return -1;
    }

    log_info("ocr_crop", "Total crop regions: %zu", cropRegions.size());
    recordTime("Step 4 - Calculate Crop Regions");
    log_info("ocr_crop", "");

    // 步骤 5: 对每个裁剪区域运行 NPU 推理
    log_info("ocr_det", "Step 5: Running NPU inference on each crop region...");
    std::vector<cv::Mat> heatmaps;

    // 记录推理前的内存状态
    MemoryInfo beforeInference = getMemoryInfo();
    log_info("memory", "");
    log_info("memory",
             "[Memory Before Inference] VmRSS: %.2f MB, VmPeak: %.2f MB",
             beforeInference.vmRSS / 1024.0,
             beforeInference.vmPeak / 1024.0);

    for (size_t i = 0; i < cropRegions.size(); i++) {
        const CropRegion& region = cropRegions[i];
        log_info("ocr_det", "");
        log_info("ocr_det",
                 "Processing region %zu/%zu [%d,%d,%d,%d]...",
                 i + 1,
                 cropRegions.size(),
                 region.x1,
                 region.y1,
                 region.x2,
                 region.y2);

        // 记录裁剪开始时间
        recordTime("Step 5 - Crop Start");

        // 裁剪图像
        BGRImage croppedImage = cropAndResize(bgrImage,
                                              region.x1,
                                              region.y1,
                                              region.x2,
                                              region.y2,
                                              targetWidth,
                                              targetHeight);

        if (!croppedImage.isValid()) {
            log_error("ocr_crop",
                      "  Warning: Failed to crop region %zu, skipping...",
                      i);
            continue;
        }

        // 记录裁剪完成时间
        recordTime(("Step 5." + std::to_string(i + 1) + " - Crop").c_str());

        log_info("ocr_crop",
                 "  Cropped image: %dx%d",
                 croppedImage.width,
                 croppedImage.height);

        // 运行 NPU 推理
        cv::Mat heatmapFloat =
            detector.detect(croppedImage, targetWidth, targetHeight);

        if (heatmapFloat.empty()) {
            log_error("ocr_det",
                      "  Warning: Failed to generate heatmap for region %zu, "
                      "skipping...",
                      i);
            continue;
        }

        // 记录 NPU 推理完成时间
        recordTime(
            ("Step 5." + std::to_string(i + 1) + " - NPU Inference").c_str());

        // 立即二值化，减少内存占用
        cv::Mat heatmapBinary = binarizeHeatmap(heatmapFloat, 0.1f);
        log_info("ocr_det",
                 "  Heatmap binarized: %dx%d, type=CV_8UC1 (uint8, memory "
                 "reduced from 1.2MB to 0.3MB)",
                 heatmapBinary.cols,
                 heatmapBinary.rows);

        // 每次推理后记录内存
        MemoryInfo currentMem = getMemoryInfo();
        log_info("memory",
                 "  [Memory after region %zu] VmRSS: %.2f MB, VmPeak: %.2f MB",
                 i + 1,
                 currentMem.vmRSS / 1024.0,
                 currentMem.vmPeak / 1024.0);

        heatmaps.push_back(heatmapBinary);
        log_info("ocr_det",
                 "  Binary heatmap stored: %dx%d",
                 heatmapBinary.cols,
                 heatmapBinary.rows);
    }

    log_info("ocr_det", "");
    log_info("ocr_det", "Total heatmaps generated: %zu", heatmaps.size());
    recordTime("Step 5 - NPU Inference");

    // 记录推理后的内存状态
    MemoryInfo afterInference = getMemoryInfo();
    log_info("memory", "");
    log_info("memory",
             "[Memory After Inference] VmRSS: %.2f MB, VmPeak: %.2f MB",
             afterInference.vmRSS / 1024.0,
             afterInference.vmPeak / 1024.0);
    log_info("memory",
             "[Memory Increase] VmRSS: +%.2f MB, VmPeak: +%.2f MB",
             (afterInference.vmRSS - beforeInference.vmRSS) / 1024.0,
             (afterInference.vmPeak - beforeInference.vmPeak) / 1024.0);
    printMemoryUsage("Step 5 - After All Inferences");
    log_info("memory", "");

    // 步骤 6: 将所有热力图拼接为原图尺寸（获取热力图数据）
    // 使用二值合并函数，因为每个小热力图已经是二值的
    log_info("ocr_merge",
             "Step 6: Merging binary heatmaps to original size...");
    cv::Mat mergedBinaryHeatmap =
        mergeBinaryHeatmaps(bgrImage, heatmaps, cropRegions);

    recordTime("Step 6 - Merge Heatmaps");
    printMemoryUsage("Step 6 - After Merge");
    log_info("ocr_merge", "");

    // 步骤 7: 从二值热力图提取边界框
    log_info("ocr_postprocess",
             "Step 7: Extracting bounding boxes from binary heatmap...");
    std::vector<BoundingBox> boxes =
        extractBoxesFromBinaryHeatmap(mergedBinaryHeatmap, 128, 500, 70);

    // 打印所有方框信息
    log_info("ocr_postprocess", "  Detected %zu bounding boxes:", boxes.size());
    for (size_t i = 0; i < boxes.size(); i++) {
        log_info("ocr_postprocess",
                 "    Box %zu: [%d, %d, %d, %d], score=%.3f",
                 i,
                 boxes[i].x1,
                 boxes[i].y1,
                 boxes[i].x2,
                 boxes[i].y2,
                 boxes[i].score);
    }
    recordTime("Step 7 - Postprocess");
    printMemoryUsage("Step 7 - After Postprocess");
    log_info("ocr_postprocess", "");

    // 步骤 8: 加载 OCR 识别模型和字典
    log_info("ocr_rec",
             "Step 8: Loading OCR recognition model and dictionary...");
    OcrRecNPU recognizer;

    if (!recognizer.loadModel(recModelPath)) {
        log_error("ocr_rec", "Failed to load recognition model!");
        return -1;
    }

    if (!recognizer.loadDictionary(dictPath)) {
        log_error("ocr_rec", "Failed to load dictionary!");
        return -1;
    }

    log_info("ocr_rec",
             "Recognition model and dictionary loaded successfully!");
    recordTime("Step 8 - Load Recognition Model");
    printMemoryUsage("Step 8 - After Load Recognition");
    log_info("ocr_rec", "");

    // 步骤 9: 对每个方框进行裁剪、保存、识别
    log_info(
        "ocr_rec",
        "Step 9: Cropping, saving and recognizing text in bounding boxes...");
    std::vector<RecognitionResultWithIndex> recognitionResults;

    // 创建 image 文件夹（仅在 --save 时）
    std::string imageFolder = "image";
    if (saveImages) {
        mkdir(imageFolder.c_str(), 0755);
        log_info("file_io", "  Created image folder: %s", imageFolder);
    }

    for (size_t i = 0; i < boxes.size(); i++) {
        const BoundingBox& box = boxes[i];

        // 使用作用域块，让 boxImages 在每次循环结束后自动释放
        {
            // 1. 裁剪方框图像（包括完整大框和小框）
            std::vector<BGRImage> boxImages =
                cropTextLinesFromBox(bgrImage, box.x1, box.y1, box.x2, box.y2);

            if (boxImages.empty()) {
                log_error("ocr_crop",
                          "  Warning: Failed to crop box %zu, skipping...",
                          i);
                continue;
            }

            // 2. 保存图像到文件（仅在 --save 时）
            if (saveImages) {
                for (size_t j = 0; j < boxImages.size(); j++) {
                    std::string fileName = std::to_string(i);
                    if (j > 0) {
                        fileName += "_" + std::to_string(j);
                    }
                    std::string imagePath =
                        imageFolder + "/" + fileName + ".jpg";

                    if (!saveImage(boxImages[j], imagePath)) {
                        log_error(
                            "file_io",
                            "  Warning: Failed to save image %s, skipping...",
                            imagePath.c_str());
                        continue;
                    }
                }
                log_info("file_io",
                         "  Saved %zu cropped image(s) for box %zu",
                         boxImages.size(),
                         i);
            }

            // 3. 对裁剪的图像进行识别
            if (boxImages.size() == 1) {
                // 只有大框，直接识别
                const BGRImage& boxImage = boxImages[0];
                log_info("ocr_rec",
                         "  Recognizing box %zu (full box, %dx%d)...",
                         i,
                         boxImage.width,
                         boxImage.height);

                OCRRecResult recResult = recognizer.recognize(boxImage);

                if (recResult.success && !recResult.text.empty()) {
                    log_info("ocr_rec",
                             "    -> Text: \"%s\" (confidence: %.4f)",
                             recResult.text.c_str(),
                             recResult.confidence);

                    RecognitionResultWithIndex result;
                    result.boxIndex = i;
                    result.subBoxIndex = 0;
                    result.box = box;
                    result.result = recResult;
                    recognitionResults.push_back(result);
                }
            } else {
                // 存在小框，识别所有小框并合并
                log_info("ocr_rec",
                         "  Recognizing %zu sub-boxes for box %zu...",
                         boxImages.size() - 1,
                         i);

                std::vector<RecognitionResultWithIndex> subBoxResults;
                for (size_t j = 1; j < boxImages.size(); j++) {
                    const BGRImage& boxImage = boxImages[j];
                    log_info("ocr_rec",
                             "    Recognizing sub-box %zu_%zu (%dx%d)...",
                             i,
                             j,
                             boxImage.width,
                             boxImage.height);

                    OCRRecResult recResult = recognizer.recognize(boxImage);

                    if (recResult.success && !recResult.text.empty()) {
                        log_info("ocr_rec",
                                 "      -> Text: \"%s\" (confidence: %.4f)",
                                 recResult.text.c_str(),
                                 recResult.confidence);

                        RecognitionResultWithIndex result;
                        result.boxIndex = i;
                        result.subBoxIndex = j;
                        result.box = box;
                        result.result = recResult;
                        subBoxResults.push_back(result);
                    }
                }

                // 合并小框结果
                if (!subBoxResults.empty()) {
                    log_info("ocr_rec", "");
                    log_info("ocr_rec",
                             "  Merging %zu sub-box results for box %zu...",
                             subBoxResults.size(),
                             i);
                    std::string mergedText =
                        OcrRecNPU::mergeAllTexts(subBoxResults);

                    RecognitionResultWithIndex mergedResult;
                    mergedResult.boxIndex = i;
                    mergedResult.subBoxIndex = 0;
                    mergedResult.box = box;
                    mergedResult.result.text = mergedText;
                    mergedResult.result.success = true;
                    mergedResult.result.confidence =
                        subBoxResults[0].result.confidence;

                    recognitionResults.push_back(mergedResult);
                    log_info("ocr_rec",
                             "  Box %zu final merged text: \"%s\"",
                             i,
                             mergedText.c_str());
                    log_info("ocr_rec", "");
                }
            }
            // boxImages 在作用域结束时自动释放
        }
    }

    log_info(
        "ocr_rec", "Total recognition results: %zu", recognitionResults.size());
    recordTime("Step 9 - Crop, Save & Recognize");
    printMemoryUsage("Step 9 - After Recognition");
    log_info("ocr_rec", "");

    // 仅在 --save 时
    if (saveImages) {
        // 步骤 10: 将热力图可视化（带方框）
        log_info("ocr_vis",
                 "Step 10: Visualizing merged heatmap with bounding boxes...");
        BGRImage visImage =
            visualizeMergedHeatmap(bgrImage, mergedBinaryHeatmap, 0.5f, boxes);
        recordTime("Step 10 - Visualize");
        printMemoryUsage("Step 10 - After Visualization");
        log_info("ocr_vis", "");

        // 步骤 11: 保存可视化结果
        std::string visImagePath = outputPath + "_vis.jpg";
        log_info("file_io", "Step 11: Saving visualization result...");
        if (saveImage(visImage, visImagePath)) {
            log_info("file_io", "  Saved: %s", visImagePath.c_str());
        } else {
            log_error("file_io",
                      "  Warning: Failed to save %s",
                      visImagePath.c_str());
        }
        recordTime("Step 11 - Save Image");
        printMemoryUsage("Step 11 - After Save");
        log_info("file_io", "");

        // 可视化完成后立即释放 mergedBinaryHeatmap
        log_info("resource", "Releasing merged heatmap...");
        mergedBinaryHeatmap.release();
        recordTime("Step 12 - Cleanup Merged Heatmap");
        printMemoryUsage("Step 12 - After Cleanup Merged Heatmap");
        log_info("resource", "");
    } else {
        log_info("ocr_vis",
                 "Step 11: Skipping visualization save (use --save to enable)");
        log_info("ocr_vis", "");

        // 不保存可视化时也释放 mergedBinaryHeatmap
        log_info("resource", "Releasing merged heatmap...");
        mergedBinaryHeatmap.release();
        recordTime("Step 12 - Cleanup Merged Heatmap");
        printMemoryUsage("Step 12 - After Cleanup Merged Heatmap");
        log_info("resource", "");
    }

    // 步骤 12: 对象离开作用域，自动释放 NPU 资源和热力图数据
    log_info("resource", "Step 12: Destroying objects...");
    // detector, bgrImage, boxes 等对象会自动调用析构函数
    log_info("resource",
             "All objects destroyed, resources released automatically");
    recordTime("Step 12 - Cleanup");
    printMemoryUsage("Step 12 - After Cleanup");
    log_info("resource", "");

    // 步骤 13: 生成 JSON 结果
    log_info("json", "Step 13: Generating JSON result...");
    std::string jsonContent = formatResultsToJson(frameMetadata,
                                                  bgrImage.width,
                                                  bgrImage.height,
                                                  recognitionResults,
                                                  boxes);

    // 根据参数决定是否保存或上传
    if (saveJson) {
        // 如果同时需要上传，保存到固定临时文件
        std::string jsonPath;
        if (uploadJson) {
            jsonPath = "/tmp/result.json";
            log_info("json",
                     "  Saving JSON to temp file for upload: %s",
                     jsonPath.c_str());
        } else {
            jsonPath = outputPath + "_result.json";
            log_info("json", "  Saving JSON to file: %s", jsonPath.c_str());
        }

        std::ofstream jsonFile(jsonPath);
        if (jsonFile.is_open()) {
            jsonFile << jsonContent;
            jsonFile.close();
            log_info("json", "  Saved JSON result: %s", jsonPath.c_str());
        } else {
            log_error("json",
                      "  Error: Cannot create JSON file: %s",
                      jsonPath.c_str());
        }
    }

    bool uploadSucceeded = true;

    // 如果需要上传
    if (uploadJson) {
        // 检查是否有识别结果
        if (recognitionResults.empty()) {
            log_info("upload",
                     "  Skipping upload: No text recognized (block_count = 0)");
        } else {
            log_info("upload", "  Uploading JSON to MCP service...");
            if (uploadJsonToMcp(jsonContent, mcpUrl)) {
                log_info("upload", "  Upload successful!");
            } else {
                log_error("upload", "  Warning: Upload failed!");
                uploadSucceeded = false;
            }
        }
    }

    recordTime("Step 13 - Generate & Save/Upload JSON");
    printMemoryUsage("Step 13 - After Save JSON");
    log_info("json", "");

    log_info("ocr_pipeline", "========================================");
    if (uploadSucceeded) {
        log_info("ocr_pipeline", "Test completed successfully!");
    } else {
        log_error("ocr_pipeline",
                  "OCR completed, but observation upload failed");
    }
    log_info("ocr_pipeline", "========================================");

    // 打印时间统计
    printTiming();

    // 打印内存分析总结
    printFinalMemorySummary();

    return uploadSucceeded ? 0 : -1;
}

int ocr_pipline(std::string detModelPath,
                std::string recModelPath,
                std::string dictPath,
                std::string inputImagePath,
                bool saveImages,
                bool saveJson = true,
                bool uploadJson = false,
                std::string mcpUrl = "http://127.0.0.1:8743/v1/observations")
{
    OCRImage image(inputImagePath);
    if (!image.isLoaded()) {
        log_error("ocr_image", "Failed to load image!");
        return -1;
    }

    std::string outputBase = inputImagePath;
    const size_t lastDot = outputBase.find_last_of('.');
    if (lastDot != std::string::npos) {
        outputBase = outputBase.substr(0, lastDot);
    }

    OCRFrameMetadata metadata;
    return ocrPipelineFrame(detModelPath,
                            recModelPath,
                            dictPath,
                            image.getImage(),
                            metadata,
                            inputImagePath,
                            outputBase,
                            saveImages,
                            saveJson,
                            uploadJson,
                            mcpUrl);
}

int processSocketFrame(const LatestFrame& frame,
                       const std::string& detModelPath,
                       const std::string& recModelPath,
                       const std::string& dictPath,
                       bool saveImages,
                       bool saveJson,
                       bool uploadJson,
                       const std::string& mcpUrl)
{
    OCRImage image(frame.jpeg);
    if (!image.isLoaded()) {
        log_error("ocr_image",
                  "Failed to decode socket frame %s sequence=%llu",
                  frame.sessionId.c_str(),
                  static_cast<unsigned long long>(frame.sequence));
        return -1;
    }
    if (image.getWidth() != frame.width || image.getHeight() != frame.height) {
        log_error("ocr_image",
                  "Socket JPEG dimensions %dx%d do not match header %dx%d",
                  image.getWidth(),
                  image.getHeight(),
                  frame.width,
                  frame.height);
        return -1;
    }

    OCRFrameMetadata metadata;
    metadata.capturedAt = frame.capturedAt;
    metadata.captureId = frame.captureId;
    metadata.sessionId = frame.sessionId;
    metadata.sequence = frame.sequence;

    const std::string inputDescription =
        "socket:" + frame.sessionId + ":" + std::to_string(frame.sequence);
    const std::string outputBase = "/tmp/nano_ocr_"
        + safeOutputToken(frame.sessionId) + "_"
        + std::to_string(frame.sequence);

    return ocrPipelineFrame(detModelPath,
                            recModelPath,
                            dictPath,
                            image.getImage(),
                            metadata,
                            inputDescription,
                            outputBase,
                            saveImages,
                            saveJson,
                            uploadJson,
                            mcpUrl);
}

int runSocketMode(const std::string& socketPath,
                  int waitMs,
                  const std::string& detModelPath,
                  const std::string& recModelPath,
                  const std::string& dictPath,
                  bool saveImages,
                  bool saveJson,
                  bool uploadJson,
                  const std::string& mcpUrl)
{
    signal(SIGINT, signalHandler);
    signal(SIGTERM, signalHandler);

    SnapshotClient client(socketPath);
    LatestFrameState frameState;
    constexpr std::array<int, 4> retrySeconds {1, 2, 5, 10};
    std::size_t retryIndex = 0;
    int processedCount = 0;

    while (g_running) {
        LatestFrameResult result =
            client.getLatest(frameState.cursor(), waitMs);
        const LatestFrameEvaluation evaluation = frameState.evaluate(result);
        if (evaluation.disposition == LatestFrameDisposition::NotModified) {
            retryIndex = 0;
            if (evaluation.sessionChanged) {
                log_warn("snapshot_client",
                         "kvm_vin session changed while no OCR frame is "
                         "available: %s -> %s; keeping the committed cursor "
                         "until OCR succeeds",
                         frameState.cursor().sessionId.c_str(),
                         result.currentSessionId.c_str());
            }
            continue;
        }

        if (result.status != LatestFrameStatus::Frame) {
            const int delay = retrySeconds[retryIndex];
            retryIndex = std::min(retryIndex + 1, retrySeconds.size() - 1);
            log_error("snapshot_client",
                      "get_latest failed: %s; reconnecting in %d second(s)",
                      result.error.c_str(),
                      delay);
            interruptibleSleepSeconds(delay);
            continue;
        }

        const LatestFrame& frame = result.frame;
        if (evaluation.disposition == LatestFrameDisposition::Stale) {
            const int delay = retrySeconds[retryIndex];
            retryIndex = std::min(retryIndex + 1, retrySeconds.size() - 1);
            log_error(
                "snapshot_client",
                "get_latest returned a stale or non-increasing OCR "
                "sequence: last=%s/%llu current=%s/%llu; retrying in "
                "%d second(s)",
                frameState.cursor().sessionId.c_str(),
                static_cast<unsigned long long>(frameState.cursor().sequence),
                frame.sessionId.c_str(),
                static_cast<unsigned long long>(frame.sequence),
                delay);
            interruptibleSleepSeconds(delay);
            continue;
        }
        if (evaluation.sessionChanged) {
            log_warn("snapshot_client",
                     "kvm_vin session changed: %s -> %s; the cursor will move "
                     "after OCR succeeds",
                     frameState.cursor().sessionId.c_str(),
                     frame.sessionId.c_str());
        }
        if (evaluation.gap > 0) {
            log_warn(
                "snapshot_client",
                "OCR latest-frame gap: dropped=%llu last=%llu "
                "current=%llu session=%s",
                static_cast<unsigned long long>(evaluation.gap),
                static_cast<unsigned long long>(frameState.cursor().sequence),
                static_cast<unsigned long long>(frame.sequence),
                frame.sessionId.c_str());
        }

        log_info("snapshot_client",
                 "Processing canonical frame: session=%s sequence=%llu "
                 "capture_id=%s captured_at=%s size=%zu",
                 frame.sessionId.c_str(),
                 static_cast<unsigned long long>(frame.sequence),
                 frame.captureId.c_str(),
                 frame.capturedAt.c_str(),
                 frame.jpeg.size());

        const int pipelineResult = processSocketFrame(frame,
                                                      detModelPath,
                                                      recModelPath,
                                                      dictPath,
                                                      saveImages,
                                                      saveJson,
                                                      uploadJson,
                                                      mcpUrl);
        const bool cursorCommitted =
            frameState.complete(frame, pipelineResult == 0);
        if (pipelineResult != 0) {
            const int delay = retrySeconds[retryIndex];
            retryIndex = std::min(retryIndex + 1, retrySeconds.size() - 1);
            log_error("snapshot_client",
                      "OCR failed for sequence=%llu; retrying latest frame in "
                      "%d second(s)",
                      static_cast<unsigned long long>(frame.sequence),
                      delay);
            interruptibleSleepSeconds(delay);
            continue;
        }
        if (!cursorCommitted) {
            const int delay = retrySeconds[retryIndex];
            retryIndex = std::min(retryIndex + 1, retrySeconds.size() - 1);
            log_error("snapshot_client",
                      "OCR succeeded but frame cursor could not be committed; "
                      "retrying in %d second(s)",
                      delay);
            interruptibleSleepSeconds(delay);
            continue;
        }

        retryIndex = 0;
        processedCount++;
        log_info("snapshot_client",
                 "Socket OCR processed %d frame(s); cursor=%s/%llu",
                 processedCount,
                 frameState.cursor().sessionId.c_str(),
                 static_cast<unsigned long long>(frameState.cursor().sequence));
    }

    printf("\nSocket monitoring stopped. Total processed: %d frame(s)\n",
           processedCount);
    return 0;
}

int main(int argc, char** argv)
{
    bool saveImages = false;  // 默认不保存小图和可视化结果
    bool saveJson = true;  // 默认保存 JSON
    bool uploadJson = false;  // 默认不上传
    LogLevel configuredLogLevel = LogLevel::INFO;
    std::string socketPath;  // kvm_vin latest-frame Socket 路径
    int socketWaitMs = DEFAULT_SOCKET_WAIT_MS;
    std::string mcpUrl =
        "http://127.0.0.1:8743/v1/observations";  // MCP 服务地址

    // 默认路径
    std::string detModelPath =
        "/kvmcomm/models/pp_ocr/ch_PP_OCRv3_det_npu.axmodel";
    std::string recModelPath =
        "/kvmcomm/models/pp_ocr/ch_PP_OCRv4_rec_npu.axmodel";
    std::string dictPath = "/kvmcomm/models/pp_ocr/ppocr_keys_v1.txt";
    std::string inputImagePath;

    // 解析参数：支持 --save, --det_model, --rec_model, --dict, --image,
    // --snapshot-socket, --upload, --mcp-url, --time_log, --mem_log, --log 选项
    for (int i = 1; i < argc; i++) {
        std::string arg = argv[i];
        if (arg == "--save") {
            saveImages = true;
        } else if (arg == "--no-save-json") {
            saveJson = false;
        } else if (arg == "--upload") {
            uploadJson = true;
        } else if (arg == "--time_log") {
            g_enableTimeLog = true;
        } else if (arg == "--mem_log") {
            g_enableMemLog = true;
        } else if (arg == "--log" && i + 1 < argc) {
            configuredLogLevel = string_to_log_level(argv[++i]);
        } else if (arg == "--snapshot-socket" && i + 1 < argc) {
            socketPath = argv[++i];
        } else if (arg == "--socket-wait-ms" && i + 1 < argc) {
            if (!parseNonNegativeInt(argv[++i], socketWaitMs)
                || socketWaitMs > SNAPSHOT_MAX_WAIT_MS) {
                printf(
                    "Error: --socket-wait-ms must be an integer in "
                    "0..30000\n");
                return -1;
            }
        } else if (arg == "--mcp-url" && i + 1 < argc) {
            mcpUrl = argv[++i];
        } else if (arg == "--det_model" && i + 1 < argc) {
            detModelPath = argv[++i];
        } else if (arg == "--rec_model" && i + 1 < argc) {
            recModelPath = argv[++i];
        } else if (arg == "--dict" && i + 1 < argc) {
            dictPath = argv[++i];
        } else if (arg == "--image" && i + 1 < argc) {
            inputImagePath = argv[++i];
        } else if (arg == "--help" || arg == "-h") {
            printf("Usage: %s [OPTIONS]\n", argv[0]);
            printf("Options:\n");
            printf(
                "  --save          Save cropped images and visualization "
                "result\n");
            printf(
                "  --no-save-json  Do not save JSON to file (default: save)\n");
            printf("  --upload        Upload JSON to MCP service\n");
            printf(
                "  --time_log      Enable timing logs (default: disabled)\n");
            printf(
                "  --mem_log       Enable memory monitoring logs (default: "
                "disabled)\n");
            printf(
                "  --log LEVEL     Set log level (none/error/info, default: "
                "info)\n");
            printf(
                "  --mcp-url URL   MCP service URL (default: "
                "http://127.0.0.1:8743/v1/observations)\n");
            printf(
                "  --snapshot-socket PATH  Long-poll canonical OCR frames "
                "from the "
                "kvm_vin snapshot Socket\n");
            printf(
                "  --socket-wait-ms M  get_latest long-poll timeout in ms "
                "(0..30000, default: 15000)\n");
            printf(
                "  --det_model PATH Detection model path (default: "
                "/kvmcomm/models/pp_ocr/ch_PP_OCRv3_det_npu.axmodel)\n");
            printf(
                "  --rec_model PATH Recognition model path (default: "
                "/kvmcomm/models/pp_ocr/ch_PP_OCRv4_rec_npu.axmodel)\n");
            printf(
                "  --dict PATH     Dictionary path (default: "
                "/kvmcomm/models/pp_ocr/ppocr_keys_v1.txt)\n");
            printf(
                "  --image PATH    Input image path for offline diagnostics\n");
            printf("  --help, -h      Show this help message\n");
            printf("\nExample:\n");
            printf("%s --image ./test.png\n", argv[0]);
            printf(
                "%s --save --image ./test.png --det_model "
                "/path/to/model.axmodel\n",
                argv[0]);
            printf("%s --snapshot-socket /run/kvm/vin_snapshot.sock --upload\n",
                   argv[0]);
            printf("%s --image ./test.png --time_log --mem_log\n", argv[0]);
            return 0;
        } else {
            printf("Error: Unknown option: %s\n", arg.c_str());
            return -1;
        }
    }

    log_set_level(configuredLogLevel);

    // Socket 模式不与单图诊断输入混用。
    if (!socketPath.empty() && !inputImagePath.empty()) {
        printf(
            "Error: --snapshot-socket is mutually exclusive with --image\n");
        return -1;
    }
    if (inputImagePath.empty() && socketPath.empty()) {
        printf(
            "Error: One of --image or --snapshot-socket is required!\n");
        printf(
            "Usage: %s [--save] --image <input_image> [--det_model PATH] "
            "[--rec_model PATH] [--dict PATH]\n",
            argv[0]);
        printf(
            "   or: %s --snapshot-socket <socket_path> [--socket-wait-ms M] "
            "[--upload]\n",
            argv[0]);
        printf("Use --help for more information.\n");
        return -1;
    }

    if (!socketPath.empty()) {
        printf("========================================\n");
        printf("OCR Canonical Socket Mode\n");
        printf("========================================\n");
        printf("Snapshot socket: %s\n", socketPath.c_str());
        printf("Long-poll wait: %d ms\n", socketWaitMs);
        printf("Upload observations: %s\n", uploadJson ? "yes" : "no");
        printf("Press Ctrl+C to stop monitoring\n");
        printf("========================================\n\n");
        return runSocketMode(socketPath,
                             socketWaitMs,
                             detModelPath,
                             recModelPath,
                             dictPath,
                             saveImages,
                             saveJson,
                             uploadJson,
                             mcpUrl);
    }

    // 单张图片处理模式
    printf("========================================\n");
    printf("OCR Single Image Mode\n");
    printf("========================================\n");

    // 调用 OCR 流程函数（单图模式：保存 JSON，不上传）
    ocr_pipline(detModelPath,
                recModelPath,
                dictPath,
                inputImagePath,
                saveImages,
                saveJson,
                uploadJson,
                mcpUrl);

    return 0;
}
