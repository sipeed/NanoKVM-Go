cmake_minimum_required(VERSION 3.14)

project(
    libkvm
    VERSION 0.1.0
    DESCRIPTION "MJPEG & H264 Stream Library for Sipeed NanoKVMPro"
    # HOMEPAGE_URL "https://example.com/"
    LANGUAGES CXX C
)

# set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -march=armv8-a")
# set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -march=armv8-a")
#
# add_compile_options(
#     -fno-strict-aliasing
#     -fno-tree-vectorize
#     -O2  # 避免使用 -O3 或 -Ofast
# )
#
# set(CMAKE_POSITION_INDEPENDENT_CODE ON)


# ---- Declare library ----
find_package(fmt REQUIRED)
find_package(spdlog REQUIRED)
find_package(tomlplusplus REQUIRED)
find_package(ALSA REQUIRED)
find_package(Opus REQUIRED)

add_library(
    libkvm_libkvm SHARED
    source/kvmimpl.cpp
    source/control_client.cpp

    # video stream reader
    source/video_socket/unix_video_frame_reader.cpp

    # logger
    source/logger/logger_default.cpp

    # pcm capture
    source/pcm/alsa_abi_guard.c
    source/pcm/pcm_device.cpp
    source/pcm/opus_encoder.cpp
    source/pcm/pcm_capture.cpp

    # opus playback
    source/pcm/pcm_playback.cpp
    source/pcm/opus_decoder.cpp
    source/pcm/audio_player.cpp
)
add_library(libkvm::libkvm ALIAS libkvm_libkvm)

include(GenerateExportHeader)
generate_export_header(
    libkvm_libkvm
    BASE_NAME libkvm
    EXPORT_FILE_NAME export/libkvm/libkvm_export.hpp
    CUSTOM_CONTENT_FROM_VARIABLE pragma_suppress_c4251
)

if(NOT BUILD_SHARED_LIBS)
    target_compile_definitions(libkvm_libkvm PUBLIC LIBKVM_STATIC_DEFINE)
endif()

set_target_properties(
    libkvm_libkvm PROPERTIES
    VERSION "${PROJECT_VERSION}"
    SOVERSION "${PROJECT_VERSION_MAJOR}"
    EXPORT_NAME kvm
    OUTPUT_NAME kvm
)

target_include_directories(
    libkvm_libkvm ${warning_guard}
    PUBLIC
    "\$<BUILD_INTERFACE:${PROJECT_SOURCE_DIR}/include>"
)

target_include_directories(
    libkvm_libkvm SYSTEM
    PUBLIC
    "\$<BUILD_INTERFACE:${PROJECT_BINARY_DIR}/export>"
)

target_include_directories(
    libkvm_libkvm
    PRIVATE
    "${PROJECT_SOURCE_DIR}/source/"
    "${PROJECT_SOURCE_DIR}/source/logger/"
)

target_compile_features(libkvm_libkvm PUBLIC cxx_std_20)

find_package(Git QUIET)
set(GENERATED_DIR "${CMAKE_CURRENT_BINARY_DIR}/include/")
file(MAKE_DIRECTORY "${GENERATED_DIR}")

string(TIMESTAMP BUILD_TIMESTAMP "%Y-%m-%dT%H:%M:%S")
set(GIT_HASH "unknown")
set(GIT_DIRTY "unknown")

if(GIT_FOUND)
    execute_process(
        COMMAND ${GIT_EXECUTABLE} rev-parse HEAD
        WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
        OUTPUT_VARIABLE GIT_HASH
        OUTPUT_STRIP_TRAILING_WHITESPACE
        ERROR_QUIET
    )

    execute_process(
        COMMAND ${GIT_EXECUTABLE} diff-index --quiet HEAD --
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        RESULT_VARIABLE IS_DIRTY
    )
    if(IS_DIRTY)
        set(GIT_DIRTY "dirty")
    else()
        set(GIT_DIRTY "clean")
    endif()
endif()

configure_file(
    ${CMAKE_CURRENT_SOURCE_DIR}/source/version.h.in
    ${GENERATED_DIR}/version.h
)

add_library(buildinfo INTERFACE)
target_include_directories(buildinfo INTERFACE
    "$<BUILD_INTERFACE:${CMAKE_CURRENT_BINARY_DIR}/include>"
    "$<INSTALL_INTERFACE:include>"
)

target_compile_features(buildinfo INTERFACE cxx_std_20)

add_custom_command(
    OUTPUT "${GENERATED_DIR}/version.h"
    COMMAND ${CMAKE_COMMAND} -P "${CMAKE_CURRENT_LIST_FILE}"
    DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/source/version.h.in"
    COMMENT "Updating build info"
)

target_link_libraries(libkvm_libkvm PRIVATE
    fmt::fmt
    spdlog::spdlog
    tomlplusplus::tomlplusplus
    ALSA::ALSA
    Opus::opus
    rt
    buildinfo
)

target_link_options(libkvm_libkvm PRIVATE
    -Wl,-rpath=/opt/lib
)

include(CTest)
if(BUILD_TESTING)
    add_subdirectory(test)
endif()
