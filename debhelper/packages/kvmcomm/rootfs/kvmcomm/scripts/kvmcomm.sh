#!/usr/bin/env bash

readonly INSTALL_ROOT="/kvmcomm"
readonly APP_ROOT="/var/lib/nanokvm-runtime/kvmcomm"
readonly SCRIPTS_ROOT="${INSTALL_ROOT}/scripts"
readonly LOG_DIR="/var/log/kvmcomm"
readonly LOG_FILE="${LOG_DIR}/kvmcommd.log"
readonly SERVER_DEF="go"
readonly RESET_LOCK_FILE="/run/lock/nanokvm-reset-to-default.lock"
readonly RESET_LOCK_FD=199
readonly RESET_MARKER_FILE="/run/nanokvm/reset_in_progress"
readonly LEGACY_RESET_MARKER_FILE="/dev/shm/reset_in_progress.lock"
readonly FIRMWARE_UPDATE_LOCK_FILE="/run/lock/nanokvm-firmware-update.lock"
readonly FIRMWARE_UPDATE_LOCK_FD=198
readonly FIRMWARE_UPDATE_UNIT="nanokvm-firmware-update"
readonly DEVICE_MAINTENANCE_LOCK_FILE="/run/lock/nanokvm-device-maintenance.lock"
readonly DEVICE_MAINTENANCE_LOCK_FD=197
readonly APPLICATION_UPDATE_UNIT="nanokvm-self-update.service"
readonly FIRMWARE_MAINTENANCE_WAIT_SECONDS=300
readonly FIRMWARE_MAINTENANCE_POLL_SECONDS=2
readonly TARGETS=(
    "${APP_ROOT}/vin/kvm_vin"
    "${APP_ROOT}/ui/kvm_ui"
)
readonly REQUIRED_FIRMWARE_VERSION="v1.0.10"

declare -A CHILD_PIDS=()
declare -A FAIL_COUNTS=()
declare -A WAS_RESTARTED=()

factory_reset_in_progress() {
    if ! command -v flock >/dev/null 2>&1; then
        [ -e "$RESET_MARKER_FILE" ] || [ -e "$LEGACY_RESET_MARKER_FILE" ]
        return
    fi

    mkdir -p -- "$(dirname -- "$RESET_LOCK_FILE")" 2>/dev/null || {
        [ -e "$RESET_MARKER_FILE" ] || [ -e "$LEGACY_RESET_MARKER_FILE" ]
        return
    }
    if ! { exec 199>"$RESET_LOCK_FILE"; } 2>/dev/null; then
        [ -e "$RESET_MARKER_FILE" ] || [ -e "$LEGACY_RESET_MARKER_FILE" ]
        return
    fi

    local lock_result=0
    if flock -n "$RESET_LOCK_FD"; then
        rm -f -- "$RESET_MARKER_FILE" 2>/dev/null || true
        rm -f -- "$LEGACY_RESET_MARKER_FILE" 2>/dev/null || true
        flock -u "$RESET_LOCK_FD" 2>/dev/null || true
        { exec 199>&-; } 2>/dev/null || true
        return 1
    else
        lock_result=$?
    fi

    { exec 199>&-; } 2>/dev/null || true
    if [ "$lock_result" -eq 1 ]; then
        return 0
    fi

    echo "$(date) - Failed to inspect factory reset lock (flock returned $lock_result); skipping startup" >>"$LOG_FILE"
    return 0
}

application_update_blocks_maintenance() {
    local properties=""
    local load_state=""
    local state=""

    if ! properties=$(systemctl show --property=LoadState --property=ActiveState \
        "$APPLICATION_UPDATE_UNIT" 2>/dev/null); then
        echo "$(date) - Failed to inspect application update unit; blocking firmware update." >>"$LOG_FILE"
        return 0
    fi

    while IFS='=' read -r property value; do
        case "$property" in
            LoadState) load_state="$value" ;;
            ActiveState) state="$value" ;;
        esac
    done <<<"$properties"

    case "$state" in
        active|activating|deactivating|reloading|maintenance|refreshing)
            echo "$(date) - Application update unit is $state; blocking firmware update." >>"$LOG_FILE"
            return 0
            ;;
        inactive|failed)
            case "$load_state" in
                loaded|not-found) return 1 ;;
            esac
            ;;
    esac

    echo "$(date) - Application update unit returned unexpected state '$load_state/$state'; blocking firmware update." >>"$LOG_FILE"
    return 0
}

release_firmware_update_lock() {
    flock -u "$FIRMWARE_UPDATE_LOCK_FD" 2>/dev/null || true
    { exec 198>&-; } 2>/dev/null || true
}

