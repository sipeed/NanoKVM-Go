# Share a browser webcam with the Go+

Open the Go+ dashboard over HTTPS. Under **Settings → Device**, enable **Webcam** to present a USB camera to the attached computer. Changing this setting briefly reconnects the USB devices. Eject and detach virtual media first. The feature needs free USB endpoints and the Nomad UVC-enabled kernel. The Nomad Normal update includes the kernel, feeder and scripts. Reboot after replacing the kernel. [Kernel support](../support/patches/maix_ax620e_sdk_kernel/README.md)

Open the toolbar webcam menu, choose a local camera and click **Start**. Grant camera access in the browser. One dashboard can publish at a time. The remote computer can open the USB webcam before or after calibration. Microphone sharing has a separate control.

| Menu state | What the numbers mean |
| --- | --- |
| Calibrating | Received JPEG Mbps, acknowledged frames per second and average time from upload to the next grant, in milliseconds. |
| Ready | The calibrated bitrate and frame-rate target. The remote computer is idle. |
| Streaming | Live received Mbps, acknowledged frames per second and average upload-to-grant delay. |

## How calibration works

The browser starts at **5 Mbps and 30 fps**. It skips the first second and then measures **five two-second windows**, for 11 seconds total. On a healthy, sufficiently loaded link it tests the 10 Mbps ceiling. When a target is congested it bisects the remaining range between the working and failing rates. Lower targets also ease the frame cadence to preserve JPEG detail. The bounds are **2–10 Mbps** and **15–30 fps**. A quiet scene can use less than the target and provide too little evidence to raise it. If the remote computer opens the webcam early, the stream takes over immediately.

During streaming, the controller ordinarily assesses ten seconds of delivery. More than 10% missed upload opportunities triggers a rate cut. More than 25% over three seconds triggers a larger cut. It can also lower the capture cadence if smaller frames fail to improve delivery. Clean delivery permits gradual upward probes. A miss measures a frame the sender could have uploaded at its paced cadence. The displayed delay includes grant pacing and is different from a network ping.

The browser sends JPEGs up to 128 KiB each. The Go+ exposes a 1280 × 720 MJPEG USB profile at 30 fps. Actual fresh frames depend on the camera, encoder and link. Supported browsers can capture and encode in a worker. The video-element path supports WebKit timing. **Stop** releases the local camera and restores black USB output. Closing the dashboard also ends sharing. The device setting survives a normal reboot. Webcam starts off on stock systems. Upgrades retain the selected device setting. Each new browser session requires **Start**.

The host-visible webcam name defaults to the kernel’s generic USB camera name. Change it under **Settings → Device → Advanced → USB Webcam Name**. The new name applies the next time the USB function is created.
