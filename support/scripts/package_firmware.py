#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import contextlib
import hashlib
import hmac
import json
import lzma
import os
import re
import shutil
import stat
import tarfile
import tempfile
import zlib
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import BinaryIO, Dict, Iterator

from update_signing import FILE_MODE, ensure_keyring_matches, sign_artifact


GZIP_MAGIC = b"\x1f\x8b"
XZ_MAGIC = b"\xfd7zXZ\x00"
MAX_ARCHIVE_ENTRIES = 4096
MAX_ARCHIVE_FILE_BYTES = 256 << 20
MAX_ARCHIVE_EXPANDED_BYTES = 512 << 20
MAX_ARCHIVE_ENVELOPE_BYTES = MAX_ARCHIVE_EXPANDED_BYTES + (8 << 20)
# Extended PAX/GNU path metadata is intentionally rejected below. A direct
# ustar header can represent at most a 155-byte prefix, slash, and 100-byte name.
MAX_ARCHIVE_PATH_BYTES = 256
MAX_ARCHIVE_PATH_COMPONENT_BYTES = 255
MAX_VERSION_FILE_BYTES = 4096
MAX_B2SUM_FILE_BYTES = 1 << 20
MAX_UINT64 = (1 << 64) - 1
MAX_XZ_MEMORY_BYTES = 128 << 20
IO_CHUNK_BYTES = 1024 * 1024
TAR_BLOCK_BYTES = 512
ARCHIVE_FORMATS = {
    "gz": ".tar.gz",
    "xz": ".tar.xz",
}
VERSION_TEXT = r"(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)"
VERSION_PATTERN = re.compile(rf"^{VERSION_TEXT}$")
VERSION_TOKEN_PATTERN = re.compile(
    rf"(?<![0-9A-Za-z])([vV])({VERSION_TEXT})(?![0-9A-Za-z.+-])"
)
B2SUM_LINE_PATTERN = re.compile(r"^([0-9a-f]{128})  (.+)$")
REQUIRED_VERSION_PATH = "overlay/boot/ver"
REQUIRED_B2SUM_PATH = "b2sum.txt"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Generate deterministic metadata with an embedded Ed25519 signature "
            "for one final NanoKVM Firmware archive. Archive members must use "
            "direct ustar-compatible headers; PAX/GNU extensions are rejected."
        )
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Canonical Firmware version without a leading v, for example 1.0.2.",
    )
    parser.add_argument(
        "--archive",
        type=Path,
        required=True,
        help=(
            "Final axera_firmware_v<version>.tar.gz artifact. The legacy "
            ".tar.xz packaging parser remains test-only and is not accepted "
            "by the release verifier."
        ),
    )
    parser.add_argument(
        "--signing-key",
        type=Path,
        required=True,
        help="Offline Ed25519 private key used only to sign the final archive bytes.",
    )
    parser.add_argument(
        "--keyring",
        type=Path,
        required=True,
        help="Packaged public key ring, which must contain the signing key.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        required=True,
        help="Output path, whose basename must be firmware_<version>.json.",
    )
    parser.add_argument(
        "--update-uboot",
        action="store_true",
        help=(
            "Set updateUboot=true in the metadata. The default is false; "
            "this policy field is not covered by the archive signature."
        ),
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Replace existing metadata and remove its obsolete .json.sig sidecar.",
    )
    return parser.parse_args()


def absolute_without_resolving(path: Path) -> Path:
    if path.is_absolute():
        return path
    return Path.cwd() / path


def validate_version(value: str) -> str:
    if not VERSION_PATTERN.fullmatch(value):
        raise SystemExit(
            "version must be three canonical decimal components without a leading v"
        )
    if any(int(component) > MAX_UINT64 for component in value.split(".")):
        raise SystemExit("version component exceeds the unsigned 64-bit range")
    return value


