#include <cmath>
#include <cstdio>
#include <fstream>

#include "ocr_det.h"

/**
 * @brief 计算图像分块裁剪参数
 */
std::vector<CropRegion> calculateCropRegions(int imageWidth,
                                             int imageHeight,
                                             float scaleFactor,
                                             int overlapPixels,
                                             int targetWidth,
                                             int targetHeight)
{
    std::vector<CropRegion> regions;

    // 参数验证
    if (imageWidth <= 0 || imageHeight <= 0) {
        printf("Error: Invalid image dimensions (%dx%d)\n",
               imageWidth,
               imageHeight);
        return regions;
    }

    if (scaleFactor <= 0) {
        printf("Error: Invalid scale factor (%.2f)\n", scaleFactor);
        return regions;
    }

    if (overlapPixels < 0) {
        printf("Error: Invalid overlap pixels (%d)\n", overlapPixels);
        return regions;
    }

    if (targetWidth <= 0 || targetHeight <= 0) {
        printf("Error: Invalid target dimensions (%dx%d)\n",
               targetWidth,
               targetHeight);
        return regions;
    }

    // 步骤 1: 计算缩小后的图像尺寸
    int scaledWidth = static_cast<int>(imageWidth / scaleFactor);
    int scaledHeight = static_cast<int>(imageHeight / scaleFactor);

    printf("Image tiling calculation:\n");
    printf("  Original size: %dx%d\n", imageWidth, imageHeight);
    printf("  Scale factor: %.2f\n", scaleFactor);
    printf("  Scaled size: %dx%d\n", scaledWidth, scaledHeight);
    printf("  Target tile size: %dx%d\n", targetWidth, targetHeight);
    printf("  Overlap: %d pixels\n", overlapPixels);
    printf("\n");

    // 步骤 2: 计算在原图上的步长
    // 目标尺寸是在缩小后的图像上的尺寸，需要映射回原图
    int stepX = static_cast<int>((targetWidth - overlapPixels) * scaleFactor);
    int stepY = static_cast<int>((targetHeight - overlapPixels) * scaleFactor);

    // 确保步长为正
    if (stepX <= 0)
        stepX = 1;
    if (stepY <= 0)
        stepY = 1;

    // 步骤 3: 计算裁剪区域大小（在原图上）
    int cropWidth = static_cast<int>(targetWidth * scaleFactor);
    int cropHeight = static_cast<int>(targetHeight * scaleFactor);

    // 步骤 4: 生成裁剪区域（允许超出边界，后面会填充黑色）
    int index = 0;
    for (int y = 0; y < imageHeight; y += stepY) {
        for (int x = 0; x < imageWidth; x += stepX) {
            int x1 = x;
            int y1 = y;
            int x2 = x + cropWidth;  // 允许超出边界
            int y2 = y + cropHeight;  // 允许超出边界

            regions.emplace_back(index, x1, y1, x2, y2);
            index++;
        }

        // 如果当前行已经到达底部，跳出循环
        if (y + stepY >= imageHeight) {
            break;
        }
    }

    printf("  Generated %zu crop regions\n", regions.size());

    // 打印前几个区域的信息（用于调试）
    for (size_t i = 0; i < std::min(regions.size(), static_cast<size_t>(5));
         i++) {
        const CropRegion& region = regions[i];
        printf("    Region %d: [%d, %d, %d, %d] (%dx%d)\n",
               region.index,
               region.x1,
               region.y1,
               region.x2,
               region.y2,
               region.width,
               region.height);
    }
    if (regions.size() > 5) {
        printf("    ... (%zu more regions)\n", regions.size() - 5);
    }
    printf("\n");

    return regions;
}

/**
 * @brief 将 float32 热力图二值化为 uint8
 */
cv::Mat binarizeHeatmap(const cv::Mat& heatmap, float threshold)
{
    cv::Mat binary;
    cv::threshold(heatmap, binary, threshold, 255.0, cv::THRESH_BINARY);
    binary.convertTo(binary, CV_8UC1);
    return binary;
}

/**
 * @brief 将多个二值热力图拼接为原图尺寸（二值版本）
 */
