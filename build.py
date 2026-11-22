#!/usr/bin/env python3
from __future__ import annotations

import argparse
import configparser
import json
from contextlib import contextmanager
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Iterator, Mapping, Sequence

from support.scripts.update_signing import (
    key_id as update_key_id,
    public_key_from_private,
    public_key_from_public,
    write_keyring,
)


ROOT = Path(__file__).resolve().parent
BUILD_DIR = ROOT / "build"
DEBHELPER_PACKAGES_DIR = ROOT / "debhelper" / "packages"
KVMCOMM_PACKAGE_DIR = DEBHELPER_PACKAGES_DIR / "kvmcomm"
NANOKVM_GO_PACKAGE_DIR = DEBHELPER_PACKAGES_DIR / "nanokvm-go"
APPS_GITLINK_PATH = Path("debhelper/packages/kvmcomm/rootfs/kvmcomm/apps")
APPS_DIR = ROOT / APPS_GITLINK_PATH
SUPPORT_SCRIPTS_DIR = ROOT / "support" / "scripts"
SUPPORT_TOOLCHAINS_DIR = ROOT / "support" / "toolchains"
DOCKER_PACKAGE_SCRIPT = ROOT / "debhelper" / "docker" / "trixie" / "build_in_docker.sh"
NANOKVM_GO_INI = SUPPORT_SCRIPTS_DIR / "nanokvm-go.ini"
UPDATE_VERIFIER = BUILD_DIR / "nanokvm-update-verify"
UPDATE_KEYRING = BUILD_DIR / "update-keys.json"

BUILD_SEQUENCE = ("libkvm", "kvm_vin", "kvm_ui", "nano_ocr", "webcam", "memfab", "server", "web")
DPKG_BUILDPACKAGE = ("dpkg-buildpackage", "-d", "-us", "-uc", "-b", "-aarmhf")
PACKAGE_ARTIFACT_SUFFIXES = ("deb", "changes", "buildinfo")
UPDATE_VERSION_PATTERN = re.compile(r"[0-9]+\.[0-9]+\.[0-9]+")

KVMCOMM_INPUT_FILES = (
    BUILD_DIR / "nomad-uvc-feeder",
    BUILD_DIR / "kvm_vin",
    BUILD_DIR / "kvm_ui",
    BUILD_DIR / "nano_ocr_app",
    UPDATE_VERIFIER,
    UPDATE_KEYRING,
    KVMCOMM_PACKAGE_DIR / "rootfs" / "usr" / "bin" / "ax_efuse_tool",
    KVMCOMM_PACKAGE_DIR / "rootfs" / "kvmcomm" / "models" / "pp_ocr" / "ch_PP_OCRv3_det_npu.axmodel",
    KVMCOMM_PACKAGE_DIR / "rootfs" / "kvmcomm" / "models" / "pp_ocr" / "ch_PP_OCRv4_rec_npu.axmodel",
    KVMCOMM_PACKAGE_DIR / "rootfs" / "kvmcomm" / "models" / "pp_ocr" / "ppocr_keys_v1.txt",
)
NANOKVM_GO_INPUT_FILES = (
    BUILD_DIR / "NanoKVM-Server",
    BUILD_DIR / "libkvm.so",
    BUILD_DIR / "libkvm.so.0",
    BUILD_DIR / "libkvm.so.0.1.0",
    BUILD_DIR / "memfab",
)
NANOKVM_GO_INPUT_DIRS = (BUILD_DIR / "web",)


def log(target: str, message: str) -> None:
    print(f"[{target}] {message}", flush=True)


