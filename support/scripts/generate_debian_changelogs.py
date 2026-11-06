#!/usr/bin/env python3
"""Generate synchronized Debian changelog entries with git-cliff."""

from __future__ import annotations

import argparse
import email.utils
import os
import re
import shlex
import shutil
import stat
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Sequence


ROOT = Path(__file__).resolve().parents[2]
CONFIG_PATH = Path(__file__).with_name("git-cliff-debian.toml")
VERSION_PATTERN = re.compile(r"[0-9]+\.[0-9]+\.[0-9]+")
DEBIAN_TOKEN_PATTERN = re.compile(r"[A-Za-z0-9][A-Za-z0-9.+~-]*")
MINIMUM_GIT_CLIFF_VERSION = (2, 0, 0)

SHARED_PATHS = (
    "build.py",
    ".github/**",
    "debhelper/docker/**",
    "support/scripts/**",
    "support/toolchains/**",
    "support/README.md",
)


@dataclass(frozen=True)
class PackageSpec:
    name: str
    directory: Path
    paths: tuple[str, ...]

    @property
    def changelog(self) -> Path:
        return self.directory / "debian" / "changelog"

    @property
    def control(self) -> Path:
        return self.directory / "debian" / "control"


PACKAGES = (
    PackageSpec(
        name="kvmcomm",
        directory=ROOT / "debhelper" / "packages" / "kvmcomm",
        paths=(
            "debhelper/packages/kvmcomm/**",
            "kvm_vin/**",
            "kvm_ui/**",
            "nano_ocr/**",
        ),
    ),
    PackageSpec(
        name="nanokvm-go",
        directory=ROOT / "debhelper" / "packages" / "nanokvm-go",
        paths=(
            "debhelper/packages/nanokvm-go/**",
            "server/**",
            "web/**",
            "libkvm/**",
            "memfab/**",
        ),
    ),
)


class ChangelogError(RuntimeError):
    pass


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Generate and prepend synchronized kvmcomm and nanokvm-go Debian "
            "changelog entries with git-cliff."
        )
    )
    parser.add_argument("--version", required=True, help="new Debian package version")
    parser.add_argument(
        "--from",
        dest="from_ref",
        help="starting Git ref (default: current common changelog version)",
    )
    parser.add_argument("--to", default="HEAD", help="ending Git ref (default: HEAD)")
    parser.add_argument(
        "--distribution", default="unstable", help="Debian distribution (default: unstable)"
    )
    parser.add_argument(
        "--urgency",
        choices=("low", "medium", "high", "critical", "emergency"),
        default="medium",
        help="Debian urgency (default: medium)",
    )
    parser.add_argument(
        "--all-commits",
        action="store_true",
        help="use the complete range for both packages instead of package path filters",
    )
    parser.add_argument(
        "--allow-dirty",
        action="store_true",
        help="allow generation from a worktree with uncommitted changes",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="generate and validate candidates, then print them without replacing files",
    )
    parser.add_argument(
        "--git-cliff",
        default=os.environ.get("GIT_CLIFF", "git-cliff"),
        help="git-cliff executable (default: git-cliff or $GIT_CLIFF)",
    )
    return parser.parse_args(argv)


def resolve_tool(name: str) -> str:
    resolved = shutil.which(name)
    if resolved:
        return resolved

    candidate = Path(name)
    if not candidate.is_absolute():
        candidate = ROOT / candidate
    if candidate.is_file() and os.access(candidate, os.X_OK):
        return str(candidate)
    raise ChangelogError(f"required command not found: {name}")


