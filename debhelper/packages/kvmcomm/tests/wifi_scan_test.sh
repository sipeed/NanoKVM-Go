#!/usr/bin/env bash

set -u
set -o pipefail

readonly TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly WIFI_SCRIPT="${TEST_DIR}/../rootfs/opt/scripts/wifi.sh"
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT

WIFI_SCAN_WAIT_ATTEMPTS=4
WIFI_SCAN_POLL_INTERVAL_SECONDS=0
DEBUG="false"

fail() {
    printf 'not ok - %s\n' "$*" >&2
    exit 1
}

ok() {
    printf 'ok - %s\n' "$1"
}

assert_eq() {
    local name="$1"
    local want="$2"
    local got="$3"
    [[ "$got" == "$want" ]] || fail "$name: got '$got', want '$want'"
    ok "$name"
}

assert_contains() {
    local name="$1"
    local haystack="$2"
    local needle="$3"
    [[ "$haystack" == *"$needle"* ]] || fail "$name: missing '$needle'"
    ok "$name"
}

assert_nonzero() {
    local name="$1"
    local got="$2"
    [[ "$got" != "0" ]] || fail "$name: expected nonzero status"
    ok "$name"
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

function_sources=""
for function_name in \
    debug_print \
    last_nonempty_wpa_reply_line \
    scan_request_is_accepted \
    scan_request_is_busy \
    read_scan_state \
    scan_results_have_bss_rows \
    scan_results_have_header \
    wait_for_scan_results \
    try_scan \
    try_scan_while_ap_active \
    scan_results_while_ap_active; do
    function_source=$(extract_function "$function_name") ||
        fail "could not extract $function_name"
    function_sources+="${function_source}"$'\n'
done

# Evaluate only the helpers owned by the scan path. The complete script is not sourced.
eval "$function_sources"

SCENARIO_ID=0
MOCK_ROOT=""
WIFI_ALIAS_REGISTRY=""
MOCK_SCAN_REPLY="OK"
MOCK_SCAN_EXIT=0
MOCK_STATUS_EXIT=0
MOCK_PGREP_EXIT=1
MOCK_IW_PLAIN_EXIT=1
MOCK_IW_FORCE_EXIT=1
MOCK_IW_OUTPUT=""
MOCK_IW_FORCE_OUTPUT=""
LAST_STATUS=0
LAST_OUTPUT=""

reset_mock() {
    SCENARIO_ID=$((SCENARIO_ID + 1))
    MOCK_ROOT="${TEST_ROOT}/scenario-${SCENARIO_ID}"
    mkdir -p "$MOCK_ROOT"
    WIFI_ALIAS_REGISTRY="${MOCK_ROOT}/band_aliases.tsv"
    MOCK_SCAN_REPLY="OK"
    MOCK_SCAN_EXIT=0
    MOCK_STATUS_EXIT=0
    MOCK_PGREP_EXIT=1
    MOCK_IW_PLAIN_EXIT=1
    MOCK_IW_FORCE_EXIT=1
    MOCK_IW_OUTPUT=""
    MOCK_IW_FORCE_OUTPUT=""
    printf '0\n' >"${MOCK_ROOT}/status.cursor"
    printf '0\n' >"${MOCK_ROOT}/results.cursor"
    : >"${MOCK_ROOT}/wpa.log"
    : >"${MOCK_ROOT}/clear.log"
    : >"${MOCK_ROOT}/format.log"
    : >"${MOCK_ROOT}/sleep.log"
    : >"${MOCK_ROOT}/iw.log"
    : >"${MOCK_ROOT}/service.log"
    : >"$WIFI_ALIAS_REGISTRY"
}

queue_response() {
    local kind="$1"
    local response="$2"
    local next_file="${MOCK_ROOT}/${kind}.next"
    local index=0

    [[ -r "$next_file" ]] && index=$(<"$next_file")
    printf '%s' "$response" >"${MOCK_ROOT}/${kind}.${index}"
    printf '%s\n' "$response" >"${MOCK_ROOT}/${kind}.last"
    printf '%s\n' "$((index + 1))" >"$next_file"
}

next_response() {
    local kind="$1"
    local cursor_file="${MOCK_ROOT}/${kind}.cursor"
    local index=0
    local response_file

    [[ -r "$cursor_file" ]] && index=$(<"$cursor_file")
    response_file="${MOCK_ROOT}/${kind}.${index}"
    [[ -r "$response_file" ]] || response_file="${MOCK_ROOT}/${kind}.last"
    [[ -r "$response_file" ]] || return 1
    printf '%s\n' "$((index + 1))" >"$cursor_file"
    cat "$response_file"
}

get_wireless_interface() {
    printf 'wlan0\n'
}

start_service() {
    printf 'start_service\n' >>"${MOCK_ROOT}/service.log"
    return 0
}

clear_wifi_alias_registry() {
    printf 'clear\n' >>"${MOCK_ROOT}/clear.log"
    : >"$WIFI_ALIAS_REGISTRY"
}

format_scan_results() {
    local scan_output="$1"

    printf 'format\n' >>"${MOCK_ROOT}/format.log"
    printf '%s' "$scan_output" >"${MOCK_ROOT}/formatted-input"
    if scan_results_have_bss_rows "$scan_output"; then
        printf '[{"ssid":"Test AP"}]\n'
    else
        printf '[]\n'
    fi
}

sleep() {
    printf '%s\n' "$1" >>"${MOCK_ROOT}/sleep.log"
    return 0
}

wpa_cli() {
    local command="${3:-}"

    printf '%s\n' "$command" >>"${MOCK_ROOT}/wpa.log"
    case "$command" in
    scan)
        printf '%s\n' "$MOCK_SCAN_REPLY"
        return "$MOCK_SCAN_EXIT"
        ;;
    status)
        next_response status
        return "$MOCK_STATUS_EXIT"
        ;;
    scan_results)
        next_response results
        ;;
    *)
        return 1
        ;;
    esac
}

