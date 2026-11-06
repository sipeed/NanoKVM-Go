add_library(
    global_para INTERFACE
)

target_include_directories(
    global_para INTERFACE
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
)
