cmake_minimum_required(VERSION 3.30)

get_filename_component(prj_name ${CMAKE_CURRENT_SOURCE_DIR} NAME)

project(
    ${prj_name}
    VERSION 0.1.0
    DESCRIPTION "Short description"
    HOMEPAGE_URL "https://example.com/"
    LANGUAGES CXX C
)

include(CTest)

option(NANOKVM_GO_RELEASE_BUILD "Build the product release binary with debug CLI disabled" OFF)
option(NANOKVM_GO_DEBUG_BUILD "Enable debug-only runtime diagnostics" OFF)
option(NANOKVM_GO_STRIP_RELEASE_BINARY "Strip unneeded symbols from the product binary after linking" OFF)
option(NANOKVM_GO_SIMULATOR_BUILD "Build the native x86_64 SDL simulator" OFF)
option(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE "Enable runtime-switchable background images" OFF)

if(NANOKVM_GO_DEBUG_BUILD AND NANOKVM_GO_RELEASE_BUILD)
    message(FATAL_ERROR "NANOKVM_GO_DEBUG_BUILD and NANOKVM_GO_RELEASE_BUILD cannot both be enabled")
endif()

if(NANOKVM_GO_SIMULATOR_BUILD)
    if(NANOKVM_GO_RELEASE_BUILD)
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD requires the non-release UI")
    endif()
    if(NANOKVM_GO_DEBUG_BUILD)
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD is incompatible with NANOKVM_GO_DEBUG_BUILD")
    endif()
    if(NOT CMAKE_HOST_SYSTEM_NAME STREQUAL "Linux")
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD is supported only on Linux hosts")
    endif()

    string(TOLOWER "${CMAKE_HOST_SYSTEM_PROCESSOR}" NANOKVM_GO_HOST_PROCESSOR)
    if(NOT NANOKVM_GO_HOST_PROCESSOR MATCHES "^(x86_64|amd64)$")
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD requires an x86_64 host, got '${CMAKE_HOST_SYSTEM_PROCESSOR}'")
    endif()
    if(CMAKE_CROSSCOMPILING)
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD is native-only and cannot be cross-compiled")
    endif()

    string(TOLOWER "${CMAKE_SYSTEM_PROCESSOR}" NANOKVM_GO_TARGET_PROCESSOR)
    if(NOT NANOKVM_GO_TARGET_PROCESSOR MATCHES "^(x86_64|amd64)$")
        message(FATAL_ERROR "NANOKVM_GO_SIMULATOR_BUILD requires an x86_64 target, got '${CMAKE_SYSTEM_PROCESSOR}'")
    endif()
endif()

set(NANOKVM_GO_SIMULATOR_BUILD_VALUE 0)
if(NANOKVM_GO_SIMULATOR_BUILD)
    set(NANOKVM_GO_SIMULATOR_BUILD_VALUE 1)
endif()
add_compile_definitions(NANOKVM_GO_SIMULATOR_BUILD=${NANOKVM_GO_SIMULATOR_BUILD_VALUE})

set(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE_VALUE 0)
if(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE)
    set(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE_VALUE 1)
endif()
add_compile_definitions(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE=${NANOKVM_GO_ENABLE_BACKGROUND_IMAGE_VALUE})

if(NANOKVM_GO_RELEASE_BUILD)
    add_compile_definitions(NANOKVM_GO_RELEASE_BUILD=1)
endif()

if(NANOKVM_GO_DEBUG_BUILD)
    add_compile_definitions(NANOKVM_GO_DEBUG_BUILD=1)
else()
    add_compile_definitions(NANOKVM_GO_DEBUG_BUILD=0)
endif()

# Compile-time log level for the SPDLOG_DEBUG/SPDLOG_TRACE call-site macros.
# Release removes debug-level call sites from the binary; every other build
# keeps the full runtime level range (the spdlog header default is INFO,
# which would otherwise silently strip debug logs from non-release builds).
if(NANOKVM_GO_RELEASE_BUILD)
    add_compile_definitions(SPDLOG_ACTIVE_LEVEL=SPDLOG_LEVEL_INFO)
else()
    add_compile_definitions(SPDLOG_ACTIVE_LEVEL=SPDLOG_LEVEL_TRACE)
endif()

