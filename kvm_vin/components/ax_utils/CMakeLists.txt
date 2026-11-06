find_package(fmt REQUIRED)
find_package(spdlog REQUIRED)
find_package(nlohmann_json REQUIRED)

file(GLOB AX_UTILS_SO_FILES "${CMAKE_CURRENT_SOURCE_DIR}/lib/*.so")
foreach(SO_FILE ${AX_UTILS_SO_FILES})
    get_filename_component(LIB_NAME ${SO_FILE} NAME_WE)
    string(REGEX REPLACE "^lib" "" LIB_NAME ${LIB_NAME})
    list(APPEND AX_UTILS_LIBS ${LIB_NAME})
endforeach()

add_library(
    ax_utils OBJECT
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_core.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_pool.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_vin.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_ivps.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_venc.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_jenc.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_vdec.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ax_utils_link.cpp
)
target_include_directories(
    ax_utils PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
)
target_link_directories(ax_utils PUBLIC "${CMAKE_CURRENT_SOURCE_DIR}/lib")
target_link_libraries(ax_utils PUBLIC fmt::fmt spdlog::spdlog nlohmann_json::nlohmann_json logger dl ${AX_UTILS_LIBS})
target_compile_features(ax_utils PUBLIC cxx_std_20)
