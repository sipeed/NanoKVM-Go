# NanoKVM-go-Nomad Build and Packaging Guide

This guide covers builds from the NanoKVM-go-Nomad fork.
Use the root-level Python build driver for normal builds and release packaging.

## Official Entry Point

Run build and packaging commands from the repository root:

```bash
pnpm --dir web install --frozen-lockfile
python3 build.py build
python3 build.py release --version <version>
# Add --signing-key /offline/path/update-signing.pem for a signed release.
```

The repository root does not use `build.sh` as the formal entry point. Component
local `builder.sh` scripts and `server/build.sh` may remain as implementation
details called by `build.py`.

`support/` contains auxiliary setup resources only. It is not the release
packaging entry point.

## Prerequisites

| Component | Requirements |
| --- | --- |
| `server` | Go 1.26.5 or later, Python 3, `patchelf`, configured ARMv7 toolchain |
| `web` | Node.js, pnpm |
| `libkvm`, `kvm_vin`, `kvm_ui`, `nano_ocr` | Python 3, CMake, Conan, configured ARMv7 toolchain |
| `webcam` | Python 3 and the configured ARMv7 C toolchain |
| `memfab` | Go 1.26.5 or later, Python 3, CMake, configured ARMv7 toolchain from `support/scripts/nanokvm-go.ini` |
| Debian packaging | `dpkg-dev`, `debhelper` |
| Signed releases | OpenSSL with Ed25519 `pkey` and `pkeyutl -rawin` support |
| Docker packaging | Docker |

## Toolchain Setup

The formal toolchain setup flow uses the `support/scripts/` and
`support/toolchains/` layout:

```bash
cd support/scripts
./toolchain_setup.sh
cd ../..
```

`support/toolchains/` is treated as local generated state. The repository root
must not contain formal `scripts/` or `toolchains/` directories.

`toolchain_setup.sh` is the normal setup entry. It installs the ARMv7 toolchain,
writes `support/toolchains/toolchain.ini`, and refreshes component Conan
profiles.

`toolchain_setup_components.sh` is a component-profile helper. Use it directly
only when switching or regenerating Conan profiles without reinstalling the
toolchain:

```bash
cd support/scripts
./toolchain_setup_components.sh --config nanokvm-go.ini --switch
```

## Build Commands

Before the first frontend or full build, install the frontend dependencies from
the repository root. Repeat this step after changes to the dependency manifest
or lockfile:

```bash
pnpm --dir web install --frozen-lockfile
```

The build driver expects these dependencies to be installed already.

```bash
python3 build.py server
python3 build.py web
python3 build.py libkvm
python3 build.py kvm_vin
python3 build.py kvm_ui
python3 build.py nano_ocr
python3 build.py webcam
python3 build.py memfab
python3 build.py build
```

`build` runs `clean`, then builds all component artifacts in dependency order.

`nano_ocr` downloads OpenCV 4.6.0 from the official OpenCV GitHub repository,
verifies the archive's SHA-256, and builds static `core`, `imgproc` and
`imgcodecs` libraries with the same ARMv7 toolchain as OCR. The first build
requires network access. Source and build files are cached under
`nano_ocr/.cache/`; the installation is under `nano_ocr/prefix/opencv/`.
These generated files are ignored by Git. To build without network access,
place the official archive at `nano_ocr/.cache/opencv/4.6.0.tar.gz`; it must
match SHA-256
`1ec1cba65f9f20fe5a41fda1586e01c70ea0c9a6d7b67c9e13edf0cfe2239277`.
For direct CMake builds, build OpenCV with `cd nano_ocr && ./builder.sh build`
first, or set `OCR_OPENCV_DIR` to an existing OpenCV 4.6.0 CMake configuration
with those three modules.

FFmpeg is loaded from the target system; its shared
libraries must not be added to the source repository.

Expected local artifacts:

