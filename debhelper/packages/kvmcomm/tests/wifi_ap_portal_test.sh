#!/usr/bin/env bash

set -u
set -o pipefail

readonly TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly WIFI_SCRIPT="${TEST_DIR}/../rootfs/opt/scripts/wifi.sh"

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

helper_sources=""
for function_name in ensure_ap_portal_redirect remove_ap_portal_redirect captive_dns_status stop_captive_dns ensure_captive_dns; do
    function_source=$(extract_function "$function_name") ||
        fail "could not extract $function_name"
    helper_sources+="${function_source}"$'\n'
done

# Only the new rule helpers are evaluated. Sourcing the device script would
# acquire locks and touch target networking paths.
debug_print() {
    :
}
eval "$helper_sources"

PORTAL_RULE_PRESENT=false
PORTAL_FAIL_ADD=false
PORTAL_FAIL_DELETE=false
PORTAL_IPTABLES_ACTIONS=()

assert_exact_portal_rule() {
    local -a args=("$@")
    local action="${args[2]:-}"
    local -a expected=(
        -t nat "$action" PREROUTING -i wlan0 -p tcp --dport 80 -j REDIRECT --to-ports 80
    )
    local index

    case "$action" in
    -C | -A | -D) ;;
    *) fail "unexpected iptables action: $action" ;;
    esac

    [[ "${#args[@]}" == "${#expected[@]}" ]] ||
        fail "iptables rule argument count is not exact"
    for index in "${!expected[@]}"; do
        [[ "${args[$index]}" == "${expected[$index]}" ]] ||
            fail "iptables argument $index = '${args[$index]}', want '${expected[$index]}'"
    done
}

iptables() {
    local -a args=("$@")
    local action="${args[2]:-}"

    assert_exact_portal_rule "${args[@]}"
    PORTAL_IPTABLES_ACTIONS+=("$action")

    case "$action" in
    -C)
        [[ "$PORTAL_RULE_PRESENT" == "true" ]]
        ;;
    -A)
        [[ "$PORTAL_FAIL_ADD" == "false" ]] || return 1
        PORTAL_RULE_PRESENT=true
        ;;
    -D)
        [[ "$PORTAL_FAIL_DELETE" == "false" ]] || return 1
        PORTAL_RULE_PRESENT=false
        ;;
    esac
}

reset_actions() {
    PORTAL_IPTABLES_ACTIONS=()
}

reset_actions
ensure_ap_portal_redirect wlan0 || fail "initial ensure must be non-fatal"
assert_eq "initial ensure checks before adding" "-C -A" "${PORTAL_IPTABLES_ACTIONS[*]}"
assert_eq "initial ensure installs the redirect" "true" "$PORTAL_RULE_PRESENT"

reset_actions
ensure_ap_portal_redirect wlan0 || fail "repeat ensure must be non-fatal"
assert_eq "repeat ensure does not duplicate the redirect" "-C" "${PORTAL_IPTABLES_ACTIONS[*]}"
assert_eq "repeat ensure keeps the redirect" "true" "$PORTAL_RULE_PRESENT"

reset_actions
remove_ap_portal_redirect wlan0 || fail "remove must be non-fatal"
assert_eq "remove checks before deleting the exact redirect" "-C -D" "${PORTAL_IPTABLES_ACTIONS[*]}"
assert_eq "remove clears the redirect" "false" "$PORTAL_RULE_PRESENT"

reset_actions
remove_ap_portal_redirect wlan0 || fail "absent remove must be non-fatal"
assert_eq "absent remove does not issue a delete" "-C" "${PORTAL_IPTABLES_ACTIONS[*]}"

PORTAL_FAIL_ADD=true
reset_actions
ensure_ap_portal_redirect wlan0 || fail "failed add must not fail AP startup"
assert_eq "failed add still performs the exact check and add" "-C -A" "${PORTAL_IPTABLES_ACTIONS[*]}"
assert_eq "failed add leaves no redirect" "false" "$PORTAL_RULE_PRESENT"
PORTAL_FAIL_ADD=false

