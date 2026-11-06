from __future__ import annotations

import base64
import hashlib
import io
import json
import os
import subprocess
import sys
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import update_signing


REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
REPOSITORY_TEMP_PARENT = REPOSITORY_ROOT / "build/test-tmp"
SIGNING_VECTORS = (
    REPOSITORY_ROOT
    / "server/updateverify/testdata/artifact_signing_vectors_v1.json"
)
# RFC 8032 test vector 1. This seed is public test data and must never be used
# for release signing.
TEST_ONLY_ED25519_SEED = bytes.fromhex(
    "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
)
ED25519_PKCS8_SEED_PREFIX = bytes.fromhex("302e020100300506032b657004220420")
PACKAGE_NAMES = ("nanokvm-go", "kvmcomm")


def repository_temporary_directory() -> tempfile.TemporaryDirectory:
    REPOSITORY_TEMP_PARENT.mkdir(parents=True, exist_ok=True)
    return tempfile.TemporaryDirectory(
        prefix=".test-update-signing-", dir=REPOSITORY_TEMP_PARENT
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


def run_package_update(
    version: str,
    input_dir: Path,
    output_dir: Path,
    signing_key: Path | None,
    keyring: Path | None,
    *,
    source_date_epoch: int = 1234567890,
    archive_name: str | None = None,
    latest_name: str | None = None,
    force: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess[bytes]:
    command = [
        sys.executable,
        str(Path(__file__).with_name("package_update.py")),
        "--version",
        version,
        "--input-dir",
        str(input_dir),
        "--output-dir",
        str(output_dir),
    ]
    if force:
        command.append("--force")
    if archive_name is not None:
        command.extend(("--archive-name", archive_name))
    if latest_name is not None:
        command.extend(("--latest-name", latest_name))
    if signing_key is not None:
        command.extend(("--signing-key", str(signing_key)))
    if keyring is not None:
        command.extend(("--keyring", str(keyring)))
    environment = os.environ.copy()
    environment["SOURCE_DATE_EPOCH"] = str(source_date_epoch)
    return subprocess.run(
        command,
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=environment,
    )


def openssl_verifies(
    public_key: Path, artifact: Path, signature: Path
) -> bool:
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


class UpdateSigningTest(unittest.TestCase):
    def test_detached_metadata_signing_cli_is_removed(self) -> None:
        self.assertFalse(hasattr(update_signing, "sign_metadata"))
        command = [sys.executable, str(Path(update_signing.__file__))]
        removed = subprocess.run(
            [*command, "sign"],
            check=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertNotEqual(removed.returncode, 0)
        self.assertIn(b"invalid choice", removed.stderr)

        help_result = subprocess.run(
            [*command, "--help"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertIn(b"{keyring,artifact-signature}", help_result.stdout)
        self.assertNotIn(b"adjacent detached signature", help_result.stdout)

    def test_unsigned_keyring_is_empty(self) -> None:
        with repository_temporary_directory() as directory:
            path = Path(directory) / "keys.json"
            self.assertIsNone(update_signing.write_keyring(None, path))
            self.assertEqual(
                json.loads(path.read_text(encoding="utf-8")),
                {"schema": 1, "keys": []},
            )

    def test_package_update_embeds_deb_signatures_and_leaves_archive_metadata_unsigned(
        self,
    ) -> None:
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
            version = "9.8.7"
            deb_bytes = {
                package: b"!<arch>\n" + f"fixture-{package}\0".encode("ascii")
                for package in PACKAGE_NAMES
            }
            for package, content in deb_bytes.items():
                (root / f"{package}_{version}_armhf.deb").write_bytes(content)

            output = root / "output"
            run_package_update(
                version, root, output, signing_key, keyring
            )

            archive_path = output / f"nanokvm-go_{version}.tar.gz"
            latest_path = output / "latest.json"
            self.assertEqual(
                {path.name for path in output.iterdir()},
                {archive_path.name, latest_path.name},
            )
            self.assertEqual(list(output.rglob("*.sig")), [])

            package_dir = f"nanokvm-go_{version}"
            expected_names = [package_dir]
            for package in PACKAGE_NAMES:
                expected_names.extend(
                    [
                        f"{package_dir}/{package}_{version}_armhf.deb",
                        f"{package_dir}/{package}_{version}.json",
                    ]
                )

            with tarfile.open(archive_path, "r:gz") as archive:
                members = archive.getmembers()
                self.assertEqual([member.name for member in members], expected_names)
                self.assertTrue(members[0].isdir())
                self.assertTrue(all(member.isreg() for member in members[1:]))

                for package in PACKAGE_NAMES:
                    deb_name = f"{package_dir}/{package}_{version}_armhf.deb"
                    metadata_name = f"{package_dir}/{package}_{version}.json"
                    archived_deb = archive.extractfile(deb_name)
                    archived_metadata = archive.extractfile(metadata_name)
                    self.assertIsNotNone(archived_deb)
                    self.assertIsNotNone(archived_metadata)
                    archived_deb_bytes = archived_deb.read()
                    metadata_bytes = archived_metadata.read()
                    self.assertEqual(archived_deb_bytes, deb_bytes[package])

                    metadata = json.loads(metadata_bytes)
                    self.assertEqual(
                        list(metadata),
                        [
                            "version",
                            "name",
                            "sha512",
                            "size",
                            "package",
                            "architecture",
                            "signature",
                        ],
                    )
                    self.assertEqual(metadata["version"], version)
                    self.assertEqual(
                        metadata["name"], f"{package}_{version}_armhf.deb"
                    )
                    self.assertEqual(metadata["size"], len(deb_bytes[package]))
                    self.assertEqual(
                        metadata["sha512"],
                        base64.b64encode(
                            hashlib.sha512(deb_bytes[package]).digest()
                        ).decode("ascii"),
                    )
                    self.assertEqual(metadata["package"], package)
                    self.assertEqual(metadata["architecture"], "armhf")
                    self.assertEqual(
                        list(metadata["signature"]),
                        ["schema", "algorithm", "keyId", "value"],
                    )
                    self.assertEqual(metadata["signature"]["schema"], 1)
                    self.assertEqual(
                        metadata["signature"]["algorithm"], "ed25519"
                    )
                    self.assertEqual(
                        metadata["signature"]["keyId"], signing_key_id
                    )
                    raw_signature = base64.b64decode(
                        metadata["signature"]["value"], validate=True
                    )
                    self.assertEqual(
                        len(raw_signature), update_signing.ED25519_SIGNATURE_SIZE
                    )
                    self.assertEqual(
                        metadata["signature"],
                        update_signing.sign_artifact(
                            root / metadata["name"], signing_key, keyring
                        ),
                    )
                    self.assertEqual(
                        metadata_bytes,
                        (
                            json.dumps(metadata, indent=2, ensure_ascii=False)
                            + "\n"
                        ).encode("utf-8"),
                    )
                    self.assertNotIn(b"\r", metadata_bytes)

                    signature_path = root / f"{package}.signature"
                    signature_path.write_bytes(raw_signature)
                    self.assertTrue(
                        openssl_verifies(
                            public_key, root / metadata["name"], signature_path
                        )
                    )
                    tampered_path = root / f"tampered-{metadata['name']}"
                    tampered_path.write_bytes(deb_bytes[package] + b"tampered")
                    self.assertFalse(
                        openssl_verifies(public_key, tampered_path, signature_path)
                    )

            latest_bytes = latest_path.read_bytes()
            archive_metadata = json.loads(latest_bytes)
            self.assertEqual(
                list(archive_metadata), ["version", "name", "sha512", "size"]
            )
            self.assertEqual(archive_metadata["version"], version)
            self.assertEqual(archive_metadata["name"], archive_path.name)
            self.assertEqual(archive_metadata["size"], archive_path.stat().st_size)
            self.assertEqual(
                archive_metadata["sha512"],
                base64.b64encode(
                    hashlib.sha512(archive_path.read_bytes()).digest()
                ).decode("ascii"),
            )
            self.assertEqual(
                latest_bytes,
                (
                    json.dumps(archive_metadata, indent=2, ensure_ascii=False)
                    + "\n"
                ).encode("utf-8"),
            )
            self.assertNotIn(b"\r", latest_bytes)

    def test_package_update_is_deterministic_for_source_date_epoch(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            version = "9.8.7"
            deb_paths = []
            for package in PACKAGE_NAMES:
                deb_path = root / f"{package}_{version}_armhf.deb"
                deb_path.write_bytes(f"deterministic-{package}".encode("ascii"))
                deb_paths.append(deb_path)

            epoch = 1700000000
            for deb_path in deb_paths:
                os.utime(deb_path, (100, 100))
            first_output = root / "first"
            run_package_update(
                version,
                root,
                first_output,
                signing_key,
                keyring,
                source_date_epoch=epoch,
            )

            for deb_path in deb_paths:
                os.utime(deb_path, (200, 200))
            second_output = root / "second"
            run_package_update(
                version,
                root,
                second_output,
                signing_key,
                keyring,
                source_date_epoch=epoch,
            )

            artifact_names = (
                f"nanokvm-go_{version}.tar.gz",
                "latest.json",
            )
            for artifact_name in artifact_names:
                self.assertEqual(
                    (first_output / artifact_name).read_bytes(),
                    (second_output / artifact_name).read_bytes(),
                )

            archive_path = first_output / artifact_names[0]
            archive_bytes = archive_path.read_bytes()
            self.assertEqual(archive_bytes[:2], b"\x1f\x8b")
            self.assertEqual(int.from_bytes(archive_bytes[4:8], "little"), epoch)
            self.assertEqual(archive_bytes[3] & 0x08, 0)
            with tarfile.open(archive_path, "r:gz") as archive:
                members = archive.getmembers()
            self.assertTrue(all(member.mtime == epoch for member in members))
            self.assertEqual(members[0].mode, 0o755)
            self.assertTrue(all(member.mode == 0o644 for member in members[1:]))
            self.assertTrue(all(member.uid == 0 for member in members))
            self.assertTrue(all(member.gid == 0 for member in members))
            self.assertTrue(all(member.uname == "root" for member in members))
            self.assertTrue(all(member.gname == "root" for member in members))
            self.assertTrue(all(not member.pax_headers for member in members))
            for artifact_name in artifact_names:
                self.assertEqual(
                    (first_output / artifact_name).stat().st_mode & 0o777,
                    0o644,
                )

    def test_package_update_creates_unsigned_bundle_without_signature_fields(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            version = "0.0.25"
            deb_bytes = {}
            for package in PACKAGE_NAMES:
                content = f"unsigned-{package}".encode("ascii")
                deb_bytes[package] = content
                (root / f"{package}_{version}_armhf.deb").write_bytes(content)
            output = root / "output"
            result = run_package_update(version, root, output, None, None)
            self.assertIn(
                b"WARNING: creating an unsigned application update bundle",
                result.stdout,
            )

            archive_path = output / f"nanokvm-go_{version}.tar.gz"
            latest_path = output / "latest.json"
            self.assertEqual(
                {path.name for path in output.iterdir()},
                {archive_path.name, latest_path.name},
            )
            self.assertEqual(list(output.rglob("*.sig")), [])

            package_dir = f"nanokvm-go_{version}"
            with tarfile.open(archive_path, "r:gz") as archive:
                self.assertEqual(
                    archive.getnames(),
                    [
                        package_dir,
                        f"{package_dir}/nanokvm-go_{version}_armhf.deb",
                        f"{package_dir}/nanokvm-go_{version}.json",
                        f"{package_dir}/kvmcomm_{version}_armhf.deb",
                        f"{package_dir}/kvmcomm_{version}.json",
                    ],
                )
                for package in PACKAGE_NAMES:
                    archived_deb = archive.extractfile(
                        f"{package_dir}/{package}_{version}_armhf.deb"
                    )
                    archived_metadata = archive.extractfile(
                        f"{package_dir}/{package}_{version}.json"
                    )
                    self.assertIsNotNone(archived_deb)
                    self.assertIsNotNone(archived_metadata)
                    self.assertEqual(archived_deb.read(), deb_bytes[package])
                    metadata = json.load(archived_metadata)
                    self.assertEqual(
                        list(metadata),
                        [
                            "version",
                            "name",
                            "sha512",
                            "size",
                            "package",
                            "architecture",
                        ],
                    )
                    self.assertNotIn("signature", metadata)
                    self.assertEqual(metadata["version"], version)
                    self.assertEqual(
                        metadata["name"], f"{package}_{version}_armhf.deb"
                    )
                    self.assertEqual(metadata["size"], len(deb_bytes[package]))
                    self.assertEqual(
                        metadata["sha512"],
                        base64.b64encode(
                            hashlib.sha512(deb_bytes[package]).digest()
                        ).decode("ascii"),
                    )
                    self.assertEqual(metadata["package"], package)
                    self.assertEqual(metadata["architecture"], "armhf")

            latest = json.loads(latest_path.read_text(encoding="utf-8"))
            self.assertEqual(list(latest), ["version", "name", "sha512", "size"])
            self.assertEqual(latest["version"], version)
            self.assertEqual(latest["name"], archive_path.name)
            self.assertEqual(latest["size"], archive_path.stat().st_size)
            self.assertEqual(
                latest["sha512"],
                base64.b64encode(hashlib.sha512(archive_path.read_bytes()).digest()).decode(
                    "ascii"
                ),
            )

    def test_package_update_requires_signing_key_and_keyring_together(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            version = "0.0.25"
            for package in PACKAGE_NAMES:
                (root / f"{package}_{version}_armhf.deb").write_bytes(b"deb")

            for supplied_key, supplied_keyring in (
                (signing_key, None),
                (None, keyring),
            ):
                with self.subTest(
                    signing_key=supplied_key, keyring=supplied_keyring
                ):
                    output = root / (
                        "only-key" if supplied_key is not None else "only-keyring"
                    )
                    result = run_package_update(
                        version,
                        root,
                        output,
                        supplied_key,
                        supplied_keyring,
                        check=False,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(
                        b"--signing-key and --keyring must be supplied together",
                        result.stderr,
                    )
                    self.assertFalse(output.exists())

    def test_unsigned_package_update_is_deterministic(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            version = "9.8.7"
            for package in PACKAGE_NAMES:
                (root / f"{package}_{version}_armhf.deb").write_bytes(
                    f"unsigned-deterministic-{package}".encode("ascii")
                )

            first_output = root / "unsigned-first"
            second_output = root / "unsigned-second"
            run_package_update(version, root, first_output, None, None)
            run_package_update(version, root, second_output, None, None)

            for artifact_name in (f"nanokvm-go_{version}.tar.gz", "latest.json"):
                self.assertEqual(
                    (first_output / artifact_name).read_bytes(),
                    (second_output / artifact_name).read_bytes(),
                )

    def test_package_update_force_removes_obsolete_external_metadata(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            version = "9.8.7"
            for package in PACKAGE_NAMES:
                (root / f"{package}_{version}_armhf.deb").write_bytes(
                    f"fixture-{package}".encode("ascii")
                )

            output = root / "output"
            output.mkdir()
            obsolete_paths = (
                output / f"nanokvm-go_{version}.json",
                output / "latest.json.sig",
                output / f"nanokvm-go_{version}.json.sig",
            )
            for path in obsolete_paths:
                path.write_text("obsolete\n", encoding="utf-8")

            blocked = run_package_update(
                version,
                root,
                output,
                signing_key,
                keyring,
                check=False,
            )
            self.assertNotEqual(blocked.returncode, 0)
            self.assertIn(
                b"obsolete external application metadata exists", blocked.stderr
            )
            self.assertTrue(all(path.exists() for path in obsolete_paths))

            run_package_update(
                version,
                root,
                output,
                signing_key,
                keyring,
                force=True,
            )

            self.assertTrue(all(not path.exists() for path in obsolete_paths))
            self.assertEqual(list(output.rglob("*.sig")), [])
            self.assertEqual(
                {path.name for path in output.iterdir()},
                {
                    f"nanokvm-go_{version}.tar.gz",
                    "latest.json",
                },
            )

    def test_package_update_rejects_unsafe_or_colliding_output_names(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            update_signing.write_keyring(signing_key, keyring)
            version = "9.8.7"
            for package in PACKAGE_NAMES:
                (root / f"{package}_{version}_armhf.deb").write_bytes(
                    f"fixture-{package}".encode("ascii")
                )

            cases = (
                {"archive_name": "../escape.tar.gz"},
                {"latest_name": "../escape.json"},
                {"latest_name": f"nanokvm-go_{version}.json"},
            )
            for index, options in enumerate(cases):
                with self.subTest(options=options):
                    output = root / f"output-{index}"
                    result = run_package_update(
                        version,
                        root,
                        output,
                        signing_key,
                        keyring,
                        check=False,
                        **options,
                    )
                    self.assertNotEqual(result.returncode, 0)
                    self.assertFalse(output.exists())

            result = run_package_update(
                version,
                root,
                root,
                signing_key,
                keyring,
                latest_name=keyring.name,
                check=False,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(b"collides with keyring", result.stderr)

    def test_package_update_preserves_credentials_that_collide_with_cleanup_paths(
        self,
    ) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            version = "9.8.7"
            for package in PACKAGE_NAMES:
                (root / f"{package}_{version}_armhf.deb").write_bytes(
                    f"fixture-{package}".encode("ascii")
                )

            cases = (
                ("signing-key-latest-sidecar", "signing-key", "latest.json.sig"),
                (
                    "keyring-versioned-sidecar",
                    "keyring",
                    f"nanokvm-go_{version}.json.sig",
                ),
            )
            for case_name, credential_kind, relative_path in cases:
                with self.subTest(case=case_name):
                    case_root = root / case_name
                    output = case_root / "output"
                    credential_path = output / relative_path
                    credential_path.parent.mkdir(parents=True)
                    normal_signing_key = case_root / "signing.pem"
                    normal_signing_key.parent.mkdir(parents=True, exist_ok=True)
                    write_fixed_test_key(normal_signing_key)
                    normal_keyring = case_root / "keys.json"
                    update_signing.write_keyring(normal_signing_key, normal_keyring)

                    if credential_kind == "signing-key":
                        write_fixed_test_key(credential_path)
                        signing_key = credential_path
                        keyring = normal_keyring
                    else:
                        credential_path.write_bytes(normal_keyring.read_bytes())
                        signing_key = normal_signing_key
                        keyring = credential_path
                    credential_bytes = credential_path.read_bytes()

                    result = run_package_update(
                        version,
                        root,
                        output,
                        signing_key,
                        keyring,
                        force=True,
                        check=False,
                    )

                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(
                        b"collides with ",
                        result.stderr,
                    )
                    self.assertEqual(credential_path.read_bytes(), credential_bytes)

    def test_fixed_artifact_signing_vectors(self) -> None:
        fixture = json.loads(SIGNING_VECTORS.read_text(encoding="utf-8"))
        self.assertEqual(fixture["schema"], 1)
        self.assertIs(fixture["testOnly"], True)

        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            keyring = root / "keys.json"
            signing_key_id = update_signing.write_keyring(signing_key, keyring)
            self.assertEqual(signing_key_id, fixture["keyId"])
            self.assertEqual(
                base64.b64encode(
                    update_signing.public_key_from_private(signing_key)
                ).decode("ascii"),
                fixture["publicKey"],
            )

            seen_kinds: set[str] = set()
            for vector in fixture["vectors"]:
                seen_kinds.add(vector["kind"])
                artifact_bytes = base64.b64decode(
                    vector["artifact"], validate=True
                )
                artifact_path = root / vector["name"]
                artifact_path.write_bytes(artifact_bytes)

                actual = update_signing.sign_artifact(
                    artifact_path, signing_key, keyring
                )
                self.assertEqual(actual, vector["signature"])
                self.assertEqual(
                    update_signing.sign_artifact(
                        artifact_path, signing_key, keyring
                    ),
                    actual,
                )
                self.assertEqual(
                    len(base64.b64decode(actual["value"], validate=True)),
                    update_signing.ED25519_SIGNATURE_SIZE,
                )

                tampered = bytearray(artifact_bytes)
                tampered[-1] ^= 1
                tampered_path = root / f"tampered-{vector['name']}"
                tampered_path.write_bytes(tampered)
                self.assertNotEqual(
                    update_signing.sign_artifact(
                        tampered_path, signing_key, keyring
                    )["value"],
                    actual["value"],
                )

                if vector["kind"] == "deb":
                    cli = subprocess.run(
                        [
                            sys.executable,
                            str(Path(update_signing.__file__)),
                            "artifact-signature",
                            "--signing-key",
                            str(signing_key),
                            "--keyring",
                            str(keyring),
                            str(artifact_path),
                        ],
                        check=True,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                    )
                    self.assertEqual(json.loads(cli.stdout), actual)
                elif vector["kind"] == "firmware":
                    with tarfile.open(
                        fileobj=io.BytesIO(artifact_bytes), mode="r:gz"
                    ) as archive:
                        self.assertEqual(
                            archive.getnames(),
                            [
                                "overlay",
                                "overlay/boot",
                                "overlay/boot/ver",
                                "b2sum.txt",
                            ],
                        )
                        version_file = archive.extractfile("overlay/boot/ver")
                        checksum_file = archive.extractfile("b2sum.txt")
                        self.assertIsNotNone(version_file)
                        self.assertIsNotNone(checksum_file)
                        version_bytes = version_file.read()
                        expected_checksum = (
                            hashlib.blake2b(version_bytes).hexdigest()
                            + "  overlay/boot/ver\n"
                        ).encode("ascii")
                        self.assertEqual(checksum_file.read(), expected_checksum)

            self.assertEqual(seen_kinds, {"deb", "firmware"})

    def test_artifact_signer_rejects_non_regular_and_invalid_length(self) -> None:
        with repository_temporary_directory() as directory:
            root = Path(directory)
            signing_key = root / "rfc8032-test-only.pem"
            write_fixed_test_key(signing_key)
            artifact = root / "artifact.bin"
            artifact.write_bytes(b"artifact")

            with self.assertRaisesRegex(SystemExit, "does not exist"):
                update_signing.sign_exact_file(root / "missing", signing_key)
            with self.assertRaisesRegex(SystemExit, "not a regular file"):
                update_signing.sign_exact_file(root, signing_key)
            symlink = root / "artifact-link"
            symlink.symlink_to(artifact)
            with self.assertRaisesRegex(SystemExit, "not a regular file"):
                update_signing.sign_exact_file(symlink, signing_key)

            for length in (63, 65):
                with self.subTest(signature_length=length):
                    result = subprocess.CompletedProcess(
                        args=[], returncode=0, stdout=b"x" * length, stderr=b""
                    )
                    with mock.patch.object(
                        update_signing, "run_openssl", return_value=result
                    ):
                        with self.assertRaisesRegex(SystemExit, "signature length"):
                            update_signing.sign_exact_file(artifact, signing_key)

if __name__ == "__main__":
    unittest.main()
