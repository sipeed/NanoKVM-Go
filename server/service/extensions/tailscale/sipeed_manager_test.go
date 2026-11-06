package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

type fakeSipeedPairingClient struct {
	mu                  sync.Mutex
	events              *[]string
	enrollResponse      *sipeedEnrollResponse
	enrollErr           error
	joinKeyResponse     *sipeedJoinKeyResponse
	joinKeyErr          error
	reportErr           error
	deleteErr           error
	deleteNetworkErr    error
	reportNodeID        string
	enrollTokens        []string
	joinKeyTokens       []string
	reportTokens        []string
	reportIPs           []string
	deleteTokens        []string
	deleteNodeIDs       []string
	deleteNetworkTokens []string
	deleteNetworkIDs    []string
	enrollStarted       chan struct{}
	enrollRelease       chan struct{}
	reportDone          chan struct{}
}

func (fake *fakeSipeedPairingClient) Enroll(
	_ context.Context,
	_ string,
	token string,
	_ string,
) (*sipeedEnrollResponse, error) {
	fake.mu.Lock()
	if fake.events != nil {
		*fake.events = append(*fake.events, "pairing.enroll")
	}
	fake.enrollTokens = append(fake.enrollTokens, token)
	err := fake.enrollErr
	response := fake.enrollResponse
	started := fake.enrollStarted
	release := fake.enrollRelease
	fake.mu.Unlock()
	if started != nil {
		close(started)
	}
	if release != nil {
		<-release
	}
	if err != nil {
		return nil, err
	}
	copy := *response
	return &copy, nil
}

func (fake *fakeSipeedPairingClient) JoinKey(
	_ context.Context,
	token string,
) (*sipeedJoinKeyResponse, error) {
	fake.mu.Lock()
	if fake.events != nil {
		*fake.events = append(*fake.events, "pairing.joinKey")
	}
	fake.joinKeyTokens = append(fake.joinKeyTokens, token)
	err := fake.joinKeyErr
	response := fake.joinKeyResponse
	fake.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if response == nil {
		return &sipeedJoinKeyResponse{
			AuthKey:   "hskey-auth-memory-only",
			ExpiresAt: "2026-09-17T08:00:00Z",
		}, nil
	}
	copy := *response
	return &copy, nil
}

func (fake *fakeSipeedPairingClient) ReportSelfNode(
	_ context.Context,
	token string,
	ip string,
) (string, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.events != nil {
		*fake.events = append(*fake.events, "pairing.selfNode")
	}
	fake.reportTokens = append(fake.reportTokens, token)
	fake.reportIPs = append(fake.reportIPs, ip)
	err := fake.reportErr
	done := fake.reportDone
	if done != nil {
		close(done)
		fake.reportDone = nil
	}
	if err != nil {
		return "", err
	}
	if fake.reportNodeID != "" {
		return fake.reportNodeID, nil
	}
	return "3", nil
}

func (fake *fakeSipeedPairingClient) DeleteDevice(
	_ context.Context,
	token string,
	nodeID string,
) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.events != nil {
		*fake.events = append(*fake.events, "pairing.deleteDevice")
	}
	fake.deleteTokens = append(fake.deleteTokens, token)
	fake.deleteNodeIDs = append(fake.deleteNodeIDs, nodeID)
	return fake.deleteErr
}

func (fake *fakeSipeedPairingClient) DeleteNetwork(
	_ context.Context,
	token string,
	networkID string,
) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.events != nil {
		*fake.events = append(*fake.events, "pairing.deleteNetwork")
	}
	fake.deleteNetworkTokens = append(fake.deleteNetworkTokens, token)
	fake.deleteNetworkIDs = append(fake.deleteNetworkIDs, networkID)
	return fake.deleteNetworkErr
}

type fakeSipeedLocalClient struct {
	events           *[]string
	statuses         []*TsStatus
	statusErr        error
	currentProfiles  []*TsProfile
	profiles         []TsProfile
	startErr         error
	newProfileErr    error
	startManagedErr  error
	switchErr        error
	logoutErr        error
	statusCalls      int
	currentCalls     int
	switchCalls      []string
	authKeys         []string
	loginInteractive []bool
}

func (fake *fakeSipeedLocalClient) event(name string) {
	if fake.events != nil {
		*fake.events = append(*fake.events, name)
	}
}

func (fake *fakeSipeedLocalClient) Start() error {
	fake.event("local.startDaemon")
	return fake.startErr
}

func (fake *fakeSipeedLocalClient) Status() (*TsStatus, error) {
	fake.event("local.status")
	if fake.statusErr != nil {
		return nil, fake.statusErr
	}
	index := fake.statusCalls
	fake.statusCalls++
	if index >= len(fake.statuses) {
		index = len(fake.statuses) - 1
	}
	if index < 0 {
		return nil, errors.New("missing fake status")
	}
	return fake.statuses[index], nil
}

