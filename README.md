# NanoKVM-Go Nomad

[中文](README.zh.md) · [Download a build](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest) · [Installation](#install-a-nomad-build) · [Webcam setup and calibration](docs/remote-webcam.md) · [Changelog](CHANGELOG.md)

## About

Nomad is a fork of Sipeed’s NanoKVM-Go for digital nomads who leave a workstation connected to a **NanoKVM Go+** and use it from a browser while traveling. The workstation sends its screen to you. Nomad also sends your laptop or phone camera in the other direction, so you can join a call on the workstation using the camera you have with you.

This repository contains the source and offline builds for the **Go+ running firmware v1.0.10**. The comparison below uses Sipeed’s 0.2.7 codebase as its starting point. [Official Sipeed project](https://github.com/sipeed/NanoKVM-Go)

## New Features

- **USB webcam sharing.** Present a browser camera to the workstation as a 720p USB webcam. The stream adapts to the available upload bandwidth and reports live delivery statistics.
- **Target keyboard layouts.** Choose from 11 layouts for dashboard Paste and MCP text entry, including Canadian French, Canadian Multilingual Standard and Microsoft Pinyin.
- **USB endpoint management.** See how many of the Go+’s seven USB IN endpoints are available. The mouse uses one endpoint by default.

## Improvements

- **Menu bar.** Click an icon for its everyday action and hover for secondary controls. Active icons show what is running. Screen brings view-only mode, cropping and EDID selection together.
- **Activate on Connect.** Set up each browser to start audio playback, microphone sharing, webcam sharing or Mouse Jiggler independently when it connects.
- **Mouse Jiggler.** Sipeed’s built-in jiggler gains a toolbar toggle, a countdown and different movement depending on whether someone is viewing the desktop.
- **Display profiles.** Each of Sipeed’s five display modes now advertises a distinct monitor identity. The selected EDID supplies the name and timing together.
- **USB device names.** Advanced settings let you name the webcam and USB configuration. The composite device, audio, storage and camera have updated defaults.
- **Offline updates.** A single Nomad bundle installs the application and matching webcam-enabled kernel. The dashboard and update screen show the Nomad build.

## Bug Fixes

- **Windows device detection.** The five stock EDIDs shared one manufacturer and product ID. Nomad gives each profile its own monitor ID and model name for Windows to read. The webcam also advertises its own USB interface name. Windows can display a separately stored friendly name for that device. [How Windows chooses the display name](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/devpkey-name--device-instance-)
- **Chrome viewer resizing.** Opening, closing or dragging Chrome’s side panel resizes the remote screen to fit the available width. The viewer previously mistook the width change for page zoom.
- **Input with multiple viewers.** Switching the USB pointer layout briefly reconnects USB. Other open dashboards now pick up the new layout and resume sending input.
- **Webcam startup and recovery.** The dashboard reports when the Go+ receives no frames during startup. Bitrate can recover on an improving link even if the source camera has reached its frame-rate limit.
- **USB configuration recovery.** If the host fails to configure the webcam after a switch, the Go+ restores its base USB profile. Failed USB identity changes roll back the saved settings.
- **Menu auto-hide.** Closing Settings with the mouse resumes the five-second hide timer when the pointer is outside the bar. Keyboard focus on the bar keeps it visible.

## Details

### Webcam and microphone

1. Open the dashboard over HTTPS. Under **Settings → Device**, turn off USB display mode, check that two USB IN endpoints are free, then enable **Webcam**. Eject and detach mounted virtual media first. Changing the device setting briefly reconnects USB.
2. Click **Webcam** to share your default camera, or open its menu to choose a source. Grant camera access in the browser. One browser can publish at a time. Use **Microphone** separately if the call also needs your voice.
3. Select the USB camera in the workstation’s calling app. The browser sends JPEG frames over a secure WebSocket. The Go+ presents them as a standard **1280 × 720 MJPEG USB Video Class (UVC) camera at up to 30 fps**.

Calibration starts at **8 Mbps and 25 fps**. After a one-second warmup, the browser samples five two-second windows. While streaming, it adjusts JPEG bitrate and frame rate within **2–10 Mbps** and **15–30 fps** using missed upload opportunities, queue growth and upload delay. The menu shows received bitrate, acknowledged frames per second and upload delay during calibration and streaming. **Ready** shows the calibrated targets while the workstation is idle. [Webcam controller and statistics](docs/remote-webcam.md)

Browsers that support background workers can capture and encode frames in a worker. Safari and other WebKit browsers can use a video element for capture. Stopping sharing or closing the dashboard releases your camera and returns the USB video to black. The Go+ remembers its Webcam device setting across reboots. Turn that setting off to free its two endpoints.

### Menu bar and connection preferences

| Control | Click the icon | Open the menu |
| --- | --- | --- |
| Screen | Toggle view-only Preview. A red eye means remote input is paused. | Select an EDID or change cropping. |
| Volume | Mute or restore the last audible level. | Adjust the playback volume. |
| Webcam | Start or stop sharing your camera. | Choose a source or check stream statistics. |
| Microphone | Start or stop sharing your microphone. | — |
| Mouse Jiggler | Start or stop movement using the last mode. | Set the runtime or read the countdown. |
| Terminal | Open the Go+ terminal in a new tab. | — |

Hover to open a menu with a mouse, or tap its arrow on a touchscreen. Under **Settings → Appearance**, **Show icon** controls shortcut visibility on the Go+. **Activate on Connect** saves a separate preference in each browser for Volume, Microphone, Webcam and Mouse Jiggler. A hidden shortcut can still start on connect. If you hide an active camera or microphone, its Appearance row offers **Stop sharing**. Using a shortcut during a session leaves the next connection’s preference intact.

### Mouse Jiggler

Enable **Show icon** for Mouse Jiggler under **Settings → Appearance** to add it to a fresh menu bar. Choose relative or absolute movement under **Settings → Device → Mouse Jiggler**. The toolbar menu accepts whole hours, with **8 hours** as the default and **0** for unlimited. Changing the limit during a run restarts its countdown. The Go+ owns the timer, so the run continues after the browser closes and keeps its deadline across a reboot.

| Desktop connection | Movement while idle |
| --- | --- |
| A viewer is connected, or the last viewer left less than five minutes ago | A short nudge after 30–60 seconds without KVM input. New input restarts the idle wait. |
| All viewers have been away for at least five minutes | Smooth 5–8 second curves steer inward near estimated screen edges. Reconnecting interrupts a curve in progress. |

### Screen, cropping and EDID

**Screen → Cropping** changes the part of the incoming picture you see. **Auto** detects black borders, **Manual** lets you draw and save a region or select a resolution, and **Off** restores the full frame. Absolute pointer input follows the visible content.

**Screen → EDID** and **Settings → Screen → EDID** change what the workstation sees as its connected monitor. Choose the resolution and refresh rate in either menu. The profile carries the corresponding manufacturer, product ID and model name.

| Display mode | Monitor reported by the Go+ |
| --- | --- |
| 1920 × 1080 at 60 Hz | Dell U2414H |
| 2560 × 1440 at 60 Hz | Dell S2716DG |
| 3440 × 1440 at 60 Hz | Dell U3415W |
| 3840 × 2160 at 30 Hz | Dell P2415Q |
| 3840 × 2160 at 50 Hz | Acer XV273K |

To use a different monitor identity, upload an EDID `.bin` under **Screen → EDID** and select it. An upgrade preserves the selected profile and retires the old standalone **Monitor Name** override.

### Keyboard and MCP

Under **Keyboard → Keyboards**, select the layout active on the workstation and leave Caps Lock off. The submenu shows the current selection and includes **Show virtual keyboard**. The Go+ saves one target layout for both **Keyboard → Paste** and the MCP tool `type_text`. For example, selecting **French → Canadian Multilingual Standard** makes both send the keystrokes for accented text on that layout.

| Language | Target layouts |
| --- | --- |
| English | US (default), UK |
| French | Canada, Canadian Multilingual Standard, France |
| German and Spanish | Germany, Spain |
| Chinese | Simplified Chinese · Microsoft Pinyin |
| Russian | Standard Windows ЙЦУКЕН |
| Persian | Persian, Persian (Standard, ISIRI 9147) |

Both text-entry paths accept up to 1,024 characters per request and validate the entire string before typing. An unsupported character produces an error before the first keystroke. A newline sends Enter and a tab sends Tab. The Pinyin profile requires Microsoft Pinyin in Chinese mode with Full Pinyin and V-mode enabled on the workstation. It can enter mixed Chinese and Latin text using printable characters in the Unicode Basic Multilingual Plane.

Sipeed provides the optional MCP service for screen capture and input. Enable **Settings → AI → MCP Service** and connect your client to `https://<device>/api/mcp` with the generated API key. Nomad’s `type_text` uses the saved layout. Use `press_keys` for physical key combinations. Memory and OCR tools require the separate **Memory Fabric** setting. [MCP setup, tools and examples](docs/mcp-usage.md)

### USB devices and pointer modes

**Settings → Device** shows used and free USB IN endpoints. By default, the keyboard and mouse use one endpoint each, leaving five available. The webcam needs two, virtual audio needs one, and a virtual network interface needs two, so all three fit together. Turning off **Mouse → Mouse mode → Use one USB pointer endpoint** adds a second pointer endpoint.

In single-endpoint mode the Go+ exposes only the selected pointer. Switching layouts affects all viewers. Mouse Jiggler uses the available mouse and is unavailable in stylus-only or touch-only layouts. [USB pointer modes](docs/kvmgo-device-usb-hid.md)

Under **Settings → Device → Advanced**, the composite USB device and audio function default to **Logitech P710e**, storage uses Logitech drive names, and **USB Webcam Name** defaults to **Anker Powerconf C200**. Saved overrides take precedence. The webcam name takes effect the next time Webcam is enabled or the Go+ restarts. **USB Configuration Name** is optional.

### Install a Nomad build

1. Check the Go+ firmware requirement in the release notes and download the matching offline update. Leave the `.tar.gz` compressed. [Latest Nomad release](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. Open **Settings → Check for Updates → Offline update** and upload the bundle. For an unsigned bundle, temporarily turn off **Update signature verification**.
3. Reboot the Go+, restore signature verification if you disabled it, and confirm the Nomad build on the update screen.

The installer checks the model, firmware and boot layout. When replacing the kernel, it saves a backup of the previous image. [Build and packaging guide](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed user guide](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html) · [License](LICENSE)
