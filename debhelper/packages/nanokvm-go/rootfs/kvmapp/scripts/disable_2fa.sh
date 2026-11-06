#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

readonly KVM_CONFIG_DIR="${KVM_CONFIG_DIR:-/etc/kvm}"
readonly TOTP_SECRET_FILE="${TOTP_SECRET_FILE:-${KVM_CONFIG_DIR}/totp.secret}"
readonly RECOVERY_CODES_FILE="${RECOVERY_CODES_FILE:-${KVM_CONFIG_DIR}/recovery_codes.json}"
readonly RESTART_NANOKVM="${RESTART_NANOKVM:-1}"
readonly NANOKVM_SERVICES="${NANOKVM_SERVICES:-nanokvm-go.service nanokvm.service}"

disable_2fa() {
    mkdir -p -- "$(dirname -- "${TOTP_SECRET_FILE}")"
    : >"${TOTP_SECRET_FILE}"
    chmod 600 -- "${TOTP_SECRET_FILE}"

    rm -f -- "${RECOVERY_CODES_FILE}"
    sync
}

restart_nanokvm_if_active() {
    if [[ "${RESTART_NANOKVM}" != "1" ]]; then
        return 0
    fi
    if ! command -v systemctl >/dev/null 2>&1; then
        return 0
    fi

    local service
    local services=()
    IFS=' ' read -r -a services <<<"${NANOKVM_SERVICES}"

    for service in "${services[@]}"; do
        if systemctl is-active --quiet "${service}"; then
            systemctl restart "${service}"
            return 0
        fi
    done
}

main() {
    disable_2fa
    restart_nanokvm_if_active
    echo "2FA disabled"
}

main "$@"
