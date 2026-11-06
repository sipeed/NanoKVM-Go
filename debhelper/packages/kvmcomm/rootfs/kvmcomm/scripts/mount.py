#!/usr/bin/env python3

import subprocess
import sys
import argparse
import os
import shlex
import time
import dbus
import contextlib
import fcntl
import json

debug = False
data_mount_path = "/data"
data_mount_timeout_seconds = 15
min_image_size_mb = 5120
reserved_space_mb = 5120
storage_lock_path = "/run/nanokvm-storage.lock"
usb_access_marker = "/run/nanokvm-data-usb"
data_image_path = "/exfat.img"
gadget_functions = "/sys/kernel/config/usb_gadget/g0/functions"


@contextlib.contextmanager
def storage_lock():
    # USB rebuilds and the server pass the same open file description on fd 3.
    # Closing our duplicate must not unlock the parent's transaction.
    if os.environ.get("NANOKVM_STORAGE_LOCK_FD") == "3":
        if not os.path.samestat(os.fstat(3), os.stat(storage_lock_path)):
            raise RuntimeError("Invalid storage lock descriptor")
        descriptor = os.dup(3)
    else:
        descriptor = os.open(storage_lock_path, os.O_RDWR | os.O_CREAT | os.O_CLOEXEC, 0o600)
    try:
        try:
            fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise RuntimeError("Storage is busy; finish or cancel the file transfer first") from error
        yield
    finally:
        os.close(descriptor)


def backing_file(disk):
    try:
        with open(f"{gadget_functions}/mass_storage.{disk}/lun.0/file") as backing:
            return backing.read().strip()
    except FileNotFoundError:
        return ""


def require_storage_detached():
    if backing_file("disk1"):
        raise RuntimeError("Disconnect the virtual disk first")
    if backing_file("disk0"):
        raise RuntimeError("Unmount the selected image first")


def data_mount():
    result = subprocess.run(
        ["findmnt", "--json", "--mountpoint", data_mount_path, "--output", "SOURCE,FSTYPE,OPTIONS"],
        capture_output=True, text=True, check=False,
    )
    if result.returncode == 1 and not result.stdout.strip():
        return None
    if result.returncode != 0:
        raise RuntimeError(f"Cannot inspect {data_mount_path}: {result.stderr.strip()}")
    mounts = json.loads(result.stdout).get("filesystems", [])
    if len(mounts) != 1:
        raise RuntimeError(f"Unexpected mount state for {data_mount_path}")
    mounted = mounts[0]
    source = mounted["source"]
    if mounted["fstype"] != "exfat" or not source.startswith("/dev/loop"):
        raise RuntimeError(f"Unexpected backing device for {data_mount_path}: {source}")
    result = subprocess.run(
        ["losetup", "--noheadings", "--output", "BACK-FILE", source],
        capture_output=True, text=True, check=True,
    )
    if os.path.realpath(result.stdout.strip()) != os.path.realpath(data_image_path):
        raise RuntimeError(f"Unexpected backing file for {data_mount_path}")
    return mounted


def verify_data_mode(writable):
    mounted = data_mount()
    if mounted is None or ("rw" in mounted["options"].split(",")) != writable:
        raise RuntimeError(f"Failed to make {data_mount_path} {'writable' if writable else 'read-only'}")


def prepare_usb_storage():
    if not os.path.isfile(data_image_path):
        raise RuntimeError("Virtual disk image does not exist")
    if backing_file("disk0"):
        raise RuntimeError("Unmount the selected image first")
    if data_mount() is None:
        manage_data_mount("enable")
    if run(["mount", "-o", "remount,ro", data_mount_path]) != 0:
        raise RuntimeError("Storage is busy; cannot make /data read-only")
    verify_data_mode(False)
    # Set before exposing the image: even a failed/aborted USB rebuild must
    # invalidate the filesystem cache before local writes resume.
    with open(usb_access_marker, "w"):
        pass


def refresh_readonly_storage():
    # The gadget writes the regular backing file, while /data reads a loop
    # block device with a separate cache. Recreate the read-only mount rather
    # than dropping VFS dentries and leaving stale loop blocks behind. The
    # caller holds the exclusive storage lock; no local reader is active.
    if backing_file("disk1") != data_image_path or backing_file("disk0"):
        raise RuntimeError("The shared disk is not exclusively owned by USB")
    verify_data_mode(False)
    manage_data_mount("unmount")
    manage_data_mount("enable")
    verify_data_mode(False)


