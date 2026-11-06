#!/usr/bin/env python3

import argparse
import json
import os
import re
import select
import struct
import time
import uuid
from pathlib import Path

try:
    from evdev import InputDevice, ecodes, list_devices
except ModuleNotFoundError:
    InputDevice = None
    ecodes = None
    list_devices = None


UHID_DESTROY = 1
UHID_CREATE2 = 11
UHID_INPUT2 = 12
BUS_USB = 3

KEYBOARD_DESCRIPTOR = bytes.fromhex(
    "05010906a101050719e029e71500250175019508810295017508810395057501050819012905"
    "910295017503910395067508150025e70507190029e78100c0"
)
RELATIVE_MOUSE_DESCRIPTOR = bytes.fromhex(
    "05010902a1010901a10005091901290315002501950375018102950175058103050109300931"
    "09381581257f750895038106c0c0"
)
ABSOLUTE_POINTER_DESCRIPTOR = bytes.fromhex(
    "05010902a10185010901a1000509190129051500250175019505810275039501810305010930"
    "0931150026ff7f75109502810209381581257f750895018106c0c0"
)
DIGITIZER_STYLUS_DESCRIPTOR = bytes.fromhex(
    "050d0905a10185020920a100050d094209321500250175019502810275069501810305010930"
    "093116000026ff7f751095028102050d0930150026ff00750895018102c0c0"
)
TOUCHSCREEN_DESCRIPTOR_PREFIX = bytes.fromhex(
    "050d0904a1018503"
)
TOUCH_CONTACT_DESCRIPTOR = bytes.fromhex(
    "050d0922a10209420947150025017501950281027506950181037508250a095195018102"
    "0501150026ff7f751095010930810209318102c0"
)
TOUCHSCREEN_DESCRIPTOR_SUFFIX = bytes.fromhex(
    "050d1500250a750895010954810285040955b102c0"
)
TOUCHSCREEN_DESCRIPTOR = (
    TOUCHSCREEN_DESCRIPTOR_PREFIX
    + TOUCH_CONTACT_DESCRIPTOR * 10
    + TOUCHSCREEN_DESCRIPTOR_SUFFIX
)

HID2_PROFILES = {
    "absolute": (ABSOLUTE_POINTER_DESCRIPTOR, 7),
    "digitizer-stylus": (DIGITIZER_STYLUS_DESCRIPTOR, 7),
    "touch-screen": (TOUCHSCREEN_DESCRIPTOR, 62),
}


def fixed_string(value, size):
    encoded = value.encode("utf-8")[: size - 1]
    return encoded + bytes(size - len(encoded))


def create_device(handle, name, descriptor):
    request = (
        fixed_string(name, 128)
        + fixed_string("nanokvm-go/input-router-probe", 64)
        + fixed_string(str(uuid.uuid4()), 64)
        + struct.pack("<HHIIII", len(descriptor), BUS_USB, 0x359F, 0x0226, 1, 0)
        + descriptor
    )
    os.write(handle, struct.pack("<I", UHID_CREATE2) + request)


def send_report(handle, report):
    os.write(handle, struct.pack("<IH", UHID_INPUT2, len(report)) + report)


def find_event_device(name, required_types, timeout=10.0):
    require_evdev()
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        for device_path in list_devices():
            device = InputDevice(device_path)
            try:
                if device.name == name and all(
                    event_type in device.capabilities() for event_type in required_types
                ):
                    return device
            except OSError:
                pass
            device.close()
        time.sleep(0.05)
    raise RuntimeError(f"event device not found: {name}")


def collect_events(device, timeout=0.8):
    events = []
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        remaining = max(0.0, deadline - time.monotonic())
        if not select.select([device.fd], [], [], remaining)[0]:
            break
        events.extend(device.read())
    return events