```text
build/NanoKVM-Server
build/web/
build/libkvm.so*
build/kvm_vin
build/kvm_ui
build/nano_ocr_app
build/nomad-uvc-feeder
build/memfab
build/nanokvm-update-verify
build/update-keys.json
```

## Packaging Commands

```bash
python3 build.py package_kvmcomm
python3 build.py package_nanokvm_go
python3 build.py package_kvmcomm --version <version>
python3 build.py package_nanokvm_go --version <version>
python3 build.py package_kvmcomm --docker
python3 build.py package_nanokvm_go --docker
python3 build.py package_update --version <version>
python3 build.py package_update --version <version> --signing-key /offline/path/update-signing.pem
python3 build.py release --version <version>
python3 build.py release --version <version> --signing-key /offline/path/update-signing.pem
python3 build.py release --version <version> --docker
python3 build.py release --version <version> --signing-key /offline/path/update-signing.pem --docker
```

`package_update --version <version>` requires matching
`build/kvmcomm_<version>_armhf.deb` and
`build/nanokvm-go_<version>_armhf.deb`. For both `package_update` and `release`,
supplying `--signing-key` creates a signed application bundle; omitting it
creates an unsigned bundle and prints a warning.

Each numbered NanoKVM Go+ Nomad update includes the application and matching webcam kernel. On Go+ firmware v1.0.10, upload the tar.gz through the offline application updater and reboot if the kernel changed.

For **NanoKVM-Go Nomad 0.2.7, build 1**, use the Git tag `v0.2.7-nomad.1`.

To make the Normal update on Linux, start with the two original application Debian packages in `build/`. The packaging script checks the approved kernel digest and builds the numbered dashboard archive, its package metadata and the two release Debian packages.

```bash
bash support/scripts/package_nomad_normal.sh 0.2.7 1 build \
  build/kernel/NanoKVM-go-Nomad-4.19.125-uvc-name/kernel.img build/v0.2.7-nomad.1
```

The installer checks the Go+ model, firmware and SD boot layout. When the kernel changes it keeps the previous image in `/boot/kernel.img.nomad-backup-<sha256>` and replaces the boot file atomically. Reinstalling the same kernel skips the boot write. Upgrades retain the existing Webcam setting.

Release and versioned `kvmcomm` packaging require an initialized, clean Apps
submodule checkout whose commit exactly matches the superproject gitlink. This
preflight makes the Debian bootstrap payload attributable to the reviewed
superproject revision. The package includes the shared App runtime and its
documentation, but excludes the host-only `apps/_utils` development helpers.

This packaging boundary does not pin the device's official online Apps catalog.
The built-in official Apps server intentionally follows its mutable repository
HEAD so Apps can update independently of NanoKVM releases. That publisher and
repository boundary, including the root-level impact of installed Apps, is an
accepted product trust boundary and must be reviewed operationally rather than
silently replaced with a packaged-commit pin.

Direct `package_kvmcomm` and `package_nanokvm_go` commands remain driven by
the tracked Debian changelog when `--version` is omitted. When `--version` is
present, `build.py` validates Debian version syntax and builds from a temporary
copy under `debhelper/packages/` whose first changelog entry uses that version.
The tracked changelog is never rewritten. Debian epochs (`1:<version>`) are not
accepted because release artifact filenames must contain the exact requested
version and Debian artifact filenames omit the epoch.

For local development, unversioned `package_kvmcomm` retains the existing
working-checkout behavior and does not enforce the release gitlink/cleanliness
preflight. Its manifest inputs must still exist, and `apps/_utils` remains
excluded from the resulting package.

Direct versioned Debian package commands accept other `dpkg`-valid syntax such
as `0.0.26~rc1-1`. `release` and `package_update` require exactly three numeric
components within the update verifier's 64-bit component range, such as
`0.0.26`. This narrower contract keeps ordering identical between the update
verifier and `dpkg`: Debian revisions and SemVer-style prerelease/build suffixes
must not be emitted in an application update bundle.

