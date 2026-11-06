#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PROJECT_PATH="$SCRIPT_DIR"
export PLUGIN_DIR="$SCRIPT_DIR/bplugins"

readonly PROJECT_PATH PLUGIN_DIR
readonly LVGL_REMOTE="https://github.com/lvgl/lvgl.git"
readonly LVGL_BRANCH="release/v9.2"
readonly LVGL_DIR="$PROJECT_PATH/external/lvgl"
readonly LVGL_PINNED_COMMIT="933b2352bfec08b231e289546286aa0ef63d5ef8"
readonly MODE_FILE="$PROJECT_PATH/build/mode.conf"
readonly SIMULATOR_BUILD_DIR="$PROJECT_PATH/build-simulator"
readonly SIMULATOR_CONAN_DIR="$SIMULATOR_BUILD_DIR/conan"
readonly HOST_PROFILE_PATH="$PROJECT_PATH/profile/nanokvm-go"
readonly MONOREPO_HOST_PROFILE_PATH="$PROJECT_PATH/../support/scripts/nanokvm-go"
readonly SYSTEMD_DEB_CACHE_DIR="$PROJECT_PATH/.cache/deb"
readonly SYSTEMD_SYSROOT_DIR="$PROJECT_PATH/build/sysroot"
readonly SYSTEMD_SYSROOT_INCLUDE_DIR="$SYSTEMD_SYSROOT_DIR/usr/include"
readonly SYSTEMD_SYSROOT_LIB_DIR="$SYSTEMD_SYSROOT_DIR/usr/lib"
readonly SYSTEMD_DEB_LIB_DIR="arm-linux-gnueabihf"
# Debian bookworm matches the toolchain glibc 2.36 baseline; the target's newer
# libsystemd.so.0 stays ABI-compatible with these public sd-bus APIs. The values
# mirror the monorepo toolchain setup manifest.
readonly SYSTEMD_DEB_MANIFEST="\
libsystemd-dev|https://deb.debian.org/debian/pool/main/s/systemd/libsystemd-dev_252.39-1~deb12u2_armhf.deb|1b06a10f98154f855d4f4e140a7cc59d0821baf6debd6a943636fbac315f69d5
libsystemd0|https://deb.debian.org/debian/pool/main/s/systemd/libsystemd0_252.39-1~deb12u2_armhf.deb|fe49d9caa76c1ebc90210936c3aa85aeb6e9bb283d4128b825519887117fdb4d
libcap2|https://deb.debian.org/debian/pool/main/libc/libcap2/libcap2_2.66-4+deb12u3+b1_armhf.deb|26c599095f5bdac228385ea99ff82344ab1395c828cb24e7e55da823f3ca2e8c
libgcrypt20|https://deb.debian.org/debian/pool/main/libg/libgcrypt20/libgcrypt20_1.10.1-3+deb12u1_armhf.deb|e217694621014cbb2849c2d9fd49e3a9ce9219c67c244286f35d277300a07b6f
libgpg-error0|https://deb.debian.org/debian/pool/main/libg/libgpg-error/libgpg-error0_1.46-1_armhf.deb|caa6ee2f9d1e5d7cd21a2bf712be8c836766cfa4fc76853feca7c4e69ec9b7ac
liblz4-1|https://deb.debian.org/debian/pool/main/l/lz4/liblz4-1_1.9.4-1_armhf.deb|a646819a654404c38c219f5df4504ba1c214326daca6a00ce4ad9bb073447fbd
liblzma5|https://deb.debian.org/debian/pool/main/x/xz-utils/liblzma5_5.4.1-1+deb12u1_armhf.deb|bfbb78b257213e3d66a8c5940c1f5e922dc3bad5c427c9077324657a5eb78e00
libzstd1|https://deb.debian.org/debian/pool/main/libz/libzstd/libzstd1_1.5.4+dfsg2-5_armhf.deb|a2ca48bc43bb51d3cbc6beb07aefd3fe252859868a4ccb9c1edbfc942786e4b1"

__load_mode() {
    if [ -f "$MODE_FILE" ]; then
        cat "$MODE_FILE"
    else
        echo "release"
    fi
}

__save_mode() {
    local MODE="$1"
    mkdir -p "$PROJECT_PATH/build"
    echo "$MODE" >"$MODE_FILE"
}

__is_valid_mode() {
    case "$1" in
    debug | release) return 0 ;;
    *) return 1 ;;
    esac
}

