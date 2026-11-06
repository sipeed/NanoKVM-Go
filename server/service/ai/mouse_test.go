package ai

import "testing"

func TestAbsoluteCoordinateClampsToHIDRange(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  uint16
	}{
		{name: "negative", value: -0.1, want: 0},
		{name: "zero", value: 0, want: 0},
		{name: "middle", value: 0.5, want: 0x4000},
		{name: "one", value: 1, want: 0x7fff},
		{name: "above one", value: 1.1, want: 0x7fff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := absoluteCoordinate(tt.value); got != tt.want {
				t.Fatalf("absoluteCoordinate(%v) = %#x, want %#x", tt.value, got, tt.want)
			}
		})
	}
}

func TestBuildAbsolutePointerReport(t *testing.T) {
	report := buildAbsolutePointerReport(0x1234, 0x5678, true, false)
	want := []byte{0x01, 0x01, 0x34, 0x12, 0x78, 0x56, 0}

	if string(report) != string(want) {
		t.Fatalf("absolute pointer report = %v, want %v", report, want)
	}
}

func TestBuildAbsolutePointerReleaseReport(t *testing.T) {
	report := buildAbsolutePointerReport(0x1234, 0x5678, false, false)
	want := []byte{0x01, 0, 0x34, 0x12, 0x78, 0x56, 0}

	if string(report) != string(want) {
		t.Fatalf("absolute pointer release report = %v, want %v", report, want)
	}
}

func TestBuildAbsolutePointerSecondaryButtonReport(t *testing.T) {
	report := buildAbsolutePointerReport(0x1234, 0x5678, true, true)
	want := []byte{0x01, 0x03, 0x34, 0x12, 0x78, 0x56, 0}

	if string(report) != string(want) {
		t.Fatalf("absolute pointer secondary report = %v, want %v", report, want)
	}
}
