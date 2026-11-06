#!/bin/bash

# -----------------------------------------------------------------------------
# Firmware directory structure for firmware update script:
#
# firmware/
# ├─ u-boot_signed.bin                                      # U-Boot binary
# ├─ boot_signed.bin                                        # Bootloader binary
# └─ AX620Q_emmc_arm32_k419_sipeed_nanoagent_signed.dtb
#
# overlay/
# ├─ boot/ver                                               # Version file
# └─ boot/configs                                           # Configuration file
#
# Usage:
# ./firmware_update.sh gen_b2sum        # Generate b2sum.txt
# ./firmware_update.sh fetch <version>  # Fetch firmware package from CDN
# ./firmware_update.sh update <version> # Verify/cache/flash firmware
# ./firmware_update.sh update --allow-same-version <version> # Explicit manual CDN reinstall
# ./firmware_update.sh update <archive> <metadata> # Explicit manual local install (same version allowed)
# -----------------------------------------------------------------------------

set -eo pipefail
umask 077

UBOOT_FILE=firmware/u-boot_signed.bin
DTB_FILE=firmware/AX620Q_emmc_arm32_k419_sipeed_nanoagent_signed.dtb
KERNEL_FILE=firmware/boot_signed.bin

CDN_URLS=(
    "https://cdn.sipeed.com/nanokvm-go/v2/firmware"
    "https://cdn.sipeed.com/nanokvm-go/v2/preview/firmware"
)

CACHE_BASE_DIR="${FIRMWARE_CACHE_DIR:-/root/.kvmcache}"
NETWORK_WAIT_SECONDS="${FIRMWARE_NETWORK_WAIT_SECONDS:-180}"
NETWORK_POLL_SECONDS="${FIRMWARE_NETWORK_POLL_SECONDS:-2}"
UPDATE_VERIFIER="${NANOKVM_UPDATE_VERIFIER:-/usr/bin/nanokvm-update-verify}"
UPDATE_KEYRING="${NANOKVM_UPDATE_KEYRING:-/usr/share/nanokvm/update-keys.json}"
CURRENT_VERSION_FILE="${FIRMWARE_CURRENT_VERSION_FILE:-/boot/ver}"
FIRMWARE_METADATA_MAX_BYTES=$((64 * 1024))
FIRMWARE_DISK_SAFETY_MARGIN=$((64 * 1024 * 1024))
ACTIVE_WORK_DIR=""
FIRMWARE_EXTRACT_DIR=""
FIRMWARE_UPDATE_UBOOT="false"
VERIFIED_FIRMWARE_UPDATE_UBOOT="false"

cleanup_active_work_dir() {
    if [ -n "$ACTIVE_WORK_DIR" ] && \
        [[ "$ACTIVE_WORK_DIR" == "${CACHE_BASE_DIR}/.firmware-update."* ]]; then
        rm -rf -- "$ACTIVE_WORK_DIR"
    fi
    ACTIVE_WORK_DIR=""
}

trap cleanup_active_work_dir EXIT

create_work_dir() {
    cleanup_active_work_dir
    if ! install -d -m 0700 -- "$CACHE_BASE_DIR"; then
        echo "Error: Failed to create firmware cache directory"
        return 1
    fi
    if ! ACTIVE_WORK_DIR=$(mktemp -d "${CACHE_BASE_DIR}/.firmware-update.XXXXXX"); then
        echo "Error: Failed to create firmware staging directory"
        return 1
    fi
    chmod 0700 -- "$ACTIVE_WORK_DIR"
}

normalize_firmware_version() {
    local version="${1:-}"
    local component=""
    version="${version#v}"
    if ! [[ "$version" =~ ^(0|[1-9][0-9]{0,19})\.(0|[1-9][0-9]{0,19})\.(0|[1-9][0-9]{0,19})$ ]]; then
        echo "Error: Invalid firmware version: ${1:-missing}" >&2
        return 1
    fi
    # Match the verifier's uint64 component boundary. A syntactically valid
    # but overflowing /boot/ver value is not allowed to poison verification of
    # an otherwise valid explicitly requested signed Firmware release.
    IFS=. read -r -a version_components <<<"$version"
    for component in "${version_components[@]}"; do
        if [ "${#component}" -eq 20 ] && [[ "$component" > "18446744073709551615" ]]; then
            echo "Error: Invalid firmware version: ${1:-missing}" >&2
            return 1
        fi
    done
    printf '%s\n' "$version"
}

wait_for_network() {
    local probe_path="$1"
    local start_time=$SECONDS
    local elapsed=0
    local base probe_url

    echo "Waiting up to ${NETWORK_WAIT_SECONDS}s for the firmware download endpoint..."

    while ((elapsed < NETWORK_WAIT_SECONDS)); do
        for base in "${CDN_URLS[@]}"; do
            probe_url="${base}/${probe_path}"
            if curl -sSL --connect-timeout 3 --max-time 5 \
                -o /dev/null "$probe_url" 2>/dev/null; then
                echo "Firmware download endpoint is reachable after ${elapsed}s: $probe_url"
                return 0
            fi
        done

        sleep "$NETWORK_POLL_SECONDS"
        elapsed=$((SECONDS - start_time))
    done

    echo "Firmware download endpoint did not become reachable within ${NETWORK_WAIT_SECONDS}s"
    return 1
}

