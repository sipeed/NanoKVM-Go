#!/usr/bin/env python3
"""Build a rootfs tar.gz font package from a verified font catalog."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
import gzip
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path, PurePosixPath
import sys
import tarfile
import tempfile
from types import ModuleType
from typing import Any, Iterable, List, Mapping, Optional, Sequence, Tuple


SCRIPT_DIR = Path(__file__).resolve().parent
LOCK_PATH = SCRIPT_DIR / "font-lock.json"
DEFAULT_OUTPUT_PATH = SCRIPT_DIR / "kvm-ui-fonts.tar.gz"
METADATA_DIRECTORY = ".kvm-ui-fonts"
SHA256SUMS_NAME = f"{METADATA_DIRECTORY}/SHA256SUMS"
INSTALL_STATE_NAME = f"{METADATA_DIRECTORY}/install-state"
STATUS_SCRIPT_NAME = f"{METADATA_DIRECTORY}/status.sh"
UNINSTALL_SCRIPT_NAME = f"{METADATA_DIRECTORY}/uninstall.sh"
VERIFY_SCRIPT_NAME = f"{METADATA_DIRECTORY}/verify.sh"
ROOTFS_FONT_DIRECTORY = PurePosixPath("usr/share/fonts")
SHA256_SHELL_PATTERN = "[0123456789abcdef]" * 64


class FontPackageError(RuntimeError):
    """A recoverable packaging or source-catalog verification error."""


@dataclass(frozen=True)
class ArchiveEntry:
    """One regular file to place below the target font directory."""

    archive_path: str
    sha256: str
    size: int
    source: Optional[Path] = None
    content: Optional[bytes] = None
    mode: int = 0o644


def load_fetcher_module() -> ModuleType:
    """Load the sibling fetcher so validation rules have one owner."""

    module_name = "kvm_ui_font_fetcher_for_packaging"
    existing = sys.modules.get(module_name)
    if existing is not None:
        return existing

    fetcher_path = SCRIPT_DIR / "fetch_fonts.py"
    spec = importlib.util.spec_from_file_location(module_name, fetcher_path)
    if spec is None or spec.loader is None:
        raise FontPackageError(f"cannot load font fetcher '{fetcher_path}'")
    module = importlib.util.module_from_spec(spec)
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


fetch_fonts = load_fetcher_module()


def parse_arguments(argv: Optional[Sequence[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build a rootfs tar.gz package for the kvm_ui font catalog."
    )
    parser.add_argument(
        "--source-directory",
        default=SCRIPT_DIR,
        type=Path,
        help="verified font catalog directory (default: this script's directory)",
    )
    parser.add_argument(
        "--output",
        default=DEFAULT_OUTPUT_PATH,
        type=Path,
        help="tar.gz output path (default: fonts/kvm-ui-fonts.tar.gz)",
    )
    return parser.parse_args(argv)


def resolve_path(path: Path) -> Path:
    return Path(os.path.abspath(path.expanduser()))


def require_regular_file(path: Path, description: str) -> None:
    if path.is_symlink():
        raise FontPackageError(f"{description} '{path}' must not be a symlink")
    if not path.is_file():
        raise FontPackageError(f"{description} '{path}' is missing or not a file")


def require_directory(path: Path, description: str) -> None:
    if path.is_symlink():
        raise FontPackageError(f"{description} '{path}' must not be a symlink")
    if not path.is_dir():
        raise FontPackageError(f"{description} '{path}' is missing or not a directory")


def archive_path_for(value: str, field: str) -> str:
    """Validate one normalized target-relative POSIX path."""

    candidate = PurePosixPath(value)
    if (
        not value
        or candidate.is_absolute()
        or candidate.as_posix() != value
        or any(part in ("", ".", "..") for part in candidate.parts)
        or any(character in value for character in ("\0", "\r", "\n", "\\"))
    ):
        raise FontPackageError(f"{field} is not a safe archive path")
    return value


def rootfs_archive_path(target_relative_path: str) -> str:
    """Map a safe target-relative path to the rootfs-relative tar member name."""

    relative_path = archive_path_for(target_relative_path, "target archive path")
    return (ROOTFS_FONT_DIRECTORY / PurePosixPath(relative_path)).as_posix()


def make_entry(source: Path, archive_path: str) -> ArchiveEntry:
    require_regular_file(source, "catalog file")
    digest = fetch_fonts.file_digest(source, None)
    return ArchiveEntry(
        archive_path=archive_path_for(archive_path, "archive path"),
        sha256=digest.sha256,
        size=digest.size,
        source=source,
    )


def make_generated_entry(archive_path: str, content: bytes, mode: int = 0o644) -> ArchiveEntry:
    if mode not in (0o644, 0o755):
        raise FontPackageError(f"generated entry '{archive_path}' has an unsupported mode")
    return ArchiveEntry(
        archive_path=archive_path_for(archive_path, "generated archive path"),
        sha256=hashlib.sha256(content).hexdigest(),
        size=len(content),
        content=content,
        mode=mode,
    )


def source_file(source_directory: Path, relative_path: str, description: str) -> Path:
    path = source_directory / relative_path
    require_regular_file(path, description)
    return path


def verify_attestation(
    lock: Mapping[str, Any], source_directory: Path, lock_path: Path
) -> ArchiveEntry:
    path = source_directory / fetch_fonts.ATTESTATION_NAME
    require_regular_file(path, "catalog attestation")
    try:
        attestation = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise FontPackageError(f"cannot read catalog attestation '{path}': {exc}") from exc
    if not isinstance(attestation, dict):
        raise FontPackageError("catalog attestation must be a JSON object")

    expected = {
        "schema_version": 1,
        "catalog": lock.get("catalog"),
        "font_lock_sha256": fetch_fonts.file_digest(lock_path, None).sha256,
        "asset_count": lock.get("expected_asset_count"),
        "total_bytes": lock.get("expected_total_bytes"),
    }
    for field, value in expected.items():
        if attestation.get(field) != value:
            raise FontPackageError(
                f"catalog attestation field '{field}' does not match the current lock"
            )
    return make_entry(path, f"{METADATA_DIRECTORY}/{fetch_fonts.ATTESTATION_NAME}")


def explicit_font_entries(
    lock: Mapping[str, Any], source_directory: Path
) -> List[ArchiveEntry]:
    entries: List[ArchiveEntry] = []
    for asset in fetch_fonts.explicit_assets(lock):
        path = source_file(source_directory, asset.install_path, f"asset '{asset.asset_id}'")
        digest = fetch_fonts.file_digest(path, None)
        try:
            fetch_fonts.verify_digest(asset, digest, path)
        except fetch_fonts.FontFetchError as exc:
            raise FontPackageError(str(exc)) from exc
        entries.append(
            ArchiveEntry(
                archive_path=archive_path_for(asset.install_path, "asset install path"),
                sha256=digest.sha256,
                size=digest.size,
                source=path,
            )
        )
    return entries


def noto_font_entries(
    lock: Mapping[str, Any], source_directory: Path
) -> List[ArchiveEntry]:
    config = lock.get("noto_tree")
    if not isinstance(config, dict):
        raise FontPackageError("font lock is missing noto_tree")
    install_subdirectory = fetch_fonts.require_string(
        config.get("install_subdirectory"), "noto_tree.install_subdirectory"
    )
    install_directory = PurePosixPath(install_subdirectory)
    if install_directory.is_absolute() or ".." in install_directory.parts:
        raise FontPackageError("noto_tree.install_subdirectory is not a safe path")

    overrides = config.get("path_install_paths", {})
    if not isinstance(overrides, dict):
        raise FontPackageError("noto_tree.path_install_paths must be an object")

    entries: List[Tuple[str, ArchiveEntry]] = []
    installed_paths = set()
    for source_path, install_path_value in overrides.items():
        source_path = fetch_fonts.validate_install_path(
            source_path,
            "noto_tree.path_install_paths source path",
            (".ttf", ".otf", ".ttc", ".otc"),
        )
        try:
            PurePosixPath(source_path).relative_to("hinted/ttf")
        except ValueError as exc:
            raise FontPackageError(
                f"Noto override source path '{source_path}' has no hinted/ttf prefix"
            ) from exc
        install_path = fetch_fonts.validate_install_path(
            install_path_value,
            f"noto_tree.path_install_paths.{source_path}",
            (".ttf", ".otf", ".ttc", ".otc"),
        )
        path = source_file(source_directory, install_path, f"Noto asset '{source_path}'")
        entry = make_entry(path, install_path)
        entries.append((source_path, entry))
        installed_paths.add(install_path)

    noto_directory = source_directory / install_subdirectory
    require_directory(noto_directory, "Noto catalog directory")
    for path in sorted(noto_directory.rglob("*.ttf")):
        require_regular_file(path, "Noto catalog file")
        install_path = path.relative_to(source_directory).as_posix()
        if install_path in installed_paths:
            continue
        source_path = "hinted/ttf/" + path.relative_to(noto_directory).as_posix()
        entries.append((source_path, make_entry(path, install_path)))
        installed_paths.add(install_path)

    source_paths = [source_path for source_path, _ in entries]
    if len(source_paths) != len(set(source_paths)):
        raise FontPackageError("Noto catalog has duplicate source paths")

    expected_count = fetch_fonts.require_nonnegative_int(
        config.get("expected_count"), "noto_tree.expected_count"
    )
    if len(entries) != expected_count:
        raise FontPackageError(
            f"Noto catalog has {len(entries)} files; expected {expected_count}"
        )
    expected_total_bytes = fetch_fonts.require_nonnegative_int(
        config.get("expected_total_bytes"), "noto_tree.expected_total_bytes"
    )
    actual_total_bytes = sum(entry.size for _, entry in entries)
    if actual_total_bytes != expected_total_bytes:
        raise FontPackageError(
            f"Noto catalog has {actual_total_bytes} bytes; expected {expected_total_bytes}"
        )

    expected_manifest = fetch_fonts.require_sha256(
        config.get("payload_manifest_sha256"), "noto_tree.payload_manifest_sha256"
    )
    actual_manifest = fetch_fonts.payload_manifest_digest(
        (source_path, entry.sha256, entry.size) for source_path, entry in entries
    )
    if actual_manifest != expected_manifest:
        raise FontPackageError(
            "Noto catalog SHA-256 manifest differs from the current font lock"
        )
    return [entry for _, entry in entries]


def license_entries(lock: Mapping[str, Any], source_directory: Path) -> List[ArchiveEntry]:
    entries: List[ArchiveEntry] = []
    licenses = lock.get("license_evidence")
    if not isinstance(licenses, list):
        raise FontPackageError("font lock is missing license_evidence")
    for raw_entry in licenses:
        if not isinstance(raw_entry, dict):
            raise FontPackageError("license_evidence entries must be objects")
        asset = fetch_fonts.license_asset(raw_entry)
        path = source_file(source_directory, asset.install_path, f"license '{asset.asset_id}'")
        digest = fetch_fonts.file_digest(path, None)
        if digest.sha256 != asset.expected_sha256:
            raise FontPackageError(
                f"{asset.asset_id}: SHA-256 mismatch for existing '{path}'"
            )
        entries.append(
            ArchiveEntry(
                archive_path=archive_path_for(asset.install_path, "license install path"),
                sha256=digest.sha256,
                size=digest.size,
                source=path,
            )
        )

    canonical = lock.get("canonical_license")
    if not isinstance(canonical, dict):
        raise FontPackageError("font lock is missing canonical_license")
    install_path = fetch_fonts.validate_install_path(
        canonical.get("install_path"), "canonical_license.install_path", (".txt",)
    )
    expected_sha256 = fetch_fonts.require_sha256(
        canonical.get("sha256"), "canonical_license.sha256"
    )
    path = source_file(source_directory, install_path, "canonical license")
    entry = make_entry(path, install_path)
    if entry.sha256 != expected_sha256:
        raise FontPackageError("canonical license differs from the current font lock")
    entries.append(entry)

    local_licenses = lock.get("local_license_evidence")
    if not isinstance(local_licenses, list):
        raise FontPackageError("font lock is missing local_license_evidence")
    for raw_entry in local_licenses:
        if not isinstance(raw_entry, dict):
            raise FontPackageError("local_license_evidence entries must be objects")
        asset = fetch_fonts.local_license_asset(raw_entry)
        path = source_file(
            source_directory, asset.install_path, f"local license '{asset.asset_id}'"
        )
        digest = fetch_fonts.file_digest(path, None)
        if digest.sha256 != asset.expected_sha256:
            raise FontPackageError(
                f"{asset.asset_id}: SHA-256 mismatch for existing '{path}'"
            )
        entries.append(
            ArchiveEntry(
                archive_path=archive_path_for(
                    asset.install_path, "local license install path"
                ),
                sha256=digest.sha256,
                size=digest.size,
                source=path,
            )
        )
    return entries


def collect_catalog_entries(
    lock: Mapping[str, Any], source_directory: Path
) -> List[ArchiveEntry]:
    entries = noto_font_entries(lock, source_directory) + explicit_font_entries(
        lock, source_directory
    )
    archive_paths = [entry.archive_path for entry in entries]
    if len(archive_paths) != len(set(archive_paths)):
        raise FontPackageError("font catalog has duplicate installed paths")
    expected_count = fetch_fonts.require_nonnegative_int(
        lock.get("expected_asset_count"), "expected_asset_count"
    )
    expected_total_bytes = fetch_fonts.require_nonnegative_int(
        lock.get("expected_total_bytes"), "expected_total_bytes"
    )
    if len(entries) != expected_count:
        raise FontPackageError(
            f"font catalog has {len(entries)} files; expected {expected_count}"
        )
    if sum(entry.size for entry in entries) != expected_total_bytes:
        raise FontPackageError("font catalog bytes do not match expected_total_bytes")
    return entries + license_entries(lock, source_directory)


def sha256sums(entries: Iterable[ArchiveEntry]) -> bytes:
    lines = [
        f"{entry.sha256}  {entry.archive_path}\n"
        for entry in sorted(entries, key=lambda item: item.archive_path)
    ]
    return "".join(lines).encode("ascii")


def verification_plan(entries: Iterable[ArchiveEntry]) -> str:
    """Encode digest/path records without relying on shell field splitting."""

    lines: List[str] = []
    for entry in sorted(entries, key=lambda item: item.archive_path):
        lines.extend((entry.sha256, entry.archive_path))
    return "\n".join(lines)


def normalized_tar_info(archive: tarfile.TarFile, entry: ArchiveEntry) -> tarfile.TarInfo:
    if entry.source is None or entry.content is not None:
        raise FontPackageError(f"archive entry '{entry.archive_path}' has no source file")
    require_regular_file(entry.source, f"archive entry '{entry.archive_path}'")
    info = archive.gettarinfo(
        str(entry.source), arcname=rootfs_archive_path(entry.archive_path)
    )
    if not info.isreg():
        raise FontPackageError(f"archive entry '{entry.archive_path}' is not regular")
    info.uid = 0
    info.gid = 0
    info.uname = ""
    info.gname = ""
    info.mtime = 0
    info.mode = entry.mode
    info.pax_headers = {}
    return info


def generated_tar_info(name: str, size: int, mode: int) -> tarfile.TarInfo:
    info = tarfile.TarInfo(rootfs_archive_path(name))
    info.size = size
    info.uid = 0
    info.gid = 0
    info.uname = ""
    info.gname = ""
    info.mtime = 0
    info.mode = mode
    info.pax_headers = {}
    return info


def add_archive_entry(archive: tarfile.TarFile, entry: ArchiveEntry) -> None:
    if entry.source is not None:
        info = normalized_tar_info(archive, entry)
        with entry.source.open("rb") as source:
            archive.addfile(info, source)
        return
    if entry.content is None:
        raise FontPackageError(f"archive entry '{entry.archive_path}' has no content")
    info = generated_tar_info(entry.archive_path, len(entry.content), entry.mode)
    archive.addfile(info, io.BytesIO(entry.content))


def validate_static_entries(entries: Iterable[ArchiveEntry]) -> List[ArchiveEntry]:
    static_entries = list(entries)
    paths = [entry.archive_path for entry in static_entries]
    if len(paths) != len(set(paths)):
        raise FontPackageError("font package has duplicate static archive paths")
    if INSTALL_STATE_NAME in paths:
        raise FontPackageError("install-state is reserved for archive lifecycle markers")
    for entry in static_entries:
        archive_path_for(entry.archive_path, "static archive path")
        rootfs_archive_path(entry.archive_path)
        if entry.mode not in (0o644, 0o755):
            raise FontPackageError(
                f"archive entry '{entry.archive_path}' has an unsupported mode"
            )
        if (entry.source is None) == (entry.content is None):
            raise FontPackageError(
                f"archive entry '{entry.archive_path}' must have one content source"
            )
    return static_entries


def write_rootfs_archive(path: Path, static_entries: Iterable[ArchiveEntry]) -> None:
    """Write the reproducible rootfs archive with lifecycle state at both ends.

    The gzip stream is pinned to ``mtime=0`` and no original filename (FNAME)
    so the package is byte-identical regardless of the output path or build
    time. A naive ``tarfile.open(path, "w:gz")`` would stamp the wall-clock
    time and the output basename into the gzip header and break reproducibility.
    """

    entries = validate_static_entries(static_entries)
    installing_state = make_generated_entry(INSTALL_STATE_NAME, b"installing\n")
    installed_state = make_generated_entry(INSTALL_STATE_NAME, b"installed\n")
    with path.open("wb") as raw:
        with gzip.GzipFile(
            filename="", mode="wb", compresslevel=9, fileobj=raw, mtime=0
        ) as gz:
            with tarfile.open(
                fileobj=gz, mode="w", format=tarfile.PAX_FORMAT
            ) as archive:
                add_archive_entry(archive, installing_state)
                for entry in sorted(entries, key=lambda item: item.archive_path):
                    add_archive_entry(archive, entry)
                add_archive_entry(archive, installed_state)


def require_machine_value(value: object, field: str) -> str:
    if not isinstance(value, str) or not value:
        raise FontPackageError(f"{field} must be a non-empty string")
    if any(character in value for character in ("\0", "\r", "\n")):
        raise FontPackageError(f"{field} must be a single-line string")
    return value


def shell_quote(value: str) -> str:
    """Return a POSIX shell single-quoted literal for one validated value."""

    return "'" + value.replace("'", "'\"'\"'") + "'"


def render_shell_template(template: str, values: Mapping[str, str]) -> bytes:
    rendered = template
    for name, value in values.items():
        marker = f"@{name}@"
        if marker not in rendered:
            raise FontPackageError(f"helper template marker '{marker}' is missing")
        rendered = rendered.replace(marker, value)
    return rendered.encode("utf-8")


def helper_preamble() -> str:
    return """#!/bin/sh
