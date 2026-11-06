package tailscale

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	log "github.com/sirupsen/logrus"
)

const (
	sipeedEnableTimeout           = 8 * time.Minute
	sipeedConnectTimeout          = 90 * time.Second
	sipeedConnectPoll             = time.Second
	sipeedLogoutReconcileAttempts = 5
	sipeedLogoutReconcileInterval = time.Second
	// sipeedClaimCodeRenewSkew renews the shared single-use claim code shortly
	// before its upstream expiry instead of serving a code that is about to
	// die. Every transport (the web handlers and the root-local gate) shares
	// this threshold, so a code is either valid for both lanes or renewed for
	// both.
	sipeedClaimCodeRenewSkew = 20 * time.Second
)

var (
	// errSipeedManagedIdentityUnavailable reports a claim-code request without
	// a usable managed identity. Transports map it to their own
	// identity-missing failure.
	errSipeedManagedIdentityUnavailable = errors.New("managed tailscale identity is unavailable")
	// errSipeedManagedPairingUnavailable reports a manager whose pairing
	// client does not implement the device-management contract.
	errSipeedManagedPairingUnavailable = errors.New("managed pairing service is unavailable")
	// errSipeedClaimCodeProtocol marks an upstream claim-code response that
	// violates the contract (bad code shape or expiry). It is a real fault,
	// never a retryable not-ready signal.
	errSipeedClaimCodeProtocol = errors.New("pairing service returned an invalid claim code response")
)

type sipeedManagedLocalClient interface {
	Start() error
	Status() (*TsStatus, error)
	Profiles() ([]TsProfile, error)
	CurrentProfile() (*TsProfile, error)
	NewProfile() error
	SwitchProfile(profileID string) error
	StartManaged(authKey, controlURL, hostname string, loginInteractive bool) error
	Logout() error
}

type sipeedManagerDeps struct {
	store          *sipeedStateStore
	pairing        sipeedPairingClient
	newLocalClient func(context.Context) sipeedManagedLocalClient
	isInstalled    func() bool
	install        func() error
	hostname       func() (string, error)
	now            func() time.Time
	wait           func(context.Context, time.Duration) error
	connectTimeout time.Duration
	connectPoll    time.Duration
}

func defaultSipeedManagerDeps() sipeedManagerDeps {
	return sipeedManagerDeps{
		store:   newSipeedStateStore(defaultSipeedStatePath),
		pairing: newHTTPSipeedPairingClient(sipeedControlURL, nil),
		newLocalClient: func(ctx context.Context) sipeedManagedLocalClient {
			return NewLocalClientWithContext(ctx)
		},
		isInstalled:    isInstalled,
		install:        install,
		hostname:       os.Hostname,
		now:            time.Now,
		wait:           waitSipeedDuration,
		connectTimeout: sipeedConnectTimeout,
		connectPoll:    sipeedConnectPoll,
	}
}

type sipeedManager struct {
	mu       sync.Mutex
	state    sipeedPersistedState
	storeErr error
	deps     sipeedManagerDeps

	// claimMu serializes the claim-code check-and-fetch so concurrent lanes
	// share one upstream request (single-flight). It is never held by code
	// that also holds mu, and mu is never held across the upstream call:
	// the lock order is always claimMu -> mu.
	claimMu sync.Mutex
	claim   *sipeedClaimCodeCacheEntry
}

// sipeedClaimCodeCacheEntry is the shared one-time claim code. The issuing
// device token is stored with the entry and re-checked on every serve, so a
// stale code can never outlive its identity even if an explicit invalidation
// site is missed.
type sipeedClaimCodeCacheEntry struct {
	token     string
	code      string
	expiresAt time.Time
	singleUse bool
}

func newSipeedManager(deps sipeedManagerDeps) *sipeedManager {
	manager := &sipeedManager{deps: normalizeSipeedManagerDeps(deps)}
	state, err := manager.deps.store.load()
	if err != nil {
		manager.state = newDisabledSipeedState()
		manager.state.ManagedState = proto.TailscaleManagedDegraded
		manager.state.ManagedError = "stateFileInvalid"
		manager.storeErr = err
		return manager
	}
	manager.state = state
	if state.ManagedState == proto.TailscaleManagedEnrolling ||
		state.ManagedState == proto.TailscaleManagedEnabledConnecting {
		manager.state.ManagedState = proto.TailscaleManagedDegraded
		manager.state.ManagedError = "interrupted"
		if err := manager.deps.store.save(manager.state); err != nil {
			manager.storeErr = err
			manager.state.ManagedError = "stateWriteFailed"
		}
	}
	return manager
}

