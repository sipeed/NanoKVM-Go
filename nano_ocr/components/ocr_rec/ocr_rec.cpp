#include <algorithm>
#include <cmath>
#include <cstdio>
#include <cstring>

#include "ocr_rec.h"

OcrRecNPU::OcrRecNPU()
    : handle(0)
    , modelLoaded(false)
    , dictSize(0)
    , cachedIOInfo(nullptr)
    , ioBuffersAllocated(false)
{ memset(&cachedIOData, 0, sizeof(cachedIOData)); }

OcrRecNPU::~OcrRecNPU()
{
    // 释放缓存的 IO 缓冲区
    if (ioBuffersAllocated) {
        freeIO(&cachedIOData);
        ioBuffersAllocated = false;
    }

    if (handle) {
        AX_ENGINE_DestroyHandle(handle);
    }
    AX_ENGINE_Deinit();
    AX_SYS_Deinit();
}

bool OcrRecNPU::fileExists(const std::string& filename)
{
    FILE* fp = fopen(filename.c_str(), "r");
    if (fp) {
        fclose(fp);
        return true;
    }
    return false;
}

bool OcrRecNPU::loadModel(const std::string& modelPath)
{
    if (!fileExists(modelPath)) {
        printf("Error: Model file not found: %s\n", modelPath.c_str());
        return false;
    }

    FILE* fp = fopen(modelPath.c_str(), "rb");
    if (!fp) {
        printf("Error: Cannot open model file: %s\n", modelPath.c_str());
        return false;
    }

    fseek(fp, 0, SEEK_END);
    long size = ftell(fp);
    fseek(fp, 0, SEEK_SET);

    modelBuffer.resize(size);

    if (fread(modelBuffer.data(), 1, size, fp) != (size_t)size) {
        printf("Error: Cannot read model file\n");
        fclose(fp);
        return false;
    }

    fclose(fp);
    printf("Model loaded: %s (%ld bytes)\n", modelPath.c_str(), size);

    // 初始化 AXERA SDK
    AX_S32 ret = AX_SYS_Init();
    if (ret != 0) {
        printf("Error: AX_SYS_Init failed: 0x%x\n", ret);
        return false;
    }

    // 初始化 ENGINE
    AX_ENGINE_NPU_ATTR_T npuAttr;
    memset(&npuAttr, 0, sizeof(AX_ENGINE_NPU_ATTR_T));
    npuAttr.eHardMode = AX_ENGINE_VIRTUAL_NPU_DISABLE;
    ret = AX_ENGINE_Init(&npuAttr);
    if (ret != 0) {
        printf("Error: AX_ENGINE_Init failed: 0x%x\n", ret);
        AX_SYS_Deinit();
        return false;
    }

    // 使用模型缓冲区创建 handle
    ret =
        AX_ENGINE_CreateHandle(&handle, modelBuffer.data(), modelBuffer.size());
    if (ret != 0 || !handle) {
        printf("Error: AX_ENGINE_CreateHandle failed: 0x%x\n", ret);
        AX_ENGINE_DestroyHandle(handle);
        handle = 0;
        AX_ENGINE_Deinit();
        AX_SYS_Deinit();
        return false;
    }

    // 创建上下文
    ret = AX_ENGINE_CreateContext(handle);
    if (ret != 0) {
        printf("Error: AX_ENGINE_CreateContext failed: 0x%x\n", ret);
        AX_ENGINE_DestroyHandle(handle);
        handle = 0;
        AX_ENGINE_Deinit();
        AX_SYS_Deinit();
        return false;
    }

    modelLoaded = true;

    // 预分配 IO 缓冲区以加速推理
    printf("Pre-allocating IO buffers for faster inference...\n");
    cachedIOInfo = new AX_ENGINE_IO_INFO_T;
    memset(cachedIOInfo, 0, sizeof(AX_ENGINE_IO_INFO_T));

    AX_S32 retIO = AX_ENGINE_GetIOInfo(handle, &cachedIOInfo);
    if (retIO != 0) {
        printf(
            "Warning: AX_ENGINE_GetIOInfo failed, will allocate per "
            "inference\n");
        delete cachedIOInfo;
        cachedIOInfo = nullptr;
    } else {
        if (prepareIO(cachedIOInfo, &cachedIOData)) {
            ioBuffersAllocated = true;
            printf("IO buffers allocated successfully!\n");
        } else {
            printf("Warning: prepareIO failed, will allocate per inference\n");
            delete cachedIOInfo;
            cachedIOInfo = nullptr;
        }
    }

    return true;
}

