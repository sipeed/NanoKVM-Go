#!/bin/bash

set -eo pipefail

readonly LOG_DIR="/var/log/kvmcomm"
readonly LOG_FILE="${LOG_DIR}/kvm_reset_to_default.log"
readonly TMP_LOG_FILE="/tmp/kvm_reset.log"
readonly RESET_UNIT_NAME="nanokvm-factory-reset"
readonly RESET_WORKER_FLAG="--nanokvm-reset-worker"
readonly RESET_INDEPENDENT="${NANOKVM_RESET_INDEPENDENT:-0}"
readonly RESET_RUNTIME_SCRIPT="${NANOKVM_RESET_RUNTIME_SCRIPT:-}"
readonly LOCK_FILE="/run/lock/nanokvm-reset-to-default.lock"
readonly LOCK_FD=9
readonly DEVICE_MAINTENANCE_LOCK_FILE="/run/lock/nanokvm-device-maintenance.lock"
readonly DEVICE_MAINTENANCE_LOCK_FD=8
readonly APPLICATION_UPDATE_UNIT="nanokvm-self-update.service"
readonly RESET_STATE_DIR="/run/nanokvm"
readonly RESET_MARKER_FILE="${RESET_STATE_DIR}/reset_in_progress"
readonly LEGACY_RESET_MARKER_FILE="/dev/shm/reset_in_progress.lock"
readonly KVM_CONFIG_DIR="${KVM_CONFIG_DIR:-/etc/kvm}"
readonly TOTP_SECRET_FILE="${TOTP_SECRET_FILE:-${KVM_CONFIG_DIR}/totp.secret}"
readonly RECOVERY_CODES_FILE="${RECOVERY_CODES_FILE:-${KVM_CONFIG_DIR}/recovery_codes.json}"

RESET_LOCK_ACQUIRED=0
MAINTENANCE_LOCK_ACQUIRED=0
RESET_MARKER_OWNED=0
USBDEV_WAS_ACTIVE=0

run_in_independent_unit() {
    if [ "$RESET_INDEPENDENT" = "1" ]; then
        local current_script=""
        local runtime_script=""

        if [ -z "${INVOCATION_ID:-}" ]; then
            echo "Error: Factory reset worker must run in a systemd service"
            return 1
        fi
        if [ "${1:-}" != "$RESET_WORKER_FLAG" ]; then
            echo "Error: Invalid factory reset worker invocation"
            return 1
        fi
        if [ -z "$RESET_RUNTIME_SCRIPT" ] || \
            [[ "$RESET_RUNTIME_SCRIPT" != /run/nanokvm-reset-to-default.*.sh ]]; then
            echo "Error: Invalid factory reset runtime script"
            return 1
        fi
        if ! current_script=$(readlink -f -- "$0") || \
            ! runtime_script=$(readlink -f -- "$RESET_RUNTIME_SCRIPT") || \
            [ "$current_script" != "$runtime_script" ]; then
            echo "Error: Factory reset runtime script does not match the worker"
            return 1
        fi
        return 0
    fi
    if ! command -v systemd-run >/dev/null 2>&1; then
        echo "Error: systemd-run is required for an independent factory reset"
        return 1
    fi
    if ! command -v flock >/dev/null 2>&1; then
        echo "Error: flock is required for factory reset locking"
        return 1
    fi

    local script_path=""
    local runtime_script=""
    if ! script_path=$(readlink -f -- "$0"); then
        echo "Error: Failed to resolve factory reset script path: $0"
        return 1
    fi
    if [ -z "$script_path" ] || [ ! -f "$script_path" ]; then
        echo "Error: Failed to resolve factory reset script path: $0"
        return 1
    fi
    if ! runtime_script=$(mktemp "/run/nanokvm-reset-to-default.XXXXXX.sh"); then
        echo "Error: Failed to create runtime factory reset script"
        return 1
    fi
    if ! install -m 0700 -- "$script_path" "$runtime_script"; then
        rm -f -- "$runtime_script" || true
        echo "Error: Failed to prepare runtime factory reset script"
        return 1
    fi
    if ! /bin/bash -n "$runtime_script"; then
        rm -f -- "$runtime_script" || true
        echo "Error: Factory reset runtime script failed syntax validation"
        return 1
    fi

    if ! systemd-run \
        --system \
        --no-ask-password \
        --unit="$RESET_UNIT_NAME" \
        --collect \
        --service-type=exec \
        --working-directory=/ \
        --property=KillMode=control-group \
        --setenv=NANOKVM_RESET_INDEPENDENT=1 \
        --setenv="NANOKVM_RESET_RUNTIME_SCRIPT=$runtime_script" \
        --setenv="KVM_CONFIG_DIR=$KVM_CONFIG_DIR" \
        --setenv="TOTP_SECRET_FILE=$TOTP_SECRET_FILE" \
        --setenv="RECOVERY_CODES_FILE=$RECOVERY_CODES_FILE" \
        /bin/bash "$runtime_script" "$RESET_WORKER_FLAG"; then
        rm -f -- "$runtime_script" || true
        echo "Error: Failed to start independent factory reset unit"
        return 1
    fi

    echo "Factory reset started in ${RESET_UNIT_NAME}.service"
    exit 0
}

