package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type fakeSipeedManagementPairing struct {
	*fakeSipeedPairingClient
	approveCalls int
	devicesCalls int
	devicesErr   error
	claimCalls   int
	claimErr     error
	pending      []sipeedPendingRequest
}

func newFakeSipeedManagementPairing() *fakeSipeedManagementPairing {
	return &fakeSipeedManagementPairing{fakeSipeedPairingClient: &fakeSipeedPairingClient{}}
}

func (fake *fakeSipeedManagementPairing) ClaimCode(
	context.Context,
	string,
) (*sipeedClaimCodeResponse, error) {
	fake.claimCalls++
	if fake.claimErr != nil {
		return nil, fake.claimErr
	}
	// Rotate the code on every upstream issuance, like the real single-use
	// service: a correct shared cache keeps serving the first code, so a test
	// can tell a cache hit from a fresh upstream call.
	code := "123 456"
	if fake.claimCalls > 1 {
		code = "654 321"
	}
	return &sipeedClaimCodeResponse{
		Code: code, ExpiresAt: "2026-09-20T08:00:00Z", SingleUse: true,
	}, nil
}

func (fake *fakeSipeedManagementPairing) Pending(
	context.Context,
	string,
	int,
) (*sipeedPendingResponse, error) {
	return &sipeedPendingResponse{Pending: fake.pending}, nil
}

func (fake *fakeSipeedManagementPairing) ApprovePending(
	context.Context,
	string,
	string,
	string,
) (*sipeedApproveResponse, error) {
	fake.approveCalls++
	return &sipeedApproveResponse{NodeID: "node-2", Name: "laptop", IPs: []string{"100.64.0.4"}}, nil
}

func (fake *fakeSipeedManagementPairing) RejectPending(
	context.Context,
	string,
	string,
) (*sipeedRejectResponse, error) {
	return &sipeedRejectResponse{PendingID: "pending-1", Rejected: true}, nil
}

func (fake *fakeSipeedManagementPairing) Authorize(
	context.Context,
	string,
	string,
	string,
	string,
) (*sipeedAuthorizeResponse, error) {
	return &sipeedAuthorizeResponse{NodeID: "node-2", Name: "phone", IPs: []string{"100.64.0.4"}}, nil
}

func (fake *fakeSipeedManagementPairing) Devices(
	context.Context,
	string,
) (*sipeedDevicesResponse, error) {
	fake.devicesCalls++
	if fake.devicesErr != nil {
		return nil, fake.devicesErr
	}
	return &sipeedDevicesResponse{NetworkID: "net-test", Devices: []sipeedDevice{}}, nil
}

func (fake *fakeSipeedManagementPairing) RenameDevice(
	context.Context,
	string,
	string,
	string,
) (*sipeedRenameDeviceResponse, error) {
	return &sipeedRenameDeviceResponse{NodeID: "node-2", Name: "renamed"}, nil
}

func newSipeedManagementHandlerService(
	t *testing.T,
	pairing *fakeSipeedManagementPairing,
	now time.Time,
) *Service {
	t.Helper()
	useTestServerFeatureFlags(t, true)
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	if err := store.save(validTestSipeedState(t)); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store:       store,
		pairing:     pairing,
		isInstalled: func() bool { return false },
		now:         func() time.Time { return now },
	})
	return &Service{
		sipeed:      manager,
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
	}
}

func newSipeedManagementContext(
	method string,
	target string,
	body string,
	params gin.Params,
) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(method, target, bytes.NewBufferString(body))
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Params = params
	return ginContext, recorder
}

type sipeedManagementTestResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data proto.GetTailscaleStatusRsp `json:"data"`
}

func decodeSipeedManagementResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) sipeedManagementTestResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response sipeedManagementTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v: %s", err, recorder.Body.String())
	}
	return response
}

func disableTailscaleChallengesForTest(t *testing.T) {
	t.Helper()
	previous := tailscaleActionRequiresChallenge
	tailscaleActionRequiresChallenge = func(string) bool { return false }
	t.Cleanup(func() { tailscaleActionRequiresChallenge = previous })
}

