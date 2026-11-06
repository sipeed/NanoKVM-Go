package tailscalegate

import (
	"strings"
	"testing"
)

func TestOffCode(t *testing.T) {
	if got := OffCode(); got != "%off%" {
		t.Fatalf("OffCode() = %q, want %q", got, "%off%")
	}
}

func TestPlainCode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "passthrough digits", in: "123456", want: "123456"},
		{name: "filters non printable", in: "\x00123\x7f", want: "123"},
		{name: "empty stays empty", in: "", want: ""},
		{name: "long value keeps length", in: strings.Repeat("A", 200), want: strings.Repeat("A", 200)},
		{
			name: "long value drops only non printable",
			in:   strings.Repeat("A", 200) + strings.Repeat("\x00", 5),
			want: strings.Repeat("A", 200),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlainCode(tc.in)
			if got != tc.want {
				t.Fatalf("PlainCode(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if len(tc.in) == 200 && len(got) != 200 {
				t.Fatalf("PlainCode preserved length = %d, want 200", len(got))
			}
		})
	}
}

func TestOnlineDirectiveSuccess(t *testing.T) {
	got, err := OnlineDirective("MockKVM", "100.64.0.1")
	if err != nil {
		t.Fatalf("OnlineDirective returned error: %v", err)
	}
	want := "%online%MockKVM%100.64.0.1%"
	if got != want {
		t.Fatalf("OnlineDirective = %q, want %q", got, want)
	}
}

func TestOnlineDirectiveRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name string
		peer string
		ip   string
	}{
		{name: "empty name", peer: "", ip: "100.64.0.1"},
		{name: "empty ip", peer: "MockKVM", ip: ""},
		{name: "percent in name", peer: "Mock%KVM", ip: "100.64.0.1"},
		{name: "percent in ip", peer: "MockKVM", ip: "100.64.0.1%"},
		{name: "percent only name", peer: "%", ip: "100.64.0.1"},
		{name: "percent only ip", peer: "MockKVM", ip: "%"},
		{name: "non printable in name", peer: "Mock\x00KVM", ip: "100.64.0.1"},
		{name: "non printable in ip", peer: "MockKVM", ip: "100.64.0.1\x7f"},
		{name: "whitespace only is not empty but is printable", peer: " ", ip: "100.64.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := OnlineDirective(tc.peer, tc.ip)
			if tc.name == "whitespace only is not empty but is printable" {
				if err != nil {
					t.Fatalf("OnlineDirective(%q, %q) unexpected error: %v", tc.peer, tc.ip, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("OnlineDirective(%q, %q) = %q, want error", tc.peer, tc.ip, got)
			}
			if got != "" {
				t.Fatalf("OnlineDirective(%q, %q) returned %q with error", tc.peer, tc.ip, got)
			}
		})
	}
}
