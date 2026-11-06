#!/usr/bin/env python3

import importlib.util
import subprocess
import sys
import types
import unittest
import tempfile
import os
from pathlib import Path
from unittest import mock


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "rootfs"
    / "kvmcomm"
    / "scripts"
    / "mount.py"
)

fake_dbus = types.ModuleType("dbus")
fake_dbus.SystemBus = mock.Mock()
fake_dbus.Interface = mock.Mock()
sys.modules.setdefault("dbus", fake_dbus)

spec = importlib.util.spec_from_file_location("kvmcomm_mount", SCRIPT_PATH)
mount = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mount)


class RunTests(unittest.TestCase):
    def test_run_passes_literal_arguments_without_a_shell(self):
        command = ["mkfs.exfat", "/tmp/image with spaces;$(touch nope).img"]
        completed = subprocess.CompletedProcess(command, 0, "formatted\n", "")

        with mock.patch.object(mount.subprocess, "run", return_value=completed) as run:
            self.assertEqual(mount.run(command, timeout=7), 0)

        run.assert_called_once_with(
            command,
            shell=False,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=7,
        )

    def test_run_preserves_failure_code_and_check_false_behavior(self):
        command = ["command", "literal;argument"]
        failed = subprocess.CompletedProcess(command, 23, "", "failure")

        with mock.patch.object(mount.subprocess, "run", return_value=failed):
            self.assertEqual(mount.run(command), 23)
            self.assertEqual(mount.run(command, check=False), 23)

    def test_run_reports_timeout_and_missing_executable(self):
        command = ["slow-command", "argument"]
        with mock.patch.object(
            mount.subprocess,
            "run",
            side_effect=subprocess.TimeoutExpired(command, 3),
        ):
            self.assertEqual(mount.run(command, timeout=3), -1)

        with mock.patch.object(
            mount.subprocess,
            "run",
            side_effect=FileNotFoundError("missing"),
        ):
            self.assertEqual(mount.run(command), 127)


class ImageTests(unittest.TestCase):
    def test_create_image_formats_literal_path(self):
        image = "/tmp/image with spaces;$(touch nope).img"
        opened = mock.mock_open()
        opened.return_value.fileno.return_value = 17

        with (
            mock.patch("builtins.open", opened),
            mock.patch.object(mount.os, "ftruncate") as ftruncate,
            mock.patch.object(mount, "run", return_value=0) as run,
        ):
            mount.create_exfat_image(image, 8)

        opened.assert_called_once_with(image, "wb")
        ftruncate.assert_called_once_with(17, 8 * 1024 * 1024)
        run.assert_called_once_with(["mkfs.exfat", image])

    def test_default_size_parses_df_in_python_for_literal_target(self):
        target = "/tmp/target with spaces;$(touch nope)"
        output = (
            "Filesystem 1M-blocks Used Available Use% Mounted on\n"
            "/dev/root 20000M 1000M 19000M 5% /tmp/target with spaces\n"
        )
        completed = subprocess.CompletedProcess([], 0, output, "")

        with mock.patch.object(mount.subprocess, "run", return_value=completed) as run:
            size = mount.get_default_image_size_mb(
                target_dir=target,
                reserve_mb=5000,
                min_required_mb=5000,
            )

        self.assertEqual(size, 14000)
        run.assert_called_once_with(
            ["df", "-BM", "--", target],
            shell=False,
            capture_output=True,
            text=True,
            check=True,
        )

    def test_default_size_rejects_failure_malformed_output_and_low_space(self):
        failed = subprocess.CalledProcessError(9, ["df"])
        malformed = subprocess.CompletedProcess([], 0, "Filesystem\ninvalid\n", "")
        low = subprocess.CompletedProcess(
            [],
            0,
            "Filesystem 1M-blocks Used Available Use% Mounted on\n"
            "/dev/root 6000M 1000M 5000M 20% /\n",
            "",
        )

        for result in (failed, malformed, low):
            side_effect = result if isinstance(result, Exception) else None
            return_value = None if side_effect else result
            with (
                self.subTest(result=result),
                mock.patch.object(
                    mount.subprocess,
                    "run",
                    side_effect=side_effect,
                    return_value=return_value,
                ),
                self.assertRaises(SystemExit) as exit_status,
            ):
                mount.get_default_image_size_mb()
            self.assertEqual(exit_status.exception.code, 1)