cv::Mat mergeBinaryHeatmaps(const BGRImage& originalImage,
                            const std::vector<cv::Mat>& heatmaps,
                            const std::vector<CropRegion>& cropRegions)
{
    printf("Merging %zu binary heatmaps to original size...\n",
           heatmaps.size());

    // 创建二值全图（初始化为 0）
    cv::Mat fullBinary =
        cv::Mat::zeros(originalImage.height, originalImage.width, CV_8UC1);

    // 使用 OR 操作合并所有二值热力图
    for (size_t i = 0; i < heatmaps.size(); i++) {
        const CropRegion& region = cropRegions[i];
        printf("  Merging region %zu/%zu [%d,%d,%d,%d]...\n",
               i + 1,
               heatmaps.size(),
               region.x1,
               region.y1,
               region.x2,
               region.y2);

        // 将二值热力图缩放到裁剪区域的实际尺寸
        cv::Mat heatmapResized;
        cv::resize(heatmaps[i],
                   heatmapResized,
                   cv::Size(region.width, region.height),
                   0,
                   0,
                   cv::INTER_NEAREST);  // 二值图使用最近邻插值

        // 计算在原图中的有效区域
        int x1 = std::max(0, region.x1);
        int y1 = std::max(0, region.y1);
        int x2 = std::min(originalImage.width, region.x2);
        int y2 = std::min(originalImage.height, region.y2);

        // 计算在热力图中的对应区域
        int heatmapX1 = std::max(0, -region.x1);
        int heatmapY1 = std::max(0, -region.y1);
        int heatmapWidth = x2 - x1;
        int heatmapHeight = y2 - y1;

        if (heatmapWidth <= 0 || heatmapHeight <= 0) {
            printf("    Warning: Invalid region, skipping...\n");
            continue;
        }

        // 确保 ROI 不越界
        heatmapWidth = std::min(heatmapWidth, heatmapResized.cols - heatmapX1);
        heatmapHeight =
            std::min(heatmapHeight, heatmapResized.rows - heatmapY1);

        if (heatmapWidth <= 0 || heatmapHeight <= 0) {
            printf("    Warning: ROI out of bounds, skipping...\n");
            continue;
        }

        // 使用 OR 操作合并二值图
        for (int y = 0; y < heatmapHeight; y++) {
            for (int x = 0; x < heatmapWidth; x++) {
                uchar srcVal =
                    heatmapResized.at<uchar>(heatmapY1 + y, heatmapX1 + x);
                if (srcVal > 0) {
                    fullBinary.at<uchar>(y1 + y, x1 + x) = 255;
                }
            }
        }
    }

    printf("Binary heatmap merged to full size: %dx%d\n",
           fullBinary.cols,
           fullBinary.rows);

    return fullBinary;
}

/**
 * @brief 从二值热力图中提取边界框
 */
std::vector<BoundingBox> extractBoxesFromBinaryHeatmap(
    const cv::Mat& binaryHeatmap, int threshold, int minArea, int expandPercent)
{
    printf("Extracting boxes from binary heatmap...\n");
    printf("  Threshold: %d, Min area: %d, Expand: %d%%\n",
           threshold,
           minArea,
           expandPercent);
    printf("  Binary heatmap dimensions: %dx%d, type: %d\n",
           binaryHeatmap.cols,
           binaryHeatmap.rows,
           binaryHeatmap.type());

    // 使用 connectedComponentsWithStats 直接获取统计信息
    cv::Mat labels;
    cv::Mat stats;
    cv::Mat centroids;
    int numLabels = cv::connectedComponentsWithStats(
        binaryHeatmap, labels, stats, centroids, 8, CV_32S);
    printf("  Found %d connected components (including background)\n",
           numLabels - 1);

    std::vector<BoundingBox> boxes;

    // stats 的每一行对应一个连通域，列分别为：[left, top, width, height, area]
    for (int label = 1; label < numLabels; label++) {
        int* statRow = stats.ptr<int>(label);
        int left = statRow[0];
        int top = statRow[1];
        int width = statRow[2];
        int height = statRow[3];
        int area = statRow[4];

        // 过滤小区域
        if (area < minArea) {
            continue;
        }

        // 扩展边界框：使用与旧版本相同的逻辑
        // 垂直方向扩展：扩展到原来的 (100 + expandPercent)%
        // 水平方向扩展：扩展与垂直方向相同的像素数
        int expandPixels = 0;
        if (expandPercent > 0) {
            // 垂直方向扩展的像素数
            int expandY = height * expandPercent / 100;
            // 上下左右各扩展一半
            expandPixels = expandY / 2;
        }

        // 应用扩展（确保不超出图像边界）
        int right = left + width;
        int bottom = top + height;

        if (expandPixels > 0) {
            top = std::max(0, top - expandPixels);
            bottom = std::min(binaryHeatmap.rows, bottom + expandPixels);
            left = std::max(0, left - expandPixels);
            right = std::min(binaryHeatmap.cols, right + expandPixels);
        }

        BoundingBox bbox;
        bbox.x1 = left;
        bbox.y1 = top;
        bbox.x2 = right;
        bbox.y2 = bottom;
        bbox.score = 1.0f;

        boxes.push_back(bbox);
    }

    printf("  Extracted %zu valid boxes\n", boxes.size());

    return boxes;
}

/**
 * @brief 从热力图中提取边界框
 */
