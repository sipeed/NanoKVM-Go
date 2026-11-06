package inputrouter

import (
	"errors"
	"testing"
	"time"
)

func TestRouterQuiesceReleasesAndClearsEveryRouteState(t *testing.T) {
	output := &fakeReleasingOutput{}
	router := New(output, NewTraceHub(32))
	frames := [][]byte{
		append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), []byte{2, 0, 4, 0, 0, 0, 0, 0}...),
		append(makeHeader(SourceBrowserPointer, KindRelativePointer), []byte{1, 4, 0, 3, 0, 0, 0}...),
		append(makeHeader(SourceBrowserPointer, KindAbsolutePointer), absolutePayload(1, 0x2000, 0x3000, 0)...),
		append(makeHeader(SourceBrowserPointer, KindStylus), stylusPayload(3, 0x4000, 0x5000, 0xff)...),
		append(makeHeader(SourceBrowserTouch, KindTouchscreen), []byte{0}...),
	}
	for index, frame := range frames {
		if err := router.RouteSession(SessionID(index+1), frame); err != nil {
			t.Fatalf("route state %d: %v", index, err)
		}
	}
	writesBeforeQuiesce := len(output.writes)

	epoch, err := router.BeginInputQuiesce("pointer mode change")
	if err != nil {
		t.Fatalf("BeginInputQuiesce: %v", err)
	}
	if epoch == 0 || !inputEpochQuiescing(epoch) || output.releaseCalls != 1 {
		t.Fatalf("quiesce epoch/release = %d/%d", epoch, output.releaseCalls)
	}
	status := router.Status()
	if !status.Quiescing || status.InputEpoch != epoch || status.ActiveSessions != 0 {
		t.Fatalf("status while quiescing = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Owner != 0 {
			t.Fatalf("route owner survived quiesce: %+v", route)
		}
	}

	if err := router.RouteSession(99, frames[0]); !errors.Is(err, ErrInputRouterQuiescing) {
		t.Fatalf("route while quiescing error = %v, want quiescing", err)
	}
	if len(output.writes) != writesBeforeQuiesce {
		t.Fatalf("writes while quiescing = %d, want %d", len(output.writes), writesBeforeQuiesce)
	}

	if err := router.EndInputQuiesce(epoch, "pointer mode changed"); err != nil {
		t.Fatalf("EndInputQuiesce: %v", err)
	}
	status = router.Status()
	if status.Quiescing || status.InputEpoch != epoch+1 || status.ActiveSessions != 0 {
		t.Fatalf("status after resume = %+v", status)
	}
	if err := router.RouteSession(100, frames[0]); err != nil {
		t.Fatalf("fresh epoch route: %v", err)
	}
	if len(output.writes) != writesBeforeQuiesce+1 {
		t.Fatalf("writes after fresh route = %d, want %d", len(output.writes), writesBeforeQuiesce+1)
	}

	stages := make(map[string]bool)
	for _, record := range router.Traces().Snapshot(32) {
		stages[record.Stage] = true
	}
	for _, stage := range []string{"quiesce", "quiescing", "resume"} {
		if !stages[stage] {
			t.Fatalf("missing %q trace stage: %+v", stage, stages)
		}
	}
}

func TestRouterRejectsQueuedEventFromPreviousEpochAfterResume(t *testing.T) {
	output := &fakeReleasingOutput{}
	router := New(output, NewTraceHub(16))
	frame := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), make([]byte, 8)...)

	// Hold route lookup after RouteSession captures its epoch. This models an
	// event queued before quiescing that does not reach dispatch until later.
	router.routeMutex.Lock()
	locked := true
	defer func() {
		if locked {
			router.routeMutex.Unlock()
		}
	}()
	result := make(chan error, 1)
	go func() {
		result <- router.RouteSession(77, frame)
	}()
	waitForRouterCondition(t, func() bool { return router.received.Load() == 1 })

	epoch, err := router.BeginInputQuiesce("pointer mode change")
	if err != nil {
		t.Fatalf("BeginInputQuiesce: %v", err)
	}
	if err := router.EndInputQuiesce(epoch, "pointer mode changed"); err != nil {
		t.Fatalf("EndInputQuiesce: %v", err)
	}
	router.routeMutex.Unlock()
	locked = false

	select {
	case err := <-result:
		if !errors.Is(err, ErrStaleInputRoutingEpoch) {
			t.Fatalf("queued route error = %v, want stale epoch", err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued route did not finish")
	}
	if len(output.writes) != 0 {
		t.Fatalf("queued stale event wrote reports: %v", output.writes)
	}
	status := router.Status()
	if status.ActiveSessions != 0 || status.Dispatched != 0 || status.Rejected != 1 || status.DispatchFailed != 1 {
		t.Fatalf("status after stale event = %+v", status)
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Stage != "stale-epoch" {
		t.Fatalf("stale event trace = %+v", records)
	}
}

func TestRouterRemainsQuiescedAfterReleaseFailure(t *testing.T) {
	output := &fakeReleasingOutput{}
	router := New(output, NewTraceHub(16))
	frame := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), make([]byte, 8)...)
	if err := router.RouteSession(7, frame); err != nil {
		t.Fatalf("initial route: %v", err)
	}
	releaseErr := errors.New("neutral release failed")
	output.err = releaseErr

	epoch, err := router.BeginInputQuiesce("pointer mode change")
	if !errors.Is(err, releaseErr) {
		t.Fatalf("BeginInputQuiesce error = %v, want release failure", err)
	}
	status := router.Status()
	if !status.Quiescing || status.InputEpoch != epoch || status.ActiveSessions != 0 {
		t.Fatalf("status after failed release = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Owner != 0 {
			t.Fatalf("owner survived failed release: %+v", route)
		}
	}
	if err := router.RouteSession(8, frame); !errors.Is(err, ErrInputRouterQuiescing) {
		t.Fatalf("route after failed release = %v, want quiescing", err)
	}
	if err := router.EndInputQuiesce(epoch, "rollback complete"); err != nil {
		t.Fatalf("EndInputQuiesce after release failure: %v", err)
	}
}

func TestRouterQuiesceTokenPreventsUnrelatedResume(t *testing.T) {
	router := New(&fakeReleasingOutput{}, NewTraceHub(8))
	epoch, err := router.BeginInputQuiesce("first change")
	if err != nil {
		t.Fatalf("BeginInputQuiesce: %v", err)
	}
	if _, err := router.BeginInputQuiesce("nested change"); !errors.Is(err, ErrInputRouterQuiescing) {
		t.Fatalf("nested BeginInputQuiesce error = %v, want quiescing", err)
	}
	if err := router.EndInputQuiesce(epoch+2, "wrong change"); !errors.Is(err, ErrInputRouterEpochMismatch) {
		t.Fatalf("wrong EndInputQuiesce error = %v, want epoch mismatch", err)
	}
	if !router.Status().Quiescing {
		t.Fatal("wrong quiesce token resumed routing")
	}
	if err := router.EndInputQuiesce(epoch, "first change complete"); err != nil {
		t.Fatalf("EndInputQuiesce: %v", err)
	}
	if err := router.EndInputQuiesce(epoch, "duplicate resume"); !errors.Is(err, ErrInputRouterNotQuiescing) {
		t.Fatalf("duplicate EndInputQuiesce error = %v, want not quiescing", err)
	}
}

func waitForRouterCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for router condition")
		}
		time.Sleep(time.Millisecond)
	}
}