release_device_maintenance_lock() {
    flock -u "$DEVICE_MAINTENANCE_LOCK_FD" 2>/dev/null || true
    { exec 197>&-; } 2>/dev/null || true
}

wait_for_firmware_maintenance_retry() {
    local deadline="$1"
    local remaining=$((deadline - SECONDS))
    local delay="$FIRMWARE_MAINTENANCE_POLL_SECONDS"

    if [ "$remaining" -le 0 ]; then
        return 1
    fi
    if [ "$delay" -gt "$remaining" ]; then
        delay="$remaining"
    fi
    sleep "$delay"
}

graceful_stop() {
    echo "$(date) - Received stop signal, terminating child processes..." >>"$LOG_FILE"
    # Terminate all daemons
    for pid in "${CHILD_PIDS[@]}"; do
        kill -SIGTERM "$pid" 2>/dev/null && wait "$pid" 2>/dev/null
    done
    echo "Service has stopped" >>"$LOG_FILE"
    exit 0
}

stop_stale_process() {
    local target="$1"
    local pid

    while read -r pid; do
        if [ -n "$pid" ]; then
            echo "$(date) - Stopping stale process: $target (PID:$pid)" >>"$LOG_FILE"
            kill -SIGTERM "$pid" 2>/dev/null || true
        fi
    done < <(pgrep -f "^${target}$" 2>/dev/null || true)
}

load_ko() {
    local ko_path="$1"
    local ko_name

    if [ ! -f "$ko_path" ]; then
        echo "$(date) - Error: Kernel module file '$ko_path' not found." >>"$LOG_FILE"
        return 1
    fi

    ko_name=$(basename "$ko_path" .ko)

    if lsmod | grep -q "^$ko_name"; then
        echo "$(date) - Module '$ko_name' already loaded." >>"$LOG_FILE"
        return 0
    fi

    if insmod "$ko_path"; then
        echo "$(date) - Module '$ko_name' loaded successfully." >>"$LOG_FILE"
        return 0
    else
        echo "$(date) - Error: Failed to load module '$ko_name'." >>"$LOG_FILE"
        return 1
    fi
}

unload_ko() {
    local ko_path="$1"
    local ko_name

    ko_name=$(basename "$ko_path" .ko)

    if ! lsmod | grep -q "^$ko_name"; then
        echo "$(date) - Module '$ko_name' is not loaded, skip unload." >>"$LOG_FILE"
        return 0
    fi

    if rmmod "$ko_name"; then
        echo "$(date) - Module '$ko_name' unloaded successfully." >>"$LOG_FILE"
        return 0
    else
        echo "$(date) - Error: Failed to unload module '$ko_name'." >>"$LOG_FILE"
        return 1
    fi
}