func (fake *fakeSipeedLocalClient) Profiles() ([]TsProfile, error) {
	fake.event("local.profiles")
	return append([]TsProfile(nil), fake.profiles...), nil
}

func (fake *fakeSipeedLocalClient) CurrentProfile() (*TsProfile, error) {
	fake.event("local.currentProfile")
	index := fake.currentCalls
	fake.currentCalls++
	if index >= len(fake.currentProfiles) {
		index = len(fake.currentProfiles) - 1
	}
	if index < 0 {
		return nil, errors.New("missing fake current profile")
	}
	profile := *fake.currentProfiles[index]
	return &profile, nil
}

func (fake *fakeSipeedLocalClient) NewProfile() error {
	fake.event("local.newProfile")
	return fake.newProfileErr
}

func (fake *fakeSipeedLocalClient) SwitchProfile(profileID string) error {
	fake.event("local.switchProfile:" + profileID)
	fake.switchCalls = append(fake.switchCalls, profileID)
	return fake.switchErr
}

func (fake *fakeSipeedLocalClient) StartManaged(authKey, controlURL, hostname string, loginInteractive bool) error {
	fake.event("local.startManaged")
	fake.authKeys = append(fake.authKeys, authKey)
	fake.loginInteractive = append(fake.loginInteractive, loginInteractive)
	if controlURL != sipeedControlURL || hostname != "nano-kvm" {
		return errors.New("unexpected managed start parameters")
	}
	return fake.startManagedErr
}

func (fake *fakeSipeedLocalClient) Logout() error {
	fake.event("local.logout")
	return fake.logoutErr
}

func testSipeedStatus(state string, ips ...string) *TsStatus {
	status := &TsStatus{Version: TailscaleVersion, BackendState: state}
	status.Self.HostName = "nano-kvm"
	status.Self.TailscaleIPs = ips
	return status
}

func testSipeedManager(
	t *testing.T,
	local *fakeSipeedLocalClient,
	pairing *fakeSipeedPairingClient,
	now time.Time,
) *sipeedManager {
	t.Helper()
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json"))
	return newSipeedManager(sipeedManagerDeps{
		store:       store,
		pairing:     pairing,
		isInstalled: func() bool { return true },
		install:     func() error { t.Fatal("install must not be called"); return nil },
		hostname:    func() (string, error) { return "nano-kvm", nil },
		now:         func() time.Time { return now },
		wait:        func(context.Context, time.Duration) error { return nil },
		newLocalClient: func(context.Context) sipeedManagedLocalClient {
			return local
		},
	})
}

func successfulSipeedFakes(events *[]string) (*fakeSipeedLocalClient, *fakeSipeedPairingClient) {
	local := &fakeSipeedLocalClient{
		events: events,
		statuses: []*TsStatus{
			testSipeedStatus("NeedsLogin"),
			testSipeedStatus("Running", "fd7a:115c:a1e0::1", "100.64.0.3"),
		},
		currentProfiles: []*TsProfile{
			{},
			{ID: "1ab3", Name: "Sipeed", NodeID: "node-1", ControlURL: sipeedControlURL},
		},
	}
	pairing := &fakeSipeedPairingClient{
		events: events,
		enrollResponse: &sipeedEnrollResponse{
			NetworkID: "net-test",
			AuthKey:   "hskey-auth-memory-only",
			ExpiresAt: "2026-09-17T08:00:00Z",
		},
		joinKeyResponse: &sipeedJoinKeyResponse{
			AuthKey:   "hskey-auth-memory-only",
			ExpiresAt: "2026-09-17T08:00:00Z",
		},
	}
	return local, pairing
}

func TestSipeedEnableStartsStoppedDaemon(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	local.statusErr = errors.New("daemon is stopped")
	manager := testSipeedManager(t, local, pairing, time.Date(2026, 9, 17, 7, 0, 0, 0, time.UTC))
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.deps.wait = func(context.Context, time.Duration) error {
		local.statusErr = nil
		return nil
	}
	manager.enable(context.Background())
	if manager.snapshot().ManagedState != proto.TailscaleManagedEnabledConnected {
		t.Fatalf("enable from a stopped daemon did not connect: %+v", manager.snapshot())
	}
	starts := 0
	for _, event := range events {
		if event == "local.startDaemon" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("daemon starts = %d, want 1; events = %v", starts, events)
	}
}

