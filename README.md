# NanoKVM-Go

English | [中文](README.zh.md)

NanoKVM-Go is the open-source software stack for the **NanoKVM Go series**, a
pocket-sized USB-C KVM from [Sipeed](https://sipeed.com) built for phones,
tablets, and portable computers.

After the initial network setup, connect it to a device that outputs video over
USB-C and control that device from a browser on another phone or computer —
screen, clicks, swipes, and typing. Because video is captured and input is
emulated in hardware, the target needs no driver or remote-control client. It
keeps working when the operating system is hung, offline, or not yet installed,
so it is also useful for UEFI/BIOS setup, OS installation, and recovery.

> Product documentation, quick start, and the full compatibility list live on
> the [Sipeed wiki](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/introduction.html).

## Models

The series has two models. Both share the same KVM, MCP, wireless, and extension
capabilities; Go+ adds an on-device NPU and Memory Fabric.

| | NanoKVM Go | NanoKVM Go+ |
| --- | --- | --- |
| Processor | Dual-core Cortex-A53 | Dual-core Cortex-A53 + 3.2 TOPS NPU |
| Memory | 256 MB LPDDR4x | 512 MB LPDDR4 |
| Storage | 16 GB | 64 GB |
| Display | 1.83" color touchscreen | 1.83" color touchscreen |
| Max capture | 4K @ 50 Hz; 2K @ 90 Hz | 4K @ 50 Hz; 2K @ 90 Hz |
| Typical latency | ~60 ms @ 1080p60, ~80 ms @ 2K60, ~100 ms @ 4K30 | same |
| Wireless | Wi-Fi 6, 2.4/5 GHz, up to 286 Mbps | same |
| Virtual devices | Keyboard, mouse, USB drive, virtual NIC | same |
| Ports | USB-C data (DP Alt Mode), USB-C power (USB-PD) | same |
| Size / power | ~45 × 40 × 15 mm, ~1.6 W @ 4K30 | ~45 × 40 × 15 mm, ~2.0 W @ 4K30 |
| Memory Fabric | — | Supported |

## Features

- **Browser-based control** — H.264 and H.265 over WebRTC, with a direct H.264
  stream fallback, plus bidirectional audio.
- **Input without an agent** — absolute and relative mouse, digitizer stylus, and
  multi-touch, with per-session route ownership across concurrent clients.
- **Virtual media and networking** — mount ISO/IMG images for remote OS
  installation, share files with the target, and expose a virtual NIC.
- **Local touchscreen UI** — a C++20/LVGL interface running directly on
  framebuffer and evdev for Wi-Fi, display, and app settings.
- **AI integration** — an optional MCP server so MCP-compatible agents can read
  the screen and send input; Go+ additionally provides Memory Fabric, which
  extracts searchable text and context from screen changes.
- **Remote access** — built-in Tailscale, plus ZeroTier, NetBird, frp, and
  Cloudflare Tunnel extensions, so the management port need not be exposed.
- **Security** — TOTP two-factor authentication and Ed25519-signed application
  and firmware updates.
- **Finger robot** — drive the target's power button remotely over the expansion
  IO on the USB-PD port.

MCP and Memory Fabric are off by default and must be enabled by the user.
Captured screens can contain accounts, messages, or other sensitive data, so use
them only on trusted networks with trusted AI clients.

## Repository layout

This repository is the build and packaging entry point for the series. Each
component is built into `build/` and then packaged into Debian packages.

| Path | Contents |
| --- | --- |
| [`server/`](server) | Go backend: API, authentication, HID, streaming, extensions |
| [`web/`](web) | React + TypeScript web UI |
| [`kvm_ui/`](kvm_ui) | Local LVGL touch UI for the device screen |
| [`kvm_vin/`](kvm_vin) | Video capture and encoding pipeline |
| [`libkvm/`](libkvm) | Hardware abstraction library |
| [`nano_ocr/`](nano_ocr) | OCR runtime |
| [`memfab/`](memfab) | Memory Fabric runtime for OCR-backed local memory |
| [`debhelper/`](debhelper) | Debian packaging for `kvmcomm` and `nanokvm-go` |
| [`support/`](support) | Toolchain setup, packaging scripts, kernel patches, release keys |
| [`docs/`](docs) | Architecture, API, and build documentation |

The `debhelper/packages/kvmcomm/rootfs/kvmcomm/apps` directory is a Git
submodule; clone with `--recurse-submodules`, or run
`git submodule update --init --recursive` after cloning.

## Build

The root-level Python driver is the official entry point. Build and packaging
commands run from the repository root.

Prerequisites:

| Component | Requirements |
| --- | --- |
| `server`, `memfab` | Go 1.26.5+, Python 3, `patchelf`, configured ARMv7 toolchain |
| `web` | Node.js `^20.19.0` or `>=22.12.0`, pnpm |
| `libkvm`, `kvm_vin`, `kvm_ui`, `nano_ocr` | Python 3, CMake, Conan, configured ARMv7 toolchain |
| Debian packaging | `dpkg-dev`, `debhelper` |
| Signed releases | OpenSSL with Ed25519 `pkey` and `pkeyutl -rawin` support |

Set up the ARMv7 toolchain first:

```bash
cd support/scripts
./toolchain_setup.sh
cd ../..
```

Install the frontend dependencies from the repository root, then build:

```bash
pnpm --dir web install --frozen-lockfile
python3 build.py build        # clean, then build every component
python3 build.py server       # or build a single component
python3 build.py release --version <version>
```

`nano_ocr` downloads OpenCV 4.6.0 from the official repository and verifies its
SHA-256, so the first build needs network access. FFmpeg is loaded from the
target system and is not part of this repository.

See **[docs/build-packaging.md](docs/build-packaging.md)** for toolchain
details, Debian packaging, update bundles, and release signing.

## Documentation

| Document | Description |
| --- | --- |
| [Build and Packaging Guide](docs/build-packaging.md) | Toolchain setup, component builds, Debian packaging, update bundles, signing |
| [Changelog](CHANGELOG.md) | Notable changes and release history |
| [MCP Usage](docs/mcp-usage.md) | Enable and use the MCP server with AI clients |
| [Storage and File Sharing](docs/storage.md) | Virtual media, ISO/IMG mounting, and file transfer API |
| [Input Router Architecture](docs/input-router.md) | Input routing, session ownership, and the tested client matrix |
| [USB HID Notes](docs/kvmgo-device-usb-hid.md) | HID report descriptors for mouse, stylus, and multi-touch |
| [Two-Factor Authentication](docs/2fa.md) | TOTP setup and runtime files |

## Compatibility

The target must expose video over a USB-C port that supports **DisplayPort Alt
Mode**; ports limited to charging or plain USB data cannot carry video. Apple
devices work only in the model ranges listed on the wiki (for example iPhone 15
and later, with AssistiveTouch enabled). The controller side only needs a modern
browser that can reach the device.

See the [compatibility list](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/introduction.html)
on the wiki for the complete, current tables.

## Contributing

Issues and pull requests are welcome. Please run the relevant component builds
and tests before submitting a change, and keep commit messages focused on a
single change.

## License

NanoKVM-Go is released under the [GNU General Public License v3.0](LICENSE).
