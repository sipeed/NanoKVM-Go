package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/service/tailscalegate"

	"github.com/gin-gonic/gin"
)

// newClaimCodeTestManager builds a manager seeded with a connected managed
// identity over the given pairing fake, for claim-code cache tests. The
// manager clock is real time; tests set the fake's claim expiry relative to
// time.Now.
func newClaimCodeTestManager(t *testing.T, pairing *fakeGatePairing) *sipeedManager {
	t.Helper()
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json"))
	if err := store.save(gateConnectedState()); err != nil {
		t.Fatalf("seed managed state: %v", err)
	}
	return newSipeedManager(sipeedManagerDeps{
		store:   store,
		pairing: pairing,
		now:     time.Now,
	})
}

func validClaimPairing() *fakeGatePairing {
	return &fakeGatePairing{
		claimCode:   "123 456",
		claimExpiry: time.Now().Add(5 * time.Minute),
	}
}

func TestSipeedManagerClaimCodeCachesUntilTheRenewSkew(t *testing.T) {
	pairing := validClaimPairing()
	manager := newClaimCodeTestManager(t, pairing)

	first, err := manager.claimCode(context.Background())
	if err != nil {
		t.Fatalf("claimCode: %v", err)
	}
	second, err := manager.claimCode(context.Background())
	if err != nil {
		t.Fatalf("claimCode: %v", err)
	}
	if first.Code != "123 456" || second.Code != first.Code || second.ExpiresAt != first.ExpiresAt {
		t.Fatalf("cached serves diverged: %+v vs %+v", first, second)
	}
	if claim, _, _, _ := pairing.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (second serve was cached)", claim)
	}
}

func TestSipeedManagerClaimCodeRenewsInsideTheSkew(t *testing.T) {
	pairing := validClaimPairing()
	pairing.claimExpiry = time.Now().Add(10 * time.Second)
	manager := newClaimCodeTestManager(t, pairing)

	for attempt := 0; attempt < 2; attempt++ {
		if _, err := manager.claimCode(context.Background()); err != nil {
			t.Fatalf("claimCode attempt %d: %v", attempt+1, err)
		}
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2 (renewed inside skew)", claim)
	}
}

func TestSipeedManagerClaimCodeTokenChangeInvalidatesTheCache(t *testing.T) {
	pairing := validClaimPairing()
	manager := newClaimCodeTestManager(t, pairing)

	if _, err := manager.claimCode(context.Background()); err != nil {
		t.Fatalf("claimCode: %v", err)
	}

	// Rotate the persisted device token: the cached code belongs to the old
	// identity and must never be served for the new one.
	otherToken, err := newSipeedCredential(sipeedDeviceTokenBytes)
	if err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.state.DeviceToken = otherToken
	if err := manager.saveLocked(); err != nil {
		manager.mu.Unlock()
		t.Fatalf("rotate device token: %v", err)
	}
	manager.mu.Unlock()

	if _, err := manager.claimCode(context.Background()); err != nil {
		t.Fatalf("claimCode after token rotation: %v", err)
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2 (new identity renews)", claim)
	}

	// Dropping the identity entirely fails closed instead of serving a stale
	// code.
	if err := manager.clearIdentity(); err != nil {
		t.Fatalf("clearIdentity: %v", err)
	}
	if _, err := manager.claimCode(context.Background()); !errors.Is(err, errSipeedManagedIdentityUnavailable) {
		t.Fatalf("claimCode without identity = %v, want errSipeedManagedIdentityUnavailable", err)
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2 (no upstream call without identity)", claim)
	}
}

func TestSipeedManagerClaimCodePrepareEnableDropsTheCache(t *testing.T) {
	pairing := validClaimPairing()
	manager := newClaimCodeTestManager(t, pairing)

	if _, err := manager.claimCode(context.Background()); err != nil {
		t.Fatalf("claimCode: %v", err)
	}
	if err := manager.prepareEnable(); err != nil {
		t.Fatalf("prepareEnable: %v", err)
	}
	if _, err := manager.claimCode(context.Background()); err != nil {
		t.Fatalf("claimCode after prepareEnable: %v", err)
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2 (new enrollment cycle renews)", claim)
	}
}

func TestSipeedManagerClaimCodeSingleFlight(t *testing.T) {
	pairing := validClaimPairing()
	// Hold the signals locally: the fake reads its fields under its own mutex,
	// so the test must not read them back while a claim call may be running.
	claimStarted := make(chan struct{})
	claimRelease := make(chan struct{})
	pairing.claimStarted = claimStarted
	pairing.claimRelease = claimRelease
	manager := newClaimCodeTestManager(t, pairing)

	const callers = 2
	codes := make([]string, callers)
	errs := make([]error, callers)
	var waitGroup sync.WaitGroup
	for index := 0; index < callers; index++ {
		waitGroup.Add(1)
		go func(slot int) {
			defer waitGroup.Done()
			response, err := manager.claimCode(context.Background())
			errs[slot] = err
			if err == nil {
				codes[slot] = response.Code
			}
		}(index)
	}

	waitForSignal(t, claimStarted, "the single upstream claim call")
	// Give any wrongly-parallel second call a chance to reach the upstream.
	time.Sleep(50 * time.Millisecond)
	close(claimRelease)
	waitGroup.Wait()

	for slot := 0; slot < callers; slot++ {
		if errs[slot] != nil {
			t.Fatalf("caller %d: %v", slot, errs[slot])
		}
		if codes[slot] != "123 456" {
			t.Fatalf("caller %d code = %q, want the shared code", slot, codes[slot])
		}
	}
	if claim, _, _, _ := pairing.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (single-flight)", claim)
	}
}