func TestSipeedEnableWorkflowUsesExpectedOrderAndPersistsNoAuthKey(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager := testSipeedManager(t, local, pairing, time.Date(2026, 9, 17, 7, 0, 0, 0, time.UTC))
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())

	wantEvents := []string{
		"local.status",
		"local.currentProfile",
		"pairing.enroll",
		"local.profiles",
		"local.newProfile",
		"local.startManaged",
		"local.status",
		"local.currentProfile",
		"pairing.selfNode",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
	state := manager.snapshot()
	if state.ManagedState != proto.TailscaleManagedEnabledConnected || state.Phase != sipeedPhaseComplete ||
		state.NetworkID != "net-test" || state.NodeID != "3" || state.ProfileID != "1ab3" ||
		state.IP != "100.64.0.3" {
		t.Fatalf("final state = %+v", state)
	}
	if !validRawURLCredential(state.DeviceID, sipeedDeviceIDBytes) ||
		!validRawURLCredential(state.DeviceToken, sipeedDeviceTokenBytes) {
		t.Fatalf("invalid persisted identity: %+v", state)
	}
	if len(pairing.enrollTokens) != 1 || pairing.enrollTokens[0] != state.DeviceToken ||
		len(pairing.reportTokens) != 1 || pairing.reportTokens[0] != state.DeviceToken {
		t.Fatal("pairing calls did not reuse the persisted bearer token")
	}
	if len(local.authKeys) != 1 || local.authKeys[0] != "hskey-auth-memory-only" {
		t.Fatalf("LocalAPI auth keys = %v", local.authKeys)
	}
	if len(local.loginInteractive) != 1 || !local.loginInteractive[0] {
		t.Fatalf("initial managed login did not request interactive completion: %v", local.loginInteractive)
	}
	body, err := os.ReadFile(manager.deps.store.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "hskey-auth-memory-only") || strings.Contains(string(body), sipeedControlURL) {
		t.Fatalf("state file leaked transient control data: %s", body)
	}
}

func TestSipeedEnableLogsOutOfficialBeforeEnrollment(t *testing.T) {
	for _, tc := range []struct {
		name  string
		retry bool
	}{
		{name: "first enable"}, {name: "retry", retry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []string
			local, pairing := successfulSipeedFakes(&events)
			local.statuses[0] = testSipeedStatus("Running", "100.64.0.8")
			local.statuses[0].HaveNodeKey = true
			local.currentProfiles = []*TsProfile{
				{ID: "beef", NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com"},
				{}, local.currentProfiles[1],
			}
			manager := testSipeedManager(t, local, pairing, time.Now())
			if tc.retry {
				manager.state = validTestSipeedState(t)
				manager.state.ManagedState = proto.TailscaleManagedDegraded
				manager.state.Phase = sipeedPhaseEnrolled
				manager.state.ProfileID = ""
			}
			if err := manager.prepareEnable(); err != nil {
				t.Fatal(err)
			}
			manager.enable(context.Background())
			state := manager.snapshot()
			if state.ManagedState != proto.TailscaleManagedEnabledConnected || state.PreviousProfileID != "" {
				t.Fatalf("official logout and managed enable state = %+v", state)
			}
			request := "pairing.enroll"
			if tc.retry {
				request = "pairing.joinKey"
			}
			wantPrefix := []string{"local.status", "local.currentProfile", "local.logout", "local.currentProfile", request}
			if len(events) < len(wantPrefix) || !reflect.DeepEqual(events[:len(wantPrefix)], wantPrefix) {
				t.Fatalf("old official identity was not logged out before enrollment: %v", events)
			}
			if !reflect.DeepEqual(local.loginInteractive, []bool{true}) {
				t.Fatalf("managed start reused the logged-out node key: %v", local.loginInteractive)
			}
		})
	}
}

func TestSipeedEnableStopsWhenOfficialLogoutFails(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.statuses[0] = testSipeedStatus("Running", "100.64.0.8")
	local.currentProfiles[0] = &TsProfile{ID: "beef", NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com"}
	local.logoutErr = errors.New("logout failed")
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	if manager.snapshot().ManagedError != "profileSetupFailed" || len(pairing.enrollTokens) != 0 || len(local.authKeys) != 0 {
		t.Fatalf("logout failure did not stop enrollment: %+v", manager.snapshot())
	}
}

func TestSipeedProfileFailureRollsBackAndKeepsRetryIdentity(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.currentProfiles = []*TsProfile{{ID: "beef"}}
	local.startManagedErr = errors.New("start failed")
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	state := manager.snapshot()
	if state.ManagedError != "profileStartFailed" || state.DeviceToken == "" || state.NetworkID != "net-test" ||
		!reflect.DeepEqual(local.switchCalls, []string{"beef"}) {
		t.Fatalf("rollback result state=%+v switches=%v", state, local.switchCalls)
	}

	local, pairing = successfulSipeedFakes(nil)
	local.currentProfiles = []*TsProfile{{ID: "beef"}}
	local.startManagedErr = errors.New("start failed")
	local.switchErr = errors.New("rollback failed")
	manager = testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	if got := manager.snapshot().ManagedError; got != "rollbackFailed" {
		t.Fatalf("rollback failure code = %q", got)
	}
}

func TestSipeedProfileCreationFailureRestoresRememberedProfile(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.currentProfiles = []*TsProfile{{ID: "beef"}}
	local.newProfileErr = errors.New("create profile failed after switching")
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())

	state := manager.snapshot()
	if state.ManagedError != "profileSetupFailed" || state.PreviousProfileID != "beef" ||
		state.DeviceToken == "" || state.NetworkID != "net-test" ||
		!reflect.DeepEqual(local.switchCalls, []string{"beef"}) {
		t.Fatalf("profile creation rollback state=%+v switches=%v", state, local.switchCalls)
	}
}