class MountAndCliTests(unittest.TestCase):
    def setUp(self):
        mount.debug = False

    def test_manage_data_mount_waits_for_systemd_jobs(self):
        manager = mock.Mock()
        systemd = mock.Mock()
        bus = mock.Mock()
        bus.get_object.return_value = systemd
        events = []

        with (
            mock.patch.object(mount.dbus, "SystemBus", return_value=bus),
            mock.patch.object(mount.dbus, "Interface", return_value=manager) as interface,
            mock.patch.object(mount, "run", side_effect=lambda command, **kwargs: events.append(command) or 0) as run,
            mock.patch.object(mount, "wait_for_data_mount", side_effect=lambda mounted: events.append(mounted)),
        ):
            mount.manage_data_mount("enable")
            mount.manage_data_mount("disable")

        interface.assert_has_calls(
            [mock.call(systemd, "org.freedesktop.systemd1.Manager")] * 2
        )
        manager.EnableUnitFiles.assert_called_once_with(["data.mount"], False, False)
        manager.StartUnit.assert_not_called()
        manager.StopUnit.assert_not_called()
        manager.DisableUnitFiles.assert_called_once_with(["data.mount"], False)
        self.assertEqual(events, [
            ["systemctl", "start", "data.mount"], True,
            ["systemctl", "stop", "data.mount"], False,
        ])
        self.assertTrue(all(call.kwargs["timeout"] == mount.data_mount_timeout_seconds for call in run.call_args_list))

    def test_failed_systemd_job_cannot_be_mistaken_for_a_completed_mount(self):
        for action in ("enable", "disable", "unmount"):
            with (
                self.subTest(action=action),
                mock.patch.object(mount, "run", return_value=1),
                mock.patch.object(mount, "wait_for_data_mount") as wait,
                self.assertRaisesRegex(RuntimeError, "Failed to (start|stop) data.mount"),
            ):
                mount.manage_data_mount(action)
            wait.assert_not_called()

    def test_clear_data_directory_uses_filesystem_boundaries(self):
        def is_file(path):
            return path == "/data/file.img"

        def is_dir(path):
            return path == "/data/folder"

        with (
            mock.patch.object(mount.os.path, "exists", return_value=True),
            mock.patch.object(mount.os, "listdir", return_value=["file.img", "folder"]),
            mock.patch.object(mount.os.path, "isfile", side_effect=is_file),
            mock.patch.object(mount.os.path, "islink", return_value=False),
            mock.patch.object(mount.os.path, "isdir", side_effect=is_dir),
            mock.patch.object(mount.os, "unlink") as unlink,
            mock.patch("shutil.rmtree") as rmtree,
        ):
            mount.clear_data_directory()

        unlink.assert_called_once_with("/data/file.img")
        rmtree.assert_called_once_with("/data/folder")

    def run_cli(self, *arguments):
        with (
            mock.patch.object(sys, "argv", [str(SCRIPT_PATH), *arguments]),
            mock.patch.object(mount, "storage_lock"),
            mock.patch.object(mount, "backing_file", return_value=""),
        ):
            mount.main()

    def test_start_cli_creates_image_clears_data_and_enables_mount(self):
        image = "/tmp/image with spaces;$(touch nope).img"
        with (
            mock.patch.object(mount.os.path, "exists", return_value=False),
            mock.patch.object(mount, "create_exfat_image") as create,
            mock.patch.object(mount, "clear_data_directory") as clear,
            mock.patch.object(mount, "manage_data_mount") as manage,
            mock.patch.object(mount, "validate_image_size_mb"),
            mock.patch.object(mount, "restore_local_storage") as restore,
        ):
            self.run_cli("start", "--size", "64", "--file", image)

        create.assert_called_once_with(img_file=image, size_mb=64)
        clear.assert_called_once_with()
        manage.assert_called_once_with("enable")
        restore.assert_called_once_with()

    def test_stop_restart_and_remove_cli_preserve_action_order(self):
        image = "/tmp/literal;image.img"
        with (
            mock.patch.object(mount, "manage_data_mount") as manage,
            mock.patch.object(mount, "restore_local_storage") as restore,
        ):
            self.run_cli("stop")
            self.run_cli("restart")

        self.assertEqual(
            manage.call_args_list,
            [
                mock.call("disable"),
            ],
        )
        restore.assert_called_once_with(refresh=True)

        with (
            mock.patch.object(mount, "manage_data_mount") as manage,
            mock.patch.object(mount.os.path, "exists", return_value=True),
            mock.patch.object(mount.os, "remove") as remove,
        ):
            self.run_cli("remove", "--file", image)

        manage.assert_called_once_with("disable")
        remove.assert_called_once_with(image)