PORTAL_RULE_PRESENT=true
PORTAL_FAIL_DELETE=true
reset_actions
remove_ap_portal_redirect wlan0 || fail "failed delete must not fail AP shutdown"
assert_eq "failed delete still targets the exact redirect" "-C -D" "${PORTAL_IPTABLES_ACTIONS[*]}"
assert_eq "failed delete leaves the existing redirect for a later cleanup" "true" "$PORTAL_RULE_PRESENT"
PORTAL_FAIL_DELETE=false

reset_actions
ensure_ap_portal_redirect "" || fail "empty interface ensure must be non-fatal"
remove_ap_portal_redirect "" || fail "empty interface remove must be non-fatal"
assert_eq "empty interface never invokes iptables" "0" "${#PORTAL_IPTABLES_ACTIONS[@]}"

if [[ "$helper_sources" == *WIFI_PSW* || "$helper_sources" == *Password* ]]; then
    fail "portal rule helpers must not reference or log the AP password"
fi
ok "portal rule helpers do not reference AP credentials"

start_source=$(extract_function start_wifi_ap) || fail "could not extract start_wifi_ap"
stop_source=$(extract_function stop_wifi_ap) || fail "could not extract stop_wifi_ap"
captive_dns_stop_source=$(extract_function stop_captive_dns) || fail "could not extract stop_captive_dns"
failed_start_cleanup_source=$(extract_function cleanup_failed_ap_portal_start) || fail "could not extract failed AP startup cleanup"

assert_eq \
    "AP start wires the portal redirect in both lifecycle paths" \
    "2" \
    "$(grep -cF 'ensure_ap_portal_redirect "$WIFI_IFACE"' <<<"$start_source")"

same_ssid_block=$(sed -n '/if \[\[ "$current_ssid" == "$WIFI_SSID" \]\]; then/,/return 0/p' <<<"$start_source")
[[ "$same_ssid_block" == *'ensure_ap_portal_redirect "$WIFI_IFACE"'* ]] ||
    fail "same-SSID AP start must re-ensure the portal redirect"
ok "same-SSID AP start re-ensures the portal redirect"
[[ "$same_ssid_block" == *'ensure_captive_dns "$WIFI_IFACE"'* ]] ||
    fail "same-SSID AP start must re-ensure captive DNS"
ok "same-SSID AP start re-ensures captive DNS"
[[ "$same_ssid_block" != *udhcpd* ]] ||
    fail "same-SSID AP start must not restart DHCP"
ok "same-SSID AP start preserves DHCP"

pre_switch_stop_dns_line=$(grep -nF 'stop_captive_dns || debug_print "[wifi] AP captive DNS cleanup before AP switch failed"' <<<"$start_source" | cut -d: -f1)
pre_switch_remove_line=$(grep -nF 'remove_ap_portal_redirect "$WIFI_IFACE"' <<<"$start_source" | head -n1 | cut -d: -f1)
disconnect_line=$(grep -nF 'disconnect_wifi "ap-switch"' <<<"$start_source" | cut -d: -f1)
stop_service_line=$(grep -nFx '    stop_service' <<<"$start_source" | cut -d: -f1)
[[ -n "$pre_switch_stop_dns_line" && -n "$pre_switch_remove_line" && -n "$disconnect_line" && -n "$stop_service_line" && "$pre_switch_stop_dns_line" -lt "$disconnect_line" && "$pre_switch_stop_dns_line" -lt "$stop_service_line" && "$pre_switch_remove_line" -lt "$disconnect_line" && "$pre_switch_remove_line" -lt "$stop_service_line" ]] ||
    fail "AP SSID switches must stop DNS and remove the redirect before helpers flush the interface"
ok "AP SSID switches clean portal primitives before interface flush helpers"

