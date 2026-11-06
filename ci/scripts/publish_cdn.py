#!/usr/bin/env python3
"""Publish a signed application update to the Qiniu-backed CDN channel.

The publish set is exactly nanokvm-go_<version>.tar.gz and latest.json. The
tarball goes first and latest.json last, so a failed upload leaves devices on
the previous release. The channel must already have a latest.json, so a
mistyped prefix cannot quietly start a new channel. The first release on a
channel, such as the root-signed v2 bridge release, is published by hand with
--initialize-channel, which in turn refuses a channel that already exists.

Credentials come from QINIU_ACCESS_KEY and QINIU_SECRET_KEY. Uploads use the
S3-compatible endpoint through the AWS CLI; the CDN refresh uses the Qiniu
fusion API.
"""
from __future__ import annotations

import argparse
import base64
import gzip
import hashlib
import hmac
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

VERSION_PATTERN = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
MANIFEST_FIELDS = {"version", "name", "sha512", "size"}
FUSION_API = "https://fusion.qiniuapi.com"
TARBALL_TYPE = "application/gzip"
MANIFEST_TYPE = "application/json; charset=utf-8"


class PublishError(RuntimeError):
    pass


def log(message: str) -> None:
    print(message, flush=True)


def parse_version(version: str) -> tuple[int, int, int]:
    match = VERSION_PATTERN.fullmatch(version)
    if match is None:
        raise PublishError(f"version {version!r} is not X.Y.Z")
    return tuple(int(part) for part in match.groups())  # type: ignore[return-value]


def sha512_b64(data: bytes) -> str:
    return base64.b64encode(hashlib.sha512(data).digest()).decode()


def check_local(directory: Path, version: str) -> tuple[bytes, bytes, dict]:
    tar_name = f"nanokvm-go_{version}.tar.gz"
    manifest_bytes = (directory / "latest.json").read_bytes()
    tarball = (directory / tar_name).read_bytes()
    manifest = json.loads(manifest_bytes)
    if not isinstance(manifest, dict) or set(manifest) != MANIFEST_FIELDS:
        raise PublishError(f"latest.json must have exactly {sorted(MANIFEST_FIELDS)}")
    if manifest["version"] != version or manifest["name"] != tar_name:
        raise PublishError(f"latest.json names {manifest['name']} {manifest['version']}, expected {tar_name}")
    if manifest["size"] != len(tarball) or manifest["sha512"] != sha512_b64(tarball):
        raise PublishError("latest.json size or SHA-512 does not match the tarball")
    with tarfile.open(directory / tar_name, "r:gz") as archive:
        roots = {member.name.split("/", 1)[0] for member in archive.getmembers()}
    if roots != {f"nanokvm-go_{version}"}:
        raise PublishError(f"tarball top level is {sorted(roots)}, expected nanokvm-go_{version}")
    return tarball, manifest_bytes, manifest