std::vector<BoundingBox> extractBoxesFromHeatmap(const HeatmapData& heatmap,
                                                 float threshold,
                                                 int minArea,
                                                 int expandPercent)
{
    if (!heatmap.checkValid()) {
        printf("Error: Invalid heatmap data\n");
        return std::vector<BoundingBox>();
    }

    printf("Extracting boxes from heatmap...\n");
    printf("  Threshold: %.3f, Min area: %d, Expand: %d%%\n",
           threshold,
           minArea,
           expandPercent);
    printf("  Heatmap dimensions: %dx%d, data pointer: %p\n",
           heatmap.width,
           heatmap.height,
           (void*)heatmap.data);

    // 检查数据有效性
    if (heatmap.data == nullptr) {
        printf("  ERROR: heatmap.data is nullptr!\n");
        return std::vector<BoundingBox>();
    }

    // 尝试读取前几个像素值（使用指针算术）
    float* pData = heatmap.data;
    printf("  Trying to read first pixel...\n");
    fflush(stdout);
    float val0 = *pData;
    printf("  First pixel value: %.4f\n", val0);
    fflush(stdout);
    printf("  First 5 pixel values: %.4f, %.4f, %.4f, %.4f, %.4f\n",
           val0,
           *(pData + 1),
           *(pData + 2),
           *(pData + 3),
           *(pData + 4));
    fflush(stdout);

    // 创建 cv::Mat 包装器（不共享数据，而是复制）
    cv::Mat heatmapMat =
        cv::Mat(heatmap.height, heatmap.width, CV_32FC1, heatmap.data).clone();
    printf("  cv::Mat created: %dx%d, continuous: %d\n",
           heatmapMat.cols,
           heatmapMat.rows,
           heatmapMat.isContinuous());

    // 二值化：大于阈值的设为 255，其余设为 0
    // 注意：我们的数据已经是 0.0 或 1.0，所以二值化后直接转换为 uint8
    printf("  Starting binarization (%dx%d pixels)...\n",
           heatmap.height,
           heatmap.width);
    cv::Mat binary(heatmap.height, heatmap.width, CV_8UC1);
    int processedCount = 0;
    for (int y = 0; y < heatmap.height; y++) {
        for (int x = 0; x < heatmap.width; x++) {
            float val = heatmapMat.at<float>(y, x);
            binary.at<uchar>(y, x) = (val > threshold) ? 255 : 0;
            processedCount++;
            if (processedCount == 1000) {
                printf("    Processed 1000 pixels, no crash yet...\n");
            }
        }
    }
    printf("  Binarization completed (manual loop), processed %d pixels\n",
           processedCount);

    // 查找连通域
    cv::Mat labels;
    int numLabels = cv::connectedComponents(binary, labels, 8, CV_32S);

    printf("  Found %d connected components (including background)\n",
           numLabels);

    // 优化：使用一次遍历收集所有连通域的信息
    // 为每个 label 维护：最小/最大 x,y 坐标，像素点列表，总分
    std::vector<int> minX(numLabels, INT_MAX);
    std::vector<int> maxX(numLabels, INT_MIN);
    std::vector<int> minY(numLabels, INT_MAX);
    std::vector<int> maxY(numLabels, INT_MIN);
    std::vector<double> sumVal(numLabels, 0.0);
    std::vector<int> pixelCount(numLabels, 0);

    // 一次遍历全图，收集所有连通域的信息
    printf("  Starting connected components analysis (%dx%d pixels)...\n",
           labels.rows,
           labels.cols);
    int processedPixels = 0;
    for (int y = 0; y < labels.rows; y++) {
        for (int x = 0; x < labels.cols; x++) {
            int label = labels.at<int>(y, x);
            if (label == 0)
                continue;  // 跳过背景

            // 更新边界
            if (x < minX[label])
                minX[label] = x;
            if (x > maxX[label])
                maxX[label] = x;
            if (y < minY[label])
                minY[label] = y;
            if (y > maxY[label])
                maxY[label] = y;

            // 累加分数
            sumVal[label] += heatmapMat.at<float>(y, x);
            pixelCount[label]++;

            processedPixels++;
            if (processedPixels % 1000000 == 0) {
                printf("    Processed %d non-background pixels...\n",
                       processedPixels);
            }
        }
    }
    printf(
        "  Connected components analysis completed. Total non-background "
        "pixels: %d\n",
        processedPixels);

    // 生成边界框
    std::vector<BoundingBox> boxes;
    for (int label = 1; label < numLabels; label++) {
        // 检查是否有像素
        if (pixelCount[label] == 0)
            continue;

        // 计算原始边界框
        int x1 = minX[label];
        int y1 = minY[label];
        int x2 = maxX[label] + 1;  // +1 因为 boundingRect 是开区间
        int y2 = maxY[label] + 1;

        int width = x2 - x1;
        int height = y2 - y1;
        int area = width * height;

        // 检查面积
        if (area < minArea) {
            continue;
        }

        // 计算扩展量
        // 垂直方向扩展：扩展到原来的 (100 + expandPercent)%
        // 水平方向扩展：扩展与垂直方向相同的像素数
        int expandPixels = 0;
        if (expandPercent > 0) {
            // 垂直方向扩展的像素数
            int expandY = height * expandPercent / 100;
            // 上下各扩展一半
            expandPixels = expandY / 2;
        }

        // 应用扩展（确保不超出图像边界）
        if (expandPixels > 0) {
            y1 = std::max(0, y1 - expandPixels);
            y2 = std::min(heatmap.height, y2 + expandPixels);
            x1 = std::max(0, x1 - expandPixels);
            x2 = std::min(heatmap.width, x2 + expandPixels);
        }

        // 计算平均置信度
        float avgScore = sumVal[label] / pixelCount[label];

        // 添加到结果
        BoundingBox box(x1, y1, x2, y2, avgScore);
        boxes.push_back(box);
    }

    printf("  Extracted %zu valid boxes\n", boxes.size());

    return boxes;
}

/**
 * @brief 将热力图可视化并叠加到原图上
 */
