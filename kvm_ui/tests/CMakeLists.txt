find_package(Catch2 REQUIRED)
find_package(CURL CONFIG REQUIRED)
find_package(httplib REQUIRED)
find_package(nlohmann_json REQUIRED)
find_package(spdlog REQUIRED)
find_package(stb REQUIRED)
find_package(tomlplusplus REQUIRED)

add_executable(
    nanokvm_go_pure_tests
    accent_palette_test.cpp
    auto_rotation_runtime_test.cpp
    app_catalog_test.cpp
    simulator_app_launcher_test.cpp
    simulator_display_config_test.cpp
    simulator_policy_test.cpp
    simulator_runtime_test.cpp
    framebuffer_snapshot_test.cpp
    launcher_child_control_test.cpp
    launcher_exit_confirmation_test.cpp
    launcher_exit_gesture_test.cpp
    http_api_contract_test.cpp
    lvgl_ui_startup_config_test.cpp
    app_config_test.cpp
    screenshot_control_test.cpp
    screenshot_cli_test.cpp
    backlight_control_test.cpp
    nanokvm_ip_selection_test.cpp
    touch_wake_filter_test.cpp
    wifi_latency_probe_stats_test.cpp
    wifi_script_parser_test.cpp
    wifi_ssid_display_policy_test.cpp
    wifi_page_action_policy_test.cpp
    runtime_font_catalog_test.cpp
    servo_pwm_test.cpp
    settings_cli_test.cpp
    local_service_state_client_test.cpp
    nanokvm_server_transport_test.cpp
    uds_http_client_test.cpp
    nanokvm_runtime_http_api_state_test.cpp
    nanokvm_runtime_ncm_state_test.cpp
    nanokvm_runtime_ncm_lifecycle_test.cpp
    nanokvm_runtime_wifi_ap_lifecycle_test.cpp
    nanokvm_runtime_tailscale_test.cpp
    wifi_runtime_controller_test.cpp
    runtime_status_mailbox_test.cpp
    ssid_raster_result_mailbox_test.cpp
    ssid_raster_worker_test.cpp
    ssid_marquee_window_test.cpp
    runtime_state_store_test.cpp
    systemd_dbus_client_test.cpp
    settings_tile_layout_test.cpp
    region_page_policy_test.cpp
    status_bar_transition_test.cpp
    settings_state_action_policy_test.cpp
    settings_state_route_policy_test.cpp
    hardware_version_test.cpp
    device_number_parser_test.cpp
    package_version_test.cpp
    device_region_test.cpp
    tailscale_client_test.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/background_assets.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/local_service_state_client.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/servo_pwm.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/settings_tile_layout.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/status_bar_transition.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/ssid_marquee_window.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/wifi_page_action_policy.cpp
    ${CMAKE_SOURCE_DIR}/components/ui/src/wifi_ssid_display_policy.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/app_catalog.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/simulator_app_launcher.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/http_api_contract.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/lvgl_ui_effective_config.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/simulator_policy.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/simulator_runtime.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/auto_rotation_runtime.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/app_config.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/screenshot_control.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/screenshot_cli.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/backlight_control.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/framebuffer_snapshot.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/launcher_child_control.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/launcher_exit_confirmation.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/launcher_exit_gesture.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_runtime.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_json_http_utils.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_command_runner.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/systemd_dbus_client.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_status_collectors.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_wifi_ap_support.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_runtime_worker.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_server_transport.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_tailscale_client.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/uds_http_client.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_runtime_config_persist.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/ncm_runtime_state.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/package_version.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/device_region.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/runtime_state_store.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/runtime_thread_pool.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/runtime_font_catalog.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/ssid_raster_worker.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/wifi_runtime_controller.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/wifi_runtime_controller_detail.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/settings_cli.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/nanokvm_runtime_settings_cli.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/touch_wake_filter.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/wifi_script_parser.cpp
)

target_include_directories(
    nanokvm_go_pure_tests PRIVATE
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/include
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src
    ${CMAKE_SOURCE_DIR}/components/ui/include
    ${NANOKVM_GO_GENERATED_INCLUDE_DIR}
)

