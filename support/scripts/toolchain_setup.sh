#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)
CONFIG_FILE="${SCRIPT_DIR}/config.ini"
TOOLCHAIN_DIR="${SCRIPT_DIR}/../toolchains"
TOOLCHAIN_INI="${TOOLCHAIN_DIR}/toolchain.ini"
TARGET_TRIPLE="arm-none-linux-gnueabihf"
DEB_LIB_DIR="arm-linux-gnueabihf"

get_config() {
    "${SCRIPT_DIR}/getconfig.py" "$CONFIG_FILE" "$1" "$2"
}

require_command() {
    if ! command -v "$1" &>/dev/null; then
        echo "Error: required command not found: $1" >&2
        exit 1
    fi
}

refresh_component_profiles() {
    local component_config="${SCRIPT_DIR}/nanokvm-go.ini"
    local component_setup="${SCRIPT_DIR}/toolchain_setup_components.sh"

    if [[ ! -f "$component_config" || ! -x "$component_setup" ]]; then
        echo "Warning: component profile setup is unavailable; skipping Conan profile generation" >&2
        return
    fi

    echo "Refreshing component Conan profiles..."
    "$component_setup" --config "$component_config" --switch
}

verify_executable() {
    local path="$1"
    local name
    name=$(basename "$path")

    if [[ ! -x "$path" ]]; then
        echo "Error: executable not found: $path" >&2
        exit 1
    fi

    if ! "$path" --version &>/dev/null; then
        echo "Error: unable to execute $name" >&2
        exit 1
    fi

    echo "Verified: $name"
}

verify_checksum() {
    local file="$1"
    local expected="$2"
    local actual

    actual=$(sha256sum "$file" | awk '{print $1}')
    if [[ "$actual" != "$expected" ]]; then
        echo "Error: checksum mismatch" >&2
        echo "Expected: $expected" >&2
        echo "Actual:   $actual" >&2
        exit 1
    fi
}

extract_deb_data() {
    local deb="$1"
    local dest="$2"

    mkdir -p "$dest"
    (
        cd "$dest"
        ar x "$deb"
        if [[ -f data.tar.xz ]]; then
            tar xf data.tar.xz
        elif [[ -f data.tar.zst ]]; then
            require_command zstd
            zstd -d data.tar.zst -o data.tar
            tar xf data.tar
        elif [[ -f data.tar.gz ]]; then
            tar xzf data.tar.gz
        else
            echo "Error: unsupported deb data archive: $deb" >&2
            exit 1
        fi
    )
}

install_libopus() {
    local sysroot="$1"
    local libopus_url="$2"
    local libopus_dev_url="$3"
    local temp_dir
    temp_dir=$(mktemp -d)

    echo "Installing ARMv7 libopus into sysroot..."

    curl -fsSL -o "${temp_dir}/libopus0.deb" "$libopus_url"
    curl -fsSL -o "${temp_dir}/libopus-dev.deb" "$libopus_dev_url"

    extract_deb_data "${temp_dir}/libopus0.deb" "${temp_dir}/libopus0"
    extract_deb_data "${temp_dir}/libopus-dev.deb" "${temp_dir}/libopus-dev"

    mkdir -p "${sysroot}/usr/include" "${sysroot}/usr/lib"

    if [[ -d "${temp_dir}/libopus-dev/usr/include" ]]; then
        cp -a "${temp_dir}/libopus-dev/usr/include/." "${sysroot}/usr/include/"
    fi

    if [[ -d "${temp_dir}/libopus0/usr/lib/${DEB_LIB_DIR}" ]]; then
        cp -a "${temp_dir}/libopus0/usr/lib/${DEB_LIB_DIR}/libopus.so"* "${sysroot}/usr/lib/"
    fi

    if [[ -d "${temp_dir}/libopus-dev/usr/lib/${DEB_LIB_DIR}" ]]; then
        cp -a "${temp_dir}/libopus-dev/usr/lib/${DEB_LIB_DIR}/libopus.a" "${sysroot}/usr/lib/" 2>/dev/null || true
        cp -a "${temp_dir}/libopus-dev/usr/lib/${DEB_LIB_DIR}/pkgconfig" "${sysroot}/usr/lib/" 2>/dev/null || true
    fi

    rm -rf "$temp_dir"
}