func TestSipeedManagementHandlerEnforcesLoginChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	pairing := newFakeSipeedManagementPairing()
	service := newSipeedManagementHandlerService(t, pairing, now)

	previousRequires := tailscaleActionRequiresChallenge
	previousVerify := verifyTailscaleChallenge
	tailscaleActionRequiresChallenge = func(action string) bool {
		return action == "tailscale_login"
	}
	verifyTailscaleChallenge = func(_ *gin.Context, challenge, action string) error {
		if action != "tailscale_login" || challenge != "approved" {
			return errors.New("challenge rejected")
		}
		return nil
	}
	t.Cleanup(func() {
		tailscaleActionRequiresChallenge = previousRequires
		verifyTailscaleChallenge = previousVerify
	})

	deniedContext, deniedRecorder := newSipeedManagementContext(
		http.MethodPost,
		"/api/extensions/tailscale/sipeed/pending/pending-1/approve",
		`{"challenge":"wrong","name":"laptop"}`,
		gin.Params{{Key: "id", Value: "pending-1"}},
	)
	service.SipeedApprove(deniedContext)
	denied := decodeSipeedManagementResponse(t, deniedRecorder)
	if denied.Code != -3 || pairing.approveCalls != 0 {
		t.Fatalf("denied approval = %+v, calls=%d", denied, pairing.approveCalls)
	}

	approvedContext, approvedRecorder := newSipeedManagementContext(
		http.MethodPost,
		"/api/extensions/tailscale/sipeed/pending/pending-1/approve",
		`{"challenge":"approved","name":"laptop"}`,
		gin.Params{{Key: "id", Value: "pending-1"}},
	)
	service.SipeedApprove(approvedContext)
	approved := decodeSipeedManagementResponse(t, approvedRecorder)
	if approved.Code != 0 || pairing.approveCalls != 1 {
		t.Fatalf("approved response = %+v, calls=%d", approved, pairing.approveCalls)
	}
}

func TestSipeedManagementHandlerMapsUpstreamUnauthorizedIntoBusinessEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	pairing := newFakeSipeedManagementPairing()
	pairing.devicesErr = &sipeedPairingAPIError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       "invalid_token",
	}
	service := newSipeedManagementHandlerService(t, pairing, now)

	ginContext, recorder := newSipeedManagementContext(
		http.MethodGet,
		"/api/extensions/tailscale/sipeed/devices",
		"",
		nil,
	)
	service.SipeedDevices(ginContext)
	response := decodeSipeedManagementResponse(t, recorder)
	if response.Code != -2 || response.Data.ManagedState != proto.TailscaleManagedDisabled {
		t.Fatalf("unauthorized response = %+v", response)
	}
	state := service.sipeed.snapshot()
	if state.ManagedState != proto.TailscaleManagedDisabled || state.DeviceToken != "" {
		t.Fatalf("unauthorized request kept identity: %+v", state)
	}
}

func TestSipeedManagementHandlerPersistsAndReusesRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	pairing := newFakeSipeedManagementPairing()
	pairing.devicesErr = &sipeedPairingAPIError{
		HTTPStatus: http.StatusTooManyRequests,
		Code:       "rate_limited",
		RetryAfter: 37 * time.Second,
	}
	service := newSipeedManagementHandlerService(t, pairing, now)

	for attempt := 0; attempt < 2; attempt++ {
		ginContext, recorder := newSipeedManagementContext(
			http.MethodGet,
			"/api/extensions/tailscale/sipeed/devices",
			"",
			nil,
		)
		service.SipeedDevices(ginContext)
		response := decodeSipeedManagementResponse(t, recorder)
		if response.Code != -2 || response.Data.RetryAfterSeconds != 37 {
			t.Fatalf("attempt %d rate-limit response = %+v", attempt+1, response)
		}
	}
	if pairing.devicesCalls != 1 {
		t.Fatalf("upstream device calls = %d, want 1", pairing.devicesCalls)
	}
	if got := service.sipeed.snapshot().RetryAfter; got != now.Add(37*time.Second).Format(time.RFC3339) {
		t.Fatalf("persisted retry deadline = %q", got)
	}
}

func TestSipeedManagementHandlerRejectsRemovingCurrentDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	pairing := newFakeSipeedManagementPairing()
	service := newSipeedManagementHandlerService(t, pairing, now)

	ginContext, recorder := newSipeedManagementContext(
		http.MethodDelete,
		"/api/extensions/tailscale/sipeed/devices/3",
		`{}`,
		gin.Params{{Key: "id", Value: "3"}},
	)
	service.SipeedRemoveDevice(ginContext)
	response := decodeSipeedManagementResponse(t, recorder)
	if response.Code != -1 || response.Msg != "the current device can only be removed by logout" {
		t.Fatalf("self-removal response = %+v", response)
	}
	if len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf("self-removal reached upstream: %v", pairing.deleteNodeIDs)
	}
}