BGRImage visualizeHeatmap(const BGRImage& originalImage,
                          const cv::Mat& heatmap,
                          const CropRegion& cropRegion,
                          float alpha)
{
    // 创建原图的副本
    cv::Mat resultImage;
    originalImage.data.copyTo(resultImage);

    // 步骤 1: 将热力图归一化到 0-255 范围
    cv::Mat heatmapNorm;
    double minVal, maxVal;
    cv::minMaxLoc(heatmap, &minVal, &maxVal);

    if (maxVal - minVal > 0) {
        cv::Mat heatmapScaled = (heatmap - minVal) / (maxVal - minVal) * 255.0;
        heatmapScaled.convertTo(heatmapNorm, CV_8UC1);
    } else {
        heatmapNorm = cv::Mat::zeros(heatmap.size(), CV_8UC1);
    }

    // 步骤 2: 将热力图转换为伪彩色图（使用 JET 色彩映射）
    cv::Mat heatmapColor;
    cv::applyColorMap(heatmapNorm, heatmapColor, cv::COLORMAP_JET);

    // 步骤 3: 调整伪彩色图大小以匹配裁剪区域
    cv::Mat heatmapResized;
    if (heatmapColor.cols != cropRegion.width
        || heatmapColor.rows != cropRegion.height) {
        cv::resize(heatmapColor,
                   heatmapResized,
                   cv::Size(cropRegion.width, cropRegion.height),
                   0,
                   0,
                   cv::INTER_LINEAR);
    } else {
        heatmapResized = heatmapColor;
    }

    // 步骤 4: 将热力图叠加到原图的对应位置
    // 计算裁剪区域在原图中的位置
    int x1 = std::max(0, cropRegion.x1);
    int y1 = std::max(0, cropRegion.y1);
    int x2 = std::min(originalImage.width, cropRegion.x2);
    int y2 = std::min(originalImage.height, cropRegion.y2);

    // 计算热力图中对应的区域（处理边界情况）
    int heatmapX1 = std::max(0, -cropRegion.x1);
    int heatmapY1 = std::max(0, -cropRegion.y1);
    int heatmapX2 = heatmapX1 + (x2 - x1);
    int heatmapY2 = heatmapY1 + (y2 - y1);

    // 确保不越界
    if (x1 >= x2 || y1 >= y2) {
        printf("Warning: Invalid crop region for visualization\n");
        BGRImage result;
        result.data = resultImage;
        result.width = resultImage.cols;
        result.height = resultImage.rows;
        result.channels = resultImage.channels();
        return result;
    }

    // 创建 ROI 区域
    cv::Rect roi(x1, y1, x2 - x1, y2 - y1);
    cv::Rect heatmapRoi(heatmapX1, heatmapY1, x2 - x1, y2 - y1);

    // 步骤 5: 使用加权叠加
    cv::Mat overlay;
    heatmapResized(heatmapRoi).convertTo(overlay, CV_32FC3, 1.0 / 255.0);

    cv::Mat originalROI = resultImage(roi);
    for (int y = 0; y < overlay.rows; y++) {
        for (int x = 0; x < overlay.cols; x++) {
            cv::Vec3b& origPixel = originalROI.at<cv::Vec3b>(y, x);
            cv::Vec3f heatmapPixel = overlay.at<cv::Vec3f>(y, x);

            // BGR 通道加权混合
            origPixel[0] =
                static_cast<uchar>((1.0f - alpha) * origPixel[0]
                                   + alpha * heatmapPixel[0] * 255.0f);
            origPixel[1] =
                static_cast<uchar>((1.0f - alpha) * origPixel[1]
                                   + alpha * heatmapPixel[1] * 255.0f);
            origPixel[2] =
                static_cast<uchar>((1.0f - alpha) * origPixel[2]
                                   + alpha * heatmapPixel[2] * 255.0f);
        }
    }

    // 创建返回的 BGRImage
    BGRImage result;
    result.data = resultImage;
    result.width = resultImage.cols;
    result.height = resultImage.rows;
    result.channels = resultImage.channels();

    printf(
        "Heatmap visualization created: %dx%d\n", result.width, result.height);

    return result;
}

/**
 * @brief 将多个热力图拼接为原图尺寸
 */