def restore_local_storage(refresh=False):
    if backing_file("disk1"):
        raise RuntimeError("Disconnect the virtual disk first")
    if not os.path.exists(data_image_path):
        return
    mounted = data_mount()
    fresh = refresh or os.path.exists(usb_access_marker)
    if mounted is not None and "rw" in mounted["options"].split(",") and not fresh:
        return
    require_storage_detached()
    if mounted is not None:
        # A normal unmount fails on open images/files. Never use lazy unmount.
        manage_data_mount("unmount")
    # A fresh mount also replaces a loop device created read-only at boot;
    # remount,rw alone cannot make that loop device writable.
    if run([
        "mount", "-t", "exfat", "-o",
        "loop,rw,relatime,fmask=0022,dmask=0022,iocharset=utf8",
        data_image_path, data_mount_path,
    ]) != 0:
        raise RuntimeError("Failed to mount local storage read-write")
    verify_data_mode(True)
    try:
        os.remove(usb_access_marker)
    except FileNotFoundError:
        pass

def log(*args, **kwargs):
    if debug:
        print(*args, **kwargs)

def run(cmd, check=True, timeout=None):
    display_cmd = shlex.join(cmd)
    log(f"[RUN] {display_cmd}")
    try:
        result = subprocess.run(
            cmd,
            shell=False,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=timeout
        )

        if debug:
            if result.stdout:
                log("[STDOUT]", result.stdout.strip())
            if result.stderr:
                log("[STDERR]", result.stderr.strip())

        if check and result.returncode != 0:
            log(f"[ERROR] Command failed: {display_cmd}")
            log(f"Exit code: {result.returncode}")
            if result.stderr:
                log(f"Error message: {result.stderr.strip()}")
            return result.returncode

        return result.returncode

    except subprocess.TimeoutExpired:
        log(f"[TIMEOUT] {display_cmd} exceeded {timeout}s")
        return -1
    except FileNotFoundError:
        log(f"[ERROR] Command not found: {cmd[0]}")
        return 127

def wait_for_data_mount(mounted, timeout=data_mount_timeout_seconds):
    deadline = time.monotonic() + timeout
    while os.path.ismount(data_mount_path) != mounted:
        if time.monotonic() >= deadline:
            state = "mount" if mounted else "unmount"
            raise RuntimeError(f"Timed out waiting for {data_mount_path} to {state}")
        time.sleep(0.1)

def manage_data_mount(action):
    bus = dbus.SystemBus()
    systemd = bus.get_object('org.freedesktop.systemd1', '/org/freedesktop/systemd1')
    mgr = dbus.Interface(systemd, 'org.freedesktop.systemd1.Manager')

    if action == 'enable':
        mgr.EnableUnitFiles(['data.mount'], False, False)
        if run(['systemctl', 'start', 'data.mount'], timeout=data_mount_timeout_seconds) != 0:
            raise RuntimeError("Failed to start data.mount")
        wait_for_data_mount(True)
        log("Enabled and started data.mount")
    elif action in ('disable', 'unmount'):
        if run(['systemctl', 'stop', 'data.mount'], timeout=data_mount_timeout_seconds) != 0:
            raise RuntimeError("Failed to stop data.mount")
        wait_for_data_mount(False)
        if action == 'disable':
            mgr.DisableUnitFiles(['data.mount'], False)
        log("Disabled and stopped data.mount")
    else:
        log("Invalid action. Use 'enable' or 'disable'.")

def clear_data_directory():
    data_dir = "/data"
    if os.path.exists(data_dir):
        import shutil
        for item in os.listdir(data_dir):
            item_path = os.path.join(data_dir, item)
            try:
                if os.path.isfile(item_path) or os.path.islink(item_path):
                    os.unlink(item_path)
                elif os.path.isdir(item_path):
                    shutil.rmtree(item_path)
            except Exception as e:
                log(f"Failed to remove {item_path}: {e}")
        log("Cleared /data directory.")
    else:
        log("/data directory does not exist.")

def create_exfat_image(img_file="exfat.img", size_mb=500):
    size_bytes = size_mb * 1024 * 1024
    log(f"Creating {size_mb}MB exFAT image: {img_file}")
    try:
        with open(img_file, 'wb') as f:
            os.ftruncate(f.fileno(), size_bytes)
    except Exception as e:
        log(f"Failed to create file: {e}")
        sys.exit(1)
    if run(["mkfs.exfat", img_file]) != 0:
        try:
            os.remove(img_file)
        except OSError:
            pass
        raise RuntimeError(f"Failed to format exFAT image: {img_file}")
    log(f"exFAT image {img_file} created successfully.")

