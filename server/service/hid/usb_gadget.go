package hid

import (
	"context"
	"sync"
	"time"
)

var usbGadgetMutationMu sync.Mutex

const usbGadgetMutationLockPoll = 10 * time.Millisecond

// LockUsbGadgetMutation serializes complete USB gadget rebuild transactions.
// Callers must release it with UnlockUsbGadgetMutation.
func LockUsbGadgetMutation() {
	usbGadgetMutationMu.Lock()
}

// LockUsbGadgetMutationContext acquires the process-wide gadget mutation lock
// while honoring ctx cancellation. The existing blocking LockUsbGadgetMutation
// API remains available for older callers that do not have a request context.
func LockUsbGadgetMutationContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if usbGadgetMutationMu.TryLock() {
		if err := ctx.Err(); err != nil {
			usbGadgetMutationMu.Unlock()
			return err
		}
		return nil
	}

	ticker := time.NewTicker(usbGadgetMutationLockPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := ctx.Err(); err != nil {
				return err
			}
			if usbGadgetMutationMu.TryLock() {
				if err := ctx.Err(); err != nil {
					usbGadgetMutationMu.Unlock()
					return err
				}
				return nil
			}
		}
	}
}

func UnlockUsbGadgetMutation() {
	usbGadgetMutationMu.Unlock()
}
