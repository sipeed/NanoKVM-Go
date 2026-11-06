package tailscalegate

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"

	log "github.com/sirupsen/logrus"
)

// Default client-facing budgets: code, heartbeat, and disable requests carry
// a one-second budget while start and exit carry a fifteen-second budget.
const (
	// defaultReadBudget bounds a snapshot refresh and a login-code fetch.
	defaultReadBudget = time.Second
	// defaultActionBudget bounds an initialize or deinitialize action.
	defaultActionBudget = 15 * time.Second
	// defaultDisableBudget bounds the fire-and-forget session teardown.
	defaultDisableBudget = time.Second
)

// Deps are the injectable dependencies of a Handler. The zero value is valid:
// NewHandler fills every unset field with its production default, so a caller
// (or a test) overrides only what it needs.
type Deps struct {
	// Driver is the only external boundary. nil selects the built-in simulated
	// demo driver, which performs no system access; see NewHandler.
	Driver Driver
	// LoadRegion reports the persisted region. It no longer controls Tailscale
	// availability; it is retained because the consent-gated manifest sync is
	// region-scoped, and it feeds SyncFeatureManifest for a missing manifest.
	LoadRegion func() proto.ApplicationRegion
	// LoadFeatureState reads the tri-state server-feature manifest once per
	// request. It is the preferred reader: only FeatureStateEnabled passes the
	// gate, FeatureStateDisabled answers the stable DISABLED message on start,
	// and FeatureStateMissing is repaired by a consent-gated sync before a
	// start initializes. nil falls back to LoadFeatureFlag and then to
	// application.LoadServerFeatureState.
	LoadFeatureState func() application.FeatureState
	// LoadFeatureFlag is the legacy boolean manifest reader, kept for callers
	// that predate the tri-state reader. It is consulted only when
	// LoadFeatureState is nil: true maps to FeatureStateEnabled and false to
	// FeatureStateDisabled.
	LoadFeatureFlag func() bool
	// LoadConsent reports whether the privacy-terms acceptance is already
	// persisted. nil selects the fail-closed default of false, so an un-wired
	// handler never initializes without a recorded agreement.
	LoadConsent func() bool
	// PersistConsent idempotently persists the privacy-terms acceptance before
	// any upstream contact. nil selects a no-op that reports success, so the
	// zero-value/demo handler keeps working without touching device
	// configuration; production wiring binds it to the shared managed service.
	PersistConsent func() error
	// SyncFeatureManifest fetches, validates, and atomically publishes the
	// shared feature manifest for one region. It is called at most once per
	// start, only after consent is resolved, and only when the manifest is
	// missing. nil means "not wired": a start that needs the sync answers FAIL
	// without any network I/O, so the zero-value handler stays hermetic.
	SyncFeatureManifest func(ctx context.Context, region proto.ApplicationRegion) error
	// ReadBudget bounds snapshot and login-code calls.
	ReadBudget time.Duration
	// ActionBudget bounds initialize and deinitialize calls.
	ActionBudget time.Duration
	// DisableBudget bounds the background EndLoginSession call.
	DisableBudget time.Duration
}

// Handler serves LoginCodePath over the root-local state socket.
//
// It owns the retained gate state and the single action slot. Read requests
// (heartbeat and login code) never take the slot, so a slow action cannot
// starve the three-second heartbeat the client sends on its own connection.
type Handler struct {
	driver              Driver
	loadRegion          func() proto.ApplicationRegion
	loadFeatureState    func() application.FeatureState
	loadConsent         func() bool
	persistConsent      func() error
	syncFeatureManifest func(ctx context.Context, region proto.ApplicationRegion) error
	readBudget          time.Duration
	actionBudget        time.Duration
	disableBudget       time.Duration

	gateMu sync.Mutex
	gate   GateState

	actionSlot chan struct{}
}

