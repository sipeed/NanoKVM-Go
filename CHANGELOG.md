# Changelog

Notable changes to NanoKVM-Go, with the newest version first.

## NanoKVM-Go Nomad 0.2.7, build 1

- Share a browser camera with the Go+ USB host as a 1024 × 600 MJPEG webcam. The device setting is optional and persists across restarts. Each browser session starts sharing explicitly. The USB output returns to black when the publisher stops.
- Calibrate on Start with one second of warmup and five two-second samples. Auto starts at 5 Mbps and 30 fps, searches within 2–8 Mbps and 15–30 fps, and continues adjusting during use. The menu shows live throughput, frame rate and upload delay during calibration and streaming, and the calibrated target when ready. [Webcam operation](docs/remote-webcam.md)
- Support worker-based capture and a Safari/WebKit fallback. Add a configurable host-visible webcam name and USB endpoint checks. The Normal offline update bundles the UVC kernel and application for Go+ firmware v1.0.10.
- Add webcam and one-click Mouse Jiggler toolbar controls, active-state colors, configurable shortcut visibility and a localized jiggler tour step. Keep the jiggler shortcut hidden on a fresh toolbar.
- Identify the fork and build number in the dashboard and update UI. Use Sipeed’s timezone and EDID defaults and preserve configured values on upgrade.
- Package the webcam feeder separately from the Memory Fabric build and set the link stub’s library name explicitly for the ARM server build.

## 0.2.7 - 2026-11-05

- Support custom HTTP/HTTPS update servers with persistent settings and stable and preview channels.
- Add a localized update-server switch and editor, and prevent online installation while server settings have unsaved changes.
- Report update check failures separately from installation failures.
- Preserve spaces in OCR recognition results.

## 0.2.6 - 2026-11-04

- Remove the speaker_read MCP tool; agents transcribe audio from their WebRTC media session.
- Update frontend dependencies.

## 0.2.5 - 2026-10-30

- Add virtual media management and file sharing with file browsing and transfers, and restore local write access after USB host ejection.
- Add FRP, Cloudflare Tunnel, ZeroTier, and NetBird remote-access extensions with shared controls and two-factor protection, and fix NetBird restart crashes.
- Add on-device network setup with privacy consent, verification codes, and pairing status, and streamline Tailscale provider switching.
- Support WPA3/SAE Wi-Fi, including short and UTF-8 passwords, while preserving WPA2 compatibility.
- Add one-scan Wi-Fi setup from camera and screenshot QR codes.
- Add replayable onboarding tours for desktop and mobile users.
- Add picture-in-picture and preserve video sessions across layout changes, with clearer WebRTC fallback guidance.
- Extend two-factor protection to Web Terminal and Finger Robot press and arm-replacement actions.
- Coordinate local and USB storage ownership and keep keyboard and mouse HID working across storage conflicts.
- Enforce image upload size and free-space limits, and improve failure feedback for uploads, scripts, WoL history, and timelapse deletion.
- Reduce memory and CPU use, and refresh the device clock after timezone changes without restarting.
- Require firmware v1.0.10.

## 0.2.4 - 2026-09-24

- Persist Sipeed network consent and remove region restrictions for Sipeed Tailscale mode.
- Refresh remote-network feature settings and stabilize device pairing.
- Load Screen Timelapse playback on demand.
- Stop polling disconnected Finger Robot hardware while idle.
- Automatically dismiss the bandwidth warning after two seconds.

## 0.2.3 - 2026-09-23

- Add Sipeed-managed remote networking with pairing and device management.
- Improve Tailscale login, network cleanup, and mode switching.
- Add confirmed Finger Robot arm replacement and improve servo feedback.
- Improve AP Wi-Fi provisioning with scan-first setup and mobile-friendly network lists.
- Align MCP screenshots and pointer coordinates, and stabilize pointer capture and release.
- Improve Wi-Fi scanning, SSID scrolling, and screenshots on the device screen.

## 0.2.2 - 2026-09-04

- Keep pointer input aligned with video under Safari page zoom and scaled layouts.
- Restore the floating toolbar after orientation changes and align input calibration controls.
- Allow deleting the current Screen Timelapse day while protecting active recordings from automatic cleanup.
- Fix Screen Timelapse playback and search-result layouts on mobile screens.

## 0.2.1 - 2026-09-04

- Keep the pointer mode in sync between the browser and the device screen.
- Improve desktop and Portal viewport behavior, mobile safe areas, and interaction without a video signal.
- Improve Screen Timelapse recording controls, retention handling, and recording quality.
- Add Wi-Fi preset management with persistent boot fallback and improve touch responsiveness.

## 0.2.0 - 2026-09-01

- Add per-session input with multitouch and WebHID support, and recover from input lag.
- Add automatic and four-point input calibration with shared profiles and Portal crop controls.
- Integrate USB network controls into Virtual Devices and improve iPhone home-screen guidance.
- Validate EDID uploads, add the 4K50 preset, and stabilize virtual audio and EDID switching.
- Improve Finger Robot reliability.
- Improve video-mode transitions, Direct-mode audio, terminal, and fullscreen behavior.

## Earlier releases

See the Debian changelogs for the [application](debhelper/packages/nanokvm-go/debian/changelog)
and [device services](debhelper/packages/kvmcomm/debian/changelog).