func normalizeSipeedManagerDeps(deps sipeedManagerDeps) sipeedManagerDeps {
	defaults := defaultSipeedManagerDeps()
	if deps.store == nil {
		deps.store = defaults.store
	}
	if deps.pairing == nil {
		deps.pairing = defaults.pairing
	}
	if deps.newLocalClient == nil {
		deps.newLocalClient = defaults.newLocalClient
	}
	if deps.isInstalled == nil {
		deps.isInstalled = defaults.isInstalled
	}
	if deps.install == nil {
		deps.install = defaults.install
	}
	if deps.hostname == nil {
		deps.hostname = defaults.hostname
	}
	if deps.now == nil {
		deps.now = defaults.now
	}
	if deps.wait == nil {
		deps.wait = defaults.wait
	}
	if deps.connectTimeout <= 0 {
		deps.connectTimeout = defaults.connectTimeout
	}
	if deps.connectPoll <= 0 {
		deps.connectPoll = defaults.connectPoll
	}
	return deps
}

func waitSipeedDuration(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// errManagedRateLimited marks a prepareEnable rejection caused by the
// persisted upstream Retry-After deadline. Transports with their own failure
// vocabulary (the root-local gate) map it to a short stable code.
var errManagedRateLimited = errors.New("managed operation is rate limited")

func (manager *sipeedManager) prepareEnable() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.storeErr != nil {
		return errors.New("managed state is unavailable")
	}
	if deadline := manager.retryDeadlineLocked(); !deadline.IsZero() && manager.deps.now().Before(deadline) {
		return errManagedRateLimited
	}
	manager.state.ManagedState = proto.TailscaleManagedEnrolling
	manager.state.ManagedError = ""
	manager.state.RetryAfter = ""
	// Rollback applies only to the current attempt, not a saved official login.
	manager.state.PreviousProfileID = ""
	// A new enrollment cycle may replace the network; never serve a code that
	// was issued for the previous cycle.
	manager.claim = nil
	return manager.saveLocked()
}

func (manager *sipeedManager) runEnable(release func()) {
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), sipeedEnableTimeout)
	defer cancel()
	defer func() {
		if recover() != nil {
			manager.fail("internalError", false, nil)
		}
	}()
	manager.enable(ctx)
}