set(NANOKVM_GO_APP_NAME "${PROJECT_NAME}")
set(NANOKVM_GO_APP_VERSION "${PROJECT_VERSION}")
set(NANOKVM_GO_GIT_COMMIT "unknown")
find_package(Git QUIET)
if(Git_FOUND)
    execute_process(
        COMMAND ${GIT_EXECUTABLE} rev-parse --short=12 HEAD
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        RESULT_VARIABLE NANOKVM_GO_GIT_RESULT
        OUTPUT_VARIABLE NANOKVM_GO_GIT_OUTPUT
        OUTPUT_STRIP_TRAILING_WHITESPACE
        ERROR_QUIET
    )
    if(NANOKVM_GO_GIT_RESULT EQUAL 0 AND NOT NANOKVM_GO_GIT_OUTPUT STREQUAL "")
        set(NANOKVM_GO_GIT_COMMIT "${NANOKVM_GO_GIT_OUTPUT}")
    endif()
endif()

set(NANOKVM_GO_GENERATED_INCLUDE_DIR "${CMAKE_CURRENT_BINARY_DIR}/generated/include")
file(MAKE_DIRECTORY "${NANOKVM_GO_GENERATED_INCLUDE_DIR}/lvgl_main")
configure_file(
    ${CMAKE_CURRENT_SOURCE_DIR}/cmake/build_info.hpp.in
    ${NANOKVM_GO_GENERATED_INCLUDE_DIR}/lvgl_main/build_info.hpp
    @ONLY
)

if(NANOKVM_GO_SIMULATOR_BUILD)
    set(NANOKVM_GO_SIMULATOR_STATE_DIR "${CMAKE_CURRENT_BINARY_DIR}/state")
    set(NANOKVM_GO_SIMULATOR_CONFIG_PATH "${NANOKVM_GO_SIMULATOR_STATE_DIR}/kvm_ui.toml")
    file(MAKE_DIRECTORY "${NANOKVM_GO_SIMULATOR_STATE_DIR}")
    file(CHMOD "${NANOKVM_GO_SIMULATOR_STATE_DIR}"
        PERMISSIONS
            OWNER_READ
            OWNER_WRITE
            OWNER_EXECUTE
    )
    set(NANOKVM_GO_SIMULATOR_APPS_PATH "${CMAKE_CURRENT_SOURCE_DIR}/tests/fixtures/apps")
    set(NANOKVM_GO_SIMULATOR_LOGICAL_WIDTH 270)
    set(NANOKVM_GO_SIMULATOR_LOGICAL_HEIGHT 240)
    set(NANOKVM_GO_SIMULATOR_WINDOW_ZOOM 2)
    set(NANOKVM_GO_SIMULATOR_WINDOW_TITLE "NanoKVM UI Simulator")
    configure_file(
        ${CMAKE_CURRENT_SOURCE_DIR}/cmake/simulator_build_config.hpp.in
        ${NANOKVM_GO_GENERATED_INCLUDE_DIR}/lvgl_main/simulator_build_config.hpp
        @ONLY
    )
endif()

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_compile_options(-ffile-prefix-map=${CMAKE_SOURCE_DIR}=.)
add_compile_options($<$<COMPILE_LANG_AND_ID:CXX,GNU>:-Wno-psabi>)
add_compile_options(-ffunction-sections -fdata-sections)
add_link_options(-Wl,--gc-sections)

add_subdirectory(components/lvgl)
add_subdirectory(components/ui)
add_subdirectory(components/lvgl_main)

if(BUILD_TESTING)
    add_subdirectory(tests)
endif()

set(COMPONENTS
    lvgl_main
    ui
)

find_package(spdlog REQUIRED)

add_executable(${PROJECT_NAME} src/main.cpp)

set_property(TARGET ${PROJECT_NAME} PROPERTY OUTPUT_NAME ${PROJECT_NAME})

target_compile_features(${PROJECT_NAME} PRIVATE cxx_std_20)

target_link_libraries(
    ${PROJECT_NAME} PRIVATE
    ${COMPONENTS}
    lvgl
    lvgl::thorvg
    spdlog::spdlog
)

if(NOT NANOKVM_GO_SIMULATOR_BUILD)
    target_link_libraries(${PROJECT_NAME} PRIVATE systemd)
endif()

target_link_options(${PROJECT_NAME} PRIVATE
    -Wl,-rpath=/opt/lib
)

if(NANOKVM_GO_STRIP_RELEASE_BINARY)
    add_custom_command(
        TARGET ${PROJECT_NAME}
        POST_BUILD
        COMMAND ${CMAKE_STRIP} --strip-unneeded $<TARGET_FILE:${PROJECT_NAME}>
        VERBATIM
    )
endif()
