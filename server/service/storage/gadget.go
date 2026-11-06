package storage

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/storage/diskfs"
	"NanoKVM-Server/storageguard"
	"NanoKVM-Server/utils"
	log "github.com/sirupsen/logrus"
)

const (
	gadgetRoot         = "/sys/kernel/config/usb_gadget/g0"
	gadgetUDC          = gadgetRoot + "/UDC"
	sharedDiskMarker   = "/boot/usb.disk1"
	storageMountScript = "/kvmcomm/scripts/mount.py"
)

type diskState string

const (
	diskStateNone        diskState = "none"
	diskStateImage       diskState = "image"
	diskStatePartition   diskState = "partition"
	diskStateUnsupported diskState = "unsupported"
)

type diskStatus struct {
	State    diskState
	Backing  string
	HidOnly  bool
	Cdrom    bool
	ReadOnly bool
	Lun      string
}

var ErrMediumLocked = errors.New("the target machine is using the medium")
var readDiskStatus = func() (diskStatus, error) { return readDiskStatusAt(gadgetRoot, pathExists(hid.HidOnlyFlag)) }

func pathExists(path string) bool { _, err := os.Stat(path); return err == nil }

func readDiskStatusAt(root string, hidOnly bool) (diskStatus, error) {
	if hidOnly || !pathExists(filepath.Join(root, "UDC")) {
		return diskStatus{State: diskStateUnsupported, HidOnly: hidOnly}, nil
	}
	status := diskStatus{State: diskStateNone}
	for _, disk := range []string{"disk0", "disk1"} {
		lun := filepath.Join(root, "functions/mass_storage."+disk, "lun.0")
		if !pathExists(filepath.Join(root, "configs/c.1/mass_storage."+disk)) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(lun, "file"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return diskStatus{}, err
		}
		backing := strings.TrimSpace(string(data))
		if backing == "" {
			continue
		}
		if status.Backing != "" {
			return diskStatus{}, errors.New("both virtual disks have a medium attached")
		}
		status.Backing = backing
		status.Lun = lun
		status.State = diskStateImage
		if disk == "disk1" {
			if backing != partitionDevice {
				return diskStatus{}, errors.New("unexpected shared disk backing file")
			}
			status.State = diskStatePartition
		} else {
			for name, field := range map[string]*bool{"cdrom": &status.Cdrom, "ro": &status.ReadOnly} {
				data, err := os.ReadFile(filepath.Join(lun, name))
				if err != nil {
					return diskStatus{}, err
				}
				*field = strings.TrimSpace(string(data)) == "1"
			}
		}
	}
	return status, nil
}

func mutateMedia(fn func(*storageguard.Lock) error) error {
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()
	lock, err := storageguard.Exclusive()
	if err != nil {
		return err
	}
	defer lock.Close()
	return fn(lock)
}

