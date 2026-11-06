#!/bin/bash

###
### NanoKVM WiFi Manager — Wireless Network Connection and Hotspot Management Tool
###
### Usage:
###   $0 [command] [parameters]
###
### Commands:
### start                            Load driver and start wpa_supplicant (system service entry)
### stop                             Stop all WiFi services and unload driver (system service exit)
### on                               Turn on WiFi and reconnect to the preferred saved network
### off                              Turn off WiFi (disconnect, keep saved networks)
### connect <SSID> [Password]        Connect to specified WiFi and save configuration
### enterprise <SSID> <PEAP|TLS|TTLS> <identity> [options...]
###                                  Connect to enterprise WiFi (EAP)
### status                           Show current WiFi status
### band                             Show current WiFi band (2.4G/5G/UNKNOWN)
### signal                           Get current WiFi signal strength (dBm)
### scan                             Scan for available WiFi networks (JSON)
### saved                            List saved WiFi networks (JSON)
### boot_enabled                     Show durable Station WiFi enabled intent (true/false)
### preset save <SSID> [Password]    Pre-store one target WiFi without connecting; persists until changed or removed
### preset status                    Show pre-stored WiFi (JSON)
### preset remove                    Remove the pre-stored WiFi
### remove <SSID>                    Remove a saved WiFi network
### clear                            Clear all saved WiFi networks
### ap <SSID> [Password]             Start WiFi hotspot
### ap_off                           Stop WiFi hotspot
### device_count                     Get AP-connected device count
### ap_ip                            Get AP gateway IP address
###
### Options:
###   -h, --help        Display this help information
###   -d, --debug      Enable debug mode (show detailed logs)
###
### Configuration:
###   Temporary Directory: /dev/shm/tmp/wifi
###   Driver Type: nl80211
###   Ethernet Interface: eth0
###   DHCP Client: systemd udhcpc@<iface>.service
### Persistent Configuration: /etc/kvm/wifi_state.conf, /etc/wpa_supplicant.conf
### Static WiFi IP: /boot/wifi.nodhcp (one entry per line: IP/CIDR [Gateway])
###
### Example:
###   $0 on                              # Turn on and auto-reconnect
###   $0 off                             # Disconnect
###   $0 connect MyWiFi 12345678         # Connect and save
###   $0 status                          # Show status
###   $0 signal                          # Get signal strength
###   $0 ap MyHotspot mypassword         # Create a hotspot
###

DEBUG="false"
for arg in "$@"; do
    if [[ "$arg" == "-d" || "$arg" == "--debug" ]]; then
        DEBUG="true"
        break
    fi
done

# Configuration Constants
readonly TMP_DIR="${WIFI_TMP_DIR:-/dev/shm/tmp/wifi}"
readonly WPA_CONF_FILE="${WPA_CONF_FILE:-/etc/wpa_supplicant.conf}"
readonly WPA_PID_FILE="${TMP_DIR}/wpa.pid"
readonly DRIVER="nl80211"
readonly ETH_IFACE="eth0"
readonly SUBNET="192.168.12"
readonly GATEWAY="$SUBNET.1"
readonly HOSTAPD_CONF="$TMP_DIR/hostapd.conf"
readonly UDHCPD_CONF="$TMP_DIR/udhcpd.wlan0.conf"
readonly KVM_CONFIG_DIR="${KVM_CONFIG_DIR:-/etc/kvm}"
readonly WIFI_STATE_FILE="${WIFI_STATE_FILE:-${KVM_CONFIG_DIR}/wifi_state.conf}"
WIFI_STATE_DIR="$(dirname "$WIFI_STATE_FILE")"
readonly WIFI_STATE_DIR
readonly WIFI_PRESET_FILE="${WIFI_PRESET_FILE:-${KVM_CONFIG_DIR}/wifi_preset.conf}"
readonly WIFI_LEGACY_CFG_FILE="${WIFI_LEGACY_CFG_FILE:-${KVM_CONFIG_DIR}/wifi.conf}"
readonly WIFI_LEGACY_LAST_FILE="${WIFI_LEGACY_LAST_FILE:-${KVM_CONFIG_DIR}/last_wifi}"
readonly WIFI_LEGACY_SAVE_FILE="${WIFI_LEGACY_SAVE_FILE:-${KVM_CONFIG_DIR}/wifi_save}"
readonly HW_WIFI_CFG_PATH="${KVM_CONFIG_DIR}/hw/wifi"
readonly LOCK_FILE="${TMP_DIR}/wifi.lock"
readonly LOCK_FD=200
readonly LOCK_WAIT_SECONDS=90
readonly DHCP_WAIT_SECONDS=35
readonly DHCP_POLL_INTERVAL_SECONDS=1
readonly AP_PREVIOUS_STATE_FILE="${KVM_CONFIG_DIR}/wifi_before_ap.conf"
readonly WIFI_ALIAS_REGISTRY="${WIFI_ALIAS_REGISTRY_PATH:-${TMP_DIR}/band_aliases.tsv}"
readonly WIFI_BAND_MARKER_PREFIX="nanokvm-band:"
readonly WIFI_BAND_MARKER_24G="${WIFI_BAND_MARKER_PREFIX}2.4G"
readonly WIFI_BAND_MARKER_5G="${WIFI_BAND_MARKER_PREFIX}5G"
readonly WIFI_PRESET_NETWORK_MARKER="nanokvm-preset"
readonly WIFI_BGSCAN_CONFIG="simple:30:-65:300"
readonly WIFI_CONNECT_WAIT_SECONDS=10
readonly WIFI_SCAN_WAIT_ATTEMPTS="${WIFI_SCAN_WAIT_ATTEMPTS:-24}"
readonly WIFI_SCAN_POLL_INTERVAL_SECONDS="${WIFI_SCAN_POLL_INTERVAL_SECONDS:-0.25}"
readonly CAPTIVE_DNS_BINARY="${CAPTIVE_DNS_BINARY:-/kvmapp/server/NanoKVM-Server}"
readonly CAPTIVE_DNS_FALLBACK_DNS="8.8.8.8"
readonly CAPTIVE_DNS_READY_ATTEMPTS=6
readonly CAPTIVE_DNS_READY_DELAY_SECONDS=1

# The durable Station-state file is deliberately a data format rather than a
# shell fragment.  Do not source it: SSIDs may contain spaces, quotes, and
# non-ASCII bytes, and a malformed control file must fail closed.
WIFI_STATE_LOAD_RESULT="missing"
WIFI_STATE_ENABLED="0"
WIFI_STATE_PREFERRED_SSID=""

# The pre-stored (preset) target WiFi. At most one entry exists; it holds only
# the SSID. Credentials live in the wpa_supplicant config network block. The
# marker persists across successful connections: it stays in effect until the
# user saves a new preset or explicitly removes it.
WIFI_PRESET_SSID=""

base64_encode_field() {
    printf '%s' "$1" | base64 | tr -d '\n'
}

base64_decode_field() {
    printf '%s' "$1" | base64 -d 2>/dev/null
}

wifi_state_ssid_is_valid() {
    local ssid="$1"
    local byte_count

    [[ -n "$ssid" ]] || return 1
    [[ "$ssid" != *$'\n'* && "$ssid" != *$'\r'* ]] || return 1
    byte_count=$(LC_ALL=C printf '%s' "$ssid" | wc -c) || return 1
    [[ "$byte_count" =~ ^[0-9]+$ ]] || return 1
    ((byte_count <= 32)) || return 1
    return 0
}

wifi_state_decode_ssid() {
    local encoded="$1"
    local decoded canonical

    [[ "$encoded" =~ ^[A-Za-z0-9+/]+={0,2}$ ]] || return 1
    (( ${#encoded} % 4 == 0 )) || return 1
    decoded=$(base64_decode_field "$encoded") || return 1
    wifi_state_ssid_is_valid "$decoded" || return 1
    canonical=$(base64_encode_field "$decoded") || return 1
    [[ "$canonical" == "$encoded" ]] || return 1
    printf '%s' "$decoded"
}

wifi_state_reset_loaded_values() {
    WIFI_STATE_LOAD_RESULT="invalid"
    WIFI_STATE_ENABLED="0"
    WIFI_STATE_PREFERRED_SSID=""
}

load_wifi_state_file() {
    wifi_state_reset_loaded_values

    if [[ ! -e "$WIFI_STATE_FILE" && ! -L "$WIFI_STATE_FILE" ]]; then
        WIFI_STATE_LOAD_RESULT="missing"
        return 1
    fi
    [[ -f "$WIFI_STATE_FILE" && ! -L "$WIFI_STATE_FILE" && -r "$WIFI_STATE_FILE" ]] ||
        return 1

    local line key value extra
    local line_count=0
    local version="" enabled="" preferred_b64=""
    local seen_version="false" seen_enabled="false" seen_preferred="false"
    while :; do
        if ! IFS= read -r line; then
            [[ -n "$line" ]] && return 1
            break
        fi
        ((line_count++))
        [[ "$line" != *$'\r'* ]] || return 1
        case "$line" in
        WIFI_STATE_VERSION=*)
            [[ "$seen_version" == "false" ]] || return 1
            version=${line#WIFI_STATE_VERSION=}
            seen_version="true"
            ;;
        WIFI_ENABLED=*)
            [[ "$seen_enabled" == "false" ]] || return 1
            enabled=${line#WIFI_ENABLED=}
            seen_enabled="true"
            ;;
        WIFI_PREFERRED_SSID_B64=*)
            [[ "$seen_preferred" == "false" ]] || return 1
            preferred_b64=${line#WIFI_PREFERRED_SSID_B64=}
            seen_preferred="true"
            ;;
        *) return 1 ;;
        esac
    done <"$WIFI_STATE_FILE"

    [[ "$line_count" == "3" && "$seen_version" == "true" &&
        "$seen_enabled" == "true" && "$seen_preferred" == "true" ]] || return 1
    [[ "$version" == "1" && ( "$enabled" == "0" || "$enabled" == "1" ) ]] ||
        return 1

    local preferred=""
    if [[ -n "$preferred_b64" ]]; then
        preferred=$(wifi_state_decode_ssid "$preferred_b64") || return 1
    fi

    WIFI_STATE_LOAD_RESULT="valid"
    WIFI_STATE_ENABLED="$enabled"
    WIFI_STATE_PREFERRED_SSID="$preferred"
    return 0
}

wifi_state_sync_path() {
    local path="$1"

    sync -f "$path" 2>/dev/null || sync
}

write_wifi_state_file() {
    local enabled="$1"
    local preferred_ssid="${2:-}"
    local encoded="" tmp_file=""

    [[ "$enabled" == "0" || "$enabled" == "1" ]] || return 1
    if [[ -n "$preferred_ssid" ]]; then
        wifi_state_ssid_is_valid "$preferred_ssid" || return 1
        encoded=$(base64_encode_field "$preferred_ssid") || return 1
    fi

    mkdir -p "$WIFI_STATE_DIR" || return 1
    tmp_file=$(mktemp "${WIFI_STATE_DIR}/.wifi_state.conf.XXXXXX") || return 1
    if ! chmod 0600 "$tmp_file" ||
        ! {
            printf 'WIFI_STATE_VERSION=1\n'
            printf 'WIFI_ENABLED=%s\n' "$enabled"
            printf 'WIFI_PREFERRED_SSID_B64=%s\n' "$encoded"
        } >"$tmp_file" ||
        ! wifi_state_sync_path "$tmp_file" ||
        ! mv -f "$tmp_file" "$WIFI_STATE_FILE" ||
        ! chmod 0600 "$WIFI_STATE_FILE" ||
        ! wifi_state_sync_path "$WIFI_STATE_DIR"; then
        rm -f "$tmp_file" || true
        return 1
    fi

    WIFI_STATE_LOAD_RESULT="valid"
    WIFI_STATE_ENABLED="$enabled"
    WIFI_STATE_PREFERRED_SSID="$preferred_ssid"
    return 0
}

write_wifi_preset_file() {
    local ssid="$1"
    local encoded="" tmp_file=""

    wifi_state_ssid_is_valid "$ssid" || return 1
    encoded=$(base64_encode_field "$ssid") || return 1

    mkdir -p "$WIFI_STATE_DIR" || return 1
    tmp_file=$(mktemp "${WIFI_STATE_DIR}/.wifi_preset.conf.XXXXXX") || return 1
    if ! chmod 0600 "$tmp_file" ||
        ! {
            printf 'WIFI_PRESET_VERSION=1\n'
            printf 'WIFI_PRESET_SSID_B64=%s\n' "$encoded"
        } >"$tmp_file" ||
        ! wifi_state_sync_path "$tmp_file" ||
        ! mv -f "$tmp_file" "$WIFI_PRESET_FILE" ||
        ! chmod 0600 "$WIFI_PRESET_FILE" ||
        ! wifi_state_sync_path "$WIFI_STATE_DIR"; then
        rm -f "$tmp_file" || true
        return 1
    fi

    WIFI_PRESET_SSID="$ssid"
    return 0
}

remove_wifi_preset_file() {
    rm -f "$WIFI_PRESET_FILE" || return 1
    WIFI_PRESET_SSID=""
    return 0
}

read_wifi_preset() {
    WIFI_PRESET_SSID=""

    if [[ ! -f "$WIFI_PRESET_FILE" || -L "$WIFI_PRESET_FILE" || ! -r "$WIFI_PRESET_FILE" ]]; then
        return 1
    fi

    local line key value extra
    local line_count=0
    local version="" ssid_b64=""
    local seen_version="false" seen_ssid="false"
    while :; do
        if ! IFS= read -r line; then
            [[ -n "$line" ]] && return 1
            break
        fi
        ((line_count++))
        [[ "$line" != *$'\r'* ]] || return 1
        case "$line" in
        WIFI_PRESET_VERSION=*)
            [[ "$seen_version" == "false" ]] || return 1
            version=${line#WIFI_PRESET_VERSION=}
            seen_version="true"
            ;;
        WIFI_PRESET_SSID_B64=*)
            [[ "$seen_ssid" == "false" ]] || return 1
            ssid_b64=${line#WIFI_PRESET_SSID_B64=}
            seen_ssid="true"
            ;;
        *) return 1 ;;
        esac
    done <"$WIFI_PRESET_FILE"

    [[ "$line_count" == "2" && "$seen_version" == "true" && "$seen_ssid" == "true" ]] || return 1
    [[ "$version" == "1" ]] || return 1

    # An empty SSID is not a valid preset.
    [[ -n "$ssid_b64" ]] || return 1
    local decoded=""
    decoded=$(wifi_state_decode_ssid "$ssid_b64") || return 1

    WIFI_PRESET_SSID="$decoded"
    return 0
}

read_legacy_wifi_cfg_ssid() {
    [[ -f "$WIFI_LEGACY_CFG_FILE" && ! -L "$WIFI_LEGACY_CFG_FILE" &&
        -r "$WIFI_LEGACY_CFG_FILE" ]] || return 1

    local line value=""
    local line_count=0
    while IFS= read -r line || [[ -n "$line" ]]; do
        ((line_count++))
        [[ "$line" == WIFI_CFG_SSID=* ]] || return 1
        value=${line#WIFI_CFG_SSID=}
    done <"$WIFI_LEGACY_CFG_FILE"

    [[ "$line_count" == "1" ]] || return 1
    wifi_state_ssid_is_valid "$value" || return 1
    printf '%s' "$value"
}

read_legacy_wifi_ssid_file() {
    local legacy_file="$1"
    local value

    [[ -f "$legacy_file" && ! -L "$legacy_file" && -r "$legacy_file" ]] ||
        return 1
    value=$(<"$legacy_file")
    wifi_state_ssid_is_valid "$value" || return 1
    printf '%s' "$value"
}

cleanup_legacy_wifi_state_files() {
    rm -f "$WIFI_LEGACY_CFG_FILE" "$WIFI_LEGACY_LAST_FILE" \
        "$WIFI_LEGACY_SAVE_FILE" || return 1
}

migrate_legacy_wifi_state() {
    local enabled="0" preferred="" legacy_ssid=""

    if legacy_ssid=$(read_legacy_wifi_cfg_ssid); then
        enabled="1"
        preferred="$legacy_ssid"
    elif legacy_ssid=$(read_legacy_wifi_ssid_file "$WIFI_LEGACY_LAST_FILE"); then
        preferred="$legacy_ssid"
    elif legacy_ssid=$(read_legacy_wifi_ssid_file "$WIFI_LEGACY_SAVE_FILE"); then
        preferred="$legacy_ssid"
    fi

    write_wifi_state_file "$enabled" "$preferred" || return 1
    cleanup_legacy_wifi_state_files ||
        printf '[wifi] failed to remove migrated legacy WiFi state files\n' >&2
    return 0
}

load_or_migrate_wifi_state() {
    if load_wifi_state_file; then
        # A prior cleanup failure must not restore authority to old markers.
        cleanup_legacy_wifi_state_files ||
            printf '[wifi] failed to remove stale legacy WiFi state files\n' >&2
        return 0
    fi

    [[ "$WIFI_STATE_LOAD_RESULT" == "missing" ]] || return 1
    migrate_legacy_wifi_state
}

persist_connected_wifi_state() {
    local ssid="$1"
    wifi_state_ssid_is_valid "$ssid" || return 1
    write_wifi_state_file "1" "$ssid"
}

print_wifi_boot_enabled() {
    if load_wifi_state_file; then
        if [[ "$WIFI_STATE_ENABLED" == "1" ]]; then
            printf 'true\n'
        else
            printf 'false\n'
        fi
        return 0
    fi

    printf 'false\n'
    return 1
}

# This read-only UI projection deliberately runs before creating directories,
# acquiring the long operation lock, or migrating legacy data. Atomic rename
# means it observes either a complete state file or fails closed.
WIFI_READONLY_COMMAND=""
for wifi_initial_arg in "$@"; do
    case "$wifi_initial_arg" in
    -d | --debug) ;;
    *)
        WIFI_READONLY_COMMAND="$wifi_initial_arg"
        break
        ;;
    esac
