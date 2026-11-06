#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

readonly APP_ROOT="/var/lib/nanokvm-runtime/kvmapp"
readonly SCRIPTS_ROOT="/kvmapp/scripts"
readonly LOG_DIR="/var/log/nanokvm-go"
readonly LOG_FILE="${LOG_DIR}/nanokvm-go.log"
readonly TARGETS=(
    "${APP_ROOT}/server/NanoKVM-Server"
)
readonly HTTPS_CRT=(
    "/etc/kvm/server.key"
    "/etc/kvm/server.crt"
)

declare -A CHILD_PIDS=()
declare -A FAIL_COUNTS=()
declare -A WAS_RESTARTED=()

MAX_FAILS=3
LOG_MAX_SIZE=$((10 * 1024 * 1024))

check_log_size() {
    if [ -f "$LOG_FILE" ]; then
        local log_size
        log_size=$(stat -c%s "$LOG_FILE")
        if [ "$log_size" -gt "$LOG_MAX_SIZE" ]; then
            echo "$(date) - Log file exceeded 10MB, rotating..." >>"$LOG_FILE"
            mv -f "$LOG_FILE" "${LOG_FILE}.1" 2>/dev/null || true
            : >"$LOG_FILE"
            echo "$(date) - Log file cleared" >>"$LOG_FILE"
        fi
    fi
}

graceful_stop() {
    echo "$(date) - Received stop signal, terminating child processes..." >>"$LOG_FILE"
    for pid in "${CHILD_PIDS[@]}"; do
        kill -SIGTERM -- "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    echo "$(date) - NanoKVM Go service has stopped" >>"$LOG_FILE"
    exit 0
}

makesure_https_crt() {
    local need_delete=0
    local crt_mod=""
    local key_mod=""

    if ! command -v openssl >/dev/null 2>&1; then
        echo "$(date) - openssl not found" >>"$LOG_FILE"
        exit 1
    fi

    echo "$(date) - Starting certificate verification..." >>"$LOG_FILE"

    for cert in "${HTTPS_CRT[@]}"; do
        if [ ! -f "$cert" ]; then
            echo "$(date) - Certificate file not found: $cert" >>"$LOG_FILE"
            need_delete=1
        fi
    done

    if [ $need_delete -eq 0 ]; then
        if ! openssl x509 -in /etc/kvm/server.crt -noout -text >/dev/null 2>&1; then
            echo "$(date) - Invalid certificate: /etc/kvm/server.crt" >>"$LOG_FILE"
            need_delete=1
        fi

        if ! openssl rsa -in /etc/kvm/server.key -check -noout >/dev/null 2>&1; then
            echo "$(date) - Invalid private key: /etc/kvm/server.key" >>"$LOG_FILE"
            need_delete=1
        fi

        if [ $need_delete -eq 0 ]; then
            crt_mod=$(openssl x509 -noout -modulus -in /etc/kvm/server.crt 2>/dev/null | openssl md5)
            key_mod=$(openssl rsa -noout -modulus -in /etc/kvm/server.key 2>/dev/null | openssl md5)
            if [ "$crt_mod" != "$key_mod" ]; then
                echo "$(date) - Certificate and private key do not match" >>"$LOG_FILE"
                need_delete=1
            fi
        fi
    fi

    if [ $need_delete -eq 1 ]; then
        echo "$(date) - Invalid or missing certificates, initiating cleanup..." >>"$LOG_FILE"

        for cert in "${HTTPS_CRT[@]}"; do
            if [ -e "$cert" ]; then
                rm -f "$cert" && echo "$(date) - Deleted: $cert" >>"$LOG_FILE"
            fi
        done

        echo "$(date) - Generating new SSL certificates..." >>"$LOG_FILE"
        if ! openssl req -x509 -newkey rsa:2048 \
            -keyout /etc/kvm/server.key \
            -out /etc/kvm/server.crt \
            -days 3650 -nodes -subj "/CN=localhost" >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to generate SSL certificates" >>"$LOG_FILE"
            exit 1
        fi
        sync
        echo "$(date) - SSL certificates generated successfully" >>"$LOG_FILE"
    else
        echo "$(date) - All certificates valid" >>"$LOG_FILE"
    fi
}

#==================== main ====================#
mkdir -p "$LOG_DIR"
echo "$(date) - Starting nanokvm-go service" >>"$LOG_FILE"

makesure_https_crt

trap graceful_stop SIGTERM SIGINT

while true; do
    check_log_size
    for target in "${TARGETS[@]}"; do
        exe_name="$(basename -- "$target")"
        pid="${CHILD_PIDS[$exe_name]:-}"

        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            FAIL_COUNTS["$exe_name"]=0
            WAS_RESTARTED["$exe_name"]="no"
            continue
        fi

        if ! pgrep -f "^${target}$" >/dev/null; then
            echo "$(date) - Restart: $target" >>"$LOG_FILE"
            "$target" >>"$LOG_DIR/${exe_name}.log" 2>&1 &
            pid=$!
            CHILD_PIDS["$exe_name"]=$pid
            echo "$(date) - Started $exe_name (PID:$pid)" >>"$LOG_FILE"

            if [ "${WAS_RESTARTED[$exe_name]:-no}" = "yes" ]; then
                ((FAIL_COUNTS["$exe_name"]++)) || true
            else
                FAIL_COUNTS["$exe_name"]=1
            fi
            WAS_RESTARTED["$exe_name"]="yes"

            if [ ${FAIL_COUNTS["$exe_name"]} -ge ${MAX_FAILS} ]; then
                echo "$(date) - $exe_name restarted ${FAIL_COUNTS["$exe_name"]} times, exiting" >>"$LOG_FILE"
                exit 1
            fi
        else
            FAIL_COUNTS["$exe_name"]=0
            WAS_RESTARTED["$exe_name"]="no"
        fi
    done
    sleep 2
done