disable_2fa() {
    echo "$(date) - Disabling two-factor authentication" >>"$LOG_FILE"

    mkdir -p -- "$(dirname -- "$TOTP_SECRET_FILE")"
    : >"$TOTP_SECRET_FILE"
    chmod 600 -- "$TOTP_SECRET_FILE"

    rm -f -- "$RECOVERY_CODES_FILE"
    sync

    echo "$(date) - Two-factor authentication disabled successfully" >>"$LOG_FILE"
    echo "Two-factor authentication disabled"
}

logout_tailscale_before_isolation() {
    if ! command -v tailscale >/dev/null 2>&1 || \
        ! systemctl is-active --quiet tailscaled.service 2>/dev/null; then
        return 0
    fi

    echo "$(date) - Requesting Tailscale logout before clearing local identity" >>"$LOG_FILE"
    if ! command -v timeout >/dev/null 2>&1; then
        echo "$(date) - timeout command not found; skipping remote Tailscale logout" >>"$LOG_FILE"
        return 0
    fi

    if timeout 15 tailscale logout >>"$LOG_FILE" 2>&1; then
        echo "$(date) - Tailscale logout completed" >>"$LOG_FILE"
    else
        echo "$(date) - Tailscale logout failed or timed out; local identity will still be removed" >>"$LOG_FILE"
    fi
}

disable_tailscale_autostart() {
    local enabled_state=""

    # A runtime-masked unit is silently ignored by `systemctl disable`, so
    # remove any stale runtime mask before changing the persistent state.
    if ! systemctl unmask --runtime tailscaled.service >>"$LOG_FILE" 2>&1; then
        echo "$(date) - Failed to remove the Tailscale runtime mask" >>"$LOG_FILE"
        return 1
    fi

    enabled_state=$(systemctl is-enabled tailscaled.service 2>>"$LOG_FILE" || true)
    if [ "$enabled_state" = "not-found" ]; then
        return 0
    fi
    if [ -z "$enabled_state" ]; then
        echo "$(date) - Failed to inspect Tailscale service autostart" >>"$LOG_FILE"
        return 1
    fi

    echo "$(date) - Disabling Tailscale service autostart" >>"$LOG_FILE"

    if ! systemctl disable --now tailscaled.service >>"$LOG_FILE" 2>&1; then
        echo "$(date) - Failed to disable Tailscale service autostart" >>"$LOG_FILE"
        return 1
    fi

    enabled_state=$(systemctl is-enabled tailscaled.service 2>>"$LOG_FILE" || true)
    if [ "$enabled_state" != "disabled" ]; then
        echo "$(date) - Tailscale service autostart is not disabled: $enabled_state" >>"$LOG_FILE"
        return 1
    fi

    echo "$(date) - Tailscale service autostart disabled" >>"$LOG_FILE"
}

stop_and_disable_optional_services() {
    local failed=0
    local service=""

    for service in \
        nanokvm-frpc.service \
        nanokvm-cloudflared.service \
        nanokvm-zerotier.service \
        nanokvm-netbird.service \
        nano_ocr.service \
        memfab.service \
        ssh.socket \
        ssh.service; do
        if ! systemctl cat "$service" >/dev/null 2>&1; then
            continue
        fi

        echo "$(date) - Stopping and disabling $service" >>"$LOG_FILE"
        if ! systemctl disable --now "$service" >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to stop and disable $service" >>"$LOG_FILE"
            echo "Error: Failed to stop and disable $service"
            failed=1
        fi
    done

    return "$failed"
}

reset_wifi_private_data() {
    echo "$(date) - Clearing saved Wi-Fi configuration" >>"$LOG_FILE"

    if [ ! -x /opt/scripts/wifi.sh ]; then
        echo "$(date) - Wi-Fi management script is missing or not executable" >>"$LOG_FILE"
        return 1
    fi
    if ! /opt/scripts/wifi.sh clear >>"$LOG_FILE" 2>&1; then
        echo "$(date) - Failed to clear saved Wi-Fi configuration" >>"$LOG_FILE"
        return 1
    fi

    echo "$(date) - Saved Wi-Fi configuration cleared" >>"$LOG_FILE"
}

