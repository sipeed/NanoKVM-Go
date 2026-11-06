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

func sipeedEnableTestContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/extensions/tailscale/sipeed/enable",
		bytes.NewBufferString(body),
	)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	return ginContext, recorder
}

func decodeSipeedEnableResponse(t *testing.T, recorder *httptest.ResponseRecorder) proto.Response {
	t.Helper()
	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v: %s", err, recorder.Body.String())
	}
	return response
}

func TestSipeedEnableInGlobalUsesLoginChallengeAndRejectsConcurrentRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	useTestServerFeatureFlags(t, true)
	coordinator := isolatedMutationCoordinator(t)
	previousRequires := tailscaleActionRequiresChallenge
	previousVerify := verifyTailscaleChallenge
	tailscaleActionRequiresChallenge = func(action string) bool {
		if action != "tailscale_login" {
			t.Fatalf("challenge action = %q", action)
		}
		return true
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

	local, pairing := successfulSipeedFakes(nil)
	pairing.enrollStarted = make(chan struct{})
	pairing.enrollRelease = make(chan struct{})
	manager := newSipeedManager(sipeedManagerDeps{
		store:          newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json")),
		pairing:        pairing,
		newLocalClient: func(context.Context) sipeedManagedLocalClient { return local },
		isInstalled:    func() bool { return true },
		hostname:       func() (string, error) { return "nano-kvm", nil },
		now:            time.Now,
		wait:           func(context.Context, time.Duration) error { return nil },
	})
	service := &Service{
		sipeed:          manager,
		disclaimerStore: &fakeSipeedDisclaimerStore{accepted: true},
		regionStore:     useTestTailscaleRegion(t, proto.TailscaleRegionGlobal),
	}
	t.Cleanup(func() {
		close(pairing.enrollRelease)
		// ReportSelfNode returns before the final state write. Wait for
		// runEnable's deferred release before TempDir cleanup, even on failure.
		deadline := time.Now().Add(2 * time.Second)
		for coordinator.state.Load() != 0 {
			if time.Now().After(deadline) {
				t.Fatal("background enable did not finish")
			}
			time.Sleep(time.Millisecond)
		}
		state, err := manager.deps.store.load()
		if err != nil {
			t.Fatalf("load completed enable state: %v", err)
		}
		if state.ManagedState != proto.TailscaleManagedEnabledConnected ||
			state.Phase != sipeedPhaseComplete || state.NodeID != "3" {
			t.Fatalf("persisted enable state = %q, phase = %q, node = %q",
				state.ManagedState, state.Phase, state.NodeID)
		}
	})

	firstContext, firstRecorder := sipeedEnableTestContext(`{"challenge":"approved"}`)
	service.EnableSipeed(firstContext)
	if response := decodeSipeedEnableResponse(t, firstRecorder); response.Code != 0 {
		t.Fatalf("first response = %+v", response)
	}
	select {
	case <-pairing.enrollStarted:
	case <-time.After(time.Second):
		t.Fatal("background enable did not reach pairing service")
	}

	secondContext, secondRecorder := sipeedEnableTestContext(`{"challenge":"approved"}`)
	service.EnableSipeed(secondContext)
	if response := decodeSipeedEnableResponse(t, secondRecorder); response.Code != -1 {
		t.Fatalf("concurrent response = %+v", response)
	}
}

func TestSipeedEnableRejectsUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	useTestServerFeatureFlags(t, true)
	service := &Service{
		sipeed:      &sipeedManager{},
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
	}

	context, recorder := sipeedEnableTestContext(`{"challenge":"ok","controlURL":"https://evil.invalid"}`)
	service.EnableSipeed(context)
	response := decodeSipeedEnableResponse(t, recorder)
	if response.Code != -1 || response.Msg != "invalid parameters" {
		t.Fatalf("unexpected response = %+v", response)
	}
}

func TestSipeedEnablePreparationFailureReturnsRateLimitStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	useTestServerFeatureFlags(t, true)
	isolatedMutationCoordinator(t)
	previousRequires := tailscaleActionRequiresChallenge
	tailscaleActionRequiresChallenge = func(string) bool { return false }
	t.Cleanup(func() { tailscaleActionRequiresChallenge = previousRequires })

	now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	state := validTestSipeedState(t)
	state.ManagedState = proto.TailscaleManagedDegraded
	state.ManagedError = "rateLimited"
	state.RetryAfter = now.Add(37 * time.Second).Format(time.RFC3339)
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	manager := newSipeedManager(sipeedManagerDeps{
		store: store,
		now:   func() time.Time { return now },
	})
	service := &Service{
		sipeed:          manager,
		disclaimerStore: &fakeSipeedDisclaimerStore{accepted: true},
		regionStore:     useTestTailscaleRegion(t, proto.TailscaleRegionCN),
	}

	context, recorder := sipeedEnableTestContext(`{}`)
	service.EnableSipeed(context)
	var response struct {
		Code int                         `json:"code"`
		Msg  string                      `json:"msg"`
		Data proto.GetTailscaleStatusRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v: %s", err, recorder.Body.String())
	}
	if response.Code != -1 || response.Msg != "managed tailscale enable is rate limited" {
		t.Fatalf("rate-limit envelope = %+v", response)
	}
	if response.Data.ManagedState != proto.TailscaleManagedDegraded ||
		response.Data.ManagedError != "rateLimited" || response.Data.RetryAfterSeconds != 37 {
		t.Fatalf("rate-limit status = %+v", response.Data)
	}
}