__resolve_mode() {
    local INPUT_MODE="$1"
    local RES_MODE=""

    if [ -n "$INPUT_MODE" ]; then
        RES_MODE="$INPUT_MODE"
        if ! __is_valid_mode "$RES_MODE"; then
            echo -e "\033[31m[ERROR] invalid mode '$RES_MODE'\033[0m" >&2
            return 1
        fi
        __save_mode "$RES_MODE"
    else
        RES_MODE="$(__load_mode)"
        if ! __is_valid_mode "$RES_MODE"; then
            echo -e "\033[31m[ERROR] invalid stored mode '$RES_MODE'; set '$MODE_FILE' to 'debug' or 'release'\033[0m" >&2
            return 1
        fi
    fi

    echo "$RES_MODE"
    return 0
}

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

__host_profile_path() {
    if [ -f "$HOST_PROFILE_PATH" ]; then
        echo "$HOST_PROFILE_PATH"
        return 0
    fi

    if [ -f "$MONOREPO_HOST_PROFILE_PATH" ]; then
        echo "$MONOREPO_HOST_PROFILE_PATH"
        return 0
    fi

    echo "[ERROR] Conan host profile not found: $HOST_PROFILE_PATH or $MONOREPO_HOST_PROFILE_PATH" >&2
    return 1
}

__is_repo_clean() {
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        if [ -z "$(git status --porcelain -- "$PROJECT_PATH")" ]; then
            return 0
        else
            echo "[ERROR] Project path is not clean, commit or stash changes first"
            git status --short -- "$PROJECT_PATH"
            return 1
        fi
    else
        echo "[WARN] Not inside a git repository, skip clean check"
        return 0
    fi
}

__ensure_lvgl_repo() {
    if [ -e "$LVGL_DIR" ] && [ ! -d "$LVGL_DIR/.git" ]; then
        echo "[ERROR] $LVGL_DIR exists but is not a git repository"
        return 1
    fi

    if [ -d "$LVGL_DIR/.git" ]; then
        echo "[INFO] lvgl repository already exists at $LVGL_DIR"
        return 0
    fi

    echo "[INFO] lvgl repository not found, cloning..."
    mkdir -p "$PROJECT_PATH/external"
    if ! git clone --depth 1 --branch "$LVGL_BRANCH" "$LVGL_REMOTE" "$LVGL_DIR"; then
        echo "[ERROR] Failed to clone lvgl repository"
        return 1
    fi
    echo "[INFO] lvgl repository cloned successfully"
    return 0
}

__verify_lvgl_commit() {
    if [ ! -d "$LVGL_DIR/.git" ]; then
        echo "[ERROR] $LVGL_DIR is not a git repository"
        return 1
    fi

    pushd "$LVGL_DIR" >/dev/null || return 1

    local current_commit pinned_commit_short
    current_commit=$(git rev-parse --short=8 HEAD)
    pinned_commit_short=$(git rev-parse --short=8 "$LVGL_PINNED_COMMIT")
    if [ "$current_commit" = "$pinned_commit_short" ]; then
        if [ -z "$(git status --porcelain)" ]; then
            popd >/dev/null || true
            echo "[INFO] lvgl is at pinned commit $LVGL_PINNED_COMMIT and clean"
            return 0
        fi
        echo "[ERROR] lvgl is at pinned commit but working tree is not clean"
        git status --short
        popd >/dev/null || true
        return 1
    fi

    echo "[INFO] lvgl at $current_commit, checking out pinned commit $LVGL_PINNED_COMMIT..."
    if [ -n "$(git status --porcelain)" ]; then
        echo "[ERROR] lvgl working tree is not clean, cannot checkout pinned commit"
        git status --short
        popd >/dev/null || true
        return 1
    fi

    if ! git fetch --depth 1 origin "$LVGL_PINNED_COMMIT"; then
        if ! git fetch --depth 1 origin +"$LVGL_PINNED_COMMIT"; then
            echo "[ERROR] Failed to fetch lvgl pinned commit $LVGL_PINNED_COMMIT"
            popd >/dev/null || true
            return 1
        fi
    fi

    if ! git -c advice.detachedHead=false checkout "$LVGL_PINNED_COMMIT"; then
        echo "[ERROR] Failed to checkout lvgl pinned commit $LVGL_PINNED_COMMIT"
        popd >/dev/null || true
        return 1
    fi

    echo "[INFO] lvgl checked out to pinned commit $LVGL_PINNED_COMMIT"
    popd >/dev/null || return 1
    return 0
}

__run_external_plugin() {
    local cmd="$1"
    shift || true

    local plugin="$PLUGIN_DIR/$cmd"

    if [ -x "$plugin" ]; then
        exec "$plugin" "$@"
    fi

    return 127
}