install_libsystemd() {
    local sysroot="$1"
    local package_manifest="$2"
    local temp_dir
    temp_dir=$(mktemp -d)

    echo "Installing ARMv7 libsystemd development files into sysroot..."

    local package_name package_url package_sha256
    while IFS='|' read -r package_name package_url package_sha256; do
        [[ -n "$package_name" ]] || continue
        if [[ -z "$package_url" || -z "$package_sha256" ]]; then
            echo "Error: invalid libsystemd package manifest entry: $package_name" >&2
            exit 1
        fi

        local deb_path="${temp_dir}/${package_name}.deb"
        curl -fsSL -o "$deb_path" "$package_url"
        verify_checksum "$deb_path" "$package_sha256"
        extract_deb_data "$deb_path" "${temp_dir}/${package_name}"
    done < <(printf '%s\n' "$package_manifest")

    local dev_root="${temp_dir}/libsystemd-dev"
    if [[ ! -f "${dev_root}/usr/include/systemd/sd-bus.h" ||
          ! -L "${dev_root}/usr/lib/${DEB_LIB_DIR}/libsystemd.so" ]]; then
        echo "Error: ARMv7 libsystemd development package is incomplete" >&2
        exit 1
    fi

    mkdir -p "${sysroot}/usr/include" "${sysroot}/usr/lib/pkgconfig"
    cp -a "${dev_root}/usr/include/systemd" "${sysroot}/usr/include/"
    cp -a "${dev_root}/usr/lib/${DEB_LIB_DIR}/libsystemd.so" \
        "${sysroot}/usr/lib/"
    cp -a "${dev_root}/usr/lib/${DEB_LIB_DIR}/pkgconfig/libsystemd.pc" \
        "${sysroot}/usr/lib/pkgconfig/"

    local package_root library_dir
    for package_root in "${temp_dir}"/libsystemd0 "${temp_dir}"/libcap2 \
                        "${temp_dir}"/libgcrypt20 "${temp_dir}"/libgpg-error0 \
                        "${temp_dir}"/liblz4-1 "${temp_dir}"/liblzma5 \
                        "${temp_dir}"/libzstd1; do
        for library_dir in "${package_root}/lib/${DEB_LIB_DIR}" \
                           "${package_root}/usr/lib/${DEB_LIB_DIR}"; do
            [[ -d "$library_dir" ]] || continue
            find "$library_dir" -maxdepth 1 \
                \( -type f -o -type l \) -name '*.so*' \
                -exec cp -a -t "${sysroot}/usr/lib" {} +
        done
    done

    if [[ ! -e "${sysroot}/usr/lib/libsystemd.so" ]]; then
        echo "Error: ARMv7 libsystemd linker input was not installed" >&2
        exit 1
    fi

    rm -rf "$temp_dir"
}

main() {
    require_command curl
    require_command tar
    require_command ar
    require_command sha256sum

    local name url sha256 cc_path cxx_path ld_path
    local libopus_url libopus_dev_url libsystemd_packages

    name=$(get_config toolchain name)
    url=$(get_config toolchain url)
    sha256=$(get_config toolchain sha256)
    cc_path=$(get_config toolchain cc)
    cxx_path=$(get_config toolchain cxx)
    ld_path=$(get_config toolchain ld)
    libopus_url=$(get_config libopus url)
    libopus_dev_url=$(get_config libopus dev_url)
    libsystemd_packages=$(get_config libsystemd packages)

    if [[ -z "$name" || -z "$url" || -z "$sha256" || -z "$cc_path" || -z "$cxx_path" || -z "$ld_path" ]]; then
        echo "Error: missing ARMv7 toolchain configuration in $CONFIG_FILE" >&2
        exit 1
    fi

    if [[ -z "$libopus_url" || -z "$libopus_dev_url" ]]; then
        echo "Error: missing ARMv7 libopus package configuration in $CONFIG_FILE" >&2
        exit 1
    fi

    if [[ -z "$libsystemd_packages" ]]; then
        echo "Error: missing ARMv7 libsystemd package configuration in $CONFIG_FILE" >&2
        exit 1
    fi

    mkdir -p "$TOOLCHAIN_DIR"

    local install_dir="${TOOLCHAIN_DIR}/${name}"
    local temp_file
    temp_file=$(mktemp "${TMPDIR:-/tmp}/toolchain.XXXXXX")

    rm -rf "$install_dir"
    mkdir -p "$install_dir"

    echo "Downloading ARMv7 toolchain..."
    curl -#L -o "$temp_file" "$url"

    echo "Verifying toolchain checksum..."
    verify_checksum "$temp_file" "$sha256"

    echo "Installing toolchain to $install_dir..."
    tar -xJf "$temp_file" -C "$install_dir" --strip-components=1
    rm -f "$temp_file"

    local cc="${install_dir}/${cc_path}"
    local cxx="${install_dir}/${cxx_path}"
    local ld="${install_dir}/${ld_path}"
    local sysroot="${install_dir}/${TARGET_TRIPLE}/libc"

    verify_executable "$cc"
    verify_executable "$cxx"
    verify_executable "$ld"

    if [[ ! -d "$sysroot" ]]; then
        echo "Error: ARMv7 sysroot not found: $sysroot" >&2
        exit 1
    fi

    install_libopus "$sysroot" "$libopus_url" "$libopus_dev_url"
    install_libsystemd "$sysroot" "$libsystemd_packages"

    cat >"$TOOLCHAIN_INI" <<EOF
[toolchain]
cc = $cc
cxx = $cxx
ld = $ld
EOF

    refresh_component_profiles

    echo "ARMv7 toolchain setup complete"
    echo "Toolchain config: $TOOLCHAIN_INI"
}

main "$@"
