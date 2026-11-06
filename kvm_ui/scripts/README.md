# Device integration and diagnostic scripts

`scripts/` contains target hardware/API diagnostics and demo tools. They
are not Launcher Apps and are not integrated into the `kvm_ui` binary.

## Tools

- `post_hid_state_demo.sh` — pushes a short HID state demo to
  `POST /api/v1/hid/state` and clears the state when it finishes (see the
  local HTTP API section of [README.md](../README.md)).
- `touch/raw_touch_board.py` — draws raw framebuffer and records raw
  evdev coordinates.
- `bt/mgmt_diag.py` and `bt/bt_hid_demo.py` — Bluetooth experiments (see
  [bt/README.md](bt/README.md)).

## Safety rules

- The raw touch board contends with the UI for `/dev/fb0` and
  `/dev/input/event*`: stop or isolate the UI service before running it
  and restore the UI afterwards.
- HTTP tokens, Wi-Fi passwords, and full credential QR payloads never go
  into logs or commits.
- Examples use `<device-ip>` placeholders; one device's address, kernel,
  or software versions are not general facts.
- Hardware-facing behavior can only be validated on the target device.

See [README.md](../README.md) for the project overview.