read_current_firmware_version() {
    local raw_version=""

    if [ ! -f "$CURRENT_VERSION_FILE" ]; then
        return 1
    fi
    raw_version=$(head -n1 "$CURRENT_VERSION_FILE")
    if [[ "$raw_version" =~ [Cc][Uu][Ss][Tt][Oo][Mm] ]]; then
        echo "Warning: custom firmware version is not comparable; verifying only the explicitly requested signed version" >&2
        return 1
    fi
    # Keep compatibility with kvmcomm.sh, which extracts an embedded vX.Y.Z
    # token from decorated /boot/ver text instead of treating the whole line
    # as a semantic version.
    if [[ "$raw_version" =~ [vV]([0-9]+\.[0-9]+\.[0-9]+) ]]; then
        local candidate_version="${BASH_REMATCH[1]}"
        if normalize_firmware_version "$candidate_version" >/dev/null 2>&1; then
            printf '%s\n' "$candidate_version"
            return 0
        fi
    fi

    echo "Warning: current firmware version is not comparable; verifying only the explicitly requested signed version" >&2
    return 1
}

verify_firmware_pair() {
    local metadata_path="$1"
    local artifact_path="$2"
    local requested_version="$3"
    local allow_same_version="$4"
    local current_version=""
    local verifier_args=()
    local output=""
    local expected_output_false=""
    local expected_output_true=""

    # Never let a previous successful verification influence this attempt.
    VERIFIED_FIRMWARE_UPDATE_UBOOT="false"

    if [ "$allow_same_version" != "true" ] && [ "$allow_same_version" != "false" ]; then
        echo "Error: Invalid same-version Firmware policy"
        return 1
    fi

    if [ ! -f "$metadata_path" ] || [ -L "$metadata_path" ] || \
        [ ! -f "$artifact_path" ] || [ -L "$artifact_path" ]; then
        echo "Error: Firmware metadata and archive must be regular files"
        return 1
    fi
    if [ "$(basename -- "$metadata_path")" != "firmware_${requested_version}.json" ] || \
        [ "$(basename -- "$artifact_path")" != "axera_firmware_v${requested_version}.tar.gz" ]; then
        echo "Error: Firmware pair filenames do not match requested version"
        return 1
    fi

    current_version=$(read_current_firmware_version || true)
    verifier_args=(
        firmware
        --metadata "$metadata_path"
        --artifact "$artifact_path"
        --keyring "$UPDATE_KEYRING"
        --requested-version "$requested_version"
    )
    if [ -n "$current_version" ]; then
        verifier_args+=(--current-version "$current_version")
        if [ "$allow_same_version" = "true" ]; then
            verifier_args+=(--allow-same-version)
        fi
    fi

    if ! output=$("$UPDATE_VERIFIER" "${verifier_args[@]}"); then
        echo "Error: Firmware archive signature verification failed"
        return 1
    fi
    expected_output_false=$(printf \
        '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":false}' \
        "$requested_version" "$requested_version")
    expected_output_true=$(printf \
        '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":true}' \
        "$requested_version" "$requested_version")
    case "$output" in
        "$expected_output_false")
            ;;
        "$expected_output_true")
            VERIFIED_FIRMWARE_UPDATE_UBOOT="true"
            ;;
        *)
            echo "Error: Firmware verifier returned an unexpected result"
            return 1
            ;;
    esac
    echo "Firmware archive signature and structure verified"
}

gen_b2sum() {
    local out_file="b2sum.txt"
    echo "Generating checksums for firmware and overlay files..."
    : > "$out_file"

    if [ -d "firmware" ] || [ -d "overlay" ]; then
        find firmware overlay -type f 2>/dev/null | sort | xargs b2sum > "$out_file"
    fi

    echo "b2sum have been saved to $out_file ($(wc -l < "$out_file") files)"
}

check_sum() {
    local sum_file="b2sum.txt"
    if [ ! -f "$sum_file" ] || [ -L "$sum_file" ]; then
        echo "Checksum file $sum_file not found!"
        return 1
    fi

    echo "Verifying checksums..."
    if b2sum -c "$sum_file"; then
        echo "All checksums are valid"
    else
        echo "Checksum verification failed!"
        return 1
    fi
}

sd_uboot_device_is_unmounted() {
    local device="$1"
    local targets=""
    local status=0

    if targets=$(findmnt -rn -S "$device" -o TARGET 2>/dev/null); then
        echo "Error: Refusing SD U-Boot update because $device is already mounted at: $targets"
        return 1
    else
        status=$?
    fi
    if [ "$status" -ne 1 ]; then
        echo "Error: Failed to inspect mounts for $device"
        return 1
    fi
}

