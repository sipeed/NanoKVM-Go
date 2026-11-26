# Parent project does not export its library target, so this CML implicitly
# depends on being added from it, i.e. the testing is done only from the build
# tree and is not feasible from an install location

project(kvm_vin_Tests LANGUAGES CXX)

# ---- Dependencies ----

find_package(Catch2 REQUIRED)
find_package(spdlog REQUIRED)
find_package(Threads REQUIRED)
include(Catch)

# ---- Tests ----

add_executable(kvm_vin_test
    source/kvm_vin_test.cpp
    ../source/capture_coordinator.cpp
    ../source/encoder_settings.cpp
    ../source/ocr_latest_frame_store.cpp
    ../source/snapshot_http_contract.cpp
    ../source/snapshot_http_server.cpp
    ../source/snapshot_service.cpp
    ../source/snapshot_socket_server.cpp
    ../source/timelapse_commit.cpp
    ../source/timelapse_index.cpp
    ../source/timelapse_ts_muxer.cpp

    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/capture_coordinator.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/ocr_latest_frame_store.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/snapshot_http_server.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/snapshot_service.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/snapshot_socket_server.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/encoder/encoder_settings.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/generator/test.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/scheduler/scheduler.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/timelapse_ts_muxer.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/timelapse_commit.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/timelapse_index.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/tomlparser/tomlparser.cpp
)

target_include_directories(kvm_vin_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source/include"
)

target_link_libraries(
    kvm_vin_test PRIVATE
    ${COMPONENTS}
    Catch2::Catch2WithMain
    nlohmann_json::nlohmann_json
    httplib::httplib
)

target_compile_features(kvm_vin_test PRIVATE cxx_std_20)
target_compile_definitions(kvm_vin_test PRIVATE
    KVM_VIN_TEST_SNAPSHOT_INTERVAL_MS=1
)

# Keep the HTTP server tests runnable without loading AX/FFmpeg libraries.
add_executable(snapshot_http_server_test
    ../source/capture_coordinator.cpp
    ../source/snapshot_http_contract.cpp
    ../source/snapshot_http_server.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/capture/snapshot_http_server.cpp
)

target_include_directories(snapshot_http_server_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source/include"
    "${CMAKE_CURRENT_SOURCE_DIR}/../components/ax_utils/include"
    "${CMAKE_CURRENT_SOURCE_DIR}/../components/logger/include"
)

target_link_libraries(snapshot_http_server_test PRIVATE
    Catch2::Catch2WithMain
    nlohmann_json::nlohmann_json
    httplib::httplib
    spdlog::spdlog
    Threads::Threads
)

target_compile_features(snapshot_http_server_test PRIVATE cxx_std_20)

# Keep shared-memory permission and frame-flow checks independent from the
# hardware-linked kvm_vin test binary.
add_executable(shm_frame_test
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/catch/shm_frame.cpp
)

target_include_directories(shm_frame_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source/include"
)

target_link_libraries(shm_frame_test PRIVATE
    Catch2::Catch2WithMain
    rt
)

target_compile_features(shm_frame_test PRIVATE cxx_std_20)

# Exercise the real reconfigure/link transaction with test-only VIN and AX SDK
# boundaries, without linking the production ax_utils object or device SDKs.
add_executable(reconfigure_vin_pipeline_test
    ../components/ax_utils/src/ax_utils_venc.cpp
    ../components/ax_utils/src/ax_utils_link.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/ax_utils/reconfigure_vin_pipeline.cpp
)

target_include_directories(reconfigure_vin_pipeline_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../components/ax_utils/include"
    "${CMAKE_CURRENT_SOURCE_DIR}/../components/logger/include"
)

target_link_libraries(reconfigure_vin_pipeline_test PRIVATE
    Catch2::Catch2WithMain
    nlohmann_json::nlohmann_json
    spdlog::spdlog
    Threads::Threads
)

target_compile_features(reconfigure_vin_pipeline_test PRIVATE cxx_std_20)
target_compile_options(reconfigure_vin_pipeline_test PRIVATE
    -ffunction-sections
    -fdata-sections
)
target_link_options(reconfigure_vin_pipeline_test PRIVATE
    -Wl,--gc-sections
)

# Keep the commit guard independently runnable in cross-compiled environments.
# The complete kvm_vin_test binary links AX/FFmpeg dependencies that are not
# required to validate the lock and date-only helper.
add_executable(timelapse_commit_test
    ../source/timelapse_commit.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/timelapse_commit.cpp
)

target_include_directories(timelapse_commit_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source/include"
)

target_link_libraries(timelapse_commit_test PRIVATE
    Catch2::Catch2WithMain
)

target_compile_features(timelapse_commit_test PRIVATE cxx_std_20)

# Exercise TS output and first-use loading without the hardware SDK libraries.
add_executable(timelapse_ts_muxer_test
    ../source/timelapse_ts_muxer.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/timelapse_ts_muxer.cpp
)
add_executable(timelapse_ffmpeg_loading_test
    ../source/timelapse_ts_muxer.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/subtest/timelapse/ffmpeg_loading.cpp
)

foreach(TARGET timelapse_ts_muxer_test timelapse_ffmpeg_loading_test)
    target_include_directories(${TARGET} PRIVATE
        "${CMAKE_CURRENT_SOURCE_DIR}/../source/include"
    )
    target_link_libraries(${TARGET} PRIVATE ffmpeg Catch2::Catch2WithMain Threads::Threads)
    target_compile_features(${TARGET} PRIVATE cxx_std_20)
endforeach()

target_link_options(timelapse_ffmpeg_loading_test PRIVATE -Wl,--wrap=dlopen)

if(NOT CMAKE_CROSSCOMPILING)
    add_test(NAME kvm_vin_test COMMAND kvm_vin_test)
    add_test(NAME snapshot_http_server_test COMMAND snapshot_http_server_test)
    add_test(NAME shm_frame_test COMMAND shm_frame_test)
    add_test(NAME reconfigure_vin_pipeline_test COMMAND reconfigure_vin_pipeline_test)
    add_test(NAME timelapse_commit_test COMMAND timelapse_commit_test)
    add_test(NAME timelapse_ts_muxer_test COMMAND timelapse_ts_muxer_test)
    add_test(NAME timelapse_ffmpeg_loading_test COMMAND timelapse_ffmpeg_loading_test)
endif()
