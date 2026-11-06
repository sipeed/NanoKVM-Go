package tailscale

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/tailscalegate"
)

// Fakes and seams for the gate driver. The pairing fake implements the full
// management pairing contract with scripted responses; the clock and the wait
// seam make the auto-approve dwell deterministic without sleeping.

var (
	gateTestDeviceID     = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{5}, sipeedDeviceIDBytes))
	gateTestDeviceToken  = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, sipeedDeviceTokenBytes))
	gateTestPendingStamp = "2026-09-22T08:00:00Z"
)

type gateClock struct {
	mu  sync.Mutex
	now time.Time
}

func newGateClock() *gateClock {
	return &gateClock{now: time.Date(2026, time.September, 22, 8, 0, 0, 0, time.UTC)}
}

func (c *gateClock) time() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *gateClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// gateBlockingWait is a wait seam that signals when a wait starts and holds it
// until the test releases it, so auto-approve timing is fully deterministic.
type gateBlockingWait struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func newGateBlockingWait() *gateBlockingWait {
	return &gateBlockingWait{entered: make(chan struct{}), release: make(chan struct{})}
}

func (w *gateBlockingWait) wait(context.Context, time.Duration) error {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return nil
}

func gatePendingItem(id string, sas ...string) sipeedPendingRequest {
	return sipeedPendingRequest{PendingID: id, SAS: sas, CreatedAt: gateTestPendingStamp}
}

type fakeGatePairing struct {
	mu sync.Mutex

	claimCode    string
	claimExpiry  time.Time
	claimErr     error
	claimCalls   int
	claimStarted chan struct{}
	claimRelease chan struct{}

	pendingScript [][]sipeedPendingRequest
	pendingErr    error
	pendingCalls  int

	approveErr   error
	approveCalls int
	approveIDs   []string
	approveNames []string
	approved     chan struct{}

	devicesResp  *sipeedDevicesResponse
	devicesErr   error
	devicesCalls int
	devicesDone  chan struct{}

	enrollStarted chan struct{}
	enrollRelease chan struct{}
	enrollCalls   int
}

func (f *fakeGatePairing) ClaimCode(context.Context, string) (*sipeedClaimCodeResponse, error) {
	f.mu.Lock()
	f.claimCalls++
	started := f.claimStarted
	release := f.claimRelease
	if started != nil {
		f.claimStarted = nil
	}
	f.mu.Unlock()
	if started != nil {
		close(started)
	}
	if release != nil {
		<-release
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return &sipeedClaimCodeResponse{
		Code:      f.claimCode,
		ExpiresAt: f.claimExpiry.UTC().Format(time.RFC3339),
		SingleUse: true,
	}, nil
}

func (f *fakeGatePairing) Pending(context.Context, string, int) (*sipeedPendingResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pendingCalls++
	if f.pendingErr != nil {
		return nil, f.pendingErr
	}
	if len(f.pendingScript) == 0 {
		return &sipeedPendingResponse{}, nil
	}
	items := f.pendingScript[0]
	if len(f.pendingScript) > 1 {
		f.pendingScript = f.pendingScript[1:]
	}
	return &sipeedPendingResponse{Pending: append([]sipeedPendingRequest(nil), items...)}, nil
}

func (f *fakeGatePairing) ApprovePending(_ context.Context, _ string, pendingID string, name string) (*sipeedApproveResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.approveCalls++
	f.approveIDs = append(f.approveIDs, pendingID)
	f.approveNames = append(f.approveNames, name)
	if f.approved != nil {
		close(f.approved)
		f.approved = nil
	}
	if f.approveErr != nil {
		return nil, f.approveErr
	}
	return &sipeedApproveResponse{NodeID: "node-2", Name: "peer", IPs: []string{"100.64.0.4"}}, nil
}

func (f *fakeGatePairing) RejectPending(_ context.Context, _ string, pendingID string) (*sipeedRejectResponse, error) {
	return &sipeedRejectResponse{PendingID: pendingID, Rejected: true}, nil
}

func (f *fakeGatePairing) Authorize(context.Context, string, string, string, string) (*sipeedAuthorizeResponse, error) {
	return nil, errors.New("unexpected authorize call")
}

func (f *fakeGatePairing) Devices(context.Context, string) (*sipeedDevicesResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.devicesCalls++
	if f.devicesDone != nil {
		close(f.devicesDone)
		f.devicesDone = nil
	}
	if f.devicesErr != nil {
		return nil, f.devicesErr
	}
	if f.devicesResp == nil {
		return &sipeedDevicesResponse{}, nil
	}
	return f.devicesResp, nil
}

func (f *fakeGatePairing) RenameDevice(context.Context, string, string, string) (*sipeedRenameDeviceResponse, error) {
	return nil, errors.New("unexpected rename call")
}

func (f *fakeGatePairing) DeleteNetwork(context.Context, string, string) error { return nil }

func (f *fakeGatePairing) DeleteDevice(context.Context, string, string) error { return nil }

func (f *fakeGatePairing) Enroll(_ context.Context, deviceID string, _ string, _ string) (*sipeedEnrollResponse, error) {
	f.mu.Lock()
	f.enrollCalls++
	started := f.enrollStarted
	release := f.enrollRelease
	f.mu.Unlock()
	if started != nil {
		close(started)
	}
	if release != nil {
		<-release
	}
	return &sipeedEnrollResponse{
		NetworkID: "net-test",
		DeviceID:  deviceID,
		AuthKey:   "hskey-auth-memory-only",
		ExpiresAt: "2026-09-22T09:00:00Z",
	}, nil
}

func (f *fakeGatePairing) JoinKey(context.Context, string) (*sipeedJoinKeyResponse, error) {
	return &sipeedJoinKeyResponse{AuthKey: "hskey-auth-memory-only", ExpiresAt: "2026-09-22T09:00:00Z"}, nil
}

func (f *fakeGatePairing) ReportSelfNode(context.Context, string, string) (string, error) {
	return "node-1", nil
}

func (f *fakeGatePairing) counts() (claim, pending, approve, devices int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claimCalls, f.pendingCalls, f.approveCalls, f.devicesCalls
}

// gateRenewingPairing wraps the scripted fake and runs renew after a
// successful ApprovePending, while the approval is still in flight from the
// driver's point of view. It lets a test prove that the late invalidation does
// not clobber a claim code issued mid-approval. Every other method is promoted
// from the embedded fake.
type gateRenewingPairing struct {
	*fakeGatePairing
	renew func()
}

func (p *gateRenewingPairing) ApprovePending(
	ctx context.Context,
	token string,
	pendingID string,
	name string,
) (*sipeedApproveResponse, error) {
	response, err := p.fakeGatePairing.ApprovePending(ctx, token, pendingID, name)
	if err != nil {
		return nil, err
	}
	if p.renew != nil {
		p.renew()
	}
	return response, nil
}

type fakeGateProbe struct {
	status     *TsStatus
	statusErr  error
	profile    *TsProfile
	profileErr error
}

func (f *fakeGateProbe) Status() (*TsStatus, error) { return f.status, f.statusErr }

func (f *fakeGateProbe) CurrentProfile() (*TsProfile, error) { return f.profile, f.profileErr }

type fakeGateActionClient struct {
	events    *[]string
	mu        sync.Mutex
	stopCalls int
	stopErr   error
	stopBlock chan struct{}
	stopDone  chan struct{}
}

func (f *fakeGateActionClient) Start() error   { return nil }
func (f *fakeGateActionClient) Restart() error { return nil }
func (f *fakeGateActionClient) Up() error      { return nil }
func (f *fakeGateActionClient) Down() error    { return nil }
func (f *fakeGateActionClient) Logout() error  { return nil }

func (f *fakeGateActionClient) Stop() error {
	if f.stopBlock != nil {
		<-f.stopBlock
	}
	f.mu.Lock()
	f.stopCalls++
	f.mu.Unlock()
	if f.events != nil {
		*f.events = append(*f.events, "action.stop")
	}
	if f.stopDone != nil {
		close(f.stopDone)
	}
	return f.stopErr
}

func (f *fakeGateActionClient) stops() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopCalls
}