validate_sd_uboot_mount() {
    local mount_dir="$1"
    local device="$2"
    local write_mode="${3:-any}"
    local record=""
    local target=""
    local source=""
    local fstype=""
    local options=""
    local extra=""
    local required_option=""

    case "$write_mode" in
        any|async|sync) ;;
        *)
            echo "Error: Invalid SD boot write mode"
            return 1
            ;;
    esac

    if ! record=$(findmnt -rn -o TARGET,SOURCE,FSTYPE,OPTIONS \
        --mountpoint "$mount_dir" 2>/dev/null); then
        echo "Error: Failed to inspect the temporary SD boot mount"
        return 1
    fi
    if [ -z "$record" ] || [[ "$record" == *$'\n'* ]]; then
        echo "Error: Expected exactly one SD boot mount"
        return 1
    fi
    read -r target source fstype options extra <<<"$record"
    if [ -n "$extra" ] || [ "$target" != "$mount_dir" ] ||
        [ "$source" != "$device" ] || [ "$fstype" != "vfat" ]; then
        echo "Error: Temporary SD boot mount has unexpected identity"
        return 1
    fi
    case ",$options," in
        *,ro,*)
            echo "Error: SD boot partition is mounted read-only"
            return 1
            ;;
        *,rw,*) ;;
        *)
            echo "Error: SD boot partition is not mounted read-write"
            return 1
            ;;
    esac
    for required_option in nosuid nodev noexec; do
        case ",$options," in
            *,"$required_option",*) ;;
            *)
                echo "Error: SD boot partition is missing $required_option"
                return 1
                ;;
        esac
    done
    case "$write_mode" in
        async)
            case ",$options," in
                *,sync,*)
                    echo "Error: SD boot staging mount is synchronous"
                    return 1
                    ;;
            esac
            ;;
        sync)
            case ",$options," in
                *,async,*)
                    echo "Error: SD boot commit mount is asynchronous"
                    return 1
                    ;;
                *,sync,*) ;;
                *)
                    echo "Error: SD boot commit mount is not synchronous"
                    return 1
                    ;;
            esac
            ;;
    esac
}

validate_private_mount_namespace() {
    local parent_namespace=""
    local current_namespace=""
    local propagation=""

    parent_namespace=$(readlink "/proc/$PPID/ns/mnt") || return 1
    current_namespace=$(readlink /proc/self/ns/mnt) || return 1
    if [ -z "$parent_namespace" ] || [ "$current_namespace" = "$parent_namespace" ]; then
        echo "Error: SD U-Boot worker is not in a private mount namespace"
        return 1
    fi
    propagation=$(findmnt -rn -o PROPAGATION --target / 2>/dev/null) || return 1
    case ",$propagation," in
        *,shared,*|*,unbindable,*)
            echo "Error: SD U-Boot worker has unsafe mount propagation"
            return 1
            ;;
        *,private,*) ;;
        *)
            echo "Error: SD U-Boot worker mount propagation is not private"
            return 1
            ;;
    esac
}

validate_sd_uboot_worker_lock() {
    local lock_file="/run/lock/nanokvm-sd-uboot.lock"
    local inherited_fd="${NANOKVM_SD_UBOOT_LOCK_FD:-}"
    local inherited_path=""

    if ! [[ "$inherited_fd" =~ ^[0-9]+$ ]] ||
        ! inherited_path=$(readlink -f -- "/proc/self/fd/$inherited_fd") ||
        [ "$inherited_path" != "$lock_file" ]; then
        echo "Error: SD U-Boot worker did not inherit the update lock"
        return 1
    fi
    if ! flock -n "$inherited_fd"; then
        echo "Error: Another SD U-Boot update is already running"
        return 1
    fi
}

sync_sd_uboot_filesystem() {
    local mount_dir="$1"
    local device="$2"

    if ! validate_sd_uboot_mount "$mount_dir" "$device"; then
        return 1
    fi
    if ! sync -f -- "$mount_dir"; then
        echo "Error: Failed to sync the SD boot filesystem"
        return 1
    fi
}

enable_sd_uboot_sync_writes() {
    local mount_dir="$1"
    local device="$2"

    if ! validate_sd_uboot_mount "$mount_dir" "$device" async; then
        return 1
    fi
    if ! mount -t vfat -o remount,rw,nosuid,nodev,noexec,sync \
        "$device" "$mount_dir"; then
        echo "Error: Failed to enable synchronous SD boot writes"
        return 1
    fi
    validate_sd_uboot_mount "$mount_dir" "$device" sync
}

