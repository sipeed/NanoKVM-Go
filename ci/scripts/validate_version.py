#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
KVMCOMM_CHANGELOG = ROOT / "debhelper/packages/kvmcomm/debian/changelog"
NANOKVM_GO_CHANGELOG = ROOT / "debhelper/packages/nanokvm-go/debian/changelog"
TAG_PATTERN = re.compile(r"[0-9]+\.[0-9]+\.[0-9]+")


class ValidationError(RuntimeError):
    pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Validate the NanoKVM-Go Debian package versions and optionally "
            "match them against a release tag."
        )
    )
    parser.add_argument(
        "--tag",
        default=os.environ.get("CI_COMMIT_TAG", ""),
        help="Release tag to validate. Defaults to CI_COMMIT_TAG when set.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("version.env"),
        help="Dotenv output path. Defaults to version.env in the current directory.",
    )
    parser.add_argument(
        "--kvmcomm-changelog",
        type=Path,
        default=KVMCOMM_CHANGELOG,
        help="Path to the kvmcomm Debian changelog.",
    )
    parser.add_argument(
        "--nanokvm-go-changelog",
        type=Path,
        default=NANOKVM_GO_CHANGELOG,
        help="Path to the nanokvm-go Debian changelog.",
    )
    parser.add_argument(
        "--verify-release-context",
        action="store_true",
        help=(
            "For tag pipelines, require a real local Git tag at HEAD, a push "
            "pipeline source."
        ),
    )
    parser.add_argument(
        "--repository",
        type=Path,
        default=ROOT,
        help="Git repository used for release-tag verification.",
    )
    return parser.parse_args()


def read_debian_version(changelog: Path) -> str:
    try:
        result = subprocess.run(
            ["dpkg-parsechangelog", "-l", str(changelog), "-S", "Version"],
            check=False,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError as exc:
        raise ValidationError("dpkg-parsechangelog is not installed") from exc

    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip() or "unknown error"
        raise ValidationError(f"cannot parse {changelog}: {detail}")

    version = result.stdout.strip()
    if not version:
        raise ValidationError(f"{changelog} does not contain a package version")
    return version


def validate_versions(kvmcomm_version: str, nanokvm_go_version: str, tag: str) -> str:
    if kvmcomm_version != nanokvm_go_version:
        raise ValidationError(
            "Debian changelog versions differ: "
            f"kvmcomm={kvmcomm_version}, nanokvm-go={nanokvm_go_version}"
        )

    if tag:
        if TAG_PATTERN.fullmatch(tag) is None:
            raise ValidationError(
                f"invalid release tag {tag!r}; expected x.y.z without a leading v"
            )
        if tag != kvmcomm_version:
            raise ValidationError(
                f"release tag {tag!r} does not match changelog version "
                f"{kvmcomm_version!r}"
            )

    return kvmcomm_version


def git_commit(repository: Path, revision: str) -> str:
    try:
        result = subprocess.run(
            [
                "git",
                "-C",
                str(repository),
                "rev-parse",
                "--verify",
                f"{revision}^{{commit}}",
            ],
            check=False,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError as exc:
        raise ValidationError("git is not installed") from exc
    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip() or "unknown error"
        raise ValidationError(f"cannot resolve Git revision {revision!r}: {detail}")
    return result.stdout.strip()


def verify_release_context(repository: Path, tag: str, pipeline_source: str) -> None:
    if not tag:
        return
    if pipeline_source != "push":
        raise ValidationError(
            f"release tag pipeline source must be 'push', got {pipeline_source!r}"
        )

    head_commit = git_commit(repository, "HEAD")
    tag_commit = git_commit(repository, f"refs/tags/{tag}")
    if tag_commit != head_commit:
        raise ValidationError(
            f"release tag {tag!r} points to {tag_commit}, but HEAD is {head_commit}"
        )


def write_dotenv(output: Path, version: str) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(f"RELEASE_VERSION={version}\n", encoding="utf-8")


def main() -> int:
    args = parse_args()
    try:
        version = validate_versions(
            read_debian_version(args.kvmcomm_changelog),
            read_debian_version(args.nanokvm_go_changelog),
            args.tag,
        )
        if args.verify_release_context:
            verify_release_context(
                args.repository,
                args.tag,
                os.environ.get("CI_PIPELINE_SOURCE", ""),
            )
        write_dotenv(args.output, version)
    except ValidationError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1

    print(f"validated release version {version}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
