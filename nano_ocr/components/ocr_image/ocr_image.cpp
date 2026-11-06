#include <cstdio>

#include "ocr_image.h"

/**
 * @brief 构造函数 - 从文件加载图像并转换为 BGR 格式
 */
OCRImage::OCRImage(const std::string& imagePath)
{
    // 使用 cv::IMREAD_COLOR 确保加载为 3 通道图像
    cv::Mat loadedImage = cv::imread(imagePath, cv::IMREAD_COLOR);

    if (loadedImage.empty()) {
        printf("Error: Failed to load image or image is empty: %s\n",
               imagePath.c_str());
        return;
    }

    assignLoadedImage(loadedImage, imagePath);
}

/**
 * @brief 构造函数 - 从内存中的 JPEG 字节解码为 BGR 图像
 */
OCRImage::OCRImage(const std::vector<std::uint8_t>& encodedImage)
{
    if (encodedImage.empty()) {
        printf("Error: Cannot decode an empty encoded image\n");
        return;
    }

    cv::Mat loadedImage = cv::imdecode(encodedImage, cv::IMREAD_COLOR);
    if (loadedImage.empty()) {
        printf("Error: Failed to decode JPEG bytes (%zu bytes)\n",
               encodedImage.size());
        return;
    }

    assignLoadedImage(loadedImage, "in-memory JPEG");
}

void OCRImage::assignLoadedImage(const cv::Mat& loadedImage,
                                 const std::string& sourceDescription)
{
    if (loadedImage.empty()) {
        return;
    }

    cv::Mat bgrMat;

    // 确保图像格式为 BGR
    if (loadedImage.channels() == 4) {
        // BGRA 转 BGR
        cv::cvtColor(loadedImage, bgrMat, cv::COLOR_BGRA2BGR);
        printf("Image loaded: %s (%dx%d, BGRA->BGR)\n",
               sourceDescription.c_str(),
               bgrMat.cols,
               bgrMat.rows);
    } else if (loadedImage.channels() == 1) {
        // 灰度图转 BGR
        cv::cvtColor(loadedImage, bgrMat, cv::COLOR_GRAY2BGR);
        printf("Image loaded: %s (%dx%d, GRAY->BGR)\n",
               sourceDescription.c_str(),
               bgrMat.cols,
               bgrMat.rows);
    } else if (loadedImage.channels() == 3) {
        // 已经是 BGR 格式，直接复制
        bgrMat = loadedImage.clone();
        printf("Image loaded: %s (%dx%d, BGR)\n",
               sourceDescription.c_str(),
               bgrMat.cols,
               bgrMat.rows);
    } else {
        printf("Error: Unsupported image channels: %d\n",
               loadedImage.channels());
        return;
    }

    // 创建 BGRImage 结构体
    bgrImage = BGRImage(bgrMat);
}

/**
 * @brief 析构函数 - 释放图像内存
 *
 * BGRImage 结构体会自动管理内存，无需手动操作
 */
OCRImage::~OCRImage()
{
    // BGRImage 析构时会自动释放 cv::Mat 内存
}

/**
 * @brief 获取 BGR 图像结构体
 */
BGRImage OCRImage::getImage() const { return bgrImage; }

/**
 * @brief 检查图像是否成功加载
 */
bool OCRImage::isLoaded() const { return bgrImage.isValid(); }

/**
 * @brief 获取图像宽度
 */
int OCRImage::getWidth() const { return bgrImage.width; }

/**
 * @brief 获取图像高度
 */
int OCRImage::getHeight() const { return bgrImage.height; }

/**
 * @brief 保存 BGR 图像到文件
 */
bool saveImage(const BGRImage& image, const std::string& outputPath)
{
    if (!image.isValid()) {
        printf("Error: Cannot save invalid or empty image\n");
        return false;
    }

    bool success = cv::imwrite(outputPath, image.data);

    if (success) {
        printf("Image saved: %s (%dx%d)\n",
               outputPath.c_str(),
               image.width,
               image.height);
    } else {
        printf("Error: Failed to save image: %s\n", outputPath.c_str());
    }

    return success;
}

/**
 * @brief 裁剪并缩放图像（带 letterbox，支持超出边界填充黑色）
 */
