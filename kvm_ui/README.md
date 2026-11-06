# NanoKVM Go KVM UI

`kvm_ui` is a local display and touch UI for embedded Linux, written in
C++20 with LVGL 9.2 and running directly on fbdev/evdev. It drives the
NanoKVM device screen (Main, Settings, Wi-Fi, Panel, Apps pages) and talks
to device state, a small HTTP API, and hardware control through its runtime
service. This directory is a standalone project boundary.

## Hardware and display

- Physical framebuffer 284x240 (RGB565); normal visible canvas 270x240;
  the left 14 physical pixels are a dead area.
- Default devices: `/dev/fb0` for display, `/dev/input/event0` for touch.
- Panel: ST7789P3 driven through fbtft.

## Quick start

Prepare Conan 2, an ARM cross toolchain, and a local host profile first
(see [profile/README.md](profile/README.md)). Common commands:

```bash
./builder.sh help        # command list
./builder.sh build       # product release (may download dependencies)
./builder.sh build debug # RelWithDebInfo with debug diagnostics
./builder.sh minibuild   # incremental, only when build/ is configured
ctest --test-dir build --output-on-failure
```

A full build can clone the pinned LVGL checkout and download Conan
packages; do not trigger it just to inspect sources or docs.

## Simulator

`./builder.sh simulator` builds a native x86_64 Linux SDL version of the UI for
desktop work without NanoKVM hardware. It uses deterministic in-memory mocks,
never binds real HTTP ports, and never touches device services or files. The
`simulator_fetch_lvgl`, `simulator_fetch_conan`, `simulator_minibuild`,
`simulator_run`, `simulator_test`, and `simulator_clean` subcommands keep their
own build and state directories, separate from the ARM build. The window shows
the 270x240 logical canvas at 2x zoom and does not emulate the physical panel
or its left dead strip; hardware behavior still needs device validation.

## Runtime configuration

At startup `kvm_ui` loads `/etc/kvm/kvm_ui.toml`. A missing file is created
with complete defaults, invalid fields are repaired to defaults where
possible, and a file with broken TOML syntax is backed up and replaced. Some
fields are read only at process start and require a restart.

## Local HTTP API

`kvm_ui` can expose a small HTTP API for status pushes from device services.
It binds `127.0.0.1:65501` by default; binding a non-loopback address requires
a Bearer token. The API has no TLS, so keep it on loopback or protect it at the network level.
Authentication failures and malformed bodies return JSON error bodies. Three endpoints
under `/api/v1` accept JSON objects: HID state (`POST /api/v1/hid/state`,
demonstrated by `scripts/post_hid_state_demo.sh`), display labels, and
background image selection. The API is disabled in the simulator build.

## System information

The More page shows hardware, network, and build details collected on
intervals. Identifiers such as UID and MAC address, and network addresses
such as IP and SSID, are masked before display; credentials are never
recorded.

## Further reading

- [scripts/README.md](scripts/README.md) — device diagnostic scripts.
- [scripts/bt/README.md](scripts/bt/README.md) — Bluetooth experiments.
- [fonts/README.md](fonts/README.md) — font catalog kit.