// NewHandler builds a Handler and fills unset dependencies with production
// defaults. It performs no I/O at construction; the manifest is read once per
// request and, for a consented start on a missing manifest, synced at most
// once.
//
// The tri-state reader resolution is explicit LoadFeatureState, then a wrapper
// over the legacy LoadFeatureFlag, then application.LoadServerFeatureState, so
// existing callers keep compiling while new callers can distinguish a missing
// manifest from one that forbids the feature. SyncFeatureManifest is
// deliberately left nil when unset: a sync-needing start fails closed instead
// of reaching the network from an un-wired handler.
//
// An unset Driver selects the built-in SIMULATED demo driver, which performs
// no system access at all. That default exists for tests and host-side
// walkthroughs: production wiring lives in the state socket server
// (service/ai/state_socket.go), which injects the real managed-network driver
// and fails closed on the not-implemented driver when it is unavailable. The
// demo's initialized state and MockKVM/100.64.0.1 peer are simulated demo
// data, not a real Tailscale connection.
func NewHandler(deps Deps) *Handler {
	if deps.Driver == nil {
		deps.Driver = NewDemoDriver(DemoOptions{})
	}
	if deps.LoadRegion == nil {
		deps.LoadRegion = application.LoadRegion
	}
	if deps.LoadFeatureState == nil {
		if deps.LoadFeatureFlag != nil {
			// Legacy boolean reader: true admits the feature, false behaves like
			// a policy-disabled manifest.
			loadFeatureFlag := deps.LoadFeatureFlag
			deps.LoadFeatureState = func() application.FeatureState {
				if loadFeatureFlag() {
					return application.FeatureStateEnabled
				}
				return application.FeatureStateDisabled
			}
		} else {
			deps.LoadFeatureState = application.LoadServerFeatureState
		}
	}
	if deps.LoadConsent == nil {
		// Fail closed: an un-injected handler reports no persisted consent.
		deps.LoadConsent = func() bool { return false }
	}
	if deps.PersistConsent == nil {
		// The demo/zero-value handler has no consent store to write, so the
		// explicit flag degrades to a no-op success instead of a nil panic.
		deps.PersistConsent = func() error { return nil }
	}
	if deps.ReadBudget <= 0 {
		deps.ReadBudget = defaultReadBudget
	}
	if deps.ActionBudget <= 0 {
		deps.ActionBudget = defaultActionBudget
	}
	if deps.DisableBudget <= 0 {
		deps.DisableBudget = defaultDisableBudget
	}
	return &Handler{
		driver:              deps.Driver,
		loadRegion:          deps.LoadRegion,
		loadFeatureState:    deps.LoadFeatureState,
		loadConsent:         deps.LoadConsent,
		persistConsent:      deps.PersistConsent,
		syncFeatureManifest: deps.SyncFeatureManifest,
		readBudget:          deps.ReadBudget,
		actionBudget:        deps.ActionBudget,
		disableBudget:       deps.DisableBudget,
		actionSlot:          make(chan struct{}, 1),
	}
}

// NewDefaultHandler returns the demo handler: the built-in simulated demo
// driver with the real feature-manifest reader. It is retained for tests and host
// walkthroughs; the production socket server injects the real driver instead.
// It must not be mistaken for real Tailscale behavior: the initialized state
// and MockKVM/100.64.0.1 peer are simulated demo data.
func NewDefaultHandler() *Handler { return NewHandler(Deps{}) }

// observation bundles a successful refresh so a caller can use the resolved
// snapshot for gate decisions and the raw report for the peer identity.
type observation struct {
	report   Report
	snapshot Snapshot
}