done
if [[ "$WIFI_READONLY_COMMAND" == "boot_enabled" ]]; then
    print_wifi_boot_enabled
    exit $?
fi

# Conditionally set locale to avoid warnings on systems without en_US.UTF-8
if locale -a 2>/dev/null | grep -q "en_US.UTF-8"; then
    export LANG=en_US.UTF-8
    export LC_ALL=en_US.UTF-8
    export LC_CTYPE=en_US.UTF-8
else
    export LC_ALL=C
    export LANG=C
    export LC_CTYPE=C
fi

# set -x

debug_print() {
    [[ "${DEBUG}" == "true" ]] && printf "[DEBUG] %s\n" "$*"
}

show_help() {
    sed -rn 's/^### ?//;T;p' "$0" | awk '
        BEGIN { in_example = 0 }
        /^EXAMPLE: / { in_example = 1 }
        in_example && /^###   \S/ { sub("###   ", "  "); print }
        !in_example { print }
    '
}

# Ensure temporary directory exists
mkdir -p "${TMP_DIR}" || {
    debug_print "[wifi] failed to create temporary directory: ${TMP_DIR}" >&2
    exit 1
}

# Ensure persistent config directories exist
mkdir -p "${KVM_CONFIG_DIR}" "${KVM_CONFIG_DIR}/hw" || {
    debug_print "[wifi] failed to create persistent config directories under /etc/kvm" >&2
    exit 1
}

acquire_lock() {
    if ! command -v flock >/dev/null 2>&1; then
        echo "[wifi] flock command not found, cannot acquire lock" >&2
        exit 1
    fi

    exec 200>"$LOCK_FILE"

    local rc=0
    flock -n "$LOCK_FD"
    rc=$?
    if [[ $rc -eq 0 ]]; then
        return 0
    fi
    if [[ $rc -ne 1 ]]; then
        echo "[wifi] failed to acquire lock: flock returned ${rc}" >&2
        exit 1
    fi

    debug_print "[wifi] another instance is running, waiting for lock..."
    local attempt=0
    for ((attempt = 1; attempt <= LOCK_WAIT_SECONDS; attempt++)); do
        flock -n "$LOCK_FD"
        rc=$?
        if [[ $rc -eq 0 ]]; then
            debug_print "[wifi] lock acquired after ${attempt}s wait"
            return 0
        fi
        if [[ $rc -ne 1 ]]; then
            echo "[wifi] failed while waiting for lock: flock returned ${rc}" >&2
            exit 1
        fi
        sleep 1
    done

    echo "[wifi] failed to acquire lock after ${LOCK_WAIT_SECONDS} seconds" >&2
    exit 1
}

release_lock() {
    flock -u "$LOCK_FD" 2>/dev/null || true
}

cleanup_on_exit() {
    local status=$?
    trap - EXIT INT TERM
    sync || true
    release_lock
    exit "$status"
}

trap cleanup_on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

acquire_lock

validate_arguments() {
    local expected=$1
    local actual=$2
    ((actual >= expected)) || {
        debug_print "[wifi] invalid argument count. Expected: ${expected}, Received: ${actual}" >&2
        exit 1
    }
}

get_wireless_interface() {
    local interfaces
    interfaces=$(ip -o link show | grep -o 'wlan[^:]*' | head -1)

    if [[ -z "${interfaces}" ]]; then
        # debug_print "No available wireless interfaces detected" >&2
        return 1
    fi

    echo "${interfaces}"
}

ensure_wpa_conf() {
    if [[ -s "$WPA_CONF_FILE" ]]; then
        # Ensure ctrl_interface uses /var/run for consistency with wifi.sh
        if grep -q '^ctrl_interface=/run/wpa_supplicant' "$WPA_CONF_FILE"; then
            sed -i 's|^ctrl_interface=/run/wpa_supplicant|ctrl_interface=/var/run/wpa_supplicant|' "$WPA_CONF_FILE"
            debug_print "[wifi] updated ctrl_interface path in $WPA_CONF_FILE"
        fi
        chmod 600 "$WPA_CONF_FILE"
        return $?
    fi
    if [[ ! -s "$WPA_CONF_FILE" ]]; then
        mkdir -p "$(dirname "$WPA_CONF_FILE")"
        (
            umask 077
            cat <<EOF >"$WPA_CONF_FILE"
ctrl_interface=/var/run/wpa_supplicant
update_config=1
EOF
        ) || return 1
        chmod 600 "$WPA_CONF_FILE" || return 1
        debug_print "[wifi] created default wpa config at $WPA_CONF_FILE"
    fi
    return 0
}

mark_saved_networks_disabled() {
    if [[ ! -f "$WPA_CONF_FILE" ]]; then
        return 0
    fi

    local tmp_file="${WPA_CONF_FILE}.tmp"
    (umask 077; : >"$tmp_file") || return 1
    chmod 600 "$tmp_file" || return 1
    if ! awk '
        BEGIN { in_network = 0; has_disabled = 0 }
        /^[[:space:]]*network[[:space:]]*=\{/ {
            in_network = 1
            has_disabled = 0
            print
            next
        }
        in_network && /^[[:space:]]*disabled[[:space:]]*=/ {
            if (!has_disabled) {
                print "        disabled=1"
                has_disabled = 1
            }
            next
        }
        in_network && /^[[:space:]]*\}/ {
            if (!has_disabled) {
                print "        disabled=1"
            }
            in_network = 0
            has_disabled = 0
            print
            next
        }
        { print }
    ' "$WPA_CONF_FILE" >"$tmp_file"; then
        rm -f "$tmp_file" || true
        debug_print "[wifi] failed to mark saved networks disabled in $WPA_CONF_FILE" >&2
        return 1
    fi

    mv "$tmp_file" "$WPA_CONF_FILE"
    debug_print "[wifi] marked all saved networks disabled in $WPA_CONF_FILE"
    return 0
}

read_current_sta_ssid() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || return 1

    if [[ ! -S "/var/run/wpa_supplicant/${WIFI_IFACE}" ]]; then
        return 1
    fi

    local state
    state=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^wpa_state=" | cut -d= -f2)
    if [[ "$state" != "COMPLETED" ]]; then
        return 1
    fi

    local raw_ssid ssid
    raw_ssid=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^ssid=" | cut -d= -f2-)
    ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
    [[ -n "$ssid" ]] || return 1

    echo "$ssid"
    return 0
}

save_ap_previous_state() {
    local ssid
    ssid=$(read_current_sta_ssid 2>/dev/null || true)

    if [[ -n "$ssid" ]]; then
        cat <<EOF >"$AP_PREVIOUS_STATE_FILE"
AP_PREVIOUS_STATE=CONNECTED
AP_PREVIOUS_SSID=${ssid}
EOF
        debug_print "[wifi] saved AP previous state: CONNECTED ${ssid}"
        return 0
    fi

    cat <<EOF >"$AP_PREVIOUS_STATE_FILE"
AP_PREVIOUS_STATE=DISABLED
AP_PREVIOUS_SSID=
EOF
    debug_print "[wifi] saved AP previous state: DISABLED"
    return 0
}

clear_ap_previous_state() {
    rm -f "$AP_PREVIOUS_STATE_FILE" || true
}

is_ap_runtime_active() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface 2>/dev/null || true)

    if pgrep hostapd >/dev/null 2>&1; then
        return 0
    fi

    if [[ -n "$WIFI_IFACE" ]] && ip -4 addr show dev "$WIFI_IFACE" 2>/dev/null | grep -qE "inet ${GATEWAY}/24"; then
        return 0
    fi

    if [[ -f "$AP_PREVIOUS_STATE_FILE" ]]; then
        return 0
    fi

    return 1
}

wait_for_ipv4_addr() {
    local iface="$1"

    local waited=0
    while [ "$waited" -lt "$DHCP_WAIT_SECONDS" ]; do
        if ip -4 addr show dev "$iface" 2>/dev/null | grep -qE 'inet '; then
            debug_print "[wifi] IPv4 address acquired on $iface after ${waited}s"
            return 0
        fi
        sleep "$DHCP_POLL_INTERVAL_SECONDS"
        waited=$((waited + DHCP_POLL_INTERVAL_SECONDS))
    done

    if ip -4 addr show dev "$iface" 2>/dev/null | grep -qE 'inet '; then
        debug_print "[wifi] IPv4 address acquired on $iface after ${DHCP_WAIT_SECONDS}s"
        return 0
    fi

    return 1
}

log_dhcp_debug_state() {
    local iface="$1"
    local addr_state wpa_status line

    addr_state=$(ip -br addr show "$iface" 2>/dev/null || true)
    debug_print "[wifi] DHCP debug ip state: ${addr_state:-unavailable}"

    wpa_status=$(wpa_cli -i "$iface" status 2>/dev/null | grep -E '^(wpa_state|ssid|bssid)=' || true)
    if [[ -z "$wpa_status" ]]; then
        debug_print "[wifi] DHCP debug wpa status: unavailable"
        return 0
    fi

    while IFS= read -r line; do
        [[ -n "$line" ]] && debug_print "[wifi] DHCP debug wpa status: $line"
    done <<<"$wpa_status"
}

dhcp_client_running() {
    local iface="$1"

    if systemctl is-active --quiet "udhcpc@${iface}.service" 2>/dev/null; then
        return 0
    fi

    local pid cmdline
    for pid in $(pgrep -x udhcpc 2>/dev/null || true); do
        cmdline=$(tr '\0' ' ' <"/proc/${pid}/cmdline" 2>/dev/null || true)
        if [[ " $cmdline " == *" -i ${iface} "* ]] || [[ " $cmdline " == *" -i${iface} "* ]]; then
            return 0
        fi
    done

    return 1
}

start_dhcp_client() {
    local iface="$1"

    stop_dhcp_client "$iface"
    ip -4 addr flush dev "$iface" 2>/dev/null || true

    if systemctl start "udhcpc@${iface}.service" >/dev/null 2>&1; then
        debug_print "[wifi] started udhcpc@${iface}.service"
        if wait_for_ipv4_addr "$iface"; then
            return 0
        fi
        debug_print "[wifi] DHCP timeout after ${DHCP_WAIT_SECONDS}s on $iface" >&2
        log_dhcp_debug_state "$iface"
        stop_dhcp_client "$iface"
        return 1
    fi

    debug_print "[wifi] systemd udhcpc service unavailable, falling back to direct udhcpc"
    udhcpc -i "$iface" -s /usr/share/udhcpc/default.script -f -p "/run/udhcpc.${iface}.pid" >/dev/null 2>&1 &
    local direct_pid=$!
    debug_print "[wifi] started direct udhcpc on $iface (PID $direct_pid)"
    if wait_for_ipv4_addr "$iface"; then
        return 0
    fi

    debug_print "[wifi] direct udhcpc timeout after ${DHCP_WAIT_SECONDS}s on $iface" >&2
    log_dhcp_debug_state "$iface"
    stop_dhcp_client "$iface"
    return 1
}

stop_dhcp_client() {
    local iface="$1"
    systemctl stop "udhcpc@${iface}.service" >/dev/null 2>&1 || true
    local udhcpc_pids=""
    local pid cmdline
    for pid in $(pgrep -x udhcpc 2>/dev/null || true); do
        cmdline=$(tr '\0' ' ' <"/proc/${pid}/cmdline" 2>/dev/null || true)
        if [[ " $cmdline " == *" -i ${iface} "* ]] || [[ " $cmdline " == *" -i${iface} "* ]]; then
            udhcpc_pids+="${pid}"$'\n'
        fi
    done

    if [ -n "$udhcpc_pids" ]; then
        while IFS= read -r pid; do
            if [ -n "$pid" ]; then
                kill -TERM "$pid" 2>/dev/null && debug_print "[wifi] direct udhcpc (PID $pid) terminated"
            fi
        done <<<"$udhcpc_pids"
        sleep 0.2
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
                debug_print "[wifi] direct udhcpc (PID $pid) force-killed"
            fi
        done <<<"$udhcpc_pids"
    fi
}

