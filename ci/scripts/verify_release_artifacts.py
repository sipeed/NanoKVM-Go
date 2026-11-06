#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import subprocess
import tarfile
from pathlib import Path
from typing import Any


PACKAGES = ("nanokvm-go", "kvmcomm")
ARCH = "armhf"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verify NanoKVM-Go Debian packages and update bundle metadata."
    )
    parser.add_argument("--build-dir", type=Path, default=Path("build"))
    parser.add_argument("--version", required=True)
    return parser.parse_args()


def sha512_base64(path: Path) -> str:
    digest = hashlib.sha512()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            digest.update(chunk)
    return base64.b64encode(digest.digest()).decode("ascii")


def sha512_base64_stream(file: Any) -> tuple[str, int]:
    digest = hashlib.sha512()
    size = 0
    for chunk in iter(lambda: file.read(1024 * 1024), b""):
        digest.update(chunk)
        size += len(chunk)
    return base64.b64encode(digest.digest()).decode("ascii"), size


def read_json_bytes(data: bytes, label: str) -> dict[str, Any]:
    try:
        value = json.loads(data.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise SystemExit(f"invalid JSON in {label}: {error}") from error
    if not isinstance(value, dict):
        raise SystemExit(f"expected JSON object in {label}")
    return value


def read_json_file(path: Path) -> dict[str, Any]:
    if not path.is_file():
        raise SystemExit(f"missing JSON file: {path}")
    return read_json_bytes(path.read_bytes(), str(path))


def require_metadata(
    metadata: dict[str, Any],
    path: Path,
    version: str,
    *,
    expected_name: str,
    package: str | None = None,
) -> None:
    expected: dict[str, Any] = {
        "version": version,
        "name": expected_name,
        "sha512": sha512_base64(path),
        "size": path.stat().st_size,
    }
    if package is not None:
        expected["package"] = package
        expected["architecture"] = ARCH
    if metadata != expected:
        raise SystemExit(
            f"metadata mismatch for {path}: expected {expected}, got {metadata}"
        )


def dpkg_field(path: Path, field: str) -> str:
    try:
        result = subprocess.run(
            ("dpkg-deb", "--field", str(path), field),
            check=True,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError as error:
        raise SystemExit("dpkg-deb is required to verify release artifacts") from error
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip() or error.stdout.strip()
        raise SystemExit(f"dpkg-deb failed for {path}: {detail}") from error
    return result.stdout.strip()


def verify_deb(path: Path, package: str, version: str) -> None:
    if not path.is_file():
        raise SystemExit(f"missing Debian package: {path}")
    actual_package = dpkg_field(path, "Package")
    actual_version = dpkg_field(path, "Version")
    actual_arch = dpkg_field(path, "Architecture")
    if actual_package != package:
        raise SystemExit(
            f"package mismatch for {path}: expected {package}, got {actual_package}"
        )
    if actual_version != version:
        raise SystemExit(
            f"version mismatch for {path}: expected {version}, got {actual_version}"
        )
    if actual_arch != ARCH:
        raise SystemExit(
            f"architecture mismatch for {path}: expected {ARCH}, got {actual_arch}"
        )


def verify_archive(build_dir: Path, version: str) -> None:
    update_dir = build_dir / "update"
    archive = update_dir / f"nanokvm-go_{version}.tar.gz"
    latest = update_dir / "latest.json"
    package_dir = f"nanokvm-go_{version}"

    if not archive.is_file():
        raise SystemExit(f"missing update archive: {archive}")

    expected_names = [package_dir]
    for package in PACKAGES:
        expected_names.extend(
            (
                f"{package_dir}/{package}_{version}_{ARCH}.deb",
                f"{package_dir}/{package}_{version}.json",
            )
        )

    with tarfile.open(archive, "r:gz") as tar:
        members = tar.getmembers()
        actual_names = [member.name for member in members]
        if actual_names != expected_names:
            raise SystemExit(
                f"archive contents mismatch: expected {expected_names}, got {actual_names}"
            )

        if not members[0].isdir():
            raise SystemExit(f"archive root is not a directory: {members[0].name}")
        for member in members[1:]:
            if not member.isfile():
                raise SystemExit(f"archive member is not a regular file: {member.name}")

        for package in PACKAGES:
            deb_path = build_dir / f"{package}_{version}_{ARCH}.deb"
            deb_name = f"{package_dir}/{package}_{version}_{ARCH}.deb"
            deb_file = tar.extractfile(deb_name)
            if deb_file is None:
                raise SystemExit(f"missing archive Debian package: {deb_name}")
            archive_sha512, archive_size = sha512_base64_stream(deb_file)
            expected_sha512 = sha512_base64(deb_path)
            expected_size = deb_path.stat().st_size
            if archive_sha512 != expected_sha512 or archive_size != expected_size:
                raise SystemExit(
                    f"archive Debian package mismatch for {deb_name}: "
                    f"expected size={expected_size}, sha512={expected_sha512}; "
                    f"got size={archive_size}, sha512={archive_sha512}"
                )

            metadata_name = f"{package_dir}/{package}_{version}.json"
            metadata_file = tar.extractfile(metadata_name)
            if metadata_file is None:
                raise SystemExit(f"missing archive metadata: {metadata_name}")
            metadata = read_json_bytes(metadata_file.read(), metadata_name)
            require_metadata(
                metadata,
                deb_path,
                version,
                expected_name=deb_path.name,
                package=package,
            )

    latest_metadata = read_json_file(latest)
    require_metadata(
        latest_metadata,
        archive,
        version,
        expected_name=archive.name,
    )


def main() -> None:
    args = parse_args()
    build_dir = args.build_dir.resolve()
    version = args.version.strip()
    if not version:
        raise SystemExit("version must not be empty")

    for package in PACKAGES:
        verify_deb(
            build_dir / f"{package}_{version}_{ARCH}.deb",
            package,
            version,
        )

    verify_archive(build_dir, version)
    print(f"verified release artifacts for NanoKVM-Go {version}")


if __name__ == "__main__":
    main()