func (manager *sipeedManager) enable(ctx context.Context) {
	if !manager.deps.isInstalled() {
		if err := manager.deps.install(); err != nil {
			manager.fail("installFailed", false, nil)
			return
		}
	}

	client := manager.deps.newLocalClient(ctx)
	status, err := manager.daemonStatus(ctx, client)
	if err != nil {
		manager.fail("localAPIUnavailable", false, nil)
		return
	}

	currentProfile, err := client.CurrentProfile()
	if err != nil {
		manager.fail("profileReadFailed", false, nil)
		return
	}
	state := manager.snapshot()
	currentIsSipeed := currentProfile.ControlURL == sipeedControlURL
	loginInteractive := !status.HaveNodeKey
	if !currentIsSipeed && (currentProfile.NodeID != "" || status.BackendState == "Running") {
		if err := client.Logout(); err != nil {
			manager.fail("profileSetupFailed", false, nil)
			return
		}
		loginInteractive = true
		currentProfile, err = client.CurrentProfile()
		if err != nil {
			manager.fail("profileReadFailed", false, nil)
			return
		}
	}

	hostname, err := manager.managedHostname(status)
	if err != nil {
		manager.fail("hostnameUnavailable", false, nil)
		return
	}

	if currentIsSipeed && status.BackendState == "Running" {
		if ipv4 := tailscaleStatusIPv4(status); ipv4 != "" && state.DeviceToken != "" &&
			(state.Phase == sipeedPhaseConnecting || state.Phase == sipeedPhaseReporting ||
				state.Phase == sipeedPhaseComplete) {
			manager.persistConnectedProfile(currentProfile, status, ipv4, "", sipeedPhaseReporting)
			if nodeID, ok := manager.reportSelfNode(ctx, client, currentProfile, ipv4); ok {
				manager.persistConnectedProfile(currentProfile, status, ipv4, nodeID, sipeedPhaseComplete)
			}
			return
		}
	}

	identity, ok := manager.ensureIdentity(hostname)
	if !ok {
		return
	}

	// Enroll is reserved for the first identity creation. Once the local
	// identity has a network, renew a one-shot key through join-key instead;
	// retrying enroll would consume its separate rate limit and could create a
	// second network after a transient LocalAPI failure.
	var networkID string
	var authKey string
	if identity.NetworkID == "" {
		enrollment, err := manager.deps.pairing.Enroll(
			ctx,
			identity.DeviceID,
			identity.DeviceToken,
			hostname,
		)
		if err != nil {
			manager.failPairing(err, false, client, managedProfileForInvalidation(currentProfile))
			return
		}
		if enrollment == nil {
			manager.fail("protocolError", false, nil)
			return
		}
		networkID = enrollment.NetworkID
		authKey = enrollment.AuthKey
		enrollment.AuthKey = ""
	} else {
		joinKey, err := manager.deps.pairing.JoinKey(ctx, identity.DeviceToken)
		if err != nil {
			manager.failPairing(err, false, client, managedProfileForInvalidation(currentProfile))
			return
		}
		if joinKey == nil {
			manager.fail("protocolError", false, nil)
			return
		}
		networkID = identity.NetworkID
		authKey = joinKey.AuthKey
		joinKey.AuthKey = ""
	}
	if !validSipeedStateString(networkID, 256, false) || strings.TrimSpace(authKey) == "" {
		authKey = ""
		manager.fail("protocolError", false, nil)
		return
	}
	if !manager.persistEnrollment(networkID) {
		authKey = ""
		return
	}
	if !currentIsSipeed && currentProfile.ID != "" &&
		(currentProfile.NodeID != "" || state.PreviousProfileID == "") &&
		!manager.persistPreviousProfile(currentProfile.ID) {
		authKey = ""
		return
	}

	previousProfileID, profileChanged, err := manager.selectManagedProfile(client, currentProfile)
	if err != nil {
		authKey = ""
		manager.fail("profileSetupFailed", profileChanged, client)
		return
	}
	rollbackNeeded := profileChanged || previousProfileID != ""
	if !manager.persistConnecting(previousProfileID) {
		authKey = ""
		manager.fail("stateWriteFailed", rollbackNeeded, client)
		return
	}

	err = client.StartManaged(authKey, sipeedControlURL, hostname, loginInteractive)
	authKey = ""
	if err != nil {
		manager.fail("profileStartFailed", rollbackNeeded, client)
		return
	}

	connectedStatus, ipv4, err := manager.waitConnected(ctx, client)
	if err != nil {
		manager.fail("connectionFailed", rollbackNeeded, client)
		return
	}
	managedProfile, err := client.CurrentProfile()
	if err != nil || managedProfile.ControlURL != sipeedControlURL {
		manager.fail("profileVerificationFailed", rollbackNeeded, client)
		return
	}
	manager.persistConnectedProfile(managedProfile, connectedStatus, ipv4, "", sipeedPhaseReporting)
	nodeID, ok := manager.reportSelfNode(ctx, client, managedProfile, ipv4)
	if !ok {
		return
	}
	manager.persistConnectedProfile(managedProfile, connectedStatus, ipv4, nodeID, sipeedPhaseComplete)
}

