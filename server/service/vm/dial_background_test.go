package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDialBackgroundImageName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "dial_bg.bmp", want: true},
		{name: "Dial-01.BMP", want: true},
		{name: "../dial.bmp", want: false},
		{name: "sub/dial.bmp", want: false},
		{name: "dial bg.bmp", want: false},
		{name: "dial.png", want: false},
		{name: ".bmp", want: false},
		{name: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDialBackgroundImageName(tt.name); got != tt.want {
				t.Fatalf("isDialBackgroundImageName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidateBMPFile(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.bmp")
	invalidPath := filepath.Join(dir, "invalid.bmp")

	if err := os.WriteFile(validPath, []byte{'B', 'M', 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(invalidPath, []byte{'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateBMPFile(validPath); err != nil {
		t.Fatalf("validateBMPFile(validPath) failed: %s", err)
	}

	if err := validateBMPFile(invalidPath); err == nil {
		t.Fatal("validateBMPFile(invalidPath) succeeded, want error")
	}
}