bool OcrRecNPU::loadDictionary(const std::string& dictPath)
{
    if (!fileExists(dictPath)) {
        printf("Error: Dictionary file not found: %s\n", dictPath.c_str());
        return false;
    }

    FILE* fp = fopen(dictPath.c_str(), "r");
    if (!fp) {
        printf("Error: Cannot open dictionary file: %s\n", dictPath.c_str());
        return false;
    }

    dictionary.clear();
    char line[1024];

    while (fgets(line, sizeof(line), fp)) {
        // 移除换行符
        size_t len = strlen(line);
        if (len > 0 && line[len - 1] == '\n') {
            line[len - 1] = '\0';
            len--;
        }
        if (len > 0 && line[len - 1] == '\r') {
            line[len - 1] = '\0';
            len--;
        }

        if (len > 0) {
            dictionary.push_back(std::string(line));
        }
    }

    fclose(fp);

    dictionary.push_back(" ");

    dictSize = dictionary.size();
    printf(
        "Dictionary loaded: %s (%d characters)\n", dictPath.c_str(), dictSize);

    return true;
}

std::string OcrRecNPU::getChar(int index) const
{
    if (index >= 0 && index < (int)dictionary.size()) {
        return dictionary[index];
    }
    return "";
}

void OcrRecNPU::freeIO(AX_ENGINE_IO_T* io)
{
    if (!io)
        return;

    for (AX_U32 i = 0; i < io->nInputSize; i++) {
        if (io->pInputs[i].pVirAddr) {
            AX_SYS_MemFree(io->pInputs[i].phyAddr, io->pInputs[i].pVirAddr);
        }
    }

    for (AX_U32 i = 0; i < io->nOutputSize; i++) {
        if (io->pOutputs[i].pVirAddr) {
            AX_SYS_MemFree(io->pOutputs[i].phyAddr, io->pOutputs[i].pVirAddr);
        }
    }

    delete[] io->pInputs;
    delete[] io->pOutputs;
}

bool OcrRecNPU::prepareIO(AX_ENGINE_IO_INFO_T* ioInfo, AX_ENGINE_IO_T* ioData)
{
    memset(ioData, 0, sizeof(*ioData));

    // 分配输入缓冲区
    ioData->pInputs = new AX_ENGINE_IO_BUFFER_T[ioInfo->nInputSize];
    memset(
        ioData->pInputs, 0, sizeof(AX_ENGINE_IO_BUFFER_T) * ioInfo->nInputSize);
    ioData->nInputSize = ioInfo->nInputSize;

    for (AX_U32 i = 0; i < ioInfo->nInputSize; i++) {
        auto& meta = ioInfo->pInputs[i];
        auto& buffer = ioData->pInputs[i];

        AX_S32 ret = AX_SYS_MemAlloc((AX_U64*)(&buffer.phyAddr),
                                     &buffer.pVirAddr,
                                     meta.nSize,
                                     OCR_REC_CMM_ALIGN_SIZE,
                                     (const AX_S8*)OCR_REC_CMM_SESSION_NAME);

        if (ret != 0) {
            printf("Error: Failed to allocate input buffer %d (size: %u)\n",
                   i,
                   meta.nSize);
            freeIO(ioData);
            return false;
        }
    }

    // 分配输出缓冲区
    ioData->pOutputs = new AX_ENGINE_IO_BUFFER_T[ioInfo->nOutputSize];
    memset(ioData->pOutputs,
           0,
           sizeof(AX_ENGINE_IO_BUFFER_T) * ioInfo->nOutputSize);
    ioData->nOutputSize = ioInfo->nOutputSize;

    for (AX_U32 i = 0; i < ioInfo->nOutputSize; i++) {
        auto& meta = ioInfo->pOutputs[i];
        auto& buffer = ioData->pOutputs[i];
        buffer.nSize = meta.nSize;

        AX_S32 ret = AX_SYS_MemAlloc((AX_U64*)(&buffer.phyAddr),
                                     &buffer.pVirAddr,
                                     meta.nSize,
                                     OCR_REC_CMM_ALIGN_SIZE,
                                     (const AX_S8*)OCR_REC_CMM_SESSION_NAME);

        if (ret != 0) {
            printf("Error: Failed to allocate output buffer %d (size: %u)\n",
                   i,
                   meta.nSize);
            freeIO(ioData);
            return false;
        }
    }

    return true;
}

