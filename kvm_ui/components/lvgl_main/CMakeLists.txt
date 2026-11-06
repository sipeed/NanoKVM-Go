set(LVGL_MAIN_SOURCES
    ${CMAKE_CURRENT_SOURCE_DIR}/src/app_catalog.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_ui.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_color_lut_flush.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/wifi_latency_probe.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_input_reader.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_display_rotation.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_runtime_binding.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/tailscale_projection.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_fb_exit_confirmation.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_app_session.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_touch_diagnostics.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_ui_effective_config.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/simulator_policy.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/auto_rotation_runtime.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/app_config.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/device_region.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/screenshot_control.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/screenshot_cli.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/backlight_control.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/http_api_server.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/http_api_contract.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/launcher_exit_confirmation.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/launcher_exit_gesture.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_json_http_utils.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_command_runner.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_status_collectors.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_wifi_ap_support.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime_worker.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_server_transport.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_tailscale_client.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/uds_http_client.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime_config_persist.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ncm_runtime_state.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/package_version.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/runtime_state_store.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/runtime_thread_pool.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/runtime_font_catalog.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/ssid_raster_worker.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/touch_wake_filter.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/src/wifi_script_parser.cpp
)

if(NANOKVM_GO_SIMULATOR_BUILD)
    list(REMOVE_ITEM LVGL_MAIN_SOURCES
        "${CMAKE_CURRENT_SOURCE_DIR}/src/http_api_server.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_json_http_utils.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_command_runner.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_status_collectors.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_wifi_ap_support.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime_worker.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_server_transport.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_tailscale_client.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/uds_http_client.cpp"
        "${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime_config_persist.cpp"
    )
    list(APPEND LVGL_MAIN_SOURCES
        ${CMAKE_CURRENT_SOURCE_DIR}/src/simulator_app_launcher.cpp
        ${CMAKE_CURRENT_SOURCE_DIR}/src/simulator_runtime.cpp
    )
else()
    list(APPEND LVGL_MAIN_SOURCES
        ${CMAKE_CURRENT_SOURCE_DIR}/src/systemd_dbus_client.cpp
        ${CMAKE_CURRENT_SOURCE_DIR}/src/wifi_runtime_controller.cpp
        ${CMAKE_CURRENT_SOURCE_DIR}/src/wifi_runtime_controller_detail.cpp
    )
    if(NOT NANOKVM_GO_RELEASE_BUILD)
        list(APPEND LVGL_MAIN_SOURCES
            ${CMAKE_CURRENT_SOURCE_DIR}/src/nanokvm_runtime_settings_cli.cpp
        )
    endif()
endif()

if(NOT NANOKVM_GO_RELEASE_BUILD)
    list(APPEND LVGL_MAIN_SOURCES ${CMAKE_CURRENT_SOURCE_DIR}/src/settings_cli.cpp)
endif()

if(NOT NANOKVM_GO_SIMULATOR_BUILD)
    list(APPEND LVGL_MAIN_SOURCES
        ${CMAKE_CURRENT_SOURCE_DIR}/src/framebuffer_snapshot.cpp
        ${CMAKE_CURRENT_SOURCE_DIR}/src/launcher_child_control.cpp
    )
endif()

add_library(
    lvgl_main OBJECT
    ${LVGL_MAIN_SOURCES}
)

target_include_directories(
    lvgl_main PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<BUILD_INTERFACE:${NANOKVM_GO_GENERATED_INCLUDE_DIR}>
)

target_include_directories(
    lvgl_main PRIVATE
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/src>
)

find_package(httplib REQUIRED)
find_package(CURL CONFIG REQUIRED)
find_package(nlohmann_json REQUIRED)
find_package(tomlplusplus REQUIRED)
find_package(spdlog REQUIRED)

target_link_libraries(
    lvgl_main PUBLIC
    ui
    PRIVATE
    spdlog::spdlog
    lvgl_lib
    httplib::httplib
    CURL::libcurl
    nlohmann_json::nlohmann_json
    tomlplusplus::tomlplusplus
)

if(NOT NANOKVM_GO_SIMULATOR_BUILD)
    target_link_libraries(lvgl_main PRIVATE systemd)
endif()
target_compile_features(lvgl_main PUBLIC cxx_std_20)
