find_package(fmt REQUIRED)
find_package(spdlog REQUIRED)

add_library(
    logger OBJECT
    ${CMAKE_CURRENT_SOURCE_DIR}/src/file_sink.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/default.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/default2.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/tools.cpp
)
target_include_directories(
    logger PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
)
target_link_libraries(logger PUBLIC fmt::fmt spdlog::spdlog)
target_compile_features(logger PUBLIC cxx_std_20)
