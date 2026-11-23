package inputrouter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"testing"
	"time"
)

type fakeOutput struct {
	route  string
	report []byte
	writes [][]byte
	err    error
}

type fakeReleasingOutput struct {
	fakeOutput
	releaseCalls int
}

type validatingAbsoluteOutput struct {
	fakeOutput
}

type transientTouchOutput struct {
	fakeOutput
	timeouts int
}

type transientRelativeOutput struct {
	fakeOutput
	timeouts int
}

type scriptedTouchOutput struct {
	fakeOutput
	failAt map[int]error
}

func (output *fakeOutput) write(route string, report []byte) error {
	output.route = route
	output.report = bytes.Clone(report)
	output.writes = append(output.writes, bytes.Clone(report))
	return output.err
}

func (output *fakeOutput) WriteKeyboardReport(report []byte) error {
	return output.write("keyboard", report)
}

func (output *fakeOutput) WriteRelativeMouseReport(report []byte) error {
	return output.write("relative", report)
}

func (output *fakeOutput) WriteAbsoluteMouseReport(report []byte) error {
	return output.write("absolute", report)
}

func (output *fakeOutput) WriteTouchScreenReport(report []byte) error {
	return output.write("touchscreen", report)
}

func (output *validatingAbsoluteOutput) WriteAbsoluteMouseReport(report []byte) error {
	if len(report) != 7 || (report[0] != absoluteReportIDMouse && report[0] != absoluteReportIDStylus) {
		return errors.New("invalid absolute pointer report id")
	}
	return output.write("absolute", report)
}

func (output *transientTouchOutput) WriteTouchScreenReport(report []byte) error {
	if output.timeouts > 0 {
		output.timeouts--
		return os.ErrDeadlineExceeded
	}
	return output.write("touchscreen", report)
}

func (output *transientRelativeOutput) WriteRelativeMouseReport(report []byte) error {
	if output.timeouts > 0 {
		output.timeouts--
		return os.ErrDeadlineExceeded
	}
	return output.write("relative", report)
}

func (output *scriptedTouchOutput) WriteTouchScreenReport(report []byte) error {
	output.route = "touchscreen"
	output.report = bytes.Clone(report)
	output.writes = append(output.writes, bytes.Clone(report))
	return output.failAt[len(output.writes)]
}

func (output *fakeReleasingOutput) ReleaseAllInputs() error {
	output.releaseCalls++
	return output.err
}

func TestReliableVirtualAbsolutePointerReportsFailedMovement(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, nil)
	const session = 1
	start := []byte{absoluteReportIDMouse, 0, 0, 0, 0, 0, 0}
	end := []byte{absoluteReportIDMouse, 0, 0xff, 0x7f, 0xff, 0x7f, 0}
	if err := router.RouteReliableVirtualPointerSession(session, start); err != nil {
		t.Fatal(err)
	}

	output.err = os.ErrDeadlineExceeded
	if err := router.RouteVirtualPointerSession(session, end); err != nil {
		t.Fatalf("ordinary absolute movement should remain droppable: %v", err)
	}
	if err := router.RouteReliableVirtualPointerSession(session, end); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("reliable absolute movement error = %v, want deadline exceeded", err)
	}
	if got, want := len(output.writes), 2+criticalWriteAttempts; got != want {
		t.Fatalf("write attempts = %d, want %d", got, want)
	}
	for _, report := range output.writes[2:] {
		if !bytes.Equal(report, end) {
			t.Fatalf("retried report = %v, want %v", report, end)
		}
	}

	output.err = nil
	if err := router.ReleaseInputSession(session, "drag failed"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.report, start) {
		t.Fatalf("release report = %v, want last successful position %v", output.report, start)
	}
}

func TestDecodeRejectsInvalidFrames(t *testing.T) {
	tests := [][]byte{
		{},
		make([]byte, HeaderSize-1),
		append([]byte{2, byte(SourceBrowserPointer), byte(KindRelativePointer), 0}, make([]byte, 15)...),
		append(makeHeader(Source(99), KindRelativePointer), make([]byte, 7)...),
		append(makeHeader(SourceBrowserPointer, KindRelativePointer), make([]byte, 6)...),
	}
	for _, frame := range tests {
		if _, err := Decode(frame); err == nil {
			t.Fatalf("Decode(%v) unexpectedly succeeded", frame)
		}
	}
}

func TestDecodeAssociatesBrowserCaptureContext(t *testing.T) {
	frame := makeHeader(SourceBrowserPointer, KindAbsolutePointer)
	frame[3] = FlagCaptureContext
	payload := absolutePayload(0, 0x4000, 0x8000, 0)
	context := make([]byte, CaptureContextSize)
	binary.LittleEndian.PutUint32(context[0:4], 77)
	binary.LittleEndian.PutUint32(context[4:8], 12345)

	event, err := Decode(append(append(frame, payload...), context...))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if event.CaptureSeq != 77 || event.ClientDelayUS != 12345 {
		t.Fatalf("capture context = sequence %d delay %d", event.CaptureSeq, event.ClientDelayUS)
	}
	if event.Absolute == nil || event.Absolute.X != 0x4000 || event.Absolute.Y != 0x8000 {
		t.Fatalf("absolute event = %+v", event.Absolute)
	}
}

func TestDecodeRejectsMalformedCaptureContext(t *testing.T) {
	frame := makeHeader(SourceBrowserPointer, KindAbsolutePointer)
	frame[3] = FlagCaptureContext
	if _, err := Decode(append(frame, absolutePayload(0, 0, 0, 0)...)); err == nil {
		t.Fatal("missing capture context unexpectedly succeeded")
	}

	context := make([]byte, CaptureContextSize)
	if _, err := Decode(append(append(frame, absolutePayload(0, 0, 0, 0)...), context...)); err == nil {
		t.Fatal("zero capture sequence unexpectedly succeeded")
	}
}