BGRImage cropAndResize(const BGRImage& image,
                       int x1,
                       int y1,
                       int x2,
                       int y2,
                       int targetWidth,
                       int targetHeight)
{
    // 参数验证
    if (!image.isValid()) {
        printf("Error: Input image is invalid\n");
        return BGRImage();
    }

    if (x1 >= x2 || y1 >= y2) {
        printf("Error: Invalid crop region (x1=%d, y1=%d, x2=%d, y2=%d)\n",
               x1,
               y1,
               x2,
               y2);
        return BGRImage();
    }

    // 步骤 1: 处理可能超出边界的情况
    int cropWidth = x2 - x1;
    int cropHeight = y2 - y1;

    // 检查是否有部分在原图范围内
    bool hasValidRegion =
        (x2 > 0 && y2 > 0 && x1 < image.width && y1 < image.height);

    cv::Mat croppedImage;
    int actualCropWidth, actualCropHeight;

    if (!hasValidRegion) {
        // 完全在原图范围外，创建全黑图像
        croppedImage = cv::Mat::zeros(cropHeight, cropWidth, CV_8UC3);
        actualCropWidth = cropWidth;
        actualCropHeight = cropHeight;
        printf("Crop region completely outside image, using black fill\n");
    } else {
        // 部分或全部在原图范围内
        // 计算与原图的有效交集
        int validX1 = std::max(0, x1);
        int validY1 = std::max(0, y1);
        int validX2 = std::min(image.width, x2);
        int validY2 = std::min(image.height, y2);

        // 裁剪有效区域
        cv::Rect validRect(
            validX1, validY1, validX2 - validX1, validY2 - validY1);
        cv::Mat validCrop = image.data(validRect).clone();

        // 如果完全在原图范围内，直接返回
        if (x1 == validX1 && y1 == validY1 && x2 == validX2 && y2 == validY2) {
            croppedImage = validCrop;
            actualCropWidth = cropWidth;
            actualCropHeight = cropHeight;
        } else {
            // 需要填充黑色边框
            croppedImage = cv::Mat::zeros(cropHeight, cropWidth, CV_8UC3);

            // 计算在原图中的偏移
            int offsetX = validX1 - x1;
            int offsetY = validY1 - y1;

            // 将有效区域复制到黑色背景上
            cv::Rect pasteRect(
                offsetX, offsetY, validCrop.cols, validCrop.rows);
            validCrop.copyTo(croppedImage(pasteRect));

            actualCropWidth = cropWidth;
            actualCropHeight = cropHeight;

            printf(
                "Crop region partially outside image, filled with black "
                "(offset: [%d,%d])\n",
                offsetX,
                offsetY);
        }
    }

    // 步骤 2: 计算缩放比例（等比缩放）
    float cropAspect = static_cast<float>(actualCropWidth) / actualCropHeight;
    float targetAspect = static_cast<float>(targetWidth) / targetHeight;

    cv::Mat resizedImage;
    int newWidth, newHeight;

    if (cropAspect > targetAspect) {
        // 裁剪区域更宽，按宽度缩放
        newWidth = targetWidth;
        newHeight = static_cast<int>(targetWidth / cropAspect);
    } else {
        // 裁剪区域更高，按高度缩放
        newHeight = targetHeight;
        newWidth = static_cast<int>(targetHeight * cropAspect);
    }

    cv::resize(croppedImage,
               resizedImage,
               cv::Size(newWidth, newHeight),
               0,
               0,
               cv::INTER_LINEAR);

    // 步骤 3: Letterbox 填充（如果需要）
    cv::Mat finalImage;
    if (newWidth == targetWidth && newHeight == targetHeight) {
        // 不需要填充
        finalImage = resizedImage;
    } else {
        // 创建黑色背景
        finalImage = cv::Mat::zeros(targetHeight, targetWidth, CV_8UC3);

        // 计算居中位置
        int offsetX = (targetWidth - newWidth) / 2;
        int offsetY = (targetHeight - newHeight) / 2;

        // 将缩放后的图像复制到中心位置
        cv::Rect pasteRect(offsetX, offsetY, newWidth, newHeight);
        resizedImage.copyTo(finalImage(pasteRect));
    }

    // 步骤 4: 创建并返回 BGRImage 结构体
    BGRImage result(finalImage);

    printf(
        "Crop and resize: [%d,%d,%d,%d] -> %dx%d (crop: %dx%d, resize: %dx%d, "
        "letterbox offset: [%d,%d])\n",
        x1,
        y1,
        x2,
        y2,
        targetWidth,
        targetHeight,
        actualCropWidth,
        actualCropHeight,
        newWidth,
        newHeight,
        (targetWidth - newWidth) / 2,
        (targetHeight - newHeight) / 2);

    return result;
}

/**
 * @brief 从单个方框裁剪小框图像数组
 */
