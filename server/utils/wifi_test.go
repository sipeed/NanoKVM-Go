package utils

import "testing"

func TestParseWiFiDetailsIncludesIDString(t *testing.T) {
	t.Parallel()

	details := parseWiFiDetails([]byte("bssid=04:f4:1c:f2:b8:c6\nssid=Sipeed\nid=0\nid_str=nanokvm-band:5G\nfreq=5320\nwpa_state=COMPLETED\n"))

	if details.SSID != "Sipeed" {
		t.Fatalf("SSID = %q, want Sipeed", details.SSID)
	}
	if details.IDString != "nanokvm-band:5G" {
		t.Fatalf("IDString = %q, want nanokvm-band:5G", details.IDString)
	}
	if details.Freq != "5320" {
		t.Fatalf("Freq = %q, want 5320", details.Freq)
	}
}

func TestParseWiFiDetailsPreservesEqualsAndDecodesSSID(t *testing.T) {
	t.Parallel()

	details := parseWiFiDetails([]byte("ssid=Lab=\\xe4\\xb8\\xad\nid_str=vendor=value\n"))

	if details.SSID != "Lab=中" {
		t.Fatalf("SSID = %q, want %q", details.SSID, "Lab=中")
	}
	if details.IDString != "vendor=value" {
		t.Fatalf("IDString = %q, want vendor=value", details.IDString)
	}
}
