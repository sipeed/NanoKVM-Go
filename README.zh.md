# NanoKVM-Go Nomad

通过 NanoKVM Go+，远程电脑可将浏览器端的摄像头用作 720p USB 摄像头。Build #2 适用于 Go+ 固件 v1.0.10。[English](README.md) · [Build #2 更新记录](CHANGELOG.md)

## Build #2 控制台变化

### 画面、预览与 EDID

- **预览已并入 Screen。** 点击 **Screen** 图标进入只读模式，此时仍可观看画面，键盘和鼠标操作不会发送到远程电脑。红色眼睛图标表示正在预览，再点一次即可恢复控制。将鼠标悬停在图标上或轻点菜单箭头，可打开显示设置。
- **Screen → Cropping** 可自动去除黑边。需要固定裁剪范围时，可手动框选区域，之后可直接选用已保存的区域。
- **Screen → EDID** 决定远程电脑识别到的显示器名称、分辨率和刷新率。例如，1080p60 预设对应 Dell U2414H。五个内置配置都自带名称，原来的 **Monitor Name** 字段已移除，升级后旧字段中保存的名称不再生效。如需自定义名称，请上传并选用包含该名称的 EDID `.bin` 文件。也可在 **Settings → Screen → EDID** 中切换配置。

### 鼠标抖动器

点击 **Mouse Jiggler** 可启动或停止抖动器。如果工具栏未显示该图标，可在 **Settings → Appearance** 中勾选 **Show icon**。在 **Settings → Device → Mouse Jiggler** 中可选择相对或绝对移动模式。

图标菜单会显示剩余时间，也可按整小时设置运行上限。默认 **8 小时**，设为 **0** 表示不限时。运行期间修改上限会重新开始计时。关闭控制台后抖动器仍会运行，重启 Go+ 也会保留原定的结束时间。

| 桌面连接状态 | 抖动器开启时的指针行为 |
| --- | --- |
| 至少一个客户端已连接 | 连续 30–60 秒没有 KVM 输入后，指针会轻微移动。新的输入会重新开始空闲计时。 |
| 最后一个客户端已断开 | 前五分钟仍是短距离轻移，之后每次空闲移动会沿平滑曲线持续 5–8 秒。重新连接时会立即中止正在进行的曲线移动，恢复短距离轻移。 |

### 摄像头、工具栏与 USB

- **摄像头。** 通过 HTTPS 打开控制台，在 **Settings → Device → Webcam** 中启用摄像头。点击 **Webcam** 开始或停止共享，在图标菜单中可选择摄像头并查看上传数据。远程电脑现在接收 720p 视频，浏览器会根据连接状况同时调整码率和帧率。[摄像头设置与上传数据](docs/remote-webcam.md)
- **工具栏快捷操作。** 点击 **Volume** 静音或恢复原音量，点击 **Microphone** 开始或停止共享麦克风，点击 **Terminal** 在新标签页打开终端。用鼠标关闭设置后，如果指针在工具栏外停留五秒，工具栏就会自动隐藏。键盘焦点在工具栏上时会保持显示。
- **连接时启用。** 在 **Settings → Appearance** 中，每个浏览器可单独设置连接时是否开启音频播放、麦克风共享、摄像头共享和鼠标抖动器。**Show icon** 仅控制图标是否显示。隐藏图标仍可在连接时自动启用对应功能，会话中手动切换也不会更改启动偏好。
- **USB 设备。** **Settings → Device** 显示七个 USB IN 端点中已用和可用的数量。摄像头需要两个端点。如果只差一个，启用 **Mouse → Mouse mode → Use one USB pointer endpoint** 可仅向主机提供所选的指针设备，从而释放一个端点。切换时 USB 会短暂重连。抖动器需要鼠标端点，在此布局下使用时请选择鼠标模式。
- **USB 标识。** **Settings → Device → Advanced → USB Device Identity** 除了设备 ID 和标识字符串，现在也可修改配置名称。摄像头名称在独立的 **USB Webcam Name** 中设置。

## 安装

1. 下载与固件匹配的离线更新包，保留 `.tar.gz` 压缩格式。[已发布的版本](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. 通过 **Settings → Check for Updates → Offline update** 上传更新包。若更新包未签名，可暂时关闭 **Update signature verification**。
3. 重启 Go+。如果关闭了签名验证，请重新启用。

[构建 Nomad 更新包](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed 使用指南](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/user_guide.html) · [许可证](LICENSE)
