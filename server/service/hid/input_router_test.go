package hid

import (
	"bytes"
	"errors"
	"testing"
)

type inputRouterSinkRecorder struct {
	keyboardSession uint64
	keyboardReport  []byte
	pointerSession  uint64
	pointerReport   []byte
	touchSession    uint64
	touchReport     []byte
	releasedSession uint64
	releaseReason   string
	resetReason     string
	quiesceReason   string
	quiesceEpoch    uint64
	resumeReason    string
	resumedEpoch    uint64
}

func (recorder *inputRouterSinkRecorder) RouteVirtualKeyboardSession(session uint64, report []byte) error {
	recorder.keyboardSession = session
	recorder.keyboardReport = bytes.Clone(report)
	return nil
}

func (recorder *inputRouterSinkRecorder) RouteVirtualPointerSession(session uint64, report []byte) error {
	recorder.pointerSession = session
	recorder.pointerReport = bytes.Clone(report)
	return nil
}

func (recorder *inputRouterSinkRecorder) RouteVirtualTouchSession(session uint64, report []byte) error {
	recorder.touchSession = session
	recorder.touchReport = bytes.Clone(report)
	return nil
}

func (recorder *inputRouterSinkRecorder) ReleaseInputSession(session uint64, reason string) error {
	recorder.releasedSession = session
	recorder.releaseReason = reason
	return nil
}

func (recorder *inputRouterSinkRecorder) ResetInputSessions(reason string) error {
	recorder.resetReason = reason
	return nil
}

func (recorder *inputRouterSinkRecorder) BeginInputQuiesce(reason string) (uint64, error) {
	recorder.quiesceReason = reason
	recorder.quiesceEpoch = 17
	return recorder.quiesceEpoch, nil
}

func (recorder *inputRouterSinkRecorder) EndInputQuiesce(epoch uint64, reason string) error {
	recorder.resumedEpoch = epoch
	recorder.resumeReason = reason
	return nil
}

func TestHidRoutesInternalReportsThroughInstalledSink(t *testing.T) {
	recorder := &inputRouterSinkRecorder{}
	SetInputRouterSink(recorder)
	t.Cleanup(func() { SetInputRouterSink(nil) })

	h := &Hid{}
	keyboard := []byte{0, 0, 4, 0, 0, 0, 0, 0}
	pointer := []byte{0, 1, 2, 3}
	touch := make([]byte, TouchScreenReportSize)
	touch[0] = TouchScreenReportID

	if err := h.RouteKeyboardInput(11, keyboard); err != nil {
		t.Fatalf("RouteKeyboardInput: %v", err)
	}
	if err := h.RoutePointerInput(12, pointer); err != nil {
		t.Fatalf("RoutePointerInput: %v", err)
	}
	if err := h.RouteTouchInput(13, touch); err != nil {
		t.Fatalf("RouteTouchInput: %v", err)
	}
	if err := h.ReleaseInputSession(14, "test"); err != nil {
		t.Fatalf("ReleaseInputSession: %v", err)
	}
	if err := h.ResetInputRouting("rebuild"); err != nil {
		t.Fatalf("ResetInputRouting: %v", err)
	}
	epoch, err := h.BeginInputRoutingQuiesce("pointer mode change")
	if err != nil {
		t.Fatalf("BeginInputRoutingQuiesce: %v", err)
	}
	if err := h.EndInputRoutingQuiesce(epoch, "pointer mode changed"); err != nil {
		t.Fatalf("EndInputRoutingQuiesce: %v", err)
	}

	if recorder.keyboardSession != 11 || !bytes.Equal(recorder.keyboardReport, keyboard) {
		t.Fatalf("keyboard sink record = %d %v", recorder.keyboardSession, recorder.keyboardReport)
	}
	if recorder.pointerSession != 12 || !bytes.Equal(recorder.pointerReport, pointer) {
		t.Fatalf("pointer sink record = %d %v", recorder.pointerSession, recorder.pointerReport)
	}
	if recorder.touchSession != 13 || !bytes.Equal(recorder.touchReport, touch) {
		t.Fatalf("touch sink record = %d %v", recorder.touchSession, recorder.touchReport)
	}
	if recorder.releasedSession != 14 || recorder.releaseReason != "test" {
		t.Fatalf("release sink record = %d %q", recorder.releasedSession, recorder.releaseReason)
	}
	if recorder.resetReason != "rebuild" {
		t.Fatalf("reset sink reason = %q", recorder.resetReason)
	}
	if recorder.quiesceReason != "pointer mode change" || recorder.quiesceEpoch != 17 {
		t.Fatalf("quiesce sink record = %q %d", recorder.quiesceReason, recorder.quiesceEpoch)
	}
	if recorder.resumedEpoch != 17 || recorder.resumeReason != "pointer mode changed" {
		t.Fatalf("resume sink record = %d %q", recorder.resumedEpoch, recorder.resumeReason)
	}
}

func TestHidRoutesInternalReportsFallbackWithoutSink(t *testing.T) {
	SetInputRouterSink(nil)
	keyboardFile := &fakeHIDFile{}
	mouseFile := &fakeHIDFile{}
	h := &Hid{
		g0: hidEndpoint{file: keyboardFile},
		g1: hidEndpoint{file: mouseFile},
	}

	keyboard := []byte{0, 0, 4, 0, 0, 0, 0, 0}
	pointer := []byte{0, 1, 2, 0}
	if err := h.RouteKeyboardInput(1, keyboard); err != nil {
		t.Fatalf("keyboard fallback: %v", err)
	}
	if err := h.RoutePointerInput(1, pointer); err != nil {
		t.Fatalf("pointer fallback: %v", err)
	}
	if len(keyboardFile.writes) != 1 || !bytes.Equal(keyboardFile.writes[0], keyboard) {
		t.Fatalf("keyboard fallback writes = %v", keyboardFile.writes)
	}
	if len(mouseFile.writes) != 1 || !bytes.Equal(mouseFile.writes[0], pointer) {
		t.Fatalf("pointer fallback writes = %v", mouseFile.writes)
	}
}

func TestInternalInputSessionIDsAreJSONSafe(t *testing.T) {
	for _, session := range []uint64{
		InternalInputSessionPaste,
		InternalInputSessionAI,
		InternalInputSessionJiggler,
		InternalInputSessionKeyboardQueue,
		InternalInputSessionMouseQueue,
	} {
		if session > 1<<53-1 {
			t.Fatalf("internal input session %d exceeds JavaScript safe integer range", session)
		}
	}
}

func TestInputRoutingQuiesceRequiresInstalledSink(t *testing.T) {
	SetInputRouterSink(nil)
	h := &Hid{}

	if _, err := h.BeginInputRoutingQuiesce("test"); !errors.Is(err, ErrInputRouterUnavailable) {
		t.Fatalf("BeginInputRoutingQuiesce error = %v, want unavailable", err)
	}
	if err := h.EndInputRoutingQuiesce(1, "test"); !errors.Is(err, ErrInputRouterUnavailable) {
		t.Fatalf("EndInputRoutingQuiesce error = %v, want unavailable", err)
	}
}
