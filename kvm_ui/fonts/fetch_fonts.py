#!/usr/bin/env python3
"""Fetch the immutable base-image font catalog described by font-lock.json."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import sys
import tempfile
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Mapping, Optional, Sequence, Tuple
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import Request, urlopen


SCRIPT_DIR = Path(__file__).resolve().parent
LOCK_PATH = SCRIPT_DIR / "font-lock.json"
ATTESTATION_NAME = "catalog-attestation.json"
NETWORK_TIMEOUT_SECONDS = 60
DOWNLOAD_CHUNK_SIZE = 128 * 1024
MAX_TREE_RESPONSE_BYTES = 32 * 1024 * 1024
USER_AGENT = "kvm-ui-font-fetcher/1"


class FontFetchError(RuntimeError):
    """A recoverable catalog fetch or verification error."""


@dataclass(frozen=True)
class Asset:
    asset_id: str
    source_set: str
    source_path: str
    install_path: str
    expected_size: int
    expected_sha256: Optional[str]
    expected_git_blob_sha1: Optional[str]
    role: str


@dataclass(frozen=True)
class FileDigest:
    size: int
    sha256: str
    git_blob_sha1: Optional[str]


def parse_arguments(argv: Optional[Sequence[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Fetch the locked kvm_ui base-image font catalog."
    )
    parser.add_argument(
        "target_directory",
        nargs="?",
        help="directory to receive font files (default: this script's directory)",
    )
    return parser.parse_args(argv)


def resolve_target(target_argument: Optional[str]) -> Path:
    if target_argument is None:
        return SCRIPT_DIR
    return Path(target_argument).expanduser().resolve()


def prepare_target(target: Path) -> None:
    try:
        target.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        raise FontFetchError(
            f"cannot create target directory '{target}': {exc}"
        ) from exc

    if not target.is_dir():
        raise FontFetchError(f"target '{target}' is not a directory")

    probe_path: Optional[Path] = None
    try:
        fd, probe_name = tempfile.mkstemp(
            prefix=".font-fetch-write-check-", suffix=".part", dir=target
        )
        probe_path = Path(probe_name)
        os.close(fd)
        probe_path.unlink()
    except OSError as exc:
        raise FontFetchError(f"target '{target}' is not writable: {exc}") from exc
    finally:
        if probe_path is not None and probe_path.exists():
            try:
                probe_path.unlink()
            except OSError:
                pass


def load_lock(path: Path = LOCK_PATH) -> Dict[str, Any]:
    try:
        with path.open("r", encoding="utf-8") as source:
            lock = json.load(source)
    except (OSError, json.JSONDecodeError) as exc:
        raise FontFetchError(f"cannot read font lock '{path}': {exc}") from exc

    if not isinstance(lock, dict) or lock.get("schema_version") != 3:
        raise FontFetchError("font lock must be a schema_version 3 JSON object")
    if not isinstance(lock.get("source_sets"), dict):
        raise FontFetchError("font lock is missing source_sets")
    if not isinstance(lock.get("assets"), list):
        raise FontFetchError("font lock is missing assets")
    return lock


def require_string(value: Any, field: str) -> str:
    if not isinstance(value, str) or not value:
        raise FontFetchError(f"font lock field '{field}' must be a non-empty string")
    return value


def require_nonnegative_int(value: Any, field: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < 0:
        raise FontFetchError(
            f"font lock field '{field}' must be a non-negative integer"
        )
    return value


def require_sha256(value: Any, field: str) -> str:
    digest = require_string(value, field).lower()
    if re.fullmatch(r"[0-9a-f]{64}", digest) is None:
        raise FontFetchError(f"font lock field '{field}' is not a SHA-256 digest")
    return digest


def require_git_sha1(value: Any, field: str) -> str:
    digest = require_string(value, field).lower()
    if re.fullmatch(r"[0-9a-f]{40}", digest) is None:
        raise FontFetchError(f"font lock field '{field}' is not a Git SHA-1")
    return digest


def validate_install_path(value: Any, field: str, extensions: Tuple[str, ...]) -> str:
    path = require_string(value, field)
    candidate = PurePosixPath(path)
    if candidate.is_absolute() or ".." in candidate.parts:
        raise FontFetchError(f"font lock field '{field}' is not a safe relative path")
    if candidate.suffix.lower() not in extensions:
        suffixes = ", ".join(extensions)
        raise FontFetchError(f"font lock field '{field}' must end in {suffixes}")
    return path


def validate_install_name(value: Any, field: str) -> str:
    install_name = validate_install_path(value, field, (".ttf", ".otf", ".ttc", ".otc"))
    if PurePosixPath(install_name).name != install_name:
        raise FontFetchError(f"font lock field '{field}' must be a plain filename")
    return install_name


def install_path_for_noto(
    source_path: str,
    config: Mapping[str, Any],
) -> str:
    install_overrides = config.get("path_install_paths", {})
    if not isinstance(install_overrides, dict):
        raise FontFetchError("noto_tree.path_install_paths must be an object")
    override = install_overrides.get(source_path)
    if override is not None:
        return validate_install_path(
            override,
            f"noto_tree.path_install_paths.{source_path}",
            (".ttf", ".otf", ".ttc", ".otc"),
        )

    install_subdirectory = require_string(
        config.get("install_subdirectory"),
        "noto_tree.install_subdirectory",
    )
    directory = PurePosixPath(install_subdirectory)
    if directory.is_absolute() or ".." in directory.parts:
        raise FontFetchError("noto_tree.install_subdirectory is not a safe path")
    try:
        relative_source_path = PurePosixPath(source_path).relative_to("hinted/ttf")
    except ValueError as exc:
        raise FontFetchError(
            f"noto-tree: selected source path '{source_path}' has no hinted/ttf prefix"
        ) from exc
    return validate_install_path(
        str(directory / relative_source_path),
        f"noto_tree.install_path.{source_path}",
        (".ttf", ".otf", ".ttc", ".otc"),
    )


def source_url(source_sets: Mapping[str, Any], asset: Asset) -> str:
    source_set = source_sets.get(asset.source_set)
    if not isinstance(source_set, dict):
        raise FontFetchError(f"asset '{asset.asset_id}' has unknown source set")
    template = require_string(
        source_set.get("url_template"),
        f"source_sets.{asset.source_set}.url_template",
    )
    return template.replace("{path}", quote(asset.source_path, safe="/"))


def explicit_assets(lock: Mapping[str, Any]) -> List[Asset]:
    result: List[Asset] = []
    source_sets = lock["source_sets"]
    assert isinstance(source_sets, dict)
    for index, entry in enumerate(lock["assets"]):
        prefix = f"assets[{index}]"
        if not isinstance(entry, dict):
            raise FontFetchError(f"{prefix} must be an object")
        asset = Asset(
            asset_id=require_string(entry.get("id"), f"{prefix}.id"),
            source_set=require_string(entry.get("source_set"), f"{prefix}.source_set"),
            source_path=validate_install_path(
                entry.get("source_path"),
                f"{prefix}.source_path",
                (".ttf", ".otf", ".ttc", ".otc"),
            ),
            install_path=validate_install_name(
                entry.get("install_name"), f"{prefix}.install_name"
            ),
            expected_size=require_nonnegative_int(entry.get("size"), f"{prefix}.size"),
            expected_sha256=require_sha256(entry.get("sha256"), f"{prefix}.sha256"),
            expected_git_blob_sha1=None,
            role=require_string(entry.get("role"), f"{prefix}.role"),
        )
        source_url(source_sets, asset)
        result.append(asset)
    return result


def tree_manifest_digest(entries: Iterable[Mapping[str, Any]]) -> str:
    digest = hashlib.sha256()
    for entry in sorted(entries, key=lambda item: str(item["path"])):
        path = require_string(entry.get("path"), "tree.path")
        blob_sha = require_git_sha1(entry.get("sha"), f"tree.{path}.sha")
        size = require_nonnegative_int(entry.get("size"), f"tree.{path}.size")
        digest.update(f"{path}\0{blob_sha}\0{size}\n".encode("utf-8"))
    return digest.hexdigest()


def payload_manifest_digest(
    entries: Iterable[Tuple[str, str, int]],
) -> str:
    digest = hashlib.sha256()
    for path, sha256, size in sorted(entries, key=lambda item: item[0]):
        digest.update(f"{path}\0{sha256}\0{size}\n".encode("utf-8"))
    return digest.hexdigest()


def read_json_url(url: str, item_id: str) -> Mapping[str, Any]:
    request = Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urlopen(request, timeout=NETWORK_TIMEOUT_SECONDS) as response:
            payload = response.read(MAX_TREE_RESPONSE_BYTES + 1)
    except (HTTPError, URLError, OSError, TimeoutError) as exc:
        raise FontFetchError(f"{item_id}: cannot download '{url}': {exc}") from exc

    if len(payload) > MAX_TREE_RESPONSE_BYTES:
        raise FontFetchError(f"{item_id}: metadata response is too large")
    try:
        decoded = json.loads(payload.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise FontFetchError(f"{item_id}: invalid JSON response: {exc}") from exc
    if not isinstance(decoded, dict):
        raise FontFetchError(f"{item_id}: metadata response must be an object")
    return decoded


def noto_tree_assets(lock: Mapping[str, Any]) -> List[Asset]:
    config = lock.get("noto_tree")
    if not isinstance(config, dict):
        raise FontFetchError("font lock is missing noto_tree")

    source_set = require_string(config.get("source_set"), "noto_tree.source_set")
    source_sets = lock["source_sets"]
    assert isinstance(source_sets, dict)
    if source_set not in source_sets:
        raise FontFetchError("noto_tree source set is not defined")

    tree_sha1 = require_git_sha1(config.get("tree_sha1"), "noto_tree.tree_sha1")
    tree_url = require_string(config.get("tree_url"), "noto_tree.tree_url")
    path_pattern = require_string(config.get("path_regex"), "noto_tree.path_regex")
    try:
        selector = re.compile(path_pattern)
    except re.error as exc:
        raise FontFetchError(f"noto_tree.path_regex is invalid: {exc}") from exc

    tree = read_json_url(tree_url, "noto-tree")
    if require_git_sha1(tree.get("sha"), "noto-tree.sha") != tree_sha1:
        raise FontFetchError("noto-tree: tree SHA-1 differs from font lock")
    if tree.get("truncated") is not False:
        raise FontFetchError("noto-tree: recursive tree response is truncated")
    raw_entries = tree.get("tree")
    if not isinstance(raw_entries, list):
        raise FontFetchError("noto-tree: response is missing tree entries")

    selected: List[Mapping[str, Any]] = []
    for entry in raw_entries:
        if not isinstance(entry, dict):
            continue
        path = entry.get("path")
        if entry.get("type") == "blob" and isinstance(path, str) and selector.fullmatch(path):
            selected.append(entry)

    expected_count = require_nonnegative_int(
        config.get("expected_count"), "noto_tree.expected_count"
    )
    expected_total_bytes = require_nonnegative_int(
        config.get("expected_total_bytes"), "noto_tree.expected_total_bytes"
    )
    expected_manifest = require_sha256(
        config.get("tree_manifest_sha256"), "noto_tree.tree_manifest_sha256"
    )
    if len(selected) != expected_count:
        raise FontFetchError(
            f"noto-tree: expected {expected_count} selected files, got {len(selected)}"
        )
    total_bytes = sum(
        require_nonnegative_int(entry.get("size"), "noto-tree.entry.size")
        for entry in selected
    )
    if total_bytes != expected_total_bytes:
        raise FontFetchError(
            f"noto-tree: expected {expected_total_bytes} bytes, got {total_bytes}"
        )
    if tree_manifest_digest(selected) != expected_manifest:
        raise FontFetchError("noto-tree: selected manifest digest differs from lock")

    default_role = require_string(config.get("default_role"), "noto_tree.default_role")
    path_roles = config.get("path_roles", {})
    if not isinstance(path_roles, dict):
        raise FontFetchError("noto_tree.path_roles must be an object")

    result: List[Asset] = []
    names = set()
    for entry in selected:
        path = require_string(entry.get("path"), "noto-tree.entry.path")
        install_path = install_path_for_noto(path, config)
        if install_path in names:
            raise FontFetchError(
                f"noto-tree: duplicate installed path '{install_path}'"
            )
        names.add(install_path)
        role_value = path_roles.get(path, default_role)
        result.append(
            Asset(
                asset_id=f"noto/{path}",
                source_set=source_set,
                source_path=path,
                install_path=install_path,
                expected_size=require_nonnegative_int(
                    entry.get("size"), f"noto-tree.{path}.size"
                ),
                expected_sha256=None,
                expected_git_blob_sha1=require_git_sha1(
                    entry.get("sha"), f"noto-tree.{path}.sha"
                ),
                role=require_string(role_value, f"noto_tree.path_roles.{path}"),
            )
        )
    return result


def file_digest(path: Path, expected_git_blob_size: Optional[int]) -> FileDigest:
    sha256 = hashlib.sha256()
    git_sha1 = None
    if expected_git_blob_size is not None:
        git_sha1 = hashlib.sha1()
        git_sha1.update(f"blob {expected_git_blob_size}\0".encode("ascii"))

    size = 0
    try:
        with path.open("rb") as source:
            while True:
                block = source.read(DOWNLOAD_CHUNK_SIZE)
                if not block:
                    break
                size += len(block)
                sha256.update(block)
                if git_sha1 is not None:
                    git_sha1.update(block)
    except OSError as exc:
        raise FontFetchError(f"cannot read '{path}': {exc}") from exc
    return FileDigest(
        size=size,
        sha256=sha256.hexdigest(),
        git_blob_sha1=git_sha1.hexdigest() if git_sha1 is not None else None,
    )


def verify_digest(asset: Asset, digest: FileDigest, location: Path) -> None:
    if digest.size != asset.expected_size:
        raise FontFetchError(
            f"{asset.asset_id}: '{location}' has {digest.size} bytes; "
            f"expected {asset.expected_size}"
        )
    if asset.expected_sha256 is not None and digest.sha256 != asset.expected_sha256:
        raise FontFetchError(
            f"{asset.asset_id}: SHA-256 mismatch for '{location}' "
            f"(expected {asset.expected_sha256}, got {digest.sha256})"
        )
    if (
        asset.expected_git_blob_sha1 is not None
        and digest.git_blob_sha1 != asset.expected_git_blob_sha1
    ):
        raise FontFetchError(
            f"{asset.asset_id}: Git blob SHA-1 mismatch for '{location}'"
        )


def download_asset(asset: Asset, url: str, target_path: Path) -> FileDigest:
    try:
        fd, part_name = tempfile.mkstemp(
            prefix=f".{target_path.name}.", suffix=".part", dir=target_path.parent
        )
    except OSError as exc:
        raise FontFetchError(
            f"{asset.asset_id}: cannot create temporary file in "
            f"'{target_path.parent}': {exc}"
        ) from exc

    part_path = Path(part_name)
    try:
        sha256 = hashlib.sha256()
        git_sha1 = None
        if asset.expected_git_blob_sha1 is not None:
            git_sha1 = hashlib.sha1()
            git_sha1.update(f"blob {asset.expected_size}\0".encode("ascii"))

        request = Request(url, headers={"User-Agent": USER_AGENT})
        size = 0
        with os.fdopen(fd, "wb") as destination:
            with urlopen(request, timeout=NETWORK_TIMEOUT_SECONDS) as response:
                while True:
                    block = response.read(DOWNLOAD_CHUNK_SIZE)
                    if not block:
                        break
                    size += len(block)
                    sha256.update(block)
                    if git_sha1 is not None:
                        git_sha1.update(block)
                    destination.write(block)
            destination.flush()
            os.fsync(destination.fileno())

        digest = FileDigest(
            size=size,
            sha256=sha256.hexdigest(),
            git_blob_sha1=git_sha1.hexdigest() if git_sha1 is not None else None,
        )
        verify_digest(asset, digest, part_path)
        os.replace(part_path, target_path)
        return digest
    except (HTTPError, URLError, OSError, TimeoutError) as exc:
        raise FontFetchError(
            f"{asset.asset_id}: cannot download '{url}': {exc}"
        ) from exc
    finally:
        if part_path.exists():
            try:
                part_path.unlink()
            except OSError:
                pass


def fetch_asset(
    asset: Asset,
    source_sets: Mapping[str, Any],
    target: Path,
) -> FileDigest:
    target_path = target / asset.install_path
    target_path.parent.mkdir(parents=True, exist_ok=True)
    if target_path.is_symlink():
        raise FontFetchError(
            f"{asset.asset_id}: refusing symlink target '{target_path}'"
        )
    if target_path.exists():
        if not target_path.is_file():
            raise FontFetchError(
                f"{asset.asset_id}: existing target '{target_path}' is not a file"
            )
        digest = file_digest(
            target_path,
            asset.expected_size if asset.expected_git_blob_sha1 is not None else None,
        )
        verify_digest(asset, digest, target_path)
        print(f"verified {asset.install_path}")
        return digest

    url = source_url(source_sets, asset)
    digest = download_asset(asset, url, target_path)
    print(f"downloaded {asset.install_path}")
    return digest


def license_asset(entry: Mapping[str, Any]) -> Asset:
    entry_id = require_string(entry.get("id"), "license_evidence.id")
    install_path = validate_install_path(
        entry.get("install_path"),
        f"license_evidence.{entry_id}.install_path",
        (".txt", ".license", ".md", ""),
    )
    if not install_path.startswith("LICENSES/"):
        raise FontFetchError(
            f"license_evidence.{entry_id}.install_path must be under LICENSES/"
        )
    return Asset(
        asset_id=f"license/{entry_id}",
        source_set="license",
        source_path="",
        install_path=install_path,
        expected_size=0,
        expected_sha256=require_sha256(
            entry.get("sha256"), f"license_evidence.{entry_id}.sha256"
        ),
        expected_git_blob_sha1=None,
        role="license",
    )


def local_license_asset(entry: Mapping[str, Any]) -> Asset:
    entry_id = require_string(entry.get("id"), "local_license_evidence.id")
    source_path = validate_install_path(
        entry.get("source_path"),
        f"local_license_evidence.{entry_id}.source_path",
        (".txt", ".license", ".md", ""),
    )
    install_path = validate_install_path(
        entry.get("install_path"),
        f"local_license_evidence.{entry_id}.install_path",
        (".txt", ".license", ".md", ""),
    )
    if not source_path.startswith("LICENSES/"):
        raise FontFetchError(
            f"local_license_evidence.{entry_id}.source_path must be under LICENSES/"
        )
    if not install_path.startswith("LICENSES/"):
        raise FontFetchError(
            f"local_license_evidence.{entry_id}.install_path must be under LICENSES/"
        )
    return Asset(
        asset_id=f"local-license/{entry_id}",
        source_set="local-license",
        source_path=source_path,
        install_path=install_path,
        expected_size=0,
        expected_sha256=require_sha256(
            entry.get("sha256"), f"local_license_evidence.{entry_id}.sha256"
        ),
        expected_git_blob_sha1=None,
        role="local-license",
    )


def fetch_license(entry: Mapping[str, Any], target: Path) -> None:
    asset = license_asset(entry)
    url = require_string(entry.get("url"), f"{asset.asset_id}.url")
    target_path = target / asset.install_path
    target_path.parent.mkdir(parents=True, exist_ok=True)

    if target_path.is_symlink():
        raise FontFetchError(
            f"{asset.asset_id}: refusing symlink target '{target_path}'"
        )
    if target_path.exists():
        digest = file_digest(target_path, None)
        if digest.sha256 != asset.expected_sha256:
            raise FontFetchError(
                f"{asset.asset_id}: SHA-256 mismatch for existing '{target_path}'"
            )
        print(f"verified {asset.install_path}")
        return

    try:
        fd, part_name = tempfile.mkstemp(
            prefix=f".{target_path.name}.", suffix=".part", dir=target_path.parent
        )
    except OSError as exc:
        raise FontFetchError(
            f"{asset.asset_id}: cannot create temporary license file: {exc}"
        ) from exc
    part_path = Path(part_name)
    try:
        sha256 = hashlib.sha256()
        request = Request(url, headers={"User-Agent": USER_AGENT})
        with os.fdopen(fd, "wb") as destination:
            with urlopen(request, timeout=NETWORK_TIMEOUT_SECONDS) as response:
                while True:
                    block = response.read(DOWNLOAD_CHUNK_SIZE)
                    if not block:
                        break
                    sha256.update(block)
                    destination.write(block)
            destination.flush()
            os.fsync(destination.fileno())
        if sha256.hexdigest() != asset.expected_sha256:
            raise FontFetchError(f"{asset.asset_id}: SHA-256 mismatch")
        os.replace(part_path, target_path)
        print(f"downloaded {asset.install_path}")
    except (HTTPError, URLError, OSError, TimeoutError) as exc:
        raise FontFetchError(
            f"{asset.asset_id}: cannot download '{url}': {exc}"
        ) from exc
    finally:
        if part_path.exists():
            try:
                part_path.unlink()
            except OSError:
                pass


def canonical_license_asset(config: Mapping[str, Any]) -> Asset:
    entry_id = require_string(config.get("id"), "canonical_license.id")
    source_rel = validate_install_path(
        config.get("source_path"), "canonical_license.source_path", (".txt",)
    )
    destination_rel = validate_install_path(
        config.get("install_path"), "canonical_license.install_path", (".txt",)
    )
    if not source_rel.startswith("LICENSES/"):
        raise FontFetchError("canonical_license.source_path must be under LICENSES/")
    if not destination_rel.startswith("LICENSES/"):
        raise FontFetchError("canonical_license.install_path must be under LICENSES/")
    return Asset(
        asset_id=f"canonical-license/{entry_id}",
        source_set="canonical-license",
        source_path=source_rel,
        install_path=destination_rel,
        expected_size=0,
        expected_sha256=require_sha256(
            config.get("sha256"), "canonical_license.sha256"
        ),
        expected_git_blob_sha1=None,
        role="canonical-license",
    )


def copy_tracked_license(
    asset: Asset, source: Path, target: Path, record_description: str
) -> None:
    expected_sha256 = asset.expected_sha256
    if expected_sha256 is None:
        raise FontFetchError(f"{record_description} is missing a SHA-256 digest")
    destination = target / asset.install_path
    if source.is_symlink():
        raise FontFetchError(
            f"{record_description}: refusing symlink source '{source}'"
        )
    if not source.is_file():
        raise FontFetchError(f"{record_description} source '{source}' is missing")
    if file_digest(source, None).sha256 != expected_sha256:
        raise FontFetchError(f"{record_description} source differs from font lock")

    if destination == source:
        print(f"verified {asset.install_path}")
        return
    try:
        destination.parent.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        raise FontFetchError(
            f"{record_description}: cannot create destination directory: {exc}"
        ) from exc
    if destination.is_symlink():
        raise FontFetchError(
            f"{record_description}: refusing symlink target '{destination}'"
        )
    if destination.exists():
        if not destination.is_file():
            raise FontFetchError(
                f"{record_description}: existing target '{destination}' is not a file"
            )
        if file_digest(destination, None).sha256 != expected_sha256:
            raise FontFetchError(
                f"{record_description}: existing '{destination}' differs from source"
            )
        print(f"verified {asset.install_path}")
        return

    try:
        fd, part_name = tempfile.mkstemp(
            prefix=f".{destination.name}.", suffix=".part", dir=destination.parent
        )
    except OSError as exc:
        raise FontFetchError(
            f"{record_description}: cannot create temporary file: {exc}"
        ) from exc
    part_path = Path(part_name)
    try:
        with source.open("rb") as source_file, os.fdopen(fd, "wb") as output:
            shutil.copyfileobj(source_file, output)
            output.flush()
            os.fsync(output.fileno())
        os.replace(part_path, destination)
        print(f"copied {asset.install_path}")
    except OSError as exc:
        raise FontFetchError(f"cannot stage {record_description}: {exc}") from exc
    finally:
        if part_path.exists():
            try:
                part_path.unlink()
            except OSError:
                pass


def copy_canonical_license(lock: Mapping[str, Any], target: Path) -> None:
    config = lock.get("canonical_license")
    if not isinstance(config, dict):
        raise FontFetchError("font lock is missing canonical_license")
    asset = canonical_license_asset(config)
    copy_tracked_license(
        asset, SCRIPT_DIR / asset.source_path, target, "canonical license"
    )


def copy_local_license(
    entry: Mapping[str, Any], target: Path, source_directory: Optional[Path] = None
) -> None:
    asset = local_license_asset(entry)
    source_root = SCRIPT_DIR if source_directory is None else source_directory
    copy_tracked_license(
        asset, source_root / asset.source_path, target, "local license"
    )


def catalog_lock_sha256() -> str:
    return file_digest(LOCK_PATH, None).sha256


def write_attestation(lock: Mapping[str, Any], target: Path) -> None:
    attestation = {
        "schema_version": 1,
        "catalog": lock["catalog"],
        "font_lock_sha256": catalog_lock_sha256(),
        "asset_count": lock["expected_asset_count"],
        "total_bytes": lock["expected_total_bytes"],
    }
    target_path = target / ATTESTATION_NAME
    fd, part_name = tempfile.mkstemp(
        prefix=f".{ATTESTATION_NAME}.", suffix=".part", dir=target
    )
    part_path = Path(part_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as output:
            json.dump(attestation, output, indent=2, sort_keys=True)
            output.write("\n")
            output.flush()
            os.fsync(output.fileno())
        os.replace(part_path, target_path)
    except OSError as exc:
        raise FontFetchError(f"cannot write catalog attestation: {exc}") from exc
    finally:
        if part_path.exists():
            try:
                part_path.unlink()
            except OSError:
                pass


def remove_attestation(target: Path) -> None:
    attestation = target / ATTESTATION_NAME
    if attestation.is_symlink():
        raise FontFetchError(f"refusing symlink attestation '{attestation}'")
    if attestation.exists():
        try:
            attestation.unlink()
        except OSError as exc:
            raise FontFetchError(
                f"cannot remove stale catalog attestation '{attestation}': {exc}"
            ) from exc


def validate_license_references(lock: Mapping[str, Any]) -> None:
    canonical_license = lock.get("canonical_license")
    if not isinstance(canonical_license, dict):
        raise FontFetchError("font lock is missing canonical_license")
    canonical_asset = canonical_license_asset(canonical_license)
    canonical_id = require_string(canonical_license.get("id"), "canonical_license.id")
    source_evidence_ids = {canonical_id}
    record_ids = {canonical_id}
    install_paths = {canonical_asset.install_path}

    licenses = lock.get("license_evidence")
    if not isinstance(licenses, list):
        raise FontFetchError("font lock is missing license_evidence")
    for entry in licenses:
        if not isinstance(entry, dict):
            raise FontFetchError("license_evidence entries must be objects")
        evidence_id = require_string(entry.get("id"), "license_evidence.id")
        if evidence_id in record_ids:
            raise FontFetchError(f"duplicate license evidence ID '{evidence_id}'")
        asset = license_asset(entry)
        require_string(entry.get("url"), f"license_evidence.{evidence_id}.url")
        if asset.install_path in install_paths:
            raise FontFetchError(
                f"duplicate legal-record install path '{asset.install_path}'"
            )
        source_evidence_ids.add(evidence_id)
        record_ids.add(evidence_id)
        install_paths.add(asset.install_path)

    local_licenses = lock.get("local_license_evidence")
    if not isinstance(local_licenses, list):
        raise FontFetchError("font lock is missing local_license_evidence")
    for entry in local_licenses:
        if not isinstance(entry, dict):
            raise FontFetchError("local_license_evidence entries must be objects")
        evidence_id = require_string(entry.get("id"), "local_license_evidence.id")
        if evidence_id in record_ids:
            raise FontFetchError(f"duplicate license evidence ID '{evidence_id}'")
        asset = local_license_asset(entry)
        if asset.install_path in install_paths:
            raise FontFetchError(
                f"duplicate legal-record install path '{asset.install_path}'"
            )
        record_ids.add(evidence_id)
        install_paths.add(asset.install_path)

    source_sets = lock["source_sets"]
    assert isinstance(source_sets, dict)
    for name, source_set in source_sets.items():
        if not isinstance(source_set, dict):
            raise FontFetchError(f"source set '{name}' must be an object")
        evidence_id = require_string(
            source_set.get("license_evidence_id"),
            f"source_sets.{name}.license_evidence_id",
        )
        if evidence_id not in source_evidence_ids:
            raise FontFetchError(
                f"source set '{name}' references unknown license evidence "
                f"'{evidence_id}'"
            )


def fetch_catalog(
    lock: Mapping[str, Any],
    target: Path,
    local_license_source_directory: Optional[Path] = None,
) -> List[Asset]:
    source_sets = lock["source_sets"]
    assert isinstance(source_sets, dict)
    validate_license_references(lock)
    assets = noto_tree_assets(lock) + explicit_assets(lock)
    names = [asset.install_path for asset in assets]
    if len(names) != len(set(names)):
        raise FontFetchError("font lock has duplicate installed filenames")

    expected_count = require_nonnegative_int(
        lock.get("expected_asset_count"), "expected_asset_count"
    )
    expected_total_bytes = require_nonnegative_int(
        lock.get("expected_total_bytes"), "expected_total_bytes"
    )
    if len(assets) != expected_count:
        raise FontFetchError(
            f"font lock expects {expected_count} assets but resolves to {len(assets)}"
        )
    if sum(asset.expected_size for asset in assets) != expected_total_bytes:
        raise FontFetchError("font lock asset bytes do not match expected_total_bytes")

    digests: Dict[str, FileDigest] = {}
    for asset in assets:
        digests[asset.source_path] = fetch_asset(asset, source_sets, target)

    noto_config = lock["noto_tree"]
    assert isinstance(noto_config, dict)
    expected_noto_payload = require_sha256(
        noto_config.get("payload_manifest_sha256"),
        "noto_tree.payload_manifest_sha256",
    )
    noto_payload = payload_manifest_digest(
        (
            asset.source_path,
            digests[asset.source_path].sha256,
            digests[asset.source_path].size,
        )
        for asset in assets
        if asset.source_set == "noto"
    )
    if noto_payload != expected_noto_payload:
        raise FontFetchError("Noto payload SHA-256 manifest differs from lock")

    licenses = lock.get("license_evidence")
    if not isinstance(licenses, list):
        raise FontFetchError("font lock is missing license_evidence")
    for entry in licenses:
        if not isinstance(entry, dict):
            raise FontFetchError("license_evidence entries must be objects")
        fetch_license(entry, target)
    copy_canonical_license(lock, target)
    local_licenses = lock.get("local_license_evidence")
    if not isinstance(local_licenses, list):
        raise FontFetchError("font lock is missing local_license_evidence")
    for entry in local_licenses:
        if not isinstance(entry, dict):
            raise FontFetchError("local_license_evidence entries must be objects")
        copy_local_license(entry, target, local_license_source_directory)
    return assets


def main(argv: Optional[Sequence[str]] = None) -> int:
    args = parse_arguments(argv)
    target = resolve_target(args.target_directory)
    try:
        prepare_target(target)
        lock = load_lock()
        remove_attestation(target)
        assets = fetch_catalog(lock, target)
        write_attestation(lock, target)
    except FontFetchError as exc:
        print(f"font fetch failed: {exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("font fetch interrupted", file=sys.stderr)
        return 130

    total_bytes = sum(asset.expected_size for asset in assets)
    print(f"catalog complete: {len(assets)} assets, {total_bytes} bytes")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
