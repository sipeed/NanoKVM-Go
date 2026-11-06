package application

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

// applicationUpdateLease keeps the in-process admission slot and both
// cross-process locks until a transient systemd worker has successfully exec'd.
// The worker opens the same locks in blocking mode before the parent releases
// them. Firmware and factory-reset entry points also recheck the fixed unit
// after acquiring the shared maintenance lock, closing the handoff window.
type applicationUpdateLease struct {
	once            sync.Once
	applicationLock *os.File
	maintenanceLock *os.File
	processSlot     bool
}

func (lease *applicationUpdateLease) Release() {
	if lease == nil {
		return
	}
	lease.once.Do(func() {
		unlockApplicationFile(lease.maintenanceLock)
		unlockApplicationFile(lease.applicationLock)
		if lease.processSlot {
			updateMutex.Lock()
			isUpdating = false
			updateMutex.Unlock()
		}
	})
}

func beginApplicationUpdateAdmission(
	source UpdateSource,
	version string,
	phase UpdatePhase,
) (*applicationUpdateLease, ApplicationUpdateStatus, error) {
	lease := &applicationUpdateLease{}

	updateMutex.Lock()
	if isUpdating {
		updateMutex.Unlock()
		return nil, ApplicationUpdateStatus{}, errApplicationUpdateBusy
	}
	isUpdating = true
	lease.processSlot = true
	updateMutex.Unlock()

	fail := func(err error) (*applicationUpdateLease, ApplicationUpdateStatus, error) {
		lease.Release()
		return nil, ApplicationUpdateStatus{}, err
	}

	busy, err := applicationInstallUnitBlocksAdmission()
	if err != nil {
		return fail(err)
	}
	if busy {
		return fail(errApplicationUpdateBusy)
	}

	lease.applicationLock, err = openAndLockApplicationFile(applicationUpdateLockPath, true)
	if err != nil {
		if errors.Is(err, errApplicationUpdateBusy) {
			return fail(errApplicationUpdateBusy)
		}
		return fail(fmt.Errorf("acquire application update lock: %w", err))
	}
	lease.maintenanceLock, err = openAndLockApplicationFile(deviceMaintenanceLockPath, true)
	if err != nil {
		if errors.Is(err, errApplicationUpdateBusy) {
			return fail(errApplicationUpdateBusy)
		}
		return fail(fmt.Errorf("acquire device maintenance lock: %w", err))
	}

	// Recheck after taking both locks. A unit which entered activating between
	// the first check and flock must win; no request body has been read yet.
	busy, err = applicationInstallUnitBlocksAdmission()
	if err != nil {
		return fail(err)
	}
	if busy {
		return fail(errApplicationUpdateBusy)
	}

	status, err := beginApplicationUpdate(source, version, phase)
	if err != nil {
		return fail(err)
	}
	broadcastApplicationUpdateStatus(status)
	return lease, status, nil
}

func applicationInstallUnitBlocksAdmission() (bool, error) {
	unit, err := inspectApplicationUpdateUnit()
	if err != nil {
		return false, err
	}
	return unit.active(), nil
}

func openAndLockApplicationFile(path string, nonBlocking bool) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("application lock path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	operation := unix.LOCK_EX
	if nonBlocking {
		operation |= unix.LOCK_NB
	}
	if err := unix.Flock(int(file.Fd()), operation); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, errApplicationUpdateBusy
		}
		return nil, err
	}
	return file, nil
}

func unlockApplicationFile(file *os.File) {
	if file == nil {
		return
	}
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
	_ = file.Close()
}

func acquireApplicationUpdateWorkerLease() (*applicationUpdateLease, error) {
	lease := &applicationUpdateLease{}
	var err error
	lease.applicationLock, err = openAndLockApplicationFile(applicationUpdateLockPath, false)
	if err != nil {
		return nil, fmt.Errorf("wait for application update lock: %w", err)
	}
	lease.maintenanceLock, err = openAndLockApplicationFile(deviceMaintenanceLockPath, false)
	if err != nil {
		lease.Release()
		return nil, fmt.Errorf("wait for device maintenance lock: %w", err)
	}
	return lease, nil
}
