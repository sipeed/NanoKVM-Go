find_package(fmt REQUIRED)
find_package(spdlog REQUIRED)

add_library(
    lt7911_utils OBJECT
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lt7911_utils.cpp
)
target_include_directories(
    lt7911_utils PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
)
target_link_libraries(lt7911_utils PUBLIC fmt::fmt spdlog::spdlog logger)
target_compile_features(lt7911_utils PUBLIC cxx_std_20)
