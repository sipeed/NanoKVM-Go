#!/usr/bin/env bash

set -euo pipefail

readonly EXECUTABLE_NAME="NanoKVM-Server"
readonly TARGET_GOOS="linux"
readonly TARGET_GOARCH="arm"
readonly TARGET_GOARM="7"
readonly TOOLCHAIN_INI="${TOOLCHAIN_INI:-../support/toolchains/toolchain.ini}"
readonly GETCONFIG_SCRIPT="../support/scripts/getconfig.py"

CC_BIN=""
CXX_BIN=""
SYSROOT_DIR=""

cleanup() {
    if [[ -f "dl_lib/libkvm.so" || -f "dl_lib/libkvm.so.0" ]]; then
        log_info "Cleaning up temporary files..."
        rm -f dl_lib/libkvm.so*
    fi
}

trap cleanup EXIT

log_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

log_error() {
    echo -e "\033[31m[ERROR]\033[0m $1" >&2
}

find_python() {
    if command -v python3 &>/dev/null; then
        echo "python3"
        return 0
    fi

    if command -v python &>/dev/null; then
        echo "python"
        return 0
    fi

    return 1
}

read_toolchain_value() {
    local key="$1"
    local python_cmd

    python_cmd=$(find_python) || {
        log_error "Python is not installed or not in PATH"
        return 1
    }

    "$python_cmd" "$GETCONFIG_SCRIPT" "$TOOLCHAIN_INI" "toolchain" "$key"
}

resolve_executable() {
    local candidate="$1"

    if [[ "$candidate" == */* ]]; then
        if [[ -x "$candidate" ]]; then
            echo "$candidate"
            return 0
        fi
        return 1
    fi

    command -v "$candidate"
}

resolve_configured_executable() {
    local candidate="$1"
    local base_dir

    if [[ "$candidate" == /* || "$candidate" != */* ]]; then
        resolve_executable "$candidate"
        return
    fi

    base_dir=$(cd "$(dirname "$TOOLCHAIN_INI")" &>/dev/null && pwd -P)
    resolve_executable "${base_dir}/${candidate}"
}

check_dependencies() {
    log_info "Checking dependencies..."

    if ! command -v go &>/dev/null; then
        log_error "Go is not installed or not in PATH"
        return 1
    fi

    if ! command -v patchelf &>/dev/null; then
        log_error "patchelf is not installed or not in PATH"
        return 1
    fi

    if [[ -z "${CC:-}" ]]; then
        if [[ ! -f "$TOOLCHAIN_INI" ]]; then
            log_error "Toolchain configuration file not found: $TOOLCHAIN_INI"
            log_error "Set CC, CXX, and SYSROOT to build without toolchain.ini"
            return 1
        fi

        if [[ ! -f "$GETCONFIG_SCRIPT" ]]; then
            log_error "Toolchain configuration script not found: $GETCONFIG_SCRIPT"
            return 1
        fi

        find_python >/dev/null || {
            log_error "Python is not installed or not in PATH"
            return 1
        }
    fi

    log_info "All dependencies are available"
}

check_project_root() {
    local required_dirs=("common" "config" "dl_lib" "include" "logger" "middleware" "proto" "router" "service" "utils")

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_error "Missing required project directory: $dir"
            log_error "This script must be executed from the server project root directory."
            return 1
        fi
    done

    log_info "Server project root verified"
}

load_toolchain() {
    local cc_config="${CC:-}"
    local cxx_config="${CXX:-}"
    local cc_from_env="false"
    local machine

    if [[ -n "$cc_config" ]]; then
        cc_from_env="true"
    fi

    if [[ -z "$cc_config" ]]; then
        cc_config=$(read_toolchain_value "cc")
    fi

    if [[ -z "$cxx_config" && "$cc_from_env" == "false" && -f "$TOOLCHAIN_INI" && -f "$GETCONFIG_SCRIPT" ]]; then
        cxx_config=$(read_toolchain_value "cxx")
    fi

    if [[ -z "$cc_config" ]]; then
        log_error "Missing ARMv7 C compiler. Set CC or configure toolchain.cc in $TOOLCHAIN_INI"
        return 1
    fi

    if [[ "$cc_from_env" == "true" ]]; then
        CC_BIN=$(resolve_executable "$cc_config") || {
            log_error "ARMv7 C compiler not found or not executable: $cc_config"
            return 1
        }
    else
        CC_BIN=$(resolve_configured_executable "$cc_config") || {
            log_error "ARMv7 C compiler not found or not executable: $cc_config"
            return 1
        }
    fi

    if [[ -z "$cxx_config" ]]; then
        local guessed_cxx="${CC_BIN%gcc}g++"
        if [[ "$guessed_cxx" != "$CC_BIN" && -x "$guessed_cxx" ]]; then
            cxx_config="$guessed_cxx"
        fi
    fi

    if [[ -z "$cxx_config" ]]; then
        log_error "Missing ARMv7 C++ compiler. Set CXX or configure toolchain.cxx in $TOOLCHAIN_INI"
        return 1
    fi

    if [[ "$cc_from_env" == "true" ]]; then
        CXX_BIN=$(resolve_executable "$cxx_config") || {
            log_error "ARMv7 C++ compiler not found or not executable: $cxx_config"
            return 1
        }
    else
        CXX_BIN=$(resolve_configured_executable "$cxx_config") || {
            log_error "ARMv7 C++ compiler not found or not executable: $cxx_config"
            return 1
        }
    fi

    machine=$("$CC_BIN" -dumpmachine 2>/dev/null || true)
    if [[ -z "$machine" ]]; then
        log_error "Failed to detect target machine from compiler: $CC_BIN"
        return 1
    fi

    if [[ "$machine" == aarch64* ]]; then
        log_error "Configured compiler targets ARM64 ($machine), but NanoKVM now targets ARMv7"
        return 1
    fi

    if [[ "$machine" != arm* ]]; then
        log_error "Configured compiler target is not ARMv7-compatible: $machine"
        return 1
    fi

    log_info "Using CC: $CC_BIN"
    log_info "Using CXX: $CXX_BIN"
    log_info "Using compiler target: $machine"
}

