package tailscale

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

// manifestGatePairing counts every upstream pairing call so a refused Sipeed
// endpoint can prove it never reached the cloud boundary.
type manifestGatePairing struct {
	*fakeSipeedManagementPairing

	mu    sync.Mutex
	calls int
}

func newManifestGatePairing() *manifestGatePairing {
	return &manifestGatePairing{fakeSipeedManagementPairing: newFakeSipeedManagementPairing()}
}

func (fake *manifestGatePairing) record() {
	fake.mu.Lock()
	fake.calls++
	fake.mu.Unlock()
}

func (fake *manifestGatePairing) callCount() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.calls
}

func (fake *manifestGatePairing) ClaimCode(ctx context.Context, token string) (*sipeedClaimCodeResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.ClaimCode(ctx, token)
}

func (fake *manifestGatePairing) Pending(ctx context.Context, token string, wait int) (*sipeedPendingResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.Pending(ctx, token, wait)
}

func (fake *manifestGatePairing) ApprovePending(ctx context.Context, token, pendingID, name string) (*sipeedApproveResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.ApprovePending(ctx, token, pendingID, name)
}

func (fake *manifestGatePairing) RejectPending(ctx context.Context, token, pendingID string) (*sipeedRejectResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.RejectPending(ctx, token, pendingID)
}

func (fake *manifestGatePairing) Authorize(ctx context.Context, token, code, authID, name string) (*sipeedAuthorizeResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.Authorize(ctx, token, code, authID, name)
}

func (fake *manifestGatePairing) Devices(ctx context.Context, token string) (*sipeedDevicesResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.Devices(ctx, token)
}

func (fake *manifestGatePairing) RenameDevice(ctx context.Context, token, nodeID, name string) (*sipeedRenameDeviceResponse, error) {
	fake.record()
	return fake.fakeSipeedManagementPairing.RenameDevice(ctx, token, nodeID, name)
}

func (fake *manifestGatePairing) DeleteDevice(ctx context.Context, token, nodeID string) error {
	fake.record()
	return fake.fakeSipeedManagementPairing.DeleteDevice(ctx, token, nodeID)
}

func (fake *manifestGatePairing) DeleteNetwork(ctx context.Context, token, networkID string) error {
	fake.record()
	return fake.fakeSipeedManagementPairing.DeleteNetwork(ctx, token, networkID)
}

// newManifestGateService builds a cn-region service with an existing managed
// identity record but the manifest switch off, and returns the state file path
// so a test can prove the refusal wrote nothing.
func newManifestGateService(t *testing.T, pairing *manifestGatePairing) (*Service, string) {
	t.Helper()
	useTestServerFeatureFlags(t, false)
	disableTailscaleChallengesForTest(t)

	path := filepath.Join(t.TempDir(), "remote-network", "state.json")
	store := newSipeedStateStore(path)
	if err := store.save(validTestSipeedState(t)); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:       store,
		pairing:     pairing,
		isInstalled: func() bool { return false },
		now:         time.Now,
	})
	return &Service{
		sipeed:      manager,
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
	}, path
}

func readManifestGateStateFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	return string(data)
}

// TestSipeedEndpointsRefuseWhenManifestDisabled walks every registered Sipeed
// endpoint in the behavior matrix's cn + manifest-off row. Each one must answer
// the existing -1 "managed tailscale mode is unavailable" envelope, leave the
// persisted identity untouched, and make no upstream pairing call.
func TestSipeedEndpointsRefuseWhenManifestDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		method string
		target string
		body   string
		params gin.Params
		invoke func(*Service, *gin.Context)
	}{
		{
			name:   "enable",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/enable",
			body:   `{}`,
			invoke: func(s *Service, c *gin.Context) { s.EnableSipeed(c) },
		},
		{
			name:   "logout",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/logout",
			body:   `{}`,
			invoke: func(s *Service, c *gin.Context) { s.SipeedLogout(c) },
		},
		{
			name:   "revoke disclaimer",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/disclaimer/revoke",
			body:   `{}`,
			invoke: func(s *Service, c *gin.Context) { s.SipeedRevokeDisclaimer(c) },
		},
		{
			name:   "claim code",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/claim-code",
			body:   `{}`,
			invoke: func(s *Service, c *gin.Context) { s.SipeedClaimCode(c) },
		},
		{
			name:   "pending",
			method: http.MethodGet,
			target: "/api/extensions/tailscale/sipeed/pending",
			invoke: func(s *Service, c *gin.Context) { s.SipeedPending(c) },
		},
		{
			name:   "approve",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/pending/pending-1/approve",
			body:   `{"challenge":"","name":"laptop"}`,
			params: gin.Params{{Key: "id", Value: "pending-1"}},
			invoke: func(s *Service, c *gin.Context) { s.SipeedApprove(c) },
		},
		{
			name:   "reject",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/pending/pending-1/reject",
			body:   `{}`,
			params: gin.Params{{Key: "id", Value: "pending-1"}},
			invoke: func(s *Service, c *gin.Context) { s.SipeedReject(c) },
		},
		{
			name:   "authorize",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/authorize",
			body:   `{"challenge":"","code":"23456789","name":"phone"}`,
			invoke: func(s *Service, c *gin.Context) { s.SipeedAuthorize(c) },
		},
		{
			name:   "devices",
			method: http.MethodGet,
			target: "/api/extensions/tailscale/sipeed/devices",
			invoke: func(s *Service, c *gin.Context) { s.SipeedDevices(c) },
		},
		{
			name:   "rename device",
			method: http.MethodPost,
			target: "/api/extensions/tailscale/sipeed/devices/3/name",
			body:   `{"challenge":"","name":"renamed"}`,
			params: gin.Params{{Key: "id", Value: "3"}},
			invoke: func(s *Service, c *gin.Context) { s.SipeedRenameDevice(c) },
		},
		{
			name:   "remove device",
			method: http.MethodDelete,
			target: "/api/extensions/tailscale/sipeed/devices/3",
			body:   `{}`,
			params: gin.Params{{Key: "id", Value: "3"}},
			invoke: func(s *Service, c *gin.Context) { s.SipeedRemoveDevice(c) },
		},
	}

	for _, region := range []proto.TailscaleRegion{proto.TailscaleRegionCN, proto.TailscaleRegionGlobal} {
		for _, tc := range cases {
			t.Run(string(region)+"/"+tc.name, func(t *testing.T) {
				pairing := newManifestGatePairing()
				service, statePath := newManifestGateService(t, pairing)
				service.regionStore = useTestTailscaleRegion(t, region)
				before := readManifestGateStateFile(t, statePath)

				context, recorder := newSipeedManagementContext(tc.method, tc.target, tc.body, tc.params)
				tc.invoke(service, context)

				response := decodeSipeedManagementResponse(t, recorder)
				if response.Code != -1 || response.Msg != "managed tailscale mode is unavailable" {
					t.Fatalf("response = %+v, want code -1 / managed tailscale mode is unavailable", response)
				}
				if calls := pairing.callCount(); calls != 0 {
					t.Fatalf("upstream pairing calls = %d, want 0", calls)
				}
				if after := readManifestGateStateFile(t, statePath); after != before {
					t.Fatalf("state file changed on a refused request")
				}
				if state := service.sipeed.snapshot(); state.ManagedState != proto.TailscaleManagedEnabledConnected {
					t.Fatalf("managed state changed on a refused request: %+v", state)
				}
			})
		}
	}
}
