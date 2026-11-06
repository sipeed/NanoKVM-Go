package stream

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNormalizeVideoStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result         int
		captureEnabled bool
		want           int
	}{
		"positive capture result":     {result: 2, captureEnabled: true, want: VideoStatusNormal},
		"zero capture result":         {result: 0, captureEnabled: true, want: VideoStatusNormal},
		"no image":                    {result: -1, captureEnabled: true, want: -1},
		"encoder error":               {result: -2, captureEnabled: true, want: -2},
		"buffer full":                 {result: -3, captureEnabled: true, want: -3},
		"mode mismatch":               {result: VideoStatusInconsistentMode, captureEnabled: true, want: VideoStatusInconsistentMode},
		"capture disabled on success": {result: 2, captureEnabled: false, want: VideoStatusCaptureDisabled},
		"capture disabled on error":   {result: -1, captureEnabled: false, want: VideoStatusCaptureDisabled},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeVideoStatus(test.result, test.captureEnabled); got != test.want {
				t.Fatalf("NormalizeVideoStatus(%d, %t) = %d, want %d", test.result, test.captureEnabled, got, test.want)
			}
		})
	}
}

func TestMarshalVideoStatus(t *testing.T) {
	t.Parallel()

	payload, err := MarshalVideoStatus(-1)
	if err != nil {
		t.Fatalf("MarshalVideoStatus() error = %v", err)
	}

	var message VideoStatusMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if message.Event != VideoStatusEvent {
		t.Errorf("event = %q, want %q", message.Event, VideoStatusEvent)
	}
	if message.Data != "-1" {
		t.Errorf("data = %q, want %q", message.Data, "-1")
	}
}

func TestIsKnownVideoStatus(t *testing.T) {
	t.Parallel()

	if IsKnownVideoStatus(VideoStatusUnknown) {
		t.Error("unknown status reported as known")
	}
	if !IsKnownVideoStatus(-1) {
		t.Error("negative status reported as unknown")
	}
}

func TestReadVideoFrameForMode(t *testing.T) {
	t.Parallel()

	const (
		expectedMode uint8 = 1
		otherMode    uint8 = 2
		frameResult        = 4
	)

	t.Run("stable mode returns frame", func(t *testing.T) {
		mode := expectedMode
		frame := []byte{1, 2, 3}
		reads := 0

		data, result, ok := ReadVideoFrameForMode(
			expectedMode,
			func() uint8 { return mode },
			func() ([]byte, int) {
				reads++
				return frame, frameResult
			},
		)

		if !ok || result != frameResult || !bytes.Equal(data, frame) {
			t.Fatalf("unexpected read result: data=%v result=%d ok=%t", data, result, ok)
		}
		if reads != 1 {
			t.Fatalf("read count = %d, want 1", reads)
		}
	})

	t.Run("mode switch stops next drain read", func(t *testing.T) {
		mode := expectedMode
		reads := 0
		readFrame := func() ([]byte, int) {
			reads++
			return []byte{1}, frameResult
		}
		currentMode := func() uint8 { return mode }

		_, firstResult, firstOK := ReadVideoFrameForMode(
			expectedMode,
			currentMode,
			readFrame,
		)
		if !firstOK || firstResult != frameResult || reads != 1 {
			t.Fatalf("unexpected first read: result=%d ok=%t reads=%d", firstResult, firstOK, reads)
		}

		mode = otherMode

		data, result, ok := ReadVideoFrameForMode(
			expectedMode,
			currentMode,
			readFrame,
		)

		if ok || result != VideoStatusInconsistentMode || data != nil {
			t.Fatalf("unexpected read result: data=%v result=%d ok=%t", data, result, ok)
		}
		if reads != 1 {
			t.Fatalf("read count = %d, want 1", reads)
		}
	})

	t.Run("mode changed during read", func(t *testing.T) {
		mode := expectedMode

		data, result, ok := ReadVideoFrameForMode(
			expectedMode,
			func() uint8 { return mode },
			func() ([]byte, int) {
				mode = otherMode
				return []byte{1}, frameResult
			},
		)

		if ok || result != VideoStatusInconsistentMode || data != nil {
			t.Fatalf("unexpected read result: data=%v result=%d ok=%t", data, result, ok)
		}
	})
}
