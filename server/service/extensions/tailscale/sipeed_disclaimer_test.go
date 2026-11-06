package tailscale

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type fakeSipeedDisclaimerStore struct {
	accepted bool
	saveErr  error
	writes   []bool
}

func (store *fakeSipeedDisclaimerStore) Accepted() bool { return store.accepted }

func (store *fakeSipeedDisclaimerStore) SetAccepted(accepted bool) error {
	store.writes = append(store.writes, accepted)
	if store.saveErr != nil {
		return store.saveErr
	}
	store.accepted = accepted
	return nil
}

func TestSipeedEnableRequiresSavedOrExplicitConsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRequires := tailscaleActionRequiresChallenge
	tailscaleActionRequiresChallenge = func(string) bool { return false }
	t.Cleanup(func() { tailscaleActionRequiresChallenge = previousRequires })
	for _, tc := range []struct {
		name         string
		body         string
		accepted     bool
		saveErr      error
		wantError    string
		wantAccepted bool
		wantWrites   []bool
	}{
		{name: "missing consent", body: `{}`, wantError: errSipeedDisclaimerRequired.Error()},
		{name: "false is not consent", body: `{"acceptDisclaimer":false}`, wantError: errSipeedDisclaimerRequired.Error()},
		{name: "save failure blocks enable", body: `{"acceptDisclaimer":true}`, saveErr: errors.New("disk full"), wantError: errSipeedDisclaimerSaveFailed.Error(), wantWrites: []bool{true}},
		{name: "explicit consent is saved before preparation", body: `{"acceptDisclaimer":true}`, wantAccepted: true, wantWrites: []bool{true}},
		{name: "saved consent is reused", body: `{}`, accepted: true, wantAccepted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			useTestServerFeatureFlags(t, true)
			store := &fakeSipeedDisclaimerStore{accepted: tc.accepted, saveErr: tc.saveErr}
			// Fail preparation deliberately: no cloud or LocalAPI work is allowed
			// in this test, even when consent permits reaching preparation.
			manager := &sipeedManager{state: newDisabledSipeedState(), storeErr: errors.New("state unavailable")}
			service := &Service{sipeed: manager, disclaimerStore: store, regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN)}
			ctx, recorder := sipeedEnableTestContext(tc.body)
			service.EnableSipeed(ctx)
			response := decodeSipeedEnableResponse(t, recorder)
			if response.Code != -1 || (tc.wantError != "" && response.Msg != tc.wantError) {
				t.Fatalf("unexpected response: %+v", response)
			}
			if store.accepted != tc.wantAccepted || !reflect.DeepEqual(store.writes, tc.wantWrites) {
				t.Fatalf("consent = %+v", store)
			}
			if service.managedStatusSnapshot().SipeedDisclaimerAccepted != tc.wantAccepted {
				t.Fatal("status does not reflect saved consent")
			}
			release, ok := tailscaleMutations.tryBegin(false)
			if !ok {
				t.Fatal("failed enable retained mutation lock")
			}
			release()
		})
	}
}

func TestSipeedDisclaimerWithdrawalAndLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRequires := tailscaleActionRequiresChallenge
	tailscaleActionRequiresChallenge = func(string) bool { return false }
	t.Cleanup(func() { tailscaleActionRequiresChallenge = previousRequires })
	for _, tc := range []struct {
		name     string
		revoke   bool
		saveErr  error
		cloudErr error
		localErr error
		stopErr  error
	}{
		{name: "normal logout preserves consent"},
		{name: "withdrawal clears consent and network", revoke: true},
		{name: "cloud failure still clears consent and identity", revoke: true, cloudErr: errors.New("offline")},
		{name: "save failure leaves connection available for retry", revoke: true, saveErr: errors.New("disk full")},
		{name: "local failure keeps consent revoked and identity retryable", revoke: true, localErr: errors.New("local logout failed")},
		{name: "stop failure is reported after clearing consent and identity", revoke: true, stopErr: errors.New("stop failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			events := []string{}
			service, manager, _, local, pairing, action := newManagedLifecycleService(t, &events)
			store := &fakeSipeedDisclaimerStore{accepted: true, saveErr: tc.saveErr}
			service.disclaimerStore = store
			pairing.deleteNetworkErr = tc.cloudErr
			local.logoutErr = tc.localErr
			action.stopErr = tc.stopErr
			ctx, recorder := sipeedEnableTestContext(`{}`)
			if tc.revoke {
				service.SipeedRevokeDisclaimer(ctx)
			} else {
				service.SipeedLogout(ctx)
			}
			response := decodeSipeedEnableResponse(t, recorder)
			cleanupFailed := tc.saveErr != nil || tc.localErr != nil
			failed := cleanupFailed || tc.stopErr != nil
			if (response.Code != 0) != failed {
				t.Fatalf("response = %+v", response)
			}
			if store.accepted != (!tc.revoke || tc.saveErr != nil) {
				t.Fatalf("consent = %v", store.accepted)
			}
			if manager.hasManagedRecord() != cleanupFailed {
				t.Fatalf("managed identity retained = %v", manager.hasManagedRecord())
			}
			deleted := false
			for _, event := range events {
				if event == "pairing.deleteNetwork" {
					deleted = true
				}
			}
			if deleted == cleanupFailed {
				t.Fatalf("network cleanup events = %v", events)
			}
			if tc.saveErr != nil && len(events) != 0 {
				t.Fatalf("save failure changed connection: %v", events)
			}
			wantStopCalls := 0
			if tc.revoke && !cleanupFailed {
				wantStopCalls = 1
				if events[len(events)-1] != "action.stop" {
					t.Fatalf("withdrawal must stop the daemon after identity cleanup: %v", events)
				}
			}
			if action.stopCalls != wantStopCalls {
				t.Fatalf("stop calls = %d, want %d", action.stopCalls, wantStopCalls)
			}
			if tc.stopErr != nil {
				action.stopErr = nil
				ctx, recorder = sipeedEnableTestContext(`{}`)
				service.SipeedRevokeDisclaimer(ctx)
				response = decodeSipeedEnableResponse(t, recorder)
				if response.Code != 0 || action.stopCalls != 2 {
					t.Fatalf("withdrawal retry response=%+v stop calls=%d", response, action.stopCalls)
				}
			}
		})
	}
}