try_static_ip() {
    local iface="$1"
    local nodhcp_file="/boot/wifi.nodhcp"

    [[ -f "$nodhcp_file" ]] || return 1

    debug_print "[wifi] found /boot/wifi.nodhcp, attempting static IP assignment"

    while read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" || "$line" == \#* ]] && continue

        # Entries are either IP/CIDR or IP/CIDR followed by an explicit gateway.
        if ! [[ "$line" =~ ^([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/[0-9]{1,2})([[:space:]]+([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}))?$ ]]; then
            debug_print "[wifi] skipping invalid entry in nodhcp file: $line"
            continue
        fi

        local static_ip="${BASH_REMATCH[1]}"
        local ip="${static_ip%/*}"
        local gateway="${BASH_REMATCH[3]:-}"
        debug_print "[wifi] trying static IP: $static_ip${gateway:+, gateway: $gateway}"

        ip addr flush dev "$iface" 2>/dev/null || true

        local ip_free=true
        if command -v arping >/dev/null 2>&1; then
            if arping -D -S 0.0.0.0 -c 2 -w 2 -I "$iface" "$ip" >/dev/null 2>&1; then
                debug_print "[wifi] $ip is already in use on the network, trying next"
                ip_free=false
            fi
        fi

        if [[ "$ip_free" == "true" ]]; then
            ip addr add "$static_ip" dev "$iface" 2>/dev/null || {
                debug_print "[wifi] failed to assign $static_ip to $iface, skipping"
                continue
            }
            if [[ -z "$gateway" ]]; then
                gateway=$(echo "$ip" | awk -F. '{print $1"."$2"."$3".1"}')
            fi
            ip route add default via "$gateway" dev "$iface" 2>/dev/null || true
            debug_print "[wifi] static IP configured: $static_ip, gateway: $gateway"
            return 0
        fi
    done <"$nodhcp_file"

    debug_print "[wifi] no available static IP found in $nodhcp_file, falling back to DHCP"
    return 1
}

# Configure IP for STA mode: try static assignment first, fall back to DHCP.
configure_sta_ip() {
    local iface="$1"
    local reuse_existing="${2:-false}"
    if [[ -f "/boot/wifi.nodhcp" ]]; then
        stop_dhcp_client "$iface"
    fi

    if try_static_ip "$iface"; then
        return 0
    fi

    if [[ "$reuse_existing" == "true" ]] &&
        ip -4 addr show dev "$iface" 2>/dev/null | grep -qE 'inet ' &&
        dhcp_client_running "$iface"; then
        debug_print "[wifi] IPv4 address and DHCP client already active on $iface"
        return 0
    fi

    start_dhcp_client "$iface"
}

start_service() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || exit 1

    ensure_wpa_conf

    if ! pgrep -f "wpa_supplicant.*${WIFI_IFACE}" >/dev/null; then
        debug_print "[wifi] starting wpa_supplicant..."
        mkdir -p /var/run/wpa_supplicant
        wpa_supplicant -D "${DRIVER}" -i "$WIFI_IFACE" -c "$WPA_CONF_FILE" -B -P "$WPA_PID_FILE" >/dev/null 2>&1
        local WPA_PID=$(cat "$WPA_PID_FILE" 2>/dev/null)
        if [[ -n "$WPA_PID" ]]; then
            echo $WPA_PID >/sys/fs/cgroup/cgroup.procs 2>/dev/null || true
        fi
        sleep 1
    fi

    if [[ ! -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        debug_print "[wifi] warning: control socket not found!"
    else
        debug_print "[wifi] wpa_supplicant is running on $WIFI_IFACE"
    fi

    ip link set dev "$WIFI_IFACE" up
}

stop_service() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        debug_print "[wifi] no wireless interface found"
        return 1
    }

    debug_print "[wifi] stopping wpa_supplicant on $WIFI_IFACE..."

    stop_dhcp_client "$WIFI_IFACE"

    if [[ -f "$WPA_PID_FILE" ]]; then
        local pid
        pid=$(<"$WPA_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" && debug_print "[wifi] wpa_supplicant (PID $pid) terminated"
        fi
        rm -f "$WPA_PID_FILE"
    else
        pkill -f "wpa_supplicant.*${WIFI_IFACE}" 2>/dev/null || true
        pkill -f "wpa_supplicant" 2>/dev/null || true
    fi

    ip addr flush dev "$WIFI_IFACE"
}

# Persistent boot hook: a valid pre-stored WiFi is attempted on every service
# start, even when it already equals the durable preferred SSID. The marker is
# never consumed here; it stays in effect until the user saves a new preset or
# removes it. Return 0 on a committed preset connection, 1 when no valid preset
# exists, and 2 when a valid preset was attempted but failed so the caller can
# continue the unchanged durable reconnect path.
try_wifi_preset_boot() {
    local iface="$1"

    if ! read_wifi_preset; then
        return 1
    fi

    local preset_ssid="$WIFI_PRESET_SSID"
    debug_print "[wifi] preset '$preset_ssid' found; trying boot scan"

    local scan_reply scan_reply_line
    if ! scan_reply=$(wpa_cli -i "$iface" scan 2>/dev/null); then
        debug_print "[wifi] preset boot scan request failed; falling back" >&2
        return 2
    fi
    scan_reply_line=$(last_nonempty_wpa_reply_line "$scan_reply")
    if scan_request_is_busy "$scan_reply_line"; then
        debug_print "[wifi] preset boot scan already in progress; waiting for its results"
    elif ! scan_request_is_accepted "$scan_reply_line"; then
        debug_print "[wifi] preset boot scan rejected: ${scan_reply_line:-empty reply}; falling back" >&2
        return 2
    fi

    if ! wait_for_scan_results "$iface" >/dev/null; then
        debug_print "[wifi] preset boot scan did not complete; falling back" >&2
        return 2
    fi

    if ! scan_results_contain_ssid "$iface" "$preset_ssid"; then
        debug_print "[wifi] preset SSID not in range; falling back" >&2
        return 2
    fi

    local network_id
    network_id=$(find_wifi_preset_network_id_by_ssid "$iface" "$preset_ssid" 2>/dev/null) || {
        debug_print "[wifi] preset network block missing; falling back" >&2
        return 2
    }

    # Upgrade legacy preset blocks to an explicit identity. Persist all preset
    # metadata while the candidate is still disabled, then enable it only in
    # the current wpa_supplicant runtime.
    if ! network_has_wifi_preset_marker "$iface" "$network_id"; then
        if ! set_wifi_preset_network_marker "$iface" "$network_id"; then
            debug_print "[wifi] failed to identify legacy preset network; falling back" >&2
            return 2
        fi
    fi

    if ! wpa_cli_ok "$iface" disable_network all ||
        ! wpa_cli_ok "$iface" set_network "$network_id" mesh_fwding 0 ||
        ! apply_sta_roaming_config "$iface" "$network_id" ||
        ! wpa_cli_ok "$iface" save_config ||
        ! wpa_cli_ok "$iface" enable_network "$network_id" ||
        ! wpa_cli_ok "$iface" select_network "$network_id"; then
        debug_print "[wifi] failed to select preset network; falling back" >&2
        wpa_cli -i "$iface" disable_network all >/dev/null 2>&1 || true
        return 2
    fi

    if ! wait_for_ssid_connection "$iface" "$preset_ssid" "$WIFI_CONNECT_WAIT_SECONDS" "AUTO"; then
        debug_print "[wifi] preset connection did not complete; falling back" >&2
        wpa_cli -i "$iface" disable_network all >/dev/null 2>&1 || true
        return 2
    fi

    if ! configure_sta_ip "$iface"; then
        debug_print "[wifi] preset IP configuration failed; falling back" >&2
        wpa_cli -i "$iface" disable_network all >/dev/null 2>&1 || true
        return 2
    fi

    if ! persist_connected_wifi_state "$preset_ssid"; then
        debug_print "[wifi] failed to persist preset WiFi state; falling back" >&2
        disable_wifi_runtime_after_state_failure "$iface"
        return 2
    fi

    debug_print "[wifi] connected to preset WiFi '$preset_ssid' at boot"
    return 0
}

connect_wifi_at_boot() {
    local iface="$1"
    local state_loaded="$2"
    local preset_result

    if try_wifi_preset_boot "$iface"; then
        # The preset link already committed enabled=1 + preferred=preset SSID.
        printf 'true\n' >"$HW_WIFI_CFG_PATH" ||
            debug_print "[wifi] failed to refresh WiFi hardware marker" >&2
        return 0
    else
        preset_result=$?
    fi

    if [[ "$preset_result" == "2" ]]; then
        debug_print "[wifi] preset boot attempt failed; continuing normal reconnect"
    fi

    if [[ "$state_loaded" == "true" && "$WIFI_STATE_ENABLED" == "1" ]]; then
        # This is a hardware-capability marker, not Station preference state.
        # Keep the legacy configured-start side effect without reintroducing a
        # second durable reconnect authority.
        printf 'true\n' >"$HW_WIFI_CFG_PATH" ||
            debug_print "[wifi] failed to refresh WiFi hardware marker" >&2
        if ! reconnect_saved_networks "$iface" "$WIFI_STATE_PREFERRED_SSID"; then
            # Preserve the existing final reconnect behavior. In particular,
            # do not add a new disable/disconnect step after all-saved timeout.
            debug_print "[wifi] boot reconnect did not complete: ${WIFI_RECONNECT_ERROR:-unknown}" >&2
        fi
    else
        debug_print "[wifi] unified state is disabled; leaving all saved networks disabled"
    fi

    return 0
}

start_wifi_system() {
    local state_loaded="true"
    if ! load_or_migrate_wifi_state; then
        state_loaded="false"
        debug_print "[wifi] unified WiFi state is invalid or could not be migrated; starting fail-disabled" >&2
    fi

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }

    # Stop dbus-mode wpa_supplicant if present to avoid conflicts
    if systemctl is-active wpa_supplicant >/dev/null 2>&1; then
        debug_print "[wifi] stopping dbus wpa_supplicant service"
        systemctl stop wpa_supplicant >/dev/null 2>&1 || true
        sleep 1
    fi

    # Clean up any stale per-interface wpa_supplicant instances
    pkill -f "wpa_supplicant.*${WIFI_IFACE}" 2>/dev/null || true
    sleep 0.5

    ensure_wpa_conf
    # Never allow persisted per-network flags to make an implicit association
    # before the durable NanoKVM-wide enabled intent is evaluated.
    mark_saved_networks_disabled || true

    start_service

    local retry=0
    while [ $retry -lt 20 ]; do
        if [[ -S "/var/run/wpa_supplicant/${WIFI_IFACE}" ]]; then
            break
        fi
        sleep 0.5
        ((retry++))
    done

    if [[ ! -S "/var/run/wpa_supplicant/${WIFI_IFACE}" ]]; then
        echo "ERROR: WPA_NOT_RUNNING"
        return 1
    fi

    wpa_cli -i "$WIFI_IFACE" disable_network all >/dev/null 2>&1 || true
    connect_wifi_at_boot "$WIFI_IFACE" "$state_loaded"
}

stop_wifi_system() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface 2>/dev/null || true)

    local state_loaded="true"
    if ! load_or_migrate_wifi_state; then
        state_loaded="false"
        debug_print "[wifi] unified WiFi state is invalid while stopping; preserving fail-disabled state" >&2
    fi

    local status
    if [[ -n "$WIFI_IFACE" ]]; then
        status=$(get_raw_connection_status 2>/dev/null || echo "DISCONNECTED")
    else
        status="ERROR: NO_INTERFACE"
    fi

    if [[ "$state_loaded" == "true" && "$status" == CONNECTED* ]]; then
        local connected_ssid
        connected_ssid=${status#CONNECTED }
        if [[ -n "$connected_ssid" ]] &&
            ! write_wifi_state_file "$WIFI_STATE_ENABLED" "$connected_ssid"; then
            debug_print "[wifi] failed to refresh preferred WiFi while stopping" >&2
        fi
    elif [[ "$status" == AP* ]] || is_ap_runtime_active; then
        debug_print "[wifi] stop snapshot: AP mode; preserving durable Station intent"
    else
        debug_print "[wifi] stop snapshot: disconnected; preserving durable Station intent"
    fi

    stop_wifi_ap
    stop_service

    if [ -n "$WIFI_IFACE" ]; then
        ip link set dev "$WIFI_IFACE" down 2>/dev/null || true
    fi

    # Network disabled flags are runtime safety guards. They do not define the
    # user's durable intent, which remains unchanged across service restarts.
    ensure_wpa_conf
    mark_saved_networks_disabled || true
    clear_ap_previous_state

    echo "OK: WiFi stopped"
    return 0
}

ensure_ap_portal_redirect() {
    local WIFI_IFACE="$1"

    if [[ -z "$WIFI_IFACE" ]]; then
        debug_print "[wifi] AP captive portal HTTP redirect skipped: no wireless interface"
        return 0
    fi

    if iptables -t nat -C PREROUTING -i "$WIFI_IFACE" -p tcp --dport 80 -j REDIRECT --to-ports 80 >/dev/null 2>&1; then
        return 0
    fi

    if ! iptables -t nat -A PREROUTING -i "$WIFI_IFACE" -p tcp --dport 80 -j REDIRECT --to-ports 80 >/dev/null 2>&1; then
        debug_print "[wifi] failed to add AP captive portal HTTP redirect"
    fi

    return 0
}

remove_ap_portal_redirect() {
    local WIFI_IFACE="$1"

    if [[ -z "$WIFI_IFACE" ]]; then
        debug_print "[wifi] AP captive portal HTTP redirect cleanup skipped: no wireless interface"
        return 0
    fi

    if iptables -t nat -C PREROUTING -i "$WIFI_IFACE" -p tcp --dport 80 -j REDIRECT --to-ports 80 >/dev/null 2>&1; then
        if ! iptables -t nat -D PREROUTING -i "$WIFI_IFACE" -p tcp --dport 80 -j REDIRECT --to-ports 80 >/dev/null 2>&1; then
            debug_print "[wifi] failed to remove AP captive portal HTTP redirect"
        fi
    fi

    return 0
}

captive_dns_status() {
    if [[ ! -x "$CAPTIVE_DNS_BINARY" ]]; then
        return 1
    fi

    "$CAPTIVE_DNS_BINARY" captive-dns status >/dev/null 2>&1
}

stop_captive_dns() {
    local retry=0

    if [[ ! -x "$CAPTIVE_DNS_BINARY" ]]; then
        debug_print "[wifi] AP captive DNS stop skipped: server binary unavailable"
        return 1
    fi

    if ! "$CAPTIVE_DNS_BINARY" captive-dns stop >/dev/null 2>&1; then
        debug_print "[wifi] failed to stop AP captive DNS"
        return 1
    fi

    # stop only sends SIGTERM. Wait until status no longer reports the managed
    # listener as alive before a fresh AP start tries to bind the same ports.
    while [ "$retry" -lt "$CAPTIVE_DNS_READY_ATTEMPTS" ]; do
        if ! captive_dns_status; then
            return 0
        fi
        sleep "$CAPTIVE_DNS_READY_DELAY_SECONDS"
        ((retry++))
    done

    debug_print "[wifi] AP captive DNS did not stop before restart"
    return 1
}

ensure_captive_dns() {
    local WIFI_IFACE="$1"
    local retry=0

    if [[ -z "$WIFI_IFACE" ]]; then
        debug_print "[wifi] AP captive DNS skipped: no wireless interface"
        return 1
    fi
    if captive_dns_status; then
        return 0
    fi
    if ! stop_captive_dns; then
        debug_print "[wifi] AP captive DNS has no safe restart path"
        return 1
    fi

    "$CAPTIVE_DNS_BINARY" captive-dns serve >/dev/null 2>&1 &
    while [ "$retry" -lt "$CAPTIVE_DNS_READY_ATTEMPTS" ]; do
        if captive_dns_status; then
            return 0
        fi
        sleep "$CAPTIVE_DNS_READY_DELAY_SECONDS"
        ((retry++))
    done

    stop_captive_dns || true
    debug_print "[wifi] AP captive DNS did not become ready"
    return 1
}

cleanup_failed_ap_portal_start() {
    local WIFI_IFACE="$1"

    # udhcpd can fail after hostapd has already accepted stations. Do not leave
    # a same-SSID retry to mistake that half-started AP for a healthy service.
    pkill hostapd || true
    pkill udhcpd || true
    stop_captive_dns || debug_print "[wifi] failed to stop AP captive DNS during AP startup cleanup"
    remove_ap_portal_redirect "$WIFI_IFACE"
    iptables -t nat -D POSTROUTING -o "$ETH_IFACE" -j MASQUERADE 2>/dev/null || true
    iptables -D FORWARD -i "$WIFI_IFACE" -o "$ETH_IFACE" -j ACCEPT 2>/dev/null || true
    iptables -D FORWARD -i "$ETH_IFACE" -o "$WIFI_IFACE" -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
    sysctl -w net.ipv4.ip_forward=0 >/dev/null || true
    rm -f "$HOSTAPD_CONF" || true
}

start_wifi_ap() {
    local WIFI_IFACE=$(get_wireless_interface)
    local WIFI_SSID="$1"
    local WIFI_PSW="$2"

    if pgrep hostapd >/dev/null; then
        local current_ssid
        current_ssid=$(awk -F= '/^ssid=/{print $2}' "$HOSTAPD_CONF" 2>/dev/null)
        if [[ "$current_ssid" == "$WIFI_SSID" ]]; then
            debug_print "[wifi] AP '$WIFI_SSID' already active on $WIFI_IFACE, skipping reconfiguration"
            ensure_ap_portal_redirect "$WIFI_IFACE"
            ensure_captive_dns "$WIFI_IFACE" || debug_print "[wifi] AP captive DNS recheck failed; preserving active DHCP configuration"
            return 0
        else
            debug_print "[wifi] hostapd running with SSID '$current_ssid', restarting for new SSID '$WIFI_SSID'"
        fi
    fi

    # A previous AP can still own the gateway address while an SSID switch
    # reaches disconnect_wifi/stop_service below. Those helpers flush the
    # interface, so release AP-only listeners and rules before invoking them.
    stop_captive_dns || debug_print "[wifi] AP captive DNS cleanup before AP switch failed"
    remove_ap_portal_redirect "$WIFI_IFACE"

    save_ap_previous_state
    disconnect_wifi "ap-switch"
    stop_service

    local retry=0
    while pgrep -f "wpa_supplicant.*${WIFI_IFACE}" >/dev/null && [ $retry -lt 10 ]; do
        sleep 0.5
        ((retry++))
    done

    if pgrep -f "wpa_supplicant.*${WIFI_IFACE}" >/dev/null; then
        pkill -9 -f "wpa_supplicant.*${WIFI_IFACE}" || true
        sleep 0.5
    fi

    pkill hostapd || true
    pkill udhcpd || true

    debug_print "[wifi] configuring interface: ${WIFI_IFACE}"
    ip addr flush dev "$WIFI_IFACE"
    ip addr add "$GATEWAY/24" dev "$WIFI_IFACE"
    ip link set dev "$WIFI_IFACE" up

    # Install all client-visible AP services before hostapd/udhcpd can allow a
    # newly associated station to obtain a lease and run its connectivity probe.
    ensure_ap_portal_redirect "$WIFI_IFACE"
    local DHCP_DNS="$CAPTIVE_DNS_FALLBACK_DNS"
    if ensure_captive_dns "$WIFI_IFACE"; then
        DHCP_DNS="$GATEWAY"
    else
        debug_print "[wifi] AP captive DNS unavailable; retaining public DHCP DNS fallback"
    fi

    debug_print "[wifi] configuring network routing"
    sysctl -w net.ipv4.ip_forward=1 >/dev/null || true
    iptables -t nat -A POSTROUTING -o "$ETH_IFACE" -j MASQUERADE 2>/dev/null || true
    iptables -A FORWARD -i "$WIFI_IFACE" -o "$ETH_IFACE" -j ACCEPT 2>/dev/null || true
    iptables -A FORWARD -i "$ETH_IFACE" -o "$WIFI_IFACE" -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true

    debug_print "[wifi] generating hostapd configuration"
    if [[ -z "$WIFI_PSW" ]]; then
        cat >"$HOSTAPD_CONF" <<EOF
interface=$WIFI_IFACE
driver=nl80211
ssid=$WIFI_SSID
hw_mode=g
channel=6
ieee80211n=1
wmm_enabled=1
ht_capab=[HT40+][SHORT-GI-20][SHORT-GI-40]
macaddr_acl=0
auth_algs=1
ignore_broadcast_ssid=0
ctrl_interface=/var/run/hostapd
ctrl_interface_group=0
EOF
    else
        cat >"$HOSTAPD_CONF" <<EOF
interface=$WIFI_IFACE
driver=nl80211
ssid=$WIFI_SSID
hw_mode=g
channel=6
ieee80211n=1
wmm_enabled=1
ht_capab=[HT40+][SHORT-GI-20][SHORT-GI-40]
macaddr_acl=0
auth_algs=1
ignore_broadcast_ssid=0
wpa=2
wpa_passphrase=$WIFI_PSW
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
ctrl_interface=/var/run/hostapd
ctrl_interface_group=0
EOF
    fi

    debug_print "[wifi] starting hostapd service"
    if ! hostapd -B "$HOSTAPD_CONF" >/dev/null 2>&1; then
        debug_print "[wifi] failed to start hostapd, rolling back AP previous state"
        cleanup_failed_ap_portal_start "$WIFI_IFACE"
        clear_ap_previous_state
        return 1
    fi

    local ap_retry=0
    while [ $ap_retry -lt 6 ]; do
        if pgrep hostapd >/dev/null 2>&1; then
            debug_print "[wifi] hostapd is running"
            break
        fi
        sleep 0.5
        ((ap_retry++))
    done
    if ! pgrep hostapd >/dev/null 2>&1; then
        debug_print "[wifi] hostapd did not stay running, rolling back AP previous state"
        cleanup_failed_ap_portal_start "$WIFI_IFACE"
        clear_ap_previous_state
        return 1
    fi

    debug_print "[wifi] generating udhcpd configuration"
    cat >"$UDHCPD_CONF" <<EOF
start           ${SUBNET}.100
end             ${SUBNET}.200
interface       ${WIFI_IFACE}
opt     subnet  255.255.255.0
opt     router  ${GATEWAY}
opt     dns     ${DHCP_DNS}
lease_file      ${TMP_DIR}/udhcpd.leases
option  lease   86400
EOF

    debug_print "[wifi] starting udhcpd service"
    if ! udhcpd "$UDHCPD_CONF" >/dev/null 2>&1; then
        debug_print "[wifi] failed to start udhcpd, rolling back AP previous state"
        cleanup_failed_ap_portal_start "$WIFI_IFACE"
        clear_ap_previous_state
        return 1
    fi

    debug_print "[wifi] access point activated - SSID: ${WIFI_SSID}"
    return 0
}

