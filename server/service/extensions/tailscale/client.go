package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"NanoKVM-Server/utils"
)

var errInvalidTailscaleLoginURL = errors.New("invalid tailscale login URL")

const (
	tailscaleLocalAPITimeout = 15 * time.Second

	// Login is interactive, so it gets its own budget instead of the shared
	// per-request timeout. This preserves the login command's 10-minute budget.
	tailscaleLoginTimeout = 10 * time.Minute

	// The daemon publishes AuthURL on GET /status within seconds on the
	// target device, so the poll starts fast and backs off once the URL is
	// clearly not imminent.
	tailscaleLoginInitialPollInterval = 250 * time.Millisecond
	tailscaleLoginInitialPollCount    = 10
	tailscaleLoginPollInterval        = time.Second
)

// Advertised exit nodes are expressed as this route pair in the daemon prefs;
// routes that carry the exit node bit are not valid subnet routes.
const (
	exitNodeRouteIPv4 = "0.0.0.0/0"
	exitNodeRouteIPv6 = "::/0"
)

type LocalClient struct {
	ctx context.Context

	// localAPI stays lazy so the systemd lifecycle methods (Start, Restart,
	// Stop) never build an HTTP client.
	localAPI *localAPIClient

	// loginTimeout overrides tailscaleLoginTimeout; tests shorten it.
	loginTimeout time.Duration

	// commandTimeout overrides tailscaleLocalAPITimeout; tests shorten it.
	commandTimeout time.Duration
}

type TsStatus struct {
	Version      string `json:"Version"`
	BackendState string `json:"BackendState"`
	HaveNodeKey  bool   `json:"HaveNodeKey"`

	// AuthURL is filled in by the daemon while an interactive login is
	// pending, which is what makes the poll in Login work.
	AuthURL string `json:"AuthURL"`

	Self struct {
		HostName     string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
	} `json:"Self"`

	CurrentTailnet struct {
		Name string `json:"Name"`
	} `json:"CurrentTailnet"`
}

type TsPrefs struct {
	AdvertiseRoutes []string `json:"AdvertiseRoutes"`
	LoggedOut       bool     `json:"LoggedOut"`
}

// TsProfile is the non-secret subset of a LocalAPI login profile needed to
// identify and restore the active control-plane identity.
type TsProfile struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	NodeID     string `json:"NodeID"`
	ControlURL string `json:"ControlURL"`
}

// localAPIStartPrefs is the full Prefs subset supplied through
// ipn.Options.UpdatePrefs. Unlike PATCH /prefs this request is not masked.
type localAPIStartPrefs struct {
	ControlURL  string `json:"ControlURL"`
	RouteAll    bool   `json:"RouteAll"`
	CorpDNS     bool   `json:"CorpDNS"`
	WantRunning bool   `json:"WantRunning"`
	Hostname    string `json:"Hostname"`
}

type localAPIStartOptions struct {
	UpdatePrefs localAPIStartPrefs `json:"UpdatePrefs"`
	AuthKey     string             `json:"AuthKey"`
}

func NewLocalClient() *LocalClient {
	return NewLocalClientWithContext(context.Background())
}

func NewLocalClientWithContext(ctx context.Context) *LocalClient {
	if ctx == nil {
		ctx = context.Background()
	}
	return &LocalClient{ctx: ctx}
}

func (c *LocalClient) Start() error {
	return utils.StartService(Tailscaled, true)
}

func (c *LocalClient) Restart() error {
	return utils.RestartService(Tailscaled)
}

func (c *LocalClient) Stop() error {
	return utils.StopService(Tailscaled, true)
}

func (c *LocalClient) Up() error {
	return c.setWantRunning("up", true)
}

func (c *LocalClient) Down() error {
	return c.setWantRunning("down", false)
}