pgrep() {
    if [[ "${1:-}" == "hostapd" ]]; then
        return "$MOCK_PGREP_EXIT"
    fi
    return 1
}

iw() {
    printf '%s\n' "$*" >>"${MOCK_ROOT}/iw.log"
    if [[ "$*" == *"ap-force"* ]]; then
        printf '%s' "$MOCK_IW_FORCE_OUTPUT"
        return "$MOCK_IW_FORCE_EXIT"
    fi
    printf '%s' "$MOCK_IW_OUTPUT"
    return "$MOCK_IW_PLAIN_EXIT"
}

run_try_scan() {
    local output_file="${MOCK_ROOT}/output"

    if try_scan >"$output_file"; then
        LAST_STATUS=0
    else
        LAST_STATUS=$?
    fi
    LAST_OUTPUT=$(<"$output_file")
}

scan_header='bssid / frequency / signal level / flags / ssid'
scan_row=$'02:00:00:00:00:01\t2412\t-42\t[WPA2-PSK-CCMP][ESS]\tTest AP'
scan_rows="${scan_header}"$'\n'"${scan_row}"

reset_mock
queue_response status $'scan_state=SCAN_STARTED'
queue_response status $'scan_state=SCAN_COMPLETED'
queue_response results "$scan_rows"
run_try_scan
assert_eq "accepted scan succeeds after completion" "0" "$LAST_STATUS"
assert_contains "accepted scan returns formatted SSID JSON" "$LAST_OUTPUT" '"ssid":"Test AP"'
assert_eq "accepted scan formats only the completed result" "1" "$(wc -l <"${MOCK_ROOT}/format.log")"
scan_request_is_accepted "42" || fail "numeric manual scan id was not accepted"
ok "numeric manual scan id is accepted"

reset_mock
MOCK_SCAN_REPLY="FAIL-BUSY"
queue_response status $'scan_state=SCAN_REQUESTED'
queue_response status $'scan_state=SCAN_COMPLETED'
queue_response results "$scan_rows"
run_try_scan
assert_eq "exit-zero FAIL-BUSY waits for the existing scan" "0" "$LAST_STATUS"
assert_contains "busy scan returns the existing scan SSID" "$LAST_OUTPUT" '"ssid":"Test AP"'
assert_eq "busy response was a successful wpa_cli process call" "0" "$MOCK_SCAN_EXIT"
if grep -qx 'abort_scan' "${MOCK_ROOT}/wpa.log"; then
    fail "busy scan must not abort a scan owned by wpa_supplicant"