def run_device_probe(name, descriptor, required_types, reports, expected_events, validator=None):
    require_evdev()
    handle = os.open("/dev/uhid", os.O_RDWR | os.O_CLOEXEC | os.O_NONBLOCK)
    try:
        create_device(handle, name, descriptor)
        device = find_event_device(name, required_types)
        try:
            device.grab()
            for report in reports:
                send_report(handle, report)
                time.sleep(0.05)
            observed = [
                (event.type, event.code, event.value)
                for event in collect_events(device)
                if event.type != ecodes.EV_SYN
            ]
            missing = [event for event in expected_events if event not in observed]
            if missing:
                raise RuntimeError(f"{name} missing events {missing}; observed {observed}")
            if validator:
                validator(observed)
            return {
                "name": name,
                "event_node": device.path,
                "observed": [
                    {"type": event_type, "code": code, "value": value}
                    for event_type, code, value in observed
                ],
            }
        finally:
            device.ungrab()
            device.close()
    finally:
        try:
            os.write(handle, struct.pack("<I", UHID_DESTROY))
        finally:
            os.close(handle)


def descriptors_from_usbdev(script_path, marker):
    descriptors = []
    for line in script_path.read_text(encoding="utf-8").splitlines():
        if marker not in line or "echo -ne" not in line:
            continue
        descriptors.append(bytes(int(value, 16) for value in re.findall(r"\\x([0-9a-fA-F]{1,2})", line)))
    return descriptors


def configure_hid2_source(script_path):
    source = script_path.read_text(encoding="utf-8")
    start = source.find("configure_hid2() {")
    if start < 0:
        raise RuntimeError("configure_hid2 is missing from usbdev.sh")
    end = source.find("\n}", start)
    if end < 0:
        raise RuntimeError("configure_hid2 is not terminated in usbdev.sh")
    return source[start:end]


def hid2_profile_sources(script_path):
    source = configure_hid2_source(script_path)
    profiles = {}
    modes = tuple(HID2_PROFILES)
    for mode in modes:
        match = re.search(
            rf"^\s*{re.escape(mode)}\)\s*$([\s\S]*?)(?=^\s*(?:{'|'.join(map(re.escape, modes))})\)\s*$|^\s*esac\s*$)",
            source,
            re.MULTILINE,
        )
        if not match:
            raise RuntimeError(f"HID2 profile case is missing from usbdev.sh: {mode}")
        case_source = match.group(1)
        report_length_match = re.search(
            r"echo\s+(\d+)\s*>\s*functions/hid[.]GS2/report_length",
            case_source,
        )
        descriptor_match = re.search(r"^\s*report_desc=([0-9a-fA-F]+)\s*$", case_source, re.MULTILINE)
        if not report_length_match or not descriptor_match:
            raise RuntimeError(f"HID2 profile is incomplete in usbdev.sh: {mode}")

        descriptor_hex = descriptor_match.group(1)
        if mode == "touch-screen":
            contact_match = re.search(
                r"^\s*touch_contact=([0-9a-fA-F]+)\s*$",
                case_source,
                re.MULTILINE,
            )
            suffix_match = re.search(
                r"^\s*report_desc\+=([0-9a-fA-F]+)\s*$",
                case_source,
                re.MULTILINE,
            )
            repeat_match = re.search(r"for\s+_\s+in\s+\$\(seq\s+1\s+(\d+)\)", case_source)
            if not contact_match or not suffix_match or not repeat_match:
                raise RuntimeError("touch-screen descriptor assembly is incomplete in usbdev.sh")
            descriptor_hex += contact_match.group(1) * int(repeat_match.group(1))
            descriptor_hex += suffix_match.group(1)

        profiles[mode] = (bytes.fromhex(descriptor_hex), int(report_length_match.group(1)))
    return profiles