// setWantRunning mirrors `tailscale up` / `tailscale down`: both are masked
// prefs updates of WantRunning and never touch the systemd unit. Login already
// sets accept-dns=false, so a bare up resumes the existing profile without
// restating every non-default route flag.
func (c *LocalClient) setWantRunning(action string, wantRunning bool) error {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	body := localAPIPrefsPatch{
		WantRunning:    &wantRunning,
		WantRunningSet: true,
	}
	if err := c.ensureLocalAPI().do(ctx, http.MethodPatch, tailscaleLocalAPIPrefsPath, body, nil); err != nil {
		return fmt.Errorf("tailscale %s failed: %w", action, err)
	}
	return nil
}

func (c *LocalClient) Status() (*TsStatus, error) {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	return c.status(ctx)
}

func (c *LocalClient) Profiles() ([]TsProfile, error) {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	var profiles []TsProfile
	if err := c.ensureLocalAPI().do(ctx, http.MethodGet, tailscaleLocalAPIProfilesPath, nil, &profiles); err != nil {
		return nil, fmt.Errorf("list tailscale profiles failed: %w", err)
	}
	return profiles, nil
}

func (c *LocalClient) CurrentProfile() (*TsProfile, error) {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	var profile TsProfile
	if err := c.ensureLocalAPI().do(ctx, http.MethodGet, tailscaleLocalAPICurrentProfilePath, nil, &profile); err != nil {
		return nil, fmt.Errorf("get current tailscale profile failed: %w", err)
	}
	return &profile, nil
}

func (c *LocalClient) NewProfile() error {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	if err := c.ensureLocalAPI().do(ctx, http.MethodPut, tailscaleLocalAPIProfilesPath, nil, nil); err != nil {
		return fmt.Errorf("create tailscale profile failed: %w", err)
	}
	return nil
}

func (c *LocalClient) SwitchProfile(profileID string) error {
	if !validTailscaleProfileID(profileID) {
		return errors.New("invalid tailscale profile id")
	}

	ctx, cancel := c.routingCommandContext()
	defer cancel()
	path := tailscaleLocalAPIProfilesPath + profileID
	if err := c.ensureLocalAPI().do(ctx, http.MethodPost, path, nil, nil); err != nil {
		return fmt.Errorf("switch tailscale profile failed: %w", err)
	}
	return nil
}

// StartManaged starts a managed identity. The auth key is carried only in this
// in-memory request body and is never retained by this package after the call
// returns. loginInteractive mirrors `tailscale up`: it is required for a new
// profile with no node key, but must stay false when reconnecting an existing
// profile.
func (c *LocalClient) StartManaged(authKey, controlURL, hostname string, loginInteractive bool) error {
	if strings.TrimSpace(authKey) == "" || strings.TrimSpace(controlURL) == "" || strings.TrimSpace(hostname) == "" {
		return errors.New("invalid managed tailscale start parameters")
	}

	ctx, cancel := c.routingCommandContext()
	defer cancel()
	body := localAPIStartOptions{
		AuthKey: authKey,
		UpdatePrefs: localAPIStartPrefs{
			ControlURL:  controlURL,
			RouteAll:    false,
			CorpDNS:     false,
			WantRunning: true,
			Hostname:    hostname,
		},
	}
	if err := c.ensureLocalAPI().do(ctx, http.MethodPost, tailscaleLocalAPIStartPath, body, nil); err != nil {
		return fmt.Errorf("start managed tailscale profile failed: %w", err)
	}
	// `tailscale up --auth-key=...` performs a Start followed by
	// /login-interactive when the selected profile has no node key yet. The
	// LocalAPI /start endpoint only applies preferences and does not complete
	// that first login by itself, so mirror the CLI's second request here.
	if loginInteractive {
		if err := c.ensureLocalAPI().do(ctx, http.MethodPost, tailscaleLocalAPILoginInteractivePath, nil, nil); err != nil {
			return fmt.Errorf("start managed tailscale login failed: %w", err)
		}
	}
	return nil
}