safe_replace_sd_uboot() {
    (
        local source_file="$1"
        local mount_dir="$2"
        local device="$3"
        local target_file="uboot.bin"
        local candidate_file=""
        local backup_file=""
        local restore_file=""
        local keep_backup=0
        local commit_synced=0
        local restore_synced=0
        local temp_file=""

        cleanup_sd_uboot_files() {
            for temp_file in "$candidate_file" "$restore_file"; do
                case "$temp_file" in
                    .nkv-uboot-*) rm -f -- "$temp_file" || true ;;
                esac
            done
            if [ "$keep_backup" -eq 0 ]; then
                case "$backup_file" in
                    .nkv-uboot-*) rm -f -- "$backup_file" || true ;;
                esac
            fi
        }

        trap cleanup_sd_uboot_files EXIT
        trap 'exit 1' HUP INT TERM

        if [ ! -f "$source_file" ] || [ -L "$source_file" ] || [ ! -s "$source_file" ]; then
            echo "Error: SD U-Boot source must be a non-empty regular file"
            return 1
        fi
        if ! cd -- "$mount_dir"; then
            echo "Error: Failed to enter the temporary SD boot mount"
            return 1
        fi
        if [ ! -f "$target_file" ] || [ -L "$target_file" ] || [ ! -s "$target_file" ]; then
            echo "Error: Refusing to replace missing, empty, or non-regular uboot.bin"
            return 1
        fi
        if cmp -s -- "$source_file" "$target_file"; then
            echo "SD U-Boot is already up to date"
            return 0
        fi

        if ! candidate_file=$(mktemp ".nkv-uboot-candidate.XXXXXX"); then
            echo "Error: Failed to create the SD U-Boot candidate file"
            return 1
        fi
        if ! backup_file=$(mktemp ".nkv-uboot-backup.XXXXXX"); then
            echo "Error: Failed to create the SD U-Boot backup file"
            return 1
        fi
        if ! cp -- "$source_file" "$candidate_file" ||
            ! cp -- "$target_file" "$backup_file" ||
            ! sync_sd_uboot_filesystem "$mount_dir" "$device"; then
            echo "Error: Failed to stage the SD U-Boot update; current U-Boot was not changed"
            return 1
        fi
        if ! enable_sd_uboot_sync_writes "$mount_dir" "$device"; then
            echo "Error: Current U-Boot was not changed"
            return 1
        fi
        if ! cmp -s -- "$source_file" "$candidate_file" ||
            ! cmp -s -- "$target_file" "$backup_file"; then
            echo "Error: SD U-Boot inputs changed after enabling synchronous writes"
            return 1
        fi

        trap '' HUP INT TERM
        if ! mv -fT --no-copy -- "$candidate_file" "$target_file"; then
            echo "Warning: SD U-Boot rename reported an error; verifying the target"
        fi
        if sync_sd_uboot_filesystem "$mount_dir" "$device"; then
            commit_synced=1
        fi
        if [ "$commit_synced" -eq 1 ] && cmp -s -- "$source_file" "$target_file"; then
            candidate_file=""
            echo "SD U-Boot updated and verified"
            return 0
        fi
        if cmp -s -- "$backup_file" "$target_file"; then
            echo "Error: SD U-Boot update failed; previous image was preserved"
            return 1
        fi

        echo "Error: SD U-Boot verification failed; restoring the previous image"
        if ! restore_file=$(mktemp ".nkv-uboot-restore.XXXXXX") || \
            ! cp -- "$backup_file" "$restore_file" ||
            ! sync_sd_uboot_filesystem "$mount_dir" "$device" ||
            ! cmp -s -- "$backup_file" "$restore_file"; then
            keep_backup=1
            echo "Critical: Previous image retained at $device:/$backup_file"
            return 1
        fi
        if ! mv -fT --no-copy -- "$restore_file" "$target_file"; then
            echo "Warning: SD U-Boot restore rename reported an error; verifying the target"
        fi
        if sync_sd_uboot_filesystem "$mount_dir" "$device"; then
            restore_synced=1
        fi
        if cmp -s -- "$backup_file" "$target_file"; then
            restore_file=""
            if [ "$restore_synced" -ne 1 ]; then
                keep_backup=1
                echo "Critical: Restored image could not be persisted; backup retained at $device:/$backup_file"
            else
                echo "Previous SD U-Boot image restored and verified"
            fi
            return 1
        fi

        keep_backup=1
        echo "Critical: Failed to restore SD U-Boot; backup retained at $device:/$backup_file"
        return 1
    )
}

perform_private_sd_uboot_update() {
    (
        local source_file="$1"
        local mount_dir="$2"
        local device="/dev/mmcblk1p1"
        local mounted=0
        local update_status=1

        cleanup_private_sd_mount() {
            if [ "$mounted" -eq 1 ]; then
                cd / || true
                sync_sd_uboot_filesystem "$mount_dir" "$device" || true
                umount "$mount_dir" || true
                mounted=0
            fi
        }

        trap cleanup_private_sd_mount EXIT
        trap 'exit 1' HUP INT TERM

        if [ "$EUID" -ne 0 ] || [ ! -b "$device" ] ||
            [ ! -d "$mount_dir" ] || [ -L "$mount_dir" ] ||
            [[ "$mount_dir" != /run/.nanokvm-sd-boot.* ]]; then
            echo "Error: Invalid private SD U-Boot worker environment"
            return 1
        fi
        if ! sd_uboot_device_is_unmounted "$device"; then
            return 1
        fi
        if mount -t vfat -o rw,nosuid,nodev,noexec,async "$device" "$mount_dir"; then
            mounted=1
        else
            update_status=$?
            if findmnt -rn --mountpoint "$mount_dir" >/dev/null 2>&1; then
                mounted=1
            fi
            echo "Error: Failed to mount $device for the SD U-Boot update"
            return "$update_status"
        fi
        if ! validate_sd_uboot_mount "$mount_dir" "$device" async; then
            return 1
        fi

        if safe_replace_sd_uboot "$source_file" "$mount_dir" "$device"; then
            update_status=0
        else
            update_status=$?
        fi
        if ! sync_sd_uboot_filesystem "$mount_dir" "$device"; then
            echo "Error: Failed to sync the SD boot partition before unmount"
            update_status=1
        fi

        cd / || true
        if ! umount "$mount_dir"; then
            echo "Critical: Failed to unmount the SD boot partition after update"
            update_status=1
        else
            mounted=0
        fi
        if findmnt -rn --mountpoint "$mount_dir" >/dev/null 2>&1; then
            echo "Critical: SD boot partition remains mounted after update"
            update_status=1
        fi
        return "$update_status"
    )
}