target_link_libraries(
    nanokvm_go_pure_tests PRIVATE
    Catch2::Catch2WithMain
    CURL::libcurl
    httplib::httplib
    nlohmann_json::nlohmann_json
    lvgl
    spdlog::spdlog
    stb::stb
    tomlplusplus::tomlplusplus
    systemd
)

if(NANOKVM_GO_ENABLE_BACKGROUND_IMAGE)
    target_sources(
        nanokvm_go_pure_tests PRIVATE
        background_assets_test.cpp
    )
endif()

add_executable(
    nanokvm_go_ui_tests
    about_page_hold_confirm_test.cpp
    about_app_ver_secret_click_test.cpp
    apps_page_refresh_test.cpp
    bcon_pending_indicator_test.cpp
    hold_confirm_behavior_test.cpp
    loading_arc_test.cpp
    main_dashboard_view_test.cpp
    more_page_test.cpp
    network_page_action_result_test.cpp
    network_page_code_test.cpp
    network_page_disabled_test.cpp
    network_page_paired_test.cpp
    network_page_preinit_test.cpp
    network_page_terms_test.cpp
    page_manager_characterization_test.cpp
    region_page_test.cpp
    root_pager_test.cpp
    settings_dashboard_view_test.cpp
    settings_navigator_test.cpp
    settings_state_toggle_page_test.cpp
    settings_toggle_coordinator_test.cpp
    settings_toggle_mediator_test.cpp
    subpage_back_navigation_test.cpp
    tailscale_gate_view_test.cpp
    tailscale_page_test.cpp
    wifi_connect_method_page_test.cpp
    wifi_flow_coordinator_test.cpp
    wifi_page_navigation_test.cpp
    wifi_page_hold_confirm_test.cpp
    wifi_qr_page_test.cpp
    wifi_ssid_page_test.cpp
    ssid_text_image_cache_test.cpp
    lvgl_touch_diagnostics_test.cpp
)

if(NOT NANOKVM_GO_RELEASE_BUILD)
    target_sources(
        nanokvm_go_ui_tests PRIVATE
        servo_page_lifecycle_test.cpp
    )
endif()

target_sources(
    nanokvm_go_ui_tests PRIVATE
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/lvgl_touch_diagnostics.cpp
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src/tailscale_projection.cpp
)

target_include_directories(
    nanokvm_go_ui_tests PRIVATE
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/src
    ${CMAKE_SOURCE_DIR}/components/lvgl_main/include
    ${CMAKE_SOURCE_DIR}/components/ui/src
)

target_link_libraries(
    nanokvm_go_ui_tests PRIVATE
    Catch2::Catch2WithMain
    ui
)

set(NANOKVM_GO_TESTS_CAN_RUN TRUE)

if(CMAKE_CROSSCOMPILING)
    set(NANOKVM_GO_TESTS_CAN_RUN FALSE)
    find_program(QEMU_ARM_EMULATOR qemu-arm-static)
    if(QEMU_ARM_EMULATOR)
        execute_process(
            COMMAND ${CMAKE_CXX_COMPILER} -print-sysroot
            OUTPUT_VARIABLE ARM_SYSROOT
            OUTPUT_STRIP_TRAILING_WHITESPACE
            ERROR_QUIET
        )
        if(ARM_SYSROOT AND EXISTS "${ARM_SYSROOT}")
            set_property(
                TARGET nanokvm_go_pure_tests nanokvm_go_ui_tests
                PROPERTY CROSSCOMPILING_EMULATOR
                "${QEMU_ARM_EMULATOR};-L;${ARM_SYSROOT}"
            )
            set(NANOKVM_GO_TESTS_CAN_RUN TRUE)
        endif()
    endif()
endif()

if(NANOKVM_GO_TESTS_CAN_RUN)
    add_test(NAME nanokvm_go_pure_tests COMMAND nanokvm_go_pure_tests)
    add_test(NAME nanokvm_go_ui_tests COMMAND nanokvm_go_ui_tests)
