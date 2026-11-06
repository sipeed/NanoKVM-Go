#ifndef OCR_REC_H
#define OCR_REC_H

#include <memory>
#include <string>
#include <vector>

#include "ocr_det.h"  // 用于 BoundingBox 定义
#include "ocr_image.h"

// AXERA SDK headers
#include "ax_engine_api.h"
#include "ax_sys_api.h"

#define OCR_REC_CMM_ALIGN_SIZE   128
#define OCR_REC_CMM_SESSION_NAME "npu"

/**
 * @brief OCR 识别结果结构体
 *
 * 存储识别的文字内容、置信度等信息
 */
struct OCRRecResult {
    std::string text;  // 识别的文字
    float confidence;  // 平均置信度
    std::vector<float> charConfidences;  // 每个字符的置信度
    std::vector<int> charIndices;  // 字符在字典中的索引
    bool success;  // 是否成功
    std::string errorMessage;  // 错误信息

    /**
     * @brief 默认构造函数
     */
    OCRRecResult()
        : confidence(0.0f)
        , success(false)
    {}

    /**
     * @brief 检查识别是否成功
     */
    bool isValid() const
    { return success && !text.empty() && confidence > 0.0f; }
};

/**
 * @brief 带索引的识别结果结构体
 *
 * 用于跟踪每个识别结果对应的大框和小框索引
 */
struct RecognitionResultWithIndex {
    size_t boxIndex;  // 大框索引
    size_t subBoxIndex;  // 小框索引（0 表示大框，>0 表示小框）
    BoundingBox box;  // 边界框
    OCRRecResult result;  // 识别结果
};

/**
 * @brief OCR 识别类
 *
 * 功能：
 * 1. 加载 OCR 识别模型和字典文件
 * 2. 输入 48x320 的 BGR 图像，输出识别的文字内容
 * 3. 析构函数自动释放模型资源
 */
class OcrRecNPU {
  public:
    /**
     * @brief 默认构造函数
     */
    OcrRecNPU();

    /**
     * @brief 析构函数 - 释放模型资源
     */
    ~OcrRecNPU();

    /**
     * @brief 加载识别模型
     * @param modelPath 模型文件路径 (.axmodel)
     * @return true 加载成功，false 加载失败
     */
    bool loadModel(const std::string& modelPath);

    /**
     * @brief 加载字典文件
     * @param dictPath 字典文件路径 (.txt)
     * @return true 加载成功，false 加载失败
     */
    bool loadDictionary(const std::string& dictPath);

    /**
     * @brief 识别图像中的文字
     * @param inputImage 输入的 BGR 图像 (48x320)
     * @return OCRRecResult 识别结果
     */
    OCRRecResult recognize(const BGRImage& inputImage);

    /**
     * @brief 检查模型是否已加载
     * @return true 模型已加载，false 模型未加载
     */
    bool isModelLoaded() const { return modelLoaded; }

    /**
     * @brief 获取字典大小
     * @return int 字典字符数量
     */
    int getDictSize() const { return dictSize; }

    /**
     * @brief 根据索引获取字符
     * @param index 字符索引
     * @return std::string 字符字符串
     */
    std::string getChar(int index) const;

    /**
     * @brief 检查文件是否存在
     * @param filename 文件路径
     * @return true 文件存在，false 文件不存在
     */
    static bool fileExists(const std::string& filename);

    /**
     * @brief 智能拼接两个文本，查找重合部分并去重
     * 算法：text1 的最后 n 个字符与 text2 的前 n 个字符两两比较
     *      例如 checkRange=4，则比较：
     *      a1b1, a1b2, a1b3, a1b4,
     *      a2b1, a2b2, a2b3, a2b4,
     *      a3b1, a3b2, a3b3, a3b4,
     *      a4b1, a4b2, a4b3, a4b4
     *      共 16 次比较，找到相同的字符后，以此为重叠点进行合并
     * @param text1 第一个文本（A）
     * @param text2 第二个文本（B）
     * @param checkRange 检查范围（前后各多少个字符，默认 4 个）
     * @return 拼接后的文本
     */
    static std::string mergeTexts(const std::string& text1,
                                  const std::string& text2,
                                  size_t checkRange = 4);

    /**
     * @brief 智能合并所有识别结果中的连续文本
     * @param results 识别结果数组（包含 boxIndex 和 subBoxIndex）
     * @return 合并后的文本
     */
    static std::string mergeAllTexts(
        const std::vector<RecognitionResultWithIndex>& results);

  private:
    AX_ENGINE_HANDLE handle;  // NPU 引擎句柄
    bool modelLoaded;  // 模型是否已加载
    std::vector<char> modelBuffer;  // 模型文件缓冲区

    std::vector<std::string> dictionary;  // 字典
    int dictSize;  // 字典大小

    // 缓存的 IO 信息和数据（用于加速推理）
    AX_ENGINE_IO_INFO_T* cachedIOInfo;
    AX_ENGINE_IO_T cachedIOData;
    bool ioBuffersAllocated;

    /**
     * @brief 准备 IO 缓冲区
     */
    bool prepareIO(AX_ENGINE_IO_INFO_T* ioInfo, AX_ENGINE_IO_T* ioData);

    /**
     * @brief 释放 IO 缓冲区
     */
    void freeIO(AX_ENGINE_IO_T* io);

    /**
     * @brief CTC 解码 - 将模型输出转换为文字
     */
    std::string decodeCTC(const float* outputData,
                          int width,
                          int height,
                          std::vector<float>& charConfidences,
                          std::vector<int>& charIndices);

    /**
     * @brief 图像预处理 - 将输入图像转换为模型需要的格式
     */
    cv::Mat preprocessImage(const cv::Mat& inputImage,
                            int targetWidth = 320,
                            int targetHeight = 48);
};

#endif  // OCR_REC_H
