package vm

import (
	"strings"
	"testing"
)

func TestNormalizeDiskInquiryString(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "standard identity", in: "sipeed  Flash Drive     1.00", want: "sipeed  Flash Drive     1.00"},
		{name: "short identity", in: "NanoKVM", want: "NanoKVM"},
		{name: "surrounding whitespace", in: "  NanoKVM  ", want: "NanoKVM"},
		{name: "empty", in: "   ", wantErr: true},
		{name: "maximum length", in: strings.Repeat("A", diskInquiryMaxLen), want: strings.Repeat("A", diskInquiryMaxLen)},
		{name: "too long", in: strings.Repeat("A", diskInquiryMaxLen+1), wantErr: true},
		{name: "non ASCII", in: "NanoKVM 存储", wantErr: true},
		{name: "newline", in: "NanoKVM\nDrive", wantErr: true},
		{name: "delete", in: "NanoKVM\x7fDrive", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeDiskInquiryString(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("normalizeDiskInquiryString(%q) expected error, got %q", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDiskInquiryString(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeDiskInquiryString(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeBootString(t *testing.T) {
	tests := []struct {
		in      string
		maxLen  int
		want    string
		wantErr bool
	}{
		{in: "sipeed  Flash Drive     1.00", maxLen: diskInquiryMaxLen, want: "sipeed  Flash Drive     1.00"},
		{in: "  NanoKVM-Go Audio  ", maxLen: uac2FunctionNameMaxLen, want: "NanoKVM-Go Audio"},
		{in: "", maxLen: 36, wantErr: true},
		{in: "   ", maxLen: 36, wantErr: true},
		{in: "has\nnewline", maxLen: 36, wantErr: true},
		{in: "has\ttab", maxLen: 36, wantErr: true},
		{in: "exactly-thirty-six-characters-here!!", maxLen: 36, want: "exactly-thirty-six-characters-here!!"},
		{in: "exactly-thirty-six-characters-here!!!", maxLen: 36, wantErr: true},
	}

	for _, tc := range tests {
		got, err := normalizeBootString(tc.in, tc.maxLen)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeBootString(%q, %d) expected error, got %q", tc.in, tc.maxLen, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeBootString(%q, %d) unexpected error: %v", tc.in, tc.maxLen, err)
		}
		if got != tc.want {
			t.Fatalf("normalizeBootString(%q, %d) = %q, want %q", tc.in, tc.maxLen, got, tc.want)
		}
	}
}