class Bucket:
    def __init__(self, args: argparse.Namespace) -> None:
        self.bucket = args.bucket
        self.common = ["--endpoint-url", args.endpoint, "--region", args.region]
        self.env = dict(
            os.environ,
            AWS_ACCESS_KEY_ID=os.environ["QINIU_ACCESS_KEY"],
            AWS_SECRET_ACCESS_KEY=os.environ["QINIU_SECRET_KEY"],
            # Without this the CLI sends aws-chunked bodies, which Qiniu stores
            # with a bogus Content-Encoding and the CDN then serves garbled.
            AWS_REQUEST_CHECKSUM_CALCULATION="when_required",
            AWS_RESPONSE_CHECKSUM_VALIDATION="when_required",
            AWS_RETRY_MODE="standard",
            AWS_MAX_ATTEMPTS="6",
        )
        # The link from a GitHub-hosted runner to the bucket can drop to tens
        # of KB/s, and a single PUT of the tarball then outlives its signature
        # (RequestTimeTooSkewed after 15 minutes). Upload in small parts, each
        # signed when it is sent.
        self._config = tempfile.NamedTemporaryFile("w", prefix="aws-config-", suffix=".ini", delete=False)
        self._config.write(
            "[default]\ns3 =\n"
            "  multipart_threshold = 5MB\n"
            "  multipart_chunksize = 5MB\n"
            "  max_concurrent_requests = 4\n"
        )
        self._config.close()
        self.env["AWS_CONFIG_FILE"] = self._config.name
        self._cache: dict[str, bytes] = {}

    def _api(self, *command: str, check: bool = True) -> subprocess.CompletedProcess:
        return subprocess.run(
            ["aws", "s3api", *command, "--bucket", self.bucket, *self.common, "--output", "json"],
            env=self.env, capture_output=True, text=True, check=check,
        )

    def head(self, key: str) -> dict | None:
        result = self._api("head-object", "--key", key, check=False)
        if result.returncode == 0:
            return json.loads(result.stdout)
        if "(404)" in result.stderr:
            return None
        raise PublishError(f"head-object {key} failed: {result.stderr.strip()}")

    def get(self, key: str) -> bytes:
        if key not in self._cache:
            with tempfile.TemporaryDirectory() as directory:
                target = Path(directory) / "object"
                self._api("get-object", "--key", key, str(target))
                self._cache[key] = target.read_bytes()
        return self._cache[key]

    def metadata_problem(self, key: str, size: int, content_type: str) -> str | None:
        head = self.head(key)
        if head is None:
            return "missing"
        if head.get("ContentLength") != size:
            return f"size {head.get('ContentLength')}"
        if head.get("ContentEncoding"):
            return f"Content-Encoding {head['ContentEncoding']!r}"
        if not head.get("ContentType", "").startswith(content_type.split(";", 1)[0]):
            return f"Content-Type {head.get('ContentType')!r}"
        return None

    def put(self, key: str, data: bytes, content_type: str) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "object"
            source.write_bytes(data)
            subprocess.run(
                ["aws", "s3", "cp", str(source), f"s3://{self.bucket}/{key}", *self.common,
                 "--content-type", content_type, "--cache-control", "no-transform",
                 "--no-progress", "--only-show-errors"],
                env=self.env, capture_output=True, text=True, check=True,
            )
        self._cache.pop(key, None)
        problem = self.metadata_problem(key, len(data), content_type)
        if problem:
            raise PublishError(f"{key} is stored wrongly after upload: {problem}")
        log(f"uploaded {key}")

    def ensure(self, key: str, data: bytes, content_type: str) -> None:
        """Upload unless the object already holds these bytes with sane metadata."""
        if self.head(key) is not None and sha512_b64(self.get(key)) == sha512_b64(data):
            problem = self.metadata_problem(key, len(data), content_type)
            if problem is None:
                log(f"{key} already in place")
                return
            log(f"{key} has the right bytes but {problem}; uploading again")
        self.put(key, data, content_type)