func TestSipeedSelfNodeFailureKeepsConnectedProfileForReportingRetry(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	pairing.reportErr = &sipeedPairingAPIError{HTTPStatus: 502, Code: "upstream_error"}
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	state := manager.snapshot()
	if state.ManagedState != proto.TailscaleManagedDegraded || state.ManagedError != "pairingUpstreamError" ||
		state.Phase != sipeedPhaseReporting || state.ProfileID != "1ab3" || state.IP != "100.64.0.3" ||
		len(local.switchCalls) != 0 {
		t.Fatalf("self-node failure state=%+v switches=%v", state, local.switchCalls)
	}
}

func TestSipeedInvalidTokenClearsIdentityAndRateLimitPersistsDeadline(t *testing.T) {
	t.Run("invalid token", func(t *testing.T) {
		local, pairing := successfulSipeedFakes(nil)
		pairing.enrollErr = &sipeedPairingAPIError{HTTPStatus: 401, Code: "invalid_token"}
		manager := testSipeedManager(t, local, pairing, time.Now())
		if err := manager.prepareEnable(); err != nil {
			t.Fatal(err)
		}
		manager.enable(context.Background())
		state := manager.snapshot()
		if state.ManagedState != proto.TailscaleManagedInvalidIdentity || state.ManagedError != "invalidToken" ||
			state.DeviceID != "" || state.DeviceToken != "" || state.NetworkID != "" {
			t.Fatalf("invalid-token state = %+v", state)
		}
	})

	t.Run("invalid rejoin token resets selected managed profile", func(t *testing.T) {
		var events []string
		local, pairing := successfulSipeedFakes(&events)
		pairing.joinKeyErr = &sipeedPairingAPIError{HTTPStatus: 401, Code: "invalid_token"}
		managedProfile := TsProfile{
			ID: "managed-profile", NodeID: "node-1", ControlURL: sipeedControlURL,
		}
		local.statuses = []*TsStatus{testSipeedStatus("NeedsLogin")}
		local.currentProfiles = []*TsProfile{&managedProfile}
		local.profiles = []TsProfile{
			managedProfile,
			{ID: "empty-profile"},
		}

		store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
		state := validTestSipeedState(t)
		state.ManagedState = proto.TailscaleManagedDegraded
		state.ManagedError = "connectionLost"
		state.Phase = sipeedPhaseEnrolled
		state.ProfileID = managedProfile.ID
		state.PreviousProfileID = "official-profile"
		state.IP = ""
		if err := store.save(state); err != nil {
			t.Fatal(err)
		}
		manager := newSipeedManager(sipeedManagerDeps{
			store:          store,
			pairing:        pairing,
			newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
			isInstalled:    func() bool { return true },
			hostname:       func() (string, error) { return "nano-kvm", nil },
			now:            time.Now,
			wait:           func(context.Context, time.Duration) error { return nil },
		})
		if err := manager.prepareEnable(); err != nil {
			t.Fatal(err)
		}
		manager.enable(context.Background())

		final := manager.snapshot()
		if final.ManagedState != proto.TailscaleManagedInvalidIdentity ||
			final.ManagedError != "invalidToken" || final.DeviceID != "" ||
			final.DeviceToken != "" || final.NetworkID != "" || final.NodeID != "" ||
			final.ProfileID != "" || final.IP != "" ||
			final.PreviousProfileID != "" {
			t.Fatalf("invalid rejoin state = %+v", final)
		}
		wantEvents := []string{
			"local.status",
			"local.currentProfile",
			"pairing.joinKey",
			"local.logout",
			"local.newProfile",
			"local.profiles",
			"local.switchProfile:empty-profile",
		}
		if !reflect.DeepEqual(events, wantEvents) ||
			!reflect.DeepEqual(local.switchCalls, []string{"empty-profile"}) ||
			len(pairing.enrollTokens) != 0 || len(pairing.joinKeyTokens) != 1 {
			t.Fatalf("invalid rejoin events=%v switches=%v", events, local.switchCalls)
		}
	})

	t.Run("rate limit", func(t *testing.T) {
		now := time.Date(2026, 9, 17, 7, 0, 0, 0, time.UTC)
		local, pairing := successfulSipeedFakes(nil)
		pairing.enrollErr = &sipeedPairingAPIError{HTTPStatus: 429, Code: "rate_limited", RetryAfter: 37 * time.Second}
		manager := testSipeedManager(t, local, pairing, now)
		if err := manager.prepareEnable(); err != nil {
			t.Fatal(err)
		}
		manager.enable(context.Background())
		data := &proto.GetTailscaleStatusRsp{}
		manager.appendStatus(data)
		if data.ManagedError != "rateLimited" || data.RetryAfterSeconds != 37 {
			t.Fatalf("rate-limit status = %+v", data)
		}
		if err := manager.prepareEnable(); err == nil {
			t.Fatal("prepareEnable ignored persisted rate limit")
		}
	})
}

