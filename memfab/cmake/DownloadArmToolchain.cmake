if(NOT DEFINED URL OR URL STREQUAL "")
    message(FATAL_ERROR "URL is required")
endif()
if(NOT DEFINED OUT OR OUT STREQUAL "")
    message(FATAL_ERROR "OUT is required")
endif()
if(NOT DEFINED EXTRACT_DIR OR EXTRACT_DIR STREQUAL "")
    message(FATAL_ERROR "EXTRACT_DIR is required")
endif()

# Custom command argument quoting can leave literal quotes in -D values on some generators.
string(REGEX REPLACE "^\"|\"$" "" URL "${URL}")
string(REGEX REPLACE "^\"|\"$" "" OUT "${OUT}")
string(REGEX REPLACE "^\"|\"$" "" EXTRACT_DIR "${EXTRACT_DIR}")

get_filename_component(out_dir "${OUT}" DIRECTORY)
file(MAKE_DIRECTORY "${out_dir}")
file(MAKE_DIRECTORY "${EXTRACT_DIR}")

if(NOT EXISTS "${OUT}")
    message(STATUS "Downloading ARM toolchain: ${URL}")
    file(DOWNLOAD
        "${URL}"
        "${OUT}"
        SHOW_PROGRESS
        STATUS download_status
        TLS_VERIFY ON
    )
    list(GET download_status 0 status_code)
    list(GET download_status 1 status_message)
    if(NOT status_code EQUAL 0)
        file(REMOVE "${OUT}")
        message(FATAL_ERROR "Download failed: ${status_code} ${status_message}")
    endif()
else()
    message(STATUS "ARM toolchain archive already exists: ${OUT}")
endif()

set(expected_dir "${EXTRACT_DIR}/gcc-arm-10.3-2021.07-x86_64-arm-none-linux-gnueabihf")
if(NOT EXISTS "${expected_dir}/bin/arm-none-linux-gnueabihf-gcc")
    message(STATUS "Extracting ARM toolchain to ${EXTRACT_DIR}")
    execute_process(
        COMMAND "${CMAKE_COMMAND}" -E tar xf "${OUT}"
        WORKING_DIRECTORY "${EXTRACT_DIR}"
        RESULT_VARIABLE extract_result
    )
    if(NOT extract_result EQUAL 0)
        message(FATAL_ERROR "Failed to extract ARM toolchain archive")
    endif()
else()
    message(STATUS "ARM toolchain already extracted: ${expected_dir}")
endif()

message(STATUS "ARM compiler: ${expected_dir}/bin/arm-none-linux-gnueabihf-gcc")