build_3rd_lib() {
    local MODE="${1:-release}"
    local CONAN_BUILD_TYPE="Release"
    local HOST_PROFILE
    if [ "$MODE" = "debug" ]; then
        CONAN_BUILD_TYPE="RelWithDebInfo"
    fi

    HOST_PROFILE=$(__host_profile_path) || return 1
    cd "$PROJECT_PATH" || return 1
    conan install . --build missing -pr:b=default -pr:h="$HOST_PROFILE" -s:h build_type="$CONAN_BUILD_TYPE"
}

rebuild_3rd_lib() {
    local MODE="${1:-release}"
    local CONAN_BUILD_TYPE="Release"
    local HOST_PROFILE
    if [ "$MODE" = "debug" ]; then
        CONAN_BUILD_TYPE="RelWithDebInfo"
    fi

    HOST_PROFILE=$(__host_profile_path) || return 1
    cd "$PROJECT_PATH" || return 1
    conan install . --build=* -pr:b=default -pr:h="$HOST_PROFILE" -s:h build_type="$CONAN_BUILD_TYPE"
}

build_simulator_3rd_lib() {
    cd "$PROJECT_PATH" || return 1
    conan install . --build missing -pr:b=default -pr:h=default \
        -s:h build_type=Release --output-folder="$SIMULATOR_BUILD_DIR"
}

clean_target() {
    cd "$PROJECT_PATH" || return 1
    rm -rf build
    rm -rf conan
    echo "[INFO] Cleaned build/ and conan/, external/ preserved"
}

clean_simulator_target() {
    if [ "$SIMULATOR_BUILD_DIR" != "$PROJECT_PATH/build-simulator" ]; then
        echo "[ERROR] Refusing to clean unexpected simulator directory: $SIMULATOR_BUILD_DIR" >&2
        return 1
    fi

    if [ -e "$SIMULATOR_BUILD_DIR" ]; then
        rm -rf -- "$SIMULATOR_BUILD_DIR"
    fi
    echo "[INFO] Cleaned build-simulator/ only"
}

__require_command() {
    local CMD="$1"
    if ! command -v "$CMD" >/dev/null 2>&1; then
        echo "[ERROR] Required command not found: $CMD" >&2
        return 1
    fi
    return 0
}

__sha256_file() {
    local FILE="$1"
    local DIGEST=""
    if [ ! -f "$FILE" ]; then
        echo "[ERROR] Cannot hash missing file: $FILE" >&2
        return 1
    fi
    DIGEST="$(sha256sum -- "$FILE")" || {
        echo "[ERROR] Cannot hash file: $FILE" >&2
        return 1
    }
    printf '%s\n' "${DIGEST%% *}"
}

__deb_cache_matches() {
    local NAME="$1"
    local EXPECTED_SHA256="$2"
    local DEB_PATH="$SYSTEMD_DEB_CACHE_DIR/${NAME}.deb"
    local ACTUAL_SHA256=""

    [ -f "$DEB_PATH" ] || return 1
    ACTUAL_SHA256="$(__sha256_file "$DEB_PATH")" || return 1
    [ "$ACTUAL_SHA256" = "$EXPECTED_SHA256" ]
}

__deb_cache_fetch() {
    local NAME="$1"
    local URL="$2"
    local EXPECTED_SHA256="$3"
    local DEB_PATH="$SYSTEMD_DEB_CACHE_DIR/${NAME}.deb"
    local RECORD_PATH="$SYSTEMD_DEB_CACHE_DIR/${NAME}.deb.sha256"
    local TMP_PATH="$SYSTEMD_DEB_CACHE_DIR/${NAME}.deb.tmp"
    local ACTUAL_SHA256=""
    local ATTEMPT=""

    if __deb_cache_matches "$NAME" "$EXPECTED_SHA256"; then
        return 0
    fi

    __require_command curl || return 1

    if [ -e "$DEB_PATH" ]; then
        echo "[WARN] Cached deb failed sha256 verification, re-downloading: $DEB_PATH" >&2
    fi

    for ATTEMPT in 1 2; do
        rm -f -- "$TMP_PATH"
        if ! curl -fsSL -o "$TMP_PATH" "$URL"; then
            echo "[ERROR] Failed to download $NAME from $URL" >&2
            rm -f -- "$TMP_PATH"
            return 1
        fi

        ACTUAL_SHA256="$(__sha256_file "$TMP_PATH")" || {
            rm -f -- "$TMP_PATH"
            return 1
        }

        if [ "$ACTUAL_SHA256" = "$EXPECTED_SHA256" ]; then
            if ! mv -f -- "$TMP_PATH" "$DEB_PATH"; then
                echo "[ERROR] Failed to move downloaded deb into the cache: $DEB_PATH" >&2
                rm -f -- "$TMP_PATH"
                return 1
            fi
            printf '%s  %s\n' "$ACTUAL_SHA256" "${NAME}.deb" >"$RECORD_PATH" || {
                echo "[ERROR] Failed to write the cache record: $RECORD_PATH" >&2
                return 1
            }
            echo "[INFO] Fetched $NAME -> $DEB_PATH"
            return 0
        fi

        echo "[WARN] sha256 mismatch for $NAME (download $ATTEMPT/2)" >&2
        echo "[WARN]   expected: $EXPECTED_SHA256" >&2
        echo "[WARN]   actual:   $ACTUAL_SHA256" >&2
        rm -f -- "$TMP_PATH"
    done

    echo "[ERROR] $NAME still does not match the expected sha256 after re-download" >&2
    echo "[ERROR] The upstream package may have changed; update the inline libsystemd deb manifest in builder.sh" >&2
    echo "[ERROR] Refusing to use the unverified package; the previous cache entry was left untouched" >&2
    return 1
}

