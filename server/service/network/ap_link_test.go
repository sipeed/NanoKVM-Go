package network

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setAPLinkSeams(t *testing.T, neighbors map[string]string, wireless map[string]bool) {
	t.Helper()

	previousNeighbor := apLinkNeighborDevice
	previousWireless := isWirelessInterface
	apLinkNeighborDevice = func(addr netip.Addr) (string, bool) {
		device, ok := neighbors[addr.String()]
		return device, ok
	}
	isWirelessInterface = func(name string) bool { return wireless[name] }
	t.Cleanup(func() {
		apLinkNeighborDevice = previousNeighbor
		isWirelessInterface = previousWireless
	})
}

func newAPLinkRequest(local string, remote string, header bool) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "http://192.168.12.1/api/network/wifi", nil)
	req.RemoteAddr = remote
	if local != "" {
		addr := net.TCPAddrFromAddrPort(netip.MustParseAddrPort(local))
		req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, addr))
	}
	if header {
		req.Header.Set(APPortalHeader, "1")
	}
	return req
}

func TestIsAPLinkRequest(t *testing.T) {
	setAPLinkSeams(t,
		map[string]string{"192.168.12.100": "wlan0", "192.168.12.101": "eth0"},
		map[string]bool{"wlan0": true},
	)

	tests := []struct {
		name   string
		local  string
		remote string
		header bool
		want   bool
	}{
		{name: "station on AP", local: "192.168.12.1:80", remote: "192.168.12.100:50000", header: true, want: true},
		{name: "station on AP via v4-mapped local", local: "[::ffff:192.168.12.1]:80", remote: "[::ffff:192.168.12.100]:50000", header: true, want: true},
		{name: "missing portal header", local: "192.168.12.1:80", remote: "192.168.12.100:50000", want: false},
		{name: "no local address", remote: "192.168.12.100:50000", header: true, want: false},
		{name: "request to LAN address", local: "192.168.1.20:80", remote: "192.168.12.100:50000", header: true, want: false},
		{name: "source outside AP subnet", local: "192.168.12.1:80", remote: "192.168.1.50:50000", header: true, want: false},
		{name: "source is the gateway", local: "192.168.12.1:80", remote: "192.168.12.1:50000", header: true, want: false},
		{name: "neighbor on wired interface", local: "192.168.12.1:80", remote: "192.168.12.101:50000", header: true, want: false},
		{name: "no neighbor entry", local: "192.168.12.1:80", remote: "192.168.12.102:50000", header: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAPLinkRequest(newAPLinkRequest(tt.local, tt.remote, tt.header)); got != tt.want {
				t.Fatalf("isAPLinkRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAPLinkRequestRequiresGatewayHost(t *testing.T) {
	setAPLinkSeams(t, map[string]string{"192.168.12.100": "wlan0"}, map[string]bool{"wlan0": true})

	for host, want := range map[string]bool{
		"192.168.12.1":        true,
		"192.168.12.1:80":     true,
		"192.168.12.1:443":    true,
		"192.168.12.1:8080":   false,
		"evil.example":        false,
		"evil.example:80":     false,
		"192.168.12.1.nip.io": false,
		"":                    false,
	} {
		req := newAPLinkRequest("192.168.12.1:80", "192.168.12.100:50000", true)
		req.Host = host
		if got := isAPLinkRequest(req); got != want {
			t.Errorf("isAPLinkRequest(Host=%q) = %v, want %v", host, got, want)
		}
	}
}

func TestFindARPDevice(t *testing.T) {
	const table = `IP address       HW type     Flags       HW address            Mask     Device
192.168.12.100   0x1         0x2         aa:bb:cc:dd:ee:01     *        wlan0
192.168.12.101   0x1         0x0         00:00:00:00:00:00     *        wlan0
192.168.1.1      0x1         0x6         aa:bb:cc:dd:ee:02     *        eth0
`
	tests := []struct {
		addr   string
		device string
		ok     bool
	}{
		{addr: "192.168.12.100", device: "wlan0", ok: true},
		{addr: "192.168.12.101", ok: false},
		{addr: "192.168.1.1", device: "eth0", ok: true},
		{addr: "192.168.12.200", ok: false},
	}

	for _, tt := range tests {
		scanner := bufio.NewScanner(strings.NewReader(table))
		device, ok := findARPDevice(scanner, netip.MustParseAddr(tt.addr))
		if device != tt.device || ok != tt.ok {
			t.Fatalf("findARPDevice(%s) = %q, %v; want %q, %v", tt.addr, device, ok, tt.device, tt.ok)
		}
	}
}

func performVerifyApLogin(t *testing.T, req *http.Request) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	NewService().VerifyApLogin(c)

	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	return response.Code
}

func TestVerifyApLoginAcceptsAPLinkWithoutKey(t *testing.T) {
	setScanAuthorizationSeams(t, true, "ap-secret", nil)
	setAPLinkSeams(t, map[string]string{"192.168.12.100": "wlan0"}, map[string]bool{"wlan0": true})

	if code := performVerifyApLogin(t, newAPLinkRequest("192.168.12.1:80", "192.168.12.100:50000", true)); code != 0 {
		t.Fatalf("AP link verify code = %d, want 0", code)
	}
}

func TestVerifyApLoginRejectsNonLinkWithoutKey(t *testing.T) {
	setScanAuthorizationSeams(t, true, "ap-secret", nil)
	setAPLinkSeams(t, map[string]string{}, map[string]bool{"wlan0": true})

	if code := performVerifyApLogin(t, newAPLinkRequest("192.168.12.1:80", "192.168.12.100:50000", true)); code != -4 {
		t.Fatalf("non-link verify code = %d, want -4", code)
	}
}

func TestVerifyApLoginRejectsAPLinkOutsideConfigurationMode(t *testing.T) {
	setScanAuthorizationSeams(t, false, "ap-secret", nil)
	setAPLinkSeams(t, map[string]string{"192.168.12.100": "wlan0"}, map[string]bool{"wlan0": true})

	if code := performVerifyApLogin(t, newAPLinkRequest("192.168.12.1:80", "192.168.12.100:50000", true)); code != -1 {
		t.Fatalf("verify outside configuration mode code = %d, want -1", code)
	}
}