def refresh(*urls: str) -> None:
    path = "/v2/tune/refresh"
    body = json.dumps({"urls": list(urls)}).encode()
    signature = hmac.new(os.environ["QINIU_SECRET_KEY"].encode(), (path + "\n").encode(), hashlib.sha1)
    token = base64.urlsafe_b64encode(signature.digest()).decode()
    request = urllib.request.Request(
        FUSION_API + path, data=body, method="POST",
        headers={
            "Content-Type": "application/json",
            "Authorization": f"QBox {os.environ['QINIU_ACCESS_KEY']}:{token}",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            result = json.loads(response.read())
    except urllib.error.HTTPError as error:
        raise PublishError(f"CDN refresh HTTP {error.code}: {error.read().decode(errors='replace')}")
    except (OSError, ValueError) as error:
        raise PublishError(f"CDN refresh failed: {error}")
    if not isinstance(result, dict) or result.get("code") != 200:
        raise PublishError(f"CDN refresh rejected: {result}")
    log(f"refresh submitted for {', '.join(urls)} ({result.get('requestId', '')})")


def fetch(url: str, encoding: str) -> bytes:
    request = urllib.request.Request(url, headers={"Accept-Encoding": encoding, "User-Agent": "nanokvm-publish"})
    with urllib.request.urlopen(request, timeout=60) as response:
        data = response.read()
        if response.headers.get("Content-Encoding", "").lower() == "gzip":
            data = gzip.decompress(data)
    return data


def wait_for(description: str, check, timeout: int) -> None:
    """Retry check() until it returns None, for both identity and gzip requests."""
    deadline = time.monotonic() + timeout
    pending = ["identity", "gzip"]
    while True:
        for encoding in list(pending):
            try:
                problem = check(encoding)
            except (OSError, ValueError) as error:
                problem = str(error)
            if problem is None:
                log(f"{encoding}: {description} OK")
                pending.remove(encoding)
            else:
                log(f"{encoding}: {description}: {problem}")
        if not pending:
            return
        if time.monotonic() > deadline:
            raise PublishError(f"CDN never served {description} correctly for {pending}")
        time.sleep(20)


def tarball_check(url: str, manifest: dict):
    def check(encoding: str) -> str | None:
        data = fetch(url, encoding)
        if len(data) != manifest["size"] or sha512_b64(data) != manifest["sha512"]:
            return f"got {len(data)} bytes that do not match latest.json"
        return None
    return check


def manifest_check(url: str, manifest: dict):
    def check(encoding: str) -> str | None:
        served = json.loads(fetch(url, encoding))
        if served != manifest:
            version = served.get("version") if isinstance(served, dict) else None
            return f"still serves {version!r}"
        return None
    return check


def publish(args: argparse.Namespace) -> None:
    version = args.version
    candidate = parse_version(version)
    tarball, manifest_bytes, manifest = check_local(args.dir, version)
    prefix = args.prefix.strip("/")
    tar_key = f"{prefix}/{manifest['name']}"
    manifest_key = f"{prefix}/latest.json"
    base_url = f"{args.cdn_base.rstrip('/')}/{prefix}"
    tar_url = f"{base_url}/{manifest['name']}"
    manifest_url = f"{base_url}/latest.json"
    bucket = Bucket(args)

    if bucket.head(manifest_key) is None:
        if not args.initialize_channel:
            raise PublishError(
                f"{manifest_key} does not exist; the first release on a channel is published "
                "by hand with --initialize-channel"
            )
        log(f"channel {prefix} is empty; initializing it with {version}")
        current = None
    else:
        if args.initialize_channel:
            raise PublishError(f"--initialize-channel given, but {manifest_key} already exists")
        current_bytes = bucket.get(manifest_key)
        current = json.loads(current_bytes)
        if not isinstance(current, dict) or not isinstance(current.get("version"), str):
            raise PublishError(f"{manifest_key} is not a valid manifest")
        log(f"channel {prefix} currently serves: {current_bytes.decode().strip()}")
        current_version = parse_version(current["version"])
        if current_version > candidate:
            raise PublishError(f"refusing to downgrade {prefix} from {current['version']} to {version}")
        if current_version == candidate and current != manifest:
            raise PublishError(f"{prefix} already serves a different {version}")
    if bucket.head(tar_key) is not None and sha512_b64(bucket.get(tar_key)) != manifest["sha512"]:
        raise PublishError(f"{tar_key} already exists with different contents")

    # Devices follow latest.json, so the tarball must already be served
    # correctly through the CDN before latest.json points at it.
    bucket.ensure(tar_key, tarball, TARBALL_TYPE)
    refresh(tar_url)
    wait_for(manifest["name"], tarball_check(tar_url, manifest), args.timeout)

    bucket.ensure(manifest_key, manifest_bytes, MANIFEST_TYPE)
    refresh(manifest_url)
    wait_for("latest.json", manifest_check(manifest_url, manifest), args.timeout)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    parser.add_argument("--version", required=True)
    parser.add_argument("--dir", type=Path, required=True, help="directory holding the tarball and latest.json")
    parser.add_argument("--bucket", required=True)
    parser.add_argument("--prefix", required=True, help="channel key prefix, e.g. nanokvm-go/v2")
    parser.add_argument("--endpoint", required=True)
    parser.add_argument("--region", required=True)
    parser.add_argument("--cdn-base", required=True, help="public base URL, e.g. https://cdn.sipeed.com")
    parser.add_argument("--timeout", type=int, default=900, help="seconds to wait for the CDN")
    parser.add_argument(
        "--initialize-channel", action="store_true",
        help="publish the first release to a channel that has no latest.json yet (manual use only)",
    )
    args = parser.parse_args()
    try:
        publish(args)
    except (PublishError, subprocess.CalledProcessError, KeyError, TypeError, ValueError) as error:
        detail = getattr(error, "stderr", "") or ""
        print(f"error: {error} {detail}".strip(), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
