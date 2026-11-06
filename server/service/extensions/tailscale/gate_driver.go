package tailscale

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/tailscalegate"

	log "github.com/sirupsen/logrus"
)

// This file adapts the shared managed Tailscale service to the tailscalegate
// Driver boundary, so the LCD subpage drives the real Sipeed managed network
// instead of the simulated demo driver. The mapping is the product contract:
//
//   - Initialize   = the web "enable" action (Service.enableManaged).
//   - Deinitialize = the web "stop" action (Service.deinitializeManaged:
//     logout + stop tailscaled + disable autostart).
//   - LoginCode    = the pairing ceremony: first the six-digit claim code,
//     then the three-digit SAS of the single pending join request, which the
//     driver auto-approves once it has actually been displayed.
//   - Snapshot     = gate fields from the persisted managed state (immune to
//     transient LocalAPI hiccups) plus a minimal live probe for tunnel health.
//
// Concurrency: every method may run concurrently with the others (the handler
// serves reads and actions on separate connections). The session mutex is
// never held across network I/O, and all upstream calls run under the
// caller's context or a bounded background one.

const (
	// gateDeinitializeBudget bounds the detached teardown work. The web stop
	// path can exceed the client's 15-second action budget (LocalAPI logout
	// reconcile plus a best-effort cloud deletion with a 40s HTTP timeout),
	// so the work runs detached and the caller only waits for its own budget.
	gateDeinitializeBudget = 90 * time.Second
	// gateApproveDwell is how long the SAS must have been displayed before
	// the driver may auto-approve its pending request. One heartbeat period
	// guarantees at least one real render.
	gateApproveDwell = 3 * time.Second
	// gateApproveRecency bounds how long ago the SAS was last served when the
	// approval fires: an abandoned display is never approved.
	gateApproveRecency = 10 * time.Second
	// gateDevicesProbeBudget bounds one background devices probe.
	gateDevicesProbeBudget = 10 * time.Second
	// gateDevicesProbeMinGap rate-limits the background peer-discovery probe
	// that re-establishes the pairing latch for an already-connected network.
	gateDevicesProbeMinGap = time.Minute
	// gateAutoApproveBudget bounds one auto-approve attempt, including its
	// time-of-check pending re-read.
	gateAutoApproveBudget = 30 * time.Second
)

// errGateAmbiguousPending reports more than one pending join request. The
// driver must never guess which one the LCD is showing, so the code lane
// fails the fetch (warn-level) and approves nothing.
var errGateAmbiguousPending = errors.New("tailscale gate: multiple pending join requests")

// gateMessageError publishes a short stable code as the action failure
// message on the gate wire (tailscalegate.MessageError).
type gateMessageError string

func (e gateMessageError) Error() string { return "tailscale gate action: " + string(e) }

// GateMessage implements tailscalegate.MessageError.
func (e gateMessageError) GateMessage() string { return string(e) }

// gateCodeLaneError classifies pairing upstream failures for the code lane.
// Rate limiting, timeouts, and transport unavailability are transient: the
// client retries on its own backoff and keeps the previous display, so they
// map to ErrLoginCodeNotReady (debug-level). A typed API rejection (an invalid
// identity, a protocol error, a bad gateway) is a real fault and is returned
// as-is so the handler logs it at warn level.
func gateCodeLaneError(err error) error {
	var apiError *sipeedPairingAPIError
	if errors.As(err, &apiError) && apiError.HTTPStatus != http.StatusTooManyRequests {
		return err
	}
	return tailscalegate.ErrLoginCodeNotReady
}

// mapGateActionError maps service-core failures to the gate vocabulary. A
// busy mutation coordinator becomes BUSY, a persisted upstream rate limit
// becomes RATELIMIT, and a missing privacy-terms acceptance becomes CONSENT so
// no path can leak a consent problem as the generic FAIL. Anything else falls
// through to the handler's FAIL fallback (context errors still become TIMEOUT
// there).
func mapGateActionError(err error) error {
	switch {
	case errors.Is(err, errTailscaleMutationBusy):
		return gateMessageError(tailscalegate.MessageBusy)
	case errors.Is(err, errManagedRateLimited):
		return gateMessageError("RATELIMIT")
	case errors.Is(err, errSipeedDisclaimerRequired):
		return gateMessageError(tailscalegate.MessageConsent)
	default:
		return err
	}
}