def require_regular_archive(path: Path) -> os.stat_result:
    try:
        archive_stat = path.lstat()
    except OSError as exc:
        raise SystemExit(f"firmware archive does not exist: {path}") from exc
    if not stat.S_ISREG(archive_stat.st_mode):
        raise SystemExit(f"firmware archive is not a regular file: {path}")
    if archive_stat.st_size <= 0:
        raise SystemExit(f"firmware archive is empty: {path}")
    return archive_stat


def expected_archive_format(path: Path, version: str) -> str:
    base_name = f"axera_firmware_v{version}"
    for archive_format, suffix in ARCHIVE_FORMATS.items():
        if path.name == f"{base_name}{suffix}":
            return archive_format
    raise SystemExit(
        "firmware archive name must be exactly "
        f"{base_name}.tar.gz or {base_name}.tar.xz"
    )


def detect_archive_format(path: Path) -> str:
    try:
        with path.open("rb") as archive:
            magic = archive.read(len(XZ_MAGIC))
    except OSError as exc:
        raise SystemExit(f"failed to read firmware archive: {path}") from exc
    if magic.startswith(GZIP_MAGIC):
        return "gz"
    if magic.startswith(XZ_MAGIC):
        return "xz"
    raise SystemExit("firmware archive has unsupported compression magic")


def ensure_output_is_safe(
    output: Path,
    version: str,
    archive: Path,
    signing_key: Path,
    keyring: Path,
    force: bool,
) -> Path:
    expected_name = f"firmware_{version}.json"
    if output.name != expected_name:
        raise SystemExit(f"output basename must be exactly {expected_name}")

    try:
        output_resolved = output.resolve()
        protected_paths = {
            archive.resolve(): "firmware archive",
            signing_key.resolve(): "signing key",
            keyring.resolve(): "keyring",
        }
    except OSError as exc:
        raise SystemExit("failed to resolve Firmware packaging paths") from exc
    protected_label = protected_paths.get(output_resolved)
    if protected_label is not None:
        raise SystemExit(f"output path collides with {protected_label}")

    if output.exists() or output.is_symlink():
        if not force:
            raise SystemExit(f"{output} already exists; pass --force to overwrite")
        output_mode = output.lstat().st_mode
        if not (stat.S_ISREG(output_mode) or stat.S_ISLNK(output_mode)):
            raise SystemExit(
                f"refusing to replace non-regular output path: {output}"
            )

    obsolete_signature = Path(f"{output}.sig")
    try:
        obsolete_resolved = obsolete_signature.resolve()
    except OSError as exc:
        raise SystemExit(
            f"failed to resolve obsolete Firmware signature: {obsolete_signature}"
        ) from exc
    protected_label = protected_paths.get(obsolete_resolved)
    if protected_label is not None:
        raise SystemExit(
            "obsolete Firmware signature path collides with "
            f"{protected_label}: {obsolete_signature}"
        )
    if obsolete_signature.exists() or obsolete_signature.is_symlink():
        if not force:
            raise SystemExit(
                f"obsolete Firmware signature exists: {obsolete_signature}; "
                "pass --force to remove it"
            )
        obsolete_mode = obsolete_signature.lstat().st_mode
        if not (stat.S_ISREG(obsolete_mode) or stat.S_ISLNK(obsolete_mode)):
            raise SystemExit(
                "refusing to remove non-regular obsolete Firmware signature: "
                f"{obsolete_signature}"
            )
    return obsolete_signature


def validate_archive_member_name(
    name: str,
    *,
    directory: bool = False,
) -> PurePosixPath:
    if (
        not name
        or len(name.encode("utf-8", errors="surrogateescape"))
        > MAX_ARCHIVE_PATH_BYTES
        or name.startswith("/")
        or "\\" in name
        or "\0" in name
        or any(ord(character) < 32 or ord(character) == 127 for character in name)
    ):
        raise SystemExit(f"unsafe firmware archive member path: {name!r}")

    canonical_name = name[:-1] if directory and name.endswith("/") else name
    path = PurePosixPath(canonical_name)
    if not canonical_name or path.is_absolute() or str(path) != canonical_name or any(
        component in ("", ".", "..") for component in path.parts
    ):
        raise SystemExit(f"non-canonical firmware archive member path: {name!r}")
    if any(
        len(component.encode("utf-8", errors="surrogateescape"))
        > MAX_ARCHIVE_PATH_COMPONENT_BYTES
        for component in path.parts
    ):
        raise SystemExit(f"firmware archive path component is too long: {name!r}")
    if (
        canonical_name != REQUIRED_B2SUM_PATH
        and path.parts[0] not in ("firmware", "overlay")
    ):
        raise SystemExit(f"unsupported top-level firmware archive path: {name!r}")
    return path


