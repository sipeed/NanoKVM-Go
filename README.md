# NanoKVM-Go Nomad

A NanoKVM Go+ fork for digital nomads. Use your laptop’s webcam in a call on the remote computer you control through the Go+.

[How Nomad’s controller works](docs/remote-webcam.md) · [Install build](#install-build) · [Official Sipeed project](https://github.com/sipeed/NanoKVM-Go)

## Nomad features

- **Webcam as a USB device**
  - **720p (1280 × 720), targeting 15–30 fps and 2–10 Mbps.** Calibration starts at 8 Mbps and 25 fps. During a call, Nomad lowers bitrate and frame rate together when uploads fall behind and raises them as the link recovers. [Google Congestion Control background](https://datatracker.ietf.org/doc/html/draft-ietf-rmcat-gcc-02)
  - **Motion JPEG (MJPEG).** Your browser compresses each camera frame as a JPEG. The Go+ sends those frames to the remote computer as a standard USB webcam stream.
  - **Start a call.** Open the dashboard over HTTPS. Enable **Settings → Device → Webcam**, then click the **Webcam** icon to start sharing. Hover over it or tap its menu arrow to choose a camera. Microphone sharing has its own control.
  - **Webcam name.** Change **Settings → Device → Advanced → USB Webcam Name** to set the name reported to the host.
- **EDID profiles.** Build 2 updates the five built-in display profiles. Choose a profile under **Settings → Screen → EDID**. The Go+ reports both the monitor name and the supported display modes from that profile, keeping them consistent when you switch. To simplify setup, build 2 removes the separate **Monitor Name** setting.
- **Mouse Jiggler.** The Go+ can keep the remote host awake with timed USB mouse movements, even after you close the browser. [Controls and movement patterns](#mouse-jiggler)
- **Toolbar shortcuts.** Click **Screen** to toggle view-only mode, **Volume** to mute or restore the last audible level, and **Terminal** to open a new tab. Open **Screen → Cropping** to detect borders automatically, select an area manually, or turn cropping off. Hover over a toolbar menu for its controls or use its arrow on a touchscreen.
- **Activate on Connect.** Under **Settings → Appearance**, choose whether this browser starts volume, microphone, webcam or Mouse Jiggler when it connects or reconnects. Each preference works even if its toolbar icon is hidden. The webcam must be enabled on the Go+. Volume and microphone use WebRTC, and the browser still controls capture permissions.
- **USB endpoint usage.** See how many of the seven USB IN endpoints are free and switch to one USB pointer to make room for another device. [Endpoint costs and combinations](#usb-limits)
- **Activity indicators.** Visible toolbar icons show when the microphone, webcam or jiggler is active. If a sharing icon is hidden, use **Stop sharing** in Appearance.

**Defaults**

- A stock Go+ starts with keyboard, mouse, absolute pointer and USB networking enabled. Webcam, Virtual Disk and USB audio are optional.
- Upgrades preserve your selected EDID profile, timezone, USB identity, Webcam setting and saved toolbar layout.

## Mouse Jiggler

**Start or stop.** The jiggler is off by default. Turn it on under **Settings → Device → Mouse Jiggler**. **Relative Mode** (the default) sends mouse displacements. **Absolute Mode** sends screen coordinates. To add the shortcut, enable **Mouse Jiggler** under **Settings → Appearance**. Click its icon to toggle the jiggler using the last selected mode. A green icon means it is running.

**Set the runtime.** Hover over the toolbar icon or tap its menu arrow on a touchscreen. Type a whole number of hours into **max runtime** or use the arrows. Edits save automatically. The default is **8 hours**. Set **0** for an unlimited run. The menu shows `HH:mm:ss remaining` or `∞ remaining`. Changing a running timer to **6** immediately starts a fresh six-hour countdown. Turning the jiggler off and back on also starts a fresh run. Closing the browser leaves it running. After a device restart, a timed run resumes only for the time left before its saved deadline.

**When the mouse moves.** After 30 seconds without keyboard or mouse input through the Go+, the jiggler picks a fresh delay of 0–30 seconds. Movement therefore starts after **30–60 seconds of inactivity**. New remote input interrupts a movement or cancels a pending one and starts the idle wait again. The maximum runtime continues counting down throughout.

| Desktop viewer connections | Movement at the next idle trigger |
| --- | --- |
| Any desktop viewer is connected, including a background tab | A quick nudge. Relative mode moves out and back. Absolute mode sends two nearby positions near the screen center. |
| All viewers have disconnected within the last five minutes | The same quick nudge. |
| All viewers have been disconnected for more than five minutes | A random, smooth arc or hook lasting **5–8 seconds**. Reconnecting interrupts the curve. |

Curves send pointer movement only, with no clicks or scrolling. They ease into and out of each bend and scale to the HDMI display size. In the outer quarter of either screen axis, the path steers inward on that axis. The Go+ estimates the cursor position from the absolute coordinates and relative movements it sends. Host pointer acceleration and a mouse connected directly to the host can make that estimate drift.

## USB limits

The Go+ has **7 USB IN endpoints**. **Settings → Device** shows the used and free counts above Webcam. By default, the keyboard and two pointer interfaces use **1 each**, leaving **4** for optional devices.

**Mouse → Mouse mode → Use one USB pointer endpoint** frees one more endpoint. Choose **Follow Mouse** for absolute positioning or **Exclusive Mouse** for relative movement. The jiggler uses whichever mouse endpoint is available. With the single-endpoint Android stylus or multi-touch profile, it pauses movement until a mouse endpoint is available again. Changing the active profile briefly reconnects USB for all viewers. Free an endpoint before restoring both pointers if the counter reads 7 of 7.

| Feature | IN endpoints |
| --- | ---: |
| Webcam | 2 |
| Virtual Audio (mic only or mic + speakers) | 1 |
| Virtual Network (NCM or RNDIS) | 2 per interface |
| Virtual Disk or mounted image | 1 per USB drive |
| USB serial (ACM) | 2 |

- **Webcam + Network** fills all 7 with both pointers. With one pointer, the budget also fits Virtual Audio.
- **Webcam + Audio** uses 6 with both pointers. One disk fits the software check. Using all three together still needs a host test.
- **Switching devices.** Eject and detach mounted media before changing Webcam. Turn **Settings → Device → Webcam** off to free its endpoints. Click the toolbar webcam icon again to stop sharing.

## Install build

1. Download the offline update bundle (`.tar.gz`) from the [latest Nomad release](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest). Leave it compressed.
2. Check the firmware requirement in the release notes. If your Go+ needs a firmware update, install the specified Sipeed version first. [Official firmware](https://github.com/sipeed/NanoKVM-Go/releases) · [Flashing guide](https://en.wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/system/flashing.html)
3. Open **Settings → Check for Updates**. For an unsigned release, temporarily turn off **Update signature verification**. Choose **Offline update → Select bundle → Upload and install**.
4. Once installation finishes, reboot the Go+, re-enable signature verification if needed, and confirm the installed build on the update page.

[Build from source](docs/build-packaging.md) · [License](LICENSE)