fi
ok "busy scan does not abort the existing scan"

for scan_round in 1 2 3; do
    reset_mock
    queue_response status $'scan_state=SCAN_STARTED'
    queue_response status $'scan_state=SCAN_COMPLETED'
    queue_response results "$scan_rows"
    run_try_scan
    assert_eq "consecutive scan ${scan_round} succeeds" "0" "$LAST_STATUS"
    assert_contains "consecutive scan ${scan_round} never returns header-only empty JSON" \
        "$LAST_OUTPUT" '"ssid":"Test AP"'
done
ok "three consecutive scans run with the mocked zero-delay sleep"

reset_mock
MOCK_STATUS_EXIT=1
queue_response results "$scan_header"
queue_response results "$scan_rows"
run_try_scan
assert_eq "unsupported scan_state falls back to bounded result readiness" "0" "$LAST_STATUS"
assert_contains "fallback waits past the header-only result" "$LAST_OUTPUT" '"ssid":"Test AP"'
assert_eq "fallback formats only once after a BSS row appears" "1" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
queue_response status $'scan_state=SCAN_COMPLETED'
queue_response results "$scan_header"
run_try_scan
assert_eq "confirmed empty scan succeeds" "0" "$LAST_STATUS"
assert_eq "confirmed empty scan remains JSON empty array" "[]" "$LAST_OUTPUT"
assert_eq "confirmed empty scan still formats once" "1" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
MOCK_SCAN_EXIT=1
run_try_scan
assert_nonzero "scan command transport failure fails" "$LAST_STATUS"
assert_eq "scan command transport failure retains the existing error payload shape" "[]" "$LAST_OUTPUT"
assert_eq "scan command transport failure does not format a result" "0" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
queue_response status $'scan_state=SCAN_COMPLETED'
queue_response results "FAIL"
run_try_scan
assert_nonzero "invalid completed scan response fails" "$LAST_STATUS"
assert_eq "invalid completed scan response is not formatted as empty success" "[]" "$LAST_OUTPUT"
assert_eq "invalid completed scan response does not call the formatter" "0" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
MOCK_SCAN_REPLY="FAIL"
printf 'stale alias\n' >"$WIFI_ALIAS_REGISTRY"
run_try_scan
assert_nonzero "unrecoverable scan reply fails" "$LAST_STATUS"
assert_eq "unrecoverable reply retains the existing error payload shape" "[]" "$LAST_OUTPUT"
assert_eq "unrecoverable reply clears aliases" "0" "$(wc -c <"$WIFI_ALIAS_REGISTRY")"
assert_eq "unrecoverable reply does not format a result" "0" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
queue_response status $'scan_state=SCAN_ABORTED'
run_try_scan
assert_nonzero "aborted scan fails" "$LAST_STATUS"
assert_eq "aborted scan is not formatted as empty success" "[]" "$LAST_OUTPUT"
assert_eq "aborted scan does not call the formatter" "0" "$(wc -l <"${MOCK_ROOT}/format.log")"

reset_mock
for _ in 1 2 3 4; do
    queue_response status $'scan_state=SCAN_STARTED'
done
run_try_scan
assert_nonzero "scan-state deadline fails" "$LAST_STATUS"
assert_eq "scan-state deadline is not formatted as empty success" "[]" "$LAST_OUTPUT"
assert_eq "scan-state deadline clears aliases" "0" "$(wc -c <"$WIFI_ALIAS_REGISTRY")"

assert_registry_cleared() {
    local name="$1"
    [[ ! -s "$WIFI_ALIAS_REGISTRY" ]] || fail "$name: alias registry was not cleared"
    ok "$name"
}

assert_json_array() {
    local name="$1"
    local output="$2"
    [[ "$output" == "["*"]" ]] || fail "$name: not a JSON array"
    ok "$name"
}

