from __future__ import annotations

import base64
import gzip
import hashlib
import io
import json
import lzma
import os
import subprocess
import sys
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import package_firmware
import update_signing


REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
REPOSITORY_TEMP_PARENT = REPOSITORY_ROOT / "build/test-tmp"
SIGNING_VECTORS = (
    REPOSITORY_ROOT
    / "server/updateverify/testdata/artifact_signing_vectors_v1.json"
)
TEST_ONLY_ED25519_SEED = bytes.fromhex(
    "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
)
ED25519_PKCS8_SEED_PREFIX = bytes.fromhex("302e020100300506032b657004220420")
EXPECTED_XZ_SHA512 = (
    "gvKiubDXzLPm3Og6JC5N06IGOr4trEf+eNDSCj6shiQ8BbJNRyAT5SabF1ndn1lzm"
    "HH6sVWYPj3B6njRwrFb+g=="
)


def repository_temporary_directory() -> tempfile.TemporaryDirectory:
    REPOSITORY_TEMP_PARENT.mkdir(parents=True, exist_ok=True)
    return tempfile.TemporaryDirectory(
        prefix=".test-package-firmware-", dir=REPOSITORY_TEMP_PARENT
    )


def write_fixed_test_key(path: Path) -> None:
    der = ED25519_PKCS8_SEED_PREFIX + TEST_ONLY_ED25519_SEED
    encoded = base64.b64encode(der)
    lines = [encoded[offset : offset + 64] for offset in range(0, len(encoded), 64)]
    path.write_bytes(
        b"-----BEGIN PRIVATE KEY-----\n"
        + b"\n".join(lines)
        + b"\n-----END PRIVATE KEY-----\n"
    )
    path.chmod(0o600)


def firmware_vector() -> dict[str, object]:
    fixture = json.loads(SIGNING_VECTORS.read_text(encoding="utf-8"))
    return next(vector for vector in fixture["vectors"] if vector["kind"] == "firmware")