version_lt() {
    [ "$1" = "$2" ] && return 1
    local IFS=.
    local i ver1=($1) ver2=($2)
    for ((i = ${#ver1[@]}; i < ${#ver2[@]}; i++)); do
        ver1[i]=0
    done
    for ((i = 0; i < ${#ver2[@]}; i++)); do
        if ((10#${ver1[i]} < 10#${ver2[i]})); then
            return 0
        elif ((10#${ver1[i]} > 10#${ver2[i]})); then
            return 1
        fi
    done
    return 1
}

check_firmware_update() {
    local current_version="unknown"

    if [ -f /boot/ver ]; then
        current_version=$(</boot/ver)

        if echo "$current_version" | grep -qi 'custom'; then
            current_version="custom"
        fi
    fi
    echo "$(date) - Current firmware version: $current_version" >>"$LOG_FILE"

    if [ "$current_version" = "custom" ]; then
        echo "$(date) - Custom firmware detected, skipping update check." >>"$LOG_FILE"
        return
    fi

    if [[ "$current_version" =~ v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
        current_version="${BASH_REMATCH[0]}"
        cur="${current_version#v}"
        req="${REQUIRED_FIRMWARE_VERSION#v}"
    else
        echo "$(date) - Firmware version is invalid or missing ('$current_version'), forcing update." >>"$LOG_FILE"
        cur="0.0.0"
        req="${REQUIRED_FIRMWARE_VERSION#v}"
    fi

    if version_lt "$cur" "$req"; then
        echo "$(date) - Updating firmware to version $REQUIRED_FIRMWARE_VERSION" >>"$LOG_FILE"

        # The firmware updater owns cache discovery and re-verifies the signed
        # metadata/archive pair on every use. Never pass it a bare archive.
        if ! "$SCRIPTS_ROOT/firmware_update.sh" update "$REQUIRED_FIRMWARE_VERSION" >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Firmware update failed for version $REQUIRED_FIRMWARE_VERSION" >>"$LOG_FILE"
            return 1
        fi
    else
        echo "$(date) - Firmware is up-to-date." >>"$LOG_FILE"
    fi

    if [ -f "/var/run/reboot-required" ]; then
        echo "$(date) - Firmware update reboot required." >>"$LOG_FILE"
        sync
        echo "Rebooting in 3 seconds..." >>"$LOG_FILE"
        sleep 3
        systemctl reboot
    fi
}

run_firmware_update_worker() {
    mkdir -p "$LOG_DIR"

    if ! command -v flock >/dev/null 2>&1; then
        echo "$(date) - Failed to start firmware update: flock is unavailable." >>"$LOG_FILE"
        return 1
    fi

    mkdir -p -- "$(dirname -- "$FIRMWARE_UPDATE_LOCK_FILE")" 2>/dev/null || {
        echo "$(date) - Failed to create firmware update lock directory." >>"$LOG_FILE"
        return 1
    }
    if ! { exec 198>"$FIRMWARE_UPDATE_LOCK_FILE"; } 2>/dev/null; then
        echo "$(date) - Failed to open firmware update lock file." >>"$LOG_FILE"
        return 1
    fi

    local lock_result=0
    flock -n "$FIRMWARE_UPDATE_LOCK_FD" || lock_result=$?
    if [ "$lock_result" -eq 1 ]; then
        echo "$(date) - Firmware update is already running; skipping duplicate worker." >>"$LOG_FILE"
        { exec 198>&-; } 2>/dev/null || true
        return 0
    elif [ "$lock_result" -ne 0 ]; then
        echo "$(date) - Failed to acquire firmware update lock (flock returned $lock_result)." >>"$LOG_FILE"
        { exec 198>&-; } 2>/dev/null || true
        return 1
    fi

    local wait_deadline=$((SECONDS + FIRMWARE_MAINTENANCE_WAIT_SECONDS))
    local waiting=0
    while true; do
        if factory_reset_in_progress; then
            echo "$(date) - Factory reset is in progress; skipping firmware update." >>"$LOG_FILE"
            release_firmware_update_lock
            return 0
        fi

        # Lock order is always operation-specific first, then the shared device
        # maintenance lock. Keep only the firmware lock while waiting so the App
        # update can finish and release the shared lock.
        if application_update_blocks_maintenance; then
            if [ "$waiting" -eq 0 ]; then
                echo "$(date) - Waiting up to $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds for the firmware update maintenance window." >>"$LOG_FILE"
                waiting=1
            fi
            if ! wait_for_firmware_maintenance_retry "$wait_deadline"; then
                echo "$(date) - Timed out after $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds waiting for the firmware update maintenance window; skipping firmware update." >>"$LOG_FILE"
                release_firmware_update_lock
                return 1
            fi
            continue
        fi

        if ! { exec 197>"$DEVICE_MAINTENANCE_LOCK_FILE"; } 2>/dev/null; then
            echo "$(date) - Failed to open device maintenance lock file." >>"$LOG_FILE"
            release_firmware_update_lock
            return 1
        fi

        lock_result=0
        flock -n "$DEVICE_MAINTENANCE_LOCK_FD" || lock_result=$?
        if [ "$lock_result" -eq 1 ]; then
            { exec 197>&-; } 2>/dev/null || true
            if [ "$waiting" -eq 0 ]; then
                echo "$(date) - Another device maintenance operation is active; waiting up to $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds." >>"$LOG_FILE"
                waiting=1
            fi
            if ! wait_for_firmware_maintenance_retry "$wait_deadline"; then
                echo "$(date) - Timed out after $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds waiting for the firmware update maintenance window; skipping firmware update." >>"$LOG_FILE"
                release_firmware_update_lock
                return 1
            fi
            continue
        elif [ "$lock_result" -ne 0 ]; then
            echo "$(date) - Failed to acquire device maintenance lock (flock returned $lock_result)." >>"$LOG_FILE"
            { exec 197>&-; } 2>/dev/null || true
            release_firmware_update_lock
            return 1
        fi

        # Recheck after acquiring the shared lock so a self-update unit entering
        # activating during the lock handoff cannot race firmware flashing.
        if application_update_blocks_maintenance; then
            release_device_maintenance_lock
            if [ "$waiting" -eq 0 ]; then
                echo "$(date) - Waiting up to $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds for the firmware update maintenance window." >>"$LOG_FILE"
                waiting=1
            fi
            if ! wait_for_firmware_maintenance_retry "$wait_deadline"; then
                echo "$(date) - Timed out after $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds waiting for the firmware update maintenance window; skipping firmware update." >>"$LOG_FILE"
                release_firmware_update_lock
                return 1
            fi
            continue
        fi

        # Factory reset may have started after the check at the top of the
        # loop. Once the shared lock is held, give it one final chance to win
        # before any firmware work begins.
        if factory_reset_in_progress; then
            echo "$(date) - Factory reset started while waiting; skipping firmware update." >>"$LOG_FILE"
            release_device_maintenance_lock
            release_firmware_update_lock
            return 0
        fi

        # Do not begin firmware work after the maintenance wait budget has
        # expired, even if the App unit and shared lock became available while
        # the final probe was running.
        if [ "$SECONDS" -ge "$wait_deadline" ]; then
            echo "$(date) - Timed out after $FIRMWARE_MAINTENANCE_WAIT_SECONDS seconds waiting for the firmware update maintenance window; skipping firmware update." >>"$LOG_FILE"
            release_device_maintenance_lock
            release_firmware_update_lock
            return 1
        fi
        break
    done

    local update_result=0
    check_firmware_update || update_result=$?
    release_device_maintenance_lock
    release_firmware_update_lock
    return "$update_result"
}

start_firmware_update_async() {
    local unit="${FIRMWARE_UPDATE_UNIT}.service"
    local state
    local output

    state=$(systemctl show --property=ActiveState --value "$unit" 2>/dev/null || true)
    case "$state" in
        active|activating|deactivating)
            echo "$(date) - Firmware update unit is already $state; skipping duplicate start." >>"$LOG_FILE"
            return 0
            ;;
        failed)
            systemctl reset-failed "$unit" >/dev/null 2>&1 || true
            ;;
    esac

    if output=$(systemd-run \
        --unit="$FIRMWARE_UPDATE_UNIT" \
        --description="NanoKVM firmware update" \
        --collect \
        --no-block \
        --service-type=exec \
        "$SCRIPTS_ROOT/kvmcomm.sh" --firmware-update 2>&1); then
        echo "$(date) - Started asynchronous firmware update unit: ${output:-$unit}" >>"$LOG_FILE"
        return 0
    fi

    echo "$(date) - Failed to start asynchronous firmware update unit: ${output:-unknown error}" >>"$LOG_FILE"
    return 1
}

if [ "${1:-}" = "--firmware-update" ]; then
    run_firmware_update_worker
    exit $?
fi

#==================== main ====================#
mkdir -p "$LOG_DIR"
echo "$(date) - Starting kvmcomm service (PID:$$)" >>"$LOG_FILE"

if factory_reset_in_progress; then
    echo "$(date) - Factory reset is in progress; skipping kvmcomm startup" >>"$LOG_FILE"
    exit 0
fi

# ensure /bin/sh points to bash
shell_target=$(readlink /bin/sh)
if [ "$shell_target" != "bash" ] && [ "$shell_target" != "/bin/bash" ]; then
    echo "Updating /bin/sh to point to bash..." >>"$LOG_FILE"
    ln -sf bash /bin/sh
    sleep 3
    systemctl reboot
fi

start_firmware_update_async

if [ -e "/sys/class/usb_role/8000000.dwc3-role-switch/role" ]; then
    echo "device" >/sys/class/usb_role/8000000.dwc3-role-switch/role
fi

load_ko "${INSTALL_ROOT}/ko/lt7911_manage.ko"
load_ko "${INSTALL_ROOT}/ko/fb_st7789p3.ko"

stop_stale_process "${INSTALL_ROOT}/vin/kvm_vin"
stop_stale_process "${INSTALL_ROOT}/ui/kvm_ui"

trap graceful_stop SIGTERM SIGINT

while true; do
    for target in "${TARGETS[@]}"; do
        exe_name="$(basename -- "$target")"
        if ! pgrep -f "^${target}$" >/dev/null; then
            echo "$(date) - Restart: $target" >>"$LOG_FILE"
            "$target" >>"$LOG_DIR/${exe_name}.log" 2>&1 &

            pid=$!
            CHILD_PIDS["$exe_name"]=$pid
            echo "$(date) - Started $exe_name (PID:$pid, parent PID:$$)" >>"$LOG_FILE"

            if [ "${WAS_RESTARTED[$exe_name]:-no}" = "yes" ]; then
                ((FAIL_COUNTS["$exe_name"]++)) || true
            else
                FAIL_COUNTS["$exe_name"]=1
            fi
            WAS_RESTARTED["$exe_name"]="yes"

            if [ ${FAIL_COUNTS["$exe_name"]} -ge 3 ]; then
                exit 1
            fi
        else
            FAIL_COUNTS["$exe_name"]=0
            WAS_RESTARTED["$exe_name"]="no"
        fi
    done
    sleep 2
done