def read_bounded_file(
    file: BinaryIO, size: int, limit: int, label: str
) -> bytes:
    if size > limit:
        raise SystemExit(f"{label} exceeds the {limit}-byte limit")
    remaining = size
    chunks = []
    while remaining:
        chunk = file.read(min(remaining, IO_CHUNK_BYTES))
        if not chunk:
            raise SystemExit(f"failed to read bounded {label}")
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def blake2b_stream(file: BinaryIO, expected_size: int) -> str:
    hasher = hashlib.blake2b()
    remaining = expected_size
    while remaining:
        chunk = file.read(min(remaining, IO_CHUNK_BYTES))
        if not chunk:
            raise SystemExit("firmware archive member is truncated")
        hasher.update(chunk)
        remaining -= len(chunk)
    return hasher.hexdigest()


def parse_b2sum(data: bytes) -> Dict[str, str]:
    try:
        text = data.decode("ascii")
    except UnicodeDecodeError as exc:
        raise SystemExit("b2sum.txt must be ASCII") from exc
    if not text or not text.endswith("\n"):
        raise SystemExit("b2sum.txt must be non-empty and end with a newline")

    checksums: Dict[str, str] = {}
    for line in text[:-1].split("\n"):
        if not line:
            raise SystemExit("b2sum.txt contains an empty checksum line")
        match = B2SUM_LINE_PATTERN.fullmatch(line)
        if match is None:
            raise SystemExit("b2sum.txt contains an invalid checksum line")
        digest, name = match.groups()
        validate_archive_member_name(name)
        if name == REQUIRED_B2SUM_PATH:
            raise SystemExit("b2sum.txt must not checksum itself")
        if name in checksums:
            raise SystemExit(f"b2sum.txt contains a duplicate path: {name}")
        checksums[name] = digest
    return checksums


def validate_internal_version(data: bytes, expected_version: str) -> None:
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise SystemExit("overlay/boot/ver must be UTF-8 text") from exc
    versions = VERSION_TOKEN_PATTERN.findall(text)
    if len(versions) != 1:
        raise SystemExit(
            "overlay/boot/ver must contain exactly one canonical vX.Y.Z token"
        )
    prefix, internal_version = versions[0]
    if prefix != "v":
        raise SystemExit("overlay/boot/ver version token must use a lowercase v")
    if internal_version != expected_version:
        raise SystemExit(
            "firmware archive internal version does not match requested version: "
            f"{internal_version} != {expected_version}"
        )


def write_decompressed_bytes(
    output: BinaryIO,
    data: bytes,
    total: int,
    limit: int,
) -> int:
    if len(data) > limit - total:
        raise SystemExit(
            f"firmware archive decompressed envelope exceeds {limit} bytes"
        )
    output.write(data)
    return total + len(data)


def decompress_gzip_stream(
    source: BinaryIO, output: BinaryIO, envelope_limit: int
) -> int:
    decompressor = zlib.decompressobj(16 + zlib.MAX_WBITS)
    total = 0
    try:
        while not decompressor.eof:
            pending = source.read(IO_CHUNK_BYTES)
            if not pending:
                raise SystemExit("firmware gzip stream is truncated")
            while pending and not decompressor.eof:
                output_limit = min(IO_CHUNK_BYTES, envelope_limit - total + 1)
                data = decompressor.decompress(pending, output_limit)
                total = write_decompressed_bytes(
                    output, data, total, envelope_limit
                )
                pending = decompressor.unconsumed_tail
            if decompressor.unused_data:
                raise SystemExit(
                    "firmware gzip archive contains trailing bytes or multiple streams"
                )
    except zlib.error as exc:
        raise SystemExit(f"invalid firmware gzip stream: {exc}") from exc

    if source.read(1):
        raise SystemExit(
            "firmware gzip archive contains trailing bytes or multiple streams"
        )
    return total


