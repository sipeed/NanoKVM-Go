# NanoKVM-Go

[English](README.md) | 中文

NanoKVM-Go 是 **NanoKVM Go 系列**的开源软件仓库。该系列是
[Sipeed](https://sipeed.com) 推出的掌上 USB-C KVM，针对手机、平板和便携电脑优化。

完成初始网络配置后，将它连接到支持 USB-C 视频输出的设备，即可从另一台手机或电脑的
浏览器中查看画面并进行点击、滑动和输入。由于画面从硬件层采集、输入以硬件方式模拟，
被控端无需安装专用驱动或远程控制客户端；即使操作系统卡死、无法联网或尚未启动，也能用于
UEFI/BIOS 设置、系统安装和故障排查。

> 产品介绍、快速上手与完整兼容列表见
> [Sipeed Wiki](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/introduction.html)。

## 型号

系列包含两个型号，二者拥有相同的 KVM、MCP、无线网络和扩展能力；Go+ 额外提供端侧 NPU
与记忆织网。

| | NanoKVM Go | NanoKVM Go+ |
| --- | --- | --- |
| 处理器 | 双核 Cortex-A53 | 双核 Cortex-A53 + 3.2 TOPS NPU |
| 内存 | 256 MB LPDDR4x | 512 MB LPDDR4 |
| 存储 | 16 GB | 64 GB |
| 显示屏 | 1.83 英寸彩色触摸屏 | 1.83 英寸彩色触摸屏 |
| 最大视频采集 | 4K @ 50 Hz；2K @ 90 Hz | 4K @ 50 Hz；2K @ 90 Hz |
| 典型视频延迟 | 约 60 ms @ 1080P60、80 ms @ 2K60、100 ms @ 4K30 | 相同 |
| 无线网络 | Wi-Fi 6，2.4 GHz / 5 GHz，最高 286 Mbps | 相同 |
| 虚拟设备 | 键盘、鼠标、U 盘、虚拟网卡 | 相同 |
| 接口 | USB-C 数据接口（DP Alt Mode）、USB-C 供电接口（USB-PD） | 相同 |
| 尺寸 / 功耗 | 约 45 × 40 × 15 mm，约 1.6 W @ 4K30 | 约 45 × 40 × 15 mm，约 2.0 W @ 4K30 |
| 记忆织网 | — | 支持 |

## 功能特性

- **浏览器远程控制** —— 基于 WebRTC 的 H.264 / H.265 视频与双向音频，
  并提供 H.264 直连流回退。
- **无需客户端驱动的输入** —— 绝对/相对鼠标、数位笔与多点触控，
  多客户端并发时按会话维护各自的输入路由归属。
- **虚拟介质与虚拟网络** —— 挂载 ISO/IMG 镜像进行远程系统安装，
  与被控端共享文件，并提供虚拟网卡。
- **本地触摸屏界面** —— 基于 C++20 与 LVGL，直接运行在 framebuffer 与 evdev 上，
  用于 Wi-Fi、显示和应用设置。
- **AI 集成** —— 可选的 MCP Server，兼容 MCP 的 AI 工具可获取画面并发送键鼠操作；
  Go+ 额外提供记忆织网，从画面变化中提取可检索的文字与上下文。
- **远程访问** —— 内置 Tailscale，另提供 ZeroTier、NetBird、frp 与 Cloudflare Tunnel
  扩展，无需把管理端口暴露到公网。
- **安全** —— TOTP 双因素认证，应用与固件更新均使用 Ed25519 签名校验。
- **手指机器人** —— 通过 USB-PD 供电接口上的扩展 IO 远程按下被控设备电源键。

MCP 与记忆织网默认关闭，需由用户主动启用。采集的画面可能包含账号、消息等敏感信息，
请只在可信网络与可信 AI 工具中使用。

## 仓库结构

本仓库是该系列的构建与打包入口。各组件构建产物汇总到 `build/`，
随后打包为 Debian 软件包。

| 目录 | 内容 |
| --- | --- |
| [`server/`](server) | Go 后端：API、认证、HID、推流、扩展 |
| [`web/`](web) | React + TypeScript 前端 |
| [`kvm_ui/`](kvm_ui) | 设备本地 LVGL 触摸界面 |
| [`kvm_vin/`](kvm_vin) | 视频采集与编码流水线 |
| [`libkvm/`](libkvm) | 硬件抽象库 |
| [`nano_ocr/`](nano_ocr) | OCR 运行时 |
| [`memfab/`](memfab) | 记忆织网运行时，提供 OCR 支撑的本地记忆 |
| [`debhelper/`](debhelper) | `kvmcomm` 与 `nanokvm-go` 的 Debian 打包 |
| [`support/`](support) | 工具链配置、打包脚本、内核补丁、发布公钥 |
| [`docs/`](docs) | 架构、API 与构建文档 |

`debhelper/packages/kvmcomm/rootfs/kvmcomm/apps` 是一个 Git 子模块。
克隆时请加 `--recurse-submodules`，或在克隆后执行
`git submodule update --init --recursive`。

## 构建

根目录的 Python 构建脚本是唯一正式入口，所有构建与打包命令均在仓库根目录执行。

依赖要求：

| 组件 | 要求 |
| --- | --- |
| `server`、`memfab` | Go 1.26.5+、Python 3、`patchelf`、已配置的 ARMv7 工具链 |
| `web` | Node.js `^20.19.0` 或 `>=22.12.0`、pnpm |
| `libkvm`、`kvm_vin`、`kvm_ui`、`nano_ocr` | Python 3、CMake、Conan、已配置的 ARMv7 工具链 |
| Debian 打包 | `dpkg-dev`、`debhelper` |
| 签名发布 | 支持 Ed25519 `pkey` 与 `pkeyutl -rawin` 的 OpenSSL |

先配置 ARMv7 工具链：

```bash
cd support/scripts
./toolchain_setup.sh
cd ../..
```

回到仓库根目录安装前端依赖，然后构建：

```bash
pnpm --dir web install --frozen-lockfile
python3 build.py build        # 清理后构建全部组件
python3 build.py server       # 或单独构建某个组件
python3 build.py release --version <version>
```

`nano_ocr` 会从 OpenCV 官方仓库下载 4.6.0 并校验 SHA-256，
因此首次构建需要网络访问。FFmpeg 由目标系统提供，不随本仓库分发。

工具链细节、Debian 打包、update bundle 与发布签名说明见
**[`docs/build-packaging.md`](docs/build-packaging.md)**。

## 文档

| 文档 | 说明 |
| --- | --- |
| [构建与打包指南](docs/build-packaging.md) | 工具链配置、组件构建、Debian 打包、update bundle、签名 |
| [更新日志](CHANGELOG.md) | 主要变更与版本历史 |
| [MCP 使用说明（英文）](docs/mcp-usage.md) | 在 AI 客户端中启用并使用 MCP 服务 |
| [存储与文件共享](docs/storage.md) | 虚拟介质、ISO/IMG 挂载与文件传输 API |
| [输入路由架构](docs/input-router.md) | 输入路由、会话归属与已验证的客户端矩阵 |
| [USB HID 说明](docs/kvmgo-device-usb-hid.md) | 鼠标、数位笔与多点触控的 HID 报文描述符 |
| [双因素认证](docs/2fa.md) | TOTP 配置与运行时文件 |

## 兼容性

被控设备的 USB-C 接口必须支持 **DisplayPort Alt Mode** 才能输出画面；
仅支持充电或普通 USB 数据传输的接口无法使用。Apple 设备仅限 wiki 中列出的型号范围
（例如 iPhone 15 及后续机型，且需开启辅助触控）。控制端只需能访问设备网络的现代浏览器。

完整且最新的兼容列表见
[Wiki 兼容设备章节](https://wiki.sipeed.com/hardware/zh/kvm/NanoKVM_Go/introduction.html)。

## 参与贡献

欢迎提交 issue 与 pull request。提交前请确保相关组件可以正常构建且测试通过，
并尽量让每个提交只聚焦一处改动。

## 许可证

NanoKVM-Go 基于 [GNU General Public License v3.0](LICENSE) 发布。
