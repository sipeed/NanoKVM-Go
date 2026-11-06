#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import gzip
import hashlib
import json
import os
import shutil
import tarfile
import tempfile
from pathlib import Path
from typing import Dict, List, Optional

from update_signing import atomic_write, ensure_keyring_matches, sign_artifact


PACKAGES = ("nanokvm-go", "kvmcomm")
DEFAULT_ARCH = "armhf"
ARCHIVE_FILE_MODE = 0o644
ARCHIVE_DIR_MODE = 0o755
MAX_GZIP_MTIME = (1 << 32) - 1


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate update package metadata and bundle NanoKVM Go deb files."
    )
    parser.add_argument(
        "-v",
        "--version",
        help="Package version, for example 0.0.1. If omitted, the script prompts for it.",
    )
    parser.add_argument(
        "-i",
        "--input-dir",
        type=Path,
        default=Path.cwd(),
        help="Directory containing nanokvm-go_<version>_<arch>.deb and kvmcomm_<version>_<arch>.deb.",
    )
    parser.add_argument(
        "-o",
        "--output-dir",
        type=Path,
        help="Directory for the staging package directory, tar.gz, and latest metadata. Defaults to --input-dir.",
    )
    parser.add_argument(
        "--arch",
        default=DEFAULT_ARCH,
        help=f"Debian package architecture suffix. Defaults to {DEFAULT_ARCH}.",
    )
    parser.add_argument(
        "--archive-name",
        help="Output archive filename. Defaults to nanokvm-go_<version>.tar.gz.",
    )
    parser.add_argument(
        "--latest-name",
        default="latest.json",
        help="Latest metadata filename. Defaults to latest.json.",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite generated outputs and remove obsolete external application metadata.",
    )
    parser.add_argument(
        "--signing-key",
        type=Path,
        help=(
            "Offline Ed25519 private key used only to sign bundled deb artifacts; "
            "omit it and --keyring to create an unsigned bundle."
        ),
    )
    parser.add_argument(
        "--keyring",
        type=Path,
        help=(
            "Packaged public key ring; must be supplied with --signing-key and "
            "must match it."
        ),
    )
    return parser.parse_args()


def get_version(version: Optional[str]) -> str:
    if version is None:
        version = input("Version: ")

    version = version.strip()
    if not version:
        raise SystemExit("version must not be empty")
    if version.startswith(("v", "V")):
        raise SystemExit("version must omit the leading 'v' prefix, for example use 0.0.2")
    if "/" in version or "\\" in version or "\0" in version:
        raise SystemExit(f"invalid version for filenames: {version!r}")

    return version


def deb_name(package: str, version: str, arch: str) -> str:
    return f"{package}_{version}_{arch}.deb"


def json_name(package: str, version: str) -> str:
    return f"{package}_{version}.json"


def package_dir_name(version: str) -> str:
    return f"nanokvm-go_{version}"


def validate_output_name(value: str, option: str, suffix: str) -> str:
    if (
        not value
        or value in (".", "..")
        or "/" in value
        or "\\" in value
        or "\0" in value
    ):
        raise SystemExit(f"{option} must be a plain filename")
    if not value.endswith(suffix):
        raise SystemExit(f"{option} must end with {suffix}")
    return value


def sha512_base64(path: Path) -> str:
    hasher = hashlib.sha512()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            hasher.update(chunk)
    return base64.b64encode(hasher.digest()).decode("ascii")


def file_metadata(file_path: Path, version: str) -> Dict[str, object]:
    return {
        "version": version,
        "name": file_path.name,
        "sha512": sha512_base64(file_path),
        "size": file_path.stat().st_size,
    }


def deb_metadata(
    deb_path: Path,
    version: str,
    package: str,
    architecture: str,
    signing_key: Optional[Path],
    keyring: Optional[Path],
) -> Dict[str, object]:
    metadata = file_metadata(deb_path, version)
    metadata["package"] = package
    metadata["architecture"] = architecture
    if signing_key is not None and keyring is not None:
        metadata["signature"] = sign_artifact(deb_path, signing_key, keyring)
    return metadata


