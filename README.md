# NanoKVM-Go Nomad

NanoKVM Go+ presents your browser’s camera to the remote computer as a 720p USB webcam. Nomad supports Go+ firmware v1.0.10. [中文](README.zh.md) · [Changelog](CHANGELOG.md)

## Dashboard controls

### Screen, Preview and EDID

- **Preview.** Click the **Screen** icon to watch without sending keyboard or mouse input. A red eye marks view-only mode. Click it again to regain control. Hover over the icon or tap its menu arrow to open display settings.
- **Screen → Cropping** can remove black borders automatically. For a fixed crop, select an area manually and reuse the saved region.
- **Screen → EDID** sets the monitor name, resolution and refresh rate the remote computer sees. For example, the 1080p60 preset identifies as a Dell U2414H. Each of the five built-in profiles supplies its own monitor name. For a custom name, upload and select an EDID `.bin` containing it. You can also switch profiles under **Settings → Screen → EDID**.

### Mouse Jiggler

Click **Mouse Jiggler** to start or stop it. If the shortcut is hidden, enable **Show icon** under **Settings → Appearance**. Choose relative or absolute movement under **Settings → Device → Mouse Jiggler**.

The toolbar menu shows the time remaining and lets you set a limit in whole hours. The default is **8 hours**. Set **0** for unlimited. Changing the limit during a run restarts the countdown. Closing the dashboard leaves it running, and rebooting the Go+ preserves the original deadline.

| Desktop viewers | What the pointer does while the jiggler is on |
| --- | --- |
| At least one connected | After 30–60 seconds without KVM input, the pointer makes a short nudge. New input restarts the idle wait. |
| Last viewer disconnected | Short nudges continue for the first five minutes. After that, idle movements follow smooth curves lasting 5–8 seconds. Reconnecting stops a curve in progress and restores short nudges. |

### Webcam, toolbar and USB

- **Webcam.** Open the dashboard over HTTPS and enable **Settings → Device → Webcam**. Click **Webcam** to start or stop sharing. Its menu lets you choose the camera and check upload stats. The browser adjusts bitrate and frame rate together as the connection changes. [Webcam setup and upload stats](docs/remote-webcam.md)
- **Toolbar shortcuts.** Click **Volume** to mute or restore the previous level, **Microphone** to toggle sharing, or **Terminal** to open a new tab. After you close Settings with the mouse, the bar hides again after five seconds with the pointer outside it. Keyboard focus keeps the bar visible.
- **Activate on Connect.** Under **Settings → Appearance**, each browser can start audio playback, microphone sharing, webcam sharing and Mouse Jiggler independently. **Show icon** only controls visibility. Hidden shortcuts can still activate on connect, and toggling one during a session leaves its startup preference intact.
- **USB devices.** **Settings → Device** shows used and free USB IN endpoints out of seven. Webcam needs two. If you are short by one, **Mouse → Mouse mode → Use one USB pointer endpoint** frees one by exposing only the selected pointer. Switching layouts briefly reconnects USB. The jiggler needs a mouse endpoint, so choose a mouse mode to use it in this layout.
- **USB identity.** **Settings → Device → Advanced → USB Device Identity** lets you edit the configuration name, device IDs and device strings. The webcam has its own **USB Webcam Name** setting.

## Install

1. Download the matching offline update and leave the `.tar.gz` compressed. [Published releases](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. Upload it under **Settings → Check for Updates → Offline update**. For unsigned bundles, temporarily disable **Update signature verification**.
3. Reboot the Go+ and restore signature verification if disabled.

[Build the Nomad bundle](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed user guide](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html) · [License](LICENSE)