// gateProbeClient is the minimal LocalAPI subset the tunnel probe needs.
type gateProbeClient interface {
	Status() (*TsStatus, error)
	CurrentProfile() (*TsProfile, error)
}

// GateDriver must always satisfy the gate driver boundary.
var _ tailscalegate.Driver = (*GateDriver)(nil)

// gateTunnelProbe is one live tunnel observation.
type gateTunnelProbe struct {
	online bool
	name   string
	ip     string
}

// GateDriver implements the tailscalegate Driver boundary over the shared
// managed Tailscale service. See the file comment for the mapping.
type GateDriver struct {
	service *Service
	pairing sipeedNetworkPairingClient
	latch   *gateLatchStore

	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	newProbe func(ctx context.Context) gateProbeClient

	deinitBudget       time.Duration
	approveDwell       time.Duration
	approveRecency     time.Duration
	devicesProbeBudget time.Duration
	devicesProbeMinGap time.Duration

	probeMu      sync.Mutex
	probeRunning bool
	probeNextAt  time.Time

	// Pairing session state (the ClientInit code lane). gen increments on
	// every session teardown and invalidates queued auto-approve work. The
	// claim code itself is NOT session state: it lives in the shared manager
	// cache (sipeedManager.claimCode) so the LCD and the web always display
	// the same code.
	mu               sync.Mutex
	gen              uint64
	servedPendingID  string
	firstServedAt    time.Time
	lastServedAt     time.Time
	approveQueuedGen uint64
	approveQueuedID  string
	// memPairedNetwork mirrors the latch file in memory so a transient save
	// failure cannot stall a completed pairing.
	memPairedNetwork string
}

// gateDriverDeps carries the injectable seams. The zero value of every
// optional field selects its production default.
type gateDriverDeps struct {
	service *Service
	pairing sipeedNetworkPairingClient
	latch   *gateLatchStore

	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	newProbe func(ctx context.Context) gateProbeClient

	deinitBudget       time.Duration
	approveDwell       time.Duration
	approveRecency     time.Duration
	devicesProbeBudget time.Duration
	devicesProbeMinGap time.Duration
}

// NewGateDriver builds the production gate driver over the shared service.
// It fails (rather than silently degrading) when the managed mode is not
// wired, so the caller can fall back to a fail-closed handler.
func NewGateDriver(service *Service) (*GateDriver, error) {
	if service == nil || service.sipeed == nil {
		return nil, errors.New("tailscale gate: managed service is unavailable")
	}
	pairing, ok := service.sipeed.deps.pairing.(sipeedNetworkPairingClient)
	if !ok || pairing == nil {
		return nil, errors.New("tailscale gate: managed pairing client is unavailable")
	}
	return newGateDriver(gateDriverDeps{service: service, pairing: pairing}), nil
}

func newGateDriver(deps gateDriverDeps) *GateDriver {
	driver := &GateDriver{
		service:            deps.service,
		pairing:            deps.pairing,
		latch:              deps.latch,
		now:                deps.now,
		wait:               deps.wait,
		newProbe:           deps.newProbe,
		deinitBudget:       deps.deinitBudget,
		approveDwell:       deps.approveDwell,
		approveRecency:     deps.approveRecency,
		devicesProbeBudget: deps.devicesProbeBudget,
		devicesProbeMinGap: deps.devicesProbeMinGap,
	}
	if driver.latch == nil {
		driver.latch = newGateLatchStore("")
	}
	if driver.now == nil {
		driver.now = time.Now
	}
	if driver.wait == nil {
		driver.wait = waitSipeedDuration
	}
	if driver.newProbe == nil {
		driver.newProbe = func(ctx context.Context) gateProbeClient {
			return NewLocalClientWithContext(ctx)
		}
	}
	if driver.deinitBudget <= 0 {
		driver.deinitBudget = gateDeinitializeBudget
	}
	if driver.approveDwell <= 0 {
		driver.approveDwell = gateApproveDwell
	}
	if driver.approveRecency <= 0 {
		driver.approveRecency = gateApproveRecency
	}
	if driver.devicesProbeBudget <= 0 {
		driver.devicesProbeBudget = gateDevicesProbeBudget
	}
	if driver.devicesProbeMinGap <= 0 {
		driver.devicesProbeMinGap = gateDevicesProbeMinGap
	}
	return driver
}