`release --version <version> [--signing-key <path>]` runs the full flow:

```text
build
package_kvmcomm
package_nanokvm_go
package_update
```

The release version is validated before component builds start and is passed
unchanged to both Debian package builds and the update bundle. Versioned
packaging requires exact `<package>_<version>_armhf.{deb,changes,buildinfo}`
outputs; artifacts from older builds do not satisfy this check. Before moving
the files into `build/`, `build.py` also reads the `.deb` control fields and
requires the expected package name, exact version, and `armhf` architecture.

Passing `--docker` to `package_kvmcomm`, `package_nanokvm_go`, or `release`
runs the Debian package build inside the Docker packaging environment. Component
builds and `package_update` still run through the normal host-side `build.py`
flow, and Debian artifacts are still collected into root `build/`.

`package_kvmcomm` expects:

```text
build/kvm_vin
build/kvm_ui
build/nano_ocr_app
build/nomad-uvc-feeder
build/nanokvm-update-verify
build/update-keys.json
debhelper/packages/kvmcomm/rootfs/usr/bin/ax_efuse_tool
debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv3_det_npu.axmodel
debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ch_PP_OCRv4_rec_npu.axmodel
debhelper/packages/kvmcomm/rootfs/kvmcomm/models/pp_ocr/ppocr_keys_v1.txt
```

`package_nanokvm_go` expects:

```text
build/NanoKVM-Server
build/web/
build/libkvm.so*
build/memfab
```

Packaged binary locations:

```text
kvmcomm:    /kvmcomm/vin/kvm_vin
kvmcomm:    /kvmcomm/ui/kvm_ui
kvmcomm:    /kvmcomm/ocr/nano_ocr_app
kvmcomm:    /kvmcomm/webcam/nomad-uvc-feeder
kvmcomm:    /usr/bin/ax_efuse_tool
kvmcomm:    /usr/bin/nanokvm-update-verify
kvmcomm:    /usr/share/nanokvm/update-keys.json
nanokvm-go: /usr/bin/memfab
```

Debian package artifacts are moved into the root `build/` directory:

```text
build/kvmcomm_<version>_armhf.deb
build/kvmcomm_<version>_armhf.changes
build/kvmcomm_<version>_armhf.buildinfo
build/nanokvm-go_<version>_armhf.deb
build/nanokvm-go_<version>_armhf.changes
build/nanokvm-go_<version>_armhf.buildinfo
```

## Application Update Signing

`0.0.25` already exists as a Git tag but did not ship this embedded-signature
update, so it must not be rebuilt or overwritten. The first release that uses
this format is `0.0.26`.

When `--signing-key` is supplied, `package_update` signs the two final Debian
packages, then embeds one signature in each package's JSON inside the
application tar. Without `--signing-key`, it generates the same archive layout
but omits both `signature` fields. The tar has one top-level directory and
exactly four regular files:

```text
nanokvm-go_<version>/
├── nanokvm-go_<version>_armhf.deb
├── nanokvm-go_<version>.json
├── kvmcomm_<version>_armhf.deb
└── kvmcomm_<version>.json
```

Each internal JSON describes its corresponding Debian package. A signed bundle
also authenticates the package with the final `signature` member:

```json
{
  "version": "0.0.26",
  "name": "kvmcomm_0.0.26_armhf.deb",
  "sha512": "<Base64 SHA-512>",
  "size": 12345678,
  "package": "kvmcomm",
  "architecture": "armhf",
  "signature": {
    "schema": 1,
    "algorithm": "ed25519",
    "keyId": "sha256:<64 lowercase hex characters>",
    "value": "<Base64 64-byte Ed25519 signature>"
  }
}
```

