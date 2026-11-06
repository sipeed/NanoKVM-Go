#!/usr/bin/env bash

set -u
set -o pipefail

TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR
WIFI_SCRIPT="${TEST_DIR}/../rootfs/opt/scripts/wifi.sh"
readonly WIFI_SCRIPT
TEST_TMP=$(mktemp -d) || exit 1
readonly TEST_TMP
trap 'rm -rf "$TEST_TMP"' EXIT

fail() {
    printf 'not ok - %s\n' "$*" >&2
    exit 1
}

assert_eq() {
    local name="$1"
    local want="$2"
    local got="$3"
    [[ "$got" == "$want" ]] || fail "$name: got '$got', want '$want'"
    printf 'ok - %s\n' "$name"
}

assert_file_absent() {
    local name="$1"
    local path="$2"
    [[ ! -e "$path" && ! -L "$path" ]] || fail "$name: $path still exists"
    printf 'ok - %s\n' "$name"
}

extract_function() {
    local function_name="$1"
    awk -v function_name="$function_name" '
        $0 ~ ("^" function_name "\\(\\)[[:space:]]*\\{[[:space:]]*$") {
            in_function = 1
            found_start = 1
        }
        in_function {
            print
            if ($0 ~ /^}[[:space:]]*$/) {
                found_end = 1
                exit
            }
        }
        END {
            if (!found_start) exit 2
            if (!found_end) exit 3
        }
    ' "$WIFI_SCRIPT"
}

[[ -r "$WIFI_SCRIPT" ]] || fail "cannot read $WIFI_SCRIPT"