__extract_deb_data() {
    local DEB_PATH="$1"
    local DEST_DIR="$2"

    mkdir -p "$DEST_DIR" || return 1
    (
        cd "$DEST_DIR" || exit 1
        ar x "$DEB_PATH" || exit 1
        if [ -f data.tar.xz ]; then
            tar xf data.tar.xz || exit 1
        elif [ -f data.tar.zst ]; then
            __require_command zstd || exit 1
            zstd -d data.tar.zst -o data.tar || exit 1
            tar xf data.tar || exit 1
        elif [ -f data.tar.gz ]; then
            tar xzf data.tar.gz || exit 1
        else
            echo "[ERROR] Unsupported deb data archive in $DEB_PATH (expected data.tar.xz, data.tar.zst, or data.tar.gz)" >&2
            exit 1
        fi
    )
}

__ensure_systemd_sysroot() {
    if [ "$SYSTEMD_SYSROOT_DIR" != "$PROJECT_PATH/build/sysroot" ]; then
        echo "[ERROR] Refusing to manage unexpected sysroot directory: $SYSTEMD_SYSROOT_DIR" >&2
        return 1
    fi

    __require_command tar || return 1
    __require_command ar || return 1
    __require_command sha256sum || return 1

    mkdir -p "$SYSTEMD_DEB_CACHE_DIR" "$PROJECT_PATH/build" || return 1

    local LEFTOVER_DIR=""
    for LEFTOVER_DIR in "$PROJECT_PATH"/build/deb-unpack.*; do
        [ -d "$LEFTOVER_DIR" ] || continue
        rm -rf -- "$LEFTOVER_DIR" || return 1
    done

    if [ -e "$SYSTEMD_SYSROOT_DIR" ]; then
        rm -rf -- "$SYSTEMD_SYSROOT_DIR" || return 1
    fi
    mkdir -p "$SYSTEMD_SYSROOT_INCLUDE_DIR" "$SYSTEMD_SYSROOT_LIB_DIR" || return 1

    local STAGING_DIR=""
    STAGING_DIR="$(mktemp -d "${PROJECT_PATH}/build/deb-unpack.XXXXXX")" || {
        echo "[ERROR] Failed to create a deb unpack staging directory under $PROJECT_PATH/build" >&2
        return 1
    }

    local NAME=""
    local URL=""
    local EXPECTED_SHA256=""
    while IFS='|' read -r NAME URL EXPECTED_SHA256; do
        [ -n "$NAME" ] || continue
        if [ -z "$URL" ] || [ -z "$EXPECTED_SHA256" ]; then
            echo "[ERROR] Invalid deb manifest entry '$NAME': expected name|url|sha256" >&2
            rm -rf -- "$STAGING_DIR"
            return 1
        fi

        if ! __deb_cache_fetch "$NAME" "$URL" "$EXPECTED_SHA256"; then
            rm -rf -- "$STAGING_DIR"
            return 1
        fi

        if ! __extract_deb_data "$SYSTEMD_DEB_CACHE_DIR/${NAME}.deb" "$STAGING_DIR/$NAME"; then
            echo "[ERROR] Failed to unpack $NAME into $STAGING_DIR/$NAME" >&2
            rm -rf -- "$STAGING_DIR"
            return 1
        fi
    done <<<"$SYSTEMD_DEB_MANIFEST"

    local DEV_ROOT="$STAGING_DIR/libsystemd-dev"
    local DEV_LINK="$DEV_ROOT/usr/lib/$SYSTEMD_DEB_LIB_DIR/libsystemd.so"
    if [ ! -f "$DEV_ROOT/usr/include/systemd/sd-bus.h" ] ||
        { [ ! -e "$DEV_LINK" ] && [ ! -L "$DEV_LINK" ]; }; then
        echo "[ERROR] libsystemd-dev package is incomplete: sd-bus.h or libsystemd.so is missing" >&2
        rm -rf -- "$STAGING_DIR"
        return 1
    fi

    if ! cp -a "$DEV_ROOT/usr/include/systemd" "$SYSTEMD_SYSROOT_INCLUDE_DIR/"; then
        echo "[ERROR] Failed to install systemd headers into $SYSTEMD_SYSROOT_INCLUDE_DIR" >&2
        rm -rf -- "$STAGING_DIR"
        return 1
    fi

    if ! cp -a "$DEV_ROOT/usr/lib/$SYSTEMD_DEB_LIB_DIR/libsystemd.so" "$SYSTEMD_SYSROOT_LIB_DIR/"; then
        echo "[ERROR] Failed to install libsystemd.so into $SYSTEMD_SYSROOT_LIB_DIR" >&2
        rm -rf -- "$STAGING_DIR"
        return 1
    fi

    local PACKAGE_NAME=""
    local PACKAGE_ROOT=""
    local LIBRARY_DIR=""
    for PACKAGE_NAME in libsystemd0 libcap2 libgcrypt20 libgpg-error0 liblz4-1 liblzma5 libzstd1; do
        PACKAGE_ROOT="$STAGING_DIR/$PACKAGE_NAME"
        [ -d "$PACKAGE_ROOT" ] || continue
        for LIBRARY_DIR in "$PACKAGE_ROOT/lib/$SYSTEMD_DEB_LIB_DIR" "$PACKAGE_ROOT/usr/lib/$SYSTEMD_DEB_LIB_DIR"; do
            [ -d "$LIBRARY_DIR" ] || continue
            find "$LIBRARY_DIR" -maxdepth 1 \( -type f -o -type l \) -name '*.so*' \
                -exec cp -a -t "$SYSTEMD_SYSROOT_LIB_DIR" {} + || {
                echo "[ERROR] Failed to copy runtime libraries from $LIBRARY_DIR" >&2
                rm -rf -- "$STAGING_DIR"
                return 1
            }
        done
    done

    local REQUIRED_LIBRARY=""
    for REQUIRED_LIBRARY in libsystemd.so libsystemd.so.0 libcap.so.2 libgcrypt.so.20 \
        libgpg-error.so.0 liblz4.so.1 liblzma.so.5 libzstd.so.1; do
        if [ ! -e "$SYSTEMD_SYSROOT_LIB_DIR/$REQUIRED_LIBRARY" ]; then
            echo "[ERROR] Missing $SYSTEMD_SYSROOT_LIB_DIR/$REQUIRED_LIBRARY after unpacking the libsystemd debs" >&2
            rm -rf -- "$STAGING_DIR"
            return 1
        fi
    done

    rm -rf -- "$STAGING_DIR"

    if [ ! -f "$SYSTEMD_SYSROOT_INCLUDE_DIR/systemd/sd-bus.h" ]; then
        echo "[ERROR] Missing $SYSTEMD_SYSROOT_INCLUDE_DIR/systemd/sd-bus.h after unpacking the libsystemd deb" >&2
        return 1
    fi

    if [ ! -e "$SYSTEMD_SYSROOT_LIB_DIR/libsystemd.so" ]; then
        echo "[ERROR] Missing $SYSTEMD_SYSROOT_LIB_DIR/libsystemd.so after unpacking the libsystemd deb" >&2
        return 1
    fi

    echo "[INFO] libsystemd sysroot is ready at $SYSTEMD_SYSROOT_DIR"
    return 0
}