func withMediaHID(fn func() error) error {
	h := hid.GetHid()
	if err := h.ResetInputRouting("virtual media transition"); err != nil {
		log.Warnf("failed to release input before media transition: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() { h.OpenNoLock(); h.Unlock() }()
	return fn()
}

func (s *Service) attachMedium(backing string, cdrom, ro, force bool) error {
	ro = ro || cdrom
	return mutateMedia(func(lock *storageguard.Lock) error {
		status, err := readDiskStatus()
		if err != nil {
			return err
		}
		if status.State == diskStateUnsupported {
			return errStorageDisabled
		}
		if backing == partitionDevice && cdrom {
			return errNotOptical
		}
		if sameMediaPath(backing, status.Backing) && cdrom == status.Cdrom && (backing == partitionDevice || status.ReadOnly == ro) {
			return nil
		}
		if backing == partitionDevice && !pathExists(partitionDevice) {
			return errStorageDisabled
		}
		// Reject missing images and invalid optical requests before ejecting
		// the target's current medium. A shared disk needs its read-only loop
		// mount refreshed before checking a file the host just copied there.
		if backing != partitionDevice {
			if status.State == diskStatePartition {
				if err := runStorageMount("refresh-readonly", lock); err != nil {
					return err
				}
			}
			canonical, err := utils.CanonicalMediaPath(backing)
			if err != nil {
				return err
			}
			info, err := os.Lstat(canonical)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return utils.ErrInvalidMediaID
			}
			if cdrom {
				kind, err := diskfs.Detect(canonical)
				if err != nil {
					return err
				}
				if !kind.Optical() {
					return errNotOptical
				}
			}
			backing = canonical
		}
		return withMediaHID(func() error {
			if err := detachLun(status, force); err != nil {
				return err
			}
			if backing == partitionDevice {
				if err := removeMarker(usbDisk); err != nil {
					return err
				}
				if err := os.WriteFile(sharedDiskMarker, nil, 0o644); err != nil {
					return err
				}
				err := runUSBRestart(lock)
				after, stateErr := readDiskStatus()
				if err == nil && stateErr == nil && after.State == diskStatePartition {
					return nil
				}
				if err == nil {
					err = errors.New("shared disk did not attach")
				}
				if stateErr == nil && after.Backing == "" {
					err = errors.Join(err, runStorageMount("local", lock))
				}
				return errors.Join(err, stateErr)
			}
			// The script releases disk1 and recreates the local loop mount before
			// disk0 receives a file that the target machine may just have copied.
			if err := restartUSB(1, lock); err != nil {
				rollbackImageMount(lock)
				return err
			}
			fail := func(err error) error { rollbackImageMount(lock); return err }
			canonical, err := utils.CanonicalMediaPath(backing)
			if err != nil {
				return fail(err)
			}
			info, err := os.Lstat(canonical)
			if err != nil {
				return fail(err)
			}
			if !info.Mode().IsRegular() {
				return fail(utils.ErrInvalidMediaID)
			}
			if cdrom {
				kind, err := diskfs.Detect(canonical)
				if err != nil {
					return fail(err)
				}
				if !kind.Optical() {
					return fail(errNotOptical)
				}
			}
			if err := os.WriteFile(cdromFlag, []byte(boolAttr(cdrom)), 0o666); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(roFlag, []byte(boolAttr(ro || cdrom)), 0o666); err != nil {
				return fail(err)
			}
			setMediumName(filepath.Dir(mountDevice), canonical)
			if err := os.WriteFile(mountDevice, []byte(canonical), 0o666); err != nil {
				return fail(err)
			}
			if err := rebindUDC(); err != nil {
				return fail(err)
			}
			return nil
		})
	})
}

func (s *Service) detachMedium(force bool) error {
	return mutateMedia(func(lock *storageguard.Lock) error {
		status, err := readDiskStatus()
		if err != nil {
			return err
		}
		if status.State == diskStateNone {
			return runStorageMount("local", lock)
		}
		if status.State == diskStateUnsupported {
			return nil
		}
		return withMediaHID(func() error {
			if err := detachLun(status, force); err != nil {
				return err
			}
			if err := removeMarker(usbDisk); err != nil {
				return err
			}
			// Retain the shared-disk preference, but leave its LUN empty until the
			// next explicit mount. Eject must not immediately export it again.
			if err := runStorageMount("local", lock); err != nil {
				return err
			}
			setMediumName(status.Lun, "")
			return rebindUDC()
		})
	})
}

func detachLun(status diskStatus, force bool) error {
	if status.Backing == "" {
		return nil
	}
	err := os.WriteFile(filepath.Join(status.Lun, "file"), []byte("\n"), 0o666)
	if !errors.Is(err, syscall.EBUSY) {
		return err
	}
	if !force {
		return ErrMediumLocked
	}
	if err := os.WriteFile(gadgetUDC, []byte("\n"), 0o666); err != nil {
		return err
	}
	// Restore the other USB functions even when the second detach fails.
	err = os.WriteFile(filepath.Join(status.Lun, "file"), []byte("\n"), 0o666)
	return errors.Join(err, rebindUDC())
}
func removeMarker(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func runUSBRestart(lock *storageguard.Lock) error {
	cmd := usbRestartCommand()
	lock.Attach(cmd)
	return cmd.Run()
}
func runStorageMount(action string, lock *storageguard.Lock) error {
	cmd := exec.Command(storageMountScript, action)
	lock.Attach(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storage %s: %w: %s", action, err, strings.TrimSpace(string(out)))
	}
	return nil
}
func boolAttr(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
func cutName(name string, size int) string {
	if len(name) <= size {
		return name
	}
	name = name[:size]
	for !utf8.ValidString(name) {
		name = name[:len(name)-1]
	}
	return name
}
func setMediumName(lun, backing string) {
	values := map[string]string{
		"inquiry_string":       "sipeed  Flash Drive     1.00",
		"inquiry_string_cdrom": "sipeed  Optical Drive   1.00",
	}
	product := "NanoKVM-Go"
	if backing != "" {
		name := strings.TrimSuffix(filepath.Base(backing), filepath.Ext(backing))
		product = cutName(name, 126)
		for attr := range values {
			values[attr] = fmt.Sprintf("%-8s%-16s%s", "sipeed", cutName(name, 16), "1.00")
		}
	} else {
		if configured, err := os.ReadFile("/boot/usb.product"); err == nil {
			product = strings.TrimSpace(string(configured))
		}
		for attr := range values {
			if configured, err := os.ReadFile("/boot/disk." + attr); err == nil {
				values[attr] = strings.TrimSpace(string(configured))
			}
		}
	}
	for attr, value := range values {
		// Older kernels have no separate optical inquiry string.
		path := filepath.Join(lun, attr)
		if attr == "inquiry_string_cdrom" && !pathExists(path) {
			continue
		}
		if err := os.WriteFile(path, []byte(value), 0o666); err != nil {
			log.Debugf("set medium inquiry name: %s", err)
		}
	}
	if err := os.WriteFile(gadgetRoot+"/strings/0x409/product", []byte(product), 0o666); err != nil {
		log.Debugf("set USB product name: %s", err)
	}
}
func rebindUDC() error {
	devices, err := os.ReadDir("/sys/class/udc")
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return errors.New("no USB device controller")
	}
	if err := os.WriteFile(gadgetUDC, []byte("\n"), 0o666); err != nil && !errors.Is(err, syscall.ENODEV) {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return os.WriteFile(gadgetUDC, []byte(devices[0].Name()+"\n"), 0o666)
}
