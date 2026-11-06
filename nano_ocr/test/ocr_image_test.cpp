#include <cstdint>
#include <cstdlib>
#include <vector>

#include "ocr_image.h"

#include <catch2/catch_test_macros.hpp>
#include <opencv2/core.hpp>
#include <opencv2/imgcodecs.hpp>
#include <unistd.h>

TEST_CASE("OCRImage file and memory JPEG inputs decode identically",
          "[ocr][image]")
{
    cv::Mat source(3, 5, CV_8UC3, cv::Scalar(10, 20, 30));
    std::vector<std::uint8_t> jpeg;
    REQUIRE(cv::imencode(".jpg", source, jpeg));

    char temporaryPath[] = "/tmp/nano-ocr-image-XXXXXX";
    const int fd = mkstemp(temporaryPath);
    REQUIRE(fd >= 0);
    std::size_t written = 0;
    while (written < jpeg.size()) {
        const ssize_t count =
            write(fd, jpeg.data() + written, jpeg.size() - written);
        REQUIRE(count > 0);
        written += static_cast<std::size_t>(count);
    }
    REQUIRE(close(fd) == 0);

    OCRImage fileImage(temporaryPath);
    OCRImage memoryImage(jpeg);
    unlink(temporaryPath);

    REQUIRE(fileImage.isLoaded());
    REQUIRE(memoryImage.isLoaded());
    CHECK(memoryImage.getWidth() == fileImage.getWidth());
    CHECK(memoryImage.getHeight() == fileImage.getHeight());
    CHECK(memoryImage.getImage().channels == fileImage.getImage().channels);
    CHECK(cv::norm(memoryImage.getImage().data,
                   fileImage.getImage().data,
                   cv::NORM_INF)
          == 0.0);
}

TEST_CASE("OCRImage rejects empty and invalid encoded bytes", "[ocr][image]")
{
    OCRImage empty(std::vector<std::uint8_t> {});
    CHECK_FALSE(empty.isLoaded());

    OCRImage invalid(std::vector<std::uint8_t> {0x01, 0x02, 0x03});
    CHECK_FALSE(invalid.isLoaded());
}
