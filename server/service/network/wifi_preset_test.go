package network

import (
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

func TestWifiPresetRequestValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  proto.PresetWifiReq
		want bool
	}{
		{name: "plain ssid without password", req: proto.PresetWifiReq{Ssid: "Office-2.4G"}, want: true},
		{name: "ssid with password", req: proto.PresetWifiReq{Ssid: "Office-2.4G", Password: "sup3r-secret"}, want: true},
		{name: "empty ssid", req: proto.PresetWifiReq{Ssid: "", Password: "x"}, want: false},
		{name: "32 byte ascii ssid", req: proto.PresetWifiReq{Ssid: strings.Repeat("a", 32)}, want: true},
		{name: "33 byte ascii ssid", req: proto.PresetWifiReq{Ssid: strings.Repeat("a", 33)}, want: false},
		{name: "cjk ssid within 32 bytes", req: proto.PresetWifiReq{Ssid: "实验室wifi"}, want: true},
		{name: "11 rune cjk ssid exceeds 32 bytes", req: proto.PresetWifiReq{Ssid: strings.Repeat("实", 11)}, want: false},
		{name: "128 byte password", req: proto.PresetWifiReq{Ssid: "ap", Password: strings.Repeat("p", 128)}, want: true},
		{name: "129 byte password", req: proto.PresetWifiReq{Ssid: "ap", Password: strings.Repeat("p", 129)}, want: false},
		{name: "password with multibyte runes counts bytes", req: proto.PresetWifiReq{Ssid: "ap", Password: strings.Repeat("密", 43)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := wifiPresetRequestValid(&tt.req); got != tt.want {
				t.Fatalf("wifiPresetRequestValid(%+v) = %v, want %v", tt.req, got, tt.want)
			}
		})
	}
}

func TestParseWifiPresetStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain ssid", input: `{"ssid":"Office-2.4G"}` + "\n", want: "Office-2.4G"},
		{name: "empty ssid", input: `{"ssid":""}` + "\n", want: ""},
		{name: "non-ascii ssid with hex escapes", input: `{"ssid":"\xe5\xae\x9e\xe9\xaa\x8c\xe5\xae\xa4"}` + "\n", want: "实验室"},
		{name: "ssid with quote escape", input: `{"ssid":"My\"WiFi"}` + "\n", want: `My"WiFi`},
		{name: "invalid json", input: `{"ssid":`, wantErr: true},
		{name: "wrong type", input: `{"ssid":123}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseWifiPresetStatus([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseWifiPresetStatus(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWifiPresetStatus(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseWifiPresetStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
