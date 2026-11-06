#!/usr/bin/env python3
"""
mgmt_diag.py — BT Management Protocol 诊断工具

读取蓝牙控制器信息并显示当前状态，用于调试蓝牙配置问题。
无需 BlueZ (hciconfig/bluetoothctl)，通过 BT Management Protocol 直接查询内核。

使用方法:
    python3 mgmt_diag.py

输出示例:
    BD_ADDR: 38:7a:cc:9b:3e:a9
    Current settings: 0x000000db
    Active: powered connectable discoverable bondable ssp br/edr

关键点:
    - HID 设备正常工作需要 Active 中包含:
      powered + connectable + discoverable + bondable + ssp + br/edr
    - 如果缺少 connectable/discoverable/bondable，说明没有通过 mgmt 正确配置
    - 缺少上述位时先确认 SET_POWERED 已成功，再按顺序配置其他属性
"""
import socket, struct, ctypes, ctypes.util, select

AF_BLUETOOTH = 31
BTPROTO_HCI = 1


def mgmt_open():
    """打开 BT Management 套接字 (HCI channel 3)。

    Python 的 socket.bind() 不支持 hci_channel 参数，
    必须使用 ctypes 手动构造 sockaddr_hci 并调用底层 bind。
    """
    s = socket.socket(AF_BLUETOOTH, socket.SOCK_RAW, BTPROTO_HCI)
    # sockaddr_hci: { sa_family=AF_BLUETOOTH, hci_dev=0xFFFF, hci_channel=3 }
    addr = struct.pack('<HHH', AF_BLUETOOTH, 0xFFFF, 3)
    libc = ctypes.CDLL(ctypes.util.find_library('c'), use_errno=True)
    ret = libc.bind(s.fileno(), addr, len(addr))
    if ret != 0:
        err = ctypes.get_errno()
        s.close()
        raise OSError(err, "mgmt bind failed")
    return s


def mgmt_cmd(s, opcode, index, params=b''):
    """发送 mgmt 命令并等待响应。"""
    s.send(struct.pack('<HHH', opcode, index, len(params)) + params)
    r, _, _ = select.select([s], [], [], 3)
    if r:
        return s.recv(4096)
    return None


def main():
    s = mgmt_open()
    print("Mgmt socket opened")

    # READ_INFO (opcode 0x0004) — 读取控制器详细信息
    d = mgmt_cmd(s, 0x0004, 0)
    if not d or len(d) <= 20:
        print("Bad response:", d[:20].hex() if d else "none")
        s.close()
        return

    # 响应格式: evt_hdr(6) + cmd_opcode(2) + status(1) + controller_info(...)
    # evt_hdr: evt_code(2) + index(2) + plen(2)
    cmd_opcode = struct.unpack_from('<H', d, 6)[0]
    status = d[8]
    print("cmd_complete opcode=0x%04x status=%d" % (cmd_opcode, status))

    if status != 0:
        print("Error: status=%d (0x0D=InvalidParams, 0x0F=NotPowered)" % status)
        s.close()
        return

    # controller_info 从 d[9] 开始:
    #   BD_ADDR(6) + bt_ver(1) + mfg(2) + supported(4) + current(4) + CoD(3) + name(249)
    off = 9
    bd = d[off:off + 6]; off += 6
    print("BD_ADDR:", ':'.join('%02x' % b for b in reversed(bd)))

    bt_ver = d[off]; off += 1
    mfg = struct.unpack_from('<H', d, off)[0]; off += 2
    print("BT ver=%d MFG=%d" % (bt_ver, mfg))

    supported = struct.unpack_from('<I', d, off)[0]; off += 4
    current = struct.unpack_from('<I', d, off)[0]; off += 4
    print("Supported settings: 0x%08x" % supported)
    print("Current settings:   0x%08x" % current)

    flags = {
        0: 'powered', 1: 'connectable', 2: 'fast-conn',
        3: 'discoverable', 4: 'bondable', 5: 'link-security',
        6: 'ssp', 7: 'br/edr', 8: 'hs', 9: 'le',
        10: 'advertising', 11: 'secure-conn', 12: 'debug-keys',
        13: 'privacy', 14: 'config', 15: 'static-addr',
    }
    cur_list = [name for bit, name in sorted(flags.items()) if current & (1 << bit)]
    sup_list = [name for bit, name in sorted(flags.items()) if supported & (1 << bit)]
    print("Active:", ' '.join(cur_list))
    print("Supported:", ' '.join(sup_list))

    # 检查 HID 设备必需的 flags
    required = {'powered', 'connectable', 'discoverable', 'bondable', 'ssp', 'br/edr'}
    missing = required - set(cur_list)
    if missing:
        print("\n⚠️  WARNING: Missing flags for HID: %s" % ', '.join(missing))
        print("   Run bt_hid_demo.py to configure via mgmt protocol.")
    else:
        print("\n✅ All required HID flags are active.")

    cod = d[off:off + 3]; off += 3
    print("CoD: %s" % cod.hex())

    name_raw = d[off:off + 249]
    name = name_raw.split(b'\x00')[0].decode('utf-8', 'ignore')
    print("Name:", repr(name))

    s.close()
    print("Done")


if __name__ == '__main__':
    main()