def get_source_date_epoch() -> int:
    value = os.environ.get("SOURCE_DATE_EPOCH", "0")
    if not value or not value.isascii() or not value.isdigit():
        raise SystemExit("SOURCE_DATE_EPOCH must be a non-negative decimal integer")
    epoch = int(value, 10)
    if epoch > MAX_GZIP_MTIME:
        raise SystemExit(
            f"SOURCE_DATE_EPOCH must not exceed {MAX_GZIP_MTIME} for gzip output"
        )
    return epoch


def ensure_can_write(path: Path, force: bool) -> None:
    if path.exists() and not force:
        raise SystemExit(f"{path} already exists; pass --force to overwrite")


def ensure_paths_do_not_collide(
    generated_paths: Dict[str, Path], protected_paths: Dict[str, Path]
) -> None:
    seen: Dict[Path, str] = {}
    for label, path in generated_paths.items():
        try:
            resolved = path.resolve()
        except OSError as exc:
            raise SystemExit(f"failed to resolve {label} path: {path}") from exc
        previous = seen.get(resolved)
        if previous is not None:
            raise SystemExit(f"generated output paths collide: {previous} and {label}")
        seen[resolved] = label

    protected = {}
    for label, path in protected_paths.items():
        try:
            protected[path.resolve()] = label
        except OSError as exc:
            raise SystemExit(f"failed to resolve {label} path: {path}") from exc
    for resolved, label in seen.items():
        protected_label = protected.get(resolved)
        if protected_label is not None:
            raise SystemExit(
                f"generated output path {label} collides with {protected_label}"
            )


def remove_obsolete_output(path: Path, force: bool) -> None:
    if not path.exists() and not path.is_symlink():
        return
    if not force:
        raise SystemExit(
            f"obsolete external application metadata exists: {path}; "
            "pass --force to remove it"
        )
    if path.is_dir() and not path.is_symlink():
        raise SystemExit(
            f"refusing to remove obsolete external metadata directory: {path}"
        )
    path.unlink()


def write_json(path: Path, data: Dict[str, object], force: bool) -> None:
    encoded = (json.dumps(data, indent=2, ensure_ascii=False) + "\n").encode(
        "utf-8"
    )
    atomic_write(path, encoded, force)


def copy_file(source: Path, target: Path) -> None:
    shutil.copyfile(source, target)
    target.chmod(ARCHIVE_FILE_MODE)


def add_directory(tar: tarfile.TarFile, archive_name: str, mtime: int) -> None:
    info = tarfile.TarInfo(name=archive_name)
    info.type = tarfile.DIRTYPE
    info.mtime = mtime
    info.uid = 0
    info.gid = 0
    info.uname = "root"
    info.gname = "root"
    info.mode = ARCHIVE_DIR_MODE
    tar.addfile(info)


def add_file(
    tar: tarfile.TarFile, source: Path, archive_name: str, mtime: int
) -> None:
    stat = source.stat()
    info = tarfile.TarInfo(name=archive_name)
    info.size = stat.st_size
    info.mtime = mtime
    info.uid = 0
    info.gid = 0
    info.uname = "root"
    info.gname = "root"
    info.mode = ARCHIVE_FILE_MODE

    with source.open("rb") as file:
        tar.addfile(info, file)


def create_archive(
    archive_path: Path,
    package_dir: Path,
    files: List[Path],
    force: bool,
    mtime: int,
) -> None:
    ensure_can_write(archive_path, force)
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{archive_path.name}.", dir=archive_path.parent
    )
    os.close(descriptor)
    temporary_path = Path(temporary_name)
    try:
        with temporary_path.open("wb") as output:
            with gzip.GzipFile(
                filename="",
                mode="wb",
                compresslevel=9,
                fileobj=output,
                mtime=mtime,
            ) as compressed:
                with tarfile.open(
                    fileobj=compressed, mode="w", format=tarfile.PAX_FORMAT
                ) as tar:
                    add_directory(tar, package_dir.name, mtime)
                    for path in files:
                        add_file(
                            tar,
                            path,
                            f"{package_dir.name}/{path.name}",
                            mtime,
                        )
            output.flush()
            os.fsync(output.fileno())

        expected = [package_dir.name] + [
            f"{package_dir.name}/{path.name}" for path in files
        ]
        with tarfile.open(temporary_path, "r:gz") as tar:
            names = tar.getnames()
        if names != expected:
            raise SystemExit(
                f"archive contents mismatch: expected {expected}, got {names}"
            )

        temporary_path.chmod(ARCHIVE_FILE_MODE)
        os.replace(temporary_path, archive_path)
    finally:
        temporary_path.unlink(missing_ok=True)