std::vector<BGRImage> cropTextLinesFromBox(
    const BGRImage& image, int x1, int y1, int x2, int y2)
{
    std::vector<BGRImage> result;

    // 参数验证
    if (!image.isValid()) {
        printf("Error: Input image is invalid\n");
        return result;
    }

    if (x1 >= x2 || y1 >= y2) {
        printf("Error: Invalid box region (x1=%d, y1=%d, x2=%d, y2=%d)\n",
               x1,
               y1,
               x2,
               y2);
        return result;
    }

    // 计算方框尺寸
    int boxWidth = x2 - x1;
    int boxHeight = y2 - y1;

    printf("  Box size: %dx%d (width x height)\n", boxWidth, boxHeight);

    // 步骤 1: 从原图裁剪出完整的大框图像
    BGRImage fullBoxImage =
        cropAndResize(image, x1, y1, x2, y2, boxWidth, boxHeight);

    if (!fullBoxImage.isValid()) {
        printf("Error: Failed to crop full box image\n");
        return result;
    }

    // 步骤 2: 根据高度决定裁剪策略
    if (boxHeight < 48) {
        // 高度小于 48，直接按固定长度裁剪
        printf(
            "  Height < 48, cropping with fixed width (320px, overlap 48px), "
            "output size: 320x48\n");

        int cropWidth = 320;
        int overlap = 48;
        int step = cropWidth - overlap;

        // 如果方框宽度小于 320，直接返回完整大框（缩放到 320x48）
        if (boxWidth <= cropWidth) {
            printf("  Box width (%d) <= 320, resizing full box to 320x48\n",
                   boxWidth);
            BGRImage resizedFullBox =
                cropAndResize(fullBoxImage, 0, 0, boxWidth, boxHeight, 320, 48);
            result.push_back(resizedFullBox);
            return result;
        }

        // 按固定步长裁剪
        int numCrops = 0;
        for (int startX = 0; startX < boxWidth; startX += step) {
            int endX = std::min(startX + cropWidth, boxWidth);
            int subBoxWidth = endX - startX;

            // 裁剪小框并缩放到 320x48
            BGRImage subBox = cropAndResize(
                fullBoxImage, startX, 0, endX, boxHeight, 320, 48);

            if (subBox.isValid()) {
                result.push_back(subBox);
                numCrops++;
            }

            // 如果已经到达右边界，退出循环
            if (endX >= boxWidth) {
                break;
            }
        }

        printf("  Cropped %d sub-boxes from box (all resized to 320x48)\n",
               numCrops);

    } else {
        // 高度大于等于 48，先等比缩小到高度 48
        printf(
            "  Height >= 48, scaling to height 48 first, output size: "
            "320x48\n");

        float scaleFactor = 48.0f / boxHeight;
        int scaledWidth = static_cast<int>(boxWidth * scaleFactor);
        int scaledHeight = 48;

        printf("  Scale factor: %.2f, scaled size: %dx%d\n",
               scaleFactor,
               scaledWidth,
               scaledHeight);

        // 缩放到高度 48
        BGRImage scaledBox = cropAndResize(
            fullBoxImage, 0, 0, boxWidth, boxHeight, scaledWidth, scaledHeight);

        if (!scaledBox.isValid()) {
            printf("Error: Failed to scale box image\n");
            return result;
        }

        // 如果缩放后宽度小于 320，直接返回缩放后的图像（填充到 320x48）
        if (scaledWidth <= 320) {
            printf("  Scaled width (%d) <= 320, padding to 320x48\n",
                   scaledWidth);
            BGRImage paddedBox = cropAndResize(
                scaledBox, 0, 0, scaledWidth, scaledHeight, 320, 48);
            result.push_back(paddedBox);
            return result;
        }

        // 按固定步长裁剪
        int cropWidth = 320;
        int overlap = 48;
        int step = cropWidth - overlap;

        int numCrops = 0;
        for (int startX = 0; startX < scaledWidth; startX += step) {
            int endX = std::min(startX + cropWidth, scaledWidth);

            // 裁剪小框并缩放到 320x48
            BGRImage subBox = cropAndResize(
                scaledBox, startX, 0, endX, scaledHeight, 320, 48);

            if (subBox.isValid()) {
                result.push_back(subBox);
                numCrops++;
            }

            // 如果已经到达右边界，退出循环
            if (endX >= scaledWidth) {
                break;
            }
        }

        printf(
            "  Cropped %d sub-boxes from scaled box (all resized to 320x48)\n",
            numCrops);
    }

    // 将完整大框图像添加到结果最前面（缩放到 320x48）
    BGRImage resizedFullBox =
        cropAndResize(fullBoxImage, 0, 0, boxWidth, boxHeight, 320, 48);
    result.insert(result.begin(), resizedFullBox);

    printf("  Total: 1 full box (320x48) + %zu sub-boxes (320x48)\n",
           result.size() - 1);

    return result;
}
