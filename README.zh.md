# NanoKVM-Go Nomad

使用 Nomad，可在连接到 NanoKVM Go+ 的远程电脑上调用笔记本摄像头参加视频会议。远程电脑会将它识别为标准的 720p USB 摄像头。[English](README.md)

## 控制台

- **菜单栏。** 点击 **Screen** 切换只读模式，点击 **Volume** 静音或恢复音量，点击 **Terminal** 在新标签页打开终端。将指针悬停在图标上或轻点旁边的箭头，可查看更多选项。
- **摄像头。** 通过 HTTPS 打开控制台，在 **Settings → Device → Webcam** 中启用摄像头，再点击工具栏的 **Webcam** 开始共享。视频流会根据上行带宽自动调整。[摄像头指南](docs/remote-webcam.md)
- **显示器。** 在 **Settings → Screen → EDID** 中选择配置，即可同时设置显示器名称和支持的显示模式，无需单独设置 **Monitor Name**。
- **鼠标抖动器。** 在 **Settings → Device** 中启用 **Mouse Jiggler**，关闭浏览器后仍可保持远程电脑唤醒。默认运行 8 小时。
- **USB 设备。** 在 **Settings → Device** 中查看可用 USB 端点。启用 **Mouse → Mouse mode → Use one USB pointer endpoint** 可释放一个端点。
- **连接时启用。** 在 **Settings → Appearance** 中选择浏览器连接时是否自动启用音频、摄像头共享或鼠标抖动器。

## 安装

1. 下载离线更新包（`.tar.gz`），并查看发布说明中要求的 Sipeed 固件版本。无需解压。[最新 Nomad 版本](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. 打开 **Settings → Check for Updates → Offline update** 并上传更新包。若更新包未签名，可暂时关闭 **Update signature verification**。
3. 安装完成后重启 Go+。如果关闭了签名验证，请重新启用。

[从源码构建](docs/build-packaging.md) · [Sipeed 使用指南](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/user_guide.html) · [许可证](LICENSE)