func TestRouterEncodesRelativePointer(t *testing.T) {
	frame := makeHeader(SourceBrowserPointer, KindRelativePointer)
	payload := make([]byte, 7)
	payload[0] = 0x03
	binary.LittleEndian.PutUint16(payload[1:3], uint16(400))
	putInt16(payload[3:5], -300)
	putInt16(payload[5:7], -2)
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))

	if err := router.Route(append(frame, payload...)); err != nil {
		t.Fatalf("Route: %v", err)
	}
	if output.route != "relative" {
		t.Fatalf("route = %q, want relative", output.route)
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Semantic == nil || records[0].Semantic.Buttons != 3 || records[0].Semantic.DeltaX != 400 || records[0].Semantic.DeltaY != -300 || records[0].Semantic.Wheel != -2 {
		t.Fatalf("relative semantic trace = %+v", records)
	}
	want := []byte{0x03, 0x7f, 0x81, 0xfe}
	if !bytes.Equal(output.report, want) {
		t.Fatalf("report = %v, want %v", output.report, want)
	}
}

func TestRouterRoutesInternalVirtualInputs(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(16))
	router.now = func() time.Time { return time.UnixMilli(4321) }

	keyboard := []byte{0x02, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00}
	if err := router.RouteVirtualKeyboardSession(9001, keyboard); err != nil {
		t.Fatalf("RouteVirtualKeyboardSession: %v", err)
	}
	if output.route != "keyboard" || !bytes.Equal(output.report, keyboard) {
		t.Fatalf("keyboard output = %s %v", output.route, output.report)
	}
	keyboardRecords := router.Traces().Snapshot(2)
	if len(keyboardRecords) != 2 || keyboardRecords[0].Source != "virtual-keyboard" || keyboardRecords[0].Stage != "internal-adapted" || keyboardRecords[1].Stage != "dispatched" {
		t.Fatalf("keyboard traces = %+v", keyboardRecords)
	}

	pointer := []byte{0x01, 0x00, 0x00, 0x3f, 0x00, 0x3f, 0x00}
	if err := router.RouteVirtualPointerSession(9002, pointer); err != nil {
		t.Fatalf("RouteVirtualPointerSession: %v", err)
	}
	if output.route != "absolute" || len(output.report) != 7 || output.report[0] != 0x01 {
		t.Fatalf("pointer output = %s %v", output.route, output.report)
	}

	touch := make([]byte, touchscreenReportSize)
	touch[0] = touchscreenReportID
	touch[1] = 0x03
	touch[2] = 0x01
	touch[len(touch)-1] = 1
	if err := router.RouteVirtualTouchSession(9003, touch); err != nil {
		t.Fatalf("RouteVirtualTouchSession: %v", err)
	}
	if output.route != "touchscreen" || output.report[0] != touchscreenReportID || output.report[len(output.report)-1] != 1 {
		t.Fatalf("touch output = %s %v", output.route, output.report)
	}
	touchRecords := router.Traces().Snapshot(2)
	if len(touchRecords) != 2 || touchRecords[0].Source != "virtual-touchscreen" || touchRecords[1].Route != "hidg2/touchscreen" {
		t.Fatalf("touch traces = %+v", touchRecords)
	}
}

func TestRouterRejectsInvalidInternalTouchReport(t *testing.T) {
	router := New(&fakeOutput{}, NewTraceHub(4))
	router.now = func() time.Time { return time.UnixMilli(4321) }

	if err := router.RouteVirtualTouchSession(9004, nil); err == nil {
		t.Fatal("empty internal touchscreen report unexpectedly routed")
	}
	status := router.Status()
	if status.Received != 1 || status.Rejected != 1 || status.DecodeRejected != 1 || status.Dispatched != 0 {
		t.Fatalf("runtime status = %+v", status)
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Stage != "internal-adapted-error" || records[0].Kind != "touchscreen" {
		t.Fatalf("invalid internal touch trace = %+v", records)
	}
}

func TestRouterEncodesAbsoluteEndpoints(t *testing.T) {
	for _, test := range []struct {
		kind    Kind
		payload []byte
		wantID  byte
	}{
		{kind: KindAbsolutePointer, payload: absolutePayload(0x01, 0xffff, 0x8000, -1), wantID: 0x01},
		{kind: KindStylus, payload: stylusPayload(0x03, 0xffff, 0x8000, 0xee), wantID: 0x02},
	} {
		output := &fakeOutput{}
		router := New(output, NewTraceHub(8))
		if err := router.Route(append(makeHeader(SourceBrowserPointer, test.kind), test.payload...)); err != nil {
			t.Fatalf("Route kind %d: %v", test.kind, err)
		}
		if output.route != "absolute" || len(output.report) != 7 || output.report[0] != test.wantID {
			t.Fatalf("kind %d output = %s %v", test.kind, output.route, output.report)
		}
		if got := binary.LittleEndian.Uint16(output.report[2:4]); got != 0x7fff {
			t.Fatalf("kind %d x = 0x%04x, want 0x7fff", test.kind, got)
		}
		if got := binary.LittleEndian.Uint16(output.report[4:6]); got != 0x4000 {
			t.Fatalf("kind %d y = 0x%04x, want 0x4000", test.kind, got)
		}
	}
}

func TestRouterEncodesMultitouchAndTrace(t *testing.T) {
	payload := []byte{2}
	payload = append(payload, touchContact(4, 0x03, 0, 0xffff, 0xff)...)
	payload = append(payload, touchContact(9, 0x03, 0x8000, 0x4000, 0x80)...)
	output := &fakeOutput{}
	traces := NewTraceHub(8)
	router := New(output, traces)

	if err := router.Route(append(makeHeader(SourceBrowserTouch, KindTouchscreen), payload...)); err != nil {
		t.Fatalf("Route: %v", err)
	}
	if output.route != "touchscreen" || len(output.report) != touchscreenReportSize {
		t.Fatalf("output = %s %v", output.route, output.report)
	}
	if output.report[0] != touchscreenReportID || output.report[2] != 4 || output.report[8] != 9 {
		t.Fatalf("touch report ids = %v", output.report)
	}
	if output.report[len(output.report)-1] != 2 {
		t.Fatalf("touch count = %d, want 2", output.report[len(output.report)-1])
	}
	records := traces.Snapshot(1)
	if len(records) != 1 || records[0].Route != "hidg2/touchscreen" || len(records[0].Points) != 2 {
		t.Fatalf("trace = %+v", records)
	}
	if !records[0].Points[0].Touching || !records[0].Points[0].InRange || records[0].Points[0].Pressure != 0xff {
		t.Fatalf("touch trace point = %+v", records[0].Points[0])
	}
}

func TestRouterReportsRuntimeStatus(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))

	if err := router.Route(append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), make([]byte, 8)...)); err != nil {
		t.Fatalf("Route keyboard: %v", err)
	}
	if err := router.Route([]byte{ProtocolVersion}); err == nil {
		t.Fatal("invalid route unexpectedly succeeded")
	}

	status := router.Status()
	if status.Received != 2 || status.Dispatched != 1 || status.Rejected != 1 {
		t.Fatalf("status counters = %+v", status)
	}
	if status.DecodeRejected != 1 || status.DispatchFailed != 0 {
		t.Fatalf("status failure counters = %+v", status)
	}
	if status.LastError == "" || status.LastActivityMS == 0 || len(status.Routes) != 5 {
		t.Fatalf("status metadata = %+v", status)
	}
	if status.ActiveSessions != 1 || len(status.Sessions) != 1 || status.Sessions[0].Events != 1 {
		t.Fatalf("status sessions = %+v", status.Sessions)
	}
	if len(status.Sources) != 1 || status.Sources[0] != (SourceStatus{Source: "browser-keyboard", Events: 1}) {
		t.Fatalf("status sources = %+v", status.Sources)
	}
	for index := 1; index < len(status.Routes); index++ {
		if status.Routes[index-1].Route > status.Routes[index].Route {
			t.Fatalf("routes are not sorted: %+v", status.Routes)
		}
	}
}