__build_target() {
    local MODE="$1"
    local BUILD_TYPE="Release"
    local COMMON_SIZE_FLAGS="-Os -ffunction-sections -fdata-sections"
    local RELEASE_SIZE_FLAGS="$COMMON_SIZE_FLAGS -DNDEBUG -fno-asynchronous-unwind-tables"
    local CXX_FLAGS="$COMMON_SIZE_FLAGS"
    local C_FLAGS="$COMMON_SIZE_FLAGS"
    local GLOBAL_EXE_LINKER_FLAGS="-Wl,--gc-sections"
    local CONFIG_EXE_LINKER_FLAGS="$GLOBAL_EXE_LINKER_FLAGS"
    local BUILD_TESTING="ON"
    local NANOKVM_GO_RELEASE_BUILD="OFF"
    local NANOKVM_GO_DEBUG_BUILD="OFF"
    local STRIP_RELEASE_BINARY="OFF"

    local COLOR="\033[0;32m"
    case "$MODE" in
    release)
        COLOR="\033[0;33m"
        CXX_FLAGS="$RELEASE_SIZE_FLAGS -fvisibility=hidden -fvisibility-inlines-hidden"
        C_FLAGS="$RELEASE_SIZE_FLAGS -fvisibility=hidden"
        # 4 KiB max page size matches the armv7 Linux kernel page and avoids
        # the default 64 KiB LOAD-segment alignment hole trapping size wins.
        CONFIG_EXE_LINKER_FLAGS="-s $GLOBAL_EXE_LINKER_FLAGS -Wl,-z,max-page-size=4096"
        BUILD_TESTING="OFF"
        NANOKVM_GO_RELEASE_BUILD="ON"
        STRIP_RELEASE_BINARY="ON"
        ;;
    debug)
        COLOR="\033[0;36m"
        BUILD_TYPE="RelWithDebInfo"
        NANOKVM_GO_DEBUG_BUILD="ON"
        CXX_FLAGS="-O2 -g -fno-omit-frame-pointer -ffunction-sections -fdata-sections"
        C_FLAGS="-O2 -g -fno-omit-frame-pointer -ffunction-sections -fdata-sections"
        CONFIG_EXE_LINKER_FLAGS="$GLOBAL_EXE_LINKER_FLAGS"
        ;;
    esac

    # Resolve sd-bus headers and libsystemd from the kvm_ui-local sysroot, so
    # target builds do not depend on the toolchain sysroot already carrying the
    # unpacked libsystemd debs. Explicit -isystem/-L entries are searched before
    # the compiler's built-in sysroot defaults, so the local copies win.
    local SYSTEMD_SYSROOT_INCLUDE_FLAGS="-isystem $SYSTEMD_SYSROOT_INCLUDE_DIR"
    local SYSTEMD_SYSROOT_LIB_FLAGS="-L$SYSTEMD_SYSROOT_LIB_DIR"
    CXX_FLAGS="$CXX_FLAGS $SYSTEMD_SYSROOT_INCLUDE_FLAGS"
    C_FLAGS="$C_FLAGS $SYSTEMD_SYSROOT_INCLUDE_FLAGS"
    GLOBAL_EXE_LINKER_FLAGS="$GLOBAL_EXE_LINKER_FLAGS $SYSTEMD_SYSROOT_LIB_FLAGS"
    CONFIG_EXE_LINKER_FLAGS="$CONFIG_EXE_LINKER_FLAGS $SYSTEMD_SYSROOT_LIB_FLAGS"

    echo -e "${COLOR}===== Building mode: $MODE =====\033[0m"

    mkdir -p "$PROJECT_PATH/build"
    cd "$PROJECT_PATH/build" || return 1
    __ensure_systemd_sysroot || {
        cd "$PROJECT_PATH" || true
        return 1
    }
    source ../conan/conanbuild.sh || return 1
    cmake .. \
        -DCMAKE_TOOLCHAIN_FILE=../conan/conan_toolchain.cmake \
        -DCMAKE_BUILD_TYPE="$BUILD_TYPE" \
        -DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
        -DBUILD_TESTING="$BUILD_TESTING" \
        -DNANOKVM_GO_SIMULATOR_BUILD=OFF \
        -DNANOKVM_GO_ENABLE_BACKGROUND_IMAGE=OFF \
        -DNANOKVM_GO_RELEASE_BUILD="$NANOKVM_GO_RELEASE_BUILD" \
        -DNANOKVM_GO_DEBUG_BUILD="$NANOKVM_GO_DEBUG_BUILD" \
        -DNANOKVM_GO_STRIP_RELEASE_BINARY="$STRIP_RELEASE_BINARY" \
        -DCMAKE_EXE_LINKER_FLAGS="$GLOBAL_EXE_LINKER_FLAGS" \
        -DCMAKE_C_FLAGS_RELEASE="$C_FLAGS" \
        -DCMAKE_CXX_FLAGS_RELEASE="$CXX_FLAGS" \
        -DCMAKE_EXE_LINKER_FLAGS_RELEASE="$CONFIG_EXE_LINKER_FLAGS" \
        -DCMAKE_C_FLAGS_RELWITHDEBINFO="$C_FLAGS" \
        -DCMAKE_CXX_FLAGS_RELWITHDEBINFO="$CXX_FLAGS" \
        -DCMAKE_EXE_LINKER_FLAGS_RELWITHDEBINFO="$CONFIG_EXE_LINKER_FLAGS" || {
        cd "$PROJECT_PATH" || true
        source conan/deactivate_conanbuild.sh || true
        return 1
    }
    cmake --build . --parallel "$(nproc)" || {
        cd "$PROJECT_PATH" || true
        source conan/deactivate_conanbuild.sh || true
        return 1
    }
    cd "$PROJECT_PATH" || return 1
    source conan/deactivate_conanbuild.sh || return 1

    echo -e "${COLOR}===== Finished building mode: $MODE =====\033[0m"
    return 0
}

