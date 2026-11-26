cmake_minimum_required(VERSION 3.14)

project(libkvmTests LANGUAGES CXX C)

find_package(Catch2 REQUIRED)
find_package(Threads REQUIRED)
include(Catch)

# ---- Tests ----

add_executable(libkvm_test
    source/libkvm_test.cpp
)
target_link_libraries(
    libkvm_test PRIVATE
    libkvm_libkvm
    Catch2::Catch2WithMain
)
target_compile_features(libkvm_test PRIVATE cxx_std_20)

add_executable(libkvm_pcm_test
    source/pcm_capture_test.cpp
    source/opus_encoder_test.cpp
    source/opus_decoder_test.cpp
    source/pcm_playback_test.cpp
    source/audio_player_test.cpp
    ../source/pcm/pcm_device.cpp
    ../source/pcm/opus_encoder.cpp
    ../source/pcm/pcm_capture.cpp
    ../source/pcm/opus_decoder.cpp
    ../source/pcm/pcm_playback.cpp
    ../source/pcm/audio_player.cpp
)
target_include_directories(
    libkvm_pcm_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source"
)
target_link_libraries(
    libkvm_pcm_test PRIVATE
    Catch2::Catch2WithMain
    Threads::Threads
    ALSA::ALSA
    Opus::opus
    rt
)
target_compile_features(libkvm_pcm_test PRIVATE cxx_std_20)

add_executable(libkvm_c_header_test
    source/kvm_vision_header_test.c
)
target_include_directories(
    libkvm_c_header_test PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../include"
)
target_compile_features(libkvm_c_header_test PRIVATE c_std_11)

add_executable(libkvm_audio_playback_probe
    source/audio_playback_probe_test.cpp
    ../source/pcm/opus_encoder.cpp
)
target_include_directories(
    libkvm_audio_playback_probe PRIVATE
    "${CMAKE_CURRENT_SOURCE_DIR}/../source"
)
target_link_libraries(
    libkvm_audio_playback_probe PRIVATE
    libkvm_libkvm
    Catch2::Catch2WithMain
    Opus::opus
)
target_compile_features(libkvm_audio_playback_probe PRIVATE cxx_std_20)

add_test(
    NAME libkvm_abi_non_hardware
    COMMAND libkvm_test
)
add_test(
    NAME libkvm_pcm_non_hardware
    COMMAND libkvm_pcm_test
)
add_test(
    NAME libkvm_c_header
    COMMAND libkvm_c_header_test
)