HeatmapData mergeHeatmaps(const BGRImage& originalImage,
                          const std::vector<cv::Mat>& heatmaps,
                          const std::vector<CropRegion>& cropRegions)
{
    printf("Merging %zu heatmaps to original size...\n", heatmaps.size());

    // 检查第一个热力图的类型，决定使用 float 还是 uint8 合并
    bool isBinary = (heatmaps[0].type() == CV_8UC1);

    if (isBinary) {
        // 二值化热力图合并：使用最大值操作
        printf("  Using binary merge mode (uint8)\n");
        cv::Mat fullHeatmap =
            cv::Mat::zeros(originalImage.height, originalImage.width, CV_8UC1);

        for (size_t i = 0; i < heatmaps.size(); i++) {
            const CropRegion& region = cropRegions[i];
            printf("  Merging region %zu/%zu [%d,%d,%d,%d]...\n",
                   i + 1,
                   heatmaps.size(),
                   region.x1,
                   region.y1,
                   region.x2,
                   region.y2);

            // 将二值热力图缩放到裁剪区域的实际尺寸
            cv::Mat heatmapResized;
            cv::resize(heatmaps[i],
                       heatmapResized,
                       cv::Size(region.width, region.height),
                       0,
                       0,
                       cv::INTER_NEAREST);  // 二值图使用最近邻插值

            // 计算在原图中的有效区域
            int x1 = std::max(0, region.x1);
            int y1 = std::max(0, region.y1);
            int x2 = std::min(originalImage.width, region.x2);
            int y2 = std::min(originalImage.height, region.y2);

            // 计算在热力图中的对应区域
            int heatmapX1 = std::max(0, -region.x1);
            int heatmapY1 = std::max(0, -region.y1);
            int heatmapWidth = x2 - x1;
            int heatmapHeight = y2 - y1;

            if (heatmapWidth <= 0 || heatmapHeight <= 0) {
                printf("    Warning: Invalid region, skipping...\n");
                continue;
            }

            // 确保 ROI 不越界
            heatmapWidth =
                std::min(heatmapWidth, heatmapResized.cols - heatmapX1);
            heatmapHeight =
                std::min(heatmapHeight, heatmapResized.rows - heatmapY1);

            if (heatmapWidth <= 0 || heatmapHeight <= 0) {
                printf("    Warning: ROI out of bounds, skipping...\n");
                continue;
            }

            // 使用最大值合并二值图（手动循环，避免 cv::max 的 ROI 问题）
            for (int y = 0; y < heatmapHeight; y++) {
                for (int x = 0; x < heatmapWidth; x++) {
                    uchar srcVal =
                        heatmapResized.at<uchar>(heatmapY1 + y, heatmapX1 + x);
                    uchar& dstVal = fullHeatmap.at<uchar>(y1 + y, x1 + x);
                    if (srcVal > dstVal) {
                        dstVal = srcVal;
                    }
                }
            }
        }

        printf("Binary heatmap merged to full size: %dx%d\n",
               fullHeatmap.cols,
               fullHeatmap.rows);

        // 转换为 float32 以便后续处理（extractBoxesFromHeatmap 需要 float
        // 输入） 二值图值为 0 或 255，转换后为 0.0 或 1.0
        // 直接分配内存并填充，避免 cv::Mat 的内存对齐问题
        int totalPixels = fullHeatmap.cols * fullHeatmap.rows;
        printf("  Allocating float array: %d pixels\n", totalPixels);

        float* floatData = new float[totalPixels];
        for (int y = 0; y < fullHeatmap.rows; y++) {
            const uchar* srcRow = fullHeatmap.ptr<uchar>(y);
            float* dstRow = floatData + y * fullHeatmap.cols;
            for (int x = 0; x < fullHeatmap.cols; x++) {
                dstRow[x] = (srcRow[x] > 128) ? 1.0f : 0.0f;
            }
        }

        printf("  Float conversion completed, data pointer=%p\n",
               (void*)floatData);

        // 创建 HeatmapData 结构体
        HeatmapData result;
        result.width = fullHeatmap.cols;
        result.height = fullHeatmap.rows;
        result.channels = 1;
        result.isValid = true;
        result.data = floatData;

        printf("  HeatmapData creation completed\n");

        return result;
    } else {
        // float32 热力图合并：使用累加平均
        printf("  Using float merge mode (float32)\n");
        cv::Mat fullHeatmap =
            cv::Mat::zeros(originalImage.height, originalImage.width, CV_32FC1);
        cv::Mat countMap =
            cv::Mat::zeros(originalImage.height, originalImage.width, CV_32FC1);

        for (size_t i = 0; i < heatmaps.size(); i++) {
            const CropRegion& region = cropRegions[i];
            printf("  Merging region %zu/%zu [%d,%d,%d,%d]...\n",
                   i + 1,
                   heatmaps.size(),
                   region.x1,
                   region.y1,
                   region.x2,
                   region.y2);

            // 将热力图缩放到裁剪区域的实际尺寸
            cv::Mat heatmapResized;
            cv::resize(heatmaps[i],
                       heatmapResized,
                       cv::Size(region.width, region.height),
                       0,
                       0,
                       cv::INTER_LINEAR);

            // 计算在原图中的有效区域
            int x1 = std::max(0, region.x1);
            int y1 = std::max(0, region.y1);
            int x2 = std::min(originalImage.width, region.x2);
            int y2 = std::min(originalImage.height, region.y2);

            // 计算在热力图中的对应区域
            int heatmapX1 = std::max(0, -region.x1);
            int heatmapY1 = std::max(0, -region.y1);
            int heatmapWidth = x2 - x1;
            int heatmapHeight = y2 - y1;

            if (heatmapWidth <= 0 || heatmapHeight <= 0) {
                printf("    Warning: Invalid region, skipping...\n");
                continue;
            }

            // 将热力图数据累加到全图中
            for (int y = 0; y < heatmapHeight; y++) {
                for (int x = 0; x < heatmapWidth; x++) {
                    float heatmapVal =
                        heatmapResized.at<float>(heatmapY1 + y, heatmapX1 + x);
                    fullHeatmap.at<float>(y1 + y, x1 + x) += heatmapVal;
                    countMap.at<float>(y1 + y, x1 + x) += 1.0f;
                }
            }
        }

        // 对重叠区域取平均值
        cv::Mat mergedHeatmap;
        cv::divide(fullHeatmap, countMap, mergedHeatmap, 1.0, CV_32FC1);

        printf("Heatmap merged to full size: %dx%d\n",
               mergedHeatmap.cols,
               mergedHeatmap.rows);

        // 创建 HeatmapData 结构体
        HeatmapData result;
        result.width = mergedHeatmap.cols;
        result.height = mergedHeatmap.rows;
        result.channels = 1;
        result.isValid = true;

        // 分配内存并复制数据
        result.data = new float[result.width * result.height];
        memcpy(result.data,
               mergedHeatmap.data,
               result.width * result.height * sizeof(float));

        return result;
    }
}

