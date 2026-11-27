# NanoKVM-Go Nomad

[English](README.md) · [下载构建版本](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest) · [安装方法](#安装-nomad) · [摄像头设置与校准](docs/remote-webcam.md) · [更新记录](CHANGELOG.md)

## 关于 Nomad

Nomad 是 Sipeed NanoKVM-Go 的一个分支，面向外出时仍需使用固定工作站的数字游民。把 **NanoKVM Go+** 留在工作站旁，即可在旅途中用浏览器查看并操作工作站。视频也可以反向传输：在工作站上参加会议时，可以使用随身笔记本或手机上的摄像头。

本仓库提供适用于 **Go+（固件 v1.0.10）** 的源代码和离线更新包。以下列出相对于 Sipeed 0.2.7 的变化。[Sipeed 官方项目](https://github.com/sipeed/NanoKVM-Go)

## 新功能

- **USB 摄像头共享。** 工作站可以把浏览器端的摄像头当作 720p USB 摄像头使用。传输会适应上行带宽，菜单中可查看实时数据。
- **目标键盘布局。** 控制台的粘贴功能与 MCP 文本输入支持 11 种布局，包括加拿大法语、加拿大多语言标准和微软拼音。
- **USB 端点管理。** 可以查看 Go+ 的七个 USB IN 端点还有几个可用。鼠标默认使用一个端点。

## 改进

- **菜单栏。** 单击图标执行常用操作，悬停可打开更多选项。图标颜色反映运行状态。只读预览、裁剪和 EDID 选择集中在 Screen 菜单。
- **连接时自动启用。** 每个浏览器可以分别设置连接后是否启用声音播放、麦克风共享、摄像头共享和鼠标抖动器。
- **鼠标抖动器。** 在 Sipeed 原有功能上增加了工具栏开关、倒计时，以及根据桌面是否有人连接而改变的移动方式。
- **显示配置。** 原有的五种显示模式现在各自对应一个显示器型号。所选 EDID 同时提供型号和时序。
- **USB 设备名称。** 高级设置可单独指定摄像头和 USB 配置名称。复合设备、音频、存储和摄像头均有更新后的默认名称。
- **离线更新。** 一个 Nomad 更新包即可安装应用程序和配套的摄像头内核。控制台和更新页均显示 Nomad 构建版本。

## 错误修复

- **Windows 设备识别。** 原有五个 EDID 使用相同的厂商和产品编号。Nomad 为各配置提供独立的显示器标识和型号，供 Windows 读取。摄像头也会报告自己的 USB 接口名称。Windows 也可能显示为该设备单独保存的名称。[Windows 如何选择显示名称](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/devpkey-name--device-instance-)
- **Chrome 窗口缩放。** 打开、关闭 Chrome 侧边栏或拖动其边界时，远程画面会随可用宽度调整。此前查看器会把宽度变化误判为网页缩放。
- **多用户输入。** 切换 USB 指针布局时，USB 会短暂重连。其他已打开的控制台会同步新布局并恢复输入。
- **摄像头启动与码率恢复。** 启动时若 Go+ 未收到任何帧，控制台会报错。带宽恢复后，即使摄像头已达到帧率上限，码率仍可回升。
- **USB 配置恢复。** 切换摄像头后若主机无法完成 USB 配置，Go+ 会恢复基本 USB 配置。修改 USB 标识失败时会恢复已保存的设置。
- **菜单栏自动隐藏。** 用鼠标关闭设置且指针位于菜单栏外时，会恢复五秒自动隐藏倒计时。键盘焦点位于菜单栏内时，菜单栏保持可见。

## 详细说明

### 摄像头与麦克风

1. 通过 HTTPS 打开控制台。在 **Settings → Device** 中关闭 USB 显示模式，确认有两个空闲的 USB IN 端点，再启用 **Webcam**。先弹出并卸载已挂载的虚拟介质。更改此设备设置会使 USB 短暂重连。
2. 单击 **Webcam** 共享默认摄像头，或打开菜单选择其他摄像头。在浏览器中授予摄像头权限。同一时间只能由一个浏览器发送画面。需要传送语音时，单独开启 **Microphone**。
3. 在工作站的会议应用中选择 USB 摄像头。浏览器通过加密的 WebSocket 发送 JPEG 帧，Go+ 将其作为标准的 **1280 × 720 MJPEG USB 视频类（UVC）摄像头**输出，最高 30 fps。

校准从 **8 Mbps、25 fps** 开始，先预热一秒，再采集五个时长两秒的样本。传输期间，浏览器根据错过的上传时机、队列积压和上传延迟，在 **2–10 Mbps** 和 **15–30 fps** 范围内调整 JPEG 码率与帧率。校准和传输时，菜单显示实际接收码率、每秒确认接收的帧数和上传延迟。工作站尚未请求画面时，**Ready** 显示校准后的目标值。[摄像头传输控制与统计](docs/remote-webcam.md)

支持后台工作线程的浏览器可在线程中采集和编码视频帧。Safari 等 WebKit 浏览器也可以使用视频元素采集画面。停止共享或关闭控制台会释放本地摄像头，USB 视频输出恢复黑屏。Go+ 重启后保留 Webcam 设备设置。关闭该设置才会释放两个端点。

### 菜单栏与连接偏好

| 控件 | 单击图标 | 打开菜单 |
| --- | --- | --- |
| Screen | 切换只读预览。红色眼睛表示已暂停远程输入。 | 选择 EDID 或调整裁剪。 |
| Volume | 静音或恢复上次的非零音量。 | 调整播放音量。 |
| Webcam | 开始或停止摄像头共享。 | 选择摄像头或查看传输数据。 |
| Microphone | 开始或停止麦克风共享。 | — |
| Mouse Jiggler | 按上次使用的模式启动或停止。 | 设置运行时限或查看倒计时。 |
| Terminal | 在新标签页打开 Go+ 终端。 | — |

有菜单的控件支持鼠标悬停打开，触摸屏则可轻点箭头。**Settings → Appearance** 中的 **Show icon** 控制 Go+ 上的快捷图标是否可见。**Activate on Connect** 则由各浏览器单独保存，可为 Volume、Microphone、Webcam 和 Mouse Jiggler 分别设置。隐藏图标仍可自动启用功能。摄像头或麦克风正在共享而图标被隐藏时，可在对应设置行点击 **Stop sharing**。会话中临时切换开关不影响下次连接的偏好。

### 鼠标抖动器

新菜单栏默认隐藏此图标，可在 **Settings → Appearance** 中为 Mouse Jiggler 打开 **Show icon**。在 **Settings → Device → Mouse Jiggler** 中选择相对或绝对移动。工具栏菜单按整小时设置运行时限，默认 **8 小时**，**0** 表示不限时。运行中修改时限会重新开始倒计时。计时由 Go+ 负责，因此关闭浏览器后仍会运行，重启设备也会保留原截止时间。

| 桌面连接状态 | 空闲时的移动方式 |
| --- | --- |
| 仍有桌面客户端连接，或最后一个客户端断开未满五分钟 | 连续 30–60 秒无 KVM 输入后轻微移动。新输入会重新开始空闲计时。 |
| 所有桌面客户端均已断开至少五分钟 | 指针沿平滑曲线移动，每次 5–8 秒，接近估算的屏幕边缘时向内转。重新连接会中断当前曲线。 |

### 画面裁剪与 EDID

**Screen → Cropping** 控制浏览器显示输入画面的哪个部分。**Auto** 自动检测黑边，**Manual** 可框选并保存区域或选择分辨率，**Off** 恢复完整画面。绝对指针输入会对齐可见内容。

**Screen → EDID** 和 **Settings → Screen → EDID** 控制工作站识别到的显示器。任选一个入口选择分辨率和刷新率，对应的 EDID 会提供厂商、产品编号和型号。

| 显示模式 | Go+ 报告的显示器型号 |
| --- | --- |
| 1920 × 1080，60 Hz | Dell U2414H |
| 2560 × 1440，60 Hz | Dell S2716DG |
| 3440 × 1440，60 Hz | Dell U3415W |
| 3840 × 2160，30 Hz | Dell P2415Q |
| 3840 × 2160，50 Hz | Acer XV273K |

要使用其他型号，请在 **Screen → EDID** 中上传相应的 EDID `.bin` 文件并选中。升级时会保留所选配置，原先单独设置的 **Monitor Name** 将由配置内的名称取代。

### 键盘与 MCP

在 **Keyboard → Keyboards** 中选择与工作站当前输入布局一致的选项，并关闭大写锁定。子菜单会标记当前布局，也可以从这里打开 **Show virtual keyboard**。Go+ 保存的目标布局同时用于 **Keyboard → Paste** 和 MCP 的 `type_text`。例如，选择 **French → Canadian Multilingual Standard** 后，两种输入方式都会按该布局发送重音字母所需的按键。

| 语言 | 目标布局 |
| --- | --- |
| 英语 | 美国（默认）、英国 |
| 法语 | 加拿大、加拿大多语言标准、法国 |
| 德语和西班牙语 | 德国、西班牙 |
| 中文 | 简体中文 · 微软拼音 |
| 俄语 | Windows 标准 ЙЦУКЕН |
| 波斯语 | 波斯语、波斯语标准版（ISIRI 9147） |

两种输入方式每次最多接受 1,024 个字符，并会先检查整段文字。遇到不支持的字符时，会在发送第一个按键前报错。换行会发送 Enter，制表符会发送 Tab。使用微软拼音布局时，工作站须切换到中文输入模式并启用全拼及 V 模式，可输入由 Unicode 基本多文种平面内可打印字符组成的中英文混合文本。

Sipeed 提供了可选的 MCP 服务，用于截屏和发送输入。在 **Settings → AI → MCP Service** 启用后，使用生成的 API 密钥将客户端连接到 `https://<device>/api/mcp`。Nomad 的 `type_text` 使用已保存的布局。发送物理组合键可用 `press_keys`。记忆与 OCR 工具还需单独启用 **Memory Fabric**。[MCP 配置、工具与示例](docs/mcp-usage.md)

### USB 设备与指针模式

**Settings → Device** 显示已用和空闲的 USB IN 端点数。默认情况下，键盘和鼠标各占用一个端点，剩余五个。摄像头需要两个，虚拟音频需要一个，虚拟网络接口需要两个，因此三者可以同时使用。关闭 **Mouse → Mouse mode → Use one USB pointer endpoint** 会增加一个指针端点。

单端点模式只向工作站提供当前所选的指针设备。更改布局会影响所有查看者。鼠标抖动器会使用可用的鼠标；仅启用触控笔或触摸屏时，抖动器不可用。[USB 指针模式](docs/kvmgo-device-usb-hid.md)

在 **Settings → Device → Advanced** 中，复合 USB 设备和音频功能默认使用 **Logitech P710e**，存储设备使用 Logitech 驱动器名称，**USB Webcam Name** 默认为 **Anker Powerconf C200**。已保存的自定义值优先。下次启用 Webcam 或重启 Go+ 时，新的摄像头名称生效。**USB Configuration Name** 为可选项。

### 安装 Nomad

1. 查看发布说明中的 Go+ 固件要求，下载匹配的离线更新包。保留压缩的 `.tar.gz` 文件。[最新 Nomad 版本](https://github.com/maxhabra/NanoKVM-go-Nomad/releases/latest)
2. 在 **Settings → Check for Updates → Offline update** 中上传更新包。若使用未签名的包，请暂时关闭 **Update signature verification**。
3. 重启 Go+，重新开启此前关闭的签名验证，并在更新页面确认 Nomad 构建版本。

安装程序会检查设备型号、固件与启动分区布局。需要更换内核时，会先备份原镜像。[构建与打包指南](docs/build-packaging.md#nomad-go-offline-bundle) · [Sipeed 用户指南](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/user_guide.html) · [许可证](LICENSE)
