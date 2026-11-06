package inputrouter

import (
	"encoding/json"
	"testing"
)

func TestRecordBrowserPointerCapturePublishesRawEvent(t *testing.T) {
	hub := NewTraceHub(4)
	router := New(&fakeOutput{}, hub)
	capture := validBrowserPointerCapture()
	data, err := json.Marshal(capture)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if err := router.RecordBrowserCapture(9, data); err != nil {
		t.Fatalf("RecordBrowserCapture: %v", err)
	}
	records := hub.Snapshot(1)
	if len(records) != 1 {
		t.Fatalf("records = %+v", records)
	}
	record := records[0]
	if record.Stage != "browser-capture" || record.Source != "browser-pointer-event" || record.CaptureSequence != capture.Sequence {
		t.Fatalf("capture trace = %+v", record)
	}
	if record.BrowserEvent == nil || record.BrowserEvent.Type != "pointermove" {
		t.Fatalf("browser event = %+v", record.BrowserEvent)
	}
	var content BrowserPointerEventContent
	if err := json.Unmarshal(record.BrowserEvent.Content, &content); err != nil || content.PointerType != "touch" {
		t.Fatalf("browser event content = %+v, err = %v", content, err)
	}
	if status := router.Status(); status.BrowserCaptured != 1 || status.BrowserCaptureRejected != 0 {
		t.Fatalf("capture status = %+v", status)
	}
}

func TestRecordBrowserTouchCapturePublishesSynchronizedContacts(t *testing.T) {
	hub := NewTraceHub(4)
	router := New(&fakeOutput{}, hub)
	capture := BrowserTouchCapture{
		API:          "TouchEvent",
		Sequence:     78,
		CapturedAtMS: 1235,
		Type:         "touchmove",
		Content: BrowserTouchEventContent{
			Touches: []BrowserTouchPoint{
				{Identifier: 4, ClientX: 12, ClientY: 34, Force: 0.5},
				{Identifier: 7, ClientX: 56, ClientY: 78, Force: 1},
			},
			TargetTouches: []BrowserTouchPoint{
				{Identifier: 4, ClientX: 12, ClientY: 34, Force: 0.5},
				{Identifier: 7, ClientX: 56, ClientY: 78, Force: 1},
			},
			ChangedTouches: []BrowserTouchPoint{
				{Identifier: 4, ClientX: 12, ClientY: 34, Force: 0.5},
			},
			TargetID: "kvm-input-region",
		},
	}
	data, err := json.Marshal(capture)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if err := router.RecordBrowserCapture(9, data); err != nil {
		t.Fatalf("RecordBrowserCapture: %v", err)
	}
	record := hub.Snapshot(1)[0]
	if record.Source != "browser-touch-event" || record.Kind != "touch-event" || record.BrowserEvent == nil {
		t.Fatalf("capture trace = %+v", record)
	}
	var content BrowserTouchEventContent
	if err := json.Unmarshal(record.BrowserEvent.Content, &content); err != nil || len(content.TargetTouches) != 2 {
		t.Fatalf("browser touch content = %+v, err = %v", content, err)
	}
}

func TestRouterTraceLinksCaptureContextWithoutDuplicatingBrowserPayload(t *testing.T) {
	hub := NewTraceHub(4)
	router := New(&fakeOutput{}, hub)
	capture := validBrowserPointerCapture()
	data, err := json.Marshal(capture)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := router.RecordBrowserCapture(9, data); err != nil {
		t.Fatalf("RecordBrowserCapture: %v", err)
	}

	frame := makeHeader(SourceBrowserTouch, KindAbsolutePointer)
	frame[3] = FlagCaptureContext
	context := make([]byte, CaptureContextSize)
	putUint32(context[0:4], capture.Sequence)
	putUint32(context[4:8], 6789)
	frame = append(frame, absolutePayload(0, 0x2000, 0x6000, 0)...)
	if err := router.RouteSession(9, append(frame, context...)); err != nil {
		t.Fatalf("RouteSession: %v", err)
	}

	records := hub.Snapshot(2)
	if len(records) != 2 || records[0].BrowserEvent == nil {
		t.Fatalf("traces = %+v", records)
	}
	routed := records[1]
	if routed.CaptureSequence != capture.Sequence || routed.ClientDelayUS != 6789 {
		t.Fatalf("routed capture link = %+v", routed)
	}
	if routed.BrowserEvent != nil {
		t.Fatalf("routed trace duplicated browser payload: %+v", routed.BrowserEvent)
	}
	if routed.Route != "hidg2/absolute-pointer" || routed.Report == "" || routed.RouterDurationUS < 0 || routed.HIDWriteUS < 0 {
		t.Fatalf("routed trace = %+v", routed)
	}
}

func TestRecordBrowserPointerCaptureRejectsUnknownFields(t *testing.T) {
	router := New(&fakeOutput{}, NewTraceHub(4))
	data := []byte(`{"api":"PointerEvent","sequence":1,"capturedAtMS":1,"type":"pointermove","content":{},"unexpected":true}`)
	if err := router.RecordBrowserCapture(1, data); err == nil {
		t.Fatal("unknown browser event field unexpectedly succeeded")
	}
	if status := router.Status(); status.BrowserCaptured != 0 || status.BrowserCaptureRejected != 1 {
		t.Fatalf("capture status = %+v", status)
	}
}

func validBrowserPointerCapture() BrowserPointerCapture {
	return BrowserPointerCapture{
		API:          "PointerEvent",
		Sequence:     77,
		CapturedAtMS: 1234,
		Type:         "pointermove",
		Content: BrowserPointerEventContent{
			PointerID:   3,
			PointerType: "touch",
			IsPrimary:   true,
			ClientX:     12.5,
			ClientY:     34.5,
			Width:       8,
			Height:      9,
			Pressure:    0.5,
			Cancelable:  true,
			Composed:    true,
			TargetID:    "kvm-input-region",
		},
	}
}

func putUint32(target []byte, value uint32) {
	target[0] = byte(value)
	target[1] = byte(value >> 8)
	target[2] = byte(value >> 16)
	target[3] = byte(value >> 24)
}