// ServeHTTP classifies the request, applies the tri-state feature-manifest
// gate, and dispatches to one of the six behaviors in any region.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d := dispatch(r)
	if d.Status != 0 {
		if d.Status == http.StatusBadRequest {
			writeError(w, http.StatusBadRequest, errorMethodNotSupported)
		} else {
			writeError(w, http.StatusNotFound, errorNotFound)
		}
		return
	}
	if h == nil || h.driver == nil || h.loadFeatureState == nil ||
		h.loadConsent == nil || h.persistConsent == nil {
		writeError(w, http.StatusServiceUnavailable, errorStateUnavailable)
		return
	}
	// Re-read the manifest per request so a disabled switch takes effect at
	// once. Missing and Disabled both close the read lanes; only the start
	// action may run, because it either answers DISABLED or performs the
	// consent-gated sync that can repair a missing manifest. A nil
	// SyncFeatureManifest is a valid state here: it only matters if a start
	// actually needs the sync.
	if state := h.loadFeatureState(); state != application.FeatureStateEnabled && d.Route != routeStart {
		writeError(w, http.StatusNotFound, errorNotFound)
		return
	}

	switch d.Route {
	case routeStart:
		h.handleStart(w, r)
	case routeExit:
		h.handleExit(w, r)
	case routeDisable:
		h.handleDisable(w, r)
	case routeHeartbeat:
		h.handleHeartbeat(w, r)
	case routeCode:
		h.handleCode(w, r)
	default:
		writeError(w, http.StatusNotFound, errorNotFound)
	}
}

// refresh runs one snapshot read under ReadBudget.
//
// On success it replaces the retained gate with a whole-value Confirm and
// returns the resolved snapshot together with the raw report. On failure it
// returns false without touching the retained gate: a failed poll must not
// push a guiding or running page back to Start, and the next successful poll
// can restore the live gate. Failures log a static operation name only.
func (h *Handler) refresh(ctx context.Context) (observation, bool) {
	budgetCtx, cancel := context.WithTimeout(ctx, h.readTimeout())
	defer cancel()

	report, err := h.driver.Snapshot(budgetCtx)
	if err != nil {
		log.Warn("tailscale gate snapshot refresh failed")
		return observation{}, false
	}

	snapshot := Resolve(report)
	h.gateMu.Lock()
	h.gate = h.gate.Confirm(snapshot)
	h.gateMu.Unlock()
	return observation{report: report, snapshot: snapshot}, true
}

// currentGate returns the retained gate state.
func (h *Handler) currentGate() GateState {
	h.gateMu.Lock()
	defer h.gateMu.Unlock()
	return h.gate
}

// acquireAction takes the single action slot without waiting. A failed acquire
// means another action is in flight; the caller answers BUSY instead of
// queueing behind it.
func (h *Handler) acquireAction() bool {
	if h.actionSlot == nil {
		return false
	}
	select {
	case h.actionSlot <- struct{}{}:
		return true
	default:
		return false
	}
}

// releaseAction frees the action slot.
func (h *Handler) releaseAction() {
	if h.actionSlot == nil {
		return
	}
	<-h.actionSlot
}

func (h *Handler) readTimeout() time.Duration {
	if h.readBudget > 0 {
		return h.readBudget
	}
	return defaultReadBudget
}

func (h *Handler) actionTimeout() time.Duration {
	if h.actionBudget > 0 {
		return h.actionBudget
	}
	return defaultActionBudget
}

func (h *Handler) disableTimeout() time.Duration {
	if h.disableBudget > 0 {
		return h.disableBudget
	}
	return defaultDisableBudget
}

// handleHeartbeat answers with the five-field snapshot or the documented 503
// when the whole refresh fails. The 503 is the deliberate exception to the
// HTTP-200 convention: the client must observe a transport failure so its
// sticky rule keeps the last successful snapshot instead of trusting a
// synthesized one.
func (h *Handler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	obs, ok := h.refresh(r.Context())
	if !ok {
		writeError(w, http.StatusServiceUnavailable, errorStateUnavailable)
		return
	}
	log.Debug("tailscale gate heartbeat")
	writeJSON(w, http.StatusOK, newSnapshotResponse(obs.snapshot))
}

