# Included in kvmcomm's maintainer scripts by package_nomad_normal.sh.
# Run in a subshell so traps and positional arguments stay local.
nomad_kernel() (
    set -eu
    fail() { echo "Nomad kernel: $*" >&2; exit 1; }
    test "$(cat /boot/ver)" = 'nanokvm-go-2026-10-08-v1.0.10' || fail 'requires Go+ firmware v1.0.10'
    case "$(ax_efuse_tool --str-get)" in
        NanoKVM-Go-Plus\[*) ;;
        *) fail 'could not confirm a NanoKVM Go+ model' ;;
    esac
    grep -Eq '(^| )root=/dev/mmcblk1p2( |$)' /proc/cmdline || fail 'requires SD system boot'
    target=/boot/kernel.img
    test -s "$target" && test ! -L "$target" || fail 'missing or invalid boot kernel'
    test "$1" = configure || exit 0

    payload=/usr/share/nomad-kernel/kernel.img
    expected=@KERNEL_SHA256@
    hash() { sha256sum "$1" | cut -d' ' -f1; }
    test "$(hash "$payload")" = "$expected" || fail 'packaged kernel checksum mismatch'
    previous=$(hash "$target")
    test -n "$previous" || fail 'could not read current boot kernel'
    if test "$previous" = "$expected"; then
        echo 'Nomad kernel already installed.'
        exit 0
    fi

    # Hard links preserve the old inode across the atomic replacement, including retries.
    backup="/boot/kernel.img.nomad-backup-$previous"
    if test ! -e "$backup"; then ln "$target" "$backup"; fi
    test ! -L "$backup" && test "$(hash "$backup")" = "$previous" || fail 'invalid kernel backup'
    candidate=$(mktemp /boot/.nomad-kernel.XXXXXX)
    trap 'rm -f -- "$candidate"' EXIT
    install -m 0644 "$payload" "$candidate"
    test "$(hash "$candidate")" = "$expected" || fail 'staged kernel checksum mismatch'
    sync -f /boot
    touch /run/reboot-required
    mv -fT "$candidate" "$target"
    sync -f /boot
    echo "Nomad webcam kernel installed. Reboot the Go+ after the update finishes. Backup: $backup"
)