stop_wifi_ap() {
    local WIFI_IFACE=$(get_wireless_interface)

    debug_print "[wifi] terminating ap services"
    stop_captive_dns || debug_print "[wifi] AP captive DNS cleanup failed"
    pkill hostapd || true
    pkill udhcpd || true
    sleep 0.5

    rm -f "$HOSTAPD_CONF" || true

    debug_print "[wifi] cleaning network configuration"
    iptables -t nat -D POSTROUTING -o "$ETH_IFACE" -j MASQUERADE 2>/dev/null || true
    if [[ -n "$WIFI_IFACE" ]]; then
        remove_ap_portal_redirect "$WIFI_IFACE"
        iptables -D FORWARD -i "$WIFI_IFACE" -o "$ETH_IFACE" -j ACCEPT 2>/dev/null || true
        iptables -D FORWARD -i "$ETH_IFACE" -o "$WIFI_IFACE" -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
    fi

    sysctl -w net.ipv4.ip_forward=0 >/dev/null

    if [[ -n "$WIFI_IFACE" ]]; then
        debug_print "[wifi] resetting interface configuration: ${WIFI_IFACE}"
        ip addr flush dev "$WIFI_IFACE"
    else
        debug_print "[wifi] no wireless interface available while stopping AP"
    fi
    clear_ap_previous_state

    debug_print "[wifi] access point deactivated successfully"
}

device_check() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "0"
        return 1
    }

    local count
    count=$(hostapd_cli -i "$WIFI_IFACE" list_sta 2>/dev/null | grep -cE '^[0-9a-f]{2}(:[0-9a-f]{2}){5}$' 2>/dev/null) || count=0
    echo "$count"

    return 0
}

get_ap_ip() {
    echo "$GATEWAY"
}

last_nonempty_wpa_reply_line() {
    local reply="$1"
    local line last_line=""

    while IFS= read -r line || [[ -n "$line" ]]; do
        line="${line%$'\r'}"
        [[ -n "${line//[[:space:]]/}" ]] || continue
        last_line="$line"
    done <<<"$reply"

    printf '%s\n' "$last_line"
}

scan_request_is_accepted() {
    [[ "$1" == "OK" || "$1" =~ ^[0-9]+$ ]]
}

scan_request_is_busy() {
    [[ "$1" == "FAIL-BUSY" ]]
}

read_scan_state() {
    local iface="$1"
    local output key value

    output=$(wpa_cli -i "$iface" status driver 2>/dev/null) || return 1
    while IFS='=' read -r key value; do
        [[ "$key" == "scan_state" ]] || continue
        value="${value%$'\r'}"
        case "$value" in
        SCAN_REQUESTED | SCAN_STARTED | SCAN_COMPLETED | SCAN_ABORTED)
            printf '%s\n' "$value"
            return 0
            ;;
        esac
    done <<<"$output"

    return 1
}

scan_results_have_bss_rows() {
    local scan_output="$1"
    local bssid frequency signal flags ssid rest

    while IFS=$'\t' read -r bssid frequency signal flags ssid rest; do
        [[ "$bssid" =~ ^([[:xdigit:]]{2}:){5}[[:xdigit:]]{2}$ ]] || continue
        [[ "$frequency" =~ ^[0-9]+$ ]] || continue
        [[ "$signal" =~ ^-?[0-9]+$ ]] || continue
        return 0
    done <<<"$scan_output"

    return 1
}

scan_results_have_header() {
    local scan_output="$1"
    local first_line

    IFS= read -r first_line <<<"$scan_output"
    first_line="${first_line%$'\r'}"
    [[ "$first_line" == "bssid / frequency / signal level / flags / ssid" ]]
}

wait_for_scan_results() {
    local iface="$1"
    local wait_attempts="$WIFI_SCAN_WAIT_ATTEMPTS"
    local poll_interval="$WIFI_SCAN_POLL_INTERVAL_SECONDS"
    local attempt=0 scan_state scan_output

    [[ "$wait_attempts" =~ ^[1-9][0-9]*$ ]] || wait_attempts=24

    while ((attempt < wait_attempts)); do
        if scan_state=$(read_scan_state "$iface"); then
            case "$scan_state" in
            SCAN_COMPLETED)
                if ! scan_output=$(wpa_cli -i "$iface" scan_results 2>/dev/null); then
                    debug_print "[wifi] failed to read completed scan results" >&2
                    return 1
                fi
                if [[ -z "$scan_output" ]]; then
                    debug_print "[wifi] completed scan returned no results" >&2
                    return 1
                fi
                if ! scan_results_have_header "$scan_output"; then
                    debug_print "[wifi] completed scan returned an invalid response" >&2
                    return 1
                fi
                printf '%s\n' "$scan_output"
                return 0
                ;;
            SCAN_ABORTED)
                debug_print "[wifi] scan was aborted" >&2
                return 1
                ;;
            SCAN_REQUESTED | SCAN_STARTED)
                ;;
            esac
        else
            if ! scan_output=$(wpa_cli -i "$iface" scan_results 2>/dev/null); then
                debug_print "[wifi] failed to read scan results while waiting" >&2
                return 1
            fi
            if ! scan_results_have_header "$scan_output"; then
                debug_print "[wifi] scan results response is invalid" >&2
                return 1
            fi
            if scan_results_have_bss_rows "$scan_output"; then
                printf '%s\n' "$scan_output"
                return 0
            fi
        fi

        ((attempt += 1))
        if ((attempt < wait_attempts)); then
            sleep "$poll_interval"
        fi
    done

    debug_print "[wifi] scan did not complete before the deadline" >&2
    return 1
}