def decompress_xz_stream(
    source: BinaryIO, output: BinaryIO, envelope_limit: int
) -> int:
    decompressor = lzma.LZMADecompressor(
        format=lzma.FORMAT_XZ,
        memlimit=MAX_XZ_MEMORY_BYTES,
    )
    total = 0
    try:
        while not decompressor.eof:
            compressed = (
                source.read(IO_CHUNK_BYTES) if decompressor.needs_input else b""
            )
            if decompressor.needs_input and not compressed:
                raise SystemExit("firmware xz stream is truncated")
            output_limit = min(IO_CHUNK_BYTES, envelope_limit - total + 1)
            data = decompressor.decompress(compressed, max_length=output_limit)
            total = write_decompressed_bytes(output, data, total, envelope_limit)
            if (
                not data
                and not decompressor.needs_input
                and not decompressor.eof
            ):
                raise SystemExit("firmware xz decompressor made no progress")
    except lzma.LZMAError as exc:
        raise SystemExit(f"invalid firmware xz stream: {exc}") from exc

    if decompressor.unused_data or source.read(1):
        raise SystemExit(
            "firmware xz archive contains trailing bytes or multiple streams"
        )
    return total


def decompress_archive_to_tar(
    archive_path: Path,
    archive_format: str,
    tar_path: Path,
    envelope_limit: int,
) -> int:
    if envelope_limit <= 0:
        raise SystemExit("firmware archive decompressed envelope limit is invalid")
    try:
        with archive_path.open("rb") as source, tar_path.open("xb") as output:
            if archive_format == "gz":
                total = decompress_gzip_stream(source, output, envelope_limit)
            elif archive_format == "xz":
                total = decompress_xz_stream(source, output, envelope_limit)
            else:
                raise SystemExit(
                    f"unsupported firmware archive format: {archive_format}"
                )
            output.flush()
            os.fsync(output.fileno())
        tar_path.chmod(0o400)
        return total
    except SystemExit:
        raise
    except OSError as exc:
        raise SystemExit(f"failed to decompress Firmware archive: {exc}") from exc


def read_tar_block(stream: BinaryIO, label: str) -> bytes:
    block = stream.read(TAR_BLOCK_BYTES)
    if len(block) != TAR_BLOCK_BYTES:
        raise SystemExit(f"Firmware tar archive has a truncated {label}")
    return block


def discard_tar_padding(stream: BinaryIO, size: int) -> None:
    padding = (-size) % TAR_BLOCK_BYTES
    if padding:
        data = stream.read(padding)
        if len(data) != padding:
            raise SystemExit("Firmware tar archive has truncated member padding")
        if any(data):
            raise SystemExit("Firmware tar archive has non-zero member padding")


def validate_tar_end_padding(stream: BinaryIO) -> None:
    trailing_size = 0
    while True:
        data = stream.read(IO_CHUNK_BYTES)
        if not data:
            break
        trailing_size += len(data)
        if any(data):
            raise SystemExit(
                "Firmware tar archive contains non-zero data after its end marker"
            )
    if trailing_size % TAR_BLOCK_BYTES != 0:
        raise SystemExit("Firmware tar archive has unaligned end padding")


