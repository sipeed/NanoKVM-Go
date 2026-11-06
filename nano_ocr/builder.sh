#!/bin/bash

PROJECT_PATH=$(pwd)
HOST_PROFILE="../kvm_vin/profile/nanokvm-go"

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
    cd "$PROJECT_PATH"
    conan install . --build missing -pr:b=default -pr:h="$HOST_PROFILE" \
        -s:h compiler.libcxx=libstdc++11 \
        -o:h 'libcurl/*:shared=True'
}

clean_target() {
    cd "$PROJECT_PATH"
    rm -rf build
    rm -rf conan
}

build_opencv() {
    cmake -S "$PROJECT_PATH/cmake/opencv" \
        -B "$PROJECT_PATH/.cache/opencv-build" \
        -DCMAKE_TOOLCHAIN_FILE="$PROJECT_PATH/conan/conan_toolchain.cmake" \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX="$PROJECT_PATH/prefix/opencv" \
        -DFETCHCONTENT_BASE_DIR="$PROJECT_PATH/.cache/opencv" \
        -DCMAKE_CXX_FLAGS_RELEASE="-Os -ffunction-sections -fdata-sections" \
        -DCMAKE_C_FLAGS_RELEASE="-Os -ffunction-sections -fdata-sections" || return
    cmake --build "$PROJECT_PATH/.cache/opencv-build" \
        --parallel "$(nproc)" || return
    cmake --install "$PROJECT_PATH/.cache/opencv-build"
}

__build_target() {
    cd "$PROJECT_PATH" || return
    source conan/conanbuild.sh || return
    build_opencv || return
    mkdir -p "$PROJECT_PATH/build" || return
    cd "$PROJECT_PATH/build" || return
    cmake .. -DCMAKE_TOOLCHAIN_FILE=../conan/conan_toolchain.cmake \
        -DOCR_OPENCV_DIR="$PROJECT_PATH/prefix/opencv/lib/cmake/opencv4" \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
        -DCMAKE_CXX_FLAGS_RELEASE="-Os -ffunction-sections -fdata-sections" \
        -DCMAKE_EXE_LINKER_FLAGS_RELEASE="-s -Wl,--gc-sections" || return
    cmake --build . --target nano_ocr_app --parallel "$(nproc)" || return
    cd "$PROJECT_PATH"
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
        echo "  minibuild  - Run CMake configure/build using existing Conan files"
        echo "  help       - Show this help message"
        return 1
    fi

    case "$1" in
    help)
        echo "Nano OCR Project Builder"
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  build      - Build the project"
        echo "  rebuild    - Clean and rebuild the project"
        echo "  clean      - Clean build artifacts"
        echo "  build_3rd  - Build 3rd party libraries"
        echo "  minibuild  - Run CMake configure/build using existing Conan files"
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
}

builder_main "$@"