func TestSipeedManagerClaimCodePersistsRetryAfter(t *testing.T) {
	pairing := validClaimPairing()
	pairing.claimErr = &sipeedPairingAPIError{
		HTTPStatus: http.StatusTooManyRequests,
		Code:       "rate_limited",
		RetryAfter: 5 * time.Minute,
	}
	manager := newClaimCodeTestManager(t, pairing)

	if _, err := manager.claimCode(context.Background()); err == nil {
		t.Fatal("claimCode must propagate the upstream 429")
	}
	if remaining := manager.pairingRetryRemaining(); remaining <= 0 {
		t.Fatal("the upstream 429 must persist a Retry-After deadline")
	}
	if manager.claim != nil {
		t.Fatal("a failed renewal must not populate the cache")
	}
}

func TestSipeedManagerClaimCodeProtocolViolationIsNotCached(t *testing.T) {
	pairing := &fakeGatePairing{
		claimCode:   "12",
		claimExpiry: time.Now().Add(5 * time.Minute),
	}
	manager := newClaimCodeTestManager(t, pairing)

	if _, err := manager.claimCode(context.Background()); !errors.Is(err, errSipeedClaimCodeProtocol) {
		t.Fatalf("claimCode error = %v, want errSipeedClaimCodeProtocol", err)
	}
	if manager.claim != nil {
		t.Fatal("a contract violation must not populate the cache")
	}

	// A later healthy response is served normally: the violation poisoned
	// nothing.
	pairing.mu.Lock()
	pairing.claimCode = "123 456"
	pairing.mu.Unlock()
	if _, err := manager.claimCode(context.Background()); err != nil {
		t.Fatalf("claimCode after recovery: %v", err)
	}
	if claim, _, _, _ := pairing.counts(); claim != 2 {
		t.Fatalf("claim calls = %d, want 2", claim)
	}
}

// The whole point of the shared cache: one upstream issuance serves both
// transports, and the LCD and the web display the same digits.
func TestSipeedClaimCodeSharedAcrossGateAndWeb(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disableTailscaleChallengesForTest(t)
	pairing := validClaimPairing()
	service := newGateConnectingService(t, pairing, nil, nil)
	driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

	// Web lane first.
	ginContext, recorder := newSipeedManagementContext(
		http.MethodPost,
		"/api/extensions/tailscale/sipeed/claim-code",
		`{}`,
		nil,
	)
	service.SipeedClaimCode(ginContext)
	if recorder.Code != http.StatusOK {
		t.Fatalf("web claim-code HTTP status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var webResponse struct {
		Code int `json:"code"`
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &webResponse); err != nil {
		t.Fatalf("decode web response: %v", err)
	}
	if webResponse.Code != 0 || webResponse.Data.Code != "123 456" {
		t.Fatalf("web claim code = %+v", webResponse)
	}

	// LCD lane second: same upstream code, no second upstream call.
	lcdCode, err := driver.LoginCode(context.Background())
	if err != nil {
		t.Fatalf("gate LoginCode: %v", err)
	}
	if lcdCode != "123456" {
		t.Fatalf("gate code = %q, want the same six digits as the web lane", lcdCode)
	}
	if claim, _, _, _ := pairing.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (one upstream issuance for both lanes)", claim)
	}

	// And in the opposite order: the gate-issued code serves the web.
	pairing2 := validClaimPairing()
	service2 := newGateConnectingService(t, pairing2, nil, nil)
	driver2 := newGateTestDriver(t, service2, pairing2, newGateClock(), nil, nil)
	if _, err := driver2.LoginCode(context.Background()); err != nil {
		t.Fatalf("gate LoginCode: %v", err)
	}
	ginContext2, recorder2 := newSipeedManagementContext(
		http.MethodPost,
		"/api/extensions/tailscale/sipeed/claim-code",
		`{}`,
		nil,
	)
	service2.SipeedClaimCode(ginContext2)
	var webResponse2 struct {
		Code int `json:"code"`
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder2.Body.Bytes(), &webResponse2); err != nil {
		t.Fatalf("decode web response: %v", err)
	}
	if webResponse2.Code != 0 || webResponse2.Data.Code != "123 456" {
		t.Fatalf("web claim code after gate fetch = %+v", webResponse2)
	}
	if claim, _, _, _ := pairing2.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (gate-issued code serves the web)", claim)
	}
}

// The identity clear inside a full exit (the LCD Exit path) must drop the
// shared code: the network is gone, so no lane may keep displaying it.
func TestSipeedClaimCodeDroppedByIdentityClear(t *testing.T) {
	pairing := validClaimPairing()
	service := newGateConnectingService(t, pairing, nil, nil)
	driver := newGateTestDriver(t, service, pairing, newGateClock(), nil, nil)

	if _, err := driver.LoginCode(context.Background()); err != nil {
		t.Fatalf("LoginCode: %v", err)
	}
	if err := service.sipeed.clearIdentity(); err != nil {
		t.Fatalf("clearIdentity: %v", err)
	}
	if _, err := driver.LoginCode(context.Background()); !errors.Is(err, tailscalegate.ErrLoginCodeNotReady) {
		t.Fatalf("LoginCode after identity clear = %v, want not-ready", err)
	}
	if claim, _, _, _ := pairing.counts(); claim != 1 {
		t.Fatalf("claim calls = %d, want 1 (no renewal after the clear)", claim)
	}
}
