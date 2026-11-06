package inputrouter

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestAdaptLegacyKeyboard(t *testing.T) {
	report := []byte{0x02, 0, 0x04, 0, 0, 0, 0, 0}
	frame, kind, err := adaptLegacyKeyboard(report, 17, 29)
	if err != nil {
		t.Fatalf("adaptLegacyKeyboard: %v", err)
	}
	if kind != KindKeyboardState {
		t.Fatalf("kind = %d, want %d", kind, KindKeyboardState)
	}
	event, err := Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if event.Source != SourceLegacyKeyboard || event.Sequence != 17 || event.CapturedAtMS != 29 {
		t.Fatalf("event metadata = %+v", event)
	}
	if !bytes.Equal(event.Keyboard.Report[:], report) {
		t.Fatalf("keyboard report = %v, want %v", event.Keyboard.Report, report)
	}
}

func TestAdaptLegacyRelativePointer(t *testing.T) {
	frame, kind, err := adaptLegacyPointer([]byte{0x03, 0x80, 0x7f, 0xff}, 4, 8)
	if err != nil {
		t.Fatalf("adaptLegacyPointer: %v", err)
	}
	if kind != KindRelativePointer {
		t.Fatalf("kind = %d, want %d", kind, KindRelativePointer)
	}
	event, err := Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := RelativePointer{Buttons: 0x03, DeltaX: -128, DeltaY: 127, Wheel: -1}
	if *event.Relative != want || event.Source != SourceLegacyPointer {
		t.Fatalf("relative event = %+v, want %+v", event.Relative, want)
	}
}

func TestAdaptLegacyAbsoluteMouse(t *testing.T) {
	report := []byte{absoluteReportIDMouse, 0x05, 0x00, 0x40, 0xff, 0x7f, 0xff}
	frame, kind, err := adaptLegacyPointer(report, 3, 5)
	if err != nil {
		t.Fatalf("adaptLegacyPointer: %v", err)
	}
	if kind != KindAbsolutePointer {
		t.Fatalf("kind = %d, want %d", kind, KindAbsolutePointer)
	}
	event, err := Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if event.Absolute.Buttons != 0x05 || event.Absolute.X != expandLegacyAxis(0x4000) || event.Absolute.Y != 0xffff || event.Absolute.Wheel != -1 {
		t.Fatalf("absolute event = %+v", event.Absolute)
	}
}

func TestAdaptLegacyStylus(t *testing.T) {
	report := []byte{absoluteReportIDStylus, 0x03, 0, 0, 0xff, 0x7f, 201}
	frame, kind, err := adaptLegacyPointer(report, 3, 5)
	if err != nil {
		t.Fatalf("adaptLegacyPointer: %v", err)
	}
	if kind != KindStylus {
		t.Fatalf("kind = %d, want %d", kind, KindStylus)
	}
	event, err := Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := Stylus{State: 0x03, X: 0, Y: 0xffff, Pressure: 201}
	if *event.Stylus != want {
		t.Fatalf("stylus event = %+v, want %+v", event.Stylus, want)
	}
}

func TestAdaptLegacyTouchscreen(t *testing.T) {
	report := make([]byte, touchscreenReportSize)
	report[0] = touchscreenReportID
	report[1] = 0x03
	report[2] = 4
	binary.LittleEndian.PutUint16(report[3:5], 0)
	binary.LittleEndian.PutUint16(report[5:7], 0x7fff)
	report[7] = 0x03
	report[8] = 9
	binary.LittleEndian.PutUint16(report[9:11], 0x7fff)
	binary.LittleEndian.PutUint16(report[11:13], 0)
	report[len(report)-1] = 2

	frame, kind, err := adaptLegacyPointer(report, 11, 13)
	if err != nil {
		t.Fatalf("adaptLegacyPointer: %v", err)
	}
	if kind != KindTouchscreen {
		t.Fatalf("kind = %d, want %d", kind, KindTouchscreen)
	}
	event, err := Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	contacts := event.Touchscreen.Contacts
	if len(contacts) != 2 {
		t.Fatalf("contact count = %d, want 2", len(contacts))
	}
	if contacts[0].ID != 4 || contacts[0].State != 3 || contacts[0].X != 0 || contacts[0].Y != 0xffff {
		t.Fatalf("contact 0 = %+v", contacts[0])
	}
	if contacts[1].ID != 9 || contacts[1].State != 3 || contacts[1].X != 0xffff || contacts[1].Y != 0 {
		t.Fatalf("contact 1 = %+v", contacts[1])
	}
}