// gateDrive selects how a test drives the manager into an in-flight state
// after construction (such states cannot be seeded through the store file).
type gateDrive int

const (
	driveNone gateDrive = iota
	driveEnrolling
	driveConnecting
)

func gateConnectedState() sipeedPersistedState {
	return sipeedPersistedState{
		Version:      sipeedStateVersion,
		ManagedState: proto.TailscaleManagedEnabledConnected,
		Phase:        sipeedPhaseComplete,
		DeviceID:     gateTestDeviceID,
		DeviceToken:  gateTestDeviceToken,
		NetworkID:    "net-test",
		NodeID:       "node-1",
		ProfileID:    "prof-1",
		Hostname:     "nano-kvm",
		IP:           "100.64.0.3",
	}
}

// newGateTestService builds a Service whose manager is seeded with state. A
// disabled state simply leaves the store file absent. Only stable states may
// be seeded through the file: newSipeedManager normalizes a persisted
// in-flight state (enrolling/connecting) to degraded+interrupted on load, so
// tests needing those states must drive the manager methods instead
// (newGateEnrollingService / newGateConnectingService).
func newGateTestService(
	t *testing.T,
	state sipeedPersistedState,
	pairing *fakeGatePairing,
	local sipeedManagedLocalClient,
	action tailscaleActionClient,
) *Service {
	t.Helper()
	useTestServerFeatureFlags(t, true)
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json"))
	if state.Version != 0 && state.ManagedState != proto.TailscaleManagedDisabled {
		if err := store.save(state); err != nil {
			t.Fatalf("seed managed state: %v", err)
		}
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:          store,
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		install:        func() error { return nil },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	return &Service{
		sipeed:          manager,
		disclaimerStore: &fakeSipeedDisclaimerStore{accepted: true},
		regionStore:     useTestTailscaleRegion(t, proto.TailscaleRegionCN),
		newActionClient: func() tailscaleActionClient { return action },
	}
}

// newGateEnrollingService drives a fresh manager into the enrolling state the
// same way the enable job does, without running the async job.
func newGateEnrollingService(
	t *testing.T,
	pairing *fakeGatePairing,
	local sipeedManagedLocalClient,
	action tailscaleActionClient,
) *Service {
	t.Helper()
	service := newGateTestService(t, sipeedPersistedState{
		Version:      sipeedStateVersion,
		ManagedState: proto.TailscaleManagedDisabled,
	}, pairing, local, action)
	if err := service.sipeed.prepareEnable(); err != nil {
		t.Fatalf("prepareEnable: %v", err)
	}
	return service
}

// newGateConnectingService walks a fresh manager to enabledConnecting through
// the real persistence transitions of the enable job.
func newGateConnectingService(
	t *testing.T,
	pairing *fakeGatePairing,
	local sipeedManagedLocalClient,
	action tailscaleActionClient,
) *Service {
	t.Helper()
	service := newGateEnrollingService(t, pairing, local, action)
	if _, ok := service.sipeed.ensureIdentity("nano-kvm"); !ok {
		t.Fatal("ensureIdentity failed")
	}
	if !service.sipeed.persistEnrollment("net-test") {
		t.Fatal("persistEnrollment failed")
	}
	if !service.sipeed.persistConnecting("") {
		t.Fatal("persistConnecting failed")
	}
	return service
}

func newGateTestDriver(
	t *testing.T,
	service *Service,
	pairing sipeedNetworkPairingClient,
	clock *gateClock,
	waitFn func(context.Context, time.Duration) error,
	probe gateProbeClient,
) *GateDriver {
	t.Helper()
	if waitFn == nil {
		waitFn = func(context.Context, time.Duration) error { return nil }
	}
	return newGateDriver(gateDriverDeps{
		service: service,
		pairing: pairing,
		latch:   newGateLatchStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-gate.json")),
		now:     clock.time,
		wait:    waitFn,
		newProbe: func(context.Context) gateProbeClient {
			return probe
		},
	})
}

func gateReportField[T any](t *testing.T, value *T) T {
	t.Helper()
	if value == nil {
		t.Fatal("report field is nil")
	}
	return *value
}

func waitForSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func waitForGateApproval(t *testing.T, driver *GateDriver, approved <-chan struct{}) {
	t.Helper()
	waitForSignal(t, approved, "auto-approve request")

	// The fake signals inside ApprovePending, before the worker persists the
	// latch. Wait for the worker to finish before asserting or cleaning TempDir.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		driver.mu.Lock()
		queued := driver.approveQueuedID != ""
		driver.mu.Unlock()
		if !queued {
			if !driver.latch.paired(driver.service.sipeed.snapshot().NetworkID) {
				t.Fatal("auto-approve must persist the pairing latch")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for auto-approve completion")
}

// assertNoApprove polls briefly to prove no approval happened without sleeping
// the whole window.
func assertNoApprove(t *testing.T, pairing *fakeGatePairing) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, _, approve, _ := pairing.counts(); approve != 0 {
			t.Fatal("unexpected approve call")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// --- Snapshot ---

func TestGateDriverSnapshotMapsPersistedGates(t *testing.T) {
	cases := []struct {
		name       string
		state      sipeedPersistedState
		drive      gateDrive
		wantStart  bool
		wantStatus string
	}{
		{
			name:       "disabled",
			state:      sipeedPersistedState{Version: sipeedStateVersion, ManagedState: proto.TailscaleManagedDisabled},
			drive:      driveNone,
			wantStart:  false,
			wantStatus: string(tailscalegate.StatusOffline),
		},
		{
			name:       "enrolling",
			drive:      driveEnrolling,
			wantStart:  true,
			wantStatus: string(tailscalegate.StatusOffline),
		},
		{
			name:       "connecting",
			drive:      driveConnecting,
			wantStart:  true,
			wantStatus: string(tailscalegate.StatusOffline),
		},
		{
			name: "persisted degraded returns to start",
			state: sipeedPersistedState{
				Version:      sipeedStateVersion,
				ManagedState: proto.TailscaleManagedDegraded,
				ManagedError: "pairingRejected",
			},
			drive:      driveNone,
			wantStart:  false,
			wantStatus: string(tailscalegate.StatusOffline),
		},
		{
			name: "invalid identity returns to start",
			state: sipeedPersistedState{
				Version:      sipeedStateVersion,
				ManagedState: proto.TailscaleManagedInvalidIdentity,
				ManagedError: "invalidToken",
			},
			drive:      driveNone,
			wantStart:  false,
			wantStatus: string(tailscalegate.StatusOffline),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pairing := &fakeGatePairing{}
			service := newGateTestService(t, tc.state, pairing, nil, nil)
			switch tc.drive {
			case driveEnrolling:
				if err := service.sipeed.prepareEnable(); err != nil {
					t.Fatalf("prepareEnable: %v", err)
				}
			case driveConnecting:
				if _, ok := service.sipeed.ensureIdentity("nano-kvm"); !ok {
					t.Fatal("ensureIdentity failed")
				}
				if !service.sipeed.persistEnrollment("net-test") {
					t.Fatal("persistEnrollment failed")
				}
				if !service.sipeed.persistConnecting("") {
					t.Fatal("persistConnecting failed")
				}
			}
			driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

			report, err := driver.Snapshot(context.Background())
			if err != nil {
				t.Fatalf("Snapshot: %v", err)
			}
			if got := gateReportField(t, report.Start); got != tc.wantStart {
				t.Fatalf("start = %v, want %v", got, tc.wantStart)
			}
			if got := gateReportField(t, report.Match); got {
				t.Fatal("match = true, want false without a pairing latch")
			}
			if got := gateReportField(t, report.Status); got != tc.wantStatus {
				t.Fatalf("status = %q, want %q", got, tc.wantStatus)
			}
		})
	}
}

func TestGateDriverSnapshotTunnelProbe(t *testing.T) {
	onlineProbe := &fakeGateProbe{
		status:  testSipeedStatus("Running", "100.64.0.3"),
		profile: &TsProfile{ID: "prof-1", Name: "Sipeed", NodeID: "node-1", ControlURL: sipeedControlURL},
	}

	t.Run("online probe publishes identity", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, onlineProbe)
		driver.markPaired("net-test")

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if got := gateReportField(t, report.Status); got != string(tailscalegate.StatusOnline) {
			t.Fatalf("status = %q, want online", got)
		}
		if !gateReportField(t, report.Start) || !gateReportField(t, report.Match) {
			t.Fatalf("gates = %v/%v, want start/match true", *report.Start, *report.Match)
		}
		if got := gateReportField(t, report.Name); got != "nano-kvm" {
			t.Fatalf("name = %q", got)
		}
		if got := gateReportField(t, report.IP); got != "100.64.0.3" {
			t.Fatalf("ip = %q", got)
		}
	})

	t.Run("probe failure is offline but keeps the gates and identity", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil,
			&fakeGateProbe{statusErr: errors.New("local api down")})
		driver.markPaired("net-test")

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if got := gateReportField(t, report.Status); got != string(tailscalegate.StatusOffline) {
			t.Fatalf("status = %q, want offline", got)
		}
		if !gateReportField(t, report.Start) || !gateReportField(t, report.Match) {
			t.Fatal("a probe failure must not move the gates")
		}
		// The raw report still carries the persisted identity so the handler
		// can build the pairing directive while offline.
		if got := gateReportField(t, report.Name); got != "nano-kvm" {
			t.Fatalf("raw name = %q", got)
		}
		if got := gateReportField(t, report.IP); got != "100.64.0.3" {
			t.Fatalf("raw ip = %q", got)
		}
	})

	t.Run("non managed profile is offline", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil,
			&fakeGateProbe{
				status:  testSipeedStatus("Running", "100.64.0.3"),
				profile: &TsProfile{ID: "prof-2", ControlURL: "https://controlplane.tailscale.com"},
			})
		driver.markPaired("net-test")

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if got := gateReportField(t, report.Status); got != string(tailscalegate.StatusOffline) {
			t.Fatalf("status = %q, want offline for an official profile", got)
		}
	})

	t.Run("running without an address is offline", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil,
			&fakeGateProbe{
				status:  testSipeedStatus("Running"),
				profile: &TsProfile{ID: "prof-1", ControlURL: sipeedControlURL},
			})
		driver.markPaired("net-test")

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if got := gateReportField(t, report.Status); got != string(tailscalegate.StatusOffline) {
			t.Fatalf("status = %q, want offline without an IPv4", got)
		}
	})

	t.Run("not connected never probes", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateConnectingService(t, pairing, nil, nil)
		probeCalls := 0
		driver := newGateDriver(gateDriverDeps{
			service: service,
			pairing: pairing,
			latch:   newGateLatchStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-gate.json")),
			newProbe: func(context.Context) gateProbeClient {
				probeCalls++
				return onlineProbe
			},
		})

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if gateReportField(t, report.Status) != string(tailscalegate.StatusOffline) {
			t.Fatal("connecting state must be offline")
		}
		if probeCalls != 0 {
			t.Fatalf("probe calls = %d, want 0 before enabledConnected", probeCalls)
		}
	})
}