func TestSipeedRetryReusesIdentityAndNetwork(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "connectionFailed"
	state.Phase = sipeedPhaseEnrolled
	state.ProfileID = ""
	state.IP = ""
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	final := manager.snapshot()
	if final.DeviceID != state.DeviceID || final.DeviceToken != state.DeviceToken || final.NetworkID != state.NetworkID {
		t.Fatalf("retry replaced identity: before=%+v after=%+v", state, final)
	}
	if len(pairing.enrollTokens) != 0 {
		t.Fatalf("registered identity retried enroll %d times", len(pairing.enrollTokens))
	}
	if len(pairing.joinKeyTokens) != 1 || pairing.joinKeyTokens[0] != state.DeviceToken {
		t.Fatalf("join-key tokens = %v, want persisted bearer token", pairing.joinKeyTokens)
	}
}

func TestSipeedJoinKeyFailureDoesNotFallBackToEnroll(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	pairing.joinKeyErr = &sipeedPairingAPIError{HTTPStatus: 404, Code: "not_found"}
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "connectionFailed"
	state.Phase = sipeedPhaseEnrolled
	state.ProfileID = ""
	state.IP = ""
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())

	final := manager.snapshot()
	if final.ManagedState != proto.TailscaleManagedDegraded || final.ManagedError != "nodeNotFound" ||
		final.NetworkID != state.NetworkID || final.DeviceToken != state.DeviceToken {
		t.Fatalf("join-key failure state = %+v", final)
	}
	if len(pairing.joinKeyTokens) != 1 || len(pairing.enrollTokens) != 0 {
		t.Fatalf("pairing calls after join-key failure: join-key=%d enroll=%d", len(pairing.joinKeyTokens), len(pairing.enrollTokens))
	}
}

func TestSipeedJoinKeyRateLimitBlocksRemoteRetry(t *testing.T) {
	now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
	local, pairing := successfulSipeedFakes(nil)
	pairing.joinKeyErr = &sipeedPairingAPIError{
		HTTPStatus: 429,
		Code:       "rate_limited",
		RetryAfter: 29 * time.Second,
	}
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "connectionFailed"
	state.Phase = sipeedPhaseEnrolled
	state.ProfileID = ""
	state.IP = ""
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            func() time.Time { return now },
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())

	data := &proto.GetTailscaleStatusRsp{}
	manager.appendStatus(data)
	if data.ManagedState != proto.TailscaleManagedDegraded || data.ManagedError != "rateLimited" ||
		data.RetryAfterSeconds != 29 {
		t.Fatalf("join-key rate limit status = %+v", data)
	}
	if err := manager.prepareEnable(); err == nil {
		t.Fatal("persisted join-key rate limit allowed an early retry")
	}
	if len(pairing.joinKeyTokens) != 1 || len(pairing.enrollTokens) != 0 {
		t.Fatalf("rate-limited pairing calls: join-key=%d enroll=%d", len(pairing.joinKeyTokens), len(pairing.enrollTokens))
	}
}

func TestSipeedRetryAtReportingOnlyRepeatsSelfNodeCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "pairingUpstreamError"
	state.Phase = sipeedPhaseReporting
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	local := &fakeSipeedLocalClient{
		statuses:        []*TsStatus{testSipeedStatus("Running", "100.64.0.3")},
		currentProfiles: []*TsProfile{{ID: state.ProfileID, NodeID: "node-1", ControlURL: sipeedControlURL}},
	}
	pairing := &fakeSipeedPairingClient{reportDone: make(chan struct{})}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	if len(pairing.enrollTokens) != 0 || len(pairing.reportTokens) != 1 {
		t.Fatalf("retry calls: enroll=%d report=%d", len(pairing.enrollTokens), len(pairing.reportTokens))
	}
	if got := manager.snapshot().ManagedState; got != proto.TailscaleManagedEnabledConnected {
		t.Fatalf("report retry state = %q", got)
	}
}