# The AP-active path exercises the production formatter and alias registry,
# so the mocked scan-path doubles above are replaced with the real helpers.
for function_name in \
    wifi_security_from_flags \
    decode_wpa_ssid \
    escape_non_ascii \
    json_escape_wifi_string \
    wifi_band_from_frequency \
    wifi_display_ssid \
    base64_encode_field \
    base64_decode_field \
    commit_wifi_alias_registry \
    clear_wifi_alias_registry \
    format_scan_results; do
    function_source=$(extract_function "$function_name") ||
        fail "could not extract $function_name"
    eval "$function_source"
done
ok "AP scan tests use the production formatter and alias registry"

AP_IW_WPA2=$'BSS 02:00:00:00:00:01(on wlan0)\n\tfreq: 2412\n\tcapability: ESS Privacy ShortPreamble ShortSlotTime (0x0421)\n\tsignal: -42.00 dBm\n\tSSID: ApNet24\n\tRSN:\t * Version: 1\n\t\t * Group cipher: CCMP\n\t\t * Pairwise ciphers: CCMP\n\t\t * Authentication suites: PSK'

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=0
MOCK_IW_OUTPUT="$AP_IW_WPA2"
run_try_scan
assert_eq "AP-active scan succeeds without wpa_supplicant" "0" "$LAST_STATUS"
assert_json_array "AP-active scan returns a JSON array" "$LAST_OUTPUT"
assert_contains "AP-active scan keeps the SSID" "$LAST_OUTPUT" '"ssid":"ApNet24"'
assert_contains "AP-active scan keeps the BSSID" "$LAST_OUTPUT" '"bssid":"02:00:00:00:00:01"'
assert_contains "AP-active scan keeps the signal" "$LAST_OUTPUT" '"signal":-42'
assert_contains "AP-active scan keeps the frequency" "$LAST_OUTPUT" '"frequency":2412'
assert_contains "AP-active scan classifies WPA2" "$LAST_OUTPUT" '"security":"WPA2"'
assert_contains "AP-active single-band name stays literal" "$LAST_OUTPUT" '"ssid":"ApNet24"'
assert_eq "AP-active scan never invokes wpa_cli" "0" "$(wc -c <"${MOCK_ROOT}/wpa.log")"
assert_eq "AP-active scan never starts a service" "0" "$(wc -c <"${MOCK_ROOT}/service.log")"
assert_eq "AP-active plain trigger needs no ap-force fallback" "0" "$(grep -c 'ap-force' "${MOCK_ROOT}/iw.log" || true)"

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=1
MOCK_IW_FORCE_EXIT=0
MOCK_IW_FORCE_OUTPUT="$AP_IW_WPA2"
run_try_scan
assert_eq "AP-active ap-force fallback succeeds" "0" "$LAST_STATUS"
assert_contains "AP-active fallback returns the SSID" "$LAST_OUTPUT" '"ssid":"ApNet24"'
assert_eq "AP-active fallback invokes ap-force exactly once" "1" "$(grep -c 'ap-force' "${MOCK_ROOT}/iw.log" || true)"
assert_eq "AP-active fallback never invokes wpa_cli" "0" "$(wc -c <"${MOCK_ROOT}/wpa.log")"

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=1
MOCK_IW_FORCE_EXIT=1
printf 'stale alias\n' >"$WIFI_ALIAS_REGISTRY"
run_try_scan
assert_nonzero "AP-active double trigger failure fails" "$LAST_STATUS"
assert_eq "AP-active double failure prints the empty payload" "[]" "$LAST_OUTPUT"
assert_registry_cleared "AP-active double failure clears aliases"
assert_eq "AP-active double failure never invokes wpa_cli" "0" "$(wc -c <"${MOCK_ROOT}/wpa.log")"
assert_eq "AP-active double failure never starts a service" "0" "$(wc -c <"${MOCK_ROOT}/service.log")"

run_ap_matrix_case() {
    local ssid="$1"
    local freq="$2"
    local capability="$3"
    local extra="$4"
    local want="$5"

    reset_mock
    MOCK_PGREP_EXIT=0
    MOCK_IW_PLAIN_EXIT=0
    MOCK_IW_OUTPUT=$'BSS 02:00:00:00:00:09(on wlan0)\n\tfreq: '"$freq"$'\n\tcapability: '"$capability"$'\n\tsignal: -50.00 dBm\n\tSSID: '"$ssid"$'\n'"$extra"
    run_try_scan
    assert_eq "AP matrix ${ssid} succeeds" "0" "$LAST_STATUS"
    assert_contains "AP matrix ${ssid} keeps its SSID" "$LAST_OUTPUT" "\"ssid\":\"${ssid}\""
    assert_contains "AP matrix ${ssid} classifies ${want}" "$LAST_OUTPUT" "\"security\":\"${want}\""
}

