package network

import "testing"

func TestFormatWiFiSSID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ssid     string
		idString string
		want     string
	}{
		{name: "2.4 GHz marker", ssid: "Sipeed", idString: wifiBandMarker24G, want: "[2.4G] Sipeed"},
		{name: "5 GHz marker", ssid: "Sipeed", idString: wifiBandMarker5G, want: "[5G] Sipeed"},
		{name: "legacy network", ssid: "Sipeed", want: "Sipeed"},
		{name: "unknown marker", ssid: "Sipeed", idString: "vendor-band:5G", want: "Sipeed"},
		{name: "marker-like suffix", ssid: "Sipeed", idString: "nanokvm-band:5G-extra", want: "Sipeed"},
		{name: "non-ASCII SSID", ssid: "实验室", idString: wifiBandMarker5G, want: "[5G] 实验室"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatWiFiSSID(tt.ssid, tt.idString); got != tt.want {
				t.Fatalf("formatWiFiSSID(%q, %q) = %q, want %q", tt.ssid, tt.idString, got, tt.want)
			}
		})
	}
}