The Ed25519 input is the exact, complete byte sequence of the final `.deb`.
The JSON, its SHA-512 string, and the outer tar are not the signature input.
The two debs are signed and verified independently. The `size` and `sha512`
fields retain their integrity role; with signature verification enabled by
default, successful Ed25519 verification against the packaged key ring supplies
publisher authenticity. The authenticated update setting may explicitly skip
Ed25519 verification for application updates, but it does not relax the exact
package identity, version, architecture, size, SHA-512, or archive checks.

An unsigned bundle completely omits `signature`; it never writes `null`, an
empty object, or a placeholder signature. A device with the default signature
verification setting rejects that bundle. It can be installed only after an
authenticated user explicitly disables application update signature
verification. This exception applies only to application updates: Firmware
and factory-recovery Firmware remain signed and fail closed.

The application CDN publish set is exactly:

```text
build/update/nanokvm-go_<version>.tar.gz
build/update/latest.json
```

Publish it under `https://cdn.sipeed.com/nanokvm-go/v2/` (stable) or
`v2/preview/`, and Firmware pairs under `v2/firmware/` or `v2/preview/firmware/`.

`latest.json` retains the existing four fields (`version`, `name`, `sha512`,
and `size`). It locates and checks the transport of the tar but does not
authorize installation. Do not generate or publish an external
`nanokvm-go_<version>.json`, `latest.json.sig`, or
`nanokvm-go_<version>.json.sig`. The `nanokvm-go_<version>.json` inside the tar
is a different, mandatory file: it describes and authenticates the
`nanokvm-go` deb when signed, and every install path must reach that package
metadata.

For a signed `release`, `build.py` first derives the public key and key ID,
writes `build/update-keys.json`, and packages that key ring and
`nanokvm-update-verify` into `kvmcomm`. Before signing either deb, it extracts
the final `kvmcomm` deb, requires both installed files, and byte-compares its
key ring with the release key ring. This prevents a bundle from being signed
for a key that the packaged verifier cannot trust.

Unless `--trusted-public-key` is supplied, the key ring pins both checked-in
official public keys under `support/keys/` (the offline root and the CI key),
whether the build is unsigned or signed by either of them. A build signed by
any other key pins only that key. An empty key ring cannot verify a later
signed application or Firmware update after the unsigned release is installed,
so `package_kvmcomm` refuses one unless `--allow-empty-keyring` is passed.

`--trusted-public-key` replaces the defaults rather than adding to them. A key
ring that pins some official keys but not all of them is refused unless
`--allow-partial-official-keys` is passed, so a rotation cannot silently drop
the CI key or the root. To create an unsigned test bundle from already-built
debs without rebuilding their packaged key ring, prefer
`package_update --version <version>`.

### Nightly builds

`.github/workflows/nightly.yml` runs whenever a CI run for a push to `main`
succeeds, and on demand. It takes the unsigned bundle of the head of `main`
from that commit's successful CI run and republishes it as the rolling
`nightly` prerelease, replacing the previous one. It skips the run when
`nightly` already points at that commit, or when `main` has moved on and its
new head has no successful CI run yet; that run's completion publishes it.
It does not rebuild, sign, or touch the CDN, but it refuses a bundle whose key
ring, in `update-keys.json` and inside the kvmcomm package, is not exactly the
official one.

A nightly carries the `debian/changelog` version, so it reports the same plain
`X.Y.Z` as the last release and is identified only by the commit in its
release notes. Testers install it by offline upload after switching off
application update signature verification, and switch verification back on
afterwards. Because the packaged key ring is the official one, later signed
releases still install over a nightly. A device on a nightly reports the last
release's version, so that same release is not offered online; reinstall it by
offline upload.

### Releasing a tested nightly

Merge the release commit that bumps the two `debian/changelog` files and
`CHANGELOG.md`, and test the nightly built from it. Then:

1. Run **Create Tag** (`.github/workflows/create-tag.yml`) with the version.
   Leave the commit empty to tag the head of `main`, or give the SHA of the
   tested nightly; a short SHA of 7 or more characters is enough. It refuses unless the tag is a new `X.Y.Z` equal to the
   `debian/changelog` version at that commit, and the commit is on `main` with
   a successful CI run for a push to `main`. A tag created with `GITHUB_TOKEN`
   starts no workflow, and GitHub refuses it for a commit whose
   `.github/workflows` differ from `main`. With **Start Release on the new
   tag** checked (the default) it then starts step 2 itself, passing on
   **Publish to the CDN after signing**.
2. **Release** (`.github/workflows/release.yml`) runs from `main` with the tag;
   run it by hand when Create Tag did not start it. With **Publish to the CDN
   after signing** checked it publishes as soon as the bundle is signed;
   otherwise publish later with Publish to CDN.
   Release builds the tagged commit with the CI pipeline and signs only a
   commit that is on `main`.

On the public repository the `X.Y.Z` tag ruleset must let this workflow create
tags; see the comment at the top of `create-tag.yml` for the trade-off.

### Publishing to the CDN

With **Publish to the CDN after signing** checked, the Release workflow ends
with `.github/workflows/publish-cdn.yml`; run it by hand from `main` with the
tag to publish a release later. It runs `ci/scripts/publish_cdn.py` against the stable channel named by `QINIU_PREFIX`
and then makes the draft GitHub release public. Before uploading it checks
the release assets against `SHA256SUMS`, verifies the bundle with the device
verifier and the official key ring, and checks that `latest.json` matches the
tarball and that the tarball holds only `nanokvm-go_<version>/`.

The script uploads the tarball, refreshes its CDN URL and waits until the CDN
serves it with the expected SHA-512 for both identity and gzip requests. Only
then does it upload `latest.json`, refresh it, and wait until the CDN serves
the new `latest.json` for both kinds of request. It refuses to:

- publish to a channel that has no `latest.json` yet, so a mistyped prefix
  cannot start a new channel. The first release on a channel, such as the
  root-signed v2 bridge release, is published by running the script by hand
  with `--initialize-channel`, which refuses a channel that already exists;
- downgrade the channel, or replace a published version or tarball with
  different contents.

Running Publish to CDN again with the tag retries a failed publish without
rebuilding or re-signing; it skips uploads whose contents and stored metadata
are already right, and uploads again objects whose metadata is wrong. Leave
**Publish to the CDN after signing** unchecked until the bridge release is on
the channel.

The publish job is a second deployment to the `release` environment, so a
release asks its reviewers twice: once to sign and once to publish.

## Firmware Update Signing

Firmware is signed independently after the final `.tar.gz` has been produced.
Generate its metadata with the same release key ring:

```bash
python3 support/scripts/package_firmware.py \
  --version <firmware-version> \
  --archive /offline/artifacts/axera_firmware_v<firmware-version>.tar.gz \
  --signing-key /offline/path/update-signing.pem \
  --keyring build/update-keys.json \
  --output /publish/firmware_<firmware-version>.json
```

By default the metadata contains `"updateUboot": false`, so the updater skips
U-Boot even when the signed archive contains a U-Boot image. Add
`--update-uboot` only when that release is intended to install the archive's
U-Boot image; it emits `"updateUboot": true`. This is an explicit opt-in.

The generated metadata embeds the signature:

```json
{
  "version": "1.0.6",
  "name": "axera_firmware_v1.0.6.tar.gz",
  "sha512": "<Base64 SHA-512>",
  "size": 12345678,
  "updateUboot": false,
  "signature": {
    "schema": 1,
    "algorithm": "ed25519",
    "keyId": "sha256:<64 lowercase hex characters>",
    "value": "<Base64 64-byte Ed25519 signature>"
  }
}
```

The signature input is the exact, complete byte sequence of the final Firmware
archive, not the JSON or its SHA-512 string. Do not recompress, rewrite, rename,
or otherwise mutate the archive after signing. The supported release pair is
the `.tar.gz` plus `firmware_<version>.json`; do not publish a
`firmware_<version>.json.sig` sidecar. The Firmware tool is intentionally
separate from `build.py release`: the archive is an external input and the
private key is never copied into `build/`.