func TestAdaptLegacyRejectsInvalidReports(t *testing.T) {
	if _, _, err := adaptLegacyKeyboard(make([]byte, 7), 0, 0); err == nil {
		t.Fatal("short keyboard report unexpectedly accepted")
	}
	if _, _, err := adaptLegacyPointer(make([]byte, 5), 0, 0); err == nil {
		t.Fatal("unknown pointer report length unexpectedly accepted")
	}
	if _, _, err := adaptLegacyPointer([]byte{99, 0, 0, 0, 0, 0, 0}, 0, 0); err == nil {
		t.Fatal("unknown absolute report ID unexpectedly accepted")
	}
	touch := make([]byte, touchscreenReportSize)
	touch[0] = touchscreenReportID
	touch[len(touch)-1] = MaxContacts + 1
	if _, _, err := adaptLegacyPointer(touch, 0, 0); err == nil {
		t.Fatal("excessive touch contact count unexpectedly accepted")
	}
}

func TestRouteLegacyRecordsAdaptationAndDispatch(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(8)
	router := New(output, traces)
	router.now = func() time.Time { return time.UnixMilli(1234) }
	report := []byte{0, 1, 0x80, 0}

	if err := router.RouteLegacyPointerSession(77, report, 6, 9); err != nil {
		t.Fatalf("RouteLegacyPointerSession: %v", err)
	}
	if output.route != "relative" || !bytes.Equal(output.report, []byte{0, 1, 0x81, 0}) {
		t.Fatalf("output = %s %v", output.route, output.report)
	}
	records := traces.Snapshot(2)
	if len(records) != 2 || records[0].Stage != "legacy-adapted" || records[1].Stage != "dispatched" {
		t.Fatalf("records = %+v", records)
	}
	if records[0].Capture != "00018000" || records[1].Source != "legacy-websocket-pointer" || records[1].Route != "hidg1/relative-pointer" {
		t.Fatalf("legacy traces = %+v", records)
	}
	status := router.Status()
	if status.Received != 1 || status.Dispatched != 1 || status.Rejected != 0 || status.ActiveSessions != 1 {
		t.Fatalf("runtime status = %+v", status)
	}
	if len(status.Sources) != 1 || status.Sources[0].Source != "legacy-websocket-pointer" || status.Sources[0].Events != 1 {
		t.Fatalf("source status = %+v", status.Sources)
	}
}

func TestRouteLegacyRecordsAdaptationError(t *testing.T) {
	router := New(&fakeOutput{}, NewTraceHub(4))
	router.now = func() time.Time { return time.UnixMilli(4321) }

	if err := router.RouteLegacyKeyboardSession(88, []byte{1, 2}, 7, 10); err == nil {
		t.Fatal("short keyboard report unexpectedly routed")
	}
	status := router.Status()
	if status.Received != 1 || status.Dispatched != 0 || status.Rejected != 1 || status.DecodeRejected != 1 {
		t.Fatalf("runtime status = %+v", status)
	}
	records := router.Traces().Snapshot(1)
	if len(records) != 1 || records[0].Stage != "legacy-adapt-error" || records[0].Source != "legacy-websocket-keyboard" || records[0].SessionID != 88 {
		t.Fatalf("error trace = %+v", records)
	}
}
