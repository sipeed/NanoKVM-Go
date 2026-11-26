# NanoKVM-Go Nomad

NanoKVM Go+ presents your browser’s camera to the remote computer as a 720p USB webcam. Build #2 targets firmware v1.0.10. [中文](README.zh.md) · [Build #2 changelog](CHANGELOG.md)

## Dashboard controls

- **Webcam.** Open the dashboard over HTTPS. Enable **Settings → Device → Webcam**, then click **Webcam** to share or stop. Hover or tap its arrow to choose a camera. [Setup and upload stats](docs/remote-webcam.md)
- **Screen.** Click to toggle view-only mode. **Screen → Cropping** removes black borders automatically or lets you select an area. **Settings → Screen → EDID** sets the monitor name and display modes from one profile.
- **Toolbar.** **Volume** mutes or restores the previous level. **Microphone** toggles sharing. **Terminal** opens a tab. Closing Settings with the mouse resumes the five-second auto-hide timer once the pointer leaves the bar.
- **On connect.** Under **Settings → Appearance**, set **Show icon** and **Activate on Connect** separately for Volume, Microphone, Webcam and Mouse Jiggler.
- **Mouse Jiggler.** Enable it under **Settings → Device**. Set hours in its toolbar menu (**8** by default, **0** for unlimited). The timer continues after disconnect.
- **USB.** **Settings → Device** shows free IN endpoints. Webcam needs two. **Mouse → Mouse mode → Use one USB pointer endpoint** frees one. Under **Device → Advanced → USB Device Identity**, edit the IDs, device strings and configuration name.

## Install

1. Download the matching offline update and leave the `.tar.gz` compressed. [Published releases](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. Upload it under **Settings → Check for Updates → Offline update**. For unsigned bundles, temporarily disable **Update signature verification**.
3. Reboot the Go+ and restore signature verification if disabled.

[Build the Nomad bundle](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed user guide](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html) · [License](LICENSE)