func validTailscaleProfileID(profileID string) bool {
	if profileID == "" || len(profileID) > 128 {
		return false
	}
	for _, char := range profileID {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// status reads GET /status with the caller's context. Status() bounds it with
// the shared routing timeout; Login() reuses it so its poll loop observes the
// login deadline instead of a per-request one.
func (c *LocalClient) status(ctx context.Context) (*TsStatus, error) {
	var status TsStatus
	if err := c.ensureLocalAPI().do(ctx, http.MethodGet, tailscaleLocalAPIStatusPath, nil, &status); err != nil {
		return nil, fmt.Errorf("tailscale status failed: %w", err)
	}

	return &status, nil
}

// Login mirrors `tailscale login --accept-dns=false` over the LocalAPI: it
// switches to a fresh empty profile, disables MagicDNS for it, makes sure the
// backend is running and then starts an interactive login. The daemon answers
// POST /login-interactive with 204 and publishes the browser URL on
// GET /status.AuthURL, so the URL is polled from there. A login that completes
// without a browser round trip reports Running instead and returns an empty
// URL, matching what the service handler does with it.
func (c *LocalClient) Login() (string, error) {
	ctx, cancel := c.loginContext()
	defer cancel()

	// The new profile is what makes the following login interactive again;
	// the running profile is left untouched until the login completes.
	if err := c.ensureLocalAPI().do(ctx, http.MethodPut, tailscaleLocalAPIProfilesPath, nil, nil); err != nil {
		return "", fmt.Errorf("tailscale login failed: %w", err)
	}

	// --accept-dns=false.
	corpDNS := false
	prefsPatch := localAPIPrefsPatch{
		CorpDNS:    &corpDNS,
		CorpDNSSet: true,
	}
	if err := c.ensureLocalAPI().do(ctx, http.MethodPatch, tailscaleLocalAPIPrefsPath, prefsPatch, nil); err != nil {
		return "", fmt.Errorf("tailscale login failed: %w", err)
	}

	// The CLI starts the backend before starting the interactive login, but
	// NeedsLogin and Running already have it up, so only the stopped states
	// need the extra request.
	status, err := c.status(ctx)
	if err != nil {
		return "", fmt.Errorf("tailscale login failed: %w", err)
	}
	if status.BackendState == "Stopped" || status.BackendState == "NoState" {
		if err := c.ensureLocalAPI().do(ctx, http.MethodPost, tailscaleLocalAPIStartPath, struct{}{}, nil); err != nil {
			return "", fmt.Errorf("tailscale login failed: %w", err)
		}
	}

	if err := c.ensureLocalAPI().do(ctx, http.MethodPost, tailscaleLocalAPILoginInteractivePath, nil, nil); err != nil {
		return "", fmt.Errorf("tailscale login failed: %w", err)
	}

	for attempt := 0; ; attempt++ {
		status, err := c.status(ctx)
		if err != nil {
			return "", fmt.Errorf("tailscale login failed: %w", err)
		}
		if status.AuthURL != "" {
			loginURL, err := validateTailscaleLoginURL(status.AuthURL)
			if err != nil {
				return "", fmt.Errorf("tailscale login failed: %w", err)
			}
			return loginURL, nil
		}
		if status.BackendState == "Running" {
			return "", nil
		}

		wait := tailscaleLoginPollInterval
		if attempt < tailscaleLoginInitialPollCount {
			wait = tailscaleLoginInitialPollInterval
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", fmt.Errorf("tailscale login failed: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// validateTailscaleLoginURL accepts HTTPS login URLs emitted by tailscaled.
func validateTailscaleLoginURL(raw string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" || strings.IndexFunc(candidate, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) != -1 {
		return "", errInvalidTailscaleLoginURL
	}

	parsed, err := url.Parse(candidate)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") ||
		parsed.Hostname() == "" || parsed.User != nil {
		return "", errInvalidTailscaleLoginURL
	}

	return candidate, nil
}

func (c *LocalClient) loginContext() (context.Context, context.CancelFunc) {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.loginTimeout
	if timeout <= 0 {
		timeout = tailscaleLoginTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// Logout mirrors `tailscale logout`: the daemon drops the node key and
// returns to the logged-out state, answering 204 with no body. The request
// carries no payload, so no Content-Type is set.
func (c *LocalClient) Logout() error {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	err := c.ensureLocalAPI().do(ctx, http.MethodPost, tailscaleLocalAPILogoutPath, nil, nil)
	if err == nil {
		return nil
	}

	// tailscaled can persist LoggedOut before POST /logout returns. In that case
	// the request may time out, and a retry answers "no nodekey to log out" even
	// though the requested local security transition already completed. Re-read
	// prefs with a fresh per-command context and accept the authoritative state.
	if prefs, prefsErr := c.Prefs(); prefsErr == nil && prefs != nil && prefs.LoggedOut {
		return nil
	}

	return fmt.Errorf("tailscale logout failed: %w", err)
}

func (c *LocalClient) Prefs() (*TsPrefs, error) {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	var prefs TsPrefs
	if err := c.ensureLocalAPI().do(ctx, http.MethodGet, tailscaleLocalAPIPrefsPath, nil, &prefs); err != nil {
		return nil, fmt.Errorf("tailscale prefs failed: %w", err)
	}
	return &prefs, nil
}

func (c *LocalClient) SetRouting(exitNode bool, routes []string) error {
	ctx, cancel := c.routingCommandContext()
	defer cancel()

	advertised, err := advertiseRoutes(exitNode, routes)
	if err != nil {
		return fmt.Errorf("tailscale set routing failed: %w", err)
	}

	body := localAPIPrefsPatch{
		AdvertiseRoutes:    &advertised,
		AdvertiseRoutesSet: true,
	}
	if err := c.ensureLocalAPI().do(ctx, http.MethodPatch, tailscaleLocalAPIPrefsPath, body, nil); err != nil {
		return fmt.Errorf("tailscale set routing failed: %w", err)
	}
	return nil
}

func (c *LocalClient) routingCommandContext() (context.Context, context.CancelFunc) {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.commandTimeout
	if timeout <= 0 {
		timeout = tailscaleLocalAPITimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func (c *LocalClient) ensureLocalAPI() *localAPIClient {
	if c.localAPI == nil {
		c.localAPI = newLocalAPIClient()
	}
	return c.localAPI
}

// advertiseRoutes builds the AdvertiseRoutes value the daemon expects: an
// advertised exit node is exactly the pair of default routes, followed by the
// validated subnet routes in sorted order. SetRouting always writes the full
// set, so exitNode=false can never leave a default route behind.
func advertiseRoutes(exitNode bool, routes []string) ([]string, error) {
	subnetRoutes, err := normalizeAdvertisedSubnetRoutes(routes)
	if err != nil {
		return nil, err
	}

	advertised := make([]string, 0, len(subnetRoutes)+2)
	if exitNode {
		advertised = append(advertised, exitNodeRouteIPv4, exitNodeRouteIPv6)
	}
	advertised = append(advertised, subnetRoutes...)
	return advertised, nil
}

// normalizeAdvertisedSubnetRoutes validates, deduplicates and sorts the subnet
// routes handed to SetRouting. Default routes are rejected here because the
// exitNode bit is the only supported way to advertise 0.0.0.0/0 and ::/0.
func normalizeAdvertisedSubnetRoutes(routes []string) ([]string, error) {
	normalized := make([]string, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(route))
		if err != nil || prefix != prefix.Masked() || prefix.Bits() == 0 {
			return nil, fmt.Errorf("invalid advertised subnet route %q", route)
		}

		value := prefix.String()
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized, nil
}
