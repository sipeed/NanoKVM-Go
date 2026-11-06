# Changelog

Notable changes to NanoKVM-Go, with the newest version first.

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