class StorageOwnershipTests(unittest.TestCase):
    def test_mount_inspection_verifies_exfat_and_loop_backing(self):
        for backing, expected in (("/exfat.img\n", True), ("/different.img\n", False)):
            with self.subTest(backing=backing):
                replies = [
                    subprocess.CompletedProcess([], 0, '{"filesystems":[{"source":"/dev/loop0","fstype":"exfat","options":"ro,relatime"}]}', ''),
                    subprocess.CompletedProcess([], 0, backing, ''),
                ]
                with mock.patch.object(mount.subprocess, "run", side_effect=replies):
                    if expected:
                        self.assertEqual(mount.data_mount()["options"], "ro,relatime")
                    else:
                        with self.assertRaisesRegex(RuntimeError, "backing file"):
                            mount.data_mount()

    def test_readonly_mount_success_sets_recovery_marker(self):
        with tempfile.TemporaryDirectory() as directory:
            marker = str(Path(directory) / "usb")
            with (
                mock.patch.object(mount, "usb_access_marker", marker),
                mock.patch.object(mount.os.path, "isfile", return_value=True),
                mock.patch.object(mount, "backing_file", return_value=""),
                mock.patch.object(mount, "data_mount", return_value={"options": "ro"}),
                mock.patch.object(mount, "run", return_value=0),
            ):
                mount.prepare_usb_storage()
            self.assertTrue(Path(marker).exists())

    def test_remount_failure_never_marks_usb_ready(self):
        with (
            mock.patch.object(mount.os.path, "isfile", return_value=True),
            mock.patch.object(mount, "backing_file", return_value=""),
            mock.patch.object(mount, "data_mount", return_value={"options": "rw"}),
            mock.patch.object(mount, "run", return_value=1),
            mock.patch("builtins.open") as opened,
        ):
            with self.assertRaisesRegex(RuntimeError, "read-only"):
                mount.prepare_usb_storage()
            opened.assert_not_called()

    def test_usb_export_checks_actual_readonly_before_marking(self):
        events = []
        with (
            mock.patch.object(mount.os.path, "isfile", return_value=True),
            mock.patch.object(mount, "backing_file", return_value=""),
            mock.patch.object(mount, "data_mount", return_value={"options": "rw"}),
            mock.patch.object(mount, "run", side_effect=lambda cmd: events.append(cmd) or 0),
            mock.patch("builtins.open") as opened,
        ):
            with self.assertRaisesRegex(RuntimeError, "read-only"):
                mount.prepare_usb_storage()
            opened.assert_not_called()
        self.assertEqual(events, [["mount", "-o", "remount,ro", "/data"]])

    def test_restore_refuses_live_usb_backing(self):
        with (
            mock.patch.object(mount, "backing_file", return_value="/exfat.img"),
            mock.patch.object(mount, "run") as run,
        ):
            with self.assertRaisesRegex(RuntimeError, "Disconnect"):
                mount.restore_local_storage()
            run.assert_not_called()

    def test_restore_replaces_readonly_loop_and_clears_marker_after_verification(self):
        events = []
        with (
            mock.patch.object(mount, "backing_file", return_value=""),
            mock.patch.object(mount.os.path, "exists", return_value=True),
            mock.patch.object(mount, "data_mount", return_value={"options": "ro"}),
            mock.patch.object(mount, "manage_data_mount", side_effect=lambda action: events.append(action)),
            mock.patch.object(mount, "run", side_effect=lambda cmd: events.append(cmd) or 0),
            mock.patch.object(mount, "verify_data_mode", side_effect=lambda mode: events.append(mode)),
            mock.patch.object(mount.os, "remove", side_effect=lambda path: events.append(path)),
        ):
            mount.restore_local_storage()
        self.assertEqual(events[0], "unmount")
        self.assertEqual(events[1][-2:], ["/exfat.img", "/data"])
        self.assertIn("loop,rw", events[1][4])
        self.assertEqual(events[2:], [True, mount.usb_access_marker])

    def test_busy_unmount_cannot_fall_through_to_writable_mount(self):
        with (
            mock.patch.object(mount, "backing_file", return_value=""),
            mock.patch.object(mount.os.path, "exists", return_value=True),
            mock.patch.object(mount, "data_mount", return_value={"options": "ro"}),
            mock.patch.object(mount, "manage_data_mount", side_effect=RuntimeError("busy")),
            mock.patch.object(mount, "run") as run,
            mock.patch.object(mount.os, "remove") as remove,
        ):
            with self.assertRaisesRegex(RuntimeError, "busy"):
                mount.restore_local_storage()
            run.assert_not_called()
            remove.assert_not_called()

    def test_refresh_refuses_both_virtual_disk_and_selected_image(self):
        for disk in ("disk0", "disk1"):
            with (
                self.subTest(disk=disk),
                mock.patch.object(mount, "backing_file", side_effect=lambda name: "/data/image.iso" if name == disk else ""),
                mock.patch.object(mount, "restore_local_storage") as restore,
            ):
                with self.assertRaises(RuntimeError):
                    mount.execute_action(types.SimpleNamespace(command="restart"))
                restore.assert_not_called()

    def test_lock_rejects_an_active_file_transfer(self):
        with tempfile.TemporaryDirectory() as directory:
            lock_path = str(Path(directory) / "lock")
            with (
                mock.patch.object(mount, "storage_lock_path", lock_path),
                mock.patch.dict(os.environ, {"NANOKVM_STORAGE_LOCK_FD": ""}),
                open(lock_path, "w") as transfer,
            ):
                mount.fcntl.flock(transfer, mount.fcntl.LOCK_SH)
                with self.assertRaisesRegex(RuntimeError, "busy"):
                    with mount.storage_lock():
                        self.fail("transition acquired lock during file transfer")
            with mock.patch.object(mount, "storage_lock_path", lock_path):
                with mount.storage_lock():
                    pass


