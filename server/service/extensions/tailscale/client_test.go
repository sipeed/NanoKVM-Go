package tailscale

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateTailscaleLoginURLAcceptsHTTPSURLs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "tailscale control plane",
			raw:  "https://login.tailscale.com/a/example-token",
			want: "https://login.tailscale.com/a/example-token",
		},
		{
			name: "case insensitive scheme",
			raw:  "HTTPS://control.example.test/login/device?token=opaque",
			want: "HTTPS://control.example.test/login/device?token=opaque",
		},
		{
			name: "custom control plane with escaped query",
			raw:  "https://headscale.example.test/register/device?key=value%2Fpart&next=%2Fconsole#finish",
			want: "https://headscale.example.test/register/device?key=value%2Fpart&next=%2Fconsole#finish",
		},
		{
			name: "outer whitespace is trimmed",
			raw:  "  https://login.tailscale.com/a/example-token  ",
			want: "https://login.tailscale.com/a/example-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTailscaleLoginURL(tt.raw)
			if err != nil {
				t.Fatalf("validateTailscaleLoginURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("validateTailscaleLoginURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateTailscaleLoginURLRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "http", raw: "http://login.tailscale.com/a/token"},
		{name: "javascript", raw: "javascript:alert(1)"},
		{name: "script containing https", raw: "<script>location='https://evil.example'</script>"},
		{name: "url embedded in prose", raw: "Visit https://login.tailscale.com/a/token now"},
		{name: "relative", raw: "/login/device"},
		{name: "protocol relative", raw: "//login.tailscale.com/a/token"},
		{name: "missing host", raw: "https:///login/device"},
		{name: "userinfo", raw: "https://user:password@login.tailscale.com/a/token"},
		{name: "multiple values", raw: "https://safe.example/a https://evil.example/b"},
		{name: "invalid escape", raw: "https://login.tailscale.com/a/%zz"},
		{name: "embedded whitespace", raw: "https://login.tailscale.com/a/token\tmore"},
		{name: "embedded control", raw: "https://login.tailscale.com/a/token\x00more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTailscaleLoginURL(tt.raw)
			if got != "" {
				t.Fatalf("validateTailscaleLoginURL() = %q, want empty URL", got)
			}
			if !errors.Is(err, errInvalidTailscaleLoginURL) {
				t.Fatalf("validateTailscaleLoginURL() error = %v, want %v", err, errInvalidTailscaleLoginURL)
			}
			if strings.Contains(err.Error(), tt.raw) {
				t.Fatalf("validation error contains the untrusted URL: %v", err)
			}
		})
	}
}

// TestLocalAPIClientEndpointContract is the request-shape contract for the
// single-request LocalClient methods: each call has to produce exactly one
// LocalAPI request with the exact method, path, Host, Content-Type and body
// the daemon expects. Calls without a payload must not send a body or a
// Content-Type.
//
// Login() is deliberately absent: it is a multi-request flow (PUT /profiles/,
// PATCH /prefs, an optional POST /start, POST /login-interactive and the
// AuthURL poll), so its contract lives in the login cases of localapi_test.go.
func TestLocalAPIClientEndpointContract(t *testing.T) {
	type capturedRequest struct {
		method      string
		path        string
		host        string
		contentType string
		body        string
	}

	const (
		subnetRoute = "192.168.0.0/22"
		otherSubnet = "10.0.0.0/24"
	)

	tests := []struct {
		name        string
		call        func(client *LocalClient) error
		method      string
		path        string
		contentType string
		body        string
	}{
		{
			name:        "Up patches WantRunning",
			call:        func(client *LocalClient) error { return client.Up() },
			method:      http.MethodPatch,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
			contentType: "application/json",
			body:        `{"WantRunning":true,"WantRunningSet":true}`,
		},
		{
			name:        "Down patches WantRunning",
			call:        func(client *LocalClient) error { return client.Down() },
			method:      http.MethodPatch,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
			contentType: "application/json",
			body:        `{"WantRunning":false,"WantRunningSet":true}`,
		},
		{
			name: "Status reads the status endpoint",
			call: func(client *LocalClient) error {
				_, err := client.Status()
				return err
			},
			method: http.MethodGet,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath,
		},
		{
			name: "Prefs reads the prefs endpoint",
			call: func(client *LocalClient) error {
				_, err := client.Prefs()
				return err
			},
			method: http.MethodGet,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
		},
		{
			name:   "Logout posts to the logout endpoint",
			call:   func(client *LocalClient) error { return client.Logout() },
			method: http.MethodPost,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath,
		},
		{
			name:        "SetRouting without the exit node patches the subnet routes",
			call:        func(client *LocalClient) error { return client.SetRouting(false, []string{otherSubnet, subnetRoute}) },
			method:      http.MethodPatch,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
			contentType: "application/json",
			body:        `{"AdvertiseRoutes":["10.0.0.0/24","192.168.0.0/22"],"AdvertiseRoutesSet":true}`,
		},
		{
			name:        "SetRouting with the exit node prepends the default routes",
			call:        func(client *LocalClient) error { return client.SetRouting(true, []string{subnetRoute}) },
			method:      http.MethodPatch,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
			contentType: "application/json",
			body:        `{"AdvertiseRoutes":["0.0.0.0/0","::/0","192.168.0.0/22"],"AdvertiseRoutesSet":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []capturedRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				requests = append(requests, capturedRequest{
					method:      r.Method,
					path:        r.URL.Path,
					host:        r.Host,
					contentType: r.Header.Get("Content-Type"),
					body:        string(body),
				})

				switch {
				case r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStatusPath:
					_, _ = io.WriteString(w, deviceStatusJSON)
				case r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIPrefsPath:
					_, _ = io.WriteString(w, devicePrefsJSON)
				case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILogoutPath:
					// POST /logout answers 204 with no body.
					w.WriteHeader(http.StatusNoContent)
				}
				// PATCH /prefs answers 200 with an empty body.
			}))
			defer server.Close()

			client := NewLocalClient()
			client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

			if err := tt.call(client); err != nil {
				t.Fatalf("call error = %v", err)
			}

			if len(requests) != 1 {
				t.Fatalf("requests = %#v, want exactly 1", requests)
			}
			request := requests[0]
			if request.method != tt.method {
				t.Errorf("request method = %q, want %q", request.method, tt.method)
			}
			if request.path != tt.path {
				t.Errorf("request path = %q, want %q", request.path, tt.path)
			}
			if request.host != tailscaleLocalAPIHost {
				t.Errorf("request Host = %q, want %q", request.host, tailscaleLocalAPIHost)
			}
			if request.contentType != tt.contentType {
				t.Errorf("request Content-Type = %q, want %q", request.contentType, tt.contentType)
			}
			if request.body != tt.body {
				t.Errorf("request body = %q, want %q", request.body, tt.body)
			}
		})
	}
}

func TestLocalAPITimeoutIsBounded(t *testing.T) {
	if tailscaleLocalAPITimeout < 5*time.Second || tailscaleLocalAPITimeout > 30*time.Second {
		t.Fatalf("tailscale localapi timeout = %s, want 5s..30s", tailscaleLocalAPITimeout)
	}
}
