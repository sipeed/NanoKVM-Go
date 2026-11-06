cmake_minimum_required(VERSION 3.30)

project(
    nano_ocr
    VERSION 0.1.0
    DESCRIPTION "Nano OCR process for NanoKVM-Go"
    LANGUAGES CXX C
)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_compile_options(-ffile-prefix-map=${CMAKE_SOURCE_DIR}=.)
add_compile_options(-Wno-psabi)

set(NANOKVM_GO_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/.." CACHE PATH "NanoKVM-Go repository root")

set(OCR_OPENCV_DIR
    "${CMAKE_CURRENT_SOURCE_DIR}/prefix/opencv/lib/cmake/opencv4"
    CACHE PATH "OpenCVConfig.cmake directory for the OCR process")
set(OCR_AX_LIB_DIR
    "${NANOKVM_GO_ROOT}/kvm_vin/components/ax_utils/lib"
    CACHE PATH "AX SDK library directory")
set(OCR_AX_COMMON_INCLUDE_DIR
    "${NANOKVM_GO_ROOT}/kvm_vin/components/ax_utils/include"
    CACHE PATH "AX common include directory")
set(OCR_AX_ENGINE_INCLUDE_DIR
    "${CMAKE_CURRENT_SOURCE_DIR}/third_party/ax/include"
    CACHE PATH "AX engine include directory")

if(NOT EXISTS "${OCR_AX_ENGINE_INCLUDE_DIR}/ax_engine_api.h")
    message(FATAL_ERROR
        "Missing ${OCR_AX_ENGINE_INCLUDE_DIR}/ax_engine_api.h. "
        "This dependency is intentionally left for migration step 3. "
        "Populate OCR_AX_ENGINE_INCLUDE_DIR with the BSP MSP include directory "
        "or pass -DOCR_AX_ENGINE_INCLUDE_DIR=/path/to/include.")
endif()

if(NOT EXISTS "${OCR_AX_LIB_DIR}")
    message(FATAL_ERROR "OCR_AX_LIB_DIR does not exist: ${OCR_AX_LIB_DIR}")
endif()

set(OpenCV_DIR "${OCR_OPENCV_DIR}" CACHE PATH "OpenCVConfig.cmake directory" FORCE)
find_package(OpenCV 4.6.0 EXACT REQUIRED COMPONENTS core imgproc imgcodecs)
find_package(CURL REQUIRED)
find_package(nlohmann_json REQUIRED)

set(OCR_COMPONENT_DIR "${CMAKE_CURRENT_SOURCE_DIR}/components")

add_library(ocr_image STATIC
    "${OCR_COMPONENT_DIR}/ocr_image/ocr_image.cpp"
)
target_include_directories(ocr_image PUBLIC
    "${OCR_COMPONENT_DIR}/ocr_image"
)
target_link_libraries(ocr_image PUBLIC ${OpenCV_LIBS})

add_library(ocr_det STATIC
    "${OCR_COMPONENT_DIR}/ocr_det/ocr_det.cpp"
)
target_include_directories(ocr_det PUBLIC
    "${OCR_COMPONENT_DIR}/ocr_det"
    "${OCR_COMPONENT_DIR}/ocr_image"
    "${OCR_AX_COMMON_INCLUDE_DIR}"
    "${OCR_AX_ENGINE_INCLUDE_DIR}"
)
target_link_directories(ocr_det PUBLIC "${OCR_AX_LIB_DIR}")
target_link_libraries(ocr_det PUBLIC ocr_image ${OpenCV_LIBS})

add_library(ocr_rec STATIC
    "${OCR_COMPONENT_DIR}/ocr_rec/ocr_rec.cpp"
)
target_include_directories(ocr_rec PUBLIC
    "${OCR_COMPONENT_DIR}/ocr_rec"
    "${OCR_COMPONENT_DIR}/ocr_image"
    "${OCR_COMPONENT_DIR}/ocr_det"
    "${OCR_AX_COMMON_INCLUDE_DIR}"
    "${OCR_AX_ENGINE_INCLUDE_DIR}"
)
target_link_directories(ocr_rec PUBLIC "${OCR_AX_LIB_DIR}")
target_link_libraries(ocr_rec PUBLIC ocr_image ocr_det ${OpenCV_LIBS})

add_library(ocr_result_writer STATIC
    "${OCR_COMPONENT_DIR}/result_writer/result_writer.cpp"
)
target_include_directories(ocr_result_writer PUBLIC
    "${OCR_COMPONENT_DIR}/result_writer"
    "${OCR_COMPONENT_DIR}/ocr_det"
    "${OCR_COMPONENT_DIR}/ocr_rec"
)
target_link_libraries(ocr_result_writer PUBLIC ocr_det ocr_rec)

add_library(ocr_log STATIC
    "${OCR_COMPONENT_DIR}/log/log.cpp"
)
target_include_directories(ocr_log PUBLIC
    "${OCR_COMPONENT_DIR}/log"
)

add_library(ocr_snapshot_client STATIC
    "${OCR_COMPONENT_DIR}/snapshot_client/latest_frame_state.cpp"
    "${OCR_COMPONENT_DIR}/snapshot_client/snapshot_client.cpp"
)
target_include_directories(ocr_snapshot_client PUBLIC
    "${OCR_COMPONENT_DIR}/snapshot_client"
)
target_link_libraries(ocr_snapshot_client PUBLIC
    nlohmann_json::nlohmann_json
)

add_executable(nano_ocr_app
    "${CMAKE_CURRENT_SOURCE_DIR}/source/nano_ocr.cpp"
)
target_include_directories(nano_ocr_app PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/source"
)

set(OCR_AX_LIBS
    ax_sys
    ax_proton
    ax_ivps
    ax_venc
    ax_engine
    ax_interpreter
)

target_link_directories(nano_ocr_app PRIVATE "${OCR_AX_LIB_DIR}")
target_link_libraries(nano_ocr_app PRIVATE
    ocr_image
    ocr_det
    ocr_rec
    ocr_result_writer
    ocr_log
    ocr_snapshot_client
    ${OCR_AX_LIBS}
    pthread
    dl
    ${OpenCV_LIBS}
    CURL::libcurl
)

target_link_options(nano_ocr_app PRIVATE
    -Wl,-rpath=/opt/lib
)

set_target_properties(nano_ocr_app PROPERTIES
    SKIP_BUILD_RPATH TRUE
)

include(CTest)
if(BUILD_TESTING)
    find_package(Catch2 REQUIRED)
    add_subdirectory(test)
endif()