update_sd_uboot() {
    (
        local device="/dev/mmcblk1p1"
        local source_path=""
        local script_path=""
        local mount_dir=""
        local lock_file="/run/lock/nanokvm-sd-uboot.lock"
        local lock_fd=""
        local update_status=1

        cleanup_sd_mount_dir() {
            if [ -n "$mount_dir" ] && [[ "$mount_dir" == /run/.nanokvm-sd-boot.* ]] &&
                [ -d "$mount_dir" ]; then
                rmdir -- "$mount_dir" || true
            fi
        }

        trap cleanup_sd_mount_dir EXIT
        trap 'exit 1' HUP INT TERM

        if [ "$EUID" -ne 0 ] || [ ! -b "$device" ]; then
            echo "Error: SD U-Boot update requires root and $device"
            return 1
        fi
        if [ ! -f "$UBOOT_FILE" ] || [ -L "$UBOOT_FILE" ] || [ ! -s "$UBOOT_FILE" ]; then
            echo "Error: Invalid SD U-Boot source"
            return 1
        fi
        source_path=$(readlink -f -- "$UBOOT_FILE") || return 1
        script_path=$(readlink -f -- "$0") || return 1
        if [ ! -x "$script_path" ]; then
            echo "Error: Invalid SD U-Boot source or update script"
            return 1
        fi

        install -d -m 0755 -- /run/lock || return 1
        if ! exec {lock_fd}>"$lock_file" || ! flock -n "$lock_fd"; then
            echo "Error: Another SD U-Boot update is already running"
            return 1
        fi
        if ! sd_uboot_device_is_unmounted "$device"; then
            return 1
        fi
        mount_dir=$(mktemp -d "/run/.nanokvm-sd-boot.XXXXXX") || return 1

        if NANOKVM_SD_UBOOT_LOCK_FD="$lock_fd" \
            unshare --mount --propagation private \
            "$script_path" __sd_uboot_update "$source_path" "$mount_dir"; then
            update_status=0
        else
            update_status=$?
        fi
        if ! rmdir -- "$mount_dir"; then
            echo "Critical: Failed to remove temporary SD boot mount directory"
            return 1
        fi
        mount_dir=""
        return "$update_status"
    )
}

flash_image() {
    local file="$1"
    local script="$2"
    shift 2
    local partitions=("$@")

    if [ ! -f "$file" ]; then
        return 0
    fi

    echo "Flashing $file ..."
    if [ -f "$script" ]; then
        if ! "$script" "$file" > /dev/null 2>&1; then
            echo "Error: Failed to flash $file with $script"
            return 1
        fi
    else
        for part in "${partitions[@]}"; do
            if ! dd if="$file" of="$part" bs=4K conv=notrunc > /dev/null 2>&1; then
                echo "Error: Failed to flash $file to $part"
                return 1
            fi
        done
    fi
}

