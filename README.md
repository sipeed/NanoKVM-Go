# NanoKVM-Go Nomad

A NanoKVM Go+ fork for digital nomads. Use your laptop’s webcam in a call on the remote computer you control through the Go+.

[How Nomad’s controller works](docs/remote-webcam.md) · [Install build](#install-build) · [Official Sipeed project](https://github.com/sipeed/NanoKVM-Go)

## Nomad features

- **Webcam as a USB device**
  - **720p (1280 × 720), targeting 15–30 fps and 2–10 Mbps.** Nomad measures delivery during startup, then lowers quality when uploads fall behind and raises it as the link recovers. [Google Congestion Control background](https://datatracker.ietf.org/doc/html/draft-ietf-rmcat-gcc-02)
  - **Motion JPEG (MJPEG).** Your browser compresses each camera frame as a JPEG. The Go+ sends those frames to the remote computer as a standard USB webcam stream.
  - **Start a call.** Open the dashboard over HTTPS. Enable **Settings → Device → Webcam**, then click the **Webcam** icon to start sharing. Hover over it or tap its menu arrow to choose a camera. Microphone sharing has its own control.
  - **Webcam name.** Change **Settings → Device → Advanced → USB Webcam Name** to set the name reported to the host.
- **Mouse Jiggler in the menu bar.** Hidden by default. Enable the shortcut under **Settings → Appearance**, then click it to toggle movement using your last mode. Set **max runtime** in whole hours in the jiggler menu. The default is 8 hours. Set it to **0** to run indefinitely. Changing the limit while the jiggler is on starts a fresh countdown for the selected number of hours. After all desktop viewers have been disconnected for more than five minutes, each idle movement follows a random curve for 2–5 seconds. Reconnecting or sending keyboard or mouse input interrupts the curve.
- **Activity indicators.** The menu bar shows when the microphone, webcam or jiggler is active.

**Defaults**

- A stock Go+ starts with keyboard, mouse, absolute pointer and USB networking enabled. Webcam, Virtual Disk and USB audio are optional.
- Upgrades preserve your monitor EDID, timezone, USB identity, Webcam setting and saved toolbar layout.

## USB limits

The Go+ has **7 USB IN endpoints**. The keyboard and two pointer interfaces use **1 each**, leaving **4** for optional devices.

| Feature | IN endpoints |
| --- | ---: |
| Webcam | 2 |
| Virtual Audio (mic only or mic + speakers) | 1 |
| Virtual Network (NCM or RNDIS) | 2 per interface |
| Virtual Disk or mounted image | 1 per USB drive |
| USB serial (ACM) | 2 |

- **Webcam + Network** fills all 7. Adding Audio or Disk exceeds the limit.
- **Webcam + Audio** uses 6. One disk fits the software check. Using all three together still needs a host test.
- **Switching devices.** Eject and detach mounted media before changing Webcam. Turn **Settings → Device → Webcam** off to free its endpoints. Click the toolbar webcam icon again to stop sharing.

## Install build

1. Download the offline update bundle (`.tar.gz`) from the [latest Nomad release](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest). Leave it compressed.
2. Check the firmware requirement in the release notes. If your Go+ needs a firmware update, install the specified Sipeed version first. [Official firmware](https://github.com/sipeed/NanoKVM-Go/releases) · [Flashing guide](https://en.wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/system/flashing.html)
3. Open **Settings → Check for Updates**. For an unsigned release, temporarily turn off **Update signature verification**. Choose **Offline update → Select bundle → Upload and install**.
4. Once installation finishes, reboot the Go+, re-enable signature verification if needed, and confirm the installed build on the update page.

[Build from source](docs/build-packaging.md) · [License](LICENSE)