# Generated by fonts/package_fonts.py. Do not edit this file.
set -u
set -f

emit_error() {
    error_reason=$1
    error_code=$2
    error_message=$3
    printf '%s\\n' 'result=error'
    printf 'reason=%s\\n' "$error_reason"
    printf 'exit_code=%s\\n' "$error_code"
    printf '%s\\n' "$error_message" >&2
    exit "$error_code"
}

emit_success() {
    printf '%s\\n' 'result=success'
    printf '%s\\n' 'reason=success'
    printf '%s\\n' 'exit_code=0'
}

require_command() {
    command -v "$1" >/dev/null 2>&1 ||
        emit_error environment_unavailable 10 "required command is unavailable: $1"
}

safe_target_relative_path() {
    candidate_relative_path=$1
    case "$candidate_relative_path" in
        ""|/*|.|./*|../*|*/../*|*/..|*/./*|*/.|*//*)
            return 1
            ;;
    esac
    return 0
}

valid_sha256() {
    case "$1" in
        @SHA256_PATTERN@) return 0 ;;
        *) return 1 ;;
    esac
}

resolve_layout() {
    require_command dirname
    script_path=$0
    case "$script_path" in
        */*) ;;
        *) emit_error unsafe_or_invalid_metadata 13 "script path must include its directory" ;;
    esac
    case "$script_path" in
        /*) ;;
        *) script_path=$(pwd -P)/$script_path ;;
    esac
    if [ -L "$script_path" ]; then
        emit_error unsafe_or_invalid_metadata 13 "refusing a symbolic-link maintenance script"
    fi
    script_parent=$(dirname "$script_path" 2>/dev/null) ||
        emit_error target_unavailable 11 "cannot inspect maintenance script directory"
    if [ -L "$script_parent" ]; then
        emit_error unsafe_or_invalid_metadata 13 "refusing a symbolic-link control directory"
    fi
    control_directory=$(CDPATH= cd "$script_parent" 2>/dev/null && pwd -P) ||
        emit_error target_unavailable 11 "cannot access maintenance script directory"
    case "$control_directory" in
        */.kvm-ui-fonts) ;;
        *) emit_error unsafe_or_invalid_metadata 13 "maintenance script is outside the control directory" ;;
    esac
    target_candidate=$(dirname "$control_directory" 2>/dev/null) ||
        emit_error target_unavailable 11 "cannot derive font target directory"
    target_directory=$(CDPATH= cd "$target_candidate" 2>/dev/null && pwd -P) ||
        emit_error target_unavailable 11 "cannot access font target directory"
    case "$target_directory" in
        */usr/share/fonts) ;;
        *) emit_error unsafe_or_invalid_metadata 13 "maintenance script is outside usr/share/fonts" ;;
    esac
    if [ ! -d "$target_directory" ] || [ -L "$target_directory" ]; then
        emit_error target_unavailable 11 "font target directory is unavailable"
    fi
}

