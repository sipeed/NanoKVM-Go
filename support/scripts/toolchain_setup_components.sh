#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)
ROOT_DIR=$(cd "${SCRIPT_DIR}/../.." &>/dev/null && pwd -P)
TOOLCHAIN_DIR="${SCRIPT_DIR}/../toolchains"
TOOLCHAIN_INI="${TOOLCHAIN_DIR}/toolchain.ini"

CONF_FILE=""
SECTION="toolchain"
TC_NAME=""
TC_URL=""
TC_SHA256=""
PROFILE_NAME=""
TARGET_ARCH=""
CPP_STD=""
COMPILER_VERSION=""
CC_PATH=""
CXX_PATH=""
LD_PATH=""
ACTION=""
INTERACTIVE=1

usage() {
    cat <<EOF
Usage: $(basename "$0") [--config <ini>] [--switch|--install|--uninstall]

Generate NanoKVM-Go component Conan profiles under:
  libkvm/profile/nanokvm-go
  kvm_vin/profile/nanokvm-go
  kvm_ui/profile/nanokvm-go

When no action is provided, this script keeps the interactive menu.
EOF
}

resolve_config_path() {
    local config="$1"

    if [[ "$config" = /* ]]; then
        printf '%s\n' "$config"
    elif [[ -f "$config" ]]; then
        cd "$(dirname "$config")" &>/dev/null && printf '%s/%s\n' "$(pwd -P)" "$(basename "$config")"
    else
        printf '%s/%s\n' "$SCRIPT_DIR" "$config"
    fi
}

parse_args() {
    while (($# > 0)); do
        case "$1" in
        --config)
            if (($# < 2)); then
                echo "Error: --config requires a file" >&2
                exit 1
            fi
            CONF_FILE=$(resolve_config_path "$2")
            shift 2
            ;;
        --switch | switch)
            ACTION="switch"
            INTERACTIVE=0
            shift
            ;;
        --install | install)
            ACTION="install"
            INTERACTIVE=0
            shift
            ;;
        --uninstall | uninstall)
            ACTION="uninstall"
            INTERACTIVE=0
            shift
            ;;
        -h | --help | help)
            usage
            exit 0
            ;;
        *)
            echo "Error: unknown argument: $1" >&2
            usage >&2
            exit 1
            ;;
        esac
    done
}

select_config_file() {
    shopt -s nullglob
    local all_inis=("${SCRIPT_DIR}"/*.ini)
    local choices=()
    local f

    for f in "${all_inis[@]}"; do
        case "$(basename "$f")" in
        toolchain.ini | config.ini)
            ;;
        *)
            choices+=("$f")
            ;;
        esac
    done
    shopt -u nullglob

    if ((${#choices[@]} == 0)); then
        echo "Error: no component toolchain configuration files found in $SCRIPT_DIR" >&2
        exit 1
    fi

    if ((${#choices[@]} == 1)); then
        CONF_FILE="${choices[0]}"
        echo "Auto-selected config: $(basename "$CONF_FILE")"
        return
    fi

    echo "------------------------------------------------"
    echo "   Select Target Configuration"
    echo "------------------------------------------------"
    PS3="Please select a target config: "
    select choice in "${choices[@]}" "Exit"; do
        if [[ "$choice" == "Exit" ]]; then
            exit 0
        elif [[ -n "$choice" ]]; then
            CONF_FILE="$choice"
            break
        else
            echo "Invalid selection."
        fi
    done
}

get_config() {
    "${SCRIPT_DIR}/getconfig.py" "$CONF_FILE" "$SECTION" "$1"
}

load_config() {
    echo "Loading configuration from: $CONF_FILE..."

    if [[ ! -f "$CONF_FILE" ]]; then
        echo "Error: file not found: $CONF_FILE" >&2
        exit 1
    fi

    TC_NAME=$(get_config "name")
    TC_URL=$(get_config "url")
    TC_SHA256=$(get_config "sha256")

    PROFILE_NAME=$(get_config "profile_name")
    [[ -z "$PROFILE_NAME" ]] && PROFILE_NAME="nanokvm-go"

    TARGET_ARCH=$(get_config "arch")
    [[ -z "$TARGET_ARCH" ]] && TARGET_ARCH="armv7hf"

    CPP_STD=$(get_config "cpp_std")
    [[ -z "$CPP_STD" ]] && CPP_STD="gnu20"

    COMPILER_VERSION=$(get_config "compiler_version")

    CC_PATH=$(get_config "cc")
    CXX_PATH=$(get_config "cxx")
    LD_PATH=$(get_config "ld")

    if [[ -z "$TC_NAME" || -z "$TC_URL" || -z "$TC_SHA256" || -z "$CC_PATH" || -z "$CXX_PATH" || -z "$LD_PATH" ]]; then
        echo "Error: missing required toolchain fields in $CONF_FILE" >&2
        exit 1
    fi
}

check_executable() {
    local exe_path="$1"
    if [[ ! -x "$exe_path" ]]; then
        return 1
    fi
    if ! timeout 5s "$exe_path" --version &>/dev/null; then
        return 1
    fi
    return 0
}

validate_toolchain() {
    local target_dir="$1"
    local cc="${target_dir}/${CC_PATH}"
    local cxx="${target_dir}/${CXX_PATH}"
    local ld="${target_dir}/${LD_PATH}"
    local check_results=0

    check_executable "$cc" || check_results=$((check_results | 1))
    check_executable "$cxx" || check_results=$((check_results | 2))
    check_executable "$ld" || check_results=$((check_results | 4))

    if [[ $check_results -ne 0 ]]; then
        echo "Verification failed code: $check_results" >&2
        return 1
    fi

    echo "Toolchain binaries verified."
    return 0
}

detect_compiler_version() {
    local cc="$1"
    local version

    if [[ -n "$COMPILER_VERSION" ]]; then
        echo "$COMPILER_VERSION"
        return
    fi

    version=$("$cc" -dumpfullversion -dumpversion 2>/dev/null | head -n 1)
    version="${version%%.*}"
    if [[ -z "$version" ]]; then
        echo "Error: unable to detect compiler version from $cc" >&2
        exit 1
    fi

    echo "$version"
}

gen_toolchain_path() {
    local target_dir="$1"
    local cc="${target_dir}/${CC_PATH}"
    local cxx="${target_dir}/${CXX_PATH}"
    local ld="${target_dir}/${LD_PATH}"

    mkdir -p "$TOOLCHAIN_DIR"

    cat <<EOF >"$TOOLCHAIN_INI"
[toolchain]
cc = $cc
cxx = $cxx
ld = $ld
EOF
}

gen_conan_profile() {
    local target_dir="$1"
    local cc="${target_dir}/${CC_PATH}"
    local cxx="${target_dir}/${CXX_PATH}"
    local ld="${target_dir}/${LD_PATH}"
    local compiler_version

    compiler_version=$(detect_compiler_version "$cc")

    echo "Generating Conan profile file: $PROFILE_NAME..."

    cat <<EOF >"${SCRIPT_DIR}/${PROFILE_NAME}"
[settings]
os=Linux
arch=${TARGET_ARCH}
compiler=gcc
build_type=Release
compiler.cppstd=${CPP_STD}
compiler.libcxx=libstdc++
compiler.version=${compiler_version}
[buildenv]
CC=$cc
CXX=$cxx
LD=$ld
[conf]
tools.build:cflags=['-Os', '-ffunction-sections', '-fdata-sections']
tools.build:cxxflags=['-Os', '-ffunction-sections', '-fdata-sections', '-fvisibility=hidden']
tools.build:exelinkflags=['-Wl,--gc-sections', '-s']
EOF
}

link_active_profile() {
    local profile_path="${SCRIPT_DIR}/${PROFILE_NAME}"

    echo "Activating toolchain links..."
    echo "Linking component profiles to: $profile_path"

    mkdir -p "${ROOT_DIR}/libkvm/profile" "${ROOT_DIR}/kvm_vin/profile" "${ROOT_DIR}/kvm_ui/profile"

    ln -sf "$profile_path" "${ROOT_DIR}/libkvm/profile/nanokvm-go"
    ln -sf "$profile_path" "${ROOT_DIR}/kvm_vin/profile/nanokvm-go"
    ln -sf "$profile_path" "${ROOT_DIR}/kvm_ui/profile/nanokvm-go"
}

is_installed() {
    local target_dir="$1"
    [[ -d "${target_dir}/bin" ]]
}

do_uninstall() {
    local target_dir="${TOOLCHAIN_DIR}/${TC_NAME}"

    echo "------------------------------------------------"
    echo "   Action: Uninstall"
    echo "   Target: $TC_NAME"
    echo "   Profile: $PROFILE_NAME"
    echo "------------------------------------------------"

    echo "Removing toolchain directory..."
    rm -rf "$target_dir"

    echo "Cleaning up configurations..."
    rm -f "$TOOLCHAIN_INI"
    rm -f "${SCRIPT_DIR}/${PROFILE_NAME}"

    echo "Cleaning up symlinks..."
    rm -f "${ROOT_DIR}/libkvm/profile/nanokvm-go"
    rm -f "${ROOT_DIR}/kvm_vin/profile/nanokvm-go"
    rm -f "${ROOT_DIR}/kvm_ui/profile/nanokvm-go"

    echo "Uninstallation complete."
}

do_install() {
    local target_dir="${TOOLCHAIN_DIR}/${TC_NAME}"
    local temp_file
    local computed_sha256

    echo "------------------------------------------------"
    echo "   Action: Install / Update"
    echo "   Target: $TC_NAME"
    echo "   Arch:   $TARGET_ARCH"
    echo "------------------------------------------------"

    if is_installed "$target_dir"; then
        if ((INTERACTIVE)); then
            read -r -p "   Force reinstall? (This will delete the old one) [y/N]: " confirm
            if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
                echo "Skipped install; switching to existing toolchain."
                do_switch
                return
            fi
        fi
        rm -rf "$target_dir"
    fi

    mkdir -p "$target_dir"

    temp_file=$(mktemp "${TMPDIR:-/tmp}/toolchain.XXXXXX")
    trap 'rm -f "$temp_file"' EXIT

    echo "Downloading..."
    if ! curl -#L -o "$temp_file" "$TC_URL"; then
        echo "Download failed." >&2
        return 1
    fi

    echo "Verifying SHA256..."
    computed_sha256=$(sha256sum "$temp_file" | awk '{print $1}')
    if [[ "$computed_sha256" != "$TC_SHA256" ]]; then
        echo "Hash mismatch!" >&2
        return 1
    fi

    echo "Extracting..."
    if ! tar -xJf "$temp_file" -C "$target_dir" --strip-components=1; then
        echo "Extraction failed." >&2
        return 1
    fi
    rm -f "$temp_file"
    trap - EXIT

    if ! validate_toolchain "$target_dir"; then
        echo "Toolchain broken. Removing..." >&2
        rm -rf "$target_dir"
        return 1
    fi

    gen_toolchain_path "$target_dir"
    gen_conan_profile "$target_dir"
    link_active_profile

    echo "Installation successfully completed!"
}

do_switch() {
    local target_dir="${TOOLCHAIN_DIR}/${TC_NAME}"

    echo "------------------------------------------------"
    echo "   Action: Switch Toolchain"
    echo "   Switching to: $TC_NAME"
    echo "   Profile: $PROFILE_NAME"
    echo "------------------------------------------------"

    if ! is_installed "$target_dir"; then
        echo "Error: toolchain '$TC_NAME' is not installed." >&2
        echo "Please run support/scripts/toolchain_setup.sh first, or use --install." >&2
        return 1
    fi

    if ! validate_toolchain "$target_dir"; then
        echo "Error: toolchain binaries are missing or corrupted." >&2
        return 1
    fi

    gen_toolchain_path "$target_dir"
    gen_conan_profile "$target_dir"
    link_active_profile

    echo "Switched successfully! Active profile is now: $PROFILE_NAME"
}

run_interactive_menu() {
    echo ""
    echo "Target Selected: $TC_NAME"
    echo "Architecture:    $TARGET_ARCH"
    echo "Profile Name:    $PROFILE_NAME"
    echo "------------------------------------------"

    PS3="Choose an action: "
    local options=("Switch to this Toolchain" "Install / Reinstall" "Uninstall" "Exit")

    select opt in "${options[@]}"; do
        case "$opt" in
        "Switch to this Toolchain")
            do_switch
            break
            ;;
        "Install / Reinstall")
            do_install
            break
            ;;
        "Uninstall")
            do_uninstall
            break
            ;;
        "Exit")
            exit 0
            ;;
        *)
            echo "Invalid option."
            ;;
        esac
    done
}

main() {
    echo "=========================================="
    echo "      NanoKVM-Go Component Toolchain"
    echo "=========================================="

    parse_args "$@"

    if [[ -z "$CONF_FILE" ]]; then
        select_config_file
    fi

    load_config

    case "$ACTION" in
    switch)
        do_switch
        ;;
    install)
        do_install
        ;;
    uninstall)
        do_uninstall
        ;;
    "")
        run_interactive_menu
        ;;
    *)
        echo "Error: unknown action: $ACTION" >&2
        exit 1
        ;;
    esac
}

main "$@"