reset_tailscale_private_data() {
    local tailscale_installed=0
    local env_file="/etc/default/tailscaled"
    local env_file_tmp=""

    if [ -x /usr/sbin/tailscaled ] || [ -e "$env_file" ] || \
        systemctl cat tailscaled.service >/dev/null 2>&1; then
        tailscale_installed=1
    fi

    echo "$(date) - Clearing Tailscale node identity" >>"$LOG_FILE"

    if ! rm -rf -- \
        /var/lib/tailscale \
        /var/cache/tailscale \
        /run/tailscale \
        /root/.tailscale \
        /etc/systemd/system/tailscaled.service.d; then
        echo "$(date) - Failed to remove Tailscale private state" >>"$LOG_FILE"
        return 1
    fi

    if [ "$tailscale_installed" -eq 1 ]; then
        if ! env_file_tmp=$(mktemp "${env_file}.XXXXXX"); then
            echo "$(date) - Failed to create a temporary Tailscale environment file" >>"$LOG_FILE"
            return 1
        fi
        if ! chmod 0644 -- "$env_file_tmp" || \
            ! printf '%s\n' 'PORT=41641' 'FLAGS=""' >"$env_file_tmp" || \
            ! mv -f -- "$env_file_tmp" "$env_file"; then
            rm -f -- "$env_file_tmp" 2>/dev/null || true
            echo "$(date) - Failed to restore the default Tailscale environment" >>"$LOG_FILE"
            return 1
        fi
        if ! install -d -m 0700 -- /var/lib/tailscale; then
            echo "$(date) - Failed to recreate an empty Tailscale state directory" >>"$LOG_FILE"
            return 1
        fi
    fi

    echo "$(date) - Tailscale node identity cleared" >>"$LOG_FILE"
}

