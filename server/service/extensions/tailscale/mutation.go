package tailscale

import (
	"errors"
	"sync"
	"sync/atomic"
)

// errTailscaleMutationBusy reports that another Tailscale mutation is already
// in flight. It is the shared busy signal for every entry point that funnels
// into tailscaleMutations, including the root-local gate driver, so callers
// can map it to their own transport's busy response.
var errTailscaleMutationBusy = errors.New("tailscale operation already in progress")

const (
	mutationActive uint32 = 1 << iota
	mutationInstalling
)

// mutationCoordinator rejects overlapping Tailscale mutations without holding
// a mutex for the lifetime of a potentially long-running operation. Keeping
// the active and installing bits in one atomic word also makes installing()
// an immediate, consistent status read.
type mutationCoordinator struct {
	state atomic.Uint32
}

func (m *mutationCoordinator) tryBegin(installing bool) (func(), bool) {
	next := mutationActive
	if installing {
		next |= mutationInstalling
	}
	if !m.state.CompareAndSwap(0, next) {
		return nil, false
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			m.state.Store(0)
		})
	}
	return release, true
}

func (m *mutationCoordinator) installing() bool {
	return m.state.Load()&mutationInstalling != 0
}

var tailscaleMutations = &mutationCoordinator{}