func TestRouterManagementSeparatesDispatchFailures(t *testing.T) {
	output := &fakeOutput{err: errors.New("endpoint timeout")}
	router := New(output, NewTraceHub(8))
	frame := append(makeHeader(SourceBrowserTouch, KindTouchscreen), []byte{0}...)

	if err := router.Route(frame); err == nil {
		t.Fatal("failed HID dispatch unexpectedly succeeded")
	}
	status := router.Status()
	if status.Received != 1 || status.Rejected != 1 || status.DecodeRejected != 0 || status.DispatchFailed != 1 {
		t.Fatalf("status failure counters = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Route == "hidg2/touchscreen" {
			if route.Dispatched != 0 || route.Failed != 1 {
				t.Fatalf("touchscreen route status = %+v", route)
			}
			return
		}
	}
	t.Fatal("touchscreen route status not found")
}

func TestRouterReleaseUsesValidAbsoluteReportIDs(t *testing.T) {
	output := &validatingAbsoluteOutput{}
	router := New(output, NewTraceHub(8))

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(1, 0x8000, 0x8000, 0)...,
	)); err != nil {
		t.Fatalf("route absolute pointer: %v", err)
	}
	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindStylus),
		stylusPayload(0x03, 0x8000, 0x8000, 0)...,
	)); err != nil {
		t.Fatalf("route stylus: %v", err)
	}
	if err := router.ReleaseSession(7, "test disconnect"); err != nil {
		t.Fatalf("release absolute routes: %v", err)
	}

	status := router.Status()
	if status.ActiveSessions != 0 || status.DispatchFailed != 0 || status.Rejected != 0 {
		t.Fatalf("status after release = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Owner != 0 || route.Failed != 0 {
			t.Fatalf("route after release = %+v", route)
		}
	}
}

func TestRouterReleaseRetriesTransientHIDTimeouts(t *testing.T) {
	output := &transientTouchOutput{}
	router := New(output, NewTraceHub(8))
	frame := append(makeHeader(SourceBrowserTouch, KindTouchscreen), []byte{0}...)

	if err := router.RouteSession(9, frame); err != nil {
		t.Fatalf("route touchscreen: %v", err)
	}
	output.timeouts = 2
	if err := router.ReleaseSession(9, "test disconnect"); err != nil {
		t.Fatalf("release touchscreen after transient timeouts: %v", err)
	}

	status := router.Status()
	if status.DispatchFailed != 0 || status.Rejected != 0 || status.ActiveSessions != 0 {
		t.Fatalf("status after retried release = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Route == "hidg2/touchscreen" {
			if route.WriteRetries != 2 || route.Failed != 0 || route.Owner != 0 {
				t.Fatalf("touchscreen route after retried release = %+v", route)
			}
			return
		}
	}
	t.Fatal("touchscreen route status not found")
}

func TestRouterReleaseFlushesEveryTouchscreenContactID(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))
	payload := append([]byte{1}, touchContact(7, 3, 0x4000, 0x8000, 0xff)...)

	if err := router.RouteSession(9, append(makeHeader(SourceBrowserTouch, KindTouchscreen), payload...)); err != nil {
		t.Fatalf("route active touchscreen: %v", err)
	}
	if err := router.ReleaseSession(9, "test abrupt disconnect"); err != nil {
		t.Fatalf("release active touchscreen: %v", err)
	}

	report := output.report
	if output.route != "touchscreen" || len(report) != touchscreenReportSize || report[len(report)-1] != MaxContacts {
		t.Fatalf("touchscreen flush report = %s %v", output.route, report)
	}
	for index := 0; index < MaxContacts; index++ {
		offset := 1 + index*6
		if report[offset] != 0 || report[offset+1] != byte(index+1) {
			t.Fatalf("touchscreen flush slot %d = %v", index, report[offset:offset+6])
		}
	}
}