// gateStarted maps the persisted managed state to the initialization gate.
// Only real transitions reach the persisted state, so a transient live
// degradation (a lost LocalAPI read or a tunnel hiccup) can never push the
// gate back to Start, while a failed or rolled-back enable can.
func gateStarted(state proto.TailscaleManagedState) bool {
	switch state {
	case proto.TailscaleManagedEnrolling,
		proto.TailscaleManagedEnabledConnecting,
		proto.TailscaleManagedEnabledConnected:
		return true
	default:
		return false
	}
}

// Snapshot reports the gate and tunnel state. The gate fields are a pure
// function of the persisted managed state plus the pairing latch; only the
// tunnel health and identity come from a live probe, and a probe failure
// degrades them to offline without touching the gates. The raw report always
// carries the last-known name/ip so the handler can still build the pairing
// completion directive while the tunnel is offline.
func (d *GateDriver) Snapshot(ctx context.Context) (tailscalegate.Report, error) {
	if err := ctx.Err(); err != nil {
		return tailscalegate.Report{}, err
	}
	if d == nil || d.service == nil || d.service.sipeed == nil {
		return tailscalegate.Report{}, errManagedModeUnavailable
	}

	state := d.service.sipeed.snapshot()
	start := gateStarted(state.ManagedState)
	connected := state.ManagedState == proto.TailscaleManagedEnabledConnected

	online := false
	name := state.Hostname
	ip := state.IP
	if connected {
		if probe := d.probeTunnel(ctx); probe.online {
			online = true
			if probe.name != "" {
				name = probe.name
			}
			ip = probe.ip
		}
	}

	match := connected && d.pairedFor(state.NetworkID)
	if connected && !match {
		d.scheduleDevicesProbe(state.NetworkID)
	}

	status := string(tailscalegate.StatusOffline)
	if online {
		status = string(tailscalegate.StatusOnline)
	}
	return tailscalegate.Report{
		Status: &status,
		Start:  &start,
		Match:  &match,
		Name:   &name,
		IP:     &ip,
	}, nil
}

// probeTunnel performs the minimal live check for tunnel health: daemon
// running with an IPv4 address on a profile that belongs to the managed
// control server. Any failure is an offline observation, never an error.
func (d *GateDriver) probeTunnel(ctx context.Context) gateTunnelProbe {
	if d.newProbe == nil {
		return gateTunnelProbe{}
	}
	client := d.newProbe(ctx)
	if client == nil {
		return gateTunnelProbe{}
	}
	status, err := client.Status()
	if err != nil || status == nil || status.BackendState != "Running" {
		return gateTunnelProbe{}
	}
	ipv4 := tailscaleStatusIPv4(status)
	if ipv4 == "" {
		return gateTunnelProbe{}
	}
	profile, err := client.CurrentProfile()
	if err != nil || profile == nil || profile.ControlURL != sipeedControlURL {
		return gateTunnelProbe{}
	}
	return gateTunnelProbe{online: true, name: strings.TrimSpace(status.Self.HostName), ip: ipv4}
}

// pairedFor reports whether networkID has a proven paired peer, checking the
// in-memory mirror before the persisted latch.
func (d *GateDriver) pairedFor(networkID string) bool {
	if networkID == "" {
		return false
	}
	d.mu.Lock()
	mem := d.memPairedNetwork
	d.mu.Unlock()
	if mem == networkID {
		return true
	}
	return d.latch.paired(networkID)
}