export KVM_CONFIG_DIR="${TEST_TMP}/kvm"
constant_sources=$(awk '
    /^readonly[[:space:]]+(KVM_CONFIG_DIR|WIFI_STATE_FILE|WIFI_LEGACY_CFG_FILE|WIFI_LEGACY_LAST_FILE|WIFI_LEGACY_SAVE_FILE)=/ { print }
    /^WIFI_STATE_DIR=/ { print }
    /^readonly[[:space:]]+WIFI_STATE_DIR$/ { print }
' "$WIFI_SCRIPT") || fail "could not extract WiFi state constants"

function_sources=""
for function_name in \
    base64_encode_field \
    base64_decode_field \
    wifi_state_ssid_is_valid \
    wifi_state_decode_ssid \
    wifi_state_reset_loaded_values \
    load_wifi_state_file \
    wifi_state_sync_path \
    write_wifi_state_file \
    read_legacy_wifi_cfg_ssid \
    read_legacy_wifi_ssid_file \
    cleanup_legacy_wifi_state_files \
    migrate_legacy_wifi_state \
    load_or_migrate_wifi_state \
    persist_connected_wifi_state \
    print_wifi_boot_enabled \
    turn_on_wifi \
    disconnect_wifi \
    clear_saved_networks \
    stop_wifi_system; do
    function_source=$(extract_function "$function_name") ||
        fail "could not extract $function_name"
    function_sources+="${function_source}"$'\n'
done

# Evaluate only the state-format owner and its explicit dependencies. Sourcing
# the device script would acquire locks and operate on system network state.
eval "${constant_sources}"$'\n'"${function_sources}"

write_wifi_state_file 1 "中文 WiFi" || fail "could not atomically write valid state"
assert_eq "state write uses mode 0600" "600" "$(stat -c '%a' "$WIFI_STATE_FILE")"
load_wifi_state_file || fail "could not load written state"
assert_eq "state write/load preserves enabled intent" "1" "$WIFI_STATE_ENABLED"
assert_eq "state write/load preserves UTF-8 SSID" "中文 WiFi" "$WIFI_STATE_PREFERRED_SSID"
assert_eq "state file is deterministic" $'WIFI_STATE_VERSION=1\nWIFI_ENABLED=1\nWIFI_PREFERRED_SSID_B64=5Lit5paHIFdpRmk=\n' "$(cat "$WIFI_STATE_FILE")"$'\n'

mktemp() {
    return 1
}
if write_wifi_state_file 0 "Must Not Publish"; then
    fail "state write must report temporary-file allocation failure"
fi
unset -f mktemp
load_wifi_state_file || fail "atomic write failure must retain the prior valid state"
assert_eq "failed state replacement retains enabled intent" "1" "$WIFI_STATE_ENABLED"
assert_eq "failed state replacement retains the previous SSID" "中文 WiFi" "$WIFI_STATE_PREFERRED_SSID"

if write_wifi_state_file 1 "123456789012345678901234567890123"; then
    fail "SSIDs over 32 bytes must be rejected"
fi
assert_eq "SSID length validation rejects overlong values" "1" "$WIFI_STATE_ENABLED"

printf '%s\n' \
    'WIFI_STATE_VERSION=1' \
    'WIFI_ENABLED=1' \
    'WIFI_ENABLED=0' \
    'WIFI_PREFERRED_SSID_B64=' >"$WIFI_STATE_FILE"
if load_wifi_state_file; then
    fail "duplicate state keys must be rejected"
fi
assert_eq "duplicate state fails disabled" "0" "$WIFI_STATE_ENABLED"

printf '%s\n' \
    'WIFI_STATE_VERSION=1' \
    'WIFI_ENABLED=1' \
    'WIFI_PREFERRED_SSID_B64=YQ==' \
    'UNEXPECTED=value' >"$WIFI_STATE_FILE"
if load_wifi_state_file; then
    fail "unknown state keys must be rejected"
fi
printf 'WIFI_STATE_VERSION=1\nWIFI_ENABLED=1\nWIFI_PREFERRED_SSID_B64=YQ==' >"$WIFI_STATE_FILE"
if load_wifi_state_file; then
    fail "state without a final newline must be rejected"
fi
printf '%s\n' \
    'WIFI_STATE_VERSION=1' \
    'WIFI_ENABLED=1' \
    'WIFI_PREFERRED_SSID_B64=YQ=' >"$WIFI_STATE_FILE"
if load_wifi_state_file; then
    fail "non-canonical Base64 must be rejected"
fi
printf 'ok - malformed states fail disabled\n'

rm -f "$WIFI_STATE_FILE" "$WIFI_LEGACY_CFG_FILE" "$WIFI_LEGACY_LAST_FILE" "$WIFI_LEGACY_SAVE_FILE"
printf 'WIFI_CFG_SSID=Primary Network\n' >"$WIFI_LEGACY_CFG_FILE"
printf 'Last Network\n' >"$WIFI_LEGACY_LAST_FILE"
load_or_migrate_wifi_state || fail "legacy enabled migration failed"
assert_eq "legacy wifi.conf sets enabled intent" "1" "$WIFI_STATE_ENABLED"
assert_eq "legacy wifi.conf wins preference" "Primary Network" "$WIFI_STATE_PREFERRED_SSID"
assert_file_absent "legacy files are removed after state publish" "$WIFI_LEGACY_CFG_FILE"
assert_file_absent "legacy last marker is removed after state publish" "$WIFI_LEGACY_LAST_FILE"

rm -f "$WIFI_STATE_FILE" "$WIFI_LEGACY_CFG_FILE" "$WIFI_LEGACY_LAST_FILE" "$WIFI_LEGACY_SAVE_FILE"
printf 'Last Network\n' >"$WIFI_LEGACY_LAST_FILE"
printf 'Older Network\n' >"$WIFI_LEGACY_SAVE_FILE"
load_or_migrate_wifi_state || fail "legacy disabled migration failed"
assert_eq "legacy last marker remains disabled" "0" "$WIFI_STATE_ENABLED"
assert_eq "legacy last marker wins fallback preference" "Last Network" "$WIFI_STATE_PREFERRED_SSID"

rm -f "$WIFI_STATE_FILE" "$WIFI_LEGACY_CFG_FILE" "$WIFI_LEGACY_LAST_FILE" "$WIFI_LEGACY_SAVE_FILE"
printf 'WIFI_CFG_SSID=\n' >"$WIFI_LEGACY_CFG_FILE"
printf 'Retained Last\n' >"$WIFI_LEGACY_LAST_FILE"
load_or_migrate_wifi_state || fail "malformed legacy config fallback migration failed"
assert_eq "empty legacy wifi.conf fails disabled" "0" "$WIFI_STATE_ENABLED"
assert_eq "valid legacy last SSID is retained disabled" "Retained Last" "$WIFI_STATE_PREFERRED_SSID"

printf '%s\n' \
    'WIFI_STATE_VERSION=999' \
    'WIFI_ENABLED=1' \
    'WIFI_PREFERRED_SSID_B64=UHJpbWFyeQ==' >"$WIFI_STATE_FILE"
printf 'WIFI_CFG_SSID=Should Not Be Used\n' >"$WIFI_LEGACY_CFG_FILE"
if load_or_migrate_wifi_state; then
    fail "invalid unified state must not fall back to legacy files"
fi
assert_eq "invalid unified state is fail-disabled" "0" "$WIFI_STATE_ENABLED"
assert_eq "invalid unified state clears preference" "" "$WIFI_STATE_PREFERRED_SSID"
[[ -f "$WIFI_LEGACY_CFG_FILE" ]] || fail "invalid unified state must not clean legacy input"
printf 'ok - migration precedence and invalid-state fail-closed behavior\n'

write_wifi_state_file 1 "Boot Network" || fail "could not restore boot state"
if ! WIFI_TMP_DIR="${TEST_TMP}/must-not-exist" "$WIFI_SCRIPT" boot_enabled \
    >"${TEST_TMP}/boot-output"; then
    fail "valid boot_enabled command failed"
fi
assert_eq "boot_enabled emits exact enabled token" "true" "$(cat "${TEST_TMP}/boot-output")"
[[ ! -e "${TEST_TMP}/must-not-exist" ]] || fail "boot_enabled must not create the runtime directory"
write_wifi_state_file 0 "Disabled Network" || fail "could not prepare disabled boot state"
if ! "$WIFI_SCRIPT" boot_enabled >"${TEST_TMP}/boot-output"; then
    fail "valid disabled boot_enabled command failed"
fi
assert_eq "boot_enabled emits exact disabled token" "false" "$(cat "${TEST_TMP}/boot-output")"
rm -f "$WIFI_STATE_FILE"
if "$WIFI_SCRIPT" boot_enabled >"${TEST_TMP}/boot-output"; then
    fail "missing state must make boot_enabled fail closed"
fi
assert_eq "boot_enabled emits false for missing state" "false" "$(cat "${TEST_TMP}/boot-output")"
printf 'WIFI_CFG_SSID=Legacy Must Not Migrate\n' >"$WIFI_LEGACY_CFG_FILE"
if "$WIFI_SCRIPT" boot_enabled >"${TEST_TMP}/boot-output"; then
    fail "boot_enabled must not migrate legacy state"
fi
assert_eq "boot_enabled emits false instead of migrating legacy state" "false" \
    "$(cat "${TEST_TMP}/boot-output")"
[[ -f "$WIFI_LEGACY_CFG_FILE" && ! -e "$WIFI_STATE_FILE" ]] ||
    fail "boot_enabled must leave legacy files and unified state untouched"
rm -f "$WIFI_LEGACY_CFG_FILE"
printf '%s\n' \
    'WIFI_STATE_VERSION=999' \
    'WIFI_ENABLED=1' \
    'WIFI_PREFERRED_SSID_B64=' >"$WIFI_STATE_FILE"
if "$WIFI_SCRIPT" boot_enabled >"${TEST_TMP}/boot-output"; then
    fail "invalid state must make boot_enabled fail closed"
fi
assert_eq "boot_enabled emits false for invalid state" "false" \
    "$(cat "${TEST_TMP}/boot-output")"

debug_print() {
    :
}

get_wireless_interface() {
    return 1
}

clear_wifi_alias_registry() {
    :
}

stop_wifi_ap() {
    :
}

stop_service() {
    :
}

ensure_wpa_conf() {
    :
}

mark_saved_networks_disabled() {
    :
}

clear_ap_previous_state() {
    :
}

is_ap_runtime_active() {
    return 1
}

write_wifi_state_file 1 "Remembered Network" || fail "could not prepare off transition"
disconnect_wifi "manual-off" || fail "off transition must commit before runtime work"
load_wifi_state_file || fail "could not load disabled off transition"
assert_eq "off keeps the preference while disabling intent" "0" "$WIFI_STATE_ENABLED"
assert_eq "off retains the remembered preference" "Remembered Network" "$WIFI_STATE_PREFERRED_SSID"

write_wifi_state_file 0 "Unavailable Interface Network" ||
    fail "could not prepare no-interface enable transition"
if turn_on_wifi >/dev/null 2>&1; then
    fail "on fixture must report no interface after committing enabled intent"
fi
load_wifi_state_file || fail "could not load no-interface enable transition"
assert_eq "on retains enabled intent when the interface is unavailable" "1" "$WIFI_STATE_ENABLED"
assert_eq "on retains the preference when the interface is unavailable" \
    "Unavailable Interface Network" "$WIFI_STATE_PREFERRED_SSID"

write_wifi_state_file 1 "AP Transition Network" || fail "could not prepare AP transition"
disconnect_wifi "ap-switch" || fail "AP transition disconnect failed"
load_wifi_state_file || fail "could not load AP transition state"
assert_eq "AP switch preserves enabled intent" "1" "$WIFI_STATE_ENABLED"
assert_eq "AP switch preserves preference" "AP Transition Network" "$WIFI_STATE_PREFERRED_SSID"

write_wifi_state_file 1 "Restart Network" || fail "could not prepare service stop transition"
stop_wifi_system >/dev/null || fail "service stop must preserve state without an interface"
load_wifi_state_file || fail "could not load service stop state"
assert_eq "service stop preserves enabled intent" "1" "$WIFI_STATE_ENABLED"
assert_eq "service stop preserves preference" "Restart Network" "$WIFI_STATE_PREFERRED_SSID"

write_wifi_state_file 1 "Clear Network" || fail "could not prepare clear transition"
if clear_saved_networks >/dev/null 2>&1; then
    fail "clear fixture must stop after its no-interface state commit"
fi
load_wifi_state_file || fail "could not load clear transition"
assert_eq "clear resets enabled intent before runtime work" "0" "$WIFI_STATE_ENABLED"
assert_eq "clear removes the preference before runtime work" "" "$WIFI_STATE_PREFERRED_SSID"

start_source=$(extract_function start_wifi_system)
boot_connect_source=$(extract_function connect_wifi_at_boot)
on_source=$(extract_function turn_on_wifi)
reconnect_source=$(extract_function reconnect_saved_networks)
printf '%s\n' "$start_source" | grep -q 'connect_wifi_at_boot' ||
    fail "start must delegate socket-ready WiFi selection to the boot helper"
printf '%s\n' "$boot_connect_source" | grep -q 'reconnect_saved_networks' ||
    fail "boot helper must use the common reconnect helper"
printf '%s\n' "$boot_connect_source" | grep -q 'HW_WIFI_CFG_PATH' ||
    fail "enabled boot helper must retain the hardware availability marker"
printf '%s\n' "$on_source" | grep -q 'reconnect_saved_networks' ||
    fail "on must use the common reconnect helper"
printf '%s\n' "$on_source" | grep -q 'mark_saved_networks_disabled' ||
    fail "on must disable runtime candidates before starting wpa_supplicant"
on_disable_line=$(printf '%s\n' "$on_source" | grep -n 'mark_saved_networks_disabled' | head -n1 | cut -d: -f1)
on_start_line=$(printf '%s\n' "$on_source" | grep -n '^    start_service$' | head -n1 | cut -d: -f1)
[[ "$on_disable_line" =~ ^[0-9]+$ && "$on_start_line" =~ ^[0-9]+$ &&
    "$on_disable_line" -lt "$on_start_line" ]] ||
    fail "on must disable runtime candidates before starting the service"
printf '%s\n' "$reconnect_source" | grep -q 'select_saved_network_for_reconnect' ||
    fail "reconnect must attempt the preferred saved network first"
printf '%s\n' "$reconnect_source" | grep -q 'enable_network all' ||
    fail "reconnect must retain the all-saved fallback"
if grep -Eq '(^|[^[:alnum:]_])(WIFI_CFG_PATH|LAST_WIFI_FILE|PREVIOUS_WIFI_SAVE|PREVIOUS_WIFI)([^[:alnum:]_]|$)' "$WIFI_SCRIPT"; then
    fail "legacy state aliases must not remain in production code"
fi
printf 'ok - state transitions and preferred-first reconnect wiring are correct\n'

printf 'all unified WiFi state cases passed\n'