func TestRouterClearsTouchscreenSlotsAcrossSessionOwners(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(16))
	firstPayload := append([]byte{1}, touchContact(1, 3, 0x2000, 0x3000, 0xff)...)
	secondPayload := append([]byte{1}, touchContact(2, 3, 0x8000, 0x9000, 0xff)...)

	if err := router.RouteSession(11, append(makeHeader(SourceBrowserTouch, KindTouchscreen), firstPayload...)); err != nil {
		t.Fatalf("route first touchscreen owner: %v", err)
	}
	if err := router.RouteSession(22, append(makeHeader(SourceBrowserTouch, KindTouchscreen), secondPayload...)); err != nil {
		t.Fatalf("route second touchscreen owner: %v", err)
	}
	if len(output.writes) != 3 || output.writes[1][touchscreenReportSize-1] != MaxContacts || output.writes[2][2] != 2 {
		t.Fatalf("owner switch writes = %v", output.writes)
	}
	if err := router.ReleaseSession(22, "second owner disconnected"); err != nil {
		t.Fatalf("restore first touchscreen owner: %v", err)
	}
	if len(output.writes) != 5 || output.writes[3][touchscreenReportSize-1] != MaxContacts || output.writes[4][2] != 1 {
		t.Fatalf("owner restore writes = %v", output.writes)
	}
	status := router.Status()
	for _, route := range status.Routes {
		if route.Route == "hidg2/touchscreen" && route.Owner != 11 {
			t.Fatalf("touchscreen owner = %d, want 11", route.Owner)
		}
	}
	records := router.Traces().Snapshot(16)
	var switchClear, restoreClear bool
	for _, record := range records {
		switchClear = switchClear || record.Stage == "owner-switch-clear"
		restoreClear = restoreClear || record.Stage == "restore-clear"
	}
	if !switchClear || !restoreClear {
		t.Fatalf("owner-change traces = %+v", records)
	}
}

func TestRouterDropsOwnerAfterFailedTouchscreenOwnerSwitch(t *testing.T) {
	output := &scriptedTouchOutput{failAt: map[int]error{3: errors.New("touchscreen write failed")}}
	router := New(output, NewTraceHub(16))
	firstPayload := append([]byte{1}, touchContact(1, 3, 0x2000, 0x3000, 0xff)...)
	secondPayload := append([]byte{1}, touchContact(2, 3, 0x8000, 0x9000, 0xff)...)

	if err := router.RouteSession(11, append(makeHeader(SourceBrowserTouch, KindTouchscreen), firstPayload...)); err != nil {
		t.Fatalf("route first touchscreen owner: %v", err)
	}
	if err := router.RouteSession(22, append(makeHeader(SourceBrowserTouch, KindTouchscreen), secondPayload...)); err == nil {
		t.Fatal("failed touchscreen owner switch unexpectedly succeeded")
	}
	for _, route := range router.Status().Routes {
		if route.Route == "hidg2/touchscreen" && route.Owner != 0 {
			t.Fatalf("touchscreen owner after failed switch = %d, want none", route.Owner)
		}
	}
}

func TestRouterDropsOwnerAfterFailedTouchscreenRestore(t *testing.T) {
	output := &scriptedTouchOutput{failAt: map[int]error{5: errors.New("touchscreen restore failed")}}
	router := New(output, NewTraceHub(16))
	firstPayload := append([]byte{1}, touchContact(1, 3, 0x2000, 0x3000, 0xff)...)
	secondPayload := append([]byte{1}, touchContact(2, 3, 0x8000, 0x9000, 0xff)...)

	if err := router.RouteSession(11, append(makeHeader(SourceBrowserTouch, KindTouchscreen), firstPayload...)); err != nil {
		t.Fatalf("route first touchscreen owner: %v", err)
	}
	if err := router.RouteSession(22, append(makeHeader(SourceBrowserTouch, KindTouchscreen), secondPayload...)); err != nil {
		t.Fatalf("route second touchscreen owner: %v", err)
	}
	if err := router.ReleaseSession(22, "second owner disconnected"); err == nil {
		t.Fatal("failed touchscreen restore unexpectedly succeeded")
	}
	for _, route := range router.Status().Routes {
		if route.Route == "hidg2/touchscreen" && route.Owner != 0 {
			t.Fatalf("touchscreen owner after failed restore = %d, want none", route.Owner)
		}
	}
}

func TestRouterRetriesCriticalTouchscreenTransitions(t *testing.T) {
	output := &transientTouchOutput{}
	router := New(output, NewTraceHub(8))
	empty := append(makeHeader(SourceBrowserTouch, KindTouchscreen), []byte{0}...)
	contact := []byte{1}
	contact = append(contact, touchContact(1, 3, 0x4000, 0x8000, 0xff)...)

	if err := router.RouteSession(9, empty); err != nil {
		t.Fatalf("route empty touchscreen: %v", err)
	}
	output.timeouts = 2
	if err := router.RouteSession(9, append(makeHeader(SourceBrowserTouch, KindTouchscreen), contact...)); err != nil {
		t.Fatalf("route touchscreen contact after transient timeouts: %v", err)
	}

	status := router.Status()
	if status.Dispatched != 2 || status.DispatchFailed != 0 || status.BackpressureDropped != 0 {
		t.Fatalf("status after critical retry = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Route == "hidg2/touchscreen" {
			if route.WriteRetries != 2 || route.Failed != 0 || route.Dropped != 0 {
				t.Fatalf("touchscreen route after critical retry = %+v", route)
			}
			return
		}
	}
	t.Fatal("touchscreen route status not found")
}

