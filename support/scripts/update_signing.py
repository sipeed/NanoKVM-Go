#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import stat
import subprocess
import tempfile
from pathlib import Path
from typing import Any, Dict, Iterable, TypedDict


ED25519_SPKI_PREFIX = bytes.fromhex("302a300506032b6570032100")
ED25519_SIGNATURE_SIZE = 64
SIGNATURE_SCHEMA = 1
SIGNATURE_ALGORITHM = "ed25519"
FILE_MODE = 0o644


class ArtifactSignature(TypedDict):
    schema: int
    algorithm: str
    keyId: str
    value: str


def run_openssl(arguments: list[str]) -> subprocess.CompletedProcess[bytes]:
    try:
        return subprocess.run(
            ["openssl", *arguments],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
    except FileNotFoundError as exc:
        raise SystemExit("openssl is required for release signing") from exc
    except subprocess.CalledProcessError as exc:
        detail = exc.stderr.decode("utf-8", errors="replace").strip()
        raise SystemExit(f"openssl failed: {detail or exc.returncode}") from exc


def public_key_from_private(signing_key: Path) -> bytes:
    if not signing_key.is_file():
        raise SystemExit(f"signing key does not exist: {signing_key}")
    result = run_openssl(
        ["pkey", "-in", str(signing_key), "-pubout", "-outform", "DER"]
    )
    if not result.stdout.startswith(ED25519_SPKI_PREFIX):
        raise SystemExit("signing key is not an Ed25519 private key")
    public_key = result.stdout[len(ED25519_SPKI_PREFIX) :]
    if len(public_key) != 32:
        raise SystemExit("invalid Ed25519 public key length")
    return public_key


def public_key_from_public(public_key_path: Path) -> bytes:
    if not public_key_path.is_file():
        raise SystemExit(f"public key does not exist: {public_key_path}")
    result = run_openssl(
        ["pkey", "-pubin", "-in", str(public_key_path), "-outform", "DER"]
    )
    if not result.stdout.startswith(ED25519_SPKI_PREFIX):
        raise SystemExit(f"public key is not Ed25519: {public_key_path}")
    public_key = result.stdout[len(ED25519_SPKI_PREFIX) :]
    if len(public_key) != 32:
        raise SystemExit("invalid Ed25519 public key length")
    return public_key


def key_id(public_key: bytes) -> str:
    return f"sha256:{hashlib.sha256(public_key).hexdigest()}"


def keyring_document(public_keys: Iterable[bytes]) -> Dict[str, Any]:
    unique_keys: Dict[str, bytes] = {}
    for public_key in public_keys:
        unique_keys[key_id(public_key)] = public_key
    return {
        "schema": 1,
        "keys": [
            {
                "keyId": public_key_id,
                "algorithm": "ed25519",
                "publicKey": base64.b64encode(public_key).decode("ascii"),
            }
            for public_key_id, public_key in unique_keys.items()
        ],
    }


def empty_keyring_document() -> Dict[str, Any]:
    return {"schema": 1, "keys": []}


def atomic_write(path: Path, data: bytes, force: bool) -> None:
    if (path.exists() or path.is_symlink()) and not force:
        raise SystemExit(f"{path} already exists; pass --force to overwrite")
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary_path = Path(temporary_name)
    try:
        with os.fdopen(descriptor, "wb") as output:
            output.write(data)
            output.flush()
            os.fsync(output.fileno())
        temporary_path.chmod(FILE_MODE)
        os.replace(temporary_path, path)
    finally:
        if temporary_path.exists():
            temporary_path.unlink()


def write_keyring(
    signing_key: Path | None,
    output: Path,
    force: bool = False,
    public_key_paths: Iterable[Path] = (),
) -> str | None:
    public_keys = [public_key_from_public(path) for path in public_key_paths]
    if signing_key is None and not public_keys:
        document = empty_keyring_document()
        result_key_id = None
    else:
        if signing_key is not None:
            signing_public_key = public_key_from_private(signing_key)
            public_keys.insert(0, signing_public_key)
            result_key_id = key_id(signing_public_key)
        else:
            result_key_id = key_id(public_keys[0])
        document = keyring_document(public_keys)
    data = (json.dumps(document, indent=2, ensure_ascii=False) + "\n").encode("utf-8")
    atomic_write(output, data, force)
    return result_key_id


def ensure_keyring_matches(signing_key: Path, keyring: Path) -> str:
    public_key = public_key_from_private(signing_key)
    expected_id = key_id(public_key)
    try:
        document = json.loads(keyring.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise SystemExit(f"failed to read packaged update key ring: {keyring}") from exc
    if document.get("schema") != 1 or not isinstance(document.get("keys"), list):
        raise SystemExit("packaged update key ring has an invalid schema")
    expected_public_key = base64.b64encode(public_key).decode("ascii")
    for entry in document["keys"]:
        if (
            isinstance(entry, dict)
            and entry.get("keyId") == expected_id
            and entry.get("algorithm") == "ed25519"
            and entry.get("publicKey") == expected_public_key
        ):
            return expected_id
    raise SystemExit("signing key does not match the packaged update key ring")


def sign_exact_file(artifact: Path, signing_key: Path) -> bytes:
    try:
        artifact_stat = artifact.lstat()
    except OSError as exc:
        raise SystemExit(f"artifact does not exist: {artifact}") from exc
    if not stat.S_ISREG(artifact_stat.st_mode):
        raise SystemExit(f"artifact is not a regular file: {artifact}")

    result = run_openssl(
        [
            "pkeyutl",
            "-sign",
            "-rawin",
            "-inkey",
            str(signing_key),
            "-in",
            str(artifact),
        ]
    )
    signature = result.stdout
    if len(signature) != ED25519_SIGNATURE_SIZE:
        raise SystemExit("openssl produced an invalid Ed25519 signature length")
    return signature


def artifact_signature_document(
    signing_key_id: str, signature: bytes
) -> ArtifactSignature:
    if len(signature) != ED25519_SIGNATURE_SIZE:
        raise SystemExit("invalid Ed25519 signature length")
    return {
        "schema": SIGNATURE_SCHEMA,
        "algorithm": SIGNATURE_ALGORITHM,
        "keyId": signing_key_id,
        "value": base64.b64encode(signature).decode("ascii"),
    }


def sign_artifact(
    artifact: Path,
    signing_key: Path,
    keyring: Path,
) -> ArtifactSignature:
    signing_key_id = ensure_keyring_matches(signing_key, keyring)
    signature = sign_exact_file(artifact, signing_key)
    return artifact_signature_document(signing_key_id, signature)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Create NanoKVM Ed25519 update trust artifacts.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    keyring_parser = subparsers.add_parser("keyring", help="derive a public key ring from a private signing key")
    keyring_parser.add_argument("--signing-key", type=Path)
    keyring_parser.add_argument("--public-key", type=Path, action="append", default=[])
    keyring_parser.add_argument("--output", type=Path, required=True)
    keyring_parser.add_argument("--force", action="store_true")

    artifact_parser = subparsers.add_parser(
        "artifact-signature",
        help="print an embedded signature object for one exact artifact",
    )
    artifact_parser.add_argument("--signing-key", type=Path, required=True)
    artifact_parser.add_argument("--keyring", type=Path, required=True)
    artifact_parser.add_argument("artifact", type=Path)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    if args.command == "keyring":
        if args.signing_key is None and not args.public_key:
            raise SystemExit("keyring requires --signing-key and/or --public-key")
        result_key_id = write_keyring(
            args.signing_key, args.output, args.force, args.public_key
        )
        print(f"created {args.output} ({result_key_id})")
        return
    if args.command == "artifact-signature":
        document = sign_artifact(args.artifact, args.signing_key, args.keyring)
        print(json.dumps(document, indent=2, ensure_ascii=False))
        return


if __name__ == "__main__":
    main()