func TestGateDriverSnapshotDevicesProbe(t *testing.T) {
	t.Run("probe re-establishes the pairing latch", func(t *testing.T) {
		pairing := &fakeGatePairing{
			devicesResp: &sipeedDevicesResponse{Devices: []sipeedDevice{
				{NodeID: "node-1", Name: "nano-kvm", IsSelf: true},
				{NodeID: "node-2", Name: "phone"},
			}},
			devicesDone: make(chan struct{}),
		}
		devicesDone := pairing.devicesDone
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, &fakeGateProbe{})

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if gateReportField(t, report.Match) {
			t.Fatal("match must stay false until the probe proves a peer")
		}
		waitForSignal(t, devicesDone, "devices probe")
		deadline := time.Now().Add(2 * time.Second)
		for !driver.pairedFor("net-test") && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if !driver.pairedFor("net-test") {
			t.Fatal("devices probe must set the pairing latch when a peer exists")
		}

		report, err = driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if !gateReportField(t, report.Match) {
			t.Fatal("match must flip once the latch is set")
		}

		// A latched gate stops scheduling probes.
		_, _, _, devices := pairing.counts()
		if _, err := driver.Snapshot(context.Background()); err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		_, _, _, devicesAfter := pairing.counts()
		if devicesAfter != devices {
			t.Fatal("a latched gate must not keep probing the upstream")
		}
	})

	t.Run("self-only devices do not set the latch", func(t *testing.T) {
		pairing := &fakeGatePairing{
			devicesResp: &sipeedDevicesResponse{Devices: []sipeedDevice{
				{NodeID: "node-1", Name: "nano-kvm", IsSelf: true},
			}},
			devicesDone: make(chan struct{}),
		}
		devicesDone := pairing.devicesDone
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, &fakeGateProbe{})

		if _, err := driver.Snapshot(context.Background()); err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		waitForSignal(t, devicesDone, "devices probe")
		// Let the probe goroutine finish before judging the latch.
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			driver.probeMu.Lock()
			running := driver.probeRunning
			driver.probeMu.Unlock()
			if !running {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if driver.pairedFor("net-test") {
			t.Fatal("a network with only the self device must not latch the pairing")
		}
	})

	t.Run("retry-after suppresses the probe", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		if err := service.sipeed.setPairingRetryAfter(time.Minute); err != nil {
			t.Fatalf("set retry-after: %v", err)
		}
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, &fakeGateProbe{})

		if _, err := driver.Snapshot(context.Background()); err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			driver.probeMu.Lock()
			running := driver.probeRunning
			driver.probeMu.Unlock()
			if !running {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if _, _, _, devices := pairing.counts(); devices != 0 {
			t.Fatalf("devices calls = %d, want 0 inside Retry-After", devices)
		}
	})

	t.Run("a probe 429 persists the retry deadline", func(t *testing.T) {
		pairing := &fakeGatePairing{
			devicesErr: &sipeedPairingAPIError{
				HTTPStatus: http.StatusTooManyRequests,
				Code:       "rate_limited",
				RetryAfter: 5 * time.Minute,
			},
			devicesDone: make(chan struct{}),
		}
		devicesDone := pairing.devicesDone
		service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, &fakeGateProbe{})

		if _, err := driver.Snapshot(context.Background()); err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		waitForSignal(t, devicesDone, "devices probe")
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			driver.probeMu.Lock()
			running := driver.probeRunning
			driver.probeMu.Unlock()
			if !running {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if remaining := service.sipeed.pairingRetryRemaining(); remaining <= 0 {
			t.Fatal("a probe 429 must persist the Retry-After deadline")
		}
	})
}