func TestSipeedEnableReusesExistingManagedProfile(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.currentProfiles[0] = &TsProfile{ID: "beef"}
	local.profiles = []TsProfile{{ID: "1ab3", NodeID: "old-node", ControlURL: sipeedControlURL}}
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	if !reflect.DeepEqual(local.switchCalls, []string{"1ab3"}) {
		t.Fatalf("managed profile switches = %v", local.switchCalls)
	}
	if got := manager.snapshot().ManagedState; got != proto.TailscaleManagedEnabledConnected {
		t.Fatalf("reuse result = %q", got)
	}
}

func TestSipeedEnableInstallsPinnedClientWhenMissing(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	installed := false
	installCalls := 0
	manager := newSipeedManager(sipeedManagerDeps{
		store:          newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json")),
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return installed },
		install: func() error {
			installCalls++
			installed = true
			return nil
		},
		hostname: func() (string, error) { return "nano-kvm", nil },
		now:      time.Now,
		wait:     func(context.Context, time.Duration) error { return nil },
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	if installCalls != 1 || manager.snapshot().ManagedState != proto.TailscaleManagedEnabledConnected {
		t.Fatalf("install calls=%d state=%+v", installCalls, manager.snapshot())
	}
}

func TestSipeedConnectionFailureRestoresPreviousProfile(t *testing.T) {
	now := time.Date(2026, 9, 17, 7, 0, 0, 0, time.UTC)
	local, pairing := successfulSipeedFakes(nil)
	local.currentProfiles = []*TsProfile{{ID: "beef"}}
	local.statuses = []*TsStatus{testSipeedStatus("NeedsLogin")}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json")),
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            func() time.Time { return now },
		wait: func(_ context.Context, duration time.Duration) error {
			now = now.Add(duration)
			return nil
		},
		connectTimeout: 2 * time.Second,
		connectPoll:    time.Second,
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	state := manager.snapshot()
	if state.ManagedError != "connectionFailed" || !reflect.DeepEqual(local.switchCalls, []string{"beef"}) {
		t.Fatalf("connection rollback state=%+v switches=%v", state, local.switchCalls)
	}
}

func TestSipeedInterruptedRetryDiscardsPreviousOfficialProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "interrupted"
	state.Phase = sipeedPhaseConnecting
	state.PreviousProfileID = "beef"
	state.IP = ""
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}

	local, pairing := successfulSipeedFakes(nil)
	local.statuses = []*TsStatus{testSipeedStatus("NeedsLogin")}
	local.currentProfiles = []*TsProfile{{ID: state.ProfileID, ControlURL: sipeedControlURL}}
	local.startManagedErr = errors.New("start failed")
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())

	if got := manager.snapshot().ManagedError; got != "profileStartFailed" {
		t.Fatalf("retry error = %q", got)
	}
	if len(local.switchCalls) != 0 || manager.snapshot().PreviousProfileID != "" {
		t.Fatalf("retry restored a previously saved official profile: %v", local.switchCalls)
	}
}

func TestSipeedSelfNodeInvalidTokenClearsIdentityAndManagedProfile(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.profiles = []TsProfile{{ID: "empty-profile"}}
	pairing.reportErr = &sipeedPairingAPIError{HTTPStatus: 401, Code: "invalid_token"}
	manager := testSipeedManager(t, local, pairing, time.Now())
	if err := manager.prepareEnable(); err != nil {
		t.Fatal(err)
	}
	manager.enable(context.Background())
	state := manager.snapshot()
	if state.ManagedState != proto.TailscaleManagedInvalidIdentity || state.DeviceToken != "" ||
		state.ProfileID != "" || state.IP != "" ||
		!reflect.DeepEqual(local.switchCalls, []string{"empty-profile"}) {
		t.Fatalf("self-node invalid token state = %+v", state)
	}
}

func TestSipeedStartupMarksInterruptedStateDegraded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedEnabledConnecting
	state.Phase = sipeedPhaseConnecting
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{store: store})
	got := manager.snapshot()
	if got.ManagedState != proto.TailscaleManagedDegraded || got.ManagedError != "interrupted" {
		t.Fatalf("startup recovery state = %+v", got)
	}
	reloaded, err := store.load()
	if err != nil || reloaded.ManagedError != "interrupted" {
		t.Fatalf("startup recovery was not persisted: %+v, %v", reloaded, err)
	}
}

func TestPublicManagedStatusContainsNoSecretFields(t *testing.T) {
	manager := &sipeedManager{
		state: validTestSipeedState(t),
		deps:  normalizeSipeedManagerDeps(sipeedManagerDeps{now: time.Now}),
	}
	data := &proto.GetTailscaleStatusRsp{
		AvailableModes: []proto.TailscaleMode{proto.TailscaleModeOfficial, proto.TailscaleModeSipeed},
		ManagedState:   proto.TailscaleManagedDisabled,
	}
	manager.appendStatus(data)
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{
		manager.state.DeviceToken,
		manager.state.NetworkID,
		manager.state.ProfileID,
		sipeedControlURL,
		"deviceToken",
		"networkId",
		"profileId",
		"controlURL",
	} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("public status leaked %q: %s", secret, payload)
		}
	}
}

