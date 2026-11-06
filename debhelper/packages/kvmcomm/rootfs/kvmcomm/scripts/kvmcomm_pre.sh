#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SILENT="${SILENT:-1}"
if [[ "${SILENT}" == "1" ]]; then
    exec >/dev/null 2>&1
fi

readonly APP_ROOT="/kvmcomm"
readonly RUNTIME_ROOT="/var/lib/nanokvm-runtime"
readonly RUNTIME_APP_ROOT="${RUNTIME_ROOT}/kvmcomm"

safe_rm() {
    local target="$1"
    if [[ -e "${target}" || -L "${target}" ]]; then
        rm -rf -- "${target}"
    fi
}

stage_file() {
    local src="$1"
    local dest="$2"

    if [[ ! -f "${src}" ]]; then
        echo "source file not found: ${src}" >&2
        exit 1
    fi

    mkdir -p -- "$(dirname "${dest}")"

    if ! ln -- "${src}" "${dest}" 2>/dev/null; then
        cp -a -- "${src}" "${dest}"
    fi
}

main() {
    local tmp_dir="${RUNTIME_APP_ROOT}.tmp"

    safe_rm "${tmp_dir}"
    mkdir -p -- "${tmp_dir}/vin" "${tmp_dir}/ui"

    stage_file "${APP_ROOT}/vin/kvm_vin" "${tmp_dir}/vin/kvm_vin"
    stage_file "${APP_ROOT}/ui/kvm_ui" "${tmp_dir}/ui/kvm_ui"

    safe_rm "${RUNTIME_APP_ROOT}"
    mv -- "${tmp_dir}" "${RUNTIME_APP_ROOT}"
}

main "$@"