class SharedReadonlyRefreshTests(unittest.TestCase):
    def test_refresh_recreates_only_a_verified_readonly_mount(self):
        events = []
        with (
            mock.patch.object(mount, "backing_file", side_effect=lambda disk: mount.data_image_path if disk == "disk1" else ""),
            mock.patch.object(mount, "verify_data_mode", side_effect=lambda writable: events.append(("verify", writable))),
            mock.patch.object(mount, "manage_data_mount", side_effect=lambda action: events.append(("mount", action))),
        ):
            mount.refresh_readonly_storage()
        self.assertEqual(events, [("verify", False), ("mount", "unmount"), ("mount", "enable"), ("verify", False)])

    def test_refresh_refuses_a_writable_mount_or_another_backing(self):
        for backing, invalid_mode in (("/other.img", False), (mount.data_image_path, True)):
            with (
                self.subTest(backing=backing, invalid_mode=invalid_mode),
                mock.patch.object(mount, "backing_file", side_effect=lambda disk: backing if disk == "disk1" else ""),
                mock.patch.object(mount, "verify_data_mode", side_effect=RuntimeError("writable") if invalid_mode else None),
                mock.patch.object(mount, "manage_data_mount") as manage,
                self.assertRaises(RuntimeError),
            ):
                mount.refresh_readonly_storage()
            manage.assert_not_called()


if __name__ == "__main__":
    unittest.main()