/**
 * @brief 将热力图可视化并叠加到原图上
 */
BGRImage visualizeMergedHeatmap(const BGRImage& originalImage,
                                const HeatmapData& mergedHeatmap,
                                float alpha,
                                const std::vector<BoundingBox>& boxes)
{
    if (!mergedHeatmap.checkValid()) {
        printf("Error: Invalid heatmap data\n");
        BGRImage result;
        result.data = originalImage.data.clone();
        result.width = originalImage.width;
        result.height = originalImage.height;
        result.channels = originalImage.channels;
        return result;
    }

    printf("Visualizing merged heatmap...\n");

    // 创建 cv::Mat 包装器（不复制数据）
    cv::Mat heatmapMat(mergedHeatmap.height,
                       mergedHeatmap.width,
                       CV_32FC1,
                       mergedHeatmap.data);

    // 统计热力图值范围
    double minVal, maxVal;
    cv::minMaxLoc(heatmapMat, &minVal, &maxVal);
    printf("  Heatmap value range: [%.6f, %.6f]\n", minVal, maxVal);

    // 与原图叠加
    cv::Mat resultImage;
    originalImage.data.copyTo(resultImage);

    // 遍历热力图，只在有数值的位置叠加红色
    int redCount = 0;
    for (int y = 0; y < heatmapMat.rows; y++) {
        for (int x = 0; x < heatmapMat.cols; x++) {
            float heatmapVal = heatmapMat.at<float>(y, x);

            // 只要有数值（大于 0），就显示为半透明红色
            if (heatmapVal > 0.0f) {
                cv::Vec3b& origPixel = resultImage.at<cv::Vec3b>(y, x);

                // 红色：B=0, G=0, R=255
                // BGR 通道加权混合
                origPixel[0] = static_cast<uchar>((1.0f - alpha)
                                                  * origPixel[0]);  // B 通道
                origPixel[1] = static_cast<uchar>((1.0f - alpha)
                                                  * origPixel[1]);  // G 通道
                origPixel[2] = static_cast<uchar>((1.0f - alpha) * origPixel[2]
                                                  + alpha * 255.0f);  // R 通道

                redCount++;
            }
        }
    }

    printf("  Red pixels: %d / %d (%.2f%%)\n",
           redCount,
           heatmapMat.rows * heatmapMat.cols,
           (float)redCount / (heatmapMat.rows * heatmapMat.cols) * 100.0f);

    // 绘制绿色边界框
    if (!boxes.empty()) {
        printf("  Drawing %zu bounding boxes...\n", boxes.size());
        for (size_t i = 0; i < boxes.size(); i++) {
            const BoundingBox& box = boxes[i];
            cv::rectangle(resultImage,
                          cv::Point(box.x1, box.y1),
                          cv::Point(box.x2, box.y2),
                          cv::Scalar(0, 255, 0),
                          2);
        }
        printf("  Bounding boxes drawn\n");
    }

    // 转换为 BGRImage
    BGRImage result;
    result.data = resultImage;
    result.width = resultImage.cols;
    result.height = resultImage.rows;
    result.channels = 3;

    return result;
}

/**
 * @brief 将合并后的二值热力图可视化并叠加到原图
 */
BGRImage visualizeMergedHeatmap(const BGRImage& originalImage,
                                const cv::Mat& binaryHeatmap,
                                float alpha,
                                const std::vector<BoundingBox>& boxes)
{
    printf("Visualizing binary merged heatmap...\n");

    // 与原图叠加
    cv::Mat resultImage;
    originalImage.data.copyTo(resultImage);

    // 遍历二值热力图，只在有数值的位置叠加红色
    int redCount = 0;
    for (int y = 0; y < binaryHeatmap.rows; y++) {
        for (int x = 0; x < binaryHeatmap.cols; x++) {
            uchar heatmapVal = binaryHeatmap.at<uchar>(y, x);

            // 二值图：大于 128 就显示为红色
            if (heatmapVal > 128) {
                cv::Vec3b& origPixel = resultImage.at<cv::Vec3b>(y, x);

                // 红色：B=0, G=0, R=255
                // BGR 通道加权混合
                origPixel[0] = static_cast<uchar>((1.0f - alpha)
                                                  * origPixel[0]);  // B 通道
                origPixel[1] = static_cast<uchar>((1.0f - alpha)
                                                  * origPixel[1]);  // G 通道
                origPixel[2] = static_cast<uchar>((1.0f - alpha) * origPixel[2]
                                                  + alpha * 255.0f);  // R 通道

                redCount++;
            }
        }
    }

    printf(
        "  Red pixels: %d / %d (%.2f%%)\n",
        redCount,
        binaryHeatmap.rows * binaryHeatmap.cols,
        (float)redCount / (binaryHeatmap.rows * binaryHeatmap.cols) * 100.0f);

    // 绘制绿色边界框
    if (!boxes.empty()) {
        printf("  Drawing %zu bounding boxes...\n", boxes.size());
        for (size_t i = 0; i < boxes.size(); i++) {
            const BoundingBox& box = boxes[i];
            cv::rectangle(resultImage,
                          cv::Point(box.x1, box.y1),
                          cv::Point(box.x2, box.y2),
                          cv::Scalar(0, 255, 0),
                          2);
        }
        printf("  Bounding boxes drawn\n");
    }

    // 转换为 BGRImage
    BGRImage result;
    result.data = resultImage;
    result.width = resultImage.cols;
    result.height = resultImage.rows;
    result.channels = 3;

    return result;
}

