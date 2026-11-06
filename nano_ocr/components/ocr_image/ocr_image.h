#ifndef OCR_IMAGE_H
#define OCR_IMAGE_H

#include <cstdint>
#include <string>
#include <vector>

#include <opencv2/opencv.hpp>

/**
 * @brief BGR 图像结构体
 *
 * 包装 OpenCV 的 cv::Mat，包含图像数据和元数据
 */
struct BGRImage {
    cv::Mat data;  ///< 图像数据（BGR 格式，3 通道，uint8）
    int width;  ///< 图像宽度（像素）
    int height;  ///< 图像高度（像素）
    int channels;  ///< 通道数（固定为 3，BGR 格式）

    /**
     * @brief 默认构造函数
     */
    BGRImage()
        : width(0)
        , height(0)
        , channels(3)
    {}

    /**
     * @brief 从 cv::Mat 创建 BGRImage
     * @param mat BGR 格式的 cv::Mat 对象
     */
    explicit BGRImage(const cv::Mat& mat)
        : data(mat)
        , width(mat.cols)
        , height(mat.rows)
        , channels(3)
    {}

    /**
     * @brief 检查图像是否有效
     * @return true 图像已加载且有效，false 图像为空
     */
    bool isValid() const { return !data.empty() && width > 0 && height > 0; }

    /**
     * @brief 释放图像内存
     */
    void release()
    {
        data.release();
        width = 0;
        height = 0;
    }
};

/**
 * @brief OCR 图像处理类
 *
 * 功能：
 * 1. 构造函数加载图像并自动转换为 BGR 格式
 * 2. 析构函数自动释放图像内存
 * 3. 提供获取 BGRImage 结构体的方法
 */
class OCRImage {
  public:
    /**
     * @brief 构造函数 - 从文件加载图像并转换为 BGR 格式
     * @param imagePath 图像文件路径（支持 PNG/JPG/BMP/TIFF 等格式）
     */
    explicit OCRImage(const std::string& imagePath);

    /**
     * @brief 构造函数 - 从内存中的 JPEG 字节解码为 BGR 图像
     * @param encodedImage 完整的 JPEG 编码字节
     */
    explicit OCRImage(const std::vector<std::uint8_t>& encodedImage);

    /**
     * @brief 析构函数 - 释放图像内存
     */
    ~OCRImage();

    /**
     * @brief 获取 BGR 图像结构体
     * @return BGRImage BGR 图像结构体
     */
    BGRImage getImage() const;

    /**
     * @brief 检查图像是否成功加载
     * @return true 图像已加载，false 图像为空
     */
    bool isLoaded() const;

    /**
     * @brief 获取图像宽度
     * @return int 图像宽度
     */
    int getWidth() const;

    /**
     * @brief 获取图像高度
     * @return int 图像高度
     */
    int getHeight() const;

  private:
    void assignLoadedImage(const cv::Mat& loadedImage,
                           const std::string& sourceDescription);

    BGRImage bgrImage;  ///< BGR 格式图像结构体
};

/**
 * @brief 保存 BGR 图像到文件
 * @param image BGR 图像结构体
 * @param outputPath 输出文件路径
 * @return true 保存成功，false 保存失败
 */
bool saveImage(const BGRImage& image, const std::string& outputPath);

/**
 * @brief 裁剪并缩放图像
 *
 * 从原图中裁剪指定区域并缩放到目标尺寸。如果裁剪区域的宽高比与目标尺寸不一致，
 * 会对裁剪后的图像进行等比缩放（letterbox），用黑色像素填充上下/左右边。
 *
 * @param image 输入 BGR 图像结构体
 * @param x1 裁剪区域左上角 x 坐标
 * @param y1 裁剪区域左上角 y 坐标
 * @param x2 裁剪区域右下角 x 坐标
 * @param y2 裁剪区域右下角 y 坐标
 * @param targetWidth 目标宽度
 * @param targetHeight 目标高度
 * @return BGRImage 裁剪缩放后的 BGR 图像结构体
 */
BGRImage cropAndResize(const BGRImage& image,
                       int x1,
                       int y1,
                       int x2,
                       int y2,
                       int targetWidth,
                       int targetHeight);

/**
 * @brief 从单个方框裁剪小框图像数组
 *
 * 根据方框高度决定裁剪策略：
 * - 高度 < 48：直接按固定长度裁剪（每 320 像素裁一次，重叠 48 像素）
 * - 高度 > 48：先等比缩小到高度 48，再按固定长度裁剪
 *
 * @param image 原始 BGR 图像结构体
 * @param x1 方框左上角 x 坐标
 * @param y1 方框左上角 y 坐标
 * @param x2 方框右下角 x 坐标
 * @param y2 方框右下角 y 坐标
 * @return std::vector<BGRImage> 裁剪后的小框图像数组
 */
std::vector<BGRImage> cropTextLinesFromBox(
    const BGRImage& image, int x1, int y1, int x2, int y2);

#endif  // OCR_IMAGE_H
