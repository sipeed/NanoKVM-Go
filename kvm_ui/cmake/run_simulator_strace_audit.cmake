foreach(required_variable
        STRACE_EXECUTABLE
        SIMULATOR_SMOKE_BINARY
        SIMULATOR_LIBRARY_PATH
        TRACE_FILE)
    if(NOT DEFINED ${required_variable} OR "${${required_variable}}" STREQUAL "")
        message(FATAL_ERROR "${required_variable} is required")
    endif()
endforeach()

if(NOT EXISTS "${STRACE_EXECUTABLE}")
    message(FATAL_ERROR "strace does not exist: ${STRACE_EXECUTABLE}")
endif()
if(NOT EXISTS "${SIMULATOR_SMOKE_BINARY}")
    message(FATAL_ERROR
        "Simulator smoke binary does not exist: ${SIMULATOR_SMOKE_BINARY}")
endif()

execute_process(
    COMMAND "${CMAKE_COMMAND}" -E env
        "LD_LIBRARY_PATH=${SIMULATOR_LIBRARY_PATH}"
        "SDL_VIDEODRIVER=dummy"
        "SDL_RENDER_DRIVER=software"
        "SDL_AUDIODRIVER=dummy"
        "XMODIFIERS=@im=none"
        "${STRACE_EXECUTABLE}"
        -f
        -e trace=file,process,network
        -o "${TRACE_FILE}"
        "${SIMULATOR_SMOKE_BINARY}"
    RESULT_VARIABLE audit_result
    OUTPUT_VARIABLE audit_output
    ERROR_VARIABLE audit_error
)
if(NOT audit_result EQUAL 0)
    message(FATAL_ERROR
        "Simulator strace audit failed (${audit_result}): ${audit_output}${audit_error}")
endif()

file(READ "${TRACE_FILE}" trace_content)

set(protected_path_prefixes
    "/dev/fb"
    "/dev/input/"
    "/sys/"
    "/boot/"
    "/run/kvm/"
    "/etc/kvm/"
    "/opt/"
    "/kvmcomm/"
    "/tmp/kvm/"
)
foreach(protected_path_prefix IN LISTS protected_path_prefixes)
    string(FIND "${trace_content}" "${protected_path_prefix}" path_index)
    if(NOT path_index EQUAL -1)
        message(FATAL_ERROR
            "Simulator syscall trace accessed protected target path prefix: ${protected_path_prefix}")
    endif()
endforeach()

foreach(forbidden_syscall fork vfork bind listen)
    string(FIND "${trace_content}" "${forbidden_syscall}(" syscall_index)
    if(NOT syscall_index EQUAL -1)
        message(FATAL_ERROR
            "Simulator syscall trace contains forbidden syscall: ${forbidden_syscall}")
    endif()
endforeach()

# SDL's optional text-input integration may contact a host-local D-Bus socket.
# It is not part of the simulator runtime, and protected target Unix paths are
# checked above. Network connects remain forbidden.
string(REPLACE "\n" ";" trace_lines "${trace_content}")
foreach(trace_line IN LISTS trace_lines)
    string(FIND "${trace_line}" "connect(" connect_index)
    if(NOT connect_index EQUAL -1)
        string(FIND "${trace_line}" "AF_UNIX" unix_socket_index)
        if(unix_socket_index EQUAL -1)
            message(FATAL_ERROR
                "Simulator syscall trace contains a network connect: ${trace_line}")
        endif()
    endif()
endforeach()

string(REGEX MATCHALL "execve\\(" execve_calls "${trace_content}")
list(LENGTH execve_calls execve_count)
if(execve_count GREATER 1)
    message(FATAL_ERROR
        "Simulator syscall trace contains secondary execve calls: ${execve_count}")
endif()