// handleStart initializes the component under the tri-state manifest gate.
//
// Ordering is load-bearing:
//
//  0. a confirmed start short-circuits to success (documented idempotency);
//  1. the single action slot (else BUSY);
//  2. a policy-disabled manifest answers DISABLED before anything is read or
//     written, so a switched-off feature never invites a terms agreement and
//     never touches the driver;
//  3. consent resolution: the explicit ?consent=true flag mirrors the web
//     terms-agree ordering (persist consent, then initialize), while a
//     flag-less start only proceeds when an earlier agreement was persisted;
//  4. a missing manifest is repaired by one bounded sync that runs strictly
//     after consent, then re-evaluated: enabled proceeds, disabled answers
//     DISABLED, still missing or a sync failure answers FAIL;
//  5. the driver initializes; 6. a best-effort refresh reports success.
//
// The consent-before-sync order is the whole point of the channel: a start on
// an un-agreed, manifest-less device answers CONSENT and issues no CDN request.
// Only FeatureStateMissing is ever synced; a disabled or corrupt manifest is a
// terminal policy state and is never re-fetched.
//
// Action endpoints answer HTTP 200 even on failure: the client reads the status
// bool and the message string.
func (h *Handler) handleStart(w http.ResponseWriter, r *http.Request) {
	if h.currentGate().Start {
		// Documented idempotency: an already initialized server answers
		// success without calling the driver again. A confirmed gate implies a
		// previously enabled manifest and a recorded agreement, so the consent
		// and manifest work below is redundant for this idempotent start.
		writeJSON(w, http.StatusOK, SuccessAction())
		return
	}
	if !h.acquireAction() {
		writeJSON(w, http.StatusOK, ActionResponse{Message: MessageBusy})
		return
	}
	defer h.releaseAction()

	// A policy-disabled manifest is answered before consent resolution: the
	// device must not persist an agreement for a feature that is switched off,
	// and no fetch or driver call may run.
	if h.loadFeatureState() == application.FeatureStateDisabled {
		writeJSON(w, http.StatusOK, ActionResponse{Message: MessageDisabled})
		return
	}

	// Resolve consent before any upstream contact.
	if queryFlag(r.URL.Query(), "consent") {
		if err := h.persistConsent(); err != nil {
			log.Warn("tailscale gate consent persist failed")
			writeJSON(w, http.StatusOK, ActionResponse{Message: MessageFailure})
			return
		}
	} else if !h.loadConsent() {
		writeJSON(w, http.StatusOK, ActionResponse{Message: MessageConsent})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.actionTimeout())
	defer cancel()

	// Repair a missing manifest exactly once, inside the action budget and
	// strictly after consent. A nil seam means the sync is not wired and fails
	// closed without network I/O.
	if h.loadFeatureState() == application.FeatureStateMissing {
		if h.syncFeatureManifest == nil {
			writeJSON(w, http.StatusOK, ActionResponse{Message: MessageFailure})
			return
		}
		if err := h.syncFeatureManifest(ctx, h.loadRegion()); err != nil {
			log.Warn("tailscale gate feature manifest sync failed")
			writeJSON(w, http.StatusOK, ActionResponse{Message: MessageFailure})
			return
		}
		switch h.loadFeatureState() {
		case application.FeatureStateEnabled:
		case application.FeatureStateDisabled:
			writeJSON(w, http.StatusOK, ActionResponse{Message: MessageDisabled})
			return
		default:
			// The sync reported success but no usable manifest was published:
			// the state is still Missing, so initialization must not proceed.
			writeJSON(w, http.StatusOK, ActionResponse{Message: MessageFailure})
			return
		}
	}

	if err := h.driver.Initialize(ctx); err != nil {
		log.Warn("tailscale gate initialize action failed")
		writeJSON(w, http.StatusOK, FailureAction(err))
		return
	}

	// Re-confirm immediately, but never turn a successful action into a
	// failure when the follow-up read fails; the retained gate is kept.
	h.refresh(r.Context())
	writeJSON(w, http.StatusOK, SuccessAction())
}

