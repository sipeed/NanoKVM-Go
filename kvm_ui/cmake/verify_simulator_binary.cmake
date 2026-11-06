if(NOT DEFINED SIMULATOR_BINARY OR SIMULATOR_BINARY STREQUAL "")
    message(FATAL_ERROR "SIMULATOR_BINARY is required")
endif()
if(NOT EXISTS "${SIMULATOR_BINARY}")
    message(FATAL_ERROR "Simulator binary does not exist: ${SIMULATOR_BINARY}")
endif()

if(NOT DEFINED NM_EXECUTABLE OR NM_EXECUTABLE STREQUAL "")
    find_program(NM_EXECUTABLE NAMES nm llvm-nm REQUIRED)
endif()

execute_process(
    COMMAND "${NM_EXECUTABLE}" -C "${SIMULATOR_BINARY}"
    RESULT_VARIABLE nm_result
    OUTPUT_VARIABLE simulator_symbols
    ERROR_VARIABLE nm_error
)
if(NOT nm_result EQUAL 0)
    message(FATAL_ERROR
        "Failed to inspect simulator binary with ${NM_EXECUTABLE}: ${nm_error}")
endif()

set(forbidden_symbols
    "NanoKvmRuntimeService"
    "HttpApiServer"
    "UILVGLApp::run_app_session"
    "ExitBar"
)

foreach(forbidden_symbol IN LISTS forbidden_symbols)
    string(FIND "${simulator_symbols}" "${forbidden_symbol}" symbol_index)
    if(NOT symbol_index EQUAL -1)
        message(FATAL_ERROR
            "Simulator binary contains forbidden target symbol: ${forbidden_symbol}")
    endif()
endforeach()