cv::Mat OcrRecNPU::preprocessImage(const cv::Mat& inputImage,
                                   int targetWidth,
                                   int targetHeight)
{
    // 快速路径：如果图像已经是正确的尺寸和 3 通道，直接返回
    if (inputImage.cols == targetWidth && inputImage.rows == targetHeight
        && inputImage.channels() == 3) {
        return inputImage.clone();
    }

    printf("  Converting image: %dx%d channels=%d -> %dx%d\n",
           inputImage.cols,
           inputImage.rows,
           inputImage.channels(),
           targetWidth,
           targetHeight);

    cv::Mat output;

    // 模型需要 3 通道 BGR 图像
    if (inputImage.channels() == 1) {
        // 灰度图转 BGR
        cv::cvtColor(inputImage, output, cv::COLOR_GRAY2BGR);
    } else if (inputImage.channels() == 4) {
        // BGRA 转 BGR
        cv::cvtColor(inputImage, output, cv::COLOR_BGRA2BGR);
    } else if (inputImage.channels() == 3) {
        // 已经是 BGR，直接克隆
        output = inputImage.clone();
    } else {
        // 其他情况
        output = inputImage.clone();
    }

    // 缩放到目标尺寸
    if (output.cols != targetWidth || output.rows != targetHeight) {
        printf("    -> Resizing from %dx%d to %dx%d\n",
               output.cols,
               output.rows,
               targetWidth,
               targetHeight);
        cv::resize(output,
                   output,
                   cv::Size(targetWidth, targetHeight),
                   0,
                   0,
                   cv::INTER_AREA);
    }

    // 保持 uint8 格式 - 模型需要 uint8 输入 (0-255 范围)
    // 不要归一化到 [0,1] 或转换为 float

    return output;
}

std::string OcrRecNPU::decodeCTC(const float* outputData,
                                 int width,
                                 int height,
                                 std::vector<float>& charConfidences,
                                 std::vector<int>& charIndices)
{
    std::string result;
    charConfidences.clear();
    charIndices.clear();

    int lastIndex = -1;
    float totalConfidence = 0.0f;
    int charCount = 0;

    // CTC 解码：对每个时间步，找到概率最大的字符
    for (int t = 0; t < width; t++) {
        int maxIndex = 0;
        float maxValue = outputData[t * height];

        // 在这个时间步中找到最大值（跨越所有字符）
        for (int c = 1; c < height; c++) {
            float value = outputData[t * height + c];
            if (value > maxValue) {
                maxValue = value;
                maxIndex = c;
            }
        }

        // 跳过空白（索引 0）和重复字符
        if (maxIndex > 0 && maxIndex != lastIndex) {
            // 模型输出已经是 softmax 概率，直接使用 maxValue
            float confidence = maxValue;

            // 修正索引：模型输出在索引 0 处包含空白，但字典从实际字符开始
            // 所以需要减 1 来匹配字典索引
            int dictIndex = maxIndex - 1;

            if (dictIndex >= 0 && dictIndex < (int)dictionary.size()) {
                result += dictionary[dictIndex];
                charConfidences.push_back(confidence);
                charIndices.push_back(dictIndex);
                totalConfidence += confidence;
                charCount++;
            }
        }

        lastIndex = maxIndex;
    }

    return result;
}