else()
    message(STATUS "Skipping nanokvm_go_pure_tests and nanokvm_go_ui_tests registration: cross emulator unavailable")
endif()

add_test(
    NAME nanokvm_go_simulator_cross_config_rejected
    COMMAND "${CMAKE_COMMAND}"
        "-DSOURCE_DIR=${CMAKE_SOURCE_DIR}"
        "-DTEST_BINARY_DIR=${CMAKE_BINARY_DIR}/simulator-cross-config-rejected"
        -P "${CMAKE_SOURCE_DIR}/cmake/verify_simulator_cross_config_rejected.cmake"
)
set_tests_properties(
    nanokvm_go_simulator_cross_config_rejected PROPERTIES
    TIMEOUT 30
)

if(NANOKVM_GO_SIMULATOR_BUILD)
    set(
        NANOKVM_GO_SIMULATOR_TEST_LIBRARY_DIRS
        ${CMAKE_CXX_IMPLICIT_LINK_DIRECTORIES}
        "/lib/${CMAKE_LIBRARY_ARCHITECTURE}"
        "/usr/lib/${CMAKE_LIBRARY_ARCHITECTURE}"
        "/lib"
        "/usr/lib"
    )
    list(REMOVE_DUPLICATES NANOKVM_GO_SIMULATOR_TEST_LIBRARY_DIRS)
    list(
        JOIN NANOKVM_GO_SIMULATOR_TEST_LIBRARY_DIRS ":"
        NANOKVM_GO_SIMULATOR_TEST_LIBRARY_PATH
    )

    add_executable(
        nanokvm_go_simulator_smoke
        simulator_smoke_test.cpp
    )

    target_link_libraries(
        nanokvm_go_simulator_smoke PRIVATE
        lvgl_main
        ui
        lvgl
        lvgl::thorvg
        spdlog::spdlog
    )

    add_test(
        NAME nanokvm_go_simulator_smoke
        COMMAND nanokvm_go_simulator_smoke
    )
    set_tests_properties(
        nanokvm_go_simulator_smoke PROPERTIES
        ENVIRONMENT
            "LD_LIBRARY_PATH=${NANOKVM_GO_SIMULATOR_TEST_LIBRARY_PATH};SDL_VIDEODRIVER=dummy;SDL_RENDER_DRIVER=software;SDL_AUDIODRIVER=dummy;XMODIFIERS=@im=none"
        TIMEOUT 10
    )

    add_test(
        NAME nanokvm_go_simulator_binary_audit
        COMMAND "${CMAKE_COMMAND}"
            "-DSIMULATOR_BINARY=$<TARGET_FILE:${PROJECT_NAME}>"
            "-DNM_EXECUTABLE=${CMAKE_NM}"
            -P "${CMAKE_SOURCE_DIR}/cmake/verify_simulator_binary.cmake"
    )
    set_tests_properties(
        nanokvm_go_simulator_binary_audit PROPERTIES
        TIMEOUT 10
    )

    find_program(NANOKVM_GO_STRACE_EXECUTABLE NAMES strace)
    if(NANOKVM_GO_STRACE_EXECUTABLE)
        add_test(
            NAME nanokvm_go_simulator_strace_audit
            COMMAND "${CMAKE_COMMAND}"
                "-DSTRACE_EXECUTABLE=${NANOKVM_GO_STRACE_EXECUTABLE}"
                "-DSIMULATOR_SMOKE_BINARY=$<TARGET_FILE:nanokvm_go_simulator_smoke>"
                "-DSIMULATOR_LIBRARY_PATH=${NANOKVM_GO_SIMULATOR_TEST_LIBRARY_PATH}"
                "-DTRACE_FILE=${CMAKE_BINARY_DIR}/simulator-safety.strace"
                -P "${CMAKE_SOURCE_DIR}/cmake/run_simulator_strace_audit.cmake"
        )
        set_tests_properties(
            nanokvm_go_simulator_strace_audit PROPERTIES
            TIMEOUT 15
        )
    else()
        message(STATUS "Skipping simulator strace audit: strace not found")
    endif()
endif()