// markPaired records a completed peer pairing for networkID.
func (d *GateDriver) markPaired(networkID string) {
	if networkID == "" {
		return
	}
	d.mu.Lock()
	d.memPairedNetwork = networkID
	d.mu.Unlock()
	if err := d.latch.mark(networkID, d.now()); err != nil {
		log.Warn("tailscale gate pairing latch save failed")
	}
}

// scheduleDevicesProbe starts the single-flight background peer-discovery
// probe. It re-establishes the pairing latch when the network is already
// connected and paired but the latch is missing (process restart, first run
// after an upgrade). It never runs on the request goroutine's budget and
// respects the persisted upstream Retry-After.
func (d *GateDriver) scheduleDevicesProbe(networkID string) {
	if networkID == "" {
		return
	}
	d.probeMu.Lock()
	if d.probeRunning || d.now().Before(d.probeNextAt) {
		d.probeMu.Unlock()
		return
	}
	d.probeRunning = true
	d.probeMu.Unlock()

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Debug("tailscale gate devices probe panicked")
			}
			d.probeMu.Lock()
			d.probeRunning = false
			d.probeNextAt = d.now().Add(d.devicesProbeMinGap)
			d.probeMu.Unlock()
		}()

		if d.service.sipeed.pairingRetryRemaining() > 0 {
			return
		}
		token, _, err := d.service.sipeed.pairingCredentials()
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), d.devicesProbeBudget)
		defer cancel()
		devices, err := d.pairing.Devices(ctx, token)
		if err != nil {
			d.persistPairingRetryAfter(err)
			log.Debug("tailscale gate devices probe failed")
			return
		}
		for _, device := range devices.Devices {
			if !device.IsSelf {
				d.markPaired(networkID)
				return
			}
		}
	}()
}

// persistPairingRetryAfter mirrors the management handlers: an upstream 429
// is persisted so no gate lane can hammer the pairing service inside the
// Retry-After window, even across a process restart.
func (d *GateDriver) persistPairingRetryAfter(err error) {
	var apiError *sipeedPairingAPIError
	if !errors.As(err, &apiError) || apiError.HTTPStatus != http.StatusTooManyRequests {
		return
	}
	if saveErr := d.service.sipeed.setPairingRetryAfter(apiError.RetryAfter); saveErr != nil {
		log.Warn("tailscale gate retry-after save failed")
	}
}

// Initialize starts the managed network, exactly as the web enable action
// does. It is idempotent for an already initialized or in-flight enable, and
// it returns as soon as the asynchronous enable job is launched: stage
// progression is then driven by heartbeat snapshots.
func (d *GateDriver) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.service == nil || d.service.sipeed == nil {
		return errManagedModeUnavailable
	}
	if gateStarted(d.service.sipeed.snapshot().ManagedState) {
		return nil
	}
	if err := d.service.enableManaged(false); err != nil {
		return mapGateActionError(err)
	}
	return nil
}