// --- Initialize ---

func TestGateDriverInitialize(t *testing.T) {
	t.Run("disabled dispatches the async enable", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		pairing := &fakeGatePairing{
			enrollStarted: make(chan struct{}),
			enrollRelease: make(chan struct{}),
		}
		t.Cleanup(func() { close(pairing.enrollRelease) })
		local := &fakeSipeedLocalClient{
			statuses:        []*TsStatus{testSipeedStatus("NeedsLogin")},
			currentProfiles: []*TsProfile{{}},
		}
		service := newGateTestService(t, sipeedPersistedState{
			Version:      sipeedStateVersion,
			ManagedState: proto.TailscaleManagedDisabled,
		}, pairing, local, &fakeGateActionClient{})
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if err := driver.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize: %v", err)
		}
		if got := service.sipeed.snapshot().ManagedState; got != proto.TailscaleManagedEnrolling {
			t.Fatalf("managed state = %q, want enrolling", got)
		}
		waitForSignal(t, pairing.enrollStarted, "async enable job")
	})

	t.Run("in-flight enable is idempotent", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		pairing := &fakeGatePairing{}
		service := newGateConnectingService(t, pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if err := driver.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize: %v", err)
		}
		if _, _, _, devices := pairing.counts(); devices != 0 {
			t.Fatal("an idempotent initialize must not touch the upstream")
		}
		if _, ok := tailscaleMutations.tryBegin(false); !ok {
			t.Fatal("idempotent initialize must not take the mutation lock")
		}
	})

	t.Run("busy mutation lock answers BUSY", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		release, ok := tailscaleMutations.tryBegin(false)
		if !ok {
			t.Fatal("seed the mutation lock")
		}
		t.Cleanup(release)

		pairing := &fakeGatePairing{}
		service := newGateTestService(t, sipeedPersistedState{
			Version:      sipeedStateVersion,
			ManagedState: proto.TailscaleManagedDisabled,
		}, pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		err := driver.Initialize(context.Background())
		var messageErr tailscalegate.MessageError
		if !errors.As(err, &messageErr) || messageErr.GateMessage() != tailscalegate.MessageBusy {
			t.Fatalf("Initialize error = %v, want BUSY", err)
		}
	})

	t.Run("persisted rate limit answers RATELIMIT", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		pairing := &fakeGatePairing{}
		state := sipeedPersistedState{
			Version:      sipeedStateVersion,
			ManagedState: proto.TailscaleManagedDegraded,
			ManagedError: "rateLimited",
			RetryAfter:   time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		}
		service := newGateTestService(t, state, pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		err := driver.Initialize(context.Background())
		var messageErr tailscalegate.MessageError
		if !errors.As(err, &messageErr) || messageErr.GateMessage() != "RATELIMIT" {
			t.Fatalf("Initialize error = %v, want RATELIMIT", err)
		}
	})

	t.Run("unavailable mode fails closed", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		useTestServerFeatureFlags(t, false)
		pairing := &fakeGatePairing{}
		store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json"))
		manager := newSipeedManager(sipeedManagerDeps{store: store, pairing: pairing})
		service := &Service{
			sipeed:      manager,
			regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionGlobal),
		}
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if err := driver.Initialize(context.Background()); !errors.Is(err, errManagedModeUnavailable) {
			t.Fatalf("Initialize error = %v, want errManagedModeUnavailable", err)
		}
	})
}

// TestMapGateActionErrorConsent pins the defensive consent mapping: a start
// path that reaches the shared enable core without persisted consent must
// answer the stable CONSENT message on the gate wire instead of the generic
// FAIL fallback.
func TestMapGateActionErrorConsent(t *testing.T) {
	mapped := mapGateActionError(errSipeedDisclaimerRequired)
	var messageErr tailscalegate.MessageError
	if !errors.As(mapped, &messageErr) {
		t.Fatalf("mapped error %v does not implement MessageError", mapped)
	}
	if got := messageErr.GateMessage(); got != tailscalegate.MessageConsent {
		t.Fatalf("GateMessage() = %q, want %q", got, tailscalegate.MessageConsent)
	}
}

// --- Deinitialize ---