func TestRouterDropsReplaceableTouchscreenMotionUnderBackpressure(t *testing.T) {
	output := &transientTouchOutput{}
	router := New(output, NewTraceHub(8))
	first := []byte{1}
	first = append(first, touchContact(1, 3, 0x1000, 0x2000, 0xff)...)
	second := []byte{1}
	second = append(second, touchContact(1, 3, 0x3000, 0x4000, 0xff)...)

	if err := router.RouteSession(9, append(makeHeader(SourceBrowserTouch, KindTouchscreen), first...)); err != nil {
		t.Fatalf("route first touchscreen position: %v", err)
	}
	output.timeouts = 1
	if err := router.RouteSession(9, append(makeHeader(SourceBrowserTouch, KindTouchscreen), second...)); err != nil {
		t.Fatalf("replaceable touchscreen motion returned an error: %v", err)
	}

	status := router.Status()
	if status.Received != 2 || status.Dispatched != 1 || status.Rejected != 0 || status.DispatchFailed != 0 || status.BackpressureDropped != 1 {
		t.Fatalf("status after dropped motion = %+v", status)
	}
	for _, route := range status.Routes {
		if route.Route == "hidg2/touchscreen" && (route.Dropped != 1 || route.Failed != 0) {
			t.Fatalf("touchscreen route after dropped motion = %+v", route)
		}
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Stage != "backpressure-drop" || records[0].Error == "" {
		t.Fatalf("backpressure trace = %+v", records)
	}
}

func TestRouterRetriesRelativeButtonsWithoutRepeatingMotion(t *testing.T) {
	output := &transientRelativeOutput{}
	router := New(output, NewTraceHub(8))
	neutral := append(makeHeader(SourceBrowserPointer, KindRelativePointer), make([]byte, 7)...)
	payload := make([]byte, 7)
	payload[0] = 1
	putInt16(payload[1:3], 40)
	putInt16(payload[3:5], -30)
	putInt16(payload[5:7], 2)

	if err := router.RouteSession(7, neutral); err != nil {
		t.Fatalf("route neutral relative pointer: %v", err)
	}
	output.timeouts = 1
	if err := router.RouteSession(7, append(makeHeader(SourceBrowserPointer, KindRelativePointer), payload...)); err != nil {
		t.Fatalf("route relative button after transient timeout: %v", err)
	}
	if !bytes.Equal(output.report, []byte{1, 0, 0, 0}) {
		t.Fatalf("recovered relative report = %v, want buttons without motion", output.report)
	}
	status := router.Status()
	if status.Dispatched != 2 || status.BackpressureDropped != 0 || status.DispatchFailed != 0 {
		t.Fatalf("status after relative recovery = %+v", status)
	}
}

func TestRouterManagementTracksSessionSourcesAndRoutes(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))
	keyboard := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), make([]byte, 8)...)
	pointer := append(makeHeader(SourceVirtualPointer, KindRelativePointer), make([]byte, 7)...)

	if err := router.RouteSession(7, keyboard); err != nil {
		t.Fatalf("RouteSession keyboard: %v", err)
	}
	if err := router.RouteSession(9, pointer); err != nil {
		t.Fatalf("RouteSession pointer: %v", err)
	}
	status := router.Status()
	if status.ActiveSessions != 2 || len(status.Sessions) != 2 || len(status.Sources) != 2 {
		t.Fatalf("management status = %+v", status)
	}
	if status.Sessions[0].ID != 7 || status.Sessions[0].Routes[0] != "hidg0/keyboard" {
		t.Fatalf("keyboard session = %+v", status.Sessions[0])
	}
	if status.Sessions[1].ID != 9 || status.Sessions[1].Routes[0] != "hidg1/relative-pointer" {
		t.Fatalf("pointer session = %+v", status.Sessions[1])
	}

	if err := router.ReleaseSession(7, "keyboard disconnected"); err != nil {
		t.Fatalf("ReleaseSession: %v", err)
	}
	status = router.Status()
	if status.ActiveSessions != 1 || len(status.Sessions) != 1 || status.Sessions[0].ID != 9 {
		t.Fatalf("management status after release = %+v", status)
	}
}

func TestRouterReleaseSessionDoesNotReplayTransientPointerMotion(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))
	firstPayload := make([]byte, 7)
	firstPayload[0] = 1
	putInt16(firstPayload[1:3], 40)
	putInt16(firstPayload[3:5], -30)
	putInt16(firstPayload[5:7], 2)
	secondPayload := make([]byte, 7)

	if err := router.RouteSession(11, append(makeHeader(SourceBrowserPointer, KindRelativePointer), firstPayload...)); err != nil {
		t.Fatalf("RouteSession first: %v", err)
	}
	if err := router.RouteSession(22, append(makeHeader(SourceBrowserPointer, KindRelativePointer), secondPayload...)); err != nil {
		t.Fatalf("RouteSession second: %v", err)
	}
	if err := router.ReleaseSession(22, "second disconnected"); err != nil {
		t.Fatalf("ReleaseSession second: %v", err)
	}
	if !bytes.Equal(output.report, []byte{1, 0, 0, 0}) {
		t.Fatalf("restored relative report = %v, want buttons without motion", output.report)
	}
}

// An absolute pointer has no position-free report form: releasing the session
// that moved it must keep its position, otherwise a one-shot move is silently
// undone by the other session's stale coordinates.
func TestRouterReleaseSessionKeepsAbsolutePointerPosition(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))
	browser := append(makeHeader(SourceBrowserPointer, KindAbsolutePointer), absolutePayload(0, 0x1000, 0x2000, 0)...)
	oneShot := append(makeHeader(SourceVirtualPointer, KindAbsolutePointer), absolutePayload(0, 0x3000, 0x4000, 0)...)

	if err := router.RouteSession(11, browser); err != nil {
		t.Fatalf("RouteSession browser: %v", err)
	}
	if err := router.RouteSession(22, oneShot); err != nil {
		t.Fatalf("RouteSession one-shot: %v", err)
	}
	if err := router.ReleaseSession(22, "AI action complete"); err != nil {
		t.Fatalf("ReleaseSession: %v", err)
	}

	// The released session's coordinates survive; only its buttons are cleared.
	// 0x3000/0x4000 in the v1 payload become 0x1800/0x2000 in the HID report.
	want := []byte{absoluteReportIDMouse, 0, 0x00, 0x18, 0x00, 0x20, 0}
	if !bytes.Equal(output.report, want) {
		t.Fatalf("release report = %v, want the released position %v", output.report, want)
	}
	if owner := routeOwner(router, "hidg2/absolute-pointer"); owner != 11 {
		t.Fatalf("owner after release = %d, want the browser session 11", owner)
	}

	// The browser session still owns the route, so its own release keeps its
	// position as well and only clears the buttons.
	if err := router.ReleaseSession(11, "browser disconnected"); err != nil {
		t.Fatalf("ReleaseSession browser: %v", err)
	}
	if !bytes.Equal(output.report, []byte{absoluteReportIDMouse, 0, 0x00, 0x08, 0x00, 0x10, 0}) {
		t.Fatalf("browser release report = %v", output.report)
	}
}

func routeOwner(router *Router, name string) uint64 {
	for _, route := range router.Status().Routes {
		if route.Route == name {
			return route.Owner
		}
	}
	return 0
}

