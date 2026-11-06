cmake_minimum_required(VERSION 3.20)

project(Memfab LANGUAGES NONE)

find_program(GO_EXECUTABLE go REQUIRED)
find_program(CMAKE_TAR_EXECUTABLE tar)

set(MEMFAB_VERSION "0.1.0-memfab" CACHE STRING "memfab Go version")
set(MEMFAB_OUTPUT_DIR "${CMAKE_BINARY_DIR}/dist" CACHE PATH "Output directory for built binaries")
set(MEMFAB_GO_PACKAGE "./cmd/memfab" CACHE STRING "Go package to build")
set(MEMFAB_GO_TAGS "sqlite_fts5" CACHE STRING "Go build tags")
set(MEMFAB_ARM_GO_TAGS "sqlite_fts5" CACHE STRING "Go build tags for ARM targets")
set(MEMFAB_ARM_TOOLCHAIN_URL "https://armkeil.blob.core.windows.net/developer/Files/downloads/gnu-a/10.3-2021.07/binrel/gcc-arm-10.3-2021.07-x86_64-arm-none-linux-gnueabihf.tar.xz" CACHE STRING "ARM GNU toolchain archive URL")
set(MEMFAB_ARM_TOOLCHAIN_ARCHIVE "${CMAKE_BINARY_DIR}/toolchains/gcc-arm-10.3-2021.07-x86_64-arm-none-linux-gnueabihf.tar.xz" CACHE FILEPATH "Downloaded ARM GNU toolchain archive")
set(MEMFAB_ARM_TOOLCHAIN_DIR "${CMAKE_SOURCE_DIR}/../support/toolchains/armv7hf-toolchains" CACHE PATH "ARM hard-float Linux toolchain directory")
set(MEMFAB_ARM_CC "${MEMFAB_ARM_TOOLCHAIN_DIR}/bin/arm-none-linux-gnueabihf-gcc" CACHE FILEPATH "ARM hard-float Linux C compiler")

option(MEMFAB_DOWNLOAD_ARM_TOOLCHAIN "Download the ARM GNU toolchain archive with CMake" OFF)
option(MEMFAB_USE_ARM_TOOLCHAIN "Use MEMFAB_ARM_CC for the armv7 cgo build" ON)

set(GO_LDFLAGS "-s -w -X main.version=${MEMFAB_VERSION}")

file(MAKE_DIRECTORY "${MEMFAB_OUTPUT_DIR}")

add_custom_target(go-tidy
    COMMAND "${GO_EXECUTABLE}" mod tidy
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    COMMENT "Tidying Go modules"
    VERBATIM
)

add_custom_target(go-test
    COMMAND "${GO_EXECUTABLE}" test -tags "${MEMFAB_GO_TAGS}" ./...
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    COMMENT "Running Go tests"
    VERBATIM
)

add_custom_target(memfab-native ALL
    COMMAND "${CMAKE_COMMAND}" -E env CGO_ENABLED=1
            "${GO_EXECUTABLE}" build -tags "${MEMFAB_GO_TAGS}" -trimpath -ldflags=${GO_LDFLAGS}
            -o "${MEMFAB_OUTPUT_DIR}/memfab"
            "${MEMFAB_GO_PACKAGE}"
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    COMMENT "Building native memfab Go binary"
    VERBATIM
)

add_custom_target(download-arm-toolchain
    COMMAND "${CMAKE_COMMAND}" -E make_directory "${CMAKE_BINARY_DIR}/toolchains"
    COMMAND "${CMAKE_COMMAND}"
            -DURL="${MEMFAB_ARM_TOOLCHAIN_URL}"
            -DOUT="${MEMFAB_ARM_TOOLCHAIN_ARCHIVE}"
            -DEXTRACT_DIR="${CMAKE_BINARY_DIR}/toolchains"
            -P "${CMAKE_SOURCE_DIR}/cmake/DownloadArmToolchain.cmake"
    COMMENT "Downloading and extracting ARM GNU toolchain"
    VERBATIM
)

if(MEMFAB_USE_ARM_TOOLCHAIN)
    set(ARM_BUILD_ENV
        CGO_ENABLED=1
        GOOS=linux
        GOARCH=arm
        GOARM=7
        CC=${MEMFAB_ARM_CC}
    )
else()
    set(ARM_BUILD_ENV
        CGO_ENABLED=1
        GOOS=linux
        GOARCH=arm
        GOARM=7
    )
endif()

add_custom_target(memfab-linux-armv7
    COMMAND "${CMAKE_COMMAND}" -E env ${ARM_BUILD_ENV}
            "${GO_EXECUTABLE}" build -tags "${MEMFAB_ARM_GO_TAGS}" -trimpath -ldflags=${GO_LDFLAGS}
            -o "${MEMFAB_OUTPUT_DIR}/memfab-linux-armv7"
            "${MEMFAB_GO_PACKAGE}"
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    COMMENT "Cross-building memfab Go for linux/armv7 hard-float"
    VERBATIM
)

if(MEMFAB_DOWNLOAD_ARM_TOOLCHAIN)
    add_custom_target(memfab-linux-armv7-with-toolchain
        COMMAND "${CMAKE_COMMAND}" --build "${CMAKE_BINARY_DIR}" --target download-arm-toolchain
        COMMAND "${CMAKE_COMMAND}" --build "${CMAKE_BINARY_DIR}" --target memfab-linux-armv7
        COMMENT "Downloading ARM toolchain then cross-building memfab Go"
        VERBATIM
    )
else()
    add_custom_target(memfab-linux-armv7-with-toolchain
        COMMAND "${CMAKE_COMMAND}" --build "${CMAKE_BINARY_DIR}" --target memfab-linux-armv7
        COMMENT "Cross-building memfab Go with configured ARM toolchain"
        VERBATIM
    )
endif()

add_custom_target(memfab-linux-armv7-nocgo
    COMMAND "${CMAKE_COMMAND}" -E env CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7
            "${GO_EXECUTABLE}" build -tags sqlite_fts5 -trimpath -ldflags=${GO_LDFLAGS}
            -o "${MEMFAB_OUTPUT_DIR}/memfab-linux-armv7-nocgo"
            "${MEMFAB_GO_PACKAGE}"
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    COMMENT "Cross-building memfab Go for linux/armv7 without cgo"
    VERBATIM
)
