#!/usr/bin/env python3

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "rootfs/opt/scripts/usbdev.sh"


class USBStorageTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        (self.root / "boot").mkdir()
        self.log = self.root / "events"
        self.manager = self.root / "mount.py"
        self.manager.write_text(
            '#!/bin/sh\nprintf "%s\\n" "$1" >> "$STORAGE_TEST_LOG"\n'
            'if [ "$STORAGE_TEST_EXIT" != 0 ]; then echo "storage mount failed" >&2; fi\n'
            'exit "$STORAGE_TEST_EXIT"\n'
        )
        self.manager.chmod(0o700)
        source = SCRIPT.read_text()
        source = source.replace("/boot/", str(self.root / "boot") + "/")
        source = source.replace("/exfat.img", str(self.root / "exfat.img"))
        source = source.replace("/sys/", str(self.root / "sys") + "/")
        source = source.replace("/kvmcomm/scripts/mount.py", str(self.manager))
        self.functions, self.dispatch = source.rsplit("lock_storage || exit 1", 1)

    def run_shell(self, code, *arguments, storage_exit=0):
        env = dict(os.environ, STORAGE_TEST_LOG=str(self.log), STORAGE_TEST_EXIT=str(storage_exit))
        return subprocess.run(["bash", "-c", code, "storage-test", *arguments], env=env, capture_output=True, text=True)

    def events(self):
        return self.log.read_text().splitlines() if self.log.exists() else []

    def test_prepare_selects_readonly_only_for_whole_disk_export(self):
        (self.root / "exfat.img").touch()
        (self.root / "boot/usb.disk1").touch()
        result = self.run_shell(self.functions + "\nprepare_storage\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.events(), ["readonly"])
        (self.root / "boot/usb.disk0").touch()
        result = self.run_shell(self.functions + "\nprepare_storage\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.events(), ["readonly", "local"])

    def gadget_fixture(self):
        gadget_root = self.root / "sys/kernel/config/usb_gadget"
        gadget_root.mkdir(parents=True)
        udc = self.root / "sys/class/udc"
        udc.mkdir(parents=True)
        (udc / "test-udc").touch()
        # ConfigFS creates these children when a function/config is added.
        return self.functions + r'''
mkdir() {
    command mkdir "$@" || return 1
    local target="${@: -1}"
    case "$target" in
        g0) command mkdir -p g0/functions g0/configs g0/strings g0/os_desc ;;
        configs/c.*) command mkdir -p "$target/strings" ;;
        functions/mass_storage.*) command mkdir -p "$target/lun.0" ;;
        functions/ncm.usb0) command mkdir -p "$target/os_desc/interface.ncm" ;;
    esac
}
set_usb_role() { :; }
gen_mac_addr() { echo 02:00:00:00:00:01; }
start_dhcp_service() { echo "dhcp:$1" >> "$STORAGE_TEST_LOG"; }
'''

    def assert_base_gadget_started(self):
        gadget = self.root / "sys/kernel/config/usb_gadget/g0"
        for function in ("hid.GS0", "hid.GS1", "hid.GS2", "ncm.usb0", "uac2.usb0"):
            self.assertTrue((gadget / "functions" / function).is_dir(), function)
            self.assertTrue((gadget / "configs/c.1" / function).is_symlink(), function)
        self.assertEqual((gadget / "UDC").read_text().strip(), "test-udc")
        self.assertIn("dhcp:ncm", self.events())
        return gadget

    def test_failed_storage_preparation_preserves_hid_network_and_audio(self):
        for disk in (None, "disk0", "disk1"):
            with self.subTest(disk=disk):
                fixture = self.gadget_fixture()
                for marker in ("usb.ncm", "usb.uac2"):
                    (self.root / "boot" / marker).touch()
                (self.root / "exfat.img").touch()
                if disk:
                    (self.root / "boot" / f"usb.{disk}").touch()
                result = self.run_shell(fixture + "\nhid_start\n", storage_exit=1)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("storage mount failed", result.stderr)
                self.assertIn("starting USB without mass storage", result.stderr)
                gadget = self.assert_base_gadget_started()
                self.assertFalse((gadget / "functions/mass_storage.disk0").exists())
                self.assertFalse((gadget / "functions/mass_storage.disk1").exists())
                if disk:
                    self.assertTrue((self.root / "boot" / f"usb.{disk}").exists())
                # Each case starts from a fresh fake ConfigFS tree.
                shutil.rmtree(self.root / "sys")
                if disk:
                    (self.root / "boot" / f"usb.{disk}").unlink()

    def test_successful_readonly_preparation_still_exports_disk(self):
        fixture = self.gadget_fixture()
        for marker in ("usb.ncm", "usb.uac2", "usb.disk1"):
            (self.root / "boot" / marker).touch()
        (self.root / "exfat.img").touch()
        result = self.run_shell(fixture + "\nhid_start\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        gadget = self.assert_base_gadget_started()
        self.assertEqual(
            (gadget / "functions/mass_storage.disk1/lun.0/file").read_text().strip(),
            str(self.root / "exfat.img"),
        )
        self.assertEqual(self.events()[0], "readonly")

    def test_failed_local_restore_does_not_block_stop_or_hid_only(self):
        stubs = '\nhid_stop() { echo detach >> "$STORAGE_TEST_LOG"; }\nhid_only_start() { echo hid-only >> "$STORAGE_TEST_LOG"; }\n'
        for action in ("stop", "hid-only"):
            with self.subTest(action=action):
                self.log.unlink(missing_ok=True)
                result = self.run_shell(self.functions + stubs + self.dispatch, action, storage_exit=1)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("local image storage is unavailable", result.stderr)
                self.assertEqual(self.events(), ["detach", "local"] + (["hid-only"] if action == "hid-only" else []))

    def test_failed_detach_never_restores_local_writes_or_restarts_usb(self):
        stubs = '\nhid_stop() { return 1; }\nhid_start() { echo start >> "$STORAGE_TEST_LOG"; }\nhid_only_start() { echo hid-only >> "$STORAGE_TEST_LOG"; }\n'
        for action in ("stop", "restart", "hid-only"):
            with self.subTest(action=action):
                result = self.run_shell(self.functions + stubs + self.dispatch, action)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.events(), [])


if __name__ == "__main__":
    unittest.main()