func TestRouterReleaseSessionRestoresPreviousSession(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))
	firstReport := []byte{1, 0, 4, 0, 0, 0, 0, 0}
	secondReport := []byte{2, 0, 5, 0, 0, 0, 0, 0}
	first := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), firstReport...)
	second := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), secondReport...)

	if err := router.RouteSession(11, first); err != nil {
		t.Fatalf("RouteSession first: %v", err)
	}
	if err := router.RouteSession(22, second); err != nil {
		t.Fatalf("RouteSession second: %v", err)
	}
	if err := router.ReleaseSession(22, "second disconnected"); err != nil {
		t.Fatalf("ReleaseSession second: %v", err)
	}
	if !bytes.Equal(output.report, firstReport) {
		t.Fatalf("restored report = %v, want previous session report", output.report)
	}
	if router.Status().ActiveSessions != 1 || router.Status().Routes[0].Owner != 11 {
		t.Fatalf("status after restore = %+v", router.Status())
	}

	if err := router.ReleaseSession(11, "first disconnected"); err != nil {
		t.Fatalf("ReleaseSession first: %v", err)
	}
	if !bytes.Equal(output.report, make([]byte, 8)) {
		t.Fatalf("neutral report = %v, want zero report", output.report)
	}
}

func TestRouterReleaseSessionNeutralizesAtLastAbsolutePosition(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(8))

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(1, 0xffff, 0x0000, 0)...,
	)); err != nil {
		t.Fatalf("route absolute pointer: %v", err)
	}
	if err := router.ReleaseSession(7, "test disconnect"); err != nil {
		t.Fatalf("release absolute session: %v", err)
	}

	// toHIDAxis maps the wire coordinates 0xffff->0x7fff and 0x0000->0x0000.
	want := []byte{absoluteReportIDMouse, 0, 0xff, 0x7f, 0x00, 0x00, 0}
	if !bytes.Equal(output.report, want) {
		t.Fatalf("neutral absolute report = %v, want last position with buttons up %v", output.report, want)
	}
	if bytes.Equal(output.report[2:6], []byte{0x00, 0x40, 0x00, 0x40}) {
		t.Fatalf("neutral absolute report fell back to descriptor center: %v", output.report)
	}
}

func TestNeutralAbsolutePointerRequiresLastReport(t *testing.T) {
	for _, last := range [][]byte{nil, {}, {1, 2, 3}} {
		if report := neutralAbsoluteMouse(last); report != nil {
			t.Fatalf("neutralAbsoluteMouse(%v) = %v, want nil", last, report)
		}
		if report := neutralStylus(last); report != nil {
			t.Fatalf("neutralStylus(%v) = %v, want nil", last, report)
		}
		if report := restoreAbsolutePointer(last); report != nil {
			t.Fatalf("restoreAbsolutePointer(%v) = %v, want nil", last, report)
		}
	}

	last := []byte{absoluteReportIDStylus, 1, 0x34, 0x12, 0x78, 0x56, 0xff}
	if report := neutralAbsoluteMouse(last); !bytes.Equal(report, []byte{absoluteReportIDMouse, 0, 0x34, 0x12, 0x78, 0x56, 0}) {
		t.Fatalf("neutralAbsoluteMouse(last) = %v, want last position with buttons up", report)
	}
	wantRestore := []byte{absoluteReportIDStylus, 1, 0x34, 0x12, 0x78, 0x56, 0}
	if report := restoreAbsolutePointer(last); !bytes.Equal(report, wantRestore) {
		t.Fatalf("restoreAbsolutePointer(last) = %v, want %v", report, wantRestore)
	}
}

func TestDecodeAcceptsAnyValidSourceKind(t *testing.T) {
	frame := append(makeHeader(SourceBrowserKeyboard, KindTouchscreen), []byte{0}...)
	if _, err := Decode(frame); err != nil {
		t.Fatalf("valid source and event kind rejected: %v", err)
	}
}

func TestRouterReleasesAllInputs(t *testing.T) {
	output := &fakeReleasingOutput{}
	traces := NewTraceHub(8)
	router := New(output, traces)

	if err := router.Release("test disconnect"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if output.releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", output.releaseCalls)
	}
	records := traces.Snapshot(1)
	if len(records) != 1 || records[0].Stage != "release" || records[0].Meaning != "test disconnect" {
		t.Fatalf("release trace = %+v", records)
	}
}

func TestRouterResetClearsSessionsAfterReleaseFailure(t *testing.T) {
	output := &fakeReleasingOutput{}
	router := New(output, NewTraceHub(8))
	frame := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), []byte{0x02, 0, 4, 0, 0, 0, 0, 0}...)
	if err := router.RouteSession(77, frame); err != nil {
		t.Fatalf("RouteSession: %v", err)
	}
	output.err = errors.New("release failed")

	if err := router.ResetInputSessions("USB gadget rebuild"); err == nil {
		t.Fatal("ResetInputSessions unexpectedly succeeded")
	}
	status := router.Status()
	if status.ActiveSessions != 0 || len(status.Sessions) != 0 {
		t.Fatalf("sessions after reset = %+v", status.Sessions)
	}
	for _, route := range status.Routes {
		if route.Owner != 0 {
			t.Fatalf("route owner after reset = %+v", route)
		}
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Stage != "reset" || records[0].Error == "" {
		t.Fatalf("reset trace = %+v", records)
	}
}

func TestDecodeRejectsDuplicateTouchContactIDs(t *testing.T) {
	payload := []byte{2}
	payload = append(payload, touchContact(1, 3, 0, 0, 0)...)
	payload = append(payload, touchContact(1, 3, 1, 1, 0)...)
	if _, err := Decode(append(makeHeader(SourceBrowserTouch, KindTouchscreen), payload...)); err == nil {
		t.Fatal("duplicate contact IDs unexpectedly accepted")
	}
}