// Deinitialize performs the web stop semantics: managed logout (which also
// best-effort destroys the cloud network) followed by stopping tailscaled and
// disabling its autostart.
//
// The teardown runs on a detached, generously bounded context so the
// client-side action timeout cannot cancel it half-applied; this call waits
// only for its own budget and reports the context error (TIMEOUT on the wire)
// when the work outlasts it. A retry while the teardown still holds the
// mutation lock answers BUSY.
func (d *GateDriver) Deinitialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.service == nil {
		return errManagedModeUnavailable
	}

	// The teardown runs on a detached, generously bounded context so the
	// client-side action timeout cannot cancel it half-applied; this call
	// waits only for its own budget and reports the context error (TIMEOUT on
	// the wire) when the work outlasts it. The worker owns cancel: a caller
	// timeout never reaches the teardown, and the 90s budget bounds the leak.
	workCtx, cancel := context.WithTimeout(context.Background(), d.deinitBudget)
	done := make(chan error, 1)
	go func() {
		defer cancel()
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Warn("tailscale gate deinitialize panicked")
				done <- errors.New("tailscale gate: deinitialize panicked")
			}
		}()
		done <- d.service.deinitializeManaged(workCtx)
	}()

	select {
	case err := <-done:
		if err == nil {
			d.clearPairingState()
		}
		return mapGateActionError(err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// clearPairingState drops every pairing artifact after a successful
// deinitialize: the session, the in-memory latch, and the persisted latch
// (best effort — the latch is keyed by network id, so a cleared identity
// invalidates it anyway).
func (d *GateDriver) clearPairingState() {
	// The shared claim-code cache is dropped by the identity clear inside the
	// teardown itself; only the gate session artifacts are reset here.
	d.mu.Lock()
	d.gen++
	d.servedPendingID = ""
	d.firstServedAt = time.Time{}
	d.lastServedAt = time.Time{}
	d.memPairedNetwork = ""
	d.mu.Unlock()
	if err := d.latch.clear(); err != nil {
		log.Debug("tailscale gate pairing latch removal failed")
	}
}

// LoginCode serves the pairing ceremony codes. The session order is: the
// six-digit claim code while nobody is joining, then the three-digit SAS of
// the single pending join request. Serving the SAS is what arms its
// auto-approval.
//
// A transiently unavailable code (identity not enrolled yet, Retry-After
// active, flaky upstream) answers tailscalegate.ErrLoginCodeNotReady, which
// the handler turns into a 503 the client retries on its own backoff.
func (d *GateDriver) LoginCode(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if d == nil || d.service == nil || d.service.sipeed == nil {
		return "", errManagedModeUnavailable
	}
	state := d.service.sipeed.snapshot()
	if state.DeviceToken == "" || state.NetworkID == "" {
		// The enable job has not enrolled yet; there is nothing to claim.
		return "", tailscalegate.ErrLoginCodeNotReady
	}
	if d.service.sipeed.pairingRetryRemaining() > 0 {
		return "", tailscalegate.ErrLoginCodeNotReady
	}

	pending, err := d.pairing.Pending(ctx, state.DeviceToken, 0)
	if err != nil {
		d.persistPairingRetryAfter(err)
		return "", gateCodeLaneError(err)
	}
	switch len(pending.Pending) {
	case 0:
		d.clearServedPending()
		return d.serveClaimCode(ctx)
	case 1:
		item := pending.Pending[0]
		sas := strings.Join(item.SAS, "")
		d.noteServedPending(item.PendingID)
		return sas, nil
	default:
		d.clearServedPending()
		return "", errGateAmbiguousPending
	}
}

// serveClaimCode returns the shared claim code held by the managed service
// (sipeedManager.claimCode), so the LCD and the web always display the same
// code and one lane's renewal can never invalidate the other lane's display.
// The code leaves as bare digits: the LCD renders a six-digit code with its
// own "3 + space + 3" rule, matching the web display.
func (d *GateDriver) serveClaimCode(ctx context.Context) (string, error) {
	response, err := d.service.sipeed.claimCode(ctx)
	if err != nil {
		// A malformed upstream response is a contract violation, not a
		// retryable not-ready condition.
		if errors.Is(err, errSipeedClaimCodeProtocol) {
			return "", errors.New("tailscale gate: invalid claim code from pairing service")
		}
		return "", gateCodeLaneError(err)
	}
	digits, err := normalizeSipeedClaimCode(response.Code)
	if err != nil {
		return "", errors.New("tailscale gate: invalid claim code from pairing service")
	}
	return digits, nil
}

// noteServedPending records that the LCD code lane is displaying the SAS of
// pendingID and queues its auto-approval (one goroutine per pending per
// session generation).
func (d *GateDriver) noteServedPending(pendingID string) {
	d.mu.Lock()
	now := d.now()
	if d.servedPendingID != pendingID {
		d.servedPendingID = pendingID
		d.firstServedAt = now
	}
	d.lastServedAt = now
	gen := d.gen
	if d.approveQueuedGen != gen || d.approveQueuedID != pendingID {
		d.approveQueuedGen = gen
		d.approveQueuedID = pendingID
		d.mu.Unlock()
		go d.autoApprove(gen, pendingID)
		return
	}
	d.mu.Unlock()
}

// clearServedPending drops the displayed-pending record when the pending list
// no longer contains it, so a vanished request can never be approved. The
// approve watermark is reset too: if the same pending reappears, its next
// serve queues a fresh evaluation.
func (d *GateDriver) clearServedPending() {
	d.mu.Lock()
	d.servedPendingID = ""
	d.firstServedAt = time.Time{}
	d.lastServedAt = time.Time{}
	d.approveQueuedID = ""
	d.mu.Unlock()
}

// autoApprove approves pendingID after the dwell, only while every safety
// rule still holds: same session generation, same pending still being
// displayed (recently served), no active Retry-After, and a time-of-check
// re-read that still shows exactly this one pending request.
func (d *GateDriver) autoApprove(gen uint64, pendingID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Debug("tailscale gate auto-approve panicked")
		}
	}()
	defer func() {
		// Re-arm: whether this attempt approved, refused, or aborted, a later
		// serve of the same pending queues a fresh evaluation.
		d.mu.Lock()
		if d.approveQueuedGen == gen && d.approveQueuedID == pendingID {
			d.approveQueuedID = ""
		}
		d.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), gateAutoApproveBudget)
	defer cancel()
	if err := d.wait(ctx, d.approveDwell); err != nil {
		return
	}

	d.mu.Lock()
	alive := d.gen == gen && d.servedPendingID == pendingID &&
		!d.firstServedAt.IsZero() &&
		d.now().Sub(d.firstServedAt) >= d.approveDwell &&
		d.now().Sub(d.lastServedAt) <= d.approveRecency
	d.mu.Unlock()
	if !alive {
		return
	}
	if d.service.sipeed.pairingRetryRemaining() > 0 {
		return
	}
	token, _, err := d.service.sipeed.pairingCredentials()
	if err != nil {
		return
	}

	// Time-of-check re-read: approve only when the same request is still the
	// single pending one.
	pending, err := d.pairing.Pending(ctx, token, 0)
	if err != nil {
		d.persistPairingRetryAfter(err)
		return
	}
	if len(pending.Pending) != 1 || pending.Pending[0].PendingID != pendingID {
		return
	}

	// Capture the shared claim-code entry before the mutation: the approval
	// consumes the code it presents, so the cache must be dropped afterwards.
	// The invalidator snapshots the current entry and clears it only while the
	// pointer and token are unchanged, so a newer code issued while the
	// approval was in flight survives. Capturing after the mutation would
	// snapshot that newer entry and wrongly drop it.
	invalidateClaim := d.service.sipeed.claimCodeInvalidator(token)
	if _, err := d.pairing.ApprovePending(ctx, token, pendingID, ""); err != nil {
		d.persistPairingRetryAfter(err)
		log.Warn("tailscale gate auto-approve failed")
		return
	}
	// Fire unconditionally on success: when no code was cached the invalidator
	// is a no-op, and a failed approval must leave any cached code untouched.
	invalidateClaim()

	d.markPaired(d.service.sipeed.snapshot().NetworkID)
	// The pending id is validated to [A-Za-z0-9_-] by the pairing layer, so it
	// is safe to audit; the SAS, claim code, and tokens are never logged.
	log.Infof("tailscale gate auto-approved pending %s", pendingID)
}

// EndLoginSession ends the pairing session only: the displayed-pending record
// is dropped and the generation bump cancels any queued auto-approval. It
// never touches the initialized gate, the managed identity, or the shared
// claim-code cache: the code issued upstream stays valid for every lane, so
// ending the LCD session must not invalidate a code the web is displaying.
func (d *GateDriver) EndLoginSession(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.gen++
	d.servedPendingID = ""
	d.firstServedAt = time.Time{}
	d.lastServedAt = time.Time{}
	return nil
}