`updateUboot` is also outside the current signature input because the JSON is
not signed. It controls only whether the updater installs the U-Boot image
already contained in the signed archive; it does not permit replacement of that
image. Upgrade the device verifier/updater to a version that understands this
field before publishing such metadata. On the new updater, both a missing field
and the generated default `false` skip U-Boot. An older updater rejects metadata
that contains this new field, while its legacy behavior for metadata without
the field is to update U-Boot. Do not publish this metadata until the device
updater has been upgraded.

## Signing-Key Boundary

The Ed25519 private key is an external, read-only release-operator input. Keep
it outside the repository and never copy it into Git, `build/`, a Debian
package, the application tar, a Firmware archive, a log, the CDN, or a device.
Only the public key ring is packaged. The key ID is the lowercase hexadecimal
SHA-256 of the raw 32-byte Ed25519 public key, prefixed with `sha256:`.

Application package/version pairs and Firmware versions are immutable signing
identities: do not rebuild or replace already published bytes under the same
version. Publish application artifacts in dependency order (tar, then
`latest.json`) and Firmware artifacts as the final archive followed by its
metadata. Do not use `debsigs` or `dpkg-sig`; future APT repository signing
remains a separate OpenPGP `InRelease` trust chain.

For key rotation, first add the next public key to a release still signed by
the current key, deploy that client, and verify coverage before switching the
signer. See `support/keys/README.md` for the root and CI key roles. List every
key that must stay pinned next to the new one:

```bash
python3 build.py release --version <version> \
  --signing-key /offline/path/current.pem \
  --trusted-public-key support/keys/nanokvm-go-release.pub.pem \
  --trusted-public-key support/keys/nanokvm-go-ci.pub.pem \
  --trusted-public-key /offline/path/next-public.pem
```

To replace a compromised CI key, sign with the root, list the root and the new
CI key, and pass `--allow-partial-official-keys` to drop the old CI key; then
replace `support/keys/nanokvm-go-ci.pub.pem` in the same change.

Recover from a failed rotation by publishing a new version signed by a key that
is still pinned; never replace already signed bytes under an existing version.
Signature verification itself does not implement anti-downgrade policy, and
factory recovery may reinstall the same valid version. Do not publish a
replacement public key beside CDN objects—the device trusts only its packaged
key ring.

## Docker Packaging

Docker packaging is available through the root `build.py` driver:

```bash
python3 build.py package_kvmcomm --docker
python3 build.py package_nanokvm_go --docker
python3 build.py package_kvmcomm --version <version> --docker
python3 build.py package_nanokvm_go --version <version> --docker
python3 build.py release --version <version> --docker
python3 build.py release --version <version> \
  --signing-key /offline/path/update-signing.pem --docker
```

Build required component inputs with `python3 build.py build` first, or use the
specific component commands listed above. The lower-level helper remains
available for direct debugging:

```bash
./debhelper/docker/trixie/build_in_docker.sh kvmcomm
./debhelper/docker/trixie/build_in_docker.sh nanokvm-go
```

## Cleaning

```bash
python3 build.py clean
```

`clean` removes the root `build/` directory and Debian packaging generated
state, including `.debhelper/`, `debian/files`, `debian/*.substvars`,
`debian/*.debhelper`, `debian/debhelper-build-stamp`, and package staging
directories such as `debian/kvmcomm/` and `debian/nanokvm-go/`.

## Notes

- `ax_efuse_tool` is not migrated as a source/build target; the prebuilt
  `debhelper/packages/kvmcomm/rootfs/usr/bin/ax_efuse_tool` is packaged into
  `/usr/bin/ax_efuse_tool`.
- `support/toolchains/`, component `build/` directories, Conan state, Debian
  staging directories, and root `build/` outputs are local generated state.