scan_results_while_ap_active() {
    local iface="$1"
    local iw_output scan_output

    if ! iw_output=$(iw dev "$iface" scan 2>/dev/null); then
        if ! iw_output=$(iw dev "$iface" scan ap-force 2>/dev/null); then
            return 1
        fi
    fi

    scan_output=$(printf '%s\n' "$iw_output" | awk '
        function emit_record(   flags) {
            if (ssid == "" || bssid == "" || freq == "" || sig == "") {
                return
            }
            flags = ""
            if (rsn_psk) {
                flags = flags "[WPA2-PSK-CCMP]"
            }
            if (rsn_eap) {
                flags = flags "[WPA2-EAP-CCMP]"
            }
            if (sae) {
                flags = flags "[SAE]"
            }
            if (wpa_psk) {
                flags = flags "[WPA-PSK-CCMP]"
            }
            if (wpa_eap) {
                flags = flags "[WPA-EAP-CCMP]"
            }
            if (flags == "" && privacy) {
                flags = "[WEP]"
            }
            flags = flags "[ESS]"
            print bssid, freq, sig, flags, ssid
        }
        BEGIN {
            OFS = "\t"
        }
        /^BSS / {
            emit_record()
            bssid = $2
            sub(/[(].*/, "", bssid)
            freq = ""
            sig = ""
            ssid = ""
            rsn_psk = 0
            rsn_eap = 0
            wpa_psk = 0
            wpa_eap = 0
            sae = 0
            privacy = 0
            section = ""
            next
        }
        /^[[:space:]]*freq:/ {
            if ($2 ~ /^[0-9]+$/) {
                freq = $2
            }
            next
        }
        /^[[:space:]]*signal:/ {
            s = $2
            split(s, parts, ".")
            if (parts[1] ~ /^-?[0-9]+$/) {
                sig = parts[1]
            }
            next
        }
        /^[[:space:]]*SSID:/ {
            line = $0
            sub(/^[^:]*:[[:space:]]*/, "", line)
            ssid = line
            next
        }
        /^[[:space:]]*capability:/ {
            if ($0 ~ /Privacy/) {
                privacy = 1
            }
            next
        }
        /^[[:space:]]*RSN:/ {
            section = "rsn"
            next
        }
        /^[[:space:]]*WPA:/ {
            section = "wpa"
            next
        }
        /^[[:space:]]*[A-Za-z][A-Za-z0-9 _\/.-]*:/ {
            section = ""
            next
        }
        /[Aa]uthentication suites:/ {
            if ($0 ~ /SAE/) {
                sae = 1
            }
            if ($0 ~ /PSK/) {
                if (section == "wpa") {
                    wpa_psk = 1
                } else {
                    rsn_psk = 1
                }
            }
            if ($0 ~ /802\.1[Xx]/ || $0 ~ /EAP/) {
                if (section == "wpa") {
                    wpa_eap = 1
                } else {
                    rsn_eap = 1
                }
            }
            next
        }
        END {
            emit_record()
        }
    ') || return 1

    [[ -n "$scan_output" ]] || return 1
    printf '%s\n' "$scan_output"
}

try_scan_while_ap_active() {
    local iface="$1"
    local scan_output

    if ! scan_output=$(scan_results_while_ap_active "$iface"); then
        debug_print "[wifi] AP-mode scan failed" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    if ! format_scan_results "$scan_output"; then
        debug_print "[wifi] failed to format scan results" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    return 0
}

try_scan() {
    local WIFI_IFACE=$(get_wireless_interface)

    if [ -z "$WIFI_IFACE" ]; then
        debug_print "[wifi] no wireless interface found" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    if pgrep hostapd >/dev/null 2>&1; then
        try_scan_while_ap_active "$WIFI_IFACE"
        return $?
    fi

    start_service
    debug_print "[wifi] scanning for available networks..."

    local scan_reply scan_reply_line scan_output
    if ! scan_reply=$(wpa_cli -i "${WIFI_IFACE}" scan 2>/dev/null); then
        debug_print "[wifi] scan command transport failed" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    scan_reply_line=$(last_nonempty_wpa_reply_line "$scan_reply")
    if scan_request_is_accepted "$scan_reply_line"; then
        :
    elif scan_request_is_busy "$scan_reply_line"; then
        debug_print "[wifi] scan already in progress; waiting for its results"
    else
        debug_print "[wifi] scan command was rejected: ${scan_reply_line:-empty reply}" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    if ! scan_output=$(wait_for_scan_results "$WIFI_IFACE"); then
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    if ! format_scan_results "$scan_output"; then
        debug_print "[wifi] failed to format scan results" >&2
        clear_wifi_alias_registry
        echo "[]"
        return 1
    fi

    return 0
}

escape_non_ascii() {
    local input="$1"
    local escaped=""
    local i c ord b

    for ((i = 0; i < ${#input}; i++)); do
        c="${input:i:1}"
        LC_CTYPE=C
        ord=$(printf "%d" "'$c" 2>/dev/null || ord=0)
        if [ "$ord" -ge 32 ] && [ "$ord" -le 126 ]; then
            if [ "$c" = "\\" ]; then
                escaped+="\\\\"
            else
                escaped+="$c"
            fi
        else
            while read -r b; do
                escaped+="\\x$b"
            done < <(echo -n "$c" | xxd -p -c1)
        fi
    done

    printf '%s' "$escaped"
}

unescape_non_ascii() {
    local input="$1"
    local result=""
    local i=0

    while [ $i -lt ${#input} ]; do
        local c="${input:i:1}"

        if [ "$c" = "\\" ]; then
            local next="${input:i+1:1}"

            if [ "$next" = "\\" ]; then
                result+="\\\\"
                ((i += 2))
            elif [ "$next" = "x" ]; then
                local hex="${input:i+2:2}"
                if [[ "$hex" =~ ^[0-9a-fA-F]{2}$ ]]; then
                    result+=$(printf "\\x$hex")
                    ((i += 4))
                else
                    result+="$c"
                    ((i += 1))
                fi
            else
                result+="$c"
                ((i += 1))
            fi
        else
            result+="$c"
            ((i += 1))
        fi
    done

    printf '%b' "$result"
}

decode_non_ascii_x_escape_run() {
    local hex_run="$1"
    local offset hex value
    local has_non_ascii="false"

    for ((offset = 0; offset < ${#hex_run}; offset += 2)); do
        hex="${hex_run:offset:2}"
        value=$((16#$hex))
        if [ "$value" -ge 128 ]; then
            has_non_ascii="true"
            break
        fi
    done

    if [[ "$has_non_ascii" != "true" ]]; then
        return 1
    fi

    printf '%s' "$hex_run" | xxd -r -p
}

normalize_ssid_arg() {
    local input="$1"
    local output=""
    local i=0
    local c next hex start hex_run decoded

    while [ $i -lt ${#input} ]; do
        c="${input:i:1}"
        next="${input:i+1:1}"
        hex="${input:i+2:2}"

        if [[ "$c" == "\\" && "$next" == "x" && "$hex" =~ ^[0-9a-fA-F]{2}$ ]]; then
            start=$i
            hex_run=""
            while [ $((i + 3)) -lt ${#input} ]; do
                hex="${input:i+2:2}"
                if [[ "${input:i:2}" != "\\x" || ! "$hex" =~ ^[0-9a-fA-F]{2}$ ]]; then
                    break
                fi
                hex_run+="$hex"
                ((i += 4))
            done

            if decoded=$(decode_non_ascii_x_escape_run "$hex_run"); then
                output+="$decoded"
            else
                output+="${input:start:i-start}"
            fi
        else
            output+="$c"
            ((i += 1))
        fi
    done

    printf '%s' "$output"
}

wifi_band_from_frequency() {
    local frequency="$1"

    if ! [[ "$frequency" =~ ^[0-9]+$ ]]; then
        echo "UNKNOWN"
        return 1
    fi

    if ((frequency >= 2400 && frequency < 2500)); then
        echo "2.4G"
        return 0
    fi

    if ((frequency >= 4900 && frequency < 5900)); then
        echo "5G"
        return 0
    fi

    echo "UNKNOWN"
    return 1
}

decode_wpa_ssid() {
    local input="$1"
    local output=""
    local i=0
    local c next hex decoded

    while ((i < ${#input})); do
        c="${input:i:1}"
        if [[ "$c" != "\\" ]]; then
            output+="$c"
            ((i++))
            continue
        fi

        next="${input:i+1:1}"
        if [[ "$next" == "\\" ]]; then
            output+="\\"
            ((i += 2))
            continue
        fi

        hex="${input:i+2:2}"
        if [[ "$next" == "x" && "$hex" =~ ^[0-9a-fA-F]{2}$ ]]; then
            if [[ "$hex" == "00" ]]; then
                return 1
            fi
            printf -v decoded '%b' "\\x${hex}"
            output+="$decoded"
            ((i += 4))
            continue
        fi

        output+="\\"
        ((i++))
    done

    printf '%s' "$output"
}

json_escape_wifi_string() {
    local escaped
    escaped=$(escape_non_ascii "$1")
    escaped=${escaped//\"/\\\"}
    printf '%s' "$escaped"
}

wifi_security_from_flags() {
    local flags="$1"

    if [[ "$flags" == *WPA2-EAP* || "$flags" == *WPA-EAP* || "$flags" == *EAP* ]]; then
        echo "EAP"
    elif [[ "$flags" == *SAE* ]]; then
        echo "WPA3"
    elif [[ "$flags" == *WPA2-PSK* || "$flags" == *WPA2* ]]; then
        echo "WPA2"
    elif [[ "$flags" == *WPA-PSK* || "$flags" == *WPA* ]]; then
        echo "WPA"
    elif [[ "$flags" == *WEP* ]]; then
        echo "WEP"
    else
        echo "OPEN"
    fi
}

wifi_display_ssid() {
    local ssid="$1"
    local band="$2"

    case "$band" in
    "2.4G" | "5G") printf '[%s] %s' "$band" "$ssid" ;;
    *) printf '%s' "$ssid" ;;
    esac
}

wifi_band_from_marker() {
    local marker="$1"
    marker=${marker#\"}
    marker=${marker%\"}

    case "$marker" in
    "$WIFI_BAND_MARKER_24G") echo "2.4G" ;;
    "$WIFI_BAND_MARKER_5G") echo "5G" ;;
    *) return 1 ;;
    esac
}

filter_frequencies_for_band() {
    local band="$1"
    local frequencies="$2"
    local frequency current_band
    local output=""

    for frequency in $frequencies; do
        [[ "$frequency" =~ ^[0-9]+$ ]] || continue
        current_band=$(wifi_band_from_frequency "$frequency") || true
        [[ "$current_band" == "$band" ]] || continue
        if [[ -n "$output" ]]; then
            output+=" "
        fi
        output+="$frequency"
    done

    [[ -n "$output" ]] || return 1
    printf '%s\n' "$output"
}

get_supported_frequencies() {
    local iface="$1"

    if [[ -n "${WIFI_SUPPORTED_FREQUENCIES:-}" ]]; then
        printf '%s\n' "$WIFI_SUPPORTED_FREQUENCIES"
        return 0
    fi

    local phy output
    phy=$(iw dev "$iface" info 2>/dev/null | awk '$1 == "wiphy" {print "phy" $2; exit}')
    if [[ -n "$phy" ]]; then
        output=$(iw phy "$phy" info 2>/dev/null)
    else
        output=$(iw list 2>/dev/null)
    fi

    printf '%s\n' "$output" |
        awk '/\*[[:space:]]+[0-9]+[[:space:]]+MHz/ && $0 !~ /\(disabled\)/ {print $2}' |
        sort -n -u |
        paste -sd ' ' -
}

build_freq_list_for_band() {
    local iface="$1"
    local band="$2"
    local supported

    supported=$(get_supported_frequencies "$iface") || return 1
    filter_frequencies_for_band "$band" "$supported"
}

clear_wifi_alias_registry() {
    rm -f "$WIFI_ALIAS_REGISTRY" 2>/dev/null || true
}

commit_wifi_alias_registry() {
    local staged_registry="$1"

    chmod 0600 "$staged_registry" || return 1
    mv -f "$staged_registry" "$WIFI_ALIAS_REGISTRY" || return 1
    chmod 0600 "$WIFI_ALIAS_REGISTRY" || return 1
}

resolve_wifi_alias() {
    local display_ssid="$1"
    [[ -f "$WIFI_ALIAS_REGISTRY" && ! -L "$WIFI_ALIAS_REGISTRY" ]] || return 1

    local display_b64 alias_b64 real_b64 band extra decoded
    display_b64=$(base64_encode_field "$display_ssid") || return 1

    while IFS=$'\t' read -r alias_b64 real_b64 band extra; do
        [[ -z "$extra" && "$alias_b64" == "$display_b64" ]] || continue
        [[ "$band" == "2.4G" || "$band" == "5G" ]] || return 1
        decoded=$(base64_decode_field "$real_b64") || return 1
        [[ -n "$decoded" ]] || return 1
        printf '%s\t%s\n' "$real_b64" "$band"
        return 0
    done <"$WIFI_ALIAS_REGISTRY"

    return 1
}

resolve_connection_target() {
    local input="$1"
    local mode="${2:-user}"
    local display_ssid real_ssid real_b64 display_b64 selected_band policy_mode resolved

    display_ssid=$(normalize_ssid_arg "$input")
    real_ssid="$display_ssid"
    selected_band="AUTO"
    policy_mode="AUTO"

    if [[ "$mode" == "saved" ]]; then
        policy_mode="PRESERVE"
    elif resolved=$(resolve_wifi_alias "$display_ssid"); then
        IFS=$'\t' read -r real_b64 selected_band <<<"$resolved"
        real_ssid=$(base64_decode_field "$real_b64") || return 1
        policy_mode="EXPLICIT"
    fi

    real_b64=$(base64_encode_field "$real_ssid") || return 1
    display_b64=$(base64_encode_field "$display_ssid") || return 1
    printf '%s\t%s\t%s\t%s\n' "$real_b64" "$selected_band" "$display_b64" "$policy_mode"
}

format_scan_results() {
    local scan_output="$1"
    local registry_dir records_tmp sorted_tmp registry_tmp
    registry_dir=$(dirname "$WIFI_ALIAS_REGISTRY")
    mkdir -p "$registry_dir" || return 1

    records_tmp=$(mktemp "${registry_dir}/scan-records.XXXXXX") || return 1
    sorted_tmp=$(mktemp "${registry_dir}/scan-sorted.XXXXXX") || {
        rm -f "$records_tmp"
        return 1
    }
    registry_tmp=$(mktemp "${WIFI_ALIAS_REGISTRY}.tmp.XXXXXX") || {
        rm -f "$records_tmp" "$sorted_tmp"
        return 1
    }
    chmod 0600 "$records_tmp" "$sorted_tmp" "$registry_tmp" || {
        rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
        return 1
    }

    local -A ssids=()
    local -A has_24g=()
    local -A has_5g=()
    local -A best_signal=()
    local -A best_bssid=()
    local -A best_frequency=()
    local -A best_security=()
    local bssid frequency signal flags raw_ssid rest real_ssid ssid_b64 band pair security

    while IFS=$'\t' read -r bssid frequency signal flags raw_ssid rest; do
        [[ "$frequency" =~ ^[0-9]+$ && "$signal" =~ ^-?[0-9]+$ ]] || continue
        if [[ -n "$rest" ]]; then
            raw_ssid+=$'\t'"$rest"
        fi
        [[ -n "$raw_ssid" ]] || continue

        real_ssid=$(decode_wpa_ssid "$raw_ssid") || continue
        [[ -n "$real_ssid" ]] || continue
        ssid_b64=$(base64_encode_field "$real_ssid") || continue
        [[ -n "$ssid_b64" ]] || continue

        band=$(wifi_band_from_frequency "$frequency") || band="OTHER"
        pair="${ssid_b64}|${band}"
        security=$(wifi_security_from_flags "$flags")

        ssids["$ssid_b64"]=1
        [[ "$band" == "2.4G" ]] && has_24g["$ssid_b64"]=1
        [[ "$band" == "5G" ]] && has_5g["$ssid_b64"]=1

        if [[ -z "${best_signal[$pair]+x}" || $signal -gt ${best_signal[$pair]} ]]; then
            best_signal["$pair"]="$signal"
            best_bssid["$pair"]="$bssid"
            best_frequency["$pair"]="$frequency"
            best_security["$pair"]="$security"
        fi
    done <<<"$scan_output"

    local -A used_names=()
    local key candidate candidate_b64 suffix display_b64
    for key in "${!ssids[@]}"; do
        used_names["$key"]=1
    done

    local -a sorted_ssids=()
    if ((${#ssids[@]} > 0)); then
        mapfile -t sorted_ssids < <(printf '%s\n' "${!ssids[@]}" | LC_ALL=C sort)
    fi

    for key in "${sorted_ssids[@]}"; do
        if [[ -n "${has_24g[$key]+x}" && -n "${has_5g[$key]+x}" ]]; then
            real_ssid=$(base64_decode_field "$key") || {
                rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
                return 1
            }
            for band in "2.4G" "5G"; do
                pair="${key}|${band}"
                candidate=$(wifi_display_ssid "$real_ssid" "$band")
                candidate_b64=$(base64_encode_field "$candidate") || {
                    rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
                    return 1
                }
                suffix=1
                while [[ -n "${used_names[$candidate_b64]+x}" ]]; do
                    candidate="[${band}] [${suffix}] ${real_ssid}"
                    candidate_b64=$(base64_encode_field "$candidate") || {
                        rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
                        return 1
                    }
                    ((suffix++))
                done
                used_names["$candidate_b64"]=1
                printf '%s\t%s\t%s\n' "$candidate_b64" "$key" "$band" >>"$registry_tmp" || {
                    rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
                    return 1
                }
                printf '%s\t%s\t%s\t%s\t%s\n' \
                    "${best_signal[$pair]}" "$candidate_b64" "${best_bssid[$pair]}" \
                    "${best_frequency[$pair]}" "${best_security[$pair]}" >>"$records_tmp" || {
                    rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
                    return 1
                }
            done
            continue
        fi

        local best_pair=""
        for band in "2.4G" "5G" "OTHER"; do
            pair="${key}|${band}"
            [[ -n "${best_signal[$pair]+x}" ]] || continue
            if [[ -z "$best_pair" || ${best_signal[$pair]} -gt ${best_signal[$best_pair]} ]]; then
                best_pair="$pair"
            fi
        done
        [[ -n "$best_pair" ]] || continue
        printf '%s\t%s\t%s\t%s\t%s\n' \
            "${best_signal[$best_pair]}" "$key" "${best_bssid[$best_pair]}" \
            "${best_frequency[$best_pair]}" "${best_security[$best_pair]}" >>"$records_tmp" || {
            rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
            return 1
        }
    done

    if ! LC_ALL=C sort -t $'\t' -k1,1nr -k2,2 "$records_tmp" >"$sorted_tmp"; then
        rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
        return 1
    fi
    if ! commit_wifi_alias_registry "$registry_tmp"; then
        rm -f "$records_tmp" "$sorted_tmp" "$registry_tmp"
        return 1
    fi

    local first=1 escaped_display
    printf '[\n'
    while IFS=$'\t' read -r signal display_b64 bssid frequency security; do
        [[ -n "$display_b64" ]] || continue
        real_ssid=$(base64_decode_field "$display_b64") || continue
        escaped_display=$(json_escape_wifi_string "$real_ssid")
        if ((first == 0)); then
            printf ',\n'
        fi
        first=0
        printf '  {"ssid":"%s","bssid":"%s","signal":%d,"frequency":%d,"security":"%s"}' \
            "$escaped_display" "$bssid" "$signal" "$frequency" "$security"
    done <"$sorted_tmp"
    printf '\n]\n'

    rm -f "$records_tmp" "$sorted_tmp"
    return 0
}

scan_results_contain_ssid() {
    local iface="$1"
    local target_ssid="$2"
    local scan_output bssid frequency signal flags raw_ssid rest decoded

    scan_output=$(wpa_cli -i "$iface" scan_results 2>/dev/null) || return 1
    while IFS=$'\t' read -r bssid frequency signal flags raw_ssid rest; do
        [[ "$frequency" =~ ^[0-9]+$ ]] || continue
        if [[ -n "$rest" ]]; then
            raw_ssid+=$'\t'"$rest"
        fi
        decoded=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
        [[ "$decoded" == "$target_ssid" ]] && return 0
    done <<<"$scan_output"

    return 1
}

wpa_cli_ok() {
    local iface="$1"
    shift

    if [[ "${1:-}" == "save_config" ]]; then
        # wpa_supplicant replaces the config through this temporary file.
        # Pre-create it privately even when the running daemon has umask 022:
        # SAE credentials must retain the original password, not just a PSK.
        chmod 600 "$WPA_CONF_FILE" || return 1
        (umask 077; : >>"${WPA_CONF_FILE}.tmp") || return 1
        chmod 600 "${WPA_CONF_FILE}.tmp" || return 1
    fi

    local output last_line
    output=$(wpa_cli -i "$iface" "$@" 2>/dev/null) || return 1
    last_line=$(printf '%s\n' "$output" | tail -n1)
    [[ "$last_line" == "OK" ]]
}

get_network_value() {
    local iface="$1"
    local network_id="$2"
    local field="$3"
    local output

    output=$(wpa_cli -i "$iface" get_network "$network_id" "$field" 2>/dev/null) || return 1
    output=$(printf '%s\n' "$output" | tail -n1)
    [[ "$output" != "FAIL" ]] || return 1
    printf '%s\n' "$output"
}

ssid_to_hex() {
    local ssid="$1"
    local encoded

    encoded=$(printf '%s' "$ssid" | xxd -p -c 256 | tr -d '\n') || return 1
    [[ -n "$encoded" ]] || return 1
    printf '%s\n' "$encoded"
}

configure_wifi_credentials() {
    local iface="$1"
    local network_id="$2"
    local ssid="$3"
    local password="$4"

    if [[ -z "$password" ]]; then
        wpa_cli_ok "$iface" set_network "$network_id" key_mgmt NONE
        return $?
    fi

    local psk password_hex password_bytes key_mgmt="SAE" pmf=2
    # Hex is wpa_supplicant's byte-preserving string syntax; quotes, spaces,
    # backslashes, and non-ASCII password bytes need no shell/config escaping.
    password_hex=$(ssid_to_hex "$password") || return 1

    # SAE uses the original password without WPA2's 8..63-byte passphrase
    # restriction. Derive a PSK only when WPA2 can use it; count bytes even
    # when the shell's locale counts UTF-8 characters.
    password_bytes=$((${#password_hex} / 2))
    if ((password_bytes >= 8 && password_bytes <= 63)); then
        psk=$(wpa_passphrase "$ssid" "$password" 2>/dev/null | awk -F= '/^[ \t]*psk=[^#]/{print $2; exit}')
        [[ "$psk" =~ ^[[:xdigit:]]{64}$ ]] || return 1
        wpa_cli_ok "$iface" set_network "$network_id" psk "$psk" || return 1
        key_mgmt="WPA-PSK SAE"
        pmf=1
    fi

    # Pure SAE requires PMF; dual-mode keeps WPA2 compatibility. Support both
    # legacy and hash-to-element SAE password derivation in either case.
    wpa_cli_ok "$iface" set_network "$network_id" sae_password "$password_hex" &&
        wpa_cli_ok "$iface" set_network "$network_id" key_mgmt "$key_mgmt" &&
        wpa_cli_ok "$iface" set_network "$network_id" ieee80211w "$pmf" &&
        wpa_cli_ok "$iface" set sae_pwe 2
}

find_network_ids_by_ssid() {
    local iface="$1"
    local target_ssid="$2"
    local output network_id raw_ssid bssid flags rest decoded

    output=$(wpa_cli -i "$iface" list_networks 2>/dev/null) || return 1
    while IFS=$'\t' read -r network_id raw_ssid bssid flags rest; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        decoded=$(decode_wpa_ssid "$raw_ssid") || continue
        [[ "$decoded" == "$target_ssid" ]] || continue
        printf '%s\n' "$network_id"
    done <<<"$output"
}

normalize_wifi_network_marker() {
    local marker="$1"
    marker=${marker#\"}
    marker=${marker%\"}
    printf '%s\n' "$marker"
}

network_has_wifi_preset_marker() {
    local iface="$1"
    local network_id="$2"
    local marker

    marker=$(get_network_value "$iface" "$network_id" id_str) || return 1
    marker=$(normalize_wifi_network_marker "$marker")
    [[ "$marker" == "$WIFI_PRESET_NETWORK_MARKER" ]]
}

set_wifi_preset_network_marker() {
    local iface="$1"
    local network_id="$2"
    wpa_cli_ok "$iface" set_network "$network_id" id_str \
        "\"$WIFI_PRESET_NETWORK_MARKER\""
}

clear_wifi_preset_network_marker() {
    local iface="$1"
    local network_id="$2"
    wpa_cli_ok "$iface" set_network "$network_id" id_str NULL
}

find_wifi_preset_network_ids() {
    local iface="$1"
    local output network_id raw_ssid bssid flags rest

    output=$(wpa_cli -i "$iface" list_networks 2>/dev/null) || return 1
    while IFS=$'\t' read -r network_id raw_ssid bssid flags rest; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        network_has_wifi_preset_marker "$iface" "$network_id" || continue
        printf '%s\n' "$network_id"
    done <<<"$output"
}

find_wifi_preset_network_id_by_ssid() {
    local iface="$1"
    local target_ssid="$2"
    local matching_ids network_id legacy_id="" preset_id=""

    matching_ids=$(find_network_ids_by_ssid "$iface" "$target_ssid" 2>/dev/null) || return 1
    while read -r network_id; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        [[ -n "$legacy_id" ]] || legacy_id="$network_id"
        if network_has_wifi_preset_marker "$iface" "$network_id"; then
            # A successful save leaves exactly one active preset marker. Use
            # the last tagged entry if an interrupted older version left more.
            preset_id="$network_id"
        fi
    done <<<"$matching_ids"

    if [[ -n "$preset_id" ]]; then
        printf '%s\n' "$preset_id"
    elif [[ -n "$legacy_id" ]]; then
        printf '%s\n' "$legacy_id"
    else
        return 1
    fi
}

current_wifi_network_id() {
    local iface="$1"
    local status_output state network_id

    status_output=$(wpa_cli -i "$iface" status 2>/dev/null) || return 1
    state=$(printf '%s\n' "$status_output" | awk -F= '/^wpa_state=/{print $2; exit}')
    [[ "$state" == "COMPLETED" ]] || return 1
    network_id=$(printf '%s\n' "$status_output" | awk -F= '/^id=/{print $2; exit}')
    [[ "$network_id" =~ ^[0-9]+$ ]] || return 1
    printf '%s\n' "$network_id"
}

restore_network_band_policy() {
    local iface="$1"
    local network_id="$2"
    local freq_present="$3"
    local freq_value="$4"
    local marker_present="$5"
    local marker_value="$6"
    local failed=0

    if [[ "$freq_present" == "true" ]]; then
        wpa_cli_ok "$iface" set_network "$network_id" freq_list "$freq_value" || failed=1
    else
        wpa_cli_ok "$iface" set_network "$network_id" freq_list NULL || failed=1
    fi

    if [[ "$marker_present" == "true" ]]; then
        wpa_cli_ok "$iface" set_network "$network_id" id_str "$marker_value" || failed=1
    else
        wpa_cli_ok "$iface" set_network "$network_id" id_str NULL || failed=1
    fi

    ((failed == 0))
}

apply_explicit_band_policy() {
    local iface="$1"
    local network_id="$2"
    local band="$3"
    local freq_list marker

    freq_list=$(build_freq_list_for_band "$iface" "$band") || return 1
    case "$band" in
    "2.4G") marker="$WIFI_BAND_MARKER_24G" ;;
    "5G") marker="$WIFI_BAND_MARKER_5G" ;;
    *) return 1 ;;
    esac

    wpa_cli_ok "$iface" set_network "$network_id" freq_list "$freq_list" || return 1
    if ! wpa_cli_ok "$iface" set_network "$network_id" id_str "\"$marker\""; then
        wpa_cli_ok "$iface" set_network "$network_id" freq_list NULL || true
        return 1
    fi
}

clear_nanokvm_band_policy() {
    local iface="$1"
    local network_id="$2"
    local marker

    marker=$(get_network_value "$iface" "$network_id" id_str) || return 0
    wifi_band_from_marker "$marker" >/dev/null || return 0

    wpa_cli_ok "$iface" set_network "$network_id" freq_list NULL || return 1
    wpa_cli_ok "$iface" set_network "$network_id" id_str NULL || return 1
}

network_selected_band() {
    local iface="$1"
    local network_id="$2"
    local marker

    marker=$(get_network_value "$iface" "$network_id" id_str) || {
        echo "AUTO"
        return 0
    }
    wifi_band_from_marker "$marker" || echo "AUTO"
}

display_ssid_for_network_id() {
    local iface="$1"
    local network_id="$2"
    local ssid="$3"
    local band

    band=$(network_selected_band "$iface" "$network_id") || band="AUTO"
    wifi_display_ssid "$ssid" "$band"
}

apply_sta_roaming_config() {
    local iface="$1"
    local network_id="$2"

    [[ -n "$network_id" ]] || return 0
    wpa_cli_ok "$iface" set_network "$network_id" bgscan "\"$WIFI_BGSCAN_CONFIG\""
}

apply_roaming_config_to_saved_networks() {
    local iface="$1"
    local saved_networks="$2"
    local network_id

    while read -r network_id _rest; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        apply_sta_roaming_config "$iface" "$network_id" ||
            debug_print "[wifi] failed to refresh bgscan for network id=$network_id"
    done <<<"$saved_networks"
}

restore_band_policies_to_saved_networks() {
    local iface="$1"
    local saved_networks="$2"
    local network_id selected_band

    while read -r network_id _rest; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        selected_band=$(network_selected_band "$iface" "$network_id") || return 1
        if [[ "$selected_band" != "AUTO" ]]; then
            apply_explicit_band_policy "$iface" "$network_id" "$selected_band" || return 1
        fi
    done <<<"$saved_networks"
}

wait_for_ssid_connection() {
    local iface="$1"
    local ssid="$2"
    local retry_limit="$3"
    local selected_band="${4:-AUTO}"
    local retry=0

    while [ $retry -lt "$retry_limit" ]; do
        local status_output raw_ssid current_ssid frequency current_band
        status_output=$(wpa_cli -i "$iface" status 2>/dev/null)
        raw_ssid=$(printf '%s\n' "$status_output" | grep "^ssid=" | cut -d= -f2-)
        current_ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
        if printf '%s\n' "$status_output" | grep -qF "wpa_state=COMPLETED" && [[ "$current_ssid" == "$ssid" ]]; then
            if [[ "$selected_band" == "AUTO" ]]; then
                return 0
            fi
            frequency=$(printf '%s\n' "$status_output" | grep "^freq=" | cut -d= -f2 | head -n1)
            current_band=$(wifi_band_from_frequency "$frequency") || current_band="UNKNOWN"
            if [[ "$current_band" == "$selected_band" ]]; then
                return 0
            fi
            if [[ "$current_band" != "UNKNOWN" ]]; then
                debug_print "[wifi] connected on $current_band while $selected_band was requested" >&2
                return 2
            fi
        fi
        sleep 1
        ((retry++))
    done

    return 1
}

current_connection_matches_saved_band() {
    local iface="$1"
    local network_id="$2"
    local selected_band status_output frequency current_band

    selected_band=$(network_selected_band "$iface" "$network_id") || return 1
    [[ "$selected_band" != "AUTO" ]] || return 0

    status_output=$(wpa_cli -i "$iface" status 2>/dev/null) || return 1
    frequency=$(printf '%s\n' "$status_output" | grep "^freq=" | cut -d= -f2 | head -n1)
    current_band=$(wifi_band_from_frequency "$frequency") || return 1
    [[ "$current_band" == "$selected_band" ]]
}

WIFI_RECONNECT_ERROR=""
WIFI_RECONNECT_SSID=""
WIFI_RECONNECT_NETWORK_ID=""

reset_wifi_reconnect_result() {
    WIFI_RECONNECT_ERROR=""
    WIFI_RECONNECT_SSID=""
    WIFI_RECONNECT_NETWORK_ID=""
}

disable_wifi_runtime_after_state_failure() {
    local iface="$1"

    wpa_cli -i "$iface" disable_network all >/dev/null 2>&1 || true
    wpa_cli -i "$iface" disconnect >/dev/null 2>&1 || true
    mark_saved_networks_disabled || true
    ip addr flush dev "$iface" >/dev/null 2>&1 || true
}

commit_reconnected_wifi_state() {
    local iface="$1"
    local ssid="$2"
    local network_id="$3"

    if ! persist_connected_wifi_state "$ssid"; then
        debug_print "[wifi] failed to persist connected WiFi state; disconnecting runtime association" >&2
        disable_wifi_runtime_after_state_failure "$iface"
        WIFI_RECONNECT_ERROR="STATE_SAVE_FAILED"
        return 1
    fi

    WIFI_RECONNECT_SSID="$ssid"
    WIFI_RECONNECT_NETWORK_ID="$network_id"
    return 0
}

select_saved_network_for_reconnect() {
    local iface="$1"
    local network_id="$2"
    local selected_band

    selected_band=$(network_selected_band "$iface" "$network_id") || return 1
    if ! wpa_cli_ok "$iface" disable_network all ||
        ! wpa_cli_ok "$iface" set_network "$network_id" mesh_fwding 0; then
        return 1
    fi
    if [[ "$selected_band" != "AUTO" ]] &&
        ! apply_explicit_band_policy "$iface" "$network_id" "$selected_band"; then
        return 1
    fi
    if ! apply_sta_roaming_config "$iface" "$network_id" ||
        ! wpa_cli_ok "$iface" save_config ||
        ! wpa_cli_ok "$iface" enable_network "$network_id" ||
        ! wpa_cli_ok "$iface" select_network "$network_id"; then
        return 1
    fi
    if [[ "$selected_band" == "AUTO" ]] &&
        ! wpa_cli_ok "$iface" save_config; then
        return 1
    fi

    printf '%s\n' "$selected_band"
}

reconnect_saved_networks() {
    local iface="$1"
    local preferred_ssid="${2:-}"
    local retry_limit="${3:-10}"
    local status current_network_id current_ssid selected_band

    reset_wifi_reconnect_result
    status=$(get_raw_connection_status) || {
        WIFI_RECONNECT_ERROR="NO_INTERFACE"
        return 1
    }
    if [[ "$status" == AP* ]]; then
        WIFI_RECONNECT_ERROR="ALREADY_CONNECTED"
        return 1
    fi

    if [[ "$status" == CONNECTED* ]]; then
        current_network_id=$(wpa_cli -i "$iface" status 2>/dev/null |
            grep "^id=" | cut -d= -f2 | head -n1)
        if [[ ! "$current_network_id" =~ ^[0-9]+$ ]] ||
            ! current_connection_matches_saved_band "$iface" "$current_network_id"; then
            WIFI_RECONNECT_ERROR="BAND_MISMATCH"
            return 1
        fi
        if ! configure_sta_ip "$iface" "true"; then
            WIFI_RECONNECT_ERROR="DHCP_FAILED"
            return 1
        fi
        current_ssid=${status#CONNECTED }
        commit_reconnected_wifi_state "$iface" "$current_ssid" \
            "$current_network_id"
        return $?
    fi

    local saved_networks any_network
    saved_networks=$(wpa_cli -i "$iface" list_networks 2>/dev/null || true)
    any_network=$(printf '%s\n' "$saved_networks" |
        awk 'NR > 1 && $1 ~ /^[0-9]+$/ { print $1; exit }')
    if [[ -z "$any_network" ]]; then
        WIFI_RECONNECT_ERROR="NO_SAVED_NETWORK"
        return 1
    fi

    if [[ -n "$preferred_ssid" ]]; then
        local preferred_network_id
        preferred_network_id=$(find_network_ids_by_ssid "$iface" "$preferred_ssid" \
            2>/dev/null | head -n1)
        if [[ -n "$preferred_network_id" ]]; then
            selected_band=$(select_saved_network_for_reconnect "$iface" \
                "$preferred_network_id") || selected_band=""
            if [[ -n "$selected_band" ]] &&
                wait_for_ssid_connection "$iface" "$preferred_ssid" \
                    "$retry_limit" "$selected_band"; then
                if ! configure_sta_ip "$iface"; then
                    # A preferred association without an address is not a
                    # successful reconnect. Fall through to the all-saved
                    # fallback after resetting the runtime candidate set.
                    debug_print "[wifi] preferred saved network could not configure IP; falling back to all saved networks"
                else
                    commit_reconnected_wifi_state "$iface" "$preferred_ssid" \
                        "$preferred_network_id"
                    return $?
                fi
            fi
            debug_print "[wifi] preferred saved network failed; falling back to all saved networks"
        fi
    fi

    if ! wpa_cli_ok "$iface" disable_network all; then
        WIFI_RECONNECT_ERROR="ENABLE_FAILED"
        return 1
    fi
    apply_roaming_config_to_saved_networks "$iface" "$saved_networks"
    if ! restore_band_policies_to_saved_networks "$iface" "$saved_networks" ||
        ! wpa_cli_ok "$iface" save_config ||
        ! wpa_cli_ok "$iface" enable_network all; then
        WIFI_RECONNECT_ERROR="ENABLE_FAILED"
        return 1
    fi

    local retry=0
    while ((retry < retry_limit)); do
        status=$(get_raw_connection_status 2>/dev/null || true)
        if [[ "$status" == CONNECTED* ]]; then
            current_ssid=${status#CONNECTED }
            current_network_id=$(wpa_cli -i "$iface" status 2>/dev/null |
                grep "^id=" | cut -d= -f2 | head -n1)
            if [[ ! "$current_network_id" =~ ^[0-9]+$ ]] ||
                ! current_connection_matches_saved_band "$iface" "$current_network_id"; then
                WIFI_RECONNECT_ERROR="BAND_MISMATCH"
                return 1
            fi
            if ! configure_sta_ip "$iface"; then
                WIFI_RECONNECT_ERROR="DHCP_FAILED"
                return 1
            fi
            commit_reconnected_wifi_state "$iface" "$current_ssid" \
                "$current_network_id"
            return $?
        fi
        sleep 1
        ((retry++))
    done

    WIFI_RECONNECT_ERROR="NO_SAVED_NETWORK"
    return 1
}

wpa_control_socket_exists() {
    local iface="$1"
    [[ -S "/var/run/wpa_supplicant/$iface" ]]
}

get_current_wifi_band() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }

    if ! wpa_control_socket_exists "$WIFI_IFACE"; then
        echo "ERROR: NOT_CONNECTED"
        return 1
    fi

    local status_output state frequency band
    status_output=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null)
    state=$(echo "$status_output" | grep "^wpa_state=" | cut -d= -f2)
    if [[ "$state" != "COMPLETED" ]]; then
        echo "ERROR: NOT_CONNECTED"
        return 1
    fi

    frequency=$(echo "$status_output" | grep "^freq=" | cut -d= -f2 | head -n1)
    if [[ -z "$frequency" ]]; then
        frequency=$(wpa_cli -i "$WIFI_IFACE" signal_poll 2>/dev/null | grep "^FREQUENCY=" | cut -d= -f2 | head -n1)
    fi
    if [[ -z "$frequency" ]]; then
        frequency=$(iw dev "$WIFI_IFACE" link 2>/dev/null | grep -Eo 'freq: [0-9]+' | awk '{print $2}' | head -n1)
    fi

    band=$(wifi_band_from_frequency "$frequency") || true
    echo "$band"
    [[ "$band" != "UNKNOWN" ]]
}

connect_wifi() {
    local input_ssid="$1"
    local password="${2:-}"
    local connect_mode="${3:-user}"
    local resolved real_b64 selected_band display_b64 policy_mode ssid display_ssid

    resolved=$(resolve_connection_target "$input_ssid" "$connect_mode") || {
        debug_print "[wifi] failed to resolve WiFi selection" >&2
        return 1
    }
    IFS=$'\t' read -r real_b64 selected_band display_b64 policy_mode <<<"$resolved"
    ssid=$(base64_decode_field "$real_b64") || return 1
    display_ssid=$(base64_decode_field "$display_b64") || return 1

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || return 1

    start_service

    if ! wpa_control_socket_exists "$WIFI_IFACE"; then
        echo "[wifi] wpa_supplicant socket not found for $WIFI_IFACE" >&2
        return 1
    fi

    local matching_ids existing="" network_id="" created_network="false"
    matching_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$ssid" 2>/dev/null || true)
    existing=$(printf '%s\n' "$matching_ids" | awk '/^[0-9]+$/ {print; exit}')

    if [[ -n "$password" && -n "$existing" ]]; then
        existing=""
    fi

    if [[ -n "$existing" ]]; then
        network_id="$existing"
        debug_print "[wifi] found existing network id=$network_id, reusing"
    else
        debug_print "[wifi] adding new network: $ssid"
        network_id=$(wpa_cli -i "$WIFI_IFACE" add_network 2>/dev/null | tail -n1)
        [[ "$network_id" =~ ^[0-9]+$ ]] || {
            debug_print "[wifi] failed to add network" >&2
            return 1
        }
        created_network="true"

        local ssid_hex
        ssid_hex=$(ssid_to_hex "$ssid") || {
            wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
            return 1
        }
        if ! wpa_cli_ok "$WIFI_IFACE" set_network "$network_id" ssid "$ssid_hex"; then
            wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
            return 1
        fi

        if ! configure_wifi_credentials "$WIFI_IFACE" "$network_id" "$ssid" "$password"; then
            wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
            return 1
        fi
    fi

    local old_freq_present="false" old_freq_value=""
    local old_marker_present="false" old_marker_value=""
    if old_freq_value=$(get_network_value "$WIFI_IFACE" "$network_id" freq_list); then
        old_freq_present="true"
    fi
    if old_marker_value=$(get_network_value "$WIFI_IFACE" "$network_id" id_str); then
        old_marker_present="true"
    fi

    if ! wpa_cli_ok "$WIFI_IFACE" disable_network all ||
        ! wpa_cli_ok "$WIFI_IFACE" set_network "$network_id" mesh_fwding 0; then
        [[ "$created_network" == "true" ]] && wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        return 1
    fi

    local policy_ok="true"
    case "$policy_mode" in
    "EXPLICIT")
        apply_explicit_band_policy "$WIFI_IFACE" "$network_id" "$selected_band" || policy_ok="false"
        ;;
    "AUTO")
        clear_nanokvm_band_policy "$WIFI_IFACE" "$network_id" || policy_ok="false"
        selected_band="AUTO"
        ;;
    "PRESERVE")
        selected_band=$(network_selected_band "$WIFI_IFACE" "$network_id") || policy_ok="false"
        if [[ "$policy_ok" == "true" && "$selected_band" != "AUTO" ]]; then
            apply_explicit_band_policy "$WIFI_IFACE" "$network_id" "$selected_band" || policy_ok="false"
        fi
        ;;
    *) policy_ok="false" ;;
    esac

    if [[ "$policy_ok" != "true" ]] || ! apply_sta_roaming_config "$WIFI_IFACE" "$network_id"; then
        restore_network_band_policy "$WIFI_IFACE" "$network_id" \
            "$old_freq_present" "$old_freq_value" "$old_marker_present" "$old_marker_value" || true
        [[ "$created_network" == "true" ]] && wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        debug_print "[wifi] failed to configure network band policy" >&2
        return 1
    fi

    # Persist the frequency guard before the network can associate.
    if ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        restore_network_band_policy "$WIFI_IFACE" "$network_id" \
            "$old_freq_present" "$old_freq_value" "$old_marker_present" "$old_marker_value" || true
        [[ "$created_network" == "true" ]] && wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        debug_print "[wifi] failed to save network policy" >&2
        return 1
    fi

    local old_id removed_duplicate="false"
    while read -r old_id; do
        [[ "$old_id" =~ ^[0-9]+$ && "$old_id" != "$network_id" ]] || continue
        if ! wpa_cli_ok "$WIFI_IFACE" remove_network "$old_id"; then
            debug_print "[wifi] failed to remove duplicate network id=$old_id" >&2
            return 1
        fi
        removed_duplicate="true"
    done <<<"$matching_ids"
    if [[ "$removed_duplicate" == "true" ]]; then
        if ! wpa_cli_ok "$WIFI_IFACE" save_config; then
            debug_print "[wifi] failed to save deduplicated network" >&2
            return 1
        fi
    fi

    if ! wpa_cli_ok "$WIFI_IFACE" enable_network "$network_id" ||
        ! wpa_cli_ok "$WIFI_IFACE" select_network "$network_id"; then
        debug_print "[wifi] failed to select network id=$network_id" >&2
        return 1
    fi
    # This target's wpa_supplicant accepts freq_list at runtime but omits it
    # from save_config. Keep explicit networks disabled in the file so every
    # service start reapplies the guard from the persistent id_str first.
    if [[ "$selected_band" == "AUTO" ]] && ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        debug_print "[wifi] failed to persist automatic network selection" >&2
        return 1
    fi

    debug_print "[wifi] waiting for connection..."
    if ! wait_for_ssid_connection "$WIFI_IFACE" "$ssid" "$WIFI_CONNECT_WAIT_SECONDS" "$selected_band"; then
        debug_print "[wifi] connection timeout or requested band unavailable" >&2
        return 1
    fi
    debug_print "[wifi] connection established"

    debug_print "[wifi] requesting IP address..."
    if ! configure_sta_ip "$WIFI_IFACE"; then
        echo "[wifi] IP configuration failed on $WIFI_IFACE" >&2
        return 1
    fi

    if ! persist_connected_wifi_state "$ssid"; then
        debug_print "[wifi] failed to persist connected WiFi state; disconnecting runtime association" >&2
        disable_wifi_runtime_after_state_failure "$WIFI_IFACE"
        return 1
    fi

    debug_print "[wifi] connected to $display_ssid"
    return 0
}

connect_enterprise_wifi() {
    local resolved real_b64 selected_band display_b64 policy_mode ssid display_ssid
    resolved=$(resolve_connection_target "$1" "user") || return 1
    IFS=$'\t' read -r real_b64 selected_band display_b64 policy_mode <<<"$resolved"
    ssid=$(base64_decode_field "$real_b64") || return 1
    display_ssid=$(base64_decode_field "$display_b64") || return 1
    local eap_method="$2"
    local identity="$3"
    shift 3

    local password="" phase2="" ca_cert="" client_cert="" private_key="" private_key_passwd=""
    local anonymous_identity="" domain=""

    for param in "$@"; do
        case "$param" in
        password=*) password="${param#password=}" ;;
        phase2=*) phase2="${param#phase2=}" ;;
        ca_cert=*) ca_cert="${param#ca_cert=}" ;;
        client_cert=*) client_cert="${param#client_cert=}" ;;
        private_key=*) private_key="${param#private_key=}" ;;
        private_key_passwd=*) private_key_passwd="${param#private_key_passwd=}" ;;
        anonymous_identity=*) anonymous_identity="${param#anonymous_identity=}" ;;
        domain=*) domain="${param#domain=}" ;;
        -d | --debug) ;;
        *)
            debug_print "[wifi] unknown enterprise parameter: $param" >&2
            ;;
        esac
    done

    eap_method=$(echo "$eap_method" | tr '[:lower:]' '[:upper:]')

    case "$eap_method" in
    PEAP | TLS | TTLS) ;;
    *)
        echo "[wifi] unsupported EAP method: $eap_method (supported: PEAP, TLS, TTLS)" >&2
        return 1
        ;;
    esac

    case "$eap_method" in
    PEAP)
        if [[ -z "$password" ]]; then
            echo "[wifi] PEAP requires password parameter" >&2
            return 1
        fi
        [[ -z "$phase2" ]] && phase2="auth=MSCHAPV2"
        ;;
    TLS)
        if [[ -z "$client_cert" || -z "$private_key" ]]; then
            echo "[wifi] EAP-TLS requires client_cert and private_key parameters" >&2
            return 1
        fi
        if [[ ! -f "$client_cert" ]]; then
            echo "[wifi] client certificate not found: $client_cert" >&2
            return 1
        fi
        if [[ ! -f "$private_key" ]]; then
            echo "[wifi] private key not found: $private_key" >&2
            return 1
        fi
        ;;
    TTLS)
        if [[ -z "$password" ]]; then
            echo "[wifi] EAP-TTLS requires password parameter" >&2
            return 1
        fi
        [[ -z "$phase2" ]] && phase2="auth=PAP"
        ;;
    esac

    if [[ -n "$ca_cert" && ! -f "$ca_cert" ]]; then
        echo "[wifi] CA certificate not found: $ca_cert" >&2
        return 1
    fi

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface)

    start_service

    if [[ ! -S "/var/run/wpa_supplicant/${WIFI_IFACE}" ]]; then
        echo "[wifi] wpa_supplicant socket not found for $WIFI_IFACE" >&2
        return 1
    fi

    debug_print "[wifi] enterprise connect: ssid=$ssid eap=$eap_method identity=$identity"

    local existing_ids
    existing_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$ssid" 2>/dev/null || true)

    local id
    id=$(wpa_cli -i "$WIFI_IFACE" add_network | tail -n1)
    debug_print "[wifi] added enterprise network id=$id"

    local ssid_hex
    ssid_hex=$(ssid_to_hex "$ssid") || return 1
    wpa_cli -i "$WIFI_IFACE" set_network "$id" ssid "$ssid_hex" >/dev/null
    wpa_cli -i "$WIFI_IFACE" set_network "$id" key_mgmt "WPA-EAP" >/dev/null
    wpa_cli -i "$WIFI_IFACE" set_network "$id" eap "$eap_method" >/dev/null
    wpa_cli -i "$WIFI_IFACE" set_network "$id" identity "\"$identity\"" >/dev/null

    if [[ -n "$anonymous_identity" ]]; then
        wpa_cli -i "$WIFI_IFACE" set_network "$id" anonymous_identity "\"$anonymous_identity\"" >/dev/null
    fi

    if [[ -n "$ca_cert" ]]; then
        wpa_cli -i "$WIFI_IFACE" set_network "$id" ca_cert "\"$ca_cert\"" >/dev/null
    fi

    if [[ -n "$domain" ]]; then
        wpa_cli -i "$WIFI_IFACE" set_network "$id" domain_suffix_match "\"$domain\"" >/dev/null
    fi

    case "$eap_method" in
    PEAP)
        wpa_cli -i "$WIFI_IFACE" set_network "$id" password "\"$password\"" >/dev/null
        wpa_cli -i "$WIFI_IFACE" set_network "$id" phase2 "\"$phase2\"" >/dev/null
        debug_print "[wifi] PEAP configured with phase2=$phase2"
        ;;
    TLS)
        wpa_cli -i "$WIFI_IFACE" set_network "$id" client_cert "\"$client_cert\"" >/dev/null
        wpa_cli -i "$WIFI_IFACE" set_network "$id" private_key "\"$private_key\"" >/dev/null
        if [[ -n "$private_key_passwd" ]]; then
            wpa_cli -i "$WIFI_IFACE" set_network "$id" private_key_passwd "\"$private_key_passwd\"" >/dev/null
        fi
        debug_print "[wifi] EAP-TLS configured with cert=$client_cert key=$private_key"
        ;;
    TTLS)
        wpa_cli -i "$WIFI_IFACE" set_network "$id" password "\"$password\"" >/dev/null
        wpa_cli -i "$WIFI_IFACE" set_network "$id" phase2 "\"$phase2\"" >/dev/null
        debug_print "[wifi] EAP-TTLS configured with phase2=$phase2"
        ;;
    esac

    if ! wpa_cli_ok "$WIFI_IFACE" disable_network all ||
        ! wpa_cli_ok "$WIFI_IFACE" set_network "$id" mesh_fwding 0; then
        wpa_cli_ok "$WIFI_IFACE" remove_network "$id" || true
        return 1
    fi
    if [[ "$policy_mode" == "EXPLICIT" ]]; then
        if ! apply_explicit_band_policy "$WIFI_IFACE" "$id" "$selected_band"; then
            wpa_cli_ok "$WIFI_IFACE" remove_network "$id" || true
            return 1
        fi
    else
        selected_band="AUTO"
    fi
    if ! apply_sta_roaming_config "$WIFI_IFACE" "$id" || ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        wpa_cli_ok "$WIFI_IFACE" remove_network "$id" || true
        return 1
    fi

    local old_id
    while read -r old_id; do
        [[ "$old_id" =~ ^[0-9]+$ && "$old_id" != "$id" ]] || continue
        wpa_cli_ok "$WIFI_IFACE" remove_network "$old_id" || return 1
    done <<<"$existing_ids"
    if [[ -n "$existing_ids" ]] && ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        return 1
    fi

    if ! wpa_cli_ok "$WIFI_IFACE" enable_network "$id" ||
        ! wpa_cli_ok "$WIFI_IFACE" select_network "$id"; then
        return 1
    fi
    if [[ "$selected_band" == "AUTO" ]] && ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        return 1
    fi

    debug_print "[wifi] waiting for enterprise connection..."
    if ! wait_for_ssid_connection "$WIFI_IFACE" "$ssid" 20 "$selected_band"; then
        echo "[wifi] enterprise connection timeout after 20 seconds" >&2
        local status_output
        status_output=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null)
        local eap_status
        eap_status=$(echo "$status_output" | grep "EAP state=" | cut -d= -f2)
        if [[ -n "$eap_status" ]]; then
            debug_print "[wifi] EAP state at timeout: $eap_status"
        fi
        return 1
    fi

    debug_print "[wifi] requesting IP address..."
    if ! configure_sta_ip "$WIFI_IFACE"; then
        echo "[wifi] IP configuration failed on $WIFI_IFACE" >&2
        return 1
    fi

    if ! persist_connected_wifi_state "$ssid"; then
        debug_print "[wifi] failed to persist connected enterprise WiFi state; disconnecting runtime association" >&2
        disable_wifi_runtime_after_state_failure "$WIFI_IFACE"
        return 1
    fi

    debug_print "[wifi] connected to enterprise network: $display_ssid (EAP: $eap_method)"
    return 0
}