require_control_file() {
    control_name=$1
    control_path=$control_directory/$control_name
    if [ -L "$control_path" ]; then
        emit_error unsafe_or_invalid_metadata 13 "control path is a symbolic link: $control_name"
    fi
    if [ ! -f "$control_path" ]; then
        emit_error metadata_missing 12 "required control metadata is missing: $control_name"
    fi
}

read_install_state() {
    require_control_file install-state
    install_state=$(cat "$control_directory/install-state") ||
        emit_error target_unavailable 11 "cannot read install state"
    case "$install_state" in
        installing|installed|uninstalling|uninstalled) ;;
        *) emit_error unsafe_or_invalid_metadata 13 "install state is invalid" ;;
    esac
}

ensure_parent_inside_target() {
    candidate_path=$1
    candidate_parent=$(dirname "$candidate_path" 2>/dev/null) || return 1
    while [ ! -d "$candidate_parent" ]; do
        next_parent=$(dirname "$candidate_parent" 2>/dev/null) || return 1
        [ "$next_parent" = "$candidate_parent" ] && return 1
        candidate_parent=$next_parent
    done
    physical_parent=$(CDPATH= cd "$candidate_parent" 2>/dev/null && pwd -P) ||
        return 1
    case "$physical_parent" in
        "$target_directory"|"$target_directory"/*) return 0 ;;
        *) return 1 ;;
    esac
}

first_word() {
    set -- $1
    printf '%s\\n' "$1"
}
"""


def render_status_script(metadata: Mapping[str, str]) -> bytes:
    template = helper_preamble() + """
CATALOG_NAME=@CATALOG_NAME@
LOCK_SHA256=@LOCK_SHA256@
CATALOG_ASSET_COUNT=@CATALOG_ASSET_COUNT@
CATALOG_TOTAL_BYTES=@CATALOG_TOTAL_BYTES@
MANAGED_FILE_COUNT=@MANAGED_FILE_COUNT@
MANAGED_TOTAL_BYTES=@MANAGED_TOTAL_BYTES@

if [ "$#" -ne 0 ]; then
    emit_error usage_error 2 "status.sh accepts no arguments"
fi

require_command cat
resolve_layout
require_control_file font-lock.json
require_control_file catalog-attestation.json
require_control_file SHA256SUMS
require_control_file verify.sh
require_control_file uninstall.sh
require_control_file status.sh
read_install_state

emit_success
printf 'target_directory=%s\\n' "$target_directory"
printf 'install_state=%s\\n' "$install_state"
printf '%s\\n' 'integrity=not_checked'
printf 'catalog=%s\\n' "$CATALOG_NAME"
printf 'font_lock_sha256=%s\\n' "$LOCK_SHA256"
printf 'catalog_asset_count=%s\\n' "$CATALOG_ASSET_COUNT"
printf 'catalog_total_bytes=%s\\n' "$CATALOG_TOTAL_BYTES"
printf 'managed_file_count=%s\\n' "$MANAGED_FILE_COUNT"
printf 'managed_total_bytes=%s\\n' "$MANAGED_TOTAL_BYTES"
"""
    return render_shell_template(
        template,
        {
            "SHA256_PATTERN": SHA256_SHELL_PATTERN,
            "CATALOG_NAME": shell_quote(metadata["catalog"]),
            "LOCK_SHA256": shell_quote(metadata["lock_sha256"]),
            "CATALOG_ASSET_COUNT": shell_quote(metadata["catalog_asset_count"]),
            "CATALOG_TOTAL_BYTES": shell_quote(metadata["catalog_total_bytes"]),
            "MANAGED_FILE_COUNT": shell_quote(metadata["managed_file_count"]),
            "MANAGED_TOTAL_BYTES": shell_quote(metadata["managed_total_bytes"]),
        },
    )


def render_verify_script(expected_entries: Iterable[ArchiveEntry]) -> bytes:
    entries = list(expected_entries)
    template = helper_preamble() + """
EXPECTED_FILE_COUNT=@EXPECTED_FILE_COUNT@

if [ "$#" -ne 0 ]; then
    emit_error usage_error 2 "verify.sh accepts no arguments"
fi

require_command cat
resolve_layout
read_install_state
case "$install_state" in
    uninstalled)
        emit_error catalog_uninstalled 22 "font catalog is marked uninstalled"
        ;;
    installing|uninstalling)
        emit_error catalog_operation_incomplete 23 "font catalog operation is incomplete"
        ;;
    installed) ;;
esac

require_command sha256sum
verified_file_count=0
while IFS= read -r expected_hash && IFS= read -r expected_path; do
    if ! valid_sha256 "$expected_hash" || ! safe_target_relative_path "$expected_path"; then
        emit_error unsafe_or_invalid_metadata 13 "embedded verification metadata is invalid"
    fi
    candidate_path=$target_directory/$expected_path
    if ! ensure_parent_inside_target "$candidate_path"; then
        emit_error unsafe_or_invalid_metadata 13 "verification path escapes the font target"
    fi
    if [ -L "$candidate_path" ]; then
        emit_error unsafe_or_invalid_metadata 13 "verification path is a symbolic link: $expected_path"
    fi
    if [ ! -f "$candidate_path" ]; then
        emit_error managed_file_missing 20 "managed file is missing: $expected_path"
    fi
    actual_digest=$(sha256sum "$candidate_path" 2>/dev/null) ||
        emit_error target_unavailable 11 "cannot hash managed file: $expected_path"
    actual_hash=$(first_word "$actual_digest")
    if ! valid_sha256 "$actual_hash"; then
        emit_error target_unavailable 11 "sha256sum returned invalid output"
    fi
    if [ "$actual_hash" != "$expected_hash" ]; then
        emit_error sha256_mismatch 21 "managed file digest differs: $expected_path"
    fi
    verified_file_count=$((verified_file_count + 1))
done <<'__KVM_UI_FONT_VERIFY_PLAN__'
@VERIFY_PLAN@
__KVM_UI_FONT_VERIFY_PLAN__

if [ "$verified_file_count" -ne "$EXPECTED_FILE_COUNT" ]; then
    emit_error unsafe_or_invalid_metadata 13 "embedded verification metadata is incomplete"
fi

emit_success
printf 'target_directory=%s\\n' "$target_directory"
printf 'install_state=%s\\n' "$install_state"
printf '%s\\n' 'integrity=verified'
printf 'verified_file_count=%s\\n' "$verified_file_count"
"""
    return render_shell_template(
        template,
        {
            "SHA256_PATTERN": SHA256_SHELL_PATTERN,
            "EXPECTED_FILE_COUNT": str(len(entries)),
            "VERIFY_PLAN": verification_plan(entries),
        },
    )


def render_uninstall_script(removal_entries: Iterable[ArchiveEntry]) -> bytes:
    entries = list(removal_entries)
    removal_plan = "\n".join(entry.archive_path for entry in entries)
    template = helper_preamble() + """
EXPECTED_REMOVAL_COUNT=@EXPECTED_REMOVAL_COUNT@

validate_removal_path() {
    if ! safe_target_relative_path "$1"; then
        return 13
    fi
    case "$1" in
        .kvm-ui-fonts|.kvm-ui-fonts/*) return 30 ;;
    esac
    return 0
}

preflight_removal_plan() {
    removal_plan_count=0
    while IFS= read -r removal_path; do
        validate_removal_path "$removal_path"
        validation_code=$?
        if [ "$validation_code" -ne 0 ]; then
            if [ "$validation_code" -eq 30 ]; then
                emit_error uninstall_preflight_refused 30 "removal plan targets the control directory"
            fi
            emit_error unsafe_or_invalid_metadata 13 "removal plan contains an unsafe path"
        fi
        candidate_path=$target_directory/$removal_path
        if ! ensure_parent_inside_target "$candidate_path"; then
            emit_error unsafe_or_invalid_metadata 13 "removal path escapes the font target"
        fi
        if [ -L "$candidate_path" ]; then
            emit_error unsafe_or_invalid_metadata 13 "removal path is a symbolic link: $removal_path"
        fi
        if [ -e "$candidate_path" ] && [ ! -f "$candidate_path" ]; then
            emit_error unsafe_or_invalid_metadata 13 "removal path is not a regular file: $removal_path"
        fi
        removal_plan_count=$((removal_plan_count + 1))
    done <<'__KVM_UI_FONT_REMOVAL_PLAN__'
@REMOVAL_PLAN@
__KVM_UI_FONT_REMOVAL_PLAN__
    if [ "$removal_plan_count" -ne "$EXPECTED_REMOVAL_COUNT" ]; then
        emit_error unsafe_or_invalid_metadata 13 "removal plan is incomplete"
    fi
}

count_removal_paths() {
    planned_count=0
    removed_count=0
    missing_count=0
    failed_count=0
    while IFS= read -r removal_path; do
        candidate_path=$target_directory/$removal_path
        planned_count=$((planned_count + 1))
        if [ ! -e "$candidate_path" ]; then
            missing_count=$((missing_count + 1))
        fi
    done <<'__KVM_UI_FONT_REMOVAL_PLAN__'
@REMOVAL_PLAN@
__KVM_UI_FONT_REMOVAL_PLAN__
}

write_install_state() {
    next_state=$1
    state_temp=$(umask 077 && mktemp "$control_directory/.install-state.XXXXXX") ||
        emit_error target_unavailable 11 "cannot create temporary install state"
    case "$state_temp" in
        "$control_directory"/.install-state.*) ;;
        *) emit_error unsafe_or_invalid_metadata 13 "temporary install state escaped control directory" ;;
    esac
    if ! printf '%s\\n' "$next_state" > "$state_temp"; then
        rm -f "$state_temp" >/dev/null 2>&1 || :
        emit_error target_unavailable 11 "cannot write temporary install state"
    fi
    if ! chmod 0644 "$state_temp"; then
        rm -f "$state_temp" >/dev/null 2>&1 || :
        emit_error target_unavailable 11 "cannot set install state permissions"
    fi
    if ! mv -f "$state_temp" "$control_directory/install-state"; then
        rm -f "$state_temp" >/dev/null 2>&1 || :
        emit_error target_unavailable 11 "cannot replace install state"
    fi
}

remove_payload_paths() {
    planned_count=0
    removed_count=0
    missing_count=0
    failed_count=0
    while IFS= read -r removal_path; do
        candidate_path=$target_directory/$removal_path
        planned_count=$((planned_count + 1))
        if [ -e "$candidate_path" ]; then
            if rm -f "$candidate_path"; then
                removed_count=$((removed_count + 1))
            else
                failed_count=$((failed_count + 1))
            fi
        else
            missing_count=$((missing_count + 1))
        fi
    done <<'__KVM_UI_FONT_REMOVAL_PLAN__'
@REMOVAL_PLAN@
__KVM_UI_FONT_REMOVAL_PLAN__
}

prune_empty_payload_directories() {
    while IFS= read -r removal_path; do
        payload_parent=$(dirname "$target_directory/$removal_path")
        while [ "$payload_parent" != "$target_directory" ]; do
            rmdir "$payload_parent" >/dev/null 2>&1 || break
            payload_parent=$(dirname "$payload_parent")
        done
    done <<'__KVM_UI_FONT_REMOVAL_PLAN__'
@REMOVAL_PLAN@
__KVM_UI_FONT_REMOVAL_PLAN__
}

emit_uninstall_result() {
    operation_name=$1
    result_state=$2
    emit_success
    printf 'operation=%s\\n' "$operation_name"
    printf 'target_directory=%s\\n' "$target_directory"
    printf 'install_state=%s\\n' "$result_state"
    printf 'planned_count=%s\\n' "$planned_count"
    printf 'removed_count=%s\\n' "$removed_count"
    printf 'missing_count=%s\\n' "$missing_count"
    printf 'failed_count=%s\\n' "$failed_count"
}

emit_uninstall_delete_failure() {
    printf '%s\\n' 'result=error'
    printf '%s\\n' 'reason=uninstall_delete_failed'
    printf '%s\\n' 'exit_code=31'
    printf 'operation=uninstall\\n'
    printf 'target_directory=%s\\n' "$target_directory"
    printf '%s\\n' 'install_state=uninstalling'
    printf 'planned_count=%s\\n' "$planned_count"
    printf 'removed_count=%s\\n' "$removed_count"
    printf 'missing_count=%s\\n' "$missing_count"
    printf 'failed_count=%s\\n' "$failed_count"
    printf '%s\\n' 'one or more managed files could not be removed' >&2
    exit 31
}

case "$#" in
    1)
        case "$1" in
            --dry-run) operation=dry_run ;;
            --yes) operation=uninstall ;;
            *) emit_error usage_error 2 "uninstall.sh requires --dry-run or --yes" ;;
        esac
        ;;
    *) emit_error usage_error 2 "uninstall.sh requires --dry-run or --yes" ;;
esac

require_command cat
require_command chmod
require_command dirname
require_command mktemp
require_command mv
require_command rm
require_command rmdir
resolve_layout
read_install_state
preflight_removal_plan

if [ "$operation" = dry_run ]; then
    count_removal_paths
    emit_uninstall_result dry_run "$install_state"
    exit 0
fi

write_install_state uninstalling
remove_payload_paths
if [ "$failed_count" -ne 0 ]; then
    emit_uninstall_delete_failure
fi
prune_empty_payload_directories
write_install_state uninstalled
emit_uninstall_result uninstall uninstalled
"""
    return render_shell_template(
        template,
        {
            "SHA256_PATTERN": SHA256_SHELL_PATTERN,
            "EXPECTED_REMOVAL_COUNT": str(len(entries)),
            "REMOVAL_PLAN": removal_plan,
        },
    )


def catalog_metadata(
    lock: Mapping[str, Any], lock_entry: ArchiveEntry, payload_entries: Iterable[ArchiveEntry]
) -> Mapping[str, str]:
    entries = list(payload_entries)
    catalog_name = require_machine_value(lock.get("catalog"), "catalog")
    asset_count = fetch_fonts.require_nonnegative_int(
        lock.get("expected_asset_count"), "expected_asset_count"
    )
    total_bytes = fetch_fonts.require_nonnegative_int(
        lock.get("expected_total_bytes"), "expected_total_bytes"
    )
    return {
        "catalog": catalog_name,
        "lock_sha256": lock_entry.sha256,
        "catalog_asset_count": str(asset_count),
        "catalog_total_bytes": str(total_bytes),
        "managed_file_count": str(len(entries)),
        "managed_total_bytes": str(sum(entry.size for entry in entries)),
    }


def prepare_output_path(output_path: Path) -> None:
    parent = output_path.parent
    try:
        parent.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        raise FontPackageError(
            f"cannot create output directory '{parent}': {exc}"
        ) from exc
    require_directory(parent, "output directory")


def build_archive(
    source_directory: Path,
    output_path: Path,
    lock_path: Path = LOCK_PATH,
) -> None:
    """Validate a catalog and atomically write its rootfs tar.gz package."""

    source_directory = resolve_path(source_directory)
    output_path = resolve_path(output_path)
    lock_path = resolve_path(lock_path)
    require_directory(source_directory, "source directory")
    require_regular_file(lock_path, "font lock")
    if output_path.suffixes[-2:] != [".tar", ".gz"]:
        raise FontPackageError("archive output must end with '.tar.gz'")
    if output_path.is_symlink():
        raise FontPackageError(f"archive output '{output_path}' must not be a symlink")
    if output_path.exists() and not output_path.is_file():
        raise FontPackageError(f"archive output '{output_path}' is not a regular file")

    try:
        lock = fetch_fonts.load_lock(lock_path)
        fetch_fonts.validate_license_references(lock)
        payload_entries = collect_catalog_entries(lock, source_directory)
        attestation_entry = verify_attestation(lock, source_directory, lock_path)
        lock_entry = make_entry(lock_path, f"{METADATA_DIRECTORY}/font-lock.json")
        metadata = catalog_metadata(lock, lock_entry, payload_entries)

        status_entry = make_generated_entry(
            STATUS_SCRIPT_NAME, render_status_script(metadata), mode=0o755
        )
        uninstall_entry = make_generated_entry(
            UNINSTALL_SCRIPT_NAME,
            render_uninstall_script(payload_entries),
            mode=0o755,
        )
        manifest_entries = payload_entries + [
            attestation_entry,
            lock_entry,
            status_entry,
            uninstall_entry,
        ]
        manifest = sha256sums(manifest_entries)
        sums_entry = make_generated_entry(SHA256SUMS_NAME, manifest)
        verify_entry = make_generated_entry(
            VERIFY_SCRIPT_NAME,
            render_verify_script(manifest_entries + [sums_entry]),
            mode=0o755,
        )
        static_entries = manifest_entries + [sums_entry, verify_entry]
        validate_static_entries(static_entries)
    except fetch_fonts.FontFetchError as exc:
        raise FontPackageError(str(exc)) from exc

    prepare_output_path(output_path)
    output_file: Optional[Path] = None
    try:
        output_fd, output_name = tempfile.mkstemp(
            prefix=f".{output_path.name}.", suffix=".part", dir=output_path.parent
        )
        os.close(output_fd)
        output_file = Path(output_name)
        write_rootfs_archive(output_file, static_entries)
        with output_file.open("rb") as output:
            os.fsync(output.fileno())
        os.chmod(output_file, 0o644)
        os.replace(output_file, output_path)
        output_file = None
    except (OSError, tarfile.TarError) as exc:
        raise FontPackageError(f"cannot write archive '{output_path}': {exc}") from exc
    finally:
        if output_file is not None and output_file.exists():
            try:
                output_file.unlink()
            except OSError:
                pass


def main(argv: Optional[Sequence[str]] = None) -> int:
    args = parse_arguments(argv)
    try:
        build_archive(args.source_directory, args.output)
    except FontPackageError as exc:
        print(f"font package failed: {exc}", file=sys.stderr)
        return 1
    print(f"font package written to {resolve_path(args.output)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