OCRRecResult OcrRecNPU::recognize(const BGRImage& inputImage)
{
    OCRRecResult result;

    if (!modelLoaded) {
        result.errorMessage = "Model not loaded";
        return result;
    }

    if (!inputImage.isValid()) {
        result.errorMessage = "Input image is invalid";
        return result;
    }

    // 预处理图像
    cv::Mat processedImage = preprocessImage(inputImage.data, 320, 48);

    if (processedImage.empty()) {
        result.errorMessage = "Failed to preprocess image";
        return result;
    }

    // 如果缓存了 IO 信息和缓冲区，则使用（更快！）
    AX_ENGINE_IO_INFO_T* ioInfo = nullptr;
    AX_ENGINE_IO_T ioData;
    bool useCachedIO = false;

    if (cachedIOInfo && ioBuffersAllocated) {
        ioInfo = cachedIOInfo;
        ioData = cachedIOData;
        useCachedIO = true;
    } else {
        // 回退：动态分配 IO 缓冲区（较慢）
        ioInfo = new AX_ENGINE_IO_INFO_T;
        memset(ioInfo, 0, sizeof(AX_ENGINE_IO_INFO_T));

        AX_S32 ret2 = AX_ENGINE_GetIOInfo(handle, &ioInfo);
        if (ret2 != 0 || !ioInfo) {
            result.errorMessage = "AX_ENGINE_GetIOInfo failed";
            delete ioInfo;
            return result;
        }

        if (!prepareIO(ioInfo, &ioData)) {
            result.errorMessage = "Failed to prepare IO";
            delete ioInfo;
            return result;
        }
    }

    // 复制图像数据到输入缓冲区
    // 输入 shape: [1, 48, 320, 3] for BGR NHWC format
    AX_U8* inputData = (AX_U8*)ioData.pInputs[0].pVirAddr;

    // 根据模型输入 shape 计算预期大小
    // 模型需要 48x320x3 = 46080 字节的 BGR 图像
    size_t expectedSize = 48 * 320 * 3 * sizeof(AX_U8);

    printf("  Preprocessed image: %dx%d, channels=%d, size=%zu bytes\n",
           processedImage.cols,
           processedImage.rows,
           processedImage.channels(),
           processedImage.total() * processedImage.elemSize());
    printf("  Expected input size: %zu bytes\n", expectedSize);

    // 确保图像在内存中是连续的
    if (processedImage.isContinuous()) {
        memcpy(inputData,
               processedImage.data,
               processedImage.total() * processedImage.elemSize());
    } else {
        // 逐行复制
        size_t rowSize = processedImage.cols * processedImage.elemSize();
        for (int r = 0; r < processedImage.rows; r++) {
            memcpy(inputData + r * processedImage.cols * 3,
                   processedImage.ptr<AX_U8>(r),
                   rowSize);
        }
    }

    // 运行推理
    AX_S32 ret = AX_ENGINE_RunSync(handle, &ioData);
    if (ret != 0) {
        printf("Error: AX_ENGINE_RunSync failed: 0x%x\n", ret);
        freeIO(&ioData);
        result.errorMessage = "Inference failed";
        return result;
    }

    // 获取输出数据
    // 输出 shape: [1, 6624, 1, 1] 或类似（取决于模型）
    // 对于 PaddleOCR: [1, num_classes, sequence_length, 1]
    float* outputData = (float*)ioData.pOutputs[0].pVirAddr;

    // 获取输出维度
    int outputSize = 1;
    for (AX_U32 i = 0; i < ioInfo->pOutputs[0].nShapeSize; i++) {
        outputSize *= ioInfo->pOutputs[0].pShape[i];
    }

    printf("  Output shape: ");
    for (AX_U32 i = 0; i < ioInfo->pOutputs[0].nShapeSize; i++) {
        printf("%d ", ioInfo->pOutputs[0].pShape[i]);
    }
    printf("\n");
    printf("  Total output size: %d floats\n", outputSize);

    int sequenceLength = ioInfo->pOutputs[0].pShape[1];  // 通常是第二个维度
    int numClasses = ioInfo->pOutputs[0].pShape[2];  // 字符类别数量

    printf("  Parsed: sequenceLength=%d, numClasses=%d\n",
           sequenceLength,
           numClasses);

    // 对于 PaddleOCR rec 模型，输出通常是 [1, 6624, 1, 1] 或 [1, T, C, 1]
    // 其中 T 是序列长度，C 是类别数
    // 我们需要 reshape 为 [T, C] 用于 CTC 解码

    // 解码 CTC 输出
    std::vector<float> charConfidences;
    std::vector<int> charIndices;

    // 假设输出是 [1, sequenceLength, numClasses, 1] NHWC 格式
    // reshape 为 [sequenceLength, numClasses]
    result.text = decodeCTC(
        outputData, sequenceLength, numClasses, charConfidences, charIndices);
    result.charConfidences = charConfidences;
    result.charIndices = charIndices;

    printf("  Decoded text: \"%s\" (length=%zu, chars=%d)\n",
           result.text.c_str(),
           result.text.length(),
           (int)charIndices.size());

    // 计算平均置信度
    if (!charConfidences.empty()) {
        float sum = 0.0f;
        for (float conf : charConfidences) {
            sum += conf;
        }
        result.confidence = sum / charConfidences.size();
    }

    result.success = true;

    // 清理：仅当动态分配时才释放（不使用缓存缓冲区）
    if (!useCachedIO) {
        freeIO(&ioData);
        delete ioInfo;
    }
    // 如果使用缓存 IO，缓冲区保留用于下次推理（在析构函数中释放）

    return result;
}