portal_line=$(grep -nF 'ensure_ap_portal_redirect "$WIFI_IFACE"' <<<"$start_source" | tail -n1 | cut -d: -f1)
dns_line=$(grep -nF 'if ensure_captive_dns "$WIFI_IFACE"; then' <<<"$start_source" | tail -n1 | cut -d: -f1)
routing_line=$(grep -nF 'iptables -A FORWARD -i "$ETH_IFACE" -o "$WIFI_IFACE" -m state --state RELATED,ESTABLISHED -j ACCEPT' <<<"$start_source" | cut -d: -f1)
hostapd_line=$(grep -nF 'hostapd -B "$HOSTAPD_CONF"' <<<"$start_source" | cut -d: -f1)
udhcpd_line=$(grep -nF 'udhcpd "$UDHCPD_CONF"' <<<"$start_source" | cut -d: -f1)
[[ -n "$portal_line" && -n "$dns_line" && -n "$routing_line" && -n "$hostapd_line" && -n "$udhcpd_line" && "$portal_line" -lt "$dns_line" && "$dns_line" -lt "$routing_line" && "$routing_line" -lt "$hostapd_line" && "$hostapd_line" -lt "$udhcpd_line" ]] ||
    fail "fresh AP start must prepare redirect, DNS, and routing before hostapd/DHCP"
ok "fresh AP start prepares portal primitives before hostapd and DHCP"

[[ "$start_source" == *'local DHCP_DNS="$CAPTIVE_DNS_FALLBACK_DNS"'* ]] ||
    fail "AP startup must begin with a public DNS fallback"
[[ "$start_source" == *'DHCP_DNS="$GATEWAY"'* ]] ||
    fail "AP startup must advertise the gateway only after DNS readiness"
[[ "$start_source" == *'opt     dns     ${DHCP_DNS}'* ]] ||
    fail "DHCP configuration must use the selected DNS value"
ok "DHCP chooses captive DNS only when ready"

[[ "$start_source" == *'AP captive DNS unavailable; retaining public DHCP DNS fallback'* ]] ||
    fail "AP startup must retain public DNS after captive DNS failure"
ok "captive DNS failure does not block AP startup"

[[ "$failed_start_cleanup_source" == *'stop_captive_dns'* && "$failed_start_cleanup_source" == *'remove_ap_portal_redirect "$WIFI_IFACE"'* ]] ||
    fail "failed AP startup must clean captive DNS and the redirect"
ok "failed AP startup cleans portal primitives"
[[ "$failed_start_cleanup_source" == *'pkill hostapd || true'* && "$failed_start_cleanup_source" == *'pkill udhcpd || true'* ]] ||
    fail "failed AP startup must not leave hostapd or DHCP running"
ok "failed AP startup terminates partial AP services"

assert_eq \
    "AP stop wires one portal redirect cleanup" \
    "1" \
    "$(grep -cF 'remove_ap_portal_redirect "$WIFI_IFACE"' <<<"$stop_source")"
remove_line=$(grep -nF 'remove_ap_portal_redirect "$WIFI_IFACE"' <<<"$stop_source" | cut -d: -f1)
stop_dns_line=$(grep -nF 'stop_captive_dns || debug_print "[wifi] AP captive DNS cleanup failed"' <<<"$stop_source" | cut -d: -f1)
flush_line=$(grep -nF 'ip addr flush dev "$WIFI_IFACE"' <<<"$stop_source" | cut -d: -f1)
[[ -n "$remove_line" && -n "$stop_dns_line" && -n "$flush_line" && "$stop_dns_line" -lt "$remove_line" && "$remove_line" -lt "$flush_line" ]] ||
    fail "AP stop must stop DNS and remove the redirect before flushing the wireless interface"
ok "AP stop cleans portal DNS and redirect before interface flush"

[[ "$captive_dns_stop_source" == *'while [ "$retry" -lt "$CAPTIVE_DNS_READY_ATTEMPTS" ]; do'* && "$captive_dns_stop_source" == *'if ! captive_dns_status; then'* ]] ||
    fail "captive DNS stop must wait for asynchronous SIGTERM before a restart"
ok "captive DNS stop waits before rapid AP restart"

printf 'all AP captive portal rule cases passed\n'
