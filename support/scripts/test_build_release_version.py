#!/usr/bin/env python3
from __future__ import annotations

import argparse
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPOSITORY_ROOT))

import build


class BuildReleaseVersionTests(unittest.TestCase):
    def setUp(self) -> None:
        test_parent = REPOSITORY_ROOT / "build/test-tmp"
        test_parent.mkdir(parents=True, exist_ok=True)
        self.temporary_directory = tempfile.TemporaryDirectory(
            prefix=".test-build-release-version-",
            dir=test_parent,
        )
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)
        self.packages_dir = self.root / "packages"
        self.build_dir = self.root / "build"
        self.packages_dir.mkdir()

        packages_patch = mock.patch.object(
            build,
            "DEBHELPER_PACKAGES_DIR",
            self.packages_dir,
        )
        build_patch = mock.patch.object(build, "BUILD_DIR", self.build_dir)
        packages_patch.start()
        build_patch.start()
        self.addCleanup(packages_patch.stop)
        self.addCleanup(build_patch.stop)

    def make_package(self, package_name: str, version: str = "0.0.24") -> Path:
        package_dir = self.packages_dir / package_name
        debian_dir = package_dir / "debian"
        debian_dir.mkdir(parents=True)
        (debian_dir / "changelog").write_bytes(
            f"{package_name} ({version}) unstable; urgency=medium\n\n"
            "  * Existing entry.\n".encode()
        )
        (package_dir / "payload").write_bytes(b"payload\n")
        return package_dir

    def make_artifacts(
        self,
        package_name: str,
        version: str,
        architecture: str = "armhf",
    ) -> list[Path]:
        basename = f"{package_name}_{version}_{architecture}"
        paths = []
        for suffix in build.PACKAGE_ARTIFACT_SUFFIXES:
            path = self.packages_dir / f"{basename}.{suffix}"
            path.write_bytes(suffix.encode())
            paths.append(path)
        return paths

    def test_validate_debian_version_accepts_dpkg_valid_version(self) -> None:
        with mock.patch.object(
            build,
            "run",
            return_value=subprocess.CompletedProcess([], 0, stdout="", stderr=""),
        ) as run_mock:
            self.assertEqual(
                build.validate_debian_version("release", "0.0.25~rc1-1"),
                "0.0.25~rc1-1",
            )

        run_mock.assert_called_once_with(
            "release",
            ("dpkg", "--validate-version", "0.0.25~rc1-1"),
            capture_output=True,
        )

    def test_validate_update_version_accepts_exact_numeric_version(self) -> None:
        with mock.patch.object(
            build,
            "validate_debian_version",
            return_value="0.0.25",
        ):
            self.assertEqual(
                build.validate_update_version("release", "0.0.25"),
                "0.0.25",
            )

    def test_validate_update_version_rejects_debian_only_syntax(self) -> None:
        for version in (
            "0.0.25~rc1-1",
            "0.0.25-1",
            "0.0.25-rc.1",
            "0.0.25+build.1",
            "0.0.25.1",
        ):
            with (
                self.subTest(version=version),
                mock.patch.object(
                    build,
                    "validate_debian_version",
                    return_value=version,
                ),
                self.assertRaisesRegex(SystemExit, "exactly three numeric components"),
            ):
                build.validate_update_version("release", version)

    def test_validate_update_version_rejects_component_overflow(self) -> None:
        version = "18446744073709551616.0.0"
        with (
            mock.patch.object(
                build,
                "validate_debian_version",
                return_value=version,
            ),
            self.assertRaisesRegex(SystemExit, "64-bit component range"),
        ):
            build.validate_update_version("release", version)

    def test_validate_debian_version_rejects_invalid_version(self) -> None:
        error = subprocess.CalledProcessError(
            1,
            ["dpkg", "--validate-version", "bad version"],
            stderr="version string has embedded spaces",
        )
        with mock.patch.object(build, "run", side_effect=error):
            with self.assertRaisesRegex(SystemExit, "invalid Debian package version"):
                build.validate_debian_version("release", "bad version")

    def test_validate_debian_version_rejects_epoch_for_exact_filename_contract(self) -> None:
        with mock.patch.object(build, "run") as run_mock:
            with self.assertRaisesRegex(SystemExit, "epoch"):
                build.validate_debian_version("release", "1:0.0.25")
        run_mock.assert_not_called()

    def test_validate_debian_version_rejects_package_update_incompatible_prefix(self) -> None:
        with mock.patch.object(build, "run") as run_mock:
            with self.assertRaisesRegex(SystemExit, "leading 'v'"):
                build.validate_debian_version("release", "v0.0.25")
        run_mock.assert_not_called()

    def test_staged_changelog_changes_only_copy_and_cleans_after_success(self) -> None:
        package_dir = self.make_package("kvmcomm")
        source_bytes = (package_dir / "debian" / "changelog").read_bytes()

        with build.staged_debian_package(
            "package_kvmcomm",
            "kvmcomm",
            package_dir,
            "0.0.25",
        ) as staged_dir:
            staged_path = staged_dir
            staged_bytes = (staged_dir / "debian" / "changelog").read_bytes()
            self.assertTrue(staged_bytes.startswith(b"kvmcomm (0.0.25) "))
            self.assertEqual(staged_bytes.split(b"\n", 1)[1], source_bytes.split(b"\n", 1)[1])
            self.assertEqual((staged_dir / "payload").read_bytes(), b"payload\n")
            self.assertEqual((package_dir / "debian" / "changelog").read_bytes(), source_bytes)

        self.assertFalse(staged_path.exists())
        self.assertEqual((package_dir / "debian" / "changelog").read_bytes(), source_bytes)

    def test_staged_package_cleans_after_failure(self) -> None:
        package_dir = self.make_package("kvmcomm")
        staged_path: Path | None = None

        with self.assertRaisesRegex(RuntimeError, "stop"):
            with build.staged_debian_package(
                "package_kvmcomm",
                "kvmcomm",
                package_dir,
                "0.0.25",
            ) as staged_dir:
                staged_path = staged_dir
                raise RuntimeError("stop")

        self.assertIsNotNone(staged_path)
        self.assertFalse(staged_path.exists())

    def test_versioned_host_build_runs_from_staged_directory(self) -> None:
        package_dir = self.make_package("kvmcomm")
        stale_exact = self.make_artifacts("kvmcomm", "0.0.25")
        stale_other_version = self.make_artifacts("kvmcomm", "0.0.24")
        seen_workdirs: list[Path] = []

        def record_run(_target, command, **kwargs):
            self.assertEqual(command, build.DPKG_BUILDPACKAGE)
            seen_workdirs.append(kwargs["cwd"])
            return subprocess.CompletedProcess(command, 0)

        with (
            mock.patch.object(build, "validate_debian_version", return_value="0.0.25"),
            mock.patch.object(build, "run", side_effect=record_run),
            mock.patch.object(build, "collect_debian_artifacts") as collect_mock,
        ):
            build.build_debian_package(
                "package_kvmcomm",
                "kvmcomm",
                package_dir,
                use_docker=False,
                version="0.0.25",
            )

        self.assertEqual(len(seen_workdirs), 1)
        self.assertNotEqual(seen_workdirs[0], package_dir)
        self.assertTrue(seen_workdirs[0].name.startswith(".kvmcomm-release-"))
        self.assertFalse(seen_workdirs[0].exists())
        self.assertTrue(all(not artifact.exists() for artifact in stale_exact))
        self.assertTrue(all(artifact.exists() for artifact in stale_other_version))
        collect_mock.assert_called_once_with("package_kvmcomm", "kvmcomm", "0.0.25")

    def test_versioned_docker_build_receives_staged_directory_basename(self) -> None:
        package_dir = self.make_package("kvmcomm")
        docker_script = self.root / "build_in_docker.sh"
        docker_script.write_text("#!/bin/sh\n", encoding="utf-8")
        commands: list[tuple[str, ...]] = []

        def record_run(_target, command, **_kwargs):
            commands.append(tuple(command))
            return subprocess.CompletedProcess(command, 0)

        with (
            mock.patch.object(build, "DOCKER_PACKAGE_SCRIPT", docker_script),
            mock.patch.object(build, "validate_debian_version", return_value="0.0.25"),
            mock.patch.object(build, "run", side_effect=record_run),
            mock.patch.object(build, "collect_debian_artifacts"),
        ):
            build.build_debian_package(
                "package_kvmcomm",
                "kvmcomm",
                package_dir,
                use_docker=True,
                version="0.0.25",
            )

        self.assertEqual(len(commands), 1)
        self.assertEqual(commands[0][0], str(docker_script))
        self.assertTrue(commands[0][1].startswith(".kvmcomm-release-"))

    def test_unversioned_docker_build_retains_package_name_contract(self) -> None:
        package_dir = self.make_package("kvmcomm")
        docker_script = self.root / "build_in_docker.sh"
        docker_script.write_text("#!/bin/sh\n", encoding="utf-8")

        with (
            mock.patch.object(build, "DOCKER_PACKAGE_SCRIPT", docker_script),
            mock.patch.object(
                build,
                "run",
                return_value=subprocess.CompletedProcess([], 0),
            ) as run_mock,
            mock.patch.object(build, "validate_debian_version") as validate_mock,
            mock.patch.object(build, "collect_debian_artifacts") as collect_mock,
        ):
            build.build_debian_package(
                "package_kvmcomm",
                "kvmcomm",
                package_dir,
                use_docker=True,
            )

        validate_mock.assert_not_called()
        run_mock.assert_called_once_with(
            "package_kvmcomm",
            (str(docker_script), "kvmcomm"),
            cwd=build.ROOT,
        )
        collect_mock.assert_called_once_with("package_kvmcomm", "kvmcomm", None)

    def test_versioned_build_cleans_stage_when_build_command_fails(self) -> None:
        package_dir = self.make_package("kvmcomm")
        seen_workdirs: list[Path] = []

        def fail_run(_target, command, **kwargs):
            seen_workdirs.append(kwargs["cwd"])
            raise subprocess.CalledProcessError(2, command)

        with (
            mock.patch.object(build, "validate_debian_version", return_value="0.0.25"),
            mock.patch.object(build, "run", side_effect=fail_run),
            mock.patch.object(build, "collect_debian_artifacts") as collect_mock,
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                build.build_debian_package(
                    "package_kvmcomm",
                    "kvmcomm",
                    package_dir,
                    use_docker=False,
                    version="0.0.25",
                )

        self.assertEqual(len(seen_workdirs), 1)
        self.assertFalse(seen_workdirs[0].exists())
        collect_mock.assert_not_called()

    def test_unversioned_direct_build_retains_changelog_driven_behavior(self) -> None:
        package_dir = self.make_package("kvmcomm")
        source_bytes = (package_dir / "debian" / "changelog").read_bytes()

        with (
            mock.patch.object(
                build,
                "run",
                return_value=subprocess.CompletedProcess(build.DPKG_BUILDPACKAGE, 0),
            ) as run_mock,
            mock.patch.object(build, "validate_debian_version") as validate_mock,
            mock.patch.object(build, "collect_debian_artifacts") as collect_mock,
        ):
            build.build_debian_package(
                "package_kvmcomm",
                "kvmcomm",
                package_dir,
                use_docker=False,
            )

        validate_mock.assert_not_called()
        self.assertEqual(run_mock.call_args.kwargs["cwd"], package_dir)
        collect_mock.assert_called_once_with("package_kvmcomm", "kvmcomm", None)
        self.assertEqual((package_dir / "debian" / "changelog").read_bytes(), source_bytes)

    def test_release_forwards_exact_version_to_every_packaging_step(self) -> None:
        args = argparse.Namespace(
            version="0.0.26",
            docker=True,
            signing_key=Path("signing.pem"),
            trusted_public_key=[Path("next.pem")],
        )
        with (
            mock.patch.object(build, "validate_update_version", return_value="0.0.26") as validate_mock,
            mock.patch.object(
                build,
                "validate_optional_update_signing_key",
                return_value=args.signing_key,
            ) as signing_key_mock,
            mock.patch.object(build, "build_all") as build_all_mock,
            mock.patch.object(build, "package_kvmcomm") as kvmcomm_mock,
            mock.patch.object(build, "package_nanokvm_go") as nanokvm_mock,
            mock.patch.object(build, "package_update") as update_mock,
        ):
            self.assertEqual(build.release(args), 0)

        validate_mock.assert_called_once_with("release", "0.0.26")
        signing_key_mock.assert_called_once_with(args.signing_key)
        build_all_mock.assert_called_once()
        self.assertEqual(kvmcomm_mock.call_args.args[0].version, "0.0.26")
        self.assertEqual(nanokvm_mock.call_args.args[0].version, "0.0.26")
        self.assertEqual(update_mock.call_args.args[0].version, "0.0.26")
        self.assertTrue(kvmcomm_mock.call_args.args[0].docker)
        self.assertTrue(nanokvm_mock.call_args.args[0].docker)

    def test_release_without_signing_key_forwards_unsigned_mode(self) -> None:
        args = argparse.Namespace(
            version="0.0.26",
            docker=False,
            signing_key=None,
            trusted_public_key=[],
        )
        with (
            mock.patch.object(build, "validate_update_version", return_value="0.0.26"),
            mock.patch.object(
                build,
                "validate_optional_update_signing_key",
                return_value=None,
            ),
            mock.patch.object(build, "build_all") as build_all_mock,
            mock.patch.object(build, "package_kvmcomm"),
            mock.patch.object(build, "package_nanokvm_go"),
            mock.patch.object(build, "package_update") as update_mock,
        ):
            self.assertEqual(build.release(args), 0)

        self.assertIsNone(build_all_mock.call_args.args[0].signing_key)
        self.assertIsNone(update_mock.call_args.args[0].signing_key)

    def test_release_and_package_update_parsers_make_signing_key_optional(self) -> None:
        for command in ("release", "package_update"):
            with self.subTest(command=command):
                self.assertIsNone(
                    build.parse_args([command, "--version", "0.0.26"]).signing_key
                )

        signing_key = Path("/offline/update-signing.pem")
        self.assertEqual(
            build.parse_args(
                [
                    "package_update",
                    "--version",
                    "0.0.26",
                    "--signing-key",
                    str(signing_key),
                ]
            ).signing_key,
            signing_key,
        )

    def test_validate_optional_update_signing_key_accepts_missing_and_rejects_directory(self) -> None:
        self.assertIsNone(build.validate_optional_update_signing_key(None))
        with self.assertRaisesRegex(SystemExit, "external Ed25519 signing key"):
            build.validate_optional_update_signing_key(self.root)

        signing_key = self.root / "signing.pem"
        signing_key.write_text("test-only\n", encoding="utf-8")
        self.assertEqual(
            build.validate_optional_update_signing_key(signing_key), signing_key
        )

    def test_package_update_selects_signed_or_unsigned_helper_arguments(self) -> None:
        version = "0.0.26"
        self.build_dir.mkdir(parents=True)
        for package in ("nanokvm-go", "kvmcomm"):
            (self.build_dir / f"{package}_{version}_armhf.deb").write_bytes(b"deb")
        signing_key = self.root / "signing.pem"
        signing_key.write_text("test-only\n", encoding="utf-8")

        for supplied_key in (None, signing_key):
            with (
                self.subTest(signed=supplied_key is not None),
                mock.patch.object(
                    build, "verify_packaged_update_keyring"
                ) as verify_keyring_mock,
                mock.patch.object(build, "run") as run_mock,
            ):
                self.assertEqual(
                    build.package_update(
                        argparse.Namespace(version=version, signing_key=supplied_key)
                    ),
                    0,
                )

            command = list(run_mock.call_args.args[1])
            if supplied_key is None:
                verify_keyring_mock.assert_not_called()
                self.assertNotIn("--signing-key", command)
                self.assertNotIn("--keyring", command)
            else:
                verify_keyring_mock.assert_called_once_with(
                    "package_update",
                    self.build_dir / f"kvmcomm_{version}_armhf.deb",
                )
                self.assertEqual(
                    command[-4:],
                    [
                        "--signing-key",
                        str(signing_key),
                        "--keyring",
                        str(build.UPDATE_KEYRING),
                    ],
                )

    def test_packaged_keyring_inspection_uses_build_directory(self) -> None:
        package = self.build_dir / "kvmcomm_0.0.26_armhf.deb"
        package.parent.mkdir(parents=True)
        package.write_bytes(b"test-only")
        update_keyring = self.build_dir / "update-keys.json"
        update_keyring.write_bytes(b'{"schema":1,"keys":[]}\n')

        def extract_fixture(_target, command, **_kwargs):
            extraction_root = Path(command[-1])
            self.assertEqual(extraction_root.parent, self.build_dir)
            self.assertTrue(extraction_root.name.startswith(".nanokvm-update-package-"))
            verifier = extraction_root / "usr/bin/nanokvm-update-verify"
            packaged_keyring = extraction_root / "usr/share/nanokvm/update-keys.json"
            verifier.parent.mkdir(parents=True)
            verifier.write_bytes(b"test-only")
            packaged_keyring.parent.mkdir(parents=True)
            packaged_keyring.write_bytes(update_keyring.read_bytes())
            return subprocess.CompletedProcess(command, 0)

        with (
            mock.patch.object(build, "UPDATE_KEYRING", update_keyring),
            mock.patch.object(build, "run", side_effect=extract_fixture),
        ):
            build.verify_packaged_update_keyring("package_update", package)

        self.assertEqual(
            list(self.build_dir.glob(".nanokvm-update-package-*")),
            [],
        )

    def test_direct_package_parsers_make_version_optional(self) -> None:
        self.assertIsNone(build.parse_args(["package_kvmcomm"]).version)
        self.assertEqual(
            build.parse_args(["package_kvmcomm", "--version", "0.0.25"]).version,
            "0.0.25",
        )
        self.assertIsNone(build.parse_args(["package_nanokvm_go"]).version)

    def test_versioned_collection_moves_only_exact_artifacts(self) -> None:
        exact = self.make_artifacts("kvmcomm", "0.0.25")
        stale = self.make_artifacts("kvmcomm", "0.0.24")

        with mock.patch.object(build, "verify_debian_package_metadata") as verify_mock:
            build.collect_debian_artifacts("package_kvmcomm", "kvmcomm", "0.0.25")

        verify_mock.assert_called_once_with(
            "package_kvmcomm",
            exact[0],
            "kvmcomm",
            "0.0.25",
        )
        for artifact in exact:
            self.assertFalse(artifact.exists())
            self.assertTrue((self.build_dir / artifact.name).is_file())
        for artifact in stale:
            self.assertTrue(artifact.is_file())

    def test_versioned_collection_rejects_stale_other_version(self) -> None:
        self.make_artifacts("kvmcomm", "0.0.24")
        with self.assertRaisesRegex(SystemExit, "expected Debian artifact"):
            build.collect_debian_artifacts("package_kvmcomm", "kvmcomm", "0.0.25")

    def test_versioned_collection_rejects_wrong_architecture_filename(self) -> None:
        self.make_artifacts("kvmcomm", "0.0.25", architecture="amd64")
        with self.assertRaisesRegex(SystemExit, "expected Debian artifact"):
            build.collect_debian_artifacts("package_kvmcomm", "kvmcomm", "0.0.25")

    def test_debian_metadata_rejects_wrong_package_version_or_architecture(self) -> None:
        package_path = self.root / "package.deb"
        package_path.write_bytes(b"fixture")
        cases = (
            (["wrong", "0.0.25", "armhf"], "Package"),
            (["kvmcomm", "0.0.24", "armhf"], "Version"),
            (["kvmcomm", "0.0.25", "amd64"], "Architecture"),
        )
        for outputs, field in cases:
            with self.subTest(field=field):
                results = [
                    subprocess.CompletedProcess([], 0, stdout=f"{value}\n")
                    for value in outputs
                ]
                with mock.patch.object(build, "run", side_effect=results):
                    with self.assertRaisesRegex(SystemExit, f"unexpected {field}"):
                        build.verify_debian_package_metadata(
                            "package_kvmcomm",
                            package_path,
                            "kvmcomm",
                            "0.0.25",
                        )


if __name__ == "__main__":
    unittest.main()