/**
 * @brief 检查文件是否存在
 */
bool OcrDetNPU::fileExists(const std::string& filepath)
{
    std::ifstream file(filepath);
    return file.good();
}

/**
 * @brief 构造函数 - 加载模型到 NPU
 */
OcrDetNPU::OcrDetNPU(const std::string& modelPath)
    : modelLoaded(false)
    , modelPath(modelPath)
    , handle(0)
{
    printf("========================================\n");
    printf("Loading OCR Detection Model to NPU\n");
    printf("========================================\n");
    printf("Model path: %s\n", modelPath.c_str());

    // 检查模型文件是否存在
    if (!fileExists(modelPath)) {
        printf("Error: Model file not found: %s\n", modelPath.c_str());
        return;
    }

    // 读取模型文件
    printf("Reading model file...\n");
    std::ifstream modelFile(modelPath, std::ios::binary | std::ios::ate);
    if (!modelFile.is_open()) {
        printf("Error: Failed to open model file: %s\n", modelPath.c_str());
        return;
    }

    std::streamsize fileSize = modelFile.tellg();
    modelFile.seekg(0, std::ios::beg);

    std::vector<char> modelBuffer(fileSize);
    if (!modelFile.read(modelBuffer.data(), fileSize)) {
        printf("Error: Failed to read model file\n");
        return;
    }

    printf("Model loaded: %s (%ld bytes)\n", modelPath.c_str(), fileSize);

    // 初始化 AXERA SDK
    printf("Initializing AXERA SDK...\n");
    AX_S32 ret = AX_SYS_Init();
    if (ret != 0) {
        printf("Error: AX_SYS_Init failed: 0x%x\n", ret);
        return;
    }

    // 初始化 ENGINE
    printf("Initializing ENGINE...\n");
    AX_ENGINE_NPU_ATTR_T npuAttr;
    memset(&npuAttr, 0, sizeof(AX_ENGINE_NPU_ATTR_T));
    npuAttr.eHardMode = AX_ENGINE_VIRTUAL_NPU_DISABLE;
    ret = AX_ENGINE_Init(&npuAttr);
    if (ret != 0) {
        printf("Error: AX_ENGINE_Init failed: 0x%x\n", ret);
        AX_SYS_Deinit();
        return;
    }

    // 创建 handle
    printf("Creating model handle...\n");
    ret =
        AX_ENGINE_CreateHandle(&handle, modelBuffer.data(), modelBuffer.size());
    if (ret != 0 || !handle) {
        printf("Error: AX_ENGINE_CreateHandle failed: 0x%x\n", ret);
        AX_ENGINE_Deinit();
        AX_SYS_Deinit();
        return;
    }

    // 创建 context
    printf("Creating context...\n");
    ret = AX_ENGINE_CreateContext(handle);
    if (ret != 0) {
        printf("Error: AX_ENGINE_CreateContext failed: 0x%x\n", ret);
        AX_ENGINE_DestroyHandle(handle);
        AX_ENGINE_Deinit();
        AX_SYS_Deinit();
        return;
    }

    modelLoaded = true;
    printf("Model successfully loaded to NPU!\n");
    printf("========================================\n\n");
}

/**
 * @brief 析构函数 - 释放 NPU 资源
 */
OcrDetNPU::~OcrDetNPU()
{
    if (handle) {
        printf("Releasing NPU resources...\n");
        AX_ENGINE_DestroyHandle(handle);
        handle = 0;
    }
    AX_ENGINE_Deinit();
    AX_SYS_Deinit();
    printf("NPU resources released.\n");
}

/**
 * @brief 检查模型是否成功加载
 */
bool OcrDetNPU::isModelLoaded() const { return modelLoaded; }

/**
 * @brief 准备 IO 缓冲区
 */
bool OcrDetNPU::prepareIO(AX_ENGINE_IO_INFO_T* ioInfo, AX_ENGINE_IO_T* ioData)
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
                                     OCR_DET_CMM_ALIGN_SIZE,
                                     (const AX_S8*)OCR_DET_CMM_SESSION_NAME);

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
                                     OCR_DET_CMM_ALIGN_SIZE,
                                     (const AX_S8*)OCR_DET_CMM_SESSION_NAME);

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

/**
 * @brief 释放 IO 缓冲区
 */