resolve_sysroot() {
    local machine
    local printed_sysroot
    local toolchain_root
    local candidates=()

    if [[ -n "${SYSROOT:-}" ]]; then
        SYSROOT_DIR="$SYSROOT"
    else
        printed_sysroot=$("$CC_BIN" -print-sysroot 2>/dev/null || true)
        if [[ -n "$printed_sysroot" && "$printed_sysroot" != "/" && -d "$printed_sysroot" ]]; then
            SYSROOT_DIR="$printed_sysroot"
        fi
    fi

    if [[ -z "$SYSROOT_DIR" ]]; then
        machine=$("$CC_BIN" -dumpmachine 2>/dev/null || true)
        toolchain_root=$(dirname "$(dirname "$CC_BIN")")
        candidates=(
            "${toolchain_root}/${machine}/libc"
            "${toolchain_root}/${machine}/sysroot"
            "${toolchain_root}/arm-linux-gnueabihf/libc"
            "${toolchain_root}/arm-none-linux-gnueabihf/libc"
            "${toolchain_root}/sysroot"
        )

        for candidate in "${candidates[@]}"; do
            if [[ -d "$candidate" ]]; then
                SYSROOT_DIR="$candidate"
                break
            fi
        done
    fi

    if [[ -z "$SYSROOT_DIR" || ! -d "$SYSROOT_DIR" ]]; then
        log_error "Failed to locate ARMv7 sysroot. Set SYSROOT to the target rootfs/sysroot path."
        return 1
    fi

    log_info "Using sysroot: $SYSROOT_DIR"
}

build_link_stub() {
    log_info "Building link stub library..."

    "$CXX_BIN" --sysroot="$SYSROOT_DIR" -Iinclude -fPIC -shared -o dl_lib/libkvm.so dl_lib/kvmimpl.cpp || {
        log_error "Failed to build dl_lib/libkvm.so"
        return 1
    }

    cp dl_lib/libkvm.so dl_lib/libkvm.so.0 || {
        log_error "Failed to copy dl_lib/libkvm.so.0"
        return 1
    }
}

build_application() {
    local version
    local build_time
    local commit
    local go_version
    local build_host
    local build_user
    local git_branch
    local cgo_cflags
    local cgo_ldflags

    version=$(cat version.txt 2>/dev/null || echo "dev")
    build_time=$(date +"%Y-%m-%d %H:%M:%S")
    commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    go_version=$(go version | awk '{print $3}')
    build_host="${BUILD_HOST:-${HOSTNAME:-unknown}}"
    build_user="${BUILD_USER:-$(whoami)@${build_host}}"
    git_branch=$(git branch --show-current 2>/dev/null || echo "unknown")

    cgo_cflags="${CGO_CFLAGS:-} --sysroot=${SYSROOT_DIR} -I${SYSROOT_DIR}/usr/include"
    cgo_ldflags="${CGO_LDFLAGS:-} --sysroot=${SYSROOT_DIR} -L${SYSROOT_DIR}/usr/lib -L${SYSROOT_DIR}/usr/lib/arm-linux-gnueabihf -L${SYSROOT_DIR}/lib -L${SYSROOT_DIR}/lib/arm-linux-gnueabihf"

    log_info "Building $EXECUTABLE_NAME"
    log_info "Version: $version"
    log_info "Commit: $commit"
    log_info "Build time: $build_time"
    log_info "GOOS: $TARGET_GOOS"
    log_info "GOARCH: $TARGET_GOARCH"
    log_info "GOARM: $TARGET_GOARM"
    log_info "CGO_CFLAGS: $cgo_cflags"
    log_info "CGO_LDFLAGS: $cgo_ldflags"

    CGO_ENABLED=1 \
        GOOS="$TARGET_GOOS" \
        GOARCH="$TARGET_GOARCH" \
        GOARM="$TARGET_GOARM" \
        CC="$CC_BIN" \
        CXX="$CXX_BIN" \
        CGO_CFLAGS="$cgo_cflags" \
        CGO_LDFLAGS="$cgo_ldflags" \
        go build -trimpath -buildvcs=false -o "$EXECUTABLE_NAME" -ldflags "\
        -X 'main.Version=$version' \
        -X 'main.BuildTime=$build_time' \
        -X 'main.Commit=$commit' \
        -X 'main.GoVersion=$go_version' \
        -X 'main.GitBranch=$git_branch' \
        -X 'main.BuildUser=$build_user'" || {
        log_error "Go build failed"
        return 1
    }
}

patch_runtime_path() {
    log_info "Adding runtime library path to $EXECUTABLE_NAME"

    patchelf --add-rpath '$ORIGIN/dl_lib' "$EXECUTABLE_NAME" || {
        log_error "Failed to add runtime library path"
        return 1
    }
}

main() {
    log_info "Starting ARMv7 build for NanoKVM"

    check_dependencies
    check_project_root
    load_toolchain
    resolve_sysroot
    build_link_stub
    build_application
    patch_runtime_path

    log_info "Build completed successfully"
    log_info "Executable: $EXECUTABLE_NAME"
}

main "$@"
