package storage

import (
	"errors"
	"sync"
	"time"

	"NanoKVM-Server/storageguard"
)

var sharedRefreshMu sync.Mutex
var sharedLastRefresh time.Time

// USB writes /exfat.img, while /data reads a loop device over that file.
// Dropping dentries alone does not invalidate the loop device's block cache.
// Refresh the read-only mount under the same exclusive lock used by every
// USB transition; active readers keep their mount until their handles close.
// Refresh is demand-driven and throttled so a closed storage page costs no IO.
func refreshSharedStorage() error {
	sharedRefreshMu.Lock()
	defer sharedRefreshMu.Unlock()
	if time.Since(sharedLastRefresh) < time.Second {
		return nil
	}
	status, err := readDiskStatus()
	if err != nil || status.State != diskStatePartition {
		return err
	}
	lock, err := storageguard.Exclusive()
	if errors.Is(err, storageguard.ErrBusy) {
		return nil
	}
	if err != nil {
		return err
	}
	defer lock.Close()
	status, err = readDiskStatus()
	if err != nil || status.State != diskStatePartition {
		return err
	}
	if err := runStorageMount("refresh-readonly", lock); err != nil {
		return err
	}
	sharedLastRefresh = time.Now()
	return nil
}