func TestRouterPointerReleaseNeutralizesOwnedRoutes(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(16)
	router := New(output, traces)

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(1, 0xffff, 0x0000, 0)...,
	)); err != nil {
		t.Fatalf("route absolute pointer: %v", err)
	}
	relative := make([]byte, 7)
	relative[0] = 1
	putInt16(relative[1:3], 40)
	putInt16(relative[3:5], -30)
	putInt16(relative[5:7], 2)
	if err := router.RouteSession(7, append(makeHeader(SourceBrowserPointer, KindRelativePointer), relative...)); err != nil {
		t.Fatalf("route relative pointer: %v", err)
	}
	writesBefore := len(output.writes)

	if err := router.RouteSession(7, pointerReleaseFrame(ReleaseRouteRelativePointer|ReleaseRouteAbsolutePointer)); err != nil {
		t.Fatalf("pointer release: %v", err)
	}
	if len(output.writes) != writesBefore+2 {
		t.Fatalf("pointer release writes = %d, want %d", len(output.writes), writesBefore+2)
	}
	if !bytes.Equal(output.writes[writesBefore], []byte{0, 0, 0, 0}) {
		t.Fatalf("neutral relative report = %v, want zero report", output.writes[writesBefore])
	}
	wantAbsolute := []byte{absoluteReportIDMouse, 0, 0xff, 0x7f, 0x00, 0x00, 0}
	if !bytes.Equal(output.writes[writesBefore+1], wantAbsolute) {
		t.Fatalf("neutral absolute report = %v, want %v", output.writes[writesBefore+1], wantAbsolute)
	}
	if len(router.states[KindAbsolutePointer]) != 0 || len(router.states[KindRelativePointer]) != 0 {
		t.Fatalf("states after pointer release = %+v", router.states)
	}
	if _, ok := router.owners[KindAbsolutePointer]; ok {
		t.Fatal("absolute pointer owner survived pointer release")
	}
	if _, ok := router.owners[KindRelativePointer]; ok {
		t.Fatal("relative pointer owner survived pointer release")
	}
}

func TestRouterPointerReleaseLeavesOtherOwnerUntouched(t *testing.T) {
	output := &fakeOutput{}
	router := New(output, NewTraceHub(16))

	if err := router.RouteSession(22, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(1, 0x8000, 0x4000, 0)...,
	)); err != nil {
		t.Fatalf("route first absolute pointer: %v", err)
	}
	if err := router.RouteSession(11, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(0, 0x2000, 0x1000, 0)...,
	)); err != nil {
		t.Fatalf("route second absolute pointer: %v", err)
	}
	if owner := router.owners[KindAbsolutePointer]; owner != 11 {
		t.Fatalf("absolute pointer owner = %d, want 11", owner)
	}
	ownerReport := bytes.Clone(router.states[KindAbsolutePointer][11].report)
	writesBefore := len(output.writes)

	if err := router.RouteSession(22, pointerReleaseFrame(ReleaseRouteAbsolutePointer)); err != nil {
		t.Fatalf("non-owner pointer release: %v", err)
	}
	if len(output.writes) != writesBefore {
		t.Fatalf("non-owner pointer release wrote %d reports", len(output.writes)-writesBefore)
	}
	if len(router.states[KindAbsolutePointer]) != 1 {
		t.Fatalf("non-owner pointer release states = %+v", router.states[KindAbsolutePointer])
	}
	if owner := router.owners[KindAbsolutePointer]; owner != 11 {
		t.Fatalf("absolute pointer owner = %d, want 11", owner)
	}
	if !bytes.Equal(router.states[KindAbsolutePointer][11].report, ownerReport) {
		t.Fatal("non-owner pointer release modified the owner report")
	}
}

func TestRouterPointerReleaseRestoresPreviousOwner(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(32)
	router := New(output, traces)
	firstPayload := append([]byte{1}, touchContact(1, 3, 0x2000, 0x3000, 0xff)...)
	secondPayload := append([]byte{1}, touchContact(2, 3, 0x8000, 0x9000, 0xff)...)

	if err := router.RouteSession(11, append(makeHeader(SourceBrowserTouch, KindTouchscreen), firstPayload...)); err != nil {
		t.Fatalf("route first touchscreen owner: %v", err)
	}
	if err := router.RouteSession(22, append(makeHeader(SourceBrowserTouch, KindTouchscreen), secondPayload...)); err != nil {
		t.Fatalf("route second touchscreen owner: %v", err)
	}
	firstReport := bytes.Clone(router.states[KindTouchscreen][11].report)
	writesBefore := len(output.writes)

	if err := router.RouteSession(22, pointerReleaseFrame(ReleaseRouteTouchscreen)); err != nil {
		t.Fatalf("touchscreen pointer release: %v", err)
	}
	if len(output.writes) != writesBefore+2 {
		t.Fatalf("restore release writes = %d, want %d", len(output.writes), writesBefore+2)
	}
	if output.writes[writesBefore][touchscreenReportSize-1] != MaxContacts {
		t.Fatalf("restore-clear report = %v, want flushed slots", output.writes[writesBefore])
	}
	if !bytes.Equal(output.writes[writesBefore+1], firstReport) {
		t.Fatalf("restored touchscreen report = %v, want %v", output.writes[writesBefore+1], firstReport)
	}
	if owner := router.owners[KindTouchscreen]; owner != 11 {
		t.Fatalf("touchscreen owner = %d, want 11", owner)
	}
	var restored, restoreCleared bool
	for _, record := range traces.Snapshot(32) {
		restored = restored || record.Stage == "restore"
		restoreCleared = restoreCleared || record.Stage == "restore-clear"
	}
	if !restored || !restoreCleared {
		t.Fatalf("restore traces = %+v", traces.Snapshot(32))
	}
}

func TestRouterPointerReleaseSkipsGatedRoutes(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(16)
	reject := false
	gate := func(kind Kind) error {
		if !reject {
			return nil
		}
		if _, ok := hid2ReportIDForKind(kind); ok {
			return errors.New("pointer mode gate rejected")
		}
		return nil
	}
	router := newRouter(output, traces, gate)

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(0, 0x8000, 0x4000, 0)...,
	)); err != nil {
		t.Fatalf("route absolute pointer: %v", err)
	}
	writesBefore := len(output.writes)
	reject = true

	if err := router.RouteSession(7, pointerReleaseFrame(ReleaseRouteAbsolutePointer)); err != nil {
		t.Fatalf("gated pointer release returned an error: %v", err)
	}
	if len(output.writes) != writesBefore {
		t.Fatalf("gated pointer release wrote %d reports", len(output.writes)-writesBefore)
	}
	if status := router.Status(); status.DispatchFailed != 0 {
		t.Fatalf("gated pointer release failure counters = %+v", status)
	}
	if _, ok := router.owners[KindAbsolutePointer]; ok {
		t.Fatal("gated pointer release kept the absolute pointer owner")
	}
	if len(router.states[KindAbsolutePointer]) != 0 {
		t.Fatalf("gated pointer release states = %+v", router.states[KindAbsolutePointer])
	}
	var gated bool
	for _, record := range traces.Snapshot(16) {
		if record.Stage == "mode-gate" && record.Kind == "absolute-pointer" && record.Error != "" {
			gated = true
		}
	}
	if !gated {
		t.Fatalf("gated pointer release traces = %+v", traces.Snapshot(16))
	}
}