def format_path(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def format_command(command: Sequence[str]) -> str:
    return shlex.join(str(part) for part in command)


def run(
    target: str,
    command: Sequence[str],
    *,
    cwd: Path | None = None,
    env: Mapping[str, str] | None = None,
    capture_output: bool = False,
) -> subprocess.CompletedProcess[str]:
    workdir = cwd or ROOT
    process_env = os.environ.copy()
    if env is not None:
        process_env.update(env)

    log(target, f"run: {format_command(command)}")
    log(target, f"cwd: {format_path(workdir)}")
    return subprocess.run(
        [str(part) for part in command],
        cwd=workdir,
        env=process_env,
        check=True,
        text=True,
        capture_output=capture_output,
    )


def conan_default_profile() -> Path:
    conan_home = os.environ.get("CONAN_HOME")
    if conan_home:
        return Path(conan_home).expanduser() / "profiles" / "default"

    return Path.home() / ".conan2" / "profiles" / "default"


def ensure_conan_default_profile(target: str) -> None:
    default_profile = conan_default_profile()
    if default_profile.is_file():
        return

    run(target, ("conan", "profile", "detect", "--force"))


def require_file(path: Path, label: str | None = None) -> Path:
    if not path.is_file():
        name = label or format_path(path)
        raise SystemExit(f"missing required file: {name}")
    return path


def require_dir(path: Path, label: str | None = None) -> Path:
    if not path.is_dir():
        name = label or format_path(path)
        raise SystemExit(f"missing required directory: {name}")
    return path


def copy_file(target: str, source: Path, destination: Path) -> None:
    require_file(source)
    destination.parent.mkdir(parents=True, exist_ok=True)
    remove_path(target, destination)
    shutil.copy2(source, destination, follow_symlinks=False)
    log(target, f"artifact: {format_path(destination)}")


def copy_tree(target: str, source: Path, destination: Path) -> None:
    require_dir(source)
    if destination.exists():
        remove_path(target, destination)
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(source, destination, symlinks=True)
    log(target, f"artifact: {format_path(destination)}")


def copy_glob(target: str, pattern: str, destination_dir: Path) -> None:
    sources = sorted(ROOT.glob(pattern))
    if not sources:
        raise SystemExit(f"missing required artifact: {pattern}")

    for source in sources:
        copy_file(target, source, destination_dir / source.name)


def move_file(target: str, source: Path, destination: Path) -> None:
    require_file(source)
    destination.parent.mkdir(parents=True, exist_ok=True)
    remove_path(target, destination)
    shutil.move(str(source), str(destination))
    log(target, f"artifact: {format_path(destination)}")


def remove_path(target: str, path: Path) -> None:
    if not path.exists() and not path.is_symlink():
        return

    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    else:
        path.unlink()

    log(target, f"removed: {format_path(path)}")


def remove_globs(target: str, base: Path, patterns: Sequence[str]) -> None:
    if not base.exists():
        return

    for pattern in patterns:
        for path in sorted(base.glob(pattern)):
            remove_path(target, path)


def should_skip_python_cache_path(path: Path) -> bool:
    skipped_roots = (ROOT / ".git", SUPPORT_TOOLCHAINS_DIR)
    return any(path == root or root in path.parents for root in skipped_roots)


def remove_python_caches(target: str) -> None:
    for path in sorted(
        ROOT.rglob("__pycache__"),
        key=lambda item: len(item.parts),
        reverse=True,
    ):
        if path.is_dir() and not should_skip_python_cache_path(path):
            remove_path(target, path)

    for pattern in ("*.pyc", "*.pyo"):
        for path in sorted(ROOT.rglob(pattern)):
            if not should_skip_python_cache_path(path):
                remove_path(target, path)


def require_files(paths: Sequence[Path]) -> None:
    for path in paths:
        require_file(path)


def require_dirs(paths: Sequence[Path]) -> None:
    for path in paths:
        require_dir(path)


def verify_release_apps_checkout(target: str) -> None:
    """Require the packaged Apps checkout to match the reviewed gitlink exactly."""
    require_dir(APPS_DIR, "initialized Apps submodule checkout")
    commands = (
        (("git", "rev-parse", f"HEAD:{APPS_GITLINK_PATH.as_posix()}"), ROOT),
        (("git", "rev-parse", "--show-toplevel"), APPS_DIR),
        (("git", "rev-parse", "HEAD"), APPS_DIR),
        (("git", "status", "--porcelain=v1", "--untracked-files=all"), APPS_DIR),
    )
    outputs: list[str] = []
    try:
        for command, cwd in commands:
            result = run(target, command, cwd=cwd, capture_output=True)
            outputs.append((result.stdout or "").strip())
    except subprocess.CalledProcessError as error:
        raise SystemExit("unable to verify the Apps submodule checkout") from error

    expected_commit, checkout_root, actual_commit, dirty_state = outputs
    if not re.fullmatch(r"[0-9a-f]{40}", expected_commit):
        raise SystemExit("Apps path is not a valid submodule gitlink in HEAD")
    if Path(checkout_root).resolve() != APPS_DIR.resolve():
        raise SystemExit("Apps submodule checkout is not initialized")
    if actual_commit != expected_commit:
        raise SystemExit(
            "Apps submodule checkout does not match the superproject gitlink: "
            f"expected {expected_commit}, got {actual_commit}"
        )
    if dirty_state:
        raise SystemExit("Apps submodule checkout is dirty; release packaging refused")


def validate_debian_version(target: str, version: str) -> str:
    if not version:
        raise SystemExit("Debian package version must not be empty")
    if version.startswith(("v", "V")):
        raise SystemExit(
            "Debian package version must omit the leading 'v' prefix, "
            "for example use 0.0.26"
        )
    if ":" in version:
        raise SystemExit(
            "Debian package versions with an epoch are not supported because release "
            "artifact filenames must contain the exact requested version"
        )

    try:
        run(
            target,
            ("dpkg", "--validate-version", version),
            capture_output=True,
        )
    except subprocess.CalledProcessError as error:
        detail = (error.stderr or "").strip()
        message = f"invalid Debian package version: {version!r}"
        if detail:
            message = f"{message}: {detail}"
        raise SystemExit(message) from error

    return version


def validate_update_version(target: str, version: str) -> str:
    validated_version = validate_debian_version(target, version)
    if UPDATE_VERSION_PATTERN.fullmatch(validated_version) is None:
        raise SystemExit(
            "release/update version must use exactly three numeric components, "
            "for example 0.0.26"
        )
    if any(int(component) > (1 << 64) - 1 for component in validated_version.split(".")):
        raise SystemExit(
            "release/update version components must fit the update verifier's "
            "64-bit component range"
        )
    return validated_version


def rewrite_staged_changelog(
    changelog: Path,
    package_name: str,
    version: str,
) -> None:
    contents = require_file(changelog).read_bytes()
    first_line, separator, remainder = contents.partition(b"\n")
    carriage_return = b"\r" if first_line.endswith(b"\r") else b""
    line = first_line.removesuffix(carriage_return)
    prefix = f"{package_name} (".encode()
    if not line.startswith(prefix):
        raise SystemExit(
            f"unexpected first changelog line in {format_path(changelog)}: "
            f"expected package {package_name}"
        )
    closing_parenthesis = line.find(b")", len(prefix))
    if closing_parenthesis < 0:
        raise SystemExit(f"invalid first changelog line in {format_path(changelog)}")

    rewritten = (
        prefix
        + version.encode()
        + line[closing_parenthesis:]
        + carriage_return
        + separator
        + remainder
    )
    changelog.write_bytes(rewritten)


@contextmanager
def staged_debian_package(
    target: str,
    package_name: str,
    package_dir: Path,
    version: str,
) -> Iterator[Path]:
    with tempfile.TemporaryDirectory(
        prefix=f".{package_name}-release-",
        dir=DEBHELPER_PACKAGES_DIR,
    ) as directory:
        staged_dir = Path(directory)
        shutil.copytree(
            package_dir,
            staged_dir,
            symlinks=True,
            dirs_exist_ok=True,
        )
        rewrite_staged_changelog(
            staged_dir / "debian" / "changelog",
            package_name,
            version,
        )
        log(target, f"staged package version {version}: {format_path(staged_dir)}")
        yield staged_dir


def verify_debian_package_metadata(
    target: str,
    package_path: Path,
    package_name: str,
    version: str,
) -> None:
    expected_fields = {
        "Package": package_name,
        "Version": version,
        "Architecture": "armhf",
    }
    for field, expected in expected_fields.items():
        result = run(
            target,
            ("dpkg-deb", "-f", str(package_path), field),
            capture_output=True,
        )
        actual = (result.stdout or "").strip()
        if actual != expected:
            raise SystemExit(
                f"unexpected {field} in {format_path(package_path)}: "
                f"expected {expected!r}, got {actual!r}"
            )


def debian_artifact_paths(package_name: str, version: str) -> list[Path]:
    basename = f"{package_name}_{version}_armhf"
    return [
        DEBHELPER_PACKAGES_DIR / f"{basename}.{suffix}"
        for suffix in PACKAGE_ARTIFACT_SUFFIXES
    ]


def collect_debian_artifacts(
    target: str,
    package_name: str,
    version: str | None = None,
) -> None:
    if version is not None:
        artifacts = debian_artifact_paths(package_name, version)
        for artifact in artifacts:
            require_file(artifact, f"expected Debian artifact {artifact.name}")
        verify_debian_package_metadata(
            target,
            artifacts[0],
            package_name,
            version,
        )
        for source in artifacts:
            move_file(target, source, BUILD_DIR / source.name)
        return

    moved = False
    for suffix in PACKAGE_ARTIFACT_SUFFIXES:
        for source in sorted(DEBHELPER_PACKAGES_DIR.glob(f"{package_name}_*.{suffix}")):
            move_file(target, source, BUILD_DIR / source.name)
            moved = True

    if not moved:
        raise SystemExit(f"missing Debian package artifacts for {package_name}")


def build_debian_package(
    target: str,
    package_name: str,
    package_dir: Path,
    *,
    use_docker: bool,
    version: str | None = None,
) -> None:
    def run_package_build(build_dir: Path) -> None:
        if use_docker:
            require_file(DOCKER_PACKAGE_SCRIPT)
            log(target, "mode: docker")
            run(target, (str(DOCKER_PACKAGE_SCRIPT), build_dir.name), cwd=ROOT)
        else:
            log(target, "mode: host")
            run(target, DPKG_BUILDPACKAGE, cwd=build_dir)

    if version is None:
        run_package_build(package_dir)
    else:
        validated_version = validate_debian_version(target, version)
        for artifact in debian_artifact_paths(package_name, validated_version):
            remove_path(target, artifact)
        with staged_debian_package(
            target,
            package_name,
            package_dir,
            validated_version,
        ) as staged_dir:
            run_package_build(staged_dir)

    collect_debian_artifacts(target, package_name, version)


def clean(_: argparse.Namespace) -> int:
    target = "clean"
    log(target, f"target: {target}")
    remove_path(target, BUILD_DIR)
    remove_path(target, ROOT / "debhelper" / ".debhelper")
    remove_python_caches(target)

    remove_globs(
        target,
        DEBHELPER_PACKAGES_DIR,
        (
            "*.deb",
            "*.changes",
            "*.buildinfo",
            "*.build",
            "*.dsc",
            "*.tar.*",
            "*/.debhelper",
            "*/debian/.debhelper",
            "*/debian/files",
            "*/debian/kvmcomm",
            "*/debian/nanokvm-go",
            "*/debian/*.substvars",
            "*/debian/*.debhelper",
            "*/debian/debhelper-build-stamp",
        ),
    )

    log(target, "done")
    return 0


def run_component_builder(
    target: str,
    component_dir: Path,
    builder_args: Sequence[str],
    artifact: Path,
    destination: Path,
) -> int:
    log(target, f"target: {target}")
    require_file(component_dir / "builder.sh")
    ensure_conan_default_profile(target)
    run(target, ("./builder.sh", *builder_args), cwd=component_dir)
    copy_file(target, artifact, destination)
    return 0


def build_libkvm(_: argparse.Namespace) -> int:
    target = "libkvm"
    log(target, f"target: {target}")
    component_dir = ROOT / "libkvm"
    require_file(component_dir / "builder.sh")
    ensure_conan_default_profile(target)
    run(target, ("./builder.sh", "rebuild"), cwd=component_dir)
    copy_glob(target, "libkvm/build/libkvm.so*", BUILD_DIR)
    return 0


def build_kvm_vin(_: argparse.Namespace) -> int:
    return run_component_builder(
        "kvm_vin",
        ROOT / "kvm_vin",
        ("rebuild",),
        ROOT / "kvm_vin" / "build" / "kvm_vin",
        BUILD_DIR / "kvm_vin",
    )


def build_kvm_ui(_: argparse.Namespace) -> int:
    return run_component_builder(
        "kvm_ui",
        ROOT / "kvm_ui",
        ("build",),
        ROOT / "kvm_ui" / "build" / "kvm_ui",
        BUILD_DIR / "kvm_ui",
    )


def build_nano_ocr(_: argparse.Namespace) -> int:
    return run_component_builder(
        "nano_ocr",
        ROOT / "nano_ocr",
        ("rebuild",),
        ROOT / "nano_ocr" / "build" / "nano_ocr_app",
        BUILD_DIR / "nano_ocr_app",
    )


def read_toolchain_value(config: configparser.ConfigParser, key: str) -> str:
    if not config.has_option("toolchain", key):
        raise SystemExit(f"missing toolchain.{key} in {format_path(NANOKVM_GO_INI)}")

    value = config.get("toolchain", key).strip()
    if not value:
        raise SystemExit(f"empty toolchain.{key} in {format_path(NANOKVM_GO_INI)}")

    return value


def arm_toolchain() -> tuple[Path, Path]:
    require_file(NANOKVM_GO_INI)

    config = configparser.ConfigParser()
    config.read(NANOKVM_GO_INI)
    if not config.has_section("toolchain"):
        raise SystemExit(f"missing [toolchain] section in {format_path(NANOKVM_GO_INI)}")

    toolchain_name = read_toolchain_value(config, "name")
    toolchain_cc = read_toolchain_value(config, "cc")
    toolchain_dir = SUPPORT_TOOLCHAINS_DIR / toolchain_name
    arm_cc = toolchain_dir / toolchain_cc

    if not arm_cc.is_file() or not os.access(arm_cc, os.X_OK):
        raise SystemExit(
            "ARM compiler not found or not executable: "
            f"{format_path(arm_cc)}; run support/scripts/toolchain_setup.sh "
            f"or check {format_path(NANOKVM_GO_INI)}"
        )

    return toolchain_dir, arm_cc


def build_webcam(_: argparse.Namespace) -> int:
    target = "webcam"
    log(target, f"target: {target}")
    _, arm_cc = arm_toolchain()
    BUILD_DIR.mkdir(exist_ok=True)
    run(
        target,
        (
            str(arm_cc), "-O2", "-Wall", "-Wextra", "-Werror",
            str(ROOT / "support" / "webcam" / "feeder.c"),
            "-o", str(BUILD_DIR / "nomad-uvc-feeder"),
        ),
    )
    return 0


def build_memfab(_: argparse.Namespace) -> int:
    target = "memfab"
    log(target, f"target: {target}")
    toolchain_dir, arm_cc = arm_toolchain()
    memfab_dir = ROOT / "memfab"
    build_dir = memfab_dir / "build-go"
    run(
        target,
        (
            "cmake",
            "-S",
            str(memfab_dir),
            "-B",
            str(build_dir),
            f"-DMEMFAB_ARM_TOOLCHAIN_DIR={toolchain_dir}",
            f"-DMEMFAB_ARM_CC={arm_cc}",
        ),
    )
    run(
        target,
        ("cmake", "--build", str(build_dir), "--target", "memfab-linux-armv7-with-toolchain"),
    )
    copy_file(target, build_dir / "dist" / "memfab-linux-armv7", BUILD_DIR / "memfab")
    return 0


# Public halves of the official signing keys. Checked in on purpose: they are
# not secrets, and defaulting to them keeps a build without a private key from
# emitting an empty key ring. The offline release key is the root that every
# shipped device already pins; the CI key signs routine releases from GitHub
# Actions. Both stay pinned so either can sign, and the root can always ship a
# key ring that replaces a compromised CI key.
DEFAULT_TRUSTED_PUBLIC_KEYS = (
    ROOT / "support" / "keys" / "nanokvm-go-release.pub.pem",
    ROOT / "support" / "keys" / "nanokvm-go-ci.pub.pem",
)


def resolve_trusted_public_keys(
    signing_key: Path | None,
    trusted_public_keys: Sequence[Path],
) -> Sequence[Path]:
    """Fall back to the checked-in official public keys.

    An explicit --trusted-public-key always wins, so a rotation is never
    silently joined by the current keys. Without one, an unsigned build pins
    every official key, and so does a build signed by one of them: otherwise
    an offline release signed by the root would ship a key ring without the CI
    key, and devices would reject the next CI-signed release. A throwaway
    signing key still pins only itself.
    """
    if trusted_public_keys:
        return trusted_public_keys
    present = tuple(path for path in DEFAULT_TRUSTED_PUBLIC_KEYS if path.is_file())
    if not present:
        return trusted_public_keys
    if len(present) != len(DEFAULT_TRUSTED_PUBLIC_KEYS):
        missing = ", ".join(
            format_path(path) for path in DEFAULT_TRUSTED_PUBLIC_KEYS if path not in present
        )
        raise SystemExit(f"official update public key missing: {missing}")
    if signing_key is None:
        return present
    signing_public_key = public_key_from_private(signing_key)
    if any(public_key_from_public(path) == signing_public_key for path in present):
        return present
    return trusted_public_keys


def require_complete_official_keys(
    target: str, keyring: Path, allow_partial: bool
) -> None:
    """Refuse a key ring that pins some official keys but not all of them.

    --trusted-public-key replaces the defaults, so a rotation recipe that
    lists only the next key silently drops the CI key (or, under a CI signer,
    the root that recovers from a CI compromise). Rings with no official key
    at all are test builds and pass. Dropping an official key on purpose, as
    when replacing a compromised CI key, needs --allow-partial-official-keys.
    """
    official = {
        update_key_id(public_key_from_public(path)): path
        for path in DEFAULT_TRUSTED_PUBLIC_KEYS
        if path.is_file()
    }
    document = json.loads(keyring.read_text(encoding="utf-8"))
    pinned = {entry.get("keyId") for entry in document.get("keys", [])}
    missing = [path for identifier, path in official.items() if identifier not in pinned]
    if not missing or len(missing) == len(official):
        return
    names = ", ".join(format_path(path) for path in missing)
    if allow_partial:
        log(target, f"WARNING: key ring drops official key(s) on request: {names}")
        return
    raise SystemExit(
        f"refusing a key ring that drops official key(s): {names}. "
        "--trusted-public-key replaces the defaults, so list every official "
        "key to keep, or pass --allow-partial-official-keys to drop one on purpose."
    )


def prepare_update_keyring(
    target: str,
    signing_key: Path | None,
    trusted_public_keys: Sequence[Path] = (),
    allow_partial_official_keys: bool = False,
) -> None:
    UPDATE_KEYRING.parent.mkdir(parents=True, exist_ok=True)
    resolved = resolve_trusted_public_keys(signing_key, trusted_public_keys)
    if resolved is not trusted_public_keys:
        defaults = ", ".join(format_path(path) for path in resolved)
        log(target, f"update trust: defaulting to {defaults}")
    key_id = write_keyring(
        signing_key,
        UPDATE_KEYRING,
        force=True,
        public_key_paths=resolved,
    )
    if key_id is None:
        log(target, "update trust: development build with an empty key ring")
    else:
        log(target, f"update trust: pinned {key_id}")
        require_complete_official_keys(
            target, UPDATE_KEYRING, allow_partial_official_keys
        )
    log(target, f"artifact: {format_path(UPDATE_KEYRING)}")


def require_non_empty_update_keyring(target: str, allow_empty: bool) -> None:
    """Refuse to package an empty update key ring.

    kvmcomm.install ships build/update-keys.json to
    /usr/share/nanokvm/update-keys.json, so an empty ring does not merely make
    the package unverifiable: installing it replaces the device key ring with
    the empty one, and the device then rejects officially signed updates too,
    permanently, unless signature verification stays disabled.

    Keep the checked-in official public keys under support/keys/ (build
    without --trusted-public-key) or, deliberately, pass --allow-empty-keyring.
    """
    if not UPDATE_KEYRING.is_file():
        # Absence is a different failure, and one the packaging step already
        # reports: kvmcomm.install references the file, so dpkg refuses to
        # build. This gate only rejects a ring that exists and is empty.
        return
    try:
        document = json.loads(UPDATE_KEYRING.read_text(encoding="utf-8"))
    except (OSError, ValueError) as error:
        raise SystemExit(f"unable to read {format_path(UPDATE_KEYRING)}: {error}")
    keys = document.get("keys") if isinstance(document, dict) else None
    if keys:
        return
    if allow_empty:
        log(target, "WARNING: packaging an EMPTY update key ring on request")
        log(target, "WARNING: installing this package disables future signed updates")
        return
    raise SystemExit(
        "refusing to package an empty update key ring: installing it would "
        "overwrite the device key ring and permanently break signed updates. "
        "Restore the official public keys under support/keys/ and rebuild "
        "without --trusted-public-key, or pass --allow-empty-keyring to override."
    )


def verify_packaged_update_keyring(target: str, package_path: Path) -> None:
    require_file(UPDATE_KEYRING)
    BUILD_DIR.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(
        prefix=".nanokvm-update-package-",
        dir=BUILD_DIR,
    ) as directory:
        root = Path(directory)
        run(target, ("dpkg-deb", "-x", str(package_path), str(root)))
        packaged_verifier = root / "usr" / "bin" / "nanokvm-update-verify"
        packaged_keyring = root / "usr" / "share" / "nanokvm" / "update-keys.json"
        require_file(packaged_verifier, "packaged nanokvm-update-verify")
        require_file(packaged_keyring, "packaged update-keys.json")
        if packaged_keyring.read_bytes() != UPDATE_KEYRING.read_bytes():
            raise SystemExit("packaged update key ring does not match the release key ring")
    log(target, "verified packaged update verifier and matching key ring")


def validate_optional_update_signing_key(signing_key: Path | None) -> Path | None:
    if signing_key is None:
        return None
    return require_file(signing_key, "external Ed25519 signing key")


def build_server(args: argparse.Namespace) -> int:
    target = "server"
    log(target, f"target: {target}")
    prepare_update_keyring(
        target,
        getattr(args, "signing_key", None),
        getattr(args, "trusted_public_key", ()),
        getattr(args, "allow_partial_official_keys", False),
    )
    server_dir = ROOT / "server"
    require_file(server_dir / "build.sh")
    server_env = {"TOOLCHAIN_INI": str(SUPPORT_TOOLCHAINS_DIR / "toolchain.ini")}
    run(target, ("./build.sh",), cwd=server_dir, env=server_env)
    copy_file(target, server_dir / "NanoKVM-Server", BUILD_DIR / "NanoKVM-Server")
    verifier_env = {
        "CGO_ENABLED": "0",
        "GOOS": "linux",
        "GOARCH": "arm",
        "GOARM": "7",
    }
    run(
        target,
        (
            "go",
            "build",
            "-trimpath",
            "-buildvcs=false",
            "-o",
            str(UPDATE_VERIFIER),
            "./cmd/nanokvm-update-verify",
        ),
        cwd=server_dir,
        env=verifier_env,
    )
    log(target, f"artifact: {format_path(UPDATE_VERIFIER)}")
    return 0


def build_web(_: argparse.Namespace) -> int:
    target = "web"
    log(target, f"target: {target}")
    web_dir = ROOT / "web"
    require_file(web_dir / "package.json")
    run(target, ("pnpm", "build"), cwd=web_dir)
    copy_tree(target, web_dir / "dist", BUILD_DIR / "web")
    return 0


BUILD_TARGETS = {
    "server": build_server,
    "web": build_web,
    "libkvm": build_libkvm,
    "kvm_vin": build_kvm_vin,
    "kvm_ui": build_kvm_ui,
    "nano_ocr": build_nano_ocr,
    "webcam": build_webcam,
    "memfab": build_memfab,
}


def build_all(args: argparse.Namespace) -> int:
    target = "build"
    log(target, f"target: {target}")
    clean(args)

    for name in BUILD_SEQUENCE:
        handler = BUILD_TARGETS[name]
        handler(
            argparse.Namespace(
                target=name,
                signing_key=getattr(args, "signing_key", None) if name == "server" else None,
                trusted_public_key=(
                    getattr(args, "trusted_public_key", ()) if name == "server" else ()
                ),
                allow_partial_official_keys=(
                    name == "server"
                    and getattr(args, "allow_partial_official_keys", False)
                ),
            )
        )

    log(target, "done")
    return 0


def package_kvmcomm(args: argparse.Namespace) -> int:
    target = "package_kvmcomm"
    log(target, f"target: {target}")
    version = getattr(args, "version", None)
    if version is not None and not getattr(args, "apps_checkout_verified", False):
        verify_release_apps_checkout(target)
    require_dir(KVMCOMM_PACKAGE_DIR)
    require_files(KVMCOMM_INPUT_FILES)
    require_non_empty_update_keyring(
        target, getattr(args, "allow_empty_keyring", False)
    )
    build_debian_package(
        target,
        "kvmcomm",
        KVMCOMM_PACKAGE_DIR,
        use_docker=getattr(args, "docker", False),
        version=version,
    )
    return 0


def package_nanokvm_go(args: argparse.Namespace) -> int:
    target = "package_nanokvm_go"
    log(target, f"target: {target}")
    require_dir(NANOKVM_GO_PACKAGE_DIR)
    require_files(NANOKVM_GO_INPUT_FILES)
    require_dirs(NANOKVM_GO_INPUT_DIRS)
    build_debian_package(
        target,
        "nanokvm-go",
        NANOKVM_GO_PACKAGE_DIR,
        use_docker=getattr(args, "docker", False),
        version=getattr(args, "version", None),
    )
    return 0


def package_update(args: argparse.Namespace) -> int:
    target = "package_update"
    log(target, f"target: {target}")
    version = validate_update_version(target, args.version)
    require_file(BUILD_DIR / f"nanokvm-go_{version}_armhf.deb")
    require_file(BUILD_DIR / f"kvmcomm_{version}_armhf.deb")
    signing_key = validate_optional_update_signing_key(
        getattr(args, "signing_key", None)
    )
    update_dir = BUILD_DIR / "update"
    if signing_key is not None:
        verify_packaged_update_keyring(
            target,
            BUILD_DIR / f"kvmcomm_{version}_armhf.deb",
        )
    else:
        log(target, "WARNING: creating an unsigned application update bundle")
    remove_path(target, update_dir)
    command = [
        sys.executable,
        str(SUPPORT_SCRIPTS_DIR / "package_update.py"),
        "--version",
        version,
        "--input-dir",
        str(BUILD_DIR),
        "--output-dir",
        str(update_dir),
    ]
    if signing_key is not None:
        command.extend(
            ("--signing-key", str(signing_key), "--keyring", str(UPDATE_KEYRING))
        )
    run(
        target,
        command,
    )
    log(target, f"artifact: {format_path(update_dir / f'nanokvm-go_{version}.tar.gz')}")
    log(target, f"artifact: {format_path(update_dir / 'latest.json')}")
    return 0


def release(args: argparse.Namespace) -> int:
    target = "release"
    log(target, f"target: {target}")
    version = validate_update_version(target, args.version)
    verify_release_apps_checkout(target)
    signing_key = validate_optional_update_signing_key(args.signing_key)
    use_docker = getattr(args, "docker", False)
    build_all(
        argparse.Namespace(
            target="build",
            signing_key=signing_key,
            trusted_public_key=args.trusted_public_key,
            allow_partial_official_keys=getattr(
                args, "allow_partial_official_keys", False
            ),
        )
    )
    package_kvmcomm(
        argparse.Namespace(
            target="package_kvmcomm",
            docker=use_docker,
            version=version,
            apps_checkout_verified=True,
            allow_empty_keyring=getattr(args, "allow_empty_keyring", False),
        )
    )
    package_nanokvm_go(
        argparse.Namespace(
            target="package_nanokvm_go",
            docker=use_docker,
            version=version,
        )
    )
    package_update(
        argparse.Namespace(
            target="package_update",
            version=version,
            signing_key=signing_key,
        )
    )
    log(target, "done")
    return 0


def add_target_parser(
    subparsers: argparse._SubParsersAction[argparse.ArgumentParser],
    name: str,
    handler,
    *,
    help_text: str,
) -> argparse.ArgumentParser:
    parser = subparsers.add_parser(name, help=help_text)
    parser.set_defaults(func=handler, target=name)
    return parser


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="NanoKVM-Go build and packaging driver."
    )
    subparsers = parser.add_subparsers(dest="target", required=True)

    for name, handler in BUILD_TARGETS.items():
        target_parser = add_target_parser(
            subparsers,
            name,
            handler,
            help_text=f"Build {name} and copy artifacts into build/",
        )
        if name == "server":
            target_parser.add_argument(
                "--signing-key",
                type=Path,
                help="Derive the packaged update key ring from this Ed25519 key.",
            )
            target_parser.add_argument(
                "--trusted-public-key",
                type=Path,
                action="append",
                default=[],
                help=(
                    "Ed25519 public key to pin (repeatable). Replaces the "
                    "default official keys, so list every key to keep."
                ),
            )
            target_parser.add_argument(
                "--allow-partial-official-keys",
                action="store_true",
                help=(
                    "Allow --trusted-public-key to drop an official key from "
                    "support/keys/, e.g. when replacing a compromised CI key."
                ),
            )

    build_parser = add_target_parser(
        subparsers,
        "build",
        build_all,
        help_text="Clean root output, then build all component artifacts",
    )
    build_parser.add_argument(
        "--signing-key",
        type=Path,
        help="Derive the packaged update key ring from this Ed25519 key.",
    )
    build_parser.add_argument(
        "--trusted-public-key",
        type=Path,
        action="append",
        default=[],
        help=(
            "Ed25519 public key to pin (repeatable). Replaces the default "
            "official keys, so list every key to keep."
        ),
    )
    build_parser.add_argument(
        "--allow-partial-official-keys",
        action="store_true",
        help=(
            "Allow --trusted-public-key to drop an official key from "
            "support/keys/, e.g. when replacing a compromised CI key."
        ),
    )

    package_kvmcomm_parser = add_target_parser(
        subparsers,
        "package_kvmcomm",
        package_kvmcomm,
        help_text="Build the kvmcomm Debian package",
    )
    package_kvmcomm_parser.add_argument(
        "--docker",
        action="store_true",
        help="Build the Debian package inside the Docker packaging environment",
    )
    package_kvmcomm_parser.add_argument(
        "--version",
        help="Build an exact Debian version without modifying the tracked changelog",
    )
    package_kvmcomm_parser.add_argument(
        "--allow-empty-keyring",
        action="store_true",
        help=(
            "Package an empty update key ring. Installing such a package "
            "permanently breaks signed updates on the device."
        ),
    )

    package_nanokvm_go_parser = add_target_parser(
        subparsers,
        "package_nanokvm_go",
        package_nanokvm_go,
        help_text="Build the nanokvm-go Debian package",
    )
    package_nanokvm_go_parser.add_argument(
        "--docker",
        action="store_true",
        help="Build the Debian package inside the Docker packaging environment",
    )
    package_nanokvm_go_parser.add_argument(
        "--version",
        help="Build an exact Debian version without modifying the tracked changelog",
    )

    package_update_parser = subparsers.add_parser(
        "package_update",
        help="Generate the updater bundle",
    )
    package_update_parser.add_argument("--version", required=True, help="Package version")
    package_update_parser.add_argument(
        "--signing-key",
        type=Path,
        help=(
            "External Ed25519 key used to sign the exact bundled deb bytes; "
            "omit it to create an unsigned update bundle"
        ),
    )
    package_update_parser.set_defaults(func=package_update, target="package_update")

    release_parser = subparsers.add_parser(
        "release",
        help="Run build, package, and updater bundle steps",
    )
    release_parser.add_argument("--version", required=True, help="Package version")
    release_parser.add_argument(
        "--signing-key",
        type=Path,
        help=(
            "External Ed25519 key used to derive the packaged public key ring "
            "and sign the exact bundled deb bytes; omit it for an unsigned bundle"
        ),
    )
    release_parser.add_argument(
        "--trusted-public-key",
        type=Path,
        action="append",
        default=[],
        help=(
            "Ed25519 public key to pin (repeatable). Replaces the default "
            "official keys, so list every key to keep."
        ),
    )
    release_parser.add_argument(
        "--allow-partial-official-keys",
        action="store_true",
        help=(
            "Allow --trusted-public-key to drop an official key from "
            "support/keys/, e.g. when replacing a compromised CI key."
        ),
    )
    release_parser.add_argument(
        "--docker",
        action="store_true",
        help="Build Debian packages inside the Docker packaging environment",
    )
    release_parser.set_defaults(func=release, target="release")

    add_target_parser(
        subparsers,
        "clean",
        clean,
        help_text="Remove root build output and Debian package build artifacts",
    )

    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    try:
        return args.func(args)
    except subprocess.CalledProcessError as error:
        target = getattr(args, "target", "build")
        log(target, f"command failed with exit code {error.returncode}: {format_command(error.cmd)}")
        return error.returncode


if __name__ == "__main__":
    raise SystemExit(main())
