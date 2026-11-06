# Bluetooth HID experiment tools

`scripts/bt/` holds two target-only experiment tools: `mgmt_diag.py`
(read-only controller diagnostic) and `bt_hid_demo.py` (temporarily
configures the device as a Bluetooth keyboard+mouse and sends a short demo
after pairing). They require a Linux Bluetooth stack with HCI and L2CAP,
usually root or equivalent capabilities, and they are not integrated into
the product. The demo is not a production HID service.

## Rules

- Run the read-only diagnostic first, then decide about the demo.
- Do not run alongside services that share the same controller or L2CAP
  PSMs.
- Cleanup requests discoverable/connectable off but does not restore the
  original controller properties or power state; record the device state
  first and restore it afterwards.
- `SIGKILL` or an abnormal termination skips the cleanup handler.

See [README.md](../README.md) for the script inventory.