disconnect_wifi() {
    local disconnect_reason="${1:-manual-off}"
    local preserve_durable_state="false"
    case "$disconnect_reason" in
    ap-switch | preserve-state | service-stop) preserve_durable_state="true" ;;
    esac

    debug_print "[wifi] disconnecting..."
    if ! load_or_migrate_wifi_state; then
        debug_print "[wifi] unified WiFi state is invalid while disconnecting; repairing it as disabled" >&2
    fi

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface 2>/dev/null || true)
    local current_ssid=""
    if [[ -n "$WIFI_IFACE" ]]; then
        current_ssid=$(read_current_sta_ssid 2>/dev/null || true)
        if ! wifi_state_ssid_is_valid "$current_ssid"; then
            current_ssid=""
        fi
    fi

    if [[ "$preserve_durable_state" != "true" ]]; then
        local preferred_ssid="$current_ssid"
        [[ -n "$preferred_ssid" ]] || preferred_ssid="$WIFI_STATE_PREFERRED_SSID"
        if ! write_wifi_state_file "0" "$preferred_ssid"; then
            debug_print "[wifi] failed to persist disabled WiFi state; leaving runtime connection unchanged" >&2
            return 1
        fi
    fi

    if [[ -z "$WIFI_IFACE" ]]; then
        debug_print "[wifi] no wireless interface, durable WiFi state is unchanged"
        return 0
    fi

    stop_dhcp_client "$WIFI_IFACE"

    if [[ -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        start_service

        local network_id=$(wpa_cli -i "$WIFI_IFACE" list_networks | grep '\[CURRENT\]' | awk '{print $1}')

        if [[ -n "$network_id" ]]; then
            debug_print "[wifi] disabling network id=$network_id"
            wpa_cli -i "$WIFI_IFACE" disable_network "$network_id" >/dev/null
        fi

        wpa_cli -i "$WIFI_IFACE" disconnect >/dev/null
        wpa_cli_ok "$WIFI_IFACE" save_config >/dev/null
    fi

    ip addr flush dev "${WIFI_IFACE}"

    # Persist disabled state into wpa_supplicant.conf so that
    # wpa_supplicant will NOT auto-reconnect on next start.
    mark_saved_networks_disabled || true

    return 0
}

remove_wifi() {
    local requested_ssid
    requested_ssid=$(normalize_ssid_arg "$1")
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface)

    local state_loaded="true"
    if ! load_or_migrate_wifi_state; then
        state_loaded="false"
        debug_print "[wifi] unified WiFi state is invalid while removing a network; treating it as disabled" >&2
    fi

    start_service

    if [[ ! -S "/var/run/wpa_supplicant/${WIFI_IFACE}" ]]; then
        debug_print "[wifi] wpa_supplicant socket not found for $WIFI_IFACE" >&2
        return 1
    fi

    local ssid="$requested_ssid" resolved real_b64 selected_band
    if resolved=$(resolve_wifi_alias "$requested_ssid"); then
        IFS=$'\t' read -r real_b64 selected_band <<<"$resolved"
        ssid=$(base64_decode_field "$real_b64") || return 1
    else
        local literal_ids
        literal_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$requested_ssid" 2>/dev/null || true)
        if [[ -z "$literal_ids" ]]; then
            local networks_output net_id raw_ssid bssid flags rest saved_ssid saved_display
            networks_output=$(wpa_cli -i "$WIFI_IFACE" list_networks 2>/dev/null || true)
            while IFS=$'\t' read -r net_id raw_ssid bssid flags rest; do
                [[ "$net_id" =~ ^[0-9]+$ ]] || continue
                saved_ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
                saved_display=$(display_ssid_for_network_id "$WIFI_IFACE" "$net_id" "$saved_ssid")
                if [[ "$saved_display" == "$requested_ssid" ]]; then
                    ssid="$saved_ssid"
                    break
                fi
            done <<<"$networks_output"
        fi
    fi
    debug_print "[wifi] removing network: $ssid"

    local network_ids network_id
    network_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$ssid" 2>/dev/null || true)
    network_id=$(printf '%s\n' "$network_ids" | awk '/^[0-9]+$/ {print; exit}')

    if [[ -z "$network_id" ]]; then
        debug_print "[wifi] network not found: $ssid" >&2
        return 1
    fi

    local current_network_id is_current="false"
    current_network_id=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^id=" | cut -d= -f2 | head -n1)
    if printf '%s\n' "$network_ids" | grep -qxF "$current_network_id"; then
        is_current="true"
    fi

    local removes_preferred="false"
    if [[ "$WIFI_STATE_PREFERRED_SSID" == "$ssid" ]]; then
        removes_preferred="true"
    fi
    if [[ "$state_loaded" != "true" || "$is_current" == "true" ||
        "$removes_preferred" == "true" ]]; then
        if ! write_wifi_state_file "0" ""; then
            debug_print "[wifi] failed to clear preferred WiFi state before removing network" >&2
            return 1
        fi
    fi

    if [[ "$is_current" == "true" ]]; then
        debug_print "[wifi] network is currently connected, disconnecting first..."
        if ! disconnect_wifi "preserve-state"; then
            return 1
        fi
        sleep 1
    fi

    while read -r network_id; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        debug_print "[wifi] removing network id=$network_id"
        if ! wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id"; then
            debug_print "[wifi] failed to remove network id=$network_id" >&2
            return 1
        fi
    done <<<"$network_ids"
    wpa_cli_ok "$WIFI_IFACE" save_config || return 1
    debug_print "[wifi] network removed and config saved: $ssid"

    clear_wifi_alias_registry

    return 0
}