def get_default_image_size_mb(
    target_dir='/', reserve_mb=reserved_space_mb, min_required_mb=min_image_size_mb
):
    try:
        result = subprocess.run(
            ["df", "-BM", "--", target_dir],
            shell=False,
            capture_output=True,
            text=True,
            check=True
        )
        lines = [line for line in result.stdout.splitlines() if line.strip()]
        if len(lines) < 2:
            raise ValueError("empty df output")

        fields = lines[-1].split()
        if len(fields) < 4 or not fields[3].endswith('M'):
            raise ValueError("unexpected df output")

        available_mb = int(fields[3][:-1])
        default_size = available_mb - reserve_mb

        if default_size < min_required_mb:
            print(f"[ERROR] Not enough free space on {target_dir}: available {available_mb}MB", file=sys.stderr)
            sys.exit(1)

        return default_size
    except subprocess.CalledProcessError as e:
        print(f"[ERROR] Failed to check disk space: {e}", file=sys.stderr)
        sys.exit(1)
    except ValueError as e:
        print(f"[ERROR] Failed to parse disk space output: {e}", file=sys.stderr)
        sys.exit(1)

def validate_image_size_mb(size_mb, target_dir='/'):
    max_size_mb = get_default_image_size_mb(target_dir=target_dir)
    if size_mb < min_image_size_mb or size_mb > max_size_mb:
        print(
            f"[ERROR] Image size must be between {min_image_size_mb}MB and {max_size_mb}MB",
            file=sys.stderr
        )
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Manage exFAT image creation.")
    subparsers = parser.add_subparsers(dest='command', required=True, help='Available commands')

    start_parser = subparsers.add_parser('start', help='Create exFAT image and clear data directory')
    start_parser.add_argument('--size', type=int, default=None, help='Size of the image in MB (default: available space - 5120MB)')
    start_parser.add_argument('--file', default='/exfat.img', help='Output image file name (default: /exfat.img)')
    start_parser.add_argument('--debug', action='store_true', help='Enable debug output')
    stop_parser = subparsers.add_parser('stop', help='Disable data mount')
    stop_parser.add_argument('--debug', action='store_true', help='Enable debug output')
    restart_parser = subparsers.add_parser('restart', help='Restart data mount')
    restart_parser.add_argument('--debug', action='store_true', help='Enable debug output')
    remove_parser = subparsers.add_parser('remove', help='Remove exFAT image and disable mount')
    remove_parser.add_argument('--file', default='/exfat.img', help='Image file name to remove (default: /exfat.img)')
    remove_parser.add_argument('--debug', action='store_true', help='Enable debug output')
    for command in ('readonly', 'local', 'refresh-readonly'):
        mode_parser = subparsers.add_parser(command, help='Switch storage ownership under the shared lock')
        mode_parser.add_argument('--debug', action='store_true')

    args = parser.parse_args()
    global debug
    debug = args.debug

    with storage_lock():
        execute_action(args)


def execute_action(args):
    if args.command == 'readonly':
        prepare_usb_storage()
    elif args.command == 'refresh-readonly':
        refresh_readonly_storage()
    elif args.command == 'local':
        restore_local_storage()
    elif args.command == 'start':
        if not os.path.exists(args.file):
            require_storage_detached()
            target_dir = os.path.dirname(os.path.abspath(args.file)) or '/'
            if args.size is None:
                args.size = get_default_image_size_mb(target_dir=target_dir)
            else:
                validate_image_size_mb(args.size, target_dir=target_dir)
            create_exfat_image(img_file=args.file, size_mb=args.size)
            clear_data_directory()
        manage_data_mount('enable')
        if backing_file('disk1'):
            prepare_usb_storage()
        else:
            restore_local_storage()

    elif args.command == 'stop':
        if backing_file('disk0'):
            raise RuntimeError("Unmount the selected image first")
        manage_data_mount('disable')

    elif args.command == 'restart':
        require_storage_detached()
        restore_local_storage(refresh=True)

    elif args.command == 'remove':
        require_storage_detached()
        manage_data_mount('disable')
        if os.path.exists(args.file):
            try:
                os.remove(args.file)
            except Exception:
                restore_local_storage()
                raise

if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"[ERROR] {error}", file=sys.stderr)
        sys.exit(1)
