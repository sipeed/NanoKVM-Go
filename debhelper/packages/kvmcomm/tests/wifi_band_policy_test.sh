#!/usr/bin/env bash

set -u
set -o pipefail

readonly TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly WIFI_SCRIPT="${TEST_DIR}/../rootfs/opt/scripts/wifi.sh"

fail() {
    printf 'not ok - %s\n' "$*" >&2
    exit 1
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
    wifi_band_from_frequency \
    wifi_band_from_marker \
    wifi_display_ssid \
    filter_frequencies_for_band; do
    function_source=$(extract_function "$function_name") ||
        fail "could not extract $function_name"
    function_sources+="${function_source}"$'\n'
done

constant_sources=$(awk '
    /^readonly[[:space:]]+WIFI_BAND_MARKER_(PREFIX|24G|5G)=/ { print }
' "$WIFI_SCRIPT") || fail "could not extract band marker constants"

# Evaluate only the pure helpers and their allowlisted constants. Sourcing the
# device script itself would acquire locks and touch target paths.
eval "${constant_sources}"$'\n'"${function_sources}"

assert_eq() {
    local name="$1"
    local want="$2"
    local got="$3"
    if [[ "$got" != "$want" ]]; then
        fail "$name: got '$got', want '$want'"
    fi
    printf 'ok - %s\n' "$name"
}

assert_eq "2412 MHz is 2.4G" "2.4G" "$(wifi_band_from_frequency 2412)"
assert_eq "2484 MHz is 2.4G" "2.4G" "$(wifi_band_from_frequency 2484)"
assert_eq "5180 MHz is 5G" "5G" "$(wifi_band_from_frequency 5180)"
assert_eq "5895 MHz is 5G" "5G" "$(wifi_band_from_frequency 5895)"

if wifi_band_from_frequency 5955 >/dev/null; then
    fail "6 GHz frequency must not be classified as 5G"
fi
printf 'ok - unsupported frequencies are not assigned to a selectable band\n'

assert_eq \
    "2.4G frequency list filters supported channels" \
    "2412 2437 2484" \
    "$(filter_frequencies_for_band "2.4G" "2412 2437 2484 5180 5320 5955")"
assert_eq \
    "5G frequency list filters supported channels" \
    "5180 5320" \
    "$(filter_frequencies_for_band "5G" "2412 2437 2484 5180 5320 5955")"

assert_eq "2.4G marker is recognized" "2.4G" "$(wifi_band_from_marker nanokvm-band:2.4G)"
assert_eq "quoted 5G marker is recognized" "5G" "$(wifi_band_from_marker '"nanokvm-band:5G"')"
if wifi_band_from_marker nanokvm-band:5G-extra >/dev/null; then
    fail "marker parsing must require an exact namespace value"
fi
printf 'ok - unrelated markers are ignored\n'

assert_eq "2.4G display prefix" "[2.4G] Sipeed" "$(wifi_display_ssid Sipeed 2.4G)"
assert_eq "5G display prefix" "[5G] Sipeed" "$(wifi_display_ssid Sipeed 5G)"
assert_eq "automatic display remains literal" "Sipeed" "$(wifi_display_ssid Sipeed AUTO)"

if grep -Eq '^(should_fallback_to_24g|select_preferred_bss_from_scan_results|prefer_network_bssid)\(\)' "$WIFI_SCRIPT"; then
    fail "automatic RSSI/BSSID cross-band selection helpers must be removed"
fi
printf 'ok - automatic cross-band preference logic is absent\n'

printf 'all WiFi band policy cases passed\n'