func TestGateDriverDeinitialize(t *testing.T) {
	logoutLocal := func() *fakeSipeedLocalClient {
		return &fakeSipeedLocalClient{
			currentProfiles: []*TsProfile{
				{ID: "prof-1", Name: "Sipeed", NodeID: "node-1", ControlURL: sipeedControlURL},
			},
			profiles: []TsProfile{{ID: "prof-9"}},
		}
	}

	t.Run("success runs logout then stop and clears pairing", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		events := []string{}
		pairing := &fakeGatePairing{}
		action := &fakeGateActionClient{events: &events}
		local := logoutLocal()
		local.events = &events
		service := newGateTestService(t, gateConnectedState(), pairing, local, action)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)
		driver.markPaired("net-test")

		if err := driver.Deinitialize(context.Background()); err != nil {
			t.Fatalf("Deinitialize: %v", err)
		}
		if got := service.sipeed.snapshot().ManagedState; got != proto.TailscaleManagedDisabled {
			t.Fatalf("managed state = %q, want disabled", got)
		}
		if action.stops() != 1 {
			t.Fatalf("stop calls = %d, want 1", action.stops())
		}
		if driver.pairedFor("net-test") {
			t.Fatal("pairing latch must be cleared after deinitialize")
		}
		logoutAt, stopAt := -1, -1
		for i, event := range events {
			if event == "local.logout" && logoutAt < 0 {
				logoutAt = i
			}
			if event == "action.stop" {
				stopAt = i
			}
		}
		if logoutAt < 0 || stopAt < 0 || logoutAt > stopAt {
			t.Fatalf("events %v: logout must precede stop", events)
		}

		report, err := driver.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if gateReportField(t, report.Start) || gateReportField(t, report.Match) {
			t.Fatal("gate must be fully reset after deinitialize")
		}
	})

	t.Run("logout failure keeps the identity for retry", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		local := logoutLocal()
		local.logoutErr = errors.New("local api refused")
		pairing := &fakeGatePairing{}
		action := &fakeGateActionClient{}
		service := newGateTestService(t, gateConnectedState(), pairing, local, action)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		err := driver.Deinitialize(context.Background())
		if err == nil || errors.Is(err, errTailscaleMutationBusy) {
			t.Fatalf("Deinitialize error = %v, want a plain failure", err)
		}
		if action.stops() != 0 {
			t.Fatal("stop must not run when the logout failed")
		}
		if got := service.sipeed.snapshot().ManagedState; got != proto.TailscaleManagedEnabledConnected {
			t.Fatalf("managed state = %q, want enabledConnected preserved for retry", got)
		}
	})

	t.Run("budget timeout returns the context error while work continues", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		pairing := &fakeGatePairing{}
		action := &fakeGateActionClient{
			stopBlock: make(chan struct{}),
			stopDone:  make(chan struct{}),
		}
		local := logoutLocal()
		captured := make(chan context.Context, 1)
		service := newGateTestService(t, gateConnectedState(), pairing, local, action)
		service.sipeed.deps.newLocalClient = func(ctx context.Context) sipeedManagedLocalClient {
			select {
			case captured <- ctx:
			default:
			}
			return local
		}
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err := driver.Deinitialize(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Deinitialize error = %v, want context deadline", err)
		}

		// The detached teardown's context must survive the caller's timeout:
		// cancellation here would leave a half-applied logout.
		var workCtx context.Context
		select {
		case workCtx = <-captured:
		case <-time.After(2 * time.Second):
			t.Fatal("teardown never reached the local client")
		}
		if workCtx.Err() != nil {
			t.Fatalf("detached teardown context was cancelled: %v", workCtx.Err())
		}

		// The detached teardown is still running: the mutation lock is held and
		// a retry answers BUSY.
		if err := driver.Deinitialize(context.Background()); err == nil {
			t.Fatal("retry during teardown must fail")
		} else {
			var messageErr tailscalegate.MessageError
			if !errors.As(err, &messageErr) || messageErr.GateMessage() != tailscalegate.MessageBusy {
				t.Fatalf("retry error = %v, want BUSY", err)
			}
		}

		close(action.stopBlock)
		waitForSignal(t, action.stopDone, "detached stop")
		deadline := time.Now().Add(2 * time.Second)
		for service.sipeed.snapshot().ManagedState != proto.TailscaleManagedDisabled && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if got := service.sipeed.snapshot().ManagedState; got != proto.TailscaleManagedDisabled {
			t.Fatalf("managed state = %q, want disabled after detached teardown", got)
		}
	})

	t.Run("busy mutation lock answers BUSY", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		release, ok := tailscaleMutations.tryBegin(false)
		if !ok {
			t.Fatal("seed the mutation lock")
		}
		t.Cleanup(release)

		pairing := &fakeGatePairing{}
		service := newGateTestService(t, gateConnectedState(), pairing, logoutLocal(), &fakeGateActionClient{})
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		err := driver.Deinitialize(context.Background())
		var messageErr tailscalegate.MessageError
		if !errors.As(err, &messageErr) || messageErr.GateMessage() != tailscalegate.MessageBusy {
			t.Fatalf("Deinitialize error = %v, want BUSY", err)
		}
	})
}

// --- LoginCode and auto-approve ---

func TestGateDriverLoginCodeClaim(t *testing.T) {
	t.Run("fetches once and caches until the renew skew", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateConnectingService(t, pairing, nil, nil)
		clock := newGateClock()
		driver := newGateTestDriver(t, service, pairing, clock, nil, nil)
		pairing.claimCode = "123 456"
		// The first fetched code expires inside the renew skew, so the second
		// fetch renews instead of serving the cache. The shared cache owner is
		// the manager, whose clock is real time in these tests.
		pairing.claimExpiry = time.Now().Add(10 * time.Second)

		code, err := driver.LoginCode(context.Background())
		if err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		if code != "123456" {
			t.Fatalf("code = %q, want the bare six digits", code)
		}
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		if claim, _, _, _ := pairing.counts(); claim != 2 {
			t.Fatalf("claim calls = %d, want 2 (renewed inside skew)", claim)
		}

		// A fresh expiry is cached and served without an upstream call.
		pairing.mu.Lock()
		pairing.claimExpiry = time.Now().Add(5 * time.Minute)
		pairing.mu.Unlock()
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		if claim, _, _, _ := pairing.counts(); claim != 3 {
			t.Fatalf("claim calls = %d, want 3 (third fetch was cached)", claim)
		}
	})

	t.Run("identity not ready is a not-ready failure", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateTestService(t, sipeedPersistedState{
			Version:      sipeedStateVersion,
			ManagedState: proto.TailscaleManagedEnrolling,
		}, pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if _, err := driver.LoginCode(context.Background()); !errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
			t.Fatalf("LoginCode error = %v, want ErrLoginCodeNotReady", err)
		}
		if claim, pending, _, _ := pairing.counts(); claim+pending != 0 {
			t.Fatal("no upstream call may happen before enrollment")
		}
	})

	t.Run("persisted retry-after blocks upstream calls", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		service := newGateConnectingService(t, pairing, nil, nil)
		if err := service.sipeed.setPairingRetryAfter(time.Minute); err != nil {
			t.Fatalf("set retry-after: %v", err)
		}
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if _, err := driver.LoginCode(context.Background()); !errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
			t.Fatalf("LoginCode error = %v, want ErrLoginCodeNotReady", err)
		}
		if claim, pending, _, _ := pairing.counts(); claim+pending != 0 {
			t.Fatal("Retry-After window must not reach the upstream")
		}
	})

	t.Run("upstream failures map to not-ready and persist retry-after", func(t *testing.T) {
		pairing := &fakeGatePairing{
			pendingErr: &sipeedPairingAPIError{
				HTTPStatus: http.StatusTooManyRequests,
				Code:       "rate_limited",
				RetryAfter: 5 * time.Minute,
			},
		}
		service := newGateConnectingService(t, pairing, nil, nil)
		driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

		if _, err := driver.LoginCode(context.Background()); !errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
			t.Fatalf("LoginCode error = %v, want ErrLoginCodeNotReady", err)
		}
		if remaining := service.sipeed.pairingRetryRemaining(); remaining <= 0 {
			t.Fatal("the upstream 429 must persist a Retry-After deadline")
		}
	})

	t.Run("a malformed claim code is a real failure", func(t *testing.T) {
		pairing := &fakeGatePairing{claimCode: "12"}
		service := newGateConnectingService(t, pairing, nil, nil)
		clock := newGateClock()
		pairing.claimExpiry = time.Now().Add(5 * time.Minute)
		driver := newGateTestDriver(t, service, pairing, clock, nil, nil)

		_, err := driver.LoginCode(context.Background())
		if err == nil || errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
			t.Fatalf("LoginCode error = %v, want a contract violation failure", err)
		}
	})
}

