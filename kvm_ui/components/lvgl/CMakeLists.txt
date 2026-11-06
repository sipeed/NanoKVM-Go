if(NANOKVM_GO_SIMULATOR_BUILD)
    set(LV_CONF_PATH ${CMAKE_CURRENT_SOURCE_DIR}/config/lv_conf_simulator.h)
else()
    set(LV_CONF_PATH ${CMAKE_CURRENT_SOURCE_DIR}/config/lv_conf.h)
endif()

set(LV_CONF_BUILD_DISABLE_EXAMPLES ON CACHE BOOL "Disable LVGL examples" FORCE)
set(LV_CONF_BUILD_DISABLE_DEMOS    ON CACHE BOOL "Disable LVGL demos"    FORCE)

add_subdirectory(
    "${CMAKE_SOURCE_DIR}/external/lvgl"
    "${CMAKE_CURRENT_BINARY_DIR}/lvgl"
)

# The pinned ThorVG public header uses the fixed-width integer types without
# including their standard C++ header. Newer GCC and Clang versions no longer
# expose those types through incidental transitive includes.
target_compile_options(
    lvgl_thorvg PRIVATE
    $<$<COMPILE_LANGUAGE:CXX>:-include>
    $<$<COMPILE_LANGUAGE:CXX>:cstdint>
)

target_link_libraries(
    lvgl PUBLIC
    lvgl::thorvg
)

if(NANOKVM_GO_SIMULATOR_BUILD)
    find_package(PkgConfig REQUIRED)
    pkg_check_modules(SDL2 REQUIRED IMPORTED_TARGET sdl2)
    target_link_libraries(lvgl PUBLIC PkgConfig::SDL2)
endif()

add_library(
    lvgl_lib STATIC
    ${CMAKE_CURRENT_SOURCE_DIR}/src/lvgl_adapter.cpp
)

target_link_libraries(
    lvgl_lib PUBLIC
    lvgl
    lvgl::thorvg
)

target_compile_features(lvgl_lib PUBLIC cxx_std_20)
