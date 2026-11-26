# NanoKVM-Go Nomad

通过 NanoKVM Go+，远程电脑可将浏览器端的摄像头用作 720p USB 摄像头。Build #2 适用于 Go+ 固件 v1.0.10。[English](README.md) · [Build #2 更新记录](CHANGELOG.md)

## 控制台

- **摄像头。** 通过 HTTPS 打开控制台，在 **Settings → Device → Webcam** 中启用摄像头。点击工具栏的 **Webcam** 开始或停止共享，悬停或轻点箭头可选择摄像头。[设置与上传数据](docs/remote-webcam.md)
- **画面。** 点击 **Screen** 切换只读模式。**Screen → Cropping** 可自动去除黑边，也可手动选择区域。在 **Settings → Screen → EDID** 中选择配置，可同时设置显示器名称和支持的显示模式。
- **工具栏。** 点击 **Volume** 静音或恢复原音量，点击 **Microphone** 开始或停止共享麦克风，点击 **Terminal** 在新标签页打开终端。用鼠标关闭设置后，只要指针在工具栏外，自动隐藏的五秒倒计时就会恢复。
- **连接时启用。** 在 **Settings → Appearance** 中，可为音量、麦克风、摄像头和鼠标抖动器分别设置 **Show icon** 和 **Activate on Connect**。
- **鼠标抖动器。** 在 **Settings → Device** 中启用 **Mouse Jiggler**。工具栏菜单显示剩余时间，默认运行 **8 小时**，设为 **0** 表示不限时。断开连接后仍会运行。
- **USB。** **Settings → Device** 显示可用的 USB IN 端点数。摄像头需要两个。启用 **Mouse → Mouse mode → Use one USB pointer endpoint** 可释放一个。在 **Device → Advanced → USB Device Identity** 中可修改 USB ID、设备标识字符串和配置名称。

## 安装

1. 下载与固件匹配的离线更新包，保留 `.tar.gz` 压缩格式。[已发布的版本](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. 通过 **Settings → Check for Updates → Offline update** 上传更新包。若更新包未签名，可暂时关闭 **Update signature verification**。
3. 重启 Go+。如果关闭了签名验证，请重新启用。

[构建 Nomad 更新包](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed 使用指南](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/user_guide.html) · [许可证](LICENSE)
