#!/usr/bin/env bash
# Run on Linux. The input directory contains the original application-only debs.
set -euo pipefail
if [[ $# != 5 ]]; then
    echo "Usage: $0 APP_VERSION BUILD_NUMBER INPUT_DEB_DIR VERIFIED_KERNEL_IMG OUTPUT_DIR" >&2
    exit 2
fi
version=$1
number=$2
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ && $number =~ ^[1-9][0-9]*$ ]]
input=$(realpath "$3")
kernel=$(realpath "$4")
scripts=$(dirname "$(realpath "$0")")
mkdir -p "$5"
out=$(realpath "$5")
[[ $input != "$out" ]]
stage=$(mktemp -d "$out/.nomad-normal.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
name=kvmcomm_${version}_armhf.deb
app=nanokvm-go_${version}_armhf.deb
archive=nanokvm-go_${version}-nomad.${number}.tar.gz
root=$stage/root
dpkg-deb -R "$input/$name" "$root"
[[ ! -e "$root/usr/share/nomad-kernel" ]] || { echo "Use the original application-only debs as inputs" >&2; exit 1; }
# Pin the signed image for this firmware line. A new kernel requires explicit review.
digest=10594e683bf2fa062e067dbcf816b98b04443e4fe0e124c88ec7a9a0cc41e0d0
echo "$digest  $kernel" | sha256sum -c -
install -Dm644 "$kernel" "$root/usr/share/nomad-kernel/kernel.img"
for phase in preinst postinst; do
    path=$root/DEBIAN/$phase
    if [[ -e $path ]]; then
        [[ $(head -n 1 "$path") == '#!/bin/sh' ]]
        tail -n +2 "$path" > "$stage/original"
    else
        : > "$stage/original"
    fi
    {
        printf '#!/bin/sh\nset -e\n'
        sed "s/@KERNEL_SHA256@/$digest/g" "$scripts/nomad-kernel.sh"
        if [[ $phase == preinst ]]; then
            echo 'case "$1" in install|upgrade) nomad_kernel check ;; esac'
        else
            echo 'if [ "$1" = configure ]; then nomad_kernel configure; fi'
        fi
        cat "$stage/original"
    } > "$path"
    chmod 755 "$path"
done
(cd "$root" && md5sum usr/share/nomad-kernel/kernel.img >> DEBIAN/md5sums)
size=$(du -sk --apparent-size --exclude=DEBIAN "$root" | cut -f1)
sed -i "s/^Installed-Size: .*/Installed-Size: $size/" "$root/DEBIAN/control"
dpkg-deb --root-owner-group -Zxz -b "$root" "$stage/$name"
cp "$input/$app" "$stage/$app"
python3 "$scripts/package_update.py" --version "$version" --input-dir "$stage" \
    --output-dir "$stage/release" --archive-name "$archive"
mkdir -p "$out/release"
mv "$stage/$name" "$stage/$app" "$out/"
mv "$stage/release/$archive" "$out/release/"
mv "$stage/release/latest.json" "$out/release/"
