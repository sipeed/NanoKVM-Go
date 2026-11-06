# Touch Screen HID kernel patch

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
