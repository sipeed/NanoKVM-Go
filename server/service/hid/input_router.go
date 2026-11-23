package hid

import (
	"errors"
	"sync"
)

const (
	internalInputSessionBase                 = uint64(1) << 52
	InternalInputSessionPaste         uint64 = internalInputSessionBase + 1
	InternalInputSessionAI            uint64 = internalInputSessionBase + 2
	InternalInputSessionJiggler       uint64 = internalInputSessionBase + 3
	InternalInputSessionKeyboardQueue uint64 = internalInputSessionBase + 4
	InternalInputSessionMouseQueue    uint64 = internalInputSessionBase + 5
)

type InputRouterSink interface {
	RouteVirtualKeyboardSession(session uint64, report []byte) error
	RouteVirtualPointerSession(session uint64, report []byte) error
	RouteVirtualTouchSession(session uint64, report []byte) error
	ReleaseInputSession(session uint64, reason string) error
	ResetInputSessions(reason string) error
	BeginInputQuiesce(reason string) (uint64, error)
	EndInputQuiesce(epoch uint64, reason string) error
}

var ErrInputRouterUnavailable = errors.New("input router is unavailable")

var inputRouterState struct {
	sync.RWMutex
	sink InputRouterSink
}

func SetInputRouterSink(sink InputRouterSink) {
	inputRouterState.Lock()
	inputRouterState.sink = sink
	inputRouterState.Unlock()
}

func InputRouterSinkInstalled() bool {
	inputRouterState.RLock()
	defer inputRouterState.RUnlock()
	return inputRouterState.sink != nil
}

func (h *Hid) RouteKeyboardInput(session uint64, report []byte) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.RouteVirtualKeyboardSession(session, report)
	}
	return h.WriteKeyboardReport(report)
}

func (h *Hid) RoutePointerInput(session uint64, report []byte) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.RouteVirtualPointerSession(session, report)
	}
	switch len(report) {
	case 4:
		return h.WriteRelativeMouseReport(report)
	case 7:
		return h.WriteAbsoluteMouseReport(report)
	default:
		return errors.New("invalid virtual pointer report length")
	}
}

func (h *Hid) RouteTouchInput(session uint64, report []byte) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.RouteVirtualTouchSession(session, report)
	}
	return h.WriteTouchScreenReport(report)
}

func (h *Hid) ReleaseInputSession(session uint64, reason string) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.ReleaseInputSession(session, reason)
	}
	return nil
}

func (h *Hid) ResetInputRouting(reason string) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.ResetInputSessions(reason)
	}
	return h.ReleaseAllInputs()
}

func (h *Hid) BeginInputRoutingQuiesce(reason string) (uint64, error) {
	if sink := inputRouterSink(); sink != nil {
		return sink.BeginInputQuiesce(reason)
	}
	return 0, ErrInputRouterUnavailable
}

func (h *Hid) EndInputRoutingQuiesce(epoch uint64, reason string) error {
	if sink := inputRouterSink(); sink != nil {
		return sink.EndInputQuiesce(epoch, reason)
	}
	return ErrInputRouterUnavailable
}

func RouteKeyboardInput(session uint64, report []byte) error {
	return GetHid().RouteKeyboardInput(session, report)
}

func RoutePointerInput(session uint64, report []byte) error {
	return GetHid().RoutePointerInput(session, report)
}

// RouteReliablePointerInput requires a successful write of the full report.
// Relative movement and wheel deltas return a failed write without replay.
func RouteReliablePointerInput(session uint64, report []byte) error {
	sink, ok := inputRouterSink().(interface {
		RouteReliableVirtualPointerSession(session uint64, report []byte) error
	})
	if !ok {
		return ErrInputRouterUnavailable
	}
	return sink.RouteReliableVirtualPointerSession(session, report)
}

func RouteTouchInput(session uint64, report []byte) error {
	return GetHid().RouteTouchInput(session, report)
}

func ReleaseInputSession(session uint64, reason string) error {
	return GetHid().ReleaseInputSession(session, reason)
}

func ResetInputRouting(reason string) error {
	return GetHid().ResetInputRouting(reason)
}

func BeginInputRoutingQuiesce(reason string) (uint64, error) {
	return GetHid().BeginInputRoutingQuiesce(reason)
}

func EndInputRoutingQuiesce(epoch uint64, reason string) error {
	return GetHid().EndInputRoutingQuiesce(epoch, reason)
}

func inputRouterSink() InputRouterSink {
	inputRouterState.RLock()
	defer inputRouterState.RUnlock()
	return inputRouterState.sink
}