func TestGateDriverLoginCodeServesSASAndAutoApproves(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	pairing := &fakeGatePairing{
		pendingScript: [][]sipeedPendingRequest{{gatePendingItem("pending-1", "4", "6", "8")}},
		approved:      make(chan struct{}),
	}
	approved := pairing.approved
	service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
	clock := newGateClock()
	driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)

	code, err := driver.LoginCode(context.Background())
	if err != nil {
		t.Fatalf("LoginCode: %v", err)
	}
	if code != "468" {
		t.Fatalf("code = %q, want the joined three-digit SAS", code)
	}

	// The approval waits for the dwell; it fires only after the SAS has been
	// displayed long enough.
	waitForSignal(t, wait.entered, "auto-approve dwell")
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)

	waitForGateApproval(t, driver, approved)
	_, _, approve, _ := pairing.counts()
	if approve != 1 {
		t.Fatalf("approve calls = %d, want 1", approve)
	}
	pairing.mu.Lock()
	if pairing.approveIDs[0] != "pending-1" || pairing.approveNames[0] != "" {
		t.Fatalf("approve = %v/%v, want pending-1 with an empty name", pairing.approveIDs, pairing.approveNames)
	}
	pairing.mu.Unlock()
	if !driver.pairedFor("net-test") {
		t.Fatal("approval must set the pairing latch")
	}

	// The next snapshot reports the completed pairing.
	report, err := driver.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !gateReportField(t, report.Match) {
		t.Fatal("match must flip after the peer pairing completed")
	}
}

// gateClaimSnapshot reads the shared claim-code cache under the manager lock.
func gateClaimSnapshot(service *Service) *sipeedClaimCodeCacheEntry {
	service.sipeed.mu.Lock()
	defer service.sipeed.mu.Unlock()
	return service.sipeed.claim
}

// A completed auto-approve consumes the claim code it presented, so the shared
// cache must be dropped: neither the LCD nor the web may keep displaying a code
// that the approval has already spent.
func TestGateDriverAutoApproveInvalidatesClaimCode(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	pairing := &fakeGatePairing{
		claimCode:   "123 456",
		claimExpiry: time.Now().Add(5 * time.Minute),
		pendingScript: [][]sipeedPendingRequest{
			{}, // no pending: the six-digit claim-code round
			{gatePendingItem("pending-1", "4", "6", "8")},
		},
		approved: make(chan struct{}),
	}
	approved := pairing.approved
	service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
	clock := newGateClock()
	driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)

	// Seed the shared cache through the driver's own claim-code lane and prove
	// the pre-approval entry is really there.
	code, err := driver.LoginCode(context.Background())
	if err != nil {
		t.Fatalf("LoginCode claim round: %v", err)
	}
	if code != "123456" {
		t.Fatalf("claim code = %q, want 123456", code)
	}
	if cached := gateClaimSnapshot(service); cached == nil || cached.code != "123 456" {
		t.Fatalf("shared cache must hold the pre-approval code, got %+v", cached)
	}

	// The single pending arms the auto-approval.
	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode SAS round: %v", err)
	}
	waitForSignal(t, wait.entered, "auto-approve dwell")
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)
	waitForGateApproval(t, driver, approved)

	if cached := gateClaimSnapshot(service); cached != nil {
		t.Fatalf("auto-approve must drop the pre-approval claim code, still cached %+v", cached)
	}
	if _, _, approve, _ := pairing.counts(); approve != 1 {
		t.Fatalf("approve calls = %d, want 1", approve)
	}
}

// A claim code renewed while the approval was in flight must survive the
// post-approval invalidation: the invalidator is pointer-guarded and may only
// drop the exact entry captured before the mutation, mirroring the web
// approve path.
func TestGateDriverAutoApprovePreservesRenewedClaimCode(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	pairing := &fakeGatePairing{
		claimCode:   "123 456",
		claimExpiry: time.Now().Add(5 * time.Minute),
		pendingScript: [][]sipeedPendingRequest{
			{}, // no pending: the six-digit claim-code round
			{gatePendingItem("pending-1", "4", "6", "8")},
		},
		approved: make(chan struct{}),
	}
	approved := pairing.approved
	service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
	clock := newGateClock()

	// The wrapped approval renews the shared code through the manager while the
	// driver call is still in flight, i.e. after the invalidator captured the
	// pre-approval entry but before it fires.
	renewal := make(chan error, 1)
	driverPairing := &gateRenewingPairing{
		fakeGatePairing: pairing,
		renew: func() {
			_, err := service.sipeed.getClaimCode(context.Background(), true)
			renewal <- err
		},
	}
	driver := newGateTestDriver(t, service, driverPairing, clock, wait.wait, nil)

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode claim round: %v", err)
	}
	if cached := gateClaimSnapshot(service); cached == nil || cached.code != "123 456" {
		t.Fatalf("shared cache must hold the pre-approval code, got %+v", cached)
	}
	pairing.mu.Lock()
	pairing.claimCode = "654 321"
	pairing.mu.Unlock()

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode SAS round: %v", err)
	}
	waitForSignal(t, wait.entered, "auto-approve dwell")
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)
	waitForGateApproval(t, driver, approved)

	if err := <-renewal; err != nil {
		t.Fatalf("mid-approval renewal: %v", err)
	}
	cached := gateClaimSnapshot(service)
	if cached == nil || cached.code != "654 321" {
		t.Fatalf("renewed claim code must survive the late invalidation, got %+v", cached)
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2 (seed + renewal)", claim)
	}
}

