# Share a browser webcam with the Go+

Open the Go+ dashboard over HTTPS. Under **Settings → Device**, enable **Webcam** to present a USB camera to the attached computer. Changing this setting briefly reconnects the USB devices. Eject and detach virtual media first. The webcam requires two free USB IN endpoints, USB display mode turned off, and the Nomad UVC-enabled kernel. **Settings → Device** shows the endpoint count. If you are short by one, **Mouse → Mouse mode → Use one USB pointer endpoint** frees one. The Nomad Normal update includes the kernel, feeder and scripts. Reboot after replacing the kernel. [Kernel support](../support/patches/maix_ax620e_sdk_kernel/README.md#uvc-webcam-gadget-on-the-go)

Click the toolbar webcam icon to start sharing the default camera. To choose a camera before sharing, hover over the icon or tap its menu arrow. To share the default camera automatically when this browser connects, check **Activate on Connect** in the **Webcam** row under **Settings → Appearance**. Grant camera access in the browser. One dashboard can publish at a time. The remote computer can open the USB webcam before or after calibration. Microphone sharing has a separate control. If startup reports that no frames reached the Go+, check the selected camera and its browser permission, then start sharing again.

| Menu state | What the numbers mean |
| --- | --- |
| Calibrating | Received JPEG Mbps, acknowledged frames per second and average time from upload to the next grant, in milliseconds. |
| Ready | The calibrated bitrate and frame-rate target. No application on the remote computer is requesting frames. |
| Streaming | Live received Mbps, acknowledged frames per second and average upload-to-grant delay. |

## How calibration works

The browser starts at **8 Mbps and 25 fps**. It skips the first second and then measures **five two-second windows**, for 11 seconds total. On a healthy, sufficiently loaded link it tests the 10 Mbps ceiling. When a target is congested it bisects the remaining range between the working and failing rates. Lower targets also ease the frame cadence to preserve JPEG detail. The bounds are **2–10 Mbps** and **15–30 fps**. A quiet scene can use less than the target and provide too little evidence to raise it. If the remote computer opens the webcam early, the stream takes over immediately.

During streaming, five seconds with less than 3% missed upload opportunities, a stable grant delay, no queue growth, and delivery of at least 90% of the target fps allow a joint probe. The controller raises bitrate by 15% and frame rate by 10%, rounded to the nearest whole fps and capped at the host limit. From 8 Mbps / 25 fps, two healthy windows reach 9.2 Mbps / 28 fps and then 10 Mbps / 30 fps.

More than 10% missed over five seconds triggers a 10% reduction. More than 25% over three seconds triggers a 20% reduction. Both cuts reduce frame rate and use the smaller of the measured JPEG budget at the requested cadence and the target bitrate as the bitrate baseline. A failed upward probe restores the previous bitrate and frame rate. Upward probes pause for 20 seconds after a rollback and at least 10 seconds after a regular cut. A miss measures a frame the sender could have uploaded at its paced cadence. The displayed delay includes grant pacing and is different from a network ping.

The browser sends JPEGs up to 128 KiB each. The Go+ exposes a 1280 × 720 MJPEG USB profile at 30 fps. Actual fresh frames depend on the camera, encoder and link. Supported browsers can capture and encode in a worker. The video-element path supports WebKit timing. Clicking the webcam icon again releases the local camera and restores black USB output. Closing the dashboard also ends sharing. The device setting survives a normal reboot. Webcam starts off on stock systems. Upgrades retain the selected device setting.

The host-visible webcam name defaults to **Anker Powerconf C200**. Change it under **Settings → Device → Advanced → USB Webcam Name**. The new name applies the next time the USB function is created.
