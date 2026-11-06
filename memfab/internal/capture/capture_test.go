package capture

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNormalizeOCRExample(t *testing.T) {
	data, err := os.ReadFile("../../tests/ocr_example.json")
	if err != nil {
		t.Fatal(err)
	}
	var o Observation
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatal(err)
	}
	if err := o.Normalize(time.Now); err != nil {
		t.Fatal(err)
	}
	if o.SchemaVersion != 1 || o.Type != ObservationType {
		t.Fatalf("unexpected schema/type: %+v", o)
	}
	if _, err := ParseTimestamp(o.CapturedAt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(o.OCR.Text, "Sipeed") {
		t.Fatalf("expected OCR text to contain Sipeed")
	}
	if o.OCR.BlockCount == 0 || len(o.OCR.Blocks) == 0 {
		t.Fatalf("expected OCR blocks")
	}
	if len(o.Tags) == 0 {
		t.Fatalf("expected tags")
	}
	if o.CaptureID != "fixture:2026-05-12T19:31:58+08:00" {
		t.Fatalf("capture_id=%q", o.CaptureID)
	}
}

func TestObservationCaptureIDIsRequiredAndSerialized(t *testing.T) {
	o := Observation{
		SchemaVersion: SchemaVersion,
		Type:          ObservationType,
		CapturedAt:    "2026-07-14T05:43:21.123456789Z",
		CaptureID:     "session-id:42",
		OCR:           OCR{Text: "capture id"},
	}
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"capture_id":"session-id:42"`) {
		t.Fatalf("capture_id should be serialized: %s", data)
	}

	var decoded Observation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CaptureID != o.CaptureID {
		t.Fatalf("capture_id=%q, want %q", decoded.CaptureID, o.CaptureID)
	}

	o.CaptureID = " \t"
	if err := o.Normalize(time.Now); err == nil || !strings.Contains(err.Error(), "capture_id is required") {
		t.Fatalf("err=%v, want capture_id validation", err)
	}
}

func TestRejectOldCaptureSchema(t *testing.T) {
	o := Observation{SchemaVersion: 2, Type: "capture", CapturedAt: "2026-05-12T19:00:00+08:00", OCR: OCR{Text: "x"}}
	if err := o.Normalize(time.Now); err == nil {
		t.Fatal("expected old schema rejection")
	}
}

func TestSafeFilenameRoundTrip(t *testing.T) {
	ts := "2026-05-12T19:31:58+08:00"
	stem, err := SafeFilename(ts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stem, ":") || strings.Contains(stem, "+") {
		t.Fatalf("unsafe stem %q", stem)
	}
}
