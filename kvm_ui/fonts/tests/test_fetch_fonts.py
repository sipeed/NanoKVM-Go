#!/usr/bin/env python3
"""Focused offline contract tests for the font fetcher and packer."""

from __future__ import annotations

import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest import mock
from urllib.error import URLError
from contextlib import redirect_stderr


FETCHER_PATH = Path(__file__).resolve().parents[1] / "fetch_fonts.py"
SPEC = importlib.util.spec_from_file_location("fetch_fonts_under_test", FETCHER_PATH)
assert SPEC is not None and SPEC.loader is not None
fetch_fonts = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = fetch_fonts
SPEC.loader.exec_module(fetch_fonts)

PACKAGER_PATH = Path(__file__).resolve().parents[1] / "package_fonts.py"
PACKAGER_SPEC = importlib.util.spec_from_file_location(
    "package_fonts_under_test", PACKAGER_PATH
)
assert PACKAGER_SPEC is not None and PACKAGER_SPEC.loader is not None
package_fonts = importlib.util.module_from_spec(PACKAGER_SPEC)
sys.modules[PACKAGER_SPEC.name] = package_fonts
PACKAGER_SPEC.loader.exec_module(package_fonts)


class FakeResponse:
    def __init__(self, payload: bytes) -> None:
        self._stream = io.BytesIO(payload)

    def __enter__(self) -> "FakeResponse":
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> None:
        self._stream.close()

    def read(self, size: int = -1) -> bytes:
        return self._stream.read(size)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def git_blob_sha1(data: bytes) -> str:
    digest = hashlib.sha1()
    digest.update(f"blob {len(data)}\0".encode("ascii"))
    digest.update(data)
    return digest.hexdigest()


class FontFetcherTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_directory = tempfile.TemporaryDirectory()
        self.target = Path(self.temp_directory.name) / "payload"
        self.local_license_source_directory = (
            Path(self.temp_directory.name) / "tracked-license-records"
        )
        self.local_notice_data = b"locked GNU Unifont attribution notice"
        self.local_notice_path = (
            self.local_license_source_directory
            / "LICENSES"
            / "GNU-Unifont-NOTICE.txt"
        )
        self.local_notice_path.parent.mkdir(parents=True)
        self.local_notice_path.write_bytes(self.local_notice_data)
        self.noto_data = b"tiny noto font"
        self.explicit_data = b"tiny explicit font"
        self.noto_license_data = b"upstream Noto notice"
        self.gnu_license_data = b"complete GNU Unifont license"
        self.tree_url = "https://example.invalid/tree"
        self.raw_prefix = "https://example.invalid/raw/"
        self.noto_license_url = "https://example.invalid/Noto-LICENSE"
        self.gnu_license_url = "https://example.invalid/GNU-Unifont-LICENSE"
        self.noto_path = "hinted/ttf/NotoTiny/NotoTiny-Regular.ttf"
        self.tree_entry = {
            "path": self.noto_path,
            "type": "blob",
            "sha": git_blob_sha1(self.noto_data),
            "size": len(self.noto_data),
        }
        self.lock = {
            "schema_version": 3,
            "catalog": "test-catalog",
            "expected_asset_count": 2,
            "expected_total_bytes": len(self.noto_data) + len(self.explicit_data),
            "source_sets": {
                "noto": {
                    "url_template": self.raw_prefix + "{path}",
                    "license_evidence_id": "noto",
                },
                "demo": {
                    "url_template": self.raw_prefix + "{path}",
                    "license_evidence_id": "gnu-unifont",
                },
            },
            "noto_tree": {
                "source_set": "noto",
                "tree_sha1": "0123456789012345678901234567890123456789",
                "tree_url": self.tree_url,
                "path_regex": "^hinted/ttf/Noto.+-Regular\\.ttf$",
                "install_subdirectory": "noto",
                "default_role": "future-catalog",
                "path_roles": {},
                "path_install_paths": {},
                "expected_count": 1,
                "expected_total_bytes": len(self.noto_data),
                "tree_manifest_sha256": fetch_fonts.tree_manifest_digest(
                    [self.tree_entry]
                ),
                "payload_manifest_sha256": fetch_fonts.payload_manifest_digest(
                    [(self.noto_path, sha256(self.noto_data), len(self.noto_data))]
                ),
            },
            "assets": [
                {
                    "id": "demo/explicit",
                    "source_set": "demo",
                    "source_path": "Explicit-Regular.ttf",
                    "install_name": "Explicit-Regular.ttf",
                    "size": len(self.explicit_data),
                    "sha256": sha256(self.explicit_data),
                    "role": "test",
                }
            ],
            "license_evidence": [
                {
                    "id": "noto",
                    "url": self.noto_license_url,
                    "install_path": "LICENSES/Noto-LICENSE.txt",
                    "sha256": sha256(self.noto_license_data),
                },
                {
                    "id": "gnu-unifont",
                    "url": self.gnu_license_url,
                    "install_path": "LICENSES/GNU-Unifont-LICENSE.txt",
                    "sha256": sha256(self.gnu_license_data),
                }
            ],
            "local_license_evidence": [
                {
                    "id": "gnu-unifont-attribution",
                    "source_path": "LICENSES/GNU-Unifont-NOTICE.txt",
                    "install_path": "LICENSES/GNU-Unifont-NOTICE.txt",
                    "sha256": sha256(self.local_notice_data),
                }
            ],
            "canonical_license": {
                "id": "canonical-ofl-1.1",
                "source_path": "LICENSES/OFL-1.1.txt",
                "install_path": "LICENSES/OFL-1.1.txt",
                "sha256": fetch_fonts.file_digest(
                    FETCHER_PATH.parent / "LICENSES" / "OFL-1.1.txt", None
                ).sha256,
            },
        }

    def tearDown(self) -> None:
        self.temp_directory.cleanup()

    def fake_urlopen(self, request, timeout: int):
        url = request.full_url
        if url == self.tree_url:
            return FakeResponse(
                json.dumps(
                    {
                        "sha": self.lock["noto_tree"]["tree_sha1"],
                        "truncated": False,
                        "tree": [self.tree_entry],
                    }
                ).encode("utf-8")
            )
        if url == self.raw_prefix + self.noto_path:
            return FakeResponse(self.noto_data)
        if url == self.raw_prefix + "Explicit-Regular.ttf":
            return FakeResponse(self.explicit_data)
        if url == self.noto_license_url:
            return FakeResponse(self.noto_license_data)
        if url == self.gnu_license_url:
            return FakeResponse(self.gnu_license_data)
        raise URLError(f"unexpected URL: {url}")

    def test_fetch_catalog_verifies_assets_and_license(self) -> None:
        fetch_fonts.prepare_target(self.target)
        with mock.patch.object(fetch_fonts, "urlopen", side_effect=self.fake_urlopen):
            assets = fetch_fonts.fetch_catalog(
                self.lock, self.target, self.local_license_source_directory
            )

        self.assertEqual(2, len(assets))
        self.assertEqual(
            self.noto_data,
            (self.target / "noto" / "NotoTiny" / "NotoTiny-Regular.ttf").read_bytes(),
        )
        self.assertEqual(
            self.explicit_data,
            (self.target / "Explicit-Regular.ttf").read_bytes(),
        )
        self.assertEqual(
            self.noto_license_data,
            (self.target / "LICENSES" / "Noto-LICENSE.txt").read_bytes(),
        )
        self.assertEqual(
            self.gnu_license_data,
            (self.target / "LICENSES" / "GNU-Unifont-LICENSE.txt").read_bytes(),
        )
        self.assertEqual(
            self.local_notice_data,
            (self.target / "LICENSES" / "GNU-Unifont-NOTICE.txt").read_bytes(),
        )

    def test_corrupt_cached_font_is_rejected_without_attestation(self) -> None:
        fetch_fonts.prepare_target(self.target)
        self.target.mkdir(exist_ok=True)
        nested_font = self.target / "noto" / "NotoTiny" / "NotoTiny-Regular.ttf"
        nested_font.parent.mkdir(parents=True)
        nested_font.write_bytes(b"corrupt")

        with mock.patch.object(fetch_fonts, "urlopen", side_effect=self.fake_urlopen):
            with self.assertRaises(fetch_fonts.FontFetchError):
                fetch_fonts.fetch_catalog(
                    self.lock, self.target, self.local_license_source_directory
                )
        self.assertFalse((self.target / fetch_fonts.ATTESTATION_NAME).exists())

    def test_download_failure_cleans_temporary_files(self) -> None:
        fetch_fonts.prepare_target(self.target)

        def failing_urlopen(request, timeout: int):
            if request.full_url == self.raw_prefix + "Explicit-Regular.ttf":
                raise URLError("network unavailable")
            return self.fake_urlopen(request, timeout)

        with mock.patch.object(fetch_fonts, "urlopen", side_effect=failing_urlopen):
            with self.assertRaises(fetch_fonts.FontFetchError):
                fetch_fonts.fetch_catalog(
                    self.lock, self.target, self.local_license_source_directory
                )
        self.assertEqual([], list(self.target.rglob("*.part")))
        self.assertFalse((self.target / fetch_fonts.ATTESTATION_NAME).exists())

    def test_hash_mismatch_is_not_installed(self) -> None:
        fetch_fonts.prepare_target(self.target)

        def wrong_hash_urlopen(request, timeout: int):
            if request.full_url == self.raw_prefix + "Explicit-Regular.ttf":
                return FakeResponse(b"unexpected bytes")
            return self.fake_urlopen(request, timeout)

        with mock.patch.object(fetch_fonts, "urlopen", side_effect=wrong_hash_urlopen):
            with self.assertRaises(fetch_fonts.FontFetchError):
                fetch_fonts.fetch_catalog(
                    self.lock, self.target, self.local_license_source_directory
                )
        self.assertFalse((self.target / "Explicit-Regular.ttf").exists())
        self.assertEqual([], list(self.target.rglob("*.part")))

    def test_local_license_record_digest_mismatch_is_rejected(self) -> None:
        fetch_fonts.prepare_target(self.target)
        self.local_notice_path.write_bytes(b"corrupt local attribution")

        with mock.patch.object(fetch_fonts, "urlopen", side_effect=self.fake_urlopen):
            with self.assertRaises(fetch_fonts.FontFetchError):
                fetch_fonts.fetch_catalog(
                    self.lock, self.target, self.local_license_source_directory
                )
        self.assertFalse(
            (self.target / "LICENSES" / "GNU-Unifont-NOTICE.txt").exists()
        )

    def test_symlinked_local_license_record_is_rejected(self) -> None:
        fetch_fonts.prepare_target(self.target)
        source = self.local_notice_path.with_name("notice-source.txt")
        source.write_bytes(self.local_notice_data)
        self.local_notice_path.unlink()
        self.local_notice_path.symlink_to(source)

        with mock.patch.object(fetch_fonts, "urlopen", side_effect=self.fake_urlopen):
            with self.assertRaises(fetch_fonts.FontFetchError):
                fetch_fonts.fetch_catalog(
                    self.lock, self.target, self.local_license_source_directory
                )
        self.assertFalse(
            (self.target / "LICENSES" / "GNU-Unifont-NOTICE.txt").exists()
        )

    def test_target_resolution_and_argument_limit(self) -> None:
        self.assertEqual(FETCHER_PATH.parent, fetch_fonts.resolve_target(None))
        with io.StringIO() as stderr, redirect_stderr(stderr):
            with self.assertRaises(SystemExit):
                fetch_fonts.parse_arguments(["one", "two"])

    def test_checked_in_lock_defines_the_full_catalog(self) -> None:
        lock = fetch_fonts.load_lock()
        explicit_assets = fetch_fonts.explicit_assets(lock)
        fetch_fonts.validate_license_references(lock)

        self.assertEqual(3, lock["schema_version"])
        self.assertEqual(242, lock["expected_asset_count"])
        self.assertEqual(144586156, lock["expected_total_bytes"])
        self.assertEqual(233, lock["noto_tree"]["expected_count"])
        self.assertEqual(9, len(explicit_assets))
        self.assertEqual(
            "b7a97d3a34d554df24414b58658f98b639548ab771ff6a147d751a1294e7b353",
            lock["noto_tree"]["payload_manifest_sha256"],
        )
        self.assertEqual(
            {
                "active-cjk-hk",
                "active-cjk-jp",
                "active-cjk-kr",
                "active-cjk-sc",
                "active-cjk-tc",
                "future-emoji-cbdt",
                "future-emoji-colrv1",
                "future-unifont",
                "future-unifont-upper",
            },
            {asset.role for asset in explicit_assets},
        )
        self.assertEqual(
            {
                "noto": "noto-fonts",
                "source-han": "source-han-sans",
                "noto-emoji": "noto-emoji",
                "gnu-unifont": "gnu-unifont",
            },
            {
                name: source_set["license_evidence_id"]
                for name, source_set in lock["source_sets"].items()
            },
        )
        licenses = {entry["id"]: entry for entry in lock["license_evidence"]}
        self.assertEqual(
            "https://unifoundry.com/LICENSE.txt", licenses["gnu-unifont"]["url"]
        )
        self.assertEqual(
            "LICENSES/GNU-Unifont-LICENSE.txt",
            licenses["gnu-unifont"]["install_path"],
        )
        self.assertEqual(
            "1e74cb82bf476843e97c2596297b04219b1a7e51f7238944a8c031cb9401fa87",
            licenses["gnu-unifont"]["sha256"],
        )
        local_licenses = {
            entry["id"]: entry for entry in lock["local_license_evidence"]
        }
        local_notice = local_licenses["gnu-unifont-attribution"]
        self.assertEqual(
            "LICENSES/GNU-Unifont-NOTICE.txt", local_notice["install_path"]
        )
        self.assertEqual(
            local_notice["sha256"],
            sha256(
                (FETCHER_PATH.parent / local_notice["source_path"]).read_bytes()
            ),
        )

    def test_non_directory_target_is_rejected(self) -> None:
        non_directory = Path(self.temp_directory.name) / "not-a-directory"
        non_directory.write_text("not a directory", encoding="utf-8")
        with self.assertRaises(fetch_fonts.FontFetchError):
            fetch_fonts.prepare_target(non_directory)

    def test_attestation_binds_the_current_lock(self) -> None:
        fetch_fonts.prepare_target(self.target)
        lock_path = self.target / "test-font-lock.json"
        lock_path.write_text(json.dumps(self.lock), encoding="utf-8")
        with mock.patch.object(fetch_fonts, "LOCK_PATH", lock_path):
            fetch_fonts.write_attestation(self.lock, self.target)
            expected_digest = fetch_fonts.file_digest(lock_path, None).sha256

        attestation = json.loads(
            (self.target / fetch_fonts.ATTESTATION_NAME).read_text(encoding="utf-8")
        )
        self.assertEqual(expected_digest, attestation["font_lock_sha256"])
        self.assertEqual(2, attestation["asset_count"])

    def write_package_attestation(self, source: Path, lock_path: Path) -> None:
        attestation = {
            "schema_version": 1,
            "catalog": self.lock["catalog"],
            "font_lock_sha256": sha256(lock_path.read_bytes()),
            "asset_count": self.lock["expected_asset_count"],
            "total_bytes": self.lock["expected_total_bytes"],
        }
        (source / fetch_fonts.ATTESTATION_NAME).write_text(
            json.dumps(attestation), encoding="utf-8"
        )

    def populate_package_source(self, source: Path, lock_path: Path) -> None:
        fetch_fonts.prepare_target(source)
        with mock.patch.object(fetch_fonts, "urlopen", side_effect=self.fake_urlopen):
            fetch_fonts.fetch_catalog(
                self.lock, source, self.local_license_source_directory
            )
        self.write_package_attestation(source, lock_path)

    def create_package_source(self) -> tuple[Path, Path]:
        source = Path(self.temp_directory.name) / "source"
        lock_path = Path(self.temp_directory.name) / "font-lock.json"
        lock_path.write_text(json.dumps(self.lock), encoding="utf-8")
        self.populate_package_source(source, lock_path)
        return source, lock_path

    def build_test_archive(self, source: Path, lock_path: Path, name: str) -> Path:
        output = Path(self.temp_directory.name) / name
        package_fonts.build_archive(source, output, lock_path=lock_path)
        return output

    def extract_archive(self, archive: Path, destination_name: str) -> Path:
        destination = Path(self.temp_directory.name) / destination_name
        destination.mkdir()
        result = subprocess.run(
            ["tar", "-xzf", str(archive), "-C", str(destination)],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        return destination / "usr" / "share" / "fonts"

    def run_helper(
        self, font_root: Path, name: str, *arguments: str, env: object = None
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                str(font_root / package_fonts.METADATA_DIRECTORY / name),
                *arguments,
            ],
            check=False,
            capture_output=True,
            text=True,
            env=env,
        )

    def result_fields(self, result: subprocess.CompletedProcess[str]) -> dict[str, str]:
        fields: dict[str, str] = {}
        for line in result.stdout.splitlines():
            self.assertIn("=", line, result.stdout)
            key, value = line.split("=", 1)
            fields[key] = value
        return fields

    def assert_error(
        self, result: subprocess.CompletedProcess[str], code: int, reason: str
    ) -> None:
        self.assertEqual(code, result.returncode, result.stderr)
        fields = self.result_fields(result)
        self.assertEqual("error", fields.get("result"))
        self.assertEqual(reason, fields.get("reason"))
        self.assertEqual(str(code), fields.get("exit_code"))

    def helper_environment_with(self, name: str, body: str) -> dict[str, str]:
        command_directory = Path(self.temp_directory.name) / f"commands-{name}"
        command_directory.mkdir()
        command = command_directory / name
        command.write_text(body, encoding="utf-8")
        command.chmod(0o755)
        environment = dict(os.environ)
        environment["PATH"] = f"{command_directory}:{environment.get('PATH', '')}"
        return environment

    def test_package_tar_gz_extracts_rootfs_layout_and_helpers(self) -> None:
        source, lock_path = self.create_package_source()
        first_archive = self.build_test_archive(source, lock_path, "first.tar.gz")
        second_archive = self.build_test_archive(source, lock_path, "second.tar.gz")

        self.assertEqual(first_archive.read_bytes(), second_archive.read_bytes())
        self.assertEqual(
            package_fonts.DEFAULT_OUTPUT_PATH,
            package_fonts.parse_arguments([]).output,
        )
        with self.assertRaises(package_fonts.FontPackageError):
            package_fonts.build_archive(
                source,
                Path(self.temp_directory.name) / "retired-installer.run",
                lock_path=lock_path,
            )

        state_member_name = (
            f"usr/share/fonts/{package_fonts.INSTALL_STATE_NAME}"
        )
        with tarfile.open(first_archive, mode="r:gz") as archive:
            members = archive.getmembers()
            names = [member.name for member in members]
            self.assertEqual(state_member_name, names[0])
            self.assertEqual(state_member_name, names[-1])
            self.assertEqual(b"installing\n", archive.extractfile(members[0]).read())
            self.assertEqual(b"installed\n", archive.extractfile(members[-1]).read())
            self.assertTrue(
                all(name.startswith("usr/share/fonts/") for name in names)
            )
            self.assertTrue(
                all(
                    not Path(name).is_absolute()
                    and ".." not in Path(name).parts
                    for name in names
                )
            )
            self.assertTrue(
                all(
                    member.uid == 0
                    and member.gid == 0
                    and member.uname == ""
                    and member.gname == ""
                    and member.mtime == 0
                    for member in members
                )
            )
            member_by_name = {member.name: member for member in members}
            for script_name in (
                package_fonts.STATUS_SCRIPT_NAME,
                package_fonts.UNINSTALL_SCRIPT_NAME,
                package_fonts.VERIFY_SCRIPT_NAME,
            ):
                self.assertEqual(
                    0o755,
                    member_by_name[f"usr/share/fonts/{script_name}"].mode,
                )
            for member in members:
                if member.name not in {
                    f"usr/share/fonts/{package_fonts.STATUS_SCRIPT_NAME}",
                    f"usr/share/fonts/{package_fonts.UNINSTALL_SCRIPT_NAME}",
                    f"usr/share/fonts/{package_fonts.VERIFY_SCRIPT_NAME}",
                }:
                    self.assertEqual(0o644, member.mode)

        font_root = self.extract_archive(first_archive, "rootfs")
        self.assertEqual(
            self.noto_data,
            (font_root / "noto" / "NotoTiny" / "NotoTiny-Regular.ttf").read_bytes(),
        )
        self.assertEqual(
            self.explicit_data, (font_root / "Explicit-Regular.ttf").read_bytes()
        )
        self.assertEqual(
            self.gnu_license_data,
            (font_root / "LICENSES" / "GNU-Unifont-LICENSE.txt").read_bytes(),
        )
        for script_name in ("status.sh", "uninstall.sh", "verify.sh"):
            self.assertTrue(
                os.access(
                    font_root / package_fonts.METADATA_DIRECTORY / script_name,
                    os.X_OK,
                )
            )

        status_result = self.run_helper(font_root, "status.sh")
        self.assertEqual(0, status_result.returncode, status_result.stderr)
        status = self.result_fields(status_result)
        self.assertEqual(str(font_root.resolve()), status.get("target_directory"))
        self.assertEqual("installed", status.get("install_state"))
        self.assertEqual("not_checked", status.get("integrity"))
        self.assertEqual(self.lock["catalog"], status.get("catalog"))
        self.assertEqual("6", status.get("managed_file_count"))

        verify_result = self.run_helper(font_root, "verify.sh")
        self.assertEqual(0, verify_result.returncode, verify_result.stderr)
        verify = self.result_fields(verify_result)
        self.assertEqual("verified", verify.get("integrity"))
        self.assertEqual("installed", verify.get("install_state"))

    def test_partial_archive_state_is_observable_and_verification_refuses(self) -> None:
        source, lock_path = self.create_package_source()
        archive_path = self.build_test_archive(source, lock_path, "font-package.tar.gz")
        partial_root = Path(self.temp_directory.name) / "partial-rootfs"
        partial_root.mkdir()

        with tarfile.open(archive_path, mode="r:gz") as archive:
            members = archive.getmembers()
            archive.extractall(partial_root, members=members[:-1])

        font_root = partial_root / "usr" / "share" / "fonts"
        self.assertEqual(
            "installing\n",
            (font_root / package_fonts.INSTALL_STATE_NAME).read_text(
                encoding="utf-8"
            ),
        )
        status_result = self.run_helper(font_root, "status.sh")
        self.assertEqual(0, status_result.returncode, status_result.stderr)
        self.assertEqual(
            "installing", self.result_fields(status_result).get("install_state")
        )
        self.assert_error(
            self.run_helper(font_root, "verify.sh"),
            23,
            "catalog_operation_incomplete",
        )

    def test_verify_detects_missing_and_changed_verifiable_content(self) -> None:
        source, lock_path = self.create_package_source()
        archive_path = self.build_test_archive(source, lock_path, "font-package.tar.gz")
        font_root = self.extract_archive(archive_path, "verify-rootfs")

        missing_path = font_root / "Explicit-Regular.ttf"
        missing_path.unlink()
        self.assert_error(
            self.run_helper(font_root, "verify.sh"), 20, "managed_file_missing"
        )
        missing_path.write_bytes(self.explicit_data)

        corruptions = (
            font_root / "LICENSES" / "GNU-Unifont-LICENSE.txt",
            font_root / package_fonts.METADATA_DIRECTORY / "font-lock.json",
            font_root / package_fonts.METADATA_DIRECTORY / "SHA256SUMS",
            font_root / package_fonts.METADATA_DIRECTORY / "status.sh",
        )
        for path in corruptions:
            original = path.read_bytes()
            path.write_bytes(b"corrupt")
            self.assert_error(
                self.run_helper(font_root, "verify.sh"), 21, "sha256_mismatch"
            )
            path.write_bytes(original)

    def test_uninstall_is_noninteractive_forceful_and_preserves_control_layer(self) -> None:
        source, lock_path = self.create_package_source()
        archive_path = self.build_test_archive(source, lock_path, "font-package.tar.gz")
        font_root = self.extract_archive(archive_path, "uninstall-rootfs")
        obsolete_path = font_root / "old-release.ttf"
        obsolete_path.write_bytes(b"not part of this release")

        reextract_result = subprocess.run(
            ["tar", "-xzf", str(archive_path), "-C", str(font_root.parents[2])],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(0, reextract_result.returncode, reextract_result.stderr)
        self.assertTrue(obsolete_path.is_file())

        self.assert_error(
            self.run_helper(font_root, "uninstall.sh"), 2, "usage_error"
        )
        dry_run = self.run_helper(font_root, "uninstall.sh", "--dry-run")
        self.assertEqual(0, dry_run.returncode, dry_run.stderr)
        dry_fields = self.result_fields(dry_run)
        self.assertEqual("dry_run", dry_fields.get("operation"))
        self.assertEqual("6", dry_fields.get("planned_count"))
        self.assertEqual("0", dry_fields.get("removed_count"))
        self.assertEqual("0", dry_fields.get("failed_count"))
        self.assertTrue((font_root / "Explicit-Regular.ttf").is_file())

        (font_root / "Explicit-Regular.ttf").write_bytes(b"tampered font")
        uninstall_result = self.run_helper(font_root, "uninstall.sh", "--yes")
        self.assertEqual(0, uninstall_result.returncode, uninstall_result.stderr)
        uninstall = self.result_fields(uninstall_result)
        self.assertEqual("uninstall", uninstall.get("operation"))
        self.assertEqual("uninstalled", uninstall.get("install_state"))
        self.assertEqual("0", uninstall.get("failed_count"))
        self.assertFalse((font_root / "Explicit-Regular.ttf").exists())
        self.assertFalse((font_root / "noto").exists())
        self.assertFalse((font_root / "LICENSES").exists())
        self.assertTrue(obsolete_path.is_file())

        control_directory = font_root / package_fonts.METADATA_DIRECTORY
        self.assertTrue((control_directory / "status.sh").is_file())
        self.assertTrue((control_directory / "uninstall.sh").is_file())
        self.assertTrue((control_directory / "verify.sh").is_file())
        self.assertEqual(
            "uninstalled\n",
            (control_directory / "install-state").read_text(encoding="utf-8"),
        )
        self.assert_error(
            self.run_helper(font_root, "verify.sh"), 22, "catalog_uninstalled"
        )
        after_uninstall = self.run_helper(font_root, "status.sh")
        self.assertEqual(0, after_uninstall.returncode, after_uninstall.stderr)
        self.assertEqual(
            "uninstalled",
            self.result_fields(after_uninstall).get("install_state"),
        )

        repeated_uninstall = self.run_helper(font_root, "uninstall.sh", "--yes")
        self.assertEqual(
            0, repeated_uninstall.returncode, repeated_uninstall.stderr
        )
        repeated = self.result_fields(repeated_uninstall)
        self.assertEqual("0", repeated.get("removed_count"))
        self.assertEqual("6", repeated.get("missing_count"))

    def test_helper_error_codes_cover_environment_metadata_and_uninstall_failures(
        self,
    ) -> None:
        source, lock_path = self.create_package_source()
        archive_path = self.build_test_archive(source, lock_path, "font-package.tar.gz")

        environment_root = self.extract_archive(archive_path, "environment-rootfs")
        empty_path = Path(self.temp_directory.name) / "empty-path"
        empty_path.mkdir()
        environment = dict(os.environ)
        environment["PATH"] = str(empty_path)
        self.assert_error(
            self.run_helper(environment_root, "status.sh", env=environment),
            10,
            "environment_unavailable",
        )

        metadata_root = self.extract_archive(archive_path, "metadata-rootfs")
        (metadata_root / package_fonts.INSTALL_STATE_NAME).unlink()
        self.assert_error(
            self.run_helper(metadata_root, "status.sh"), 12, "metadata_missing"
        )

        unsafe_root = self.extract_archive(archive_path, "unsafe-rootfs")
        unsafe_script = (
            unsafe_root / package_fonts.METADATA_DIRECTORY / "uninstall.sh"
        )
        unsafe_script.write_text(
            unsafe_script.read_text(encoding="utf-8").replace(
                "Explicit-Regular.ttf", "../unsafe-path"
            ),
            encoding="utf-8",
        )
        unsafe_script.chmod(0o755)
        self.assert_error(
            self.run_helper(unsafe_root, "uninstall.sh", "--yes"),
            13,
            "unsafe_or_invalid_metadata",
        )

        protected_root = self.extract_archive(archive_path, "protected-rootfs")
        protected_script = (
            protected_root / package_fonts.METADATA_DIRECTORY / "uninstall.sh"
        )
        protected_script.write_text(
            protected_script.read_text(encoding="utf-8").replace(
                "Explicit-Regular.ttf", ".kvm-ui-fonts/status.sh"
            ),
            encoding="utf-8",
        )
        protected_script.chmod(0o755)
        self.assert_error(
            self.run_helper(protected_root, "uninstall.sh", "--yes"),
            30,
            "uninstall_preflight_refused",
        )

        target_error_root = self.extract_archive(archive_path, "target-error-rootfs")
        self.assert_error(
            self.run_helper(
                target_error_root,
                "uninstall.sh",
                "--yes",
                env=self.helper_environment_with("mktemp", "#!/bin/sh\nexit 1\n"),
            ),
            11,
            "target_unavailable",
        )

        delete_error_root = self.extract_archive(archive_path, "delete-error-rootfs")
        delete_result = self.run_helper(
            delete_error_root,
            "uninstall.sh",
            "--yes",
            env=self.helper_environment_with("rm", "#!/bin/sh\nexit 1\n"),
        )
        self.assert_error(delete_result, 31, "uninstall_delete_failed")
        delete_fields = self.result_fields(delete_result)
        self.assertGreater(int(delete_fields["failed_count"]), 0)
        self.assertEqual(
            "uninstalling",
            (
                delete_error_root
                / package_fonts.METADATA_DIRECTORY
                / "install-state"
            ).read_text(encoding="utf-8").strip(),
        )

    def test_package_rejects_corrupt_staged_local_license_record(self) -> None:
        source, lock_path = self.create_package_source()
        (source / "LICENSES" / "GNU-Unifont-NOTICE.txt").write_bytes(b"corrupt")

        with self.assertRaises(package_fonts.FontPackageError):
            package_fonts.build_archive(
                source,
                Path(self.temp_directory.name) / "font-package.tar.gz",
                lock_path=lock_path,
            )

    def test_package_rejects_stale_attestation(self) -> None:
        source, lock_path = self.create_package_source()
        stale_attestation = json.loads(
            (source / fetch_fonts.ATTESTATION_NAME).read_text(encoding="utf-8")
        )
        stale_attestation["font_lock_sha256"] = "0" * 64
        (source / fetch_fonts.ATTESTATION_NAME).write_text(
            json.dumps(stale_attestation), encoding="utf-8"
        )

        with self.assertRaises(package_fonts.FontPackageError):
            package_fonts.build_archive(
                source,
                Path(self.temp_directory.name) / "font-package.tar.gz",
                lock_path=lock_path,
            )


if __name__ == "__main__":
    unittest.main()