def validate_uncompressed_tar(tar_path: Path, expected_version: str) -> None:
    seen = set()
    member_kinds: Dict[str, str] = {}
    actual_checksums: Dict[str, str] = {}
    version_data: bytes | None = None
    b2sum_data: bytes | None = None
    entry_count = 0
    expanded_size = 0

    try:
        with tar_path.open("rb") as archive:
            while True:
                header = read_tar_block(archive, "header")
                if header == b"\0" * TAR_BLOCK_BYTES:
                    if read_tar_block(archive, "end-of-archive marker") != (
                        b"\0" * TAR_BLOCK_BYTES
                    ):
                        raise SystemExit(
                            "Firmware tar archive has an invalid end-of-archive marker"
                        )
                    validate_tar_end_padding(archive)
                    break

                entry_count += 1
                if entry_count > MAX_ARCHIVE_ENTRIES:
                    raise SystemExit("firmware archive entry count limit exceeded")
                member = tarfile.TarInfo.frombuf(
                    header,
                    encoding="utf-8",
                    errors="surrogateescape",
                )
                header_name = member.name
                if member.type == tarfile.DIRTYPE:
                    kind = "directory"
                elif member.type in (tarfile.REGTYPE, tarfile.AREGTYPE):
                    kind = "file"
                else:
                    if member.type in (
                        tarfile.XHDTYPE,
                        tarfile.XGLTYPE,
                        tarfile.GNUTYPE_LONGNAME,
                        tarfile.GNUTYPE_LONGLINK,
                        tarfile.GNUTYPE_SPARSE,
                    ):
                        raise SystemExit(
                            "unsupported firmware archive member type: extended "
                            "PAX/GNU metadata is not allowed; use direct ustar headers: "
                            f"{header_name}"
                        )
                    raise SystemExit(
                        f"unsupported firmware archive member type: {header_name}"
                    )

                name = str(
                    validate_archive_member_name(
                        header_name,
                        directory=kind == "directory",
                    )
                )
                if name in seen:
                    raise SystemExit(f"duplicate firmware archive member path: {name}")
                seen.add(name)

                if member.size < 0 or member.size > MAX_ARCHIVE_FILE_BYTES:
                    raise SystemExit(
                        f"firmware archive member size limit exceeded: {name}"
                    )
                if member.size > MAX_ARCHIVE_EXPANDED_BYTES - expanded_size:
                    raise SystemExit("firmware archive expanded size limit exceeded")
                expanded_size += member.size

                if kind == "directory":
                    if member.size != 0:
                        raise SystemExit(
                            f"firmware archive directory has non-zero size: {name}"
                        )
                    member_kinds[name] = "directory"
                    continue
                member_kinds[name] = "file"

                if name == REQUIRED_VERSION_PATH:
                    version_data = read_bounded_file(
                        archive,
                        member.size,
                        MAX_VERSION_FILE_BYTES,
                        REQUIRED_VERSION_PATH,
                    )
                    actual_checksums[name] = hashlib.blake2b(version_data).hexdigest()
                elif name == REQUIRED_B2SUM_PATH:
                    b2sum_data = read_bounded_file(
                        archive,
                        member.size,
                        MAX_B2SUM_FILE_BYTES,
                        REQUIRED_B2SUM_PATH,
                    )
                else:
                    actual_checksums[name] = blake2b_stream(archive, member.size)
                discard_tar_padding(archive, member.size)
    except SystemExit:
        raise
    except (OSError, tarfile.TarError, EOFError) as exc:
        raise SystemExit(f"failed to read Firmware tar archive: {exc}") from exc

    for name, kind in member_kinds.items():
        if name in ("firmware", "overlay") and kind != "directory":
            raise SystemExit(f"firmware archive root must be a directory: {name}")
        parts = PurePosixPath(name).parts
        for length in range(1, len(parts)):
            parent = "/".join(parts[:length])
            if member_kinds.get(parent) == "file":
                raise SystemExit(
                    "firmware archive file conflicts with descendant path: "
                    f"{parent} and {name}"
                )

    if version_data is None:
        raise SystemExit(f"firmware archive is missing {REQUIRED_VERSION_PATH}")
    if b2sum_data is None:
        raise SystemExit(f"firmware archive is missing {REQUIRED_B2SUM_PATH}")
    validate_internal_version(version_data, expected_version)

    declared_checksums = parse_b2sum(b2sum_data)
    missing = sorted(set(actual_checksums) - set(declared_checksums))
    unknown = sorted(set(declared_checksums) - set(actual_checksums))
    if missing or unknown:
        details = []
        if missing:
            details.append(f"missing entries: {', '.join(missing)}")
        if unknown:
            details.append(f"unknown entries: {', '.join(unknown)}")
        raise SystemExit("b2sum.txt file set mismatch (" + "; ".join(details) + ")")
    for name, actual_digest in actual_checksums.items():
        if declared_checksums[name] != actual_digest:
            raise SystemExit(f"b2sum.txt checksum mismatch: {name}")