func TestSipeedConsentCannotChangeBeforeChallengeOrDuringMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRequires, previousVerify := tailscaleActionRequiresChallenge, verifyTailscaleChallenge
	t.Cleanup(func() {
		tailscaleActionRequiresChallenge = previousRequires
		verifyTailscaleChallenge = previousVerify
	})
	for _, revoke := range []bool{false, true} {
		for _, blockedByChallenge := range []bool{false, true} {
			t.Run(fmt.Sprintf("revoke=%v/challenge=%v", revoke, blockedByChallenge), func(t *testing.T) {
				isolatedMutationCoordinator(t)
				events := []string{}
				service, _, _, _, _, _ := newManagedLifecycleService(t, &events)
				store := &fakeSipeedDisclaimerStore{accepted: revoke}
				service.disclaimerStore = store
				tailscaleActionRequiresChallenge = func(string) bool { return blockedByChallenge }
				verifyTailscaleChallenge = func(_ *gin.Context, _, action string) error {
					want := "tailscale_login"
					if revoke {
						want = "tailscale_logout"
					}
					if action != want {
						t.Fatalf("challenge action = %q", action)
					}
					return errors.New("verification required")
				}
				wantCode := -3
				if !blockedByChallenge {
					release, ok := tailscaleMutations.tryBegin(false)
					if !ok {
						t.Fatal("cannot hold mutation lock")
					}
					defer release()
					wantCode = -1
				}
				ctx, recorder := sipeedEnableTestContext(`{"acceptDisclaimer":true}`)
				if revoke {
					ctx, recorder = sipeedEnableTestContext(`{}`)
					service.SipeedRevokeDisclaimer(ctx)
				} else {
					service.EnableSipeed(ctx)
				}
				response := decodeSipeedEnableResponse(t, recorder)
				if response.Code != wantCode {
					t.Fatalf("response = %+v", response)
				}
				if len(store.writes) != 0 || store.accepted != revoke || len(events) != 0 {
					t.Fatalf("blocked request changed state: store=%+v events=%v", store, events)
				}
			})
		}
	}
}

func TestScreenEnableCannotGrantSipeedConsent(t *testing.T) {
	isolatedMutationCoordinator(t)
	service, manager, _, _, _, _ := newManagedLifecycleService(t, nil)
	store := &fakeSipeedDisclaimerStore{}
	service.disclaimerStore = store
	before := manager.snapshot()
	if err := service.enableManaged(false); !errors.Is(err, errSipeedDisclaimerRequired) {
		t.Fatalf("enable without web consent = %v", err)
	}
	if len(store.writes) != 0 || manager.snapshot() != before {
		t.Fatal("screen enable changed consent or identity")
	}
}

// TestSipeedDisclaimerExportedAccessors pins the thin exported read/write
// surface the root-local gate handler uses: it delegates to the same store as
// the web path, and a nil service reports the save-failure sentinel instead of
// panicking on the write path.
func TestSipeedDisclaimerExportedAccessors(t *testing.T) {
	store := &fakeSipeedDisclaimerStore{}
	service := &Service{disclaimerStore: store}

	if service.SipeedDisclaimerAccepted() {
		t.Fatal("fresh store reports accepted consent")
	}
	if err := service.GrantSipeedDisclaimer(); err != nil {
		t.Fatalf("GrantSipeedDisclaimer: %v", err)
	}
	if !service.SipeedDisclaimerAccepted() {
		t.Fatal("GrantSipeedDisclaimer did not persist acceptance")
	}
	if got := store.writes; !reflect.DeepEqual(got, []bool{true}) {
		t.Fatalf("store writes = %v, want [true]", got)
	}

	// A second grant is an idempotent re-persist, matching the gate's
	// re-sent-flag semantics.
	if err := service.GrantSipeedDisclaimer(); err != nil {
		t.Fatalf("second GrantSipeedDisclaimer: %v", err)
	}
	if got := store.writes; !reflect.DeepEqual(got, []bool{true, true}) {
		t.Fatalf("store writes = %v, want [true true]", got)
	}

	var nilService *Service
	if nilService.SipeedDisclaimerAccepted() {
		t.Fatal("nil service reports accepted consent")
	}
	if err := nilService.GrantSipeedDisclaimer(); !errors.Is(err, errSipeedDisclaimerSaveFailed) {
		t.Fatalf("nil GrantSipeedDisclaimer = %v, want the save-failure sentinel", err)
	}

	store.saveErr = errors.New("disk full")
	if err := service.GrantSipeedDisclaimer(); !errors.Is(err, errSipeedDisclaimerSaveFailed) {
		t.Fatalf("failed GrantSipeedDisclaimer = %v, want the save-failure sentinel", err)
	}
}