def verify_descriptor_source(script_path):
    keyboard = descriptors_from_usbdev(script_path, "functions/hid.GS0/report_desc")
    relative_mouse = descriptors_from_usbdev(script_path, "functions/hid.GS1/report_desc")
    if not keyboard or any(value != KEYBOARD_DESCRIPTOR for value in keyboard):
        raise RuntimeError("UHID keyboard descriptor differs from usbdev.sh")
    if not relative_mouse or any(value != RELATIVE_MOUSE_DESCRIPTOR for value in relative_mouse):
        raise RuntimeError("UHID relative mouse descriptor differs from usbdev.sh")
    actual_profiles = hid2_profile_sources(script_path)
    for mode, expected in HID2_PROFILES.items():
        actual_descriptor, actual_report_length = actual_profiles[mode]
        expected_descriptor, expected_report_length = expected
        if actual_descriptor != expected_descriptor:
            raise RuntimeError(f"UHID {mode} descriptor differs from usbdev.sh")
        if actual_report_length != expected_report_length:
            raise RuntimeError(
                f"UHID {mode} report_length is {actual_report_length}; "
                f"want {expected_report_length}"
            )


def require_evdev():
    if InputDevice is None or ecodes is None or list_devices is None:
        raise RuntimeError(
            "python evdev is required for UHID event probing; "
            "use --verify-descriptors-only for source validation"
        )


def validate_touchscreen_events(observed):
    tracking_ids = [
        value
        for event_type, code, value in observed
        if event_type == ecodes.EV_ABS and code == ecodes.ABS_MT_TRACKING_ID
    ]
    if not any(value >= 0 for value in tracking_ids) or -1 not in tracking_ids:
        raise RuntimeError(f"touchscreen tracking IDs were not created and released: {observed}")
    touch_states = [
        value
        for event_type, code, value in observed
        if event_type == ecodes.EV_KEY and code == ecodes.BTN_TOUCH
    ]
    if touch_states != [1, 0]:
        raise RuntimeError(f"touchscreen BTN_TOUCH states = {touch_states}; want [1, 0]")


def validate_absolute_pointer_events(observed):
    button_states = [
        value
        for event_type, code, value in observed
        if event_type == ecodes.EV_KEY and code == ecodes.BTN_LEFT
    ]
    if button_states != [1, 0]:
        raise RuntimeError(f"absolute pointer BTN_LEFT states = {button_states}; want [1, 0]")
    required = {
        (ecodes.EV_ABS, ecodes.ABS_X, 0x1234),
        (ecodes.EV_ABS, ecodes.ABS_Y, 0x5678),
    }
    missing = required.difference(observed)
    if missing:
        raise RuntimeError(f"absolute pointer missing events {sorted(missing)}; observed {observed}")


def validate_stylus_events(observed):
    touch_states = [
        value
        for event_type, code, value in observed
        if event_type == ecodes.EV_KEY and code == ecodes.BTN_TOUCH
    ]
    if touch_states != [1, 0]:
        raise RuntimeError(f"stylus BTN_TOUCH states = {touch_states}; want [1, 0]")
    required = {
        (ecodes.EV_ABS, ecodes.ABS_X, 0x2345),
        (ecodes.EV_ABS, ecodes.ABS_Y, 0x6789),
        (ecodes.EV_ABS, ecodes.ABS_PRESSURE, 0x7F),
    }
    missing = required.difference(observed)
    if missing:
        raise RuntimeError(f"stylus missing events {sorted(missing)}; observed {observed}")


def absolute_pointer_report(buttons, x, y, wheel=0):
    return bytes([1, buttons & 0x1F]) + struct.pack("<HHb", x, y, wheel)


def stylus_report(state, x, y, pressure=0):
    return bytes([2, state & 0x03]) + struct.pack("<HHB", x, y, pressure)


def touchscreen_report(active_contacts):
    report = bytearray(62)
    report[0] = 3
    for index in range(10):
        report[1 + index * 6 + 1] = index + 1
    for index, (contact_id, x, y) in enumerate(active_contacts):
        offset = 1 + index * 6
        report[offset] = 3
        report[offset + 1] = contact_id
        struct.pack_into("<HH", report, offset + 2, x, y)
    report[-1] = len(active_contacts) or 10
    return bytes(report)


