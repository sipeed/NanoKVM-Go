package inputrouter

import (
	"errors"
	"fmt"
)

var (
	ErrInputRouterQuiescing     = errors.New("input router is quiescing")
	ErrStaleInputRoutingEpoch   = errors.New("input event belongs to a stale routing epoch")
	ErrInputRouterNotQuiescing  = errors.New("input router is not quiescing")
	ErrInputRouterEpochMismatch = errors.New("input router quiesce epoch mismatch")
)

func inputEpochQuiescing(epoch uint64) bool {
	return epoch&1 != 0
}

func inputEpochGate(eventEpoch, currentEpoch uint64) (error, string, string) {
	if inputEpochQuiescing(eventEpoch) {
		return fmt.Errorf("%w: epoch %d", ErrInputRouterQuiescing, eventEpoch),
			"quiescing", "input rejected while the router is quiescing"
	}
	if eventEpoch != currentEpoch {
		return fmt.Errorf(
			"%w: event epoch %d, current epoch %d",
			ErrStaleInputRoutingEpoch,
			eventEpoch,
			currentEpoch,
		), "stale-epoch", "input rejected after the routing epoch changed"
	}
	return nil, "", ""
}

// BeginInputQuiesce closes the current routing epoch before waiting for an
// in-flight dispatch. Once it returns, all previously accepted reports have
// completed, the old active HID profile has been released, and session state
// has been discarded. The returned odd epoch is required to resume routing.
// A release error leaves the router quiesced so callers can rebuild or roll
// back without admitting input across the device boundary.
func (router *Router) BeginInputQuiesce(reason string) (uint64, error) {
	var quiesceEpoch uint64
	for {
		currentEpoch := router.routingEpoch.Load()
		if inputEpochQuiescing(currentEpoch) {
			return 0, fmt.Errorf("%w: epoch %d", ErrInputRouterQuiescing, currentEpoch)
		}
		quiesceEpoch = currentEpoch + 1
		if router.routingEpoch.CompareAndSwap(currentEpoch, quiesceEpoch) {
			break
		}
	}

	router.dispatch.Lock()
	err := router.releaseAndClearInputSessionsLocked("quiesce", reason)
	router.dispatch.Unlock()
	return quiesceEpoch, err
}

// EndInputQuiesce starts a fresh routing epoch. Reports captured before or
// during quiescing retain an older epoch and are rejected even if they were
// waiting for the dispatch lock until after this method returns.
func (router *Router) EndInputQuiesce(quiesceEpoch uint64, reason string) error {
	router.dispatch.Lock()
	defer router.dispatch.Unlock()

	currentEpoch := router.routingEpoch.Load()
	if !inputEpochQuiescing(currentEpoch) {
		return fmt.Errorf("%w: epoch %d", ErrInputRouterNotQuiescing, currentEpoch)
	}
	if quiesceEpoch != currentEpoch {
		return fmt.Errorf(
			"%w: token %d, current epoch %d",
			ErrInputRouterEpochMismatch,
			quiesceEpoch,
			currentEpoch,
		)
	}
	if !router.routingEpoch.CompareAndSwap(currentEpoch, currentEpoch+1) {
		return fmt.Errorf("%w: epoch changed from %d", ErrInputRouterEpochMismatch, currentEpoch)
	}
	router.traces.Publish(TraceRecord{
		Timestamp: router.now().UnixMilli(),
		Stage:     "resume",
		Meaning:   reason,
	})
	return nil
}

func (router *Router) releaseAndClearInputSessionsLocked(stage, reason string) error {
	var err error
	if router.releaser != nil {
		err = router.releaser.ReleaseAllInputs()
	}

	// Clear state even when release fails. Restoring an old owner or held report
	// after USB re-enumeration is less safe than reporting the release failure.
	clear(router.states)
	clear(router.owners)
	clear(router.sessions)

	record := TraceRecord{
		Timestamp: router.now().UnixMilli(),
		Stage:     stage,
		Meaning:   reason,
	}
	if err != nil {
		router.dispatchFailed.Add(1)
		router.reject(err)
		record.Error = err.Error()
	}
	router.traces.Publish(record)
	return err
}
