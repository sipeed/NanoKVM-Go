package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type fakeTailscaleLoginClient struct {
	status    *TsStatus
	statusErr error

	startCalls  int
	statusCalls int
	loginCalls  int
	startFunc   func()
	loginFunc   func() error
}

func (f *fakeTailscaleLoginClient) Status() (*TsStatus, error) {
	f.statusCalls++
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.status, nil
}

func (f *fakeTailscaleLoginClient) Start() error {
	f.startCalls++
	if f.startFunc != nil {
		f.startFunc()
	}
	return nil
}

func (f *fakeTailscaleLoginClient) Login() (string, error) {
	f.loginCalls++
	if f.loginFunc != nil {
		return "", f.loginFunc()
	}
	return "", nil
}

func installLoginHandlerTestHooks(
	t *testing.T,
	constructor func(context.Context) tailscaleLoginClient,
) {
	t.Helper()

	previousRequires := tailscaleActionRequiresChallenge
	previousConstructor := newTailscaleLoginClient
	tailscaleActionRequiresChallenge = func(string) bool { return false }
	newTailscaleLoginClient = constructor
	t.Cleanup(func() {
		tailscaleActionRequiresChallenge = previousRequires
		newTailscaleLoginClient = previousConstructor
	})
}

func newLoginHandlerContext(t *testing.T, ctx context.Context) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"/api/extensions/tailscale/login",
		nil,
	)
	return ginContext, recorder
}

func decodeLoginHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder) proto.Response {
	t.Helper()
	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return response
}

func TestGetStatusHidesManagedModeWhenManifestDisabled(t *testing.T) {
	useTestServerFeatureFlags(t, false)
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	if err := store.save(validTestSipeedState(t)); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		sipeed: newSipeedManager(sipeedManagerDeps{
			store:       store,
			isInstalled: func() bool { return false },
		}),
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
	}

	data, err := service.getStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := data.AvailableModes, []proto.TailscaleMode{proto.TailscaleModeOfficial}; !reflect.DeepEqual(got, want) {
		t.Fatalf("AvailableModes = %v, want %v", got, want)
	}
	if data.ActiveMode != nil {
		t.Fatalf("ActiveMode = %v, want nil", *data.ActiveMode)
	}
	if data.ManagedState != proto.TailscaleManagedDisabled {
		t.Fatalf("ManagedState = %q, want disabled", data.ManagedState)
	}
	if data.ManagedError != "" || data.RetryAfterSeconds != 0 {
		t.Fatalf("managed details leaked: %+v", data)
	}
}

func TestGetStatusReflectsManagedModeInGlobalWhenManifestEnabled(t *testing.T) {
	useTestServerFeatureFlags(t, true)
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	if err := store.save(validTestSipeedState(t)); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		sipeed: newSipeedManager(sipeedManagerDeps{
			store:       store,
			isInstalled: func() bool { return false },
		}),
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionGlobal),
	}

	data, err := service.getStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := data.AvailableModes, []proto.TailscaleMode{proto.TailscaleModeOfficial, proto.TailscaleModeSipeed}; !reflect.DeepEqual(got, want) {
		t.Fatalf("AvailableModes = %v, want %v", got, want)
	}
	if data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeSipeed {
		t.Fatalf("ActiveMode = %v, want sipeed", data.ActiveMode)
	}
	if data.ManagedState != proto.TailscaleManagedDegraded || data.ManagedError != "clientMissing" {
		t.Fatalf("ManagedState = %q/%q, want degraded/clientMissing", data.ManagedState, data.ManagedError)
	}
}

func TestManagedStatusSnapshotFollowsTheManifestGate(t *testing.T) {
	newService := func(t *testing.T) *Service {
		t.Helper()
		store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
		if err := store.save(validTestSipeedState(t)); err != nil {
			t.Fatal(err)
		}
		return &Service{
			sipeed:      newSipeedManager(sipeedManagerDeps{store: store}),
			regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
		}
	}

	t.Run("manifest off hides a persisted managed record", func(t *testing.T) {
		useTestServerFeatureFlags(t, false)
		service := newService(t)

		data := service.managedStatusSnapshot()
		if got, want := data.AvailableModes, []proto.TailscaleMode{proto.TailscaleModeOfficial}; !reflect.DeepEqual(got, want) {
			t.Fatalf("AvailableModes = %v, want %v", got, want)
		}
		if data.ManagedState != proto.TailscaleManagedDisabled {
			t.Fatalf("ManagedState = %q, want disabled", data.ManagedState)
		}
		if data.ManagedError != "" || data.RetryAfterSeconds != 0 || data.ActiveMode != nil {
			t.Fatalf("managed details leaked: %+v", data)
		}
	})

	t.Run("manifest on reflects a persisted managed record", func(t *testing.T) {
		useTestServerFeatureFlags(t, true)
		service := newService(t)

		data := service.managedStatusSnapshot()
		if data.ManagedState != proto.TailscaleManagedEnabledConnected {
			t.Fatalf("ManagedState = %q, want enabledConnected", data.ManagedState)
		}
		if data.ActiveMode == nil || *data.ActiveMode != proto.TailscaleModeSipeed {
			t.Fatalf("ActiveMode = %v, want sipeed", data.ActiveMode)
		}
	})
}