def run(
    command: Sequence[str],
    *,
    cwd: Path = ROOT,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        [str(part) for part in command],
        cwd=cwd,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if check and result.returncode:
        detail = result.stderr.strip() or result.stdout.strip()
        message = f"command failed ({result.returncode}): {shlex.join(command)}"
        if detail:
            message += f"\n{detail}"
        raise ChangelogError(message)
    return result


def control_field(path: Path, field: str) -> str:
    prefix = f"{field}:"
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith(prefix):
            return line[len(prefix) :].strip()
        if not line.strip():
            break
    raise ChangelogError(f"missing {field} in {path.relative_to(ROOT)}")


def changelog_field(dpkg_parsechangelog: str, path: Path, field: str) -> str:
    result = run(
        (dpkg_parsechangelog, "-l", str(path), "-S", field),
    )
    return result.stdout.strip()


def check_worktree_clean(git: str) -> None:
    result = run((git, "status", "--porcelain=v1", "--untracked-files=all"))
    changes = result.stdout.rstrip()
    if not changes:
        return

    summary = "\n".join(changes.splitlines()[:10])
    if len(changes.splitlines()) > 10:
        summary += "\n..."
    raise ChangelogError(
        "refusing to generate a release changelog from a dirty worktree; "
        "commit or stash these changes, or use --allow-dirty:\n"
        f"{summary}"
    )


def resolve_ref(git: str, ref: str) -> str:
    result = run(
        (git, "rev-parse", "--verify", "--end-of-options", f"{ref}^{{commit}}")
    )
    return result.stdout.strip()


def validate_git_cliff_version(git_cliff: str) -> None:
    result = run((git_cliff, "--version"))
    match = re.fullmatch(r"git-cliff (\d+)\.(\d+)\.(\d+)", result.stdout.strip())
    if not match:
        raise ChangelogError(
            f"cannot parse git-cliff version: {result.stdout.strip()!r}"
        )
    version = tuple(int(part) for part in match.groups())
    if version < MINIMUM_GIT_CLIFF_VERSION:
        required = ".".join(str(part) for part in MINIMUM_GIT_CLIFF_VERSION)
        raise ChangelogError(f"git-cliff {required} or newer is required")


def generate_entries(
    git_cliff: str,
    version: str,
    revision_range: str,
    paths: Sequence[str],
) -> str:
    command = [
        git_cliff,
        "--config",
        str(CONFIG_PATH),
        "--tag",
        version,
    ]
    for path in paths:
        command.extend(("--include-path", path))
    command.extend(("--", revision_range))

    result = run(command)
    entries: list[str] = []
    for line in result.stdout.splitlines():
        marker = line.strip()
        if not marker:
            continue
        if marker.startswith("@@GROUP@@"):
            entries.append(f"  * {marker.removeprefix('@@GROUP@@')}:")
        elif marker.startswith("@@ITEM@@"):
            entries.append(f"    - {marker.removeprefix('@@ITEM@@')}")
        else:
            raise ChangelogError(f"unexpected git-cliff output line: {line!r}")

    if not entries:
        scope = ", ".join(paths) if paths else "the complete repository"
        raise ChangelogError(
            f"git-cliff produced no changelog entries for {scope} in {revision_range}"
        )
    return "\n".join(entries)


def render_changelog(
    spec: PackageSpec,
    *,
    version: str,
    distribution: str,
    urgency: str,
    maintainer: str,
    release_date: str,
    entries: str,
    previous: str,
) -> str:
    return (
        f"{spec.name} ({version}) {distribution}; urgency={urgency}\n\n"
        f"{entries}\n\n"
        f" -- {maintainer}  {release_date}\n\n"
        f"{previous}"
    )


def validate_candidate(
    dpkg_parsechangelog: str,
    candidate: Path,
    *,
    spec: PackageSpec,
    version: str,
    distribution: str,
    urgency: str,
    maintainer: str,
) -> None:
    expected = {
        "Source": spec.name,
        "Version": version,
        "Distribution": distribution,
        "Urgency": urgency,
        "Maintainer": maintainer,
    }
    for field, value in expected.items():
        actual = changelog_field(dpkg_parsechangelog, candidate, field)
        if actual != value:
            raise ChangelogError(
                f"generated {spec.name} changelog has {field}={actual!r}, "
                f"expected {value!r}"
            )


def write_temporary_changelog(spec: PackageSpec, content: str, mode: int) -> Path:
    path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            prefix=".changelog.",
            dir=spec.changelog.parent,
            delete=False,
        ) as temporary:
            path = Path(temporary.name)
            temporary.write(content)
        path.chmod(mode)
        return path
    except BaseException:
        if path is not None and path.exists():
            path.unlink()
        raise


def restore_original(spec: PackageSpec, data: bytes, mode: int) -> None:
    path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="wb",
            prefix=".changelog.restore.",
            dir=spec.changelog.parent,
            delete=False,
        ) as temporary:
            path = Path(temporary.name)
            temporary.write(data)
        path.chmod(mode)
        os.replace(path, spec.changelog)
    finally:
        if path is not None and path.exists():
            path.unlink()