def run_package_firmware(
    *,
    version: str | None,
    archive: Path | None,
    signing_key: Path | None,
    keyring: Path | None,
    output: Path | None,
    update_uboot: bool = False,
    force: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess[bytes]:
    command = [sys.executable, str(Path(__file__).with_name("package_firmware.py"))]
    for option, value in (
        ("--version", version),
        ("--archive", archive),
        ("--signing-key", signing_key),
        ("--keyring", keyring),
        ("--output", output),
    ):
        if value is not None:
            command.extend((option, str(value)))
    if force:
        command.append("--force")
    if update_uboot:
        command.append("--update-uboot")
    return subprocess.run(
        command,
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=os.environ.copy(),
    )


def openssl_verifies(public_key: Path, artifact: Path, signature: Path) -> bool:
    result = subprocess.run(
        [
            "openssl",
            "pkeyutl",
            "-verify",
            "-rawin",
            "-pubin",
            "-inkey",
            str(public_key),
            "-in",
            str(artifact),
            "-sigfile",
            str(signature),
        ],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.returncode == 0


def compress_tar(tar_bytes: bytes, archive_format: str) -> bytes:
    if archive_format == "gz":
        return gzip.compress(tar_bytes, compresslevel=9, mtime=0)
    if archive_format == "xz":
        return lzma.compress(
            tar_bytes,
            format=lzma.FORMAT_XZ,
            check=lzma.CHECK_CRC64,
            preset=9,
        )
    raise AssertionError(f"unsupported test archive format: {archive_format}")


def make_archive(
    entries: list[tuple[str, str, bytes | str]],
    archive_format: str = "gz",
    *,
    tar_format: int = tarfile.PAX_FORMAT,
) -> bytes:
    tar_buffer = io.BytesIO()
    with tarfile.open(
        fileobj=tar_buffer, mode="w", format=tar_format
    ) as archive:
        for name, kind, value in entries:
            info = tarfile.TarInfo(name)
            info.mtime = 0
            info.uid = 0
            info.gid = 0
            info.uname = "root"
            info.gname = "root"
            if kind == "dir":
                info.type = tarfile.DIRTYPE
                info.mode = 0o755
                archive.addfile(info)
            elif kind in ("file", "contiguous", "sparse", "pax-header"):
                assert isinstance(value, bytes)
                if kind == "contiguous":
                    info.type = tarfile.CONTTYPE
                elif kind == "sparse":
                    info.type = tarfile.GNUTYPE_SPARSE
                elif kind == "pax-header":
                    info.type = tarfile.XHDTYPE
                info.mode = 0o644
                info.size = len(value)
                archive.addfile(info, io.BytesIO(value))
            elif kind == "symlink":
                assert isinstance(value, str)
                info.type = tarfile.SYMTYPE
                info.linkname = value
                info.mode = 0o777
                archive.addfile(info)
            elif kind == "hardlink":
                assert isinstance(value, str)
                info.type = tarfile.LNKTYPE
                info.linkname = value
                info.mode = 0o644
                archive.addfile(info)
            else:
                raise AssertionError(f"unsupported test member kind: {kind}")
    return compress_tar(tar_buffer.getvalue(), archive_format)


def make_header_only_archive(
    typeflag: bytes,
    declared_size: int,
    archive_format: str,
) -> bytes:
    info = tarfile.TarInfo("overlay/metadata")
    info.type = typeflag
    info.mode = 0o600
    info.uid = 0
    info.gid = 0
    info.mtime = 0
    info.size = declared_size
    header = info.tobuf(format=tarfile.GNU_FORMAT)
    if len(header) != tarfile.BLOCKSIZE:
        raise AssertionError("test header unexpectedly expanded")
    return compress_tar(header + (b"\0" * (2 * tarfile.BLOCKSIZE)), archive_format)


def with_nonzero_tar_padding(tar_bytes: bytes) -> bytes:
    changed = bytearray(tar_bytes)
    offset = 0
    while offset + tarfile.BLOCKSIZE <= len(changed):
        header = bytes(changed[offset : offset + tarfile.BLOCKSIZE])
        if header == b"\0" * tarfile.BLOCKSIZE:
            break
        member = tarfile.TarInfo.frombuf(
            header,
            encoding="utf-8",
            errors="surrogateescape",
        )
        data_offset = offset + tarfile.BLOCKSIZE
        padding = (-member.size) % tarfile.BLOCKSIZE
        if padding:
            changed[data_offset + member.size] = 1
            return bytes(changed)
        offset = data_offset + member.size + padding
    raise AssertionError("test tar has no member padding to corrupt")


def b2sum_document(files: dict[str, bytes]) -> bytes:
    return "".join(
        f"{hashlib.blake2b(content).hexdigest()}  {name}\n"
        for name, content in sorted(files.items())
    ).encode("ascii")


def valid_entries(
    *,
    version_text: bytes = b"NanoKVM firmware v1.0.2 test archive\n",
    extra_files: dict[str, bytes] | None = None,
    checksum_override: bytes | None = None,
) -> list[tuple[str, str, bytes | str]]:
    files = {"overlay/boot/ver": version_text}
    if extra_files:
        files.update(extra_files)
    checksums = checksum_override if checksum_override is not None else b2sum_document(files)
    directories = [
        ("overlay", "dir", b""),
        ("overlay/boot", "dir", b""),
    ]
    if any(name.startswith("firmware/") for name in files):
        directories.insert(0, ("firmware", "dir", b""))
    return [
        *directories,
        *((name, "file", content) for name, content in files.items()),
        ("b2sum.txt", "file", checksums),
    ]


class PackageFirmwareTest(unittest.TestCase):
    def test_gzip_metadata_matches_fixed_artifact_vector(self) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            signing_key_id = update_signing.write_keyring(signing_key, keyring)
            public_key = root / "rfc8032-test-only-public.pem"
            subprocess.run(
                [
                    "openssl",
                    "pkey",
                    "-in",
                    str(signing_key),
                    "-pubout",
                    "-out",
                    str(public_key),
                ],
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )

            archive_path = root / vector["name"]
            archive_path.write_bytes(archive_bytes)
            output = root / "output" / "firmware_1.0.2.json"
            result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
            )

            self.assertEqual(archive_path.read_bytes(), archive_bytes)
            metadata_bytes = output.read_bytes()
            metadata = json.loads(metadata_bytes)
            self.assertEqual(
                list(metadata),
                ["version", "name", "sha512", "size", "updateUboot", "signature"],
            )
            self.assertEqual(metadata["version"], "1.0.2")
            self.assertEqual(metadata["name"], vector["name"])
            self.assertEqual(metadata["size"], len(archive_bytes))
            self.assertIs(metadata["updateUboot"], False)
            self.assertEqual(
                metadata["sha512"],
                base64.b64encode(hashlib.sha512(archive_bytes).digest()).decode(
                    "ascii"
                ),
            )
            self.assertEqual(metadata["signature"], vector["signature"])
            self.assertEqual(
                list(metadata["signature"]),
                ["schema", "algorithm", "keyId", "value"],
            )
            self.assertEqual(metadata["signature"]["keyId"], signing_key_id)
            raw_signature = base64.b64decode(
                metadata["signature"]["value"], validate=True
            )
            self.assertEqual(len(raw_signature), update_signing.ED25519_SIGNATURE_SIZE)
            self.assertEqual(
                metadata_bytes,
                (json.dumps(metadata, indent=2, ensure_ascii=False) + "\n").encode(
                    "utf-8"
                ),
            )
            self.assertNotIn(b"\r", metadata_bytes)
            self.assertEqual(output.stat().st_mode & 0o777, 0o644)
            self.assertFalse(Path(f"{output}.sig").exists())
            self.assertEqual(
                {path.name for path in output.parent.iterdir()},
                {output.name},
            )

            signature_path = root / "firmware.signature"
            signature_path.write_bytes(raw_signature)
            self.assertTrue(
                openssl_verifies(public_key, archive_path, signature_path)
            )
            self.assertFalse(
                openssl_verifies(public_key, output, signature_path)
            )

            record = json.loads(result.stdout)
            self.assertEqual(
                list(record),
                [
                    "metadataName",
                    "version",
                    "name",
                    "size",
                    "sha512",
                    "updateUboot",
                    "keyId",
                    "signature",
                ],
            )
            self.assertEqual(record["metadataName"], output.name)
            self.assertEqual(record["version"], metadata["version"])
            self.assertEqual(record["name"], metadata["name"])
            self.assertEqual(record["size"], metadata["size"])
            self.assertEqual(record["sha512"], metadata["sha512"])
            self.assertIs(record["updateUboot"], False)
            self.assertEqual(record["keyId"], metadata["signature"]["keyId"])
            self.assertEqual(record["signature"], metadata["signature"]["value"])

    def test_update_uboot_requires_explicit_opt_in(self) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / vector["name"]
            archive_path.write_bytes(archive_bytes)

            enabled_output = root / "enabled" / "firmware_1.0.2.json"
            enabled_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=enabled_output,
                update_uboot=True,
            )

            enabled_metadata = json.loads(enabled_output.read_bytes())
            self.assertIs(enabled_metadata["updateUboot"], True)
            self.assertIs(json.loads(enabled_result.stdout)["updateUboot"], True)
            self.assertEqual(enabled_metadata["signature"], vector["signature"])

    def test_xz_metadata_signs_exact_final_archive_and_is_deterministic(self) -> None:
        vector = firmware_vector()
        gzip_bytes = base64.b64decode(vector["artifact"], validate=True)
        uncompressed_tar = gzip.decompress(gzip_bytes)
        xz_bytes = compress_tar(uncompressed_tar, "xz")
        self.assertTrue(xz_bytes.startswith(package_firmware.XZ_MAGIC))

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            public_key = root / "rfc8032-test-only-public.pem"
            subprocess.run(
                [
                    "openssl",
                    "pkey",
                    "-in",
                    str(signing_key),
                    "-pubout",
                    "-out",
                    str(public_key),
                ],
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )

            metadata_paths = []
            archive_paths = []
            for index, mtime in enumerate((100, 200)):
                release_dir = root / f"release-{index}"
                release_dir.mkdir()
                archive_path = release_dir / "axera_firmware_v1.0.2.tar.xz"
                archive_path.write_bytes(xz_bytes)
                os.utime(archive_path, (mtime, mtime))
                output = release_dir / "firmware_1.0.2.json"
                run_package_firmware(
                    version="1.0.2",
                    archive=archive_path,
                    signing_key=signing_key,
                    keyring=keyring,
                    output=output,
                )
                archive_paths.append(archive_path)
                metadata_paths.append(output)

            self.assertEqual(
                metadata_paths[0].read_bytes(), metadata_paths[1].read_bytes()
            )
            metadata = json.loads(metadata_paths[0].read_bytes())
            self.assertEqual(metadata["name"], "axera_firmware_v1.0.2.tar.xz")
            self.assertEqual(metadata["size"], len(xz_bytes))
            self.assertEqual(
                metadata["sha512"],
                EXPECTED_XZ_SHA512,
            )
            self.assertEqual(
                EXPECTED_XZ_SHA512,
                base64.b64encode(hashlib.sha512(xz_bytes).digest()).decode("ascii"),
            )
            self.assertEqual(
                metadata["signature"],
                update_signing.sign_artifact(
                    archive_paths[0], signing_key, keyring
                ),
            )
            self.assertNotEqual(
                metadata["signature"]["value"], vector["signature"]["value"]
            )
            self.assertTrue(
                all(path.read_bytes() == xz_bytes for path in archive_paths)
            )

            raw_signature = base64.b64decode(
                metadata["signature"]["value"], validate=True
            )
            signature_path = root / "firmware-xz.signature"
            signature_path.write_bytes(raw_signature)
            gzip_path = root / "same-tar.tar.gz"
            gzip_path.write_bytes(gzip_bytes)
            tar_path = root / "same-content.tar"
            tar_path.write_bytes(uncompressed_tar)
            self.assertTrue(
                openssl_verifies(public_key, archive_paths[0], signature_path)
            )
            self.assertFalse(
                openssl_verifies(public_key, gzip_path, signature_path)
            )
            self.assertFalse(
                openssl_verifies(public_key, tar_path, signature_path)
            )

    def test_accepts_short_paths_with_direct_gnu_tar_headers(self) -> None:
        entries = [
            (f"{name}/" if kind == "dir" else name, kind, value)
            for name, kind, value in valid_entries()
        ]
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            for archive_format in ("gz", "xz"):
                with self.subTest(archive_format=archive_format):
                    release_dir = root / archive_format
                    release_dir.mkdir()
                    suffix = ".tar.gz" if archive_format == "gz" else ".tar.xz"
                    archive_path = release_dir / f"axera_firmware_v1.0.2{suffix}"
                    archive_path.write_bytes(
                        make_archive(
                            entries,
                            archive_format,
                            tar_format=tarfile.GNU_FORMAT,
                        )
                    )
                    output = release_dir / "firmware_1.0.2.json"
                    run_package_firmware(
                        version="1.0.2",
                        archive=archive_path,
                        signing_key=signing_key,
                        keyring=keyring,
                        output=output,
                    )
                    self.assertEqual(
                        json.loads(output.read_bytes())["name"],
                        archive_path.name,
                    )

    def test_rejects_truncated_trailing_and_concatenated_compression_streams(
        self,
    ) -> None:
        vector = firmware_vector()
        gzip_bytes = base64.b64decode(vector["artifact"], validate=True)
        tar_bytes = gzip.decompress(gzip_bytes)
        xz_bytes = compress_tar(tar_bytes, "xz")
        bad_padding_tar = with_nonzero_tar_padding(tar_bytes)
        cases = (
            ("gzip-trailing", "gz", gzip_bytes + b"JUNK"),
            ("gzip-concatenated", "gz", gzip_bytes + gzip_bytes),
            ("gzip-truncated-byte", "gz", gzip_bytes[:-1]),
            ("gzip-truncated-trailer", "gz", gzip_bytes[:-8]),
            ("gzip-hidden-after-tar", "gz", compress_tar(tar_bytes + b"HIDDEN", "gz")),
            ("gzip-second-tar", "gz", compress_tar(tar_bytes + tar_bytes, "gz")),
            ("gzip-unaligned-zero", "gz", compress_tar(tar_bytes + b"\0", "gz")),
            ("gzip-nonzero-padding", "gz", compress_tar(bad_padding_tar, "gz")),
            ("xz-trailing", "xz", xz_bytes + b"JUNK"),
            ("xz-concatenated", "xz", xz_bytes + xz_bytes),
            ("xz-truncated-byte", "xz", xz_bytes[:-1]),
            ("xz-truncated-footer", "xz", xz_bytes[:-12]),
            ("xz-hidden-after-tar", "xz", compress_tar(tar_bytes + b"HIDDEN", "xz")),
            ("xz-second-tar", "xz", compress_tar(tar_bytes + tar_bytes, "xz")),
            ("xz-unaligned-zero", "xz", compress_tar(tar_bytes + b"\0", "xz")),
            ("xz-nonzero-padding", "xz", compress_tar(bad_padding_tar, "xz")),
        )

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)

            for label, archive_format, archive_bytes in cases:
                with self.subTest(case=label):
                    case_dir = root / label
                    case_dir.mkdir()
                    suffix = ".tar.gz" if archive_format == "gz" else ".tar.xz"
                    archive_path = case_dir / f"axera_firmware_v1.0.2{suffix}"
                    archive_path.write_bytes(archive_bytes)
                    output = case_dir / "firmware_1.0.2.json"
                    result = run_package_firmware(
                        version="1.0.2",
                        archive=archive_path,
                        signing_key=signing_key,
                        keyring=keyring,
                        output=output,
                        check=False,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertFalse(output.exists())
                    self.assertEqual(
                        list(case_dir.glob(f".{archive_path.name}.*")),
                        [],
                    )

            for archive_format, archive_bytes in (
                ("gz", gzip_bytes),
                ("xz", xz_bytes),
            ):
                with self.subTest(case=f"{archive_format}-envelope-limit"):
                    suffix = ".tar.gz" if archive_format == "gz" else ".tar.xz"
                    limited_archive = root / f"axera_firmware_v1.0.2{suffix}"
                    limited_archive.write_bytes(archive_bytes)
                    with self.assertRaisesRegex(SystemExit, "decompressed envelope"):
                        package_firmware.validate_firmware_archive(
                            limited_archive,
                            archive_format,
                            "1.0.2",
                            envelope_limit=1024,
                        )
                    self.assertEqual(
                        list(root.glob(f".{limited_archive.name}.tar-*")),
                        [],
                    )

                    trailing_expansion = compress_tar(
                        tar_bytes + (b"x" * 4096),
                        archive_format,
                    )
                    limited_archive.write_bytes(trailing_expansion)
                    with self.assertRaisesRegex(SystemExit, "decompressed envelope"):
                        package_firmware.validate_firmware_archive(
                            limited_archive,
                            archive_format,
                            "1.0.2",
                            envelope_limit=len(tar_bytes) + 2048,
                        )
                    self.assertEqual(
                        list(root.glob(f".{limited_archive.name}.tar-*")),
                        [],
                    )

    def test_rejects_oversized_pax_and_gnu_longname_headers_without_reading_payload(
        self,
    ) -> None:
        declared_size = 1 << 40
        cases = (
            (
                "pax",
                "gz",
                make_header_only_archive(
                    tarfile.XHDTYPE,
                    declared_size,
                    "gz",
                ),
            ),
            (
                "gnu-longname",
                "xz",
                make_header_only_archive(
                    tarfile.GNUTYPE_LONGNAME,
                    declared_size,
                    "xz",
                ),
            ),
        )

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            for label, archive_format, archive_bytes in cases:
                with self.subTest(case=label):
                    case_dir = root / label
                    case_dir.mkdir()
                    suffix = ".tar.gz" if archive_format == "gz" else ".tar.xz"
                    archive_path = case_dir / f"axera_firmware_v1.0.2{suffix}"
                    archive_path.write_bytes(archive_bytes)
                    output = case_dir / "firmware_1.0.2.json"
                    result = run_package_firmware(
                        version="1.0.2",
                        archive=archive_path,
                        signing_key=signing_key,
                        keyring=keyring,
                        output=output,
                        check=False,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(
                        b"unsupported firmware archive member type",
                        result.stderr,
                    )
                    self.assertFalse(output.exists())

    def test_rejects_version_name_magic_and_output_mismatches(self) -> None:
        vector = firmware_vector()
        gzip_bytes = base64.b64decode(vector["artifact"], validate=True)
        xz_bytes = compress_tar(gzip.decompress(gzip_bytes), "xz")
        internal_mismatch = make_archive(
            valid_entries(version_text=b"NanoKVM firmware v1.0.3\n")
        )

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            cases = (
                ("leading-v", "v1.0.2", "axera_firmware_v1.0.2.tar.gz", gzip_bytes, None),
                ("noncanonical", "1.00.2", "axera_firmware_v1.00.2.tar.gz", gzip_bytes, None),
                (
                    "uint64-overflow",
                    "18446744073709551616.0.0",
                    "axera_firmware_v18446744073709551616.0.0.tar.gz",
                    gzip_bytes,
                    None,
                ),
                ("archive-name", "1.0.2", "axera_firmware_v1.0.3.tar.gz", gzip_bytes, None),
                ("internal-version", "1.0.2", "axera_firmware_v1.0.2.tar.gz", internal_mismatch, None),
                ("gzip-as-xz", "1.0.2", "axera_firmware_v1.0.2.tar.xz", gzip_bytes, None),
                ("xz-as-gzip", "1.0.2", "axera_firmware_v1.0.2.tar.gz", xz_bytes, None),
                ("bad-magic", "1.0.2", "axera_firmware_v1.0.2.tar.gz", b"not-an-archive", None),
                ("output-name", "1.0.2", "axera_firmware_v1.0.2.tar.gz", gzip_bytes, "wrong.json"),
            )
            for label, version, archive_name, content, output_name in cases:
                with self.subTest(case=label):
                    case_dir = root / label
                    case_dir.mkdir()
                    archive_path = case_dir / archive_name
                    archive_path.write_bytes(content)
                    before = archive_path.read_bytes()
                    output = case_dir / (output_name or f"firmware_{version}.json")
                    result = run_package_firmware(
                        version=version,
                        archive=archive_path,
                        signing_key=signing_key,
                        keyring=keyring,
                        output=output,
                        check=False,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(archive_path.read_bytes(), before)
                    self.assertFalse(output.exists())

    def test_rejects_uppercase_internal_version_token(self) -> None:
        archive_bytes = make_archive(
            valid_entries(version_text=b"NanoKVM firmware V1.0.2\n")
        )
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            output = root / "firmware_1.0.2.json"

            result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(b"lowercase v", result.stderr)
            self.assertFalse(output.exists())

    def test_rejects_ambiguous_or_invalid_archive_contents(self) -> None:
        good_version = b"NanoKVM firmware v1.0.2\n"
        good_checksum = b2sum_document({"overlay/boot/ver": good_version})
        config = b"config"
        vertical_tab_checksums = (
            hashlib.blake2b(good_version).hexdigest().encode("ascii")
            + b"  overlay/boot/ver\v"
            + hashlib.blake2b(config).hexdigest().encode("ascii")
            + b"  overlay/boot/config\n"
        )
        cases = (
            (
                "missing-version",
                [("b2sum.txt", "file", b"\n")],
                b"missing overlay/boot/ver",
            ),
            (
                "version-symlink",
                [
                    ("overlay", "dir", b""),
                    ("overlay/boot", "dir", b""),
                    ("overlay/boot/ver", "symlink", "target"),
                    ("b2sum.txt", "file", good_checksum),
                ],
                b"unsupported firmware archive member type",
            ),
            (
                "duplicate-version",
                [
                    ("overlay/boot/ver", "file", good_version),
                    ("overlay/boot/ver", "file", good_version),
                    ("b2sum.txt", "file", good_checksum),
                ],
                b"duplicate firmware archive member path",
            ),
            (
                "multiple-version-tokens",
                valid_entries(version_text=b"v1.0.2 and v1.0.3\n"),
                b"exactly one canonical",
            ),
            (
                "uppercase-version-token-hidden-before-lowercase",
                valid_entries(version_text=b"V9.9.9 and v1.0.2\n"),
                b"exactly one canonical",
            ),
            (
                "missing-b2sum",
                [("overlay/boot/ver", "file", good_version)],
                b"missing b2sum.txt",
            ),
            (
                "bad-b2sum",
                valid_entries(
                    version_text=good_version,
                    checksum_override=(b"0" * 128 + b"  overlay/boot/ver\n"),
                ),
                b"checksum mismatch",
            ),
            (
                "incomplete-b2sum",
                valid_entries(
                    version_text=good_version,
                    extra_files={"overlay/boot/configs": b"config"},
                    checksum_override=good_checksum,
                ),
                b"file set mismatch",
            ),
            (
                "non-lf-b2sum-separator",
                valid_entries(
                    version_text=good_version,
                    extra_files={"overlay/boot/config": config},
                    checksum_override=vertical_tab_checksums,
                ),
                b"unsafe firmware archive member path",
            ),
            (
                "unsafe-path",
                [*valid_entries(version_text=good_version), ("../escape", "file", b"x")],
                b"non-canonical firmware archive member path",
            ),
            (
                "payload-hardlink",
                [
                    *valid_entries(version_text=good_version),
                    ("overlay/boot/link", "hardlink", "overlay/boot/ver"),
                ],
                b"unsupported firmware archive member type",
            ),
            (
                "contiguous-file-type",
                [
                    ("overlay", "dir", b""),
                    ("overlay/boot", "dir", b""),
                    ("overlay/boot/ver", "contiguous", good_version),
                ],
                b"unsupported firmware archive member type",
            ),
            (
                "gnu-sparse-file-type",
                [
                    ("overlay", "dir", b""),
                    ("overlay/boot", "dir", b""),
                    ("overlay/boot/ver", "sparse", good_version),
                ],
                b"unsupported firmware archive member type",
            ),
            (
                "pax-metadata-type",
                [("overlay/pax", "pax-header", b"20 path=overlay/file\n")],
                b"unsupported firmware archive member type",
            ),
            (
                "file-parent",
                [
                    ("overlay", "file", b"not-a-directory"),
                    ("overlay/boot/ver", "file", good_version),
                    (
                        "b2sum.txt",
                        "file",
                        b2sum_document(
                            {
                                "overlay": b"not-a-directory",
                                "overlay/boot/ver": good_version,
                            }
                        ),
                    ),
                ],
                b"root must be a directory",
            ),
        )

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            for label, entries, expected_error in cases:
                with self.subTest(case=label):
                    case_dir = root / label
                    case_dir.mkdir()
                    archive_path = case_dir / "axera_firmware_v1.0.2.tar.gz"
                    archive_path.write_bytes(make_archive(entries))
                    output = case_dir / "firmware_1.0.2.json"
                    result = run_package_firmware(
                        version="1.0.2",
                        archive=archive_path,
                        signing_key=signing_key,
                        keyring=keyring,
                        output=output,
                        check=False,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(expected_error, result.stderr)
                    self.assertFalse(output.exists())

    def test_rejects_wrong_keyring_missing_key_and_non_regular_archive(self) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            empty_keyring = root / "empty-keys.json"
            empty_keyring.write_text('{"schema":1,"keys":[]}\n', encoding="utf-8")

            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            output = root / "firmware_1.0.2.json"
            wrong_key_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=empty_keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(wrong_key_result.returncode, 0)
            self.assertIn(b"does not match", wrong_key_result.stderr)
            self.assertFalse(output.exists())

            missing_key_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=None,
                keyring=keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(missing_key_result.returncode, 0)
            self.assertFalse(output.exists())

            target = root / "target.tar.gz"
            target.write_bytes(archive_bytes)
            archive_path.unlink()
            archive_path.symlink_to(target)
            symlink_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(symlink_result.returncode, 0)
            self.assertIn(b"not a regular file", symlink_result.stderr)
            self.assertFalse(output.exists())

    def test_refuses_existing_metadata_and_cleans_obsolete_sidecar_only_with_force(
        self,
    ) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            output = root / "firmware_1.0.2.json"
            output.write_bytes(b"sentinel\n")

            existing_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(existing_result.returncode, 0)
            self.assertEqual(output.read_bytes(), b"sentinel\n")

            output.unlink()
            sidecar = Path(f"{output}.sig")
            sidecar.write_bytes(b"obsolete\n")
            sidecar_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
                check=False,
            )
            self.assertNotEqual(sidecar_result.returncode, 0)
            self.assertTrue(sidecar.exists())
            self.assertFalse(output.exists())

            run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=output,
                force=True,
            )
            self.assertTrue(output.is_file())
            self.assertFalse(sidecar.exists())
            self.assertEqual(
                list(root.glob(f".{output.name}.*")),
                [],
            )

            collision_dir = root / "protected-sidecar"
            collision_dir.mkdir()
            collision_output = collision_dir / "firmware_1.0.2.json"
            collision_keyring = Path(f"{collision_output}.sig")
            update_signing.write_keyring(signing_key, collision_keyring)
            keyring_before = collision_keyring.read_bytes()
            collision_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=collision_keyring,
                output=collision_output,
                force=True,
                check=False,
            )
            self.assertNotEqual(collision_result.returncode, 0)
            self.assertIn(b"collides with keyring", collision_result.stderr)
            self.assertEqual(collision_keyring.read_bytes(), keyring_before)
            self.assertFalse(collision_output.exists())

            fifo_dir = root / "fifo-targets"
            fifo_dir.mkdir()
            fifo_output = fifo_dir / "firmware_1.0.2.json"
            os.mkfifo(fifo_output)
            fifo_output_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=fifo_output,
                force=True,
                check=False,
            )
            self.assertNotEqual(fifo_output_result.returncode, 0)
            self.assertIn(b"non-regular output", fifo_output_result.stderr)
            self.assertTrue(fifo_output.exists())

            fifo_output.unlink()
            fifo_sidecar = Path(f"{fifo_output}.sig")
            os.mkfifo(fifo_sidecar)
            fifo_sidecar_result = run_package_firmware(
                version="1.0.2",
                archive=archive_path,
                signing_key=signing_key,
                keyring=keyring,
                output=fifo_output,
                force=True,
                check=False,
            )
            self.assertNotEqual(fifo_sidecar_result.returncode, 0)
            self.assertIn(b"non-regular obsolete", fifo_sidecar_result.stderr)
            self.assertTrue(fifo_sidecar.exists())
            self.assertFalse(fifo_output.exists())

    def test_snapshot_prevents_same_size_archive_change_from_being_signed(
        self,
    ) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            original_stat = archive_path.stat()
            output = root / "firmware_1.0.2.json"
            signed_paths: list[Path] = []

            def mutate_original_then_sign(
                snapshot_path: Path,
                signing_key_path: Path,
                keyring_path: Path,
            ) -> dict[str, object]:
                signed_paths.append(snapshot_path)
                self.assertNotEqual(snapshot_path, archive_path)
                self.assertEqual(snapshot_path.read_bytes(), archive_bytes)
                changed = bytearray(archive_bytes)
                changed[len(changed) // 2] ^= 1
                archive_path.write_bytes(changed)
                os.utime(
                    archive_path,
                    ns=(original_stat.st_atime_ns, original_stat.st_mtime_ns),
                )
                return update_signing.sign_artifact(
                    snapshot_path,
                    signing_key_path,
                    keyring_path,
                )

            with mock.patch.object(
                package_firmware,
                "sign_artifact",
                side_effect=mutate_original_then_sign,
            ):
                with self.assertRaisesRegex(SystemExit, "archive changed"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                    )

            self.assertEqual(archive_path.stat().st_ino, original_stat.st_ino)
            self.assertEqual(archive_path.stat().st_size, original_stat.st_size)
            self.assertEqual(
                archive_path.stat().st_mtime_ns,
                original_stat.st_mtime_ns,
            )
            self.assertNotEqual(archive_path.read_bytes(), archive_bytes)
            self.assertFalse(output.exists())
            self.assertEqual(len(signed_paths), 1)
            self.assertFalse(signed_paths[0].exists())
            self.assertFalse(signed_paths[0].parent.exists())

            archive_path.write_bytes(archive_bytes)
            staged_race_stat = archive_path.stat()
            real_commit = package_firmware.commit_staged_firmware_metadata

            def mutate_at_commit_then_continue(
                staged_path: Path,
                metadata_path: Path,
                force: bool,
                snapshot: package_firmware.FirmwareArchiveSnapshot,
            ) -> None:
                changed = bytearray(archive_bytes)
                changed[len(changed) // 3] ^= 1
                archive_path.write_bytes(changed)
                os.utime(
                    archive_path,
                    ns=(staged_race_stat.st_atime_ns, staged_race_stat.st_mtime_ns),
                )
                real_commit(staged_path, metadata_path, force, snapshot)

            with mock.patch.object(
                package_firmware,
                "commit_staged_firmware_metadata",
                side_effect=mutate_at_commit_then_continue,
            ):
                with self.assertRaisesRegex(SystemExit, "archive changed"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                    )
            self.assertFalse(output.exists())
            self.assertEqual(list(root.glob(f".{output.name}.*")), [])

    def test_force_keeps_old_sidecar_until_metadata_replace_succeeds(self) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            output = root / "firmware_1.0.2.json"
            sidecar = Path(f"{output}.sig")
            old_metadata = b"old metadata\n"
            old_sidecar = b"old signature\n"
            output.write_bytes(old_metadata)
            sidecar.write_bytes(old_sidecar)

            def fail_metadata_commit(
                staged_path: Path,
                path: Path,
                force: bool,
                snapshot: package_firmware.FirmwareArchiveSnapshot,
            ) -> None:
                self.assertTrue(staged_path.is_file())
                self.assertEqual(path, output)
                self.assertTrue(force)
                self.assertEqual(snapshot.original_path, archive_path)
                self.assertEqual(output.read_bytes(), old_metadata)
                self.assertEqual(sidecar.read_bytes(), old_sidecar)
                raise OSError("simulated metadata replace failure")

            with mock.patch.object(
                package_firmware,
                "commit_staged_firmware_metadata",
                side_effect=fail_metadata_commit,
            ):
                with self.assertRaisesRegex(OSError, "simulated metadata"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                        force=True,
                    )

            self.assertEqual(output.read_bytes(), old_metadata)
            self.assertEqual(sidecar.read_bytes(), old_sidecar)

            real_metadata_commit = package_firmware.commit_staged_firmware_metadata
            observed_sidecar = []

            def observe_metadata_commit(
                staged_path: Path,
                path: Path,
                force: bool,
                snapshot: package_firmware.FirmwareArchiveSnapshot,
            ) -> None:
                observed_sidecar.append(sidecar.read_bytes())
                real_metadata_commit(staged_path, path, force, snapshot)

            with mock.patch.object(
                package_firmware,
                "commit_staged_firmware_metadata",
                side_effect=observe_metadata_commit,
            ):
                package_firmware.write_firmware_metadata(
                    "1.0.2",
                    archive_path,
                    signing_key,
                    keyring,
                    output,
                    force=True,
                )

            self.assertEqual(observed_sidecar, [old_sidecar])
            self.assertNotEqual(output.read_bytes(), old_metadata)
            self.assertFalse(sidecar.exists())

            output.write_bytes(old_metadata)
            sidecar.write_bytes(old_sidecar)
            with mock.patch.object(
                package_firmware,
                "remove_obsolete_firmware_signature",
                side_effect=SystemExit("simulated sidecar unlink failure"),
            ):
                with self.assertRaisesRegex(SystemExit, "sidecar unlink"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                        force=True,
                    )
            self.assertNotEqual(output.read_bytes(), old_metadata)
            self.assertEqual(sidecar.read_bytes(), old_sidecar)

    def test_default_commit_never_clobbers_concurrent_output_and_reports_sidecar(
        self,
    ) -> None:
        vector = firmware_vector()
        archive_bytes = base64.b64decode(vector["artifact"], validate=True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            archive_path = root / "axera_firmware_v1.0.2.tar.gz"
            archive_path.write_bytes(archive_bytes)
            output = root / "firmware_1.0.2.json"
            sidecar = Path(f"{output}.sig")
            real_commit = package_firmware.commit_staged_firmware_metadata
            concurrent_metadata = b"concurrent metadata\n"

            def create_output_then_commit(
                staged_path: Path,
                metadata_path: Path,
                force: bool,
                snapshot: package_firmware.FirmwareArchiveSnapshot,
            ) -> None:
                metadata_path.write_bytes(concurrent_metadata)
                real_commit(staged_path, metadata_path, force, snapshot)

            with mock.patch.object(
                package_firmware,
                "commit_staged_firmware_metadata",
                side_effect=create_output_then_commit,
            ):
                with self.assertRaisesRegex(SystemExit, "already exists"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                    )
            self.assertEqual(output.read_bytes(), concurrent_metadata)
            self.assertFalse(sidecar.exists())
            self.assertEqual(list(root.glob(f".{output.name}.*")), [])

            output.unlink()
            concurrent_sidecar = b"concurrent signature\n"

            def commit_then_create_sidecar(
                staged_path: Path,
                metadata_path: Path,
                force: bool,
                snapshot: package_firmware.FirmwareArchiveSnapshot,
            ) -> None:
                real_commit(staged_path, metadata_path, force, snapshot)
                sidecar.write_bytes(concurrent_sidecar)

            with mock.patch.object(
                package_firmware,
                "commit_staged_firmware_metadata",
                side_effect=commit_then_create_sidecar,
            ):
                with self.assertRaisesRegex(SystemExit, "signature appeared"):
                    package_firmware.write_firmware_metadata(
                        "1.0.2",
                        archive_path,
                        signing_key,
                        keyring,
                        output,
                    )
            self.assertTrue(output.is_file())
            self.assertEqual(sidecar.read_bytes(), concurrent_sidecar)

    def test_atomic_write_does_not_replace_dangling_symlink_without_force(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            output = root / "metadata.json"
            output.symlink_to(root / "missing-target")
            with self.assertRaisesRegex(SystemExit, "already exists"):
                update_signing.atomic_write(output, b"replacement\n", force=False)
            self.assertTrue(output.is_symlink())
            self.assertFalse((root / "missing-target").exists())


if __name__ == "__main__":
    unittest.main()