wireless_ip_exists() {
    local WIFI_IFACE=$(get_wireless_interface)

    if [ -z "$WIFI_IFACE" ]; then
        echo "false"
        return 1
    fi

    if ! ip link show "$WIFI_IFACE" &>/dev/null; then
        echo "false"
        return 1
    fi

    if ip addr show "$WIFI_IFACE" 2>/dev/null | grep -E 'inet ' &>/dev/null; then
        echo "true"
        return 0
    else
        echo "false"
        return 0
    fi
}

has_wifi_config() {
    if load_wifi_state_file && [[ "$WIFI_STATE_ENABLED" == "1" ]]; then
        echo "true"
        return 0
    else
        echo "false"
        return 1
    fi
}

list_networks() {
    local WIFI_IFACE=$(get_wireless_interface)

    if [ -z "$WIFI_IFACE" ]; then
        echo "[]"
        return 1
    fi

    if [[ ! -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        echo "[]"
        return 1
    fi

    local networks_output
    networks_output=$(wpa_cli -i "$WIFI_IFACE" list_networks 2>/dev/null)

    if [ -z "$networks_output" ]; then
        echo "[]"
        return 1
    fi

    echo "$networks_output" | awk '
    NR > 1 && NF >= 3 {
        network_id = $1

        if ($NF ~ /^\[/) {
            flags = $NF
            bssid = $(NF-1)
            ssid = ""
            for (i = 2; i < NF - 1; i++) {
                ssid = ssid (i > 2 ? " " : "") $i
            }
        } else {
            flags = ""
            bssid = $NF
            ssid = ""
            for (i = 2; i < NF; i++) {
                ssid = ssid (i > 2 ? " " : "") $i
            }
        }

        flag_status = ""
        is_current = 0
        if (index(flags, "[CURRENT]") > 0) {
            flag_status = "CURRENT"
            is_current = 1
        } else if (index(flags, "[DISABLED]") > 0) {
            flag_status = "DISABLED"
        } else {
            flag_status = ""
        }

        network_json = sprintf("{\"ssid\":\"%s\",\"flags\":\"%s\"}", ssid, flag_status)

        if (is_current) {
            current_network = network_json
        } else {
            other_networks[other_count++] = network_json
        }
    }
    END {
        print "["

        count = 0
        if (current_network) {
            printf "  %s", current_network
            count++
        }

        for (i = 0; i < other_count; i++) {
            if (count > 0) print ","
            printf "  %s", other_networks[i]
            count++
        }

        print ""
        print "]"
    }'

    return 0
}

# ========== New high-level helper functions ==========

get_raw_connection_status() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }

    if pgrep hostapd >/dev/null 2>&1; then
        local ap_ssid
        ap_ssid=$(awk -F= '/^ssid=/{print $2}' "$HOSTAPD_CONF" 2>/dev/null)
        echo "AP ${ap_ssid:-unknown}"
        return 0
    fi

    if [[ -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        local state
        state=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^wpa_state=" | cut -d= -f2)
        if [[ "$state" == "COMPLETED" ]]; then
            local raw_ssid ssid
            raw_ssid=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^ssid=" | cut -d= -f2-)
            ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
            echo "CONNECTED ${ssid}"
            return 0
        fi
    fi

    echo "DISCONNECTED"
    return 0
}

get_connection_status() {
    local raw_status
    raw_status=$(get_raw_connection_status) || {
        printf '%s\n' "$raw_status"
        return 1
    }

    if [[ "$raw_status" != CONNECTED* ]]; then
        printf '%s\n' "$raw_status"
        return 0
    fi

    local WIFI_IFACE status_output raw_ssid ssid marker band
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }
    status_output=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null)
    raw_ssid=$(printf '%s\n' "$status_output" | grep "^ssid=" | cut -d= -f2-)
    ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
    marker=$(printf '%s\n' "$status_output" | grep "^id_str=" | cut -d= -f2-)
    band=$(wifi_band_from_marker "$marker" 2>/dev/null || true)
    printf 'CONNECTED %s\n' "$(wifi_display_ssid "$ssid" "$band")"
}

get_signal_strength() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }

    if [[ ! -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        echo "ERROR: NOT_CONNECTED"
        return 1
    fi

    local state
    state=$(wpa_cli -i "$WIFI_IFACE" status 2>/dev/null | grep "^wpa_state=" | cut -d= -f2)
    if [[ "$state" != "COMPLETED" ]]; then
        echo "ERROR: NOT_CONNECTED"
        return 1
    fi

    local rssi
    rssi=$(wpa_cli -i "$WIFI_IFACE" signal_poll 2>/dev/null | grep "^RSSI=" | cut -d= -f2)
    if [[ -n "$rssi" ]]; then
        echo "$rssi"
        return 0
    fi

    rssi=$(iw dev "$WIFI_IFACE" station dump 2>/dev/null | grep "signal:" | awk '{print $2}')
    if [[ -n "$rssi" ]]; then
        echo "$rssi"
        return 0
    fi

    echo "ERROR: SIGNAL_NOT_AVAILABLE"
    return 1
}

get_saved_networks() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface)

    if [ -z "$WIFI_IFACE" ]; then
        echo "[]"
        return 1
    fi

    if [[ ! -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        echo "[]"
        return 1
    fi

    local networks_output
    networks_output=$(wpa_cli -i "$WIFI_IFACE" list_networks 2>/dev/null)

    if [ -z "$networks_output" ]; then
        echo "[]"
        return 1
    fi

    local output="["
    local first=1 net_id raw_ssid bssid flags rest ssid display_ssid escaped_ssid ssid_b64
    local -A seen_ssids=()
    while IFS=$'\t' read -r net_id raw_ssid bssid flags rest; do
        [[ "$net_id" =~ ^[0-9]+$ && -n "$raw_ssid" ]] || continue
        ssid=$(decode_wpa_ssid "$raw_ssid" 2>/dev/null || printf '%s' "$raw_ssid")
        ssid_b64=$(base64_encode_field "$ssid") || continue
        [[ -z "${seen_ssids[$ssid_b64]+x}" ]] || continue
        seen_ssids["$ssid_b64"]=1

        display_ssid=$(display_ssid_for_network_id "$WIFI_IFACE" "$net_id" "$ssid")
        escaped_ssid=$(json_escape_wifi_string "$display_ssid")
        if ((first == 1)); then
            first=0
        else
            output+=","
        fi
        output+="{\"ssid\":\"${escaped_ssid}\"}"
    done <<<"$networks_output"
    output+="]"
    printf '%s\n' "$output"
}

clear_saved_networks() {
    clear_wifi_alias_registry

    if ! write_wifi_state_file "0" ""; then
        echo "ERROR: STATE_SAVE_FAILED"
        return 1
    fi

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || {
        echo "ERROR: NO_INTERFACE"
        return 1
    }

    start_service

    if [[ -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        wpa_cli -i "$WIFI_IFACE" disconnect >/dev/null 2>&1
        wpa_cli -i "$WIFI_IFACE" remove_network all >/dev/null 2>&1
        wpa_cli_ok "$WIFI_IFACE" save_config >/dev/null 2>&1
    fi

    ip addr flush dev "$WIFI_IFACE" 2>/dev/null || true
    echo "OK: All records cleared"
    return 0
}

turn_on_wifi() {
    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface 2>/dev/null || true)

    local status
    if [[ -n "$WIFI_IFACE" ]]; then
        status=$(get_raw_connection_status)
        if [[ "$status" == AP* ]]; then
            echo "ERROR: ALREADY_CONNECTED"
            return 1
        fi
    fi

    if ! load_or_migrate_wifi_state; then
        debug_print "[wifi] unified WiFi state is invalid while enabling; replacing it with explicit enabled intent" >&2
    fi
    if ! write_wifi_state_file "1" "$WIFI_STATE_PREFERRED_SSID"; then
        echo "ERROR: STATE_SAVE_FAILED"
        return 1
    fi

    # Enabled intent is durable even when the driver/interface is temporarily
    # unavailable. A later service start can consume this state and reconnect.
    if [[ -z "$WIFI_IFACE" ]]; then
        echo "ERROR: NO_INTERFACE"
        return 1
    fi

    # Do not let pre-existing per-profile flags associate an arbitrary network
    # between starting wpa_supplicant and the preferred-first reconnect helper.
    ensure_wpa_conf
    if ! mark_saved_networks_disabled; then
        echo "ERROR: ENABLE_FAILED"
        return 1
    fi

    start_service
    if [[ ! -S "/var/run/wpa_supplicant/$WIFI_IFACE" ]]; then
        echo "ERROR: WPA_NOT_RUNNING"
        return 1
    fi

    if ! reconnect_saved_networks "$WIFI_IFACE" "$WIFI_STATE_PREFERRED_SSID"; then
        echo "ERROR: ${WIFI_RECONNECT_ERROR:-NO_SAVED_NETWORK}"
        return 1
    fi

    echo "OK: Connected to $(display_ssid_for_network_id "$WIFI_IFACE" \
        "$WIFI_RECONNECT_NETWORK_ID" "$WIFI_RECONNECT_SSID")"
    return 0
}

turn_off_wifi() {
    if disconnect_wifi "manual-off"; then
        echo "OK: Disconnected"
        return 0
    else
        echo "ERROR: DISCONNECT_FAILED"
        return 1
    fi
}

remove_saved_wifi() {
    local ssid="$1"
    if [[ -z "$ssid" ]]; then
        echo "ERROR: INVALID_ARGS"
        return 1
    fi

    if remove_wifi "$ssid"; then
        echo "OK: Removed ${ssid}"
        return 0
    else
        echo "ERROR: NOT_FOUND"
        return 1
    fi
}

wifi_network_id_in_list() {
    local target_id="$1"
    local ids="$2"
    local network_id

    while read -r network_id; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        [[ "$network_id" == "$target_id" ]] && return 0
    done <<<"$ids"
    return 1
}

rollback_wifi_preset_save() {
    local iface="$1"
    local new_network_id="$2"
    local old_preset_ids="$3"
    local network_id failed=0

    wpa_cli_ok "$iface" remove_network "$new_network_id" || failed=1
    while read -r network_id; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        set_wifi_preset_network_marker "$iface" "$network_id" || failed=1
    done <<<"$old_preset_ids"
    wpa_cli_ok "$iface" save_config || failed=1

    ((failed == 0))
}

cleanup_replaced_wifi_preset_networks() {
    local iface="$1"
    local old_matching_ids="$2"
    local old_preset_ids="$3"
    local current_id fallback_id="" network_id removed="false"

    [[ -n "$old_matching_ids" ]] || return 0

    # Duplicate cleanup is optional and must never guess which live block can
    # be removed. Skip it when wpa_supplicant cannot identify the current id.
    current_id=$(current_wifi_network_id "$iface") || return 0
    if wifi_network_id_in_list "$current_id" "$old_matching_ids"; then
        fallback_id="$current_id"
    else
        while read -r network_id; do
            [[ "$network_id" =~ ^[0-9]+$ ]] || continue
            if wifi_network_id_in_list "$network_id" "$old_matching_ids"; then
                fallback_id="$network_id"
            fi
        done <<<"$old_preset_ids"
        if [[ -z "$fallback_id" ]]; then
            fallback_id=$(printf '%s\n' "$old_matching_ids" |
                awk '/^[0-9]+$/ {print; exit}')
        fi
    fi

    while read -r network_id; do
        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
        [[ "$network_id" == "$fallback_id" ]] && continue
        if wpa_cli_ok "$iface" remove_network "$network_id"; then
            removed="true"
        else
            debug_print "[wifi] failed to remove old duplicate WiFi network id=$network_id" >&2
        fi
    done <<<"$old_matching_ids"

    if [[ "$removed" == "true" ]] && ! wpa_cli_ok "$iface" save_config; then
        debug_print "[wifi] failed to save old duplicate WiFi cleanup" >&2
    fi
    return 0
}

preset_save_wifi() {
    local input_ssid="$1"
    local password="${2:-}"
    local resolved real_b64 selected_band display_b64 policy_mode ssid

    resolved=$(resolve_connection_target "$input_ssid" "user") || {
        debug_print "[wifi] failed to resolve preset WiFi selection" >&2
        return 1
    }
    IFS=$'\t' read -r real_b64 selected_band display_b64 policy_mode <<<"$resolved"
    ssid=$(base64_decode_field "$real_b64") || return 1

    local WIFI_IFACE
    WIFI_IFACE=$(get_wireless_interface) || return 1

    start_service

    if ! wpa_control_socket_exists "$WIFI_IFACE"; then
        debug_print "[wifi] wpa_supplicant socket not found for $WIFI_IFACE" >&2
        return 1
    fi

    # Preserve every existing block until the new disabled candidate and its
    # marker are durable. This keeps a current same-SSID link and its old
    # credentials available throughout the save transaction.
    local old_preset_ids old_matching_ids network_id
    old_preset_ids=$(find_wifi_preset_network_ids "$WIFI_IFACE" 2>/dev/null || true)
    old_matching_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$ssid" 2>/dev/null || true)

    network_id=$(wpa_cli -i "$WIFI_IFACE" add_network 2>/dev/null | tail -n1)
    [[ "$network_id" =~ ^[0-9]+$ ]] || {
        debug_print "[wifi] failed to add preset network" >&2
        return 1
    }

    local ssid_hex
    ssid_hex=$(ssid_to_hex "$ssid") || {
        wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        return 1
    }
    if ! wpa_cli_ok "$WIFI_IFACE" set_network "$network_id" ssid "$ssid_hex"; then
        wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        return 1
    fi

    if ! configure_wifi_credentials "$WIFI_IFACE" "$network_id" "$ssid" "$password"; then
        wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        return 1
    fi

    if ! set_wifi_preset_network_marker "$WIFI_IFACE" "$network_id" ||
        ! wpa_cli_ok "$WIFI_IFACE" disable_network "$network_id"; then
        wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
        debug_print "[wifi] failed to identify the new preset network" >&2
        return 1
    fi

    local old_network_id
    while read -r old_network_id; do
        [[ "$old_network_id" =~ ^[0-9]+$ ]] || continue
        if ! clear_wifi_preset_network_marker "$WIFI_IFACE" "$old_network_id"; then
            rollback_wifi_preset_save "$WIFI_IFACE" "$network_id" "$old_preset_ids" ||
                debug_print "[wifi] failed to roll back preset identity changes" >&2
            debug_print "[wifi] failed to demote old preset network id=$old_network_id" >&2
            return 1
        fi
    done <<<"$old_preset_ids"

    if ! wpa_cli_ok "$WIFI_IFACE" save_config; then
        rollback_wifi_preset_save "$WIFI_IFACE" "$network_id" "$old_preset_ids" ||
            debug_print "[wifi] failed to roll back preset configuration" >&2
        debug_print "[wifi] failed to persist preset network" >&2
        return 1
    fi

    # Publish the SSID marker last. A failed atomic replacement leaves the old
    # marker intact, then the wpa_supplicant identity switch is rolled back.
    if ! write_wifi_preset_file "$ssid"; then
        rollback_wifi_preset_save "$WIFI_IFACE" "$network_id" "$old_preset_ids" ||
            debug_print "[wifi] failed to roll back preset marker publication" >&2
        debug_print "[wifi] failed to write preset marker file" >&2
        return 1
    fi

    cleanup_replaced_wifi_preset_networks "$WIFI_IFACE" "$old_matching_ids" \
        "$old_preset_ids"

    debug_print "[wifi] preset WiFi saved: $ssid"
    return 0
}

preset_remove_wifi() {
    if [[ ! -f "$WIFI_PRESET_FILE" ]]; then
        echo "ERROR: NOT_FOUND"
        return 1
    fi

    if read_wifi_preset; then
        local ssid="$WIFI_PRESET_SSID"
        local WIFI_IFACE network_id
        local state_loaded="true"
        if ! load_or_migrate_wifi_state; then
            state_loaded="false"
            debug_print "[wifi] unified WiFi state is invalid while removing the preset; keeping the preset network block" >&2
        fi

        if WIFI_IFACE=$(get_wireless_interface 2>/dev/null); then
            start_service
            if wpa_control_socket_exists "$WIFI_IFACE"; then
                local preset_ids target_ids current_id="" tagged_target="false"
                preset_ids=$(find_wifi_preset_network_ids "$WIFI_IFACE" 2>/dev/null || true)
                target_ids=$(find_network_ids_by_ssid "$WIFI_IFACE" "$ssid" 2>/dev/null || true)
                current_id=$(current_wifi_network_id "$WIFI_IFACE" 2>/dev/null || true)

                while read -r network_id; do
                    [[ "$network_id" =~ ^[0-9]+$ ]] || continue
                    if wifi_network_id_in_list "$network_id" "$preset_ids"; then
                        tagged_target="true"
                        break
                    fi
                done <<<"$target_ids"

                if [[ "$tagged_target" == "true" ]]; then
                    # New-format presets have an explicit owner marker. Remove
                    # only that candidate when dormant; otherwise demote it to
                    # an ordinary saved block so the live link is not dropped.
                    while read -r network_id; do
                        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
                        if wifi_network_id_in_list "$network_id" "$target_ids"; then
                            if [[ "$state_loaded" != "true" ||
                                "$WIFI_STATE_PREFERRED_SSID" == "$ssid" ||
                                "$network_id" == "$current_id" ]]; then
                                clear_wifi_preset_network_marker "$WIFI_IFACE" "$network_id" || true
                            else
                                wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
                            fi
                        else
                            # Any stale preset identity belongs to a replaced
                            # candidate and must become an ordinary saved WiFi.
                            clear_wifi_preset_network_marker "$WIFI_IFACE" "$network_id" || true
                        fi
                    done <<<"$preset_ids"
                    wpa_cli_ok "$WIFI_IFACE" save_config || true
                else
                    # Legacy target blocks have no explicit identity. Any
                    # unrelated tagged candidates are stale remnants from an
                    # interrupted replacement and must be demoted first.
                    while read -r network_id; do
                        [[ "$network_id" =~ ^[0-9]+$ ]] || continue
                        clear_wifi_preset_network_marker "$WIFI_IFACE" "$network_id" || true
                    done <<<"$preset_ids"
                    if [[ -n "$preset_ids" ]]; then
                        wpa_cli_ok "$WIFI_IFACE" save_config || true
                    fi

                    # Preserve the conservative preferred/current protection
                    # used before tagged candidates existed.
                    local keep_legacy="false"
                    if [[ "$state_loaded" != "true" || "$WIFI_STATE_PREFERRED_SSID" == "$ssid" ]]; then
                        keep_legacy="true"
                    elif [[ -n "$current_id" ]] && wifi_network_id_in_list "$current_id" "$target_ids"; then
                        keep_legacy="true"
                    fi

                    if [[ "$keep_legacy" != "true" ]]; then
                        while read -r network_id; do
                            [[ "$network_id" =~ ^[0-9]+$ ]] || continue
                            wpa_cli_ok "$WIFI_IFACE" remove_network "$network_id" || true
                        done <<<"$target_ids"
                        wpa_cli_ok "$WIFI_IFACE" save_config || true
                    fi
                fi
            fi
        fi
    fi

    if ! remove_wifi_preset_file; then
        echo "ERROR: PRESET_FAILED"
        return 1
    fi

    echo "OK: Removed preset"
    return 0
}

preset_status_wifi() {
    local ssid=""
    if read_wifi_preset; then
        ssid="$WIFI_PRESET_SSID"
    fi

    local escaped
    escaped=$(json_escape_wifi_string "$ssid")
    printf '{"ssid":"%s"}\n' "$escaped"
    return 0
}

ARGS=()
for arg in "$@"; do
    if [[ "$arg" != "-d" && "$arg" != "--debug" ]]; then
        ARGS+=("$arg")
    fi
done
set -- "${ARGS[@]}"

case "$1" in
"start")
    start_wifi_system
    ;;
"stop")
    stop_wifi_system
    ;;