def replace_candidates(
    candidates: dict[PackageSpec, Path],
    originals: dict[PackageSpec, tuple[bytes, int]],
) -> None:
    for spec, (data, _) in originals.items():
        if spec.changelog.read_bytes() != data:
            raise ChangelogError(
                f"{spec.changelog.relative_to(ROOT)} changed during generation"
            )

    replaced: list[PackageSpec] = []
    try:
        for spec, candidate in candidates.items():
            os.replace(candidate, spec.changelog)
            replaced.append(spec)
    except BaseException as error:
        rollback_errors: list[str] = []
        for spec in reversed(replaced):
            data, mode = originals[spec]
            try:
                restore_original(spec, data, mode)
            except BaseException as rollback_error:
                rollback_errors.append(f"{spec.name}: {rollback_error}")
        if rollback_errors:
            detail = "; ".join(rollback_errors)
            raise ChangelogError(
                f"failed to replace changelogs and rollback was incomplete: {detail}"
            ) from error
        raise ChangelogError(f"failed to replace changelog files: {error}") from error


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv if argv is not None else sys.argv[1:])
    git = resolve_tool("git")
    dpkg = resolve_tool("dpkg")
    dpkg_parsechangelog = resolve_tool("dpkg-parsechangelog")
    git_cliff = resolve_tool(args.git_cliff)
    validate_git_cliff_version(git_cliff)

    if not CONFIG_PATH.is_file():
        raise ChangelogError(f"missing git-cliff config: {CONFIG_PATH.relative_to(ROOT)}")

    if not VERSION_PATTERN.fullmatch(args.version):
        raise ChangelogError("--version must use the numeric X.Y.Z release tag format")
    if not DEBIAN_TOKEN_PATTERN.fullmatch(args.distribution):
        raise ChangelogError(f"invalid Debian distribution: {args.distribution!r}")
    run((dpkg, "--validate-version", args.version))

    current_versions: dict[PackageSpec, str] = {}
    maintainers: dict[PackageSpec, str] = {}
    for spec in PACKAGES:
        if not spec.changelog.is_file() or not spec.control.is_file():
            raise ChangelogError(f"missing Debian metadata for {spec.name}")
        if changelog_field(dpkg_parsechangelog, spec.changelog, "Source") != spec.name:
            raise ChangelogError(f"unexpected source in {spec.changelog.relative_to(ROOT)}")

        current_versions[spec] = changelog_field(
            dpkg_parsechangelog, spec.changelog, "Version"
        )
        maintainers[spec] = control_field(spec.control, "Maintainer")
        current_maintainer = changelog_field(
            dpkg_parsechangelog, spec.changelog, "Maintainer"
        )
        if current_maintainer != maintainers[spec]:
            raise ChangelogError(
                f"Maintainer mismatch for {spec.name}: changelog={current_maintainer!r}, "
                f"control={maintainers[spec]!r}"
            )

    version_values = set(current_versions.values())
    if len(version_values) != 1:
        versions = ", ".join(
            f"{spec.name}={version}" for spec, version in current_versions.items()
        )
        raise ChangelogError(f"package changelog versions are not synchronized: {versions}")
    current_version = next(iter(version_values))

    comparison = run(
        (dpkg, "--compare-versions", current_version, "lt", args.version),
        check=False,
    )
    if comparison.returncode != 0:
        raise ChangelogError(
            f"new version {args.version!r} must be greater than current version "
            f"{current_version!r}"
        )

    originals = {
        spec: (spec.changelog.read_bytes(), stat.S_IMODE(spec.changelog.stat().st_mode))
        for spec in PACKAGES
    }

    if not args.dry_run and not args.allow_dirty:
        check_worktree_clean(git)

    from_ref = args.from_ref or current_version
    from_oid = resolve_ref(git, from_ref)
    to_oid = resolve_ref(git, args.to)
    ancestry = run(
        (git, "merge-base", "--is-ancestor", from_oid, to_oid),
        check=False,
    )
    if ancestry.returncode != 0:
        if ancestry.returncode == 1:
            raise ChangelogError(f"{from_ref!r} is not an ancestor of {args.to!r}")
        detail = ancestry.stderr.strip() or "unknown git error"
        raise ChangelogError(f"cannot validate revision range: {detail}")
    revision_range = f"{from_oid}..{to_oid}"

    release_date = email.utils.format_datetime(datetime.now().astimezone())
    generated: dict[PackageSpec, str] = {}
    for spec in PACKAGES:
        paths = () if args.all_commits else spec.paths + SHARED_PATHS
        entries = generate_entries(git_cliff, args.version, revision_range, paths)
        generated[spec] = render_changelog(
            spec,
            version=args.version,
            distribution=args.distribution,
            urgency=args.urgency,
            maintainer=maintainers[spec],
            release_date=release_date,
            entries=entries,
            previous=originals[spec][0].decode("utf-8"),
        )

    candidates: dict[PackageSpec, Path] = {}
    try:
        for spec, content in generated.items():
            candidate = write_temporary_changelog(
                spec, content, originals[spec][1]
            )
            candidates[spec] = candidate
            validate_candidate(
                dpkg_parsechangelog,
                candidate,
                spec=spec,
                version=args.version,
                distribution=args.distribution,
                urgency=args.urgency,
                maintainer=maintainers[spec],
            )
        if args.dry_run:
            for spec in PACKAGES:
                print(f"===== {spec.changelog.relative_to(ROOT)} =====")
                print(generated[spec], end="")
            return 0

        replace_candidates(candidates, originals)
    finally:
        for candidate in candidates.values():
            if candidate.exists():
                candidate.unlink()

    print(
        f"Generated Debian changelogs for {args.version} from {revision_range}:"
    )
    for spec in PACKAGES:
        print(f"  {spec.changelog.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (ChangelogError, OSError, UnicodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
