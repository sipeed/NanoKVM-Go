#!/usr/bin/env bash

set -euo pipefail

readonly TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly USBDEV_SCRIPT="${TEST_DIR}/../rootfs/opt/scripts/usbdev.sh"
readonly KERNEL_PATCH="${TEST_DIR}/../../../../support/patches/maix_ax620e_sdk_kernel/0001-hid-touchscreen-feature-report.patch"
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT

extract_function() {
    local function_name="$1"
    awk -v function_name="$function_name" '
        $0 ~ ("^" function_name "\\(\\)[[:space:]]*\\{[[:space:]]*$") { in_function = 1 }
        in_function {
            print
            if ($0 ~ /^}[[:space:]]*$/) exit
        }
    ' "$USBDEV_SCRIPT"
}

read_mode_source=$(extract_function read_hid2_mode)
configure_source=$(extract_function configure_hid2)
[[ -n "$read_mode_source" ]]
[[ -n "$configure_source" ]]
eval "$read_mode_source"
eval "$configure_source"

enable_hid_wakeup_on_write() {
    local hid_func="$1"

    [[ "$hid_func" == "hid.GS2" ]]
    echo 1 > "functions/${hid_func}/wakeup_on_write"
}

assert_profile() {
    local case_name="$1"
    local marker_value="$2"
    local expected_profile="$3"
    local expect_warning="$4"
    local case_root="${TEST_ROOT}/${case_name}"
    local marker="${case_root}/usb.hid2_mode"

    mkdir -p "${case_root}/gadget/functions" "${case_root}/gadget/configs/c.1"
    if [[ "$marker_value" != "__missing__" ]]; then
        printf '%s\n' "$marker_value" > "$marker"
    fi

    (
        cd "${case_root}/gadget"
        configure_hid2 "$marker" 2> "${case_root}/warning.log"

        [[ "$(cat functions/hid.GS2/no_out_endpoint)" == "1" ]]
        [[ "$(cat functions/hid.GS2/wakeup_on_write)" == "1" ]]
        [[ "$(cat functions/hid.GS2/protocol)" == "0" ]]
        [[ "$(cat functions/hid.GS2/subclass)" == "0" ]]
        [[ -L configs/c.1/hid.GS2 ]]
        [[ "$(readlink configs/c.1/hid.GS2)" == "functions/hid.GS2" ]]
        [[ "$(find configs/c.1 -maxdepth 1 -type l -name 'hid.GS2' | wc -l)" == "1" ]]

        python3 - "$expected_profile" functions/hid.GS2/report_desc functions/hid.GS2/report_length <<'PY'
from collections import defaultdict
import pathlib
import sys

profile = sys.argv[1]
descriptor = pathlib.Path(sys.argv[2]).read_bytes()
report_length = int(pathlib.Path(sys.argv[3]).read_text().strip())

absolute = bytes.fromhex(
    "05010902a10185010901a10005091901290515002501750195058102750395018103"
    "050109300931150026ff7f75109502810209381581257f750895018106c0c0"
)
stylus = bytes.fromhex(
    "050d0905a10185020920a100050d0942093215002501750195028102750695018103"
    "05010930093116000026ff7f751095028102050d0930150026ff00750895018102c0c0"
)
touch_header = bytes.fromhex("050d0904a1018503")
touch_contact = bytes.fromhex(
    "050d0922a10209420947150025017501950281027506950181037508250a09519501"
    "81020501150026ff7f751095010930810209318102c0"
)
touch_tail = bytes.fromhex("050d1500250a750895010954810285040955b102c0")
touch_screen = touch_header + touch_contact * 10 + touch_tail

expected_descriptors = {
    "absolute": absolute,
    "digitizer-stylus": stylus,
    "touch-screen": touch_screen,
}
expected_lengths = {
    "absolute": (65, 7),
    "digitizer-stylus": (69, 7),
    "touch-screen": (589, 62),
}

assert descriptor == expected_descriptors[profile]
assert (len(descriptor), report_length) == expected_lengths[profile]


def report_layout(data):
    report_size = 0
    report_count = 0
    report_id = 0
    report_ids = set()
    totals = defaultdict(int)
    offset = 0
    while offset < len(data):
        prefix = data[offset]
        offset += 1
        if prefix == 0xFE:
            length = data[offset]
            offset += 2 + length
            continue
        size_code = prefix & 0x03
        size = 4 if size_code == 3 else size_code
        item_type = (prefix >> 2) & 0x03
        tag = (prefix >> 4) & 0x0F
        value = int.from_bytes(data[offset:offset + size], "little")
        offset += size
        if item_type == 1 and tag == 7:
            report_size = value
        elif item_type == 1 and tag == 8:
            report_id = value
            report_ids.add(value)
        elif item_type == 1 and tag == 9:
            report_count = value
        elif item_type == 0 and tag in (8, 9, 11):
            totals[(tag, report_id)] += report_size * report_count
    return report_ids, totals


report_ids, bits = report_layout(descriptor)
if profile == "absolute":
    assert report_ids == {1}
    assert bits[(8, 1)] + 8 == 7 * 8
elif profile == "digitizer-stylus":
    assert report_ids == {2}
    assert bits[(8, 2)] + 8 == 7 * 8
    assert bytes.fromhex("050d0905a10185020920a100") in descriptor
    assert bytes.fromhex("050d0902a1018502") not in descriptor
else:
    assert report_ids == {3, 4}
    assert bits[(8, 3)] + 8 == 62 * 8
    assert bits[(11, 4)] + 8 == 2 * 8
    assert descriptor.count(bytes.fromhex("050d0922a102")) == 10
    assert descriptor.count(bytes.fromhex("250a095195018102")) == 10
    assert bytes.fromhex("0954810285040955b102") in descriptor
PY
    )

    if [[ "$expect_warning" == "yes" ]]; then
        grep -q "falling back to absolute" "${case_root}/warning.log"
    else
        [[ ! -s "${case_root}/warning.log" ]]
    fi
}