"restart")
    stop_wifi_system
    start_wifi_system
    ;;
"on")
    turn_on_wifi "manual"
    ;;
"off" | "connect_stop")
    turn_off_wifi
    ;;
"connect" | "connect_start")
    validate_arguments 1 $#
    if [[ -z "$2" ]]; then
        echo "ERROR: INVALID_ARGS"
        exit 1
    fi
    stop_wifi_ap
    if ! disconnect_wifi "manual-off"; then
        echo "ERROR: CONNECT_FAILED"
        exit 1
    fi
    if connect_wifi "$2" "${3:-}"; then
        echo "OK: Connected to $2"
    else
        echo "ERROR: CONNECT_FAILED"
        exit 1
    fi
    ;;
"status")
    get_connection_status
    ;;
"band")
    get_current_wifi_band
    ;;
"signal")
    get_signal_strength
    ;;
"scan" | "try_scan")
    try_scan
    ;;
"saved")
    get_saved_networks
    ;;
"boot_enabled")
    print_wifi_boot_enabled
    ;;
"preset")
    validate_arguments 1 $#
    case "$2" in
    "save")
        validate_arguments 2 $#
        if [[ -z "$3" ]]; then
            echo "ERROR: INVALID_ARGS"
            exit 1
        fi
        if preset_save_wifi "$3" "${4:-}"; then
            echo "OK: Preset $3"
        else
            echo "ERROR: PRESET_FAILED"
            exit 1
        fi
        ;;
    "status")
        preset_status_wifi
        ;;
    "remove")
        preset_remove_wifi
        ;;
    *)
        echo "ERROR: INVALID_ARGS"
        exit 1
        ;;
    esac
    ;;
"remove")
    validate_arguments 1 $#
    if [[ -z "$2" ]]; then
        echo "ERROR: INVALID_ARGS"
        exit 1
    fi
    remove_saved_wifi "$2"
    ;;
"clear")
    clear_saved_networks
    ;;
"ap")
    validate_arguments 2 $#
    if start_wifi_ap "$2" "$3"; then
        echo "OK: AP started $2"
    else
        echo "ERROR: AP_FAILED"
        exit 1
    fi
    ;;
"ap_off")
    stop_wifi_ap
    echo "OK: AP stopped"
    ;;
"enterprise")
    validate_arguments 3 $#
    if [[ -z "$2" || -z "$3" || -z "$4" ]]; then
        echo "ERROR: INVALID_ARGS"
        exit 1
    fi
    stop_wifi_ap
    if ! disconnect_wifi "manual-off"; then
        echo "ERROR: CONNECT_FAILED"
        exit 1
    fi
    if connect_enterprise_wifi "$2" "$3" "$4" "${@:5}"; then
        echo "OK: Connected to $2"
    else
        echo "ERROR: CONNECT_FAILED"
        exit 1
    fi
    ;;
"device_count")
    device_check || echo "ERROR: NO_INTERFACE"
    ;;
"ap_ip")
    get_ap_ip
    ;;
"h" | "-h" | "--help")
    show_help
    exit 0
    ;;
*)
    show_help
    exit 1
    ;;
esac
