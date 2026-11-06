#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SILENT="${SILENT:-1}"
if [[ "${SILENT}" == "1" ]]; then
    exec >/dev/null 2>&1
fi

readonly APP_ROOT="/kvmapp"
readonly RUNTIME_ROOT="/var/lib/nanokvm-runtime"
readonly RUNTIME_APP_ROOT="${RUNTIME_ROOT}/kvmapp"
readonly SRC_SERVER_ROOT="${APP_ROOT}/server"

safe_rm() {
    local target="$1"
    if [[ -e "${target}" || -L "${target}" ]]; then
        rm -rf -- "${target}"
    fi
}

stage_file() {
    local src="$1"
    local dest="$2"

    if [[ ! -e "${src}" ]]; then
        echo "source file not found: ${src}" >&2
        exit 1
    fi

    mkdir -p -- "$(dirname "${dest}")"

    if [[ -L "${src}" ]]; then
        ln -s -- "$(readlink -- "${src}")" "${dest}"
        return
    fi

    if ! ln -- "${src}" "${dest}" 2>/dev/null; then
        cp -a -- "${src}" "${dest}"
    fi
}

stage_libs() {
    local tmp_dir="$1"
    local libs=("${SRC_SERVER_ROOT}"/dl_lib/libkvm.so*)

    if [[ ! -e "${libs[0]}" ]]; then
        echo "libkvm files not found: ${SRC_SERVER_ROOT}/dl_lib/libkvm.so*" >&2
        exit 1
    fi

    for lib in "${libs[@]}"; do
        stage_file "${lib}" "${tmp_dir}/server/dl_lib/$(basename -- "${lib}")"
    done
}

main() {
    local tmp_dir="${RUNTIME_APP_ROOT}.tmp"

    safe_rm "${tmp_dir}"
    mkdir -p -- "${tmp_dir}/server/dl_lib"

    stage_file "${SRC_SERVER_ROOT}/NanoKVM-Server" "${tmp_dir}/server/NanoKVM-Server"
    stage_libs "${tmp_dir}"

    if [[ ! -d "${SRC_SERVER_ROOT}/web" ]]; then
        echo "web dir not found: ${SRC_SERVER_ROOT}/web" >&2
        exit 1
    fi
    ln -s -- "${SRC_SERVER_ROOT}/web" "${tmp_dir}/server/web"

    safe_rm "${RUNTIME_APP_ROOT}"
    mv -- "${tmp_dir}" "${RUNTIME_APP_ROOT}"
}

main "$@"