std::string OcrRecNPU::mergeTexts(const std::string& text1,
                                  const std::string& text2,
                                  size_t checkRange)
{
    if (text1.empty())
        return text2;
    if (text2.empty())
        return text1;

    printf("  Merging texts:\n");
    printf("    Text1: \"%s\"\n", text1.c_str());
    printf("    Text2: \"%s\"\n", text2.c_str());

    size_t len1 = text1.length();
    size_t len2 = text2.length();

    // 实际检查的字符数（不能超过字符串长度）
    size_t actualCheck1 = std::min(checkRange, len1);
    size_t actualCheck2 = std::min(checkRange, len2);

    // 存储最佳匹配位置
    int bestA = -1;  // text1 中的位置（从后往前数）
    int bestB = -1;  // text2 中的位置（从前往后数）

    // 两两比较：text1 的最后 actualCheck1 个字符 vs text2 的前 actualCheck2
    // 个字符
    printf("  Comparing characters (max %zu comparisons):\n",
           actualCheck1 * actualCheck2);

    for (size_t i = 0; i < actualCheck1; i++) {
        for (size_t j = 0; j < actualCheck2; j++) {
            // text1 倒数第 (i+1) 个字符
            char a = text1[len1 - 1 - i];
            // text2 正数第 j 个字符
            char b = text2[j];

            // 大小写不敏感比较
            if (tolower(a) == tolower(b)) {
                printf("    Match found: a%zu='%c' == b%zu='%c'\n",
                       i + 1,
                       a,
                       j + 1,
                       b);
                // 记录第一个找到的匹配（最靠后的重叠点）
                if (bestA < 0) {
                    bestA = i;
                    bestB = j;
                }
            }
        }
    }

    // 如果找到匹配，进行合并
    if (bestA >= 0 && bestB >= 0) {
        // text1 保留到倒数第 (bestA+1) 个字符（包含）
        // text2 从第 (bestB+1) 个字符开始（跳过匹配的字符）
        std::string merged =
            text1.substr(0, len1 - bestA) + text2.substr(bestB + 1);
        printf("    -> Merged at a%d='%c' == b%d='%c': \"%s\"\n",
               bestA + 1,
               text1[len1 - 1 - bestA],
               bestB + 1,
               text2[bestB],
               merged.c_str());
        return merged;
    }

    // 没找到匹配，直接拼接
    std::string merged = text1 + text2;
    printf("    -> No match found, direct concatenation: \"%s\"\n",
           merged.c_str());
    return merged;
}

std::string OcrRecNPU::mergeAllTexts(
    const std::vector<RecognitionResultWithIndex>& results)
{
    if (results.empty()) {
        return "";
    }

    printf("\n========================================\n");
    printf("Merging %zu recognition results...\n", results.size());
    printf("========================================\n");

    // 按 boxIndex 和 subBoxIndex 排序
    std::vector<RecognitionResultWithIndex> sortedResults = results;
    std::sort(sortedResults.begin(),
              sortedResults.end(),
              [](const RecognitionResultWithIndex& a,
                 const RecognitionResultWithIndex& b) {
                  if (a.boxIndex != b.boxIndex) {
                      return a.boxIndex < b.boxIndex;
                  }
                  return a.subBoxIndex < b.subBoxIndex;
              });

    // 逐步合并
    std::string mergedText = sortedResults[0].result.text;

    for (size_t i = 1; i < sortedResults.size(); i++) {
        const auto& item = sortedResults[i];

        // 合并当前文本
        mergedText = mergeTexts(mergedText, item.result.text);
    }

    printf("\nFinal merged text: \"%s\"\n", mergedText.c_str());
    printf("========================================\n\n");

    return mergedText;
}
