cmake_minimum_required(VERSION 3.30)
include(FetchContent)

get_filename_component(prj_name ${CMAKE_CURRENT_SOURCE_DIR} NAME)

project(
    ${prj_name}
    VERSION 0.1.0
    DESCRIPTION "Short description"
    HOMEPAGE_URL "https://example.com/"
    LANGUAGES CXX C
)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_compile_options(-ffile-prefix-map=${CMAKE_SOURCE_DIR}=.)
add_compile_options(-Wno-psabi)

find_package(cxxopts REQUIRED)
find_package(Catch2 REQUIRED)
find_package(nlohmann_json)
find_package(httplib REQUIRED)

add_subdirectory(components/buildinfo)
add_subdirectory(components/global_para)
add_subdirectory(components/logger)
add_subdirectory(components/tools)
add_subdirectory(components/ax_utils)
add_subdirectory(components/lt7911_utils)
add_subdirectory(components/ffmpeg)

set(COMPONENTS
    buildinfo
    global_para
    logger
    tools
    ax_utils
    lt7911_utils
    ffmpeg
)

add_executable(${PROJECT_NAME}
    source/main.cpp
    source/capture_coordinator.cpp
    source/control_server.cpp
    source/encoder_settings.cpp
    source/frame_capture.cpp
    source/ocr_latest_frame_store.cpp
    source/snapshot_http_contract.cpp
    source/snapshot_http_server.cpp
    source/video_socket_server.cpp
    source/ax_runtime.cpp
    source/snapshot_socket_server.cpp
    source/snapshot_service.cpp
    source/timelapse_commit.cpp
    source/timelapse_index.cpp
    source/timelapse_service.cpp
    source/timelapse_ts_muxer.cpp
    source/lt7911_monitor.cpp
)

set_property(TARGET ${PROJECT_NAME} PROPERTY OUTPUT_NAME ${PROJECT_NAME})

target_include_directories( ${PROJECT_NAME} PRIVATE
    "${PROJECT_SOURCE_DIR}/source/include/"
)

target_compile_features(${PROJECT_NAME} PRIVATE cxx_std_20)

target_link_libraries(
    ${PROJECT_NAME} PRIVATE
    ${COMPONENTS}
    rt
    dl
    cxxopts::cxxopts
    Catch2::Catch2
    nlohmann_json::nlohmann_json
    httplib::httplib
)

target_link_options(${PROJECT_NAME} PRIVATE
    -Wl,-rpath=/opt/lib
)

set_target_properties(${PROJECT_NAME} PROPERTIES
    SKIP_BUILD_RPATH TRUE
)

target_compile_options(${PROJECT_NAME} PRIVATE
    $<$<COMPILE_LANGUAGE:CXX>:-fcoroutines>
)

include(CTest)
if(BUILD_TESTING)
    add_subdirectory(test)
endif()