func TestConnectedManagedStatusDynamicallyDegrades(t *testing.T) {
	for _, code := range []string{"clientMissing", "localAPIUnavailable", "profileChanged", "connectionLost"} {
		data := &proto.GetTailscaleStatusRsp{
			ManagedState: proto.TailscaleManagedEnabledConnected,
		}
		degradeConnectedManagedStatus(data, code)
		if data.ManagedState != proto.TailscaleManagedDegraded || data.ManagedError != code {
			t.Fatalf("dynamic degradation = %+v", data)
		}
	}
}

func newConnectedSipeedLogoutManager(
	t *testing.T,
	local *fakeSipeedLocalClient,
	pairing *fakeSipeedPairingClient,
) (*sipeedManager, sipeedPersistedState) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	state.PreviousProfileID = "official-profile"
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	return newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	}), state
}

func TestDisableForGlobalLogsOutWithoutRestoringOfficialProfile(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{
		ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL,
	}}
	local.profiles = []TsProfile{
		{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL},
		{
			ID:         state.PreviousProfileID,
			NodeID:     "official-node",
			ControlURL: "https://controlplane.tailscale.com",
		},
	}

	cloudCleanupFailed, err := manager.disableForGlobal(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("disableForGlobal() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{
		"local.currentProfile",
		"local.logout",
		"pairing.deleteNetwork",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("disableForGlobal events = %v, want %v", events, wantEvents)
	}
	if len(pairing.deleteNetworkTokens) != 1 || pairing.deleteNetworkTokens[0] != state.DeviceToken ||
		len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID {
		t.Fatalf(
			"cloud cleanup arguments = tokens=%v networks=%v",
			pairing.deleteNetworkTokens,
			pairing.deleteNetworkIDs,
		)
	}
	if len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf("region switch deleted only the local node: %v", pairing.deleteNodeIDs)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.NetworkID != "" || got.PreviousProfileID != "" {
		t.Fatalf("region switch retained managed identity: %+v", got)
	}
}

func TestDisableForGlobalReportsCloudNetworkDeleteFailureAndClearsIdentity(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	pairing.deleteNetworkErr = &sipeedPairingAPIError{
		HTTPStatus: http.StatusBadGateway,
		Code:       "upstream_error",
	}
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{
		ID: state.PreviousProfileID, NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com",
	}}

	cloudCleanupFailed, err := manager.disableForGlobal(context.Background())
	if err != nil || !cloudCleanupFailed {
		t.Fatalf("disableForGlobal() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	if len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID {
		t.Fatalf("network cleanup calls = %v", pairing.deleteNetworkIDs)
	}
	if len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf("cloud failure fell back to deleting one node: %v", pairing.deleteNodeIDs)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.NetworkID != "" {
		t.Fatalf("cloud failure did not clear local identity: %+v", got)
	}
}

func TestPrepareRegionChangeToGlobalPreservesManagedNetwork(t *testing.T) {
	previousMutations := tailscaleMutations
	tailscaleMutations = &mutationCoordinator{}
	t.Cleanup(func() { tailscaleMutations = previousMutations })

	local, pairing := successfulSipeedFakes(nil)
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{
		ID: state.PreviousProfileID, NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com",
	}}
	service := &Service{sipeed: manager}

	cloudCleanupFailed, release, err := service.PrepareRegionChange(
		context.Background(),
		proto.ApplicationRegionGlobal,
	)
	if err != nil || cloudCleanupFailed || release == nil {
		t.Fatalf(
			"PrepareRegionChange() = cleanupFailed=%v, release=%v, err=%v",
			cloudCleanupFailed,
			release != nil,
			err,
		)
	}
	if len(pairing.deleteNetworkIDs) != 0 {
		t.Fatalf("network cleanup calls = %v", pairing.deleteNetworkIDs)
	}
	if got := manager.snapshot(); got != state {
		t.Fatalf("region change modified managed identity: %+v", got)
	}
	if overlappingRelease, ok := tailscaleMutations.tryBegin(false); ok {
		overlappingRelease()
		t.Fatal("region coordinator released the Tailscale mutation guard too early")
	}

	release()
	afterRelease, ok := tailscaleMutations.tryBegin(false)
	if !ok {
		t.Fatal("region coordinator did not release the Tailscale mutation guard")
	}
	afterRelease()
}