// A failed auto-approve must leave the shared claim-code cache untouched: the
// invalidator is captured before the mutation but only ever fires after a
// successful approve, so a code is never dropped for an approval that did not
// happen, and a retry can still display it.
func TestGateDriverAutoApproveFailureKeepsClaimCode(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	pairing := &fakeGatePairing{
		claimCode:   "123 456",
		claimExpiry: time.Now().Add(5 * time.Minute),
		pendingScript: [][]sipeedPendingRequest{
			{}, // no pending: the six-digit claim-code round
			{gatePendingItem("pending-1", "4", "6", "8")},
		},
		approveErr: errors.New("upstream rejected the approval"),
		approved:   make(chan struct{}),
	}
	approved := pairing.approved
	service := newGateTestService(t, gateConnectedState(), pairing, nil, nil)
	clock := newGateClock()
	driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode claim round: %v", err)
	}
	if cached := gateClaimSnapshot(service); cached == nil || cached.code != "123 456" {
		t.Fatalf("shared cache must hold the pre-approval code, got %+v", cached)
	}

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode SAS round: %v", err)
	}
	waitForSignal(t, wait.entered, "auto-approve dwell")
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)
	waitForSignal(t, approved, "auto-approve request")

	// The fake signals inside ApprovePending, before the worker re-arms. Wait
	// for the worker to finish the failed attempt before asserting.
	deadline := time.Now().Add(2 * time.Second)
	for {
		driver.mu.Lock()
		queued := driver.approveQueuedID != ""
		driver.mu.Unlock()
		if !queued {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the failed auto-approve to finish")
		}
		time.Sleep(time.Millisecond)
	}

	if _, _, approve, _ := pairing.counts(); approve != 1 {
		t.Fatalf("approve calls = %d, want 1", approve)
	}
	if cached := gateClaimSnapshot(service); cached == nil || cached.code != "123 456" {
		t.Fatalf("a failed approve must keep the cached claim code, got %+v", cached)
	}
	if driver.pairedFor("net-test") {
		t.Fatal("a failed approve must not set the pairing latch")
	}
}

func TestGateDriverAutoApproveGuards(t *testing.T) {
	setup := func(t *testing.T, script [][]sipeedPendingRequest) (*fakeGatePairing, *GateDriver, *gateClock, *gateBlockingWait, chan struct{}) {
		t.Helper()
		isolatedMutationCoordinator(t)
		wait := newGateBlockingWait()
		approved := make(chan struct{})
		pairing := &fakeGatePairing{pendingScript: script, approved: approved}
		service := newGateConnectingService(t, pairing, nil, nil)
		clock := newGateClock()
		driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)
		return pairing, driver, clock, wait, approved
	}

	t.Run("no approval without a served SAS", func(t *testing.T) {
		pairing, _, _, _, _ := setup(t, [][]sipeedPendingRequest{{gatePendingItem("pending-1", "4", "6", "8")}})
		assertNoApprove(t, pairing)
	})

	t.Run("time-of-check re-read rejects a changed pending", func(t *testing.T) {
		pairing, driver, clock, wait, _ := setup(t, [][]sipeedPendingRequest{
			{gatePendingItem("pending-1", "4", "6", "8")},
			{gatePendingItem("pending-2", "1", "2", "3")},
		})
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		waitForSignal(t, wait.entered, "auto-approve dwell")
		clock.advance(gateApproveDwell + time.Second)
		close(wait.release)
		assertNoApprove(t, pairing)
	})

	t.Run("multiple pendings fail the fetch and approve nothing", func(t *testing.T) {
		// Documented fail-closed choice: with more than one pending the lane
		// cannot know which one the LCD would show, so it answers 503 (keeping
		// the previous display) and never approves.
		pairing, driver, _, _, _ := setup(t, [][]sipeedPendingRequest{{
			gatePendingItem("pending-1", "4", "6", "8"),
			gatePendingItem("pending-2", "1", "2", "3"),
		}})
		_, err := driver.LoginCode(context.Background())
		if !errors.Is(err, errGateAmbiguousPending) {
			t.Fatalf("LoginCode error = %v, want errGateAmbiguousPending", err)
		}
		assertNoApprove(t, pairing)
	})

	t.Run("an abandoned display is never approved", func(t *testing.T) {
		pairing, driver, clock, wait, _ := setup(t, [][]sipeedPendingRequest{
			{gatePendingItem("pending-1", "4", "6", "8")},
		})
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		waitForSignal(t, wait.entered, "auto-approve dwell")
		// The dwell passed but the SAS has not been served for longer than the
		// recency window (the user walked away mid-pairing).
		clock.advance(gateApproveDwell + gateApproveRecency + time.Second)
		close(wait.release)
		assertNoApprove(t, pairing)
	})

	t.Run("session teardown cancels the queued approval", func(t *testing.T) {
		pairing, driver, clock, wait, _ := setup(t, [][]sipeedPendingRequest{
			{gatePendingItem("pending-1", "4", "6", "8")},
		})
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		waitForSignal(t, wait.entered, "auto-approve dwell")
		if err := driver.EndLoginSession(context.Background()); err != nil {
			t.Fatalf("EndLoginSession: %v", err)
		}
		clock.advance(gateApproveDwell + time.Second)
		close(wait.release)
		assertNoApprove(t, pairing)
	})

	t.Run("the dwell is mandatory", func(t *testing.T) {
		pairing, driver, clock, wait, _ := setup(t, [][]sipeedPendingRequest{
			{gatePendingItem("pending-1", "4", "6", "8")},
		})
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		waitForSignal(t, wait.entered, "auto-approve dwell")
		clock.advance(gateApproveDwell - time.Second)
		close(wait.release)
		assertNoApprove(t, pairing)
	})

	t.Run("an active retry-after blocks the approval", func(t *testing.T) {
		pairing, driver, clock, wait, _ := setup(t, [][]sipeedPendingRequest{
			{gatePendingItem("pending-1", "4", "6", "8")},
		})
		if _, err := driver.LoginCode(context.Background()); err != nil {
			t.Fatalf("LoginCode: %v", err)
		}
		waitForSignal(t, wait.entered, "auto-approve dwell")
		if err := driver.service.sipeed.setPairingRetryAfter(time.Minute); err != nil {
			t.Fatalf("set retry-after: %v", err)
		}
		clock.advance(gateApproveDwell + time.Second)
		close(wait.release)
		assertNoApprove(t, pairing)
	})
}

func TestGateDriverEndLoginSession(t *testing.T) {
	pairing := &fakeGatePairing{}
	service := newGateConnectingService(t, pairing, nil, nil)
	clock := newGateClock()
	pairing.claimCode = "123 456"
	pairing.claimExpiry = time.Now().Add(5 * time.Minute)
	driver := newGateTestDriver(t, service, pairing, clock, nil, nil)

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode: %v", err)
	}
	if err := driver.EndLoginSession(context.Background()); err != nil {
		t.Fatalf("EndLoginSession: %v", err)
	}
	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode after teardown: %v", err)
	}
	// The claim code lives in the shared manager cache and stays valid for
	// every lane: ending the LCD session must not rotate the code the web
	// may be displaying.
	if claim, _, _, _ := pairing.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (shared code survives the session end)", claim)
	}
	if got := service.sipeed.snapshot().ManagedState; got != proto.TailscaleManagedEnabledConnecting {
		t.Fatalf("managed state = %q, want untouched connecting", got)
	}
}