def parse_args():
    repository = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(
        description=(
            "Validate NanoKVM-Go keyboard, relative mouse, absolute pointer, "
            "digitizer stylus, and touchscreen descriptors through Linux UHID."
        )
    )
    parser.add_argument(
        "--usbdev-script",
        type=Path,
        default=repository / "debhelper/packages/kvmcomm/rootfs/opt/scripts/usbdev.sh",
    )
    parser.add_argument("--output", type=Path)
    parser.add_argument(
        "--verify-descriptors-only",
        action="store_true",
        help="verify usbdev.sh profile descriptors without requiring evdev or /dev/uhid",
    )
    return parser.parse_args()


def main():
    options = parse_args()
    verify_descriptor_source(options.usbdev_script)
    if options.verify_descriptors_only:
        evidence = {
            "schema": "nanokvm-input-router-linux-uhid/v2",
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "usbdev_script": str(options.usbdev_script),
            "descriptor_profiles": {
                mode: {
                    "descriptor_length": len(descriptor),
                    "report_length": report_length,
                }
                for mode, (descriptor, report_length) in HID2_PROFILES.items()
            },
            "event_probe_skipped": True,
            "pass": True,
        }
        write_evidence(evidence, options.output)
        return

    require_evdev()
    probes = [
        run_device_probe(
            "NanoKVM GO UHID Keyboard",
            KEYBOARD_DESCRIPTOR,
            [ecodes.EV_KEY],
            [bytes([2, 0, 0, 0, 0, 0, 0, 0]), bytes(8)],
            [
                (ecodes.EV_KEY, ecodes.KEY_LEFTSHIFT, 1),
                (ecodes.EV_KEY, ecodes.KEY_LEFTSHIFT, 0),
            ],
        ),
        run_device_probe(
            "NanoKVM GO UHID Relative Mouse",
            RELATIVE_MOUSE_DESCRIPTOR,
            [ecodes.EV_REL],
            [bytes([0, 1, 0, 0]), bytes([0, 255, 0, 0])],
            [
                (ecodes.EV_REL, ecodes.REL_X, 1),
                (ecodes.EV_REL, ecodes.REL_X, -1),
            ],
        ),
        run_device_probe(
            "NanoKVM GO UHID Absolute Pointer",
            ABSOLUTE_POINTER_DESCRIPTOR,
            [ecodes.EV_ABS, ecodes.EV_KEY],
            [
                absolute_pointer_report(1, 0x1234, 0x5678),
                absolute_pointer_report(0, 0x1234, 0x5678),
            ],
            [],
            validator=validate_absolute_pointer_events,
        ),
        run_device_probe(
            "NanoKVM GO UHID Digitizer Stylus",
            DIGITIZER_STYLUS_DESCRIPTOR,
            [ecodes.EV_ABS, ecodes.EV_KEY],
            [
                stylus_report(3, 0x2345, 0x6789, 0x7F),
                stylus_report(0, 0x2345, 0x6789, 0),
            ],
            [],
            validator=validate_stylus_events,
        ),
        run_device_probe(
            "NanoKVM GO UHID Touchscreen",
            TOUCHSCREEN_DESCRIPTOR,
            [ecodes.EV_ABS, ecodes.EV_KEY],
            [
                touchscreen_report([(1, 0x4000, 0x5000), (2, 0x6000, 0x7000)]),
                touchscreen_report([]),
            ],
            [],
            validator=validate_touchscreen_events,
        ),
    ]
    evidence = {
        "schema": "nanokvm-input-router-linux-uhid/v2",
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "kernel": os.uname().release,
        "usbdev_script": str(options.usbdev_script),
        "descriptor_profiles": {
            mode: {
                "descriptor_length": len(descriptor),
                "report_length": report_length,
            }
            for mode, (descriptor, report_length) in HID2_PROFILES.items()
        },
        "event_probe_skipped": False,
        "grabbed_before_input": True,
        "probes": probes,
        "pass": True,
    }
    write_evidence(evidence, options.output)


def write_evidence(evidence, output_path):
    output = json.dumps(evidence, indent=2) + "\n"
    if output_path:
        output_path.write_text(output, encoding="utf-8")
    print(output, end="")


if __name__ == "__main__":
    main()