func TestLoginHandlerPassesRequestContextAndSkipsStartAfterCancellation(t *testing.T) {
	isolatedMutationCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &fakeTailscaleLoginClient{statusErr: context.Canceled}
	var captured context.Context
	installLoginHandlerTestHooks(t, func(got context.Context) tailscaleLoginClient {
		captured = got
		return fake
	})

	ginContext, recorder := newLoginHandlerContext(t, ctx)
	started := time.Now()
	NewService().Login(ginContext)

	response := decodeLoginHandlerResponse(t, recorder)
	if response.Code != -1 || response.Msg != "unknown status" {
		t.Fatalf("canceled login response = %+v, want code -1/unknown status", response)
	}
	if captured == nil {
		t.Fatal("constructor received a nil context")
	}
	if !errors.Is(captured.Err(), context.Canceled) {
		t.Fatalf("constructor context error = %v, want %v", captured.Err(), context.Canceled)
	}
	if fake.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", fake.statusCalls)
	}
	if fake.startCalls != 0 {
		t.Fatalf("start calls = %d, want 0 after cancellation", fake.startCalls)
	}
	if fake.loginCalls != 0 {
		t.Fatalf("login calls = %d, want 0 after cancellation", fake.loginCalls)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("canceled login took %s, want no fixed retry wait", elapsed)
	}
}

func TestLoginHandlerStopsLoginPollingWhenRequestIsCanceled(t *testing.T) {
	isolatedMutationCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loginStarted := make(chan struct{})
	fake := &fakeTailscaleLoginClient{
		status: &TsStatus{BackendState: "NeedsLogin"},
		loginFunc: func() error {
			close(loginStarted)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	var captured context.Context
	installLoginHandlerTestHooks(t, func(got context.Context) tailscaleLoginClient {
		captured = got
		return fake
	})

	ginContext, recorder := newLoginHandlerContext(t, ctx)
	done := make(chan struct{})
	go func() {
		NewService().Login(ginContext)
		close(done)
	}()

	select {
	case <-loginStarted:
	case <-time.After(time.Second):
		t.Fatal("login polling did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("login handler did not stop after request cancellation")
	}

	response := decodeLoginHandlerResponse(t, recorder)
	if response.Code != -2 || response.Msg != "login failed" {
		t.Fatalf("canceled polling response = %+v, want code -2/login failed", response)
	}
	if captured == nil {
		t.Fatal("constructor received a nil context")
	}
	if !errors.Is(captured.Err(), context.Canceled) {
		t.Fatalf("constructor context error = %v, want %v", captured.Err(), context.Canceled)
	}
	if fake.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", fake.statusCalls)
	}
	if fake.startCalls != 0 {
		t.Fatalf("start calls = %d, want 0", fake.startCalls)
	}
	if fake.loginCalls != 1 {
		t.Fatalf("login calls = %d, want 1", fake.loginCalls)
	}
}

func TestLoginHandlerCancellationInterruptsStatusRetryWait(t *testing.T) {
	isolatedMutationCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startCalled := make(chan struct{})
	fake := &fakeTailscaleLoginClient{
		statusErr: errors.New("daemon unavailable"),
		startFunc: func() {
			close(startCalled)
		},
	}
	installLoginHandlerTestHooks(t, func(context.Context) tailscaleLoginClient {
		return fake
	})

	ginContext, recorder := newLoginHandlerContext(t, ctx)
	done := make(chan struct{})
	go func() {
		NewService().Login(ginContext)
		close(done)
	}()

	select {
	case <-startCalled:
	case <-time.After(time.Second):
		t.Fatal("daemon start was not attempted after the initial status failure")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("login handler did not interrupt the status retry wait")
	}

	response := decodeLoginHandlerResponse(t, recorder)
	if response.Code != -1 || response.Msg != "unknown status" {
		t.Fatalf("canceled retry response = %+v, want code -1/unknown status", response)
	}
	if fake.statusCalls != 1 {
		t.Fatalf("status calls = %d, want no second poll after cancellation", fake.statusCalls)
	}
	if fake.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", fake.startCalls)
	}
	if fake.loginCalls != 0 {
		t.Fatalf("login calls = %d, want 0", fake.loginCalls)
	}
}

func TestLoginHandlerDoesNotRestoreOldOfficialProfiles(t *testing.T) {
	for _, managedState := range []proto.TailscaleManagedState{proto.TailscaleManagedDisabled, proto.TailscaleManagedInvalidIdentity} {
		t.Run(string(managedState), func(t *testing.T) {
			isolatedMutationCoordinator(t)
			client := &fakeTailscaleLoginClient{status: testSipeedStatus("NeedsLogin")}
			installLoginHandlerTestHooks(t, func(context.Context) tailscaleLoginClient { return client })
			store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
			state := newDisabledSipeedState()
			state.ManagedState = managedState
			state.PreviousProfileID = "official-profile"
			if err := store.save(state); err != nil {
				t.Fatal(err)
			}
			service := &Service{sipeed: newSipeedManager(sipeedManagerDeps{
				store:       store,
				isInstalled: func() bool { return true },
				newLocalClient: func(context.Context) sipeedManagedLocalClient {
					t.Fatal("old profiles must not be restored")
					return nil
				},
			})}
			ctx, recorder := newLoginHandlerContext(t, context.Background())
			service.Login(ctx)
			if response := decodeLoginHandlerResponse(t, recorder); response.Code != 0 || client.loginCalls != 1 {
				t.Fatalf("fresh official login response=%+v calls=%d", response, client.loginCalls)
			}
			if managedState == proto.TailscaleManagedInvalidIdentity && service.sipeed.snapshot().ManagedState != proto.TailscaleManagedDisabled {
				t.Fatal("invalid managed identity was not cleared")
			}
		})
	}
}
