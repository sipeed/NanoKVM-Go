package hid

import (
	"context"
	"sync"
	"time"
)

type Service struct {
	hid                *Hid
	pointerModeManager *pointerModeManager
	pointerModeSwitch  *pointerModeTransaction
	pointerModeAPIMu   sync.Mutex
	pasteSleep         func(time.Duration)
	pasteMutex         sync.Mutex
}

func NewService() *Service {
	h := GetHid()
	manager := getPointerModeManager()
	service := &Service{
		hid:                h,
		pointerModeManager: manager,
		pasteSleep:         time.Sleep,
	}
	service.pointerModeSwitch = newPointerModeTransaction(manager, h, pointerModeTransactionOptions{})
	return service
}

// SwitchPointerMode runs the backend transaction core used by the pointer-mode
// API. HID reset reuses its locked recovery entry point when runtime state is
// degraded.
func (service *Service) SwitchPointerMode(ctx context.Context, mode PointerMode) (PointerModeStatus, error) {
	return service.pointerModeTransaction().run(ctx, mode)
}

func (service *Service) pointerModeTransaction() *pointerModeTransaction {
	transaction := service.pointerModeSwitch
	if transaction == nil {
		manager := service.pointerModeManager
		if manager == nil {
			manager = getPointerModeManager()
		}
		h := service.hid
		if h == nil {
			h = GetHid()
		}
		transaction = newPointerModeTransaction(manager, h, pointerModeTransactionOptions{})
	}
	return transaction
}