func TestSipeedClaimCodeServesTheSharedCachedCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	pairing := newFakeSipeedManagementPairing()
	service := newSipeedManagementHandlerService(t, pairing, now)

	type claimData struct {
		Code      string `json:"code"`
		ExpiresAt string `json:"expiresAt"`
		SingleUse bool   `json:"singleUse"`
	}
	type claimResponse struct {
		Code int       `json:"code"`
		Data claimData `json:"data"`
	}

	request := func(body string) claimResponse {
		ginContext, recorder := newSipeedManagementContext(
			http.MethodPost,
			"/api/extensions/tailscale/sipeed/claim-code",
			body,
			nil,
		)
		service.SipeedClaimCode(ginContext)
		if recorder.Code != http.StatusOK {
			t.Fatalf("HTTP status = %d: %s", recorder.Code, recorder.Body.String())
		}
		var response claimResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v: %s", err, recorder.Body.String())
		}
		return response
	}

	first := request(`{}`)
	second := request(`{}`)
	if first.Code != 0 || second.Code != 0 {
		t.Fatalf("claim-code responses = %d, %d", first.Code, second.Code)
	}
	if first.Data.Code != "123 456" || second.Data.Code != first.Data.Code ||
		second.Data.ExpiresAt != first.Data.ExpiresAt {
		t.Fatalf("repeated fetches diverged: %+v vs %+v", first.Data, second.Data)
	}
	if pairing.claimCalls != 1 {
		t.Fatalf("upstream claim calls = %d, want 1 (shared cache)\n", pairing.claimCalls)
	}

	renewed := request(`{"refresh":true}`)
	if renewed.Code != 0 || renewed.Data.Code != "654 321" || pairing.claimCalls != 2 {
		t.Fatalf("explicit renewal = %+v, upstream calls = %d", renewed, pairing.claimCalls)
	}
	cached := request(`{}`)
	if cached.Code != 0 || cached.Data.Code != renewed.Data.Code || pairing.claimCalls != 2 {
		t.Fatalf("renewed code was not shared: %+v, upstream calls = %d", cached, pairing.claimCalls)
	}
}

func TestSipeedClaimCodeFailedRenewalDropsOldCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	pairing := newFakeSipeedManagementPairing()
	service := newSipeedManagementHandlerService(t, pairing, time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC))
	if _, err := service.sipeed.claimCode(context.Background()); err != nil {
		t.Fatal(err)
	}
	pairing.claimErr = errors.New("upstream renewal failed")
	ctx, recorder := newSipeedManagementContext(http.MethodPost,
		"/api/extensions/tailscale/sipeed/claim-code", `{"refresh":true}`, nil)
	service.SipeedClaimCode(ctx)
	if response := decodeSipeedManagementResponse(t, recorder); response.Code == 0 {
		t.Fatal("failed renewal reported success")
	}
	pairing.claimErr = nil
	code, err := service.sipeed.claimCode(context.Background())
	if err != nil || code.Code != "654 321" || pairing.claimCalls != 3 {
		t.Fatalf("read after failed renewal reused old code: %+v, error=%v, calls=%d", code, err, pairing.claimCalls)
	}
}

func TestSipeedPairingInvalidatesConsumedClaimCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	for _, action := range []string{"pending", "approve", "reject", "authorize"} {
		t.Run(action, func(t *testing.T) {
			pairing := newFakeSipeedManagementPairing()
			pairing.pending = []sipeedPendingRequest{{PendingID: "pending-1"}}
			service := newSipeedManagementHandlerService(t, pairing, time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC))
			if _, err := service.sipeed.claimCode(context.Background()); err != nil {
				t.Fatal(err)
			}
			body := `{}`
			if action == "authorize" {
				body = `{"code":"2345-678A"}`
			}
			ctx, recorder := newSipeedManagementContext(http.MethodPost,
				"/api/extensions/tailscale/sipeed/"+action, body,
				gin.Params{{Key: "id", Value: "pending-1"}})
			switch action {
			case "pending":
				service.SipeedPending(ctx)
			case "approve":
				service.SipeedApprove(ctx)
			case "reject":
				service.SipeedReject(ctx)
			case "authorize":
				service.SipeedAuthorize(ctx)
			}
			if response := decodeSipeedManagementResponse(t, recorder); response.Code != 0 {
				t.Fatalf("pairing action failed: %+v", response)
			}
			code, err := service.sipeed.claimCode(context.Background())
			if err != nil || code.Code != "654 321" || pairing.claimCalls != 2 {
				t.Fatalf("consumed code was cached: %+v, error=%v, calls=%d", code, err, pairing.claimCalls)
			}
		})
	}
}

func TestSipeedLatePairingResponsePreservesRenewedClaimCode(t *testing.T) {
	pairing := newFakeSipeedManagementPairing()
	service := newSipeedManagementHandlerService(t, pairing, time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC))
	if _, err := service.sipeed.claimCode(context.Background()); err != nil {
		t.Fatal(err)
	}
	token, _, err := service.sipeed.pairingCredentials()
	if err != nil {
		t.Fatal(err)
	}
	invalidateClaim := service.sipeed.claimCodeInvalidator(token)
	if _, err := service.sipeed.getClaimCode(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	invalidateClaim()
	code, err := service.sipeed.claimCode(context.Background())
	if err != nil || code.Code != "654 321" || pairing.claimCalls != 2 {
		t.Fatalf("late pairing response invalidated renewed code: %+v, error=%v, calls=%d", code, err, pairing.claimCalls)
	}
}
