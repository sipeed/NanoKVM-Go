set(FFMPEG_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/include")

set(FFMPEG_REQUIRED_FILES
    "${FFMPEG_INCLUDE_DIR}/libavformat/avformat.h"
    "${FFMPEG_INCLUDE_DIR}/libavcodec/avcodec.h"
    "${FFMPEG_INCLUDE_DIR}/libavcodec/packet.h"
    "${FFMPEG_INCLUDE_DIR}/libavutil/avutil.h"
)

foreach(REQUIRED_FILE ${FFMPEG_REQUIRED_FILES})
    if(NOT EXISTS "${REQUIRED_FILE}")
        message(FATAL_ERROR "Missing FFmpeg vendor file: ${REQUIRED_FILE}")
    endif()
endforeach()

add_library(ffmpeg INTERFACE)

target_include_directories(ffmpeg INTERFACE
    "${FFMPEG_INCLUDE_DIR}"
)


target_link_libraries(ffmpeg INTERFACE ${CMAKE_DL_LIBS})
