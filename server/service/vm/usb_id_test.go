package vm

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestUsbIdentityLastWriteRestoresOverrides(t *testing.T) {
	dir := t.TempDir()
	vid, serial, product := filepath.Join(dir, "vid"), filepath.Join(dir, "serial"), filepath.Join(dir, "product")
	last := filepath.Join(dir, "missing-parent", "configuration")
	if err := os.WriteFile(vid, []byte("old vid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(serial, []byte("old serial\n"), 0644); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{vid: "new vid", product: "new product", last: "new configuration"}
	err := applyUsbIdentity([]string{vid, serial, product, last}, values, func() error {
		t.Fatal("unexpected gadget restart")
		return nil
	})
	if err == nil {
		t.Fatal("expected last write to fail")
	}
	for path, want := range map[string]string{vid: "old vid\n", serial: "old serial\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s after rollback: %q, %v", path, got, err)
		}
	}
	if _, err := os.Stat(product); !os.IsNotExist(err) {
		t.Fatalf("new override survived: %v", err)
	}
}
