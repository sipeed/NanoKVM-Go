#!/usr/bin/env bash
set -euo pipefail

release_version="${1:-}"
if [[ -z "$release_version" ]]; then
    echo "usage: $0 <release-version>" >&2
    exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

test -s build/web/index.html
test "$(go env GOVERSION)" = "go1.26.5"
test "$(node --version)" = "v22.17.1"
test "$(pnpm --version)" = "11.3.0"
test "$(conan --version)" = "Conan version 2.30.0"
test "$(cmake --version | head -n 1)" = "cmake version 3.31.6"
test "$(/opt/armv7-toolchain/bin/arm-none-linux-gnueabihf-gcc -dumpmachine)" = \
    "arm-none-linux-gnueabihf"
test -f /opt/armv7-toolchain/arm-none-linux-gnueabihf/libc/usr/include/opus/opus.h
test -e /opt/armv7-toolchain/arm-none-linux-gnueabihf/libc/usr/lib/libopus.so
test -f /opt/armv7-toolchain/arm-none-linux-gnueabihf/libc/usr/include/systemd/sd-bus.h
test -e /opt/armv7-toolchain/arm-none-linux-gnueabihf/libc/usr/lib/libsystemd.so

toolchain_link="$repo_root/support/toolchains/armv7-toolchain"
mkdir -p "$(dirname "$toolchain_link")"
if [[ -e "$toolchain_link" || -L "$toolchain_link" ]]; then
    rm -rf -- "$toolchain_link"
fi
ln -s /opt/armv7-toolchain "$toolchain_link"

./support/scripts/toolchain_setup_components.sh \
    --config support/scripts/nanokvm-go.ini \
    --switch
conan profile detect --force
conan remote disable conancenter

test -f support/toolchains/toolchain.ini
test -f libkvm/profile/nanokvm-go
test -f kvm_vin/profile/nanokvm-go
test -f kvm_ui/profile/nanokvm-go

# build.py pins the checked-in official public keys by default, so the packaged
# key ring matches an official release even though CI holds no private key.
for target in libkvm kvm_vin kvm_ui nano_ocr memfab server; do
    python3 build.py "$target"
done

python3 build.py package_kvmcomm
python3 build.py package_nanokvm_go

verify_debian_artifacts() {
    local package_name="$1"
    local artifact_prefix="build/${package_name}_${release_version}_armhf"
    local suffix

    for suffix in deb changes buildinfo; do
        test -s "${artifact_prefix}.${suffix}"
    done

    test "$(dpkg-deb --field "${artifact_prefix}.deb" Package)" = "$package_name"
    test "$(dpkg-deb --field "${artifact_prefix}.deb" Version)" = "$release_version"
    test "$(dpkg-deb --field "${artifact_prefix}.deb" Architecture)" = "armhf"
    grep -Fx "Version: $release_version" "${artifact_prefix}.changes"
    grep -Fx "Architecture: armhf" "${artifact_prefix}.changes"
    grep -Fx "Version: $release_version" "${artifact_prefix}.buildinfo"
    grep -Fx "Architecture: armhf" "${artifact_prefix}.buildinfo"
}

verify_debian_artifacts kvmcomm
verify_debian_artifacts nanokvm-go

python3 build.py package_update --version "$release_version"

# These artifacts carry no Ed25519 signature, so a device only installs them
# with signature verification switched off. Mark them so nobody mistakes a CI
# build for something publishable.
cat > build/update/UNSIGNED-DO-NOT-DISTRIBUTE <<'MARKER'
This bundle was produced by CI and is NOT signed.

Installing it requires signature verification to be switched off on the
device. Official releases are signed offline with the private key, which
never enters CI. Do not publish these files to the release CDN.
MARKER
python3 ci/scripts/verify_release_artifacts.py \
    --build-dir build \
    --version "$release_version"

test -s "build/kvmcomm_${release_version}_armhf.deb"
test -s "build/nanokvm-go_${release_version}_armhf.deb"
test -s "build/update/nanokvm-go_${release_version}.tar.gz"
test -s build/update/latest.json
