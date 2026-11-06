// Package storageguard coordinates local file access with USB storage rebuilds.
package storageguard

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const LockPath = "/run/nanokvm-storage.lock"

var (
	ErrBusy        = errors.New("storage is busy; finish or cancel the file transfer first")
	ErrReadOnly    = errors.New("disconnect the virtual disk before writing local storage")
	ErrUnavailable = errors.New("local image storage is not mounted")
)

type Lock struct{ file *os.File }

func acquire(path string, exclusive bool) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	if err := unix.Flock(int(file.Fd()), mode|unix.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, err
	}
	return &Lock{file: file}, nil
}

func Exclusive() (*Lock, error) { return acquire(LockPath, true) }

// Close does not explicitly unlock: an in-flight child must retain its lock
// even if the server exits. The last reference releases the flock.
func (lock *Lock) Close() { _ = lock.file.Close() }

// Attach lets usbdev.sh and mount.py reuse this transaction on descriptor 3.
func (lock *Lock) Attach(command *exec.Cmd) {
	command.ExtraFiles = []*os.File{lock.file}
	command.Env = append(os.Environ(), "NANOKVM_STORAGE_LOCK_FD=3")
}

type storagePaths struct {
	image, directory, lun, mountInfo, devices string
	imageLun, usbMarker                       string
}

var systemPaths = storagePaths{
	image: "/exfat.img", directory: "/data",
	lun:       "/sys/kernel/config/usb_gadget/g0/functions/mass_storage.disk1/lun.0/file",
	mountInfo: "/proc/self/mountinfo", devices: "/sys/dev/block",
	imageLun:  "/sys/kernel/config/usb_gadget/g0/functions/mass_storage.disk0/lun.0/file",
	usbMarker: "/run/nanokvm-data-usb",
}

// Local holds a shared lock for the entire file operation. With no virtual
// disk image, /data remains the ordinary image directory on the rootfs.
func Local(writable bool) (func(), error) {
	return local(systemPaths, LockPath, writable, func(lock *Lock) error {
		command := exec.Command("/kvmcomm/scripts/mount.py", "local")
		lock.Attach(command)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("restore local storage: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	})
}

func local(paths storagePaths, lockPath string, writable bool, restore func(*Lock) error) (func(), error) {
	lock, err := acquire(lockPath, false)
	if err != nil {
		return nil, err
	}
	err = checkLocal(paths, writable)
	if writable && errors.Is(err, ErrReadOnly) {
		// A host-side eject empties the LUN without calling our eject API. Only
		// recover a known USB ownership handoff, never an unrelated read-only
		// filesystem or a disk that the target still holds.
		ready, inspectErr := canRestoreLocal(paths)
		if inspectErr != nil {
			err = inspectErr
		} else if ready {
			lock.Close()
			lock, err = acquire(lockPath, true)
			if err != nil {
				return nil, err
			}
			err = restoreLocalAccess(paths, lock, restore)
		}
	}
	if err != nil {
		lock.Close()
		return nil, err
	}
	return lock.Close, nil
}

func restoreLocalAccess(paths storagePaths, lock *Lock, restore func(*Lock) error) error {
	// The gadget may have changed while we exchanged locks.
	err := checkLocal(paths, true)
	if errors.Is(err, ErrReadOnly) {
		ready, inspectErr := canRestoreLocal(paths)
		if inspectErr != nil {
			return inspectErr
		}
		if ready {
			err = restore(lock)
		}
	}
	if err != nil {
		return err
	}
	// Keep the lease through the caller's operation. Recheck after downgrading
	// too, because flock conversions need not be atomic.
	if err := unix.Flock(int(lock.file.Fd()), unix.LOCK_SH|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return ErrBusy
		}
		return err
	}
	return checkLocal(paths, true)
}

func canRestoreLocal(paths storagePaths) (bool, error) {
	if _, err := os.Stat(paths.usbMarker); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	for _, path := range []string{paths.lun, paths.imageLun} {
		backing, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if strings.TrimSpace(string(backing)) != "" {
			return false, nil
		}
	}
	return true, nil
}

func checkLocal(paths storagePaths, writable bool) error {
	if writable {
		backing, err := os.ReadFile(paths.lun)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if strings.TrimSpace(string(backing)) != "" {
			return ErrReadOnly
		}
	}
	if _, err := os.Stat(paths.image); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	contents, err := os.ReadFile(paths.mountInfo)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || unescapeMount(fields[4]) != paths.directory {
			continue
		}
		separator := strings.Index(line, " - ")
		if separator < 0 || !strings.HasPrefix(line[separator+3:], "exfat ") {
			return ErrUnavailable
		}
		backing, err := os.ReadFile(filepath.Join(paths.devices, fields[2], "loop/backing_file"))
		if err != nil {
			return fmt.Errorf("inspect data loop device: %w", err)
		}
		if filepath.Clean("/"+strings.TrimPrefix(strings.TrimSpace(string(backing)), "/")) != paths.image {
			return ErrUnavailable
		}
		if writable && !strings.Contains(","+fields[5]+",", ",rw,") {
			return ErrReadOnly
		}
		return nil
	}
	return ErrUnavailable
}

func unescapeMount(value string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(value)
}
