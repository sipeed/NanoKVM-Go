package utils

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		host      string
		origin    string
		tls       bool
		forwarded string
		want      bool
	}{
		{name: "missing origin native client", target: "http://nanokvm.local/api/ws", host: "nanokvm.local", want: true},
		{name: "same origin HTTP", target: "http://nanokvm.local/api/ws", host: "nanokvm.local", origin: "http://nanokvm.local", want: true},
		{name: "same origin HTTPS", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://nanokvm.local", tls: true, want: true},
		{name: "HTTP default port", target: "http://nanokvm.local/api/ws", host: "nanokvm.local:80", origin: "http://nanokvm.local", want: true},
		{name: "HTTPS default port", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://nanokvm.local:443", tls: true, want: true},
		{name: "same explicit port", target: "http://nanokvm.local:8080/api/ws", host: "nanokvm.local:8080", origin: "http://nanokvm.local:8080", want: true},
		{name: "same DNS root notation", target: "https://nanokvm.local./api/ws", host: "nanokvm.local.", origin: "https://nanokvm.local.", tls: true, want: true},
		{name: "equivalent DNS root notation", target: "https://nanokvm.local./api/ws", host: "nanokvm.local.", origin: "https://nanokvm.local", tls: true, want: true},
		{name: "numeric ports normalized", target: "http://nanokvm.local:8080/api/ws", host: "nanokvm.local:08080", origin: "http://nanokvm.local:8080", want: true},
		{name: "IPv4 same origin", target: "http://192.168.1.9/api/ws", host: "192.168.1.9", origin: "http://192.168.1.9", want: true},
		{name: "IPv6 normalized", target: "https://[2001:db8::1]/api/ws", host: "[2001:0db8::1]", origin: "https://[2001:db8::1]", tls: true, want: true},
		{name: "foreign host", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://evil.example", tls: true},
		{name: "foreign port", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://nanokvm.local:444", tls: true},
		{name: "foreign scheme", target: "http://nanokvm.local/api/ws", host: "nanokvm.local:443", origin: "https://nanokvm.local", want: false},
		{name: "forwarded headers ignored", target: "http://127.0.0.1/api/ws", host: "127.0.0.1", origin: "https://public.example", forwarded: "public.example"},
		{name: "credentials rejected", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://user@nanokvm.local", tls: true},
		{name: "path rejected", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://nanokvm.local/path", tls: true},
		{name: "opaque origin rejected", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "null", tls: true},
		{name: "script scheme rejected", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "javascript:alert(1)", tls: true},
		{name: "malformed origin rejected", target: "https://nanokvm.local/api/ws", host: "nanokvm.local", origin: "https://[", tls: true},
		{name: "malformed request host rejected", target: "https://nanokvm.local/api/ws", host: "bad:host:name", origin: "https://nanokvm.local", tls: true},
		{name: "named origin port rejected", target: "http://nanokvm.local/api/ws", host: "nanokvm.local", origin: "http://nanokvm.local:http"},
		{name: "empty origin port rejected", target: "http://nanokvm.local/api/ws", host: "nanokvm.local", origin: "http://nanokvm.local:"},
		{name: "multiple DNS root dots rejected", target: "https://nanokvm.local../api/ws", host: "nanokvm.local..", origin: "https://nanokvm.local..", tls: true},
		{name: "out of range origin port rejected", target: "http://nanokvm.local/api/ws", host: "nanokvm.local", origin: "http://nanokvm.local:65536"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.target, nil)
			req.Host = test.host
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.forwarded != "" {
				req.Header.Set("X-Forwarded-Host", test.forwarded)
				req.Header.Set("X-Forwarded-Proto", "https")
			}
			if test.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if got := CheckWebSocketOrigin(req); got != test.want {
				t.Fatalf("CheckWebSocketOrigin() = %t, want %t", got, test.want)
			}
		})
	}
}
