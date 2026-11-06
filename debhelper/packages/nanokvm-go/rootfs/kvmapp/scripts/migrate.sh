#!/bin/sh
set -u

migration_root="${NANOKVM_MIGRATION_ROOT:-}"
case "$migration_root" in
	"") ;;
	/*) migration_root="${migration_root%/}" ;;
	*)
		echo "invalid migration root: $migration_root" >&2
		exit 2
		;;
esac

migration_dir="$migration_root/var/lib/nanokvm-go-update/migrations"
timelapse_config="$migration_root/etc/kvm/timelapse.json"

migrate_0_2_1() {
	pending_file="$migration_dir/0.2.1.pending"

	if [ -L "$pending_file" ]; then
		echo "migration pending marker must not be a symbolic link: $pending_file" >&2
		return 1
	fi
	if [ ! -e "$pending_file" ]; then
		return 0
	fi
	if [ ! -f "$pending_file" ]; then
		echo "migration pending marker is not a regular file: $pending_file" >&2
		return 1
	fi

	/usr/bin/python3 - "$timelapse_config" "$pending_file" <<'PY'
import json
import os
import stat
import sys
import tempfile

CONFIG_MAX_BYTES = 16 * 1024
CONFIG_VERSION = 1
RETENTION_DAYS = 180
MAX_STORAGE_MIB = 48 * 1024
MAX_REVISION = (1 << 63) - 1


class MigrationError(Exception):
    pass


def reject_duplicate_fields(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise MigrationError(f"duplicate JSON field: {key}")
        value[key] = item
    return value


def read_config(path):
    try:
        info = os.lstat(path)
    except FileNotFoundError:
        return None

    if stat.S_ISLNK(info.st_mode) or not stat.S_ISREG(info.st_mode):
        raise MigrationError("timelapse config is not a regular file")
    if info.st_size > CONFIG_MAX_BYTES:
        raise MigrationError("timelapse config is too large")

    flags = os.O_RDONLY | getattr(os, "O_CLOEXEC", 0) | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        opened = os.fstat(descriptor)
        if not stat.S_ISREG(opened.st_mode) or opened.st_size > CONFIG_MAX_BYTES:
            raise MigrationError("timelapse config changed while opening")
        data = os.read(descriptor, CONFIG_MAX_BYTES + 1)
    finally:
        os.close(descriptor)

    if len(data) > CONFIG_MAX_BYTES:
        raise MigrationError("timelapse config is too large")
    try:
        config = json.loads(
            data,
            object_pairs_hook=reject_duplicate_fields,
            parse_constant=lambda value: (_ for _ in ()).throw(
                MigrationError(f"invalid JSON constant: {value}")
            ),
        )
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise MigrationError(f"invalid timelapse config: {error}") from error
    return config


def validate_config(config):
    if not isinstance(config, dict) or set(config) != {"version", "revision", "cleanup"}:
        raise MigrationError("timelapse config has unexpected or missing fields")
    if type(config["version"]) is not int or config["version"] != CONFIG_VERSION:
        raise MigrationError("unsupported timelapse config version")
    if type(config["revision"]) is not int or not 0 <= config["revision"] <= MAX_REVISION:
        raise MigrationError("invalid timelapse config revision")

    cleanup = config["cleanup"]
    if not isinstance(cleanup, dict) or set(cleanup) != {
        "enabled",
        "retentionDays",
        "maxStorageMiB",
    }:
        raise MigrationError("timelapse cleanup config has unexpected or missing fields")
    if type(cleanup["enabled"]) is not bool:
        raise MigrationError("invalid timelapse cleanup enabled value")
    if type(cleanup["retentionDays"]) is not int or not 0 <= cleanup["retentionDays"] <= 3650:
        raise MigrationError("invalid timelapse retention days")
    if (
        type(cleanup["maxStorageMiB"]) is not int
        or not 0 <= cleanup["maxStorageMiB"] <= MAX_REVISION
    ):
        raise MigrationError("invalid timelapse storage limit")
    if cleanup["enabled"] and cleanup["retentionDays"] == 0 and cleanup["maxStorageMiB"] == 0:
        raise MigrationError("enabled cleanup requires a positive retention limit")


def sync_directory(path):
    descriptor = os.open(path, os.O_RDONLY | getattr(os, "O_CLOEXEC", 0) | os.O_DIRECTORY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def write_config(path, config):
    directory = os.path.dirname(path)
    descriptor, temporary_path = tempfile.mkstemp(prefix=".timelapse.json.", dir=directory)
    try:
        os.fchmod(descriptor, 0o600)
        data = (json.dumps(config, indent=2, ensure_ascii=False) + "\n").encode()
        with os.fdopen(descriptor, "wb", closefd=True) as temporary:
            descriptor = -1
            temporary.write(data)
            temporary.flush()
            os.fsync(temporary.fileno())
        os.replace(temporary_path, path)
        temporary_path = ""
        sync_directory(directory)
    finally:
        if descriptor >= 0:
            os.close(descriptor)
        if temporary_path:
            try:
                os.unlink(temporary_path)
            except FileNotFoundError:
                pass


def finish_migration(pending_path):
    pending = os.lstat(pending_path)
    if stat.S_ISLNK(pending.st_mode) or not stat.S_ISREG(pending.st_mode):
        raise MigrationError("migration pending marker is not a regular file")
    os.unlink(pending_path)
    sync_directory(os.path.dirname(pending_path))


def main():
    config_path, pending_path = sys.argv[1:]
    config = read_config(config_path)
    if config is None:
        finish_migration(pending_path)
        return

    validate_config(config)

    cleanup = config["cleanup"]
    changed = (
        not cleanup["enabled"]
        or cleanup["retentionDays"] != RETENTION_DAYS
        or cleanup["maxStorageMiB"] != MAX_STORAGE_MIB
    )
    if changed:
        if config["revision"] == MAX_REVISION:
            raise MigrationError("timelapse config revision is exhausted")
        config = {
            "version": CONFIG_VERSION,
            "revision": config["revision"] + 1,
            "cleanup": {
                "enabled": True,
                "retentionDays": RETENTION_DAYS,
                "maxStorageMiB": MAX_STORAGE_MIB,
            },
        }
        write_config(config_path, config)
    finish_migration(pending_path)


try:
    main()
except (MigrationError, OSError, ValueError) as error:
    print(f"screen timelapse migration 0.2.1 failed: {error}", file=sys.stderr)
    raise SystemExit(1)
PY
}

migrate_0_2_1