func TestSipeedLogoutUsesFreshEmptyProfileBeforeCloudCleanup(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL}}
	local.profiles = []TsProfile{
		{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL},
		{ID: "empty-profile", ControlURL: "https://controlplane.tailscale.com"},
	}

	cloudCleanupFailed, err := manager.logout(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{
		"local.currentProfile",
		"local.logout",
		"local.newProfile",
		"local.profiles",
		"local.switchProfile:empty-profile",
		"pairing.deleteNetwork",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("logout events = %v, want %v", events, wantEvents)
	}
	if len(pairing.deleteNetworkTokens) != 1 || pairing.deleteNetworkTokens[0] != state.DeviceToken ||
		len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID {
		t.Fatalf(
			"cloud cleanup arguments = tokens=%v networks=%v",
			pairing.deleteNetworkTokens,
			pairing.deleteNetworkIDs,
		)
	}
	if len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf("logout deleted only the local node: %v", pairing.deleteNodeIDs)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.PreviousProfileID != "" {
		t.Fatalf("logout state still contains identity: %+v", got)
	}
}

func TestSipeedLogoutAcceptsAnonymousEmptyProfileFromLocalAPI(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{
		{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL},
		{},
	}
	local.profiles = nil

	cloudCleanupFailed, err := manager.logout(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{
		"local.currentProfile",
		"local.logout",
		"local.newProfile",
		"local.profiles",
		"local.currentProfile",
		"pairing.deleteNetwork",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("logout events = %v, want %v", events, wantEvents)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.PreviousProfileID != "" {
		t.Fatalf("logout state still contains identity: %+v", got)
	}
}

func TestSipeedLogoutReconcilesLocalAPITimeoutAfterProfileClears(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	local.logoutErr = context.DeadlineExceeded
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{
		{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL},
		{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL},
		{},
	}

	cloudCleanupFailed, err := manager.logout(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{
		"local.currentProfile",
		"local.logout",
		"local.currentProfile",
		"local.currentProfile",
		"pairing.deleteNetwork",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("logout reconciliation events = %v, want %v", events, wantEvents)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.PreviousProfileID != "" {
		t.Fatalf("reconciled logout state still contains identity: %+v", got)
	}
}

func TestSipeedLogoutRetryAcceptsExistingAnonymousProfile(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager, _ := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{}}

	cloudCleanupFailed, err := manager.logout(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{"local.currentProfile", "pairing.deleteNetwork"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("logout retry events = %v, want %v", events, wantEvents)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.PreviousProfileID != "" {
		t.Fatalf("logout retry state still contains identity: %+v", got)
	}
}

func TestSipeedLogoutPreservesCurrentOfficialProfile(t *testing.T) {
	var events []string
	local, pairing := successfulSipeedFakes(&events)
	manager, _ := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{
		ID: "current-official", NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com",
	}}

	cloudCleanupFailed, err := manager.logout(context.Background())
	if err != nil || cloudCleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cloudCleanupFailed, err)
	}
	wantEvents := []string{"local.currentProfile", "pairing.deleteNetwork"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("logout events = %v, want %v", events, wantEvents)
	}
	if len(local.switchCalls) != 0 {
		t.Fatalf("logout switched away from current official profile: %v", local.switchCalls)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.PreviousProfileID != "" {
		t.Fatalf("logout state still contains identity: %+v", got)
	}
}

func TestSipeedLogoutKeepsIdentityWhenLocalTransitionFails(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	local.logoutErr = errors.New("local logout failed")
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL}}

	if cleanupFailed, err := manager.logout(context.Background()); err == nil || cleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v; want local failure", cleanupFailed, err)
	}
	got := manager.snapshot()
	if got.ManagedState != proto.TailscaleManagedEnabledConnected || got.DeviceToken != state.DeviceToken {
		t.Fatalf("local failure discarded identity: %+v", got)
	}
	if len(pairing.deleteNetworkIDs) != 0 || len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf(
			"cloud cleanup ran after local failure: networks=%v nodes=%v",
			pairing.deleteNetworkIDs,
			pairing.deleteNodeIDs,
		)
	}
}

func TestSipeedLogoutClearsIdentityWhenCloudNetworkDeleteFails(t *testing.T) {
	local, pairing := successfulSipeedFakes(nil)
	pairing.deleteNetworkErr = &sipeedPairingAPIError{
		HTTPStatus: http.StatusBadGateway,
		Code:       "upstream_error",
	}
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL}}
	local.profiles = []TsProfile{{ID: "empty-profile", ControlURL: "https://controlplane.tailscale.com"}}

	if cleanupFailed, err := manager.logout(context.Background()); err != nil || !cleanupFailed {
		t.Fatalf("logout() = cleanupFailed=%v, err=%v", cleanupFailed, err)
	}
	if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled ||
		got.DeviceToken != "" || got.NetworkID != "" {
		t.Fatalf("cloud failure did not clear local identity: %+v", got)
	}
	if len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID {
		t.Fatalf("network cleanup calls = %v", pairing.deleteNetworkIDs)
	}
	if len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf("cloud failure fell back to deleting one node: %v", pairing.deleteNodeIDs)
	}
}
