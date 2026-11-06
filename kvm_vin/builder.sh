#!/bin/bash

PROJECT_PATH=$(pwd)

__check_conan2_env() {
    CONAN_DEFAULT_PROFILE_PATH="$HOME/.conan2/profiles/default"
    CONAN_PATH=$(which conan 2>/dev/null)

    if [ -x "$CONAN_PATH" ]; then
        echo "[INFO] Found conan at: $CONAN_PATH"
    else
        echo "[WARN] conan not found, attempting to install..."
        if ! python3 -m pip install --user conan >/dev/null 2>&1; then
            echo "[ERROR] Failed to install conan"
            return 1
        fi
        CONAN_PATH=$(which conan 2>/dev/null)
        if [ -z "$CONAN_PATH" ]; then
            echo "[ERROR] conan installation failed"
            return 1
        fi
    fi

    CONAN_VERSION=$($CONAN_PATH -v 2>/dev/null)
    if [ -z "$CONAN_VERSION" ]; then
        echo "[ERROR] Unable to get conan version"
        return 1
    fi
    echo "[INFO] Using conan version: $CONAN_VERSION"

    if [ -f "$CONAN_DEFAULT_PROFILE_PATH" ]; then
        echo "[INFO] Found default conan profile"
    else
        echo "[WARN] Default conan profile not found, creating..."
        if ! $CONAN_PATH profile detect >/dev/null 2>&1; then
            echo "[ERROR] Failed to create default conan profile"
            return 1
        fi
        if [ -f "$CONAN_DEFAULT_PROFILE_PATH" ]; then
            echo "[INFO] Default conan profile created successfully"
        else
            echo "[ERROR] Default conan profile creation failed"
            return 1
        fi
    fi
    return 0
}

build_3rd_lib() {
    cd $PROJECT_PATH
    conan install . --build missing -pr:b=default -pr:h=profile/nanokvm-go
}

clean_target() {
    cd $PROJECT_PATH
    rm -rf build
    rm -rf conan
}

__build_target() {
    mkdir -p $PROJECT_PATH/build
    cd $PROJECT_PATH/build
    source ../conan/conanbuild.sh
    cmake .. -DCMAKE_TOOLCHAIN_FILE=../conan/conan_toolchain.cmake \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
        -DBUILD_TESTING=ON \
        -DCMAKE_CXX_FLAGS_RELEASE="-Os -ffunction-sections -fdata-sections" \
        -DCMAKE_EXE_LINKER_FLAGS_RELEASE="-s -Wl,--gc-sections"
    cmake --build . --parallel $(nproc)
    cd $PROJECT_PATH
    source conan/deactivate_conanbuild.sh
}

build_target() {
    if ! __check_conan2_env || ! build_3rd_lib || ! __build_target; then
        echo "[ERROR] Build failed"
        return 1
    fi
    return 0
}

rebuild_target() {
    if ! __check_conan2_env || ! clean_target || ! build_3rd_lib || ! __build_target; then
        echo "[ERROR] Rebuild failed"
        return 1
    fi
    return 0
}

builder_main() {
    if [ -z "$1" ]; then
        echo "Usage: $0 <command>"
        echo "Commands:"
        echo "  build      - Build the project"
        echo "  rebuild    - Clean and rebuild the project"
        echo "  clean      - Clean build artifacts"
        echo "  build_3rd  - Build 3rd party libraries"
        echo "  help       - Show this help message"
        return 1
    fi

    case "$1" in
    help)
        echo "C/C++ Project Builder"
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  build      - Build the project"
        echo "  rebuild    - Clean and rebuild the project"
        echo "  clean      - Clean build artifacts"
        echo "  build_3rd  - Build 3rd party libraries"
        echo "  help       - Show this help message"
        ;;
    build)
        build_target
        ;;
    rebuild)
        rebuild_target
        ;;
    minibuild)
        __build_target
        ;;
    clean)
        clean_target
        ;;
    build_3rd)
        build_3rd_lib
        ;;
    *)
        echo "Error: Unknown command '$1'"
        echo "Run '$0 help' for usage information"
        return 1
        ;;
    esac
    return 0
}

builder_main "$@"
