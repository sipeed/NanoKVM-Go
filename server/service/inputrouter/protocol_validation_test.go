package inputrouter

import "testing"

func TestDecodeRejectsUnsupportedSemanticValues(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
	}{
		{name: "flags", frame: validationFrameWithFlags()},
		{name: "keyboard reserved", frame: validationFrame(SourceBrowserKeyboard, KindKeyboardState, []byte{0, 1, 0, 0, 0, 0, 0, 0})},
		{name: "keyboard usage", frame: validationFrame(SourceBrowserKeyboard, KindKeyboardState, []byte{0, 0, 0xe8, 0, 0, 0, 0, 0})},
		{name: "relative buttons", frame: validationFrame(SourceBrowserPointer, KindRelativePointer, []byte{0x20, 0, 0, 0, 0, 0, 0})},
		{name: "absolute buttons", frame: validationFrame(SourceBrowserPointer, KindAbsolutePointer, []byte{0x20, 0, 0, 0, 0, 0, 0})},
		{name: "stylus state", frame: validationFrame(SourceBrowserPointer, KindStylus, []byte{0x04, 0, 0, 0, 0, 0})},
		{name: "zero touch contact id", frame: validationFrame(SourceBrowserTouch, KindTouchscreen, []byte{1, 0, 3, 0, 0, 0, 0, 0, 0})},
		{name: "touch contact id", frame: validationFrame(SourceBrowserTouch, KindTouchscreen, []byte{1, 11, 3, 0, 0, 0, 0, 0, 0})},
		{name: "touch state", frame: validationFrame(SourceBrowserTouch, KindTouchscreen, []byte{1, 1, 4, 0, 0, 0, 0, 0, 0})},
		{name: "touch reserved", frame: validationFrame(SourceBrowserTouch, KindTouchscreen, []byte{1, 1, 3, 0, 0, 0, 0, 0, 1})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode(test.frame); err == nil {
				t.Fatalf("Decode(%v) unexpectedly succeeded", test.frame)
			}
		})
	}
}

func TestDecodeAcceptsPointerReleaseMasks(t *testing.T) {
	for _, mask := range []uint8{
		ReleaseRouteRelativePointer,
		ReleaseRouteAbsolutePointer,
		ReleaseRouteStylus,
		ReleaseRouteTouchscreen,
		ReleaseRouteAllPointers,
	} {
		event, err := Decode(validationFrame(SourceBrowserPointer, KindPointerRelease, []byte{mask}))
		if err != nil {
			t.Fatalf("Decode mask 0x%02x: %v", mask, err)
		}
		if event.PointerRelease == nil || event.PointerRelease.Routes != mask {
			t.Fatalf("pointer release event = %+v, want mask 0x%02x", event.PointerRelease, mask)
		}
	}
}

func TestDecodeRejectsInvalidPointerRelease(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
	}{
		{name: "empty payload", frame: validationFrame(SourceBrowserPointer, KindPointerRelease, nil)},
		{name: "two byte payload", frame: validationFrame(SourceBrowserPointer, KindPointerRelease, []byte{1, 0})},
		{name: "zero mask", frame: validationFrame(SourceBrowserPointer, KindPointerRelease, []byte{0})},
		{name: "undefined low bit", frame: validationFrame(SourceBrowserPointer, KindPointerRelease, []byte{0x10})},
		{name: "undefined high bit", frame: validationFrame(SourceBrowserPointer, KindPointerRelease, []byte{0x80})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode(test.frame); err == nil {
				t.Fatalf("Decode(%v) unexpectedly succeeded", test.frame)
			}
		})
	}
}

func TestDecodeAcceptsSemanticBoundaryValues(t *testing.T) {
	frames := [][]byte{
		validationFrame(SourceBrowserKeyboard, KindKeyboardState, []byte{0xff, 0, 0xe7, 0, 0, 0, 0, 0}),
		validationFrame(SourceBrowserPointer, KindRelativePointer, []byte{0x1f, 0, 0, 0, 0, 0, 0}),
		validationFrame(SourceBrowserPointer, KindAbsolutePointer, []byte{0x1f, 0, 0, 0, 0, 0, 0}),
		validationFrame(SourceBrowserPointer, KindStylus, []byte{0x03, 0, 0, 0, 0, 0xff}),
		validationFrame(SourceBrowserPointer, KindTouchscreen, []byte{0}),
		validationFrame(SourceBrowserTouch, KindTouchscreen, []byte{1, 10, 3, 0xff, 0xff, 0xff, 0xff, 0xff, 0}),
	}

	for _, frame := range frames {
		if _, err := Decode(frame); err != nil {
			t.Fatalf("Decode(%v): %v", frame, err)
		}
	}
}

func validationFrame(source Source, kind Kind, payload []byte) []byte {
	return append(semanticFrame(source, kind, nil, 1, 2), payload...)
}

func validationFrameWithFlags() []byte {
	frame := validationFrame(SourceBrowserKeyboard, KindKeyboardState, make([]byte, 8))
	frame[3] = 1
	return frame
}
