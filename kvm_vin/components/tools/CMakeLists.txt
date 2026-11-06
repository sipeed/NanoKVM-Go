find_package(libunwind REQUIRED)
find_package(tomlplusplus REQUIRED)
find_package(cppcoro REQUIRED)

add_library(
    tools OBJECT
    ${CMAKE_CURRENT_SOURCE_DIR}/src/segfault_stack_tracer.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/state_manager.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/system_uptime.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/termination_handler.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/toml_file_parser.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/generator.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/shell.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/scheduler.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/tag.cpp
)

target_include_directories(
    tools PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
)

target_link_libraries(
    tools PRIVATE
    logger
    global_para
    libunwind::libunwind
    tomlplusplus::tomlplusplus
    Catch2::Catch2
)

target_link_libraries(
    tools PUBLIC
    cppcoro::cppcoro
)

target_compile_features(tools PUBLIC cxx_std_20)
target_compile_options(tools PRIVATE
    $<$<COMPILE_LANGUAGE:CXX>:-fcoroutines>
)
