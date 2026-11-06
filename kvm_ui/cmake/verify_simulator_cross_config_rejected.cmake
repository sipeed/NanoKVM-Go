if(NOT DEFINED SOURCE_DIR OR SOURCE_DIR STREQUAL "")
    message(FATAL_ERROR "SOURCE_DIR is required")
endif()
if(NOT DEFINED TEST_BINARY_DIR OR TEST_BINARY_DIR STREQUAL "")
    message(FATAL_ERROR "TEST_BINARY_DIR is required")
endif()

execute_process(
    COMMAND "${CMAKE_COMMAND}"
        -S "${SOURCE_DIR}"
        -B "${TEST_BINARY_DIR}"
        -DCMAKE_SYSTEM_NAME=Generic
        -DCMAKE_SYSTEM_PROCESSOR=armv7hf
        -DNANOKVM_GO_SIMULATOR_BUILD=ON
        -DNANOKVM_GO_RELEASE_BUILD=OFF
        -DNANOKVM_GO_DEBUG_BUILD=OFF
    RESULT_VARIABLE configure_result
    OUTPUT_VARIABLE configure_output
    ERROR_VARIABLE configure_error
)

set(configure_log "${configure_output}\n${configure_error}")
if(configure_result EQUAL 0)
    message(FATAL_ERROR
        "Simulator configuration unexpectedly accepted a cross target")
endif()
if(NOT configure_log MATCHES
   "NANOKVM_GO_SIMULATOR_BUILD is native-only and cannot be cross-compiled")
    message(FATAL_ERROR
        "Simulator cross-target rejection used an unexpected diagnostic:\n${configure_log}")
endif()
