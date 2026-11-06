find_package(Git QUIET)

set(GENERATED_DIR "${CMAKE_CURRENT_BINARY_DIR}/include/buildinfo")
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
    ${CMAKE_CURRENT_SOURCE_DIR}/include/buildinfo/version.hpp.in
    ${GENERATED_DIR}/version.hpp
)

add_library(buildinfo INTERFACE)
target_include_directories(buildinfo INTERFACE
    "$<BUILD_INTERFACE:${CMAKE_CURRENT_BINARY_DIR}/include>"
    "$<INSTALL_INTERFACE:include>"
)

target_compile_features(buildinfo INTERFACE cxx_std_20)

add_custom_command(
    OUTPUT "${GENERATED_DIR}/version.hpp"
    COMMAND ${CMAKE_COMMAND} -P "${CMAKE_CURRENT_LIST_FILE}"
    DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/version.h.in"
    COMMENT "Updating build info"
)
