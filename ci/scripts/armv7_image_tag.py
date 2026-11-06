#!/usr/bin/env python3
"""Derive the content-addressed tag of the ARMv7 CI build image.

The ARMv7 image bakes a Conan cache built from the component recipes, and the
release build runs with `conan remote disable conancenter`. A dependency that
is not in that baked cache therefore cannot be fetched at build time. The tag
ties a published image to the recipes it was baked from, so it is derived from
those recipes instead of being typed by hand: CI pulls the image with this tag
when it exists and builds it otherwise.

Boundary worth knowing: the tag covers the image recipe and the component
recipes, not the `conan install` flags that */builder.sh passes at release
time. Those can still move a package_id out from under a baked cache. Hashing
the builder scripts whole was rejected because kvm_ui/builder.sh changes often
for unrelated reasons and every such change would force an image rebuild.

Usage:
    armv7_image_tag.py                 print the tag the current tree needs
"""

from __future__ import annotations

import argparse
import hashlib
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCKERFILE = Path("ci/images/armv7/Dockerfile")

# Every file whose content ends up inside the baked Conan cache: the recipe for
# the image itself plus everything it COPYs in. Keeping this list in sync with
# the Dockerfile is enforced below rather than trusted.
HASHED_INPUTS = (
    # Must stay first and must stay present: assert_inputs_cover_dockerfile()
    # reads the Dockerfile to check the rest, and cannot vouch for the
    # Dockerfile itself (it is not one of its own COPY sources).
    DOCKERFILE,
    Path("support/scripts/getconfig.py"),
    Path("support/scripts/nanokvm-go.ini"),
    Path("support/scripts/toolchain_setup_components.sh"),
    Path("libkvm/conanfile.py"),
    Path("kvm_vin/conanfile.py"),
    Path("kvm_ui/conanfile.py"),
    Path("nano_ocr/conanfile.py"),
)

TAG_PREFIX = "img-"
TAG_LENGTH = 12
COPY_PATTERN = re.compile(
    r"^\s*(?:COPY|ADD)\s+(?P<operands>.+?)\s*\\?\s*$", re.IGNORECASE
)


class GuardError(Exception):
    """A condition that must stop the pipeline."""


def read(path: Path) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def dockerfile_copy_sources() -> set[str]:
    """Build-context paths the Dockerfile copies in (ignoring stage copies)."""

    sources: set[str] = set()
    pending: list[str] = []
    for raw in read(DOCKERFILE).splitlines():
        if pending:
            operands, pending = pending + raw.split(), []
            if raw.rstrip().endswith("\\"):
                pending = operands[:-1] if operands[-1] == "\\" else operands
                continue
        else:
            match = COPY_PATTERN.match(raw)
            if match is None:
                continue
            operands = match.group("operands").split()
            if raw.rstrip().endswith("\\"):
                pending = operands
                continue
        operands = [item for item in operands if item != "\\"]
        if any(item.startswith("--from=") for item in operands):
            continue
        sources.update(item for item in operands[:-1] if not item.startswith("--"))
    return sources


def assert_inputs_cover_dockerfile() -> None:
    """Fail if the Dockerfile bakes in something the tag does not hash.

    Only operands that actually resolve inside the tree are judged. Anything
    else -- a glob, a JSON-form operand list, a stray token from a line this
    parser read wrong -- is left alone on purpose: a false positive here turns
    the image job red for every change with no way to satisfy it,
    which is worse than the narrow case it would catch.
    """

    if DOCKERFILE not in HASHED_INPUTS:
        raise GuardError(
            f"{DOCKERFILE} must itself be hashed; nothing else can detect a "
            f"change to it"
        )

    hashed = {path.as_posix() for path in HASHED_INPUTS}
    unhashed_files: list[str] = []
    directories: list[str] = []
    for source in sorted(dockerfile_copy_sources()):
        if source in hashed:
            continue
        resolved = ROOT / source
        if resolved.is_dir():
            directories.append(source)
        elif resolved.is_file():
            unhashed_files.append(source)

    if directories:
        raise GuardError(
            f"{DOCKERFILE} copies whole directories, whose contents the image "
            f"tag cannot hash:\n  "
            + "\n  ".join(directories)
            + f"\nList the files individually in HASHED_INPUTS in "
            f"{Path(__file__).name}, or copy them individually."
        )
    if unhashed_files:
        raise GuardError(
            f"{DOCKERFILE} bakes in files that the image tag does not hash, so a "
            f"change to them would reuse a stale image:\n  "
            + "\n  ".join(unhashed_files)
            + f"\nAdd them to HASHED_INPUTS in {Path(__file__).name}."
        )


def compute_tag() -> str:
    assert_inputs_cover_dockerfile()
    digest = hashlib.sha256()
    for path in sorted(HASHED_INPUTS, key=lambda item: item.as_posix()):
        try:
            blob = (ROOT / path).read_bytes()
        except OSError as error:
            raise GuardError(
                f"{path} is hashed into the image tag but could not be read "
                f"({error.strerror}); the tag is undefined without it"
            ) from error
        # Length-prefix both fields so no rename can collide with a content edit.
        digest.update(f"{path.as_posix()}\0{len(blob)}\0".encode("utf-8"))
        digest.update(blob)
    return f"{TAG_PREFIX}{digest.hexdigest()[:TAG_LENGTH]}"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.parse_args()
    try:
        print(compute_tag())
    except GuardError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