func TestRouterPointerReleaseSkipsStatelessAbsoluteWrite(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(16)
	router := New(output, traces)

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(0, 0x8000, 0x4000, 0)...,
	)); err != nil {
		t.Fatalf("route stateless absolute pointer: %v", err)
	}
	writesBefore := len(output.writes)

	if err := router.RouteSession(7, pointerReleaseFrame(ReleaseRouteAbsolutePointer)); err != nil {
		t.Fatalf("stateless pointer release: %v", err)
	}
	if len(output.writes) != writesBefore {
		t.Fatalf("stateless pointer release wrote %d reports", len(output.writes)-writesBefore)
	}
	if _, ok := router.owners[KindAbsolutePointer]; ok {
		t.Fatal("stateless pointer release kept the absolute pointer owner")
	}
	if len(router.states[KindAbsolutePointer]) != 0 {
		t.Fatalf("stateless pointer release states = %+v", router.states[KindAbsolutePointer])
	}
	for _, record := range traces.Snapshot(16) {
		if record.Kind == "absolute-pointer" && record.Error != "" {
			t.Fatalf("stateless pointer release trace = %+v", record)
		}
	}
}

func TestRouterPointerReleasePublishesClientReleaseTrace(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(16)
	router := New(output, traces)

	if err := router.RouteSession(7, append(makeHeader(SourceBrowserPointer, KindRelativePointer), make([]byte, 7)...)); err != nil {
		t.Fatalf("route relative pointer: %v", err)
	}
	before := router.Status().Dispatched

	if err := router.RouteSession(7, pointerReleaseFrame(ReleaseRouteRelativePointer)); err != nil {
		t.Fatalf("pointer release: %v", err)
	}
	status := router.Status()
	if status.Dispatched != before+1 {
		t.Fatalf("dispatched = %d, want %d", status.Dispatched, before+1)
	}
	if status.DispatchFailed != 0 {
		t.Fatalf("pointer release failure counters = %+v", status)
	}
	clientReleaseRecords := 0
	for _, record := range traces.Snapshot(16) {
		if record.Stage != "client-release" {
			continue
		}
		clientReleaseRecords++
		if record.Kind != "pointer-release" || record.SessionID != 7 {
			t.Fatalf("client-release trace = %+v", record)
		}
	}
	if clientReleaseRecords != 1 {
		t.Fatalf("client-release traces = %d, want 1", clientReleaseRecords)
	}
	for _, session := range status.Sessions {
		if session.ID != 7 {
			continue
		}
		if len(session.Routes) != 0 {
			t.Fatalf("session routes after pointer release = %v", session.Routes)
		}
	}
}

func TestRouterPointerReleaseReportsWriteFailureOnce(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(16)
	router := New(output, traces)

	if err := router.RouteSession(7, append(
		makeHeader(SourceBrowserPointer, KindAbsolutePointer),
		absolutePayload(1, 0x8000, 0x4000, 0)...,
	)); err != nil {
		t.Fatalf("route absolute pointer: %v", err)
	}
	writeErr := errors.New("absolute release write failed")
	output.err = writeErr
	before := router.Status()

	err := router.RouteSession(7, pointerReleaseFrame(ReleaseRouteAbsolutePointer))
	if !errors.Is(err, writeErr) {
		t.Fatalf("pointer release error = %v, want %v", err, writeErr)
	}
	status := router.Status()
	if status.DispatchFailed != before.DispatchFailed+1 {
		t.Fatalf("dispatch failures = %d, want %d", status.DispatchFailed, before.DispatchFailed+1)
	}
	if status.Dispatched != before.Dispatched {
		t.Fatalf("failed pointer release counted as dispatched: %d", status.Dispatched)
	}
	var releaseErrors int
	for _, record := range traces.Snapshot(16) {
		if record.Stage == "release" && record.Error != "" {
			releaseErrors++
		}
	}
	if releaseErrors != 1 {
		t.Fatalf("release error traces = %d, want 1", releaseErrors)
	}
}

func makeHeader(source Source, kind Kind) []byte {
	header := make([]byte, HeaderSize)
	header[0] = ProtocolVersion
	header[1] = byte(source)
	header[2] = byte(kind)
	binary.LittleEndian.PutUint32(header[4:8], 42)
	binary.LittleEndian.PutUint32(header[8:12], 1234)
	return header
}

func absolutePayload(buttons byte, x, y uint16, wheel int16) []byte {
	payload := make([]byte, 7)
	payload[0] = buttons
	binary.LittleEndian.PutUint16(payload[1:3], x)
	binary.LittleEndian.PutUint16(payload[3:5], y)
	binary.LittleEndian.PutUint16(payload[5:7], uint16(wheel))
	return payload
}

func stylusPayload(state byte, x, y uint16, pressure byte) []byte {
	payload := make([]byte, 6)
	payload[0] = state
	binary.LittleEndian.PutUint16(payload[1:3], x)
	binary.LittleEndian.PutUint16(payload[3:5], y)
	payload[5] = pressure
	return payload
}

func touchContact(id, state byte, x, y uint16, pressure byte) []byte {
	payload := make([]byte, 8)
	payload[0] = id
	payload[1] = state
	binary.LittleEndian.PutUint16(payload[2:4], x)
	binary.LittleEndian.PutUint16(payload[4:6], y)
	payload[6] = pressure
	return payload
}

func putInt16(target []byte, value int16) {
	binary.LittleEndian.PutUint16(target, uint16(value))
}

func pointerReleaseFrame(routes uint8) []byte {
	return append(makeHeader(SourceBrowserPointer, KindPointerRelease), routes)
}
