from __future__ import annotations

import fcntl
import os
import re
import shutil
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
REPOSITORY_TEMP_PARENT = ROOT / "build/test-tmp"
FIRMWARE_SCRIPT = (
    ROOT
    / "debhelper/packages/kvmcomm/rootfs/kvmcomm/scripts/firmware_update.sh"
)
KVMCOMM_SCRIPT = (
    ROOT / "debhelper/packages/kvmcomm/rootfs/kvmcomm/scripts/kvmcomm.sh"
)
RECOVERY_SCRIPT = (
    ROOT
    / "debhelper/packages/kvmcomm/rootfs/kvmcomm/scripts/reset_to_default.sh"
)


def repository_temporary_directory() -> tempfile.TemporaryDirectory:
    REPOSITORY_TEMP_PARENT.mkdir(parents=True, exist_ok=True)
    return tempfile.TemporaryDirectory(
        prefix=".test-update-trust-", dir=REPOSITORY_TEMP_PARENT
    )


class FirmwareUpdateTrustTest(unittest.TestCase):
    def test_factory_reset_keeps_packages_and_restarts_core_services(self) -> None:
        source = RECOVERY_SCRIPT.read_text(encoding="utf-8")

        for obsolete in (
            "curl ",
            "download_packages",
            "verify_recovery_bundle",
            "reinstall_packages",
            "dpkg -i",
            "apt-get install -f",
            "systemctl reboot",
            "lt7911_info/edid",
            "rm -rf /kvmcomm",
            "rm -rf /kvmapp",
            "systemctl mask --runtime",
        ):
            self.assertNotIn(obsolete, source)

        self.assertIn("stop_and_disable_optional_services", source)
        self.assertIn("nano_ocr.service", source)
        self.assertIn("memfab.service", source)
        optional_services = self.function_source(
            source, "stop_and_disable_optional_services"
        )
        self.assertIn("ssh.socket", optional_services)
        self.assertIn("ssh.service", optional_services)
        self.assertIn('systemctl disable --now "$service"', optional_services)
        self.assertLess(
            optional_services.index("ssh.socket"),
            optional_services.index("ssh.service"),
        )
        self.assertIn("reset_kvm_configuration", source)
        self.assertIn("restore_usbdev_if_needed", source)
        self.assertIn('systemctl restart "$service"', source)
        self.assertNotIn("systemctl stop kvmcomm.service", source)
        self.assertNotIn("systemctl stop nanokvm-go.service", source)

        wifi_reset = self.function_source(source, "reset_wifi_private_data")
        self.assertIn("/opt/scripts/wifi.sh clear", wifi_reset)
        self.assertNotIn("wpa_supplicant.conf", wifi_reset)

        config_reset = source.rindex("reset_kvm_configuration")
        final_marker = source.rindex("remove_reset_marker")
        final_release = source.rindex("release_factory_reset_lock")
        final_restart = source.rindex("restart_core_services")
        self.assertLess(config_reset, final_marker)
        self.assertLess(final_marker, final_release)
        self.assertLess(final_release, final_restart)

    def test_firmware_uses_only_embedded_archive_signatures_before_extraction(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        for obsolete in (
            ".json.sig",
            "--signature",
            "--allow-unsigned",
            "UPDATE_METADATA_STATE",
            "legacy unsigned",
            "legacy-unsigned",
            ".tar.xz",
        ):
            self.assertNotIn(obsolete, source)

        verify_source = self.function_source(source, "verify_firmware_pair")
        self.assertIn("firmware", verify_source)
        self.assertIn('--metadata "$metadata_path"', verify_source)
        self.assertIn('--artifact "$artifact_path"', verify_source)
        self.assertIn('--keyring "$UPDATE_KEYRING"', verify_source)
        self.assertIn('--requested-version "$requested_version"', verify_source)

        prepare_source = self.function_source(source, "prepare_firmware_pair")
        self.assertLess(
            prepare_source.index("verify_firmware_pair"),
            prepare_source.index("extract_verified_firmware"),
        )
        extract_source = self.function_source(source, "extract_verified_firmware")
        self.assertIn("tar ", extract_source)

    def test_firmware_uboot_policy_accepts_only_canonical_verifier_output(self) -> None:
        for policy in ("false", "true"):
            with self.subTest(policy=policy):
                result, logs = self.run_firmware_command(
                    ["fetch", "v1.0.1"], verifier_update_uboot=policy
                )
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertNotEqual(logs["tar"], "")

        for output_mode in ("missing", "string", "extra", "multiple"):
            with self.subTest(output_mode=output_mode):
                result, logs = self.run_firmware_command(
                    ["fetch", "v1.0.1"], verifier_output_mode=output_mode
                )
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(
                    "Firmware verifier returned an unexpected result",
                    result.stdout,
                )
                self.assertEqual(logs["tar"], "")

    def test_failed_prepare_clears_previous_uboot_policy_and_tree(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        prepare = self.function_source(source, "prepare_firmware_pair")
        with repository_temporary_directory() as directory:
            root = Path(directory)
            harness = root / "prepare-policy-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eo pipefail
                FIRMWARE_EXTRACT_DIR="old-tree"
                FIRMWARE_UPDATE_UBOOT="true"
                VERIFIED_FIRMWARE_UPDATE_UBOOT="false"
                verify_calls=0
                validate_firmware_pair_directory() {{ return 0; }}
                verify_firmware_pair() {{
                    verify_calls=$((verify_calls + 1))
                    VERIFIED_FIRMWARE_UPDATE_UBOOT="false"
                    if [ "$verify_calls" -eq 1 ]; then
                        VERIFIED_FIRMWARE_UPDATE_UBOOT="true"
                        return 0
                    fi
                    return 1
                }}
                extract_verified_firmware() {{
                    FIRMWARE_EXTRACT_DIR="extracted-$2"
                }}
                {prepare}
                prepare_firmware_pair first 1.0.1 false
                [ "$FIRMWARE_UPDATE_UBOOT" = "true" ]
                [ "$FIRMWARE_EXTRACT_DIR" = "extracted-1.0.1" ]
                if prepare_firmware_pair second 1.0.2 false; then
                    exit 90
                fi
                [ "$FIRMWARE_UPDATE_UBOOT" = "false" ]
                [ -z "$FIRMWARE_EXTRACT_DIR" ]
                """,
            )
            result = subprocess.run(
                ["bash", str(harness)],
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_uboot_policy_controls_both_boot_media_without_skipping_payloads(
        self,
    ) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        flash = self.function_source(source, "flash_verified_firmware")
        with repository_temporary_directory() as directory:
            root = Path(directory)
            (root / "firmware").mkdir()
            (root / "firmware/u-boot_signed.bin").write_bytes(b"uboot")
            (root / "firmware/board.dtb").write_bytes(b"dtb")
            (root / "firmware/boot_signed.bin").write_bytes(b"kernel")
            (root / "overlay/boot").mkdir(parents=True)
            (root / "overlay/boot/ver").write_text("v1.0.1\n", encoding="utf-8")
            harness = root / "flash-policy-harness.sh"
            operations = root / "operations.log"
            marker = root / "reboot-required"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eo pipefail
                FIRMWARE_UPDATE_UBOOT="$1"
                ROOT_DEVICE="$2"
                UBOOT_FILE=firmware/u-boot_signed.bin
                if [ "${{3:-}}" = "missing-uboot" ]; then
                    UBOOT_FILE=firmware/missing-uboot.bin
                fi
                DTB_FILE=firmware/board.dtb
                KERNEL_FILE=firmware/boot_signed.bin
                check_sum() {{ return 0; }}
                grep() {{ printf 'root=%s\n' "$ROOT_DEVICE"; }}
                mktemp() {{ : >"$MARKER"; printf '%s\n' "$MARKER"; }}
                flash_image() {{ printf 'flash:%s\n' "$1" >>"$OPERATIONS"; }}
                update_sd_uboot() {{ printf 'sd-uboot\n' >>"$OPERATIONS"; }}
                cp() {{ printf 'copy:%s\n' "$*" >>"$OPERATIONS"; }}
                rsync() {{ printf 'rsync\n' >>"$OPERATIONS"; }}
                sync() {{ printf 'sync\n' >>"$OPERATIONS"; }}
                mv() {{ printf 'marker\n' >>"$OPERATIONS"; }}
                rm() {{ return 0; }}
                {flash}
                cd "$WORK_DIR"
                flash_verified_firmware
                """,
            )
            base_env = os.environ.copy()
            base_env.update(
                {
                    "WORK_DIR": str(root),
                    "OPERATIONS": str(operations),
                    "MARKER": str(marker),
                }
            )

            cases = (
                ("false", "/dev/mmcblk0p17", False),
                ("true", "/dev/mmcblk0p17", True),
                ("false", "/dev/mmcblk1p2", False),
                ("true", "/dev/mmcblk1p2", True),
            )
            for policy, root_device, expect_uboot in cases:
                with self.subTest(policy=policy, root_device=root_device):
                    operations.unlink(missing_ok=True)
                    result = subprocess.run(
                        ["bash", str(harness), policy, root_device],
                        env=base_env,
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    log = operations.read_text(encoding="utf-8")
                    if root_device == "/dev/mmcblk0p17":
                        self.assertIn("flash:firmware/board.dtb", log)
                        self.assertIn("flash:firmware/boot_signed.bin", log)
                        self.assertEqual(
                            "flash:firmware/u-boot_signed.bin" in log,
                            expect_uboot,
                        )
                    else:
                        self.assertIn("copy:firmware/board.dtb /boot/dtb.img", log)
                        self.assertIn(
                            "copy:firmware/boot_signed.bin /boot/kernel.img", log
                        )
                        self.assertEqual("sd-uboot" in log, expect_uboot)
                    self.assertIn("rsync", log)
                    self.assertIn("sync", log)

            operations.unlink(missing_ok=True)
            invalid = subprocess.run(
                ["bash", str(harness), "invalid", "/dev/mmcblk1p2"],
                env=base_env,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertNotEqual(invalid.returncode, 0)
            self.assertIn("Invalid verified U-Boot update policy", invalid.stdout)
            self.assertFalse(operations.exists())

            operations.unlink(missing_ok=True)
            missing = subprocess.run(
                [
                    "bash",
                    str(harness),
                    "true",
                    "/dev/mmcblk1p2",
                    "missing-uboot",
                ],
                env=base_env,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertNotEqual(missing.returncode, 0)
            self.assertIn("U-Boot update was requested", missing.stdout)
            self.assertFalse(operations.exists())

    def test_cache_replaces_same_archive_when_uboot_policy_changes(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        publish = self.function_source(source, "publish_firmware_cache")
        validate = self.function_source(source, "validate_firmware_pair_directory")
        with repository_temporary_directory() as directory:
            root = Path(directory)
            cache_dir = root / "cache"
            candidate_dir = root / "candidate"
            target_dir = cache_dir / "firmware_1.0.1"
            target_dir.mkdir(parents=True)
            candidate_dir.mkdir()
            for pair_dir, metadata_value in (
                (target_dir, "cached-policy"),
                (candidate_dir, "candidate-policy"),
            ):
                (pair_dir / "axera_firmware_v1.0.1.tar.gz").write_bytes(
                    b"same-archive"
                )
                (pair_dir / "firmware_1.0.1.json").write_text(
                    metadata_value, encoding="utf-8"
                )
            harness = root / "publish-policy-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eo pipefail
                CACHE_BASE_DIR="$CACHE_DIR"
                FIRMWARE_UPDATE_UBOOT=true
                verify_calls=0
                {validate}
                verify_firmware_pair() {{
                    verify_calls=$((verify_calls + 1))
                    if [ "$verify_calls" -eq 1 ]; then
                        VERIFIED_FIRMWARE_UPDATE_UBOOT=false
                    else
                        VERIFIED_FIRMWARE_UPDATE_UBOOT=true
                    fi
                    return 0
                }}
                {publish}
                publish_firmware_cache "$CANDIDATE_DIR" 1.0.1 false
                [ ! -d "$CANDIDATE_DIR" ]
                [ "$(cat "$CACHE_DIR/firmware_1.0.1/firmware_1.0.1.json")" = candidate-policy ]
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "CACHE_DIR": str(cache_dir),
                    "CANDIDATE_DIR": str(candidate_dir),
                }
            )
            result = subprocess.run(
                ["bash", str(harness)],
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_sd_uboot_update_uses_a_small_private_mount_transaction(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        update = self.function_source(source, "update_sd_uboot")
        worker = self.function_source(source, "perform_private_sd_uboot_update")
        mount_helper = self.function_source(source, "validate_sd_uboot_mount")
        sync_helper = self.function_source(source, "sync_sd_uboot_filesystem")
        sync_writes_helper = self.function_source(
            source, "enable_sd_uboot_sync_writes"
        )
        replacement = self.function_source(source, "safe_replace_sd_uboot")
        flash = self.function_source(source, "flash_verified_firmware")
        flash_image = self.function_source(source, "flash_image")

        self.assertIn('local device="/dev/mmcblk1p1"', update)
        self.assertIn('flock -n "$lock_fd"', update)
        self.assertIn('mktemp -d "/run/.nanokvm-sd-boot.XXXXXX"', update)
        self.assertIn("unshare --mount --propagation private", update)
        self.assertIn('NANOKVM_SD_UBOOT_LOCK_FD="$lock_fd"', update)
        self.assertIn("__sd_uboot_update", update)
        self.assertIn("! validate_sd_uboot_worker_lock", source)

        self.assertIn(
            'mount -t vfat -o rw,nosuid,nodev,noexec,async "$device" "$mount_dir"',
            worker,
        )
        self.assertIn("validate_sd_uboot_mount", worker)
        self.assertIn("safe_replace_sd_uboot", worker)
        self.assertIn('umount "$mount_dir"', worker)
        self.assertIn('validate_sd_uboot_mount "$mount_dir" "$device"', sync_helper)
        self.assertIn('sync -f -- "$mount_dir"', sync_helper)
        self.assertIn(
            "mount -t vfat -o remount,rw,nosuid,nodev,noexec,sync",
            sync_writes_helper,
        )
        self.assertIn(
            'validate_sd_uboot_mount "$mount_dir" "$device" async',
            sync_writes_helper,
        )
        self.assertIn(
            'validate_sd_uboot_mount "$mount_dir" "$device" sync',
            sync_writes_helper,
        )
        for option in ("sync", "nosuid", "nodev", "noexec"):
            self.assertIn(option, mount_helper)
        self.assertIn("enable_sd_uboot_sync_writes", replacement)
        self.assertLess(
            replacement.index("enable_sd_uboot_sync_writes"),
            replacement.index('mv -fT --no-copy -- "$candidate_file"'),
        )

        bare_sync = re.compile(
            r"(?m)(?<![-\w])sync(?=\s*(?:\|\||&&|;|$))"
        )
        self.assertIsNone(bare_sync.search(worker))

        staging = replacement[
            replacement.index('mktemp ".nkv-uboot-candidate.XXXXXX"') :
            replacement.index("enable_sd_uboot_sync_writes")
        ]
        self.assertEqual(staging.count("sync_sd_uboot_filesystem"), 1)
        self.assertIn(
            'mv -fT --no-copy -- "$candidate_file" "$target_file"',
            replacement,
        )

        self.assertLess(
            flash.index("! sync"),
            flash.index('mv -fT --no-copy -- "$reboot_marker_candidate"'),
        )
        self.assertIn('mktemp "/var/run/.reboot-required.XXXXXX"', flash)
        self.assertIn("Failed to commit firmware overlay", flash)
        self.assertIn("if ! dd ", flash_image)
        self.assertNotIn('cp "$UBOOT_FILE" /boot/uboot.bin', source)

    def test_sd_uboot_mount_checks_fail_closed(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        mount_helper = self.function_source(source, "validate_sd_uboot_mount")
        unmounted_helper = self.function_source(
            source, "sd_uboot_device_is_unmounted"
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            mount_dir = root / "mount"
            bin_dir = root / "bin"
            mount_dir.mkdir()
            bin_dir.mkdir()
            self.write_executable(
                bin_dir / "findmnt",
                """
                #!/bin/bash
                set -u
                if [ "$FINDMNT_EXIT" = "0" ]; then
                    printf '%b\n' "$FINDMNT_RECORD"
                fi
                exit "$FINDMNT_EXIT"
                """,
            )
            mount_harness = root / "validate-mount.sh"
            self.write_executable(
                mount_harness,
                f"""
                #!/bin/bash
                set -eo pipefail
                {mount_helper}
                validate_sd_uboot_mount "$MOUNT_DIR" /dev/mmcblk1p1 "$WRITE_MODE"
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{bin_dir}:{env['PATH']}",
                    "MOUNT_DIR": str(mount_dir),
                    "FINDMNT_EXIT": "0",
                    "WRITE_MODE": "any",
                }
            )
            valid_async = (
                f"{mount_dir} /dev/mmcblk1p1 vfat "
                "rw,nosuid,nodev,noexec,relatime"
            )
            valid_sync = (
                f"{mount_dir} /dev/mmcblk1p1 vfat "
                "rw,nosuid,nodev,noexec,sync"
            )
            records = (
                ("expected-async", valid_async, "async", True),
                ("expected-sync", valid_sync, "sync", True),
                ("wrong-device", valid_async.replace("mmcblk1p1", "mmcblk0p1"), "async", False),
                ("wrong-filesystem", valid_async.replace(" vfat ", " ext4 "), "async", False),
                ("read-only", valid_async.replace(" rw,", " ro,"), "async", False),
                ("missing-nodev", valid_async.replace(",nodev", ""), "async", False),
                ("async-with-sync", valid_sync, "async", False),
                ("sync-without-sync", valid_async, "sync", False),
                ("multiple", f"{valid_async}\\n{valid_sync}", "any", False),
                ("invalid-mode", valid_async, "invalid", False),
            )
            for name, record, write_mode, expected_success in records:
                with self.subTest(mount=name):
                    env["FINDMNT_RECORD"] = record
                    env["WRITE_MODE"] = write_mode
                    result = subprocess.run(
                        ["bash", str(mount_harness)],
                        env=env,
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode == 0, expected_success)

            unmounted_harness = root / "validate-unmounted.sh"
            self.write_executable(
                unmounted_harness,
                f"""
                #!/bin/bash
                set -eo pipefail
                {unmounted_helper}
                sd_uboot_device_is_unmounted /dev/mmcblk1p1
                """,
            )
            for name, status, record, expected_success in (
                ("unmounted", "1", "", True),
                ("mounted", "0", "/mnt/sd", False),
                ("inspection-error", "2", "", False),
            ):
                with self.subTest(device=name):
                    env.update(
                        {"FINDMNT_EXIT": status, "FINDMNT_RECORD": record}
                    )
                    result = subprocess.run(
                        ["bash", str(unmounted_harness)],
                        env=env,
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode == 0, expected_success)

    def test_safe_sd_uboot_replace_verifies_and_recovers_failures(self) -> None:
        source = FIRMWARE_SCRIPT.read_text(encoding="utf-8")
        sync_helper = self.function_source(source, "sync_sd_uboot_filesystem")
        helper = self.function_source(source, "safe_replace_sd_uboot")
        real_cmp = shutil.which("cmp")
        real_mv = shutil.which("mv")
        real_sync = shutil.which("sync")
        self.assertIsNotNone(real_cmp)
        self.assertIsNotNone(real_mv)
        self.assertIsNotNone(real_sync)

        def run_replacement(
            mode: str,
            *,
            target: str = "regular",
            source_kind: str = "regular",
        ) -> tuple[
            subprocess.CompletedProcess[str], bytes, bytes, dict[str, bytes], str
        ]:
            with repository_temporary_directory() as directory:
                root = Path(directory)
                mount_dir = root / "mount"
                bin_dir = root / "bin"
                source_file = root / "u-boot_signed.bin"
                victim = root / "victim.bin"
                mount_dir.mkdir()
                bin_dir.mkdir()
                victim.write_bytes(b"victim-data\x00")
                if source_kind == "symlink":
                    source_target = root / "source-target.bin"
                    source_target.write_bytes(b"new-uboot\x00\x01")
                    source_file.symlink_to(source_target)
                else:
                    source_file.write_bytes(b"new-uboot\x00\x01")

                target_file = mount_dir / "uboot.bin"
                if target == "regular":
                    target_file.write_bytes(b"old-uboot\x02\x03")
                elif target == "symlink":
                    target_file.symlink_to(victim)

                self.write_executable(
                    bin_dir / "cmp",
                    """
                    #!/bin/bash
                    set -u
                    left="${@: -2:1}"
                    right="${@: -1}"
                    if [ "$TEST_MODE" = "candidate-verify-fails" ] &&
                       [[ "$right" == .nkv-uboot-candidate.* ]]; then
                        exit 1
                    fi
                    if { [ "$TEST_MODE" = "post-verify-fails" ] ||
                         [ "$TEST_MODE" = "restore-fails" ] ||
                         [ "$TEST_MODE" = "restore-sync-fails" ]; } &&
                       [ "$left" = "$SOURCE_FILE" ] && [ "$right" = "uboot.bin" ] &&
                       "$REAL_CMP" "$@"; then
                        exit 1
                    fi
                    exec "$REAL_CMP" "$@"
                    """,
                )
                self.write_executable(
                    bin_dir / "mv",
                    """
                    #!/bin/bash
                    set -u
                    source_path="${@: -2:1}"
                    destination="${@: -1}"
                    if [ "$destination" = "uboot.bin" ]; then
                        if [ "$TEST_MODE" = "rename-fails-before" ]; then
                            exit 5
                        fi
                        if [ "$TEST_MODE" = "rename-fails-after" ]; then
                            "$REAL_MV" "$@"
                            exit 5
                        fi
                        if [ "$TEST_MODE" = "restore-fails" ] &&
                           [[ "$source_path" == .nkv-uboot-restore.* ]]; then
                            exit 5
                        fi
                    fi
                    exec "$REAL_MV" "$@"
                    """,
                )
                self.write_executable(
                    bin_dir / "sync",
                    """
                    #!/bin/bash
                    set -u
                    if [ "$#" -ne 3 ] || [ "$1" != "-f" ] ||
                       [ "$2" != "--" ] || [ "$3" != "$MOUNT_DIR" ]; then
                        exit 97
                    fi
                    count=0
                    if [ -f "$SYNC_STATE" ]; then
                        read -r count <"$SYNC_STATE"
                    fi
                    count=$((count + 1))
                    printf '%s\n' "$count" >"$SYNC_STATE"
                    if [ "$TEST_MODE" = "candidate-sync-fails" ] && [ "$count" -eq 1 ]; then
                        exit 5
                    fi
                    if [ "$TEST_MODE" = "commit-sync-fails" ] && [ "$count" -eq 2 ]; then
                        exit 5
                    fi
                    if [ "$TEST_MODE" = "restore-sync-fails" ] && [ "$count" -eq 4 ]; then
                        exit 5
                    fi
                    exec "$REAL_SYNC" "$@"
                    """,
                )
                harness = root / "replace.sh"
                self.write_executable(
                    harness,
                    f"""
                    #!/bin/bash
                    set -eo pipefail
                    validate_sd_uboot_mount() {{ return 0; }}
                    enable_sd_uboot_sync_writes() {{
                        [ "$TEST_MODE" != "remount-sync-fails" ]
                    }}
                    {sync_helper}
                    {helper}
                    safe_replace_sd_uboot "$SOURCE_FILE" "$MOUNT_DIR" /dev/mmcblk1p1
                    """,
                )
                env = os.environ.copy()
                env.update(
                    {
                        "PATH": f"{bin_dir}:{env['PATH']}",
                        "MOUNT_DIR": str(mount_dir),
                        "SOURCE_FILE": str(source_file),
                        "TEST_MODE": mode,
                        "REAL_CMP": str(real_cmp),
                        "REAL_MV": str(real_mv),
                        "REAL_SYNC": str(real_sync),
                        "SYNC_STATE": str(root / "sync.state"),
                    }
                )
                result = subprocess.run(
                    ["bash", str(harness)],
                    env=env,
                    text=True,
                    capture_output=True,
                    check=False,
                )
                target_data = target_file.read_bytes() if target_file.is_file() else b""
                victim_data = victim.read_bytes()
                leftovers = {
                    path.name: path.read_bytes()
                    for path in sorted(mount_dir.glob(".nkv-uboot-*"))
                }
                if target_file.is_symlink():
                    state = f"symlink:{os.readlink(target_file)}"
                elif target_file.exists():
                    state = "regular"
                else:
                    state = "missing"
                return result, target_data, victim_data, leftovers, state

        success, data, victim, leftovers, state = run_replacement("success")
        self.assertEqual(success.returncode, 0, success.stdout + success.stderr)
        self.assertEqual(data, b"new-uboot\x00\x01")
        self.assertEqual(victim, b"victim-data\x00")
        self.assertEqual(leftovers, {})
        self.assertEqual(state, "regular")

        rename_after, data, _, leftovers, _ = run_replacement(
            "rename-fails-after"
        )
        self.assertEqual(
            rename_after.returncode, 0, rename_after.stdout + rename_after.stderr
        )
        self.assertEqual(data, b"new-uboot\x00\x01")
        self.assertEqual(leftovers, {})

        for mode in (
            "candidate-verify-fails",
            "candidate-sync-fails",
            "commit-sync-fails",
            "remount-sync-fails",
            "rename-fails-before",
            "post-verify-fails",
        ):
            with self.subTest(mode=mode):
                result, data, _, leftovers, state = run_replacement(mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(data, b"old-uboot\x02\x03")
                self.assertEqual(leftovers, {})
                self.assertEqual(state, "regular")

        restore_failed, data, _, leftovers, _ = run_replacement("restore-fails")
        self.assertNotEqual(restore_failed.returncode, 0)
        self.assertEqual(data, b"new-uboot\x00\x01")
        self.assertEqual(list(leftovers.values()), [b"old-uboot\x02\x03"])
        self.assertIn("backup retained", restore_failed.stdout)

        sync_failed, data, _, leftovers, _ = run_replacement(
            "restore-sync-fails"
        )
        self.assertNotEqual(sync_failed.returncode, 0)
        self.assertEqual(data, b"old-uboot\x02\x03")
        self.assertEqual(list(leftovers.values()), [b"old-uboot\x02\x03"])
        self.assertIn("could not be persisted", sync_failed.stdout)

        for target in ("missing", "symlink"):
            with self.subTest(target=target):
                result, _, victim, leftovers, state = run_replacement(
                    "success", target=target
                )
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(victim, b"victim-data\x00")
                self.assertEqual(leftovers, {})
                if target == "missing":
                    self.assertEqual(state, "missing")
                else:
                    self.assertTrue(state.startswith("symlink:"))

        result, data, _, leftovers, state = run_replacement(
            "success", source_kind="symlink"
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(data, b"old-uboot\x02\x03")
        self.assertEqual(leftovers, {})
        self.assertEqual(state, "regular")

    def test_private_sd_uboot_worker_rejects_direct_invocation(self) -> None:
        with repository_temporary_directory() as directory:
            source_file = Path(directory) / "u-boot_signed.bin"
            source_file.write_bytes(b"signed-uboot")
            result = subprocess.run(
                [
                    "bash",
                    str(FIRMWARE_SCRIPT),
                    "__sd_uboot_update",
                    str(source_file),
                    "/run/.nanokvm-sd-boot.direct-test",
                ],
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(
                "worker is not in a private mount namespace", result.stdout
            )

    def test_cdn_download_keeps_metadata_and_archive_on_the_same_base(self) -> None:
        result, logs = self.run_firmware_command(
            ["fetch", "v1.0.1"], curl_mode="stable-archive-fails"
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotIn(" -k", f" {logs['curl']}")

        curl_calls = logs["curl"].splitlines()
        stable_metadata = next(
            index
            for index, call in enumerate(curl_calls)
            if "/firmware/firmware_1.0.1.json" in call
            and "/preview/" not in call
        )
        stable_archive = next(
            index
            for index, call in enumerate(curl_calls)
            if "/firmware/axera_firmware_v1.0.1.tar.gz" in call
            and "/preview/" not in call
        )
        preview_metadata = next(
            index
            for index, call in enumerate(curl_calls)
            if "/preview/firmware/firmware_1.0.1.json" in call
        )
        preview_archive = next(
            index
            for index, call in enumerate(curl_calls)
            if "/preview/firmware/axera_firmware_v1.0.1.tar.gz" in call
        )
        self.assertLess(stable_metadata, stable_archive)
        self.assertLess(stable_archive, preview_metadata)
        self.assertLess(preview_metadata, preview_archive)

        verifier_calls = logs["verifier"].splitlines()
        self.assertGreaterEqual(len(verifier_calls), 2)
        for call in verifier_calls:
            self.assertTrue(call.startswith("preview|preview|firmware "), call)
            self.assertIn("--requested-version 1.0.1", call)

    def test_firmware_verification_failure_never_extracts_or_publishes(self) -> None:
        result, logs = self.run_firmware_command(
            ["fetch", "v1.0.1"], verifier_exit="25"
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(logs["verifier"].splitlines()), 2)
        self.assertEqual(logs["tar"], "")
        self.assertFalse(logs["cache_target_exists"])

    def test_local_archive_only_is_rejected_and_pair_calls_firmware_verifier(self) -> None:
        archive_only, archive_logs = self.run_firmware_command(
            ["update", "{archive}"]
        )
        self.assertNotEqual(archive_only.returncode, 0)
        self.assertEqual(archive_logs["verifier"], "")
        self.assertEqual(archive_logs["tar"], "")

        pair_result, pair_logs = self.run_firmware_command(
            ["update", "{archive}", "{metadata}"], verifier_exit="25"
        )
        self.assertNotEqual(pair_result.returncode, 0)
        verifier_calls = pair_logs["verifier"].splitlines()
        self.assertEqual(len(verifier_calls), 1)
        self.assertIn("|firmware --metadata ", verifier_calls[0])
        self.assertIn("--artifact ", verifier_calls[0])
        self.assertIn("--requested-version 1.0.1", verifier_calls[0])
        self.assertEqual(pair_logs["tar"], "")

    def test_manual_firmware_same_version_policy_is_explicit_and_propagated(self) -> None:
        strict_result, strict_logs = self.run_firmware_command(
            ["update", "v1.0.1"],
            cache_state="pair",
            curl_mode="fail",
            current_version="NanoKVM firmware v1.0.1",
            extra_env={"FIRMWARE_ALLOW_SAME_VERSION": "true"},
        )
        self.assertNotEqual(strict_result.returncode, 0)
        strict_verifier_calls = strict_logs["verifier"].splitlines()
        self.assertEqual(len(strict_verifier_calls), 1)
        self.assertNotIn("--allow-same-version", strict_verifier_calls[0])
        self.assertEqual(strict_logs["tar"], "")

        local_result, local_logs = self.run_firmware_command(
            ["update", "{archive}", "{metadata}"],
            current_version="NanoKVM firmware v1.0.1",
        )
        # The fixture cannot flash the host, so the command stops only after
        # both the staged pair and published cache have been verified.
        self.assertNotEqual(local_result.returncode, 0)
        local_verifier_calls = local_logs["verifier"].splitlines()
        self.assertEqual(len(local_verifier_calls), 2)
        for call in local_verifier_calls:
            self.assertIn("--current-version 1.0.1", call)
            self.assertIn("--allow-same-version", call)
        self.assertNotEqual(local_logs["tar"], "")

        cdn_result, cdn_logs = self.run_firmware_command(
            ["update", "--allow-same-version", "v1.0.1"],
            current_version="NanoKVM firmware v1.0.1",
        )
        self.assertNotEqual(cdn_result.returncode, 0)
        cdn_verifier_calls = cdn_logs["verifier"].splitlines()
        self.assertEqual(len(cdn_verifier_calls), 2)
        for call in cdn_verifier_calls:
            self.assertIn("--current-version 1.0.1", call)
            self.assertIn("--allow-same-version", call)
        self.assertNotEqual(cdn_logs["tar"], "")

        cached_result, cached_logs = self.run_firmware_command(
            ["update", "--allow-same-version", "v1.0.1"],
            cache_state="pair",
            current_version="NanoKVM firmware v1.0.1",
        )
        self.assertNotEqual(cached_result.returncode, 0)
        cached_verifier_calls = cached_logs["verifier"].splitlines()
        self.assertEqual(len(cached_verifier_calls), 1)
        self.assertIn("--allow-same-version", cached_verifier_calls[0])
        self.assertEqual(cached_logs["curl"], "")

    def test_archive_only_cache_is_rejected_and_valid_cache_is_reverified(self) -> None:
        invalid_result, invalid_logs = self.run_firmware_command(
            ["fetch", "v1.0.1"], cache_state="archive-only", curl_mode="fail"
        )
        self.assertNotEqual(invalid_result.returncode, 0)
        self.assertEqual(invalid_logs["verifier"], "")
        self.assertEqual(invalid_logs["tar"], "")

        valid_result, valid_logs = self.run_firmware_command(
            ["fetch", "v1.0.1"], cache_state="pair", repetitions=2
        )
        self.assertEqual(valid_result.returncode, 0, valid_result.stderr)
        verifier_calls = valid_logs["verifier"].splitlines()
        self.assertEqual(len(verifier_calls), 2)
        for call in verifier_calls:
            self.assertTrue(call.startswith("cache|cache|firmware "), call)
        self.assertEqual(valid_logs["curl"], "")

    def test_kvmcomm_delegates_cache_selection_using_only_required_version(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        check_source = self.function_source(source, "check_firmware_update")
        self.assertNotIn(".tar.xz", source)
        self.assertNotIn("firmware_file", check_source)
        self.assertNotIn("cache_dir", check_source)
        self.assertEqual(check_source.count("firmware_update.sh"), 1)
        self.assertIn(
            'firmware_update.sh" update "$REQUIRED_FIRMWARE_VERSION"',
            check_source,
        )
        self.assertNotIn("--allow-same-version", check_source)

    def test_firmware_worker_holds_shared_device_maintenance_lock(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        self.assertIn(
            'DEVICE_MAINTENANCE_LOCK_FILE="/run/lock/nanokvm-device-maintenance.lock"',
            source,
        )
        self.assertIn('APPLICATION_UPDATE_UNIT="nanokvm-self-update.service"', source)
        self.assertIn("FIRMWARE_MAINTENANCE_WAIT_SECONDS=300", source)

        worker = self.function_source(source, "run_firmware_update_worker")
        dedicated_lock = worker.index('flock -n "$FIRMWARE_UPDATE_LOCK_FD"')
        first_unit_check = worker.index("application_update_blocks_maintenance")
        shared_lock = worker.index('flock -n "$DEVICE_MAINTENANCE_LOCK_FD"')
        second_unit_check = worker.index(
            "application_update_blocks_maintenance", first_unit_check + 1
        )
        update = worker.index("check_firmware_update")
        self.assertLess(dedicated_lock, first_unit_check)
        self.assertLess(first_unit_check, shared_lock)
        self.assertLess(shared_lock, second_unit_check)
        self.assertLess(second_unit_check, update)
        self.assertLess(update, worker.rindex("release_device_maintenance_lock"))

    def test_factory_reset_holds_shared_device_maintenance_lock(self) -> None:
        source = RECOVERY_SCRIPT.read_text(encoding="utf-8")
        self.assertIn(
            'DEVICE_MAINTENANCE_LOCK_FILE="/run/lock/nanokvm-device-maintenance.lock"',
            source,
        )
        self.assertIn('APPLICATION_UPDATE_UNIT="nanokvm-self-update.service"', source)

        acquire = self.function_source(source, "acquire_reset_lock")
        dedicated_lock = acquire.index('flock -n "$LOCK_FD"')
        first_unit_check = acquire.index("application_update_blocks_reset")
        shared_lock = acquire.index('flock -n "$DEVICE_MAINTENANCE_LOCK_FD"')
        second_unit_check = acquire.index(
            "application_update_blocks_reset", first_unit_check + 1
        )
        marker = acquire.index("RESET_MARKER_OWNED=1")
        self.assertLess(dedicated_lock, first_unit_check)
        self.assertLess(first_unit_check, shared_lock)
        self.assertLess(shared_lock, second_unit_check)
        self.assertLess(second_unit_check, marker)

        release = self.function_source(source, "release_reset_locks")
        self.assertIn('flock -u "$DEVICE_MAINTENANCE_LOCK_FD"', release)
        self.assertIn("release_factory_reset_lock", release)
        dedicated_release = self.function_source(
            source, "release_factory_reset_lock"
        )
        self.assertIn('flock -u "$LOCK_FD"', dedicated_release)
        cleanup = self.function_source(source, "cleanup_on_exit")
        self.assertIn("release_reset_locks", cleanup)

    def test_firmware_worker_waits_for_application_update_then_runs_once(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        helpers = "\n".join(
            self.function_source(source, name)
            for name in (
                "release_firmware_update_lock",
                "release_device_maintenance_lock",
                "wait_for_firmware_maintenance_retry",
                "run_firmware_update_worker",
            )
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            check_marker = root / "firmware-check-called"
            log_dir = root / "log"
            log_file = log_dir / "kvmcomm.log"
            harness = root / "firmware-app-wait-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eu
                application_checks=0
                factory_reset_in_progress() {{ return 1; }}
                application_update_blocks_maintenance() {{
                    application_checks=$((application_checks + 1))
                    [ "$application_checks" -eq 1 ]
                }}
                check_firmware_update() {{ printf 'called\n' >>"$CHECK_MARKER"; }}
                sleep() {{ SECONDS=$((SECONDS + $1)); }}
                {helpers}
                run_firmware_update_worker
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "FIRMWARE_UPDATE_LOCK_FILE": str(root / "firmware.lock"),
                    "FIRMWARE_UPDATE_LOCK_FD": "198",
                    "DEVICE_MAINTENANCE_LOCK_FILE": str(root / "device-maintenance.lock"),
                    "DEVICE_MAINTENANCE_LOCK_FD": "197",
                    "FIRMWARE_MAINTENANCE_WAIT_SECONDS": "300",
                    "FIRMWARE_MAINTENANCE_POLL_SECONDS": "2",
                    "LOG_DIR": str(log_dir),
                    "LOG_FILE": str(log_file),
                    "CHECK_MARKER": str(check_marker),
                }
            )

            result = subprocess.run(
                ["bash", str(harness)],
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(check_marker.read_text(encoding="utf-8"), "called\n")
            self.assertIn(
                "Waiting up to 300 seconds",
                log_file.read_text(encoding="utf-8"),
            )

    def test_firmware_worker_enforces_five_minute_shared_lock_timeout(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        helpers = "\n".join(
            self.function_source(source, name)
            for name in (
                "release_firmware_update_lock",
                "release_device_maintenance_lock",
                "wait_for_firmware_maintenance_retry",
                "run_firmware_update_worker",
            )
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            shared_lock = root / "device-maintenance.lock"
            dedicated_lock = root / "firmware.lock"
            check_marker = root / "firmware-check-called"
            log_dir = root / "log"
            log_file = log_dir / "kvmcomm.log"
            harness = root / "firmware-lock-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eu
                factory_reset_in_progress() {{ return 1; }}
                application_update_blocks_maintenance() {{ return 1; }}
                check_firmware_update() {{ : >"$CHECK_MARKER"; }}
                sleep() {{ SECONDS=$((SECONDS + $1)); }}
                {helpers}
                run_firmware_update_worker
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "FIRMWARE_UPDATE_LOCK_FILE": str(dedicated_lock),
                    "FIRMWARE_UPDATE_LOCK_FD": "198",
                    "DEVICE_MAINTENANCE_LOCK_FILE": str(shared_lock),
                    "DEVICE_MAINTENANCE_LOCK_FD": "197",
                    "FIRMWARE_MAINTENANCE_WAIT_SECONDS": "300",
                    "FIRMWARE_MAINTENANCE_POLL_SECONDS": "2",
                    "LOG_DIR": str(log_dir),
                    "LOG_FILE": str(log_file),
                    "CHECK_MARKER": str(check_marker),
                }
            )

            with shared_lock.open("w", encoding="utf-8") as lock_handle:
                fcntl.flock(lock_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                result = subprocess.run(
                    ["bash", str(harness)],
                    env=env,
                    text=True,
                    capture_output=True,
                    check=False,
                )

            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(check_marker.exists())
            self.assertIn(
                "Timed out after 300 seconds",
                log_file.read_text(encoding="utf-8"),
            )

    def test_firmware_worker_does_not_start_after_deadline_becomes_available(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        helpers = "\n".join(
            self.function_source(source, name)
            for name in (
                "release_firmware_update_lock",
                "release_device_maintenance_lock",
                "wait_for_firmware_maintenance_retry",
                "run_firmware_update_worker",
            )
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            check_marker = root / "firmware-check-called"
            log_dir = root / "log"
            log_file = log_dir / "kvmcomm.log"
            harness = root / "firmware-deadline-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eu
                application_checks=0
                factory_reset_in_progress() {{ return 1; }}
                application_update_blocks_maintenance() {{
                    application_checks=$((application_checks + 1))
                    [ "$application_checks" -eq 1 ]
                }}
                check_firmware_update() {{ : >"$CHECK_MARKER"; }}
                sleep() {{ SECONDS=$((SECONDS + 301)); }}
                {helpers}
                run_firmware_update_worker
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "FIRMWARE_UPDATE_LOCK_FILE": str(root / "firmware.lock"),
                    "FIRMWARE_UPDATE_LOCK_FD": "198",
                    "DEVICE_MAINTENANCE_LOCK_FILE": str(root / "device-maintenance.lock"),
                    "DEVICE_MAINTENANCE_LOCK_FD": "197",
                    "FIRMWARE_MAINTENANCE_WAIT_SECONDS": "300",
                    "FIRMWARE_MAINTENANCE_POLL_SECONDS": "2",
                    "LOG_DIR": str(log_dir),
                    "LOG_FILE": str(log_file),
                    "CHECK_MARKER": str(check_marker),
                }
            )

            result = subprocess.run(
                ["bash", str(harness)],
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(check_marker.exists())
            self.assertIn(
                "Timed out after 300 seconds",
                log_file.read_text(encoding="utf-8"),
            )

    def test_firmware_worker_yields_to_factory_reset_after_shared_lock(self) -> None:
        source = KVMCOMM_SCRIPT.read_text(encoding="utf-8")
        helpers = "\n".join(
            self.function_source(source, name)
            for name in (
                "release_firmware_update_lock",
                "release_device_maintenance_lock",
                "wait_for_firmware_maintenance_retry",
                "run_firmware_update_worker",
            )
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            check_marker = root / "firmware-check-called"
            log_dir = root / "log"
            log_file = log_dir / "kvmcomm.log"
            harness = root / "firmware-reset-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -eu
                reset_checks=0
                factory_reset_in_progress() {{
                    reset_checks=$((reset_checks + 1))
                    [ "$reset_checks" -ge 2 ]
                }}
                application_update_blocks_maintenance() {{ return 1; }}
                check_firmware_update() {{ : >"$CHECK_MARKER"; }}
                {helpers}
                run_firmware_update_worker
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "FIRMWARE_UPDATE_LOCK_FILE": str(root / "firmware.lock"),
                    "FIRMWARE_UPDATE_LOCK_FD": "198",
                    "DEVICE_MAINTENANCE_LOCK_FILE": str(root / "device-maintenance.lock"),
                    "DEVICE_MAINTENANCE_LOCK_FD": "197",
                    "FIRMWARE_MAINTENANCE_WAIT_SECONDS": "300",
                    "FIRMWARE_MAINTENANCE_POLL_SECONDS": "2",
                    "LOG_DIR": str(log_dir),
                    "LOG_FILE": str(log_file),
                    "CHECK_MARKER": str(check_marker),
                }
            )

            result = subprocess.run(
                ["bash", str(harness)],
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(check_marker.exists())
            self.assertIn(
                "Factory reset started while waiting",
                log_file.read_text(encoding="utf-8"),
            )

    def test_factory_reset_rejects_busy_shared_maintenance_lock(self) -> None:
        source = RECOVERY_SCRIPT.read_text(encoding="utf-8")
        acquire = self.function_source(source, "acquire_reset_lock")
        with repository_temporary_directory() as directory:
            root = Path(directory)
            shared_lock = root / "device-maintenance.lock"
            dedicated_lock = root / "reset.lock"
            state_dir = root / "state"
            marker = state_dir / "reset_in_progress"
            harness = root / "reset-lock-harness.sh"
            self.write_executable(
                harness,
                f"""
                #!/bin/bash
                set -u
                application_update_blocks_reset() {{ return 1; }}
                {acquire}
                acquire_reset_lock
                """,
            )
            env = os.environ.copy()
            env.update(
                {
                    "LOCK_FILE": str(dedicated_lock),
                    "LOCK_FD": "9",
                    "DEVICE_MAINTENANCE_LOCK_FILE": str(shared_lock),
                    "DEVICE_MAINTENANCE_LOCK_FD": "8",
                    "RESET_STATE_DIR": str(state_dir),
                    "RESET_MARKER_FILE": str(marker),
                    "LEGACY_RESET_MARKER_FILE": str(root / "legacy-reset-marker"),
                    "LOG_FILE": str(root / "reset.log"),
                    "RESET_LOCK_ACQUIRED": "0",
                    "MAINTENANCE_LOCK_ACQUIRED": "0",
                    "RESET_MARKER_OWNED": "0",
                }
            )

            with shared_lock.open("w", encoding="utf-8") as lock_handle:
                fcntl.flock(lock_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                result = subprocess.run(
                    ["bash", str(harness)],
                    env=env,
                    text=True,
                    capture_output=True,
                    check=False,
                )

            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(marker.exists())
            self.assertIn("Another device maintenance operation", result.stdout)

    def run_firmware_command(
        self,
        arguments: list[str],
        *,
        verifier_exit: str = "0",
        verifier_update_uboot: str = "false",
        verifier_output_mode: str = "canonical",
        curl_mode: str = "success",
        cache_state: str = "none",
        repetitions: int = 1,
        current_version: str | None = None,
        extra_env: dict[str, str] | None = None,
    ) -> tuple[subprocess.CompletedProcess[str], dict[str, str | bool]]:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            bin_dir = root / "bin"
            cache_dir = root / "cache"
            local_dir = root / "local"
            bin_dir.mkdir()
            cache_dir.mkdir()
            local_dir.mkdir()

            metadata = local_dir / "firmware_1.0.1.json"
            archive = local_dir / "axera_firmware_v1.0.1.tar.gz"
            metadata.write_text("local\n", encoding="utf-8")
            archive.write_text("local\n", encoding="utf-8")

            cached_pair = cache_dir / "firmware_1.0.1"
            if cache_state != "none":
                cached_pair.mkdir()
                (cached_pair / archive.name).write_text("cache\n", encoding="utf-8")
                if cache_state == "pair":
                    (cached_pair / metadata.name).write_text(
                        "cache\n", encoding="utf-8"
                    )

            current_version_file = root / "boot-ver"
            if current_version is not None:
                current_version_file.write_text(
                    f"{current_version}\n", encoding="utf-8"
                )

            self.write_executable(
                bin_dir / "curl",
                """
                #!/bin/bash
                set -eu
                output=""
                url=""
                while [ "$#" -gt 0 ]; do
                    case "$1" in
                        -o) output="$2"; shift 2 ;;
                        --connect-timeout|--max-time|--max-filesize) shift 2 ;;
                        -*) shift ;;
                        *) url="$1"; shift ;;
                    esac
                done
                printf '%s\n' "$url" >>"$CURL_LOG"
                if [ "$CURL_MODE" = "fail" ]; then
                    exit 22
                fi
                origin=stable
                case "$url" in
                    */preview/*) origin=preview ;;
                esac
                if [ "$CURL_MODE" = "stable-archive-fails" ] &&
                   [ "$origin" = "stable" ]; then
                    case "$url" in
                        *.tar.gz) exit 22 ;;
                    esac
                fi
                case "$url" in
                    *.json|*.tar.gz) printf '%s\n' "$origin" >"$output" ;;
                    *) exit 22 ;;
                esac
                """,
            )
            self.write_executable(
                bin_dir / "nanokvm-update-verify",
                """
                #!/bin/bash
                set -eu
                original="$*"
                metadata=""
                artifact=""
                requested=""
                current=""
                allow_same=false
                while [ "$#" -gt 0 ]; do
                    case "$1" in
                        --metadata) metadata="$2"; shift 2 ;;
                        --artifact) artifact="$2"; shift 2 ;;
                        --requested-version) requested="$2"; shift 2 ;;
                        --current-version) current="$2"; shift 2 ;;
                        --allow-same-version) allow_same=true; shift ;;
                        --keyring) shift 2 ;;
                        *) shift ;;
                    esac
                done
                metadata_origin=$(head -n1 "$metadata")
                artifact_origin=$(head -n1 "$artifact")
                printf '%s|%s|%s\n' \
                    "$metadata_origin" "$artifact_origin" "$original" \
                    >>"$VERIFIER_LOG"
                [ "$metadata_origin" = "$artifact_origin" ] || exit 25
                [ "$VERIFIER_EXIT" = "0" ] || exit "$VERIFIER_EXIT"
                if [ -n "$current" ] && [ "$current" = "$requested" ] && \
                   [ "$allow_same" != "true" ]; then
                    exit 30
                fi
                case "$VERIFIER_OUTPUT_MODE" in
                    canonical)
                        printf '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":%s}\n' \
                            "$requested" "$requested" "$VERIFIER_UPDATE_UBOOT"
                        ;;
                    missing)
                        printf '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz"}\n' \
                            "$requested" "$requested"
                        ;;
                    string)
                        printf '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":"false"}\n' \
                            "$requested" "$requested"
                        ;;
                    extra)
                        printf '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":false,"extra":true}\n' \
                            "$requested" "$requested"
                        ;;
                    multiple)
                        printf '{"mode":"firmware","version":"%s","name":"axera_firmware_v%s.tar.gz","updateUboot":false}\nextra\n' \
                            "$requested" "$requested"
                        ;;
                    *) exit 91 ;;
                esac
                """,
            )
            self.write_executable(
                bin_dir / "tar",
                """
                #!/bin/bash
                set -eu
                printf '%s\n' "$*" >>"$TAR_LOG"
                destination=""
                while [ "$#" -gt 0 ]; do
                    case "$1" in
                        -C) destination="$2"; shift 2 ;;
                        *) shift ;;
                    esac
                done
                [ -n "$destination" ]
                install -d -m 0700 "$destination/overlay/boot"
                printf 'fixture\n' >"$destination/b2sum.txt"
                printf 'v1.0.1\n' >"$destination/overlay/boot/ver"
                """,
            )
            self.write_executable(
                bin_dir / "b2sum",
                """
                #!/bin/bash
                printf '%s\n' "$*" >>"$B2SUM_LOG"
                exit 0
                """,
            )
            self.write_executable(bin_dir / "sleep", "#!/bin/bash\nexit 0\n")

            curl_log = root / "curl.log"
            verifier_log = root / "verifier.log"
            tar_log = root / "tar.log"
            b2sum_log = root / "b2sum.log"
            env = os.environ.copy()
            env.update(
                {
                    "PATH": f"{bin_dir}:{env['PATH']}",
                    "CURL_LOG": str(curl_log),
                    "VERIFIER_LOG": str(verifier_log),
                    "TAR_LOG": str(tar_log),
                    "B2SUM_LOG": str(b2sum_log),
                    "CURL_MODE": curl_mode,
                    "VERIFIER_EXIT": verifier_exit,
                    "VERIFIER_UPDATE_UBOOT": verifier_update_uboot,
                    "VERIFIER_OUTPUT_MODE": verifier_output_mode,
                    "FIRMWARE_CACHE_DIR": str(cache_dir),
                    "FIRMWARE_NETWORK_WAIT_SECONDS": "0",
                    "FIRMWARE_CURRENT_VERSION_FILE": str(current_version_file),
                    "NANOKVM_UPDATE_VERIFIER": str(bin_dir / "nanokvm-update-verify"),
                    "NANOKVM_UPDATE_KEYRING": str(root / "keys.json"),
                }
            )
            if extra_env is not None:
                env.update(extra_env)
            resolved_arguments = [
                str(archive)
                if argument == "{archive}"
                else str(metadata)
                if argument == "{metadata}"
                else argument
                for argument in arguments
            ]
            result: subprocess.CompletedProcess[str] | None = None
            combined_stdout: list[str] = []
            combined_stderr: list[str] = []
            for _ in range(repetitions):
                result = subprocess.run(
                    ["bash", str(FIRMWARE_SCRIPT), *resolved_arguments],
                    env=env,
                    text=True,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    check=False,
                )
                combined_stdout.append(result.stdout)
                combined_stderr.append(result.stderr)
                if result.returncode != 0:
                    break
            assert result is not None
            result = subprocess.CompletedProcess(
                result.args,
                result.returncode,
                "".join(combined_stdout),
                "".join(combined_stderr),
            )
            return result, {
                "curl": curl_log.read_text(encoding="utf-8")
                if curl_log.exists()
                else "",
                "verifier": verifier_log.read_text(encoding="utf-8")
                if verifier_log.exists()
                else "",
                "tar": tar_log.read_text(encoding="utf-8")
                if tar_log.exists()
                else "",
                "b2sum": b2sum_log.read_text(encoding="utf-8")
                if b2sum_log.exists()
                else "",
                "cache_target_exists": cached_pair.exists(),
            }

    @staticmethod
    def write_executable(path: Path, content: str) -> None:
        path.write_text(textwrap.dedent(content).lstrip(), encoding="utf-8")
        path.chmod(0o755)

    @staticmethod
    def function_source(source: str, function_name: str) -> str:
        start_marker = f"{function_name}() {{"
        start = source.find(start_marker)
        if start < 0:
            raise AssertionError(f"missing shell helper: {function_name}")
        end = source.find("\n}\n", start)
        if end < 0:
            raise AssertionError(f"unterminated shell helper: {function_name}")
        return source[start : end + 3]


if __name__ == "__main__":
    unittest.main()
