#!/usr/bin/env python3
"""
bt_hid_demo.py — Bluetooth HID Keyboard+Mouse Demo

Pure-Python implementation using BT Management Protocol + L2CAP sockets.
No BlueZ required. Runs on Linux with kernel BT stack (AF_BLUETOOTH).

Usage:
    python3 bt_hid_demo.py

After launch, pair from your PC's Bluetooth settings to "NanoKVM-BT".
The script will then type "hello" and move the mouse as a demo.
"""

import socket
import struct
import ctypes
import ctypes.util
import fcntl
import select
import threading
import time
import sys
import os
import signal
import traceback

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

# Linux kernel AF_BLUETOOTH family — NOT available in all Python builds,
# so we hardcode the values instead of using socket.AF_BLUETOOTH.
AF_BLUETOOTH = 31
BTPROTO_HCI = 1
BTPROTO_L2CAP = 0

# HCI ioctl numbers (arch-dependent; these are for ARM/generic Linux)
HCIDEVUP = 0x400448C9
HCIDEVDOWN = 0x400448CA

# BT Management Protocol channel.
# Python's socket.bind() only supports channel 0 (raw HCI).
# Opening channel 3 requires ctypes — see MgmtController.open().
HCI_CHANNEL_CONTROL = 3
HCI_DEV_NONE = 0xFFFF  # "non-specific" index, used for mgmt global commands

# Mgmt opcodes — values are kernel-version-specific!
# Verified against kernel 4.19 include/net/bluetooth/mgmt.h.
# WARNING: many online references give WRONG opcode numbers.
# Use READ_COMMANDS (0x0002) to verify at runtime if unsure.
MGMT_OP_READ_VERSION     = 0x0001
MGMT_OP_READ_INFO        = 0x0004
MGMT_OP_SET_POWERED      = 0x0005
MGMT_OP_SET_DISCOVERABLE = 0x0006
MGMT_OP_SET_CONNECTABLE  = 0x0007
MGMT_OP_SET_BONDABLE     = 0x0009  # was SET_PAIRABLE in older kernels
MGMT_OP_SET_SSP          = 0x000B  # NOT 0x0011 (that's REMOVE_UUID)
MGMT_OP_SET_DEV_CLASS    = 0x000E  # NOT 0x000D (that's SET_LE)
MGMT_OP_SET_LOCAL_NAME   = 0x000F  # params: 249B name + 11B short_name
MGMT_OP_PIN_CODE_REPLY         = 0x0016
MGMT_OP_PIN_CODE_NEG_REPLY     = 0x0017
MGMT_OP_SET_IO_CAP       = 0x0018  # NOT 0x001B (that's UNPAIR_DEVICE)
MGMT_OP_USER_CONFIRM_REPLY     = 0x001C  # NOT 0x001E
MGMT_OP_USER_CONFIRM_NEG_REPLY = 0x001D  # NOT 0x001F

# Mgmt events
MGMT_EV_CMD_COMPLETE     = 0x0001
MGMT_EV_CMD_STATUS       = 0x0002
MGMT_EV_NEW_SETTINGS     = 0x0006
MGMT_EV_NEW_LINK_KEY     = 0x0009
MGMT_EV_DEVICE_CONNECTED = 0x000B
MGMT_EV_DEVICE_DISCONNECTED = 0x000C
MGMT_EV_PIN_CODE_REQUEST = 0x000E
MGMT_EV_USER_CONFIRM_REQ = 0x000F
MGMT_EV_USER_PASSKEY_REQ = 0x0010
MGMT_EV_AUTH_FAILED      = 0x0011
MGMT_EV_PASSKEY_NOTIFY   = 0x0017

# Mgmt IO capabilities
MGMT_IO_CAP_NO_INPUT_NO_OUTPUT = 0x03

# Mgmt settings bits (returned by READ_INFO current_settings / supported_settings)
# HID device needs at minimum: 0xDB = POWERED|CONNECTABLE|DISCOVERABLE|BONDABLE|SSP|BREDR
MGMT_SETTING_POWERED      = 0x0001
MGMT_SETTING_CONNECTABLE  = 0x0002  # CRITICAL: without this, kernel rejects all inbound ACL
MGMT_SETTING_DISCOVERABLE = 0x0008
MGMT_SETTING_BONDABLE     = 0x0010  # CRITICAL: without this, pairing is refused
MGMT_SETTING_SSP          = 0x0040
MGMT_SETTING_BREDR        = 0x0080

# L2CAP PSMs
PSM_SDP = 0x0001
PSM_HID_CONTROL = 0x0011
PSM_HID_INTERRUPT = 0x0013

# BT security levels for setsockopt
SOL_BLUETOOTH = 274
BT_SECURITY = 4
BT_SECURITY_LOW = 1
BT_SECURITY_MEDIUM = 2

# HID report IDs
REPORT_ID_KEYBOARD = 0x01
REPORT_ID_MOUSE = 0x02

# HID transaction types (for control channel)
HID_HANDSHAKE = 0x00
HID_CONTROL = 0x10
HID_GET_REPORT = 0x40
HID_SET_REPORT = 0x50
HID_GET_PROTOCOL = 0x60
HID_SET_PROTOCOL = 0x70
HID_DATA = 0xA0

# HID handshake results
HID_HANDSHAKE_SUCCESS = 0x00
HID_HANDSHAKE_NOT_READY = 0x01
HID_HANDSHAKE_ERR_INVALID_REPORT_ID = 0x02
HID_HANDSHAKE_ERR_UNSUPPORTED_REQUEST = 0x03
HID_HANDSHAKE_ERR_INVALID_PARAMETER = 0x04
HID_HANDSHAKE_ERR_UNKNOWN = 0x0E
HID_HANDSHAKE_ERR_FATAL = 0x0F