run_ap_matrix_case "MtxWpa2" "2412" "ESS Privacy (0x0421)" \
    $'\tRSN:\t * Version: 1\n\t\t * Authentication suites: PSK' "WPA2"
run_ap_matrix_case "MtxEap" "5240" "ESS Privacy (0x0011)" \
    $'\tRSN:\t * Version: 1\n\t\t * Authentication suites: IEEE 802.1X' "EAP"
run_ap_matrix_case "MtxWpa3" "5180" "ESS Privacy (0x0011)" \
    $'\tRSN:\t * Version: 1\n\t\t * Authentication suites: PSK SAE' "WPA3"
run_ap_matrix_case "MtxWpa" "2437" "ESS Privacy (0x0011)" \
    $'\tWPA:\t * Version: 1\n\t\t * Authentication suites: PSK' "WPA"
run_ap_matrix_case "MtxWep" "2462" "ESS Privacy (0x0011)" "" "WEP"
run_ap_matrix_case "MtxOpen" "2437" "ESS (0x0001)" "" "OPEN"
# Tolerance for iw variants that emit a lowercase field name for the same line.
run_ap_matrix_case "MtxLowerIw" "2412" "ESS Privacy (0x0421)" \
    $'\tRSN:\t * Version: 1\n\t\t * authentication suites: PSK' "WPA2"

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=0
MOCK_IW_OUTPUT=$'BSS 02:00:00:00:00:0a(on wlan0)\n\tfreq: 2412\n\tcapability: ESS Privacy (0x0421)\n\tsignal: -44.00 dBm\n\tSSID: DualNet\n\tRSN:\t * Version: 1\n\t\t * Authentication suites: PSK\nBSS 02:00:00:00:00:0b(on wlan0)\n\tfreq: 5180\n\tcapability: ESS Privacy (0x0011)\n\tsignal: -57.00 dBm\n\tSSID: DualNet\n\tRSN:\t * Version: 1\n\t\t * Authentication suites: PSK'
run_try_scan
assert_eq "AP-active dual-band scan succeeds" "0" "$LAST_STATUS"
assert_contains "AP-active dual-band decorates the 2.4G name" "$LAST_OUTPUT" '"ssid":"[2.4G] DualNet"'
assert_contains "AP-active dual-band decorates the 5G name" "$LAST_OUTPUT" '"ssid":"[5G] DualNet"'
[[ -s "$WIFI_ALIAS_REGISTRY" ]] || fail "AP-active dual-band writes the alias registry"
ok "AP-active dual-band writes the alias registry"
assert_eq "AP-active alias registry mode is 0600" "600" "$(stat -c %a "$WIFI_ALIAS_REGISTRY")"

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=0
MOCK_IW_OUTPUT=""
printf 'stale alias\n' >"$WIFI_ALIAS_REGISTRY"
run_try_scan
assert_nonzero "AP-active empty dump fails" "$LAST_STATUS"
assert_eq "AP-active empty dump prints the empty payload" "[]" "$LAST_OUTPUT"
assert_registry_cleared "AP-active empty dump clears aliases"

reset_mock
MOCK_PGREP_EXIT=0
MOCK_IW_PLAIN_EXIT=0
MOCK_IW_OUTPUT=$'BSS 02:00:00:00:00:0c(on wlan0)\n\tfreq: 2412\n\tcapability: ESS Privacy (0x0421)\n\tsignal: -60.00 dBm\n\tSSID: '
run_try_scan
assert_nonzero "AP-active hidden-only dump fails" "$LAST_STATUS"
assert_eq "AP-active hidden-only dump prints the empty payload" "[]" "$LAST_OUTPUT"
assert_registry_cleared "AP-active hidden-only dump clears aliases"

printf 'all WiFi scan cases passed\n'
