package tailscale

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestCurrentSipeedProfileDoesNotOverrideReadableOfficialProfile(t *testing.T) {
	useTestServerFeatureFlags(t, true)
	service := &Service{}
	official := &TsProfile{
		NodeID:     "official-node",
		ControlURL: "https://controlplane.tailscale.com",
	}

	if service.isCurrentSipeedProfile(proto.TailscaleRegionCN, official, nil, true) {
		t.Fatal("readable official profile was classified as Sipeed")
	}
	if !service.isCurrentSipeedProfile(
		proto.TailscaleRegionCN,
		nil,
		errors.New("profile unavailable"),
		true,
	) {
		t.Fatal("managed state was not used when the current profile was unavailable")
	}
	if !service.isCurrentSipeedProfile(
		proto.TailscaleRegionGlobal,
		&TsProfile{ControlURL: sipeedControlURL},
		nil,
		true,
	) {
		t.Fatal("Sipeed profile was hidden in global region")
	}
}

func TestManagedTransitionKeepsSelectedModeWhileOfficialProfileIsStillActive(t *testing.T) {
	for _, state := range []proto.TailscaleManagedState{
		proto.TailscaleManagedEnrolling, proto.TailscaleManagedEnabledConnecting,
	} {
		data := &proto.GetTailscaleStatusRsp{State: proto.TailscaleRunning, ManagedState: state}
		official := &TsProfile{NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com"}
		if normalizeNonSipeedProfileStatus(data, true, official, nil) ||
			data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeSipeed {
			t.Fatalf("managed transition was replaced by the old official profile: %+v", data)
		}
	}
}

func TestReadableOfficialProfilePreservesManagedFailure(t *testing.T) {
	data := &proto.GetTailscaleStatusRsp{
		State:             proto.TailscaleRunning,
		ManagedState:      proto.TailscaleManagedDegraded,
		ManagedError:      "rateLimited",
		RetryAfterSeconds: 37,
	}
	official := &TsProfile{NodeID: "official-node", ControlURL: "https://controlplane.tailscale.com"}
	if !normalizeNonSipeedProfileStatus(data, true, official, nil) {
		t.Fatal("readable official profile was not recognized")
	}
	if data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeOfficial {
		t.Fatalf("active mode = %v, want official", data.ActiveMode)
	}
	if data.ManagedState != proto.TailscaleManagedDegraded || data.ManagedError != "rateLimited" ||
		data.RetryAfterSeconds != 37 {
		t.Fatalf("official profile swallowed managed failure: %+v", data)
	}
}

func TestReadableEmptyProfileKeepsManagedRetryPageReachable(t *testing.T) {
	data := &proto.GetTailscaleStatusRsp{
		State:        proto.TailscaleNotLogin,
		ManagedState: proto.TailscaleManagedDegraded,
		ManagedError: "pairingUnavailable",
	}
	if normalizeNonSipeedProfileStatus(data, true, &TsProfile{}, nil) {
		t.Fatal("empty profile was classified as an active official identity")
	}
	if data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeSipeed ||
		data.ManagedError != "pairingUnavailable" {
		t.Fatalf("managed retry state = %+v", data)
	}
}

func TestReadableEmptyProfileDegradesStaleConnectedManagedState(t *testing.T) {
	data := &proto.GetTailscaleStatusRsp{
		State:        proto.TailscaleNotLogin,
		ManagedState: proto.TailscaleManagedEnabledConnected,
	}
	if normalizeNonSipeedProfileStatus(data, true, &TsProfile{}, nil) {
		t.Fatal("empty profile was classified as an active official identity")
	}
	if data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeSipeed ||
		data.ManagedState != proto.TailscaleManagedDegraded || data.ManagedError != "connectionLost" {
		t.Fatalf("stale managed connection status = %+v", data)
	}
}

func TestOfficialProfileIsNotActiveWhileTailscaleNeedsLogin(t *testing.T) {
	profile := &TsProfile{NodeID: "official-node"}
	if isOfficialTailscaleProfileActive(profile, nil, proto.TailscaleNotLogin) {
		t.Fatal("official profile was treated as active while Tailscale needs login")
	}
	if !isOfficialTailscaleProfileActive(profile, nil, proto.TailscaleRunning) {
		t.Fatal("running official profile was not treated as active")
	}
}

func TestEffectiveTailscaleStatePrefersLoggedOutPrefs(t *testing.T) {
	state, ok := effectiveTailscaleState(
		&TsStatus{BackendState: "Stopped", HaveNodeKey: true},
		&TsPrefs{LoggedOut: true},
	)
	if !ok || state != proto.TailscaleNotLogin {
		t.Fatalf("effective state = %q, %v, want %q, true", state, ok, proto.TailscaleNotLogin)
	}

	state, ok = effectiveTailscaleState(
		&TsStatus{BackendState: "Stopped", HaveNodeKey: true},
		&TsPrefs{LoggedOut: false},
	)
	if !ok || state != proto.TailscaleStopped {
		t.Fatalf("effective state = %q, %v, want %q, true", state, ok, proto.TailscaleStopped)
	}
}

func isolatedMutationCoordinator(t *testing.T) *mutationCoordinator {
	t.Helper()

	previous := tailscaleMutations
	coordinator := &mutationCoordinator{}
	tailscaleMutations = coordinator
	t.Cleanup(func() {
		tailscaleMutations = previous
	})
	return coordinator
}

func TestGetStatusPrefersInstallingState(t *testing.T) {
	coordinator := isolatedMutationCoordinator(t)
	release, ok := coordinator.tryBegin(true)
	if !ok {
		t.Fatal("failed to begin install mutation")
	}
	defer release()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/extensions/tailscale/status", nil)

	NewService().GetStatus(context)

	var response struct {
		Code int                         `json:"code"`
		Data proto.GetTailscaleStatusRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("status response code = %d, want 0", response.Code)
	}
	if response.Data.State != proto.TailscaleInstalling {
		t.Fatalf("status state = %q, want %q", response.Data.State, proto.TailscaleInstalling)
	}
}

func TestBeginTailscaleMutationRejectsConcurrentHandler(t *testing.T) {
	coordinator := isolatedMutationCoordinator(t)
	release, ok := coordinator.tryBegin(false)
	if !ok {
		t.Fatal("failed to begin first mutation")
	}
	defer release()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/extensions/tailscale/start", nil)
	var response proto.Response

	secondRelease, ok := beginTailscaleMutation(context, &response, false)
	if ok || secondRelease != nil {
		t.Fatal("concurrent mutation was accepted")
	}

	var body proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode busy response: %v", err)
	}
	if body.Code != -1 || !strings.Contains(body.Msg, "already in progress") {
		t.Fatalf("busy response = %+v", body)
	}
}

func TestMutationCoordinatorReleasesGuard(t *testing.T) {
	coordinator := &mutationCoordinator{}
	firstRelease, ok := coordinator.tryBegin(true)
	if !ok {
		t.Fatal("failed to begin first mutation")
	}
	if !coordinator.installing() {
		t.Fatal("install mutation did not expose installing state")
	}
	if release, acquired := coordinator.tryBegin(false); acquired || release != nil {
		t.Fatal("second mutation acquired an active guard")
	}

	firstRelease()
	if coordinator.installing() {
		t.Fatal("installing state remained set after release")
	}

	secondRelease, ok := coordinator.tryBegin(false)
	if !ok {
		t.Fatal("guard was not available after release")
	}
	// A stale duplicate release must not clear a newer operation.
	firstRelease()
	if release, acquired := coordinator.tryBegin(false); acquired || release != nil {
		t.Fatal("duplicate release cleared a newer operation")
	}
	secondRelease()

	thirdRelease, ok := coordinator.tryBegin(false)
	if !ok {
		t.Fatal("guard was not released by the second operation")
	}
	thirdRelease()
}
