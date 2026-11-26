#!/bin/bash
# All mutations are serialized by usbdev.sh's storage lock.
set -euo pipefail
base=/kvmcomm/webcam
run=/run/nanokvm-webcam
g=/sys/kernel/config/usb_gadget/g0
f=$g/functions/uvc.nomad
c=$g/configs/c.1
m=$f/streaming/mjpeg/m
unit=nanokvm-webcam.service
marker=/boot/usb.webcam
rule=60-persistent-v4l.rules
override=/run/udev/rules.d/$rule
fail() { echo "$*" >&2; exit 1; }

install_rule() {
    # The vendor UVC gadget disconnects the composite device when v4l_id closes it.
    local original=/usr/lib/udev/rules.d/$rule
    [[ ! -e /etc/udev/rules.d/$rule ]] || fail "Custom V4L udev rules are installed."
    [[ ! -e $override || $override -ef $run/$rule ]] || fail "Conflicting V4L udev rule."
    mkdir -p "$run" /run/udev/rules.d
    if [[ ! -e $override ]]; then
        grep -Fxq 'LABEL="persistent_v4l_end"' "$original"
        { echo 'SUBSYSTEM=="video4linux", KERNELS=="8000000.dwc3", GOTO="persistent_v4l_end"'; cat "$original"; } > "$run/$rule"
        ln "$run/$rule" "$override"
        udevadm control --reload
    fi
}
budget() {
    local link used=2
    for link in "$c"/*; do
        [[ -L $link ]] || continue
        case "${link##*/}" in
            hid.*|mass_storage.*|uac2.*) ((used+=1));;
            ncm.*|rndis.*|acm.*) ((used+=2));;
            uvc.nomad) ;;
            *) fail "Unsupported USB combination: ${link##*/}";;
        esac
    done
    ((used<=7)) || fail "Webcam requires two free USB IN endpoints. Disable another USB device first."
}
create() {
    budget
    install_rule
    mkdir "$f" || fail "The installed kernel must support ConfigFS UVC."
    if [[ -e $f/function_name ]]; then
        if [[ -f /boot/uvc.function_name ]]; then
            cat /boot/uvc.function_name > "$f/function_name"
        else
            echo 'Anker Powerconf C200' > "$f/function_name"
        fi
    fi
    mkdir -p "$f/control/header/h" "$f/streaming/header/h" "$m/720p"
    echo 1280 > "$m/720p/wWidth"
    echo 720 > "$m/720p/wHeight"
    echo 333333 > "$m/720p/dwFrameInterval"
    echo 333333 > "$m/720p/dwDefaultFrameInterval"
    echo 131072 > "$m/720p/dwMaxVideoFrameBufferSize"
    echo 1 > "$m/720p/dwMinBitRate"
    echo 31457280 > "$m/720p/dwMaxBitRate"
    echo 1 > "$m/bDefaultFrameIndex"
    echo 0x0100 > "$f/control/header/h/bcdUVC"
    echo 512 > "$f/streaming_maxpacket"
    echo 1 > "$f/streaming_interval"
    echo 0 > "$f/streaming_mult"
    echo 0 > "$f/streaming_bulk"
    ln -s "$m" "$f/streaming/header/h/m"
    for speed in fs hs ss; do ln -s "$f/streaming/header/h" "$f/streaming/class/$speed/h"; done
    for speed in fs ss; do ln -s "$f/control/header/h" "$f/control/class/$speed/h"; done
}
remove() {
    local item
    for item in "$c/uvc.nomad" "$f"/streaming/class/{ss,hs,fs}/h "$f"/control/class/{ss,fs}/h "$f/streaming/header/h/m"; do
        if [[ -L $item ]]; then unlink "$item"; fi
    done
    # An upgrade can leave the previous 600p descriptor bound until teardown.
    for item in "$m/720p" "$m/600p" "$m" "$f/streaming/header/h" "$f/control/header/h" "$f"; do
        if [[ -d $item ]]; then rmdir "$item"; fi
    done
    if [[ -e $run/$rule && $override -ef $run/$rule ]]; then
        unlink "$override"
        udevadm control --reload
    fi
}
serve() {
    local name node
    for ((i=0;i<50;i++)); do
        for name in /sys/class/video4linux/video*/function_name; do
            [[ -f $name && $(cat "$name") == uvc.nomad ]] || continue
            node=${name%/*}
            if [[ -c /dev/${node##*/} ]]; then
                exec "$base/nomad-uvc-feeder" "/dev/${node##*/}" "$base/camera-off.jpg" "$(cat "$f/control/bInterfaceNumber")" "$(cat "$f/streaming/bInterfaceNumber")"
            fi
        done
        sleep .1
    done
    fail "UVC output node missing."
}
launch() {
    if ! systemctl is-active --quiet "$unit"; then
        rm -f "$run/live.sock" "$run/streaming"
        systemctl reset-failed "$unit" 2>/dev/null || true
        systemd-run --quiet --collect --unit="$unit" "$base/control.sh" serve
    fi
    for ((i=0;i<75;i++)); do
        systemctl is-active --quiet "$unit" || break
        [[ ! -S $run/live.sock ]] || return 0
        sleep .1
    done
    fail "The webcam feeder did not start."
}
stop() {
    if systemctl is-active --quiet "$unit"; then systemctl stop "$unit"; fi
    rm -f "$run/live.sock" "$run/streaming"
}
wait_configured() {
    local controller=$1 attempts=$2 i
    for ((i=0;i<attempts;i++)); do
        [[ $(cat "/sys/class/udc/$controller/state") != configured ]] || return 0
        sleep .1
    done
    return 1
}
set_enabled() {
    local mode=$1 controller previous lun
    controller=$(cat "$g/UDC")
    [[ $controller == 8000000.dwc3 ]] || fail "Go+ USB controller unavailable."
    previous=$(cat "/sys/class/udc/$controller/state")
    for lun in "$g"/functions/mass_storage.*/lun.*/file; do
        if [[ -f $lun ]]; then
            local backing
            backing=$(cat "$lun")
            [[ -z $backing ]] || fail "Eject and detach virtual media before changing Webcam."
        fi
    done
    if [[ $mode == on ]]; then
        [[ ! -e /dev/shm/tmp/hid_only ]] || fail "Webcam requires the full USB profile."
        if [[ -L $c/uvc.nomad ]]; then
            if [[ -S $run/live.sock ]] && systemctl is-active --quiet "$unit"; then
                touch "$marker"
                return
            fi
        else
            [[ ! -d $f ]] || fail "Existing UVC function is busy."
            # Validate the budget, kernel and descriptors before disconnecting HID.
            trap remove EXIT
            create
            remove
            trap - EXIT
        fi
        touch "$marker"
    else
        rm -f "$marker"
        [[ -d $f ]] || return 0
    fi
    # The vendor DWC3 driver can stay soft-disconnected after UVC closes.
    # The stock restart cycles the controller and inherits the storage lock.
    if /opt/scripts/usbdev.sh restart &&
       { [[ $mode == off && ! -d $f ]] || [[ $mode == on && -L $c/uvc.nomad && -S $run/live.sock ]]; } &&
       { [[ $previous != configured ]] || wait_configured "$controller" 420; }; then
        return 0
    fi
    rm -f "$marker"
    /opt/scripts/usbdev.sh restart || fail "USB controller recovery failed."
    fail "The host did not configure the requested webcam profile. The base USB profile was restored."
}
case "$1" in
serve) serve;;
launch) launch;;
stop) stop;;
create) create; ln -s "$f" "$c/uvc.nomad";;
remove) remove;;
on|off) set_enabled "$1";;
*) exit 2;;
esac
