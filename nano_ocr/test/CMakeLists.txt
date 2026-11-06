add_executable(nano_ocr_test
    latest_frame_state_test.cpp
    log_test.cpp
    snapshot_client_test.cpp
    snapshot_client_io_test.cpp
    ocr_image_test.cpp
    result_writer_test.cpp
)

target_link_libraries(nano_ocr_test PRIVATE
    Catch2::Catch2WithMain
    ocr_snapshot_client
    ocr_image
    ocr_log
    ocr_result_writer
    nlohmann_json::nlohmann_json
    ${OpenCV_LIBS}
    pthread
)

if(NOT CMAKE_CROSSCOMPILING)
    include(Catch)
    catch_discover_tests(nano_ocr_test)
endif()