assert_profile absolute absolute absolute no
assert_profile digitizer_stylus digitizer-stylus digitizer-stylus no
assert_profile legacy_absolute_android absolute-android digitizer-stylus no
assert_profile touch_screen touch-screen touch-screen no
assert_profile missing_marker __missing__ absolute no
[[ "$(read_hid2_mode "${TEST_ROOT}/missing_marker/usb.hid2_mode")" == "single-absolute" ]]
[[ "$(read_hid2_mode "${TEST_ROOT}/absolute/usb.hid2_mode")" == "absolute" ]]
assert_profile invalid_marker invalid absolute yes

python3 - "$USBDEV_SCRIPT" <<'PY'
import pathlib
import sys

script = pathlib.Path(sys.argv[1]).read_text()


def function_body(name):
    start = script.index(f"{name}() {{")
    end = script.index("\n}", start)
    return script[start:end]


for function_name in ("hid_start", "hid_only_start"):
    body = function_body(function_name)
    assert "configure_usb_identity" in body
    gs0 = body.index("mkdir functions/hid.GS0")
    gs1 = body.index("mkdir functions/hid.GS1")
    gs2 = body.index("configure_hid2")
    assert gs0 < gs1 < gs2
    assert body.count("configure_hid2") == 1
    assert "mkdir functions/hid.GS2" not in body

assert script.count("mkdir functions/hid.GS2") == 1
PY

grep -q 'HIDG_TOUCHSCREEN_FEATURE_REPORT_ID.*4' "$KERNEL_PATCH"
grep -q 'HIDG_TOUCHSCREEN_MAX_CONTACTS.*10' "$KERNEL_PATCH"
grep -Fq 'echo 0x0521 > bcdDevice' "$USBDEV_SCRIPT"

printf 'ok - HID2 profile descriptors\n'
