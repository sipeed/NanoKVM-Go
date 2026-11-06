file(GLOB UI_SOURCES CONFIGURE_DEPENDS src/*.cpp)
if(NANOKVM_GO_RELEASE_BUILD)
    list(REMOVE_ITEM UI_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/src/servo_page.cpp")
    list(REMOVE_ITEM UI_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/src/servo_pwm.cpp")
endif()
file(GLOB UI_SRCS CONFIGURE_DEPENDS srcs/*.c)
find_package(nlohmann_json REQUIRED)
find_package(stb REQUIRED)
find_package(tomlplusplus REQUIRED)
find_package(spdlog REQUIRED)
set(UI_FONT_SOURCES
    fonts/MontserratBold_18.c
    fonts/MontserratBold_20.c
    fonts/MontserratBold_24.c
    fonts/MontserratBold_26.c
    fonts/MontserratBold_28.c
    fonts/MontserratBold_30.c
    fonts/MontserratBold_36.c
    fonts/wifi_zh_font.c
)

add_library(ui STATIC
    ${UI_SOURCES}
    ${UI_SRCS}
    ${UI_FONT_SOURCES}
)

target_include_directories(ui
    PUBLIC
        include
)

target_link_libraries(ui
    PUBLIC
        lvgl
    PRIVATE
        spdlog::spdlog
        nlohmann_json::nlohmann_json
        stb::stb
        tomlplusplus::tomlplusplus
)

target_compile_features(ui PUBLIC cxx_std_20)
