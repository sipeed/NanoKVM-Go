#!/bin/sh
set -e

root=${1:-}
timezone=America/New_York
zoneinfo=/usr/share/zoneinfo/$timezone

if [ ! -f "$root$zoneinfo" ]; then
    echo "Missing time zone file: $zoneinfo" >&2
    exit 1
fi

ln -sf "..$zoneinfo" "$root/etc/localtime"
printf '%s\n' "$timezone" > "$root/etc/timezone"