func (manager *sipeedManager) daemonStatus(
	ctx context.Context,
	client sipeedManagedLocalClient,
) (*TsStatus, error) {
	status, err := client.Status()
	if err == nil {
		return status, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := client.Start(); err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 10; attempt++ {
		if err := manager.deps.wait(ctx, time.Second); err != nil {
			return nil, err
		}
		status, err = client.Status()
		if err == nil {
			return status, nil
		}
	}
	return nil, err
}

func (manager *sipeedManager) managedHostname(status *TsStatus) (string, error) {
	hostname, err := manager.deps.hostname()
	hostname = strings.TrimSpace(hostname)
	if err != nil || hostname == "" {
		if status != nil {
			hostname = strings.TrimSpace(status.Self.HostName)
		}
	}
	if hostname == "" || !validSipeedStateString(hostname, 255, false) {
		return "", errors.New("hostname is unavailable")
	}
	return hostname, nil
}

func (manager *sipeedManager) ensureIdentity(hostname string) (sipeedPersistedState, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.state.DeviceID == "" || manager.state.DeviceToken == "" {
		deviceID, err := newSipeedCredential(sipeedDeviceIDBytes)
		if err != nil {
			manager.failLocked("identityGenerationFailed")
			return manager.state, false
		}
		deviceToken, err := newSipeedCredential(sipeedDeviceTokenBytes)
		if err != nil {
			manager.failLocked("identityGenerationFailed")
			return manager.state, false
		}
		manager.state.DeviceID = deviceID
		manager.state.DeviceToken = deviceToken
		manager.state.NetworkID = ""
		manager.state.NodeID = ""
	}
	manager.state.Hostname = hostname
	manager.state.Phase = sipeedPhaseIdentity
	manager.state.ManagedState = proto.TailscaleManagedEnrolling
	manager.state.ManagedError = ""
	if err := manager.saveLocked(); err != nil {
		return manager.state, false
	}
	return manager.state, true
}

func (manager *sipeedManager) persistEnrollment(networkID string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.state.NetworkID = networkID
	manager.state.Phase = sipeedPhaseEnrolled
	manager.state.ManagedState = proto.TailscaleManagedEnrolling
	manager.state.ManagedError = ""
	return manager.saveLocked() == nil
}

func (manager *sipeedManager) selectManagedProfile(
	client sipeedManagedLocalClient,
	current *TsProfile,
) (previousProfileID string, changed bool, err error) {
	state := manager.snapshot()
	if current.ControlURL == sipeedControlURL {
		return state.PreviousProfileID, false, nil
	}
	previousProfileID = current.ID
	if state.PreviousProfileID != "" {
		previousProfileID = state.PreviousProfileID
	}
	profiles, err := client.Profiles()
	if err != nil {
		return previousProfileID, false, err
	}

	managedProfileID := ""
	for _, profile := range profiles {
		if profile.ControlURL != sipeedControlURL {
			continue
		}
		if profile.ID == state.ProfileID {
			managedProfileID = profile.ID
			break
		}
		if managedProfileID == "" {
			managedProfileID = profile.ID
		}
	}
	if managedProfileID != "" {
		if err := client.SwitchProfile(managedProfileID); err != nil {
			return previousProfileID, true, err
		}
		return previousProfileID, true, nil
	}
	if err := client.NewProfile(); err != nil {
		return previousProfileID, true, err
	}
	return previousProfileID, true, nil
}

func (manager *sipeedManager) persistPreviousProfile(profileID string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.state.PreviousProfileID = profileID
	return manager.saveLocked() == nil
}

func (manager *sipeedManager) persistConnecting(previousProfileID string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if previousProfileID != "" {
		manager.state.PreviousProfileID = previousProfileID
	}
	manager.state.Phase = sipeedPhaseConnecting
	manager.state.ManagedState = proto.TailscaleManagedEnabledConnecting
	manager.state.ManagedError = ""
	return manager.saveLocked() == nil
}

func (manager *sipeedManager) waitConnected(
	ctx context.Context,
	client sipeedManagedLocalClient,
) (*TsStatus, string, error) {
	deadline := manager.deps.now().Add(manager.deps.connectTimeout)
	for {
		status, err := client.Status()
		if err == nil && status.BackendState == "Running" {
			if ipv4 := tailscaleStatusIPv4(status); ipv4 != "" {
				return status, ipv4, nil
			}
		}
		if !manager.deps.now().Before(deadline) {
			return nil, "", errors.New("managed tailscale connection timed out")
		}
		if err := manager.deps.wait(ctx, manager.deps.connectPoll); err != nil {
			return nil, "", err
		}
	}
}

func tailscaleStatusIPv4(status *TsStatus) string {
	if status == nil {
		return ""
	}
	for _, value := range status.Self.TailscaleIPs {
		ip := net.ParseIP(value)
		if ip != nil && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

func (manager *sipeedManager) persistConnectedProfile(
	profile *TsProfile,
	status *TsStatus,
	ipv4 string,
	nodeID string,
	phase sipeedPhase,
) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if profile != nil && profile.ID != "" {
		manager.state.ProfileID = profile.ID
	}
	if status != nil && strings.TrimSpace(status.Self.HostName) != "" {
		manager.state.Hostname = strings.TrimSpace(status.Self.HostName)
	}
	manager.state.IP = ipv4
	if nodeID != "" {
		manager.state.NodeID = nodeID
	}
	manager.state.PreviousProfileID = ""
	manager.state.Phase = phase
	manager.state.ManagedError = ""
	manager.state.RetryAfter = ""
	if phase == sipeedPhaseComplete {
		manager.state.ManagedState = proto.TailscaleManagedEnabledConnected
	} else {
		manager.state.ManagedState = proto.TailscaleManagedEnabledConnecting
	}
	_ = manager.saveLocked()
}

func (manager *sipeedManager) reportSelfNode(
	ctx context.Context,
	client sipeedManagedLocalClient,
	profile *TsProfile,
	ipv4 string,
) (string, bool) {
	state := manager.snapshot()
	nodeID, err := manager.deps.pairing.ReportSelfNode(ctx, state.DeviceToken, ipv4)
	if err != nil {
		manager.failPairing(err, false, client, managedProfileForInvalidation(profile))
		return "", false
	}
	return nodeID, true
}

func managedProfileForInvalidation(profile *TsProfile) *TsProfile {
	if profile == nil || profile.ControlURL != sipeedControlURL {
		return nil
	}
	return profile
}

func (manager *sipeedManager) failPairing(
	err error,
	rollback bool,
	client sipeedManagedLocalClient,
	invalidProfile *TsProfile,
) {
	code, invalidIdentity, retryAfter := managedErrorFromPairing(err)
	if invalidIdentity {
		manager.resetInvalidLocalProfile(client, invalidProfile)
		manager.mu.Lock()
		invalid := newDisabledSipeedState()
		invalid.ManagedState = proto.TailscaleManagedInvalidIdentity
		invalid.ManagedError = code
		manager.state = invalid
		manager.claim = nil
		_ = manager.saveLocked()
		manager.mu.Unlock()
		log.Warnf("sipeed enable failed: %s", code)
		return
	}
	manager.mu.Lock()
	if retryAfter > 0 {
		manager.state.RetryAfter = manager.deps.now().Add(retryAfter).UTC().Format(time.RFC3339)
	}
	manager.mu.Unlock()
	manager.fail(code, rollback, client)
}

// A rejected bearer token can no longer recover the selected managed
// profile. Drop its node key before forgetting the credential so a revoked or
// recycled device cannot remain active locally as an orphaned identity.
func (manager *sipeedManager) resetInvalidLocalProfile(
	client sipeedManagedLocalClient,
	profile *TsProfile,
) {
	if client == nil || profile == nil || profile.ControlURL != sipeedControlURL {
		return
	}
	if err := client.Logout(); err != nil {
		log.Warnf("reset invalid Sipeed profile failed")
		return
	}
	if err := manager.createAndSelectEmptyProfile(client, profile); err != nil {
		log.Warnf("create empty profile for invalid Sipeed identity failed")
	}
}

func (manager *sipeedManager) fail(
	code string,
	rollback bool,
	client sipeedManagedLocalClient,
) {
	// Only emit the stable internal code. Never include the remote error,
	// request body, bearer token, or short-lived auth key in logs.
	log.Warnf("sipeed enable failed: %s", code)
	previousProfileID := ""
	if rollback && client != nil {
		manager.mu.Lock()
		previousProfileID = manager.state.PreviousProfileID
		manager.mu.Unlock()
	}
	if previousProfileID != "" {
		if err := client.SwitchProfile(previousProfileID); err != nil {
			code = "rollbackFailed"
		}
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.failLocked(code)
}

func (manager *sipeedManager) failLocked(code string) {
	manager.state.ManagedState = proto.TailscaleManagedDegraded
	manager.state.ManagedError = code
	if err := manager.saveLocked(); err != nil {
		manager.state.ManagedError = "stateWriteFailed"
	}
}

func (manager *sipeedManager) saveLocked() error {
	if err := manager.deps.store.save(manager.state); err != nil {
		manager.storeErr = err
		manager.state.ManagedState = proto.TailscaleManagedDegraded
		manager.state.ManagedError = "stateWriteFailed"
		return err
	}
	return nil
}

func (manager *sipeedManager) snapshot() sipeedPersistedState {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.state
}

func (manager *sipeedManager) retryDeadlineLocked() time.Time {
	if manager.state.RetryAfter == "" {
		return time.Time{}
	}
	deadline, _ := time.Parse(time.RFC3339, manager.state.RetryAfter)
	return deadline
}

func (manager *sipeedManager) appendStatus(data *proto.GetTailscaleStatusRsp) {
	if manager == nil || data == nil {
		return
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	data.ManagedState = manager.state.ManagedState
	data.ManagedError = manager.state.ManagedError
	if deadline := manager.retryDeadlineLocked(); !deadline.IsZero() {
		remaining := deadline.Sub(manager.deps.now()).Seconds()
		if remaining > 0 {
			data.RetryAfterSeconds = int(math.Ceil(remaining))
		}
	}
}

// disableForGlobal removes the managed identity without touching an active
// official identity. Local managed logout is mandatory; destroying the
// managed cloud network is best effort and is reported separately.
func (manager *sipeedManager) disableForGlobal(ctx context.Context) (bool, error) {
	if manager == nil {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	state := manager.snapshot()
	hasManagedLocalProfile := state.ProfileID != "" ||
		state.Phase == sipeedPhaseConnecting ||
		state.Phase == sipeedPhaseReporting ||
		state.Phase == sipeedPhaseComplete ||
		state.ManagedState == proto.TailscaleManagedEnabledConnecting ||
		state.ManagedState == proto.TailscaleManagedEnabledConnected

	if manager.deps.isInstalled() {
		client := manager.deps.newLocalClient(ctx)
		current, err := client.CurrentProfile()
		if err != nil {
			if hasManagedLocalProfile {
				return false, fmt.Errorf("read managed tailscale profile: %w", err)
			}
		} else if current.ControlURL == sipeedControlURL {
			if err := client.Logout(); err != nil {
				return false, fmt.Errorf("logout managed tailscale profile: %w", err)
			}
		}
	} else if hasManagedLocalProfile {
		return false, errors.New("managed tailscale client is unavailable")
	}

	cloudCleanupFailed := manager.deleteNetwork(ctx, state)

	if err := manager.clearIdentity(); err != nil {
		return cloudCleanupFailed, fmt.Errorf("clear managed tailscale identity: %w", err)
	}
	return cloudCleanupFailed, nil
}

// logout removes an active Sipeed identity and leaves tailscaled on a fresh
// empty profile. If a previous attempt already reached an anonymous profile,
// or another non-Sipeed profile is current, that profile is left untouched so
// retrying logout is idempotent and never disrupts an official identity.
// Destroying the entire cloud network is best effort after the local
// transition has completed.
func (manager *sipeedManager) logout(ctx context.Context) (bool, error) {
	if manager == nil {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	state := manager.snapshot()
	if !manager.hasManagedRecord() {
		return false, nil
	}
	if !manager.deps.isInstalled() {
		return false, errors.New("managed tailscale client is unavailable")
	}

	client := manager.deps.newLocalClient(ctx)
	current, err := client.CurrentProfile()
	if err != nil {
		return false, fmt.Errorf("read managed tailscale profile: %w", err)
	}
	if current != nil && current.ControlURL == sipeedControlURL {
		if err := client.Logout(); err != nil {
			if !manager.waitForManagedLogout(ctx, client) {
				return false, fmt.Errorf("logout managed tailscale profile: %w", err)
			}
		} else if err := manager.createAndSelectEmptyProfile(client, current); err != nil {
			return false, err
		}
	}

	cloudCleanupFailed := manager.deleteNetwork(ctx, state)
	if err := manager.clearIdentity(); err != nil {
		return cloudCleanupFailed, fmt.Errorf("clear managed tailscale identity: %w", err)
	}
	return cloudCleanupFailed, nil
}

func (manager *sipeedManager) deleteNetwork(
	ctx context.Context,
	state sipeedPersistedState,
) bool {
	// An identity that never completed enrollment has no cloud network to
	// destroy. It can be discarded locally without reporting a cleanup failure.
	if state.NetworkID == "" {
		return false
	}
	if state.DeviceToken == "" {
		log.Warn("sipeed cloud network cleanup failed: device token is unavailable")
		return true
	}
	if err := manager.deps.pairing.DeleteNetwork(ctx, state.DeviceToken, state.NetworkID); err != nil {
		// Pairing errors are intentionally stable and contain neither the bearer
		// token nor an upstream response body.
		log.Warnf("sipeed cloud network cleanup failed: %s", err)
		return true
	}
	return false
}

// A LocalAPI logout can outlive the request timeout while tailscaled waits for
// its control server. Re-read the selected profile for a short bounded period:
// once it is no longer a Sipeed profile, the local security transition did
// complete and cloud/state cleanup can safely continue.
func (manager *sipeedManager) waitForManagedLogout(
	ctx context.Context,
	client sipeedManagedLocalClient,
) bool {
	for attempt := 0; attempt <= sipeedLogoutReconcileAttempts; attempt++ {
		profile, err := client.CurrentProfile()
		if err == nil && profile != nil && profile.ControlURL != sipeedControlURL {
			return true
		}
		if attempt == sipeedLogoutReconcileAttempts {
			break
		}
		if err := manager.deps.wait(ctx, sipeedLogoutReconcileInterval); err != nil {
			break
		}
	}
	return false
}

// createAndSelectEmptyProfile is intentionally explicit.  LocalAPI creates a
// profile with PUT /profiles/, but older daemons do not consistently make the
// new profile current, so we discover the new identity and POST a switch.
func (manager *sipeedManager) createAndSelectEmptyProfile(
	client sipeedManagedLocalClient,
	previous *TsProfile,
) error {
	if client == nil {
		return errors.New("managed tailscale client is unavailable")
	}
	// A completed LocalAPI logout is represented by an anonymous current
	// profile and an empty profile list on current tailscaled releases.  That
	// is already the clean state we need, and it also makes logout retryable
	// after an older server returned an error between the local transition and
	// clearing the persisted Sipeed record.
	if isAnonymousEmptyTailscaleProfile(previous) {
		return nil
	}
	previousID := ""
	if previous != nil {
		previousID = previous.ID
	}
	if err := client.NewProfile(); err != nil {
		return fmt.Errorf("create empty tailscale profile: %w", err)
	}

	// Prefer the profile list because it works with daemons that do not switch
	// automatically.  The newest empty profile is the one PUT just created.
	if profiles, err := client.Profiles(); err == nil {
		for index := len(profiles) - 1; index >= 0; index-- {
			profile := profiles[index]
			if profile.ID == "" || profile.ID == previousID || profile.NodeID != "" ||
				profile.ControlURL == sipeedControlURL {
				continue
			}
			if err := client.SwitchProfile(profile.ID); err != nil {
				return fmt.Errorf("switch to empty tailscale profile: %w", err)
			}
			return nil
		}
	}

	// Fallback for LocalAPI implementations that expose only the current
	// profile after creation (and for the small test doubles used by callers).
	created, err := client.CurrentProfile()
	if err != nil {
		return fmt.Errorf("read empty tailscale profile: %w", err)
	}
	if isAnonymousEmptyTailscaleProfile(created) {
		return nil
	}
	if created == nil || created.ID == "" || created.ID == previousID ||
		created.NodeID != "" || created.ControlURL == sipeedControlURL {
		return errors.New("new tailscale profile is not empty")
	}
	if err := client.SwitchProfile(created.ID); err != nil {
		return fmt.Errorf("switch to empty tailscale profile: %w", err)
	}
	return nil
}

func isAnonymousEmptyTailscaleProfile(profile *TsProfile) bool {
	return profile != nil && profile.ID == "" && profile.Name == "" && profile.NodeID == "" &&
		profile.ControlURL == ""
}

// invalidateIdentity handles an upstream 401.  The token is no longer
// usable, so persisted credentials are discarded even if a best-effort local
// profile reset cannot be completed.  This path is deliberately different
// from an explicit logout, where a local failure must preserve the identity
// for retry.
func (manager *sipeedManager) invalidateIdentity(ctx context.Context) error {
	if manager == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if manager.deps.isInstalled() {
		client := manager.deps.newLocalClient(ctx)
		if current, err := client.CurrentProfile(); err == nil && current != nil &&
			current.ControlURL == sipeedControlURL {
			if err := client.Logout(); err != nil {
				log.Warnf("reset invalid Sipeed profile failed")
			} else if err := manager.createAndSelectEmptyProfile(client, current); err != nil {
				log.Warnf("create empty profile for invalid Sipeed identity failed")
			}
		}
	}
	return manager.clearIdentity()
}

func (manager *sipeedManager) pairingCredentials() (string, string, error) {
	if manager == nil {
		return "", "", errors.New("managed tailscale mode is unavailable")
	}
	state := manager.snapshot()
	if !validRawURLCredential(state.DeviceToken, sipeedDeviceTokenBytes) {
		return "", "", errors.New("managed tailscale identity is unavailable")
	}
	return state.DeviceToken, state.NodeID, nil
}

// claimCode returns the managed network's current single-use claim code. One
// cache is shared by every transport (the authenticated HTTP handlers and the
// root-local gate driver), so the LCD and the web always display the same
// code: while the cached code is further than the renew skew from its
// upstream expiry it is served as-is, otherwise exactly one upstream renewal
// runs under claimMu (single-flight). The state mutex is only taken briefly
// to read the identity and to publish the entry, never across the network
// call. An upstream 429 persists its Retry-After deadline, matching the
// per-lane persistence the callers have always applied.
func (manager *sipeedManager) claimCode(ctx context.Context) (*sipeedClaimCodeResponse, error) {
	return manager.getClaimCode(ctx, false)
}

// getClaimCode also supports an explicit browser renewal. A renewal drops the
// previous code before contacting upstream, including when that request fails.
func (manager *sipeedManager) getClaimCode(ctx context.Context, refresh bool) (*sipeedClaimCodeResponse, error) {
	if manager == nil {
		return nil, errManagedModeUnavailable
	}
	pairing, ok := manager.deps.pairing.(sipeedNetworkPairingClient)
	if !ok || pairing == nil {
		return nil, errSipeedManagedPairingUnavailable
	}

	manager.claimMu.Lock()
	defer manager.claimMu.Unlock()

	manager.mu.Lock()
	token := manager.state.DeviceToken
	if refresh {
		manager.claim = nil
	}
	entry := manager.claim
	manager.mu.Unlock()

	if !validRawURLCredential(token, sipeedDeviceTokenBytes) {
		return nil, errSipeedManagedIdentityUnavailable
	}
	if entry != nil && entry.token == token &&
		manager.deps.now().Add(sipeedClaimCodeRenewSkew).Before(entry.expiresAt) {
		return &sipeedClaimCodeResponse{
			Code:      entry.code,
			ExpiresAt: entry.expiresAt.UTC().Format(time.RFC3339),
			SingleUse: entry.singleUse,
		}, nil
	}

	response, err := pairing.ClaimCode(ctx, token)
	if err != nil {
		var apiError *sipeedPairingAPIError
		if errors.As(err, &apiError) && apiError.HTTPStatus == http.StatusTooManyRequests {
			if saveErr := manager.setPairingRetryAfter(apiError.RetryAfter); saveErr != nil {
				log.Warn("sipeed claim code retry-after save failed")
			}
		}
		return nil, err
	}
	// The pairing client already validates the response; re-check before
	// caching so a contract violation can never poison the shared entry.
	digits, err := normalizeSipeedClaimCode(response.Code)
	if err != nil {
		return nil, errSipeedClaimCodeProtocol
	}
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return nil, errSipeedClaimCodeProtocol
	}

	issued := &sipeedClaimCodeResponse{
		Code:      formatSipeedClaimCode(digits),
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		SingleUse: response.SingleUse,
	}
	manager.mu.Lock()
	// The identity may have changed while the upstream call was in flight;
	// only publish a code that still belongs to the current token.
	if manager.state.DeviceToken == token {
		manager.claim = &sipeedClaimCodeCacheEntry{
			token:     token,
			code:      issued.Code,
			expiresAt: expiresAt,
			singleUse: issued.SingleUse,
		}
	}
	manager.mu.Unlock()
	return issued, nil
}

// Capture the current code before a pairing operation. Its response may consume
// that code, but must not invalidate a newer code issued while it was in flight.
func (manager *sipeedManager) claimCodeInvalidator(token string) func() {
	manager.mu.Lock()
	entry := manager.claim
	manager.mu.Unlock()
	return func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		if entry != nil && entry.token == token && manager.claim == entry {
			manager.claim = nil
		}
	}
}

// pairingRetryRemaining is shared by all browser-facing management endpoints.
// The deadline is persisted so a restart cannot immediately repeat a
// rate-limited upstream request.
func (manager *sipeedManager) pairingRetryRemaining() time.Duration {
	if manager == nil {
		return 0
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	deadline := manager.retryDeadlineLocked()
	if deadline.IsZero() {
		return 0
	}
	remaining := deadline.Sub(manager.deps.now())
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (manager *sipeedManager) setPairingRetryAfter(retryAfter time.Duration) error {
	if manager == nil {
		return errors.New("managed tailscale mode is unavailable")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if retryAfter <= 0 {
		manager.state.RetryAfter = ""
	} else {
		if retryAfter > 24*time.Hour {
			retryAfter = 24 * time.Hour
		}
		manager.state.RetryAfter = manager.deps.now().Add(retryAfter).UTC().Format(time.RFC3339)
	}
	return manager.saveLocked()
}

func (manager *sipeedManager) clearIdentity() error {
	disabled := newDisabledSipeedState()
	if err := manager.deps.store.save(disabled); err != nil {
		manager.mu.Lock()
		manager.storeErr = err
		manager.mu.Unlock()
		return err
	}

	manager.mu.Lock()
	manager.state = disabled
	manager.storeErr = nil
	manager.claim = nil
	manager.mu.Unlock()
	return nil
}

// hasManagedRecord reports whether the local state contains any Sipeed
// identity or operation history. A failed/rate-limited identity is still a
// meaningful record: status normalization must not erase it merely because a
// readable official profile happens to be current.
func (manager *sipeedManager) hasManagedRecord() bool {
	if manager == nil {
		return false
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	state := manager.state
	return state.ManagedState != proto.TailscaleManagedDisabled ||
		state.ManagedError != "" || state.RetryAfter != "" ||
		state.DeviceID != "" || state.DeviceToken != "" || state.NetworkID != "" ||
		state.NodeID != "" || state.ProfileID != "" || state.Hostname != "" ||
		state.IP != "" || state.Phase != sipeedPhaseNone
}