# USB HID key codes
KEY_NONE = 0x00
KEY_A = 0x04
KEY_B = 0x05
KEY_C = 0x06
KEY_D = 0x07
KEY_E = 0x08
KEY_F = 0x09
KEY_G = 0x0A
KEY_H = 0x0B
KEY_I = 0x0C
KEY_J = 0x0D
KEY_K = 0x0E
KEY_L = 0x0F
KEY_M = 0x10
KEY_N = 0x11
KEY_O = 0x12
KEY_P = 0x13
KEY_Q = 0x14
KEY_R = 0x15
KEY_S = 0x16
KEY_T = 0x17
KEY_U = 0x18
KEY_V = 0x19
KEY_W = 0x1A
KEY_X = 0x1B
KEY_Y = 0x1C
KEY_Z = 0x1D
KEY_ENTER = 0x28
KEY_SPACE = 0x2C

CHAR_TO_KEYCODE = {
    'a': KEY_A, 'b': KEY_B, 'c': KEY_C, 'd': KEY_D, 'e': KEY_E,
    'f': KEY_F, 'g': KEY_G, 'h': KEY_H, 'i': KEY_I, 'j': KEY_J,
    'k': KEY_K, 'l': KEY_L, 'm': KEY_M, 'n': KEY_N, 'o': KEY_O,
    'p': KEY_P, 'q': KEY_Q, 'r': KEY_R, 's': KEY_S, 't': KEY_T,
    'u': KEY_U, 'v': KEY_V, 'w': KEY_W, 'x': KEY_X, 'y': KEY_Y,
    'z': KEY_Z, ' ': KEY_SPACE, '\n': KEY_ENTER,
}

# HID Report Descriptor: keyboard (report ID 1) + mouse (report ID 2)
HID_REPORT_DESCRIPTOR = bytes([
    # Keyboard
    0x05, 0x01,        # Usage Page (Generic Desktop)
    0x09, 0x06,        # Usage (Keyboard)
    0xA1, 0x01,        # Collection (Application)
    0x85, 0x01,        #   Report ID (1)
    0x05, 0x07,        #   Usage Page (Key Codes)
    0x19, 0xE0,        #   Usage Minimum (224) - Left Control
    0x29, 0xE7,        #   Usage Maximum (231) - Right GUI
    0x15, 0x00,        #   Logical Minimum (0)
    0x25, 0x01,        #   Logical Maximum (1)
    0x75, 0x01,        #   Report Size (1)
    0x95, 0x08,        #   Report Count (8)
    0x81, 0x02,        #   Input (Data, Variable, Absolute) - Modifier byte
    0x95, 0x01,        #   Report Count (1)
    0x75, 0x08,        #   Report Size (8)
    0x81, 0x03,        #   Input (Constant) - Reserved byte
    0x95, 0x05,        #   Report Count (5)
    0x75, 0x01,        #   Report Size (1)
    0x05, 0x08,        #   Usage Page (LEDs)
    0x19, 0x01,        #   Usage Minimum (1)
    0x29, 0x05,        #   Usage Maximum (5)
    0x91, 0x02,        #   Output (Data, Variable, Absolute) - LED report
    0x95, 0x01,        #   Report Count (1)
    0x75, 0x03,        #   Report Size (3)
    0x91, 0x03,        #   Output (Constant) - LED padding
    0x95, 0x06,        #   Report Count (6)
    0x75, 0x08,        #   Report Size (8)
    0x15, 0x00,        #   Logical Minimum (0)
    0x25, 0x65,        #   Logical Maximum (101)
    0x05, 0x07,        #   Usage Page (Key Codes)
    0x19, 0x00,        #   Usage Minimum (0)
    0x29, 0x65,        #   Usage Maximum (101)
    0x81, 0x00,        #   Input (Data, Array) - Key array (6 keys)
    0xC0,              # End Collection

    # Mouse
    0x05, 0x01,        # Usage Page (Generic Desktop)
    0x09, 0x02,        # Usage (Mouse)
    0xA1, 0x01,        # Collection (Application)
    0x85, 0x02,        #   Report ID (2)
    0x09, 0x01,        #   Usage (Pointer)
    0xA1, 0x00,        #   Collection (Physical)
    0x05, 0x09,        #     Usage Page (Buttons)
    0x19, 0x01,        #     Usage Minimum (1)
    0x29, 0x03,        #     Usage Maximum (3)
    0x15, 0x00,        #     Logical Minimum (0)
    0x25, 0x01,        #     Logical Maximum (1)
    0x95, 0x03,        #     Report Count (3)
    0x75, 0x01,        #     Report Size (1)
    0x81, 0x02,        #     Input (Data, Variable, Absolute) - Buttons
    0x95, 0x01,        #     Report Count (1)
    0x75, 0x05,        #     Report Size (5)
    0x81, 0x03,        #     Input (Constant) - Padding
    0x05, 0x01,        #     Usage Page (Generic Desktop)
    0x09, 0x30,        #     Usage (X)
    0x09, 0x31,        #     Usage (Y)
    0x09, 0x38,        #     Usage (Wheel)
    0x15, 0x81,        #     Logical Minimum (-127)
    0x25, 0x7F,        #     Logical Maximum (127)
    0x75, 0x08,        #     Report Size (8)
    0x95, 0x03,        #     Report Count (3)
    0x81, 0x06,        #     Input (Data, Variable, Relative)
    0xC0,              #   End Collection
    0xC0,              # End Collection
])

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