def main() -> None:
    args = parse_args()
    version = get_version(args.version)
    input_dir = args.input_dir.resolve()
    output_dir = (args.output_dir or args.input_dir).resolve()
    package_dir = output_dir / package_dir_name(version)
    archive_name = validate_output_name(
        args.archive_name or f"nanokvm-go_{version}.tar.gz",
        "--archive-name",
        ".tar.gz",
    )
    latest_name = validate_output_name(args.latest_name, "--latest-name", ".json")
    archive_path = output_dir / archive_name
    latest_path = output_dir / latest_name
    obsolete_versioned_metadata_path = output_dir / f"nanokvm-go_{version}.json"
    obsolete_latest_signature_path = Path(f"{latest_path}.sig")
    obsolete_versioned_metadata_signature_path = Path(
        f"{obsolete_versioned_metadata_path}.sig"
    )
    if latest_path == obsolete_versioned_metadata_path:
        raise SystemExit(
            "--latest-name must not use the reserved bundled package metadata filename"
        )
    signing_key = args.signing_key
    keyring = args.keyring
    if (signing_key is None) != (keyring is None):
        raise SystemExit("--signing-key and --keyring must be supplied together")
    signed = signing_key is not None
    if input_dir == package_dir:
        raise SystemExit("--input-dir must not be the generated package directory")

    mtime = get_source_date_epoch()
    source_debs = {
        package: input_dir / deb_name(package, version, args.arch)
        for package in PACKAGES
    }
    for source_deb_path in source_debs.values():
        if not source_deb_path.is_file():
            raise SystemExit(f"missing deb file: {source_deb_path}")
    protected_paths = {
        f"{package} input deb": source_deb_path
        for package, source_deb_path in source_debs.items()
    }
    if signed:
        assert keyring is not None
        protected_paths.update(
            {
                "signing key": signing_key,
                "keyring": keyring,
            }
        )
    ensure_paths_do_not_collide(
        {
            "archive": archive_path,
            "latest metadata": latest_path,
            "obsolete versioned metadata": obsolete_versioned_metadata_path,
            "obsolete latest signature": obsolete_latest_signature_path,
            "obsolete versioned metadata signature": (
                obsolete_versioned_metadata_signature_path
            ),
            "staging directory": package_dir,
        },
        protected_paths,
    )
    if signed:
        assert signing_key is not None and keyring is not None
        ensure_keyring_matches(signing_key, keyring)
    else:
        print("WARNING: creating an unsigned application update bundle")

    output_dir.mkdir(parents=True, exist_ok=True)
    output_paths = [archive_path, latest_path, package_dir]
    for path in output_paths:
        ensure_can_write(path, args.force)
    for path in (
        obsolete_versioned_metadata_path,
        obsolete_latest_signature_path,
        obsolete_versioned_metadata_signature_path,
    ):
        remove_obsolete_output(path, args.force)
    if package_dir.exists():
        shutil.rmtree(package_dir)
    package_dir.mkdir(mode=ARCHIVE_DIR_MODE)
    package_dir.chmod(ARCHIVE_DIR_MODE)

    files_for_archive: List[Path] = []
    for package in PACKAGES:
        source_deb_path = source_debs[package]
        deb_path = package_dir / source_deb_path.name
        copy_file(source_deb_path, deb_path)

        json_path = package_dir / json_name(package, version)
        metadata = deb_metadata(
            deb_path,
            version,
            package,
            args.arch,
            signing_key,
            keyring,
        )
        write_json(json_path, metadata, args.force)

        files_for_archive.extend([deb_path, json_path])

    create_archive(archive_path, package_dir, files_for_archive, args.force, mtime)
    archive_metadata = file_metadata(archive_path, version)
    # latest.json is the discovery pointer, so publish it last locally as well.
    write_json(latest_path, archive_metadata, args.force)
    shutil.rmtree(package_dir)

    print(f"created {archive_path}")
    print(f"created {latest_path}")
    print(f"removed {package_dir}")
    print("archive entries:")
    print(f"  {package_dir.name}/")
    for path in files_for_archive:
        print(f"  {package_dir.name}/{path.name}")


if __name__ == "__main__":
    main()