__deactivate_simulator_conanbuild() {
    if [ -f "$SIMULATOR_CONAN_DIR/deactivate_conanbuild.sh" ]; then
        source "$SIMULATOR_CONAN_DIR/deactivate_conanbuild.sh"
    fi
}

__build_simulator() {
    if [ ! -f "$SIMULATOR_CONAN_DIR/conanbuild.sh" ] || \
        [ ! -f "$SIMULATOR_CONAN_DIR/conan_toolchain.cmake" ]; then
        echo "[ERROR] Simulator Conan files are missing; run '$0 simulator' first" >&2
        return 1
    fi

    source "$SIMULATOR_CONAN_DIR/conanbuild.sh" || return 1
    cmake -S "$PROJECT_PATH" -B "$SIMULATOR_BUILD_DIR" \
        -DCMAKE_TOOLCHAIN_FILE="$SIMULATOR_CONAN_DIR/conan_toolchain.cmake" \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
        -DBUILD_TESTING=ON \
        -DNANOKVM_GO_SIMULATOR_BUILD=ON \
        -DNANOKVM_GO_ENABLE_BACKGROUND_IMAGE=OFF \
        -DNANOKVM_GO_RELEASE_BUILD=OFF \
        -DNANOKVM_GO_DEBUG_BUILD=OFF \
        -DNANOKVM_GO_STRIP_RELEASE_BINARY=OFF || {
        __deactivate_simulator_conanbuild || true
        return 1
    }
    cmake --build "$SIMULATOR_BUILD_DIR" --parallel "$(nproc)" || {
        __deactivate_simulator_conanbuild || true
        return 1
    }
    __deactivate_simulator_conanbuild || return 1

    echo "[INFO] Finished simulator build"
}