def log(tag, msg):
    print(f"[{tag}] {msg}", flush=True)

def log_err(tag, msg):
    print(f"[{tag}] ERROR: {msg}", file=sys.stderr, flush=True)

# ---------------------------------------------------------------------------
# BT Management Controller
# ---------------------------------------------------------------------------

class MgmtController:
    """Manage BT device via Linux BT Management Protocol (HCI_CHANNEL_CONTROL).

    This properly configures the kernel BT stack so that incoming connections,
    SSP pairing, and link-key management are handled correctly.
    """

    def __init__(self, dev_id=0):
        self.dev_id = dev_id
        self._sock = None
        self._libc = ctypes.CDLL(ctypes.util.find_library('c'), use_errno=True)
        self._lock = threading.Lock()
        self._evt_thread = None
        self._stop = threading.Event()
        self.bd_addr = "00:00:00:00:00:00"
        self.current_settings = 0
        self.link_keys = {}
        self.paired_event = threading.Event()

    def open(self):
        """Open management socket and ensure HCI device is up.

        Two-step process:
        1. ioctl(HCIDEVUP) — brings up the HCI transport (e.g. AIC8800 SDIO)
        2. ctypes bind to HCI_CHANNEL_CONTROL — opens the mgmt interface

        Once the mgmt socket is open, the kernel sets the HCI_MGMT flag on
        the device, which means ALL configuration MUST go through mgmt commands.
        Raw HCI commands for scan/SSP/etc. will be ignored or conflict.
        """
        # Bring HCI interface up first via ioctl
        ioctl_sock = socket.socket(AF_BLUETOOTH, socket.SOCK_RAW, BTPROTO_HCI)
        try:
            fcntl.ioctl(ioctl_sock.fileno(), HCIDEVUP, self.dev_id)
            log("MGMT", "hci%d UP" % self.dev_id)
        except OSError as e:
            if e.errno in (16, 114):  # EBUSY or EALREADY
                log("MGMT", "hci%d already UP" % self.dev_id)
            else:
                ioctl_sock.close()
                raise
        ioctl_sock.close()

        # Open management channel socket via ctypes bind.
        # Python's socket.bind((dev_id,)) hardcodes hci_channel=0 (raw HCI).
        # We need channel=3 (mgmt), so we build sockaddr_hci manually:
        #   struct sockaddr_hci { AF_BLUETOOTH(2), hci_dev(2), hci_channel(2) }
        # hci_dev=0xFFFF means "non-specific" — required for mgmt.
        self._sock = socket.socket(AF_BLUETOOTH, socket.SOCK_RAW, BTPROTO_HCI)
        addr = struct.pack('<HHH', AF_BLUETOOTH, HCI_DEV_NONE, HCI_CHANNEL_CONTROL)
        ret = self._libc.bind(self._sock.fileno(), addr, len(addr))
        if ret != 0:
            errno = ctypes.get_errno()
            raise OSError(errno, "mgmt bind failed: %s" % os.strerror(errno))
        log("MGMT", "Management socket opened")

    def _send_cmd(self, opcode, index, params=b''):
        """Send mgmt command and return (evt_code, response_data) or None."""
        hdr = struct.pack('<HHH', opcode, index, len(params))
        with self._lock:
            self._sock.send(hdr + params)

    def _recv(self, timeout=5.0):
        """Receive one mgmt event. Returns raw bytes or None on timeout."""
        r, _, _ = select.select([self._sock], [], [], timeout)
        if r:
            return self._sock.recv(4096)
        return None

    def _send_and_wait(self, opcode, index, params=b'', timeout=5.0):
        """Send mgmt command and wait for matching command complete."""
        hdr = struct.pack('<HHH', opcode, index, len(params))
        with self._lock:
            self._sock.send(hdr + params)
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                break
            r, _, _ = select.select([self._sock], [], [], min(remaining, 1.0))
            if not r:
                continue
            data = self._sock.recv(4096)
            if not data or len(data) < 6:
                continue
            evt_code = struct.unpack_from('<H', data, 0)[0]
            evt_idx = struct.unpack_from('<H', data, 2)[0]
            evt_plen = struct.unpack_from('<H', data, 4)[0]
            payload = data[6:6 + evt_plen]
            if evt_code == MGMT_EV_CMD_COMPLETE and len(payload) >= 3:
                resp_opcode = struct.unpack_from('<H', payload, 0)[0]
                status = payload[2]
                if resp_opcode == opcode:
                    return status, payload[3:]
                # Not our response, might be an event — handle it
                self._dispatch_event(evt_code, evt_idx, payload)
            elif evt_code == MGMT_EV_CMD_STATUS and len(payload) >= 3:
                resp_opcode = struct.unpack_from('<H', payload, 0)[0]
                status = payload[2]
                if resp_opcode == opcode:
                    return status, b''
                self._dispatch_event(evt_code, evt_idx, payload)
            else:
                # Async event, handle it
                self._dispatch_event(evt_code, evt_idx, payload)
        log_err("MGMT", "cmd 0x%04x timeout" % opcode)
        return None

    def setup(self, name="NanoKVM-BT"):
        """Configure BT controller for HID device operation.

        Order matters! SET_POWERED must come first.
        Without it, SSP/IO_CAP/DEV_CLASS return status 0x0F (Not Powered).

        Target state: 0xDB = powered+connectable+discoverable+bondable+ssp+br/edr
        """
        # Read controller info
        result = self._send_and_wait(MGMT_OP_READ_INFO, self.dev_id)
        if result is None:
            raise RuntimeError("Failed to read controller info")
        status, info = result
        if status != 0:
            raise RuntimeError("Read info failed: status=%d" % status)
        if len(info) >= 17:
            bd = info[0:6]
            self.bd_addr = ':'.join('%02x' % b for b in reversed(bd))
            # READ_INFO response layout (after opcode+status):
            #   offset 0:  BD_ADDR (6B)
            #   offset 6:  BT version (1B)
            #   offset 7:  manufacturer (2B LE)
            #   offset 9:  supported_settings (4B LE)
            #   offset 13: current_settings (4B LE)  ← NOT 14!
            #   offset 17: class_of_device (3B)
            #   offset 20: name (249B)
            self.current_settings = struct.unpack_from('<I', info, 13)[0]
            log("MGMT", "BD_ADDR: %s" % self.bd_addr)
            log("MGMT", "Current settings: 0x%08x" % self.current_settings)

        # --- Must power on first via mgmt (even if HCI is UP) ---
        r = self._send_and_wait(MGMT_OP_SET_POWERED, self.dev_id, bytes([0x01]))
        log("MGMT", "Powered: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set bondable (pairable) — can set while powered
        r = self._send_and_wait(MGMT_OP_SET_BONDABLE, self.dev_id, bytes([0x01]))
        log("MGMT", "Bondable: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set SSP mode
        r = self._send_and_wait(MGMT_OP_SET_SSP, self.dev_id, bytes([0x01]))
        log("MGMT", "SSP: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set IO Capability to NoInputNoOutput (Just Works)
        r = self._send_and_wait(MGMT_OP_SET_IO_CAP, self.dev_id,
                                bytes([MGMT_IO_CAP_NO_INPUT_NO_OUTPUT]))
        log("MGMT", "IO Cap: %s" % ("NoInputNoOutput" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set local name (249 bytes name + 11 bytes short_name)
        name_bytes = name.encode('utf-8')[:249].ljust(249, b'\x00')
        short_name = name.encode('utf-8')[:11].ljust(11, b'\x00')
        r = self._send_and_wait(MGMT_OP_SET_LOCAL_NAME, self.dev_id,
                                name_bytes + short_name)
        log("MGMT", "Name: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set device class — params: [major_byte, minor_byte]
        # These map directly to CoD byte 1 (major) and byte 0 (minor).
        #   major=0x05: Peripheral (bits 12-8 of CoD)
        #   minor=0xC0: bits 7-6 = 11 = Combo Keyboard+Pointing
        r = self._send_and_wait(MGMT_OP_SET_DEV_CLASS, self.dev_id,
                                bytes([0x05, 0xC0]))
        log("MGMT", "DevClass: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set connectable
        r = self._send_and_wait(MGMT_OP_SET_CONNECTABLE, self.dev_id, bytes([0x01]))
        log("MGMT", "Connectable: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Set discoverable (timeout=0 means infinite)
        disc_params = struct.pack('<BH', 0x01, 0x0000)  # enable + timeout=0
        r = self._send_and_wait(MGMT_OP_SET_DISCOVERABLE, self.dev_id, disc_params)
        log("MGMT", "Discoverable: %s" % ("OK" if r and r[0] == 0 else "status=%s" % (r,)))

        # Re-read final settings
        result = self._send_and_wait(MGMT_OP_READ_INFO, self.dev_id)
        if result and result[0] == 0 and len(result[1]) >= 17:
            self.current_settings = struct.unpack_from('<I', result[1], 13)[0]
            log("MGMT", "Final settings: 0x%08x" % self.current_settings)
            flags = {0: 'powered', 1: 'connectable', 3: 'discoverable',
                     4: 'bondable', 6: 'ssp', 7: 'br/edr'}
            active = [n for b, n in sorted(flags.items())
                      if self.current_settings & (1 << b)]
            log("MGMT", "Active: %s" % ' '.join(active))

    def start_event_loop(self):
        """Start background thread to handle mgmt events (pairing, etc.)."""
        self._evt_thread = threading.Thread(target=self._event_loop, daemon=True)
        self._evt_thread.start()
        log("MGMT", "Event loop started")

    def stop(self):
        """Stop event loop."""
        self._stop.set()

    def _event_loop(self):
        """Background loop: read and handle mgmt events."""
        while not self._stop.is_set():
            r, _, _ = select.select([self._sock], [], [], 0.5)
            if not r:
                continue
            try:
                data = self._sock.recv(4096)
            except OSError:
                continue
            if not data or len(data) < 6:
                continue
            evt_code = struct.unpack_from('<H', data, 0)[0]
            evt_idx = struct.unpack_from('<H', data, 2)[0]
            evt_plen = struct.unpack_from('<H', data, 4)[0]
            payload = data[6:6 + evt_plen]
            self._dispatch_event(evt_code, evt_idx, payload)
        log("MGMT", "Event loop stopped")

    def _dispatch_event(self, evt_code, index, payload):
        """Handle a single mgmt event."""
        if evt_code == MGMT_EV_USER_CONFIRM_REQ:
            # SSP User Confirmation Request.
            # With IO_CAP=NoInputNoOutput on our side and DisplayYesNo on PC,
            # the pairing method is "Just Works" — we must auto-confirm.
            # If we don't reply, pairing times out on the PC side.
            # payload: addr(6) + addr_type(1) + confirm_hint(1) + value(4)
            if len(payload) >= 12:
                addr = ':'.join('%02x' % b for b in reversed(payload[0:6]))
                addr_type = payload[6]
                confirm_hint = payload[7]
                value = struct.unpack_from('<I', payload, 8)[0]
                log("PAIR", "User Confirm Request from %s (value=%d, hint=%d)"
                    % (addr, value, confirm_hint))
                # Auto-confirm (Just Works)
                reply_params = payload[0:7]  # addr + addr_type
                self._send_cmd(MGMT_OP_USER_CONFIRM_REPLY, self.dev_id, reply_params)
                log("PAIR", "  -> Auto-confirmed")

        elif evt_code == MGMT_EV_PIN_CODE_REQUEST:
            if len(payload) >= 8:
                addr = ':'.join('%02x' % b for b in reversed(payload[0:6]))
                addr_type = payload[6]
                log("PAIR", "PIN Code Request from %s" % addr)
                # Reply with PIN '0000'
                pin = b'0000'
                reply = payload[0:7] + bytes([len(pin)]) + pin.ljust(16, b'\x00')
                self._send_cmd(MGMT_OP_PIN_CODE_REPLY, self.dev_id, reply)
                log("PAIR", "  -> Replied PIN '0000'")

        elif evt_code == MGMT_EV_NEW_LINK_KEY:
            if len(payload) >= 30:
                store_hint = payload[0]
                addr = ':'.join('%02x' % b for b in reversed(payload[1:7]))
                addr_type = payload[7]
                key_type = payload[8]
                link_key = payload[9:25]
                pin_len = payload[25]
                self.link_keys[addr] = link_key
                log("PAIR", "New Link Key from %s (type=%d, store=%d)"
                    % (addr, key_type, store_hint))

        elif evt_code == MGMT_EV_DEVICE_CONNECTED:
            if len(payload) >= 7:
                addr = ':'.join('%02x' % b for b in reversed(payload[0:6]))
                addr_type = payload[6]
                log("PAIR", "Device Connected: %s (type=%d)" % (addr, addr_type))
                self.paired_event.set()

        elif evt_code == MGMT_EV_DEVICE_DISCONNECTED:
            if len(payload) >= 7:
                addr = ':'.join('%02x' % b for b in reversed(payload[0:6]))
                log("PAIR", "Device Disconnected: %s" % addr)

        elif evt_code == MGMT_EV_AUTH_FAILED:
            if len(payload) >= 8:
                addr = ':'.join('%02x' % b for b in reversed(payload[0:6]))
                status = payload[7]
                log_err("PAIR", "Auth Failed: %s status=%d" % (addr, status))

        elif evt_code == MGMT_EV_NEW_SETTINGS:
            if len(payload) >= 4:
                new_settings = struct.unpack_from('<I', payload, 0)[0]
                self.current_settings = new_settings
                log("MGMT", "Settings changed: 0x%08x" % new_settings)

        elif evt_code in (MGMT_EV_CMD_COMPLETE, MGMT_EV_CMD_STATUS):
            pass  # Handled in _send_and_wait

        else:
            log("MGMT", "Event 0x%04x idx=%d plen=%d" % (evt_code, index, len(payload)))

    def close(self):
        """Shutdown: disable scan and close socket."""
        self.stop()
        if self._sock:
            try:
                # Set not discoverable
                self._send_cmd(MGMT_OP_SET_DISCOVERABLE, self.dev_id,
                               struct.pack('<BH', 0x00, 0x0000))
                time.sleep(0.1)
                # Set not connectable
                self._send_cmd(MGMT_OP_SET_CONNECTABLE, self.dev_id, bytes([0x00]))
                time.sleep(0.1)
            except Exception:
                pass
            self._sock.close()
            log("MGMT", "Closed")

# ---------------------------------------------------------------------------
# SDP Server
# ---------------------------------------------------------------------------

def build_sdp_hid_record():
    """Build HID SDP service record as raw SDP Data Element bytes.

    This record tells the PC that we are a HID device (keyboard+mouse combo).
    The PC fetches this via SDP query BEFORE establishing HID L2CAP connections.
    If the record is malformed or missing key attributes, the PC won't recognize
    us as a HID device even after successful pairing.

    Critical attributes:
    - 0x0001: ServiceClassIDList with HID UUID (0x1124)
    - 0x0004: ProtocolDescriptorList with L2CAP PSM for Control channel
    - 0x000D: AdditionalProtocolDescriptorLists with PSM for Interrupt channel
    - 0x0206: HIDDescriptorList containing the HID Report Descriptor
    """

    def uint8(v):
        return bytes([0x08, v])

    def uint16(v):
        return bytes([0x09]) + struct.pack('>H', v)

    def uint32(v):
        return bytes([0x0A]) + struct.pack('>I', v)

    def bool_val(v):
        return bytes([0x28, 0x01 if v else 0x00])

    def text(s):
        if isinstance(s, str):
            b = s.encode('utf-8')
        else:
            b = s
        if len(b) < 256:
            return bytes([0x25, len(b)]) + b
        else:
            return bytes([0x26]) + struct.pack('>H', len(b)) + b

    def uuid16(v):
        return bytes([0x19]) + struct.pack('>H', v)

    def seq(data):
        if len(data) < 256:
            return bytes([0x35, len(data)]) + data
        else:
            return bytes([0x36]) + struct.pack('>H', len(data)) + data

    attrs = b''

    # ServiceRecordHandle
    attrs += uint16(0x0000) + uint32(0x00010001)

    # ServiceClassIDList
    attrs += uint16(0x0001) + seq(uuid16(0x1124))

    # ProtocolDescriptorList
    attrs += uint16(0x0004) + seq(
        seq(uuid16(0x0100) + uint16(PSM_HID_CONTROL)) +
        seq(uuid16(0x0011))
    )

    # BrowseGroupList
    attrs += uint16(0x0005) + seq(uuid16(0x1002))

    # LanguageBaseAttributeIDList
    attrs += uint16(0x0006) + seq(
        uint16(0x656E) + uint16(0x006A) + uint16(0x0100)
    )

    # BluetoothProfileDescriptorList
    attrs += uint16(0x0009) + seq(seq(uuid16(0x1124) + uint16(0x0101)))

    # AdditionalProtocolDescriptorLists
    attrs += uint16(0x000D) + seq(seq(
        seq(uuid16(0x0100) + uint16(PSM_HID_INTERRUPT)) +
        seq(uuid16(0x0011))
    ))

    # ServiceName
    attrs += uint16(0x0100) + text("NanoKVM-BT HID")

    # ServiceDescription
    attrs += uint16(0x0101) + text("Keyboard and Mouse")

    # ProviderName
    attrs += uint16(0x0102) + text("NanoKVM")

    # HID attributes
    attrs += uint16(0x0200) + uint16(0x0100)  # HIDDeviceReleaseNumber
    attrs += uint16(0x0201) + uint16(0x0111)  # HIDParserVersion
    attrs += uint16(0x0202) + uint8(0xC0)     # HIDDeviceSubclass
    attrs += uint16(0x0203) + uint8(0x00)     # HIDCountryCode
    attrs += uint16(0x0204) + bool_val(True)  # HIDVirtualCable
    attrs += uint16(0x0205) + bool_val(True)  # HIDReconnectInitiate

    # HIDDescriptorList
    attrs += uint16(0x0206) + seq(seq(uint8(0x22) + text(HID_REPORT_DESCRIPTOR)))

    # HIDLANGIDBaseList
    attrs += uint16(0x0207) + seq(seq(uint16(0x0409) + uint16(0x0100)))

    # HIDProfileVersion
    attrs += uint16(0x020B) + uint16(0x0101)

    # HIDSupervisionTimeout
    attrs += uint16(0x020C) + uint16(0x0C80)

    # HIDNormallyConnectable
    attrs += uint16(0x020D) + bool_val(True)

    # HIDBootDevice
    attrs += uint16(0x020E) + bool_val(True)

    # HIDSSRHostMaxLatency
    attrs += uint16(0x020F) + uint16(0x0640)

    # HIDSSRHostMinTimeout
    attrs += uint16(0x0210) + uint16(0x0020)

    return seq(attrs)


class SDPServer(threading.Thread):
    """Minimal SDP server on L2CAP PSM 1."""

    SDP_SVC_SEARCH_ATTR_REQ = 0x06
    SDP_SVC_SEARCH_ATTR_RSP = 0x07
    SDP_SVC_SEARCH_REQ = 0x02
    SDP_SVC_SEARCH_RSP = 0x03
    SDP_SVC_ATTR_REQ = 0x04
    SDP_SVC_ATTR_RSP = 0x05
    SDP_ERROR_RSP = 0x01

    def __init__(self, bd_addr):
        super().__init__(daemon=True)
        self.bd_addr = bd_addr
        self._stop_event = threading.Event()
        self._record = build_sdp_hid_record()

    def stop(self):
        self._stop_event.set()

    def run(self):
        server = socket.socket(
            socket.AF_BLUETOOTH, socket.SOCK_SEQPACKET, socket.BTPROTO_L2CAP
        )
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind((self.bd_addr, PSM_SDP))
        server.listen(1)
        server.setblocking(False)
        log("SDP", f"Listening on PSM 0x{PSM_SDP:04x}")

        while not self._stop_event.is_set():
            r, _, _ = select.select([server], [], [], 0.5)
            if not r:
                continue
            try:
                client, addr = server.accept()
                log("SDP", f"Connection from {addr}")
                self._handle_client(client)
            except OSError as e:
                log_err("SDP", f"Accept error: {e}")

        server.close()

    def _handle_client(self, client):
        """Handle SDP queries from a single client."""
        client.setblocking(False)
        try:
            while not self._stop_event.is_set():
                r, _, _ = select.select([client], [], [], 1.0)
                if not r:
                    continue
                try:
                    data = client.recv(1024)
                except OSError:
                    break
                if not data:
                    break
                self._process_pdu(client, data)
        except Exception as e:
            log_err("SDP", f"Client handler: {e}")
        finally:
            client.close()

    def _process_pdu(self, client, data):
        if len(data) < 5:
            return
        pdu_id = data[0]
        txn_id = struct.unpack_from('>H', data, 1)[0]
        param_len = struct.unpack_from('>H', data, 3)[0]
        params = data[5:5 + param_len]

        log("SDP", f"PDU id={pdu_id}, txn={txn_id}, len={param_len}")

        if pdu_id == self.SDP_SVC_SEARCH_REQ:
            self._handle_svc_search(client, txn_id, params)
        elif pdu_id == self.SDP_SVC_ATTR_REQ:
            self._handle_svc_attr(client, txn_id, params)
        elif pdu_id == self.SDP_SVC_SEARCH_ATTR_REQ:
            self._handle_svc_search_attr(client, txn_id, params)
        else:
            # Error response
            err = struct.pack('>BHH H', self.SDP_ERROR_RSP, txn_id, 2, 0x0003)
            client.send(err)

    def _handle_svc_search(self, client, txn_id, params):
        """Respond to SDP ServiceSearch with our record handle."""
        # Return our service record handle
        handle = struct.pack('>I', 0x00010001)
        resp_params = struct.pack('>HH', 1, 1) + handle + bytes([0])  # continuation = 0
        resp = struct.pack('>BHH', self.SDP_SVC_SEARCH_RSP, txn_id, len(resp_params)) + resp_params
        client.send(resp)
        log("SDP", "  -> ServiceSearchResponse (1 handle)")

    def _handle_svc_attr(self, client, txn_id, params):
        """Respond to SDP ServiceAttribute with full record."""
        attr_list = self._record
        resp_params = struct.pack('>H', len(attr_list)) + attr_list + bytes([0])
        resp = struct.pack('>BHH', self.SDP_SVC_ATTR_RSP, txn_id, len(resp_params)) + resp_params
        client.send(resp)
        log("SDP", f"  -> ServiceAttributeResponse ({len(attr_list)} bytes)")

    def _handle_svc_search_attr(self, client, txn_id, params):
        """Respond to SDP ServiceSearchAttribute with full record."""
        # Wrap record in a sequence (list of attribute lists)
        record_list = self._record
        # Wrap in outer sequence
        if len(record_list) < 256:
            attr_lists = bytes([0x35, len(record_list)]) + record_list
        else:
            attr_lists = bytes([0x36]) + struct.pack('>H', len(record_list)) + record_list

        resp_params = struct.pack('>H', len(attr_lists)) + attr_lists + bytes([0])
        resp = struct.pack('>BHH', self.SDP_SVC_SEARCH_ATTR_RSP, txn_id, len(resp_params)) + resp_params
        client.send(resp)
        log("SDP", f"  -> ServiceSearchAttributeResponse ({len(attr_lists)} bytes)")

# ---------------------------------------------------------------------------
# HID Server
# ---------------------------------------------------------------------------

class HIDServer:
    """L2CAP HID server: accepts control + interrupt channels."""

    def __init__(self, bd_addr):
        self.bd_addr = bd_addr
        self._ctrl_server = None
        self._intr_server = None
        self._ctrl_client = None
        self._intr_client = None
        self._protocol_mode = 1  # 0=boot, 1=report

    def start_listening(self):
        """Bind and listen on HID Control + Interrupt PSMs."""
        self._ctrl_server = socket.socket(
            socket.AF_BLUETOOTH, socket.SOCK_SEQPACKET, socket.BTPROTO_L2CAP
        )
        self._ctrl_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            self._ctrl_server.setsockopt(SOL_BLUETOOTH, BT_SECURITY,
                                         struct.pack('<BBI', BT_SECURITY_LOW, 0, 0))
        except OSError:
            pass
        self._ctrl_server.bind((self.bd_addr, PSM_HID_CONTROL))
        self._ctrl_server.listen(1)

        self._intr_server = socket.socket(
            socket.AF_BLUETOOTH, socket.SOCK_SEQPACKET, socket.BTPROTO_L2CAP
        )
        self._intr_server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            self._intr_server.setsockopt(SOL_BLUETOOTH, BT_SECURITY,
                                         struct.pack('<BBI', BT_SECURITY_LOW, 0, 0))
        except OSError:
            pass
        self._intr_server.bind((self.bd_addr, PSM_HID_INTERRUPT))
        self._intr_server.listen(1)

        log("HID", f"Listening on PSM 0x{PSM_HID_CONTROL:04x} (Control) "
                    f"and 0x{PSM_HID_INTERRUPT:04x} (Interrupt)")

    def wait_for_connection(self, timeout=120):
        """Wait for both control and interrupt connections."""
        log("HID", "Waiting for HID connections...")
        deadline = time.monotonic() + timeout

        # Accept control channel first
        self._ctrl_server.setblocking(False)
        self._intr_server.setblocking(False)

        ctrl_connected = False
        intr_connected = False

        while time.monotonic() < deadline:
            socks = []
            if not ctrl_connected:
                socks.append(self._ctrl_server)
            if not intr_connected:
                socks.append(self._intr_server)
            if not socks:
                break

            r, _, _ = select.select(socks, [], [], 1.0)
            for s in r:
                if s is self._ctrl_server and not ctrl_connected:
                    self._ctrl_client, addr = self._ctrl_server.accept()
                    log("HID", f"Control channel connected from {addr}")
                    ctrl_connected = True
                elif s is self._intr_server and not intr_connected:
                    self._intr_client, addr = self._intr_server.accept()
                    log("HID", f"Interrupt channel connected from {addr}")
                    intr_connected = True

            if ctrl_connected and intr_connected:
                return True

        if not ctrl_connected or not intr_connected:
            log_err("HID", "Timeout waiting for HID connections")
            return False
        return True

    def handle_control_channel(self):
        """Handle incoming HID control messages (non-blocking, call in loop)."""
        if not self._ctrl_client:
            return
        self._ctrl_client.setblocking(False)
        try:
            r, _, _ = select.select([self._ctrl_client], [], [], 0)
            if r:
                data = self._ctrl_client.recv(256)
                if data:
                    self._process_control(data)
        except OSError:
            pass

    def _process_control(self, data):
        if not data:
            return
        msg_type = data[0] & 0xF0
        param = data[0] & 0x0F

        if msg_type == HID_SET_PROTOCOL:
            self._protocol_mode = param
            log("HID", f"SET_PROTOCOL: {'report' if param else 'boot'} mode")
            resp = bytes([HID_HANDSHAKE | HID_HANDSHAKE_SUCCESS])
            self._ctrl_client.send(resp)

        elif msg_type == HID_GET_PROTOCOL:
            log("HID", f"GET_PROTOCOL -> {self._protocol_mode}")
            resp = bytes([HID_DATA | 0x00, self._protocol_mode])
            self._ctrl_client.send(resp)

        elif msg_type == HID_SET_REPORT:
            log("HID", f"SET_REPORT: {data.hex()}")
            resp = bytes([HID_HANDSHAKE | HID_HANDSHAKE_SUCCESS])
            self._ctrl_client.send(resp)

        elif msg_type == HID_GET_REPORT:
            log("HID", f"GET_REPORT: {data.hex()}")
            resp = bytes([HID_HANDSHAKE | HID_HANDSHAKE_ERR_UNSUPPORTED_REQUEST])
            self._ctrl_client.send(resp)

        else:
            log("HID", f"Unknown control: 0x{data[0]:02x} {data.hex()}")

    def send_keyboard_report(self, modifier, keys):
        """Send keyboard input report on interrupt channel.
        modifier: bitmask (bit0=LCtrl, bit1=LShift, ...)
        keys: list of up to 6 key codes
        """
        if not self._intr_client:
            return False
        report = bytes([
            0xA1,              # DATA | Input
            REPORT_ID_KEYBOARD,
            modifier,
            0x00,              # reserved
        ]) + bytes(keys[:6]).ljust(6, b'\x00')
        try:
            self._intr_client.send(report)
            return True
        except OSError as e:
            log_err("HID", f"Send keyboard report failed: {e}")
            return False

    def send_mouse_report(self, buttons, dx, dy, wheel=0):
        """Send mouse input report on interrupt channel.
        buttons: bit0=left, bit1=right, bit2=middle
        dx, dy: -127..127
        wheel: -127..127
        """
        if not self._intr_client:
            return False
        dx = max(-127, min(127, dx))
        dy = max(-127, min(127, dy))
        wheel = max(-127, min(127, wheel))
        report = bytes([
            0xA1,           # DATA | Input
            REPORT_ID_MOUSE,
            buttons,
        ]) + struct.pack('bbb', dx, dy, wheel)
        try:
            self._intr_client.send(report)
            return True
        except OSError as e:
            log_err("HID", f"Send mouse report failed: {e}")
            return False

    def close(self):
        for s in (self._ctrl_client, self._intr_client,
                  self._ctrl_server, self._intr_server):
            if s:
                try:
                    s.close()
                except OSError:
                    pass

# ---------------------------------------------------------------------------
# Demo Test Sequence
# ---------------------------------------------------------------------------

def run_demo(hid: HIDServer):
    """Send test keyboard and mouse input."""

    log("DEMO", "=== Starting demo in 3 seconds... ===")
    log("DEMO", ">>> Please open a text editor on the PC and keep it focused <<<")
    time.sleep(3)

    # Type "hello"
    text = "hello"
    log("DEMO", f'Typing "{text}"...')
    for ch in text:
        keycode = CHAR_TO_KEYCODE.get(ch, KEY_NONE)
        if keycode == KEY_NONE:
            continue
        # Key press
        hid.send_keyboard_report(0x00, [keycode])
        time.sleep(0.05)
        # Key release
        hid.send_keyboard_report(0x00, [KEY_NONE])
        time.sleep(0.05)

    time.sleep(0.5)

    # Mouse: move right 100px in small steps
    log("DEMO", "Moving mouse right...")
    for _ in range(20):
        hid.send_mouse_report(0x00, 5, 0)
        time.sleep(0.02)

    time.sleep(0.3)

    # Mouse: left click
    log("DEMO", "Left click...")
    hid.send_mouse_report(0x01, 0, 0)  # press
    time.sleep(0.1)
    hid.send_mouse_report(0x00, 0, 0)  # release
    time.sleep(0.1)

    log("DEMO", "=== Demo complete! ===")

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    mgmt = None
    sdp = None
    hid = None

    def cleanup(signum=None, frame=None):
        log("MAIN", "Shutting down...")
        if hid:
            hid.close()
        if sdp:
            sdp.stop()
        if mgmt:
            mgmt.close()
        log("MAIN", "Done.")
        sys.exit(0)

    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)

    try:
        # --- Step 1: BT Management setup ---
        mgmt = MgmtController(dev_id=0)
        mgmt.open()
        mgmt.setup(name="NanoKVM-BT")

        # --- Step 2: Start mgmt event loop (handles pairing) ---
        mgmt.start_event_loop()

        # --- Step 3: Start SDP server ---
        sdp = SDPServer(mgmt.bd_addr)
        sdp.start()

        # --- Step 4: Start HID server ---
        hid = HIDServer(mgmt.bd_addr)
        hid.start_listening()

        print()
        print("=" * 52)
        print("  >>> Please open Bluetooth settings on your PC  <<<")
        print('  >>> Search and pair with "NanoKVM-BT"          <<<')
        print("=" * 52)
        print()

        # --- Step 5: Wait for connection ---
        if not hid.wait_for_connection(timeout=120):
            log_err("MAIN", "No HID connection established. Exiting.")
            cleanup()
            return

        print()
        print("=" * 52)
        print("  >>> Connected! Demo will start in 3 seconds    <<<")
        print("  >>> Open a text editor on the PC NOW            <<<")
        print("=" * 52)
        print()

        # Process any pending control messages
        for _ in range(10):
            hid.handle_control_channel()
            time.sleep(0.1)

        # --- Step 6: Run demo ---
        run_demo(hid)

        # Keep running briefly to handle any cleanup
        time.sleep(2)

    except Exception as e:
        log_err("MAIN", f"Fatal: {e}")
        traceback.print_exc()
    finally:
        cleanup()


if __name__ == "__main__":
    main()
