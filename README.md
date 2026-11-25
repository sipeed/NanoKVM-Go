# NanoKVM-Go Nomad

Nomad lets you use your laptop’s webcam in calls on a remote computer connected to a NanoKVM Go+. The remote computer sees a standard 720p USB webcam. [中文](README.zh.md)

## Dashboard controls

- **Menu bar.** Click **Screen** to toggle view-only mode, **Volume** to mute or unmute, or **Terminal** to open a new tab. Hover over an icon or tap its arrow for more options.
- **Webcam.** Open the dashboard over HTTPS, enable **Settings → Device → Webcam**, then click **Webcam** on the toolbar to share. The stream adapts to your upload speed. [Webcam guide](docs/remote-webcam.md)
- **Display.** Choose a profile under **Settings → Screen → EDID**. It sets the monitor name and supported display modes together, so there’s no separate **Monitor Name** setting.
- **Mouse Jiggler.** Enable it under **Settings → Device** to keep the remote computer awake after you close the browser. The default runtime is 8 hours.
- **USB devices.** Check **Settings → Device** for available USB endpoints. If you need one more, enable **Mouse → Mouse mode → Use one USB pointer endpoint**.
- **On connect.** Under **Settings → Appearance**, choose whether this browser starts audio, webcam sharing, or Mouse Jiggler when it connects.

## Install

1. Download the offline update bundle (`.tar.gz`) and check the release notes for the required Sipeed firmware. Leave the bundle compressed. [Latest Nomad release](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. Open **Settings → Check for Updates → Offline update** and upload the bundle. For an unsigned release, temporarily disable **Update signature verification**.
3. Reboot the Go+ after installation and re-enable signature verification if you disabled it.

[Build from source](docs/build-packaging.md) · [Sipeed user guide](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html) · [License](LICENSE)