build_target() {
    local MODE="$1"
    if ! __ensure_lvgl_repo || ! __verify_lvgl_commit || ! __check_conan2_env || ! build_3rd_lib "$MODE" || ! __build_target "$MODE"; then
        echo "[ERROR] Build failed"
        return 1
    fi
    return 0
}

rebuild_target() {
    local MODE="$1"
    if ! __ensure_lvgl_repo || ! __verify_lvgl_commit || ! __check_conan2_env || ! clean_target || ! rebuild_3rd_lib "$MODE" || ! __build_target "$MODE"; then
        echo "[ERROR] Rebuild failed"
        return 1
    fi
    return 0
}

build_simulator_target() {
    if ! simulator_fetch_lvgl || ! simulator_fetch_conan || ! __build_simulator; then
        echo "[ERROR] Simulator build failed"
        return 1
    fi
    return 0
}

release_target() {
    local MODE="$1"
    if ! __is_repo_clean; then
        return 1
    fi
    build_target "$MODE"
}

release_rebuild_target() {
    local MODE="$1"
    if ! __is_repo_clean; then
        return 1
    fi
    rebuild_target "$MODE"
}

minibuild() {
    local MODE="$1"
    __build_target "$1"
}

simulator_minibuild() {
    __build_simulator
}

# Fetch-only entry points: CI wraps these individually so a transient download
# failure is retried without re-running the compile. `simulator` composes them.
simulator_fetch_lvgl() {
    __ensure_lvgl_repo && __verify_lvgl_commit
}

simulator_fetch_conan() {
    __check_conan2_env && build_simulator_3rd_lib
}

simulator_test() {
    if [ ! -d "$SIMULATOR_BUILD_DIR" ]; then
        echo "[ERROR] Simulator build directory is missing: $SIMULATOR_BUILD_DIR" >&2
        return 1
    fi
    ctest --test-dir "$SIMULATOR_BUILD_DIR" --output-on-failure
}

simulator_run() {
    local simulator_binary="$SIMULATOR_BUILD_DIR/kvm_ui"
    if [ ! -x "$simulator_binary" ]; then
        echo "[ERROR] Simulator binary is missing: $simulator_binary" >&2
        echo "[ERROR] Run '$0 simulator' first" >&2
        return 1
    fi
    exec "$simulator_binary"
}

