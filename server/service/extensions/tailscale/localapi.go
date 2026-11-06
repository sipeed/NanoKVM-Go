package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
)

const (
	// tailscaled serves its LocalAPI on a unix socket. /run is the location
	// used by the units installed by this integration; /var/run is kept as a
	// fallback for layouts that still point there.
	tailscaleLocalAPISocketPath         = "/run/tailscale/tailscaled.sock"
	tailscaleLocalAPISocketFallbackPath = "/var/run/tailscale/tailscaled.sock"

	// The daemon validates the Host header and rejects anything else (for
	// example localhost) with 403, so the request URL has to use this host.
	tailscaleLocalAPIHost     = "local-tailscaled.sock"
	tailscaleLocalAPIBasePath = "/localapi/v0"

	tailscaleLocalAPIStatusPath           = "/status"
	tailscaleLocalAPIPrefsPath            = "/prefs"
	tailscaleLocalAPILogoutPath           = "/logout"
	tailscaleLocalAPIProfilesPath         = "/profiles/"
	tailscaleLocalAPICurrentProfilePath   = "/profiles/current"
	tailscaleLocalAPIStartPath            = "/start"
	tailscaleLocalAPILoginInteractivePath = "/login-interactive"

	tailscaleLocalAPIMaxResponseBytes = 8 << 20
	tailscaleLocalAPIErrorTextLimit   = 256
)

// localAPIPrefsPatch is the masked-prefs body of PATCH /localapi/v0/prefs.
// The daemon only applies the fields whose Set companion is true, so every
// field that is written has to be paired with its mask flag.
type localAPIPrefsPatch struct {
	WantRunning    *bool `json:"WantRunning,omitempty"`
	WantRunningSet bool  `json:"WantRunningSet,omitempty"`

	// AdvertiseRoutes is a pointer so that an empty list still serializes as
	// [] : a zero-length value would be dropped by omitempty and leave the
	// daemon with nothing but the mask bit.
	AdvertiseRoutes    *[]string `json:"AdvertiseRoutes,omitempty"`
	AdvertiseRoutesSet bool      `json:"AdvertiseRoutesSet,omitempty"`

	// CorpDNS is a pointer for the same reason: login writes false, which
	// omitempty would otherwise drop before the daemon ever sees it.
	CorpDNS    *bool `json:"CorpDNS,omitempty"`
	CorpDNSSet bool  `json:"CorpDNSSet,omitempty"`
}

// localAPIClient talks to the tailscaled LocalAPI over its unix socket.
type localAPIClient struct {
	socketPath         string
	fallbackSocketPath string
	dial               func(ctx context.Context, network, addr string) (net.Conn, error)

	httpClient *http.Client
}

func newLocalAPIClient() *localAPIClient {
	return &localAPIClient{
		socketPath:         tailscaleLocalAPISocketPath,
		fallbackSocketPath: tailscaleLocalAPISocketFallbackPath,
	}
}

// newLocalAPIClientWithDial builds a client whose transport delegates to dial
// instead of connecting to the tailscaled unix socket, so tests can serve
// LocalAPI responses from an httptest server or a temporary socket.
func newLocalAPIClientWithDial(dial func(ctx context.Context, network, addr string) (net.Conn, error)) *localAPIClient {
	client := newLocalAPIClient()
	client.dial = dial
	return client
}

func (c *localAPIClient) client() *http.Client {
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: &http.Transport{
				// Every LocalClient builds its own client and is dropped after the
				// request, so idle connections must not linger between operations.
				DisableKeepAlives:  true,
				DisableCompression: true,
				DialContext:        c.dialContext,
			},
		}
	}
	return c.httpClient
}

func (c *localAPIClient) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if c.dial != nil {
		return c.dial(ctx, network, addr)
	}

	dialer := &net.Dialer{}
	socketPath := c.socketPath
	if socketPath == "" {
		socketPath = tailscaleLocalAPISocketPath
	}

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err == nil {
		return conn, nil
	}
	if c.fallbackSocketPath == "" || c.fallbackSocketPath == socketPath ||
		!errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return dialer.DialContext(ctx, "unix", c.fallbackSocketPath)
}

// do performs a LocalAPI request and decodes the JSON response into out.
// A non-nil in value is JSON-encoded as the request body, which is how the
// daemon receives masked-prefs updates; out may be nil when the response body
// is irrelevant. Any 2xx status counts as success: the write endpoints answer
// with their own codes (200 for PATCH /prefs, 201 for PUT /profiles/, 204 for
// the POST endpoints). The returned error keeps the underlying cause (dial
// errors included) wrapped so callers can keep distinguishing "daemon
// unreachable" from other failures.
func (c *localAPIClient) do(ctx context.Context, method, apiPath string, in, out any) error {
	var requestBody io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode tailscale localapi %s %s request: %w", method, apiPath, err)
		}
		requestBody = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		"http://"+tailscaleLocalAPIHost+tailscaleLocalAPIBasePath+apiPath,
		requestBody,
	)
	if err != nil {
		return fmt.Errorf("build tailscale localapi request: %w", err)
	}
	if in != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.client().Do(request)
	if err != nil {
		return fmt.Errorf("tailscale localapi %s %s: %w", method, apiPath, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(response.Body, tailscaleLocalAPIMaxResponseBytes))
	if err != nil {
		return fmt.Errorf("read tailscale localapi %s response: %w", method, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"tailscale localapi %s %s failed: %s: %s",
			method,
			apiPath,
			response.Status,
			localAPIErrorText(body),
		)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode tailscale localapi %s response: %w: %s", method, err, localAPIErrorText(body))
	}
	return nil
}

func localAPIErrorText(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "<empty response body>"
	}
	runes := []rune(text)
	if len(runes) > tailscaleLocalAPIErrorTextLimit {
		return string(runes[:tailscaleLocalAPIErrorTextLimit]) + "..."
	}
	return text
}