flash_verified_firmware() {
    (
    local reboot_marker_candidate=""
    local update_uboot="${FIRMWARE_UPDATE_UBOOT:-}"

    case "$update_uboot" in
        true)
            echo "U-Boot update policy: enabled"
            ;;
        false)
            echo "U-Boot update policy: disabled; skipping U-Boot"
            ;;
        *)
            echo "Error: Invalid verified U-Boot update policy"
            return 1
            ;;
    esac

    cleanup_reboot_marker_candidate() {
        if [ -n "$reboot_marker_candidate" ] &&
            [[ "$reboot_marker_candidate" == /var/run/.reboot-required.* ]]; then
            rm -f -- "$reboot_marker_candidate" || true
        fi
    }

    trap cleanup_reboot_marker_candidate EXIT

    echo "Checking b2sum ..."
    if ! check_sum; then
        return 1
    fi

    if [ ! -f overlay/boot/ver ] || [ -L overlay/boot/ver ]; then
        echo "Version file not found!"
        return 1
    fi
    if ! reboot_marker_candidate=$(mktemp "/var/run/.reboot-required.XXXXXX"); then
        echo "Error: Failed to prepare the firmware reboot marker"
        return 1
    fi

    local root_dev
    root_dev=$(grep -o 'root=[^ ]*' /proc/cmdline | head -n1 | cut -d= -f2)

    if [ "$update_uboot" = "true" ] && {
        [ ! -f "$UBOOT_FILE" ] || [ -L "$UBOOT_FILE" ] || [ ! -s "$UBOOT_FILE" ]
    }; then
        echo "Error: U-Boot update was requested but the image is missing or invalid"
        return 1
    fi

    if [ "$root_dev" = "/dev/mmcblk0p17" ]; then
        echo "Updating firmware for EMMC boot ..."
        if [ "$update_uboot" = "true" ] && \
            ! flash_image "$UBOOT_FILE" /kvmcomm/scripts/axuboot.sh /dev/mmcblk0p5 /dev/mmcblk0p6; then
            return 1
        fi
        if ! flash_image "$DTB_FILE" /kvmcomm/scripts/axdtb.sh /dev/mmcblk0p12 /dev/mmcblk0p13 ||
            ! flash_image "$KERNEL_FILE" /kvmcomm/scripts/axkernel.sh /dev/mmcblk0p14 /dev/mmcblk0p15; then
            return 1
        fi
    elif [ "$root_dev" = "/dev/mmcblk1p2" ]; then
        echo "Updating firmware for SD boot ..."
        if [ "$update_uboot" = "true" ] && ! update_sd_uboot; then
            return 1
        fi
        if ! cp "$DTB_FILE" /boot/dtb.img || ! cp "$KERNEL_FILE" /boot/kernel.img; then
            echo "Error: Failed to update SD boot DTB or kernel"
            return 1
        fi
    else
        echo "Unsupported boot root device: ${root_dev:-unknown}"
        echo "Expected root=/dev/mmcblk0p17 (EMMC) or root=/dev/mmcblk1p2 (SD)"
        return 1
    fi

    echo "Syncing overlay files to root filesystem ..."
    if ! rsync -aR --exclude='/boot' overlay/./ / ||
        ! cp -r overlay/boot/* /boot ||
        ! sync; then
        echo "Error: Failed to commit firmware overlay"
        return 1
    fi
    if mv -fT --no-copy -- "$reboot_marker_candidate" /var/run/reboot-required; then
        reboot_marker_candidate=""
    elif [ ! -f /var/run/reboot-required ]; then
        echo "Error: Firmware was updated but the reboot marker could not be created"
        return 1
    fi

    echo "Firmware update completed"
    )
}

validate_firmware_pair_directory() {
    local pair_dir="$1"
    local version="$2"
    local metadata_path="${pair_dir}/firmware_${version}.json"
    local archive_path="${pair_dir}/axera_firmware_v${version}.tar.gz"
    local entry_count=""

    if [ ! -d "$pair_dir" ] || [ -L "$pair_dir" ] || \
        [ ! -f "$metadata_path" ] || [ -L "$metadata_path" ] || \
        [ ! -f "$archive_path" ] || [ -L "$archive_path" ]; then
        return 1
    fi
    entry_count=$(find "$pair_dir" -mindepth 1 -maxdepth 1 -printf '.' | wc -c)
    [ "$entry_count" -eq 2 ]
}

extract_verified_firmware() {
    local pair_dir="$1"
    local version="$2"
    local archive_path="${pair_dir}/axera_firmware_v${version}.tar.gz"
    local extract_dir="${ACTIVE_WORK_DIR}/extracted"

    rm -rf -- "$extract_dir"
    if ! install -d -m 0700 -- "$extract_dir"; then
        return 1
    fi
    echo "Extracting verified firmware archive..."
    if ! (
        umask 022
        tar --warning=no-timestamp \
            -xzf "$archive_path" \
            -C "$extract_dir" \
            --no-same-owner \
            --no-same-permissions \
            --no-overwrite-dir
    ); then
        echo "Error: Failed to extract verified firmware archive"
        return 1
    fi
    if ! (cd -- "$extract_dir" && check_sum); then
        echo "Error: Extracted firmware checksum verification failed"
        return 1
    fi
    FIRMWARE_EXTRACT_DIR="$extract_dir"
}

prepare_firmware_pair() {
    local pair_dir="$1"
    local version="$2"
    local allow_same_version="$3"
    local metadata_path="${pair_dir}/firmware_${version}.json"
    local archive_path="${pair_dir}/axera_firmware_v${version}.tar.gz"
    local verified_update_uboot="false"

    # A failed or replacement candidate must not reuse a previous policy or
    # extracted tree. The prepared policy is published only after extraction.
    FIRMWARE_EXTRACT_DIR=""
    FIRMWARE_UPDATE_UBOOT="false"

    if ! validate_firmware_pair_directory "$pair_dir" "$version"; then
        echo "Error: Firmware pair must contain exactly the signed metadata and .tar.gz archive"
        return 1
    fi
    if ! verify_firmware_pair \
        "$metadata_path" "$archive_path" "$version" "$allow_same_version"; then
        return 1
    fi
    verified_update_uboot="$VERIFIED_FIRMWARE_UPDATE_UBOOT"
    if ! extract_verified_firmware "$pair_dir" "$version"; then
        return 1
    fi
    FIRMWARE_UPDATE_UBOOT="$verified_update_uboot"
}

copy_firmware_pair() {
    local source_dir="$1"
    local destination_dir="$2"
    local version="$3"
    local metadata_name="firmware_${version}.json"
    local archive_name="axera_firmware_v${version}.tar.gz"

    if ! validate_firmware_pair_directory "$source_dir" "$version"; then
        return 1
    fi
    if ! install -d -m 0700 -- "$destination_dir" || \
        ! install -m 0600 -- "${source_dir}/${metadata_name}" "${destination_dir}/${metadata_name}" || \
        ! install -m 0600 -- "${source_dir}/${archive_name}" "${destination_dir}/${archive_name}"; then
        echo "Error: Failed to stage firmware pair"
        return 1
    fi
}

use_cached_firmware_pair() {
    local version="$1"
    local allow_same_version="$2"
    local cache_dir="${CACHE_BASE_DIR}/firmware_${version}"
    local staged_dir=""

    if ! validate_firmware_pair_directory "$cache_dir" "$version"; then
        return 1
    fi
    if ! create_work_dir; then
        return 1
    fi
    staged_dir="${ACTIVE_WORK_DIR}/firmware_${version}"
    if ! copy_firmware_pair "$cache_dir" "$staged_dir" "$version" || \
        ! prepare_firmware_pair "$staged_dir" "$version" "$allow_same_version"; then
        echo "Cached firmware pair is not trusted; downloading a replacement"
        cleanup_active_work_dir
        return 1
    fi
    echo "Verified cached firmware pair for version $version"
}

publish_firmware_cache() {
    local candidate_dir="$1"
    local version="$2"
    local allow_same_version="$3"
    local target_dir="${CACHE_BASE_DIR}/firmware_${version}"
    local candidate_update_uboot="${FIRMWARE_UPDATE_UBOOT:-false}"
    local cached_update_uboot="false"

    if validate_firmware_pair_directory "$target_dir" "$version" && \
        verify_firmware_pair \
            "${target_dir}/firmware_${version}.json" \
            "${target_dir}/axera_firmware_v${version}.tar.gz" \
            "$version" "$allow_same_version" >/dev/null; then
        cached_update_uboot="$VERIFIED_FIRMWARE_UPDATE_UBOOT"
        if [ "$cached_update_uboot" = "$candidate_update_uboot" ]; then
            echo "Existing signed firmware cache remains trusted"
            return 0
        fi
        if ! cmp -s -- \
            "${candidate_dir}/axera_firmware_v${version}.tar.gz" \
            "${target_dir}/axera_firmware_v${version}.tar.gz"; then
            echo "Error: Same-version firmware cache has a different archive and U-Boot policy"
            return 1
        fi
        echo "Replacing same-version firmware cache to update U-Boot policy"
    fi

    if [ -e "$target_dir" ] || [ -L "$target_dir" ]; then
        if ! mv -T --exchange --no-copy -- "$candidate_dir" "$target_dir"; then
            echo "Error: Failed to atomically exchange firmware cache"
            return 1
        fi
        if ! validate_firmware_pair_directory "$target_dir" "$version" || \
            ! verify_firmware_pair \
                "${target_dir}/firmware_${version}.json" \
                "${target_dir}/axera_firmware_v${version}.tar.gz" \
                "$version" "$allow_same_version" >/dev/null; then
            echo "Error: Published firmware cache failed verification; rolling back"
            if ! mv -T --exchange --no-copy -- "$candidate_dir" "$target_dir"; then
                echo "Error: Failed to restore previous firmware cache"
            fi
            return 1
        fi
        rm -rf -- "$candidate_dir"
    else
        if ! mv -T --no-copy -- "$candidate_dir" "$target_dir"; then
            echo "Error: Failed to atomically publish firmware cache"
            return 1
        fi
        if ! validate_firmware_pair_directory "$target_dir" "$version" || \
            ! verify_firmware_pair \
                "${target_dir}/firmware_${version}.json" \
                "${target_dir}/axera_firmware_v${version}.tar.gz" \
                "$version" "$allow_same_version" >/dev/null; then
            echo "Error: Published firmware cache failed verification"
            if ! mv -T --no-copy -- "$target_dir" "$candidate_dir"; then
                echo "Error: Failed to withdraw invalid firmware cache"
            fi
            return 1
        fi
    fi
    echo "Published signed firmware cache for version $version"
}

download_budget() {
    local available_bytes=""

    available_bytes=$(df -PB1 -- "$CACHE_BASE_DIR" | awk 'NR == 2 { print $4 }') || return 1
    if ! [[ "$available_bytes" =~ ^[0-9]+$ ]] || \
        [ "$available_bytes" -le "$FIRMWARE_DISK_SAFETY_MARGIN" ]; then
        echo "Error: Insufficient disk space for firmware download" >&2
        return 1
    fi
    printf '%s\n' "$((available_bytes - FIRMWARE_DISK_SAFETY_MARGIN))"
}

download_file_with_retries() {
    local url="$1"
    local destination="$2"
    local max_bytes="$3"
    local connect_timeout="$4"
    local total_timeout="$5"
    local retries=3
    local retry_delay=1
    local attempt=0
    local partial="${destination}.part"

    rm -f -- "$destination" "$partial"
    for ((attempt=1; attempt<=retries; attempt++)); do
        if curl -fsSL \
            --connect-timeout "$connect_timeout" \
            --max-time "$total_timeout" \
            --max-filesize "$max_bytes" \
            "$url" \
            -o "$partial"; then
            chmod 0600 -- "$partial"
            mv -T --no-copy -- "$partial" "$destination"
            return 0
        fi
        rm -f -- "$partial"
        if [ "$attempt" -lt "$retries" ]; then
            sleep "$retry_delay"
        fi
    done
    return 1
}

fetch_firmware_pair_from_cdn() {
    local requested="$1"
    local allow_same_version="$2"
    local version=""
    local metadata_name=""
    local archive_name=""
    local base=""
    local candidate_dir=""
    local archive_budget=""

    version=$(normalize_firmware_version "$requested") || return 1
    metadata_name="firmware_${version}.json"
    archive_name="axera_firmware_v${version}.tar.gz"

    if ! wait_for_network "$metadata_name"; then
        echo "Continuing firmware download attempt after network wait timeout"
    fi
    if ! create_work_dir; then
        return 1
    fi

    for base in "${CDN_URLS[@]}"; do
        candidate_dir="${ACTIVE_WORK_DIR}/firmware_${version}"
        rm -rf -- "$candidate_dir" "${ACTIVE_WORK_DIR}/extracted"
        install -d -m 0700 -- "$candidate_dir"

        echo "Trying signed firmware pair from: $base"
        if ! download_file_with_retries \
            "${base}/${metadata_name}" \
            "${candidate_dir}/${metadata_name}" \
            "$FIRMWARE_METADATA_MAX_BYTES" 5 30; then
            echo "Failed to download firmware metadata from: $base"
            continue
        fi
        archive_budget=$(download_budget) || return 1
        if ! download_file_with_retries \
            "${base}/${archive_name}" \
            "${candidate_dir}/${archive_name}" \
            "$archive_budget" 15 300; then
            echo "Failed to download firmware archive from: $base"
            continue
        fi
        if ! prepare_firmware_pair \
            "$candidate_dir" "$version" "$allow_same_version"; then
            echo "Signed firmware pair from $base failed verification"
            continue
        fi
        if ! publish_firmware_cache \
            "$candidate_dir" "$version" "$allow_same_version"; then
            continue
        fi
        echo "Signed firmware pair is ready from: $base"
        return 0
    done

    echo "Error: Failed to obtain a valid signed firmware pair for version $version"
    cleanup_active_work_dir
    return 1
}

stage_local_firmware_pair() {
    local archive_source="$1"
    local metadata_source="$2"
    local metadata_name=""
    local archive_name=""
    local version=""
    local candidate_dir=""

    if [ ! -f "$archive_source" ] || [ -L "$archive_source" ] || \
        [ ! -f "$metadata_source" ] || [ -L "$metadata_source" ]; then
        echo "Error: Local firmware update requires a regular archive and metadata file"
        return 1
    fi
    metadata_name=$(basename -- "$metadata_source")
    if [[ "$metadata_name" =~ ^firmware_((0|[1-9][0-9]{0,19})\.(0|[1-9][0-9]{0,19})\.(0|[1-9][0-9]{0,19}))\.json$ ]]; then
        version="${BASH_REMATCH[1]}"
    else
        echo "Error: Local firmware metadata filename is invalid"
        return 1
    fi
    archive_name="axera_firmware_v${version}.tar.gz"
    if [ "$(basename -- "$archive_source")" != "$archive_name" ]; then
        echo "Error: Local firmware archive filename does not match metadata"
        return 1
    fi
    if ! create_work_dir; then
        return 1
    fi
    candidate_dir="${ACTIVE_WORK_DIR}/firmware_${version}"
    if ! install -d -m 0700 -- "$candidate_dir" || \
        ! install -m 0600 -- "$metadata_source" "${candidate_dir}/${metadata_name}" || \
        ! install -m 0600 -- "$archive_source" "${candidate_dir}/${archive_name}"; then
        echo "Error: Failed to stage local firmware pair"
        return 1
    fi
    if ! prepare_firmware_pair "$candidate_dir" "$version" "true" || \
        ! publish_firmware_cache "$candidate_dir" "$version" "true"; then
        return 1
    fi
    FIRMWARE_VERSION="$version"
}

prepare_firmware_version() {
    local requested="$1"
    local allow_same_version="$2"
    local version=""

    version=$(normalize_firmware_version "$requested") || return 1
    if use_cached_firmware_pair "$version" "$allow_same_version"; then
        FIRMWARE_VERSION="$version"
        return 0
    fi
    if ! fetch_firmware_pair_from_cdn "$version" "$allow_same_version"; then
        return 1
    fi
    FIRMWARE_VERSION="$version"
}

install_prepared_firmware() {
    if [ -z "${FIRMWARE_EXTRACT_DIR:-}" ] || \
        [ ! -d "$FIRMWARE_EXTRACT_DIR" ] || [ -L "$FIRMWARE_EXTRACT_DIR" ]; then
        echo "Error: No verified firmware is prepared"
        return 1
    fi
    (cd -- "$FIRMWARE_EXTRACT_DIR" && flash_verified_firmware)
}

case "${1:-}" in
    __sd_uboot_update)
        if [ "$#" -ne 3 ]; then
            echo "Error: Invalid private SD U-Boot worker arguments"
            exit 1
        fi
        if ! validate_private_mount_namespace || \
            ! validate_sd_uboot_worker_lock || \
            ! perform_private_sd_uboot_update "$2" "$3"; then
            exit 1
        fi
        ;;
    gen_b2sum)
        echo "Generating b2sum ..."
        gen_b2sum
        ;;
    fetch)
        if [ "$#" -ne 2 ]; then
            echo "Usage: $0 fetch <version>"
            exit 1
        fi
        if ! prepare_firmware_version "$2" "false"; then
            exit 1
        fi
        echo "Verified firmware version $FIRMWARE_VERSION is cached and ready"
        ;;
    update)
        if [ "$#" -eq 2 ]; then
            if ! prepare_firmware_version "$2" "false"; then
                exit 1
            fi
        elif [ "$#" -eq 3 ] && [ "$2" = "--allow-same-version" ]; then
            if ! prepare_firmware_version "$3" "true"; then
                exit 1
            fi
        elif [ "$#" -eq 3 ]; then
            if ! stage_local_firmware_pair "$2" "$3"; then
                exit 1
            fi
        else
            echo "Usage: $0 update <version>"
            echo "   or: $0 update --allow-same-version <version>"
            echo "   or: $0 update <archive.tar.gz> <firmware_version.json>"
            exit 1
        fi
        if ! install_prepared_firmware; then
            exit 1
        fi
        ;;
    *)
        echo "usage:"
        echo "  $0 gen_b2sum"
        echo "  $0 fetch <version>"
        echo "  $0 update <version>"
        echo "  $0 update --allow-same-version <version>"
        echo "  $0 update <archive.tar.gz> <firmware_version.json>"
        exit 1
        ;;
esac