def validate_firmware_archive(
    archive_path: Path,
    archive_format: str,
    expected_version: str,
    *,
    envelope_limit: int = MAX_ARCHIVE_ENVELOPE_BYTES,
) -> None:
    private_directory = Path(
        tempfile.mkdtemp(
            prefix=f".{archive_path.name}.tar-",
            dir=archive_path.parent,
        )
    )
    private_directory.chmod(0o700)
    tar_path = private_directory / "archive.tar"
    try:
        decompress_archive_to_tar(
            archive_path,
            archive_format,
            tar_path,
            envelope_limit,
        )
        validate_uncompressed_tar(tar_path, expected_version)
    finally:
        shutil.rmtree(private_directory, ignore_errors=True)


def stat_identity(archive_stat: os.stat_result) -> tuple[int, int, int, int, int]:
    return (
        archive_stat.st_dev,
        archive_stat.st_ino,
        archive_stat.st_size,
        archive_stat.st_mtime_ns,
        archive_stat.st_ctime_ns,
    )


@dataclass(frozen=True)
class FirmwareArchiveSnapshot:
    original_path: Path
    source_fd: int
    initial_stat: os.stat_result
    path: Path
    size: int
    sha512_digest: bytes


def hash_open_file_descriptor(descriptor: int) -> tuple[int, bytes]:
    hasher = hashlib.sha512()
    total = 0
    os.lseek(descriptor, 0, os.SEEK_SET)
    while True:
        chunk = os.read(descriptor, IO_CHUNK_BYTES)
        if not chunk:
            break
        total += len(chunk)
        hasher.update(chunk)
    return total, hasher.digest()


def verify_original_archive(snapshot: FirmwareArchiveSnapshot) -> None:
    try:
        before = os.fstat(snapshot.source_fd)
        path_before = snapshot.original_path.lstat()
        total, digest = hash_open_file_descriptor(snapshot.source_fd)
        after = os.fstat(snapshot.source_fd)
        path_after = snapshot.original_path.lstat()
    except OSError as exc:
        raise SystemExit(
            "firmware archive changed while metadata was generated"
        ) from exc

    identity = stat_identity(snapshot.initial_stat)
    if (
        stat_identity(before) != identity
        or stat_identity(after) != identity
        or stat_identity(path_before) != identity
        or stat_identity(path_after) != identity
        or not stat.S_ISREG(path_after.st_mode)
        or total != snapshot.size
        or not hmac.compare_digest(digest, snapshot.sha512_digest)
    ):
        raise SystemExit("firmware archive changed while metadata was generated")