builder_main() {
    local CMD="$1"
    local INPUT_MODE="$2"
    local RESOLVE_INPUT_MODE="$INPUT_MODE"

    local MODE
    case "$CMD" in
    simulator | simulator_fetch_lvgl | simulator_fetch_conan | simulator_minibuild | simulator_test | simulator_run | simulator_clean)
        if [ -n "$INPUT_MODE" ]; then
            echo "[ERROR] '$CMD' does not accept a target build mode" >&2
            return 1
        fi
        ;;
    *)
        if [[ "$CMD" = "build" || "$CMD" = "release" || "$CMD" = "release_rb" ]] && [ -z "$RESOLVE_INPUT_MODE" ]; then
            RESOLVE_INPUT_MODE="release"
        fi
        MODE=$(__resolve_mode "$RESOLVE_INPUT_MODE") || return 1
        ;;
    esac

    case "$CMD" in
    help)
        echo "C/C++ Project Builder"
        echo ""
        echo "Usage:"
        echo "  $0 <command> [mode]"
        echo ""
        echo "Commands:"
        printf "  %-12s %s\n" "build" "Build project (defaults to release; pass debug for debug)"
        printf "  %-12s %s\n" "rebuild" "Clean and rebuild project"
        printf "  %-12s %s\n" "release" "Build release mode only if repository is clean"
        printf "  %-12s %s\n" "release_rb" "Clean & rebuild release mode only if repository is clean"
        printf "  %-12s %s\n" "minibuild" "Internal build only (skip Conan)"
        printf "  %-12s %s\n" "clean" "Remove ./build and Conan artifacts"
        printf "  %-12s %s\n" "simulator" "Build the isolated native simulator"
        printf "  %-12s %s\n" "simulator_fetch_lvgl" "Fetch the pinned lvgl dependency (skip Conan/compile)"
        printf "  %-12s %s\n" "simulator_fetch_conan" "Install simulator Conan dependencies (skip compile)"
        printf "  %-12s %s\n" "simulator_minibuild" "Rebuild the configured native simulator (skip Conan)"
        printf "  %-12s %s\n" "simulator_test" "Run tests from build-simulator/"
        printf "  %-12s %s\n" "simulator_run" "Run only build-simulator/kvm_ui (does not build)"
        printf "  %-12s %s\n" "simulator_clean" "Remove build-simulator/ only"
        printf "  %-12s %s\n" "build_3rd" "Build 3rd-party libraries"
        printf "  %-12s %s\n" "rebuild_3rd" "Force rebuild 3rd-party libraries"
        echo ""
        echo "Mode handling:"
        echo "  - Available modes: debug | release"
        echo "  - If [mode] is provided: use it AND store it to build/mode.conf"
        echo "  - If omitted: build/release/release_rb use 'release'; other target commands use build/mode.conf"
        echo "  - If build/mode.conf doesn't exist: default to 'release'"
        echo "  - Invalid stored modes are rejected; edit build/mode.conf to debug or release"
        echo ""
        echo "Examples:"
        echo "  $0 build              # Build release mode"
        echo "  $0 build debug        # Build with debug mode and store it"
        echo "  $0 build release      # Build stripped release binary with locked CLI"
        echo "  $0 release            # Build release mode if repository is clean"
        echo "  $0 rebuild debug      # Rebuild debug mode and store it"
        echo ""
        return 0
        ;;
    build)
        build_target "$MODE"
        ;;
    rebuild)
        rebuild_target "$MODE"
        ;;
    release)
        release_target "$MODE"
        ;;
    release_rb)
        release_rebuild_target "$MODE"
        ;;
    minibuild)
        minibuild "$MODE"
        ;;
    clean)
        clean_target
        ;;
    simulator)
        build_simulator_target
        ;;
    simulator_fetch_lvgl)
        simulator_fetch_lvgl
        ;;
    simulator_fetch_conan)
        simulator_fetch_conan
        ;;
    simulator_minibuild)
        simulator_minibuild
        ;;
    simulator_test)
        simulator_test
        ;;
    simulator_run)
        simulator_run
        ;;
    simulator_clean)
        clean_simulator_target
        ;;
    build_3rd)
        build_3rd_lib "$MODE"
        ;;
    rebuild_3rd)
        rebuild_3rd_lib "$MODE"
        ;;
    format)
        echo "Error: Unknown command '$1'"
        echo "Run '$0 help' for usage information"
        return 1
        ;;
    *)
        if __run_external_plugin "$CMD" "$INPUT_MODE"; then
            return 0
        fi

        echo "Error: Unknown command '$1'"
        echo "Run '$0 help' for usage information"
        return 1
        ;;
    esac
    return $?
}

builder_main "$@"
