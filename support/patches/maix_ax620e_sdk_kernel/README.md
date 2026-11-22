# NanoKVM Go kernel patches

## Touch Screen HID on the older SDK

NanoKVM-Go's Linux 4.19 HID gadget returns zero-filled data for every
`GET_REPORT` request. Windows requires the Touch Screen top-level collection's
Contact Count Maximum feature report to return a non-zero value.

Apply `0001-hid-touchscreen-feature-report.patch` to the official
`sipeed/maix_ax620e_sdk_kernel` source before building NanoKVM-Go firmware. The
patch is based on these exact revisions:

- `sipeed/maix_ax620e_sdk`: `997e55e9b6245d0ff1abdeba6bd0a78af1fdd2e6`
- `sipeed/maix_ax620e_sdk_kernel`: `25f24840e9ce9e9af85a372bad8e5a5905dee749`

The patch makes Feature Report ID 4 return a maximum contact count of 10. These
values must stay synchronized with `configure_hid2` in `/opt/scripts/usbdev.sh`.

## UVC webcam gadget on the Go+

`0002-go-enable-uvc-gadget.patch` enables `CONFIG_USB_CONFIGFS_F_UVC=y` in
the ARM32 NanoAgent board config. `0003-uvc-arm32-clock-division.patch` uses
the kernel's division helper for the vendor driver's 48 MHz timestamp. The
original remainder operation introduces an unresolved `__aeabi_uldivmod` on
ARM32. `0004-uvc-configurable-function-name.patch` adds a per-camera ConfigFS
`function_name` for the USB association and control interface. Set it before
linking the function into a configuration. The October 8 SDK already includes the HID feature-report fix.

| Input | Revision |
| --- | --- |
| [Sipeed SDK](https://github.com/sipeed/maix_ax620e_sdk/tree/8aac11dd693deafa47f56f46cd0d628a4c0e3ef1) | `8aac11dd693deafa47f56f46cd0d628a4c0e3ef1` |
| [Kernel submodule](https://github.com/sipeed/maix_ax620e_sdk_kernel/tree/fd3114bfd3f984b5570fa8a5a564f1770cb8f751) | `fd3114bfd3f984b5570fa8a5a564f1770cb8f751` (Linux 4.19.125) |
| [MSP driver headers](https://github.com/sipeed/maix_ax620e_sdk_msp/tree/71ca5afe0b7db721c1c95a7488db6d9223060365) | `71ca5afe0b7db721c1c95a7488db6d9223060365` |
| Compiler | Arm GNU Toolchain 10.3-2021.07, `arm-none-linux-gnueabihf-gcc` 10.3.1 |

Build on Linux with the compiler on `PATH`, the SDK kernel build dependencies,
`cpio`, and the Python `rsa` module. Starting in the Nomad repository, use a
fresh SDK checkout on a case-sensitive filesystem.

```sh
patch_dir="$PWD/support/patches/maix_ax620e_sdk_kernel"
git clone https://github.com/sipeed/maix_ax620e_sdk.git
cd maix_ax620e_sdk
git checkout --detach 8aac11dd693deafa47f56f46cd0d628a4c0e3ef1
git -c url.https://github.com/.insteadOf=git@github.com: submodule update --init kernel msp
git -C kernel apply "$patch_dir/0002-go-enable-uvc-gadget.patch" "$patch_dir/0003-uvc-arm32-clock-division.patch" "$patch_dir/0004-uvc-configurable-function-name.patch"

sdk="$PWD"
p=AX620Q_emmc_arm32_k419_sipeed_nanoagent
(cd "build/projects/$p" && bash gen_initramfs.sh)
cd kernel/linux
make -f Makefile.kernel p="$p" prepare
make -j6 -f Makefile.kernel p="$p" linux
case "$(uname -m)" in
    aarch64) make -f Makefile.kernel p="$p" "AX_GZIP=qemu-x86_64 $sdk/tools/ax_gzip_tool/ax_gzip" install ;;
    x86_64)  make -f Makefile.kernel p="$p" install ;;
esac
```

The SDK's `install` target writes into `build/out/$p/images/` in the checkout.
Its `boot_signed.bin` is the signed kernel component Sipeed packages as
`kernel.img`. The web UI's offline updater takes an application package.
Testing this component requires booting a Go+ test system. USB enumeration
and streaming also require configuring the gadget and supplying video frames.

The test unit reported boot release v1.0.10. The public SDK revision above
is from the same day as its kernel build, but the SDK's `bootfs/ver` still
identifies v1.0.9. The exact factory source revision remains unconfirmed.
The linker also reports a section mismatch in the vendor sleep driver:
`axera_slp_stat_probe()` calls the init-only `register_persistent_clock()`.