// handleExit deinitializes the component once the gate is ready. A not-ready
// gate answers NOT_READY without touching the driver or the slot; the reference
// server accepts ?exit only after start and match are both confirmed.
func (h *Handler) handleExit(w http.ResponseWriter, r *http.Request) {
	gate := h.currentGate()
	if !gate.Start || !gate.Match {
		writeJSON(w, http.StatusOK, ActionResponse{Message: MessageNotReady})
		return
	}
	if !h.acquireAction() {
		writeJSON(w, http.StatusOK, ActionResponse{Message: MessageBusy})
		return
	}
	defer h.releaseAction()

	ctx, cancel := context.WithTimeout(r.Context(), h.actionTimeout())
	defer cancel()
	if err := h.driver.Deinitialize(ctx); err != nil {
		log.Warn("tailscale gate exit action failed")
		writeJSON(w, http.StatusOK, FailureAction(err))
		return
	}

	// Re-confirm immediately, but never turn a successful action into a
	// failure when the follow-up read fails; the retained gate is kept. The
	// reference server keeps reporting the old gate until the component's own
	// report converges, so no optimistic state is written here either.
	h.refresh(r.Context())
	writeJSON(w, http.StatusOK, SuccessAction())
}

// handleDisable ends the client initialization session.
//
// It is fire-and-forget: the compatibility %off% response is written before the
// driver runs, and the driver outcome can never change the response. The
// teardown runs on a background context because the request context ends with
// the response, and it is skipped when an action is already in flight.
func (h *Handler) handleDisable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, codeResponse{Code: OffCode()})

	if !h.acquireAction() {
		return
	}
	budget := h.disableTimeout()
	go func() {
		defer h.releaseAction()
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Debug("tailscale gate disable session panicked")
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), budget)
		defer cancel()
		if err := h.driver.EndLoginSession(ctx); err != nil {
			log.Debug("tailscale gate disable session failed")
			return
		}
		log.Debug("tailscale gate disable session done")
	}()
}

// handleCode returns one of the three code shapes: a plain verification code,
// the pairing completion directive, or the %off% compatibility value.
//
// The directive is built from the raw driver report rather than the resolved
// snapshot: the snapshot hides name and ip while the tunnel is offline, but a
// matched peer must still receive the completion directive, and the reference
// implementation builds the directive from the raw peer values.
func (h *Handler) handleCode(w http.ResponseWriter, r *http.Request) {
	// One read budget covers the whole code request. The client gives a code
	// fetch a single one-second budget, so the optional snapshot refresh and the
	// code fetch share it instead of each consuming its own.
	ctx, cancel := context.WithTimeout(r.Context(), h.readTimeout())
	defer cancel()

	obs, ok := h.refresh(ctx)

	if ok && obs.snapshot.Match {
		name := rawReportField(obs.report.Name, snapshotNameMaxBytes)
		ip := rawReportField(obs.report.IP, snapshotIPMaxBytes)
		directive, err := OnlineDirective(name, ip)
		if err != nil {
			log.Warn("tailscale gate peer identity unavailable")
			writeError(w, http.StatusServiceUnavailable, errorPeerIdentityUnavailable)
			return
		}
		log.Debug("tailscale gate login code directive")
		writeJSON(w, http.StatusOK, codeResponse{Code: directive})
		return
	}

	if !ok && h.currentGate().Match {
		// The client retries code fetches on its own backoff, so a transient
		// refresh failure must not invent a peer identity and must not drop the
		// pairing completion signal by answering a plain code.
		writeError(w, http.StatusServiceUnavailable, errorStateUnavailable)
		return
	}

	if !h.currentGate().Start {
		log.Debug("tailscale gate login code off")
		writeJSON(w, http.StatusOK, codeResponse{Code: OffCode()})
		return
	}

	raw, err := h.driver.LoginCode(ctx)
	if err != nil {
		if errors.Is(err, ErrLoginCodeNotReady) {
			log.Debug("tailscale gate login code not ready")
		} else {
			log.Warn("tailscale gate login code failed")
		}
		writeError(w, http.StatusServiceUnavailable, errorLoginCodeUnavailable)
		return
	}
	log.Debug("tailscale gate login code")
	writeJSON(w, http.StatusOK, codeResponse{Code: PlainCode(raw)})
}

// rawReportField filters one raw driver report string for directive use. A nil
// field becomes empty and fails OnlineDirective validation.
func rawReportField(value *string, limit int) string {
	if value == nil {
		return ""
	}
	return printableASCII(*value, limit)
}