@contextlib.contextmanager
def create_firmware_archive_snapshot(
    archive_path: Path,
    temporary_parent: Path | None = None,
) -> Iterator[FirmwareArchiveSnapshot]:
    require_regular_archive(archive_path)
    source_fd = -1
    private_directory: Path | None = None
    body_started = False
    try:
        flags = os.O_RDONLY | os.O_CLOEXEC
        if not hasattr(os, "O_NOFOLLOW"):
            raise SystemExit("this platform cannot safely snapshot a firmware archive")
        flags |= os.O_NOFOLLOW
        source_fd = os.open(archive_path, flags)
        initial_stat = os.fstat(source_fd)
        path_stat = archive_path.lstat()
        if (
            not stat.S_ISREG(initial_stat.st_mode)
            or initial_stat.st_size <= 0
            or stat_identity(path_stat) != stat_identity(initial_stat)
        ):
            raise SystemExit(
                f"firmware archive is not a stable regular file: {archive_path}"
            )

        private_directory = Path(
            tempfile.mkdtemp(
                prefix=f".{archive_path.name}.snapshot-",
                dir=temporary_parent or archive_path.parent,
            )
        )
        private_directory.chmod(0o700)
        snapshot_path = private_directory / "archive"
        snapshot_descriptor = os.open(
            snapshot_path,
            os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_CLOEXEC,
            0o600,
        )
        hasher = hashlib.sha512()
        total = 0
        os.lseek(source_fd, 0, os.SEEK_SET)
        with os.fdopen(snapshot_descriptor, "wb") as output:
            while True:
                chunk = os.read(source_fd, IO_CHUNK_BYTES)
                if not chunk:
                    break
                output.write(chunk)
                hasher.update(chunk)
                total += len(chunk)
            output.flush()
            os.fsync(output.fileno())
        snapshot_path.chmod(0o400)
        if total != initial_stat.st_size:
            raise SystemExit("firmware archive changed while it was snapshotted")

        snapshot = FirmwareArchiveSnapshot(
            original_path=archive_path,
            source_fd=source_fd,
            initial_stat=initial_stat,
            path=snapshot_path,
            size=total,
            sha512_digest=hasher.digest(),
        )
        verify_original_archive(snapshot)
        body_started = True
        yield snapshot
    except SystemExit:
        raise
    except OSError as exc:
        if body_started:
            raise
        raise SystemExit(f"failed to snapshot firmware archive: {exc}") from exc
    finally:
        if source_fd >= 0:
            os.close(source_fd)
        if private_directory is not None:
            shutil.rmtree(private_directory, ignore_errors=True)


def metadata_from_snapshot(
    version: str,
    archive_path: Path,
    snapshot: FirmwareArchiveSnapshot,
    signing_key: Path,
    keyring: Path,
    update_uboot: bool = False,
) -> Dict[str, object]:
    if not isinstance(update_uboot, bool):
        raise SystemExit("update_uboot must be a boolean")

    expected_format = expected_archive_format(archive_path, version)
    actual_format = detect_archive_format(snapshot.path)
    if actual_format != expected_format:
        raise SystemExit(
            "firmware archive compression magic does not match its filename extension"
        )

    ensure_keyring_matches(signing_key, keyring)
    validate_firmware_archive(snapshot.path, actual_format, version)
    return {
        "version": version,
        "name": archive_path.name,
        "sha512": base64.b64encode(snapshot.sha512_digest).decode("ascii"),
        "size": snapshot.size,
        "updateUboot": update_uboot,
        "signature": sign_artifact(snapshot.path, signing_key, keyring),
    }


def generate_firmware_metadata(
    version: str,
    archive: Path,
    signing_key: Path,
    keyring: Path,
    update_uboot: bool = False,
) -> Dict[str, object]:
    version = validate_version(version)
    archive_path = absolute_without_resolving(archive)
    with create_firmware_archive_snapshot(archive_path) as snapshot:
        metadata = metadata_from_snapshot(
            version,
            archive_path,
            snapshot,
            signing_key,
            keyring,
            update_uboot,
        )
        verify_original_archive(snapshot)
        return metadata


def prepare_output_parent(output_path: Path) -> None:
    try:
        output_path.parent.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        raise SystemExit(
            f"failed to create Firmware metadata directory: {output_path.parent}"
        ) from exc


def stage_firmware_metadata(output_path: Path, encoded: bytes) -> Path:
    prepare_output_parent(output_path)
    descriptor = -1
    temporary_path: Path | None = None
    try:
        descriptor, temporary_name = tempfile.mkstemp(
            prefix=f".{output_path.name}.",
            dir=output_path.parent,
        )
        temporary_path = Path(temporary_name)
        with os.fdopen(descriptor, "wb") as staged:
            descriptor = -1
            staged.write(encoded)
            staged.flush()
            os.fchmod(staged.fileno(), FILE_MODE)
            os.fsync(staged.fileno())
        return temporary_path
    except OSError as exc:
        if descriptor >= 0:
            os.close(descriptor)
        if temporary_path is not None:
            temporary_path.unlink(missing_ok=True)
        raise SystemExit(f"failed to stage Firmware metadata: {exc}") from exc


