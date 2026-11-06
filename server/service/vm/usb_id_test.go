package vm

import "testing"

func TestNormalizeUsbIdHex(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "0x359F", want: "0x359F"},
		{in: "0x359f", want: "0x359F"},
		{in: "359F", want: "0x359F"},
		{in: " 0X1 ", want: "0x0001"},
		{in: "0x3305", want: "0x3305"},
		{in: "", wantErr: true},
		{in: "0x", wantErr: true},
		{in: "0x12345", wantErr: true},
		{in: "xyz", wantErr: true},
		{in: "0xGG", wantErr: true},
	}

	for _, tc := range tests {
		got, err := normalizeUsbIdHex(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeUsbIdHex(%q) expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeUsbIdHex(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("normalizeUsbIdHex(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeUsbIdentityString(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: ""},
		{in: "   ", want: ""},
		{in: "sipeed", want: "sipeed"},
		{in: "  NanoKVM-Go  ", want: "NanoKVM-Go"},
		{in: "has\nnewline", wantErr: true},
		{in: "has\ttab", wantErr: true},
		{in: string(make([]byte, usbIdentityStringMaxLen+1)), wantErr: true},
	}

	// Build a max-length valid string for the happy path.
	maxOK := make([]byte, usbIdentityStringMaxLen)
	for i := range maxOK {
		maxOK[i] = 'a'
	}
	tests = append(tests, struct {
		in      string
		want    string
		wantErr bool
	}{in: string(maxOK), want: string(maxOK)})

	for _, tc := range tests {
		got, err := normalizeUsbIdentityString(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeUsbIdentityString(%q) expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeUsbIdentityString(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("normalizeUsbIdentityString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