reset_ssh_private_data() {
    local home_dir=""

    echo "$(date) - Clearing SSH authorization, client keys, and host identity" >>"$LOG_FILE"

    if ! rm -rf -- /root/.ssh || \
        ! rm -f -- \
            /root/.bash_history \
            /root/.ash_history \
            /root/.zsh_history \
            /root/.lesshst; then
        echo "$(date) - Failed to remove root SSH data or shell history" >>"$LOG_FILE"
        return 1
    fi
    if [ -d /home ]; then
        for home_dir in /home/*; do
            [ -e "$home_dir" ] || continue
            if ! rm -rf -- "$home_dir/.ssh" || \
                ! rm -f -- \
                    "$home_dir/.bash_history" \
                    "$home_dir/.ash_history" \
                    "$home_dir/.zsh_history" \
                    "$home_dir/.lesshst"; then
                echo "$(date) - Failed to remove SSH data or shell history from $home_dir" >>"$LOG_FILE"
                return 1
            fi
        done
    fi

    if [ -d /etc/ssh ]; then
        if ! command -v ssh-keygen >/dev/null 2>&1; then
            echo "$(date) - ssh-keygen not found; refusing to remove SSH host keys" >>"$LOG_FILE"
            return 1
        fi
        if ! rm -f -- /etc/ssh/ssh_host_*; then
            echo "$(date) - Failed to remove existing SSH host keys" >>"$LOG_FILE"
            return 1
        fi
        if ! ssh-keygen -A >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to generate new SSH host keys" >>"$LOG_FILE"
            return 1
        fi
        if command -v sshd >/dev/null 2>&1 && ! sshd -t >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Warning: sshd configuration validation failed after host key rotation" >>"$LOG_FILE"
        fi
    fi

    echo "$(date) - SSH private data cleared and host identity regenerated" >>"$LOG_FILE"
}

request_default_hostname_restore() {
    echo "$(date) - Requesting default hostname restoration on next boot" >>"$LOG_FILE"
    if ! touch /boot/first_time_boot; then
        echo "$(date) - Failed to create /boot/first_time_boot" >>"$LOG_FILE"
        return 1
    fi
}

verify_usb_storage_detached() {
    local gadget_root="/sys/kernel/config/usb_gadget/g0"
    local state=""
    local lun_file=""

    if [ -e "${gadget_root}/UDC" ]; then
        if [ ! -r "${gadget_root}/UDC" ]; then
            echo "$(date) - USB gadget UDC state is not readable" >>"$LOG_FILE"
            return 1
        fi
        state=$(<"${gadget_root}/UDC")
        if [ -n "${state//[[:space:]]/}" ]; then
            echo "$(date) - USB gadget is still bound to UDC: $state" >>"$LOG_FILE"
            return 1
        fi
    fi

    for lun_file in \
        "${gadget_root}/functions/mass_storage.disk0/lun.0/file" \
        "${gadget_root}/functions/mass_storage.disk1/lun.0/file"; do
        [ -e "$lun_file" ] || continue
        if [ ! -r "$lun_file" ]; then
            echo "$(date) - USB mass-storage LUN state is not readable: $lun_file" >>"$LOG_FILE"
            return 1
        fi
        state=$(<"$lun_file")
        if [ -n "${state//[[:space:]]/}" ]; then
            echo "$(date) - USB mass-storage LUN is still attached: $lun_file -> $state" >>"$LOG_FILE"
            return 1
        fi
    done
}

validate_no_data_submounts() {
    local data_dir="$1"
    local mount_targets=""
    local target=""

    if ! mount_targets=$(findmnt -rn -o TARGET); then
        echo "$(date) - Failed to inspect mounts below $data_dir" >>"$LOG_FILE"
        return 1
    fi
    while IFS= read -r target; do
        if [[ "$target" == "$data_dir/"* ]]; then
            echo "$(date) - Refusing to clear $data_dir with nested mount: $target" >>"$LOG_FILE"
            return 1
        fi
    done <<<"$mount_targets"
}

is_data_mounted() {
    local data_dir="$1"
    local target=""

    target=$(findmnt -rn -o TARGET --mountpoint "$data_dir" 2>/dev/null || true)
    [ "$target" = "$data_dir" ]
}

validate_expected_data_mount() {
    local data_dir="$1"
    local image_file="$2"
    local source=""
    local filesystem=""
    local expected_image=""
    local backing_file=""
    local actual_image=""

    if [ ! -f "$image_file" ] || [ -L "$image_file" ]; then
        echo "$(date) - Expected data image is missing or unsafe: $image_file" >>"$LOG_FILE"
        return 1
    fi
    if ! expected_image=$(readlink -f -- "$image_file"); then
        echo "$(date) - Failed to resolve data image: $image_file" >>"$LOG_FILE"
        return 1
    fi

    filesystem=$(findmnt -rn -o FSTYPE --mountpoint "$data_dir" 2>/dev/null || true)
    if [ "$filesystem" != "exfat" ] && [ "$filesystem" != "fuseblk" ]; then
        echo "$(date) - Refusing to clear unexpected $data_dir filesystem: $filesystem" >>"$LOG_FILE"
        return 1
    fi

    source=$(findmnt -rn -o SOURCE --mountpoint "$data_dir" 2>/dev/null || true)
    if [ "$source" = "$image_file" ]; then
        return 0
    fi
    if [[ "$source" != /dev/loop* ]]; then
        echo "$(date) - Refusing to clear unexpected $data_dir source: $source" >>"$LOG_FILE"
        return 1
    fi
    if ! command -v losetup >/dev/null 2>&1; then
        echo "$(date) - losetup is required to verify $data_dir backing storage" >>"$LOG_FILE"
        return 1
    fi

    if ! backing_file=$(losetup --noheadings --output BACK-FILE "$source" 2>/dev/null | head -n 1); then
        echo "$(date) - Failed to inspect loop backing file for $source" >>"$LOG_FILE"
        return 1
    fi
    backing_file="${backing_file#"${backing_file%%[![:space:]]*}"}"
    backing_file="${backing_file%"${backing_file##*[![:space:]]}"}"
    backing_file="${backing_file% (deleted)}"
    if [ -z "$backing_file" ] || ! actual_image=$(readlink -f -- "$backing_file"); then
        echo "$(date) - Failed to resolve loop backing file for $source" >>"$LOG_FILE"
        return 1
    fi
    if [ "$actual_image" != "$expected_image" ]; then
        echo "$(date) - Refusing to clear unexpected loop backing file: $actual_image" >>"$LOG_FILE"
        return 1
    fi
}

clear_data_directory_contents() {
    local data_dir="$1"
    local remaining=""

    if [ ! -d "$data_dir" ] || [ -L "$data_dir" ]; then
        echo "$(date) - Refusing to clear unsafe data directory: $data_dir" >>"$LOG_FILE"
        return 1
    fi
    if ! find "$data_dir" -xdev -depth -mindepth 1 -delete; then
        echo "$(date) - Failed to clear data directory: $data_dir" >>"$LOG_FILE"
        return 1
    fi
    if ! remaining=$(find "$data_dir" -xdev -mindepth 1 -print -quit); then
        echo "$(date) - Failed to verify data directory cleanup: $data_dir" >>"$LOG_FILE"
        return 1
    fi
    if [ -n "$remaining" ]; then
        echo "$(date) - Data directory is not empty after cleanup: $remaining" >>"$LOG_FILE"
        return 1
    fi
}

reset_data_private_data() {
    local data_dir="/data"
    local image_file="/exfat.img"
    local was_mounted=0
    local usbdev_state=""

    echo "$(date) - Clearing user data from $data_dir" >>"$LOG_FILE"

    if ! command -v findmnt >/dev/null 2>&1; then
        echo "$(date) - findmnt is required to safely clear $data_dir" >>"$LOG_FILE"
        return 1
    fi
    if [ -L "$data_dir" ]; then
        echo "$(date) - Refusing to clear symlinked data directory: $data_dir" >>"$LOG_FILE"
        return 1
    fi
    if [ ! -d "$data_dir" ] && ! install -d -m 0755 -- "$data_dir"; then
        echo "$(date) - Failed to create data directory: $data_dir" >>"$LOG_FILE"
        return 1
    fi

    usbdev_state=$(systemctl is-active usbdev.service 2>/dev/null || true)
    case "$usbdev_state" in
        active) USBDEV_WAS_ACTIVE=1 ;;
        inactive|failed|unknown) ;;
        *)
            echo "$(date) - Refusing to change USB state with unexpected usbdev.service state: $usbdev_state" >>"$LOG_FILE"
            return 1
            ;;
    esac
    if [ -x /opt/scripts/usbdev.sh ]; then
        if ! /bin/bash /opt/scripts/usbdev.sh stop >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to stop USB device state for data cleanup" >>"$LOG_FILE"
            restore_usbdev_if_needed || true
            return 1
        fi
    fi
    if ! verify_usb_storage_detached; then
        return 1
    fi
    if ! validate_no_data_submounts "$data_dir"; then
        return 1
    fi

    if is_data_mounted "$data_dir"; then
        was_mounted=1
        if ! validate_expected_data_mount "$data_dir" "$image_file"; then
            return 1
        fi
    elif [ -e "$image_file" ]; then
        if [ ! -f "$image_file" ] || [ -L "$image_file" ]; then
            echo "$(date) - Refusing to mount unsafe data image: $image_file" >>"$LOG_FILE"
            return 1
        fi
        if ! systemctl start data.mount; then
            echo "$(date) - Failed to mount $image_file for cleanup" >>"$LOG_FILE"
            return 1
        fi
        if ! is_data_mounted "$data_dir" || \
            ! validate_no_data_submounts "$data_dir" || \
            ! validate_expected_data_mount "$data_dir" "$image_file"; then
            systemctl stop data.mount >/dev/null 2>&1 || true
            return 1
        fi
    fi

    if is_data_mounted "$data_dir"; then
        if [ -e "$image_file" ] && ! /kvmcomm/scripts/mount.py local >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to make $data_dir writable for cleanup" >>"$LOG_FILE"
            return 1
        fi
        if ! clear_data_directory_contents "$data_dir"; then
            return 1
        fi
        sync
    fi

    if ! systemctl stop data.mount; then
        echo "$(date) - Failed to unmount $data_dir after cleanup" >>"$LOG_FILE"
        return 1
    fi
    if is_data_mounted "$data_dir"; then
        echo "$(date) - $data_dir is still mounted after stopping data.mount" >>"$LOG_FILE"
        return 1
    fi
    if ! validate_no_data_submounts "$data_dir" || \
        ! clear_data_directory_contents "$data_dir" || \
        ! install -d -m 0755 -- "$data_dir"; then
        return 1
    fi

    if [ "$was_mounted" -eq 1 ]; then
        if ! systemctl start data.mount || \
            ! validate_expected_data_mount "$data_dir" "$image_file"; then
            echo "$(date) - Failed to restore the original $data_dir mount state" >>"$LOG_FILE"
            return 1
        fi
    fi
    if ! restore_usbdev_if_needed; then
        echo "$(date) - Failed to restore USB state after data cleanup" >>"$LOG_FILE"
        return 1
    fi

    echo "$(date) - User data cleared from $data_dir" >>"$LOG_FILE"
}

reset_private_data() {
    local failed=0

    if ! reset_data_private_data; then
        failed=1
    fi
    if ! reset_wifi_private_data; then
        failed=1
    fi
    if ! reset_tailscale_private_data; then
        failed=1
    fi
    if ! rm -rf -- /var/lib/zerotier-one; then
        echo "$(date) - Failed to clear ZeroTier node identity and networks" >>"$LOG_FILE"
        failed=1
    fi
    if ! rm -rf -- /var/lib/nanokvm-netbird; then
        echo "$(date) - Failed to clear NetBird node identity" >>"$LOG_FILE"
        failed=1
    fi
    if ! reset_ssh_private_data; then
        failed=1
    fi

    if [ "$failed" -ne 0 ]; then
        echo "$(date) - One or more private data cleanup steps failed" >>"$LOG_FILE"
        return 1
    fi
}

reset_kvm_configuration() {
    local config_dir="$KVM_CONFIG_DIR"
    local canonical_dir=""
    local parent_dir=""

    if [ -z "$config_dir" ] || [[ "$config_dir" != /* ]]; then
        echo "$(date) - Refusing to clear an invalid KVM configuration directory: $config_dir" >>"$LOG_FILE"
        echo "Error: Invalid KVM configuration directory"
        return 1
    fi
    if [ -L "$config_dir" ]; then
        echo "$(date) - Refusing to clear symlinked KVM configuration directory: $config_dir" >>"$LOG_FILE"
        echo "Error: Symlinked KVM configuration directory"
        return 1
    fi

    if [ -e "$config_dir" ]; then
        if [ ! -d "$config_dir" ] || ! canonical_dir=$(readlink -f -- "$config_dir"); then
            echo "$(date) - Refusing to clear unsafe KVM configuration directory: $config_dir" >>"$LOG_FILE"
            echo "Error: Unsafe KVM configuration directory"
            return 1
        fi
    else
        if ! parent_dir=$(readlink -f -- "$(dirname -- "$config_dir")"); then
            echo "$(date) - Failed to resolve KVM configuration directory parent: $config_dir" >>"$LOG_FILE"
            echo "Error: Unsafe KVM configuration directory"
            return 1
        fi
        canonical_dir="${parent_dir}/$(basename -- "$config_dir")"
    fi

    case "$canonical_dir" in
        /|/bin|/boot|/dev|/etc|/home|/lib|/lib64|/opt|/proc|/root|/run|/sbin|/sys|/tmp|/usr|/var|/kvmcomm|/kvmapp)
            echo "$(date) - Refusing to clear broad KVM configuration path: $canonical_dir" >>"$LOG_FILE"
            echo "Error: Unsafe KVM configuration directory"
            return 1
            ;;
    esac

    echo "$(date) - Clearing KVM configuration directory: $config_dir" >>"$LOG_FILE"
    if ! rm -rf -- "$config_dir" || ! install -d -m 0755 -- "$config_dir"; then
        echo "$(date) - Failed to clear KVM configuration directory: $config_dir" >>"$LOG_FILE"
        echo "Error: Failed to clear KVM configuration directory"
        return 1
    fi
    echo "$(date) - KVM configuration directory cleared" >>"$LOG_FILE"
}

clear_stale_runtime_masks() {
    local unit=""

    # Older reset scripts left runtime masks behind because they rebooted at
    # the end. Remove only those temporary masks so this in-place reset can
    # mount data and restart the two core services normally.
    for unit in \
        data.mount \
        kvmcomm.service \
        nanokvm-go.service \
        wifi.service \
        usbdev.service \
        nano_ocr.service \
        memfab.service \
        ssh.socket \
        ssh.service; do
        if ! systemctl unmask --runtime "$unit" >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to remove stale runtime mask: $unit" >>"$LOG_FILE"
        fi
    done
}

application_update_blocks_reset() {
    local properties=""
    local load_state=""
    local state=""

    if ! properties=$(systemctl show --property=LoadState --property=ActiveState \
        "$APPLICATION_UPDATE_UNIT" 2>/dev/null); then
        echo "$(date) - Failed to inspect application update unit; blocking factory reset" >>"$LOG_FILE"
        echo "Error: Failed to confirm that application update is idle"
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
            echo "$(date) - Application update unit is $state; blocking factory reset" >>"$LOG_FILE"
            echo "Error: Application update is in progress"
            return 0
            ;;
        inactive|failed)
            case "$load_state" in
                loaded|not-found) return 1 ;;
            esac
            ;;
    esac


    echo "$(date) - Application update unit returned unexpected state '$load_state/$state'; blocking factory reset" >>"$LOG_FILE"
    echo "Error: Failed to confirm that application update is idle"
    return 0
}

acquire_reset_lock() {
    if ! command -v flock >/dev/null 2>&1; then
        echo "$(date) - flock command not found" >>"$LOG_FILE"
        echo "Error: flock is required for factory reset locking"
        return 1
    fi

    mkdir -p -- "$(dirname -- "$LOCK_FILE")"
    exec 9>"$LOCK_FILE"

    local lock_result=0
    if flock -n "$LOCK_FD"; then
        lock_result=0
    else
        lock_result=$?
    fi

    if [ "$lock_result" -eq 1 ]; then
        echo "$(date) - Another factory reset is already running" >>"$LOG_FILE"
        echo "Error: Another factory reset is already running"
        return 1
    fi
    if [ "$lock_result" -ne 0 ]; then
        echo "$(date) - Failed to acquire factory reset lock: flock returned $lock_result" >>"$LOG_FILE"
        echo "Error: Failed to acquire factory reset lock"
        return 1
    fi

    RESET_LOCK_ACQUIRED=1

    # Lock order is always operation-specific first, then the shared device
    # maintenance lock. Recheck the application unit after acquiring the
    # shared lock to close its cross-process lock handoff window.
    if application_update_blocks_reset; then
        return 1
    fi

    if ! { exec 8>"$DEVICE_MAINTENANCE_LOCK_FILE"; } 2>/dev/null; then
        echo "$(date) - Failed to open device maintenance lock file" >>"$LOG_FILE"
        echo "Error: Failed to open device maintenance lock"
        return 1
    fi

    lock_result=0
    if flock -n "$DEVICE_MAINTENANCE_LOCK_FD"; then
        lock_result=0
    else
        lock_result=$?
    fi
    if [ "$lock_result" -eq 1 ]; then
        echo "$(date) - Another device maintenance operation is running" >>"$LOG_FILE"
        echo "Error: Another device maintenance operation is running"
        return 1
    fi
    if [ "$lock_result" -ne 0 ]; then
        echo "$(date) - Failed to acquire device maintenance lock: flock returned $lock_result" >>"$LOG_FILE"
        echo "Error: Failed to acquire device maintenance lock"
        return 1
    fi
    MAINTENANCE_LOCK_ACQUIRED=1

    if application_update_blocks_reset; then
        return 1
    fi

    RESET_MARKER_OWNED=1
    mkdir -p -- "$RESET_STATE_DIR"
    chmod 0755 -- "$RESET_STATE_DIR"
    rm -f -- "$RESET_MARKER_FILE"
    printf '%s\n' "$$" >"$RESET_MARKER_FILE"
    rm -f -- "$LEGACY_RESET_MARKER_FILE" 2>/dev/null || true
    echo "$(date) - Factory reset lock acquired by PID $$" >>"$LOG_FILE"
    echo "Reset lock acquired"
}

remove_reset_marker() {
    if [ "$RESET_MARKER_OWNED" -eq 1 ]; then
        rm -f -- "$RESET_MARKER_FILE" "$LEGACY_RESET_MARKER_FILE" || true
        RESET_MARKER_OWNED=0
    fi
}

release_reset_locks() {
    if [ "$MAINTENANCE_LOCK_ACQUIRED" -eq 1 ]; then
        flock -u "$DEVICE_MAINTENANCE_LOCK_FD" 2>/dev/null || true
        { exec 8>&-; } 2>/dev/null || true
        MAINTENANCE_LOCK_ACQUIRED=0
    fi
    release_factory_reset_lock
}

release_factory_reset_lock() {
    if [ "$RESET_LOCK_ACQUIRED" -eq 1 ]; then
        flock -u "$LOCK_FD" 2>/dev/null || true
        { exec 9>&-; } 2>/dev/null || true
        RESET_LOCK_ACQUIRED=0
    fi
}

restart_core_services() {
    local failed=0
    local service=""

    # Start the Go server first so kvmcomm's UI/VIN children see the reset
    # configuration as soon as they are brought back up.
    for service in nanokvm-go.service kvmcomm.service; do
        echo "$(date) - Restarting $service" >>"$LOG_FILE"
        if ! systemctl restart "$service" >>"$LOG_FILE" 2>&1; then
            echo "$(date) - Failed to restart $service" >>"$LOG_FILE"
            echo "Error: Failed to restart $service"
            failed=1
        fi
    done

    return "$failed"
}

remove_runtime_script() {
    local current_script=""
    local runtime_script=""

    if [ -n "$RESET_RUNTIME_SCRIPT" ] && \
        [[ "$RESET_RUNTIME_SCRIPT" == /run/nanokvm-reset-to-default.*.sh ]] && \
        current_script=$(readlink -f -- "$0") && \
        runtime_script=$(readlink -f -- "$RESET_RUNTIME_SCRIPT") && \
        [ "$current_script" = "$runtime_script" ]; then
        rm -f -- "$runtime_script" 2>/dev/null || true
    fi
}

cleanup_on_exit() {
    local exit_code=$?
    trap - EXIT INT TERM HUP

    if [ "$exit_code" -ne 0 ]; then
        echo "$(date) - Script exiting with error (exit code: $exit_code)" >>"$LOG_FILE" 2>/dev/null || true
    fi
    restore_usbdev_if_needed || true
    remove_reset_marker
    remove_runtime_script
    release_reset_locks
    exit "$exit_code"
}

handle_reset_signal() {
    local signal_name="$1"
    local exit_code="$2"

    echo "$(date) - Script interrupted by $signal_name" >>"$LOG_FILE" 2>/dev/null || true
    exit "$exit_code"
}

restore_usbdev_if_needed() {
    if [ "$USBDEV_WAS_ACTIVE" -ne 1 ]; then
        return 0
    fi

    if [ ! -x /opt/scripts/usbdev.sh ]; then
        echo "$(date) - USB device script is missing; cannot restore USB state" >>"$LOG_FILE"
        return 1
    fi
    if ! /bin/bash /opt/scripts/usbdev.sh start >>"$LOG_FILE" 2>&1; then
        echo "$(date) - Failed to restore USB device state" >>"$LOG_FILE"
        return 1
    fi

    USBDEV_WAS_ACTIVE=0
}

run_in_independent_unit "$@"

mkdir -p -- "$LOG_DIR"
exec 1>>"$TMP_LOG_FILE"
exec 2>&1

trap cleanup_on_exit EXIT
trap 'handle_reset_signal SIGINT 130' INT
trap 'handle_reset_signal SIGTERM 143' TERM
trap 'handle_reset_signal SIGHUP 129' HUP

echo "$(date) - Starting reset to default..." >>"$LOG_FILE"

acquire_reset_lock

# Package recovery is intentionally disabled: factory reset keeps the
# installed nanokvm-go and kvmcomm files and only resets runtime state.

clear_stale_runtime_masks

logout_tailscale_before_isolation

disable_tailscale_autostart

stop_and_disable_optional_services

echo "Restoring default configuration files..."

rm -f /boot/usb* /boot/rndis* /boot/ncm* || true
rm -f /boot/force* || true
request_default_hostname_restore
rm -f /boot/wifi.nodhcp || true

systemctl daemon-reload

echo "Resetting system user passwords..."

# Reset root password
echo "$(date) - Resetting root password" >>"$LOG_FILE"
echo "root:sipeed" | chpasswd
if [ $? -eq 0 ]; then
    echo "$(date) - Successfully reset root password" >>"$LOG_FILE"
    echo "Root password reset to: sipeed"
else
    echo "$(date) - Failed to reset root password" >>"$LOG_FILE"
    echo "Error: Failed to reset root password"
fi

reset_private_data

if [ -f "/usr/share/zoneinfo/Asia/Shanghai" ]; then
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
    echo "Asia/Shanghai" > /etc/timezone
    if [ $? -eq 0 ]; then
        echo "$(date) - Successfully reset timezone to Asia/Shanghai" >>"$LOG_FILE"
    else
        echo "$(date) - Failed to reset timezone" >>"$LOG_FILE"
    fi
else
    echo "$(date) - Timezone file not found: /usr/share/zoneinfo/Asia/Shanghai" >>"$LOG_FILE"
fi

reset_kvm_configuration

disable_2fa

if ! rm -rf -- /var/lib/memfab; then
    echo "$(date) - Failed to clear Memory Fabric data" >>"$LOG_FILE"
    echo "Error: Failed to clear Memory Fabric data"
    exit 1
fi

echo "$(date) - Reset to default completed" >>"$LOG_FILE"
sync

# kvmcomm.sh refuses to start while either reset marker or the reset lock is
# present. Remove the marker and release only that lock before restarting;
# keep the shared maintenance lock until both services have restarted.
remove_reset_marker
release_factory_reset_lock

if ! restart_core_services; then
    echo "$(date) - Reset completed, but one or more core services failed to restart" >>"$LOG_FILE"
    exit 1
fi

sync
echo "Reset to default process completed successfully"
exit 0
