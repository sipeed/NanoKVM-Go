cmake_minimum_required(VERSION 3.30)

project(nano_ocr_opencv LANGUAGES C CXX)

# Keep OpenCV in a separate CMake project so its build options do not affect OCR.
set(BUILD_LIST "core,imgproc,imgcodecs" CACHE STRING "" FORCE)
set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(OPENCV_FORCE_3RDPARTY_BUILD ON CACHE BOOL "" FORCE)
set(OPENCV_LIB_INSTALL_PATH "lib" CACHE PATH "" FORCE)
set(OPENCV_CONFIG_INSTALL_PATH "lib/cmake/opencv4" CACHE PATH "" FORCE)

foreach(build_option
        BUILD_TESTS BUILD_PERF_TESTS BUILD_EXAMPLES BUILD_DOCS BUILD_JAVA
        BUILD_opencv_apps BUILD_opencv_python2 BUILD_opencv_python3
        OPENCV_GENERATE_SETUPVARS)
    set(${build_option} OFF CACHE BOOL "" FORCE)
endforeach()

# OCR only needs image codecs and image processing. Avoid video, GUI and
# optional acceleration dependencies, including any bundled FFmpeg binaries.
foreach(feature
        FFMPEG GSTREAMER GTK QT V4L IPP ITT CAROTENE EIGEN LAPACK OPENCL OPENEXR
        PROTOBUF QUIRC ADE)
    set(WITH_${feature} OFF CACHE BOOL "" FORCE)
endforeach()

include(FetchContent)
FetchContent_Declare(opencv
    URL https://codeload.github.com/opencv/opencv/tar.gz/refs/tags/4.6.0
    URL_HASH SHA256=1ec1cba65f9f20fe5a41fda1586e01c70ea0c9a6d7b67c9e13edf0cfe2239277
    DOWNLOAD_NAME 4.6.0.tar.gz
    DOWNLOAD_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../.cache/opencv"
    DOWNLOAD_EXTRACT_TIMESTAMP TRUE
)
FetchContent_MakeAvailable(opencv)