void OcrDetNPU::freeIO(AX_ENGINE_IO_T* ioData)
{
    if (!ioData)
        return;

    for (AX_U32 i = 0; i < ioData->nInputSize; i++) {
        if (ioData->pInputs[i].pVirAddr) {
            AX_SYS_MemFree(ioData->pInputs[i].phyAddr,
                           ioData->pInputs[i].pVirAddr);
        }
    }

    for (AX_U32 i = 0; i < ioData->nOutputSize; i++) {
        if (ioData->pOutputs[i].pVirAddr) {
            AX_SYS_MemFree(ioData->pOutputs[i].phyAddr,
                           ioData->pOutputs[i].pVirAddr);
        }
    }

    delete[] ioData->pInputs;
    delete[] ioData->pOutputs;
}

/**
 * @brief 预处理图像（letterbox 缩放到目标尺寸）
 */
cv::Mat OcrDetNPU::preprocessImage(const BGRImage& image,
                                   int targetWidth,
                                   int targetHeight)
{
    // 计算缩放比例
    float scale = std::min(static_cast<float>(targetWidth) / image.width,
                           static_cast<float>(targetHeight) / image.height);

    int newWidth = static_cast<int>(image.width * scale);
    int newHeight = static_cast<int>(image.height * scale);

    // 缩放图像
    cv::Mat resizedImage;
    cv::resize(image.data,
               resizedImage,
               cv::Size(newWidth, newHeight),
               0,
               0,
               cv::INTER_LINEAR);

    // 创建 letterbox 图像（灰色背景）
    cv::Mat letterboxImage = cv::Mat::zeros(targetHeight, targetWidth, CV_8UC3);

    // 计算居中位置
    int offsetX = (targetWidth - newWidth) / 2;
    int offsetY = (targetHeight - newHeight) / 2;

    // 复制图像到中心
    cv::Rect roi(offsetX, offsetY, newWidth, newHeight);
    resizedImage.copyTo(letterboxImage(roi));

    return letterboxImage;
}

/**
 * @brief 检测文本区域，输出热力图
 */
cv::Mat OcrDetNPU::detect(const BGRImage& image,
                          int targetWidth,
                          int targetHeight)
{
    cv::Mat heatmap;

    if (!modelLoaded) {
        printf("Error: Model not loaded\n");
        return heatmap;
    }

    if (!image.isValid()) {
        printf("Error: Input image is invalid\n");
        return heatmap;
    }

    printf("Running OCR detection...\n");
    printf("  Input image: %dx%d\n", image.width, image.height);
    printf("  Target size: %dx%d\n", targetWidth, targetHeight);

    // 预处理图像
    cv::Mat letterboxImage = preprocessImage(image, targetWidth, targetHeight);
    printf(
        "  Letterbox image: %dx%d\n", letterboxImage.cols, letterboxImage.rows);

    // 获取 IO 信息
    AX_ENGINE_IO_INFO_T* ioInfo = nullptr;
    AX_S32 ret = AX_ENGINE_GetIOInfo(handle, &ioInfo);
    if (ret != 0 || !ioInfo) {
        printf("Error: AX_ENGINE_GetIOInfo failed: 0x%x\n", ret);
        return heatmap;
    }

    printf("  Input tensor shape: [");
    for (AX_U32 i = 0; i < ioInfo->pInputs[0].nShapeSize; i++) {
        printf("%d", (int)ioInfo->pInputs[0].pShape[i]);
        if (i < ioInfo->pInputs[0].nShapeSize - 1)
            printf(", ");
    }
    printf("]\n");

    // 准备 IO 缓冲区
    AX_ENGINE_IO_T ioData;
    if (!prepareIO(ioInfo, &ioData)) {
        printf("Error: Failed to prepare IO\n");
        return heatmap;
    }

    // 复制图像数据到输入缓冲区
    AX_U8* inputData = (AX_U8*)ioData.pInputs[0].pVirAddr;
    AX_U32 inputSize = targetWidth * targetHeight * 3;
    memcpy(inputData, letterboxImage.data, inputSize);

    // 运行推理
    printf("  Running inference on NPU...\n");
    ret = AX_ENGINE_RunSync(handle, &ioData);
    if (ret != 0) {
        printf("Error: AX_ENGINE_RunSync failed: 0x%x\n", ret);
        freeIO(&ioData);
        return heatmap;
    }
    printf("  Inference completed!\n");

    // 获取输出数据
    float* outputData = (float*)ioData.pOutputs[0].pVirAddr;
    AX_U32 outputHeight = ioInfo->pOutputs[0].pShape[2];  // 480
    AX_U32 outputWidth = ioInfo->pOutputs[0].pShape[3];  // 640

    printf("  Output tensor shape: [");
    for (AX_U32 i = 0; i < ioInfo->pOutputs[0].nShapeSize; i++) {
        printf("%d", (int)ioInfo->pOutputs[0].pShape[i]);
        if (i < ioInfo->pOutputs[0].nShapeSize - 1)
            printf(", ");
    }
    printf("]\n");

    // 保存热力图数据
    heatmap =
        cv::Mat(outputHeight, outputWidth, CV_32FC1, (void*)outputData).clone();

    // 释放 IO 缓冲区
    freeIO(&ioData);

    printf("  Heatmap generated: %dx%d, type=CV_32FC1 (float32)\n",
           outputHeight,
           outputWidth);
    printf("OCR detection completed!\n\n");

    return heatmap;
}