def fsync_directory(path: Path) -> None:
    descriptor = os.open(path, os.O_RDONLY | os.O_CLOEXEC | os.O_DIRECTORY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def commit_staged_firmware_metadata(
    staged_path: Path,
    output_path: Path,
    force: bool,
    snapshot: FirmwareArchiveSnapshot,
) -> None:
    verify_original_archive(snapshot)
    try:
        if force:
            os.replace(staged_path, output_path)
        else:
            os.link(staged_path, output_path)
            staged_path.unlink()
        fsync_directory(output_path.parent)
    except FileExistsError as exc:
        raise SystemExit(
            f"{output_path} already exists; pass --force to overwrite"
        ) from exc
    except OSError as exc:
        raise SystemExit(f"failed to commit Firmware metadata: {exc}") from exc


def path_exists_without_following(path: Path) -> bool:
    try:
        path.lstat()
        return True
    except FileNotFoundError:
        return False
    except OSError as exc:
        raise SystemExit(f"failed to inspect path: {path}") from exc


def remove_obsolete_firmware_signature(path: Path) -> None:
    try:
        mode = path.lstat().st_mode
    except FileNotFoundError:
        return
    except OSError as exc:
        raise SystemExit(
            f"failed to inspect obsolete Firmware signature: {path}"
        ) from exc
    if not (stat.S_ISREG(mode) or stat.S_ISLNK(mode)):
        raise SystemExit(
            f"refusing to remove non-regular obsolete Firmware signature: {path}"
        )
    try:
        path.unlink()
        fsync_directory(path.parent)
    except OSError as exc:
        raise SystemExit(
            f"failed to remove obsolete Firmware signature: {path}"
        ) from exc


def write_firmware_metadata(
    version: str,
    archive: Path,
    signing_key: Path,
    keyring: Path,
    output: Path,
    force: bool = False,
    update_uboot: bool = False,
) -> Dict[str, object]:
    version = validate_version(version)
    archive_path = absolute_without_resolving(archive)
    output_path = absolute_without_resolving(output)
    obsolete_signature = ensure_output_is_safe(
        output_path,
        version,
        archive_path,
        signing_key,
        keyring,
        force,
    )

    prepare_output_parent(output_path)
    with create_firmware_archive_snapshot(
        archive_path,
        temporary_parent=output_path.parent,
    ) as snapshot:
        metadata = metadata_from_snapshot(
            version,
            archive_path,
            snapshot,
            signing_key,
            keyring,
            update_uboot,
        )
        encoded = (json.dumps(metadata, indent=2, ensure_ascii=False) + "\n").encode(
            "utf-8"
        )
        staged_path = stage_firmware_metadata(output_path, encoded)
        try:
            commit_staged_firmware_metadata(
                staged_path,
                output_path,
                force,
                snapshot,
            )
            if force:
                remove_obsolete_firmware_signature(obsolete_signature)
            elif path_exists_without_following(obsolete_signature):
                raise SystemExit(
                    "obsolete Firmware signature appeared while metadata was generated: "
                    f"{obsolete_signature}"
                )
            return metadata
        finally:
            staged_path.unlink(missing_ok=True)


def main() -> None:
    args = parse_args()
    metadata = write_firmware_metadata(
        args.version,
        args.archive,
        args.signing_key,
        args.keyring,
        args.output,
        args.force,
        args.update_uboot,
    )
    signature = metadata["signature"]
    assert isinstance(signature, dict)
    record = {
        "metadataName": args.output.name,
        "version": metadata["version"],
        "name": metadata["name"],
        "size": metadata["size"],
        "sha512": metadata["sha512"],
        "updateUboot": metadata["updateUboot"],
        "keyId": signature["keyId"],
        "signature": signature["value"],
    }
    print(json.dumps(record, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
