#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPOSITORY_ROOT))

import build


EXPECTED_APPS_COMMIT = "1" * 40


class AppsPackagingTests(unittest.TestCase):
    def git_results(self, *, actual: str = EXPECTED_APPS_COMMIT, dirty: str = ""):
        return (
            subprocess.CompletedProcess([], 0, stdout=f"{EXPECTED_APPS_COMMIT}\n"),
            subprocess.CompletedProcess([], 0, stdout=f"{build.APPS_DIR}\n"),
            subprocess.CompletedProcess([], 0, stdout=f"{actual}\n"),
            subprocess.CompletedProcess([], 0, stdout=dirty),
        )

    def test_clean_matching_checkout_passes(self) -> None:
        with (
            mock.patch.object(build, "require_dir"),
            mock.patch.object(build, "run", side_effect=self.git_results()) as run_mock,
        ):
            build.verify_release_apps_checkout("release")

        self.assertEqual(run_mock.call_count, 4)
        self.assertEqual(
            run_mock.call_args_list[0].args[1],
            ("git", "rev-parse", f"HEAD:{build.APPS_GITLINK_PATH.as_posix()}"),
        )
        self.assertEqual(
            run_mock.call_args_list[1].args[1],
            ("git", "rev-parse", "--show-toplevel"),
        )
        self.assertEqual(run_mock.call_args_list[1].kwargs["cwd"], build.APPS_DIR)

    def test_uninitialized_checkout_is_rejected(self) -> None:
        results = list(self.git_results())
        results[1] = subprocess.CompletedProcess([], 0, stdout=f"{build.ROOT}\n")
        with (
            mock.patch.object(build, "require_dir"),
            mock.patch.object(build, "run", side_effect=results),
            self.assertRaisesRegex(SystemExit, "not initialized"),
        ):
            build.verify_release_apps_checkout("release")

    def test_mismatched_checkout_is_rejected(self) -> None:
        with (
            mock.patch.object(build, "require_dir"),
            mock.patch.object(build, "run", side_effect=self.git_results(actual="2" * 40)),
            self.assertRaisesRegex(SystemExit, "does not match"),
        ):
            build.verify_release_apps_checkout("release")

    def test_dirty_checkout_is_rejected(self) -> None:
        with (
            mock.patch.object(build, "require_dir"),
            mock.patch.object(build, "run", side_effect=self.git_results(dirty="?? local.py\n")),
            self.assertRaisesRegex(SystemExit, "dirty"),
        ):
            build.verify_release_apps_checkout("release")

    def test_missing_checkout_is_rejected_before_git(self) -> None:
        with (
            mock.patch.object(build, "require_dir", side_effect=SystemExit("missing")),
            mock.patch.object(build, "run") as run_mock,
            self.assertRaisesRegex(SystemExit, "missing"),
        ):
            build.verify_release_apps_checkout("release")
        run_mock.assert_not_called()

    def test_release_checks_apps_before_component_builds(self) -> None:
        args = argparse.Namespace(
            version="0.0.26",
            docker=False,
            signing_key=Path("signing.pem"),
            trusted_public_key=[],
        )
        with (
            mock.patch.object(build, "validate_update_version", return_value="0.0.26"),
            mock.patch.object(build, "verify_release_apps_checkout", side_effect=SystemExit("dirty")),
            mock.patch.object(build, "build_all") as build_mock,
            self.assertRaisesRegex(SystemExit, "dirty"),
        ):
            build.release(args)
        build_mock.assert_not_called()

    def test_versioned_kvmcomm_packaging_checks_apps_checkout(self) -> None:
        args = argparse.Namespace(version="0.0.25", docker=False)
        with (
            mock.patch.object(build, "verify_release_apps_checkout", side_effect=SystemExit("dirty")),
            mock.patch.object(build, "build_debian_package") as package_mock,
            self.assertRaisesRegex(SystemExit, "dirty"),
        ):
            build.package_kvmcomm(args)
        package_mock.assert_not_called()

    def test_kvmcomm_packaging_refuses_an_empty_update_key_ring(self) -> None:
        args = argparse.Namespace(version=None, docker=False)
        with tempfile.TemporaryDirectory() as directory:
            keyring = Path(directory) / "update-keys.json"
            keyring.write_text(json.dumps({"schema": 1, "keys": []}), encoding="utf-8")
            with (
                mock.patch.object(build, "UPDATE_KEYRING", keyring),
                mock.patch.object(build, "require_dir"),
                mock.patch.object(build, "require_files"),
                mock.patch.object(build, "build_debian_package") as package_mock,
                self.assertRaisesRegex(SystemExit, "empty update key ring"),
            ):
                build.package_kvmcomm(args)
            package_mock.assert_not_called()

    def test_kvmcomm_packaging_allows_an_empty_key_ring_on_request(self) -> None:
        args = argparse.Namespace(
            version=None, docker=False, allow_empty_keyring=True
        )
        with tempfile.TemporaryDirectory() as directory:
            keyring = Path(directory) / "update-keys.json"
            keyring.write_text(json.dumps({"schema": 1, "keys": []}), encoding="utf-8")
            with (
                mock.patch.object(build, "UPDATE_KEYRING", keyring),
                mock.patch.object(build, "require_dir"),
                mock.patch.object(build, "require_files"),
                mock.patch.object(build, "build_debian_package") as package_mock,
            ):
                self.assertEqual(build.package_kvmcomm(args), 0)
            package_mock.assert_called_once()

    def test_missing_key_ring_is_left_to_the_packaging_step(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            keyring = Path(directory) / "absent.json"
            with mock.patch.object(build, "UPDATE_KEYRING", keyring):
                build.require_non_empty_update_keyring("test", False)

    def test_key_ring_defaults_to_the_checked_in_official_public_keys(self) -> None:
        defaults = list(build.DEFAULT_TRUSTED_PUBLIC_KEYS)
        self.assertEqual(
            [path.name for path in defaults],
            ["nanokvm-go-release.pub.pem", "nanokvm-go-ci.pub.pem"],
        )
        self.assertEqual(list(build.resolve_trusted_public_keys(None, ())), defaults)
        explicit = Path("/nonexistent/other.pem")
        self.assertEqual(
            list(build.resolve_trusted_public_keys(None, (explicit,))), [explicit]
        )

    def test_official_signing_key_keeps_every_official_key_pinned(self) -> None:
        official = {path: bytes([index]) * 32 for index, path in enumerate(build.DEFAULT_TRUSTED_PUBLIC_KEYS)}
        signing_key = Path("/offline/key.pem")
        for path, public_key in official.items():
            with (
                self.subTest(signer=path.name),
                mock.patch.object(build, "public_key_from_public", side_effect=official.__getitem__),
                mock.patch.object(build, "public_key_from_private", return_value=public_key),
            ):
                self.assertEqual(
                    list(build.resolve_trusted_public_keys(signing_key, ())),
                    list(build.DEFAULT_TRUSTED_PUBLIC_KEYS),
                )

    def test_throwaway_signing_key_pins_only_itself(self) -> None:
        official = {path: bytes([index]) * 32 for index, path in enumerate(build.DEFAULT_TRUSTED_PUBLIC_KEYS)}
        with (
            mock.patch.object(build, "public_key_from_public", side_effect=official.__getitem__),
            mock.patch.object(build, "public_key_from_private", return_value=b"\xff" * 32),
        ):
            self.assertEqual(
                list(build.resolve_trusted_public_keys(Path("/tmp/test.pem"), ())), []
            )

    def test_key_ring_dropping_an_official_key_is_refused(self) -> None:
        official = {path: bytes([index]) * 32 for index, path in enumerate(build.DEFAULT_TRUSTED_PUBLIC_KEYS)}
        root_id, ci_id = (build.update_key_id(key) for key in official.values())
        cases = {
            "complete": ([root_id, ci_id], False, None),
            "no official key": (["sha256:" + "f" * 64], False, None),
            "drops ci": ([root_id, "sha256:" + "e" * 64], False, "drops official key"),
            "drops ci on request": ([root_id], True, None),
        }
        for name, (pinned, allow, error) in cases.items():
            with self.subTest(name), tempfile.TemporaryDirectory() as directory:
                keyring = Path(directory) / "update-keys.json"
                keyring.write_text(
                    json.dumps({"schema": 1, "keys": [{"keyId": key} for key in pinned]}),
                    encoding="utf-8",
                )
                with mock.patch.object(build, "public_key_from_public", side_effect=official.__getitem__):
                    if error is None:
                        build.require_complete_official_keys("test", keyring, allow)
                    else:
                        with self.assertRaisesRegex(SystemExit, error):
                            build.require_complete_official_keys("test", keyring, allow)

    def test_partially_missing_official_keys_are_refused(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            present = Path(directory) / "release.pub.pem"
            present.write_text("unused", encoding="utf-8")
            absent = Path(directory) / "ci.pub.pem"
            with (
                mock.patch.object(build, "DEFAULT_TRUSTED_PUBLIC_KEYS", (present, absent)),
                self.assertRaisesRegex(SystemExit, "official update public key missing"),
            ):
                build.resolve_trusted_public_keys(None, ())

    def test_unversioned_kvmcomm_packaging_retains_development_behavior(self) -> None:
        args = argparse.Namespace(version=None, docker=False)
        with (
            mock.patch.object(build, "verify_release_apps_checkout") as verify_mock,
            mock.patch.object(build, "require_dir"),
            mock.patch.object(build, "require_files"),
            mock.patch.object(build, "require_non_empty_update_keyring"),
            mock.patch.object(build, "build_debian_package") as package_mock,
        ):
            self.assertEqual(build.package_kvmcomm(args), 0)

        verify_mock.assert_not_called()
        package_mock.assert_called_once_with(
            "package_kvmcomm",
            "kvmcomm",
            build.KVMCOMM_PACKAGE_DIR,
            use_docker=False,
            version=None,
        )

    def test_debian_manifest_excludes_host_only_apps_utils(self) -> None:
        manifest = (
            build.ROOT
            / "debhelper/packages/kvmcomm/debian/kvmcomm.install"
        ).read_text(encoding="utf-8")
        self.assertNotIn("rootfs/kvmcomm/*", manifest)
        self.assertNotIn("apps/_utils", manifest)
        packaged_apps_files = {
            line.split()[0].removeprefix("rootfs/kvmcomm/apps/")
            for line in manifest.splitlines()
            if line.startswith("rootfs/kvmcomm/apps/")
        }
        expected_apps_files = {
            path.relative_to(build.APPS_DIR).as_posix()
            for path in build.APPS_DIR.rglob("*")
            if path.is_file()
            and ".git" not in path.relative_to(build.APPS_DIR).parts
            and "_utils" not in path.relative_to(build.APPS_DIR).parts
        }
        self.assertEqual(packaged_apps_files, expected_apps_files)

    def test_debian_manifest_preserves_other_kvmcomm_runtime_inputs(self) -> None:
        manifest = (
            build.ROOT
            / "debhelper/packages/kvmcomm/debian/kvmcomm.install"
        ).read_text(encoding="utf-8")
        declared_runtime_entries = {
            line.split()[0].removeprefix("rootfs/kvmcomm/").split("/", 1)[0]
            for line in manifest.splitlines()
            if line.startswith("rootfs/kvmcomm/")
        }
        expected_runtime_entries = {
            path.name
            for path in (build.APPS_DIR.parent).iterdir()
        }
        self.assertEqual(declared_runtime_entries, expected_runtime_entries)


if __name__ == "__main__":
    unittest.main()
