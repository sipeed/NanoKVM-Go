# NanoKVM-Go Nomad

Nomad is a fork of Sipeed’s NanoKVM-Go for the **Go+ running firmware v1.0.10**. [Original project](https://github.com/sipeed/NanoKVM-Go) · [中文](README.zh.md) · [Changelog](CHANGELOG.md)

## What Nomad adds

- **USB webcam sharing**
  - Use the camera on the computer or phone running your browser as a **720p USB webcam at up to 30 fps** on the remote computer.
  - Automatic calibration adjusts bitrate and frame rate as the connection changes. The webcam menu shows throughput, delivered frame rate and upload delay.
- **Menu bar improvements**
  - Hover over a shortcut to open its menu, or tap its arrow on a touchscreen. Click the icon itself to mute or restore volume, toggle the webcam or microphone, or open a terminal tab.
  - **Screen** combines Preview and cropping. Click the icon for view-only mode (a red eye), or open the menu to remove black borders automatically or select a saved crop.
  - Closing Settings with the mouse resumes the five-second auto-hide timer when the pointer is outside the bar. Keyboard focus keeps the bar visible.
  - **Settings → Appearance** lets each browser choose which icons to show and whether audio playback, microphone, webcam or Mouse Jiggler starts on connect. Hidden icons can still activate on connect.
- **Mouse Jiggler improvements**
  - Start or stop the built-in jiggler from the toolbar, set a runtime limit and see the time remaining. Movement changes after the last desktop viewer disconnects.
- **Monitor EDIDs**
  - Five revised display profiles give the remote computer a monitor model, resolution and refresh rate together. The selected profile supplies the monitor name.
- **USB configuration and offline updates**
  - Device settings show used and free USB IN endpoints out of seven. **Mouse → Mouse mode → Use one USB pointer endpoint** frees one by exposing only the selected pointer. Switching layouts briefly reconnects USB.
  - Advanced device settings let you edit the USB configuration name and webcam name. The composite USB device defaults to Logitech P710e and the webcam to Anker Powerconf C200.
  - The offline update bundles the application and webcam-enabled kernel. The dashboard and update screen identify the installed Nomad build.

## How the features work

### Webcam

Open the dashboard over HTTPS, enable **Settings → Device → Webcam**, then click **Webcam** to share your camera. The browser sends JPEG frames over a secure WebSocket. The Go+ passes them to its USB Video Class (UVC) device, so an app on the remote computer can select it like a locally plugged-in camera. It uses two free USB IN endpoints and requires USB display mode to be off.

Calibration starts at **8 Mbps and 25 fps**. During streaming, the browser tracks missed frames, upload delay and queue growth. It lowers its targets when frames miss their upload window. When the link recovers, it can raise bitrate again even if the camera has reached its frame-rate limit. The targets range from **2–10 Mbps** and **15–30 fps**. Stopping sharing releases the browser camera and returns the USB video to black. [Webcam setup, calibration and live statistics](docs/remote-webcam.md)

### Mouse Jiggler

The Go+ sends USB mouse movements in relative or absolute mode. If the toolbar shortcut is hidden, enable **Show icon** under **Settings → Appearance**. Select the mode under **Settings → Device → Mouse Jiggler**. Its toolbar menu sets the maximum runtime in whole hours (**8** by default, **0** for unlimited). Changing the limit restarts the countdown. The timer runs on the Go+, so closing the browser leaves it running and a reboot preserves its deadline.

| Desktop viewers | Pointer behavior |
| --- | --- |
| At least one connected | A short nudge after 30–60 seconds without KVM input. New input restarts the idle wait. |
| All disconnected | Short nudges continue for five minutes. Subsequent movements trace smooth 5–8 second curves, turning inward near screen edges. Reconnecting stops a curve in progress. |

### Monitor EDIDs

The remote computer reads the selected EDID to identify the attached display and its supported modes. Choose a preset under **Screen → EDID** or **Settings → Screen → EDID**. Each preset includes the monitor name, so there is no separate **Monitor Name** field. For a different identity, upload an EDID `.bin` with the desired name through **Screen → EDID**, then select it.

| Display mode | Monitor reported to the remote computer |
| --- | --- |
| 1920 × 1080 at 60 Hz | Dell U2414H |
| 2560 × 1440 at 60 Hz | Dell S2716DG |
| 3440 × 1440 at 60 Hz | Dell U3415W |
| 3840 × 2160 at 30 Hz | Dell P2415Q |
| 3840 × 2160 at 50 Hz | Acer XV273K |

## Install

1. Download the matching offline update and leave the `.tar.gz` compressed. [Published releases](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. Upload it under **Settings → Check for Updates → Offline update**. For unsigned bundles, temporarily disable **Update signature verification**.
3. Reboot the Go+ and restore signature verification if disabled.

[Build the Nomad bundle](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed user guide](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html) · [License](LICENSE)