// An attempt aborted by a transient condition must not permanently consume the
// pairing: the next serve of the same pending queues a fresh evaluation.
func TestGateDriverAutoApproveRearmsAfterAbort(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	approved := make(chan struct{})
	pairing := &fakeGatePairing{
		pendingScript: [][]sipeedPendingRequest{{gatePendingItem("pending-1", "4", "6", "8")}},
		approved:      approved,
	}
	service := newGateConnectingService(t, pairing, nil, nil)
	clock := newGateClock()
	driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode: %v", err)
	}
	waitForSignal(t, wait.entered, "auto-approve dwell")

	// Abort the first attempt with an active Retry-After window.
	if err := service.sipeed.setPairingRetryAfter(time.Minute); err != nil {
		t.Fatalf("set retry-after: %v", err)
	}
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)
	assertNoApprove(t, pairing)

	// The window passes and the same pending is still displayed: the next serve
	// re-arms and approves (the wait seam is already released, and the dwell is
	// anchored at the first serve).
	if err := service.sipeed.setPairingRetryAfter(0); err != nil {
		t.Fatalf("clear retry-after: %v", err)
	}
	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode: %v", err)
	}
	waitForGateApproval(t, driver, approved)
	if !driver.pairedFor("net-test") {
		t.Fatal("the re-armed approval must set the pairing latch")
	}
}

// A typed upstream rejection (an invalid identity, a protocol error) is not a
// transient not-ready: it must surface as a warn-level failure so the fault is
// observable instead of silently retried forever.
func TestGateDriverLoginCodeTypedFailuresAreVisible(t *testing.T) {
	pairing := &fakeGatePairing{
		pendingErr: &sipeedPairingAPIError{HTTPStatus: http.StatusUnauthorized, Code: "invalid_token"},
	}
	service := newGateConnectingService(t, pairing, nil, nil)
	driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

	_, err := driver.LoginCode(context.Background())
	if err == nil || errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
		t.Fatalf("LoginCode error = %v, want the raw typed api error", err)
	}
	var apiError *sipeedPairingAPIError
	if !errors.As(err, &apiError) || apiError.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("LoginCode error = %v, want the 401 api error", err)
	}
}

// R3.4: a restarted driver over the same latch store must report the
// completed pairing on its first snapshot, without any upstream call.
func TestGateDriverLatchRecoversAcrossInstances(t *testing.T) {
	clock := newGateClock()
	latchPath := filepath.Join(t.TempDir(), "remote-network", "tailscale-gate.json")

	pairingA := &fakeGatePairing{}
	serviceA := newGateTestService(t, gateConnectedState(), pairingA, nil, nil)
	driverA := newGateDriver(gateDriverDeps{
		service: serviceA,
		pairing: pairingA,
		latch:   newGateLatchStore(latchPath),
		now:     clock.time,
	})
	driverA.markPaired("net-test")

	// A fresh driver instance (the post-restart process) over the same store.
	pairingB := &fakeGatePairing{}
	serviceB := newGateTestService(t, gateConnectedState(), pairingB, nil, nil)
	driverB := newGateDriver(gateDriverDeps{
		service: serviceB,
		pairing: pairingB,
		latch:   newGateLatchStore(latchPath),
		now:     clock.time,
	})

	report, err := driverB.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !gateReportField(t, report.Match) {
		t.Fatal("a restarted driver must recover the pairing latch on the first snapshot")
	}
	if _, _, _, devices := pairingB.counts(); devices != 0 {
		t.Fatal("a recovered latch must not trigger a devices probe")
	}

	// A latch keyed by a different network is invalid for this one.
	pairingC := &fakeGatePairing{}
	serviceC := newGateTestService(t, gateConnectedState(), pairingC, nil, nil)
	if err := serviceC.sipeed.clearIdentity(); err != nil {
		t.Fatalf("clear identity: %v", err)
	}
	driverC := newGateDriver(gateDriverDeps{
		service: serviceC,
		pairing: pairingC,
		latch:   newGateLatchStore(latchPath),
		now:     clock.time,
	})
	report, err = driverC.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if gateReportField(t, report.Start) || gateReportField(t, report.Match) {
		t.Fatal("a cleared identity must reset both gates regardless of the latch file")
	}
}

// The full code-lane walk inside one session: the six-digit claim code is
// served while nobody is joining, and the three-digit SAS takes over the
// moment the joiner's pending request appears.
func TestGateDriverLoginCodeWalksClaimThenSAS(t *testing.T) {
	isolatedMutationCoordinator(t)
	wait := newGateBlockingWait()
	approved := make(chan struct{})
	pairing := &fakeGatePairing{
		// First poll: no pending. Second and later polls: one pending.
		pendingScript: [][]sipeedPendingRequest{
			{},
			{gatePendingItem("pending-1", "4", "6", "8")},
		},
		approved: approved,
	}
	service := newGateConnectingService(t, pairing, nil, nil)
	clock := newGateClock()
	pairing.claimCode = "123 456"
	pairing.claimExpiry = time.Now().Add(5 * time.Minute)
	driver := newGateTestDriver(t, service, pairing, clock, wait.wait, nil)

	code, err := driver.LoginCode(context.Background())
	if err != nil {
		t.Fatalf("LoginCode (claim): %v", err)
	}
	if code != "123456" {
		t.Fatalf("first code = %q, want the six-digit claim code", code)
	}

	code, err = driver.LoginCode(context.Background())
	if err != nil {
		t.Fatalf("LoginCode (sas): %v", err)
	}
	if code != "468" {
		t.Fatalf("second code = %q, want the three-digit SAS", code)
	}

	// The armed approval completes the ceremony.
	waitForSignal(t, wait.entered, "auto-approve dwell")
	clock.advance(gateApproveDwell + time.Second)
	close(wait.release)
	waitForGateApproval(t, driver, approved)
}

func TestNewGateDriver(t *testing.T) {
	t.Run("nil service fails closed", func(t *testing.T) {
		if _, err := NewGateDriver(nil); err == nil {
			t.Fatal("a nil service must fail")
		}
	})

	t.Run("narrow pairing client fails closed", func(t *testing.T) {
		manager := newSipeedManager(sipeedManagerDeps{
			store:   newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")),
			pairing: &fakeSipeedPairingClient{},
		})
		if _, err := NewGateDriver(&Service{sipeed: manager}); err == nil {
			t.Fatal("a pairing client without the management contract must fail")
		}
	})

	t.Run("production shape builds with defaults", func(t *testing.T) {
		pairing := &fakeGatePairing{}
		manager := newSipeedManager(sipeedManagerDeps{
			store:   newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")),
			pairing: pairing,
		})
		driver, err := NewGateDriver(&Service{sipeed: manager})
		if err != nil {
			t.Fatalf("NewGateDriver: %v", err)
		}
		if driver.now == nil || driver.wait == nil || driver.newProbe == nil || driver.latch == nil {
			t.Fatal("production defaults were not filled")
		}
		if driver.deinitBudget != gateDeinitializeBudget || driver.approveDwell != gateApproveDwell {
			t.Fatal("production budgets were not filled")
		}
	})
}
